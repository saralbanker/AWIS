# M14 — Validation Checklist (binary; exit list)
Global DoD (IMP §24) + M14 rows (IMP §27.M14; PRD §32 runtime-mgmt/execution/validation
subsets; F-3/F-5). Verifier executes via cards/M14-V1.md.

- [x] V-COMMON block all ✅ (`make build test lint race` + `make e1`; clean tree)
- [x] `make pytest` green (three-suite regression)
- [x] `docs/CLI_CONTRACT.md` (TDS-07) exists: F-3 interaction model + CONTRA-5 verbatim from
      SPEC; full command tree (M17 rows marked); per-command human + --json formats; error
      taxonomy template; PID/SIGTERM + data-dir conventions
- [x] `go build ./cmd/awis` produces a working binary; `awis version` human + --json golden
- [x] Every shipped command supports `--json` (PP-6) — golden evidence per command
- [x] Errors render what/where/what-now via the shared helper; exit codes 0/1/2/3 as
      documented — test evidence
- [x] `awis start`: discovers ./workflows/*.yaml, registers them, prints startup header incl.
      intelligence level line, writes PID file, SIGTERM graceful — system-test evidence
- [x] `awis stop`: PID+SIGTERM path stops a running instance gracefully
- [x] `status` lists instances; `status <id>` details; `--watch` re-polls
- [x] `submit` (--input k=v and @file, --wait exit-code semantics), `signal`, `cancel`
      (--compensate) all function against a live runtime from a SECOND process (F-3 model
      proven)
- [x] `trace` human format matches TDS-07 (QG-2 shape); `--json` schema matches; `--full`
      includes payloads
- [x] `workflow validate` = PRD §18 output, exit 3 on invalid; `workflow list`; `workflow
      show` — goldens
- [x] System test: real binary crash-recovery (SIGKILL runtime → restart → state consistent)
- [x] `WorkflowRegistered` audit row written on registration (F-4) — evidence
- [x] `plugin install <path>` registers manifest + PluginRegistered audit; `plugin list`
      shows it (F-5) — evidence
- [x] `docs/CLI.md` documents exactly the shipped commands (DoD)
- [x] No new Go dependency (stdlib flag mux; go.mod diff empty); no changes to
      engine/storage/sdk/dsl/plugin packages (diff proof); no existing test modified
- [x] PRD §32 checklist rows for runtime-mgmt/execution/validation (excluding init/plugins-
      full/M17 items) each map to a passing test or system-test step — evidence table
