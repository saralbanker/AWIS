package intelligence

import (
	"fmt"

	"github.com/awis/awis/internal/core"
)

// Locality describes whether a provider runs locally (on-device / on-premises)
// or remotely (cloud API). It is registration metadata (EDR-009); it is not a
// frozen interface surface and may evolve before M08.
type Locality int

const (
	// LocalityLocal marks an on-device or on-premises provider (e.g. Ollama).
	LocalityLocal Locality = iota
	// LocalityCloud marks a remote cloud API provider.
	LocalityCloud
)

// Registration pairs an adapter with its routing metadata (EDR-009). Traits
// (Locality, CostRank, QualityRank) are internal configuration, not frozen
// surface.
type Registration struct {
	// Adapter is the IntelligencePort implementation.
	Adapter core.IntelligencePort
	// Locality indicates whether the provider is local or cloud.
	Locality Locality
	// CostRank is a relative cost ranking; lower = cheaper. Used by hint "fast".
	CostRank int
	// QualityRank is a relative quality ranking; lower = higher quality. Used by hint "quality".
	QualityRank int
}

// Decision is the result of a routing decision: the selected adapter and its
// associated registration metadata.
type Decision struct {
	// Adapter is the selected IntelligencePort.
	Adapter core.IntelligencePort
	// Registration is the metadata associated with the selected adapter.
	Registration Registration
}

// CapabilityUnavailableError is returned by Route when no eligible provider can
// serve a capability and required=true (Blueprint §17). Sentinel-checkable via
// errors.As.
type CapabilityUnavailableError struct {
	// Capability is the requested capability name.
	Capability string
}

func (e CapabilityUnavailableError) Error() string {
	return fmt.Sprintf("capability %q: no available provider", e.Capability)
}

// FallbackSignal is returned by Route when no eligible provider can serve a
// capability and required=false (Blueprint §17). It signals that the engine must
// activate the step's fallback path. The router only reports this decision;
// activation is M06 engine behaviour.
type FallbackSignal struct {
	// Capability is the requested capability name.
	Capability string
}

func (e FallbackSignal) Error() string {
	return fmt.Sprintf("capability %q: no available provider; activate step fallback", e.Capability)
}

// CapabilityRouter routes intelligence requests to registered adapters per the
// Blueprint §17 decision tree.
//
// Selection universe: only adapters named in the fallbackChain are eligible for
// selection (Blueprint §14: "adapters registered + chain declared together").
// Adapters that are registered but absent from the chain are never routed to.
// This is intentional: the chain is the router's complete operating universe.
type CapabilityRouter struct {
	regs  map[string]Registration // keyed by ProviderName()
	chain []string                // ordered fallback chain of ProviderName()s
}

// NewRouter constructs a CapabilityRouter from the given registrations and
// fallback chain. Every fallbackChain entry must name a registration; an error
// is returned otherwise.
func NewRouter(regs []Registration, fallbackChain []string) (*CapabilityRouter, error) {
	byName := make(map[string]Registration, len(regs))
	for _, r := range regs {
		byName[r.Adapter.ProviderName()] = r
	}
	for _, name := range fallbackChain {
		if _, ok := byName[name]; !ok {
			return nil, fmt.Errorf("router: fallback chain entry %q names no registered adapter", name)
		}
	}
	return &CapabilityRouter{regs: byName, chain: fallbackChain}, nil
}

// adapterHasCapability reports whether the adapter in reg declares capability.
func adapterHasCapability(reg Registration, capability string) bool {
	for _, c := range reg.Adapter.Capabilities() {
		if c.Name == capability {
			return true
		}
	}
	return false
}

// Eligible returns the preference-ordered list of available, capable adapters
// from the fallback chain. IsAvailable() is evaluated at call time.
//
// The selection universe is always the fallback chain (Blueprint §14). Adapters
// absent from the chain are never returned even if registered.
//
// Hint semantics (unknown hint treated identically to ""):
//   - "local":   local chain members first (chain order among locals), then cloud
//     chain members (chain order).
//   - "fast":    cheapest cloud first (min CostRank; tie → chain order); if no
//     cloud eligible, all eligible in chain order.
//   - "quality": best cloud first (min QualityRank; tie → chain order); if no
//     cloud eligible, all eligible in chain order.
//   - "" (or any unrecognised value): strict chain order.
func (r *CapabilityRouter) Eligible(capability, hint string) []Decision {
	// Build eligible set in chain order: capable + available + in chain.
	var eligible []Decision
	for _, name := range r.chain {
		reg, ok := r.regs[name]
		if !ok {
			continue
		}
		if !adapterHasCapability(reg, capability) {
			continue
		}
		if !reg.Adapter.IsAvailable() {
			continue
		}
		eligible = append(eligible, Decision{Adapter: reg.Adapter, Registration: reg})
	}
	if len(eligible) == 0 {
		return nil
	}

	switch hint {
	case "local":
		// Local adapters first (chain order), then cloud adapters (chain order).
		var locals, clouds []Decision
		for _, d := range eligible {
			if d.Registration.Locality == LocalityLocal {
				locals = append(locals, d)
			} else {
				clouds = append(clouds, d)
			}
		}
		return append(locals, clouds...)

	case "fast":
		// Cheapest cloud first (ascending CostRank, tie → chain order).
		// No cloud eligible → fall back to chain order over all eligible.
		var cloud []Decision
		for _, d := range eligible {
			if d.Registration.Locality == LocalityCloud {
				cloud = append(cloud, d)
			}
		}
		if len(cloud) == 0 {
			return eligible
		}
		return stableSortByCost(cloud)

	case "quality":
		// Highest quality cloud first (ascending QualityRank, tie → chain order).
		// No cloud eligible → fall back to chain order over all eligible.
		var cloud []Decision
		for _, d := range eligible {
			if d.Registration.Locality == LocalityCloud {
				cloud = append(cloud, d)
			}
		}
		if len(cloud) == 0 {
			return eligible
		}
		return stableSortByQuality(cloud)

	default:
		// "" or any unrecognised hint → strict chain order (already the input order).
		return eligible
	}
}

// Route selects the single highest-preference eligible adapter per Eligible, or
// returns a typed error (CapabilityUnavailableError or FallbackSignal) if no
// eligible adapter exists.
func (r *CapabilityRouter) Route(capability, hint string, required bool) (Decision, error) {
	list := r.Eligible(capability, hint)
	if len(list) == 0 {
		if required {
			return Decision{}, CapabilityUnavailableError{Capability: capability}
		}
		return Decision{}, FallbackSignal{Capability: capability}
	}
	return list[0], nil
}

// stableSortByCost returns a copy of ds sorted ascending by CostRank, using
// insertion sort to preserve chain order on ties (stable).
func stableSortByCost(ds []Decision) []Decision {
	out := make([]Decision, len(ds))
	copy(out, ds)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Registration.CostRank < out[j-1].Registration.CostRank; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// stableSortByQuality returns a copy of ds sorted ascending by QualityRank,
// preserving chain order on ties (stable insertion sort).
func stableSortByQuality(ds []Decision) []Decision {
	out := make([]Decision, len(ds))
	copy(out, ds)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Registration.QualityRank < out[j-1].Registration.QualityRank; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
