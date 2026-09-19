package storage

// Compensating step-claim release (D-6).
//
// ClaimStep commits the step_claims INSERT and the instance version bump in
// its own transaction (sqlite.go). The StepStarted emission that must follow
// a successful claim happens separately (internal/engine/tick.go
// gatherDispatch); if that emission fails, the claim is orphaned — until now,
// with no release path short of a terminal-status UpsertInstance (sqlite.go)
// or a RebuildState wipe (rebuild.go). activatableFor never consults
// step_claims, so the step stays reported activatable forever while the claim
// can never be won again (PK conflict), wedging the instance permanently.
//
// ReleaseStepClaim gives the engine a way to undo a claim it could not
// complete. It is exposed as an ADDITIVE storage interface — NOT a change to
// StoragePort (frozen, Blueprint §20) — following the same type-asserted
// pattern as cancellationStore (internal/engine/cancel.go), RecallStore
// (recall.go), and AuditAppender (audit.go).

import (
	"context"
	"fmt"

	"github.com/awis/awis/internal/core"
)

// ReleaseStepClaim deletes the step_claims row for (instanceID, stepID), if
// present. It does not touch workflow_instances.version: the version bump
// ClaimStep made in the same transaction as the INSERT is left in place — an
// un-reverted version increment only makes a future OCC check see one extra
// generation, which is harmless (EDR-006 does not require the claim row and
// the version counter to move in lockstep on the release path).
//
// Absence of the row is not an error: a caller may race this release against
// a concurrent terminal-status UpsertInstance or RebuildState wipe, both of
// which already clear step_claims.
func (s *SQLiteStorage) ReleaseStepClaim(ctx context.Context, instanceID core.InstanceID, stepID string) error {
	if _, err := s.db.db.ExecContext(ctx,
		`DELETE FROM step_claims WHERE instance_id = ? AND step_id = ?`,
		string(instanceID), stepID,
	); err != nil {
		return fmt.Errorf("storage: ReleaseStepClaim delete: %w", err)
	}
	return nil
}
