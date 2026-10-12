package engine

// LA-P2-REPRO — deterministic reproducers for shutdown / cancellation with
// in-flight work (Loop A Priority 2). REPRODUCERS ONLY: this file adds no
// production code and fixes nothing. Each test asserts the frozen contract and
// records what is actually persisted. A FAIL here is the deliverable (the
// failing baseline), not a defect in this file.
//
// Contract text (quoted verbatim; repeated where each test relies on it):
//
//	PRD FR-RM-04 (AWIS_PRD.md:382):
//	  "`awis stop` gracefully stops the runtime (in-flight steps complete; no new steps dispatched)"
//	PRD FR-WE-08 (AWIS_PRD.md:399):
//	  "Cancellation is graceful: in-flight steps complete; no next step is activated after cancellation_requested is set"
//	docs/CLI_CONTRACT.md §SIGTERM Handling (TDS-07, lines 49-53):
//	  "Context cancel → engine tick completes in-flight step → graceful drain → exit 0"
//	  "FR-RM-06: graceful shutdown is the contract; kill -9 (SIGKILL) skips drain (crash path)"
//	AWIS_ARCHITECTURE_BLUEPRINT.md:598-600:
//	  "In-flight steps (already dispatched) run to completion. Their `StepCompleted` or `StepFailed` events are appended normally. The runtime does not kill step goroutines or processes."
//	  "If set, no next step is activated regardless of transition conditions."
//	  "Without `--compensate` (default): instance transitions to `cancelled` after in-flight steps settle."

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// laHang is the hang guard for every cross-goroutine wait. Synchronization is
// by channel only; the guard exists so a regression that never signals fails
// loudly instead of wedging the suite.
const laHang = 10 * time.Second

// laSignal is a one-shot "step handler entered" signal.
type laSignal struct {
	once sync.Once
	ch   chan struct{}
}

func newLASignal() *laSignal { return &laSignal{ch: make(chan struct{})} }

func (s *laSignal) fire() { s.once.Do(func() { close(s.ch) }) }

func (s *laSignal) wait(t *testing.T) {
	t.Helper()
	select {
	case <-s.ch:
	case <-time.After(laHang):
		t.Fatalf("hang guard: step handler not entered within %s", laHang)
	}
}

// laGate holds a step handler inside Execute until open() is called. open is
// registered as a Cleanup so a failing test never leaves a handler blocked.
type laGate struct {
	once sync.Once
	ch   chan struct{}
}

func newLAGate(t *testing.T) *laGate {
	t.Helper()
	g := &laGate{ch: make(chan struct{})}
	t.Cleanup(g.open)
	return g
}

func (g *laGate) open() { g.once.Do(func() { close(g.ch) }) }

// laHandler is a core.StepHandler dispatched through the native runner. Its
// Execute takes NO context (core.StepHandler has no ctx parameter), so it cannot
// observe the run ctx or cancellation at all; it only blocks on its gate.
type laHandler struct {
	id      string
	calls   atomic.Int32
	entered *laSignal // nil => no entry signal
	gate    *laGate   // nil => return immediately
	outputs map[string]any
}

func (h *laHandler) ID() string { return h.id }

func (h *laHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	h.calls.Add(1)
	if h.entered != nil {
		h.entered.fire()
	}
	if h.gate != nil {
		<-h.gate.ch
	}
	return core.StepResult{Outputs: h.outputs}, nil
}

// laCtxRunner is an engine.Runner. Unlike core.StepHandler, engine.Runner.Run
// RECEIVES the dispatch ctx (processInstance → dispatchOne → runInvoke →
// Runner.Run). It is the only way a test-level step can observe ctx at V1; the
// native runner consumes ctx only in its own deadline select and never forwards
// it to the handler. Step "a" blocks on its gate while IGNORING ctx, then
// returns a StepError carrying ctx.Err() if ctx is done at the completion point
// (the card's S2 "handler honors ctx" harness: cancel lands while a is blocked,
// the gate opens afterwards). Every other step delegates to inner.
type laCtxRunner struct {
	inner   *native.NativeRunner
	entered *laSignal
	gate    *laGate
	aCalls  atomic.Int32
}

func (r *laCtxRunner) Run(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.StepError) {
	if string(step.Handler) != "a" {
		return r.inner.Run(ctx, sc, step)
	}
	r.aCalls.Add(1)
	r.entered.fire()
	<-r.gate.ch
	if err := ctx.Err(); err != nil {
		return core.StepResult{}, &core.StepError{Code: "ctx_done", Message: err.Error()}
	}
	return core.StepResult{Outputs: map[string]any{"r": "a"}}, nil
}

// laLog is a mutex-guarded sink for the engine's slog output. The Run relay
// writes from its own goroutine, so an unguarded buffer would race.
type laLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *laLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

// diagnostics returns the ERROR and WARN lines the engine logged (per-instance
// failure, "tick failed", signal-scan warnings).
func (l *laLog) diagnostics() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, line := range strings.Split(l.buf.String(), "\n") {
		if strings.Contains(line, "level=ERROR") || strings.Contains(line, "level=WARN") {
			out = append(out, line)
		}
	}
	return out
}

func laLogger(l *laLog) *slog.Logger {
	return slog.New(slog.NewTextHandler(l, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// laDef is the card's workflow: native "a" → native "b" (final).
func laDef(id string) core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: id, Version: "1.0.0", Namespace: "la", Name: id,
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("a", "a"), nativeStep("b", "b")},
		Transitions: []core.Transition{{From: "a", To: "b"}},
		InitialStep: "a", FinalSteps: []string{"b"}, Metadata: map[string]any{},
	}
}

// laSetup opens a fresh storage (openStorage: per-test temp DB) and registers laDef.
func laSetup(t *testing.T, id string) *storage.SQLiteStorage {
	t.Helper()
	s := openStorage(t)
	if err := s.RegisterWorkflow(context.Background(), laDef(id)); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	return s
}

// laNewEngine builds an engine on s with the fake clock (determinism) and a
// 1ms Run ticker (no sleeps: Run's ticker is the only real-time element).
func laNewEngine(s *storage.SQLiteStorage, workerID string, runners map[core.StepType]Runner, logger *slog.Logger) *Engine {
	return New(s, runners, Config{
		TickInterval:     time.Millisecond,
		MaxParallelSteps: 1,
		WorkerID:         workerID,
		Clock:            newFakeClock(engineStart, 10*time.Millisecond).now,
	}, logger)
}

// laAwaitErr receives one error from done with the hang guard.
func laAwaitErr(t *testing.T, done <-chan error, what string) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(laHang):
		t.Fatalf("hang guard: %s did not return within %s", what, laHang)
		return nil
	}
}

// laDriveTicks calls Tick (no ticker) until the instance is terminal or the
// bounded guard trips. It never calls testing methods, so it may run off the
// test goroutine.
func laDriveTicks(ctx context.Context, e *Engine, s *storage.SQLiteStorage, iid core.InstanceID) error {
	for i := 0; i < 200; i++ {
		if err := e.Tick(ctx); err != nil {
			return err
		}
		inst, err := s.GetInstance(ctx, iid)
		if err != nil {
			return err
		}
		if isTerminal(inst.Status) {
			return nil
		}
	}
	return fmt.Errorf("instance %s not terminal within 200 ticks", iid)
}

// laEvents reads the full persisted stream for iid.
func laEvents(t *testing.T, s *storage.SQLiteStorage, iid core.InstanceID) []core.ExecutionEvent {
	t.Helper()
	evs, err := s.ReadEvents(context.Background(), iid, 0)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	return evs
}

// laHas reports whether a (type, step) event is persisted; step "" matches the
// workflow-level event.
func laHas(evs []core.ExecutionEvent, et core.EventType, step string) bool {
	for _, ev := range evs {
		if ev.EventType == et && ev.StepID == step {
			return true
		}
	}
	return false
}

// laSummary renders the persisted stream in append order. StepFailed and
// WorkflowFailed carry their error code; WorkflowCancelled carries its reason.
func laSummary(t *testing.T, evs []core.ExecutionEvent) string {
	t.Helper()
	parts := make([]string, 0, len(evs))
	for _, ev := range evs {
		p := string(ev.EventType)
		if ev.StepID != "" {
			p += "(" + ev.StepID + ")"
		}
		switch ev.EventType {
		case core.EventTypeStepFailed:
			var sp stepFailedPayload
			decodePayload(t, ev, &sp)
			p += "[code=" + sp.Error.Code + " retrying=" + strconv.FormatBool(sp.Retrying) + "]"
		case core.EventTypeWorkflowFailed:
			var wp workflowFailedPayload
			decodePayload(t, ev, &wp)
			p += "[code=" + wp.Error.Code + " step=" + wp.StepID + "]"
		case core.EventTypeWorkflowCancelled:
			var cp workflowCancelledPayload
			decodePayload(t, ev, &cp)
			p += "[reason=" + cp.Reason + "]"
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, "; ")
}

// laShutdownRun is the observable outcome of one Run-shutdown with a in flight.
type laShutdownRun struct {
	s      *storage.SQLiteStorage
	iid    core.InstanceID
	runErr error
	logs   *laLog
	aCalls *atomic.Int32
	bCalls *atomic.Int32
}

// laRunShutdown drives the card's S1/S2 harness. Run(runCtx) starts with a
// short ticker; when a's step signals entered (a is in flight inside DISPATCH),
// runCtx is cancelled; only then is a's gate opened; the test then waits for Run
// to return (hang-guarded). ctxAware=false is S1 (native StepHandler, which
// cannot see ctx); ctxAware=true is S2 (laCtxRunner, which receives ctx).
func laRunShutdown(t *testing.T, ctxAware bool) laShutdownRun {
	t.Helper()
	s := laSetup(t, "la.shutdown")
	entered := newLASignal()
	gate := newLAGate(t)

	hb := &laHandler{id: "b", outputs: map[string]any{"r": "b"}}
	nr := native.New()
	nr.Register(hb)

	var aCalls *atomic.Int32
	runners := map[core.StepType]Runner{}
	if ctxAware {
		cr := &laCtxRunner{inner: nr, entered: entered, gate: gate}
		runners[core.StepTypeNative] = cr
		aCalls = &cr.aCalls
	} else {
		ha := &laHandler{id: "a", entered: entered, gate: gate, outputs: map[string]any{"r": "a"}}
		nr.Register(ha)
		runners[core.StepTypeNative] = nr
		aCalls = &ha.calls
	}

	lg := &laLog{}
	e := laNewEngine(s, "la-repro-1", runners, laLogger(lg))
	ctx := context.Background()
	iid, err := e.Submit(ctx, "la.shutdown", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	t.Cleanup(cancelRun)
	runDone := make(chan error, 1)
	go func() { runDone <- e.Run(runCtx) }()

	entered.wait(t) // a is in flight; Run is blocked inside Tick's DISPATCH
	cancelRun()     // context cancel while a is in flight (the SIGTERM path)
	gate.open()     // only now may a's handler return

	runErr := laAwaitErr(t, runDone, "Run")
	return laShutdownRun{s: s, iid: iid, runErr: runErr, logs: lg, aCalls: aCalls, bCalls: &hb.calls}
}

// TestLA_Shutdown_S1_RunCancelHandlerIgnoresCtx — S1.
//
// FR-RM-04: "in-flight steps complete; no new steps dispatched". FR-WE-08:
// "in-flight steps complete; no next step is activated after cancellation_requested
// is set". Handler "a" ignores ctx and completes only after release.
// Expected on current code: FAIL (see report for the persisted stream).
func TestLA_Shutdown_S1_RunCancelHandlerIgnoresCtx(t *testing.T) {
	r := laRunShutdown(t, false)
	evs := laEvents(t, r.s, r.iid)

	t.Logf("S1 persisted: %s", laSummary(t, evs))
	t.Logf("S1 Run returned: %v", r.runErr)
	t.Logf("S1 handler exec count: a=%d b=%d", r.aCalls.Load(), r.bCalls.Load())
	for _, d := range r.logs.diagnostics() {
		t.Logf("S1 engine log: %s", d)
	}

	if !errors.Is(r.runErr, context.Canceled) {
		t.Errorf("Run must return ctx.Err() on cancel (tick.go Run doc); got %v", r.runErr)
	}
	if !laHas(evs, core.EventTypeStepCompleted, "a") {
		t.Errorf("FR-RM-04/FR-WE-08: in-flight step a must complete (StepCompleted a); persisted: %s", laSummary(t, evs))
	}
	if laHas(evs, core.EventTypeStepStarted, "b") {
		t.Errorf("FR-RM-04: no new steps dispatched after stop; StepStarted b persisted: %s", laSummary(t, evs))
	}
}

// TestLA_Shutdown_S2_RunCancelHandlerHonorsCtx — S2.
//
// Same harness as S1, but the step is a engine.Runner that receives ctx and
// returns ctx.Err() as a StepError when ctx is done at its completion point.
//
// Design caveat recorded for the report: under this harness ctx is ALWAYS done
// by the time a's gate opens, so a StepCompleted for a is unreachable by the
// handler's own construction. The StepCompleted check is kept verbatim from the
// card. The engine-attributable check is separate: the in-flight step a must
// leave SOME terminal event (StepCompleted or StepFailed) persisted, otherwise
// current_steps stays dangling with a bare StepStarted.
func TestLA_Shutdown_S2_RunCancelHandlerHonorsCtx(t *testing.T) {
	r := laRunShutdown(t, true)
	evs := laEvents(t, r.s, r.iid)

	t.Logf("S2 persisted: %s", laSummary(t, evs))
	t.Logf("S2 Run returned: %v", r.runErr)
	t.Logf("S2 handler exec count: a=%d b=%d", r.aCalls.Load(), r.bCalls.Load())
	for _, d := range r.logs.diagnostics() {
		t.Logf("S2 engine log: %s", d)
	}

	if !laHas(evs, core.EventTypeStepCompleted, "a") {
		t.Errorf("FR-RM-04/FR-WE-08 (card S2 literal): StepCompleted for a after release post-cancel; persisted: %s", laSummary(t, evs))
	}
	if !laHas(evs, core.EventTypeStepCompleted, "a") && !laHas(evs, core.EventTypeStepFailed, "a") {
		t.Errorf("engine-attributable: in-flight step a has NO terminal event persisted (dangling StepStarted); persisted: %s", laSummary(t, evs))
	}
	if laHas(evs, core.EventTypeStepStarted, "b") {
		t.Errorf("FR-RM-04: no new steps dispatched after stop; StepStarted b persisted: %s", laSummary(t, evs))
	}
}

// TestLA_Shutdown_S3_RestartAfterS1Shutdown — S3.
//
// Reproduces S1's shutdown exactly, then constructs a NEW Engine on the same
// storage and ticks it up to 50 times with the fake clock. Correct behaviour:
// the instance completes and handler a executes exactly once across S1+S3
// (its in-flight result is persisted and cached under instance/step/attempt).
func TestLA_Shutdown_S3_RestartAfterS1Shutdown(t *testing.T) {
	r := laRunShutdown(t, false)

	ha2 := &laHandler{id: "a", outputs: map[string]any{"r": "a"}}
	hb2 := &laHandler{id: "b", outputs: map[string]any{"r": "b"}}
	nr2 := native.New()
	nr2.Register(ha2)
	nr2.Register(hb2)
	lg2 := &laLog{}
	e2 := laNewEngine(r.s, "la-repro-2", map[core.StepType]Runner{core.StepTypeNative: nr2}, laLogger(lg2))

	ctx := context.Background()
	var tickErrs []string
	for i := 0; i < 50; i++ {
		if err := e2.Tick(ctx); err != nil {
			tickErrs = append(tickErrs, err.Error())
		}
		inst, err := r.s.GetInstance(ctx, r.iid)
		if err != nil {
			t.Fatalf("GetInstance: %v", err)
		}
		if isTerminal(inst.Status) {
			break
		}
	}

	inst, err := r.s.GetInstance(ctx, r.iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	evs := laEvents(t, r.s, r.iid)
	total := r.aCalls.Load() + ha2.calls.Load()

	t.Logf("S3 final instance status: %s", inst.Status)
	t.Logf("S3 persisted: %s", laSummary(t, evs))
	t.Logf("S3 handler a exec total across S1+S3: %d (S1=%d, S3=%d)", total, r.aCalls.Load(), ha2.calls.Load())
	t.Logf("S3 Tick errors: %v", tickErrs)
	for _, d := range lg2.diagnostics() {
		t.Logf("S3 engine log: %s", d)
	}

	if inst.Status != core.InstanceStatusCompleted {
		t.Errorf("restart must complete the instance; final status = %q", inst.Status)
	}
	if total != 1 {
		t.Errorf("handler a must execute exactly once across S1+S3 (cached result); executed %d times", total)
	}
}

// TestLA_Cancel_S4_InFlightCancelCompletesStepThenCancels — S4.
//
// FR-WE-08: "Cancellation is graceful: in-flight steps complete; no next step is
// activated after cancellation_requested is set". Blueprint: "Without
// `--compensate` (default): instance transitions to `cancelled` after in-flight
// steps settle." Cancel is issued while a's handler is blocked inside Execute;
// no ctx cancellation is involved, so this is the cancellation path proper.
func TestLA_Cancel_S4_InFlightCancelCompletesStepThenCancels(t *testing.T) {
	s := laSetup(t, "la.cancel")
	entered := newLASignal()
	gate := newLAGate(t)

	ha := &laHandler{id: "a", entered: entered, gate: gate, outputs: map[string]any{"r": "a"}}
	hb := &laHandler{id: "b", outputs: map[string]any{"r": "b"}}
	nr := native.New()
	nr.Register(ha)
	nr.Register(hb)
	lg := &laLog{}
	e := laNewEngine(s, "la-repro-cancel", map[core.StepType]Runner{core.StepTypeNative: nr}, laLogger(lg))

	ctx := context.Background()
	iid, err := e.Submit(ctx, "la.cancel", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	driveDone := make(chan error, 1)
	go func() { driveDone <- laDriveTicks(ctx, e, s, iid) }()

	entered.wait(t) // a is in flight inside DISPATCH
	if err := e.Cancel(ctx, iid, "x", false); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	gate.open() // a's handler may now complete

	if err := laAwaitErr(t, driveDone, "tick driver"); err != nil {
		t.Fatalf("drive ticks: %v", err)
	}

	evs := laEvents(t, s, iid)
	inst, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	t.Logf("S4 persisted: %s", laSummary(t, evs))
	t.Logf("S4 final instance status: %s; handler exec a=%d b=%d", inst.Status, ha.calls.Load(), hb.calls.Load())
	for _, d := range lg.diagnostics() {
		t.Logf("S4 engine log: %s", d)
	}

	if !laHas(evs, core.EventTypeStepCompleted, "a") {
		t.Errorf("FR-WE-08: in-flight step a must complete (StepCompleted a); persisted: %s", laSummary(t, evs))
	}
	if laHas(evs, core.EventTypeStepStarted, "b") {
		t.Errorf("FR-WE-08: no next step activated after cancellation_requested; StepStarted b persisted: %s", laSummary(t, evs))
	}
	if inst.Status != core.InstanceStatusCancelled {
		t.Errorf("Blueprint: instance must be cancelled after in-flight steps settle; final status = %q", inst.Status)
	}
	if !laHas(evs, core.EventTypeWorkflowCancelled, "") {
		t.Errorf("WorkflowCancelled must be persisted; persisted: %s", laSummary(t, evs))
	}
	assertProjectionEquivalence(t, s, iid)
}
