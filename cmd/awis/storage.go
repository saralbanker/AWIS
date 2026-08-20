package main

import (
	"fmt"
	"os"

	"github.com/awis/awis/sdk"
)

// OpenStorage opens (or creates) the AWIS SQLite storage at
// <dataDir>/runtime.db, creating dataDir with os.MkdirAll if necessary.
//
// WAL mode and busy_timeout are applied by sdk.SQLiteStorage → internal/storage.Open
// (see internal/storage/db.go applyPragmas: PRAGMA journal_mode=WAL,
// PRAGMA busy_timeout=5000). No additional PRAGMA is required here.
func OpenStorage(dataDir string) (sdk.StoragePort, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("open storage: mkdir %q: %w", dataDir, err)
	}
	dbPath := dataDir + "/runtime.db"
	store, err := sdk.SQLiteStorage(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}
	return store, nil
}
