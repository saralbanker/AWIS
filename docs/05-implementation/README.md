# 05 — Implementation Modules (the operational surface)

One directory per milestone, M00–M18. Names, scope, effort, and dependencies are fixed by
IMP §27 as amended by Verification F-1..F-5. Milestones are NEVER reordered without dependency
evidence.

## Materialization policy (deterministic, drift-proof)
- **M00 and M01 are fully materialized now** (implementation begins there; their content depends
  on nothing unexecuted).
- **M02–M18 are partitioned but lazily materialized.** Each directory holds its README (identity,
  objective, dependencies, amendments, source coordinates). The remaining six module files are
  compiled AT MILESTONE ENTRY as the milestone's first task (≈30 min, mechanical) by executing
  the per-milestone compilation block in `../../IMPLEMENTATION_KNOWLEDGE_BASE.md` §4 against the
  immutable sources. Rationale (Verification-grade, recorded):
  1. `HANDOFF.md` is defined as *guaranteed outputs, dependencies satisfied, known limitations* —
     truthful only after the predecessor actually executed. Pre-writing it would fabricate actuals.
  2. Eager copies of week-5/6 content would freeze pre-G2/G3 assumptions and create the exact
     drift this KB exists to prevent (Zero Architectural Drift principle).
  3. The partitioning rules prohibit duplication not required for execution safety.
- A milestone module, once materialized, is updated only by: (a) executing its milestone,
  (b) a gate outcome, or (c) a recorded CONTRA entry. Nothing else touches it.

## Module contract (every materialized milestone directory)
`README.md` · `IMPLEMENTATION_SPEC.md` · `AI_EXECUTION_CONTEXT.md` · `VALIDATION_CHECKLIST.md`
· `HANDOFF.md` (finalized at completion) · `TRACEABILITY.md` · `DEPENDENCY_MAP.md`

## EEOS v2.0 ruling (founder-approved 2026-07-08 — AWIS_EEOS.md §15 D1)
From M07 on, a module file may **satisfy its contract by reference** to the milestone's
`cards/` directory (persisted Execution Cards, EEOS §4): `IMPLEMENTATION_SPEC.md` = card index
(objectives + scope-wall summary + card→TRACEABILITY rows) and `AI_EXECUTION_CONTEXT.md` =
model row + dispatch order + milestone escalation deltas, each ≤15 lines. The transcription
layer they used to carry lives once, inside the cards. The 7-file contract and this policy's
drift rules are otherwise unchanged; `cards/` is an optional additional subdirectory. The
execution ledger `STATE.md` and the immutable verification block `V-COMMON.md` live beside
the milestone directories (EEOS §1).
