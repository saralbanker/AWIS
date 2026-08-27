package main

// status.go — 'awis status' command (TDS-07 §4; M14-C2 T5).
//
// No arg → instance table (active + recent completed).
// 'status <instance-id>' → detail incl. current steps + last error.
// --watch → re-poll every 1s (per SPEC §2: "1s re-poll"; TDS-07 says 5s re-print
// for --watch; we implement per SPEC §2 "status --watch = 1s re-poll" pin).
// --json → JSON output (TDS-07 §4 schema).

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

func init() {
	commands["status"] = command{fn: runStatus, summary: "Live status: active + recent instances"}
}

// statusActiveJSON is one element of the 'active' array in TDS-07 §4 status JSON schema.
type statusActiveJSON struct {
	InstanceID        string  `json:"instance_id"`
	WorkflowID        string  `json:"workflow_id"`
	Namespace         string  `json:"namespace"`
	Status            string  `json:"status"`
	CurrentStep       *string `json:"current_step"`
	SignalName        *string `json:"signal_name"`
	TimeoutRemainingS *int    `json:"timeout_remaining_s"`
	ElapsedS          int     `json:"elapsed_s"`
}

// statusRecentJSON is one element of the 'recent' array in TDS-07 §4 status JSON schema.
type statusRecentJSON struct {
	InstanceID  string  `json:"instance_id"`
	WorkflowID  string  `json:"workflow_id"`
	Namespace   string  `json:"namespace"`
	Status      string  `json:"status"`
	DurationMs  int     `json:"duration_ms"`
	FailedStep  *string `json:"failed_step"`
	CompletedAt string  `json:"completed_at"`
}

// statusOutputJSON is the full TDS-07 §4 status JSON schema.
type statusOutputJSON struct {
	Timestamp string             `json:"timestamp"`
	Active    []statusActiveJSON `json:"active"`
	Recent    []statusRecentJSON `json:"recent"`
}

func runStatus(args []string) {
	fs := newFlagSet("status")
	var namespace string
	var all bool
	var watch bool
	var n int
	fs.StringVar(&namespace, "namespace", "", "Filter by namespace (default: all)")
	fs.BoolVar(&all, "all", false, "Include all terminal instances")
	fs.BoolVar(&watch, "watch", false, "Re-poll every 1s until interrupted")
	fs.IntVar(&n, "n", 10, "Number of recent completed instances")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) > 0 {
		// Instance detail mode.
		instanceID := rest[0]
		runStatusDetail(instanceID)
		return
	}

	// Table mode.
	if watch {
		for {
			printStatusTable(namespace, all, n)
			time.Sleep(1 * time.Second)
		}
		// unreachable — user interrupts with Ctrl-C
	}
	printStatusTable(namespace, all, n)
}

// printStatusTable prints the status table for active + recent instances.
func printStatusTable(namespace string, all bool, n int) {
	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("status: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	ctx := context.Background()
	now := time.Now()

	// Collect active instances (running, waiting, compensating, pending).
	activeStatuses := []core.InstanceStatus{
		core.InstanceStatusRunning,
		core.InstanceStatusWaiting,
		core.InstanceStatusCompensating,
		core.InstanceStatusPending,
	}
	var activeInsts []core.WorkflowInstance
	for _, st := range activeStatuses {
		filter := core.InstanceFilter{Namespace: namespace, Status: st}
		insts, err := store.ListInstances(ctx, filter)
		if err != nil {
			continue
		}
		activeInsts = append(activeInsts, insts...)
	}

	// Collect recent terminal instances.
	terminalStatuses := []core.InstanceStatus{
		core.InstanceStatusCompleted,
		core.InstanceStatusFailed,
		core.InstanceStatusCancelled,
		core.InstanceStatusCompensated,
		core.InstanceStatusCompensationFailed,
	}
	var recentInsts []core.WorkflowInstance
	for _, st := range terminalStatuses {
		filter := core.InstanceFilter{Namespace: namespace, Status: st}
		insts, err := store.ListInstances(ctx, filter)
		if err != nil {
			continue
		}
		recentInsts = append(recentInsts, insts...)
	}

	// Sort recent by UpdatedAt descending (simple selection; stdlib sort is available
	// but we use a manual approach to avoid importing sort package).
	sortByUpdatedAt(recentInsts)
	if !all && len(recentInsts) > n {
		recentInsts = recentInsts[:n]
	}

	if globalJSON {
		out := buildStatusJSON(ctx, store, now, activeInsts, recentInsts)
		emitJSON(out)
		return
	}

	printStatusTableTo(os.Stdout, now, activeInsts, recentInsts, all, n)
}

// printStatusTableTo writes the human status table to w.
// Extracted for testability (status_test.go uses it with a bytes.Buffer).
func printStatusTableTo(w io.Writer, now time.Time, activeInsts, recentInsts []core.WorkflowInstance, all bool, n int) {
	// Human output (TDS-07 §4 shape):
	// AWIS status  2026-07-10 14:23:01
	//
	// ACTIVE
	//  ●  running   capture-decision  i-a1b2c3  step: draft-entry  12s elapsed
	//  ○  waiting   capture-decision  i-d4e5f6  signal: confirm-entry  71h 48m remaining
	//  ✗  failed    capture-decision  i-g7h8i9  step: publish-entry  3m ago
	//
	// RECENT (last 10)
	//  ✓  completed  capture-decision  i-j0k1l2  89s  2026-07-10 14:21:32

	_, _ = fmt.Fprintf(w, "AWIS status  %s\n", now.Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintln(w)

	_, _ = fmt.Fprintln(w, "ACTIVE")
	if len(activeInsts) == 0 {
		_, _ = fmt.Fprintln(w, " (none)")
	}
	for _, inst := range activeInsts {
		symbol := statusSymbol(inst.Status)
		step := currentStepStr(inst)
		elapsed := int(now.Sub(inst.StartedAt).Seconds())
		_, _ = fmt.Fprintf(w, " %s  %-9s %-20s  %s  %s  %ds elapsed\n",
			symbol, string(inst.Status), inst.DefinitionID, shortID(inst.InstanceID), step, elapsed)
	}

	_, _ = fmt.Fprintln(w)
	label := fmt.Sprintf("RECENT (last %d)", n)
	if all {
		label = "RECENT (all)"
	}
	_, _ = fmt.Fprintln(w, label)
	if len(recentInsts) == 0 {
		_, _ = fmt.Fprintln(w, " (none)")
	}
	for _, inst := range recentInsts {
		symbol := statusSymbol(inst.Status)
		var dur string
		if inst.CompletedAt != nil {
			d := int(inst.CompletedAt.Sub(inst.StartedAt).Seconds())
			dur = fmt.Sprintf("%ds", d)
		}
		at := inst.UpdatedAt.Format("2006-01-02 15:04:05")
		_, _ = fmt.Fprintf(w, " %s  %-10s %-20s  %s  %s  %s\n",
			symbol, string(inst.Status), inst.DefinitionID, shortID(inst.InstanceID), dur, at)
	}
}

// runStatusDetail prints detail for a single instance.
func runStatusDetail(instanceID string) {
	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("status: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	inst, err := store.GetInstance(context.Background(), core.InstanceID(instanceID))
	if err != nil {
		fail(1, fmt.Sprintf("instance %q not found", instanceID),
			"--data-dir "+globalDataDir,
			"awis status")
	}

	if globalJSON {
		now := time.Now()
		active := buildActiveJSON(now, inst, lookupWait(context.Background(), store, inst))
		emitJSON(active)
		return
	}

	now := time.Now()
	elapsed := int(now.Sub(inst.StartedAt).Seconds())
	fmt.Printf("Instance:  %s\n", inst.InstanceID)
	fmt.Printf("Workflow:  %s v%s  (namespace: %s)\n", inst.DefinitionID, inst.DefinitionVersion, inst.Namespace)
	fmt.Printf("Status:    %s\n", inst.Status)
	if len(inst.CurrentSteps) > 0 {
		fmt.Printf("Steps:     %s\n", strings.Join(inst.CurrentSteps, ", "))
	}
	if wait := lookupWait(context.Background(), store, inst); wait != nil {
		fmt.Printf("Awaiting:  signal %q  (awis signal %s %s)\n",
			wait.SignalName, inst.InstanceID, wait.SignalName)
		if wait.TimeoutAt != nil {
			remaining := int(wait.TimeoutAt.Sub(now).Seconds())
			if remaining < 0 {
				remaining = 0
			}
			fmt.Printf("Timeout:   %ds remaining (%s)\n", remaining, wait.TimeoutAction)
		}
	}
	fmt.Printf("Elapsed:   %ds\n", elapsed)
	if inst.CompletedAt != nil {
		dur := int(inst.CompletedAt.Sub(inst.StartedAt).Seconds())
		fmt.Printf("Duration:  %ds\n", dur)
	}
}

// buildStatusJSON builds the full statusOutputJSON from active and recent instances.
func buildStatusJSON(ctx context.Context, store any, now time.Time, active, recent []core.WorkflowInstance) statusOutputJSON {
	activeJSON := make([]statusActiveJSON, 0, len(active))
	for _, inst := range active {
		activeJSON = append(activeJSON, buildActiveJSON(now, inst, lookupWait(ctx, store, inst)))
	}
	recentJSON := make([]statusRecentJSON, 0, len(recent))
	for _, inst := range recent {
		recentJSON = append(recentJSON, buildRecentJSON(inst))
	}
	return statusOutputJSON{
		Timestamp: now.UTC().Format(time.RFC3339),
		Active:    activeJSON,
		Recent:    recentJSON,
	}
}

func buildActiveJSON(now time.Time, inst core.WorkflowInstance, wait *storage.WaitRecord) statusActiveJSON {
	var currentStep *string
	if len(inst.CurrentSteps) > 0 {
		s := inst.CurrentSteps[0]
		currentStep = &s
	}
	elapsedS := int(now.Sub(inst.StartedAt).Seconds())

	// signal_name / timeout_remaining_s are part of the published status JSON
	// schema (TDS-07 §4) but were never assigned: an instance reported as
	// `waiting` gave no indication of WHICH signal it awaited, so an operator
	// could not tell what to pass to `awis signal` without reading the
	// workflow YAML, and a UI could not render the wait at all. The answer was
	// already durable in wait_records the whole time (B-28).
	var signalName *string
	var timeoutRemainingS *int
	if wait != nil {
		n := wait.SignalName
		signalName = &n
		if wait.TimeoutAt != nil {
			// Report a clamped, whole-second countdown. A negative value would
			// mean the wait is already due and the timeout scan simply has not
			// run yet; surfacing that as a negative number reads as a bug to
			// whoever sees it, so it floors at 0.
			remaining := int(wait.TimeoutAt.Sub(now).Seconds())
			if remaining < 0 {
				remaining = 0
			}
			timeoutRemainingS = &remaining
		}
	}

	return statusActiveJSON{
		InstanceID:        string(inst.InstanceID),
		WorkflowID:        inst.DefinitionID,
		Namespace:         inst.Namespace,
		Status:            string(inst.Status),
		CurrentStep:       currentStep,
		SignalName:        signalName,
		TimeoutRemainingS: timeoutRemainingS,
		ElapsedS:          elapsedS,
	}
}

// waitRecordStore is the additive storage capability needed to describe a
// wait. Declared locally and type-asserted, the established pattern for
// reaching *SQLiteStorage methods that are not on the frozen 12-method
// core.StoragePort.
type waitRecordStore interface {
	ListWaitRecordsByInstance(ctx context.Context, instanceID core.InstanceID) ([]storage.WaitRecord, error)
}

// lookupWait returns the wait record describing why inst is parked, or nil.
//
// It queries only for instances whose status is actually `waiting`, so the
// cost is bounded by the number of genuinely parked instances rather than by
// the size of the active set. A store without the capability yields nil,
// which degrades to the previous (null) output rather than failing a
// read-only status command.
func lookupWait(ctx context.Context, store any, inst core.WorkflowInstance) *storage.WaitRecord {
	if inst.Status != core.InstanceStatusWaiting {
		return nil
	}
	ws, ok := store.(waitRecordStore)
	if !ok {
		return nil
	}
	records, err := ws.ListWaitRecordsByInstance(ctx, inst.InstanceID)
	if err != nil || len(records) == 0 {
		return nil
	}
	// An instance can hold several wait records (a parallel join of signal
	// steps). Prefer the one for the step status already reports as current,
	// so signal_name and current_step describe the same wait; otherwise fall
	// back to the first, which ListWaitRecordsByInstance orders by step_id for
	// determinism.
	if len(inst.CurrentSteps) > 0 {
		for i := range records {
			if records[i].StepID == inst.CurrentSteps[0] {
				return &records[i]
			}
		}
	}
	return &records[0]
}

func buildRecentJSON(inst core.WorkflowInstance) statusRecentJSON {
	var durationMs int
	var completedAt string
	if inst.CompletedAt != nil {
		durationMs = int(inst.CompletedAt.Sub(inst.StartedAt).Milliseconds())
		completedAt = inst.CompletedAt.UTC().Format(time.RFC3339)
	} else {
		completedAt = inst.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return statusRecentJSON{
		InstanceID:  string(inst.InstanceID),
		WorkflowID:  inst.DefinitionID,
		Namespace:   inst.Namespace,
		Status:      string(inst.Status),
		DurationMs:  durationMs,
		CompletedAt: completedAt,
	}
}

// statusSymbol returns the TDS-07 status symbol for an instance status.
func statusSymbol(status core.InstanceStatus) string {
	switch status {
	case core.InstanceStatusRunning:
		return "●"
	case core.InstanceStatusWaiting:
		return "○"
	case core.InstanceStatusCompleted, core.InstanceStatusCompensated:
		return "✓"
	case core.InstanceStatusFailed, core.InstanceStatusCompensationFailed:
		return "✗"
	case core.InstanceStatusCancelled:
		return "✗"
	case core.InstanceStatusCompensating:
		return "→"
	default:
		return "·"
	}
}

// currentStepStr returns "step: <id>" or "signal: <id>" based on instance status/steps.
func currentStepStr(inst core.WorkflowInstance) string {
	if len(inst.CurrentSteps) == 0 {
		return ""
	}
	step := inst.CurrentSteps[0]
	if inst.Status == core.InstanceStatusWaiting {
		return "signal: " + step
	}
	return "step: " + step
}

// shortID returns the full instance id (TDS-07 uses full IDs in status output).
func shortID(id core.InstanceID) string {
	return string(id)
}

// sortByUpdatedAt sorts instances in descending UpdatedAt order (most recent first).
// Uses a simple insertion sort sufficient for small result sets.
func sortByUpdatedAt(insts []core.WorkflowInstance) {
	for i := 1; i < len(insts); i++ {
		for j := i; j > 0 && insts[j].UpdatedAt.After(insts[j-1].UpdatedAt); j-- {
			insts[j], insts[j-1] = insts[j-1], insts[j]
		}
	}
}
