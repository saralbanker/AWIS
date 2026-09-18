package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// openTestDB opens a fresh DB in a temp directory with a real clock.
func openTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestOpenAppliesMigration verifies that a fresh empty DB is migrated to the
// current head version on Open (head = 6 since migrations 0003_signals +
// 0004_audit + 0005_plugins, M07-C1 / F-4; M12-C2).
func TestOpenAppliesMigration(t *testing.T) {
	db := openTestDB(t)
	v, err := currentVersion(db.db)
	if err != nil {
		t.Fatalf("currentVersion: %v", err)
	}
	if v != 7 {
		t.Fatalf("want schema_version 7, got %d", v)
	}
}

// TestOpenIdempotent verifies that reopening an already-migrated DB is a no-op:
// version stays at the head version and no error is returned.
func TestOpenIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// First open — migrates.
	db1, err := Open(path, nil)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_ = db1.Close()

	// Second open — must be idempotent.
	db2, err := Open(path, nil)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer func() {
		if cerr := db2.Close(); cerr != nil {
			t.Errorf("db2.Close: %v", cerr)
		}
	}()

	v, err := currentVersion(db2.db)
	if err != nil {
		t.Fatalf("currentVersion after reopen: %v", err)
	}
	if v != 7 {
		t.Fatalf("want schema_version 7 after reopen, got %d", v)
	}
}

// TestOpen_DatabaseFileMode0600 verifies the runtime database file — and its
// WAL sidecars, when present — are created at mode 0600 (D-11, PRD §32 rows
// 38/53, founder ruling DEC-9): a same-UID-but-different-owner process (e.g.
// a plugin subprocess spawned under a dedicated plugins_user, internal/plugin
// spawnPlugin) must be unable to open the file directly.
func TestOpen_DatabaseFileMode0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runtime.db")

	db, err := Open(path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q): %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("db file mode = %v, want 0600", got)
	}

	// -wal/-shm sidecars: WAL mode is engaged synchronously by applyPragmas
	// (PRAGMA journal_mode=WAL), so both exist by the time Open returns.
	for _, suffix := range []string{"-wal", "-shm"} {
		sidecar := path + suffix
		info, err := os.Stat(sidecar)
		if err != nil {
			t.Fatalf("Stat(%q): %v (expected WAL sidecar to exist)", sidecar, err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("sidecar %q mode = %v, want 0600", sidecar, got)
		}
	}
}

// TestOpen_InMemoryDatabaseSkipsChmod verifies that Open(":memory:", ...) —
// used by sdk/testing and examples/hello_workflow — does not attempt to
// chmod a backing file that does not exist (D-11 regression guard: this
// previously failed outright with "chmod :memory:: no such file or
// directory").
func TestOpen_InMemoryDatabaseSkipsChmod(t *testing.T) {
	db, err := Open(":memory:", nil)
	if err != nil {
		t.Fatalf("Open(\":memory:\"): %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
}

// TestNMinus1Fixture tests the from-scratch fixture path: an empty DB (version
// 0) migrated to the head version by Open (head = 6 since 0003_signals +
// 0004_audit + 0005_plugins). The explicit N-1 case for migration 0002 is TestNMinus1Fixture0002.
func TestNMinus1Fixture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.db")

	// Open a raw sqlite DB and create the schema_version table only,
	// leaving it empty (version 0 fixture).
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
	v0, err := currentVersion(rawDB)
	if err != nil {
		closeRaw()
		t.Fatalf("currentVersion before: %v", err)
	}
	if v0 != 0 {
		closeRaw()
		t.Fatalf("want version 0 fixture, got %d", v0)
	}
	closeRaw()

	// Now open through our Open() which should apply migration 0001.
	db, err := Open(path, nil)
	if err != nil {
		t.Fatalf("Open after fixture: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			t.Errorf("db.Close: %v", cerr)
		}
	}()

	v1, err := currentVersion(db.db)
	if err != nil {
		t.Fatalf("currentVersion after migration: %v", err)
	}
	if v1 != 7 {
		t.Fatalf("want head version 7 after migration, got %d", v1)
	}
}

// TestNMinus1Fixture0002 tests the N-1 fixture path for migration 0002: a DB
// already at version 1 (0001 applied, 0002 NOT) is migrated to version 2 on
// Open, which creates domain_events + its index (F-2 / EDR-011 §7).
func TestNMinus1Fixture0002(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture0002.db")

	// Build a version-1 fixture: apply ONLY migration 0001 by hand, recording it
	// in schema_version, then leave the DB at version 1 (0002 unapplied).
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
	sql0001, err := migrationsFS.ReadFile("migrations/0001_core_execution.sql")
	if err != nil {
		closeRaw()
		t.Fatalf("read 0001: %v", err)
	}
	if _, err := rawDB.Exec(string(sql0001)); err != nil {
		closeRaw()
		t.Fatalf("apply 0001: %v", err)
	}
	if _, err := rawDB.Exec(
		`INSERT INTO schema_version (version, applied_at) VALUES (1, '2026-01-01T00:00:00Z')`,
	); err != nil {
		closeRaw()
		t.Fatalf("record 0001: %v", err)
	}
	v1, err := currentVersion(rawDB)
	if err != nil {
		closeRaw()
		t.Fatalf("currentVersion before: %v", err)
	}
	if v1 != 1 {
		closeRaw()
		t.Fatalf("want version 1 fixture, got %d", v1)
	}
	// domain_events must NOT yet exist in the N-1 fixture.
	var tbl string
	if err := rawDB.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='domain_events'`,
	).Scan(&tbl); err == nil {
		closeRaw()
		t.Fatalf("domain_events must not exist before migration 0002")
	}
	closeRaw()

	// Open through Open() which applies migration 0002.
	db, err := Open(path, nil)
	if err != nil {
		t.Fatalf("Open after fixture: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			t.Errorf("db.Close: %v", cerr)
		}
	}()

	v2, err := currentVersion(db.db)
	if err != nil {
		t.Fatalf("currentVersion after migration: %v", err)
	}
	// Open applies every pending migration, so a version-1 fixture advances to
	// the current head (7) — not merely to 2. This test still proves 0002
	// created domain_events (below); the head just moves with later migrations.
	if v2 != 7 {
		t.Fatalf("want head version 7 after migration, got %d", v2)
	}
	if err := db.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='domain_events'`,
	).Scan(&tbl); err != nil {
		t.Fatalf("domain_events not created by 0002: %v", err)
	}
	var idx string
	if err := db.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='index' AND name='idx_domain_events_ns_type_time'`,
	).Scan(&idx); err != nil {
		t.Fatalf("domain_events index not created by 0002: %v", err)
	}
}

// TestWALModeActive verifies that journal_mode=WAL is in effect after Open.
func TestWALModeActive(t *testing.T) {
	db := openTestDB(t)
	var mode string
	row := db.db.QueryRow(`PRAGMA journal_mode`)
	if err := row.Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("want journal_mode=wal, got %q", mode)
	}
}

// TestTablesExist verifies all tables from migration 0001 are present
// (execution_events, workflow_definitions, step_results_cache, workflow_instances,
// step_claims — fold-forward per M03).
func TestTablesExist(t *testing.T) {
	db := openTestDB(t)

	want := []string{
		"execution_events",
		"workflow_definitions",
		"step_results_cache",
		"workflow_instances",
		"step_claims",
	}
	for _, tbl := range want {
		var name string
		err := db.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", tbl, err)
		}
	}
}

// TestIndexesExist verifies all indexes from migration 0001 are present.
func TestIndexesExist(t *testing.T) {
	db := openTestDB(t)

	wantIndexes := []string{
		"idx_events_instance_seq",
		"idx_events_ns_time",
		"idx_instances_ns_status",
	}
	for _, idx := range wantIndexes {
		var name string
		err := db.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, idx,
		).Scan(&name)
		if err != nil {
			t.Errorf("index %q not found: %v", idx, err)
		}
	}
}

// TestExecutionEventsColumns verifies that execution_events has exactly the 9
// specified columns (in any order).
func TestExecutionEventsColumns(t *testing.T) {
	db := openTestDB(t)

	rows, err := db.db.Query(`PRAGMA table_info(execution_events)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			t.Errorf("rows.Close: %v", cerr)
		}
	}()

	want := map[string]bool{
		"event_id":       true,
		"instance_id":    true,
		"namespace":      true,
		"event_type":     true,
		"step_id":        true,
		"payload":        true,
		"emitted_at":     true,
		"sequence_num":   true,
		"schema_version": true,
	}

	got := map[string]bool{}
	for rows.Next() {
		var cid int
		var colName, colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &colName, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan table_info row: %v", err)
		}
		got[colName] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("table_info rows: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("execution_events: want %d columns, got %d: %v", len(want), len(got), got)
	}
	for col := range want {
		if !got[col] {
			t.Errorf("execution_events missing column %q", col)
		}
	}
}
