# Report 2 — After the adversarial review

Branch `engine-hardening`, base `7146214`. 29 commits (22 at the point of
[Report 1](01-PRE-REVIEW-REPORT.md), +7 from this round).

Three independent reviews were run against the branch. They found **eight real
defects**, including one security defect and one correction to a claim Report 1
made about its own evidence. All are fixed, committed, and verified.

---

## 1. What was run

| Reviewer | Scope | Method |
|---|---|---|
| Independent verifier | all gates, every regression test | clean git worktree; re-derived every gate, then tried to prove each regression test was **not** load-bearing by reverting the fix under it |
| Engine / storage reviewer | `internal/engine`, `internal/storage` | adversarial correctness pass; wrote and ran white-box probes to confirm hypotheses |
| CLI / SDK / plugin / security reviewer | `cmd`, `sdk`, `internal/plugin`, `internal/examples`, `plugins`, `python` | adversarial pass; reproduced findings against the **real binary** |

`/code-review ultra` is user-triggered and billed; it could not be launched
from inside the session. This round is the in-session substitute and overlaps
much of the same ground.

---

## 2. Findings and fixes

| # | Finding | Severity | Status |
|---|---|---|---|
| 1 | `config show` / `config set` / audit printed the value of any credential-shaped key the substring denylist failed to anticipate | **SECURITY** | fixed `0ddf1ce` |
| 2 | `RebuildState` never checked `schema_version` — `awis rebuild-state` would silently reinterpret a newer build's events under v1 assumptions, *while wiping the projection* | **HIGH** | fixed `c444b40` |
| 3 | B-18 migrated only 3 of 4 status-write sites; `handleCrashLocked` was still fire-and-forget — and the doc comment claimed otherwise | **HIGH** | fixed `41e288e` |
| 4 | **The B-4 integration tests passed with the fix removed** | **MEDIUM** | fixed `e370d92` |
| 5 | `permuteArgs` consumed its own `--` terminator as a dangling flag's value | **MEDIUM** | fixed `acc3e26` |
| 6 | `awis signal` compared only `CurrentSteps[0]`, mis-reporting a multi-branch advance | MEDIUM | fixed `acc3e26` |
| 7 | `hydrate` could place a step in **both** `e.pending` and `e.failed` after a restart | LOW (latent) | fixed `e370d92` |
| 8 | `asInt("12abc") == 12` — `Sscanf` accepted trailing garbage; sdk page-bound drift guard compared a constant to itself | LOW | fixed `62f53e9` |

---

## 3. The security defect

`isSecretConfigKey` matched credential-shaped **substrings**
(`key`/`token`/`secret`/`password`/`credential`). That is a denylist wearing a
disguise: it printed the value of any key whose name the list failed to
anticipate.

A key named `authorization` — the exact shape a non-Anthropic provider's Bearer
credential takes — matched none of those fragments. Reproduced live:

```
awis config set authorization "Bearer <secret>"
awis config show          → printed verbatim
awis --json config show   → printed verbatim
awis --json audit         → stored verbatim, in an APPEND-ONLY table
```

The audit row is the serious part. It cannot be redacted afterwards.

**Fix: inverted to an allowlist.** Only eight settings — the recognised V1 keys
that hold no credential, plus `model` and `timeout` — have their values
printed. Everything else, including every key nobody has thought of yet, is
masked.

Enumerating every credential-shaped name is not a winnable game. Enumerating
the handful of settings an operator needs to read back is trivial. The costs
are wildly asymmetric: a masked non-secret costs one `cat config.yaml`; a
printed secret is in terminal scrollback, CI logs and an immutable audit row
forever.

`TestVisibleKeysHoldNoCredential` guards the one direction the inversion
cannot — someone adding a credential-shaped name to the visible list.

**Verified live:** `authorization` masked in all three sinks, `namespace` still
readable, zero sentinel leaks.

---

## 4. The correction to Report 1

Report 1 called the integration B-4 tests "the direct B-4 instrument". **They
were not.** All three passed with the guard removed.

The cause was the fixture's shape. B-4 is the join gate re-nominating a
terminally-failed step, and `isActivatable` only re-nominates a step with an
**inbound transition from a completed step** — a step with no inbound edges
self-activates only when it is the initial step and nothing has completed yet.
`primary` *was* the initial step, so the fixture was structurally immune to the
defect it was named after.

Adding a `seed` predecessor fixed it. With the guard removed,
`TestB4_OnErrorRoutingReachesCompleted` now times out at **20s** (the hang
signature) instead of passing in 0.23s.

### What tracing this established

**`on_error` is the only route where B-4 can occur**, and this is now recorded
where it will be read:

- The **fallback** route is immune because `StepFallbackActivated`'s projection
  writes a `Variables` sentinel for the *originating* step — so convergent join
  gates treat it as done, which also places it in `completedSet`, so
  `activatableFor`'s `completed[id]` check excludes it before `failed[id]` is
  ever consulted.
- The **plain `WorkflowFailed`** route is immune because the instance goes
  terminal in the same tick, leaving no later tick to re-activate on.

The guard is load-bearing on exactly one of three branches. The two
shape-parity tests now say so in their comments, and explain why they are still
worth keeping: `_Fallback` protects an immunity that depends on a projection
rule in a different file — if that sentinel write were ever removed, that
fixture would start hanging and point straight at the interaction.

---

## 5. The corruption path

`AppendEvent`, `ReadEvents` and `ReadEventRange` all refuse rows above
`MaxSupportedSchemaVersion`, on the stated reasoning that mis-projecting the
append-only source of truth is worse than a failed read.

`RebuildState` replays `execution_events` with its own raw SQL and never
selected `schema_version` at all. So it was the one path that would silently
reinterpret a newer build's events under v1 field assumptions — and it is the
worst possible place for that gap, because it **wipes the projection first**.
The mis-projection replaces correct state rather than merely accompanying it.

The check is now pre-flight, before any destructive work: a rebuild is
all-or-nothing, and discovering this halfway through would leave the operator
with a wiped projection and an error. The regression test asserts both halves —
the typed `ErrUnsupportedSchemaVersion`, and that the existing projection
survives a refused rebuild intact.

---

## 6. The doc comment that was worse than the bug

The B-18 fix converted three of four fire-and-forget
`go func(){ _ = m.store.SetPluginStatus(...) }()` sites. `handleCrashLocked`
was missed — and the doc comment written alongside it claimed all of them had
been converted.

The false comment is the worse half. It would have stopped the next reader from
checking.

That site is reachable from a plugin read/protocol error during `Call`. Its
goroutine was not tracked by `Shutdown`'s `WaitGroup`, so the `"failed"` write
could be lost at process exit, and could still land out of order relative to a
later-queued write — precisely the failure mode B-18 exists to prevent.

`TestEveryStatusWriteGoesThroughTheOrderedWriter` now asserts at the **source
level** that `drainStatusQueue` is the only direct caller. A source assertion is
the right shape here: the defect is "someone added or kept a write that bypasses
the queue", and the race it creates is timing-dependent, so no behavioural test
catches it reliably.

---

## 7. What the reviews cleared

Not everything was a finding. These were checked and found sound:

- **`_txlock=immediate` does not serialise reads.** This was my own stated
  worry about the B-30 fix. Every `BeginTx` site in storage is a genuine
  writer; every hot read the 100ms tick loop performs (`ListInstances`,
  `GetInstance`, `ReadEvents`, `ListInstancesPaged`) is a plain autocommit
  query that never sees the option — confirmed against the driver source, where
  `beginMode` is consulted only inside `conn.Begin`/`BeginTx`.
- **`busy_timeout=5000` is nowhere near binding** under the 30-instance stress
  load (0.35–0.51s observed).
- **No test was weakened or deleted.** The only removal across the whole branch
  was a vacuous stub-success test, replaced by two stronger ones. No
  `time.Sleep`-as-synchronisation was introduced. The one `t.Skip` added is
  environment-gated (`python3 not available`), not failure-masking.
- **G1 frozen surfaces show zero diff** — the 12 `StoragePort` methods, the 12
  event types and their payload field names, and `core.StepError`. The
  forward-projection ≡ `RebuildState` equivalence fixtures pass unweakened.
- **No secrets in branch history.** `.gitignore` covers `api_keys/`, `*.pem`,
  `secrets.yaml`, `.env*`.
- **`trace.go` JSON escaping** survives embedded quotes, backslashes, unicode,
  emoji and long payloads in both `--json` and `--json --full`.
- **`buildDSN`** handles `:memory:` and `file:` URIs correctly. A path
  containing a literal `?` mis-splits, but that is a pre-existing
  `modernc.org/sqlite` DSN quirk present before this change, not a regression.

---

## 8. Final gate state

```
make verify        →  verify: ALL GATES PASSED
make integration   →  ok  (8.2s, 8.2s — deterministic)
go test -race ./... →  clean
integration tests  →  21
```

Every fix in this round was verified **load-bearing** by reverting it and
observing the corresponding test go red, then restoring:

| Fix | Reverted → observed |
|---|---|
| B-4 fixture | `TestB4_OnErrorRoutingReachesCompleted` times out at 20s |
| B-22 rebuild guard | `RebuildState err = <nil>, want ErrUnsupportedSchemaVersion` |
| B-18 ordered writer | source guard fails, naming `manager.go:642` |
| `permuteArgs` | dangling flag silently parsed as `--` instead of erroring |
| config masking | sentinel value printed in all three sinks |

---

## 9. Standing open items

Unchanged from Report 1 — no new blockers were introduced, and none of the
findings above moved the freeze verdict.

| Item | Kind |
|---|---|
| No global event cursor | GUI prerequisite |
| No definition → YAML serializer | GUI prerequisite |
| `"default"` overloaded as a namespace wildcard | product decision |
| `awis init` namespace mismatch | cosmetic |
| `waiting` is still a non-evented status | architectural (EDR-007 §9) |
| Intelligence path untested against a live provider | unverified, stated |

**Verdict unchanged: the engine is ready to freeze; the GUI is blocked on two
small prerequisites.** No `v1.0.0` tag.

### One process note

Three defects in this round (the false doc comment, the immune B-4 fixture, the
substring denylist) were *self-inflicted by the hardening work itself* and
survived my own verification pass. In each case I had looked at the right thing
and drawn too generous a conclusion — I observed that only one of three B-4
tests went red and reasoned it away rather than treating it as a coverage gap.
That is the specific value the independent reviewers added, and it is an
argument for running them before declaring a freeze, not after.
