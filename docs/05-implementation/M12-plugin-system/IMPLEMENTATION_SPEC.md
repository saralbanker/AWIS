# M12 — Implementation Spec
**Canonical sources:** IMP §27.M12; IMP §18 L334; Blueprint §11 (verbatim: JSON-RPC 2.0 example,
lifecycle list, manifest YAML, registry SQL, "What Plugins Cannot Do"); TDS-05
`docs/PLUGIN_PROTOCOL.md` (written day 1 — THIS milestone authors it); PRD FR-PS-01..06, 14, 15;
NFR-S-02; F-4 (PluginRegistered audit write site; audit API exists since M07 —
`internal/storage/audit.go`); IMP §14 (plugins registry migration — renumbered per F-4 to
`0005_plugins.sql`, since 0003/0004 landed at M07 as signals/audit); IMP §20 checkpoint M12;
Finalization idle-kill recommendation (<2s respawn).

## Scope
TDS-05 + golden JSON-RPC wire files; manifest parsing (`awis-plugin.yaml`); plugin registry
(migration `0005_plugins.sql` — Blueprint §11 SQL verbatim — additive PluginStore storage
interface following the M07 audit pattern; `PluginRegistered` audit write site per F-4);
lifecycle manager (FSM below, CE-pinned); JSON-RPC 2.0 transport over long-lived stdin/stdout;
PluginRunner (engine Runner for `type: plugin`); sdk runtime wiring; Python `awis-plugin`
library + `awis_plugin.testing.mock_request` offline harness (FR-PS-14/15); §20 checkpoint
(external kill mid-call → auto-restart → step retry succeeds; idle-kill → transparent respawn);
plugin developer guide.

## CE-pinned protocol decisions (TDS-05 normative; deviations are E3)
- **Framing:** newline-delimited JSON-RPC 2.0 objects (NDJSON), UTF-8, over a LONG-LIVED
  stdin/stdout pair. stderr = free-form logs, captured bounded (4KiB tail), never protocol.
- **Direction:** runtime is the JSON-RPC client; the plugin only responds. Request ids are
  strings (`"req-<n>"`, monotonic per process).
- **Methods:**
  - `handshake` params `{"protocol_version":"awis-plugin/1"}` → result = the plugin's manifest
    echo `{"name","version","capabilities":[{"id",...}]}`. Sent immediately after spawn;
    response deadline 5s; validation: name+version equal the registered manifest, declared
    capability ids are a superset of the registered ones. Failure/timeout ⇒ crash-equivalent.
  - `execute` params `{"capability":"<id>","step_id":"…","inputs":{…},"timeout_ms":N}` →
    result `{"outputs":{…},"duration_ms":N}` or JSON-RPC error `{code,message,data}`.
    (Blueprint §11's example omits `capability` — illustrative prose; TDS-05 is the normative
    home and REQUIRES it. Recorded in TRACEABILITY dispositions.)
  - `shutdown` notification (no id) → plugin exits cleanly; runtime waits ≤2s then SIGKILLs
    the process group.
- **Capability resolution (runtime side; frozen fixture compatibility):** a plugin step's
  `handler` is EITHER a capability id (exact match in `plugin_capabilities`) OR a plugin name
  (the frozen Blueprint §7 fixture uses `handler: git-context-plugin`). For the plugin-name
  form the runtime selects the unique capability whose declared input KEY SET equals the
  step's resolved input key set; zero or multiple matches → typed StepError
  `plugin_capability_ambiguous` (remedy: use the capability id as handler). The execute
  request always carries the resolved explicit `capability`.
- **Error mapping (runner side):** spawn failure → `spawn_error`; handshake failure/timeout →
  `plugin_handshake_error` (counts as crash); process exit while call in-flight →
  `plugin_crash`; call deadline exceeded → `timeout` (controlled kill; does NOT count as
  crash); JSON-RPC error response → `plugin_error` (or `data.code` when the plugin supplies
  one) with message/data verbatim; plugin FAILED (restart budget exhausted) → `plugin_failed`;
  unparseable stdout line / id mismatch → `protocol_error` (crash-equivalent: kill + count).
- **Effective call timeout:** `min(step ctx deadline, step.Timeout if set, capability
  timeout_ms)`; `timeout_ms` param carries the effective remaining budget in ms.

## CE-pinned lifecycle FSM (the IMP §28 Opus-designated core; Fable adversarial review at D-CLOSE)
In-memory states per plugin: `REGISTERED, SPAWNING, HANDSHAKING, ACTIVE, IDLE, TERMINATED,
FAILED`. DB `plugins.status` stays coarse: `registered|active|failed` (Blueprint SQL comment
lists `suspended` — unused in V1, recorded disposition). Transitions:
- Registration (API `Manager.Register(manifestPath)`): parse+validate manifest → DB rows
  (plugins + plugin_capabilities, `plugin_id` = manifest `name`; V1 pins one installed version
  per name — disposition) → audit `PluginRegistered` (F-4) → state REGISTERED. Re-register
  same name = replace rows (upsert) + audit.
- **Lazy spawn:** SPAWN happens on demand at first call (pull philosophy). `ensureActive()`:
  REGISTERED/IDLE → SPAWNING (exec manifest `runtime.command`+`args`; env = manifest `env` +
  `PATH` ONLY (NFR-S-02 minimal env); no extra fds (os/exec default; NFR-S-02); process group
  Setpgid) → HANDSHAKING (5s) → ACTIVE. DB status → active on first successful handshake.
- **Calls are serialized per plugin** (mutex; V1 pin — engine concurrency spans distinct
  plugins; recorded disposition).
- **Crash:** unexpected process exit (monitor goroutine on Wait). In-flight call fails
  `plugin_crash`; consecutive-crash counter += 1; counter > 3 ⇒ FAILED + DB status failed +
  every later call fails `plugin_failed` (until re-register). Otherwise state → REGISTERED
  (respawn on next call = "auto-restart"; the §20 checkpoint proves kill → restart → step
  retry OK). Counter RESETS on a successful execute completion. Handshake failures count.
- **IDLE:** no execute for `idle_timeout_s` → kill process group, state IDLE (DB stays
  active); next call respawns transparently (<2s expected — Finalization). Idle watchdog is a
  per-plugin timer reset on each call; Manager config exposes a clock/interval seam for fast
  deterministic tests (internal package — allowed).
- **Shutdown:** `Manager.Shutdown(ctx)`: per active plugin, `shutdown` notification, ≤2s
  grace, SIGKILL group; state TERMINATED. Idempotent.
- **Invariants (adversarial review targets):** (1) no call may be dispatched to a process in
  any state but ACTIVE; (2) a crash during HANDSHAKING must not deadlock waiters; (3) the
  idle killer must never kill a process with a call in flight (mutex covers both); (4) counter
  transitions and state transitions are atomic under one lock; (5) monitor goroutines never
  leak past Shutdown.

## 1. TDS-05 + goldens + manifest parsing (M12-C1) — `internal/plugin`
- `docs/PLUGIN_PROTOCOL.md` (TDS-05): all pins above in normative tables; lifecycle diagram
  (Blueprint §11 list verbatim + the pinned transition semantics); manifest format (Blueprint
  §11 YAML verbatim as the oracle); goldens embedded.
- Goldens at `internal/plugin/testdata/protocol/`: `handshake-request.json`,
  `handshake-response.json`, `execute-request.json` (the Blueprint §11 example + `capability`),
  `execute-response.json`, `execute-error.json`, `shutdown-notification.json`,
  `bad-jsonrpc.json`.
- `internal/plugin/manifest.go`: `ParseManifest(path)` / `ParseManifestBytes` →
  `Manifest{Name,Version,Description,Author,Capabilities[]{ID,Inputs,Outputs,TimeoutMS},
  Runtime{Command,Args,Env,IdleTimeoutS}}` (yaml.v3, KnownFields, field names from the
  Blueprint YAML verbatim); validation: name/version(semver)/≥1 capability/unique capability
  ids/command non-empty; file+line errors like internal/dsl. Tests incl. the Blueprint §11
  manifest verbatim as testdata oracle.

## 2. Registry + audit write site (M12-C2) — storage
- Migration `internal/storage/migrations/0005_plugins.sql`: Blueprint §11 SQL VERBATIM
  (tables `plugins`, `plugin_capabilities`) + the standard migration header comments used by
  0003/0004.
- Additive interface in `internal/storage` following the audit.go pattern (type-asserted, NOT
  a StoragePort change): `PluginStore` with `RegisterPlugin(manifest Manifest-JSON string,
  name, version string) error` (upsert plugins + replace capabilities, status registered),
  `GetPlugin(name)`, `ListPlugins()`, `SetPluginStatus(name, status)`,
  `LookupCapability(capabilityID) (pluginName string, err)`, `PluginByInputKeys` NOT here —
  key-set resolution is manager logic, not SQL.
- Audit write site: registration writes `PluginRegistered` via the M07 audit API in the SAME
  transaction pattern audit.go documents (F-4).
- SQLite impl + tests (pattern: audit store tests); storagetest contract additions if the
  existing storagetest suite pattern expects them.

## 3. FSM manager + JSON-RPC transport + PluginRunner + wiring (M12-C3) — `internal/plugin`
- `transport.go`: long-lived process handle; NDJSON read/write loop; request/response
  correlation by id; bounded stderr tail; process-group spawn/kill (reuse the M11 patterns —
  cite `internal/runner/subprocess` coordinates in comments, but this is a DIFFERENT protocol:
  long-lived, JSON-RPC framed).
- `manager.go`: the CE-pinned FSM above, exactly. `NewManager(store PluginStore, cfg
  ManagerConfig)`; `Register(manifestPath)`; `Call(ctx, capability|handler, stepID, inputs,
  effectiveTimeout) (outputs, *core.StepError)`; `Shutdown(ctx)`.
- `runner.go`: `PluginRunner` (engine.Runner): resolve handler per the pinned resolution rule
  → `manager.Call` → map to StepResult/StepError.
- sdk wiring: `sdk/runtime.go` constructs the manager over `cfg.Storage` (type assert
  PluginStore; nil-safe: storage lacking PluginStore ⇒ runner returns `plugin_error`
  "storage does not support plugins") and adds `core.StepTypePlugin` to the runners map;
  Runtime.Close/Stop path calls manager.Shutdown. Wiring only — no exported sdk surface change.
- Tests: golden conformance (wire bytes vs goldens); fake plugin executables (sh/go fixtures
  speaking NDJSON JSON-RPC): happy path; handshake mismatch; crash mid-call → `plugin_crash`
  + restart on next call; 4th consecutive crash → FAILED/`plugin_failed`; counter reset after
  success; idle kill → respawn (fast clock seam); timeout → controlled kill, no crash count;
  Shutdown grace + kill; **§20 checkpoint test**: harness workflow, plugin step with
  retry attempts≥2, externally SIGKILL the plugin process mid-call → engine retry → restarted
  plugin serves the retry → workflow completes.

## 4. `awis-plugin` Python lib + offline harness + e2e + guide (M12-C4) — `python/awis-plugin`
- `awis_plugin/__init__.py`: `plugin` registry object — `@plugin.capability(id="…")`
  decorating `fn(inputs: dict) -> dict`; `plugin.serve(manifest_path=None)`: NDJSON JSON-RPC
  loop implementing handshake (echo manifest — from awis-plugin.yaml beside the module or
  passed path)/execute (dispatch by `capability`)/shutdown (clean exit); errors → JSON-RPC
  error with `data.code` (`handler_error` + traceback; unregistered capability →
  `capability_not_found`). Stdlib only; py>=3.10.
- `awis_plugin/testing.py`: `mock_request(capability, inputs, plugin=plugin)` — invokes the
  registered handler through the SAME dispatch path serve() uses, no process/runtime needed
  (FR-PS-15); plus `mock_handshake()` returning the manifest echo.
- pytest vs the SAME goldens (repo-root traversal, no copies): handshake echo, execute
  dispatch, error shapes, shutdown behavior, mock_request parity with serve dispatch.
- e2e Go test: real Python plugin (testdata: minimal awis-plugin.yaml + plugin module using
  awis_plugin) registered via Manager.Register; harness workflow with `type: plugin` step
  (handler = capability id) runs it; outputs flow onward. t.Skip without python3.
- `docs/PLUGIN_GUIDE.md`: developer guide section (DoD) — manifest, @plugin.capability,
  serve, mock_request testing, install pointer (CLI arrives M14/F-5); cites TDS-05.
- Makefile `pytest` target extended to run BOTH python package test suites.

## Non-scope (do not implement in M12)
- CLI `awis plugin install|list|status|remove` (M14 minimal per F-5; M17 full).
- The reference `git-context-plugin` itself (M13). Per-plugin venvs (M13, PR-5).
- Remote install / marketplace (V2: FR-PS-11/12).
- No changes to: core frozen shapes, engine, expr, validate, dsl, runner/native,
  runner/subprocess (patterns are CITED, code is not modified), StoragePort 12-method set
  (PluginStore is additive type-assert like audit), sdk exported surface.
- No new Go dependency (yaml.v3 + stdlib suffice); no Python runtime deps.
