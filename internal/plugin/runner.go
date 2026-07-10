package plugin

// runner.go — PluginRunner implements engine.Runner for core.StepTypePlugin.
//
// Capability resolution rule: TDS-05 §6, IMPLEMENTATION_SPEC.md §3.
// Delegates to Manager.Call; maps outputs/errors.
//
// TRACEABILITY: T9 (PluginRunner + capability resolution rule).

import (
	"context"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/engine"
)

// compile-time assertion: *PluginRunner satisfies engine.Runner.
var _ engine.Runner = (*PluginRunner)(nil)

// PluginRunner dispatches type=plugin steps to the Manager (TDS-05 §6, §7).
// It resolves the handler field per the capability-resolution rule and delegates
// to Manager.Call.
type PluginRunner struct {
	mgr *Manager
}

// NewPluginRunner constructs a PluginRunner backed by mgr.
func NewPluginRunner(mgr *Manager) *PluginRunner {
	return &PluginRunner{mgr: mgr}
}

// Run implements engine.Runner. It resolves the handler, dispatches the call,
// and maps the result to core.StepResult / *core.StepError.
//
// The effective call timeout follows TDS-05 §5:
//   min(ctx deadline, step.Timeout if set, capability timeout_ms)
// The stepTimeout argument to Manager.Call carries the step-level timeout; the
// Manager computes the full effective minimum internally.
func (r *PluginRunner) Run(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.StepError) {
	// Apply step-level timeout to the context if set, so ctx.Deadline() is
	// available to Manager.computeEffectiveTimeout.
	if d := parsePluginTimeout(step.Timeout); d > 0 {
		deadline := time.Now().Add(d)
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
		if sc.Deadline.IsZero() || deadline.Before(sc.Deadline) {
			sc.Deadline = deadline
		}
	}

	outputs, stepErr := r.mgr.Call(ctx, string(step.Handler), sc.StepID, sc.Inputs, step.Timeout)
	if stepErr != nil {
		return core.StepResult{}, stepErr
	}
	if outputs == nil {
		outputs = map[string]any{}
	}
	return core.StepResult{Outputs: outputs}, nil
}
