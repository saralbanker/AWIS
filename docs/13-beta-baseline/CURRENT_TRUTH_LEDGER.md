# AWIS — Current Truth Ledger

Status: ACTIVE — the single living state document for the Beta Baseline program.
Opened: 2026-09-18
HEAD at open: f3a897b (branch main)
Governed by: docs/13-beta-baseline/OPERATING_PROTOCOL.md

This file supersedes, for operational decision-making, every status/verdict/
readiness/audit document listed in §5. Those remain on disk as history until
Phase 4 retires them; they MUST NOT be used as decision inputs.

Every entry carries a proof token. Entries without one are UNKNOWN by rule.

---

## 1. Confirmed facts

| ID | Fact | Proof |
|----|------|-------|
| F-1 | main is at f3a897b; all other local branches are ancestors or stale. | `git rev-parse main` |
| F-2 | Working tree holds exactly 8 untracked files: 7 milestone contract docs + this program's OPERATING_PROTOCOL.md. No tracked file is modified. | `git ls-files --others --exclude-standard` (count 8); `git status --porcelain` shows no M/D rows |
| F-3 | `go build ./...` and `go vet ./...` clean on both modules. | Verifier run, exit 0, no output |
| F-4 | Full test suite PASSES: 22 tested packages `ok`, 7 no-test-files, 0 fail (main module); `ok github.com/awis/oip 31.99s`. | `go test -count=1 ./...` both modules |
| F-5 | `gofmt -l .` empty; `golangci-lint run ./...` reports "0 issues." on both modules at CI's pinned v2.12.2. | Verifier run |
| F-6 | `-race` clean on internal/engine, internal/signal, internal/storage — no DATA RACE reports. | `go test -race` on those three packages |
| F-7 | AWIS-E1 zero-AI acceptance gate passes 8/8 subtests (linear, fan-out/join, retry-success, retry-exhaust-fallback, failure-compensation, cancel-mid-flight, trigger-ingestion, intelligence-absent-fallback). | `go test ./internal/engine/... -run TestE1 -count=1` |
| F-8 | A real workflow executes end to end through the CLI: `init` -> `start` -> `submit hello-world --input name=World --wait` -> `Status: completed`, and `trace` renders a full StepStarted/StepCompleted/WorkflowCompleted timeline. | Verifier runtime smoke, exit 0 |
| F-9 | API server starts and serves: `/api/v1/healthz` 200 `{"status":"ok"}`, `/info` 200, `/workflows` 200, `/instances` 200, unknown path 404. | Verifier smoke on 127.0.0.1:18091, scratchpad DB |
| F-10 | GUI builds: esbuild produces bundle.js 43.9kb, byte-identical to the committed artifact (`git diff` empty). | `npm run build` in web/ |
| F-11 | CI exists: .github/workflows/ci.yml, two jobs — `verify` (gofmt/vet/lint/oip-isolation/build/test/race/e1) and `docs-lint`. | file read |
| F-12 | The API surface is 7 routes, ALL GET. There is no submit/cancel/signal over HTTP. | internal/api/router.go |
| F-13 | M15, M16, M17 branch tips are all ancestors of main. Their code is IN main. | `git merge-base --is-ancestor <branch> main` -> true for all three |
| F-14 | They reached main by STACKING, not by a gated merge: the first commit on each ancestry path to main is the NEXT milestone's A-INIT commit (m15 -> "M16 A-INIT", m16 -> "M17 A-INIT", m17 -> a B-2/B-3 hardening commit). No merge or squash commit represents a gate decision. | `git log --ancestry-path <branch>..main \| tail -1` |
| F-15 | api_keys/ is git-ignored at .gitignore:40 and was never committed. No credential exposure in history. | `git check-ignore -v`; `git log --all -- api_keys/` empty |
| F-16 | Repo binaries (awis, awis-server) and awis-server.db* are untracked and ignored. | `.gitignore`; `git ls-files` |
| F-17 | All 14 product surfaces have real implementing code; the only stub is `Embed` -> `ErrEmbedUnavailable`, intentional per CONTRA-3. No TODO/FIXME anywhere in internal/, cmd/, sdk/, plugins/. | Investigator grep sweep |
| F-18 | m17-c3-wip is the ONLY branch not in main (1 commit ahead). 6 worktree-agent-* branches and 3 origin/claude/* audit branches exist; worktree branches are 0 ahead. | `git log main..<ref>` per branch |
| F-19 | Full-scope CI-parity race PASSES: `make race` (`-race ./...` on both modules) green 2/2, second run after `go clean -testcache` to force real re-execution. No DATA RACE block anywhere. The prior session's reported transient race failure did NOT reproduce. | M17-V1 re-run, isolated worktree |
| F-20 | Python suites pass 41/41 (`make pytest`: awis-step 8, awis-plugin 16, git-context-plugin 17). | M17-V1 re-run |
| F-22 | The M15 P2 seam (AND-join / fallback-join sentinel) is PRESENT in BOTH the forward path (internal/engine/emit.go:374-391) and the rebuild path (internal/storage/rebuild.go:403-415), writing an identical empty-map sentinel, preserving EDR-007 forward-equals-rebuild equivalence. Covered by 19 test references incl. rebuild_test.go:169-181. Director-verified, not taken on report. | sed/grep on both files; test grep |
| F-21 | M17 row 3 (replay goldens + TDS-07) is SATISFIED: replay.json/replay.txt exist AND are consumed (`c1r_test.go:81 checkGolden`), 42 golden tests pass, every command has a TDS-07 section in docs/CLI_CONTRACT.md. | `go test -run 'TestReplayGolden...'` PASS |

Note on F-17: this is rank-1 evidence of code EXISTENCE only. It is not evidence
of correctness. Correctness for each surface is tracked in the Phase 1 matrix.

## 2. Confirmed defects

| ID | Sev | Defect | Proof | State |
|----|-----|--------|-------|-------|
| D-1 | P1 | CI was RED on main: docs-lint failed on a clean checkout for ALL THREE of M15/M16/M17 against the 7-file contract. | `git archive HEAD docs scripts \| tar -x && bash scripts/docs-lint.sh` -> exit 1, three errors | **FIXED on branch beta-baseline/docs-lint-green** (24449d9). Clean-checkout re-run: `docs-lint: OK`, EXIT=0. NOT yet merged to main. |
| D-1a | — | M15 cause: 2 EXTRA tracked files at maxdepth 1 — BOUNDARY_EVIDENCE.md, G3_BRIEF.md (9 files vs contract 7). | `git ls-files` + find -maxdepth 1 | FIXED: git mv into cards/ (DEC-3) |
| D-1b | — | M16 cause: 4 contract files MISSING from HEAD (AI_EXECUTION_CONTEXT, DEPENDENCY_MAP, HANDOFF, TRACEABILITY) — they exist untracked in the working tree. Tracked+untracked union = exactly the 7. | clean-checkout listing vs `git ls-files --others` | FIXED: committed |
| D-1c | — | M17 cause: 2 contract files MISSING from HEAD (AI_EXECUTION_CONTEXT, DEPENDENCY_MAP) — exist untracked. Union = exactly the 7. | same | FIXED: committed |
| D-2 | P1 | Milestone ledger contradicts git. STATE.md records M15/M16 as "PHASE: E-MERGE (blocked on founder)" and M17 as "PHASE: C-VERIFY", while all three are already in main (F-13, F-14). | docs/05-implementation/STATE.md L103,L127,L139 vs F-13/F-14 | OPEN |
| D-3 | P1 | M17 shipped to main carrying a FAILED verification, now CONFIRMED by independent re-run (2026-09-18, isolated worktree, HEAD f3a897b): M17-V1 verdict FAIL. Row 1 FAIL, row 8 literal wording FAIL, row 7 UNKNOWN. Row 3 re-measured PASS (see F-21). | M17-V1 independent re-run | CONFIRMED |
| D-6 | P2 | UNREPRODUCED INTERMITTENT. One real observed timeout of TestSystemRehearsalInitStartSubmitTrace under `-race -count=3` (rep 3, deadline 4m30s). Root cause UNKNOWN: NOT reproduced in 17 subsequent repetitions across 5 invocations, two matching the exact command, two under doubled CPU contention. `make race` green 2/2. | M17-V1 re-run (1 fail) vs root-cause pass (0/17 fail) | OPEN — known risk, not a Beta blocker |
| D-6a | — | CORRECTION of record: the failure's `last observed status: ""` does NOT indicate the instance was never scheduled. It is a Go zero-value. The test only inspects the `Recent` (terminal-only) bucket and never the `Active` bucket, so a healthy running instance produces the identical message. Any inference of scheduling starvation from that string is unsupported. | cmd/awis/system_test.go:403-443; cmd/awis/status.go buildStatusJSON | RECORDED |
| D-8 | P2 | PRODUCT CODE swallows errors: cmd/awis/status.go per-status-bucket ListInstances loop does `if err != nil { continue }`, so a storage error silently yields an incomplete status report with no diagnostic. Not proven to have fired in D-6, but a real latent defect in user-facing status reporting. | cmd/awis/status.go:89-147, 247-262 | OPEN |
| D-9 | P2 | TEST DEFECT destroyed diagnosability: system_test.go's rehearsal poll reads only the terminal `Recent` bucket, so it can never observe or report a non-terminal state, and its failure message is misleading by construction. This is why D-6 is un-diagnosable from its own output. | cmd/awis/system_test.go:403-443 | **FIXED** on branch beta-baseline/d9-test-diagnosability (3f2eeaf). Observability only; deadline and terminal-status gate unchanged. Unmerged. |
| D-7 | P3 | M17's VALIDATION_CHECKLIST.md still records row 3 as `[ ] FAIL` although row 3 now passes (F-21). The checklist was never updated after C1r wired the goldens. | checklist file vs execution | FIXED (b349ed6): row 3 -> PASS |
| D-4 | P2 | Authority drift: ~20 documents independently claim CANONICAL/FINAL/AUTHORITATIVE over overlapping scopes. TRUTH_CLOSURE.md declares 24 documents superseded while all 24 remain in place and readable as current. | Authority map inventory | OPEN (Phase 4) |
| D-5 | P2 | docs/05-implementation/STATE.md — the EEOS execution ledger — was last updated 2026-08-21 and predates 30+ subsequent hardening commits. | `git log -1 -- docs/05-implementation/STATE.md` | OPEN (Phase 4) |

No P0 defect has been found. Engine, CLI, API, GUI, and storage all execute.

## 3. Confirmed unknowns

| ID | Unknown | Why it is unknown | Resolution path |
|----|---------|-------------------|-----------------|
| U-1 | RESOLVED -> F-19. Full-scope race is green 2/2. | — | CLOSED |
| U-2 | RESOLVED -> D-6 (row 1 FAIL) and F-21 (row 3 PASS). | — | CLOSED |
| U-7 | D-6's root cause. B2 (test swallowing an error) is RULED OUT with high confidence by code reading. All other mechanisms remain open and unobserved; 0/17 reproduction. Cannot be resolved without capturing a live failing run with its DB preserved. | Not reproducible on demand | OPEN — parked |
| U-8 | M17 row 7 (PRD §32 mapping). The checklist item declares itself non-binary and spot-check-only, so it cannot be measured as PASS/FAIL as written. | Ambiguous acceptance criterion | Needs DEC-4 |
| U-3 | M17-V1 row 8 (frozen-surface scope diff). | Reserved to founder/CE by the ledger; NOT resolvable by any agent | DECISION DEC-2 |
| U-4 | Test-suite flakiness under repetition. One clean deterministic `-count=1` pass observed; no repeat sweep. | Not executed — time budget | Phase 1 if warranted |
| U-5 | Whether the founder verbally approved the M15/M16 merges before they entered main. | Human fact, not in the repository | DECISION DEC-1 |
| U-6 | Runtime behavior of the 3 origin/claude/* audit branches and m17-c3-wip (1 commit ahead). | Not examined | Phase 4 |

## 4. Decisions (founder-only)

Ruled 2026-09-18 by the founder. A ruling is a decision, not evidence — it
changes what we do, not what is true. Facts remain as measured.

| ID | Decision | RULING | Execution state |
|----|----------|--------|-----------------|
| DEC-1 | M15/M16 gates, given the code is already in main via stacking (F-13, F-14). | **Perform G3 + TDS-06 retroactively.** | PARTIAL. G3 Q1 (platform boundary/QG-4) = **PASS**, founder ruling 2026-09-18, recorded at cb07eba; brief corrected to name three seams. G3 Q2 (TDS-06 sign-off) = **STILL PENDING**; RECORD_FORMAT.md remains DRAFT. STATE.md stays unreconciled until Q2 signs. |
| DEC-2 | M17 row 8 frozen-surface scope wording vs measured diffs. | **WAIVED — diffs are legitimate.** | DONE (b349ed6). Row 8 recorded WAIVED, not PASS. |
| DEC-3 | M15's two extra tracked files vs the 7-file contract. | **Move both into cards/.** | DONE (24449d9), incl. 5 broken path citations repaired. |
| DEC-4 | M17 row 7 is written as a non-binary, self-declared spot-check and cannot be measured as PASS/FAIL. | OPEN | Blocks M17 closure. Needs restating or formal waiver. |
| DEC-7 | TDS-06 has three specification defects found on review (dead `superseded` enum; `distinguishes` required/optional self-contradiction; ID scheme has no concurrency rule). Fix before signing, or sign and amend later. | OPEN | Blocks G3 Q2, therefore DEC-1, therefore D-2. |
| DEC-5 | M17 row 1 / D-6. | **Fix D-9 first, then re-measure.** | D-9 FIXED (3f2eeaf). Re-measure done: 0/3 reproduction under the instrumented test. Row 1 still not closeable on evidence — D-6 has not recurred but has no root cause. Needs a final call once you decide how long to keep probing. |
| DEC-6 | Landing strategy for the D-1 repair. | **Branch, verify CI green, then merge.** | Branch built and verified green. MERGE TO MAIN PENDING AUTHORIZATION — not pushed, not merged. |

## 5. Claims disproven by evidence

Recorded so they are never re-derived. Each of these appears in a document or a
subagent report and is FALSE against the repository.

| Claim | Source | Reality |
|-------|--------|---------|
| "api_keys/ is unignored — one `git add -A` from committing live keys." | Investigator (repository map) | FALSE. Ignored at .gitignore:40, never committed (F-15). |
| 13+ root authority documents are "untracked (no git history)". | Investigator (authority map) | FALSE. All spot-checked files are TRACKED (F-2). |
| docs-lint fails only on M15, caused solely by tracked files, "reproducible on clean HEAD". | Verifier (execution truth) | INCOMPLETE. It ran against the dirty working tree. On a true clean checkout M16 and M17 fail as well, because their contract files are absent from HEAD (D-1b, D-1c). |
| Engine readiness 3.9/10; "REJECTED FOR RELEASE". | ENGINE_READINESS_SCORECARD.md, FINAL_VERDICT.md | STALE. Pre-hardening. Current: full suite green, E1 8/8, workflow executes end to end (F-4, F-7, F-8). |
| M15/M16 are "blocked on founder" / awaiting merge. | docs/05-implementation/STATE.md | STALE. Already in main (F-13, F-14). |

Methodology note: two of three Investigator git-tracking claims were wrong.
Any Investigator claim about git tracking state is spot-checked by the Director
before it enters this ledger.

## 6. Phase gate status

- Phase 0 — Truth: **COMPLETE**. Maps built, state validated by execution,
  ledger created, U-1 and U-2 closed by measurement.
- Phase 1 — Product health: matrix seeded by F-3..F-10, F-17. Surfaces execute;
  per-feature depth verification not yet performed.
- Phase 2 — Defect elimination: D-1 FIXED on branch and verified green; awaiting
  merge authorization. D-7 fixed. Remaining open: D-6/D-8/D-9, all P2, none
  Beta-blocking.
- Phase 3 — Milestone closure: M17 is CONFIRMED NOT CLOSEABLE (D-3). M15/M16
  blocked on DEC-1 only. M17 additionally blocked on DEC-2, DEC-4, and D-6.
- Phase 4 — Consolidation: scoped by D-4, D-5. Not started.
- Phase 5 — Beta Baseline: NOT DECLARED. Gate is P0=0 (MET — none found) and
  P1=0 (NOT met: D-1, D-2, D-3). D-6 downgraded to P2 and is not a blocker.
  D-1 is the only P1 with a known bounded repair; it needs DEC-3 only.
