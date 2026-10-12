package engine

// LA-C2 — graceful shutdown: no new work after stop, in-flight work settles.
//
// Contract (quoted verbatim; see la_shutdown_repro_test.go for the full header):
//
//	PRD FR-RM-04: "`awis stop` gracefully stops the runtime (in-flight steps complete; no new steps dispatched)"
//	docs/CLI_CONTRACT.md §SIGTERM Handling: "Context cancel → engine tick completes in-flight step → graceful drain → exit 0"
//
// Coverage map (tick.go stop gates, see the tick doc):
//
//	S6        Run-level: two instances, stop while A's step is in flight (gate b, via Run)
//	S7        public Tick(cancelledCtx) semantics are unchanged
//	S8        gate (a): a tick entered with stop already closed does nothing
//	S9        gate (b) + (d-1): stop lands after ListInstances; no processInstance, no SIGNAL_SCAN
//	S10       gate (c): stop lands inside processInstance; no gatherDispatch
//	S11       gate (d-2): stop lands between SIGNAL_SCAN and SIGNAL_TIMEOUT_SCAN

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// laGateStore wraps the real SQLite storage, counts the storage calls the stop
// gates are defined over, and runs optional hooks AFTER the real call returned,
// on the tick goroutine. A hook that closes stop therefore lands exactly between
// the call it follows and the next stop check.
type laGateStore struct {
	*storage.SQLiteStorage

	onListInstances   func()
	onGetInstance     func()
	onListUndelivered func()

	listInstanceCalls atomic.Int32 // ListInstances reached storage (tick SCAN_RUNNABLE)
	getInstanceCalls  atomic.Int32 // GetInstance reached storage (processInstance entered)
	undeliveredScans  atomic.Int32 // ListUndeliveredSignals reached storage (SIGNAL_SCAN)
	dueTimeoutScans   atomic.Int32 // ListDueWaitRecords reached storage (SIGNAL_TIMEOUT_SCAN)
}

func (g *laGateStore) ListInstances(ctx context.Context, f core.InstanceFilter) ([]core.WorkflowInstance, error) {
	g.listInstanceCalls.Add(1)
	out, err := g.SQLiteStorage.ListInstances(ctx, f)
	if g.onListInstances != nil {
		g.onListInstances()
	}
	return out, err
}

func (g *laGateStore) GetInstance(ctx context.Context, id core.InstanceID) (core.WorkflowInstance, error) {
	g.getInstanceCalls.Add(1)
	out, err := g.SQLiteStorage.GetInstance(ctx, id)
	if g.onGetInstance != nil {
		g.onGetInstance()
	}
	return out, err
}

func (g *laGateStore) ListUndeliveredSignals(ctx context.Context, id core.InstanceID) ([]storage.Signal, error) {
	g.undeliveredScans.Add(1)
	out, err := g.SQLiteStorage.ListUndeliveredSignals(ctx, id)
	if g.onListUndelivered != nil {
		g.onListUndelivered()
	}
	return out, err
}

func (g *laGateStore) ListDueWaitRecords(ctx context.Context, now time.Time) ([]storage.WaitRecord, error) {
	g.dueTimeoutScans.Add(1)
	return g.SQLiteStorage.ListDueWaitRecords(ctx, now)
}

// laStopOnce returns a stop channel and an idempotent closer for it.
func laStopOnce() (chan struct{}, func()) {
	stop := make(chan struct{})
	var once sync.Once
	return stop, func() { once.Do(func() { close(stop) }) }
}

// laNativeRunner registers the given handlers on a fresh native runner.
func laNativeRunner(hs ...*laHandler) *native.NativeRunner {
	nr := native.New()
	for _, h := range hs {
		nr.Register(h)
	}
	return nr
}

// laHasType reports whether any persisted event has type et.
func laHasType(evs []core.ExecutionEvent, et core.EventType) bool {
	for _, ev := range evs {
		if ev.EventType == et {
			return true
		}
	}
	return false
}

// laShutdownSubmit submits one instance of the workflow id and returns its id.
func laShutdownSubmit(t *testing.T, e *Engine, id string) core.InstanceID {
	t.Helper()
	iid, err := e.Submit(context.Background(), id, "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return iid
}

// laGateSetup opens a fresh store, wraps it in laGateStore, and builds an engine
// on the wrapper. The handler "a" is returned for exec counting; it has no gate.
func laGateSetup(t *testing.T, id string) (*storage.SQLiteStorage, *laGateStore, *laHandler, *Engine) {
	t.Helper()
	s := laShutdownSetup(t, id)
	g := &laGateStore{SQLiteStorage: s}
	ha := &laHandler{id: "a", outputs: map[string]any{"r": "a"}}
	hb := &laHandler{id: "b", outputs: map[string]any{"r": "b"}}
	e := New(g, map[core.StepType]Runner{core.StepTypeNative: laNativeRunner(ha, hb)}, Config{
		TickInterval:     time.Millisecond,
		MaxParallelSteps: 1,
		WorkerID:         "la-c2-gate",
		Clock:            newFakeClock(engineStart, 10*time.Millisecond).now,
	}, laLogger(&laLog{}))
	return s, g, ha, e
}

// laAssertIdle asserts that a tick the stop gate fully withheld left the instance
// untouched: no StepStarted (no claim, no dispatch) and no handler execution.
func laAssertIdle(t *testing.T, s *storage.SQLiteStorage, ha *laHandler, iid core.InstanceID) {
	t.Helper()
	evs := laShutdownEvents(t, s, iid)
	if laHasType(evs, core.EventTypeStepStarted) {
		t.Errorf("FR-RM-04: withheld tick must not claim or start a step; persisted: %s", laSummary(t, evs))
	}
	if n := ha.calls.Load(); n != 0 {
		t.Errorf("FR-RM-04: withheld tick must not execute handler a; executed %d times", n)
	}
	inst, err := s.GetInstance(context.Background(), iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if inst.Status != core.InstanceStatusRunning {
		t.Errorf("instance must stay running across a withheld tick; status = %q", inst.Status)
	}
}

// TestLA_Shutdown_S6_NoNewWorkAfterStop — two instances of one workflow (a → b),
// both submitted before Run. The tick lists running instances in started_at
// order, so A (listed first) is processed first and B is reached only after A.
// Stop fires while A's step a is in flight.
//
// Expected (FR-RM-04): A's in-flight a completes and its StepCompleted is
// persisted; B — not yet reached — gets NO StepStarted and its handler is never
// called; Run returns context.Canceled. A NEW engine on the same storage then
// completes both instances: the undispatched work is picked up after restart.
func TestLA_Shutdown_S6_NoNewWorkAfterStop(t *testing.T) {
	s := laShutdownSetup(t, "la.s6")
	entered := newLASignal()
	gate := newLAGate(t)
	ha := &laHandler{id: "a", entered: entered, gate: gate, outputs: map[string]any{"r": "a"}}
	hb := &laHandler{id: "b", outputs: map[string]any{"r": "b"}}
	e := laNewEngine(s, "la-s6-1", map[core.StepType]Runner{core.StepTypeNative: laNativeRunner(ha, hb)}, laLogger(&laLog{}))

	ctx := context.Background()
	laShutdownSubmit(t, e, "la.s6")
	laShutdownSubmit(t, e, "la.s6")
	running, err := s.ListInstances(ctx, core.InstanceFilter{Status: core.InstanceStatusRunning})
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	if len(running) != 2 {
		t.Fatalf("precondition: want 2 running instances, got %d", len(running))
	}
	iidA, iidB := running[0].InstanceID, running[1].InstanceID

	runCtx, cancelRun := context.WithCancel(ctx)
	t.Cleanup(cancelRun)
	runDone := make(chan error, 1)
	go func() { runDone <- e.Run(runCtx) }()

	entered.wait(t) // A's a is in flight inside DISPATCH; B has not been reached
	cancelRun()     // stop fires while A is in flight
	gate.open()     // only now may a's handler return

	runErr := laAwaitErr(t, runDone, "Run")
	if !errors.Is(runErr, context.Canceled) {
		t.Errorf("Run must return ctx.Err() on stop; got %v", runErr)
	}

	evsA := laShutdownEvents(t, s, iidA)
	evsB := laShutdownEvents(t, s, iidB)
	t.Logf("S6 A persisted: %s", laSummary(t, evsA))
	t.Logf("S6 B persisted: %s", laSummary(t, evsB))
	t.Logf("S6 handler exec during drain: a=%d b=%d", ha.calls.Load(), hb.calls.Load())

	if !laHas(evsA, core.EventTypeStepCompleted, "a") {
		t.Errorf("FR-RM-04: in-flight step a of A must complete (StepCompleted a); persisted: %s", laSummary(t, evsA))
	}
	if laHasType(evsB, core.EventTypeStepStarted) {
		t.Errorf("FR-RM-04: no new steps dispatched after stop; B has a StepStarted: %s", laSummary(t, evsB))
	}
	if ha.calls.Load() != 1 || hb.calls.Load() != 0 {
		t.Errorf("exactly one handler execution (A.a) during the drain; got a=%d b=%d", ha.calls.Load(), hb.calls.Load())
	}

	// Restart: a NEW engine on the same storage, fresh handlers, live ctx.
	ha2 := &laHandler{id: "a", outputs: map[string]any{"r": "a"}}
	hb2 := &laHandler{id: "b", outputs: map[string]any{"r": "b"}}
	e2 := laNewEngine(s, "la-s6-2", map[core.StepType]Runner{core.StepTypeNative: laNativeRunner(ha2, hb2)}, laLogger(&laLog{}))
	for i := 0; i < 200; i++ {
		if err := e2.Tick(ctx); err != nil {
			t.Fatalf("restart Tick: %v", err)
		}
		instA, errA := s.GetInstance(ctx, iidA)
		instB, errB := s.GetInstance(ctx, iidB)
		if errA != nil || errB != nil {
			t.Fatalf("GetInstance: A=%v B=%v", errA, errB)
		}
		if isTerminal(instA.Status) && isTerminal(instB.Status) {
			break
		}
	}
	for _, iid := range []core.InstanceID{iidA, iidB} {
		inst, err := s.GetInstance(ctx, iid)
		if err != nil {
			t.Fatalf("GetInstance: %v", err)
		}
		if inst.Status != core.InstanceStatusCompleted {
			t.Errorf("restart must complete instance %s; status = %q", iid, inst.Status)
		}
	}
	// A.a was persisted before the stop, so it is NOT re-executed; only B.a runs
	// on the restarted engine. Each instance runs b once.
	if ha2.calls.Load() != 1 {
		t.Errorf("after restart only B.a runs (A.a is persisted); a executed %d times", ha2.calls.Load())
	}
	if hb2.calls.Load() != 2 {
		t.Errorf("after restart b runs once per instance; executed %d times", hb2.calls.Load())
	}
}

// TestLA_Shutdown_S7_TickPublicSemanticsUnchanged pins the public Tick contract
// that LA-C2 must not alter. Observable behavior of Tick(ctx) with an already
// cancelled ctx, before and after LA-C2: the first storage call fails with
// context.Canceled, Tick returns an error wrapping it, and nothing is dispatched
// or persisted (no StepStarted, handler never called, instance still running).
// A later Tick on a live ctx then completes the workflow: a cancelled Tick leaves
// no residue.
func TestLA_Shutdown_S7_TickPublicSemanticsUnchanged(t *testing.T) {
	s := laShutdownSetup(t, "la.s7")
	ha := &laHandler{id: "a", outputs: map[string]any{"r": "a"}}
	hb := &laHandler{id: "b", outputs: map[string]any{"r": "b"}}
	e := laNewEngine(s, "la-s7", map[core.StepType]Runner{core.StepTypeNative: laNativeRunner(ha, hb)}, laLogger(&laLog{}))

	ctx := context.Background()
	iid := laShutdownSubmit(t, e, "la.s7")

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	err := e.Tick(cancelled)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Tick(cancelledCtx) must return an error wrapping context.Canceled; got %v", err)
	}

	evs := laShutdownEvents(t, s, iid)
	t.Logf("S7 persisted after cancelled Tick: %s", laSummary(t, evs))
	laAssertIdle(t, s, ha, iid)
	if hb.calls.Load() != 0 {
		t.Errorf("cancelled Tick must not execute handler b; executed %d times", hb.calls.Load())
	}

	if err := laDriveTicks(ctx, e, s, iid); err != nil {
		t.Fatalf("live Tick after cancelled Tick: %v", err)
	}
	inst, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if inst.Status != core.InstanceStatusCompleted {
		t.Errorf("live ticks must complete the instance after a cancelled Tick; status = %q", inst.Status)
	}
}

// TestLA_Shutdown_S8_GateA_TickEntryStop — gate (a). A tick entered with stop
// already closed does nothing: not even the read-only ListInstances runs, so no
// SCAN_TRIGGERABLE, no SCAN_RUNNABLE, no SIGNAL_SCAN, no SIGNAL_TIMEOUT_SCAN.
func TestLA_Shutdown_S8_GateA_TickEntryStop(t *testing.T) {
	s, g, ha, e := laGateSetup(t, "la.s8")
	iid := laShutdownSubmit(t, e, "la.s8")

	stop, closeStop := laStopOnce()
	closeStop()
	if err := e.tick(context.Background(), stop); err != nil {
		t.Fatalf("stopped tick: %v", err)
	}

	if n := g.listInstanceCalls.Load(); n != 0 {
		t.Errorf("gate (a): stopped tick must not list instances; ListInstances called %d times", n)
	}
	if n := g.undeliveredScans.Load(); n != 0 {
		t.Errorf("gate (a): stopped tick must not run SIGNAL_SCAN; scans=%d", n)
	}
	if n := g.dueTimeoutScans.Load(); n != 0 {
		t.Errorf("gate (a): stopped tick must not run SIGNAL_TIMEOUT_SCAN; scans=%d", n)
	}
	laAssertIdle(t, s, ha, iid)
}

// TestLA_Shutdown_S9_GateB_StopAfterListing — gate (b) and (d-1). Stop lands
// right after ListInstances returns the running instance. The tick must not enter
// processInstance for it (no GetInstance after the stop), and must not run
// SIGNAL_SCAN or SIGNAL_TIMEOUT_SCAN.
func TestLA_Shutdown_S9_GateB_StopAfterListing(t *testing.T) {
	s, g, ha, e := laGateSetup(t, "la.s9")
	iid := laShutdownSubmit(t, e, "la.s9")

	stop, closeStop := laStopOnce()
	var getsAtListing int32
	g.onListInstances = func() {
		getsAtListing = g.getInstanceCalls.Load()
		closeStop()
	}
	if err := e.tick(context.Background(), stop); err != nil {
		t.Fatalf("stopped tick: %v", err)
	}

	if n := g.getInstanceCalls.Load(); n != getsAtListing {
		t.Errorf("gate (b): no processInstance may start after stop; GetInstance calls after listing = %d", n-getsAtListing)
	}
	if n := g.undeliveredScans.Load(); n != 0 {
		t.Errorf("gate (d-1): SIGNAL_SCAN must be skipped once stop is closed; scans=%d", n)
	}
	if n := g.dueTimeoutScans.Load(); n != 0 {
		t.Errorf("gate (d-2): SIGNAL_TIMEOUT_SCAN must be skipped once stop is closed; scans=%d", n)
	}
	laAssertIdle(t, s, ha, iid)
}

// TestLA_Shutdown_S10_GateC_StopInsideProcessInstance — gate (c). processInstance
// IS entered (its first storage read is GetInstance), and stop lands during that
// read. The tick must then return before gatherDispatch: no claim, no StepStarted,
// no handler execution. The instance is not failed by the withheld dispatch.
func TestLA_Shutdown_S10_GateC_StopInsideProcessInstance(t *testing.T) {
	s, g, ha, e := laGateSetup(t, "la.s10")
	iid := laShutdownSubmit(t, e, "la.s10")

	stop, closeStop := laStopOnce()
	g.onGetInstance = closeStop
	if err := e.tick(context.Background(), stop); err != nil {
		t.Fatalf("stopped tick must return nil for a withheld dispatch; got %v", err)
	}

	if n := g.getInstanceCalls.Load(); n == 0 {
		t.Fatalf("precondition: processInstance must be entered (GetInstance reached storage); it was not")
	}
	if n := g.undeliveredScans.Load(); n != 0 {
		t.Errorf("gate (d-1): SIGNAL_SCAN must be skipped once stop is closed; scans=%d", n)
	}
	if n := g.dueTimeoutScans.Load(); n != 0 {
		t.Errorf("gate (d-2): SIGNAL_TIMEOUT_SCAN must be skipped once stop is closed; scans=%d", n)
	}
	laAssertIdle(t, s, ha, iid)
}

// TestLA_Shutdown_S11_GateD2_StopBetweenSignalScans — gate (d-2). No instance is
// submitted, so SCAN_RUNNABLE is empty and SIGNAL_SCAN is reached with stop still
// open (gate d-1 open: the scan must run). Stop lands during SIGNAL_SCAN's storage
// call; SIGNAL_TIMEOUT_SCAN must then be skipped.
func TestLA_Shutdown_S11_GateD2_StopBetweenSignalScans(t *testing.T) {
	_, g, _, e := laGateSetup(t, "la.s11")

	stop, closeStop := laStopOnce()
	g.onListUndelivered = closeStop
	if err := e.tick(context.Background(), stop); err != nil {
		t.Fatalf("stopped tick: %v", err)
	}

	if n := g.undeliveredScans.Load(); n != 1 {
		t.Errorf("gate (d-1): SIGNAL_SCAN must run while stop is open; scans=%d", n)
	}
	if n := g.dueTimeoutScans.Load(); n != 0 {
		t.Errorf("gate (d-2): SIGNAL_TIMEOUT_SCAN must be skipped once stop closed during SIGNAL_SCAN; scans=%d", n)
	}
}
