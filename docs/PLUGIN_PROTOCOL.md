# TDS-05 — Plugin Protocol

**Status:** Canonical · frozen at M12-C1
**Authority:** IMPLEMENTATION_SPEC.md §1 (CE-pinned decisions); Blueprint §11
**Coordinates:** IMP §12 TDS-05 row; PRD FR-PS-01..06, FR-PS-14, FR-PS-15; NFR-S-02
**Companion:** `internal/plugin/` (M12-C1/C3); `python/awis-plugin` (M12-C4)

---

## Scope

This document defines the wire protocol between the AWIS Go runtime (Plugin Manager) and
any external plugin process. The protocol is **language-neutral**: a compliant plugin may
be written in Python, Go, or any other language that can read and write newline-delimited
JSON.

**Long-lived model:** unlike the one-shot subprocess protocol (TDS-04), a plugin process
is **long-lived**. The runtime spawns a plugin once and then routes multiple step
executions to it over a persistent stdin/stdout pair via JSON-RPC 2.0.

**Runtime is the client:** the runtime sends JSON-RPC requests; the plugin only responds.
The plugin never initiates messages.

**Python reference library:** `python/awis-plugin` (`awis-plugin` pip package) implements
the full plugin-side protocol; its `@plugin.capability` decorator and `plugin.serve()`
function are the canonical V1 Python reference implementation.

---

## 1. Framing Rules

- **Framing:** newline-delimited JSON-RPC 2.0 objects (NDJSON), UTF-8, over a **long-lived
  stdin/stdout pair**.
- Each message is a single JSON object terminated by a newline (`\n`).
- **stderr** is free-form logs, captured bounded (4 KiB tail), **never protocol**. Plugins
  may write arbitrary diagnostic text to stderr at any time.
- **Direction:** runtime is the JSON-RPC client; the plugin only responds.
- **Request ids** are strings (`"req-<n>"`, monotonic per process).
- **Notifications** (shutdown) carry no `id` field.

---

## 2. Handshake Method

Sent **immediately after spawn**; response deadline **5 seconds**.

### Request

```json
{
  "jsonrpc": "2.0",
  "id": "req-1",
  "method": "handshake",
  "params": {
    "protocol_version": "awis-plugin/1"
  }
}
```

| Field | Type | Value |
|---|---|---|
| `jsonrpc` | string | `"2.0"` |
| `id` | string | `"req-<n>"` (monotonic) |
| `method` | string | `"handshake"` |
| `params.protocol_version` | string | `"awis-plugin/1"` |

### Response (success)

```json
{
  "jsonrpc": "2.0",
  "id": "req-1",
  "result": {
    "name": "git-context-plugin",
    "version": "1.0.0",
    "capabilities": [
      {
        "id": "git.context.assemble"
      },
      {
        "id": "git.diff.fetch"
      }
    ]
  }
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `result.name` | string | yes | Must equal the registered manifest `name` |
| `result.version` | string | yes | Must equal the registered manifest `version` |
| `result.capabilities` | array | yes | Each element must have an `id`; the set must be a **superset** of the registered capability ids |

**Validation:** if `name`, `version`, or capability id set fails the above checks, or if
the deadline (5 s) elapses without a response, the runtime treats this as a
crash-equivalent (see §8 error mapping: `plugin_handshake_error`; counts as a crash).

### Golden file

`internal/plugin/testdata/protocol/handshake-request.json`,
`internal/plugin/testdata/protocol/handshake-response.json`

---

## 3. Execute Method

Sent for each step execution dispatched to this plugin.

### Request

```json
{
  "jsonrpc": "2.0",
  "id": "req-2",
  "method": "execute",
  "params": {
    "capability": "git.context.assemble",
    "step_id": "assemble-context",
    "inputs": {
      "repo_path": "/path/to/repo",
      "ref": "abc123"
    },
    "timeout_ms": 30000
  }
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `jsonrpc` | string | yes | `"2.0"` |
| `id` | string | yes | `"req-<n>"` (monotonic per process) |
| `method` | string | yes | `"execute"` |
| `params.capability` | string | yes | The resolved capability id (always explicit; TDS-05 is the normative home — see TRACEABILITY disposition) |
| `params.step_id` | string | yes | `core.StepContext.StepID` |
| `params.inputs` | object | yes | Resolved step inputs |
| `params.timeout_ms` | integer | yes | Effective remaining budget in milliseconds (see §5) |

**Note:** the Blueprint §11 example omits `capability` (illustrative prose). TDS-05 is the
designated normative home (IMP §12) and **requires** it. Recorded in TRACEABILITY
dispositions.

### Response (success)

```json
{
  "jsonrpc": "2.0",
  "id": "req-2",
  "result": {
    "outputs": {
      "context": {}
    },
    "duration_ms": 450
  }
}
```

| Field | Type | Notes |
|---|---|---|
| `result.outputs` | object | Key/value step outputs |
| `result.duration_ms` | integer | Wall-clock time the plugin spent executing |

### Response (error)

```json
{
  "jsonrpc": "2.0",
  "id": "req-2",
  "error": {
    "code": -32000,
    "message": "handler_error",
    "data": {
      "code": "capability_not_found",
      "detail": "no handler registered for capability"
    }
  }
}
```

| Field | Type | Notes |
|---|---|---|
| `error.code` | integer | JSON-RPC error code |
| `error.message` | string | Human-readable summary |
| `error.data` | object | Plugin-supplied detail; `data.code` used as `StepError.Code` when present |

### Golden files

`internal/plugin/testdata/protocol/execute-request.json`,
`internal/plugin/testdata/protocol/execute-response.json`,
`internal/plugin/testdata/protocol/execute-error.json`

---

## 4. Shutdown Notification

Sent by the runtime when it wants the plugin to exit cleanly. This is a JSON-RPC
**notification** (no `id` field). The plugin should flush any pending state and exit.

```json
{
  "jsonrpc": "2.0",
  "method": "shutdown"
}
```

The runtime waits **≤ 2 seconds** for the process to exit, then sends **SIGKILL** to the
process group.

### Golden file

`internal/plugin/testdata/protocol/shutdown-notification.json`

---

## 5. Effective Call Timeout

The `timeout_ms` parameter in an execute request carries the **effective remaining budget**
in milliseconds, computed as:

```
timeout_ms = min(step ctx deadline, step.Timeout if set, capability timeout_ms)
```

Plugins must use this value to self-limit. If the plugin does not respond within
`timeout_ms`, the runtime kills the process group with SIGKILL and returns a `timeout`
error (see §8). A timeout kill does **not** increment the crash counter (controlled kill).

---

## 6. Capability Resolution Rule

A plugin step's `handler` field is resolved by the runtime as follows (frozen fixture
compatibility — Blueprint §7 fixture uses `handler: git-context-plugin`):

1. **Capability id (exact match):** if `handler` matches an id in `plugin_capabilities`,
   it is used directly. The execute request carries this id as `capability`.

2. **Plugin name:** if `handler` matches a plugin `name`, the runtime selects the unique
   capability whose declared input KEY SET equals the step's resolved input key set.
   - **Zero matches or multiple matches** → typed `StepError` with code
     `plugin_capability_ambiguous`. Remedy: use the explicit capability id as `handler`.

The execute request **always** carries the resolved explicit `capability` field.

---

## 7. Plugin Lifecycle

### States

From Blueprint §11 (verbatim):

```
REGISTER     Plugin declares capabilities in awis-plugin.yaml
SPAWN        Runtime spawns plugin process with environment
HANDSHAKE    Plugin sends Manifest; runtime validates capabilities
ACTIVE       Runtime routes steps to plugin via JSON-RPC
IDLE         Plugin receives no requests for idle_timeout; paused
TERMINATE    Runtime sends shutdown; plugin exits cleanly
RESTART      On crash: runtime detects exit code; auto-restarts (max 3x)
```

**In-memory states (FSM):** `REGISTERED`, `SPAWNING`, `HANDSHAKING`, `ACTIVE`, `IDLE`,
`TERMINATED`, `FAILED`.

**DB `plugins.status`** stays coarse: `registered | active | failed`. In-memory `IDLE`
keeps DB status `active`. `suspended` (Blueprint SQL comment) is unused in V1.

### Transitions (CE-pinned)

**Registration** (`Manager.Register(manifestPath)`)

- Parse and validate manifest → write DB rows (`plugins` + `plugin_capabilities`;
  `plugin_id` = manifest `name`; V1 pins one installed version per name) → emit
  `PluginRegistered` audit event (F-4) → state: `REGISTERED`.
- Re-registering the same name = upsert rows + new audit event.

**Lazy spawn** (`ensureActive()`)

- `REGISTERED` / `IDLE` → `SPAWNING`: exec manifest `runtime.command` + `args`; env =
  manifest `env` + `PATH` only (NFR-S-02 minimal env); no extra file descriptors
  (os/exec default; NFR-S-02); process group `Setpgid`.
- `SPAWNING` → `HANDSHAKING`: send handshake request; 5 s deadline.
- `HANDSHAKING` → `ACTIVE`: handshake validated; DB status → `active`.

**Call serialization:** calls are serialized per plugin (mutex; V1 pin). Cross-plugin
concurrency is unaffected.

**Crash** (unexpected process exit detected by monitor goroutine)

- In-flight call fails with `plugin_crash`.
- Consecutive-crash counter += 1.
- Counter > 3 → state `FAILED`; DB status `failed`; all further calls fail
  `plugin_failed` until re-registration.
- Counter ≤ 3 → state `REGISTERED` (auto-restart on next call).
- Counter **resets** on a successful `execute` completion.
- Handshake failures count as crashes.

**IDLE**

- No `execute` for `idle_timeout_s` seconds → kill process group; state `IDLE`; DB
  status stays `active`.
- Next call respawns transparently (< 2 s expected).
- Idle watchdog is a per-plugin timer; idle kill must never fire while a call is in
  flight (mutex covers both).

**Shutdown** (`Manager.Shutdown(ctx)`)

- For each active plugin: send `shutdown` notification; wait ≤ 2 s; SIGKILL process
  group; state `TERMINATED`.
- Idempotent.

### Invariants (adversarial review targets)

1. No call may be dispatched to a process in any state but `ACTIVE`.
2. A crash during `HANDSHAKING` must not deadlock waiters.
3. The idle killer must never kill a process with a call in flight (mutex covers both).
4. Counter transitions and state transitions are atomic under one lock.
5. Monitor goroutines never leak past `Shutdown`.

---

## 8. Error Mapping (Runtime Side)

These are the stable error codes the runtime produces. Plugin-supplied codes come from
`error.data.code` and are passed through unchanged.

| Condition | Runtime error code | Notes |
|---|---|---|
| Spawn failure (exec not found, not executable, etc.) | `spawn_error` | No child process started |
| Handshake failure or timeout (5 s) | `plugin_handshake_error` | Counts as crash |
| Process exit while call in-flight | `plugin_crash` | Counts as crash |
| Call deadline exceeded | `timeout` | Controlled kill; does NOT count as crash |
| JSON-RPC error response from plugin | `plugin_error` (or `data.code` when plugin supplies one) | `message`/`data` passed verbatim |
| Plugin in `FAILED` state (crash budget exhausted) | `plugin_failed` | Permanent until re-registration |
| Unparseable stdout line or id mismatch | `protocol_error` | Crash-equivalent: kill + count |

---

## 9. Manifest Format

The canonical manifest format is defined by the Blueprint §11 YAML below (verbatim oracle).
This is the reference for `internal/plugin/manifest.go` field names and validation rules.

```yaml
# awis-plugin.yaml
name: git-context-plugin
version: 1.0.0
description: Assembles git context for capture workflows
author: awis

capabilities:
  - id: git.context.assemble
    inputs:
      repo_path: string
      ref: string
    outputs:
      context: object
    timeout_ms: 30000

  - id: git.diff.fetch
    inputs:
      repo_path: string
      from_ref: string
      to_ref: string
    outputs:
      diff: string
    timeout_ms: 10000

runtime:
  command: "python"
  args: ["-m", "git_context_plugin"]
  env:
    GIT_TERMINAL_PROMPT: "0"
  idle_timeout_s: 300
```

### Manifest validation rules (SPEC §1)

| Rule | Detail |
|---|---|
| `name` non-empty | Plugin registration identity; `plugin_id` = `name` in V1 |
| `version` semver | Must match `^(0\|[1-9]\d*)\.(0\|[1-9]\d*)\.(0\|[1-9]\d*)` |
| ≥ 1 capability | Plugins must declare at least one capability |
| Capability `id` unique | No two capabilities may share the same `id` |
| `runtime.command` non-empty | Required to spawn the process |

### Parsed struct (Go)

```go
type Manifest struct {
    Name         string
    Version      string
    Description  string
    Author       string
    Capabilities []Capability
    Runtime      Runtime
}

type Capability struct {
    ID        string
    Inputs    map[string]any
    Outputs   map[string]any
    TimeoutMS int
}

type Runtime struct {
    Command      string
    Args         []string
    Env          map[string]string
    IdleTimeoutS int
}
```

Unknown YAML fields are rejected with a file+line error (yaml.v3 `KnownFields(true)`).

---

## 10. Security (NFR-S-02)

**Minimal environment:** the plugin process inherits **only** `env` from the manifest plus
`PATH`. No other environment variables from the parent runtime process are inherited.

**No file descriptor inheritance:** the process is spawned with no extra open file
descriptors (os/exec default, NFR-S-02).

**Process group isolation:** each plugin process is started in its own process group
(`Setpgid`). Kills use `SIGKILL` to the process group to ensure all descendants are
reaped.

### What Plugins Cannot Do

From Blueprint §11 (verbatim):

- Access the EventLog or StateStore directly (they only receive step inputs and return step outputs)
- Register new workflow definitions (only applications can do this via the SDK)
- Hold state between calls (plugin state must be external to the plugin process)
- Access another namespace's data

---

## 11. Golden Protocol Files

The canonical wire examples live at `internal/plugin/testdata/protocol/`. They are the
single wire truth shared by the Go conformance tests (M12-C3) and the Python pytest suite
(M12-C4).

| File | Description |
|---|---|
| `handshake-request.json` | Runtime → plugin handshake request |
| `handshake-response.json` | Plugin → runtime handshake response (manifest echo) |
| `execute-request.json` | Runtime → plugin execute request (Blueprint §11 example values + explicit `capability`) |
| `execute-response.json` | Plugin → runtime execute success response |
| `execute-error.json` | Plugin → runtime execute JSON-RPC error response |
| `shutdown-notification.json` | Runtime → plugin shutdown notification (no `id`) |
| `bad-jsonrpc.json` | Invalid JSON-RPC: missing `jsonrpc` field; runtime must reject with `protocol_error` |
