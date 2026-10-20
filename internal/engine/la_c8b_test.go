package engine

// LA-C8b (PR #4 review round 5, R5-3 and R5-4(a)): TEST-ONLY file. It adds no production code.
//
// R5-3: a read error in the committed-route recovery stage (route_recovery.go and the hydrate.go
// bodies it runs) keeps the instance queued (setRouteRecovery(iid, true)), so the next tick converges
// to the no-read-fault control. One test per reachable read site. Each armed site fails once, on its
// first matching read, without reaching the store. Some sites do not depend on the keep: their error
// is absorbed (S11c, the InstanceVersion fallback in expectedVersion), or the instance is already
// terminal when the read runs (S12, S14, S15). Their tests still pin that the injected read fault
// does not diverge; their comments say so.
//
// R5-4(a): pins the two WorkflowCompensating guards: hydrate.go applyTerminalFence (the early return
// when the last terminal-class event is WorkflowCompensating) and route_recovery.go
// projectionBehindTerminal (the WorkflowCompensating exclusion).
//
// Site attribution (c8bChain): a storage read is attributed to the engine function that made it by
// walking the call stack (runtime.CallersFrames). The innermost frame named in c8bSiteFns is the
// inner site, and the next such frame outward is the outer site. Helpers (getInstance, project,
// expectedVersion, defViewFor, cancellationRequested) are transparent: their reads belong to the
// site that called them. c8bSiteFns is the eight functions the rows name plus maybeCompensate, whose
// own read must not be attributed to its caller. Renaming one of them breaks the attribution, so the
// armed fault never fires and assertion (a) FAILS. It never skips.
//
// Oracle: the no-read-fault control of the same scenario. The scenario always includes the injected
// WorkflowFailed append failure (fail-before, or ack-lost for the ACK-LOST rows), so the control is
// the same scenario with no read fault armed.

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// c8bSiteFns names the engine functions that own a read site (see the file comment).
var c8bSiteFns = map[string]bool{
	"completeRouteRecovery":          true,
	"completeAwaitingRoutes":         true,
	"completeCommittedRoutes":        true,
	"fenceLoggedTerminal":            true,
	"applyLoggedTerminal":            true,
	"applyTerminalFence":             true,
	"reprojectTerminal":              true,
	"committedRouteAwaitingTerminal": true,
	"maybeCompensate":                true,
}

// c8bChain returns the inner and outer site names of the storage read that is running.
func c8bChain() (string, string) {
	pcs := make([]uintptr, 128)
	n := runtime.Callers(1, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var names []string
	for {
		fr, more := frames.Next()
		if i := strings.LastIndex(fr.Function, "."); i >= 0 && c8bSiteFns[fr.Function[i+1:]] {
			names = append(names, fr.Function[i+1:])
		}
		if !more {
			break
		}
	}
	inner, outer := "", ""
	if len(names) > 0 {
		inner = names[0]
	}
	if len(names) > 1 {
		outer = names[1]
	}
	return inner, outer
}

// c8bSite is one armed read fault. It fails the first read whose storage method is op and whose
// attributed sites are inner and outer ("" = any outer site).
type c8bSite struct {
	label   string // R5-3 row name (S1, S4, ...)
	op      string // GetInstance | ReadEvents | CancellationRequested | GetWorkflow | InstanceVersion
	inner   string // innermost site function that made the read
	outer   string // next site function outward; "" = any
	tripped bool   // assertion (a): the injected fault fired
}

// c8bSiteStore wraps the fault layer with per-site read faults. Writes and the untouched reads are
// forwarded by laFaultStorage and the embedded SQLiteStorage.
type c8bSiteStore struct {
	*laFaultStorage
	mu    sync.Mutex
	sites []*c8bSite
}

// hit reports whether the current read, made by op, is an armed site, and marks that site tripped.
func (f *c8bSiteStore) hit(op string) bool {
	inner, outer := c8bChain()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sites {
		if s.tripped || s.op != op || s.inner != inner {
			continue
		}
		if s.outer != "" && s.outer != outer {
			continue
		}
		s.tripped = true
		return true
	}
	return false
}

func (f *c8bSiteStore) GetInstance(ctx context.Context, iid core.InstanceID) (core.WorkflowInstance, error) {
	if f.hit("GetInstance") {
		return core.WorkflowInstance{}, laInjected(laOp("GetInstance"))
	}
	return f.laFaultStorage.GetInstance(ctx, iid)
}

func (f *c8bSiteStore) ReadEvents(ctx context.Context, iid core.InstanceID, fromSeq int) ([]core.ExecutionEvent, error) {
	if f.hit("ReadEvents") {
		return nil, laInjected(laOp("ReadEvents"))
	}
	return f.laFaultStorage.ReadEvents(ctx, iid, fromSeq)
}

func (f *c8bSiteStore) CancellationRequested(ctx context.Context, iid core.InstanceID) (bool, error) {
	if f.hit("CancellationRequested") {
		return false, laInjected(laOp("CancellationRequested"))
	}
	return f.laFaultStorage.CancellationRequested(ctx, iid)
}

func (f *c8bSiteStore) GetWorkflow(ctx context.Context, id string, version core.SemVer) (core.WorkflowDefinition, error) {
	if f.hit("GetWorkflow") {
		return core.WorkflowDefinition{}, laInjected(laOp("GetWorkflow"))
	}
	return f.laFaultStorage.GetWorkflow(ctx, id, version)
}

func (f *c8bSiteStore) InstanceVersion(ctx context.Context, iid core.InstanceID) (int, error) {
	if f.hit("InstanceVersion") {
		return 0, laInjected(laOp("InstanceVersion"))
	}
	return f.laFaultStorage.InstanceVersion(ctx, iid)
}

// c8bRun is what one R5-3 scenario run leaves behind.
type c8bRun struct {
	status core.InstanceStatus
	wf     int
	undos  string
	pairs  []pair
	trip   error
	events string
}

// c8bR53Run drives the c6SiblingDef fixture (root -> {w signal "go", b1 route (3)}) with no signal.
// Tick 1 completes root. The trip tick settles b1 as route (3), and its WorkflowFailed append fails
// once (fail-before, or ack-lost when ackLost). restartBeforeTrip runs the trip tick on a fresh engine
// (a cold definition cache and no in-memory bookkeeping). The sites are armed for the whole run. Then
// live ticks run until a terminal status (bounded by laK). The run is serial, so the attribution is exact.
func c8bR53Run(t *testing.T, sites []*c8bSite, ackLost, restartBeforeTrip bool) c8bRun {
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
	model := laModelFailBeforeWrite
	if ackLost {
		model = laModelAckLost
	}
	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowFailed, model: model})
	st := &c8bSiteStore{laFaultStorage: fs, sites: sites}
	e := laEngine(st, clk, ids, 2, handlers...)
	iid := laSubmit(t, e, def.ID)
	ctx := context.Background()
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("tick 1: %v", err)
	}
	if restartBeforeTrip {
		e = laEngine(st, clk, ids, 2, handlers...)
	}
	trip := e.Tick(ctx)
	// Tick errors after the trip are not the oracle; the status and the event log are.
	laTickUntil(t, e, s, iid, laK, laIsTerminal)
	evs := laEvents(t, s, iid)
	pairs, _ := eventPairs(t, s, iid)
	return c8bRun{
		status: laInstance(t, s, iid).Status,
		wf:     laCount(evs, core.EventTypeWorkflowFailed),
		undos:  u.calls(),
		pairs:  pairs,
		trip:   trip,
		events: laSummary(t, evs),
	}
}

// c8bR53Case runs the scenario with sites armed and the no-read-fault control, then asserts:
// (a) every armed site fired; (b) the faulted run equals the control on status, WorkflowFailed count,
// undo invocations and the (event_type, step_id) sequence, and ends terminal. The injected
// WorkflowFailed failure must surface through the trip tick in both runs.
func c8bR53Case(t *testing.T, label string, ackLost bool, sites ...*c8bSite) {
	t.Helper()
	ctl := c8bR53Run(t, nil, ackLost, false)
	if !errors.Is(ctl.trip, errLAInjected) || !isTerminal(ctl.status) || ctl.wf != 1 || ctl.undos != "undo-root" {
		t.Fatalf("%s: the no-read-fault control is not the expected run (trip=%v status=%s wf=%d undos=%q); the oracle is broken; events: %s",
			label, ctl.trip, ctl.status, ctl.wf, ctl.undos, ctl.events)
	}
	got := c8bR53Run(t, sites, ackLost, false)
	for _, s := range sites {
		if !s.tripped {
			t.Errorf("%s: read site %s (%s, inner=%s outer=%s) was never reached: the injected read fault did not fire",
				label, s.label, s.op, s.inner, s.outer)
		}
	}
	if !errors.Is(got.trip, errLAInjected) {
		t.Errorf("%s: the injected WorkflowFailed failure did not surface through the trip tick: %v", label, got.trip)
	}
	if !isTerminal(got.status) {
		t.Errorf("%s: status=%s after K=%d ticks; the instance must reach a terminal status; events: %s", label, got.status, laK, got.events)
	}
	if got.status != ctl.status {
		t.Errorf("%s: status=%s; the no-read-fault control ends %s; events: %s", label, got.status, ctl.status, got.events)
	}
	if got.wf != ctl.wf {
		t.Errorf("%s: WorkflowFailed count=%d; the control has %d; events: %s", label, got.wf, ctl.wf, got.events)
	}
	if got.undos != ctl.undos {
		t.Errorf("%s: undo invocations=%q; the control undoes %q", label, got.undos, ctl.undos)
	}
	laDiffPairs(t, label, ctl.pairs, got.pairs)
}

// ── R5-3: one test per reachable read site (fail-before unless the name says ACK-LOST) ───────────

// S1: completeRouteRecovery's status read of an errored instance (route_recovery.go, errored loop).
func TestLA_C8b_R53_S1_ErroredStatusGetInstance(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S1 errored-status GetInstance", false,
		&c8bSite{label: "S1", op: "GetInstance", inner: "completeRouteRecovery"})
}

// S4: completeAwaitingRoutes' instance read (route_recovery.go, the first read of the stage).
func TestLA_C8b_R53_S4_StageGetInstance(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S4 stage GetInstance", false,
		&c8bSite{label: "S4", op: "GetInstance", inner: "completeAwaitingRoutes", outer: "completeRouteRecovery"})
}

// S5: completeAwaitingRoutes' event-log read.
func TestLA_C8b_R53_S5_StageReadEvents(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S5 stage ReadEvents", false,
		&c8bSite{label: "S5", op: "ReadEvents", inner: "completeAwaitingRoutes", outer: "completeRouteRecovery"})
}

// S9: completeAwaitingRoutes' cancellation-flag read.
func TestLA_C8b_R53_S9_StageCancellationRequested(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S9 stage CancellationRequested", false,
		&c8bSite{label: "S9", op: "CancellationRequested", inner: "completeAwaitingRoutes", outer: "completeRouteRecovery"})
}

// S10: completeCommittedRoutes' instance read (hydrate.go, the committed route (3) completion).
func TestLA_C8b_R53_S10_CompleteCommittedGetInstance(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S10 completeCommittedRoutes GetInstance", false,
		&c8bSite{label: "S10", op: "GetInstance", inner: "completeCommittedRoutes", outer: "completeAwaitingRoutes"})
}

// S10 + S10b: the same route completion fails its instance read, and the fence's event-log read
// (fenceLoggedTerminal, after the route error) fails too. Both must be retried.
func TestLA_C8b_R53_S10S10b_DoubleFenceReads(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S10+S10b completeCommittedRoutes GetInstance and fence ReadEvents", false,
		&c8bSite{label: "S10", op: "GetInstance", inner: "completeCommittedRoutes", outer: "completeAwaitingRoutes"},
		&c8bSite{label: "S10b", op: "ReadEvents", inner: "fenceLoggedTerminal", outer: "completeAwaitingRoutes"})
}

// S11a (ACK-LOST): reprojectTerminal's instance read. The durable WorkflowFailed is re-projected by
// the stage's terminal fence; the read fails, so the instance must stay queued.
func TestLA_C8b_R53_S11a_AckLostReprojectGetInstance(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S11a ACK-LOST reprojectTerminal GetInstance", true,
		&c8bSite{label: "S11a", op: "GetInstance", inner: "reprojectTerminal", outer: "applyTerminalFence"})
}

// S11c (ACK-LOST): the OCC version read inside project (reached from reprojectTerminal). It is
// transparent to the caller: expectedVersion falls back to the cached version on a non-NotFound error
// (emit.go), so the fault fires but does not fail the stage.
func TestLA_C8b_R53_S11c_AckLostReprojectInstanceVersion(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S11c ACK-LOST reprojectTerminal InstanceVersion", true,
		&c8bSite{label: "S11c", op: "InstanceVersion", inner: "reprojectTerminal", outer: "applyTerminalFence"})
}

// S11d (ACK-LOST): applyTerminalFence's read for an owed compensation, after the re-projection
// succeeded. The compensation is owed for the durable WorkflowFailed, so the stage must retry it.
func TestLA_C8b_R53_S11d_AckLostOwedCompensationGetInstance(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S11d ACK-LOST owed-compensation GetInstance", true,
		&c8bSite{label: "S11d", op: "GetInstance", inner: "applyTerminalFence", outer: "applyLoggedTerminal"})
}

// S12: committedRouteAwaitingTerminal's instance read (the stage's awaiting check). NOT keep-dependent:
// the trip tick completes the route inside completeAwaitingRoutes, so the instance is terminal when this
// check runs. The awaiting-check keep (route_recovery.go, the error path) is not load-bearing here, and
// this test does not fail when it is removed (mutation run recorded in the commit report).
func TestLA_C8b_R53_S12_AwaitingCheckGetInstance(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S12 awaiting-check GetInstance", false,
		&c8bSite{label: "S12", op: "GetInstance", inner: "committedRouteAwaitingTerminal", outer: "completeRouteRecovery"})
}

// S14: committedRouteAwaitingTerminal's event-log read (the stage's awaiting check). NOT keep-dependent
// for the reason given at S12.
func TestLA_C8b_R53_S14_AwaitingCheckReadEvents(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S14 awaiting-check ReadEvents", false,
		&c8bSite{label: "S14", op: "ReadEvents", inner: "committedRouteAwaitingTerminal", outer: "completeRouteRecovery"})
}

// S15: committedRouteAwaitingTerminal's cancellation-flag read (the stage's awaiting check). NOT
// keep-dependent for the reason given at S12.
func TestLA_C8b_R53_S15_AwaitingCheckCancellationRequested(t *testing.T) {
	c8bR53Case(t, "LA-C8b R5-3 S15 awaiting-check CancellationRequested", false,
		&c8bSite{label: "S15", op: "CancellationRequested", inner: "committedRouteAwaitingTerminal", outer: "completeRouteRecovery"})
}

// S8 and S13 (GetWorkflow through defViewFor) are NOT per-site tests: the definition cache is loaded
// before the stage can hold an instance. Submit caches the definition (submit.go). The stage holds an
// instance only after processInstance (which calls defViewFor first) or the gate (which calls
// committedRouteAwaitingTerminal, and so defViewFor, first). Nothing evicts the cache (e.defs is never
// deleted from). This tripwire arms both faults and asserts that neither fires, both on the engine that
// submitted the instance and on a fresh engine (cold cache) that runs the trip tick. If it fails, the
// cache policy changed, and S8/S13 need their own per-site tests.
func TestLA_C8b_R53_S8S13_GetWorkflowUnreachable(t *testing.T) {
	for _, restart := range []bool{false, true} {
		sites := []*c8bSite{
			{label: "S8", op: "GetWorkflow", inner: "completeAwaitingRoutes", outer: "completeRouteRecovery"},
			{label: "S13", op: "GetWorkflow", inner: "committedRouteAwaitingTerminal", outer: "completeRouteRecovery"},
		}
		c8bR53Run(t, sites, false, restart)
		for _, s := range sites {
			if s.tripped {
				t.Errorf("%s (fresh engine before the trip tick=%t): the GetWorkflow read fault fired in the stage; the definition cache is no longer warm there", s.label, restart)
			}
		}
	}
}

// ── R5-4(a): the two WorkflowCompensating guards ─────────────────────────────────────────────────

// c8bWCEnv is a single-step instance whose log is the WC-last shape: WorkflowStarted (projected), then
// the RAW (unprojected) StepStarted(a), StepFailed(a, retrying:false), WorkflowFailed(a) and
// WorkflowCompensating(a). The projection stays non-terminal (running). The log's last terminal-class
// event is WorkflowCompensating, so LA-D1 leaves the instance alone.
type c8bWCEnv struct {
	s   *storage.SQLiteStorage
	fs  *laFaultStorage
	clk *fakeClock
	ids func() string
	h   *stepHandler
	iid core.InstanceID
}

func c8bWCLast(t *testing.T) c8bWCEnv {
	t.Helper()
	def := laDefSingle()
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	h := &stepHandler{id: "la_a", outputs: map[string]any{}}
	e := laEngine(fs, clk, ids, 1, h)
	iid := laSubmit(t, e, def.ID)
	inst := laInstance(t, s, iid)
	laC4aAppendRaw(t, e, inst, core.EventTypeStepStarted, "a", stepStartedPayload{StepID: "a", Attempt: 1, Inputs: map[string]any{}})
	laC4aAppendRaw(t, e, inst, core.EventTypeStepFailed, "a", stepFailedPayload{StepID: "a", Attempt: 1,
		Error: core.StepError{Code: "la_probe", Message: "probe"}, Retrying: false})
	laC4aAppendRaw(t, e, inst, core.EventTypeWorkflowFailed, "a", workflowFailedPayload{StepID: "a",
		Error: core.StepError{Code: "la_probe", Message: "probe"}})
	laC4aAppendRaw(t, e, inst, core.EventTypeWorkflowCompensating, "", workflowCompensatingPayload{FromStep: "a"})
	if got := laInstance(t, s, iid).Status; got != core.InstanceStatusRunning {
		t.Fatalf("setup: projection status=%s; want running (the raw events are not projected)", got)
	}
	return c8bWCEnv{s: s, fs: fs, clk: clk, ids: ids, h: h, iid: iid}
}

// TestLA_C8b_WCLast_HydrateFenceDoesNotProject pins applyTerminalFence's WorkflowCompensating early
// return (hydrate.go). The fence must not apply to a WC-last log: it returns applied=false without
// projecting the WorkflowCompensating event, so the projection keeps its status. Mutation: drop the
// "|| lastTerm.EventType == core.EventTypeWorkflowCompensating" clause, and the fence projects the
// WorkflowCompensating event (status becomes compensating) and reports applied.
func TestLA_C8b_WCLast_HydrateFenceDoesNotProject(t *testing.T) {
	env := c8bWCLast(t)
	ctx := context.Background()
	before := laInstance(t, env.s, env.iid)
	e2 := laEngine(env.fs, env.clk, env.ids, 1, env.h) // restart shape: no in-memory bookkeeping
	dv, err := e2.defViewFor(ctx, before)
	if err != nil {
		t.Fatalf("defViewFor: %v", err)
	}
	applied, err := e2.applyTerminalFence(ctx, dv, before, laEvents(t, env.s, env.iid))
	if err != nil {
		t.Fatalf("applyTerminalFence: %v", err)
	}
	if applied {
		t.Errorf("WC-last log: applyTerminalFence applied; the log's last terminal-class event is WorkflowCompensating (LA-D1: left untouched)")
	}
	if got := laInstance(t, env.s, env.iid).Status; got != before.Status {
		t.Errorf("WC-last log: applyTerminalFence rewrote the projection: status %s -> %s", before.Status, got)
	}
	laC4aHydrate(t, env.s, e2, env.iid)
	if after := laInstance(t, env.s, env.iid); after.Status != before.Status {
		t.Errorf("WC-last log: hydrate rewrote the projection: status %s -> %s; events: %s",
			before.Status, after.Status, laSummary(t, laEvents(t, env.s, env.iid)))
	}
}

// TestLA_C8b_WCLast_AwaitingCheckNotAwaiting pins projectionBehindTerminal's WorkflowCompensating
// exclusion (route_recovery.go). On a WC-last log the committed route (3) is decided and the awaiting
// check must report it as not awaiting. Mutation: drop the "&& last.EventType !=
// core.EventTypeWorkflowCompensating" clause, and the check reports the non-terminal projection as
// behind a durable terminal event (awaiting, step "").
func TestLA_C8b_WCLast_AwaitingCheckNotAwaiting(t *testing.T) {
	env := c8bWCLast(t)
	e2 := laEngine(env.fs, env.clk, env.ids, 1, env.h)
	step, awaiting, err := e2.committedRouteAwaitingTerminal(context.Background(), env.iid)
	if err != nil {
		t.Fatalf("committedRouteAwaitingTerminal: %v", err)
	}
	if awaiting || step != "" {
		t.Errorf("WC-last log: committedRouteAwaitingTerminal reported awaiting (step %q, awaiting %t); the log's last terminal-class event is WorkflowCompensating (LA-D1: not awaiting)", step, awaiting)
	}
}
