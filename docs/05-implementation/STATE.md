# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

MILESTONE: M08-sdk-public-surface
BRANCH: m08-sdk-public-surface
PHASE: C-VERIFY
GATE: none
CARDS:
  M08-C1   DONE         (7d23931)
  M08-C2   DONE         (92828e7)
  M08-C2r  DONE         (e8aca86)
  M08-V1   DISPATCHED   awis-verifier
BLOCKERS: none
NEXT: awis-verifier returns report → all-✅ → STATE→D-CLOSE
LAST: 2026-07-08 M08-C2r DONE (e8aca86): 5 test gaps closed; V1 re-dispatched

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06 M07   # history: git log + module HANDOFFs
