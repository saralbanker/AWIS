package engine

// LA-C3 (Loop A Priority 1; leads L7, L8; L9 investigate-only): committed
// terminal-failure routing must be completed after a transient append failure.
//
// Routing order (EDR-011 §8): (1) fallback wins; (2) on_error transitions; (3)
// WorkflowFailed + compensation (compensation.go). StepFailed{retrying:false} is the
// committed routing decision; the follow-on emission (StepFallbackActivated for (1),
// WorkflowFailed for (3)) is what a single transient fault can lose.
//
// Each lead has a same-process test (laK further Ticks on the faulted engine, after
// Tick's LA-C1 F1 reset) and an _AfterRestart test (a fresh Engine on the same
// storage). The expected behavior for L7/L8 is the no-fault event sequence: the
// faulted and recovered run must emit the same (event_type, step_id) pairs as a run
// with no fault. TEST-ONLY: no production code in this file.

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// laDefRoute builds a workflow whose initial step "a" (handler la_a) carries the
// given retry policy and, when fallback is set, Fallback "fb" (handler la_fb, the
// sole final step). Without fallback, "a" is the sole final step.
func laDefRoute(id string, retry *core.RetryPolicy, fallback bool) core.WorkflowDefinition {
	a := core.Step{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "la_a", Retry: retry}
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: id, Version: "1.0.0", Namespace: "la", Name: id,
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	if !fallback {
		def.Steps = []core.Step{a}
		return def
	}
	a.Fallback = "fb"
	def.Steps = []core.Step{a, nativeStep("fb", "la_fb")}
	def.FinalSteps = []string{"fb"}
	return def
}

// laFallbackHandlers returns fresh handlers for laDefRoute: a always fails with a
// handler error (no retry unless the definition sets one); fb completes.
func laFallbackHandlers() (*stepHandler, *stepHandler) {
	return &stepHandler{id: "la_a", err: errLAHandler},
		&stepHandler{id: "la_fb", outputs: map[string]any{"r": "recovered"}}
}

// laCompRun is one compDef() run's handler set: a, b, c complete; d fails with
// errBoom (route 3); undo-a/b/c/x record their call order (compensation.go).
type laCompRun struct {
	mu       sync.Mutex
	order    []string
	hd       *stepHandler
	handlers []core.StepHandler
}

func newLACompRun() *laCompRun {
	r := &laCompRun{hd: &stepHandler{id: "hd", err: errBoom}}
	r.handlers = []core.StepHandler{
		&stepHandler{id: "ha", outputs: map[string]any{"r": "a"}},
		&stepHandler{id: "hb", outputs: map[string]any{"r": "b"}},
		&stepHandler{id: "hc", outputs: map[string]any{"r": "c"}},
		r.hd,
		&recordingHandler{id: "undo-a", order: &r.order, mu: &r.mu},
		&recordingHandler{id: "undo-b", order: &r.order, mu: &r.mu},
		&recordingHandler{id: "undo-c", order: &r.order, mu: &r.mu},
		&recordingHandler{id: "undo-x", order: &r.order, mu: &r.mu},
	}
	return r
}

// undoOrder returns the undo handlers' call order, joined by commas.
func (r *laCompRun) undoOrder() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := ""
	for i, id := range r.order {
		if i > 0 {
			out += ","
		}
		out += id
	}
	return out
}

// laDrive registers def on a fresh store, submits one instance and drives it to a
// terminal status within laK Ticks after the optional fault. ft == nil is the
// no-fault baseline. With ft set, the fault trips on its first matching call; restart
// then drops the faulted engine for a fresh one on the same storage (a process
// restart). Handlers are shared by every engine of the run, so their call counts
// span the whole run.
func laDrive(t *testing.T, def core.WorkflowDefinition, ft *laFault, restart bool, handlers ...core.StepHandler) (*storage.SQLiteStorage, core.InstanceID, int, []error) {
	t.Helper()
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 1, handlers...)
	iid := laSubmit(t, e, def.ID)
	var errs []error
	if ft != nil {
		fs.arm(*ft)
		errs = laTickUntilTripped(t, e, fs)
		laWantInjected(t, errs)
		if restart {
			e = laEngine(fs, clk, ids, 1, handlers...)
		}
	}
	n, more := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	return s, iid, n, append(errs, more...)
}

// laDiffPairs asserts the faulted run's (event_type, step_id) sequence equals the
// no-fault baseline's (LA-C3 card: "same event types in the same order").
func laDiffPairs(t *testing.T, label string, want, got []pair) {
	t.Helper()
	if fmt.Sprint(want) != fmt.Sprint(got) {
		t.Errorf("%s: event sequence differs from the no-fault path:\n  want=%v\n  got =%v", label, want, got)
	}
}

// laCount returns how many events of type et are in evs.
func laCount(evs []core.ExecutionEvent, et core.EventType) int {
	n := 0
	for _, ev := range evs {
		if ev.EventType == et {
			n++
		}
	}
	return n
}

// ── L7 route (1): StepFailed{a, retrying:false} committed, then the
// StepFallbackActivated(a) append fails once. Expected: fb activates and runs once,
// the instance completes, and the sequence equals the no-fault path's.

func laScenarioL7(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("L7", restart)
	def := laDefRoute("la.fallback", nil, true)
	ft := laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepFallbackActivated, stepID: "a"}

	ba, bfb := laFallbackHandlers()
	bs, biid, _, _ := laDrive(t, def, nil, false, ba, bfb)
	want, _ := eventPairs(t, bs, biid)

	ha, hfb := laFallbackHandlers()
	s, iid, n, errs := laDrive(t, def, &ft, restart, ha, hfb)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count a=%d fb=%d", label, n, len(errs), laCalls(ha), laCalls(hfb))
	laCheck(t, s, iid, label, core.InstanceStatusCompleted,
		laExpect{"a", "StepFailed(code=handler_error,retrying=false)"},
		laExpect{"fb", "StepCompleted"})
	if got := laCalls(ha); got != 1 {
		t.Errorf("%s: step a executions = %d, want 1 (no retry in this fixture)", label, got)
	}
	if got := laCalls(hfb); got != 1 {
		t.Errorf("%s: fallback fb executions = %d, want exactly 1", label, got)
	}
	evs := laEvents(t, s, iid)
	if got := laCount(evs, core.EventTypeStepFallbackActivated); got != 1 {
		t.Errorf("%s: StepFallbackActivated count = %d, want exactly 1", label, got)
	}
	for _, ev := range evs {
		if ev.EventType != core.EventTypeStepFallbackActivated {
			continue
		}
		var p stepFallbackActivatedPayload
		decodePayload(t, ev, &p)
		if p.StepID != "a" || p.FallbackStepID != "fb" || p.Reason != "handler_error" {
			t.Errorf("%s: StepFallbackActivated payload = %+v, want step a -> fb reason handler_error", label, p)
		}
	}
	got, _ := eventPairs(t, s, iid)
	laDiffPairs(t, label, want, got)
}

func TestLA_L7_FallbackStepFailed_FallbackAppendFails(t *testing.T) {
	laScenarioL7(t, false)
}

func TestLA_L7_FallbackStepFailed_FallbackAppendFails_AfterRestart(t *testing.T) {
	laScenarioL7(t, true)
}

// ── L8 route (3): StepFailed{d, retrying:false} committed, then the WorkflowFailed
// append fails once. Expected: WorkflowFailed carries the ORIGINAL failing step id
// and error code (not "stalled"), compensation runs, and the sequence equals the
// no-fault path's (compDef: a→b→c→d, d fails, plan [a,b,c,x]).

// laWorkflowFailedPayloads decodes every WorkflowFailed payload in evs.
func laWorkflowFailedPayloads(t *testing.T, evs []core.ExecutionEvent) []workflowFailedPayload {
	t.Helper()
	var out []workflowFailedPayload
	for _, ev := range evs {
		if ev.EventType != core.EventTypeWorkflowFailed {
			continue
		}
		var p workflowFailedPayload
		decodePayload(t, ev, &p)
		out = append(out, p)
	}
	return out
}

func laScenarioL8(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("L8", restart)
	def := compDef()
	ft := laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowFailed}

	base := newLACompRun()
	bs, biid, _, _ := laDrive(t, def, nil, false, base.handlers...)
	want, _ := eventPairs(t, bs, biid)

	r := newLACompRun()
	s, iid, n, errs := laDrive(t, def, &ft, restart, r.handlers...)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count d=%d undo_order=%s",
		label, n, len(errs), laCalls(r.hd), r.undoOrder())
	laCheck(t, s, iid, label, core.InstanceStatusCompensated,
		laExpect{"d", "StepFailed(code=handler_error,retrying=false)"})
	if got := laCalls(r.hd); got != 1 {
		t.Errorf("%s: step d executions = %d, want 1", label, got)
	}
	if got := r.undoOrder(); got != "undo-c,undo-b,undo-a" {
		t.Errorf("%s: undo order = %q, want %q (reverse, x skipped)", label, got, "undo-c,undo-b,undo-a")
	}
	evs := laEvents(t, s, iid)
	wfs := laWorkflowFailedPayloads(t, evs)
	if len(wfs) != 1 {
		t.Errorf("%s: WorkflowFailed count = %d, want exactly 1", label, len(wfs))
	}
	for _, p := range wfs {
		if p.StepID != "d" || p.Error.Code != "handler_error" {
			t.Errorf("%s: WorkflowFailed = step %q code %q, want step \"d\" code \"handler_error\" (not \"stalled\")",
				label, p.StepID, p.Error.Code)
		}
	}
	got, _ := eventPairs(t, s, iid)
	laDiffPairs(t, label, want, got)
}

func TestLA_L8_WorkflowFailedAppendFails(t *testing.T) {
	laScenarioL8(t, false)
}

func TestLA_L8_WorkflowFailedAppendFails_AfterRestart(t *testing.T) {
	laScenarioL8(t, true)
}

// ── L9a / L9b: DEFERRED (CE ruling LA-C3, binding). A compensation emission that fails
// once after WorkflowFailed is committed is not recovered by any path: L9a fails
// WorkflowCompensating (status ends failed, no compensation ever runs); L9b fails
// WorkflowCompensated after the undo handlers ran (status ends compensating). Recovery
// needs a scan of non-running instances, which is a new tick stage (design, not a
// fix-only change). These reproducers are gated: they run only when LA_DEFERRED=1, and
// they must FAIL on current code. See STATE.md LOOP-A BACKLOG.
//
// Expected (no-fault path): status compensated, undo order undo-c,undo-b,undo-a, and
// the same (event_type, step_id) sequence as the no-fault run.

func laScenarioL9(t *testing.T, restart bool, et core.EventType) {
	t.Helper()
	label := laLabel("L9 "+string(et), restart)
	base := newLACompRun()
	bs, biid, _, _ := laDrive(t, compDef(), nil, false, base.handlers...)
	want, _ := eventPairs(t, bs, biid)

	r := newLACompRun()
	s, iid, n, errs := laDrive(t, compDef(), &laFault{op: laOpAppendEvent, nth: 1, eventType: et}, restart, r.handlers...)
	t.Logf("%s: ticks=%d tick_errors=%d undo_order=%q", label, n, len(errs), r.undoOrder())
	laCheck(t, s, iid, label, core.InstanceStatusCompensated)
	if got := r.undoOrder(); got != "undo-c,undo-b,undo-a" {
		t.Errorf("%s: undo order = %q, want %q", label, got, "undo-c,undo-b,undo-a")
	}
	got, _ := eventPairs(t, s, iid)
	laDiffPairs(t, label, want, got)
}

func TestLA_L9a_CompensatingAppendFails(t *testing.T) {
	if os.Getenv("LA_DEFERRED") != "1" {
		t.Skip("LA-L9 deferred to backlog: compensation append failure — see STATE.md LOOP-A BACKLOG")
	}
	laScenarioL9(t, false, core.EventTypeWorkflowCompensating)
}

func TestLA_L9a_CompensatingAppendFails_AfterRestart(t *testing.T) {
	if os.Getenv("LA_DEFERRED") != "1" {
		t.Skip("LA-L9 deferred to backlog: compensation append failure — see STATE.md LOOP-A BACKLOG")
	}
	laScenarioL9(t, true, core.EventTypeWorkflowCompensating)
}

func TestLA_L9b_CompensatedAppendFails(t *testing.T) {
	if os.Getenv("LA_DEFERRED") != "1" {
		t.Skip("LA-L9 deferred to backlog: compensation append failure — see STATE.md LOOP-A BACKLOG")
	}
	laScenarioL9(t, false, core.EventTypeWorkflowCompensated)
}

func TestLA_L9b_CompensatedAppendFails_AfterRestart(t *testing.T) {
	if os.Getenv("LA_DEFERRED") != "1" {
		t.Skip("LA-L9 deferred to backlog: compensation append failure — see STATE.md LOOP-A BACKLOG")
	}
	laScenarioL9(t, true, core.EventTypeWorkflowCompensated)
}

// ── Guards (must pass before AND after any fix). Cancellation emits
// StepFailed{code:"cancelled", retrying:false} WITHOUT a routing decision
// (cancel.go handleCancellation: retry-wait cancellation). A route-recovery that
// treated such a step as route (1) or (3) would emit StepFallbackActivated or
// WorkflowFailed on a cancelled instance. Setup: a fails once with a retry policy
// (StepFailed{retrying:true}); Cancel while a awaits its retry; the WorkflowCancelled
// append then fails once. Expected: the instance ends cancelled, with no WorkflowFailed
// and no StepFallbackActivated, same-process and after restart.

func laScenarioCancelRetry(t *testing.T, withFallback, restart bool) {
	t.Helper()
	label := laLabel(fmt.Sprintf("LA-C3 guard cancel-retry fallback=%t", withFallback), restart)
	def := laDefRoute("la.cancel.retry", &core.RetryPolicy{Attempts: 2, Backoff: "immediate"}, withFallback)
	ha, hfb := laFallbackHandlers()
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 1, ha, hfb)
	iid := laSubmit(t, e, def.ID)
	ctx := context.Background()
	if err := e.Tick(ctx); err != nil { // a fails: StepFailed{retrying:true}, retry scheduled
		t.Fatalf("%s: setup Tick: %v", label, err)
	}
	if err := e.Cancel(ctx, iid, "la-c3 guard", false); err != nil {
		t.Fatalf("%s: Cancel: %v", label, err)
	}
	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowCancelled})
	laWantInjected(t, laTickUntilTripped(t, e, fs))
	if restart {
		e = laEngine(fs, clk, ids, 1, ha, hfb)
	}
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count a=%d fb=%d", label, n, len(errs), laCalls(ha), laCalls(hfb))
	laCheck(t, s, iid, label, core.InstanceStatusCancelled,
		laExpect{"a", "StepFailed(code=cancelled,retrying=false)"})
	evs := laEvents(t, s, iid)
	if laHasEvent(evs, core.EventTypeWorkflowFailed) {
		t.Errorf("%s: WorkflowFailed emitted on a cancelled instance (B4 finalizes as cancelled)", label)
	}
	if laHasEvent(evs, core.EventTypeStepFallbackActivated) {
		t.Errorf("%s: StepFallbackActivated emitted on a cancelled instance (B4 step 3: no new activation)", label)
	}
	if got := laCalls(ha); got != 1 {
		t.Errorf("%s: step a executions = %d, want 1", label, got)
	}
}

func TestLA_C3_Guard_RetryWaitCancel_NoRouting(t *testing.T) {
	laScenarioCancelRetry(t, false, false)
}

func TestLA_C3_Guard_RetryWaitCancel_NoRouting_AfterRestart(t *testing.T) {
	laScenarioCancelRetry(t, false, true)
}

func TestLA_C3_Guard_RetryWaitCancel_WithFallback_NoRouting(t *testing.T) {
	laScenarioCancelRetry(t, true, false)
}

func TestLA_C3_Guard_RetryWaitCancel_WithFallback_NoRouting_AfterRestart(t *testing.T) {
	laScenarioCancelRetry(t, true, true)
}

// ── Guard: a fallback activation that IS durable must not be emitted again on
// restart (no duplicate StepFallbackActivated; fb runs exactly once).

func TestLA_C3_Guard_FallbackDurable_NoDuplicateOnRestart(t *testing.T) {
	def := laDefRoute("la.fallback.dup", nil, true)
	ha, hfb := laFallbackHandlers()
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 1, ha, hfb)
	iid := laSubmit(t, e, def.ID)
	ctx := context.Background()
	if err := e.Tick(ctx); err != nil { // a fails; StepFallbackActivated(a) committed
		t.Fatalf("setup Tick: %v", err)
	}
	e = laEngine(fs, clk, ids, 1, ha, hfb) // restart after the fallback is durable
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("ticks=%d tick_errors=%d", n, len(errs))
	laCheck(t, s, iid, "fallback-durable restart", core.InstanceStatusCompleted,
		laExpect{"fb", "StepCompleted"})
	evs := laEvents(t, s, iid)
	if got := laCount(evs, core.EventTypeStepFallbackActivated); got != 1 {
		t.Errorf("StepFallbackActivated count = %d, want exactly 1", got)
	}
	if got := laCalls(hfb); got != 1 {
		t.Errorf("fallback fb executions = %d, want exactly 1", got)
	}
}

// laEngineManual builds an Engine on a manual clock (signal timeouts need to advance
// time explicitly; the fakeClock used elsewhere advances on every read).
func laEngineManual(st core.StoragePort, clk *manualClock, ids func() string, handlers ...core.StepHandler) *Engine {
	nr := native.New()
	for _, h := range handlers {
		nr.Register(h)
	}
	return New(st, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: 2, Clock: clk.now, NewID: ids}, discardLogger())
}

// ── Guard (c), CE ruling LA-C3 round 2: signal_timeout with a Fallback ──────────
// Signal step w (timeout 5s, timeout_action compensate or fail) has Fallback "fb".
// Its timeout commits StepFailed{code:"signal_timeout", retrying:false}; then one
// transient fault hits the next append (WorkflowFailed for compensate, which skips
// routing; StepFallbackActivated for fail). The wait_record is already deleted and the
// action is not durable (signal_timeout.go:58-60), so recovery must not complete a
// route for w: no StepFallbackActivated is ever emitted for it. The instance parks in
// waiting, so a second signal "go2" to wb makes it running; hydrate then runs on a
// fresh engine (AfterRestart) or after the in-process reset Tick applies on error.

func laDefSignalTimeoutFallback(action string) core.WorkflowDefinition {
	w := signalStepTimeout("w", "go1", "5s", action)
	w.Fallback = "fb"
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.sigto.fb", Version: "1.0.0", Namespace: "la", Name: "sigto",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "la_a"), w, signalStep("wb", "go2"), nativeStep("fb", "la_fb"),
		},
		Transitions: []core.Transition{{From: "a", To: "w"}, {From: "a", To: "wb"}},
		InitialStep: "a", FinalSteps: []string{"wb"}, Metadata: map[string]any{},
	}
}

func laScenarioSignalTimeoutFallback(t *testing.T, action string, faultOn core.EventType, restart bool) {
	t.Helper()
	label := laLabel(fmt.Sprintf("LA-C3 guard signal_timeout action=%s fault=%s", action, faultOn), restart)
	def := laDefSignalTimeoutFallback(action)
	s, fs := laSetup(t, def)
	clk := newManualClock(engineStart)
	ids := laIDs()
	ha := &stepHandler{id: "la_a", outputs: map[string]any{"r": "a"}}
	hfb := &stepHandler{id: "la_fb", outputs: map[string]any{"r": "fb"}}
	e := laEngineManual(fs, clk, ids, ha, hfb)
	ctx := context.Background()
	iid := laSubmit(t, e, def.ID)

	// Park both waits: a completes, then w and wb enter waiting.
	parked := false
	for i := 0; i < laSetupTicks && !parked; i++ {
		if err := e.Tick(ctx); err != nil {
			t.Fatalf("%s: park Tick: %v", label, err)
		}
		inst := laInstance(t, s, iid)
		parked = inst.Status == core.InstanceStatusWaiting && len(inst.CurrentSteps) == 2
	}
	if !parked {
		t.Fatalf("%s: w and wb never parked", label)
	}

	// The timeout is processed inside Tick's SIGNAL_TIMEOUT_SCAN and its error is logged
	// there rather than returned, so the trip is observed on the fault layer.
	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: faultOn})
	clk.advance(6 * time.Second)
	var tickErrs []error
	for i := 0; i < laSetupTicks && fs.trippedCount() == 0; i++ {
		if err := e.Tick(ctx); err != nil {
			tickErrs = append(tickErrs, err)
		}
	}
	if fs.trippedCount() == 0 {
		t.Fatalf("%s: timeout fault never tripped within %d ticks", label, laSetupTicks)
	}

	if restart {
		e = laEngineManual(fs, clk, ids, ha, hfb) // process restart
	} else {
		e.resetInstanceRecovery(iid) // in-process analog of Tick's LA-C1 F1 reset
	}
	if err := e.Signal(ctx, iid, "go2", map[string]any{}); err != nil {
		t.Fatalf("%s: Signal go2: %v", label, err)
	}
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count fb=%d (timeout tick errors=%d)",
		label, n, len(errs), laCalls(hfb), len(tickErrs))
	evs := laEvents(t, s, iid)
	if got := laCount(evs, core.EventTypeStepFallbackActivated); got != 0 {
		t.Errorf("%s: LA-C3 guard (c): StepFallbackActivated count = %d, want 0 (recovery must not route a signal_timeout)", label, got)
	}
	if got := laCalls(hfb); got != 0 {
		t.Errorf("%s: fallback fb executions = %d, want 0", label, got)
	}
	laCheck(t, s, iid, label, core.InstanceStatusCompleted,
		laExpect{"w", "StepFailed(code=signal_timeout,retrying=false)"})
}

func TestLA_C3_Guard_SignalTimeoutCompensateFallback_NoRouting(t *testing.T) {
	laScenarioSignalTimeoutFallback(t, "compensate", core.EventTypeWorkflowFailed, false)
}

func TestLA_C3_Guard_SignalTimeoutCompensateFallback_NoRouting_AfterRestart(t *testing.T) {
	laScenarioSignalTimeoutFallback(t, "compensate", core.EventTypeWorkflowFailed, true)
}

func TestLA_C3_Guard_SignalTimeoutFailFallback_NoRouting(t *testing.T) {
	laScenarioSignalTimeoutFallback(t, "fail", core.EventTypeStepFallbackActivated, false)
}

func TestLA_C3_Guard_SignalTimeoutFailFallback_NoRouting_AfterRestart(t *testing.T) {
	laScenarioSignalTimeoutFallback(t, "fail", core.EventTypeStepFallbackActivated, true)
}

// ── Static classification, CE ruling LA-C3 round 2 (residual R-1, pre-existing) ─
// Step a (2 attempts, immediate retry) has ONE on_error transition a→h whose condition
// steps.a.status == "running" is TRUE when the attempt-2 failure is settled (a is still
// in current_steps, so buildEnv marks it running) and FALSE at recovery (StepFailed
// removed a from current_steps and it is not in Variables, so steps.a.status is null).
// Route completion must not treat a as route (3): no WorkflowFailed for step a may be
// emitted by LA-C3. The outcome must be the pre-fix one: failed via SEC-08 (the stall
// that predates LA-C3), and h never runs. Pre-fix and post-fix results are identical.

func laDefOnErrorStatusDrift() core.WorkflowDefinition {
	a := nativeStep("a", "la_a")
	a.Retry = &core.RetryPolicy{Attempts: 2, Backoff: "immediate"}
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.onerror.drift", Version: "1.0.0", Namespace: "la", Name: "drift",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:    []core.Step{a, nativeStep("h", "la_h")},
		Transitions: []core.Transition{
			{From: "a", To: "h", OnError: true, Condition: `steps.a.status == 'running'`},
		},
		InitialStep: "a", FinalSteps: []string{"h"}, Metadata: map[string]any{},
	}
}

func laScenarioOnErrorStatusDrift(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("LA-C3 guard on_error static classification", restart)
	def := laDefOnErrorStatusDrift()
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	ha := &stepHandler{id: "la_a", err: errLAHandler}
	hh := &stepHandler{id: "la_h", outputs: map[string]any{"r": "h"}}
	e := laEngine(fs, clk, ids, 1, ha, hh)
	iid := laSubmit(t, e, def.ID)

	// Attempt 2 of a fails and routes (2) live; h's StepStarted is the transient fault.
	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepStarted, stepID: "h"})
	laWantInjected(t, laTickUntilTripped(t, e, fs))
	if restart {
		e = laEngine(fs, clk, ids, 1, ha, hh)
	}
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count a=%d h=%d",
		label, n, len(errs), laCalls(ha), laCalls(hh))
	evs := laEvents(t, s, iid)
	for _, p := range laWorkflowFailedPayloads(t, evs) {
		if p.StepID == "a" {
			t.Errorf("%s: LA-C3 completed route (3) for a from a condition-dependent route (R-1): WorkflowFailed{step a, code %s}",
				label, p.Error.Code)
		}
	}
	// Pre-fix outcome: no route completion; SEC-08 fails the instance; h never runs.
	laCheck(t, s, iid, label, core.InstanceStatusFailed,
		laExpect{"a", "StepFailed(code=handler_error,retrying=false)"})
	if got := laCalls(hh); got != 0 {
		t.Errorf("%s: pre-fix outcome requires h never run (R-1 pending drift, pre-existing): h executions = %d", label, got)
	}
}

func TestLA_C3_Guard_OnErrorStaticClassification_NoRoute3Completion(t *testing.T) {
	laScenarioOnErrorStatusDrift(t, false)
}

func TestLA_C3_Guard_OnErrorStaticClassification_NoRoute3Completion_AfterRestart(t *testing.T) {
	laScenarioOnErrorStatusDrift(t, true)
}
