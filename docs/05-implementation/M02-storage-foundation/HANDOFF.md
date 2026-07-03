# M02 → M03 Handoff
**Status: EXECUTED + VERIFIED — awaiting human squash-merge** — *Guaranteed outputs* are the contract; *Actuals* below (merge commit filled at merge).

## Guaranteed outputs (contract)
- `internal/storage`: SQLite adapter with live EventLog (append/read/range), WorkflowRegistry, StepResultCache; StateStore methods stubbed with ErrNotImplemented
- Migration runner (embedded, sequential, auto-on-open) + `0001` covering events/definitions/cache + `schema_version` tracking table
- `internal/storage/storagetest` contract suite (factory-driven, adapter-agnostic) — extend, don't fork
- Crash-durability demonstrated (NFR-R-02); `make contract` live

## What M03 may assume
- Folding `workflow_instances` (+ `version` optimistic-lock col + `cancellation_requested`, IMP §14) into `0001` is sanctioned pre-tag fold-forward; the N−1 fixture test pattern exists to copy.
- Append path is CE-deep-reviewed and contract-locked: build `rebuild-state` on ReadEvents, never on raw SQL.
- Sequence monotonicity is ENFORCED by storage; assignment belongs to the engine (EDR-005) — rebuild logic may trust per-instance ordering.

## Known limitations (by design)
- No StateStore behavior; ClaimStep/Upsert/Get/List return ErrNotImplemented until M03.
- No signals/audit/domain_events/plugin tables. No pruning (`prune-events` is M17).
- 100K timing is informational; NFR-P-05/06 bind at M03/M14 benchmarks.

## Actuals (drafted at execution 2026-07-03; merge commit filled at merge)
- Merge commit: _ (pending founder squash-merge; branch tip `17ea646`, verified `08ed6d2` M02-V1 19/20 + item-16 fix re-checked)
- Contract suite: 13 subtests (incl. SchemaVersionZeroNormalized added by CE review); crash: child SIGKILLed after 20 committed seqs, all present + intact on reopen (NFR-R-02)
- 100K append wall time: 7.36s ≈ 13,594 events/sec (informational; binding targets NFR-P-05/06 at M03/M14)
- Driver pinned: `modernc.org/sqlite v1.53.0` (direct require)
- CE deep review (append path, per EDR-004): 1 defect fixed pre-verification (schema_version 0→1 DDL-default mirror) + RFC3339Nano trailing-zero ordering wobble recorded as non-defect (sequence_num authoritative per TDS-01 §3 — revisit only if a milestone ever sorts by emitted_at within a second)
- Deviations: (1) verification found go.mod `// indirect` mis-marking (item 16) — fixed via `go mod tidy` in `17ea646`; F4 note: CE review missed go.mod hygiene, caught by independent verification. (2) `make contract` made verbose per verifier note. (3) TestAppend100K skipped under `-race` (instrumentation overhead), runs in plain CI job.
- Open issues created: none
