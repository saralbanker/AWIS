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

func TestHandleListInstances(t *testing.T) {
	rt, store := newTestRuntime(t)
	def := linearWorkflow("wf-instances-list")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("register workflow: %v", err)
	}
	ctx := context.Background()
	id, err := rt.Submit(ctx, def.ID, nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	tickUntil(t, rt, id, core.InstanceStatusCompleted, 2*time.Second)

	router := NewRouter(rt, store, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var page instanceListResponse
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if page.Limit != 100 {
		t.Errorf("Limit = %d, want the storage-applied default 100", page.Limit)
	}
	found := false
	for _, e := range page.Instances {
		if e.InstanceID == string(id) {
			found = true
			if e.Status != string(core.InstanceStatusCompleted) {
				t.Errorf("status = %q, want completed", e.Status)
			}
		}
	}
	if !found {
		t.Fatalf("submitted instance not found in list response: %+v", page.Instances)
	}
}

// TestHandleListInstances_DefinitionIDFilter is the GUI Beta BE-1 acceptance
// criterion: filtering by definition_id returns only matching instances,
// with a correct total, not a client-side approximation.
func TestHandleListInstances_DefinitionIDFilter(t *testing.T) {
	rt, store := newTestRuntime(t)
	defA := linearWorkflow("wf-defid-a")
	defB := linearWorkflow("wf-defid-b")
	if err := rt.RegisterWorkflow(defA); err != nil {
		t.Fatalf("register wf-defid-a: %v", err)
	}
	if err := rt.RegisterWorkflow(defB); err != nil {
		t.Fatalf("register wf-defid-b: %v", err)
	}
	ctx := context.Background()
	idA, err := rt.Submit(ctx, defA.ID, nil)
	if err != nil {
		t.Fatalf("submit wf-defid-a: %v", err)
	}
	tickUntil(t, rt, idA, core.InstanceStatusCompleted, 2*time.Second)
	idB1, err := rt.Submit(ctx, defB.ID, nil)
	if err != nil {
		t.Fatalf("submit wf-defid-b #1: %v", err)
	}
	tickUntil(t, rt, idB1, core.InstanceStatusCompleted, 2*time.Second)
	idB2, err := rt.Submit(ctx, defB.ID, nil)
	if err != nil {
		t.Fatalf("submit wf-defid-b #2: %v", err)
	}
	tickUntil(t, rt, idB2, core.InstanceStatusCompleted, 2*time.Second)

	router := NewRouter(rt, store, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances?definition_id=wf-defid-b", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var page instanceListResponse
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("Total = %d, want 2 (only wf-defid-b instances)", page.Total)
	}
	for _, e := range page.Instances {
		if e.DefinitionID != "wf-defid-b" {
			t.Errorf("instance %s has definition_id=%q, want wf-defid-b (leaked from the unfiltered set)",
				e.InstanceID, e.DefinitionID)
		}
	}
}

func TestHandleGetInstance_Waiting(t *testing.T) {
	rt, store := newTestRuntime(t)
	def := waitWorkflow("wf-instance-waiting")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("register workflow: %v", err)
	}
	ctx := context.Background()
	id, err := rt.Submit(ctx, def.ID, nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	tickUntil(t, rt, id, core.InstanceStatusWaiting, 2*time.Second)

	router := NewRouter(rt, store, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/"+string(id), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var detail instanceDetail
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if detail.Status != string(core.InstanceStatusWaiting) {
		t.Fatalf("status = %q, want waiting", detail.Status)
	}
	// This is the card's exact acceptance criterion: signal_name/
	// timeout_remaining_s are non-null for a waiting instance — the gap
	// Runtime.Status leaves open on its own (GUI_ARCHITECTURE.md §6).
	if detail.SignalName == nil || *detail.SignalName != "go" {
		t.Fatalf("signal_name = %v, want \"go\"", detail.SignalName)
	}
	if detail.TimeoutRemainingS == nil {
		t.Fatal("timeout_remaining_s is nil, want a clamped countdown (5m timeout configured)")
	}
	if *detail.TimeoutRemainingS <= 0 || *detail.TimeoutRemainingS > 300 {
		t.Fatalf("timeout_remaining_s = %d, want in (0, 300]", *detail.TimeoutRemainingS)
	}
}

func TestHandleGetInstance_NotFound(t *testing.T) {
	rt, store := newTestRuntime(t)
	router := NewRouter(rt, store, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestHandleListInstances_NewestFirstOnLandingPage(t *testing.T) {
	rt, store := newTestRuntime(t)
	def := linearWorkflow("wf-landing-order")
	_ = rt.RegisterWorkflow(def)
	ctx := context.Background()

	// Submit 5 instances with increasing timestamps.
	var lastID core.InstanceID
	for i := 0; i < 5; i++ {
		id, err := rt.Submit(ctx, def.ID, nil)
		if err != nil {
			t.Fatalf("Submit %d: %v", i, err)
		}
		lastID = id
		time.Sleep(10 * time.Millisecond)
	}

	router := NewRouter(rt, store, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances?limit=50&offset=0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp struct {
		Instances []struct {
			InstanceID string `json:"instance_id"`
		} `json:"instances"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Instances) != 5 {
		t.Fatalf("got %d instances, want 5", len(resp.Instances))
	}
	// First returned instance must be the newest one (lastID)
	if resp.Instances[0].InstanceID != string(lastID) {
		t.Fatalf("expected newest instance %s at index 0, got %s", lastID, resp.Instances[0].InstanceID)
	}
}
