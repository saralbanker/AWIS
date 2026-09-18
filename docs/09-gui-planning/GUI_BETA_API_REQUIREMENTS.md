# GUI Beta — API Requirements

Exact backend additions required for Beta. Two, both small and additive. Everything
else Beta needs is already served by the 6 routes that shipped with the MVP — confirmed
by direct code re-read, not assumed.

---

## Required addition 1 — `definition_id` filter on `/instances`

**Why.** "Show me every instance of workflow X" is the most natural operational
question an operator has that today's API cannot answer server-side. `core.InstanceFilter`
(`internal/core/ports.go:200-207`) has exactly two fields — `Namespace`, `Status`. A
client-side filter on one already-paginated page would silently corrupt `total` and
pagination, which is worse than not having the feature.

**Why this is low-effort, not just desirable.** `ListInstances`, `ListInstancesPaged`,
and `CountInstances` all share one predicate builder,
`instanceFilterPredicate` (`internal/storage/sqlite.go`) — adding a field to
`InstanceFilter` and one clause to that single function updates all three call sites at
once. This is exactly the kind of additive, contained change this codebase's own
established pattern favors.

**Exact change:**

1. `internal/core/ports.go` — add one field:
   ```go
   type InstanceFilter struct {
       Namespace    string
       Status       InstanceStatus
       // DefinitionID restricts results to the given workflow definition id.
       // Empty = no predicate.
       DefinitionID string
   }
   ```
2. `internal/storage/sqlite.go`'s `instanceFilterPredicate` — add one `AND definition_id = ?`
   clause when `filter.DefinitionID != ""`, following the exact pattern the existing
   `Namespace`/`Status` clauses already use.
3. `internal/api/instances.go`'s `handleListInstances` — read an optional
   `?definition_id=` query param into `core.InstanceFilter.DefinitionID`, same pattern as
   the existing `?namespace=`/`?status=` handling.

**Not required:** any change to `GetInstance`, `ReadEvents`, or any other route — this
is confined to the list/count/paged trio.

**Acceptance criteria:** `GET /api/v1/instances?definition_id=hello-world` returns only
`hello-world` instances, with a correct `total` reflecting that filter, correctly
paginated.

---

## Required addition 2 — version/uptime visibility

**Why.** `/healthz` (`internal/api/healthz.go`) returns `{"status":"ok"}` only — no way
for the GUI (or an operator) to tell which AWIS version is running or how long the
process has been up, which matters for "understand system state entirely from the GUI"
when diagnosing whether they're looking at a stale, hung, or freshly-restarted server.

**A real hazard to design around, found during investigation:** the version string
already exists, but only as `const version = "0.1.0-dev"`, unexported, inside
`cmd/awis/main.go` — `cmd/awis-server` is a separate binary and cannot see it. Duplicating
the constant in `cmd/awis-server/main.go` would work but creates two places that must be
bumped in lockstep and will eventually drift.

**Exact change:**

1. Promote the version string to a tiny shared package, e.g. `internal/buildinfo`:
   ```go
   package buildinfo

   const Version = "0.1.0-dev"
   ```
2. `cmd/awis/main.go` — replace its local `const version = "0.1.0-dev"` with
   `buildinfo.Version`, confirming `awis version`'s existing output is unchanged
   (regression-test this — it's the one place this touches an already-shipped command).
3. `cmd/awis-server/main.go` — record a start time (`startedAt := time.Now()`) when `run()`
   begins.
4. Extend `internal/api`: either widen `healthzResponse` with two more fields, or add a
   small new `GET /api/v1/info` route — either is fine; a separate `/info` route is
   slightly cleaner (keeps `/healthz` a pure liveness check, matches the "additive, not
   redesigned" pattern this API has followed throughout). Response shape:
   ```json
   {"version": "0.1.0-dev", "go_version": "go1.26.5", "uptime_s": 1234}
   ```
   `go_version` mirrors `awis version`'s existing `runtime.Version()` call
   (`cmd/awis/main.go`) for consistency between the CLI and the GUI's answer to "what am I
   running."

**Acceptance criteria:** `GET /api/v1/info` (or the widened `/healthz`) returns a version
string matching `awis version`'s output and an uptime that increases monotonically while
the server runs.

---

## Explicitly NOT required for Beta — confirmed, not assumed

| Candidate | Why it stays out |
|---|---|
| **SSE / `state_changes` migration (G2)** | Genuinely out of scope: gated on the same unmade founder decision (D1) and unbuilt migration this was always gated on since the original Phase 1 planning. Beta's freshness problem is solved by tab-visibility-aware polling plus a staleness indicator (frontend-only, see `GUI_BETA_PRD.md` F4) — not by finally building the live-eventing layer. |
| **`/api/v1/stats` aggregate endpoint** | Not needed: Beta does not add a distinct counts-based Dashboard screen (Instance List, enhanced with the `definition_id` filter and freshness/connection indicators, continues to serve as the landing view). If a true aggregate dashboard is ever wanted later, the N-parallel-`?status=X&limit=1`-calls technique already documented in `GUI_PHASE1_API_MATRIX.md` remains available with zero backend change — revisit only if that technique proves genuinely insufficient in practice, not preemptively. |
| **Distinct-namespaces list endpoint** | Confirmed low-value at real scale: every real workflow file in this repo (`apps/oip/workflows/*.yaml`, `examples/workflows/*.yaml`, `cmd/awis/scaffold/workflows/*.yaml`) uses exactly two namespaces (`oip`, `examples`). A free-text filter is adequate; a dedicated enumeration endpoint would be speculative work for a problem that doesn't exist yet at this scale. |
| **Free-text/fuzzy search across all instances** | No search infrastructure exists or is proposed elsewhere in this program; disproportionate to a read-only local ops tool. The exact-ID-jump (frontend-only) plus the `definition_id` filter (above) cover the realistic operational need. |
| **Time-range filtering on `/instances`** | A real precedent exists elsewhere (`ReadEventRange`'s `emitted_at BETWEEN` on events), but Instance List's existing newest-first sort already approximates "recent instances" on page 1 without it. Legitimate second-order nice-to-have, not a Beta gap — revisit only if `definition_id` filtering alone proves insufficient in practice. |
| **Event Timeline pagination changes beyond the client page-size bump** | The backend's existing `ReadEventsPaged`/cursor contract already supports everything Beta needs (`GUI_BETA_PRD.md` F8) — the gap is entirely on the frontend's hardcoded page size, not the API. |

---

*Companion documents: `GUI_BETA_GAP_ANALYSIS.md`, `GUI_BETA_PRD.md`,
`GUI_BETA_WORK_BREAKDOWN.md`, `GUI_BETA_EXECUTION_ORDER.md`,
`GUI_BETA_FINAL_RECOMMENDATION.md`.*
