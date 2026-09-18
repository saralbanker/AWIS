# AWIS Release Readiness Report

**Audit date:** 2026-09-06
**HEAD evaluated:** `engine-hardening` @ `8a87f70` (2026-09-05 15:04:53 +0530)
**Method:** Full re-derivation from current repository state — source, tests, git history, CI config. No prior audit's conclusions were assumed correct; every claim below is cited to a file:line, command output, or commit hash produced during this review.

---

## Verdict: **CONDITIONAL PASS**

Not a clean PASS, and not a hard FAIL. The reasons are unusual: the underlying *code* is in materially better shape than the repository's own document trail suggests, but the repository is not currently in a state that can be released at all, for reasons that are procedural rather than technical. Every blocking item below is closeable in hours, not weeks — none require a rewrite.

Two different systems are being evaluated here and they are in different states:

| Surface | State | Verdict |
|---|---|---|
| CLI engine (`cmd/awis`, `internal/core`, `internal/engine`, `internal/storage`, `internal/plugin`, `internal/runner`, `internal/signal`, `internal/intelligence`) | Committed, tested, CI-covered | **CONDITIONAL PASS** (two architectural items should be escalated, see `RELEASE_BLOCKERS.md`) |
| HTTP API + Web GUI (`internal/api`, `internal/buildinfo`, `cmd/awis-server`, `web/`) | **Not committed to git on any branch, ever** | **FAIL as a release candidate** — not because the code is bad, but because it does not exist in version control |

---

## 1. Architecture vs. implementation consistency

The canonical architecture — Step + EventLog + IntelligencePort + StoragePort, pull-based execution — is real, not aspirational prose:

- `core.StoragePort` — `internal/core/ports.go:11-39`
- `core.IntelligencePort` — `internal/core/ports.go:44-63`
- `core.Step` / `core.StepType` — `internal/core/step.go:6-53`
- `core.ExecutionEvent` (EventLog, append-only, monotonic `SequenceNum` per instance) — `internal/core/event.go:12-32`
- Pull-based execution confirmed: `internal/engine/tick.go:20-53` — `Run()` drives a `time.NewTicker` at a 100ms default (`internal/engine/engine.go:21-23`), each tick polls `ListInstances(status=running)` and processes them. `internal/engine/doc.go:1-4` states this explicitly.

The plugin system (`internal/plugin`), subprocess runner (`internal/runner/subprocess`), and signal subsystem (`internal/signal`) are sometimes described as scope creep beyond the "four abstractions" pitch — they are not. Each maps to an explicit PRD requirement (§10.6 Plugin System / FR-PS-01..10, TC-2, `wait_signal` step config at `core.WaitConfig`, `internal/core/step.go:75-82`) and each carries a `doc.go` citing the relevant TDS section. The README's four-abstraction framing is a simplification for newcomers, not a stale description.

**Gap:** the canonical architecture and PRD documents were last substantively written July 2–3, 2026 and have not been amended since (see §4, below, and `DOCUMENTATION_DIVERGENCE_REPORT.md`). They predate the GUI/API entirely and, more importantly, **explicitly scope the HTTP API as "V2" work and the web dashboard plus authentication as "V3" work.** Nothing in the canonical corpus authorizes shipping either in a V1 Beta. This is a governance gap, not a code gap — see `RELEASE_BLOCKERS.md` item 2.

## 2. Documentation vs. implementation consistency

Covered exhaustively in `DOCUMENTATION_DIVERGENCE_REPORT.md`. Headline: the repository root carries 16+ self-generated, mutually-inconsistent audit/verdict documents produced across three separate bursts (Aug 30, Sep 1, Sep 3, Sep 5 ×2), all untracked in git, several reaching different verdicts on the same defects using four incompatible ID-naming schemes with no cross-reference. The single most important fact — that `internal/api`/`web`/`cmd/awis-server` are uncommitted — was independently rediscovered at least four times over a week and never once fixed.

## 3. GUI vs. backend consistency

**Clean.** Verified by direct build/typecheck and route-by-route contract comparison:

- All 5 claimed screens (Instance List, Instance Detail, Workflow List, Workflow Detail, Event Timeline) are real implementations — genuine `fetch` calls, loading/error/empty states, polling — not stubs. E.g. `web/src/screens/instanceList.ts:83-146`, `web/src/screens/eventTimeline.ts:47-87` (cursor pagination via `next_cursor`).
- Every route the GUI calls (`web/src/apiClient.ts:57-111`) matches a registered backend route (`internal/api/router.go:33-41`) with field-exact JSON shapes on both sides.
- `npx tsc --noEmit` → 0 errors. `npm run build` → succeeds (43.9kb bundle). `go build ./cmd/awis-server/... ./internal/api/...` → exit 0.
- The embedded static bundle (`cmd/awis-server/static/`, itself also uncommitted) was rebuilt fresh from `web/src` during this audit and diffed byte-identical against the committed-in-working-tree version — it is current, not stale.
- One cosmetic gap: `core.Step` sends `inputs`/`outputs` JSON Schema (`internal/core/step.go:16-19`) that the GUI's `Step` type (`web/src/types.ts:47-65`) never declares or renders. Harmless at runtime; a coverage gap, not a defect.

## 4. API contract vs. implementation

No mismatches found in path, method, field name/casing, or pagination handling across all 6 registered routes (`/healthz`, `/info`, `/workflows`, `/workflows/{id}/{version}`, `/instances`, `/instances/{id}`, `/instances/{id}/events`). The API is entirely **read-only (GET-only)** — confirmed via `internal/api/router.go`. This matters for defect adjudication: see item 6 below.

## 5. Runtime behavior vs. documented behavior

- `go build ./...` and `go vet ./...` — clean.
- `go test ./... -count=1` — one failure on first run: `TestSystemRehearsalInitStartSubmitTrace` (`cmd/awis/system_test.go:434`), "did not reach a terminal status within 10s." Re-run in isolation: **PASS in 11.13s.** This is a load-induced flake (the audit was running under heavy concurrent load from parallel investigation agents), not a functional regression — but the fact that solo execution takes 11.13s against a 10s internal timeout means the test's margin is too thin for any loaded CI runner. Recommend widening the timeout, not shipping as-is.
- All other 20 test packages passed clean, including `internal/storage` (37.4s, the heaviest package — SQLite + FTS + migrations), `internal/api` (0.317s), `internal/intelligence/adapters/anthropic`.

## 6. Verification of the eight "closed defect" claims

Each independently re-derived from current code (not from any prior report):

| Claim | Verdict | Evidence |
|---|---|---|
| B-31 silent-defaults validation | **CONFIRMED FIXED** | `e75c1f1` adds 5 previously-unenforced validator rules; `internal/validate/validate.go:114-522` implements them with explicit "no documented default" rationale in comments. |
| B-31 "reopened across API surface" (claimed in `RELEASE_AUDIT_REPORT.md`) | **REFUTED** | `internal/api` never calls `validate.*` (`grep -rn "validate\." internal/api/*.go` → zero hits) and is GET-only — there is no write path for this defect class to recur on. The prior report's claim is incorrect. |
| Cancellation durability | **CONFIRMED FIXED** | Migration `internal/storage/migrations/0007_cancellation_intent.sql` adds `cancellation_reason`/`cancellation_compensate` columns, explicitly tagged RC-2, "Persist cancellation reason and compensation intent across process restarts." |
| Crash recovery / state hydration | **CONFIRMED FIXED** | `internal/engine/hydrate.go:31` — dedicated `hydrate()` function invoked on engine start. |
| Dead-end workflow stalls | **CONFIRMED FIXED** | Dedicated detection logic in `internal/engine/tick.go` with a dedicated regression suite, `internal/engine/stall_test.go`. |
| Subprocess environment leakage | **CONFIRMED FIXED** | `internal/runner/subprocess/subprocess.go:129` builds environment explicitly via `buildSubprocessEnv(sc)` rather than inheriting the parent process's environment. |
| Health endpoint correctness | **CONFIRMED FIXED** | `internal/api/healthz.go:26-33` — comment states "Before SEC-12 this returned a [richer body]," now a deliberately pure liveness check. |
| Dashboard/event ordering | **CONFIRMED FIXED** | `internal/storage/sqlite.go:791,820` — explicit tie-breaker `ORDER BY started_at DESC, instance_id DESC` with a comment noting "started_at alone is not a [unique sort key]," confirming a prior non-deterministic-ordering bug was closed with a real tie-breaker. |
| Anthropic adapter corrections | **CONFIRMED FIXED** | `internal/intelligence/adapters/anthropic` has a dedicated commit history (`3b12689` initial adapter, `8a87f70` "correct a model-ID regression"); package tests pass (`ok ... adapters/anthropic 0.043s`). |

All eight hold up. This is genuinely good evidence — the hardening work described in the commit log is real, not narrative.

## 7. Migration state

`internal/storage/migrations/0001` through `0007` are sequential, present, and each maps to a real schema need (core execution → domain events → signals → audit → plugins → recall FTS → cancellation intent). No gaps, no out-of-order numbering.

## 8. Branch state

`engine-hardening` is 62 commits ahead of `main` but also **6 commits behind** `main` (diverged after merge-base `827a0842`). The 6 main-only commits are CI/toolchain hygiene (macOS matrix, gofmt, fetch-depth) plus `f6aa755` ("M10-M14"), which is a merge commit whose content is already common ancestry with `engine-hardening` — safe. The one real risk: both branches independently rewrote `.github/workflows/ci.yml` from the same base and will produce a genuine textual merge conflict requiring manual reconciliation, not silent data loss. See `RELEASE_BLOCKERS.md`.

## 9. Hidden blockers

The uncommitted GUI/API/static tree (§10) is the dominant one. Secondary: CI has zero coverage of that same code (it can't build/test what isn't checked in) — confirmed via `grep -iE 'awis-server|web/' Makefile ci.yml` returning nothing. No Engineering Decision Record governs either the uncommitted-tree situation or the main/engine-hardening reconciliation plan.

## 10. The headline finding

`internal/api/`, `internal/buildinfo/`, `cmd/awis-server/` (including its `static/` embed directory), and the entire `web/` GUI tree are **untracked in git on every branch and tag, with zero history — confirmed via `git log --all --oneline -- <path>` (empty for all four) and an exhaustive `git rev-list --all | git ls-tree -r` scan across every commit on every branch (empty).**

This is not cosmetic. A tracked, committed file — `cmd/awis/main.go` — has already been modified in the working tree to import `internal/buildinfo` (`git diff cmd/awis/main.go`). Committing the current modifications to `main.go` and `start.go` without also committing `internal/buildinfo/` would break the build for every other clone of this repository. A worktree checkout of HEAD alone builds fine today only because the *modifications* to those two tracked files are themselves still uncommitted.

**If the founder froze development today, `git clone` on any machine other than this working directory would produce a repository with no GUI, no HTTP API, and two source files (`cmd/awis/main.go`, `cmd/awis/start.go`) silently reverted to a pre-refactor state relative to what's on disk right now.** That is the central fact this entire review turns on.

---

## Bottom line

The CLI/engine core is solid, well-tested, and its hardening claims check out under direct re-verification. The GUI/API layer built on top of it is also, on its own technical merits, solid — no contract bugs, clean builds, real screens. But none of that second layer is a releasable artifact yet, because it isn't in version control, was never authorized against the canonical V1 scope, and has never been exercised by CI. See `RELEASE_BLOCKERS.md` for the precise, short list of what closes this gap.
