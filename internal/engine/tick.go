package engine

import (
	"context"
	"fmt"
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
func (e *Engine) Tick(ctx context.Context) error {
	// SCAN_TRIGGERABLE (T5/F-2) — trigger.go.
	e.scanTriggerable(ctx)

	// SCAN_RUNNABLE.
	instances, err := e.storage.ListInstances(ctx, core.InstanceFilter{Status: core.InstanceStatusRunning})
	if err != nil {
		return fmt.Errorf("engine: Tick list running instances: %w", err)
	}
	for _, inst := range instances {
		if err := e.processInstance(ctx, inst); err != nil {
			return err
		}
	}

	// SIGNAL_SCAN — STUB (M07).
	e.signalScan(ctx)
	return nil
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
	if len(items) == 0 {
		return nil
	}

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
		return nil // stalled (join stall / no final reached) — author semantics, EDR-011 §1.
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
			e.logger.Info("claim lost", "instance_id", string(inst.InstanceID), "step_id", stepID)
			continue
		}
		step := dv.steps[stepID]
		sc, aerr := e.assembleContext(inst, step, 1)
		if aerr != nil {
			items = append(items, dispatchItem{step: step, assembleErr: aerr})
			continue
		}
		startedAt, err := e.emitStepStarted(ctx, inst, stepID, sc.Attempt, sc.Inputs)
		if err != nil {
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

	// Prefer the ADJ-8 usage side-channel when the runner reports it.
	var (
		out   core.StepResult
		usage *core.Usage
		serr  *core.StepError
	)
	if ur, ok := runner.(UsageRunner); ok {
		out, usage, serr = ur.RunWithUsage(ctx, sc, step)
	} else {
		out, serr = runner.Run(ctx, sc, step)
	}
	if serr != nil {
		return dispatchResult{stepErr: serr}
	}
	if err := e.storage.CacheResult(ctx, key, out, idempotencyTTL); err != nil {
		e.logger.Warn("cache write failed",
			"instance_id", string(iid), "step_id", step.ID, "error", err.Error())
	}
	return dispatchResult{out: out, usage: usage}
}
