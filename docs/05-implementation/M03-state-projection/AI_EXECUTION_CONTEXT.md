# M03 — AI Execution Context
**Model allocation (IMP §28):** Sonnet implementation + deep review on rebuild fidelity ("Opus review" row → CE/Fable review per EDR-004). No gate; CONTRA-6/7 adjudicated at entry (founder, 2026-07-03).

## Session loading order (≈12k)
1. `docs/00-foundation/README.md`
2. This module's IMPLEMENTATION_SPEC + VALIDATION_CHECKLIST
3. TDS-01 §2–§3 (event payloads + replay statements) — projection ground truth
4. M02 HANDOFF (storage layer contract) + `internal/storage` code as card names it

## Hard constraints
- InstanceStatus now has NINE values (ADJ-5); `compensation_failed` is terminal.
- Claim mechanism per CONTRA-6: `step_claims` table; NEVER an 'in_flight' status write.
- Projection rules are EDR-007-recorded; a projection choice not in EDR-007 → STOP.
- Rebuild is a LIBRARY function; no CLI wiring (M14/M17).
- Byte-identical means byte-identical: row-level snapshot equality, not semantic equality.
- Registry/cache/EventLog code from M02 is contract-locked: extend tests, never alter behavior.

## Escalate (STOP) when
- Any projection rule needed that EDR-007 doesn't state.
- The frozen 12-method surface can't express a needed operation (e.g., claim release) — the EDR-006 release-site decision covers SETTLE-time release only.
- Fixture replay reveals an event sequence the rules can't project deterministically.
