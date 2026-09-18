# GUI Beta — Gap Analysis

Current MVP vs. what "install AWIS, run workflows, and understand system state entirely
from the GUI" requires. Every row is grounded in the actual shipped code
(`web/`, `internal/api/`), not the earlier planning corpus's aspirations — per this
task's evidence hierarchy, running behavior overrides documentation.

---

## Screens

| | MVP (shipped) | Beta gap |
|---|---|---|
| Instance List | ✅ Landing, filters (namespace free-text, status dropdown), offset pagination, 15s poll | No way to filter by workflow (`definition_id`); no way to jump directly to a known instance ID; row order visibly shuffles on poll if any `updated_at` changes between polls |
| Instance Detail | ✅ Full state, wait-record wrapper (signal name, live countdown) | None found — this screen is already Beta-quality |
| Event Timeline | ✅ Cursor pagination, 12-event-type summaries, raw JSON expandable | **Confirmed real gap, not just untested:** client `PAGE_SIZE` is hardcoded to 50 (`web/src/screens/eventTimeline.ts`) while the backend already supports up to 1000/page (`MaxEventsPageSize`, `internal/storage/sqlite.go`) — a workflow with hundreds/thousands of events requires dozens of manual "Load more" clicks against a continuously-growing, unbounded DOM list, and `JSON.stringify(payload)` runs on every row on every re-render regardless of whether its `<details>` is expanded |
| Workflow List | ✅ Unbounded list, namespace filter | None — this screen is already Beta-quality |
| **Workflow Detail** | ❌ **Does not exist** | Fully specified already (`GUI_PHASE1_SCREEN_SPEC.md` §5) but never built — the largest screen-level gap. **The existing spec is also incomplete, not just unbuilt:** its Steps table has no column for `intelligence`-type steps' fields (`capability`/`model_hint`/`context_budget`/`required`) or `wait_signal`-type steps' fields — confirmed against real, shipped workflows: `examples/workflows/with-intelligence.yaml` and `apps/oip/workflows/capture-decision.yaml` both use `type: intelligence` steps with **no `handler` field at all**, which the original spec's Handler column assumed every step has. 2 of the 3 real workflows in this repo would render with a blank/meaningless Handler cell under the as-designed spec. |
| Dashboard/landing aggregate | ❌ Does not exist (deliberately folded into Instance List's default sort in MVP) | No cross-status count view; evaluated, not yet decided whether Beta needs it |

## API surface

| Route | MVP | Beta gap |
|---|---|---|
| `GET /healthz` | ✅ `{"status":"ok"}` only | No version/uptime/build info — an operator cannot tell from the GUI whether they're looking at a freshly-restarted server or a long-running one, or which AWIS version is running |
| `GET /workflows`, `/workflows/{id}/{version}` | ✅ | No gap for Workflow List; `/workflows/{id}/{version}` already returns everything Workflow Detail needs (confirmed: full `WorkflowDefinition` including `steps`, `transitions`, `triggers` — see `GUI_PHASE1_API_MATRIX.md`) |
| `GET /instances` | ✅ namespace + status filters | **No `definition_id` filter** — `core.InstanceFilter` (`internal/core/ports.go:200-207`) has exactly two fields, `Namespace` and `Status`. "Show me every instance of workflow X" is not answerable server-side today; a client-side filter on an already-paginated page would silently corrupt `total`/pagination. |
| `GET /instances/{id}` | ✅ | No gap |
| `GET /instances/{id}/events` | ✅ cursor pagination | No gap in the API itself; a Beta-scale UX question exists at the frontend (see Event Timeline row above) |
| — | — | **No distinct-namespaces endpoint.** Not necessarily a gap — evaluated in `GUI_BETA_API_REQUIREMENTS.md` against whether it's actually needed at Beta scale. |
| — | — | **No `/stats`-style aggregate endpoint.** Same — evaluated, not assumed necessary. |

## UX and operational gaps (not screen- or route-specific)

| Gap | Why it matters for Beta specifically |
|---|---|
| No onboarding/first-run guidance | A fresh install shows "no workflows registered" / "no instances match these filters" — technically correct, gives a first-time user zero indication of what to do next. MVP's audience (the person who just built it) doesn't need this; Beta's audience does. |
| No connection-health indicator | If `awis-server` restarts or the network hiccups, each screen independently shows its own error banner — there's no persistent, always-visible signal of "is this even connected right now." |
| No definition-scoped browsing | Related to the `definition_id` filter gap above — an operator's most natural question ("what's happening with my X workflow") has no direct path today. |
| Row-reorder jitter on poll | **Confirmed real bug**, not cosmetic-only: `instanceList.ts` fully re-sorts the row array by `updated_at` on every poll tick (every 15s), so any row whose timestamp changes swaps position — the table visibly reshuffles on a routine background refresh, not just on a meaningful state change. Erodes trust in the live view. |
| No polling pause when tab backgrounded | Confirmed zero `visibilitychange`/`document.hidden` handling anywhere in `web/src` — a backgrounded browser tab polls forever, with no staleness indicator to tell a user how out-of-date what they're looking at might be. |

## What is explicitly NOT a gap (confirmed, not assumed)

- **SSE / true live updates** — the underlying `state_changes` migration (G2) and founder decision (D1) this depends on were never built and remain outside this task's scope. Beta does not require them; see `GUI_BETA_API_REQUIREMENTS.md` for the reasoning.
- **Free-text fuzzy search across all instances** — no backend search infrastructure exists or is proposed; an exact-ID-jump plus the `definition_id` filter cover the realistic operational need at V1 scale.
- **Workflow creation/editing/mutation from the GUI** — explicitly out of scope per this task's brief; the CLI remains the only way to create data.

---

*Companion documents: `GUI_BETA_PRD.md`, `GUI_BETA_API_REQUIREMENTS.md`,
`GUI_BETA_WORK_BREAKDOWN.md`, `GUI_BETA_EXECUTION_ORDER.md`,
`GUI_BETA_FINAL_RECOMMENDATION.md`.*
