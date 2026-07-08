# M07 → M08/M09/M14/M17 Handoff
**Status: STAGED — activates at M06 merge; actuals filled at M07 completion.**

## Guaranteed outputs (contract — to be confirmed as actuals)
- `internal/signal`: wait-record lifecycle, inbox intake, delivery scan executing the Finalization B3
  single-transaction verbatim (idempotency guard → SignalReceived append → optimistic-lock resume),
  timeout actions (fail/compensate/continue), cancellation cleanup (B4.6).
- Migrations `0003_signals.sql` (Blueprint §9 DDL verbatim) + `0004_audit.sql` (audit_log, F-4)
  + internal audit append API with first call site `SignalDelivered`.
- Engine: SIGNAL_SCAN stage live on the tick; `waiting` status transitions owned and evented.
- Crash-injection harness for the three-table transaction (reusable pattern for M12 FSM proofs).

## What M08 may assume (drafted; confirm at completion)
- Engine exposes the complete signal intake surface to wrap (no SDK types leak from internal/).

## Known limitations (drafted)
- Delivery latency benchmark is informational until M18 (NFR-P-04).
- G2 open items from M06 HANDOFF (EDR-011 join stall semantics; definition identity option (b))
  are reviewed at G2 alongside this milestone's evidence.

## Actuals (filled at completion)
- Merge commit: _ · Verification: _ · Deviations: _
