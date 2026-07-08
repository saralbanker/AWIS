// Package signal implements the SIGNAL_SCAN delivery scan: it matches
// undelivered inbox signals against the instances currently waiting on them and
// drives each match through the Finalization B3 atomic-delivery transaction
// (storage.DeliverSignal), then forward-projects the resume (M07-C2, F-4).
//
// The B3 transaction itself owns the projection update (its step 3 moves the
// instance waiting→running, mirroring EDR-007 §2's SignalReceived rule); the
// scanner's role is to (a) resolve which undelivered signal has a live wait, (b)
// build the SignalReceived payload VERBATIM, (c) invoke the tx with the awaited
// version, and (d) at the delivery site record the SignalDelivered audit fact
// and notify the engine so its in-memory sequence/version cursors stay in
// lock-step with the storage-side bump (as M06 emit.go keeps them).
//
// Dependency direction (IMP §6): signal imports storage + core; the engine
// imports signal. Nothing outside the platform imports internal/.
package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// Store is the persistence surface the delivery scan needs. It is satisfied by
// the additive *SQLiteStorage methods (ListUndeliveredSignals + DeliverSignal +
// AppendAudit); StoragePort's 12 methods stay frozen (Blueprint §20).
type Store interface {
	ListUndeliveredSignals(ctx context.Context, instanceID core.InstanceID) ([]storage.Signal, error)
	DeliverSignal(ctx context.Context, in storage.DeliverInput) (storage.DeliverOutcome, error)
	AppendAudit(ctx context.Context, entry storage.AuditEntry) error
}

// Waits resolves the engine's in-memory wait bookkeeping and receives delivery
// notifications. The wait map is V1 single-process runtime state — reconstructible
// from the EventLog, lost on crash (EDR-011 §8) — that mirrors the persisted
// wait_records written at WAIT-step entry.
type Waits interface {
	// Resolve reports whether instanceID is actively waiting on signalName and,
	// if so, returns the awaited step id and the optimistic-lock version the
	// resume must guard (B3 step 3 <expected_version>). ok=false means no live
	// wait — an early/spurious signal or an already-resumed instance — and the
	// scan skips it so a signal never resumes the wrong wait.
	Resolve(instanceID core.InstanceID, signalName string) (stepID string, expectedVersion int, ok bool)
	// OnDelivered notifies the engine that the B3 tx committed for
	// (instanceID, signalName): the engine advances its sequence/version cursors
	// (the tx assigned seq MAX+1 and bumped version) and clears the in-memory wait.
	OnDelivered(instanceID core.InstanceID, signalName string)
	// CompleteStep is called at the delivery site after OnDelivered: the engine
	// emits StepCompleted with the signal payload as outputs and deletes the
	// wait_record (OUTPUT 0 post-delivery resumption completion; TDS-01 §2
	// StepCompleted REPLAY carries outputs). Called only on Delivered outcome.
	CompleteStep(ctx context.Context, instanceID core.InstanceID, stepID string, payload map[string]any) error
}

// signalReceivedPayload is the SignalReceived payload, field names transcribed
// VERBATIM from EVENTLOG_FORMAT §6: {signal_name, payload: map}.
type signalReceivedPayload struct {
	SignalName string         `json:"signal_name"`
	Payload    map[string]any `json:"payload"`
}

// Scanner runs one SIGNAL_SCAN pass per engine tick.
type Scanner struct {
	store  Store
	waits  Waits
	now    func() time.Time
	newID  func() string
	actor  string
	logger *slog.Logger
}

// NewScanner builds a Scanner. now/newID are the engine's injectable clock and id
// source (IMP §3 determinism rule); actor is the worker id recorded on the
// SignalDelivered audit entry.
func NewScanner(store Store, waits Waits, now func() time.Time, newID func() string, actor string, logger *slog.Logger) *Scanner {
	return &Scanner{store: store, waits: waits, now: now, newID: newID, actor: actor, logger: logger}
}

// Scan delivers every currently-deliverable signal: undelivered inbox × matching
// live wait. A real store error aborts the pass (returned to the caller for
// logging); a per-signal no-op / lock-conflict / audit failure is logged and the
// scan continues with the next signal.
func (sc *Scanner) Scan(ctx context.Context) error {
	// Empty instanceID ⇒ scan across all instances (subsystem-wide undelivered set).
	sigs, err := sc.store.ListUndeliveredSignals(ctx, "")
	if err != nil {
		return fmt.Errorf("signal: list undelivered: %w", err)
	}
	for _, sig := range sigs {
		if err := sc.deliverOne(ctx, sig); err != nil {
			return err
		}
	}
	return nil
}

// deliverOne attempts the B3 delivery of one undelivered signal. It returns an
// error only for a real store fault on DeliverSignal itself; the B3-defined
// no-op and lock-conflict outcomes are benign and logged.
func (sc *Scanner) deliverOne(ctx context.Context, sig storage.Signal) error {
	stepID, expectedVersion, ok := sc.waits.Resolve(sig.InstanceID, sig.SignalName)
	if !ok {
		// No live wait for this (instance, signal): early/spurious signal, or the
		// instance already resumed. Leave it undelivered; a later wait may claim it.
		return nil
	}

	payloadMap := sig.Payload
	if payloadMap == nil {
		payloadMap = map[string]any{}
	}
	payload, err := json.Marshal(signalReceivedPayload{SignalName: sig.SignalName, Payload: payloadMap})
	if err != nil {
		return fmt.Errorf("signal: marshal SignalReceived payload (instance=%s): %w", sig.InstanceID, err)
	}

	outcome, err := sc.store.DeliverSignal(ctx, storage.DeliverInput{
		InstanceID:      sig.InstanceID,
		SignalName:      sig.SignalName,
		EventPayload:    payload,
		ExpectedVersion: expectedVersion,
		EventID:         sc.newID(),
		Now:             sc.now(),
	})
	if err != nil {
		return fmt.Errorf("signal: deliver (instance=%s signal=%s): %w", sig.InstanceID, sig.SignalName, err)
	}

	switch outcome {
	case storage.Delivered:
		// Delivery site: keep the engine's cursors in step, then complete the
		// waiting step (OUTPUT 0 post-delivery resumption), then audit.
		sc.waits.OnDelivered(sig.InstanceID, sig.SignalName)
		sc.logger.Info("signal delivered",
			"instance_id", string(sig.InstanceID), "signal_name", sig.SignalName, "step_id", stepID)
		// Post-delivery step completion (M07-C3r OUTPUT 0): emit StepCompleted
		// with signal payload as outputs so the workflow can advance. The B3 tx
		// already committed; a failure here is logged but does not undo delivery.
		if err := sc.waits.CompleteStep(ctx, sig.InstanceID, stepID, payloadMap); err != nil {
			sc.logger.Error("signal step completion after delivery",
				"instance_id", string(sig.InstanceID), "step_id", stepID, "error", err.Error())
		}
		entry := storage.AuditEntry{
			Timestamp:      sc.now(),
			EventType:      "SignalDelivered",
			Actor:          sc.actor,
			PayloadSummary: fmt.Sprintf("signal=%s instance=%s", sig.SignalName, sig.InstanceID),
		}
		if err := sc.store.AppendAudit(ctx, entry); err != nil {
			// Audit is a side-record: a failure must not wedge delivery of other
			// signals (the B3 tx already committed). Log and continue.
			sc.logger.Warn("signal-delivered audit append failed",
				"instance_id", string(sig.InstanceID), "signal_name", sig.SignalName, "error", err.Error())
		}
	case storage.DeliverLockConflict:
		// Rolled back; entry stays undelivered; retried next tick (B3).
		sc.logger.Info("signal delivery lock conflict; will retry",
			"instance_id", string(sig.InstanceID), "signal_name", sig.SignalName)
	case storage.DeliverNoOp:
		// Idempotency guard matched nothing (already delivered) — B3 no-op.
		sc.logger.Info("signal delivery no-op (already delivered)",
			"instance_id", string(sig.InstanceID), "signal_name", sig.SignalName)
	}
	return nil
}
