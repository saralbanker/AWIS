package storage

import (
	"database/sql"
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

// TestOpenAppliesMigration verifies that a fresh empty DB is migrated to
// version 1 on Open.
func TestOpenAppliesMigration(t *testing.T) {
	db := openTestDB(t)
	v, err := currentVersion(db.db)
	if err != nil {
		t.Fatalf("currentVersion: %v", err)
	}
	if v != 1 {
		t.Fatalf("want schema_version 1, got %d", v)
	}
}

// TestOpenIdempotent verifies that reopening an already-migrated DB is a no-op:
// version stays at 1 and no error is returned.
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
	if v != 1 {
		t.Fatalf("want schema_version 1 after reopen, got %d", v)
	}
}

// TestNMinus1Fixture tests the N-1 fixture path: an empty DB (version 0)
// migrated to version 1. This is the explicit N-1 case for migration 0001.
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
	if v1 != 1 {
		t.Fatalf("want version 1 after migration, got %d", v1)
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
