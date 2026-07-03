# M03 → M06 Handoff
**Status: PENDING EXECUTION** — *Guaranteed outputs* are the contract; *Actuals* filled at merge.

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

## Actuals (fill at merge)
- Merge commit: _
- Contract suite size: _ ; 10K replay result: _ ; 100K rebuild time: _
- Deviations: _
