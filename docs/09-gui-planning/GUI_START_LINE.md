# GUI Start Line

**Question:** what is the earliest safe point GUI development can begin?

**Answer: today, for a real slice of it — in parallel with engine work, not after it.**
There are two distinct start lines, not one. Conflating them is what made prior plans
look slower than they are.

---

## Start line 1 — GUI engineering against fixtures: T+0 (today)

Frontend scaffolding, the SPA shell, and a read-only canvas can be built starting now, in
parallel with all G0-G4 engine work, with **no rework risk worth the name** — provided
the fixture is shaped from code and reconciled design docs, not from
`docs/CLI_CONTRACT.md` (which is known-divergent from the binary on ≥11 points; do not
build against it, fixture or otherwise).

**Build the fixture from:**
- `internal/core/event.go`'s `ExecutionEvent` struct — 12 `EventType` consts, each with
  a documented payload shape already in its doc-comment.
- `internal/core/ports.go`'s `StoragePort` method set (`ListPaged`, `Status`,
  `ReadEvents`, `ListWorkflows`, `GetWorkflow`).
- `GUI_ARCHITECTURE.md` §6's route table (already reconciled against the code).
- `GET /instances/{id}` responses including the not-yet-built wait-record wrapper
  fields (`signal_name`, `timeout_remaining_s`) — spec'd in §6, not guessed.
- `WorkflowDefinition` fixtures exercising all six relationship types from §8.1.

**What can be built against it, starting today, with zero engine dependency:**

| Card | What | Effort |
|---|---|---|
| `G-G5-1` | SPA shell & API client | 1d |
| `G-G5-2..5` | Definitions list/detail, instance list/detail, static timeline | 4.5d |
| `G-G5-8` | Read-only canvas — the edge-rendering logic needs a `WorkflowDefinition` fixture, not a live fetch | 2d |
| `G-G6-1` | YAML key-order ADR (design decision, no code dependency on anything) | 1d |

**≈8.5 engineering-days of real product code, startable today**, comparable in size to
G0-G4's own ≈13-day engine critical path. Only `G-G5-6/7` (SSE) and `G-G5-9`
(contract-drift CI, needs a real golden snapshot to diff against) must wait for real
artifacts — everything else in this list converges onto the real API later without a
rebuild, because the fixture was never guessing.

---

## Start line 2 — GUI against real backend data: the minimal engine gate

Not all of G0-G4 (≈13 engineering days) gates this. The true dependency chain, traced
card-by-card (not milestone-by-milestone), is six read-only routes plus two small
prerequisite cards:

| Card | What | Effort | Why it's on the gate |
|---|---|---|---|
| `E-G0-1` | Commit the B-31 remediation | 0.5h | G1 depends on it |
| `E-G1-3` | `GetWorkflow` typed sentinel | 2h | `E-G4-4` depends on it directly |
| `E-G4-1` | `cmd/awis-server` skeleton | 4h | — |
| `E-G4-3` | `internal/api` skeleton | 4h | — |
| `E-G4-4` | Read-only workflow routes | 3h | — |
| `E-G4-5` | Read-only instance routes | 4h | — |
| `E-G4-6` | Event-history route | 3h | — |
| `E-G4-7` | `/healthz` | 0.5h | Direct `G-G5-1` acceptance dependency |

**Total: ≈21 hours ≈ 2.5-3 engineering days of code**, none of it gated on D1, D2, or D4,
and only softly gated on D3 (which already has a safe, recommended default — in-module —
that `E-G4-1` can start against without waiting for ratification).

**This set alone fully unblocks `G-G5-1` through `G-G5-5`** — the complete Phase-1
read-only dashboard — against real data, with zero further engine work.

**If a team prefers to land G0 and G1 as clean, complete milestones** rather than
cherry-picking the two load-bearing cards out of each, the same real-data start line
arrives at ≈4.5-5.5 engineering-days instead (full G0 ≈1-1.5d + full G1 ≈2.6d + this G4
subset's internal ≈1d critical path). Both numbers are correct; they're a tradeoff
between speed and milestone hygiene, not a disagreement about what's required.

---

## What is explicitly NOT on either start line

D1, D2, D4, and "Phase A" (STATE.md/main-merge reconciliation) gate nothing above. All of
G2, all of G3, and G4's mutation/introspection cards (`E-G4-8/9/10/12/13`) are real,
correctly-scoped, eventually-necessary work — but they gate later phases (Phase 2, Phase
3, Phase 4/5), not the GUI start line itself. Full accounting in
`MINIMUM_ENGINE_PROGRAM.md` and `DEFERRED_WORK_REGISTER.md`.

---

*Companion documents: `MINIMUM_ENGINE_PROGRAM.md` (the full card-level MUST/SHOULD/DEFER
classification this start line is drawn from), `EXECUTION_SEQUENCE.md` (the ordered,
staffed implementation path), `DEFERRED_WORK_REGISTER.md`, `FINAL_RECOMMENDATION.md`.*
