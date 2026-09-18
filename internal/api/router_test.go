package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// TestNewRouter_Healthz verifies the acceptance criterion for E-G4-3: the
// router mounts with GET /api/v1/healthz returning 200 regardless of what
// else is registered.
func TestNewRouter_Healthz(t *testing.T) {
	rt, store := newTestRuntime(t)
	router := NewRouter(rt, store, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body healthzResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status field = %q, want %q", body.Status, "ok")
	}
}

// deadPingStore wraps a real StoragePort but fails Ping, simulating a dead
// database connection (SEC-12).
type deadPingStore struct {
	core.StoragePort
}

func (deadPingStore) Ping(ctx context.Context) error {
	return errors.New("simulated connection failure")
}

// TestNewRouter_Healthz_PingFailureReturns503 verifies SEC-12's fix: a store
// whose Ping fails must surface as 503, not the old static 200.
func TestNewRouter_Healthz_PingFailureReturns503(t *testing.T) {
	rt, store := newTestRuntime(t)
	router := NewRouter(rt, deadPingStore{StoragePort: store}, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	var body healthzResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Status != "unavailable" {
		t.Fatalf("status field = %q, want %q", body.Status, "unavailable")
	}
}

// TestNewRouter_UnknownRoute verifies the router does not silently accept
// arbitrary paths — it should 404 through ServeMux's own not-found handling
// for anything outside the route table this card defines.
func TestNewRouter_UnknownRoute(t *testing.T) {
	rt, store := newTestRuntime(t)
	router := NewRouter(rt, store, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
