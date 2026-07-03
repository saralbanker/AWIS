package expr

// Env is the shared resolution environment for both expression languages. Its
// shape mirrors the M03 / EDR-007 projection variables — {"inputs": …,
// "<step_id>": outputs} plus step statuses and (conditions only) the trigger
// event payload. Nested map[string]any values are walked segment-by-segment for
// multi-segment keys; any absence anywhere along the path yields (nil, false),
// i.e. null/missing.
type Env struct {
	// Inputs backs workflow.inputs.<key…> lookups (the workflow's original
	// submission inputs).
	Inputs map[string]any
	// StepOutputs backs steps.<id>.outputs.<key…> lookups (a step's output map,
	// keyed by step id).
	StepOutputs map[string]map[string]any
	// StepStatus backs steps.<id>.status reads; values are one of
	// pending | running | completed | failed | cancelled.
	StepStatus map[string]string
	// Event backs event.<key…> lookups (top-level DomainEvent.payload fields).
	// Only meaningful for condition expressions in trigger filter fields.
	Event map[string]any
}

// lookup resolves a parsed path (segs[0] is the scope) to a value. It returns
// (nil, false) for any absent path segment. It performs a pure read: template
// status-gating (FR-WD-05 "not yet completed") is applied by the caller in
// Template.Resolve, not here — a raw status read is never gated.
//
// Path shapes (per TDS-03 scope definitions / EDR-007):
//   - workflow.inputs.<key…>   → walk Inputs
//   - steps.<id>.outputs.<key…> → walk StepOutputs[id]
//   - steps.<id>.status         → StepStatus[id]
//   - event.<key…>              → walk Event
func (e Env) lookup(segs []string) (any, bool) {
	if len(segs) < 2 {
		return nil, false
	}
	switch segs[0] {
	case "workflow":
		// Only workflow.inputs.<key…> is a defined scope; anything else is missing.
		if len(segs) >= 3 && segs[1] == "inputs" {
			return walk(e.Inputs, segs[2:])
		}
		return nil, false
	case "steps":
		if len(segs) < 3 {
			return nil, false
		}
		id := segs[1]
		if len(segs) == 3 && segs[2] == "status" {
			s, ok := e.StepStatus[id]
			if !ok {
				return nil, false
			}
			return s, true
		}
		if len(segs) >= 4 && segs[2] == "outputs" {
			out, ok := e.StepOutputs[id]
			if !ok {
				return nil, false
			}
			return walk(out, segs[3:])
		}
		return nil, false
	case "event":
		return walk(e.Event, segs[1:])
	default:
		return nil, false
	}
}

// walk descends a nested map[string]any following keys. A nil map, a non-map
// intermediate value, or an absent key all yield (nil, false).
func walk(m map[string]any, keys []string) (any, bool) {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := mm[k]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}
