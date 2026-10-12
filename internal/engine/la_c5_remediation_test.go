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
		if restart {
			e = laEngine(fs, clk, ids, maxPar, handlers...)
		}
	}
	_, more := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	errs = append(errs, more...)
	out := c5Outcome{
		s:      s,
		iid:    iid,
		status: laInstance(t, s, iid).Status,
		evs:    laEvents(t, s, iid),
		calls:  map[string]int32{},
		errs:   len(errs),
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
