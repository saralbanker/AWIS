package engine

// LA-C7 HIGH-1 and HIGH-2 (independent review round 4 of LA-C6 R3-M1). TEST-ONLY file: a
// WorkflowFailed append that is ack-lost (it commits, then returns an error; emit.go appends before
// it projects) leaves the projection `waiting` behind a durable terminal event. The fixtures are
// la_c7_test.go's (c7Run, c7Case, c7Storage). The oracle is the no-fault control of the same fixture.
//
// Each row runs in the same process and after a restart (the _AfterRestart variants), because the
// route-recovery record is in memory and a restart must not be the only thing that completes it.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// c7AckLostWF arms the first WorkflowFailed append to commit and then fail (laModelAckLost).
func c7AckLostWF(fs *laFaultStorage) core.StoragePort {
	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowFailed, model: laModelAckLost})
	return fs
}

// c7CompletionAckLostWF: the live route (3) WorkflowFailed of b1 fails before the write, so the
// route-recovery stage completes the route; that completion's own WorkflowFailed (the next append)
// commits and then fails. The stage, not the settle loop, is what ack-loses the terminal event.
func c7CompletionAckLostWF(fs *laFaultStorage) core.StoragePort {
	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowFailed, model: laModelAckLost})
	return &c7Storage{laFaultStorage: fs, wfFail: 1}
}

// ── HIGH-1: the signal sibling and the timeout siblings ─────────────────────────────────────────

// TestLA_C7_HIGH1_SignalSibling_AckLostWF (FAIL-before at e8ef604: the sibling's "go" is delivered
// after b1's durable decision, so SignalReceived is logged and the run diverges).
func TestLA_C7_HIGH1_SignalSibling_AckLostWF(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-1 signal sibling, WorkflowFailed ack-lost", signalStep("w", "go"), true, c7AckLostWF, false)
}

func TestLA_C7_HIGH1_SignalSibling_AckLostWF_AfterRestart(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-1 signal sibling, WorkflowFailed ack-lost, after restart", signalStep("w", "go"), true, c7AckLostWF, true)
}

// TestLA_C7_HIGH1_TimeoutContinueSibling_AckLostWF (FAIL-before at e8ef604: the continue timeout
// completes w, so StepCompleted(w) is logged).
func TestLA_C7_HIGH1_TimeoutContinueSibling_AckLostWF(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-1 timeout(continue) sibling, WorkflowFailed ack-lost",
		signalStepTimeout("w", "go", "1ms", "continue"), false, c7AckLostWF, false)
}

func TestLA_C7_HIGH1_TimeoutContinueSibling_AckLostWF_AfterRestart(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-1 timeout(continue) sibling, WorkflowFailed ack-lost, after restart",
		signalStepTimeout("w", "go", "1ms", "continue"), false, c7AckLostWF, true)
}

// TestLA_C7_HIGH1_TimeoutFailSibling_AckLostWF (FAIL-before at e8ef604: the fail timeout appends a
// second WorkflowFailed for w, so the log holds two).
func TestLA_C7_HIGH1_TimeoutFailSibling_AckLostWF(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-1 timeout(fail) sibling, WorkflowFailed ack-lost",
		signalStepTimeout("w", "go", "1ms", "fail"), false, c7AckLostWF, false)
}

func TestLA_C7_HIGH1_TimeoutFailSibling_AckLostWF_AfterRestart(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-1 timeout(fail) sibling, WorkflowFailed ack-lost, after restart",
		signalStepTimeout("w", "go", "1ms", "fail"), false, c7AckLostWF, true)
}

// ── HIGH-2: no signal at all ────────────────────────────────────────────────────────────────────

// TestLA_C7_HIGH2_NoSignal_AckLostWF (FAIL-before at e8ef604: the instance stays `waiting` with no
// compensation, because nothing else reads the instance). The no-fault outcome is failed and
// compensated, with undo-root only.
func TestLA_C7_HIGH2_NoSignal_AckLostWF(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-2 no signal, WorkflowFailed ack-lost", signalStep("w", "go"), false, c7AckLostWF, false)
}

func TestLA_C7_HIGH2_NoSignal_AckLostWF_AfterRestart(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-2 no signal, WorkflowFailed ack-lost, after restart", signalStep("w", "go"), false, c7AckLostWF, true)
}

// ── Completion-path ack-lost WorkflowFailed (the route-recovery stage's own append) ────────────

// TestLA_C7_HIGH1_CompletionPath_AckLostWF_SignalSibling (FAIL-before at e8ef604: the stage's
// completion ack-loses WorkflowFailed, its error is returned, and the signal is then delivered).
func TestLA_C7_HIGH1_CompletionPath_AckLostWF_SignalSibling(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-1 completion-path WorkflowFailed ack-lost, signal sibling",
		signalStep("w", "go"), true, c7CompletionAckLostWF, false)
}

func TestLA_C7_HIGH1_CompletionPath_AckLostWF_SignalSibling_AfterRestart(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-1 completion-path WorkflowFailed ack-lost, signal sibling, after restart",
		signalStep("w", "go"), true, c7CompletionAckLostWF, true)
}

// TestLA_C7_HIGH2_CompletionPath_AckLostWF_NoSignal (FAIL-before at e8ef604: stays `waiting`).
func TestLA_C7_HIGH2_CompletionPath_AckLostWF_NoSignal(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-2 completion-path WorkflowFailed ack-lost, no signal",
		signalStep("w", "go"), false, c7CompletionAckLostWF, false)
}

func TestLA_C7_HIGH2_CompletionPath_AckLostWF_NoSignal_AfterRestart(t *testing.T) {
	c7Case(t, "LA-C7 HIGH-2 completion-path WorkflowFailed ack-lost, no signal, after restart",
		signalStep("w", "go"), false, c7CompletionAckLostWF, true)
}

// ── Gate: a durable terminal event whose projection is behind refuses delivery ─────────────────

// TestLA_C7_GateRefusesBehindTerminal (FAIL-before at e8ef604: the gate reads the logged
// WorkflowFailed as "not awaiting" and Resolve delivers the parked signal). The stage's first
// GetInstance after the ack-lost append fails, so the trip tick leaves the projection `waiting`. A
// fresh engine (the restart shape, with no in-memory record) must refuse the delivery, because the
// log holds a terminal event that the projection has not absorbed. The run then reaches the no-fault
// outcome.
func TestLA_C7_GateRefusesBehindTerminal(t *testing.T) {
	w := signalStep("w", "go")
	base, baseTrip := c7Run(t, w, true, c7Plain, false)
	if baseTrip != nil {
		t.Fatalf("no-fault trip tick returned %v", baseTrip)
	}

	def := c6SiblingDef(w)
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	var u c5Undo
	rootH := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1H := &stepHandler{id: "la_b1", err: errLAHandler}
	handlers := []core.StepHandler{rootH, b1H, u.handler("undo-root"), u.handler("undo-w"), u.handler("undo-b1")}
	st := &c7Storage{laFaultStorage: fs, getFailAfterWF: true}
	c7AckLostWF(fs)
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
		t.Fatalf("trip tick: the injected failures did not surface: %v", trip)
	}
	if got := laInstance(t, s, iid).Status; got != core.InstanceStatusWaiting {
		t.Fatalf("setup: status=%s; want waiting (WorkflowFailed durable, projection behind)", got)
	}

	fresh := laEngine(st, clk, ids, 2, handlers...)
	if _, _, ok := fresh.Resolve(ctx, iid, "go"); ok {
		t.Errorf("a durable WorkflowFailed whose projection is behind: Resolve delivered; it must refuse")
	}
	if n := c5Count(laEvents(t, s, iid), core.EventTypeSignalReceived); n != 0 {
		t.Errorf("%d SignalReceived event(s) after the refused Resolve; want 0", n)
	}
	_, more := laTickUntil(t, fresh, s, iid, laK, laIsTerminal)
	got := c5Outcome{
		s: s, iid: iid, status: laInstance(t, s, iid).Status, evs: laEvents(t, s, iid),
		calls: map[string]int32{"la_root": laCalls(rootH), "la_b1": laCalls(b1H)},
		errs:  len(more), undos: u.calls(),
	}
	got.pairs, _ = eventPairs(t, s, iid)
	c6SiblingCheck(t, "LA-C7 gate, ack-lost WorkflowFailed, projection behind", base, got)
}
