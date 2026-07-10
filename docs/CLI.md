# AWIS CLI Reference

**Normative contract:** `docs/CLI_CONTRACT.md` (TDS-07)
This document is a navigational reference for the shipped M14 commands. All
output formats, JSON schemas, exit codes, and error taxonomy are defined
verbatim in TDS-07. If there is any conflict between this file and TDS-07,
TDS-07 is authoritative.

**M17 commands** (init, history, logs, metrics, recall, replay, audit,
plugin status/remove, config, rebuild-state, export, prune-events) are
planned and marked as upcoming in TDS-07 §3. They are not described here.

---

## Global Flags

```
awis [--data-dir <path>] [--json] <command> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--data-dir <path>` | `./.awis/` | Data directory (runtime.db, awis.pid, logs) |
| `--json` | false | Output machine-readable JSON (PP-6; all M14 commands) |

The data directory is created with `os.MkdirAll` on first use. PRD §25
local-first: the path is relative to the working directory unless absolute.

---

## version

**Synopsis:** `awis version`

Show the AWIS binary version and the Go runtime version.

```
awis version 0.1.0-dev  go go1.26.4
```

**--json:**
```json
{"version": "0.1.0-dev", "go_version": "go1.26.4"}
```

Exit code: 0 always. See TDS-07 §4 for the full contract.

---

## start

**Synopsis:** `awis start [--config=<path>]`

Start the AWIS runtime in the foreground. Discovers YAML workflow definitions
via `dsl.Discover(cwd)`, registers them, optionally loads plugins from
`plugins/*/awis-plugin.yaml`, writes the PID file, then runs the engine pull
loop at 100ms tick until SIGTERM or SIGINT.

**Flags:**
- `--config=<path>` — path to config.yaml

**Startup output:**
```
AWIS v0.1.0-dev  db: .awis/runtime.db
Workflows registered: 2  (capture-decision v1.0.0, recall-decision v1.0.0)
Plugins registered:   1  (git-context-plugin)
Intelligence:         none (zero-AI mode)
● running  PID 12345
```

Exit code: 0 graceful shutdown, 1 startup error. See TDS-07 §4 for the full
contract including the --json streaming schema.

---

## stop

**Synopsis:** `awis stop`

Read `<data-dir>/awis.pid`, send SIGTERM to the engine process, wait up to 10s.

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

**--json:**
```json
{"pid": 12345, "stopped": true, "elapsed_ms": 142}
```

Exit code: 0 stopped, 1 error. See TDS-07 §4.

---

## status

**Synopsis:** `awis status [--namespace=<ns>] [--all] [--watch] [--n=<int>]`

Live status: all active + waiting instances, plus last N completed (default N=10).

**Flags:**
- `--namespace=<ns>` — filter by namespace
- `--all` — include all terminal instances
- `--watch` — re-print every 5s
- `--n=<int>` — number of recent completed instances (default 10)

**Example output:**
```
AWIS status  2026-07-10 14:23:01

ACTIVE
 ●  running   capture-decision  i-a1b2c3  step: draft-entry  12s elapsed
 ○  waiting   capture-decision  i-d4e5f6  signal: confirm-entry  71h 48m remaining

RECENT (last 10)
 ✓  completed  capture-decision  i-j0k1l2  89s  2026-07-10 14:21:32
```

Exit code: 0 always. See TDS-07 §4 for the full --json schema.

---

## submit

**Synopsis:** `awis submit <workflow-id> [--input k=v | --input @file.json] [--wait] [--timeout=<dur>]`

Submit a new workflow instance.

**Flags:**
- `--input k=v` — key=value input (repeatable)
- `--input @file.json` — read inputs from JSON file
- `--wait` — block until terminal status
- `--timeout=<dur>` — maximum wait (default: 1h; requires --wait)

**Example output:**
```
Submitted: capture-decision v1.0.0
Instance:  i-a1b2c3
Status:    pending → running

Monitor:  awis status
Debug:    awis trace i-a1b2c3
```

**--json:**
```json
{"instance_id": "i-a1b2c3", "workflow_id": "capture-decision",
 "workflow_version": "1.0.0", "namespace": "oip",
 "status": "pending", "duration_ms": null}
```

Exit code: 0 submitted (or completed with --wait), 1 error. See TDS-07 §4.

---

## signal

**Synopsis:** `awis signal <instance-id> <signal-name> [--payload k=v | --payload @file.json]`

Deliver a signal to a waiting instance (M07 signal path).

**Flags:**
- `--payload k=v` — key=value payload (repeatable)
- `--payload @file.json` — read payload from JSON file

**Example output:**
```
Signal delivered: entry_confirmed → i-d4e5f6
Instance resumed; next step: append-to-record
```

**--json:**
```json
{"instance_id": "i-d4e5f6", "signal_name": "entry_confirmed",
 "delivered": true, "next_step": "append-to-record"}
```

Exit code: 0 delivered, 1 error. See TDS-07 §4.

---

## cancel

**Synopsis:** `awis cancel <instance-id> [--reason=<string>] [--compensate]`

Cancel a running or waiting instance.

**Flags:**
- `--reason=<string>` — cancellation reason
- `--compensate` — run compensation handlers

**Example output:**
```
Cancellation requested: i-a1b2c3
In-flight steps will complete; no new steps will start.
```

**--json:**
```json
{"instance_id": "i-a1b2c3", "requested": true,
 "compensate": false, "reason": null}
```

Exit code: 0 requested, 1 error. See TDS-07 §4.

---

## trace

**Synopsis:** `awis trace <instance-id> [--json] [--full]`

Full execution trace for one instance (reads EventLog in chronological order).

**Flags:**
- `--full` — do not truncate step outputs (default: truncated to 120 chars)

**Example output:**
```
Trace: capture-decision / i-a1b2c3
Status: completed ✓  Duration: 89s  Trigger: manual

Timeline:
  00:00  ● WorkflowStarted
  00:00  ► StepStarted      draft-entry (attempt 1)
  00:13  ✓ StepCompleted    draft-entry  13.1s
  00:13  ✓ WorkflowCompleted  total: 89s
```

Exit code: 0 found, 1 not found or storage error. See TDS-07 §4 for the
full timeline symbols and --json schema.

---

## workflow validate

**Synopsis:** `awis workflow validate <file>`

Validate a YAML workflow definition file. Uses `dsl.ValidateFile`. Renders
validation errors in the PRD §18 format.

**Example output (valid):**
```
workflows/capture-decision.yaml  valid ✓
```

**Example output (invalid):**
```
awis: workflow validation failed: workflows/capture-decision.yaml
  Line 45: step 'confirm-entry' declares fallback 'manual-entry'
           but 'manual-entry' is not defined in this workflow

  Suggestion: Add a step with id 'manual-entry', or remove the fallback declaration.
```

Exit code: 0 valid, 1 file error, 3 validation failure. See TDS-07 §4.

---

## workflow list

**Synopsis:** `awis workflow list [--namespace=<ns>]`

List registered workflow definitions from storage.

**Flags:**
- `--namespace=<ns>` — filter by namespace

**Example output:**
```
REGISTERED WORKFLOWS

  ID                    VERSION   NAMESPACE   STEPS
  capture-decision      1.0.0     oip         5
  recall-decision       1.0.0     oip         4
```

**--json:**
```json
{"workflows": [{"id": "capture-decision", "version": "1.0.0",
                "namespace": "oip", "step_count": 5}]}
```

Exit code: 0 always. See TDS-07 §4.

---

## workflow show

**Synopsis:** `awis workflow show <id> [--namespace=<ns>]`

Show a registered workflow definition: steps, transitions, signals required.

**Example output:**
```
Workflow: capture-decision v1.0.0  (namespace: oip)

Steps:
  draft-entry       intelligence  → confirm-entry
  confirm-entry     signal        → append-to-record  (signal: entry_confirmed)
  append-to-record  native        → publish-entry
  publish-entry     native        → [end]
```

Exit code: 0 found, 1 not found. See TDS-07 §4.

---

## plugin install

**Synopsis:** `awis plugin install <path>`

Install a plugin from a local directory or manifest file. Validates the
`awis-plugin.yaml` manifest, registers it in the PluginStore, and writes a
`PluginRegistered` audit row. V1 semantics: no file copy — registers the
local path in place.

**Example output:**
```
Plugin installed: git-context-plugin  v1.0.0
  Path:     plugins/git-context-plugin
  Provides: git.context.assemble, git.diff.fetch
```

**--json:**
```json
{"name": "git-context-plugin", "version": "1.0.0",
 "path": "/abs/path/to/plugin",
 "provides": ["git.context.assemble", "git.diff.fetch"]}
```

**Errors:**
- Exit 1: path not found
- Exit 1: manifest missing or invalid
- Exit 1: storage open failed

See TDS-07 §4 for the full contract.

---

## plugin list

**Synopsis:** `awis plugin list`

List installed plugins from PluginStore.

**Example output:**
```
INSTALLED PLUGINS

  NAME                  VERSION   STATUS   PATH
  git-context-plugin    1.0.0     registered   plugins/git-context-plugin
```

**--json:**
```json
{"plugins": [{"name": "git-context-plugin", "version": "1.0.0",
              "status": "registered", "path": "/abs/path/to/plugin"}]}
```

Exit code: 0 always. See TDS-07 §4.

---

## Error Format

All errors follow the what/where/what-now taxonomy (PRD §26):

```
awis: <what happened>
  Where:    <instance id | file | plugin name>
  What now: <concrete command or action>
```

Exit codes: 0 ok, 1 operational error, 2 usage error, 3 validation failure.

See TDS-07 §5 for the full error taxonomy with examples.

---

## Upcoming Commands (M17)

The following commands are planned for M17 and are not yet implemented:
init, history, logs, metrics, recall, replay, audit, plugin status,
plugin remove, config show/set/validate/edit, rebuild-state, export,
prune-events.

See TDS-07 §3 for the full M17 command tree.
