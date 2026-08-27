//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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

// TestB28_WaitingInstanceReportsWhichSignalItAwaits is B-28 through the
// shipped binary.
//
// signal_name and timeout_remaining_s are declared in the status JSON schema
// (TDS-07 §4) but were never assigned, so they were permanently null. An
// operator looking at a `waiting` instance had no way to learn which signal
// to deliver without opening the workflow YAML — and a UI could not render
// the wait at all. The answer was durable in wait_records the whole time.
func TestB28_WaitingInstanceReportsWhichSignalItAwaits(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("waiter.yaml", signalOnly)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "waiter")
	f.waitForStep(sub.InstanceID, "waiting", "wait-a", 10*time.Second)

	var st statusWaitJSON
	f.runJSON(&st, "status", sub.InstanceID)

	if st.SignalName == nil {
		t.Fatalf("signal_name is null for an instance parked on a signal; status=%+v", st)
	}
	if *st.SignalName != "go" {
		t.Errorf("signal_name = %q, want %q", *st.SignalName, "go")
	}
	if st.TimeoutRemainingS == nil {
		t.Error("timeout_remaining_s is null although the fixture declares timeout: 72h")
	} else if *st.TimeoutRemainingS <= 0 {
		t.Errorf("timeout_remaining_s = %d, want a positive countdown", *st.TimeoutRemainingS)
	}

	// The reported name must be the one that actually advances the instance:
	// a signal_name that does not work is worse than a null one.
	if res := f.run("signal", sub.InstanceID, *st.SignalName); res.exitCode != 0 {
		t.Fatalf("signalling the reported signal_name %q failed: exit=%d stderr=%s",
			*st.SignalName, res.exitCode, res.stderr)
	}
	f.waitForStep(sub.InstanceID, "waiting", "wait-b", 10*time.Second)

	// And it must now report the NEXT wait, not the one just satisfied.
	f.runJSON(&st, "status", sub.InstanceID)
	if st.SignalName == nil || *st.SignalName != "done" {
		t.Errorf("after advancing, signal_name = %v, want %q", st.SignalName, "done")
	}
}

// statusWaitJSON mirrors the wait-describing fields of 'awis --json status <id>'.
type statusWaitJSON struct {
	InstanceID        string  `json:"instance_id"`
	Status            string  `json:"status"`
	CurrentStep       *string `json:"current_step"`
	SignalName        *string `json:"signal_name"`
	TimeoutRemainingS *int    `json:"timeout_remaining_s"`
}
