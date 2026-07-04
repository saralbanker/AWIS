package engine

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/awis/awis/internal/core"
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
	// SCAN_TRIGGERABLE — STUB (C3, T5/F-2).
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

// scanTriggerable is the SCAN_TRIGGERABLE stage stub. C3 (T5, F-2) fills it with
// domain-event → workflow trigger matching and Ingest-driven submission.
func (e *Engine) scanTriggerable(_ context.Context) {}

// signalScan is the SIGNAL_SCAN stage stub. M07 fills it with signal delivery /
// waiting-status exit (EDR-007 §9 waiting-entry gap is M07's too).
func (e *Engine) signalScan(_ context.Context) {}

// dispatchItem is one claimed step awaiting dispatch within a tick.
type dispatchItem struct {
	step        core.Step
	sc          core.StepContext
	startedAt   time.Time
	assembleErr *core.StepError // set when input assembly failed (template_error)
}

// dispatchResult is the outcome of one step's Runner execution.
type dispatchResult struct {
	out     core.StepResult
	stepErr *core.StepError
}

// processInstance runs CLAIM → DISPATCH → SETTLE for one running instance.
func (e *Engine) processInstance(ctx context.Context, inst core.WorkflowInstance) error {
	dv, err := e.defViewFor(ctx, inst)
	if err != nil {
		return err
	}

	acts := activatableSteps(dv, inst)
	if len(acts) == 0 {
		return nil
	}

	// ── CLAIM + StepStarted (serial, deterministic order) ─────────────────────
	// Claims give at-most-once ownership (EDR-006); a lost claim is skipped.
	// Dispatch is bounded by MaxParallelSteps — excess activatable steps wait for
	// the next tick, where the stateless scan re-derives them.
	items := make([]dispatchItem, 0, e.cfg.MaxParallelSteps)
	for _, stepID := range acts {
		if len(items) >= e.cfg.MaxParallelSteps {
			break
		}
		won, err := e.claim(ctx, inst.InstanceID, stepID)
		if err != nil {
			return err
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
			return err
		}
		e.logger.Info("dispatch", "instance_id", string(inst.InstanceID), "step_id", stepID)
		items = append(items, dispatchItem{step: step, sc: sc, startedAt: startedAt})
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
			// C1 degenerate failure path: StepFailed{retrying:false} → WorkflowFailed.
			// C2 seam: retry / fallback / on_error transitions generalize this.
			if err := e.emitStepFailed(ctx, inst, it.step.ID, it.sc.Attempt, *r.stepErr, false); err != nil {
				return err
			}
			if err := e.emitWorkflowFailed(ctx, inst, it.step.ID, *r.stepErr); err != nil {
				return err
			}
			e.logger.Info("settle", "instance_id", string(inst.InstanceID), "step_id", it.step.ID, "outcome", "failed")
			return nil // workflow terminal.
		}
		if err := e.emitStepCompleted(ctx, inst, it.step.ID, it.sc.Attempt, r.out.Outputs, it.startedAt); err != nil {
			return err
		}
		e.logger.Info("settle", "instance_id", string(inst.InstanceID), "step_id", it.step.ID, "outcome", "completed")
	}

	// ── Completion check ──────────────────────────────────────────────────────
	// A workflow completes when a FinalStep has completed, no steps are running,
	// and no further steps are activatable (EDR-011 §2).
	inst2, found, err := e.getInstance(ctx, inst.InstanceID)
	if err != nil {
		return err
	}
	if !found || len(inst2.CurrentSteps) != 0 {
		return nil
	}
	if len(activatableSteps(dv, inst2)) != 0 {
		return nil
	}
	finals := completedFinalOutputs(dv, inst2)
	if len(finals) == 0 {
		return nil // stalled (join stall / no final reached) — author semantics, EDR-011 §1.
	}
	return e.emitWorkflowCompleted(ctx, inst2, finals)
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
		// later; EDR-011 §5). C2's T4 inserts retry/fallback in front of this.
		return dispatchResult{stepErr: &core.StepError{
			Code:    "runner_unavailable",
			Message: fmt.Sprintf("no runner registered for step type %q", step.Type),
		}}
	}

	out, serr := runner.Run(ctx, sc, step)
	if serr != nil {
		return dispatchResult{stepErr: serr}
	}
	if err := e.storage.CacheResult(ctx, key, out, idempotencyTTL); err != nil {
		e.logger.Warn("cache write failed",
			"instance_id", string(iid), "step_id", step.ID, "error", err.Error())
	}
	return dispatchResult{out: out}
}
