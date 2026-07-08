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
// timeout_at is left NULL here: the WaitConfig.Timeout Duration has no frozen
// serialization form (scalars.go — "the frozen corpus does not show a
// serialization form"), and the timeout scan is C3's scope wall. TimeoutAction
// is persisted VERBATIM for C3.
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
	wr := storage.WaitRecord{
		InstanceID:    inst.InstanceID,
		StepID:        step.ID,
		SignalName:    wc.SignalName,
		CreatedAt:     now,
		TimeoutAt:     nil, // Duration form undefined (scalars.go); timeout scan is C3.
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
	cur, found, err := e.getInstance(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("engine: enterWait: instance %s vanished", inst.InstanceID)
	}
	cur.Status = core.InstanceStatusWaiting
	cur.UpdatedAt = now
	e.mu.Lock()
	expected := e.ver[inst.InstanceID]
	e.mu.Unlock()
	if err := e.storage.UpsertInstance(ctx, cur, expected); err != nil {
		return fmt.Errorf("engine: enterWait set waiting %s: %w", inst.InstanceID, err)
	}
	e.mu.Lock()
	e.ver[inst.InstanceID] = expected + 1
	e.mu.Unlock()

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
// signalName and, if so, returns the awaited step id and the current tracked
// optimistic-lock version (the B3 step-3 <expected_version>).
func (e *Engine) Resolve(iid core.InstanceID, signalName string) (string, int, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	m, ok := e.waits[iid]
	if !ok {
		return "", 0, false
	}
	stepID, ok := m[signalName]
	if !ok {
		return "", 0, false
	}
	return stepID, e.ver[iid], true
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
