# AWIS Plugin Developer Guide

**Protocol authority:** `docs/PLUGIN_PROTOCOL.md` (TDS-05)
**Library:** `python/awis-plugin` (`awis-plugin` pip package)
**Milestone:** M12-C4

---

## Overview

An AWIS plugin is a **long-lived process** that speaks JSON-RPC 2.0 over stdin/stdout.
The runtime spawns it once and routes multiple step executions to it over a persistent
channel. This document covers everything needed to author, test, and operate a plugin
using the Python reference library. For the full wire protocol see TDS-05.

---

## 1. Manifest Authoring

Every plugin declares a manifest file named `awis-plugin.yaml` placed beside the
entry module. The manifest specifies the plugin's identity, capabilities, and how
the runtime spawns it.

**Minimal example:**

```yaml
name: my-plugin
version: 1.0.0
description: A minimal AWIS plugin
author: yourname

capabilities:
  - id: my.capability.hello
    inputs:
      name: string
    outputs:
      greeting: string
    timeout_ms: 5000

runtime:
  command: python3
  args: ["-m", "my_plugin"]
  idle_timeout_s: 300
```

**Manifest validation rules (TDS-05 §9):**

| Rule | Detail |
|---|---|
| `name` non-empty | Plugin identity; used as `plugin_id` in V1 |
| `version` semver | Must match `^(0\|[1-9]\d*)\.(0\|[1-9]\d*)\.(0\|[1-9]\d*)` |
| At least one capability | A plugin must declare at least one capability |
| Capability `id` unique | No two capabilities may share the same `id` |
| `runtime.command` non-empty | Required to spawn the process |

**Environment (NFR-S-02):** the process receives only the `env` keys from the manifest
plus `PATH`. No other parent environment variables are inherited.

---

## 2. `@plugin.capability` Decorator

Register a handler function for a capability id:

```python
from awis_plugin import plugin

@plugin.capability(id="my.capability.hello")
def hello(inputs: dict) -> dict:
    return {"greeting": f"Hello, {inputs['name']}!"}
```

- The decorated function receives `inputs` as a plain `dict`.
- Return a `dict` of outputs; the keys must match the manifest `outputs` schema.
- Raise any `Exception` to signal a handler error; the library captures the traceback
  and returns a JSON-RPC error with `data.code = "handler_error"`.

Multiple capabilities can be registered in the same module:

```python
@plugin.capability(id="my.capability.add")
def add(inputs: dict) -> dict:
    return {"sum": inputs["a"] + inputs["b"]}

@plugin.capability(id="my.capability.mul")
def mul(inputs: dict) -> dict:
    return {"product": inputs["a"] * inputs["b"]}
```

---

## 3. `plugin.serve()`

Call `plugin.serve()` at the bottom of your entry module to start the NDJSON
JSON-RPC loop:

```python
if __name__ == "__main__":
    plugin.serve()
```

**What `serve()` does (TDS-05 §2–§4):**

1. **Handshake** — on first message the runtime sends a handshake request;
   `serve()` reads `awis-plugin.yaml` beside the entry module and echoes
   `{"name", "version", "capabilities"}` as the result. The runtime validates
   that `name` and `version` match the registered manifest and that all declared
   capability ids are present.

2. **Execute** — for each step execution the runtime sends an execute request with
   `params.capability`, `params.inputs`, `params.step_id`, and `params.timeout_ms`.
   `serve()` dispatches to the registered handler and returns
   `{"outputs": ..., "duration_ms": ...}` on success, or a JSON-RPC error on
   failure.

3. **Shutdown** — the runtime sends a `shutdown` notification (no `id`); `serve()`
   flushes pending state and exits with code 0.

**`manifest_path` argument:** by default `serve()` searches for `awis-plugin.yaml`
beside the entry module (`sys.argv[0]`). Pass an explicit path when the file is
elsewhere:

```python
plugin.serve(manifest_path="/opt/my-plugin/awis-plugin.yaml")
```

---

## 4. Offline Testing with `mock_request`

The `awis_plugin.testing` module provides offline test helpers that exercise the
SAME dispatch path as `serve()` — no subprocess, no stdin/stdout I/O, no runtime
required (FR-PS-15).

```python
from awis_plugin import plugin
from awis_plugin.testing import mock_request, mock_handshake, PluginCallError

# Register a capability in your plugin module (import it first).
import my_plugin  # this imports and registers @plugin.capability handlers

# Test a happy-path call.
outputs = mock_request("my.capability.hello", {"name": "World"})
assert outputs["greeting"] == "Hello, World!"

# Test the handshake echo.
manifest = mock_handshake(manifest_path="path/to/awis-plugin.yaml")
assert manifest["name"] == "my-plugin"
assert any(c["id"] == "my.capability.hello" for c in manifest["capabilities"])

# Test error handling.
try:
    mock_request("nonexistent.capability", {})
except PluginCallError as e:
    assert e.code == "capability_not_found"
```

**`mock_request(capability, inputs, plugin=plugin)`**

- Invokes the registered handler through the same `_dispatch()` function `serve()` uses.
- Returns `dict` outputs on success.
- Raises `PluginCallError` (with `.code`, `.detail`, `.data`) on error.

**`mock_handshake(manifest_path=None)`**

- Returns the manifest echo dict `{"name", "version", "capabilities"}`.
- No process is started.

**`PluginCallError`**

- `.code` — matches `data.code` from the JSON-RPC error (`"capability_not_found"` or `"handler_error"`).
- `.detail` — human-readable detail string.
- `.data` — full error data dict (includes `"traceback"` on `handler_error`).

---

## 5. Lifecycle Expectations

**Spawn** — the runtime spawns the plugin process on demand (lazy spawn) at the
first step dispatched to it. Cold-start time should be minimised; the handshake
must complete within 5 seconds (TDS-05 §2).

**Idle kill** — if no `execute` is received for `idle_timeout_s` seconds, the
runtime kills the process group. The plugin is respawned transparently on the next
call (expected < 2 s). Do not rely on in-process state surviving across idle kills.

**Auto-restart on crash** — if the process exits unexpectedly the runtime auto-restarts
it on the next call (up to 3 consecutive crashes). After 4 consecutive crashes the
plugin enters `FAILED` state and all calls return `plugin_failed` until re-registration.
The crash counter resets after any successful `execute`.

**Shutdown** — the runtime sends a `shutdown` notification and waits ≤ 2 seconds for
the process to exit cleanly, then sends SIGKILL to the process group.

---

## 6. Security Boundary — What Plugins Cannot Do

From Blueprint §11 (TDS-05 §10, verbatim):

- Access the EventLog or StateStore directly (they only receive step inputs and return step outputs)
- Register new workflow definitions (only applications can do this via the SDK)
- Hold state between calls (plugin state must be external to the plugin process)
- Access another namespace's data

The runtime enforces minimal environment inheritance (manifest `env` + `PATH` only;
NFR-S-02) and process group isolation (SIGKILL reaps all descendants).

---

## 7. Install Pointer

In V1, plugin installation is manual: place the `awis-plugin.yaml` manifest and
the plugin module on the target host, then call `Manager.Register(manifestPath)` from
your application code or use the AWIS SDK runtime.

A CLI command (`awis plugin install`) is planned for M14 (F-5). Full marketplace
support is V2 (FR-PS-11/12).

---

## 8. Protocol Reference

All wire-format details, golden examples, error codes, and lifecycle FSM transitions
are specified in **TDS-05** at `docs/PLUGIN_PROTOCOL.md`.

Quick reference:

| Method | Direction | Notes |
|---|---|---|
| `handshake` | runtime → plugin | Sent immediately after spawn; 5 s deadline |
| `execute` | runtime → plugin | One per step dispatch; carries `capability`, `inputs`, `timeout_ms` |
| `shutdown` | runtime → plugin | Notification (no `id`); plugin must exit cleanly |

Error codes returned by plugins (`data.code`):

| Code | Cause |
|---|---|
| `capability_not_found` | No handler registered for the requested capability |
| `handler_error` | Handler raised an unexpected exception (traceback in `data`) |
