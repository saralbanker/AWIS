# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

MILESTONE: M07-signal-subsystem
BRANCH: m07-signal-subsystem
PHASE: B-BUILD
GATE: G2 (pending — verdict after M07 D-CLOSE; IMP §23)
CARDS:
  M07-C1  READY  awis-core-engineer
  M07-C2  READY  awis-core-engineer
  M07-C3  READY  awis-core-engineer
  M07-V1  READY  awis-verifier
BLOCKERS: none
NEXT: dispatch M07-C1 (then C2 → C3, serialized — same packages; then V1 on clean tree)
LAST: 2026-07-08 founder squash-merged M06 → main b691965 (tag milestone/M06); CE flipped ledger per staged block; M06 HANDOFF actuals filled

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06   # history: git log + module HANDOFFs
