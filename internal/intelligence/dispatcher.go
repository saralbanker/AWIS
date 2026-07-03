// Package intelligence implements the intelligence seam: the routing, budgeting,
// and dispatching layer that sits between the workflow engine and intelligence
// adapters (Blueprint §13–§14, §17).
//
// The seam has two deliberate absences in V1:
//   - Classify dispatch is absent. FR-IL-10 declares Classify a non-callable
//     placeholder; adapters implement the method for interface completeness but no
//     runtime path invokes it. Classify dispatch is wired at a future milestone.
//   - Embed dispatch is absent. CONTRA-3 declares Embed unavailable in V1. The
//     NullAdapter returns the zero vector (Blueprint §14) but no Dispatcher method
//     routes to it.
//
// Token budget enforcement uses the conservative ceil(len(s)/4) estimator defined
// in budget.go (EDR-008). Provider-actual token counts (FR-IL-09) arrive at M16
// and are never used for budget enforcement here.
//
// Adapter registration traits — Locality, CostRank, QualityRank — are defined in
// router.go per EDR-009.
package intelligence

import (
	"context"
	"errors"

	"github.com/awis/awis/internal/core"
)

// Dispatcher executes intelligence capability dispatch over the ordered fallback
// chain managed by a CapabilityRouter (Blueprint §17). Budget enforcement is
// applied before any routing or adapter call so that over-budget contexts never
// touch a provider.
type Dispatcher struct {
	router *CapabilityRouter
}

// NewDispatcher returns a Dispatcher backed by the given router.
func NewDispatcher(router *CapabilityRouter) *Dispatcher {
	return &Dispatcher{router: router}
}

// Draft dispatches a Draft request to the first eligible adapter in the fallback
// chain (Blueprint §17).
//
// Budget is enforced on dr.Context before any routing or provider call (FR-IL-08,
// EDR-008). An over-budget context returns ContextBudgetExceededError immediately
// without consulting the router or any adapter.
//
// If no adapter is eligible and required is true, Draft returns
// CapabilityUnavailableError. If required is false it returns FallbackSignal,
// signalling the engine to activate the step's fallback path (activation is M06
// engine behaviour).
//
// If all eligible adapters fail (Blueprint §17: "Provider call fails → try next in
// fallback_chain"), the typed error and the last provider error are joined via
// errors.Join so both are inspectable via errors.As/errors.Is.
func (d *Dispatcher) Draft(ctx context.Context, req core.IntelReq, required bool, dr core.DraftRequest) (core.DraftResponse, error) {
	if err := enforceBudget(dr.Context, req.ContextBudget); err != nil {
		return core.DraftResponse{}, err
	}
	list := d.router.Eligible(req.Capability, req.ModelHint)
	if len(list) == 0 {
		return core.DraftResponse{}, d.noEligibleError(req.Capability, required)
	}
	var lastErr error
	for _, dec := range list {
		resp, err := dec.Adapter.Draft(ctx, dr)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return core.DraftResponse{}, errors.Join(d.noEligibleError(req.Capability, required), lastErr)
}

// Synthesize dispatches a Synthesize request to the first eligible adapter in the
// fallback chain (Blueprint §17).
//
// Budget is enforced on sr.Query before any routing or provider call (FR-IL-08,
// EDR-008). Entries are not budget-counted in V1 — they are structured data, not
// assembled context; assembled-context budgeting is Draft's concern (Blueprint §13).
//
// If no adapter is eligible and required is true, Synthesize returns
// CapabilityUnavailableError. If required is false it returns FallbackSignal.
//
// If all eligible adapters fail, the typed error and the last provider error are
// joined via errors.Join so both are inspectable via errors.As/errors.Is.
func (d *Dispatcher) Synthesize(ctx context.Context, req core.IntelReq, required bool, sr core.SynthesisRequest) (core.SynthesisResponse, error) {
	if err := enforceBudget(sr.Query, req.ContextBudget); err != nil {
		return core.SynthesisResponse{}, err
	}
	list := d.router.Eligible(req.Capability, req.ModelHint)
	if len(list) == 0 {
		return core.SynthesisResponse{}, d.noEligibleError(req.Capability, required)
	}
	var lastErr error
	for _, dec := range list {
		resp, err := dec.Adapter.Synthesize(ctx, sr)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return core.SynthesisResponse{}, errors.Join(d.noEligibleError(req.Capability, required), lastErr)
}

// noEligibleError returns the correct typed error for the no-eligible-adapter
// case: CapabilityUnavailableError when required, FallbackSignal otherwise.
func (d *Dispatcher) noEligibleError(capability string, required bool) error {
	if required {
		return CapabilityUnavailableError{Capability: capability}
	}
	return FallbackSignal{Capability: capability}
}
