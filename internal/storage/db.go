// Package storage implements the SQLite-backed StoragePort adapter for the AWIS
// runtime (Blueprint §20; edr-003). Only migration + DB-open concerns live in
// this card (M02-C1); StoragePort method implementations follow in M02-C2.
package storage

import (
	"database/sql"
	"fmt"
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

	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("storage: open sqlite %q: %w", path, err)
	}

	// Single-writer semantics (Blueprint §9 "no concurrent writers", local mode).
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
