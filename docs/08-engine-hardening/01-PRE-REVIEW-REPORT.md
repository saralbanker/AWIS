# Report 1 — Engine hardening, before the adversarial review

Branch `engine-hardening`, base `7146214`. State captured at commit `4c483dd`
(22 commits), before the three-agent review round documented in
[Report 2](02-POST-REVIEW-REPORT.md).

This is the "what was fixed" record. The root-cause analysis lives in
[ENGINE_HARDENING_PLAN.md](ENGINE_HARDENING_PLAN.md); the freeze verdict in
[ENGINE_FREEZE_REPORT.md](ENGINE_FREEZE_REPORT.md).

---

## 1. Gate state at this point

| Check | Result |
|---|---|
| `make verify` | ALL GATES PASSED (gofmt, vet, golangci-lint, unit, race, oip isolation, E1) |
| `make integration` ×3 | PASS / PASS / PASS — deterministic (7.8s, 8.2s, 8.6s) |
| Integration tests | 21, enumerated by name |
| `go test -race ./...` | clean |
| Secrets in branch history | 0 matches; `api_keys/` ignored, never tracked |

Scope: 98 files, +9,115 / −403.

---

## 2. Live end-to-end evidence

The documented quickstart, run against the binary built from this commit:

```
awis init .  →  awis start  →  awis submit hello-world --input name=World
                                                    → completed
```

Real outputs flow between steps (`greet → {"message":"Hello, c6!"}` →
`log → {"logged":true}`). The signal path reports
`signal: approved, timeout_remaining_s: 259198`, and delivery drives the
instance to `completed`. Eight concurrent submissions all reached `completed`.

---

## 3. The 25 defects closed

### CRITICAL (11)

| ID | Defect |
|---|---|
| B-0 | restart orphaned in-flight instances |
| B-1 | `rebuild-state` destroyed `waiting` status |
| B-2 | a stuck handler froze the whole runtime |
| B-3 | a handler panic killed the process |
| B-4 | terminal-failure routing left the instance `running` forever |
| B-5 | cancel broken by a stale optimistic-concurrency version |
| B-6 | flags written after a positional were silently dropped |
| B-7 | `trace --json` emitted empty output with exit 0 |
| B-9 / B-19 / B-23 / B-24 | the CLI could not run its own documented quickstart |
| B-15 | retries / pending activations were not durable |
| B-30 | concurrent `awis submit` failed with `SQLITE_BUSY` |

### IMPORTANT (12)

B-8 (discarded JSON encode errors), B-10 (config written to one path, read from
another), B-11 (SDK recall surface stubbed), B-16 (FTS query quoting),
B-17 (unbounded instance reads), B-18 (plugin status write races),
B-20 (`plugin remove` was cosmetic), B-21 (the shipped reference plugin could
not be installed), B-22 (no schema-version ceiling), B-25 (`ListWorkflows`
empty namespace matched nothing), B-26 (`export` applied its namespace filter
to only half its output), B-28 (`signal_name` / `timeout_remaining_s` declared
in the JSON schema and never assigned).

### Partial

B-27 (read-model seam) — `Runtime.ListPaged` landed; the GUI blockers remain.

---

## 4. Four root causes produced every CRITICAL defect

**RC-A — the engine kept durable state in process memory.** Six per-instance
maps (sequence, version, retries, pending activations, waits, cancellations)
were lost on restart while the instances they described survived in SQLite. The
engine then acted on a zeroed view of live state. In five of six cases the
correct data was *already in the database*: the fix was to read, not to build.

**RC-C — the join gate re-activated terminally-failed steps.** A terminally
failed step leaves `current_steps` but never enters `Variables`, and the
completed-set derives from `Variables` keys. So the gate saw its upstream
complete and re-nominated it every tick. The claim was already held, the step
was skipped, and the completion check bailed out because the activatable set
was non-empty — the instance sat in `running` with nothing running.

**RC-E — the shipped CLI never assembled a working runtime.** `awis start`
registered zero native handlers, making two of five step types unreachable
through the binary.

**RC-J — deferred transactions cannot upgrade to writers under WAL.** See §5.

---

## 5. The defect that justified the method

```
awis: submit: engine: append WorkflowStarted seq=1:
  storage: AppendEvent insert: database is locked (5) (SQLITE_BUSY)
```

WAL was enabled. `busy_timeout=5000` was set. Both had been set for a long
time, and that is precisely why nobody had looked there.

They do not help. `AppendEvent` opens a transaction, `SELECT MAX(sequence_num)`
to establish its monotonicity invariant, then `INSERT`s on what it read. Under
SQLite's default DEFERRED locking that transaction **begins as a reader**, and
in WAL mode the write upgrade fails with `SQLITE_BUSY` *immediately* —
`busy_timeout` does not apply, because waiting cannot repair a stale read
snapshot. `SetMaxOpenConns(1)` protected nothing: it serialises writers within
one process, and every `awis` invocation is a separate process.

The fix is one DSN option, `_txlock=immediate`.

**No in-process unit test can produce this defect.** It requires separate
processes contending for one file. It sat behind two correct-looking pragmas,
in the hottest write path in the system, in a product whose central claim is an
append-only event log — and it surfaced within minutes of the binary-level
concurrency probe existing.

---

## 6. The recurring pattern

The investigation was told to look for "the correct implementation exists one
file over". It was found four times:

1. **B-16** — a hardened `sanitizeFTSQuery` already existed in
   `apps/oip/internal/index/fts.go`.
2. **RC-A** — five of six engine maps had durable sources already in SQLite.
3. **B-1** — `RebuildState` already snapshotted non-evented columns to preserve
   definition identity; `waiting` and the cancellation flag needed the same
   technique.
4. **B-28** — `wait_records` held the signal name and timeout, and
   `ListWaitRecordsByInstance` already existed to read them.

---

## 7. Integration tier built from nothing

21 tests, build-tagged `integration`, excluded from `make verify`. Each launches
the real binary as a subprocess in a throwaway project.

Covering: lifecycle and the `--json` contract across 10 read-only subcommands;
fallback / `on_error` / plain `WorkflowFailed` routing; retry; handler-panic
containment; step timeout; cancel of waiting, running, and across a restart;
restart resumption; rebuild-state; wait visibility; and concurrency (30
simultaneous submissions, event-sequence contiguity under load, mixed outcomes).

### Three harness faults that had made the suite lie

- `TestMain` sat in `harness.go`, a **non-test file**. Go only honours it in a
  `_test.go` file, so it compiled, never ran, and left the binary path empty.
- `waitForStatus` polled only on *status*, but an instance parked on signal
  `go` and the same instance parked on `done` are both `waiting` — so a poll
  after delivering a signal returned instantly on the pre-signal state.
- Two tests asserted that `linear-native` **fails** with `handler_not_found`,
  encoding defect B-9 as the expected behaviour.

A fourth was caught while writing the stress tests: `t.Fatalf` called from a
spawned goroutine calls `runtime.Goexit` and does **not** fail the test, so a
concurrent step could fail while the suite reported PASS.

---

## 8. Beyond scope, on security

Two hardenings were made that the brief did not ask for:

- `api_keys/apikeys.txt` was in the working tree **untracked and unignored** —
  one `git add -A` from an unrecoverable commit. Verified it never entered
  history; the path is now ignored.
- Config masking was an exact-match allowlist of two key names. It was widened
  to also match credential-shaped substrings.

*(Report 2 records that this second fix was still not sufficient, and what
replaced it.)*

---

## 9. Known-open at this point

None in the approved CRITICAL/IMPORTANT scope. Recorded so the freeze decision
is made against a complete picture:

| Item | Kind |
|---|---|
| No global event cursor | GUI prerequisite |
| No definition → YAML serializer | GUI prerequisite |
| `"default"` overloaded as a namespace wildcard | product decision |
| `awis init` writes `namespace: default` while scaffolding into `examples` | cosmetic |
| `waiting` is still a non-evented status | architectural (EDR-007 §9) |
| Intelligence path never tested against a live provider | unverified, stated |

No `v1.0.0` tag, per the brief.
