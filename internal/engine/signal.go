package engine

// Signal subsystem wiring (M07-C2, F-4): the external signal intake, the
// WAIT-step entry that parks a type=signal step, and the engine's implementation
// of signal.Waits (the in-memory wait bookkeeping the SIGNAL_SCAN delivery scan
// consults and notifies).
//
// The atomic delivery itself lives in storage.DeliverSignal (Finalization B3);
// the scan loop lives in internal/signal. This file holds only the engine-side
// surface: intake → inbox insert, wait-record + waiting-status entry, and the
// cursor/version bookkeeping that keeps the engine in lock-step with the B3 tx.
//
// Signal-store methods are additive *SQLiteStorage methods reached through
// type-asserted interfaces (as cancellation.go / triggers.go do); StoragePort's
// 12 methods stay frozen (Blueprint §20).

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/signal"
	"github.com/awis/awis/internal/storage"
)

// compile-time assertion: *Engine satisfies signal.Waits (consumed by signalScan).
var _ signal.Waits = (*Engine)(nil)

// signalIntakeStore is the additive slice used by the signal intake path.
type signalIntakeStore interface {
	InsertSignal(ctx context.Context, sig storage.Signal) error
}

// signalWaitStore is the additive slice used by WAIT-step entry.
type signalWaitStore interface {
	CreateWaitRecord(ctx context.Context, wr storage.WaitRecord) error
}

// signalWaitListStore is the additive storage slice used by Resolve's durable
// recovery path (engine-hardening Step 2.2, closes B-15): after a restart the
// in-memory e.waits map is empty even though the wait durably survives in
// wait_records, so Resolve falls back to listing an instance's live waits and
// matching on signal_name (GetWaitRecord is keyed by step_id, not usable here).
type signalWaitListStore interface {
	ListWaitRecordsByInstance(ctx context.Context, instanceID core.InstanceID) ([]storage.WaitRecord, error)
}

// signalWaitDeleteStore is the additive storage slice for single wait_record
// deletion: used at the delivery completion site (CompleteStep, OUTPUT 0) and
// by the timeout scan (signal_timeout.go).
type signalWaitDeleteStore interface {
	DeleteWaitRecord(ctx context.Context, instanceID core.InstanceID, stepID string) error
}

// Signal records an external signal into the inbox (intake). A later tick's
// SIGNAL_SCAN delivers it to the matching waiting instance through the B3
// transaction. instanceID and name must be non-empty; a nil payload is stored as
// an empty map. The SDK SignalAPI wrapper is M08.
//
// Intake is order-independent with respect to the wait: a signal that arrives
// before its wait sits undelivered in the inbox (the scan finds no live wait and
// skips it) until WAIT-step entry records the wait, after which the next scan
// delivers it.
func (e *Engine) Signal(ctx context.Context, instanceID core.InstanceID, name string, payload map[string]any) error {
	if instanceID == "" || name == "" {
		return fmt.Errorf("engine: Signal: instance_id and signal name must be non-empty")
	}
	store, ok := e.storage.(signalIntakeStore)
	if !ok {
		return fmt.Errorf("engine: Signal: storage does not support signal intake")
	}
	if payload == nil {
		payload = map[string]any{}
	}
	sig := storage.Signal{
		SignalID:   e.newID(),
		InstanceID: instanceID,
		SignalName: name,
		Payload:    payload,
		ReceivedAt: e.now(),
	}
	if err := store.InsertSignal(ctx, sig); err != nil {
		return fmt.Errorf("engine: Signal intake %s/%s: %w", instanceID, name, err)
	}
	e.logger.Info("signal intake",
		"instance_id", string(instanceID), "signal_name", name, "signal_id", sig.SignalID)
	return nil
}

// enterWait parks a just-activated type=signal step: it writes the wait_record
// from the step's WaitConfig, transitions the projection waiting (non-evented —
// EDR-007 §9 waiting-entry gap: no event produces `waiting`), and records the
// wait in the engine's in-memory index for the SIGNAL_SCAN to match.
//
// OUTPUT 0b: timeout_at is populated as an absolute RFC3339 timestamp
// (engine clock + WaitConfig.Timeout); absent Timeout writes NULL. No new
// Duration serialization form is introduced — core.Duration is parsed inline
// via time.ParseDuration (the natural Go duration literal form, e.g. "1m30s").
// Cite: Blueprint §9 L1410 (timeout_at column RFC3339 form).
func (e *Engine) enterWait(ctx context.Context, inst core.WorkflowInstance, step core.Step) error {
	wc := step.WaitSignal
	if wc == nil {
		// validate.Validate rejects a type=signal step without wait_signal; this
		// guard keeps the path typed.
		return fmt.Errorf("engine: enterWait: signal step %s has no wait_signal config", step.ID)
	}
	store, ok := e.storage.(signalWaitStore)
	if !ok {
		return fmt.Errorf("engine: enterWait: storage does not support wait records")
	}

	now := e.now()

	// OUTPUT 0b: compute timeout_at = engine clock + WaitConfig.Timeout.
	// Blueprint §9 L1410: timeout_at stores an absolute RFC3339 timestamp.
	// Parse core.Duration as a Go duration literal; absent Timeout → NULL.
	var timeoutAt *time.Time
	if string(wc.Timeout) != "" {
		d, err := time.ParseDuration(string(wc.Timeout))
		if err == nil {
			t := now.Add(d)
			timeoutAt = &t
		} else {
			e.logger.Warn("signal step: unparseable timeout; wait_record written with NULL timeout_at",
				"step_id", step.ID, "timeout", string(wc.Timeout), "error", err.Error())
		}
	}

	wr := storage.WaitRecord{
		InstanceID:    inst.InstanceID,
		StepID:        step.ID,
		SignalName:    wc.SignalName,
		CreatedAt:     now,
		TimeoutAt:     timeoutAt, // absolute RFC3339; NULL when no Timeout (Blueprint §9 L1410)
		TimeoutAction: wc.TimeoutAction,
	}
	if err := store.CreateWaitRecord(ctx, wr); err != nil {
		if errors.Is(err, storage.ErrDuplicateWaitRecord) {
			// Already waiting on this step (idempotent re-entry): re-record the
			// in-memory wait and return without re-transitioning the projection.
			e.rememberWait(inst.InstanceID, wc.SignalName, step.ID)
			return nil
		}
		return fmt.Errorf("engine: enterWait create wait_record %s/%s: %w", inst.InstanceID, step.ID, err)
	}

	// Transition the projection running→waiting. This is a non-evented projection
	// write (EDR-007 §9): `waiting` is an engine tick-time decision, not an event,
	// so a cold RebuildState reconstructs `running` (the documented §9 gap) — the
	// forward and rebuild paths reconverge once SignalReceived is delivered.
	//
	// engine-hardening Step 1.4: guarded by the durable version (expectedVersion,
	// same B-0 fix as project()), with one retry on ErrVersionConflict that
	// re-reads the durable version rather than assuming expected+1.
	cur, found, err := e.getInstance(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("engine: enterWait: instance %s vanished", inst.InstanceID)
	}
	cur.Status = core.InstanceStatusWaiting
	cur.UpdatedAt = now
	for attempt := 0; ; attempt++ {
		expected := e.expectedVersion(ctx, inst.InstanceID)
		err := e.storage.UpsertInstance(ctx, cur, expected)
		if err == nil {
			e.mu.Lock()
			e.ver[inst.InstanceID] = expected + 1
			e.mu.Unlock()
			break
		}
		if errors.Is(err, storage.ErrVersionConflict) && attempt == 0 {
			continue // one retry: re-read the durable version and try again.
		}
		return fmt.Errorf("engine: enterWait set waiting %s: %w", inst.InstanceID, err)
	}

	e.rememberWait(inst.InstanceID, wc.SignalName, step.ID)
	e.logger.Info("wait entry",
		"instance_id", string(inst.InstanceID), "step_id", step.ID, "signal_name", wc.SignalName)
	return nil
}

// rememberWait records a live wait in the in-memory index (V1 runtime state,
// EDR-011 §8).
func (e *Engine) rememberWait(iid core.InstanceID, signalName, stepID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	m := e.waits[iid]
	if m == nil {
		m = make(map[string]string)
		e.waits[iid] = m
	}
	m[signalName] = stepID
}

// Resolve implements signal.Waits: it reports whether iid is actively waiting on
// signalName and, if so, returns the awaited step id and the durable
// optimistic-lock version (the B3 step-3 <expected_version>).
//
// engine-hardening (Step 1.6 + Step 2.2, closes B-15): the in-memory e.waits
// map is V1 runtime state that is EMPTY right after a restart even though the
// instance is durably `waiting` — the wait_record row and the projection both
// survive the process boundary. On an in-memory miss, Resolve consults
// storage's wait_records for iid and matches on signalName; a hit repopulates
// e.waits (so the next Resolve for the same wait is a pure map lookup again).
// The returned version is always the durable one (expectedVersion — the same
// B-0 fix project() uses), never the possibly-stale e.ver cache directly.
func (e *Engine) Resolve(ctx context.Context, iid core.InstanceID, signalName string) (string, int, bool) {
	e.mu.Lock()
	m, ok := e.waits[iid]
	var stepID string
	if ok {
		stepID, ok = m[signalName]
	}
	e.mu.Unlock()

	if !ok {
		stepID, ok = e.recoverWait(ctx, iid, signalName)
		if !ok {
			return "", 0, false
		}
	}
	return stepID, e.expectedVersion(ctx, iid), true
}

// recoverWait durably resolves a live wait for (iid, signalName) when the
// in-memory index has no entry, by scanning storage's wait_records and
// matching on signal_name. A hit remembers the wait in-memory so subsequent
// Resolve calls for the same wait are warm again (B-15).
func (e *Engine) recoverWait(ctx context.Context, iid core.InstanceID, signalName string) (string, bool) {
	store, ok := e.storage.(signalWaitListStore)
	if !ok {
		return "", false
	}
	wrs, err := store.ListWaitRecordsByInstance(ctx, iid)
	if err != nil {
		e.logger.Warn("Resolve: durable wait-record recovery failed",
			"instance_id", string(iid), "signal_name", signalName, "error", err.Error())
		return "", false
	}
	for _, wr := range wrs {
		if wr.SignalName == signalName {
			e.rememberWait(iid, signalName, wr.StepID)
			return wr.StepID, true
		}
	}
	return "", false
}

// OnDelivered implements signal.Waits: after the B3 tx commits it advances the
// engine's per-instance sequence and version cursors (the tx assigned seq MAX+1
// and did version=version+1) and clears the resolved wait, so the engine's view
// stays in lock-step with storage (mirrors bumpVersion in emit.go).
func (e *Engine) OnDelivered(iid core.InstanceID, signalName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.seq[iid]++
	e.ver[iid]++
	if m := e.waits[iid]; m != nil {
		delete(m, signalName)
		if len(m) == 0 {
			delete(e.waits, iid)
		}
	}
}

// clearWait removes the in-memory wait for (iid, signalName). Called by the
// timeout scan (processTimeout) when a wait expires without delivery.
func (e *Engine) clearWait(iid core.InstanceID, signalName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if m := e.waits[iid]; m != nil {
		delete(m, signalName)
		if len(m) == 0 {
			delete(e.waits, iid)
		}
	}
}

// CompleteStep implements signal.Waits: after the B3 delivery tx commits it
// deletes the wait_record and emits StepCompleted with the signal payload as
// outputs through the existing StepCompleted emission path (M07-C3r OUTPUT 0;
// TDS-01 §2 StepCompleted REPLAY carries outputs; Blueprint §8 SETTLE →
// transition evaluation).
//
// semantics-bearing: wait_record deleted on step completion (OUTPUT 0); an
// open wait_record after delivery would misfire the timeout scan. attempt=1
// always: a signal step enters wait on its first (only) activation.
func (e *Engine) CompleteStep(ctx context.Context, iid core.InstanceID, stepID string, payload map[string]any) error {
	// Delete wait_record first: the step is done; the timeout scan must not re-fire.
	if ds, ok := e.storage.(signalWaitDeleteStore); ok {
		if err := ds.DeleteWaitRecord(ctx, iid, stepID); err != nil {
			return fmt.Errorf("engine: CompleteStep delete wait_record %s/%s: %w", iid, stepID, err)
		}
	}
	// Read the current instance state (after B3 resume: status=running).
	inst, found, err := e.getInstance(ctx, iid)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("engine: CompleteStep: instance %s vanished after delivery", iid)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	// Emit StepCompleted with signal payload as outputs (TDS-01 §2).
	// startedAt ≈ now: the WAIT step has no real execution time; duration_ms
	// is informational for signal steps (delivery latency, not handler time).
	return e.emitStepCompleted(ctx, inst, stepID, 1, payload, e.now())
}
