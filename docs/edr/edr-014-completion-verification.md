# EDR-014 — Independent Task-Completion Verification (LA-Q1) — Feasibility Study

**Status:** Proposed (feasibility) — the founder set the direction (2026-10-18); nothing here is
authorized for implementation. Phase 1 needs founder approval because it changes the RC-1 outcome
contract. Phase 2 needs a CONTRA (frozen surfaces).
**Authored:** Loop A, CE (2026-10-18)
**Coordinates:** RC-1 / B-0 (hydrate.go in-flight recovery) · Blueprint §20 (IdempotencyKey) ·
TDS-04 (subprocess protocol) · plugin JSON-RPC protocol · PRD V2 server mode · EDR-013 (stop path)

---

## 1. Question

A step's handler succeeded, but the `StepCompleted` write was lost (append failure or crash). Today
RC-1 records it as `StepFailed{worker_crash}` (LA-Q1). The founder asked: can completion be verified
independently, using multiple sources with fallback (local system/HAL, server), and how are
disagreements resolved?

## 2. Candidate sources and their availability in V1

| # | Source | What it proves | Durable? | Available in V1? | Cost to adopt |
|---|---|---|---|---|---|
| S1 | EventLog `StepCompleted` | completion (authoritative) | yes | yes | none; absent by definition in this case |
| S2 | Idempotency cache `step_results_cache[instance/step/attempt]`, written by `dispatchOne` after a successful Run, **before** settle | the handler returned success, with its outputs | yes, 24h TTL | **yes, already written today** | small: RC-1 consults it before `worker_crash` |
| S3 | Runner receipts: subprocess exit status + stdout, plugin JSON-RPC response | the runner saw success | no (in-memory; gone after reap/return) | not durably | persisting them would duplicate S2 |
| S4 | Handler-provided verifier: native `Verify(ctx, sc)`, plugin `verify` method, subprocess verify mode; it queries the external system the step acted on | the external side effect exists | depends on the external system | **no: new surface** | frozen SDK + plugin/subprocess protocol (TDS-04) change → **CONTRA** |
| S5 | "Local system / HAL": OS facts (process table, files, exit codes) | workload-specific facts only | partly | not generically | only meaningful *inside* an S4 verifier for a given handler |
| S6 | "Server" as an independent witness | another node's view | yes | **no**: `awis-server` serves static GUI assets; server mode is V2 (PRD "Server mode (HTTP API) — V2") | V2 architecture |

**Feasibility verdict.**
- S2 is the only independent durable source that exists today. Phase 1 is feasible now, with no
  format change.
- S4 is feasible but needs a frozen-surface change. It is the only way to verify **external** side
  effects.
- S5 is not a generic source; it is an implementation detail of individual S4 verifiers.
- S6 is not available until V2.

## 3. Proposed phases

**Phase 1 (cache-backed recovery).** In RC-1, before settling an in-flight step as `worker_crash`,
look up `instance/step/attempt` in `step_results_cache`.
- **Hit:** append `StepCompleted` with the cached outputs (a durable record of the handler's own
  success), then follow the normal transition path. The handler is not re-run.
- **Miss** (cache write failed, TTL expired, handler never finished): current behavior
  (`worker_crash` → retry policy or terminal route).

This answers LA-Q1 for every runner type that reaches `dispatchOne` (native, plugin, subprocess,
intelligence).

**Phase 2 (verifier interface, CONTRA).** Add an optional verifier per handler type. Use it only
when S2 misses and the step has a verifier, or to cross-check an S2 hit when the step declares
`verify: required`.

**Phase 3 (V2).** A server or cluster witness, when a multi-worker StateStore exists.

## 4. Disagreement resolution (must be fixed before any implementation)

Principles:
- The EventLog is authoritative.
- A positive durable source can complete a step.
- A "not done" from a non-authoritative source never authorizes re-running a handler beyond its
  retry policy.
- Any contradiction **stops with evidence** (EDR-013 stop path), never guesses.

| S1 log | S2 cache | S4 verifier (Phase 2) | Outcome |
|---|---|---|---|
| completed | any | any | done (no action; the fence prevents duplicates) |
| — | hit | not used / unavailable | complete from cache (Phase 1) |
| — | hit | done | complete from cache |
| — | hit | **not done** | **disagreement → stop** (`WorkflowCompensationFailed` / `WorkflowFailed{code: verification_conflict}` with evidence from both sources) |
| — | miss | done + outputs | complete with the verifier's outputs (Phase 2) |
| — | miss | done, no outputs | stop with evidence (cannot fabricate outputs) |
| — | miss | not done | `worker_crash` (current RC-1: retry policy, or route terminal) |
| — | miss | unavailable / error / timeout | fallback: `worker_crash` (current) |

## 5. Failure modes

- **Cache written, handler's external effect only partially applied** (the handler returned
  success incorrectly). S2 cannot detect this; only S4 can. Phase 1 trusts the handler's own
  success report, exactly as the no-fault path already does.
- **Cache write failed after the handler succeeded** → miss → `worker_crash` → the handler may run
  again under its retry policy. This is the residual at-least-once window, the same as today.
- **TTL expiry (24h):** recovery after more than 24h falls back to current behavior.
- **Intelligence steps:** the ADJ-8 usage triple (adapter, model, tokens) is a side channel that is
  **not** cached today. A cache-recovered intelligence `StepCompleted` would lack usage, so
  `awis trace` would show no adapter/tokens for it. Options: also cache usage (same table, a second
  key), or accept and document. Founder choice.
- **Ack-lost `StepCompleted`** (durable but reported failed): the log already has it, so S1 wins
  and no duplicate is appended (LA-C4a fence and catch-up).
- **Verifier hazards (Phase 2):** a verifier is user code. It can hang (needs its own timeout),
  fail, or lie. Mitigations: bounded timeout, "unavailable" falls back as above, contradictions stop.

## 6. Acceptance tests

| # | Scenario | Assertion |
|---|---|---|
| Q1-1 | L1: handler succeeds, `StepCompleted` append fails (same process, and after restart) | `StepCompleted` with the cached outputs; handler ran exactly once; workflow completes like the no-fault run |
| Q1-2 | as Q1-1, cache write also failed | current behavior (`worker_crash`), recorded |
| Q1-3 | cache entry expired (fake clock +25h) | current behavior |
| Q1-4 | retry attempt n: cache holds attempt n−1 only | no false hit (key includes attempt) |
| Q1-5 | ack-lost `StepCompleted` | no duplicate `StepCompleted` |
| Q1-6 | intelligence step recovered from cache | outputs match; usage per the §5 ruling |
| Q1-7…n | Phase 2 truth table §4, one test per row | as in the table |

## 7. Founder decisions needed

1. Approve Phase 1. It changes the LA-Q1 classification from `worker_crash` to `StepCompleted from
   cache` on a cache hit.
2. Intelligence usage on cache recovery: cache it, or document its absence.
3. Phase 2: open a CONTRA for the verifier interface (SDK + plugin + subprocess protocols), or defer.
4. Confirm that disagreement always stops (no majority vote among sources).
