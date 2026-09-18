# GUI Beta — Work Breakdown

Implementation-ready cards. `BE-*` = backend (Go), `FE-*` = frontend (`web/src/`). Every
card cites the exact file(s) it touches and its real dependencies — not milestone-label
ordering. ~35-39 engineering-hours total; see `GUI_BETA_EXECUTION_ORDER.md` for
sequencing and parallelization.

---

## Backend cards

### `BE-1` — `definition_id` filter on `/instances`

**Objective:** let the API answer "every instance of workflow X" server-side, with
correct pagination.

**Files:** `internal/core/ports.go` (add `DefinitionID string` to `InstanceFilter`),
`internal/storage/sqlite.go` (`instanceFilterPredicate` — one more `AND` clause),
`internal/api/instances.go` (`handleListInstances` — read `?definition_id=`).

**Acceptance criteria:** `GET /api/v1/instances?definition_id=hello-world` returns only
matching instances with a correct `total`; existing `?namespace=`/`?status=` behavior
unchanged (regression: `internal/storage`'s existing `ListInstancesPaged` tests still
pass).

**Effort:** 3h. **Dependencies:** none. **Parallel with:** everything except `FE-2`
(which needs this landed first).

### `BE-2a` — Promote the version string to a shared package

**Objective:** stop `cmd/awis-server` from needing a duplicated, driftable copy of
`cmd/awis`'s version constant.

**Files:** new `internal/buildinfo/buildinfo.go` (`const Version = "0.1.0-dev"`),
`cmd/awis/main.go` (replace local `const version` with `buildinfo.Version`).

**Acceptance criteria:** `awis version` and `awis version --json` output is byte-for-byte
unchanged (regression-test this explicitly — it's the one place this card touches an
already-shipped, tested command).

**Effort:** 1.5h. **Dependencies:** none. **Parallel with:** everything.

### `BE-2b` — `GET /api/v1/info`

**Objective:** expose version/Go-version/uptime for the GUI's connection/version display.

**Files:** `cmd/awis-server/main.go` (record `startedAt := time.Now()` in `run()`),
`internal/api/info.go` (NEW — handler returning
`{"version","go_version","uptime_s"}`), `internal/api/router.go` (register the route).

**Acceptance criteria:** `GET /api/v1/info` returns a version matching `awis version`'s
output and an `uptime_s` that increases monotonically while the process runs.

**Effort:** 2h. **Dependencies:** `BE-2a` (needs `buildinfo.Version` to exist).
**Parallel with:** `BE-1`.

---

## Frontend cards

### `FE-1` — Workflow Detail screen

**Objective:** build the screen `GUI_PHASE1_SCREEN_SPEC.md` §5 designed but MVP never
shipped, corrected for the intelligence/signal-step gap this program's own review found.

**Files:** new `web/src/screens/workflowDetail.ts`; `web/src/main.ts` (register
`/workflows/:id/:version`); `web/src/screens/workflowList.ts` (rows become clickable
again — remove the "no destination" comment block and wire `navigate()` back in, exact
reverse of the change MVP made to avoid a dead link).

**Design, per the corrected spec:** steps table (id/name/type/initial-final badges) plus
a **type-specific detail cell** per row — `intelligence`-type steps render
`capability`/`model_hint`/`context_budget`/`required`; `signal`-type steps render
`wait_signal`'s `signal_name`/`timeout`/`timeout_action`; all other types show
retry/timeout/fallback as before. Transitions table (from/to/condition/error-badge).
Triggers list. No graph/canvas rendering.

**Acceptance criteria:** all three real workflows in this repo
(`hello-world`, `with-signal`, `with-intelligence`) render correctly, specifically
confirming the `intelligence` step in `with-intelligence` shows its capability/budget
fields rather than a blank Handler cell.

**Effort:** 10-14h (the largest card in this program — three sub-tables plus new
type-specific rendering logic, zero existing screen to extend). **Dependencies:** none
(the API route already returns everything needed, confirmed in
`GUI_BETA_API_REQUIREMENTS.md`). **Parallel with:** all other `FE-*` cards.

### `FE-2` — `definition_id` filter UI on Instance List

**Objective:** consume `BE-1`.

**Files:** `web/src/screens/instanceList.ts` (new filter control + query param wiring),
`web/src/apiClient.ts` (`ListInstancesParams` gains `definitionId`).

**Acceptance criteria:** selecting a workflow filters the table with a correct page
count.

**Effort:** 2h. **Dependencies:** `BE-1`. **Parallel with:** all other `FE-*` cards.

### `FE-3` — Jump to instance by ID

**Objective:** a text input that navigates directly to `/instances/{id}` on submit.

**Files:** `web/src/screens/instanceList.ts` (add the input to the filter bar) or a
small shared component if reused elsewhere.

**Acceptance criteria:** a valid ID navigates correctly; an invalid one still navigates
(to preserve one code path) and Instance Detail's existing 404 handling renders the
"does not exist" state — no new error handling needed here, this card is pure
navigation.

**Effort:** 1.5h. **Dependencies:** none. **Parallel with:** all other `FE-*` cards.

### `FE-4a` — Pause polling when the tab is backgrounded

**Objective:** stop wasting requests (and, more importantly, stop lying about
freshness) when nobody is looking.

**Files:** `web/src/screens/instanceList.ts`, `web/src/screens/instanceDetail.ts` (the
two screens with `setInterval` polling) — add a `visibilitychange` listener that clears/
restarts the interval and triggers an immediate refetch on `document.hidden` → visible.
Consider factoring the pattern into a small shared helper in `web/src/ui.ts` if both
call sites end up identical, rather than duplicating it — judgment call for whoever
implements this, not a hard requirement.

**Acceptance criteria:** backgrounding the tab for longer than one poll interval and
returning triggers an immediate refetch rather than waiting for the next scheduled tick.

**Effort:** 2h. **Dependencies:** none. **Parallel with:** all other `FE-*` cards.

### `FE-4b` — "As of Xs ago" freshness indicator

**Objective:** make staleness visible instead of implicit.

**Files:** `web/src/ui.ts` (a small shared `freshnessLabel(fetchedAt: number)` helper,
same relative-time logic `relativeTime()` already has — reuse, don't reimplement);
`web/src/screens/instanceList.ts`, `instanceDetail.ts` (surface it near the table/header;
`instanceDetail.ts` already tracks `fetchedAt` for the countdown, this is mostly wiring).

**Acceptance criteria:** the indicator visibly updates as time passes without requiring
a new fetch (same tick-without-refetch pattern the countdown already uses).

**Effort:** 2h. **Dependencies:** none (pairs naturally with `FE-4a` but does not require
it). **Parallel with:** all other `FE-*` cards.

### `FE-5` — Onboarding / true-empty-state

**Objective:** a genuinely distinct component (not reused `emptyView` text) shown only
when both `/workflows` and `/instances` are empty, with a dynamically-populated CLI
example.

**Files:** new `web/src/onboarding.ts` (or a case added to `ui.ts` if small enough —
implementer's judgment, but it must be visually distinct per the acceptance criteria,
not a text-only variant of the existing component); `web/src/main.ts` or a shared
top-level check (needs both `/workflows` and `/instances` results, which no single
existing screen currently fetches together — this card owns that combined check).

**Acceptance criteria:** distinguishable at a glance from the ordinary "no results for
this filter" state (`GUI_BETA_PRD.md` F5); the example command references a real
registered workflow ID when at least one workflow exists, falls back to a generic
`awis init` pointer only when `/workflows` is also empty.

**Effort:** 3h. **Dependencies:** none. **Parallel with:** all other `FE-*` cards.

### `FE-6a` — Connection-health indicator

**Objective:** one persistent, always-visible signal of API reachability, not
per-screen-only error banners.

**Files:** `web/index.html` / `web/src/main.ts` (a small indicator in the `.topnav`),
polling `GET /api/v1/healthz` independently of any screen's own data-fetching lifecycle
(so it reflects connectivity even while sitting on Workflow Detail, which has no poll
loop of its own).

**Acceptance criteria:** stopping `awis-server` while the GUI is open visibly changes
this indicator within one poll interval.

**Effort:** 2h. **Dependencies:** none. **Parallel with:** all other `FE-*` cards.

### `FE-6b` — Consume `/api/v1/info`

**Objective:** surface version/uptime somewhere visible (nav bar or a footer —
implementer's judgment on exact placement).

**Files:** `web/src/apiClient.ts` (add `getInfo()`), `web/src/main.ts` or wherever
`FE-6a`'s indicator lives (natural place to co-locate it).

**Acceptance criteria:** displayed version matches `awis version`'s CLI output.

**Effort:** 1h. **Dependencies:** `BE-2b`. **Parallel with:** all other `FE-*` cards
(can be built as a stub against a mocked response and wired to the real endpoint once
`BE-2b` lands, per the same fixture-first technique `GUI_START_LINE.md` established for
MVP).

### `FE-7` — Row-order stability on Instance List

**Objective:** fix the confirmed reorder-jitter bug.

**Files:** `web/src/screens/instanceList.ts` — switch the table body to lit-html's
keyed `repeat()` directive (`import {repeat} from "lit-html/directives/repeat.js"`),
keyed by `instance_id`; stop the unconditional full re-sort on every `load()` call
(currently `[...resp.instances].sort(...)` runs every poll) — only sort when the
underlying instance set actually changes shape (new/removed row), not on every routine
refresh of already-displayed rows.

**Acceptance criteria:** watching the table across several poll intervals with a mix of
active/terminal instances produces no visible row-position swap unless a row's presence
in the set genuinely changed.

**Effort:** 3h. **Dependencies:** none. **Parallel with:** all other `FE-*` cards (same
file as `FE-2`/`FE-3`/`FE-4a`/`FE-4b` — see `GUI_BETA_EXECUTION_ORDER.md` for how to
sequence same-file cards without merge contention).

### `FE-8` — Event Timeline page-size bump

**Objective:** cut "Load more" click count for real event volumes.

**Files:** `web/src/screens/eventTimeline.ts` — raise `PAGE_SIZE` from 50 toward the
backend's cap (e.g. 200-500; pick a value and confirm it against
`storage.MaxEventsPageSize` = 1000 so it never silently gets clamped below what was
requested). While in this file: only call `JSON.stringify(ev.payload)` when a row's
`<details>` is actually expanded, not unconditionally on every render — cheap, same
card, confirmed real (if minor) performance issue.

**Acceptance criteria:** an instance with several hundred events is readable within a
small, bounded number of clicks; expensive JSON serialization only happens for rows a
user actually opens.

**Effort:** 1.5h. **Dependencies:** none. **Parallel with:** all other `FE-*` cards.

---

*Companion documents: `GUI_BETA_GAP_ANALYSIS.md`, `GUI_BETA_PRD.md`,
`GUI_BETA_API_REQUIREMENTS.md`, `GUI_BETA_EXECUTION_ORDER.md`,
`GUI_BETA_FINAL_RECOMMENDATION.md`.*
