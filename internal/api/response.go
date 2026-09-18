package api

import (
	"encoding/json"
	"net/http"
)

// writeJSON writes v as a JSON body with the given status code. It is the
// shared success-response helper every handler beyond healthz.go uses, so
// the Content-Type/WriteHeader/Encode triplet isn't repeated per handler.
func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
