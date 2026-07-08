package engine

// Signal timeout scan fixtures (M07-C3r). One fixture per timeout_action (fail,
// compensate, continue). Each drives the minimal path through
// SIGNAL_TIMEOUT_SCAN (signalTimeoutScan → processTimeout) using a real Engine
// + real SQLiteStorage with a manualClock that is advanced past the timeout_at
// boundary. Every fixture asserts the EXACT ordered event stream (closed TDS-01
// vocabulary only) and assertProjectionEquivalence at the terminal / resumed
// state where the forward and rebuild paths converge.
//
// EDR-007 §9 gap: the transient `waiting` status is non-evented; forward ≡
// rebuild is only asserted after the timeout fires (where the gap closes).

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// signalStepTimeout builds a signal step with an explicit timeout and action.
func signalStepTimeout(id, signalName, timeout, action string) core.Step {
	return core.Step{
		ID: id, Name: id, Type: core.StepTypeSignal,
		WaitSignal: &core.WaitConfig{
			SignalName:    signalName,
			Timeout:       core.Duration(timeout),
			TimeoutAction: action,
		},
	}
}

// newSignalEngine builds an Engine that has both a NativeRunner (for any native
// steps) and a signal-capable storage, using the given clock. It is
// newNativeEngine-equivalent but named explicitly for clarity in signal tests.
func newSignalEngine(t *testing.T, def core.WorkflowDefinition, clk func() time.Time, handlers ...core.StepHandler) (*Engine, *storage.SQLiteStorage) {
	t.Helper()
	s := openStorage(t)
	if err := s.RegisterWorkflow(context.Background(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	nr := native.New()
	for _, h := range handlers {
		nr.Register(h)
	}
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: 1, Clock: clk}, discardLogger())
	return e, s
}

// ── timeout_action = "fail" ───────────────────────────────────────────────────

// TestSignalTimeout_Fail proves that a signal step whose wait_record timeout_at
// has elapsed and whose timeout_action is "fail" routes through routeTerminalFailure:
// StepFailed + WorkflowFailed (no fallback, no compensation plan).
func TestSignalTimeout_Fail(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.stfail", Version: "1.0.0", Namespace: "t", Name: "stfail",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{signalStepTimeout("w", "go", "5s", "fail")},
		InitialStep: "w", FinalSteps: []string{"w"}, Metadata: map[string]any{},
	}
	clk := newManualClock(engineStart)
	e, s := newSignalEngine(t, def, clk.now)
	ctx := context.Background()

	iid, err := e.Submit(ctx, "t.stfail", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick 1: signal step activates and parks the instance in `waiting`; a
	// wait_record with timeout_at = now+5s is inserted (OUTPUT 0b).
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 1: %v", err)
	}
	inst, _ := s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusWaiting {
		t.Fatalf("after WAIT entry status = %q, want waiting", inst.Status)
	}

	// Verify the wait_record has a non-nil timeout_at (OUTPUT 0b).
	wr, found, err := s.GetWaitRecord(ctx, iid, "w")
	if err != nil {
		t.Fatalf("GetWaitRecord: %v", err)
	}
	if !found {
		t.Fatalf("wait_record not found after WAIT entry")
	}
	if wr.TimeoutAt == nil {
		t.Fatalf("wait_record.timeout_at must be non-nil when Timeout is set (OUTPUT 0b)")
	}

	// Advance clock past the 5s timeout so ListDueWaitRecords returns this record.
	clk.advance(6 * second)

	// Tick 2: SIGNAL_TIMEOUT_SCAN fires → processTimeout → timeout_action=fail →
	// routeTerminalFailure (no fallback, no on_error, no compensation) →
	// StepFailed + WorkflowFailed.
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 2 (timeout): %v", err)
	}

	inst, _ = s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusFailed {
		t.Fatalf("after timeout-fail status = %q, want failed", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "w"},
		{core.EventTypeStepFailed, "w"},   // timeout signal_timeout code
		{core.EventTypeWorkflowFailed, ""},
	})

	// StepFailed.error.code = "signal_timeout" (VERBATIM: processTimeout stepErr).
	sf := findEvent(t, evs, core.EventTypeStepFailed, "w")
	var sfp struct {
		Retrying bool           `json:"retrying"`
		Error    core.StepError `json:"error"`
	}
	decodePayload(t, sf, &sfp)
	if sfp.Error.Code != "signal_timeout" {
		t.Fatalf("StepFailed.error.code = %q, want signal_timeout", sfp.Error.Code)
	}
	if sfp.Retrying {
		t.Fatalf("StepFailed.retrying = true, want false")
	}

	// wait_record must be deleted (processTimeout deletes it before routing).
	_, wrFound, err := s.GetWaitRecord(ctx, iid, "w")
	if err != nil {
		t.Fatalf("GetWaitRecord after timeout: %v", err)
	}
	if wrFound {
		t.Fatalf("wait_record must be deleted after timeout processing")
	}

	// forward ≡ rebuild (terminal state: the waiting gap closes before WorkflowFailed).
	assertProjectionEquivalence(t, s, iid)
}

// ── timeout_action = "compensate" ────────────────────────────────────────────

// TestSignalTimeout_Compensate proves that timeout_action="compensate" bypasses
// fallback/on_error routing and forces StepFailed + WorkflowFailed + compensation.
// A native step "a" runs first so the completed-set is non-empty, triggering
// runCompensation. Event sequence: WorkflowStarted, StepStarted("a"),
// StepCompleted("a"), StepStarted("w"), StepFailed("w"), WorkflowFailed,
// WorkflowCompensating, WorkflowCompensated.
func TestSignalTimeout_Compensate(t *testing.T) {
	undoOrder := []string{}
	undoMu := &sync.Mutex{}

	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.stcomp", Version: "1.0.0", Namespace: "t", Name: "stcomp",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "ha"),
			signalStepTimeout("w", "go", "5s", "compensate"),
		},
		Transitions: []core.Transition{{From: "a", To: "w"}},
		Compensation: &core.CompensationPlan{Steps: []core.CompensationStep{
			{StepID: "a", UndoHandler: "undo-a"},
		}},
		InitialStep: "a", FinalSteps: []string{"w"}, Metadata: map[string]any{},
	}
	clk := newManualClock(engineStart)
	ha := &stepHandler{id: "ha", outputs: map[string]any{"ra": "ok"}}
	undoA := &recordingHandler{id: "undo-a", order: &undoOrder, mu: undoMu}
	e, s := newSignalEngine(t, def, clk.now, ha, undoA)
	ctx := context.Background()

	iid, err := e.Submit(ctx, "t.stcomp", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Run until "w" is parked (a completes, then w parks in waiting).
	waiting := false
	for i := 0; i < 10 && !waiting; i++ {
		if err := e.Tick(ctx); err != nil {
			t.Fatalf("Tick %d: %v", i, err)
		}
		inst, _ := s.GetInstance(ctx, iid)
		waiting = inst.Status == core.InstanceStatusWaiting
	}
	if !waiting {
		t.Fatalf("instance never reached waiting")
	}

	// After parking: WorkflowStarted, StepStarted("a"), StepCompleted("a"), StepStarted("w").
	pairs, _ := eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepCompleted, "a"},
		{core.EventTypeStepStarted, "w"},
	})

	// Advance clock past the 5s timeout.
	clk.advance(6 * second)

	// Timeout tick: timeout_action=compensate → StepFailed + WorkflowFailed + compensation.
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompensated {
		t.Fatalf("after timeout-compensate status = %q, want compensated", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepCompleted, "a"},
		{core.EventTypeStepStarted, "w"},
		{core.EventTypeStepFailed, "w"},   // code="signal_timeout"
		{core.EventTypeWorkflowFailed, ""},
		{core.EventTypeWorkflowCompensating, ""},
		{core.EventTypeWorkflowCompensated, ""},
	})

	// StepFailed.error.code = "signal_timeout".
	sf := findEvent(t, evs, core.EventTypeStepFailed, "w")
	var sfp struct {
		Error core.StepError `json:"error"`
	}
	decodePayload(t, sf, &sfp)
	if sfp.Error.Code != "signal_timeout" {
		t.Fatalf("StepFailed.error.code = %q, want signal_timeout", sfp.Error.Code)
	}

	// WorkflowCompensating.from_step = "w" (the timed-out step).
	wcg := findEvent(t, evs, core.EventTypeWorkflowCompensating, "")
	var wcgp workflowCompensatingPayload
	decodePayload(t, wcg, &wcgp)
	if wcgp.FromStep != "w" {
		t.Fatalf("WorkflowCompensating.from_step = %q, want w", wcgp.FromStep)
	}

	// undo-a ran (only completed steps are undone).
	undoMu.Lock()
	got := append([]string{}, undoOrder...)
	undoMu.Unlock()
	if len(got) != 1 || got[0] != "undo-a" {
		t.Fatalf("undo order = %v, want [undo-a]", got)
	}

	// forward ≡ rebuild at terminal state.
	assertProjectionEquivalence(t, s, iid)
}

// ── timeout_action = "continue" ───────────────────────────────────────────────

// TestSignalTimeout_Continue proves that timeout_action="continue" emits
// StepCompleted with empty outputs so the workflow advances through normal
// transition evaluation. The signal step "w" connects to native step "b" via a
// forward transition; after the timeout, "b" activates and the workflow
// completes. The test also verifies that timeout_at with "continue" DOES NOT
// cause a stall (the instance is written back to running before StepCompleted
// so SCAN_RUNNABLE picks it up on the next tick).
func TestSignalTimeout_Continue(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.stcont", Version: "1.0.0", Namespace: "t", Name: "stcont",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			signalStepTimeout("w", "go", "5s", "continue"),
			nativeStep("b", "hb"),
		},
		Transitions: []core.Transition{{From: "w", To: "b"}},
		InitialStep: "w", FinalSteps: []string{"b"}, Metadata: map[string]any{},
	}
	clk := newManualClock(engineStart)
	hb := &stepHandler{id: "hb", outputs: map[string]any{"rb": "ok"}}
	e, s := newSignalEngine(t, def, clk.now, hb)
	ctx := context.Background()

	iid, err := e.Submit(ctx, "t.stcont", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick 1: signal step parks.
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 1: %v", err)
	}
	inst, _ := s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusWaiting {
		t.Fatalf("after WAIT entry status = %q, want waiting", inst.Status)
	}

	// Advance past the 5s timeout.
	clk.advance(6 * second)

	// Tick 2: timeout fires → timeout_action=continue → StepCompleted("w", {}).
	// Instance is set back to running (non-evented) so SCAN_RUNNABLE can pick it
	// up on the next tick.
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 2 (timeout-continue): %v", err)
	}

	inst, _ = s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusRunning {
		t.Fatalf("after timeout-continue status = %q, want running", inst.Status)
	}
	if len(inst.CurrentSteps) != 0 {
		t.Fatalf("after timeout-continue current_steps = %v, want [] (w removed by StepCompleted)", inst.CurrentSteps)
	}

	// Event stream after Tick 2: WorkflowStarted, StepStarted("w"), StepCompleted("w").
	// (assertProjectionEquivalence is only called at the terminal state below;
	// calling RebuildState here would reset the DB version to 1 while e.ver stays
	// at 6, creating a gap that the single-retry in project cannot bridge and
	// causing a false conflict on the next claim.)
	pairs, evs := eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "w"},
		{core.EventTypeStepCompleted, "w"}, // empty outputs (timeout, no signal payload)
	})

	// StepCompleted("w").outputs = {} (no signal payload — timed out).
	sc := findEvent(t, evs, core.EventTypeStepCompleted, "w")
	var scp struct {
		Outputs map[string]any `json:"outputs"`
	}
	decodePayload(t, sc, &scp)
	if len(scp.Outputs) != 0 {
		t.Fatalf("StepCompleted outputs = %v, want {} (timeout-continue has no signal payload)", scp.Outputs)
	}

	// Run to completion: "b" activates (transition w→b fires since variables["w"]
	// is set) and completes. WorkflowCompleted follows.
	inst = runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("final status = %q, want completed", inst.Status)
	}

	pairs, _ = eventPairs(t, s, iid)
	assertPairs(t, pairs, []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "w"}, {core.EventTypeStepCompleted, "w"},
		{core.EventTypeStepStarted, "b"}, {core.EventTypeStepCompleted, "b"},
		{core.EventTypeWorkflowCompleted, ""},
	})

	// forward ≡ rebuild at terminal state (the waiting gap is fully resolved;
	// forward and rebuild both produce completed with w+b in variables).
	assertProjectionEquivalence(t, s, iid)
}

// second is a named duration constant used by the clock advance calls.
const second = time.Second
