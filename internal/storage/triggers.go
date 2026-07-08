package storage

// Domain-event (trigger) storage accessors (M06-C3, F-2 / T5).
//
// StoragePort's 12 methods are frozen (Blueprint §20). Trigger storage — the
// domain_events table added by migration 0002 — is exposed via ADDITIVE
// *SQLiteStorage methods, NOT new StoragePort methods, exactly as the
// cancellation flag is (cancellation.go). The engine reaches them through an
// engine-internal interface it type-asserts (triggerStore in trigger.go), so
// StoragePort stays untouched.
//
// Semantics are frozen by EDR-011 §4: an event is consumed when ≥1 workflow
// fired from it; unmatched events survive until the 7-day TTL prune.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// ErrDuplicateDomainEvent is returned by InsertDomainEvent when a row with the
// same event_id already exists. It mirrors the ErrAlreadyRegistered idiom used
// by RegisterWorkflow for a duplicate (id, version): domain-event ingestion is
// NOT idempotent-skip — a duplicate event_id is a typed, caller-observable error
// so a mis-emitting source is surfaced rather than silently swallowed.
var ErrDuplicateDomainEvent = errors.New("storage: domain event already ingested")

// InsertDomainEvent persists ev into domain_events (F-2, EDR-011 §4). The
// payload is JSON-marshalled; emitted_at is normalised to RFC3339Nano UTC (the
// AppendEvent / UpsertInstance convention). consumed_at is written NULL (a fresh
// event is unconsumed). A duplicate event_id returns ErrDuplicateDomainEvent.
func (s *SQLiteStorage) InsertDomainEvent(ctx context.Context, ev core.DomainEvent) error {
	payload := ev.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("storage: InsertDomainEvent marshal payload: %w", err)
	}

	var consumedAt *string
	if ev.ConsumedAt != nil {
		c := ev.ConsumedAt.UTC().Format(time.RFC3339Nano)
		consumedAt = &c
	}

	_, err = s.db.db.ExecContext(ctx, `
		INSERT INTO domain_events (event_id, namespace, event_type, source, payload, emitted_at, consumed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ev.EventID,
		ev.Namespace,
		ev.EventType,
		ev.Source,
		string(data),
		ev.EmittedAt.UTC().Format(time.RFC3339Nano),
		consumedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("%w: event_id=%s", ErrDuplicateDomainEvent, ev.EventID)
		}
		return fmt.Errorf("storage: InsertDomainEvent insert: %w", err)
	}
	return nil
}

// ListUnconsumedDomainEvents returns every domain event with consumed_at NULL,
// ordered by (emitted_at, event_id) for a deterministic scan (EDR-011 §4). An
// empty namespace matches ALL namespaces: the engine's SCAN_TRIGGERABLE stage
// has no namespace enumeration of its own, so it lists across namespaces and
// resolves each event against ListWorkflows(ev.Namespace). A non-empty namespace
// restricts the result to that namespace.
func (s *SQLiteStorage) ListUnconsumedDomainEvents(ctx context.Context, namespace string) ([]core.DomainEvent, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT event_id, namespace, event_type, source, payload, emitted_at, consumed_at
		FROM domain_events
		WHERE consumed_at IS NULL AND (? = '' OR namespace = ?)
		ORDER BY emitted_at, event_id`,
		namespace, namespace,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: ListUnconsumedDomainEvents query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []core.DomainEvent
	for rows.Next() {
		var (
			ev            core.DomainEvent
			payloadStr    string
			emittedAtStr  string
			consumedAtStr sql.NullString
		)
		if err := rows.Scan(&ev.EventID, &ev.Namespace, &ev.EventType, &ev.Source,
			&payloadStr, &emittedAtStr, &consumedAtStr); err != nil {
			return nil, fmt.Errorf("storage: ListUnconsumedDomainEvents scan: %w", err)
		}
		if err := json.Unmarshal([]byte(payloadStr), &ev.Payload); err != nil {
			return nil, fmt.Errorf("storage: ListUnconsumedDomainEvents unmarshal payload: %w", err)
		}
		emittedAt, err := parseTimeStr(emittedAtStr)
		if err != nil {
			return nil, fmt.Errorf("storage: ListUnconsumedDomainEvents emitted_at: %w", err)
		}
		ev.EmittedAt = emittedAt
		if consumedAtStr.Valid {
			c, err := parseTimeStr(consumedAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("storage: ListUnconsumedDomainEvents consumed_at: %w", err)
			}
			ev.ConsumedAt = &c
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListUnconsumedDomainEvents rows: %w", err)
	}
	return out, nil
}

// MarkDomainEventConsumed stamps consumed_at=at (RFC3339Nano UTC) for eventID
// (EDR-011 §4: consumed once ≥1 workflow fired). Marking an absent or
// already-pruned event is a benign no-op (no error): the trigger scan may race
// the TTL prune, and consumption is monotonic, so a missing row is not a fault.
func (s *SQLiteStorage) MarkDomainEventConsumed(ctx context.Context, eventID string, at time.Time) error {
	_, err := s.db.db.ExecContext(ctx,
		`UPDATE domain_events SET consumed_at = ? WHERE event_id = ? AND consumed_at IS NULL`,
		at.UTC().Format(time.RFC3339Nano), eventID,
	)
	if err != nil {
		return fmt.Errorf("storage: MarkDomainEventConsumed update: %w", err)
	}
	return nil
}

// PruneExpiredDomainEvents deletes every domain event whose emitted_at is
// strictly before olderThan and returns the number deleted (Blueprint §10 L724
// "TTL 7d"; EDR-011 §4). Reading: the 7-day TTL is a table-wide RETENTION bound,
// so BOTH consumed and unconsumed events older than the bound are pruned —
// otherwise consumed rows would accumulate without limit in a table the frozen
// text calls TTL'd. EDR-011 §4's "unmatched events survive until the prune" is
// the special case (unmatched == never consumed); this prune subsumes it by
// bounding retention for all events alike.
func (s *SQLiteStorage) PruneExpiredDomainEvents(ctx context.Context, olderThan time.Time) (int64, error) {
	res, err := s.db.db.ExecContext(ctx,
		`DELETE FROM domain_events WHERE emitted_at < ?`,
		olderThan.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("storage: PruneExpiredDomainEvents delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("storage: PruneExpiredDomainEvents rows affected: %w", err)
	}
	return n, nil
}
