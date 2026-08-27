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
	"path/filepath"
	"testing"

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
