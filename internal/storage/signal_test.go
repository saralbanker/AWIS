package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// newSignalStore opens a fresh DB with a fixed clock and returns the adapter.
func newSignalStore(t *testing.T) *SQLiteStorage {
	t.Helper()
	db := openTestDB(t)
	return NewSQLiteStorage(db, func() time.Time { return time.Unix(0, 0).UTC() })
}

// ── migrations 0003 / 0004 present ────────────────────────────────────────────

func TestSignalAndAuditTablesExist(t *testing.T) {
	db := openTestDB(t)
	wantTables := []string{"signal_inbox", "wait_records", "audit_log"}
	for _, tbl := range wantTables {
		var name string
		if err := db.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl,
		).Scan(&name); err != nil {
			t.Fatalf("table %q not found: %v", tbl, err)
		}
	}
	wantIndexes := []string{"idx_wait_timeout", "idx_signals_instance", "idx_audit_timestamp"}
	for _, idx := range wantIndexes {
		var name string
		if err := db.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, idx,
		).Scan(&name); err != nil {
			t.Fatalf("index %q not found: %v", idx, err)
		}
	}
}

// TestMigrationsReapplyIsNoOp verifies that reopening an already-migrated DB
// leaves the head at 5 and neither re-runs nor errors on 0003/0004/0005.
func TestMigrationsReapplyIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/reapply.db"
	db1, err := Open(path, nil)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_ = db1.Close()
	db2, err := Open(path, nil)
	if err != nil {
		t.Fatalf("second Open (re-apply): %v", err)
	}
	t.Cleanup(func() { _ = db2.Close() })
	v, err := currentVersion(db2.db)
	if err != nil {
		t.Fatalf("currentVersion: %v", err)
	}
	if v != 6 {
		t.Fatalf("want head 6 after re-apply, got %d", v)
	}
}

// ── signal_inbox intake + list-undelivered ────────────────────────────────────

func TestInsertSignal_RoundTrip(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	sig := Signal{
		SignalID:   "sig-1",
		InstanceID: "inst-1",
		SignalName: "approved",
		Payload:    map[string]any{"by": "alice", "n": float64(2)},
		ReceivedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := s.InsertSignal(ctx, sig); err != nil {
		t.Fatalf("InsertSignal: %v", err)
	}
	got, err := s.ListUndeliveredSignals(ctx, "inst-1")
	if err != nil {
		t.Fatalf("ListUndeliveredSignals: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 signal, got %d", len(got))
	}
	g := got[0]
	if g.SignalID != "sig-1" || g.SignalName != "approved" || g.InstanceID != "inst-1" {
		t.Fatalf("scalar mismatch: %+v", g)
	}
	if g.Payload["by"] != "alice" || g.Payload["n"] != float64(2) {
		t.Fatalf("payload round-trip mismatch: %+v", g.Payload)
	}
	if g.DeliveredAt != nil {
		t.Fatalf("fresh signal must be undelivered, got delivered_at=%v", g.DeliveredAt)
	}
	if !g.ReceivedAt.Equal(sig.ReceivedAt) {
		t.Fatalf("received_at mismatch: got %v want %v", g.ReceivedAt, sig.ReceivedAt)
	}
}

func TestInsertSignal_DuplicateIsTypedError(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	sig := Signal{SignalID: "dup", InstanceID: "i", SignalName: "n", ReceivedAt: time.Unix(1, 0).UTC()}
	if err := s.InsertSignal(ctx, sig); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	err := s.InsertSignal(ctx, sig)
	if !errors.Is(err, ErrDuplicateSignal) {
		t.Fatalf("err = %v, want ErrDuplicateSignal", err)
	}
}

// TestListUndeliveredSignals_OrderingAndDeliveredHidden verifies deterministic
// (received_at, signal_id) ordering and that a delivered signal is excluded.
func TestListUndeliveredSignals_OrderingAndDeliveredHidden(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	delivered := base.Add(time.Hour)
	must := func(sig Signal) {
		if err := s.InsertSignal(ctx, sig); err != nil {
			t.Fatalf("insert %s: %v", sig.SignalID, err)
		}
	}
	// b2/b1 share received_at → disambiguated by signal_id; sig-delivered excluded.
	must(Signal{SignalID: "b2", InstanceID: "i", SignalName: "n", ReceivedAt: base.Add(time.Hour)})
	must(Signal{SignalID: "a1", InstanceID: "i", SignalName: "n", ReceivedAt: base})
	must(Signal{SignalID: "b1", InstanceID: "i", SignalName: "n", ReceivedAt: base.Add(time.Hour)})
	must(Signal{SignalID: "sig-delivered", InstanceID: "i", SignalName: "n", ReceivedAt: base, DeliveredAt: &delivered})

	got, err := s.ListUndeliveredSignals(ctx, "i")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := []string{"a1", "b1", "b2"}
	if len(got) != len(want) {
		t.Fatalf("want %d undelivered, got %d (%+v)", len(want), len(got), got)
	}
	for i, id := range want {
		if got[i].SignalID != id {
			t.Fatalf("order[%d] = %s, want %s", i, got[i].SignalID, id)
		}
	}
}

// TestListUndeliveredSignals_InstanceScoping verifies per-instance isolation
// (namespace respected: a signal never crosses the instance boundary) and that
// an empty instanceID scans across all instances.
func TestListUndeliveredSignals_InstanceScoping(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	base := time.Unix(10, 0).UTC()
	_ = s.InsertSignal(ctx, Signal{SignalID: "a-1", InstanceID: "inst-a", SignalName: "n", ReceivedAt: base})
	_ = s.InsertSignal(ctx, Signal{SignalID: "b-1", InstanceID: "inst-b", SignalName: "n", ReceivedAt: base})

	a, err := s.ListUndeliveredSignals(ctx, "inst-a")
	if err != nil {
		t.Fatalf("list inst-a: %v", err)
	}
	if len(a) != 1 || a[0].SignalID != "a-1" {
		t.Fatalf("instance scope failed: %+v", a)
	}
	all, err := s.ListUndeliveredSignals(ctx, "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("empty instanceID must scan all, got %d", len(all))
	}
}

// ── wait_records CRUD ─────────────────────────────────────────────────────────

func TestCreateAndGetWaitRecord_RoundTrip(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	created := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	timeout := created.Add(2 * time.Hour)
	wr := WaitRecord{
		InstanceID:    "inst-1",
		StepID:        "wait-step",
		SignalName:    "approved",
		CreatedAt:     created,
		TimeoutAt:     &timeout,
		TimeoutAction: "fail",
	}
	if err := s.CreateWaitRecord(ctx, wr); err != nil {
		t.Fatalf("CreateWaitRecord: %v", err)
	}
	got, found, err := s.GetWaitRecord(ctx, "inst-1", "wait-step")
	if err != nil {
		t.Fatalf("GetWaitRecord: %v", err)
	}
	if !found {
		t.Fatalf("want found=true")
	}
	if got.SignalName != "approved" || got.TimeoutAction != "fail" {
		t.Fatalf("scalar mismatch: %+v", got)
	}
	if !got.CreatedAt.Equal(created) || got.TimeoutAt == nil || !got.TimeoutAt.Equal(timeout) {
		t.Fatalf("time round-trip mismatch: %+v", got)
	}
}

func TestGetWaitRecord_AbsentIsNotFound(t *testing.T) {
	s := newSignalStore(t)
	_, found, err := s.GetWaitRecord(context.Background(), "nope", "step")
	if err != nil {
		t.Fatalf("GetWaitRecord absent err = %v, want nil", err)
	}
	if found {
		t.Fatalf("want found=false for absent record")
	}
}

func TestCreateWaitRecord_DuplicateIsTypedError(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	wr := WaitRecord{InstanceID: "i", StepID: "s", SignalName: "n", CreatedAt: time.Unix(1, 0).UTC(), TimeoutAction: "fail"}
	if err := s.CreateWaitRecord(ctx, wr); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := s.CreateWaitRecord(ctx, wr)
	if !errors.Is(err, ErrDuplicateWaitRecord) {
		t.Fatalf("err = %v, want ErrDuplicateWaitRecord", err)
	}
}

// TestCreateWaitRecord_UnboundedTimeout verifies a NULL timeout_at round-trips
// and is excluded from the due scan (partial index idx_wait_timeout).
func TestCreateWaitRecord_UnboundedTimeout(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	wr := WaitRecord{InstanceID: "i", StepID: "s", SignalName: "n", CreatedAt: time.Unix(1, 0).UTC(), TimeoutAction: "continue"}
	if err := s.CreateWaitRecord(ctx, wr); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, found, err := s.GetWaitRecord(ctx, "i", "s")
	if err != nil || !found {
		t.Fatalf("get: found=%v err=%v", found, err)
	}
	if got.TimeoutAt != nil {
		t.Fatalf("want nil timeout_at, got %v", got.TimeoutAt)
	}
	due, err := s.ListDueWaitRecords(ctx, time.Unix(1<<40, 0).UTC())
	if err != nil {
		t.Fatalf("list due: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("unbounded wait must never be due, got %d", len(due))
	}
}

// TestListDueWaitRecords_BoundaryAndOrdering verifies the <= now predicate
// (inclusive boundary) and deterministic (timeout_at, instance_id, step_id)
// ordering; a not-yet-due record is excluded.
func TestListDueWaitRecords_BoundaryAndOrdering(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	mk := func(inst, step string, to time.Time) {
		t.Helper()
		timeout := to
		if err := s.CreateWaitRecord(ctx, WaitRecord{
			InstanceID: core.InstanceID(inst), StepID: step, SignalName: "n",
			CreatedAt: base, TimeoutAt: &timeout, TimeoutAction: "fail",
		}); err != nil {
			t.Fatalf("create %s/%s: %v", inst, step, err)
		}
	}
	now := base.Add(time.Hour)
	mk("i2", "s", base)                 // due (before now)
	mk("i1", "s", now)                  // due (== now, inclusive)
	mk("i3", "s", now.Add(time.Minute)) // NOT due (after now)

	due, err := s.ListDueWaitRecords(ctx, now)
	if err != nil {
		t.Fatalf("ListDueWaitRecords: %v", err)
	}
	// Both due; ordered by timeout_at then instance_id: i2@base, then i1@now.
	wantOrder := []string{"i2", "i1"}
	if len(due) != len(wantOrder) {
		t.Fatalf("want %d due, got %d (%+v)", len(wantOrder), len(due), due)
	}
	for i, id := range wantOrder {
		if string(due[i].InstanceID) != id {
			t.Fatalf("due[%d] = %s, want %s", i, due[i].InstanceID, id)
		}
	}
}

func TestDeleteWaitRecordsByInstance(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	created := time.Unix(1, 0).UTC()
	mk := func(inst, step string) {
		if err := s.CreateWaitRecord(ctx, WaitRecord{
			InstanceID: core.InstanceID(inst), StepID: step, SignalName: "n",
			CreatedAt: created, TimeoutAction: "fail",
		}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	mk("inst-1", "a")
	mk("inst-1", "b")
	mk("inst-2", "a")

	if err := s.DeleteWaitRecordsByInstance(ctx, "inst-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, found, _ := s.GetWaitRecord(ctx, "inst-1", "a"); found {
		t.Fatalf("inst-1/a should be deleted")
	}
	if _, found, _ := s.GetWaitRecord(ctx, "inst-1", "b"); found {
		t.Fatalf("inst-1/b should be deleted")
	}
	if _, found, _ := s.GetWaitRecord(ctx, "inst-2", "a"); !found {
		t.Fatalf("inst-2/a must survive delete-by-instance of inst-1")
	}
}

func TestDeleteWaitRecordsByInstance_EmptyIsNoOp(t *testing.T) {
	s := newSignalStore(t)
	if err := s.DeleteWaitRecordsByInstance(context.Background(), "no-such"); err != nil {
		t.Fatalf("delete of empty instance = %v, want nil", err)
	}
}

// ── audit_log append ──────────────────────────────────────────────────────────

func TestAppendAudit_RoundTrip(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	entry := AuditEntry{
		Timestamp:      ts,
		EventType:      "SignalDelivered",
		Actor:          "engine",
		PayloadSummary: "signal=approved instance=inst-1",
	}
	if err := s.AppendAudit(ctx, entry); err != nil {
		t.Fatalf("AppendAudit: %v", err)
	}
	// No read path exists at storage level (M17); query the table directly.
	var (
		gotType, gotActor, gotSummary, gotTS string
	)
	if err := s.db.db.QueryRowContext(ctx,
		`SELECT event_type, actor, payload_summary, timestamp FROM audit_log`,
	).Scan(&gotType, &gotActor, &gotSummary, &gotTS); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if gotType != "SignalDelivered" || gotActor != "engine" || gotSummary != "signal=approved instance=inst-1" {
		t.Fatalf("audit scalar mismatch: type=%q actor=%q summary=%q", gotType, gotActor, gotSummary)
	}
	if gotTS != ts.Format(time.RFC3339Nano) {
		t.Fatalf("timestamp mismatch: got %q want %q", gotTS, ts.Format(time.RFC3339Nano))
	}
}

// TestAppendAudit_IsAppendOnly verifies multiple appends accumulate (never an
// in-place update) and each gets a distinct monotonic surrogate id — the
// append-only invariant (PRD NFR-S-05 / §26).
func TestAppendAudit_IsAppendOnly(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	ts := time.Unix(100, 0).UTC()
	for _, et := range []string{"WorkflowRegistered", "PluginRegistered", "ConfigChanged"} {
		if err := s.AppendAudit(ctx, AuditEntry{Timestamp: ts, EventType: et, Actor: "op", PayloadSummary: et}); err != nil {
			t.Fatalf("append %s: %v", et, err)
		}
	}
	var count int
	if err := s.db.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Fatalf("want 3 append-only rows, got %d", count)
	}
	// Surrogate ids are distinct and monotonic (append order preserved).
	var minID, maxID, distinct int
	if err := s.db.db.QueryRowContext(ctx,
		`SELECT MIN(id), MAX(id), COUNT(DISTINCT id) FROM audit_log`,
	).Scan(&minID, &maxID, &distinct); err != nil {
		t.Fatalf("id stats: %v", err)
	}
	if distinct != 3 || maxID-minID != 2 {
		t.Fatalf("want 3 distinct contiguous ids, got distinct=%d min=%d max=%d", distinct, minID, maxID)
	}
}
