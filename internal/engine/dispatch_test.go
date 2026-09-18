package engine

// dispatchOne tests: unregistered step type ⇒ runner_unavailable; idempotency
// cache hit skips a second execution (Blueprint §5 L2 / §20).

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
)

// countingRunner is an engine.Runner that counts executions and returns a fixed
// result, so a cache hit is observable as a non-incrementing counter.
type countingRunner struct {
	calls int
	out   core.StepResult
}

func (r *countingRunner) Run(_ context.Context, _ core.StepContext, _ core.Step) (core.StepResult, *core.StepError) {
	r.calls++
	return r.out, nil
}

func TestDispatchOne_UnregisteredStepType(t *testing.T) {
	s := openStorage(t)
	// Empty runner map ⇒ no runner for the step's type.
	e := New(s, map[core.StepType]Runner{}, Config{Clock: func() time.Time { return storageBase }}, discardLogger())

	step := core.Step{ID: "sig", Type: core.StepTypeSignal, Handler: "x"}
	res := e.dispatchOne(context.Background(), "inst-1", step, core.StepContext{Attempt: 1})
	if res.stepErr == nil {
		t.Fatalf("expected a StepError for an unregistered step type")
	}
	if res.stepErr.Code != "runner_unavailable" {
		t.Fatalf("code = %q, want runner_unavailable", res.stepErr.Code)
	}
}

func TestDispatchOne_IdempotencyCacheHit(t *testing.T) {
	s := openStorage(t)
	runner := &countingRunner{out: core.StepResult{Outputs: map[string]any{"r": "one"}}}
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: runner},
		Config{Clock: func() time.Time { return storageBase }}, discardLogger())

	ctx := context.Background()
	step := core.Step{ID: "step-a", Type: core.StepTypeNative, Handler: "h"}
	sc := core.StepContext{InstanceID: "inst-1", StepID: "step-a", Attempt: 1}

	// First dispatch: executes and caches.
	r1 := e.dispatchOne(ctx, "inst-1", step, sc)
	if r1.stepErr != nil {
		t.Fatalf("first dispatch errored: %+v", r1.stepErr)
	}
	if runner.calls != 1 {
		t.Fatalf("after first dispatch calls = %d, want 1", runner.calls)
	}

	// Second dispatch with the same (instance, step, attempt) key: cache hit ⇒
	// no re-execution; identical result.
	r2 := e.dispatchOne(ctx, "inst-1", step, sc)
	if r2.stepErr != nil {
		t.Fatalf("second dispatch errored: %+v", r2.stepErr)
	}
	if runner.calls != 1 {
		t.Fatalf("after cached second dispatch calls = %d, want 1 (no re-exec)", runner.calls)
	}
	if !reflect.DeepEqual(r2.out.Outputs, map[string]any{"r": "one"}) {
		t.Fatalf("cached outputs = %#v, want {r:one}", r2.out.Outputs)
	}
}

// ── runInvoke panic backstop (engine-hardening Step 2c) ─────────────────────
//
// internal/runner/native already recovers a panicking user HANDLER into
// StepError{code:"handler_panic"} (native.go, Blueprint B-3) — that recovery
// lives inside the native Runner's own Run method, one level below here. The
// tests in this section cover the SEPARATE backstop in tick.go's runInvoke:
// the Runner (or UsageRunner) ITSELF panicking — e.g. a buggy intelligence /
// subprocess / plugin / custom Runner implementation, or any future runner
// kind — must not crash dispatchOne's goroutine (and, transitively, the whole
// process). runInvoke recovers such a panic into StepError{code:
// "runner_panic"}, a DISTINCT code from "handler_panic" so a caller can tell
// "the runner itself broke" from "the dispatched handler broke".

// panicRunner is an engine.Runner whose Run always panics with panicVal.
type panicRunner struct {
	panicVal any
}

func (r *panicRunner) Run(_ context.Context, _ core.StepContext, _ core.Step) (core.StepResult, *core.StepError) {
	panic(r.panicVal)
}

// panicUsageRunner is an engine.Runner + UsageRunner whose RunWithUsage always
// panics; its Run must never be invoked (runInvoke prefers RunWithUsage when
// the concrete Runner also implements UsageRunner).
type panicUsageRunner struct {
	panicVal any
}

func (r *panicUsageRunner) Run(_ context.Context, _ core.StepContext, _ core.Step) (core.StepResult, *core.StepError) {
	return core.StepResult{}, &core.StepError{Code: "should_not_be_called", Message: "Run must not run when RunWithUsage is available"}
}

func (r *panicUsageRunner) RunWithUsage(_ context.Context, _ core.StepContext, _ core.Step) (core.StepResult, *core.Usage, *core.StepError) {
	panic(r.panicVal)
}

// recursePanic grows the captured stack trace by n stack frames before
// panicking with val, so debug.Stack() output can be forced past
// maxRunnerPanicStackBytes (used by the truncation test below).
func recursePanic(n int, val any) {
	if n <= 0 {
		panic(val)
	}
	recursePanic(n-1, val)
}

// deepPanicRunner panics after recursing depth frames deep.
type deepPanicRunner struct {
	depth    int
	panicVal any
}

func (r *deepPanicRunner) Run(_ context.Context, _ core.StepContext, _ core.Step) (core.StepResult, *core.StepError) {
	recursePanic(r.depth, r.panicVal)
	return core.StepResult{}, nil // unreachable
}

// TestRunInvoke_RunnerPanicRecovered: a plain Runner whose Run panics is
// recovered by runInvoke into StepError{code:"runner_panic"}; the panic value
// (error, string, or other) is folded into the message, and the process
// (this test function) survives to return normally.
func TestRunInvoke_RunnerPanicRecovered(t *testing.T) {
	for _, tc := range []struct {
		name     string
		panicVal any
	}{
		{"error value", errors.New("kaboom")},
		{"string value", "boom"},
		{"int value", 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := openStorage(t)
			runner := &panicRunner{panicVal: tc.panicVal}
			e := New(s, map[core.StepType]Runner{core.StepTypeNative: runner},
				Config{Clock: func() time.Time { return storageBase }}, discardLogger())

			step := core.Step{ID: "s-panic", Type: core.StepTypeNative, Handler: "h"}
			sc := core.StepContext{InstanceID: "inst-1", StepID: "s-panic", Attempt: 1}

			out, usage, serr := e.runInvoke(context.Background(), runner, sc, step)
			if serr == nil {
				t.Fatalf("expected a StepError from a panicking Runner, got nil (out=%+v usage=%+v)", out, usage)
			}
			if serr.Code != "runner_panic" {
				t.Fatalf("code = %q, want runner_panic", serr.Code)
			}
			if !strings.Contains(serr.Message, step.ID) {
				t.Fatalf("message %q must name the step %q", serr.Message, step.ID)
			}
			wantSubstr := fmt.Sprintf("%v", tc.panicVal)
			if !strings.Contains(serr.Message, wantSubstr) {
				t.Fatalf("message %q must include the recovered panic value %q", serr.Message, wantSubstr)
			}
			if usage != nil {
				t.Fatalf("usage must be nil on a panic, got %+v", usage)
			}
			if serr.Details == nil {
				t.Fatalf("runner_panic must carry Details[\"stack\"]")
			}
			stack, ok := serr.Details["stack"].(string)
			if !ok || stack == "" {
				t.Fatalf("Details[\"stack\"] must be a non-empty string; got %#v", serr.Details["stack"])
			}
		})
	}
}

// TestRunInvoke_UsageRunnerPanicRecovered: a UsageRunner whose RunWithUsage
// panics is recovered identically to a plain Runner panic (same code, same
// shape) — RunWithUsage's own Run method (which would report a different,
// wrong code) must never be reached.
func TestRunInvoke_UsageRunnerPanicRecovered(t *testing.T) {
	s := openStorage(t)
	runner := &panicUsageRunner{panicVal: errors.New("usage-boom")}
	e := New(s, map[core.StepType]Runner{core.StepTypeIntelligence: runner},
		Config{Clock: func() time.Time { return storageBase }}, discardLogger())

	step := core.Step{ID: "s-usage-panic", Type: core.StepTypeIntelligence, Handler: "h"}
	sc := core.StepContext{InstanceID: "inst-1", StepID: "s-usage-panic", Attempt: 1}

	out, usage, serr := e.runInvoke(context.Background(), runner, sc, step)
	if serr == nil {
		t.Fatalf("expected a StepError from a panicking UsageRunner, got nil (out=%+v usage=%+v)", out, usage)
	}
	if serr.Code != "runner_panic" {
		t.Fatalf("code = %q, want runner_panic (RunWithUsage panic must NOT surface should_not_be_called)", serr.Code)
	}
	if !strings.Contains(serr.Message, "usage-boom") {
		t.Fatalf("message %q must include the recovered panic value", serr.Message)
	}
	if usage != nil {
		t.Fatalf("usage must be nil on a panic, got %+v", usage)
	}
}

// TestRunInvoke_PanicStackBoundedAndTruncated: a deep-recursion panic forces
// debug.Stack() past maxRunnerPanicStackBytes; runInvoke must truncate to
// EXACTLY maxRunnerPanicStackBytes plus the truncation marker, never persist
// an unbounded stack (the payload lands in the append-only EventLog and can
// never be rewritten).
func TestRunInvoke_PanicStackBoundedAndTruncated(t *testing.T) {
	s := openStorage(t)
	runner := &deepPanicRunner{depth: 400, panicVal: "deep-boom"}
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: runner},
		Config{Clock: func() time.Time { return storageBase }}, discardLogger())

	step := core.Step{ID: "s-deep", Type: core.StepTypeNative, Handler: "h"}
	sc := core.StepContext{InstanceID: "inst-1", StepID: "s-deep", Attempt: 1}

	_, _, serr := e.runInvoke(context.Background(), runner, sc, step)
	if serr == nil || serr.Code != "runner_panic" {
		t.Fatalf("want runner_panic, got %+v", serr)
	}
	stack, ok := serr.Details["stack"].(string)
	if !ok || stack == "" {
		t.Fatalf("Details[\"stack\"] must be a non-empty string; got %#v", serr.Details["stack"])
	}
	const marker = "\n... stack truncated ..."
	wantLen := maxRunnerPanicStackBytes + len(marker)
	if len(stack) != wantLen {
		t.Fatalf("truncated stack length = %d, want exactly %d (maxRunnerPanicStackBytes=%d + marker=%d) — a 400-deep recursion must exceed the bound",
			len(stack), wantLen, maxRunnerPanicStackBytes, len(marker))
	}
	if !strings.HasSuffix(stack, marker) {
		t.Fatalf("truncated stack must end with the truncation marker %q; got suffix %q", marker, stack[len(stack)-len(marker):])
	}
}

// TestRunInvoke_ShallowPanicStackNotTruncated: a shallow panic's captured
// stack is well under the bound and must be recorded AS-IS (no marker
// appended, no truncation).
func TestRunInvoke_ShallowPanicStackNotTruncated(t *testing.T) {
	s := openStorage(t)
	runner := &panicRunner{panicVal: "shallow-boom"}
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: runner},
		Config{Clock: func() time.Time { return storageBase }}, discardLogger())

	step := core.Step{ID: "s-shallow", Type: core.StepTypeNative, Handler: "h"}
	sc := core.StepContext{InstanceID: "inst-1", StepID: "s-shallow", Attempt: 1}

	_, _, serr := e.runInvoke(context.Background(), runner, sc, step)
	if serr == nil || serr.Code != "runner_panic" {
		t.Fatalf("want runner_panic, got %+v", serr)
	}
	stack := serr.Details["stack"].(string)
	if len(stack) >= maxRunnerPanicStackBytes {
		t.Fatalf("a shallow panic's stack (%d bytes) should not reach the %d-byte bound", len(stack), maxRunnerPanicStackBytes)
	}
	if strings.Contains(stack, "truncated") {
		t.Fatalf("an untruncated stack must not contain the truncation marker; got %q", stack)
	}
}

// TestEngine_RunnerPanicSurvivesTickAndTerminatesWorkflow is the E2E backstop
// proof: a workflow whose single step's Runner panics on every dispatch must
// (a) never crash Tick (the process survives — this test function returning
// normally through Tick calls IS the proof), (b) reach a TERMINAL status
// (`failed`, not hang) via the ordinary settleFailure/routeTerminalFailure
// path (a runner_panic StepError is just another *core.StepError to settle),
// and (c) leave the engine usable for LATER ticks (a second, independent
// instance on a normal handler is submitted and driven to completion in the
// ticks that follow, proving the panic didn't wedge the tick loop or the
// engine's internal state for other instances).
func TestEngine_RunnerPanicSurvivesTickAndTerminatesWorkflow(t *testing.T) {
	panicDef := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.panic", Version: "1.0.0", Namespace: "t", Name: "panic",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"}},
		InitialStep: "a", FinalSteps: []string{}, Metadata: map[string]any{},
	}
	okDef := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.ok", Version: "1.0.0", Namespace: "t", Name: "ok",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("z", "hz")},
		InitialStep: "z", FinalSteps: []string{"z"}, Metadata: map[string]any{},
	}

	s := openStorage(t)
	ctx := context.Background()
	if err := s.RegisterWorkflow(ctx, panicDef); err != nil {
		t.Fatalf("RegisterWorkflow panicDef: %v", err)
	}
	if err := s.RegisterWorkflow(ctx, okDef); err != nil {
		t.Fatalf("RegisterWorkflow okDef: %v", err)
	}

	runner := &panicRunner{panicVal: errors.New("runner broke")}
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: runner},
		Config{MaxParallelSteps: 1, Clock: func() time.Time { return storageBase }}, discardLogger())

	panicIID, err := e.Submit(ctx, "t.panic", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit panic instance: %v", err)
	}

	// Tick must return no error: the panic never escapes runInvoke/dispatchOne.
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick with a panicking Runner returned an error (process-crash surrogate): %v", err)
	}

	panicInst := runToTerminal(t, e, s, panicIID)
	if panicInst.Status != core.InstanceStatusFailed {
		t.Fatalf("status = %q, want failed (runner panic must terminate, not hang)", panicInst.Status)
	}

	pairs, evs := eventPairs(t, s, panicIID)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("panicking step must not be re-dispatched: got %d StepStarted, want 1", n)
	}
	sf := findEvent(t, evs, core.EventTypeStepFailed, "a")
	var sfp stepFailedPayload
	decodePayload(t, sf, &sfp)
	if sfp.Error.Code != "runner_panic" {
		t.Fatalf("StepFailed.error.code = %q, want runner_panic", sfp.Error.Code)
	}
	if !strings.Contains(sfp.Error.Message, "a") {
		t.Fatalf("StepFailed.error.message %q must name step 'a'", sfp.Error.Message)
	}
	if sfp.Error.Details == nil || sfp.Error.Details["stack"] == "" {
		t.Fatalf("StepFailed.error.details.stack must survive the EventLog round trip; got %+v", sfp.Error.Details)
	}
	if n := countPairs(pairs, core.EventTypeWorkflowFailed, ""); n != 1 {
		t.Fatalf("WorkflowFailed count = %d, want 1", n)
	}

	// "Later ticks still work": a second, independent instance on a normal
	// handler is submitted AFTER the panic and driven to completion — proving
	// the engine's tick loop and internal bookkeeping were not wedged by it.
	// This engine's core.StepTypeNative slot holds the raw panicRunner (not a
	// *native.NativeRunner), so the "ok" workflow is driven by a second engine
	// over the SAME storage; Runner registration is per-engine, but the
	// process-survival property under test (the panic never escapes to crash
	// the process/goroutine) is process-wide, so proving a second engine keeps
	// functioning is the correct-shaped proof.
	hz := &stepHandler{id: "hz", outputs: map[string]any{"r": "ok"}}
	e2, s2 := newNativeEngine(t, okDef, 1, func() time.Time { return storageBase }, hz)
	okIID, err := e2.Submit(ctx, "t.ok", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit ok instance: %v", err)
	}
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("later Tick on the panic-instance engine errored: %v", err)
	}
	okInst := runToTerminal(t, e2, s2, okIID)
	if okInst.Status != core.InstanceStatusCompleted {
		t.Fatalf("ok instance status = %q, want completed (later ticks must still work)", okInst.Status)
	}

	assertProjectionEquivalence(t, s, panicIID)
}

// TestEngine_RunnerPanicDistinctFromHandlerPanic: a native step whose
// user-supplied HANDLER panics produces "handler_panic" (recovered inside
// internal/runner/native, one layer below the engine); a step whose RUNNER
// ITSELF panics produces "runner_panic" (recovered by runInvoke, this card).
// The two codes must differ so a caller can distinguish "the handler broke"
// from "the runner broke".
func TestEngine_RunnerPanicDistinctFromHandlerPanic(t *testing.T) {
	handlerPanicDef := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.hpanic", Version: "1.0.0", Namespace: "t", Name: "hpanic",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"}},
		InitialStep: "a", FinalSteps: []string{}, Metadata: map[string]any{},
	}
	runnerPanicDef := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.rpanic", Version: "1.0.0", Namespace: "t", Name: "rpanic",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{{ID: "b", Name: "b", Type: core.StepTypeSubprocess, Handler: "hb"}},
		InitialStep: "b", FinalSteps: []string{}, Metadata: map[string]any{},
	}

	s := openStorage(t)
	ctx := context.Background()
	if err := s.RegisterWorkflow(ctx, handlerPanicDef); err != nil {
		t.Fatalf("RegisterWorkflow handlerPanicDef: %v", err)
	}
	if err := s.RegisterWorkflow(ctx, runnerPanicDef); err != nil {
		t.Fatalf("RegisterWorkflow runnerPanicDef: %v", err)
	}

	nr := native.New()
	nr.Register(&panicHandler{id: "ha", panicVal: "handler broke"})
	rp := &panicRunner{panicVal: "runner broke"}
	e := New(s, map[core.StepType]Runner{
		core.StepTypeNative:     nr,
		core.StepTypeSubprocess: rp,
	}, Config{MaxParallelSteps: 1, Clock: func() time.Time { return storageBase }}, discardLogger())

	hIID, err := e.Submit(ctx, "t.hpanic", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit handlerPanicDef: %v", err)
	}
	rIID, err := e.Submit(ctx, "t.rpanic", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit runnerPanicDef: %v", err)
	}

	hInst := runToTerminal(t, e, s, hIID)
	if hInst.Status != core.InstanceStatusFailed {
		t.Fatalf("handler-panic instance status = %q, want failed", hInst.Status)
	}
	rInst := runToTerminal(t, e, s, rIID)
	if rInst.Status != core.InstanceStatusFailed {
		t.Fatalf("runner-panic instance status = %q, want failed", rInst.Status)
	}

	_, hEvs := eventPairs(t, s, hIID)
	var hsfp stepFailedPayload
	decodePayload(t, findEvent(t, hEvs, core.EventTypeStepFailed, "a"), &hsfp)
	if hsfp.Error.Code != "handler_panic" {
		t.Fatalf("handler-panic instance StepFailed.error.code = %q, want handler_panic", hsfp.Error.Code)
	}

	_, rEvs := eventPairs(t, s, rIID)
	var rsfp stepFailedPayload
	decodePayload(t, findEvent(t, rEvs, core.EventTypeStepFailed, "b"), &rsfp)
	if rsfp.Error.Code != "runner_panic" {
		t.Fatalf("runner-panic instance StepFailed.error.code = %q, want runner_panic", rsfp.Error.Code)
	}

	if hsfp.Error.Code == rsfp.Error.Code {
		t.Fatalf("handler_panic and runner_panic must be distinct codes")
	}
}

// panicHandler is a core.StepHandler whose Execute always panics — used to
// prove native's OWN handler-panic recovery (handler_panic) is distinct from
// runInvoke's runner-panic recovery (runner_panic).
type panicHandler struct {
	id       string
	panicVal any
}

func (h *panicHandler) ID() string { return h.id }

func (h *panicHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	panic(h.panicVal)
}
