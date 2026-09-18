package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/awis/awis/internal/storage"
	"github.com/awis/awis/sdk"
)

// HandlerFunc is the convention every internal/api route handler follows:
// do the work, write a success body on the happy path, and return an error
// on failure instead of writing an error body directly. wrap() is the
// sentinel-to-HTTP-status error-mapping middleware that turns a returned
// error into the correct status code and a JSON error envelope — handlers
// never choose a status code for a failure themselves.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// errorEnvelope is the response-body convention for every failed request:
// a single JSON object with an "error" field holding a human-readable
// message. This is the DTO/response-envelope convention this card
// establishes for error responses; success responses are the resource's own
// JSON shape, unwrapped (see healthz.go).
type errorEnvelope struct {
	Error string `json:"error"`
}

// wrap adapts a HandlerFunc into an http.HandlerFunc. It is the
// error-mapping middleware: any error the handler returns is mapped by
// statusFor to an HTTP status code and written as an errorEnvelope. A nil
// error means the handler already wrote its own success response.
func wrap(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}
		writeError(w, err)
	}
}

// writeError maps err to an HTTP status via statusFor and writes it as a
// JSON errorEnvelope.
func writeError(w http.ResponseWriter, err error) {
	status := statusFor(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: err.Error()})
}

// statusFor maps a known sentinel error to its HTTP status code, per
// GUI_ARCHITECTURE.md §6.1: ErrInstanceNotFound, ErrPluginNotFound,
// ErrCapabilityNotFound, and ErrWorkflowNotFound (all internal/storage) are
// 404; ErrDuplicateDomainEvent and ErrVersionConflict (both internal/storage)
// are 409; ErrPaginationUnsupported (sdk) is 501. Anything else — including
// an unwrapped error — maps to 500.
//
// This switch is deliberately a flat, additive list so it stays trivially
// extensible: a new sentinel is one more errors.Is case in the appropriate
// bucket, not a redesign.
func statusFor(err error) int {
	switch {
	case errors.Is(err, storage.ErrInstanceNotFound),
		errors.Is(err, storage.ErrPluginNotFound),
		errors.Is(err, storage.ErrCapabilityNotFound),
		errors.Is(err, storage.ErrWorkflowNotFound):
		return http.StatusNotFound

	case errors.Is(err, storage.ErrDuplicateDomainEvent),
		errors.Is(err, storage.ErrVersionConflict):
		return http.StatusConflict

	case errors.Is(err, sdk.ErrPaginationUnsupported):
		return http.StatusNotImplemented

	default:
		return http.StatusInternalServerError
	}
}
