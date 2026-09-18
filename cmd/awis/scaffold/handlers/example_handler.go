package handlers

// example_handler.go — worked examples of AWIS native step handlers.
//
// These are the handlers behind the workflows in ../workflows/. The `awis`
// binary already has its own compiled-in copies, so `awis start` runs the
// scaffolded workflows without any of this. This file is here for the step
// AFTER the tutorial: writing your OWN native handlers.
//
// A native handler is anything implementing sdk.StepHandler:
//
//	ID() string                                   // matches `handler:` in the YAML
//	Execute(sdk.StepContext) (sdk.StepResult, error)
//
// You register handlers with a Runtime you construct yourself, in your own
// program — EMBEDDED SDK MODE:
//
//	rt, _ := sdk.NewRuntime(sdk.Config{Namespace: "examples", Storage: store})
//	rt.RegisterHandler(HelloGreet{})
//	rt.RegisterWorkflow(def)
//	rt.Start(ctx)
//
// This is the only way your own native handlers can run. `awis start` is a
// pre-compiled binary and Go cannot load handlers into it at runtime, so a
// workflow that references a handler this binary does not contain will fail
// that step with `handler_not_found` — `awis start` warns about exactly that at
// startup. See examples/hello_workflow/main.go in the AWIS repo for a complete
// runnable program.
//
// Inputs arrive in ctx.Inputs, resolved from the step's `inputs:` templates in
// the YAML. A step with no `inputs:` mapping receives an empty map — the
// workflow's own inputs do NOT flow in implicitly.
//
// Return an error to fail the step; the engine turns it into a StepFailed event
// with code `handler_error` and applies the step's retry / fallback / on_error
// policy. Panics are contained and reported as `handler_panic`, but returning
// an error is the supported way to signal failure.

import (
	"fmt"

	"github.com/awis/awis/sdk"
)

// HelloGreet backs the hello-world workflow's `greet` step.
type HelloGreet struct{}

// ID returns the identifier used as `handler:` in workflows/hello-world.yaml.
func (HelloGreet) ID() string { return "examples.hello.greet" }

// Execute builds the greeting. `name` comes from the step's inputs mapping
// (`name: "{{workflow.inputs.name}}"`), so it is empty unless the instance was
// submitted with that input.
func (HelloGreet) Execute(ctx sdk.StepContext) (sdk.StepResult, error) {
	name, _ := ctx.Inputs["name"].(string)
	if name == "" {
		name = "World"
	}
	return sdk.StepResult{Outputs: map[string]any{
		"message": fmt.Sprintf("Hello, %s!", name),
	}}, nil
}

// HelloLog backs the hello-world workflow's `log` step.
type HelloLog struct{}

// ID returns the identifier used as `handler:` in workflows/hello-world.yaml.
func (HelloLog) ID() string { return "examples.hello.log" }

// Execute records the message through the step-scoped logger. Prefer
// ctx.Logger over fmt.Println: a handler that writes to stdout corrupts the
// `--json` output of whatever process is hosting the runtime.
func (HelloLog) Execute(ctx sdk.StepContext) (sdk.StepResult, error) {
	msg, _ := ctx.Inputs["message"].(string)
	if ctx.Logger != nil {
		ctx.Logger.Info("hello-world", "message", msg)
	}
	return sdk.StepResult{Outputs: map[string]any{"logged": true}}, nil
}

// SignalPrepare backs the with-signal workflow's `prepare` step, which runs
// before the workflow parks waiting for the `approved` signal.
type SignalPrepare struct{}

// ID returns the identifier used as `handler:` in workflows/with-signal.yaml.
func (SignalPrepare) ID() string { return "examples.signal.prepare" }

// Execute marks the workflow ready to wait for approval.
func (SignalPrepare) Execute(sdk.StepContext) (sdk.StepResult, error) {
	return sdk.StepResult{Outputs: map[string]any{"ready": true}}, nil
}

// SignalFinalize backs the with-signal workflow's `finalize` step, which runs
// once the awaited signal has been delivered.
type SignalFinalize struct{}

// ID returns the identifier used as `handler:` in workflows/with-signal.yaml.
func (SignalFinalize) ID() string { return "examples.signal.finalize" }

// Execute completes the workflow. `approval` carries whatever the signal
// payload supplied — deliver it with `awis signal <instance-id> approved`.
func (SignalFinalize) Execute(ctx sdk.StepContext) (sdk.StepResult, error) {
	return sdk.StepResult{Outputs: map[string]any{
		"done":     true,
		"approval": ctx.Inputs["approval"],
	}}, nil
}

// IntelGather backs the with-intelligence workflow's `gather-context` step,
// assembling the context string handed to the intelligence step after it.
type IntelGather struct{}

// ID returns the identifier used as `handler:` in workflows/with-intelligence.yaml.
func (IntelGather) ID() string { return "examples.intel.gather" }

// Execute passes the query through as the assembled context.
func (IntelGather) Execute(ctx sdk.StepContext) (sdk.StepResult, error) {
	query, _ := ctx.Inputs["query"].(string)
	return sdk.StepResult{Outputs: map[string]any{"context": query}}, nil
}

// IntelReview backs the with-intelligence workflow's manual-entry fallback
// step, reached when the intelligence provider is unavailable.
type IntelReview struct{}

// ID returns the identifier used as `handler:` in workflows/with-intelligence.yaml.
func (IntelReview) ID() string { return "examples.intel.review" }

// Execute stands in for a human review decision.
func (IntelReview) Execute(sdk.StepContext) (sdk.StepResult, error) {
	return sdk.StepResult{Outputs: map[string]any{"approved": true}}, nil
}

// All returns every handler in this file, ready to hand to RegisterHandler:
//
//	for _, h := range handlers.All() {
//	    if err := rt.RegisterHandler(h); err != nil { log.Fatal(err) }
//	}
func All() []sdk.StepHandler {
	return []sdk.StepHandler{
		HelloGreet{}, HelloLog{},
		SignalPrepare{}, SignalFinalize{},
		IntelGather{}, IntelReview{},
	}
}
