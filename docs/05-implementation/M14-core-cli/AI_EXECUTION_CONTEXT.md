# M14 — AI Execution Context (IKB §4 compilation)
**Model (IMP §28):** Sonnet. All cards to **`awis-builder` (Sonnet)**; V1 to **`awis-verifier`**.
Dispatch order: C1 → C2 → C3 → C4 (C2/C3 both build on C1's skeleton; serialized to keep the
mux/renderer stable) → V1. Prompt: EEOS.md P1 (+P2 on re-dispatch).
Branch: `m14-core-cli` (stacked on `m13-git-context-plugin`).

**Gate:** — (no human gate)

**Key constraints:**
1. **The F-3 decision is made** (SPEC "F-3 DECISION" + CONTRA-5): direct SQLite WAL access +
   PID/SIGTERM for stop. TDS-07 transcribes it; builders implement it. Re-opening the socket
   question is a STOP, not a design space.
2. **PP-6:** every command ships `--json` in the same card that ships the command. A command
   without --json is a defect.
3. **CLI is a CONSUMER**: it calls sdk/dsl/plugin/storage APIs that exist. Any needed change
   to those layers → STOP (E1). The one sanctioned touch: none. (Audit write for
   WorkflowRegistered goes through the existing audit API from the CLI/sdk registration path
   WITHOUT modifying sdk — if that proves impossible without an sdk change → STOP, CE will
   adjudicate.)
4. **No new dependencies** — stdlib flag mux, no cobra.

**Milestone escalation deltas:**
1. TDS-07 gap discovered mid-C2/C3/C4 → STOP (E3) — the contract is C1's output; later cards
   never extend it silently.
2. Golden-test nondeterminism → fix via injected clock/id seams that ALREADY exist
   (DeterministicMode); never sleep/retry in goldens.
3. Crash-recovery system test failing for an engine reason → STOP (E1; engine is frozen).
