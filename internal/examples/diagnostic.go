package examples

// diagnostic.go — built-in handlers that make the FAILURE semantics reachable
// through the shipped binary.
//
// # Why these exist
//
// AWIS has two execution modes (see handlers.go). In EMBEDDED SDK MODE a user
// registers their own handlers, so any behaviour is reachable. In CLI DAEMON
// MODE `awis start` is a pre-compiled binary and can only dispatch handlers
// compiled into it — and before this file, every compiled-in handler always
// succeeded.
//
// That had two consequences, and the second is the serious one:
//
//  1. A user could not smoke-test their own workflow's recovery topology
//     (retry / fallback / on_error / compensation) without first writing Go.
//     The routing is a property of the DEFINITION, not of the handler, so
//     being forced to write a handler just to see whether an on_error edge
//     fires is a real gap.
//  2. The binary-level integration tier could not exercise a single failure
//     path. Retry, fallback, on_error, terminal failure, handler panic
//     containment and step timeout were reachable only through Go unit tests
//     that call the engine directly — which is exactly the class of evidence
//     the hardening program was told not to trust on its own.
//
// So these are deliberately-failing handlers. They are diagnostics, not
// examples: no workflow that `awis init` scaffolds references them, and they
// are namespaced `examples.diagnostic.*` to keep that distinction legible.
// Every one is DETERMINISTIC and STATELESS — `flaky` decides from
// ctx.Attempt, which the engine derives from the durable EventLog, so its
// behaviour is identical before and after a restart. None of them holds
// process-local state that a restart would silently reset.

import (
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// diagnosticHandlers returns the built-in diagnostic handlers. Kept separate
// from the scaffold-backing set in handlers.go so the two purposes stay
// distinct; Handlers() concatenates them.
func diagnosticHandlers() []core.StepHandler {
	return []core.StepHandler{
		handlerFunc{id: "examples.diagnostic.succeed", fn: diagSucceed},
		handlerFunc{id: "examples.diagnostic.fail", fn: diagFail},
		handlerFunc{id: "examples.diagnostic.flaky", fn: diagFlaky},
		handlerFunc{id: "examples.diagnostic.panic", fn: diagPanic},
		handlerFunc{id: "examples.diagnostic.slow", fn: diagSlow},
	}
}

// diagSucceed always succeeds. It is the counterpart target for a fallback:
// or on_error: edge, so a recovery route can be proven to have been taken:
// its `recovered: true` output appears in the instance's Variables only if
// the step actually ran.
func diagSucceed(ctx core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{
		"recovered": true,
		"step_id":   ctx.StepID,
		"attempt":   ctx.Attempt,
	}}, nil
}

// diagFail always fails. The engine turns the returned error into a
// StepFailed event and applies the step's retry / fallback / on_error policy,
// so this is the handler to point a step at when what you are testing is the
// ROUTING rather than the work.
//
// Optional input `message` overrides the failure text.
func diagFail(ctx core.StepContext) (core.StepResult, error) {
	msg, _ := ctx.Inputs["message"].(string)
	if msg == "" {
		msg = "deliberate failure from examples.diagnostic.fail"
	}
	return core.StepResult{}, fmt.Errorf("%s (step %s, attempt %d)", msg, ctx.StepID, ctx.Attempt)
}

// diagFlaky fails every attempt before `succeed_on` and succeeds from that
// attempt onward — the handler for exercising a retry: policy end to end.
// `succeed_on` defaults to 2 (fail once, then succeed).
//
// The decision is a pure function of ctx.Attempt, which the engine derives
// from the durable EventLog rather than from process memory. That matters:
// a flaky handler backed by a counter in process state would silently reset
// on restart and quietly turn a restart-resumption test into a test of
// nothing.
func diagFlaky(ctx core.StepContext) (core.StepResult, error) {
	succeedOn := asInt(ctx.Inputs["succeed_on"], 2)
	if succeedOn < 1 {
		succeedOn = 1
	}
	if ctx.Attempt < succeedOn {
		return core.StepResult{}, fmt.Errorf(
			"examples.diagnostic.flaky: attempt %d of a handler that succeeds on attempt %d",
			ctx.Attempt, succeedOn)
	}
	return core.StepResult{Outputs: map[string]any{
		"attempt":    ctx.Attempt,
		"succeed_on": succeedOn,
	}}, nil
}

// diagPanic panics. The native runner recovers it into
// StepError{Code:"handler_panic"} and the engine's own backstop recovers a
// panic from a Runner into StepError{Code:"runner_panic"}; this handler
// exercises the first of those through the shipped binary, proving the
// process SURVIVES a panicking handler rather than taking every other
// in-flight instance down with it.
func diagPanic(ctx core.StepContext) (core.StepResult, error) {
	panic(fmt.Sprintf("deliberate panic from examples.diagnostic.panic (step %s, attempt %d)",
		ctx.StepID, ctx.Attempt))
}

// maxDiagnosticSleep bounds diagSlow. A handler that outlives its step's
// deadline is ABANDONED by the native runner, not killed — Go cannot
// interrupt a goroutine — so an unbounded sleep here would leak a goroutine
// for that whole duration. The cap keeps a misconfigured workflow from
// parking a goroutine indefinitely while still being far longer than any
// timeout a timeout test would set.
const maxDiagnosticSleep = 60 * time.Second

// diagSlow sleeps for `duration_ms` (default 1000) before succeeding — the
// handler for exercising a step's timeout: policy, and for occupying a step
// long enough to observe `running` or to cancel it mid-flight.
//
// It cannot observe cancellation: core.StepContext carries a Deadline but no
// context.Context, so the native runner enforces the timeout by abandoning
// the handler goroutine. The sleep is capped accordingly.
func diagSlow(ctx core.StepContext) (core.StepResult, error) {
	d := clampSleep(time.Duration(asInt(ctx.Inputs["duration_ms"], 1000)) * time.Millisecond)
	time.Sleep(d)
	return core.StepResult{Outputs: map[string]any{"slept_ms": d.Milliseconds()}}, nil
}

// clampSleep bounds a requested sleep to [0, maxDiagnosticSleep]. Split out
// from diagSlow so the bound can be asserted without a test having to sleep
// for the full cap to observe it.
func clampSleep(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d > maxDiagnosticSleep {
		return maxDiagnosticSleep
	}
	return d
}

// asInt coerces a resolved template input to an int, falling back to def.
//
// Step inputs arrive as `any` after template resolution, and the concrete
// type depends on the path taken: a YAML literal decodes to int, a value that
// round-tripped through JSON (the EventLog payload, a --input flag) arrives
// as float64, and a template that interpolated into a string arrives as a
// string. Accepting all three keeps a diagnostic from behaving differently
// depending on how its input happened to reach it.
func asInt(v any, def int) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(n, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return def
}
