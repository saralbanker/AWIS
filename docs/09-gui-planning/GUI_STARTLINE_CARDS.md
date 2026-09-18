# GUI Start-Line Cards

The exact, code-verified cards required before GUI development becomes the primary
engineering effort. This supersedes the 8-card list in `GUI_START_LINE.md` /
`MINIMUM_ENGINE_PROGRAM.md`: independent review against the actual current codebase
found one false dependency and two safe merges, tightening 8 cards to **6**, ≈18-20
engineering-hours (≈2.5 days, essentially unchanged from the prior ≈21h estimate — the
merges and the false-dependency removal are wash against one card, E-G4-5, that turned
out smaller than scoped because its hardest part — the wait-record wrapper — already
exists as working code).

**What changed and why**, each verified against the current tree, not assumed:

1. **`E-G1-3` does not depend on `E-G0-1`.** The two touch entirely disjoint files
   (`internal/storage/sqlite.go` vs. `internal/validate/validate.go` + scaffolds) — the
   claimed dependency was inherited from the old G0→G1 milestone label, not a real code
   relationship. They are independent and parallelizable.
2. **`E-G4-7` (`/healthz`) merges into `E-G4-3`.** It is one row in the same route table,
   same package, zero dependencies of its own — not a separable unit of work.
3. **`E-G4-4` and `E-G4-5` merge.** Both are pure `internal/api` work sharing the same
   router-registration point and response conventions from `E-G4-3` — one coherent PR.
4. **`E-G4-5`'s estimate drops from 4h to ≈2-2.5h.** The wait-record wrapper it
   describes as new work — `signal_name`/`timeout_remaining_s` resolution — already
   exists, tested, at `cmd/awis/status.go:264-348` (`buildActiveJSON`, `waitRecordStore`,
   `lookupWait`). This card ports it into `internal/api`; it does not design it.
5. **`E-G4-6` gets an explicit storage-layer sub-scope.** `ReadEvents`
   (`internal/storage/sqlite.go:163-178`) has no `LIMIT` today; this card must add a
   paginated storage method, not just an API wrapper — the original estimate didn't
   name this second package.

No card was found unnecessary. `E-G4-6` remains required for the GUI-dominant gate as
originally scoped, but can be *sequenced* after the dashboard-live milestone without
blocking it — only the static-timeline view needs it, not the list/detail views. See
`IMPLEMENTATION_ORDER.md`.

---

## The 6 cards

### `E-G0-1` — Commit the B-31 remediation

- **Objective:** commit the already-correct, already-passing 18-file validator
  hardening diff sitting uncommitted in the working tree.
- **Inputs:** current working tree — `go build ./...` and `go vet ./...` both pass with
  the diff applied (verified).
- **Outputs:** one commit.
- **Files:** `internal/validate/validate.go`, `internal/validate/validate_test.go`,
  `cmd/awis/assembly_test.go`, plus 9 fixture/doc files (full list: `git status --short`).
- **Acceptance criteria:** `git status --short` clean; `go build ./...`, `go vet ./...`,
  `go test ./internal/validate/... ./cmd/awis/...` all pass.
- **Validation:** `go build ./... && go vet ./... && go test ./...`
- **Dependency requirements:** none.
- **Parallelization:** fully independent — run any time, blocks nothing else in this set.

### `E-G1-3` — `GetWorkflow` typed sentinel

- **Objective:** replace `GetWorkflow`'s bare not-found error with a package sentinel.
- **Inputs:** `internal/storage/sqlite.go:37-58` (existing sentinel var block, precedent:
  `ErrInstanceNotFound`), `:296` (current bare `fmt.Errorf`).
- **Outputs:** `ErrWorkflowNotFound = errors.New("storage: workflow not found")` added to
  the var block; line 296 rewritten as
  `fmt.Errorf("%w: id=%s version=%s", ErrWorkflowNotFound, id, version)`.
- **Files:** `internal/storage/sqlite.go`; a test addition in `internal/storage/*_test.go`.
- **Acceptance criteria:** `errors.Is(err, storage.ErrWorkflowNotFound)` is true for a
  missing (id, version) pair; existing callers (`internal/engine/engine.go:223`,
  `internal/engine/submit.go:25`, both already `%w`-wrapping) are unaffected.
- **Validation:** `go build ./... && go vet ./... && go test ./internal/storage/... ./internal/engine/...`
- **Dependency requirements:** none (verified — no dependency on `E-G0-1`).
- **Parallelization:** independent of `E-G0-1`, `E-G4-1`, `E-G4-3`. Coordinate the
  `internal/storage/sqlite.go` touch with `E-G4-6` — same file, different line ranges;
  low but real conflict risk. Sequence them or assign one owner to both.

### `E-G4-1` — `cmd/awis-server` skeleton

- **Objective:** new binary embedding the engine, following `cmd/awis start`'s bootstrap
  idiom exactly.
- **Inputs:** `sdk.SQLiteStorage(path)`, `sdk.NewRuntime` (confirmed: `sdk/runtime.go:62-134`
  spawns no goroutines); `cmd/awis/start.go:264-289` as the direct template
  (`signal.Notify` SIGTERM/SIGINT → context cancel; `rt.Start(ctx)` blocks — confirmed
  against `internal/engine/tick.go:18-31`).
- **Outputs:** `cmd/awis-server/main.go` — opens storage, constructs `*sdk.Runtime`, runs
  `rt.Start(ctx)` in a goroutine, runs `http.Server` (bound `127.0.0.1`) on the main
  goroutine, coordinated shutdown on signal.
- **Files (all NEW):** `cmd/awis-server/main.go`, `cmd/awis-server/*_test.go`.
- **Acceptance criteria:** binary builds and starts; SIGTERM/SIGINT triggers clean
  shutdown of both the engine loop and the HTTP listener with no goroutine leak.
- **Validation:** `go build ./... && go vet ./...`; manual — start the binary,
  `curl localhost:PORT/api/v1/healthz` (once `E-G4-3` is wired in), send SIGTERM, confirm
  clean exit.
- **Dependency requirements:** none to start (can serve an empty mux); needs `E-G4-3` to
  be useful.
- **Parallelization:** safe in parallel with `E-G4-3` — disjoint files, join only at a
  thin `main.go` import.

### `E-G4-3` — `internal/api` skeleton + `/healthz` (merged with former `E-G4-7`)

- **Objective:** router, DTO/response-envelope conventions, sentinel-to-status-code
  error-mapping middleware, and the `/healthz` route.
- **Inputs:** `GUI_ARCHITECTURE.md` §6.1's existing sentinel table
  (`ErrInstanceNotFound`, `ErrPluginNotFound`, `ErrCapabilityNotFound`,
  `ErrDuplicateDomainEvent`, `ErrVersionConflict`, `ErrPaginationUnsupported` →
  404/409/501); stdlib `http.ServeMux` (Go 1.26 method+path patterns — no third-party
  router, consistent with this repo's no-framework convention, confirmed via
  `docs/PROVIDERS.md:30` and `cmd/awis/main.go`).
- **Outputs:** `internal/api/router.go`, `internal/api/errors.go` (sentinel-mapping
  middleware), `internal/api/healthz.go` (`GET /api/v1/healthz`).
- **Files (all NEW):** `internal/api/*.go`.
- **Acceptance criteria:** router mounts with zero business routes but `/healthz` returns
  200; error-mapping middleware unit-tested against every known sentinel.
- **Validation:** `go build ./... && go vet ./... && go test ./internal/api/...`;
  `curl -i localhost:PORT/api/v1/healthz` → 200.
- **Dependency requirements:** none.
- **Parallelization:** safe in parallel with `E-G4-1`. Blocks `E-G4-4/5` and `E-G4-6`.

### `E-G4-4/5` — Read-only workflow + instance routes (merged)

- **Objective:** `GET /workflows`, `GET /workflows/{id}/{version}`, `GET /instances`
  (via `Runtime.ListPaged`), `GET /instances/{id}` (via `Runtime.Status` + the
  wait-record wrapper).
- **Inputs:** `E-G4-3`'s router; `E-G1-3`'s `ErrWorkflowNotFound`; **port
  `cmd/awis/status.go:264-348`'s `waitRecordStore`/`lookupWait`/`buildActiveJSON`
  verbatim** — this is the reference implementation for the `signal_name`/
  `timeout_remaining_s` DTO, not new design.
- **Outputs:** `internal/api/workflows.go`, `internal/api/instances.go`.
- **Files (all NEW, plus a router-registration edit):** `internal/api/workflows.go`,
  `internal/api/instances.go`; `internal/api/router.go`.
- **Acceptance criteria:** missing workflow returns 404 via the sentinel, not a string
  match; a waiting instance's response includes non-null `signal_name`/
  `timeout_remaining_s`, matching `cmd/awis status --json`'s value for the same instance.
- **Validation:** `go build ./... && go vet ./... && go test ./internal/api/...`;
  `curl localhost:PORT/api/v1/workflows/{id}/{version}` against a known-missing id → 404
  JSON.
- **Dependency requirements:** `E-G4-3` (router/middleware), `E-G1-3` (sentinel).
- **Parallelization:** parallel with `E-G4-6` once `E-G4-3` lands; coordinate the shared
  router-registration file (low risk — each card adds independent route lines).

### `E-G4-6` — Event-history route

- **Objective:** `GET /instances/{id}/events` with limit+cursor, off a new bounded
  storage read (currently unbounded).
- **Inputs:** `StoragePort.ReadEvents` (`internal/storage/sqlite.go:163-178`, no `LIMIT`
  today); `ListInstancesPaged`/`CountInstances` (`sqlite.go:748`, `:836`) as the direct
  pagination template.
- **Outputs:** a new paginated storage method (`ReadEventsPaged`-style — adds `LIMIT` to
  the existing query, `fromSeq`/`sequence_num` doubling as the cursor);
  `internal/api/events.go` wrapping it.
- **Files:** `internal/storage/sqlite.go` (new method — **this card has a real
  storage-layer sub-scope, budget for it explicitly**); `internal/api/events.go` (NEW).
- **Acceptance criteria:** a single large-event instance returns a bounded page, not the
  full table; the next-page cursor equals the last returned `sequence_num` + 1.
- **Validation:** `go build ./... && go vet ./... && go test ./internal/storage/... ./internal/api/...`;
  `curl localhost:PORT/api/v1/instances/{id}/events?limit=50`.
- **Dependency requirements:** `E-G4-3` (router). Not required for the dashboard-live
  milestone — only for the static-timeline view (`G-G5-5`).
- **Parallelization:** coordinate the `internal/storage/sqlite.go` touch with `E-G1-3`
  (same file, different range).

---

*Companion documents: `IMPLEMENTATION_ORDER.md` (exact sequencing),
`PARALLELIZATION_PLAN.md` (what runs simultaneously), `VALIDATION_PLAN.md` (per-card and
gate-level verification), `EXECUTION_HANDOFF.md` (ready to assign).*
