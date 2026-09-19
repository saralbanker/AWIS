# Verified Defect Register: AWIS Beta

**Date:** 2026-09-05
**Status:** All P0/P1 entries closed. P3 entries deferred with recorded justification.

Severity key: P0 = correctness, P1 = data-loss risk, P2 = security risk, P3 = scalability risk, P4 = maintainability, P5 = enhancement.

---

## D-01 — In-flight step crash recovery deadlock

- **Severity:** P0 (correctness) / P1 (data-loss adjacent — a wedged instance never completes its side effects)
- **Impact:** Any process crash or restart while a step is executing permanently wedges that workflow instance in `status: running`. No further ticks make progress; the instance can never complete, fail, or be retried.
- **Root cause:** `hydrate()` only parsed `EventTypeStepFailed` when reconstructing retry state on process start. A step whose last event was `StepStarted` (in-flight at crash time) was left in `CurrentSteps` but absent from the retry map, so both the activation path (`activatableFor` skips running steps) and the retry path (`dueRetries` — empty map) produced nothing to dispatch.
- **Status:** **FIXED** (prior session, verified this session). `hydrate()` now detects any `CurrentSteps` entry whose latest event is `StepStarted` and routes it through `settleFailure` with a synthetic `worker_crash` error, so normal retry/fallback/terminal-failure policy applies.
- **Evidence:** `internal/engine/hydrate.go`; `TestRestart_InFlightStepCrashRecovery_WithRetry`, `TestRestart_InFlightStepCrashRecovery_NoRetry` (both pass fresh).

---

## D-02 — Cancellation intent lost on restart

- **Severity:** P0 (correctness) / P1 (data-loss — compensation is skipped, external side effects go uncompensated)
- **Impact:** `Cancel(reason, compensate=true)` recorded intent only in an in-memory map. A restart before the cancellation finalized silently dropped `reason` and `compensate`, causing the workflow to finalize as plain `cancelled` with an empty reason and **without running compensation handlers**.
- **Root cause:** `e.cancels[instanceID]` was the sole store of cancellation intent; `workflow_instances.cancellation_requested` was the only durable bit, carrying no reason/compensate payload.
- **Status:** **FIXED** (prior session, verified this session). Migration `0007_cancellation_intent.sql` adds `cancellation_reason`/`cancellation_compensate` columns; `Engine.Cancel` persists intent durably; `finalizeCancellation` falls back to a storage read when the in-memory map is empty.
- **Evidence:** `internal/storage/migrations/0007_cancellation_intent.sql`, `internal/engine/cancel.go`; `TestCancel_DurableIntentSurvivesRestart`, `TestCancel_DurableReasonSurvivesRestart`, integration `TestCancel_SurvivesRestart` (all pass fresh).

---

## D-03 — Dashboard landing page shows oldest instances, not newest

- **Severity:** P0 (correctness — the primary dashboard use case is broken)
- **Impact:** On any deployment with >50 instances, the operator landing page showed the 50 *oldest* instances ever recorded; active/recent/failed workflows never appeared on page 1.
- **Root cause:** `ListInstancesPaged` executed `ORDER BY started_at, instance_id` (ascending).
- **Status:** **FIXED** (prior session, verified this session). Changed to `ORDER BY started_at DESC, instance_id DESC`. A stale explanatory comment in the frontend (`web/src/screens/instanceList.ts`) describing the old ascending order was also corrected this session (comment-only; no behavior change — the client-side sort it described was never itself wrong).
- **Evidence:** `internal/storage/sqlite.go:ListInstancesPaged`; `TestListInstancesPagedCoversAllRowsExactlyOnce`, `TestHandleListInstances_NewestFirstOnLandingPage` (pass fresh).

---

## D-04 — Subprocess steps inherit full host environment

- **Severity:** P2 (security — secret leakage)
- **Impact:** Any workflow author with access to a `subprocess` step type could read `ANTHROPIC_API_KEY` and any other host secret via `os.Environ()` inheritance.
- **Root cause:** `cmd.Env` was never set on the `exec.Command`, so Go's stdlib inherited the full parent environment by default.
- **Status:** **FIXED** (prior session, verified this session). `buildSubprocessEnv` now allowlists `PATH`, `HOME`, `TMPDIR`/`TEMP`/`TMP`, `LANG`, `LC_ALL` (+ Windows equivalents) and injects `AWIS_INSTANCE_ID`/`AWIS_STEP_ID`/`AWIS_ATTEMPT` only.
- **Evidence:** `internal/runner/subprocess/subprocess.go`; `TestSubprocess_SanitizedEnvironment` (passes fresh).

---

## D-05 — Anthropic model identifiers: original bug real, first fix wrong

- **Severity:** P0 (correctness — live Claude-backed steps would 404) for the original defect; **P0 regression** for the first remediation attempt.
- **Impact (original):** `modelFast = "claude-haiku-4-5"` was missing its date suffix; a live call would 404.
- **Impact (first remediation, now reverted):** The fix that shipped ahead of this verification replaced both constants with deprecated Claude 3.5-generation IDs (`claude-3-5-sonnet-20241022`, `claude-3-5-haiku-20241022`), incorrectly overwriting `modelQuality`, which was **already correct** (`claude-sonnet-5`). The regression test (`TestModelConstants`) was rewritten to pin the wrong values, so it could not catch this.
- **Root cause:** Both the original audit and the first remediation relied on stale/incorrect "current model ID" knowledge, and no test in the suite cross-checks model IDs against a source independent of the constant itself (`ANTHROPIC_LIVE` live test is skipped in CI).
- **Status:** **FIXED (corrected) this session.** `defaultModelFast = "claude-haiku-4-5-20251001"`, `defaultModelQuality = "claude-sonnet-5"`. `TestModelConstants` re-pinned to these values.
- **Evidence:** `internal/intelligence/adapters/anthropic/anthropic.go`, `anthropic_test.go`. See `VERIFIED_GEMINI_FINDINGS.md` §SEC-05 for the full reasoning.
- **Residual risk (recommended follow-up, P3):** No test in this repo calls the live Anthropic API. Run the existing `ANTHROPIC_LIVE`-gated `TestLiveSmoke` at least once with a real key before shipping, to confirm these IDs resolve against the live API rather than resting solely on this verifier's knowledge.

---

## D-06 — Non-final dead-end branch wedges the instance in `running` forever

- **Severity:** P0 (correctness)
- **Impact:** A workflow where an actually-taken execution path terminates on a step with no outgoing transition that isn't listed in `final_steps` (while a sibling branch that *would* reach a final step never fires) leaves the instance permanently in `status: running`, consuming a tick every cycle with no possible progress.
- **Root cause:** Two compounding gaps. (1) `internal/validate/validate.go` checks final_steps entries are defined and that every defined step is reachable from `initial_step`, but never checks the converse — that every graph leaf (no outgoing transition, no fallback) is itself a final step. (2) Neither `sdk.Runtime.RegisterWorkflow` nor the `awis start` YAML-discovery path invokes `validate.Validate()` at all, so this workflow shape is directly deployable, not just a validator gap.
- **Status:** **FIXED this session** (new finding, confirmed by new reproduction, not present in the audit's five headline defects but matches the audit's separately-listed SEC-08). `internal/engine/tick.go`'s completion check now emits `WorkflowFailed{code: "stalled"}` when `CurrentSteps`, `activatableFor`, and `completedFinalOutputs` are simultaneously empty for a running instance — proven unreachable for the legitimate "more work coming" cases (waiting instances carry a distinct `status: waiting` excluded from this scan; steps with a pending retry stay in `CurrentSteps`).
- **Evidence:** `internal/engine/tick.go`; new test `internal/engine/stall_test.go:TestStall_DeadEndBranchFailsInsteadOfWedging` (passes; fails against the pre-fix code, confirming it's a real reproduction and not a vacuous test).

---

## D-07 — Healthcheck endpoint never checks the database

- **Severity:** P2 (observability / operational risk — an orchestrator will keep routing traffic to a dead instance)
- **Impact:** `GET /api/v1/healthz` unconditionally returned HTTP 200 `{"status":"ok"}` regardless of database or engine state.
- **Root cause:** `handleHealthz` had no dependencies at all by design; it was never wired to storage.
- **Status:** **FIXED this session** (new finding, matches audit SEC-12). Added an additive `Ping(ctx) error` method to `*SQLiteStorage` (not part of the frozen 12-method `StoragePort`) and changed `handleHealthz` into a factory that type-asserts for it, returning HTTP 503 `{"status":"unavailable"}` on a failed ping.
- **Evidence:** `internal/storage/sqlite.go:Ping`, `internal/api/healthz.go`, `internal/api/router.go`; new test `TestNewRouter_Healthz_PingFailureReturns503` (passes) alongside the pre-existing happy-path `TestNewRouter_Healthz` (still passes).

---

## D-08 — FTS index fully rebuilt on every search query

- **Severity:** P3 (scalability)
- **Impact:** `SearchEvents` acquires an exclusive write transaction and does `DELETE` + full re-`INSERT ... SELECT` over `execution_events_fts` on every call. At high event volumes this is O(n) per search and, combined with D-09's single connection, can stall the engine tick loop while a search runs.
- **Root cause:** Deliberate V1 simplification (explicitly commented in-code) to avoid per-insert trigger overhead, on the assumption search is a low-frequency, developer-facing path.
- **Status:** **NOT FIXED — deferred with justification.** Real and verified, but not a Beta-blocking correctness or data-loss defect, and changing to trigger-based incremental indexing is a non-trivial migration change against a documented tradeoff, not a bug. Recommended as future work: `AFTER INSERT` triggers or a `last_indexed_rowid` incremental-index cursor.
- **Evidence:** `internal/storage/recall.go:rebuildFTSIndex`.

---

## D-09 — Single SQLite connection shared by engine tick and all HTTP reads

- **Severity:** P3 (scalability)
- **Impact:** `sqlDB.SetMaxOpenConns(1)` means every HTTP API read and the 100ms engine tick contend for the same connection. Under load, dashboard polling can delay tick execution and vice versa.
- **Root cause:** Single-connection design chosen for write-serialization simplicity; no separate read pool exists despite WAL mode being enabled (which would otherwise support concurrent readers).
- **Status:** **NOT FIXED — deferred with justification.** Real and verified via direct code inspection, but neither the original audit nor this verification produced a load test demonstrating actual production-scale contention, and AWIS V1 is architected single-tenant/dev-scale. A connection-pool redesign is a structural change to a frozen-adjacent surface; making it without demonstrated need would violate the "do not modify frozen surfaces without proving necessity" principle this verification operated under. Recommended as a P3 item for the first milestone that targets concurrent multi-user load.
- **Evidence:** `internal/storage/db.go:42`.

---

## D-10 — `workflow_definitions` primary key omits namespace

- **Severity:** P3 (scalability / multi-tenancy)
- **Impact:** Two different namespaces cannot register the same `(id, version)` pair; namespace is a filter label, not an isolation boundary.
- **Status:** **NOT FIXED — out of V1 scope.** AWIS V1 is single-tenant by design. Correctly deferred to the multi-tenant milestone (also tracked as `PHASE2_BLOCKERS.md` BLOCKER-05).
- **Evidence:** `internal/storage/migrations/0001_core_execution.sql:18`.

---

## D-11 — No workflow mutation API or YAML serializer

- **Severity:** P5 (enhancement — expected absence, not a defect)
- **Impact:** Blocks any workflow-creation/editing UI or visual builder; read-only V1 GUI cannot save changes.
- **Status:** **NOT FIXED — out of V1 scope by design.** The current GUI is deliberately read-only; this is explicitly Phase-2 scope (`PHASE2_BLOCKERS.md` BLOCKER-01/02/03), not an engine or Beta defect.
- **Evidence:** `internal/api/router.go` (zero mutation routes).

---

## D-12 — Strict acyclic DAG precludes loop constructs

- **Severity:** N/A — not a defect
- **Impact:** Cannot express n8n-style iteration/loop nodes.
- **Status:** **NOT A DEFECT.** Deliberate, cited, frozen architectural decision (`EDR-010`). No action taken or recommended without a product decision to support loops, which would require redesigning step identity on a frozen surface.
- **Evidence:** `internal/validate/validate.go` (cycle rejection), `internal/storage/migrations/0001_core_execution.sql:48` (`step_claims` PK).

---

## D-13 — Flaky system-rehearsal test under parallel load

- **Severity:** P4 (maintainability / CI hygiene — not a product defect)
- **Impact:** `TestSystemRehearsalInitStartSubmitTrace` (`cmd/awis`) intermittently fails with "did not reach a terminal status within 10s" when run as part of the full `go test ./...` suite, though it consistently passes (5/5 in this verification) when run in isolation, taking ~11s wall-clock even then.
- **Root cause:** The test's 10-second polling deadline is tighter than its own typical single-run duration (~11s), so any additional CPU contention from running the full suite in parallel pushes it over. Not caused by, or a regression from, any fix in this report — reproduced both before and after all fixes in this session.
- **Status:** **NOT FIXED — recommended follow-up (P4).** Loosen the deadline (e.g. to 20–30s) or mark the test to run outside `-parallel`, since it is exercising real process spawn + CLI + engine tick latency, not a hung condition.
- **Evidence:** `cmd/awis/system_test.go`; observed failure in the full-suite run, 3/3 pass in isolated re-runs, 5/5 pass in a `-count=5` isolated re-run.
