# M09 — AI Execution Context (IKB §4 compilation)
**Model (IMP §28 / IKB §4 row):** Sonnet — "wiring deterministic sources through an existing
engine." All cards dispatch to **`awis-builder` (Sonnet)**; V1 to **`awis-verifier`**.

**Dispatch order:** C1 → C2 → C3 (serialized; C2 needs C1's seams, C3 needs C2's harness) → V1.
Prompt: EEOS.md P1 (+P2 on re-dispatch). Branch: `m09-test-infrastructure` (create from `main`
**after M08 squash-merges**; do not branch from `m08-sdk-public-surface`).

**Gate:** — (no human gate; non-gated boundary per EEOS phase machine)

**Key constraint: the harness drives the REAL engine.** Any harness code path that re-implements
engine semantics (its own step loop, its own signal delivery, its own state projection) is the
IMP §27.M9 risk row realized → reject. The harness may only: construct a Runtime, register,
submit, tick, signal, and read status through the sdk surface (plus `internal/*` helpers for
assertions).

**Surface discipline (post-M08 freeze, IMP §13):** M09 may ADD `sdk/testing/*` (new package),
`sdk.DeterministicMode()`, and the two additive `sdk.Config` fields (Clock, NewID) — all
mandated by FR-SDK-06/07/08. It may not change or remove any existing sdk exported identifier,
nor any of the four M08-frozen core shapes.

**Milestone escalation deltas (beyond identity triggers):**
1. Any change to an EXISTING sdk exported identifier or an M08-frozen shape → STOP (IMP §13).
2. Any new sdk exported symbol beyond DeterministicMode + Config fields + `sdk/testing`
   package surface named in the spec → STOP (surface creep).
3. Any new StoragePort method (Blueprint §20 frozen) or schema migration → STOP.
4. Harness logic that duplicates engine semantics instead of calling the engine → STOP.
5. QG-5 test needing > 1s or an external service to pass → STOP (architecture question,
   not a test-tuning question).
6. Existing engine/signal/storage tests needing modification (beyond additive fixtures) to
   stay green → STOP (regression risk; the seams must be nil-safe no-ops).
