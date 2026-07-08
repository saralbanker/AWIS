# M07 — AI Execution Context (EEOS §15 D1 stub; operational content lives in cards/)
**Model (IMP §28):** **Opus** — `awis-core-engineer` on ALL implementation cards (atomicity is a
correctness cliff; no downward substitution, EDR-004). Verifier: `awis-verifier` (Sonnet).
**Dispatch order:** C1 → C2 → C3 (serialized, same packages) → V1. Prompt: EEOS.md P1 (+P2 on re-dispatch).
**Gate:** G2 follows this milestone — CE assembles the Gate Brief at D-CLOSE (IMP §23 row G2).
**Milestone escalation deltas (beyond identity triggers):**
1. Any timeout action that cannot be expressed in the closed TDS-01 event vocabulary → STOP
   (frozen post-G1 format; E2/E3-grade — never add an event type).
2. Any deviation needed from the B3 transaction text (order, guards, tx boundary) → STOP (CONTRA).
3. `audit_log` DDL questions beyond PRD §26's shape are E0 (CONTRA-4: additive, unenumerated,
   StoragePort-internal) — decide, record in report DEVIATIONS.
