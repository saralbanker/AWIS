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

# ── M13 awaiting founder merge ────────────────────────────────────────────────
MILESTONE: M13-git-context-plugin
BRANCH: m13-git-context-plugin (stacked on m12-plugin-system)
PHASE: E-MERGE (blocked on founder)
GATE: none (non-gated boundary)
CARDS:
  M13-C1   DONE    awis-builder   76574ff
  M13-C2   DONE    awis-builder   080a367
  M13-V1   DONE    awis-verifier  (report 2026-07-10 at d120ade; PASS all rows)
BLOCKERS: none
MERGE-RECOMMENDATION: APPROVED FOR SQUASH MERGE — D-CLOSE (Fable, 2026-07-10): consumer-only
  milestone; OIP plugin dependency exists; manifest Blueprint-verbatim (python3 delta recorded).
EVIDENCE: module TRACEABILITY + ticked VALIDATION_CHECKLIST + HANDOFF actuals.
NEXT: founder squash-merge; M14 proceeds stacked on m13-git-context-plugin

# ── M14 awaiting founder merge ────────────────────────────────────────────────
MILESTONE: M14-core-cli
BRANCH: m14-core-cli (stacked on m13-git-context-plugin)
PHASE: E-MERGE (blocked on founder)
GATE: none (non-gated boundary)
CARDS:
  M14-C1   DONE    awis-builder   709ce0e
  M14-C2   DONE    awis-builder   73435f7
  M14-C3   DONE    awis-builder   add54dd
  M14-C4   DONE    awis-builder   51c2575
  M14-V1   DONE    awis-verifier  (21/22 at 08982be; row 9 → C3r 234a935; CE re-verify green)
  M14-C3r  DONE    awis-builder   234a935
BLOCKERS: none
MERGE-RECOMMENDATION: APPROVED FOR SQUASH MERGE — D-CLOSE (Fable, 2026-07-10): F-3/CONTRA-5
  settled + proven live; dev loop (submit->status->trace->signal) real-binary tested incl.
  crash recovery; frozen layers untouched.
NEXT: founder squash-merge; M15 proceeds stacked on m14-core-cli

# ── M15 active ────────────────────────────────────────────────────────────────
MILESTONE: M15-oip-on-awis
BRANCH: m15-oip-on-awis (stacked on m14-core-cli)
PHASE: E-MERGE (blocked on founder: G3 VERDICT + TDS-06 SIGN-OFF — see G3_BRIEF.md)
GATE: G3 + TDS-06 sign-off (FOUNDER-ONLY)
CARDS:
  M15-P0   DONE    awis-builder   c3fba29   (disclosed platform seam)
  M15-C1   DONE    awis-builder   80ef917
  M15-C2   DONE    awis-builder   6731297   (P2 salvage OK)
  M15-C3   DONE    awis-builder   0ef33c8   (3 deviations adjudicated)
  M15-C3r  DONE    awis-builder   0e452f8   (fixture restored; P1/P2 seams disclosed)
  M15-V1   DONE    awis-verifier  (PASS all rows, 2026-07-11 at 9973a3f)
BLOCKERS: none
MERGE-RECOMMENDATION: APPROVED subject to G3 verdict + TDS-06 sign-off. Seams disclosed:
  P0 LoadWorkflowFile, P1 RegisterPlugin/ns-scoped Submit fallback/--namespace, P2 engine
  fallback-join sentinel (frozen fixture exposed AND-join defect; RB mechanism).
NEXT: founder G3 + merge; M16 proceeds stacked

# ── M16 active ────────────────────────────────────────────────────────────────
MILESTONE: M16-anthropic-adapter
BRANCH: m16-anthropic-adapter (stacked on m15-oip-on-awis)
PHASE: E-MERGE (blocked on founder)
GATE: none
CARDS:
  M16-C1   DONE    awis-builder   3b12689
  M16-V1   DONE    awis-verifier  (PASS all rows at 3b12689)
MERGE-RECOMMENDATION: APPROVED FOR SQUASH MERGE (D-CLOSE Fable 2026-07-11)
NEXT: founder merge; M17 proceeds stacked

# ── M17 active ────────────────────────────────────────────────────────────────
MILESTONE: M17-full-cli-init
BRANCH: m17-full-cli-init (stacked on m16-anthropic-adapter)
PHASE: B-BUILD (returned from C-VERIFY — V1 FAIL; revision cards needed, not yet cut)
CARDS:
  M17-C1   DONE    awis-builder   a342394
  M17-C2   DONE    awis-builder   744d3d0
  M17-C3   DONE    awis-builder   afe526d   (P2 salvage re-dispatch; see LAST-2)
  M17-V1   FAIL    awis-verifier  (2026-08-21, clean tree at ece8694; see LAST-2)
BLOCKERS: E2-candidate — see LAST-2 "frozen-surface scope violation"; needs CE/founder
  adjudication (CONTRA per EEOS rule 7) before any revision-card work resumes, since it
  concerns whether sdk/engine/plugin/storage changes already on this branch (introduced by
  C1/C2, not C3) were an approved, undisclosed deviation or genuine scope overreach.
NEXT: human/CE adjudication of the scope-freeze finding first; THEN cut revision card(s)
  (working title M17-C1r/C2r) for: (a) missing goldens for replay + orphaned
  config_show/config_set/config_validate/rebuild_state goldens, (b) missing TDS-07 sections
  in docs/CLI_CONTRACT.md for every M17 command except init, (c) whatever the adjudication
  above decides about the non-scope files. Re-run M17-V1 clean once revision cards close.

LAST-2: 2026-08-21 M17-V1 dispatched to awis-verifier (this session, on human request) on a
  clean checkout at ece8694 (HEAD after the M17-C3 ledger commit). VERDICT: FAIL. V-COMMON:
  build/test/race/e1 ✅; lint showed 9 errcheck hits on first run but they cited a path
  belonging to a different, concurrently-running agent worktree (shared golangci-lint cache
  cross-contamination) — reported as a flake per protocol, not silently re-run past; a cache
  clean + rerun gave 0 issues. Migration 0006 fresh+upgrade ✅, StoragePort untouched ✅. Cron
  fake-clock tests ✅. `awis init` scaffold/--force/rehearsal ✅ (rehearsal instance legitimately
  ends `failed` — no compiled-in handlers in the generic binary, matches the CE pin's "measured
  loosely" framing, not a defect). docs/CLI.md full-tree coverage ✅. TWO independently-
  sufficient FAIL rows: (1) most M17 commands lack goldens and/or a TDS-07 contract section —
  `replay` has no goldens at all ("Goldens not needed for replay" per its own test comment,
  contradicting the checklist), `config_show/config_set/config_validate/rebuild_state` goldens
  exist on disk but are orphaned (no test references them), and docs/CLI_CONTRACT.md §3 still
  labels the whole M17 tree "Planned Commands (not yet implemented)" — only `init` (added by
  C3 this session) has a §4 contract section. (2) **frozen-surface scope violation**: diff
  `m16-anthropic-adapter...HEAD` touches `internal/dsl/dsl.go`, `internal/engine/*_test.go`
  (4 files), `internal/plugin/{manager.go,runner.go,transport.go}` (production code) +
  `e2e_gitcontext_test.go`/`e2e_python_test.go`, `sdk/runtime.go`, `sdk/testing/mock.go`,
  `internal/storage/{audit.go,db_test.go,plugins_test.go,signal_test.go}` — this contradicts
  IMPLEMENTATION_SPEC.md's own Non-scope line ("Frozen surfaces: StoragePort method set,
  engine semantics ... core types — all untouched; sdk untouched") and the card-level
  acceptance criteria (M17-C1/C2 ACCEPTANCE: "engine/core/sdk untouched"). Traced by file
  timestamps/diff boundaries: **this predates M17-C3** — all of these files were already in
  the C1/C2 diff before this session's C3 dispatch; this session neither introduced nor
  touched them. Full table + evidence pointers: module TRACEABILITY.md V1 record (verbatim
  verifier report) and VALIDATION_CHECKLIST.md (ticked/unticked this session). No files edited
  by the verifier (read-only agent, per its tool contract) other than this ledger commit made
  afterward to record the result.

LAST: 2026-08-21 M17-C3 P2 salvage re-dispatch (this session): STATE showed M17-C3 DISPATCHED
      with no commit on m17-full-cli-init — a prior run died without finishing. Preserved WIP
      on branch `m17-c3-wip` (9 files: cmd/awis/init.go + init_test.go + scaffold/*) was
      explicitly marked "does NOT build green — do not merge"; kept as reference only, NOT
      merged, NOT deleted. Re-dispatched to awis-builder (isolated worktree) with the P2
      salvage preamble: audited the WIP against the card's OUTPUTS/ACCEPTANCE, salvaged what
      passed, fixed 4 defects found in it (go:embed missing `all:` prefix silently dropped
      `.gitignore`; TestEmbeddedWorkflowsByteIdentical source paths 2 levels too deep;
      no init goldens existed; no rehearsal system test existed) and closed 2 remaining gaps
      (docs/CLI.md not covering the full command tree; CLI_CONTRACT.md missing the TDS-07
      init section). Result committed once: afe526d. Independently re-verified in this
      session (not just the builder's self-report): `make verify` full green at afe526d
      (gofmt-check, vet, lint 0 issues, oip-isolation, build, test, race, e1); rehearsal test
      `TestSystemRehearsalInitStartSubmitTrace` confirmed running un-skipped (11.01s, real
      binary subprocess init→start→submit→status-poll→trace) and passing; go.mod/go.sum diff
      vs 49ce489 is empty. Full record: docs/05-implementation/M17-full-cli-init/TRACEABILITY.md
      and HANDOFF.md (both created this session — module previously had neither). Per this
      session's explicit scope: M17-V1 was NOT dispatched, M18 was NOT started, nothing was
      merged, EEOS/cards were not altered.

LAST-PRIOR: 2026-07-10 ledger-repo reconciliation (Fable): STATE said M09 E-MERGE blocked, but main
      HEAD 03d5045 IS the founder's M09 squash-merge ("M09 — Test Infrastructure") and
      m10-yaml-dsl is based on it — founder merged after this branch's A-INIT snapshot.
      Resolution: M09 → DONE-MILESTONES (merge sha 03d5045). Same day: M10 D-CLOSE (Fable)
      PASS, merge recommendation APPROVED; founder directive: continue through M18 with
      Sonnet/Haiku subagents only (no Opus; Opus-designated work is done inline by Fable,
      upward substitution per EEOS rule 8); milestones stack branches, E-MERGE stays human.

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06 M07 M08 M09   # history: git log + module HANDOFFs
# M09 merge sha 03d5045 (founder squash-merge, verified against main log at reconciliation)
