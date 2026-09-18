# GUI Phase 1 — Product Requirements

**Scope: a read-only operational dashboard, and nothing else.** No workflow editing,
creation, or mutation of any kind; no auth; no observability expansion beyond what the
existing API already returns; no serializer work. Everything below is buildable today
against the API surface that shipped this session (`internal/api/`, `cmd/awis-server`) —
see `GUI_PHASE1_API_MATRIX.md` for the exact route-to-field mapping.

---

## 1. Users

**The operator.** The same person who runs `awis submit`/`awis status`/`awis signal`
from the CLI today. Single user, local machine, no multi-tenancy — matches the current
security posture exactly (`cmd/awis-server` binds `127.0.0.1` only; there is no
identity/session concept anywhere in the codebase). This GUI does not change who can use
AWIS or how many people can use it at once; it gives the existing single operator a
second way to look at the same data the CLI already exposes.

There is no second persona for Phase 1. A "workflow author" (someone who edits or
creates workflows) and a "team member with restricted access" (multi-user, authz) are
both explicitly out of scope — the former needs the serializer (G6), the latter needs
G10. Designing for personas this phase doesn't serve would be exactly the kind of
speculative scope this exercise forbids.

## 2. Use cases

- **"What's running right now?"** — see active (running/waiting) instances without
  running `awis status` by hand.
- **"Why is this instance stuck?"** — open one instance and see its current step,
  and — if waiting — what signal it's waiting on and how long until timeout.
- **"What actually happened to this instance?"** — read its full event history in
  order, to debug a failure or confirm a workflow behaved as expected.
- **"What workflows exist, and what do they do?"** — browse registered definitions and
  see their structure (steps, transitions, triggers) without opening the YAML file.

No use case in this phase requires writing anything back to the engine. Every one of
them is already answerable by an existing CLI command run by hand; this GUI's entire
value proposition is *not having to run the CLI command by hand*.

## 3. Screens

Full detail in `GUI_PHASE1_SCREEN_SPEC.md`. Named here for navigation purposes:

1. **Instance List** — the landing screen. Paginated, filterable by namespace and status.
2. **Instance Detail** — one instance's full state, including the wait-record wrapper
   (`signal_name`/`timeout_remaining_s`) the CLI's `status --json` already exposes.
3. **Event Timeline** — one instance's ordered event history, reached from Instance
   Detail.
4. **Workflow List** — registered definitions, filterable by namespace.
5. **Workflow Detail** — one definition's steps/transitions/triggers, as structured
   tables (explicitly not a graph/canvas rendering — see `GUI_PHASE1_ARCHITECTURE.md`
   for why, and `GUI_PHASE1_BACKLOG.md` for where a visual rendering would eventually
   belong).

A sixth candidate — a cross-status aggregate "Dashboard" (counts of running/waiting/
failed/etc.) — was evaluated and is addressed explicitly in `GUI_PHASE1_SCREEN_SPEC.md`
and scoped in `GUI_PHASE1_BACKLOG.md`; see those documents for the MVP/Beta placement
and the reasoning (the existing API can only build it via N parallel per-status calls,
not a single aggregate query).

## 4. Navigation

```
┌────────────────────────────────────────┐
│  Top nav: [Instances] [Workflows]       │
└────────────────────────────────────────┘

Instances (landing)
  → click a row → Instance Detail
      → "Events" tab/link → Event Timeline

Workflows
  → click a row → Workflow Detail
```

Two top-level sections, each with a list→detail drill-down. Event Timeline is reached
only through Instance Detail — it is never a top-level nav item, because an event
history is meaningless without knowing which instance it belongs to. This is a
two-level navigation depth maximum (list → detail → sub-view), deliberately shallow.

## 5. Acceptance criteria

- Every screen is backed by a real API call returning live data from the current
  `awis-server` instance — no mock/fixture data in the shipped product (the fixture
  approach used during initial frontend scaffolding, per the earlier `GUI_START_LINE.md`
  finding, is a development-time technique only; it does not ship).
- The namespace filter's default state means "all namespaces," and this is correct
  against the real API today — `ListWorkflows`/`ListPaged`'s empty-string namespace
  parameter already means "no predicate" (confirmed in `internal/api/workflows.go` and
  `instances.go`, both built and tested this session). No dependency on the G3 namespace
  milestone for this to work correctly.
- A waiting instance's detail view shows a non-null signal name and a countdown that
  visibly decreases on refresh — the concrete, user-facing form of the B-a finding this
  whole API layer exists to fix, now verifiable in a browser rather than only via `curl`.
- Every event in the timeline renders in ascending order with no gap or duplicate across
  a paginated fetch (mirrors `TestHandleListEvents_Pagination`'s server-side guarantee,
  extended to the client's page-stitching logic).
- Loading and error states are visibly distinct from "no data" — an empty instance list
  must not look identical to a failed API call. See `GUI_PHASE1_SCREEN_SPEC.md` for the
  exact states each screen must render.
- Data is not stale beyond one poll interval (~15s, see `GUI_PHASE1_ARCHITECTURE.md`) or
  a manual refresh, whichever the user triggers first — no promise of live/instant
  updates, since that needs the SSE work (G2/G4-SSE) this phase deliberately excludes.

## 6. Explicit non-goals (restated from the task brief, for a single source of truth)

No workflow editing, creation, or versioning. No canvas/graph editing or drag-and-drop.
No submit/signal/cancel actions from the GUI — the CLI remains how an operator drives
workflows during this phase. No authentication or per-user access control. No new
observability/metrics infrastructure. No AI-assisted features. No live/streaming updates
(SSE). Building any of these now would be premature relative to the API surface that
actually exists.

---

*Companion documents: `GUI_PHASE1_ARCHITECTURE.md` (how it's built), `GUI_PHASE1_SCREEN_SPEC.md`
(every screen in detail), `GUI_PHASE1_API_MATRIX.md` (screen → route → field mapping),
`GUI_PHASE1_BACKLOG.md` (MVP/Beta/Nice-to-have).*
