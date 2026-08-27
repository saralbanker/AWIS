//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestLifecycle_JSONContract exercises every read-only, always-available
// subcommand and asserts the JSON contract: exit 0 and non-empty, valid JSON
// under --json (version, status, workflow list/validate, history, metrics,
// audit, export, config show, plugin list).
func TestLifecycle_JSONContract(t *testing.T) {
	f := newProject(t)
	wfPath := f.writeWorkflow("linear.yaml", linearNative)
	f.start()

	cases := [][]string{
		{"version"},
		{"status"},
		{"workflow", "list"},
		{"workflow", "validate", wfPath},
		{"history"},
		{"metrics"},
		{"audit"},
		{"export"},
		{"config", "show"},
		{"plugin", "list"},
	}
	for _, args := range cases {
		args := args
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			res := f.run(append([]string{"--json"}, args...)...)
			if res.exitCode != 0 {
				t.Fatalf("%v exited %d (want 0): stdout=%s stderr=%s", args, res.exitCode, res.stdout, res.stderr)
			}
			if strings.TrimSpace(res.stdout) == "" {
				t.Fatalf("%v: empty stdout", args)
			}
			if !json.Valid([]byte(res.stdout)) {
				t.Fatalf("%v: invalid JSON: %s", args, res.stdout)
			}
		})
	}
}
