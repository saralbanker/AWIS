# Startline Execution Report

Four cards executed this pass — the four dependency-free cards identified in
`GUI_STARTLINE_CARDS.md`. All were built/committed by the three subagents specified in
the execution protocol; every acceptance criterion below was independently re-verified
by the supervisor (this document), not just taken from subagent self-reports.

---

## `E-G0-1` — Commit the B-31 remediation

**Objective:** commit the already-correct, already-passing 18-file validator hardening
diff that was sitting uncommitted in the working tree.

**Files changed:** exactly the 18 named files — `internal/validate/validate.go`,
`internal/validate/validate_test.go`, `cmd/awis/assembly_test.go`,
`internal/engine/retry.go`, `internal/intelligence/adapters/anthropic/anthropic.go`,
`internal/intelligence/adapters/anthropic/anthropic_test.go`,
`internal/intelligence/adapters/anthropic/retry_test.go`, 4 testdata JSON files under
that same adapter directory, `apps/oip/qg3_test.go`,
`apps/oip/workflows/capture-decision.yaml`,
`cmd/awis/scaffold/workflows/with-signal.yaml`, `examples/workflows/with-signal.yaml`,
`internal/dsl/testdata/capture-decision.yaml`, `docs/DSL.md`, `docs/PROVIDERS.md`.
Staged by explicit path (no `-A`/`.`) — confirmed via the subagent's reported `git add`
invocation and independently re-confirmed by the supervisor's own `git status --short`
after the commit.

**Acceptance criteria status: PASS.**
- `git status --short` no longer lists any of the 18 files as modified (re-verified by
  the supervisor).
- `go build ./...`, `go vet ./...` clean.
- `go test ./internal/validate/... ./cmd/awis/... ./internal/intelligence/... ./internal/engine/...`
  all `ok`.

**Validation evidence:** commit `e75c1f1`, "Close B-31 silent-defaults validation
class," on `engine-hardening`. Test run before commit: validate 0.004s, cmd/awis
34.116s, intelligence 0.004s, anthropic 0.018s, null 0.004s, porttest 0.004s, engine
0.835s — all `ok`.

---

## `E-G1-3` — `GetWorkflow` typed sentinel

**Objective:** replace `GetWorkflow`'s bare not-found error with a package sentinel,
matching this codebase's existing sentinel-error idiom.

**Files changed:** `internal/storage/sqlite.go` (sentinel declaration + rewritten
not-found return), `internal/storage/sqlite_test.go` (new `TestGetWorkflowNotFound`).

**Acceptance criteria status: PASS.**
- `ErrWorkflowNotFound = errors.New("storage: workflow not found")` declared at
  `sqlite.go:60-62`, immediately following the existing `ErrInstanceNotFound` at
  `sqlite.go:56-58` — same idiom, same var block, independently re-confirmed by the
  supervisor via `grep -n "ErrWorkflowNotFound\|ErrInstanceNotFound"`.
- `GetWorkflow`'s not-found path at `sqlite.go:300` now reads
  `fmt.Errorf("%w: id=%s version=%s", ErrWorkflowNotFound, id, version)`.
- `errors.Is(err, storage.ErrWorkflowNotFound)` confirmed true by `TestGetWorkflowNotFound`.
- Existing callers (`internal/engine/engine.go:223`, `internal/engine/submit.go:25`)
  needed no changes — both already `%w`-wrap.

**Validation evidence:** `go test ./internal/storage/... ./internal/engine/...` — both
`ok` (storage 36.017s including the new test; engine 0.819s unmodified-behavior
confirmation). `gofmt -l` and `golangci-lint run ./internal/storage/...` both clean.
Uncommitted, as instructed.

---

## `E-G4-1` — `cmd/awis-server` skeleton

**Objective:** a new binary embedding the engine, following `cmd/awis start`'s
bootstrap idiom.

**Files changed (both NEW):** `cmd/awis-server/main.go`, `cmd/awis-server/main_test.go`.

**Acceptance criteria status: PASS.**
- Binary builds; starts against a real SQLite DB; `signal.Notify` on SIGTERM/SIGINT
  cancels a shared context mirroring `cmd/awis/start.go:264-271` exactly.
- `rt.Start(ctx)` runs in a goroutine, `http.Server` runs on the main goroutine,
  shutdown is coordinated via `srv.Shutdown` on a bounded 5s context.
- One real defect was found and fixed during the subagent's own testing:
  `internal/engine/tick.go`'s `Run` returns `ctx.Err()` (i.e. `context.Canceled`) on
  graceful shutdown by design; the initial implementation surfaced this as a false
  "engine: context canceled" failure on exit code 1. Fixed by filtering
  `errors.Is(engineErr, context.Canceled)` to nil — the same behavior `cmd/awis/start.go`
  already relies on by discarding `rt.Start`'s return value entirely. This is a
  straightforward application of the documented contract, not a new design decision.

**Validation evidence — independently re-run by the supervisor, not just the subagent's
self-report:**
```
go build ./...   → exit 0
go vet ./...      → exit 0
go test ./cmd/awis-server/... -count=1  → ok (0.863s / 1.026s across runs)
```
Manual gate-level smoke test (see `STARTLINE_INTEGRATION_REPORT.md` for the integrated
version): binary started against a temp SQLite DB, `GET /` returned 404 from the
placeholder mux prior to integration; SIGTERM produced a clean exit within 1s, no
lingering process.

---

## `E-G4-3` — `internal/api` skeleton + `/healthz`

**Objective:** router, DTO/response-envelope conventions, sentinel-to-status-code
error-mapping middleware, and `/healthz`.

**Files changed (all NEW):** `internal/api/router.go` (`NewRouter() http.Handler`),
`internal/api/errors.go` (`HandlerFunc`, `wrap()`, `errorEnvelope`, `statusFor()`),
`internal/api/healthz.go`, `internal/api/router_test.go`, `internal/api/errors_test.go`.

**Acceptance criteria status: PASS.**
- `GET /api/v1/healthz` returns 200 JSON `{"status":"ok"}` — re-verified by the
  supervisor against a real running binary post-integration (see
  `STARTLINE_INTEGRATION_REPORT.md`).
- Error-mapping middleware table-tested against every sentinel it maps:
  `storage.ErrInstanceNotFound`, `ErrPluginNotFound`, `ErrCapabilityNotFound`,
  `ErrWorkflowNotFound` (the concurrent `E-G1-3` sentinel, correctly picked up),
  `ErrDuplicateDomainEvent`, `ErrVersionConflict`, `sdk.ErrPaginationUnsupported`, a
  `%w`-wrapped case, and an unmapped error → 500.

**One deviation, reviewed and accepted:** the card's spec said `E-G4-3` could map
`ErrWorkflowNotFound` "if it exists yet" without blocking on it. It appeared in the
shared working tree mid-session (the concurrent `E-G1-3` task), and the subagent added
one `errors.Is` case for it. This is within the card's own stated allowance, not scope
expansion — verified by re-reading the original card spec.

**One correction to the card's guess:** the spec assumed `ErrPaginationUnsupported` lived
in `internal/storage`; the subagent found it in package `sdk`
(`sdk/runtime_readmodel.go`) via grep before writing code, and used the correct import.
Noted as evidence the subagent verified against real code rather than the spec's
assumption.

**Validation evidence:** `go test ./internal/api/... -v` — 13/13 subtests pass.
`go build ./...`, `go vet ./...`, `gofmt -l`, `golangci-lint run ./internal/api/...` —
all clean.

---

*Companion documents: `STARTLINE_INTEGRATION_REPORT.md` (how these four cards were
joined), `DASHBOARD_GATE_STATUS.md`, `STARTLINE_FINAL_VERDICT.md`.*
