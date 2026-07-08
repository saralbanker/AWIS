# M07 — Implementation Spec (card index; EEOS §15 D1 — the normative cut lives in cards/)
Compiled 2026-07-08 from IMP §27.M7 + Finalization B3/B4 + F-4 + IKB §4/M07. Branch: `m07-signal-subsystem`.

| Card | Objective (one line) | Scope | TRACEABILITY rows |
|---|---|---|---|
| [C1](cards/M07-C1.md) | Migrations 0003 (signals, verbatim DDL) + 0004 (audit, per F-4/CONTRA-4) + storage CRUD + internal audit append API | `internal/storage` only | T1–T3 |
| [C2](cards/M07-C2.md) | Atomic delivery EXACTLY per B3 SQL (single tx) + `internal/signal` scan + WAIT entry/intake + engine SIGNAL_SCAN fill + SignalDelivered audit site | `internal/storage` (DeliverSignal tx), `internal/signal`, `internal/engine` | T4–T8 |
| [C3](cards/M07-C3.md) | Timeout actions (fail/compensate/continue, closed event vocabulary) + cancel deletes wait_records (B4.6) + crash-injection proof (IR-3/§20.M7) + NFR-P-04 bench | `internal/signal`, `internal/engine/cancel.go`, crash/bench tests | T9–T12 |
| [V1](cards/M07-V1.md) | Independent verification: V-COMMON + M07 checkpoints + G2 evidence items | read/run only | — |

Scope walls (all cards): no `sdk/` changes (M08); no CLI (M14/M17); no new event types (TDS-01 closed,
G1-frozen); no new dependencies; migration numbers 0003/0004 fixed by M06 HANDOFF reservation (F-4).
Cards serialized C1→C2→C3 (same packages, one agent).
