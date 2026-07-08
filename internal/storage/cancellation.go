package storage

// Cancellation flag accessors (M06-C2, Finalization B4).
//
// StoragePort's 12 methods are frozen (Blueprint §20). Cancellation needs a
// per-instance request flag that the engine sets and reads; per the M06 spec
// (T6) that flag lives in the pre-existing workflow_instances.cancellation_requested
// column (present since migration 0001) and is exposed via ADDITIVE
// *SQLiteStorage methods — NOT new StoragePort methods. The engine reaches them
// through an internal interface it type-asserts (same pattern as the M03
// RebuildState library method and the T5 trigger store).
//
// The column is DB-internal projection state, not evented: RebuildState always
// resets it to 0 (rebuild.go), so a crash mid-cancellation loses the request and
// the caller re-issues Cancel (EDR-011 §8 documents this V1 limitation).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/awis/awis/internal/core"
)

// SetCancellationRequested sets workflow_instances.cancellation_requested=1 for
// instanceID (Finalization B4 step 1). It does NOT touch status or version:
// cancellation is a request flag consumed by the engine's tick, never a status
// (CONTRA-6 — claims/flags never a status). Returns ErrInstanceNotFound when no
// row exists.
func (s *SQLiteStorage) SetCancellationRequested(ctx context.Context, instanceID core.InstanceID) error {
	res, err := s.db.db.ExecContext(ctx,
		`UPDATE workflow_instances SET cancellation_requested = 1 WHERE instance_id = ?`,
		string(instanceID),
	)
	if err != nil {
		return fmt.Errorf("storage: SetCancellationRequested update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: SetCancellationRequested rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", ErrInstanceNotFound, instanceID)
	}
	return nil
}

// CancellationRequested reports whether cancellation has been requested for
// instanceID. Returns ErrInstanceNotFound when no row exists.
func (s *SQLiteStorage) CancellationRequested(ctx context.Context, instanceID core.InstanceID) (bool, error) {
	var flag int
	err := s.db.db.QueryRowContext(ctx,
		`SELECT cancellation_requested FROM workflow_instances WHERE instance_id = ?`,
		string(instanceID),
	).Scan(&flag)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%w: %s", ErrInstanceNotFound, instanceID)
		}
		return false, fmt.Errorf("storage: CancellationRequested query: %w", err)
	}
	return flag != 0, nil
}
