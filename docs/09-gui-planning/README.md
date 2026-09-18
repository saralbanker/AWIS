# GUI Planning

The transition plan from AWIS CLI + Engine to AWIS Platform (Engine + API Layer +
GUI Layer). Basis: repo-wide, code-verified review of `engine-hardening` @ `f004f4f`
plus the uncommitted B-31 remediation, 2026-08-30. Every claim is cited to code, a
test, a migration, or observed runtime/test behaviour and has been independently
re-verified at least once (see `GUI_MASTER_PLAN.md` for the record, including one
correction made after the fact).

## The documents

**Evidence base (2026-08-30):**

| File | What it is |
|---|---|
| [GUI_COMPREHENSIVE_SUMMARY.md](GUI_COMPREHENSIVE_SUMMARY.md) | **Master Synthesis** — single comprehensive summary containing all findings, decisions, architectures, roadmaps, and risks from all 5 documents without omissions. |
| [REPOSITORY_TRUTH_AUDIT.md](REPOSITORY_TRUTH_AUDIT.md) | **Adversarial re-check** (2026-08-30, later pass; addendum same day) — an independent falsification attempt against every other document in this folder. Nothing was disproven; seven new, additive findings surfaced (observability infra is fully absent, not partial; a YAML round-trip risk for G6; a tick-loop scaling unknown; and — added in a later addendum — this whole plan is evaluated against a branch `main` is 60 commits behind, N7). Read this after the plan itself, as the "has this held up" record. |
| [GUI_MASTER_PLAN.md](GUI_MASTER_PLAN.md) | The evidence base — engine freeze verdict, the real blocker list, event-sourcing sufficiency, founder decisions D1–D4, risk register. Read this first. |
| [GUI_PRD.md](GUI_PRD.md) | Product requirements, organized by the five delivery phases (read-only dashboard → monitoring → management → visual editor → full n8n-class), with functional requirements scoped to what the engine can actually support at each phase. |
| [GUI_ARCHITECTURE.md](GUI_ARCHITECTURE.md) | System design — process model, data layer (the `state_changes` migration), API layer, streaming, editor domain model, security posture. Formalized as ADRs for the four founder decisions. |
| [GUI_ROADMAP.md](GUI_ROADMAP.md) | The G0–G10 milestones mapped onto the five product phases, with sequencing paths and parallelization notes. |
| [GUI_DEPENDENCY_MAP.md](GUI_DEPENDENCY_MAP.md) | What blocks what — a dependency graph from founder decisions through milestones to phases, plus the blocker-resolution table. |

**Execution program (2026-09-01, synthesized from the evidence base above — no new investigation, only card-level decomposition and sequencing decisions):**

| File | What it is |
|---|---|
| [ENGINE_GUI_TRANSITION_PROGRAM.md](ENGINE_GUI_TRANSITION_PROGRAM.md) | The exact critical path, milestone structure (G0–G10 with card-level effort sums), and per-milestone acceptance gates. Start here for "what is this program." |
| [ENGINE_GUI_WORK_BREAKDOWN.md](ENGINE_GUI_WORK_BREAKDOWN.md) | Card-level work packages for every milestone — ~80 cards with descriptions, effort estimates, dependencies, acceptance criteria, and parallelization notes. Hand this directly to implementation agents. |
| [ENGINE_GUI_DECISION_RECORD.md](ENGINE_GUI_DECISION_RECORD.md) | The recommended sequencing path plus two alternatives (cheapest/fastest-to-screen, safest/decision-gated), with a tradeoff table. |
| [FINAL_VERDICT.md](FINAL_VERDICT.md) | Direct answers to the founder's brief: what must happen first, what can start now, what can be deferred, fastest paths, effort by phase, recommended next action. Read this if you only have five minutes. |

**Optimization pass (2026-09-01, second pass — challenges and cuts the execution program above; no new investigation, no new milestones):**

| File | What it is |
|---|---|
| [GUI_START_LINE.md](GUI_START_LINE.md) | The sharpest single finding in this folder: GUI engineering can start *today*, in parallel with all engine work, against a code-derived fixture — and GUI-against-real-data needs only 8 specific cards (≈2.5-3 days), not full milestones G0-G4. |
| [MINIMUM_ENGINE_PROGRAM.md](MINIMUM_ENGINE_PROGRAM.md) | Every G0-G4 card (and the G7-G10 backend cards) reclassified as MUST / SHOULD / MAY-DEFER / AFTER-GUI-START, with the milestone-level `G1→G2` dependency shown to be looser than the original roadmap implied. |
| [EXECUTION_SEQUENCE.md](EXECUTION_SEQUENCE.md) | The actual staffed, ordered runbook — two parallel tracks from day one, staged to a live dashboard in ~4-5 days and a view-only beta in ~2.5-3 weeks. |
| [DEFERRED_WORK_REGISTER.md](DEFERRED_WORK_REGISTER.md) | Everything parked, with the exact trigger condition that pulls it back into scope — not just "later." |
| [FINAL_RECOMMENDATION.md](FINAL_RECOMMENDATION.md) | What to work on next, what not to, fastest path to a usable GUI, fastest path to beta. Read this if you only have two minutes. |

**Implementation preparation (2026-09-01, third pass — turns the 8-card gate above into 6 code-verified, execution-ready cards; no new investigation, no code written):**

| File | What it is |
|---|---|
| [GUI_STARTLINE_CARDS.md](GUI_STARTLINE_CARDS.md) | The exact 6 cards (down from 8 — one false dependency and two safe merges found by grounding every claim in real file:line citations), each with objective/inputs/outputs/files/acceptance/validation/dependencies. |
| [IMPLEMENTATION_ORDER.md](IMPLEMENTATION_ORDER.md) | The verified dependency graph and exact execution order, with single-engineer and 3-engineer staffing scenarios (≈2.5 days vs. ≈1.5 days to the dashboard-live gate). |
| [PARALLELIZATION_PLAN.md](PARALLELIZATION_PLAN.md) | Which card pairs are safe to run fully simultaneously, which need light file-conflict coordination, and which are strictly sequential — classified by real file overlap, not card-ID proximity. |
| [VALIDATION_PLAN.md](VALIDATION_PLAN.md) | Per-card validation commands plus a 9-step gate-level check proving "dashboard live" is actually true, not just that each card's own tests pass. |
| [EXECUTION_HANDOFF.md](EXECUTION_HANDOFF.md) | Ready to assign to engineering agents, with an agent-fit table and a "Recommended Next Card" answer. |

**GUI Phase 1 implementation package (2026-09-01, fourth pass — the backend enablement above is done and verified; this defines the read-only dashboard to build on top of it):**

| File | What it is |
|---|---|
| [GUI_PHASE1_PRD.md](GUI_PHASE1_PRD.md) | Users, use cases, screens, navigation, acceptance criteria for the read-only dashboard. |
| [GUI_PHASE1_ARCHITECTURE.md](GUI_PHASE1_ARCHITECTURE.md) | Frontend architecture (lit-html + esbuild, no SPA framework), state management, hash routing, API integration, deployment model (embedded static assets + a dev-mode disk-serving flag). |
| [GUI_PHASE1_SCREEN_SPEC.md](GUI_PHASE1_SCREEN_SPEC.md) | Every screen specified in full: layout, states, pagination, acceptance criteria. |
| [GUI_PHASE1_API_MATRIX.md](GUI_PHASE1_API_MATRIX.md) | Screen → route → exact response fields used → gaps, re-verified against the real `internal/api`/`internal/core` source, including a full 12-event-type payload table. |
| [GUI_PHASE1_BACKLOG.md](GUI_PHASE1_BACKLOG.md) | MVP / Beta / Nice-to-have, plus the one blocking backend addition (static-file serving) MVP needs. |
| [GUI_PHASE1_FINAL_RECOMMENDATION.md](GUI_PHASE1_FINAL_RECOMMENDATION.md) | Direct answers: what to build first, what framework, fastest MVP, highest-leverage backend addition, what not to build yet. Read this first. |

**GUI Phase 1 MVP — built and verified (2026-09-01, fifth pass — the code, not just the plan; lives outside this folder in `web/`, `cmd/awis-server/`, `internal/api/`):** all 4 MVP screens implemented, running end-to-end against the real backend, verified live in a real browser (`mcp__claude-in-chrome__*`) with real workflow data, plus independent re-verification by a Verifier subagent. Three real bugs found via live testing and fixed: a missing landing-route redirect; a lit-html/DOM-mutation conflict on screen navigation; and (found by the Verifier) all 4 screens silently discarding good data on a transient fetch error instead of showing it alongside the error banner. No separate report document for this pass — the code and its tests (`go test ./cmd/awis-server/... ./internal/api/...`) are the record.

**GUI Beta program (2026-09-01, sixth pass — the next shippable increment on top of the working MVP):**

| File | What it is |
|---|---|
| [GUI_BETA_GAP_ANALYSIS.md](GUI_BETA_GAP_ANALYSIS.md) | Current MVP vs. Beta requirement, grounded in the shipped code — includes two gaps a challenger subagent found that the initial framing missed (Workflow Detail's design gap for intelligence/signal steps; Event Timeline's scale story). |
| [GUI_BETA_PRD.md](GUI_BETA_PRD.md) | Two personas (operator + new user), 8 features (F1-F8) with acceptance criteria. |
| [GUI_BETA_API_REQUIREMENTS.md](GUI_BETA_API_REQUIREMENTS.md) | Exactly two backend additions (a `definition_id` instance filter; version/uptime), plus an explicit, evidence-backed "not required" list (SSE, `/stats`, namespace enumeration, fuzzy search). |
| [GUI_BETA_WORK_BREAKDOWN.md](GUI_BETA_WORK_BREAKDOWN.md) | 13 implementation-ready cards (`BE-1/2a/2b`, `FE-1` through `FE-8`). |
| [GUI_BETA_EXECUTION_ORDER.md](GUI_BETA_EXECUTION_ORDER.md) | The dependency graph (only two real edges), staffing scenarios, ~3 days with two engineers. |
| [GUI_BETA_FINAL_RECOMMENDATION.md](GUI_BETA_FINAL_RECOMMENDATION.md) | Direct answers: what's next, what's deferred, fastest path, highest-leverage backend work, effort estimate. Read this first. |

## Suggested reading order

**Two minutes:** `GUI_BETA_FINAL_RECOMMENDATION.md` (what's next) or `EXECUTION_HANDOFF.md`'s "Recommended Next Card" section (prior backend track), depending which track you're picking up.

**Five minutes:** `FINAL_VERDICT.md`, then `FINAL_RECOMMENDATION.md` for what changed.

**Full picture:**
0. `GUI_COMPREHENSIVE_SUMMARY.md` — for an all-in-one synthesis and complete overview.
1. `GUI_MASTER_PLAN.md` — why each decision and finding exists, with evidence.
1b. `REPOSITORY_TRUTH_AUDIT.md` — whether it still holds up, plus what's new since.
2. `GUI_DEPENDENCY_MAP.md` — what depends on what.
3. `GUI_ARCHITECTURE.md` — what to build.
4. `GUI_ROADMAP.md` — in what order, and what can run in parallel.
5. `GUI_PRD.md` — what each phase looks like to a user.
6. `ENGINE_GUI_TRANSITION_PROGRAM.md` — the exact critical path and acceptance gates (comprehensive version).
7. `ENGINE_GUI_WORK_BREAKDOWN.md` — the cards, ready to assign.
8. `ENGINE_GUI_DECISION_RECORD.md` — which sequencing path to actually run.
9. `FINAL_VERDICT.md` — the comprehensive-plan answers.
10. `GUI_START_LINE.md` — the optimized start line (supersedes §9's timing, not its facts).
11. `MINIMUM_ENGINE_PROGRAM.md` — the cut-down card classification.
12. `EXECUTION_SEQUENCE.md` — the runbook to actually follow.
13. `DEFERRED_WORK_REGISTER.md` — what's parked and why.
14. `FINAL_RECOMMENDATION.md` — the optimized answers, restated plainly.

## Standing conclusion

**Superseded by the optimization pass — see `FINAL_RECOMMENDATION.md` for the current
answer.** The original conclusion (engine work gates the live view via G0→G1→G2, ~5-6
engineering days) is still directionally correct but overstates what gates the *start
line*: card-level tracing shows only 8 specific cards (≈2.5-3 days) are required before
GUI-dominant work can begin, and G2/G3 are not required at all until Phase 2/3
respectively — see `GUI_START_LINE.md` and `MINIMUM_ENGINE_PROGRAM.md`. Nothing in the
evidence base (`GUI_MASTER_PLAN.md` §14 included) was found wrong; the optimization pass
only found the milestone-level rollups conservative relative to what the card-level
dependencies actually require.
