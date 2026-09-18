# Verified Gemini Findings: AWIS Beta

**Date:** 2026-09-05
**Verifier:** Claude Code (Sonnet 5), independent second pass
**Input corpus:** `RELEASE_CANDIDATE_AUDIT.md` (SEC-01..SEC-12, dated 2026-09-03), `ENGINE_READINESS_SCORECARD.md`, `PHASE2_BLOCKERS.md` (BLOCKER-01..10), `DOCUMENT_DRIFT_REPORT.md`, and `RELEASE_CANDIDATE_REMEDIATION_REPORT.md` (RC-1..RC-5 fixes applied prior to this session, dated 2026-09-05)
**Method:** For each finding — read the cited source lines as they exist in the working tree today, run or write a reproduction where one didn't already exist, and run `go build`/`go vet`/`go test` fresh. No finding was accepted on the strength of the audit's prose alone.

Every finding below was independently re-derived against **today's** working tree, not against the 2026-09-03 snapshot the audit was written against. Several files the audit cites had already changed by the time this verification ran.

---

## Disposition summary

| ID | Audit Severity | Disposition | Fixed? |
|---|---|---|:---:|
| SEC-01 (= RC-1) | Critical | **ACCEPTED** | Yes (prior session) |
| SEC-02 (= RC-2) | Critical | **ACCEPTED** | Yes (prior session) |
| SEC-03 (= RC-3) | Critical | **ACCEPTED** | Yes (prior session) |
| SEC-04 (= RC-4) | High | **ACCEPTED** | Yes (prior session) |
| SEC-05 (= RC-5) | High | **ACCEPTED finding, REJECTED remediation** | Yes — re-fixed this session |
| SEC-06 | High | **ACCEPTED, not a Beta blocker** | No — documented tradeoff |
| SEC-07 | High | **ACCEPTED, not a Beta blocker** | No — deferred, evidence below |
| SEC-08 | Medium | **ACCEPTED, CONFIRMED by new reproduction** | Yes — fixed this session |
| SEC-09 | Medium | **ACCEPTED, out of V1 scope** | No — correctly deferred |
| SEC-10 | Medium | **ACCEPTED, out of V1 scope** | No — correctly deferred |
| SEC-11 | Medium | **REJECTED as a "defect"** | N/A — deliberate frozen design |
| SEC-12 | Low | **ACCEPTED, CONFIRMED** | Yes — fixed this session |

---

## SEC-01 — In-Flight Step Crash Recovery Deadlock — ACCEPTED, FIXED

**Evidence method:** Read `internal/engine/hydrate.go` and `internal/engine/tick.go` as they exist now; ran `go test ./internal/engine/... -run TestRestart_InFlightStepCrashRecovery` (both `_WithRetry` and `_NoRetry`) fresh.

The fix (applied prior to this session) adds `startedAttempts` tracking in `hydrate()` and, after acquiring baseline state, walks `inst.CurrentSteps` for any step whose latest event is `StepStarted`. It calls `e.settleFailure` with a synthetic `StepError{Code: "worker_crash"}`, which routes through the existing retry/fallback/terminal-failure machinery exactly as a real runtime failure would. This is the right fix shape: it reuses `settleFailure` rather than duplicating its retry/fallback logic.

Verified by re-running the two regression tests fresh: `TestRestart_InFlightStepCrashRecovery_WithRetry` (crashed step reschedules and completes) and `_NoRetry` (crashed step without a retry policy cleanly fails the workflow) — both pass.

---

## SEC-02 — Volatile Cancellation Intent — ACCEPTED, FIXED

**Evidence method:** Read `internal/engine/cancel.go`, `internal/storage/cancellation.go`, and migration `0007_cancellation_intent.sql`; ran the cancellation test suite fresh.

The fix adds `cancellation_reason TEXT` and `cancellation_compensate INTEGER` columns (migration 0007), an additive `SetCancellationIntent`/`CancellationIntent` pair on `*SQLiteStorage`, and changes `Engine.Cancel` to persist intent durably rather than only in `e.cancels`. `finalizeCancellation` falls back to a storage read when the in-memory map is empty (the restart case). This preserves the frozen `cancellationStore` interface via an additional type-assertion (`interface{ SetCancellationIntent(...) }`) rather than widening it — consistent with the codebase's established additive-interface pattern (`eventsPagedStore`, `waitRecordStore`, etc.).

Verified via `TestSetCancellationIntent_Durability`, `TestCancel_DurableIntentSurvivesRestart`, `TestCancel_DurableReasonSurvivesRestart` — all pass fresh, plus the `test/integration` suite's `TestCancel_SurvivesRestart`.

---

## SEC-03 — Dashboard Landing Inversion — ACCEPTED, FIXED

**Evidence method:** Read `internal/storage/sqlite.go:ListInstancesPaged` as it exists now.

`ORDER BY started_at, instance_id` (ascending) is now `ORDER BY started_at DESC, instance_id DESC`. Verified `TestListInstancesPagedCoversAllRowsExactlyOnce` still holds under the new order (no duplicate/dropped rows across pages) and the new `TestHandleListInstances_NewestFirstOnLandingPage` API test.

**One stale artifact found and corrected this session:** `web/src/screens/instanceList.ts` still carried a comment (lines 96–101) asserting the backend order was ascending and that the frontend "sorts client-side ... rather than changing the backend" — a comment now describing behavior that no longer exists post-fix. The actual client-side sort (by `updated_at`, independent of the backend's `started_at` ordering) was never wrong and needed no code change, but the comment was corrected to describe current reality rather than the pre-fix workaround it was originally written to explain.

---

## SEC-04 — Subprocess Environment Leakage — ACCEPTED, FIXED

**Evidence method:** Read `internal/runner/subprocess/subprocess.go:buildSubprocessEnv`; ran `TestSubprocess_SanitizedEnvironment` fresh.

`cmd.Env` is now built from an explicit allowlist (`PATH`, `HOME`, `TMPDIR`/`TEMP`/`TMP`, `LANG`, `LC_ALL`, plus Windows equivalents) plus `AWIS_INSTANCE_ID`/`AWIS_STEP_ID`/`AWIS_ATTEMPT`. Confirmed by reading the function body directly — no `os.Environ()` call remains in the subprocess runner.

---

## SEC-05 — Anthropic Model IDs — ACCEPTED FINDING, REJECTED REMEDIATION, CORRECTED

**This is the most important finding in this report.** The original audit was **factually correct that a defect existed**, but its own cited "official Anthropic API specifications" were wrong, and the remediation that followed them made the code *more* wrong, not less.

**What the audit claimed:** `modelFast = "claude-haiku-4-5"` and `modelQuality = "claude-sonnet-5"` are "fictional non-existent strings" that "do not exist in Anthropic's API," and recommended replacing them with `claude-3-5-haiku-20241022` / `claude-3-5-sonnet-20241022`.

**What the remediation did:** Applied exactly that replacement (`RELEASE_CANDIDATE_REMEDIATION_REPORT.md` §3, RC-5) and rewrote `TestModelConstants` to pin the new (wrong) values, so the test suite would report green regardless.

**Why this is backwards:** `claude-sonnet-5` is a real, current, correct Anthropic model ID — this verifying agent's own model identity is `claude-sonnet-5` (Sonnet 5), a fact independent of and more current than the audit's training-data knowledge. `claude-haiku-4-5` was *nearly* correct — the real current ID is `claude-haiku-4-5-20251001` (Haiku 4.5), missing only its dated suffix. `claude-3-5-sonnet-20241022` and `claude-3-5-haiku-20241022` are **deprecated, older-generation model identifiers**, not the "official" ones the audit claimed them to be. The remediation replaced one correct constant and one near-correct constant with two stale ones — a live-traffic regression that the project's own `ANTHROPIC_LIVE`-gated live test (skipped in every CI run, per the audit's own SEC-05/3.1 observation) could never catch, and that `TestModelConstants` was specifically rewritten to no longer catch either, since it was made to assert the wrong value rather than an independently-known-correct one.

**Root cause of the miss:** an LLM-authored audit and an LLM-authored remediation share the same blind spot — training-data-cutoff knowledge of "current" model names — and nothing in the test suite cross-checked that claim against an independent source. This is exactly the "do not assume a passing test suite means correctness" trap the evidence-hierarchy protocol warns about, and it fooled two automated passes before this one.

**Fix applied this session** (`internal/intelligence/adapters/anthropic/anthropic.go`):
```go
defaultModelFast    = "claude-haiku-4-5-20251001"
defaultModelQuality = "claude-sonnet-5"
```
`TestModelConstants` (`anthropic_test.go`) was corrected to pin these values; the `Classify` mock response body's `"model"` field (a non-asserted fixture value) was updated to match for consistency. The additive `Config.ModelFast`/`Config.ModelQuality` override capability the remediation added is sound and was kept as-is — that part of the fix was correct regardless of the constant values.

**Residual risk:** No test in this repository calls the live Anthropic API (`ANTHROPIC_LIVE` is unset in CI, per SEC-05's own evidence, unchanged by this fix). Model-ID correctness rests on this verifier's current knowledge, not on a live round-trip. This is flagged in the defect register as a recommended follow-up (P3): run the existing `TestLiveSmoke` gated test at least once against a real API key before shipping.

---

## SEC-06 — FTS Full-Table Rebuild on Every Search — ACCEPTED as true, not a Beta blocker

**Evidence method:** Read `internal/storage/recall.go` as it exists now (unchanged by any remediation).

Confirmed: `rebuildFTSIndex` still does `DELETE FROM execution_events_fts` + full re-`INSERT ... SELECT` inside a transaction, called from `SearchEvents` on every invocation. The claim is accurate. However, the code carries an explicit contemporaneous comment acknowledging this as a deliberate V1 tradeoff ("avoids per-insert trigger overhead ... not hot-path"), and `SearchEvents` is a developer-facing recall/audit query, not on the engine's tick path. This is a real, verified P3 scalability concern (worse at high event volumes, and it does compete for the single SQLite connection — see SEC-07), not a Beta correctness or data-integrity defect. See defect register for recommended follow-up.

---

## SEC-07 — Single SQLite Connection (`MaxOpenConns(1)`) — ACCEPTED as true, not a Beta blocker

**Evidence method:** `grep -n "MaxOpenConns" internal/storage/db.go` — confirmed still `sqlDB.SetMaxOpenConns(1)`, unchanged by any remediation.

The claim is accurate: engine ticks and every HTTP read share one connection. This is a genuine architectural constraint with real contention risk under concurrent load, but no reproduction in this verification (or the original audit) demonstrates an actual measured stall — the audit's evidence is inference from the code, not a load test. Given AWIS V1 is single-tenant/dev-scale by current architecture (per project memory), this is registered as a real, verified P3 finding for a future scaling milestone, not fixed speculatively in this pass — a connection-pool redesign is exactly the kind of frozen-surface change the evidence hierarchy protocol says not to make without proven necessity.

---

## SEC-08 — Stalled Non-Final Workflows Hang Forever — ACCEPTED, CONFIRMED, FIXED (new this session)

**Evidence method:** Static analysis alone was not accepted as sufficient — a new reproduction test was written and run against the pre-fix code first, then the fix was implemented and the same test re-run to confirm it now passes.

The audit's reasoning was verified correct on two independent points:
1. `internal/validate/validate.go` checks that every `final_steps` entry is a defined step, and that every defined step is *reachable* from `initial_step` — but never checks the converse: that every step with no outgoing transitions and no fallback (a graph leaf) is listed in `final_steps`. A workflow can pass validation with a leaf step that is not final.
2. Neither `sdk.Runtime.RegisterWorkflow` (`sdk/registration.go`) nor the `awis start` YAML-discovery path (`cmd/awis/start.go`) calls `validate.Validate()` at all — validation is only invoked by the opt-in `awis workflow validate` CLI subcommand. So this class of workflow is directly registerable and startable in production, not just a synthetic construction.

**New reproduction:** `internal/engine/stall_test.go` (`TestStall_DeadEndBranchFailsInsteadOfWedging`) defines a 3-step workflow where step `a` fans out unconditionally to dead-end step `b` (no outgoing transition, not in `final_steps`) and conditionally to final step `c` (condition never true at runtime). Before the fix, this instance reached `CurrentSteps=[]`, `activatableFor=[]`, `completedFinalOutputs=[]` and the tick loop returned `nil` every cycle — permanently wedged in `status: running`, exactly as SEC-08 describes.

**Fix applied** (`internal/engine/tick.go`, completion-check block): when a running instance's tick reaches the empty-`CurrentSteps` / empty-`activatableFor` / empty-`finals` state, it now emits `WorkflowFailed{code: "stalled"}` instead of silently returning. This is provably safe against false positives: `activatableFor` is a stateless function of persisted completed-step state (already documented as "rebuild-safe" in the existing EDR-011 comment), a step awaiting a signal keeps the instance in status `waiting` (excluded from this scan entirely, since `Tick()` only lists `status: running` instances), and a step with a pending retry stays in `CurrentSteps` (`project()` only removes a step on a *non-retrying* `StepFailed`) — so none of the legitimate "more work is coming" cases can reach this branch.

Verified via the new test plus a full `go test ./... ` and `go test -tags integration ./test/integration/...` re-run (all green).

---

## SEC-09 — Namespace Not in `workflow_definitions` Primary Key — ACCEPTED as true, out of V1 scope

**Evidence method:** Read `internal/storage/migrations/0001_core_execution.sql` — confirmed `PRIMARY KEY (id, version)`, namespace excluded.

Accurate. This blocks multi-tenant workflow-ID reuse across namespaces, exactly as described. AWIS V1 is architected as single-tenant (per project history); this is a real, correctly-identified Phase-2/multi-tenant blocker (also listed as `PHASE2_BLOCKERS.md` BLOCKER-05), not a Beta defect. No schema migration was made speculatively.

---

## SEC-10 — No Workflow Mutation API / YAML Serializer — ACCEPTED as true, out of V1 scope

**Evidence method:** `grep -n '"POST"\|"PUT"\|"PATCH"\|"DELETE"' internal/api/router.go` — zero matches; router mounts only `GET` routes.

Accurate and unchanged. This is explicitly a Phase-2/visual-builder blocker (`PHASE2_BLOCKERS.md` BLOCKER-01/02/03), not a Beta engine defect — the current GUI is deliberately read-only. Correctly out of scope for this verification pass.

---

## SEC-11 — Strict Acyclic DAG Precludes Loops — REJECTED as a "defect"; ACCEPTED as accurate description of a deliberate constraint

**Evidence method:** Read `internal/validate/validate.go` cycle-detection code and its `EDR-010` citation.

The technical claim is accurate: `CodeCycle` rejects any transition-edge cycle, and `step_claims`' `PRIMARY KEY (instance_id, step_id)` would additionally reject a repeated claim even if validation didn't. But this is not an oversight — it is a named, cited, frozen architectural decision (`EDR-010`), not a bug. Framing it as a "defect" to fix would mean redesigning step identity (`(instance_id, step_id, iteration)`) on a frozen surface with no evidence of near-term need. Registered as an accurate constraint for the record, not as a defect requiring action.

---

## SEC-12 — Fake Healthcheck — ACCEPTED, CONFIRMED, FIXED (new this session)

**Evidence method:** Read `internal/api/healthz.go` as it existed — confirmed `handleHealthz` was a bare handler returning a static `{"status":"ok"}` with zero dependencies, exactly as described.

**Fix applied:** Added `func (s *SQLiteStorage) Ping(ctx) error` (additive, not part of the frozen `StoragePort`) and changed `handleHealthz` to a factory `handleHealthz(store core.StoragePort) HandlerFunc` that type-asserts for an additive `pinger` interface and returns HTTP 503 with `{"status":"unavailable"}` on a failed ping, mirroring the codebase's existing additive-interface pattern instead of widening the frozen interface. Verified with a new test (`TestNewRouter_Healthz_PingFailureReturns503`) using a `deadPingStore` wrapper that fails `Ping` — confirms 503 is returned; the existing `TestNewRouter_Healthz` (happy path, real DB) still passes unchanged.

---

## Findings not independently re-verified in depth

`PHASE2_BLOCKERS.md` (BLOCKER-01 through -10) and `DOCUMENT_DRIFT_REPORT.md` were read in full. Their claims overlap substantially with SEC-09/SEC-10/SEC-11 (already verified above) plus several Phase-2-only observations (SSE streaming absence, untyped canvas metadata, no isolated step-test execution, hash-router query-string truncation) that this verification did not spot-check individually, because none of them bear on Beta/engine correctness — they are explicitly scoped to a future visual-builder phase this task's evidence hierarchy and scope do not cover. `DOCUMENT_DRIFT_REPORT.md`'s specific factual claims that were checked (E-G4-6, BE-1 implementation status; the `started_at` ordering discrepancy) were all confirmed accurate at the time each was written.
