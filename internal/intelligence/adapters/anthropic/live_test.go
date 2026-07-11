package anthropic

// live_test.go — live smoke test for the Anthropic adapter.
//
// This test is SKIPPED in CI (no network). It runs only when both environment
// variables are set:
//
//	ANTHROPIC_LIVE=1
//	ANTHROPIC_API_KEY=<real key>
//
// IMP Val row: "manual".
//
// Run manually:
//
//	ANTHROPIC_LIVE=1 ANTHROPIC_API_KEY=sk-ant-... go test ./internal/intelligence/adapters/anthropic/ -run TestLiveSmoke -v

import (
	"context"
	"os"
	"testing"

	"github.com/awis/awis/internal/core"
)

// TestLiveSmoke is a live integration smoke test that exercises Draft and
// Synthesize against the real Anthropic API.
func TestLiveSmoke(t *testing.T) {
	if os.Getenv("ANTHROPIC_LIVE") != "1" {
		t.Skip("ANTHROPIC_LIVE not set; skipping live smoke test")
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set; skipping live smoke test")
	}

	a := New(Config{APIKey: apiKey, MaxAttempts: 2})

	if !a.IsAvailable() {
		t.Fatal("IsAvailable: want true with API key set")
	}

	t.Run("Draft", func(t *testing.T) {
		resp, err := a.Draft(context.Background(), core.DraftRequest{
			Context: "Describe the number 42 in one sentence.",
		})
		if err != nil {
			t.Fatalf("Draft: %v", err)
		}
		if len(resp.Output) == 0 {
			t.Fatal("Draft.Output: want non-empty map, got empty")
		}
		if resp.Usage.TokensUsed <= 0 {
			t.Fatalf("Draft.Usage.TokensUsed: want >0, got %d", resp.Usage.TokensUsed)
		}
		t.Logf("Draft.Output: %v", resp.Output)
		t.Logf("Draft.Usage: adapter=%s model=%s tokens=%d",
			resp.Usage.Adapter, resp.Usage.Model, resp.Usage.TokensUsed)
	})

	t.Run("Synthesize", func(t *testing.T) {
		resp, err := a.Synthesize(context.Background(), core.SynthesisRequest{
			Query:   "What is the capital of France?",
			Entries: []any{"France is a country in Western Europe.", "Paris is its capital city."},
			MaxLen:  50,
		})
		if err != nil {
			t.Fatalf("Synthesize: %v", err)
		}
		if resp.Text == "" {
			t.Fatal("Synthesize.Text: want non-empty")
		}
		t.Logf("Synthesize.Text: %s", resp.Text)
		t.Logf("Synthesize.Usage: adapter=%s model=%s tokens=%d",
			resp.Usage.Adapter, resp.Usage.Model, resp.Usage.TokensUsed)
	})

	t.Run("EmbedUnavailable", func(t *testing.T) {
		_, err := a.Embed(context.Background(), "test")
		if err == nil {
			t.Fatal("Embed: want ErrEmbedUnavailable, got nil")
		}
	})
}
