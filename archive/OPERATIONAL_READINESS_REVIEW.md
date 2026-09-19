# AWIS — Operational Readiness Review

**Date:** 2026-09-05 · **HEAD:** `8a87f70` · **Basis:** Tier 1 (clean clone, live execution, direct probing)

Scope: deployability, recoverability, monitoring, supportability. The question is not "does the code work" but **"can someone who is not its author run this in production and survive an incident."**

**Overall: LOW — developer preview.** Strong engine, essentially no operational envelope.

| Dimension | Rating | One-line basis |
|---|---|---|
| Deployability | **Not ready** | Server binary is absent from a clean clone |
| Recoverability | **Partial** | Excellent crash recovery; no backup/restore/rollback story |
| Monitoring | **Weak** | Real healthcheck; no metrics endpoint, no log guidance |
| Supportability | **Weak** | Excellent diagnostics; no operator documentation |

---

## 1. Deployability — **Not ready**

### 1.1 The artifact cannot be built from the repository

```
$ git clone <repo> fresh && cd fresh
$ go build ./cmd/awis-server
  stat .../fresh/cmd/awis-server: directory not found
```

`cmd/awis-server`, `internal/api`, `web/`, and `internal/buildinfo` have **zero tracked files** (RA-01). A clean clone yields the CLI and nothing else. There is no deployable artifact and no way to produce one.

Everything below is therefore assessed against the *working tree*, which is not a thing anyone can deploy.

### 1.2 No release pipeline

`Makefile` — `release-dry:` prints `NOT-YET (wired at M02/M06/M18)`. There is no versioned artifact, no checksum, no container image, no install path. CI is verification-only and says so honestly in its header comment.

### 1.3 No installation or deployment documentation

`README.md` is one sentence about module boundaries. No install step, no quickstart. A `find` across `docs/` for deploy/install/ops/runbook/backup/security returns **nothing** (RA-10).

### 1.4 What is good

Genuinely strong, and worth preserving:
- **Single static binary**, two dependencies total (`yaml.v3`, `modernc.org/sqlite`), pure-Go SQLite — **no cgo, no system libraries**. This is an excellent deployment story that the missing pipeline squanders.
- **GUI embedded via `go:embed`** — one process, no asset directory to ship.
- **Sane defaults**: loopback bind, DB in the working directory, zero directory-creation logic.
- **Correct graceful shutdown**: SIGTERM/SIGINT cancel a shared context; HTTP drains with a 5s bounded window while the engine loop stops. I read this path closely and it is right, including the non-obvious detail of cancelling the engine when `ListenAndServe` fails to bind.

### 1.5 Configuration gaps

`awis-server` has three flags: `--db`, `--addr`, `--static-dir`. There is **no** way to configure the namespace (hardcoded `defaultNamespace = "default"`), the tick interval (hardcoded 100ms), log level or format, or any timeout. An operator cannot tune the deployment without a recompile.

Note a **cross-binary inconsistency**: the CLI defaults to `./.awis/runtime.db`; the server defaults to `./awis-server.db`. Point the server at a CLI project without knowing this and it silently creates a *new empty database* and reports zero instances — indistinguishable from a working but idle system. I hit this during the audit and had to read the source to understand it.

---

## 2. Recoverability — **Partial**

### 2.1 Crash recovery is genuinely good

The strongest operational property in the system, and it survived adversarial re-derivation:

- Event-sourced: `execution_events` is append-only; no `UPDATE` or `DELETE` is ever issued against it (enforced and tested).
- `rebuild-state` reconstructs the `workflow_instances` projection from the EventLog — a real, exposed operator recovery tool.
- Crash-recovery and cancellation-durability defects were fixed and are covered by dedicated tests (`deliver_crash_test.go`, `crash_test.go`, `cancel_test.go`, `rebuild_test.go`), all passing under `-race`.
- WAL + `_txlock=immediate` + `busy_timeout(5000)` is a correct, well-reasoned configuration, and its rationale comment is accurate.

### 2.2 No backup or restore procedure

Nothing documents how to back up a running AWIS. This is **not** as simple as copying the file: in WAL mode, `runtime.db` without `-wal` and `-shm` is a torn backup that silently loses recent transactions. There is no `awis backup`, no documented `VACUUM INTO`, no guidance. An operator will copy the `.db` file, believe they have a backup, and discover otherwise during an incident.

**This is the single largest recoverability gap.**

### 2.3 Durability stops short of crash-proof commit

`synchronous=NORMAL` (AD-05): consistent across crashes, but recently committed transactions can be lost on power failure. Defensible — but undocumented, in a system whose recovery model rests on the EventLog being authoritative.

### 2.4 No rollback safety

No downgrade guard (AD-06/RA-09): an older binary opening a newer database proceeds silently instead of refusing. Rolling back a bad deploy — the most basic incident response — is currently unsafe.

### 2.5 `prune-events` is dry-run only in V1

Correctly conservative, and the help text is honest. But it means **there is no supported way to reclaim space** from a growing EventLog. See `SCALABILITY_ASSESSMENT.md`.

---

## 3. Monitoring — **Weak**

### 3.1 What exists

- `GET /api/v1/healthz` — a **real** dependency check. It pings the database and returns 503 on failure. I verified both the code and live behaviour; the SEC-12 fix is genuine, not a stub.
- `GET /api/v1/info` — version, Go version, uptime.
- `awis metrics` — aggregate execution statistics, and notably the **best-implemented command in the CLI**: SQL aggregation, 0.37s on 20k instances.
- Structured JSON logging via `slog`.

### 3.2 What is missing

- **No metrics endpoint.** `awis metrics` is CLI-only; nothing is scrapable. No Prometheus endpoint, no counters, no latency histograms. An operator cannot alert on failure rate, queue depth, or step latency.
- **No readiness/liveness split.** One `healthz` serves both. During a long `rebuild-state`, the process is live but not ready, and an orchestrator cannot tell.
- **No request logging on the HTTP surface** — no seam to add it (AD-07). API traffic is invisible.
- **No log configuration.** Level and format are hardcoded; no rotation guidance.

### 3.3 A concrete logging defect

Engine `slog` output is interleaved into ordinary CLI command output. Observed during a plain `awis submit`:

```
{"time":"...","level":"INFO","msg":"emit","instance_id":"fdabc71b-...","event_type":"WorkflowStarted"}
Submitted: hello-world v1.0.0
```

The test suite works around this by using `Output()` (stdout only) rather than `CombinedOutput()` and *documents the workaround in a comment*. A workaround known well enough to be written down in the tests, but never fixed for users.

---

## 4. Supportability — **Weak envelope around excellent internals**

### 4.1 Diagnostics are genuinely strong

Better than most projects at this stage: `trace`, `logs`, `history`, `metrics`, `recall`, `audit`, `export`, `replay` (dry-run), `rebuild-state`. There is a real audit log. Error messages follow a consistent `fail(code, message, context, hint)` shape with actionable hints. `awis init` scaffolds a working project and prints correct next steps — I followed them verbatim to a successful run.

**The debugging story is the product's strength.** It is undermined entirely by the absence of documentation telling anyone it exists.

### 4.2 Diagnostics degrade exactly when needed

`awis status` at 20k instances: **4.93s**. `awis history`: **5.17s**. `awis recall`: **3–7s** at 80k events (RA-05, RA-06).

These are the commands an operator reaches for *during an incident*, when the database is largest and time matters most. Quadratic degradation in the incident-response path is a supportability defect, not merely a performance one.

### 4.3 No support documentation

No troubleshooting guide, no error-code reference, no known-issues list, no upgrade notes, no security contact or disclosure policy. `docs/CLI.md`, `docs/DSL.md`, and the protocol references are good reference material — but reference material assumes a reader who already knows what they're doing.

### 4.4 A real key sits in the working tree

`api_keys/apikeys.txt` (116 bytes) exists untracked. **It is correctly gitignored, and I confirmed no secret was ever committed** — a full history scan for `sk-ant-` and for key/env/pem filenames found only placeholders in docs and tests. The `.gitignore` comment explaining *why* `api_keys/` was added is exemplary.

Still: a live credential in the repository directory is one `git add -f`, one tarball, or one backup script away from exposure. Move it out of the tree.

---

## 5. Priorities

**Before Beta:**
1. Commit the deliverable (RA-01) — everything else is unverifiable until this lands.
2. Document backup/restore, including the WAL caveat.
3. Add the migration downgrade guard.
4. Write an install + quickstart README.
5. Set HTTP server timeouts; add auth before any non-loopback bind.

**Before a second operator exists:**
6. Metrics endpoint and readiness/liveness split.
7. Fix `status`/`history` performance (RA-05) — incident-response path.
8. Separate engine logs from CLI stdout.
9. Make namespace, tick interval, and log level configurable.
10. Reconcile the CLI/server default DB paths, or make the mismatch loud.
