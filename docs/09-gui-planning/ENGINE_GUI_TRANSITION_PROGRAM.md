# Engine → GUI Transition Program

Status: **executable**. This is the synthesis document: it converts the evidence and
decisions already established in this folder (`GUI_MASTER_PLAN.md`,
`GUI_DEPENDENCY_MAP.md`, `GUI_ARCHITECTURE.md`, `GUI_ROADMAP.md`, `GUI_PRD.md`,
`REPOSITORY_TRUTH_AUDIT.md`) into one program document with an exact critical path,
milestone structure, and acceptance gates. It does not re-derive any finding — it cites
them. Where this document adds new material, it is card-level decomposition (see
`ENGINE_GUI_WORK_BREAKDOWN.md`) and cross-document reconciliation, not new investigation.

**Basis and freshness.** The underlying corpus is dated 2026-08-30, evaluated against
`engine-hardening` @ `f004f4f` plus an uncommitted B-31 remediation. This document was
assembled 2026-09-01. Before synthesis, the supervisor independently re-confirmed the
corpus's highest-load-bearing claims directly against the current tree (Priority-1
evidence): zero `state_changes` table or references, migrations still end at `0006`,
zero `yaml.Marshal` calls repo-wide, zero network surface (`ListenAndServe`/`net.Listen`/
`http.Server`/`grpc.NewServer`), zero metrics/tracing infrastructure, zero
tenant/role/authz hits, and the B-31 validator fix (`CodeSignalTimeoutAction` and
siblings) present in the uncommitted diff. A dedicated engine-side subagent separately
re-confirmed, same session, that G0's four checklist items are unchanged: B-31 fix still
uncommitted, D4 still undecided, the stale root `awis` binary still present (dated Aug
26, predates the Aug 28 B-28 fix), and the `TestSystemRehearsalInitStartSubmitTrace`
10-second-deadline flake still present at `cmd/awis/system_test.go:404`. **Nothing has
moved since the corpus was written.** It is current truth as of this document.

---

## 1. Executive summary

The engine is done; the GUI-enabling layer does not exist yet. Concretely:

- **Core workflow engine:** ~90-95% complete by scope (parse/validate/execute/retry/
  signal/compensate/replay/recover). Zero open CRITICAL/IMPORTANT defects.
  `make verify` and `make integration` both green (`REPOSITORY_TRUTH_AUDIT.md` §2).
- **GUI-enabling layer:** ~10-15% complete by scope. There is no HTTP surface, no
  metrics/tracing infrastructure, and no authorization model anywhere in the repository
  — each a build-from-zero addition, not a fix (`REPOSITORY_TRUTH_AUDIT.md` §2, §6).
- **The gap is architectural, not a backlog of bugs.** Every identified blocker
  (B-a through B-f) is additive: a new table, a new method behind a type assertion, a
  new accessor. Nothing requires touching the G1-frozen surfaces (12 event types, 12
  `StoragePort` methods) (`GUI_MASTER_PLAN.md` §14, answer 6).
- **Frontend and read-only history views can start immediately.** The live view needs
  one additive migration (G2, gated on founder decision D1) first. The visual editor's
  save path needs a YAML serializer that does not exist yet (G6, the largest single
  milestone in the whole roadmap).
- **One precondition is procedural, not technical, and outranks everything else in
  cost-to-value:** `main` is 60 commits behind `engine-hardening` and does not contain
  the 30-commit defect-closure program this entire plan is built on, including
  data-loss-class fixes. `STATE.md` has not been told. A GUI effort branched from `main`
  today would silently re-inherit closed defects. Fixing this costs ~1 day of founder
  process time and zero engineering (`GUI_MASTER_PLAN.md` §1.5, `REPOSITORY_TRUTH_AUDIT.md`
  N7, ranked as action #0 — higher priority than G0 itself).

---

## 2. The exact critical path

```
Phase A (process, ~1d, zero engineering)
  Reconcile STATE.md; merge M15→M16→M17→engine-hardening into main,
  or explicitly declare engine-hardening the new base.
        │
        ▼
G0 ── Freeze close-out (commit B-31, decide D4, delete stale binary, fix flake)  [~1-1.5d eng + D4 decision]
        │
        ▼
G1 ── Contract hardening (golden snapshots, StoragePort pin, sentinels, fix CLI_CONTRACT.md)  [~2.6d]
        │
        ▼
G2 ── Cursor + eventing: migration 0007 (state_changes)  [~2.4d, HARD-GATED ON DECISION D1]
        │
        ├──────────────┐
        ▼              ▼
G4 ── API server   G3 ── Namespace resolution  [~1.6d, HARD-GATED ON DECISION D2, runs parallel with G2/G4]
   [~5.3d, gated
    on D3 (soft)]
        │
        ▼
G5 ── Read-only + live GUI (Phases 1-2)  [~8-9d]
        │
        ▼  ← a demonstrable, truthful live GUI exists here
        │
  ┌─────┴─────┐
  ▼           ▼
G6 (Phase 4)  G10 (Phase 5, multi-user — only needs G5, not G6-G9)
Editor/       [~12.5d, runs parallel with G6-G9]
serializer
[~10-12d]
  │
  ▼
G7 ── Palette/introspection [~3.5d]
  │
  ├──────────────┐
  ▼              ▼
G8 Observability  G9 AI generation
[~9d, re-scoped   [~8d, gated on D4]
 up from 4-5d,
 needs G2 only —
 can run parallel
 with G5/G6/G7]
```

**Critical path to a demonstrable, truthful live GUI:** Phase A → G0 → G1 → G2 → G4 → G5,
with G3 running parallel to G2/G4. **≈3-4 weeks**, matching
`GUI_MASTER_PLAN.md` §13's own figure, independently re-confirmed at card level by both
subagents this session (engine-side sum: G0 1-1.5d + G1 2.6d + G2 2.4d + G4 5.3d ≈ 11-12d
engineering days on the serial spine, plus D1/D2/D4 decision latency and G3's 1.6d
absorbed into the G2/G4 parallel window).

**Critical path to n8n-class (full Phase 4, visual editor):** add G6 (~10-12d, largest
single milestone — zero existing serializer code, `yaml.Marshal` is called nowhere in the
repo today) and G7 (~3.5d) after G5. **≈6-7 weeks total** from today, decision latency
aside.

**G8, G9, G10 do not sit on this path.** G8 depends only on G2 (can start once G2 lands,
runs parallel with G5/G6/G7). G9 depends on G6 and D4. G10 depends only on G5. All three
can be staffed in parallel with the editor once G4/G5 land, not queued behind it
(`GUI_DEPENDENCY_MAP.md` §3).

---

## 3. Exact remaining engine work (G0-G3)

| Milestone | What | Card-level sum | Corpus rollup | Hard gate |
|---|---|---|---|---|
| **G0** | Commit B-31 remediation; decide D4; delete/rebuild stale binary; fix the system-test flake; prune stale branches | ~1-1.5d eng + D4 decision | 1d | D4 (decision) |
| **G1** | Golden JSON snapshots on `ExecutionEvent`; reflective `StoragePort` method-count pin; `GetWorkflow` not-found sentinel; fix `start.go`'s string-matched error check; correct `CLI_CONTRACT.md`'s ≥11 divergences; CLI JSON golden snapshots | ~2.6d | 2-3d | G0 |
| **G2** | Migration 0007 (`state_changes`); write it into `AppendEvent` and `UpsertInstance`'s existing transactions; `ReadChangesSince`; SDK wrapper; regression tests for all three non-evented transitions (`waiting`, timeout-resume, cancel-requested) | ~2.4d | 2d | **D1** — nothing in G2 can start before this decision |
| **G3** | Unify the three duplicated `"default"`-namespace predicate implementations; retire the overload; `--all-namespaces` flag; `ReadEventRange` wildcard branch; regression test that `StepStats` no longer silently zeros | ~1.6d | 2d | **D2** — nothing in G3 can start before this decision; otherwise fully parallel with G2 |

Independently re-confirmed today (2026-09-01, not merely carried over from the
2026-08-30 corpus): B-31's five validator rules (`CodeStepTypeUnknown`,
`CodeTriggerTypeUnknown`, `CodeSignalTimeoutAction`, `CodeRetryBackoff`,
`CodeIntelligenceCapability`) are present in the uncommitted `internal/validate/validate.go`
diff (+143 lines); D4 has not been decided anywhere in the repo (`.github/workflows/`
has zero live/schedule/secret references); the stale root `./awis` binary (17.2MB, dated
Aug 26) is still present; the flake at `cmd/awis/system_test.go:404` is unchanged.

**Governance flag carried forward, not resolved here:** G2 and G3 are the correctness-
critical engine-internals class — G2 edits the two write-transactions of the append-only
EventLog and the projection table; G3 touches the namespace read path that silently
returns wrong-but-plausible zeros today (R2, Critical). This is the same defect class
the B-31 remediation just closed five instances of. `GUI_MASTER_PLAN.md` §1.4 already
found that the 30-commit hardening program which produced this codebase ran on Opus, in
violation of this repository's own frozen Model Allocation Policy (Sonnet/Haiku only).
That tension is unresolved and is not this document's to resolve — it is flagged here
because G2/G3 are exactly the tier of change where it recurs, and because both subagents
that produced the card breakdown independently raised it unprompted.

---

## 4. Exact remaining API work (G4)

`cmd/awis-server` + `internal/api`, entirely in-module (decision D3). ~5.3 engineering
days across 13 cards (`E-G4-1` through `E-G4-13`, see `ENGINE_GUI_WORK_BREAKDOWN.md`).
Seven of those cards — server skeleton, PID/lock guard, API package skeleton,
`/healthz`, and the read-only workflow/instance/event routes — have **zero dependency on
G2 or G3** and can be built the moment D3 is provisionally accepted, front-loaded against
G1's contract while G2/G3 land. Only two cards have a hard gate: the SSE `/stream`
handler needs G2 complete (it is G2's first real consumer), and `/stats/steps` needs G3
complete (it is R2's fix point). The cancel-with-compensate route additionally requires
a small SDK fix first — `sdk/runtime_runner.go:127` currently hardcodes
`compensate=false`; only the CLI bypasses the SDK to reach `--compensate` today.

---

## 5. Exact remaining GUI work (G5-G10)

Mapped to the five product phases (`GUI_PRD.md`, `GUI_ROADMAP.md`):

| Milestone | Product phase | What | Card-level sum | Corpus rollup |
|---|---|---|---|---|
| **G5** | Phase 1 (read-only dashboard) + Phase 2 (live monitoring) + a Phase 4 preview | SPA shell, definitions/instances/timeline views, SSE live wiring, honest tick-quantized live copy, **a read-only canvas that needs no serializer and can ship before G6** | ~8-9d | 8-10d |
| **G6** | Phase 4 (visual editor) | YAML key-order strategy (must precede everything else in G6 — addresses N4), emitter, round-trip suite, save-from-canvas API, editable canvas, 6-relationship edge editing, inline validation, expression squiggles, layout persistence, E2E round-trip acceptance | ~10-12d | 8-12d |
| **G7** | Phase 4 (palette) | `List()` accessors on `engine.runners`, `NativeRunner.handlers`, `plugin.Manager`; `/api/v1/palette`; frontend palette wired to live registry instead of a hardcoded list | ~3.5d | 3-4d |
| **G8** | Phase 5 (observability) | **Re-scoped upward** — N1 established zero metrics/tracing infrastructure exists at all (not "StepStats needs an aggregate," but "stand up observability from nothing"); minimal metrics substrate, incremental step-stats aggregate off `state_changes`, cost/token index, stats/cost endpoints, a tick-loop benchmark at 1K/10K instances (N6), dashboard UI | ~9d | 4-5d (understated per audit's own action #7) |
| **G9** | Phase 5 (AI generation) | JSON Schema artifact for `WorkflowDefinition`, full-`Issue` plumbing through the CLI/API (currently dropped to line+message only), NL-to-workflow endpoint with repair loop, live-intelligence CI gate (D4), generation UI landing into G6's canvas | ~8d | 5-8d |
| **G10** | Phase 5 (multi-user) | Identity/session model, per-namespace authz, subprocess env scrubbing (R13, independent — can land any time), remote bind support, login/role UI, cross-namespace authz integration test, security review checkpoint | ~12.5d | 10-15d |

Full card-level detail, acceptance criteria, and per-card parallelization notes are in
`ENGINE_GUI_WORK_BREAKDOWN.md`.

---

## 6. Milestone structure — full table

| ID | Milestone | Depends on | Blocks | Resolves blocker | Est. |
|---|---|---|---|---|---|
| **A** | Process: STATE.md reconciliation + main merge | — | Naming "the engine" unambiguously; not on the engineering critical path | N7 | ~1d, zero engineering |
| **G0** | Freeze close-out | D4 | G1 | — | 1-1.5d + decision |
| **G1** | Contract hardening | G0 | G2 | R3, R4 | 2-3d (2.6d) |
| **G2** | Cursor + eventing | G1, **D1** | G4, Phase 2 | B-a (Critical), B-b | 2d (2.4d) |
| **G3** | Namespace resolution | **D2** | G4 | B-d (High), R2 (Critical) | 2d (1.6d), parallel with G2 |
| **G4** | API server | G2, G3, **D3** | G5, G8, Phase 1 | — | 5-7d (5.3d) |
| **G5** | Read-only + live GUI | G4 | G6, G10, Phase 3 | — | 8-10d (8-9d) |
| **G6** | Serializer + editor | G5, B-c | G7, G9, Phase 4 | B-c (High) | 8-12d (10-12d) |
| **G7** | Palette + introspection | G6 | Phase 4 (palette) | — | 3-4d (3.5d), parallel with G8/G9/G10 |
| **G8** | Observability | G4 (not G6/G7) | Phase 5 (observability) | B-e (partial) | 4-5d → **9d re-scoped** |
| **G9** | AI generation | G6, D4 | Phase 5 (AI generation) | — | 5-8d (8d) |
| **G10** | Multi-user | G5 only | Phase 5 (multi-user) | B-f | 10-15d (12.5d) |

Parenthesized figures are this document's card-level sums; the plain figures are the
original corpus's milestone-level estimates, retained as a cross-check. All five G0-G4
sums land inside or just above the corpus's own range; only G8 required a material
upward correction, and the corpus's own adversarial audit (`REPOSITORY_TRUTH_AUDIT.md`
§12, action #7) had already flagged that its G8 estimate understated the true scope.

---

## 7. Acceptance gates

Gates are the PRD's own phase-level acceptance criteria (`GUI_PRD.md`), restated here as
the program's formal exit conditions per milestone group.

**G0 exit (freeze gate):** `git status` clean on the B-31 remediation; D4 decided and, if
"gate," a scheduled CI job exists asserting the live model's `model` field; no stale
binary at repo root; `TestSystemRehearsalInitStartSubmitTrace` passes under concurrent
full-suite load.

**G1 exit (contract gate):** a deliberate rename of an `ExecutionEvent` field fails a
snapshot test; a deliberate `StoragePort` method addition/removal fails a reflective
test; `GetWorkflow`'s not-found path returns a typed sentinel; `docs/CLI_CONTRACT.md`
matches the built binary on every previously-divergent point.

**G2 exit (live-truth gate — the concrete form of the B-a fix):** submitting a
`wait_signal` workflow and reading `state_changes` directly shows a row for the
`running → waiting` transition; the same for timeout-`continue` resumption and
cancellation-requested. This is also `GUI_PRD.md` Phase 2's exact, testable acceptance
criterion once G4's SSE route exists: *"submitting [a wait_signal workflow] and watching
the GUI shows a `waiting` state transition in the live view within one tick interval,
without a page reload."*

**G3 exit (namespace-correctness gate):** `StepStats` called with a namespace that does
not literally match the stored events' namespace returns a correct result or an explicit
error — never silent zeros; a namespace literally named `default` behaves as an ordinary
name, not a wildcard.

**G4 exit (API-surface gate):** every route in `GUI_ARCHITECTURE.md` §6 is live, loopback-
bound, backed by a real engine call (no mocked handlers); `/healthz` reports true
liveness; a second `awis-server` process against the same DB fails fast (PID/lock guard).

**G5 exit (Phase 1+2 gate):** every view is backed by a real API endpoint, no mock data
(`GUI_PRD.md` Phase 1 AC); the live-truth gate above renders correctly end-to-end in the
browser, not just at the storage layer.

**G6 exit (Phase 4 gate):** a workflow built entirely in the visual editor, saved, and
resubmitted produces identical runtime behavior to the same workflow authored by hand in
YAML — verified by comparing `Transition`, `Compensation`, and `Fallback` sets
(`GUI_PRD.md` Phase 4 AC, exact wording).

**G7 exit:** the node palette reflects only step types and plugin capabilities actually
registered in the running deployment; adding a plugin changes the palette without a
frontend deploy.

**G8 exit:** a cost/token query for "total tokens spent by workflow X in the last 7 days"
returns in bounded time regardless of total event volume (`GUI_PRD.md` Phase 5 AC).

**G9 exit:** the AI-generation repair loop converges to a valid definition within a
bounded number of retries against a test prompt corpus; if D4 chose "gate," the live-CI
job fails loudly on model-ID drift (this is precisely the failure class the stale
`claude-sonnet-4-5` bug already demonstrated and the fix already closed once).

**G10 exit:** a second GUI user scoped to one namespace cannot read or act on another
namespace's instances — verified by an authorization integration test, not UI hiding
(`GUI_PRD.md` Phase 5 AC, exact wording); subprocess steps no longer inherit the full
parent environment; a security review checkpoint signs off before non-operator access
ships.

---

*Companion documents: `ENGINE_GUI_WORK_BREAKDOWN.md` (card-level detail),
`ENGINE_GUI_DECISION_RECORD.md` (recommended and alternative sequencing paths),
`FINAL_VERDICT.md` (direct answers to the founder's brief). Evidence base:
`GUI_MASTER_PLAN.md`, `GUI_DEPENDENCY_MAP.md`, `GUI_ARCHITECTURE.md`, `GUI_ROADMAP.md`,
`GUI_PRD.md`, `REPOSITORY_TRUTH_AUDIT.md`.*
