//go:build integration

package integration

// Workflow fixtures used across this suite. Every fixture is parsed and
// validated by TestFixturesValidate (fixtures_validate_test.go) via
// 'awis workflow validate', so a fixture that stops parsing fails there with
// one clear reason instead of failing every dependent test for the wrong one.
//
// Field names follow internal/dsl (TDS-02 YAML field names verbatim) and
// internal/validate's structural rules (initial_step/final_steps reachable
// over transition + fallback edges, EDR-010). name/description/outputs are
// omitted where not required — neither dsl nor validate enforces them.

// linearNative is 2 native steps chained end-to-end, namespace default,
// using the scaffold handler ids (examples.hello.greet / examples.hello.log).
const linearNative = `schema_version: 1
id: linear-native
version: 1.0.0
namespace: default
triggers:
  - type: manual
steps:
  - id: greet
    type: native
    handler: examples.hello.greet
    inputs:
      name: "{{workflow.inputs.name}}"
  - id: log
    type: native
    handler: examples.hello.log
    inputs:
      message: "{{steps.greet.outputs.message}}"
initial_step: greet
transitions:
  - from: greet
    to: log
final_steps: [log]
`

// signalOnly is 2 type: signal steps chained (wait-a awaiting signal "go",
// wait-b awaiting signal "done"), namespace default — the only workflow
// shape that reliably parks an instance without native handlers. Shape
// copied verbatim from the reference waiter.yaml fixture.
const signalOnly = `schema_version: 1
id: waiter
version: 1.0.0
namespace: default
triggers:
  - type: manual
steps:
  - id: wait-a
    type: signal
    wait_signal:
      signal_name: go
      timeout: 72h
      timeout_action: fail
  - id: wait-b
    type: signal
    wait_signal:
      signal_name: done
      timeout: 72h
      timeout_action: fail
initial_step: wait-a
transitions:
  - from: wait-a
    to: wait-b
final_steps: [wait-b]
`

// retryFlaky is a single native step carrying a retry: policy, backed by
// examples.diagnostic.flaky. With succeed_on 2 the step fails attempt 1 and
// succeeds attempt 2, so a correct engine produces exactly one
// StepFailed{retrying:true} and then a StepCompleted.
const retryFlaky = `schema_version: 1
id: retry-flaky
version: 1.0.0
namespace: default
triggers:
  - type: manual
steps:
  - id: flaky
    type: native
    handler: examples.diagnostic.flaky
    inputs:
      succeed_on: 2
    retry:
      attempts: 3
      backoff: immediate
      initial_delay: 100ms
      max_delay: 1s
initial_step: flaky
final_steps: [flaky]
`

// withFallback is a native step carrying a fallback: to a recovery step. No
// forward transition is declared — the fallback edge alone satisfies the
// orphan/reachability check (internal/validate, EDR-010).
const withFallback = `schema_version: 1
id: with-fallback
version: 1.0.0
namespace: default
triggers:
  - type: manual
steps:
  - id: primary
    type: native
    handler: examples.diagnostic.fail
    fallback: rescue
  - id: rescue
    type: native
    handler: examples.diagnostic.succeed
initial_step: primary
final_steps: [rescue]
`

// withOnError is a native step with an on_error: true transition to a
// recovery step (B-4 fixture).
const withOnError = `schema_version: 1
id: with-on-error
version: 1.0.0
namespace: default
triggers:
  - type: manual
steps:
  - id: primary
    type: native
    handler: examples.diagnostic.fail
  - id: recover
    type: native
    handler: examples.diagnostic.succeed
transitions:
  - from: primary
    to: recover
    on_error: true
initial_step: primary
final_steps: [recover]
`

// terminalFail is a single native step that always fails with no retry, no
// fallback and no on_error edge — the plain WorkflowFailed route. It is the
// oracle for "a failing workflow actually reaches a TERMINAL status", which
// is the half of B-4 that a recovery-routed fixture cannot prove.
const terminalFail = `schema_version: 1
id: terminal-fail
version: 1.0.0
namespace: default
triggers:
  - type: manual
steps:
  - id: doomed
    type: native
    handler: examples.diagnostic.fail
    inputs:
      message: "terminal-fail fixture"
initial_step: doomed
final_steps: [doomed]
`

// panicStep is a single native step whose handler panics. The native runner
// must contain the panic as StepError{code:"handler_panic"} and the runtime
// process must survive — B-3 through the shipped binary rather than through a
// Go-level unit test.
const panicStep = `schema_version: 1
id: panic-step
version: 1.0.0
namespace: default
triggers:
  - type: manual
steps:
  - id: boom
    type: native
    handler: examples.diagnostic.panic
initial_step: boom
final_steps: [boom]
`

// slowStep is a native step that sleeps well past its own timeout: the
// engine must time it out rather than block the tick loop (B-2). timeout is
// deliberately far below duration_ms so the deadline, not the handler
// returning, is what ends the step.
const slowStep = `schema_version: 1
id: slow-step
version: 1.0.0
namespace: default
triggers:
  - type: manual
steps:
  - id: slow
    type: native
    handler: examples.diagnostic.slow
    timeout: 1s
    inputs:
      duration_ms: 30000
initial_step: slow
final_steps: [slow]
`

// slowThenDone pairs a slow step with a following step, giving a stable
// window in which the instance is observably 'running' — used to cancel an
// instance mid-flight rather than while it is parked on a signal.
const slowThenDone = `schema_version: 1
id: slow-then-done
version: 1.0.0
namespace: default
triggers:
  - type: manual
steps:
  - id: slow
    type: native
    handler: examples.diagnostic.slow
    timeout: 30s
    inputs:
      duration_ms: 5000
  - id: done
    type: native
    handler: examples.diagnostic.succeed
initial_step: slow
transitions:
  - from: slow
    to: done
final_steps: [done]
`
