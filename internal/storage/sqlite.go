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

// MaxSupportedSchemaVersion is the highest execution_events.schema_version
// value this build of AWIS knows how to project (B-22).
//
// The EventLog is the append-only source of truth (Blueprint §8): every
// reader projects state from the raw events it reads back. A newer AWIS
// build could append events with a higher schema_version once the payload
// schema changes; an older build reading those rows back must not silently
// treat them as schema_version 1 and mis-project their payload — that is
// silent corruption of the derived state, strictly worse than refusing to
// read. AppendEvent rejects writing above this ceiling and ReadEvents /
// ReadEventRange reject reading a row above it; both return the typed
// ErrUnsupportedSchemaVersion so a caller can distinguish this from a
// generic I/O error. Raise this constant only when this build actually
// gains the ability to project the new schema version.
const MaxSupportedSchemaVersion = 1

// Sentinel errors returned by the adapter.
var (
	// ErrSequenceViolation is returned by AppendEvent when the supplied
	// sequence_num is not strictly greater than the current maximum for the
	// instance (FR-ST-01; TDS-01 §3).
	ErrSequenceViolation = errors.New("storage: sequence_num must be strictly increasing per instance")

	// ErrUnsupportedSchemaVersion is returned by AppendEvent when the event's
	// SchemaVersion exceeds MaxSupportedSchemaVersion, and by ReadEvents /
	// ReadEventRange when a row above that ceiling is encountered (B-22).
	ErrUnsupportedSchemaVersion = errors.New("storage: schema_version exceeds MaxSupportedSchemaVersion")

	// ErrAlreadyRegistered is returned by RegisterWorkflow when a definition
	// with the same (id, version) pair already exists (TDS-02 §5 immutability).
	ErrAlreadyRegistered = errors.New("storage: workflow (id, version) already registered")

	// ErrVersionConflict is returned by UpsertInstance when the expectedVersion
	// does not match the stored version (optimistic concurrency control; M03).
	ErrVersionConflict = errors.New("storage: version conflict")

	// ErrInstanceNotFound is returned by GetInstance when no row exists for the
	// given instance_id (M03).
	ErrInstanceNotFound = errors.New("storage: instance not found")

	// ErrWorkflowNotFound is returned by GetWorkflow when no row exists for the
	// given (id, version) pair (E-G1-3).
	ErrWorkflowNotFound = errors.New("storage: workflow not found")
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

// Ping verifies the underlying database connection is live (SEC-12). It is
// additive — not part of the frozen 12-method StoragePort — so callers that
// need a liveness check (internal/api's healthz handler) type-assert for it,
// the same pattern used for cancellationStore and the other additive slices.
func (s *SQLiteStorage) Ping(ctx context.Context) error {
	return s.db.db.PingContext(ctx)
}

// ── EventLog ──────────────────────────────────────────────────────────────────

// AppendEvent appends one event to the EventLog inside a single transaction.
// It enforces per-instance strictly-increasing sequence_num: if event.SequenceNum
// is <= the current maximum for the instance the call returns ErrSequenceViolation
// (FR-ST-01; TDS-01 §3). No UPDATE or DELETE is ever issued on execution_events.
//
// It also enforces the schema-version ceiling (B-22): an event.SchemaVersion
// greater than MaxSupportedSchemaVersion is rejected with
// ErrUnsupportedSchemaVersion rather than written, since this build cannot
// guarantee it can later project that row correctly.
func (s *SQLiteStorage) AppendEvent(ctx context.Context, event core.ExecutionEvent) error {
	if event.SchemaVersion > MaxSupportedSchemaVersion {
		return fmt.Errorf("%w: got %d, max supported is %d",
			ErrUnsupportedSchemaVersion, event.SchemaVersion, MaxSupportedSchemaVersion)
	}

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

// Pagination bounds for ReadEventsPaged (E-G4-6). DefaultEventsPageSize
// applies when a caller passes limit <= 0; MaxEventsPageSize is a hard
// ceiling, mirroring ListInstancesPaged's B-17 pattern so a caller can never
// force an unbounded scan through this method either. Exported (unlike
// ListInstancesPaged's equivalents) because internal/api needs to know the
// applied default to detect a full page — see ReadEventsPaged's doc comment.
const (
	DefaultEventsPageSize = 100
	MaxEventsPageSize     = 1000
)

// ReadEventsPaged returns at most limit events for instanceID with
// sequence_num >= fromSeq, ordered by sequence_num ascending — the same
// predicate and ordering as ReadEvents, with a bound added. limit <= 0 is
// normalised to DefaultEventsPageSize; limit above MaxEventsPageSize is
// clamped down to it. This is purely additive: it does not change
// ReadEvents, which remains part of the frozen 12-method StoragePort and
// stays unbounded.
func (s *SQLiteStorage) ReadEventsPaged(ctx context.Context, instanceID core.InstanceID, fromSeq, limit int) ([]core.ExecutionEvent, error) {
	if limit <= 0 {
		limit = DefaultEventsPageSize
	}
	if limit > MaxEventsPageSize {
		limit = MaxEventsPageSize
	}

	rows, err := s.db.db.QueryContext(ctx, `
		SELECT event_id, instance_id, namespace, event_type, step_id, payload,
		       emitted_at, sequence_num, schema_version
		FROM execution_events
		WHERE instance_id = ? AND sequence_num >= ?
		ORDER BY sequence_num
		LIMIT ?`,
		string(instanceID), fromSeq, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: ReadEventsPaged query: %w", err)
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

		// Refuse to project a row above the schema-version ceiling rather than
		// silently mis-interpreting it as MaxSupportedSchemaVersion (B-22):
		// mis-projection of the append-only source of truth is worse than a
		// failed read.
		if e.SchemaVersion > MaxSupportedSchemaVersion {
			return nil, fmt.Errorf("%w: event_id=%s got %d, max supported is %d",
				ErrUnsupportedSchemaVersion, e.EventID, e.SchemaVersion, MaxSupportedSchemaVersion)
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
			return core.WorkflowDefinition{}, fmt.Errorf("%w: id=%s version=%s", ErrWorkflowNotFound, id, version)
		}
		return core.WorkflowDefinition{}, fmt.Errorf("storage: GetWorkflow query: %w", err)
	}

	var def core.WorkflowDefinition
	if err := json.Unmarshal([]byte(data), &def); err != nil {
		return core.WorkflowDefinition{}, fmt.Errorf("storage: GetWorkflow unmarshal: %w", err)
	}
	return def, nil
}

// ListWorkflows returns workflow definitions. A non-empty namespace restricts
// results to that namespace; an empty namespace adds no predicate and returns
// definitions across ALL namespaces (B-25) — this mirrors the
// ListInstances/InstanceFilter convention elsewhere in this file, where a
// zero-valued filter field means "no predicate", not "match the empty
// string". Callers that want only rows whose namespace is literally the
// empty string have no way to express that with this method; none do today.
func (s *SQLiteStorage) ListWorkflows(ctx context.Context, namespace string) ([]core.WorkflowDefinition, error) {
	query := `SELECT definition FROM workflow_definitions WHERE 1=1`
	var args []any
	if namespace != "" {
		query += " AND namespace = ?"
		args = append(args, namespace)
	}

	rows, err := s.db.db.QueryContext(ctx, query, args...)
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

// ── StateStore ────────────────────────────────────────────────────────────────

// UpsertInstance writes a WorkflowInstance with optimistic concurrency control.
//
// Algorithm (M03 / CONTRA-6 / EDR-006):
//   - Attempt UPDATE ... WHERE instance_id=? AND version=expectedVersion,
//     setting version=expectedVersion+1 and all user fields.
//   - If 0 rows affected AND expectedVersion==0 AND no row exists → INSERT with
//     version=1.
//   - Otherwise 0 rows affected → typed ErrVersionConflict (wraps instance id +
//     expected version).
//   - If the resulting status is terminal (completed|failed|cancelled|compensated|
//     compensation_failed) the instance's step_claims rows are deleted in the
//     same transaction (EDR-006 release site).
//
// Times are stored as RFC3339Nano UTC (matches AppendEvent convention).
// current_steps and variables are stored as JSON; completed_at is nullable.
// cancellation_requested is DB-internal and is not modified here.
func (s *SQLiteStorage) UpsertInstance(ctx context.Context, instance core.WorkflowInstance, expectedVersion int) error {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage: UpsertInstance begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Marshal JSON fields; normalise nil slices/maps.
	currentSteps := instance.CurrentSteps
	if currentSteps == nil {
		currentSteps = []string{}
	}
	currentStepsJSON, err := json.Marshal(currentSteps)
	if err != nil {
		return fmt.Errorf("storage: UpsertInstance marshal current_steps: %w", err)
	}

	variables := instance.Variables
	if variables == nil {
		variables = map[string]any{}
	}
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return fmt.Errorf("storage: UpsertInstance marshal variables: %w", err)
	}

	startedAt := instance.StartedAt.UTC().Format(time.RFC3339Nano)
	updatedAt := instance.UpdatedAt.UTC().Format(time.RFC3339Nano)

	var completedAt *string
	if instance.CompletedAt != nil {
		v := instance.CompletedAt.UTC().Format(time.RFC3339Nano)
		completedAt = &v
	}

	// Attempt UPDATE guarded by version.
	res, err := tx.ExecContext(ctx, `
		UPDATE workflow_instances SET
			definition_id      = ?,
			definition_version = ?,
			namespace          = ?,
			status             = ?,
			current_steps      = ?,
			variables          = ?,
			started_at         = ?,
			updated_at         = ?,
			completed_at       = ?,
			version            = ?
		WHERE instance_id = ? AND version = ?`,
		instance.DefinitionID,
		string(instance.DefinitionVersion),
		instance.Namespace,
		string(instance.Status),
		string(currentStepsJSON),
		string(variablesJSON),
		startedAt,
		updatedAt,
		completedAt,
		expectedVersion+1,
		string(instance.InstanceID),
		expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("storage: UpsertInstance update: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: UpsertInstance rows affected: %w", err)
	}

	if n == 0 {
		// Determine whether to INSERT or conflict.
		if expectedVersion == 0 {
			var exists int
			if err := tx.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM workflow_instances WHERE instance_id = ?`,
				string(instance.InstanceID),
			).Scan(&exists); err != nil {
				return fmt.Errorf("storage: UpsertInstance existence check: %w", err)
			}
			if exists > 0 {
				return fmt.Errorf("%w: instance_id=%s expected=%d",
					ErrVersionConflict, instance.InstanceID, expectedVersion)
			}
			// Row absent → INSERT with version=1.
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO workflow_instances
					(instance_id, definition_id, definition_version, namespace, status,
					 current_steps, variables, started_at, updated_at, completed_at,
					 version, cancellation_requested)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0)`,
				string(instance.InstanceID),
				instance.DefinitionID,
				string(instance.DefinitionVersion),
				instance.Namespace,
				string(instance.Status),
				string(currentStepsJSON),
				string(variablesJSON),
				startedAt,
				updatedAt,
				completedAt,
			); err != nil {
				return fmt.Errorf("storage: UpsertInstance insert: %w", err)
			}
		} else {
			return fmt.Errorf("%w: instance_id=%s expected=%d",
				ErrVersionConflict, instance.InstanceID, expectedVersion)
		}
	}

	// On terminal status: release all step_claims for this instance (EDR-006).
	if isTerminalStatus(instance.Status) {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM step_claims WHERE instance_id = ?`,
			string(instance.InstanceID),
		); err != nil {
			return fmt.Errorf("storage: UpsertInstance delete step_claims: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage: UpsertInstance commit: %w", err)
	}
	return nil
}

// GetInstance fetches a WorkflowInstance by id. Returns ErrInstanceNotFound
// (typed, use errors.Is) when absent. All 10 user-visible fields are mapped
// faithfully; CompletedAt is nil when the DB column is NULL; times are UTC.
func (s *SQLiteStorage) GetInstance(ctx context.Context, instanceID core.InstanceID) (core.WorkflowInstance, error) {
	var (
		idStr, defID, defVer, ns, status string
		stepsJSON, varsJSON              string
		startedAtStr, updatedAtStr       string
		completedAtStr                   sql.NullString
		version                          int // DB-internal; not exposed
	)

	err := s.db.db.QueryRowContext(ctx, `
		SELECT instance_id, definition_id, definition_version, namespace, status,
		       current_steps, variables, started_at, updated_at, completed_at, version
		FROM workflow_instances
		WHERE instance_id = ?`,
		string(instanceID),
	).Scan(&idStr, &defID, &defVer, &ns, &status,
		&stepsJSON, &varsJSON,
		&startedAtStr, &updatedAtStr, &completedAtStr,
		&version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.WorkflowInstance{}, fmt.Errorf("%w: %s", ErrInstanceNotFound, instanceID)
		}
		return core.WorkflowInstance{}, fmt.Errorf("storage: GetInstance query: %w", err)
	}

	var inst core.WorkflowInstance
	inst.InstanceID = core.InstanceID(idStr)
	inst.DefinitionID = defID
	inst.DefinitionVersion = core.SemVer(defVer)
	inst.Namespace = ns
	inst.Status = core.InstanceStatus(status)

	if err := json.Unmarshal([]byte(stepsJSON), &inst.CurrentSteps); err != nil {
		return core.WorkflowInstance{}, fmt.Errorf("storage: GetInstance unmarshal current_steps: %w", err)
	}
	if err := json.Unmarshal([]byte(varsJSON), &inst.Variables); err != nil {
		return core.WorkflowInstance{}, fmt.Errorf("storage: GetInstance unmarshal variables: %w", err)
	}

	startedAt, err := parseTimeStr(startedAtStr)
	if err != nil {
		return core.WorkflowInstance{}, fmt.Errorf("storage: GetInstance parse started_at: %w", err)
	}
	inst.StartedAt = startedAt

	updatedAt, err := parseTimeStr(updatedAtStr)
	if err != nil {
		return core.WorkflowInstance{}, fmt.Errorf("storage: GetInstance parse updated_at: %w", err)
	}
	inst.UpdatedAt = updatedAt

	if completedAtStr.Valid {
		t, err := parseTimeStr(completedAtStr.String)
		if err != nil {
			return core.WorkflowInstance{}, fmt.Errorf("storage: GetInstance parse completed_at: %w", err)
		}
		inst.CompletedAt = &t
	}

	return inst, nil
}

// InstanceVersion returns the optimistic-lock version of an instance row
// (engine-hardening B-0: the durable version, read fresh from storage, is what
// makes UpsertInstance's OCC check correct across a process restart — the
// engine's in-memory e.ver cache is empty on a fresh process while this column
// is not). Returns ErrInstanceNotFound when the row is absent.
func (s *SQLiteStorage) InstanceVersion(ctx context.Context, instanceID core.InstanceID) (int, error) {
	var version int
	err := s.db.db.QueryRowContext(ctx,
		`SELECT version FROM workflow_instances WHERE instance_id = ?`,
		string(instanceID),
	).Scan(&version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%w: %s", ErrInstanceNotFound, instanceID)
		}
		return 0, fmt.Errorf("storage: InstanceVersion query: %w", err)
	}
	return version, nil
}

// ListInstances returns instances matching filter. Fields with zero values add
// no WHERE predicate; non-zero Namespace or Status each add one. Results are
// ordered by started_at ascending (NFR-S-04 namespace predicate via
// idx_instances_ns_status).
func (s *SQLiteStorage) ListInstances(ctx context.Context, filter core.InstanceFilter) ([]core.WorkflowInstance, error) {
	where, args := instanceFilterPredicate(filter)
	query := `
		SELECT instance_id, definition_id, definition_version, namespace, status,
		       current_steps, variables, started_at, updated_at, completed_at
		FROM workflow_instances
		` + where + `
		ORDER BY started_at`

	rows, err := s.db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: ListInstances query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var instances []core.WorkflowInstance
	for rows.Next() {
		var (
			idStr, defID, defVer, ns, status string
			stepsJSON, varsJSON              string
			startedAtStr, updatedAtStr       string
			completedAtStr                   sql.NullString
		)
		if err := rows.Scan(
			&idStr, &defID, &defVer, &ns, &status,
			&stepsJSON, &varsJSON,
			&startedAtStr, &updatedAtStr, &completedAtStr,
		); err != nil {
			return nil, fmt.Errorf("storage: ListInstances scan: %w", err)
		}

		var inst core.WorkflowInstance
		inst.InstanceID = core.InstanceID(idStr)
		inst.DefinitionID = defID
		inst.DefinitionVersion = core.SemVer(defVer)
		inst.Namespace = ns
		inst.Status = core.InstanceStatus(status)

		if err := json.Unmarshal([]byte(stepsJSON), &inst.CurrentSteps); err != nil {
			return nil, fmt.Errorf("storage: ListInstances unmarshal current_steps: %w", err)
		}
		if err := json.Unmarshal([]byte(varsJSON), &inst.Variables); err != nil {
			return nil, fmt.Errorf("storage: ListInstances unmarshal variables: %w", err)
		}

		startedAt, err := parseTimeStr(startedAtStr)
		if err != nil {
			return nil, fmt.Errorf("storage: ListInstances parse started_at: %w", err)
		}
		inst.StartedAt = startedAt

		updatedAt, err := parseTimeStr(updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("storage: ListInstances parse updated_at: %w", err)
		}
		inst.UpdatedAt = updatedAt

		if completedAtStr.Valid {
			t, err := parseTimeStr(completedAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("storage: ListInstances parse completed_at: %w", err)
			}
			inst.CompletedAt = &t
		}

		instances = append(instances, inst)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListInstances rows: %w", err)
	}
	return instances, nil
}

// Pagination bounds for ListInstancesPaged (B-17). defaultInstancesPageSize
// applies when a caller passes limit <= 0; maxInstancesPageSize is a hard
// ceiling so a caller cannot request the entire table in one page.
const (
	defaultInstancesPageSize = 100
	maxInstancesPageSize     = 1000
)

// ListInstancesPaged returns at most limit instances matching filter, skipping
// offset rows, ordered deterministically (B-17).
//
// limit <= 0 is normalised to defaultInstancesPageSize (100); limit above
// maxInstancesPageSize (1000) is clamped down to it, so a caller can never
// force an unbounded scan through this method. offset beyond the end of the
// result set returns an empty (not error) slice, matching normal SQL
// LIMIT/OFFSET semantics.
//
// Ordering is "ORDER BY started_at DESC, instance_id DESC": started_at alone is not a
// unique key (multiple instances can share the same started_at value,
// especially with a coarse or injected clock), so a tiebreak on the unique
// instance_id is required for pagination to be stable — without it, a row
// with a duplicate started_at could be skipped or repeated across pages
// depending on how SQLite happens to order ties.
//
// This method is purely additive: it does not change ListInstances, which
// remains unbounded (the engine's tick loop depends on every running
// instance being returned, not a page of them). Both methods share the
// unexported instanceFilterPredicate query builder so their WHERE-clause
// semantics can never drift apart.
func (s *SQLiteStorage) ListInstancesPaged(ctx context.Context, filter core.InstanceFilter, limit, offset int) ([]core.WorkflowInstance, error) {
	if limit <= 0 {
		limit = defaultInstancesPageSize
	}
	if limit > maxInstancesPageSize {
		limit = maxInstancesPageSize
	}
	if offset < 0 {
		offset = 0
	}

	where, args := instanceFilterPredicate(filter)
	query := `
		SELECT instance_id, definition_id, definition_version, namespace, status,
		       current_steps, variables, started_at, updated_at, completed_at
		FROM workflow_instances
		` + where + `
		ORDER BY started_at DESC, instance_id DESC
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: ListInstancesPaged query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var instances []core.WorkflowInstance
	for rows.Next() {
		var (
			idStr, defID, defVer, ns, status string
			stepsJSON, varsJSON              string
			startedAtStr, updatedAtStr       string
			completedAtStr                   sql.NullString
		)
		if err := rows.Scan(
			&idStr, &defID, &defVer, &ns, &status,
			&stepsJSON, &varsJSON,
			&startedAtStr, &updatedAtStr, &completedAtStr,
		); err != nil {
			return nil, fmt.Errorf("storage: ListInstancesPaged scan: %w", err)
		}

		var inst core.WorkflowInstance
		inst.InstanceID = core.InstanceID(idStr)
		inst.DefinitionID = defID
		inst.DefinitionVersion = core.SemVer(defVer)
		inst.Namespace = ns
		inst.Status = core.InstanceStatus(status)

		if err := json.Unmarshal([]byte(stepsJSON), &inst.CurrentSteps); err != nil {
			return nil, fmt.Errorf("storage: ListInstancesPaged unmarshal current_steps: %w", err)
		}
		if err := json.Unmarshal([]byte(varsJSON), &inst.Variables); err != nil {
			return nil, fmt.Errorf("storage: ListInstancesPaged unmarshal variables: %w", err)
		}

		startedAt, err := parseTimeStr(startedAtStr)
		if err != nil {
			return nil, fmt.Errorf("storage: ListInstancesPaged parse started_at: %w", err)
		}
		inst.StartedAt = startedAt

		updatedAt, err := parseTimeStr(updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("storage: ListInstancesPaged parse updated_at: %w", err)
		}
		inst.UpdatedAt = updatedAt

		if completedAtStr.Valid {
			t, err := parseTimeStr(completedAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("storage: ListInstancesPaged parse completed_at: %w", err)
			}
			inst.CompletedAt = &t
		}

		instances = append(instances, inst)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListInstancesPaged rows: %w", err)
	}
	return instances, nil
}

// CountInstances returns the number of instances matching filter, using the
// same predicate semantics as ListInstances/ListInstancesPaged (B-17) so a
// paginated consumer can compute a total-pages figure.
func (s *SQLiteStorage) CountInstances(ctx context.Context, filter core.InstanceFilter) (int, error) {
	where, args := instanceFilterPredicate(filter)
	query := `SELECT COUNT(*) FROM workflow_instances ` + where

	var count int
	if err := s.db.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("storage: CountInstances query: %w", err)
	}
	return count, nil
}

// ClaimStep atomically claims a step for a worker using the step_claims table
// as an at-most-once gate (CONTRA-6 / EDR-006 / Blueprint §8 step 3).
//
// Single transaction semantics:
//  1. Verify the instance exists (else return error).
//  2. INSERT INTO step_claims — PK conflict means another worker already holds
//     the claim; return (false, nil).
//  3. On successful INSERT bump workflow_instances.version by 1 (optimistic-lock
//     coupling so a concurrent UpsertInstance sees the version change).
//  4. Commit → (true, nil).
//
// claimed_at is sourced from the injectable clock.
func (s *SQLiteStorage) ClaimStep(ctx context.Context, instanceID core.InstanceID, stepID string, workerID string) (bool, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("storage: ClaimStep begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Step 1: instance must exist.
	var exists int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM workflow_instances WHERE instance_id = ?`,
		string(instanceID),
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("storage: ClaimStep existence check: %w", err)
	}
	if exists == 0 {
		return false, fmt.Errorf("storage: ClaimStep: instance %q not found", instanceID)
	}

	// Step 2: attempt INSERT.
	claimedAt := s.now().UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO step_claims (instance_id, step_id, worker_id, claimed_at)
		VALUES (?, ?, ?, ?)`,
		string(instanceID), stepID, workerID, claimedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			// Another worker already holds the claim.
			return false, nil
		}
		return false, fmt.Errorf("storage: ClaimStep insert: %w", err)
	}

	// Step 3: bump instance version.
	if _, err := tx.ExecContext(ctx,
		`UPDATE workflow_instances SET version = version + 1 WHERE instance_id = ?`,
		string(instanceID),
	); err != nil {
		return false, fmt.Errorf("storage: ClaimStep version bump: %w", err)
	}

	// Step 4: commit.
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("storage: ClaimStep commit: %w", err)
	}
	return true, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// instanceFilterPredicate builds the WHERE clause (as "WHERE 1=1 [AND ...]",
// so callers can always append " ORDER BY ..." or " LIMIT ..." directly) and
// its positional args for filter, per InstanceFilter's documented "zero value
// means no predicate" semantics. Shared by ListInstances, ListInstancesPaged,
// and CountInstances (B-17) so the three can never apply different
// predicates for the same filter value.
func instanceFilterPredicate(filter core.InstanceFilter) (string, []any) {
	where := "WHERE 1=1"
	var args []any
	if filter.Namespace != "" {
		where += " AND namespace = ?"
		args = append(args, filter.Namespace)
	}
	if filter.Status != "" {
		where += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	if filter.DefinitionID != "" {
		where += " AND definition_id = ?"
		args = append(args, filter.DefinitionID)
	}
	return where, args
}

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

// isTerminalStatus reports whether status is a terminal workflow lifecycle
// state. Terminal instances have their step_claims released on upsert (EDR-006).
func isTerminalStatus(status core.InstanceStatus) bool {
	switch status {
	case core.InstanceStatusCompleted,
		core.InstanceStatusFailed,
		core.InstanceStatusCancelled,
		core.InstanceStatusCompensated,
		core.InstanceStatusCompensationFailed:
		return true
	}
	return false
}

// parseTimeStr parses a time string stored by the adapter (RFC3339Nano or RFC3339).
func parseTimeStr(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse time %q: %w", s, err)
		}
	}
	return t.UTC(), nil
}
