# GUI Phase 1 — Backlog

Prioritized, split into MVP / Beta / Nice-to-have. MVP is the smallest slice that is
genuinely useful on its own — not a stub. Scope revised once against the self-challenge
in `GUI_PHASE1_ARCHITECTURE.md`; see the note there on the "Frontend challenger"
subagent's failure and the direct supervisor challenge that replaced it.

---

## Backend additions (do these first — nothing else ships without #1)

| # | Addition | Effort | Leverage |
|---|---|---|---|
| 1 | `--static-dir <path>` flag on `cmd/awis-server`, serving from disk when set and from an embedded `embed.FS` otherwise, mounted at `/` alongside the existing `/api/v1/*` routes | ~1-2h | **Blocking.** Nothing in this backlog ships without a way to serve the frontend at all — `GET /` 404s today. Highest-leverage item in this entire document. |
| 2 | (Beta, optional) A `/api/v1/stats`-style aggregate endpoint returning instance counts by status in one query | ~half a day, per the existing corpus's G8 estimate | Removes the 9-parallel-request workaround for a true counts dashboard, if one is ever built. Not needed for MVP — MVP's landing view is Instance List's default sort, not an aggregate. |

Nothing else in this phase requires a backend change. Every other item below is
buildable against the 6 routes that already exist and are already tested.

---

## MVP — smallest useful, ship first

The bar: an operator can stop running `awis status`/`awis list` by hand for their most
common need (checking what's active, drilling into one instance, reading its history)
and can see what workflows exist without opening a YAML file.

| Item | Why MVP |
|---|---|
| Backend addition #1 (`--static-dir`) | Blocking, see above |
| Frontend shell: `apiClient.ts`, hash router, base layout/nav | Foundation everything else needs |
| **Instance List**, as the landing screen, default view unfiltered and sorted by `updated_at` descending | Serves both "what's active" and doubles as the "recent activity" landing variant evaluated in `GUI_PHASE1_SCREEN_SPEC.md` §6 — no separate Dashboard screen needed for MVP, see that document's reasoning |
| Namespace + status filters on Instance List | Cheap (already-supported query params), directly serves the primary use case |
| **Instance Detail**, including the wait-record wrapper (signal name, live countdown) | This is the concrete, user-facing form of the B-a fix this API layer exists to demonstrate — arguably the single most important screen in the whole phase |
| **Event Timeline**, with the 12-event-type formatter table from `GUI_PHASE1_API_MATRIX.md` and cursor pagination | Completes the "why is this instance in the state it's in" use case; the formatter table is zero-cost (already documented in code comments, not new design work) |
| **Workflow List** | Revised into MVP on self-challenge: this screen is one unbounded GET request and a plain table with no pagination, no complex filtering — materially cheaper than any other screen in this document, and its absence would leave a GUI user with no way to discover what workflows exist at all. Cutting it saved almost no engineering time while measurably reducing completeness. |
| Loading / empty / error states per `GUI_PHASE1_SCREEN_SPEC.md`, for every MVP screen | Required by this phase's own acceptance criteria (`GUI_PHASE1_PRD.md` §5) — an empty result must not look like a broken fetch |

**Explicitly not in MVP, deferred to Beta:** Workflow Detail (the one screen with real
design complexity — two tables plus a triggers list), and everything below it in this
document.

## Beta — second slice

| Item | Why Beta, not MVP |
|---|---|
| **Workflow Detail** (steps/transitions/triggers tables) | Real design complexity (see `GUI_PHASE1_SCREEN_SPEC.md` §5) relative to every MVP screen; "what does this workflow do" is answerable today by reading the YAML file directly, so it's valuable but not blocking |
| Namespace filter as a populated dropdown rather than free text (requires client-side collection of distinct namespaces seen across paginated results, or waiting on a dedicated namespaces-list endpoint) | Polish; free-text namespace filtering is fully functional for MVP, just less discoverable |
| Filter/pagination state persisted in the URL hash (so a reload or shared link preserves the current view) | Real usability improvement, not required for the core use cases |
| Backend addition #2 (`/stats` endpoint) + a true counts-variant Dashboard screen, if operator feedback says the recent-activity landing view isn't enough | Speculative until MVP ships and gets used — do not build ahead of a demonstrated need |
| `WorkflowDefinition.Compensation` rendering on Workflow Detail | A failure-path detail, not needed to answer "what does this workflow normally do" |

## Nice-to-have — no forcing function

| Item | Notes |
|---|---|
| Live updates via SSE | Explicitly a different, later phase (Phase 2 in the original corpus), gated on the `state_changes` migration (G2) and founder decision D1 — out of this document's scope entirely, not merely deprioritized |
| A read-only graph/canvas rendering of a workflow's steps and transitions | Explicitly excluded from this phase's instructions ("do not propose canvas editing"); a *read-only, non-editable* graph view is a legitimate future nice-to-have distinct from an editor, but building it now would risk exactly the scope drift this phase was told to avoid — not recommended until Phase 4 planning is actually underway |
| Per-step `inputs`/`outputs` JSON-Schema-aware rendering (vs. generic pretty-printed JSON) on Workflow Detail | Cosmetic improvement over the MVP/Beta generic JSON view |
| Client-side caching of the namespace list across screens | Minor UX polish |
| Dark mode / visual theming | Not a functional gap |
| Export/download of an event timeline as JSON | Convenience feature, no use case in `GUI_PHASE1_PRD.md` requires it |

---

*Companion documents: `GUI_PHASE1_PRD.md`, `GUI_PHASE1_ARCHITECTURE.md`,
`GUI_PHASE1_SCREEN_SPEC.md`, `GUI_PHASE1_API_MATRIX.md`.*
