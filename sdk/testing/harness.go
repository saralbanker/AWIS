// Package awistesting provides WorkflowTestHarness: a test helper that drives
// the real AWIS engine with deterministic clocks and IDs (PRD FR-SDK-06;
// Blueprint §27). Import path: github.com/awis/awis/sdk/testing.
package awistesting

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	sdk "github.com/awis/awis/sdk"
)

// RunResult is the return value of Harness.Run.
type RunResult struct {
	// InstanceID is the id of the submitted workflow instance.
	InstanceID core.InstanceID
}

// Option configures a Harness at construction time.
type Option func(*harnessConfig)

// harnessConfig collects options before the harness is built.
type harnessConfig struct {
	intelligence core.IntelligencePort
	handlers     []core.StepHandler
	clock        func() time.Time
	newID        func() string
}

// WithMockIntelligence routes all intelligence steps to the supplied port.
// Use NewMockIntelligence to create a fixture-driven mock.
func WithMockIntelligence(m core.IntelligencePort) Option {
	return func(c *harnessConfig) { c.intelligence = m }
}

// WithStepHandler registers a native step handler with the harness runtime.
func WithStepHandler(h core.StepHandler) Option {
	return func(c *harnessConfig) { c.handlers = append(c.handlers, h) }
}

// WithClock overrides the deterministic default clock with fn.
func WithClock(fn func() time.Time) Option {
	return func(c *harnessConfig) { c.clock = fn }
}

// WithIDSource overrides the deterministic default ID source with fn.
func WithIDSource(fn func() string) Option {
	return func(c *harnessConfig) { c.newID = fn }
}

// Harness is a test fixture that drives the real AWIS engine with deterministic
// sources. It never spawns goroutines that outlive the test and never sleeps on
// a poll interval — tests advance execution manually via Tick (Blueprint §27).
// Construct via NewHarness; never copy.
type Harness struct {
	t   *testing.T
	rt  *sdk.Runtime
	ctx context.Context
}

// NewHarness builds a Harness backed by an in-memory SQLite storage and the
// DeterministicMode preset. Options may override individual settings (clock,
// id source, handlers, intelligence port). The returned harness is ready for
// immediate use; t.Fatal is called if construction fails.
func NewHarness(t *testing.T, opts ...Option) *Harness {
	t.Helper()

	cfg := &harnessConfig{}
	for _, o := range opts {
		o(cfg)
	}

	store, err := sdk.SQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("awistesting.NewHarness: SQLiteStorage: %v", err)
	}

	detCfg := sdk.DeterministicMode()
	detCfg.Namespace = "test"
	detCfg.Storage = store
	if cfg.intelligence != nil {
		detCfg.Intelligence = cfg.intelligence
	}
	if cfg.clock != nil {
		detCfg.Clock = cfg.clock
	}
	if cfg.newID != nil {
		detCfg.NewID = cfg.newID
	}

	rt, err := sdk.NewRuntime(detCfg)
	if err != nil {
		t.Fatalf("awistesting.NewHarness: NewRuntime: %v", err)
	}

	for _, h := range cfg.handlers {
		if err := rt.RegisterHandler(h); err != nil {
			t.Fatalf("awistesting.NewHarness: RegisterHandler(%q): %v", h.ID(), err)
		}
	}

	return &Harness{
		t:   t,
		rt:  rt,
		ctx: context.Background(),
	}
}

// isTerminal reports whether status is a terminal lifecycle status.
func isTerminal(s core.InstanceStatus) bool {
	switch s {
	case core.InstanceStatusCompleted,
		core.InstanceStatusFailed,
		core.InstanceStatusCancelled,
		core.InstanceStatusCompensated,
		core.InstanceStatusCompensationFailed:
		return true
	}
	return false
}

// Run registers def with the harness runtime, submits a new instance with
// inputs, then ticks synchronously until the instance is terminal or waiting on
// a signal. It returns a RunResult carrying the InstanceID. Run calls t.Fatal
// on any error. The harness never re-implements engine semantics; all state
// transitions originate from Runtime/engine calls (IMP §27.M9 real-engine
// mandate).
func (h *Harness) Run(def *core.WorkflowDefinition, inputs map[string]any) (RunResult, error) {
	h.t.Helper()

	if err := h.rt.RegisterWorkflow(def); err != nil {
		return RunResult{}, fmt.Errorf("awistesting: Run: RegisterWorkflow: %w", err)
	}

	id, err := h.rt.Submit(h.ctx, def.ID, inputs)
	if err != nil {
		return RunResult{}, fmt.Errorf("awistesting: Run: Submit: %w", err)
	}

	// Tick until terminal or waiting.
	if err := h.tickUntilQuiet(id); err != nil {
		return RunResult{}, err
	}

	return RunResult{InstanceID: id}, nil
}

// tickUntilQuiet ticks the engine until the instance identified by id reaches a
// terminal state or enters the waiting state. It is the caller's responsibility
// to avoid infinite loops (the engine is deterministic; waiting is a stable
// halt state).
func (h *Harness) tickUntilQuiet(id core.InstanceID) error {
	h.t.Helper()
	for {
		if err := h.rt.Tick(h.ctx); err != nil {
			return fmt.Errorf("awistesting: Tick: %w", err)
		}
		st, err := h.rt.Status(h.ctx, id)
		if err != nil {
			return fmt.Errorf("awistesting: Status: %w", err)
		}
		if isTerminal(st.Status) || st.Status == core.InstanceStatusWaiting {
			return nil
		}
	}
}

// Signal delivers a named signal with payload to the waiting instance, then
// ticks synchronously until the instance is terminal or waiting again. Signal
// calls t.Fatal on any error.
func (h *Harness) Signal(id core.InstanceID, name string, payload map[string]any) {
	h.t.Helper()
	if err := h.rt.Signal(h.ctx, id, name, payload); err != nil {
		h.t.Fatalf("awistesting: Signal(%q, %q): %v", id, name, err)
	}
	if err := h.tickUntilQuiet(id); err != nil {
		st, _ := h.rt.Status(h.ctx, id)
		h.t.Fatalf("awistesting: Signal: tick-until-quiet failed: %v (instance status: %v)", err, st.Status)
	}
}

// Tick advances the engine by exactly one tick. It calls t.Fatal on error.
func (h *Harness) Tick() {
	h.t.Helper()
	if err := h.rt.Tick(h.ctx); err != nil {
		h.t.Fatalf("awistesting: Tick: %v", err)
	}
}

// WaitForCompletion ticks the engine until id reaches a terminal state or
// timeout elapses. It calls t.Fatal if the instance has not reached a terminal
// state within timeout, reporting the last observed status.
func (h *Harness) WaitForCompletion(id core.InstanceID, timeout time.Duration) {
	h.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := h.rt.Tick(h.ctx); err != nil {
			h.t.Fatalf("awistesting: WaitForCompletion: Tick: %v", err)
		}
		st, err := h.rt.Status(h.ctx, id)
		if err != nil {
			h.t.Fatalf("awistesting: WaitForCompletion: Status: %v", err)
		}
		if isTerminal(st.Status) {
			return
		}
	}
	st, _ := h.rt.Status(h.ctx, id)
	h.t.Fatalf("awistesting: WaitForCompletion: timed out after %v; last status: %v", timeout, st.Status)
}

// GetOutput reads the output keyed by key from the instance's variables. It
// returns nil if the key is absent. GetOutput calls t.Fatal if the status
// lookup fails.
func (h *Harness) GetOutput(id core.InstanceID, key string) any {
	h.t.Helper()
	st, err := h.rt.Status(h.ctx, id)
	if err != nil {
		h.t.Fatalf("awistesting: GetOutput: Status: %v", err)
	}
	return st.Outputs[key]
}
