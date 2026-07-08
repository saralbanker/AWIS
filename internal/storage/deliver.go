package storage

// Atomic signal delivery — Finalization Blocker 3 (Option A), transcribed
// VERBATIM as ONE transaction (M07-C2, F-4).
//
// DeliverSignal performs the three-write B3 transaction: (1) an idempotency
// guard that flips the inbox row's delivered_at, (2) the SignalReceived append
// to the EventLog (sequence assigned inside per EDR-005), and (3) the
// optimistic-lock resume that moves the instance waiting→running. The guards
// give the two invariants B3 mandates:
//
//   - guard finds no row  ⇒ whole tx is a no-op (idempotent);              (B3)
//   - step-3 lock fails   ⇒ ROLLBACK; the inbox entry stays undelivered and
//     the next scan re-attempts.                                          (B3)
//
// Invariant: SignalReceived in EventLog ⇔ delivered (B3). Because all three
// writes share one transaction, a crash between them leaves the log and the
// inbox consistent — never a delivered inbox row without its SignalReceived
// event, nor the reverse.
//
// StoragePort's 12 methods stay frozen (Blueprint §20); DeliverSignal is an
// ADDITIVE *SQLiteStorage method, reached by the engine through a type-asserted
// interface exactly as cancellation.go / triggers.go are.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// DeliverInput carries the parameters of one B3 signal-delivery transaction.
// EventPayload is the fully-built SignalReceived payload ({signal_name, payload}
// per EVENTLOG_FORMAT §6), marshalled by the caller (the delivery scan) from the
// inbox row. Now is used for BOTH the event's emitted_at AND the instance's
// updated_at so the forward projection matches what RebuildState reconstructs
// from the SignalReceived event (EDR-007 §2; forward ≡ rebuild).
type DeliverInput struct {
	// InstanceID is the target waiting instance.
	InstanceID core.InstanceID
	// SignalName keys the idempotency guard (B3 step 1).
	SignalName string
	// EventPayload is the SignalReceived payload bytes ({signal_name, payload}).
	EventPayload json.RawMessage
	// ExpectedVersion is the optimistic-lock version the resume requires (B3 step 3).
	ExpectedVersion int
	// EventID is the UUID of the SignalReceived event (caller-minted, EDR-005).
	EventID string
	// Now is the single delivery timestamp (delivered_at / emitted_at / updated_at).
	Now time.Time
}

// DeliverOutcome enumerates the three terminal outcomes of a B3 delivery attempt.
type DeliverOutcome int

const (
	// DeliverNoOp: the idempotency guard matched no undelivered row — the signal
	// was already delivered (or never in the inbox). The tx made no lasting
	// change (B3 idempotency).
	DeliverNoOp DeliverOutcome = iota
	// DeliverLockConflict: the guard matched but the step-3 optimistic lock did
	// not (version skew or status≠waiting). The tx is ROLLED BACK; the inbox entry
	// stays undelivered for the next scan (B3).
	DeliverLockConflict
	// Delivered: all three writes committed; the instance is running and the
	// SignalReceived event is durable.
	Delivered
)

// DeliverSignal executes the Finalization B3 transaction VERBATIM. It never
// bypasses or reorders the three writes: guard (UPDATE signal_inbox) → append
// (INSERT execution_events) → resume (UPDATE workflow_instances), in that order,
// with both guards.
//
// semantics-bearing: single tx per Finalization B3 (guard → append → optimistic
// lock; SignalReceived in EventLog ⇔ delivered).
func (s *SQLiteStorage) DeliverSignal(ctx context.Context, in DeliverInput) (DeliverOutcome, error) {
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return DeliverNoOp, fmt.Errorf("storage: DeliverSignal begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	nowStr := in.Now.UTC().Format(time.RFC3339Nano)

	// ── B3 step 1: idempotency guard ──────────────────────────────────────────
	// UPDATE signal_inbox SET delivered_at = <now>
	//   WHERE instance_id = <id> AND signal_name = <name> AND delivered_at IS NULL
	res1, err := tx.ExecContext(ctx, `
		UPDATE signal_inbox SET delivered_at = ?
		WHERE instance_id = ? AND signal_name = ? AND delivered_at IS NULL`,
		nowStr, string(in.InstanceID), in.SignalName,
	)
	if err != nil {
		return DeliverNoOp, fmt.Errorf("storage: DeliverSignal guard: %w", err)
	}
	n1, err := res1.RowsAffected()
	if err != nil {
		return DeliverNoOp, fmt.Errorf("storage: DeliverSignal guard rows: %w", err)
	}
	if n1 == 0 {
		// Guard finds no row ⇒ whole tx is a no-op (idempotent) — B3. The deferred
		// Rollback discards the (empty) tx.
		return DeliverNoOp, nil
	}

	// The event's namespace is not carried by the signal; read it from the
	// instance row (also confirms the instance exists). The next sequence_num is
	// MAX+1 per instance, assigned inside this tx (EDR-005). Both are reads, not
	// among B3's three writes.
	var namespace string
	if err := tx.QueryRowContext(ctx,
		`SELECT namespace FROM workflow_instances WHERE instance_id = ?`,
		string(in.InstanceID),
	).Scan(&namespace); err != nil {
		return DeliverNoOp, fmt.Errorf("storage: DeliverSignal read namespace: %w", err)
	}
	var maxSeq sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(sequence_num) FROM execution_events WHERE instance_id = ?`,
		string(in.InstanceID),
	).Scan(&maxSeq); err != nil {
		return DeliverNoOp, fmt.Errorf("storage: DeliverSignal read max seq: %w", err)
	}
	seqNum := 1
	if maxSeq.Valid {
		seqNum = int(maxSeq.Int64) + 1
	}

	payload := []byte(in.EventPayload)
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	// ── B3 step 2: append SignalReceived to the EventLog ──────────────────────
	// INSERT INTO execution_events (... 'SignalReceived', <payload>, ...).
	// step_id is NULL: SignalReceived is a workflow-level event (§9).
	var stepID *string
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO execution_events
			(event_id, instance_id, namespace, event_type, step_id, payload,
			 emitted_at, sequence_num, schema_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.EventID,
		string(in.InstanceID),
		namespace,
		string(core.EventTypeSignalReceived),
		stepID,
		string(payload),
		nowStr,
		seqNum,
		1,
	); err != nil {
		return DeliverNoOp, fmt.Errorf("storage: DeliverSignal append event: %w", err)
	}

	// ── B3 step 3: optimistic-lock resume waiting→running ─────────────────────
	// UPDATE workflow_instances SET status='running', updated_at=<now>,
	//   version=version+1 WHERE instance_id=<id> AND status='waiting'
	//   AND version=<expected_version>.
	res3, err := tx.ExecContext(ctx, `
		UPDATE workflow_instances
		SET status = 'running', updated_at = ?, version = version + 1
		WHERE instance_id = ? AND status = 'waiting' AND version = ?`,
		nowStr, string(in.InstanceID), in.ExpectedVersion,
	)
	if err != nil {
		return DeliverNoOp, fmt.Errorf("storage: DeliverSignal resume: %w", err)
	}
	n3, err := res3.RowsAffected()
	if err != nil {
		return DeliverNoOp, fmt.Errorf("storage: DeliverSignal resume rows: %w", err)
	}
	if n3 == 0 {
		// Step-3 lock fails ⇒ ROLLBACK (deferred); entry stays undelivered; next
		// scan re-attempts (B3). The guard flip AND the appended event are both
		// discarded, preserving "SignalReceived in EventLog ⇔ delivered".
		return DeliverLockConflict, nil
	}

	if err := tx.Commit(); err != nil {
		return DeliverNoOp, fmt.Errorf("storage: DeliverSignal commit: %w", err)
	}
	return Delivered, nil
}
