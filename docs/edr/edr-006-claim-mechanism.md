# EDR-006 — step_claims: claim mechanism + release site

- **Status:** ADDITIVE — pre-tag, reversible (G2 docket item noted)
- **Date:** 2026-07-03
- **Founder approval:** 2026-07-03
- **Coordinates:** CONTRA-6; Blueprint §8 (loop/claim semantics, steps 3 and 5);
  IMP §14 (Finalization B4); M03-C1 (`internal/storage/sqlite.go`)

---

## Decision

**Mechanism (CONTRA-6 resolution):** worker exclusivity is implemented as an
additive `step_claims` table rather than an `InstanceStatus` enum value.

| Column | Type | Notes |
|--------|------|-------|
| `instance_id` | TEXT NOT NULL | FK to workflow_instances |
| `step_id` | TEXT NOT NULL | step being claimed |
| `worker_id` | TEXT NOT NULL | claiming worker identity |
| `claimed_at` | TEXT NOT NULL | RFC3339Nano UTC from injectable clock |
| PRIMARY KEY | `(instance_id, step_id)` | at-most-once gate |

**`InstanceStatus` enum:** untouched. Blueprint §8's informal phrase "in_flight"
is read as a description of a worker's local view, not a persisted status value.
The status enum remains at 9 values (running, waiting, pending, completed, failed,
cancelled, compensating, compensated, compensation_failed).

**Optimistic-lock coupling:** a successful `ClaimStep` bumps
`workflow_instances.version` by 1 inside the same transaction (§8 step 3). This
ensures a concurrent `UpsertInstance` with a pre-claim `expectedVersion` receives
`ErrVersionConflict`, preventing lost-update races.

**Release site (EDR-006 primary decision):** claims are released by deleting all
`step_claims` rows for an instance inside the same `UpsertInstance` transaction
whenever the incoming status is terminal
(`completed | failed | cancelled | compensated | compensation_failed`).
This is §8 step 5's "release claim" — no dedicated release method exists on
`StoragePort`; the release is co-located with the terminal state write, which is
the only correct moment: the step is done, the result is persisted, the lock
must drop atomically.

---

## Coordinates

- **CONTRA-6** — foundational adjudication authorising additive table.
- **Blueprint §8 step 3** — `ClaimStep` INSERT + version bump.
- **Blueprint §8 step 5** — claim release at terminal upsert.
- **IMP §14 / Finalization B4** — `cancellation_requested` column (non-evented
  request flag, not managed by `UpsertInstance`; rebuilds as 0).
- **M03-C1** — `ClaimStep` and `UpsertInstance` in `internal/storage/sqlite.go`.

---

## Rationale

- **Additive table** isolates claim state from instance lifecycle state; the two
  can evolve independently.
- **Atomic release** (same tx as terminal upsert) prevents a window where
  `step_claims` rows linger after an instance completes and block re-execution
  paths (e.g. rebuild-state → re-claim → re-run, or the
  TerminalUpsertReleasesClaims contract test).
- **Version bump on claim** ensures the engine (M06) can detect concurrent
  claim and upsert via OCC without a separate read-before-write.

---

## Reversibility

Additive pre-tag. The table, version-bump, and release logic are fully contained
in `internal/storage/sqlite.go` and `migrations/0001_core_execution.sql`. A G2
revisit could replace the table with a dedicated `StoragePort.ReleaseClaim` method
or move release to the engine; both are single-file changes with no external
install surface.
