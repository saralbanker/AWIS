# AWIS GUI — Product Requirements

Status: proposed. Requirements below are scoped strictly to what the engine can
actually support at each phase, per the evidence in `GUI_MASTER_PLAN.md`. Where a
requirement depends on engine work that doesn't exist yet, it's flagged explicitly
rather than assumed.

Target: an n8n-class visual workflow product, reached in five phases, each one a
shippable increment rather than a throwaway prototype.

---

## Phase 1 — Read-only dashboard

**Goal.** An operator can see what the engine is doing without touching the CLI.

**User stories**
- As an operator, I can see a list of workflow instances (running, waiting, completed,
  failed, cancelled) without running `awis status` by hand.
- As an operator, I can open one instance and see its current state, its step history,
  and (if waiting) what signal it's waiting on and how long until timeout.
- As an operator, I can see the list of registered workflow definitions and their
  versions.
- As an operator, I can filter the instance list by namespace and status.

**Functional requirements**
- FR-1.1 — Paginated instance list, backed by `Runtime.ListPaged` (already a real
  cursor; no engine change needed).
- FR-1.2 — Instance detail view showing status, current step(s), and — for waiting
  instances — signal name and timeout remaining (`Runtime.Status` today omits both;
  the API layer must add the wait-record lookup — see `GUI_ARCHITECTURE.md` §6).
- FR-1.3 — Event history / timeline per instance, backed by `ReplayInstance`.
- FR-1.4 — Workflow definition list and detail, backed by `ListWorkflows`/`GetWorkflow`.
- FR-1.5 — Namespace filter. **Blocked on D2/G3** if it's to mean what it says — before
  that fix, filtering by `"default"` silently returns *every* namespace's instances.
  Ship with an explicit "all namespaces" toggle rather than the ambiguous default.

**Non-goals for this phase**
- No live updates. A page refresh (or short poll) is acceptable; do not build
  streaming infrastructure yet.
- No submit/signal/cancel actions — view only.
- No workflow editing.

**Acceptance criteria**
- Every view is backed by a real API endpoint returning data the CLI would show for
  the same query — no mock data in the shipped product.
- Namespace filtering behaves correctly for a namespace literally named `default`
  (i.e., D2/G3 must land before FR-1.5 ships, or the toggle must be the *only* way to
  select "all namespaces").

**Depends on:** G0, G1, G4 (minus the SSE route), portions of G5. See
`GUI_ROADMAP.md`, Phase 1.

---

## Phase 2 — Workflow monitoring

**Goal.** The dashboard updates live, and the live view is *truthful* — it does not
silently freeze on the most common state transition in the system.

**User stories**
- As an operator, I watch a running instance and see its status update without
  reloading the page.
- As an operator, when an instance parks on a signal wait, I see it transition to
  "waiting" in the UI within about a tick — not never.
- As an operator, when I request a cancel, I see "cancel requested" immediately and
  the terminal state once the in-flight step actually finishes (cancel is
  cooperative, not preemptive — see `GUI_ARCHITECTURE.md` §5.2).

**Functional requirements**
- FR-2.1 — Live-updating instance list and detail view over SSE (`GET /api/v1/stream`).
- FR-2.2 — The live timeline must show the `waiting` transition, the resumption on
  signal-timeout `continue`, and cancellation-*requested* (not just
  cancellation-applied) — **this is the B-a finding, and it is a hard blocker, not a
  nice-to-have.** Building this phase before migration 0007 (G2) ships a live view
  that visibly hangs — on a `StepStarted` — the moment any instance parks, which is
  worse than no live view at all, because it looks broken rather than absent.
- FR-2.3 — UI copy must reflect tick-quantized latency honestly: no spinner or
  animation implying sub-second reactivity. 100ms tick interval is the ceiling.
- FR-2.4 — Cancel control reads as "Cancel requested," never "Stopped," until the
  terminal event actually lands.

**Non-goals for this phase**
- No guarantee of sub-tick latency — do not scope work to reduce it; it would require
  reworking the engine's pull-based tick loop, which is out of scope for the GUI
  program entirely (see `GUI_ARCHITECTURE.md` §5.2).

**Acceptance criteria**
- Given a workflow with a `wait_signal` step, submitting it and watching the GUI shows
  a `waiting` state transition in the live view within one tick interval, without a
  page reload. This is the concrete, testable form of the B-a fix.

**Depends on:** G2 (migration 0007), the SSE route in G4, remainder of G5. **Hard
blocked on decision D1.**

---

## Phase 3 — Workflow management

**Goal.** An operator can drive workflows from the GUI instead of the CLI.

**User stories**
- As an operator, I can submit a new instance of a registered workflow from the GUI.
- As an operator, I can deliver a signal to a waiting instance from the GUI.
- As an operator, I can cancel a running instance, with the option to trigger
  compensation.
- As an operator, I can register a new workflow definition (upload/paste YAML).

**Functional requirements**
- FR-3.1 — Submit form: pick a definition + version, provide inputs, submit.
- FR-3.2 — Signal action on a waiting instance's detail view.
- FR-3.3 — Cancel action, with a compensate toggle. **Requires an engine-layer fix
  first**: `Runtime.Cancel` currently hardcodes `compensate=false`; only the CLI
  bypasses the SDK to reach `--compensate`. The API layer needs this exposed properly,
  not replicate the CLI's bypass.
- FR-3.4 — Register-definition form (YAML paste or file upload), validated via
  `POST /workflows/validate` before submission, surfacing the full structured `Issue`
  list (code, field, message, position) — not just the line+message the CLI currently
  exposes.

**Non-goals for this phase**
- No visual editing of the workflow graph — this phase accepts/produces raw YAML.
- No update-in-place for definitions — registering an existing id+version conflicts;
  a new version is required, matching current engine behavior.

**Acceptance criteria**
- All four actions (submit, signal, cancel, register) are exercised end-to-end against
  the real engine in an integration test, not mocked.
- A cancel with `compensate=true` actually runs the compensation chain, verified
  against `WorkflowDefinition.Compensation`.

**Depends on:** G4 (API server — this phase adds no new engine milestone beyond it),
plus the `Cancel`/`compensate` SDK fix called out in FR-3.3.

---

## Phase 4 — Visual workflow editor

**Goal.** An operator can build and modify a workflow graph visually and save it back
as a valid workflow definition.

**User stories**
- As a workflow author, I can drag steps onto a canvas and connect them.
- As a workflow author, I see validation errors inline, at the step/field that caused
  them, as I edit — not only after I try to save.
- As a workflow author, I can add fallback and compensation steps, and the canvas
  makes clear these are different relationships from a normal transition.
- As a workflow author, I can save my edits back to a YAML definition and re-register
  it.

**Functional requirements**
- FR-4.1 — Canvas renders all six relationship kinds distinctly (unconditional
  transition, conditional transition, error transition, fallback, compensation,
  retry/timeout policy) — see `GUI_ARCHITECTURE.md` §8.1 for where each lives in the
  data model. **A canvas that treats fallback edges as ordinary edges for cycle
  detection will reject graphs the engine accepts** — this must be tested explicitly.
- FR-4.2 — Inline validation using `validate.Validate`'s structured `Issue` list —
  this is available today with zero engine changes.
- FR-4.3 — Save path: canvas → `WorkflowDefinition` → YAML, round-tripped through a
  serializer that does not exist yet (B-c). **This is the single largest piece of net
  new engine-adjacent work in the whole program** — building the editor's save button
  before this exists is not possible, only deferrable.
- FR-4.4 — Node positions persist via `WorkflowDefinition.Metadata.ui.positions` — no
  schema change required (`GUI_ARCHITECTURE.md` §8.3).
- FR-4.5 — Node palette reflects the step types and (once G7 lands) the plugin
  capabilities actually registered in this deployment, not a hardcoded list that can
  drift from reality.
- FR-4.6 — Expression fields (conditions, templates) get inline error squiggles from
  `expr.ParseTemplate`/`expr.ParseCondition`'s byte-offset error positions.

**Non-goals for this phase**
- No autocomplete of in-scope variables at first ship — the engine has no
  variable-scope enumeration (`expr.Env` resolves at runtime only); this is deferred
  work the editor team should scope separately once the graph-walking logic exists.
- No collaborative/multi-editor real-time editing — single-editor-at-a-time is
  sufficient for V1.

**Acceptance criteria**
- A workflow built entirely in the visual editor, saved, and resubmitted produces
  identical runtime behavior to the same workflow authored by hand in YAML — verified
  by a round-trip test (author in canvas → serialize → parse → compare `Transition`,
  `Compensation`, `Fallback` sets).
- `Step.Inputs` is presented in the UI as what it actually is (an execution-time
  template value map), not as a JSON-Schema field editor — this is called out
  specifically because the field's name and doc comment are misleading
  (`GUI_ARCHITECTURE.md` §8.2).

**Depends on:** G3 (clean namespace semantics before an editor writes definitions),
G6 (serializer), G7 (introspection for the palette).

---

## Phase 5 — Full n8n-class experience

**Goal.** Cost/performance observability, AI-assisted authoring, and safe multi-user
operation — the features that separate a workflow *viewer* from a workflow *platform*.

**User stories**
- As an operator, I can see per-step token usage and cost, aggregated by workflow and
  by time range.
- As a workflow author, I can describe a workflow in natural language and get a
  generated, validated draft to refine rather than author from scratch.
- As a team lead, I can give a teammate access to the GUI without giving them the
  operator's own credentials or full filesystem/network access.

**Functional requirements**
- FR-5.1 — Cost dashboard: per-step `{adapter, model, tokens_used}` is already
  persisted in every `StepCompleted` event payload (verified end-to-end by an existing
  test), but the payload is opaque JSON with no index and no aggregation layer today.
  Build the query layer on `json_extract` plus a generated column/index, fed
  incrementally off the `state_changes` cursor (G8) rather than full-table scans.
- FR-5.2 — AI-assisted generation: expose the `validate.Validate` repair loop (generate
  → validate → structured errors → retry) through the API, backed by a generated JSON
  Schema artifact for `WorkflowDefinition` (none exists today — the LLM is currently
  prompted from a markdown doc or the Go structs directly).
- FR-5.3 — AI generation ships gated on decision D4: either behind a scheduled,
  secret-gated CI job that asserts the live model's `model` field in its response (so
  a stale model ID like the one already found and fixed fails loudly), or explicitly
  labeled "unverified" in the product if the founder chooses to ship without that
  gate.
- FR-5.4 — Multi-user: authn, per-namespace authorization, and non-loopback binding.
  Zero of this exists today — there is no user/role/session/tenant concept anywhere in
  the codebase. Scope as a full milestone (G10), not a checkbox.
- FR-5.5 — Before FR-5.4 ships to any user other than the original operator, subprocess
  steps must have their environment scrubbed the way plugin execution already is
  (`internal/runner/subprocess/subprocess.go` currently inherits the full parent
  environment, including API keys). This is a security precondition of multi-user
  support, not a separate nice-to-have.

**Non-goals for this phase**
- No OS-level sandboxing (seccomp/namespaces/chroot/cgroups) for workflow execution —
  out of scope for this program; track separately if untrusted-author execution
  becomes a real requirement.
- No Postgres or other non-SQLite storage backend — aspirational only; no adapter
  exists and nothing above requires one.

**Acceptance criteria**
- A cost query for "total tokens spent by workflow X in the last 7 days" returns in
  bounded time regardless of total event volume (i.e., it uses the incremental
  aggregate, not a full scan).
- A second GUI user, given access to only namespace `team-a`, cannot read or act on
  instances in namespace `team-b` — verified by an authorization test, not just UI
  hiding.

**Depends on:** G8 (observability), G9 (AI generation, itself gated on D4 and G6),
G10 (multi-user, gated only on G5 — can be built in parallel with G6–G9).

---

## Cross-phase requirements

These apply to every phase above and are not restated per-phase:

- **NFR-1 — Build against the binary, not the doc.** `docs/CLI_CONTRACT.md` diverges
  from the actual binary on at least eleven points (`GUI_MASTER_PLAN.md` §6.3). No
  API or frontend code may be written against that document; pin behavior with golden
  tests against the real binary/API instead (G1).
- **NFR-2 — The G1 freeze needs enforcement, not just documentation.** Add a snapshot
  test on `ExecutionEvent` serialization and a reflective test pinning `StoragePort`
  at its current method count before any GUI code depends on either shape.
- **NFR-3 — Loopback-only until G10.** `cmd/awis-server` binds to `127.0.0.1` for every
  phase before multi-user ships. Do not expose it to a network interface earlier for
  "just a demo" — there is no authz to fall back on.
- **NFR-4 — Every phase's UI must disclose tick-quantized latency honestly** where it's
  relevant (this matters starting Phase 2) rather than implying instant reactivity the
  engine cannot deliver.

---

*Companion to `GUI_MASTER_PLAN.md` (evidence base), `GUI_ARCHITECTURE.md` (system
design), `GUI_ROADMAP.md` (sequencing), and `GUI_DEPENDENCY_MAP.md` (what blocks
what).*
