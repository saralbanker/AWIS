// Package null provides the NullAdapter: a deterministic, always-degraded
// IntelligencePort implementation used in test and fallback scenarios
// (Blueprint §14, FR-IL-01). It never reports itself as available
// (IsAvailable returns false), so the router treats it as a degraded-mode
// last resort rather than an eligible provider.
package null

import (
	"context"
	"errors"

	"github.com/awis/awis/internal/core"
)

// Compile-time assertion: *Adapter must satisfy core.IntelligencePort.
var _ core.IntelligencePort = (*Adapter)(nil)

// Adapter is the NullAdapter. The zero value is valid and returns deterministic
// empty outputs. Use New with options to customise fixture data or error modes.
type Adapter struct {
	draftFixture map[string]any
	draftErr     error
}

// Option configures an Adapter at construction time.
type Option func(*Adapter)

// WithDraftFixture sets the output map returned by every Draft call.
// If not set, Draft returns an empty map[string]any{}.
func WithDraftFixture(fixture map[string]any) Option {
	return func(a *Adapter) { a.draftFixture = fixture }
}

// WithDraftError makes every Draft call return the supplied error instead of
// a fixture. Takes precedence over WithDraftFixture.
func WithDraftError(err error) Option {
	return func(a *Adapter) { a.draftErr = err }
}

// New constructs a NullAdapter with the supplied options. Zero-config default:
// Draft returns a deterministic empty Output; no error.
func New(opts ...Option) *Adapter {
	a := &Adapter{}
	for _, o := range opts {
		o(a)
	}
	return a
}

// nullUsage is the Usage stamped on every NullAdapter response.
func nullUsage() core.Usage {
	return core.Usage{Adapter: "null", Model: "null", TokensUsed: 0}
}

// Draft returns the configured fixture (or empty map) deterministically.
// If WithDraftError was supplied, it returns that error instead.
// Usage: {Adapter: "null", Model: "null", TokensUsed: 0}.
func (a *Adapter) Draft(_ context.Context, _ core.DraftRequest) (core.DraftResponse, error) {
	if a.draftErr != nil {
		return core.DraftResponse{}, a.draftErr
	}
	out := a.draftFixture
	if out == nil {
		out = map[string]any{}
	}
	return core.DraftResponse{Output: out, Usage: nullUsage()}, nil
}

// Embed returns the null zero vector of length 768 (all zeros, conventional
// embedding dimension). CONTRA-3: Embed is unavailable in V1; the zero vector
// is the null response per Blueprint §14.
func (a *Adapter) Embed(_ context.Context, _ string) ([]float32, error) {
	return make([]float32, 768), nil
}

// Synthesize returns the frozen literal "The record is silent." per
// Blueprint §14 (FR-IL-01). Usage: {Adapter: "null", Model: "null", TokensUsed: 0}.
func (a *Adapter) Synthesize(_ context.Context, _ core.SynthesisRequest) (core.SynthesisResponse, error) {
	return core.SynthesisResponse{Text: "The record is silent.", Usage: nullUsage()}, nil
}

// Classify returns the first category with Confidence 0.0 (FR-IL-10 placeholder).
// Returns an error if categories is empty.
func (a *Adapter) Classify(_ context.Context, _ string, categories []string) (core.Classification, error) {
	if len(categories) == 0 {
		return core.Classification{}, errors.New("null adapter: classify requires at least one category")
	}
	return core.Classification{Category: categories[0], Confidence: 0.0}, nil
}

// IsAvailable always returns false. The NullAdapter is a degraded-mode adapter;
// the router never selects it as an available provider (Blueprint §14, FR-IL-01).
func (a *Adapter) IsAvailable() bool { return false }

// Capabilities declares all four capability names the NullAdapter nominally
// implements: "draft", "embed", "synthesize", "classify".
func (a *Adapter) Capabilities() []core.Capability {
	return []core.Capability{
		{Name: "draft"},
		{Name: "embed"},
		{Name: "synthesize"},
		{Name: "classify"},
	}
}

// ProviderName returns the stable name "null".
func (a *Adapter) ProviderName() string { return "null" }
