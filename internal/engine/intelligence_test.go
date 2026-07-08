package engine

// Intelligence-step E2E (Blueprint §17, EDR-011 §8, ADJ-8). Drives the real
// engine with the intelligence Runner registered in the dispatch map:
//   - capability_fallback: an unserveable required=false step routes to its
//     step.Fallback (StepFailed → StepFallbackActivated → fallback runs).
//   - capability_unavailable: an unserveable required=true step with no fallback
//     fails the workflow.
//   - ADJ-8 payload: a SUCCEEDING intelligence step's StepCompleted carries the
//     {adapter, model, tokens_used} triple; a native step's StepCompleted lacks it.

import (
	"context"
	"testing"

	"github.com/awis/awis/internal/core"
	intel "github.com/awis/awis/internal/intelligence"
	"github.com/awis/awis/internal/intelligence/adapters/null"
	runintel "github.com/awis/awis/internal/runner/intelligence"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// stubAdapter is a minimal AVAILABLE IntelligencePort for the success path (the
// null adapter cannot succeed — IsAvailable()=false). It serves "draft" with a
// fixed Usage so the ADJ-8 triple is assertable to exact values.
type stubAdapter struct {
	name  string
	usage core.Usage
	out   map[string]any
}

func (a *stubAdapter) Draft(_ context.Context, _ core.DraftRequest) (core.DraftResponse, error) {
	return core.DraftResponse{Output: a.out, Usage: a.usage}, nil
}
func (a *stubAdapter) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }
func (a *stubAdapter) Synthesize(_ context.Context, _ core.SynthesisRequest) (core.SynthesisResponse, error) {
	return core.SynthesisResponse{}, nil
}
func (a *stubAdapter) Classify(_ context.Context, _ string, _ []string) (core.Classification, error) {
	return core.Classification{}, nil
}
func (a *stubAdapter) IsAvailable() bool { return true }
func (a *stubAdapter) Capabilities() []core.Capability {
	return []core.Capability{{Name: "draft"}}
}
func (a *stubAdapter) ProviderName() string { return a.name }

// newIntelEngine builds an engine with BOTH the native runner (for handlers) and
// the intelligence runner (backed by disp) registered.
func newIntelEngine(t *testing.T, def core.WorkflowDefinition, disp *intel.Dispatcher, handlers ...core.StepHandler) (*Engine, *storage.SQLiteStorage) {
	t.Helper()
	s := openStorage(t)
	if err := s.RegisterWorkflow(context.Background(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	nr := native.New()
	for _, h := range handlers {
		nr.Register(h)
	}
	e := New(s, map[core.StepType]Runner{
		core.StepTypeNative:       nr,
		core.StepTypeIntelligence: runintel.New(disp),
	}, Config{MaxParallelSteps: 1, Clock: newManualClock(engineStart).now}, discardLogger())
	return e, s
}

// nullDispatcher assembles a Dispatcher whose only chain member is the null
// adapter (IsAvailable()=false) — every capability is therefore unserveable.
func nullDispatcher(t *testing.T) *intel.Dispatcher {
	t.Helper()
	router, err := intel.NewRouter(
		[]intel.Registration{{Adapter: null.New(), Locality: intel.LocalityLocal}},
		[]string{"null"},
	)
	if err != nil {
		t.Fatalf("NewRouter(null): %v", err)
	}
	return intel.NewDispatcher(router)
}

func intelStep(id, capability string, required bool, fallback string) core.Step {
	return core.Step{
		ID: id, Name: id, Type: core.StepTypeIntelligence, Fallback: fallback,
		Intelligence: &core.IntelReq{Capability: capability, Required: required, ContextBudget: 1000},
	}
}

// ── capability_fallback → step.Fallback ───────────────────────────────────────

func TestIntelligence_CapabilityFallbackToManual(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.ifb", Version: "1.0.0", Namespace: "t", Name: "ifb",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			intelStep("gen", "draft", false, "manual"),
			nativeStep("manual", "hman"),
		},
		InitialStep: "gen", FinalSteps: []string{"manual"}, Metadata: map[string]any{},
	}
	hman := &stepHandler{id: "hman", outputs: map[string]any{"r": "manual-done"}}
	e, s := newIntelEngine(t, def, nullDispatcher(t), hman)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.ifb", "1.0.0", nil)
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
		{core.EventTypeStepStarted, "gen"},
		{core.EventTypeStepFailed, "gen"},
		{core.EventTypeStepFallbackActivated, "gen"},
		{core.EventTypeStepStarted, "manual"},
		{core.EventTypeStepCompleted, "manual"},
		{core.EventTypeWorkflowCompleted, ""},
	}
	assertPairs(t, pairs, want)

	// StepFailed{retrying:false, error.code=="capability_fallback"} (capability_fallback skips retries).
	sf := findEvent(t, evs, core.EventTypeStepFailed, "gen")
	var sfp stepFailedPayload
	decodePayload(t, sf, &sfp)
	if sfp.Retrying {
		t.Fatalf("StepFailed.retrying = true, want false")
	}
	if sfp.Error.Code != "capability_fallback" {
		t.Fatalf("StepFailed.error.code = %q, want capability_fallback", sfp.Error.Code)
	}

	// StepFallbackActivated{step_id, fallback_step_id=="manual", reason=="capability_fallback"}.
	fa := findEvent(t, evs, core.EventTypeStepFallbackActivated, "gen")
	var fap stepFallbackActivatedPayload
	decodePayload(t, fa, &fap)
	if fap.StepID != "gen" || fap.FallbackStepID != "manual" || fap.Reason != "capability_fallback" {
		t.Fatalf("fallback payload = %+v, want {gen, manual, capability_fallback}", fap)
	}

	assertProjectionEquivalence(t, s, iid)
}

// ── capability_unavailable → WorkflowFailed ───────────────────────────────────

func TestIntelligence_RequiredUnavailableFailsWorkflow(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.ireq", Version: "1.0.0", Namespace: "t", Name: "ireq",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{intelStep("gen", "draft", true, "")},
		InitialStep: "gen", FinalSteps: []string{"gen"}, Metadata: map[string]any{},
	}
	e, s := newIntelEngine(t, def, nullDispatcher(t))
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.ireq", "1.0.0", nil)
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
		{core.EventTypeStepStarted, "gen"},
		{core.EventTypeStepFailed, "gen"},
		{core.EventTypeWorkflowFailed, ""},
	}
	assertPairs(t, pairs, want)

	sf := findEvent(t, evs, core.EventTypeStepFailed, "gen")
	var sfp stepFailedPayload
	decodePayload(t, sf, &sfp)
	if sfp.Retrying || sfp.Error.Code != "capability_unavailable" {
		t.Fatalf("StepFailed = {retrying:%v, code:%q}, want {false, capability_unavailable}", sfp.Retrying, sfp.Error.Code)
	}
	wf := findEvent(t, evs, core.EventTypeWorkflowFailed, "")
	var wfp workflowFailedPayload
	decodePayload(t, wf, &wfp)
	if wfp.Error.Code != "capability_unavailable" {
		t.Fatalf("WorkflowFailed.error.code = %q, want capability_unavailable", wfp.Error.Code)
	}

	assertProjectionEquivalence(t, s, iid)
}

// ── ADJ-8 StepCompleted usage triple (present for intelligence, absent for native) ─

func TestIntelligence_SuccessADJ8Payload(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.iok", Version: "1.0.0", Namespace: "t", Name: "iok",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			intelStep("gen", "draft", false, ""),
			nativeStep("post", "hp"),
		},
		Transitions: []core.Transition{{From: "gen", To: "post"}},
		InitialStep: "gen", FinalSteps: []string{"post"}, Metadata: map[string]any{},
	}
	stub := &stubAdapter{
		name:  "test-adapter",
		usage: core.Usage{Adapter: "test-adapter", Model: "test-model", TokensUsed: 42},
		out:   map[string]any{"draft": "generated"},
	}
	router, err := intel.NewRouter(
		[]intel.Registration{{Adapter: stub, Locality: intel.LocalityCloud}},
		[]string{"test-adapter"},
	)
	if err != nil {
		t.Fatalf("NewRouter(stub): %v", err)
	}
	hp := &stepHandler{id: "hp", outputs: map[string]any{"r": "posted"}}
	e, s := newIntelEngine(t, def, intel.NewDispatcher(router), hp)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.iok", "1.0.0", nil)
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
		{core.EventTypeStepStarted, "gen"}, {core.EventTypeStepCompleted, "gen"},
		{core.EventTypeStepStarted, "post"}, {core.EventTypeStepCompleted, "post"},
		{core.EventTypeWorkflowCompleted, ""},
	}
	assertPairs(t, pairs, want)

	// Intelligence StepCompleted carries the ADJ-8 triple with EXACT values.
	genDone := findEvent(t, evs, core.EventTypeStepCompleted, "gen")
	assertHasKeys(t, genDone, "adapter", "model", "tokens_used")
	var gp stepCompletedUsagePayload
	decodePayload(t, genDone, &gp)
	if gp.Adapter != "test-adapter" || gp.Model != "test-model" || gp.TokensUsed != 42 {
		t.Fatalf("ADJ-8 usage = {%q,%q,%d}, want {test-adapter,test-model,42}", gp.Adapter, gp.Model, gp.TokensUsed)
	}

	// Native StepCompleted omits the usage triple entirely.
	postDone := findEvent(t, evs, core.EventTypeStepCompleted, "post")
	assertLacksKeys(t, postDone, "adapter", "model", "tokens_used")

	assertProjectionEquivalence(t, s, iid)
}
