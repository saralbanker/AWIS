# Implementation Report: AWIS Beta Verification & Remediation

**Date:** 2026-09-05
**Scope of this report:** Changes made *by this verification session*. Fixes for RC-1 through RC-4 (SEC-01..SEC-04) were already applied and committed to the working tree by a prior session before this verification began; they are listed in `VERIFIED_GEMINI_FINDINGS.md` as independently re-verified, not re-implemented here. This report covers only the changes this session authored: the SEC-05 correction, and the two new defects (D-06/SEC-08, D-07/SEC-12) this session found and fixed.

---

## Change 1: Corrected Anthropic model identifiers (SEC-05 regression fix)

**Files:**
- `internal/intelligence/adapters/anthropic/anthropic.go`
- `internal/intelligence/adapters/anthropic/anthropic_test.go`

**Reason:** The prior session's SEC-05 fix replaced the original (partially wrong) model constants with deprecated Claude 3.5-generation IDs, incorrectly overwriting `modelQuality`, which was already the correct current ID (`claude-sonnet-5`). See `VERIFIED_GEMINI_FINDINGS.md` §SEC-05 for the full evidentiary reasoning.

**What changed:**
```diff
- defaultModelFast    = "claude-3-5-haiku-20241022"
- defaultModelQuality = "claude-3-5-sonnet-20241022"
+ defaultModelFast    = "claude-haiku-4-5-20251001"
+ defaultModelQuality = "claude-sonnet-5"
```
`TestModelConstants` was re-pinned to the corrected values; the `Classify` mock response body's non-asserted `"model"` field was updated for consistency (`"claude-haiku-4-5"` → `"claude-haiku-4-5-20251001"`).

**Evidence:** `go build ./...`, `go vet ./...`, `go test ./internal/intelligence/...` all pass. No live API call was made (this repo has no live Anthropic credentials available in this environment) — see the residual-risk note in the defect register recommending a live-gated smoke test before shipping.

---

## Change 2: Stalled dead-end branches now fail instead of wedging (new defect, SEC-08)

**Files:**
- `internal/engine/tick.go` (fix)
- `internal/engine/stall_test.go` (new regression test)

**Reason:** Confirmed via new reproduction that a workflow whose taken execution path dead-ends on a non-final leaf step (while a sibling branch that would reach a final step never fires) permanently wedges the instance in `status: running`. This matches the audit's SEC-08 finding, which the prior remediation session did not address (it fixed only the five RC-1..RC-5 headline defects).

**What changed** (`internal/engine/tick.go`, completion-check block at the end of `processInstance`):
```diff
  finals := completedFinalOutputs(dv, inst2)
  if len(finals) == 0 {
-     return nil // stalled (join stall / no final reached) — author semantics, EDR-011 §1.
+     // SEC-08: permanent fixed point — see comment in the file for the full
+     // safety argument (waiting instances excluded by status, retries stay
+     // in CurrentSteps).
+     return e.emitWorkflowFailed(ctx, inst2, "", core.StepError{
+         Code:    "stalled",
+         Message: "workflow reached a dead end: no final step was completed and no further step is activatable",
+     })
  }
  return e.emitWorkflowCompleted(ctx, inst2, finals)
```

**Why this is safe** (verified by reading, not just asserted): the three preconditions (`CurrentSteps` empty, `activatableFor` empty, `completedFinalOutputs` empty) can only co-occur when no further progress is structurally possible. `activatableFor` is a stateless, rebuild-safe function of persisted completed-step state (pre-existing EDR-011 §1 comment). A step awaiting a signal moves the instance to a distinct `status: waiting`, which is excluded from the `Tick()` scan (`ListInstances(Status: running)`) before this code ever runs. A step with a pending retry remains in `CurrentSteps` — `project()` (`internal/engine/emit.go`) only removes a step from `CurrentSteps` on a *non-retrying* `StepFailed` — so a retry-in-backoff cannot present as `CurrentSteps == []`.

**Tests added:** `internal/engine/stall_test.go:TestStall_DeadEndBranchFailsInsteadOfWedging` — a 3-step workflow (`a`→`b` unconditional dead end, `a`→`c` conditional-but-never-true final target) run to terminal via the existing `runToTerminal` test helper. Confirmed the test fails against the pre-fix code (instance never reaches terminal within 200 ticks) and passes against the fix (instance reaches `failed` with `error.code = "stalled"`).

**Evidence:** `go test -run TestStall_DeadEndBranchFailsInsteadOfWedging -v ./internal/engine/...` passes; full `go test ./...` and `go test -tags integration ./test/integration/...` re-runs are green (see `REGRESSION_REPORT.md`).

---

## Change 3: Healthcheck now verifies database liveness (new defect, SEC-12)

**Files:**
- `internal/storage/sqlite.go` (new additive method)
- `internal/api/healthz.go` (fix)
- `internal/api/router.go` (wiring)
- `internal/api/router_test.go` (new regression test)

**Reason:** Confirmed `handleHealthz` was a static, dependency-free 200 response, exactly as SEC-12 described, unaddressed by the prior remediation session.

**What changed:**
1. Added `func (s *SQLiteStorage) Ping(ctx context.Context) error { return s.db.db.PingContext(ctx) }` — additive, does not touch the frozen 12-method `StoragePort` interface.
2. Changed `handleHealthz` from a bare `HandlerFunc` to a factory `handleHealthz(store core.StoragePort) HandlerFunc` that type-asserts `store` against a new additive `pinger` interface (mirroring the existing `cancellationStore`/`eventsPagedStore` additive-interface pattern in this codebase) and returns HTTP 503 `{"status":"unavailable"}` on a failed ping, HTTP 200 `{"status":"ok"}` otherwise (including when `store` doesn't implement `pinger` — graceful degradation, same fallback convention as the other additive slices).
3. `router.go`: `handleHealthz` → `handleHealthz(store)`.

**Tests added:** `internal/api/router_test.go:TestNewRouter_Healthz_PingFailureReturns503`, using a `deadPingStore{core.StoragePort}` wrapper whose `Ping` always errors, to confirm the 503 path without needing to actually kill a database connection. The pre-existing `TestNewRouter_Healthz` (happy path against a real, live SQLite-backed store) was left unchanged and still passes.

**Evidence:** `go build ./...`, `go vet ./...`, `go test ./internal/api/... ./internal/storage/...` all pass.

---

## Change 4: Stale frontend comment corrected (documentation-only, no behavior change)

**File:** `web/src/screens/instanceList.ts`

**Reason:** The comment explaining the client-side instance sort described the backend as ordering ascending ("oldest first") and framed the client-side sort as a workaround for that — a description that stopped being true once the prior session's SEC-03 fix changed the backend to `ORDER BY started_at DESC`. The sort code itself (by `updated_at`, independent of the backend's `started_at` ordering) was never incorrect and required no change.

**What changed:** Comment text only, updated to describe the current (post-fix) backend ordering and reframe the client-side sort as a secondary, independent ordering within the already-newest-first page rather than a correction for an inverted order.

**Evidence:** `node web/build.mjs` rebuilds cleanly (bundle regenerated at `cmd/awis-server/static/bundle.js`); no `.ts` compile errors.

---

## Full verification commands run this session

```
go build ./...                                    # exit 0
go vet ./...                                       # exit 0, zero warnings
go test -count=1 ./...                             # all packages pass (see REGRESSION_REPORT.md for the one flaky exception)
go test -tags integration -count=1 ./test/integration/...   # 100% pass
node web/build.mjs                                 # clean build, no TS errors
```

See `REGRESSION_REPORT.md` for full output and the flaky-test investigation.
