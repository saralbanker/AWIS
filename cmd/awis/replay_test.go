package main

// replay_test.go — tests for 'awis replay' command (M17-C1 TDS-07 §3).
//
// Tests:
//   - JSON schema has required fields
//   - replayAction returns non-empty strings for all known event types
//   - dry_run always true in JSON output
//   - Goldens not needed for replay (no stable empty-state; instance required)

import (
	"encoding/json"
	"testing"

	"github.com/awis/awis/internal/core"
)

// TestReplayOutputJSONShape verifies replayOutputJSON has required fields.
func TestReplayOutputJSONShape(t *testing.T) {
	out := replayOutputJSON{
		InstanceID: "i-abc123",
		WorkflowID: "capture-decision",
		Namespace:  "default",
		Status:     "completed",
		DryRun:     true,
		Steps:      []replayStepJSON{},
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"instance_id", "workflow_id", "namespace", "status", "dry_run", "steps"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("replayOutputJSON: missing field %q", field)
		}
	}
	// dry_run must always be true.
	if dr, ok := parsed["dry_run"].(bool); !ok || !dr {
		t.Error("replayOutputJSON: dry_run must be true")
	}
}

// TestReplayStepJSONShape verifies replayStepJSON has required fields.
func TestReplayStepJSONShape(t *testing.T) {
	s := replayStepJSON{
		Order:      1,
		EventType:  "WorkflowStarted",
		EmittedAt:  "2026-07-10T14:00:00Z",
		RelativeMs: 0,
		Action:     "initialize workflow state",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"order", "event_type", "emitted_at", "relative_ms", "action"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("replayStepJSON: missing field %q", field)
		}
	}
}

// TestReplayAction verifies that known event types produce non-empty action strings.
func TestReplayAction(t *testing.T) {
	knownTypes := []core.EventType{
		core.EventTypeWorkflowStarted,
		core.EventTypeStepStarted,
		core.EventTypeStepCompleted,
		core.EventTypeStepFailed,
		core.EventTypeSignalReceived,
		core.EventTypeWorkflowCompleted,
		core.EventTypeWorkflowFailed,
		core.EventTypeWorkflowCancelled,
		core.EventTypeWorkflowCompensating,
		core.EventTypeWorkflowCompensated,
		core.EventTypeWorkflowCompensationFailed,
		core.EventTypeStepFallbackActivated,
	}
	for _, et := range knownTypes {
		action := replayAction(et)
		if action == "" {
			t.Errorf("replayAction(%q) returned empty string", et)
		}
	}
	// Unknown type should return non-empty fallback.
	if action := replayAction("UnknownEventType"); action == "" {
		t.Error("replayAction(unknown) should return non-empty fallback")
	}
}
