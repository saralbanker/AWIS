package examples

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/awis/awis/internal/core"
)

// repoRoot walks up from this test file until it finds go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up from " + file)
		}
		dir = parent
	}
}

// TestEveryScaffoldedHandlerIsBuiltIn is the guard for defect B-9: `awis start`
// can only dispatch native steps whose handler is compiled into the binary, so
// every `handler:` referenced by a workflow that `awis init` scaffolds MUST have
// a built-in implementation here. Without this invariant the documented
// out-of-the-box path fails at the first step with `handler_not_found`.
//
// The check reads the scaffold YAML directly rather than a copy, so adding a
// scaffolded workflow that references a new handler fails this test until the
// handler exists.
func TestEveryScaffoldedHandlerIsBuiltIn(t *testing.T) {
	root := repoRoot(t)

	registered := make(map[string]bool)
	for _, id := range IDs() {
		registered[id] = true
	}

	// Both directories must hold byte-identical workflows; check both so a drift
	// between the embedded scaffold and the examples/ copies is caught here too.
	dirs := []string{
		filepath.Join(root, "cmd", "awis", "scaffold", "workflows"),
		filepath.Join(root, "examples", "workflows"),
	}

	seen := 0
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			for _, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if !strings.HasPrefix(trimmed, "handler:") {
					continue
				}
				ref := strings.TrimSpace(strings.TrimPrefix(trimmed, "handler:"))
				if ref == "" {
					continue
				}
				seen++
				if !registered[ref] {
					t.Errorf("%s references handler %q, which has no built-in implementation in package examples; "+
						"`awis start` would fail that step with handler_not_found", path, ref)
				}
			}
		}
	}

	if seen == 0 {
		t.Fatal("no handler: references found in the scaffold workflows — the check is not actually running")
	}
}

// TestHandlersHaveUniqueIDs guards against a duplicate ID silently shadowing an
// earlier handler: NativeRunner.Register overwrites on collision.
func TestHandlersHaveUniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, h := range Handlers() {
		if seen[h.ID()] {
			t.Errorf("duplicate handler id %q", h.ID())
		}
		seen[h.ID()] = true
	}
}

// TestHandlersExecute exercises each built-in handler once so a nil-map or
// type-assertion regression is caught here rather than at runtime.
func TestHandlersExecute(t *testing.T) {
	cases := map[string]struct {
		inputs   map[string]any
		wantKeys []string
	}{
		"examples.hello.greet":    {inputs: map[string]any{"name": "AWIS"}, wantKeys: []string{"message"}},
		"examples.hello.log":      {inputs: map[string]any{"message": "hi"}, wantKeys: []string{"logged"}},
		"examples.signal.prepare": {inputs: nil, wantKeys: []string{"ready"}},
		"examples.signal.finalize": {
			inputs:   map[string]any{"approval": true},
			wantKeys: []string{"done", "approval"},
		},
		"examples.intel.gather": {inputs: map[string]any{"query": "q"}, wantKeys: []string{"context"}},
		"examples.intel.review": {inputs: nil, wantKeys: []string{"approved"}},

		"examples.diagnostic.succeed": {inputs: nil, wantKeys: []string{"recovered", "step_id", "attempt"}},
		// flaky succeeds from attempt succeed_on onward; attempt 1 with
		// succeed_on 1 is its success path. Its FAILING path is in
		// diagnostic_test.go.
		"examples.diagnostic.flaky": {
			inputs:   map[string]any{"succeed_on": 1},
			wantKeys: []string{"attempt", "succeed_on"},
		},
	}

	for _, h := range Handlers() {
		if diagnosticsCoveredElsewhere[h.ID()] {
			continue
		}
		tc, ok := cases[h.ID()]
		if !ok {
			t.Errorf("handler %q has no execution case in this test", h.ID())
			continue
		}
		out, err := h.Execute(core.StepContext{StepID: "s", Attempt: 1, Inputs: tc.inputs})
		if err != nil {
			t.Errorf("%s: Execute returned error: %v", h.ID(), err)
			continue
		}
		for _, k := range tc.wantKeys {
			if _, ok := out.Outputs[k]; !ok {
				t.Errorf("%s: outputs missing key %q (got %v)", h.ID(), k, out.Outputs)
			}
		}
	}
}

// TestHelloGreetDefaultsMissingName covers the nil-input path: a workflow
// submitted without --input must not panic.
func TestHelloGreetDefaultsMissingName(t *testing.T) {
	out, err := helloGreet(core.StepContext{Inputs: nil})
	if err != nil {
		t.Fatalf("helloGreet: %v", err)
	}
	if got := out.Outputs["message"]; got != "Hello, World!" {
		t.Fatalf("message = %v, want %q", got, "Hello, World!")
	}
}

// diagnosticsCoveredElsewhere lists built-in handlers whose contract is to
// FAIL, to panic, or to block — none of which a single "call it and expect
// outputs" case in TestHandlersExecute can express. They are covered by
// diagnostic_test.go instead. Listing them explicitly keeps TestHandlersExecute's
// "every handler has a case" guarantee intact: a NEW handler still fails that
// test until someone writes a case for it.
var diagnosticsCoveredElsewhere = map[string]bool{
	"examples.diagnostic.fail":  true,
	"examples.diagnostic.panic": true,
	"examples.diagnostic.slow":  true,
}
