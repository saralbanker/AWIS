package engine

// LA-C6 R3-M1 (PR #4 review round 3): the signal and timeout gate and the route-recovery stage
// (route_recovery.go). TEST-ONLY file. Kept self-contained (it uses only helpers that already exist
// on 248b7c2) so that its FAIL-before run can compile against the pre-fix tree.
//
// Oracle: the NO-FAULT control of the same fixture. A faulted run must equal it on final status,
// the (event_type, step_id) sequence, handler and undo invocations, and must contain no
// SignalReceived and exactly one WorkflowFailed attributed to b1.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// c6Faults wraps laFaultStorage with two LA-C6 faults. The next wfFails WorkflowFailed appends fail
// before the write (nothing commits), then forward. While readFail is set every ReadEvents fails.
type c6Faults struct {
	*laFaultStorage
	mu          sync.Mutex
	wfRemaining int
	readFail    bool
}

func newC6Faults(fs *laFaultStorage, wfFails int) *c6Faults {
	return &c6Faults{laFaultStorage: fs, wfRemaining: wfFails}
}

func (f *c6Faults) setReadFail(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readFail = v
}

func (f *c6Faults) AppendEvent(ctx context.Context, ev core.ExecutionEvent) error {
	if ev.EventType == core.EventTypeWorkflowFailed {
		f.mu.Lock()
		hit := f.wfRemaining > 0
		if hit {
			f.wfRemaining--
		}
		f.mu.Unlock()
		if hit {
			return laInjected(laOpAppendEvent)
		}
	}
	return f.laFaultStorage.AppendEvent(ctx, ev)
}

func (f *c6Faults) ReadEvents(ctx context.Context, iid core.InstanceID, fromSeq int) ([]core.ExecutionEvent, error) {
	f.mu.Lock()
	fail := f.readFail
	f.mu.Unlock()
	if fail {
		return nil, errLAInjected
	}
	return f.laFaultStorage.ReadEvents(ctx, iid, fromSeq)
}

// c6SiblingDef: root → {w, b1}. w is a parked wait (a signal "go", or a timeout step); b1 fails with
// a route-(3) decision (no fallback, no on_error). Compensation covers root, w and b1.
func c6SiblingDef(w core.Step) core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.c6.sibling", Version: "1.0.0", Namespace: "la", Name: "c6sibling",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("root", "la_root"), w, nativeStep("b1", "la_b1")},
		Transitions: []core.Transition{{From: "root", To: "w"}, {From: "root", To: "b1"}},
		Compensation: &core.CompensationPlan{Steps: []core.CompensationStep{
			{StepID: "root", UndoHandler: "undo-root"},
			{StepID: "w", UndoHandler: "undo-w"},
			{StepID: "b1", UndoHandler: "undo-b1"},
		}},
		InitialStep: "root", FinalSteps: []string{"w", "b1"}, Metadata: map[string]any{},
	}
}

// c6SiblingRun drives c6SiblingDef. Tick 1 completes root. The signal "go" is pre-sent when
// withSignal. Tick 2 (the trip tick) gathers w (parks it) and b1 (fails route 3), and the timeout
// scan runs after it. wfFails is the number of WorkflowFailed appends that fail (0 = the no-fault
// control). restart replaces the engine on the same storage after the trip tick.
func c6SiblingRun(t *testing.T, w core.Step, withSignal bool, wfFails int, restart bool) c5Outcome {
	t.Helper()
	def := c6SiblingDef(w)
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	var u c5Undo
	rootH := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1H := &stepHandler{id: "la_b1", err: errLAHandler}
	handlers := []core.StepHandler{rootH, b1H, u.handler("undo-root"), u.handler("undo-w"), u.handler("undo-b1")}
	var st core.StoragePort = fs
	if wfFails > 0 {
		st = newC6Faults(fs, wfFails)
	}
	e := laEngine(st, clk, ids, 2, handlers...)
	iid := laSubmit(t, e, def.ID)
	ctx := context.Background()
	if err := e.Tick(ctx); err != nil { // tick 1: root dispatched and completed
		t.Fatalf("tick 1: %v", err)
	}
	if withSignal {
		if err := e.Signal(ctx, iid, "go", map[string]any{"by": "alice"}); err != nil {
			t.Fatalf("Signal: %v", err)
		}
	}
	var errs []error
	trip := e.Tick(ctx) // tick 2: the trip tick
	if wfFails > 0 && !errors.Is(trip, errLAInjected) {
		t.Fatalf("trip tick: the injected WorkflowFailed failure did not surface through Tick: %v", trip)
	}
	if wfFails == 0 && trip != nil {
		t.Fatalf("no-fault trip tick returned %v", trip)
	}
	if trip != nil {
		errs = append(errs, trip)
	}
	if restart {
		e = laEngine(st, clk, ids, 2, handlers...)
	}
	_, more := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	errs = append(errs, more...)
	out := c5Outcome{
		s:      s,
		iid:    iid,
		status: laInstance(t, s, iid).Status,
		evs:    laEvents(t, s, iid),
		calls:  map[string]int32{"la_root": laCalls(rootH), "la_b1": laCalls(b1H)},
		errs:   len(errs),
		undos:  u.calls(),
	}
	out.pairs, _ = eventPairs(t, s, iid)
	return out
}

// c6SiblingCheck asserts the faulted run equals the no-fault control (see the file comment).
func c6SiblingCheck(t *testing.T, label string, base, got c5Outcome) {
	t.Helper()
	if base.errs != 0 {
		t.Fatalf("%s: no-fault control returned %d tick errors", label, base.errs)
	}
	if n := c5Count(got.evs, core.EventTypeSignalReceived); n != 0 {
		t.Errorf("%s: %d SignalReceived event(s); the sibling w must not be delivered after b1's committed route (3) decision", label, n)
	}
	wfs := laWorkflowFailedPayloads(t, got.evs)
	if len(wfs) != 1 || wfs[0].StepID != "b1" || wfs[0].Error.Code != "handler_error" {
		t.Errorf("%s: WorkflowFailed payloads=%+v; want exactly one {step_id:b1, code:handler_error}", label, wfs)
	}
	if got.status != base.status {
		t.Errorf("%s: status=%s; the no-fault run ends %s", label, got.status, base.status)
	}
	laDiffPairs(t, label, base.pairs, got.pairs)
	c5DiffCalls(t, label, base.calls, got.calls)
	if got.undos != base.undos {
		t.Errorf("%s: undo invocations=%q; the no-fault run undoes %q", label, got.undos, base.undos)
	}
}

// c6SignalCase runs the signal sibling with wfFails failing WorkflowFailed appends.
func c6SignalCase(t *testing.T, wfFails int, restart bool) {
	t.Helper()
	label := laLabel("LA-C6 R3-M1 signal sibling, WorkflowFailed appends failing", restart)
	base := c6SiblingRun(t, signalStep("w", "go"), true, 0, false)
	if base.undos != "undo-root" {
		t.Fatalf("%s: no-fault control undid %q; the oracle is broken (want undo-root)", label, base.undos)
	}
	got := c6SiblingRun(t, signalStep("w", "go"), true, wfFails, restart)
	c6SiblingCheck(t, label, base, got)
}

// c6TimeoutCase runs the timeout sibling (a 1ms wait, so it is due in the trip tick's timeout scan).
func c6TimeoutCase(t *testing.T, action string, wfFails int, restart bool) {
	t.Helper()
	label := laLabel("LA-C6 R3-M1 timeout sibling ("+action+"), WorkflowFailed appends failing", restart)
	base := c6SiblingRun(t, signalStepTimeout("w", "go", "1ms", action), false, 0, false)
	if base.undos != "undo-root" {
		t.Fatalf("%s: no-fault control undid %q; the oracle is broken (want undo-root)", label, base.undos)
	}
	got := c6SiblingRun(t, signalStepTimeout("w", "go", "1ms", action), false, wfFails, restart)
	c6SiblingCheck(t, label, base, got)
}

// ── Signal sibling: the WorkflowFailed append fails once (the card's scenario) ───────────

func TestLA_C6_R3M1_SignalSibling_WFAppendFailsOnce(t *testing.T) {
	c6SignalCase(t, 1, false)
}

func TestLA_C6_R3M1_SignalSibling_WFAppendFailsOnce_AfterRestart(t *testing.T) {
	c6SignalCase(t, 1, true)
}

// ── Signal sibling: the completion's own WorkflowFailed append fails too, so the trip tick leaves the
// decision awaiting and the gate in Resolve (not the same-tick completion) is what refuses delivery ──

func TestLA_C6_R3M1_SignalSibling_WFAppendFailsTwice_GateRefusesDelivery(t *testing.T) {
	c6SignalCase(t, 2, false)
}

func TestLA_C6_R3M1_SignalSibling_WFAppendFailsTwice_GateRefusesDelivery_AfterRestart(t *testing.T) {
	c6SignalCase(t, 2, true)
}

// ── Timeout sibling: the wait is due in the trip tick (routing it would be the divergence) ──

func TestLA_C6_R3M1_TimeoutFailSibling_WFAppendFailsOnce(t *testing.T) {
	c6TimeoutCase(t, "fail", 1, false)
}

func TestLA_C6_R3M1_TimeoutFailSibling_WFAppendFailsOnce_AfterRestart(t *testing.T) {
	c6TimeoutCase(t, "fail", 1, true)
}

func TestLA_C6_R3M1_TimeoutFailSibling_WFAppendFailsTwice_GateRefusesTimeout(t *testing.T) {
	c6TimeoutCase(t, "fail", 2, false)
}

func TestLA_C6_R3M1_TimeoutFailSibling_WFAppendFailsTwice_GateRefusesTimeout_AfterRestart(t *testing.T) {
	c6TimeoutCase(t, "fail", 2, true)
}

func TestLA_C6_R3M1_TimeoutContinueSibling_WFAppendFailsOnce(t *testing.T) {
	c6TimeoutCase(t, "continue", 1, false)
}

func TestLA_C6_R3M1_TimeoutContinueSibling_WFAppendFailsOnce_AfterRestart(t *testing.T) {
	c6TimeoutCase(t, "continue", 1, true)
}

func TestLA_C6_R3M1_TimeoutContinueSibling_WFAppendFailsTwice_GateRefusesTimeout(t *testing.T) {
	c6TimeoutCase(t, "continue", 2, false)
}

func TestLA_C6_R3M1_TimeoutContinueSibling_WFAppendFailsTwice_GateRefusesTimeout_AfterRestart(t *testing.T) {
	c6TimeoutCase(t, "continue", 2, true)
}

// TestLA_C6_R3M1_GateReadErrorAndAwaiting: a fresh engine (the restart shape: empty in-memory
// waits and route set) faces an awaiting route (3). While ReadEvents fails, Resolve refuses without
// delivering; once reads succeed it still refuses, because the decision is awaiting. The sibling
// is never delivered, and the run ends as the no-fault control.
func TestLA_C6_R3M1_GateReadErrorAndAwaiting(t *testing.T) {
	w := signalStep("w", "go")
	base := c6SiblingRun(t, w, true, 0, false)

	def := c6SiblingDef(w)
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	var u c5Undo
	rootH := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1H := &stepHandler{id: "la_b1", err: errLAHandler}
	handlers := []core.StepHandler{rootH, b1H, u.handler("undo-root"), u.handler("undo-w"), u.handler("undo-b1")}
	st := newC6Faults(fs, 2) // the settle append and the same-tick completion both fail
	e := laEngine(st, clk, ids, 2, handlers...)
	iid := laSubmit(t, e, def.ID)
	ctx := context.Background()
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("tick 1: %v", err)
	}
	if err := e.Signal(ctx, iid, "go", map[string]any{"by": "alice"}); err != nil {
		t.Fatalf("Signal: %v", err)
	}
	if trip := e.Tick(ctx); !errors.Is(trip, errLAInjected) {
		t.Fatalf("trip tick: injected failures did not surface: %v", trip)
	}
	if got := laInstance(t, s, iid).Status; got != core.InstanceStatusWaiting {
		t.Fatalf("setup: status=%s; want waiting (w parked, WorkflowFailed not durable)", got)
	}

	fresh := laEngine(st, clk, ids, 2, handlers...)
	st.setReadFail(true)
	if _, _, ok := fresh.Resolve(ctx, iid, "go"); ok {
		t.Errorf("read error in the committed-route check: Resolve delivered; it must refuse this tick")
	}
	st.setReadFail(false)
	if _, _, ok := fresh.Resolve(ctx, iid, "go"); ok {
		t.Errorf("committed route (3) awaiting its terminal event: Resolve delivered; it must refuse")
	}
	if n := c5Count(laEvents(t, s, iid), core.EventTypeSignalReceived); n != 0 {
		t.Errorf("%d SignalReceived event(s) after the refused Resolve calls; want 0", n)
	}

	_, errs := laTickUntil(t, fresh, s, iid, laK, laIsTerminal)
	got := c5Outcome{
		s: s, iid: iid, status: laInstance(t, s, iid).Status, evs: laEvents(t, s, iid),
		calls: map[string]int32{"la_root": laCalls(rootH), "la_b1": laCalls(b1H)},
		errs:  len(errs), undos: u.calls(),
	}
	got.pairs, _ = eventPairs(t, s, iid)
	c6SiblingCheck(t, "LA-C6 R3-M1 gate read-error and awaiting", base, got)
}
