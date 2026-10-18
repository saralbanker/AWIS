package engine

// Signal timeout scan (M07-C3r). On each tick, after SIGNAL_SCAN, every
// wait_record whose timeout_at ≤ now is expired and routed by its
// timeout_action through the EXISTING failure / compensation / transition paths.
//
// Three actions (Blueprint §9 L1411 VERBATIM: timeout_action TEXT NOT NULL —
// fail | compensate | continue):
//
//   - fail:        route through routeTerminalFailure (failure.go): fallback
//                  → on_error transitions → WorkflowFailed + compensation check.
//   - compensate:  skip fallback/on_error; force StepFailed + WorkflowFailed +
//                  compensation chain (compensation.go). The explicit intent of
//                  timeout_action=compensate is compensation, not fallback routing.
//   - continue:    the step "completes" with empty outputs (no signal payload);
//                  the workflow advances normally on the next tick.
//
// ALL actions use only the 12 closed TDS-01 event types. No new event type is
// introduced; any inexpressible action would require STOP (E2, closed vocabulary).

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// signalTimeoutStore is the additive storage slice the timeout scan needs:
// list all wait_records past their timeout, and delete one at a time as each
// is processed (prevents the next tick from re-processing the same record).
type signalTimeoutStore interface {
	ListDueWaitRecords(ctx context.Context, now time.Time) ([]storage.WaitRecord, error)
	DeleteWaitRecord(ctx context.Context, instanceID core.InstanceID, stepID string) error
}

// signalTimeoutScan checks wait_records whose timeout_at ≤ now and routes each
// by its timeout_action. Called once per tick after SIGNAL_SCAN. A storage that
// does not implement signalTimeoutStore no-ops (graceful degradation).
func (e *Engine) signalTimeoutScan(ctx context.Context) {
	store, ok := e.storage.(signalTimeoutStore)
	if !ok {
		return
	}
	now := e.now()
	records, err := store.ListDueWaitRecords(ctx, now)
	if err != nil {
		e.log().Warn("signal timeout scan: list due wait records", "error", err.Error())
		return
	}
	for _, wr := range records {
		// LA-C6 R3-M1: a committed route (3) of this instance awaits its terminal event (the
		// no-fault run never times this wait out: the instance is terminal after WorkflowFailed).
		// The record is not deleted, so it is re-checked on a later tick (routeRecoveryGate,
		// route_recovery.go).
		if e.routeRecoveryGate(ctx, wr.InstanceID, "timeout") {
			continue
		}
		if err := e.processTimeout(ctx, store, wr); err != nil {
			e.log().Error("signal timeout: process expired wait",
				"instance_id", string(wr.InstanceID), "step_id", wr.StepID,
				"timeout_action", wr.TimeoutAction, "error", err.Error())
		}
	}
}

// processTimeout handles one expired wait_record. It deletes the record
// (so the next tick does not re-process it), clears the in-memory wait,
// then routes by timeout_action.
//
// LA-C4c F-W: the timeout is durably routed only once its routing event (the
// first StepFailed / StepCompleted for the step) is appended. If any step fails
// before that append, the expired wait_record is restored (restoreTimedOutWait)
// so the next SIGNAL_TIMEOUT_SCAN re-fires it instead of parking the instance in
// `waiting` with no wait_record forever. Once the routing event is durable the
// record is never restored: a second fire would route the step twice.
//
// LA-C5 N3: any error here resets the instance's recovery bookkeeping, the same
// deferred pattern as CompleteStep (signal.go). Continue-timeout writes
// status=running, then appends StepCompleted; if that append's projection fails,
// the durable StepCompleted is applied only by a re-hydration (hydrate.go catch-up),
// and nothing else re-hydrates this instance in-process. The reset makes the next
// tick re-hydrate it. Call-site invariant (verified): processTimeout runs only from
// signalTimeoutScan, which tick calls after the SCAN_RUNNABLE loop; processInstance has
// returned (its dispatch goroutines joined by wg.Wait), and tick holds tickMu for its
// whole body (LA-C4b F-M), so no dispatch for wr.InstanceID is in flight.
func (e *Engine) processTimeout(ctx context.Context, store signalTimeoutStore, wr storage.WaitRecord) (err error) {
	defer func() {
		if err != nil {
			e.resetInstanceRecovery(wr.InstanceID)
		}
	}()
	// LA-C4c F-W: capture the durable sequence baseline BEFORE any mutation, so the
	// routing event can be told apart from events appended earlier. A failure here
	// leaves the wait_record intact; the next scan re-fires it.
	baseline, err := e.maxSeq(ctx, wr.InstanceID)
	if err != nil {
		return fmt.Errorf("signal timeout: read sequence baseline %s/%s: %w", wr.InstanceID, wr.StepID, err)
	}

	// Delete the expired wait_record immediately: must not re-fire on the next
	// tick even if downstream routing fails (idempotent timeout processing).
	if err := store.DeleteWaitRecord(ctx, wr.InstanceID, wr.StepID); err != nil {
		return fmt.Errorf("signal timeout: delete wait_record %s/%s: %w", wr.InstanceID, wr.StepID, err)
	}
	e.clearWait(wr.InstanceID, wr.SignalName)

	if err := e.routeTimedOutWait(ctx, wr); err != nil {
		// LA-C4c F-W: best-effort restore; the ORIGINAL routing error is returned.
		e.restoreTimedOutWait(ctx, wr, baseline, err)
		return err
	}
	return nil
}

// routeTimedOutWait routes one expired wait by its timeout_action. Every error it
// returns happens after the wait_record was deleted, so the caller decides
// (processTimeout) whether the record must be restored.
func (e *Engine) routeTimedOutWait(ctx context.Context, wr storage.WaitRecord) error {
	inst, found, err := e.getInstance(ctx, wr.InstanceID)
	if err != nil {
		return err
	}
	if !found || isTerminalStatus(inst.Status) {
		// Instance already terminal (race between timeout and normal completion/delivery).
		return nil
	}

	dv, err := e.defViewFor(ctx, inst)
	if err != nil {
		return err
	}

	stepErr := core.StepError{
		Code:    "signal_timeout",
		Message: fmt.Sprintf("signal %q wait timed out", wr.SignalName),
	}

	switch wr.TimeoutAction {
	case "fail":
		// Route through the existing terminal-failure chain (failure.go §8):
		// (1) fallback → (2) on_error → (3) WorkflowFailed + compensation.
		step := dv.steps[wr.StepID]
		_, err = e.routeTerminalFailure(ctx, dv, inst, step, 1, stepErr)
		return err

	case "compensate":
		// Force compensation: skip fallback/on_error routing (the explicit intent
		// of timeout_action=compensate is to compensate the workflow, not to
		// attempt recovery via a fallback step).
		if err := e.emitStepFailed(ctx, inst, wr.StepID, 1, stepErr, false); err != nil {
			return err
		}
		cur, found, err := e.getInstance(ctx, wr.InstanceID)
		if err != nil || !found {
			return err
		}
		if err := e.emitWorkflowFailed(ctx, cur, wr.StepID, stepErr); err != nil {
			return err
		}
		cur, _, _ = e.getInstance(ctx, wr.InstanceID)
		return e.maybeCompensate(ctx, dv, cur, wr.StepID)

	case "continue":
		// semantics-bearing: timeout→continue uses StepCompleted (TDS-01 closed
		// vocabulary); the step completes with empty outputs so the workflow
		// advances through normal transition evaluation on the next tick.
		//
		// Before emitting StepCompleted, the instance must be written back to
		// `running` (it is currently `waiting` from the non-evented enterWait
		// write). Without this, SCAN_RUNNABLE skips the waiting instance and
		// the workflow stalls. This mirrors the B3 delivery path: B3 sets
		// status=running via SignalReceived (applyEventToInstance), whereas
		// timeout-continue sets it via a non-evented UpsertInstance (same as
		// enterWait's non-evented transition in reverse).
		cur, err := e.resumeTimedOutInstance(ctx, wr.InstanceID)
		if err != nil {
			return err
		}
		startedAt := e.now()
		return e.emitStepCompleted(ctx, cur, wr.StepID, 1, map[string]any{}, startedAt)

	default:
		e.log().Warn("signal timeout: unknown timeout_action; treating as fail",
			"timeout_action", wr.TimeoutAction,
			"instance_id", string(wr.InstanceID), "step_id", wr.StepID)
		step := dv.steps[wr.StepID]
		_, err = e.routeTerminalFailure(ctx, dv, inst, step, 1, stepErr)
		return err
	}
}

// resumeTimedOutInstance writes the instance back to `running` for timeout→continue.
//
// LA-C4c F-V: the write is guarded by the DURABLE version (expectedVersion), never
// the cached e.ver: a restarted process's cache is empty while
// workflow_instances.version is not (B-0). It follows project()'s order — the
// version is read first, then the row — so a concurrent writer between the two
// reads surfaces as ErrVersionConflict instead of a silent overwrite. One retry
// on ErrVersionConflict re-reads both (same budget as enterWait).
func (e *Engine) resumeTimedOutInstance(ctx context.Context, iid core.InstanceID) (core.WorkflowInstance, error) {
	for attempt := 0; ; attempt++ {
		expected := e.expectedVersion(ctx, iid)
		cur, found, err := e.getInstance(ctx, iid)
		if err != nil {
			return core.WorkflowInstance{}, err
		}
		if !found {
			return core.WorkflowInstance{}, fmt.Errorf("signal timeout continue: instance %s vanished", iid)
		}
		cur.Status = core.InstanceStatusRunning
		cur.UpdatedAt = e.now()
		err = e.storage.UpsertInstance(ctx, cur, expected)
		if err == nil {
			e.mu.Lock()
			e.ver[iid] = expected + 1
			e.mu.Unlock()
			return cur, nil
		}
		if errors.Is(err, storage.ErrVersionConflict) && attempt == 0 {
			continue // one retry: re-read the durable version and the row.
		}
		return core.WorkflowInstance{}, fmt.Errorf("signal timeout continue: resume instance %s: %w", iid, err)
	}
}

// restoreTimedOutWait is the LA-C4c F-W best-effort restore. It runs only after
// routeTimedOutWait failed and restores the ORIGINAL wait_record (same fields,
// including its original timeout_at, so the next scan re-fires it) only when the
// routing event is provably NOT durable. If durability cannot be confirmed the
// record is not restored: a re-fire could route the step twice, and the failure is
// logged at Error. This is the D-6 best-effort pattern (tick.go gatherDispatch):
// a restore failure is logged with both errors and never masks the ORIGINAL error,
// which the caller still returns.
func (e *Engine) restoreTimedOutWait(ctx context.Context, wr storage.WaitRecord, baseline int, routeErr error) {
	durable, err := e.routingEventDurable(ctx, wr, baseline)
	if err != nil {
		e.log().Error("signal timeout: routing failed and durability of the routing event could not be confirmed; wait_record not restored",
			"instance_id", string(wr.InstanceID), "step_id", wr.StepID,
			"routing_error", routeErr.Error(), "check_error", err.Error())
		return
	}
	if durable {
		// LA-C4c F-W: routing event durable — never re-fire (would route twice).
		return
	}
	ws, ok := e.storage.(signalWaitStore)
	if !ok {
		e.log().Error("signal timeout: routing failed and storage cannot restore wait records; instance may remain waiting",
			"instance_id", string(wr.InstanceID), "step_id", wr.StepID, "routing_error", routeErr.Error())
		return
	}
	if err := ws.CreateWaitRecord(ctx, wr); err != nil && !errors.Is(err, storage.ErrDuplicateWaitRecord) {
		e.log().Error("signal timeout: routing failed and wait_record restore failed; instance may remain waiting",
			"instance_id", string(wr.InstanceID), "step_id", wr.StepID,
			"routing_error", routeErr.Error(), "restore_error", err.Error())
		return
	}
	e.log().Warn("signal timeout: routing failed before its routing event was durable; wait_record restored for re-fire",
		"instance_id", string(wr.InstanceID), "step_id", wr.StepID, "routing_error", routeErr.Error())
}

// routingEventDurable reports whether this timeout's routing event is durably in
// the log: the first StepFailed or StepCompleted for wr.StepID appended after
// baseline. Every timeout action appends one of these first (fail/compensate:
// StepFailed via routeTerminalFailure / emitStepFailed; continue: StepCompleted).
// It reads the durable log, so an append that returned an error but actually
// committed is correctly seen as durable.
func (e *Engine) routingEventDurable(ctx context.Context, wr storage.WaitRecord, baseline int) (bool, error) {
	evs, err := e.storage.ReadEvents(ctx, wr.InstanceID, 0)
	if err != nil {
		return false, fmt.Errorf("signal timeout: read events %s: %w", wr.InstanceID, err)
	}
	for _, ev := range evs {
		if ev.SequenceNum <= baseline || ev.StepID != wr.StepID {
			continue
		}
		if ev.EventType == core.EventTypeStepFailed || ev.EventType == core.EventTypeStepCompleted {
			return true, nil
		}
	}
	return false, nil
}
