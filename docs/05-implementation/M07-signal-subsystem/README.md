# M07 — Signal Subsystem  → GATE G2
**Status:** Partitioned — materialize at entry · **Effort:** 2d · **Window:** Week 2
**Objective:** WAIT steps, wait_records, signal inbox, single-transaction atomic delivery EXACTLY per Finalization Blocker 3 SQL (delivered_at IS NULL guard → EventLog append → optimistic-lock transition), timeout actions (fail/compensate/continue), wait-record cleanup on cancel; migration 0002. **[F-4] `audit_log` table + internal append API land HERE** (first write site: SignalDelivered); later milestones add their own call sites.
**Depends on:** M06 · **Blocks:** M08 · **Gate:** G2 (execution semantics vs Finalization text; crash-injection evidence)
**Primary sources:** IMP §27.M7; Finalization Blockers 3–4; Verification F-4 · **Compilation spec:** IKB §4/M07
**Key ACs:** crash-injection between each write pair holds the invariant (SignalReceived in EventLog ⇔ delivered); double-delivery no-op (NFR-R-04); delivery ≤200ms (NFR-P-04, informational until M18).
