package null_test

import (
	"context"
	"errors"
	"testing"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/intelligence/adapters/null"
)

func bg() context.Context { return context.Background() }

// TestIsAvailable_AlwaysFalse asserts the adapter is never available.
func TestIsAvailable_AlwaysFalse(t *testing.T) {
	a := null.New()
	if a.IsAvailable() {
		t.Error("IsAvailable: want false, got true")
	}
}

// TestProviderName asserts the stable name.
func TestProviderName(t *testing.T) {
	a := null.New()
	if got := a.ProviderName(); got != "null" {
		t.Errorf("ProviderName: want %q, got %q", "null", got)
	}
}

// TestCapabilities asserts all four capabilities are declared with non-empty names.
func TestCapabilities(t *testing.T) {
	a := null.New()
	caps := a.Capabilities()
	if len(caps) != 4 {
		t.Fatalf("Capabilities: want 4 entries, got %d", len(caps))
	}
	want := map[string]bool{"draft": true, "embed": true, "synthesize": true, "classify": true}
	for _, c := range caps {
		if c.Name == "" {
			t.Error("Capabilities: found entry with empty Name")
		}
		if !want[c.Name] {
			t.Errorf("Capabilities: unexpected name %q", c.Name)
		}
		delete(want, c.Name)
	}
	for name := range want {
		t.Errorf("Capabilities: missing %q", name)
	}
}

// TestDraft_DefaultEmptyOutput asserts the zero-config default returns empty Output.
func TestDraft_DefaultEmptyOutput(t *testing.T) {
	a := null.New()
	resp, err := a.Draft(bg(), core.DraftRequest{Context: "test"})
	if err != nil {
		t.Fatalf("Draft: unexpected error: %v", err)
	}
	if resp.Output == nil {
		t.Error("Draft: Output must not be nil (want empty map)")
	}
	if len(resp.Output) != 0 {
		t.Errorf("Draft: want empty Output, got %v", resp.Output)
	}
}

// TestDraft_DefaultDeterministic asserts two calls return equal results.
func TestDraft_DefaultDeterministic(t *testing.T) {
	a := null.New()
	req := core.DraftRequest{Context: "test"}
	r1, err1 := a.Draft(bg(), req)
	r2, err2 := a.Draft(bg(), req)
	if err1 != nil || err2 != nil {
		t.Fatalf("Draft: errors: %v, %v", err1, err2)
	}
	if len(r1.Output) != len(r2.Output) {
		t.Errorf("Draft: outputs differ: %v vs %v", r1.Output, r2.Output)
	}
}

// TestDraft_Fixture asserts WithDraftFixture is returned on every call.
func TestDraft_Fixture(t *testing.T) {
	fixture := map[string]any{"answer": "42"}
	a := null.New(null.WithDraftFixture(fixture))
	resp, err := a.Draft(bg(), core.DraftRequest{})
	if err != nil {
		t.Fatalf("Draft: unexpected error: %v", err)
	}
	if resp.Output["answer"] != "42" {
		t.Errorf("Draft fixture: want answer=42, got %v", resp.Output["answer"])
	}
}

// TestDraft_Error asserts WithDraftError makes Draft return the error.
func TestDraft_Error(t *testing.T) {
	sentinel := errors.New("forced error")
	a := null.New(null.WithDraftError(sentinel))
	_, err := a.Draft(bg(), core.DraftRequest{})
	if !errors.Is(err, sentinel) {
		t.Errorf("Draft error: want sentinel, got %v", err)
	}
}

// TestDraft_Usage asserts the usage fields on a successful Draft response.
func TestDraft_Usage(t *testing.T) {
	a := null.New()
	resp, err := a.Draft(bg(), core.DraftRequest{})
	if err != nil {
		t.Fatalf("Draft: unexpected error: %v", err)
	}
	if resp.Usage.Adapter != "null" {
		t.Errorf("Usage.Adapter: want %q, got %q", "null", resp.Usage.Adapter)
	}
	if resp.Usage.Model != "null" {
		t.Errorf("Usage.Model: want %q, got %q", "null", resp.Usage.Model)
	}
	if resp.Usage.TokensUsed != 0 {
		t.Errorf("Usage.TokensUsed: want 0, got %d", resp.Usage.TokensUsed)
	}
}

// TestEmbed_ZeroVector asserts Embed returns a zero vector of length 768.
func TestEmbed_ZeroVector(t *testing.T) {
	a := null.New()
	vec, err := a.Embed(bg(), "hello")
	if err != nil {
		t.Fatalf("Embed: unexpected error: %v", err)
	}
	if len(vec) != 768 {
		t.Errorf("Embed: want length 768, got %d", len(vec))
	}
	for i, v := range vec {
		if v != 0 {
			t.Errorf("Embed: vec[%d] = %v, want 0", i, v)
			break
		}
	}
}

// TestSynthesize_FrozenLiteral asserts the verbatim frozen response text.
func TestSynthesize_FrozenLiteral(t *testing.T) {
	const want = "The record is silent."
	a := null.New()
	resp, err := a.Synthesize(bg(), core.SynthesisRequest{Query: "anything"})
	if err != nil {
		t.Fatalf("Synthesize: unexpected error: %v", err)
	}
	if resp.Text != want {
		t.Errorf("Synthesize.Text: want %q, got %q", want, resp.Text)
	}
}

// TestSynthesize_Usage asserts the usage fields on a Synthesize response.
func TestSynthesize_Usage(t *testing.T) {
	a := null.New()
	resp, err := a.Synthesize(bg(), core.SynthesisRequest{})
	if err != nil {
		t.Fatalf("Synthesize: unexpected error: %v", err)
	}
	if resp.Usage.Adapter != "null" {
		t.Errorf("Synthesize.Usage.Adapter: want %q, got %q", "null", resp.Usage.Adapter)
	}
	if resp.Usage.Model != "null" {
		t.Errorf("Synthesize.Usage.Model: want %q, got %q", "null", resp.Usage.Model)
	}
	if resp.Usage.TokensUsed != 0 {
		t.Errorf("Synthesize.Usage.TokensUsed: want 0, got %d", resp.Usage.TokensUsed)
	}
}

// TestClassify_FirstCategory asserts the first category is returned with confidence 0.0.
func TestClassify_FirstCategory(t *testing.T) {
	a := null.New()
	result, err := a.Classify(bg(), "some text", []string{"alpha", "beta"})
	if err != nil {
		t.Fatalf("Classify: unexpected error: %v", err)
	}
	if result.Category != "alpha" {
		t.Errorf("Classify.Category: want %q, got %q", "alpha", result.Category)
	}
	if result.Confidence != 0.0 {
		t.Errorf("Classify.Confidence: want 0.0, got %v", result.Confidence)
	}
}

// TestClassify_EmptyCategories asserts the required error when slice is empty.
func TestClassify_EmptyCategories(t *testing.T) {
	a := null.New()
	_, err := a.Classify(bg(), "some text", []string{})
	if err == nil {
		t.Fatal("Classify with empty categories: want error, got nil")
	}
	const wantMsg = "null adapter: classify requires at least one category"
	if err.Error() != wantMsg {
		t.Errorf("Classify error message: want %q, got %q", wantMsg, err.Error())
	}
}
