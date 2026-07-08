package engine

// dispatchOne tests: unregistered step type ⇒ runner_unavailable; idempotency
// cache hit skips a second execution (Blueprint §5 L2 / §20).

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// countingRunner is an engine.Runner that counts executions and returns a fixed
// result, so a cache hit is observable as a non-incrementing counter.
type countingRunner struct {
	calls int
	out   core.StepResult
}

func (r *countingRunner) Run(_ context.Context, _ core.StepContext, _ core.Step) (core.StepResult, *core.StepError) {
	r.calls++
	return r.out, nil
}

func TestDispatchOne_UnregisteredStepType(t *testing.T) {
	s := openStorage(t)
	// Empty runner map ⇒ no runner for the step's type.
	e := New(s, map[core.StepType]Runner{}, Config{Clock: func() time.Time { return storageBase }}, discardLogger())

	step := core.Step{ID: "sig", Type: core.StepTypeSignal, Handler: "x"}
	res := e.dispatchOne(context.Background(), "inst-1", step, core.StepContext{Attempt: 1})
	if res.stepErr == nil {
		t.Fatalf("expected a StepError for an unregistered step type")
	}
	if res.stepErr.Code != "runner_unavailable" {
		t.Fatalf("code = %q, want runner_unavailable", res.stepErr.Code)
	}
}

func TestDispatchOne_IdempotencyCacheHit(t *testing.T) {
	s := openStorage(t)
	runner := &countingRunner{out: core.StepResult{Outputs: map[string]any{"r": "one"}}}
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: runner},
		Config{Clock: func() time.Time { return storageBase }}, discardLogger())

	ctx := context.Background()
	step := core.Step{ID: "step-a", Type: core.StepTypeNative, Handler: "h"}
	sc := core.StepContext{InstanceID: "inst-1", StepID: "step-a", Attempt: 1}

	// First dispatch: executes and caches.
	r1 := e.dispatchOne(ctx, "inst-1", step, sc)
	if r1.stepErr != nil {
		t.Fatalf("first dispatch errored: %+v", r1.stepErr)
	}
	if runner.calls != 1 {
		t.Fatalf("after first dispatch calls = %d, want 1", runner.calls)
	}

	// Second dispatch with the same (instance, step, attempt) key: cache hit ⇒
	// no re-execution; identical result.
	r2 := e.dispatchOne(ctx, "inst-1", step, sc)
	if r2.stepErr != nil {
		t.Fatalf("second dispatch errored: %+v", r2.stepErr)
	}
	if runner.calls != 1 {
		t.Fatalf("after cached second dispatch calls = %d, want 1 (no re-exec)", runner.calls)
	}
	if !reflect.DeepEqual(r2.out.Outputs, map[string]any{"r": "one"}) {
		t.Fatalf("cached outputs = %#v, want {r:one}", r2.out.Outputs)
	}
}
