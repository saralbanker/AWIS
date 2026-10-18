package engine

// LA-C6 (PR #4 review round 3 remediation): R3-H1 (settle side), R3-M1, R3-M2, R3-L1 and the
// LA-R6 backlog reproducer. TEST-ONLY file.
//
// Oracle for every test here: the NO-FAULT run of the SAME fixture (a faulted run must equal it
// on final status, the (event_type, step_id) sequence, handler and undo invocations), unless a
// test states otherwise. Fault models are documented on laFaultStorage (la_stranded_repro_test.go).

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/awis/awis/internal/core"
)

// ── R3-H1 (settle side): the route of a conditional on_error is decided at settle time ────
//
// b1 has an on_error edge to oe whose condition is FALSE (root's flag is "no"), so b1's live
// route is (3): no target fires. Its StepFailed{retrying:false} append is ack-lost. Before LA-C6
// the settle loop classified b1 statically (it has an on_error transition), read the lost append
// as non-terminal, and settled the sibling b2 after b1's committed decision: the faulted run
// ended `completed`. The settle-side check now uses the settle-time route (see tick.go
// committedRouteThree).
//
// Recovery side, NOT fixed here (LA-C6 scope): completeCommittedRoutes (hydrate.go, LA-C3) still
// classifies statically, so it never completes a step with an on_error edge. The committed route
// (3) of b1 is therefore not completed by recovery. Recorded divergence, observed today: b2 was
// never settled in the trip tick; the next tick's RC-1 settles b2 as worker_crash (its StepStarted
// is stranded, the sibling's StepStarted has no decision in the log), which routes a
// WorkflowFailed attributed to b2, not b1. The STATE.md expectation "SEC-08 stalled, no
// compensation" was NOT what the code produced; the observed end is logged by the test below.
// Full no-fault equivalence needs LA-D2 (EDR-012, founder direction A approved, implementation
// pending); TestLA_C6_R3H1_ConditionalRoute3_FullNoFaultEquivalence is gated for it.

// c6R3H1Def is laDefFanOut (root → {b1, b2}) plus an on_error edge b1 → oe whose condition is
// false at settle time, with a compensation plan over root, b1 and b2.
func c6R3H1Def() core.WorkflowDefinition {
	def := laDefFanOut()
	def.Steps = append(def.Steps, nativeStep("oe", "la_oe"))
	def.Transitions = append(def.Transitions, core.Transition{
		From: "b1", To: "oe", OnError: true, Condition: "steps.root.outputs.flag == 'yes'",
	})
	def.Compensation = &core.CompensationPlan{Steps: []core.CompensationStep{
		{StepID: "root", UndoHandler: "undo-root"},
		{StepID: "b1", UndoHandler: "undo-b1"},
		{StepID: "b2", UndoHandler: "undo-b2"},
	}}
	return def
}

// c6R3H1Run executes c6R3H1Def; ft == nil is the no-fault oracle.
func c6R3H1Run(t *testing.T, restart bool, ft *laFault) c5Outcome {
	t.Helper()
	var u c5Undo
	handlers := []core.StepHandler{
		&stepHandler{id: "la_root", outputs: map[string]any{"flag": "no"}},
		&stepHandler{id: "la_b1", err: errLAHandler},
		&stepHandler{id: "la_b2", outputs: map[string]any{"who": "b2"}},
		&stepHandler{id: "la_oe", outputs: map[string]any{}},
		u.handler("undo-root"), u.handler("undo-b1"), u.handler("undo-b2"),
	}
	out := c5Run(t, c6R3H1Def(), 2, ft, restart, handlers...)
	out.undos = u.calls()
	return out
}

// TestLA_C6_R3H1_ConditionalRoute3_AckLostStepFailed_SettleStops (FAIL-before: 248b7c2 tick.go
// settles b2 after b1's committed decision and the run ends completed).
func TestLA_C6_R3H1_ConditionalRoute3_AckLostStepFailed_SettleStops(t *testing.T) {
	c6R3H1SettleStops(t, false)
}

func TestLA_C6_R3H1_ConditionalRoute3_AckLostStepFailed_SettleStops_AfterRestart(t *testing.T) {
	c6R3H1SettleStops(t, true)
}

func c6R3H1SettleStops(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C6 R3-H1 conditional on_error (false) route 3, StepFailed ack-lost", restart)
	ft := laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepFailed, stepID: "b1", model: laModelAckLost}
	got := c6R3H1Run(t, restart, &ft)

	// The fault must have committed the decision (ack-lost: the write is durable).
	if !c5HasEventType(got.atTrip, core.EventTypeStepFailed, "b1") {
		t.Fatalf("%s: no durable StepFailed(b1) in the trip tick; the ack-lost fault did not commit", label)
	}
	// Settle-stop: b1's committed decision ends the settle loop in the trip tick, so the sibling
	// b2 is neither completed nor failed in that tick.
	if n := c5RoutingCount(got.atTrip, "b2"); n != 0 {
		t.Errorf("%s: %d StepFailed/StepCompleted event(s) for b2 in the trip tick; b1's committed decision must stop settling", label, n)
	}
	if _, ok := got.tripVars["b2"]; ok {
		t.Errorf("%s: b2 entered Variables in the trip tick (its StepCompleted was settled after b1's committed decision)", label)
	}
	// The instance must never end completed: the no-fault run ends failed with WorkflowFailed(b1).
	if got.status == core.InstanceStatusCompleted {
		t.Errorf("%s: instance ended completed; the no-fault run ends %s", label, c6NoFaultStatus(t))
	}
	// Recorded, not asserted (see the file comment): the end state the recovery side produces today.
	t.Logf("%s: observed end status=%s WorkflowFailed=%+v undos=%q", label, got.status,
		laWorkflowFailedPayloads(t, got.evs), got.undos)
}

// c6NoFaultStatus returns the no-fault status of the R3-H1 fixture, for failure messages.
func c6NoFaultStatus(t *testing.T) core.InstanceStatus {
	t.Helper()
	return c6R3H1Run(t, false, nil).status
}

// TestLA_C6_R3H1_ConditionalRoute3_FullNoFaultEquivalence is the SECOND assertion of R3-H1: the
// faulted run equals the no-fault run (WorkflowFailed(b1) + compensation). Gated: recovery of a
// condition-dependent route needs LA-D2.
func TestLA_C6_R3H1_ConditionalRoute3_FullNoFaultEquivalence(t *testing.T) {
	c6R3H1FullEquivalence(t, false)
}

func TestLA_C6_R3H1_ConditionalRoute3_FullNoFaultEquivalence_AfterRestart(t *testing.T) {
	c6R3H1FullEquivalence(t, true)
}

func c6R3H1FullEquivalence(t *testing.T, restart bool) {
	t.Helper()
	if os.Getenv("LA_DEFERRED") != "1" {
		t.Skip("LA-D2 pending (founder-approved direction A, implementation not yet authorized)")
	}
	label := laLabel("LA-C6 R3-H1 full no-fault equivalence", restart)
	base := c6R3H1Run(t, false, nil)
	ft := laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepFailed, stepID: "b1", model: laModelAckLost}
	got := c6R3H1Run(t, restart, &ft)
	if got.status != base.status {
		t.Errorf("%s: status=%s; the no-fault run ends %s", label, got.status, base.status)
	}
	laDiffPairs(t, label, base.pairs, got.pairs)
	c5DiffCalls(t, label, base.calls, got.calls)
	if got.undos != base.undos {
		t.Errorf("%s: undo invocations=%q; the no-fault run undoes %q", label, got.undos, base.undos)
	}
}

// ── shared fault wrappers (LA-C6) ──────────────────────────────────────────────────────

// c6WFFault fails the next `remaining` WorkflowFailed appends (fail-before: nothing commits),
// then forwards every call. It wraps laFaultStorage, so every engine-reached interface stays
// promoted.
type c6WFFault struct {
	*laFaultStorage
	mu        sync.Mutex
	remaining int
}

func newC6WFFault(fs *laFaultStorage, n int) *c6WFFault {
	return &c6WFFault{laFaultStorage: fs, remaining: n}
}

func (f *c6WFFault) AppendEvent(ctx context.Context, ev core.ExecutionEvent) error {
	if ev.EventType == core.EventTypeWorkflowFailed {
		f.mu.Lock()
		hit := f.remaining > 0
		if hit {
			f.remaining--
		}
		f.mu.Unlock()
		if hit {
			return laInjected(laOpAppendEvent)
		}
	}
	return f.laFaultStorage.AppendEvent(ctx, ev)
}

// TestLA_C6_R3H1_CommittedRouteThreeClassifiesWithSettleEnv pins the settle-side rule of
// committedRouteThree: the same on_error condition is route (2) or route (3) depending on the
// settle-time environment, exactly as routeTerminalFailure decides it for that instance.
func TestLA_C6_R3H1_CommittedRouteThreeClassifiesWithSettleEnv(t *testing.T) {
	dv, err := buildDefView(c6R3H1Def())
	if err != nil {
		t.Fatalf("buildDefView: %v", err)
	}
	b1 := dv.steps["b1"]
	evs := []core.ExecutionEvent{
		c5Ev(2, core.EventTypeStepStarted, "b1", false, "", ""),
		c5Ev(3, core.EventTypeStepFailed, "b1", false, "handler_error", ""),
	}
	cases := []struct {
		name string
		flag string
		want bool // committed route (3)
	}{
		{"condition false at settle time: no target fires, route (3)", "no", true},
		{"condition true at settle time: the on_error target fires, route (2)", "yes", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inst := core.WorkflowInstance{InstanceID: "iid",
				Variables: map[string]any{"root": map[string]any{"flag": c.flag}}}
			got, err := committedRouteThree(dv, inst, b1, evs)
			if err != nil {
				t.Fatalf("committedRouteThree: %v", err)
			}
			if got != c.want {
				t.Errorf("committedRouteThree=%t; want %t", got, c.want)
			}
			// The rule must agree with the live computation it mirrors.
			if fires := len(firingOnError(dv, "b1", buildEnv(inst))) == 0; fires != c.want {
				t.Errorf("firingOnError disagrees: route (3)=%t; want %t", fires, c.want)
			}
		})
	}
}
