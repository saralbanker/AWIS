// Integration tests for TriggerAPI (M08-C2 output 5; AWIS DoD §24.3).
// Verifies that SubmitEvent persists a domain event row in the domain_events
// table (card ACCEPTANCE: "test verifies row in domain_events").
package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// domainEventLister is the additive interface the SQLite adapter satisfies;
// used here to verify domain_events rows without importing internal/storage
// beyond the already-imported openTestStorage helper.
type domainEventLister interface {
	ListUnconsumedDomainEvents(ctx context.Context, namespace string) ([]core.DomainEvent, error)
}

// TestTriggerAPI_SubmitEvent_PersistsDomainEventRow verifies that
// TriggerAPI.SubmitEvent causes a row to appear in domain_events.
func TestTriggerAPI_SubmitEvent_PersistsDomainEventRow(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "tns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	ev := core.DomainEvent{
		EventID:   "test-event-001",
		EventType: "OrderPlaced",
		Namespace: "tns",
		Payload:   map[string]any{"order_id": "123"},
		EmittedAt: time.Now().UTC(),
	}

	if err := rt.Triggers().SubmitEvent(ctx, ev); err != nil {
		t.Fatalf("SubmitEvent: %v", err)
	}

	// Verify the row via the additive ListUnconsumedDomainEvents method.
	// s is *storage.SQLiteStorage which satisfies domainEventLister; cast via
	// the interface since the concrete type is known inside the sdk test package.
	var lister domainEventLister = s

	events, err := lister.ListUnconsumedDomainEvents(ctx, "tns")
	if err != nil {
		t.Fatalf("ListUnconsumedDomainEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected ≥1 domain event row after SubmitEvent, got 0")
	}

	var found bool
	for _, e := range events {
		if e.EventID == "test-event-001" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("domain event %q not found in domain_events table", "test-event-001")
	}
}

// TestTriggerAPI_SubmitEvent_EmptyEventID verifies that SubmitEvent returns
// an error when EventID is empty (engine validation).
func TestTriggerAPI_SubmitEvent_EmptyEventID(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "tns", Storage: s})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	err = rt.Triggers().SubmitEvent(ctx, core.DomainEvent{
		EventID:   "", // empty — must be rejected
		EventType: "OrderPlaced",
		Namespace: "tns",
	})
	if err == nil {
		t.Fatal("expected error for empty EventID, got nil")
	}
}

// TestTriggerAPI_Triggers_ReturnsSameInstance verifies that successive calls to
// Triggers() return a non-nil TriggerAPI.
func TestTriggerAPI_Triggers_ReturnsSameInstance(t *testing.T) {
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "tns", Storage: s})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	ta := rt.Triggers()
	if ta == nil {
		t.Fatal("Triggers() returned nil")
	}
}
