# M08 — AI Execution Context (IKB §4 compilation)
**Model (IMP §28 / IKB §4 row):** Mixed — Opus API review + Sonnet implementation.
Per model freeze (Baseline V1, founder-directed): ALL cards dispatch to **`awis-builder` (Sonnet)**.
The "API review" portion is absorbed into V1 verification + the G2-equivalent human read-through
at PR time. If any sdk exported symbol decision requires Opus-grade architecture reasoning, STOP (E1).

**Dispatch order:** C1 → C2 (serialized; same `sdk/` package) → V1.
Prompt: EEOS.md P1 (+P2 on re-dispatch). Branch: `m08-sdk-public-surface` (create from `main`
**after M07 squash-merges**; do not branch from `m07-signal-subsystem`).

**Gate:** — (no human gate; non-gated boundary per EEOS phase machine)

**Key constraint: surface freeze.** After M08 merges, any change to an sdk exported identifier
requires written justification citing a frozen doc (IMP §13). M08 OWNS the freeze of:
- `core.WorkflowStatus` (shape: complete at M08)
- `core.HistoryQuery`, `core.ExecutionRecord`, `core.StepStatistics` (shape: complete at M08)
These shapes are not part of G1 (they are not EventLog/WorkflowDefinition format) but are
immutable after M08 merges. Choose minimal, complete shapes per Blueprint §12 + PRD.

**Milestone escalation deltas (beyond identity triggers):**
1. Any sdk exported symbol not present in Blueprint §12, IMP §27.M8, or IMP §17 L322 → STOP
   (surface creep; IMP §27.M8 risk row).
2. Any modification to a frozen F-1 type alias already in `sdk/` → STOP (CONTRA).
3. Any new StoragePort method (12-method interface, Blueprint §20) → STOP.
4. TriggerAPI implementation that bypasses the domain_events table or F-2's SCAN_TRIGGERABLE
   path → STOP (F-2 amendment is binding).
5. WorkflowRegistered audit call site omitted from RegisterWorkflow → STOP (F-4 binding).
