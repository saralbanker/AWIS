# M11 — Implementation Spec
**Canonical sources:** IMP §27.M11; Blueprint §25 (CODE SDK ARCHITECTURE: language strategy,
Python subprocess protocol, the `awis_step` example verbatim); TDS-04 `docs/SUBPROCESS_PROTOCOL.md`
(written day 1 — THIS milestone authors it; IMP §12 row: "stdin/stdout JSON exchange for
subprocess steps: request/response envelope, error mapping, timeout behavior"); PRD FR-SE-02,
FR-SDK-10; IMP §5 (paths: `internal/runner/subprocess`, `python/awis-step`); IMP §18 L325
(published to repo only in V1; pip-installable from path).

## Scope
TDS-04 protocol document + golden wire files (single wire truth for BOTH sides — the IMP
§27.M11 risk row); `internal/runner/subprocess` implementing the engine `Runner` interface
(`internal/engine/engine.go:45` — same contract NativeRunner satisfies); Python `awis-step`
library (`@step` decorator + `step.serve()`, Blueprint §25 example verbatim as the API oracle);
pytest suite against the SAME golden files; e2e keystone: a Python step executes inside an
`awistesting` harness workflow and its outputs flow to the next step (IMP §27.M11 Val row).

## 1. TDS-04 + golden protocol files (M11-C1)
Author `docs/SUBPROCESS_PROTOCOL.md` (TDS-04). Frozen decisions (CE-pinned at A-INIT to kill
the protocol-ambiguity risk; deviations are an E3 escalation, not a local fix):
- **Model:** one process per step execution, one-shot exchange. Runtime writes exactly one
  JSON request object to child stdin, then closes stdin. Child writes exactly one JSON
  response object to stdout and exits. Long-lived processes are M12 plugin territory.
- **Framing:** single JSON object per direction, UTF-8, newline-terminated. Child stderr is
  free-form logs: captured by the runtime, never parsed as protocol.
- **Request envelope** (field names frozen):
  `{"protocol":"awis-subprocess/1","handler":"<step.handler>","instance_id":"…",`
  `"step_id":"…","attempt":1,"inputs":{…},"deadline":"<RFC3339Nano>"}`
  `deadline` omitted when the step has no timeout. Field values mirror `core.StepContext`.
- **Success response:** `{"protocol":"awis-subprocess/1","outputs":{…}}`
- **Error response:** `{"protocol":"awis-subprocess/1","error":{"code":"…","message":"…",`
  `"details":{…}}}` — mapped verbatim onto the frozen `core.StepError` (TDS-01 §2.1);
  empty/missing code → `handler_error`.
- **Error mapping (runtime side; codes are the stable class set):**
  - spawn failure (exec not found, not executable) → `spawn_error`
  - deadline exceeded → kill the process group (SIGKILL) → `timeout` (same semantics as
    NativeRunner's mapping)
  - stdout is not one valid envelope / wrong `protocol` value → `protocol_error`
    (stderr tail ≤4KiB + exit code in details)
  - nonzero exit WITH a valid error envelope → the envelope wins (its code/message/details)
  - nonzero exit without a valid envelope → `subprocess_error` (exit code + stderr tail)
- **Handler → argv:** `step.handler` is whitespace-split into argv and exec'd directly —
  no shell interpretation (documented in TDS-04; Blueprint §25 "spawns this as
  `python my_step.py`").
- **Timeout:** `step.Timeout` parsed with `time.ParseDuration` (the same rule
  `internal/runner/native` uses — cite the coordinate); also serialized as `deadline` in the
  request so handlers can self-limit.
Golden files at `internal/runner/subprocess/testdata/protocol/` (canonical location; the
pytest suite reads the same files via a relative path): at minimum `request-basic.json`,
`request-deadline.json`, `response-ok.json`, `response-error.json`, plus invalid-wire cases
(`bad-protocol.json`, `bad-truncated.json`). TDS-04 embeds each golden verbatim.

## 2. SubprocessRunner (M11-C2) — `internal/runner/subprocess`
- `subprocess.New() *SubprocessRunner` implementing `engine.Runner`:
  `Run(ctx, core.StepContext, core.Step) (core.StepResult, *core.StepError)`.
- Behavior exactly per TDS-04 §1 above: build request from StepContext/Step, spawn argv
  (process group so the timeout kill reaps descendants), single exchange, map every outcome
  to StepResult or typed *core.StepError. Context deadline handling mirrors
  `internal/runner/native` (deadline from step.Timeout when positive; ctx may already carry
  one from the engine).
- Wiring: add `core.StepTypeSubprocess: subprocess.New()` to the runners map in
  `sdk/runtime.go` (the `map[core.StepType]engine.Runner` literal, sdk/runtime.go:92). This is
  internal wiring, NOT an sdk exported-surface change. The harness (sdk/testing) uses
  `sdk.NewRuntime`, so subprocess steps route automatically — do not touch sdk/testing.
- Tests (Go): golden-file conformance (requests the runner emits == golden bytes modulo JSON
  key order — unmarshal-compare; responses parsed from goldens map to expected
  StepResult/StepError); behavior tests against tiny fixture executables (sh scripts under
  testdata/bin/ or `os/exec` of `go run` helpers): ok path, error envelope path, nonzero-exit
  path, garbage-stdout path, timeout kill (fast — timeout ≤1s), spawn failure, stderr capture.

## 3. `awis-step` Python library + e2e keystone (M11-C3) — `python/awis-step`
- Package `awis_step` (pyproject.toml exists as stub — complete it: name `awis-step`,
  requires-python `>=3.10`, zero runtime dependencies). Public API is the Blueprint §25
  example VERBATIM as oracle:
  `from awis_step import step, StepContext, StepResult` / `@step(id="…")` decorating
  `fn(ctx: StepContext) -> StepResult` / `step.serve()` reads the request envelope from
  stdin, dispatches by envelope `handler` to the registered function, writes the response
  envelope to stdout. (`@step(id=…)` registers; `serve()` is one-shot per TDS-04.)
- `StepContext`: `instance_id`, `step_id`, `attempt`, `inputs`, `deadline` (datetime|None).
  `StepResult(outputs={…})`. Handler errors: unregistered handler → error envelope
  `handler_not_found`; handler raises → `handler_error` with `details.traceback`; a handler
  may `raise StepError(code=…, message=…, details=…)` for a typed error.
- pytest suite (`python/awis-step/tests/`) against the SAME goldens
  (`internal/runner/subprocess/testdata/protocol/` via repo-relative path): request parsing,
  response emission byte-compatibility (JSON-equal), error paths, serve() smoke test over
  real pipes.
- `make pytest` target added to Makefile (`python3 -m pytest python/awis-step/tests`); CI
  green includes it (IMP §27.M11 Merge row).
- **e2e keystone (Val row):** Go test in `internal/runner/subprocess` (or sdk/testing
  consumer-side test file — new file only): harness workflow of subprocess step (handler
  `python3 <testdata script using awis_step>`) → native step consuming its outputs; assert
  the value flowed (FR-SE-02, FR-SDK-10). `t.Skip` if `python3` is not on PATH (CI has it).
  The script imports `awis_step` via PYTHONPATH pointing at `python/awis-step`.

## Non-scope (do not implement in M11)
- Plugin system, JSON-RPC 2.0, manifests, long-lived processes, restarts (M12).
- TypeScript/shell runner libraries (protocol supports them; libraries are post-V1 —
  PRD FR-SE-02 is satisfied by the protocol + Python reference implementation).
- CLI surface (M14); OIP anything (M15).
- No new Go dependencies (stdlib os/exec + encoding/json suffice). No Python runtime deps.
- No migrations; no StoragePort changes; no core/engine/expr/validate/dsl changes; no sdk
  exported-surface changes (runtime.go wiring line is the ONLY sdk file edit).
