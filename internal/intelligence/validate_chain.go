package intelligence

import "fmt"

// nullAdapterName is the stable ProviderName() of the NullAdapter (null.go). The
// null adapter is the degraded-mode last resort; the production assembly rule
// (M04 HANDOFF; TDS §17) requires it to be LAST in the fallback chain so it never
// shadows a real provider. ValidateChain identifies it by this name.
const nullAdapterName = "null"

// ValidateChain is the construction-time check for a fallback chain (M04-V1
// advisory; sanctioned +1 exported function in this package, IMPLEMENTATION_SPEC
// T2). It errors unless:
//
//   - every chain entry names a supplied Registration (same universe rule as
//     NewRouter — a chain entry with no adapter can never be routed to), AND
//   - the null adapter, when present in the chain, is the LAST entry (the M04
//     runtime-assembly obligation: NullAdapter is the degraded last resort and
//     must not precede a real provider).
//
// It does not mutate its inputs and performs no I/O.
func ValidateChain(regs []Registration, chain []string) error {
	byName := make(map[string]bool, len(regs))
	for _, r := range regs {
		byName[r.Adapter.ProviderName()] = true
	}
	for _, name := range chain {
		if !byName[name] {
			return fmt.Errorf("intelligence: fallback chain entry %q names no registered adapter", name)
		}
	}
	for i, name := range chain {
		if name == nullAdapterName && i != len(chain)-1 {
			return fmt.Errorf(
				"intelligence: null adapter must be last in the fallback chain, found at position %d of %d",
				i, len(chain))
		}
	}
	return nil
}
