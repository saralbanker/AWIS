package storage_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/awis/awis/internal/storage"
)

func newPluginTestStorage(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	s, db := openTestStorage(t, filepath.Join(t.TempDir(), "plugins.db"))
	t.Cleanup(func() { _ = db.Close() })
	return s
}

// registerTestPlugin installs a plugin owning one capability.
func registerTestPlugin(t *testing.T, s *storage.SQLiteStorage, name, capability string) {
	t.Helper()
	if err := s.RegisterPlugin(context.Background(), name, "1.0.0",
		`{"name":"`+name+`"}`, []string{capability}); err != nil {
		t.Fatalf("RegisterPlugin(%s): %v", name, err)
	}
}

// TestRemovePluginStopsCapabilityRouting is the B-20 regression.
//
// `plugin remove` set plugins.status to "removed" and stopped there, while
// LookupCapability selected from plugin_capabilities with no reference to
// status — so a removed plugin still owned its capabilities and a workflow
// step naming one was still dispatched to it. Removal was cosmetic.
func TestRemovePluginStopsCapabilityRouting(t *testing.T) {
	s := newPluginTestStorage(t)
	ctx := context.Background()
	registerTestPlugin(t, s, "p1", "cap.one")

	if got, err := s.LookupCapability(ctx, "cap.one"); err != nil || got != "p1" {
		t.Fatalf("before removal: LookupCapability = %q, %v; want p1, nil", got, err)
	}

	if err := s.RemovePlugin(ctx, "p1"); err != nil {
		t.Fatalf("RemovePlugin: %v", err)
	}

	if _, err := s.LookupCapability(ctx, "cap.one"); !errors.Is(err, storage.ErrCapabilityNotFound) {
		t.Fatalf("after removal: LookupCapability err = %v, want ErrCapabilityNotFound — "+
			"a removed plugin is still being routed to", err)
	}
}

// TestLookupCapabilityExcludesRemovedRegardlessOfHowStatusWasSet is why the
// exclusion lives in LookupCapability and not only in the remove path: a
// plugin whose status reaches "removed" by ANY route must stop being routed
// to, even if its capability rows are still present.
func TestLookupCapabilityExcludesRemovedRegardlessOfHowStatusWasSet(t *testing.T) {
	s := newPluginTestStorage(t)
	ctx := context.Background()
	registerTestPlugin(t, s, "p2", "cap.two")

	// Status set directly, leaving the capability rows in place.
	if err := s.SetPluginStatus(ctx, "p2", "removed"); err != nil {
		t.Fatalf("SetPluginStatus: %v", err)
	}

	if _, err := s.LookupCapability(ctx, "cap.two"); !errors.Is(err, storage.ErrCapabilityNotFound) {
		t.Errorf("LookupCapability err = %v, want ErrCapabilityNotFound; the routing "+
			"exclusion is not at the decision point", err)
	}
}

// TestRemovePluginLeavesOtherPluginsRoutable: removal must be surgical.
func TestRemovePluginLeavesOtherPluginsRoutable(t *testing.T) {
	s := newPluginTestStorage(t)
	ctx := context.Background()
	registerTestPlugin(t, s, "keep", "cap.keep")
	registerTestPlugin(t, s, "drop", "cap.drop")

	if err := s.RemovePlugin(ctx, "drop"); err != nil {
		t.Fatalf("RemovePlugin: %v", err)
	}

	if got, err := s.LookupCapability(ctx, "cap.keep"); err != nil || got != "keep" {
		t.Errorf("unrelated plugin: LookupCapability = %q, %v; want keep, nil", got, err)
	}
}

// TestRemovePluginIsIdempotent: re-running a cleanup command must not error.
func TestRemovePluginIsIdempotent(t *testing.T) {
	s := newPluginTestStorage(t)
	ctx := context.Background()
	registerTestPlugin(t, s, "p3", "cap.three")

	for i := 1; i <= 2; i++ {
		if err := s.RemovePlugin(ctx, "p3"); err != nil {
			t.Fatalf("RemovePlugin pass %d: %v", i, err)
		}
	}
}

// TestRemovePluginUnknownName returns the typed not-found error rather than
// silently succeeding, so a typo is reported instead of looking like a
// successful removal.
func TestRemovePluginUnknownName(t *testing.T) {
	s := newPluginTestStorage(t)
	if err := s.RemovePlugin(context.Background(), "never-installed"); !errors.Is(err, storage.ErrPluginNotFound) {
		t.Fatalf("err = %v, want ErrPluginNotFound", err)
	}
}

// TestRemovePluginRetainsThePluginRow: V1 removal is a SOFT delete — the row
// is what lets `plugin list` and the audit trail still account for a plugin
// that was once installed.
func TestRemovePluginRetainsThePluginRow(t *testing.T) {
	s := newPluginTestStorage(t)
	ctx := context.Background()
	registerTestPlugin(t, s, "p4", "cap.four")

	if err := s.RemovePlugin(ctx, "p4"); err != nil {
		t.Fatalf("RemovePlugin: %v", err)
	}
	row, err := s.GetPlugin(ctx, "p4")
	if err != nil {
		t.Fatalf("GetPlugin after removal: %v — the soft-delete row was dropped", err)
	}
	if row.Status != "removed" {
		t.Errorf("status = %q, want %q", row.Status, "removed")
	}
}
