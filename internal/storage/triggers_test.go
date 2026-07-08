package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// newTriggerStore opens a fresh DB with a fixed clock and returns the adapter.
func newTriggerStore(t *testing.T) *SQLiteStorage {
	t.Helper()
	db := openTestDB(t)
	return NewSQLiteStorage(db, func() time.Time { return time.Unix(0, 0).UTC() })
}

func domainEvent(id, ns, typ string, emitted time.Time, payload map[string]any) core.DomainEvent {
	return core.DomainEvent{
		EventID: id, Namespace: ns, EventType: typ, Source: "test",
		Payload: payload, EmittedAt: emitted,
	}
}

// ── migration 0002 present ────────────────────────────────────────────────────

func TestDomainEventsTableExists(t *testing.T) {
	db := openTestDB(t)
	var name string
	if err := db.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='domain_events'`,
	).Scan(&name); err != nil {
		t.Fatalf("domain_events table not found: %v", err)
	}
	if err := db.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='index' AND name='idx_domain_events_ns_type_time'`,
	).Scan(&name); err != nil {
		t.Fatalf("domain_events index not found: %v", err)
	}
}

// ── insert / duplicate ────────────────────────────────────────────────────────

func TestInsertDomainEvent_RoundTrip(t *testing.T) {
	s := newTriggerStore(t)
	ctx := context.Background()
	ev := domainEvent("de-1", "oip", "note.captured",
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), map[string]any{"k": "v", "n": float64(3)})
	if err := s.InsertDomainEvent(ctx, ev); err != nil {
		t.Fatalf("InsertDomainEvent: %v", err)
	}
	got, err := s.ListUnconsumedDomainEvents(ctx, "oip")
	if err != nil {
		t.Fatalf("ListUnconsumedDomainEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 event, got %d", len(got))
	}
	if got[0].EventID != "de-1" || got[0].EventType != "note.captured" || got[0].Source != "test" {
		t.Fatalf("scalar mismatch: %+v", got[0])
	}
	if got[0].Payload["k"] != "v" || got[0].Payload["n"] != float64(3) {
		t.Fatalf("payload round-trip mismatch: %+v", got[0].Payload)
	}
	if got[0].ConsumedAt != nil {
		t.Fatalf("fresh event must be unconsumed, got consumed_at=%v", got[0].ConsumedAt)
	}
}

func TestInsertDomainEvent_DuplicateIsTypedError(t *testing.T) {
	s := newTriggerStore(t)
	ctx := context.Background()
	ev := domainEvent("de-dup", "oip", "t", time.Unix(100, 0).UTC(), nil)
	if err := s.InsertDomainEvent(ctx, ev); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	err := s.InsertDomainEvent(ctx, ev)
	if !errors.Is(err, ErrDuplicateDomainEvent) {
		t.Fatalf("err = %v, want ErrDuplicateDomainEvent", err)
	}
}

// ── list ordering + namespace scoping ─────────────────────────────────────────

func TestListUnconsumedDomainEvents_Ordering(t *testing.T) {
	s := newTriggerStore(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// Insert out of order; same emitted_at for b1/b2 disambiguated by event_id.
	must := func(ev core.DomainEvent) {
		if err := s.InsertDomainEvent(ctx, ev); err != nil {
			t.Fatalf("insert %s: %v", ev.EventID, err)
		}
	}
	must(domainEvent("b2", "oip", "t", base.Add(time.Hour), nil))
	must(domainEvent("a1", "oip", "t", base, nil))
	must(domainEvent("b1", "oip", "t", base.Add(time.Hour), nil))
	got, err := s.ListUnconsumedDomainEvents(ctx, "oip")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	wantOrder := []string{"a1", "b1", "b2"} // emitted_at then event_id.
	if len(got) != len(wantOrder) {
		t.Fatalf("want %d events, got %d", len(wantOrder), len(got))
	}
	for i, id := range wantOrder {
		if got[i].EventID != id {
			t.Fatalf("order[%d] = %s, want %s", i, got[i].EventID, id)
		}
	}
}

func TestListUnconsumedDomainEvents_NamespaceScoping(t *testing.T) {
	s := newTriggerStore(t)
	ctx := context.Background()
	base := time.Unix(10, 0).UTC()
	_ = s.InsertDomainEvent(ctx, domainEvent("oip-1", "oip", "t", base, nil))
	_ = s.InsertDomainEvent(ctx, domainEvent("other-1", "other", "t", base, nil))

	oip, err := s.ListUnconsumedDomainEvents(ctx, "oip")
	if err != nil {
		t.Fatalf("list oip: %v", err)
	}
	if len(oip) != 1 || oip[0].EventID != "oip-1" {
		t.Fatalf("namespace scope failed: %+v", oip)
	}
	all, err := s.ListUnconsumedDomainEvents(ctx, "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("empty namespace must return all, got %d", len(all))
	}
}

// ── mark consumed ─────────────────────────────────────────────────────────────

func TestMarkDomainEventConsumed_HidesFromUnconsumed(t *testing.T) {
	s := newTriggerStore(t)
	ctx := context.Background()
	must := s.InsertDomainEvent(ctx, domainEvent("de-c", "oip", "t", time.Unix(5, 0).UTC(), nil))
	if must != nil {
		t.Fatalf("insert: %v", must)
	}
	at := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	if err := s.MarkDomainEventConsumed(ctx, "de-c", at); err != nil {
		t.Fatalf("mark: %v", err)
	}
	got, err := s.ListUnconsumedDomainEvents(ctx, "oip")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("consumed event must not appear in unconsumed list, got %d", len(got))
	}
}

func TestMarkDomainEventConsumed_AbsentIsNoOp(t *testing.T) {
	s := newTriggerStore(t)
	// Marking a non-existent event is a benign no-op (may race the TTL prune).
	if err := s.MarkDomainEventConsumed(context.Background(), "no-such", time.Unix(1, 0).UTC()); err != nil {
		t.Fatalf("mark absent = %v, want nil", err)
	}
}

// ── prune ─────────────────────────────────────────────────────────────────────

func TestPruneExpiredDomainEvents_DeletesOldKeepsFresh(t *testing.T) {
	s := newTriggerStore(t)
	ctx := context.Background()
	old := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	fresh := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	_ = s.InsertDomainEvent(ctx, domainEvent("old-1", "oip", "t", old, nil))
	_ = s.InsertDomainEvent(ctx, domainEvent("fresh-1", "oip", "t", fresh, nil))

	cutoff := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	n, err := s.PruneExpiredDomainEvents(ctx, cutoff)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if n != 1 {
		t.Fatalf("pruned = %d, want 1", n)
	}
	got, err := s.ListUnconsumedDomainEvents(ctx, "oip")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].EventID != "fresh-1" {
		t.Fatalf("after prune want only fresh-1, got %+v", got)
	}
}

// PruneExpiredDomainEvents deletes consumed events too (retention bound).
func TestPruneExpiredDomainEvents_PrunesConsumed(t *testing.T) {
	s := newTriggerStore(t)
	ctx := context.Background()
	old := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	_ = s.InsertDomainEvent(ctx, domainEvent("consumed-old", "oip", "t", old, nil))
	if err := s.MarkDomainEventConsumed(ctx, "consumed-old", old.Add(time.Minute)); err != nil {
		t.Fatalf("mark: %v", err)
	}
	n, err := s.PruneExpiredDomainEvents(ctx, time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if n != 1 {
		t.Fatalf("consumed old event must be pruned; pruned = %d, want 1", n)
	}
}
