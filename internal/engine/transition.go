package engine

import (
	"sort"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/expr"
)

// buildEnv builds the expression environment from an instance's projection
// (EDR-007 variable shape). Completed steps are the variables keys except the
// reserved "inputs"; running steps are current_steps. Event is nil (trigger-scope
// only; C1 has no event context in transition/template evaluation).
func buildEnv(inst core.WorkflowInstance) expr.Env {
	var inputs map[string]any
	if v, ok := inst.Variables["inputs"]; ok {
		inputs, _ = v.(map[string]any)
	}
	stepOutputs := make(map[string]map[string]any)
	stepStatus := make(map[string]string)
	for k, v := range inst.Variables {
		if k == "inputs" {
			continue
		}
		if m, ok := v.(map[string]any); ok {
			stepOutputs[k] = m
		}
		stepStatus[k] = "completed"
	}
	for _, s := range inst.CurrentSteps {
		stepStatus[s] = "running"
	}
	return expr.Env{Inputs: inputs, StepOutputs: stepOutputs, StepStatus: stepStatus}
}

// completedSet returns the set of completed step ids (variables keys − "inputs").
func completedSet(inst core.WorkflowInstance) map[string]bool {
	out := make(map[string]bool, len(inst.Variables))
	for k := range inst.Variables {
		if k != "inputs" {
			out[k] = true
		}
	}
	return out
}

// transitionFires reports whether a forward (on-success) transition fires given
// env: an on_error transition never fires here (it fires only on a terminal
// StepFailed, which C1 routes straight to WorkflowFailed — on_error target
// activation is part of the failure-handling generalization owned by C2). A
// forward transition with no condition always fires; with a condition it fires
// iff the condition evaluates true (EDR-011 §1, T3).
func transitionFires(te transEval, env expr.Env) bool {
	if te.t.OnError {
		return false // C2 seam: on_error target activation.
	}
	if te.cond == nil {
		return true
	}
	ok, err := te.cond.Eval(env)
	if err != nil {
		// ConditionExpr.Eval never errors in practice (frozen null rules); a
		// defensive false is the safe non-activation outcome.
		return false
	}
	return ok
}

// activatableSteps derives — statelessly, from the projection — the ordered list
// of steps eligible to CLAIM this tick (EDR-011 §1 join gate). Because the
// derivation reads only completed step outputs (fixed once a step completes) and
// current_steps, it is rebuild-safe: any tick reconstructs the same set.
//
// Rule (EDR-011 §1): a step activates iff every DISTINCT `from` among its inbound
// transitions has completed AND ≥1 inbound transition fires; a step with no
// inbound transitions activates only as the InitialStep at the very start. Steps
// already completed or running are excluded. Steps whose upstream branch was
// skipped stall (author semantics).
//
// semantics-bearing: join gate + fan-out convergence (EDR-011 §1).
func activatableSteps(dv *defView, inst core.WorkflowInstance) []string {
	completed := completedSet(inst)
	running := make(map[string]bool, len(inst.CurrentSteps))
	for _, s := range inst.CurrentSteps {
		running[s] = true
	}
	env := buildEnv(inst)

	var out []string
	for _, step := range dv.def.Steps { // definition order (deterministic dispatch)
		id := step.ID
		if completed[id] || running[id] {
			continue
		}
		if isActivatable(dv, id, completed, running, env) {
			out = append(out, id)
		}
	}
	return out
}

// isActivatable applies the join gate to a single candidate step.
func isActivatable(dv *defView, id string, completed, running map[string]bool, env expr.Env) bool {
	inbound := dv.inbound[id]
	if len(inbound) == 0 {
		// Only the InitialStep self-activates, and only at the very start (before
		// anything has completed or is running). Fallback targets (C2) are the
		// other zero-inbound activation source.
		return id == dv.def.InitialStep && len(completed) == 0 && len(running) == 0
	}

	// (a) every distinct upstream `from` has completed.
	distinctFroms := make(map[string]bool, len(inbound))
	for _, te := range inbound {
		distinctFroms[te.t.From] = true
	}
	for from := range distinctFroms {
		if !completed[from] {
			return false
		}
	}

	// (b) at least one inbound transition fires.
	for _, te := range inbound {
		if transitionFires(te, env) {
			return true
		}
	}
	return false
}

// completedFinalOutputs returns the WorkflowCompleted outputs map:
// map[final_step_id] → that step's outputs, for every FinalStep that completed
// (EDR-011 §2). Keys are emitted in sorted order for determinism.
func completedFinalOutputs(dv *defView, inst core.WorkflowInstance) map[string]any {
	out := make(map[string]any)
	ids := make([]string, 0, len(dv.finalSet))
	for id := range dv.finalSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if v, ok := inst.Variables[id]; ok {
			out[id] = v
		}
	}
	return out
}
