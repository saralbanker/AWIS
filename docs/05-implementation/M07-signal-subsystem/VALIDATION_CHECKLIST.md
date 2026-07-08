# M07 — Validation Checklist (binary; exit list)
Global DoD (IMP §24) + M07 rows (IMP §27.M7, §20.M7, §23.G2). Verifier executes via cards/M07-V1.md.

- [x] V-COMMON block all ✅ (`make build test lint race` + `make e1`; clean tree; HEAD recorded)
- [x] Migration `0003_signals.sql` applies; DDL byte-matches Blueprint §9 L1396–1420 (signal_inbox,
      wait_records, idx_wait_timeout, idx_signals_instance)
- [x] Migration `0004_audit.sql` applies; `audit_log` append-only; internal append API present (F-4)
- [x] Delivery is ONE transaction implementing Finalization B3 exactly: `delivered_at IS NULL`
      guard → EventLog `SignalReceived` append → `waiting→running` optimistic-lock transition
- [x] Guard-miss ⇒ whole transaction no-op; re-running the scan never double-delivers (NFR-R-04)
- [x] Optimistic-lock failure ⇒ rollback; inbox entry remains undelivered; next tick re-attempts
- [x] Crash injected between EACH write pair (SQLite hook): invariant holds — `SignalReceived`
      present in EventLog ⇔ signal delivered (IMP §20.M7; IR-3)
- [x] WAIT step entry creates wait_record; delivered signal resumes exactly the waiting step
- [x] Timeout actions `fail` / `compensate` / `continue` each fixture-proven; zero new event types
- [x] Cancel deletes the instance's wait_records (Finalization B4 step 6) — fixture-proven
- [x] forward projection ≡ RebuildState on every signal fixture (EDR-007 discipline from M06)
- [x] `SignalDelivered` audit row written at the delivery site (NFR-S-05)
- [x] Delivery latency measured and reported vs ≤200ms (NFR-P-04 — informational until M18)
      **Measured: 1.72 ms/op (116× under target)**
- [x] G2 evidence assembled: event-sequence fixtures vs B3/B4 text + crash-injection transcript

**V1 result:** PASS at HEAD `6ac948a` (2026-07-08, awis-verifier)
