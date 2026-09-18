//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestB6_FlagsAfterPositionalAreHonoured: 'awis submit <id> --input k=v' —
// a flag AFTER the positional workflow-id, exactly as 'awis init' prints in
// its "Next steps" — must still record the input in WorkflowStarted.inputs.
// linear-native (not hello-world) is used so this assertion isolates B-6
// from the separate B-23 namespace defect (see TestB23).
func TestB6_FlagsAfterPositionalAreHonoured(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("linear.yaml", linearNative)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "linear-native", "--input", "name=World")

	var trace traceOutputJSON
	f.runJSON(&trace, "trace", "--full", sub.InstanceID)
	ev := findEvent(trace.Events, "WorkflowStarted")
	if ev == nil {
		t.Fatalf("no WorkflowStarted event in trace: %+v", trace.Events)
	}
	var payload struct {
		Inputs map[string]any `json:"inputs"`
	}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatalf("WorkflowStarted payload not valid JSON: %v (%s)", err, ev.Payload)
	}
	if got, _ := payload.Inputs["name"].(string); got != "World" {
		t.Fatalf("WorkflowStarted.inputs.name = %q, want %q (flag after positional was not honoured)", got, "World")
	}
}

// TestB7_TraceJSONValidForLargePayload: 'awis --json trace <id>' must
// produce non-empty, valid JSON even when an event payload exceeds the
// 120-byte truncation threshold (default, non---full trace).
func TestB7_TraceJSONValidForLargePayload(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("linear.yaml", linearNative)
	f.start()

	longVal := strings.Repeat("x", 200)
	var sub submitOutput
	f.runJSON(&sub, "submit", "--input", "note="+longVal, "linear-native")

	res := f.run("--json", "trace", sub.InstanceID)
	if strings.TrimSpace(res.stdout) == "" {
		t.Fatalf("trace --json: empty stdout")
	}
	if !json.Valid([]byte(res.stdout)) {
		t.Fatalf("trace --json: invalid JSON for a >120-byte payload: %s", res.stdout)
	}
}

// TestB9_B19_LinearNativeReachesCompleted: submitting the linear native
// workflow through the shipped binary must reach 'completed', not fail with
// handler_not_found.
func TestB9_B19_LinearNativeReachesCompleted(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("linear.yaml", linearNative)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "linear-native")

	f.waitForStatus(sub.InstanceID, "completed", 10*time.Second)
}

// TestB23_QuickstartSubmitAfterInitSucceeds: 'awis init' in an empty dir,
// then 'awis start', then exactly the command 'awis init' prints
// ('awis submit hello-world --input name=World') must succeed — no "no
// registered workflow with id" error. This is the out-of-box quickstart.
func TestB23_QuickstartSubmitAfterInitSucceeds(t *testing.T) {
	f := newProject(t)

	initRes := f.run("init")
	if initRes.exitCode != 0 {
		t.Fatalf("init exited %d (want 0): stdout=%s stderr=%s", initRes.exitCode, initRes.stdout, initRes.stderr)
	}

	f.start()

	res := f.run("submit", "hello-world", "--input", "name=World")
	if res.exitCode != 0 {
		t.Fatalf("awis submit hello-world --input name=World exited %d (want 0): stdout=%s stderr=%s", res.exitCode, res.stdout, res.stderr)
	}
	if strings.Contains(res.stderr, "no registered workflow with id") {
		t.Fatalf("quickstart submit failed: %s", res.stderr)
	}
}
