# Implementation Order

Exact execution order for the 6 cards in `GUI_STARTLINE_CARDS.md`, derived from
verified code dependencies only — no milestone-label ordering.

## Dependency graph

```
E-G0-1 ──────────────────────────────────────────────  (independent, blocks nothing here)

E-G1-3 ──────────────────────────────────┐
                                          ▼
E-G4-1 ────────────┐              E-G4-4/5 ─┐
                    │                        │
E-G4-3 ─────────────┴───▶ (router exists) ───┤──▶ [dashboard-live gate reached]
                                          │
                                    E-G4-6 ──┘ (not required for the gate — see below)
```

Real edges only: `E-G4-4/5` needs `E-G4-3` (router) and `E-G1-3` (sentinel). `E-G4-6`
needs `E-G4-3` (router). `E-G4-1` and `E-G4-3` are mutually independent. `E-G0-1` and
`E-G1-3` are independent of everything else, including each other.

## Stages

**Stage 1 — start immediately, no dependencies:**
- `E-G0-1` (0.5h) — commit whenever convenient; does not gate any later stage
- `E-G1-3` (2h) — independent of `E-G0-1`; do this before `E-G4-6` starts if the same
  engineer owns both, to keep the `internal/storage/sqlite.go` touches sequential
- `E-G4-1` (4h) — independent track
- `E-G4-3` (4.5h, includes the merged `/healthz`) — independent track

**Stage 2 — after `E-G4-3` lands (and `E-G1-3` for the sentinel):**
- `E-G4-4/5` (5-7h) — needs the router (`E-G4-3`) and the sentinel (`E-G1-3`)
- **→ Dashboard-live gate reached here.** `E-G4-6` is not required for this gate — only
  `G-G5-5` (the static timeline view) needs it, not the definitions/instances list and
  detail views.

**Stage 3 — can run any time after `E-G4-3`, does not block Stage 2's gate:**
- `E-G4-6` (3h, includes its storage-layer sub-scope) — sequence after `E-G1-3` if the
  same engineer owns both `internal/storage/sqlite.go` touches; otherwise fully
  parallel with `E-G4-4/5`

## Two staffing scenarios

**Single engineer, strictly sequential:** `E-G0-1` → `E-G1-3` → `E-G4-1` → `E-G4-3` →
`E-G4-4/5` → `E-G4-6`. Total: 0.5+2+4+4.5+6+3 = **20h ≈ 2.5 engineering-days.**

**Three engineers, maximum parallelism** (see `PARALLELIZATION_PLAN.md` for the
per-pair conflict analysis):
- Engineer A: `E-G0-1` (0.5h) → `E-G1-3` (2h) → idle until Stage 2, or start `E-G4-6`
  early if willing to touch `sqlite.go` twice non-adjacently
- Engineer B: `E-G4-1` (4h)
- Engineer C: `E-G4-3` (4.5h)
- Once B and C both land (bottleneck: C at 4.5h) and A's `E-G1-3` is in (2h, already
  done): Engineer A or C takes `E-G4-4/5` (5-7h); Engineer B or C takes `E-G4-6` (3h),
  in parallel.
- **Dashboard-live gate reached at ≈4.5h (skeleton) + ≈7h (worst-case `E-G4-4/5`) =
  ≈11.5h ≈ 1.5 engineering-days**, if `E-G4-4/5` is staffed as the critical path and
  `E-G4-6` runs alongside it on a separate engineer without contention.

Either scenario is consistent with the prior ≈2.5-3 day estimate in `GUI_START_LINE.md`;
the 3-engineer scenario is the one that actually beats it, and only because `E-G4-1`
and `E-G4-3` — previously assumed sequential via the old G0→G1→G4 milestone chain — are
verified independent and can run side by side from hour zero.

---

*Companion documents: `GUI_STARTLINE_CARDS.md` (full card specs),
`PARALLELIZATION_PLAN.md`, `VALIDATION_PLAN.md`, `EXECUTION_HANDOFF.md`.*
