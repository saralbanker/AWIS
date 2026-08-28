# AWIS Engine Hardening Plan

Branch: `engine-hardening` (base `7146214`, branched from `m17-full-cli-init`).
Status: living document — updated as defects close.

All findings below are **tier-1 evidence** (executed live behaviour of the shipped
binary or an executed Go probe) or **tier-2 evidence** (exact source reading),
labelled per finding. Nothing here is carried over on the authority of a prior
audit alone.

---

## 1. Shared root causes

The 23 defects are not 23 independent bugs. They collapse into nine root causes,
and four of those account for every CRITICAL defect.

### RC-A — Engine runtime state is in-memory but describes durable objects

`internal/engine/engine.go` keeps six per-instance maps: `seq`, `ver`, `retries`,
`pending`, `waits`, `cancels`. Every one is lost on restart while the instances
they describe survive in SQLite. The engine then acts on a zeroed view of live
state.

| Map | Durable source that already exists | Defect |
|---|---|---|
| `ver` | `workflow_instances.version` | B-0, B-5 |
| `seq` | `MAX(execution_events.sequence_num)` | (latent; had a reactive fallback) |
| `waits` | `wait_records` table | B-15 |
| `retries` | EventLog `StepFailed{retrying:true}` | B-15 |
| `pending` | EventLog `StepFallbackActivated` / `on_error` | B-15 |
| `cancels` | *nothing* — needed new columns | B-5 |

**The correct data was already in the database in five of six cases.** This is the
"the correct implementation exists one file over" pattern the investigation was
told to look for, in its purest form: the fix is to *read* rather than to *build*.

*Evidence (tier 1)* — executed Go probe, two `Engine` instances over one store:

```
after engine1 tick: status=running current=[]
engine2 tick 0 error: engine: upsert projection StepStarted
  (instance=943ae86b-…): storage: version conflict: … expected=2
after engine2 ticks: status=running current=[] — permanently stuck
```

The failure is worse than "the tick errors". `ClaimStep` and `AppendEvent` both
already committed when `UpsertInstance` failed, so the EventLog and the projection
**diverge**, and the now-held `step_claims` row makes every later tick lose the
claim and skip the step forever.

*Evidence (tier 1)* — live CLI, signal delivery across a restart:

```
$ awis signal <id> done          # after restarting the runtime
Signal delivered: done → <id>
Instance resumed; next step: wait-b     ← both lines are false

sqlite> select signal_name, delivered_at from signal_inbox;
go   |2026-08-26T09:03:32Z
done |None                              ← never delivered
sqlite> select step_id, signal_name from wait_records;
wait-b|done                             ← the durable wait was there all along
```

### RC-B — Rebuild discards non-evented projection state

`waiting` is a tick-time decision, not an event (the documented EDR-007 §9 gap), so
`storage.RebuildState` — which replays events only — cannot reconstruct it. It also
hardcodes `cancellation_requested = 0`, silently discarding an operator action.

*Evidence (tier 1)* — `awis rebuild-state --all` on a waiting instance:

```
before:  ('waiting', '["wait-b"]', 9)
after:   ('running', '["wait-b"]', 1)     ← B-1
wait_records row survived intact          ← the fix input was present
```

### RC-C — The join gate re-activates terminally-failed steps

`completedSet` is derived from `Variables` keys. A step that terminally fails is
removed from `current_steps` but never enters `Variables`, so `isActivatable` sees
its upstream complete and re-nominates it every tick. The claim is lost, the step
is skipped, and the completion check bails out because `activatableFor` is
non-empty. The workflow stays `running` forever. (B-4, tier 2.)

### RC-D — Native handlers run without isolation

No default timeout: a step with no `timeout:` gets no deadline, and `wg.Wait()` in
`processInstance` then blocks the tick loop — one stuck handler freezes every
instance (B-2). No `recover()`: a handler panic kills the process and every
in-flight instance with it (B-3). Both tier 2.

### RC-E — The shipped CLI does not assemble a working runtime

`awis start` registers zero native handlers and registers discovered plugins into
the `PluginStore` table only, never into the runtime's plugin manager. Two of five
step types are unreachable through the shipped binary (B-9/B-19). Separately, the
scaffold declares `namespace: examples` while the CLI defaults to `default`, so the
quickstart `awis init` itself prints fails immediately (**B-23, new**).

*Evidence (tier 1)* — the documented out-of-box path, verbatim:

```
$ awis init .
Next steps:
  awis start
  awis submit hello-world --input name=World

$ awis submit hello-world --input name=World
awis: submit: sdk: Submit: no registered workflow with id "hello-world"   ← B-23

$ awis --namespace examples submit hello-world --input name=World
StepFailed {"code":"handler_not_found",
            "message":"no native handler registered for \"examples.hello.greet\""}  ← B-9
WorkflowStarted {"inputs":{}}                                              ← B-6
```

Three critical defects on the first two commands a new user runs.

### RC-F — Discarded JSON encode errors

`_ = enc.Encode(out)` at every `--json` site. Combined with a truncation bug in
`trace.go` that emits `"` + raw-payload-fragment + `"` — invalid JSON whenever the
fragment contains a quote — this produces **empty stdout with exit code 0**
(B-7/B-8).

*Evidence (tier 1)*: `awis --json trace <id>` → zero bytes, `exit=0`.

### RC-G — `flag` stops parsing at the first positional

Stdlib behaviour; every flag written after the positional argument is silently
dropped, including in the command `awis init` prints (B-6, tier 1 above).

### RC-H — Read-model and storage seams

`ListInstances` has no `LIMIT`/offset and is called every 100 ms tick (B-17);
`SearchEvents` wraps the user query in bare quotes so any quote breaks FTS, while a
hardened `sanitizeFTSQuery` already exists in `apps/oip/internal/index/fts.go`
(B-16 — second instance of the "one file over" pattern); `ReplayInstance` returns an
empty struct and `ReplayTrace` *is* an empty struct (B-11); `schema_version` is
written but never range-checked on read (B-22); `awis init` writes `config.yaml` at
the project root while `awis config show` reads `.awis/config.yaml`, and `start`
reads neither (B-10).

*Evidence (tier 1)*: `awis config show` immediately after `awis init` →
`Config file not found: .awis/config.yaml`.

### RC-I — Plugin lifecycle

`SetPluginStatus` is called from fire-and-forget goroutines with the error
discarded, at four sites (B-18). `plugin remove` sets status to `removed` but
leaves `plugin_capabilities` rows intact, so capability lookup still routes to the
removed plugin (B-20). `spawnPlugin` never sets `cmd.Dir`, so the shipped
`git-context-plugin` manifest (`python3 -m git_context_plugin`) cannot resolve its
own package (B-21).

B-21 is the sharpest example of why integration testing had to come first: the
existing e2e test passes only because it **writes its own temp manifest** with an
absolute `PYTHONPATH` injected. It proves the plugin protocol works; it proves
nothing about the plugin that actually ships.

---

## 2. Defect → root cause map (final status)

All CRITICAL and IMPORTANT defects in the approved scope are CLOSED. Each was
closed only after all six verification conditions in §4 held.

| Defect | Root cause | Severity | Commit | Status |
|---|---|---|---|---|
| B-0 restart orphans in-flight instances | RC-A | CRITICAL | `7231199` | CLOSED |
| B-1 rebuild destroys `waiting` | RC-B | CRITICAL | `f689d5c` | CLOSED |
| B-2 stuck handler freezes runtime | RC-D | CRITICAL | `87d2d9d` | CLOSED |
| B-3 handler panic kills process | RC-D | CRITICAL | `87d2d9d`, `eb6813e` | CLOSED |
| B-4 terminal-route leaves `running` | RC-C | CRITICAL | `eb6813e` | CLOSED |
| B-5 cancel broken by stale version | RC-A | CRITICAL | `7231199`, `f689d5c` | CLOSED |
| B-6 flags after positional dropped | RC-G | CRITICAL | `1cc2cc1` | CLOSED |
| B-7 `trace --json` empty | RC-F | CRITICAL | `1cc2cc1` | CLOSED |
| B-8 discarded encode errors | RC-F | IMPORTANT | `1cc2cc1` | CLOSED |
| B-9/B-19 step types unreachable | RC-E | CRITICAL | `325203b`, `df1e4f3` | CLOSED |
| B-10 config split / unread | RC-H | IMPORTANT | `df1e4f3` | CLOSED |
| B-11 SDK recall wrong/stubbed | RC-H | IMPORTANT | `ee8f7d7` | CLOSED |
| B-15 retry/cancel/wait not durable | RC-A | CRITICAL | `7231199`, `eb6813e` | CLOSED |
| B-16 FTS quote handling | RC-H | IMPORTANT | `0178921` | CLOSED |
| B-17 `ListInstances` unbounded | RC-H | IMPORTANT | `0a315ff`, `be331f3` | CLOSED |
| B-18 plugin status drift | RC-I | IMPORTANT | `48a0dfc` | CLOSED |
| B-20 plugin remove cosmetic | RC-I | IMPORTANT | `57f21b7` | CLOSED |
| B-21 reference plugin uninstallable | RC-I | IMPORTANT | `48a0dfc` | CLOSED |
| B-22 no schema-version ceiling | RC-H | IMPORTANT | `0a315ff` | CLOSED |
| B-23 quickstart fails | RC-E | CRITICAL | `df1e4f3` | CLOSED |
| B-24 CLI runtime assembly | RC-E | CRITICAL | `df1e4f3` | CLOSED |
| B-25 `ListWorkflows` empty namespace | RC-H | IMPORTANT | `0a315ff` | CLOSED |
| B-26 namespace applied to half a command | RC-H | IMPORTANT | `41931a1` | CLOSED |
| **B-28 wait fields never populated** (new) | RC-H | IMPORTANT | `44577c9` | CLOSED |
| **B-30 concurrent submit → SQLITE_BUSY** (new) | RC-J | CRITICAL | `50ab926` | CLOSED |
| B-27 no read-model seam | RC-H | GUI blocker | `be331f3` | PARTIAL — see §6 |

### Two root causes discovered during execution

**RC-J — deferred transactions cannot upgrade to writers under WAL.**
`AppendEvent` opens a transaction, `SELECT MAX(sequence_num)`, then `INSERT`.
Under SQLite's default DEFERRED locking that begins as a *reader*; the write
upgrade fails with `SQLITE_BUSY` **immediately**, without honouring
`busy_timeout`, because a stale read snapshot cannot be fixed by waiting. WAL
and `busy_timeout=5000` were already configured, which is exactly why this
looked handled. Concurrent `awis submit` failed outright. Closed by
`_txlock=immediate` (B-30).

This defect was reachable **only** from the binary-level concurrency probe: it
requires separate *processes* contending for one file, which no in-process
unit test reproduces. It is the strongest single vindication of the evidence
hierarchy this program was run under.

**Fourth instance of "the correct implementation exists one file over."**
B-28: `signal_name` and `timeout_remaining_s` are declared in the published
status JSON schema and were never assigned, while `wait_records` held both and
`ListWaitRecordsByInstance` already existed to read them. The pattern's four
confirmed instances are B-16 (`sanitizeFTSQuery` in `apps/oip`), RC-A (five of
six engine maps had durable sources), B-1 (`RebuildState` already snapshotted
non-evented columns for definition identity), and B-28.

## 3. Dependency graph and execution order

```
        ┌─────────────────────────────────────────────┐
   ①    │ Integration harness (test/integration)      │  no deps — runs first,
        │ binary-level, tagged, outside `make verify` │  proves the baseline
        └─────────────────────────────────────────────┘
                            │
        ┌───────────────────┴───────────────────────────────────┐
        │  SERIAL — shared engine + storage files               │
   ②    │  Step 1 durable OCC version + seq        (B-0, B-5)   │
        │      ↓ (expectedVersion helper is a prerequisite)     │
        │  Step 2 durable waits + hydration        (B-15, B-4)  │
        │      ↓ (durable waits are a prerequisite)             │
        │  Step 3 cancel intent + rebuild          (B-5, B-1)   │
        └───────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┴───────────────────────────────────┐
   ③    │  PARALLEL — disjoint file sets                        │
        │  native isolation (B-2,B-3) │ CLI JSON+flags (B-6,7,8)│
        │  CLI assembly (B-9,19,23)   │ read model (B-10,11,16, │
        │  plugin lifecycle (B-18,20,21)              17,22)    │
        └───────────────────────────────────────────────────────┘
                            │
   ④    Full re-audit: every command, every step type, restart,
        rebuild, plugin lifecycle, concurrency, JSON contracts
                            │
   ⑤    Engine freeze verdict + GUI readiness package
```

**Serialised deliberately** (correctness depends on ordering, or files are shared):
schema changes, event semantics, recovery semantics, `internal/engine/*`,
`internal/storage/*`.

**Parallelised safely** (disjoint file ownership, no shared correctness decision):
the wave-③ lanes above, plus documentation and evidence collation.

---

## 4. Verification rule applied to every defect

A defect is closed only when all six hold:

1. root cause demonstrated (not inferred),
2. a regression test exists at the right layer,
3. the original failing scenario now passes,
4. the **integrated binary path** passes (`make integration`),
5. the relevant restart / rebuild / stress scenario passes,
6. no unintended contract drift — the G1 frozen surfaces are unchanged:
   `core.StoragePort`'s 12 methods, the 12 event types, event payload field
   names, `core.StepError`, and the forward-projection ≡ `RebuildState`
   equivalence fixtures.

Frozen-surface changes were forbidden for every agent working this program. Where
a fix needed new persistence it was added as an **additive migration** plus an
**additive `*SQLiteStorage` method reached through a locally-declared interface**,
which is the pattern already established in the repo by `cancellationStore`,
`signalWaitStore`, `PluginStore`, and `signal.Store`.

---

## 5. Final gate state

```
$ make verify
verify: ALL GATES PASSED          # gofmt, go vet, golangci-lint, unit + race,
                                  # oip module isolation, E1 contract

$ make integration
ok  github.com/awis/awis/test/integration    7.8s
```

The binary-level tier is fully green, including the restart, rebuild-state,
cancel-under-load, panic-containment, timeout, retry, fallback, on_error and
concurrency scenarios. Its red state at the start of this program was five
failures, of which two were genuine product defects (B-4, B-1) and three were
harness or fixture faults that had made the suite lie:

- `TestMain` sat in `harness.go`, a **non-test file**. Go only honours
  `TestMain` in `_test.go`, so it compiled, never ran, and left the binary path
  empty — every test failed with `exec: no command`.
- `waitForStatus` polled only on *status*, but an instance parked on signal
  "go" and the same instance parked on "done" are BOTH `waiting`, so a poll
  after delivering a signal returned instantly on the pre-signal state.
- Two tests asserted that `linear-native` FAILS with `handler_not_found` —
  encoding defect B-9 as the expected behaviour.

## 6. Remaining open items

None are in the approved CRITICAL/IMPORTANT scope. All are recorded so the
freeze decision is made against a complete picture.

| Item | Kind | Why it is not closed here |
|---|---|---|
| No global event cursor | GUI prerequisite | `execution_events.sequence_num` is per-instance; `emitted_at` is neither unique nor monotonic; `event_id` is an unordered UUID. SQLite's implicit `rowid` would work but `VACUUM` may renumber it. A durable event *stream* for a UI needs an explicit monotonic column — an additive migration, but a schema decision on the append-only table. |
| No definition → YAML serializer | GUI prerequisite | `dsl` parses YAML → definition; nothing does the reverse. A visual editor cannot round-trip. |
| `"default"` overloaded as a namespace wildcard | product decision | The global `--namespace` flag defaults to the literal string `"default"`, which the read commands treat as "every namespace". An instance genuinely in a namespace called `default` cannot be selected alone. Fixing it changes what `history`/`metrics`/`export` return for every existing project, so it needs a founder call, not a refactor. |
| `awis init` writes `namespace: default` while scaffolding workflows in `examples` | cosmetic | Works only because of the overloading above. Confusing in `config show`. |
| `waiting` is still a non-evented status | architectural (EDR-007 §9) | Now reconstructed from `wait_records` on rebuild and on restart, so both paths agree. Closing the gap properly means a 13th event type, and the vocabulary is frozen. |

## 7. Explicitly out of scope

- No `v1.0.0` tag.
- No GUI implementation until the freeze criteria in §8 of the execution brief are
  met.
- No new event types; the 12-type vocabulary stays closed.
- No redesign where a hardened sibling implementation already exists — B-16 reuses
  `apps/oip`'s `sanitizeFTSQuery` rather than writing a second escaper.
