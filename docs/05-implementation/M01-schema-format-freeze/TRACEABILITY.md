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
**Contradictions touched:** none expected; any transcription gap follows the contradiction protocol (00-foundation/README.md).
