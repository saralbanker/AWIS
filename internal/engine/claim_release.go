package engine

import (
	"context"

	"github.com/awis/awis/internal/core"
)

// claimReleaseStore is the additive slice of the SQLite adapter the engine
// needs to compensate for a claim it committed but could not complete (D-6).
// It is NOT part of the frozen StoragePort (Blueprint §20); the concrete
// adapter satisfies it (internal/storage/claims.go) — same pattern as
// cancellationStore (cancel.go), instanceVersionStore (emit.go), RecallStore,
// and AuditAppender.
type claimReleaseStore interface {
	ReleaseStepClaim(ctx context.Context, instanceID core.InstanceID, stepID string) error
}
