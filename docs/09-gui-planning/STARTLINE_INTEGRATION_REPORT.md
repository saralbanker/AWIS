# Startline Integration Report

`E-G4-1` (server skeleton) and `E-G4-3` (API router) were built in parallel by separate
subagents, deliberately decoupled — `E-G4-1` shipped with a placeholder
`http.NewServeMux()` specifically so it didn't have to wait on `E-G4-3`. This document
covers the supervisor's integration step joining them, plus the resulting dependency
and unblocking status.

## Integration status: DONE

**What was done.** In `cmd/awis-server/main.go`:
1. Replaced the placeholder-mux comment block and the `net/http` throwaway `GET /`
   handler with an import of `github.com/awis/awis/internal/api` and a single-line call
   to `api.NewRouter()`.
2. No other change to `main.go` was needed — exactly as both subagents' reports
   predicted (`E-G4-1`'s report named the precise line; `E-G4-3`'s report named the
   exact signature, `func NewRouter() http.Handler`, with zero required parameters).

**Verification performed by the supervisor, independent of both subagents:**
- `go build ./... && go vet ./...` — clean, full tree.
- `gofmt -l cmd/awis-server internal/api internal/storage` — clean.
- `go test ./cmd/awis-server/... ./internal/api/... ./internal/storage/... ./internal/engine/... -count=1`
  — all `ok`.
- **Full repository test suite**, not just the touched packages:
  `go test ./... -count=1` — every package `ok`, zero failures, zero regressions
  anywhere else in the tree.
- **Real integrated binary, started and driven end-to-end:** built the actual
  `cmd/awis-server` binary, ran it against a temp SQLite database, and confirmed:
  - `GET /api/v1/healthz` → `200 OK`, body `{"status":"ok"}` (the real router now
    answering, not the placeholder mux).
  - `GET /api/v1/nope` → `404` (confirms the router replaced the placeholder, since the
    prior placeholder only had `GET /`).
  - `SIGTERM` → clean exit within 1 second, process fully terminated, no lingering PID.

**This is the first point at which `E-G4-1` and `E-G4-3` have run together as one
binary.** Neither subagent's own validation could prove this — each verified their half
in isolation, correctly, per their scope boundary. The integration step is what turns
two independently-correct halves into a verified whole.

## Dependency status

| Card | Real dependencies | Status |
|---|---|---|
| `E-G0-1` | none | Landed (committed) |
| `E-G1-3` | none | Landed (uncommitted) |
| `E-G4-1` | none to build; needed `E-G4-3` to be *useful* | Landed and integrated |
| `E-G4-3` | none | Landed (uncommitted) |
| `E-G4-4/5` | `E-G4-3` (router), `E-G1-3` (sentinel) | **Now satisfied — both dependencies landed** |
| `E-G4-6` | `E-G4-3` (router) | **Now satisfied** |

## Newly unblocked cards

**`E-G4-4/5` (read-only workflow + instance routes) and `E-G4-6` (event-history route)
are now fully unblocked.** Both of their real dependencies — the router (`E-G4-3`) and,
for `E-G4-4/5` specifically, the `GetWorkflow` sentinel (`E-G1-3`) — are landed and
integration-verified in the same tree. Per `GUI_STARTLINE_CARDS.md`'s own scoping note,
`E-G4-4/5`'s implementation will need to widen `api.NewRouter()`'s signature to accept
`*sdk.Runtime`/storage — this is expected, already flagged in `E-G4-3`'s own doc comment
(`internal/api/router.go`), not a new blocker.

No card outside this batch was touched, and none of `E-G2-x`, `E-G3-x`, or
`E-G4-8/9/10/12/13` changed status — they remain gated on founder decisions D1/D2 or on
later phases, per `DEFERRED_WORK_REGISTER.md`, unaffected by this pass.

---

*Companion documents: `STARTLINE_EXECUTION_REPORT.md` (per-card detail),
`DASHBOARD_GATE_STATUS.md` (what remains to the gate), `STARTLINE_FINAL_VERDICT.md`.*
