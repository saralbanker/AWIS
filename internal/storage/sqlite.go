// Package storage implements the SQLite-backed StoragePort adapter for the AWIS
// runtime (Blueprint §20; edr-003). This file contains the StoragePort method
// implementations (M02-C2); DB open + migration runner live in db.go/migrate.go.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// Compile-time assertion: SQLiteStorage must satisfy core.StoragePort.
var _ core.StoragePort = (*SQLiteStorage)(nil)

// Sentinel errors returned by the adapter.
var (
	// ErrSequenceViolation is returned by AppendEvent when the supplied
	// sequence_num is not strictly greater than the current maximum for the
	// instance (FR-ST-01; TDS-01 §3).
	ErrSequenceViolation = errors.New("storage: sequence_num must be strictly increasing per instance")

	// ErrAlreadyRegistered is returned by RegisterWorkflow when a definition
	// with the same (id, version) pair already exists (TDS-02 §5 immutability).
	ErrAlreadyRegistered = errors.New("storage: workflow (id, version) already registered")

	// ErrNotImplemented is returned by StateStore stubs until M03.
	ErrNotImplemented = errors.New("storage: not implemented (M03)")
)

// SQLiteStorage is the SQLite-backed implementation of core.StoragePort.
// Obtain one via NewSQLiteStorage; do not construct directly.
type SQLiteStorage struct {
	db  *DB
	now func() time.Time // injectable clock
}

// NewSQLiteStorage wraps an open *DB and returns a StoragePort implementation.
// clock is an injectable time source; pass nil to use time.Now.
func NewSQLiteStorage(db *DB, clock func() time.Time) *SQLiteStorage {
	if clock == nil {
		clock = time.Now
	}
	return &SQLiteStorage{db: db, now: clock}
}

// ── EventLog ──────────────────────────────────────────────────────────────────

// AppendEvent appends one event to the EventLog inside a single transaction.
// It enforces per-instance strictly-increasing sequence_num: if event.SequenceNum
// is <= the current maximum for the instance the call returns ErrSequenceViolation
// (FR-ST-01; TDS-01 §3). No UPDATE or DELETE is ever issued on execution_events.
func (s *SQLiteStorage) AppendEvent(ctx context.Context, event core.ExecutionEvent) error {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage: AppendEvent begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Enforce monotonicity: SELECT max(sequence_num) for this instance.
	var maxSeq sql.NullInt64
	row := tx.QueryRowContext(ctx,
		`SELECT MAX(sequence_num) FROM execution_events WHERE instance_id = ?`,
		string(event.InstanceID),
	)
	if err := row.Scan(&maxSeq); err != nil {
		return fmt.Errorf("storage: AppendEvent read max seq: %w", err)
	}
	if maxSeq.Valid && event.SequenceNum <= int(maxSeq.Int64) {
		return fmt.Errorf("%w: got %d, max is %d", ErrSequenceViolation, event.SequenceNum, maxSeq.Int64)
	}

	// Normalise emitted_at to RFC3339 UTC.
	emittedAt := event.EmittedAt.UTC().Format(time.RFC3339Nano)

	// Payload stored as raw JSON passthrough.
	payload := []byte(event.Payload)
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	// stepID nullable.
	var stepID *string
	if event.StepID != "" {
		s := event.StepID
		stepID = &s
	}

	// Zero value mirrors the DDL default (TDS-01 §1.1: defaults to 1); never
	// persist schema_version 0.
	schemaVersion := event.SchemaVersion
	if schemaVersion == 0 {
		schemaVersion = 1
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO execution_events
			(event_id, instance_id, namespace, event_type, step_id, payload,
			 emitted_at, sequence_num, schema_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventID,
		string(event.InstanceID),
		event.Namespace,
		string(event.EventType),
		stepID,
		string(payload),
		emittedAt,
		event.SequenceNum,
		schemaVersion,
	)
	if err != nil {
		return fmt.Errorf("storage: AppendEvent insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage: AppendEvent commit: %w", err)
	}
	return nil
}

// ReadEvents returns all events for instanceID with sequence_num >= fromSeq,
// ordered by sequence_num ascending.
func (s *SQLiteStorage) ReadEvents(ctx context.Context, instanceID core.InstanceID, fromSeq int) ([]core.ExecutionEvent, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT event_id, instance_id, namespace, event_type, step_id, payload,
		       emitted_at, sequence_num, schema_version
		FROM execution_events
		WHERE instance_id = ? AND sequence_num >= ?
		ORDER BY sequence_num`,
		string(instanceID), fromSeq,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: ReadEvents query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanEvents(rows)
}

// ReadEventRange returns events for the given namespace with emitted_at in
// [from, to], ordered by emitted_at ascending (NFR-S-04 namespace predicate).
func (s *SQLiteStorage) ReadEventRange(ctx context.Context, namespace string, from, to time.Time) ([]core.ExecutionEvent, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT event_id, instance_id, namespace, event_type, step_id, payload,
		       emitted_at, sequence_num, schema_version
		FROM execution_events
		WHERE namespace = ? AND emitted_at BETWEEN ? AND ?
		ORDER BY emitted_at`,
		namespace,
		from.UTC().Format(time.RFC3339Nano),
		to.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: ReadEventRange query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanEvents(rows)
}

// scanEvents scans a *sql.Rows into []core.ExecutionEvent (faithful 9-field scan).
func scanEvents(rows *sql.Rows) ([]core.ExecutionEvent, error) {
	var events []core.ExecutionEvent
	for rows.Next() {
		var e core.ExecutionEvent
		var instanceID, eventType, stepIDNullable sql.NullString
		var emittedAtStr, payloadStr string

		if err := rows.Scan(
			&e.EventID,
			&instanceID,
			&e.Namespace,
			&eventType,
			&stepIDNullable,
			&payloadStr,
			&emittedAtStr,
			&e.SequenceNum,
			&e.SchemaVersion,
		); err != nil {
			return nil, fmt.Errorf("storage: scan event: %w", err)
		}
		e.Payload = json.RawMessage(payloadStr)

		e.InstanceID = core.InstanceID(instanceID.String)
		e.EventType = core.EventType(eventType.String)
		if stepIDNullable.Valid {
			e.StepID = stepIDNullable.String
		}

		t, err := time.Parse(time.RFC3339Nano, emittedAtStr)
		if err != nil {
			// Try RFC3339 without nanoseconds.
			t, err = time.Parse(time.RFC3339, emittedAtStr)
			if err != nil {
				return nil, fmt.Errorf("storage: parse emitted_at %q: %w", emittedAtStr, err)
			}
		}
		e.EmittedAt = t.UTC()

		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ReadEvents rows: %w", err)
	}
	return events, nil
}

// ── WorkflowRegistry ──────────────────────────────────────────────────────────

// RegisterWorkflow marshals def to JSON and inserts it. Re-registering an
// existing (id, version) pair returns ErrAlreadyRegistered (TDS-02 §5).
func (s *SQLiteStorage) RegisterWorkflow(ctx context.Context, def core.WorkflowDefinition) error {
	data, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("storage: RegisterWorkflow marshal: %w", err)
	}

	_, err = s.db.db.ExecContext(ctx, `
		INSERT INTO workflow_definitions (id, version, namespace, definition, registered_at)
		VALUES (?, ?, ?, ?, ?)`,
		def.ID,
		string(def.Version),
		def.Namespace,
		string(data),
		s.now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		// SQLite UNIQUE constraint violation text contains "UNIQUE constraint failed".
		if isUniqueConstraintError(err) {
			return fmt.Errorf("%w: id=%s version=%s", ErrAlreadyRegistered, def.ID, def.Version)
		}
		return fmt.Errorf("storage: RegisterWorkflow insert: %w", err)
	}
	return nil
}

// GetWorkflow fetches a definition by id and version and unmarshals it.
func (s *SQLiteStorage) GetWorkflow(ctx context.Context, id string, version core.SemVer) (core.WorkflowDefinition, error) {
	var data string
	err := s.db.db.QueryRowContext(ctx,
		`SELECT definition FROM workflow_definitions WHERE id = ? AND version = ?`,
		id, string(version),
	).Scan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.WorkflowDefinition{}, fmt.Errorf("storage: GetWorkflow %q %q: not found", id, version)
		}
		return core.WorkflowDefinition{}, fmt.Errorf("storage: GetWorkflow query: %w", err)
	}

	var def core.WorkflowDefinition
	if err := json.Unmarshal([]byte(data), &def); err != nil {
		return core.WorkflowDefinition{}, fmt.Errorf("storage: GetWorkflow unmarshal: %w", err)
	}
	return def, nil
}

// ListWorkflows returns all definitions in namespace (NFR-S-04 predicate).
func (s *SQLiteStorage) ListWorkflows(ctx context.Context, namespace string) ([]core.WorkflowDefinition, error) {
	rows, err := s.db.db.QueryContext(ctx,
		`SELECT definition FROM workflow_definitions WHERE namespace = ?`,
		namespace,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: ListWorkflows query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var defs []core.WorkflowDefinition
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("storage: ListWorkflows scan: %w", err)
		}
		var def core.WorkflowDefinition
		if err := json.Unmarshal([]byte(data), &def); err != nil {
			return nil, fmt.Errorf("storage: ListWorkflows unmarshal: %w", err)
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListWorkflows rows: %w", err)
	}
	return defs, nil
}

// ── StepResultCache ───────────────────────────────────────────────────────────

// CacheResult stores (or replaces) a step result under key with the given TTL.
// expires_at is computed via the injectable clock (IMP §3 determinism rule).
func (s *SQLiteStorage) CacheResult(ctx context.Context, key core.IdempotencyKey, result core.StepResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("storage: CacheResult marshal: %w", err)
	}

	now := s.now().UTC()
	_, err = s.db.db.ExecContext(ctx, `
		INSERT INTO step_results_cache (idempotency_key, result, cached_at, expires_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(idempotency_key) DO UPDATE SET
			result     = excluded.result,
			cached_at  = excluded.cached_at,
			expires_at = excluded.expires_at`,
		string(key),
		string(data),
		now.Format(time.RFC3339Nano),
		now.Add(ttl).Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("storage: CacheResult upsert: %w", err)
	}
	return nil
}

// GetCachedResult fetches a cached step result. Returns (zero, false, nil) when
// absent or expired; (result, true, nil) when present and valid.
func (s *SQLiteStorage) GetCachedResult(ctx context.Context, key core.IdempotencyKey) (core.StepResult, bool, error) {
	var data, expiresAtStr string
	err := s.db.db.QueryRowContext(ctx,
		`SELECT result, expires_at FROM step_results_cache WHERE idempotency_key = ?`,
		string(key),
	).Scan(&data, &expiresAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.StepResult{}, false, nil
		}
		return core.StepResult{}, false, fmt.Errorf("storage: GetCachedResult query: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtStr)
	if err != nil {
		expiresAt, err = time.Parse(time.RFC3339, expiresAtStr)
		if err != nil {
			return core.StepResult{}, false, fmt.Errorf("storage: GetCachedResult parse expires_at: %w", err)
		}
	}

	if s.now().UTC().After(expiresAt.UTC()) {
		return core.StepResult{}, false, nil
	}

	var result core.StepResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return core.StepResult{}, false, fmt.Errorf("storage: GetCachedResult unmarshal: %w", err)
	}
	return result, true, nil
}

// ── StateStore stubs (M03) ────────────────────────────────────────────────────

// UpsertInstance is not yet implemented (M03).
func (s *SQLiteStorage) UpsertInstance(_ context.Context, _ core.WorkflowInstance, _ int) error {
	return fmt.Errorf("%w: UpsertInstance", ErrNotImplemented)
}

// GetInstance is not yet implemented (M03).
func (s *SQLiteStorage) GetInstance(_ context.Context, _ core.InstanceID) (core.WorkflowInstance, error) {
	return core.WorkflowInstance{}, fmt.Errorf("%w: GetInstance", ErrNotImplemented)
}

// ListInstances is not yet implemented (M03).
func (s *SQLiteStorage) ListInstances(_ context.Context, _ core.InstanceFilter) ([]core.WorkflowInstance, error) {
	return nil, fmt.Errorf("%w: ListInstances", ErrNotImplemented)
}

// ClaimStep is not yet implemented (M03).
func (s *SQLiteStorage) ClaimStep(_ context.Context, _ core.InstanceID, _ string, _ string) (bool, error) {
	return false, fmt.Errorf("%w: ClaimStep", ErrNotImplemented)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// isUniqueConstraintError reports whether err is a SQLite UNIQUE constraint
// violation. modernc.org/sqlite surfaces these as an error whose message
// contains "UNIQUE constraint failed".
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for i := 0; i < len(msg)-21; i++ {
		if msg[i:i+22] == "UNIQUE constraint fail" {
			return true
		}
	}
	return false
}
