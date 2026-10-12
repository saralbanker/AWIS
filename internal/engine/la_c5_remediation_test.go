package engine

// LA-C5 remediation (second independent review of Loop A): findings N1, N2, N3 and the
// ack-lost fault model (Q3). TEST-ONLY file except where a test names production code.
//
// Oracle for every N-test: the NO-FAULT run of the SAME fixture. A faulted run must equal
// it on final status, the (event_type, step_id) sequence, and handler / undo invocation
// counts. The expected outcome is taken from the no-fault path, never from the current code.
//
// Fault models (see laFaultStorage in la_stranded_repro_test.go):
//   - laModelFailBeforeWrite (default): the nth matching call fails and nothing commits.
//   - laModelAckLost: the nth matching call commits for real, then the caller sees an error.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// c5Undo records undo-handler invocations, in call order, across every engine of one run.
type c5Undo struct {
	mu    sync.Mutex
	order []string
}

func (u *c5Undo) handler(id string) core.StepHandler {
	return &recordingHandler{id: id, order: &u.order, mu: &u.mu}
}

func (u *c5Undo) calls() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return strings.Join(u.order, ",")
}

// c5Outcome is what one run of a fixture leaves behind.
type c5Outcome struct {
	s      *storage.SQLiteStorage
	iid    core.InstanceID
	status core.InstanceStatus
	pairs  []pair
	evs    []core.ExecutionEvent
	calls  map[string]int32 // step handler id → Execute count
	undos  string           // undo invocations, in order (filled by the caller)
	errs   int              // tick errors observed (trip ticks and drive ticks)
	// atTrip is the event stream and projected Variables right after the faulted tick,
	// before any restart or drive tick (nil when the run had no fault).
	atTrip   []core.ExecutionEvent
	tripVars map[string]any
}

// c5Run drives def to a terminal status like laDrive, with maxPar parallel steps. A
// non-nil ft arms exactly one fault; restart drops the faulted engine for a fresh one on
// the same store, clock and id source (a process restart). Handlers are shared by every
// engine of the run, so their call counts span the whole run.
func c5Run(t *testing.T, def core.WorkflowDefinition, maxPar int, ft *laFault, restart bool, handlers ...core.StepHandler) c5Outcome {
	t.Helper()
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, maxPar, handlers...)
	iid := laSubmit(t, e, def.ID)
	var errs []error
	if ft != nil {
		fs.arm(*ft)
		errs = laTickUntilTripped(t, e, fs)
		laWantInjected(t, errs)
	}
	var atTrip []core.ExecutionEvent
	var tripVars map[string]any
	if ft != nil {
		atTrip = laEvents(t, s, iid)
		tripVars = laInstance(t, s, iid).Variables
		if restart {
			e = laEngine(fs, clk, ids, maxPar, handlers...)
		}
	}
	_, more := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	errs = append(errs, more...)
	out := c5Outcome{
		s:        s,
		iid:      iid,
		status:   laInstance(t, s, iid).Status,
		evs:      laEvents(t, s, iid),
		calls:    map[string]int32{},
		errs:     len(errs),
		atTrip:   atTrip,
		tripVars: tripVars,
	}
	out.pairs, _ = eventPairs(t, s, iid)
	for _, h := range handlers {
		if sh, ok := h.(*stepHandler); ok {
			out.calls[sh.id] = laCalls(sh)
		}
	}
	return out
}

// c5DiffCalls asserts handler invocation counts equal the no-fault run's.
func c5DiffCalls(t *testing.T, label string, want, got map[string]int32) {
	t.Helper()
	if fmt.Sprint(want) != fmt.Sprint(got) {
		t.Errorf("%s: handler invocations differ from the no-fault run: want=%v got=%v", label, want, got)
	}
}

// c5Count counts events of type et in evs.
func c5Count(evs []core.ExecutionEvent, et core.EventType) int {
	return laCount(evs, et)
}

// ── N1 (P1): owed compensation only for a WorkflowFailed that names a step ───────────
//
// The SEC-08 stalled WorkflowFailed (tick.go, emitWorkflowFailed with step_id "") never
// compensates on its no-fault path. Its projection write failing once must not turn it
// into a compensation on recovery (hydrate.go owed-compensation block).
//
// CE-verified emission map (LA-C5 grep over the tracked tree, base db9fd36): emitWorkflowFailed
// has exactly four call sites. Every site with a NON-EMPTY step_id runs maybeCompensate on
// its no-fault path: failure.go:217→221 (route 3, also reached by the signal_timeout "fail"
// action), hydrate.go:603→607 (LA-C3 route 3), signal_timeout.go:138→142 (compensate). The
// only EMPTY step_id site, tick.go:434 (stalled), does not compensate.
//
// The guard that a route-3 WorkflowFailed (non-empty step_id) with a failed projection still
// compensates exactly once is TestLA_C4a_OwedCompensation_WorkflowFailedProjectionFailure_
// CompletesRoute3 (la_review_fix_test.go), which runs in the same package.

// c5DefStalled is the SEC-08 fixture: a → b is an unconditional dead end (b has no outgoing
// transition and is not a final step), and the only edge into final step c is guarded by a
// condition that is false. The instance reaches tick.go's stalled branch with a and b
// COMPLETED. The compensation plan over [a, b] gives an owed compensation undo work to do,
// so any undo invocation is observable.
func c5DefStalled() core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.c5.stalled", Version: "1.0.0", Namespace: "la", Name: "c5stalled",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "ha"), nativeStep("b", "hb"), nativeStep("c", "hc"),
		},
		Transitions: []core.Transition{
			{From: "a", To: "b"},
			{From: "a", To: "c", Condition: "steps.a.outputs.flag == 'yes'"},
		},
		Compensation: &core.CompensationPlan{Steps: []core.CompensationStep{
			{StepID: "a", UndoHandler: "undo-a"},
			{StepID: "b", UndoHandler: "undo-b"},
		}},
		InitialStep: "a", FinalSteps: []string{"c"}, Metadata: map[string]any{},
	}
}

// c5StalledHandlers returns fresh handlers for c5DefStalled; undo invocations are recorded in u.
func c5StalledHandlers(u *c5Undo) []core.StepHandler {
	return []core.StepHandler{
		&stepHandler{id: "ha", outputs: map[string]any{"flag": "no"}},
		&stepHandler{id: "hb", outputs: map[string]any{"done": true}},
		&stepHandler{id: "hc", outputs: map[string]any{"done": true}},
		u.handler("undo-a"),
		u.handler("undo-b"),
	}
}

func c5N1(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C5 N1 stalled WorkflowFailed projection fails", restart)

	var bu c5Undo
	base := c5Run(t, c5DefStalled(), 1, nil, false, c5StalledHandlers(&bu)...)
	base.undos = bu.calls()
	if base.errs != 0 {
		t.Fatalf("%s: no-fault baseline returned %d tick errors", label, base.errs)
	}
	// Oracle sanity: the no-fault stalled path fails and never compensates.
	if base.status != core.InstanceStatusFailed || base.undos != "" {
		t.Fatalf("%s: no-fault baseline status=%s undos=%q; want failed and no undo", label, base.status, base.undos)
	}

	var fu c5Undo
	ft := laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusFailed}
	got := c5Run(t, c5DefStalled(), 1, &ft, restart, c5StalledHandlers(&fu)...)
	got.undos = fu.calls()
	laCheck(t, got.s, got.iid, label, core.InstanceStatusFailed)
	wfs := laWorkflowFailedPayloads(t, got.evs)
	if len(wfs) != 1 || wfs[0].StepID != "" || wfs[0].Error.Code != "stalled" {
		t.Errorf("%s: WorkflowFailed payloads=%+v; want exactly one {step_id:\"\", code:stalled}", label, wfs)
	}
	if n := c5Count(got.evs, core.EventTypeWorkflowCompensating) + c5Count(got.evs, core.EventTypeWorkflowCompensated); n != 0 {
		t.Errorf("%s: %d WorkflowCompensating/WorkflowCompensated event(s); the stalled path must not compensate", label, n)
	}
	if got.undos != "" {
		t.Errorf("%s: undo invocations=%q; want none (CE-verified: step_id \"\" never compensates)", label, got.undos)
	}
	laDiffPairs(t, label, base.pairs, got.pairs)
	c5DiffCalls(t, label, base.calls, got.calls)
}

func TestLA_C5_N1_StalledWorkflowFailedProjectionFailure_NoCompensation(t *testing.T) {
	c5N1(t, false)
}

func TestLA_C5_N1_StalledWorkflowFailedProjectionFailure_NoCompensation_AfterRestart(t *testing.T) {
	c5N1(t, true)
}

// ── N2 (P2): a committed route-(3) decision stops settling (first terminal decision wins) ──
//
// b1 and b2 are dispatched in one tick (MaxParallelSteps=2). b1 fails terminally, so its
// StepFailed{retrying:false} is durable; then its WorkflowFailed append fails once. The
// no-fault path (tick.go settle loop; failure.go route 3) stops at b1: b2 is never settled
// and b1's decision is the only terminal one. Before this fix the settle loop went on to b2
// after the lost WorkflowFailed, so b2 was settled inside the window: it became the
// workflow's failure, or it entered Variables and was undone by compensation.

// c5DefFanOutComp is laDefFanOut with a compensation plan over [root, b1, b2].
func c5DefFanOutComp() core.WorkflowDefinition {
	def := laDefFanOut()
	def.Compensation = &core.CompensationPlan{Steps: []core.CompensationStep{
		{StepID: "root", UndoHandler: "undo-root"},
		{StepID: "b1", UndoHandler: "undo-b1"},
		{StepID: "b2", UndoHandler: "undo-b2"},
	}}
	return def
}

// c5FanOutHandlers: b1 always fails (errLAHandler, code handler_error); b2 fails only when
// b2Fails, and otherwise succeeds. Undo invocations are recorded in u.
func c5FanOutHandlers(u *c5Undo, b2Fails bool) []core.StepHandler {
	b2 := &stepHandler{id: "la_b2", outputs: map[string]any{"who": "b2"}}
	if b2Fails {
		b2.err = errLAHandler
	}
	return []core.StepHandler{
		&stepHandler{id: "la_root", outputs: map[string]any{}},
		&stepHandler{id: "la_b1", err: errLAHandler},
		b2,
		u.handler("undo-root"), u.handler("undo-b1"), u.handler("undo-b2"),
	}
}

// c5RoutingCount counts StepFailed and StepCompleted events for stepID.
func c5RoutingCount(evs []core.ExecutionEvent, stepID string) int {
	n := 0
	for _, ev := range evs {
		if ev.StepID == stepID && (ev.EventType == core.EventTypeStepFailed || ev.EventType == core.EventTypeStepCompleted) {
			n++
		}
	}
	return n
}

// c5CompensatingFrom returns the from_step of every WorkflowCompensating in evs, in order.
func c5CompensatingFrom(t *testing.T, evs []core.ExecutionEvent) []string {
	t.Helper()
	var out []string
	for _, ev := range evs {
		if ev.EventType != core.EventTypeWorkflowCompensating {
			continue
		}
		var p workflowCompensatingPayload
		decodePayload(t, ev, &p)
		out = append(out, p.FromStep)
	}
	return out
}

func c5N2(t *testing.T, restart, b2Fails bool) {
	t.Helper()
	label := laLabel(fmt.Sprintf("LA-C5 N2 route-3 WorkflowFailed append fails (b2_fails=%t)", b2Fails), restart)

	var bu c5Undo
	base := c5Run(t, c5DefFanOutComp(), 2, nil, false, c5FanOutHandlers(&bu, b2Fails)...)
	base.undos = bu.calls()
	if base.errs != 0 {
		t.Fatalf("%s: no-fault baseline returned %d tick errors", label, base.errs)
	}
	// Oracle sanity: the no-fault run ends at b1's decision, with b2 never settled.
	if got := c5RoutingCount(base.evs, "b2"); got != 0 {
		t.Fatalf("%s: no-fault baseline settled b2 (%d routing events); the oracle is broken", label, got)
	}

	var fu c5Undo
	ft := laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowFailed}
	got := c5Run(t, c5DefFanOutComp(), 2, &ft, restart, c5FanOutHandlers(&fu, b2Fails)...)
	got.undos = fu.calls()

	// Window: the trip tick (b1's WorkflowFailed append failed). b2 must not be settled, and
	// the successful-sibling case must not put b2 into the completed set (Variables).
	if n := c5RoutingCount(got.atTrip, "b2"); n != 0 {
		t.Errorf("%s: %d StepFailed/StepCompleted event(s) for b2 in the trip tick; b1's committed route-3 decision must stop settling", label, n)
	}
	if _, ok := got.tripVars["b2"]; ok {
		t.Errorf("%s: b2 entered Variables during the window (its StepCompleted was settled after b1's terminal decision)", label)
	}

	// Final: exactly one WorkflowFailed, attributed to b1 with b1's error code.
	wfs := laWorkflowFailedPayloads(t, got.evs)
	if len(wfs) != 1 || wfs[0].StepID != "b1" || wfs[0].Error.Code != "handler_error" {
		t.Errorf("%s: WorkflowFailed payloads=%+v; want exactly one {step_id:b1, code:handler_error}", label, wfs)
	}
	if from := c5CompensatingFrom(t, got.evs); fmt.Sprint(from) != "[b1]" {
		t.Errorf("%s: WorkflowCompensating from_step=%v; want [b1]", label, from)
	}
	if n := c5RoutingCount(got.evs, "b2"); n != c5RoutingCount(base.evs, "b2") {
		t.Errorf("%s: b2 routing events=%d; the no-fault run has %d", label, n, c5RoutingCount(base.evs, "b2"))
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

// (a) The reported sequence: b1 and b2 both fail; the WorkflowFailed(b1) append fails once.
func TestLA_C5_N2_Route3WorkflowFailedAppendFails_StopsSettling(t *testing.T) {
	c5N2(t, false, true)
}

func TestLA_C5_N2_Route3WorkflowFailedAppendFails_StopsSettling_AfterRestart(t *testing.T) {
	c5N2(t, true, true)
}

// (b) Hypothesis (review item): a SUCCESSFUL sibling b2 in the same batch must not enter the
// completed set during the window and must not be undone during compensation.
func TestLA_C5_N2b_SuccessfulSiblingNotCompletedInWindow(t *testing.T) {
	c5N2(t, false, false)
}

func TestLA_C5_N2b_SuccessfulSiblingNotCompletedInWindow_AfterRestart(t *testing.T) {
	c5N2(t, true, false)
}

// (c) A NON-terminal settle error still settles the sibling: pinned by
// TestLA_C4b_SettleFailureDoesNotAbandonDispatchedSibling (la_settle_serial_test.go), which
// runs in the same package and must stay green after this fix.

// ── N3 (P1): a signal-timeout routing error resets this instance's recovery ──────────
//
// timeout_action=continue: resumeTimedOutInstance writes status=running, then
// emitStepCompleted(w) appends StepCompleted durably, and its projection write fails once.
// The timeout scan logged the error and returned, so nothing re-hydrated the instance: w stayed
// in current_steps while the instance sat in running, wedged until a process restart. The fix
// (signal_timeout.go processTimeout) resets this instance's recovery bookkeeping on any routing
// error, so the next tick re-hydrates it and the catch-up re-applies the durable StepCompleted.
//
// Fault model: laModelFailBeforeWrite on the projection UpsertInstance(running). nth=2, because
// nth=1 is resumeTimedOutInstance's own running write in the same tick.

func c5N3Continue(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C5 N3 continue-timeout StepCompleted projection fails", restart)
	h := laC4cPark(t, "continue")

	h.st.arm(laFault{op: laOpUpsertInstance, nth: 2, status: core.InstanceStatusRunning})
	h.clk.advance(6 * time.Second)
	trip := laTickUntilTripped(h.t, h.e, h.st.laFaultStorage)
	if len(trip) != 0 {
		t.Errorf("%s: timeout failures must be logged, not returned: tick errors=%v", label, trip)
	}
	// Precondition: StepCompleted(w) is durable, and its projection was the write that failed.
	if r := h.result(); r.routing != 1 {
		t.Fatalf("%s: precondition: routing events for w = %d after the trip; want 1", label, r.routing)
	}
	if inst := laInstance(t, h.s, h.iid); inst.Status != core.InstanceStatusRunning || len(inst.CurrentSteps) != 1 {
		t.Fatalf("%s: precondition: status=%s current_steps=%v after the trip; want running with w still projected",
			label, inst.Status, inst.CurrentSteps)
	}

	if restart {
		h.restart()
	}
	n, errs := laTickUntil(h.t, h.e, h.s, h.iid, laK, laIsTerminal)
	h.t.Logf("%s: ticks=%d tick_errors=%d", label, n, len(errs))
	laCheck(t, h.s, h.iid, label, core.InstanceStatusCompleted,
		laExpect{"w", "StepCompleted"}, laExpect{"z", "StepCompleted"})
	if got := h.result().routing; got != 1 {
		t.Errorf("%s: routing events for w = %d; want exactly 1", label, got)
	}
}

// Same process: the instance must progress within K ticks in the engine that hit the fault.
func TestLA_C5_N3_ContinueProjectionFailure_SameProcessProgresses(t *testing.T) {
	c5N3Continue(t, false)
}

// Restart control (passes before the fix: a fresh process re-hydrates the instance anyway).
func TestLA_C5_N3_ContinueProjectionFailure_AfterRestart(t *testing.T) {
	c5N3Continue(t, true)
}

// ── Q3: the ack-lost fault model over the named Loop A scenarios ──────────────────────
//
// Each row runs its scenario twice, same-process and after-restart: once with NO fault (the
// oracle) and once with its fault in laModelAckLost (the write commits, then the caller sees
// an error). The faulted run must equal the no-fault run on final status, the (event_type,
// step_id) sequence, handler invocations and undo invocations. The oracle is the no-fault run,
// not the fail-before expectations of the original L-tests: under ack-lost the write is durable,
// so the correct outcome is the one that durable event implies.
//
// A row that fails is a finding of this card, NOT a fix: it is gated behind LA_DEFERRED=1
// (its failure message is reported), so the default run stays green.

// c5RunWait is L5's fixture (laDefWait: a → w → z). The fault hits StepCompleted(w), which
// fires inside SIGNAL_SCAN after the delivery commits. ft == nil is the no-fault run.
func c5RunWait(t *testing.T, restart bool, ft *laFault) c5Outcome {
	t.Helper()
	def := laDefWait()
	s, fs := laSetup(t, def)
	ha := &stepHandler{id: "la_a", outputs: map[string]any{}}
	hz := &stepHandler{id: "la_z", outputs: map[string]any{}}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 1, ha, hz)
	iid := laSubmit(t, e, def.ID)
	laTickUntil(t, e, s, iid, laSetupTicks, func(i core.WorkflowInstance) bool {
		return i.Status == core.InstanceStatusWaiting
	})
	if got := laInstance(t, s, iid).Status; got != core.InstanceStatusWaiting {
		t.Fatalf("setup: status=%s; want waiting (parked before the signal)", got)
	}
	if ft != nil {
		fs.arm(*ft)
	}
	if err := e.Signal(context.Background(), iid, "go", map[string]any{"by": "alice"}); err != nil {
		t.Fatalf("Signal: %v", err)
	}
	if ft != nil {
		laTickUntilTripped(t, e, fs) // the fault fires inside SIGNAL_SCAN; it is logged, not returned
		if restart {
			e = laEngine(fs, clk, ids, 1, ha, hz)
		}
	}
	_, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	out := c5Outcome{
		s:      s,
		iid:    iid,
		status: laInstance(t, s, iid).Status,
		evs:    laEvents(t, s, iid),
		calls:  map[string]int32{"la_a": laCalls(ha), "la_z": laCalls(hz)},
		errs:   len(errs),
	}
	out.pairs, _ = eventPairs(t, s, iid)
	return out
}

// c5AckRow is one ack-lost scenario: run executes it (ft == nil is the no-fault oracle).
type c5AckRow struct {
	lead string
	run  func(t *testing.T, restart bool, ft *laFault) c5Outcome
	ft   laFault // the ack-lost fault (model is set by c5AckRows)
}

func c5AckRows() []c5AckRow {
	ack := func(ft laFault) laFault { ft.model = laModelAckLost; return ft }
	return []c5AckRow{
		{lead: "L1", ft: ack(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepCompleted, stepID: "a"}),
			run: func(t *testing.T, restart bool, ft *laFault) c5Outcome {
				return c5Run(t, laDefSingle(), 1, ft, restart,
					&stepHandler{id: "la_a", outputs: map[string]any{"msg": "done"}})
			}},
		{lead: "L3", ft: ack(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepFailed, stepID: "a"}),
			run: func(t *testing.T, restart bool, ft *laFault) c5Outcome {
				return c5Run(t, laDefSingle(), 1, ft, restart, &stepHandler{id: "la_a", err: errLAHandler})
			}},
		{lead: "L5", ft: ack(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepCompleted, stepID: "w"}),
			run: c5RunWait},
		{lead: "L7", ft: ack(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepFallbackActivated, stepID: "a"}),
			run: func(t *testing.T, restart bool, ft *laFault) c5Outcome {
				ha, hfb := laFallbackHandlers()
				return c5Run(t, laDefRoute("la.fallback", nil, true), 1, ft, restart, ha, hfb)
			}},
		{lead: "L8", ft: ack(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowFailed}),
			run: func(t *testing.T, restart bool, ft *laFault) c5Outcome {
				r := newLACompRun()
				out := c5Run(t, compDef(), 1, ft, restart, r.handlers...)
				out.undos = r.undoOrder()
				return out
			}},
		// N1 (projection): the committed projection leaves `running`, so no recovery runs and the
		// row does not reach the owed-compensation path. It checks only that the committed write
		// is not re-applied.
		{lead: "N1-projection", ft: ack(laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusFailed}),
			run: func(t *testing.T, restart bool, ft *laFault) c5Outcome {
				var u c5Undo
				out := c5Run(t, c5DefStalled(), 1, ft, restart, c5StalledHandlers(&u)...)
				out.undos = u.calls()
				return out
			}},
		// N1 (append): the stalled WorkflowFailed is durable, its error is returned, and the
		// projection is still `running`, so recovery runs and reaches the owed-compensation path.
		{lead: "N1-append", ft: ack(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowFailed}),
			run: func(t *testing.T, restart bool, ft *laFault) c5Outcome {
				var u c5Undo
				out := c5Run(t, c5DefStalled(), 1, ft, restart, c5StalledHandlers(&u)...)
				out.undos = u.calls()
				return out
			}},
		// N2 (WorkflowFailed): the committed WorkflowFailed(b1) is durable, so durableTerminal stops
		// the settle loop even before the N2 fix. This row does not discriminate N2 (it passes on the
		// pre-N2 code too); the fail-before tests above do.
		{lead: "N2", ft: ack(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowFailed}),
			run: func(t *testing.T, restart bool, ft *laFault) c5Outcome {
				var u c5Undo
				out := c5Run(t, c5DefFanOutComp(), 2, ft, restart, c5FanOutHandlers(&u, true)...)
				out.undos = u.calls()
				return out
			}},
		// N2b (StepFailed, ack-lost): b1's StepFailed{retrying:false} is durable, but its append
		// returned an error. Before LA-C5 N2b the settle loop read that error as non-terminal and
		// settled the sibling b2 after b1's committed decision. The settle loop's log check now
		// recognises the committed route (3) decision, so the faulted run equals the no-fault run.
		{lead: "N2-StepFailed", ft: ack(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepFailed, stepID: "b1"}),
			run: func(t *testing.T, restart bool, ft *laFault) c5Outcome {
				var u c5Undo
				out := c5Run(t, c5DefFanOutComp(), 2, ft, restart, c5FanOutHandlers(&u, true)...)
				out.undos = u.calls()
				return out
			}},
	}
}

// TestLA_C5_Q3_AckLostScenarios runs every row in both modes; each subtest reports
// PASS or FAIL for its scenario.
func TestLA_C5_Q3_AckLostScenarios(t *testing.T) {
	for _, row := range c5AckRows() {
		for _, restart := range []bool{false, true} {
			mode := "same-process"
			if restart {
				mode = "after-restart"
			}
			t.Run(row.lead+"/"+mode, func(t *testing.T) {
				base := row.run(t, false, nil)
				if base.errs != 0 {
					t.Fatalf("no-fault baseline returned %d tick errors", base.errs)
				}
				ft := row.ft
				got := row.run(t, restart, &ft)
				label := "LA-C5 Q3 ack-lost " + row.lead + " " + mode
				if got.status != base.status {
					t.Errorf("%s: status=%s; the no-fault run ends %s", label, got.status, base.status)
				}
				laDiffPairs(t, label, base.pairs, got.pairs)
				c5DiffCalls(t, label, base.calls, got.calls)
				if got.undos != base.undos {
					t.Errorf("%s: undo invocations=%q; the no-fault run undoes %q", label, got.undos, base.undos)
				}
			})
		}
	}
}

// Q3 model check: the two fault models mean what their comments say. The ack-lost write is
// durable while the caller sees errLAInjected; the fail-before write is not durable.
func TestLA_C5_Q3_FaultModelSemantics(t *testing.T) {
	def := laDefSingle()
	ctx := context.Background()
	for _, model := range []laFaultModel{laModelFailBeforeWrite, laModelAckLost} {
		s, fs := laSetup(t, def)
		iid := laSubmit(t, laEngine(fs, newFakeClock(engineStart, time.Millisecond), laIDs(), 1,
			&stepHandler{id: "la_a"}), def.ID)
		fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepStarted, stepID: "a", model: model})
		ev := core.ExecutionEvent{InstanceID: iid, SequenceNum: 2, EventType: core.EventTypeStepStarted, StepID: "a",
			Payload: []byte(`{"attempt":1,"inputs":{}}`), EmittedAt: engineStart}
		err := fs.AppendEvent(ctx, ev)
		if !errors.Is(err, errLAInjected) {
			t.Fatalf("model %d: AppendEvent error=%v; want the injected error", model, err)
		}
		durable := c5HasEventType(laEvents(t, s, iid), core.EventTypeStepStarted, "a")
		want := model == laModelAckLost
		if durable != want {
			t.Errorf("model %d: write durable=%t after the injected error; want %t", model, durable, want)
		}
	}
}

// arm rejects ack-lost on an op whose model is not implemented, so it cannot silently degrade.
func TestLA_C5_Q3_AckLostRejectedForUnmodelledOps(t *testing.T) {
	_, fs := laSetup(t, laDefSingle())
	defer func() {
		if recover() == nil {
			t.Fatalf("arm accepted ack-lost for ClaimStep; want a panic")
		}
	}()
	fs.arm(laFault{op: laOpClaimStep, nth: 1, model: laModelAckLost})
}

func c5HasEventType(evs []core.ExecutionEvent, et core.EventType, stepID string) bool {
	return c4bCount(evs, et, stepID) > 0
}

// ── N2b (LA-C5): ack-lost-aware terminal check after a settle error ───────────────────
//
// committedRouteThree is the pure classification behind durableTerminal's second case. The
// end-to-end ack-lost runs are TestLA_C5_Q3_AckLostScenarios/N2-StepFailed (fail: a durable
// route-(3) StepFailed(b1) with its error returned; the sibling must not settle) and
// TestLA_C4b_SettleFailureDoesNotAbandonDispatchedSibling (a non-terminal error still settles
// the sibling). Each case below is one row of the static and log rule.

func c5RouteDV() *defView {
	return &defView{def: core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.c5.route", Version: "1.0.0", Namespace: "la", Name: "route",
		Steps:       []core.Step{nativeStep("s", "h"), nativeStep("se", "h"), nativeStep("oe", "h")},
		Transitions: []core.Transition{{From: "se", To: "oe", OnError: true}},
		InitialStep: "s", FinalSteps: []string{"oe"}, Metadata: map[string]any{},
	}}
}

// c5Ev builds one event for committedRouteThree. A StepFailed payload is marshalled from
// stepFailedPayload; a non-empty raw overrides the payload (malformed-payload case).
func c5Ev(seq int, et core.EventType, stepID string, retrying bool, code string, raw string) core.ExecutionEvent {
	ev := core.ExecutionEvent{InstanceID: "iid", SequenceNum: seq, EventType: et, StepID: stepID}
	switch {
	case raw != "":
		ev.Payload = []byte(raw)
	case et == core.EventTypeStepFailed:
		p, err := marshalPayload(stepFailedPayload{StepID: stepID, Attempt: 1, Retrying: retrying, Error: core.StepError{Code: code}})
		if err != nil {
			panic(err)
		}
		ev.Payload = p
	}
	return ev
}

func TestLA_C5_N2b_CommittedRouteThree(t *testing.T) {
	dv := c5RouteDV()
	s := nativeStep("s", "h")
	fallbackStep := s
	fallbackStep.Fallback = "fb"
	ss := func(seq int, id string) core.ExecutionEvent {
		return c5Ev(seq, core.EventTypeStepStarted, id, false, "", "")
	}
	sf := func(seq int, id string, retrying bool, code string) core.ExecutionEvent {
		return c5Ev(seq, core.EventTypeStepFailed, id, retrying, code, "")
	}
	cases := []struct {
		name    string
		step    core.Step
		evs     []core.ExecutionEvent
		want    bool
		wantErr bool
	}{
		{"route3 StepFailed after its StepStarted is committed", s,
			[]core.ExecutionEvent{ss(2, "s"), sf(3, "s", false, "handler_error")}, true, false},
		{"StepFailed before the latest StepStarted is not this attempt's", s,
			[]core.ExecutionEvent{sf(2, "s", false, "handler_error"), ss(3, "s")}, false, false},
		{"retrying:true is not a terminal decision", s,
			[]core.ExecutionEvent{ss(2, "s"), sf(3, "s", true, "handler_error")}, false, false},
		{"code cancelled carries no routing decision", s,
			[]core.ExecutionEvent{ss(2, "s"), sf(3, "s", false, "cancelled")}, false, false},
		{"code signal_timeout carries no routing decision", s,
			[]core.ExecutionEvent{ss(2, "s"), sf(3, "s", false, "signal_timeout")}, false, false},
		{"route 1 (Fallback set) is not route 3", fallbackStep,
			[]core.ExecutionEvent{ss(2, "s"), sf(3, "s", false, "handler_error")}, false, false},
		{"route 2 (on_error transition) is not route 3", nativeStep("se", "h"),
			[]core.ExecutionEvent{ss(2, "se"), sf(3, "se", false, "handler_error")}, false, false},
		{"no StepFailed in the log", s,
			[]core.ExecutionEvent{ss(2, "s")}, false, false},
		{"a StepFailed for another step is ignored", s,
			[]core.ExecutionEvent{ss(2, "s"), sf(3, "other", false, "handler_error")}, false, false},
		{"a later StepStarted supersedes an earlier committed StepFailed", s,
			[]core.ExecutionEvent{ss(2, "s"), sf(3, "s", false, "handler_error"), ss(4, "s")}, false, false},
		{"a retry attempt that commits route 3 after its own StepStarted", s,
			[]core.ExecutionEvent{ss(2, "s"), sf(3, "s", true, "handler_error"), ss(4, "s"), sf(5, "s", false, "handler_error")}, true, false},
		{"no StepStarted (template_error item): a StepFailed counts from the start of the log", s,
			[]core.ExecutionEvent{sf(3, "s", false, "template_error")}, true, false},
		{"malformed StepFailed payload after its StepStarted is an error", s,
			[]core.ExecutionEvent{ss(2, "s"), c5Ev(3, core.EventTypeStepFailed, "s", false, "", "not json")}, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := committedRouteThree(dv, c.step, c.evs)
			if (err != nil) != c.wantErr {
				t.Fatalf("error=%v; want error=%t", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("committedRouteThree=%t; want %t", got, c.want)
			}
		})
	}
}
