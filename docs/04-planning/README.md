# 04 — Planning (pointers)

- `../../IMPLEMENTATION_MASTER_PLAN.md` — **ARCHIVAL** as of IKB generation. Its content is
  fully partitioned into `../05-implementation/`. Do not load during implementation sessions;
  it remains the immutable reconstruction source and audit reference.
- `../../IMPLEMENTATION_MASTER_PLAN_VERIFICATION_REPORT.md` — verification verdict + the five
  BINDING amendments (F-1..F-5) and observations (O-1..O-5). Amendments are already folded into
  the affected milestone modules; the report is the authority for why.
- `../../IMPLEMENTATION_KNOWLEDGE_BASE.md` — the IKB master document: partition strategy,
  per-milestone compilation spec, indices, loading strategy.

Human gates: G1 (after M01) · G2 (after M07) · G3 (after M15) · G4 (after M18) — defined in IMP §23.

## Gate log
| Gate | Date | Verdict | Record |
|---|---|---|---|
| **G1 — Schema Freeze** (after M01) | 2026-07-03 | **APPROVED** by founder, with adjudications ADJ-1..4 adopted (ADJ-1 EventLog `schema_version` column; ADJ-2/CONTRA-5 `WorkflowCancelled` = `{reason}` only; ADJ-3 WorkflowDefinition format `schema_version`; ADJ-4 StepError `{code, message, details?}` frozen as persisted type, non-persisted API types deferred to owning milestones as shells) | TDS-01/02/03 headers; M01 module TRACEABILITY.md; milestone PR |
