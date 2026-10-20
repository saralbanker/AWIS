package engine

// LA-C8a (PR #4 review round 5; founder checkpoint 1, 2026-10-20). TEST-ONLY file.
//
//   - R5-2 (TestLA_C8a_R52_*): a route (3) decision whose WorkflowFailed append fails in a tick that
//     is stopped, with the instance parked `waiting` by a sibling signal wait. The oracle is the
//     NO-FAULT control of the same fixture with the same stop timing: the hook closes stop inside the
//     first WorkflowFailed append and does not fail it. After the stopped tick ALONE, the faulted run
//     must end where the control ends (c6SiblingCheck). tick.go runs the committed-route stage in a
//     stopped tick, so these tests fail on the pre-fix tick.go.
//   - W6 (TestLA_C8a_W6_*, gated by LA_DEFERRED=1): the restart residual of R5-2. The stopped tick's
//     WorkflowFailed fails and so does the stage's completion; the process then exits.
//   - R5-5 / LA-R7 (TestLA_C8a_R55_*, gated by LA_DEFERRED=1): Cancel during a tick's WorkflowFailed
//     append.
//
// Fault models: laModelFailBeforeWrite (the append does not commit) and ack-lost (the append commits,
// then the caller sees an error); see laFaultStorage in la_stranded_repro_test.go.

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// ── R5-2 fixtures ──────────────────────────────────────────────────────────────────────────────

// c8aStopOnWF wraps laFaultStorage for the stop-trip fixtures. The first WorkflowFailed append runs
// onFirst (the test closes stop there) before its own outcome. The next `fails` WorkflowFailed appends
// (the first one included) then fail: before the write, or after it when ackLost (the write commits,
// then the error is returned). Every other append is forwarded.
type c8aStopOnWF struct {
	*laFaultStorage
	mu      sync.Mutex
	onFirst func()
	fails   int
	ackLost bool
	wfCalls int
}

func (f *c8aStopOnWF) AppendEvent(ctx context.Context, ev core.ExecutionEvent) error {
	if ev.EventType != core.EventTypeWorkflowFailed {
		return f.laFaultStorage.AppendEvent(ctx, ev)
	}
	f.mu.Lock()
	f.wfCalls++
	first := f.wfCalls == 1
	hit := f.fails > 0
	if hit {
		f.fails--
	}
	f.mu.Unlock()
	if first && f.onFirst != nil {
		f.onFirst()
	}
	if !hit {
		return f.laFaultStorage.AppendEvent(ctx, ev)
	}
	if f.ackLost {
		if err := f.laFaultStorage.AppendEvent(ctx, ev); err != nil {
			return err
		}
	}
	return laInjected(laOpAppendEvent)
}

// c8aRig is one run of a fixture, read right after its stopped tick: the store, the engine, the
// handlers, and what the stopped tick itself appended.
type c8aRig struct {
	s       *storage.SQLiteStorage
	st      *c8aStopOnWF
	clk     *fakeClock
	ids     func() string
	e       *Engine
	iid     core.InstanceID
	u       *c5Undo
	steps   []core.StepHandler
	trip    error                 // the stopped tick's error
	tickEvs []core.ExecutionEvent // the events the stopped tick appended
}

// c8aStopTrip builds def, runs pre live ticks, and then runs ONE stopped tick, e.tick(ctx, stop), in
// which stop is closed inside the first WorkflowFailed append (see c8aStopOnWF). No other tick runs.
func c8aStopTrip(t *testing.T, def core.WorkflowDefinition, pre, fails int, ackLost bool, u *c5Undo, steps ...core.StepHandler) *c8aRig {
	t.Helper()
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	stop, closeStop := laStopOnce()
	st := &c8aStopOnWF{laFaultStorage: fs, onFirst: closeStop, fails: fails, ackLost: ackLost}
	e := laEngine(st, clk, ids, 2, steps...)
	iid := laSubmit(t, e, def.ID)
	ctx := context.Background()
	for i := 0; i < pre; i++ {
		if err := e.Tick(ctx); err != nil {
			t.Fatalf("live tick %d: %v", i+1, err)
		}
	}
	before := len(laEvents(t, s, iid))
	trip := e.tick(ctx, stop)
	evs := laEvents(t, s, iid)
	return &c8aRig{s: s, st: st, clk: clk, ids: ids, e: e, iid: iid, u: u, steps: steps,
		trip: trip, tickEvs: evs[before:]}
}

// c8aSibling runs the c6SiblingDef fixture (root -> {w, b1}; b1 fails route 3) through its stop-trip
// tick. Tick 1 is live and completes root. fails and ackLost select the WorkflowFailed fault.
func c8aSibling(t *testing.T, w core.Step, fails int, ackLost bool) *c8aRig {
	t.Helper()
	u := &c5Undo{}
	rootH := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1H := &stepHandler{id: "la_b1", err: errLAHandler}
	return c8aStopTrip(t, c6SiblingDef(w), 1, fails, ackLost, u,
		rootH, b1H, u.handler("undo-root"), u.handler("undo-w"), u.handler("undo-b1"))
}

// c8aCalls returns the Execute count of every stepHandler in steps, keyed by handler id.
func c8aCalls(steps []core.StepHandler) map[string]int32 {
	out := map[string]int32{}
	for _, h := range steps {
		if sh, ok := h.(*stepHandler); ok {
			out[sh.id] = laCalls(sh)
		}
	}
	return out
}

// outcome reads the run's observable state in the c5Outcome shape (see c6SiblingCheck). A faulted
// stopped tick counts one tick error.
func (r *c8aRig) outcome(t *testing.T) c5Outcome {
	t.Helper()
	out := c5Outcome{
		s:      r.s,
		iid:    r.iid,
		status: laInstance(t, r.s, r.iid).Status,
		evs:    laEvents(t, r.s, r.iid),
		calls:  c8aCalls(r.steps),
		undos:  r.u.calls(),
	}
	if r.trip != nil {
		out.errs = 1
	}
	out.pairs, _ = eventPairs(t, r.s, r.iid)
	return out
}

// drive ticks e live (no stop) until the instance is terminal (bound laK) and returns the outcome.
// Pass r.e to continue the same process, or a fresh engine on the same store for a restart.
func (r *c8aRig) drive(t *testing.T, e *Engine) c5Outcome {
	t.Helper()
	_, more := laTickUntil(t, e, r.s, r.iid, laK, laIsTerminal)
	out := r.outcome(t)
	out.errs += len(more)
	return out
}

// c8aPairs projects evs to (event_type, step_id) pairs.
func c8aPairs(evs []core.ExecutionEvent) []pair {
	out := make([]pair, 0, len(evs))
	for _, ev := range evs {
		out = append(out, pair{Type: ev.EventType, StepID: ev.StepID})
	}
	return out
}

// c8aStepPairs keeps the pairs a gather, a delivery or a timeout appends (step, signal and fallback
// events) and drops the workflow-lifecycle and compensation events.
func c8aStepPairs(ps []pair) []pair {
	var out []pair
	for _, p := range ps {
		switch p.Type {
		case core.EventTypeStepStarted, core.EventTypeStepCompleted, core.EventTypeStepFailed,
			core.EventTypeSignalReceived, core.EventTypeStepFallbackActivated:
			out = append(out, p)
		}
	}
	return out
}

// c8aSignalTimeouts counts the StepFailed events with code signal_timeout in evs.
func c8aSignalTimeouts(t *testing.T, evs []core.ExecutionEvent) int {
	t.Helper()
	n := 0
	for _, ev := range evs {
		if ev.EventType != core.EventTypeStepFailed {
			continue
		}
		var p stepFailedPayload
		decodePayload(t, ev, &p)
		if p.Error.Code == "signal_timeout" {
			n++
		}
	}
	return n
}

// c8aLateTerminal returns the WorkflowFailed, WorkflowCompensating and WorkflowCompensated events
// appended after sequence seq.
func c8aLateTerminal(evs []core.ExecutionEvent, seq int) []core.ExecutionEvent {
	var out []core.ExecutionEvent
	for _, ev := range evs {
		if ev.SequenceNum <= seq {
			continue
		}
		switch ev.EventType {
		case core.EventTypeWorkflowFailed, core.EventTypeWorkflowCompensating, core.EventTypeWorkflowCompensated:
			out = append(out, ev)
		}
	}
	return out
}

// c8aStopTripCheck is the R5-2 (a)/(b) oracle: the faulted stopped tick ends where the no-fault
// control ends (same stop timing, no failed append), and it ends after that tick alone.
func c8aStopTripCheck(t *testing.T, label string, fails int, ackLost bool) {
	t.Helper()
	ctl := c8aSibling(t, signalStep("w", "go"), 0, false)
	if ctl.trip != nil {
		t.Fatalf("%s: no-fault stopped tick returned %v", label, ctl.trip)
	}
	base := ctl.outcome(t)
	if !isTerminal(base.status) || base.undos != "undo-root" {
		t.Fatalf("%s: oracle broken: the no-fault stopped tick ends %s undoing %q; want terminal, undo-root",
			label, base.status, base.undos)
	}
	got := c8aSibling(t, signalStep("w", "go"), fails, ackLost)
	if !errors.Is(got.trip, errLAInjected) {
		t.Fatalf("%s: the injected WorkflowFailed failure did not surface through the stopped tick: %v", label, got.trip)
	}
	out := got.outcome(t)
	if !isTerminal(out.status) {
		t.Errorf("%s: after the stopped tick alone the instance is %s; the no-fault control is %s and terminal",
			label, out.status, base.status)
	}
	c6SiblingCheck(t, label, base, out)
}

// ── R5-2 (a) and (b): the stopped tick completes the committed route ───────────────────────────

// TestLA_C8a_R52_StopTrip_WaitingSibling_WFFailsBeforeWrite (R5-2 (a)): stop closes inside the trip
// tick; b1's route (3) WorkflowFailed fails before its write; w is parked and no signal ever arrives.
// Pre-fix: the instance stays `waiting` after the stopped tick (the stage is skipped under stop).
func TestLA_C8a_R52_StopTrip_WaitingSibling_WFFailsBeforeWrite(t *testing.T) {
	c8aStopTripCheck(t, "LA-C8 R5-2 (a) stop in the trip tick, waiting sibling, no signal, WorkflowFailed fails before write", 1, false)
}

// TestLA_C8a_R52_StopTrip_WaitingSibling_WFAckLost (R5-2 (b)): the same, but the WorkflowFailed
// append commits and then returns an error (ack-lost).
func TestLA_C8a_R52_StopTrip_WaitingSibling_WFAckLost(t *testing.T) {
	c8aStopTripCheck(t, "LA-C8 R5-2 (b) stop in the trip tick, waiting sibling, no signal, WorkflowFailed ack-lost", 1, true)
}

// ── R5-2 (c): the stage does nothing else in a stopped tick ────────────────────────────────────

// TestLA_C8a_R52_StopTrip_NothingElseAppended (R5-2 (c)): the timeout sibling is due in the trip
// tick. The stopped tick must not add a gather, a delivery or a timeout: its step, signal and
// timeout events equal the control's (no SignalReceived, no signal_timeout). It must also complete
// the route as the control does. The first check holds on the pre-fix tree too (it is the
// constraint the fix must keep); the second fails there.
func TestLA_C8a_R52_StopTrip_NothingElseAppended(t *testing.T) {
	label := "LA-C8 R5-2 (c) stop in the trip tick, timeout sibling due, WorkflowFailed fails before write"
	w := signalStepTimeout("w", "go", "1ms", "fail")
	ctl := c8aSibling(t, w, 0, false)
	if ctl.trip != nil {
		t.Fatalf("%s: no-fault stopped tick returned %v", label, ctl.trip)
	}
	got := c8aSibling(t, w, 1, false)
	if !errors.Is(got.trip, errLAInjected) {
		t.Fatalf("%s: the injected WorkflowFailed failure did not surface through the stopped tick: %v", label, got.trip)
	}
	if n := c5Count(got.tickEvs, core.EventTypeSignalReceived); n != 0 {
		t.Errorf("%s: the stopped tick appended %d SignalReceived event(s); want 0", label, n)
	}
	if n := c8aSignalTimeouts(t, got.tickEvs); n != 0 {
		t.Errorf("%s: the stopped tick appended %d signal_timeout StepFailed event(s); want 0", label, n)
	}
	laDiffPairs(t, label+": step, signal and timeout events of the stopped tick",
		c8aStepPairs(c8aPairs(ctl.tickEvs)), c8aStepPairs(c8aPairs(got.tickEvs)))
	laDiffPairs(t, label+": the whole append sequence of the stopped tick",
		c8aPairs(ctl.tickEvs), c8aPairs(got.tickEvs))
}

// ── R5-2 (d): a running instance ───────────────────────────────────────────────────────────────

// TestLA_C8a_R52_StopTrip_RunningNoSibling (R5-2 (d)): laDefSingle, no parked sibling, route (3)
// WorkflowFailed fails in the stopped tick. A running instance is not parked, so the stage does not
// revisit it; SCAN_RUNNABLE does (hydrate completes its committed route). The state after the
// stopped tick is therefore not asserted here. The state after the process resumes (live ticks, the
// same engine) must equal the no-fault stop control.
func TestLA_C8a_R52_StopTrip_RunningNoSibling(t *testing.T) {
	label := "LA-C8 R5-2 (d) stop in the trip tick, running instance, no sibling"
	run := func(fails int) *c8aRig {
		a := &stepHandler{id: "la_a", err: errLAHandler}
		return c8aStopTrip(t, laDefSingle(), 0, fails, false, &c5Undo{}, a)
	}
	ctl := run(0)
	if ctl.trip != nil {
		t.Fatalf("%s: no-fault stopped tick returned %v", label, ctl.trip)
	}
	base := ctl.drive(t, ctl.e)
	if !isTerminal(base.status) {
		t.Fatalf("%s: oracle broken: the no-fault control is %s after live ticks", label, base.status)
	}
	got := run(1)
	if !errors.Is(got.trip, errLAInjected) {
		t.Fatalf("%s: the injected WorkflowFailed failure did not surface through the stopped tick: %v", label, got.trip)
	}
	t.Logf("%s: after the stopped tick alone: status=%s", label, laInstance(t, got.s, got.iid).Status)
	out := got.drive(t, got.e)
	if out.status != base.status {
		t.Errorf("%s: status=%s after live ticks; the no-fault control ends %s", label, out.status, base.status)
	}
	laDiffPairs(t, label, base.pairs, out.pairs)
	c5DiffCalls(t, label, base.calls, out.calls)
}

// ── W6: restart residual (gated) ───────────────────────────────────────────────────────────────

// TestLA_C8a_W6_StageFailsInStoppedTick_RestartConverges (W6, EDR-013 case W6; gated): the stopped
// tick's route (3) WorkflowFailed fails, and so does the route-recovery stage's completion (two failing
// appends). The process then exits. A fresh engine on the same store ticks live. The instance is
// `waiting` with a committed route (3) and no further signal or timeout. SCAN_RUNNABLE lists running
// instances only, so nothing revisits it and the route stays uncompleted. Expected: the no-fault end
// state. Closes with EDR-013's startup sweep.
func TestLA_C8a_W6_StageFailsInStoppedTick_RestartConverges(t *testing.T) {
	if os.Getenv("LA_DEFERRED") != "1" {
		t.Skip("W6 (EDR-013 case W6, restart residual of R5-2): gated until EDR-013's startup sweep (run with LA_DEFERRED=1)")
	}
	label := "LA-C8 W6 restart after a stopped tick whose route completion also failed"
	ctl := c8aSibling(t, signalStep("w", "go"), 0, false)
	if ctl.trip != nil {
		t.Fatalf("%s: no-fault stopped tick returned %v", label, ctl.trip)
	}
	base := ctl.outcome(t)
	got := c8aSibling(t, signalStep("w", "go"), 2, false)
	if !errors.Is(got.trip, errLAInjected) {
		t.Fatalf("%s: the injected WorkflowFailed failures did not surface through the stopped tick: %v", label, got.trip)
	}
	// Precondition: both failing appends (the settle's and the route-recovery stage's) were reached
	// in the stopped tick, so the stage itself failed there.
	got.st.mu.Lock()
	reached := got.st.wfCalls
	got.st.mu.Unlock()
	if reached != 2 {
		t.Fatalf("%s: %d WorkflowFailed append(s) reached the store in the stopped tick; want 2 (settle and stage)", label, reached)
	}
	fresh := laEngine(got.st, got.clk, got.ids, 2, got.steps...)
	out := got.drive(t, fresh)
	c6SiblingCheck(t, label, base, out)
}

// ── R5-5 / LA-R7 (gated) ───────────────────────────────────────────────────────────────────────

// c8aBlockWF pauses the blockAt-th WorkflowFailed append (1-based) until release is closed, after
// signalling entered. When failFirst is set, the first WorkflowFailed append fails before its write.
type c8aBlockWF struct {
	*laFaultStorage
	blockAt   int
	failFirst bool
	entered   chan struct{}
	release   chan struct{}
	mu        sync.Mutex
	count     int
}

func (f *c8aBlockWF) AppendEvent(ctx context.Context, ev core.ExecutionEvent) error {
	if ev.EventType != core.EventTypeWorkflowFailed {
		return f.laFaultStorage.AppendEvent(ctx, ev)
	}
	f.mu.Lock()
	f.count++
	n := f.count
	f.mu.Unlock()
	if n == f.blockAt {
		close(f.entered)
		<-f.release
	}
	if n == 1 && f.failFirst {
		return laInjected(laOpAppendEvent)
	}
	return f.laFaultStorage.AppendEvent(ctx, ev)
}

// c8aCancelInWindow runs the sibling fixture to its trip tick, pauses that tick's blockAt-th
// WorkflowFailed append, runs Cancel while it is paused, then resumes the tick. The LA-R7 invariant:
// once Cancel has appended WorkflowCancelled, the tick appends no WorkflowFailed, WorkflowCompensating
// or WorkflowCompensated, and no undo handler runs after Cancel returned.
func c8aCancelInWindow(t *testing.T, label string, blockAt int, failFirst bool) {
	t.Helper()
	w := signalStep("w", "go")
	def := c6SiblingDef(w)
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	var u c5Undo
	rootH := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1H := &stepHandler{id: "la_b1", err: errLAHandler}
	handlers := []core.StepHandler{rootH, b1H, u.handler("undo-root"), u.handler("undo-w"), u.handler("undo-b1")}
	st := &c8aBlockWF{laFaultStorage: fs, blockAt: blockAt, failFirst: failFirst,
		entered: make(chan struct{}), release: make(chan struct{})}
	e := laEngine(st, clk, ids, 2, handlers...)
	iid := laSubmit(t, e, def.ID)
	ctx := context.Background()
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("%s: tick 1: %v", label, err)
	}

	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(st.release) }) }
	defer release()

	tickDone := make(chan error, 1)
	go func() { tickDone <- e.Tick(ctx) }()
	select {
	case <-st.entered:
	case <-time.After(laHang):
		t.Fatalf("%s: the tick never reached its paused WorkflowFailed append", label)
	}

	// Cancel runs while the tick is paused. Today it does not take tickMu, so it returns at once.
	cancelDone := make(chan error, 1)
	go func() { cancelDone <- e.Cancel(ctx, iid, "c8a-cancel", false) }()
	var cerr error
	inWindow := false
	select {
	case cerr = <-cancelDone:
		inWindow = true
	case <-time.After(laHang):
	}
	undoAtCancel := ""
	if inWindow {
		undoAtCancel = u.calls() // the tick is paused, so no undo ran after Cancel returned
	}
	release()
	select {
	case terr := <-tickDone:
		t.Logf("%s: tick returned: %v", label, terr)
	case <-time.After(laHang):
		t.Fatalf("%s: the tick did not return after the release", label)
	}
	if !inWindow {
		select {
		case cerr = <-cancelDone:
		case <-time.After(laHang):
			t.Fatalf("%s: Cancel did not return", label)
		}
		undoAtCancel = u.calls()
	}
	t.Logf("%s: Cancel returned while the tick was paused: %t (err=%v)", label, inWindow, cerr)

	evs := laEvents(t, s, iid)
	t.Logf("%s: final status=%s events: %s", label, laInstance(t, s, iid).Status, laSummary(t, evs))
	t.Logf("%s: undo invocations: at cancel=%q final=%q", label, undoAtCancel, u.calls())
	wcSeq := 0
	for _, ev := range evs {
		if ev.EventType == core.EventTypeWorkflowCancelled {
			wcSeq = ev.SequenceNum
		}
	}
	if wcSeq == 0 || cerr != nil {
		t.Fatalf("%s: Cancel did not append WorkflowCancelled (err=%v)", label, cerr)
	}
	if late := c8aLateTerminal(evs, wcSeq); len(late) > 0 {
		t.Errorf("%s: %d terminal-class event(s) appended after WorkflowCancelled (seq %d): %s",
			label, len(late), wcSeq, laSummary(t, late))
	}
	if got := u.calls(); got != undoAtCancel {
		t.Errorf("%s: undo invocations after Cancel returned: %q; want none (at cancel %q)", label, got, undoAtCancel)
	}
}

// TestLA_C8a_R55_CancelDuringStageWFAppend (R5-5 (i); gated). LA-R7 (P2, pre-existing since v1.0.0):
// Cancel is not serialized with the tick's terminal appends; fix designed separately (Cancel under
// tickMu deadlocks handler-called Cancel). Here the settle's route (3) WorkflowFailed fails before its
// write, so the route-recovery stage's WorkflowFailed (the second append) is the paused one.
func TestLA_C8a_R55_CancelDuringStageWFAppend(t *testing.T) {
	if os.Getenv("LA_DEFERRED") != "1" {
		t.Skip("LA-R7 gated reproducer (P2, pre-existing since v1.0.0; fix designed separately): run with LA_DEFERRED=1")
	}
	c8aCancelInWindow(t, "LA-R7 (R5-5 i) Cancel during the route-recovery stage's WorkflowFailed append", 2, true)
}

// TestLA_C8a_R55_CancelDuringSettleWFAppend_NoFault (R5-5 (ii); gated): no fault. The settle's own
// route (3) WorkflowFailed (the first append) is the paused one. LA-R7 (P2, pre-existing since v1.0.0):
// Cancel is not serialized with the tick's terminal appends; fix designed separately (Cancel under
// tickMu deadlocks handler-called Cancel).
func TestLA_C8a_R55_CancelDuringSettleWFAppend_NoFault(t *testing.T) {
	if os.Getenv("LA_DEFERRED") != "1" {
		t.Skip("LA-R7 gated reproducer (P2, pre-existing since v1.0.0; fix designed separately): run with LA_DEFERRED=1")
	}
	c8aCancelInWindow(t, "LA-R7 (R5-5 ii) Cancel during the settle's own WorkflowFailed append, no fault", 1, false)
}
