# Repository Health Report: AWIS Beta

**Date:** 2026-09-05

---

## 1. Current status

All seven identified P0/P1/P2 correctness, data-loss, and security defects against the AWIS Beta engine are closed and independently re-verified against today's working tree:

- Crash-recovery deadlock (D-01) — fixed, tested.
- Cancellation durability loss (D-02) — fixed, tested.
- Dashboard ordering inversion (D-03) — fixed, tested.
- Subprocess secret leakage (D-04) — fixed, tested.
- Anthropic model IDs (D-05) — fixed correctly this session, after an incorrect first attempt was caught and reverted.
- Dead-end branch wedging the instance forever (D-06) — newly found and fixed this session.
- Fake healthcheck (D-07) — newly found and fixed this session.

`go build ./...`, `go vet ./...`, `go test -count=1 ./...` (28 packages), `go test -tags integration ./test/integration/...`, and the frontend build (`node web/build.mjs`) are all green. See `REGRESSION_REPORT.md` for full output.

## 2. Technical debt (accepted, not blocking)

Three findings are real, verified, and deliberately **not** fixed in this pass, because fixing them would mean modifying a frozen or load-bearing surface without demonstrated production need — which the evidence-hierarchy protocol this verification followed explicitly warns against:

- **D-08 (FTS full rebuild per search query):** documented V1 tradeoff, not on the engine's hot path. Recommended: trigger-based incremental indexing when search volume grows.
- **D-09 (single SQLite connection, `MaxOpenConns(1)`):** real contention risk between engine ticks and HTTP reads under concurrent load; no load test (in this verification or the original audit) demonstrates it as a present problem at V1's single-tenant scale. Recommended: a separate read connection pool at the first milestone targeting concurrent multi-user load.
- **D-10 (namespace excluded from `workflow_definitions` primary key):** blocks multi-tenant workflow-ID reuse. Correctly out of scope — V1 is single-tenant by design.

Two additional items are accurately described in the source audit but are **not defects**:

- **D-11 (no workflow mutation API / YAML serializer):** the current GUI is deliberately read-only; this is Phase-2 scope.
- **D-12 (acyclic DAG, no loop constructs):** a named, frozen architectural decision (`EDR-010`), not an oversight.

One maintainability item:

- **D-13 (flaky `TestSystemRehearsalInitStartSubmitTrace`):** a 10-second test deadline that sits below the test's own ~11-second steady-state duration, causing intermittent false failures under full-suite parallel load. Recommended: loosen the deadline.

## 3. Risks

| Risk | Severity | Mitigation status |
|---|---|---|
| Anthropic model IDs correctness rests on this verifier's knowledge, not a live API round-trip (`ANTHROPIC_LIVE` is unset in every CI run) | Medium | Recommended: run the existing gated `TestLiveSmoke` once against a real key before shipping any Claude-backed workflow to production. |
| Engine tick / HTTP read connection contention (D-09) under concurrent load | Low at current scale, rising with usage | Deferred with justification; revisit at the first multi-user milestone. |
| FTS search cost grows with event volume (D-08) | Low at current scale, rising with event count | Deferred with justification; revisit if `SearchEvents` usage grows. |
| CI flake on `cmd/awis` system-rehearsal test (D-13) | Low (false-negative risk in CI, not a product risk) | Trivial fix recommended (loosen deadline), not yet applied. |
| Whole GUI/API surface (`internal/api/`, `internal/buildinfo/`, `cmd/awis-server/`, `web/`) remains uncommitted in the working tree, alongside a large, unrelated in-progress `.agent`/`.agents` restructuring | Process risk, not a code-correctness risk | Out of scope for this verification pass; noted for awareness. Recommend committing the verified engine/API/GUI state as a clean baseline before further work compounds the uncommitted diff. |

## 4. What this verification did *not* re-litigate

`PHASE2_BLOCKERS.md` and the Phase-2-scoped portions of `RELEASE_CANDIDATE_AUDIT.md` (BLOCKER-01 through -10, SEC-09/10/11) were read and spot-checked but not independently re-derived line-by-line — they describe a future visual-builder phase, not the current Beta/engine surface this task's evidence hierarchy was scoped to. Their claims that were checked were accurate. `DOCUMENT_DRIFT_REPORT.md`'s reconciliation actions (archiving stale planning docs under `docs/09-gui-planning/`) were not executed — that is Tier-2/3 documentation housekeeping, not required by this task's deliverable list, and archiving other sessions' planning documents without being asked risks destroying context another workstream may still need.

## 5. Readiness assessment

The AWIS Beta engine, storage layer, subprocess runner, and Anthropic adapter are now correct against every Tier-1-verified defect this and the prior verification pass found, with regression tests guarding each fix. The three deferred items (D-08, D-09, D-10) are genuine technical debt but are appropriately scoped to future milestones (scale-up, multi-tenancy) rather than blocking the current single-tenant Beta.

## 6. Recommended next phase

1. Commit this verified state (the RC-1..RC-4 fixes, the SEC-05 correction, and the two new SEC-08/SEC-12 fixes) as an explicit, reviewable baseline — the working tree currently holds this alongside a large, unrelated `.agent`/`.agents` migration; separating them into distinct commits will make the engine-hardening history legible.
2. Run the live Anthropic smoke test once against a real key to close the residual D-05 risk.
3. Loosen the D-13 test deadline as a small, immediate CI-hygiene fix.
4. Treat D-08/D-09/D-10 as backlog items for the next milestone that actually stresses search volume, concurrent load, or multi-tenancy — not before.
