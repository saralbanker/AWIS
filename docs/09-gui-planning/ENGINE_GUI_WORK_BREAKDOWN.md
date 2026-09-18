# Engine → GUI Work Breakdown

Card-level work packages for milestones G0-G10. This is the granularity the rest of the
`docs/09-gui-planning/` corpus deliberately stops short of (it estimates at the
milestone level; see `GUI_MASTER_PLAN.md` §13). Every card below decomposes one
milestone into an implementable unit with an owner-agnostic acceptance test.

**ID convention.** `E-Gn-x` = engine/API-server cards (milestones G0-G4, engine and
in-module server work). `G-Gn-x` = GUI/product cards (milestones G5-G10, frontend and
GUI-adjacent backend work). This mirrors the two-subagent split used to produce this
document (Subagent A: engine/backend; Subagent B: GUI/product) and is preserved because
it maps directly onto how the two tracks can be staffed.

**Suggested agent fit (engine cards only).** This repository already defines
AWIS-specific implementation agents (`.claude/agents/`: `awis-builder`,
`awis-core-engineer`, `awis-verifier`, `awis-scribe`). Mechanical, additive cards (tests,
HTTP handlers wrapping already-correct engine calls, documentation corrections) fit the
`awis-builder`/Sonnet profile cleanly. `E-G2-2`/`E-G2-3` (the two EventLog/projection
write-transaction edits) and `E-G3-1`/`E-G3-2`/`E-G3-4` (the namespace read-path
unification, currently a silent-wrong-answer defect) are the correctness-critical class
`awis-core-engineer` exists for — see `ENGINE_GUI_TRANSITION_PROGRAM.md` §3 for the
model-allocation-policy tension this raises, which this document does not resolve. No
existing agent definition covers frontend/SPA work (G5-G10's UI cards); that is a gap
worth closing before G5 starts, not a blocker to this plan.

---

## G0 — Freeze close-out (rollup: ~1-1.5d + D4 decision)

Independently re-confirmed 2026-09-01 against the working tree (not just carried over
from the 2026-08-30 corpus):

| Check | Status | Evidence |
|---|---|---|
| B-31 fix present | **Yes, uncommitted** | `internal/validate/validate.go` diff, +143/-0; five new codes confirmed: `CodeStepTypeUnknown`, `CodeTriggerTypeUnknown`, `CodeSignalTimeoutAction`, `CodeRetryBackoff`, `CodeIntelligenceCapability` |
| D4 decided | **No** | `.github/workflows/` has zero matches for live/anthropic/schedule/secret |
| Stale root `awis` binary | **Still present** | 17,252,397 bytes, dated Aug 26 — predates the Aug 28 B-28 fix |
| System-test flake | **Still present** | `cmd/awis/system_test.go:404`, `10 * time.Second * raceScale` |

| ID | Title | Description | Effort | Deps | Acceptance | Parallel with | Path |
|---|---|---|---|---|---|---|---|
| E-G0-1 | Commit the B-31 remediation | Land the 18-file uncommitted diff (validate.go +143 lines, both scaffolds, `apps/oip/qg3_test.go`, `apps/oip/workflows/capture-decision.yaml`, `internal/dsl/testdata/capture-decision.yaml`, `docs/DSL.md`, `docs/PROVIDERS.md`, four Anthropic adapter/testdata files) | 0.5h | none | `git status --short` clean; `go test ./internal/validate/... ./internal/engine/...` green | all | critical (blocks G1) |
| E-G0-2 | D4 CI gate | Scheduled, secret-gated workflow running `TestLiveSmoke` (`live_test.go:27-83`), asserting the response `model` field | 3-4h + decision | **D4 (founder)** | Workflow runs on schedule, fails loudly on model drift, excluded from PR CI | all G0/G1 | critical to close G0 |
| E-G0-3 | Remove stale binary + guard | Delete `./awis`; add a `make build-fresh` target that live-audit workflows must use | 1h | none | `./awis` absent; `make build-fresh` reflects current source | all | deferrable, cheap |
| E-G0-4 | Fix system-test flake (R14) | Raise/derive the 10s deadline in `system_test.go:404` from real contention data | 2h | none | Passes under `go test ./cmd/awis/... -count=5` alongside `make verify` | all | deferrable, flagged (R15 neighbor) |
| E-G0-5 | Housekeeping | Prune 6 stale `worktree-agent-*` branches; correct "29 commits" wherever documented | 1h | none | `git branch -a` shows none | all | deferrable |
| E-G0-6 | STATE.md / main-merge reconciliation (process, not code) | Correct M10-M14 `STATE.md` rows; merge M15→M16→M17→`engine-hardening` into `main` or declare `engine-hardening` the new base; adjudicate M17 row 8 | ~1d, zero engineering | founder (not D1-D4) | `git merge-base --is-ancestor engine-hardening main` succeeds; `STATE.md` matches reality | all engineering cards | not on the G0-G4 engineering path, but ranked action #0 in the audit — see `ENGINE_GUI_DECISION_RECORD.md` |

---

## G1 — Contract hardening (rollup: ~2.6d, deps: G0)

| ID | Title | Description | Effort | Deps | Acceptance | Parallel with |
|---|---|---|---|---|---|---|
| E-G1-1 | Golden JSON snapshots on `ExecutionEvent` | Pin the 12 event types' payload field names (`internal/core/event.go`) | 4h | E-G0-1 | Deliberate field rename fails the test | E-G1-2..4 |
| E-G1-2 | Reflective `StoragePort` method-count pin | Assert `NumMethod()` equals today's count (`internal/core/ports.go`) | 2h | none | Adding/removing a method fails with a clear message | all |
| E-G1-3 | `GetWorkflow` not-found sentinel | Replace the bare `fmt.Errorf` at `internal/storage/sqlite.go:296` with a typed `ErrWorkflowNotFound`, update callers | 2h | none | `errors.Is` round-trips; callers unaffected | all |
| E-G1-4 | Fix `start.go`'s string-matched error check | Replace `strings.Contains(err.Error(), "already registered")` at `start.go:481` with a `*RegistrationError` assertion | 1h | none | Type assertion replaces substring check; idempotent-restart test still passes | all |
| E-G1-5 | Correct `docs/CLI_CONTRACT.md` | Fix the ≥11 documented divergences (nonexistent `export --format/--from/--to`, wrong `metrics`/`recall` shapes, nonexistent `start --config`, nonexistent namespace flags on `recall`/`rebuild-state`, `status --watch` 5s-vs-1s, `trace.trigger` always null) | 4h | none | Every documented flag/field verified against the built binary | all — **zero engine dependency** |
| E-G1-6 | CLI JSON golden snapshots + coverage lift | Snapshot tests for `status`/`list`/`trace`/`recall`/`metrics`; raises `cmd/awis` coverage from 47.1% (R15) | 1d | E-G1-1 pattern | — | E-G1-3/4/5 |

---

## G2 — Cursor + eventing (rollup: ~2.4d, deps: **D1**, G1)

| ID | Title | Description | Effort | Deps | Acceptance |
|---|---|---|---|---|---|
| E-G2-1 | Migration 0007: `state_changes` table | New file in `internal/storage/migrations/` (current tail: `0006_recall_fts.sql`), ADR-1 schema verbatim | 2h | **D1** | `make verify` runs it clean on fresh and existing-0006 DBs |
| E-G2-2 | Wire write into `AppendEvent` | Insert a `state_changes` row inside the existing transaction (`internal/storage/sqlite.go` ~line 94-156) | 3h | E-G2-1 | Every `AppendEvent` call produces one `kind='event'` row |
| E-G2-3 | Wire write into `UpsertInstance` | Same pattern (`sqlite.go:428+`) | 3h | E-G2-1 | A status-only update produces a `kind='projection'` row |
| E-G2-4 | `ReadChangesSince(ctx, since, namespace)` | New storage method behind a locally-declared interface + type assertion, matching the `cancellationStore`/`signalWaitStore`/`PluginStore` pattern already in the codebase | 3h | E-G2-1 | — |
| E-G2-5 | SDK wrapper | Expose `ReadChangesSince` on `sdk.Runtime` | 2h | E-G2-4 | — |
| E-G2-6 | Regression: parked instance produces a `state_changes` row | Reproduces `GUI_MASTER_PLAN.md` §3.1 exactly (submit `with-signal`, drive to `waiting`, assert the row exists) — not just "the migration runs" | 3h | E-G2-2, E-G2-3 | — |
| E-G2-7 | Regression: the other two non-evented transitions | Timeout-`continue` resumption (`signal_timeout.go:128-140`) and cancellation-requested (`cancel.go:118`) each produce a row | 3h | E-G2-3 | — |

**Purely D1-blocked** — no card starts before D1 is decided. E-G2-2/E-G2-3 parallelize
with each other once E-G2-1 lands; the rest are sequential-ish. Path: critical
(`G2 → G4 → G5`).

---

## G3 — Namespace resolution (rollup: ~1.6d, deps: **D2**, parallel with G2)

| ID | Title | Description | Effort | Deps | Acceptance |
|---|---|---|---|---|---|
| E-G3-1 | Shared namespace-predicate helper | Replace the three independent implementations (`export.go:158-163`, `history.go:75` inline, `metrics.go:86` inline — confirmed distinct) with one | 3h | **D2** | — |
| E-G3-2 | Retire the `"default"` overload | `""` becomes the wildcard; `"default"` becomes an ordinary literal everywhere | 2h | E-G3-1 | — |
| E-G3-3 | Add `--all-namespaces` flag | New explicit CLI flag; API's future `?all_namespaces=true` (E-G4-8) reuses it | 2h | E-G3-2 | — |
| E-G3-4 | `ReadEventRange` wildcard branch | `sqlite.go:~182` has no `namespace=""` special case today — confirmed by reading the query (always literal `WHERE namespace = ?`) | 2h | E-G3-2 | — |
| E-G3-5 | Upgrade/migration notes | Document the breaking-change disclosure for `history`/`metrics`/`export` output, per the B-26 commit's own precedent | 2h | none | Can be written early |
| E-G3-6 | Regression: `StepStats` no longer silently zeros | Reproduce the exact R2 scenario (`runtime_recall.go:112-119` calling `ReadEventRange` with a mismatched namespace) and assert correct results | 2h | E-G3-4 | — |

**Purely D2-blocked.** Path: critical to G4, fully parallel with G2 (disjoint files).

---

## G4 — API server (rollup: ~5.3d, deps: G2, G3, **D3**)

| ID | Title | Description | Effort | Deps | Notes |
|---|---|---|---|---|---|
| E-G4-1 | `cmd/awis-server` skeleton | `main.go`; embed via `rt.Start(ctx)` in a goroutine; loopback bind; graceful shutdown | 4h | D3 (soft) | Zero engine dependency — everything called already compiles out-of-module today |
| E-G4-2 | PID/lock-file guard (R7) | Prevent a second engine ticking the same DB | 3h | E-G4-1 | — |
| E-G4-3 | `internal/api` skeleton | Router, DTO conventions, error-mapping middleware (sentinels → 404/409/501) | 4h | none | Zero engine dependency |
| E-G4-4 | Read-only workflow routes | GET `/workflows`, GET `/workflows/{id}/{version}` | 3h | E-G4-3, E-G1-3 | 404 on missing workflow, not 500 |
| E-G4-5 | Read-only instance routes | GET `/instances` (`ListPaged`, already cursor-bounded), GET `/instances/{id}` (`Status` + wait-record wrapper for `signal_name`/`timeout_remaining_s`, currently omitted) | 4h | E-G4-3 | — |
| E-G4-6 | Event-history route + pagination | GET `/instances/{id}/events` off `ReadEvents`, adding limit+cursor (currently unbounded) | 3h | E-G4-3 | — |
| E-G4-7 | `/healthz` | — | 0.5h | none | Zero engine dependency |
| E-G4-8 | Mutation routes: submit + signal | POST `/instances` (`Submit`), POST `/instances/{id}/signal` (`Signal`) | 3h | E-G4-3; namespace-correctness needs G3 | — |
| E-G4-9 | Cancel route with `compensate` | POST `/instances/{id}/cancel`; requires an SDK fix first — `sdk/runtime_runner.go:127` hardcodes `compensate=false`, only the CLI bypasses it today | 4h (2h SDK + 2h route) | E-G4-3 | — |
| E-G4-10 | Registration + validate routes | POST `/workflows` (new-version-only), POST `/workflows/validate` returning the full structured `Issue{Code,StepID,Field,Position}` instead of the CLI's line+message-only shape (`workflow.go:43-51`) | 3h | E-G4-3 | — |
| E-G4-11 | SSE `/stream` | Tails `state_changes` from `?since=`, `Last-Event-ID` resume, poll interval tracking the engine tick | 1d | **G2 complete** | First real consumer of G2 |
| E-G4-12 | `/stats/steps` | Gated behind G3 (R2's fix point) | 2h | **G3 complete** | — |
| E-G4-13 | `/plugins` | `storage.ListPlugins`, in-module only (confirmed hard-blocked out-of-module by Go's internal-package rule) | 1h | E-G4-3 | — |

**Zero-engine-dependency cards, buildable today:** E-G0-3, E-G0-5, E-G1-5, E-G4-1,
E-G4-3, E-G4-7. E-G4-4/5/6 need only E-G1's contract to be pinned first, not new engine
code.

---

## G5 — Read-only GUI (Phases 1-2, plus a Phase 4 preview) (rollup: ~8-9d, deps: G4)

| ID | Title | Description | Effort | Deps | Acceptance | Parallel with | Phase |
|---|---|---|---|---|---|---|---|
| G-G5-1 | SPA shell & API client | Routing skeleton, typed client against `/api/v1` | 1d | G4 | Shell hits `/healthz`, routes resolve | — (foundation) | 1 |
| G-G5-2 | Definitions list/detail | `ListWorkflows`/`GetWorkflow` (FR-1.4) | 1d | G5-1 | Shows real registered defs, no mocks | G5-3,4,5 | 1 |
| G-G5-3 | Instance list, paged, namespace filter | `ListPaged` + explicit "all namespaces" toggle (FR-1.1, FR-1.5) | 1.5d | G5-1 | Correct for a namespace literally named `default` | G5-2,4,5 | 1 |
| G-G5-4 | Instance detail w/ wait info | Status + wait-record lookup: signal name, timeout remaining (FR-1.2) | 1d | G5-1, G4 wait-record wrapper | Waiting instance shows signal_name/timeout | G5-2,3,5 | 1 |
| G-G5-5 | Event timeline (static) | `ReplayInstance`, poll-refresh (FR-1.3) | 1d | G5-1 | Full ordered event list renders | G5-2,3,4 | 1 |
| G-G5-6 | SSE live wiring | Subscribe `/stream`, patch list/detail, `Last-Event-ID` reconnect (FR-2.1) | 2d | G2, G5-3/4 | Reconnect after drop replays no gaps | — | 2 |
| G-G5-7 | Live timeline honesty | Renders `waiting`/timeout-resume/cancel-requested; tick-quantized copy (FR-2.2-2.4) | 1.5d | G5-6 | `wait_signal` submit → visible `waiting` within 1 tick, no reload — the concrete B-a test | — | 2 |
| G-G5-8 | Read-only canvas renderer | Renders 6 edge types from a fetched `WorkflowDefinition`; positions from `Metadata.ui.positions` or auto-layout fallback; **no editing, no serializer dependency** | 2d | G5-2 | Fallback/compensation render as distinct relationships | G5-6,7 | 4 (preview) |
| G-G5-9 | Contract-drift CI check | Frontend types pinned to G1 golden snapshots | 1d | G1 | CI fails on `ExecutionEvent`/CLI-shape drift | — | 1 (cross-cutting) |

---

## G6 — Serializer + editor (Phase 4) (rollup: ~10-12d, deps: G5, B-c)

The largest single milestone in the whole program: the read side (`validate.Validate`,
`expr.Parse*`) is already pure and free; the write side has zero existing code
(`yaml.Marshal` is called nowhere in the repo today).

| ID | Title | Description | Effort | Deps | Acceptance |
|---|---|---|---|---|---|
| G-G6-1 | YAML key-order ADR | Decide ordering strategy for `Inputs`/`Outputs`/`Trigger.Config`/`Metadata` (all `map[string]any`, no `MarshalYAML`) — **addresses N4, must land before G6-2** | 1d | none — can start before G5 finishes | 3 successive saves of the same def produce byte-identical key order |
| G-G6-2 | YAML emitter core | `internal/dsl.Marshal`/`Render`, mirrors the parser's `yaml:` tags exactly for scalar/slice fields | 2d | G6-1 | `Parse(Marshal(def)) == def` for every existing `internal/dsl/testdata/*.yaml` |
| G-G6-3 | Map-field serialization | Applies G6-1's strategy to the four map-typed fields | 1.5d | G6-1, G6-2 | Round-trip preserves key order on those fields specifically |
| G-G6-4 | Round-trip regression suite | Parse→serialize→parse→diff over the full corpus incl. `apps/oip/workflows/*` | 1d | G6-2, G6-3 | Zero diffs; wired into `make verify` |
| G-G6-5 | Save-from-canvas API | Canvas JSON → `WorkflowDefinition` → serialize → `RegisterWorkflow` | 1d | G6-4, G4 POST route | Definition built only via canvas resubmits and matches hand-authored YAML |
| G-G6-6 | Editable canvas | Extends G5-8 with add/move/remove node & edge editing | 3d | G5-8 | Drag/connect produces a valid in-memory graph |
| G-G6-7 | 6-relationship edge editing | Distinct UI per relationship; fallback excluded from client cycle-check (mirrors `validate.go:26-28` exactly) | 2d | G6-6 | A cyclic-via-fallback graph is accepted client-side, matching the server |
| G-G6-8 | Inline validation | `POST /workflows/validate`, renders `Issue{Code,StepID,Field,Message,Position}` at the node/field | 1.5d | G6-6, G4 full-`Issue` route | Errors appear as-you-edit, not only on save |
| G-G6-9 | Expression squiggles | `expr.ParseTemplate`/`ParseCondition` byte offsets → inline markers | 1d | G6-6 | Malformed condition shows squiggle at correct offset |
| G-G6-10 | Layout persistence | Positions written to `Metadata.ui.positions` on drag, included in G6-5's save payload | 0.5d | G6-6, G6-5 | Reload preserves layout |
| G-G6-11 | Inputs-as-template-values UI | Avoids the `Step.Inputs` JSON-Schema naming trap | 1d | G6-6 | UI presents a value-template editor, not a schema form |
| G-G6-12 | E2E round-trip acceptance | Build in canvas → save → resubmit → compare runtime behavior to hand-authored equivalent | 1d | all above | Identical `Transition`/`Compensation`/`Fallback` sets — the PRD's exact AC |

Backend track (G6-1..5, ~6.5d) and frontend track (G6-6..11, ~7d, largely parallel once
G6-1 lands) converge at G6-12. **G6-1 is a hard predecessor nothing else in G6 can
skip** — N4's key-order risk must be designed in, not discovered mid-build.

---

## G7 — Palette + introspection (Phase 4) (rollup: ~3.5d, deps: G6)

| ID | Title | Description | Effort | Deps | Acceptance | Parallel with |
|---|---|---|---|---|---|---|
| G-G7-1 | `engine.runners` accessor | Exported `List()` over the `StepType`→handler dispatch map | 0.5d | none | Returns exactly the registered `StepType`s | G7-2,3 |
| G-G7-2 | `NativeRunner.handlers` accessor | Same pattern for native handlers | 0.5d | none | Matches compiled-in handler set | G7-1,3 |
| G-G7-3 | `plugin.Manager` accessor | Extends the existing `PluginPID` exception into a full `List()` | 1d | none | Reflects only plugins actually registered in this deployment | G7-1,2 |
| G-G7-4 | `GET /api/v1/palette` | Aggregates G7-1..3 into one node-type catalog | 0.5d | G7-1..3, G4 | Palette JSON matches live registry | — |
| G-G7-5 | Frontend palette component | Replaces G6-6's hardcoded step list with G7-4's live data | 1d | G7-4, G6-6 | Adding a plugin makes it appear without a frontend deploy | G8, G9, G10 |

---

## G8 — Observability (Phase 5) (rollup: ~9d, deps: G4 — re-scoped up from 4-5d)

N1 established that zero metrics/tracing infrastructure exists at all (not "StepStats
needs an incremental aggregate" — there is no infrastructure to build observability
*on*). This adds ~2d this breakdown accounts for explicitly (G-G8-1), matching
`REPOSITORY_TRUTH_AUDIT.md` §12 action #7's own re-scoping recommendation.

| ID | Title | Description | Effort | Deps | Acceptance | Parallel with |
|---|---|---|---|---|---|---|
| G-G8-1 | Minimal metrics substrate | Stand up `expvar` or equivalent — greenfield, not incremental (N1) | 2d | G4 | Tick duration, step-dispatch count, HTTP latency exposed | — |
| G-G8-2 | Incremental step-stats aggregate | Replaces `StepStats`'s 10-year full rescan (`runtime_recall.go:112-119`) with an aggregate fed off `state_changes` | 1.5d | G2 | Bounded-time query regardless of history depth | G8-3 |
| G-G8-3 | Cost/token index | `json_extract` + generated column/index on `execution_events.payload` for `event_type`, `tokens_used` | 1.5d | G4 | Query does not full-scan | G8-2 |
| G-G8-4 | Stats/cost endpoints | `GET /stats/steps`, `GET /costs` wiring G8-2/3 | 1d | G8-2, G8-3 | — | — |
| G-G8-5 | Tick-loop benchmark, 1K/10K instances | No benchmark exists above 510 instances (N6); quantify the O(n) scan ceiling before promising a refresh rate | 1.5d | none | Documented degradation curve | any |
| G-G8-6 | Cost/token dashboard UI | FR-5.1 views | 1.5d | G8-4 | "Tokens for workflow X, last 7d" returns in bounded time | — |

---

## G9 — AI generation (Phase 5) (rollup: ~8d, deps: G6, D4)

| ID | Title | Description | Effort | Deps | Acceptance | Parallel with |
|---|---|---|---|---|---|---|
| G-G9-1 | JSON Schema artifact | Generated from `core.WorkflowDefinition`, kept in sync by test | 1.5d | none | LLM prompt no longer needs prose/Go structs | G9-2 |
| G-G9-2 | Full-`Issue` API/CLI plumbing | Fixes `workflow.go:51` dropping `Code`/`Field`/`Position`; shares scope with E-G4-10/G6-8 if concurrent | 1d | G4 | `validate --json` and API both emit full `Issue` | G9-1 |
| G-G9-3 | NL-to-workflow endpoint | generate→validate→structured-error→retry loop using G9-1/G9-2 | 2.5d | G9-1, G9-2, D4 | Converges to a valid def within N retries on a test prompt corpus | — |
| G-G9-4 | Live-intelligence CI gate | Scheduled, secret-gated job asserting response `model` field (D4) | 1d | D4 decided | Stale model ID fails the job loudly | — |
| G-G9-5 | NL-generation UI | Inline repair-loop feedback, lands draft into G6's canvas | 2d | G9-3, G6-6 | Generated draft opens directly in the editor | — |

---

## G10 — Multi-user (Phase 5) (rollup: ~12.5d, deps: G5 only, parallel-safe with G6-G9)

| ID | Title | Description | Effort | Deps | Acceptance | Parallel with |
|---|---|---|---|---|---|---|
| G-G10-1 | Identity/session model | User table, session tokens, login endpoint — zero of this exists today | 3d | G5 | Login issues a valid session | — |
| G-G10-2 | Per-namespace authz | Role/ACL table + middleware on every `/api/v1` route | 3d | G10-1 | Unauthorized namespace access rejected server-side | — |
| G-G10-3 | Subprocess env scrubbing | Fixes `internal/runner/subprocess/subprocess.go` inheriting the full parent env (R13) to match the plugin manager's manifest-scoped pattern | 1d | none — independent | Subprocess step no longer sees `ANTHROPIC_API_KEY` | G6, G7, G8, G9 |
| G-G10-4 | Remote bind support | Removes loopback-only default (config-gated) | 1d | G10-1, G10-2 | Still defaults to `127.0.0.1` absent explicit config | — |
| G-G10-5 | Login/role-mgmt UI | Session UI, per-namespace role assignment | 2.5d | G10-1, G10-2 | — | — |
| G-G10-6 | Authz integration test | Second user scoped to `team-a` cannot read/act on `team-b` | 1d | G10-1..4 | Exact PRD Phase 5 AC, enforced server-side not UI-hidden | — |
| G-G10-7 | Security review checkpoint | Full review of authn/authz + G10-3 before non-operator access ships | 1d | all above | Sign-off gate | — |

---

## Parallelization summary

| Track | Cards that can start with zero engine/decision dependency today |
|---|---|
| Engine housekeeping | E-G0-3, E-G0-5, E-G1-5 |
| API skeleton | E-G4-1 (soft on D3), E-G4-3, E-G4-7 |
| GUI editor design | G-G6-1 (YAML key-order ADR — can start before G5 even finishes) |
| Independent security fix | G-G10-3 (subprocess env scrubbing — no dependency on any other G10 card) |
| Benchmarking | G-G8-5 (tick-loop benchmark — no dependency on G8's other cards) |

**Founder-decision-blocked (nothing else can substitute):** all of G2 (D1), all of G3
(D2), E-G0-2/G-G9-4 (D4), E-G4-1/2/3 soft-blocked on D3. **Process-blocked, not
technical:** E-G0-6 (STATE.md/main-merge) — see `ENGINE_GUI_DECISION_RECORD.md`.

---

*Companion to `ENGINE_GUI_TRANSITION_PROGRAM.md` (milestone structure, critical path,
acceptance gates), `ENGINE_GUI_DECISION_RECORD.md` (sequencing paths and tradeoffs), and
`FINAL_VERDICT.md` (direct answers). Card-level content produced by two independent
subagents (engine/backend and GUI/product tracks) against the same evidence corpus,
synthesized and reconciled by the supervisor.*
