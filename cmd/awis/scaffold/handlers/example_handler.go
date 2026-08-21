package handlers

// example_handler.go — scaffold handler stubs for AWIS example workflows.
//
// Register these handlers with your AWIS runtime before starting the engine:
//
//	rt.RegisterHandler("examples.hello.greet", handlers.HelloGreet)
//	rt.RegisterHandler("examples.hello.log",   handlers.HelloLog)
//
// See README_AWIS.md for usage and next steps.

import (
	"context"
	"fmt"
)

// HelloGreet is the handler for the hello-world workflow "greet" step.
func HelloGreet(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	name, _ := inputs["name"].(string)
	if name == "" {
		name = "World"
	}
	return map[string]any{
		"message": fmt.Sprintf("Hello, %s!", name),
	}, nil
}

// HelloLog is the handler for the hello-world workflow "log" step.
func HelloLog(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	msg, _ := inputs["message"].(string)
	fmt.Println(msg)
	return map[string]any{"logged": true}, nil
}

// SignalPrepare is the handler for the with-signal workflow "prepare" step.
func SignalPrepare(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return map[string]any{"ready": true}, nil
}

// SignalFinalize is the handler for the with-signal workflow "finalize" step.
func SignalFinalize(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return map[string]any{"done": true}, nil
}

// IntelGather is the handler for the with-intelligence workflow "gather-context" step.
func IntelGather(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	query, _ := inputs["query"].(string)
	return map[string]any{"context": query}, nil
}

// IntelReview is the handler for the with-intelligence workflow "review" step.
func IntelReview(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return map[string]any{"approved": true}, nil
}
