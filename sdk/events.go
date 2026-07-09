package sdk

import (
	"context"

	"github.com/awis/awis/internal/core"
)

// ReadEvents returns all execution events for id, starting from sequence 0.
// Primarily used by sdk/testing to read raw event streams for determinism proofs.
func (r *Runtime) ReadEvents(ctx context.Context, id core.InstanceID) ([]core.ExecutionEvent, error) {
	return r.storage.ReadEvents(ctx, id, 0)
}
