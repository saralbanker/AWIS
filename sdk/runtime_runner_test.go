// Integration tests for WorkflowRunner methods on *Runtime (M08-C2 output 5;
// AWIS DoD §24.3).
package sdk

import (
	"context"
	"testing"

	"github.com/awis/awis/internal/core"
)

// greetHandler is a minimal native StepHandler for runner tests.
type greetHandler struct{}

func (h *greetHandler) ID() string { return "greet" }
func (h *greetHandler) Execute(ctx core.StepContext) (core.StepResult, error) {
	name, _ := ctx.Inputs["name"].(string)
	if name == "" {
		name = "world"
	}
	return core.StepResult{Outputs: map[string]any{"greeting": "hello " + name}}, nil
}

// buildHelloWorkflow returns a minimal single-step WorkflowDefinition.
func buildHelloWorkflow(ns string) *core.WorkflowDefinition {
	def, _ := NewWorkflowBuilder("hello", "1.0.0").
		SetNamespace(ns).
		AddStep(core.Step{
			ID:      "greet",
			Name:    "Greet",
			Type:    core.StepTypeNative,
			Handler: core.HandlerRef("greet"),
		}).
		SetInitialStep("greet").
		AddFinalStep("greet").
		Build()
	return def
}

// TestWorkflowRunner_SubmitStatusRoundTrip is the card integration test:
// NewRuntime → Register → Submit → Tick → Status.
func TestWorkflowRunner_SubmitStatusRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "test", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def := buildHelloWorkflow("test")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "hello", map[string]any{"name": "AWIS"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if iid == "" {
		t.Fatal("Submit returned empty InstanceID")
	}

	// Run one tick so the step executes.
	if err := rt.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	status, err := rt.Status(ctx, iid)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if status.InstanceID != iid {
		t.Errorf("Status.InstanceID = %q, want %q", status.InstanceID, iid)
	}
	if status.DefinitionID != "hello" {
		t.Errorf("Status.DefinitionID = %q, want %q", status.DefinitionID, "hello")
	}
}

// TestWorkflowRunner_Submit_NoWorkflow verifies Submit returns an error when no
// workflow matching definitionID is registered.
func TestWorkflowRunner_Submit_NoWorkflow(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "test", Storage: s})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	_, err = rt.Submit(ctx, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown definitionID, got nil")
	}
}

// TestWorkflowRunner_Cancel verifies Cancel delegates to the engine without error.
func TestWorkflowRunner_Cancel(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "test", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def := buildHelloWorkflow("test")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "hello", map[string]any{"name": "cancel-test"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Cancel before any tick — instance is pending or running, Cancel must not err.
	if err := rt.Cancel(ctx, iid, "test cancellation"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
}

// TestWorkflowRunner_List verifies List returns entries after submission.
func TestWorkflowRunner_List(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "listns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def, _ := NewWorkflowBuilder("listflow", "1.0.0").
		SetNamespace("listns").
		AddStep(core.Step{
			ID:      "greet",
			Name:    "Greet",
			Type:    core.StepTypeNative,
			Handler: core.HandlerRef("greet"),
		}).
		SetInitialStep("greet").
		AddFinalStep("greet").
		Build()
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	_, err = rt.Submit(ctx, "listflow", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	statuses, err := rt.List(ctx, core.InstanceFilter{Namespace: "listns"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(statuses) == 0 {
		t.Fatal("List returned 0 statuses, want ≥1")
	}
}
