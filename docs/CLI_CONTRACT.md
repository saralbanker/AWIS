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

### M17 — Implemented Commands (added by M17-C1, C2, C3, C1r)

```
awis
│
├── RUNTIME MANAGEMENT
│   └── init [directory]                 Initialize project structure + config     [M17-C3] ✓
│
├── OBSERVABILITY
│   ├── history [--workflow=<id>]        Recent completed instances                [M17-C1] ✓
│   │   [--n=20] [--namespace=<ns>]
│   │   [--status=failed|completed]
│   ├── logs [--instance=<id>]           Structured log stream                     [M17-C1] ✓
│   │   [--level=error|info|debug] [--tail]
│   ├── metrics [--namespace=<ns>]       Aggregate execution statistics            [M17-C1] ✓
│   │   [--workflow=<id>]
│   ├── recall "<natural query>"         Query execution history                   [M17-C1] ✓
│   │   [--namespace=<ns>] [--synthesize]
│   ├── replay <instance-id>             Re-run completed instance (dry-run)      [M17-C1] ✓
│   └── audit [--limit=<int>]            View audit log entries                   [M17-C1] ✓
│
├── PLUGIN MANAGEMENT
│   ├── plugin remove <name>             Remove a plugin                          [M17-C1] ✓
│   └── plugin status <name>             Plugin health and call statistics        [M17-C1] ✓
│
├── CONFIGURATION
│   ├── config show                      View current configuration               [M17-C2] ✓
│   ├── config set <key> <value>         Set a configuration value                [M17-C2] ✓
│   ├── config validate                  Validate configuration file              [M17-C2] ✓
│   └── config edit                      Open config in $EDITOR                   [M17-C2] ✓
│
└── MAINTENANCE
    ├── rebuild-state [--namespace=x]    Rebuild StateStore from EventLog         [M17-C2] ✓
    ├── export [--format=json]           Export execution history                 [M17-C1] ✓
    │   [--namespace=<ns>]
    │   [--from=<date>] [--to=<date>]
    └── prune-events --before=<date>     Prune EventLog (--dry-run required first)[M17-C1] ✓
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

### init

**Synopsis:** `awis init [--force] [directory]` (M17-C3; FR-RM-01)

Scaffold a new AWIS project into `[directory]` (default: current directory)
from an embedded file set (`//go:embed all:scaffold`; the `all:` prefix is
required so `scaffold/.gitignore` — a dot-file — is not silently excluded by
Go's default embed rule):

- `config.yaml` — minimal project config (`namespace`, `tick`, commented
  `anthropic_api_key` note)
- `.gitignore` — excludes `.awis/` and `*.pid`
- `workflows/hello-world.yaml`, `workflows/with-signal.yaml`,
  `workflows/with-intelligence.yaml` — the three M10 example workflows,
  embedded byte-identical to `examples/workflows/`
- `handlers/example_handler.go` — native step handler stubs for an
  embedding Go program to register via `sdk.Runtime.RegisterHandler`
- `README_AWIS.md` — quick-start + project-structure reference

Refuses to write into a non-empty target directory unless `--force` is
given (existing files are left in place; scaffold files are written
alongside them).

**Flags:**
- `--force` — allow writing into a non-empty target directory

**Human output:**
```
Initialized AWIS project in /abs/path/my-project

  created  config.yaml
  created  .gitignore
  created  workflows/hello-world.yaml
  created  workflows/with-signal.yaml
  created  workflows/with-intelligence.yaml
  created  handlers/example_handler.go
  created  README_AWIS.md

Next steps:
  awis start
  awis submit hello-world --input name=World
  awis status
```

**JSON schema (`--json`):**
```json
{
  "target": "string",
  "files":  ["string"]
}
```

**Errors:**
- Exit 1: target directory not empty and `--force` not given
- Exit 1: cannot resolve/create the target directory (permissions)
- Exit 1: cannot write a scaffold file (permissions, disk space)

**Exit codes:** 0 scaffolded, 1 error

**Note:** the scaffolded `workflows/hello-world.yaml` carries its own YAML
`namespace: examples`. `sdk.Runtime.Submit`'s cross-process lookup filters
`ListWorkflows` by the CALLING process's `--namespace` (not the
definition's namespace), so submitting it requires
`awis --namespace=examples submit hello-world`.

---

### history

**Synopsis:** `awis history [--workflow=<id>] [--n=20] [--namespace=<ns>] [--status=failed|completed]`

List completed, failed, and cancelled instances (terminal states) from the database.
Shows instance ID, workflow ID, namespace, status, duration, and failure details.

**Flags:**
- `--workflow=<id>` — filter by workflow definition ID
- `--n=<int>` — number of instances to show (default 20)
- `--namespace=<ns>` — filter by namespace
- `--status=<status>` — filter by status: `completed`, `failed`, or `cancelled`

**Human output:**
```
RECENT INSTANCES (last 20)

  INSTANCE         WORKFLOW           STATUS      DURATION  COMPLETED AT
  i-a1b2c3         capture-decision   completed   89s       2026-07-10 14:21:32
  i-d4e5f6         recall-decision    failed      12s       2026-07-10 14:20:15
  i-g7h8i9         capture-decision   completed   156s      2026-07-10 14:18:47
```

**JSON schema (`--json`):**
```json
{
  "instances": [
    {
      "instance_id":   "string",
      "workflow_id":   "string",
      "namespace":     "string",
      "status":        "string",
      "duration_ms":   "integer",
      "failed_step":   "string|null",
      "completed_at":  "string (RFC3339)"
    }
  ]
}
```

**Errors:**
- Exit 1: storage open failed

**Exit codes:** 0 always (empty list is not an error)

---

### logs

**Synopsis:** `awis logs [--tail=<int>] [--level=<level>] [--instance=<id>]`

Display structured runtime log from `<data-dir>/awis.log`. Filters by log level
and instance ID. Default shows last 50 lines.

**Flags:**
- `--tail=<int>` — number of log lines to show (default 50)
- `--level=<level>` — filter by level: `error`, `info`, or `debug`
- `--instance=<id>` — filter by instance ID

**Human output (excerpt):**
```
Log file: .awis/awis.log

[2026-07-10 14:23:01] info   engine started (PID 12345)
[2026-07-10 14:23:00] debug  discovering workflows...
[2026-07-10 14:22:59] info   registered 2 workflows
```

**JSON schema (`--json`):**
```json
{
  "log_file": "string",
  "lines":    ["string"]
}
```

**Errors:**
- Exit 1: log file not found (runtime not started yet)
- Exit 1: cannot read log file (permissions)

**Exit codes:** 0 always (missing log file shows usage hint, not error)

---

### metrics

**Synopsis:** `awis metrics [--namespace=<ns>] [--workflow=<id>]`

Show aggregate execution statistics: instance counts by status, average duration,
step success rates, plugin call counts.

**Flags:**
- `--namespace=<ns>` — filter by namespace
- `--workflow=<id>` — filter by workflow definition ID

**Human output:**
```
EXECUTION METRICS

  Workflows Executed:    2
  Total Instances:       145
    Completed:           120  (82.8%)
    Failed:              18   (12.4%)
    Cancelled:           7    (4.8%)

  Average Duration:      42.3s
  Slowest Step:          draft-entry (22.1s avg)
  Plugin Calls:          284
    git-context-plugin:  187
    other:               97
```

**JSON schema (`--json`):**
```json
{
  "namespace":        "string",
  "workflows_count":  "integer",
  "instances_total":  "integer",
  "status_breakdown": {
    "completed":      "integer",
    "failed":         "integer",
    "cancelled":      "integer"
  },
  "avg_duration_ms":  "integer",
  "plugin_calls":     {
    "<plugin-name>":  "integer"
  }
}
```

**Errors:**
- Exit 1: storage open failed

**Exit codes:** 0 always

---

### recall

**Synopsis:** `awis recall "<query>" [--namespace=<ns>] [--synthesize]`

Query execution history using full-text search. Returns instances and events
matching the query string. With `--synthesize`, uses AI summarization (future).

**Flags:**
- `--namespace=<ns>` — search within namespace only
- `--synthesize` — use intelligence to summarize results (requires config)

**Human output:**
```
RECALL RESULTS for "anthropic"

  Matching Instances:
    i-a1b2c3  capture-decision   2026-07-10 14:21:32  (mentions: anthropic API timeout)
    i-d4e5f6  recall-decision    2026-07-10 14:20:15  (mentions: anthropic error)

  Matching Events:
    [i-a1b2c3] 00:13  StepCompleted  draft-entry  adapter: anthropic, tokens: 847
    [i-d4e5f6] 00:08  StepFailed     draft-entry  anthropic: rate limit exceeded
```

**JSON schema (`--json`):**
```json
{
  "query":       "string",
  "results": [
    {
      "type":       "string (instance|event)",
      "instance_id": "string",
      "workflow_id": "string|null",
      "timestamp":  "string (RFC3339)",
      "snippet":    "string",
      "score":      "number (relevance 0-1)"
    }
  ]
}
```

**Errors:**
- Exit 1: storage open failed
- Exit 1: query parse error (empty or malformed)

**Exit codes:** 0 always (no matches returns empty results, not error)

---

### replay

**Synopsis:** `awis replay <instance-id>`

Dry-run re-execution of a completed instance. Walks the event log and prints
what WOULD happen if the instance were re-run. No state is written.

**Flags:** none beyond global `--json`, `--data-dir`

**Human output:**
```
REPLAY (dry-run)  capture-decision / i-a1b2c3
Status: completed  (original)

The following steps WOULD run if this instance were re-executed:

  00:00  ● WorkflowStarted         → initialize workflow state
  00:13  ► StepStarted  draft-entry → dispatch step to worker
  00:13  ✓ StepCompleted draft-entry → record step result; advance to next step
  00:89  ✓ WorkflowCompleted        → mark instance completed

NOTE: This is a dry-run. No steps were executed and no state was written.
```

**JSON schema (`--json`):**
```json
{
  "instance_id":   "string",
  "workflow_id":   "string",
  "namespace":     "string",
  "status":        "string",
  "dry_run":       true,
  "steps": [
    {
      "order":       "integer",
      "event_type":  "string",
      "step_id":     "string|null",
      "emitted_at":  "string (RFC3339)",
      "relative_ms": "integer",
      "action":      "string"
    }
  ]
}
```

**Errors:**
- Exit 1: instance not found
- Exit 1: cannot read events (storage corruption)

**Exit codes:** 0 found, 1 error

---

### audit

**Synopsis:** `awis audit [--limit=<int>]`

View the audit log: records of all configuration changes, plugin registrations,
removals, and other administrative actions.

**Flags:**
- `--limit=<int>` — maximum number of entries to show (default 50)

**Human output:**
```
AUDIT LOG (last 50)

  TIME                    EVENT            ACTOR     DETAILS
  2026-07-10 14:23:15     ConfigChanged    cli       key=namespace, value=staging
  2026-07-10 14:22:50     PluginRegistered cli       git-context-plugin v0.1.0
  2026-07-10 14:21:32     WorkflowStarted  engine    capture-decision
```

**JSON schema (`--json`):**
```json
{
  "entries": [
    {
      "id":              "integer",
      "timestamp":       "string (RFC3339)",
      "event_type":      "string",
      "actor":           "string",
      "payload_summary": "string (JSON-encoded)"
    }
  ]
}
```

**Errors:**
- Exit 1: storage open failed

**Exit codes:** 0 always

---

### config show

**Synopsis:** `awis config show`

Display the current configuration from `<data-dir>/config.yaml`. Shows all
key-value pairs.

**Flags:** none beyond global `--json`, `--data-dir`

**Human output:**
```
Config: .awis/config.yaml

  namespace:  default
  tick:       100ms
```

**JSON schema (`--json`):**
```json
{
  "config_file": "string",
  "keys": {
    "<key>": "string"
  }
}
```

**Errors:**
- Exit 1: config file not found (will show hint to run `awis init`)

**Exit codes:** 0 found, 1 error

---

### config set

**Synopsis:** `awis config set <key> <value>`

Set a configuration key-value pair in `<data-dir>/config.yaml`. Creates the
file if it does not exist. API keys are masked in output.

**Flags:** none beyond global `--json`, `--data-dir`

**Human output:**
```
Set: namespace = staging
Written to: .awis/config.yaml
```

**JSON schema (`--json`):**
```json
{
  "config_file": "string",
  "key":         "string",
  "value":       "string",
  "masked":      "boolean"
}
```

**Errors:**
- Exit 1: cannot write config file (permissions, disk space)

**Exit codes:** 0 set, 1 error

---

### config validate

**Synopsis:** `awis config validate`

Validate the configuration file against the schema. Reports unknown keys and
malformed entries.

**Flags:** none beyond global `--json`, `--data-dir`

**Human output (valid):**
```
.awis/config.yaml  valid
```

**Human output (invalid):**
```
awis: config validation failed
  Line 3: unknown key 'invalid_key'
  Suggestion: Remove the line or use a valid key (namespace, tick, etc.)
```

**JSON schema (`--json`):**
```json
{
  "config_file": "string",
  "valid":       "boolean",
  "errors":      ["string"]
}
```

**Errors:**
- Exit 1: config file not found
- Exit 1: cannot read file (permissions)

**Exit codes:** 0 valid, 1 invalid or read error

---

### config edit

**Synopsis:** `awis config edit`

Open `<data-dir>/config.yaml` in `$EDITOR` for manual editing. Returns after
the editor closes.

**Flags:** none beyond global `--data-dir`

**Human output:**
```
Opening .awis/config.yaml in $EDITOR...
(editor window opens)
```

**Errors:**
- Exit 1: $EDITOR not set
- Exit 1: cannot open config file (permissions)
- Exit 1: editor exited with error

**Exit codes:** 0 edited, 1 error

---

### plugin remove

**Synopsis:** `awis plugin remove <name>`

Remove (uninstall) a plugin from the PluginStore. Marks it as `removed` and
writes a PluginRemoved audit entry.

**Flags:** none beyond global `--json`, `--data-dir`

**Human output:**
```
Plugin removed: git-context-plugin
  Status set to: removed
  Audit row written: PluginRemoved
```

**JSON schema (`--json`):**
```json
{
  "name":    "string",
  "status":  "string",
  "removed": "boolean"
}
```

**Errors:**
- Exit 1: plugin not found
- Exit 1: storage open failed

**Exit codes:** 0 removed, 1 error

---

### plugin status

**Synopsis:** `awis plugin status <name>`

Show plugin health and call statistics: registration status, version, path,
and cumulative call count.

**Flags:** none beyond global `--json`, `--data-dir`

**Human output:**
```
Plugin:   git-context-plugin
Version:  0.1.0
Status:   registered
Path:     plugins/git-context-plugin
Registered: 2026-07-10T14:23:00Z
```

**JSON schema (`--json`):**
```json
{
  "name":          "string",
  "version":       "string",
  "status":        "string",
  "path":          "string",
  "registered_at": "string (RFC3339)"
}
```

**Errors:**
- Exit 1: plugin not found
- Exit 1: storage open failed

**Exit codes:** 0 found, 1 error

---

### rebuild-state

**Synopsis:** `awis rebuild-state [--namespace=<ns>]`

Rebuild the `workflow_instances` projection from the EventLog. Useful after
manual EventLog edits or corruption recovery. Safe to run before starting
the engine.

**Flags:**
- `--namespace=<ns>` — rebuild only for specified namespace (default: all)

**Human output:**
```
rebuild-state: projection rebuilt from EventLog
  Mode:    all
  DB:      .awis/runtime.db

  What now: awis start  (resume engine; rebuild is safe before start)
```

**JSON schema (`--json`):**
```json
{
  "mode":    "string (all|namespace)",
  "success": "boolean",
  "message": "string"
}
```

**Errors:**
- Exit 1: storage open failed
- Exit 1: EventLog corruption detected (data integrity error)

**Exit codes:** 0 success, 1 error

---

### export

**Synopsis:** `awis export [--format=json] [--namespace=<ns>] [--from=<date>] [--to=<date>]`

Export execution history to JSON format. Includes workflow definitions, instances,
events, and audit log. Optionally filters by namespace and date range.

**Flags:**
- `--format=json` — output format (currently only `json` supported)
- `--namespace=<ns>` — export only specified namespace
- `--from=<date>` — start date (RFC3339 or YYYY-MM-DD)
- `--to=<date>` — end date (RFC3339 or YYYY-MM-DD)

**Human output (summary):**
```
Exporting execution history...
  Format:      json
  Namespace:   oip
  Range:       2026-07-01 to 2026-07-10
  Workflows:   2
  Instances:   145
  Events:      1247
  Audit rows:  89

Output written to: .awis/export-20260710.json
```

**JSON schema (`--json`):**
```json
{
  "export_date":  "string (RFC3339)",
  "namespace":    "string",
  "workflows":    ["object"],
  "instances":    ["object"],
  "events":       ["object"],
  "audit_log":    ["object"]
}
```

**Errors:**
- Exit 1: storage open failed
- Exit 1: date parse error (invalid format)

**Exit codes:** 0 exported, 1 error

---

### prune-events

**Synopsis:** `awis prune-events --before=<date> [--dry-run]`

Prune (delete) EventLog entries older than a given date. Requires `--dry-run`
on first run to preview what will be deleted.

**Flags:**
- `--before=<date>` — delete events before this date (RFC3339 or YYYY-MM-DD)
- `--dry-run` — preview what would be deleted without actually deleting

**Human output (dry-run):**
```
prune-events: dry-run mode
  Before:        2026-06-01
  Events found:  342
  Space freed:   ~45 MB

  What now: awis prune-events --before=2026-06-01  (no --dry-run to confirm)
```

**Human output (confirmed):**
```
prune-events: 342 events deleted
  Before:        2026-06-01
  Space freed:   ~45 MB
```

**JSON schema (`--json`):**
```json
{
  "before":           "string (RFC3339)",
  "dry_run":          "boolean",
  "events_deleted":   "integer",
  "estimated_bytes":  "integer"
}
```

**Errors:**
- Exit 1: date parse error (invalid format)
- Exit 1: storage open failed
- Exit 2: `--dry-run` not provided (required safety measure)

**Exit codes:** 0 success, 1 operational error, 2 usage error

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
