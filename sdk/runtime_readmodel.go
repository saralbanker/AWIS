package sdk

// runtime_readmodel.go — the paginated instance read model (B-17).
//
// Runtime.List returns EVERY instance matching a filter, with no bound. That
// is correct for the engine's own tick loop, which genuinely needs all
// running instances, but it is the wrong shape for any consumer that renders
// a list to a person: the cost of one call grows with the size of the whole
// table, forever.
//
// Storage grew ListInstancesPaged/CountInstances, but they were reachable
// only by a caller holding a *storage.SQLiteStorage — which the SDK
// deliberately does not expose. Without the seam below, the bound existed and
// nothing could use it.
//
// The type assertion is the established additive pattern in this repo
// (cancellationStore, signalWaitStore, PluginStore, signal.Store): the
// 12-method core.StoragePort is frozen at G1, so an additive capability is
// declared as a local interface here and reached by asserting the concrete
// store against it.

import (
	"context"
	"errors"
	"fmt"

	"github.com/awis/awis/internal/core"
)

// ErrPaginationUnsupported is returned by ListPaged when the configured
// StoragePort does not implement the paginated read model.
//
// ListPaged deliberately does NOT silently fall back to an unbounded
// ListInstances and slice the result: that would read the entire table to
// return one page, quietly reintroducing the exact cost the caller asked to
// avoid, and it would do so invisibly. A caller that gets this error knows
// its storage cannot serve a bounded read; a caller that got a silent
// fallback would not.
var ErrPaginationUnsupported = errors.New("sdk: storage does not support paginated instance reads")

// pagedInstanceStore is the additive storage capability ListPaged needs.
// *storage.SQLiteStorage satisfies it.
type pagedInstanceStore interface {
	ListInstancesPaged(ctx context.Context, filter core.InstanceFilter, limit, offset int) ([]core.WorkflowInstance, error)
	CountInstances(ctx context.Context, filter core.InstanceFilter) (int, error)
}

// InstancePage is one page of instances plus the totals a caller needs to
// render pagination controls without issuing a second query.
type InstancePage struct {
	// Instances is this page's rows, in the storage layer's deterministic
	// order (started_at, then instance_id as a tiebreak).
	Instances []core.WorkflowStatus
	// Total is the number of instances matching Filter across ALL pages.
	Total int
	// Limit and Offset are the values actually applied, which may differ
	// from those requested: storage normalises a non-positive limit to its
	// default and clamps an oversized one to its maximum.
	Limit  int
	Offset int
}

// HasMore reports whether at least one further row follows this page.
func (p InstancePage) HasMore() bool {
	return p.Offset+len(p.Instances) < p.Total
}

// ListPaged returns one page of instances matching filter, together with the
// total matching count.
//
// limit <= 0 takes the storage default; an oversized limit is clamped. The
// applied values are reported back in the returned page, so a caller must
// read InstancePage.Limit rather than assuming its request was honoured
// verbatim.
//
// Ordering is stable across pages (started_at, instance_id): started_at alone
// is not unique, so without the tiebreak a row sharing a timestamp with
// another could be skipped or repeated as a caller walks the offsets.
func (r *Runtime) ListPaged(ctx context.Context, filter core.InstanceFilter, limit, offset int) (InstancePage, error) {
	ps, ok := r.storage.(pagedInstanceStore)
	if !ok {
		return InstancePage{}, fmt.Errorf("%w (%T)", ErrPaginationUnsupported, r.storage)
	}

	total, err := ps.CountInstances(ctx, filter)
	if err != nil {
		return InstancePage{}, fmt.Errorf("sdk: ListPaged count: %w", err)
	}

	instances, err := ps.ListInstancesPaged(ctx, filter, limit, offset)
	if err != nil {
		return InstancePage{}, fmt.Errorf("sdk: ListPaged: %w", err)
	}

	out := make([]core.WorkflowStatus, len(instances))
	for i, inst := range instances {
		out[i] = instanceToStatus(inst)
	}

	if offset < 0 {
		offset = 0
	}
	return InstancePage{
		Instances: out,
		Total:     total,
		Limit:     appliedLimit(limit, len(out)),
		Offset:    offset,
	}, nil
}

// storageDefaultPageSize and storageMaxPageSize mirror the bounds enforced by
// internal/storage. They are duplicated rather than exported from storage
// because sdk must not import the concrete adapter — the whole point of the
// interface assertion above. TestSDKPageBoundsMatchStorage pins them to the
// real values so the two cannot drift silently.
const (
	storageDefaultPageSize = 100
	storageMaxPageSize     = 1000
)

// appliedLimit reports the limit storage actually applied. It cannot simply
// return len(rows): a short final page would then report a smaller limit than
// was in force, which would mislead a caller computing page counts from it.
func appliedLimit(requested, _ int) int {
	switch {
	case requested <= 0:
		return storageDefaultPageSize
	case requested > storageMaxPageSize:
		return storageMaxPageSize
	default:
		return requested
	}
}
