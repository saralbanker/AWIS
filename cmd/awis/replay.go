package main

// replay.go — 'awis replay' command (TDS-07 §3 M17; M17-C1; PRD Should-Have).
//
// replay <instance-id>
// DRY-RUN: re-walk recorded events and print what WOULD run. No state writes.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/awis/awis/internal/core"
)

func init() {
	commands["replay"] = command{fn: runReplay, summary: "Re-run completed instance (dry-run)"}
}

// replayStepJSON is one step in the replay dry-run plan.
type replayStepJSON struct {
	Order      int    `json:"order"`
	EventType  string `json:"event_type"`
	StepID     string `json:"step_id,omitempty"`
	EmittedAt  string `json:"emitted_at"`
	RelativeMs int    `json:"relative_ms"`
	Action     string `json:"action"`
}

// replayOutputJSON is the JSON schema for 'awis replay --json'.
type replayOutputJSON struct {
	InstanceID string           `json:"instance_id"`
	WorkflowID string           `json:"workflow_id"`
	Namespace  string           `json:"namespace"`
	Status     string           `json:"status"`
	DryRun     bool             `json:"dry_run"`
	Steps      []replayStepJSON `json:"steps"`
}

func runReplay(args []string) {
	fs := newFlagSet("replay")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "replay requires an <instance-id> argument", "", "awis replay <instance-id>")
	}
	instanceID := core.InstanceID(rest[0])

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("replay: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	ctx := context.Background()

	inst, serr := store.GetInstance(ctx, instanceID)
	if serr != nil {
		fail(1, fmt.Sprintf("instance %q not found", instanceID),
			"--data-dir "+globalDataDir,
			"awis history")
	}

	events, err := store.ReadEvents(ctx, instanceID, 0)
	if err != nil {
		fail(1, fmt.Sprintf("replay: cannot read events: %s", err), string(instanceID), "check storage integrity")
	}

	// Build the dry-run plan: for each event, compute what would run.
	steps := make([]replayStepJSON, 0, len(events))
	for i, ev := range events {
		action := replayAction(ev.EventType)
		relMs := int(ev.EmittedAt.Sub(inst.StartedAt).Milliseconds())
		if relMs < 0 {
			relMs = 0
		}
		steps = append(steps, replayStepJSON{
			Order:      i + 1,
			EventType:  string(ev.EventType),
			StepID:     ev.StepID,
			EmittedAt:  ev.EmittedAt.UTC().Format(time.RFC3339),
			RelativeMs: relMs,
			Action:     action,
		})
	}

	if globalJSON {
		out := replayOutputJSON{
			InstanceID: string(inst.InstanceID),
			WorkflowID: inst.DefinitionID,
			Namespace:  inst.Namespace,
			Status:     string(inst.Status),
			DryRun:     true,
			Steps:      steps,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	// Human output.
	fmt.Printf("REPLAY (dry-run)  %s / %s\n", inst.DefinitionID, inst.InstanceID)
	fmt.Printf("Status: %s  (original)\n\n", inst.Status)
	fmt.Println("The following steps WOULD run if this instance were re-executed:")
	fmt.Println()

	for _, s := range steps {
		mins := s.RelativeMs / 60000
		secs := (s.RelativeMs / 1000) % 60
		timeStr := fmt.Sprintf("%02d:%02d", mins, secs)
		sym := traceEventSymbol(core.EventType(s.EventType))
		stepInfo := s.EventType
		if s.StepID != "" {
			stepInfo += "  " + s.StepID
		}
		fmt.Printf("  %s  %s %-30s → %s\n", timeStr, sym, stepInfo, s.Action)
	}
	fmt.Println()
	fmt.Println("NOTE: This is a dry-run. No steps were executed and no state was written.")
}

// replayAction returns a human description of what would happen for a given event type.
func replayAction(et core.EventType) string {
	switch et {
	case core.EventTypeWorkflowStarted:
		return "initialize workflow state"
	case core.EventTypeStepStarted:
		return "dispatch step to worker"
	case core.EventTypeStepCompleted:
		return "record step result; advance to next step"
	case core.EventTypeStepFailed:
		return "record failure; apply retry policy or advance to fallback"
	case core.EventTypeSignalReceived:
		return "unblock waiting instance; resume execution"
	case core.EventTypeWorkflowCompleted:
		return "mark instance completed"
	case core.EventTypeWorkflowFailed:
		return "mark instance failed"
	case core.EventTypeWorkflowCancelled:
		return "mark instance cancelled"
	case core.EventTypeWorkflowCompensating:
		return "begin compensation plan"
	case core.EventTypeWorkflowCompensated:
		return "mark instance compensated"
	case core.EventTypeWorkflowCompensationFailed:
		return "mark compensation failed"
	case core.EventTypeStepFallbackActivated:
		return "activate fallback step"
	default:
		return "process event"
	}
}
