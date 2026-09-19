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

DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06 M07 M08 M09 M10 M11 M12 M13 M14 M15 M16 M17   # history: git log + module HANDOFFs
# M09 merge sha 03d5045 (founder squash-merge, verified against main log at reconciliation)
# M10-M17 all reached main by branch stacking, not a discrete founder squash-merge; no
# commit represents a merge or gate decision for any of these eight. M15's two founder
# gates were performed retroactively; the other seven carried no gate. See LAST-9
# (2026-09-19) and each milestone's own block above.
