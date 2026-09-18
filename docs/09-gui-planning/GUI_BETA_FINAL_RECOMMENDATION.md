# GUI Beta — Final Recommendation

Direct, implementation-focused answers. Full detail: `GUI_BETA_GAP_ANALYSIS.md`,
`GUI_BETA_PRD.md`, `GUI_BETA_API_REQUIREMENTS.md`, `GUI_BETA_WORK_BREAKDOWN.md`,
`GUI_BETA_EXECUTION_ORDER.md`.

**Process note:** the proposal behind this program was independently challenged by a
"Beta challenger" subagent per this task's protocol (unlike the prior GUI planning pass,
this challenger ran to completion — no infrastructure failure this time). It confirmed
five of seven original points, found two real content gaps (Workflow Detail's
tables-only design has no rendering path for `intelligence`/`signal`-type steps despite
2 of 3 real workflows using them; Event Timeline has no scale story), and sharpened two
"possibly" items into concrete requirements (tab-visibility-aware polling, keyed
`repeat()` for the row-reorder fix). All of that is folded into the documents above, not
left as open questions.

---

**What should be built next?**

`FE-1`, Workflow Detail — it's the single largest gap (a fully-designed screen that was
never built), has zero backend dependency, and is the long pole on the critical path, so
starting it first maximizes parallel progress on everything else. In parallel: `BE-1`
(the `definition_id` filter) and `BE-2a`/`BE-2b` (version/uptime) — both small, both
zero-dependency, both unblock small frontend cards downstream (`FE-2`, `FE-6b`).

**What should be deferred?**

Everything the API Requirements document confirmed unnecessary: SSE and the
`state_changes` migration (still gated on founder decision D1, genuinely out of scope,
not just deprioritized), a `/stats` aggregate endpoint (Instance List remains the
landing view; the N-parallel-calls technique stays available if a true dashboard is ever
justified by real usage, not built preemptively), a distinct-namespaces endpoint
(confirmed only two real namespaces exist across every workflow in this repo), and
free-text/fuzzy search (the exact-ID-jump plus `definition_id` filter cover the real
need). Full virtualization of Event Timeline is also deferred — the page-size bump
(`FE-8`) is the proportionate fix at Beta's actual scale.

**What is the shortest path to a beta release?**

**~3 days with two engineers, ~5 days with one.** 13 cards, ~36.5 engineering-hours
total, only two real dependency edges in the whole graph
(`BE-1`→`FE-2`, `BE-2a`→`BE-2b`→`FE-6b`) — ten of thirteen cards are fully independent
and can start on day one. See `GUI_BETA_EXECUTION_ORDER.md` for the exact staffing
breakdown and the recommended card order for the `instanceList.ts`-touching cluster
(four cards share that one file; sequence them, don't parallelize them across people).

**What backend additions give the highest leverage?**

Two, both small and additive, both confirmed low-effort by direct code read (not
assumed): the `definition_id` filter on `/instances` (one field on `InstanceFilter`, one
predicate clause shared by all three list/count call sites — `internal/storage/sqlite.go`'s
`instanceFilterPredicate`) and `GET /api/v1/info` for version/uptime (with the version
string first promoted out of `cmd/awis`-only scope into a tiny shared package, so the two
binaries never drift). Combined: ~6.5 hours. Nothing else in the Beta scope needs a
backend change — every other feature is confirmed frontend-only against the API surface
that already exists.

**What is the estimated effort?**

| Track | Hours |
|---|---|
| Backend (`BE-1`, `BE-2a`, `BE-2b`) | ~6.5h |
| Workflow Detail (`FE-1`) | 10-14h |
| Instance List/Detail cluster (`FE-2`, `FE-4a`, `FE-4b`, `FE-7`) | ~9h |
| Remaining independent frontend cards (`FE-3`, `FE-5`, `FE-6a`, `FE-6b`, `FE-8`) | ~9h |
| **Total** | **~35-39h** |

---

*This document and its companions are sufficient to begin implementation immediately —
no further design decisions are required.*
