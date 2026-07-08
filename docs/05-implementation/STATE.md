# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

MILESTONE: M08-sdk-public-surface
BRANCH: m08-sdk-public-surface
PHASE: B-BUILD
GATE: none
CARDS:
  M08-C1   DONE         (7d23931)
  M08-C2   READY   awis-builder
  M08-V1   READY   awis-verifier
BLOCKERS: none
NEXT: dispatch M08-C2 to awis-builder
LAST: 2026-07-08 M08-C1 DONE (7d23931): Runtime/Config/NewRuntime, RegisterHandler/RegisterWorkflow
      (PRD §18 errors), WorkflowBuilder+semver, core shapes frozen, WorkflowRegistered audit (F-4)

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06 M07   # history: git log + module HANDOFFs
