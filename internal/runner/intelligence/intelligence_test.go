package intelligence

// Unit coverage for the intelligence Runner's error mapping (mapDispatchError)
// and capability gate (RunWithUsage), exercised through a REAL Dispatcher
// assembled with the M04 null adapter (IsAvailable()=false ⇒ chain yields the
// no-eligible typed errors) and a tiny available adapter for the success path.

import (
	"context"
	"testing"

	"github.com/awis/awis/internal/core"
	intel "github.com/awis/awis/internal/intelligence"
	"github.com/awis/awis/internal/intelligence/adapters/null"
)

// availAdapter is an AVAILABLE IntelligencePort serving "draft" with fixed usage.
// draftCalls proves the capability gate does (or does not) reach a dispatch.
type availAdapter struct {
	draftCalls int
}

func (a *availAdapter) Draft(_ context.Context, _ core.DraftRequest) (core.DraftResponse, error) {
	a.draftCalls++
	return core.DraftResponse{
		Output: map[string]any{"draft": "ok"},
		Usage:  core.Usage{Adapter: "avail", Model: "avail-model", TokensUsed: 7},
	}, nil
}
func (a *availAdapter) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }
func (a *availAdapter) Synthesize(_ context.Context, _ core.SynthesisRequest) (core.SynthesisResponse, error) {
	return core.SynthesisResponse{}, nil
}
func (a *availAdapter) Classify(_ context.Context, _ string, _ []string) (core.Classification, error) {
	return core.Classification{}, nil
}
func (a *availAdapter) IsAvailable() bool               { return true }
func (a *availAdapter) Capabilities() []core.Capability { return []core.Capability{{Name: "draft"}} }
func (a *availAdapter) ProviderName() string            { return "avail" }

func nullRunner(t *testing.T) *Runner {
	t.Helper()
	router, err := intel.NewRouter(
		[]intel.Registration{{Adapter: null.New(), Locality: intel.LocalityLocal}},
		[]string{"null"},
	)
	if err != nil {
		t.Fatalf("NewRouter(null): %v", err)
	}
	return New(intel.NewDispatcher(router))
}

func intelStep(capability string, required bool) core.Step {
	return core.Step{
		ID: "s", Type: core.StepTypeIntelligence,
		Intelligence: &core.IntelReq{Capability: capability, Required: required, ContextBudget: 1000},
	}
}

func emptyCtx() core.StepContext { return core.StepContext{Inputs: map[string]any{}} }

// draft required=false against the unavailable null chain ⇒ capability_fallback.
func TestRunWithUsage_FallbackWhenNotRequired(t *testing.T) {
	r := nullRunner(t)
	out, usage, serr := r.RunWithUsage(context.Background(), emptyCtx(), intelStep("draft", false))
	if serr == nil || serr.Code != "capability_fallback" {
		t.Fatalf("serr = %+v, want code capability_fallback", serr)
	}
	if usage != nil {
		t.Fatalf("usage = %+v, want nil on error path", usage)
	}
	if len(out.Outputs) != 0 {
		t.Fatalf("out.Outputs = %v, want empty", out.Outputs)
	}
}

// draft required=true against the unavailable null chain ⇒ capability_unavailable.
func TestRunWithUsage_UnavailableWhenRequired(t *testing.T) {
	r := nullRunner(t)
	_, _, serr := r.RunWithUsage(context.Background(), emptyCtx(), intelStep("draft", true))
	if serr == nil || serr.Code != "capability_unavailable" {
		t.Fatalf("serr = %+v, want code capability_unavailable", serr)
	}
}

// classify/embed/bogus ⇒ capability_unknown with NO dispatch (FR-IL-10, CONTRA-3).
func TestRunWithUsage_CapabilityUnknownNoDispatch(t *testing.T) {
	a := &availAdapter{}
	router, err := intel.NewRouter(
		[]intel.Registration{{Adapter: a, Locality: intel.LocalityCloud}},
		[]string{"avail"},
	)
	if err != nil {
		t.Fatalf("NewRouter(avail): %v", err)
	}
	r := New(intel.NewDispatcher(router))
	for _, capability := range []string{"classify", "embed", "bogus"} {
		_, _, serr := r.RunWithUsage(context.Background(), emptyCtx(), intelStep(capability, false))
		if serr == nil || serr.Code != "capability_unknown" {
			t.Fatalf("capability %q: serr = %+v, want code capability_unknown", capability, serr)
		}
	}
	if a.draftCalls != 0 {
		t.Fatalf("capability_unknown must NOT dispatch; draftCalls = %d, want 0", a.draftCalls)
	}
}

// nil step.Intelligence ⇒ intelligence_error.
func TestRunWithUsage_NilIntelligenceConfig(t *testing.T) {
	r := nullRunner(t)
	step := core.Step{ID: "s", Type: core.StepTypeIntelligence} // Intelligence nil
	_, _, serr := r.RunWithUsage(context.Background(), emptyCtx(), step)
	if serr == nil || serr.Code != "intelligence_error" {
		t.Fatalf("serr = %+v, want code intelligence_error", serr)
	}
}

// A successful draft returns outputs plus the provider usage side-channel (ADJ-8).
func TestRunWithUsage_SuccessReturnsOutputsAndUsage(t *testing.T) {
	a := &availAdapter{}
	router, err := intel.NewRouter(
		[]intel.Registration{{Adapter: a, Locality: intel.LocalityCloud}},
		[]string{"avail"},
	)
	if err != nil {
		t.Fatalf("NewRouter(avail): %v", err)
	}
	r := New(intel.NewDispatcher(router))
	out, usage, serr := r.RunWithUsage(context.Background(), emptyCtx(), intelStep("draft", false))
	if serr != nil {
		t.Fatalf("unexpected serr = %+v", serr)
	}
	if out.Outputs["draft"] != "ok" {
		t.Fatalf("out.Outputs[draft] = %v, want ok", out.Outputs["draft"])
	}
	if usage == nil {
		t.Fatalf("usage = nil, want non-nil on success")
	}
	if usage.Adapter != "avail" || usage.Model != "avail-model" || usage.TokensUsed != 7 {
		t.Fatalf("usage = %+v, want {avail, avail-model, 7}", *usage)
	}
	if a.draftCalls != 1 {
		t.Fatalf("draftCalls = %d, want 1", a.draftCalls)
	}
}
