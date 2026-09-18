# AWIS Deferred Technical Debt

Items genuinely safe to leave for later, with the concrete reasoning and the specific condition that would change that.

---

## 1. Single SQLite connection architecture — SAFE TO DEFER

**What it is:** `internal/storage/db.go:42` sets `SetMaxOpenConns(1)` — a real single connection, not a pool, with WAL mode (`db.go:109`) and `_txlock=immediate` + `busy_timeout=5000` (`db.go:98,110`) to avoid a documented historical bug (stale-snapshot `BEGIN DEFERRED` failing `SQLITE_BUSY` instantly).

**Why it's safe:** there is no application-level retry logic anywhere in the codebase (`grep` for `SQLITE_BUSY`/retry patterns finds only the one-time DSN config, no ongoing "fighting" the limitation). This is the opposite of a system straining against its own constraint — it's a deliberate, working design for its intended load. `ARCHITECTURAL_DEBT_REGISTER.md` AD-03's own load test shows a clean, textbook single-server queue (throughput plateaus ~3.4k rps as concurrency rises, p50 rises linearly) — exactly what you'd expect and design for in a single-operator tool.

**Escalation trigger (explicit, not vague):** the moment a second concurrent user hits the dashboard at the same time the engine tick is writing, or a long `rebuild-state` operation holds the connection long enough to stall HTTP reads. Neither has been observed in practice yet. Revisit if/when Blocker 2 (multi-user scope) is ever approved.

---

## 2. Six stale `worktree-agent-*` branches

**What it is:** `worktree-agent-a2f68d9fea206bb94` and five siblings, all pointing at the exact same commit as `main`'s current tip (`98350f6672beb87aea9904ac7cd62ac85be9e94b`), same timestamp, 0 commits of unique work.

**Why it's safe:** dead pointers from abandoned agent sessions, no content to lose. `git worktree list` confirms no live worktree is attached to them (aside from an unrelated scratch worktree created during this audit's own verification step, which should also be pruned).

**Action, at the user's convenience, not this review's authority:** `git branch -d worktree-agent-*` after confirming none are referenced elsewhere.

---

## 3. GUI does not render `Step.inputs`/`Step.outputs` JSON Schema

**What it is:** `internal/core/step.go:16-19` sends `inputs`/`outputs` on every step in the workflow-detail API response. `web/src/types.ts:47-65`'s `Step` interface omits both fields, and `workflowDetail.ts:172-204` never renders them.

**Why it's safe:** purely a missing UI affordance. The API already sends correct data; extra JSON fields the client doesn't declare are simply ignored, not mishandled. No correctness risk, just a coverage gap a future GUI iteration can pick up.

---

## 4. `.gitignore` does not name the `awis-server` binary

**What it is:** `.gitignore` has `/awis` (the CLI binary) but no equivalent rule for the 16MB `awis-server` binary sitting in repo root, untracked.

**Why it's safe for now:** it hasn't been committed, and the generic `*.db`/`*.db-wal`/`*.db-shm` rules already cover the adjacent database artifacts. But it is one `git add -A` away from landing in a commit, which is exactly the kind of mistake worth closing off cheaply.

**Action:** add `/awis-server` to `.gitignore` alongside `/awis` — a one-line fix, bundle it with the Blocker 1 remediation commit so the binary doesn't get swept in by accident.

---

## 5. `.claude/agents/awis-core-engineer.md` still configured for Opus

**What it is:** the frozen Model Allocation Policy in `CLAUDE.md` forbids Opus for the remainder of Baseline V1 ("Opus MUST NOT be selected automatically"), but the `awis-core-engineer` subagent definition still names Opus as its model.

**Why it's safe to defer:** this is internal tooling/process configuration, not a product defect — it affects which model executes future agent-driven work in this repo, not anything that ships. Worth a quick fix for policy hygiene, not a release gate.

---

## 6. One flaky system test with a thin timeout margin

**What it is:** `cmd/awis/system_test.go`'s `TestSystemRehearsalInitStartSubmitTrace` failed once during this audit ("did not reach a terminal status within 10s") and passed cleanly in isolation immediately after, in 11.13s total.

**Why it's safe to defer as a release blocker, but worth a follow-up:** confirmed as load-induced flakiness (this audit was running five parallel investigation agents doing their own `go build`/`go test` against the same machine at the time), not a functional regression in the engine. However, an 11.13s solo runtime against a 10s internal timeout is an uncomfortably thin margin for any CI runner under normal contention. Recommend widening the timeout in a routine follow-up; does not block this release.
