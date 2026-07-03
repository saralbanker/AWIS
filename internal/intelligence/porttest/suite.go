// Package porttest provides an adapter-agnostic contract test suite for
// core.IntelligencePort implementations. It is designed to be reused verbatim
// for any adapter (NullAdapter M04, AnthropicAdapter M16, etc.) — the only
// variation is the factory function supplied by the caller.
//
// Usage:
//
//	func TestIntelligenceContract(t *testing.T) {
//	    porttest.Run(t, func(t *testing.T) core.IntelligencePort {
//	        return null.New()
//	    })
//	}
package porttest

import (
	"context"
	"testing"

	"github.com/awis/awis/internal/core"
)

func bg() context.Context { return context.Background() }

// Run executes the full IntelligencePort contract suite against the adapter
// returned by factory. Each subtest receives its own adapter instance from
// factory, matching the storagetest pattern.
func Run(t *testing.T, factory func(t *testing.T) core.IntelligencePort) {
	t.Helper()

	t.Run("CapabilitiesNonEmpty", func(t *testing.T) {
		testCapabilitiesNonEmpty(t, factory(t))
	})
	t.Run("ProviderNameStable", func(t *testing.T) {
		testProviderNameStable(t, factory(t))
	})
	t.Run("DeterministicAvailability", func(t *testing.T) {
		testDeterministicAvailability(t, factory(t))
	})
	t.Run("ClassifyContract", func(t *testing.T) {
		testClassifyContract(t, factory(t))
	})
	t.Run("SynthesizeContract", func(t *testing.T) {
		testSynthesizeContract(t, factory(t))
	})
	t.Run("DraftUsageContract", func(t *testing.T) {
		testDraftUsageContract(t, factory(t))
	})
}

// hasCapability reports whether the adapter declares the named capability.
func hasCapability(port core.IntelligencePort, name string) bool {
	for _, c := range port.Capabilities() {
		if c.Name == name {
			return true
		}
	}
	return false
}

// testCapabilitiesNonEmpty asserts that Capabilities() returns at least one
// entry and that all Names are non-empty and unique.
func testCapabilitiesNonEmpty(t *testing.T, port core.IntelligencePort) {
	t.Helper()
	caps := port.Capabilities()
	if len(caps) == 0 {
		t.Fatal("Capabilities: want ≥1 entry, got 0")
	}
	seen := make(map[string]bool, len(caps))
	for _, c := range caps {
		if c.Name == "" {
			t.Error("Capabilities: found entry with empty Name")
			continue
		}
		if seen[c.Name] {
			t.Errorf("Capabilities: duplicate name %q", c.Name)
		}
		seen[c.Name] = true
	}
}

// testProviderNameStable asserts that ProviderName() is non-empty and returns
// the identical string across two consecutive calls.
func testProviderNameStable(t *testing.T, port core.IntelligencePort) {
	t.Helper()
	n1 := port.ProviderName()
	n2 := port.ProviderName()
	if n1 == "" {
		t.Error("ProviderName: want non-empty, got empty string")
	}
	if n1 != n2 {
		t.Errorf("ProviderName: unstable across calls: %q vs %q", n1, n2)
	}
}

// testDeterministicAvailability asserts that IsAvailable() returns the same
// value on two consecutive calls (provider state must not change between them).
func testDeterministicAvailability(t *testing.T, port core.IntelligencePort) {
	t.Helper()
	a1 := port.IsAvailable()
	a2 := port.IsAvailable()
	if a1 != a2 {
		t.Errorf("IsAvailable: got %v then %v — must be identical across consecutive calls", a1, a2)
	}
}

// testClassifyContract asserts that if "classify" is declared:
//   - Classify with a non-empty categories slice returns a Category that is one
//     of the supplied values and a Confidence in [0,1].
//   - Classify with an empty categories slice returns a non-nil error.
func testClassifyContract(t *testing.T, port core.IntelligencePort) {
	t.Helper()
	if !hasCapability(port, "classify") {
		t.Skip("classify not declared by this adapter")
	}

	// Non-empty categories: result must be within the set, confidence in [0,1].
	categories := []string{"a", "b"}
	result, err := port.Classify(bg(), "test input", categories)
	if err != nil {
		t.Fatalf("Classify(%v): unexpected error: %v", categories, err)
	}
	found := false
	for _, c := range categories {
		if result.Category == c {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Classify.Category %q not in supplied set %v", result.Category, categories)
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Classify.Confidence %v not in [0,1]", result.Confidence)
	}

	// Empty categories: must return non-nil error.
	_, err = port.Classify(bg(), "test input", []string{})
	if err == nil {
		t.Error("Classify with empty categories: want error, got nil")
	}
}

// testSynthesizeContract asserts that if "synthesize" is declared:
//   - Synthesize does not panic.
//   - If both of two identical calls succeed, Usage.Adapter equals ProviderName().
//     (determinism of Text is NOT required for cloud adapters).
func testSynthesizeContract(t *testing.T, port core.IntelligencePort) {
	t.Helper()
	if !hasCapability(port, "synthesize") {
		t.Skip("synthesize not declared by this adapter")
	}

	req := core.SynthesisRequest{Query: "contract test", Entries: []any{"entry1"}, MaxLen: 100}
	r1, err1 := port.Synthesize(bg(), req)
	r2, err2 := port.Synthesize(bg(), req)

	// Both calls must not panic (ensured by reaching here); log errors but do
	// not fail — cloud adapters may legitimately return errors in test context.
	if err1 != nil && err2 != nil {
		t.Logf("Synthesize: both calls returned errors: %v, %v (accepted for cloud adapters)", err1, err2)
		return
	}

	// When a call succeeds, Usage.Adapter must equal ProviderName().
	provider := port.ProviderName()
	if err1 == nil && r1.Usage.Adapter != provider {
		t.Errorf("Synthesize r1 Usage.Adapter: want %q, got %q", provider, r1.Usage.Adapter)
	}
	if err2 == nil && r2.Usage.Adapter != provider {
		t.Errorf("Synthesize r2 Usage.Adapter: want %q, got %q", provider, r2.Usage.Adapter)
	}
}

// testDraftUsageContract asserts that if "draft" is declared and Draft succeeds,
// Usage.Adapter equals ProviderName().
func testDraftUsageContract(t *testing.T, port core.IntelligencePort) {
	t.Helper()
	if !hasCapability(port, "draft") {
		t.Skip("draft not declared by this adapter")
	}

	req := core.DraftRequest{Context: "contract test"}
	resp, err := port.Draft(bg(), req)
	if err != nil {
		t.Logf("Draft: returned error: %v (accepted for cloud adapters)", err)
		return
	}

	provider := port.ProviderName()
	if resp.Usage.Adapter != provider {
		t.Errorf("Draft Usage.Adapter: want %q, got %q", provider, resp.Usage.Adapter)
	}
}
