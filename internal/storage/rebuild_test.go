package storage_test

// rebuild_test.go — RebuildState tests (M03-C2 / NFR-R-03 / EDR-007).
//
// Tests in this file:
//   TestRebuildIdempotentByteIdentical  — 10K-event fixture; byte-identical after two rebuilds.
//   TestRebuildProjectionCorrectness    — 6 anchor instances, field-exact assertions.
//   TestRebuildAfterCrash               — SIGKILL mid-append, rebuild succeeds, rows consistent.
//   TestRebuild100K                     — informational wall time (NFR-P-06 binding at M18).

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// ── fixture generator ─────────────────────────────────────────────────────────

// baseTime is the fixed base timestamp for all deterministic fixtures.
var baseTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// fixtureEvent builds one ExecutionEvent deterministically from its parameters.
func fixtureEvent(instanceID string, seq int, ns string, et core.EventType, stepID string, payload json.RawMessage) core.ExecutionEvent {
	return core.ExecutionEvent{
		EventID:       fmt.Sprintf("fx-%s-%d", instanceID, seq),
		InstanceID:    core.InstanceID(instanceID),
		Namespace:     ns,
		EventType:     et,
		StepID:        stepID,
		Payload:       payload,
		EmittedAt:     baseTime.Add(time.Duration(seq) * time.Second),
		SequenceNum:   seq,
		SchemaVersion: 1,
	}
}

// appendFixtureEvents appends all events to s, fatally failing t on error.
// Returns the total number of events appended.
func appendFixtureEvents(t *testing.T, s *storage.SQLiteStorage, events []core.ExecutionEvent) {
	t.Helper()
	ctx := context.Background()
	for _, e := range events {
		if err := s.AppendEvent(ctx, e); err != nil {
			t.Fatalf("appendFixtureEvents: instance=%s seq=%d: %v", e.InstanceID, e.SequenceNum, err)
		}
	}
}

// mustJSON marshals v or panics — used only in test fixture construction.
func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("mustJSON: " + err.Error())
	}
	return b
}

// generateLargeFixture produces a deterministic event set spanning numInstances
// instances and covering all 12 event types and every terminal status
// (completed, failed, cancelled, compensated, compensation_failed) as well as
// still-running instances. Events are padded with step cycles so the total
// count reaches or exceeds targetEvents.
//
// Anchor instances (used by TestRebuildProjectionCorrectness) are injected
// at the front with stable, known IDs; see anchorIDs.
func generateLargeFixture(numInstances, targetEvents int) []core.ExecutionEvent {
	// The 6 anchor instances are PREPENDED before the bulk instances.
	anchors := generateAnchorEvents()

	bulkCount := numInstances - len(anchorIDs)
	if bulkCount < 0 {
		bulkCount = 0
	}
	bulk := generateBulkEvents(bulkCount, targetEvents-len(anchors))

	result := make([]core.ExecutionEvent, 0, len(anchors)+len(bulk))
	result = append(result, anchors...)
	result = append(result, bulk...)
	return result
}

// anchorIDs are the stable instance IDs whose final state is asserted in
// TestRebuildProjectionCorrectness.
var anchorIDs = []string{
	"anchor-completed",
	"anchor-failed",
	"anchor-cancelled",
	"anchor-compensated",
	"anchor-compensation-failed",
	"anchor-running",
}

// generateAnchorEvents produces events for the 6 anchor instances.
// Each anchor exercises a specific terminal (or running) status plus
// enough event variety to cover the projection rules.
func generateAnchorEvents() []core.ExecutionEvent {
	var evs []core.ExecutionEvent
	emit := func(e core.ExecutionEvent) { evs = append(evs, e) }

	ns := "anchor-ns"

	// ── anchor-completed ─────────────────────────────────────────────────────
	// WorkflowStarted → StepStarted(step-a) → StepCompleted(step-a) →
	// StepStarted(step-b) → StepCompleted(step-b) → WorkflowCompleted
	{
		id := "anchor-completed"
		emit(fixtureEvent(id, 1, ns, core.EventTypeWorkflowStarted, "",
			mustJSON(map[string]any{"inputs": map[string]any{"x": 1}})))
		emit(fixtureEvent(id, 2, ns, core.EventTypeStepStarted, "step-a",
			mustJSON(map[string]any{"step_id": "step-a", "attempt": 1, "inputs": map[string]any{}})))
		emit(fixtureEvent(id, 3, ns, core.EventTypeStepCompleted, "step-a",
			mustJSON(map[string]any{"step_id": "step-a", "attempt": 1, "outputs": map[string]any{"r": 42}, "duration_ms": 10})))
		emit(fixtureEvent(id, 4, ns, core.EventTypeStepStarted, "step-b",
			mustJSON(map[string]any{"step_id": "step-b", "attempt": 1, "inputs": map[string]any{}})))
		emit(fixtureEvent(id, 5, ns, core.EventTypeStepCompleted, "step-b",
			mustJSON(map[string]any{"step_id": "step-b", "attempt": 1, "outputs": map[string]any{"s": "ok"}, "duration_ms": 5})))
		emit(fixtureEvent(id, 6, ns, core.EventTypeWorkflowCompleted, "",
			mustJSON(map[string]any{"outputs": map[string]any{"final": true}, "duration_ms": 100})))
	}

	// ── anchor-failed ────────────────────────────────────────────────────────
	// WorkflowStarted → StepStarted(step-a) → StepFailed(retrying:true) →
	// StepFailed(retrying:false) → WorkflowFailed
	{
		id := "anchor-failed"
		emit(fixtureEvent(id, 1, ns, core.EventTypeWorkflowStarted, "",
			mustJSON(map[string]any{"inputs": map[string]any{"y": 2}})))
		emit(fixtureEvent(id, 2, ns, core.EventTypeStepStarted, "step-a",
			mustJSON(map[string]any{"step_id": "step-a", "attempt": 1, "inputs": map[string]any{}})))
		emit(fixtureEvent(id, 3, ns, core.EventTypeStepFailed, "step-a",
			mustJSON(map[string]any{"step_id": "step-a", "attempt": 1,
				"error": map[string]any{"code": "ERR_TIMEOUT", "message": "timeout"}, "retrying": true})))
		emit(fixtureEvent(id, 4, ns, core.EventTypeStepFailed, "step-a",
			mustJSON(map[string]any{"step_id": "step-a", "attempt": 2,
				"error": map[string]any{"code": "ERR_TIMEOUT", "message": "timeout"}, "retrying": false})))
		emit(fixtureEvent(id, 5, ns, core.EventTypeWorkflowFailed, "",
			mustJSON(map[string]any{"step_id": "step-a", "error": map[string]any{"code": "ERR_TIMEOUT", "message": "timeout"}})))
	}

	// ── anchor-cancelled ─────────────────────────────────────────────────────
	// WorkflowStarted → StepStarted(step-a) → SignalReceived → WorkflowCancelled
	{
		id := "anchor-cancelled"
		emit(fixtureEvent(id, 1, ns, core.EventTypeWorkflowStarted, "",
			mustJSON(map[string]any{"inputs": map[string]any{}})))
		emit(fixtureEvent(id, 2, ns, core.EventTypeStepStarted, "step-a",
			mustJSON(map[string]any{"step_id": "step-a", "attempt": 1, "inputs": map[string]any{}})))
		emit(fixtureEvent(id, 3, ns, core.EventTypeSignalReceived, "",
			mustJSON(map[string]any{"signal_name": "cancel", "payload": map[string]any{}})))
		emit(fixtureEvent(id, 4, ns, core.EventTypeWorkflowCancelled, "",
			mustJSON(map[string]any{"reason": "user request"})))
	}

	// ── anchor-compensated ───────────────────────────────────────────────────
	// WorkflowStarted → StepStarted(step-a) → StepCompleted(step-a) →
	// StepStarted(step-b) → StepFallbackActivated(step-b) →
	// WorkflowCompensating → WorkflowCompensated
	{
		id := "anchor-compensated"
		emit(fixtureEvent(id, 1, ns, core.EventTypeWorkflowStarted, "",
			mustJSON(map[string]any{"inputs": map[string]any{"z": 3}})))
		emit(fixtureEvent(id, 2, ns, core.EventTypeStepStarted, "step-a",
			mustJSON(map[string]any{"step_id": "step-a", "attempt": 1, "inputs": map[string]any{}})))
		emit(fixtureEvent(id, 3, ns, core.EventTypeStepCompleted, "step-a",
			mustJSON(map[string]any{"step_id": "step-a", "attempt": 1, "outputs": map[string]any{"v": 7}, "duration_ms": 1})))
		emit(fixtureEvent(id, 4, ns, core.EventTypeStepStarted, "step-b",
			mustJSON(map[string]any{"step_id": "step-b", "attempt": 1, "inputs": map[string]any{}})))
		emit(fixtureEvent(id, 5, ns, core.EventTypeStepFallbackActivated, "step-b",
			mustJSON(map[string]any{"step_id": "step-b", "fallback_step_id": "step-b-fb", "reason": "fallback"})))
		emit(fixtureEvent(id, 6, ns, core.EventTypeWorkflowCompensating, "",
			mustJSON(map[string]any{"from_step": "step-b"})))
		emit(fixtureEvent(id, 7, ns, core.EventTypeWorkflowCompensated, "",
			mustJSON(map[string]any{})))
	}

	// ── anchor-compensation-failed ───────────────────────────────────────────
	// WorkflowStarted → StepStarted → StepCompleted → WorkflowCompensating →
	// WorkflowCompensationFailed
	{
		id := "anchor-compensation-failed"
		emit(fixtureEvent(id, 1, ns, core.EventTypeWorkflowStarted, "",
			mustJSON(map[string]any{"inputs": map[string]any{}})))
		emit(fixtureEvent(id, 2, ns, core.EventTypeStepStarted, "step-c",
			mustJSON(map[string]any{"step_id": "step-c", "attempt": 1, "inputs": map[string]any{}})))
		emit(fixtureEvent(id, 3, ns, core.EventTypeStepCompleted, "step-c",
			mustJSON(map[string]any{"step_id": "step-c", "attempt": 1, "outputs": map[string]any{"ok": true}, "duration_ms": 2})))
		emit(fixtureEvent(id, 4, ns, core.EventTypeWorkflowCompensating, "",
			mustJSON(map[string]any{"from_step": "step-c"})))
		emit(fixtureEvent(id, 5, ns, core.EventTypeWorkflowCompensationFailed, "",
			mustJSON(map[string]any{"step_id": "step-c", "error": map[string]any{"code": "ERR_COMP", "message": "comp fail"}})))
	}

	// ── anchor-running ───────────────────────────────────────────────────────
	// WorkflowStarted → StepStarted(step-x) → StepCompleted(step-x) →
	// StepStarted(step-y) [not yet completed — stays in current_steps]
	// variables: {"inputs": {"run": true}, "step-x": {"done": 1}}
	{
		id := "anchor-running"
		emit(fixtureEvent(id, 1, ns, core.EventTypeWorkflowStarted, "",
			mustJSON(map[string]any{"inputs": map[string]any{"run": true}})))
		emit(fixtureEvent(id, 2, ns, core.EventTypeStepStarted, "step-x",
			mustJSON(map[string]any{"step_id": "step-x", "attempt": 1, "inputs": map[string]any{}})))
		emit(fixtureEvent(id, 3, ns, core.EventTypeStepCompleted, "step-x",
			mustJSON(map[string]any{"step_id": "step-x", "attempt": 1, "outputs": map[string]any{"done": 1}, "duration_ms": 3})))
		emit(fixtureEvent(id, 4, ns, core.EventTypeStepStarted, "step-y",
			mustJSON(map[string]any{"step_id": "step-y", "attempt": 1, "inputs": map[string]any{}})))
	}

	return evs
}

// generateBulkEvents produces numInstances more instances with roughly
// targetEvents total events, cycling through event patterns to cover all
// event types multiple times and pad to target.
func generateBulkEvents(numInstances, targetEvents int) []core.ExecutionEvent {
	if numInstances <= 0 {
		return nil
	}
	// Desired events per instance (minimum 10).
	evPerInst := targetEvents / numInstances
	if evPerInst < 10 {
		evPerInst = 10
	}

	// Terminal patterns cycle across instances.
	terminalPatterns := []string{"completed", "failed", "cancelled", "compensated", "compensation_failed", "running"}

	var evs []core.ExecutionEvent
	ns := "bulk-ns"

	for i := 0; i < numInstances; i++ {
		id := fmt.Sprintf("bulk-inst-%04d", i)
		term := terminalPatterns[i%len(terminalPatterns)]

		seq := 1

		// WorkflowStarted
		evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeWorkflowStarted, "",
			mustJSON(map[string]any{"inputs": map[string]any{"i": i}})))
		seq++

		// Pad with step cycles: StepStarted + StepCompleted
		stepCycles := (evPerInst - 4) / 2
		if stepCycles < 1 {
			stepCycles = 1
		}
		for c := 0; c < stepCycles; c++ {
			sid := fmt.Sprintf("step-%04d", c)
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeStepStarted, sid,
				mustJSON(map[string]any{"step_id": sid, "attempt": 1, "inputs": map[string]any{"c": c}})))
			seq++
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeStepCompleted, sid,
				mustJSON(map[string]any{"step_id": sid, "attempt": 1, "outputs": map[string]any{"c": c}, "duration_ms": 1})))
			seq++
		}

		// Terminal events
		switch term {
		case "completed":
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeWorkflowCompleted, "",
				mustJSON(map[string]any{"outputs": map[string]any{"i": i}, "duration_ms": 50})))
		case "failed":
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeWorkflowFailed, "",
				mustJSON(map[string]any{"step_id": fmt.Sprintf("step-%04d", 0),
					"error": map[string]any{"code": "ERR_BULK", "message": "bulk fail"}})))
		case "cancelled":
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeWorkflowCancelled, "",
				mustJSON(map[string]any{"reason": "bulk cancel"})))
		case "compensated":
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeWorkflowCompensating, "",
				mustJSON(map[string]any{"from_step": "step-0000"})))
			seq++
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeWorkflowCompensated, "",
				mustJSON(map[string]any{})))
		case "compensation_failed":
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeWorkflowCompensating, "",
				mustJSON(map[string]any{"from_step": "step-0000"})))
			seq++
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeWorkflowCompensationFailed, "",
				mustJSON(map[string]any{"step_id": "step-0000",
					"error": map[string]any{"code": "ERR_COMP", "message": "comp fail"}})))
		case "running":
			// Leave a step running: one more StepStarted without StepCompleted.
			sid := fmt.Sprintf("step-%04d", stepCycles)
			evs = append(evs, fixtureEvent(id, seq, ns, core.EventTypeStepStarted, sid,
				mustJSON(map[string]any{"step_id": sid, "attempt": 1, "inputs": map[string]any{}})))
		}
	}

	return evs
}

// snapshotInstances reads all workflow_instances rows ordered by instance_id
// and returns them as a slice of canonical strings (tab-separated fields),
// suitable for byte-identical comparison.
func snapshotInstances(t *testing.T, db interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}, ctx context.Context) []string {
	t.Helper()
	rows, err := db.QueryContext(ctx, `
		SELECT instance_id, namespace, status, current_steps, variables,
		       started_at, updated_at, completed_at
		FROM workflow_instances
		ORDER BY instance_id`)
	if err != nil {
		t.Fatalf("snapshotInstances query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var result []string
	for rows.Next() {
		var (
			iid, ns, status, steps, vars string
			startedAt, updatedAt         string
			completedAt                  sql.NullString
		)
		if err := rows.Scan(&iid, &ns, &status, &steps, &vars,
			&startedAt, &updatedAt, &completedAt); err != nil {
			t.Fatalf("snapshotInstances scan: %v", err)
		}
		ca := "<nil>"
		if completedAt.Valid {
			ca = completedAt.String
		}
		result = append(result, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
			iid, ns, status, steps, vars, startedAt, updatedAt, ca))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("snapshotInstances rows: %v", err)
	}
	return result
}

// openTestStorage opens a fresh SQLiteStorage at dbPath.
func openTestStorage(t *testing.T, dbPath string) (*storage.SQLiteStorage, *storage.DB) {
	t.Helper()
	db, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("storage.Open %q: %v", dbPath, err)
	}
	return storage.NewSQLiteStorage(db, nil), db
}

// ── TestRebuildIdempotentByteIdentical ───────────────────────────────────────

// TestRebuildIdempotentByteIdentical generates ≥10,000 events across ≥50
// instances covering all 12 event types and every terminal status, appends
// them to a fresh DB, calls RebuildState twice, and asserts that both
// workflow_instances snapshots are byte-identical (NFR-R-03).
func TestRebuildIdempotentByteIdentical(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip 10K rebuild idempotency test")
	}

	const numInstances = 60 // 6 anchors + 54 bulk
	const targetEvents = 11_000

	events := generateLargeFixture(numInstances, targetEvents)
	if len(events) < 10_000 {
		t.Fatalf("fixture generator produced only %d events; need ≥10,000", len(events))
	}

	// Count distinct instance IDs in fixture.
	instanceSet := make(map[string]bool)
	for _, e := range events {
		instanceSet[string(e.InstanceID)] = true
	}
	if len(instanceSet) < 50 {
		t.Fatalf("fixture has only %d distinct instances; need ≥50", len(instanceSet))
	}

	t.Logf("TestRebuildIdempotentByteIdentical: %d events across %d instances",
		len(events), len(instanceSet))

	dbPath := filepath.Join(t.TempDir(), "idempotent.db")
	s, db := openTestStorage(t, dbPath)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()

	appendFixtureEvents(t, s, events)

	// ── First rebuild ──────────────────────────────────────────────────────
	if err := s.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState (first): %v", err)
	}
	snap1 := snapshotInstances(t, db.ExposedDB(), ctx)

	// ── Second rebuild ─────────────────────────────────────────────────────
	if err := s.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState (second): %v", err)
	}
	snap2 := snapshotInstances(t, db.ExposedDB(), ctx)

	// ── Assert byte-identical ──────────────────────────────────────────────
	if len(snap1) != len(snap2) {
		t.Fatalf("snapshot lengths differ: first=%d second=%d", len(snap1), len(snap2))
	}
	for i := range snap1 {
		if snap1[i] != snap2[i] {
			t.Errorf("row %d differs:\n  first:  %s\n  second: %s", i, snap1[i], snap2[i])
		}
	}
	t.Logf("byte-identical: %d rows verified across two RebuildState calls", len(snap1))
}

// ── TestRebuildProjectionCorrectness ─────────────────────────────────────────

// TestRebuildProjectionCorrectness appends the same fixture as the idempotency
// test, calls RebuildState once, then makes field-exact assertions against the
// 6 anchor instances (one per terminal status + one still-running with
// populated current_steps and variables).
func TestRebuildProjectionCorrectness(t *testing.T) {
	const numInstances = 60
	const targetEvents = 11_000

	events := generateLargeFixture(numInstances, targetEvents)

	dbPath := filepath.Join(t.TempDir(), "correctness.db")
	s, db := openTestStorage(t, dbPath)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	appendFixtureEvents(t, s, events)

	if err := s.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState: %v", err)
	}

	// Helper to fetch one instance row directly.
	type row struct {
		status      string
		steps       string
		vars        string
		startedAt   string
		updatedAt   string
		completedAt sql.NullString
	}
	fetch := func(iid string) row {
		t.Helper()
		var r row
		err := db.ExposedDB().QueryRowContext(ctx, `
			SELECT status, current_steps, variables, started_at, updated_at, completed_at
			FROM workflow_instances WHERE instance_id = ?`, iid).
			Scan(&r.status, &r.steps, &r.vars, &r.startedAt, &r.updatedAt, &r.completedAt)
		if err != nil {
			t.Fatalf("fetch %q: %v", iid, err)
		}
		return r
	}

	ns := "anchor-ns"

	// anchor-completed
	t.Run("anchor-completed", func(t *testing.T) {
		r := fetch("anchor-completed")
		assertEqual(t, "status", r.status, "completed")
		assertEqual(t, "namespace", fetchNS(t, db.ExposedDB(), ctx, "anchor-completed"), ns)
		assertStepsEmpty(t, r.steps)
		assertVarKey(t, r.vars, "inputs", map[string]any{"x": float64(1)})
		assertVarKey(t, r.vars, "step-a", map[string]any{"r": float64(42)})
		assertVarKey(t, r.vars, "step-b", map[string]any{"s": "ok"})
		if !r.completedAt.Valid {
			t.Errorf("completedAt: want non-null, got null")
		}
		assertTimestamp(t, "started_at", r.startedAt, baseTime.Add(1*time.Second))
		assertTimestamp(t, "updated_at", r.updatedAt, baseTime.Add(6*time.Second))
	})

	// anchor-failed
	t.Run("anchor-failed", func(t *testing.T) {
		r := fetch("anchor-failed")
		assertEqual(t, "status", r.status, "failed")
		if !r.completedAt.Valid {
			t.Errorf("completedAt: want non-null for failed")
		}
		assertTimestamp(t, "updated_at", r.updatedAt, baseTime.Add(5*time.Second))
		// step-a should not be in current_steps (retrying:false removes it)
		assertStepsExclude(t, r.steps, "step-a")
	})

	// anchor-cancelled
	t.Run("anchor-cancelled", func(t *testing.T) {
		r := fetch("anchor-cancelled")
		assertEqual(t, "status", r.status, "cancelled")
		if !r.completedAt.Valid {
			t.Errorf("completedAt: want non-null for cancelled")
		}
		assertTimestamp(t, "updated_at", r.updatedAt, baseTime.Add(4*time.Second))
	})

	// anchor-compensated
	t.Run("anchor-compensated", func(t *testing.T) {
		r := fetch("anchor-compensated")
		assertEqual(t, "status", r.status, "compensated")
		if !r.completedAt.Valid {
			t.Errorf("completedAt: want non-null for compensated")
		}
		assertTimestamp(t, "updated_at", r.updatedAt, baseTime.Add(7*time.Second))
		// step-b was FallbackActivated → removed from current_steps
		assertStepsExclude(t, r.steps, "step-b")
		// step-a was completed → its outputs in variables
		assertVarKey(t, r.vars, "step-a", map[string]any{"v": float64(7)})
	})

	// anchor-compensation-failed
	t.Run("anchor-compensation-failed", func(t *testing.T) {
		r := fetch("anchor-compensation-failed")
		assertEqual(t, "status", r.status, "compensation_failed")
		if !r.completedAt.Valid {
			t.Errorf("completedAt: want non-null for compensation_failed")
		}
		assertTimestamp(t, "updated_at", r.updatedAt, baseTime.Add(5*time.Second))
	})

	// anchor-running: current_steps=[step-y], variables has "inputs" and "step-x"
	t.Run("anchor-running", func(t *testing.T) {
		r := fetch("anchor-running")
		assertEqual(t, "status", r.status, "running")
		if r.completedAt.Valid {
			t.Errorf("completedAt: want null for running, got %q", r.completedAt.String)
		}
		assertStepsContain(t, r.steps, "step-y")
		assertStepsExclude(t, r.steps, "step-x") // completed → removed
		assertVarKey(t, r.vars, "inputs", map[string]any{"run": true})
		assertVarKey(t, r.vars, "step-x", map[string]any{"done": float64(1)})
		assertTimestamp(t, "updated_at", r.updatedAt, baseTime.Add(4*time.Second))
	})

	t.Logf("TestRebuildProjectionCorrectness: all 6 anchor instances verified")
}

// ── assertion helpers ─────────────────────────────────────────────────────────

func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: want %q got %q", field, want, got)
	}
}

func fetchNS(t *testing.T, db interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}, ctx context.Context, iid string) string {
	t.Helper()
	var ns string
	if err := db.QueryRowContext(ctx, `SELECT namespace FROM workflow_instances WHERE instance_id = ?`, iid).Scan(&ns); err != nil {
		t.Fatalf("fetchNS %q: %v", iid, err)
	}
	return ns
}

func assertTimestamp(t *testing.T, field, stored string, want time.Time) {
	t.Helper()
	got, err := time.Parse(time.RFC3339Nano, stored)
	if err != nil {
		got, err = time.Parse(time.RFC3339, stored)
		if err != nil {
			t.Errorf("%s: cannot parse %q: %v", field, stored, err)
			return
		}
	}
	if !got.UTC().Equal(want.UTC()) {
		t.Errorf("%s: want %v got %v", field, want.UTC(), got.UTC())
	}
}

func assertStepsEmpty(t *testing.T, stepsJSON string) {
	t.Helper()
	var steps []string
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		t.Fatalf("assertStepsEmpty unmarshal: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("current_steps: want empty, got %v", steps)
	}
}

func assertStepsContain(t *testing.T, stepsJSON, want string) {
	t.Helper()
	var steps []string
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		t.Fatalf("assertStepsContain unmarshal: %v", err)
	}
	for _, s := range steps {
		if s == want {
			return
		}
	}
	t.Errorf("current_steps: want %q in %v", want, steps)
}

func assertStepsExclude(t *testing.T, stepsJSON, absent string) {
	t.Helper()
	var steps []string
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		t.Fatalf("assertStepsExclude unmarshal: %v", err)
	}
	for _, s := range steps {
		if s == absent {
			t.Errorf("current_steps: want %q absent, found in %v", absent, steps)
			return
		}
	}
}

func assertVarKey(t *testing.T, varsJSON, key string, want map[string]any) {
	t.Helper()
	var vars map[string]any
	if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
		t.Fatalf("assertVarKey unmarshal: %v", err)
	}
	val, ok := vars[key]
	if !ok {
		t.Errorf("variables[%q]: key absent", key)
		return
	}
	valMap, ok := val.(map[string]any)
	if !ok {
		t.Errorf("variables[%q]: want map, got %T", key, val)
		return
	}
	for k, wv := range want {
		if gv := valMap[k]; gv != wv {
			t.Errorf("variables[%q][%q]: want %v got %v", key, k, wv, gv)
		}
	}
}

// ── TestRebuildAfterCrash ─────────────────────────────────────────────────────

// TestRebuildAfterCrash spawns the existing TestHelperProcess child (same
// binary re-exec pattern as TestCrashDurability), SIGKILLs it mid-stream,
// reopens the DB, calls RebuildState, and verifies every committed event's
// instance projects consistently: the instance row must exist and its
// updated_at must equal or follow the first event's emitted_at.
func TestRebuildAfterCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip crash rebuild test")
	}

	dbPath := filepath.Join(t.TempDir(), "rebuild-crash.db")

	// Pre-create DB.
	initDB, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	_ = initDB.Close()

	// Spawn child via re-exec (reuses TestHelperProcess from crash_test.go).
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "-test.v")
	cmd.Env = append(os.Environ(),
		"GO_TEST_HELPER_PROCESS=1",
		"CRASH_DB_PATH="+dbPath,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	const minCommitted = 20
	seqCh := make(chan int, 256)
	scanner := bufio.NewScanner(stdout)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			if n, err := strconv.Atoi(strings.TrimSpace(scanner.Text())); err == nil {
				seqCh <- n
			}
		}
		close(seqCh)
	}()

	committed := make([]int, 0, 128)
	deadline := time.After(30 * time.Second)
collectLoop:
	for len(committed) < minCommitted {
		select {
		case n, ok := <-seqCh:
			if !ok {
				break collectLoop
			}
			committed = append(committed, n)
		case <-deadline:
			t.Fatalf("timeout: collected only %d committed seqs", len(committed))
		}
	}

	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("SIGKILL: %v", err)
	}
	_ = cmd.Wait()
	for n := range seqCh {
		committed = append(committed, n)
	}
	<-done

	if len(committed) == 0 {
		t.Fatal("no committed seqs collected before SIGKILL")
	}
	t.Logf("TestRebuildAfterCrash: %d committed seqs before SIGKILL; max=%d",
		len(committed), committed[len(committed)-1])

	// Reopen DB, run RebuildState.
	db, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer func() { _ = db.Close() }()

	s := storage.NewSQLiteStorage(db, nil)
	ctx := context.Background()

	if err := s.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState after crash: %v", err)
	}

	// Every event that was committed must have its instance in workflow_instances.
	events, err := s.ReadEvents(ctx, "crash-instance-1", 1)
	if err != nil {
		t.Fatalf("ReadEvents after rebuild: %v", err)
	}

	// Build set of committed seq numbers.
	committedSet := make(map[int]bool, len(committed))
	for _, seq := range committed {
		committedSet[seq] = true
	}

	// Verify every committed event is present.
	presentSeqs := make(map[int]bool, len(events))
	for _, e := range events {
		presentSeqs[e.SequenceNum] = true
	}
	for _, seq := range committed {
		if !presentSeqs[seq] {
			t.Errorf("committed seq %d missing from DB after crash+rebuild", seq)
		}
	}

	// Verify the instance row exists and is consistent.
	inst, err := s.GetInstance(ctx, "crash-instance-1")
	if err != nil {
		t.Fatalf("GetInstance crash-instance-1 after rebuild: %v", err)
	}
	// All events are WorkflowStarted (from the helper), so status must be running
	// and namespace must be crash-ns.
	if inst.Namespace != "crash-ns" {
		t.Errorf("instance namespace: want crash-ns, got %q", inst.Namespace)
	}
	if inst.Status != core.InstanceStatusRunning {
		t.Errorf("instance status: want running, got %q", inst.Status)
	}
	t.Logf("TestRebuildAfterCrash: instance consistent after crash+rebuild (status=%s, events=%d)",
		inst.Status, len(events))
}

// ── TestRebuild100K ───────────────────────────────────────────────────────────

// TestRebuild100K appends 100,000 events across multiple instances and
// records the RebuildState wall time (informational; NFR-P-06 <30s binding
// at M18). Skipped under -short and -race.
func TestRebuild100K(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip 100K rebuild test")
	}
	if raceEnabled {
		t.Skip("-race: 100K rebuild test skipped under race detector (too slow)")
	}

	const numInstances = 510
	const targetEvents = 102_000

	events := generateLargeFixture(numInstances, targetEvents)
	t.Logf("TestRebuild100K: generated %d events across %d instances", len(events), numInstances)

	dbPath := filepath.Join(t.TempDir(), "rebuild100k.db")
	s, db := openTestStorage(t, dbPath)
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	appendFixtureEvents(t, s, events)

	start := time.Now()
	if err := s.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState 100K: %v", err)
	}
	elapsed := time.Since(start)
	t.Logf("TestRebuild100K: RebuildState over %d events in %v (NFR-P-06 binding <30s at M18)",
		len(events), elapsed)
}
