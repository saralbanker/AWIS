# M11 — AI Execution Context (IKB §4 compilation)
**Model (IMP §28 / IKB §4 row):** Sonnet. All cards dispatch to **`awis-builder` (Sonnet)**;
V1 to **`awis-verifier`**.

**Dispatch order:** C1 → C2 → C3 (serialized; C2 implements the protocol C1 froze, C3's
pytest + e2e need both) → V1. Prompt: EEOS.md P1 (+P2 on re-dispatch).
Branch: `m11-subprocess-runner` (stacked on `m10-yaml-dsl` per founder directive 2026-07-10;
M10 awaits founder merge in parallel).

**Gate:** — (no human gate; non-gated boundary per EEOS phase machine)

**Key constraint: the golden files are the single wire truth for BOTH sides** (IMP §27.M11
risk row). Go runner tests and the pytest suite assert against the SAME files under
`internal/runner/subprocess/testdata/protocol/`. Any Go↔Python behavior difference that the
goldens do not arbitrate means the goldens are incomplete — extend the goldens (C1's contract),
never special-case one side.

**Milestone escalation deltas (beyond identity triggers):**
1. TDS-04 envelope decisions are CE-pinned in IMPLEMENTATION_SPEC §1 — any need to change a
   frozen field name or mapping → STOP (E3), never a local rename.
2. Any engine.Runner interface change needed → STOP (frozen M06 surface).
3. Any sdk exported-surface change beyond the runtime.go runners-map wiring line → STOP
   (IMP §13 freeze).
4. Any new Go dependency or any Python runtime dependency → STOP (dependency policy).
5. Existing engine/sdk/harness/dsl tests needing modification to stay green → STOP.
6. The e2e keystone failing for a non-M11 reason (engine/harness defect) → STOP (do not
   patch other milestones' code).
