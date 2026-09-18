# AWIS GUI — Master Synthesis & Comprehensive Summary

**A Unified Synthesis of the AWIS Platform Transition Plan (5-Document Suite)**  
**Basis:** Repo-wide architectural review of branch `engine-hardening` @ `f004f4f` plus uncommitted B-31 remediation (2026-08-30).  
**Source Documents:** `GUI_MASTER_PLAN.md`, `GUI_PRD.md`, `GUI_ARCHITECTURE.md`, `GUI_ROADMAP.md`, `GUI_DEPENDENCY_MAP.md`.

---

## Table of Contents

1. [Executive Summary & Core Finding](#1-executive-summary--core-finding)
2. [Engine Freeze Verdict & Independent Audit](#2-engine-freeze-verdict--independent-audit)
3. [The Real Blocker Hierarchy (B-a through B-f)](#3-the-real-blocker-hierarchy-b-a-through-b-f)
4. [Event-Sourcing Sufficiency Across 7 Use Cases](#4-event-sourcing-sufficiency-across-7-use-cases)
5. [The Unblocking Migration & Architecture Decision Records (ADRs)](#5-the-unblocking-migration--architecture-decision-records-adrs)
6. [System Architecture & Operational Constraints](#6-system-architecture--operational-constraints)
7. [API Layer, Error Contracts & Pagination Debt](#7-api-layer-error-contracts--pagination-debt)
8. [Visual Editor, Graph Model & AI Authoring](#8-visual-editor-graph-model--ai-authoring)
9. [Product Requirements Across the 5 Delivery Phases](#9-product-requirements-across-the-5-delivery-phases)
10. [Milestone Roadmap (G0–G10), Critical Path & Concurrency](#10-milestone-roadmap-g0g10-critical-path--concurrency)
11. [Founder Decisions & Blast Radius Analysis](#11-founder-decisions--blast-radius-analysis)
12. [Complete Risk Register (R1–R18) & Blocker Resolution Map](#12-complete-risk-register-r1r18--blocker-resolution-map)
13. [Direct Answers to Strategic Questions](#13-direct-answers-to-strategic-questions)

---

## 1. Executive Summary & Core Finding

### Can GUI Development Start Tomorrow?
**Yes** — for the frontend SPA and the read-only API backend.  
**No** — for the live execution timeline and workflow monitoring.

### Three Deciding Realities
1. **The Go SDK is genuinely embeddable today:** `sdk.NewRuntime`, `ListPaged`, `Status`, `Submit`, `Signal`, `Cancel`, and `Recall().ReplayInstance` compile and execute cleanly in an out-of-module program.
2. **Zero existing network surface:** There is no HTTP/gRPC server anywhere in non-test Go code (`net.Listen`, `ListenAndServe` return zero matches). The API layer is 100% greenfield.
3. **The event log cannot represent live execution:** The `waiting` state transition is written directly to the SQLite projection table (`UpsertInstance`) without emitting any `ExecutionEvent` (`signal.go:149-179`). Signal-timeout `continue` (`signal_timeout.go:128-140`) and cancellation-requested (`cancel.go:118`) are similarly non-evented. Tailing `execution_events` causes a live UI to show an instance emit `StepStarted` and then permanently hang as "running" when parked on a human signal.

### Master Sequencing Rule
Frontend scaffolding and historical/read-only views start immediately. The live streaming view is gated on one additive migration (**Migration 0007: `state_changes` table**, ~1–2 days).

---

## 2. Engine Freeze Verdict & Independent Audit

The engine-hardening program is **FREEZE-READY WITH TWO CORRECTIONS**. Independent code-level audits verified the primary integrity claims:

### Verification Audit Matrix
| Hardening Claim | Audit Verdict | Evidence & Notes |
|---|---|---|
| `make verify` green | **CONFIRMED (Caveated)** | Passes cleanly ("ALL GATES PASSED"). Caveat: `cmd/awis`'s `TestSystemRehearsalInitStartSubmitTrace` has a hardcoded 10s timeout that flakes under full-suite CPU contention (R14). |
| `make integration` green | **CONFIRMED** | Exit 0, completed in 8.4s. |
| 21 binary integration tests | **CONFIRMED** | Exactly 21 tests under `go test -tags integration -list`; builds real binary and executes via `exec.Command`. Zero `t.Skip`. |
| Frozen G1 surfaces intact | **CONFIRMED** | 12 event types (`internal/core/event.go`), 12 `StoragePort` methods (`ports.go`). |
| "B-0 through B-30 addressed" | **CONFIRMED (Count Adjusted)** | 26 real defect IDs exist (B-0…B-11, B-15…B-28, B-30). **B-12, B-13, B-14, and B-29 never existed** (framing artifact). |
| Load-bearing regression tests | **CONFIRMED** | Tested by live reversion of B-4 (guard) and B-30 (`_txlock=immediate`); tests failed with documented symptoms and passed on restore. |
| Zero open CRITICAL defects | **CONFIRMED** | Verified. |
| Zero open IMPORTANT defects | **CORRECTED & RESOLVED** | B-31 was open and resolved on 2026-08-30 along with 4 adjacent enum bugs; B-27 (cursor) is PARTIAL (subsumed by B-b). |

### B-31 Remediation & The 5-Field Enum Validation Defect Class
* **Root Cause:** `awis init` scaffolded `workflows/with-signal.yaml` with `timeout_action: cancel`. The core DSL (`step.go:81`) only permits `fail | compensate | continue`. `internal/validate/validate.go` omitted validation, and the runtime silently fell back to `default` (treating it as `fail` after 72 hours).
* **Defect Class Discovered & Patched:**
  1. `wait_signal.timeout_action`: unvalidated → silently became `fail` → now `CodeSignalTimeoutAction`.
  2. `retry.backoff`: unvalidated → silently entered 0-delay hot retry loop → now `CodeRetryBackoff`.
  3. `trigger.type`: unvalidated → silently failed to trigger → now `CodeTriggerTypeUnknown`.
  4. `step.type`: unhandled switch → returned `runner_unavailable` → now `CodeStepTypeUnknown`.
  5. `intelligence.capability`: unvalidated → returned `capability_unknown` → now `CodeIntelligenceCapability`.
* **Upgrade Impact (Disclosed Break R18):** Submissions of old pre-registered workflows containing invalid enums are rejected loudly at `Submit`. In-flight parked instances continue using safe runtime defaults.
* **Observability Gap:** Backstop warning logs route to stderr (via default logger when `sdk/runtime.go` passes `nil`), making engine warnings invisible to `awis logs`.

### Additional Audit Corrections & Governance
* **Stale Root Binary:** Root `awis` binary was dated Aug 26 (prior to B-28 fix on Aug 28). Rebuilding resolved spurious `signal_name: null` reports.
* **Commit Count:** Branch contains 30 commits (not 29).
* **B-11 Source Traceability (Retraction):** Early claim that B-11 had zero source traceability was refuted upon deep inspection (`sdk/variables.go`, `sdk/runtime_recall.go:79,122`, 4 tests in `sdk/runtime_recall_test.go`).
* **Governance Finding:** `CLAUDE.md` mandates Sonnet for all implementation/docs and bans Opus selection. All 30 commits carried `Co-Authored-By: Claude Opus 5`.

---

## 3. The Real Blocker Hierarchy (B-a through B-f)

The founder brief's 5 assumed blockers were revised to 6 real blockers with corrected severity weightings:

```
[B-a: Non-Evented State Transitions] (CRITICAL) ───┐
                                                   ├──► Resolved by G2 (state_changes table)
[B-b: No Global Event Cursor]        (HIGH)     ───┘
[B-c: No Definition->YAML Serializer](HIGH)     ──────► Resolved by G6 (YAML Emitter)
[B-d: Namespace Overload Silent 0s]  (HIGH)     ──────► Resolved by G3 (Retire Overload)
[B-e: Unbounded Read Paths]          (MEDIUM)   ──────► Resolved by G4 (Limits) & G8 (Aggregates)
[B-f: Zero Authn/Authz Model]        (MEDIUM/CRIT)───► Resolved by G10 (Authz Milestone)
[Live Provider Validation]           (LOW for GUI)────► Resolved by D4/G9 (Live Anthropic CI Gate)
```

### Detailed Blocker Breakdown
1. **B-a — Non-Evented State Transitions (CRITICAL):**
   * *Mechanism:* `RebuildState` (`rebuild.go:218-224`) replays the event log and manually overlays `waiting` from the `wait_records` table. `running → waiting`, timeout-`continue`, and `cancellation-requested` do not emit events.
   * *Impact:* A live UI tailing events will never show an instance enter `waiting`.
2. **B-b — No Global Event Cursor (HIGH):**
   * `sequence_num` resets per instance; `rowid` is vulnerable to `VACUUM` renumbering. Subsumed into the `state_changes` solution.
3. **B-c — No Definition → YAML Serializer (HIGH):**
   * `internal/dsl` has `Parse`, `ParseFile`, `ValidateFile`, and `Discover`, but zero `yaml.Marshal` calls. Canvas can view, but cannot save without an emitter.
4. **B-d — Namespace `"default"` Overloading & Silent Zeroes (HIGH):**
   * `"default"` is treated as a literal name on write, but mapped to `""` (wildcard match all) on CLI read paths.
   * `ReadEventRange` (`sqlite.go:182-192`) lacks a wildcard branch. Calling `StepStats` when the runtime default doesn't literally match event namespaces returns **all-zero statistics with zero errors**.
5. **B-e — Unbounded Read Paths (MEDIUM):**
   * B-17 capped `ListInstancesPaged`. However, `ReadEvents`, `ReadEventRange`, `QueryHistory`, and `ReplayInstance` remain uncapped.
6. **B-f — No Authn / Authz Whatsoever (MEDIUM for V1 / CRITICAL for Remote):**
   * Zero user, tenant, role, token, or session concepts exist in AWIS. Single-user loopback binding (`127.0.0.1`) is mandatory.

---

## 4. Event-Sourcing Sufficiency Across 7 Use Cases

| Engine Use Case | Verdict | Architectural Reality & Blocking Requirement |
|---|---|---|
| **Realtime UI** | **INSUFFICIENT** | Blocked on B-a + B-b. Requires `state_changes` change stream. |
| **Workflow Canvas** | **SUFFICIENT** | Works off static definitions. Saving blocked on B-c (serializer). |
| **Execution History** | **SUFFICIENT** | `ReplayInstance` provides complete ordered history for completed steps. |
| **Observability** | **SUFFICIENT (Needs Change)** | `StepStats` rescans 10-year window; needs incremental aggregation off cursor. |
| **Debugging** | **SUFFICIENT** | Full event log is preserved and queryable. |
| **Time-Travel Replay** | **SUFFICIENT (With Caveat)** | Log replay alone cannot reconstruct `waiting` or `cancellation_requested`; auxiliary tables required. |
| **AI Generation** | **SUFFICIENT (Needs Change)** | Validator provides rich structured issues, but CLI currently discards codes. |

---

## 5. The Unblocking Migration & Architecture Decision Records (ADRs)

### ADR-1 (Founder Decision D1) — Additive `state_changes` Cursor Table
* **Recommendation:** Create `state_changes` table (Migration 0007) with `AUTOINCREMENT`. Reject adding `global_seq` to `execution_events`.
* **Rationale:**
  * Captures both evented and non-evented transitions (`UpsertInstance` and `AppendEvent` write to it in existing transactions).
  * Avoids destructive table rebuild of append-only `execution_events` (SQLite cannot `ALTER TABLE ADD COLUMN ... AUTOINCREMENT`).
  * `AUTOINCREMENT` guarantees monotonic ordering immune to SQLite `VACUUM` (tracked via `sqlite_sequence`).
  * Preserves G1 freeze without altering core ports or 12 event types.
* **Schema (Migration 0007):**
```sql
CREATE TABLE state_changes (
  seq           INTEGER PRIMARY KEY AUTOINCREMENT,
  instance_id   TEXT NOT NULL,
  namespace     TEXT NOT NULL,
  kind          TEXT NOT NULL,     -- 'event' | 'projection'
  event_id      TEXT,              -- populated when kind='event'
  status        TEXT NOT NULL,     -- instance status after transition
  current_steps TEXT NOT NULL,    -- JSON array of active step IDs
  changed_at    TEXT NOT NULL
);
CREATE INDEX idx_state_changes_ns ON state_changes(namespace, seq);
```

### ADR-2 (Founder Decision D2) — Retire `"default"` Namespace Overload
* **Recommendation:** Make `""` the explicit wildcard sentinel everywhere (`--all-namespaces`, `?all_namespaces=true`). Treat `"default"` as an ordinary literal namespace string.
* **Rationale:** Fixes silent zero returns in `StepStats`, unifies 3 duplicated CLI predicate hacks, and prevents encoding ambiguous semantics into GUI URLs and filters.

### ADR-3 (Founder Decision D3) — In-Module GUI Backend (`cmd/awis-server`)
* **Recommendation:** Place server in `cmd/awis-server` and handlers in `internal/api`.
* **Rationale:** Out-of-module builds are hard-blocked by Go's compiler from accessing `internal/storage` row types (`PluginRow`, `RecallRow`, `AuditRow`, `WaitRecord`). In-module placement eliminates the need to prematurely design and freeze a public Go SDK / row-type contract.

### ADR-4 — Server-Sent Events (SSE) over WebSockets
* **Recommendation:** Implement `GET /api/v1/stream?since=<seq>&namespace=<ns>` using standard library SSE.
* **Rationale:** Traffic is strictly unidirectional (server → client). Mutations use standard HTTP POST. Provides automatic reconnection and `Last-Event-ID` resumption without adding third-party WebSocket dependencies.

---

## 6. System Architecture & Operational Constraints

```
┌────────────────────────────────────────────────────────┐
│                   Browser SPA                          │
│     Canvas · Run List · Run Detail · Visual Editor     │
└───────────────────────────┬────────────────────────────┘
                            │ JSON / HTTP POST + SSE (/stream)
┌───────────────────────────▼────────────────────────────┐
│   cmd/awis-server (In-Module, Loopback 127.0.0.1)      │
│   internal/api: Handlers, DTOs, SSE Broadcast Hub       │
└───────────────────────────┬────────────────────────────┘
                            │ Direct In-Process Go Calls
┌───────────────────────────▼────────────────────────────┐
│   sdk.Runtime  +  internal/storage                     │
│   Engine embedded via goroutine rt.Start(ctx)          │
└───────────────────────────┬────────────────────────────┘
                            │ Single Writer Transaction Loop
┌───────────────────────────▼────────────────────────────┐
│            SQLite Database (WAL Mode)                  │
└────────────────────────────────────────────────────────┘
```

### Operational Rules & Constraints
1. **Single Engine Per Database (R7):** The engine maintains in-memory state (retry queues, pending activations, live waits). Running `awis-server` alongside `awis start` on the same DB file will cause split-brain memory divergence. `awis-server` replaces `awis start` and must enforce a PID/lock file.
2. **Tick-Quantized Latency Model:** The engine is pull-based without condvars or DB notification triggers. All transitions are quantized to the 100ms `TickInterval` (`--tick`). UI must never promise sub-second instant reactivity.
3. **Cooperative Cancellation Semantics:** In-flight execution steps are not killed abruptly. UI cancel buttons must display "Cancel requested" rather than "Stopped" until terminal state is achieved.
4. **Canvas Layout Persistence:** Node positions persist inside `WorkflowDefinition.Metadata["ui"]["positions"]` (parsed from YAML via existing `Metadata map[string]any`). Requires zero schema amendments.
5. **Subprocess Runner Environment Isolation (R13):** Plugin runners are scrubbed (`env` + `PATH`), but `internal/runner/subprocess` currently passes full host environment (including `ANTHROPIC_API_KEY`). Must be scrubbed before multi-user authoring is enabled.

---

## 7. API Layer, Error Contracts & Pagination Debt

### Complete API Surface (`/api/v1`)
| Method | Path | Target Engine / Storage Method | Notes & Gating Prerequisites |
|---|---|---|---|
| `GET` | `/workflows` | `StoragePort.ListWorkflows` | List all workflow definitions |
| `GET` | `/workflows/{id}/{version}` | `StoragePort.GetWorkflow` | Needs sentinel error for 404 mapping |
| `POST` | `/workflows` | `Runtime.RegisterWorkflow` | Registration only; immutable versioning |
| `POST` | `/workflows/validate` | `validate.Validate` | Returns full structured `Issue` list |
| `GET` | `/instances` | `Runtime.ListPaged` | Already cursor-paginated (100–1000) |
| `GET` | `/instances/{id}` | `Runtime.Status` + wait-record lookup | API layer must join `wait_records` |
| `GET` | `/instances/{id}/events` | `StoragePort.ReadEvents` | Requires query limit + cursor |
| `POST` | `/instances` | `Runtime.Submit` | Trigger workflow instance |
| `POST` | `/instances/{id}/signal` | `Runtime.Signal` | Deliver human / external signal |
| `POST` | `/instances/{id}/cancel` | `engine.Cancel` | Expose `compensate=true/false` in SDK |
| `GET` | `/stats/steps` | `Runtime.StepStats` | Gated on D2/G3 namespace resolution |
| `GET` | `/plugins` | `storage.ListPlugins` | In-module only; lists registered plugins |
| `GET` | `/stream` | SSE tailing `state_changes` | Gated on D1/G2 migration 0007 |
| `GET` | `/healthz` | Direct Ping / DB Check | Process liveness & DB reachability |

### Error Handling Gaps
* `GetWorkflow` returns a generic `fmt.Errorf("... not found")` without a typed sentinel.
* `cmd/awis/start.go` uses string-matching for `"already registered"` instead of type-asserting `*RegistrationError`.
* Both must be standardized to typed sentinels in milestone G1 to allow clean 404 vs 409 vs 500 HTTP mapping.

### Pagination Debt Status
* `ListInstancesPaged`: Bounded (cursor, 100 default / 1000 max).
* `ReadEvents`, `ReadEventRange`, `ReplayInstance`, `QueryHistory`: Unbounded. A limit/cursor must be added to `ReadEvents` before handling high event volumes.

---

## 8. Visual Editor, Graph Model & AI Authoring

### The 6 Canvas Relationship Types
The workflow canvas cannot treat graph edges as a single homogeneous list:
```
1. Unconditional Transition : Transition{From, To} (Solid Arrow)
2. Conditional Transition   : Transition.Condition (Labeled Arrow)
3. Error Transition         : Transition.OnError = true (Red Arrow)
4. Fallback Step            : Step.Fallback (Dotted Branch - Step-level property)
5. Compensation Chain       : WorkflowDefinition.Compensation (Reverse Recovery Path)
6. Retry & Timeout Policy   : Step.Retry & Step.WaitSignal (Step Badge / Property Sheet)
```
* **Crucial Asymmetry:** Fallback edges count toward node **reachability**, but are **excluded from cycle detection** (`validate.go:26-28`). A canvas that includes fallback edges in its cycle detector will reject valid workflows.

### Free Capabilities vs. What Must Be Built
* **Free Today:** `validate.Validate(def) []Issue` is completely pure (returns `{Code, StepID, Field, Message, Position}`). `expr.ParseTemplate` and `expr.ParseCondition` return exact byte offsets for error squiggles.
* **Must Be Built:**
  * **YAML Serializer (G6 / B-c):** No serializer exists in the repo. Must implement `yaml.Marshal` emitter matching parser struct tags, backed by round-trip tests.
  * **Variable Scope Autocomplete:** Scope is evaluated at runtime (`expr.Env`). Canvas editor must statically derive available variables by traversing upstream step `Outputs`.
  * **Palette Introspection Accessors (G7):** `engine.runners`, `NativeRunner.handlers`, and `plugin.Manager` are unexported with no `List*` methods. 3 small accessors needed to reflect actual installed plugins.
  * **Avoid `Step.Inputs` Naming Trap:** `Step.Inputs` is typed `InputSchema` but represents an execution-time template value map (not a JSON Schema definition).

### AI Generation & Repair Loop (Phase 5 / G9)
* AI generation leverages the `generate → validate → structured errors → retry` loop.
* Requires generating a formal `WorkflowDefinition` JSON Schema artifact (currently prompted from raw Go structs or markdown).
* **Live Intelligence Gate (Founder Decision D4):**
  * `modelQuality` was hardcoded to stale `claude-sonnet-4-5` (now updated to `claude-sonnet-5`).
  * `TestLiveSmoke` has never run against real Anthropic APIs in CI due to missing credentials in `ci.yml`.
  * D4 mandates establishing a scheduled, secret-gated live CI test to catch model ID deprecations.

---

## 9. Product Requirements Across the 5 Delivery Phases

### Phase 1 — Read-Only Dashboard
* **Goal:** Full visibility of definitions and historical instances without CLI.
* **Requirements:** FR-1.1 (paged instance list), FR-1.2 (instance details + wait record lookup), FR-1.3 (event history timeline), FR-1.4 (definitions list/detail), FR-1.5 (clean namespace filter).
* **Non-Goals:** No live streaming, no mutations (submit/cancel), no editing.
* **Acceptance:** Real data backing every view; accurate `"default"` vs all-namespace filtering.

### Phase 2 — Workflow Monitoring
* **Goal:** Live, truthful workflow monitoring over SSE.
* **Requirements:** FR-2.1 (live instance list/detail), FR-2.2 (live `waiting`, timeout-resumption, and cancel-requested transitions via `state_changes`), FR-2.3 (honest tick-quantized UI copy), FR-2.4 ("Cancel requested" badge).
* **Acceptance:** Submitting a workflow with a signal wait displays the `waiting` state in the UI within 100ms without page reload.

### Phase 3 — Workflow Management
* **Goal:** Full execution management and definition registration from GUI.
* **Requirements:** FR-3.1 (submit form with inputs), FR-3.2 (signal delivery button), FR-3.3 (cancel with `compensate` toggle — requires SDK fix), FR-3.4 (definition YAML paste/upload with structured validation error display).
* **Acceptance:** End-to-end integration tests verifying submit, signal, cancel-compensation, and registration.

### Phase 4 — Visual Workflow Editor
* **Goal:** Visual authoring, drag-and-drop step assembly, and round-trip YAML persistence.
* **Requirements:** FR-4.1 (6 relationship types rendered; cycle detection matches engine), FR-4.2 (inline validation diagnostics), FR-4.3 (canvas → YAML serializer save path), FR-4.4 (`Metadata.ui.positions` layout persistence), FR-4.5 (live introspection palette), FR-4.6 (expression squiggles).
* **Acceptance:** Workflow created on canvas, serialized, and submitted executes identically to hand-crafted YAML.

### Phase 5 — Full n8n-Class Platform
* **Goal:** Cost observability, AI generation, and secure multi-user deployment.
* **Requirements:** FR-5.1 (token/cost queries via `json_extract` + index on `StepCompleted` payload), FR-5.2 (AI prompt-to-workflow with JSON schema + repair loop), FR-5.3 (D4 live CI validation), FR-5.4 (authn/authz and per-namespace ACLs), FR-5.5 (subprocess env scrubbing).
* **Acceptance:** Token cost aggregations execute in bounded time; multi-user authorization prevents cross-namespace data access.

### Cross-Phase Non-Functional Requirements (NFRs)
* **NFR-1:** Build strictly against the binary and golden tests, never against stale `CLI_CONTRACT.md`.
* **NFR-2:** Enforce G1 freeze with golden JSON snapshots and `StoragePort` reflection tests.
* **NFR-3:** Loopback bind (`127.0.0.1`) only until G10 multi-user authz is fully deployed.
* **NFR-4:** Disclose engine tick quantization honestly across all UI states.

---

## 10. Milestone Roadmap (G0–G10), Critical Path & Concurrency

### Milestone Breakdown
| ID | Milestone Name | Scope & Key Deliverables | Est. | Prereqs |
|---|---|---|---|---|
| **G0** | Freeze Close-Out | Rebuild stale binary, D4 decision, fix test timeout flake | 1d | — |
| **G1** | Contract Hardening | Golden snapshots, `StoragePort` reflective pin, `GetWorkflow` sentinel | 2–3d | G0 |
| **G2** | Cursor & Eventing | Migration 0007 (`state_changes`), write hooks, `ReadChangesSince` | 2d | G1, D1 |
| **G3** | Namespace Resolution | Unify predicates, `ReadEventRange` wildcard, `--all-namespaces` | 2d | D2 |
| **G4** | API Server & SSE | `cmd/awis-server`, `internal/api`, `/api/v1` routes, SSE stream, `/healthz` | 5–7d | G2, G3, D3 |
| **G5** | Read-Only GUI | SPA shell, run list/detail, live timeline, event inspector, read-only canvas | 8–10d | G4 |
| **G6** | Serializer & Editor | YAML emitter + round-trip test, canvas editing, inline diagnostics | 8–12d | G5, B-c |
| **G7** | Palette Introspection| `List*` accessors on runners/plugins, live node palette | 3–4d | G6 |
| **G8** | Observability | Incremental step stats off cursor, token/cost dashboard via `json_extract` | 4–5d | G4 |
| **G9** | AI Generation | JSON Schema generator, validation repair loop, NL→workflow UI | 5–8d | G6, D4 |
| **G10**| Multi-User & Authz | Authn/authz, namespace ACLs, subprocess env scrubbing, remote bind | 10–15d | G5 |

### Concurrency & Parallel Execution Model
```
Week 1: [G0: Freeze] ──► [G1: Contracts] ──► [G2: Cursor (D1)]
                                                    │
Week 2:                  [G3: Namespaces (D2)] ◄────┼──► [G4: API Server (D3)]
                         (Runs parallel with G2/G4) │           │
Week 3:                                             │           ▼
                                                    └────► [G5: Live GUI]
                                                                │
Weeks 4+: ┌─────────────────────────┬───────────────────────────┴─────────────────────────┐
          ▼                         ▼                                                     ▼
   [G6: Serializer]          [G8: Observability]                                   [G10: Multi-User]
          │                         │                                                     │
   [G7: Palette]                    │                                                     │
          │                         │                                                     │
   [G9: AI Generation (D4)]         │                                                     │
```

### Critical Path:
$$\text{G0} \longrightarrow \text{G1} \longrightarrow \text{G2} \longrightarrow \text{G4} \longrightarrow \text{G5} \quad (\approx \text{3 Weeks to Demonstrable Live GUI})$$

### Execution Strategies
* **Recommended Path:** G0 → G1 → G2 in series (~5 days), G3 and G4 in parallel, then G5. Safely eliminates silent bugs and delivers a fully live GUI within a month.
* **Cheapest Path (False Economy):** Skip G2 and poll `ListPaged` every second. Delivers in ~2 weeks, but lacks live timeline and must be discarded.
* **Fastest Path:** Build G4 read-only against mocks while G2 lands. Parallelizes frontend early once G1 golden snapshots exist.
* **Safest Path:** Complete G0 → G1 → G2 → G3 completely before writing any frontend code.

---

## 11. Founder Decisions & Blast Radius Analysis

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                   FOUNDER DECISIONS (D1–D4)                                 │
├────┬──────────────────────┬─────────────────────────────┬───────────────────────────────────┤
│ ID │ Decision Summary     │ If Deferred                 │ If Decided Wrong                  │
├────┼──────────────────────┼─────────────────────────────┼───────────────────────────────────┤
│ D1 │ state_changes vs.    │ G2 cannot start; live view  │ Selecting global_seq leaves B-a   │
│    │ global_seq cursor    │ monitoring blocked.         │ open; stream lies about waiting.  │
├────┼──────────────────────┼─────────────────────────────┼───────────────────────────────────┤
│ D2 │ Retire "default"     │ G3 cannot start; StepStats  │ Postponing encodes ambiguous      │
│    │ namespace overload   │ metrics remain broken.      │ sentinel into permanent GUI URLs. │
├────┼──────────────────────┼─────────────────────────────┼───────────────────────────────────┤
│ D3 │ In-module server     │ Forces premature public Go  │ Separate service forces hasty     │
│    │ (cmd/awis-server)    │ SDK row-type redesign.      │ public SDK freeze.                │
├────┼──────────────────────┼─────────────────────────────┼───────────────────────────────────┤
│ D4 │ Live CI test gate    │ G0 cannot close; AI author- │ Shipping unverified risks silent  │
│    │ for Anthropic models │ ing ships with drift risk.  │ model deprecation outages.        │
└────┴──────────────────────┴─────────────────────────────┴───────────────────────────────────┘
```

---

## 12. Complete Risk Register (R1–R18) & Blocker Resolution Map

| Risk ID | Description & Architectural Impact | Severity | Mitigation & Action Plan |
|---|---|---|---|
| **R1** | Live stream silently omits `waiting` state transitions | **CRITICAL** | Implement ADR-1 / Migration 0007 (`state_changes` table) in G2. |
| **R2** | `StepStats` returns all zeroes with no error on namespace mismatch | **CRITICAL** | Implement ADR-2 (retire overload, add wildcard branch) in G3. |
| **R3** | GUI developed against diverging `CLI_CONTRACT.md` specification | **HIGH** | Pin actual binary behavior with golden snapshot tests in G1. |
| **R4** | G1 engine freeze is comment-only and not enforced by tests | **HIGH** | Add reflective `StoragePort` test and event snapshot tests in G1. |
| **R5** | Stale Anthropic model IDs in intelligence adapter | **MEDIUM** | Updated `modelQuality` to `claude-sonnet-5`; mandate D4 live CI gate. |
| **R6** | Scaffold ships invalid `timeout_action: cancel` | **CLOSED** | Patched scaffold, examples, and 5 validator enum rules (2026-08-30). |
| **R7** | Split-brain memory corruption from 2 engines on 1 SQLite DB | **MEDIUM** | Add PID/lock file check to `cmd/awis-server`; replace `awis start`. |
| **R8** | Unbounded memory/network load on `ReadEvents` queries | **MEDIUM** | Introduce cursor pagination to `ReadEvents` before high-volume load. |
| **R9** | Exposure of unauthenticated endpoints on network interfaces | **CRITICAL (Remote)** | Hardcode `127.0.0.1` loopback binding until G10 authz milestone. |
| **R10**| `webhook` trigger type is a declared but unhandled enum | **LOW** | Hide in visual palette until API receiver is implemented. |
| **R11**| `pending` instance status is declared but unreachable | **LOW** | Remove from UI status filters or document as reserved. |
| **R12**| Deep SQLite lock-in prevents straightforward Postgres port | **LOW (V1)** | Accept SQLite for V1/V2; Postgres adapter deferred. |
| **R13**| Subprocess runner inherits parent process environment / secrets | **HIGH (Multi-User)**| Scrub environment in `subprocess.go` before Phase 4/5 multi-user access. |
| **R14**| `TestSystemRehearsalInitStartSubmitTrace` flakes under load | **MEDIUM** | Increase hardcoded 10s deadline in `cmd/awis` test suite. |
| **R15**| Low test coverage (47.1%) in `cmd/awis` package | **MEDIUM** | Increase contract test coverage alongside G1 snapshot work. |
| **R16**| Weak assertions in `recall_test.go` and `plugin_remove_test.go` | **LOW** | Strengthen assertions as part of test suite maintenance. |
| **R17**| Stale unmerged `worktree-agent-*` branches in repository | **LOW** | Prune dead worktrees during pre-release housekeeping. |
| **R18**| Upgrade break: pre-existing bad definitions rejected on Submit | **MEDIUM** | Disclose in upgrade release notes; validator provides exact line fix. |

---

## 13. Direct Answers to Strategic Questions

1. **What work remains before the engine can be frozen?**  
   Hygiene and validation closeout: D4 decision, deleting/rebuilding stale root binaries, adjusting the flaky 10s rehearsal test timeout, and pruning dead worktrees.
2. **What work remains before GUI development can begin?**  
   Nothing for the frontend and read-only views (starts immediately). For live monitoring: Migration 0007 (G2). For visual editor saving: YAML serializer (G6).
3. **Can the GUI be developed independently once started?**  
   Yes, provided it resides in-module (`cmd/awis-server`). After G2 (cursor) and G3 (namespaces) land, engine work remains untouched until G7 (introspection) and G10 (authz).
4. **Were there hidden architectural blockers?**  
   Yes, three: the Go `internal/` module boundary blocking out-of-module storage access, the silent zero returns in `StepStats` due to namespace overloading, and the 11-point divergence between `CLI_CONTRACT.md` and the binary.
5. **Will any future GUI requirement force an engine redesign?**  
   No. All future needs (SSE stream, introspection, layout metadata, incremental stats) are strictly additive. Sub-tick latency would require a push-based engine rewrite, so the 100ms pull-based tick quantization model should be maintained.
