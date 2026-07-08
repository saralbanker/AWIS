package engine

// Event-sequence fixtures (E2E through the real Engine + real SQLite storage).
// Each fixture asserts the EXACT ordered (event_type, step_id) stream and then
// proves EDR-007 forward ≡ RebuildState equivalence.
//
// Determinism note (fan-out): within one tick every StepStarted is emitted in
// the serial CLAIM loop (definition order) and every StepCompleted in the
// serial SETTLE loop (dispatch order = claim order). Only handler execution is
// concurrent, so the emitted event stream is byte-order deterministic even at
// MaxParallelSteps ≥ 2; the diamond fixture therefore asserts an exact order.

import (
	"context"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// buildEngine registers def, wires a NativeRunner with handlers, and constructs
// an Engine driven by a 10ms stepping fake clock. It returns the engine, the
// concrete storage (for RebuildState), and the clock.
func buildEngine(t *testing.T, def core.WorkflowDefinition, maxParallel int, handlers ...*stepHandler) (*Engine, *storage.SQLiteStorage, *fakeClock) {
	t.Helper()
	s := openStorage(t)
	if err := s.RegisterWorkflow(context.Background(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	nr := native.New()
	for _, h := range handlers {
		nr.Register(h)
	}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: maxParallel, Clock: clk.now}, discardLogger())
	return e, s, clk
}

func nativeStep(id, handler string) core.Step {
	return core.Step{ID: id, Name: id, Type: core.StepTypeNative, Handler: core.HandlerRef(handler)}
}

// assertStepDurations verifies each StepCompleted.duration_ms equals the
// fake-clock delta between that step's StepStarted and StepCompleted (10ms).
func assertStepDuration(t *testing.T, evs []core.ExecutionEvent, step string) {
	t.Helper()
	ss := findEvent(t, evs, core.EventTypeStepStarted, step)
	sc := findEvent(t, evs, core.EventTypeStepCompleted, step)
	var p struct {
		DurationMs int64 `json:"duration_ms"`
	}
	decodePayload(t, sc, &p)
	want := sc.EmittedAt.Sub(ss.EmittedAt).Milliseconds()
	if p.DurationMs != want {
		t.Fatalf("step %s duration_ms = %d, want %d (clock-consistent)", step, p.DurationMs, want)
	}
	if want != 10 {
		t.Fatalf("step %s clock delta = %dms, want 10ms (stepping clock)", step, want)
	}
}

// ── Fixture A: linear a → b → c (c final) ──────────────────────────────────────

func TestFixtureA_LinearHappyPath(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.linear", Version: "1.0.0", Namespace: "t", Name: "linear",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "ha"), nativeStep("b", "hb"), nativeStep("c", "hc"),
		},
		Transitions: []core.Transition{{From: "a", To: "b"}, {From: "b", To: "c"}},
		InitialStep: "a", FinalSteps: []string{"c"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", outputs: map[string]any{"ra": "a-done"}}
	hb := &stepHandler{id: "hb", outputs: map[string]any{"rb": "b-done"}}
	hc := &stepHandler{id: "hc", outputs: map[string]any{"rc": "c-done"}}

	e, s, _ := buildEngine(t, def, 1, ha, hb, hc)
	iid, err := e.Submit(context.Background(), "t.linear", "1.0.0", map[string]any{"in": "v"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepCompleted, "a"},
		{core.EventTypeStepStarted, "b"}, {core.EventTypeStepCompleted, "b"},
		{core.EventTypeStepStarted, "c"}, {core.EventTypeStepCompleted, "c"},
		{core.EventTypeWorkflowCompleted, ""},
	}
	assertPairs(t, pairs, want)

	for _, step := range []string{"a", "b", "c"} {
		assertStepDuration(t, evs, step)
	}

	// WorkflowCompleted payload: outputs = {"c": c's outputs}; duration_ms is
	// emitted_at − StartedAt (EDR-011 §2).
	ws := findEvent(t, evs, core.EventTypeWorkflowStarted, "")
	wc := findEvent(t, evs, core.EventTypeWorkflowCompleted, "")
	var wcp struct {
		Outputs    map[string]any `json:"outputs"`
		DurationMs int64          `json:"duration_ms"`
	}
	decodePayload(t, wc, &wcp)
	assertJSONEqual(t, wcp.Outputs, map[string]any{"c": map[string]any{"rc": "c-done"}})
	if wantDur := wc.EmittedAt.Sub(ws.EmittedAt).Milliseconds(); wcp.DurationMs != wantDur {
		t.Fatalf("WorkflowCompleted duration_ms = %d, want %d (clock-consistent)", wcp.DurationMs, wantDur)
	}

	assertProjectionEquivalence(t, s, iid)
}

// ── Fixture B: fan-out + join diamond ─────────────────────────────────────────

func TestFixtureB_FanOutJoinDiamond(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.diamond", Version: "1.0.0", Namespace: "t", Name: "diamond",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "ha"), nativeStep("b", "hb"),
			nativeStep("c", "hc"), nativeStep("d", "hd"),
		},
		Transitions: []core.Transition{
			{From: "a", To: "b"}, {From: "a", To: "c"},
			{From: "b", To: "d"}, {From: "c", To: "d"},
		},
		InitialStep: "a", FinalSteps: []string{"d"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", outputs: map[string]any{"ra": "a"}}
	hb := &stepHandler{id: "hb", outputs: map[string]any{"rb": "b"}}
	hc := &stepHandler{id: "hc", outputs: map[string]any{"rc": "c"}}
	hd := &stepHandler{id: "hd", outputs: map[string]any{"rd": "d"}}

	e, s, _ := buildEngine(t, def, 2, ha, hb, hc, hd)
	iid, err := e.Submit(context.Background(), "t.diamond", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepCompleted, "a"},
		{core.EventTypeStepStarted, "b"}, {core.EventTypeStepStarted, "c"},
		{core.EventTypeStepCompleted, "b"}, {core.EventTypeStepCompleted, "c"},
		{core.EventTypeStepStarted, "d"}, {core.EventTypeStepCompleted, "d"},
		{core.EventTypeWorkflowCompleted, ""},
	}
	assertPairs(t, pairs, want)

	// Order constraints (independent of the exact-sequence assertion above):
	// b,c start after a completes and before d starts; d starts after both
	// b and c complete.
	idx := indexer(pairs)
	if idx(core.EventTypeStepCompleted, "a") >= idx(core.EventTypeStepStarted, "b") ||
		idx(core.EventTypeStepCompleted, "a") >= idx(core.EventTypeStepStarted, "c") {
		t.Fatalf("b/c must start after a completes")
	}
	if idx(core.EventTypeStepStarted, "b") >= idx(core.EventTypeStepStarted, "d") ||
		idx(core.EventTypeStepStarted, "c") >= idx(core.EventTypeStepStarted, "d") {
		t.Fatalf("d must start after b and c start")
	}
	if idx(core.EventTypeStepCompleted, "b") >= idx(core.EventTypeStepStarted, "d") ||
		idx(core.EventTypeStepCompleted, "c") >= idx(core.EventTypeStepStarted, "d") {
		t.Fatalf("d must start only after BOTH b and c complete (join gate)")
	}

	wc := findEvent(t, evs, core.EventTypeWorkflowCompleted, "")
	var wcp struct {
		Outputs map[string]any `json:"outputs"`
	}
	decodePayload(t, wc, &wcp)
	assertJSONEqual(t, wcp.Outputs, map[string]any{"d": map[string]any{"rd": "d"}})

	assertProjectionEquivalence(t, s, iid)
}

// ── Fixture C: degenerate failure (no retry / no fallback) ────────────────────

func TestFixtureC_DegenerateFailure(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.fail", Version: "1.0.0", Namespace: "t", Name: "fail",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("a", "ha")},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", err: errBoom}

	e, s, _ := buildEngine(t, def, 1, ha)
	iid, err := e.Submit(context.Background(), "t.fail", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusFailed {
		t.Fatalf("status = %q, want failed", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"},
		{core.EventTypeStepFailed, "a"},
		{core.EventTypeWorkflowFailed, ""},
	}
	assertPairs(t, pairs, want)

	// StepFailed payload: retrying=false (degenerate no-retry path).
	sf := findEvent(t, evs, core.EventTypeStepFailed, "a")
	var sfp struct {
		Retrying bool           `json:"retrying"`
		Error    core.StepError `json:"error"`
	}
	decodePayload(t, sf, &sfp)
	if sfp.Retrying {
		t.Fatalf("StepFailed.retrying = true, want false")
	}
	if sfp.Error.Code != "handler_error" {
		t.Fatalf("StepFailed.error.code = %q, want handler_error", sfp.Error.Code)
	}

	assertProjectionEquivalence(t, s, iid)
}
