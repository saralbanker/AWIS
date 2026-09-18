# GUI Phase 1 — API Matrix

Screen → route → response fields actually used → gaps. Every route and field name below
is copied from the real, tested implementation (`internal/api/*.go`,
`internal/core/{event,workflow,step}.go`), re-verified against source while writing this
document — not from the earlier planning corpus's aspirational route table.

---

## Instance List

| | |
|---|---|
| Route | `GET /api/v1/instances?namespace=&status=&limit=&offset=` |
| Response | `{instances: [instanceEntry], total, limit, offset}` |
| Fields used | `instances[].instance_id`, `.definition_id`, `.version`, `.status`, `.current_steps[]`, `.updated_at` (sort/display), `total` (pagination footer), `limit`/`offset` (page-state sync) |
| Fields available but unused here | `instances[].created_at` (available if a "started" column is wanted later) |
| Gaps | None for this screen. `status` query param takes one exact `InstanceStatus` value — the UI's status filter must be a fixed dropdown of the 9 known values (`pending`/`running`/`waiting`/`completed`/`failed`/`cancelled`/`compensating`/`compensated`/`compensation_failed`), not free text, since the API does no fuzzy matching. |

## Instance Detail

| | |
|---|---|
| Route | `GET /api/v1/instances/{id}` |
| Response | `instanceEntry` fields + `inputs`, `outputs`, `signal_name` (nullable), `timeout_remaining_s` (nullable) |
| Fields used | all of them — this is the one screen that uses the full DTO, including `inputs`/`outputs` (rendered as formatted JSON, since they're `map[string]any` with no fixed schema) |
| Gaps | None. **Note:** `signal_name`/`timeout_remaining_s` are only non-null when `status == "waiting"` — the frontend must not treat a null value here as an error state for a non-waiting instance. |

## Event Timeline

| | |
|---|---|
| Route | `GET /api/v1/instances/{id}/events?from=&limit=` |
| Response | `{events: [ExecutionEvent], next_cursor (nullable)}` |
| Fields used | `events[].event_id` (React-key-equivalent/DOM-key), `.event_type`, `.step_id`, `.emitted_at`, `.sequence_num` (ordering/dedup guard), `.payload` (raw JSON, parsed per-type — see below), `next_cursor` (pagination) |
| Gaps | **`.payload`'s shape is not machine-readable** — it's `json.RawMessage`, typed only by the doc comments on the 12 `EventType` constants in `internal/core/event.go`. This is real, current-code Tier-1 evidence, not a guess, so the frontend CAN hardcode a 12-entry formatter table from it today — but if the payload shape ever drifts from those comments, nothing catches it client-side. See `GUI_PHASE1_BACKLOG.md` for the payload-formatter table this screen needs, copied verbatim from the doc comments: |

| EventType | Documented payload | Suggested one-line summary |
|---|---|---|
| `WorkflowStarted` | `{inputs}` | "Workflow started" |
| `StepStarted` | `{step_id, attempt, inputs}` | "Step {step_id} started (attempt {attempt})" |
| `StepCompleted` | `{step_id, attempt, outputs, duration_ms}` | "Step {step_id} completed in {duration_ms}ms" |
| `StepFailed` | `{step_id, attempt, error, retrying}` | "Step {step_id} failed: {error}" + "(retrying)" if `retrying` |
| `StepFallbackActivated` | `{step_id, fallback_step_id, reason}` | "Step {step_id} fell back to {fallback_step_id}: {reason}" |
| `SignalReceived` | `{signal_name, payload}` | "Signal '{signal_name}' received" |
| `WorkflowCompleted` | `{outputs, duration_ms}` | "Workflow completed in {duration_ms}ms" |
| `WorkflowFailed` | `{step_id, error}` | "Workflow failed at {step_id}: {error}" |
| `WorkflowCancelled` | `{reason}` | "Workflow cancelled: {reason}" |
| `WorkflowCompensating` | `{from_step}` | "Compensation started from {from_step}" |
| `WorkflowCompensated` | `{}` | "Compensation completed" |
| `WorkflowCompensationFailed` | `{step_id, error}` | "Compensation failed at {step_id}: {error}" |

An event type not in this table (future-added, or a bug) must degrade to showing the raw
`event_type` string and the raw JSON payload, never crash the row.

## Workflow List

| | |
|---|---|
| Route | `GET /api/v1/workflows?namespace=` |
| Response | `[{id, version, namespace, step_count}]` |
| Fields used | all four |
| Gaps | **No update timestamp, no "last run" info.** The list DTO is intentionally minimal (`internal/api/workflows.go`'s own doc comment: mirrors the CLI's `workflowEntryJSON` shape, not the full definition). A "workflows sorted by recent activity" view is not buildable from this route alone — would need a join against instance data the API doesn't offer as one query. Not needed for Phase 1's use cases; flagged for `GUI_PHASE1_BACKLOG.md`. |

## Workflow Detail

| | |
|---|---|
| Route | `GET /api/v1/workflows/{id}/{version}` |
| Response | full `WorkflowDefinition`: `schema_version`, `id`, `version`, `namespace`, `name`, `description`, `triggers[]`, `steps[]`, `transitions[]`, `initial_step`, `final_steps[]`, `compensation` (nullable), `timeout`, `metadata` |
| Fields used | `name`, `description`, `namespace`, `version` (header); `steps[]` — `.id`, `.name`, `.type`, `.handler`, `.retry` (nullable), `.timeout` (nullable), `.fallback` (nullable), `.wait_signal` (nullable, only present on `type=signal` steps) rendered as a table; `transitions[]` — `.from`, `.to`, `.condition` (empty means unconditional), `.on_error` (boolean: an error-transition badge, distinct from a normal edge) rendered as a table; `triggers[]` — `.type`, `.config` rendered as a list; `initial_step`/`final_steps[]` (badges on the steps table) |
| Fields available but unused | `schema_version`, `metadata`, `compensation`, per-step `inputs`/`outputs` (JSON-Schema-shaped, not rendered in Phase 1 — see `GUI_PHASE1_BACKLOG.md`) |
| Gaps | None — `Transition{From, To, Condition, OnError}` confirmed in full (`internal/core/workflow.go:41-51`). No graph/canvas rendering — steps and transitions are shown as two separate tables, not a rendered graph, per this phase's explicit exclusion of canvas work. One real limitation worth naming: `Step.Fallback` (a step-level field, not a `Transitions` entry) and `WorkflowDefinition.Compensation` (a separate ordered plan, not read by this screen at all in Phase 1) are two of the six relationship types the original `GUI_ARCHITECTURE.md` §8.1 names for a full graph view — a *tables-only* Workflow Detail correctly shows fallback (as a per-step field) but does not show compensation at all. Acceptable for Phase 1 (compensation is a failure-path detail, not needed to answer "what does this workflow normally do"); flagged in `GUI_PHASE1_BACKLOG.md` as a Beta addition, not an MVP gap. |

## (Evaluated, not a distinct MVP route) Dashboard / landing aggregate

No dedicated backend route exists for cross-status counts. Two ways to build a landing
view without one:
1. **N parallel calls**, one per `InstanceStatus` value, `?status=X&limit=1`, reading
   only `.total` from each — authoritative counts, 9 requests.
2. **One call**, `?limit=20` with no status filter, sorted by `updated_at` — "recently
   active instances," zero aggregate math, one request, not authoritative counts.

See `GUI_PHASE1_BACKLOG.md` for which (if either) ships in MVP.

---

## Summary: gaps that are backend work, not frontend work

| Gap | Which screen | Severity |
|---|---|---|
| No static-file serving in `cmd/awis-server` — `GET /` 404s today | all (deployment blocker, not a screen-specific gap) | **Blocks shipping**, not blocks building |
| No `/stats`-style aggregate endpoint | Dashboard/landing (if the counts variant is chosen) | Low — the N-parallel-call workaround exists |
| No machine-readable event-payload schema | Event Timeline | Low — doc comments suffice today, see table above |
| `Transition`'s full field set unconfirmed past `From`/`To` in this pass | Workflow Detail | Needs a 1-line re-check before implementation, not new backend work |

---

*Companion documents: `GUI_PHASE1_PRD.md`, `GUI_PHASE1_SCREEN_SPEC.md`,
`GUI_PHASE1_ARCHITECTURE.md`, `GUI_PHASE1_BACKLOG.md`.*
