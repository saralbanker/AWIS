package engine

import (
	"context"

	"github.com/awis/awis/internal/core"
)

// Compensation (Blueprint §8 L532–546, EDR-011 §6). When a workflow fails after
// ≥1 step completed, or when a --compensate cancellation drains, the engine runs
// the definition's CompensationPlan inline: WorkflowCompensating{from_step}, then
// the plan's undo actions in REVERSE order (skipping entries whose forward step
// never completed), then WorkflowCompensated — or WorkflowCompensationFailed if
// any undo exhausts its retries.
//
// Undo actions run through the engine's native Runner and emit NO ExecutionEvents
// (the 12 frozen event types have no per-undo events; only the three bracket
// events + slog record undo progress, EDR-011 §5). Each undo's StepContext inputs
// are the completed forward step's recorded outputs (variables[step_id]).
//
// V1 judgment call (documented): inline compensation retries honor the undo's own
// RetryPolicy Attempts + RetryableErrors, but execute back-to-back — the backoff
// delay is computed (for the audit log) and logged, NOT slept, because a real
// sleep would block the tick and injecting a sleep clock is out of scope. The
// audit_log write site arrives with M07's F-4 (EDR-011 §5); at M06 it is slog.

// maybeCompensate runs compensation after a WorkflowFailed emission when the
// definition has a plan AND at least one step completed (EDR-011 §6). It re-reads
// the instance so the completed-step set and recorded outputs are current.
func (e *Engine) maybeCompensate(ctx context.Context, dv *defView, inst core.WorkflowInstance, failedStep string) error {
	if dv.def.Compensation == nil {
		return nil
	}
	cur, found, err := e.getInstance(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	if len(completedSet(cur)) == 0 {
		return nil // nothing to undo (Blueprint §8: compensation only if ≥1 completed).
	}
	return e.runCompensation(ctx, dv, cur, failedStep)
}

// runCompensation emits the compensation bracket and executes the plan's undos in
// reverse. from_step is the failed forward step (empty for --compensate
// cancellation, which has no prior failure — judgment call, documented).
//
// semantics-bearing: reverse-order compensation, own retries (Blueprint §8 / EDR-011 §6).
func (e *Engine) runCompensation(ctx context.Context, dv *defView, inst core.WorkflowInstance, fromStep string) error {
	if err := e.emitWorkflowCompensating(ctx, inst, fromStep); err != nil {
		return err
	}
	completed := completedSet(inst)
	plan := dv.def.Compensation

	nr, ok := e.runners[core.StepTypeNative]
	if !ok {
		// No native runner registered ⇒ no undo can run; the plan cannot be
		// honored. Treat as a compensation failure at the first eligible undo
		// (documented edge; production always registers a native runner).
		for i := len(plan.Steps) - 1; i >= 0; i-- {
			cs := plan.Steps[i]
			if !completed[cs.StepID] {
				continue
			}
			e.log().Error("compensation failed: no native runner registered",
				"instance_id", string(inst.InstanceID), "step_id", cs.StepID)
			return e.emitWorkflowCompensationFailed(ctx, inst, cs.StepID, core.StepError{
				Code:    "runner_unavailable",
				Message: "no native runner registered for compensation undo",
			})
		}
		return e.emitWorkflowCompensated(ctx, inst)
	}

	for i := len(plan.Steps) - 1; i >= 0; i-- {
		cs := plan.Steps[i]
		if !completed[cs.StepID] {
			continue // forward step never completed ⇒ no undo (EDR-011 §6).
		}
		if undoErr := e.runUndo(ctx, inst, cs, nr); undoErr != nil {
			e.log().Error("compensation undo exhausted",
				"instance_id", string(inst.InstanceID), "step_id", cs.StepID,
				"error", undoErr.Message)
			return e.emitWorkflowCompensationFailed(ctx, inst, cs.StepID, *undoErr)
		}
	}
	return e.emitWorkflowCompensated(ctx, inst)
}

// runUndo executes one CompensationStep's undo handler through the native runner
// with its own retry policy (EDR-011 §6). It returns the last error if the undo
// exhausts its attempts, or nil on success. The undo's StepContext inputs are the
// completed forward step's recorded outputs (variables[step_id]).
func (e *Engine) runUndo(ctx context.Context, inst core.WorkflowInstance, cs core.CompensationStep, nr Runner) *core.StepError {
	maxAttempts := 1
	if cs.Retry != nil && cs.Retry.Attempts > 0 {
		maxAttempts = cs.Retry.Attempts
	}
	undoInputs := outputsOf(inst, cs.StepID)
	step := core.Step{
		ID:      cs.StepID,
		Type:    core.StepTypeNative,
		Handler: cs.UndoHandler,
	}

	var last *core.StepError
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		sc := core.StepContext{
			InstanceID: string(inst.InstanceID),
			StepID:     cs.StepID,
			Attempt:    attempt,
			Inputs:     undoInputs,
			Logger: e.log().With(
				"instance_id", string(inst.InstanceID),
				"step_id", cs.StepID,
				"phase", "compensation",
			),
		}
		_, serr := nr.Run(ctx, sc, step)
		if serr == nil {
			return nil
		}
		last = serr
		if attempt < maxAttempts && cs.Retry != nil && codeRetryable(cs.Retry, serr.Code) {
			// Backoff computed for the audit trail; NOT slept (V1, see file doc).
			delay := backoffDelay(*cs.Retry, attempt)
			e.log().Info("compensation undo retry",
				"instance_id", string(inst.InstanceID), "step_id", cs.StepID,
				"attempt", attempt, "backoff", delay.String())
			continue
		}
		break
	}
	return last
}

// outputsOf returns the recorded outputs of a completed step (variables[stepID])
// as a map, or an empty map when absent (EDR-007 variable shape).
func outputsOf(inst core.WorkflowInstance, stepID string) map[string]any {
	if v, ok := inst.Variables[stepID]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}
