package engine

// la_review_fix_test.go — LA-C4a: independent-review findings in hydrate.go.
//
//   - F-T (terminal fence, P2 / finding #1, and the terminal case of #6): once a
//     workflow-terminal event is durable nothing may be appended after it, even when its
//     projection write failed. A re-projection failure is returned, never swallowed, and
//     never falls through to RC-1.
//   - F-P (settled-step catch-up, P6 / finding #6): a durable StepCompleted (or
//     StepFailed{retrying:false}) whose projection write was lost is re-applied.
//   - F-C (cancellation-aware RC-1, P3 / P7 / findings #4, #9; Finalization B4 step 3):
//     with cancellation requested RC-1 never re-enters a wait and never routes.
//   - P4 (F3a lower bound): a SignalReceived BEFORE a step's latest StepStarted does not
//     complete that step.
//   - Guards (a) cancellationRequested, (b) code "cancelled", and the durable
//     WorkflowFailed guard in completeCommittedRoutes each have a test that fails when that
//     single guard is removed (verified by mutation; see the LA-C4a report).
//
// Every scenario runs same-process and, where meaningful, _AfterRestart (a fresh engine
// over the same store, clock and id source: a process restart).

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// laC4aEnv bundles one scenario: store, fault layer, clock, id source and the engine.
// restart() replaces the engine with a fresh one over the same store (a restart).
type laC4aEnv struct {
	s        *storage.SQLiteStorage
	fs       *laFaultStorage
	clk      *fakeClock
	ids      func() string
	maxPar   int
	handlers []core.StepHandler
	e        *Engine
	iid      core.InstanceID
}

func laC4aNew(t *testing.T, def core.WorkflowDefinition, maxPar int, handlers ...core.StepHandler) *laC4aEnv {
	t.Helper()
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	return &laC4aEnv{
		s: s, fs: fs, clk: clk, ids: ids, maxPar: maxPar, handlers: handlers,
		e: laEngine(fs, clk, ids, maxPar, handlers...),
	}
}

func (v *laC4aEnv) restart() {
	v.e = laEngine(v.fs, v.clk, v.ids, v.maxPar, v.handlers...)
}

// laC4aAppendRaw durably appends one event through the engine's sequence path WITHOUT
// projecting it: the projection write is deliberately absent, as if it had failed.
func laC4aAppendRaw(t *testing.T, e *Engine, inst core.WorkflowInstance, et core.EventType, stepID string, payload any) {
	t.Helper()
	raw, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", et, err)
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  et,
		StepID:     stepID,
		Payload:    raw,
		EmittedAt:  e.now(),
	}
	if err := e.appendEvent(context.Background(), &ev); err != nil {
		t.Fatalf("append %s: %v", et, err)
	}
}

// laC4aHydrate runs hydrate directly on e for iid (the processInstance call, in isolation).
func laC4aHydrate(t *testing.T, s *storage.SQLiteStorage, e *Engine, iid core.InstanceID) {
	t.Helper()
	ctx := context.Background()
	inst := laInstance(t, s, iid)
	dv, err := e.defViewFor(ctx, inst)
	if err != nil {
		t.Fatalf("defViewFor: %v", err)
	}
	if err := e.hydrate(ctx, dv, inst); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
}

// laC4aTerminalFence asserts the EventLog's terminal fence: exactly one et, and it is the
// LAST event (nothing appended after the terminal event; P2 / finding #1).
func laC4aTerminalFence(t *testing.T, label string, evs []core.ExecutionEvent, et core.EventType) {
	t.Helper()
	if n := laCount(evs, et); n != 1 {
		t.Errorf("%s: %s events = %d, want 1; log: %s", label, et, n, laSummary(t, evs))
	}
	if len(evs) == 0 || evs[len(evs)-1].EventType != et {
		t.Errorf("%s: last event is not %s (an event after the terminal event is permanent corruption); log: %s",
			label, et, laSummary(t, evs))
	}
}

// laC4aStepFailedCodes returns the error codes of every StepFailed{retrying:false} for stepID.
func laC4aStepFailedCodes(t *testing.T, evs []core.ExecutionEvent, stepID string) []string {
	t.Helper()
	var codes []string
	for _, ev := range evs {
		if ev.EventType != core.EventTypeStepFailed || ev.StepID != stepID {
			continue
		}
		var p stepFailedPayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			t.Fatalf("decode StepFailed %s: %v", stepID, err)
		}
		if !p.Retrying {
			codes = append(codes, p.Error.Code)
		}
	}
	return codes
}

func laC4aHasStep(steps []string, stepID string) bool {
	for _, s := range steps {
		if s == stepID {
			return true
		}
	}
	return false
}

// laC4aNoFallback asserts that no StepFallbackActivated was appended (B4 step 3 at the
// event level: no new activation after cancellation_requested).
func laC4aNoFallback(t *testing.T, label string, evs []core.ExecutionEvent) {
	t.Helper()
	if laHasEvent(evs, core.EventTypeStepFallbackActivated) {
		t.Errorf("%s: StepFallbackActivated appended (B4 step 3: no new activation); log: %s", label, laSummary(t, evs))
	}
}

// ── F-T: terminal fence ─────────────────────────────────────────────────────────────

// P2: WorkflowFailed is durable; its projection write (UpsertInstance status=failed) fails.
// Sibling b2 is still StepStarted. Expect exactly one WorkflowFailed, nothing after it,
// and a terminal projection.
func TestLA_C4a_P2_WorkflowFailedProjectionFailure_SingleTerminal(t *testing.T) {
	laC4aP2(t, false)
}

func TestLA_C4a_P2_WorkflowFailedProjectionFailure_SingleTerminal_AfterRestart(t *testing.T) {
	laC4aP2(t, true)
}

func laC4aP2(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a P2 WorkflowFailed projection", restart)
	def := laDefFanOut()
	root := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1 := &stepHandler{id: "la_b1", err: errLAHandler}
	b2 := &stepHandler{id: "la_b2", outputs: map[string]any{"who": "b2"}}
	v := laC4aNew(t, def, 2, root, b1, b2)
	v.iid = laSubmit(t, v.e, def.ID)
	v.fs.arm(laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusFailed})
	laWantInjected(t, laTickUntilTripped(t, v.e, v.fs))
	if restart {
		v.restart()
	}
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	evs := laEvents(t, v.s, v.iid)
	inst := laInstance(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d status=%s", label, n, len(errs), inst.Status)
	if len(errs) != 0 {
		t.Errorf("%s: drain ticks returned errors: %v", label, errs)
	}
	if inst.Status != core.InstanceStatusFailed {
		t.Errorf("%s: status = %s, want failed", label, inst.Status)
	}
	laC4aTerminalFence(t, label, evs, core.EventTypeWorkflowFailed)
	if got := laC4aStepFailedCodes(t, evs, "b2"); len(got) != 0 {
		t.Errorf("%s: StepFailed(b2) appended after the terminal event (codes %v)", label, got)
	}
}

// Re-projection itself fails: the fence returns its error (LA-C1 reset retries it next
// tick) and never falls through to RC-1, which would append a second terminal event.
func TestLA_C4a_P2_FenceReprojectionFails_ErrorNotRC1(t *testing.T) {
	label := laLabel("LA-C4a fence re-projection failure", false)
	def := laDefFanOut()
	root := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1 := &stepHandler{id: "la_b1", err: errLAHandler}
	b2 := &stepHandler{id: "la_b2", outputs: map[string]any{"who": "b2"}}
	v := laC4aNew(t, def, 2, root, b1, b2)
	v.iid = laSubmit(t, v.e, def.ID)
	v.fs.arm(laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusFailed})
	laWantInjected(t, laTickUntilTripped(t, v.e, v.fs))
	// The next failed-status projection is the fence's re-projection: fail it too.
	v.fs.arm(laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusFailed})
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	laWantInjected(t, errs)
	evs := laEvents(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d injected_trips=%d", label, n, len(errs), v.fs.trippedCount())
	if inst := laInstance(t, v.s, v.iid); inst.Status != core.InstanceStatusFailed {
		t.Errorf("%s: status = %s, want failed", label, inst.Status)
	}
	laC4aTerminalFence(t, label, evs, core.EventTypeWorkflowFailed)
}

// Terminal case of #6: WorkflowCompleted is durable; its projection write fails.
func TestLA_C4a_FenceWorkflowCompleted_ProjectionFailure(t *testing.T) {
	laC4aFenceCompleted(t, false)
}

func TestLA_C4a_FenceWorkflowCompleted_ProjectionFailure_AfterRestart(t *testing.T) {
	laC4aFenceCompleted(t, true)
}

func laC4aFenceCompleted(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a WorkflowCompleted projection", restart)
	def := laDefSingle()
	h := &stepHandler{id: "la_a", outputs: map[string]any{"msg": "done"}}
	v := laC4aNew(t, def, 1, h)
	v.iid = laSubmit(t, v.e, def.ID)
	v.fs.arm(laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusCompleted})
	laWantInjected(t, laTickUntilTripped(t, v.e, v.fs))
	if restart {
		v.restart()
	}
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	evs := laEvents(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d status=%s", label, n, len(errs), laInstance(t, v.s, v.iid).Status)
	if len(errs) != 0 {
		t.Errorf("%s: drain ticks returned errors: %v", label, errs)
	}
	if inst := laInstance(t, v.s, v.iid); inst.Status != core.InstanceStatusCompleted {
		t.Errorf("%s: status = %s, want completed", label, inst.Status)
	}
	laC4aTerminalFence(t, label, evs, core.EventTypeWorkflowCompleted)
}

// Terminal case: WorkflowCancelled is durable; its projection write fails.
func TestLA_C4a_FenceWorkflowCancelled_ProjectionFailure(t *testing.T) {
	laC4aFenceCancelled(t, false)
}

func TestLA_C4a_FenceWorkflowCancelled_ProjectionFailure_AfterRestart(t *testing.T) {
	laC4aFenceCancelled(t, true)
}

func laC4aFenceCancelled(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a WorkflowCancelled projection", restart)
	def := laDefSingle()
	h := &stepHandler{id: "la_a", outputs: map[string]any{"msg": "done"}}
	v := laC4aNew(t, def, 1, h)
	v.iid = laSubmit(t, v.e, def.ID)
	if err := v.e.Cancel(context.Background(), v.iid, "la-c4a terminal", false); err != nil {
		t.Fatalf("%s: Cancel: %v", label, err)
	}
	v.fs.arm(laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusCancelled})
	laWantInjected(t, laTickUntilTripped(t, v.e, v.fs))
	if restart {
		v.restart()
	}
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	evs := laEvents(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d", label, n, len(errs))
	if len(errs) != 0 {
		t.Errorf("%s: drain ticks returned errors: %v", label, errs)
	}
	if inst := laInstance(t, v.s, v.iid); inst.Status != core.InstanceStatusCancelled {
		t.Errorf("%s: status = %s, want cancelled", label, inst.Status)
	}
	laC4aTerminalFence(t, label, evs, core.EventTypeWorkflowCancelled)
	if laCalls(h) != 0 {
		t.Errorf("%s: step a executed %d times on a cancel-requested instance, want 0", label, laCalls(h))
	}
}

// ── F-P: settled-step projection catch-up ───────────────────────────────────────────

// P6: StepCompleted(a) is durable; its projection write fails once. Expect the
// single-step workflow to complete without rebuild-state.
func TestLA_C4a_P6_StepCompletedProjectionFailure_Completes(t *testing.T) {
	laC4aP6(t, false)
}

func TestLA_C4a_P6_StepCompletedProjectionFailure_Completes_AfterRestart(t *testing.T) {
	laC4aP6(t, true)
}

func laC4aP6(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a P6 StepCompleted projection", restart)
	def := laDefSingle()
	h := &stepHandler{id: "la_a", outputs: map[string]any{"msg": "done"}}
	v := laC4aNew(t, def, 1, h)
	v.iid = laSubmit(t, v.e, def.ID)
	// The 2nd UpsertInstance is StepCompleted(a)'s projection (the StepStarted(a) write
	// is the 1st); the precondition below proves the fault landed there.
	v.fs.arm(laFault{op: laOpUpsertInstance, nth: 2})
	laWantInjected(t, laTickUntilTripped(t, v.e, v.fs))
	pre := laInstance(t, v.s, v.iid)
	if !laHas(laEvents(t, v.s, v.iid), core.EventTypeStepCompleted, "a") || !laC4aHasStep(pre.CurrentSteps, "a") {
		t.Fatalf("%s: precondition: want durable StepCompleted(a) with projected current_steps=[a]; log: %s, current=%v",
			label, laSummary(t, laEvents(t, v.s, v.iid)), pre.CurrentSteps)
	}
	if restart {
		v.restart()
	}
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	evs := laEvents(t, v.s, v.iid)
	inst := laInstance(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d status=%s current_steps=%v", label, n, len(errs), inst.Status, inst.CurrentSteps)
	if len(errs) != 0 {
		t.Errorf("%s: drain ticks returned errors: %v", label, errs)
	}
	if inst.Status != core.InstanceStatusCompleted {
		t.Errorf("%s: status = %s, want completed (StepCompleted is durable)", label, inst.Status)
	}
	if n := laCount(evs, core.EventTypeStepCompleted); n != 1 {
		t.Errorf("%s: StepCompleted events = %d, want 1", label, n)
	}
	laC4aTerminalFence(t, label, evs, core.EventTypeWorkflowCompleted)
}

// StepFailed{retrying:false} is durable but its projection write was lost. The step still
// sits in current_steps; the committed route (on_error) must not wedge on it. Expect the
// on_error target to run and the workflow to complete, with the failure recorded once.
func TestLA_C4a_FStepFailedProjectionLost_CatchUpCompletes(t *testing.T) {
	laC4aFStepFailed(t, false)
}

func TestLA_C4a_FStepFailedProjectionLost_CatchUpCompletes_AfterRestart(t *testing.T) {
	laC4aFStepFailed(t, true)
}

func laC4aFStepFailed(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a F-P StepFailed projection lost", restart)
	def := laDefOnErrorCatchUp()
	ha := &stepHandler{id: "la_a", outputs: map[string]any{}}
	heh := &stepHandler{id: "la_eh", outputs: map[string]any{}}
	v := laC4aNew(t, def, 1, ha, heh)
	v.iid = laSubmit(t, v.e, def.ID)
	inst := laInstance(t, v.s, v.iid)
	if _, err := v.e.emitStepStarted(context.Background(), inst, "a", 1, map[string]any{}); err != nil {
		t.Fatalf("%s: emitStepStarted: %v", label, err)
	}
	laC4aAppendRaw(t, v.e, inst, core.EventTypeStepFailed, "a", stepFailedPayload{
		StepID: "a", Attempt: 1, Error: core.StepError{Code: "la_c4a_onerror", Message: "probe"}, Retrying: false,
	})
	if restart {
		v.restart()
	}
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	evs := laEvents(t, v.s, v.iid)
	final := laInstance(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d status=%s", label, n, len(errs), final.Status)
	if len(errs) != 0 {
		t.Errorf("%s: drain ticks returned errors: %v", label, errs)
	}
	if final.Status != core.InstanceStatusCompleted {
		t.Errorf("%s: status = %s, want completed (on_error target must run)", label, final.Status)
	}
	if got := laC4aStepFailedCodes(t, evs, "a"); len(got) != 1 || got[0] != "la_c4a_onerror" {
		t.Errorf("%s: StepFailed(a) codes = %v, want exactly [la_c4a_onerror]", label, got)
	}
	if laCalls(ha) != 0 {
		t.Errorf("%s: step a re-dispatched %d times after its terminal failure, want 0", label, laCalls(ha))
	}
	if !laHas(evs, core.EventTypeStepCompleted, "eh") {
		t.Errorf("%s: on_error target eh never completed; log: %s", label, laSummary(t, evs))
	}
}

// laDefOnErrorCatchUp: a (no fallback) with an on_error transition to eh; eh is final.
func laDefOnErrorCatchUp() core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.c4a.onerror", Version: "1.0.0", Namespace: "la", Name: "onerror",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("a", "la_a"), nativeStep("eh", "la_eh")},
		Transitions: []core.Transition{{From: "a", To: "eh", OnError: true}},
		InitialStep: "a", FinalSteps: []string{"eh"}, Metadata: map[string]any{},
	}
}

// ── F-C: cancellation-aware RC-1 ────────────────────────────────────────────────────

// P3: a signal step lost its wait_record (CreateWaitRecord fails after StepStarted), then the
// operator cancels the RUNNING instance. Expect the cancel to finalize, not park in waiting.
func TestLA_C4a_P3_CancelAfterLostWaitRecord_Finalizes(t *testing.T) {
	laC4aP3(t, false)
}

func TestLA_C4a_P3_CancelAfterLostWaitRecord_Finalizes_AfterRestart(t *testing.T) {
	laC4aP3(t, true)
}

func laC4aP3(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a P3 cancel after lost wait_record", restart)
	def := laDefWait()
	a := &stepHandler{id: "la_a", outputs: map[string]any{}}
	z := &stepHandler{id: "la_z", outputs: map[string]any{}}
	v := laC4aNew(t, def, 1, a, z)
	v.iid = laSubmit(t, v.e, def.ID)
	v.fs.arm(laFault{op: laOpCreateWaitRecord, nth: 1, stepID: "w"})
	laWantInjected(t, laTickUntilTripped(t, v.e, v.fs))
	if err := v.e.Cancel(context.Background(), v.iid, "operator", false); err != nil {
		t.Fatalf("%s: Cancel: %v", label, err)
	}
	if restart {
		v.restart()
	}
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	evs := laEvents(t, v.s, v.iid)
	inst := laInstance(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d status=%s current_steps=%v", label, n, len(errs), inst.Status, inst.CurrentSteps)
	if len(errs) != 0 {
		t.Errorf("%s: drain ticks returned errors: %v", label, errs)
	}
	if inst.Status != core.InstanceStatusCancelled {
		t.Errorf("%s: status = %s, want cancelled (never parked in waiting)", label, inst.Status)
	}
	if got := laC4aStepFailedCodes(t, evs, "w"); len(got) != 1 || got[0] != "cancelled" {
		t.Errorf("%s: StepFailed(w) codes = %v, want exactly [cancelled]", label, got)
	}
	if laHas(evs, core.EventTypeStepStarted, "z") || laCalls(z) != 0 {
		t.Errorf("%s: step z activated on a cancel-requested instance", label)
	}
	laC4aTerminalFence(t, label, evs, core.EventTypeWorkflowCancelled)
}

// P7: a fallback step's StepFailed append fails (its StepStarted is stranded); the instance is
// then cancelled and RC-1 runs. Expect no StepFallbackActivated and a cancelled instance.
func TestLA_C4a_P7_CancelRequestedStranded_NoRouting(t *testing.T) {
	laC4aP7(t, false)
}

func TestLA_C4a_P7_CancelRequestedStranded_NoRouting_AfterRestart(t *testing.T) {
	laC4aP7(t, true)
}

func laC4aP7(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a P7 cancel-requested stranded fallback", restart)
	def := laDefRoute("la.c4a.b4fb", nil, true)
	ha, hfb := laFallbackHandlers()
	v := laC4aNew(t, def, 1, ha, hfb)
	v.iid = laSubmit(t, v.e, def.ID)
	v.fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepFailed, stepID: "a"})
	laWantInjected(t, laTickUntilTripped(t, v.e, v.fs))
	if err := v.e.Cancel(context.Background(), v.iid, "operator", false); err != nil {
		t.Fatalf("%s: Cancel: %v", label, err)
	}
	if restart {
		v.restart()
	}
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	evs := laEvents(t, v.s, v.iid)
	inst := laInstance(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d status=%s", label, n, len(errs), inst.Status)
	if len(errs) != 0 {
		t.Errorf("%s: drain ticks returned errors: %v", label, errs)
	}
	if inst.Status != core.InstanceStatusCancelled {
		t.Errorf("%s: status = %s, want cancelled", label, inst.Status)
	}
	laC4aNoFallback(t, label, evs)
	if laHasEvent(evs, core.EventTypeWorkflowFailed) {
		t.Errorf("%s: WorkflowFailed appended on a cancel-requested instance", label)
	}
	if got := laC4aStepFailedCodes(t, evs, "a"); len(got) != 1 || got[0] != "worker_crash" {
		t.Errorf("%s: StepFailed(a) codes = %v, want exactly [worker_crash] (F-C, no routing)", label, got)
	}
	if laCalls(hfb) != 0 {
		t.Errorf("%s: fallback fb executed %d times on a cancel-requested instance, want 0", label, laCalls(hfb))
	}
	laC4aTerminalFence(t, label, evs, core.EventTypeWorkflowCancelled)
}

// F-C in-flight completion: a SignalReceived after the stranded signal step's StepStarted is
// durable when cancellation is requested. The step completes from that delivery (B4 permits
// completing in-flight work), and its successor is not activated.
func TestLA_C4a_FC_SignalDeliveredInFlight_CompletesWithoutActivation(t *testing.T) {
	laC4aFCDelivered(t, false)
}

func TestLA_C4a_FC_SignalDeliveredInFlight_CompletesWithoutActivation_AfterRestart(t *testing.T) {
	laC4aFCDelivered(t, true)
}

func laC4aFCDelivered(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a F-C in-flight signal delivery", restart)
	def := laDefWait()
	a := &stepHandler{id: "la_a", outputs: map[string]any{}}
	z := &stepHandler{id: "la_z", outputs: map[string]any{}}
	v := laC4aNew(t, def, 1, a, z)
	v.iid = laSubmit(t, v.e, def.ID)
	v.fs.arm(laFault{op: laOpCreateWaitRecord, nth: 1, stepID: "w"})
	laWantInjected(t, laTickUntilTripped(t, v.e, v.fs))
	inst := laInstance(t, v.s, v.iid)
	// A durable delivery after w's StepStarted (the B3 shape: SignalReceived carries
	// signal_name and payload, EVENTLOG_FORMAT §6).
	laC4aAppendRaw(t, v.e, inst, core.EventTypeSignalReceived, "", map[string]any{
		"signal_name": "go",
		"payload":     map[string]any{"v": "in-flight"},
	})
	if err := v.e.Cancel(context.Background(), v.iid, "operator", false); err != nil {
		t.Fatalf("%s: Cancel: %v", label, err)
	}
	if restart {
		v.restart()
	}
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	evs := laEvents(t, v.s, v.iid)
	final := laInstance(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d status=%s", label, n, len(errs), final.Status)
	if len(errs) != 0 {
		t.Errorf("%s: drain ticks returned errors: %v", label, errs)
	}
	if final.Status != core.InstanceStatusCancelled {
		t.Errorf("%s: status = %s, want cancelled", label, final.Status)
	}
	var completed []core.ExecutionEvent
	for _, ev := range evs {
		if ev.EventType == core.EventTypeStepCompleted && ev.StepID == "w" {
			completed = append(completed, ev)
		}
	}
	if len(completed) != 1 {
		t.Fatalf("%s: StepCompleted(w) events = %d, want 1; log: %s", label, len(completed), laSummary(t, evs))
	}
	var p stepCompletedPayload
	decodePayload(t, completed[0], &p)
	if p.Outputs["v"] != "in-flight" {
		t.Errorf("%s: StepCompleted(w) outputs = %v, want the in-flight delivery payload", label, p.Outputs)
	}
	if got := laC4aStepFailedCodes(t, evs, "w"); len(got) != 0 {
		t.Errorf("%s: StepFailed(w) appended despite the in-flight delivery (codes %v)", label, got)
	}
	if laHas(evs, core.EventTypeStepStarted, "z") || laCalls(z) != 0 {
		t.Errorf("%s: successor z activated on a cancel-requested instance", label)
	}
	laC4aTerminalFence(t, label, evs, core.EventTypeWorkflowCancelled)
}

// ── P4: F3(a) lower bound ───────────────────────────────────────────────────────────

// laC4aDefTwoWait: a → w1 → w2 → z, both waits on signal "go".
func laC4aDefTwoWait() core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.c4a.twowait", Version: "1.0.0", Namespace: "la", Name: "twowait",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "la_a"), signalStep("w1", "go"), signalStep("w2", "go"), nativeStep("z", "la_z"),
		},
		Transitions: []core.Transition{{From: "a", To: "w1"}, {From: "w1", To: "w2"}, {From: "w2", To: "z"}},
		InitialStep: "a", FinalSteps: []string{"z"}, Metadata: map[string]any{},
	}
}

// P4: w1 is delivered (its SignalReceived precedes w2's StepStarted); w2's wait_record
// creation then fails. Recovery must NOT complete w2 from w1's earlier delivery.
func TestLA_C4a_P4_EarlierDeliveryDoesNotCompleteLaterWait(t *testing.T) {
	laC4aP4(t, false)
}

func TestLA_C4a_P4_EarlierDeliveryDoesNotCompleteLaterWait_AfterRestart(t *testing.T) {
	laC4aP4(t, true)
}

func laC4aP4(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a P4 earlier delivery", restart)
	ctx := context.Background()
	def := laC4aDefTwoWait()
	a := &stepHandler{id: "la_a", outputs: map[string]any{}}
	z := &stepHandler{id: "la_z", outputs: map[string]any{}}
	v := laC4aNew(t, def, 1, a, z)
	v.iid = laSubmit(t, v.e, def.ID)
	for i := 0; i < 5; i++ {
		_ = v.e.Tick(ctx)
	}
	if err := v.e.Signal(ctx, v.iid, "go", map[string]any{"v": "first"}); err != nil {
		t.Fatalf("%s: Signal: %v", label, err)
	}
	for i := 0; i < 5 && !laHas(laEvents(t, v.s, v.iid), core.EventTypeStepCompleted, "w1"); i++ {
		_ = v.e.Tick(ctx)
	}
	if !laHas(laEvents(t, v.s, v.iid), core.EventTypeStepCompleted, "w1") {
		t.Fatalf("%s: setup: w1 not completed", label)
	}
	v.fs.arm(laFault{op: laOpCreateWaitRecord, nth: 1, stepID: "w2"})
	laWantInjected(t, laTickUntilTripped(t, v.e, v.fs))
	if restart {
		v.restart()
	}
	for i := 0; i < 10; i++ {
		_ = v.e.Tick(ctx)
	}
	evs := laEvents(t, v.s, v.iid)
	inst := laInstance(t, v.s, v.iid)
	t.Logf("%s: status=%s current_steps=%v SignalReceived=%d", label, inst.Status, inst.CurrentSteps, laCount(evs, core.EventTypeSignalReceived))
	if laHas(evs, core.EventTypeStepCompleted, "w2") {
		t.Errorf("%s: w2 completed from w1's earlier delivery (F3a lower bound); log: %s", label, laSummary(t, evs))
	}
	if inst.Status == core.InstanceStatusCompleted {
		t.Errorf("%s: workflow completed from w1's earlier delivery", label)
	}
	if inst.Status != core.InstanceStatusWaiting || !laC4aHasStep(inst.CurrentSteps, "w2") {
		t.Errorf("%s: want w2 re-entered its wait (status waiting, current_steps [w2]); got status=%s current=%v",
			label, inst.Status, inst.CurrentSteps)
	}
	if n := laCount(evs, core.EventTypeSignalReceived); n != 1 {
		t.Errorf("%s: SignalReceived events = %d, want 1 (no re-delivery)", label, n)
	}
}

// ── Guards (a), (b), durable WorkflowFailed: each must fail if that guard alone is removed ──

// Guard (a): the instance is cancel-requested and its latest StepFailed{retrying:false} for a
// fallback step has no StepFallbackActivated (the route was lost). completeCommittedRoutes
// must not complete the route: the B4 cancellation gate owns the instance.
func TestLA_C4a_GuardA_CancelRequested_NoRouteCompletion(t *testing.T) {
	laC4aGuardA(t, false)
}

func TestLA_C4a_GuardA_CancelRequested_NoRouteCompletion_AfterRestart(t *testing.T) {
	laC4aGuardA(t, true)
}

func laC4aGuardA(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a guard (a) cancellationRequested", restart)
	def := laDefRoute("la.c4a.guarda", nil, true)
	ha, hfb := laFallbackHandlers()
	v := laC4aNew(t, def, 1, ha, hfb)
	v.iid = laSubmit(t, v.e, def.ID)
	inst := laInstance(t, v.s, v.iid)
	ctx := context.Background()
	if _, err := v.e.emitStepStarted(ctx, inst, "a", 1, map[string]any{}); err != nil {
		t.Fatalf("%s: emitStepStarted: %v", label, err)
	}
	if err := v.e.emitStepFailed(ctx, inst, "a", 1, core.StepError{Code: "la_c4a_probe", Message: "probe"}, false); err != nil {
		t.Fatalf("%s: emitStepFailed: %v", label, err)
	}
	if err := v.e.Cancel(ctx, v.iid, "la-c4a guard a", false); err != nil {
		t.Fatalf("%s: Cancel: %v", label, err)
	}
	if restart {
		v.restart()
	}
	laC4aHydrate(t, v.s, v.e, v.iid)
	laC4aNoFallback(t, label, laEvents(t, v.s, v.iid))
	n, errs := laTickUntil(t, v.e, v.s, v.iid, laK, laIsTerminal)
	evs := laEvents(t, v.s, v.iid)
	t.Logf("%s: ticks=%d tick_errors=%d status=%s", label, n, len(errs), laInstance(t, v.s, v.iid).Status)
	if len(errs) != 0 {
		t.Errorf("%s: drain ticks returned errors: %v", label, errs)
	}
	laC4aNoFallback(t, label, evs)
	if got := laInstance(t, v.s, v.iid).Status; got != core.InstanceStatusCancelled {
		t.Errorf("%s: status = %s, want cancelled", label, got)
	}
	if laCalls(hfb) != 0 {
		t.Errorf("%s: fallback fb executed %d times, want 0", label, laCalls(hfb))
	}
}

// Guard (b): the latest StepFailed for a fallback step has code "cancelled" (cancel.go emits
// it with no routing decision). Cancellation is NOT requested here, so guard (a) is inactive.
func TestLA_C4a_GuardB_CancelledCode_NoRouteCompletion(t *testing.T) {
	laC4aGuardB(t, false)
}

func TestLA_C4a_GuardB_CancelledCode_NoRouteCompletion_AfterRestart(t *testing.T) {
	laC4aGuardB(t, true)
}

func laC4aGuardB(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a guard (b) code cancelled", restart)
	def := laDefRoute("la.c4a.guardb", nil, true)
	ha, hfb := laFallbackHandlers()
	v := laC4aNew(t, def, 1, ha, hfb)
	v.iid = laSubmit(t, v.e, def.ID)
	inst := laInstance(t, v.s, v.iid)
	ctx := context.Background()
	if _, err := v.e.emitStepStarted(ctx, inst, "a", 1, map[string]any{}); err != nil {
		t.Fatalf("%s: emitStepStarted: %v", label, err)
	}
	serr := core.StepError{Code: "cancelled", Message: "step cancelled while waiting for signal"}
	if err := v.e.emitStepFailed(ctx, inst, "a", 1, serr, false); err != nil {
		t.Fatalf("%s: emitStepFailed: %v", label, err)
	}
	if restart {
		v.restart()
	}
	laC4aHydrate(t, v.s, v.e, v.iid)
	evs := laEvents(t, v.s, v.iid)
	laC4aNoFallback(t, label, evs)
	if laCalls(hfb) != 0 {
		t.Errorf("%s: fallback fb executed %d times, want 0", label, laCalls(hfb))
	}
}

// Durable WorkflowFailed guard: a WorkflowFailed is durable (followed by its compensation
// bracket opener, so the terminal fence does not apply) and the latest StepFailed for a
// fallback step has no route. No route may be completed after a durable WorkflowFailed.
func TestLA_C4a_DurableWorkflowFailed_NoRouteCompletion(t *testing.T) {
	laC4aDurableWF(t, false)
}

func TestLA_C4a_DurableWorkflowFailed_NoRouteCompletion_AfterRestart(t *testing.T) {
	laC4aDurableWF(t, true)
}

func laC4aDurableWF(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a durable WorkflowFailed guard", restart)
	def := laDefRoute("la.c4a.durwf", nil, true)
	ha, hfb := laFallbackHandlers()
	v := laC4aNew(t, def, 1, ha, hfb)
	v.iid = laSubmit(t, v.e, def.ID)
	inst := laInstance(t, v.s, v.iid)
	ctx := context.Background()
	if _, err := v.e.emitStepStarted(ctx, inst, "a", 1, map[string]any{}); err != nil {
		t.Fatalf("%s: emitStepStarted: %v", label, err)
	}
	if err := v.e.emitStepFailed(ctx, inst, "a", 1, core.StepError{Code: "la_c4a_probe", Message: "probe"}, false); err != nil {
		t.Fatalf("%s: emitStepFailed: %v", label, err)
	}
	// Both appended WITHOUT projection: the status guard cannot mask the event-level guard.
	laC4aAppendRaw(t, v.e, inst, core.EventTypeWorkflowFailed, "a", workflowFailedPayload{
		StepID: "a", Error: core.StepError{Code: "la_c4a_probe", Message: "probe"},
	})
	laC4aAppendRaw(t, v.e, inst, core.EventTypeWorkflowCompensating, "", workflowCompensatingPayload{FromStep: "a"})
	if restart {
		v.restart()
	}
	laC4aHydrate(t, v.s, v.e, v.iid)
	evs := laEvents(t, v.s, v.iid)
	laC4aNoFallback(t, label, evs)
	if n := len(evs); n == 0 || evs[n-1].EventType != core.EventTypeWorkflowCompensating {
		t.Errorf("%s: an event was appended after the durable WorkflowFailed bracket; log: %s", label, laSummary(t, evs))
	}
}

// ── Owed compensation after a durable WorkflowFailed (CE ruling, LA-C4a) ─────────────

// Route (3) with a compensation plan, where the WorkflowFailed projection write fails once.
// The owed bracket must complete exactly as on the no-fault path (failure.go route (3):
// emitWorkflowFailed, then maybeCompensate). Expect the same (event_type, step_id) sequence
// as the no-fault baseline, each undo once in reverse order, and WorkflowCompensated last.
func TestLA_C4a_OwedCompensation_WorkflowFailedProjectionFailure_CompletesRoute3(t *testing.T) {
	laC4aOwedComp(t, false)
}

func TestLA_C4a_OwedCompensation_WorkflowFailedProjectionFailure_CompletesRoute3_AfterRestart(t *testing.T) {
	laC4aOwedComp(t, true)
}

func laC4aOwedComp(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a owed compensation (WorkflowFailed projection fails)", restart)
	base := newLACompRun()
	bs, biid, _, _ := laDrive(t, compDef(), nil, false, base.handlers...)
	want, _ := eventPairs(t, bs, biid)

	r := newLACompRun()
	// The only failed-status UpsertInstance in compDef is WorkflowFailed's projection.
	ft := laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusFailed}
	s, iid, n, errs := laDrive(t, compDef(), &ft, restart, r.handlers...)
	t.Logf("%s: ticks=%d tick_errors=%d undo_order=%q", label, n, len(errs), r.undoOrder())
	laCheck(t, s, iid, label, core.InstanceStatusCompensated)
	evs := laEvents(t, s, iid)
	if wfs := laWorkflowFailedPayloads(t, evs); len(wfs) != 1 {
		t.Errorf("%s: WorkflowFailed count = %d, want exactly 1", label, len(wfs))
	}
	if got := r.undoOrder(); got != "undo-c,undo-b,undo-a" {
		t.Errorf("%s: undo order = %q, want %q (each undo once, reverse, x skipped)", label, got, "undo-c,undo-b,undo-a")
	}
	if got := laCalls(r.hd); got != 1 {
		t.Errorf("%s: step d executions = %d, want 1", label, got)
	}
	got, _ := eventPairs(t, s, iid)
	laDiffPairs(t, label, want, got)
}

// Guard: WorkflowFailed durable with NO compensation plan. The fence re-projects the terminal
// event and nothing else: no WorkflowCompensating and no undo, and the sequence equals the
// no-fault baseline.
func TestLA_C4a_NoPlan_WorkflowFailedProjectionFailure_NoCompensation(t *testing.T) {
	laC4aNoPlan(t, false)
}

func TestLA_C4a_NoPlan_WorkflowFailedProjectionFailure_NoCompensation_AfterRestart(t *testing.T) {
	laC4aNoPlan(t, true)
}

func laC4aNoPlan(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a no plan, WorkflowFailed projection fails", restart)
	def := compDef()
	def.Compensation = nil
	base := newLACompRun()
	bs, biid, _, _ := laDrive(t, def, nil, false, base.handlers...)
	want, _ := eventPairs(t, bs, biid)

	r := newLACompRun()
	ft := laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusFailed}
	s, iid, n, errs := laDrive(t, def, &ft, restart, r.handlers...)
	t.Logf("%s: ticks=%d tick_errors=%d", label, n, len(errs))
	laCheck(t, s, iid, label, core.InstanceStatusFailed)
	evs := laEvents(t, s, iid)
	if c := laCount(evs, core.EventTypeWorkflowCompensating); c != 0 {
		t.Errorf("%s: WorkflowCompensating events = %d, want 0 (no plan)", label, c)
	}
	if got := r.undoOrder(); got != "" {
		t.Errorf("%s: undo handlers ran (%q) without a plan", label, got)
	}
	if wfs := laWorkflowFailedPayloads(t, evs); len(wfs) != 1 {
		t.Errorf("%s: WorkflowFailed count = %d, want exactly 1", label, len(wfs))
	}
	got, _ := eventPairs(t, s, iid)
	laDiffPairs(t, label, want, got)
}

// Guard: a plan exists but no forward step completed (owed requires ≥1 completed step, the
// maybeCompensate rule). The failing step d is the only step, so nothing is undone.
func TestLA_C4a_NothingCompleted_WorkflowFailedProjectionFailure_NoCompensation(t *testing.T) {
	laC4aNothingCompleted(t, false)
}

func TestLA_C4a_NothingCompleted_WorkflowFailedProjectionFailure_NoCompensation_AfterRestart(t *testing.T) {
	laC4aNothingCompleted(t, true)
}

func laC4aNothingCompleted(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C4a plan but nothing completed, WorkflowFailed projection fails", restart)
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.c4a.nothing", Version: "1.0.0", Namespace: "la", Name: "nothing",
		Triggers:     []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:        []core.Step{nativeStep("d", "hd")},
		Compensation: &core.CompensationPlan{Steps: []core.CompensationStep{{StepID: "d", UndoHandler: "undo-a"}}},
		InitialStep:  "d", FinalSteps: []string{"d"}, Metadata: map[string]any{},
	}
	base := newLACompRun()
	bs, biid, _, _ := laDrive(t, def, nil, false, base.handlers...)
	want, _ := eventPairs(t, bs, biid)

	r := newLACompRun()
	ft := laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusFailed}
	s, iid, n, errs := laDrive(t, def, &ft, restart, r.handlers...)
	t.Logf("%s: ticks=%d tick_errors=%d", label, n, len(errs))
	laCheck(t, s, iid, label, core.InstanceStatusFailed)
	evs := laEvents(t, s, iid)
	if c := laCount(evs, core.EventTypeWorkflowCompensating); c != 0 {
		t.Errorf("%s: WorkflowCompensating events = %d, want 0 (nothing completed)", label, c)
	}
	if got := r.undoOrder(); got != "" {
		t.Errorf("%s: undo handlers ran (%q) with nothing completed", label, got)
	}
	got, _ := eventPairs(t, s, iid)
	laDiffPairs(t, label, want, got)
}
