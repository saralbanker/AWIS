package storage

// Signal-inbox and wait-record storage accessors (M07-C1, F-4).
//
// StoragePort's 12 methods are frozen (Blueprint §20). The signal subsystem's
// tables — signal_inbox and wait_records, added by migration 0003 — are exposed
// via ADDITIVE *SQLiteStorage methods, NOT new StoragePort methods, exactly as
// the cancellation flag (cancellation.go) and trigger store (triggers.go) are.
// C2 (delivery, B3 transaction) and C3 (timeout logic) build on these; this
// card lands the storage foundation only — no engine changes, no delivery tx,
// no timeout scan loop.
//
// All times are persisted as-supplied by the caller in RFC3339Nano UTC — the
// AppendEvent / InsertDomainEvent convention. The caller (engine) owns the
// injectable clock (IMP §3 determinism rule); this layer is a faithful
// persistence surface and does not stamp domain times of its own.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// ErrDuplicateSignal is returned by InsertSignal when a row with the same
// signal_id already exists. It mirrors ErrDuplicateDomainEvent: signal intake is
// NOT idempotent-skip at the storage layer — a duplicate signal_id is a typed,
// caller-observable error so a re-emitting source is surfaced rather than
// silently swallowed. The B3 delivery transaction (C2) decides how to treat it.
var ErrDuplicateSignal = errors.New("storage: signal already in inbox")

// ErrDuplicateWaitRecord is returned by CreateWaitRecord when a row with the
// same (instance_id, step_id) primary key already exists. Re-entering a wait for
// an already-waiting step is a typed error, not a silent overwrite (C3 owns
// re-entry semantics).
var ErrDuplicateWaitRecord = errors.New("storage: wait record already exists")

// Signal is one row of the signal_inbox table (migration 0003). It is a
// storage-package projection, not a frozen core type: the signal subsystem's
// row shape is owned by M07 (F-4), not part of the G1 format freeze.
type Signal struct {
	// SignalID is the UUID primary key uniquely identifying the signal.
	SignalID string
	// InstanceID is the target waiting instance.
	InstanceID core.InstanceID
	// SignalName is the awaited signal's name (matched against a wait record).
	SignalName string
	// Payload is the signal's data, delivered into the waiting step.
	Payload map[string]any
	// DeliveredAt is when delivery completed; nil while the signal is undelivered.
	DeliveredAt *time.Time
	// ReceivedAt is when the signal was received into the inbox (caller-supplied).
	ReceivedAt time.Time
}

// WaitRecord is one row of the wait_records table (migration 0003). It records a
// step that is parked awaiting a named signal, with an optional timeout. It is a
// storage-package projection, not a frozen core type (M07 owns the shape, F-4).
type WaitRecord struct {
	// InstanceID is the waiting instance (part of the primary key).
	InstanceID core.InstanceID
	// StepID is the waiting step (part of the primary key).
	StepID string
	// SignalName is the awaited signal's name.
	SignalName string
	// CreatedAt is when the wait began (caller-supplied).
	CreatedAt time.Time
	// TimeoutAt is when the wait times out; nil for an unbounded wait.
	TimeoutAt *time.Time
	// TimeoutAction is the action taken on timeout (e.g. fail, compensate,
	// continue), mirroring core.WaitConfig.TimeoutAction.
	TimeoutAction string
}

// ── signal_inbox ────────────────────────────────────────────────────────────

// InsertSignal persists sig into signal_inbox (intake). delivered_at is written
// from sig.DeliveredAt (NULL for a fresh, undelivered signal); received_at from
// sig.ReceivedAt. payload is JSON-marshalled. A duplicate signal_id returns
// ErrDuplicateSignal. The B3 delivery transaction that flips delivered_at lives
// in C2 — this method never sets it beyond the caller-supplied value.
func (s *SQLiteStorage) InsertSignal(ctx context.Context, sig Signal) error {
	payload := sig.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("storage: InsertSignal marshal payload: %w", err)
	}

	var deliveredAt *string
	if sig.DeliveredAt != nil {
		d := sig.DeliveredAt.UTC().Format(time.RFC3339Nano)
		deliveredAt = &d
	}

	_, err = s.db.db.ExecContext(ctx, `
		INSERT INTO signal_inbox (signal_id, instance_id, signal_name, payload, delivered_at, received_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		sig.SignalID,
		string(sig.InstanceID),
		sig.SignalName,
		string(data),
		deliveredAt,
		sig.ReceivedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("%w: signal_id=%s", ErrDuplicateSignal, sig.SignalID)
		}
		return fmt.Errorf("storage: InsertSignal insert: %w", err)
	}
	return nil
}

// ListUndeliveredSignals returns every signal with delivered_at NULL, ordered by
// (received_at, signal_id) for a deterministic scan. An empty instanceID matches
// ALL instances (the subsystem-wide undelivered scan C2/C3 use); a non-empty
// instanceID restricts to that instance. Because instance_id is a globally
// unique UUID bound to one namespace, per-instance scoping is inherently
// namespace-respecting (NFR-S-04): a signal never crosses the instance boundary.
func (s *SQLiteStorage) ListUndeliveredSignals(ctx context.Context, instanceID core.InstanceID) ([]Signal, error) {
	id := string(instanceID)
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT signal_id, instance_id, signal_name, payload, delivered_at, received_at
		FROM signal_inbox
		WHERE delivered_at IS NULL AND (? = '' OR instance_id = ?)
		ORDER BY received_at, signal_id`,
		id, id,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: ListUndeliveredSignals query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Signal
	for rows.Next() {
		var (
			sig           Signal
			instIDStr     string
			payloadStr    string
			deliveredAt   sql.NullString
			receivedAtStr string
		)
		if err := rows.Scan(&sig.SignalID, &instIDStr, &sig.SignalName,
			&payloadStr, &deliveredAt, &receivedAtStr); err != nil {
			return nil, fmt.Errorf("storage: ListUndeliveredSignals scan: %w", err)
		}
		sig.InstanceID = core.InstanceID(instIDStr)
		if err := json.Unmarshal([]byte(payloadStr), &sig.Payload); err != nil {
			return nil, fmt.Errorf("storage: ListUndeliveredSignals unmarshal payload: %w", err)
		}
		receivedAt, err := parseTimeStr(receivedAtStr)
		if err != nil {
			return nil, fmt.Errorf("storage: ListUndeliveredSignals received_at: %w", err)
		}
		sig.ReceivedAt = receivedAt
		if deliveredAt.Valid {
			d, err := parseTimeStr(deliveredAt.String)
			if err != nil {
				return nil, fmt.Errorf("storage: ListUndeliveredSignals delivered_at: %w", err)
			}
			sig.DeliveredAt = &d
		}
		out = append(out, sig)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListUndeliveredSignals rows: %w", err)
	}
	return out, nil
}

// ── wait_records ────────────────────────────────────────────────────────────

// CreateWaitRecord persists wr into wait_records. created_at and timeout_at are
// caller-supplied (timeout_at NULL for an unbounded wait). A duplicate
// (instance_id, step_id) returns ErrDuplicateWaitRecord.
func (s *SQLiteStorage) CreateWaitRecord(ctx context.Context, wr WaitRecord) error {
	var timeoutAt *string
	if wr.TimeoutAt != nil {
		t := wr.TimeoutAt.UTC().Format(time.RFC3339Nano)
		timeoutAt = &t
	}

	_, err := s.db.db.ExecContext(ctx, `
		INSERT INTO wait_records (instance_id, step_id, signal_name, created_at, timeout_at, timeout_action)
		VALUES (?, ?, ?, ?, ?, ?)`,
		string(wr.InstanceID),
		wr.StepID,
		wr.SignalName,
		wr.CreatedAt.UTC().Format(time.RFC3339Nano),
		timeoutAt,
		wr.TimeoutAction,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("%w: instance_id=%s step_id=%s", ErrDuplicateWaitRecord, wr.InstanceID, wr.StepID)
		}
		return fmt.Errorf("storage: CreateWaitRecord insert: %w", err)
	}
	return nil
}

// ListDueWaitRecords returns every wait record whose timeout_at is non-NULL and
// at or before now, ordered by (timeout_at, instance_id, step_id) for a
// deterministic scan. It reads through idx_wait_timeout (the partial index over
// non-NULL timeout_at); unbounded waits (timeout_at NULL) are never returned.
// The timeout ACTION taken on a due record is C3's concern — this method only
// enumerates.
func (s *SQLiteStorage) ListDueWaitRecords(ctx context.Context, now time.Time) ([]WaitRecord, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT instance_id, step_id, signal_name, created_at, timeout_at, timeout_action
		FROM wait_records
		WHERE timeout_at IS NOT NULL AND timeout_at <= ?
		ORDER BY timeout_at, instance_id, step_id`,
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: ListDueWaitRecords query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []WaitRecord
	for rows.Next() {
		wr, err := scanWaitRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListDueWaitRecords rows: %w", err)
	}
	return out, nil
}

// GetWaitRecord fetches the wait record for (instanceID, stepID). It returns
// (record, true, nil) when present and (zero, false, nil) when absent — the
// GetCachedResult idiom, since an absent wait record is a legitimate state (the
// step is not waiting), not a fault.
func (s *SQLiteStorage) GetWaitRecord(ctx context.Context, instanceID core.InstanceID, stepID string) (WaitRecord, bool, error) {
	row := s.db.db.QueryRowContext(ctx, `
		SELECT instance_id, step_id, signal_name, created_at, timeout_at, timeout_action
		FROM wait_records
		WHERE instance_id = ? AND step_id = ?`,
		string(instanceID), stepID,
	)
	wr, err := scanWaitRecordRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WaitRecord{}, false, nil
	}
	if err != nil {
		return WaitRecord{}, false, err
	}
	return wr, true, nil
}

// ListWaitRecordsByInstance returns every live wait record for instanceID
// (engine-hardening B-15: the scanner resolves a wait by signal_name, not
// step_id, so GetWaitRecord's step_id key cannot serve a durable-recovery
// lookup after a restart — this method lets the engine scan its instance's
// live waits and match on signal_name itself). Ordered by step_id for a
// deterministic result; an instance with no live wait returns (nil, nil).
func (s *SQLiteStorage) ListWaitRecordsByInstance(ctx context.Context, instanceID core.InstanceID) ([]WaitRecord, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT instance_id, step_id, signal_name, created_at, timeout_at, timeout_action
		FROM wait_records
		WHERE instance_id = ?
		ORDER BY step_id`,
		string(instanceID),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: ListWaitRecordsByInstance query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []WaitRecord
	for rows.Next() {
		wr, err := scanWaitRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListWaitRecordsByInstance rows: %w", err)
	}
	return out, nil
}

// DeleteWaitRecordsByInstance removes every wait record for instanceID. It is a
// benign no-op when the instance has none (returns nil): wait records are
// cleared on signal delivery / timeout / terminal transition, and a
// clear-when-empty is not a fault.
func (s *SQLiteStorage) DeleteWaitRecordsByInstance(ctx context.Context, instanceID core.InstanceID) error {
	if _, err := s.db.db.ExecContext(ctx,
		`DELETE FROM wait_records WHERE instance_id = ?`,
		string(instanceID),
	); err != nil {
		return fmt.Errorf("storage: DeleteWaitRecordsByInstance delete: %w", err)
	}
	return nil
}

// DeleteWaitRecord removes the wait_record for (instanceID, stepID). It is a
// benign no-op when the record is absent: a record may have been cleared by a
// concurrent timeout or cancellation before delivery completes (M07-C3r).
func (s *SQLiteStorage) DeleteWaitRecord(ctx context.Context, instanceID core.InstanceID, stepID string) error {
	if _, err := s.db.db.ExecContext(ctx,
		`DELETE FROM wait_records WHERE instance_id = ? AND step_id = ?`,
		string(instanceID), stepID,
	); err != nil {
		return fmt.Errorf("storage: DeleteWaitRecord delete: %w", err)
	}
	return nil
}

// scanWaitRecord scans a *sql.Rows cursor into a WaitRecord.
func scanWaitRecord(rows *sql.Rows) (WaitRecord, error) {
	var (
		wr           WaitRecord
		instIDStr    string
		createdAtStr string
		timeoutAt    sql.NullString
	)
	if err := rows.Scan(&instIDStr, &wr.StepID, &wr.SignalName,
		&createdAtStr, &timeoutAt, &wr.TimeoutAction); err != nil {
		return WaitRecord{}, fmt.Errorf("storage: scan wait record: %w", err)
	}
	return hydrateWaitRecord(wr, instIDStr, createdAtStr, timeoutAt)
}

// scanWaitRecordRow scans a *sql.Row (single-row) into a WaitRecord. The
// sql.ErrNoRows sentinel is propagated so callers can distinguish absence.
func scanWaitRecordRow(row *sql.Row) (WaitRecord, error) {
	var (
		wr           WaitRecord
		instIDStr    string
		createdAtStr string
		timeoutAt    sql.NullString
	)
	if err := row.Scan(&instIDStr, &wr.StepID, &wr.SignalName,
		&createdAtStr, &timeoutAt, &wr.TimeoutAction); err != nil {
		return WaitRecord{}, err
	}
	return hydrateWaitRecord(wr, instIDStr, createdAtStr, timeoutAt)
}

// hydrateWaitRecord fills the parsed/typed fields shared by both scan paths.
func hydrateWaitRecord(wr WaitRecord, instIDStr, createdAtStr string, timeoutAt sql.NullString) (WaitRecord, error) {
	wr.InstanceID = core.InstanceID(instIDStr)
	createdAt, err := parseTimeStr(createdAtStr)
	if err != nil {
		return WaitRecord{}, fmt.Errorf("storage: wait record created_at: %w", err)
	}
	wr.CreatedAt = createdAt
	if timeoutAt.Valid {
		t, err := parseTimeStr(timeoutAt.String)
		if err != nil {
			return WaitRecord{}, fmt.Errorf("storage: wait record timeout_at: %w", err)
		}
		wr.TimeoutAt = &t
	}
	return wr, nil
}
