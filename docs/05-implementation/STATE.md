# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

# ── M10 awaiting founder merge ────────────────────────────────────────────────
MILESTONE: M10-yaml-dsl
BRANCH: m10-yaml-dsl
PHASE: E-MERGE (blocked on founder)
GATE: none (non-gated boundary; squash-merge on founder review)
CARDS:
  M10-C1   DONE    awis-builder   a575113
  M10-C2   DONE    awis-builder   b958c79
  M10-C3   DONE    awis-builder   872974b
  M10-V1   DONE    awis-verifier  f76b6ac
BLOCKERS: none
MERGE-RECOMMENDATION: APPROVED FOR SQUASH MERGE — D-CLOSE semantic review (Fable, 2026-07-10)
  found NO architectural drift:
  - frozen-surface diffs EMPTY (internal/core, engine, storage, validate, expr, sdk)
  - milestone purely additive: internal/dsl + 5 YAML fixtures + docs/DSL.md + yaml.v3 dep
  - no re-implemented graph semantics (validate.Validate single authority; §27.M10 risk clear)
  - deviations (schema_version default 1; wait_signal.name alias) endorsed: parse-layer only
  - gates re-run at HEAD: build/test/lint(0)/race/e1/docs-lint all green
EVIDENCE: V1 PASS 17/17 (awis-verifier, 2026-07-10, at cd2429b). Full record: module
  TRACEABILITY execution record + ticked VALIDATION_CHECKLIST + HANDOFF actuals.
PR-BODY: diff = main...m10-yaml-dsl (26 files, +2542/−30 incl. ledger docs;
  module cards under docs/05-implementation/M10-yaml-dsl/cards/)
NEXT: founder squash-merge (E-MERGE, human-only); M11 proceeds stacked on m10-yaml-dsl
  per founder directive 2026-07-10 ("complete all milestones")

# ── M11 awaiting founder merge ────────────────────────────────────────────────
MILESTONE: M11-subprocess-runner
BRANCH: m11-subprocess-runner (stacked on m10-yaml-dsl)
PHASE: E-MERGE (blocked on founder)
GATE: none (non-gated boundary)
CARDS:
  M11-C1   DONE    awis-builder   5f3cb87
  M11-C2   DONE    awis-builder   febae93
  M11-C3   DONE    awis-builder   145a266
  M11-V1   DONE    awis-verifier  (report 2026-07-10 at 5cc1a51; PASS, 1 row CE-adjudicated)
BLOCKERS: none
MERGE-RECOMMENDATION: APPROVED FOR SQUASH MERGE — D-CLOSE (Fable, 2026-07-10): frozen surfaces
  untouched; runner reviewed line-by-line (process-group kill, envelope-wins, ReadAll/Wait
  ordering all correct); TDS-04 finalized (DoD); pytest in CI; e2e keystone green.
EVIDENCE: module TRACEABILITY execution record + ticked VALIDATION_CHECKLIST + HANDOFF actuals.
NEXT: founder squash-merge; M12 proceeds stacked on m11-subprocess-runner

# ── M12 awaiting founder merge ────────────────────────────────────────────────
MILESTONE: M12-plugin-system
BRANCH: m12-plugin-system (stacked on m11-subprocess-runner)
PHASE: E-MERGE (blocked on founder)
GATE: none (non-gated boundary)
CARDS:
  M12-C1   DONE    awis-builder   5e6eebb
  M12-C2   DONE    awis-builder   58902ff
  M12-C3   DONE    awis-builder   72496b8
  M12-C4   DONE    awis-builder   c9566c6
  M12-V1   DONE    awis-verifier  (report 2026-07-10 at 147920c; PASS 27/27)
BLOCKERS: none
MERGE-RECOMMENDATION: APPROVED FOR SQUASH MERGE — D-CLOSE (Fable, 2026-07-10) incl. the
  Opus-designated adversarial FSM review (upward substitution): invariants 1-5 verified;
  frozen surfaces untouched; gates + pytest green at HEAD.
EVIDENCE: module TRACEABILITY + ticked VALIDATION_CHECKLIST + HANDOFF actuals.
NEXT: founder squash-merge; M13 proceeds stacked on m12-plugin-system

# ── M13 active ────────────────────────────────────────────────────────────────
MILESTONE: M13-git-context-plugin
BRANCH: m13-git-context-plugin (stacked on m12-plugin-system)
PHASE: B-BUILD
GATE: none
CARDS:
  M13-C1   DONE    awis-builder   76574ff
  M13-C2   DISPATCHED   awis-builder
  M13-V1   READY   awis-verifier
BLOCKERS: none
NEXT: M13-C2 (dispatch P1 → awis-builder)

LAST: 2026-07-10 ledger-repo reconciliation (Fable): STATE said M09 E-MERGE blocked, but main
      HEAD 03d5045 IS the founder's M09 squash-merge ("M09 — Test Infrastructure") and
      m10-yaml-dsl is based on it — founder merged after this branch's A-INIT snapshot.
      Resolution: M09 → DONE-MILESTONES (merge sha 03d5045). Same day: M10 D-CLOSE (Fable)
      PASS, merge recommendation APPROVED; founder directive: continue through M18 with
      Sonnet/Haiku subagents only (no Opus; Opus-designated work is done inline by Fable,
      upward substitution per EEOS rule 8); milestones stack branches, E-MERGE stays human.

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06 M07 M08 M09   # history: git log + module HANDOFFs
# M09 merge sha 03d5045 (founder squash-merge, verified against main log at reconciliation)
