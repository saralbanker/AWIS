# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

MILESTONE: M07-signal-subsystem
BRANCH: m07-signal-subsystem
PHASE: B-BUILD
GATE: G2 (pending — verdict after M07 D-CLOSE; IMP §23)
CARDS:
  M07-C1   DONE        (e223dc4)
  M07-C2   DONE        (142b684)
  M07-C3   WITHDRAWN   (superseded by C3r pre-dispatch — CE card-cutting gap, see TRACEABILITY)
  M07-C3r  DISPATCHED  awis-core-engineer
  M07-V1   READY       awis-verifier
BLOCKERS: none
NEXT: on C3r DONE → dispatch V1 on clean tree → D-CLOSE (Fable review + G2 brief)
LAST: 2026-07-08 C3r first run died (session limit, read-phase, nothing on disk); re-dispatch stopped by founder directive: run remaining M07 implementation work on SONNET (downward substitution from IMP §28 Opus row, founder authority — recorded in TRACEABILITY); C3r re-dispatched on core-engineer identity @ sonnet

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06   # history: git log + module HANDOFFs
