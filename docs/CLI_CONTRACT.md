# TDS-07 — CLI Contract
**Document:** `docs/CLI_CONTRACT.md`
**Status:** Canonical — Implementation-Ready
**Milestone:** M14-C1 (skeleton + version); C2 (lifecycle); C3 (execution + validation); C4 (plugin minimal)
**Authority:** AWIS_PRD.md §15–20, §26; IMPLEMENTATION_SPEC.md F-3; IMP §16
**Date:** 2026-07-10

---

## §1 Interaction Model (F-3 DECISION — verbatim, CE + Fable, 2026-07-10)

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

---

## §2 Conventions

### Data Directory

- Global flag: `--data-dir <path>`, default `./.awis/`
- Directory contains: `runtime.db` (SQLite), `awis.pid` (PID file), structured logs
- Created with `os.MkdirAll` on first use (mode 0755)
- PRD §25 local-first: the data directory is always relative to the working directory unless
  an absolute path is supplied

### PID File

- File: `<data-dir>/awis.pid` — written by `awis start` on engine ready
- Content: decimal PID + newline
- `awis stop` reads this file, sends SIGTERM, waits up to 10s for the process to exit, then
  reports success or error
- On clean exit the engine removes the PID file

### SIGTERM Handling

- `awis start` installs SIGTERM + SIGINT handlers that cancel the engine context
- Context cancel → engine tick completes in-flight step → graceful drain → exit 0
- FR-RM-06: graceful shutdown is the contract; kill -9 (SIGKILL) skips drain (crash path)

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Operational error (storage failure, network, step timeout, instance not found) |
| 2 | Usage error (unknown command, missing required argument, bad flag) |
| 3 | Validation failure (workflow YAML invalid; `awis workflow validate` only) |

### --json Rule (PP-6)

Every command supports `--json` as a global flag. When `--json` is set:
- Output is a single JSON object (or JSON array) on stdout, followed by a newline
- All field names and types are stable (see per-command schemas in §4)
- Errors are still written to stderr in human form (not wrapped in JSON)
- Exit codes are unchanged
- `--json` may not be retrofitted to commands that shipped without it; all M14 commands ship
  with it from day one

---

## §3 Command Tree

### M14 — Live Commands

```
awis
│
├── RUNTIME MANAGEMENT
│   ├── start [--config=<path>]          Start the runtime (foreground)            [C2]
│   ├── stop                             Gracefully stop the runtime                [C2]
│   └── version                          Show version and build info                [C1] ✓
│
├── WORKFLOW EXECUTION
│   ├── submit <workflow-id>             Submit a workflow instance                 [C3]
│   │   [--input k=v | --input @file]
│   │   [--wait] [--timeout=<dur>]
│   ├── signal <instance-id> <name>     Deliver signal to waiting instance         [C3]
│   │   [--payload k=v | --payload @file]
│   └── cancel <instance-id>            Cancel a running or waiting instance       [C3]
│       [--reason=<string>] [--compensate]
│
├── OBSERVABILITY
│   ├── status [--namespace=<ns>]        Live status: active + recent instances    [C2]
│   │   [--all] [--watch]
│   └── trace <instance-id>             Full execution trace for one instance      [C3]
│       [--json] [--full]
│
├── WORKFLOW MANAGEMENT
│   ├── workflow validate <file>         Validate a YAML definition file           [C3]
│   ├── workflow list [--namespace=x]   List registered workflow definitions       [C3]
│   └── workflow show <id>              Show definition: steps, transitions        [C3]
│
└── PLUGIN MANAGEMENT
    ├── plugin install <path>            Install a plugin from local path          [C4]
    └── plugin list                      List installed plugins                    [C4]
```

### M17 — Planned Commands (not yet implemented)

```
awis
│
├── RUNTIME MANAGEMENT
│   └── init [name]                      Initialize project structure + config     [M17]
│
├── OBSERVABILITY
│   ├── history [--workflow=<id>]        Recent completed instances                [M17]
│   │   [--n=20] [--namespace=<ns>]
│   │   [--status=failed|completed]
│   ├── logs [--instance=<id>]           Structured log stream                     [M17]
│   │   [--level=error|info|debug] [--tail]
│   ├── metrics [--namespace=<ns>]       Aggregate execution statistics            [M17]
│   │   [--workflow=<id>]
│   ├── recall "<natural query>"         Query execution history                   [M17]
│   │   [--namespace=<ns>] [--synthesize]
│   ├── replay <instance-id>             Re-run completed instance (dry-run)      [M17]
│   └── audit [--from=<date>]            View audit log entries                   [M17]
│
├── PLUGIN MANAGEMENT
│   ├── plugin remove <name>             Remove a plugin                          [M17]
│   └── plugin status <name>             Plugin health and call statistics        [M17]
│
├── CONFIGURATION
│   ├── config show                      View current configuration               [M17]
│   ├── config set <key> <value>         Set a configuration value                [M17]
│   ├── config validate                  Validate configuration file              [M17]
│   └── config edit                      Open config in $EDITOR                  [M17]
│
└── MAINTENANCE
    ├── rebuild-state [--namespace=x]    Rebuild StateStore from EventLog         [M17]
    ├── export [--format=json]           Export execution history                 [M17]
    │   [--namespace=<ns>]
    │   [--from=<date>] [--to=<date>]
    └── prune-events --before=<date>     Prune EventLog (--dry-run required first)[M17]
        [--dry-run]
```

---

## §4 Per-Command Contracts (M14 Commands)

> **Global flags:** every command accepts `--json`, `--data-dir`, and
> `--namespace` as global flags preceding the subcommand name:
> `awis [--json] [--data-dir D] [--namespace NS] <cmd> …`.
> `--namespace` (default `"default"`) targets the workflow namespace for
> submit/signal/cancel/status/trace; it is ignored by version/start/stop/workflow/plugin.
> Per-command synopses below omit these global flags for brevity.

### version

**Synopsis:** `awis version`

Show the AWIS binary version and the Go runtime version used to build it.

**Flags:** none beyond global `--data-dir`, `--json`

**Human output:**
```
awis version 0.1.0-dev  go go1.26.5
```
Format: `awis version <semver>  go <go_version>`

**JSON schema (`--json`):**
```json
{
  "version":    "string  — AWIS release semver (e.g. \"0.1.0-dev\")",
  "go_version": "string  — runtime.Version() (e.g. \"go1.26.5\")"
}
```

**Errors:** none (version command cannot fail operationally)

**Exit codes:** 0 always

---

### start

**Synopsis:** `awis start [--config=<path>]`

Start the AWIS runtime in the foreground. Discovers YAML workflow definitions via
`dsl.Discover(cwd)`, registers them, optionally loads plugins from
`plugins/*/awis-plugin.yaml`, writes the PID file, then runs the engine pull loop at 100ms
tick until SIGTERM or SIGINT.

**Flags:**
- `--config=<path>` — path to config.yaml (default: `<data-dir>/config.yaml` if it exists)

**Human startup header:**
```
AWIS v0.1.0-dev  db: .awis/runtime.db
Workflows registered: 2  (capture-decision v1.0.0, recall-decision v1.0.0)
Plugins registered:   1  (git-context-plugin)
Intelligence:         none (zero-AI mode)
● running  PID 12345
```

**JSON schema (`--json`):** streams one JSON object per significant event (startup, shutdown).
Startup line:
```json
{
  "event":      "started",
  "version":    "string",
  "db_path":    "string",
  "workflows":  ["string"],
  "plugins":    ["string"],
  "intelligence": "string",
  "pid":        "integer"
}
```

**Errors:**
- Exit 1: storage open failed (db locked, permissions)
- Exit 1: workflow parse/register failed

**Exit codes:** 0 on graceful shutdown, 1 on startup error

---

### stop

**Synopsis:** `awis stop`

Read `<data-dir>/awis.pid`, send SIGTERM to the engine process, wait up to 10s for exit.

**Flags:** none beyond global

**Human output:**
```
Stopping AWIS (PID 12345)...
stopped.
```
On timeout:
```
awis: runtime did not stop within 10s (PID 12345)
  Where:    <data-dir>/awis.pid
  What now: kill -9 12345
```

**JSON schema (`--json`):**
```json
{
  "pid":     "integer",
  "stopped": "boolean",
  "elapsed_ms": "integer"
}
```

**Errors:**
- Exit 1: PID file not found (runtime not started)
- Exit 1: process not found (stale PID file)
- Exit 1: timeout (10s)

**Exit codes:** 0 stopped, 1 error

---

### status

**Synopsis:** `awis status [--namespace=<ns>] [--all] [--watch] [--n=<int>]`

Live status: all active + waiting instances (no pagination), plus last N completed (default
N=10). Failure instances show the failed step inline. Waiting instances show the signal name
and remaining timeout. Running instances show the current step and elapsed time.

**Flags:**
- `--namespace=<ns>` — filter by namespace (default: all)
- `--all` — include all terminal instances (not just last N)
- `--watch` — re-print every 5s (not a TUI; clear + reprint)
- `--n=<int>` — number of recent completed instances (default 10)

**Human output:**
```
AWIS status  2026-07-10 14:23:01

ACTIVE
 ●  running   capture-decision  i-a1b2c3  step: draft-entry  12s elapsed
 ○  waiting   capture-decision  i-d4e5f6  signal: confirm-entry  71h 48m remaining
 ✗  failed    capture-decision  i-g7h8i9  step: publish-entry  3m ago

RECENT (last 10)
 ✓  completed  capture-decision  i-j0k1l2  89s  2026-07-10 14:21:32
 ✓  completed  recall-decision   i-m3n4o5  12s  2026-07-10 14:20:15
```

**JSON schema (`--json`):**
```json
{
  "timestamp": "string (RFC3339)",
  "active": [
    {
      "instance_id":     "string",
      "workflow_id":     "string",
      "namespace":       "string",
      "status":          "string (running|waiting|compensating)",
      "current_step":    "string|null",
      "signal_name":     "string|null",
      "timeout_remaining_s": "integer|null",
      "elapsed_s":       "integer"
    }
  ],
  "recent": [
    {
      "instance_id":   "string",
      "workflow_id":   "string",
      "namespace":     "string",
      "status":        "string (completed|failed|cancelled|compensated)",
      "duration_ms":   "integer",
      "failed_step":   "string|null",
      "completed_at":  "string (RFC3339)"
    }
  ]
}
```

**Errors:**
- Exit 1: storage open failed

**Exit codes:** 0 always (empty active section is not an error)

---

### submit

**Synopsis:** `awis submit <workflow-id> [--input k=v | --input @file.json] [--wait] [--timeout=<dur>]`

Submit a new workflow instance. Prints the instance ID immediately. With `--wait`, polls until
the instance reaches a terminal status and exits with code reflecting the outcome.

**Flags:**
- `--input k=v` — key=value input (repeatable; last value wins per key)
- `--input @file.json` — read inputs from JSON file
- `--wait` — block until terminal status
- `--timeout=<dur>` — maximum wait duration (default: 1h; requires `--wait`)

**Human output (submit only):**
```
Submitted: capture-decision v1.0.0
Instance:  i-a1b2c3
Status:    pending → running

Monitor:  awis status
Debug:    awis trace i-a1b2c3
```

**Human output (--wait, success):**
```
Instance:  i-a1b2c3
Status:    completed ✓  Duration: 89s
```

**Human output (--wait, failure):**
```
Instance:  i-a1b2c3
Status:    failed ✗   Step: draft-entry
           awis trace i-a1b2c3
```

**JSON schema (`--json`):**
```json
{
  "instance_id":   "string",
  "workflow_id":   "string",
  "workflow_version": "string",
  "namespace":     "string",
  "status":        "string",
  "duration_ms":   "integer|null"
}
```

**Errors:**
- Exit 1: workflow not registered
- Exit 1: input parse error
- Exit 1: storage open failed
- Exit 1 (--wait): instance failed or cancelled

**Exit codes:** 0 submitted (or completed when --wait), 1 error

---

### signal

**Synopsis:** `awis signal <instance-id> <signal-name> [--payload k=v | --payload @file.json]`

Deliver a signal to a waiting instance. Uses the M07 signal path (direct storage write under
WAL+busy-timeout; engine picks up on next tick ≤ 100ms).

**Flags:**
- `--payload k=v` — key=value payload (repeatable)
- `--payload @file.json` — read payload from JSON file

**Human output:**
```
Signal delivered: entry_confirmed → i-d4e5f6
Instance resumed; next step: append-to-record
```

**JSON schema (`--json`):**
```json
{
  "instance_id":   "string",
  "signal_name":   "string",
  "delivered":     "boolean",
  "next_step":     "string|null"
}
```

**Errors:**
- Exit 1: instance not found
- Exit 1: instance not in waiting state
- Exit 1: signal name mismatch
- Exit 1: storage open failed

**Exit codes:** 0 delivered, 1 error

---

### cancel

**Synopsis:** `awis cancel <instance-id> [--reason=<string>] [--compensate]`

Cancel a running or waiting instance. With `--compensate`, transitions the instance to the
compensating state instead of cancelled.

**Flags:**
- `--reason=<string>` — cancellation reason (recorded in WorkflowCancelled event payload)
- `--compensate` — run compensation handlers (transitions to compensating state)

**Human output:**
```
Cancellation requested: i-a1b2c3
In-flight steps will complete; no new steps will start.
```

With `--compensate`:
```
→ Instance transitions to 'compensating'; compensation plan runs
→ Final status: compensated
```

**JSON schema (`--json`):**
```json
{
  "instance_id": "string",
  "requested":   "boolean",
  "compensate":  "boolean",
  "reason":      "string|null"
}
```

**Errors:**
- Exit 1: instance not found
- Exit 1: instance already in terminal state
- Exit 1: storage open failed

**Exit codes:** 0 requested, 1 error

---

### trace

**Synopsis:** `awis trace <instance-id> [--full]`

Full execution trace for one instance. Reads events from the EventLog in chronological order.

**Flags:**
- `--full` — do not truncate step outputs (default: truncated to 120 chars)

**Human output:**
```
Trace: capture-decision / i-a1b2c3
Status: completed ✓  Duration: 89s  Trigger: manual

Timeline:
  00:00  ● WorkflowStarted
  00:00  ► StepStarted      draft-entry (attempt 1)
  00:13  ✓ StepCompleted    draft-entry  13.1s   adapter: anthropic, tokens: 847
  00:13  ► StepStarted      confirm-entry (attempt 1)
  00:13  ○ WaitingForSignal confirm-entry: signal 'entry_confirmed'
  71:48  ✓ SignalReceived    entry_confirmed
  71:48  ► StepStarted      append-to-record (attempt 1)
  71:49  ✓ StepCompleted    append-to-record  0.8s
  71:49  ✓ WorkflowCompleted  total: 89s
```

Symbols: ● info, ► started, ✓ completed, ✗ failed, ○ waiting, → transition

**JSON schema (`--json`):**
```json
{
  "instance_id":    "string",
  "workflow_id":    "string",
  "workflow_version": "string",
  "namespace":      "string",
  "status":         "string",
  "duration_ms":    "integer|null",
  "trigger":        "string|null",
  "events": [
    {
      "seq":          "integer",
      "event_type":   "string",
      "occurred_at":  "string (RFC3339)",
      "relative_ms":  "integer",
      "payload":      "object"
    }
  ]
}
```

**Errors:**
- Exit 1: instance not found
- Exit 1: storage open failed

**Exit codes:** 0 found, 1 error

---

### workflow validate

**Synopsis:** `awis workflow validate <file>`

Validate a YAML workflow definition file. Uses `dsl.ValidateFile`. Renders validation errors
in the PRD §18 format. Exits 3 on invalid.

**Flags:** `--json` (global)

**Human output (valid):**
```
workflows/capture-decision.yaml  valid ✓
```

**Human output (invalid):**
```
awis: workflow validation failed: workflows/capture-decision.yaml
  Line 45: step 'confirm-entry' declares fallback 'manual-entry'
           but 'manual-entry' is not defined in this workflow

  Suggestion: Add a step with id 'manual-entry', or remove the fallback declaration.
```

**JSON schema (`--json`):**
```json
{
  "file":    "string",
  "valid":   "boolean",
  "errors": [
    {
      "line":    "integer|null",
      "message": "string"
    }
  ]
}
```

**Exit codes:** 0 valid, 1 operational error (file not found / read error), 3 validation failure

---

### workflow list

**Synopsis:** `awis workflow list [--namespace=<ns>]`

List registered workflow definitions from storage (WorkflowRegistry).

**Flags:**
- `--namespace=<ns>` — filter by namespace

**Human output:**
```
REGISTERED WORKFLOWS

  ID                    VERSION   NAMESPACE   STEPS
  capture-decision      1.0.0     oip         5
  recall-decision       1.0.0     oip         4
```

**JSON schema (`--json`):**
```json
{
  "workflows": [
    {
      "id":        "string",
      "version":   "string",
      "namespace": "string",
      "step_count": "integer"
    }
  ]
}
```

**Exit codes:** 0 always (empty list is not an error)

---

### workflow show

**Synopsis:** `awis workflow show <id>`

Show a registered workflow definition summary: steps, transitions, signals required.

**Flags:**
- `--namespace=<ns>` — namespace to look in (default: all)

**Human output:**
```
Workflow: capture-decision v1.0.0  (namespace: oip)

Steps:
  draft-entry       intelligence  → confirm-entry
  confirm-entry     signal        → append-to-record  (signal: entry_confirmed)
  append-to-record  native        → publish-entry
  publish-entry     native        → [end]

Fallbacks:
  draft-entry  →  manual-entry (fallback when intelligence unavailable)
```

**JSON schema (`--json`):**
```json
{
  "id":        "string",
  "version":   "string",
  "namespace": "string",
  "steps": [
    {
      "id":         "string",
      "type":       "string",
      "next":       "string|null",
      "fallback":   "string|null",
      "wait_signal": "string|null"
    }
  ]
}
```

**Exit codes:** 0 found, 1 not found

---

### plugin install

**Synopsis:** `awis plugin install <path>`

Install a plugin from a local path. Validates the `awis-plugin.yaml` manifest at that path,
registers it in the PluginStore (local-path semantics: no file copy). Writes a
`PluginRegistered` audit row.

**Flags:** none beyond global

**Human output:**
```
Plugin installed: git-context-plugin  v0.1.0
  Path:     plugins/git-context-plugin
  Provides: git.context.assemble, git.diff.fetch
```

**JSON schema (`--json`):**
```json
{
  "name":       "string",
  "version":    "string",
  "path":       "string",
  "provides":   ["string"]
}
```

**Errors:**
- Exit 1: path not found
- Exit 1: manifest missing or invalid
- Exit 1: storage open failed

**Exit codes:** 0 installed, 1 error

---

### plugin list

**Synopsis:** `awis plugin list`

List installed plugins from PluginStore.

**Flags:** none beyond global

**Human output:**
```
INSTALLED PLUGINS

  NAME                  VERSION   STATUS   PATH
  git-context-plugin    0.1.0     active   plugins/git-context-plugin
```

**JSON schema (`--json`):**
```json
{
  "plugins": [
    {
      "name":    "string",
      "version": "string",
      "status":  "string",
      "path":    "string"
    }
  ]
}
```

**Exit codes:** 0 always

---

## §5 Error Taxonomy

Every error AWIS shows answers three questions (PRD §26 verbatim):
1. **What happened?** (specific, not generic)
2. **Where?** (instance ID, step ID, plugin name, config key, file line number)
3. **What now?** (a concrete next command or action)

### Error Template

```
awis: <what happened>
  Where:    <instance id | step id | file:line | plugin name | config key>
  What now: <concrete command or action>
```

### Examples

**Category 1 — User error (configuration, definition):**
```
awis: workflow validation failed: workflows/capture-decision.yaml
  Line 45: step 'confirm-entry' declares fallback 'manual-entry'
           but 'manual-entry' is not defined in this workflow

  Suggestion: Add a step with id 'manual-entry', or remove the fallback declaration.
  Example:
    - id: manual-entry
      name: Manual Entry
      type: signal
      wait_signal:
        name: manual_draft_provided
        timeout: 24h
        timeout_action: fail
```

**Category 2 — Execution error (step failure visible in trace):**
```
awis: step 'draft-entry' failed after 3 attempts
  Instance:  i-m4n5o6
  Workflow:  capture-decision v1.0.0
  Error:     connection refused (anthropic API)
  Status:    workflow moved to 'failed' state; compensation running

  Diagnose:  awis trace i-m4n5o6
  Check API: awis config show | grep intelligence
```

**Category 3 — Infrastructure error (storage, plugin):**
```
awis: plugin 'git-context-plugin' failed to spawn (3/3 attempts)
  Error:    python3: ModuleNotFoundError: No module named 'gitpython'
  Impact:   Steps requiring 'git.context.assemble' will fail

  Fix:      pip install gitpython
            awis plugin status git-context-plugin
```

**Category 4 — Transient errors (silently recovered):**
- Intelligence timeouts triggering retry
- Step retry within policy limits
- Plugin restart within limits (1/3, 2/3)

These do NOT appear in `awis status` or as error output. They appear ONLY in `awis trace` as
part of the execution timeline. Silent recovery is a feature, not a hidden failure.

### Instance Not Found (Empty State)

```
awis: instance 'i-notfound' not found in namespace 'oip'
  Where:    --data-dir .awis/
  What now: awis history
            awis config show | grep namespace
```

### Unknown Command (Usage Error, exit 2)

```
awis: unknown command "frobulate"

<full usage text>
```

---

## §6 Storage Notes

`OpenStorage(dataDir)` in `cmd/awis/storage.go` calls `sdk.SQLiteStorage(path)` which calls
`internal/storage.Open` which applies the following pragmas before returning:

- `PRAGMA journal_mode=WAL` — enables WAL mode for concurrent read-only access
- `PRAGMA busy_timeout=5000` — 5s busy wait before SQLITE_BUSY (handles CLI + engine concurrency)
- `PRAGMA foreign_keys=ON`
- `PRAGMA synchronous=NORMAL`

No additional PRAGMA is needed in `cmd/awis`. The WAL+busy-timeout property is a contract of
`sdk.SQLiteStorage` / `internal/storage.Open` (verified in `internal/storage/db_test.go`
`TestWALModeActive`). If this contract ever changes, `cmd/awis/storage.go` must be updated.
