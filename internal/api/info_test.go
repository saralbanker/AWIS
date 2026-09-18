package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/awis/awis/internal/buildinfo"
)

// TestHandleInfo is the GUI Beta BE-2b acceptance criterion: GET
// /api/v1/info returns a version matching buildinfo.Version (the same
// source `awis version` reads) and an uptime that reflects elapsed time
// since the process started.
func TestHandleInfo(t *testing.T) {
	rt, store := newTestRuntime(t)
	startedAt := time.Now().Add(-5 * time.Second)
	router := NewRouter(rt, store, startedAt)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var info infoResponse
	if err := json.NewDecoder(rec.Body).Decode(&info); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if info.Version != buildinfo.Version {
		t.Errorf("version = %q, want %q", info.Version, buildinfo.Version)
	}
	if info.GoVersion == "" {
		t.Error("go_version is empty")
	}
	if info.UptimeS < 5 {
		t.Errorf("uptime_s = %d, want >= 5 (startedAt was 5s in the past)", info.UptimeS)
	}
}
