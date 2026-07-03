package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// ensureSchemaVersionTable creates the migration-tracking table if it does not
// exist. The table is created outside of a migration file so the runner itself
// is always available.
func ensureSchemaVersionTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version    INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`)
	return err
}

// currentVersion returns the highest version recorded in schema_version, or 0
// if no migrations have been applied.
func currentVersion(db *sql.DB) (int, error) {
	var v int
	row := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	if err := row.Scan(&v); err != nil {
		return 0, fmt.Errorf("storage: read schema_version: %w", err)
	}
	return v, nil
}

// migrationEntry holds a parsed migration file.
type migrationEntry struct {
	version int
	name    string
	sql     string
}

// loadMigrations reads all *.sql files from the embedded FS, parses their
// version prefix (NNNN_), and returns them sorted by version.
func loadMigrations() ([]migrationEntry, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("storage: read migrations dir: %w", err)
	}

	var migrations []migrationEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		var v int
		_, err := fmt.Sscanf(e.Name(), "%04d", &v)
		if err != nil {
			return nil, fmt.Errorf("storage: parse migration filename %q: %w", e.Name(), err)
		}
		data, err := fs.ReadFile(migrationsFS, "migrations/"+e.Name())
		if err != nil {
			return nil, fmt.Errorf("storage: read migration %q: %w", e.Name(), err)
		}
		migrations = append(migrations, migrationEntry{
			version: v,
			name:    e.Name(),
			sql:     string(data),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	return migrations, nil
}

// runMigrations applies all migrations whose version is greater than the
// current schema_version. Each migration runs inside its own transaction.
// This function is idempotent: calling it on an already-migrated DB is a no-op.
func runMigrations(db *sql.DB, now func() string) error {
	if err := ensureSchemaVersionTable(db); err != nil {
		return fmt.Errorf("storage: ensure schema_version table: %w", err)
	}

	current, err := currentVersion(db)
	if err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("storage: begin migration %d tx: %w", m.version, err)
		}

		if _, err := tx.Exec(m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("storage: apply migration %d (%s): %w", m.version, m.name, err)
		}

		_, err = tx.Exec(
			`INSERT INTO schema_version (version, applied_at) VALUES (?, ?)`,
			m.version,
			now(),
		)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("storage: record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("storage: commit migration %d: %w", m.version, err)
		}
	}
	return nil
}
