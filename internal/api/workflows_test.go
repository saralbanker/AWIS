package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

func TestHandleListWorkflows(t *testing.T) {
	rt, store := newTestRuntime(t)
	def := linearWorkflow("wf-list-test")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("register workflow: %v", err)
	}

	router := NewRouter(rt, store, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var entries []workflowEntry
	if err := json.NewDecoder(rec.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.ID == "wf-list-test" && e.Version == "1.0.0" && e.Namespace == "default" && e.StepCount == 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("registered workflow not found in list response: %+v", entries)
	}
}

func TestHandleGetWorkflow(t *testing.T) {
	rt, store := newTestRuntime(t)
	def := linearWorkflow("wf-get-test")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("register workflow: %v", err)
	}

	router := NewRouter(rt, store, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/wf-get-test/1.0.0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got core.WorkflowDefinition
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != "wf-get-test" || got.Version != "1.0.0" {
		t.Fatalf("got id=%s version=%s, want wf-get-test/1.0.0", got.ID, got.Version)
	}
}

// TestHandleGetWorkflow_NotFound is the card's exact acceptance criterion:
// a missing (id, version) pair returns 404 via storage.ErrWorkflowNotFound
// (E-G1-3), not a 500 or an empty 200.
func TestHandleGetWorkflow_NotFound(t *testing.T) {
	rt, store := newTestRuntime(t)
	router := NewRouter(rt, store, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/does-not-exist/9.9.9", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	var envelope errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error == "" {
		t.Fatal("error envelope has empty message")
	}
}

func TestHandleListWorkflows_NamespaceFilter(t *testing.T) {
	rt, store := newTestRuntime(t)
	if err := rt.RegisterWorkflow(linearWorkflow("wf-ns-default")); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Register a second workflow directly into a different namespace via
	// storage (the builder's SetNamespace + RegisterWorkflow path only
	// covers the runtime's own namespace in this test's other cases, so
	// register this one straight through StoragePort to get a genuinely
	// different namespace value).
	other := linearWorkflow("wf-ns-other")
	other.Namespace = "other"
	if err := store.RegisterWorkflow(context.Background(), *other); err != nil {
		t.Fatalf("register other-namespace workflow: %v", err)
	}

	router := NewRouter(rt, store, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?namespace=other", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var entries []workflowEntry
	if err := json.NewDecoder(rec.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, e := range entries {
		if e.ID == "wf-ns-default" {
			t.Fatalf("namespace filter leaked a default-namespace workflow: %+v", entries)
		}
	}
}
