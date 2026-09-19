package storage

// Tests for migration 0008 (D-6: step_claims lease) and ReleaseStepClaim.
// Follows the fresh-apply / upgrade pattern established by
// TestOpenAppliesMigration (db_test.go) and TestRecallStoreMigration0006
// (recall_test.go).

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// TestClaimsMigration0008FreshApply verifies that a fresh DB is migrated to
// head version 8 and step_claims gains the expires_at column.
func TestClaimsMigration0008FreshApply(t *testing.T) {
	db := openTestDB(t)

	v, err := currentVersion(db.db)
	if err != nil {
		t.Fatalf("currentVersion: %v", err)
	}
	if v != 8 {
		t.Fatalf("want schema_version 8, got %d", v)
	}

	if !hasColumn(t, db.db, "step_claims", "expires_at") {
		t.Fatal("step_claims.expires_at column missing after fresh apply of migration 0008")
	}
}

// TestClaimsMigration0008Upgrade verifies the N-1 fixture path: a DB already
// at version 7 (0001–0007 applied, 0008 NOT) is migrated to version 8 on
// Open, adding step_claims.expires_at.
func TestClaimsMigration0008Upgrade(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture0008.db")

	rawDB, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	closeRaw := func() {
		if cerr := rawDB.Close(); cerr != nil {
			t.Errorf("rawDB.Close: %v", cerr)
		}
	}
	if err := applyPragmas(rawDB); err != nil {
		closeRaw()
		t.Fatalf("pragmas: %v", err)
	}
	if err := ensureSchemaVersionTable(rawDB); err != nil {
		closeRaw()
		t.Fatalf("ensureSchemaVersionTable: %v", err)
	}

	// Apply migrations 0001-0007 by hand, building a version-7 fixture.
	for _, name := range []string{
		"0001_core_execution.sql",
		"0002_domain_events.sql",
		"0003_signals.sql",
		"0004_audit.sql",
		"0005_plugins.sql",
		"0006_recall_fts.sql",
		"0007_cancellation_intent.sql",
	} {
		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			closeRaw()
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := rawDB.Exec(string(data)); err != nil {
			closeRaw()
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	for v := 1; v <= 7; v++ {
		if _, err := rawDB.Exec(
			`INSERT INTO schema_version (version, applied_at) VALUES (?, '2026-01-01T00:00:00Z')`, v,
		); err != nil {
			closeRaw()
			t.Fatalf("record version %d: %v", v, err)
		}
	}
	var fixtureVer int
	if err := rawDB.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&fixtureVer); err != nil {
		closeRaw()
		t.Fatalf("currentVersion fixture: %v", err)
	}
	if fixtureVer != 7 {
		closeRaw()
		t.Fatalf("fixture: want version 7, got %d", fixtureVer)
	}
	if hasColumnRaw(t, rawDB, "step_claims", "expires_at") {
		closeRaw()
		t.Fatal("step_claims.expires_at must not exist before migration 0008")
	}
	closeRaw()

	// Open through Open() — must apply migration 0008.
	db, err := Open(path, nil)
	if err != nil {
		t.Fatalf("Open after v7 fixture: %v", err)
	}
	defer func() { _ = db.Close() }()

	v, err := currentVersion(db.db)
	if err != nil {
		t.Fatalf("currentVersion after upgrade: %v", err)
	}
	if v != 8 {
		t.Fatalf("want schema_version 8 after upgrade, got %d", v)
	}
	if !hasColumn(t, db.db, "step_claims", "expires_at") {
		t.Fatal("step_claims.expires_at column missing after upgrade to migration 0008")
	}
}

// TestReleaseStepClaim verifies ReleaseStepClaim deletes exactly the targeted
// (instance, step) claim, leaves an unrelated claim alone, and is a no-op
// (not an error) when the row is already absent.
func TestReleaseStepClaim(t *testing.T) {
	fixedClock := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	db := openTestDB(t)
	s := NewSQLiteStorage(db, func() time.Time { return fixedClock })
	ctx := context.Background()

	inst := core.WorkflowInstance{
		InstanceID:   core.InstanceID("i-release-1"),
		DefinitionID: "wf",
		Namespace:    "default",
		Status:       core.InstanceStatusRunning,
		StartedAt:    fixedClock,
		UpdatedAt:    fixedClock,
		CurrentSteps: []string{},
	}
	if err := s.UpsertInstance(ctx, inst, 0); err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	won, err := s.ClaimStep(ctx, inst.InstanceID, "step-a", "worker-1")
	if err != nil || !won {
		t.Fatalf("ClaimStep step-a: won=%v err=%v", won, err)
	}
	won, err = s.ClaimStep(ctx, inst.InstanceID, "step-b", "worker-1")
	if err != nil || !won {
		t.Fatalf("ClaimStep step-b: won=%v err=%v", won, err)
	}

	if err := s.ReleaseStepClaim(ctx, inst.InstanceID, "step-a"); err != nil {
		t.Fatalf("ReleaseStepClaim step-a: %v", err)
	}

	// step-a is claimable again.
	won, err = s.ClaimStep(ctx, inst.InstanceID, "step-a", "worker-2")
	if err != nil || !won {
		t.Fatalf("ClaimStep step-a after release: won=%v err=%v", won, err)
	}

	// step-b's claim is untouched.
	won, err = s.ClaimStep(ctx, inst.InstanceID, "step-b", "worker-2")
	if err != nil {
		t.Fatalf("ClaimStep step-b re-probe: %v", err)
	}
	if won {
		t.Fatal("step-b claim must still be held by worker-1; ReleaseStepClaim(step-a) must not have touched it")
	}

	// Releasing an already-absent claim is a no-op, not an error.
	if err := s.ReleaseStepClaim(ctx, inst.InstanceID, "step-never-claimed"); err != nil {
		t.Fatalf("ReleaseStepClaim on absent row: %v", err)
	}
}

// TestClaimStepReclaim_NoStepStarted_Reclaimable verifies condition (b) of
// the D-6 reclaim predicate: an EXPIRED claim with NO StepStarted event for
// the same (instance, step) MUST be reclaimable — this is the orphan case
// (a claim committed by ClaimStep whose StepStarted emission never landed).
func TestClaimStepReclaim_NoStepStarted_Reclaimable(t *testing.T) {
	cur := storageBaseClock
	clock := func() time.Time { return cur }
	db := openTestDB(t)
	s := NewSQLiteStorage(db, clock)
	ctx := context.Background()

	inst := core.WorkflowInstance{
		InstanceID:   core.InstanceID("i-reclaim-orphan"),
		DefinitionID: "wf",
		Namespace:    "default",
		Status:       core.InstanceStatusRunning,
		StartedAt:    cur,
		UpdatedAt:    cur,
		CurrentSteps: []string{},
	}
	if err := s.UpsertInstance(ctx, inst, 0); err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	won, err := s.ClaimStep(ctx, inst.InstanceID, "orphan-step", "worker-1")
	if err != nil || !won {
		t.Fatalf("initial ClaimStep: won=%v err=%v", won, err)
	}
	// Deliberately do NOT append a StepStarted event — this is the orphan
	// signature the reclaim predicate must detect.

	// Advance the clock past the lease.
	cur = cur.Add(stepClaimLease + time.Second)

	won, err = s.ClaimStep(ctx, inst.InstanceID, "orphan-step", "worker-2")
	if err != nil {
		t.Fatalf("reclaim ClaimStep: %v", err)
	}
	if !won {
		t.Fatal("expired claim with NO StepStarted event must be reclaimable, but ClaimStep reported won=false")
	}

	// The reclaimed row now belongs to worker-2.
	var workerID string
	if err := db.db.QueryRow(
		`SELECT worker_id FROM step_claims WHERE instance_id = ? AND step_id = ?`,
		string(inst.InstanceID), "orphan-step",
	).Scan(&workerID); err != nil {
		t.Fatalf("query reclaimed row: %v", err)
	}
	if workerID != "worker-2" {
		t.Fatalf("want reclaimed claim held by worker-2, got %q", workerID)
	}
}

// TestClaimStepReclaim_WithStepStarted_NotReclaimable verifies condition (b)
// of the D-6 reclaim predicate the other way: an EXPIRED claim that DOES have
// a StepStarted event for the same (instance, step) must NEVER be reclaimed —
// this is the double-execution hazard the founder's safety constraint exists
// to prevent (a genuinely running step always has StepStarted, so its claim
// can never be stolen out from under it, no matter how long it runs).
func TestClaimStepReclaim_WithStepStarted_NotReclaimable(t *testing.T) {
	cur := storageBaseClock
	clock := func() time.Time { return cur }
	db := openTestDB(t)
	s := NewSQLiteStorage(db, clock)
	ctx := context.Background()

	inst := core.WorkflowInstance{
		InstanceID:   core.InstanceID("i-reclaim-running"),
		DefinitionID: "wf",
		Namespace:    "default",
		Status:       core.InstanceStatusRunning,
		StartedAt:    cur,
		UpdatedAt:    cur,
		CurrentSteps: []string{},
	}
	if err := s.UpsertInstance(ctx, inst, 0); err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	won, err := s.ClaimStep(ctx, inst.InstanceID, "running-step", "worker-1")
	if err != nil || !won {
		t.Fatalf("initial ClaimStep: won=%v err=%v", won, err)
	}

	// The step legitimately started (StepStarted was durably appended) — it
	// may simply be a long-running step, not an orphan.
	if err := s.AppendEvent(ctx, core.ExecutionEvent{
		EventID:     "ev-started-1",
		InstanceID:  inst.InstanceID,
		Namespace:   inst.Namespace,
		EventType:   core.EventTypeStepStarted,
		StepID:      "running-step",
		Payload:     []byte(`{"step_id":"running-step","attempt":1,"inputs":{}}`),
		EmittedAt:   cur,
		SequenceNum: 1,
	}); err != nil {
		t.Fatalf("AppendEvent StepStarted: %v", err)
	}

	// Advance the clock well past the lease — the claim IS expired.
	cur = cur.Add(stepClaimLease * 10)

	won, err = s.ClaimStep(ctx, inst.InstanceID, "running-step", "worker-2")
	if err != nil {
		t.Fatalf("attempted steal ClaimStep: %v", err)
	}
	if won {
		t.Fatal("expired claim WITH a StepStarted event must NEVER be reclaimable (double-execution hazard), but ClaimStep reported won=true")
	}

	// The original claim is untouched (still worker-1).
	var workerID string
	if err := db.db.QueryRow(
		`SELECT worker_id FROM step_claims WHERE instance_id = ? AND step_id = ?`,
		string(inst.InstanceID), "running-step",
	).Scan(&workerID); err != nil {
		t.Fatalf("query claim row: %v", err)
	}
	if workerID != "worker-1" {
		t.Fatalf("want claim still held by worker-1, got %q", workerID)
	}
}

// storageBaseClock is the fixed origin these reclaim-predicate tests advance
// from; kept local to this file so it does not collide with any similarly
// named fixture in another _test.go file in this package.
var storageBaseClock = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

// hasColumn reports whether table has a column named col, via PRAGMA
// table_info (mirrors TestExecutionEventsColumns' query, db_test.go).
func hasColumn(t *testing.T, db *sql.DB, table, col string) bool {
	t.Helper()
	return hasColumnRaw(t, db, table, col)
}

func hasColumnRaw(t *testing.T, db *sql.DB, table, col string) bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		if name == col {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err table_info(%s): %v", table, err)
	}
	return false
}
