package engine

import (
	"context"
	"fmt"

	"github.com/awis/awis/internal/core"
)

// Cancellation FSM — Finalization B4 VERBATIM. Cancellation is a request flag on
// the projection (workflow_instances.cancellation_requested), never a status
// (CONTRA-6). The engine sets the flag, and the tick — after in-flight steps
// settle and BEFORE any activation — checks it: set ⇒ no next step activates
// regardless of conditions (B4 step 3); when current_steps drains, the instance
// goes to cancelled (default) or through the compensation plan (--compensate).
//
// The flag lives in a DB column exposed by additive *SQLiteStorage accessors
// (cancellation.go), reached through the cancellationStore interface below so the
// engine does not depend on the concrete adapter type (StoragePort's 12 methods
// stay frozen).

// cancellationStore is the additive slice of the SQLite adapter the engine needs
// for the cancellation flag. It is NOT part of the frozen StoragePort (Blueprint
// §20); the concrete adapter satisfies it (cancellation.go).
type cancellationStore interface {
	SetCancellationRequested(ctx context.Context, instanceID core.InstanceID) error
	CancellationRequested(ctx context.Context, instanceID core.InstanceID) (bool, error)
}

// Cancel requests cancellation of an instance (Finalization B4). Behavior by
// current status:
//
//   - terminal (completed|failed|cancelled|compensated|compensation_failed) ⇒
//     no-op returning nil, with the EXACT B4 warning logged;
//   - pending ⇒ WorkflowCancelled{reason} immediately (cancelled, no steps run);
//   - otherwise (running) ⇒ set cancellation_requested=1 and remember (reason,
//     compensate) for the tick that drains in-flight work.
//
// V1 single-process: the (reason, compensate) memory is in-engine; a crash loses
// it and rebuild resets the flag to 0 (EDR-011 §8) so the caller re-issues Cancel.
func (e *Engine) Cancel(ctx context.Context, instanceID core.InstanceID, reason string, compensate bool) error {
	inst, found, err := e.getInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("engine: Cancel: instance %s not found", instanceID)
	}

	if isTerminalStatus(inst.Status) {
		// EXACT B4 idempotency warning (do not reformat).
		e.logger.Warn(fmt.Sprintf("instance %s is already in terminal state %s; cancel is a no-op",
			instanceID, inst.Status))
		return nil
	}

	if inst.Status == core.InstanceStatusPending {
		// Cancelled immediately; no steps were executed (B4).
		return e.emitWorkflowCancelled(ctx, inst, reason)
	}

	// Running (or any non-terminal, non-pending status): flag it and remember the
	// mode; the tick finalizes once in-flight work settles.
	store, ok := e.storage.(cancellationStore)
	if !ok {
		return fmt.Errorf("engine: Cancel: storage does not support the cancellation flag")
	}
	if err := store.SetCancellationRequested(ctx, instanceID); err != nil {
		return fmt.Errorf("engine: Cancel: set flag: %w", err)
	}
	e.mu.Lock()
	e.cancels[instanceID] = cancelIntent{reason: reason, compensate: compensate}
	e.mu.Unlock()
	return nil
}

// isTerminalStatus reports whether s is a terminal instance status (Finalization
// B4 idempotency set). Mirrors storage.isTerminalStatus and helpers_test.isTerminal.
func isTerminalStatus(s core.InstanceStatus) bool {
	switch s {
	case core.InstanceStatusCompleted, core.InstanceStatusFailed,
		core.InstanceStatusCancelled, core.InstanceStatusCompensated,
		core.InstanceStatusCompensationFailed:
		return true
	}
	return false
}

// cancellationRequested reads the instance's cancellation flag via the additive
// accessor. A storage that does not implement cancellationStore reports false
// (cancellation is a no-op for such adapters — none exist at M06).
func (e *Engine) cancellationRequested(ctx context.Context, instanceID core.InstanceID) (bool, error) {
	store, ok := e.storage.(cancellationStore)
	if !ok {
		return false, nil
	}
	flagged, err := store.CancellationRequested(ctx, instanceID)
	if err != nil {
		return false, fmt.Errorf("engine: read cancellation flag %s: %w", instanceID, err)
	}
	return flagged, nil
}

// handleCancellation is invoked once an instance's cancellation flag is observed
// set (either at the top of the tick, or after in-flight steps settle). It:
//
//   - terminally fails any retry-waiting steps (StepFailed{retrying:false},
//     removing them from current_steps, NO WorkflowFailed — a cancelled retry is
//     not a workflow failure, EDR-011 §8-adjacent reading);
//   - abandons all pending activations (no step activates, B4 step 3);
//   - when current_steps has drained, finalizes: WorkflowCancelled{reason}
//     (default) or the compensation plan (--compensate).
//
// It returns without finalizing if steps remain in flight (they settle on a
// later tick — cannot occur in V1's single-tick dispatch, but the drain guard is
// the correct general rule).
func (e *Engine) handleCancellation(ctx context.Context, dv *defView, inst core.WorkflowInstance) error {
	// Terminally fail retry-waiting steps (they are in current_steps but will
	// never be re-dispatched under cancellation).
	for _, stepID := range e.scheduledRetrySteps(inst.InstanceID, inst.CurrentSteps) {
		attempt := e.retryAttempt(inst.InstanceID, stepID)
		e.clearRetry(inst.InstanceID, stepID)
		serr := core.StepError{Code: "cancelled", Message: "step cancelled while awaiting retry"}
		if err := e.emitStepFailed(ctx, inst, stepID, attempt, serr, false); err != nil {
			return err
		}
		e.logger.Info("settle", "instance_id", string(inst.InstanceID), "step_id", stepID, "outcome", "cancelled")
	}
	e.dropAllPending(inst.InstanceID)

	cur, found, err := e.getInstance(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if !found || len(cur.CurrentSteps) != 0 {
		return nil // in-flight steps remain; finalize on a later tick.
	}
	return e.finalizeCancellation(ctx, dv, cur)
}

// finalizeCancellation emits the terminal cancellation events once current_steps
// has drained (Finalization B4 steps 4/5). compensate=true reuses the
// compensation machinery WITHOUT a prior WorkflowFailed (B4 step 4).
func (e *Engine) finalizeCancellation(ctx context.Context, dv *defView, inst core.WorkflowInstance) error {
	e.mu.Lock()
	intent := e.cancels[inst.InstanceID]
	delete(e.cancels, inst.InstanceID)
	e.mu.Unlock()

	if intent.compensate && dv.def.Compensation != nil && len(completedSet(inst)) > 0 {
		// --compensate ⇒ compensating → compensated (from_step empty: no failure).
		return e.runCompensation(ctx, dv, inst, "")
	}
	return e.emitWorkflowCancelled(ctx, inst, intent.reason)
}
