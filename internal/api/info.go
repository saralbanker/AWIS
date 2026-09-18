package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/awis/awis/internal/buildinfo"
)

// infoResponse is the JSON body for GET /api/v1/info (GUI Beta, BE-2b) — the
// GUI's answer to "what am I running, and how long has it been up," which
// /healthz deliberately does not carry (a pure liveness check stays pure).
type infoResponse struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	UptimeS   int    `json:"uptime_s"`
}

// handleInfo implements GET /api/v1/info. GoVersion mirrors `awis version`'s
// own runtime.Version() call (cmd/awis/main.go) so the CLI and the GUI give
// the same answer to "what am I running."
func handleInfo(startedAt time.Time) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		return writeJSON(w, http.StatusOK, infoResponse{
			Version:   buildinfo.Version,
			GoVersion: runtime.Version(),
			UptimeS:   int(time.Since(startedAt).Seconds()),
		})
	}
}
