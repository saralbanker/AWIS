# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

# ── M07 awaiting founder merge + G2 verdict ──────────────────────────────────
M07-PENDING-MERGE:
  BRANCH:      m07-signal-subsystem
  GATE:        G2 (human verdict required — see G2 brief below)
  STATE:       E-MERGE (blocked on founder; squash-merge when G2 approved)
  G2-QUESTION: "Do the engine and signal implementations match the Finalization specifications
                exactly (atomic transaction, cancellation FSM, fallback rules)?"
  G2-EVIDENCE:
    - B3 tx: deliver.go guard→append→resume in ONE SQLite tx; crash-injection PASS
    - Double-delivery no-op: DeliverNoOp on 2nd call; zero new rows/events
    - B4.6 cancel: wait_records deleted on both waiting-cancel and running-cancel paths
    - Timeout actions: fail/compensate/continue all within closed TDS-01 12-type vocabulary
    - Latency: 1.72 ms/op (116× under NFR-P-04 200ms)
    - V1: PASS at HEAD 6ac948a (awis-verifier, 2026-07-08)
  G2-CONCERN:  C3r executed on Sonnet (downward substitution, founder-directed; IMP §28 requires
               Opus for M07). Mitigations: crash-injection checkpoint (§20.M7), independent V1.
               Weigh when judging atomicity evidence.
  PR-BODY:     see D-CLOSE report (2026-07-08)

# ── M08 active milestone ──────────────────────────────────────────────────────
MILESTONE: M08-sdk-public-surface
BRANCH: m08-sdk-public-surface  (create from main AFTER M07 squash-merges)
PHASE: B-BUILD
GATE: none
CARDS:
  M08-C1   READY   awis-builder
  M08-C2   READY   awis-builder
  M08-V1   READY   awis-verifier
BLOCKERS: M07 merge (branch prerequisite — do NOT start M08-C1 until M07 is squash-merged to main)
NEXT: await M07 merge → create branch m08-sdk-public-surface from main → dispatch M08-C1 to awis-builder
LAST: 2026-07-08 D-CLOSE: M07 semantic review PASS; HANDOFF actuals + TRACEABILITY + CHECKLIST updated;
      G2 brief assembled; M08 A-INIT complete (6 module files + 3 cards written READY)

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06   # history: git log + module HANDOFFs
# M07 moves to DONE-MILESTONES after founder squash-merge
