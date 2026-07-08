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
		e.logger.Warn("signal timeout scan: list due wait records", "error", err.Error())
		return
	}
	for _, wr := range records {
		if err := e.processTimeout(ctx, store, wr); err != nil {
			e.logger.Error("signal timeout: process expired wait",
				"instance_id", string(wr.InstanceID), "step_id", wr.StepID,
				"timeout_action", wr.TimeoutAction, "error", err.Error())
		}
	}
}

// processTimeout handles one expired wait_record. It deletes the record
// (so the next tick does not re-process it), clears the in-memory wait,
// then routes by timeout_action.
func (e *Engine) processTimeout(ctx context.Context, store signalTimeoutStore, wr storage.WaitRecord) error {
	// Delete the expired wait_record immediately: must not re-fire on the next
	// tick even if downstream routing fails (idempotent timeout processing).
	if err := store.DeleteWaitRecord(ctx, wr.InstanceID, wr.StepID); err != nil {
		return fmt.Errorf("signal timeout: delete wait_record %s/%s: %w", wr.InstanceID, wr.StepID, err)
	}
	e.clearWait(wr.InstanceID, wr.SignalName)

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
		inst.Status = core.InstanceStatusRunning
		inst.UpdatedAt = e.now()
		e.mu.Lock()
		expected := e.ver[inst.InstanceID]
		e.mu.Unlock()
		if err := e.storage.UpsertInstance(ctx, inst, expected); err != nil {
			return fmt.Errorf("signal timeout continue: resume instance %s: %w", inst.InstanceID, err)
		}
		e.mu.Lock()
		e.ver[inst.InstanceID] = expected + 1
		e.mu.Unlock()
		startedAt := e.now()
		return e.emitStepCompleted(ctx, inst, wr.StepID, 1, map[string]any{}, startedAt)

	default:
		e.logger.Warn("signal timeout: unknown timeout_action; treating as fail",
			"timeout_action", wr.TimeoutAction,
			"instance_id", string(wr.InstanceID), "step_id", wr.StepID)
		step := dv.steps[wr.StepID]
		_, err = e.routeTerminalFailure(ctx, dv, inst, step, 1, stepErr)
		return err
	}
}
