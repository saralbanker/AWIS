# Dependency Index
Source: IMP §9 (DAG) + §10 (critical path) + §11 (parallel matrix), amended F-1..F-5.

## Edges (blocker → blocked)
```
M00 → M01[G1] → {M02, M04, M05}
M02 → M03 → M06
{M04, M05} → M06 → M07[G2] → {M08, M11}
M08 → {M09, M10, M14}
M11 → M12 → M13
{M09, M10, M13, M14} → M15[G3] → {M16, M17} → M18[G4]
```

## Critical path (~28 days)
M00 → M01 → M02 → M03 → M06 → M07 → M08 → M11 → M12 → M13 → M15 → M18

## Float carriers (schedule-pressure absorbers, IMP §10)
M04, M05, M09, M10, M14, M16, M17. Week-5 float pre-starts M16/M17 mechanical work (F-6 advisory).

## Gate stops
- **G1 (after M01):** the only hard full-stop — nothing downstream starts unapproved.
- **G2 (after M07), G3 (after M15), G4 (after M18):** review completed work; float work continues during review.
- **TDS-06 sign-off:** inside M15, before OIP handlers are written.

## Cross-milestone artifact flows
Per-milestone detail: each module's DEPENDENCY_MAP.md. Headline flows:
TDS-01/02 (M01) → migrations/replay (M02/M03/M06) · TDS-03 corpus (M01) → parser oracle (M05) ·
StoragePort contract suite (M02) → V2-Postgres reuse artifact · protocol TDS-04/05 (M11/M12) → Python libs ·
git-context-plugin (M13) → OIP assemble-context (M15) · TDS-07 (M14 day 1, F-3) → golden CLI fixtures (M14/M17).
