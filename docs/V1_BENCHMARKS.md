# V1 Benchmarks

**Status:** M18-C4 final compilation. NFR-P-01, 02, 04..09 are measured (PASS, 3 of 3 final runs). NFR-P-03 is UNMEASURABLE. QG-1 PASS (Linux measured; macOS CI leg cited). QG-2 is PENDING FOUNDER MEASUREMENT (not passed). QG-3, QG-4, QG-5 re-confirmed at the final state. PRD §32 re-measured at the final state (drift note below).

**Provenance:** Code state = branch `m18-hardening-release`, base commit `c614bac` plus the M18-C4 diff recorded by the commit that adds this file (prepared-statement cache in `internal/storage`; buffered human output in `cmd/awis/trace.go`; their tests and goldens). Benchmarks: `make bench` (`go test -p 1 -run '^$' -bench BenchmarkNFRP -benchtime=1x -count=1 -timeout 30m ./internal/engine/ ./cmd/awis/`), three runs at that state, 2026-10-11. Host: AMD Ryzen 7 7730U, Linux 7.2.4-arch1-2; go1.26.5; Python 3.14.7; git 2.55.0. `TMPDIR` is on the NVMe volume outside the repo (`/tmp` is tmpfs and was not used), so every store is an on-disk WAL store. Each Measured cell lists the three runs in order (run 1 / run 2 / run 3), copied from their result lines. Verdicts use `internal/nfrbench`: a bound stated without a percentile is judged on the maximum sample.

## NFR-P-01 — Execution loop tick latency (no I/O)

| Target | Measured (run 1 / run 2 / run 3) | Status | Fixture |
|---|---|---|---|
| < 5ms P99 (binding) | P99 4.493 / 4.601 / 4.862ms (P50 3.722 / 3.803 / 3.813ms; P95 4.304 / 4.385 / 4.455ms; max 5.540 / 4.805 / 5.153ms; N=1000) | PASS (3 of 3; run 3 margin 0.14ms) | 100 running instances parked in retry backoff (0 runnable steps, 0 dispatches and 0 EventLog writes during sampling); on-disk WAL store; 1000 Tick calls |

## NFR-P-02 — Native step dispatch latency (Go handler, no AI)

| Target | Measured (run 1 / run 2 / run 3) | Status | Fixture |
|---|---|---|---|
| ≤ 50ms P95 — ship gate (binding) | P95 1.566 / 1.313 / 1.342ms (P50 0.955 / 0.956 / 0.967ms; P99 15.730 / 8.313 / 8.290ms; max 22.449 / 15.903 / 14.510ms; N=500) | PASS (3 of 3) | 1 native step per instance, no-op Go handler; timed unit = one Tick that claims, dispatches and settles it; 500 instances; on-disk WAL store |
| < 10ms P95 — engineering target (informational) | P95 1.566 / 1.313 / 1.342ms (same run) | PASS (informational; cannot fail NFR-P-02) | same fixture |

**CONTRA-2 disposition:** NFR-P-02 is dual. The 50ms P95 ship gate is binding and PASSes in all three runs (P95 1.313–1.566ms). The 10ms P95 engineering target is informational and also reads PASS (1.313–1.566ms). Both numbers come from the same C1 harness.

## NFR-P-03 — Intelligence step latency (Anthropic Haiku, draft)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| < 5s P95 | UNMEASURABLE | UNMEASURABLE | Live Anthropic API. No Anthropic credential exists in any verification environment (CURRENT_TRUTH_LEDGER.md §7.6; same disposition as PRD §32 rows 28 and 32). Not run. |

## NFR-P-04 — Signal delivery latency

| Target | Measured (run 1 / run 2 / run 3) | Status | Fixture |
|---|---|---|---|
| < 200ms, max | max 121.565 / 108.266 / 100.716ms (P50 98.558 / 98.447 / 98.467ms; P95 99.501 / 99.323 / 99.581ms; N=100) | PASS (3 of 3) | 1 signal wait step per instance; runtime loop at 100ms tick (awis start default); latency = awis signal start to StepCompleted of the signal step (EventLog); 100 signals; on-disk WAL store |

## NFR-P-05 — `awis trace` query time (100K events)

| Target | Measured (run 1 / run 2 / run 3) | Status | Fixture |
|---|---|---|---|
| < 500ms, max — multi-instance shape (binding) | max 1.953 / 1.906 / 2.211ms (P50 1.421 / 1.315 / 1.324ms; P95 1.880 / 1.781 / 1.913ms; N=20) | PASS (3 of 3) | 100,000 events across 500 instances; traced instance holds 200 events; human output to /dev/null; on-disk WAL store |
| < 500ms, max — single-instance shape (binding; both shapes are binding per the CE ruling in STATE.md) | max 459.455 / 459.915 / 460.816ms (P50 454.516 / 450.565 / 451.740ms; P95 459.455 / 459.915 / 460.816ms; N=10) | PASS (3 of 3) | 100,000 events in one instance; traced instance holds 100,000 events; human output to /dev/null; on-disk WAL store |

## NFR-P-06 — `awis rebuild-state` (100K events)

| Target | Measured (run 1 / run 2 / run 3) | Status | Fixture |
|---|---|---|---|
| < 30s, max | max 406.031 / 408.427 / 664.191ms (P50 402.449 / 405.304 / 401.836ms; N=5) | PASS (3 of 3; run 3 max is an outlier, see note) | 100,000 events across 500 instances; each sample rebuilds the whole projection; 5 samples; on-disk WAL store |

Note: run 3 recorded one 664ms sample against a 402–408ms range in runs 1 and 2. The cause was not isolated; it is far inside the 30s bound and was not re-run.

## NFR-P-07 — Workflow instance startup latency (`awis submit` to first step started)

| Target | Measured (run 1 / run 2 / run 3) | Status | Fixture |
|---|---|---|---|
| < 300ms, max | max 102.007 / 100.948 / 102.403ms (P50 99.306 / 99.391 / 99.311ms; P95 100.514 / 100.090 / 100.247ms; N=100) | PASS (3 of 3) | 1 native step (no-op Go handler) per instance; runtime loop at 100ms tick (awis start default); latency = awis submit start to StepStarted of the first step (EventLog); 100 submits; on-disk WAL store |

## NFR-P-08 — Plugin spawn time (Python, cold start)

| Target | Measured (run 1 / run 2 / run 3) | Status | Fixture |
|---|---|---|---|
| < 3s, max | max 40.560 / 40.489 / 41.262ms (P50 37.853 / 38.259 / 38.023ms; P95 38.473 / 39.524 / 39.964ms; N=20 cold starts) | PASS (3 of 3) | plugins/git-context-plugin shipped awis-plugin.yaml (`python3 -m git_context_plugin`; Python 3.14.7; git 2.55.0); cold start = fresh Manager and fresh plugin process per sample; timed = first Manager.Call (lazy spawn + handshake; 15s handshake timeout) with empty inputs, which the handler rejects before any git subprocess (handler_error), so the sample is an upper bound on spawn; on-disk store; plugins_user unset (D-11 default) |

## NFR-P-09 — Plugin capability call latency (excluding handler logic)

| Target | Measured (run 1 / run 2 / run 3) | Status | Fixture |
|---|---|---|---|
| < 50ms, max — binding: successful `git.diff.fetch` on a warm process (includes the handler's git subprocess work, so a conservative upper bound on the handler-excluded figure) | max 5.468 / 7.001 / 5.290ms (P50 4.950 / 5.116 / 4.917ms; P95 5.231 / 5.761 / 5.185ms; P99 5.439 / 6.113 / 5.236ms; N=100) | PASS (3 of 3) | plugins/git-context-plugin shipped manifest; warm process (one untimed warm-up call; PID unchanged across samples); tiny temp git repo, 2 commits, 1-line diff; `git.diff.fetch` HEAD~1..HEAD with output checked for the added line; on-disk store; plugins_user unset (D-11 default) |
| Informational floor — immediate-reject call (empty inputs; handler error before any git subprocess); no verdict | max 0.742 / 0.809 / 0.703ms (P50 0.227 / 0.235 / 0.229ms; P95 0.291 / 0.313 / 0.274ms; P99 0.359 / 0.386 / 0.374ms; N=100) | INFORMATIONAL | same warm process and store; each sample is one Manager.Call with empty inputs |

## V1 quality gates (PRD §31)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| QG-1: `awis init` → `awis start` → `awis submit hello-world` → `awis trace <id>` succeeds on a completing instance; ≤ 5 minutes | Linux: wall-clock 247ms (init to trace), instance status `completed`, from `TestQG1RehearsalCompletesHelloWorld` at the final state. macOS: real GitHub Actions run 38146475809 at commit bbdcfa5, `macos-latest` Verify → Test step: `ok github.com/awis/awis/cmd/awis 36.852s`, which includes `TestQG1RehearsalCompletesHelloWorld` (no skip, build-tag or GOOS gate in that test). | PASS on Linux (final state) and macOS (CI at bbdcfa5). The C4 commit's own macOS CI run is pending the branch push and is not claimed here. | Shipped `awis` binary built from ./cmd/awis with no extra handlers and no source changes; fresh temp project; hello-world reaches `completed`; trace shows greet and log as StepCompleted and WorkflowCompleted |
| QG-2: `awis trace` output is parsed correctly by a developer who has not read AWIS documentation | Not measured by any agent | PENDING FOUNDER MEASUREMENT (human-subject checklist row; not passed; not a card deliverable) | — |
| QG-3: all workflows complete with NullAdapter as the sole intelligence provider | Re-confirmed at the final state: `make e1`, 8 of 8 `TestE1_*` PASS. CI citation: run 38146475809 at bbdcfa5, "Zero-AI gate (E1)" step success on `ubuntu-latest` and `macos-latest`. | PASS (re-confirmed) | `internal/engine/e1_test.go`; `make e1` |
| QG-4: OIP capture-decision and recall-decision workflows run on AWIS without platform changes | Boundary artifact: `docs/05-implementation/M15-oip-on-awis/cards/BOUNDARY_EVIDENCE.md` (M15 G3 verdict PASS, cb07eba). Fresh `git log --stat` check at the final HEAD over the 11 M15-subject commits: 9 platform paths outside `apps/oip/` and `docs/`, all in the disclosed seams (P0 c3fba29: `sdk/dsl.go`, `sdk/dsl_test.go`; P1 0ef33c8: `sdk/runtime.go`, `sdk/runtime_runner.go`; C3r 0e452f8: `cmd/awis/errors.go`, `cmd/awis/main.go`, `cmd/awis/submit.go`, `internal/engine/emit.go`, `internal/storage/rebuild.go`, `sdk/runtime_runner.go`). Fresh OIP run at the final state: `TestOIPSystemE2E` and 8 `TestQG3_*` PASS. | PASS as disclosed. Literal "zero platform-module diffs" is not what the M15 range shows (nine disclosed platform paths): CE RULING REQUIRED. See note. | `apps/oip` tests; `git log --stat` over the M15 subject commits |
| QG-5 (PRD §31): `WorkflowTestHarness` workflow including WAIT/signal completes in under 1s | Re-confirmed at the final state: `TestQG5_WaitSignalWithin1s` PASS (0.01s); `TestCheckpointM9_OIPCaptureWorkflow` PASS (0.01s). | PASS (re-confirmed) | `sdk/testing/qg5_test.go`; `sdk/testing/checkpoint_m9_test.go`; in-memory SQLite |

Note on QG-4: the M15 G3 brief (`cards/G3_BRIEF.md`) states "zero changes under `internal/`". The C3r commit changed `internal/engine/emit.go` and `internal/storage/rebuild.go`, which BOUNDARY_EVIDENCE.md discloses. The brief is out of date on that point; the disclosure is the authority.

## Pipeline rows

| Target | Measured | Status | Fixture |
|---|---|---|---|
| C3: `make release-dry` — 4-platform static binaries, checksums, release notes | Recorded by M18-C3 (e15497b), with the C3r and C3r2 revisions (99f7b13, 34894e7), per STATE.md. Not re-run by C4. | DONE (M18-C3; not re-measured here) | — |
| C4: final compile, QG-3/4/5 reconfirmation, PRD §32 re-measure, fix-only items for the NFR-P-01 and NFR-P-05 single-instance misses | This document. Both C1 misses fixed in C4 (see Fix record); all three final runs PASS. | DONE (M18-C4) | — |

## Fix record (M18-C4)

- NFR-P-01: `GetInstance` and `CancellationRequested` (the two per-instance reads on the engine tick path) reuse prepared statements (`internal/storage/stmt_cache.go`). The statement cache is owned by `DB`, is used only for fixed SQL, and is closed by `DB.Close`. Results and error text are unchanged (`internal/storage/stmt_cache_test.go`).
- NFR-P-05 (single-instance shape): the human timeline of `awis trace` is written through one buffered writer (`cmd/awis/trace.go`, `traceWriter`). A failed write is reported with exit 1, as `emitJSON` does for a failed encode. Output bytes are unchanged: `cmd/awis/trace_golden_test.go` and the goldens in `cmd/awis/testdata/golden/trace_golden_*` pin the human, `--full` and `--json` output.
- No engine, core, sdk or StoragePort interface changed. No benchmark, fixture size or threshold changed.

## PRD §32 drift note (re-measured at the final state)

Baseline: M17 DEC-4 (measured at 9c477b5): 49 SATISFIED; 6 NOT SATISFIED (rows 8, 15, 37, 38, 40, 53); 1 PARTIAL (row 44); 3 UNMEASURABLE (rows 5, 28, 32). Forty-five commits landed after 9c477b5 (engine hardening, step-claim lease migration 0008, plugin isolation, PRD corrections). Every row was re-checked at the final state rather than carried forward. Evidence: 108 row-mapped tests, all part of `make test`, which PASSes for every package including `apps/oip`; direct CLI runs for rows 1, 2, 14–17, 39, 44 and 53; code and golden inspection for rows 37, 40 and 41; NFR bench rows 56–59 from the three final runs.

Rows whose status changed:
- Row 8: NOT SATISFIED → SATISFIED. The PRD passage was corrected to the CLI contract (commit 49987fe, D-12). `TestParseInputFlags` PASS.
- Row 15: NOT SATISFIED → SATISFIED. Same correction (49987fe). Direct check: invalid YAML exits 3 and shows the path and the parser error.
- Row 38: NOT SATISFIED → SATISFIED under the restated text (commit 3f22284, DEC-14 per D-11; mechanism c99b3c7). Tests: `TestManager_EnvIsolation`, `TestManager_UnisolatedWarning_FiresOnce`, `TestSpawnPlugin_*` PASS. The default-deployment residual is accepted in the ledger.
- Row 53: NOT SATISFIED → SATISFIED under the restated text (3f22284, c99b3c7). `runtime.db` mode checked directly (600); `TestOpen_DatabaseFileMode0600` PASS.

Rows whose status did not change: 37 (NOT SATISFIED: `PluginRow` has no PID, call-count or latency fields, D-13 deferred by DEC-11); 40 (NOT SATISFIED as written: `status --watch` re-polls every 1s per DEC-11, the PRD still says 5s); 44 (PARTIAL: `awis metrics` shows counts and avg/min/max only, D-13); 5 (UNMEASURABLE: no test asserts in-flight completion on stop, U-10); 28 and 32 (UNMEASURABLE: no Anthropic credential; mock-verified only).

Regressions (SATISFIED → NOT SATISFIED): none found.

FLAG, row 41 (counted SATISFIED per M17; CE ruling required): the human `awis trace` timeline does not print adapter or tokens for StepCompleted events. TDS-07 §4 (docs/CLI_CONTRACT.md lines 494 and 951) and PRD row 41 both show them. The JSON output carries them in the payload. This was already true at M17 and is not changed by C4, because a fix would change output bytes and is outside the fix-only scope. If CE rules row 41 not satisfied, the count becomes 52 SATISFIED, 3 NOT SATISFIED, 1 PARTIAL, 3 UNMEASURABLE.

Totals at the final state (row 41 as counted): 53 SATISFIED (49 unchanged + rows 8, 15, 38, 53), 2 NOT SATISFIED (37, 40), 1 PARTIAL (44), 3 UNMEASURABLE (5, 28, 32). Sum 59.
