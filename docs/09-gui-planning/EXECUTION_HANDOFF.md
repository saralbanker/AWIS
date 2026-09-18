# Execution Handoff

The implementation sequence, ready to assign to engineering agents. Full card specs:
`GUI_STARTLINE_CARDS.md`. Full dependency/staffing detail: `IMPLEMENTATION_ORDER.md`,
`PARALLELIZATION_PLAN.md`. Verification: `VALIDATION_PLAN.md`.

## Assignment table

This repository defines AWIS-specific implementation agents (`.claude/agents/`):
`awis-builder` (mechanical, well-specified implementation), `awis-core-engineer`
(correctness-critical internals — expression grammars, execution engine, EventLog
append/rebuild paths), `awis-verifier` (read-only DoD verification, never edits
production code).

| Card | Assign to | Why |
|---|---|---|
| `E-G0-1` | `awis-builder` | Commit-only, zero design decisions, tests already pass |
| `E-G1-3` | `awis-builder` | Additive sentinel, matches an existing in-repo pattern exactly, zero behavior change for current callers |
| `E-G4-1` | `awis-builder` | Follows `cmd/awis/start.go`'s bootstrap idiom directly — a port, not new design |
| `E-G4-3` | `awis-builder` | New package but conventional (router, DTOs, middleware) — no correctness-critical engine internals touched |
| `E-G4-4/5` | `awis-builder` | Read-only handlers wrapping already-correct engine calls; the hardest part (wait-record wrapper) is a verbatim port of `cmd/awis/status.go:264-348` |
| `E-G4-6` | `awis-builder`, with `awis-verifier` follow-up | The new paginated storage method is mechanical (a direct template exists at `sqlite.go:748/836`), but any change to a read path that previously returned unbounded results is worth an independent verification pass given this repository's history with silent-wrong-answer defects (the B-31/namespace class) |

None of these six cards touches the correctness-critical class (`awis-core-engineer`'s
mandate — expression grammars, signal atomicity, lifecycle FSMs, EventLog append/rebuild
paths) as scoped. That class starts at `E-G2-x` (the `state_changes` migration,
deliberately deferred past this gate — see `DEFERRED_WORK_REGISTER.md`), not at any of
these 6.

## Sequence to hand off, in order

1. **Dispatch in parallel, immediately:** `E-G0-1`, `E-G1-3`, `E-G4-1`, `E-G4-3` — four
   independent agents/tasks, no shared files between any pair except the low-risk
   `internal/storage/sqlite.go` overlap between `E-G1-3` and the not-yet-dispatched
   `E-G4-6` (see below).
2. **On `E-G4-3` landing** (the longer of the two `internal/api`-adjacent tracks):
   dispatch `E-G4-4/5`.
3. **Any time after `E-G4-3` lands**, independent of step 2: dispatch `E-G4-6`. If the
   same agent handled `E-G1-3`, sequence this after it rather than in parallel, to avoid
   two concurrent edits to `internal/storage/sqlite.go`.
4. **Run the gate-level validation** in `VALIDATION_PLAN.md` once `E-G0-1`, `E-G1-3`,
   `E-G4-1`, `E-G4-3`, and `E-G4-4/5` are all merged. `E-G4-6` is not required for this
   checkpoint.
5. **On gate validation passing:** the frontend track (already running against a
   fixture per `GUI_START_LINE.md`) cuts over to the real API. GUI-dominant development
   has formally begun.

## What NOT to include in this handoff

Per `DEFERRED_WORK_REGISTER.md`: do not dispatch any `E-G2-x`, `E-G3-x`, or `E-G4-8/9/10/12/13`
card alongside this batch. They depend on founder decisions (D1, D2) or gate later
phases (Phase 3+), not this gate. Dispatching them now does not speed up the
dashboard-live milestone and consumes engineering capacity that the frontend track's own
day-one start (per `GUI_START_LINE.md`) does not need matched on the engine side.

---

## Recommended Next Card

**Which card should be implemented first?**

`E-G4-3` (`internal/api` skeleton + `/healthz`).

**Why is it first?**

It has zero dependencies of its own, and it is the single card the most downstream work
depends on — both `E-G4-4/5` and `E-G4-6` need it, and nothing needs to precede it.
`E-G0-1` and `E-G1-3` are equally dependency-free but low-leverage: they unblock nothing
else in this set (verified — see `GUI_STARTLINE_CARDS.md`'s note on the false `E-G1-3`→
`E-G0-1` dependency). `E-G4-1` is also zero-dependency and high-value, but it gates
nothing downstream by itself — a running server binary with an empty mux doesn't unblock
route work; the router does. Prioritizing `E-G4-3` first maximizes how much of the
remaining graph opens up per hour of engineering time spent.

**What does it unblock?**

`E-G4-4/5` (read-only workflow + instance routes) and `E-G4-6` (event-history route) —
the entire remaining route surface in this card set. Once it lands, both can be
dispatched in parallel with each other.

**What becomes possible immediately after it lands?**

A running, testable API skeleton — `/healthz` returns 200, the error-mapping middleware
is exercised against every known sentinel, and the router is ready to accept route
registrations without further scaffolding work. Combined with `E-G4-1` (dispatch it
simultaneously, not after — they're independent), this is the moment `cmd/awis-server`
becomes a real, runnable binary for the first time. Route work (`E-G4-4/5`, `E-G4-6`) can
start the same hour, and — because the frontend track has been building against a
code-derived fixture since day one (`GUI_START_LINE.md`) — every route landed from this
point forward is immediately consumable by already-written frontend code with no
additional wiring delay.

**Also start immediately, in parallel, at zero coordination cost:** `E-G0-1` (commit the
B-31 diff — pure hygiene, do it now so it's not sitting in the tree) and `E-G1-3` (the
`GetWorkflow` sentinel — needed by `E-G4-4/5` regardless of when it lands, so there's no
reason to delay it). Dispatching all four zero-dependency cards (`E-G0-1`, `E-G1-3`,
`E-G4-1`, `E-G4-3`) at once, rather than waiting on `E-G4-3` alone, is what reaches the
dashboard-live gate in ≈11.5h instead of ≈20h — see `IMPLEMENTATION_ORDER.md`'s
3-engineer scenario.

---

*Companion documents: `GUI_STARTLINE_CARDS.md` (full specs), `IMPLEMENTATION_ORDER.md`
(sequencing detail), `PARALLELIZATION_PLAN.md` (conflict analysis),
`VALIDATION_PLAN.md` (verification).*
