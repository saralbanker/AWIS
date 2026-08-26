package main

// trace.go — 'awis trace' command (TDS-07 §4; M14-C3 T8).
//
// Full execution trace for one instance. Reads events from the EventLog
// in chronological order. Human = TDS-07/QG-2 shape; --full includes payloads.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

func init() {
	commands["trace"] = command{fn: runTrace, summary: "Full execution trace for one instance"}
}

// truncatePayloadForTraceJSON truncates payload to at most 120 runes (never
// splitting a multi-byte rune) and marshals the result as a JSON string
// value via json.Marshal, so escaping (quotes, control characters, etc.) is
// always correct.
//
// Defect B-7: the previous implementation built the JSON string by hand
// (`"` + truncated + `"`), which produced invalid JSON whenever the raw
// payload contained an unescaped `"`. The resulting json.RawMessage then
// failed to encode, and the discarded encode error (see emitJSON, B-8)
// meant 'trace --json' silently printed nothing and exited 0.
func truncatePayloadForTraceJSON(payload json.RawMessage) (json.RawMessage, error) {
	truncated := string(payload)
	if r := []rune(truncated); len(r) > 120 {
		truncated = string(r[:120]) + "..."
	}
	b, err := json.Marshal(truncated)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// traceEventJSON is one event in the TDS-07 §4 trace JSON schema.
type traceEventJSON struct {
	Seq        int             `json:"seq"`
	EventType  string          `json:"event_type"`
	OccurredAt string          `json:"occurred_at"`
	RelativeMs int             `json:"relative_ms"`
	Payload    json.RawMessage `json:"payload"`
}

// traceOutputJSON is the full TDS-07 §4 trace JSON schema.
type traceOutputJSON struct {
	InstanceID      string           `json:"instance_id"`
	WorkflowID      string           `json:"workflow_id"`
	WorkflowVersion string           `json:"workflow_version"`
	Namespace       string           `json:"namespace"`
	Status          string           `json:"status"`
	DurationMs      *int             `json:"duration_ms"`
	Trigger         *string          `json:"trigger"`
	Events          []traceEventJSON `json:"events"`
}

func runTrace(args []string) {
	fs := newFlagSet("trace")
	var full bool
	fs.BoolVar(&full, "full", false, "Do not truncate step outputs (default: truncated to 120 chars)")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "trace requires an <instance-id> argument", "", "awis trace <instance-id>")
	}
	instanceID := core.InstanceID(rest[0])

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("trace: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	ctx := context.Background()

	// Fetch instance.
	inst, serr := store.GetInstance(ctx, instanceID)
	if serr != nil {
		fail(1, fmt.Sprintf("instance %q not found", instanceID),
			"--data-dir "+globalDataDir,
			"awis status")
	}

	// Read all events.
	events, err := store.ReadEvents(ctx, instanceID, 0)
	if err != nil {
		fail(1, fmt.Sprintf("trace: cannot read events: %s", err), string(instanceID), "check storage integrity")
	}

	// Determine duration.
	var durationMs *int
	if inst.CompletedAt != nil {
		d := int(inst.CompletedAt.Sub(inst.StartedAt).Milliseconds())
		durationMs = &d
	}

	if globalJSON {
		eventsJSON := make([]traceEventJSON, 0, len(events))
		for _, ev := range events {
			payload := ev.Payload
			if payload == nil {
				payload = json.RawMessage("{}")
			}
			if !full && len(payload) > 120 {
				tp, terr := truncatePayloadForTraceJSON(payload)
				if terr != nil {
					fail(1, fmt.Sprintf("trace: cannot marshal truncated payload: %s", terr), string(instanceID), "")
				}
				payload = tp
			}
			relMs := int(ev.EmittedAt.Sub(inst.StartedAt).Milliseconds())
			if relMs < 0 {
				relMs = 0
			}
			eventsJSON = append(eventsJSON, traceEventJSON{
				Seq:        ev.SequenceNum,
				EventType:  string(ev.EventType),
				OccurredAt: ev.EmittedAt.UTC().Format(time.RFC3339),
				RelativeMs: relMs,
				Payload:    payload,
			})
		}
		out := traceOutputJSON{
			InstanceID:      string(inst.InstanceID),
			WorkflowID:      inst.DefinitionID,
			WorkflowVersion: string(inst.DefinitionVersion),
			Namespace:       inst.Namespace,
			Status:          string(inst.Status),
			DurationMs:      durationMs,
			Events:          eventsJSON,
		}
		emitJSON(out)
		return
	}

	// Human output (TDS-07 §4):
	// Trace: capture-decision / i-a1b2c3
	// Status: completed ✓  Duration: 89s  Trigger: manual
	//
	// Timeline:
	//   00:00  ● WorkflowStarted
	//   00:00  ► StepStarted      draft-entry (attempt 1)
	//   00:13  ✓ StepCompleted    draft-entry  13.1s   adapter: anthropic, tokens: 847
	//   ...
	statusSymStr := statusSymbol(inst.Status)
	fmt.Printf("Trace: %s / %s\n", inst.DefinitionID, inst.InstanceID)
	durStr := ""
	if durationMs != nil {
		durStr = fmt.Sprintf("  Duration: %ds", *durationMs/1000)
	}
	fmt.Printf("Status: %s %s%s\n", inst.Status, statusSymStr, durStr)
	fmt.Println()
	fmt.Println("Timeline:")

	startedAt := inst.StartedAt
	for _, ev := range events {
		elapsed := ev.EmittedAt.Sub(startedAt)
		mins := int(elapsed.Minutes())
		secs := int(elapsed.Seconds()) % 60
		timeStr := fmt.Sprintf("%02d:%02d", mins, secs)

		sym := traceEventSymbol(ev.EventType)
		evStr := formatTraceEvent(ev, full)
		fmt.Printf("  %s  %s %s\n", timeStr, sym, evStr)
	}
}

// traceEventSymbol returns the TDS-07 symbol for a trace event type.
// Symbols: ● info, ► started, ✓ completed, ✗ failed, ○ waiting, → transition
func traceEventSymbol(et core.EventType) string {
	switch et {
	case core.EventTypeWorkflowStarted:
		return "●"
	case core.EventTypeStepStarted:
		return "►"
	case core.EventTypeStepCompleted, core.EventTypeWorkflowCompleted, core.EventTypeWorkflowCompensated:
		return "✓"
	case core.EventTypeStepFailed, core.EventTypeWorkflowFailed, core.EventTypeWorkflowCompensationFailed:
		return "✗"
	case core.EventTypeSignalReceived:
		return "✓"
	case core.EventTypeStepFallbackActivated, core.EventTypeWorkflowCompensating:
		return "→"
	case core.EventTypeWorkflowCancelled:
		return "✗"
	default:
		return "●"
	}
}

// formatTraceEvent formats a trace event for human display (TDS-07 §4 timeline shape).
func formatTraceEvent(ev core.ExecutionEvent, full bool) string {
	typeName := string(ev.EventType)

	// Parse payload for additional context.
	var payload map[string]any
	if len(ev.Payload) > 0 {
		_ = json.Unmarshal(ev.Payload, &payload)
	}

	switch ev.EventType {
	case core.EventTypeWorkflowStarted:
		return "WorkflowStarted"

	case core.EventTypeStepStarted:
		stepID := strFromMap(payload, "step_id")
		attempt := intFromMap(payload, "attempt")
		if stepID != "" {
			return fmt.Sprintf("%-20s %s (attempt %d)", "StepStarted", stepID, attempt)
		}
		return fmt.Sprintf("%-20s %s", "StepStarted", ev.StepID)

	case core.EventTypeStepCompleted:
		stepID := strFromMap(payload, "step_id")
		durMs := intFromMap(payload, "duration_ms")
		if stepID == "" {
			stepID = ev.StepID
		}
		durSec := float64(durMs) / 1000.0
		return fmt.Sprintf("%-20s %-20s  %.1fs", "StepCompleted", stepID, durSec)

	case core.EventTypeStepFailed:
		stepID := strFromMap(payload, "step_id")
		if stepID == "" {
			stepID = ev.StepID
		}
		return fmt.Sprintf("%-20s %-20s", "StepFailed", stepID)

	case core.EventTypeSignalReceived:
		sigName := strFromMap(payload, "signal_name")
		return fmt.Sprintf("%-20s %s", "SignalReceived", sigName)

	case core.EventTypeWorkflowCompleted:
		durMs := intFromMap(payload, "duration_ms")
		durSec := float64(durMs) / 1000.0
		return fmt.Sprintf("%-20s  total: %.0fs", "WorkflowCompleted", durSec)

	case core.EventTypeWorkflowFailed:
		return "WorkflowFailed"

	case core.EventTypeWorkflowCancelled:
		return "WorkflowCancelled"

	case core.EventTypeWorkflowCompensating:
		return "WorkflowCompensating"

	case core.EventTypeWorkflowCompensated:
		return "WorkflowCompensated"

	case core.EventTypeWorkflowCompensationFailed:
		return "WorkflowCompensationFailed"

	case core.EventTypeStepFallbackActivated:
		fallback := strFromMap(payload, "fallback_step_id")
		return fmt.Sprintf("%-20s → %s", "StepFallbackActivated", fallback)
	}

	detail := typeName
	if full && len(ev.Payload) > 0 {
		detail += "  " + string(ev.Payload)
	}
	return detail
}

func strFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func intFromMap(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}
