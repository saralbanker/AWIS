# M13 — AI Execution Context (IKB §4 compilation)
**Model (IMP §28):** Sonnet/Haiku. C1/C2 dispatch to **`awis-builder` (Sonnet)**; V1 to
**`awis-verifier`**. Dispatch order: C1 → C2 → V1. Prompt: EEOS.md P1 (+P2 on re-dispatch).
Branch: `m13-git-context-plugin` (stacked on `m12-plugin-system`).

**Gate:** — (no human gate)

**Key constraint:** this plugin is a CONSUMER of M12's frozen artifacts. Any change needed in
internal/plugin, awis_plugin, TDS-05, or the goldens → STOP (E1) — a reference plugin that
needs platform changes is an M12 defect, not an M13 fix.

**Milestone escalation deltas:**
1. Blueprint §11 manifest field deviation beyond the pinned command/args → STOP (E3).
2. Any Python dependency → STOP (PR-5; the zero-dep pin is CE-set).
3. Input-key-set resolution failing for `handler: git-context-plugin` → STOP (E1; that rule
   is M12 frozen behavior the OIP fixture depends on).
