package main

// cron.go — minimal 5-field cron parser for the start.go cron scanner (M17-C2).
//
// Supports:
//   - Exact values:  5
//   - Wildcard:      *
//   - Step:          */N
//   - Lists:         1,2,5
//   - Ranges:        1-5
//
// Field order (standard cron): minute hour day-of-month month day-of-week
// Range: minute 0-59, hour 0-23, dom 1-31, month 1-12, dow 0-6 (0=Sun).
//
// Engine and core are NOT touched; the scanner goroutine lives entirely inside
// start.go. This file is stdlib-only (IMP §5: no new third-party deps).

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// cronSchedule is a parsed 5-field cron expression.
type cronSchedule struct {
	raw string
	// Each field is a bitset of matching values.
	minutes uint64 // bits 0-59
	hours   uint64 // bits 0-23
	doms    uint32 // bits 1-31
	months  uint16 // bits 1-12
	dows    uint8  // bits 0-6
}

// parseCron parses a standard 5-field cron expression.
// Returns an error if the expression is invalid.
func parseCron(expr string) (cronSchedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return cronSchedule{}, fmt.Errorf("cron: expected 5 fields, got %d in %q", len(fields), expr)
	}

	var s cronSchedule
	s.raw = expr

	var err error

	// minute: 0-59
	s.minutes, err = parseCronField64(fields[0], 0, 59)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("cron: minute field %q: %w", fields[0], err)
	}

	// hour: 0-23
	s.hours, err = parseCronField64(fields[1], 0, 23)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("cron: hour field %q: %w", fields[1], err)
	}

	// day-of-month: 1-31
	domBits, err := parseCronField64(fields[2], 1, 31)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("cron: dom field %q: %w", fields[2], err)
	}
	s.doms = uint32(domBits)

	// month: 1-12
	monthBits, err := parseCronField64(fields[3], 1, 12)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("cron: month field %q: %w", fields[3], err)
	}
	s.months = uint16(monthBits)

	// day-of-week: 0-6 (0=Sunday)
	dowBits, err := parseCronField64(fields[4], 0, 6)
	if err != nil {
		return cronSchedule{}, fmt.Errorf("cron: dow field %q: %w", fields[4], err)
	}
	s.dows = uint8(dowBits)

	return s, nil
}

// parseCronField64 parses one cron field into a bitset uint64.
// Supports: *, */N, N, N-M, N,M,... and combinations of lists with ranges/steps.
func parseCronField64(field string, min, max int) (uint64, error) {
	var bits uint64
	// Split by comma for lists.
	for _, part := range strings.Split(field, ",") {
		partBits, err := parseCronPart64(part, min, max)
		if err != nil {
			return 0, err
		}
		bits |= partBits
	}
	return bits, nil
}

// parseCronPart64 parses a single non-comma part: *, */N, N, N-M, N-M/S.
func parseCronPart64(part string, min, max int) (uint64, error) {
	var bits uint64

	step := 1
	if idx := strings.Index(part, "/"); idx >= 0 {
		stepStr := part[idx+1:]
		s, err := strconv.Atoi(stepStr)
		if err != nil || s <= 0 {
			return 0, fmt.Errorf("invalid step %q", stepStr)
		}
		step = s
		part = part[:idx]
	}

	var lo, hi int
	if part == "*" {
		lo = min
		hi = max
	} else if idx := strings.Index(part, "-"); idx >= 0 {
		loStr, hiStr := part[:idx], part[idx+1:]
		var err error
		lo, err = strconv.Atoi(loStr)
		if err != nil {
			return 0, fmt.Errorf("invalid range start %q", loStr)
		}
		hi, err = strconv.Atoi(hiStr)
		if err != nil {
			return 0, fmt.Errorf("invalid range end %q", hiStr)
		}
	} else {
		v, err := strconv.Atoi(part)
		if err != nil {
			return 0, fmt.Errorf("invalid value %q", part)
		}
		lo = v
		hi = v
	}

	if lo < min || hi > max || lo > hi {
		return 0, fmt.Errorf("value %d-%d out of range [%d-%d]", lo, hi, min, max)
	}

	for v := lo; v <= hi; v += step {
		bits |= 1 << uint(v)
	}
	return bits, nil
}

// Matches reports whether t (truncated to minute boundary) matches the schedule.
// Uses the standard cron convention: fires when minute AND hour AND month AND
// (dom OR dow) match (Vixie cron unrestricted-dow semantics: when both dom and
// dow are restricted, either match fires; V1 uses AND — both unrestricted means
// all values match, which is correct for * * * * *).
func (s cronSchedule) Matches(t time.Time) bool {
	min := t.Minute()
	hour := t.Hour()
	dom := t.Day()
	month := int(t.Month())
	dow := int(t.Weekday()) // 0=Sunday

	minuteOK := (s.minutes>>uint(min))&1 == 1
	hourOK := (s.hours>>uint(hour))&1 == 1
	monthOK := (s.months>>uint(month))&1 == 1
	domOK := (s.doms>>uint(dom))&1 == 1
	dowOK := (s.dows>>uint(dow))&1 == 1

	return minuteOK && hourOK && monthOK && domOK && dowOK
}

// cronEntry pairs a workflow definition ID with its parsed schedule.
type cronEntry struct {
	workflowID string
	schedule   cronSchedule
	namespace  string
}

// buildCronEntries scans registered workflow definitions for type:schedule triggers
// (TriggerTypeSchedule) and parses their config.schedule field as a 5-field cron.
// Invalid schedules are skipped with a stderr warning (best-effort; start must not fail).
// warnFn receives (workflowID, schedExpr, err) for invalid schedules; nil means stderr.
func buildCronEntries(defs []cronWorkflowDef, warnFn func(id, expr string, err error)) []cronEntry {
	var entries []cronEntry
	for _, d := range defs {
		for _, t := range d.triggers {
			if t.triggerType != "schedule" {
				continue
			}
			schedExpr, ok := t.config["schedule"].(string)
			if !ok || schedExpr == "" {
				continue
			}
			sched, err := parseCron(schedExpr)
			if err != nil {
				if warnFn != nil {
					warnFn(d.id, schedExpr, err)
				}
				continue
			}
			entries = append(entries, cronEntry{
				workflowID: d.id,
				schedule:   sched,
				namespace:  d.namespace,
			})
		}
	}
	return entries
}

// cronWorkflowTrigger is a minimal trigger representation for cron scanning.
type cronWorkflowTrigger struct {
	triggerType string
	config      map[string]any
}

// cronWorkflowDef is a minimal workflow definition for cron scanning.
type cronWorkflowDef struct {
	id        string
	namespace string
	triggers  []cronWorkflowTrigger
}
