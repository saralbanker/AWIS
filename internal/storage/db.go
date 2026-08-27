// Package storage implements the SQLite-backed StoragePort adapter for the AWIS
// runtime (Blueprint §20; edr-003). Only migration + DB-open concerns live in
// this card (M02-C1); StoragePort method implementations follow in M02-C2.
package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite" // register "sqlite" driver
)

// DB is the SQLite connection wrapper. It holds a single *sql.DB opened in WAL
// mode. Methods on StoragePort are added in subsequent cards.
type DB struct {
	db  *sql.DB
	now func() string // injectable clock for migration timestamps
}

// Open opens (or creates) the SQLite database at path, applies WAL pragmas and
// connection settings, then runs all pending migrations automatically.
//
// clock is an injectable source of the current time used by the migration
// runner. Pass nil to use time.Now (UTC, RFC3339Nano).
func Open(path string, clock func() time.Time) (*DB, error) {
	if clock == nil {
		clock = time.Now
	}
	nowFn := func() string { return clock().UTC().Format(time.RFC3339Nano) }

	sqlDB, err := sql.Open("sqlite", buildDSN(path))
	if err != nil {
		return nil, fmt.Errorf("storage: open sqlite %q: %w", path, err)
	}

	// Single-writer semantics WITHIN this process (Blueprint §9, local mode).
	// It does not constrain other processes: every `awis` invocation is its
	// own process opening the same file, so cross-process write contention is
	// real and is handled by the DSN options in buildDSN.
	sqlDB.SetMaxOpenConns(1)

	if err := applyPragmas(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	d := &DB{db: sqlDB, now: nowFn}

	if err := runMigrations(sqlDB, nowFn); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("storage: migrate: %w", err)
	}

	return d, nil
}

// Close releases the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// ExposedDB returns the underlying *sql.DB for use in tests that need
// direct SQL access (e.g. raw-snapshot assertions in rebuild_test.go).
// Production code must not call this method.
func (d *DB) ExposedDB() *sql.DB {
	return d.db
}

// buildDSN turns a database file path into a driver DSN carrying the
// connection options that MUST hold from the very first statement.
//
// _txlock=immediate is the important one, and it fixes a real failure. Several
// write paths — AppendEvent most obviously — open a transaction, SELECT to
// establish an invariant (the current MAX(sequence_num) for an instance), then
// INSERT based on what they read. Under the default DEFERRED locking that
// transaction begins as a READER and only tries to upgrade to a writer at the
// INSERT. In WAL mode, if another connection has committed in the meantime,
// that upgrade fails with SQLITE_BUSY IMMEDIATELY — busy_timeout does NOT
// apply, because waiting cannot help: the transaction's read snapshot is
// already stale and it must be rolled back and retried.
//
// The observable symptom was `awis submit` failing outright with
// "database is locked (5) (SQLITE_BUSY)" whenever a few submissions ran at
// once — each `awis` invocation being a separate process contending for the
// same file. BEGIN IMMEDIATE takes the write lock up front, so there is no
// stale-snapshot upgrade to fail, and busy_timeout governs the wait.
//
// busy_timeout is set here as well as in applyPragmas because a DSN option
// applies to EVERY connection the pool opens, including one established after
// a connection is dropped and replaced — whereas a PRAGMA issued once via
// Exec applies only to the connection that happened to serve it.
//
// A path that already carries a query string keeps it; its own settings win,
// so an explicit override is still possible.
func buildDSN(path string) string {
	const opts = "_txlock=immediate&_pragma=busy_timeout(5000)"
	if strings.Contains(path, "?") {
		return path + "&" + opts
	}
	return path + "?" + opts
}

// applyPragmas configures the WAL journal mode, busy timeout, foreign keys, and
// synchronous level on the connection (edr-003; FR-ST-06).
func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA busy_timeout=5000`,
		`PRAGMA foreign_keys=ON`,
		`PRAGMA synchronous=NORMAL`,
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("storage: exec %q: %w", p, err)
		}
	}
	return nil
}
