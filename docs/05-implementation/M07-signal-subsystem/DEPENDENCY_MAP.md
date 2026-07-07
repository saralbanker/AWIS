# M07 — Dependency Map
**Critical path position:** M03 → M06 → **M07** → M08 (dependency-index; ≈28d path). Gate **G2** sits
immediately after M07 (IMP §23): failure action = "fix within M6/M7 scope before M8 merges".

## Upstream consumed (M06 HANDOFF contract)
- `internal/engine`: SIGNAL_SCAN stub hook on the tick (`tick.go`); signal-type dispatch returning
  `StepError{runner_unavailable}`; instance `waiting` status transitions reserved for M07;
  Cancel documented to gain wait_record deletion here (B4.6).
- `internal/storage`: AppendEvent single-tx discipline + EDR-005 sequencing; migration runner;
  numbers 0003/0004 reserved (F-4).
- `internal/core` (M01, frozen): `WaitConfig`, `EventTypeSignalReceived`, instance status machine.

## Downstream produced (consumed by)
- **M08 (blocked by M07):** engine Signal intake as the surface `WorkflowRunner.Signal` wraps;
  TriggerAPI intake precedent (F-2) unchanged.
- **M09:** WAIT/signal path drivable by WorkflowTestHarness (deterministic sources already injectable).
- **M14/M17:** `awis signal` (CLI) and `awis audit` (read path, M17) over this milestone's tables.
- **M18:** NFR-P-04 delivery benchmark formalized at G4 (informational here).
