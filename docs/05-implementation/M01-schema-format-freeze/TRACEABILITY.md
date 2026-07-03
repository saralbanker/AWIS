# M01 — Traceability
| Task / artifact | Canonical source | Coordinate |
|---|---|---|
| TDS-01 EventLog format | Blueprint | §9 (envelope, event types, sequence) |
| TDS-01 schema_version semantics | Blueprint §9; IMP | §23 G1 checklist |
| TDS-02 WorkflowDefinition serialization | Blueprint | §6 |
| TDS-02 semver + immutability | Blueprint §6; PRD | FR-WD family (versioned, immutable definitions) |
| TDS-03 grammars + prohibited lists | Finalization | Blocker 2 (verbatim) |
| Conformance corpus before parser | IMP | §25 IR-2 mitigation |
| `internal/core` + sdk aliases | Verification report | **F-1** (binding amendment) |
| sdk file layout | PRD | FR-SDK-02 |
| Six core structures as types | Blueprint | §6 (Step, WorkflowDefinition, WorkflowInstance, ExecutionEvent, IntelligencePort, StoragePort) |
| Gate G1 question + evidence | IMP | §23 row G1; Constitution Art. 12 (decade-reader) |
| "No executable logic" wall | IMP | §27.M1 Scope |
| Iterate-in-place rollback | IMP | §27.M1 RB |

**Requirements satisfied:** none executable yet; freezes the formats behind FR-ST-*, FR-WD-*, FR-SDK-02.
**Contradictions touched (execution findings 2026-07-03, per contradiction protocol — G1 adjudication items, never silently resolved):**

| ID | Class | Finding | Coordinates | Status |
|---|---|---|---|---|
| ADJ-1 (GAP-G1) | Tier-0 silence vs Tier-1 requirement | `schema_version` demanded by G1 checklist (IMP §23) but absent from `execution_events` DDL and envelope | Blueprint §9 L624–634 vs IMP §12/§23 | **G1-APPROVED 2026-07-03:** additive column `schema_version INTEGER NOT NULL DEFAULT 1`; normative in TDS-01 §1.1 |
| ADJ-2 (GAP-G2) | Tier-0 internal conflict → **CONTRA-5** | `WorkflowCancelled` payload: §9 `{reason}` vs §8 `{reason, cancelled_at}` | Blueprint §9 L671 vs §8 L603 | **G1-APPROVED 2026-07-03:** canonical = §9 `{reason}`; `cancelled_at` derived from envelope `emitted_at`, never duplicated; TDS-01 event #9 |
| ADJ-3 (GAP-G3) | Tier-0 silence vs Tier-1 requirement | Format-level `schema_version` absent from WorkflowDefinition (§6 `version` is the workflow's own semver, distinct) | Blueprint §6 L260–277 vs IMP §12/§23 | **G1-APPROVED 2026-07-03:** required top-level `schema_version: int` = 1; normative in TDS-02 §1/§6 |
| ADJ-4 | Tier-0 type-shape silence | 11 referenced types have no field definitions anywhere in the corpus: StepError (persisted in 3 event payloads — irreversible-format-relevant), DraftResponse, SynthesisResponse, Capability, Example, Logger, WorkflowStatus, InstanceFilter, HistoryQuery, ExecutionRecord, ReplayTrace, StepStatistics (+ scalar underlyings SemVer/HandlerRef/Duration/Condition/InputSchema/OutputSchema, CompensationPlan/Ref) | Blueprint §12 L844–963, §13 L975–991, §9 events 4/8/12 | **G1-APPROVED 2026-07-03 with refinement:** (a) StepError FROZEN as persisted type `{code, message, details?}` — TDS-01 §2.1; (b) all non-persisted API types = minimal placeholder shells, shapes completed at owning milestones (M02/M04/M08), NOT part of the G1 freeze (IMP §13 — sdk mutable until M08) |
