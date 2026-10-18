package engine

// LA-C6 R3-M1 (PR #4 review round 3): a committed route (3) decision whose WorkflowFailed append
// failed leaves its instance `waiting` while a parked sibling still waits on a signal or a timer.
// The no-fault run never delivers or times out that sibling: after WorkflowFailed the instance is
// terminal, and B3 rejects a delivery to a non-waiting instance. Without the mechanism below,
// SIGNAL_SCAN and SIGNAL_TIMEOUT_SCAN resume the sibling, and the faulted run diverges (its
// compensation includes the sibling's undo, and the workflow may complete through it). Three parts:
//
//	(1) committedRouteAwaitingTerminal reads the EventLog for a committed route (3) decision that
//	    has no workflow-terminal event after it (LA-C3's static rule, with its guards);
//	(2) the gate (routeRecoveryGate) refuses a signal delivery (Engine.Resolve) and a timeout
//	    (signalTimeoutScan) while such a decision awaits its terminal event, and records the
//	    instance in e.routeRecovery;
//	(3) completeRouteRecovery (tick, before SIGNAL_SCAN) completes the committed route for the
//	    recorded instances and for every instance whose processInstance failed while it was not
//	    running. It runs ONLY the committed-route completion (completeCommittedRoutes): it never
//	    calls gatherDispatch and never runs RC-1 (hydrate's in-flight recovery).
//
// Scope: LA-C6 R3-M1: completes an already-committed route (3); not the LA-D1 recovery stage
// (design pending founder approval). A condition-dependent route (settle-time route (3) through an
// on_error edge, LA-C6 R3-H1) is outside this rule: the EventLog cannot reproduce its route
// (LA-D2 / EDR-012, pending).

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/awis/awis/internal/core"
)

// logFailedCandidates derives the terminally failed steps of evs with hydrate's own rules
// (hydrate.go: relevant, last and fallbackAfter): a step is a candidate when its latest relevant
// event (StepStarted, StepCompleted or StepFailed) is a StepFailed{retrying:false}. The result is
// in commit order. A payload that cannot be decoded is an error, as hydrate treats it.
func logFailedCandidates(evs []core.ExecutionEvent) ([]committedFailure, error) {
	type latestFail struct {
		seq      int
		err      core.StepError
		retrying bool
	}
	relevant := make(map[string]core.EventType)
	last := make(map[string]latestFail)
	fallbackAfter := make(map[string]bool)
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
				return nil, fmt.Errorf("engine: route recovery decode StepFailed %s/%s seq=%d: %w", ev.InstanceID, ev.StepID, ev.SequenceNum, err)
			}
			relevant[ev.StepID] = ev.EventType
			last[ev.StepID] = latestFail{seq: ev.SequenceNum, err: p.Error, retrying: p.Retrying}
			delete(fallbackAfter, ev.StepID)
		case core.EventTypeStepFallbackActivated:
			var p stepFallbackActivatedPayload
			if err := json.Unmarshal(ev.Payload, &p); err != nil {
				return nil, fmt.Errorf("engine: route recovery decode StepFallbackActivated %s seq=%d: %w", ev.InstanceID, ev.SequenceNum, err)
			}
			if p.StepID != "" {
				fallbackAfter[p.StepID] = true
			}
		}
	}
	var out []committedFailure
	for stepID, et := range relevant {
		if et != core.EventTypeStepFailed {
			continue
		}
		lf := last[stepID]
		if lf.retrying {
			continue
		}
		out = append(out, committedFailure{stepID: stepID, seq: lf.seq, err: lf.err, fallbackDone: fallbackAfter[stepID]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	return out, nil
}

// hasWorkflowTerminal reports whether evs holds a workflow-terminal event (WorkflowCompleted,
// WorkflowFailed or WorkflowCancelled). When afterSeq is >= 0 only events with a greater
// sequence_num count. Nothing may be appended after such an event (LA-C4a F-T, hydrate.go).
func hasWorkflowTerminal(evs []core.ExecutionEvent, afterSeq int) bool {
	for _, ev := range evs {
		if ev.SequenceNum <= afterSeq {
			continue
		}
		switch ev.EventType {
		case core.EventTypeWorkflowCompleted, core.EventTypeWorkflowFailed, core.EventTypeWorkflowCancelled:
			return true
		}
	}
	return false
}

// routeThreeAwaiting is the pure rule of committedRouteAwaitingTerminal (LA-C6 R3-M1). It returns
// the first committed route (3) decision in commit order that has no workflow-terminal event after
// it. A candidate counts only with LA-C3's static route (3): no Fallback and no on_error transition
// (hasOnErrorTransition). Codes "cancelled" and "signal_timeout" carry no routing decision (LA-C3
// guards (b) and (c)). The step must be defined in dv.
func routeThreeAwaiting(dv *defView, evs []core.ExecutionEvent) (string, bool, error) {
	cands, err := logFailedCandidates(evs)
	if err != nil {
		return "", false, err
	}
	for _, c := range cands {
		if c.err.Code == "cancelled" || c.err.Code == "signal_timeout" {
			continue
		}
		step, ok := dv.steps[c.stepID]
		if !ok || step.Fallback != "" || hasOnErrorTransition(dv, c.stepID) {
			continue
		}
		if hasWorkflowTerminal(evs, c.seq) {
			continue
		}
		return c.stepID, true, nil
	}
	return "", false, nil
}

// committedRouteAwaitingTerminal reports whether iid holds a committed route (3) decision with no
// workflow-terminal event after it (LA-C6 R3-M1), and returns that step's id when it does. A
// cancellation request is guard (a) of LA-C3: the B4 cancellation gate owns that instance, so
// nothing is reported. A read or decode error is returned; the caller refuses the action it guards.
func (e *Engine) committedRouteAwaitingTerminal(ctx context.Context, iid core.InstanceID) (string, bool, error) {
	cur, found, err := e.getInstance(ctx, iid)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}
	dv, err := e.defViewFor(ctx, cur)
	if err != nil {
		return "", false, err
	}
	flagged, err := e.cancellationRequested(ctx, iid)
	if err != nil {
		return "", false, err
	}
	if flagged {
		return "", false, nil
	}
	evs, err := e.storage.ReadEvents(ctx, iid, 0)
	if err != nil {
		return "", false, fmt.Errorf("engine: committed route check %s: %w", iid, err)
	}
	return routeThreeAwaiting(dv, evs)
}

// completeAwaitingRoutes completes the committed route(s) of a non-terminal instance by the
// LA-C3 rule (completeCommittedRoutes) and nothing else. It never gathers dispatch and never runs
// RC-1: a waiting instance has no dispatch to settle, and RC-1 would re-enter or settle its wait.
// A log that already holds a workflow-terminal event is left alone (the terminal fence).
func (e *Engine) completeAwaitingRoutes(ctx context.Context, iid core.InstanceID) error {
	cur, found, err := e.getInstance(ctx, iid)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	evs, err := e.storage.ReadEvents(ctx, iid, 0)
	if err != nil {
		return fmt.Errorf("engine: route recovery read events %s: %w", iid, err)
	}
	if hasWorkflowTerminal(evs, -1) {
		return nil
	}
	cands, err := logFailedCandidates(evs)
	if err != nil {
		return err
	}
	if len(cands) == 0 {
		return nil
	}
	dv, err := e.defViewFor(ctx, cur)
	if err != nil {
		return err
	}
	flagged, err := e.cancellationRequested(ctx, iid)
	if err != nil {
		return err
	}
	return e.completeCommittedRoutes(ctx, dv, cur, cands, false, flagged)
}

// routeRecoveryGate reports whether a signal delivery or a timeout for iid must NOT proceed in this
// tick because a committed route (3) awaits its terminal event (LA-C6 R3-M1). site names the caller
// for the log. A refusal on a decision is recorded in e.routeRecovery so completeRouteRecovery
// finishes it. A read error refuses WITHOUT recording: the next tick's scan retries the check, and
// the signal stays undelivered in the inbox (internal/signal Waits.Resolve: ok=false leaves the
// signal for a later scan).
func (e *Engine) routeRecoveryGate(ctx context.Context, iid core.InstanceID, site string) bool {
	stepID, awaiting, err := e.committedRouteAwaitingTerminal(ctx, iid)
	if err != nil {
		e.log().Error("LA-C6 route gate: committed-route check failed; not proceeding this tick",
			"site", site, "instance_id", string(iid), "error", err.Error())
		return true
	}
	if !awaiting {
		return false
	}
	e.setRouteRecovery(iid, true)
	e.log().Info("LA-C6 route gate: committed route (3) awaits its terminal event; not proceeding this tick",
		"site", site, "instance_id", string(iid), "step_id", stepID)
	return true
}

// setRouteRecovery adds (on) or removes iid from the engine's routeRecovery set under e.mu.
func (e *Engine) setRouteRecovery(iid core.InstanceID, on bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if on {
		e.routeRecovery[iid] = true
		return
	}
	delete(e.routeRecovery, iid)
}

// routeRecoverySnapshot returns the recorded instances in a deterministic (sorted) order.
func (e *Engine) routeRecoverySnapshot() []core.InstanceID {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]core.InstanceID, 0, len(e.routeRecovery))
	for iid := range e.routeRecovery {
		out = append(out, iid)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// completeRouteRecovery is the LA-C6 R3-M1 tick stage. It runs after SCAN_RUNNABLE and before
// SIGNAL_SCAN, and only when no stop is requested. Its candidates are the instances recorded in
// e.routeRecovery and each instance in errored (processInstance failed in this tick) whose status,
// read afresh, is not running: processInstance may have parked the instance in `waiting` (enterWait)
// before it failed. For each candidate it runs completeAwaitingRoutes, then keeps the record while a
// committed route (3) still awaits its terminal event (or while its check failed), and drops it
// once a workflow-terminal event is durable. Every read error in this stage (the instance read here,
// and each read or check of completeAwaitingRoutes and the awaiting check) keeps or puts the record
// (LA-C7 HIGH-3), so the next tick retries it.
//
// LA-C6 R3-M1: completes an already-committed route (3); not the LA-D1 recovery stage (design
// pending founder approval). Errors are returned for tick to join; the instance is retried next tick.
func (e *Engine) completeRouteRecovery(ctx context.Context, errored []core.InstanceID) []error {
	var errs []error
	todo := make(map[core.InstanceID]bool)
	for _, iid := range e.routeRecoverySnapshot() {
		todo[iid] = true
	}
	for _, iid := range errored {
		if todo[iid] {
			continue
		}
		inst, found, err := e.getInstance(ctx, iid)
		if err != nil {
			// LA-C7 HIGH-3: the status is unknown, and a waiting instance is not scanned again, so the
			// record is what makes the next tick retry it (an unrecorded iid would never be retried).
			errs = append(errs, fmt.Errorf("engine: route recovery read instance %s: %w", iid, err))
			e.log().Error("LA-C7 route recovery: reading the instance failed; retried next tick",
				"instance_id", string(iid), "error", err.Error())
			e.setRouteRecovery(iid, true)
			continue
		}
		if !found || inst.Status == core.InstanceStatusRunning {
			continue
		}
		todo[iid] = true
	}
	ids := make([]core.InstanceID, 0, len(todo))
	for iid := range todo {
		ids = append(ids, iid)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, iid := range ids {
		if err := e.completeAwaitingRoutes(ctx, iid); err != nil {
			errs = append(errs, fmt.Errorf("engine: route recovery %s: %w", iid, err))
			e.log().Error("LA-C6 route recovery: completing the committed route failed; retried next tick",
				"instance_id", string(iid), "error", err.Error())
			e.setRouteRecovery(iid, true)
			continue
		}
		_, awaiting, err := e.committedRouteAwaitingTerminal(ctx, iid)
		if err != nil {
			errs = append(errs, fmt.Errorf("engine: route recovery check %s: %w", iid, err))
			e.setRouteRecovery(iid, true)
			continue
		}
		e.setRouteRecovery(iid, awaiting)
	}
	return errs
}
