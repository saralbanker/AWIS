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
- **C1 DONE `e223dc4`** (core-engineer). Deviations accepted by CE: (1) audit_log DDL as E0 per
  CONTRA-4 — timestamp/event_type/actor/payload_summary + surrogate autoincrement id, no CHECK on
  event types; (2) db_test head-version assertions 2→4 (inherent to new migrations); (3) namespace
  isolation realized per-instance_id — frozen DDL carries no namespace column (NFR-S-04 spirit,
  tested); (4) caller-supplied time semantics, clock stays engine-owned. Env note: golangci-lint
  absent for the run (gofmt+vet clean); CE installed v2.12.2 (CI-matching) before C2 — lint runs
  from C2 on and at V1.
- **C2 DONE `142b684`** (core-engineer). B3 tx verbatim in storage/deliver.go; scan in
  internal/signal; intake/enterWait/waits index in engine. Deviations accepted: (1) SignalDelivered
  audit emitted in the Scanner — the literal delivery site; (2) timeout_at written NULL (Duration
  has no frozen serialization) — resolved in C3r. **CE card-cutting defect logged (AEO §8.2):**
  post-delivery step completion was in neither C2 nor C3; C2's honest delivery-only reading
  surfaced it. Adjudication (E1, coordinates: EDR-011 L6 "M07 owns waiting-status semantics";
  TDS-02 Transition.condition over step outputs; TDS-01 StepCompleted REPLAY; Blueprint §8 SETTLE):
  the waiting step completes with the signal payload as outputs via the existing SETTLE path and
  its wait_record is deleted on completion. C3 WITHDRAWN pre-dispatch; C3r carries the delta.
