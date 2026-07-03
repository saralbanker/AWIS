# M03 — Traceability
| Task / artifact | Canonical source | Coordinate |
|---|---|---|
| workflow_instances DDL + optimistic version | Blueprint §20 SQL; §9 consistency | verbatim + version counter |
| cancellation_requested folded into 0001 | IMP §14 (greenfield fold-in); Finalization B4 | migration row 0001 |
| step_claims additive table | **CONTRA-6 disposition** (founder 2026-07-03); Blueprint §8 step 3 | claim/version at-most-once |
| Claim release at terminal upsert | **EDR-006**; Blueprint §8 step 5 ("Release claim" — no frozen method) | SETTLE semantics |
| UpsertInstance expectedVersion guard | Blueprint §9 ("optimistic locking on instance version") | consistency guarantees |
| InstanceFilter completion {Namespace, Status} | ADJ-4b (pre-M08 mutable); Blueprint §8 SCAN + NFR-S-04 | first-consumer completion |
| rebuild-state library, deliberately early | IMP §26, §27.M3 | recovery predates engine |
| Projection rules (variables scoping, status map) | TDS-01 §2 replay statements; §6/§8; **EDR-007** | per-event mapping |
| compensation_failed projection | **CONTRA-7 / G1-amendment ADJ-5** (founder 2026-07-03); Finalization B4 authority over §6 enum | 9th InstanceStatus |
| Byte-identical replay | PRD NFR-R-03; IMP §20 checkpoint | 10K fixture |
| 100K rebuild informational | PRD NFR-P-06 (<30s, binds at M18) | benchmark note |

**Contradictions touched:**
- **CONTRA-6** (§8 'in_flight' vs G1 enum): disposition = additive `step_claims`, enum untouched. Founder-approved at M03 entry. G2 revisits with the real engine.
- **CONTRA-7 → ADJ-5** (compensation_failed missing from §6 enum vs Finalization B4/§8): disposition = 9th enum value; G1-frozen TDS-02 appendix amended under founder sign-off. Recorded in gate log.
- **EDR-007 definition-identity gap (G2 docket):** no event carries definition_id/version (WorkflowStarted payload = inputs only), so a wiped projection cannot restore them from replay alone. M06 must either event them or the limitation stands documented. NOT silently resolved.
- **EDR-007 waiting-status entry (M07 docket):** no event marks entry into `waiting`; SignalReceived only marks the exit. Wait-state projection lands with M07's wait_records.
- **Noted for M06:** RetryPolicy's full frozen shape (§8: + initial_delay, max_delay, BackoffType enum) exceeds M01's 3-field completion — M06 completes it (pre-M08 mutable).
