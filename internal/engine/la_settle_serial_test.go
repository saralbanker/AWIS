package engine

// LA-C4b regression tests (independent review, tick.go):
//
//   F-S  every dispatched item is settled, even after an earlier item's settle
//        fails; a durable terminal workflow event still stops settling.
//   F-M  tick holds e.tickMu for its whole body, so ticks never overlap.
//
// Fault model is the one used by la_stranded_repro_test.go: a fault is ONE
// transient failure of the nth matching storage call; the failed call commits
// nothing (except where a test states otherwise), and every later call is forwarded.

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// c4bHas reports whether evs holds an event of type et for stepID.
func c4bHas(evs []core.ExecutionEvent, et core.EventType, stepID string) bool {
	return c4bCount(evs, et, stepID) > 0
}

// c4bCount counts events of type et for stepID; stepID "" matches any step.
func c4bCount(evs []core.ExecutionEvent, et core.EventType, stepID string) int {
	n := 0
	for _, ev := range evs {
		if ev.EventType == et && (stepID == "" || ev.StepID == stepID) {
			n++
		}
	}
	return n
}

// c4bSlow is a native handler that holds its dispatch briefly before running, so a
// concurrent tick can overlap a live dispatch. The hold is a bounded delay, not a
// synchronization point: every assertion in this file is state-based.
type c4bSlow struct{ *stepHandler }

func (h c4bSlow) Execute(sc core.StepContext) (core.StepResult, error) {
	<-time.After(time.Millisecond)
	return h.stepHandler.Execute(sc)
}

// c4bCancelFault fails every every-th CancellationRequested read (transient and
// repeating; 0 disables it). processInstance makes this read at its top gate (before
// any gather) and again post-settle, and nothing that writes an event or a projection
// row makes it. An injected failure therefore takes exactly the LA-C1 F1 path
// (processInstance error, then resetInstanceRecovery) and cannot strand a
// StepStarted of its own.
type c4bCancelFault struct {
	*laFaultStorage
	every atomic.Int64
	calls atomic.Int64
}

func (f *c4bCancelFault) CancellationRequested(ctx context.Context, id core.InstanceID) (bool, error) {
	if n := f.every.Load(); n > 0 && f.calls.Add(1)%n == 0 {
		return false, laInjected(laOp("CancellationRequested"))
	}
	return f.laFaultStorage.CancellationRequested(ctx, id)
}

// TestLA_C4b_SettleFailureDoesNotAbandonDispatchedSibling (F-S, PRD FR-RM-04 spirit;
// LA-C1 F2). b1 and b2 are dispatched in the same tick (MaxParallelSteps=2), so both
// handlers run before settle. One transient failure hits b1's StepCompleted append.
// Before LA-C4b, the settle loop returned at b1, so b2's outcome was never recorded
// and RC-1 recovered the successful b2 as worker_crash.
func TestLA_C4b_SettleFailureDoesNotAbandonDispatchedSibling(t *testing.T) {
	def := laDefFanOut()
	s, fs := laSetup(t, def)
	root := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1 := &stepHandler{id: "la_b1", outputs: map[string]any{"who": "b1"}}
	b2 := &stepHandler{id: "la_b2", outputs: map[string]any{"who": "b2"}}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	e := laEngine(fs, clk, laIDs(), 2, root, b1, b2)
	iid := laSubmit(t, e, def.ID)

	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepCompleted, stepID: "b1"})
	laWantInjected(t, laTickUntilTripped(t, e, fs))

	// Same tick: b2's handler ran exactly once and its StepCompleted is durable.
	evs := laEvents(t, s, iid)
	if !c4bHas(evs, core.EventTypeStepCompleted, "b2") {
		t.Errorf("same tick: b2 handler ran (calls=%d) but StepCompleted(b2) was not persisted", laCalls(b2))
	}
	if got := laCalls(b2); got != 1 {
		t.Errorf("b2 handler calls=%d; want 1", got)
	}

	// b1's outcome was never persisted, so RC-1 settles it as worker_crash; the
	// handler is not re-run.
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	inst := laInstance(t, s, iid)
	if !laIsTerminal(inst) {
		t.Fatalf("not terminal after %d further ticks (K=%d): status=%s current_steps=%v", n, laK, inst.Status, inst.CurrentSteps)
	}
	if inst.Status != core.InstanceStatusFailed {
		t.Errorf("status=%s; want failed (b1 worker_crash is a terminal failure of a final step)", inst.Status)
	}
	t.Logf("recovery: ticks=%d tick_errs=%d status=%s", n, len(errs), inst.Status)
	evs = laEvents(t, s, iid)
	if got := laStepOutcome(t, evs, "b1"); !strings.HasPrefix(got, "StepFailed(code=worker_crash,") {
		t.Errorf("b1 outcome=%s; want RC-1 worker_crash", got)
	}
	if got := laStepOutcome(t, evs, "b2"); got != "StepCompleted" {
		t.Errorf("b2 outcome=%s; want StepCompleted (a successful sibling must not be failed)", got)
	}
	if got := laCalls(b1); got != 1 {
		t.Errorf("b1 handler calls=%d; want 1", got)
	}
	if got := laCalls(b2); got != 1 {
		t.Errorf("b2 handler calls=%d after recovery; want 1", got)
	}
}

// TestLA_C4b_DurableTerminalFromSettleErrorStopsSettling pins the terminal stop (the
// LA-C4b coordinator rule). b1 fails terminally, so WorkflowFailed is appended
// durably. Its projection write (UpsertInstance status=failed) then fails, so the
// settle error comes AFTER a durable terminal event. Settling must stop: b2 is not
// settled and no event is appended after WorkflowFailed in this tick. The post-
// terminal recovery side is LA-C4a's hydrate fence and is not asserted here.
func TestLA_C4b_DurableTerminalFromSettleErrorStopsSettling(t *testing.T) {
	def := laDefFanOut()
	s, fs := laSetup(t, def)
	root := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1 := &stepHandler{id: "la_b1", err: errLAHandler}
	b2 := &stepHandler{id: "la_b2", outputs: map[string]any{"who": "b2"}}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	e := laEngine(fs, clk, laIDs(), 2, root, b1, b2)
	iid := laSubmit(t, e, def.ID)

	fs.arm(laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusFailed})
	laWantInjected(t, laTickUntilTripped(t, e, fs))

	evs := laEvents(t, s, iid)
	wf := -1
	for i, ev := range evs {
		if ev.EventType == core.EventTypeWorkflowFailed {
			wf = i
		}
	}
	if wf < 0 {
		t.Fatalf("WorkflowFailed was not made durable by the trip tick")
	}
	if got := c4bCount(evs, core.EventTypeWorkflowFailed, ""); got != 1 {
		t.Errorf("WorkflowFailed events=%d; want 1", got)
	}
	if extra := len(evs) - 1 - wf; extra != 0 {
		t.Errorf("%d event(s) appended after durable WorkflowFailed; want 0", extra)
	}
	if c4bHas(evs, core.EventTypeStepCompleted, "b2") {
		t.Errorf("StepCompleted(b2) persisted after a durable terminal event; settling must stop")
	}
}

// TestLA_C4b_ConcurrentTicksAreSerialized (F-M). Several goroutines call Tick on the
// same running instance while a transient CancellationRequested fault makes some
// ticks fail. Each failure takes the reset path while other ticks may be dispatching.
// Without tickMu, a reset can let a later tick's RC-1 record a live step as
// worker_crash, or let a step run twice. With tickMu, every step must complete
// exactly once with no StepFailed persisted.
//
// Measured: on the pre-LA-C4b code this failed in 20 of 20 runs (count=20). It is
// still a stress test, not a deterministic proof, because overlap is scheduling-
// dependent. A single-ticker run of the same workload passes on the pre-fix code,
// so the failure is attributable to overlapping ticks (F-M), not to the injected
// fault itself.
func TestLA_C4b_ConcurrentTicksAreSerialized(t *testing.T) {
	const (
		iterations = 20
		tickers    = 4
		ticksEach  = 12
		faultEvery = 2
	)
	for iter := 0; iter < iterations; iter++ {
		def := laDefFanOut()
		s, fs := laSetup(t, def)
		cf := &c4bCancelFault{laFaultStorage: fs}
		root := &stepHandler{id: "la_root", outputs: map[string]any{}}
		b1 := c4bSlow{&stepHandler{id: "la_b1", outputs: map[string]any{"who": "b1"}}}
		b2 := c4bSlow{&stepHandler{id: "la_b2", outputs: map[string]any{"who": "b2"}}}
		clk := newFakeClock(engineStart, 10*time.Millisecond)
		e := laEngine(cf, clk, laIDs(), 2, root, b1, b2)
		iid := laSubmit(t, e, def.ID)

		cf.every.Store(faultEvery)
		var (
			wg       sync.WaitGroup
			injected atomic.Int64
		)
		for g := 0; g < tickers; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for n := 0; n < ticksEach; n++ {
					err := e.Tick(context.Background())
					switch {
					case err == nil:
					case errors.Is(err, errLAInjected):
						injected.Add(1)
					default:
						t.Errorf("iter %d: unexpected tick error: %v", iter, err)
					}
				}
			}()
		}
		wg.Wait()
		cf.every.Store(0)
		if injected.Load() == 0 {
			t.Errorf("iter %d: no injected failure reached a tick; reset path not exercised", iter)
		}

		// Drain sequentially with faults off until the instance is terminal.
		_, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
		for _, err := range errs {
			t.Errorf("iter %d: drain tick error: %v", iter, err)
		}
		inst := laInstance(t, s, iid)
		if inst.Status != core.InstanceStatusCompleted {
			t.Errorf("iter %d: status=%s; want completed", iter, inst.Status)
		}

		evs := laEvents(t, s, iid)
		if got := c4bCount(evs, core.EventTypeStepFailed, ""); got != 0 {
			t.Errorf("iter %d: %d StepFailed persisted; want 0 (no live step may be settled worker_crash)", iter, got)
		}
		for _, step := range []string{"root", "b1", "b2"} {
			if got := c4bCount(evs, core.EventTypeStepStarted, step); got != 1 {
				t.Errorf("iter %d: StepStarted(%s)=%d; want exactly 1 dispatch", iter, step, got)
			}
			if got := c4bCount(evs, core.EventTypeStepCompleted, step); got != 1 {
				t.Errorf("iter %d: StepCompleted(%s)=%d; want 1", iter, step, got)
			}
		}
		for name, h := range map[string]*stepHandler{"root": root, "b1": b1.stepHandler, "b2": b2.stepHandler} {
			if got := laCalls(h); got != 1 {
				t.Errorf("iter %d: %s handler calls=%d; want 1 (no handler may run twice)", iter, name, got)
			}
		}
	}
}
