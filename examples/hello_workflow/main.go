// hello_workflow is a minimal AWIS example that demonstrates the sdk-only
// boundary (FR-SDK-09). It opens an in-memory SQLite storage, registers a
// native handler and a WorkflowBuilder-built workflow, submits one instance,
// runs a single tick, and prints the resulting status.
//
// compile: go build ./examples/...
// run:     go run ./examples/hello_workflow/
//
// Imports only github.com/awis/awis/sdk — no internal/ packages.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/awis/awis/sdk"
)

// greetHandler is a native step handler that echoes its inputs and adds a
// greeting message.
type greetHandler struct{}

func (h *greetHandler) ID() string { return "greet" }

func (h *greetHandler) Execute(ctx sdk.StepContext) (sdk.StepResult, error) {
	name, _ := ctx.Inputs["name"].(string)
	if name == "" {
		name = "world"
	}
	return sdk.StepResult{
		Outputs: map[string]any{
			"message": fmt.Sprintf("Hello, %s!", name),
		},
	}, nil
}

func main() {
	ctx := context.Background()

	// Open an in-memory SQLite storage via the sdk helper (Blueprint §12 L934).
	// The ":memory:" path is supported by modernc.org/sqlite.
	store, err := sdk.SQLiteStorage(":memory:")
	if err != nil {
		log.Fatalf("SQLiteStorage: %v", err)
	}

	// Build the runtime.
	rt, err := sdk.NewRuntime(sdk.Config{
		Namespace: "hello",
		Storage:   store,
		WorkerID:  "example-worker",
	})
	if err != nil {
		log.Fatalf("NewRuntime: %v", err)
	}

	// Register the native handler.
	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		log.Fatalf("RegisterHandler: %v", err)
	}

	// Build and register the workflow definition.
	def, err := sdk.NewWorkflowBuilder("hello-workflow", "1.0.0").
		SetNamespace("hello").
		SetName("Hello Workflow").
		SetDescription("A minimal greeting workflow.").
		AddStep(sdk.Step{
			ID:      "greet",
			Name:    "Greet",
			Type:    sdk.StepTypeNative,
			Handler: sdk.HandlerRef("greet"),
		}).
		SetInitialStep("greet").
		AddFinalStep("greet").
		Build()
	if err != nil {
		log.Fatalf("WorkflowBuilder.Build: %v", err)
	}

	if err := rt.RegisterWorkflow(def); err != nil {
		log.Fatalf("RegisterWorkflow: %v", err)
	}

	// Submit a new workflow instance.
	instanceID, err := rt.Submit(ctx, "hello-workflow", map[string]any{
		"name": "AWIS",
	})
	if err != nil {
		log.Fatalf("Submit: %v", err)
	}
	fmt.Printf("submitted instance: %s\n", instanceID)

	// Run one tick to activate and execute the first step.
	if err := rt.Tick(ctx); err != nil {
		log.Fatalf("Tick: %v", err)
	}

	// Read and print the resulting status.
	status, err := rt.Status(ctx, instanceID)
	if err != nil {
		log.Fatalf("Status: %v", err)
	}
	fmt.Printf("status:  %s\n", status.Status)
	fmt.Printf("outputs: %v\n", status.Outputs)
}
