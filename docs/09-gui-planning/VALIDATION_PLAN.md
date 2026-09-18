# Validation Plan

How each card is verified, plus one gate-level validation proving the dashboard-live
milestone is actually true, not just that each card's own tests pass in isolation.

## Per-card validation

| Card | Automated | Manual/integration |
|---|---|---|
| `E-G0-1` | `go build ./... && go vet ./... && go test ./...` | `git status --short` clean |
| `E-G1-3` | `go build ./... && go vet ./... && go test ./internal/storage/... ./internal/engine/...` | A test asserting `errors.Is(err, storage.ErrWorkflowNotFound)` for a missing (id, version) pair |
| `E-G4-1` | `go build ./... && go vet ./...` | Start the binary; `curl localhost:PORT/api/v1/healthz` once `E-G4-3` is wired in; send SIGTERM; confirm clean exit, no goroutine leak (`go build -race` run of the binary under a short load, or a `runtime.NumGoroutine()` check before/after shutdown) |
| `E-G4-3` | `go build ./... && go vet ./... && go test ./internal/api/...` | `curl -i localhost:PORT/api/v1/healthz` → 200; unit test the error-mapping middleware against each of the six known sentinels (`ErrInstanceNotFound`, `ErrPluginNotFound`, `ErrCapabilityNotFound`, `ErrDuplicateDomainEvent`, `ErrVersionConflict`, `ErrPaginationUnsupported`) plus the new `ErrWorkflowNotFound` |
| `E-G4-4/5` | `go build ./... && go vet ./... && go test ./internal/api/...` | `curl localhost:PORT/api/v1/workflows/{missing-id}/{version}` → 404 JSON (not 500); submit a `wait_signal` workflow via the CLI, then `curl localhost:PORT/api/v1/instances/{id}` and confirm `signal_name`/`timeout_remaining_s` match `awis status --json`'s values for the same instance exactly |
| `E-G4-6` | `go build ./... && go vet ./... && go test ./internal/storage/... ./internal/api/...` | Seed an instance with >1 page of events; `curl localhost:PORT/api/v1/instances/{id}/events?limit=50` returns exactly 50, not the full set; the returned cursor equals the last event's `sequence_num` + 1; fetching the next page with that cursor returns the next 50 with no gap or duplicate |

## Gate-level validation — "dashboard live" is actually true

Run this sequence once `E-G0-1`, `E-G1-3`, `E-G4-1`, `E-G4-3`, and `E-G4-4/5` have all
landed (per `IMPLEMENTATION_ORDER.md` Stage 2 — `E-G4-6` is not required for this gate):

1. `go build ./... && go vet ./... && go test ./...` — full tree green.
2. Start `cmd/awis-server` against a real (non-empty) database — submit a few
   `hello-world` and one `with-signal` instance via the existing `awis` CLI first.
3. `curl localhost:PORT/api/v1/healthz` → 200.
4. `curl localhost:PORT/api/v1/workflows` → returns the same registered definitions
   `awis workflow list` shows.
5. `curl localhost:PORT/api/v1/workflows/{id}/{bad-version}` → 404 with a JSON error
   body, not a 500 or an empty 200.
6. `curl localhost:PORT/api/v1/instances` → paged list matching `awis list --json`'s
   instance set.
7. `curl localhost:PORT/api/v1/instances/{waiting-instance-id}` → `signal_name` and
   `timeout_remaining_s` are both non-null and match `awis status --json` for the same
   instance.
8. No `internal/` package boundary was violated — `cmd/awis-server` and `internal/api`
   build without any `internal package ... not allowed` error (this is expected to be a
   non-issue since everything called is already `sdk`-reachable, per the module-boundary
   finding in `GUI_MASTER_PLAN.md` §6.2 — confirm it stayed true, don't just assume it).
9. Restart the server process — confirm no crash, no goroutine leak, and the same
   `curl` sequence above still passes (validates `E-G4-1`'s shutdown/restart path
   actually works, not just that it compiles).

**If all nine pass: the dashboard-live gate is real, not just "the individual cards'
own tests pass."** This is the point at which `G-G5-1` through `G-G5-5` (the frontend
track, already built against a fixture per `GUI_START_LINE.md`) can cut over to the real
API with no rebuild.

## What this validation plan deliberately does not cover

`E-G4-6`'s validation is listed above but is not part of the gate sequence — it has its
own acceptance criteria and can be verified independently whenever it lands, per
`IMPLEMENTATION_ORDER.md` Stage 3. Do not block the gate-level validation above on it.

---

*Companion documents: `GUI_STARTLINE_CARDS.md`, `IMPLEMENTATION_ORDER.md`,
`PARALLELIZATION_PLAN.md`, `EXECUTION_HANDOFF.md`.*
