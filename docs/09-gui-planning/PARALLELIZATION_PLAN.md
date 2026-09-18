# Parallelization Plan

What can execute simultaneously, what needs light coordination, and what is strictly
sequential — classified by real file/symbol overlap, not by card ID proximity.

## Safe to run fully in parallel — zero file overlap

| Pair | Files touched | Why safe |
|---|---|---|
| `E-G0-1` ∥ any other card | `internal/validate/*`, scaffolds, docs | Disjoint from every other card's files entirely |
| `E-G4-1` ∥ `E-G4-3` | `cmd/awis-server/main.go` (NEW) vs. `internal/api/*.go` (NEW) | No shared file; join only at a thin import in `main.go` once both exist |
| `E-G1-3` ∥ `E-G4-1` | `internal/storage/sqlite.go` vs. `cmd/awis-server/main.go` | Disjoint |
| `E-G1-3` ∥ `E-G4-3` | `internal/storage/sqlite.go` vs. `internal/api/*.go` | Disjoint |
| `E-G4-4/5` ∥ `E-G4-6` | `internal/api/workflows.go`+`instances.go` vs. `internal/storage/sqlite.go`+`internal/api/events.go` | Different files; only shared touchpoint is the router-registration list, see below |

## Needs light coordination — shared file, low conflict risk

| Pair | Shared file | Coordination needed |
|---|---|---|
| `E-G1-3` and `E-G4-6` | `internal/storage/sqlite.go` (different, non-adjacent line ranges — `E-G1-3` near lines 37-58/296, `E-G4-6` adds a new method near the `ReadEvents`/pagination region) | Sequence one after the other, or assign one owner to both. Merge conflict risk is low (non-adjacent edits) but real — do not land both from separate branches without a rebase check. |
| `E-G4-4/5` and `E-G4-6` | `internal/api/router.go` (both add route-registration lines) | Each adds independent lines; whoever lands second does a trivial rebase. Not a reason to serialize the actual route-handler work, only the final registration merge. |

## Strictly sequential — real dependency, cannot start early

| Card | Blocked on | Why |
|---|---|---|
| `E-G4-4/5` | `E-G4-3` (router must exist) | Handlers register against `E-G4-3`'s mux and reuse its error-mapping middleware |
| `E-G4-4/5` | `E-G1-3` (sentinel must exist) | The 404-vs-500 acceptance criterion depends on `ErrWorkflowNotFound` existing |
| `E-G4-6` | `E-G4-3` (router must exist) | Same reason as above |

## Explicitly NOT a dependency (verified, do not serialize these)

- `E-G1-3` on `E-G0-1` — disjoint files, no shared symbol. This was the one false
  dependency the review found; treat it as fully independent.
- `E-G4-1` on `E-G4-3`, or vice versa — the server binary compiles and runs (against an
  empty mux) without the API package existing yet, and the API package's own tests don't
  need a running server.
- `E-G4-6` on `E-G4-4/5` — both depend only on `E-G4-3`; there is no reason to make one
  wait for the other, and `E-G4-6` is not required for the dashboard-live gate at all
  (see `IMPLEMENTATION_ORDER.md` Stage 3).

## Recommended staffing for maximum parallelism (3 engineers)

```
t=0        Engineer A: E-G0-1 (0.5h) → E-G1-3 (2h)  [done t=2.5h]
t=0        Engineer B: E-G4-1 (4h)                   [done t=4h]
t=0        Engineer C: E-G4-3 (4.5h)                 [done t=4.5h]

t=4.5h     Engineer A or C: E-G4-4/5 (5-7h)           [dashboard-live gate at t≈11.5h]
t=4.5h     Engineer B or C: E-G4-6 (3h)               [done t≈7.5h, independent of gate]
```

If only one or two engineers are available, drop to the sequences in
`IMPLEMENTATION_ORDER.md`'s two-scenario table — the dependency graph itself doesn't
change, only how many of the "safe to parallelize" pairs actually run concurrently.

---

*Companion documents: `GUI_STARTLINE_CARDS.md`, `IMPLEMENTATION_ORDER.md`,
`VALIDATION_PLAN.md`, `EXECUTION_HANDOFF.md`.*
