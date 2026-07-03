# M02 — AI Execution Context
**Model allocation (IMP §28):** Sonnet implementation + deep review on the EventLog append path ("Opus review" row → CE/Fable semantic review per EDR-004). No gate.

## Session loading order (≈12k tokens)
1. `docs/00-foundation/README.md`
2. This module's `IMPLEMENTATION_SPEC.md` + `VALIDATION_CHECKLIST.md`
3. `docs/EVENTLOG_FORMAT.md` (TDS-01) + `docs/WORKFLOW_SCHEMA.md` (TDS-02) — the frozen formats
4. Blueprint §20 DDL/interface excerpts arrive inside task cards; do not browse the Blueprint

## Hard constraints
- TDS-01/02 are G1-frozen: column names, event-type strings, JSON field names ship VERBATIM.
- `execution_events` DDL = §20 base + `schema_version INTEGER NOT NULL DEFAULT 1` (G1 ADJ-1). This is the ONLY sanctioned delta from the §20 SQL.
- Append-only is law (FR-ST-01): the adapter exposes no update/delete on events.
- Determinism: clock injectable (IMP §3); no `time.Now()` outside the default clock source.
- `internal/core` and `sdk` are read-only this milestone.
- CGO_DISABLED everywhere; the driver is pure-Go `modernc.org/sqlite` (edr-003).

## Escalate (STOP) when
- Any mismatch between TDS-01/02 and Blueprint §20 DDL beyond the sanctioned ADJ-1 delta.
- The frozen StoragePort signatures cannot be honored without touching `internal/core`.
- Any need for a dependency beyond the SQLite driver (+testify).
