package engine

// LA-C7 (independent review round 4 of LA-C6 R3-M1). TEST-ONLY file: the shared fixture and the
// HIGH-3 test (route_recovery.go: a read error in the route-recovery stage is retried on a later
// tick). The HIGH-1/2 and gate tests are in la_c7_terminal_test.go; the LOW-2 and LOW-3 tests are in
// la_c7_route3_test.go.
//
// Oracle: the NO-FAULT control of the same fixture (c7Run with no fault), as in la_c6_r3m1_test.go.
// A faulted run must equal it on final status, the (event_type, step_id) sequence, handler and undo
// invocations; it must contain no SignalReceived and exactly one WorkflowFailed attributed to b1
// (c6SiblingCheck).

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// c7Storage wraps laFaultStorage with two LA-C7 faults. The next wfFail WorkflowFailed appends fail
// before the write (nothing commits). When getFailAfterWF is set, the first GetInstance call made
// after a WorkflowFailed append has returned an error fails once. The ack-lost WorkflowFailed fault
// is armed on the wrapped laFaultStorage (laModelAckLost); c7Storage observes its error the same way.
type c7Storage struct {
	*laFaultStorage
	mu             sync.Mutex
	wfFail         int
	getFailAfterWF bool
	wfErrored      bool // a WorkflowFailed append has returned an error
	getTripped     bool // the GetInstance fault has fired (once)
}

func (f *c7Storage) AppendEvent(ctx context.Context, ev core.ExecutionEvent) error {
	if ev.EventType != core.EventTypeWorkflowFailed {
		return f.laFaultStorage.AppendEvent(ctx, ev)
	}
	f.mu.Lock()
	hit := f.wfFail > 0
	if hit {
		f.wfFail--
	}
	f.mu.Unlock()
	var err error
	if hit {
		err = laInjected(laOpAppendEvent)
	} else {
		err = f.laFaultStorage.AppendEvent(ctx, ev)
	}
	if err != nil {
		f.mu.Lock()
		f.wfErrored = true
		f.mu.Unlock()
	}
	return err
}

func (f *c7Storage) GetInstance(ctx context.Context, iid core.InstanceID) (core.WorkflowInstance, error) {
	f.mu.Lock()
	fail := f.getFailAfterWF && f.wfErrored && !f.getTripped
	if fail {
		f.getTripped = true
	}
	f.mu.Unlock()
	if fail {
		return core.WorkflowInstance{}, laInjected(laOp("GetInstance"))
	}
	return f.laFaultStorage.GetInstance(ctx, iid)
}

// c7Plain is the no-fault store: the laFaultStorage itself.
func c7Plain(fs *laFaultStorage) core.StoragePort { return fs }

// c7Run drives the c6SiblingDef fixture as c6SiblingRun does: tick 1 completes root; the signal
// "go" is pre-sent when withSignal; the trip tick gathers w and b1 (b1 fails with a route (3)
// decision); restart replaces the engine on the same store after the trip tick. mk builds the store
// the engine uses over fs and may arm fs. It returns the outcome and the trip tick's error.
func c7Run(t *testing.T, w core.Step, withSignal bool, mk func(fs *laFaultStorage) core.StoragePort, restart bool) (c5Outcome, error) {
	t.Helper()
	def := c6SiblingDef(w)
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	var u c5Undo
	rootH := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1H := &stepHandler{id: "la_b1", err: errLAHandler}
	handlers := []core.StepHandler{rootH, b1H, u.handler("undo-root"), u.handler("undo-w"), u.handler("undo-b1")}
	st := mk(fs)
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
	trip := e.Tick(ctx) // the trip tick
	if restart {
		e = laEngine(st, clk, ids, 2, handlers...)
	}
	_, more := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	out := c5Outcome{
		s:      s,
		iid:    iid,
		status: laInstance(t, s, iid).Status,
		evs:    laEvents(t, s, iid),
		calls:  map[string]int32{"la_root": laCalls(rootH), "la_b1": laCalls(b1H)},
		errs:   len(more),
		undos:  u.calls(),
	}
	if trip != nil {
		out.errs++
	}
	out.pairs, _ = eventPairs(t, s, iid)
	return out, trip
}

// c7Case runs the faulted fixture and asserts that it equals the no-fault control. The injected
// failure must surface through the trip tick, so the run is the scenario the label names. The
// instance must reach a terminal status within laK ticks (laTickUntil's bound).
func c7Case(t *testing.T, label string, w core.Step, withSignal bool, mk func(*laFaultStorage) core.StoragePort, restart bool) {
	t.Helper()
	base, baseTrip := c7Run(t, w, withSignal, c7Plain, false)
	if baseTrip != nil {
		t.Fatalf("%s: no-fault trip tick returned %v", label, baseTrip)
	}
	got, trip := c7Run(t, w, withSignal, mk, restart)
	if !errors.Is(trip, errLAInjected) {
		t.Fatalf("%s: the injected failure did not surface through the trip tick: %v", label, trip)
	}
	if !isTerminal(got.status) {
		t.Errorf("%s: status=%s after K=%d ticks; the instance must reach a terminal status", label, got.status, laK)
	}
	c6SiblingCheck(t, label, base, got)
}

// ── HIGH-3: a read error in the route-recovery stage is retried ─────────────────────────────────

// TestLA_C7_HIGH3_StageReadErrorRetried (LA-C7 HIGH-3; FAIL-before at e8ef604): the WorkflowFailed
// append of b1's route (3) fails before the write, and the route-recovery stage's next GetInstance
// fails once. No signal is ever sent, so nothing else touches the instance: the stage must keep the
// instance recorded on a read error (completeRouteRecovery) so the next tick completes route (3).
// Before the fix the instance stayed `waiting` with no compensation.
func TestLA_C7_HIGH3_StageReadErrorRetried(t *testing.T) {
	var st *c7Storage
	c7Case(t, "LA-C7 HIGH-3 stage read error after WorkflowFailed fail-before, no signal",
		signalStep("w", "go"), false,
		func(fs *laFaultStorage) core.StoragePort {
			st = &c7Storage{laFaultStorage: fs, wfFail: 1, getFailAfterWF: true}
			return st
		}, false)
	// LA-C8 R5-4(c): the stage read fault must actually have fired. Without it the run is the
	// no-fault path and the test proves nothing about a read error in the route-recovery stage.
	st.mu.Lock()
	tripped := st.getTripped
	st.mu.Unlock()
	if !tripped {
		t.Errorf("LA-C7 HIGH-3: the injected stage read fault (GetInstance after the WorkflowFailed fail-before) never fired")
	}
}
