package main

// c3r_test.go — M14-C3r golden tests.
//
// Adds byte-stable JSON goldens for all 10 leaf commands that lacked them:
//   start, stop, status, submit, signal, cancel, trace,
//   workflow_validate, workflow_list, workflow_show.
//
// Strategy: commands that block or require external processes (start, stop,
// submit, signal, cancel, trace) are tested by marshalling a deterministic
// output struct — exactly as the production encoder does — and comparing
// against the golden.  Commands that can be invoked in-process against a
// seeded temp DB (workflow_list, workflow_show, workflow_validate, status)
// call the run-function directly with globalDataDir pointed at a temp db.
//
// Existing shape assertions (TestSubmitOutputJSON, etc.) in c3_test.go are
// preserved; this file only adds checkGolden calls.

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/sdk"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// encodeNoEscape marshals v using json.NewEncoder with SetEscapeHTML(false),
// matching the encoding used by every production run-function.
func encodeNoEscape(t *testing.T, v any) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		t.Fatalf("encodeNoEscape: %v", err)
	}
	return buf.Bytes()
}

// seedGoldenWorkflow registers the deterministic "capture-decision" workflow
// (single step, no transitions, no fallbacks) used by workflow_list and
// workflow_show goldens.
func seedGoldenWorkflow(t *testing.T, store core.StoragePort) {
	t.Helper()
	ctx := context.Background()
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "capture-decision",
		Version:       "1.0.0",
		Namespace:     "oip",
		Name:          "Capture Decision",
		Steps: []core.Step{
			{ID: "draft-entry", Type: core.StepTypeNative, Handler: "oip.draft"},
		},
		InitialStep: "draft-entry",
		FinalSteps:  []string{"draft-entry"},
	}
	if err := store.RegisterWorkflow(ctx, *def); err != nil {
		t.Fatalf("seedGoldenWorkflow: RegisterWorkflow: %v", err)
	}
}

// openGoldenStorage creates a temp SQLite storage for golden tests and
// returns a restore function that resets the globals.
func openGoldenStorage(t *testing.T) (dir string, restore func()) {
	t.Helper()
	dir = t.TempDir()
	store, err := sdk.SQLiteStorage(filepath.Join(dir, "runtime.db"))
	if err != nil {
		t.Fatalf("openGoldenStorage: %v", err)
	}
	seedGoldenWorkflow(t, store)
	restore = setGlobals(dir, true)
	return dir, restore
}

// ── start.json ────────────────────────────────────────────────────────────────

// TestStartGoldenJSON verifies the start --json startup event golden.
// runStart blocks forever; we test the output struct directly (same path
// the production encoder takes).
func TestStartGoldenJSON(t *testing.T) {
	out := startOutput{
		Event:        "started",
		Version:      version,
		DBPath:       "<data-dir>/runtime.db",
		Workflows:    []string{"test-hello 1.0.0"},
		Plugins:      []string{},
		Intelligence: "none (zero-AI mode)",
		PID:          0,
	}
	got := encodeNoEscape(t, out)
	checkGolden(t, "start.json", got)
}

// ── stop.json ─────────────────────────────────────────────────────────────────

// TestStopGoldenJSON verifies the stop --json output golden.
// runStop requires a live process; we test the output struct directly.
func TestStopGoldenJSON(t *testing.T) {
	out := stopOutput{PID: 12345, Stopped: true, ElapsedMs: 42}
	got := encodeNoEscape(t, out)
	checkGolden(t, "stop.json", got)
}

// ── status.json ───────────────────────────────────────────────────────────────

// TestStatusGoldenJSON verifies the status --json list-mode golden.
// Uses the same seeded fixture as TestStatusJSON (status_test.go) with a
// fixed clock so output is byte-stable.
func TestStatusGoldenJSON(t *testing.T) {
	store := openTestStorage(t)
	seedStatusFixture(t, store)

	epoch := time.Date(2026, 7, 10, 14, 23, 0, 0, time.UTC)
	now := time.Date(2026, 7, 10, 14, 23, 1, 0, time.UTC)
	ctx := context.Background()

	activeInsts, err := store.ListInstances(ctx, core.InstanceFilter{Status: core.InstanceStatusRunning})
	if err != nil {
		t.Fatalf("ListInstances running: %v", err)
	}
	completedInsts, err := store.ListInstances(ctx, core.InstanceFilter{Status: core.InstanceStatusCompleted})
	if err != nil {
		t.Fatalf("ListInstances completed: %v", err)
	}

	_ = epoch // anchor comment: epoch used by seedStatusFixture (2026-07-10 14:23:00 UTC)
	out := buildStatusJSON(now, activeInsts, completedInsts)
	got := encodeNoEscape(t, out)
	checkGolden(t, "status.json", got)
}

// ── submit.json ───────────────────────────────────────────────────────────────

// TestSubmitGoldenJSON verifies the submit --json output golden.
// Submission requires an engine-running instance; we test the output struct.
func TestSubmitGoldenJSON(t *testing.T) {
	out := submitOutput{
		InstanceID:      "det-000001",
		WorkflowID:      "capture-decision",
		WorkflowVersion: "1.0.0",
		Namespace:       "oip",
		Status:          "pending",
		DurationMs:      nil,
	}
	got := encodeNoEscape(t, out)
	checkGolden(t, "submit.json", got)
}

// ── signal.json ───────────────────────────────────────────────────────────────

// TestSignalGoldenJSON verifies the signal --json output golden.
func TestSignalGoldenJSON(t *testing.T) {
	next := "append-to-record"
	out := signalOutput{
		InstanceID: "det-000001",
		SignalName: "entry_confirmed",
		Delivered:  true,
		NextStep:   &next,
	}
	got := encodeNoEscape(t, out)
	checkGolden(t, "signal.json", got)
}

// ── cancel.json ───────────────────────────────────────────────────────────────

// TestCancelGoldenJSON verifies the cancel --json output golden.
func TestCancelGoldenJSON(t *testing.T) {
	out := cancelOutput{
		InstanceID: "det-000001",
		Requested:  true,
		Compensate: false,
		Reason:     nil,
	}
	got := encodeNoEscape(t, out)
	checkGolden(t, "cancel.json", got)
}

// ── trace.json ────────────────────────────────────────────────────────────────

// TestTraceGoldenJSON verifies the trace --json output golden.
// Uses fixed epoch times for byte-stable output.
func TestTraceGoldenJSON(t *testing.T) {
	epoch := time.Date(2026, 7, 10, 14, 23, 0, 0, time.UTC)
	durMs := 89000
	out := traceOutputJSON{
		InstanceID:      "det-000001",
		WorkflowID:      "capture-decision",
		WorkflowVersion: "1.0.0",
		Namespace:       "oip",
		Status:          "completed",
		DurationMs:      &durMs,
		Events: []traceEventJSON{
			{
				Seq:        1,
				EventType:  "WorkflowStarted",
				OccurredAt: epoch.Format(time.RFC3339),
				RelativeMs: 0,
				Payload:    json.RawMessage(`{}`),
			},
			{
				Seq:        2,
				EventType:  "WorkflowCompleted",
				OccurredAt: epoch.Add(89 * time.Second).Format(time.RFC3339),
				RelativeMs: 89000,
				Payload:    json.RawMessage(`{}`),
			},
		},
	}
	got := encodeNoEscape(t, out)
	checkGolden(t, "trace.json", got)
}

// ── workflow_validate.json ────────────────────────────────────────────────────

// TestWorkflowValidateGoldenJSON verifies the workflow validate --json output
// for a valid file. Calls runWorkflowValidate in-process with the helloWorkflowYAML.
func TestWorkflowValidateGoldenJSON(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "test-hello.yaml")
	if err := os.WriteFile(yamlPath, []byte(helloWorkflowYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Capture JSON output from runWorkflowValidate.
	origJSON := globalJSON
	globalJSON = true
	defer func() { globalJSON = origJSON }()

	got := captureOutput(func() {
		runWorkflowValidate([]string{yamlPath})
	})

	// Replace the temp dir prefix with the canonical golden path.
	stable := strings.ReplaceAll(string(got), yamlPath, "workflows/test-hello.yaml")
	checkGolden(t, "workflow_validate.json", []byte(stable))
}

// ── workflow_list.json ────────────────────────────────────────────────────────

// TestWorkflowListGoldenJSON verifies the workflow list --json output golden.
// Seeds the golden workflow into a temp DB and captures runWorkflowList output.
func TestWorkflowListGoldenJSON(t *testing.T) {
	_, restore := openGoldenStorage(t)
	defer restore()

	got := captureOutput(func() {
		runWorkflowList([]string{"--namespace", "oip"})
	})
	checkGolden(t, "workflow_list.json", got)
}

// ── workflow_show.json ────────────────────────────────────────────────────────

// TestWorkflowShowGoldenJSON verifies the workflow show --json output golden.
// Seeds the golden workflow into a temp DB and captures runWorkflowShow output.
func TestWorkflowShowGoldenJSON(t *testing.T) {
	_, restore := openGoldenStorage(t)
	defer restore()

	got := captureOutput(func() {
		runWorkflowShow([]string{"--namespace", "oip", "capture-decision"})
	})
	checkGolden(t, "workflow_show.json", got)
}
