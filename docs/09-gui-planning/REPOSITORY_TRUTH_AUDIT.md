# Repository Truth Audit — GUI Readiness

Date: 2026-08-30. Basis: `engine-hardening` @ `f004f4f` + the same uncommitted
B-31-class remediation present throughout this doc set (confirmed unchanged —
`git status --short` still shows the same 18 files). This is not a new plan; it is
an adversarial, falsification-first re-check of the existing plan set
(`GUI_MASTER_PLAN.md`, `GUI_PRD.md`, `GUI_ARCHITECTURE.md`, `GUI_ROADMAP.md`,
`GUI_DEPENDENCY_MAP.md`, `GUI_COMPREHENSIVE_SUMMARY.md`), commissioned specifically
to try to break those documents' conclusions rather than extend them.

**Method.** Six investigation threads ran against the current tree: one direct
(this session, engine scheduler/trigger/plugin/observability/recovery surfaces) and
four parallel forks (AI integration, DSL/authoring, scalability/concurrency, and a
dedicated falsification pass checking 12 additional load-bearing citations across
all six existing documents). Every finding below is labeled **VERIFIED FACT**
(command run or exact code read), **LIKELY TRUE** (inferred from confirmed facts,
not independently executed), or **UNPROVEN** (gap identified, no data exists to
resolve it either way). Evidence hierarchy applied throughout: runtime/test
execution > source code > prior documentation > prior reports.

**Headline result: nothing in the existing five-document plan was disproven.**
Twelve additional citations were spot-checked by the falsification fork
specifically hunting for errors (exact line numbers, exact test names, "zero X" /
"only Y" claims) and all twelve confirmed. The one previously-known error (the
false B-11-traceability claim) is already corrected in `GUI_MASTER_PLAN.md` §1.3
and stays corrected — no second error was found. Six genuinely new findings
surfaced, all additive to the existing risk register, none contradicting it.

---

## 1. Executive Summary

AWIS's core engine is substantially complete and independently re-verified sound:
`make verify` and `make integration` both pass on the current tree; zero open
CRITICAL/IMPORTANT defects; recovery-from-restart is fully tested and passing (6/6
hydrate/restart tests); AI-step failure handling is more robust than the existing
docs credit (bounded retry, no hang paths). The gap to a GUI platform is not "the
engine is unfinished" — it is "the engine has no HTTP surface, no metrics/tracing
infrastructure, and no authorization model," each a build-from-zero addition, not
a fix. This audit found no reason to change the existing plan's direction; it adds
seven new items to the risk register (§7) and confirms the standing conclusion. One
of those seven (N7) is procedural rather than technical: this whole evaluation is
against `engine-hardening`, a branch `main` is 60 commits behind and `STATE.md`
never recorded — see §7 and `GUI_MASTER_PLAN.md` §1.5.

## 2. Engine Completion Assessment

No single "percent complete" number is defensible without a fixed scope — so two
numbers, by scope:

| Scope | Estimate | Basis |
|---|---|---|
| **Core workflow engine** (parse, validate, execute, retry, signal, compensate, replay, recover) | **~90–95%** | Zero open CRITICAL/IMPORTANT defects; 81.0% coverage in `internal/engine`, 75.5% in `internal/storage`, 96.9% in `internal/validate` (all re-run this session); 21 binary-level integration tests, all passing; 6/6 recovery tests passing (`TestHydrate_*`, `TestRestart_*`) |
| **GUI-enabling layer** (HTTP API, event streaming, metrics/observability, authz) | **~10–15%** | Zero `net.Listen`/`http.Server`/`ListenAndServe` anywhere (VERIFIED FACT, re-confirmed); zero Prometheus/OpenTelemetry/expvar anywhere (VERIFIED FACT, new this pass); zero user/role/session/tenant concept anywhere (VERIFIED FACT); the ~10-15% credit is for what *is* reachable — `sdk.Runtime` is genuinely embeddable and compiles clean out-of-module |

`cmd/awis` itself sits at 47.1% coverage — the lowest of any core package, and
exactly the package a GUI's CLI/JSON contract tests will live beside (re-confirmed
this session, matches prior figure exactly).

## 3. GUI Readiness Assessment

Composite: **foundation ready, live/collaborative layer not built.** Scored
against the ten capabilities the brief named:

| Capability | Status | Evidence |
|---|---|---|
| Read-only historical API | Buildable today, zero blockers | `Runtime.ListPaged`, `Status`, `ReplayInstance` all resolve out-of-module (re-confirmed) |
| Live event stream | **Blocked** | No global cursor; `waiting`/timeout-resume/cancel-requested are non-evented (VERIFIED, pinned by passing tests `TestRebuildPreservesWaitingStatus`/`TestRebuildPreservesCancellationRequested`) |
| WebSocket/SSE support | Not built, not blocked | Zero network surface exists at all; SSE recommended and unimplemented |
| Observability | **Materially worse than the existing docs state** | Not just "StepStats needs work" — there is no metrics or tracing library of any kind anywhere in the repository (VERIFIED FACT, new finding, see §7) |
| Editor support (parse/validate) | Ready | `validate.Validate` pure and structured; expression grammar is closed with no function registry (narrower/easier than assumed — VERIFIED FACT, DSL fork) |
| Editor support (save) | **Blocked** | Zero `yaml.Marshal` calls repo-wide (re-confirmed); new risk: no key-order preservation strategy exists for the three `map[string]any` fields that will need one (§7) |
| Validation support | Ready | 21 structured codes, all enum gaps closed (re-confirmed, DSL fork found zero additional gaps beyond B-31's five) |
| Runtime introspection | **Mostly missing** | No `List*` on handlers/plugins/capabilities; one minor exception found this pass: `plugin.Manager.PluginPID(name)` exists |
| Multi-user / authz | **Zero** | Confirmed again — zero hits for tenant/role/user/session/permission |
| AI integration for GUI | **Sounder than documented, but has an untested seam** | Failure handling verified robust (bounded retry, typed StepError, no hang); budget enforcement has no engine-level integration test (§7, new) |

## 4. Verified Remaining Engine Work — Mandatory Before GUI Dominance

Unchanged from the existing roadmap's G0–G2, independently re-confirmed sound this
pass (nothing here was weakened or strengthened by the new findings):

1. **G0 — Freeze close-out.** B-31 is fixed (uncommitted); D4 decision outstanding;
   stale root binary hygiene; `TestSystemRehearsalInitStartSubmitTrace`'s hardcoded
   `10 * time.Second * raceScale` deadline re-confirmed this pass
   (`cmd/awis/system_test.go:404`).
2. **G1 — Contract hardening.** No golden-snapshot test pins `ExecutionEvent`
   shape; no reflective test pins `StoragePort` at its method count.
   `docs/CLI_CONTRACT.md` divergence re-confirmed (export flags documented but
   absent from the binary; `status --watch` documented as 5s, implemented as 1s —
   both re-verified directly this session).
3. **G2 — `state_changes` migration.** This remains the one true hard blocker for
   a live view. Re-confirmed: no `state_changes` table exists; migrations stop at
   `0006_recall_fts.sql`.

**Everything else is additive and does not gate GUI *dominance*** — it gates
individual *phases* (G3 namespace before Phase 3, G6 serializer before Phase 4,
G10 authz before multi-user), not the transition itself.

## 5. Optional Engine Improvements — Valuable, Non-Blocking

- Budget-enforcement engine-level integration test (new finding, §7) — currently
  only unit-tested at the intelligence-package boundary.
- Tick-loop scaling data — no benchmark exists above ~510 instances; the O(n)
  sequential-instance-scan mechanism is confirmed, its actual breaking point is
  not (new finding, §7).
- Concurrent-submit throughput benchmark — none exists at any scale; only
  correctness (B-30/`SQLITE_BUSY`) is tested, never throughput.
- `plugin.Manager` and native/plugin handler `List*` accessors (G7) — confirmed
  still absent except the one narrow `PluginPID` exception found this pass.
- Map-key-order-safe YAML emission strategy for `Step.Inputs`/`Outputs`/
  `Trigger.Config` — needs deciding *before* G6 is implemented, not during
  (new finding, §7).

## 6. GUI Blockers — Verified Only

| Blocker | Status | Evidence |
|---|---|---|
| B-a: non-evented state transitions | **VERIFIED, still real** | `signal.go:149-179`, `signal_timeout.go:128-140`, `cancel.go:118`; pinned by two passing regression tests re-run this session |
| B-b: no global event cursor | **VERIFIED, still real** | No `state_changes` table; migrations end at 0006 |
| B-c: no YAML serializer | **VERIFIED, still real** | Zero `yaml.Marshal` calls repo-wide, re-confirmed |
| B-d: namespace `"default"` overload | **VERIFIED, still real** | `namespacePredicate` logic re-read and matches prior description exactly |
| No authz | **VERIFIED, still real** | Zero tenant/role/user/session hits, re-confirmed |
| No metrics/tracing infrastructure | **VERIFIED, newly quantified as absolute-zero, not partial** | Zero Prometheus/OTel/expvar anywhere in the repo (new this pass — previously the docs framed this as "StepStats needs work," which understates it) |

## 7. New Findings Not Previously Documented

All six are additive to the existing risk register — none require revising an
existing conclusion.

| ID | Finding | Severity for GUI | Label |
|---|---|---|---|
| N1 | **No metrics/tracing library exists anywhere** (Prometheus, OpenTelemetry, expvar all absent) — broader than the existing docs' framing of "StepStats needs an incremental aggregate." There is no infrastructure to build observability *on*, only raw SQLite tables. | Medium — affects G8 scoping | VERIFIED FACT |
| N2 | **AI steps have no intermediate status.** A step goes `StepStarted` → (blocking call, up to ~3.5s across 3 retries) → `StepCompleted`/`StepFailed`, with no "waiting on provider" sub-state and no way to surface partial cost. A live GUI can only show "running" for the whole call. | Low-Medium — cosmetic but visible in Phase 2 | LIKELY TRUE |
| N3 | **Budget enforcement (`context_budget`) has no engine-level integration test** — only intelligence-package unit tests exercise it. A future refactor could silently break the wiring between `Submit`/step execution and `enforceBudget` with nothing catching it. | Medium — silent-regression risk | VERIFIED FACT |
| N4 | **YAML map-key-order round-trip risk for G6.** `Step.Inputs`, `Step.Outputs`, `Trigger.Config` are `map[string]any` with no custom `(Un)MarshalYAML`. A naive serializer will reorder keys on every save — spurious diffs, false "modified" flags in a GUI dirty-check. Must be designed for *before* G6 starts, not discovered during it. | Medium — G6 scoping risk | LIKELY TRUE (confirmed types + confirmed absence of ordering safeguard; not independently round-trip-tested) |
| N5 | **`dsl.Discover()` assumes one flat directory** (`<dir>/workflows/*.yaml`), no nested folders. A GUI workflow browser organized into folders (n8n-style) needs new backend support this constraint doesn't currently offer. | Low — Phase 1/4 scoping input | VERIFIED FACT |
| N6 | **Tick loop is O(n) sequential across running instances** (not parallelized at the instance level, though step dispatch *within* one instance is). No benchmark exists above 510 instances (from `TestRebuild100K`, which is a rebuild-cost test, not a tick-cost test). The actual scale at which tick freshness degrades is unmeasured. | Medium — unknown scaling ceiling for a live dashboard | VERIFIED FACT (mechanism) / UNPROVEN (threshold) |
| N7 | **This entire plan is evaluated against a branch (`engine-hardening`) that is not `main` and was never told to `STATE.md`.** `main` has M10–M14 only (`git merge-base --is-ancestor m17-full-cli-init main` fails); it is missing M15/M16/M17 and all 30 engine-hardening defect-fix commits — including data-loss-class fixes (RC-A: in-memory-only durable state lost on restart). `STATE.md`, even on `main`, still shows M10–M14 as unmerged (stale — they've been in `main` since `f6aa755`) and shows M17 stuck in `C-VERIFY` with an unresolved row needing "an actual CE/founder call." A GUI effort branched from `main` today would inherit every defect this program just closed. See `GUI_MASTER_PLAN.md` §1.5. | **High — precondition, not a blocker** — zero engineering effort, but "build against the engine" is ambiguous until resolved | VERIFIED FACT |

## 8. Invalidated / Weakened Concerns

Two items the existing docs implicitly worried about turn out better than assumed:

- **Autocomplete/variable-scope gap is smaller than implied.** `GUI_ARCHITECTURE.md`
  §8.2 frames "no variable enumeration for autocomplete" as needing to catalog an
  open-ended expression surface. It doesn't: the expression grammar has **no
  function registry at all** — only comparison operators, boolean connectives, and
  path lookups (`internal/expr/corpus/corpus.go` explicitly marks function-call
  syntax `PROHIBITED`). An editor only ever needs to enumerate `workflow.*`/
  `steps.<id>.outputs.*`/`event.*` paths — a materially smaller problem than
  "catalog a function surface." **VERIFIED FACT.**
- **AI failure handling is sounder than the existing docs credit.** Retry is
  correctly bounded (production code defaults to 3 attempts via a constructor
  guard, not silently zero); a permanently-failing AI step always resolves to a
  typed `StepError`, never a hang; `classify`/`embed` capability gaps are
  explicitly, honestly documented in-code rather than silently defaulting (unlike
  the B-31 defect class). **VERIFIED FACT.**

No previously-asserted blocker was found to be a documentation artifact rather
than real engineering work — every item in §6 is real.

## 9. Architecture Risks — Evidence-Backed Only

Carried forward from `GUI_MASTER_PLAN.md` §11, re-confirmed, plus N1–N6 above.
Nothing here contradicts the existing architecture's decisions (D1–D4, ADR-1
through ADR-4) — every new finding is scoping input for milestones already on the
roadmap (G2, G6, G7, G8), not a reason to redesign any of them.

## 10. AI System Readiness

**Better than the existing plan credits on correctness; one real test gap.**
Two adapters exist (Anthropic, Null) — no local-model adapter despite a `local`
routing enum value existing in `router.go:19` with zero adapters registered
against it (VERIFIED FACT — a "looks wired, isn't" gap worth flagging if local-model
support is ever promised). Retry, budget rejection, and capability-unavailable
paths are all correctly bounded and typed. The one real gap: budget enforcement
lacks an engine-level regression test (N3). Model IDs are current
(`claude-sonnet-5`); the live-CI-gate decision (D4) remains the founder's to make,
unchanged from the existing plan.

## 11. Fastest Path to GUI — Phase by Phase

Unchanged from `GUI_ROADMAP.md`, independently re-confirmed sound. No new finding
in this audit shortens or lengthens the critical path (G0→G1→G2→G3∥G4→G5, ~3–4
weeks to a live GUI). N3, N4, and N6 add scope *inside* existing milestones (G8,
G6, G8 respectively) rather than adding new milestones or reordering existing
ones.

- **Read-only dashboard:** no new blocker found. Buildable now.
- **Visual workflow editor:** N4 (key-order round-trip) must be designed into
  G6's serializer from the start — added as explicit G6 scope, not a new
  milestone.
- **Full n8n-style platform:** N1 (zero observability infra) means G8 is larger
  than "add an aggregate query" — it is "stand up observability infrastructure
  from nothing." Re-scope G8's estimate accordingly if precision matters before
  committing a date.

## 12. Recommended Next Actions

Ranked by impact / effort / risk:

| # | Action | Impact | Effort | Risk if skipped |
|---|---|---|---|---|
| 0 | Reconcile `STATE.md` and merge M10→M17→`engine-hardening` into `main` (or explicitly declare `engine-hardening` the new base) (N7) | High | Process only, ~1 day | "The engine" stays ambiguous; a GUI effort branched from `main` inherits 25+ closed defects, incl. data-loss-class |
| 1 | Land G0 (freeze close-out) — mostly done already | High | ~1 day | Freeze credibility gap persists |
| 2 | Decide D1–D4 (founder decisions, unchanged asks) | High | Decision only | Blocks G1–G2 entirely |
| 3 | Land G1 (contract hardening) + G2 (`state_changes`) | Critical | ~5 days | Live GUI ships lying about `waiting` state |
| 4 | Add an engine-level budget-rejection integration test (N3) | Medium | ~1 day | Silent regression risk in AI cost control |
| 5 | Decide the YAML key-order strategy for G6 before writing the serializer (N4) | Medium | Design decision, ~0.5 day | G6 ships with spurious-diff bugs discovered late |
| 6 | Add a tick-loop benchmark at 1K/10K running instances (N6) | Medium | ~1–2 days | Live dashboard freshness ceiling stays unknown until a real deployment hits it |
| 7 | Stand up minimal metrics (even just `expvar` or a `/metrics` counter set) as part of G8 scoping, not after (N1) | Medium | Re-scope G8, +2–3 days | G8 estimate is understated as currently written |

## 13. Repository Truth Appendix

Full evidence trail lives in `GUI_MASTER_PLAN.md` (original claims, all
independently re-verified this session and in the prior session) and in this
document's inline citations above. Reproduction commands for the highest-value
checks:

```
make verify && make integration
go test ./internal/storage/... -run TestRebuild100K -v
go test ./internal/engine/... -bench BenchmarkSignalReceiptToResumedStep -benchtime=200x
go test ./internal/engine/... -run "TestHydrate|TestRestart" -v
go test ./internal/dsl/... ./internal/expr/...
go test -cover ./cmd/awis/... ./internal/engine/... ./internal/storage/... ./internal/validate/...
grep -rln "net/http\|ListenAndServe\|net.Listen(" --include="*.go" . | grep -v _test.go
grep -rln "prometheus\|opentelemetry\|otel\|expvar" --include="*.go" . | grep -v _test.go
```

All commands above were run against HEAD `f004f4f` plus the uncommitted B-31
remediation during this audit; all results are reproducible on a clean checkout of
the same tree state.
