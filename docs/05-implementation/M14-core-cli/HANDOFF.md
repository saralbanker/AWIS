# M14 → M15/M17/M18 Handoff
**Status: COMPLETE — D-CLOSE passed 2026-07-10.**

## Guaranteed outputs (contract — to be confirmed as actuals)
- `docs/CLI_CONTRACT.md` (TDS-07) incl. the F-3 interaction-model decision (CONTRA-5).
- `cmd/awis`: version/start/stop/status/submit/signal/cancel/trace/workflow-validate|list|show
  + minimal plugin install|list (F-5); --json everywhere; shared what/where/what-now error
  renderer; golden tests; real-binary system test with crash recovery.
- Conventions: `--data-dir` (default ./.awis/), PID file, SIGTERM stop, exit codes 0/1/2/3.
- `docs/CLI.md` reference (DoD).

## What M15 may assume (drafted; confirm at completion)
- The full G3 dev loop works from a terminal: `awis start` (auto-discovers OIP YAMLs) →
  `plugin install plugins/git-context-plugin` → `submit capture-decision --input …` →
  `status/trace/signal`. No platform file edits needed to operate OIP.

## What M17 may assume (drafted; confirm at completion)
- Adding a command = one file in cmd/awis + mux row + TDS-07 section + goldens (the
  mechanical-batch pattern); error renderer and --json plumbing inherited.

## What M18 may assume (drafted; confirm at completion)
- QG-1/QG-2 measure these shipped formats; NFR-P-04 signal latency rides the 100ms tick.

## Known limitations (drafted)
- `--watch` is re-poll, not TUI. `plugin install` registers local path (no dep/venv
  automation — M17/PR-5 docs). ConfigChanged audit site deferred to M17 config commands.

## Actuals (filled at completion)
- C1: 709ce0e · C2: 73435f7 · C3: add54dd · C4: 51c2575 · C3r: 234a935
- V1 verification: 21/22 PASS at 08982be; row 9 (JSON goldens) ❌ → revision card C3r closed
  it (13/13 goldens; CE targeted re-verification green). V1 notes: --json global-flag synopsis
  ambiguity fixed in C3r; stop-test log line is a harness pipe artifact, not a defect.
- Deviations: stray built binary removed+gitignored (housekeeping).
