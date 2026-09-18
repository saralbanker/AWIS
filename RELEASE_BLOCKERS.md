# AWIS Release Blockers

Only items that would make it irresponsible to ship, or irresponsible to ship *silently*. Each has direct evidence. Ordered by severity.

---

## BLOCKER 1 — The GUI and HTTP API do not exist in version control

**Evidence:** `git log --all --oneline -- internal/api` / `-- internal/buildinfo` / `-- cmd/awis-server` / `-- web/src` all return empty. Confirmed a second time via exhaustive scan: `git rev-list --all | xargs -I{} git ls-tree -r {} --name-only | grep -E '^(internal/api|cmd/awis-server|internal/buildinfo|web)/'` returns nothing across every commit on every branch. `cmd/awis-server/static/` (the binary-embedded build output) is untracked too.

**Why it blocks release:** a "release" is, at minimum, a commit or tag someone else can check out and run. There is currently no such commit. Worse, two *tracked* files (`cmd/awis/main.go`, `cmd/awis/start.go`) have working-tree modifications that import the untracked `internal/buildinfo` package — committing those two files alone, without the new package, breaks the build for anyone else who pulls.

**Not new:** this exact fact was already found and documented on 2026-08-30, 2026-09-01, and 2026-09-03 (see `DOCUMENTATION_DIVERGENCE_REPORT.md`), then independently "rediscovered" twice more on 2026-09-05. It has never been acted on — no `git add` has been run against these paths in the entire project history.

**Fix:** review the working tree for anything that shouldn't ship (the `awis-server` binary and `.db*` files sitting in repo root are already gitignored or should be — see Deferred item on `.gitignore`), then commit `internal/api/`, `internal/buildinfo/`, `cmd/awis-server/`, and `web/` together with the pending changes to `cmd/awis/main.go` and `cmd/awis/start.go` as one atomic change. This is an hours-long task, not a rewrite.

---

## BLOCKER 2 — The GUI/API scope was never authorized against the canonical V1 spec

**Evidence:** `AWIS_PRD.md` (single commit, 2026-07-02, never amended since) explicitly scopes the HTTP API as V2-only work and the web dashboard plus authentication as V3-only work. The canonical architecture documents predate the GUI entirely.

**Why it blocks release:** independent of code quality, nothing in the founder-approved corpus says this feature set should exist in a V1 Beta. Shipping it anyway is a scope decision nobody with the authority to make it has actually made — it happened by accumulation, not by decision. This is also the direct root cause of Blocker 3 below.

**Fix:** this needs a founder decision, not an engineering fix — either amend the PRD to pull the API/GUI into V1 scope (with the tradeoffs that implies, starting with Blocker 3), or hold the GUI/API out of this release and ship the CLI/engine alone, which *is* in scope and *is* release-ready on its own terms.

---

## BLOCKER 3 — No authentication on the HTTP API, and no explicit decision about it

**Evidence:** `internal/api/router.go` registers all routes with no auth middleware. This is a direct consequence of Blocker 2 — the PRD defers "Authentication/authorization" to V3, so none was built.

**Why it blocks release:** an unauthenticated HTTP server that exposes workflow/instance data is only acceptable under a narrow, explicit condition (localhost-only, single-operator, no network exposure). Right now that condition is implicit, not documented or enforced anywhere in the deployment path.

**Fix:** either explicitly document and enforce a localhost-only binding for the Beta (cheap, fast), or treat this as part of the Blocker 2 scope conversation if any multi-user/network-exposed use is intended.

---

## BLOCKER 4 — CI has never once built or tested the API/GUI code

**Evidence:** `.github/workflows/ci.yml`'s single `verify` job runs `go build ./...` / `go test ./...` — but since `internal/api` and `cmd/awis-server` aren't committed (Blocker 1), CI has never seen them. `grep -iE 'awis-server|web/' Makefile ci.yml` returns nothing; there is no Node/npm step anywhere in CI.

**Why it blocks release:** whatever confidence exists in this codebase comes from CI having actually run against it. That confidence does not currently extend to roughly half of what this release is supposed to contain.

**Fix:** once Blocker 1 is resolved, add a Go build/test step for `cmd/awis-server`/`internal/api` and a Node build/typecheck step for `web/` to `ci.yml`. Small, mechanical addition.

---

## BLOCKER 5 (architectural, should be escalated, not silently deferred) — Namespace is not part of any primary key

**Evidence:** `internal/storage/migrations/0001_core_execution.sql:18` — `workflow_definitions` primary key is `(id, version)`, namespace excluded. Documented, reproducible failure mode already on record: `PHASE2_BLOCKERS.md` BLOCKER-05 — a workflow named `sync-leads@1.0.0` registered in namespace `marketing` blocks the *same name* from being registered in namespace `sales`, failing with `ErrAlreadyRegistered` (`internal/storage/sqlite.go:50`). No test exercises this collision scenario (`internal/storage/sqlite_test.go:46,77` only test that an empty-namespace query spans all namespaces, never that two non-empty namespaces actually collide or stay isolated).

**Why this belongs here and not in deferred debt:** "namespace primary-key design" has been filed elsewhere as a passive, non-blocking deferral. It isn't passive — it's a real, already-triggerable bug with a filed defect ID and zero regression coverage. It's defensible to ship without fixing it *only* because AWIS is explicitly single-tenant for V1 — but that defensibility rests on an assumption (exactly one namespace in active use) that nothing in the code enforces. Recommend: keep as V1-deferred, but track it as a committed-to fix before any multi-tenant milestone, not as generic technical debt.

---

## BLOCKER 6 (architectural, should be escalated, not silently deferred) — FTS recall rebuilds the full index on every search, unconditionally

**Evidence:** `internal/storage/recall.go:77,128-152` — `SearchEvents` calls `rebuildFTSIndex` every time, which does a full `DELETE` + `INSERT...SELECT` over the *entire* `execution_events` table inside one transaction. `internal/storage/migrations/0006_recall_fts.sql:6-9` documents this as an intentional "lazy sync" design. Measured, not assumed: `ARCHITECTURAL_DEBT_REGISTER.md` AD-04 and `SCALABILITY_ASSESSMENT.md` §2.4 both independently report 3.08s–6.69s per search at 80,000 events, with cost scaling to total log size, not new-data size. The EventLog is append-only and `prune-events` is dry-run-only (`SCALABILITY_ASSESSMENT.md` §2.5), so this cost is monotonically increasing with no relief valve currently shipped.

**Why this belongs here and not in deferred debt:** this is a measured, user-facing performance defect on a documented CLI command (`awis recall`), not a theoretical scaling concern. The fix described in `AD-04` (incremental sync keyed on `sequence_num`) is characterized as low-risk. Recommend fixing before claiming V1 is "done," or explicitly warning V1 users that `recall` degrades with log size.

---

## BLOCKER 7 — `main` and `engine-hardening` will produce a real merge conflict in CI config, with no plan to resolve it

**Evidence:** both branches independently rewrote `.github/workflows/ci.yml` from the same merge-base (`827a0842`). **Correction to an earlier draft of this document, which mischaracterized the divergence as `engine-hardening` adding `oip-isolation`/`e1`/`integration` targets — re-checked directly and that's wrong: `main`'s copy already has `oip-isolation` and the `make e1` Zero-AI gate (`git show main:.github/workflows/ci.yml`, lines matching `oip-isolation`/`make e1`), so those steps are common to both branches, not a point of difference.** The actual, verified divergence, confirmed via `git merge-tree $(git merge-base main engine-hardening) main engine-hardening` (real `<<<<<<<`/`=======`/`>>>>>>>` markers, not just a content diff): `main` retains a Linux+macOS test matrix (`strategy.matrix.os: [ubuntu-latest, macos-latest]`) and a separate `docs-lint` job guarding EEOS module completeness; `engine-hardening` independently simplified the same file to Linux-only and dropped the `docs-lint` job entirely, with its own comment explaining why (`docs-lint` currently fails because M15/M16/M17 EEOS artifacts are incomplete, and shipping it red would train people to ignore it). Both branches edited the same header-comment and `jobs.verify` block, so git cannot auto-resolve it. No EDR or other document names a reconciliation plan or a target release branch.

**Why it blocks release:** whichever branch is intended to actually cut the release needs this resolved deliberately, by a human, before merge — not discovered mid-merge.

**Fix:** one conscious decision — restore the macOS matrix and `docs-lint` job on top of `engine-hardening`'s version (or explicitly decide to drop them and update `main` instead) — then commit the resolution. Under an hour; the risk is doing it silently via `git merge -X ours/theirs` rather than making the choice explicit.

---

## Explicitly NOT blockers (see `DEFERRED_TECHNICAL_DEBT.md` for the full rationale)

- Single SQLite connection architecture
- 6 stale `worktree-agent-*` branches
- The GUI's missing render of `Step.inputs`/`Step.outputs`
- `.gitignore` not covering the `awis-server` binary by name
- `.claude/agents/awis-core-engineer.md` still pinned to Opus
- The one flaky system test (confirmed a timing-margin issue under load, not a functional regression)
