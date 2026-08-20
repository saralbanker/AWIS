package dsl

import (
	"strings"
	"testing"

	"github.com/awis/awis/internal/validate"
)

// TestParse_CaptureDecision asserts that the Blueprint §7 capture-decision
// oracle parses and that ≥8 fields decode correctly (M10-C1 ACCEPTANCE).
func TestParse_CaptureDecision(t *testing.T) {
	def, err := ParseFile("testdata/capture-decision.yaml")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	if def.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", def.SchemaVersion)
	}
	if string(def.Version) != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", def.Version)
	}
	if def.Namespace != "oip" {
		t.Errorf("Namespace = %q, want oip", def.Namespace)
	}
	if def.InitialStep != "assemble-context" {
		t.Errorf("InitialStep = %q, want assemble-context", def.InitialStep)
	}
	// final_steps
	if len(def.FinalSteps) != 1 || def.FinalSteps[0] != "append-to-record" {
		t.Errorf("FinalSteps = %v, want [append-to-record]", def.FinalSteps)
	}
	// wait_signal.timeout_action (Steps[2] = manual-entry)
	if def.Steps[2].WaitSignal == nil {
		t.Fatal("Steps[2].WaitSignal is nil")
	}
	if def.Steps[2].WaitSignal.TimeoutAction != "fail" {
		t.Errorf("manual-entry WaitSignal.TimeoutAction = %q, want fail", def.Steps[2].WaitSignal.TimeoutAction)
	}
	// trigger filter (Triggers[1] = event trigger)
	if f, ok := def.Triggers[1].Config["filter"]; !ok || f != "event.branch == 'main'" {
		t.Errorf("trigger[1].config.filter = %v, want \"event.branch == 'main'\"", f)
	}
	// inputs template strings (Steps[0] = assemble-context)
	if v, ok := def.Steps[0].Inputs["repo_path"]; !ok || v != "{{workflow.inputs.repo_path}}" {
		t.Errorf("assemble-context inputs.repo_path = %v, want {{workflow.inputs.repo_path}}", v)
	}
	if len(def.Steps) != 5 {
		t.Errorf("len(Steps) = %d, want 5", len(def.Steps))
	}
	if len(def.Transitions) != 5 {
		t.Errorf("len(Transitions) = %d, want 5", len(def.Transitions))
	}
}

// TestParse_UnknownKey asserts that an unrecognised YAML field is rejected with
// an error that names the key and includes a line number.
func TestParse_UnknownKey(t *testing.T) {
	const src = `schema_version: 1
id: test
version: 1.0.0
namespace: test
name: Test
unknown_field: bad
metadata: {}
triggers: []
steps: []
transitions: []
initial_step: ""
final_steps: []
`
	_, err := Parse(strings.NewReader(src), "test.yaml")
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "unknown_field") {
		t.Errorf("error does not name the field: %v", err)
	}
	if !strings.Contains(err.Error(), "line") {
		t.Errorf("error does not include line number: %v", err)
	}
}

// TestParse_MalformedYAML asserts that a YAML syntax error is reported with
// the filename and a line number.
func TestParse_MalformedYAML(t *testing.T) {
	const src = "schema_version: 1\nid: [\n"
	_, err := Parse(strings.NewReader(src), "bad.yaml")
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
	if !strings.Contains(err.Error(), "bad.yaml") {
		t.Errorf("error does not contain filename: %v", err)
	}
	if !strings.Contains(err.Error(), "line") {
		t.Errorf("error does not contain line info: %v", err)
	}
}

// TestParse_BadSemver asserts that a non-semver version string is rejected with
// a precise error.
func TestParse_BadSemver(t *testing.T) {
	const src = `schema_version: 1
id: test
version: not-a-semver
namespace: test
name: Test
metadata: {}
triggers: []
steps: []
transitions: []
initial_step: ""
final_steps: []
`
	_, err := Parse(strings.NewReader(src), "test.yaml")
	if err == nil {
		t.Fatal("expected error for bad semver, got nil")
	}
	if !strings.Contains(err.Error(), "not-a-semver") {
		t.Errorf("error does not mention the bad version: %v", err)
	}
}

// TestParse_WrongSchemaVersion asserts that schema_version ≠ 1 is rejected.
func TestParse_WrongSchemaVersion(t *testing.T) {
	const src = `schema_version: 99
id: test
version: 1.0.0
namespace: test
name: Test
metadata: {}
triggers: []
steps: []
transitions: []
initial_step: ""
final_steps: []
`
	_, err := Parse(strings.NewReader(src), "test.yaml")
	if err == nil {
		t.Fatal("expected error for wrong schema_version, got nil")
	}
	if !strings.Contains(err.Error(), "schema_version") {
		t.Errorf("error does not mention schema_version: %v", err)
	}
}

// TestParse_CaptureDecisionValidates asserts that the parsed capture-decision
// definition passes internal/validate.Validate with zero Issues.
func TestParse_CaptureDecisionValidates(t *testing.T) {
	def, err := ParseFile("testdata/capture-decision.yaml")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	issues := validate.Validate(*def)
	if len(issues) != 0 {
		t.Errorf("validate.Validate returned %d issue(s): %v", len(issues), issues)
	}
}
