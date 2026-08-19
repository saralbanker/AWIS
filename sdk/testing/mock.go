// MockIntelligence: fixture-driven IntelligencePort for tests (PRD FR-SDK-08;
// Blueprint §27 "Intelligence Testing"). Part of package awistesting.
package awistesting

import (
	"context"
	"errors"
	"sync"

	"github.com/awis/awis/internal/core"
)

// compile-time assertion: *MockIntelligence satisfies core.IntelligencePort.
var _ core.IntelligencePort = (*MockIntelligence)(nil)

// MockIntelligence is a fixture-driven implementation of core.IntelligencePort.
// Each On* method registers a canned response for the corresponding capability.
// Un-fixtured capabilities return an error mirroring NullAdapter degradation
// semantics. Construct via NewMockIntelligence.
type MockIntelligence struct {
	mu sync.Mutex

	// draft fixture
	hasDraft  bool
	draftResp core.DraftResponse

	// embed fixture
	hasEmbed bool
	embedVec []float32

	// classify fixture
	hasClassify  bool
	classifyResp classifyFixture

	// synthesize: not part of FR-SDK-08 exported surface; returns zero-value.
	hasSynthesize  bool
	synthesizeResp core.SynthesisResponse
}

// classifyFixture holds the canned classify response.
type classifyFixture struct {
	category   string
	confidence float64
}

// NewMockIntelligence constructs a MockIntelligence with no fixtures set.
// Un-fixtured capabilities return a degradation error until On* is called.
func NewMockIntelligence() *MockIntelligence {
	return &MockIntelligence{}
}

// OnDraft sets the canned DraftResponse returned by every Draft call.
func (m *MockIntelligence) OnDraft(resp core.DraftResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.draftResp = resp
	m.hasDraft = true
}

// OnEmbed sets the canned embedding vector returned by every Embed call.
func (m *MockIntelligence) OnEmbed(vec []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embedVec = vec
	m.hasEmbed = true
}

// OnClassify sets the canned category and confidence returned by every
// Classify call (FR-IL-10: Classify is a declared-but-not-runtime-invoked
// placeholder; the harness exercises it via direct port call).
func (m *MockIntelligence) OnClassify(category string, confidence float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.classifyResp = classifyFixture{category: category, confidence: confidence}
	m.hasClassify = true
}

// Draft returns the canned DraftResponse set by OnDraft. If OnDraft has not
// been called, Draft returns an error mirroring NullAdapter degradation.
func (m *MockIntelligence) Draft(_ context.Context, _ core.DraftRequest) (core.DraftResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasDraft {
		return core.DraftResponse{}, errors.New("awistesting: MockIntelligence: Draft not fixtured")
	}
	return m.draftResp, nil
}

// Embed returns the canned vector set by OnEmbed. If OnEmbed has not been
// called, Embed returns an error mirroring NullAdapter degradation.
func (m *MockIntelligence) Embed(_ context.Context, _ string) ([]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasEmbed {
		return nil, errors.New("awistesting: MockIntelligence: Embed not fixtured")
	}
	return m.embedVec, nil
}

// Synthesize returns the canned SynthesisResponse if set, otherwise returns
// a zero-value response (no error). Synthesize is not part of the FR-SDK-08
// exported fixture surface; the zero-value default is intentional.
func (m *MockIntelligence) Synthesize(_ context.Context, _ core.SynthesisRequest) (core.SynthesisResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasSynthesize {
		return core.SynthesisResponse{}, nil
	}
	return m.synthesizeResp, nil
}

// Classify returns the canned classification set by OnClassify. If OnClassify
// has not been called, Classify returns an error mirroring NullAdapter
// degradation semantics. Requires at least one category (FR-IL-10).
func (m *MockIntelligence) Classify(_ context.Context, _ string, categories []string) (core.Classification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(categories) == 0 {
		return core.Classification{}, errors.New("awistesting: MockIntelligence: Classify requires at least one category")
	}
	if !m.hasClassify {
		return core.Classification{}, errors.New("awistesting: MockIntelligence: Classify not fixtured")
	}
	return core.Classification{
		Category:   m.classifyResp.category,
		Confidence: float32(m.classifyResp.confidence),
	}, nil
}

// IsAvailable reports whether the mock is available. Returns true so that the
// intelligence router selects the mock rather than falling through to the
// NullAdapter.
func (m *MockIntelligence) IsAvailable() bool { return true }

// Capabilities declares the four standard capabilities.
func (m *MockIntelligence) Capabilities() []core.Capability {
	return []core.Capability{
		{Name: "draft"},
		{Name: "embed"},
		{Name: "synthesize"},
		{Name: "classify"},
	}
}

// ProviderName returns the stable provider name "mock".
func (m *MockIntelligence) ProviderName() string { return "mock" }
