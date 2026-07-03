# AWIS IMPLEMENTATION MASTER PLAN
## From Empty Repository to Production-Ready V1

**Document Status:** Canonical — the single authoritative implementation roadmap
**Version:** 1.0
**Date:** 2026-07-02
**Produced by:** Independent Engineering Planning Tribunal (Implementation Planning Mode)
**Authority (Tier 0, descending):** OIP_CONSTITUTION.md → AWIS_ARCHITECTURE_FINALIZATION.md → AWIS_ARCHITECTURE_BLUEPRINT.md → AWIS_PRD.md
**Scope discipline:** This document sequences execution. It does not redesign, reinterpret, or extend the frozen architecture, module boundaries, capability ownership, interfaces, or product scope. Where Tier 0 documents conflict internally, the conflict is documented (§25 Risk Register, CONTRA-* entries), never silently resolved.

---

## 1. EXECUTIVE SUMMARY

AWIS V1 is implemented as **19 milestones (M0–M18) across the PRD's authoritative six-week envelope**, ending in a tagged `v1.0.0` release: a single Go binary containing the pull-based execution engine, SQLite EventLog/StateStore, the Go SDK, the YAML DSL, the subprocess and plugin runners, the Null and Anthropic intelligence adapters, the full CLI — and OIP running as the first AWIS application in the same repository as a **separate Go module**, structurally incapable of touching platform internals.

The plan's spine:

1. **Days 1–3 (M0–M1):** Bootstrap the repository, then design and freeze the schemas — the EventLog format is the platform's one irreversible artifact (P8, R1). No execution code is written until the schema pack passes human approval **Gate G1**.
2. **Week 1 (M2–M5):** Two parallel tracks. Track A: StoragePort + SQLite + migrations + EventLog + StateStore. Track B: IntelligencePort + NullAdapter + CapabilityRouter, and the two bounded expression grammars + WorkflowValidator.
3. **Week 2 (M6–M7):** The execution engine (pull loop, NativeRunner, transitions, retry, fallback, compensation, cancellation) and the signal subsystem (WAIT steps, single-transaction atomic delivery). Human **Gate G2** verifies execution semantics against the Finalization specification. The AWIS-E1 zero-AI CI gate goes live here and never comes down.
4. **Week 3 (M8–M10):** The public Go SDK surface, the WorkflowTestHarness with deterministic mode, and the YAML DSL parser. From here on, every workflow is testable in CI without any external service.
5. **Week 4 (M11–M13):** SubprocessRunner + `awis-step` Python library, the plugin system + `awis-plugin` Python library, and the reference `git-context-plugin`.
6. **Week 5 (M14–M15):** The core CLI, then **the V1 validation event**: OIP's `capture-decision` and `recall-decision` workflows built as an AWIS application with zero platform modification. Human **Gate G3** renders the platform-boundary verdict (QG-4).
7. **Week 6 (M16–M18):** AnthropicAdapter, the remaining CLI surface + `awis init` scaffolding, then hardening: quality gates QG-1–QG-5, NFR benchmarks, one week of dogfooding, and release. Human **Gate G4** approves `v1.0.0`.

**Critical path (28 engineering days):** M0 → M1 → M2 → M3 → M6 → M7 → M8 → M11 → M12 → M13 → M15 → M18.
**Total estimated effort:** ~40 dev-days of work compressed into ~30 working days via two parallel tracks and AI-assisted implementation (§28, §29).
**Every milestone leaves `main` buildable, tested, and releasable.** No milestone depends on the simultaneous completion of another unfinished milestone.

---

## 2. IMPLEMENTATION PHILOSOPHY

1. **Constitution first.** FP-7 (data outlives code) dictates the ordering: the EventLog format is designed, reviewed, and frozen before any code that produces it exists. All design budget is spent on the irreversible artifact first; everything else is a replaceable shell.
2. **Gall's law as sequencing.** Every milestone produces a smaller working system that the next milestone grows. The engine runs trivial native workflows (M6) before it runs signals (M7), before it runs YAML (M10), before it runs plugins (M12), before it runs OIP (M15).
3. **The boundary is the product of V1.** The entire plan converges on one question (QG-4): can OIP be expressed as two clean AWIS workflows with no platform surgery? The repository structure enforces the boundary mechanically (§5) so the answer cannot be accidentally faked.
4. **Evidence before assumption.** Where the frozen documents leave an implementation question open (e.g., recall's FTS index), the plan chooses the smallest mechanism that satisfies the frozen requirement and records it as an additive, reversible decision — never as new architecture.
5. **Zero-AI is a standing gate, not a test case.** From M6 onward, CI runs every workflow with NullAdapter as the sole provider (AWIS-E1). Article 32 is enforced by machinery, not by intention.
6. **Solo-founder arithmetic.** One reviewer exists. Therefore: one milestone = one branch = one PR = one review cycle; no PR mixes concerns; expensive-model attention is spent on decisions and semantics, not on typing (§28).

---

## 3. ENGINEERING STRATEGY

- **Language & toolchain:** Go 1.22+ (ADR-015), single static binary, SQLite bundled via a pure-Go driver (`modernc.org/sqlite` per PRD Implementation Dependencies — no CGO, no C toolchain in CI). Python 3.10+ only for the two pip libraries and the reference plugin.
- **Dependency policy:** minimal and boring. Approved V1 third-party Go dependencies: SQLite driver, YAML parser (`gopkg.in/yaml.v3`), CLI framework (`spf13/cobra` or stdlib `flag` — implementer's choice, decided in M0 and recorded), UUID, testify (tests only). The expression language is hand-written per Finalization Blocker 2 (no expr-lang/CEL — explicitly rejected there).
- **Branch & merge discipline:** trunk-based with short-lived milestone branches (`m06-execution-engine`), per the workspace convention "always create a new dedicated branch for major code changes." Squash-merge; PR title = milestone ID + objective. `main` is releasable after every merge.
- **Review economics:** milestones are sized to a single review sitting (≤ ~1,500 lines of reviewable diff; larger milestones split into stacked PRs within the milestone).
- **Determinism as a design constraint:** all clocks, IDs, and randomness flow through injectable sources from M2 onward, so M9's deterministic mode is wiring, not surgery.
- **Documentation as code:** the normative specs (schemas, grammars, wire protocols) live in `docs/` and are extracted from the frozen Tier 0 documents in M1 — they are transcriptions, not new design.

---

## 4. REPOSITORY BOOTSTRAP PLAN (M0)

Executed once, before any product code:

1. `git init`; default branch `main`; `.gitignore` (Go, SQLite artifacts, `.awis/`, `.decisions/.index/`, Python venvs).
2. `go.work` workspace with two Go modules:
   - `github.com/awis/awis` — platform: runtime, CLI, SDK package (see CONTRA-1 in §25 for the module-path note).
   - `github.com/awis/oip` — the OIP application, at `apps/oip/` (empty scaffold until M15).
3. `Makefile` targets: `build`, `test`, `lint`, `race`, `contract`, `e1` (zero-AI gate), `bench`, `release-dry`.
4. CI skeleton (GitHub Actions): fmt/vet/lint + build + test on Linux and macOS. Red-to-green from the first commit.
5. `docs/` skeleton + `CODEBASE.md` stub; `README.md` stating the platform boundary in one paragraph.
6. Toolchain pinning: `go.mod` toolchain directive; `golangci-lint` config; `pyproject.toml` stubs for the two Python packages.

**Exit:** an empty-but-green repository. CI passes. Nothing else exists.

---

## 5. REPOSITORY STRUCTURE

```
awis/                                  (repo root)
├── go.work                            (ties the two modules for local dev)
├── go.mod                             module github.com/awis/awis
├── cmd/awis/                          CLI entrypoint + command implementations
├── sdk/                               PUBLIC surface (FR-SDK-02):
│   ├── awis.go workflow.go step.go trigger.go runner.go intelligence.go recall.go
│   └── testing/                       harness.go, mock.go (M9)
├── internal/                          NOT importable outside github.com/awis/awis
│   ├── storage/                       StoragePort impl, SQLite adapter, migration runner
│   ├── engine/                        pull loop, transition evaluator, retry, compensation, cancellation
│   ├── runner/                        native.go, subprocess.go, plugin.go, intelligence.go
│   ├── expr/                          template grammar + condition grammar (Finalization Blocker 2)
│   ├── validate/                      WorkflowValidator
│   ├── dsl/                           YAML → WorkflowDefinition
│   ├── signal/                        inbox, atomic delivery, wait records, timeouts
│   ├── intelligence/                  CapabilityRouter + adapters/ (null, anthropic)
│   ├── plugin/                        lifecycle, JSON-RPC client, registry
│   ├── observability/                 trace assembly, metrics computation, audit log, recall FTS
│   └── config/                        config.yaml + env + flag hierarchy (§30 of PRD)
├── migrations/                        embedded SQL, sequential (see §14)
├── apps/oip/                          module github.com/awis/oip (M15)
│   ├── go.mod                         ← separate module: CANNOT import awis/internal/*
│   ├── handlers/                      RecordAppendHandler, IndexFTSHandler, SemanticRankHandler
│   ├── workflows/                     capture-decision.yaml, recall-decision.yaml
│   └── cmd/oip/                       OIP entrypoint (registers handlers + workflows via SDK)
├── plugins/git-context-plugin/        Python reference plugin + awis-plugin.yaml
├── python/
│   ├── awis-step/                     pip library for subprocess step handlers (FR-SDK-10)
│   └── awis-plugin/                   pip library for plugins + testing.mock_request (FR-PS-14/15)
├── docs/                              normative specs (TDS pack, §12) + ADR index (pointer to Blueprint)
├── examples/workflows/                hello-world.yaml, with-signal.yaml, with-intelligence.yaml
│                                      (embedded into the binary; emitted by `awis init`)
└── .github/workflows/ci.yml           (§21)
```

**Why one repository, two Go modules (execution decision, not architecture):** the smallest review surface and one CI for a solo founder, while Go's `internal/` visibility rule — which is import-path-based — makes it a **compile error** for `github.com/awis/oip` to import `github.com/awis/awis/internal/...`. FR-SDK-09 and the platform boundary (P1, P5) are enforced by the toolchain, not by discipline. The boundary test (QG-4) is therefore mechanically honest.

---

## 6. PACKAGE ARCHITECTURE

Dependency direction is strictly downward; no package imports a package listed above it:

```
cmd/awis ──► sdk ──► internal/engine ──► internal/{runner, signal, validate}
                          │                    │
                          ▼                    ▼
                internal/storage      internal/{expr, intelligence, plugin}
                          │
                          ▼
                    migrations (embedded SQL)

internal/dsl ──► sdk types (produces WorkflowDefinition; consumed via registration)
internal/observability ──► internal/storage (reads EventLog; computes traces/metrics/recall)
internal/config ──► (leaf; read by cmd and sdk.NewRuntime)
apps/oip (separate module) ──► sdk ONLY
python/* ──► wire protocols only (no Go dependency)
```

Rules:
- `sdk` defines the public types (`WorkflowDefinition`, `Step`, `StepHandler`, `StepContext`, `StepResult`, `WorkflowRunner`, `RecallAPI`, `IntelligencePort`) — internal packages depend on `sdk` types, never the reverse exposing internals (FR-SDK-09).
- `internal/expr` is a leaf with zero dependencies: pure functions from strings + variable scope → values/booleans. This makes the grammars exhaustively unit-testable.
- Adapters (`intelligence/adapters/*`, storage SQLite adapter) are terminal packages; nothing imports an adapter except its registry/router (P6).

---

## 7. MODULE BREAKDOWN

Modules are exactly the Blueprint's five runtime layers plus the SDK, CLI, and OIP. No module is added or merged.

| Module | Blueprint § | Owns | Milestones |
|---|---|---|---|
| Persistence Layer | §9, §20 | EventLog, StateStore, WorkflowRegistry, StepResultCache, signal/wait/plugin/audit tables, migrations | M2, M3 |
| Intelligence Layer | §13–17 | IntelligencePort, CapabilityRouter, fallback chain, Null + Anthropic adapters | M4, M16 |
| Expression & Validation | §7 + Finalization B2 | Two grammars, WorkflowValidator | M5 |
| Workflow Engine | §5 L1, §8 | Pull loop, claim, dispatch, settle, transitions, retry, fallback, compensation, cancellation | M6 |
| Signal Subsystem | §8 | WAIT steps, signal inbox, atomic delivery, timeouts | M7 |
| Step Runtime | §5 L2 | NativeRunner (M6), SubprocessRunner (M11), PluginRunner (M12), IntelligenceRunner (M6/M4) | M6, M11, M12 |
| Application SDK | §12, §25 | Public surface, registration, TestHarness, deterministic mode | M8, M9 |
| YAML DSL | §7 | Parser → WorkflowDefinition | M10 |
| Plugin Architecture | §11 | JSON-RPC 2.0, manifest, lifecycle, registry; Python libs | M12, M13 |
| Observability | §5 L5, §26 | Trace assembly, metrics, audit log, recall FTS, structured logging | M6 (logging), M14, M17 |
| CLI | PRD §15 | Full command tree | M14, M17 |
| OIP Application | §10, §18 | Two workflows, three handlers, `oip.db`, The Record | M15 |

---

## 8. CAPABILITY BREAKDOWN

Implementation order of user-visible capability, each mapped to its PRD requirement family:

1. Durable event recording (FR-ST-01..06) — M2/M3
2. Deterministic step execution with retry/fallback/compensation/cancellation (FR-SE-*, FR-WE-07..11) — M6
3. Human-in-the-loop WAIT/signal (FR-WE-05/06, FR-SE-12/13) — M7
4. Workflow definition in Go (FR-SDK-*) — M8
5. Deterministic testing (FR-SDK-06/07/08, QG-5) — M9
6. Workflow definition in YAML + validation (FR-WD-*) — M10
7. Polyglot steps (FR-SE-02, FR-SDK-10) — M11
8. External capabilities via plugins (FR-PS-*) — M12/M13
9. Operate & debug from the terminal (FR-RM-*, FR-OB-*) — M14/M17
10. Optional cloud intelligence (FR-IL-02/05..09) — M16
11. Execution-history recall (FR-IL-11) — M17
12. Organizational decision memory — OIP (QG-3/4) — M15

---

## 9. DEPENDENCY DAG

```
M0 Bootstrap
 └─► M1 Schema & Format Freeze  [GATE G1]
      ├─► M2 Storage Foundation ──► M3 State Projection ─┐
      ├─► M4 Intelligence Seam (Null + Router) ──────────┤
      └─► M5 Expression Engine + Validator ──────────────┤
                                                         ▼
                                   M6 Execution Engine Core
                                                         │
                                   M7 Signal Subsystem ◄─┘   [GATE G2 after M7]
                                                         │
                       ┌─────────────────────────────────┤
                       ▼                                 ▼
             M8 SDK Public Surface              M11 SubprocessRunner + awis-step
                       │                                 │
          ┌────────────┼────────────┐                    ▼
          ▼            ▼            ▼           M12 Plugin System + awis-plugin
   M9 TestHarness  M10 YAML DSL  M14 Core CLI            │
          │            │            │                    ▼
          │            │            │           M13 git-context-plugin
          └────────────┴─────┬──────┴────────────────────┘
                             ▼
                  M15 OIP on AWIS   [GATE G3 — platform boundary verdict]
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
      M16 AnthropicAdapter  M17 Full CLI + init
              └──────────────┴──────────────┘
                             ▼
              M18 Hardening & v1.0.0 Release  [GATE G4]
```

Notes: M4 and M5 depend only on M1 (types/specs), not on M2 — they run fully parallel to Track A. M14 (core CLI) depends on M6/M7 (runtime) and M10 (auto-discovery of YAML workflows) but not on plugins. M16 depends only on M4's interface and can start any time after it; it is scheduled in Week 6 per the PRD sequence.

---

## 10. CRITICAL PATH

**M0 → M1 → M2 → M3 → M6 → M7 → M8 → M11 → M12 → M13 → M15 → M18** (≈ 28 engineering days).

Implications:
- The plugin chain (M11→M12→M13) is on the critical path solely because OIP's `assemble-context` step is plugin-typed. If M12 slips, the contingency (no scope change) is to begin M15 with the three native handlers and the two YAML definitions validated in the harness with a stubbed plugin capability, folding the real plugin in before Gate G3 renders its verdict.
- M4, M5, M9, M10, M14, M16, M17 all carry float; they are the designated absorbers of schedule pressure and the natural parallel work for AI-assisted execution.
- Gate G1 is the only hard full-stop: no downstream code is written against an unapproved EventLog/WorkflowDefinition schema. Gates G2–G4 review completed work and do not idle the pipeline (float work proceeds during review).

---

## 11. PARALLEL EXECUTION MATRIX

| Window | Track A (serial, critical) | Track B (parallel) | Track C (parallel) |
|---|---|---|---|
| Days 1–3 | M1 schema pack + G1 | — | — |
| Week 1 | M2 → M3 storage | M4 intelligence seam | M5 expression engine |
| Week 2 | M6 engine → M7 signals | (M5 finishes; grammar fixture suite) | structured logging (part of M6) |
| Week 3 | M8 SDK | M9 harness (after M8 API lands) | M10 YAML DSL |
| Week 4 | M11 → M12 | M13 plugin (after M12 protocol lands) | M14 core CLI |
| Week 5 | M15 OIP + G3 | M14 finish/polish | TDS-05 Record format sign-off |
| Week 6 | M18 hardening | M16 Anthropic | M17 full CLI + init |

Constraint honored: no two tracks edit the same package in the same window; merge conflicts are structurally minimized because milestones map 1:1 to new packages.

---

## 12. TECHNICAL DESIGN SPECIFICATIONS (per module, required before coding)

All TDS documents are **transcriptions and completions of the frozen documents into normative, versioned spec files** under `docs/`. None introduces new design. Each is ≤ 3 pages and is the review artifact for its gate.

| TDS | Content | Source of truth | Written in | Blocks |
|---|---|---|---|---|
| TDS-01 `docs/EVENTLOG_FORMAT.md` | ExecutionEvent envelope, all 12+ event payload schemas, `schema_version` semantics, sequence rules | Blueprint §9 | M1 | M2 |
| TDS-02 `docs/WORKFLOW_SCHEMA.md` | WorkflowDefinition/Step/Transition/Trigger JSON serialization, semver + immutability rules | Blueprint §6 | M1 | M2, M8, M10 |
| TDS-03 `docs/EXPRESSION_GRAMMARS.md` | Both formal grammars verbatim + the prohibited lists + a conformance fixture table (valid/invalid corpus) | Finalization Blocker 2 | M1 | M5 |
| TDS-04 `docs/SUBPROCESS_PROTOCOL.md` | stdin/stdout JSON exchange for subprocess steps: request/response envelope, error mapping, timeout behavior | Blueprint §25 | M11 (day 1) | M11 |
| TDS-05 `docs/PLUGIN_PROTOCOL.md` | JSON-RPC 2.0 framing, handshake/Manifest message, lifecycle state machine, restart/idle rules | Blueprint §11 | M12 (day 1) | M12 |
| TDS-06 `apps/oip/docs/RECORD_FORMAT.md` | OIP entry format (YAML frontmatter + Markdown body, ID scheme, `corrects:` linkage, append-only rules per Constitution Art. 7/9/10/11) | Constitution Title II; Blueprint §10/§18 | M15 (days 1–2) | M15 handlers — **human sign-off required; this is OIP's irreversible artifact** |
| TDS-07 `docs/CLI_CONTRACT.md` | Output formats for status/trace/history + `--json` schemas + error taxonomy templates | PRD §15–20, §26 | M14 (day 1) | M14, M17 |

---

## 13. INTERFACE CONTRACTS

All interfaces below are **frozen** (Blueprint) and are implemented, never redesigned. Implementation order:

1. **StoragePort** (Blueprint §20 — 12 methods) — M2/M3. Contract test suite written against the interface in M2 and reused verbatim for the V2 Postgres adapter.
2. **IntelligencePort** (Blueprint §13 — Draft/Embed/Synthesize/Classify + IsAvailable/Capabilities/ProviderName) — M4. `Classify` ships as the mandated non-callable placeholder (FR-IL-10); NullAdapter implements all methods.
3. **StepHandler / StepContext / StepResult** (Blueprint §12) — M6 (types in `sdk` from M1's TDS-02).
4. **WorkflowRunner** (Submit/Signal/Status/Cancel/List) — M8.
5. **RecallAPI** (QueryHistory/ReplayInstance/StepStats) — M8 (QueryHistory/StepStats), `ReplayInstance` dry-run in M17 (`awis replay`, PRD Should-Have).
6. **Subprocess JSON protocol** — M11 (TDS-04).
7. **Plugin JSON-RPC 2.0 protocol + manifest** — M12 (TDS-05).

Stability rule: after M8 merges, any change to a `sdk` exported identifier requires a written justification in the PR description referencing the frozen document that demands it. After v1.0.0, semver applies (FR-SDK-01/11).

---

## 14. STORAGE & MIGRATION PLAN

**Storage implementation order:** migration runner → EventLog append/read → StateStore upsert/get/list/claim → WorkflowRegistry → StepResultCache → signal/wait tables → plugin registry → audit log → recall FTS.

**Migration sequence (embedded, sequential, forward-only):**

| File | Contents | Milestone |
|---|---|---|
| `0001_core_execution.sql` | `schema_version` table; `execution_events`; `workflow_instances` (including `cancellation_requested INTEGER NOT NULL DEFAULT 0` — folded in for greenfield installs; the Finalization's `ALTER TABLE` described a spec delta, not a runtime migration, since no deployment predates this repo); `workflow_definitions`; `step_results_cache`; all Blueprint §20 indexes | M2/M3 |
| `0002_signals.sql` | `signal_inbox`, `wait_records` + indexes | M7 |
| `0003_plugins.sql` | `plugins`, `plugin_capabilities` | M12 |
| `0004_audit.sql` | `audit_log` (append-only; required by NFR-S-05/FR-OB-08; table mandated by frozen docs though absent from the §20 enumeration — additive, StoragePort-internal) | M14 |
| `0005_recall_fts.sql` | FTS5 virtual table over `execution_events` payloads (platform-owned, inside `runtime.db`; the smallest mechanism satisfying FR-IL-11's "FTS results"; distinct from and unrelated to OIP's `oip.db` — Blocker 5 boundary untouched) | M17 |

Rules: migrations run automatically on startup in local mode (FR-ST-06); every migration is tested by a fixture DB at version N−1; `awis rebuild-state` is implemented in M3 and re-verified after every subsequent migration. **Pre-release only** (before the v1.0.0 tag), migrations may be folded forward since no external installation exists; after tagging, the sequence is append-only forever (P8).

**Application storage:** OIP's `oip.db` (`entries`, `entries_fts`, `entry_vectors`) is created and owned entirely by OIP's handlers in `apps/oip` (Blocker 5). It has its own mini-migration runner inside the OIP module. No AWIS code references it.

---

## 15. API ROLLOUT PLAN

V1's API **is the CLI plus the embedded Go SDK** (the HTTP server is V2 — NG scope). Rollout:

1. **M14 — Core CLI** (the development loop): `start`, `stop`, `version`, `submit` (+`--input`, `--wait`), `signal`, `cancel` (+`--compensate`), `status` (+`--watch`), `trace` (+`--json`, `--full`), `workflow validate|list|show`. Every command ships with `--json` from day one (PP-6) — retrofitting is forbidden.
2. **M17 — Full surface**: `init`, `history`, `logs`, `metrics`, `recall` (+`--synthesize`), `replay`, `audit`, `plugin install|list|status|remove`, `config show|set|validate|edit`, `rebuild-state`, `export`, `prune-events --dry-run`.
3. Output contracts are fixed by TDS-07 before implementation; the trace and status formats in PRD §17/§19–20 are the acceptance fixtures (golden-file tests).
4. Error taxonomy (PRD §26: what/where/what-now) is implemented as a shared error-rendering helper in `cmd/awis` in M14 so every later command inherits it.

---

## 16. CLI ROLLOUT PLAN

Covered structurally in §15; sequencing detail:

- **Week 5, M14:** the nine commands a developer needs to run and debug OIP. This is deliberately ahead of the PRD's week-6 "CLI (all commands)" line item — an execution refinement (the boundary validation and dogfood need `submit/trace/status/signal` live), not a scope change; the PRD's week-6 completion point for *all* commands is preserved by M17.
- **Week 6, M17:** everything else, plus `awis init` scaffolding (config.yaml, .gitignore, three example workflows, `handlers/example_handler.go`, `README_AWIS.md` — FR-RM-01, embedded via `go:embed`).
- CLI acceptance is measured against PRD §32's checklists verbatim; QG-1 (init→trace ≤ 5 min on clean macOS + Linux) and QG-2 (trace readable without docs) are exit criteria of M18, not M17.

---

## 17. SDK ROLLOUT PLAN

1. **M1:** public types are specified (TDS-02); the `sdk` package is born containing only types + doc comments.
2. **M8:** `NewRuntime`, `Config`, registration (`RegisterHandler`, `RegisterWorkflow`), `WorkflowBuilder` (+ `Build()` semver validation), `WorkflowRunner`, `RecallAPI` (QueryHistory/StepStats). Registration failure messages follow PRD §18's format.
3. **M9:** `sdk/testing`: `WorkflowTestHarness` (synchronous run, `h.Signal`, `h.Tick`, `h.WaitForCompletion`, `h.GetOutput`), `DeterministicMode()` (fixed clock, seeded IDs, NullAdapter, manual tick), `NewMockIntelligence()` with `OnDraft/OnEmbed/OnClassify` fixtures. QG-5 (< 1s full-workflow test, zero external deps) is this milestone's acceptance test.
4. **M10:** the YAML DSL compiles to the identical `WorkflowDefinition` struct; a round-trip equivalence test (YAML-defined vs builder-defined identical workflow → deep-equal structs) is the milestone's keystone test (FR-WD-02).
5. **Python:** `awis-step` (M11) and `awis-plugin` (+ `awis_plugin.testing.mock_request`) (M12); both published to the repo only in V1 (pip-installable from path/git; PyPI publication is a release-day nicety, not a gate).
6. Versioning: `sdk` is semver-tagged with the platform at v1.0.0; one-major-version backward compatibility (FR-SDK-11) becomes binding at that tag.

---

## 18. PLUGIN STRATEGY

V1 plugin scope is exactly what the frozen documents demand — the subprocess JSON-RPC system with one reference plugin — justified by a concrete V1 use case: OIP's `assemble-context` step (Blocker 1/ADR-010 make OIP an AWIS application; its capture workflow's first step is plugin-typed).

- **M12:** `internal/plugin` — manifest parsing (`awis-plugin.yaml`), lifecycle FSM (REGISTER→SPAWN→HANDSHAKE→ACTIVE→IDLE→TERMINATE), JSON-RPC 2.0 over stdin/stdout, crash auto-restart (max 3), idle **process kill** + respawn (< 2s expected, per the Finalization's recommendation), registry tables, PluginRunner integration. Python `awis-plugin` library + offline test harness.
- **M13:** `git-context-plugin` (Python): `git.context.assemble`, `git.diff.fetch`; pinned dependencies; per-plugin venv (PR-5 mitigation).
- **Boundaries enforced in code:** plugins receive only declared step inputs; the plugin process inherits a minimal environment; no handle to `runtime.db` ever crosses the pipe (FR-PS-06, NFR-S-02).
- **Deliberately absent (per frozen scope):** remote install, registry/marketplace, WASM — V2/V3.

---

## 19. TESTING STRATEGY

The Blueprint §27 pyramid, mapped to concrete suites and when each becomes mandatory:

| Layer | Suite | Introduced | CI-blocking from |
|---|---|---|---|
| Unit | Per-package Go tests; `internal/expr` conformance corpus from TDS-03 (every valid AND every prohibited construct) | M2/M5 | Always |
| Contract | StoragePort suite run against SQLite adapter (reused for Postgres in V2); IntelligencePort suite run against Null (M4) and Anthropic-with-recorded-fixtures (M16) | M2/M4 | Always |
| Integration | WorkflowTestHarness end-to-end workflows: linear, fan-out + join, retry-exhaustion→fallback, compensation, cancellation (± `--compensate`), WAIT/signal, timeout actions | M9 (engine-level precursors in M6/M7) | M9 |
| System | Spawned real binary + real SQLite: start → register → submit → signal → trace → stop; crash-recovery test (kill -9 mid-workflow → restart → rebuild-state → identical state, NFR-R-03) | M14 | M14 |
| Protocol | Golden-file JSON-RPC and subprocess-protocol exchanges; Python libs tested with pytest against the same golden files (single source of wire truth) | M11/M12 | M11/M12 |
| Zero-AI (AWIS-E1) | Entire workflow test corpus re-run with NullAdapter as sole provider | M6 | M6 — permanent |
| Golden output | `awis trace`/`status`/`validate` rendered output vs fixtures (TDS-07) | M14 | M14 |
| Benchmarks | NFR-P-01..09 as Go benchmarks + a 100K-event fixture DB for trace/rebuild targets | M18 (informational earlier) | G4 gate |

Test-data policy: one shared fixture package (`internal/fixtures`) with deterministic workflow definitions used across layers, so a semantic change breaks loudly everywhere at once.

---

## 20. VERIFICATION STRATEGY

Beyond tests, each milestone has a **verification checkpoint** — an executable demonstration recorded in the PR description:

- M2/M3: append 10K events, kill the process mid-write, reopen, `rebuild-state` → byte-identical StateStore.
- M5: grammar fuzz pass (go-fuzz/`testing.F`) over both parsers — no panics; prohibited constructs rejected with precise positions.
- M6: fan-out workflow executes all branches concurrently within `max_parallel`; idempotency-cache prevents double execution on injected duplicate claims.
- M7: the three-table transaction verified under injected crash between each pair of writes (SQLite hook) — invariant: EventLog `SignalReceived` present ⇔ delivered.
- M9: OIP-shaped workflow (plugin step stubbed) runs in a unit test in < 1s.
- M12: kill the plugin process externally mid-call → auto-restart → step retry succeeds; idle-kill → transparent respawn.
- M15: **the boundary demo** — `git log --stat` proof that building OIP touched zero files under the platform module (excluding go.work). This artifact is Gate G3's evidence.
- M18: QG-1..5 executed literally, on clean macOS and Linux; NFR benchmark report committed to `docs/V1_BENCHMARKS.md`.

---

## 21. CI/CD STRATEGY

**CI (every PR and main), in gate order (Quality Gates §, mirrored):**

1. `gofmt`/`go vet`/`golangci-lint`; Python: `ruff` + `pytest` for the two libraries
2. Build: Linux + macOS, CGO_DISABLED (pure-Go SQLite driver keeps CI hermetic)
3. Unit + contract suites
4. Integration (harness) + system tests
5. **AWIS-E1 zero-AI gate** (from M6; failure blocks merge, no exceptions)
6. `go test -race` on engine/signal/storage packages
7. Golden-output tests (from M14)
8. Benchmarks: informational job on main until M18, then thresholded

**No network in CI, ever, in V1.** AnthropicAdapter is tested against recorded HTTP fixtures (`httptest`); one optional manually-triggered smoke workflow hits the live API with a repo secret — never on the merge path (P7).

**CD:** tags matching `v*` run goreleaser: static binaries (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64), checksums, release notes generated from milestone PR titles. `main` is releasable at all times; releasing is a tag, not a project.

---

## 22. DOCUMENTATION STRATEGY

- **Normative specs** (TDS pack, §12): written before their module, kept in `docs/`, treated as code (reviewed in the same PR pipeline).
- **Per-milestone doc rule (Quality Gates):** a milestone is not done until its user-visible behavior is documented — CLI commands in `docs/CLI.md` sections, SDK in godoc + one runnable example per public type, plugin/subprocess protocols in their TDS files.
- **`README_AWIS.md` / init scaffolding** (M17): the 5-minute path (PRD Flow 1), verbatim-tested by QG-1.
- **ADR discipline:** the 15 ADRs live in the frozen Blueprint; `docs/ADR_INDEX.md` points at them. New implementation-level decisions (e.g., CLI framework choice, SQLite driver) get lightweight `docs/edr/` notes — one page, no architecture.
- **No new planning documents.** This plan, the PRD, and the Blueprint are the complete paper trail for V1.

---

## 23. HUMAN REVIEW GATES

| Gate | After | Question the human answers | Evidence reviewed | Failure action |
|---|---|---|---|---|
| **G1 — Schema Freeze** | M1 | "Would I still accept this EventLog/WorkflowDefinition format in five years?" (P8, R1, decade-reader test Art. 12) | TDS-01/02/03 + fixture corpus | Iterate M1; nothing downstream starts |
| **G2 — Execution Semantics** | M7 | "Do the engine and signal implementations match the Finalization specifications exactly (atomic transaction, cancellation FSM, fallback rules)?" | Crash-injection verification runs; event-sequence fixtures vs Finalization Blocker 3/4 text | Fix within M6/M7 scope before M8 merges |
| **G3 — Platform Boundary Verdict** | M15 | "Did OIP require platform surgery?" (QG-4, R2) | The `git log --stat` boundary artifact + both workflows green in harness and via CLI | Per Blueprint: **the boundary moves, not the application** — a scoped platform-fix milestone is opened before any further work; second-application planning freezes |
| **G4 — Release** | M18 | "Do QG-1..5 pass and are NFR benchmarks within targets on clean machines?" | Benchmark report, quality-gate checklist, one week of dogfood metrics | Ship blocks; hardening continues |

Additionally: **TDS-06 (OIP Record format) requires explicit human sign-off inside M15** before the handlers are written — it is OIP's irreversible artifact and receives the same reverence as G1.

---

## 24. DEFINITION OF DONE (global; per-milestone additions in §27)

A milestone is done when ALL hold:

1. Architecture integrity: no frozen interface altered; no `internal/` leakage; dependency direction (§6) intact.
2. Compiles on Linux + macOS; zero lint findings.
3. All new logic unit-tested; suites of §19 applicable to this milestone are green, including AWIS-E1 (from M6).
4. Integration: the milestone's verification checkpoint (§20) is demonstrated and recorded in the PR.
5. Documentation updated per §22.
6. Repository health: `main` after merge is releasable; no dead code, no `TODO` without a tracked issue; migrations (if any) tested from N−1.
7. Merge approval: PR reviewed and approved by the human (gates G1–G4 where applicable).

---

## 25. RISK REGISTER

Blueprint risks R1–R10 and PRD risks PR-1–PR-10 remain in force. Implementation-plan-specific additions:

| ID | Risk | P | I | Mitigation |
|---|---|---|---|---|
| IR-1 | Schema freeze (G1) approved too casually; EventLog format defect discovered post-M6 | M | High | G1 reviews against a written checklist: schema_version present, decade-reader test, replay sufficiency for every event type; M6 replay tests exercise every event type |
| IR-2 | Expression grammars drift from Finalization text during hand-implementation | M | High | TDS-03 conformance corpus is generated from the Finalization's grammar + prohibited lists before parser code exists; fuzzing in M5 |
| IR-3 | Signal atomicity subtly wrong under SQLite WAL (e.g., transaction not spanning all three writes) | L | High | Single `BEGIN IMMEDIATE` transaction implemented verbatim from Blocker 3 SQL; crash-injection tests between each write pair (G2 evidence) |
| IR-4 | Plugin chain (M11–M13) slips and drags M15 | M | Med | Critical-path contingency in §10: OIP native handlers + stubbed plugin capability first; float from Tracks B/C reassigned |
| IR-5 | CLI output formats churn late, breaking golden tests repeatedly | M | Low | TDS-07 fixes formats before M14; PRD §17/19/20 excerpts are the fixtures |
| IR-6 | Dogfood week reveals E1-quality issues in OIP drafts (application-level, T5) | M | Med | Application concern, not platform: recorded for OIP iteration; does not block AWIS v1.0.0 (QG-3/QG-4 are the platform gates) |
| IR-7 | Solo-founder review fatigue in weeks 4–6 (three tracks) | M | Med | Milestone size cap (§3); expensive-model review assistance on low-risk PRs; gates concentrate human attention on the four decisions that matter |

**Documented Tier-0 contradictions (per mandate: recorded, not silently resolved):**

| ID | Contradiction | Disposition |
|---|---|---|
| CONTRA-1 | FR-SDK-01 names the module `github.com/awis/sdk`; PRD Flow 1 installs `github.com/awis/awis@latest`. Go's path rules make both literally true only with a repo split or vanity imports. | V1 implements one module `github.com/awis/awis` with public package `github.com/awis/awis/sdk` (identical surface). The literal `github.com/awis/sdk` import path is satisfied in V2 via vanity-import redirect or SDK repo split — a one-line import change for applications. Recorded here and in `docs/edr/edr-001-module-path.md`. |
| CONTRA-2 | NFR-P-02 requires native dispatch < 10ms P95; PRD §32 acceptance says ≤ 50ms P95. | Engineering target 10ms; ship gate 50ms. Both recorded in the benchmark suite. |
| CONTRA-3 | Blueprint §14 has AnthropicAdapter delegate Embed to OpenAI, but the OpenAI adapter is V2 (FR-IL-03). | Embed is unavailable in V1 (matches PRD §24 "embedding unavailable; semantic search disabled"). OIP's `semantic-rank` step passes FTS ordering through unchanged when no vectors exist — consistent with FR-IL-11 FTS-first. No interface change. |
| CONTRA-4 | FR-IL-11 requires FTS over execution history, but no FTS structure exists in the frozen `runtime.db` schema enumeration; likewise `audit_log` is mandated (NFR-S-05) but unenumerated. | Additive migrations 0004/0005 (§14), internal to StoragePort, invisible to every frozen interface. Recorded as required-by-requirements additions. |

---

## 26. ROLLBACK STRATEGY

- **Code:** one milestone = one squash-merged PR ⇒ `git revert <merge-commit>` restores the previous healthy state exactly. New functionality lands in new packages, so reverts do not ripple.
- **Schema (pre-v1.0.0):** no external installations exist; a defective migration is folded forward (fixed in place) with the fixture-DB tests re-run. From the v1.0.0 tag onward: migrations append-only; a bad migration is corrected by a new forward migration, never edited (P3/P8 discipline applied to our own repo).
- **State:** the EventLog is authoritative by construction; any StateStore damage from a defective build is repaired by `awis rebuild-state` (available from M3 — deliberately early, so the recovery tool predates everything that could need it).
- **Release:** binaries are immutable per tag; a bad release is superseded by the next patch tag, never overwritten.
- **Per-milestone rollback criteria** are listed in §27; the universal trigger is: any Quality Gate (§ Definition of Done item 1–6) found violated post-merge ⇒ revert first, fix on a branch, re-review.

---

## 27. RELEASE MILESTONES

Legend: **Obj** objective · **Scope** in/out · **In** inputs · **Out** outputs · **Dep** dependencies · **Risk** dominant risk · **Val** validation · **AC** acceptance criteria · **DoD** additions to §24 · **Merge** merge criteria · **RB** rollback criteria · **Repo** expected repository state after merge. Effort in solo-founder days (AI-assisted).

---

### M0 — Repository Bootstrap (0.5d, Week 0/Day 1)
**Obj:** Empty-but-green repository per §4. **Scope:** repo, two-module workspace, Makefile, CI skeleton, lint config; no product code. **In:** §4–§5 of this plan. **Out:** green CI on `main`. **Dep:** none. **Risk:** tool-choice churn — decide CLI framework + SQLite driver now, record in `docs/edr/`. **Val:** CI green on both OSes. **AC:** `make build test lint` passes on a clean clone. **DoD:** EDR notes committed. **Merge:** CI green. **RB:** n/a (first commit). **Repo:** skeleton only, releasable trivially.

### M1 — Canonical Schema & Format Freeze (2.5d, Days 1–3) → **GATE G1**
**Obj:** Freeze the irreversible artifacts before any code: TDS-01 (EventLog), TDS-02 (WorkflowDefinition serialization), TDS-03 (grammar conformance corpus); `sdk` package created with public types + doc comments only. **Scope:** specs and types; explicitly NO executable logic. **In:** Blueprint §6/§9, Finalization Blocker 2. **Out:** three TDS docs, `sdk` types, fixture corpus files. **Dep:** M0. **Risk:** IR-1. **Val:** G1 checklist (schema_version on every table/format; every Blueprint §9 event type has a payload schema; decade-reader read-through). **AC:** human sign-off recorded in PR. **DoD:** fixtures compile as Go test data. **Merge:** G1 approved. **RB:** iterate in place — nothing depends on it yet. **Repo:** specs + types; still no behavior.

### M2 — Storage Foundation (2d, Week 1) 
**Obj:** StoragePort interface + SQLite adapter + migration runner + EventLog append/read + migration `0001` (events/definitions/cache portions). **In:** TDS-01/02; Blueprint §20. **Out:** `internal/storage`, `migrations/0001`, StoragePort contract-test suite. **Dep:** M1. **Risk:** WAL/durability subtleties. **Val:** §20 checkpoint (crash mid-write; committed events survive — NFR-R-02). **AC:** contract suite green; append 100K events < spec targets. **DoD:** contract suite documented as the V2-Postgres reuse artifact. **Merge:** CI + contract green. **RB:** revert PR. **Repo:** durable event recording exists.

### M3 — State Projection & Registry (2d, Week 1)
**Obj:** StateStore (upsert/get/list/claim + optimistic version), WorkflowRegistry (immutable semver registration), StepResultCache (TTL), `rebuild-state` as a library function. **In:** M2; Blueprint §9. **Out:** completed StoragePort implementation. **Dep:** M2. **Risk:** rebuild fidelity. **Val:** replay 10K-event fixture → projection byte-identical (NFR-R-03). **AC:** all 12 StoragePort methods contract-tested; re-registering an existing id+version fails. **DoD:** rebuild documented. **Merge:** CI green. **RB:** revert PR. **Repo:** full persistence layer complete.

### M4 — Intelligence Seam (1.5d, Week 1, parallel Track B)
**Obj:** IntelligencePort in `sdk`, NullAdapter (fixture/deterministic responses, `IsAvailable()=false`, "The record is silent."), CapabilityRouter with fallback-chain + model_hint selection and the required/fallback decision tree (Blueprint §17). `classify` present as non-callable placeholder (FR-IL-10). **In:** Blueprint §13–17. **Out:** `internal/intelligence` + `adapters/null`. **Dep:** M1 only. **Risk:** over-building the router — V1 router may be internally simple while honoring the public routing spec (Finalization open item). **Val:** router unit tests for every branch of the §17 decision tree. **AC:** context-budget rejection (not truncation) enforced (FR-IL-08). **DoD:** IntelligencePort contract suite exists. **Merge:** CI green. **RB:** revert PR. **Repo:** intelligence seam ready; no cloud code anywhere.

### M5 — Expression Engine & Validator (2d, Weeks 1–2, parallel Track C)
**Obj:** Hand-written implementations of the two Finalization grammars (template resolver; condition parser/evaluator with null semantics and hyphenated identifiers) + WorkflowValidator (orphans, cycles, handler refs, fallback refs, transition refs, grammar validation, trigger filters, intelligence field checks — FR-WD-03, PRD §18 checklist). **In:** TDS-03. **Out:** `internal/expr`, `internal/validate`. **Dep:** M1. **Risk:** IR-2. **Val:** full conformance corpus + fuzzing, zero panics. **AC:** every prohibited construct rejected with precise position; missing template path → `""` + warning (FR-WD-05). **DoD:** grammar docs cross-linked. **Merge:** CI green. **RB:** revert PR. **Repo:** all workflow-definition validation logic exists, engine-independent.

### M6 — Execution Engine Core (4d, Week 2)
**Obj:** The pull loop (SCAN→CLAIM→DISPATCH→SETTLE per Blueprint §8), NativeRunner + IntelligenceRunner dispatch, transition evaluation incl. fan-out concurrency within `max_parallel`, RetryPolicy with backoff, fallback activation, compensation (reverse order, own retries, `compensation_failed` + audit), cancellation FSM verbatim from Finalization Blocker 4, idempotency-key check against StepResultCache, structured JSON logging. **In:** M2–M5. **Out:** `internal/engine`, `internal/runner/{native,intelligence}`. **Dep:** M2,M3,M4,M5. **Risk:** highest-complexity milestone; concurrency defects. **Val:** §20 checkpoint; `-race` clean; event sequences vs Finalization fixtures (e.g., the cancellation event sequence). **AC:** AWIS-E1 gate live in CI and green; every §9 event type emitted correctly. **DoD:** engine package doc explains the tick pipeline. **Merge:** CI + race + E1 green. **RB:** revert PR. **Repo:** native workflows execute end-to-end with retry/fallback/compensation/cancellation.

### M7 — Signal Subsystem (2d, Week 2) → **GATE G2**
**Obj:** WAIT steps, `wait_records`, signal inbox, **single-transaction atomic delivery exactly per Blocker 3 SQL** (idempotency guard, EventLog append, optimistic-lock transition), timeout actions (fail/compensate/continue), wait-record cleanup on cancel; migration `0002`. **In:** M6; Finalization Blocker 3/4. **Out:** `internal/signal`. **Dep:** M6. **Risk:** IR-3. **Val:** crash-injection between each write pair; EventLog-authoritative invariant holds; double-delivery is a no-op. **AC:** NFR-R-04 exactly-once; delivery ≤ 200ms (NFR-P-04, bench informational). **DoD:** G2 review passed against Finalization text. **Merge:** G2 approved. **RB:** revert PR + drop 0002 via fold-forward. **Repo:** long-running human-in-the-loop workflows fully supported.

### M8 — SDK Public Surface (2d, Week 3)
**Obj:** `NewRuntime`/`Config`, handler + workflow registration (with PRD §18 error format), WorkflowBuilder (+semver check at Build), WorkflowRunner (Submit/Signal/Status/Cancel/List), RecallAPI (QueryHistory/StepStats). **In:** M6/M7; Blueprint §12. **Out:** completed `sdk` package. **Dep:** M6, M7. **Risk:** surface creep — anything not in Blueprint §12/§25 is rejected. **Val:** an example application in `examples/` builds against `sdk` only. **AC:** FR-SDK-03/04/05/09 verified; `internal/` unimportable from the example. **DoD:** godoc complete, one runnable example per public type. **Merge:** CI green + API read-through review. **RB:** revert PR. **Repo:** applications can be written.

### M9 — Test Infrastructure (2d, Week 3, parallel)
**Obj:** `sdk/testing`: WorkflowTestHarness, DeterministicMode, MockIntelligence. **In:** M8. **Out:** the developer test loop; QG-5 satisfied. **Dep:** M8. **Risk:** harness diverging from real engine — harness drives the *real* engine with deterministic sources, never a re-implementation. **Val:** full workflow incl. WAIT/signal in a unit test < 1s. **AC:** FR-SDK-06/07/08. **DoD:** integration suites of §19 migrated onto the harness. **Merge:** CI green. **RB:** revert PR. **Repo:** deterministic CI for all future workflow work.

### M10 — YAML DSL (2d, Week 3, parallel)
**Obj:** YAML parser → WorkflowDefinition; auto-discovery of `./workflows/*.yaml`; validation error rendering with file/line/example (PRD §18 formats). **In:** M5, M8; TDS-02. **Out:** `internal/dsl`. **Dep:** M5, M8. **Risk:** YAML/Go struct equivalence gaps. **Val:** round-trip equivalence test (FR-WD-02); the three example workflows + OIP's capture/recall YAML (pre-written as fixtures) parse and validate. **AC:** FR-WD-01..15 (validate-only path works without runtime). **DoD:** DSL docs with the full annotated example. **Merge:** CI green. **RB:** revert PR. **Repo:** both definition tiers complete and provably equivalent.

### M11 — SubprocessRunner + `awis-step` (2d, Week 4)
**Obj:** TDS-04 protocol; SubprocessRunner (spawn, JSON exchange, timeout, error mapping); Python `awis-step` library with `@step` decorator + `step.serve()`; pytest suite against golden protocol files. **In:** Blueprint §25. **Out:** `internal/runner/subprocess`, `python/awis-step`. **Dep:** M6 (M8 for e2e test). **Risk:** protocol ambiguity — golden files are the single wire truth for both sides. **Val:** Python step executes inside a harness workflow; outputs flow to the next step. **AC:** FR-SE-02, FR-SDK-10. **DoD:** TDS-04 finalized. **Merge:** CI green (incl. pytest). **RB:** revert PR. **Repo:** polyglot steps work.

### M12 — Plugin System + `awis-plugin` (3d, Week 4)
**Obj:** Full §18 plugin scope: TDS-05, lifecycle FSM, JSON-RPC 2.0, handshake/manifest validation, crash auto-restart (≤3), idle kill/respawn, migration `0003`, PluginRunner, Python `awis-plugin` + `testing.mock_request`. **In:** Blueprint §11; M11 protocol experience. **Out:** `internal/plugin`, `python/awis-plugin`. **Dep:** M11. **Risk:** lifecycle edge cases (crash during handshake; kill during call). **Val:** §20 checkpoint (external kill mid-call → restart → retry OK). **AC:** FR-PS-01..06, 14, 15; NFR-S-02 (no runtime.db access — verified by minimal env + no fd inheritance). **DoD:** plugin developer guide section. **Merge:** CI green. **RB:** revert PR + fold-forward 0003. **Repo:** external capabilities pluggable in any language.

### M13 — `git-context-plugin` (1d, Week 4, parallel tail)
**Obj:** Reference Python plugin: `git.context.assemble`, `git.diff.fetch`; pinned deps; per-plugin venv install path. **In:** M12; Blueprint §11 manifest example. **Out:** `plugins/git-context-plugin/`. **Dep:** M12. **Risk:** PR-5 (Python deps). **Val:** plugin test harness offline; then real call against this repo's git history. **AC:** FR-PS-13; capabilities callable from a harness workflow. **DoD:** plugin README. **Merge:** CI green. **RB:** revert PR. **Repo:** the OIP capture workflow's plugin dependency exists.

### M14 — Core CLI (2d, Weeks 4–5, parallel)
**Obj:** TDS-07; commands: `start` (auto-discovery, startup header incl. intelligence level), `stop` (graceful), `version`, `submit` (+`--input`,`--wait`), `signal`, `cancel`, `status` (+`--watch`), `trace` (+`--json`,`--full`), `workflow validate|list|show`; shared error-rendering (what/where/what-now); `--json` everywhere; migration `0004` (audit_log) with audit events for registration/signal/config. **In:** M6–M10; PRD §15–20, §26–27. **Out:** usable `awis` binary; golden-output tests. **Dep:** M6,M7,M8,M10. **Risk:** IR-5. **Val:** system test: real binary lifecycle incl. crash-recovery. **AC:** PRD §32 Runtime-Management + Execution + Validation checklists (those not requiring init/plugins). **DoD:** `docs/CLI.md` for shipped commands. **Merge:** CI green. **RB:** revert PR. **Repo:** the development loop (submit→status→trace→signal) is live.

### M15 — OIP on AWIS (4d, Week 5) → **GATE G3**
**Obj:** The V1 validation event. TDS-06 (Record format — human sign-off first); `apps/oip` module: `RecordAppendHandler` (writes `.decisions/entries/*.md` + `oip.db` entries/FTS in one operation), `IndexFTSHandler`, `SemanticRankHandler` (pass-through when no vectors — CONTRA-3), `rebuild-index` workflow; `capture-decision` + `recall-decision` YAML exactly per Blueprint §7/§10; OIP entrypoint registering via `sdk` only. **In:** M8–M14; Constitution Title II/III; Blocker 5/6. **Out:** OIP running on AWIS; boundary evidence artifact. **Dep:** M9, M10, M13, M14. **Risk:** R2/PR-7 — the entire reason this milestone exists. **Val:** both workflows green in harness AND via CLI (submit → signal `entry_confirmed` → entry appended → recall finds it); `git log --stat` boundary proof. **AC:** QG-3 (all OIP workflows pass with NullAdapter), QG-4 (zero platform modification). **DoD:** G3 verdict recorded; TDS-06 signed. **Merge:** G3 approved. **RB:** if surgery was needed: revert nothing — open a scoped platform-fix milestone (the boundary moves, not the application) and re-run G3. **Repo:** the platform is validated by a real application.

### M16 — AnthropicAdapter (1.5d, Week 6, parallel)
**Obj:** AnthropicAdapter: Draft (haiku/sonnet per model_hint), Synthesize (sonnet), Classify (haiku placeholder wiring); CloudRetryPolicy (429/5xx, Retry-After); adapter/model/tokens recorded in StepCompleted (FR-IL-09); config + env-var wiring; secret masking in `config show`/logs/export. **In:** Blueprint §14/§16; M4 contract suite. **Out:** `internal/intelligence/adapters/anthropic`. **Dep:** M4 (+M14 for config UX). **Risk:** R4 — interface stays frozen; provider quirks live inside the adapter. **Val:** contract suite vs recorded fixtures; manual live smoke (non-CI). **AC:** FR-IL-02/05..09; NFR-S-01; with API key: trace shows adapter+tokens; without: NullAdapter fallback path unchanged. **DoD:** provider config docs. **Merge:** CI green (fixtures only). **RB:** revert PR — platform runs identically without it (Article 32, by construction). **Repo:** intelligence power-levels 0 and 1 both real.

### M17 — Full CLI + Init (2.5d, Week 6, parallel)
**Obj:** `init` (scaffolding per FR-RM-01 via go:embed), `history`, `logs`, `metrics`, `recall` (+`--synthesize`, empty-state per PRD §21), `replay` (dry-run), `audit`, `plugin install|list|status|remove`, `config show|set|validate|edit`, `rebuild-state`, `export`, `prune-events --dry-run`; migration `0005` (recall FTS). **In:** M14 patterns; PRD §15/§21/§22. **Out:** complete PRD §15 command tree. **Dep:** M14, M12 (plugin cmds), M16 (synthesize path). **Risk:** breadth — mechanical after M14's patterns; ideal AI-assisted batch. **Val:** golden-output tests for each; `awis init` → 5-minute path rehearsal. **AC:** PRD §32 remaining checklists (plugins, intelligence, observability, storage, security). **DoD:** `docs/CLI.md` complete. **Merge:** CI green. **RB:** revert PR; fold-forward 0005. **Repo:** the full V1 product surface exists.

### M18 — Hardening & v1.0.0 Release (3d, Week 6) → **GATE G4**
**Obj:** Execute QG-1..5 literally on clean macOS + Linux; NFR benchmark suite (100K-event fixtures; NFR-P-01..09, CONTRA-2 thresholds); one week of OIP dogfooding overlapping this milestone (E1-equivalent metrics collected per Blueprint §33 V1 gate — application metric, recorded not gating, IR-6); fix-only changes; `docs/V1_BENCHMARKS.md`; tag `v1.0.0`; goreleaser artifacts. **In:** everything. **Out:** the release. **Dep:** M15, M16, M17. **Risk:** benchmark misses → targeted fixes only, no redesign. **Val:** G4 checklist. **AC:** all PRD §31 quality gates + §32 Must-Have criteria checked. **DoD:** release notes; memory/docs updated. **Merge:** G4 approved; tag pushed. **RB:** bad release superseded by patch tag. **Repo:** production-ready V1; migrations now append-only forever.

---

## 28. AI MODEL ALLOCATION MATRIX

Optimization rule (per mandate): expensive models on decisions and semantics; cheaper models on well-specified breadth; humans on irreversibility.

| Milestone | Allocation | Justification (reasoning depth / complexity / architectural sensitivity / verification burden) |
|---|---|---|
| M0 | Sonnet | Mechanical scaffolding; low sensitivity |
| M1 | **Opus + Human sign-off** | Maximum architectural sensitivity — the irreversible artifact; deep cross-document verification |
| M2, M3 | Sonnet, Opus review on EventLog append/rebuild paths | Well-specified SQL/CRUD; the append path and rebuild fidelity deserve expensive review |
| M4 | Sonnet | Interface transcription + straightforward routing tree |
| M5 | **Opus** | Parser correctness against formal grammars; high verification burden (fuzzing, corpus) |
| M6 | **Opus** | Highest implementation complexity: concurrency, FSMs, compensation/cancellation semantics |
| M7 | **Opus + Human (G2)** | Atomicity is a correctness cliff; crash-injection verification |
| M8 | Mixed (Opus API review, Sonnet implementation) | Public-surface stability is sensitive; implementation is plumbing |
| M9 | Sonnet | Wiring deterministic sources through an existing engine |
| M10 | Sonnet | Parser against a fixed schema with an equivalence oracle |
| M11 | Sonnet | Protocol from TDS + golden files on both sides |
| M12 | Mixed | Lifecycle FSM edge cases (Opus); Python lib + registry (Sonnet) |
| M13 | Sonnet/Haiku | Small, well-specified reference plugin |
| M14 | Sonnet | Broad but pattern-driven; golden fixtures constrain it |
| M15 | **Mixed + Human-only verdict** | Handlers/YAML are Sonnet-grade; TDS-06 and the G3 boundary judgment are human decisions no model renders |
| M16 | Sonnet | Single adapter behind a frozen interface with a contract suite |
| M17 | Sonnet/Haiku batch | Mechanical breadth over established patterns |
| M18 | Mixed + Human (G4) | Benchmarks/fixes are Sonnet; ship/no-ship is human |

---

## 29. ESTIMATED ENGINEERING EFFORT

| Bucket | Milestones | Effort (days) |
|---|---|---|
| Bootstrap & schema freeze | M0–M1 | 3 |
| Persistence + seams + expressions | M2–M5 | 7.5 |
| Engine + signals | M6–M7 | 6 |
| SDK + testing + DSL | M8–M10 | 6 |
| Polyglot + plugins | M11–M13 | 6 |
| CLI (core + full) | M14, M17 | 4.5 |
| OIP validation | M15 | 4 |
| Cloud intelligence | M16 | 1.5 |
| Hardening & release | M18 | 3 |
| **Total serial effort** | | **≈ 41.5 dev-days** |
| **Wall-clock with 2–3 tracks (§11) + AI assistance** | | **≈ 30 working days = the PRD's 6-week envelope** |

Confidence: *Reasoned* (Tier 3). The week-level envelope is Tier-0 (PRD-authoritative); the per-milestone split is this tribunal's engineering judgment. Buffer lives in the float milestones (§10), not in padded estimates. If the envelope is breached, scope does not grow and the frozen architecture does not change — the calendar does.

---

## 30. FINAL END-TO-END ROADMAP

```
Day 1        M0  Bootstrap                                   repo green
Days 1–3     M1  Schema & format freeze          ── G1 ──►   irreversible artifacts approved
Week 1       M2→M3 Storage        ║ M4 Intelligence ║ M5 Expressions
Week 2       M6  Execution engine → M7 Signals   ── G2 ──►   AWIS-E1 gate permanent
Week 3       M8  SDK → M9 Harness ║ M10 YAML DSL
Week 4       M11 Subprocess → M12 Plugins → M13 git-context  ║ M14 Core CLI begins
Week 5       M15 OIP on AWIS                    ── G3 ──►   platform boundary verdict
Week 6       M16 Anthropic ║ M17 Full CLI+init → M18 Harden ── G4 ──► v1.0.0
```

**Production-readiness checkpoints:** G1 (formats), G2 (execution correctness), AWIS-E1 (permanent zero-AI gate), G3 (boundary), QG-1..5 + NFR benchmarks + dogfood week (G4). After v1.0.0: V2 planning begins **only** on evidence from OIP's dogfood metrics, per the frozen roadmap (Blueprint §33) — no V2 work is pre-built in V1.

Every milestone above is independently buildable, testable, reviewable, and mergeable; each leaves the repository healthier than before; none exists in a partially-integrated state. The plan introduces zero new capabilities, zero new abstractions, and zero architectural reinterpretation. Four documented Tier-0 contradictions (CONTRA-1..4) are recorded with dispositions; none required design change.

---

**IMPLEMENTATION MASTER PLAN APPROVED**

> "The platform is implementation-ready. Begin Milestone 1. No additional planning documents should be created unless implementation produces new evidence that invalidates the frozen architecture."
