package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
	"github.com/awis/awis/sdk"
)

// instanceEntry is the response-DTO for one row of GET /api/v1/instances and
// the base of GET /api/v1/instances/{id}. Field names/casing follow this
// codebase's established JSON convention (snake_case — see
// core.ExecutionEvent, core.WorkflowInstance) rather than marshaling
// core.WorkflowStatus's Go-cased fields directly, which carry no json tags.
type instanceEntry struct {
	InstanceID   string   `json:"instance_id"`
	DefinitionID string   `json:"definition_id"`
	Version      string   `json:"version"`
	Status       string   `json:"status"`
	CurrentSteps []string `json:"current_steps"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// instanceListResponse is GET /api/v1/instances' response body. Total/Limit/
// Offset are InstancePage's own applied values (sdk/runtime_readmodel.go),
// not simply echoed request params — storage normalises a non-positive or
// oversized limit, and the caller needs to know what was actually applied.
type instanceListResponse struct {
	Instances []instanceEntry `json:"instances"`
	Total     int             `json:"total"`
	Limit     int             `json:"limit"`
	Offset    int             `json:"offset"`
}

// instanceDetail is GET /api/v1/instances/{id}'s response body: an
// instanceEntry plus Inputs/Outputs and the wait-record wrapper fields
// (SignalName, TimeoutRemainingS) that Runtime.Status omits today — the
// exact gap GUI_ARCHITECTURE.md §6 names and this card (E-G4-5) closes.
type instanceDetail struct {
	instanceEntry
	Inputs            map[string]any `json:"inputs,omitempty"`
	Outputs           map[string]any `json:"outputs,omitempty"`
	SignalName        *string        `json:"signal_name"`
	TimeoutRemainingS *int           `json:"timeout_remaining_s"`
}

func toInstanceEntry(s core.WorkflowStatus) instanceEntry {
	return instanceEntry{
		InstanceID:   string(s.InstanceID),
		DefinitionID: s.DefinitionID,
		Version:      string(s.Version),
		Status:       string(s.Status),
		CurrentSteps: s.CurrentSteps,
		CreatedAt:    s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// handleListInstances implements GET /api/v1/instances, backed by
// Runtime.ListPaged — already a real cursor (100 default / 1000 max,
// B-17), so this handler does no clamping of its own: a non-positive or
// oversized ?limit= is passed straight through and normalised by the
// storage layer, per InstancePage's own documented contract.
//
// Query params: ?namespace=, ?status=, ?definition_id=, ?limit=, ?offset=. All
// optional; omitted namespace/status/definition_id apply no predicate (GUI
// Beta, BE-1).
func handleListInstances(rt *sdk.Runtime) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.URL.Query()
		filter := core.InstanceFilter{
			Namespace:    q.Get("namespace"),
			Status:       core.InstanceStatus(q.Get("status")),
			DefinitionID: q.Get("definition_id"),
		}
		limit, _ := strconv.Atoi(q.Get("limit"))
		offset, _ := strconv.Atoi(q.Get("offset"))

		page, err := rt.ListPaged(r.Context(), filter, limit, offset)
		if err != nil {
			return err
		}

		entries := make([]instanceEntry, 0, len(page.Instances))
		for _, s := range page.Instances {
			entries = append(entries, toInstanceEntry(s))
		}
		return writeJSON(w, http.StatusOK, instanceListResponse{
			Instances: entries,
			Total:     page.Total,
			Limit:     page.Limit,
			Offset:    page.Offset,
		})
	}
}

// waitRecordStore is the additive storage capability needed to describe a
// wait — declared locally and type-asserted, the same pattern
// cmd/awis/status.go's own waitRecordStore establishes for reaching
// *storage.SQLiteStorage methods that are not on the frozen 12-method
// core.StoragePort.
type waitRecordStore interface {
	ListWaitRecordsByInstance(ctx context.Context, instanceID core.InstanceID) ([]storage.WaitRecord, error)
}

// lookupWait returns the wait record describing why an instance in status
// InstanceStatusWaiting is parked, or nil — ported from
// cmd/awis/status.go's lookupWait/buildActiveJSON (lines 264-348), adapted
// from core.WorkflowInstance to core.WorkflowStatus (the type
// Runtime.Status/ListPaged actually return; both carry the InstanceID,
// Status, and CurrentSteps fields this needs). A store without the
// capability yields nil, degrading to omitted wait info rather than failing
// the request.
func lookupWait(ctx context.Context, store core.StoragePort, s core.WorkflowStatus) *storage.WaitRecord {
	if s.Status != core.InstanceStatusWaiting {
		return nil
	}
	ws, ok := store.(waitRecordStore)
	if !ok {
		return nil
	}
	records, err := ws.ListWaitRecordsByInstance(ctx, s.InstanceID)
	if err != nil || len(records) == 0 {
		return nil
	}
	// An instance can hold several wait records (a parallel join of signal
	// steps). Prefer the one for the step CurrentSteps already reports as
	// current, so signal_name and current_steps describe the same wait;
	// otherwise fall back to the first (ListWaitRecordsByInstance orders by
	// step_id for determinism).
	if len(s.CurrentSteps) > 0 {
		for i := range records {
			if records[i].StepID == s.CurrentSteps[0] {
				return &records[i]
			}
		}
	}
	return &records[0]
}

// handleGetInstance implements GET /api/v1/instances/{id}, backed by
// Runtime.Status plus the wait-record wrapper above. A missing instance
// surfaces storage.ErrInstanceNotFound through wrap()'s error-mapping
// middleware as a 404 (Runtime.Status %w-wraps GetInstance's sentinel,
// sdk/runtime_runner.go).
func handleGetInstance(rt *sdk.Runtime, store core.StoragePort) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := core.InstanceID(r.PathValue("id"))

		status, err := rt.Status(r.Context(), id)
		if err != nil {
			return err
		}

		detail := instanceDetail{
			instanceEntry: toInstanceEntry(status),
			Inputs:        status.Inputs,
			Outputs:       status.Outputs,
		}

		if wait := lookupWait(r.Context(), store, status); wait != nil {
			name := wait.SignalName
			detail.SignalName = &name
			if wait.TimeoutAt != nil {
				// Report a clamped, whole-second countdown — a negative
				// value would mean the wait is already due and the timeout
				// scan simply hasn't run yet, which reads as a bug rather
				// than a real state (mirrors cmd/awis/status.go's identical
				// clamp).
				now := time.Now()
				remaining := int(wait.TimeoutAt.Sub(now).Seconds())
				if remaining < 0 {
					remaining = 0
				}
				detail.TimeoutRemainingS = &remaining
			}
		}

		return writeJSON(w, http.StatusOK, detail)
	}
}
