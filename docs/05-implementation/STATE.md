# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

MILESTONE: M08-sdk-public-surface
BRANCH: m08-sdk-public-surface
PHASE: B-BUILD
GATE: none
CARDS:
  M08-C1   DONE         (7d23931)
  M08-C2   DONE         (92828e7)
  M08-C2r  READY        awis-builder
  M08-V1   READY        awis-verifier
BLOCKERS: none
NEXT: dispatch M08-C2r to awis-builder (5 V1 test gaps: Signal test, Cancel assertion, QueryHistory InstanceID, StepStats, audit_log direct SELECT)
LAST: 2026-07-08 M08-V1 FAIL (5 test gaps — no code defects); C2r cut; STATE→B-BUILD

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06 M07   # history: git log + module HANDOFFs
