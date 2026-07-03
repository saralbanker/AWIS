package expr

import (
	"testing"
)

func mustParseTemplate(t *testing.T, src string) *Template {
	t.Helper()
	tmpl, err := ParseTemplate(src)
	if err != nil {
		t.Fatalf("ParseTemplate(%q) unexpected error: %v", src, err)
	}
	return tmpl
}

// TestTemplateResolveConcatenation covers T7/T8-style resolution with real Env
// values (literal+ref and multiple refs).
func TestTemplateResolveConcatenation(t *testing.T) {
	env := Env{
		Inputs: map[string]any{"name": "ada", "a": "A"},
		StepOutputs: map[string]map[string]any{
			"build": {"content": "ok"},
		},
		StepStatus: map[string]string{"build": "completed"},
	}

	// T7: prefix {{workflow.inputs.name}} suffix
	got, warns := mustParseTemplate(t, "prefix {{workflow.inputs.name}} suffix").Resolve(env)
	if got != "prefix ada suffix" {
		t.Errorf("T7 got %q, want %q", got, "prefix ada suffix")
	}
	if len(warns) != 0 {
		t.Errorf("T7 unexpected warnings: %v", warns)
	}

	// T8: {{workflow.inputs.a}}-{{steps.build.status}}
	got, warns = mustParseTemplate(t, "{{workflow.inputs.a}}-{{steps.build.status}}").Resolve(env)
	if got != "A-completed" {
		t.Errorf("T8 got %q, want %q", got, "A-completed")
	}
	if len(warns) != 0 {
		t.Errorf("T8 unexpected warnings: %v", warns)
	}
}

// TestTemplateResolveMissingPath covers FR-WD-05: a missing path resolves to ""
// and appends one Warning whose Path names the reference.
func TestTemplateResolveMissingPath(t *testing.T) {
	env := Env{Inputs: map[string]any{}}
	got, warns := mustParseTemplate(t, "x{{workflow.inputs.nope}}y").Resolve(env)
	if got != "xy" {
		t.Errorf("got %q, want %q", got, "xy")
	}
	if len(warns) != 1 {
		t.Fatalf("want 1 warning, got %d: %v", len(warns), warns)
	}
	if warns[0].Path != "workflow.inputs.nope" {
		t.Errorf("Warning.Path = %q, want %q", warns[0].Path, "workflow.inputs.nope")
	}
	if warns[0].Reason == "" {
		t.Error("Warning.Reason is empty")
	}
}

// TestTemplateResolveNotYetCompleted covers the FR-WD-05 completion gate: an
// outputs ref on a running step yields ""+Warning even though outputs exist.
func TestTemplateResolveNotYetCompleted(t *testing.T) {
	env := Env{
		StepOutputs: map[string]map[string]any{"build": {"content": "present"}},
		StepStatus:  map[string]string{"build": "running"},
	}
	got, warns := mustParseTemplate(t, "{{steps.build.outputs.content}}").Resolve(env)
	if got != "" {
		t.Errorf("got %q, want empty (step not completed)", got)
	}
	if len(warns) != 1 || warns[0].Path != "steps.build.outputs.content" {
		t.Fatalf("want 1 warning for steps.build.outputs.content, got %v", warns)
	}
}

// TestTemplateStatusReadNotGated covers that steps.<id>.status is NOT gated on
// completion: reading a running step's status returns "running".
func TestTemplateStatusReadNotGated(t *testing.T) {
	env := Env{StepStatus: map[string]string{"build": "running"}}
	got, warns := mustParseTemplate(t, "{{steps.build.status}}").Resolve(env)
	if got != "running" {
		t.Errorf("got %q, want %q", got, "running")
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
}

// TestTemplateStringification covers EDR-010 non-string stringification.
func TestTemplateStringification(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want string
	}{
		{"bool-true", true, "true"},
		{"bool-false", false, "false"},
		{"int", 42, "42"},
		{"int64", int64(-7), "-7"},
		{"float-int-valued", float64(3), "3"},
		{"float-frac", 3.14, "3.14"},
		{"float-round-trip", 0.1, "0.1"},
		{"string", "hi", "hi"},
		{"nested-map", map[string]any{"k": "v"}, "map[k:v]"},
		{"slice", []any{1, 2}, "[1 2]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := Env{Inputs: map[string]any{"x": c.val}}
			got, warns := mustParseTemplate(t, "{{workflow.inputs.x}}").Resolve(env)
			if got != c.want {
				t.Errorf("stringify(%v) via Resolve = %q, want %q", c.val, got, c.want)
			}
			if len(warns) != 0 {
				t.Errorf("unexpected warnings: %v", warns)
			}
		})
	}
}

// TestTemplateNestedMapWalk covers multi-segment key walking into nested maps.
func TestTemplateNestedMapWalk(t *testing.T) {
	env := Env{Inputs: map[string]any{"a": map[string]any{"b": map[string]any{"c": "deep"}}}}
	got, warns := mustParseTemplate(t, "{{workflow.inputs.a.b.c}}").Resolve(env)
	if got != "deep" {
		t.Errorf("got %q, want %q", got, "deep")
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
}

// TestTemplateEmpty covers T1: the empty template resolves to "".
func TestTemplateEmpty(t *testing.T) {
	got, warns := mustParseTemplate(t, "").Resolve(Env{})
	if got != "" || len(warns) != 0 {
		t.Errorf("empty template: got %q warns %v", got, warns)
	}
}

// TestTemplateWhitespaceTolerance covers the documented decision to tolerate
// surrounding ASCII spaces inside the braces (contradicts no corpus row).
func TestTemplateWhitespaceTolerance(t *testing.T) {
	env := Env{Inputs: map[string]any{"name": "ada"}}
	got, _ := mustParseTemplate(t, "{{  workflow.inputs.name  }}").Resolve(env)
	if got != "ada" {
		t.Errorf("whitespace-tolerant ref: got %q, want %q", got, "ada")
	}
}

// TestTemplateUnclosedPosition covers that an unmatched "{{" reports the offset
// of the "{{".
func TestTemplateUnclosedPosition(t *testing.T) {
	_, err := ParseTemplate("ab{{workflow.inputs.x")
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("want *ParseError, got %T (%v)", err, err)
	}
	if pe.Position != 2 {
		t.Errorf("unclosed {{ Position = %d, want 2", pe.Position)
	}
}

// TestTemplateStrayCloserIsLiteral covers that "}}" with no opener is literal.
func TestTemplateStrayCloserIsLiteral(t *testing.T) {
	got, warns := mustParseTemplate(t, "a}}b").Resolve(Env{})
	if got != "a}}b" || len(warns) != 0 {
		t.Errorf("stray }}: got %q warns %v", got, warns)
	}
}
