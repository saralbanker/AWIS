# M10 — AI Execution Context (IKB §4 compilation)
**Model (IMP §28 / IKB §4 row):** Sonnet. All cards dispatch to **`awis-builder` (Sonnet)**;
V1 to **`awis-verifier`**.

**Dispatch order:** C1 → C2 → C3 (serialized; C2 renders what C1 parses, C3's oracle needs both)
→ V1. Prompt: EEOS.md P1 (+P2 on re-dispatch). Branch: `m10-yaml-dsl` (create from `main`
**after M09 squash-merges**; do not branch from `m09-test-infrastructure`).

**Gate:** — (no human gate; non-gated boundary per EEOS phase machine)

**Key constraint: one struct, one validator, two grammars — all pre-existing.** The parser
produces the frozen `core.WorkflowDefinition` (TDS-02 field names verbatim) and hands it to
`internal/validate` (M05). Any dsl code path that re-implements validation logic, a grammar,
or a parallel definition struct is the IMP §27.M10 risk row realized → reject.

**Milestone escalation deltas (beyond identity triggers):**
1. Any field-name mismatch between TDS-02 and what yaml.v3 needs → STOP (frozen-format
   question, E3; never rename or alias a frozen field).
2. Any grammar/validator change needed in internal/expr or internal/validate to make a
   Blueprint §7 / PRD §18 example pass → STOP (frozen-surface conflict).
3. Any new dependency beyond `gopkg.in/yaml.v3` → STOP (IMP L47 dependency policy).
4. FR-WD-02 deep-equal failing for a reason other than a dsl parsing defect (i.e. Builder and
   YAML disagree by design) → STOP (architecture question, not test tuning).
5. Any sdk exported-surface change → STOP (IMP §13 freeze; M10 is internal-only).
6. Existing engine/sdk/harness tests needing modification to stay green → STOP.
