package sdk

import "github.com/awis/awis/internal/core"

// IntelligencePort is the translation layer between step declarations and
// provider implementations. It exposes Draft, Embed, Synthesize, and Classify
// plus provider introspection (IsAvailable, Capabilities, ProviderName). If the
// layer is absent the runtime routes intelligence steps to their fallbacks.
// Classify is a non-callable placeholder (FR-IL-10) in the current surface.
type IntelligencePort = core.IntelligencePort

// DraftRequest is the input to IntelligencePort.Draft: an assembled Context
// bounded by the step's budget, the expected output Schema, an optional Persona,
// and optional few-shot Examples.
type DraftRequest = core.DraftRequest

// DraftResponse is the result of IntelligencePort.Draft. Its shape is completed
// at M04; it is not part of the G1 format freeze (sdk surface mutable until M08).
type DraftResponse = core.DraftResponse

// SynthesisRequest is the input to IntelligencePort.Synthesize: a Query, the
// source Entries, and a MaxLen output bound.
type SynthesisRequest = core.SynthesisRequest

// SynthesisResponse is the result of IntelligencePort.Synthesize. Its shape is
// completed at M04; it is not part of the G1 format freeze (sdk surface mutable
// until M08).
type SynthesisResponse = core.SynthesisResponse

// Classification is the result of IntelligencePort.Classify: the chosen
// Category, a Confidence in [0,1], and optional Reasoning.
type Classification = core.Classification

// Capability describes an intelligence capability a provider declares. Its shape
// is completed at M04; it is not part of the G1 format freeze (sdk surface
// mutable until M08).
type Capability = core.Capability

// Example is a few-shot example supplied to a draft request. Its shape is
// completed at M04; it is not part of the G1 format freeze (sdk surface mutable
// until M08).
type Example = core.Example

// Usage records provider-reported consumption (adapter name, model, tokens) for
// a single intelligence call. Its shape is completed at M04; it is not part of
// the G1 format freeze (sdk surface mutable until M08).
type Usage = core.Usage
