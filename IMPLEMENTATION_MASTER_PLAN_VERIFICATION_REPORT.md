# IMPLEMENTATION MASTER PLAN — VERIFICATION REPORT
## Final Engineering Verification Before Construction

**Document Status:** Verification record — binding amendments F-1..F-5 adopted herein
**Version:** 1.0
**Date:** 2026-07-02
**Produced by:** Independent Software Engineering Verification Tribunal
**Subject under test (Tier 1 hypothesis):** IMPLEMENTATION_MASTER_PLAN.md v1.0
**Tier 0 authorities:** OIP_CONSTITUTION.md → AWIS_ARCHITECTURE_FINALIZATION.md → AWIS_ARCHITECTURE_BLUEPRINT.md → AWIS_PRD.md
**Method:** Every milestone, dependency edge, boundary, gate, and rollout claim was independently challenged against the Tier 0 texts (re-read in full during this verification) and against battle-tested engineering practice. Nothing received the benefit of the doubt. Epistemic labels used throughout: **Verified** (checked against Tier 0 text or mechanical fact) / **Supported** (follows tightly from verified premises) / **Reasoned** (engineering judgment) / **Unknown**.

---

## 1. EXECUTIVE VERDICT

The Implementation Master Plan is **structurally sound and executable**, with **zero critical findings and zero architectural drift**. The tribunal attempted to invalidate all 19 milestones, the dependency DAG, the critical path, the repository model, the migration sequence, and all four human gates. The plan survived, with **two MEDIUM plan-level gaps (F-1, F-2), one MEDIUM specification gap with a Tier-0-sanctioned resolution path (F-3), and two LOW sequencing corrections (F-4, F-5)** — all correctable as milestone-scope amendments measured in hours, none touching the frozen architecture, none blocking Milestone 0/1.

The five corrections are adopted as **binding amendments** in §28 of this report. With them in force, implementation may begin immediately.

**Verdict: VERIFIED — approved to begin Milestone 1** (full statement in §31).

---

## 2. OVERALL READINESS ASSESSMENT

| Quality Gate | Result | Basis |
|---|---|---|
| Architecture Integrity | **PASS** | No frozen interface, boundary, or ownership altered; the four CONTRA dispositions are documented tensions, not silent resolutions (§24) — Verified |
| Dependency Integrity | **PASS with amendment F-1** | One package-level import cycle in the stated package rules; mechanical fix defined (§22) — Verified |
| Repository Integrity | **PASS** | Two-module monorepo mechanically enforces FR-SDK-09/QG-4 via Go's path-based `internal/` rule (§6) — Verified |
| Testing Strategy | **PASS** | Full pyramid, zero-network CI, permanent AWIS-E1 gate from M6, contract suites designed for V2 reuse (§17) — Verified |
| Migration Safety | **PASS with amendments F-2/F-4** | Forward-only + pre-tag fold-forward is battle-tested; two additive entries re-sequenced (§12) — Supported |
| Integration Safety | **PASS** | Every milestone integrates into a running system; no big-bang merge exists anywhere in the plan — Verified |
| Rollback Capability | **PASS** | One milestone = one squash PR = one revert; EventLog-authoritative recovery tool (`rebuild-state`) deliberately built at M3, before anything that could need it — Verified |
| Production Readiness | **PASS** | QG-1..5 + NFR benchmarks + dogfood week gate the tag at G4 (§20) — Verified |
| Solo-Founder Sustainability | **PASS** | Review-sized milestones, float-bearing schedule, expensive-model budget on the four decisions that matter (§26–27) — Supported |

Readiness: **HIGH**, conditional only on the §28 amendments (all plan-level).

---

## 3. MILESTONE VERIFICATION

Each of M0–M18 was tested against all ten verification rules (independently buildable / testable / reviewable / mergeable / measurable progress / repo healthier / explicit DoD / rollback path / validation criteria / no hidden dependencies).

| Milestone | Verdict | Notes |
|---|---|---|
| M0 Bootstrap | PASS | Rollback "n/a (first commit)" is acceptable for a repo-creating milestone |
| M1 Schema Freeze | PASS (amended by F-1) | Correctly code-free; G1 placement is the plan's strongest single decision (P8/R1). F-1 adds one structural instruction for where canonical types live |
| M2 Storage Foundation | PASS | AC "append 100K events < spec targets" is vague (no throughput NFR exists) — OBSERVATION O-3; contract-suite-first design is exactly right for V2 Postgres reuse |
| M3 State Projection | PASS | `rebuild-state` scheduled before the engine exists — correct lowest-regret ordering |
| M4 Intelligence Seam | PASS | Correctly scoped to Null + router; honors the Finalization's router-simplification allowance |
| M5 Expression Engine | PASS | Corpus-before-parser + fuzzing directly mitigates IR-2; leaf-package placement makes it exhaustively testable |
| M6 Execution Engine | PASS (amended by F-2) | Hidden dependency found: Blueprint §8 tick step 2 (SCAN_TRIGGERABLE) and §10 DomainEvent consumption are in the frozen loop but absent from M6's scope text — see F-2 |
| M7 Signal Subsystem | PASS (amended by F-4) | Crash-injection verification between each write pair is the correct proof obligation for Blocker 3; audit_log call site `SignalDelivered` originates here — see F-4 |
| M8 SDK Public Surface | PASS (amended by F-1, F-2) | TriggerAPI appears in the Blueprint's SDK layer (§4) and `trigger.go` in §12/§25 but M8's objective line omits the event-intake surface — folded into F-2 |
| M9 Test Infrastructure | PASS | "Harness drives the real engine, never a re-implementation" is the critical anti-divergence rule and is stated explicitly |
| M10 YAML DSL | PASS | Over-serialized on M8 (needs only M1 types + M5 validator to start) — OBSERVATION O-1, float opportunity, not an error |
| M11 SubprocessRunner | PASS | Golden-files-as-single-wire-truth for both Go and Python sides is battle-tested |
| M12 Plugin System | PASS | Lifecycle FSM edge cases correctly flagged as the risk; offline Python test harness satisfies FR-PS-15 |
| M13 git-context-plugin | PASS | Correctly minimal |
| M14 Core CLI | PASS (amended by F-3, F-5) | The CLI↔runtime process-interaction model is unspecified — the plan's most significant specification gap (F-3); `plugin install` needed earlier than M17 for M15's CLI-path validation (F-5) |
| M15 OIP on AWIS | PASS | TDS-06 human sign-off before handler code honors the "irreversible artifact" doctrine; G3 evidence definition refined by O-2 |
| M16 AnthropicAdapter | PASS | Fixture-only CI + manual live smoke is correct; rollback claim ("platform runs identically without it") is architecturally guaranteed by Article 32/P2 — Verified |
| M17 Full CLI + Init | PASS | Correctly identified as the AI-assisted mechanical batch; soft dependency on M16 (synthesize path works against NullAdapter empty-state) — float, noted |
| M18 Hardening & Release | PASS | Week-6 load is 7 dev-days across parallel tracks in a 5-day week — F-6 (LOW) schedule pressure; plan's own float rules absorb it |

**No milestone is rejected.** Every milestone satisfies all ten rules once amendments F-1..F-5 are applied to their scope text.

---

## 4. DEPENDENCY GRAPH AUDIT

- **Edge correctness:** all stated edges re-derived from milestone inputs/outputs; every edge is real (no phantom dependencies except O-1's over-serialization of M10, which is conservative, not wrong). — Verified
- **Missing edges found:** (a) M6→(F-2 trigger scope) requires the DomainEvent storage entry created in the same milestone — internal to M6 after amendment; (b) M15→M14 is correctly present for the CLI-path AC but M15's *plugin availability* depends on either F-5's early `plugin install` or SDK-level registration — amendment resolves the ambiguity. — Verified
- **Orphan check:** no milestone lacks a consumer; no deliverable is produced twice. — Verified
- **Fan-in stress:** M15 has the highest fan-in (M9, M10, M13, M14) — appropriate for the validation event; all four inputs complete ≥ 2 days before M15 starts under the stated calendar. — Supported

**Result: PASS** (two edges clarified by amendments).

---

## 5. CRITICAL PATH AUDIT

- Arithmetic re-verified: 0.5+2.5+2+2+4+2+2+2+3+1+4+3 = **28 days** as claimed. — Verified
- Path membership re-derived: the plugin chain (M11→M12→M13) is on the path solely due to OIP's plugin-typed step; the stated contingency (stub the capability, fold the real plugin in before G3 renders its verdict) is legitimate because G3's question is about *platform surgery*, not plugin completion. — Supported
- Near-critical path: M14 (2d) must complete before M15's CLI-path AC; it has ~3 days of float in week 4–5 — adequate. — Supported
- **F-6 (LOW):** Week 6 carries M16 (1.5d) + M17 (2.5d) + M18 (3d) = 7 dev-days against 5 working days even with two tracks, because M18 depends on both. Smallest correction: exercise the plan's own float rule — begin M16 (depends only on M4) and the mechanical portions of M17 in week-5 float. No calendar or scope change required. — Reasoned

**Result: PASS** with F-6 noted.

---

## 6. REPOSITORY STRUCTURE AUDIT

- **The two-module decision is the plan's second-strongest decision.** Go's `internal/` visibility is import-path-based; `github.com/awis/oip` importing `github.com/awis/awis/internal/...` is a compile error. QG-4's honesty is enforced by the toolchain. — Verified (Go specification fact)
- Monorepo + go.work matches Tier 3 reality (one engineer, one CI, one review queue). — Supported
- CONTRA-1 (module path) is honestly documented with a low-cost V2 disposition; **O-4:** the PRD's literal `go install github.com/awis/awis@latest` additionally implies a root-level `main` package or a documented `.../cmd/awis` install path — fold this one line into the existing CONTRA-1 EDR at M0. — Verified (Go toolchain behavior)
- `.gitignore` coverage, embedded examples, and docs/ placement all conform to PRD §23/§25 layouts. — Verified

**Result: PASS.**

---

## 7. PACKAGE BOUNDARY AUDIT

- Downward-only dependency rule audited package by package. **One genuine defect found — F-1 (MEDIUM):** the plan states both "`sdk` defines the public types — internal packages depend on `sdk` types" and "`cmd/awis ──► sdk ──► internal/engine`". In Go, imports are package-level: `sdk → internal/engine → sdk` is a **compile-time import cycle**. As written, M8 would fail to build. — Verified (mechanical)
- **Smallest viable correction (adopted, §28):** canonical structs/interfaces live in a leaf package `internal/core`; `sdk` re-exports them via type aliases (`type Step = core.Step`), preserving the frozen public surface (FR-SDK-02/03) byte-for-byte while `sdk` freely imports `internal/engine` for `NewRuntime`. Applied during M1 when the types are born. Zero interface change; zero architecture change; standard Go layering practice. — Supported
- All other boundary claims (adapters terminal, `expr` leaf, `dsl` produces-only) verified consistent. — Verified

**Result: PASS after F-1.**

---

## 8. MODULE BOUNDARY AUDIT

The IMP's module table was diffed against the Blueprint's five layers + SDK/CLI/OIP: **exact correspondence; nothing added, merged, or split** — with one omission: the **Trigger subsystem** (Blueprint §5 Layer 1 "Scheduler", §8 SCAN_TRIGGERABLE, §10 DomainEvents) has no owning milestone. This is **F-2 (MEDIUM)** — a Must-Have requirement family (FR-WE-13) whose implementation work is unassigned, which would surface as improvised scope during M6 or as a QG failure at M18.

- Root cause: the IMP's engine milestone enumerated the §8 loop stages selectively (SCAN→CLAIM→DISPATCH→SETTLE) and dropped SCAN_TRIGGERABLE.
- Evidence: IMP §27 M6 objective text; grep of the IMP shows no assignment of DomainEvent ingestion, trigger matching, or TriggerAPI. Blueprint §10 additionally specifies domain-event retention (TTL, default 7 days) — implying a storage structure absent from both the frozen §20 enumeration *and* the IMP's migration table (same class as the already-documented CONTRA-4 additions).
- **Smallest viable correction (adopted, §28):** (a) add to M6 scope: SCAN_TRIGGERABLE stage + trigger matching with filter evaluation (evaluator already exists from M5) + `domain_events` table with TTL as an additive entry in migration 0001-or-0002 (CONTRA-4 class, StoragePort-internal); (b) add to M8 scope: the SDK event-intake surface already named in the frozen SDK layer (TriggerAPI / `trigger.go`); (c) assign FR-WE-14 (schedule trigger, Should Have) to M17 with explicit Should-Have deferability. No new abstraction — this is frozen architecture the plan under-scheduled. — Verified gap; Supported correction

**Result: PASS after F-2.**

---

## 9. CAPABILITY BOUNDARY AUDIT

- P1 (apps own logic / runtime owns execution): OIP's handlers, `oip.db`, and Record format live entirely in `apps/oip`; no AWIS package references them. — Verified
- Blocker 5 (FTS ownership): the recall FTS (migration 0005) indexes `execution_events` inside `runtime.db` — execution state, not application data; OIP's `entries_fts` untouched. FR-ST-04 preserved. — Verified
- Blocker 6 (compensation/append-only): the OIP capture YAML fixture carries no compensation block; TDS-06 encodes Article 7 append-only rules before handlers exist. — Verified
- Plugin boundary (FR-PS-06): minimal env, no fd inheritance, protocol-only communication — stated and testable. — Verified

**Result: PASS.**

---

## 10. INTERFACE CONTRACT AUDIT

All seven contracts in IMP §13 were diffed against their Blueprint definitions: **method-for-method identical**; implementation order is dependency-correct (StoragePort before engine, IntelligencePort before IntelligenceRunner, wire protocols before their runners). `classify` correctly retained as the mandated non-callable placeholder (FR-IL-10). The post-M8 change-control rule (written justification citing a frozen document) is an appropriately cheap stability mechanism for a solo founder. — Verified

**Result: PASS.**

---

## 11. TECHNICAL DESIGN SPECIFICATION AUDIT

- TDS-01..07 are transcriptions of frozen content, correctly front-loaded before their consuming modules, each with a named blocking relationship. — Verified
- TDS-06 (OIP Record format) receiving G1-equivalent human reverence is constitutionally correct (Articles 5–13; "the format is the UNIX inode"). — Verified
- **Gap folded into F-3:** TDS-07 is scoped to *output formats and error taxonomy* only. It must also specify the **CLI↔runtime interaction model** (see §15). — Verified gap

**Result: PASS after F-3 scope extension.**

---

## 12. DATABASE MIGRATION AUDIT

- Sequential, embedded, forward-only, auto-applied locally (FR-ST-06); N−1 fixture testing; fold-forward only before the v1.0.0 tag, append-only after — battle-tested and correctly reasoned for a greenfield repo. — Verified
- `cancellation_requested` folded into 0001: correct — the Finalization's ALTER described a spec delta; no deployment predates this repository. — Verified
- **F-4 (LOW):** `audit_log` is scheduled as migration 0004 in M14, but its write sites are created earlier: `SignalDelivered` (M7), `WorkflowRegistered` (M8), `PluginRegistered` (M12). As planned, three milestones merge without audit hooks and get retrofitted in M14 — avoidable rework and a window where NFR-S-05 events are silently unrecorded. Root cause: migration grouped with the CLI that *reads* it rather than the subsystems that *write* it. **Smallest viable correction (adopted):** move the audit table + internal append API to M7 (renumbering the pre-implementation migration sequence, which is free before any code exists); each subsequent milestone adds its own call site; `awis audit` (read path) stays in M17. — Supported
- F-2 adds the `domain_events` (+TTL) entry to the sequence (§8). — Supported

**Result: PASS after F-2/F-4 re-sequencing.**

---

## 13. API ROLLOUT AUDIT

- V1's API = CLI + embedded SDK: conforms to frozen scope (server mode V2). — Verified
- `--json` from day one (PP-6, no retrofit) and the shared what/where/what-now error renderer built first in M14 are the right ordering. — Supported
- Core-CLI-in-week-5 vs the PRD's week-6 "CLI (all commands)": verified as a legitimate execution refinement — the PRD's table is week-granular; all commands still complete in week 6 (M17); no scope moved. — Verified

**Result: PASS.**

---

## 14. SDK ROLLOUT AUDIT

- Types (M1) → surface (M8) → test infra (M9) → DSL equivalence (M10) is dependency-correct and gives the YAML tier a mechanical equivalence oracle (FR-WD-02). — Verified
- QG-5 as M9's acceptance test binds the milestone to a shipped quality gate. — Verified
- Python libraries correctly deferred to their consuming milestones; PyPI publication correctly identified as a non-gate. — Supported
- F-1's alias mechanism preserves FR-SDK-02's file/identifier surface exactly. — Verified

**Result: PASS.**

---

## 15. CLI ROLLOUT AUDIT

- **F-3 (MEDIUM) — the plan's most significant specification gap.** The IMP never specifies how a second-process CLI interacts with a running foreground runtime: `awis stop` (FR-RM-04), `submit`/`signal`/`cancel` from another terminal, and `awis logs --tail` all require a defined mechanism. The Tier 0 texts constrain but do not fully settle this: Blueprint §28 Mode 1 sanctions "embedded Go SDK (same process) **or via local socket** (if the application is a separate process)," while Blueprint §9 states local mode has "no concurrent writers" — so unmediated multi-process writes to `runtime.db` would sit in tension with the frozen text, even though SQLite WAL mechanically permits serialized multi-process writing.
  - Root cause: TDS-07 was scoped to output formats; the interaction model fell between M6 (engine) and M14 (CLI).
  - Severity: MEDIUM. Impact: improvised design on the near-critical path at M14; signal-latency AC (≤200ms, NFR-P-04) is measured across this exact boundary. Probability of surfacing: certain (M14, day 1). Confidence: High.
  - **Smallest viable correction (adopted):** extend TDS-07's mandate to specify the CLI↔runtime interaction model at M14 day 1, choosing between the two mechanisms already present in Tier 0 — (a) the Blueprint-sanctioned local socket, or (b) direct SQLite access under WAL with busy-timeout, which would require *documenting a refinement* of the §9 "no concurrent writers" phrasing (a Tier-0 tension to be recorded CONTRA-style, never silently resolved). `stop` additionally requires a PID-file + SIGTERM convention (already consistent with FR-RM-06). This is a specification task, not new architecture: both candidate mechanisms are named in the frozen Blueprint. — Verified gap; Supported correction
- **F-5 (LOW):** M15's acceptance criterion runs OIP "via CLI," and the capture workflow's first step needs `git-context-plugin` registered — but `awis plugin install` is scheduled in M17. Root cause: plugin CLI grouped with management breadth rather than with its first consumer. **Smallest viable correction (adopted):** move the minimal local-path `awis plugin install` (and `plugin list`) from M17 to M14; `plugin status/remove` stay in M17. — Verified gap; Supported correction

**Result: PASS after F-3/F-5.**

---

## 16. PLUGIN STRATEGY AUDIT

- Scope exactly matches frozen requirements (JSON-RPC 2.0, manifest, lifecycle FSM, ≤3 restarts, idle kill per the Finalization's recommendation, one reference plugin); V1 justification is concrete (OIP's plugin-typed step), satisfying the Finalization's open item. — Verified
- Deliberate absences (remote install, registry, WASM) match frozen non-scope. — Verified
- Per-plugin venv + pinned deps addresses PR-5. — Supported

**Result: PASS.**

---

## 17. TESTING STRATEGY AUDIT

- Pyramid layers map 1:1 to Blueprint §27 with concrete suites and CI-blocking dates; no layer is aspirational. — Verified
- AWIS-E1 permanent from M6 is the single most valuable gate in the plan (Article 32 as machinery). — Verified
- Shared fixture package + golden wire files as single sources of truth prevent cross-language and cross-layer drift. — Supported
- Crash-injection tests for Blocker 3 atomicity are the correct proof obligation, not merely unit tests. — Supported
- Determinism-from-M2 (injectable clocks/IDs) is what makes M9 "wiring, not surgery" — the claim is credible only because it is a stated design constraint from the start. — Supported

**Result: PASS.**

---

## 18. CI/CD AUDIT

- Gate order mirrors the mandated quality-gate ladder; zero-network CI with recorded fixtures keeps P7 honest; live-API smoke correctly quarantined off the merge path. — Verified
- CGO-disabled pure-Go SQLite keeps CI hermetic and cross-platform — consistent with the PRD's bundled-SQLite dependency note. — Verified
- Release = tag + goreleaser; `main` always releasable; artifacts immutable per tag. — Verified
- **O-5:** CI must build/test *both* Go modules once `apps/oip` exists — add the second module to the CI matrix within M15's scope (one line). — Reasoned

**Result: PASS.**

---

## 19. DOCUMENTATION AUDIT

- Specs-as-code in the PR pipeline; per-milestone doc rule bound into the DoD; QG-1 verbatim-tests the README path; ADR discipline points at the frozen Blueprint with lightweight EDRs for implementation-level choices; explicit "no new planning documents" commitment. — Verified

**Result: PASS.**

---

## 20. VERIFICATION GATE AUDIT

- G1 placement (before any code) is correct for the irreversible artifact; its checklist is concrete (schema_version, decade-reader, replay sufficiency per event type). — Verified
- G2 reviews completed M6+M7 against the Finalization text with crash-injection evidence — the right review at the right moment. — Verified
- G3: **O-2 (OBSERVATION, adopted as refinement):** the `git log --stat` boundary artifact must distinguish *defect fixes* in the platform (legitimate during M15) from *interface/boundary changes* (QG-4 failure). Root cause: evidence definition compressed. Smallest correction: G3 evidence categorizes any platform-module diff during M15 as defect-fix vs. surface-change; only the latter fails the gate. — Reasoned
- G4 gates the tag on QG-1..5 + benchmarks + dogfood; failure blocks ship. — Verified
- Gates G2–G4 do not idle the pipeline (float work proceeds) — review-bandwidth-realistic for one human. — Supported

**Result: PASS with O-2 refinement.**

---

## 21. DEFINITION OF DONE AUDIT

The seven-clause global DoD covers architecture integrity, compilation, tests, verification checkpoints, documentation, repository health, and human approval; every milestone adds specific DoD items; the DoD is enforceable (each clause is mechanically checkable or gate-checkable). "No TODO without a tracked issue" presumes an issue tracker — trivially satisfiable with repo issues; no correction needed. — Verified

**Result: PASS.**

---

## 22. CIRCULAR DEPENDENCY ANALYSIS

- **Milestone level:** DAG re-derived; acyclic. — Verified
- **Package level:** one cycle found and corrected — **F-1** (`sdk ⇄ internal/engine`, §7). After the `internal/core` + type-alias amendment: `cmd → sdk → internal/* → internal/core`, strictly acyclic. — Verified
- **Module level (Go modules):** `oip → awis(sdk)` only; platform never imports the app. Acyclic by construction. — Verified
- **Data level:** EventLog → StateStore projection is one-directional; `runtime.db` and `oip.db` share nothing. — Verified

**Result: PASS after F-1.**

---

## 23. HIDDEN ASSUMPTION ANALYSIS

Assumptions surfaced by adversarial reading, with dispositions:

| # | Hidden assumption in the IMP | Disposition |
|---|---|---|
| H-1 | A second-process CLI can reach the running runtime | Real gap → **F-3** (mechanism must be specified; both candidates already in Tier 0) |
| H-2 | DomainEvents "arrive" without an assigned ingestion path | Real gap → **F-2** |
| H-3 | `git-context-plugin` is registered before M15's CLI validation | Real gap → **F-5** |
| H-4 | `sdk` can both define types and construct the runtime | False as written → **F-1** |
| H-5 | Audit hooks can be retrofitted cheaply in M14 | Avoidable rework → **F-4** |
| H-6 | Week 6 absorbs 7 dev-days | Pressure, absorbed by documented float → **F-6** |
| H-7 | `go install github.com/awis/awis@latest` yields an `awis` binary | Path detail → **O-4**, folded into CONTRA-1's EDR |
| H-8 | Go 1.22+, Python 3.10+, clean test machines available | Verified — stated in PRD Assumptions A-1..A-3; not hidden |
| H-9 | Anthropic API availability for M16 live smoke | Verified — PRD A-4; correctly off the merge path |
| H-10 | Dogfood week can overlap M18 | Reasoned — acceptable; E1-equivalent metrics are recorded, not gating (IR-6) |

No assumption remains both hidden and unaddressed. — Supported

---

## 24. ARCHITECTURAL DRIFT ANALYSIS

Diffed the IMP against every frozen decision: the six core structures, five layers, 15 ADRs, 7 blocker resolutions, both expression grammars, the StepType enum (no `parallel`), fan-out parallelism, no-cycles rule, storage boundaries, plugin protocol, secret rules, and the V1 scope table.

- **Zero drift detected.** The IMP adds no capability, no abstraction, no V2/V3 material. — Verified
- The four CONTRA entries were re-examined: each is a *genuine* Tier-0 internal inconsistency or omission, each disposition is minimal and additive, and each is recorded rather than silently resolved — exactly as mandated. CONTRA-3's semantic-rank pass-through was specifically checked against FR-IL-11 (FTS-first) and Blocker 5: conformant. — Verified
- F-2 and F-3 are *under-scheduling and under-specification of frozen architecture*, not drift; their corrections consume only mechanisms already named in Tier 0. — Verified

**Result: PASS — no drift.**

---

## 25. TECHNICAL DEBT FORECAST

Debt the plan knowingly creates, audited for acceptability:

| Debt item | Assessment |
|---|---|
| CONTRA-1 module path deferred to V2 | Acceptable; one-line import change for applications; recorded in an EDR |
| `classify` placeholder carried through V1 | Mandated by FR-IL-10; zero-cost |
| Recall FTS as a platform-internal index (0005) | Rebuildable cache, additive; no interface exposure |
| Audit retrofit (pre-amendment) | Eliminated by F-4 |
| CLI interaction model improvisation (pre-amendment) | Eliminated by F-3 |
| Fold-forward migrations pre-tag | Standard greenfield practice; hard-stops at v1.0.0 |
| Golden-file maintenance burden | Real but small; TDS-07 freezing formats first minimizes churn |

Forecast: **LOW residual debt** after amendments. The plan creates no load-bearing shortcut. — Supported

---

## 26. SOLO-FOUNDER FEASIBILITY

- Review load: 19 PRs over 6 weeks ≈ 3/week, size-capped — sustainable. — Supported
- Cognitive load: one system, one repo, one CI; the two-module split adds negligible overhead (go.work) for a large boundary payoff. — Supported
- Effort honesty: the 41.5-day serial / 30-day wall-clock claim is labeled Reasoned in the IMP itself, with float identified and a stated "calendar slips, scope doesn't" rule — the correct failure mode for a solo founder. — Supported
- Bus-factor mitigations (R10): specs-as-code, ADR pointers, boring dependencies — present. — Verified
- Residual risk: IR-7 (review fatigue weeks 4–6) is real; F-6's week-5 pull-forward of M16/M17 also flattens the review curve. — Reasoned

**Result: PASS.**

---

## 27. AI-ASSISTED DEVELOPMENT SUITABILITY

- The allocation matrix follows the mandated rule (expensive models on decisions/semantics; cheap models on specified breadth; humans on irreversibility). Spot-checked extremes: M6/M7 (Opus-class: concurrency + atomicity semantics) and M17 (Haiku/Sonnet batch over golden fixtures) are correctly placed. — Supported
- The plan is unusually AI-executable because every mechanical milestone has a machine-checkable oracle (conformance corpus, equivalence test, golden files, contract suites) — AI output is verified by artifacts, not by trust. — Supported
- Human-only reservations (G1–G4, TDS-06, G3 verdict) are exactly the judgments models should not render. — Supported

**Result: PASS.**

---

## 28. REQUIRED CORRECTIONS

### Must Fix Before Implementation
*(Plan-level amendments, adopted as binding by this report; total effort ≈ half a day of document/scope edits, absorbed into M0/M1 without calendar impact. None alters any frozen interface, boundary, or scope.)*

**F-1 — sdk/internal import cycle (MEDIUM, Confidence: High)**
Root cause: "types in `sdk`" + "`sdk` constructs the runtime" is a package-level cycle in Go. Evidence: IMP §6 rules vs. §27 M8 objective; Go import semantics. Impact: M8 fails to compile as planned. Correction: canonical types in leaf package `internal/core`; `sdk` re-exports via type aliases; applied during M1. Public surface (FR-SDK-02/03) unchanged.

**F-2 — Trigger subsystem unassigned (MEDIUM, Confidence: High)**
Root cause: M6 enumerated the §8 loop stages selectively; DomainEvent ingestion fell through. Evidence: FR-WE-13 (Must Have), Blueprint §8 SCAN_TRIGGERABLE, §10 DomainEvents + 7-day TTL; no IMP milestone owns them. Impact: improvised scope in week 2 or a Must-Have miss at G4. Correction: amend M6 scope (trigger matching + `domain_events` table w/ TTL, CONTRA-4-class additive migration entry) and M8 scope (SDK TriggerAPI intake per the frozen SDK layer); assign FR-WE-14 (Should Have, cron) to M17 with explicit deferability.

### Should Improve During Implementation
*(Each bound to a named milestone; none blocks M0/M1.)*

**F-3 — CLI↔runtime interaction model unspecified (MEDIUM, Confidence: High; due M14 day 1)**
Correction: extend TDS-07 to specify the mechanism for `stop`/`submit`/`signal`/`cancel`/`logs` from a second process, selecting between the two Tier-0-named candidates (Blueprint §28 local socket; or direct SQLite-WAL access, which requires recording a CONTRA-style refinement of §9's "no concurrent writers" phrasing — documented, never silent). PID-file + SIGTERM for `stop` (consistent with FR-RM-06).

**F-4 — Audit log migration timing (LOW, Confidence: High; due M7)**
Correction: move `audit_log` table + append API to M7 (renumber the pre-code migration sequence); later milestones add their own call sites; read path stays in M17.

**F-5 — `plugin install` needed by M15 (LOW, Confidence: High; due M14)**
Correction: move minimal local-path `awis plugin install` + `plugin list` from M17 to M14.

**F-6 — Week-6 load (LOW, Confidence: Medium; due week 5)**
Correction: start M16 and mechanical portions of M17 in week-5 float, per the plan's own float rule.

### Future Improvements
*(Observations; adopt opportunistically.)*

- **O-1:** M10 may start after M1+M5 (drop the hard M8 edge) — extra float if week 3 compresses.
- **O-2:** G3 evidence categorizes platform diffs during M15 as defect-fix vs. surface-change; only the latter fails QG-4.
- **O-3:** Replace M2's "100K events < spec targets" with a concrete informational benchmark number.
- **O-4:** Fold the `go install` path detail into CONTRA-1's EDR at M0.
- **O-5:** Add the `apps/oip` module to the CI matrix within M15.

---

## 29. RECOMMENDED OPTIMIZATIONS

Beyond §28, the tribunal recommends **nothing**. Every other candidate "optimization" examined (earlier CLI, merged milestones, deferred plugin system, extra abstraction layers) either violated the frozen scope, enlarged a review unit past the solo-founder cap, or added complexity without measurable benefit — all rejected under the verification rules.

---

## 30. CONFIDENCE ASSESSMENT

| Claim | Level |
|---|---|
| IMP conforms to frozen architecture (no drift) | **Verified** — full Tier 0 texts diffed this session |
| Milestones satisfy all ten verification rules (post-amendment) | **Verified** |
| Dependency DAG acyclic and complete (post-F-1/F-2) | **Verified** |
| Six-week envelope achievable | **Reasoned** — labeled as such in the IMP itself; float + "calendar slips, scope doesn't" rule make the failure mode safe |
| OIP boundary validation will pass at G3 | **Unknown** — this is precisely what M15 exists to discover (R2/PR-7); the plan's contingency (the boundary moves, not the application) is the correct response to either outcome |
| E1 dogfood outcome (application value hypothesis) | **Unknown** — application-level, correctly non-gating for the platform (IR-6) |

No certainty has been manufactured. The two Unknowns are the two questions V1 is designed to answer empirically.

---

## 31. FINAL VERDICT

**"Is the Implementation Master Plan sufficiently complete, internally consistent, technically sound, and executable to begin Milestone 1?"**

**YES.** All quality gates pass with the §28 amendments adopted as binding; the amendments are plan-level scope edits (≈ half a day, absorbed at M0/M1 and the named milestones), touch no frozen interface or boundary, and none blocks the start of construction.

**IMPLEMENTATION MASTER PLAN VERIFIED**

**Approved to Begin Milestone 1**

> "Future architectural changes should now require implementation evidence rather than additional planning. Implementation becomes the primary source of truth."
