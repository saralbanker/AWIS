# AWIS GUI — Architecture

Status: proposed — implements the decisions recommended in `GUI_MASTER_PLAN.md` §12,
formalized here as ADRs. Basis: same evidence base as `GUI_MASTER_PLAN.md` (branch
`engine-hardening` @ `f004f4f`, 2026-08-30). This document is the *how*; `GUI_PRD.md`
is the *what*; `GUI_ROADMAP.md` is the *when*; `GUI_DEPENDENCY_MAP.md` is the *in what
order*.

---

## 1. System shape

```
┌────────────────────────────────────────────┐
│  Browser SPA                                │
│  canvas · run list · run detail · editor    │
└───────────────┬────────────────────────────┘
                │ JSON over HTTP  +  SSE
┌───────────────▼────────────────────────────┐
│  cmd/awis-server   (IN-MODULE — ADR-3)      │
│  internal/api/     handlers, DTOs, SSE hub  │
└───────────────┬────────────────────────────┘
                │ direct Go calls
┌───────────────▼────────────────────────────┐
│  sdk.Runtime  +  internal/storage           │
│  engine embedded via rt.Start(ctx)          │
└───────────────┬────────────────────────────┘
                │
        SQLite (WAL, single writer)
```

One new binary (`cmd/awis-server`), one new internal package (`internal/api`), zero
changes to the G1-frozen surfaces (12 event types, 12 `StoragePort` methods, event
payload field names). Everything below is additive.

## 2. Process and deployment model

**One process, engine embedded.** `sdk.NewRuntime` starts no goroutines and installs no
signal handlers; `internal/engine` and `sdk` contain zero `os.Exit` and zero
`signal.Notify` calls — those live only in `cmd/awis` (verified by grep,
`GUI_MASTER_PLAN.md` §7.1). This means `cmd/awis-server` can own the engine lifecycle
with its own `context.Context` exactly as `cmd/awis start` does today. `rt.Start(ctx)`
blocks; the server runs it in a goroutine and serves HTTP on the main one.

**Hard constraint: exactly one engine per database.** The engine keeps per-instance
state in memory — retry schedules, pending activations, terminally-failed markers, live
waits (`engine.go:77-85`) — reconstructed per-process by `hydrate()` on first touch.
Running two engine processes against one SQLite file is undocumented and unverified
territory; multi-*process* access is safe for pure storage operations (WAL,
`busy_timeout=5000`, `_txlock=immediate`, `ClaimStep`'s unique-constraint gate) — which
is why `awis submit` from a second process already works safely today — but that does
not extend to two ticking engines. **Operational rule: run `awis-server` instead of
`awis start`, not alongside it.** (R7 in the risk register; mitigate with a PID/lock
file guard as part of G4.)

**Loopback-only for V1.** Bind `127.0.0.1` by default; there is no authn/authz layer to
protect a wider bind (§8 below, ADR carried from `GUI_MASTER_PLAN.md` §10).

## 3. Data flow

**Read path (Phases 1–3):** browser → `GET /api/v1/...` → `internal/api` handler →
direct Go call into `sdk.Runtime` / `StoragePort` → JSON response. No caching layer in
V1; SQLite reads are fast enough at V1 volumes (§8.4 below covers the pagination debt
that needs closing before that assumption is load-tested).

**Write path (Phase 3+):** browser → `POST /api/v1/...` → handler → `sdk.Runtime`
mutation (`Submit`/`Signal`/`Cancel`/`RegisterWorkflow`) → same SQLite transaction
machinery the CLI already uses. The GUI does not introduce a second write path or a
queue in front of the engine.

**Live path (Phase 2+):** see §5 (Streaming).

## 4. Storage architecture

### 4.1 What exists today (unchanged)

Six migrations (`0001_core_execution.sql` … `0006_recall_fts.sql`). `execution_events`
is the append-only EventLog; `wait_records` is a side table the engine overlays onto
`RebuildState`'s replay for the `waiting` status, because pure event replay cannot
reconstruct it (`internal/storage/rebuild.go:218-224`, `internal/engine/signal.go:149-153`
— see ADR-1 for why this matters to the GUI).

### 4.2 ADR-1 — Global change cursor: additive `state_changes` table, not a `global_seq` column

**Status:** Recommended (D1 in `GUI_MASTER_PLAN.md` §12).

**Context.** The GUI needs a globally-ordered, tailable stream of "something changed"
events for the live view (Phase 2). Two designs were considered.

**Option A (rejected): add `global_seq` to `execution_events`, backfilled from
`rowid`.** Solves ordering. Does **not** solve the deeper problem: three real state
transitions never produce an event at all — `running → waiting` (`signal.go:164-168`),
`waiting → running` on timeout-continue (`signal_timeout.go:128-140`), and
cancellation-requested (`cancel.go:118`). A stream built purely from
`execution_events`, however it's ordered, would still go silent when an instance parks
— reproduced against a live daemon (`GUI_MASTER_PLAN.md` §3.1). It is also the more
invasive migration: SQLite cannot `ALTER TABLE ... ADD COLUMN ... AUTOINCREMENT`, so it
requires a create-copy-rename rebuild of the platform's source-of-truth append-only
table.

**Option B (chosen): additive `state_changes` table.**

```sql
-- migration 0007
CREATE TABLE state_changes (
  seq          INTEGER PRIMARY KEY AUTOINCREMENT,  -- the global cursor
  instance_id  TEXT NOT NULL,
  namespace    TEXT NOT NULL,
  kind         TEXT NOT NULL,     -- 'event' | 'projection'
  event_id     TEXT,              -- set when kind='event'
  status       TEXT NOT NULL,     -- instance status after the change
  current_steps TEXT NOT NULL,    -- JSON
  changed_at   TEXT NOT NULL
);
CREATE INDEX idx_state_changes_ns ON state_changes(namespace, seq);
```

One row written inside the **existing** transaction of `AppendEvent` (`sqlite.go:94-158`)
and of `UpsertInstance` (`sqlite.go:428+`) — both already open their own transaction, so
this adds no new transaction and no new lock.

**Why this wins:**
- Captures every transition, evented or not — `enterWait`, timeout-continue, and
  cancellation-requested all flow through `UpsertInstance`, so all three become visible
  to a tailing client for the first time.
- Never touches the G1 freeze. This is an additive migration plus one new method
  reached through a locally-declared interface and a type assertion — the same pattern
  `cancellationStore`, `signalWaitStore`, `PluginStore`, and `signal.Store` already
  establish.
- `AUTOINCREMENT` is immune to `VACUUM` renumbering (SQLite keeps the counter in
  `sqlite_sequence`), unlike a bare `rowid`. `migrations/0004_audit.sql`'s `audit_log`
  table already uses exactly this pattern — in-repo precedent.
- Cheap to revert: dropping an additive table is not a format decision.

**Consequence for the API design:** the change stream tells the client *what to
re-fetch*; per-instance detail still comes from the existing, already-indexed
`ReadEvents(instance, fromSeq)`. `state_changes` is a change-notification stream, not a
second copy of the event log.

**Deferred, not rejected:** whether a client also needs a globally-ordered *event*
stream (vs. a change stream plus on-demand per-instance reads) — revisit only if a
genuine firehose requirement appears; cheaper to justify then than to build
speculatively now.

### 4.3 ADR-2 — Namespace: retire `"default"` as a wildcard overload

**Status:** Recommended (D2 in `GUI_MASTER_PLAN.md` §12).

**Problem.** `"default"` is currently overloaded in two incompatible directions.
`namespacePredicate` (`cmd/awis/export.go:158-163`) maps the literal string `"default"`
to `""`, and `""` at the storage layer means *no predicate — every namespace*. So on
the read path, `--namespace default` means "all namespaces"; on the write path,
`submit` passes `"default"` through to `sdk.Config.Namespace` as a **literal name**.
The same rule is reimplemented three separate times (`export.go` via a helper,
`history.go:75` inline, `metrics.go:86` inline with a different condition), and
`ReadEventRange` (`internal/storage/sqlite.go:182-192`) has no wildcard branch at all —
so `StepStats`, which calls it with the runtime's default namespace, **returns
all-zero statistics with no error** whenever that default doesn't literally match
(§6.4). This is the review's second Critical-severity risk (R2): a metrics dashboard
built on this today renders an empty chart and reports success.

**Decision.** Make `""` the wildcard, make `"default"` an ordinary namespace name
everywhere, and add an explicit `--all-namespaces` flag / `?all_namespaces=true` query
param at the API layer. Give `ReadEventRange` a wildcard branch. Delete the three
duplicated implementations in favor of one shared helper.

**Consequence.** This changes what `history`, `metrics`, and `export` return for
existing projects — take the breakage now, before the GUI encodes the ambiguity into
saved filters and URLs, where it becomes permanent.

### 4.4 Pagination debt (carried into the API layer, §6.4)

| Read path | Bounded today? |
|---|---|
| `ListInstancesPaged` / `Runtime.ListPaged` | Yes — cursor, 100 default / 1000 max |
| `ListInstances` | No, by design (the engine tick needs every running instance) |
| `ReadEvents` | No |
| `ReadEventRange` | No |
| `QueryHistory` | No (delegates to `ListInstances`, filters in memory) |
| `ReplayInstance` | No |

Fine at V1 volumes. Before the GUI is exposed to real workload, `ReadEvents` needs a
limit+cursor — a single instance with 100k events otherwise returns all of them in one
query to render one timeline — and `StepStats`'s 10-year full-namespace rescan per call
(`sdk/runtime_recall.go:112-119`) should become an incremental aggregate fed off the
same `state_changes` cursor from ADR-1.

## 5. Streaming architecture

### 5.1 ADR-4 — SSE, not WebSocket

**Status:** Recommended.

Traffic is one-directional (server → client); every mutation goes over an ordinary
POST. SSE gives reconnection and `Last-Event-ID` resumption for free, works through
ordinary HTTP proxies, and needs no dependency beyond stdlib `net/http`. The repo has
exactly two direct dependencies today (`yaml.v3`, `modernc.org/sqlite`, both pure-Go,
no cgo) — a WebSocket library would be the third and the first with any protocol
complexity. Keeping the server dependency-light is worth protecting.

```
GET /api/v1/stream?since=<seq>&namespace=<ns>
  → text/event-stream, id: <seq>
```

The handler tails `state_changes` (ADR-1) from `since`, emits one event per row, and the
client resumes with `Last-Event-ID` after a drop.

### 5.2 Latency model — be honest about it

There is no notification mechanism to hook: the engine is pull-based, with no channel,
condvar, or DB trigger anywhere in the codebase. Every state change becomes visible
within one engine tick and no faster (default `TickInterval` 100ms, configurable via
`--tick`). Server-side poll interval on `state_changes` should track the engine's own
tick. Submit → first `StepStarted`, signal → resumption, timeout → routing, cancel →
effect are all bounded by one tick interval. **Cancel is additionally cooperative** — an
in-flight step is not interrupted — so the UI's "Cancel" affordance must read as
"requested," not "stopped" (carried into `GUI_PRD.md` FR-2.4 and NFR-4).

## 6. API layer

Versioned under `/api/v1`, entirely in-module (ADR-3, §7).

| Method | Path | Backed by | Notes |
|---|---|---|---|
| GET | `/workflows` | `StoragePort.ListWorkflows` | |
| GET | `/workflows/{id}/{version}` | `StoragePort.GetWorkflow` | needs a not-found sentinel, §8.3 |
| POST | `/workflows` | `Runtime.RegisterWorkflow` | new version only, no update path |
| POST | `/workflows/validate` | `validate.Validate` | return the full structured `Issue` (Code, StepID, Field, Position), not just line+message |
| GET | `/instances` | `Runtime.ListPaged` | already cursor-bounded |
| GET | `/instances/{id}` | `Runtime.Status` + wait-record lookup | Status omits `signal_name`/`timeout_remaining_s` today — wrapper needed |
| GET | `/instances/{id}/events` | `StoragePort.ReadEvents` | add limit+cursor (§4.4) |
| POST | `/instances` | `Runtime.Submit` | |
| POST | `/instances/{id}/signal` | `Runtime.Signal` | |
| POST | `/instances/{id}/cancel` | `engine.Cancel` | expose `compensate` — SDK hardcodes `false` today |
| GET | `/stats/steps` | `Runtime.StepStats` | gate behind ADR-2 landing — see R2 |
| GET | `/plugins` | `storage.ListPlugins` | in-module only, blocked for any external consumer (§7) |
| GET | `/stream` | `state_changes` tail (SSE) | ADR-1 + §5 |
| GET | `/healthz` | new | |

### 6.1 Error model

Partially mappable today. Real sentinels exist for `ErrInstanceNotFound`,
`ErrPluginNotFound`, `ErrCapabilityNotFound`, `ErrDuplicateDomainEvent`,
`ErrVersionConflict`, `ErrPaginationUnsupported` → clean 404/409/501. Two gaps to close
as part of building the API layer, not after: `GetWorkflow`'s not-found path is a bare
`fmt.Errorf(..."not found")` with no sentinel, and `cmd/awis/start.go` string-matches
`"already registered"` rather than asserting a typed `*RegistrationError`. A GUI needs
to distinguish 404 from 500 on its single most-called read path — add the sentinel and
fix the string match together.

## 7. Module boundary — ADR-3

**Status:** Recommended (D3 in `GUI_MASTER_PLAN.md` §12).

**Empirical basis.** An out-of-module Go program, built against this repo via a `replace`
directive, resolves and compiles cleanly against `sdk.SQLiteStorage`, `sdk.NewRuntime`,
`ListPaged`, `Status`, `Submit`, `Signal`, `Cancel`, `Recall().ReplayInstance` — verified
directly. The same program fails to build the moment it references any type from
`internal/storage` (`storage.PluginRow`, `RecallRow`, `AuditRow`, `WaitRecord`) with
Go's own `use of internal package ... not allowed` — also verified directly. An external
module cannot even *declare* an interface to type-assert against those types, because it
cannot name the return type.

**Decision.** Build the GUI backend inside this module, as `cmd/awis-server/` with
handlers in `internal/api/`. Not a workaround — the correct call for V1. It costs
nothing, unblocks the plugin/recall/audit/wait-record surfaces immediately, and — most
importantly — means the team does not have to design and freeze a public HTTP/SDK
contract before knowing what the GUI actually needs. Promoting the four blocked row
types into `sdk` is a one-day change, deferred until a second consumer exists to justify
the shape.

## 8. Editor architecture (Phase 4)

### 8.1 Graph model

The workflow graph is not one edge list; the canvas must render six distinct
relationship types, only two of which live in `Transitions`:

| Relationship | Where it lives |
|---|---|
| Unconditional transition | `Transition{From,To}` |
| Conditional transition | `Transition.Condition` |
| Error transition | `Transition.OnError` — same struct, boolean discriminator |
| Fallback | `Step.Fallback` — on the step, not in `Transitions` |
| Compensation | `WorkflowDefinition.Compensation` — separate ordered list, reverse order |
| Retry / signal-timeout | not edges — step-local policy, render as badges |

**Asymmetry the canvas must mirror exactly:** fallback edges count for reachability but
are *excluded* from cycle detection (`internal/validate/validate.go:26-28`). A canvas
that includes them in its own cycle check will reject graphs the engine accepts.

### 8.2 What's free vs. what needs building

**Free today:** `validate.Validate(def) []Issue` is pure — no storage, no runtime, no
handler registry — and returns structured `{Code, StepID, Field, Message, Position}`,
exactly the shape needed for live inline diagnostics. `expr.ParseTemplate` and
`expr.ParseCondition` return `*ParseError{Position, Msg}` with in-bounds byte offsets
for editor squiggles.

**Needs building:**
- **Serializer (G6, no engine work exists today):** `internal/dsl` exports `Parse`,
  `ParseFile`, `ValidateFile`, `Discover` and nothing else; `render.go` renders a
  *validation report*, not YAML. Repo-wide, `yaml.Marshal` is called zero times
  (verified by grep). Build the emitter against the same `yaml:` struct tags the parser
  uses, and add the round-trip test that does not exist today.
- **Variable enumeration for autocomplete:** nothing walks the graph to answer "what's
  in scope at step X"; derive it from upstream steps' declared `Outputs`.
- **Node palette / runtime introspection (G7):** `engine.runners`,
  `NativeRunner.handlers`, `Runtime.handlers`, and `plugin.Manager` are all unexported
  with no `List*` accessor. The five `StepType` kinds are compile-time constants and can
  be hardcoded, but which native handlers / plugin capabilities are *actually
  registered in this deployment* cannot be queried today. Three small accessors close
  this gap — the difference between a palette that reflects reality and one that drifts.

**A naming trap to avoid:** `Step.Inputs` is typed `InputSchema` and commented as a
schema, but per `WORKFLOW_SCHEMA.md:112-116` it is actually an execution-time *template
value map*. An editor that builds a JSON-Schema form from the field name will build the
wrong UI.

### 8.3 Layout persistence — no schema change needed

`core.Step` has no metadata field and the schema is G1-frozen, but
`WorkflowDefinition.Metadata map[string]any` exists (`workflow.go:37`), is parsed from
YAML, and is explicitly documented as application-defined data:

```yaml
metadata:
  ui:
    positions:
      fetch:   {x: 120, y: 80}
      approve: {x: 340, y: 80}
```

Zero schema change, zero founder sign-off needed, works today. A first-class `Step.UI`
field would be cleaner but requires G1 sign-off — not worth spending that before the
editor's shape has proven itself. Revisit at milestone G5.

## 9. Security architecture

**No authorization model exists.** Grep for tenant/role/user/session/permission returns
zero hits in AWIS code. There is no identity, no session, no per-namespace access
control, no API-key issuance. `config show/set` masking (fail-closed by allowlist since
`0ddf1ce`) is operator-local hygiene, not multi-tenancy.

**V1 posture:** bind `127.0.0.1`, ship. Before any remote deployment, authz is a
from-zero build — budget it as its own milestone (G10), not a task folded into another
phase.

**Two runner-isolation facts to resolve before Phase 4 lets non-operator users author
workflows:** plugins get a scrubbed environment (manifest `env` + `PATH` only,
`manager.go:772-782`), but **subprocess steps inherit the full parent environment** —
`internal/runner/subprocess/subprocess.go` never sets `cmd.Env`, so a subprocess step
sees `ANTHROPIC_API_KEY` and everything else in the daemon's environment. Neither runner
has an OS sandbox — no seccomp, namespaces, chroot, or cgroups anywhere. A workflow
author has arbitrary file and network access as the daemon user. Acceptable for a local
single-operator tool; unacceptable the moment the GUI lets someone else author
workflows.

## 10. Architecture Decision Record summary

| ADR | Decision | Status |
|---|---|---|
| ADR-1 (D1) | `state_changes` additive table for the global change cursor | Recommended |
| ADR-2 (D2) | Retire `"default"` as a namespace wildcard overload | Recommended |
| ADR-3 (D3) | GUI backend in-module (`cmd/awis-server`) | Recommended |
| ADR-4 | SSE over WebSocket for the live stream | Recommended |

D4 (live-intelligence CI gate) is a process decision, not an architecture decision —
see `GUI_PRD.md` Phase 5 (FR-5.3) and `GUI_MASTER_PLAN.md` §9.3.

## 11. Architecture risks carried forward

Full register in `GUI_MASTER_PLAN.md` §11. The architecturally load-bearing ones:

- **R1 (Critical):** without ADR-1, the live stream silently omits `waiting`.
- **R2 (Critical):** without ADR-2, `StepStats` returns zeros, not an error, on
  namespace mismatch.
- **R3 (High):** building against `CLI_CONTRACT.md` instead of the binary.
- **R4 (High):** the G1 freeze is comment-only and enforced by nothing — no
  golden-snapshot test on `ExecutionEvent` serialization, no reflective test pinning
  `StoragePort` at 12 methods. If the GUI depends on these shapes, the freeze needs
  teeth: add snapshot tests as part of the contract-hardening milestone.
- **R7 (Medium):** two engines on one DB — mitigate with a PID/lock guard in
  `cmd/awis-server`.
- **R12 (Low for V1):** SQLite lock-in (FTS5, pragmas, `AUTOINCREMENT`, string-matched
  constraint errors) — Postgres is aspirational, no adapter exists.
