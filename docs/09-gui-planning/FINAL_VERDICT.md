# Final Verdict

Direct answers only. Full reasoning and evidence: `ENGINE_GUI_TRANSITION_PROGRAM.md`,
`ENGINE_GUI_WORK_BREAKDOWN.md`, `ENGINE_GUI_DECISION_RECORD.md`, and the underlying
evidence corpus (`GUI_MASTER_PLAN.md` et al.). Basis: `engine-hardening` @ `f004f4f` +
uncommitted B-31 remediation, independently re-confirmed current as of 2026-09-01.

---

**What must be done before GUI work?**

One process action and one engineering gate. Process: reconcile `STATE.md` and merge
M15→M16→M17→`engine-hardening` into `main` (or explicitly declare `engine-hardening` the
new base) — ~1 day, zero engineering, but otherwise "the engine" is ambiguous and a fresh
branch from `main` today silently re-inherits closed, including data-loss-class, defects
(`GUI_MASTER_PLAN.md` §1.5, N7). Engineering: nothing is mandatory before the *frontend*
and *read-only history views* — those can start now. Before the *live* view: G0→G1→G2
(migration 0007, `state_changes`), gated on founder decision D1. Before the editor's save
path: G6 (the YAML serializer, which does not exist in the repo today).

**What can start immediately?**

Frontend shell, definitions/instances/timeline views against G4's read-only routes
(`ENGINE_GUI_WORK_BREAKDOWN.md` G5-1..5, G5-9), G4's API/server skeleton and read-only
routes (7 of 13 G4 cards have zero engine dependency), all of G1's contract-hardening
work, G0's housekeeping cards, and — notably — G6's YAML key-order design decision
(`G-G6-1`, addressing N4) and G10's subprocess-env-scrubbing fix (`G-G10-3`, R13), both
of which have no dependency on anything else in the program and can be done in spare
capacity at any time.

**What can be deferred?**

G7 (palette/introspection — hardcode the five compile-time `StepType`s in the meantime),
G8 beyond a minimal metrics substrate (full observability infrastructure, since N1
established none exists at all), G9 (AI generation, gated on G6 and D4 anyway), and G10
(multi-user — genuinely deferrable behind loopback-only binding for as long as the
product stays single-operator). Also explicitly deferrable per the architecture: a
first-class `Step.UI` schema field (layout already works today via
`Metadata.ui.positions`, zero schema change needed) and a globally-ordered raw *event*
stream on top of `execution_events` (the `state_changes` change-notification stream
already covers the GUI's actual need).

**Fastest path to usable GUI?**

A read-only dashboard (Phase 1): ~4-5 days of frontend work once G0/G1 and G4's read
routes exist, no engine change beyond those. A *truthful live* dashboard (Phase 2, i.e.
one that doesn't silently freeze on the most common state transition): add G2 (~2 days
engine work, hard-gated on D1) plus ~3.5-4.5 days of SSE wiring. **Total, from a clean
start with D1 decided promptly: ~3-4 weeks** — this is the recommended path's critical
path (`ENGINE_GUI_DECISION_RECORD.md` §3), not a shortcut past it. The cheapest-looking
shortcut (poll instead of building G2/SSE) is a false economy: it ships a strictly worse
product for a 1-2 week head start that gets partly clawed back rewriting the live layer
later (`ENGINE_GUI_DECISION_RECORD.md` §4).

**Fastest path to n8n-class GUI?**

Recommended-path critical path plus G6 (~10-12 days, the largest single milestone in the
program — the write side has zero existing code, `yaml.Marshal` is called nowhere in the
repo) and G7 (~3.5 days). **≈6-7 weeks total** from today, decision latency aside. One
de-risking move is available and recommended regardless of sequencing choice: the
read-only canvas (`G-G5-8`) needs no serializer and can be built and demoed during Phase
1/2, long before G6 — rendering all six workflow-graph relationship types against real
registered workflows gives visible editor progress without gating anything on the
round-trip test.

**Estimated effort by phase?**

| Phase | Milestones | Card-level estimate |
|---|---|---|
| Process prerequisite | Phase A | ~1 day, zero engineering |
| Engine freeze + contract + cursor + namespace | G0-G3 | ~7.5-8 days engineering + D1/D2/D4 decision latency |
| API server | G4 | ~5.3 days |
| Read-only + live GUI | G5 | ~8-9 days |
| Visual editor | G6-G7 | ~13.5-15.5 days |
| Observability | G8 | ~9 days (re-scoped up from the corpus's original 4-5 day estimate — N1 found zero metrics/tracing infrastructure exists, not a partial gap) |
| AI generation | G9 | ~8 days |
| Multi-user | G10 | ~12.5 days |

G8, G9, and G10 do not sit on the critical path and can be staffed in parallel once
G4/G5 land — their totals are additive to calendar time only if under-resourced, not
strictly serial.

**Recommended next action?**

Get the founder to do three things this week, in this order, none of which require
engineering capacity: (1) decide Phase A — reconcile `STATE.md` and either merge
`engine-hardening` to `main` or declare it the base; (2) decide D1 (`state_changes`) and
D2 (retire the namespace overload) — these are the two decisions that gate the most
downstream work and cost nothing to decide now versus later except opportunity cost;
(3) decide D3 (in-module server) and D4 (gate live intelligence). With all four decided,
G0 engineering can close within a day and G1-G4 proceed on the recommended path's
critical path immediately. If only partial decisions are available this week, D1 and D2
are the highest-leverage to get first — they are what `ENGINE_GUI_DEPENDENCY_MAP.md` §2
identifies as gating everything else.

---

*Evidence for every claim above lives in `ENGINE_GUI_TRANSITION_PROGRAM.md`,
`ENGINE_GUI_WORK_BREAKDOWN.md`, `ENGINE_GUI_DECISION_RECORD.md`, and the pre-existing
`GUI_MASTER_PLAN.md` / `GUI_DEPENDENCY_MAP.md` / `GUI_ARCHITECTURE.md` / `GUI_ROADMAP.md`
/ `GUI_PRD.md` / `REPOSITORY_TRUTH_AUDIT.md` corpus.*
