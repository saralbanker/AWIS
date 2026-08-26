// Unit tests for splitVariables (B-11b: Inputs/Outputs must not be the same
// map). See sdk/variables.go.
package sdk

import "testing"

// TestSplitVariables_NilMap verifies splitVariables never panics on a nil
// Variables map and returns non-nil empty maps.
func TestSplitVariables_NilMap(t *testing.T) {
	inputs, outputs := splitVariables(nil)
	if inputs == nil {
		t.Error("inputs is nil, want non-nil empty map")
	}
	if outputs == nil {
		t.Error("outputs is nil, want non-nil empty map")
	}
	if len(inputs) != 0 {
		t.Errorf("inputs = %#v, want empty", inputs)
	}
	if len(outputs) != 0 {
		t.Errorf("outputs = %#v, want empty", outputs)
	}
}

// TestSplitVariables_MissingInputsKey verifies that when "inputs" is absent,
// inputs is empty and outputs contains every key.
func TestSplitVariables_MissingInputsKey(t *testing.T) {
	vars := map[string]any{"stepA": map[string]any{"x": 1}}
	inputs, outputs := splitVariables(vars)
	if len(inputs) != 0 {
		t.Errorf("inputs = %#v, want empty", inputs)
	}
	if len(outputs) != 1 || outputs["stepA"] == nil {
		t.Errorf("outputs = %#v, want {stepA: ...}", outputs)
	}
}

// TestSplitVariables_NonMapInputsValue verifies nil-safety: a non-map value
// under "inputs" yields an empty inputs map, never a panic, and is still
// excluded from outputs.
func TestSplitVariables_NonMapInputsValue(t *testing.T) {
	vars := map[string]any{"inputs": "not-a-map", "stepA": map[string]any{"y": 2}}
	inputs, outputs := splitVariables(vars)
	if len(inputs) != 0 {
		t.Errorf("inputs = %#v, want empty", inputs)
	}
	if _, ok := outputs["inputs"]; ok {
		t.Error("outputs must not contain the \"inputs\" key")
	}
	if len(outputs) != 1 {
		t.Errorf("outputs = %#v, want exactly {stepA: ...}", outputs)
	}
}

// TestSplitVariables_SplitsCorrectlyAndDoesNotMutate verifies the normal case
// (workflow inputs plus one step's outputs) and that the source map is left
// untouched.
func TestSplitVariables_SplitsCorrectlyAndDoesNotMutate(t *testing.T) {
	vars := map[string]any{
		"inputs": map[string]any{"name": "Ada"},
		"greet":  map[string]any{"greeting": "hello Ada"},
	}
	inputs, outputs := splitVariables(vars)

	if len(inputs) != 1 || inputs["name"] != "Ada" {
		t.Errorf("inputs = %#v, want {name: Ada}", inputs)
	}
	if _, ok := outputs["inputs"]; ok {
		t.Error("outputs must not contain the \"inputs\" key")
	}
	greetOut, ok := outputs["greet"].(map[string]any)
	if !ok || greetOut["greeting"] != "hello Ada" {
		t.Errorf("outputs[\"greet\"] = %#v, want {greeting: hello Ada}", outputs["greet"])
	}

	// Source map must be unchanged (3 keys expected: inputs, greet, and
	// original length check below covers no in-place deletion/addition).
	if len(vars) != 2 {
		t.Errorf("splitVariables mutated the source map: len(vars) = %d, want 2", len(vars))
	}
	if _, ok := vars["inputs"]; !ok {
		t.Error("splitVariables removed \"inputs\" from the source map")
	}
}
