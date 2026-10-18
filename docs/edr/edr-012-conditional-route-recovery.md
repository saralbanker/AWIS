# EDR-012 — Recovery of a Condition-Dependent Terminal-Failure Route (LA-D2)

**Status:** Proposed — founder approved direction A (2026-10-18); this design and its acceptance tests
await founder approval before implementation.
**Authored:** Loop A, CE (2026-10-18)
**Coordinates:** EDR-011 §8 (failure routing order) · EDR-010 (expression semantics) · EDR-007 §2
(projection) · STATE.md "LOOP A" → REVIEW ROUND 3 (R3-H1), LA-R1, LA-D2

---

## 1. Problem

`routeTerminalFailure` (failure.go) picks the route of a terminally failed step:
(1) `fallback` set → fallback; (2) else, if any `on_error` transition *fires* → those targets;
(3) else `WorkflowFailed` and compensation. Whether (2) fires depends on transition
**conditions** evaluated against `buildEnv(inst)`. The EventLog records `StepFailed{retrying:false}`
but not which route was chosen.

If a write after that `StepFailed` is lost (append failure, ack-lost, crash), recovery must finish
the route. Today recovery (LA-C3 `completeCommittedRoutes`) only completes routes that are
determined by **static structure**. A step with any `on_error` transition is skipped, so a step
whose conditions all evaluated false ends as SEC-08 `stalled` (no compensation). The no-fault run
ends `WorkflowFailed` with compensation.

## 2. Decision (founder, direction A)

Recovery reconstructs the condition environment that the live path used, and re-evaluates the
`on_error` conditions against it. The route that results is the route the live path took.

## 3. Is the existing EventLog sufficient? — Yes, with one boundary rule

`buildEnv` (transition.go) reads exactly three things from the projection:

| Env field | Live source | EventLog source | Sufficient? |
|---|---|---|---|
| `Inputs` | `Variables["inputs"]` | `WorkflowStarted` payload `inputs` | yes |
| `StepOutputs[s]` | `Variables[s]` | latest `StepCompleted(s)` payload `outputs` (signal steps: the signal payload) | yes |
| `StepStatus[s]="completed"` | key present in `Variables` | a `StepCompleted(s)` exists | yes |
| `StepStatus[s]="running"` | `s` ∈ `CurrentSteps` | `s` started and not yet settled at the boundary (below) | yes, given the boundary |

Expressions are pure (internal/expr has no clock or randomness; `ConditionExpr.Eval` depends only
on `Env`), so the same env always gives the same result.

**The boundary.** The live path does not evaluate against "the state just before the
`StepFailed`". Each call site passes a specific snapshot:

| Failure source (by `StepFailed.error.code` / call site) | Snapshot the live path used | Reconstruction rule |
|---|---|---|
| Handler failure settled in `processInstance` (any code except below) | `inst` read at the start of `processInstance`, **before** that tick's gather | fold every event with `sequence_num` < the first `StepStarted` of the **gather batch** that contains the failing attempt's `StepStarted` |
| `worker_crash` (RC-1, hydrate) | projection at hydrate time | fold every event with `sequence_num` < the `StepFailed` |
| `signal_timeout` with `timeout_action=fail` | projection when the timeout scan fired | fold every event with `sequence_num` < the `StepFailed` |

A **gather batch** is the maximal run of consecutive `StepStarted` events (no other event type
between them) that ends at the failing attempt's `StepStarted`. `gatherDispatch` emits all of a
tick's `StepStarted` events before any dispatch or settle, so the batch boundary is the snapshot
boundary. Only `StepStatus` differs across this boundary; inputs and outputs do not.

**One residual edge:** two ticks whose `StepStarted` events are adjacent with nothing between them
(the first tick's items stranded with no settlement) would merge into one batch. RC-1 settles
stranded items before any new gather, which puts a `StepFailed` between the two runs, so this needs
a crash between the two ticks *and* a failed RC-1. In that case the reconstructed status of those
stranded steps differs. A condition that reads the status of a stranded sibling is the only
expression affected. The design records this as a known limitation instead of adding a tick
marker (a tick marker would be a frozen-format change).

## 4. Design

1. New engine function `reconstructEnvAt(evs, boundarySeq) expr.Env`. It folds events through the
   same per-event rules as `applyEventToInstance` (EDR-007 §2), and must not re-implement them:
   reuse `applyEventToInstance` on a scratch `WorkflowInstance`, then call `buildEnv`.
2. `routeBoundary(evs, failed StepFailed) int` implements the table in §3.
3. `completeCommittedRoutes` (LA-C3) replaces its static "any `on_error` ⇒ skip" rule with:
   `targets := firingOnError(dv, step, reconstructEnvAt(evs, routeBoundary(...)))`.
   - `Fallback != ""` → route (1) (unchanged);
   - `len(targets) > 0` → route (2): mark those targets pending (the targets the live path chose);
   - else → route (3): `WorkflowFailed` with the original step and error, then `maybeCompensate`.
4. The settle-side check (`committedRouteThree`, LA-C6) and recovery use the same evaluation, so a
   committed decision is classified identically on both sides.
5. The existing hydrate pending-target computation for terminally failed steps (LA-R1: it
   currently re-evaluates against the *recovery-time* env) switches to the same reconstructed env.
   This closes LA-R1.

No storage, schema, event-type or payload change.

## 5. Acceptance tests (each must FAIL on the current code where marked)

| # | Scenario | Assertion | Fails today |
|---|---|---|---|
| D2-1 | root→{b1,b2}; b1 has `on_error`→x with a condition that is FALSE; b1 fails; ack-lost `StepFailed(b1)`; b2 succeeds | event types/order and final status identical to the no-fault run (`WorkflowFailed(b1)` + compensation); same-process and after restart | yes (ends `stalled`) |
| D2-2 | as D2-1 but the condition is TRUE | x activates exactly once; no `WorkflowFailed`; identical to no-fault | guard |
| D2-3 | condition reads a sibling's output; the sibling completes in the same tick **after** b1's `StepStarted` | recovery uses the pre-tick snapshot (sibling output absent), matching the live result | yes |
| D2-4 | condition reads `steps.<sib>.status`; sibling started in the same gather batch | reconstructed status matches the live snapshot | yes |
| D2-5 | `worker_crash` (RC-1) and `signal_timeout`(fail) routes with conditional `on_error` | boundary = before the `StepFailed`; result matches the live path | yes |
| D2-6 | LA-R1: a condition whose truth changes between settle and recovery | recovery follows the settle-time truth, not the recovery-time truth | yes |
| D2-7 | property test: random fan-out workflows with conditional `on_error`, a fault injected at each write after `StepFailed{retrying:false}` | faulted run's event-type sequence == no-fault run's | yes |

## 6. Failure modes and risks

- **Rule drift:** if `applyEventToInstance` and the scratch fold ever diverge, recovery picks a
  different route. Mitigation: reuse the same function (§4.1) and test D2-7.
- **Definition change:** the condition is evaluated against the definition version the instance
  runs (`defViewFor`); definitions are immutable per version, so this is stable.
- **Cost:** one event-log fold per recovered failure; recovery-only path.
- **The §3 residual edge** (adjacent batches) stays documented and tested by name.
