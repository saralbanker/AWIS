# GUI Beta — Product Requirements

Goal: a user can install AWIS, run workflows, and understand system state entirely from
the GUI. Still strictly read-only — no workflow editing, creation, or mutation from the
GUI; no visual editor; no AI features; no auth/RBAC/multi-user. The CLI remains the only
way to create or change data; the GUI's job is to make everything the CLI already lets
you *see* fully visible and navigable without touching a terminal.

---

## 1. Users

Two personas now, where MVP had one:

**The operator** (MVP's persona, unchanged). Runs `awis submit`/`awis signal` from the
CLI, opens the GUI to monitor and debug. Beta adds: finding "everything running for
workflow X," jumping straight to a known instance ID, and trusting that the live view
isn't secretly stale.

**The new user** (Beta-only persona). Just ran `awis init`, has zero instances and
possibly zero workflows registered yet, opens the GUI for the first time. Needs to be
told what to do next — the GUI cannot create data for them (still read-only), but it can
stop leaving them looking at a wall of "no results" with no indication of why or what
comes next.

## 2. Features

### F1 — Workflow Detail screen

Pulls forward the design already specified in `GUI_PHASE1_SCREEN_SPEC.md` §5 (steps,
transitions, and triggers as structured tables, not a graph). Reached from Workflow List
(rows become clickable again — MVP deliberately made them non-clickable specifically
because this screen didn't exist yet).

**Acceptance criteria:** every field in `GUI_PHASE1_API_MATRIX.md`'s Workflow Detail row
renders; a step's fallback relationship is visually distinct from the transitions table
(not conflated into it); an `intelligence`-type step's fields (`capability`, `model_hint`,
`context_budget`, `required`) and a `signal`-type step's `wait_signal` fields (signal
name, timeout, timeout action) each render in a type-specific detail cell — confirmed
necessary, not speculative: 2 of the 3 real workflows in this repo
(`with-intelligence.yaml`, `apps/oip/workflows/capture-decision.yaml`) use `intelligence`
steps with no `handler` field at all, which the original Phase 1 screen spec's
Handler-column design never accounted for. No new backend work needed — every field is
already returned by `GET /workflows/{id}/{version}`.

### F2 — Instance filtering by workflow

An operator can filter Instance List to "only instances of workflow X" with correct
pagination and totals — not a client-side filter on one already-fetched page.

**Acceptance criteria:** selecting a workflow filter and paging through results returns
the same instances, in the same total count, as manually cross-referencing
`definition_id` across every page today — but without having to do that manually.

### F3 — Jump to instance by ID

A single input where pasting a full instance ID and pressing Enter navigates directly to
that instance's detail view.

**Acceptance criteria:** pasting a valid ID navigates correctly; pasting an invalid/
unknown ID surfaces the same distinct "this instance does not exist" state Instance
Detail already renders for a 404 — not a silent no-op or a confusing generic error.

### F4 — Live-freshness improvements

No SSE (stays out of Beta entirely — see `GUI_BETA_API_REQUIREMENTS.md`). Two specific,
confirmed-necessary mechanisms, not "possibly": (a) pause polling when the browser tab
is backgrounded (`document.hidden`) and refetch immediately on visibility restore —
confirmed zero handling of this exists anywhere in the current frontend, so a
backgrounded tab polls forever today; (b) a visible "as of Xs ago" freshness indicator
on every screen showing live-ish data — cheap, since the fetch timestamp is already
tracked internally for Instance Detail's countdown and just needs surfacing, and needs
adding to the other polling screens.

**Acceptance criteria:** an operator watching the GUI can always see how stale the
current view is; backgrounding the tab does not silently keep polling forever with no
visible consequence when they return to it.

### F5 — Onboarding / true-empty-state

A genuinely distinct visual component (not the existing generic `emptyView` with
swapped-in text — confirmed today's empty state is one interchangeable component used
identically everywhere, which would not actually satisfy the distinguishability
requirement below if reused as-is), shown specifically when both `/workflows` and
`/instances` return empty, naming the actual next CLI step. Where possible, the example
command uses a real, currently-registered workflow ID rather than a generic placeholder
— confirmed buildable with zero new API calls, since Workflow List's data is already
fetched and available.

**Acceptance criteria:** the onboarding state and the ordinary "no results for this
filter" state are visually and textually distinguishable at a glance, via a genuinely
different component, not just different copy in the same box — an operator who has just
applied an overly narrow filter must not think their AWIS install is empty.

### F6 — Connection-health visibility

A persistent, always-on-screen indicator (not buried per-screen) reflecting whether the
GUI's last request to the API succeeded.

**Acceptance criteria:** stopping `awis-server` while the GUI is open visibly changes
this indicator within one poll interval, without requiring the user to notice a
per-screen error banner first.

### F7 — Row-order stability on Instance List

Fixes the MVP-verification-flagged jitter — confirmed real, not cosmetic: today's
`instanceList.ts` fully re-sorts the entire row array by `updated_at` on every poll
tick, so any row whose timestamp changes swaps visible position on a routine background
refresh. Fix via lit-html's keyed `repeat()` directive (already the templating library
in use, no new dependency) so DOM nodes patch in place by `instance_id` instead of
tearing down and rebuilding, and only actually re-sort on a real filter/page change or a
genuinely new row appearing — not on every 15s poll.

**Acceptance criteria:** watching Instance List across several poll intervals with a mix
of active and terminal instances does not produce visible row-position swaps that aren't
tied to an actual, meaningful state change an operator would want to notice.

### F8 — Event Timeline at scale

Not in the original 7-point Beta framing; surfaced by direct code review as a real,
confirmed gap. Client-side page size is hardcoded to 50 events per "Load more" click
while the backend already supports up to 1000 — a workflow with hundreds or thousands of
events forces dozens of manual clicks against an unbounded, ever-growing DOM list.
Cheapest fix that stays proportionate to Beta's scope: raise the client page size toward
the backend's cap (e.g., 200-500), cutting click count 4-10x; full virtualization/
windowing is not required for Beta and would be disproportionate engineering effort for
the actual event volumes this system produces today.

**Acceptance criteria:** an instance with several hundred events is fully readable within
a small, bounded number of "Load more" clicks, not dozens.

## 3. Non-goals (restated, single source of truth for this document)

No workflow editing, creation, or versioning from the GUI. No canvas/graph rendering,
editable or not (Workflow Detail stays tables-only in Beta too). No submit/signal/cancel
actions. No authentication, authorization, or multi-user support. No AI-assisted
features. No new engine-side migration work (SSE/`state_changes`) — see
`GUI_BETA_API_REQUIREMENTS.md` for why this stays out of Beta specifically.

---

*Companion documents: `GUI_BETA_GAP_ANALYSIS.md`, `GUI_BETA_API_REQUIREMENTS.md`,
`GUI_BETA_WORK_BREAKDOWN.md`, `GUI_BETA_EXECUTION_ORDER.md`,
`GUI_BETA_FINAL_RECOMMENDATION.md`.*
