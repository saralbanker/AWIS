package engine

// Terminal-failure routing E2E (EDR-011 §8): retry-exhaust → fallback activation
// (bypassing the join gate) and on_error-transition routing (targets run, NO
// WorkflowFailed). Every fixture ends with forward ≡ rebuild equivalence.

import (
	"context"
	"testing"

	"github.com/awis/awis/internal/core"
)

// TestRetryExhaustThenFallback: step a retries once, exhausts, then activates its
// fallback fb directly; fb completes and the workflow completes.
func TestRetryExhaustThenFallback(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.fb", Version: "1.0.0", Namespace: "t", Name: "fb",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha",
				Retry:    &core.RetryPolicy{Attempts: 2, Backoff: "immediate"},
				Fallback: "fb"},
			{ID: "fb", Name: "fb", Type: core.StepTypeNative, Handler: "hfb"},
		},
		InitialStep: "a", FinalSteps: []string{"fb"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", err: errBoom}
	hfb := &stepHandler{id: "hfb", outputs: map[string]any{"r": "recovered"}}

	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, ha, hfb)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.fb", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"},          // attempt 1
		{core.EventTypeStepFailed, "a"},           // retrying:true
		{core.EventTypeStepStarted, "a"},          // attempt 2
		{core.EventTypeStepFailed, "a"},           // retrying:false (exhausted)
		{core.EventTypeStepFallbackActivated, "a"},
		{core.EventTypeStepStarted, "fb"},
		{core.EventTypeStepCompleted, "fb"},
		{core.EventTypeWorkflowCompleted, ""},
	}
	assertPairs(t, pairs, want)

	// StepFallbackActivated payload: {step_id, fallback_step_id, reason}.
	fa := findEvent(t, evs, core.EventTypeStepFallbackActivated, "a")
	var fap stepFallbackActivatedPayload
	decodePayload(t, fa, &fap)
	if fap.StepID != "a" || fap.FallbackStepID != "fb" {
		t.Fatalf("fallback payload = %+v, want step_id=a fallback_step_id=fb", fap)
	}
	if fap.Reason == "" {
		t.Fatalf("fallback payload reason must be non-empty (carries the error code)")
	}

	assertProjectionEquivalence(t, s, iid)
}

// TestOnErrorTransitionRouting: a failed step with a firing on_error transition
// routes to the target (which runs); NO WorkflowFailed is emitted.
func TestOnErrorTransitionRouting(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.onerr", Version: "1.0.0", Namespace: "t", Name: "onerr",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"},
			{ID: "b", Name: "b", Type: core.StepTypeNative, Handler: "hb"},
		},
		Transitions: []core.Transition{
			{From: "a", To: "b", OnError: true},
		},
		InitialStep: "a", FinalSteps: []string{"b"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", err: errBoom}
	hb := &stepHandler{id: "hb", outputs: map[string]any{"r": "handled"}}

	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, ha, hb)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.onerr", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed (on_error route succeeds)", inst.Status)
	}

	pairs, _ := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"},
		{core.EventTypeStepFailed, "a"}, // retrying:false, but NO WorkflowFailed
		{core.EventTypeStepStarted, "b"},
		{core.EventTypeStepCompleted, "b"},
		{core.EventTypeWorkflowCompleted, ""},
	}
	assertPairs(t, pairs, want)

	// Explicitly: no WorkflowFailed anywhere in the stream (EDR-011 §8 case 2).
	if idx := indexer(pairs)(core.EventTypeWorkflowFailed, ""); idx != -1 {
		t.Fatalf("on_error routing must NOT emit WorkflowFailed; found at %d", idx)
	}

	assertProjectionEquivalence(t, s, iid)
}
