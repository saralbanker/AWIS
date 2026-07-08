package engine

// SIGNAL_SCAN end-to-end fixtures (M07-C2): WAIT-step entry → intake → atomic
// B3 delivery → resume, driven through the real Engine + real SQLite storage.
// Each fixture asserts the EXACT ordered event stream and proves EDR-007
// forward ≡ RebuildState at the resumed (running) state where the forward and
// replay paths reconverge (the transient `waiting` status is the non-evented
// EDR-007 §9 gap and is not equivalence-checked).

import (
	"context"
	"testing"

	"github.com/awis/awis/internal/core"
)

func signalStep(id, signalName string) core.Step {
	return core.Step{
		ID: id, Name: id, Type: core.StepTypeSignal,
		WaitSignal: &core.WaitConfig{SignalName: signalName, TimeoutAction: "fail"},
	}
}

// TestSignalScan_WaitDeliverResume drives the minimal WAIT → deliver → resume
// path for a signal InitialStep and proves the exact event sequence, the resume
// to running, the SignalReceived payload, forward ≡ rebuild, and double-delivery
// idempotence (NFR-R-04).
func TestSignalScan_WaitDeliverResume(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.sig", Version: "1.0.0", Namespace: "t", Name: "sig",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{signalStep("w", "go")},
		InitialStep: "w", FinalSteps: []string{"w"}, Metadata: map[string]any{},
	}
	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now)
	ctx := context.Background()

	iid, err := e.Submit(ctx, "t.sig", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick 1: the signal step activates and parks the instance in `waiting`.
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 1: %v", err)
	}
	inst, _ := s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusWaiting {
		t.Fatalf("after WAIT entry status = %q, want waiting", inst.Status)
	}
	if len(inst.CurrentSteps) != 1 || inst.CurrentSteps[0] != "w" {
		t.Fatalf("current_steps = %v, want [w]", inst.CurrentSteps)
	}
	pairs, _ := eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "w"},
	})

	// A tick with no signal in the inbox delivers nothing (instance stays waiting).
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick (no signal): %v", err)
	}
	if inst, _ = s.GetInstance(ctx, iid); inst.Status != core.InstanceStatusWaiting {
		t.Fatalf("status = %q, want still waiting (no signal yet)", inst.Status)
	}

	// Intake the awaited signal, then the next SIGNAL_SCAN delivers it atomically
	// and (OUTPUT 0) immediately completes the WAIT step with the payload as outputs.
	if err := e.Signal(ctx, iid, "go", map[string]any{"by": "alice"}); err != nil {
		t.Fatalf("Signal: %v", err)
	}
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 2 (deliver+complete): %v", err)
	}

	// After delivery + step completion: instance is running with current_steps=[].
	inst, _ = s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusRunning {
		t.Fatalf("after delivery status = %q, want running (resumed)", inst.Status)
	}
	if len(inst.CurrentSteps) != 0 {
		t.Fatalf("after step completion current_steps = %v, want [] (step removed)", inst.CurrentSteps)
	}

	// Event sequence after Tick 2: WorkflowStarted, StepStarted("w"),
	// SignalReceived, StepCompleted("w"). No WorkflowCompleted yet (next tick).
	pairs, evs := eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "w"},
		{core.EventTypeSignalReceived, ""},
		{core.EventTypeStepCompleted, "w"}, // OUTPUT 0: step completes at delivery
	})

	// SignalReceived payload VERBATIM {signal_name, payload: map} (EVENTLOG §6).
	sr := findEvent(t, evs, core.EventTypeSignalReceived, "")
	var srp struct {
		SignalName string         `json:"signal_name"`
		Payload    map[string]any `json:"payload"`
	}
	decodePayload(t, sr, &srp)
	if srp.SignalName != "go" {
		t.Fatalf("SignalReceived.signal_name = %q, want go", srp.SignalName)
	}
	assertJSONEqual(t, srp.Payload, map[string]any{"by": "alice"})

	// StepCompleted outputs = signal payload (OUTPUT 0; TDS-01 StepCompleted REPLAY).
	sc := findEvent(t, evs, core.EventTypeStepCompleted, "w")
	var scp struct {
		Outputs map[string]any `json:"outputs"`
	}
	decodePayload(t, sc, &scp)
	assertJSONEqual(t, scp.Outputs, map[string]any{"by": "alice"})

	// Wait_record must be gone after step completion (OUTPUT 0).
	_, wrFound, err := s.GetWaitRecord(ctx, iid, "w")
	if err != nil {
		t.Fatalf("GetWaitRecord: %v", err)
	}
	if wrFound {
		t.Fatalf("wait_record must be deleted on step completion (OUTPUT 0)")
	}

	// Tick 3: the signal step was the FinalStep; processInstance now sees
	// current_steps=[] and the final completed → WorkflowCompleted.
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 3 (workflow complete): %v", err)
	}
	inst, _ = s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("after WorkflowCompleted status = %q, want completed", inst.Status)
	}
	pairs, _ = eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "w"},
		{core.EventTypeSignalReceived, ""},
		{core.EventTypeStepCompleted, "w"},
		{core.EventTypeWorkflowCompleted, ""},
	})

	// forward ≡ rebuild (at completed state where forward and replay paths converge).
	assertProjectionEquivalence(t, s, iid)

	// Re-scan after completion: idempotent — no new events (NFR-R-04).
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 4 (idempotent): %v", err)
	}
	pairs2, _ := eventPairs(t, s, iid)
	if len(pairs2) != 5 {
		t.Fatalf("re-scan changed the event count: %d, want 5 (idempotent)", len(pairs2))
	}
}

// TestSignalScan_ResumesExactStepAfterNativeStep proves the wait matches the
// correct step: a native step completes first, then the signal step parks and is
// resumed only by its named signal — a differently-named signal never delivers.
func TestSignalScan_ResumesExactStepAfterNativeStep(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.sig2", Version: "1.0.0", Namespace: "t", Name: "sig2",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "ha"),
			signalStep("w", "go"),
		},
		Transitions: []core.Transition{{From: "a", To: "w"}},
		InitialStep: "a", FinalSteps: []string{"w"}, Metadata: map[string]any{},
	}
	clk := newManualClock(engineStart)
	ha := &stepHandler{id: "ha", outputs: map[string]any{"ra": "ok"}}
	e, s := newNativeEngine(t, def, 1, clk.now, ha)
	ctx := context.Background()

	iid, err := e.Submit(ctx, "t.sig2", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick until the signal step parks the instance (a runs+completes, then w waits).
	waiting := false
	for i := 0; i < 5 && !waiting; i++ {
		if err := e.Tick(ctx); err != nil {
			t.Fatalf("Tick %d: %v", i, err)
		}
		inst, _ := s.GetInstance(ctx, iid)
		waiting = inst.Status == core.InstanceStatusWaiting
	}
	if !waiting {
		t.Fatalf("instance never reached waiting")
	}
	pairs, _ := eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepCompleted, "a"},
		{core.EventTypeStepStarted, "w"},
	})

	// A signal the step is NOT waiting for must never resume it (no live wait
	// matches "nope").
	if err := e.Signal(ctx, iid, "nope", nil); err != nil {
		t.Fatalf("Signal(nope): %v", err)
	}
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick after wrong signal: %v", err)
	}
	if inst, _ := s.GetInstance(ctx, iid); inst.Status != core.InstanceStatusWaiting {
		t.Fatalf("wrong-named signal resumed the wait: status = %q", inst.Status)
	}

	// The correct signal resumes exactly step w.
	if err := e.Signal(ctx, iid, "go", nil); err != nil {
		t.Fatalf("Signal(go): %v", err)
	}
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick deliver: %v", err)
	}
	inst, _ := s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusRunning {
		t.Fatalf("status = %q, want running after correct signal", inst.Status)
	}
	// After delivery tick: SignalReceived + StepCompleted("w") emitted (OUTPUT 0).
	pairs, _ = eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepCompleted, "a"},
		{core.EventTypeStepStarted, "w"},
		{core.EventTypeSignalReceived, ""},
		{core.EventTypeStepCompleted, "w"}, // OUTPUT 0: step completes at delivery
	})

	// forward ≡ rebuild: after StepCompleted, forward and rebuild both produce
	// running with w in completed_steps. The waiting gap (EDR-007 §9) has resolved.
	assertProjectionEquivalence(t, s, iid)
}
