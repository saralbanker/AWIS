package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

func TestHandleListEvents(t *testing.T) {
	rt, store := newTestRuntime(t)
	def := linearWorkflow("wf-events-list")
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/"+string(id)+"/events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var page eventsPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(page.Events) == 0 {
		t.Fatal("expected at least WorkflowStarted/StepStarted/StepCompleted/WorkflowCompleted, got zero events")
	}
	for i := 1; i < len(page.Events); i++ {
		if page.Events[i].SequenceNum <= page.Events[i-1].SequenceNum {
			t.Fatalf("events not in ascending sequence_num order: [%d]=%d, [%d]=%d",
				i-1, page.Events[i-1].SequenceNum, i, page.Events[i].SequenceNum)
		}
	}
	if page.NextCursor != nil {
		t.Fatalf("next_cursor = %v, want nil (page shorter than any applied limit)", *page.NextCursor)
	}
}

// TestHandleListEvents_Pagination is the card's exact acceptance criterion:
// a bounded page, with a correct next-page cursor, not the full event set
// in one query.
func TestHandleListEvents_Pagination(t *testing.T) {
	rt, store := newTestRuntime(t)
	def := linearWorkflow("wf-events-page")
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

	// First page, limit=2.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/"+string(id)+"/events?limit=2", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("page 1 status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var page1 eventsPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&page1); err != nil {
		t.Fatalf("decode page 1: %v", err)
	}
	if len(page1.Events) != 2 {
		t.Fatalf("page 1 len = %d, want 2 (this workflow produces at least 4 events)", len(page1.Events))
	}
	if page1.NextCursor == nil {
		t.Fatal("page 1 next_cursor is nil, want a cursor (a full page implies more may exist)")
	}

	// Second page, using the returned cursor.
	req2 := httptest.NewRequest(http.MethodGet,
		"/api/v1/instances/"+string(id)+"/events?limit=2&from="+strconv.Itoa(*page1.NextCursor), nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("page 2 status = %d, body=%s", rec2.Code, rec2.Body.String())
	}
	var page2 eventsPageResponse
	if err := json.NewDecoder(rec2.Body).Decode(&page2); err != nil {
		t.Fatalf("decode page 2: %v", err)
	}
	if len(page2.Events) == 0 {
		t.Fatal("page 2 is empty; pagination lost events between pages")
	}
	if page2.Events[0].SequenceNum != *page1.NextCursor {
		t.Fatalf("page 2 first event sequence_num = %d, want %d (no gap/duplicate)",
			page2.Events[0].SequenceNum, *page1.NextCursor)
	}
}

// TestHandleListEvents_NotFound verifies that a nonexistent instance_id
// returns 404 rather than 200 with an empty/null events list — ReadEvents/
// ReadEventsPaged both return an empty slice without error for an unknown
// instance (a correct property of their WHERE clause), which would
// otherwise make "instance doesn't exist" indistinguishable from "instance
// exists with zero events so far."
func TestHandleListEvents_NotFound(t *testing.T) {
	rt, store := newTestRuntime(t)
	router := NewRouter(rt, store, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/does-not-exist/events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
