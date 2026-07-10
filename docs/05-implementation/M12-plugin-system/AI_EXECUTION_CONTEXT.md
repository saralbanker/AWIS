# M12 — AI Execution Context (IKB §4 compilation)
**Model (IMP §28 / IKB §4 row):** Mixed — Opus-designated FSM edge cases. Founder directive
2026-07-10: no Opus subagents; substitution is UPWARD (EEOS rule 8) — all cards dispatch to
**`awis-builder` (Sonnet)** against the CE-pinned FSM/protocol in IMPLEMENTATION_SPEC, and
**Fable performs the adversarial FSM review at D-CLOSE** (the five spec invariants are the
review targets). V1 to **`awis-verifier`**.

**Dispatch order:** C1 → C2 → C3 → C4 (serialized: C2 stores what C1 parses; C3 runs what C2
registers; C4 speaks what C3 serves) → V1. Prompt: EEOS.md P1 (+P2 on re-dispatch).
Branch: `m12-plugin-system` (stacked on `m11-subprocess-runner`).

**Gate:** — (no human gate)

**Key constraint: the FSM and protocol are CE-pinned.** IMPLEMENTATION_SPEC's "CE-pinned"
sections are frozen design, not suggestions: state names, transition rules, crash-counter
semantics, error codes, envelope fields, resolution rule. A builder who needs to deviate
STOPs (E1/E3) — never a local redesign. Goldens are the single wire truth for Go and Python.

**Milestone escalation deltas (beyond identity triggers):**
1. Any change to the pinned FSM transitions, crash-counter rules, or error codes → STOP (E1).
2. Any change to frozen Blueprint §11 artifacts (registry SQL, manifest fields, lifecycle
   state names) → STOP (E3).
3. Any StoragePort 12-method-set change → STOP; PluginStore must be additive type-assert
   (audit.go precedent).
4. Any sdk exported-surface change → STOP (wiring inside NewRuntime/Stop only).
5. Any new Go dependency or Python runtime dependency → STOP.
6. Existing tests (M05–M11) needing modification → STOP.
7. Flaky lifecycle tests (sleeps, races) → fix via the clock/interval seam, never by widening
   sleeps past the ≤2s budget; still flaky → STOP (E1, FSM design question).
