package storage_test

// rebuild_nonevented_test.go — B-1 regression tests.
//
// RebuildState replays the append-only EventLog to rewrite the
// workflow_instances projection. Two pieces of that projection are NOT
// derivable from events and so cannot survive a pure replay:
//
//   - `waiting`, the status of an instance parked on a signal. Entering a
//     wait is a tick-time decision that emits no event (the documented
//     EDR-007 §9 gap), so a replay silently downgraded a waiting instance to
//     `running`.
//   - `cancellation_requested`, a non-evented operator request flag, which
//     the rebuild hardcoded back to 0 — silently discarding the request.
//
// Both are now recovered the way rebuild.go already recovered definition
// identity: from durable state that the wipe does not touch. `waiting` comes
// from wait_records (the same source the engine's own restart recovery uses),
// and the cancellation flag from a pre-wipe snapshot of the projection rows.

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// newRebuildTestStorage opens a fresh throwaway store for one test.
func newRebuildTestStorage(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	s, db := openTestStorage(t, filepath.Join(t.TempDir(), "rebuild.db"))
	t.Cleanup(func() { _ = db.Close() })
	return s
}

// seedWaitingInstance appends a minimal event stream for iid, upserts a
// projection row in the given status, and returns the storage handle.
func seedInstance(t *testing.T, s *storage.SQLiteStorage, iid string, status core.InstanceStatus, currentSteps []string) {
	t.Helper()
	ctx := context.Background()

	appendFixtureEvents(t, s, []core.ExecutionEvent{
		fixtureEvent(iid, 1, "t", core.EventTypeWorkflowStarted, "", mustJSON(map[string]any{
			"definition_id": "wf", "version": "1.0.0", "inputs": map[string]any{},
		})),
		fixtureEvent(iid, 2, "t", core.EventTypeStepStarted, "wait-a", mustJSON(map[string]any{
			"step_id": "wait-a", "attempt": 1, "inputs": map[string]any{},
		})),
	})

	inst := core.WorkflowInstance{
		InstanceID:        core.InstanceID(iid),
		DefinitionID:      "wf",
		DefinitionVersion: "1.0.0",
		Namespace:         "t",
		Status:            status,
		CurrentSteps:      currentSteps,
		Variables:         map[string]any{},
		StartedAt:         baseTime,
		UpdatedAt:         baseTime,
	}
	if err := s.UpsertInstance(ctx, inst, 0); err != nil {
		t.Fatalf("UpsertInstance(%s): %v", iid, err)
	}
}

// seedWaitRecord parks iid on a signal in the durable wait_records table.
func seedWaitRecord(t *testing.T, s *storage.SQLiteStorage, iid, stepID, signal string) {
	t.Helper()
	if err := s.CreateWaitRecord(context.Background(), storage.WaitRecord{
		InstanceID:    core.InstanceID(iid),
		StepID:        stepID,
		SignalName:    signal,
		CreatedAt:     baseTime,
		TimeoutAction: "fail",
	}); err != nil {
		t.Fatalf("CreateWaitRecord(%s): %v", iid, err)
	}
}

// statusOf reads a projection row's status and cancellation flag directly.
func statusOf(t *testing.T, s *storage.SQLiteStorage, iid string) (core.InstanceStatus, bool) {
	t.Helper()
	inst, err := s.GetInstance(context.Background(), core.InstanceID(iid))
	if err != nil {
		t.Fatalf("read instance %s: %v", iid, err)
	}
	cancelled, err := s.CancellationRequested(context.Background(), core.InstanceID(iid))
	if err != nil {
		t.Fatalf("CancellationRequested(%s): %v", iid, err)
	}
	return inst.Status, cancelled
}

// TestRebuildPreservesWaitingStatus is the direct B-1 regression: an instance
// parked on a signal must still be `waiting` after a rebuild, not `running`.
func TestRebuildPreservesWaitingStatus(t *testing.T) {
	s := newRebuildTestStorage(t)
	const iid = "inst-waiting"

	seedInstance(t, s, iid, core.InstanceStatusWaiting, []string{"wait-a"})
	seedWaitRecord(t, s, iid, "wait-a", "go")

	if err := s.RebuildState(context.Background()); err != nil {
		t.Fatalf("RebuildState: %v", err)
	}

	got, _ := statusOf(t, s, iid)
	if got != core.InstanceStatusWaiting {
		t.Errorf("status after rebuild = %q, want %q — the wait_records row was ignored",
			got, core.InstanceStatusWaiting)
	}
}

// TestRebuildPreservesCancellationRequested: an operator's cancellation
// request is a non-evented flag that the rebuild used to reset to 0, so a
// rebuild silently un-cancelled the instance.
func TestRebuildPreservesCancellationRequested(t *testing.T) {
	s := newRebuildTestStorage(t)
	ctx := context.Background()
	const iid = "inst-cancelled"

	seedInstance(t, s, iid, core.InstanceStatusRunning, []string{"a"})
	if err := s.SetCancellationRequested(ctx, core.InstanceID(iid)); err != nil {
		t.Fatalf("SetCancellationRequested: %v", err)
	}

	if err := s.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState: %v", err)
	}

	if _, cancelled := statusOf(t, s, iid); !cancelled {
		t.Error("cancellation_requested = false after rebuild; the operator's request was discarded")
	}
}

// TestRebuildDoesNotResurrectTerminalInstance is the ordering rule, and it is
// the one that matters most: a stale wait_records row must NEVER pull an
// instance back out of a terminal status. The EventLog is the source of
// truth; wait_records is only allowed to refine a non-terminal replay result.
//
// Without this rule, an instance that waited, was signalled, and then
// completed would be resurrected to `waiting` by its leftover wait record —
// turning a rebuild into a corruption.
func TestRebuildDoesNotResurrectTerminalInstance(t *testing.T) {
	s := newRebuildTestStorage(t)
	ctx := context.Background()

	terminal := []struct {
		iid    string
		status core.InstanceStatus
		event  core.EventType
	}{
		{"inst-completed", core.InstanceStatusCompleted, core.EventTypeWorkflowCompleted},
		{"inst-failed", core.InstanceStatusFailed, core.EventTypeWorkflowFailed},
		{"inst-cancelled-term", core.InstanceStatusCancelled, core.EventTypeWorkflowCancelled},
	}

	for _, tc := range terminal {
		seedInstance(t, s, tc.iid, core.InstanceStatusRunning, nil)
		appendFixtureEvents(t, s, []core.ExecutionEvent{
			fixtureEvent(tc.iid, 3, "t", tc.event, "", mustJSON(map[string]any{})),
		})
		// The stale wait record that must NOT win.
		seedWaitRecord(t, s, tc.iid, "wait-a", "go")
	}

	if err := s.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState: %v", err)
	}

	for _, tc := range terminal {
		got, _ := statusOf(t, s, tc.iid)
		if got == core.InstanceStatusWaiting {
			t.Errorf("%s: status = %q — a stale wait_records row resurrected a terminal instance",
				tc.iid, got)
		}
		if got != tc.status {
			t.Errorf("%s: status = %q, want %q", tc.iid, got, tc.status)
		}
	}
}

// TestRebuildLeavesOrdinaryInstancesAlone: an instance with no wait record
// and no cancellation must be unaffected, so the two fixes above cannot have
// bled into the normal path.
func TestRebuildLeavesOrdinaryInstancesAlone(t *testing.T) {
	s := newRebuildTestStorage(t)
	ctx := context.Background()
	const iid = "inst-plain"

	seedInstance(t, s, iid, core.InstanceStatusRunning, []string{"a"})

	if err := s.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState: %v", err)
	}

	got, cancelled := statusOf(t, s, iid)
	if got != core.InstanceStatusRunning {
		t.Errorf("status = %q, want %q", got, core.InstanceStatusRunning)
	}
	if cancelled {
		t.Error("cancellation_requested = true on an instance that was never cancelled")
	}
}

// TestRebuildIsIdempotentAcrossNonEventedState: rebuilding twice must be a
// no-op the second time. wait_records survives a rebuild, so a derivation
// that mutated it — or that depended on the projection row it just
// overwrote — would drift on the second pass.
func TestRebuildIsIdempotentAcrossNonEventedState(t *testing.T) {
	s := newRebuildTestStorage(t)
	ctx := context.Background()
	const iid = "inst-twice"

	seedInstance(t, s, iid, core.InstanceStatusWaiting, []string{"wait-a"})
	seedWaitRecord(t, s, iid, "wait-a", "go")
	if err := s.SetCancellationRequested(ctx, core.InstanceID(iid)); err != nil {
		t.Fatalf("SetCancellationRequested: %v", err)
	}

	for pass := 1; pass <= 2; pass++ {
		if err := s.RebuildState(ctx); err != nil {
			t.Fatalf("RebuildState pass %d: %v", pass, err)
		}
		got, cancelled := statusOf(t, s, iid)
		if got != core.InstanceStatusWaiting {
			t.Errorf("pass %d: status = %q, want %q", pass, got, core.InstanceStatusWaiting)
		}
		if !cancelled {
			t.Errorf("pass %d: cancellation_requested was lost", pass)
		}
	}
}

// TestRebuildRefusesUnsupportedSchemaVersion closes the B-22 gap at the
// RebuildState call site.
//
// AppendEvent, ReadEvents and ReadEventRange all refuse rows above
// MaxSupportedSchemaVersion, on the reasoning that mis-projecting the
// append-only source of truth is worse than a failed read. RebuildState reads
// execution_events with its own raw SQL and so bypassed all three — making
// `awis rebuild-state` the single command that would silently reinterpret a
// newer build's events under v1 assumptions, WHILE wiping the projection, so
// the mis-projection replaces correct state rather than merely joining it.
func TestRebuildRefusesUnsupportedSchemaVersion(t *testing.T) {
	s, dbh := openTestStorage(t, filepath.Join(t.TempDir(), "schema.db"))
	t.Cleanup(func() { _ = dbh.Close() })
	ctx := context.Background()

	seedInstance(t, s, "inst-future", core.InstanceStatusRunning, []string{"a"})

	// Write a future-version row directly: AppendEvent would (correctly)
	// refuse it, which is exactly why the rebuild path needs its own guard.
	if _, err := dbh.ExposedDB().ExecContext(ctx, `
		INSERT INTO execution_events
			(event_id, instance_id, namespace, event_type, step_id, payload,
			 emitted_at, sequence_num, schema_version)
		VALUES (?, ?, 't', ?, '', '{}', ?, 99, ?)`,
		"fx-future-99", "inst-future", string(core.EventTypeStepStarted),
		baseTime.UTC().Format(time.RFC3339Nano), storage.MaxSupportedSchemaVersion+1,
	); err != nil {
		t.Fatalf("insert future-version event: %v", err)
	}

	err := s.RebuildState(ctx)
	if !errors.Is(err, storage.ErrUnsupportedSchemaVersion) {
		t.Fatalf("RebuildState err = %v, want ErrUnsupportedSchemaVersion — the rebuild "+
			"silently projected a future schema version under v1 assumptions", err)
	}

	// And it must refuse BEFORE wiping: the projection has to survive intact,
	// or a refused rebuild would still have destroyed the operator's state.
	got, _ := statusOf(t, s, "inst-future")
	if got != core.InstanceStatusRunning {
		t.Errorf("status after refused rebuild = %q, want %q — the projection was wiped "+
			"before the check", got, core.InstanceStatusRunning)
	}
}
