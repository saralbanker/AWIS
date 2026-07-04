package engine

// SignalReceived projection coverage (EDR-011 §8 note). SignalReceived has NO
// forward emission site at M06 — signals are M07's SIGNAL_SCAN. This test drives
// the 12th event type through the engine's OWN forward projection path (e.emit)
// by appending a SignalReceived event directly after WorkflowStarted, then proves
// forward ≡ RebuildState for the sequence. It documents the honest limitation:
// the emission site (SIGNAL_SCAN) arrives with M07.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/awis/awis/internal/core"
)

func TestSignalReceived_ProjectionCoverage(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.sig", Version: "1.0.0", Namespace: "t", Name: "sig",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("a", "ha")},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}})
	ctx := context.Background()

	// Submit only (WorkflowStarted, running). Do NOT tick — no step should run.
	iid, err := e.Submit(ctx, "t.sig", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Append a SignalReceived event through the engine's own forward-projection
	// path (e.emit): appendEvent + project, exactly as the M07 emission site will.
	payload, err := json.Marshal(map[string]any{"signal_name": "go", "payload": map[string]any{}})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	ev := core.ExecutionEvent{
		InstanceID: iid,
		Namespace:  def.Namespace,
		EventType:  core.EventTypeSignalReceived,
		Payload:    payload,
		EmittedAt:  clk.now(),
	}
	if err := e.emit(ctx, &ev, def.ID, def.Version); err != nil {
		t.Fatalf("emit SignalReceived: %v", err)
	}

	// Forward projection: SignalReceived keeps the instance running (EDR-007 §2).
	inst, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if inst.Status != core.InstanceStatusRunning {
		t.Fatalf("status = %q, want running", inst.Status)
	}

	// The 12th event type is present in the log, and forward ≡ RebuildState holds.
	pairs, _ := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeSignalReceived, ""},
	}
	assertPairs(t, pairs, want)

	assertProjectionEquivalence(t, s, iid)
}
