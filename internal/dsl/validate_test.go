package dsl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/awis/awis/internal/validate"
)

// vf writes yaml to a temp file, calls ValidateFile, and returns the Report.
func vf(t *testing.T, yaml string) *Report {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "dsl-*.yaml")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := f.WriteString(yaml); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	r, err := ValidateFile(f.Name())
	if err != nil {
		t.Fatalf("ValidateFile: %v", err)
	}
	return r
}

// hasCode reports whether any Issue in r carries the given code.
func hasCode(r *Report, code string) bool {
	for _, i := range r.Issues {
		if i.Code == code {
			return true
		}
	}
	return false
}

// twoStepBase is a complete minimal YAML with two native steps a and b.
const twoStepBase = `schema_version: 1
id: test
version: 1.0.0
namespace: test
name: Test
metadata: {}
triggers:
  - type: manual
steps:
  - id: a
    name: A
    type: native
    handler: h
  - id: b
    name: B
    type: native
    handler: h
`

// oneStepBase is a complete minimal YAML with one native step a.
const oneStepBase = `schema_version: 1
id: test
version: 1.0.0
namespace: test
name: Test
metadata: {}
triggers:
  - type: manual
steps:
  - id: a
    name: A
    type: native
    handler: h
`

// TestValidateFile_OrphanedStep verifies that an unreachable step produces
// CodeOrphanedStep (PRD §18 "No orphaned steps").
func TestValidateFile_OrphanedStep(t *testing.T) {
	r := vf(t, twoStepBase+`initial_step: a
transitions: []
final_steps: [a]
`)
	if !hasCode(r, validate.CodeOrphanedStep) {
		t.Errorf("want CodeOrphanedStep, got %v", r.Issues)
	}
	if r.Issues[0].Line == 0 {
		t.Errorf("want non-zero line for orphaned step")
	}
}

// TestValidateFile_Cycle verifies that a transition cycle produces CodeCycle
// (PRD §18 "No cycles").
func TestValidateFile_Cycle(t *testing.T) {
	r := vf(t, twoStepBase+`initial_step: a
transitions:
  - from: a
    to: b
  - from: b
    to: a
final_steps: [a]
`)
	if !hasCode(r, validate.CodeCycle) {
		t.Errorf("want CodeCycle, got %v", r.Issues)
	}
}

// TestValidateFile_BadTransitionRef verifies that a transition whose to-field
// references an undefined step produces CodeTransitionToUndefined
// (PRD §18 "transition.from/to reference defined step IDs").
func TestValidateFile_BadTransitionRef(t *testing.T) {
	r := vf(t, oneStepBase+`initial_step: a
transitions:
  - from: a
    to: ghost
final_steps: [a]
`)
	if !hasCode(r, validate.CodeTransitionToUndefined) {
		t.Errorf("want CodeTransitionToUndefined, got %v", r.Issues)
	}
	if r.Issues[0].Line == 0 {
		t.Errorf("want non-zero line for bad transition")
	}
}

// TestValidateFile_BadConditionGrammar verifies that a condition that does not
// parse yields CodeConditionSyntax (PRD §18 "condition expressions parse per
// formal grammar").
func TestValidateFile_BadConditionGrammar(t *testing.T) {
	r := vf(t, twoStepBase+`initial_step: a
transitions:
  - from: a
    to: b
    condition: "!!!"
final_steps: [b]
`)
	if !hasCode(r, validate.CodeConditionSyntax) {
		t.Errorf("want CodeConditionSyntax, got %v", r.Issues)
	}
	if r.Issues[0].Line == 0 {
		t.Errorf("want non-zero line for bad condition")
	}
}

// TestValidateFile_ConditionEventScope verifies that an event-scoped path in a
// transition condition produces CodeConditionEventScope
// (PRD §18; event scope only valid in trigger filters).
func TestValidateFile_ConditionEventScope(t *testing.T) {
	r := vf(t, twoStepBase+`initial_step: a
transitions:
  - from: a
    to: b
    condition: "event.branch == 'main'"
final_steps: [b]
`)
	if !hasCode(r, validate.CodeConditionEventScope) {
		t.Errorf("want CodeConditionEventScope, got %v", r.Issues)
	}
}

// TestValidateFile_BadTriggerFilter verifies that an unparseable trigger filter
// produces CodeFilterSyntax (PRD §18 "trigger.config.filter expressions parse").
func TestValidateFile_BadTriggerFilter(t *testing.T) {
	const yaml = `schema_version: 1
id: test
version: 1.0.0
namespace: test
name: Test
metadata: {}
triggers:
  - type: event
    config:
      filter: "!!!"
steps:
  - id: a
    name: A
    type: native
    handler: h
initial_step: a
transitions: []
final_steps: [a]
`
	r := vf(t, yaml)
	if !hasCode(r, validate.CodeFilterSyntax) {
		t.Errorf("want CodeFilterSyntax, got %v", r.Issues)
	}
	if r.Issues[0].Line == 0 {
		t.Errorf("want non-zero line for bad trigger filter")
	}
}

// TestValidateFile_BadInitialStep verifies that initial_step referencing an
// undefined step produces CodeInitialStepUndefined
// (PRD §18 "initial_step references a defined step").
func TestValidateFile_BadInitialStep(t *testing.T) {
	r := vf(t, oneStepBase+`initial_step: ghost
transitions: []
final_steps: [a]
`)
	if !hasCode(r, validate.CodeInitialStepUndefined) {
		t.Errorf("want CodeInitialStepUndefined, got %v", r.Issues)
	}
	if r.Issues[0].Line == 0 {
		t.Errorf("want non-zero line for bad initial_step")
	}
}

// TestValidateFile_BadFinalStep verifies that final_steps referencing an
// undefined step produces CodeFinalStepUndefined
// (PRD §18 "final_steps references defined steps").
func TestValidateFile_BadFinalStep(t *testing.T) {
	r := vf(t, oneStepBase+`initial_step: a
transitions: []
final_steps: [ghost]
`)
	if !hasCode(r, validate.CodeFinalStepUndefined) {
		t.Errorf("want CodeFinalStepUndefined, got %v", r.Issues)
	}
	if r.Issues[0].Line == 0 {
		t.Errorf("want non-zero line for bad final_steps")
	}
}

// TestValidateFile_BadContextBudget verifies that context_budget <= 0 on an
// intelligence step produces CodeIntelligenceContextBudget
// (PRD §18 "context_budget is a positive integer").
func TestValidateFile_BadContextBudget(t *testing.T) {
	const yaml = `schema_version: 1
id: test
version: 1.0.0
namespace: test
name: Test
metadata: {}
triggers:
  - type: manual
steps:
  - id: a
    name: A
    type: intelligence
    intelligence:
      capability: draft
      context_budget: 0
initial_step: a
transitions: []
final_steps: [a]
`
	r := vf(t, yaml)
	if !hasCode(r, validate.CodeIntelligenceContextBudget) {
		t.Errorf("want CodeIntelligenceContextBudget, got %v", r.Issues)
	}
}

// TestValidateFile_BadModelHint verifies that an invalid model_hint produces
// CodeIntelligenceModelHint (PRD §18 "model_hint is fast|quality|local|nil").
func TestValidateFile_BadModelHint(t *testing.T) {
	const yaml = `schema_version: 1
id: test
version: 1.0.0
namespace: test
name: Test
metadata: {}
triggers:
  - type: manual
steps:
  - id: a
    name: A
    type: intelligence
    intelligence:
      capability: draft
      context_budget: 100
      model_hint: ultrafast
initial_step: a
transitions: []
final_steps: [a]
`
	r := vf(t, yaml)
	if !hasCode(r, validate.CodeIntelligenceModelHint) {
		t.Errorf("want CodeIntelligenceModelHint, got %v", r.Issues)
	}
}

// TestValidateFile_BadFallback verifies that step.fallback referencing an
// undefined step produces CodeFallbackUndefined
// (PRD §18 "fallback references point to steps defined in the same workflow").
func TestValidateFile_BadFallback(t *testing.T) {
	const yaml = `schema_version: 1
id: test
version: 1.0.0
namespace: test
name: Test
metadata: {}
triggers:
  - type: manual
steps:
  - id: a
    name: A
    type: native
    handler: h
    fallback: ghost
initial_step: a
transitions: []
final_steps: [a]
`
	r := vf(t, yaml)
	if !hasCode(r, validate.CodeFallbackUndefined) {
		t.Errorf("want CodeFallbackUndefined, got %v", r.Issues)
	}
	if r.Issues[0].Line == 0 {
		t.Errorf("want non-zero line for bad fallback")
	}
}

// TestValidateFile_ParseError verifies that a YAML parse failure produces a
// Report with a "parse-error" Issue (still renderable) instead of a Go error.
func TestValidateFile_ParseError(t *testing.T) {
	r := vf(t, "schema_version: 1\nid: [\n")
	if !hasCode(r, "parse-error") {
		t.Errorf("want parse-error code, got %v", r.Issues)
	}
	if r.Valid() {
		t.Error("want Valid()=false for parse error")
	}
}

// TestValidateFile_IOError verifies that an unreadable path returns a Go error.
func TestValidateFile_IOError(t *testing.T) {
	_, err := ValidateFile(filepath.Join(t.TempDir(), "no-such.yaml"))
	if err == nil {
		t.Error("want error for missing file, got nil")
	}
}

// TestDiscover_Sorted verifies that Discover returns *.yaml files in
// lexical order (T6; PRD §18 "auto-discovered by awis start").
func TestDiscover_Sorted(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	names := []string{"z.yaml", "a.yaml", "m.yaml"}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(wfDir, n), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 results, got %d: %v", len(got), got)
	}
	// Must be lexically sorted.
	for i := 0; i < len(got)-1; i++ {
		if got[i] >= got[i+1] {
			t.Errorf("not sorted: got[%d]=%q >= got[%d]=%q", i, got[i], i+1, got[i+1])
		}
	}
	// All must be under dir/workflows/.
	for _, p := range got {
		if !strings.HasPrefix(p, wfDir) {
			t.Errorf("unexpected path prefix: %q", p)
		}
	}
}

// TestDiscover_EmptyDir verifies that a workflows dir with no *.yaml files
// returns an empty slice without error.
func TestDiscover_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty slice, got %v", got)
	}
}

// TestDiscover_MissingDir verifies that a non-existent directory returns an
// empty slice and nil error (card spec: "missing directory ⇒ empty slice, nil error").
func TestDiscover_MissingDir(t *testing.T) {
	got, err := Discover(filepath.Join(t.TempDir(), "no-such-dir"))
	if err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty slice, got %v", got)
	}
}
