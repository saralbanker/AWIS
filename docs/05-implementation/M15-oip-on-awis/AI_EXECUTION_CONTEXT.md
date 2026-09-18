# M15 — AI Execution Context
**Model:** Mixed (IMP §28) — Sonnet cards; the G3 verdict + TDS-06 sign-off are HUMAN-ONLY
(no model renders them; founder directive covers building to the gate, at-risk, nothing
irreversible pre-merge). Dispatch: P0 → C1 → C2 → C3 → V1 (P1 prompt).
Branch: `m15-oip-on-awis` (stacked on m14-core-cli).

**Gate:** G3 + TDS-06 sign-off at E-MERGE (founder).

**Key constraints:**
1. **QG-4 boundary:** apps/oip imports the sdk module ONLY (never internal/*). The ONLY
   platform commit in this milestone is P0 (sdk.LoadWorkflowFile seam — CE-pinned, IMP §17
   grounded, disclosed to G3). Any OTHER platform edit = milestone failure (RB row), STOP.
2. Frozen fixtures apps/oip/workflows/*.yaml register UNCHANGED (byte-identical).
3. TDS-06 carries the DRAFT — PENDING FOUNDER SIGN-OFF banner until E-MERGE.
4. QG-3 = NullAdapter throughout; intelligence steps complete via fallback/degraded paths.

**Escalation deltas:** any second platform gap (beyond P0) → STOP (E1: G3-relevant, CE must
record); Constitution-article conflict in the record format → STOP (E2 CONTRA, never local);
oip.db needing platform storage access → STOP (B5 Option A violation).
