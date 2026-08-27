package storage

// plugin_remove.go — B-20: make `plugin remove` mean something.
//
// `awis plugin remove <name>` set plugins.status to "removed" and stopped
// there. The plugin_capabilities rows stayed behind, and LookupCapability
// selected from that table with no reference to plugins.status — so a removed
// plugin still owned its capabilities and a workflow step naming one was
// still dispatched to it. Removal was purely cosmetic.
//
// The fix has two halves, deliberately:
//
//  1. LookupCapability (plugins.go) now excludes removed plugins. This is the
//     correctness half, and it is the one that matters: it fixes routing at
//     the point where the decision is made, so a plugin whose status reaches
//     "removed" by ANY path — this command, a future one, a manual repair —
//     is excluded. A fix that only deleted rows in the remove command would
//     leave every other path broken.
//  2. RemovePlugin below also deletes the capability rows, in the same
//     transaction as the status update, so removal does not leave phantom
//     capability rows behind for inspection and re-registration to trip over.
//
// The plugins row itself is retained: V1 removal is a soft delete, and the
// row is what lets `plugin list` and the audit trail still account for a
// plugin that was once installed.

import (
	"context"
	"fmt"
)

// RemovePlugin marks a plugin removed and drops its capability rows in one
// transaction (B-20).
//
// Doing both under one transaction matters: between a status update and a
// separate capability delete, a concurrent capability lookup could observe a
// plugin that is already "removed" but still owns its capabilities — exactly
// the inconsistent state this defect is about, just narrowed to a window.
//
// Returns ErrPluginNotFound when no plugins row matches name. Removing an
// already-removed plugin is not an error: the operation is idempotent, which
// is what an operator re-running a cleanup command expects.
func (s *SQLiteStorage) RemovePlugin(ctx context.Context, name string) error {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage: RemovePlugin begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`UPDATE plugins SET status = 'removed' WHERE plugin_id = ?`, name)
	if err != nil {
		return fmt.Errorf("storage: RemovePlugin update status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: RemovePlugin rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", ErrPluginNotFound, name)
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM plugin_capabilities WHERE plugin_id = ?`, name); err != nil {
		return fmt.Errorf("storage: RemovePlugin delete capabilities: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage: RemovePlugin commit: %w", err)
	}
	return nil
}
