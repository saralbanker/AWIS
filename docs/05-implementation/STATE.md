# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

MILESTONE: M07-signal-subsystem
BRANCH: m07-signal-subsystem
PHASE: B-BUILD
GATE: G2 (pending — verdict after M07 D-CLOSE; IMP §23)
CARDS:
  M07-C1  DONE        (e223dc4)
  M07-C2  DISPATCHED  awis-core-engineer
  M07-C3  READY       awis-core-engineer
  M07-V1  READY       awis-verifier
BLOCKERS: none
NEXT: on C2 DONE → dispatch C3; then V1 on clean tree
LAST: 2026-07-08 CE accepted C1 (tests+e1 re-confirmed at HEAD); golangci-lint 2.12.2 installed to close C1's lint env gap; dispatched C2

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06   # history: git log + module HANDOFFs
