package intelligence

import (
	"context"
	"errors"
	"testing"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/intelligence/adapters/null"
)

// bg returns a background context for use in tests.
func bg() context.Context { return context.Background() }

// fakeAdapter is a configurable core.IntelligencePort for use in intelligence
// package tests. Availability is expressed as a func so that
// TestRouteAvailabilityAtCallTime can change the adapter's state between Eligible
// calls without replacing the registered instance.
//
// Usage on fake successes: set draftResp with Usage.Adapter = name to match the
// ProviderName contract; for router tests the response body is usually irrelevant.
type fakeAdapter struct {
	name       string      // returned by ProviderName
	availFn    func() bool // called by IsAvailable; returns false when nil
	caps       []string    // capability names declared by Capabilities
	draftErr   error       // returned by Draft when non-nil (takes precedence)
	draftResp  core.DraftResponse
	synthErr   error // returned by Synthesize when non-nil
	synthResp  core.SynthesisResponse
	draftCalls int // incremented on every Draft call, single-goroutine tests only
}

// compile-time assertion: *fakeAdapter satisfies core.IntelligencePort.
var _ core.IntelligencePort = (*fakeAdapter)(nil)

func (f *fakeAdapter) Draft(_ context.Context, _ core.DraftRequest) (core.DraftResponse, error) {
	f.draftCalls++
	if f.draftErr != nil {
		return core.DraftResponse{}, f.draftErr
	}
	return f.draftResp, nil
}

func (f *fakeAdapter) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}

func (f *fakeAdapter) Synthesize(_ context.Context, _ core.SynthesisRequest) (core.SynthesisResponse, error) {
	if f.synthErr != nil {
		return core.SynthesisResponse{}, f.synthErr
	}
	return f.synthResp, nil
}

func (f *fakeAdapter) Classify(_ context.Context, _ string, categories []string) (core.Classification, error) {
	if len(categories) == 0 {
		return core.Classification{}, nil
	}
	return core.Classification{Category: categories[0]}, nil
}

func (f *fakeAdapter) IsAvailable() bool {
	if f.availFn != nil {
		return f.availFn()
	}
	return false
}

func (f *fakeAdapter) Capabilities() []core.Capability {
	caps := make([]core.Capability, len(f.caps))
	for i, c := range f.caps {
		caps[i] = core.Capability{Name: c}
	}
	return caps
}

func (f *fakeAdapter) ProviderName() string { return f.name }

// available returns an availFn closure that always returns v.
func available(v bool) func() bool { return func() bool { return v } }

// --- Tests 1–11: Blueprint §17 decision tree, router contract ---

// TestRouteNoEligibleRequired: no eligible adapter × required=true →
// CapabilityUnavailableError (errors.As).
func TestRouteNoEligibleRequired(t *testing.T) {
	a := &fakeAdapter{name: "a", availFn: available(false), caps: []string{"draft"}}
	router, err := NewRouter(
		[]Registration{{Adapter: a, Locality: LocalityCloud}},
		[]string{"a"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	_, err = router.Route("draft", "", true)
	if err == nil {
		t.Fatal("Route: want error, got nil")
	}
	var capErr CapabilityUnavailableError
	if !errors.As(err, &capErr) {
		t.Errorf("errors.As CapabilityUnavailableError: not found in %T: %v", err, err)
	}
	if capErr.Capability != "draft" {
		t.Errorf("CapabilityUnavailableError.Capability: want %q, got %q", "draft", capErr.Capability)
	}
}

// TestRouteNoEligibleFallback: no eligible adapter × required=false →
// FallbackSignal (errors.As).
func TestRouteNoEligibleFallback(t *testing.T) {
	a := &fakeAdapter{name: "a", availFn: available(false), caps: []string{"draft"}}
	router, err := NewRouter(
		[]Registration{{Adapter: a, Locality: LocalityCloud}},
		[]string{"a"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	_, err = router.Route("draft", "", false)
	if err == nil {
		t.Fatal("Route: want error, got nil")
	}
	var sig FallbackSignal
	if !errors.As(err, &sig) {
		t.Errorf("errors.As FallbackSignal: not found in %T: %v", err, err)
	}
	if sig.Capability != "draft" {
		t.Errorf("FallbackSignal.Capability: want %q, got %q", "draft", sig.Capability)
	}
}

// TestRouteHintLocalPrefersLocal: local and cloud adapters both eligible,
// hint="local" → local adapter leads the Eligible list.
func TestRouteHintLocalPrefersLocal(t *testing.T) {
	// Chain order: cloud first, local second — to prove hint reorders them.
	cloud := &fakeAdapter{name: "cloud", availFn: available(true), caps: []string{"draft"}}
	local := &fakeAdapter{name: "local", availFn: available(true), caps: []string{"draft"}}
	router, err := NewRouter(
		[]Registration{
			{Adapter: cloud, Locality: LocalityCloud},
			{Adapter: local, Locality: LocalityLocal},
		},
		[]string{"cloud", "local"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	list := router.Eligible("draft", "local")
	if len(list) == 0 {
		t.Fatal("Eligible: want entries, got none")
	}
	if list[0].Registration.Locality != LocalityLocal {
		t.Errorf("Eligible[0]: want LocalityLocal, got %v (name=%s)",
			list[0].Registration.Locality, list[0].Adapter.ProviderName())
	}
}

// TestRouteHintLocalFallsToCloud: only cloud adapter is eligible, hint="local"
// → cloud adapter is still returned (no local available → fall back to chain).
func TestRouteHintLocalFallsToCloud(t *testing.T) {
	cloud := &fakeAdapter{name: "cloud", availFn: available(true), caps: []string{"draft"}}
	router, err := NewRouter(
		[]Registration{{Adapter: cloud, Locality: LocalityCloud}},
		[]string{"cloud"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	list := router.Eligible("draft", "local")
	if len(list) != 1 {
		t.Fatalf("Eligible: want 1, got %d", len(list))
	}
	if list[0].Adapter.ProviderName() != "cloud" {
		t.Errorf("Eligible[0]: want cloud, got %s", list[0].Adapter.ProviderName())
	}
}

// TestRouteHintFastCheapestCloud: hint="fast" picks ascending CostRank; on tie
// chain order is preserved (stable sort).
func TestRouteHintFastCheapestCloud(t *testing.T) {
	t.Run("CostRank 2,1 → rank-1 first", func(t *testing.T) {
		expensive := &fakeAdapter{name: "expensive", availFn: available(true), caps: []string{"draft"}}
		cheap := &fakeAdapter{name: "cheap", availFn: available(true), caps: []string{"draft"}}
		// Chain order: expensive first, cheap second — hint must invert.
		router, err := NewRouter(
			[]Registration{
				{Adapter: expensive, Locality: LocalityCloud, CostRank: 2},
				{Adapter: cheap, Locality: LocalityCloud, CostRank: 1},
			},
			[]string{"expensive", "cheap"},
		)
		if err != nil {
			t.Fatalf("NewRouter: %v", err)
		}
		list := router.Eligible("draft", "fast")
		if len(list) < 2 {
			t.Fatalf("Eligible: want ≥2, got %d", len(list))
		}
		if list[0].Adapter.ProviderName() != "cheap" {
			t.Errorf("Eligible[0]: want cheap (CostRank 1), got %s", list[0].Adapter.ProviderName())
		}
		if list[1].Adapter.ProviderName() != "expensive" {
			t.Errorf("Eligible[1]: want expensive (CostRank 2), got %s", list[1].Adapter.ProviderName())
		}
	})

	t.Run("CostRank tie → chain order", func(t *testing.T) {
		a := &fakeAdapter{name: "a", availFn: available(true), caps: []string{"draft"}}
		b := &fakeAdapter{name: "b", availFn: available(true), caps: []string{"draft"}}
		router, err := NewRouter(
			[]Registration{
				{Adapter: a, Locality: LocalityCloud, CostRank: 1},
				{Adapter: b, Locality: LocalityCloud, CostRank: 1},
			},
			[]string{"a", "b"},
		)
		if err != nil {
			t.Fatalf("NewRouter: %v", err)
		}
		list := router.Eligible("draft", "fast")
		if len(list) < 2 {
			t.Fatalf("Eligible: want ≥2, got %d", len(list))
		}
		if list[0].Adapter.ProviderName() != "a" {
			t.Errorf("Eligible[0]: want a (tie → chain order), got %s", list[0].Adapter.ProviderName())
		}
	})
}

// TestRouteHintQualityBestCloud: hint="quality" picks ascending QualityRank; on
// tie chain order is preserved (stable sort).
func TestRouteHintQualityBestCloud(t *testing.T) {
	t.Run("QualityRank 2,1 → rank-1 first", func(t *testing.T) {
		lowQ := &fakeAdapter{name: "low-quality", availFn: available(true), caps: []string{"draft"}}
		highQ := &fakeAdapter{name: "high-quality", availFn: available(true), caps: []string{"draft"}}
		// Chain order: low-quality first, high-quality second.
		router, err := NewRouter(
			[]Registration{
				{Adapter: lowQ, Locality: LocalityCloud, QualityRank: 2},
				{Adapter: highQ, Locality: LocalityCloud, QualityRank: 1},
			},
			[]string{"low-quality", "high-quality"},
		)
		if err != nil {
			t.Fatalf("NewRouter: %v", err)
		}
		list := router.Eligible("draft", "quality")
		if len(list) < 2 {
			t.Fatalf("Eligible: want ≥2, got %d", len(list))
		}
		if list[0].Adapter.ProviderName() != "high-quality" {
			t.Errorf("Eligible[0]: want high-quality (QualityRank 1), got %s", list[0].Adapter.ProviderName())
		}
		if list[1].Adapter.ProviderName() != "low-quality" {
			t.Errorf("Eligible[1]: want low-quality (QualityRank 2), got %s", list[1].Adapter.ProviderName())
		}
	})

	t.Run("QualityRank tie → chain order", func(t *testing.T) {
		a := &fakeAdapter{name: "a", availFn: available(true), caps: []string{"draft"}}
		b := &fakeAdapter{name: "b", availFn: available(true), caps: []string{"draft"}}
		router, err := NewRouter(
			[]Registration{
				{Adapter: a, Locality: LocalityCloud, QualityRank: 1},
				{Adapter: b, Locality: LocalityCloud, QualityRank: 1},
			},
			[]string{"a", "b"},
		)
		if err != nil {
			t.Fatalf("NewRouter: %v", err)
		}
		list := router.Eligible("draft", "quality")
		if len(list) < 2 {
			t.Fatalf("Eligible: want ≥2, got %d", len(list))
		}
		if list[0].Adapter.ProviderName() != "a" {
			t.Errorf("Eligible[0]: want a (tie → chain order), got %s", list[0].Adapter.ProviderName())
		}
	})
}

// TestRouteHintEmptyChainOrder: hint="" → strict chain order; an unknown hint
// value behaves identically.
func TestRouteHintEmptyChainOrder(t *testing.T) {
	a := &fakeAdapter{name: "a", availFn: available(true), caps: []string{"draft"}}
	b := &fakeAdapter{name: "b", availFn: available(true), caps: []string{"draft"}}
	router, err := NewRouter(
		[]Registration{
			{Adapter: a, Locality: LocalityCloud},
			{Adapter: b, Locality: LocalityCloud},
		},
		[]string{"a", "b"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	for _, hint := range []string{"", "unrecognised-hint"} {
		t.Run("hint="+hint, func(t *testing.T) {
			list := router.Eligible("draft", hint)
			if len(list) < 2 {
				t.Fatalf("Eligible: want ≥2, got %d", len(list))
			}
			if list[0].Adapter.ProviderName() != "a" {
				t.Errorf("Eligible[0]: want a (chain order), got %s", list[0].Adapter.ProviderName())
			}
			if list[1].Adapter.ProviderName() != "b" {
				t.Errorf("Eligible[1]: want b (chain order), got %s", list[1].Adapter.ProviderName())
			}
		})
	}
}

// TestRouteAvailabilityAtCallTime: IsAvailable is evaluated at Eligible call
// time — changing availability between calls changes the result.
func TestRouteAvailabilityAtCallTime(t *testing.T) {
	avail := false
	a := &fakeAdapter{
		name:    "a",
		availFn: func() bool { return avail },
		caps:    []string{"draft"},
	}
	router, err := NewRouter(
		[]Registration{{Adapter: a, Locality: LocalityCloud}},
		[]string{"a"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	// Call 1: unavailable → Eligible is empty.
	list := router.Eligible("draft", "")
	if len(list) != 0 {
		t.Errorf("Eligible (unavailable): want 0, got %d", len(list))
	}

	// Flip availability.
	avail = true

	// Call 2: now available → Eligible contains the adapter.
	list = router.Eligible("draft", "")
	if len(list) != 1 {
		t.Errorf("Eligible (available): want 1, got %d", len(list))
	}
	if list[0].Adapter.ProviderName() != "a" {
		t.Errorf("Eligible[0]: want a, got %s", list[0].Adapter.ProviderName())
	}
}

// TestRouteChainIsUniverse: an adapter registered but absent from the fallback
// chain is never returned by Eligible, even when capable and available.
func TestRouteChainIsUniverse(t *testing.T) {
	inChain := &fakeAdapter{name: "in-chain", availFn: available(true), caps: []string{"draft"}}
	notInChain := &fakeAdapter{name: "not-in-chain", availFn: available(true), caps: []string{"draft"}}
	router, err := NewRouter(
		[]Registration{
			{Adapter: inChain, Locality: LocalityCloud},
			{Adapter: notInChain, Locality: LocalityCloud},
		},
		[]string{"in-chain"}, // only in-chain is in the universe
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	list := router.Eligible("draft", "")
	if len(list) != 1 {
		t.Fatalf("Eligible: want 1, got %d", len(list))
	}
	if list[0].Adapter.ProviderName() != "in-chain" {
		t.Errorf("Eligible[0]: want in-chain, got %s", list[0].Adapter.ProviderName())
	}
}

// TestRouteNullNeverSelected: the real null.Adapter declares all capabilities but
// IsAvailable is always false, so it is never selected. When it is the only chain
// entry, Route returns the correct typed error per the required flag.
func TestRouteNullNeverSelected(t *testing.T) {
	na := null.New()
	router, err := NewRouter(
		[]Registration{{Adapter: na, Locality: LocalityLocal}},
		[]string{"null"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	// Eligible must be empty: null is in chain, declares "draft", but IsAvailable=false.
	list := router.Eligible("draft", "")
	if len(list) != 0 {
		t.Errorf("Eligible: want 0 (null not available), got %d", len(list))
	}

	// required=true → CapabilityUnavailableError.
	_, err = router.Route("draft", "", true)
	var capErr CapabilityUnavailableError
	if !errors.As(err, &capErr) {
		t.Errorf("Route(required=true): want CapabilityUnavailableError, got %T: %v", err, err)
	}

	// required=false → FallbackSignal.
	_, err = router.Route("draft", "", false)
	var sig FallbackSignal
	if !errors.As(err, &sig) {
		t.Errorf("Route(required=false): want FallbackSignal, got %T: %v", err, err)
	}
}

// TestNewRouterUnknownChainEntry: a fallback chain entry naming an unregistered
// adapter causes NewRouter to return a non-nil error.
func TestNewRouterUnknownChainEntry(t *testing.T) {
	a := &fakeAdapter{name: "a", availFn: available(true), caps: []string{"draft"}}
	_, err := NewRouter(
		[]Registration{{Adapter: a, Locality: LocalityCloud}},
		[]string{"a", "ghost"}, // "ghost" has no registration
	)
	if err == nil {
		t.Fatal("NewRouter: want error for unregistered chain entry, got nil")
	}
}
