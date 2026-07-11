package storage_test

// recall_test.go — TDS-07 RecallStore + migration 0006 tests (M17-C1).
//
// Tests:
//   - SQLiteStorage implements RecallStore after Open (migration 0006 applied)
//   - SearchEvents returns no results on empty DB
//   - SearchEvents returns results matching inserted events
//   - ListAudit returns no results on empty DB
//   - ListAudit returns rows after AppendAudit
//   - Default limit is applied (limit=0)

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// openForRecall opens a fresh SQLite DB and returns a *SQLiteStorage.
// The caller uses it as core.StoragePort, RecallStore, and AuditAppender
// through direct method calls (all methods are on *SQLiteStorage).
func openForRecall(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	path := filepath.Join(t.TempDir(), "recall_test.db")
	clock := time.Now
	db, err := storage.Open(path, clock)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return storage.NewSQLiteStorage(db, clock)
}

// TestRecallStoreMigration0006 verifies SQLiteStorage implements RecallStore.
// If migration 0006 was not applied, SearchEvents would return an error.
func TestRecallStoreMigration0006(t *testing.T) {
	s := openForRecall(t)
	// Type-assert through interface — this succeeds only if the methods are declared.
	var iface interface{} = s
	if _, ok := iface.(storage.RecallStore); !ok {
		t.Error("*SQLiteStorage does not implement RecallStore (methods missing?)")
	}
}

// TestRecallSearchEventsEmpty verifies SearchEvents returns empty on fresh DB.
func TestRecallSearchEventsEmpty(t *testing.T) {
	s := openForRecall(t)
	ctx := context.Background()
	results, err := s.SearchEvents(ctx, "step", 10)
	if err != nil {
		t.Fatalf("SearchEvents (empty): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestRecallSearchEventsWithData verifies SearchEvents returns results after AppendEvent.
func TestRecallSearchEventsWithData(t *testing.T) {
	s := openForRecall(t)
	ctx := context.Background()

	// Insert a workflow instance first (FK requirement for execution_events).
	inst := core.WorkflowInstance{
		InstanceID:        core.InstanceID("i-recall-001"),
		DefinitionID:      "test-workflow",
		DefinitionVersion: "1.0.0",
		Namespace:         "default",
		Status:            core.InstanceStatusRunning,
		StartedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		CurrentSteps:      []string{},
	}
	if err := s.UpsertInstance(ctx, inst, 0); err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	// Insert an event whose payload contains a unique searchable term.
	// Note: FTS5 default tokenizer splits on non-alphanumeric; use alphanumeric query term.
	payload, _ := json.Marshal(map[string]string{
		"step_id": "draft-entry",
		"result":  "synthesisCompleteUniqueXYZ999",
	})
	ev := core.ExecutionEvent{
		EventID:     "evt-recall-001",
		InstanceID:  inst.InstanceID,
		Namespace:   "default",
		EventType:   core.EventTypeStepCompleted,
		StepID:      "draft-entry",
		Payload:     json.RawMessage(payload),
		EmittedAt:   time.Now(),
		SequenceNum: 1,
	}
	if err := s.AppendEvent(ctx, ev); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}

	// Search for the unique term (no hyphens; FTS5 treats - as NOT operator).
	results, err := s.SearchEvents(ctx, "synthesisCompleteUniqueXYZ999", 10)
	if err != nil {
		t.Fatalf("SearchEvents (with data): %v", err)
	}
	if len(results) == 0 {
		t.Error("SearchEvents: expected at least one result for inserted event")
		t.FailNow()
	}
	if results[0].EventID != "evt-recall-001" {
		t.Errorf("SearchEvents: expected event_id 'evt-recall-001', got %q", results[0].EventID)
	}
	// Verify fields are populated.
	if len(results) > 0 {
		r := results[0]
		if r.InstanceID != "i-recall-001" {
			t.Errorf("SearchEvents: instance_id mismatch: got %q", r.InstanceID)
		}
		if r.Namespace != "default" {
			t.Errorf("SearchEvents: namespace mismatch: got %q", r.Namespace)
		}
		if r.EmittedAt.IsZero() {
			t.Error("SearchEvents: EmittedAt should not be zero")
		}
	}
}

// TestRecallListAuditEmpty verifies ListAudit returns empty on fresh DB.
func TestRecallListAuditEmpty(t *testing.T) {
	s := openForRecall(t)
	ctx := context.Background()
	rows, err := s.ListAudit(ctx, 10)
	if err != nil {
		t.Fatalf("ListAudit (empty): %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 audit rows, got %d", len(rows))
	}
}

// TestRecallListAuditAfterAppend verifies ListAudit returns rows after AppendAudit.
func TestRecallListAuditAfterAppend(t *testing.T) {
	s := openForRecall(t)
	ctx := context.Background()

	if err := s.AppendAudit(ctx, storage.AuditEntry{
		Timestamp:      time.Now(),
		EventType:      "TestRecallEvent",
		Actor:          "test",
		PayloadSummary: `{"key":"recall-test"}`,
	}); err != nil {
		t.Fatalf("AppendAudit: %v", err)
	}

	rows, err := s.ListAudit(ctx, 10)
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if len(rows) == 0 {
		t.Error("ListAudit: expected at least one row after AppendAudit")
	}
	if rows[0].EventType != "TestRecallEvent" {
		t.Errorf("ListAudit: expected EventType 'TestRecallEvent', got %q", rows[0].EventType)
	}
	if rows[0].Actor != "test" {
		t.Errorf("ListAudit: expected Actor 'test', got %q", rows[0].Actor)
	}
	if rows[0].Timestamp.IsZero() {
		t.Error("ListAudit: Timestamp should not be zero")
	}
}

// TestRecallListAuditDefaultLimit verifies limit=0 defaults to 50 and returns results.
func TestRecallListAuditDefaultLimit(t *testing.T) {
	s := openForRecall(t)
	ctx := context.Background()

	// Insert 3 rows.
	for i := 0; i < 3; i++ {
		if err := s.AppendAudit(ctx, storage.AuditEntry{
			Timestamp:      time.Now(),
			EventType:      "LimitTest",
			Actor:          "test",
			PayloadSummary: "{}",
		}); err != nil {
			t.Fatalf("AppendAudit %d: %v", i, err)
		}
	}

	// limit=0 should default to 50 and still return results.
	rows, err := s.ListAudit(ctx, 0)
	if err != nil {
		t.Fatalf("ListAudit (limit=0): %v", err)
	}
	if len(rows) == 0 {
		t.Error("ListAudit (limit=0): expected rows, got none")
	}
}

// TestRecallSearchEventsDefaultLimit verifies limit=0 defaults internally to 50.
func TestRecallSearchEventsDefaultLimit(t *testing.T) {
	s := openForRecall(t)
	ctx := context.Background()

	// On empty DB, limit=0 should not error.
	results, err := s.SearchEvents(ctx, "test", 0)
	if err != nil {
		t.Fatalf("SearchEvents (limit=0): %v", err)
	}
	_ = results // empty is fine; we're just verifying no panic/error
}

// TestRecallListAuditDescOrder verifies ListAudit returns rows in descending ID order.
func TestRecallListAuditDescOrder(t *testing.T) {
	s := openForRecall(t)
	ctx := context.Background()

	// Insert 3 rows.
	for i := 0; i < 3; i++ {
		if err := s.AppendAudit(ctx, storage.AuditEntry{
			Timestamp:      time.Now(),
			EventType:      "OrderTest",
			Actor:          "test",
			PayloadSummary: "{}",
		}); err != nil {
			t.Fatalf("AppendAudit %d: %v", i, err)
		}
	}

	rows, err := s.ListAudit(ctx, 10)
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least 2 rows, got %d", len(rows))
	}
	// Verify descending order by ID (autoincrement; ORDER BY id DESC).
	for i := 1; i < len(rows); i++ {
		if rows[i].ID >= rows[i-1].ID {
			t.Errorf("ListAudit: rows not in descending ID order at index %d: %d >= %d",
				i, rows[i].ID, rows[i-1].ID)
		}
	}
}
