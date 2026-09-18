# AWIS Engine Freeze Report

Branch `engine-hardening`, base `7146214`. 22 commits.
Companion to [ENGINE_HARDENING_PLAN.md](ENGINE_HARDENING_PLAN.md), which carries the root-cause analysis
and the defect→commit map.

Every claim below is **tier-1 evidence** (executed behaviour of the shipped
binary, or an executed test) unless labelled otherwise. Nothing is carried on
the authority of a prior audit.

---

## 1. Verdict

**The engine is ready to freeze. The GUI is not ready to start — two specific
prerequisites are missing, and both are small.**

| Measure | Before | After | Basis |
|---|---|---|---|
| CRITICAL defects open | 11 | **0** | each closed under the six-condition rule |
| IMPORTANT defects open | 12 | **0** | same |
| Binary-level integration tests | 0 | **21** | `make integration` |
| `make verify` | PASS | **PASS** | gofmt, vet, lint, unit, race, oip isolation, E1 |
| `make integration` | did not run at all | **PASS** | see §4 |

The "before" integration figure is not zero by oversight. The tier did not
exist; the only evidence available was unit tests calling engine internals —
precisely the class this program was told not to trust alone.

**Engine stability: high.** All twelve durability, recovery and failure-routing
defects are closed, each with a regression test whose load-bearing nature was
verified by reverting the fix and observing red. Note §9: one of those
verifications was originally too shallow and has been corrected.

**Integration coverage: good for the engine, partial for the periphery.** Every
step type, every failure route, restart, rebuild, cancel, signal, timeout,
retry and concurrency are covered against the shipped binary. Intelligence
steps against a live cloud provider are not (§5).

**GUI readiness: blocked on two items, neither large.** §6.

---

## 2. What was actually wrong

The 25 defects were not 25 independent bugs. Four root causes produced every
CRITICAL one.

**The engine kept durable state in process memory (RC-A).** Six per-instance
maps — sequence, version, retries, pending activations, waits, cancellations —
were lost on restart while the instances they described survived in SQLite. The
engine then acted on a zeroed view of live state. In five of six cases the
correct data was *already in the database*; the fix was to read rather than to
build.

**The join gate re-activated terminally-failed steps (RC-C).** `completedSet`
derives from `Variables` keys. A terminally-failed step leaves `current_steps`
but never enters `Variables`, so the gate saw its upstream complete and
re-nominated it every tick, forever. The claim was already held, so the step was
skipped, and the completion check bailed out because the activatable set was
non-empty: the instance sat in `running` with nothing running.

**The shipped CLI did not assemble a working runtime (RC-E).** `awis start`
registered zero native handlers. Two of five step types were unreachable
through the binary. The documented quickstart failed on its first two commands.

**Deferred transactions cannot upgrade to writers under WAL (RC-J).** Found
only by the concurrency probe, and the most instructive defect of the program —
see §3.

---

## 3. The defect that justifies the method

`awis submit`, run a few times concurrently:

```
awis: submit: engine: append WorkflowStarted seq=1:
  storage: AppendEvent insert: database is locked (5) (SQLITE_BUSY)
```

WAL was enabled. `busy_timeout=5000` was set. Both had been set for a long
time, and their presence is exactly why nobody looked here.

They do not help. `AppendEvent` opens a transaction, `SELECT MAX(sequence_num)`
to establish its monotonicity invariant, then `INSERT`s based on what it read.
Under SQLite's default DEFERRED locking that transaction **begins as a reader**
and only attempts the write upgrade at the `INSERT`. In WAL mode, if another
connection committed in between, the upgrade fails with `SQLITE_BUSY`
*immediately* — `busy_timeout` does not apply, because waiting cannot fix a
stale read snapshot. The transaction must be rolled back and retried.

`SetMaxOpenConns(1)` did not protect anything: it serialises writers within one
process, and every `awis` invocation is a separate process.

The fix is one DSN option, `_txlock=immediate`, so those transactions take the
write lock up front.

What matters is not the fix but the reachability. **No in-process unit test can
produce this defect** — it requires separate processes contending for one file.
It sat behind two correct-looking pragmas, in the single hottest write path in
the system, in a product whose central claim is an append-only event log. It
was found within minutes of the binary-level concurrency probe existing.

---

## 4. Integration tier

21 tests, build-tagged `integration`, excluded from `make verify`, run by
`make integration`. Every one launches the real binary as a subprocess in a
throwaway project and asserts what a user would see from a shell.

| Area | Covered |
|---|---|
| Lifecycle | submit → status → trace → completed; JSON contract across 10 read-only subcommands |
| Failure routing | fallback, `on_error`, plain `WorkflowFailed`, each asserting **exactly one** `StepStarted` per step. Only the `on_error` case guards B-4 — see §9; the other two routes are structurally immune |
| Retry | `StepFailed{retrying:true}` then success, with the attempt count pinned |
| Panic containment | a panicking handler yields `handler_panic`, the process survives, and a *subsequent* instance still completes |
| Timeout | a 30s handler behind a 1s timeout fails on deadline while an unrelated instance completes |
| Cancel | of a waiting instance, of a genuinely running instance, and across a restart; event sequence contiguous, downstream step never started |
| Restart | signal wait survives process restart and completes |
| Rebuild | `rebuild-state --all` preserves `waiting` and the cancellation flag |
| Wait visibility | `signal_name` is reported *and the reported name actually advances the instance* |
| Concurrency | 30 concurrent submissions all terminal; per-instance sequences contiguous under load; mixed success/fail/retry/recover simultaneously |

### Three harness faults that had made the suite lie

Worth recording, because each is a way a green suite can mean nothing.

`TestMain` was defined in `harness.go`, a **non-test file**. Go only honours
`TestMain` in a `_test.go` file, so it compiled, never ran, and left the binary
path empty — every test failed with `exec: no command`.

`waitForStatus` polled only on *status*. An instance parked on signal `go` and
the same instance parked on `done` are both `waiting`, so polling for
`"waiting"` after delivering a signal returned instantly on the **pre-signal**
state and asserted against it.

Two tests asserted that `linear-native` **fails** with `handler_not_found` —
encoding defect B-9 as the expected behaviour, so fixing B-9 broke them.

And in the stress tests as first written, `t.Fatalf` was called from spawned
goroutines. That calls `runtime.Goexit`, which kills the goroutine *without
failing the test* — a concurrent step could fail while the suite reported PASS.
Replaced with an error-returning helper.

---

## 5. Feature re-audit

All 22 registered commands exercised against the shipped binary. 18 read
commands returned exit 0 with valid JSON under `--json`. `prune-events`
correctly requires `--dry-run` (documented V1 behaviour, exit 2 with guidance).
Lifecycle commands — `submit`, `signal`, `cancel`, `stop`, `rebuild-state`,
`workflow validate`, `config set`, `plugin install/list/remove` — all verified.

The documented quickstart now works end to end:

```
awis init .  →  awis start  →  awis submit hello-world --input name=World
                                                    → completed
```

**Step types.** `native` (built-in handlers, formerly unreachable),
`signal` (wait/deliver/timeout), `plugin` (shipped reference plugin, live),
`intelligence` (zero-AI mode and adapter-level tests), subprocess (unit tier).

**Not covered, and stated plainly:** intelligence steps against a *live*
Anthropic endpoint. The adapter is exercised through its port contract tests
and the null adapter, not a billed call. Also unverified: multi-day wait
timeouts, and behaviour at the 100K-event scale (an informational benchmark
exists, not an assertion).

**Security constraints held.** No API key is printed anywhere. Two hardenings
were made beyond the brief: `config show`/`config set` masking now fails
**closed** — matching key/token/secret/password/credential substrings rather
than an exact allowlist of two names, so a future provider's credential is not
printed verbatim into terminal scrollback and the append-only audit table —
and `api_keys/` was found in the working tree untracked *and unignored*, one
`git add -A` from an unrecoverable commit. Verified it never entered history;
the path is now ignored.

---

## 6. GUI readiness

**What already works, verified:** the engine loads definitions from *storage*
on demand, and `Submit` falls back to storage when a definition is not in its
in-memory registry. So a GUI can register a workflow and have a **running**
daemon execute instances of it with no restart. Hot registration is not a gap.

`Runtime.ListPaged` now exposes bounded, stably-ordered instance reads with
totals — the seam a list view needs. It refuses rather than degrades: a storage
without the capability returns a typed error, never a silent fallback to an
unbounded read that gets sliced.

**Two blockers, both small:**

**No global event cursor.** `sequence_num` is per-*instance*; `emitted_at` is
neither unique nor monotonic; `event_id` is an unordered UUID. There is no
column a UI can tail to stream "what happened since I last looked". SQLite's
implicit `rowid` would work, but `VACUUM` may renumber it, so relying on it is
a latent corruption of the cursor. Recommendation: additive migration adding a
monotonic `global_seq`, backfilled from `rowid`, assigned inside the existing
`AppendEvent` transaction. Additive to an append-only table — a real schema
decision, and the founder's call.

**No definition → YAML serializer.** `dsl` parses YAML into a definition;
nothing does the reverse, anywhere in the repo. A visual editor cannot
round-trip a workflow it edits. This is squarely in the M10 "PRD §18 rendering"
scope already planned.

**Recommended shape**, once those land: a thin HTTP layer over the existing SDK
(`ListPaged`, `Status`, `QueryHistory`, `ReplayInstance`, `StepStats`,
`Submit`, `Signal`, `Cancel`) — the read model is already the right shape and
should not be reimplemented. Event streaming as SSE over a `global_seq` cursor.
The frontend is fully parallelisable against a fixture server once the JSON
contract is pinned, and that contract is now covered by integration tests.

---

## 7. Freeze conditions

Satisfied: zero open CRITICAL or IMPORTANT defects; `make verify` and
`make integration` both green; every fix carries a regression test verified to
be load-bearing; the frozen G1 surfaces are unchanged — the 12 `StoragePort`
methods, the 12 event types, event payload field names, `core.StepError`, and
the forward-projection ≡ `RebuildState` equivalence fixtures, which still pass
unweakened.

Every additive change went through the pattern the repo already established:
an additive migration or `*SQLiteStorage` method reached through a
locally-declared interface and a type assertion, as `cancellationStore`,
`signalWaitStore`, `PluginStore` and `signal.Store` already do.

Outstanding before a `v1.0.0` tag — **not tagged, per the brief**:

1. A founder decision on the `"default"` namespace overloading (§6 of the plan).
2. `global_seq` migration, if event streaming is wanted in V1.
3. A live intelligence-path smoke test against a real provider, or an explicit
   decision to ship that path unverified.

---

## 8. Evidence table

| Defect | Root cause | Fix | Regression test | Integrated proof | Status |
|---|---|---|---|---|---|
| B-0 restart orphans instances | in-memory OCC version | read `workflow_instances.version` | `TestRestart_DurableVersionSurvivesProcessRestart` | restart + signal → completed | CLOSED |
| B-1 rebuild destroys `waiting` | non-evented status, replay-only rebuild | derive from `wait_records`; snapshot cancel flag | 5 tests; 3 verified red on revert | `TestB1_RebuildStatePreservesWaitingStatus` | CLOSED |
| B-2 stuck handler freezes runtime | no default timeout | `defaultProductionTimeout` | native runner tests | `TestB2_SlowHandlerTimesOutAndDoesNotWedgeRuntime` | CLOSED |
| B-3 panic kills process | no `recover()` | recover in runner **and** engine backstop | 6 dispatch tests | `TestB3_HandlerPanicIsContainedAndRuntimeSurvives` | CLOSED |
| B-4 terminal route leaves `running` | `completedSet` from `Variables` | exclude terminally-failed set | 3 tests; verified red on revert | 3 integration tests counting `StepStarted` | CLOSED |
| B-5 cancel broken | stale version; intent lost on rebuild | durable version + preserved flag | `TestRebuildPreservesCancellationRequested` | `TestCancel_SurvivesRestart` | CLOSED |
| B-6 flags after positional | stdlib `flag` stops at positional | `permuteArgs` | `cmd/awis` tests | whole suite passes flags after positionals | CLOSED |
| B-7/B-8 JSON integrity | discarded encode errors; bad truncation | `emitJSON`; marshal on rune boundary | `cmd/awis` tests | JSON contract across 10 subcommands | CLOSED |
| B-9/B-19/B-23/B-24 CLI unusable | no handlers registered; namespace mismatch | built-in handlers; 2-stage resolution | `TestScaffoldWorkflowsResolveAgainstBuiltins` | full quickstart → completed | CLOSED |
| B-10 config split | two paths, neither read by `start` | `configPath` precedence | `TestConfigPathPrefersProjectRoot` | `config show` after `init` | CLOSED |
| B-11 SDK recall stubbed | empty structs | populate `ReplayTrace`; scope `StepStats` | sdk tests | — | CLOSED |
| B-15 retry/pending not durable | in-memory maps | `hydrate()` from EventLog | 5 hydrate tests | restart scenarios | CLOSED |
| B-16 FTS quoting | bare quote wrap | reuse `apps/oip` escaper | 4 recall tests | — | CLOSED |
| B-17 unbounded reads | no LIMIT | `ListInstancesPaged` + SDK `ListPaged` | 6 storage + 7 sdk tests | — | CLOSED |
| B-18 plugin status drift | 4 fire-and-forget goroutines | single ordered writer | plugin tests | — | CLOSED |
| B-20 plugin remove cosmetic | capabilities not excluded | exclude at lookup + txn cleanup | 6 tests | — | CLOSED |
| B-21 shipped plugin uninstallable | no `cmd.Dir`; bootstrap in `__main__.py` | manifest dir + relocate bootstrap | `TestB21_ShippedManifestRunsFromAnyWorkingDirectory`, verified red on revert | shipped manifest, foreign cwd, no injected env | CLOSED |
| B-22 no schema ceiling | never range-checked | `MaxSupportedSchemaVersion` | 4 tests | — | CLOSED |
| B-25 `ListWorkflows` empty ns | unconditional predicate | empty = no predicate | `TestListWorkflowsEmptyNamespaceMeansAllNamespaces` | — | CLOSED |
| B-26 half-filtered export | hardcoded `ListWorkflows("")` | one shared predicate | `TestNamespacePredicate` | live: defs 3→0 for unknown ns | CLOSED |
| B-28 wait fields always null | declared, never assigned | read `wait_records` | 7 tests | `TestB28_WaitingInstanceReportsWhichSignalItAwaits` | CLOSED |
| B-30 concurrent submit fails | DEFERRED tx cannot upgrade under WAL | `_txlock=immediate` | 3 DSN tests | 3 stress tests, verified red on revert | CLOSED |
| B-27 read-model seam | — | `ListPaged` landed | 7 sdk tests | — | PARTIAL (§6) |

### Risks

The `"default"` namespace overloading is the one place where correct-looking
behaviour rests on a coincidence. It works today; it will surprise someone.

`waiting` remains a non-evented status. Rebuild and restart now agree because
both read `wait_records`, but that agreement is maintained by two call sites
rather than enforced by the event model.

### Unresolved contradiction

The Anthropic adapter targets `claude-haiku-4-5` and `claude-sonnet-4-5`. Both
were confirmed Active — stale but valid. Whether to move to current models is a
product and cost decision, not a defect, and is left to the founder.

### Next execution step

Founder decision on the three §7 items. If event streaming is wanted in V1, the
`global_seq` migration is the single highest-value next change: it is additive,
small, and unblocks the GUI's entire live-update story.

---

## 9. Post-review round

After the report above was first written, three independent adversarial
reviews were run against the branch: a clean-worktree verifier that re-derived
every gate and tried to prove each regression test was NOT load-bearing, and
two correctness reviews splitting engine/storage from CLI/SDK/plugin/security.

They found **seven real defects, one of them a security defect and one a
correction to a claim in this report.** All are fixed and committed; both gates
are green again, and the integration count is unchanged at 21.

| Finding | Severity | Fix |
|---|---|---|
| `config show`/`set`/audit printed the value of any credential-shaped key the substring denylist failed to anticipate (`authorization`) | **SECURITY** | inverted to an allowlist of the eight non-credential settings; unknown keys are secret by default |
| `RebuildState` never checked `schema_version`, so `awis rebuild-state` would silently reinterpret a newer build's events under v1 assumptions — while wiping the projection | **HIGH** | pre-flight ceiling check before any destructive work |
| B-18 migrated only 3 of 4 status-write sites; `handleCrashLocked` still fire-and-forget, and the doc comment claimed otherwise | **HIGH** | migrated; source-level guard test against a fifth |
| `permuteArgs` consumed its own `--` terminator as a dangling flag's value, turning an exit-2 usage error into silent success with a nonsense value | **MEDIUM** | terminator suppressed when a flag is left dangling |
| **The B-4 integration tests passed with the fix removed** | **MEDIUM** | fixture reshaped — see below |
| `awis signal` compared only `CurrentSteps[0]`, mis-reporting a multi-branch advance | MEDIUM | compares the whole parked set |
| `asInt("12abc") == 12` — `Sscanf` accepted trailing garbage | LOW | `strconv.Atoi` |
| sdk page-bound drift guard compared a constant to itself | LOW | storage pins the literals |

### The correction that matters

This report claimed the integration B-4 tests were "the direct B-4 instrument".
They were not. All three passed with the guard removed, because `primary` was
the **initial** step — and `isActivatable` only re-nominates a step with an
inbound transition from a completed step. The fixture was structurally immune
to the defect it was named after.

Adding a `seed` predecessor fixed it: with the guard removed,
`TestB4_OnErrorRoutingReachesCompleted` now times out at 20s (the hang
signature) instead of passing in 0.23s.

Tracing this also established something worth recording: **`on_error` is the
only route where B-4 can occur.** The fallback route is immune because
`StepFallbackActivated`'s projection writes a Variables sentinel for the
originating step (so convergent join gates see it as done), which also places
it in `completedSet`. The plain `WorkflowFailed` route is immune because the
instance goes terminal in the same tick. The guard is load-bearing on exactly
one of three branches, and the two shape-parity tests now say so.

### Independently confirmed safe

The reviewers also checked, and cleared, the concern I had flagged about the
B-30 fix: `_txlock=immediate` does **not** serialise reads. Every `BeginTx`
site in storage is a genuine writer, and every hot read the 100ms tick loop
performs (`ListInstances`, `GetInstance`, `ReadEvents`, `ListInstancesPaged`)
is a plain autocommit query that never sees the option — confirmed against the
driver source. `busy_timeout=5000` was also shown to be nowhere near binding
under the 30-instance stress load.

No test was found to have been weakened or deleted. The only removal in the
whole branch was a vacuous stub-success test, replaced by two stronger ones.
The G1 frozen surfaces show zero diff.
