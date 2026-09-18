# Dashboard Gate Status

## How close are we to dashboard-live?

**4 of 6 cards landed and integration-verified. The two remaining cards have zero
outstanding dependencies — both real blockers (the router, the sentinel) are already in
the tree.** This is the last stretch, not a new phase.

| Card | Status |
|---|---|
| `E-G0-1` (commit B-31) | ✅ Landed — commit `e75c1f1` |
| `E-G1-3` (`GetWorkflow` sentinel) | ✅ Landed, uncommitted |
| `E-G4-1` (server skeleton) | ✅ Landed, integrated, uncommitted |
| `E-G4-3` (API router + `/healthz`) | ✅ Landed, integrated, uncommitted |
| `E-G4-4/5` (read-only workflow + instance routes) | ⬜ Not started — unblocked |
| `E-G4-6` (event-history route) | ⬜ Not started — unblocked, and not required for the gate itself |

## What remains

**For the dashboard-live gate specifically (per `VALIDATION_PLAN.md`'s 9-step
gate-level check): only `E-G4-4/5`.** `E-G4-6` gates the static-timeline frontend view
only (`G-G5-5`), not the definitions/instances list-and-detail views that constitute the
gate — it can land after the gate is reached without blocking it, exactly as
`IMPLEMENTATION_ORDER.md` Stage 3 scoped it.

`E-G4-4/5` needs to:
- Implement `GET /workflows`, `GET /workflows/{id}/{version}` (`Runtime`/`StoragePort`
  wrappers), `GET /instances` (`Runtime.ListPaged`), `GET /instances/{id}` (port
  `cmd/awis/status.go:264-348`'s wait-record wrapper verbatim, per `GUI_STARTLINE_CARDS.md`).
- Widen `api.NewRouter()`'s signature to accept the dependencies these handlers need
  (`*sdk.Runtime` and/or `StoragePort`) — flagged in `internal/api/router.go`'s own doc
  comment as expected, not a surprise.
- Register the new routes against the mux `E-G4-3` already exposes.

## Estimated remaining effort

| Card | Estimate | Notes |
|---|---|---|
| `E-G4-4/5` | ~5-7h | Per `GUI_STARTLINE_CARDS.md`; the wait-record wrapper is a verbatim port of already-tested code, not new design, so this is likely toward the low end |
| `E-G4-6` | ~3h | Includes its storage-layer sub-scope (a paginated `ReadEvents` variant); not required for the gate, can run in parallel with or after `E-G4-4/5` |

**To the dashboard-live gate: ~5-7 engineering-hours, one card, zero remaining
dependencies.** If staffed immediately, this is well under one engineering day.

**To all 6 cards fully landed (including `E-G4-6`):** add ~3h, parallelizable with
`E-G4-4/5` if a second engineer is available (per `PARALLELIZATION_PLAN.md` — they share
only the router-registration file, low conflict risk).

## What the gate-level check will look like once `E-G4-4/5` lands

Re-run `VALIDATION_PLAN.md`'s 9-step sequence: `/workflows` matches `awis workflow
list`, a bad workflow/version returns 404 via the sentinel (not 500), `/instances`
matches `awis list --json`, and a waiting instance's `signal_name`/`timeout_remaining_s`
match `awis status --json` exactly. That sequence is not yet runnable — steps 4-7 all
depend on routes `E-G4-4/5` hasn't been implemented yet. Steps 1-3 (build clean, healthz
200) already pass today, on the current uncommitted tree, as verified in
`STARTLINE_INTEGRATION_REPORT.md`.

---

*Companion documents: `STARTLINE_EXECUTION_REPORT.md`, `STARTLINE_INTEGRATION_REPORT.md`,
`STARTLINE_FINAL_VERDICT.md`.*
