// Tests for MockIntelligence (M09-C2; PRD FR-SDK-08; Blueprint §27 "Intelligence Testing").
package awistesting

import (
	"context"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// ── OnDraft / OnEmbed / OnClassify fixtures returned correctly ────────────────

func TestMock_OnDraft(t *testing.T) {
	m := NewMockIntelligence()
	want := core.DraftResponse{Output: map[string]any{"result": "hello"}}
	m.OnDraft(want)

	got, err := m.Draft(context.Background(), core.DraftRequest{})
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	if got.Output["result"] != "hello" {
		t.Fatalf("Draft output[result] = %v, want hello", got.Output["result"])
	}
}

func TestMock_OnEmbed(t *testing.T) {
	m := NewMockIntelligence()
	vec := []float32{0.1, 0.2, 0.3}
	m.OnEmbed(vec)

	got, err := m.Embed(context.Background(), "test")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(got) != 3 || got[0] != 0.1 {
		t.Fatalf("Embed = %v, want %v", got, vec)
	}
}

func TestMock_OnClassify(t *testing.T) {
	m := NewMockIntelligence()
	m.OnClassify("cat-a", 0.9)

	got, err := m.Classify(context.Background(), "text", []string{"cat-a", "cat-b"})
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if got.Category != "cat-a" || got.Confidence != 0.9 {
		t.Fatalf("Classify = {%q, %v}, want {cat-a, 0.9}", got.Category, got.Confidence)
	}
}

// ── Un-fixtured capabilities return degradation errors (no panic) ─────────────

func TestMock_UnfixturedDraft_Degrades(t *testing.T) {
	m := NewMockIntelligence()
	_, err := m.Draft(context.Background(), core.DraftRequest{})
	if err == nil {
		t.Fatal("un-fixtured Draft: expected error, got nil")
	}
}

func TestMock_UnfixturedEmbed_Degrades(t *testing.T) {
	m := NewMockIntelligence()
	_, err := m.Embed(context.Background(), "text")
	if err == nil {
		t.Fatal("un-fixtured Embed: expected error, got nil")
	}
}

func TestMock_UnfixturedClassify_Degrades(t *testing.T) {
	m := NewMockIntelligence()
	_, err := m.Classify(context.Background(), "text", []string{"cat"})
	if err == nil {
		t.Fatal("un-fixtured Classify: expected error, got nil")
	}
}

// ── IsAvailable is true (router selects mock over NullAdapter) ────────────────

func TestMock_IsAvailable(t *testing.T) {
	m := NewMockIntelligence()
	if !m.IsAvailable() {
		t.Fatal("MockIntelligence.IsAvailable() = false, want true")
	}
}

// ── WithMockIntelligence: intelligence step carries fixture output ─────────────

func TestHarness_WithMockIntelligence_OnDraft(t *testing.T) {
	mock := NewMockIntelligence()
	mock.OnDraft(core.DraftResponse{
		Output: map[string]any{"draft": "generated-text"},
		Usage:  core.Usage{Adapter: "mock", Model: "mock", TokensUsed: 1},
	})

	h := NewHarness(t, WithMockIntelligence(mock))

	def, err := newIntelDef("intel-draft-flow")
	if err != nil {
		t.Fatalf("newIntelDef: %v", err)
	}
	if err := h.rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	id, err := h.rt.Submit(h.ctx, def.ID, nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	h.WaitForCompletion(id, 5*time.Second)

	// Step "gen" outputs are stored under "gen" in Variables.
	raw := h.GetOutput(id, "gen")
	m, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("GetOutput(gen) type = %T, want map[string]any", raw)
	}
	if m["draft"] != "generated-text" {
		t.Fatalf("gen[draft] = %v, want generated-text", m["draft"])
	}
}

// newIntelDef returns a minimal workflow definition with one "draft" intelligence
// step (required=false, fallback="" — no fallback, so capability_unavailable
// would fail; with a live mock this succeeds).
func newIntelDef(id string) (*core.WorkflowDefinition, error) {
	return &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            id,
		Version:       core.SemVer("1.0.0"),
		Namespace:     "test",
		Name:          id,
		Triggers:      []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{{
			ID:   "gen",
			Name: "gen",
			Type: core.StepTypeIntelligence,
			Intelligence: &core.IntelReq{
				Capability:    "draft",
				Required:      true,
				ContextBudget: 1000,
			},
		}},
		InitialStep: "gen",
		FinalSteps:  []string{"gen"},
		Metadata:    map[string]any{},
	}, nil
}
