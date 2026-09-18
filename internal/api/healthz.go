package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/awis/awis/internal/core"
)

// healthzResponse is the JSON body for GET /api/v1/healthz.
type healthzResponse struct {
	Status string `json:"status"`
}

// pinger is the additive liveness-check slice (SEC-12), mirroring the
// cancellationStore / eventsPagedStore pattern: a storage backend that can
// verify its connection is live implements it; SQLiteStorage.Ping does. A
// backend that doesn't implement it is treated as always-live (graceful
// degradation — the same fallback every other additive slice in this
// codebase uses).
type pinger interface {
	Ping(ctx context.Context) error
}

// handleHealthz implements GET /api/v1/healthz. Before SEC-12 this returned a
// static 200 with no dependency check, so an orchestrator would keep routing
// traffic to an instance whose database connection had died. It now pings
// the store (when it supports pinger) and reports 503 on failure.
func handleHealthz(store core.StoragePort) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		status := http.StatusOK
		body := healthzResponse{Status: "ok"}
		if p, ok := store.(pinger); ok {
			if err := p.Ping(r.Context()); err != nil {
				status = http.StatusServiceUnavailable
				body.Status = "unavailable"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		return json.NewEncoder(w).Encode(body)
	}
}
