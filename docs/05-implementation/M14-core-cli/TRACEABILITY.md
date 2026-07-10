# M14 — Traceability
| T | Task | Source coordinates | Card |
|---|---|---|---|
| T1 | TDS-07 `docs/CLI_CONTRACT.md` incl. F-3 interaction model + CONTRA-5 record | F-3 (Verification §15); PRD §15–20, §26; Blueprint §28/§9 | C1 |
| T2 | `cmd/awis` skeleton: mux, global flags, error renderer (what/where/what-now), exit codes, `version` | PRD §26; IMP §16 L305 | C1 |
| T3 | `start` (discovery, header incl. intelligence level, PID file, graceful signals) | FR-RM-02/03/06; IMP §27.M14 Obj | C2 |
| T4 | `stop` (PID + SIGTERM + wait) | FR-RM-04; F-3 convention | C2 |
| T5 | `status` (+instance detail, `--watch`) | FR-RM-*; PRD §20 | C2 |
| T6 | System test: real binary lifecycle incl. crash-recovery (SIGKILL → restart → consistent) | IMP §27.M14 Val; IR-5 | C2 |
| T7 | `submit` (--input/--wait), `signal`, `cancel` (--compensate) | PRD §19; FR-SG/FR-EX rows | C3 |
| T8 | `trace` (--json/--full) | PRD §19/§20; QG-2 shape | C3 |
| T9 | `workflow validate|list|show` (FR-WD-15/16/17 CLI half) | PRD §18 | C3 |
| T10 | Golden-output tests all commands, human + JSON (PP-6) | IMP §27.M14 Out | C1–C3 |
| T11 | `plugin install|list` minimal (F-5) | F-5 (Verification §15); FR-PS-07/08 subset | C4 |
| T12 | `WorkflowRegistered` audit write site | F-4 write-site set; NFR-S-05 | C2 |
| T13 | `docs/CLI.md` shipped-command reference | IMP §27.M14 DoD | C4 |

## Notes / dispositions (CE, A-INIT 2026-07-10)
- **CONTRA-5 (F-3 resolution):** Blueprint §9 "no concurrent writers" vs §28 socket-or-SDK —
  refined (not resolved silently): §9 constrains to one running ENGINE; CLI control-plane
  writes ride WAL+busy-timeout through the same storage APIs. Full text in SPEC + TDS-07.
- **No cobra:** stdlib flag mux; dependency policy has no CLI framework approval.
- **Audit ConfigChanged** write site deferred to M17 with the config commands (no config
  mutation surface exists in M14).
- **`status --watch`** is a re-poll loop, not a TUI (PRD §20 scope for V1 CLI).

## Execution record (appended during B-BUILD/C-VERIFY)
- C1 (709ce0e, 2026-07-10): TDS-07 authored (F-3/CONTRA-5 verbatim; 9 command contracts; M17
  tree marked; taxonomy template); cmd/awis skeleton (mux, cliErr, OpenStorage, version cmd,
  golden harness). WAL busy_timeout=5000 confirmed already set by storage layer. 5/5 tests;
  all gates green. No deviations.
- C2 (73435f7) / C3 (add54dd), 2026-07-10: lifecycle cmds (start/stop/status incl. --watch,
  PID/SIGTERM, WorkflowRegistered audit) + system tests w/ crash recovery; execution cmds
  (submit/signal/cancel/trace/workflow validate|list|show) + goldens + F-3 second-process
  proof. cmd/awis suite green 22.6s. Housekeeping: stray built binary `awis` (committed by
  accident in the C1 ledger commit) removed + gitignored.
