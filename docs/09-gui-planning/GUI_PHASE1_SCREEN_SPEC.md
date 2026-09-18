# GUI Phase 1 — Screen Spec

Every screen defined in enough detail to start building without further design
decisions. Field names match `GUI_PHASE1_API_MATRIX.md` exactly. MVP/Beta placement per
screen is in `GUI_PHASE1_BACKLOG.md`; this document specifies every screen regardless of
which slice it ships in, since all six were required to be evaluated.

---

## 1. Instance List (landing screen)

**Purpose.** "What's running right now?" — the default view on load.

**Layout.** A filter bar above a table.

**Filter bar:**
- Namespace: a text input or dropdown populated from the distinct namespaces seen in
  the current page of results (no dedicated "list namespaces" endpoint exists — see
  `GUI_PHASE1_API_MATRIX.md`). Default: empty ("All namespaces").
- Status: a dropdown of the 9 fixed `InstanceStatus` values plus "All." Default:
  "All" for the general list, but the landing-page *entry point* should default this
  filter to the active subset (`running`, `waiting`, `pending`) so an operator's first
  view is "what needs attention," not a wall of completed instances — implemented
  client-side as three separate filtered requests OR by relaxing the default to "All"
  and sorting active statuses to the top; either is acceptable, pick whichever is
  simpler to implement against a single `?status=` param (the API takes one status
  value per call, not a list).

**Table columns:** Instance ID (truncated, full value on hover/title attr),
Definition ID, Version, Status (colored badge — active statuses visually distinct from
terminal ones), Current Step(s) (joined list), Last Updated (relative time, e.g. "2m
ago," from `updated_at`).

**Row interaction:** click anywhere on a row → navigate to Instance Detail for that
`instance_id`.

**Pagination.** Offset-based, matching the API exactly (`limit`/`offset` query params,
`total` in the response). A "Page X of Y" control plus Prev/Next; page size fixed at 50
(comfortably under the API's 100 default / 1000 max — see
`GUI_PHASE1_ARCHITECTURE.md` for why a smaller client page size than the server default
is deliberate).

**Empty state.** "No instances match these filters" — visually distinct from a loading
spinner and from an error banner. Do not render an empty table with just a header row
and no message; that reads as broken, not empty.

**Loading state.** A visible loading indicator on first load and on filter change;
subsequent poll-driven refreshes update the table in place without a full-page spinner
(avoid flicker on a 15s poll).

**Error state.** A dismissible banner ("Could not load instances: {error}") with a
manual retry button. The table below it keeps showing the last-successful data (stale
but visible) rather than clearing to empty on a transient fetch failure.

**Acceptance criteria.** Submitting a workflow via the CLI while this screen is open and
polling causes the new instance to appear within one poll interval, without a manual
reload.

---

## 2. Instance Detail

**Purpose.** "Why is this instance stuck?" / full state of one instance.

**Layout.** A header block plus two collapsible sections.

**Header:** Instance ID, Definition ID + Version (link to Workflow Detail for that
definition), Status badge, Current Step(s), Created At, Updated At.

**Wait info (shown only when `status == "waiting"`):** Signal Name (`signal_name`) and a
live countdown from `timeout_remaining_s` — rendered as "Xm Ys remaining" and
re-computed client-side between polls from the last-fetched value plus elapsed wall
time, so the countdown visibly ticks rather than only updating once per 15s poll. If
`timeout_remaining_s` is null while `status == "waiting"`, show "No timeout configured"
— this is a valid state (an unbounded wait), not an error.

**Inputs / Outputs sections:** each a collapsed-by-default, pretty-printed JSON block
(from the `inputs`/`outputs` map fields — no fixed schema exists for these, so a generic
JSON tree/pretty-print is the correct rendering, not a form).

**Navigation:** a prominent "View Events" link/tab to Event Timeline for this instance.

**Empty/loading/error states:** same conventions as Instance List. A 404 (instance not
found — e.g., the operator navigated to a stale/mistyped ID) must render a distinct "This
instance does not exist" message, not a generic error banner, since the API's 404 is a
real, structured signal (`storage.ErrInstanceNotFound`), not a fetch failure.

**Acceptance criteria.** Opening a waiting instance shows a non-null signal name and a
countdown that has visibly decreased after 15+ seconds on the page.

---

## 3. Event Timeline

**Purpose.** "What actually happened?" — reached only from Instance Detail.

**Layout.** A reverse-chronological or chronological list (chronological — oldest
first — matches how an operator reads a story: start → what happened → current state;
reverse-chronological is a valid alternative if user testing says otherwise, but
chronological is the default recommendation here since `sequence_num` ascending is also
the API's native order, requiring no client-side re-sort).

**Row content, per event:** a timestamp (`emitted_at`, relative + full on hover), the
event type as a label, and the type-specific one-line summary from
`GUI_PHASE1_API_MATRIX.md`'s 12-row table. Each row is expandable to show the full raw
`payload` as pretty-printed JSON.

**Unrecognized event type handling.** If `event_type` is not one of the 12 known values
(future addition, or a bug), render the raw `event_type` string and the raw payload with
no summary line — never throw/crash the row.

**Pagination.** Cursor-based, matching the API (`?from=&limit=`, `next_cursor` in the
response) — NOT offset-based like Instance List. "Load more" appends the next page to
the bottom of the list rather than replacing it (natural for a chronological read-through
log); stop showing "Load more" when a response's `next_cursor` is null.

**Empty state.** "No events yet" (a freshly-submitted instance may have 0-1 events
depending on poll timing) — valid, not an error.

**Acceptance criteria.** Events render in strict ascending `sequence_num` order with no
gap or duplicate across a "Load more" click, mirroring the server-side guarantee
`TestHandleListEvents_Pagination` already proves.

---

## 4. Workflow List

**Purpose.** "What workflows exist?"

**Layout.** A namespace filter (same convention as Instance List) above a table.

**Table columns:** ID, Version, Namespace, Step Count.

**Row interaction:** click → Workflow Detail for that `(id, version)` pair.

**Pagination.** None — `ListWorkflows` is unbounded by design (a definition-registry
listing, not an execution-history query; the corpus's own pagination-debt analysis
never flagged this route, and definition counts are expected to stay small relative to
instance counts). Revisit only if real deployments show this assumption wrong.

**Empty/loading/error states:** same conventions as Instance List.

---

## 5. Workflow Detail

**Purpose.** "What does this workflow do?"

**Layout.** A header block plus two tables and a list — explicitly not a graph/canvas.

**Header:** Name, ID, Version, Namespace, Description (if present).

**Steps table.** Columns: Step ID, Name, Type (badge), Handler, Retry Policy (present/
absent indicator, expandable for detail), Timeout (if set), Fallback (step ID it falls
back to, if set — rendered as a link scrolling to that row), Initial/Final badges (for
steps matching `initial_step`/`final_steps[]`).

**Transitions table.** Columns: From, To, Condition (raw expression text if present,
"—" if unconditional), Error Transition (badge if `on_error` is true — visually
distinguishes an error-path edge from a normal one, since the engine treats them as the
same struct with a boolean discriminator and the UI must not blur that distinction).

**Triggers list.** One row per trigger: Type, Config (pretty-printed JSON — config
shape is trigger-type-specific and not further modeled in Phase 1).

**Acceptance criteria.** A workflow with a fallback step shows the fallback relationship
distinctly from a normal transition (in the steps table's Fallback column, not
conflated with the Transitions table) — this is the one place a Phase-1 tables-only
design must still respect the engine's real relationship model, even without a graph.

---

## 6. Dashboard / landing aggregate (evaluated, placement decided in the backlog)

**Purpose, if built.** A single-glance operational summary: counts of instances by
status.

**Two implementations were evaluated** (see `GUI_PHASE1_API_MATRIX.md`):
- **Counts variant:** 9 parallel `?status=X&limit=1` requests, reading `.total` from
  each. Authoritative. More requests, more client complexity (aggregating 9 responses
  into one view).
- **Recent-activity variant:** one `?limit=20` request sorted by `updated_at`, no status
  filter, no aggregate math. Cheap, but shows "what changed recently," not "how many of
  each status" — a different (and arguably more actionable) piece of information for an
  operator than a count.

**If shipped, layout for the counts variant:** a row of stat tiles (one per status,
showing the count), each tile clickable → navigates to Instance List pre-filtered to
that status.

**If shipped, layout for the recent-activity variant:** identical to Instance List's
table, just unfiltered and sorted differently — arguably this makes it not a distinct
screen at all, just Instance List's default state (see `GUI_PHASE1_BACKLOG.md` for why
this document's recommendation folds this variant into Instance List rather than
building a seventh screen).

---

*Companion documents: `GUI_PHASE1_PRD.md`, `GUI_PHASE1_API_MATRIX.md`,
`GUI_PHASE1_ARCHITECTURE.md`, `GUI_PHASE1_BACKLOG.md`.*
