# M07 — Traceability
Every task → canonical coordinate. Cards: docs/05-implementation/M07-signal-subsystem/cards/.

| Row | Task (card) | Canonical coordinate |
|---|---|---|
| T1 | Migration 0003 signals DDL (C1) | Blueprint §9 L1396–1420 (verbatim); IMP §14 (`0002_signals.sql` row, renumbered — see note) |
| T2 | Migration 0004 audit_log + append API (C1) | Verification **F-4** (adopted; due M7); IMP §14 `0004_audit.sql` row; **CONTRA-4 disposition** (additive, unenumerated, StoragePort-internal); PRD NFR-S-05, §26 L1924–1936 |
| T3 | Inbox/wait storage CRUD (C1) | Blueprint §9 (tables); IMP §14 implementation order L280 |
| T4 | Single-tx atomic delivery (C2) | **Finalization B3 L219–252 verbatim** (decision: Option A); Blueprint §8 L559–583 (Update 3) |
| T5 | signal scan / SIGNAL_SCAN fill (C2) | Blueprint §8 pull loop; M06 HANDOFF ("SIGNAL_SCAN is an explicit stub hook"; `internal/engine/tick.go` stub) |
| T6 | WAIT entry + Signal intake (C2) | Blueprint §6 L291 (WaitConfig); TDS-02 §2 `wait_signal` row; PRD FR-WE-05/06, FR-SE-12/13 |
| T7 | SignalDelivered audit call site (C2) | F-4 ("first write site: SignalDelivered"); NFR-S-05 |
| T8 | forward≡rebuild signal fixtures (C2) | EDR-007; TDS-01 §SignalReceived (REPLAY note); M06 equivalence discipline |
| T9 | Timeout actions fail/compensate/continue (C3) | Blueprint §9 L1411 (`timeout_action` enum); IMP §27.M7; TDS-01 closed event list (constraint) |
| T10 | Cancel deletes wait_records (C3) | Finalization B4 step 6; M06 HANDOFF ("DOCUMENTED as M07's addition to Cancel") |
| T11 | Crash-injection proof (C3) | IMP §20.M7 L367 (verbatim procedure); IR-3; NFR-R-04 |
| T12 | Latency bench informational (C3) | PRD NFR-P-04 (≤200ms); IMP §27.M7 AC |

## Notes / dispositions
- **Migration renumbering (F-2×F-4 interaction, recorded at M06):** IMP §14 names `0002_signals.sql`,
  but F-2's `0002_domain_events.sql` landed at M06; M06 HANDOFF reserves **0003 = signals,
  0004 = audit**. Not a CONTRA — pre-implementation renumbering is sanctioned by F-4's own text.
- **CONTRA-4 (audit_log unenumerated):** disposition per IMP §25 — additive migration internal to
  StoragePort; DDL shape is implementer's E0 within PRD §26's description.
- Cards C1r/etc., deviations, and verification results are appended here as they occur (EEOS §3.3).

## Execution record (appended during B-BUILD/C-VERIFY)
- (empty — milestone staged; activates when M06 merges)
