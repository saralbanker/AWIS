package storage

// RecallStore is the additive storage interface for FTS-based execution history
// search (IMP §14; CONTRA-4 additive; StoragePort method set untouched).
//
// RecallStore is type-asserted from the storage implementation — the same pattern
// as PluginStore (plugins.go / M12-C2) and AuditLog (audit.go / M07-C1).
//
// TRACEABILITY: migration 0006_recall_fts.sql lands the FTS5 virtual table and
// the INSERT trigger that maintains it. RecallStore adds the query surface.

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// RecallRow is one result returned by SearchEvents and ListAudit.
type RecallRow struct {
	EventID    string
	InstanceID string
	Namespace  string
	EventType  string
	StepID     string
	Payload    string
	EmittedAt  time.Time
	Rank       float64 // FTS bm25 relevance (lower = more relevant for SQLite bm25)
}

// AuditRow is one row returned by ListAudit.
type AuditRow struct {
	ID             int64
	Timestamp      time.Time
	EventType      string
	Actor          string
	PayloadSummary string
}

// RecallStore is the additive storage interface for recall/audit read paths.
type RecallStore interface {
	// SearchEvents performs FTS5 full-text search over execution_events payloads.
	// Returns up to limit rows ordered by FTS relevance (bm25). limit ≤ 0 means 50.
	SearchEvents(ctx context.Context, query string, limit int) ([]RecallRow, error)

	// ListAudit returns audit_log rows in descending timestamp order.
	// limit ≤ 0 means 50.
	ListAudit(ctx context.Context, limit int) ([]AuditRow, error)
}

// ── SQLiteStorage implementation ─────────────────────────────────────────────

// SearchEvents performs FTS5 full-text search over execution_events payloads.
// Returns up to limit rows ordered by FTS bm25 relevance.
func (s *SQLiteStorage) SearchEvents(ctx context.Context, query string, limit int) ([]RecallRow, error) {
	if limit <= 0 {
		limit = 50
	}

	// Lazy FTS sync: rebuild the FTS index from execution_events before querying.
	// This avoids per-insert trigger overhead (prohibitive at 10K-100K event scale).
	// V1 recall is a developer-facing search (not latency-critical); rebuild cost
	// is amortized over all insertions since last search.
	if err := s.rebuildFTSIndex(ctx); err != nil {
		return nil, fmt.Errorf("storage: SearchEvents rebuild FTS: %w", err)
	}

	// Wrap the query in double-quotes to treat it as a literal phrase rather than
	// letting FTS5 parse hyphens/special chars as operators (e.g. "-term" = NOT term).
	// Users can still use raw FTS5 syntax by starting with a special character.
	ftsQuery := `"` + query + `"`

	// JOIN back to execution_events on event_id to retrieve emitted_at.
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT f.event_id, f.instance_id, f.namespace, f.event_type, f.step_id, f.payload,
		       e.emitted_at, bm25(execution_events_fts) AS rank
		FROM execution_events_fts f
		JOIN execution_events e ON f.event_id = e.event_id
		WHERE execution_events_fts MATCH ?
		ORDER BY rank
		LIMIT ?`,
		ftsQuery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: SearchEvents query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []RecallRow
	for rows.Next() {
		var r RecallRow
		var stepID sql.NullString
		var emittedAtStr string
		if err := rows.Scan(
			&r.EventID, &r.InstanceID, &r.Namespace, &r.EventType, &stepID,
			&r.Payload, &emittedAtStr, &r.Rank,
		); err != nil {
			return nil, fmt.Errorf("storage: SearchEvents scan: %w", err)
		}
		if stepID.Valid {
			r.StepID = stepID.String
		}
		t, err := parseTimeStr(emittedAtStr)
		if err != nil {
			return nil, fmt.Errorf("storage: SearchEvents parse emitted_at: %w", err)
		}
		r.EmittedAt = t
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: SearchEvents rows: %w", err)
	}
	return results, nil
}

// rebuildFTSIndex synchronizes execution_events_fts with execution_events.
// Uses a DELETE+INSERT approach (replace semantics) inside a single transaction
// so the FTS table is always consistent with the source table.
// Called lazily on each SearchEvents (V1 developer-facing search; not hot-path).
func (s *SQLiteStorage) rebuildFTSIndex(ctx context.Context) error {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage: rebuildFTSIndex begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Clear existing FTS content.
	if _, err := tx.ExecContext(ctx, `DELETE FROM execution_events_fts`); err != nil {
		return fmt.Errorf("storage: rebuildFTSIndex delete: %w", err)
	}

	// Repopulate from execution_events.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO execution_events_fts(event_id, instance_id, namespace, event_type, step_id, payload)
		SELECT event_id, instance_id, namespace, event_type, COALESCE(step_id, ''), payload
		FROM execution_events`); err != nil {
		return fmt.Errorf("storage: rebuildFTSIndex insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage: rebuildFTSIndex commit: %w", err)
	}
	return nil
}

// ListAudit returns audit_log rows in descending timestamp order.
func (s *SQLiteStorage) ListAudit(ctx context.Context, limit int) ([]AuditRow, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.db.QueryContext(ctx, `
		SELECT id, timestamp, event_type, actor, payload_summary
		FROM audit_log
		ORDER BY id DESC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: ListAudit query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []AuditRow
	for rows.Next() {
		var r AuditRow
		var tsStr string
		if err := rows.Scan(&r.ID, &tsStr, &r.EventType, &r.Actor, &r.PayloadSummary); err != nil {
			return nil, fmt.Errorf("storage: ListAudit scan: %w", err)
		}
		t, err := parseTimeStr(tsStr)
		if err != nil {
			return nil, fmt.Errorf("storage: ListAudit parse timestamp: %w", err)
		}
		r.Timestamp = t
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListAudit rows: %w", err)
	}
	return results, nil
}
