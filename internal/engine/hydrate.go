package engine

// hydrate.go — engine-hardening Step 2a: per-instance hydration of retry /
// pending / terminally-failed bookkeeping from the durable EventLog.
//
// The engine keeps three per-instance maps (e.retries, e.pending, e.failed) as
// V1 single-process runtime state (EDR-011 §8: "reconstructible from the
// EventLog on restart... a crash loses these and the recovery derivation is
// not exercised at V1"). This closes that gap for the steady-state restart
// case (a fresh process resuming ticking): the FIRST time this process
// touches an instance (guarded by e.hydrated), hydrate replays its event
// stream once and reconstructs exactly the bookkeeping a live engine would
// have accumulated, so a restarted process resumes retries/pending
// activations/terminal failures rather than silently forgetting them.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// hydrate reconstructs, once per instance per process (guarded by
// e.hydrated), scheduled retries (e.retries), failure-driven pending
// activations (e.pending), and terminally-failed steps (e.failed) from inst's
// durable event stream. It is called from processInstance right after
// defViewFor succeeds and before the cancellation gate. The warm path
// (already hydrated) is a single locked map lookup and an immediate return.
func (e *Engine) hydrate(ctx context.Context, dv *defView, inst core.WorkflowInstance) error {
	e.mu.Lock()
	already := e.hydrated[inst.InstanceID]
	e.mu.Unlock()
	if already {
		return nil
	}

	evs, err := e.storage.ReadEvents(ctx, inst.InstanceID, 0)
	if err != nil {
		return fmt.Errorf("engine: hydrate read events %s: %w", inst.InstanceID, err)
	}

	// lastFailure holds the StepFailed detail needed to reconstruct either a
	// retry schedule or terminal-failure membership; relevant tracks, per step
	// id, which of {StepStarted, StepCompleted, StepFailed} was most recently
	// observed — a step is only a live retry/failure candidate when StepFailed
	// is the LATEST of those three (a later StepStarted/StepCompleted means the
	// step moved on, e.g. it was successfully retried).
	//
	// StepFallbackActivated is handled separately: TDS-01 §2 records its
	// step_id as the ORIGINATING (failed) step, not the fallback target, so it
	// never changes "the latest relevant event" for that step id.
	type lastFailure struct {
		retrying  bool
		attempt   int
		emittedAt time.Time
	}
	last := make(map[string]lastFailure)
	relevant := make(map[string]core.EventType)
	var fallbackTargets []string

	for _, ev := range evs {
		switch ev.EventType {
		case core.EventTypeStepStarted, core.EventTypeStepCompleted:
			if ev.StepID == "" {
				continue
			}
			relevant[ev.StepID] = ev.EventType
			delete(last, ev.StepID)

		case core.EventTypeStepFailed:
			if ev.StepID == "" {
				continue
			}
			var p stepFailedPayload
			if err := json.Unmarshal(ev.Payload, &p); err != nil {
				return fmt.Errorf("engine: hydrate decode StepFailed %s/%s: %w", inst.InstanceID, ev.StepID, err)
			}
			relevant[ev.StepID] = ev.EventType
			last[ev.StepID] = lastFailure{retrying: p.Retrying, attempt: p.Attempt, emittedAt: ev.EmittedAt}

		case core.EventTypeStepFallbackActivated:
			var p stepFallbackActivatedPayload
			if err := json.Unmarshal(ev.Payload, &p); err != nil {
				return fmt.Errorf("engine: hydrate decode StepFallbackActivated %s: %w", inst.InstanceID, err)
			}
			if p.FallbackStepID != "" {
				fallbackTargets = append(fallbackTargets, p.FallbackStepID)
			}
		}
	}

	completed := completedSet(inst)
	current := make(map[string]bool, len(inst.CurrentSteps))
	for _, s := range inst.CurrentSteps {
		current[s] = true
	}

	// Staged in the same type the engine stores, so the copy under e.mu below
	// is a plain assignment and the two shapes cannot drift apart.
	retries := make(map[string]retrySched)
	failedSteps := make(map[string]bool)

	for stepID, et := range relevant {
		if et != core.EventTypeStepFailed {
			continue // superseded by a later StepStarted/StepCompleted.
		}
		lf := last[stepID]
		if lf.retrying {
			step, ok := dv.steps[stepID]
			if !ok || step.Retry == nil {
				continue // no retry policy to schedule against (defensive; ADJ-6).
			}
			retries[stepID] = retrySched{
				nextAttempt:   lf.attempt + 1,
				nextAttemptAt: lf.emittedAt.Add(backoffDelay(*step.Retry, lf.attempt)),
			}
			continue
		}
		// StepFailed{retrying:false} with no later StepStarted/StepCompleted
		// (relevant[stepID] would be that later type instead) — terminal (B-4).
		failedSteps[stepID] = true
	}

	// pending: fallback targets, then on_error targets for failed steps that did
	// NOT route through fallback (fallback wins over on_error, EDR-011 §8 — a
	// step with step.Fallback set never reaches routeTerminalFailure's on_error
	// branch, so recomputing firingOnError for it here would wrongly resurrect
	// a target that never actually fired live).
	pendingTargets := make(map[string]bool)
	for _, target := range fallbackTargets {
		if completed[target] || current[target] {
			continue
		}
		pendingTargets[target] = true
	}
	for stepID := range failedSteps {
		step, ok := dv.steps[stepID]
		if !ok || step.Fallback != "" {
			continue
		}
		for _, target := range firingOnError(dv, stepID, buildEnv(inst)) {
			if completed[target] || current[target] {
				continue
			}
			pendingTargets[target] = true
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.hydrated[inst.InstanceID] {
		return nil // a concurrent hydrate for this instance already ran.
	}
	for stepID, re := range retries {
		e.retries[retryKey{iid: inst.InstanceID, step: stepID}] = re
	}
	if len(pendingTargets) > 0 {
		if e.pending[inst.InstanceID] == nil {
			e.pending[inst.InstanceID] = make(map[string]bool, len(pendingTargets))
		}
		for target := range pendingTargets {
			e.pending[inst.InstanceID][target] = true
		}
	}
	if len(failedSteps) > 0 {
		if e.failed[inst.InstanceID] == nil {
			e.failed[inst.InstanceID] = make(map[string]bool, len(failedSteps))
		}
		for stepID := range failedSteps {
			e.failed[inst.InstanceID][stepID] = true
		}
	}
	e.hydrated[inst.InstanceID] = true
	return nil
}
