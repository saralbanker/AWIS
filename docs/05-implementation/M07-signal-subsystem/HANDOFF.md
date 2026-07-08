# M07 → M08/M09/M14/M17 Handoff
**Status: COMPLETE — actuals filled at D-CLOSE 2026-07-08.**

## Guaranteed outputs (confirmed actuals)
- `internal/signal`: wait-record lifecycle, inbox intake, delivery scan executing the Finalization B3
  single-transaction verbatim (idempotency guard → SignalReceived append → optimistic-lock resume),
  timeout actions (fail/compensate/continue), cancellation cleanup (B4.6).
- Migrations `0003_signals.sql` (Blueprint §9 DDL verbatim) + `0004_audit.sql` (audit_log, F-4)
  + internal audit append API with first call site `SignalDelivered`.
- Engine: SIGNAL_SCAN stage live on the tick; SIGNAL_TIMEOUT_SCAN stage live on the tick;
  `waiting` status transitions owned and evented; early-return-on-empty-items removed (C3r OUTPUT 0).
- Crash-injection harness for the three-table transaction (reusable pattern for M12 FSM proofs).

## What M08 may assume (confirmed)
- Engine exposes `Signal(ctx, instanceID, name, payload)` for external signal intake (no SDK wrapper — M08 adds that).
- Engine exposes `Submit`, `Cancel`, `Run`, `Tick` from M06; `Signal` from M07.
- Engine exposes `Ingest(ctx, DomainEvent)` for TriggerAPI (F-2, from M06).
- StoragePort's 12 methods are frozen. All signal/audit storage is via additive type-asserted
  interfaces (*SQLiteStorage methods). M08 follows the same pattern for any new storage access.
- `sdk/` type aliases (WorkflowRunner, RecallAPI, WorkflowStatus, etc.) are in place; shapes are
  empty stubs — M08 owns the freeze of these shapes (IMP §13).
- `audit_log` table live; `AppendAudit` API available; `WorkflowRegistered` is M08's call site (F-4).

## Known limitations (actuals)
- Delivery latency benchmark is informational until M18 (NFR-P-04). Measured: **1.72 ms/op**
  (116× under the 200ms target).
- G2 open items from M06 HANDOFF (EDR-011 join stall semantics; definition identity option (b))
  are reviewed at G2 alongside M07's evidence.
- EDR-007 §9 waiting-entry gap: `waiting` is a non-evented projection write — a cold RebuildState
  sees `running` until SignalReceived arrives. Documented, tested, not a defect.

## Actuals
- **Merge commit:** pending (branch `m07-signal-subsystem`; PR + G2 verdict required)
- **Verification:** M07-V1 PASS at HEAD `6ac948a`; all 8 implementation checks ✅;
  V-COMMON-1 procedural ❌ resolved by commit `7a360e9` (pre-existing CLAUDE.md + STATE ledger write)
- **Deviations:**
  1. C3 WITHDRAWN pre-dispatch (CE card-cutting gap — post-delivery WAIT-step completion was in
     neither C2 nor C3; E1 adjudication → C3r carries the delta; see TRACEABILITY)
  2. C3r executed on **Sonnet** vs Opus (IMP §28 binding); founder-directed after two Opus
     session-limit deaths; recorded in TRACEABILITY; mitigations: crash-injection checkpoint (§20.M7),
     independent V1, G2 human verdict. G2 reviewers must weigh when judging atomicity evidence.
  3. `timeout_at` NULL in C2 (WaitConfig.Duration serialization gap); resolved in C3r as
     RFC3339 absolute timestamp (Blueprint §9 L1410)
