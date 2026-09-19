# Haiku Verification Tasks — AWIS System Health Baseline

Companion to `AWIS_SYSTEM_HEALTH_BASELINE.md`. Each task is machine-verifiable (A) or repository-verifiable (B) and was left unconfirmed in §6 of the baseline. Run each independently; do not assume the cited source document is correct. Report PASS (claim confirmed as stated), FAIL (claim not confirmed / contradicted), or INCONCLUSIVE (couldn't determine) with the exact command output.

Working directory for all tasks: `/mnt/data/rj/AWIS`, branch `engine-hardening` @ current working tree (do not switch branches, do not commit).

---

### T-01 — B-26: is the `awis export` namespace guard load-bearing?
**Claim:** `TRUTH_CLOSURE.md` §4 asserts the fix for B-26 (commit `41931a1`, "apply one namespace predicate to both halves of `awis export`") is not actually covered by a regression test — reverting it leaves tests green.
**Method:**
1. `git show 41931a1 --stat` to identify the changed file(s) (expect `cmd/awis/export.go` or similar).
2. `git stash` is not needed — instead create a scratch copy: `git worktree add /tmp/b26-check 41931a1~1` (checks out the commit *before* the fix).
3. In `/tmp/b26-check`, run `go test ./cmd/awis/...` and `go test -tags integration ./test/integration/...`.
**Expected if claim TRUE:** both test runs pass (green) even without the fix, proving no test catches the regression.
**Expected if claim FALSE:** at least one test fails without the fix, proving it IS covered.
**Cleanup:** `git worktree remove /tmp/b26-check --force`.

---

### T-02 — BLOCKER 7: does resolving the 4-file merge conflict toward `main` fail to compile?
**Claim:** `RELEASE_CANDIDATE_AUDIT.md` / `TRUTH_CLOSURE.md` — a merge of `engine-hardening` into/from `main` conflicts in exactly `ci.yml`, `Makefile`, `internal/plugin/transport.go`, `subprocess_test.go`, and resolving `transport.go` toward `main`'s version does not compile (merged body calls a `dir` parameter that `main`'s function signature lacks).
**Method:**
1. `git worktree add /tmp/merge-check engine-hardening`
2. `cd /tmp/merge-check && git merge main --no-commit --no-ff` (do not push, do not commit)
3. `git diff --name-only --diff-filter=U` — list actual conflicting files, compare to the claimed 4.
4. For `internal/plugin/transport.go`, manually resolve toward `main`'s side (`git checkout --theirs internal/plugin/transport.go`) and run `go build ./...`.
**Expected if claim TRUE:** conflict file list matches (or is a superset/subset — note any difference), and `go build ./...` fails after resolving `transport.go` toward `main`.
**Cleanup:** `git merge --abort` if still mid-merge, then `git worktree remove /tmp/merge-check --force`.

---

### T-03 — RA-05/C9: is `awis status`/`awis history` an O(n²) sort, and how slow at scale?
**Claim:** `FINAL_RELEASE_VERDICT.md` C9 / `RELEASE_AUDIT_REPORT.md` RA-05 — an insertion sort in the CLI status/history path, ~4.93s at 20k instances, ~5.17s (history) and ~19s (40k) — degrades in the incident-response path.
**Method:**
1. `grep -rn "insertion\|for.*for.*swap\|bubble" cmd/awis/status.go cmd/awis/history.go internal/core/*.go` — locate the actual sort implementation (the baseline's own grep for `sort\.` found nothing in `status.go`/`history.go` directly, so the sort may live in a shared helper — search `internal/` broadly: `grep -rn "func.*[Ss]ort" internal/ cmd/`).
3. Once located, read the function and determine its time complexity (nested loop = O(n²); `sort.Slice`/`sort.Sort` = O(n log n)).
4. If a seed/benchmark script exists (check `internal/fixtures/`, `test/integration/`, or any `*_bench_test.go`), run it at increasing instance counts and record wall-clock time for `awis status` / `awis history`.
**Expected if claim TRUE:** nested-loop sort found; wall time scales roughly quadratically with instance count.
**Report:** exact file:line of the sort, its Big-O, and measured times if a benchmark harness exists (or state that scale-testing infrastructure was not found).

---

### T-04 — C8: does the storage layer refuse to open a database with a newer schema/migration version than the binary supports?
**Claim:** `FINAL_RELEASE_VERDICT.md` C8 — "Add the migration downgrade guard. Refuse to open a database whose `schema_version` exceeds the binary's newest migration." Distinguish this from the already-fixed B-22 (per-event `SchemaVersion` ceiling in `AppendEvent`/`ReadEvents`, confirmed present at `internal/storage/sqlite.go` lines ~96-103, ~279-284).
**Method:**
1. `grep -rn "user_version\|PRAGMA\|migrationVersion\|len(migrations)\|migrations\[len" internal/storage/*.go` — look for a DB-file-level (not per-event) version check performed when opening/initializing the store.
2. Read `internal/storage/migrate.go` or equivalent (find via `find internal/storage -iname "*migrat*"`) — does the `Open`/`New`/`Init` path compare the database's current migration state against the binary's known migration list and refuse to proceed if the DB is ahead?
3. If a test DB fixture exists, try manually setting a bogus/future migration marker (however the code tracks it) and attempt to open it with the current binary; observe whether it errors or proceeds silently.
**Expected if claim TRUE (guard is missing):** no code path refuses to open a DB whose migration state is newer than the binary knows about; opening such a DB either errors unrelatedly, succeeds silently, or panics.
**Report:** whether this is the same mechanism as B-22 (event-level) or a genuinely separate, still-missing DB-level guard.

---

### T-05 — C10/RA-07: does the API silently accept a malformed `?status=` query parameter?
**Claim:** `TRUTH_CLOSURE.md` §4 (narrower framing, after correcting RA-07) — `?status=TOTALLY_BOGUS` returns `200 {"instances":[],"total":0}` instead of a 400, indistinguishable from a genuinely empty result.
**Method:**
1. `grep -n "status" internal/api/*.go` — find the handler that parses the `status` query parameter.
2. Read how it validates the value (does it check against a known enum, e.g. `core.InstanceStatus*`, or pass through unchecked?).
3. Build and run the server: `go build -o /tmp/awis-server-check ./cmd/awis-server && /tmp/awis-server-check -addr 127.0.0.1:18099 &` (use a non-default port), then `curl -s 'http://127.0.0.1:18099/api/v1/instances?status=TOTALLY_BOGUS'`.
4. Kill the background server afterward.
**Expected if claim TRUE:** HTTP 200 with an empty/zero result, no 400 or validation error.
**Report:** exact response body and status code.

---

### T-06 — UF-08: reproduce the FTS full-index rebuild cost at scale
**Claim:** `TRUTH_CLOSURE.md` — `recall`/FTS search rebuilds the whole index every query; measured 27ms@1k, 302ms@10k, 1.96s@50k, 3.1s@80k events (range 2.35–3.64s); `prune-events` is dry-run-only, no relief valve.
**Method:**
1. `grep -rn "prune-events\|dry.run\|dryRun" cmd/awis/*.go` — confirm `prune-events` command exists and check whether it has a non-dry-run mode.
2. `grep -rn "FTS\|fts5\|REBUILD\|rebuild.*index" internal/storage/*.go` — locate the search/recall index rebuild code path; confirm it re-indexes the whole table per query rather than incrementally.
3. If a seed script or fixture generator exists (`internal/fixtures/`), seed a database with ~10,000 and ~50,000 events and time `awis recall <query>` (or the equivalent CLI/SDK call) at each size.
**Expected if claim TRUE:** rebuild-per-query code path found; timing roughly linear in total event count; `prune-events` has no real (non-dry-run) execution path.
**Report:** file:line of the rebuild-per-query code, and measured timings if feasible; note if scale-seeding infrastructure isn't available and timings can't be reproduced.

---

### T-07 — N-1: signal delivered-then-discarded race around `CompleteStep`
**Claim:** `TRUTH_CLOSURE.md` §4 (HIGH, singly-sourced) — a signal step sitting between the delivery-transaction commit and the scanner's out-of-transaction `CompleteStep` can be misclassified as a crashed worker; the signal is recorded delivered then discarded, orphaning the wait record.
**Method:**
1. `grep -rn "CompleteStep" internal/engine/*.go internal/storage/*.go` — locate the scanner call site and confirm whether it runs inside or outside a transaction relative to signal-delivery commit.
2. Read the surrounding hydrate/recovery logic (`internal/engine/hydrate.go`) to understand how it classifies a step as "crashed" vs "in-flight."
3. If feasible, write or locate an existing test that simulates a signal delivered concurrently with a crash-recovery scan, and check whether the wait record survives.
**Expected if claim TRUE:** the code path described exists and has no guard against the race window.
**Report:** file:line evidence either way; if reproducing requires a new test, state that clearly rather than fabricating a result.

---

### T-08 — N-3: do removed plugins remain dispatchable by name?
**Claim:** `TRUTH_CLOSURE.md` §4 — `manager.go:669` — a plugin removed via `plugin remove` can still be dispatched by name.
**Method:**
1. Read `internal/plugin/manager.go` around line 669 (line numbers may have shifted — search for the dispatch-by-name function near any `remove`/`Remove` logic).
2. Confirm whether the removal path actually deletes/deregisters the plugin from whatever map or registry the dispatch function reads from, or only marks it removed in a separate status table.
3. If a plugin test fixture exists, write/run a quick test: register a plugin, remove it, then attempt to dispatch a step to it by name; check whether it succeeds when it should fail.
**Expected if claim TRUE:** dispatch-by-name reads from a registry that removal does not actually clear.
**Report:** file:line evidence; test result if reproduced.

---

### T-09 — N-4: does `hydrate.go:197` mark an instance hydrated before a fallible loop?
**Claim:** `TRUTH_CLOSURE.md` §4 — `hydrate.go:197` marks an instance hydrated before a loop that can error, which can restore the original "D-01 wedge" on a transient storage error.
**Method:**
1. Read `internal/engine/hydrate.go` around line 197 (line numbers may have shifted — search for where "hydrated" status/flag is set relative to any loop that calls storage and can return an error).
2. Determine the ordering: is the hydrated-marker write before or after the loop completes successfully?
**Expected if claim TRUE:** the hydrated marker is set before the loop, so a mid-loop storage error leaves the instance marked hydrated but incompletely recovered.
**Report:** file:line and the exact ordering found; whether an error from the loop is handled/rolled back or not.

---

### T-10 — N-5, N-6, N-7 (as filed, no detail given in prior extraction)
**Claim:** `TRUTH_CLOSURE.md` §4 lists these as "as filed" without full detail in the summary this baseline was built from.
**Method:** Read `TRUTH_CLOSURE.md` §4 directly (`/mnt/data/rj/AWIS/TRUTH_CLOSURE.md`, the "Singly-sourced — reproduce before actioning" subsection) to get the N-5/N-6/N-7 claim text verbatim, then investigate each against current code the same way as T-07/T-08/T-09.
**Report:** claim text, file:line evidence, and whether current code still exhibits the described behavior.

---

### T-11 — C4 fix availability check: do server timeout fields exist anywhere unused/commented out?
**Claim:** baseline V-11 confirms `cmd/awis-server/main.go`'s `http.Server{}` has no timeout fields set. This task checks whether a fix was attempted-and-reverted, or never attempted.
**Method:** `git log --all -p -- cmd/awis-server/main.go | grep -B3 -A3 "ReadHeaderTimeout\|ReadTimeout\|WriteTimeout\|IdleTimeout"`.
**Expected:** likely no matches anywhere in history (confirms never attempted, not reverted) — report whichever is found.

---

### T-12 — C6/C7: README and backup/restore documentation completeness
**Claim:** `FINAL_RELEASE_VERDICT.md` C6 ("one sentence about module boundaries is not an entry point") and C7 (no documented backup/restore procedure, including the WAL torn-backup caveat).
**Method:**
1. Read `README.md` in full; count substantive sections (install, quickstart, usage) vs. its actual length.
2. `grep -rln "backup\|restore\|WAL\|torn" docs/ README.md CLI.md 2>/dev/null` (or wherever user docs live — check `docs/06-reference/` and root `CLI.md`/similar per the baseline's note that TDS files may be misfiled).
**Report:** whether a install+quickstart path exists and is followable from a fresh clone; whether backup/restore + WAL caveat is documented anywhere in the repo.

---

### T-13 — C11: does the API return internal error strings to unauthenticated clients?
**Claim:** `FINAL_RELEASE_VERDICT.md` C11.
**Method:**
1. `grep -rn "http.Error\|w.Write(\[\]byte(err\|err.Error()" internal/api/*.go` — find where errors are serialized into HTTP responses.
2. Trigger an error path with a live server (e.g., malformed request, nonexistent instance ID) and inspect the response body for a raw Go error string (file paths, internal type names) vs. a sanitized message.
**Report:** exact response body for at least one triggered error; whether it leaks internal detail.

---

### T-14 — C12: is `bundle.js.map` embedded in the production `go:embed`?
**Claim:** `FINAL_RELEASE_VERDICT.md` C12.
**Method:** `grep -rn "go:embed" cmd/awis-server/` then `find cmd/awis-server/static -name "*.map"`. Check whether the embed directive's glob includes `.map` files.
**Report:** yes/no, with the embed directive and matching files listed.

---

### T-15 — C14: is engine `slog` output mixed with CLI stdout?
**Claim:** `FINAL_RELEASE_VERDICT.md` C14 — "the test suite already documents a workaround for this."
**Method:** `grep -rn "slog\." cmd/awis/*.go internal/engine/*.go | grep -i "stdout\|os.Stdout"`, and `grep -rn "workaround" cmd/awis/*_test.go` to find the documented workaround referenced.
**Report:** file:line of the workaround comment and what it works around.

---

### T-16 — `go test -race ./...` for `apps/oip` (root module already confirmed clean)
**Claim:** the root module's race run completed clean in the baseline session (24 packages, no races). `apps/oip` was not separately race-tested.
**Method:** `cd apps/oip && go test -race ./...` — capture full output.
**Report:** PASS/FAIL with any race reports verbatim.

---

### T-17 — `go test -tags integration ./test/integration/...` and `pytest` suites
**Claim:** Multiple documents (`TRUTH_CLOSURE.md` UF-18, `RELEASE_AUDIT_REPORT.md`) state the integration and Python test suites currently pass but are not wired into CI.
**Method:**
1. `go test -tags integration -count=1 -timeout 15m ./test/integration/...`
2. `python3 -m pytest --rootdir=python/awis-step python/awis-step/tests -q && python3 -m pytest --rootdir=python/awis-plugin python/awis-plugin/tests -q && python3 -m pytest --rootdir=plugins/git-context-plugin plugins/git-context-plugin/tests -q`
3. `grep -n "integration\|pytest" .github/workflows/ci.yml` — confirm neither target is invoked by CI.
**Report:** pass/fail for both suites, and confirmation of whether CI wires them in.

---

## Reporting format

For each task, return: `T-NN | PASS/FAIL/INCONCLUSIVE | one-line finding | key evidence (file:line or command output excerpt)`. Do not editorialize about root cause or governance — that synthesis belongs in `AWIS_SYSTEM_HEALTH_BASELINE.md`, not here. If a task's target code has moved or been renamed since this file was written, say so and locate the nearest equivalent rather than reporting INCONCLUSIVE without looking.
