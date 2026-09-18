package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/awis/awis/internal/storage"
	"github.com/awis/awis/sdk"
)

// TestWrap_ErrorMapping is table-driven across every sentinel the
// error-mapping middleware maps (GUI_ARCHITECTURE.md §6.1), plus an
// errors.Is-wrapped variant and an unrecognized error, and verifies both the
// status code and the JSON errorEnvelope body via httptest.
func TestWrap_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"instance not found", storage.ErrInstanceNotFound, http.StatusNotFound},
		{"plugin not found", storage.ErrPluginNotFound, http.StatusNotFound},
		{"capability not found", storage.ErrCapabilityNotFound, http.StatusNotFound},
		{"workflow not found", storage.ErrWorkflowNotFound, http.StatusNotFound},
		{"wrapped instance not found", fmt.Errorf("%w: abc123", storage.ErrInstanceNotFound), http.StatusNotFound},
		{"duplicate domain event", storage.ErrDuplicateDomainEvent, http.StatusConflict},
		{"version conflict", storage.ErrVersionConflict, http.StatusConflict},
		{"pagination unsupported", sdk.ErrPaginationUnsupported, http.StatusNotImplemented},
		{"unmapped error", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := wrap(func(w http.ResponseWriter, r *http.Request) error {
				return tt.err
			})

			req := httptest.NewRequest(http.MethodGet, "/whatever", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}

			var body errorEnvelope
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode response body: %v", err)
			}
			if body.Error != tt.err.Error() {
				t.Fatalf("error field = %q, want %q", body.Error, tt.err.Error())
			}
		})
	}
}

// TestWrap_NilError verifies that a handler which writes its own success
// response and returns nil is left alone — wrap only intervenes on error.
func TestWrap_NilError(t *testing.T) {
	called := false
	handler := wrap(func(w http.ResponseWriter, r *http.Request) error {
		called = true
		w.WriteHeader(http.StatusOK)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/whatever", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if !called {
		t.Fatal("handler was not invoked")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
