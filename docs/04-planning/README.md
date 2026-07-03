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
| **G1 amendment ADJ-5** (at M03 entry) | 2026-07-03 | **APPROVED** by founder: `compensation_failed` added as 9th InstanceStatus (CONTRA-7 — Finalization B4/Blueprint §8 vs §6 enum; Finalization outranks). Also CONTRA-6 approved: claim = additive `step_claims` table + optimistic version, enum untouched ('in_flight' = informal §8 wording) | TDS-02 appendix; core/instance.go; M03 TRACEABILITY |
| **G1 amendment ADJ-8** (at M06 entry) | 2026-07-03 | **APPROVED** by founder: CONTRA-11 — optional `adapter`/`model`/`tokens_used` added to the TDS-01 StepCompleted payload (FR-IL-09 Must Have + Blueprint §17 vs §9-derived four-field schema; additive, intelligence steps only, replay-neutral) | TDS-01 §2; M06 TRACEABILITY |
| **G1 amendments ADJ-6/ADJ-7** (at M06 entry) | 2026-07-03 | **APPROVED** by founder: ADJ-6/CONTRA-8 — RetryPolicy amended to full Blueprint §8 shape (optional `initial_delay`/`max_delay` Durations; `backoff` constrained to `immediate\|linear\|exponential`; absent delays default 1s/30s per §16 pattern; additive, serialization-compatible). ADJ-7/CONTRA-9 — `required` (bool, optional, default false) added to the TDS-02 intelligence block (Blueprint §13 YAML + FR-IL-06/07 reachability). CONTRA-10 recorded with authority-order disposition (step `inputs` = template value-map per §7 example + PRD §18 rule 1 + FR-WD-05; §6's "JSON Schema" tag is the outlier; no format change — type already map-shaped) | TDS-02 §2/§7; core/step.go; M06 TRACEABILITY |
