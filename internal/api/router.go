// Package api is the HTTP surface for the AWIS GUI backend. It is built
// inside this module (not as a separate SDK-facing package) specifically so
// handlers can reach types that `internal/storage` and `sdk` do not expose
// across a module boundary — see docs/09-gui-planning/GUI_ARCHITECTURE.md §7
// (ADR-3).
//
// Routing uses stdlib net/http.ServeMux with Go's method+path patterns
// (go.mod pins go1.26, well past the 1.22 floor those patterns require).
// No third-party router is used or ever should be added — this repo has
// exactly two direct dependencies (yaml.v3, modernc.org/sqlite) and that is
// deliberate (docs/PROVIDERS.md; cmd/awis/main.go's stdlib-only precedent).
package api

import (
	"net/http"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/sdk"
)

// NewRouter builds the internal/api HTTP handler and mounts every /api/v1
// route this package currently owns. A separate binary (cmd/awis-server)
// mounts the returned http.Handler directly.
//
// rt is the engine-facing surface (instance list/detail); store is the
// definition-facing surface (workflow list/get, paginated event history).
// startedAt is when the server process started, for GET /api/v1/info's
// uptime (GUI Beta, BE-2b) — all three are required.
func NewRouter(rt *sdk.Runtime, store core.StoragePort, startedAt time.Time) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/healthz", wrap(handleHealthz(store)))
	mux.HandleFunc("GET /api/v1/info", wrap(handleInfo(startedAt)))

	mux.HandleFunc("GET /api/v1/workflows", wrap(handleListWorkflows(store)))
	mux.HandleFunc("GET /api/v1/workflows/{id}/{version}", wrap(handleGetWorkflow(store)))

	mux.HandleFunc("GET /api/v1/instances", wrap(handleListInstances(rt)))
	mux.HandleFunc("GET /api/v1/instances/{id}", wrap(handleGetInstance(rt, store)))
	mux.HandleFunc("GET /api/v1/instances/{id}/events", wrap(handleListEvents(store)))

	return mux
}
