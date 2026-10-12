# EEOS STATE LEDGER — single source of live execution state
# Write rule (EEOS §3.3): no phase or card transition is real until it is written here.
# Status vocabulary: DRAFT | READY | DISPATCHED | DONE | STOPPED | WITHDRAWN

# ── M10 CLOSED (reconciled 2026-09-19) ─────────────────────────────────────────
MILESTONE: M10-yaml-dsl
BRANCH: m10-yaml-dsl
PHASE: CLOSED
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
CODE-IN-MAIN: yes — 0 commits ahead of main (m10-yaml-dsl is an ancestor of main); it
  reached main by branch stacking, not by a discrete founder squash-merge (see LAST-9
  reconciliation entry below).
NEXT: none — M10 is closed.

# ── M11 CLOSED (reconciled 2026-09-19) ─────────────────────────────────────────
MILESTONE: M11-subprocess-runner
BRANCH: m11-subprocess-runner (stacked on m10-yaml-dsl)
PHASE: CLOSED
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
CODE-IN-MAIN: yes — 0 commits ahead of main (m11-subprocess-runner is an ancestor of main);
  it reached main by branch stacking, not by a discrete founder squash-merge (see LAST-9
  reconciliation entry below).
NEXT: none — M11 is closed.

# ── M12 CLOSED (reconciled 2026-09-19) ─────────────────────────────────────────
MILESTONE: M12-plugin-system
BRANCH: m12-plugin-system (stacked on m11-subprocess-runner)
PHASE: CLOSED
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
CODE-IN-MAIN: yes — 0 commits ahead of main (m12-plugin-system is an ancestor of main); it
  reached main by branch stacking, not by a discrete founder squash-merge (see LAST-9
  reconciliation entry below).
NEXT: none — M12 is closed.

# ── M13 CLOSED (reconciled 2026-09-19) ─────────────────────────────────────────
MILESTONE: M13-git-context-plugin
BRANCH: m13-git-context-plugin (stacked on m12-plugin-system)
PHASE: CLOSED
GATE: none (non-gated boundary)
CARDS:
  M13-C1   DONE    awis-builder   76574ff
  M13-C2   DONE    awis-builder   080a367
  M13-V1   DONE    awis-verifier  (report 2026-07-10 at d120ade; PASS all rows)
BLOCKERS: none
MERGE-RECOMMENDATION: APPROVED FOR SQUASH MERGE — D-CLOSE (Fable, 2026-07-10): consumer-only
  milestone; OIP plugin dependency exists; manifest Blueprint-verbatim (python3 delta recorded).
EVIDENCE: module TRACEABILITY + ticked VALIDATION_CHECKLIST + HANDOFF actuals.
CODE-IN-MAIN: yes — 0 commits ahead of main (m13-git-context-plugin is an ancestor of main);
  it reached main by branch stacking, not by a discrete founder squash-merge (see LAST-9
  reconciliation entry below).
NEXT: none — M13 is closed.

# ── M14 CLOSED (reconciled 2026-09-19) ─────────────────────────────────────────
MILESTONE: M14-core-cli
BRANCH: m14-core-cli (stacked on m13-git-context-plugin)
PHASE: CLOSED
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
CODE-IN-MAIN: yes — 0 commits ahead of main (m14-core-cli is an ancestor of main; tagged
  milestone/M14); it reached main by branch stacking, not by a discrete founder
  squash-merge (see LAST-9 reconciliation entry below).
NEXT: none — M14 is closed.

# ── M15 CLOSED (reconciled 2026-09-19) ─────────────────────────────────────────
MILESTONE: M15-oip-on-awis
BRANCH: m15-oip-on-awis (stacked on m14-core-cli)
PHASE: CLOSED
GATE: G3 COMPLETE — Question 1 (platform boundary/QG-4) PASS, founder ruling 2026-09-18;
  Question 2 (TDS-06 sign-off) PASS, founder ruling 2026-09-19. See G3_BRIEF.md.
CARDS:
  M15-P0   DONE    awis-builder   c3fba29   (disclosed platform seam)
  M15-C1   DONE    awis-builder   80ef917
  M15-C2   DONE    awis-builder   6731297   (P2 salvage OK)
  M15-C3   DONE    awis-builder   0ef33c8   (3 deviations adjudicated)
  M15-C3r  DONE    awis-builder   0e452f8   (fixture restored; P1/P2 seams disclosed)
  M15-V1   DONE    awis-verifier  (PASS all rows, 2026-07-11 at 9973a3f)
BLOCKERS: none
MERGE-RECOMMENDATION: APPROVED subject to G3 verdict + TDS-06 sign-off (both now granted).
  Seams disclosed: P0 LoadWorkflowFile, P1 RegisterPlugin/ns-scoped Submit fallback/
  --namespace, P2 engine fallback-join sentinel (frozen fixture exposed AND-join defect;
  RB mechanism).
CODE-IN-MAIN: yes — reached main by branch stacking (m15-oip-on-awis sits beneath M16's
  A-INIT commit), not by a gated merge or squash commit. No commit ever represented a
  gate decision for this milestone; both G3 gates were performed retroactively (see
  LAST-9 reconciliation entry below).
NEXT: none — M15 is closed.

# ── M16 CLOSED (reconciled 2026-09-19) ─────────────────────────────────────────
MILESTONE: M16-anthropic-adapter
BRANCH: m16-anthropic-adapter (stacked on m15-oip-on-awis)
PHASE: CLOSED
GATE: none
CARDS:
  M16-C1   DONE    awis-builder   3b12689
  M16-V1   DONE    awis-verifier  (PASS all rows at 3b12689)
MERGE-RECOMMENDATION: APPROVED FOR SQUASH MERGE (D-CLOSE Fable 2026-07-11)
CODE-IN-MAIN: yes — reached main by branch stacking (m16-anthropic-adapter sits beneath
  M17's A-INIT commit), not by a gated merge or squash commit. V1 passed. No merge commit
  ever represented a gate decision for this milestone (see LAST-9 reconciliation entry
  below).
NEXT: none — M16 is closed.

# ── M17 CLOSED (reconciled 2026-09-19) ─────────────────────────────────────────
MILESTONE: M17-full-cli-init
BRANCH: m17-full-cli-init (stacked on m16-anthropic-adapter)
PHASE: CLOSED
GATE: none
CARDS:
  M17-C1   DONE        awis-builder   a342394
  M17-C2   DONE        awis-builder   744d3d0
  M17-C3   DONE        awis-builder   afe526d   (P2 salvage re-dispatch; see LAST-2)
  M17-V1   FAIL        awis-verifier  (2026-08-21, clean tree at ece8694; see LAST-2, LAST-3)
  M17-C1r  DONE        awis-scribe    ead48d5   (see LAST-5)
  M17-V1   FAIL        awis-verifier  (2026-08-21, re-run, isolated worktree at ef626e7; see LAST-6)
  M17-C2r  DONE        awis-builder   4808070   (see LAST-8)
BLOCKERS: none — all eight VALIDATION_CHECKLIST rows are now dispositioned (see LAST-9):
  rows 2/4/5/6 PASS (LAST-6); row 3 PASS, re-measured; row 8 WAIVED per DEC-2; row 1
  CLOSED WITH RISK ACCEPTED per DEC-5; row 7 MEASURED WITH EXCEPTIONS per DEC-4.
CODE-IN-MAIN: yes — reached main by branch stacking (m17-full-cli-init sits beneath a
  B-2/B-3 hardening commit), not by a gated merge or squash commit. M17-V1 was recorded
  FAIL twice (LAST-2, LAST-6) before all eight checklist rows were dispositioned; no
  merge commit ever represented a D-CLOSE decision for this milestone.
NEXT: none — M17 is closed.

LAST-9: 2026-09-19 Beta Baseline reconciliation (defect D-5, beta-baseline program): this
  ledger had not been updated since 2026-08-21 and had been contradicted by the repository
  ever since. Correction: M15, M16, and M17 code is ALL already in main. It arrived by
  branch stacking, not by gated merges — each milestone's branch sits beneath the next
  one's A-INIT commit (m15 -> "M16 A-INIT", m16 -> "M17 A-INIT", m17 -> a B-2/B-3
  hardening commit). No merge or squash commit ever represented a gate decision for any
  of the three. Both of M15's founder gates have now been performed retroactively: G3
  Question 1 (platform boundary/QG-4) PASS 2026-09-18; G3 Question 2 (TDS-06 sign-off)
  PASS 2026-09-19 (see G3_BRIEF.md and apps/oip/docs/RECORD_FORMAT.md). M15 CLOSED. M16's
  code is in main and its V1 passed. M16 CLOSED. M17's eight VALIDATION_CHECKLIST rows
  are now all dispositioned: row 3 PASS, re-measured; row 8 WAIVED per DEC-2; row 1
  CLOSED WITH RISK ACCEPTED per DEC-5; row 7 MEASURED WITH EXCEPTIONS per DEC-4 (rows
  2/4/5/6 already PASS per LAST-6). Disposition record:
  docs/13-beta-baseline/CURRENT_TRUTH_LEDGER.md (D-3, DEC-2, DEC-4, DEC-5). M17 CLOSED.
  M15, M16, and M17 moved to DONE-MILESTONES below. This entry records that the
  reconciliation happened and that the gates were performed retroactively; it does not
  editorialize on how the gap arose.

  EXTENSION (same date, same reconciliation): this ledger also still recorded M10, M11,
  M12, M13, and M14 with phase "blocked on founder" merge. Verified git facts: each of
  m10-yaml-dsl, m11-subprocess-runner, m12-plugin-system, m13-git-context-plugin, and
  m14-core-cli has 0 commits ahead of main (each is an ancestor of main); milestone/M14
  is tagged. Unlike M15, none of these five carries a GATE — each was recorded
  "GATE: none (non-gated boundary; squash-merge on founder review)" — so there was no
  gate that could be performed retroactively; the only outstanding step recorded against
  each was the founder squash-merge itself, which never happened as a discrete act. All
  five reached main the same way M15-M17 did: by branch stacking, each milestone's branch
  sitting beneath the next milestone's work, so no merge commit ever represented a merge
  decision for any of them. M10, M11, M12, M13, and M14 are now CLOSED and moved to
  DONE-MILESTONES below on that basis.

LAST-8: 2026-08-21 (same session) — M17-C2r executed by awis-builder, HEAD 4808070, single
  commit "M17-C2r: race-scaled rehearsal deadline (row 1) and replay.txt golden (row 3)".
  Touch set exactly as scoped: cmd/awis/{c1r_test.go,racescale_norace_test.go,
  racescale_race_test.go,replay.go,system_test.go,testdata/golden/replay.txt} (6 files,
  +67/−19) — independently confirmed via `git show --stat 4808070`, matches card's Required
  Files list plus the two permitted raceScale build-tag helper files. Row 1: rehearsal-test
  deadline now `10s * raceScale` (raceScale=6 under `-race` build tag, 1 otherwise) instead of
  a hardcoded 10s; agent reports `go test -race -count=3 ./cmd/awis/...` 3/3 pass (elapsed
  11.58s/11.18s/11.51s per run) and one transient pre-commit full-suite `make race` fail on
  first cold -race binary build, reproduced clean on 2 subsequent full runs — not independently
  re-run this session. Row 3: `replay.txt` added, `printReplayHuman` extracted from `runReplay`
  byte-identical (agent states diff shows only the extraction), `TestReplayGoldenHuman` added
  sharing a fixture helper with the existing JSON test; agent states it passes clean without
  `-update`. Frozen-surface check: `git diff m16-anthropic-adapter...4808070 --stat` independently
  re-run this session — confirms ZERO new frozen-surface paths beyond what LAST-6 already
  enumerated (same internal/storage, internal/plugin, internal/engine, sdk entries; commit
  4808070 itself touches only the 6 cmd/awis files listed above). Row 8 untouched, as required.
  Card compliance (files, scope, forbidden-list) independently confirmed via diff inspection.
  Test/race NUMBERS (pass/fail, timings) are per the builder's own report and have NOT been
  independently re-executed or re-verified this session — that is M17-V1's job, not CE's.

LAST-6: 2026-08-21 (new session) — M17-V1 re-dispatched fresh (independent awis-verifier agent,
  isolated git worktree, no context from the C1r implementation session) per explicit human
  instruction to re-measure everything from scratch, trusting no prior ledger claim. VERDICT:
  FAIL, on a clean checkout of m17-full-cli-init at ef626e7. Findings vs the 8-row
  VALIDATION_CHECKLIST:
  - Row 1 (V-COMMON+pytest): build/test/lint(0 issues, no flake this run)/e1/pytest(41/41) all
    ✅; **race ❌** — `TestSystemRehearsalInitStartSubmitTrace` failed under full-suite
    `go test -race ./...` ("did not reach a terminal status within 10s"), 20.69s, but passed
    standalone in 11.06s — a timing-marginal flake against the test's hardcoded 10s deadline
    under -race contention, not a logic defect.
  - Row 2 (migration 0006 + StoragePort): ✅.
  - Row 3 (goldens+TDS-07): narrowed but still **FAIL** — all 15 TDS-07 §4 sections now present
    (C1r closed that half); but `replay` has a JSON golden+test only, no `.txt` golden/test,
    while every other M17 command has both. C1r's own card scoped only "replay.json" (see
    M17-C1r.md OUTPUTS 1a) — this gap was in the card's own scope definition, not a C1r defect.
  - Row 4 (cron fake-clock): ✅. Row 5 (init scaffold+rehearsal): ✅ standalone (the race
    failure above is the same test, but as a suite-level flake, not a functional defect).
    Row 6 (docs/CLI.md tree): ✅. Row 7 (PRD §32 spot-check): ✅, still non-exhaustive by design.
  - Row 8 (scope/frozen-surface): **FAIL**, re-flagged. The verifier did not treat LAST-3's
    trace-and-close as adjudication evidence ("the prior ledger's 'no adjudication needed'
    framing is the implementer's own adjudication and was not accepted as evidence") and
    re-measured the same diffs directly: `internal/storage/audit.go` (+7, new additive
    `AuditAppender` interface), `internal/storage/{db_test.go,plugins_test.go,signal_test.go}`
    (schema-version 5→6 bumps), `internal/engine/*_test.go`(4)/`internal/plugin/{manager.go,
    runner.go,transport.go}`+2 e2e tests/`sdk/runtime.go`/`sdk/testing/mock.go` all still show
    in `git diff --stat` vs `origin/m16-anthropic-adapter`. Confirmed (again) as whitespace/
    gofmt-only for the non-storage files and necessary-consequence-of-0006 for the storage test
    bumps, consistent with LAST-3's trace — but the checklist's literal wording ("untouched",
    "no existing test modified") is not met, so it stands as FAIL pending an actual
    CE/founder adjudication rather than a self-closed session note.
  Full verbatim report + repro commands: TRACEABILITY.md V1-rerun record. No STATE/card edits
  made by the verifier itself (read-only agent; confirmed zero commits/diff in its worktree).
  Per this session's explicit scope (verify only): no revision card cut, no adjudication
  performed, no remediation implemented — see NEXT.

LAST-5: 2026-08-21 (same session) — M17-C1r report received and independently spot-checked
  (not a full re-verification, per this session's explicit "do not verify beyond what is
  required" instruction): `git show --stat ead48d5` confirms the 4-file diff (c1r_test.go new,
  replay_test.go comment, testdata/golden/replay.json new, CLI_CONTRACT.md +644/-18) matches
  the scribe's self-report; `git diff m16-anthropic-adapter...HEAD --stat` for frozen-surface
  paths shows nothing new beyond the pre-existing entries already traced and closed at LAST-3;
  `go test -count=1 ./cmd/awis/...` re-run directly, uncached — `ok`; grep-confirmed all 15
  new §4 sections and all 7 previously-orphaned goldens (replay + config_show/set/validate +
  rebuild_state + plugin_remove/status) are now referenced by `checkGolden(...)` calls. Full
  record: module TRACEABILITY.md C1r entry. `make pytest`/lint/race/e1 taken on the scribe's
  report, not independently re-run. M17-V1 re-run intentionally NOT done this session.

LAST-4: 2026-08-21 (new session) — cut docs/05-implementation/M17-full-cli-init/cards/M17-C1r.md
  per the NEXT direction left by LAST-2/LAST-3 (goldens/TDS-07 gap only, row 7; row 8 stays
  CLOSED, not reopened). DISPATCH: awis-scribe (mechanical CLI/doc/golden-fixture batch, IKB
  fit). Scope confirmed against current repo state before cutting: replay has zero goldens;
  config_show/config_set/config_validate/rebuild_state/plugin_remove/plugin_status goldens
  exist on testdata/golden/ but are referenced by no test (verified via grep across
  cmd/awis/*_test.go); docs/CLI_CONTRACT.md §4 has sections for version/start/stop/status/
  submit/signal/cancel/trace/workflow {validate,list,show}/plugin {install,list}/init only —
  15 M17 commands still lack a §4 section. Dispatched per EEOS P1; STATE written before
  dispatch (ledger law, rule 1).

LAST-3: 2026-08-21 (same session) — CLOSED the LAST-2 frozen-surface finding without
  founder/CE adjudication, on evidence rather than judgment call. Traced every cited file to
  its introducing commit: `sdk/runtime.go`, `sdk/testing/mock.go`,
  `internal/plugin/{manager,runner,transport}.go`, `internal/dsl/dsl.go`, and 4
  `internal/engine/*_test.go` files all trace to `be13cf9` (prior-session "STABILIZATION
  S2/S3/S5/S6" commit — a disclosed, self-documenting repo-wide `gofmt -w .` pass). Verified
  directly with `git diff --ignore-all-space be13cf9^ be13cf9 -- <file>` and manual inspection
  of the two files that still showed a diff under `-w` (`runner.go`, `transport.go`): all of
  it is struct-field realignment and Go 1.19 doc-comment-block reflow, zero identifiers/
  values/logic changed. The remaining `internal/storage/{db_test.go,plugins_test.go,
  signal_test.go}` edits (from C1, a342394) are hardcoded `schema_version 5→6` assertion
  bumps — an unavoidable, minimal consequence of adding migration 0006, C1's own explicit
  deliverable; `internal/storage/audit.go` is additive, in scope of C1's RecallStore/audit
  read-path objective. Conclusion: V1's row-8 FAIL is factually correct at the byte-diff
  level (per the verifier's mandate to report fact, not intent) but does not represent the
  scope overreach the checklist item exists to catch — no behavioral change to any frozen
  surface. Downgraded from "BLOCKERS: E2-candidate, needs adjudication" to closed; the only
  real open item from V1 is the goldens/TDS-07 gap (row 7), which is unrelated and still
  needs a revision card (see NEXT). Full trace: TRACEABILITY.md.

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

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06 M07 M08 M09 M10 M11 M12 M13 M14 M15 M16 M17 M18   # history: git log + module HANDOFFs
# M09 merge sha 03d5045 (founder squash-merge, verified against main log at reconciliation)
# M10-M17 all reached main by branch stacking, not a discrete founder squash-merge; no
# commit represents a merge or gate decision for any of these eight. M15's two founder
# gates were performed retroactively; the other seven carried no gate. See LAST-9
# (2026-09-19) and each milestone's own block above.

# ── M18 A-INIT (2026-10-10) ─────────────────────────────────────────────────────
MILESTONE: M18-hardening-release
BRANCH: m18-hardening-release (branched from main at 4977de7)
PHASE: CLOSED (2026-10-12) — v1.0.0 released
GATE: G4 — Release (fires at D-CLOSE/E-MERGE; no card tags v1.0.0 or renders ship/no-ship)
CARDS:
  M18-C1   DONE    awis-builder(haiku)  16cf735  (P2 salvage re-dispatch after rate-limit death of run 1)
  M18-C2   WITHDRAWN awis-builder(haiku) (STOPPED E1: stale QG-1 premise — superseded by C2r)
  M18-C2r  DONE    awis-builder(haiku)  f895a12  (P-08 max 39.0ms PASS; P-09 max 5.7ms PASS; QG-1 linux 248ms completed)
  M18-C3   DONE    awis-builder(haiku)  e15497b  (goreleaser v2.17.0 pinned; 4 static archives + sha256 OK)
  M18-C3r  DONE    awis-builder(haiku)  99f7b13  (clone v1.0.0 release: `awis version 1.0.0`; 96-entry notes)
  M18-C3r2 DONE    awis-builder(haiku)  34894e7  (verify/release job split; persist-credentials:false; no release cache; actionlint clean)
  M18-C4   DONE    awis-builder(haiku)  54fcf15  (P-01 & P-05 fixed fix-only; 9/9 NFR PASS 3/3 runs; §32 53/2/1/3)
  M18-C4r  DONE    awis-builder(haiku)  3addeb9  (trace adapter/tokens suffix; native bytes unchanged; P-05 max 469.9ms PASS)
  M18-V1   DONE(FAIL) awis-verifier(haiku)  @4092406 (report in TRACEABILITY.md; no commit — identity forbids doc edits)
  M18-C5   DONE    awis-builder(haiku)  29cb521  (crash test deterministic: loaded 30/30 vs control 26/30; records fixed)
  M18-V1r  DONE(PASS) awis-verifier(haiku)  @b78ae44 (report in TRACEABILITY.md; checklist ticked by CE)
BLOCKERS: none
NEXT: FOUNDER — Loop A (bounded engine confidence) may start from tag v1.0.0; M18-B5 is an explicit Loop A input. Not started.
RELEASE-EVIDENCE (E-MERGE executed by the CE at the founder's explicit instruction, 2026-10-12):
  Founder G4 verdict "SHIP v1.0.0 with the exceptions above" recorded at dc503ff (M18 TRACEABILITY.md "G4 rulings").
  PR #1 squash-merged -> main d6fe341 (tree identical to dc503ff); PR CI at dc503ff green: runs 38221217034 (push), 38221224265 (pull_request).
  main CI at d6fe341: run 38221778716 success (Verify ubuntu, Verify macos, EEOS docs-lint).
  Tag v1.0.0 (annotated, object 9f92841) -> d6fe341. Release workflow run 38222249998 success (make verify; goreleaser v2.17.0).
  Release https://github.com/saralbanker/AWIS/releases/tag/v1.0.0 (not draft): 4 archives (darwin/linux x amd64/arm64) + checksums.txt; sha256sum -c OK x4; linux_amd64 `awis version` -> "awis version 1.0.0  go go1.26.8".
  Release notes disclose the G4 exceptions (QG-2 waived/NOT PASS; QG-4 limited + M18-B6; row 41 PARTIAL, --json --full; NFR-P-03 and rows 5/28/32 UNMEASURABLE; rows 37/40 NOT SATISFIED, 44 PARTIAL; M18-B5; rows 13/14 by amendment).
G4-RULINGS (CE-recorded 2026-10-12; ratified by founder G4 verdict SHIP 2026-10-12; M18 TRACEABILITY.md):
  QG-2 WAIVED for v1.0.0 (unmeasured; risk accepted; NOT PASS). QG-4 ACCEPTED, not literal PASS (G4 extends G3 Q1's
  P0/P1/P2 seams to the disclosed `--namespace` CLI seam in cmd/awis/{main,errors,submit}.go). PRD §32
  rows 13/14: TDS-07 governs; PRD wording corrected (AWIS_PRD.md), implementation unchanged →
  SATISFIED BY AMENDMENT. Row 41: PARTIAL accepted (outputs via --json --full). NFR-P-03 stays UNMEASURABLE. Prior
  exceptions (37/40/44; 5/28/32) carried forward. §32: 52 SAT (2 by amendment) / 2 NOT / 2 PARTIAL / 3 UNMEAS — founder
  exception to PRD §32 "all Must Have criteria must pass".
  Pre-ruling CI at PR head 883925a green: push 38156031710 + pull_request 38156060642.
D-CLOSE (Opus CE in Fable's role per MODEL-DIRECTIVE, 2026-10-11): semantic review PASS, no
  architectural drift (TRACEABILITY.md). V1r PASS @b78ae44; CI 38155115251 green both legs.
BACKLOG (non-blocking): M18-B1 release-note bookkeeping noise; M18-B2 one unreproduced
  SQLITE_BUSY on first `awis submit` after `awis start` (CE repro 0/50); M18-B3 QG-2
  human-subject measurement post-release (G4 waiver, founder verdict 2026-10-12); M18-B4 `awis trace`
  human view lacks step outputs + CLI_CONTRACT `--full` inconsistency (row 41 PARTIAL accepted;
  clarify/fix post-release); M18-B5 `TestDeliverSignal_CrashPreCommit` (C5) no longer crashes
  inside the B3 transaction — add mid-transaction fault injection (Loop A); M18-B6 global
  `--namespace` ignored by signal/cancel/status/trace despite TDS-07 §4 (CLI_CONTRACT.md:157-162).
FOLLOW-UP ACCEPTANCE (M18, post-v1.0.0; IDs as in BACKLOG above):
  M18-B3 QG-2 — one developer who has not read AWIS docs reads `awis trace` for a completed and a
    failed instance and correctly states step order, per-step status and the failing step;
    result recorded as the QG-2 measurement (PASS/FAIL) in this ledger. Until then QG-2 = WAIVED, NOT PASS.
  M18-B4 row 41 — founder picks the human-view format for step outputs; CLI_CONTRACT.md `--full`
    text, the TDS-07 §4 sample and cmd/awis/trace.go agree; a golden test covers outputs in the
    human view; PRD §32 row 41 re-measured.
  M18-B5 crash coverage — a test kills the process while the DeliverSignal (B3) transaction is open,
    between its writes, and asserts after reopen: delivered_at=NULL and no SignalReceived event;
    the test is shown to FAIL against a deliberately non-atomic variant.
  M18-B6 `--namespace` — signal/cancel/status/trace honour the global flag with one test per
    command, OR TDS-07 §4 is amended through CONTRA (founder); until then the flag is not
    TDS-07-compliant for those four commands.
  NFR-P-03 — stays UNMEASURABLE (DEC-8/U-9: no Anthropic credential); measure when one exists.
  PRD §32 rows 37/44 (D-13, DEC-11), 40 (D-14, DEC-11), 5/28/32 (DEC-4/DEC-8): unchanged exceptions.
CI-EVIDENCE-2: run 38154745869 @1043b9a (contains C5 29cb521) — ubuntu ✓, macos ✓, docs-lint ✓.
V1-RESULT (2026-10-11, @4092406): FAIL. Reproduced independently: all 9 NFR rows (3 runs, verdicts
  MATCH), QG-1 (linux 249ms completed; macOS real-CI cmd/awis ok 42.292s), QG-5, make verify/
  contract/e1/release-dry, 4 static archives, notes→`awis version 1.0.0`, no v* tag, frozen-
  surface diff clean (46 files, no violation). FAILs: (1) CI 38152488419 macOS red —
  TestDeliverSignal_CrashPreCommit (pre-existing M07 test race; same 5/30 on main 4977de7 under
  load) → E1/Race skipped on macOS (QG-3 macOS unevidenced); (3) PRD §32 row 14 + STOP row 13:
  PRD text vs frozen TDS-07 (scored SAT at M17); (4) card's literal `sha256sum -c
  dist/checksums.txt` must run inside dist/ (card wording defect; `cd dist && sha256sum -c` OK
  x4); (5) card omits AWIS-E1 by name (E1 runs via make verify/e1 — card defect). STOPs: QG-2
  (founder), QG-4 (verifier declines to adopt G3 ruling — founder confirms at G4).
CE-CLASSIFICATION: FAIL-1/2 = release-gate failure (CI must be green) → C5 test-only fix (child
  pauses at documented crash-point 1; assertions unchanged; not a product defect — 13⇔13
  invariant held). FAIL-3/STOP-3 = authority conflict (PRD vs TDS-07) → FOUNDER DECISION at G4,
  recorded via C5 doc fix. FAIL-4/5 = card text defects, product behavior verified — no action
  beyond record. X-3 (V1 benches on tmpfs vs C4 on NVMe) — verdicts identical, recorded.
C4r-NOTE: run 1 STOPPED on model policy (agent had not seen MODEL-DIRECTIVE; no edits) →
  re-dispatched with directive cited → DONE 3addeb9.
CE-RULING row 41 (2026-10-11): PARTIAL — timing/adapter/tokens satisfied; "outputs" not in the
  human view; CLI_CONTRACT internally inconsistent (--full text says outputs truncated to 120
  chars; TDS-07 §4 sample shows none); outputs available via --json. Picking a human format is
  design, not fix-only → FOUNDER DECISION at G4. §32 totals now 52 SAT / 2 NOT (37,40) /
  2 PARTIAL (41,44) / 3 UNMEASURABLE (5,28,32). Recorded in docs/V1_BENCHMARKS.md.
C4-REVIEW (CE): diff reviewed — stmt cache (fixed SQL only, MaxOpenConns=1 pool, closed by
  DB.Close, ad-hoc fallback) + buffered trace writer (goldens pin bytes) are fix-only; no
  engine/core/sdk/StoragePort-interface change. CE independent P-01 re-run x5 (idle host):
  P99 4.549/4.605/4.582/4.653/4.636ms → PASS 5/5 (margin ~7%, recorded as thin).
CE-RULINGS on C4 UNRESOLVED: (a) QG-4 = PASS by prior founder authority — G3 Q1 PASS
  2026-09-18 (M15 cards/G3_BRIEF.md:126) adjudicated exactly these disclosed seams P0/P1/P2
  ("zero UNDISCLOSED changes"); M18 does not re-litigate. (b) PRD §32 "trace shows ... adapter,
  tokens" (C4's row 41) is a contractual Must-Have mis-scored SATISFIED at M17 → fix authorized
  → cards/M18-C4r.md. (c) macOS for C4 commits: push after C4r. (d) P-01 margin: see above.
C3r2-REVIEW (CE): accepted. Residual (accepted, not a defect): release-job checkout receives the
  job token as its fetch credential (first-party action, not persisted); verify-job module cache
  shared with ci.yml (gate-only job, builds nothing published).
C3r-REVIEW (CE): R1/R2/R3 evidenced (throwaway clone, push URL nonexistent, token unset; real
  repo `git tag -l 'v*'` empty). CE ruling during run: clone remote = real fetch URL +
  nonexistent push URL (goreleaser cannot run without a remote URL).
  SECURITY (background commit review of 99f7b13, CE-confirmed): release.yml checkout persists
  the contents:write GITHUB_TOKEN into .git/config, readable by `make verify`; setup-go cache in
  publishing job → C3r2 (blocks any v* tag push until fixed).
BACKLOG (non-blocking, M18-B1): release notes include 49/96 milestone-prefixed bookkeeping
  subjects (HANDOFF actuals/ledger/D-CLOSE); RE2 has no lookahead so `^M[0-9]{2}` cannot exclude
  them. Still "generated from milestone titles" (IMP §21); founder may curate the GitHub release
  text at publication. Not a gate failure.
CI-EVIDENCE: run 38146475809 @ bbdcfa5 (C1+C2r on branch) — ubuntu-latest ✓, macos-latest ✓,
  docs-lint ✓. cmd/awis ok on macOS (36.9s) incl. TestQG1RehearsalCompletesHelloWorld (no
  skip/GOOS gate) → QG-1 macOS leg real-CI (completed + ≤5min asserted).
C3-REVIEW (CE): accepted; awis-server excluded (PRD: no server mode in V1) — accepted. Three
  release blockers found → C3r: (R1) buildinfo.Version const → v1.0.0 binary would print
  0.1.0-dev; (R2) no `v*` tag workflow (IMP §21); (R3) snapshot skips changelog, release notes
  never produced. ENV: host /tmp quota exhausted by pre-existing /tmp/pytest-of-virus (not
  touched; agents use TMPDIR on /mnt/data).
C2r-REVIEW (CE): diff in scope (5 files, tests/bench/docs + Makefile BENCH_EXPECT only); CE re-ran
  TestQG1RehearsalCompletesHelloWorld independently → PASS, 249ms, status completed. P-08 timed
  interval (first Manager.Call on fresh Manager = spawn+handshake+reject RPC, upper bound) accepted.
CE-RULING C2-E1 (2026-10-11): C2 STOP verified — shipped `awis` registers examples.hello.*
  (B-9, cmd/awis/start.go:179); M17 TRACEABILITY 'ends failed' claim is stale. QG-1 runs on the
  shipped binary (option A; acceptance unchanged: completed, ≤5min). NFR-P-09 binding method =
  successful git.diff.fetch on tiny repo (conservative upper bound); immediate-reject path
  informational. Revision card cards/M18-C2r.md.
C1-RESULTS (make bench, 16cf735, Ryzen 7 7730U/linux, on-disk WAL): P-01 P99 9.998ms MISS (<5ms);
  P-02 P95 1.326ms PASS (gate 50ms, target 10ms); P-04 max 99.5ms PASS; P-05 multi-instance
  max 2.15ms PASS, single-instance (100K events in one instance) max 568.7ms MISS (<500ms);
  P-06 max 466.6ms PASS; P-07 max 101.6ms PASS.
CE-RULINGS (Opus CE, 2026-10-11, on C1 UNRESOLVED a/b):
  (a) NFR-P-01 is a genuine MISS, not a measurement artifact: CE re-ran with the same SQLite
      engine on ":memory:" (zero disk I/O) → P99 6.70/6.69/6.82ms, still MISS. CPU profile:
      ~37% of tick in sqlite3Prepare (GetInstance re-parses its SQL every call; no statement
      reuse in internal/storage). Routed to C4 as a fix-only item (statement reuse inside
      SQLiteStorage; no StoragePort interface change; no engine-semantics change). Fixture
      (100 instances) stays as committed — not shrunk to pass.
  (b) NFR-P-05: BOTH shapes are binding (strictest reading of "100K events"; the passing
      shape is not selected to avoid the miss). Single-instance miss routed to C4 fix-only:
      profile shows unbuffered per-event stdout writes + per-event JSON decode in
      cmd/awis trace rendering.
MODEL-DIRECTIVE (founder, 2026-10-11 session): orchestration by Opus 5.5 (CE role); ALL card
  execution by Haiku 5.5 only — no Sonnet. This is a founder override of EEOS rule 8
  (upward-only substitution) and of CLAUDE.md's frozen Sonnet/no-Opus allocation, recorded
  here per ledger law; cards' DISPATCH agent identities are unchanged, only the model.
ENTRY-GATE EVIDENCE: this A-INIT's immediate predecessor action (same session) fixed and
  verified a real, reproduced macOS CI failure on main — TestResolvePluginUser_UnknownNumericUID
  (internal/plugin/isolation_test.go), caused by a test assuming uid 4294967294 has no passwd
  entry on every platform; macOS Directory Services maps it to the built-in "nobody" account.
  Fix: made the test probe for an actually-unassigned uid at runtime instead of hardcoding one
  (internal/plugin/isolation.go, the production code, was untouched — correct logic, bad test
  assumption). Committed 4977de7, pushed, confirmed GREEN on real GitHub Actions run 38041177134
  (ubuntu-latest + macos-latest + EEOS docs-lint, all ✓) — not a local simulation. This resolves
  CURRENT_TRUTH_LEDGER.md §7 Limitation 1 ("CI has never actually run... macos-latest has never
  been exercised") for this specific commit.
NOT IN SCOPE FOR THIS A-INIT: M18 exit work (B-BUILD dispatch, benchmarking, release tooling)
  was explicitly deferred past entry per this session's task scope — cards are READY, none
  dispatched.
AMENDMENTS APPLIED: F-6 (week-6 schedule load) is moot — M16/M17 have been closed for months;
  no float action needed.
SOURCES COMPILED: IMP §27.M18, §20-21, §23 (G4), §25 (CONTRA-2, F-6), §26, §28; IKB §4 row 18;
  PRD §31 (QG-1..5, NFR-P-01..09), §32 (Must-Have acceptance); CURRENT_TRUTH_LEDGER.md (M17
  DEC-4 disposition, open non-blocking defects D-13/D-16/D-18/D-19, NFR-P-03 credential gap).
NEXT (after B-BUILD/C-VERIFY): D-CLOSE semantic review (Fable) → G4 human verdict → `v1.0.0`
  tag → E-MERGE.

# ── LOOP A — Engine correctness under failure (opened 2026-10-12) ─────────────────
LOOP: A (bounded engine confidence; founder-authorized 2026-10-12 after v1.0.0)
BRANCH: loop-a-engine-confidence (from tag v1.0.0 = d6fe341; v1.0.0 tag untouched)
PHASE: FOUNDER REVIEW (PR open; NOT merged; one founder decision open — LA-D1 below)
MODEL: Opus 5.5 CE (orchestration, rulings, review) + Haiku 5.5 executors (all code/tests), per the
  founder MODEL-DIRECTIVE recorded in the M18 section.
CARDS (sha on loop-a-engine-confidence):
  LA-B5        a98e010  mid-transaction crash injection for DeliverSignal (M18-B5)
  LA-P1-REPRO  bd6f58a  failing reproducers L1-L6 (stranded StepStarted)
  LA-P2-REPRO  8ea1bdd  failing reproducers S1-S4 (shutdown / cancel with in-flight work)
  LA-C1        417a6a2  fix L1-L6 (in-process RC-1 re-arm; partial-dispatch; signal-step recovery)
  LA-P3-BUSY   271325c  env-gated SQLITE_BUSY experiment (M18-B2), LA_BUSY_EXPERIMENT=1
  LA-C2        8f2ce15  fix shutdown drain (FR-RM-04)
  LA-C3        78c02ca  fix L7/L8 (committed terminal routing completed on recovery); L9 gated reproducers
REPAIRED (each: deterministic reproducer FAILED pre-fix, PASSES post-fix; adjacent suites green):
  L1/L3/L6 P1 — one transient AppendEvent failure (StepCompleted / StepFailed / RC-1's own StepFailed)
    left the instance `running` with a StepStarted no in-process path settled, until restart
    (RC-1 ran once per process: hydrate.go e.hydrated). Contract: Run "a transient storage error must
    not kill the engine" (tick.go); RC-1/B-0 restart semantics (7231199). Fix LA-C1 F1: on a failed
    processInstance/CompleteStep the instance's recovery bookkeeping is reset so RC-1 re-runs next tick
    (in-process recovery == restart recovery, worker_crash for non-signal steps).
  L2 P1 — a later item's claim/StepStarted/enterWait failure stranded earlier StepStarted items (never
    dispatched; after restart failed worker_crash though never run). Fix LA-C1 F2: already-started items
    are dispatched and settled, then the error is returned.
  L4a/L4b/L5 P1 — signal steps: a failed CreateWaitRecord / running→waiting write / post-delivery
    StepCompleted wedged the instance; after restart RC-1 failed the step worker_crash even when its
    signal was durably delivered (B3: SignalReceived ⇔ delivered; M07-C3r OUTPUT 0). Fix LA-C1 F3:
    RC-1 completes a signal step from a SignalReceived after its StepStarted, else re-enters the wait
    (duplicate wait_record no longer skips the running→waiting transition).
  SHUTDOWN P1 — SIGTERM/ctx cancel aborted the in-flight step (native runner returned timeout) and every
    settle write failed "context canceled"; after restart the step failed worker_crash. Contract:
    PRD FR-RM-04; TDS-07 §SIGTERM ("engine tick completes in-flight step → graceful drain"). Fix LA-C2:
    Run executes ticks on context.WithoutCancel; once stop is requested no trigger scan, no new
    claim/StepStarted/retry, no signal scans; public Engine.Tick semantics unchanged (S7 pins them).
  L7 P1 — StepFallbackActivated append failure after a committed StepFailed{retrying:false}: fallback
    never activated; instance failed "stalled". L8 P1 — WorkflowFailed append failure: instance failed
    "stalled" with NO compensation. Contract: EDR-011 §8 routing. Fix LA-C3: recovery completes the
    durably committed route; route decided from static structure only (Fallback → (1); no on_error
    transition at all → (3)); skipped when cancellation requested, code "cancelled" (cancel.go:94,191)
    or "signal_timeout" (signal_timeout.go:103).
M18-B5: acceptance criteria MET on this branch (LA-B5): SQLite triggers + app-defined crash_point()
  SIGKILL the child inside the B3 tx (after_guard, after_append, before_commit; child proves the point
  via "AT <stage>"); after reopen delivered_at NULL + 0 SignalReceived + status waiting; recovery
  delivers once, redelivery no-op; post-commit control; non-atomic variant detected at after_append /
  before_commit (meta-test in CI; temporary deliver.go mutation FAILED at test line 659). No production
  change. Closes on founder merge.
M18-B2: NOT REPRODUCED on the reported path (E5 CLI init→start→submit 0/100; existing-DB holder /
  checkpoint contention 0 failures). Adjacent finding LA-B2a (P3): concurrent first open of a
  NON-EXISTENT DB file → SQLITE_BUSY on `PRAGMA journal_mode=WAL` (124/400 child runs) and migration
  races (14/400); loser exits with a clean error; per-migration tx (migrate.go:102-122) keeps the
  schema consistent. No retry machinery added (no contract violation evidenced).
PRIORITY 4 (cross-instance progress): serial tick / per-instance wg.Wait is the specified design
  (Blueprint §Concurrency Model "single goroutine dispatcher"; FR-SE-15 max_parallel is a ceiling);
  bounded per step by its timeout (native default 30s). Accepted limitation, no change.
RELEASE EXCEPTIONS: unchanged. PRD §32 row 5 stays UNMEASURABLE at the CLI level (no `awis stop`
  system test); LA-C2 adds engine-level evidence only (TestLA_Shutdown_S1-S3, S6).
FOUNDER DECISIONS OPEN:
  LA-D1 (L9, confirmed, P1 wedge class) — compensation-bracket append failure. L9a: WorkflowCompensating
    append fails → instance `failed`, undo never runs, nothing recovers (Tick scans only running).
    L9b: WorkflowCompensated append fails after all undos ran → stuck `compensating` forever, also after
    restart. Undo actions emit no events (EDR-011 §5, 12 frozen event types), so the EventLog cannot say
    which undos ran: any recovery must either re-run undos (at-least-once undo) or assume completion —
    new semantics / frozen-format gap (E3-grade). Reproducers committed, gated (LA_DEFERRED=1, FAIL).
    Acceptance: founder picks the semantics (or CONTRA on the event set); then a recovery stage for
    failed/compensating instances with tests that FAIL today under LA_DEFERRED=1.
  LA-Q1 — a successful handler whose StepCompleted append failed is recovered as worker_crash (RC-1);
    recovering it from the idempotency cache would be new semantics. Acceptance: founder ruling; if
    adopted, L1 tests flipped to assert StepCompleted with the cached outputs.
BACKLOG (non-blocking, with acceptance):
  LA-R1 (P2, pre-existing) hydrate re-evaluates on_error conditions against the recovery-time env for
    pending targets; a condition whose truth changed since settle can activate/skip a different target.
    Acceptance: a test with a condition flipped between settle and recovery documents the chosen rule.
  LA-R2 (P2, pre-existing) subprocess runner has no default timeout (subprocess.go:77 applies only
    step.timeout): a hung subprocess freezes the serial tick and, since LA-C2, defers shutdown past
    `awis stop`'s 10s (kill -9 = crash path). Acceptance: founder picks a default (as native's 30s) or
    documents unbounded; test asserting the bound.
  LA-R3 (P3) LA-C2 residual window: a stop arriving between the processInstance stop check and the first
    claim still dispatches that tick's gathered items; they are drained and settled, never stranded.
  LA-B2a (P3) see M18-B2 above. Acceptance: decide whether first-open concurrency must succeed; if so a
    deterministic two-opener test passes 200/200.
EVIDENCE (local, at 78c02ca): `make verify` ALL GATES PASSED; all TestLA_* -count=20 PASS; storage
  MidTx|DeliverSignal -count=20 PASS; -race engine+storage -count=2 PASS. Real CI: recorded on the PR.
NEXT: independent review + real CI on the PR → founder review; founder rules LA-D1 / LA-Q1; merge is
  founder-only (EEOS rule 6). No further loop started.
