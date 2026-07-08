package storage

// Audit-log storage accessor (M07-C1, F-4).
//
// The AuditLog is separate from the EventLog and uses a different storage table
// (audit_log, migration 0004), making it independently auditable (PRD §26). It
// records administrative facts — WorkflowRegistered, PluginRegistered,
// ConfigChanged, SignalDelivered — and is never pruned without explicit operator
// action (PRD NFR-S-05). This card lands the APPEND path only (F-4); there is no
// read path here — audit rendering is M17. The first call site is added by C2.
//
// Append-only invariant (PRD NFR-S-05 / §26): this file exposes AppendAudit and
// nothing else — no update and no delete path. The store never mutates an
// audit_log row in place.

import (
	"context"
	"fmt"
	"time"
)

// AuditEntry is one row appended to audit_log. Its fields realise the PRD §26
// render shape (FR-OB-08 / §15): timestamp, event type, actor, payload summary.
// It is a storage-package projection, not a frozen core type.
type AuditEntry struct {
	// Timestamp is when the audited action occurred (caller-supplied; the caller
	// owns the injectable clock per IMP §3).
	Timestamp time.Time
	// EventType is the audit event type, e.g. WorkflowRegistered, PluginRegistered,
	// ConfigChanged, SignalDelivered (PRD NFR-S-05; the set is intentionally
	// unenumerated in the schema — CONTRA-4).
	EventType string
	// Actor identifies who performed the audited action.
	Actor string
	// PayloadSummary is a human-readable summary of the action's payload.
	PayloadSummary string
}

// AppendAudit appends one entry to audit_log (F-4). timestamp is persisted in
// RFC3339Nano UTC (the AppendEvent convention). The DB assigns the surrogate
// append id; the append never rejects a well-formed entry (no UNIQUE beyond the
// surrogate key) and offers no in-place update — append-only per NFR-S-05.
func (s *SQLiteStorage) AppendAudit(ctx context.Context, entry AuditEntry) error {
	_, err := s.db.db.ExecContext(ctx, `
		INSERT INTO audit_log (timestamp, event_type, actor, payload_summary)
		VALUES (?, ?, ?, ?)`,
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
		entry.EventType,
		entry.Actor,
		entry.PayloadSummary,
	)
	if err != nil {
		return fmt.Errorf("storage: AppendAudit insert: %w", err)
	}
	return nil
}
