# EDR-005 — sequence_num assignment vs enforcement

- **Status:** ADDITIVE — pre-tag, reversible
- **Date:** 2026-07-03
- **Coordinates:** TDS-01 §3; IMP §2.4; M02-C2 (sqlite.go `AppendEvent`)

## Decision

TDS-01 §3 and the Blueprint §9 DDL declare `sequence_num` as "monotonically
increasing per instance" but are silent on **which layer assigns the value**.
Two candidate sites exist:

| Site | Role |
|------|------|
| **Storage adapter** (AppendEvent) | persistence boundary |
| **Engine** (M06) | event-emitting orchestrator |

The M02-C2 SQLite adapter **enforces** strictly-increasing `sequence_num` but
does **not assign** it: AppendEvent rejects any event whose `sequence_num ≤`
the current maximum for the instance (typed `ErrSequenceViolation`), but it
writes whatever value the caller supplies. Assignment responsibility belongs to
the engine (M06), which has full knowledge of instance state and retry
semantics needed to choose a correct sequence number.

## Rationale

- The storage layer is a pure persistence boundary (Blueprint §20). Assigning
  sequence numbers in AppendEvent would embed engine-level state transitions in
  the adapter, coupling the two layers prematurely.
- The contract test suite (storagetest.Run) verifies enforcement at the
  adapter boundary; the engine tests (M06) must verify correct assignment.
- The adapter-agnostic contract remains stable for the Postgres V2 reuse path:
  enforcement logic is identical regardless of assignment site.

## Coordinates

- **TDS-01 §3** — monotonicity invariant (enforcement scope).
- **IMP §2.4** — engine owns sequence_num assignment.
- **M02-C2** — `AppendEvent` in `internal/storage/sqlite.go`.
- **M06** — engine implementation (assignment site).

## Reversibility

Additive pre-tag. If a future design moves assignment into the adapter (e.g.
for atomic auto-increment), the typed `ErrSequenceViolation` check is the
single deletion site; no other layer depends on enforcement being in the
adapter.
