package engine

import (
	"context"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/expr"
)

// Terminal-failure routing (EDR-011 §8). On a terminal step failure (attempts
// exhausted, or a capability_fallback signal which skips retries entirely) the
// engine routes in this fixed order:
//
//	(1) step.Fallback ≠ ""            ⇒ StepFailed{retrying:false} then
//	                                    StepFallbackActivated; activate the
//	                                    fallback DIRECTLY (bypass the join gate).
//	(2) ≥1 firing on_error transition ⇒ StepFailed{retrying:false} then activate
//	                                    ALL firing targets directly; NO WorkflowFailed.
//	(3) otherwise                     ⇒ StepFailed{retrying:false} + WorkflowFailed
//	                                    + compensation check (compensation.go).
//
// Fallback wins over on_error because §8 L530 orders fallback explicitly; on_error
// would be dead code if failure always went fallback-or-workflow-failure
// (EDR-011 §8). Direct activations are recorded in the engine's per-instance
// pending set, consumed by the next scan (activatableFor).

// pendingSet returns the current pending-activation set for iid (a live copy is
// unnecessary; callers only read membership under the engine lock indirectly).
func (e *Engine) pendingSet(iid core.InstanceID) map[string]bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	src := e.pending[iid]
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]bool, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// addPending records a direct (join-gate-bypassing) activation of step for iid.
func (e *Engine) addPending(iid core.InstanceID, step string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pending[iid] == nil {
		e.pending[iid] = make(map[string]bool)
	}
	e.pending[iid][step] = true
}

// clearPending drops step from iid's pending set (called once the step is started
// so a subsequent scan does not re-add it).
func (e *Engine) clearPending(iid core.InstanceID, step string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if m := e.pending[iid]; m != nil {
		delete(m, step)
		if len(m) == 0 {
			delete(e.pending, iid)
		}
	}
}

// dropAllPending clears every pending activation for iid (used when cancellation
// finalizes: no step activates regardless, Finalization B4 step 3).
func (e *Engine) dropAllPending(iid core.InstanceID) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.pending, iid)
}

// markFailed records step as terminally failed for iid (engine-hardening
// Step 2b, closes B-4). A terminally-failed step is removed from
// current_steps but never enters Variables, so the join gate would otherwise
// see its upstream complete and re-nominate it on every tick forever — the
// exact hang the Finalization's text (Blockers 2/3/4) is the oracle against.
// Every routeTerminalFailure branch (all three emit StepFailed{retrying:
// false}) and handleCancellation's retry-cancellation loop call this
// alongside their StepFailed emission. A step is never un-failed within an
// instance (no corresponding "unmark").
func (e *Engine) markFailed(iid core.InstanceID, step string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.failed[iid] == nil {
		e.failed[iid] = make(map[string]bool)
	}
	e.failed[iid][step] = true
}

// failedSet returns a snapshot copy of iid's terminally-failed step set (the
// pendingSet idiom: a defensive copy so the caller can range over it without
// holding e.mu).
func (e *Engine) failedSet(iid core.InstanceID) map[string]bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	src := e.failed[iid]
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]bool, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// activatableFor derives the ordered set of steps eligible to dispatch this tick:
// the join-gate activatable set (transition.go) UNION the failure-driven pending
// set (fallback / on_error targets that bypass the join gate). Members are
// returned in definition order, excluding completed/running steps (EDR-011 §8).
//
// It is the activation source used by the tick and by the completion check, so a
// workflow with outstanding pending activations is never treated as complete.
func (e *Engine) activatableFor(dv *defView, inst core.WorkflowInstance) []string {
	completed := completedSet(inst)
	running := make(map[string]bool, len(inst.CurrentSteps))
	for _, s := range inst.CurrentSteps {
		running[s] = true
	}
	env := buildEnv(inst)
	pend := e.pendingSet(inst.InstanceID)
	// B-4: a terminally-failed step must never re-activate, even though it is
	// absent from both `completed` (never entered Variables) and `running`
	// (removed from current_steps by its terminal StepFailed) — excluded here,
	// before both the pend[id] check and the isActivatable check below.
	failed := e.failedSet(inst.InstanceID)

	var out []string
	for _, step := range dv.def.Steps { // definition order (deterministic dispatch)
		id := step.ID
		if completed[id] || running[id] || failed[id] {
			continue
		}
		if pend[id] || isActivatable(dv, id, completed, running, env) {
			out = append(out, id)
		}
	}
	return out
}

// firingOnError returns the distinct `to` targets of the failed step's on_error
// transitions that fire (condition absent ⇒ fires; present ⇒ Eval true), in
// definition order (EDR-011 §8 case 2).
func firingOnError(dv *defView, failedStep string, env expr.Env) []string {
	seen := make(map[string]bool)
	var out []string
	for _, t := range dv.def.Transitions {
		if t.From != failedStep || !t.OnError {
			continue
		}
		fires := true
		if cond := string(t.Condition); cond != "" {
			if te, ok := lookupInbound(dv, t); ok && te.cond != nil {
				ok2, err := te.cond.Eval(env)
				fires = err == nil && ok2
			}
		}
		if fires && !seen[t.To] {
			seen[t.To] = true
			out = append(out, t.To)
		}
	}
	return out
}

// lookupInbound finds the pre-parsed transEval matching t among dv's inbound
// index (conditions are parsed once at buildDefView; this reuses that parse).
func lookupInbound(dv *defView, t core.Transition) (transEval, bool) {
	for _, te := range dv.inbound[t.To] {
		if te.t.From == t.From && te.t.OnError == t.OnError && te.t.Condition == t.Condition {
			return te, true
		}
	}
	return transEval{}, false
}

// routeTerminalFailure applies the EDR-011 §8 routing order for a terminally
// failed step. It returns terminal=true iff the workflow itself reached a
// terminal state (WorkflowFailed / compensation), in which case the tick stops
// settling further items (preserving C1's fail-fast).
func (e *Engine) routeTerminalFailure(ctx context.Context, dv *defView, inst core.WorkflowInstance, step core.Step, attempt int, stepErr core.StepError) (terminal bool, err error) {
	// (1) fallback — §8 L530 is explicit that fallback wins.
	if step.Fallback != "" {
		if err := e.emitStepFailed(ctx, inst, step.ID, attempt, stepErr, false); err != nil {
			return false, err
		}
		e.markFailed(inst.InstanceID, step.ID) // B-4: never re-activate step.ID.
		if err := e.emitStepFallbackActivated(ctx, inst, step.ID, step.Fallback, stepErr.Code); err != nil {
			return false, err
		}
		e.addPending(inst.InstanceID, step.Fallback)
		e.logger.Info("settle", "instance_id", string(inst.InstanceID), "step_id", step.ID, "outcome", "fallback")
		return false, nil
	}

	// (2) on_error transitions — the definition declared an error route.
	targets := firingOnError(dv, step.ID, buildEnv(inst))
	if len(targets) > 0 {
		if err := e.emitStepFailed(ctx, inst, step.ID, attempt, stepErr, false); err != nil {
			return false, err
		}
		e.markFailed(inst.InstanceID, step.ID) // B-4: never re-activate step.ID.
		for _, to := range targets {
			e.addPending(inst.InstanceID, to)
		}
		e.logger.Info("settle", "instance_id", string(inst.InstanceID), "step_id", step.ID, "outcome", "on_error")
		return false, nil
	}

	// (3) workflow failure + compensation.
	if err := e.emitStepFailed(ctx, inst, step.ID, attempt, stepErr, false); err != nil {
		return false, err
	}
	e.markFailed(inst.InstanceID, step.ID) // B-4: never re-activate step.ID.
	if err := e.emitWorkflowFailed(ctx, inst, step.ID, stepErr); err != nil {
		return false, err
	}
	e.logger.Info("settle", "instance_id", string(inst.InstanceID), "step_id", step.ID, "outcome", "failed")
	if err := e.maybeCompensate(ctx, dv, inst, step.ID); err != nil {
		return true, err
	}
	return true, nil
}

// settleFailure is the generalized failure branch of the settle loop. It decides
// retry-vs-terminal per ADJ-6/EDR-011 §3/§8:
//
//   - capability_fallback SKIPS retries entirely (FR-IL-07) and routes terminally.
//   - otherwise, if the just-failed attempt is retry-eligible, emit
//     StepFailed{retrying:true} and schedule the re-dispatch (step stays in
//     current_steps — StepFailed{retrying:true} does not remove it).
//   - else route terminally (routeTerminalFailure).
//
// Returns terminal=true iff the workflow reached a terminal state.
func (e *Engine) settleFailure(ctx context.Context, dv *defView, inst core.WorkflowInstance, step core.Step, attempt int, stepErr core.StepError) (terminal bool, err error) {
	// capability_fallback ⇒ no retry, straight to routing (FR-IL-07, card C).
	if stepErr.Code != "capability_fallback" && retryEligible(step.Retry, attempt, stepErr.Code) {
		if err := e.emitStepFailed(ctx, inst, step.ID, attempt, stepErr, true); err != nil {
			return false, err
		}
		e.scheduleRetry(inst.InstanceID, step.ID, attempt+1, backoffDelay(*step.Retry, attempt))
		e.logger.Info("settle", "instance_id", string(inst.InstanceID), "step_id", step.ID, "outcome", "retrying")
		return false, nil
	}
	// Terminal failure: drop any retry bookkeeping and route.
	e.clearRetry(inst.InstanceID, step.ID)
	return e.routeTerminalFailure(ctx, dv, inst, step, attempt, stepErr)
}
