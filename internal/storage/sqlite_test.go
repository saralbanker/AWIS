package storage

// sqlite_test.go — engine-hardening B-25 / B-17 / B-22 read-path defect tests.
//
// Tests in this file:
//   TestListWorkflowsEmptyNamespaceMeansAllNamespaces — B-25
//   TestListInstancesPagedCoversAllRowsExactlyOnce     — B-17 pagination
//   TestListInstancesPagedLimitClamping                — B-17 limit bounds
//   TestListInstancesPagedOffsetPastEndReturnsEmpty     — B-17 offset
//   TestCountInstances                                  — B-17 CountInstances
//   TestListInstancesNotTruncatedByPagingDefault        — B-17 ListInstances unchanged
//   TestAppendEventRejectsSchemaVersionAboveCeiling     — B-22 write side
//   TestReadEventsRejectsRowAboveSchemaVersionCeiling   — B-22 read side (ReadEvents)
//   TestReadEventRangeRejectsRowAboveSchemaVersionCeiling — B-22 read side (ReadEventRange)
//   TestSchemaVersionZeroAndOneStillWork                — B-22 regression guard
//
// These are internal (package storage) tests so they can reach the
// unexported db handle (for direct-SQL fixtures that bypass AppendEvent /
// UpsertInstance validation) and the unexported pagination constants.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

func newSQLiteTestStore(t *testing.T) *SQLiteStorage {
	t.Helper()
	db := openTestDB(t)
	return NewSQLiteStorage(db, time.Now)
}

// ── B-25: ListWorkflows empty-namespace semantics ──────────────────────────

// TestListWorkflowsEmptyNamespaceMeansAllNamespaces verifies that
// ListWorkflows(ctx, "") returns definitions across every namespace rather
// than only rows whose namespace is literally the empty string (B-25).
func TestListWorkflowsEmptyNamespaceMeansAllNamespaces(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()

	defs := []core.WorkflowDefinition{
		{
			SchemaVersion: 1, ID: "wf-all-a", Version: "1.0.0", Namespace: "ns-all-a",
			Name: "wf-all-a", InitialStep: "start", FinalSteps: []string{"start"},
			Steps:    []core.Step{{ID: "start", Name: "Start", Type: core.StepTypeNative, Handler: "noop"}},
			Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
			Metadata: map[string]any{},
		},
		{
			SchemaVersion: 1, ID: "wf-all-b", Version: "1.0.0", Namespace: "ns-all-b",
			Name: "wf-all-b", InitialStep: "start", FinalSteps: []string{"start"},
			Steps:    []core.Step{{ID: "start", Name: "Start", Type: core.StepTypeNative, Handler: "noop"}},
			Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
			Metadata: map[string]any{},
		},
	}
	for _, d := range defs {
		if err := s.RegisterWorkflow(ctx, d); err != nil {
			t.Fatalf("RegisterWorkflow %s: %v", d.ID, err)
		}
	}

	got, err := s.ListWorkflows(ctx, "")
	if err != nil {
		t.Fatalf("ListWorkflows(\"\"): %v", err)
	}
	if len(got) < 2 {
		t.Fatalf("ListWorkflows(\"\"): want at least 2 definitions across namespaces, got %d", len(got))
	}
	seen := map[string]bool{}
	for _, d := range got {
		seen[d.ID] = true
	}
	for _, want := range defs {
		if !seen[want.ID] {
			t.Errorf("ListWorkflows(\"\"): missing %s (namespace %s)", want.ID, want.Namespace)
		}
	}
}

// ── B-17: ListInstancesPaged / CountInstances ──────────────────────────────

type pagingSeedRow struct {
	id        string
	startedAt string
}

// seedInstancesForPaging inserts rows directly into workflow_instances via
// SQL, bypassing UpsertInstance, so pagination fixtures (including
// deliberately duplicate started_at values, or thousands of rows for
// limit-clamping tests) can be built quickly and with exact control over
// started_at.
func seedInstancesForPaging(t *testing.T, s *SQLiteStorage, namespace, status string, rows []pagingSeedRow) {
	t.Helper()
	tx, err := s.db.db.Begin()
	if err != nil {
		t.Fatalf("seedInstancesForPaging: begin: %v", err)
	}
	for _, r := range rows {
		if _, err := tx.Exec(`
			INSERT INTO workflow_instances
				(instance_id, definition_id, definition_version, namespace, status,
				 current_steps, variables, started_at, updated_at, completed_at,
				 version, cancellation_requested)
			VALUES (?, ?, ?, ?, ?, '[]', '{}', ?, ?, NULL, 1, 0)`,
			r.id, "wf-paging", "1.0.0", namespace, status, r.startedAt, r.startedAt,
		); err != nil {
			_ = tx.Rollback()
			t.Fatalf("seedInstancesForPaging: insert %s: %v", r.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("seedInstancesForPaging: commit: %v", err)
	}
}

// TestListInstancesPagedCoversAllRowsExactlyOnce inserts 25 rows across 5
// distinct started_at timestamps (5 rows sharing each timestamp, with
// instance_id assigned out of lexicographic order within each group) and
// walks ListInstancesPaged page-by-page with a limit that does not evenly
// divide the row count. It asserts every row is returned exactly once, in
// (started_at, instance_id) order — proving the instance_id tiebreak is load
// bearing, not just present (B-17).
func TestListInstancesPagedCoversAllRowsExactlyOnce(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()
	const namespace = "ns-page-exhaustive"

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var rows []pagingSeedRow
	for g := 0; g < 5; g++ {
		ts := base.Add(time.Duration(g) * time.Minute).Format(time.RFC3339Nano)
		for k := 0; k < 5; k++ {
			// Permute the id suffix within the group so lexicographic id
			// order differs from insertion order — otherwise the tiebreak
			// could be "accidentally" satisfied by SQLite's default row order.
			suffix := (k*7 + 3) % 5
			rows = append(rows, pagingSeedRow{
				id:        fmt.Sprintf("inst-g%d-s%d", g, suffix),
				startedAt: ts,
			})
		}
	}
	seedInstancesForPaging(t, s, namespace, "running", rows)

	// Expected total order: sort a copy by (startedAt, id) ascending.
	expected := append([]pagingSeedRow(nil), rows...)
	sort.Slice(expected, func(i, j int) bool {
		if expected[i].startedAt != expected[j].startedAt {
			return expected[i].startedAt < expected[j].startedAt
		}
		return expected[i].id < expected[j].id
	})

	filter := core.InstanceFilter{Namespace: namespace}
	const pageSize = 4 // does not evenly divide 25
	var gotOrder []string
	seen := map[string]int{}
	offset := 0
	for {
		page, err := s.ListInstancesPaged(ctx, filter, pageSize, offset)
		if err != nil {
			t.Fatalf("ListInstancesPaged offset=%d: %v", offset, err)
		}
		if len(page) == 0 {
			break
		}
		for _, inst := range page {
			id := string(inst.InstanceID)
			seen[id]++
			gotOrder = append(gotOrder, id)
		}
		offset += pageSize
		if offset > 1000 {
			t.Fatalf("pagination did not terminate (possible infinite loop)")
		}
	}

	if len(gotOrder) != len(rows) {
		t.Fatalf("total rows returned across pages: want %d, got %d", len(rows), len(gotOrder))
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("instance %s returned %d times across pages, want exactly 1", id, count)
		}
	}
	for i, want := range expected {
		if gotOrder[i] != want.id {
			t.Fatalf("order mismatch at position %d: want %s, got %s", i, want.id, gotOrder[i])
		}
	}
}

// TestListInstancesPagedLimitClamping seeds more rows than
// maxInstancesPageSize and verifies: limit<=0 defaults to
// defaultInstancesPageSize; a limit above maxInstancesPageSize is clamped
// down to it; and an in-bounds limit is honored exactly (B-17).
func TestListInstancesPagedLimitClamping(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()
	const namespace = "ns-page-clamp"
	const total = maxInstancesPageSize + 50

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]pagingSeedRow, 0, total)
	for i := 0; i < total; i++ {
		rows = append(rows, pagingSeedRow{
			id:        fmt.Sprintf("clamp-%04d", i),
			startedAt: base.Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano),
		})
	}
	seedInstancesForPaging(t, s, namespace, "running", rows)

	filter := core.InstanceFilter{Namespace: namespace}

	t.Run("limit<=0 defaults", func(t *testing.T) {
		got, err := s.ListInstancesPaged(ctx, filter, 0, 0)
		if err != nil {
			t.Fatalf("ListInstancesPaged(limit=0): %v", err)
		}
		if len(got) != defaultInstancesPageSize {
			t.Errorf("want %d rows (default), got %d", defaultInstancesPageSize, len(got))
		}
	})

	t.Run("limit above ceiling is clamped", func(t *testing.T) {
		got, err := s.ListInstancesPaged(ctx, filter, total*10, 0)
		if err != nil {
			t.Fatalf("ListInstancesPaged(limit=%d): %v", total*10, err)
		}
		if len(got) != maxInstancesPageSize {
			t.Errorf("want %d rows (clamped to ceiling), got %d", maxInstancesPageSize, len(got))
		}
	})

	t.Run("in-bounds limit honored exactly", func(t *testing.T) {
		got, err := s.ListInstancesPaged(ctx, filter, 17, 0)
		if err != nil {
			t.Fatalf("ListInstancesPaged(limit=17): %v", err)
		}
		if len(got) != 17 {
			t.Errorf("want 17 rows, got %d", len(got))
		}
	})
}

// TestListInstancesPagedOffsetPastEndReturnsEmpty verifies an offset beyond
// the result set returns an empty slice rather than erroring (B-17).
func TestListInstancesPagedOffsetPastEndReturnsEmpty(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()
	const namespace = "ns-page-offset"

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	seedInstancesForPaging(t, s, namespace, "running", []pagingSeedRow{
		{id: "inst-1", startedAt: base.Format(time.RFC3339Nano)},
		{id: "inst-2", startedAt: base.Add(time.Second).Format(time.RFC3339Nano)},
	})

	got, err := s.ListInstancesPaged(ctx, core.InstanceFilter{Namespace: namespace}, 10, 1000)
	if err != nil {
		t.Fatalf("ListInstancesPaged(offset past end): unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty slice for offset past end, got %d rows", len(got))
	}
}

// TestCountInstances verifies CountInstances applies the same predicate
// semantics as ListInstances/ListInstancesPaged (B-17).
func TestCountInstances(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()
	const namespace = "ns-count"

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	running := make([]pagingSeedRow, 0, 12)
	for i := 0; i < 12; i++ {
		running = append(running, pagingSeedRow{
			id:        fmt.Sprintf("count-run-%02d", i),
			startedAt: base.Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano),
		})
	}
	waiting := make([]pagingSeedRow, 0, 3)
	for i := 0; i < 3; i++ {
		waiting = append(waiting, pagingSeedRow{
			id:        fmt.Sprintf("count-wait-%02d", i),
			startedAt: base.Add(time.Duration(100+i) * time.Second).Format(time.RFC3339Nano),
		})
	}
	seedInstancesForPaging(t, s, namespace, "running", running)
	seedInstancesForPaging(t, s, namespace, "waiting", waiting)

	total, err := s.CountInstances(ctx, core.InstanceFilter{Namespace: namespace})
	if err != nil {
		t.Fatalf("CountInstances(namespace only): %v", err)
	}
	if total != 15 {
		t.Errorf("want 15 total, got %d", total)
	}

	waitingCount, err := s.CountInstances(ctx, core.InstanceFilter{
		Namespace: namespace, Status: core.InstanceStatusWaiting,
	})
	if err != nil {
		t.Fatalf("CountInstances(namespace+status): %v", err)
	}
	if waitingCount != 3 {
		t.Errorf("want 3 waiting, got %d", waitingCount)
	}
}

// TestListInstancesNotTruncatedByPagingDefault seeds more rows than
// defaultInstancesPageSize and verifies ListInstances (the unbounded,
// engine-tick-facing method) still returns every matching row rather than
// silently truncating to the new pagination default (B-17).
func TestListInstancesNotTruncatedByPagingDefault(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()
	const namespace = "ns-unbounded"
	const total = defaultInstancesPageSize + 25

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]pagingSeedRow, 0, total)
	for i := 0; i < total; i++ {
		rows = append(rows, pagingSeedRow{
			id:        fmt.Sprintf("unbounded-%04d", i),
			startedAt: base.Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano),
		})
	}
	seedInstancesForPaging(t, s, namespace, "running", rows)

	got, err := s.ListInstances(ctx, core.InstanceFilter{Namespace: namespace})
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	if len(got) != total {
		t.Errorf("ListInstances must remain unbounded: want %d rows, got %d", total, len(got))
	}
}

// ── B-22: schema-version ceiling ────────────────────────────────────────────

// TestAppendEventRejectsSchemaVersionAboveCeiling verifies AppendEvent
// refuses to write an event whose SchemaVersion exceeds
// MaxSupportedSchemaVersion (B-22).
func TestAppendEventRejectsSchemaVersionAboveCeiling(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()

	ev := core.ExecutionEvent{
		EventID:       "evt-sv-ceiling-1",
		InstanceID:    core.InstanceID("inst-sv-ceiling"),
		Namespace:     "ns-sv",
		EventType:     core.EventTypeStepCompleted,
		Payload:       []byte(`{}`),
		EmittedAt:     time.Unix(2000, 0).UTC(),
		SequenceNum:   1,
		SchemaVersion: MaxSupportedSchemaVersion + 1,
	}
	err := s.AppendEvent(ctx, ev)
	if err == nil {
		t.Fatal("AppendEvent: want error for schema_version above ceiling, got nil")
	}
	if !errors.Is(err, ErrUnsupportedSchemaVersion) {
		t.Errorf("AppendEvent: want ErrUnsupportedSchemaVersion, got %v", err)
	}

	// Nothing should have been persisted.
	got, err := s.ReadEvents(ctx, ev.InstanceID, 0)
	if err != nil {
		t.Fatalf("ReadEvents after rejected append: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want 0 events persisted after rejected append, got %d", len(got))
	}
}

// TestReadEventsRejectsRowAboveSchemaVersionCeiling simulates a newer AWIS
// writer by inserting a row directly via SQL with schema_version above the
// ceiling, then verifies ReadEvents refuses to project it and returns
// ErrUnsupportedSchemaVersion instead of silently mis-interpreting the row
// (B-22).
func TestReadEventsRejectsRowAboveSchemaVersionCeiling(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()
	const instanceID = "inst-sv-read-ceiling"

	insertRawEventRow(t, s, instanceID, "ns-sv", 1, MaxSupportedSchemaVersion+1, time.Unix(3000, 0).UTC())

	_, err := s.ReadEvents(ctx, core.InstanceID(instanceID), 0)
	if err == nil {
		t.Fatal("ReadEvents: want error for row above schema_version ceiling, got nil")
	}
	if !errors.Is(err, ErrUnsupportedSchemaVersion) {
		t.Errorf("ReadEvents: want ErrUnsupportedSchemaVersion, got %v", err)
	}
}

// TestReadEventRangeRejectsRowAboveSchemaVersionCeiling mirrors
// TestReadEventsRejectsRowAboveSchemaVersionCeiling for ReadEventRange (B-22).
func TestReadEventRangeRejectsRowAboveSchemaVersionCeiling(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()
	const instanceID = "inst-sv-range-ceiling"
	emittedAt := time.Unix(4000, 0).UTC()

	insertRawEventRow(t, s, instanceID, "ns-sv-range", 1, MaxSupportedSchemaVersion+1, emittedAt)

	_, err := s.ReadEventRange(ctx, "ns-sv-range", emittedAt.Add(-time.Hour), emittedAt.Add(time.Hour))
	if err == nil {
		t.Fatal("ReadEventRange: want error for row above schema_version ceiling, got nil")
	}
	if !errors.Is(err, ErrUnsupportedSchemaVersion) {
		t.Errorf("ReadEventRange: want ErrUnsupportedSchemaVersion, got %v", err)
	}
}

// TestSchemaVersionZeroAndOneStillWork is a regression guard: schema_version
// 0 (normalized to 1 on write, per the pre-existing DDL-default contract) and
// schema_version 1 must both continue to append and read back exactly as
// before the B-22 ceiling was added.
func TestSchemaVersionZeroAndOneStillWork(t *testing.T) {
	s := newSQLiteTestStore(t)
	ctx := context.Background()

	cases := []struct {
		name string
		in   int
		want int
	}{
		{"zero normalizes to one", 0, 1},
		{"one stays one", 1, 1},
	}
	for i, tc := range cases {
		instanceID := core.InstanceID(fmt.Sprintf("inst-sv-ok-%d", i))
		ev := core.ExecutionEvent{
			EventID:       fmt.Sprintf("evt-sv-ok-%d", i),
			InstanceID:    instanceID,
			Namespace:     "ns-sv-ok",
			EventType:     core.EventTypeStepCompleted,
			Payload:       []byte(`{}`),
			EmittedAt:     time.Unix(int64(5000+i), 0).UTC(),
			SequenceNum:   1,
			SchemaVersion: tc.in,
		}
		if err := s.AppendEvent(ctx, ev); err != nil {
			t.Fatalf("%s: AppendEvent: %v", tc.name, err)
		}
		got, err := s.ReadEvents(ctx, instanceID, 0)
		if err != nil {
			t.Fatalf("%s: ReadEvents: %v", tc.name, err)
		}
		if len(got) != 1 {
			t.Fatalf("%s: want 1 event, got %d", tc.name, len(got))
		}
		if got[0].SchemaVersion != tc.want {
			t.Errorf("%s: SchemaVersion: want %d, got %d", tc.name, tc.want, got[0].SchemaVersion)
		}
	}
}

// insertRawEventRow inserts one execution_events row directly via SQL,
// bypassing AppendEvent's schema-version enforcement, to simulate a row
// written by a newer AWIS build (B-22 read-side tests). It also inserts the
// owning workflow_instances row so any FK-style expectations elsewhere are
// satisfied.
func insertRawEventRow(t *testing.T, s *SQLiteStorage, instanceID, namespace string, seq, schemaVersion int, emittedAt time.Time) {
	t.Helper()
	if _, err := s.db.db.Exec(`
		INSERT INTO workflow_instances
			(instance_id, definition_id, definition_version, namespace, status,
			 current_steps, variables, started_at, updated_at, completed_at,
			 version, cancellation_requested)
		VALUES (?, 'wf-sv-raw', '1.0.0', ?, 'running', '[]', '{}', ?, ?, NULL, 1, 0)`,
		instanceID, namespace, emittedAt.Format(time.RFC3339Nano), emittedAt.Format(time.RFC3339Nano),
	); err != nil {
		t.Fatalf("insertRawEventRow: seed instance: %v", err)
	}
	if _, err := s.db.db.Exec(`
		INSERT INTO execution_events
			(event_id, instance_id, namespace, event_type, step_id, payload,
			 emitted_at, sequence_num, schema_version)
		VALUES (?, ?, ?, 'step.completed', NULL, '{}', ?, ?, ?)`,
		fmt.Sprintf("evt-raw-%s-%d", instanceID, seq), instanceID, namespace,
		emittedAt.Format(time.RFC3339Nano), seq, schemaVersion,
	); err != nil {
		t.Fatalf("insertRawEventRow: insert event: %v", err)
	}
}

// TestPageSizeBoundsAreThePinnedValues pins storage's pagination bounds to
// literals.
//
// sdk/runtime_readmodel.go restates these values (storageDefaultPageSize /
// storageMaxPageSize) because sdk must not import the concrete adapter. That
// duplication needs a guard on BOTH sides or it drifts silently: the sdk-side
// test can only compare against sdk's own copy, which is a tautology. Pinning
// the literals here means a change to storage's bounds fails this test, and
// whoever fixes it is pointed at the sdk copy that must move with it.
func TestPageSizeBoundsAreThePinnedValues(t *testing.T) {
	// Keep in sync with sdk/runtime_readmodel.go's storageDefaultPageSize
	// and storageMaxPageSize.
	if defaultInstancesPageSize != 100 {
		t.Errorf("defaultInstancesPageSize = %d, want 100; update sdk's storageDefaultPageSize too",
			defaultInstancesPageSize)
	}
	if maxInstancesPageSize != 1000 {
		t.Errorf("maxInstancesPageSize = %d, want 1000; update sdk's storageMaxPageSize too",
			maxInstancesPageSize)
	}
}
