# GUI Beta — Execution Order

The fastest realistic path through the 13 cards in `GUI_BETA_WORK_BREAKDOWN.md`. Real
dependency graph first (only two edges exist), then staffing.

---

## Dependency graph

```
BE-1 (definition_id filter) ──────────────▶ FE-2 (definition_id filter UI)
BE-2a (shared version const) ──▶ BE-2b (/api/v1/info) ──▶ FE-6b (consume /info)

Every other card: zero dependencies.
```

That's the entire graph. `FE-1` (Workflow Detail, the largest card) has zero
dependencies and can start immediately — the API route it needs already exists and
returns everything required (`GUI_BETA_API_REQUIREMENTS.md`). Ten of the thirteen cards
are independent of every other card and of both backend additions.

## File-contention note (not a dependency, but real)

`instanceList.ts` is touched by four cards (`FE-2`, `FE-4a`, `FE-4b`, `FE-7`) and
`instanceDetail.ts` by two (`FE-4a`, `FE-4b`). None of these *depend* on each other, but
staffing them as four fully independent parallel engineers on the same file invites
avoidable merge conflicts. Recommendation: one engineer works this cluster as a single
continuous session, in the order below, rather than splitting it across people.

## Recommended order within the `instanceList.ts`/`instanceDetail.ts` cluster

1. `FE-7` first — switches the table to keyed `repeat()`. Every other change to this
   file lands on top of that structural change more cleanly than the reverse order.
2. `FE-4a` + `FE-4b` together — both are the "freshness" pair and touch overlapping
   render logic (the header/status area); doing them in one pass avoids two separate
   diffs touching the same lines.
3. `FE-2` last, once `BE-1` has landed — purely additive (one more filter control), no
   reason to block on it earlier.

---

## Staffing scenarios

**Single engineer, sequential:** sum of all 13 cards ≈ 36.5h ≈ **4.5-5 working days.**
Order: `BE-2a` → `BE-1` ∥ `BE-2b` (both quick, either order) → `FE-1` (the long pole,
do it first among frontend work so it's not blocking a late Beta cut) → the
`instanceList`/`instanceDetail` cluster (`FE-7` → `FE-4a`/`FE-4b` → `FE-2`) → remaining
small independent cards (`FE-3`, `FE-5`, `FE-6a`, `FE-6b`, `FE-8`) in any order.

**Two engineers, parallel:**
```
Engineer A: FE-1 (Workflow Detail)                              [10-14h, ~1.5-2 days]

Engineer B: BE-2a (1.5h) → BE-1 ∥ BE-2b (3h/2h, either order)   [~6.5h]
            → FE-7 → FE-4a → FE-4b → FE-2                        [~9h]
            → FE-3, FE-5, FE-6a, FE-6b, FE-8 (any order)         [~9h]
            total: ~24.5h ≈ ~3 days
```
**Beta cut point: ~3 days**, gated by Engineer B's combined workload (Engineer A's `FE-1`
finishes sooner). Reassign a couple of Engineer B's small independent cards (`FE-5`,
`FE-8`) to Engineer A once `FE-1` lands, to flatten this to closer to 2.5 days if that
matters.

**Three engineers:** dedicate one to the three `BE-*` cards (done well within day 1,
~6.5h), freeing Engineers A and B to split frontend work purely by file-contention
boundaries — Engineer A on `FE-1` (Workflow Detail, its own new file), Engineer B on the
`instanceList`/`instanceDetail` cluster plus the small independent cards. This does not
shorten the critical path below `FE-1`'s own ~1.5-2 days (nothing shortens a single
cohesive screen by adding more people to it), but removes any chance of backend cards
becoming an accidental bottleneck.

## Beta cut point — what must be true

All 13 cards landed; `GUI_BETA_PRD.md`'s acceptance criteria for F1-F8 all pass; full
regression per `GUI_BETA_FINAL_RECOMMENDATION.md`'s verification list (build, type-check,
`go test ./...`, and a real-browser pass through Workflow Detail specifically, since it's
new and the largest surface added this pass).

---

*Companion documents: `GUI_BETA_GAP_ANALYSIS.md`, `GUI_BETA_PRD.md`,
`GUI_BETA_API_REQUIREMENTS.md`, `GUI_BETA_WORK_BREAKDOWN.md`,
`GUI_BETA_FINAL_RECOMMENDATION.md`.*
