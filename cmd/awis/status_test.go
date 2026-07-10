package main

// status_test.go — golden tests for 'awis status' human and JSON outputs (M14-C2 T5).
// Uses sdk DeterministicMode + an in-memory storage fixture to produce stable output.

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/sdk"
)

// openTestStorage creates a temporary SQLite storage backed by a temp file.
func openTestStorage(t *testing.T) core.StoragePort {
	t.Helper()
	dir := t.TempDir()
	store, err := sdk.SQLiteStorage(dir + "/runtime.db")
	if err != nil {
		t.Fatalf("openTestStorage: %v", err)
	}
	return store
}

// seedStatusFixture inserts deterministic workflow instances into store.
// It inserts one running and one completed instance so status output is non-trivial.
func seedStatusFixture(t *testing.T, store core.StoragePort) (runningID, completedID core.InstanceID) {
	t.Helper()
	ctx := context.Background()

	// Fixed epoch times for determinism.
	epoch := time.Date(2026, 7, 10, 14, 23, 0, 0, time.UTC)

	// Register a workflow first.
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
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	// Running instance.
	runningID = "det-000001"
	startedAt := epoch.Add(-12 * time.Second)
	runningInst := core.WorkflowInstance{
		InstanceID:        runningID,
		DefinitionID:      "capture-decision",
		DefinitionVersion: "1.0.0",
		Namespace:         "oip",
		Status:            core.InstanceStatusRunning,
		CurrentSteps:      []string{"draft-entry"},
		Variables:         map[string]any{},
		StartedAt:         startedAt,
		UpdatedAt:         epoch,
	}
	if err := store.UpsertInstance(ctx, runningInst, 0); err != nil {
		t.Fatalf("UpsertInstance (running): %v", err)
	}

	// Completed instance.
	completedID = "det-000002"
	completedAt := epoch.Add(-1 * time.Minute)
	completedStartedAt := epoch.Add(-2 * time.Minute)
	completedInst := core.WorkflowInstance{
		InstanceID:        completedID,
		DefinitionID:      "capture-decision",
		DefinitionVersion: "1.0.0",
		Namespace:         "oip",
		Status:            core.InstanceStatusCompleted,
		CurrentSteps:      []string{},
		Variables:         map[string]any{},
		StartedAt:         completedStartedAt,
		UpdatedAt:         completedAt,
		CompletedAt:       &completedAt,
	}
	if err := store.UpsertInstance(ctx, completedInst, 0); err != nil {
		t.Fatalf("UpsertInstance (completed): %v", err)
	}

	return runningID, completedID
}

// TestStatusJSON verifies that status --json produces valid JSON with the required fields.
func TestStatusJSON(t *testing.T) {
	store := openTestStorage(t)
	seedStatusFixture(t, store)

	// Override globalDataDir to use the test storage path.
	// We test the buildStatusJSON function directly to avoid needing a real data-dir.
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

	out := buildStatusJSON(now, activeInsts, completedInsts)
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Verify required shape.
	var parsed statusOutputJSON
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if parsed.Timestamp == "" {
		t.Error("timestamp field missing")
	}
	if len(parsed.Active) != 1 {
		t.Errorf("active count: got %d, want 1", len(parsed.Active))
	}
	if len(parsed.Recent) != 1 {
		t.Errorf("recent count: got %d, want 1", len(parsed.Recent))
	}
	if parsed.Active[0].WorkflowID != "capture-decision" {
		t.Errorf("active[0].workflow_id: got %q, want %q", parsed.Active[0].WorkflowID, "capture-decision")
	}
	if parsed.Recent[0].Status != string(core.InstanceStatusCompleted) {
		t.Errorf("recent[0].status: got %q, want %q", parsed.Recent[0].Status, core.InstanceStatusCompleted)
	}
}

// TestStatusHuman verifies the human status output has the required structural lines.
func TestStatusHuman(t *testing.T) {
	store := openTestStorage(t)
	seedStatusFixture(t, store)

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

	// Capture output of the human printing path.
	var buf bytes.Buffer
	printStatusTableTo(&buf, now, activeInsts, completedInsts, false, 10)

	s := buf.String()
	if !strings.Contains(s, "AWIS status") {
		t.Errorf("missing 'AWIS status' header: %s", s)
	}
	if !strings.Contains(s, "ACTIVE") {
		t.Errorf("missing ACTIVE section: %s", s)
	}
	if !strings.Contains(s, "capture-decision") {
		t.Errorf("missing workflow id: %s", s)
	}
	if !strings.Contains(s, "RECENT") {
		t.Errorf("missing RECENT section: %s", s)
	}
}

// TestStatusSymbols verifies the TDS-07 status symbols.
func TestStatusSymbols(t *testing.T) {
	cases := []struct {
		status core.InstanceStatus
		want   string
	}{
		{core.InstanceStatusRunning, "●"},
		{core.InstanceStatusWaiting, "○"},
		{core.InstanceStatusCompleted, "✓"},
		{core.InstanceStatusFailed, "✗"},
		{core.InstanceStatusCancelled, "✗"},
		{core.InstanceStatusCompensated, "✓"},
	}
	for _, c := range cases {
		got := statusSymbol(c.status)
		if got != c.want {
			t.Errorf("statusSymbol(%s) = %q, want %q", c.status, got, c.want)
		}
	}
}

// TestStatusSortByUpdatedAt verifies descending sort.
func TestStatusSortByUpdatedAt(t *testing.T) {
	now := time.Now()
	insts := []core.WorkflowInstance{
		{InstanceID: "a", UpdatedAt: now.Add(-1 * time.Minute)},
		{InstanceID: "b", UpdatedAt: now},
		{InstanceID: "c", UpdatedAt: now.Add(-2 * time.Minute)},
	}
	sortByUpdatedAt(insts)
	if insts[0].InstanceID != "b" || insts[1].InstanceID != "a" || insts[2].InstanceID != "c" {
		t.Errorf("sort order wrong: got %s %s %s, want b a c",
			insts[0].InstanceID, insts[1].InstanceID, insts[2].InstanceID)
	}
}
