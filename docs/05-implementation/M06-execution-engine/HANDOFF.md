# M06 → M07/M08/M09/M11/M12 Handoff
**Status: EXECUTED + VERIFIED (M06-V1r 15/15 at `01d2f5c`)** — merge commit filled at merge.

## Guaranteed outputs (contract)
- `internal/engine`: Engine (Submit/Cancel/Ingest/Tick/Run), pull loop per §8, forward projection
  mirroring EDR-007 exactly (fixture-proven ≡ RebuildState), retry/fallback/compensation/
  cancellation (B4), fan-out within MaxParallelSteps, idempotency via StepResultCache, slog JSON.
- `internal/runner/native` + `internal/runner/intelligence`: Runner implementations; usage
  side-channel feeding ADJ-8 payload fields.
- Migration `0002_domain_events.sql` (TTL 7d) + trigger matching (F-2); `make e1` LIVE.
- Core completions: CompensationPlan/CompensationStep/CompensationRef, Logger, DomainEvent;
  ADJ-6/7 fields on RetryPolicy/IntelReq.

## What M07 may assume
- SIGNAL_SCAN is an explicit stub hook on the tick; signal-type dispatch returns
  StepError{runner_unavailable}; instance `waiting` status transitions are entirely M07's.
- Cancellation deletes wait_records is DOCUMENTED as M07's addition to Cancel (B4 step 6).
- Migration numbers 0003/0004 are reserved for signals/audit (F-4 renumber).

## What M08 may assume
- Engine methods are the complete runtime control surface to wrap; version selection and
  registration-time handler resolution are M08's additions, not engine changes.

## Known limitations (by design)
- Single-process V1: dispatch waits within the tick; multi-worker path exists via claims but is
  exercised only by unit tests, not a second process.
- EDR-011 join semantics: conditionally-skipped upstream branches stall a join step (author
  semantics per B7 note) — G2 review.
- Definition identity remains non-evented (EDR-007 §4 option (b) interim) — G2 decision.

## Actuals (filled at completion)
- Merge commit: `b691965` (squash to main, 2026-07-08, founder-directed; tag `milestone/M06`).
  Verified at `01d2f5c` (M06-V1r 15/15); delta to merge = docs/EEOS/CI-lint only (recorded in
  merge message). Deviations: none beyond TRACEABILITY §Deviations (continuation cards, EDR-011 §8).
