package dsl_test

// fixtures_test.go — all five pre-written YAML fixtures parse and validate with
// zero Issues (M10-C3; T9).  Handler existence is NOT checked (FR-WD-15; runtime-free path).

import (
	"path/filepath"
	"testing"

	"github.com/awis/awis/internal/dsl"
)

// fixtureCase pairs a human label with the path to a YAML fixture (relative to
// the module root, i.e. the directory that contains go.mod).
type fixtureCase struct {
	label string
	path  string
}

// fixtureRoot is the module root (two levels above the internal/dsl package).
const fixtureRoot = "../.."

// allFixtures returns the five pre-written workflow YAML files described in
// IMP §27.M10 Val + IMP §5 L104-105 (T9).
func allFixtures() []fixtureCase {
	root := fixtureRoot
	return []fixtureCase{
		{
			label: "examples/workflows/hello-world.yaml",
			path:  filepath.Join(root, "examples", "workflows", "hello-world.yaml"),
		},
		{
			label: "examples/workflows/with-signal.yaml",
			path:  filepath.Join(root, "examples", "workflows", "with-signal.yaml"),
		},
		{
			label: "examples/workflows/with-intelligence.yaml",
			path:  filepath.Join(root, "examples", "workflows", "with-intelligence.yaml"),
		},
		{
			label: "apps/oip/workflows/capture-decision.yaml",
			path:  filepath.Join(root, "apps", "oip", "workflows", "capture-decision.yaml"),
		},
		{
			label: "apps/oip/workflows/recall-decision.yaml",
			path:  filepath.Join(root, "apps", "oip", "workflows", "recall-decision.yaml"),
		},
	}
}

// TestFixturesParse asserts that every pre-written fixture YAML file can be
// parsed by dsl.ParseFile with no error (T9).
func TestFixturesParse(t *testing.T) {
	for _, tc := range allFixtures() {
		tc := tc
		t.Run(tc.label, func(t *testing.T) {
			_, err := dsl.ParseFile(tc.path)
			if err != nil {
				t.Errorf("ParseFile(%q): %v", tc.label, err)
			}
		})
	}
}

// TestFixturesValidate asserts that every pre-written fixture YAML file passes
// dsl.ValidateFile with zero Issues (T9; handler existence deferred per FR-WD-15).
func TestFixturesValidate(t *testing.T) {
	for _, tc := range allFixtures() {
		tc := tc
		t.Run(tc.label, func(t *testing.T) {
			report, err := dsl.ValidateFile(tc.path)
			if err != nil {
				t.Fatalf("ValidateFile(%q): I/O error: %v", tc.label, err)
			}
			if !report.Valid() {
				for _, iss := range report.Issues {
					t.Errorf("issue line=%d code=%s step=%s field=%s msg=%s",
						iss.Line, iss.Code, iss.StepID, iss.Field, iss.Message)
				}
				t.Fatalf("ValidateFile(%q): %d issue(s)", tc.label, len(report.Issues))
			}
		})
	}
}
