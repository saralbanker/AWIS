# M14 — Implementation Spec
**Canonical sources:** IMP §27.M14; IMP §16 (CLI rollout — nine dev-loop commands, --json from
day one, PP-6 no-retrofit rule); TDS-07 `docs/CLI_CONTRACT.md` (written day 1 per **F-3** —
output formats + --json schemas + error taxonomy PLUS the CLI↔runtime interaction model);
PRD §15–20 (command tree/navigation), §26 (error taxonomy what/where/what-now), §27, §32
(acceptance checklists — Runtime-Management + Execution + Validation rows not requiring
init/plugins-full); Blueprint §28 (Mode 1 socket-or-SDK text), §9 (local-mode concurrency
text); **F-5** (minimal `plugin install|list` moved here from M17); IR-5; FR-RM-*; NFR-P-04.

## F-3 DECISION (CE, Fable, 2026-07-10 — record CONTRA-5)
**Interaction model: direct SQLite access under WAL with busy-timeout.** The CLI opens the
SAME runtime.db via `sdk.SQLiteStorage` and uses the storage/sdk APIs the engine already
consumes on its 100ms pull tick: `submit` inserts the instance/trigger intake rows exactly as
an embedded app would; `signal` uses the M07 signal path; `cancel` the M06 cancellation-request
path; reads (`status/trace/workflow list|show`) are plain queries. No socket, no RPC surface —
the pull-based architecture's own grain; a second mechanism would be new architecture (Gall).
`stop` is the exception: **PID-file + SIGTERM** convention (`<data-dir>/awis.pid`, SIGTERM →
engine ctx cancel → graceful drain; consistent with FR-RM-06). Latency: engine tick default
100ms ⇒ NFR-P-04 (≤200ms signal) attainable; M18 benchmarks measure it.
**CONTRA-5 (recorded, never silently resolved):** Blueprint §9 says local mode has "no
concurrent writers" while §28 Mode 1 sanctions socket-or-SDK for a second process. TDS-07
documents the refinement: the §9 phrase constrains to a SINGLE RUNNING ENGINE instance (one
puller); control-plane row writes from a CLI process under WAL+busy-timeout are serialized by
SQLite and do not constitute a second writer in the §9 sense. Both candidate mechanisms are
Tier-0-named; this chooses (b) per the F-3 amendment's own terms.

## Conventions (CE pins)
- Module layout: `cmd/awis/` (package main) + subcommand files; stdlib `flag` with a tiny
  command mux (NO cobra — dependency policy; IMP has no CLI-framework approval).
- Data dir: `--data-dir` global flag, default `./.awis/` (runtime.db, awis.pid, logs); PRD
  §25 local-first.
- Every command supports `--json` (PP-6). Human output = the PRD §19/§20 formats; JSON =
  stable field names documented in TDS-07.
- Errors: shared renderer `what/where/what-now` (PRD §26) — one helper `cliErr(what, where,
  whatNow)`; exit codes: 0 ok, 1 operational error, 2 usage error, 3 validation failure.
- `awis start`: foreground; auto-discovery `dsl.Discover(cwd)` → ParseFile → RegisterWorkflow;
  registers plugins from `plugins/*/awis-plugin.yaml` if present (F-5 symmetry, optional);
  startup header: version, db path, workflows registered, plugins registered, intelligence
  level (NullAdapter ⇒ "none (zero-AI mode)"; PRD §21/FR-IL); writes PID file; SIGTERM/SIGINT
  → graceful stop; tick 100ms default.
- Registration audit: `WorkflowRegistered` audit row on each registration when storage
  supports audit (F-4 write-site set: registration here, SignalDelivered at M07, Plugin at
  M12; ConfigChanged arrives with config cmds at M17).
- Golden-output tests: `cmd/awis/testdata/golden/*.txt|json`; system test runs the REAL built
  binary (os/exec) through submit→status→trace→signal loop + crash-recovery (kill -9 the
  runtime; restart; rebuild-state consistency is engine-guaranteed; IR-5/IMP Val row).

## 1. TDS-07 + skeleton (M14-C1)
`docs/CLI_CONTRACT.md`: interaction model (verbatim from the F-3 decision above incl.
CONTRA-5); command tree (the M14 nine + M17 remainder marked "M17"); per-command synopsis,
human format, --json schema, exit codes; error taxonomy template (PRD §26 verbatim shape);
PID-file/SIGTERM convention; data-dir convention. Plus `cmd/awis/`: main.go (mux, global
flags --data-dir/--json, version cmd), errors.go (cliErr renderer + exit codes), storage
open helper (WAL busy-timeout via sdk.SQLiteStorage), golden test harness helper. `version`
command working end-to-end (golden test) proves the skeleton.

## 2. Lifecycle commands (M14-C2)
`start` (per pins; incl. discovery + header + PID + signals), `stop` (read PID, SIGTERM, wait
w/ timeout, report), `status` (instance table; `--watch` = 1s re-poll), instance status detail
(`status <instance-id>`). System test (build real binary): start in temp dir with a workflow
file, submit via a second CLI process, status shows it, stop graceful; crash-recovery: SIGKILL
runtime, restart, state consistent (instance resumes/completes or reports accurately).

## 3. Execution + validation commands (M14-C3)
`submit <workflow> [--input k=v|--input @file.json] [--wait]` (prints instance id; --wait
polls to terminal status, exit reflects outcome); `signal <instance-id> <name> [--payload
@file|k=v]`; `cancel <instance-id> [--compensate]`; `trace <instance-id> [--json] [--full]`
(event timeline from ReadEvents; PRD §19/§20 shape; QG-2 target readability); `workflow
validate <file>` (= dsl.ValidateFile + PRD §18 render; exit 3 on invalid); `workflow list`
(registered defs from storage); `workflow show <id>` (definition summary). Golden tests for
every human + JSON format (deterministic via fixed clock/id storage fixtures where needed).

## 4. F-5 plugin minimal + docs (M14-C4)
`plugin install <path>` (= plugin manifest parse + Manager-style registration via PluginStore
+ audit PluginRegistered; validates manifest; copies nothing — registers path in place, V1
local-path semantics) ; `plugin list` (PluginStore.ListPlugins table + --json). `docs/CLI.md`
(DoD): shipped-command reference generated from TDS-07 content (hand-written, cites TDS-07).
PRD §32 checklist rows (runtime-mgmt/execution/validation subsets) transcribed into the
module VALIDATION_CHECKLIST as binary evidence rows... (already done at A-INIT — C4 just
ensures `docs/CLI.md` matches behavior).

## Non-scope
- M17 commands: init, history, logs, metrics, recall, replay, audit, plugin status/remove,
  config *, rebuild-state, export, prune-events. (Tree documented in TDS-07, marked M17.)
- No engine/storage/sdk surface changes; no new migrations (audit landed M07); no cobra or
  any new Go dependency; no TUI (--watch is re-print, not a TUI).
- QG-1/QG-2 measurement is M18; M14 only keeps formats within TDS-07.
