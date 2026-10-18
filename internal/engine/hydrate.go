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
	"sort"
	"time"

	"github.com/awis/awis/internal/core"
)

// hydrate reconstructs, once per instance per process (guarded by
// e.hydrated), scheduled retries (e.retries), failure-driven pending
// activations (e.pending), and terminally-failed steps (e.failed) from inst's
// durable event stream. It is called from processInstance right after
// defViewFor succeeds and before the cancellation gate. The warm path
// (already hydrated) is a single locked map lookup and an immediate return.
// LA-C1 F1: resetInstanceRecovery re-arms hydration for an instance after a failed
// processInstance or CompleteStep, so RC-1 runs again in this same process.
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
		seq       int            // LA-C3: sequence_num of this StepFailed (commit order)
		err       core.StepError // LA-C3: this StepFailed's payload error
	}
	last := make(map[string]lastFailure)
	relevant := make(map[string]core.EventType)
	startedAttempts := make(map[string]int)
	// startedSeq is the sequence_num of each step's LATEST StepStarted (LA-C1 F3:
	// a delivery counts only if it is later than the activation it resumes).
	startedSeq := make(map[string]int)
	var fallbackTargets []string
	// LA-C3: fallbackAfter[step] is true when a StepFallbackActivated for step follows
	// that step's latest StepFailed; workflowFailed is true when any WorkflowFailed is
	// durable (the instance is then terminal, whatever its projected status says).
	fallbackAfter := make(map[string]bool)
	workflowFailed := false
	// LA-C4a F-P: the latest StepStarted/StepCompleted/StepFailed event per step, kept
	// whole so a projection catch-up re-applies it verbatim.
	latest := make(map[string]core.ExecutionEvent)
	for _, ev := range evs {
		switch ev.EventType {
		case core.EventTypeStepStarted:
			if ev.StepID == "" {
				continue
			}
			relevant[ev.StepID] = ev.EventType
			latest[ev.StepID] = ev
			delete(last, ev.StepID)
			startedSeq[ev.StepID] = ev.SequenceNum
			var p stepStartedPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil {
				startedAttempts[ev.StepID] = p.Attempt
			}

		case core.EventTypeStepCompleted:
			if ev.StepID == "" {
				continue
			}
			relevant[ev.StepID] = ev.EventType
			latest[ev.StepID] = ev
			delete(last, ev.StepID)
			delete(startedAttempts, ev.StepID)

		case core.EventTypeStepFailed:
			if ev.StepID == "" {
				continue
			}
			var p stepFailedPayload
			if err := json.Unmarshal(ev.Payload, &p); err != nil {
				return fmt.Errorf("engine: hydrate decode StepFailed %s/%s: %w", inst.InstanceID, ev.StepID, err)
			}
			relevant[ev.StepID] = ev.EventType
			latest[ev.StepID] = ev
			last[ev.StepID] = lastFailure{retrying: p.Retrying, attempt: p.Attempt, emittedAt: ev.EmittedAt, seq: ev.SequenceNum, err: p.Error}
			delete(startedAttempts, ev.StepID)
			delete(fallbackAfter, ev.StepID) // LA-C3: a newer failure supersedes an older fallback.

		case core.EventTypeWorkflowFailed: // LA-C3
			workflowFailed = true

		case core.EventTypeStepFallbackActivated:
			var p stepFallbackActivatedPayload
			if err := json.Unmarshal(ev.Payload, &p); err != nil {
				return fmt.Errorf("engine: hydrate decode StepFallbackActivated %s: %w", inst.InstanceID, err)
			}
			if p.FallbackStepID != "" {
				fallbackTargets = append(fallbackTargets, p.FallbackStepID)
			}
			if p.StepID != "" { // LA-C3
				fallbackAfter[p.StepID] = true
			}
		}
	}

	// ── LA-C4a F-T: terminal fence (applyTerminalFence, the one shared body; see its comment). A
	// durable workflow-terminal event is the last word for this instance: when the fence applies,
	// hydrate returns WITHOUT RC-1 and WITHOUT committed-route completion, both of which append.
	// processInstance's re-read right after hydrate (`if !found || isTerminalStatus(cur.Status)
	// { return nil }`, tick.go) then stops this instance before any gather. The LA-C7 route-recovery
	// stage (route_recovery.go completeAwaitingRoutes) applies the same fence.
	fenced, err := e.applyTerminalFence(ctx, dv, inst, evs)
	if err != nil {
		return err
	}
	if fenced {
		e.mu.Lock()
		e.hydrated[inst.InstanceID] = true
		e.mu.Unlock()
		return nil
	}

	// ── LA-C4a F-P: settled-step projection catch-up (#6, non-terminal case) ─────
	// A step still in the projected current_steps whose LATEST relevant event is
	// StepCompleted, or StepFailed{retrying:false}, has a durable event the projection
	// did not absorb (its UpsertInstance failed; the LA-C1 reset re-runs this hydrate).
	// Re-apply exactly that event, in sequence order, before RC-1 and before anything
	// below reads the projection (completedSet, the committed-route completion).
	// StepFailed{retrying:true} keeps the step in current_steps by design (project());
	// a latest StepStarted is RC-1's to settle.
	var catchUp []core.ExecutionEvent
	for _, stepID := range inst.CurrentSteps {
		lev, ok := latest[stepID]
		if !ok {
			continue
		}
		switch lev.EventType {
		case core.EventTypeStepCompleted:
			catchUp = append(catchUp, lev)
		case core.EventTypeStepFailed:
			if lf, ok := last[stepID]; ok && !lf.retrying {
				catchUp = append(catchUp, lev)
			}
		}
	}
	if len(catchUp) > 0 {
		sort.Slice(catchUp, func(i, j int) bool { return catchUp[i].SequenceNum < catchUp[j].SequenceNum })
		for _, ev := range catchUp {
			if err := e.project(ctx, ev, inst.DefinitionID, inst.DefinitionVersion); err != nil {
				return fmt.Errorf("engine: hydrate catch up %s %s/%s: %w", ev.EventType, inst.InstanceID, ev.StepID, err)
			}
		}
		cur, found, err := e.getInstance(ctx, inst.InstanceID)
		if err != nil {
			return err
		}
		if found {
			inst = cur
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
	// A target that has ITSELF terminally failed must not be reconstructed as
	// pending. It is caught by neither `completed` (a terminally-failed step
	// never enters Variables) nor `current` (it was removed from
	// current_steps), so without this check a fallback target that later
	// failed on its own would land in BOTH e.pending and e.failed after a
	// restart. activatableFor happens to test failed[] before pend[], so the
	// phantom entry is masked today — but "correct only because of the order
	// of two checks in another file" is not an invariant worth relying on.
	pendingTargets := make(map[string]bool)
	for _, target := range fallbackTargets {
		if completed[target] || current[target] || failedSteps[target] {
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
			if completed[target] || current[target] || failedSteps[target] {
				continue
			}
			pendingTargets[target] = true
		}
	}

	e.mu.Lock()
	if e.hydrated[inst.InstanceID] {
		e.mu.Unlock()
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
	e.mu.Unlock()

	// LA-C3, EDR-011 §8: complete the terminal routes already made durable by
	// StepFailed{retrying:false}, BEFORE RC-1. A committed terminal decision precedes
	// in-flight recovery: in the live path a route-(3) failure stops the serial settle
	// as soon as its StepFailed{retrying:false} is durable (LA-C5 N2; failure.go
	// routeTerminalFailure, tick.go settle loop), so a stranded StepStarted in the same
	// tick is never settled after it.
	cands := make([]committedFailure, 0, len(failedSteps))
	for stepID := range failedSteps {
		lf := last[stepID]
		cands = append(cands, committedFailure{stepID: stepID, seq: lf.seq, err: lf.err, fallbackDone: fallbackAfter[stepID]})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].seq < cands[j].seq })
	// LA-C4a F-C: the cancellation flag is read ONCE here, before RC-1. It gates the
	// committed-route completion (guard (a)) and selects the stranded-step branch below.
	flagged, err := e.cancellationRequested(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if err := e.completeCommittedRoutes(ctx, dv, inst, cands, workflowFailed, flagged); err != nil {
		return fmt.Errorf("engine: hydrate complete committed route %s: %w", inst.InstanceID, err)
	}
	if len(cands) > 0 {
		cur, found, err := e.getInstance(ctx, inst.InstanceID)
		if err != nil {
			return err
		}
		if found {
			if isTerminalStatus(cur.Status) {
				return nil // a completed route ended the workflow; RC-1 must not run.
			}
			inst = cur
		}
	}

	// LA-C4a F-C (Finalization B4 step 3): once cancellation is requested no step
	// activates, so RC-1 must neither re-enter a wait (P3) nor route a failure (P7). Each
	// stranded step (latest StepStarted, still in current_steps) is settled with no routing
	// by settleStrandedCancelled; the cancellation gate in processInstance (tick.go) then
	// finalizes once current_steps drains. The RC-1 loop below is reached only when the
	// flag is clear.
	if flagged {
		for _, stepID := range inst.CurrentSteps {
			if relevant[stepID] != core.EventTypeStepStarted {
				continue
			}
			step, ok := dv.steps[stepID]
			if !ok {
				continue
			}
			if err := e.settleStrandedCancelled(ctx, inst, step, evs, startedSeq[stepID], startedAttempts[stepID]); err != nil {
				return fmt.Errorf("engine: hydrate cancel stranded step %s: %w", stepID, err)
			}
			cur, found, err := e.getInstance(ctx, inst.InstanceID)
			if err != nil {
				return err
			}
			if found {
				inst = cur
			}
		}
		return nil
	}

	// In-flight crash recovery (RC-1): any step in current_steps whose latest
	// event is StepStarted was running in a previous worker process that crashed
	// or was killed before settling. Settle it with worker_crash so it can either
	// retry under its retry policy or route terminally.
	for _, stepID := range inst.CurrentSteps {
		if relevant[stepID] == core.EventTypeStepStarted {
			step, ok := dv.steps[stepID]
			if !ok {
				continue
			}
			// LA-C1 F3: a signal step is never worker_crash. Its only completion path
			// is the wait, so RC-1 either completes it from a durable delivery or
			// re-enters the wait (recoverSignalStep). Non-signal steps are unchanged.
			if step.Type == core.StepTypeSignal {
				if err := e.recoverSignalStep(ctx, inst, step, evs, startedSeq[stepID]); err != nil {
					return fmt.Errorf("engine: hydrate recover signal step %s: %w", stepID, err)
				}
				cur, found, err := e.getInstance(ctx, inst.InstanceID)
				if err != nil {
					return err
				}
				if found {
					inst = cur
				}
				continue
			}
			attempt := startedAttempts[stepID]
			if attempt <= 0 {
				attempt = 1
			}
			serr := core.StepError{
				Code:    "worker_crash",
				Message: "worker process crashed while step was in-flight",
			}
			terminal, err := e.settleFailure(ctx, dv, inst, step, attempt, serr)
			if err != nil {
				return fmt.Errorf("engine: hydrate recover crashed step %s: %w", stepID, err)
			}
			if terminal {
				return nil
			}
			if cur, found, err := e.getInstance(ctx, inst.InstanceID); err == nil && found {
				inst = cur
			}
		}
	}

	return nil
}

// workflowTerminalClass reports whether et is a workflow-lifecycle event that ends the
// instance (the five terminal statuses) or opens the compensation bracket
// (WorkflowCompensating; compensation.go runCompensation). LA-C4a F-T consults only the
// last such event in the log.
func workflowTerminalClass(et core.EventType) bool {
	switch et {
	case core.EventTypeWorkflowCompleted, core.EventTypeWorkflowFailed,
		core.EventTypeWorkflowCancelled, core.EventTypeWorkflowCompensating,
		core.EventTypeWorkflowCompensated, core.EventTypeWorkflowCompensationFailed:
		return true
	}
	return false
}

// reprojectTerminal re-applies the durable workflow-terminal event ev to the projection
// when the projection still reads non-terminal (LA-C4a F-T; P2 / #1, #6-terminal). Its
// failure is returned, so the LA-C1 reset retries it on the next tick; it is never
// swallowed and RC-1 never runs in its place. No-op when the projection is already
// terminal or the row is absent.
func (e *Engine) reprojectTerminal(ctx context.Context, inst core.WorkflowInstance, ev core.ExecutionEvent) error {
	cur, found, err := e.getInstance(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if !found || isTerminalStatus(cur.Status) {
		return nil
	}
	if err := e.project(ctx, ev, inst.DefinitionID, inst.DefinitionVersion); err != nil {
		return fmt.Errorf("engine: hydrate re-project %s %s: %w", ev.EventType, inst.InstanceID, err)
	}
	return nil
}

// terminalFenceState returns the last workflow-terminal-class event of evs (nil when the log has
// none; see workflowTerminalClass) and whether the log holds any WorkflowCompensating (the
// compensation bracket was opened). An owed compensation needs none (applyTerminalFence).
func terminalFenceState(evs []core.ExecutionEvent) (*core.ExecutionEvent, bool) {
	var last *core.ExecutionEvent
	sawCompensating := false
	for _, ev := range evs {
		if workflowTerminalClass(ev.EventType) {
			evCopy := ev
			last = &evCopy
		}
		if ev.EventType == core.EventTypeWorkflowCompensating {
			sawCompensating = true
		}
	}
	return last, sawCompensating
}

// applyTerminalFence is the terminal fence (LA-C4a F-T; EDR-011 §6; LA-C5 N1). It is the ONE body that
// re-projects a durable workflow-terminal event and completes its owed compensation: hydrate calls it on
// first touch, and the LA-C7 route-recovery stage (route_recovery.go completeAwaitingRoutes) calls it
// before it gates or completes a committed route (LA-C7 HIGH-1/2). Both callers apply the same rules.
//
// It applies when the log's last workflow-terminal-class event is not WorkflowCompensating; applied
// reports whether it did, and the caller then stops its own recovery. Successor rule (compensation.go
// runCompensation, EDR-011 §6): the only lifecycle event that may follow WorkflowFailed is the
// compensation bracket opener WorkflowCompensating, and only when the definition has a plan and ≥1 step
// completed. Compensation-in-progress recovery is LA-D1 (deferred): when the last terminal-class event is
// WorkflowCompensating the bracket is open, so nothing is done here (applied is false). Errors are
// returned, never swallowed.
func (e *Engine) applyTerminalFence(ctx context.Context, dv *defView, inst core.WorkflowInstance, evs []core.ExecutionEvent) (bool, error) {
	lastTerm, sawCompensating := terminalFenceState(evs)
	if lastTerm == nil || lastTerm.EventType == core.EventTypeWorkflowCompensating {
		return false, nil
	}
	if err := e.reprojectTerminal(ctx, inst, *lastTerm); err != nil {
		return true, err
	}
	// LA-C4a: completes the committed route (3) — compensation owed by a durable
	// WorkflowFailed (EDR-011 §6). Route (3) is failure.go's emitWorkflowFailed followed by
	// maybeCompensate; a crash or projection fault between the two leaves the bracket
	// owed. Only the case with no WorkflowCompensating in the log is owed; maybeCompensate
	// applies the plan and ≥1-completed-step rule itself.
	if lastTerm.EventType == core.EventTypeWorkflowFailed && !sawCompensating && dv.def.Compensation != nil {
		// LA-C5 N1: owed only for a WorkflowFailed that NAMES a step. CE-verified emission map
		// (grep over the tracked tree, base db9fd36): emitWorkflowFailed has four call sites.
		// Each non-empty step_id site runs maybeCompensate on its no-fault path: failure.go
		// routeTerminalFailure route (3) (also reached by the signal_timeout "fail" action);
		// hydrate.go completeCommittedRoutes route (3) (LA-C3); signal_timeout.go
		// routeTimedOutWait (timeout_action compensate). The only empty step_id site, the
		// processInstance completion check in tick.go (stalled, SEC-08), never compensates, so
		// its recovery must not compensate either.
		var wf workflowFailedPayload
		if err := json.Unmarshal(lastTerm.Payload, &wf); err != nil {
			return true, fmt.Errorf("engine: hydrate decode WorkflowFailed %s: %w", inst.InstanceID, err)
		}
		cur, found, err := e.getInstance(ctx, inst.InstanceID)
		if err != nil {
			return true, err
		}
		if found && wf.StepID != "" { // LA-C5 N1: see the emission map above.
			if err := e.maybeCompensate(ctx, dv, cur, wf.StepID); err != nil {
				return true, fmt.Errorf("engine: hydrate owed compensation %s: %w", inst.InstanceID, err)
			}
		}
	}
	return true, nil
}

// settleStrandedCancelled settles one stranded StepStarted while cancellation is
// requested (LA-C4a F-C; Finalization B4 step 3). It never re-enters a wait and never
// routes: no fallback, no on_error, no WorkflowFailed.
//
//   - signal step: a SignalReceived after its latest StepStarted completes it through
//     CompleteStep (completing an in-flight step is not an activation). Otherwise its
//     wait_record is deleted and StepFailed{code:"cancelled", retrying:false} is emitted
//     with cancel.go Cancel's waiting-step message.
//   - non-signal step: StepFailed{code:"worker_crash", retrying:false} with RC-1's message,
//     and nothing else (no routeTerminalFailure).
//
// Both failure paths markFailed the step so it is never re-activated (B-4).
func (e *Engine) settleStrandedCancelled(ctx context.Context, inst core.WorkflowInstance, step core.Step, evs []core.ExecutionEvent, startedSeq, startedAttempt int) error {
	iid := inst.InstanceID
	if step.Type == core.StepTypeSignal {
		if step.WaitSignal != nil {
			payload, delivered, err := signalDeliveredAfter(evs, step.WaitSignal.SignalName, startedSeq)
			if err != nil {
				return err
			}
			if delivered {
				return e.CompleteStep(ctx, iid, step.ID, payload)
			}
		}
		if ds, ok := e.storage.(signalWaitDeleteStore); ok {
			if err := ds.DeleteWaitRecord(ctx, iid, step.ID); err != nil {
				return fmt.Errorf("engine: cancel stranded wait_record %s/%s: %w", iid, step.ID, err)
			}
		}
		serr := core.StepError{Code: "cancelled", Message: "step cancelled while waiting for signal"}
		if err := e.emitStepFailed(ctx, inst, step.ID, 1, serr, false); err != nil {
			return err
		}
		e.markFailed(iid, step.ID)
		e.log().Info("settle", "instance_id", string(iid), "step_id", step.ID, "outcome", "cancelled")
		return nil
	}
	attempt := startedAttempt
	if attempt <= 0 {
		attempt = 1
	}
	serr := core.StepError{Code: "worker_crash", Message: "worker process crashed while step was in-flight"}
	if err := e.emitStepFailed(ctx, inst, step.ID, attempt, serr, false); err != nil {
		return err
	}
	e.markFailed(iid, step.ID)
	e.log().Info("settle", "instance_id", string(iid), "step_id", step.ID, "outcome", "cancelled")
	return nil
}

// committedFailure is one terminally-failed step, as hydrate reconstructs it for
// LA-C3 route completion: StepFailed{retrying:false} is its latest relevant event.
type committedFailure struct {
	stepID       string
	seq          int            // sequence_num of the latest StepFailed (commit order)
	err          core.StepError // the latest StepFailed payload's error
	fallbackDone bool           // a StepFallbackActivated(stepID) follows that StepFailed
}

// completeCommittedRoutes completes, after a transient append failure, the terminal
// routing that StepFailed{retrying:false} already made durable (LA-C3, L7/L8).
// routeTerminalFailure emits the follow-on event right after StepFailed; when that
// append failed, the route was lost and the instance stalled into SEC-08. cands is in
// commit order, so the earliest committed decision is completed first.
//
//   - route (1), Fallback set: when no StepFallbackActivated follows the StepFailed,
//     emit it (same helper and payload as routeTerminalFailure) and mark the fallback
//     pending.
//   - route (2), or any step with an on_error transition: never completed here (static
//     classification, below); the pending block in hydrate re-derives those targets.
//   - route (3): WorkflowFailed carrying the step's id and error, then maybeCompensate,
//     exactly as routeTerminalFailure does. The instance is terminal afterwards, so no
//     later candidate is applied.
//
// Route classification uses ONLY static definition structure (CE ruling LA-C3 round 2):
// route (1) iff step.Fallback != ""; route (3) only for a step with no Fallback and NO
// on_error transition at all, conditional or not. A condition-dependent route is never
// decided here. LA-C3: route decided from static structure only; a condition-dependent
// route is not reconstructable from the EventLog (residual R-1, pre-existing).
//
// Guards (CE ruling LA-C3, binding; all applied, belt and braces). None of these
// StepFailed rows carries a routing decision, so no route is completed for them:
//   - (a) cancellationRequested: the instance is cancel-requested; the B4 cancellation
//     gate owns it (Finalization B4 step 3; cancel.go Cancel and handleCancellation).
//   - (b) the step's latest StepFailed code is "cancelled": cancel.go Cancel and handleCancellation emit it
//     with no routing decision (Finalization B4 step 3).
//   - (c) the step's latest StepFailed code is "signal_timeout": signal_timeout.go processTimeout
//     deletes the wait_record before routing, and the compensate action (routeTimedOutWait)
//     skips fallback and on_error. The timeout action is not durable after expiry, so
//     recovery cannot reconstruct which route was decided.
//
// Also: no WorkflowFailed may be durable. A WorkflowFailed event is checked directly,
// not only through the projected status, because emit appends before it projects.
//
// flagged is the instance's cancellation flag, read once by hydrate (LA-C4a F-C).
func (e *Engine) completeCommittedRoutes(ctx context.Context, dv *defView, inst core.WorkflowInstance, cands []committedFailure, workflowFailed, flagged bool) error {
	if len(cands) == 0 || workflowFailed {
		return nil
	}
	if flagged {
		return nil // guard (a): B4 cancellation gate owns a cancel-requested instance.
	}
	cur, found, err := e.getInstance(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if !found || isTerminalStatus(cur.Status) {
		return nil
	}
	for _, c := range cands {
		if c.err.Code == "cancelled" {
			continue // guard (b): cancel.go Cancel and handleCancellation emit this code with no routing decision.
		}
		if c.err.Code == "signal_timeout" {
			continue // guard (c): signal_timeout.go processTimeout and routeTimedOutWait (compensate); the action is not durable.
		}
		step, ok := dv.steps[c.stepID]
		if !ok {
			continue
		}
		switch {
		case step.Fallback != "": // route (1)
			if c.fallbackDone {
				continue
			}
			if err := e.emitStepFallbackActivated(ctx, cur, c.stepID, step.Fallback, c.err.Code); err != nil {
				return err
			}
			e.addPending(cur.InstanceID, step.Fallback)
			e.log().Info("settle", "instance_id", string(cur.InstanceID), "step_id", c.stepID, "outcome", "fallback")
		case staticRouteThree(dv, step): // route (3): the static classification (LA-C3, LA-C7 LOW-3).
			if err := e.emitWorkflowFailed(ctx, cur, c.stepID, c.err); err != nil {
				return err
			}
			e.log().Info("settle", "instance_id", string(cur.InstanceID), "step_id", c.stepID, "outcome", "failed")
			return e.maybeCompensate(ctx, dv, cur, c.stepID)
		default: // route (2): an on_error transition (static). LA-C3: route decided from static
			// structure only; a condition-dependent route is not reconstructable from the EventLog
			// (residual R-1, pre-existing). Never completed here.
			continue
		}
	}
	return nil
}

// hasOnErrorTransition reports whether stepID has ANY on_error transition in the
// definition, conditional or not. It reads static structure only (LA-C3 round 2):
// buildEnv-dependent conditions are never consulted by route completion.
func hasOnErrorTransition(dv *defView, stepID string) bool {
	for _, t := range dv.def.Transitions {
		if t.From == stepID && t.OnError {
			return true
		}
	}
	return false
}

// staticRouteThree reports whether step's terminal failure is route (3) by static definition
// structure alone (LA-C3, CE ruling round 2): no Fallback and no on_error transition at all,
// conditional or not. It is the ONE static route-(3) predicate: completeCommittedRoutes (above)
// and routeThreeAwaiting (route_recovery.go) both call it. tick.go's committedRouteThree is the
// settle-time, environment-based classification (LA-C6 R3-H1) and is deliberately NOT this predicate.
func staticRouteThree(dv *defView, step core.Step) bool {
	return step.Fallback == "" && !hasOnErrorTransition(dv, step.ID)
}

// resetInstanceRecovery discards this process's recovery bookkeeping for iid, so
// the next processInstance re-hydrates iid purely from the durable EventLog, exactly
// as a freshly started process would (LA-C1 F1).
//
// Why: RC-1 (in-flight recovery, above) runs once per instance per process, guarded
// by e.hydrated, which is set BEFORE RC-1 runs. A transient storage failure during
// RC-1, or in a settle/wait write that follows a StepStarted, therefore left that
// StepStarted unsettled for the rest of the process's life (LA-C1 leads L1-L6). The
// restart semantics are the contract (worker_crash for non-signal steps; the F3 rules
// for signal steps); this makes them reachable without a restart.
//
// Cleared, each because hydrate reconstructs it from storage: e.hydrated (re-arms
// RC-1), e.retries (from StepFailed{retrying:true}), e.pending (from
// StepFallbackActivated and from on_error targets of a terminal StepFailed), e.failed
// (from a terminal StepFailed), and e.claimLostStreak (a diagnostic counter that a
// restart also resets).
//
// Kept, because each is already re-read from durable storage on use or is not
// recovery state:
//   - e.ver: expectedVersion re-reads workflow_instances.version from storage on every
//     projection write and every Resolve; the cache is never trusted alone.
//   - e.seq: a monotone append cursor, not recovery state. A gap left by a failed
//     append is harmless: ordering is all that hydrate and the projection read, and
//     internal/storage has no sequence_num contiguity check.
//   - e.waits: a hit is removed by OnDelivered/clearWait; a miss is re-derived from the
//     durable wait_records by recoverWait (Resolve), exactly as after a restart.
//   - e.cancels, e.defs: a cancel intent is not reconstructed from the EventLog and is
//     not recovery bookkeeping; the definition cache is keyed by immutable (id, version).
//   - e.routeRecovery (LA-C6 R3-M1): re-derived from the EventLog on every check
//     (committedRouteAwaitingTerminal, route_recovery.go); not reset here.
//
// Call-site invariant (verified at each caller): no dispatch goroutine for iid is in
// flight. processInstance joins its dispatch goroutines (wg.Wait) before SETTLE, and
// Tick and SIGNAL_SCAN are serial, so the reset cannot race a running step.
func (e *Engine) resetInstanceRecovery(iid core.InstanceID) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.hydrated, iid)
	delete(e.pending, iid)
	delete(e.failed, iid)
	for key := range e.retries {
		if key.iid == iid {
			delete(e.retries, key)
		}
	}
	for key := range e.claimLostStreak {
		if key.iid == iid {
			delete(e.claimLostStreak, key)
		}
	}
}

// recoverSignalStep settles a signal step whose latest event is StepStarted (LA-C1
// F3). A signal step's only completion path is its wait, so RC-1 must never fail it
// as worker_crash:
//
//	(a) the awaited signal was delivered AFTER this activation's StepStarted: the
//	    delivery is durable (SignalReceived is appended in the same B3 transaction
//	    that flips the instance to running, Finalization B3), so the step completes
//	    with that payload via CompleteStep. CompleteStep tolerates an already-deleted
//	    wait_record (DeleteWaitRecord is a benign no-op when the row is absent);
//	(b) otherwise the wait was never durably completed, so it is re-entered via
//	    enterWait, which is idempotent over an existing wait_record (L4b).
//
// validate.Validate rejects a signal step without wait_signal; enterWait returns a
// typed error should one ever reach this path.
func (e *Engine) recoverSignalStep(ctx context.Context, inst core.WorkflowInstance, step core.Step, evs []core.ExecutionEvent, startedSeq int) error {
	if step.WaitSignal != nil {
		payload, delivered, err := signalDeliveredAfter(evs, step.WaitSignal.SignalName, startedSeq)
		if err != nil {
			return err
		}
		if delivered {
			return e.CompleteStep(ctx, inst.InstanceID, step.ID, payload)
		}
	}
	return e.enterWait(ctx, inst, step)
}

// signalReceivedRecord decodes a SignalReceived payload: {signal_name, payload}
// (EVENTLOG_FORMAT §6; field names verbatim). It mirrors signal.signalReceivedPayload,
// which is unexported and therefore not importable from the engine.
type signalReceivedRecord struct {
	SignalName string         `json:"signal_name"`
	Payload    map[string]any `json:"payload"`
}

// signalDeliveredAfter returns the payload of the EARLIEST SignalReceived for
// signalName whose sequence_num is greater than afterSeq (the awaited step's latest
// StepStarted). Earliest, not latest: while this step waits, the first same-name
// delivery is the one that resumed it, and the wait is cleared on delivery
// (OnDelivered), so a later same-name delivery belongs to a later wait.
func signalDeliveredAfter(evs []core.ExecutionEvent, signalName string, afterSeq int) (map[string]any, bool, error) {
	var best map[string]any
	bestSeq := 0
	found := false
	for _, ev := range evs {
		if ev.EventType != core.EventTypeSignalReceived || ev.SequenceNum <= afterSeq {
			continue
		}
		var rec signalReceivedRecord
		if err := json.Unmarshal(ev.Payload, &rec); err != nil {
			return nil, false, fmt.Errorf("engine: hydrate decode SignalReceived seq=%d: %w", ev.SequenceNum, err)
		}
		if rec.SignalName != signalName {
			continue
		}
		if !found || ev.SequenceNum < bestSeq {
			found = true
			bestSeq = ev.SequenceNum
			best = rec.Payload
		}
	}
	return best, found, nil
}
