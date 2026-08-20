# TDS-04 — Subprocess Protocol

**Status:** Canonical · frozen at M11-C1  
**Authority:** IMPLEMENTATION_SPEC.md §1 (CE-pinned decisions); Blueprint §25  
**Coordinates:** IMP §12 TDS-04 row; PRD FR-SE-02, FR-SDK-10  
**Companion:** `internal/runner/subprocess/` (M11-C2); `python/awis-step` (M11-C3)

---

## Scope

This document defines the wire protocol between the AWIS Go runtime (`SubprocessRunner`) and
any external step handler process. The protocol is **language-neutral**: a compliant handler
may be written in Python, TypeScript, shell, or any other language that can read JSON from
stdin and write JSON to stdout.

**Python reference client:** `python/awis-step` (`awis-step` pip package) implements the full
client side of this protocol; its `@step` decorator and `step.serve()` function are the
canonical V1 reference implementation (Blueprint §25).

**One-shot model:** the runtime spawns one process per step execution. It writes exactly one
JSON request object to the child's stdin and then closes stdin. The child writes exactly one
JSON response object to stdout and exits. Long-lived / persistent processes are M12 plugin
territory and are out of scope here.

---

## 1. Framing Rules

- Each direction carries a **single JSON object**, UTF-8 encoded, terminated by a newline (`\n`).
- The runtime writes the request object to child stdin, then **closes stdin**.
- The child writes the response object to stdout and **exits**.
- Child **stderr** is free-form log output: the runtime captures up to 4 KiB of the tail for
  error details but **never parses stderr as protocol**. Handlers may write arbitrary diagnostic
  text to stderr without affecting the exchange.

---

## 2. Request Envelope

The runtime sends exactly this JSON object. Field names and ordering are frozen (CE-pinned).

```json
{
  "protocol": "awis-subprocess/1",
  "handler":  "<step.handler>",
  "instance_id": "…",
  "step_id": "…",
  "attempt": 1,
  "inputs": {},
  "deadline": "<RFC3339Nano>"
}
```

| Field | Type | Required | Source |
|---|---|---|---|
| `protocol` | string | yes | literal `"awis-subprocess/1"` |
| `handler` | string | yes | `core.Step.Handler` — the step's handler reference |
| `instance_id` | string | yes | `core.StepContext.InstanceID` |
| `step_id` | string | yes | `core.StepContext.StepID` |
| `attempt` | integer | yes | `core.StepContext.Attempt` (1-based) |
| `inputs` | object | yes | `core.StepContext.Inputs` (resolved step inputs) |
| `deadline` | string (RFC3339Nano) | no | `core.StepContext.Deadline` serialized; **omitted** when the step has no timeout |

`deadline` gives the handler the absolute wall-clock time by which it must produce a response.
Handlers may use this value to self-limit (e.g., abort expensive computation early).

---

## 3. Success Response

A handler that completes successfully writes this JSON object to stdout:

```json
{
  "protocol": "awis-subprocess/1",
  "outputs": {}
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `protocol` | string | yes | must be `"awis-subprocess/1"` |
| `outputs` | object | yes | key/value map of step outputs |

---

## 4. Error Response

A handler that detects an application-level error writes this JSON object to stdout:

```json
{
  "protocol": "awis-subprocess/1",
  "error": {
    "code": "…",
    "message": "…",
    "details": {}
  }
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `protocol` | string | yes | must be `"awis-subprocess/1"` |
| `error.code` | string | yes | stable error class; empty or missing → runtime maps to `handler_error` |
| `error.message` | string | yes | human-readable description |
| `error.details` | object | no | optional handler-supplied context |

The `error` object maps verbatim onto `core.StepError` (TDS-01 §2.1), which is carried in
`StepFailed`, `WorkflowFailed`, and `WorkflowCompensationFailed` event payloads.

---

## 5. Handler → argv Rule

`step.handler` is **whitespace-split** into argv and exec'd directly — **no shell
interpretation**. Example: handler `"python my_step.py"` becomes argv
`["python", "my_step.py"]`; the runtime calls `exec` (not `/bin/sh -c`). This avoids shell
injection and is consistent with Blueprint §25 ("spawns this as `python my_step.py`").

---

## 6. Timeout Behavior

`step.Timeout` is parsed with `time.ParseDuration` (the same rule `internal/runner/native`
uses — coordinate: `internal/runner/native`). When the step has a timeout, the runtime:

1. Computes an absolute deadline from `time.Now().Add(timeout)`.
2. Serializes the deadline as `deadline` (RFC3339Nano) in the request envelope so the handler
   can self-limit.
3. Kills the **process group** with `SIGKILL` if the child has not exited by the deadline,
   ensuring all descendants are reaped.
4. Returns a `timeout` error (`core.StepError{Code: "timeout", …}`), matching the semantics of
   `NativeRunner`'s timeout mapping.

If the context already carries a deadline from the engine, the earlier of the two deadlines
applies.

---

## 7. Error Mapping (Runtime Side)

These are the stable error codes the runtime produces on its own. Application error codes come
from the handler's error envelope and are passed through unchanged.

| Condition | Runtime error code | Notes |
|---|---|---|
| Spawn failure (exec not found, not executable) | `spawn_error` | No child process was started |
| Deadline exceeded | `timeout` | Runtime kills process group with SIGKILL; same semantics as NativeRunner |
| stdout is not one valid JSON envelope, or `protocol` value is wrong | `protocol_error` | `details` carries stderr tail (≤4 KiB) and exit code |
| Nonzero exit **with** a valid error envelope on stdout | *(envelope wins)* | The envelope's `code`/`message`/`details` are used verbatim |
| Nonzero exit **without** a valid envelope on stdout | `subprocess_error` | `details` carries exit code and stderr tail (≤4 KiB) |

**Envelope-wins rule:** if the child exits with a nonzero code but stdout contains a valid,
well-formed error envelope, that envelope's `code`, `message`, and `details` are used as the
`core.StepError`. The nonzero exit alone does not produce `subprocess_error` in this case.

---

## 8. stderr Semantics

stderr is **free-form**. The runtime captures the tail (≤4 KiB) for inclusion in error
`details` on failure paths (`protocol_error`, `subprocess_error`). On the success path, stderr
content is discarded (not surfaced). Handlers may write progress logs, warnings, or debug
output to stderr at any time without affecting protocol correctness.

---

## 9. Golden Protocol Files

The canonical wire examples live at
`internal/runner/subprocess/testdata/protocol/`. They are the single wire truth shared by the
Go conformance tests (M11-C2) and the Python pytest suite (M11-C3).

### `request-basic.json` — minimal request (no deadline)

Intent: a request envelope for a step that has no timeout configured.

```json
{
  "protocol": "awis-subprocess/1",
  "handler": "python my_step.py",
  "instance_id": "inst-001",
  "step_id": "compute-hrv",
  "attempt": 1,
  "inputs": {
    "records": [1, 2, 3]
  }
}
```

### `request-deadline.json` — request with deadline

Intent: a request envelope for a step that has a timeout; `deadline` is present.

```json
{
  "protocol": "awis-subprocess/1",
  "handler": "python my_step.py",
  "instance_id": "inst-001",
  "step_id": "compute-hrv",
  "attempt": 1,
  "inputs": {
    "records": [1, 2, 3]
  },
  "deadline": "2099-01-01T00:00:00.000000000Z"
}
```

### `response-ok.json` — success response

Intent: a well-formed success response written by a handler to stdout.

```json
{
  "protocol": "awis-subprocess/1",
  "outputs": {
    "hrv": 42.5
  }
}
```

### `response-error.json` — application error response

Intent: a well-formed error response written by a handler to stdout.

```json
{
  "protocol": "awis-subprocess/1",
  "error": {
    "code": "validation_error",
    "message": "records field must not be empty",
    "details": {
      "field": "records"
    }
  }
}
```

### `bad-protocol.json` — wrong protocol value

Intent: a response whose `protocol` field does not equal `"awis-subprocess/1"`; the runtime
must reject this with `protocol_error`.

```json
{
  "protocol": "awis-subprocess/0",
  "outputs": {}
}
```

### `bad-truncated.json` — syntactically invalid JSON

Intent: a response that is not valid JSON (simulates a handler that crashes mid-write or
writes garbage); the runtime must reject this with `protocol_error`.

```json
{"protocol":"awis-subprocess/1","outputs":
```
