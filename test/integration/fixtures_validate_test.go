//go:build integration

package integration

import "testing"

// TestFixturesValidate is a parse-and-validate smoke test for every fixture
// used elsewhere in this suite. If a fixture stops parsing, this is the one
// test that fails for that reason, instead of every dependent test failing
// for an opaque, unrelated one.
func TestFixturesValidate(t *testing.T) {
	fixtures := map[string]string{
		"linear-native.yaml":  linearNative,
		"waiter.yaml":         signalOnly,
		"retry-flaky.yaml":    retryFlaky,
		"with-fallback.yaml":  withFallback,
		"with-on-error.yaml":  withOnError,
		"terminal-fail.yaml":  terminalFail,
		"panic-step.yaml":     panicStep,
		"slow-step.yaml":      slowStep,
		"slow-then-done.yaml": slowThenDone,
	}
	for name, yaml := range fixtures {
		name, yaml := name, yaml
		t.Run(name, func(t *testing.T) {
			f := newProject(t)
			path := f.writeWorkflow(name, yaml)
			res := f.run("workflow", "validate", path)
			if res.exitCode != 0 {
				t.Fatalf("workflow validate %s exited %d (want 0): stdout=%s stderr=%s", name, res.exitCode, res.stdout, res.stderr)
			}
		})
	}
}
