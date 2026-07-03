package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
	"github.com/awis/awis/internal/storage/storagetest"
)

// TestSQLiteStorageContract runs the adapter-agnostic StoragePort contract
// suite against the SQLite adapter.
func TestSQLiteStorageContract(t *testing.T) {
	storagetest.Run(t, sqliteFactory)
}

// sqliteFactory opens a fresh SQLite DB in a temp dir and returns a
// *storagetest.FakeClockStorage so the TTL subtest can advance time.
func sqliteFactory(t *testing.T) core.StoragePort {
	t.Helper()
	path := filepath.Join(t.TempDir(), "contract.db")

	// Shared clock pointer — both NewSQLiteStorage and FakeClockStorage read it.
	clockPtr := new(time.Time)
	*clockPtr = time.Now()
	clock := func() time.Time { return *clockPtr }

	db, err := storage.Open(path, clock)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	inner := storage.NewSQLiteStorage(db, clock)
	return storagetest.WrapWithFakeClock(inner, clockPtr)
}
