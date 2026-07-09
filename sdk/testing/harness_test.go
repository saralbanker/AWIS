// Tests for WorkflowTestHarness (M09-C2; PRD FR-SDK-06; Blueprint §27).
// All tests use the real engine — no engine semantics are re-implemented here.
package awistesting

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/awis/awis/sdk"

	"github.com/awis/awis/internal/core"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// linearDef returns a minimal single-native-step workflow definition.
func linearDef(id string, handler string) *core.WorkflowDefinition {
	def, err := sdk.NewWorkflowBuilder(id, "1.0.0").
		SetNamespace("test").
		AddStep(core.Step{
			ID:      "s1",
			Name:    "s1",
			Type:    core.StepTypeNative,
			Handler: core.HandlerRef(handler),
		}).
		SetInitialStep("s1").
		AddFinalStep("s1").
		Build()
	if err != nil {
		panic("linearDef: " + err.Error())
	}
	return def
}

// waitDef returns a workflow that waits for signal "go" then completes.
func waitDef(id string) *core.WorkflowDefinition {
	def, err := sdk.NewWorkflowBuilder(id, "1.0.0").
		SetNamespace("test").
		AddStep(core.Step{
			ID:         "w",
			Name:       "wait",
			Type:       core.StepTypeSignal,
			WaitSignal: &core.WaitConfig{SignalName: "go", TimeoutAction: "fail"},
		}).
		SetInitialStep("w").
		AddFinalStep("w").
		Build()
	if err != nil {
		panic("waitDef: " + err.Error())
	}
	return def
}

// noopH is a StepHandler that succeeds immediately with no outputs.
type noopH struct{}

func (h *noopH) ID() string { return "noop" }
func (h *noopH) Execute(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{}}, nil
}

// outputH writes a key/value into outputs.
type outputH struct {
	key string
	val any
}

func (h *outputH) ID() string { return "out" }
func (h *outputH) Execute(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{h.key: h.val}}, nil
}

// ── NewHarness + Run on a minimal linear def → terminal ──────────────────────

func TestHarness_RunLinear_Terminal(t *testing.T) {
	h := NewHarness(t, WithStepHandler(&noopH{}))
	def := linearDef("linflow", "noop")

	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.InstanceID == "" {
		t.Fatal("RunResult.InstanceID is empty")
	}

	st, sErr := h.rt.Status(h.ctx, res.InstanceID)
	if sErr != nil {
		t.Fatalf("Status: %v", sErr)
	}
	if !isTerminal(st.Status) {
		t.Fatalf("expected terminal status, got %q", st.Status)
	}
}

// ── Run on WAIT def → returns while waiting; Signal → WaitForCompletion ──────

func TestHarness_RunWait_SignalCompletion(t *testing.T) {
	h := NewHarness(t)
	def := waitDef("waitflow")

	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// After Run the instance should be waiting (not yet terminal).
	st, sErr := h.rt.Status(h.ctx, res.InstanceID)
	if sErr != nil {
		t.Fatalf("Status after Run: %v", sErr)
	}
	if st.Status != core.InstanceStatusWaiting {
		t.Fatalf("expected waiting after Run on wait-def, got %q", st.Status)
	}

	// Deliver the signal; harness ticks until terminal-or-waiting.
	h.Signal(res.InstanceID, "go", nil)

	// WaitForCompletion should return immediately since we're already terminal.
	h.WaitForCompletion(res.InstanceID, 5*time.Second)

	st2, sErr2 := h.rt.Status(h.ctx, res.InstanceID)
	if sErr2 != nil {
		t.Fatalf("Status after WaitForCompletion: %v", sErr2)
	}
	if !isTerminal(st2.Status) {
		t.Fatalf("expected terminal after signal+WaitForCompletion, got %q", st2.Status)
	}
}

// ── GetOutput returns a handler-written output ────────────────────────────────

func TestHarness_GetOutput(t *testing.T) {
	handler := &outputH{key: "answer", val: "42"}
	h := NewHarness(t, WithStepHandler(handler))
	def := linearDef("outflow", "out")

	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Step outputs are stored in Variables under the step ID ("s1").
	raw := h.GetOutput(res.InstanceID, "s1")
	m, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("GetOutput(s1) type = %T, want map[string]any", raw)
	}
	if m["answer"] != "42" {
		t.Fatalf("GetOutput(s1)[answer] = %v, want 42", m["answer"])
	}
}

// ── Tick advances exactly one tick ───────────────────────────────────────────

func TestHarness_TickAdvancesOneStep(t *testing.T) {
	// A wait workflow parks after one tick; we verify the harness Tick exposes
	// the granular "one tick" control correctly by ticking manually.
	h := NewHarness(t)
	def := waitDef("tickflow")

	if err := h.rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	id, err := h.rt.Submit(h.ctx, def.ID, nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Before any tick the instance is pending or running but not waiting yet.
	h.Tick()

	st, sErr := h.rt.Status(h.ctx, id)
	if sErr != nil {
		t.Fatalf("Status: %v", sErr)
	}
	// After one tick the instance may be running or waiting (signal step entered).
	if st.Status != core.InstanceStatusRunning && st.Status != core.InstanceStatusWaiting {
		t.Fatalf("after Tick expected running or waiting, got %q", st.Status)
	}
}

// ── Determinism: same def+inputs → identical InstanceIDs ─────────────────────

func TestHarness_Determinism_IdenticalInstanceIDs(t *testing.T) {
	// Two fresh harnesses built from independent DeterministicMode presets must
	// produce the same InstanceID sequence (spec §1d; each call to
	// DeterministicMode returns an independent counter starting from 1).
	h1 := NewHarness(t, WithStepHandler(&noopH{}))
	h2 := NewHarness(t, WithStepHandler(&noopH{}))

	def1 := linearDef("detflow-a", "noop")
	def2 := linearDef("detflow-a", "noop")

	r1, err1 := h1.Run(def1, nil)
	if err1 != nil {
		t.Fatalf("h1.Run: %v", err1)
	}
	r2, err2 := h2.Run(def2, nil)
	if err2 != nil {
		t.Fatalf("h2.Run: %v", err2)
	}

	if r1.InstanceID != r2.InstanceID {
		t.Fatalf("determinism violated: h1 InstanceID=%q, h2 InstanceID=%q", r1.InstanceID, r2.InstanceID)
	}
}

// ── WithClock and WithIDSource overrides are accepted ─────────────────────────

func TestHarness_WithClockAndIDSource(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clockCalls := 0
	myClock := func() time.Time {
		clockCalls++
		return fixed
	}
	// The ID source must produce unique values: the engine uses it for both
	// instance IDs and event IDs, so returning the same string every time
	// causes a UNIQUE constraint violation on execution_events.event_id.
	idN := 0
	myID := func() string {
		idN++
		return fmt.Sprintf("cust-%d", idN)
	}

	h := NewHarness(t,
		WithStepHandler(&noopH{}),
		WithClock(myClock),
		WithIDSource(myID),
	)
	def := linearDef("customflow", "noop")

	res, err := h.Run(def, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The first call to the ID source is used as the instance ID.
	if string(res.InstanceID) != "cust-1" {
		t.Fatalf("InstanceID = %q, want \"cust-1\"", res.InstanceID)
	}
	// The clock was invoked at least once.
	if clockCalls == 0 {
		t.Fatal("WithClock: clock was never called")
	}
	// ID source was called more than once (instance + events).
	if idN <= 1 {
		t.Fatalf("WithIDSource: id source called %d times, want > 1", idN)
	}
}
