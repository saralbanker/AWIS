// Package sdk: shared helper for splitting a WorkflowInstance's accumulated
// Variables map into the caller-visible Inputs/Outputs pair used by Status,
// QueryHistory, and ReplayInstance (defect B-11b).
package sdk

// splitVariables splits a core.WorkflowInstance's Variables map into the
// workflow's original Inputs and its accumulated step Outputs.
//
// Variables has the shape {"inputs": <workflow inputs>, "<step-id>": <that
// step's outputs>, ...} (internal/core/instance.go doc comment). splitVariables
// returns:
//   - inputs: the value stored under the "inputs" key, coerced to
//     map[string]any. If the key is absent, or its value is not a
//     map[string]any, inputs is an empty (non-nil) map — never a panic.
//   - outputs: every other key of vars (i.e. vars minus "inputs").
//
// vars is never mutated; two new maps are always returned, so the caller's
// WorkflowInstance.Variables stays untouched.
func splitVariables(vars map[string]any) (inputs, outputs map[string]any) {
	inputs = make(map[string]any)
	outputs = make(map[string]any)
	for k, v := range vars {
		if k == "inputs" {
			continue
		}
		outputs[k] = v
	}
	if raw, ok := vars["inputs"]; ok {
		if m, ok := raw.(map[string]any); ok {
			for k, v := range m {
				inputs[k] = v
			}
		}
	}
	return inputs, outputs
}
