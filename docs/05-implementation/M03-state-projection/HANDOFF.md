# M03 → M06 Handoff
**Status: MERGED — milestone closed** — *Guaranteed outputs* are the contract; *Actuals* below.

## Guaranteed outputs (contract)
- StoragePort 12/12 live and contract-tested; optimistic versioning on instances; at-most-once ClaimStep (step_claims, CONTRA-6)
- `RebuildState` library: EventLog → byte-identical projection (NFR-R-03 demonstrated on 10K fixture)
- Migration 0001 complete (instances + cancellation_requested + claims); N−1 fixture pattern maintained

## What M06 may assume
- Projection rules in EDR-007 are the de-facto spec: engine event emission MUST produce states RebuildState reproduces (event-sequence fixtures at G2 verify).
- Claim: `ClaimStep` true = exclusive; release happens automatically on terminal upsert; mid-flight release semantics are M06's to define (extend EDR-006, don't improvise).
- Variables scoping: `{"inputs": ..., "<step_id>": outputs}` — templates/conditions resolve against this shape (M05 corpus compatible).

## Known limitations (by design — G2/M07 docket)
- definition_id/version are not evented; rebuild preserves them only from pre-existing rows (EDR-007 gap — M06 must event them or accept).
- `waiting` status entry is not evented until M07's wait_records.
- `cancellation_requested` rebuilds as 0 (request flag is non-evented state; B4).

## Actuals (drafted at execution 2026-07-03; merge recorded same day)
- Merge commit: `7a11348` (squash of `m03-state-projection`, founder-approved verdict 2026-07-03; branch tip `f8f04b7`, verified `980d8b6` M03-V1 17/18 + item-4 fix `2eb9604` re-checked; tagged `milestone/M03`)
- Contract suite: 22 subtests (12/12 StoragePort methods covered); 10K replay: 10,849 events / 60 instances, byte-identical across two rebuilds + 6 field-exact anchors (incl. compensation_failed); crash+rebuild consistent; 100K rebuild: **447ms** (informational; NFR-P-06 gate <30s at M18)
- Founder adjudications at entry: **CONTRA-6** (claims table, no in_flight status) + **CONTRA-7/ADJ-5** (compensation_failed 9th InstanceStatus; G1-frozen TDS-02 amended under sign-off) — gate log + TRACEABILITY
- CE deep review (rebuild fidelity, per EDR-004): 3 findings fixed pre-verification — scan-error propagation (silent-skip would corrupt recovery), strict payload sub-field parsing (unmarshalField), offline/single-writer assumption documented
- Deviations: (1) M03-V1 item 4: GetInstanceNotFound contract subtest was missing + checklist said 11 fields (10 correct) — both fixed `2eb9604`, F4-noted. (2) C2 session-limit interruption before any work; clean re-dispatch. (3) `ExposedDB()` test accessor added (rebuild tests live in storage_test package); non-behavioral. (4) CE review patches slightly exceeded the trivial allowance (~35 lines) — mechanical, unambiguous fixes; recorded per AEO §4.1 deviation practice.
- Open issues created: none (G2 docket items live in TRACEABILITY/EDR-007: definition-identity gap, waiting-entry at M07, in_flight reading)
