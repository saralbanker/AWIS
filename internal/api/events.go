package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
	"github.com/awis/awis/sdk"
)

// eventsPagedStore is the additive storage capability GET
// /api/v1/instances/{id}/events needs — declared locally and type-asserted,
// the same pattern waitRecordStore (instances.go) uses to reach a
// *storage.SQLiteStorage method that is not on the frozen 12-method
// core.StoragePort. ReadEventsPaged (internal/storage/sqlite.go) is the
// bounded sibling of the frozen ReadEvents.
type eventsPagedStore interface {
	ReadEventsPaged(ctx context.Context, instanceID core.InstanceID, fromSeq, limit int) ([]core.ExecutionEvent, error)
}

// eventsPageResponse is GET /api/v1/instances/{id}/events' response body.
// NextCursor is the fromSeq value a client should pass to fetch the next
// page; it is present only when the page came back full (== the applied
// limit), since a short page means the result set is exhausted.
type eventsPageResponse struct {
	Events     []core.ExecutionEvent `json:"events"`
	NextCursor *int                  `json:"next_cursor,omitempty"`
}

// handleListEvents implements GET /api/v1/instances/{id}/events, backed by
// ReadEventsPaged (E-G4-6) rather than the frozen, unbounded ReadEvents — a
// single instance with 100k events must not return them all in one query to
// render one timeline (GUI_ARCHITECTURE.md §4.4).
//
// Query params: ?from= (fromSeq cursor, default 0), ?limit= (passed through
// to ReadEventsPaged, which normalises/clamps it — this handler does no
// clamping of its own, matching handleListInstances' convention).
//
// If store doesn't implement eventsPagedStore, this returns
// sdk.ErrPaginationUnsupported, which wrap()'s error-mapping middleware maps
// to 501 — the correct signal that this deployment's storage backend cannot
// serve a bounded read, rather than silently falling back to an unbounded
// one.
func handleListEvents(store core.StoragePort) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		ps, ok := store.(eventsPagedStore)
		if !ok {
			return sdk.ErrPaginationUnsupported
		}

		id := core.InstanceID(r.PathValue("id"))

		// ReadEventsPaged returns an empty (not error) slice for an unknown
		// instance_id — same as ReadEvents, a correct property of an
		// unindexed WHERE match. Left unchecked, that makes "instance
		// doesn't exist" and "instance exists with zero events so far"
		// indistinguishable at 200 OK. Check existence first via the frozen
		// GetInstance, mirroring the 404 pattern handleGetWorkflow/
		// handleGetInstance already establish.
		if _, err := store.GetInstance(r.Context(), id); err != nil {
			return err
		}

		q := r.URL.Query()
		fromSeq, _ := strconv.Atoi(q.Get("from"))
		limit, _ := strconv.Atoi(q.Get("limit"))

		events, err := ps.ReadEventsPaged(r.Context(), id, fromSeq, limit)
		if err != nil {
			return err
		}

		resp := eventsPageResponse{Events: events}
		// Mirror ReadEventsPaged's own normalisation exactly (via the
		// exported constants, not a duplicated literal) so "was this page
		// full" is judged against the limit that was actually applied.
		applied := limit
		if applied <= 0 {
			applied = storage.DefaultEventsPageSize
		}
		if applied > storage.MaxEventsPageSize {
			applied = storage.MaxEventsPageSize
		}
		if len(events) == applied {
			next := events[len(events)-1].SequenceNum + 1
			resp.NextCursor = &next
		}
		return writeJSON(w, http.StatusOK, resp)
	}
}
