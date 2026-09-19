# AWIS CLI Reference

**Normative contract:** `docs/CLI_CONTRACT.md` (TDS-07)
This document is a navigational reference for the shipped commands (M14 +
M17-C1/C2/C3). All output formats, JSON schemas, exit codes, and error
taxonomy are defined verbatim in TDS-07. If there is any conflict between
this file and TDS-07, TDS-07 is authoritative.

---

## Global Flags

```
awis [--data-dir <path>] [--json] <command> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--data-dir <path>` | `./.awis/` | Data directory (runtime.db, awis.pid, logs) |
| `--json` | false | Output machine-readable JSON (PP-6; all M14 commands) |
| `--namespace <ns>` | `"default"` | Namespace for submit/signal/cancel/status/trace |

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
- `--tick=<dur>` — engine tick interval (default 100ms)
- `--namespace=<ns>` — namespace for this runtime instance (default "default")

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

**Cron trigger (F-2; M17-C2):** `start` scans registered workflow definitions
for `type: schedule` triggers with a standard 5-field `config.schedule`
expression (`*/N`, lists, ranges, and exact fields are supported; stdlib-only
parser, no third-party cron library). Matching definitions are submitted
through the same intake path `awis submit` uses, once per matching minute.
Invalid schedules are logged to stderr and skipped; `start` itself never
fails due to a bad cron expression. This scanner lives entirely in
`cmd/awis/start.go` — the engine and core packages are untouched.

**File-sink log (M17-C1):** `start` also opens `<data-dir>/awis.log`
(append mode) and writes a structured JSON line on startup; `awis logs`
reads this file.

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

## plugin status

**Synopsis:** `awis plugin status <name>`

Show health/registration info for one installed plugin (name, version,
status, path, registered-at) from the PluginStore.

**Example output:**
```
Plugin:   git-context-plugin
Version:  1.0.0
Status:   registered
Path:     plugins/git-context-plugin
Registered: 2026-07-10T14:20:00Z
```

**--json:**
```json
{"name": "git-context-plugin", "version": "1.0.0", "status": "registered",
 "path": "plugins/git-context-plugin", "registered_at": "2026-07-10T14:20:00Z"}
```

Exit code: 0 found, 1 not found or storage does not support PluginStore.
See TDS-07 §4.

---

## plugin remove

**Synopsis:** `awis plugin remove <name>`

Soft-delete a plugin: sets its PluginStore status to `"removed"` (StoragePort
is frozen; V1 has no hard delete) and writes a `PluginRemoved` audit row.

**Example output:**
```
Plugin removed: git-context-plugin
  Status set to: removed
  Audit row written: PluginRemoved
```

**--json:**
```json
{"name": "git-context-plugin", "status": "removed", "removed": true}
```

Exit code: 0 removed, 1 not found or storage error. See TDS-07 §4.

---

## history

**Synopsis:** `awis history [--workflow=<id>] [--n=20] [--namespace=<ns>] [--status=<status>]`

List completed/failed/cancelled/compensated instances with durations,
most-recent first.

**Flags:**
- `--workflow=<id>` — filter by workflow ID
- `--n=<int>` — number of instances to return (default 20)
- `--namespace=<ns>` — filter by namespace
- `--status=<status>` — restrict to one status (e.g. `failed`, `completed`)

**Example output:**
```
HISTORY (last 20)

  INSTANCE     WORKFLOW             NAMESPACE    STATUS     DURATION   COMPLETED AT
  ✓ i-a1b2c3   capture-decision     oip          completed  89s        2026-07-10T14:21:32Z
```

**--json:**
```json
{"instances": [{"instance_id": "i-a1b2c3", "workflow_id": "capture-decision",
                "namespace": "oip", "status": "completed", "duration_ms": 89000,
                "failed_step": null, "completed_at": "2026-07-10T14:21:32Z"}]}
```

Empty state (human): "No completed instances found." with a what/what-now hint.

Exit code: 0 always. See TDS-07 §4.

---

## logs

**Synopsis:** `awis logs [--tail=N] [--level=error|info|debug] [--instance=<id>]`

Read the runtime's structured log file at `<data-dir>/awis.log` (written by
`start`'s file-sink addition). Applies level/instance filters, then the
`--tail` line limit (default 50).

**Flags:**
- `--tail=<int>` — number of log lines to show (default 50)
- `--level=error|info|debug` — filter by log level
- `--instance=<id>` — filter to lines mentioning this instance ID

**--json:**
```json
{"log_file": ".awis/awis.log", "lines": [{"time": "...", "level": "info", "msg": "runtime started"}]}
```

Empty state (human): "No log file found at <path>" with a what-now hint to
run `awis start`.

Exit code: 0 always (missing log file is not an error). See TDS-07 §4.

---

## metrics

**Synopsis:** `awis metrics [--namespace=<ns>] [--workflow=<id>]`

Aggregate execution statistics from `workflow_instances`: counts by status,
avg/min/max duration for completed instances, and a per-workflow breakdown.

**Flags:**
- `--namespace=<ns>` — filter by namespace
- `--workflow=<id>` — filter by workflow ID

**Example output:**
```
METRICS  2026-07-10 14:23:01

Total instances: 12

By status:
  running                2
  completed              9
  failed                 1

Completed duration:
  Count: 9  Avg: 45.2s  Min: 12.0s  Max: 89.0s

By workflow:
  WORKFLOW                 TOTAL    COMPLETED  FAILED
  capture-decision         12       9          1
```

**--json:**
```json
{"namespace": "default", "by_status": {"completed": 9, "failed": 1},
 "total_instances": 12,
 "completed_metrics": {"count": 9, "avg_ms": 45200, "min_ms": 12000, "max_ms": 89000},
 "workflow_breakdown": [{"workflow_id": "capture-decision", "total": 12,
                          "completed": 9, "failed": 1}],
 "generated_at": "2026-07-10T14:23:01Z"}
```

Exit code: 0 always. See TDS-07 §4.

---

## recall

**Synopsis:** `awis recall "<query>" [--synthesize] [--limit=20]`

Full-text search over `execution_events` via the FTS5 index (migration
`0006_recall_fts.sql`).

**Flags:**
- `--synthesize` — route results through configured intelligence for a
  natural-language summary; with no intelligence configured (no
  `ANTHROPIC_API_KEY`), prints the PRD §21 empty-state message and falls
  back to raw FTS results
- `--limit=<int>` — maximum results (default 20)

**Example output:**
```
RECALL  "capture-decision"  (2 results)

  [1] i-a1b2c3  StepCompleted  2026-07-10T14:21:19Z
       step: draft-entry
       {"duration_ms":13100,...}
```

**--json:**
```json
{"query": "capture-decision", "synthesize": false,
 "results": [{"event_id": "...", "instance_id": "i-a1b2c3", "namespace": "oip",
              "event_type": "StepCompleted", "step_id": "draft-entry",
              "payload": "{...}", "emitted_at": "2026-07-10T14:21:19Z", "rank": 0.12}],
 "synthesis": null}
```

Exit code: 0 always (no matches is not an error), 1 if storage lacks the
RecallStore/FTS index. See TDS-07 §4.

---

## replay

**Synopsis:** `awis replay <instance-id>`

DRY-RUN only (PRD Should-Have): re-walks the recorded EventLog for an
instance and prints what WOULD run if it were re-executed. No state is
written; no steps are executed.

**Example output:**
```
REPLAY (dry-run)  capture-decision / i-a1b2c3
Status: completed  (original)

The following steps WOULD run if this instance were re-executed:

  00:00  ● WorkflowStarted              → initialize workflow state
  00:00  ► StepStarted  draft-entry     → dispatch step to worker
  00:13  ✓ StepCompleted  draft-entry   → record step result; advance to next step

NOTE: This is a dry-run. No steps were executed and no state was written.
```

**--json:**
```json
{"instance_id": "i-a1b2c3", "workflow_id": "capture-decision", "namespace": "oip",
 "status": "completed", "dry_run": true,
 "steps": [{"order": 1, "event_type": "WorkflowStarted", "emitted_at": "...",
            "relative_ms": 0, "action": "initialize workflow state"}]}
```

Exit code: 0 found, 1 instance not found or storage error. See TDS-07 §4.

---

## audit

**Synopsis:** `awis audit [--limit=50]`

Read the `audit_log` table (via `storage.RecallStore.ListAudit`): every
`WorkflowRegistered`, `ConfigChanged`, `PluginRegistered`, `PluginRemoved`
row written by F-4 write-sites.

**Flags:**
- `--limit=<int>` — maximum number of entries (default 50)

**Example output:**
```
AUDIT LOG (last 50)

  ID   TIMESTAMP                    EVENT TYPE               ACTOR      SUMMARY
  1    2026-07-10T14:20:00Z         WorkflowRegistered        awis       capture-decision v1.0.0
```

**--json:**
```json
{"entries": [{"id": 1, "timestamp": "2026-07-10T14:20:00Z",
              "event_type": "WorkflowRegistered", "actor": "awis",
              "payload_summary": "capture-decision v1.0.0"}]}
```

Exit code: 0 always. See TDS-07 §4.

---

## config show

**Synopsis:** `awis config show`

Print the current `<data-dir>/config.yaml` contents as key: value pairs.
Secrets (`anthropic_api_key`, `api_key`) are masked as `***` (NFR-S-01).

**--json:**
```json
{"config_file": ".awis/config.yaml", "keys": {"namespace": "default", "tick": "100ms"}}
```

Exit code: 0 always (a missing config file prints a what-now hint, not an error).
See TDS-07 §4.

---

## config set

**Synopsis:** `awis config set <key> <value>`

Set (or append) a `key: value` line in `<data-dir>/config.yaml`, then write
a `ConfigChanged` audit row (F-4). Secret keys are masked in both the
confirmation output and the audit payload.

**Example output:**
```
Set: namespace = oip
Written to: .awis/config.yaml
```

**--json:**
```json
{"config_file": ".awis/config.yaml", "key": "namespace", "value": "oip", "masked": false}
```

Exit code: 0 written, 2 missing arguments, 1 write error. See TDS-07 §4.

---

## config validate

**Synopsis:** `awis config validate`

Schema-check `<data-dir>/config.yaml`: every `key:` must be one of the
recognized V1 keys (`namespace`, `tick`, `anthropic_api_key`, `api_key`,
`data_dir`, `log_level`, `intelligence`, `plugins_dir`, `plugins_user`);
every line must have a colon. `plugins_user` (username, or numeric
`uid[:gid]`) isolates plugin subprocesses from `runtime.db`; setting it
requires the engine to hold `CAP_SETUID`+`CAP_SETGID` or run as root, and
plugin spawn fails explicitly if that privilege is absent (D-11).

**Example output (valid):**
```
.awis/config.yaml  valid
```

**Example output (invalid):**
```
awis: config validation failed: .awis/config.yaml
  unknown config key: "bogus_key"
```

**--json:**
```json
{"config_file": ".awis/config.yaml", "valid": false, "errors": ["unknown config key: \"bogus_key\""]}
```

Exit code: 0 valid, 1 invalid or file not found. See TDS-07 §4.

---

## config edit

**Synopsis:** `awis config edit`

Open `<data-dir>/config.yaml` in `$EDITOR` (or `$VISUAL`); creates a
default config first if none exists. Skipped in CI (no `$EDITOR` set).

Exit code: 0 editor exited cleanly, 1 no `$EDITOR` set or editor error.
See TDS-07 §4.

---

## rebuild-state

**Synopsis:** `awis rebuild-state --all` (or `awis rebuild-state <instance-id>`,
which V1 treats identically to `--all`)

Rebuild the `workflow_instances` projection from the append-only EventLog
(M03 path; EDR-007 algorithm). Intended to be run against a stopped runtime.

**Example output:**
```
rebuild-state: projection rebuilt from EventLog
  Mode:    all
  DB:      .awis/runtime.db

  What now: awis start  (resume engine; rebuild is safe before start)
```

**--json:**
```json
{"mode": "all", "success": true, "message": "projection rebuilt from EventLog"}
```

Exit code: 0 rebuilt, 1 storage error or projection rebuild unsupported,
2 missing `--all`/`<instance-id>`. See TDS-07 §4.

---

## export

**Synopsis:** `awis export [--out=<dir>] [--namespace=<ns>]`

Export all workflow instances and workflow definitions from `runtime.db` to
`instances.json` and `definitions.json` under `--out` (default: current
directory) — PRD §25 portability.

**Example output:**
```
Export complete  2026-07-10 14:23:01

  Instances:   12 → /abs/path/instances.json
  Definitions: 2 → /abs/path/definitions.json
```

**--json:**
```json
{"out_dir": "/abs/path", "instances_file": "/abs/path/instances.json",
 "definitions_file": "/abs/path/definitions.json", "instance_count": 12,
 "definition_count": 2, "exported_at": "2026-07-10T14:23:01Z"}
```

Exit code: 0 always on success, 1 on I/O or storage error. See TDS-07 §4.

---

## prune-events

**Synopsis:** `awis prune-events --dry-run [--before=<date>]`

TTL scan report only in V1 (F-2's 7-day `domain_events` TTL); destructive
pruning is post-V1 (PRD Should-Have). `--dry-run` is required.

**Flags:**
- `--dry-run` — required; report only, no deletion
- `--before=<date>` — cutoff (RFC3339 or `YYYY-MM-DD`); default 7 days ago

**Example output:**
```
PRUNE-EVENTS (dry-run)

  Cutoff:         2026-07-03T14:23:01Z
  Events scanned: 420
  Eligible:       58  (would be pruned)

  NOTE: Destructive prune is post-V1. No data was deleted.
        Re-run without --dry-run when the feature ships.
```

**--json:**
```json
{"dry_run": true, "before": "2026-07-03T14:23:01Z", "eligible_count": 58,
 "total_scanned": 420, "note": "Destructive prune is post-V1. This is a dry-run report only.",
 "scanned_at": "2026-07-10T14:23:01Z"}
```

Exit code: 2 if `--dry-run` is omitted, 0 otherwise. See TDS-07 §4.

---

## init

**Synopsis:** `awis init [--force] [directory]`

Scaffold a new AWIS project (FR-RM-01) from an embedded file set
(`//go:embed all:scaffold`) into `[directory]` (default: current
directory): `config.yaml`, `.gitignore`, the three M10 example workflows
under `workflows/` (`hello-world.yaml`, `with-signal.yaml`,
`with-intelligence.yaml` — byte-identical to `examples/workflows/`),
`handlers/example_handler.go` (stub handlers for an embedding Go program to
register), and `README_AWIS.md`.

Refuses to write into a non-empty target directory unless `--force` is given.

**Flags:**
- `--force` — overwrite files in a non-empty target directory

**Example output:**
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

**--json:**
```json
{"target": "/abs/path/my-project",
 "files": ["config.yaml", ".gitignore", "workflows/hello-world.yaml",
           "workflows/with-signal.yaml", "workflows/with-intelligence.yaml",
           "handlers/example_handler.go", "README_AWIS.md"]}
```

Note: the scaffolded `workflows/hello-world.yaml` registers under namespace
`examples` (its own YAML `namespace:` field); target it with
`awis --namespace=examples submit hello-world`, since `submit`'s
cross-process lookup filters by the CALLING process's `--namespace`, not
the definition's namespace.

Exit code: 0 scaffolded, 1 non-empty target without `--force` or I/O error.
See TDS-07 §4.

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
