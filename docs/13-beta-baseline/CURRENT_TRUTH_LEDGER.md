# AWIS — Current Truth Ledger

Status: ACTIVE — the single living state document for the Beta Baseline program.
Opened: 2026-09-18 · Last reconciled: 2026-09-19
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
| F-14 | They reached main by STACKING, not by a gated merge: the first commit on each ancestry path to main is the NEXT milestone's A-INIT commit (m15 -> "M16 A-INIT", m16 -> "M17 A-INIT", m17 -> a B-2/B-3 hardening commit). No merge or squash commit represents a gate decision. | `git log --ancestry-path <branch>..main \ | tail -1` |
| F-15 | api_keys/ is git-ignored at .gitignore:40 and was never committed. No credential exposure in history. | `git check-ignore -v`; `git log --all -- api_keys/` empty |
| F-16 | Repo binaries (awis, awis-server) and awis-server.db* are untracked and ignored. | `.gitignore`; `git ls-files` |
| F-17 | All 14 product surfaces have real implementing code; the only stub is `Embed` -> `ErrEmbedUnavailable`, intentional per CONTRA-3. No TODO/FIXME anywhere in internal/, cmd/, sdk/, plugins/. | Investigator grep sweep |
| F-18 | m17-c3-wip is the ONLY branch not in main (1 commit ahead). 6 worktree-agent-* branches and 3 origin/claude/* audit branches exist; worktree branches are 0 ahead. | `git log main..<ref>` per branch |
| F-19 | Full-scope CI-parity race PASSES: `make race` (`-race ./...` on both modules) green 2/2, second run after `go clean -testcache` to force real re-execution. No DATA RACE block anywhere. The prior session's reported transient race failure did NOT reproduce. | M17-V1 re-run, isolated worktree |
| F-20 | Python suites pass 41/41 (`make pytest`: awis-step 8, awis-plugin 16, git-context-plugin 17). | M17-V1 re-run |
| F-28 | Provider-scope discrepancy, recorded and DELIBERATELY NOT RESOLVED (founder ruling: out of Beta scope). Founder states three targets (Anthropic, OpenRouter, Local LLM); docs/12-intelligence-architecture-decision/ARCHITECTURE_DECISION_RECORD.md states "Nine providers are in scope: Anthropic, OpenAI, Gemini, Ollama, OpenRouter, LM Studio, vLLM, Together, Groq". Neither is a milestone contract; neither blocks Beta. Open question, not a defect. | ADR quote; F-26 |
| F-26 | NO milestone contract (M00-M18) requires OpenRouter or Local LLM: zero hits across docs/05-implementation/. All OpenRouter mentions sit in docs/11-intelligence-architecture/ and docs/12-intelligence-architecture-decision/ — ADR/planning prose, not contracts. "Local LLM" appears NOWHERE in the repository. Only `anthropic` and `null` adapters exist. Per founder clarification these are roadmap and do NOT fail Beta. | grep across docs/05-implementation/; adapters dir listing |
| F-27 | Plugin spawn facts (for D-11): spawn is at internal/plugin/transport.go:114. SysProcAttr IS set — `&syscall.SysProcAttr{Setpgid: true}` — but only for process-group reaping, not isolation. Env is manifest env + PATH only (NFR-S-02). DB path defaults to ./.awis/runtime.db; dir created 0o755; NO explicit mode set on the .db file. `golang.org/x/sys v0.44.0` is ALREADY a dependency (indirect). No Landlock/seccomp/unshare/chroot/setuid/Credential reference exists anywhere. CI matrix: ubuntu-latest AND macos-latest. | transport.go:100-137; go.mod; ci.yml |
| F-25 | Intelligence provider scope, per founder clarification 2026-09-18: architecture/planning define THREE provider targets — Anthropic, OpenRouter, Local LLM. Only Anthropic is implemented. OpenRouter and Local LLM are future roadmap and MUST NOT fail Beta unless a milestone contract explicitly required them (being verified). Anthropic remains in scope and is not removed. | Founder clarification; milestone-contract check in flight |
| F-23 | PRD §32 (AWIS_PRD.md:2156-2247, 59 acceptance rows) has been exhaustively audited for the first time: **49 SATISFIED, 6 NOT SATISFIED (rows 8, 15, 37, 38, 40, 53), 1 PARTIAL (row 44), 3 UNMEASURABLE (rows 5, 28, 32)**. Director recount from the Verifier's row-level verdicts; the agent's own summary tally was self-inconsistent and is not used. | PRD §32 audit, row-by-row with per-row proof |
| F-24 | Measured performance comfortably meets NFRs: native dispatch P95 = 1ms (limit 50ms, 60 step-pairs); signal delivery ~63.5ms (limit 200ms); `awis trace` over a purpose-built 100,228-event DB = 0.005s (limit 500ms); rebuild over 100,999 events = 446ms (limit 30s). | PRD §32 audit rows 56-59 |
| F-22 | The M15 P2 seam (AND-join / fallback-join sentinel) is PRESENT in BOTH the forward path (internal/engine/emit.go:374-391) and the rebuild path (internal/storage/rebuild.go:403-415), writing an identical empty-map sentinel, preserving EDR-007 forward-equals-rebuild equivalence. Covered by 19 test references incl. rebuild_test.go:169-181. Director-verified, not taken on report. | sed/grep on both files; test grep |
| F-21 | M17 row 3 (replay goldens + TDS-07) is SATISFIED: replay.json/replay.txt exist AND are consumed (`c1r_test.go:81 checkGolden`), 42 golden tests pass, every command has a TDS-07 section in docs/CLI_CONTRACT.md. | `go test -run 'TestReplayGolden...'` PASS |

Note on F-17: this is rank-1 evidence of code EXISTENCE only. It is not evidence
of correctness. Correctness for each surface is tracked in the Phase 1 matrix.

## 2. Confirmed defects

| ID | Sev | Defect | Proof | State |
|----|-----|--------|-------|-------|
| D-1 | P1 | CI was RED on main: docs-lint failed on a clean checkout for ALL THREE of M15/M16/M17 against the 7-file contract. | `git archive HEAD docs scripts \ | tar -x && bash scripts/docs-lint.sh` -> exit 1, three errors | **RESOLVED ON MAIN.** Merged as a fast-forward 2026-09-18 (founder ruling: land the docs-lint fix alone). main is at 24449d9 (now 731a1c2); clean-checkout docs-lint `OK` EXIT=0; build and vet clean. CI is GREEN on main. |
| D-1a | — | M15 cause: 2 EXTRA tracked files at maxdepth 1 — BOUNDARY_EVIDENCE.md, G3_BRIEF.md (9 files vs contract 7). | `git ls-files` + find -maxdepth 1 | FIXED: git mv into cards/ (DEC-3) |
| D-1b | — | M16 cause: 4 contract files MISSING from HEAD (AI_EXECUTION_CONTEXT, DEPENDENCY_MAP, HANDOFF, TRACEABILITY) — they exist untracked in the working tree. Tracked+untracked union = exactly the 7. | clean-checkout listing vs `git ls-files --others` | FIXED: committed |
| D-1c | — | M17 cause: 2 contract files MISSING from HEAD (AI_EXECUTION_CONTEXT, DEPENDENCY_MAP) — exist untracked. Union = exactly the 7. | same | FIXED: committed |
| D-2 | P1 | Milestone ledger contradicts git. STATE.md records M15/M16 as "PHASE: E-MERGE (blocked on founder)" and M17 as "PHASE: C-VERIFY", while all three are already in main (F-13, F-14). | docs/05-implementation/STATE.md L103,L127,L139 vs F-13/F-14 | **RESOLVED — pending merge** (a45005f). TDS-06 signed by the founder 2026-09-19, satisfying G3 Question 2; with Q1 passed 2026-09-18, Gate G3 is COMPLETE. STATE.md reconciled: all eight milestones M10-M17 now read PHASE: CLOSED against repository reality. Director-verified: no live 'blocked on founder' field remains, zero DRAFT occurrences in RECORD_FORMAT.md. |
| D-3 | P1 | M17 shipped to main carrying a FAILED verification (M17-V1 FAIL, confirmed by independent re-run). | M17-V1 independent re-run | **RESOLVED** (9c477b5). All eight rows now dispositioned: row 3 PASS (re-measured), row 8 WAIVED (DEC-2), row 1 CLOSED WITH RISK ACCEPTED (DEC-5), row 7 MEASURED WITH EXCEPTIONS (DEC-4). M17 reaches D-CLOSE with its waivers on record. |
| D-6 | **P1** | **ROOT CAUSE CONFIRMED (HIGH confidence, stuck DB captured).** An ORPHANED STEP CLAIM wedges the instance permanently. `ClaimStep` (internal/storage/sqlite.go:914-961) commits the claim in its OWN transaction, separate from the `StepStarted` emission in `gatherDispatch` (internal/engine/tick.go:251-268). If that emission fails — transient storage error under concurrent load — gatherDispatch returns the error and aborts with NO compensating release of the committed claim. Claims release ONLY on terminal-status UpsertInstance (sqlite.go:594-601) or a RebuildState wipe; there is no TTL and no other release path. `activatableFor` (internal/engine/failure.go:116-141) and hydrate.go never consult `step_claims`, so the step stays 'activatable' forever; each tick re-claims and permanently gets won=false (PK exists), swallowed as a 'claim lost' log line (tick.go:256); the completion gate (tick.go:211) refuses to finalise while anything is activatable. Permanent catch-22: status=running, current_steps=[], forever. NOT transient. | Captured DB: 3 execution_events (WorkflowStarted, StepStarted greet, StepCompleted greet) but TWO step_claims rows — `log`'s claim exists with NO corresponding event. workflow_instances: status=running, current_steps=[], version=5. | **RESOLVED — pending merge.** Fix 7808d23 INDEPENDENTLY VERIFIED PASS: the fault-injection test was re-confirmed to fail against neutralised code and pass against the fix; a 15-iteration / 45-process load soak at the exact reproducing concurrency found ZERO occurrences, including iterations 5 and 13 where it historically struck; the double-execution hazard is closed (expiry + StepStarted checks and the reclaim UPDATE are atomic in one transaction; expired-with-StepStarted and legacy-NULL both verified non-reclaimable); StoragePort diff empty; migration 0008 verified fresh AND upgrade-from-0007; exactly 7 mechanical 7->8 test edits, nothing weakened; AWIS-E1 green. Compensating release via additive `claimReleaseStore` (StoragePort zero-diff confirmed); migration 0008 adds `expires_at`; reclaim requires BOTH expiry AND no StepStarted, in the same transaction as the conflicting INSERT, with legacy NULL leases never reclaimed; lease 5 min; claim-loss escalates to Error after 3 consecutive losses. Fault-injection test demonstrated to FAIL against unfixed code and PASS with the fix. NOTE: `hello-world` is TWO steps (greet -> log, final_steps=[log]); the Director's earlier 'single-step' characterisation was wrong — it is the SECOND step that gets orphan-claimed. |
| D-6a | — | CORRECTION of record: the failure's `last observed status: ""` does NOT indicate the instance was never scheduled. It is a Go zero-value. The test only inspects the `Recent` (terminal-only) bucket and never the `Active` bucket, so a healthy running instance produces the identical message. Any inference of scheduling starvation from that string is unsupported. | cmd/awis/system_test.go:403-443; cmd/awis/status.go buildStatusJSON | RECORDED |
| D-8 | P2 | PRODUCT CODE swallows errors: cmd/awis/status.go per-status-bucket ListInstances loop does `if err != nil { continue }`, so a storage error silently yields an incomplete status report with no diagnostic. Not proven to have fired in D-6, but a real latent defect in user-facing status reporting. | cmd/awis/status.go:89-147, 247-262 | **RESOLVED** (209ec26, merged). printStatusTable now fails with an operational error (exit 1) instead of `continue` on a ListInstances failure. Deliberately NOT an added field on statusOutputJSON — that is the frozen TDS-07 §4 schema; failing before the JSON branch covers both paths and gives a machine consumer a stronger signal (nonzero exit, empty stdout) than a partial document. Success-path output verified byte-identical: the status goldens AND TestSystemRehearsalInitStartSubmitTrace (which parses `status --all --json`) pass unmodified. |
| D-9 | P2 | TEST DEFECT destroyed diagnosability: system_test.go's rehearsal poll reads only the terminal `Recent` bucket, so it can never observe or report a non-terminal state, and its failure message is misleading by construction. This is why D-6 is un-diagnosable from its own output. | cmd/awis/system_test.go:403-443 | **RESOLVED ON MAIN.** Merged 2026-09-19 as 731a1c2 (true merge commit, not a stack). Observability only; deadline and terminal-status gate verified unchanged. Every run off main is now instrumented. |
| D-10 | P3 | AWIS_RECONSTRUCTION_ANSWERS.md:35 describes the archived reports as the "Root (untracked audit layer)". Wrong twice: they are no longer at root (a650b86), and they were never untracked (F-2, and the disproven claim in section 5). A false claim preserved inside a doc retained as a spec. | sed -n '35p'; `git ls-files` | **RESOLVED — pending merge** (a45005f). The Executor checked commit a650b86 directly rather than trusting the Director's framing, and found only 19 of the named documents actually moved while 5 remained at root; the corrected sentence states what actually happened. |
| D-11 | P0 | Plugin subprocesses could read runtime.db; PRD §32 rows 38/53 asserted they could not. | PRD §32 audit; verification of c99b3c7; AC restatement 3f22284 | **RESOLVED** per founder ruling DEC-14. Two parts: (1) MECHANISM — c99b3c7 adds `plugins_user`; when set, plugins spawn under a dedicated uid/gid via SysProcAttr.Credential and runtime.db + sidecars are 0600. Independently verified correct when configured, with an explicit EPERM error and no silent unisolated fallback. (2) TRUTH — 3f22284 restates rows 38/53 to the guarantee actually delivered and documents the deployment requirement at PRD §27:1898. RESIDUAL, KNOWINGLY ACCEPTED: in a default unprivileged deployment a plugin still reads runtime.db; this is now documented rather than denied. No unprivileged, portable mechanism exists (F-30). |
| D-12 | P2 | **RECLASSIFIED — the PRD is the defective document, not the code.** docs/CLI_CONTRACT.md (TDS-07, "Canonical — Implementation-Ready", authority for M14-C1..C4) specifies exit 3 for validation failure and `--input k=v` / `@file.json`. docs/CLI.md agrees and declares TDS-07 authoritative on conflict. The code agrees (cmd/awis/errors.go:28-31, workflow.go:87,111). Only AWIS_PRD.md §32 rows 8 and 15 dissent. Per the evidence hierarchy an implementation spec outranks the PRD, so the repair is to amend the PRD — NOT to change shipped CLI behaviour. | CLI_CONTRACT.md §3 L50-62, L535-571; CLI.md L1-6; cmd/awis/errors.go:28-31 | **RESOLVED** (49987fe). PRD §32 rows 8 and 15 and §18's format block corrected to match TDS-07; precedence note added to §37 Cross-References (AWIS_PRD.md:2545). Director-verified against the diff. No code changed. |
| D-12c | P2 | **RESOLVED — NOT a code defect.** PRD §18 specified a `✗ Validation failed:` header; the CLI deliberately strips exactly that line (cmd/awis/workflow.go:103-105) and emits the contract's form instead. Semantic errors DO render `Line N:` + Suggestion + Example; parse errors carry the line number inside the yaml library's message. The code follows the governing spec (TDS-07); PRD §18 was a third instance of the same staleness. | cmd/awis/workflow.go:101-105; live CLI runs; CLI_CONTRACT.md L549-555 | **CLOSED** (49987fe — §18 now documents both error shapes). |
| D-15 | P2 | **RECLASSIFIED — TEST-HARNESS DEADLOCK, NOT AN ENGINE DEFECT.** cmd/awis/system_test.go wires the daemon's Stdout AND Stderr to one os.Pipe, and `waitForHeader` returns the instant it sees "● running" — after which NOTHING ever reads `pr` (no io.Copy, no drain goroutine; four sites: ~L117-131, L205, L257, L334-345). When the ~64KB kernel pipe buffer fills, the daemon's log write BLOCKS. Run()'s `e.logger.Error("tick failed", ...)` is synchronous in the ticker goroutine, so a blocked write FREEZES THE TICK LOOP PERMANENTLY — the instance never dispatches and the trace shows only WorkflowStarted. Amplified by D-17: an aborted tick logs every interval, filling the buffer fast. Mechanism CONFIRMED by direct reading (Director-verified); that it is what happened in the single observed occurrence is MEDIUM confidence, not observed live. | cmd/awis/system_test.go:334-345, waitForHeader; internal/engine/tick.go Run() | OPEN — no longer a Beta blocker. Fix = drain the pipe (and capture it as the diagnostic the D-9 instrumentation still lacks). |
| D-16 | P3 | docs/CLI.md:231 documents `awis trace <instance-id> [--json] [--full]`, placing --json AFTER the subcommand. That contradicts the same file's synopsis at line 14 (`awis [--data-dir <path>] [--json] <command> [flags]`) and CLI_CONTRACT.md:66 ("Every command supports `--json` as a global flag"). The global form works (exit 0); the form as documented on line 231 fails with `flag provided but not defined: -json`. | docs/CLI.md:14 vs :231; live CLI | OPEN |
| D-17 | **P1** | **ONE INSTANCE'S ERROR ABORTS THE ENTIRE ENGINE TICK.** internal/engine/tick.go SCAN_RUNNABLE loop: `for _, inst := range instances { if err := e.processInstance(ctx, inst); err != nil { return err } }`. A single instance's error returns from Tick, so EVERY remaining instance in the list is skipped, and so are `signalScan` and `signalTimeoutScan`, which sit below the loop. Run() then logs "tick failed" at Error and continues, so the daemon spins indefinitely making no progress past the failing instance. This CONTRADICTS the file's own header comment (~line 16): "a transient storage error must not kill the loop". It does not kill the daemon, but it kills every tick, for every instance, indefinitely. Wires directly into D-15: gatherDispatch returns the emission error after the D-6 compensating release, propagating gatherDispatch -> processInstance -> Tick -> abort. An instance erroring during dispatch can therefore STARVE OTHER INSTANCES — including one that has only emitted WorkflowStarted, which is exactly the D-15 signature. | internal/engine/tick.go SCAN_RUNNABLE loop and Run(); Director-read, quoted verbatim | **OPEN — likely the D-15 mechanism, under verification.** Found by targeted Director reading, not by a test: no test exercises two concurrent instances where one errors. |
| D-18 | P2 | **Engine can be frozen by a blocked log consumer.** internal/engine/tick.go Run() calls `e.logger.Error("tick failed", ...)` synchronously in the same goroutine that drives the ticker. If the log writer blocks — a full pipe, a stalled collector, a slow consumer on `awis start | ...` — workflow execution stops entirely and never recovers. Benign when stderr is a file or terminal; a real robustness hazard otherwise, and the proximate freeze mechanism behind D-15. | internal/engine/tick.go Run(); D-15 chain | OPEN |
| D-13 | P2 | Observability features specified but absent. Row 37: `plugin status` has no PID, call-count or latency — `PluginRow` (internal/storage/plugins.go:34-41) has no such fields at all, so this is missing schema, not missing formatting. Row 44: `awis metrics` computes no completion/failure rates and no latency percentiles, only counts and avg/min/max. | PRD §32 audit rows 37, 44 | **DEFERRED to post-Beta** per founder ruling DEC-11 (2026-09-18). Not a Beta blocker. `plugin status` lacks PID/call-count/latency because PluginRow (internal/storage/plugins.go:34-41) has no such FIELDS — this is missing schema and instrumentation, i.e. feature work, not stabilisation. `awis metrics` likewise computes no rates or percentiles. Recorded as a known, accepted divergence from PRD §32 rows 37 and 44. |
| D-14 | P3 | `awis status --watch` refreshes every 1s, not the specified 5s. Deliberate, with an in-code comment acknowledging the deviation from TDS-07. | cmd/awis/status.go:7-8,81 | **RESOLVED** (9c477b5) per DEC-11: founder ruled the implementation correct and the spec stale. docs/CLI_CONTRACT.md and docs/CLI.md now state 1s. AWIS_PRD.md also carries the 5s figure in 5 places and was deliberately NOT edited — flagged, not silently changed. |
| D-7 | P3 | M17's VALIDATION_CHECKLIST.md still records row 3 as `[ ] FAIL` although row 3 now passes (F-21). The checklist was never updated after C1r wired the goldens. | checklist file vs execution | FIXED (b349ed6): row 3 -> PASS |
| D-4 | P2 | Authority drift: ~20 documents independently claimed CANONICAL/FINAL/AUTHORITATIVE over overlapping scopes; TRUTH_CLOSURE.md declared 24 superseded while all remained readable as current. | Authority map inventory | **LARGELY RESOLVED** on beta-baseline/phase4-archive (a650b86): 23 stale operational reports relocated into the repository archive directory with history preserved; root .md 46 -> 23; the archive README now states non-authority and points at this ledger. Residual: the ~7 remaining spec-class docs still assert overlapping canonical scope. |
| D-5 | P2 | **BROADER THAN FIRST SCOPED.** STATE.md, the EEOS execution ledger, recorded **EIGHT** milestones as pending while all eight were already in main: M10-M14 as `E-MERGE (blocked on founder)`, M15/M16 as blocked on gates, M17 as mid-verification. Every one of m10..m17 is an ancestor of main with 0 commits ahead; milestone/M14 is even tagged. All eight reached main by BRANCH STACKING — each sitting beneath the next milestone's work — so no merge commit ever represented a merge decision. Unlike M15, M10-M14 carry `GATE: none`, so there was no gate to perform retroactively; only the founder squash-merge, which never happened as a discrete act. | `git merge-base --is-ancestor <branch> main` for m10..m17, all true, all 0 ahead; STATE.md L8,33,50,68,83 | **RESOLVED — pending merge** (a45005f). All eight milestones reconciled, including the M10-M14 scope gap the Director originally missed. The stacking history is recorded factually in STATE.md rather than erased. |

No P0 defect has been found. Engine, CLI, API, GUI, and storage all execute.

## 3. Confirmed unknowns

| ID | Unknown | Why it is unknown | Resolution path |
|----|---------|-------------------|-----------------|
| U-1 | RESOLVED -> F-19. Full-scope race is green 2/2. | — | CLOSED |
| U-2 | RESOLVED -> D-6 (row 1 FAIL) and F-21 (row 3 PASS). | — | CLOSED |
| U-7 | D-6's root cause. B2 (test swallowing an error) is RULED OUT with high confidence by code reading. All other mechanisms remain open and unobserved; 0/17 reproduction. Cannot be resolved without capturing a live failing run with its DB preserved. | Not reproducible on demand | OPEN — parked |
| U-9 | Live Anthropic intelligence path (PRD §32 rows 28, 32). | **RESOLVED AS PERMANENTLY UNAVAILABLE**, founder clarification 2026-09-18: no Anthropic credential exists in any verification environment. Status of record: **Implemented / Mock Verified / Live Verification Unavailable.** This is NOT a Beta failure. | CLOSED as far as evidence can go |
| U-10 | PRD §32 row 5: whether `awis stop` actually completes in-flight steps before exiting. No test in the repo asserts it; the only related test is a golden-output format check. start.go:265-284 shares the shutdown ctx with Engine.Run, and nothing demonstrates an in-flight step is allowed to finish rather than being cut off. | Cannot be constructed without editing repo code | Test-coverage gap; needs a new test |
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
| DEC-1 | M15/M16 gates, given the code is already in main via stacking (F-13, F-14). | **Perform G3 + TDS-06 retroactively.** | **DONE.** G3 Q1 (platform boundary/QG-4) = **PASS**, founder ruling 2026-09-18, recorded at cb07eba; brief corrected to name three seams. G3 Q2 (TDS-06 sign-off) = **STILL PENDING**; RECORD_FORMAT.md remains DRAFT. STATE.md stays unreconciled until Q2 signs. |
| DEC-2 | M17 row 8 frozen-surface scope wording vs measured diffs. | **WAIVED — diffs are legitimate.** | DONE (b349ed6). Row 8 recorded WAIVED, not PASS. |
| DEC-3 | M15's two extra tracked files vs the 7-file contract. | **Move both into cards/.** | DONE (24449d9), incl. 5 broken path citations repaired. |
| DEC-4 | M17 row 7 unmeasurable as written. | **Restate as measurable and measure it.** | DONE. PRD §32 audited in full (F-23). Row 7 -> MEASURED WITH EXCEPTIONS. |
| DEC-8 | Live Anthropic verification. | **Not possible — no credential exists.** Record Implemented / Mock Verified / Live Verification Unavailable. Do not fail Beta for unimplemented OpenRouter / Local LLM. | DONE (U-9, F-25) |
| DEC-9 | D-11 remedy mechanism. | **Dedicated UID via SysProcAttr.Credential + runtime DB at 0600.** Landlock rejected on evidence (F-30). | IMPLEMENTED (c99b3c7) and independently verified. Mechanism correct when configured; ineffective by default. |
| DEC-14 | D-11 disposition once verification showed the guarantee is conditional. | **Accept the conditional guarantee and close P0** (founder 2026-09-19). | DONE (3f22284), Director-verified: no unconditional isolation claim remains in §32. This is the same shape as the re-scoping declined on 2026-09-18 — reopened not as preference but because building it proved no unprivileged, portable mechanism exists. |
| DEC-15 | D-6 severity after its second occurrence. | **Investigate with a dedicated load reproduction** (founder 2026-09-19). | DONE. REPRODUCED on the 5th full-suite iteration under 3 concurrent processes, with full diagnostics. Root cause confirmed HIGH. D-6 raised to P1. The decision to keep investigating rather than accept a flake is what found it. |
| DEC-16 | D-6 fix approach. | **Compensating release + claim lease/TTL** (founder 2026-09-19), additive interface, migration 0008. StoragePort stays frozen. | IMPLEMENTING on beta-baseline/d6-orphan-claim. DIRECTOR-IMPOSED SAFETY CONSTRAINT: a claim may be reclaimed ONLY when expired AND carrying no StepStarted event. A bare TTL would let a long-running step's claim be stolen, causing DOUBLE EXECUTION — converting a hang into a correctness bug, strictly worse than the defect. |
| DEC-17 | How to prove the D-6 fix. | **Deterministic fault-injection test PLUS load soak** (founder 2026-09-19). | IMPLEMENTING. The fault-injection test MUST be demonstrated to FAIL against unfixed code — a regression test that passes both before and after proves nothing, and this defect survived a month of green suites. |
| DEC-13 | Three AWIS_PRD.md passages contradict TDS-07 and the shipped code. | **Amend §32 rows 8/15 and §18 together.** | DONE (49987fe). The §18 check found NO code defect — the CLI deliberately implements the contract's form. All three were PRD staleness. |
| DEC-12 | docs-lint rule 3 forbids the literal archive path token in any file under docs/ except three allowlisted "supersession registrars". This ledger is now itself a supersession registrar — the archived documents defer to it — but is not allowlisted, so it cannot describe the archival it records. Add it to the allowlist, or keep rewording around the rule. | OPEN | Adding it amends a frozen EEOS mechanism (scripts/docs-lint.sh). Worked around for now by rewording; no CI impact either way. |
| DEC-10 | D-12 exit-code half: PRD §32 says `workflow validate` exits 1; code exits 3. If docs/CLI_CONTRACT.md (a higher-ranked implementation spec) specifies 3, the PRD is the stale document and the repair is to amend the PRD, NOT the code. | OPEN | Deliberately blocked pending the authority check. Changing shipped CLI behaviour on the PRD's authority alone would invert the evidence hierarchy. |
| DEC-7 | TDS-06 has three specification defects found on review (dead `superseded` enum; `distinguishes` required/optional self-contradiction; ID scheme has no concurrency rule). Fix before signing, or sign and amend later. | OPEN | Blocks G3 Q2, therefore DEC-1, therefore D-2. |
| DEC-5 | M17 row 1 / D-6. | **Fix D-9, re-measure, then close with risk accepted.** | DONE. 25-rep soak clean (1 failure in 49 total `-race` reps). Row 1 -> CLOSED WITH RISK ACCEPTED; D-6 stays open as P2. |
| DEC-11 | D-13 (plugin status PID/calls/latency; metrics rates/percentiles) and D-14 (--watch interval). | **Fix D-12 only; defer D-13; reconcile D-14's spec to the implemented 1s.** | D-13 DEFERRED to post-Beta (needs new storage schema — feature work, not stabilisation). D-14 spec reconcile in flight. |
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

Methodology notes:
- Two of three Investigator git-tracking claims were wrong. Any Investigator
  claim about git tracking state is spot-checked by the Director before it
  enters this ledger.
- A Verifier's docs-lint root cause was incomplete because it ran against the
  working tree rather than a clean checkout. Clean-checkout simulation
  (`git archive HEAD | docs-lint`) is the only valid CI model here.
- TWICE the Director disturbed an agent's environment by operating on the shared
  working tree while an agent was active: once polluting a Verifier's read, once
  causing `git checkout` to carry an Executor's uncommitted work onto another
  branch. EVERY agent touching code MUST be worktree-isolated, and the Director
  MUST NOT switch branches in the shared directory while any agent is running.
- The Director introduced a docs-lint rule 3 violation into this ledger by
  writing the forbidden archive token into it. Self-inflicted, caught by an
  Executor, fixed by rewording. The Director's own edits are not exempt from
  the repository's gates.
- Several Director ledger edits used str.replace WITHOUT asserting a match, so
  they silently no-op'd once surrounding text had changed; four rows (D-1,
  D-6, D-12c, D-14) carried superseded status for a time. Every ledger edit
  now asserts its match count. The state document is not exempt from the
  staleness this program exists to eliminate.
- The Director omitted the AWIS-E1 zero-AI gate from a verification card. The
  Verifier flagged the omission as a card-level finding and ran it anyway
  (PASS). Verification cards MUST carry the standing gates.

## 6. Phase gate status

- Phase 0 — Truth: **COMPLETE**. Maps built, state validated by execution,
  ledger created, U-1 and U-2 closed by measurement.
- Phase 1 — Product health: matrix seeded by F-3..F-10, F-17. Surfaces execute;
  per-feature depth verification not yet performed.
- Phase 2 — Defect elimination: D-1 FIXED on branch and verified green; awaiting
  merge authorization. D-7 fixed. Remaining open: D-6/D-8/D-9, all P2, none
  Beta-blocking.
- Phase 3 — Milestone closure: **M17 CLOSED** (9c477b5), all eight rows
  dispositioned with waivers on record. M15/M16 remain blocked on DEC-1 — the
  G3 boundary verdict passed, but the TDS-06 signature is outstanding.
- Phase 4 — Consolidation: **DONE for D-4** (a650b86, unmerged). D-5 (STATE.md
  staleness) still blocked on DEC-1/TDS-06 signature. Residual: 7 worktree-agent-*
  and 3 origin/claude/audit-* branches carry no unique work and want pruning.
- Phase 5 — Beta Baseline: NOT DECLARED.
  Gate P1=0: **NOT MET** — D-17 only (engine tick aborts on any single
  instance error, starving all other instances and both signal scans).
  D-15 reclassified P2: test-harness deadlock, not an engine defect.
  **BETA BASELINE NOT DECLARED.**
  Gate P0=0: **MET** — D-11 resolved per DEC-14 (mechanism implemented and
  verified; acceptance criteria restated to the guarantee actually delivered;
  residual default-deployment exposure documented, not denied).
  Gate P1=0: D-1 FIXED (unmerged). D-3 RESOLVED — M17 closed at 9c477b5 with all
  eight rows dispositioned. D-2 remains the only open P1, blocked solely on the
  TDS-06 signature.
  Correction of record: earlier ledger revisions stated "no P0 exists". That was
  true of the facts then known and remains true of the facts — D-11's promotion
  is a severity RULING, not a new measurement.
