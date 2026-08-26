// Package examples provides the built-in native step handlers that back the
// workflows `awis init` scaffolds. They exist so that the documented
// out-of-the-box path works end to end through the SHIPPED binary:
//
//	awis init  →  awis start  →  awis submit hello-world --input name=World
//
// Before this package existed, `awis start` constructed a runtime with an EMPTY
// native handler registry, so every scaffolded workflow failed on its first step
// with `handler_not_found` and the `native` step type — the primary step type —
// was unreachable through the CLI entirely (defect B-9/B-19).
//
// # Why the handlers live in the platform binary
//
// AWIS has two execution modes, and the distinction matters:
//
//   - EMBEDDED SDK MODE — an application imports `sdk`, calls
//     `Runtime.RegisterHandler` with its own `core.StepHandler` implementations,
//     and drives the runtime in its own process. Arbitrary native handlers work.
//     `examples/hello_workflow/main.go` is the worked example.
//   - CLI DAEMON MODE — `awis start` runs a pre-compiled binary. Go cannot load
//     user-written handlers into an already-compiled binary, so the only native
//     handlers reachable this way are the ones compiled in. These are those.
//
// A user's own native handlers therefore require embedded mode. `awis start`
// reports any referenced-but-unregistered handler at startup (see
// cmd/awis/start.go) so this limitation surfaces as a clear diagnostic instead
// of a mysterious per-step runtime failure.
//
// The handler IDs below are fixed by the scaffolded YAML in
// cmd/awis/scaffold/workflows/ and the byte-identical copies in
// examples/workflows/. Changing an ID here breaks those workflows.
//
// Boundary note (QG-4): `examples.*` is the platform's own demonstration
// namespace shipped as part of the product surface, not an application
// namespace. No downstream application name appears here.
package examples

import (
	"fmt"

	"github.com/awis/awis/internal/core"
)

// handlerFunc adapts a plain function to core.StepHandler. The interface needs
// an ID() and an Execute(); everything the scaffolded handlers do fits a single
// function, so one small adapter avoids six near-identical struct types.
type handlerFunc struct {
	id string
	fn func(core.StepContext) (core.StepResult, error)
}

func (h handlerFunc) ID() string { return h.id }

func (h handlerFunc) Execute(ctx core.StepContext) (core.StepResult, error) {
	return h.fn(ctx)
}

// Handlers returns the built-in native handlers for the scaffolded example
// workflows, keyed implicitly by their ID(). The set is deliberately small and
// side-effect-free apart from examples.hello.log, which prints — it exists to
// show a handler that does something observable.
func Handlers() []core.StepHandler {
	return []core.StepHandler{
		handlerFunc{id: "examples.hello.greet", fn: helloGreet},
		handlerFunc{id: "examples.hello.log", fn: helloLog},
		handlerFunc{id: "examples.signal.prepare", fn: signalPrepare},
		handlerFunc{id: "examples.signal.finalize", fn: signalFinalize},
		handlerFunc{id: "examples.intel.gather", fn: intelGather},
		handlerFunc{id: "examples.intel.review", fn: intelReview},
	}
}

// IDs returns just the handler identifiers, for startup diagnostics.
func IDs() []string {
	hs := Handlers()
	out := make([]string, 0, len(hs))
	for _, h := range hs {
		out = append(out, h.ID())
	}
	return out
}

// helloGreet backs hello-world's `greet` step. It produces {message}.
func helloGreet(ctx core.StepContext) (core.StepResult, error) {
	name, _ := ctx.Inputs["name"].(string)
	if name == "" {
		name = "World"
	}
	return core.StepResult{Outputs: map[string]any{
		"message": fmt.Sprintf("Hello, %s!", name),
	}}, nil
}

// helloLog backs hello-world's `log` step. It writes the message through the
// step-scoped logger rather than to stdout: `awis start`'s stdout carries the
// startup header and the --json startup event, and a handler must not corrupt
// either.
func helloLog(ctx core.StepContext) (core.StepResult, error) {
	msg, _ := ctx.Inputs["message"].(string)
	if ctx.Logger != nil {
		ctx.Logger.Info("hello-world", "message", msg)
	}
	return core.StepResult{Outputs: map[string]any{"logged": true}}, nil
}

// signalPrepare backs with-signal's `prepare` step, which runs before the
// workflow parks on the `approved` signal.
func signalPrepare(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{"ready": true}}, nil
}

// signalFinalize backs with-signal's `finalize` step, which runs after the
// signal is delivered. `approval` carries whatever the signal payload supplied.
func signalFinalize(ctx core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{
		"done":     true,
		"approval": ctx.Inputs["approval"],
	}}, nil
}

// intelGather backs with-intelligence's `gather-context` step, assembling the
// context string handed to the intelligence step that follows.
func intelGather(ctx core.StepContext) (core.StepResult, error) {
	query, _ := ctx.Inputs["query"].(string)
	return core.StepResult{Outputs: map[string]any{"context": query}}, nil
}

// intelReview backs with-intelligence's manual-entry fallback step, reached when
// the intelligence provider is unavailable.
func intelReview(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{"approved": true}}, nil
}
