package engine

// restart_test.go — engine-hardening Step 1 regression (closes B-0): the
// optimistic-lock version guarding UpsertInstance must be read from durable
// storage, not trusted from an in-memory cache that starts empty on every new
// process. Before the fix, a brand-new Engine constructed over the SAME
// storage (simulating a process restart) failed its very first tick with the
// verbatim error:
//
//	engine: upsert projection StepStarted (instance=…): storage: version conflict: instance_id=… expected=…
//
// because project() used e.ver[iid] (zero in a fresh process) as the expected
// OCC version while workflow_instances.version was already >0 (durable and
// correct). See emit.go's expectedVersion/project (B-0 cite).

import (
	"context"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
)

// TestRestart_DurableVersionSurvivesProcessRestart drives a 3-step linear
// workflow (a→b→c) through ONE tick on a first Engine (step a activates,
// dispatches, and completes), then constructs a brand-new Engine over the SAME
// storage — a process restart, whose e.seq/e.ver/etc maps all start empty —
// and drives it to terminal. Every tick (on either engine) must succeed, and
// the instance must reach `completed`.
func TestRestart_DurableVersionSurvivesProcessRestart(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.restart", Version: "1.0.0", Namespace: "t", Name: "restart",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "ha"), nativeStep("b", "hb"), nativeStep("c", "hc"),
		},
		Transitions: []core.Transition{{From: "a", To: "b"}, {From: "b", To: "c"}},
		InitialStep: "a", FinalSteps: []string{"c"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", outputs: map[string]any{"ra": "a-done"}}
	hb := &stepHandler{id: "hb", outputs: map[string]any{"rb": "b-done"}}
	hc := &stepHandler{id: "hc", outputs: map[string]any{"rc": "c-done"}}

	// Engine 1 ("before restart"): Submit + one Tick. Step a activates,
	// dispatches, and completes; b is now activatable but not yet claimed.
	e1, s, _ := buildEngine(t, def, 1, ha, hb, hc)
	ctx := context.Background()
	iid, err := e1.Submit(ctx, "t.restart", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := e1.Tick(ctx); err != nil {
		t.Fatalf("Tick 1 (engine1, pre-restart): %v", err)
	}
	inst, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance after tick 1: %v", err)
	}
	if _, ok := inst.Variables["a"]; !ok {
		t.Fatalf("step a did not complete on engine1's first tick: variables=%v", inst.Variables)
	}
	if inst.Status != core.InstanceStatusRunning {
		t.Fatalf("status after tick 1 = %q, want running (workflow not yet terminal)", inst.Status)
	}

	// "Restart": a brand-new Engine over the SAME storage. Its e.seq/e.ver/
	// e.retries/e.pending/e.waits maps are all empty — exactly the in-memory
	// state a real process restart produces (see the card's B-0 background).
	nr2 := native.New()
	nr2.Register(ha)
	nr2.Register(hb)
	nr2.Register(hc)
	clk2 := newFakeClock(engineStart, 10*time.Millisecond)
	e2 := New(s, map[core.StepType]Runner{core.StepTypeNative: nr2},
		Config{MaxParallelSteps: 1, Clock: clk2.now}, discardLogger())

	// runToTerminal fails the test on ANY tick error — this is the "no tick
	// returned an error" assertion the card requires.
	final := runToTerminal(t, e2, s, iid)
	if final.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", final.Status)
	}

	pairs, _ := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepCompleted, "a"},
		{core.EventTypeStepStarted, "b"}, {core.EventTypeStepCompleted, "b"},
		{core.EventTypeStepStarted, "c"}, {core.EventTypeStepCompleted, "c"},
		{core.EventTypeWorkflowCompleted, ""},
	}
	assertPairs(t, pairs, want)

	assertProjectionEquivalence(t, s, iid)
}

// TestRestart_InFlightStepCrashRecovery_WithRetry verifies that an in-flight step
// that was running when a process crashed is recovered upon restart and retried (RC-1).
func TestRestart_InFlightStepCrashRecovery_WithRetry(t *testing.T) {
	retryPolicy := &core.RetryPolicy{
		Attempts: 2,
		Backoff:  "immediate",
	}
	stepDef := nativeStep("a", "ha")
	stepDef.Retry = retryPolicy

	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.rc1-retry", Version: "1.0.0", Namespace: "t", Name: "rc1-retry",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			stepDef,
		},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}

	startedCh := make(chan struct{})
	blockCh := make(chan struct{})
	defer close(blockCh)

	ha1 := &invocBlockingHandler{id: "ha", startedCh: startedCh, blockCh: blockCh}
	e1, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now, ha1)
	ctx := context.Background()
	iid, err := e1.Submit(ctx, "t.rc1-retry", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	go func() { _ = e1.Tick(ctx) }()

	select {
	case <-startedCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for step a to start")
	}

	// At this point, step a is in-flight (claimed, StepStarted emitted).
	// Simulate process crash: discard e1.
	// Boot fresh engine e2 over the SAME storage with a handler that succeeds.
	ha2 := &stepHandler{id: "ha", outputs: map[string]any{"recovered": true}}
	nr2 := native.New()
	nr2.Register(ha2)
	clk2 := newManualClock(engineStart.Add(time.Second))
	e2 := New(s, map[core.StepType]Runner{core.StepTypeNative: nr2},
		Config{MaxParallelSteps: 1, Clock: clk2.now}, discardLogger())

	// Tick e2 to recover and re-dispatch.
	for i := 0; i < 5; i++ {
		if err := e2.Tick(ctx); err != nil {
			t.Fatalf("e2.Tick(%d): %v", i, err)
		}
	}

	instAfter, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if instAfter.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed (current_steps=%v)", instAfter.Status, instAfter.CurrentSteps)
	}
}

// TestRestart_InFlightStepCrashRecovery_NoRetry verifies that an in-flight step
// without retry that crashed transitions to terminal failure without hanging (RC-1).
func TestRestart_InFlightStepCrashRecovery_NoRetry(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.rc1-noretry", Version: "1.0.0", Namespace: "t", Name: "rc1-noretry",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "ha"),
		},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}

	startedCh := make(chan struct{})
	blockCh := make(chan struct{})
	defer close(blockCh)

	ha1 := &invocBlockingHandler{id: "ha", startedCh: startedCh, blockCh: blockCh}
	e1, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now, ha1)
	ctx := context.Background()
	iid, err := e1.Submit(ctx, "t.rc1-noretry", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	go func() { _ = e1.Tick(ctx) }()

	select {
	case <-startedCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for step a to start")
	}

	// Simulate crash and restart
	ha2 := &stepHandler{id: "ha", outputs: map[string]any{"recovered": true}}
	nr2 := native.New()
	nr2.Register(ha2)
	clk2 := newManualClock(engineStart.Add(time.Second))
	e2 := New(s, map[core.StepType]Runner{core.StepTypeNative: nr2},
		Config{MaxParallelSteps: 1, Clock: clk2.now}, discardLogger())

	// Tick e2
	if err := e2.Tick(ctx); err != nil {
		t.Fatalf("e2.Tick: %v", err)
	}

	instAfter, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if instAfter.Status != core.InstanceStatusFailed {
		t.Fatalf("status = %q, want failed (current_steps=%v)", instAfter.Status, instAfter.CurrentSteps)
	}
	if len(instAfter.CurrentSteps) != 0 {
		t.Fatalf("current_steps not empty after failure: %v", instAfter.CurrentSteps)
	}
}

type invocBlockingHandler struct {
	id        string
	startedCh chan struct{}
	blockCh   chan struct{}
}

func (h *invocBlockingHandler) ID() string { return h.id }
func (h *invocBlockingHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	close(h.startedCh)
	<-h.blockCh
	return core.StepResult{Outputs: map[string]any{"ok": true}}, nil
}
