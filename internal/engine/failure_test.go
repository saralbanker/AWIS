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
		{core.EventTypeStepStarted, "a"}, // attempt 1
		{core.EventTypeStepFailed, "a"},  // retrying:true
		{core.EventTypeStepStarted, "a"}, // attempt 2
		{core.EventTypeStepFailed, "a"},  // retrying:false (exhausted)
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

// ── B-4: terminally-failed steps never re-activate ─────────────────────────
//
// The defect: completedSet derives from Variables keys; a terminally-failed
// step is removed from current_steps by its terminal StepFailed but never
// enters Variables (only a StepCompleted does that). Without the B-4 guard
// (failure.go's e.failed / markFailed, consulted by activatableFor), the join
// gate saw the failed step's upstream predecessor as completed and the failed
// step itself as neither completed nor running — so it re-nominated the
// failed step on EVERY tick forever (activatableFor never excluded it), and
// the workflow hung in `running`, endlessly re-dispatching the already-failed
// step. All three fixtures below use a real UPSTREAM predecessor `p` (not the
// InitialStep itself) so the failed step's join-gate inbound check
// (completed[p]==true) is satisfied — this is the exact shape that exposed
// the hang; a failed step that IS the InitialStep does not reliably reproduce
// it (the InitialStep only self-activates while len(completed)==0).

// TestB4_TerminallyFailedStepNotReactivated_OnError: p completes, a fails
// terminally (no retry policy) and routes via on_error to b, which succeeds.
// The workflow must reach `completed` within runToTerminal's bound, and a
// must be dispatched exactly once.
func TestB4_TerminallyFailedStepNotReactivated_OnError(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.b4onerr", Version: "1.0.0", Namespace: "t", Name: "b4onerr",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("p", "hp"),
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"},
			nativeStep("b", "hb"),
		},
		Transitions: []core.Transition{
			{From: "p", To: "a"},
			{From: "a", To: "b", OnError: true},
		},
		InitialStep: "p", FinalSteps: []string{"b"}, Metadata: map[string]any{},
	}
	hp := &stepHandler{id: "hp", outputs: map[string]any{"rp": "p-done"}}
	ha := &stepHandler{id: "ha", err: errBoom}
	hb := &stepHandler{id: "hb", outputs: map[string]any{"rb": "handled"}}

	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, hp, ha, hb)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.b4onerr", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// runToTerminal is bounded (200 ticks): before the B-4 fix this instance
	// never reaches a terminal status and the bound trips (test failure), which
	// IS the observable "hang" signature for this defect.
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed (on_error route must not hang)", inst.Status)
	}

	pairs, _ := eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("terminally-failed step 'a' must NOT be re-dispatched: got %d StepStarted, want 1 real attempt (B-4)", n)
	}
	if n := countPairs(pairs, core.EventTypeStepFailed, "a"); n != 1 {
		t.Fatalf("step 'a' StepFailed count = %d, want 1 (no phantom re-fails)", n)
	}
	if n := countPairs(pairs, core.EventTypeStepStarted, "b"); n != 1 {
		t.Fatalf("on_error target 'b' StepStarted count = %d, want 1", n)
	}

	assertProjectionEquivalence(t, s, iid)
}

// TestB4_TerminallyFailedStepNotReactivated_Fallback: same shape as the
// on_error fixture above, but a routes via step.Fallback instead.
//
// Honest scoping: this test does NOT exercise the B-4 guard, and passes with
// or without it. The fallback route is structurally immune, because
// StepFallbackActivated's projection (emit.go) writes a sentinel into
// Variables for the ORIGINATING step so convergent join gates can treat it as
// done — which also puts it in completedSet, so activatableFor's
// `completed[id]` check already excludes it before `failed[id]` is consulted.
//
// It is kept because that immunity is a property of a projection rule in
// another file that nobody would think to protect: if the sentinel write were
// ever removed, this fixture would start hanging, and the failure would point
// straight at the interaction. on_error is the only route where B-4 can
// actually occur, and TestB4_..._OnError is the test that guards it.
func TestB4_TerminallyFailedStepNotReactivated_Fallback(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.b4fb", Version: "1.0.0", Namespace: "t", Name: "b4fb",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("p", "hp"),
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha", Fallback: "fb"},
			nativeStep("fb", "hfb"),
		},
		Transitions: []core.Transition{
			{From: "p", To: "a"},
		},
		InitialStep: "p", FinalSteps: []string{"fb"}, Metadata: map[string]any{},
	}
	hp := &stepHandler{id: "hp", outputs: map[string]any{"rp": "p-done"}}
	ha := &stepHandler{id: "ha", err: errBoom}
	hfb := &stepHandler{id: "hfb", outputs: map[string]any{"r": "recovered"}}

	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, hp, ha, hfb)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.b4fb", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed (fallback route must not hang)", inst.Status)
	}

	pairs, _ := eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("terminally-failed step 'a' must NOT be re-dispatched: got %d StepStarted, want 1 real attempt (B-4)", n)
	}
	if n := countPairs(pairs, core.EventTypeStepStarted, "fb"); n != 1 {
		t.Fatalf("fallback target 'fb' StepStarted count = %d, want 1", n)
	}

	assertProjectionEquivalence(t, s, iid)
}

// TestB4_TerminallyFailedStepNotReactivated_WorkflowFailed: same shape, but a
// has no fallback and no firing on_error transition, so it routes straight to
// WorkflowFailed (routeTerminalFailure case 3). Included for shape parity with
// the fallback/on_error fixtures above (same p→a predecessor structure); the
// instance reaches `failed` immediately in this branch, so the B-4 hang risk
// does not actually arise here (the instance leaves `running` in the same
// tick), but the re-dispatch-count assertion still guards the shape.
func TestB4_TerminallyFailedStepNotReactivated_WorkflowFailed(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.b4wf", Version: "1.0.0", Namespace: "t", Name: "b4wf",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("p", "hp"),
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"},
		},
		Transitions: []core.Transition{
			{From: "p", To: "a"},
		},
		InitialStep: "p", FinalSteps: []string{}, Metadata: map[string]any{},
	}
	hp := &stepHandler{id: "hp", outputs: map[string]any{"rp": "p-done"}}
	ha := &stepHandler{id: "ha", err: errBoom}

	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, hp, ha)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.b4wf", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusFailed {
		t.Fatalf("status = %q, want failed", inst.Status)
	}

	pairs, _ := eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("terminally-failed step 'a' must NOT be re-dispatched: got %d StepStarted, want 1 real attempt (B-4)", n)
	}
	if n := countPairs(pairs, core.EventTypeWorkflowFailed, ""); n != 1 {
		t.Fatalf("WorkflowFailed count = %d, want 1", n)
	}

	assertProjectionEquivalence(t, s, iid)
}
