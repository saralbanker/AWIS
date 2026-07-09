# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

# ── M08 awaiting founder merge ────────────────────────────────────────────────
M08-PENDING-MERGE:
  BRANCH:      m08-sdk-public-surface
  GATE:        none (non-gated boundary; squash-merge on founder review)
  STATE:       E-MERGE (blocked on founder)
  MERGE-RECOMMENDATION: APPROVED FOR SQUASH MERGE — D-CLOSE semantic review (Fable, 2026-07-08)
    found NO architectural drift:
    - internal/core/ports.go delta = exactly the four M08-owned shape completions
      (WorkflowStatus, HistoryQuery, ExecutionRecord, StepStatistics), field-for-field per spec
    - frozen surfaces untouched (EventLog format, port interfaces, F-1 alias targets,
      StoragePort 12-method set)
    - F-2 honored (TriggerAPI → engine.Ingest only); F-4 honored (WorkflowRegistered audit)
    - FR-SDK-09 boundary proven (examples/ import grep empty; re-checked at D-CLOSE)
    - sdk.SQLiteStorage + Runtime.Tick verified canonical (Blueprint §12 L934 region, verbatim)
    - build+test+e1 re-run green at HEAD during D-CLOSE
  EVIDENCE:    V1 PASS 21/21 (awis-verifier, 2026-07-08, at 728cfed) after C2r closed the
               first-pass FAIL (5 test gaps). Full record: module TRACEABILITY execution
               record + ticked VALIDATION_CHECKLIST + HANDOFF actuals (merge sha fills at merge).
  PR-BODY:     see D-CLOSE report (2026-07-08); diff = main...m08-sdk-public-surface
               (14 files, +1649/−12; module cards under docs/05-implementation/M08-*/cards/)

# ── M09 active milestone ──────────────────────────────────────────────────────
MILESTONE: M09-test-infrastructure
BRANCH: m09-test-infrastructure  (create from main AFTER M08 squash-merges)
PHASE: B-BUILD
GATE: none
CARDS:
  M09-C1   DONE    awis-builder   b64723d
  M09-C2   DONE    awis-builder   3764a3a
  M09-C3   DONE    awis-builder   c8fc10b
  M09-V1   READY   awis-verifier
BLOCKERS: none
NEXT: dispatch M09-V1 to awis-verifier
LAST: 2026-07-09 M09-C3 DONE (c8fc10b): internal/fixtures (8 workflow shapes), sdk/testing/
      integration_test.go (7 §19 shape tests + determinism proof), qg5_test.go (QG-5 < 1s),
      checkpoint_m9_test.go (M9 OIP-shaped < 1s); sdk/events.go + harness.go ReadEvents additive;
      make build test lint race e1 all green

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06 M07   # history: git log + module HANDOFFs
# M08 moves to DONE-MILESTONES after founder squash-merge
