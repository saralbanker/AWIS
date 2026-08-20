package dsl

import (
	"strings"
	"testing"
)

// trimTrailingNewline strips at most one trailing newline for comparison.
func trimTrailingNewline(s string) string { return strings.TrimRight(s, "\n") }

// TestRenderValid_GoldenCaptureDecision is the golden output test for a valid
// definition (capture-decision.yaml; PRD §18 valid-output format).
func TestRenderValid_GoldenCaptureDecision(t *testing.T) {
	r, err := ValidateFile("testdata/capture-decision.yaml")
	if err != nil {
		t.Fatalf("ValidateFile: %v", err)
	}
	if !r.Valid() {
		t.Fatalf("expected valid, got issues: %v", r.Issues)
	}
	const want = `✓ capture-decision v1.0.0 is valid
  Steps: 5 (intelligence: 1, native: 1, plugin: 1, signal: 2)
  Intelligence: draft capability (required: false, fallback: manual-entry)
  Plugins: git-context-plugin
  Triggers: manual, event (git.push.completed)`
	got := trimTrailingNewline(r.Render())
	if got != want {
		t.Errorf("render mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

// TestRenderInvalid_GoldenBadFallback is the golden output test for the PRD §18
// example class (unknown fallback reference) — verifies Line N, Suggestion, Example.
func TestRenderInvalid_GoldenBadFallback(t *testing.T) {
	r, err := ValidateFile("testdata/render-bad-fallback.yaml")
	if err != nil {
		t.Fatalf("ValidateFile: %v", err)
	}
	if r.Valid() {
		t.Fatal("expected invalid, got valid")
	}
	const want = `✗ Validation failed: testdata/render-bad-fallback.yaml

  Line 10: fallback "no-such-step" is not a defined step

  Suggestion: Add a step with id "no-such-step", or remove the fallback reference.
  Example:
    - id: no-such-step
      name: <name>
      type: signal
      wait_signal:
        signal_name: <signal_name>
        timeout: 24h
        timeout_action: fail`
	got := trimTrailingNewline(r.Render())
	if got != want {
		t.Errorf("render mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

// TestRenderInvalid_GoldenBadInitialStep is the golden output test for an
// undefined initial_step — verifies Line N from the lineMap TopLevel map.
func TestRenderInvalid_GoldenBadInitialStep(t *testing.T) {
	r, err := ValidateFile("testdata/render-bad-initial.yaml")
	if err != nil {
		t.Fatalf("ValidateFile: %v", err)
	}
	if r.Valid() {
		t.Fatal("expected invalid, got valid")
	}
	const want = `✗ Validation failed: testdata/render-bad-initial.yaml

  Line 14: initial_step "no-such-step" is not a defined step

  Suggestion: Set initial_step to a step id that is defined in this workflow.
  Example:
    initial_step: <defined-step-id>`
	got := trimTrailingNewline(r.Render())
	if got != want {
		t.Errorf("render mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

// TestRenderInvalid_GoldenOrphanedStep is the golden output test for an
// orphaned (unreachable) step — verifies Line N from the lineMap Steps map.
func TestRenderInvalid_GoldenOrphanedStep(t *testing.T) {
	r, err := ValidateFile("testdata/render-bad-orphan.yaml")
	if err != nil {
		t.Fatalf("ValidateFile: %v", err)
	}
	if r.Valid() {
		t.Fatal("expected invalid, got valid")
	}
	const want = `✗ Validation failed: testdata/render-bad-orphan.yaml

  Line 14: step "orphan" is not reachable from initial_step

  Suggestion: Remove step "orphan" or add a transition leading to it from another step.
  Example:
    - from: <some-step>
      to: orphan`
	got := trimTrailingNewline(r.Render())
	if got != want {
		t.Errorf("render mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

// TestRenderInvalid_ParseErrorRenderable verifies that a parse-error Report
// produces renderable output even when Def is nil.
func TestRenderInvalid_ParseErrorRenderable(t *testing.T) {
	_, err := Parse(strings.NewReader("schema_version: 1\nid: [\n"), "bad.yaml")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	// Produce a parse-error Report directly (mirrors ValidateFile's parse path).
	r := &Report{
		File: "bad.yaml",
		Issues: []ReportIssue{{
			Code:    "parse-error",
			Message: err.Error(),
		}},
	}
	got := r.Render()
	if !strings.HasPrefix(got, "✗ Validation failed: bad.yaml") {
		t.Errorf("unexpected render start: %q", got[:min(len(got), 60)])
	}
	if !strings.Contains(got, "Suggestion:") {
		t.Errorf("expected Suggestion in parse-error render, got:\n%s", got)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
