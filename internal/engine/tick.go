package engine

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/signal"
)

// Run drives the pull loop at Config.TickInterval until ctx is cancelled. A tick
// error is logged and the loop continues (a transient storage error must not kill
// the engine); ctx cancellation returns ctx.Err().
func (e *Engine) Run(ctx context.Context) error {
	ticker := time.NewTicker(e.cfg.TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := e.Tick(ctx); err != nil {
				e.logger.Error("tick failed", "error", err.Error())
			}
		}
	}
}

// Tick runs one pass of the pull pipeline (Blueprint §8 stage order).
//
// D-17: a single instance's error must not prevent every other instance, or
// the two scan stages below the SCAN_RUNNABLE loop, from running. Each
// instance is processed in isolation — its error is logged immediately
// (instance_id + error, so the failure is diagnosable per instance, which is
// the entire point: this class of failure was previously undiagnosable
// because it silently starved everything after the first failing instance)
// and collected, but does NOT stop the loop. SIGNAL_SCAN and
// SIGNAL_TIMEOUT_SCAN then run unconditionally — even when ListInstances
// itself failed (so this tick saw zero instances) or every instance errored —
// because they are independent stages, not a continuation of SCAN_RUNNABLE.
//
// Tick still returns an error: the individual per-instance errors (each
// already logged) and any ListInstances error are joined with errors.Join and
// returned together, so a caller that wants the full picture (or wants to
// errors.Is/As against a specific cause) still can, while no single error
// aborts processing of anything else this tick. A nil return means every
// instance this tick, and the ListInstances call, succeeded.
func (e *Engine) Tick(ctx context.Context) error {
	// SCAN_TRIGGERABLE (T5/F-2) — trigger.go.
	e.scanTriggerable(ctx)

	// SCAN_RUNNABLE.
	var errs []error
	instances, err := e.storage.ListInstances(ctx, core.InstanceFilter{Status: core.InstanceStatusRunning})
	if err != nil {
		errs = append(errs, fmt.Errorf("engine: Tick list running instances: %w", err))
	} else {
		for _, inst := range instances {
			if err := e.processInstance(ctx, inst); err != nil {
				e.logger.Error("process instance failed; skipping for this tick",
					"instance_id", string(inst.InstanceID), "error", err.Error())
				errs = append(errs, fmt.Errorf("engine: Tick process instance %s: %w", inst.InstanceID, err))
			}
		}
	}

	// SIGNAL_SCAN — deliver undelivered inbox signals (M07-C2). Runs
	// regardless of any error above (D-17 point 2).
	e.signalScan(ctx)

	// SIGNAL_TIMEOUT_SCAN — expire timed-out wait_records (M07-C3r). Runs
	// regardless of any error above (D-17 point 2).
	e.signalTimeoutScan(ctx)

	return errors.Join(errs...)
}

// signalScan is the SIGNAL_SCAN stage: it drives the delivery scan (internal/
// signal) once per tick. A storage that does not support the signal tables
// no-ops the stage (graceful degradation, mirroring cancellationStore /
// triggerStore); a scan error is logged and never returned so a transient
// signal-store fault cannot wedge the tick.
func (e *Engine) signalScan(ctx context.Context) {
	store, ok := e.storage.(signal.Store)
	if !ok {
		return
	}
	sc := signal.NewScanner(store, e, e.now, e.newID, e.cfg.WorkerID, e.logger)
	if err := sc.Scan(ctx); err != nil {
		e.logger.Warn("signal scan failed", "error", err.Error())
	}
}

// dispatchItem is one claimed step awaiting dispatch within a tick.
type dispatchItem struct {
	step        core.Step
	sc          core.StepContext
	startedAt   time.Time
	assembleErr *core.StepError // set when input assembly failed (template_error)
}

// dispatchResult is the outcome of one step's Runner execution. usage is the
// optional ADJ-8 side-channel value (nil for native runners / cache hits).
type dispatchResult struct {
	out     core.StepResult
	usage   *core.Usage
	stepErr *core.StepError
}

// processInstance runs CLAIM → DISPATCH → SETTLE for one running instance,
// generalized (C2) with retry re-dispatch, failure routing, and the cancellation
// gate. The stages remain those C1 established; the failure branch and the
// activation sources are what C2 broadened.
func (e *Engine) processInstance(ctx context.Context, inst core.WorkflowInstance) error {
	dv, err := e.defViewFor(ctx, inst)
	if err != nil {
		return err
	}

	// engine-hardening Step 2a: reconstruct this instance's retry/pending/
	// terminally-failed bookkeeping from the EventLog the first time THIS
	// process touches it (hydrate.go); a no-op after the first tick.
	if err := e.hydrate(ctx, dv, inst); err != nil {
		return err
	}

	// Refresh instance after hydrate in case crash recovery updated status/current_steps.
	cur, found, err := e.getInstance(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if !found || isTerminalStatus(cur.Status) {
		return nil
	}
	inst = cur

	// ── Cancellation gate (top of tick) — Finalization B4 step 3 ──────────────
	// If cancellation was requested on a prior tick, NO new step activates and NO
	// retry re-dispatches; retry-waiting steps are terminally failed and the
	// instance finalizes when current_steps drains.
	flagged, err := e.cancellationRequested(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if flagged {
		return e.handleCancellation(ctx, dv, inst)
	}

	// ── Gather dispatch items ─────────────────────────────────────────────────
	items, err := e.gatherDispatch(ctx, dv, inst)
	if err != nil {
		return err
	}
	// semantics-bearing (M07-C3r OUTPUT 0): do NOT early-return when items=0.
	// SIGNAL_SCAN (or SIGNAL_TIMEOUT_SCAN) may have emitted StepCompleted for a
	// signal step in this same tick, leaving the instance with current_steps=[].
	// The completion check at the bottom of processInstance must run regardless of
	// whether this tick dispatched new work; it re-reads the projection, so it
	// safely handles the stall / join-wait case (current_steps≠[] → return nil).

	// ── DISPATCH (concurrent, bounded) ────────────────────────────────────────
	// Only handler execution is concurrent; every result index is written by a
	// single goroutine, and all event emission happens afterward, serially.
	results := make([]dispatchResult, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, e.cfg.MaxParallelSteps)
	for i := range items {
		it := items[i]
		if it.assembleErr != nil {
			results[i] = dispatchResult{stepErr: it.assembleErr}
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, di dispatchItem) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = e.dispatchOne(ctx, inst.InstanceID, di.step, di.sc)
		}(i, it)
	}
	wg.Wait()

	// ── SETTLE (serial, dispatch order) ───────────────────────────────────────
	for i := range items {
		it := items[i]
		r := results[i]
		if r.stepErr != nil {
			terminal, err := e.settleFailure(ctx, dv, inst, it.step, it.sc.Attempt, *r.stepErr)
			if err != nil {
				return err
			}
			if terminal {
				return nil // workflow reached a terminal state; stop settling.
			}
			continue
		}
		if err := e.settleSuccess(ctx, inst, it, r); err != nil {
			return err
		}
	}

	// ── Cancellation gate (post-settle) — Finalization B4 steps 2/3 ───────────
	// A handler may have requested cancellation DURING dispatch (the B4 fixture
	// cancels from inside Execute). Re-read the flag: if set, no next step
	// activates and the instance finalizes once current_steps drains.
	flagged2, err := e.cancellationRequested(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if flagged2 {
		cur, found, err := e.getInstance(ctx, inst.InstanceID)
		if err != nil {
			return err
		}
		if found {
			return e.handleCancellation(ctx, dv, cur)
		}
		return nil
	}

	// ── Completion check ──────────────────────────────────────────────────────
	// A workflow completes when a FinalStep has completed, no steps are running,
	// and no further steps are activatable — including failure-driven pending
	// activations (EDR-011 §2/§8).
	inst2, found, err := e.getInstance(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if !found || len(inst2.CurrentSteps) != 0 {
		return nil
	}
	if len(e.activatableFor(dv, inst2)) != 0 {
		return nil
	}
	finals := completedFinalOutputs(dv, inst2)
	if len(finals) == 0 {
		// SEC-08: no step is running, none is activatable, and no final step has
		// completed — every path the instance actually took has terminated
		// without ever reaching final_steps (e.g. an unconditional branch led to
		// a step with no outgoing transition, while the sibling branch that DID
		// reach a final step never fired). activatableFor is a stateless
		// function of persisted completed-step state (EDR-011 §1), so this is a
		// permanent fixed point, not a transient join-wait: a step awaiting a
		// signal keeps the instance in status "waiting" (excluded from this
		// running-instance scan), and a step with a pending retry stays in
		// CurrentSteps (project() only removes it on a non-retrying StepFailed),
		// so neither can reach this branch. Route to WorkflowFailed instead of
		// leaving the instance wedged in "running" forever.
		return e.emitWorkflowFailed(ctx, inst2, "", core.StepError{
			Code:    "stalled",
			Message: "workflow reached a dead end: no final step was completed and no further step is activatable",
		})
	}
	return e.emitWorkflowCompleted(ctx, inst2, finals)
}

// gatherDispatch builds this tick's dispatch items from two sources: newly
// activatable steps (join gate + failure-driven pending set), which CLAIM and
// emit StepStarted{attempt:1}; and due retries (steps already in current_steps
// under an existing claim), which re-emit StepStarted{attempt:n} WITHOUT
// re-claiming (retries do not re-claim, EDR-011 §3). Both are bounded by
// MaxParallelSteps; the sources are disjoint (retries are running, activatable
// excludes running).
func (e *Engine) gatherDispatch(ctx context.Context, dv *defView, inst core.WorkflowInstance) ([]dispatchItem, error) {
	items := make([]dispatchItem, 0, e.cfg.MaxParallelSteps)

	// (1) Newly activatable — CLAIM + StepStarted{attempt:1}.
	for _, stepID := range e.activatableFor(dv, inst) {
		if len(items) >= e.cfg.MaxParallelSteps {
			break
		}
		won, err := e.claim(ctx, inst.InstanceID, stepID)
		if err != nil {
			return nil, err
		}
		if !won {
			e.noteClaimLost(inst.InstanceID, stepID)
			continue
		}
		e.clearClaimLostStreak(inst.InstanceID, stepID)
		step := dv.steps[stepID]
		sc, aerr := e.assembleContext(inst, step, 1)
		if aerr != nil {
			items = append(items, dispatchItem{step: step, assembleErr: aerr})
			continue
		}
		startedAt, err := e.emitStepStarted(ctx, inst, stepID, sc.Attempt, sc.Inputs)
		if err != nil {
			// The claim above committed in ClaimStep's own transaction
			// (sqlite.go); if this emission fails, the claim must not be left
			// orphaned — activatableFor never consults step_claims, so the
			// step would stay activatable forever while the claim could never
			// be won again (PK conflict), wedging the instance permanently
			// (D-6). Best-effort compensating release: if it also fails, log
			// clearly and still return the ORIGINAL emission error (never mask
			// it with a release failure).
			if rs, ok := e.storage.(claimReleaseStore); ok {
				if rerr := rs.ReleaseStepClaim(ctx, inst.InstanceID, stepID); rerr != nil {
					e.logger.Error("failed to release orphaned step claim after StepStarted emission failure",
						"instance_id", string(inst.InstanceID), "step_id", stepID,
						"emit_error", err.Error(), "release_error", rerr.Error())
				}
			}
			return nil, err
		}
		e.clearPending(inst.InstanceID, stepID)

		// WAIT-step entry: a signal step activates through the existing step-entry
		// path (StepStarted above) but does NOT dispatch to a runner. It parks the
		// instance in `waiting` and records a wait_record; SIGNAL_SCAN resumes it
		// atomically (Finalization B3) when the awaited signal arrives.
		if step.Type == core.StepTypeSignal {
			if err := e.enterWait(ctx, inst, step); err != nil {
				return nil, err
			}
			continue
		}

		e.logger.Info("dispatch", "instance_id", string(inst.InstanceID), "step_id", stepID)
		items = append(items, dispatchItem{step: step, sc: sc, startedAt: startedAt})
	}

	// (2) Due retries — re-dispatch WITHOUT re-claiming (EDR-011 §3).
	for _, stepID := range e.dueRetries(inst.InstanceID, inst.CurrentSteps) {
		if len(items) >= e.cfg.MaxParallelSteps {
			break
		}
		attempt := e.retryAttempt(inst.InstanceID, stepID)
		step := dv.steps[stepID]
		sc, aerr := e.assembleContext(inst, step, attempt)
		if aerr != nil {
			items = append(items, dispatchItem{step: step, assembleErr: aerr})
			continue
		}
		startedAt, err := e.emitStepStarted(ctx, inst, stepID, sc.Attempt, sc.Inputs)
		if err != nil {
			return nil, err
		}
		e.logger.Info("dispatch retry", "instance_id", string(inst.InstanceID), "step_id", stepID, "attempt", attempt)
		items = append(items, dispatchItem{step: step, sc: sc, startedAt: startedAt})
	}

	return items, nil
}

// settleSuccess emits StepCompleted for a successful dispatch and clears any
// retry bookkeeping. Intelligence-type steps with a live usage side-channel
// carry the ADJ-8 {adapter, model, tokens_used} triple; every other step uses
// the plain four-field payload (usage keys absent).
func (e *Engine) settleSuccess(ctx context.Context, inst core.WorkflowInstance, it dispatchItem, r dispatchResult) error {
	e.clearRetry(inst.InstanceID, it.step.ID)
	if it.step.Type == core.StepTypeIntelligence && r.usage != nil {
		if err := e.emitStepCompletedUsage(ctx, inst, it.step.ID, it.sc.Attempt, r.out.Outputs, it.startedAt, *r.usage); err != nil {
			return err
		}
	} else {
		if err := e.emitStepCompleted(ctx, inst, it.step.ID, it.sc.Attempt, r.out.Outputs, it.startedAt); err != nil {
			return err
		}
	}
	e.logger.Info("settle", "instance_id", string(inst.InstanceID), "step_id", it.step.ID, "outcome", "completed")
	return nil
}

// claim wraps ClaimStep and keeps the engine's tracked version in lock-step with
// the claim's in-tx version bump (EDR-006).
func (e *Engine) claim(ctx context.Context, iid core.InstanceID, stepID string) (bool, error) {
	won, err := e.storage.ClaimStep(ctx, iid, stepID, e.cfg.WorkerID)
	if err != nil {
		return false, fmt.Errorf("engine: claim %s/%s: %w", iid, stepID, err)
	}
	if won {
		e.bumpVersion(iid)
	}
	return won, nil
}

// claimLostWedgeThreshold is the number of CONSECUTIVE lost-claim ticks for
// the same (instance, step) after which a lost claim stops being routine
// contention and starts being surfaced as the D-6 wedge signature (a claim
// that can never be won again because it was orphaned by a failed StepStarted
// emission with no compensating release — the case Part 1 of this fix now
// handles going forward, but a lease-bounded or pre-fix orphan can still
// exhibit this pattern). Three consecutive ticks (well under a second at the
// default 100ms tick interval) is enough to rule out a one-tick race with
// another dispatch while still surfacing genuinely stuck claims quickly.
const claimLostWedgeThreshold = 3

// noteClaimLost records one more consecutive claim-loss for (iid, stepID) and
// logs it. Below the wedge threshold this is routine, low-severity noise
// (Info); at the threshold, and every claimLostWedgeThreshold losses
// thereafter (so the signal is NOT re-logged every single tick once a wedge
// is suspected — non-noisy per D-6), it is logged at Error as the wedge
// signature so it cannot silently persist unnoticed the way the original
// defect did for a month.
func (e *Engine) noteClaimLost(iid core.InstanceID, stepID string) {
	key := retryKey{iid: iid, step: stepID}
	e.mu.Lock()
	e.claimLostStreak[key]++
	streak := e.claimLostStreak[key]
	e.mu.Unlock()

	if streak >= claimLostWedgeThreshold && streak%claimLostWedgeThreshold == 0 {
		e.logger.Error("step claim lost on repeated consecutive ticks; instance may be permanently wedged (D-6 orphan-claim signature)",
			"instance_id", string(iid), "step_id", stepID, "consecutive_losses", streak)
		return
	}
	e.logger.Info("claim lost", "instance_id", string(iid), "step_id", stepID)
}

// clearClaimLostStreak resets the consecutive-loss counter for (iid, stepID)
// once a claim is won again, so a later unrelated bout of contention starts
// counting from zero rather than inheriting a stale streak.
func (e *Engine) clearClaimLostStreak(iid core.InstanceID, stepID string) {
	key := retryKey{iid: iid, step: stepID}
	e.mu.Lock()
	delete(e.claimLostStreak, key)
	e.mu.Unlock()
}

// dispatchOne runs one step through the idempotency cache and its Runner. It is
// called from a dispatch goroutine; it performs no event emission and no engine
// state-map mutation (only storage cache reads/writes, which are concurrency-safe).
func (e *Engine) dispatchOne(ctx context.Context, iid core.InstanceID, step core.Step, sc core.StepContext) dispatchResult {
	// Idempotency key = instance_id/step_id/attempt (Blueprint §20; attempt=1 at C1).
	key := core.IdempotencyKey(string(iid) + "/" + step.ID + "/" + strconv.Itoa(sc.Attempt))
	if cached, ok, err := e.storage.GetCachedResult(ctx, key); err != nil {
		e.logger.Warn("cache read failed; treating as miss",
			"instance_id", string(iid), "step_id", step.ID, "error", err.Error())
	} else if ok {
		return dispatchResult{out: cached}
	}

	runner, ok := e.runners[step.Type]
	if !ok {
		// Unregistered StepType (signal=M07, subprocess=M11, plugin=M12 register
		// later; EDR-011 §5). Retry/fallback routing runs in front of this at settle.
		return dispatchResult{stepErr: &core.StepError{
			Code:    "runner_unavailable",
			Message: fmt.Sprintf("no runner registered for step type %q", step.Type),
		}}
	}

	// Prefer the ADJ-8 usage side-channel when the runner reports it. The
	// invocation is wrapped by runInvoke's panic backstop (engine-hardening
	// Step 2c): dispatchOne runs on a dispatch goroutine (processInstance's
	// DISPATCH stage, tick.go) with no recover above it in the call stack, so
	// an unrecovered Runner panic here would crash the whole process.
	out, usage, serr := e.runInvoke(ctx, runner, sc, step)
	if serr != nil {
		return dispatchResult{stepErr: serr}
	}
	if err := e.storage.CacheResult(ctx, key, out, idempotencyTTL); err != nil {
		e.logger.Warn("cache write failed",
			"instance_id", string(iid), "step_id", step.ID, "error", err.Error())
	}
	return dispatchResult{out: out, usage: usage}
}

// maxRunnerPanicStackBytes bounds the stack trace recorded in StepError.Details
// on a runner panic. StepError is carried inside the StepFailed payload, which
// is appended to the APPEND-ONLY EventLog and can never be rewritten (TDS-01
// §1); an unbounded goroutine stack would be persisted forever. Mirrors
// internal/runner/native's maxPanicStackBytes bound (native.go) — same
// technique, kept local so engine does not import a specific runner package.
const maxRunnerPanicStackBytes = 4096

// runInvoke calls runner.Run / RunWithUsage with a panic backstop: a panic
// inside the Runner is recovered and converted into a StepError{code:
// "runner_panic"} instead of propagating past dispatchOne's goroutine and
// crashing the process. internal/runner/native already recovers HANDLER
// panics into StepError{code:"handler_panic"} (Blueprint B-3) — that is a
// runner catching its own dispatched handler; this is the backstop for the
// runner itself (or any OTHER runner kind — intelligence, subprocess, plugin,
// or a future custom Runner) misbehaving, so the distinct code is kept so a
// caller can tell the two apart.
func (e *Engine) runInvoke(ctx context.Context, runner Runner, sc core.StepContext, step core.Step) (out core.StepResult, usage *core.Usage, serr *core.StepError) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			if len(stack) > maxRunnerPanicStackBytes {
				stack = append(stack[:maxRunnerPanicStackBytes:maxRunnerPanicStackBytes],
					[]byte("\n... stack truncated ...")...)
			}
			out = core.StepResult{}
			usage = nil
			serr = &core.StepError{
				Code:    "runner_panic",
				Message: fmt.Sprintf("step %q runner panicked: %v", step.ID, r),
				Details: map[string]any{"stack": string(stack)},
			}
		}
	}()
	if ur, ok := runner.(UsageRunner); ok {
		out, usage, serr = ur.RunWithUsage(ctx, sc, step)
		return out, usage, serr
	}
	out, serr = runner.Run(ctx, sc, step)
	return out, nil, serr
}
