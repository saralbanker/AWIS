# AWIS — Adversarial Release Audit

**Date:** 2026-09-05
**Branch / HEAD:** `engine-hardening` @ `8a87f70`
**Method:** Every conclusion re-derived from Tier 1 evidence — current source, a clean `git clone` of HEAD, live execution of both binaries, direct HTTP probing of a running server, and measured benchmarks. No prior PASS verdict, verification report, or milestone status was treated as evidence.
**Supersedes for audit purposes:** `FINAL_VERDICT.md` (2026-09-05, PASS WITH RISKS)

---

## 1. Executive summary

The **AWIS engine is in genuinely good condition.** That is a Tier 1 finding, not a courtesy: the race detector is clean across all 20 packages, `golangci-lint` reports 0 issues across both modules, the integration suite passes, and the SQLite concurrency design is one of the more carefully reasoned pieces of code in the repository. The prior audits did real work on engine correctness and their engine-level claims survived re-derivation.

The **release envelope around that engine is not ready**, and the gap was invisible to every prior audit because every prior audit examined the engine and never examined delivery.

The single decisive finding:

> **The entire Beta deliverable — the HTTP API, the server binary, and the whole GUI — is not in version control.** `internal/api`, `cmd/awis-server`, `web/`, and `internal/buildinfo` have **zero tracked files**. They are untracked, not ignored. A clean clone of `HEAD` cannot build `awis-server` at all.

Everything the "Dashboard-live / GUI MVP / GUI Beta achieved" verdicts rest on exists only in one working directory on one machine, has never been reviewed as a diff, has never been built by CI, and would be destroyed by a `git clean -fdx`.

Five prior reports assert a Beta-ready GUI. None of them checked whether the GUI was committed.

**Verdict: CONDITIONAL PASS.** See `FINAL_RELEASE_VERDICT.md`.

---

## 2. What I verified as genuinely working

Reported first, because an adversarial audit that only lists problems is not an accurate audit.

| Claim | Method | Result |
|---|---|---|
| Build is clean | `go build ./...`, `go vet ./...` | Clean, no output |
| Formatting / lint | `make gofmt-check`, `make lint` | `gofmt: clean`; **0 issues** in both modules |
| No data races | `go test -race ./...` | **Clean across all 20 packages** |
| Integration suite | `go test -tags integration ./test/integration/...` | `ok` in 8.2s |
| Happy-path user journey | Ran `init → start → submit → status → trace` by hand with the real binary | **Works.** Instance reached `completed` |
| Healthcheck is real (SEC-12) | Read `healthz.go`, probed live server | Genuinely calls `store.Ping`; not a stub |
| Anthropic model IDs | Read `anthropic.go:33-36` | `claude-haiku-4-5-20251001`, `claude-sonnet-5` — both correct and current |
| No secrets in git history | Scanned all refs for `sk-ant-`, key/env/pem filenames | **Clean.** All hits are placeholders in docs/tests |
| GUI XSS surface | Grepped `web/src` for `innerHTML`/`unsafeHTML`/`eval` | **Zero matches** — lit-html auto-escaping throughout |
| API pagination is bounded | Live probe `?limit=999999` | Correctly clamped to 1000 |
| Frontend build reproduces | `npm run build` | Byte-identical to committed output |
| SQLite concurrency design | Read `db.go:97-120` | `_txlock=immediate` + WAL + `busy_timeout` — correct, and the rationale comment is accurate |

The engine hardening work was real. The findings below are almost entirely **outside** the engine.

---

## 3. Findings

Severity: **P0** = blocks release · **P1** = must fix before public Beta · **P2** = should fix · **P3** = minor.

### RA-01 · P0 · The release candidate is not in version control

**Evidence.**

```
$ git ls-files internal/api    | wc -l   → 0
$ git ls-files cmd/awis-server | wc -l   → 0
$ git ls-files web             | wc -l   → 0
$ git ls-files internal/buildinfo | wc -l → 0

$ git check-ignore -v internal/api/router.go
  (no output — NOT ignored, merely untracked)

$ git clone /mnt/data/rj/AWIS /tmp/clone && cd /tmp/clone
$ ls cmd/
  awis                      # awis-server absent
$ go build ./cmd/awis-server
  stat .../clone/cmd/awis-server: directory not found
```

**Aggravating detail.** `.gitignore` contains a comment asserting the opposite of the truth:

> `# Frontend build tooling (web/) — source is tracked, node_modules and build output are not. cmd/awis-server/static/ IS tracked (it's what go:embed ships), so it is deliberately not listed here.`

Neither `web/` source nor `cmd/awis-server/static/` is tracked. Someone wrote down the intended state and never verified it — and that comment is precisely what would stop a reviewer from checking.

**Impact.** No history, no diff review, no backup, no reproducible build, no CI coverage (see RA-02), and no way for a second engineer to obtain the software. A public Beta cannot be cut from a working directory.

**Fix.** `git add internal/api cmd/awis-server web internal/buildinfo` and commit. Then correct the `.gitignore` comment. Minutes of work — the severity is in the consequence, not the difficulty.

---

### RA-02 · P0 · The API and GUI have never been exercised by CI

**Evidence.** `.github/workflows/ci.yml` runs exactly `make verify` = `gofmt-check vet lint oip-isolation build test race e1`. All Go. There is **no npm step, no `tsc`, no frontend build** anywhere in CI.

Combined with RA-01, CI checks out a tree that does not contain `internal/api`, `cmd/awis-server`, or `web/`. Therefore **every green CI run in this project's history was green on a codebase that did not include the Beta deliverable.**

Two further gaps in the same file:
- `make integration` is **not** part of `verify`, so the integration suite is ungated. (I ran it manually: it passes.)
- `FINAL_VERDICT.md` claims "the integration suite, and the frontend build are all green." Both may be true locally; neither is enforced anywhere.

**Fix.** Land RA-01, then add frontend build + `tsc --noEmit` + `make integration` to `verify`.

---

### RA-03 · P1 · The HTTP API has no authentication, authorization, CORS policy, rate limiting, or body limits

**Evidence.** A grep for `authoriz|authentic|bearer|api[_-]?key|token|cors|access-control|RateLimit|MaxBytes|csrf` across `internal/api/`, `cmd/awis-server/`, and `web/src/` returns **zero matches.** Confirmed live — every endpoint answers unauthenticated:

```
GET /api/v1/instances/{id}/events →
{"events":[{... "payload":{"inputs":{"name":"World"}} ...}]}
```

Full workflow **input and output payloads** — the business data — are served to any client that can reach the port.

**Mitigating.** The default bind is `127.0.0.1:8090`, which is the right default and materially limits exposure.

**Not mitigating.** `--addr` accepts any interface with no auth, no warning, and no documentation saying not to. There is no deployment guide (RA-10) to say otherwise. The first operator who wants the dashboard on a second machine will bind `0.0.0.0` and publish their entire execution history.

**Fix.** Before any non-loopback bind is supported: a shared-secret or token check on `/api/v1/`, an explicit CORS policy, and a startup warning when `--addr` is not loopback.

---

### RA-04 · P1 · No timeouts on the HTTP server

**Evidence.** `cmd/awis-server/main.go` constructs `&http.Server{Addr, Handler}` with no `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, or `IdleTimeout`. Verified empirically against the running server (loopback only):

```
Connection held OPEN for 40s with an incomplete request -> no ReadHeaderTimeout
```

A client sent partial headers and dripped one header line every 5s; the server held the connection indefinitely. With Go's default of unlimited concurrent connections, a trivial client exhausts file descriptors. This is the `gosec` G112 class.

**Fix.** Set all four timeouts. Two lines.

---

### RA-05 · P1 · `awis status` and `awis history` are quadratic and unbounded

The most-used monitoring command degrades quadratically.

**Root cause 1 — an O(n²) sort.** `cmd/awis/status.go:408`:

```go
func sortByUpdatedAt(insts []core.WorkflowInstance) {
	for i := 1; i < len(insts); i++ {
		for j := i; j > 0 && insts[j].UpdatedAt.After(insts[j-1].UpdatedAt); j-- {
			insts[j], insts[j-1] = insts[j-1], insts[j]
		}
	}
}
```

An insertion sort. The comment at the call site explains it was written this way "to avoid importing sort package" — a stdlib package, avoided at the cost of the command's scalability.

Isolated measurement (chronologically-ordered input, the natural and worst case for a descending sort):

| instances | sort time |
|---|---|
| 2,000 | 45 ms |
| 5,000 | 295 ms |
| 10,000 | 1.21 s |
| 20,000 | **4.73 s** |
| 40,000 | **19.08 s** |

Textbook quadratic — 4× per doubling.

**Root cause 2 — unbounded reads.** `ListInstances` (the frozen `core.StoragePort` method, `internal/storage/sqlite.go:701`) has **no LIMIT**. `printStatusTable` calls it **9 times** — once per each of 4 active and 5 terminal statuses — and JSON-decodes `current_steps` and `variables` for every row returned, in order to display **10 rows by default**.

**End-to-end measurement**, real binary, 20,000 seeded instances:

```
awis status   → 4.93 s   (of which 4.73 s is the sort)
awis history  → 5.17 s
awis metrics  → 0.37 s
```

**`metrics` is the control.** It answers a harder question over the same data in 0.37s because it aggregates in SQL. The correct pattern already exists in this codebase; `status` and `history` simply don't use it.

**Extrapolation** (labelled as such): at 100k instances the sort alone is ~2 minutes.

**Fix.** `sort.Slice` (or `slices.SortFunc`), and route `status`/`history` through `ListInstancesPaged`, which already exists.

---

### RA-06 · P2 · The FTS index is fully rebuilt on every `recall` query

**Evidence.** `internal/storage/recall.go:73`: *"Lazy FTS sync: rebuild the FTS index from execution_events before querying."* Measured on 80,000 events:

```
awis recall "World"  (1st) → 3.08 s
awis recall "World"  (2nd) → 6.69 s
```

The second run is **slower**, confirming the rebuild is unconditional and not cached.

**Correction to the prior assessment.** `FINAL_VERDICT.md` deferred this (D-08) as "off the hot path." It is not: `recall` is a documented, user-facing CLI command listed in `awis --help`. A 3–7 second search at a modest 80k events is a user-facing performance defect, not background debt.

---

### RA-07 · P2 · The API silently accepts malformed query parameters

**Evidence.** Live probes, all returning **200 OK**:

| Request | Response |
|---|---|
| `?status=TOTALLY_BOGUS` | `{"instances":[],"total":0}` |
| `?limit=abc` | full result, `limit:100` |
| `?limit=-5` | full result, `limit:100` |
| `?offset=-100` | full result, `offset:0` |
| `events?from=-999` | full event list |

`strconv.Atoi` errors are discarded (`fromSeq, _ := strconv.Atoi(...)` in `events.go`; same pattern in `instances.go`), and `Status` is cast straight from the raw string with no validation against the known set.

**The worst case is the status filter.** An invalid or misspelled status returns an empty list at 200 — **indistinguishable from a correct query that genuinely has no results.** A GUI filter bug or a client typo silently renders as "you have no instances."

**This is the same defect class commit `e75c1f1` claims to have closed** ("Close B-31 silent-defaults validation class"). It was closed in the CLI's diagnostic inputs and left open across the entire API surface.

---

### RA-08 · P2 · Internal error strings are returned to unauthenticated clients

`internal/api/errors.go` — `writeError` encodes `err.Error()` verbatim for every error, including everything falling into the default 500 bucket. Observed:

```
{"error":"sdk: Status: storage: instance not found: does-not-exist"}
```

Benign here, but the same path returns raw storage errors, which for SQLite carry file paths and SQL fragments. Map 5xx to a generic message and log the detail server-side.

---

### RA-09 · P2 · Migrations have no downgrade guard

`internal/storage/migrate.go:82` skips `m.version <= current` and applies everything above it. There is **no check for the reverse case**: when the database's `schema_version` exceeds the newest migration the binary knows, it proceeds silently rather than refusing to start.

An operator who rolls a binary back after an upgrade gets an older engine reading a newer schema with no warning. For an event-sourced system this is a data-integrity risk.

---

### RA-10 · P2 · There is no operator documentation

`README.md` is **one sentence** describing internal module boundaries. It contains no installation step, no quickstart, no example.

```
$ find docs -iname '*deploy*' -o -iname '*ops*' -o -iname '*backup*' \
       -o -iname '*runbook*' -o -iname '*install*' -o -iname '*security*'
  (no matches)
```

No deployment guide, no backup/restore procedure, no upgrade procedure, no monitoring guidance, no security notes. `docs/CLI.md` and the reference docs are good, but nothing tells a new user how to obtain, install, run, or operate AWIS.

**On "can a real external user use it":** the scaffolded onboarding is genuinely good — `awis init` produces a working project and prints correct next steps, which I followed verbatim to a successful run. But a user must first get that far unaided, from a one-sentence README, in a repo whose server binary isn't present.

---

### RA-11 · P3 · `awis status` mixes local time and UTC in one view

`status.go:163` renders the header from `time.Now()` (local); `status.go:194` renders each instance row via `inst.UpdatedAt.Format(...)` (UTC). Observed on an instance created four seconds earlier:

```
AWIS status  2026-09-05 15:07:00        ← local
 ✓  completed  hello-world  ...  2026-09-05 09:36:56   ← UTC
```

The row appears 5.5 hours old. **Display only** — the elapsed/duration arithmetic is correct, because Go's `time.Sub` is location-independent.

---

### RA-12 · P3 · Source maps ship in the production binary

`static.go` embeds `static/bundle.js.map` and serves it: `GET /bundle.js.map` → **200, 218 KB**, containing full TypeScript sources and `node_modules` paths. Drop it from the `go:embed` list for production builds.

---

### RA-13 · P3 · `/api/v1/info` discloses the Go toolchain version

`{"version":"0.1.0-dev","go_version":"go1.26.5","uptime_s":9}`, unauthenticated. Minor, but it hands an attacker exact version-targeting information for free.

---

### RA-14 · P3 · The frontend is never typechecked

`web/build.mjs` runs esbuild only, which **strips** TypeScript types without checking them. `typescript` is a declared devDependency that no script ever invokes, and there is no npm step in CI. The GUI's type safety is currently decorative.

---

### RA-15 · P3 · The primary acceptance test is flaky

`TestSystemRehearsalInitStartSubmitTrace` — the only end-to-end verification of the user journey — **failed on my first clean `go test ./...`**:

```
--- FAIL: TestSystemRehearsalInitStartSubmitTrace (21.15s)
    instance ... did not reach a terminal status within 10s (last observed status: "")
```

It then passed 3/3 in isolation (~11s each) and on a second full-suite run. Load-sensitive: its 10s budget is exceeded when the suite's other packages compete for CPU. It runs in `make test`, so it is in CI's path.

The engine is not at fault — the budget is. But a flaky gate on the *only* end-to-end user-journey check erodes exactly the signal a release needs.

---

## 4. Assessment of prior reports

Requested explicitly: whether documentation is misleading.

**Accurate.** The engine-level claims in `FINAL_VERDICT.md` and `VERIFIED_GEMINI_FINDINGS.md` hold up. The Anthropic model IDs are correct, the healthcheck genuinely pings, race is clean, the fixes are real and tested. `VERIFIED_GEMINI_FINDINGS.md`'s self-critical note about the model-ID regression — that an LLM-authored audit and remediation shared a blind spot — is honest and correct.

**Misleading.** `.gitignore`'s comment asserts `web/` and `cmd/awis-server/static/` are tracked. They are not. That single false comment plausibly prevented anyone from noticing RA-01.

**Overstated.** *"`go build`, `go vet`, the full unit suite (28 packages), the integration suite, and the frontend build are all green."* The unit suite is intermittently red (RA-15, which `REGRESSION_REPORT.md` does acknowledge), and neither the integration suite nor the frontend build is enforced by any gate.

**Uninvestigated by every prior report.** I grepped all five for coverage of my findings:

| Area | Prior coverage |
|---|---|
| Version-control state of the deliverable | **none** |
| HTTP auth / CORS / timeouts | **none** |
| `status` / `history` performance | **none** |
| Deployment / install documentation | **none** |
| API parameter validation | **none** |

The pattern is consistent and explains itself: the audits were scoped to engine correctness and were good at it. Nobody audited delivery. "SEC-01 through SEC-12 reviewed" refers to engine-surface security; it never included the HTTP surface.

---

## 5. Answers to the audit questions

**1. Is AWIS genuinely Beta-ready?**
The engine is. The release envelope is not. The blocker is that the deliverable isn't in version control, isn't in CI, and has no authentication.

**2. Can a real external user successfully use it?**
A user handed a working checkout *and told what to run* can: I did it, and it worked. A user starting from `git clone` **cannot** — they get no server binary, no GUI, and a one-sentence README. The onboarding that exists (`awis init`) is genuinely good; it is unreachable.

**3. Are any release blockers still present?**
Yes — RA-01 and RA-02. Both are process/delivery blockers, not engine defects, and RA-01 is minutes of work.

**4. Are any hidden P0/P1 defects likely?**
In the engine, unlikely — race-clean, lint-clean, well-tested, and my adversarial probing found no correctness defect there. In the **API/GUI layer, likely** — it has never been through CI, never been reviewed as a diff, and never been typechecked. RA-03, RA-04, and RA-07 are what an hour of probing surfaced; that layer has had no scrutiny at all.

**5. Real operational maturity level?**
**Low — roughly "developer preview."** Excellent engine engineering; essentially no operational envelope. See `OPERATIONAL_READINESS_REVIEW.md`.

**6. What work remains?**
See the conditions in `FINAL_RELEASE_VERDICT.md`.
