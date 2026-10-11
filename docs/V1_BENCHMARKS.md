# V1 Benchmarks

**Status:** M18-C2r. NFR-P-01, 02, 04..09 and the QG-1 rehearsal carry measured numbers from the run below; NFR-P-03 is UNMEASURABLE. M18-C4 compiles the final table (C1+C2 numbers, CONTRA-2 disposition, QG-3/4/5 reconfirmation, PRD §32 re-measure). TBD marks a value no run has produced yet.

**Provenance:** `make bench` (`go test -p 1 -run '^$' -bench BenchmarkNFRP -benchtime=1x ./internal/engine/ ./cmd/awis/`) on branch `m18-hardening-release`, base commit `2a0082b` plus the M18-C2r changes; run 2026-10-11. Host: AMD Ryzen 7 7730U, Linux; go1.26.5; Python 3.14.7; git 2.55.0; on-disk WAL store unless a row says otherwise. Verdicts use `internal/nfrbench`: a bound stated without a percentile is judged on the maximum sample. Every Measured cell is copied from that run's result line.

## NFR-P-01 — Execution loop tick latency (no I/O)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| < 5ms P99 (binding) | P99 8.114ms (P50 7.086ms, P95 7.848ms, max 8.540ms; N=1000) | MISS | 100 running instances parked in retry backoff (0 runnable steps, 0 dispatches and 0 EventLog writes during sampling); on-disk WAL store; 1000 Tick calls |

## NFR-P-02 — Native step dispatch latency (Go handler, no AI)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| ≤ 50ms P95 — ship gate (binding) | P95 1.226ms (P50 0.994ms, P99 1.482ms, max 1.867ms; N=500) | PASS | 1 native step per instance, no-op Go handler; timed unit = one Tick that claims, dispatches and settles it; 500 instances; on-disk WAL store |
| < 10ms P95 — engineering target (informational) | P95 1.226ms | PASS (informational; cannot fail NFR-P-02) | same fixture |
| CONTRA-2 callout | Dual threshold recorded per CONTRA-2: the 50ms P95 ship gate is binding, the 10ms P95 engineering target is informational. Both read PASS in this run. | — | — |

## NFR-P-03 — Intelligence step latency (Anthropic Haiku, draft)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| < 5s P95 | UNMEASURABLE | UNMEASURABLE | Live Anthropic API. No credential exists in any verification environment (M18 IMPLEMENTATION_SPEC.md, Non-scope; CURRENT_TRUTH_LEDGER.md §7.6). Not run. |

## NFR-P-04 — Signal delivery latency

| Target | Measured | Status | Fixture |
|---|---|---|---|
| < 200ms, max | max 99.809ms (P50 98.440ms, P95 99.214ms; N=100) | PASS | 1 signal wait step per instance; runtime loop at 100ms tick (awis start default); latency = awis signal start to StepCompleted of the signal step (EventLog); 100 signals; on-disk WAL store |

## NFR-P-05 — `awis trace` query time (100K events)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| < 500ms, max — multi-instance shape (binding) | max 2.028ms (P50 1.429ms, P95 2.005ms; N=20) | PASS | 100,000 events across 500 instances; traced instance holds 200 events; human output to /dev/null; on-disk WAL store |
| < 500ms, max — single-instance shape (binding; both shapes are binding per the CE ruling in STATE.md) | max 510.022ms (P50 501.198ms, P95 510.022ms; N=10) | MISS | 100,000 events in one instance; traced instance holds 100,000 events; human output to /dev/null; on-disk WAL store |

## NFR-P-06 — `awis rebuild-state` (100K events)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| < 30s, max | max 449.645ms (P50 403.465ms, P95 449.645ms; N=5) | PASS | 100,000 events across 500 instances; each sample rebuilds the whole projection; 5 samples; on-disk WAL store |

## NFR-P-07 — Workflow instance startup latency (`awis submit` to first step started)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| < 300ms, max | max 100.666ms (P50 99.341ms, P95 100.306ms; N=100) | PASS | 1 native step (no-op Go handler) per instance; runtime loop at 100ms tick (awis start default); latency = awis submit start to StepStarted of the first step (EventLog); 100 submits; on-disk WAL store |

## NFR-P-08 — Plugin spawn time (Python, cold start)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| < 3s, max | max 38.984ms (P50 37.905ms, P95 38.703ms; N=20 cold starts) | PASS | plugins/git-context-plugin shipped awis-plugin.yaml (`python3 -m git_context_plugin`; Python 3.14.7; git 2.55.0); cold start = fresh Manager and fresh plugin process per sample; timed = first Manager.Call (lazy spawn + handshake; 15s handshake timeout) with empty inputs, which the handler rejects before any git subprocess (handler_error), so the sample is an upper bound on spawn; on-disk store; plugins_user unset (D-11 default) |

## NFR-P-09 — Plugin capability call latency (excluding handler logic)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| < 50ms, max — binding: successful `git.diff.fetch` on a warm process (includes the handler's git subprocess work, so a conservative upper bound on the handler-excluded figure) | max 5.721ms (P50 4.994ms, P95 5.272ms, P99 5.657ms; N=100) | PASS | plugins/git-context-plugin shipped manifest; warm process (one untimed warm-up call; PID unchanged across samples); tiny temp git repo, 2 commits, 1-line diff; `git.diff.fetch` HEAD~1..HEAD with output checked for the added line; on-disk store; plugins_user unset (D-11 default) |
| Informational floor — immediate-reject call (empty inputs; handler error before any git subprocess); no verdict | max 0.929ms (P50 0.247ms, P95 0.320ms, P99 0.424ms; N=100) | INFORMATIONAL | same warm process and store; each sample is one Manager.Call with empty inputs |

## V1 quality gates (PRD §31)

| Target | Measured | Status | Fixture |
|---|---|---|---|
| QG-1: `awis init` → `awis start` → `awis submit hello-world` → `awis trace <id>` succeeds on a completing instance; ≤ 5 minutes | wall-clock 248ms (init to trace); instance status `completed` | PASS on Linux (this run); macOS leg TBD (CI evidence, M18-C4) | Shipped `awis` binary built from ./cmd/awis with no extra handlers and no source changes; fresh temp project; hello-world reaches `completed`; trace shows greet and log as StepCompleted and WorkflowCompleted; `TestQG1RehearsalCompletesHelloWorld` |
| QG-2: `awis trace` output is parsed correctly by a developer who has not read AWIS documentation | TBD | TBD (founder-measured checklist row; human-subject, not a card deliverable) | — |
| QG-3: all workflows complete with NullAdapter as the sole intelligence provider | TBD | TBD (re-confirm at M18-C4) | — |
| QG-4: OIP capture-decision and recall-decision workflows run on AWIS without platform changes | TBD | TBD (re-confirm at M18-C4) | — |
| QG-5 (PRD §31) | TBD | TBD (re-confirm at M18-C4) | — |

## Pipeline rows

| Target | Measured | Status | Fixture |
|---|---|---|---|
| C3: `make release-dry` — 4-platform static binaries, checksums, release notes | TBD | TBD (M18-C3) | — |
| C4: final compile, QG-3/4/5 reconfirmation, PRD §32 re-measure, fix-only items for the NFR-P-01 and NFR-P-05 single-instance MISSes | TBD | TBD (M18-C4) | — |
