# AWIS GUI Master Plan

**From the current engine to an n8n-class visual product.**

Status: proposed — awaiting founder decisions D1–D4 (§12).
Basis: repo-wide architecture review of branch `engine-hardening` @ `f004f4f`, 2026-08-29.
Evidence tier: every claim is cited to code, a test, a migration, or observed runtime
behaviour. Where a report and the code disagree, the code is recorded and the report
is corrected.

---

## 0. Executive answer

**Can GUI development start tomorrow? Yes — for the frontend and the read-only
backend. No — for the live execution view.**

Three facts decide this:

1. The engine is genuinely embeddable and the core read/write surface is reachable
   from an external process today. I proved this by compiling an out-of-module Go
   program against `sdk` alone: `NewRuntime`, `ListPaged`, `Status`, `Submit`,
   `Signal`, `Cancel`, `Recall().ReplayInstance` all resolve and build.
2. There is **no network surface of any kind** in the repository. A repo-wide grep
   for `ListenAndServe`, `net.Listen`, `http.Server`, `grpc.NewServer` over
   non-test Go returns zero matches. Every byte of the API layer is greenfield.
3. **The event log cannot describe a live workflow.** `waiting` is written to the
   projection without emitting any event (`internal/engine/signal.go:149-179`),
   and so are signal-timeout `continue` (`signal_timeout.go:128-140`) and
   cancellation-requested (`cancel.go:118`). A UI tailing the EventLog would watch
   an instance emit `StepStarted` and then fall silent forever while it parks on a
   signal. I reproduced this on a live daemon (§3.1).

So the sequencing is: **frontend and history views can start immediately; the live
view needs one additive migration first.** That migration is small, and it is not
the one the freeze report recommends.

---

## 1. Engine freeze verdict

**FREEZE-READY WITH TWO CORRECTIONS.** The hardening program's headline claims hold
up under independent verification.

| Claim | Verdict | Evidence |
|---|---|---|
| `make verify` green | **CONFIRMED, with a caveat** | ran it; exit 0, "verify: ALL GATES PASSED". But `cmd/awis`'s `TestSystemRehearsalInitStartSubmitTrace` is a known flake under full-suite CPU contention (hardcoded 10s deadline; already logged in `STATE.md` LAST-6). Green is reproducible, not unconditional. |
| `make integration` green | **CONFIRMED** | ran it; exit 0, 8.4s |
| 21 binary-level integration tests | **CONFIRMED, and genuinely binary-level** | `go test -tags integration -list` returns exactly 21; `main_test.go:39` runs `go build -o <tmp>/awis ./cmd/awis` and `harness.go:91,163` `exec.Command`s that binary. Zero `t.Skip` among them. |
| Frozen G1 surfaces unchanged | **CONFIRMED** | 12 event types (`internal/core/event.go`), 12 `StoragePort` methods (`ports.go`), both counted directly |
| "B-0 through B-30 addressed" | **CONFIRMED for the defects that exist — but the range is wrong** | **B-12, B-13, B-14 and B-29 do not exist.** No commit, comment, or test references them. The real set is 26 IDs: B-0…B-11, B-15…B-28, B-30. The reports never claimed those four either; the range is an artifact of the framing. |
| Every fix carries a load-bearing regression test | **CONFIRMED by live revert on a sample** | B-4's guard and B-30's `_txlock=immediate` were each reverted in a scratch tree; both times the named test failed with exactly the documented symptom, then passed on restore. |
| No open CRITICAL defects | **CONFIRMED** | see §2 |
| No open IMPORTANT defects | **CORRECTED — one was open (now fixed), one PARTIAL** | B-31 was open; **resolved 2026-08-30 along with four adjacent defects of the same class — see §1.2.** B-27 (global event cursor) is correctly self-labelled PARTIAL, not closed — it is the same thing as blocker B-b below. |

The freeze report is unusually honest for its genre. It correctly self-identifies
the `rowid`/`VACUUM` hazard, correctly states the live-provider gap, and correctly
flags the namespace decision as the founder's. I found no instance of it claiming a
fix that does not exist. Two things it understates, and one it gets wrong:

- It calls GUI readiness "blocked on two items, both small." That is true only for a
  *list-and-history* GUI. For a *live* GUI it is wrong, because the global cursor it
  recommends would stream a log that does not contain the waiting transition (§3).
- It does not carry the non-evented transitions as a GUI blocker, though the founder's
  own brief does (blocker #5). Those are the same problem, and it is the deeper one.
- "The JSON contract is now covered by integration tests" — one test
  (`TestLifecycle_JSONContract`) covers one command's shape. `docs/CLI_CONTRACT.md`
  diverges from the binary on at least eleven points (§6.3).

### 1.1 Two corrections to the record (B-31 since remediated — see §1.2)

**B-31 (IMPORTANT) — RESOLVED 2026-08-30. Retained here as the record of what was
wrong and why; the remediation and the wider defect class it exposed are in §1.2.**

`timeout_action` was unvalidated, and the shipped scaffold was invalid.

`awis init` writes `workflows/with-signal.yaml` containing `timeout_action: cancel`
(`cmd/awis/scaffold/workflows/with-signal.yaml:25`, and the same in
`examples/workflows/with-signal.yaml:25`). The legal values are `fail | compensate |
continue` (`internal/core/step.go:81`). `internal/validate/validate.go` contains no
check for the field — grep confirms zero references. `awis workflow validate` reports
the file **valid ✓** (I ran it against a freshly built binary). At runtime,
`internal/engine/signal_timeout.go:141-147` falls to `default`, logs
`"unknown timeout_action; treating as fail"` at Warn, and fails the workflow.

Net effect: every project created by `awis init` ships a workflow that passes
validation and then, 72 hours later, silently does the opposite of what it says.
This is exactly the class the hardening program was chartered to close.
*Fix: one validation case + one scaffold edit. Under an hour.*

**The repo-root `awis` binary is stale.** It is dated Aug 26; the B-28 fix landed
Aug 28 (`44577c9`). Testing against it, `awis --json status` reported
`signal_name: null` for a waiting instance whose `wait_records` row plainly held
`approved`. Rebuilding from source produced the correct `"signal_name": "approved",
"timeout_remaining_s": 259102`. **B-28 is genuinely fixed**; the binary simply
predates it. The file is gitignored (`.gitignore:29`) and untracked, so this is
hygiene rather than a defect — but any "verified live against the shipped binary"
claim is only as current as that file, and it should be deleted or rebuilt before
future live audits.

### 1.2 B-31 remediation (2026-08-30) — and what it turned out to be

B-31 was not a one-line defect. Tracing it produced a **class**.

**Origin.** `cancel` was never a legal value. `docs/DSL.md:83` documented the legal
set as `fail | cancel | continue` — contradicting `internal/core/step.go:81`, the
`wait_records` DDL comment, and the engine's own switch — and `docs/DSL.md:91`
demonstrated it. The scaffold was authored from the wrong doc. Fixing only the
validator would have left the error free to re-propagate.

**The class.** Enum-valued schema fields documented as frozen sets, with no validator
rule, silently defaulted at runtime. Five instances; `model_hint` and `context_budget`
were the only two already validated.

| Field | Was | Runtime on bad value | Now |
|---|---|---|---|
| `wait_signal.timeout_action` | unvalidated | **silent** → `fail` | `CodeSignalTimeoutAction` (empty rejected) |
| `retry.backoff` | unvalidated, and `retry.go:50` falsely claimed "the validator rejects others" | **silent** → 0-delay hot retry loop | `CodeRetryBackoff` (empty allowed; also covers compensation steps) |
| `trigger.type` | unvalidated | silent — `webhook` never fires | `CodeTriggerTypeUnknown` |
| `step.type` | switch had no `default:` | loud `runner_unavailable` | `CodeStepTypeUnknown` |
| `intelligence.capability` | unvalidated | loud `capability_unknown` | `CodeIntelligenceCapability` |

`retry.backoff` was arguably the more dangerous of the two silent ones: B-31 fails one
workflow slowly, a typo'd backoff hammers a failing dependency with zero delay.

**Design.** `internal/engine/submit.go:30` already calls `validate.Validate`, so the
validator is a single insertion point covering both `awis workflow validate` and the
runtime submit path. The runtime `default:` degrade branches were deliberately left
**unchanged** as backstops for values persisted before the fix.

**Also found and corrected:** two further `timeout_action: cancel` instances outside
the scaffold (`apps/oip/workflows/capture-decision.yaml`,
`internal/dsl/testdata/capture-decision.yaml`) and one Go literal
(`apps/oip/qg3_test.go`); `modelQuality` `claude-sonnet-4-5` → `claude-sonnet-5`
(`modelFast` was already current); `docs/PROVIDERS.md` and `docs/DSL.md`.

**Upgrade impact — disclosed, not free (R18).** Because `Submit` re-validates stored
definitions, a workflow registered *before* this change containing an illegal value is
now **rejected at submit**. Verified against a pre-fix database:

```
awis: submit: engine: Submit rejected with-signal@1.0.0:
  [signal-timeout-action] await-approval.wait_signal.timeout_action:
  timeout_action must be one of fail|compensate|continue, got "cancel"
```

**In-flight instances are unaffected** — also verified: a `wait_records` row carrying
`cancel` with an overdue timeout was routed by the backstop exactly as before
(`StepFailed` → `WorkflowFailed`, Warn logged). So the break is confined to *new
submissions of old bad definitions*, and it is loud and actionable. That is the right
trade: those workflows were already doing the opposite of what they said.

**One observability gap this exposed, not fixed.** The backstop's `Warn` goes to
**stderr**, because `sdk/runtime.go:122` passes `nil` for the logger and
`engine.go:129` defaults to a stderr handler. It never reaches `.awis/awis.log`, so
`awis logs` cannot show it — an operator running the daemon with stderr discarded sees
nothing. Engine-level diagnostics being invisible to the operator log command is a
separate defect worth its own card.

### 1.3 Where the reports would mislead a reader

The reports are honest, but three things a reader should not take at face value:

- **Report 1 lists B-18 and B-22 as flatly CLOSED.** Both were incomplete at the
  commits it cites, and only became true after the post-review corrections
  (`41e288e`, `c444b40`). Report 2 discloses this itself — but the freeze report's
  evidence table carries the uncaveated "CLOSED", so anyone reading that table alone
  is misled.
- **"29 commits"** (README, Report 2) — the branch has 30. **Independently
  reconfirmed**: `git log --oneline 7146214..HEAD` returns exactly 30 commits.

**Correction (post-publication, independent re-verification):** this document
originally claimed *"B-11 has zero in-source traceability... exists only in a
commit message."* That claim is false and is retracted. `ee8f7d7: B-11:
correct the SDK recall surface` is itself inside the 30-commit range above, and
B-11 (with sub-IDs B-11a/b/c) is documented extensively in source: `sdk/variables.go:3`,
`sdk/runtime_recall.go:79,122`, four separate comments in
`sdk/runtime_recall_test.go` (lines 168, 187, 270, 318, 447), and
`sdk/variables_test.go:1`. A `git blame`-driven audit finds it easily. This is
recorded here rather than silently fixed, per this document's own evidence
standard: a confident claim, independently checked, and found wrong.

### 1.4 A governance finding, outside the reports' scope

`CLAUDE.md` carries a frozen Model Allocation Policy: *"Opus MUST NOT be selected
automatically,"* and implementation, testing, verification, refactoring and
documentation *"MUST be performed using Sonnet."* All 30 commits on this branch carry
`Co-Authored-By: Claude Opus 5`. By its own git trail, the engine-hardening program
was executed in violation of that policy.

I raise this as evidence, not as a judgement on the work — the work is good, and this
review found the reports substantially accurate. But it is Tier-1 evidence sitting in
the log, the reports could not self-report it, and it bears on whether the policy is
still the intended constraint. If it is, the GUI program needs an enforcement
mechanism rather than a written rule. If it is not, the policy should be amended so
the next audit does not re-raise it. (For this review: the investigation was delegated
to Sonnet and Haiku subagents per your instruction; supervision and the decisions in
§12 are mine.)

### 1.5 A second governance finding: the branch this plan evaluates is not `main`

Everything above, and everywhere else in this document set, is evaluated against
`engine-hardening` (checked out HEAD `f004f4f` at the time of writing). That branch
is not merged into `main`, and neither is any of its ancestor chain back to `main`'s
tip — `git rev-list --count main..engine-hardening` = 60.

**What `main` actually has.** `git merge-base --is-ancestor m10-yaml-dsl main`
succeeds; `git merge-base --is-ancestor m17-full-cli-init main` fails. `main` HEAD
(`98350f6`) includes M10–M14 (squash-landed at `f6aa755`, "M10-M14") and stops there.
It has **no OIP-on-AWIS (M15), no Anthropic adapter (M16), no full CLI (M17)**, and
none of the 30 engine-hardening commits (`7146214..HEAD`) — the program that closed
11 CRITICAL and 12 IMPORTANT defects, several data-loss-class (RC-A: durable state
kept in process memory only, lost on restart while SQLite disagreed; RC-C: the join
gate re-activating terminally-failed steps forever — see `ENGINE_FREEZE_REPORT.md`
§2).

**The ledger has not been told any of this.** `docs/05-implementation/STATE.md` — on
`main` itself, confirmed via `git show main:docs/05-implementation/STATE.md`, not
just this branch — still lists M10 through M14 as `PHASE: E-MERGE (blocked on
founder)`, i.e. "awaiting founder merge," even though they have been in `main` since
`f6aa755`. M15–M17 are correctly shown as unmerged, but M17 itself has sat in
`C-VERIFY` since `7146214` with one explicitly unresolved row (row 8 — STATE.md's own
words: needs "an actual CE/founder call, not another self-trace") and a standing
instruction to dispatch a fresh, independent `M17-V1` re-run that was never
dispatched. Every one of the 30 engine-hardening commits since then — the entire
freeze program this plan is built on — happened with zero `STATE.md` entries and zero
`MXX-Cx` cards, outside the EEOS process this repository otherwise runs on (see
`EEOS.md`). Procedurally this is unresolved; functionally it is likely moot —
engine-hardening's own frozen-surface re-check (§1 table, "Frozen G1 surfaces
unchanged") independently re-establishes what M17-V1 row 8 was trying to test — but
nobody has written that down as the row's disposition, and 60 commits of real
engineering exist nowhere in the ledger.

**Why this matters for the GUI transition, not just for git hygiene.** "Build the API
layer against the engine" is ambiguous until someone names the commit. A GUI/API
effort branched from `main` today, in good faith, inherits every defect this
30-commit program just closed — including the in-memory-state-loss bug. That is a
materially worse starting point than anything in the blocker list at §2.

**Recommended action (Phase A, process only, ~zero engineering time, does not touch
G0–G2's scope).** Before GUI-enabling code is written: (1) founder executes the
merge sequence — correct M10–M14's `STATE.md` rows to `DONE-MILESTONES` to match
reality, then review and merge M15→M16→M17→`engine-hardening` into `main` in that
order (or explicitly declare `engine-hardening` the new base and retire the
stacked-branch ledger for it); (2) update `STATE.md` to match whichever is chosen,
including a real disposition for M17 row 8 — adjudicate it, or record it superseded
by the freeze report's independent frozen-surface re-check; (3) scope GUI work
against one named commit on `main`, not an implicit "the engine."

---

## 2. The real blocker list

The founder's brief lists five known blockers. After tracing the code, the list is
six, and their weights are not what the brief assumes.

| # | Blocker | Real severity | Why |
|---|---|---|---|
| **B-a** | Non-evented state transitions | **CRITICAL — the actual blocker** | The event stream cannot represent `waiting`. Fixing the cursor without fixing this ships a stream that lies. |
| **B-b** | No global event cursor | **HIGH** | Real, but subordinate to B-a — they are one migration. |
| **B-c** | No definition → YAML serializer | **HIGH** | Blocks the editor's save path only. Read-only canvas is unblocked. |
| **B-d** | Namespace `"default"` overloading | **HIGH — and it silently returns wrong answers today** | Not merely undecided; `StepStats` returns all-zeros with no error when the namespace does not literally match (§6.4). |
| **B-e** | Unbounded read paths | **MEDIUM** | B-17 bounded the *instance list* only. `ReadEvents`, `QueryHistory`, `ReplayInstance` remain uncapped. |
| **B-f** | No authn/authz whatsoever | **MEDIUM for V1, CRITICAL before multi-user** | Zero user/role/tenant/session concept anywhere. |
| — | Live-provider validation | **LOW for GUI** | Real gap, but it blocks *shipping AI features*, not *building the GUI*. |

### 2.1 Blocker B-a in detail — the finding that reorders the plan

`internal/storage/rebuild.go:218-224` is the tell:

```go
// a wait record must NOT resurrect an instance the event stream shows as TERMINAL.
status := r.status
if !isTerminalStatus(core.InstanceStatus(status)) && waitingInstances[r.instanceID] {
    status = string(core.InstanceStatusWaiting)
}
```

`RebuildState` replays the EventLog, then **overlays** `waiting` from the
`wait_records` side table, because replay alone cannot produce it. The engine's own
comment says so (`internal/engine/signal.go:149-153`): *"This is a non-evented
projection write (EDR-007 §9): `waiting` is an engine tick-time decision, not an
event, so a cold RebuildState reconstructs `running`."* There is a passing regression
test that pins this behaviour (`rebuild_nonevented_test.go`).

Three transitions are invisible to the event log:

| Transition | Where | Consequence for a live GUI |
|---|---|---|
| `running → waiting` | `signal.go:164-168` | An instance parks; the stream goes silent. The UI shows it as still running, forever. |
| `waiting → running` on timeout `continue` | `signal_timeout.go:128-140` | Resumption is indistinguishable from a normal signal delivery. |
| cancellation *requested* (vs. applied) | `cancel.go:118` | A user clicks Cancel; nothing appears in the stream until the instance actually terminates. |

**This is not a defect.** It is a deliberate, documented design choice tied to the
closed 12-event vocabulary frozen at G1. But it means the sentence "AWIS is event
sourced" is true of its *history* and false of its *live state*, and a GUI built on
the former assumption will be wrong in the most visible way possible: the running
workflows list.

---

## 3. Event sourcing sufficiency

Assessed against the seven uses the brief names.

| Use | Verdict | Blocking change |
|---|---|---|
| Realtime UI | **INSUFFICIENT** | B-a + B-b (one migration, §4) |
| Workflow canvas | **SUFFICIENT** (definitions, not events) | B-c for save; layout has a home already (§7.2) |
| Execution history | **SUFFICIENT** | none |
| Observability | **SUFFICIENT-WITH-CHANGE** | `StepStats` rescans a 10-year window per call (`sdk/runtime_recall.go:112-119`) |
| Debugging | **SUFFICIENT** | `ReplayInstance` returns the full ordered log |
| Time-travel replay | **SUFFICIENT-WITH-CAVEAT** | pure-log replay cannot reconstruct `waiting`/`cancellation_requested`; auxiliary tables are required and must be documented as part of the contract |
| AI workflow generation | **SUFFICIENT-WITH-CHANGE** | validator is excellent; its structured codes are discarded at the CLI boundary (§9) |

### 3.1 Runtime proof

I ran a real daemon, submitted three `hello-world` instances and one `with-signal`
instance, and read the database directly.

Global ordering — `rowid` tracks true emission order across interleaved instances,
while `sequence_num` restarts per instance:

```
rowid | instance | event_type       | sequence_num
    1 | d85d6cb8 | WorkflowStarted  | 1
    2 | a8e96503 | WorkflowStarted  | 1
    3 | 358edff5 | WorkflowStarted  | 1
    4 | d85d6cb8 | StepStarted      | 2
    ...
   19 | 2ebb775f | WorkflowStarted  | 1
   22 | 2ebb775f | StepStarted      | 4     ← last event for this instance
```

And the corresponding state:

```
instance 2ebb775f  status=waiting  current_steps=["await-approval"]
wait_records: 2ebb775f | await-approval | approved | 2026-09-01T… | cancel
```

**The instance is `waiting`. Its last event is `StepStarted`.** Nothing in the log
says it parked. That is blocker B-a, observed rather than argued.

### 3.2 Why `rowid` is not the answer

`execution_events` has no `WITHOUT ROWID` clause, so the implicit `rowid` exists, and
nothing in the codebase ever deletes from the table — `prune-events` is dry-run-only
in V1 (`cmd/awis/prune.go:39-44`) and the only real `DELETE`s target `domain_events`
and the FTS index. So `rowid` is monotonic and gap-free *today*.

It is still the wrong choice, for the reason the freeze report already identified:
`VACUUM` renumbers rowids on tables without an `INTEGER PRIMARY KEY`. No code runs
`VACUUM` today (grep confirms), but a cursor whose correctness depends on an operator
never running a routine maintenance command is a latent data-corruption bug, not a
design. **Use an explicit column.**

---

## 4. The one migration that unblocks the live GUI

**Decision D1 — recommended: add an append-only `state_changes` table, not a
`global_seq` column on `execution_events`.**

The freeze report recommends adding `global_seq` to `execution_events`, backfilled
from `rowid`. That is a reasonable design and it solves B-b. It does **not** solve
B-a — the stream would still be silent when an instance parks. And it is the more
invasive of the two options: SQLite cannot `ALTER TABLE ADD COLUMN … AUTOINCREMENT`,
so it requires a create-copy-rename rebuild of the append-only table that is the
platform's source of truth.

The alternative solves both blockers and never touches `execution_events`:

```sql
-- migration 0007
CREATE TABLE state_changes (
  seq          INTEGER PRIMARY KEY AUTOINCREMENT,  -- the global cursor
  instance_id  TEXT NOT NULL,
  namespace    TEXT NOT NULL,
  kind         TEXT NOT NULL,     -- 'event' | 'projection'
  event_id     TEXT,              -- set when kind='event'
  status       TEXT NOT NULL,     -- instance status after the change
  current_steps TEXT NOT NULL,    -- JSON
  changed_at   TEXT NOT NULL
);
CREATE INDEX idx_state_changes_ns ON state_changes(namespace, seq);
```

One row is written inside the **existing** transaction of `AppendEvent`
(`sqlite.go:94-158`) and of `UpsertInstance` (`sqlite.go:428+`). Both already open
their own transaction, so this adds no new transaction and no new lock.

Why this is the right call:

- **It captures every transition, evented or not.** `enterWait`, timeout-`continue`
  and cancellation-requested all flow through `UpsertInstance`, so all three become
  visible to a tailing client for the first time.
- **It does not touch the G1 freeze.** The 12 event types, the event payload field
  names, and the 12 `StoragePort` methods are all unchanged. This is an additive
  migration plus one new method reached through a locally-declared interface and a
  type assertion — precisely the pattern `cancellationStore`, `signalWaitStore`,
  `PluginStore` and `signal.Store` already establish, and which the freeze report
  identifies as the sanctioned route for additive change.
- **`AUTOINCREMENT` is immune to `VACUUM`.** SQLite keeps the counter in
  `sqlite_sequence`. The `audit_log` table already uses exactly this pattern
  (`migrations/0004_audit.sql`), so there is in-repo precedent.
- **It is cheap to revert.** Dropping an additive table is not a format decision.

Cost estimate: one migration file, ~40 lines in `sqlite.go`, one `ReadChangesSince`
method, one SDK wrapper, and a regression test proving a parked instance produces a
`state_changes` row. **One to two days.**

The remaining question — whether a client also needs a globally-ordered *event*
stream (as opposed to a change stream plus per-instance `ReadEvents` on demand) — I
recommend deferring. The change stream tells the UI *what to re-fetch*; the existing
`ReadEvents(instance, fromSeq)` is already indexed and correct for fetching it. Adding
`global_seq` to `execution_events` can be done later if a genuine firehose need
appears, and it will be cheaper to justify then.

---

## 5. Engine freeze checklist (deliverable A)

Ordered. Items 1–3 are the freeze gate; 4–6 are the GUI gate.

| # | Item | Size | Blocks |
|---|---|---|---|
| 1 | ~~**B-31**: validate `timeout_action`; fix both scaffold copies~~ **DONE 2026-08-30** — plus four adjacent enum rules, see §1.2 | — | freeze |
| 2 | **D4**: decide the live-intelligence question — gate it or ship it declared-unverified | decision | freeze |
| 3 | Delete or rebuild the stale repo-root `awis` binary; add a `make` target so live audits cannot use a stale one | 1h | hygiene |
| 3b | Fix the `TestSystemRehearsalInitStartSubmitTrace` deadline flake (R14); prune 6 stale `worktree-agent-*` branches; correct the "29 commits" figure | 2h | gate credibility |
| 4 | **D1 + migration 0007**: `state_changes` table, written in both existing write transactions | 1–2d | GUI live view |
| 5 | **D2**: resolve namespace semantics; unify the three duplicated wildcard implementations; give `ReadEventRange` a wildcard branch | 1–2d | GUI correctness |
| 6 | **B-c**: definition → YAML serializer + round-trip test | 2–3d | GUI editor save |

Not on the critical path, and deliberately so: pagination for events (§8.4), authz
(§10), Postgres (§11.3), live-provider gate (§9.3).

---

## 6. What a GUI needs from the engine (deliverables B + D)

### 6.1 Coverage of the twelve operations an API layer needs

| Operation | Available today | Via |
|---|---|---|
| Submit | ✅ | `Runtime.Submit` |
| Signal | ✅ | `Runtime.Signal` |
| Cancel | ⚠️ partial | `Runtime.Cancel` hardcodes `compensate=false` (`sdk/runtime_runner.go:127-129`); the CLI bypasses the SDK for `--compensate` |
| List instances, paged | ✅ | `Runtime.ListPaged` — a real cursor, page 100 / max 1000 |
| Instance detail | ⚠️ partial | `Runtime.Status` omits `signal_name`/`timeout_remaining_s`; the CLI builds those with a local type assertion |
| Instance event history | ✅ | `Runtime.Recall().ReplayInstance` |
| Query history | ✅ | `Runtime.QueryHistory` |
| Step statistics | ⚠️ correctness caveat | `Runtime.StepStats` — see §6.4 |
| List definitions | ⚠️ no wrapper | `sdk.StoragePort.ListWorkflows` directly |
| Get definition | ⚠️ no wrapper | `sdk.StoragePort.GetWorkflow` directly |
| Register definition | ✅ register-only | `Runtime.RegisterWorkflow`; no update path — new version required |
| List plugins / triggers / health | ❌ missing | §6.2 |

### 6.2 The module boundary — a hard architectural constraint

I compiled a throwaway external Go module against this repo to settle this
empirically. The result splits cleanly:

**Reachable out-of-module** (builds clean): `sdk.SQLiteStorage`, `sdk.NewRuntime`,
`ListPaged`, `Status`, `Submit`, `Signal`, `Cancel`, `Recall().ReplayInstance`, and —
because it takes only stdlib types — `RebuildState` via a locally-declared interface.

**Hard-blocked**: anything returning an `internal/storage` row type. `go build` fails
with `use of internal package github.com/awis/awis/internal/storage not allowed`.
That covers `ListPlugins` → `[]PluginRow`, `SearchEvents` → `[]RecallRow`,
`ListAudit` → `[]AuditRow`, `ListWaitRecordsByInstance` → `[]WaitRecord`. An external
module cannot even *declare* the interface to type-assert against, because it cannot
name the return type.

**Decision D3 — recommended: build the GUI backend inside this module**, as
`cmd/awis-server/` with handlers in `internal/api/`.

This is not a workaround, it is the correct call for V1. It costs nothing, unblocks
plugin/recall/audit/wait-record surfaces immediately, and — critically — it means you
do **not** have to design and freeze a public HTTP/SDK contract before you know what
the GUI actually needs. Promoting the four row types into `sdk` is a one-day change
you can make later, once a second consumer exists to justify the shape. Designing
that surface now, speculatively, is how you get a public API you regret.

### 6.3 `CLI_CONTRACT.md` is not a usable spec

The contract doc and the binary disagree on at least eleven points: `export` is
documented with `--format/--from/--to` flags that do not exist and an output schema
that bears no resemblance to what it emits; `metrics` and `recall` have entirely
different JSON field sets than documented; `start --config` does not exist;
`recall --namespace` and `rebuild-state --namespace` do not exist; `status --watch`
polls at 1s not the documented 5s; `trace`'s `trigger` field is declared and never
assigned, so it is always `null`.

**Do not build the GUI against this document.** Build against the binary, and pin the
result with golden tests — which brings up the deeper issue: the G1/M08 freeze is
asserted in code comments and enforced by nothing. There is no golden-snapshot test
on `ExecutionEvent` serialization and no reflective test pinning `StoragePort` at 12
methods. Several `internal/core/ports.go` comments still read "mutable until M08"
though M08 merged long ago. If the GUI is going to depend on these shapes, the freeze
needs teeth: add snapshot tests as part of GUI milestone G1.

### 6.4 Namespace — the decision, and why it is urgent (D2)

`"default"` is overloaded. `namespacePredicate` (`cmd/awis/export.go:158-163`) maps
the literal string `"default"` to `""`, and `""` at the storage layer means *no
predicate — every namespace*. So on the read path `--namespace default` means
**"all namespaces"**; on the write path `submit` passes `"default"` through to
`sdk.Config.Namespace` as a **literal name**.

It is worse than a single inconsistency. The same rule is reimplemented three times
independently — `export.go:158` via a helper, `history.go:75` inline, `metrics.go:86`
inline with a different condition — while `status` (`status.go:63`) and
`workflow list/show` define their own flags defaulting to `""` and never special-case
`"default"` at all. `recall` has no namespace concept whatsoever; `SearchEvents` takes
no namespace parameter, so FTS is globally unscoped regardless of any flag.

**And one path returns silently wrong data.** `ReadEventRange`
(`internal/storage/sqlite.go:182-192`) has no wildcard branch — `namespace` is always
a literal match, even when empty. `StepStats` calls it with `Runtime.namespace`
(`sdk/runtime_recall.go:112-119`). If that does not literally equal the events'
namespace column, **`StepStats` returns all-zero statistics and no error.** A GUI
metrics dashboard built on it would render an empty chart and report success. In my
own runtime test the instances landed in namespace `examples` (stamped from the
workflow YAML, not from any flag) while the runtime default was `"default"` — the
exact configuration that triggers this.

Recommendation: **option 2 — retire the overload.** Make `""` the wildcard, make
`"default"` an ordinary namespace name everywhere, and add an explicit
`--all-namespaces` flag. Then give `ReadEventRange` a wildcard branch and delete the
three duplicated implementations in favour of one helper.

This is the more disruptive option — it changes what `history`, `metrics` and
`export` return for existing projects, and the B-26 commit message flags exactly that
risk. I recommend it anyway, and specifically recommend taking the breakage **now**,
before a GUI exists: a wildcard sentinel that collides with a real namespace name is
a bug that gets more expensive every month it survives, and a GUI will encode the
ambiguity into URLs and saved filters where it becomes permanent.

Also worth correcting in the docs: `awis submit --namespace X` does not put the
instance in namespace X. `engine.Submit` stamps `def.Namespace` from the workflow's
own YAML (`internal/engine/submit.go:50`). The flag only influences definition
lookup preference.

---

## 7. GUI architecture (deliverable C)

### 7.1 Shape

```
┌────────────────────────────────────────────┐
│  Browser SPA                                │
│  canvas · run list · run detail · editor    │
└───────────────┬────────────────────────────┘
                │ JSON over HTTP  +  SSE
┌───────────────▼────────────────────────────┐
│  cmd/awis-server   (IN-MODULE, decision D3) │
│  internal/api/     handlers, DTOs, SSE hub  │
└───────────────┬────────────────────────────┘
                │ direct Go calls
┌───────────────▼────────────────────────────┐
│  sdk.Runtime  +  internal/storage           │
│  engine embedded via rt.Start(ctx)          │
└───────────────┬────────────────────────────┘
                │
        SQLite (WAL, single writer)
```

**One process, engine embedded.** `sdk.NewRuntime` starts no goroutines and installs
no signal handlers; `internal/engine` and `sdk` contain zero `os.Exit` and zero
`signal.Notify` — those live only in `cmd/awis`. So the server can own the engine
lifecycle with its own context. `rt.Start(ctx)` blocks; run it in a goroutine.

**Exactly one engine may tick a given database.** The engine keeps per-instance state
in memory — retry schedules, pending activations, terminally-failed markers, live
waits (`engine.go:77-85`) — reconstructed per-process by `hydrate` on first touch.
Two engines against one file is undocumented territory. Multi-*process* SQLite access
is safe for the *storage* operations (WAL, `busy_timeout=5000`, `_txlock=immediate`,
and `ClaimStep`'s unique-constraint gate), which is why `awis submit` from a second
process works. But "safe to submit from another process" is not "safe to run a second
engine." **Run `awis-server` instead of `awis start`, not alongside it.**

### 7.2 Canvas layout persistence — no schema change needed

`core.Step` has no metadata field and the schema is G1-frozen. But
`WorkflowDefinition.Metadata map[string]any` exists (`workflow.go:37`), is parsed
from YAML (`internal/dsl/dsl.go:49`), and is explicitly "application-defined data."

Store node positions there, keyed by step id:

```yaml
metadata:
  ui:
    positions:
      fetch:   {x: 120, y: 80}
      approve: {x: 340, y: 80}
```

Zero schema change, zero founder sign-off, works today. A first-class `Step.UI` field
would be cleaner and follows the ADJ-5/6/7 additive-amendment precedent — but that
needs G1 sign-off, and it is not worth spending that on layout before the editor has
proven its shape. Revisit at GUI milestone G5.

### 7.3 Streaming (deliverable E)

**SSE, not WebSocket.** The traffic is one-directional (server → client); mutations
go over ordinary POSTs. SSE gives reconnection and `Last-Event-ID` for free, works
through proxies, and needs no dependency beyond stdlib `net/http`. Given the repo has
exactly two direct dependencies (`yaml.v3`, `modernc.org/sqlite`, both pure-Go, no
cgo), keeping the server dependency-light is worth protecting.

```
GET /api/v1/stream?since=<seq>&namespace=<ns>
  → text/event-stream, id: <seq>
```

The handler tails `state_changes` from `since`, emits one event per row, and the
client resumes with `Last-Event-ID` after a drop. Poll interval on the server side
should track the engine tick (100ms default, `--tick`); there is no notification
mechanism to hook — the engine is pull-based with no channel, condvar, or DB trigger
anywhere, so every state change is visible within one tick and no faster.

Be honest about latency in the UI: **every transition is tick-quantized.** Submit →
first `StepStarted`, signal → resumption, timeout → routing, cancel → effect are all
bounded by one `TickInterval`. Cancel is additionally *cooperative* — an in-flight
step is not interrupted, so a "Cancel" button must read as "requested," not "stopped."

---

## 8. API layer (deliverable D)

Versioned under `/api/v1`. All in-module, so nothing is blocked.

| Method | Path | Backed by |
|---|---|---|
| GET | `/workflows` | `StoragePort.ListWorkflows` |
| GET | `/workflows/{id}/{version}` | `StoragePort.GetWorkflow` |
| POST | `/workflows` | `Runtime.RegisterWorkflow` (new version only) |
| POST | `/workflows/validate` | `validate.Validate` — returns structured `Issue`s |
| GET | `/instances` | `Runtime.ListPaged` |
| GET | `/instances/{id}` | `Runtime.Status` + wait-record lookup |
| GET | `/instances/{id}/events` | `StoragePort.ReadEvents` — **add a limit, §8.4** |
| POST | `/instances` | `Runtime.Submit` |
| POST | `/instances/{id}/signal` | `Runtime.Signal` |
| POST | `/instances/{id}/cancel` | `engine.Cancel` — expose `compensate` |
| GET | `/stats/steps` | `Runtime.StepStats` — **after D2** |
| GET | `/plugins` | `storage.ListPlugins` (in-module only) |
| GET | `/stream` | `state_changes` tail (SSE) |
| GET | `/healthz` | new |

### 8.4 Pagination debt

B-17 bounded the instance list. It did not touch the others:

| API | Bounded? |
|---|---|
| `ListInstancesPaged` / `Runtime.ListPaged` | ✅ cursor, 100 default / 1000 max |
| `ListInstances` | ❌ unbounded *by design* — the engine tick needs every running instance |
| `ReadEvents` | ❌ unbounded |
| `ReadEventRange` | ❌ unbounded |
| `QueryHistory` | ❌ unbounded (delegates to `ListInstances`, filters in memory) |
| `ReplayInstance` | ❌ unbounded |

For V1 volumes this is fine. Before the GUI is exposed to a real workload, add a
limit+cursor to `ReadEvents` — a single instance with 100k events will otherwise
return all of them in one query to render one timeline. `StepStats`'s 10-year
full-namespace rescan per call should become an incremental aggregate fed from the
same `state_changes` cursor.

### 8.5 Error model

Partially mappable. `ErrInstanceNotFound`, `ErrPluginNotFound`, `ErrCapabilityNotFound`,
`ErrDuplicateDomainEvent`, `ErrVersionConflict`, `ErrPaginationUnsupported` are real
sentinels → clean 404/409/501. But `GetWorkflow`'s not-found is a bare
`fmt.Errorf(... "not found")` with no sentinel, and `cmd/awis/start.go` string-matches
`"already registered"` rather than asserting `*RegistrationError`. Add a sentinel for
`GetWorkflow` and fix that string match as part of GUI milestone G1 — a GUI needs to
distinguish 404 from 500 on the single most-called read path.

---

## 9. Workflow editor and AI features (deliverables F + G)

### 9.1 Edge types the canvas must render

The graph is not one edge list. A canvas has to render six distinct relationships,
only two of which live in `Transitions`:

| Relationship | Where it lives |
|---|---|
| Unconditional transition | `Transition{From,To}` |
| Conditional transition | `Transition.Condition` |
| Error transition | `Transition.OnError` — same struct, boolean discriminator |
| Fallback | `Step.Fallback` — on the *step*, not in `Transitions` |
| Compensation | `WorkflowDefinition.Compensation` — a separate ordered list, run in reverse |
| Retry / signal-timeout | not edges at all — step-local policy, render as badges |

One asymmetry to mirror exactly: fallback edges count for **reachability** but are
**excluded from cycle detection** (`internal/validate/validate.go:26-28`). A canvas
that includes them in its own cycle check will reject graphs the engine accepts.

### 9.2 What the editor gets for free, and what it does not

**Free:** `validate.Validate(def) []Issue` is pure — no storage, no runtime, no
handler registry — and returns structured `{Code, StepID, Field, Message, Position}`.
That is exactly the shape for live inline diagnostics. `expr.ParseTemplate` and
`expr.ParseCondition` return `*ParseError{Position, Msg}` with in-bounds byte offsets
for squiggles.

**Not free, and worth knowing before you scope:**
- **No serializer** (B-c). `internal/dsl` exports `Parse`, `ParseFile`, `ValidateFile`,
  `Discover` — and nothing else. `render.go` renders a *validation report*, not YAML.
  Repo-wide, `yaml.Marshal` is called zero times. Build the emitter against the same
  `yaml:` struct tags the parser uses, and add the round-trip test that does not exist
  today.
- **No variable enumeration for autocomplete.** `expr.Env` resolves values at runtime;
  nothing walks the graph to answer "what is in scope at step X." The editor must
  derive this from upstream steps' declared `Outputs`.
- **No runtime introspection for the node palette.** `engine.runners`,
  `NativeRunner.handlers`, `Runtime.handlers` and `plugin.Manager` are all unexported
  with no `List*` accessor. The five `StepType` kinds are compile-time constants and
  can be hardcoded, but "which native handlers / plugin capabilities are actually
  registered in *this* deployment" cannot be queried. Add three small accessors — this
  is the difference between a palette that reflects reality and one that drifts.
- **A naming trap.** `Step.Inputs` is typed `InputSchema` and commented as a schema,
  but per `WORKFLOW_SCHEMA.md:112-116` (CONTRA-10) it is an execution-time *template
  value map*. An editor that builds a JSON-Schema editor from the field name will build
  the wrong UI.

### 9.3 AI-assisted generation

The repair loop — generate → validate → structured errors → retry — is genuinely
well-served, because `validate.Validate` already returns stable machine codes
(`CodeCycle`, `CodeFallbackUndefined`, `CodeIntelligenceContextBudget`, ~15 in all).
Two gaps:

1. **The CLI throws the codes away.** `awis workflow validate --json` emits only
   `line` + `message`, dropping `Code` and `Field` (`cmd/awis/workflow.go:43-51`).
   The API layer should surface the full `Issue`. Low-risk, high-value.
2. **There is no machine-readable schema artifact.** No `*.schema.json` exists. An LLM
   must be prompted from `WORKFLOW_SCHEMA.md` or the Go structs. Generating a JSON
   Schema from `core.WorkflowDefinition` is a half-day and makes generation markedly
   more reliable.

**Model IDs are stale, and this is a live-fire issue (D4).** The adapter pins
`modelQuality = "claude-sonnet-4-5"` and `modelFast = "claude-haiku-4-5"`
(`internal/intelligence/adapters/anthropic/anthropic.go:32-37`). Checked against the
current model table: `claude-haiku-4-5` is current; **`claude-sonnet-4-5` is
superseded** — the current Sonnet-tier ID is `claude-sonnet-5`. `Draft` and
`Synthesize` — every call an AI generation feature would make — use the stale one.

This is precisely the failure the missing live gate cannot catch. `TestLiveSmoke`
(`live_test.go:27-83`) is gated on `ANTHROPIC_LIVE=1` plus an API key; neither is set
in `.github/workflows/ci.yml`, the `Makefile`, or anywhere else. Every Anthropic test
that runs in CI is served by an `httptest` fake encoding the implementer's own
assumptions — a fake cannot reject an invalid model ID. **The Anthropic integration
has never executed against the real API in any automated gate.**

Fix: update `modelQuality` to `claude-sonnet-5`, and add a separate CI job (not in
`make verify`, since it needs a secret and costs money) running `TestLiveSmoke` on a
schedule, with an assertion on the response's `model` field so ID drift fails loudly.

Also note for cost dashboards: per-step `{adapter, model, tokens_used}` **is**
persisted into the `StepCompleted` payload (`emit.go:43-51`), verified end-to-end by
`TestIntelligence_SuccessADJ8Payload`. But `payload` is opaque JSON TEXT with no
`json_extract`, no generated column, no index on `event_type`, and there is no cost
computation anywhere. The data is complete; the query layer is entirely unbuilt.

---

## 10. Security posture

**There is no authorization model.** Grep for tenant/role/user/session/permission
returns zero AWIS-code hits. There is no identity, no session, no per-namespace access
control, no API-key issuance. Credential handling (`config show/set` masking, fail-closed
by substring since `0ddf1ce`) is hygiene for a single local operator, not multi-tenancy.

For a single-user local GUI, bind to `127.0.0.1` and ship. **Before the first remote
deployment, authz is a from-zero build** — budget it as a milestone, not a task.

Two runner-isolation facts to know before exposing workflow authoring to users who are
not the operator: plugins get a scrubbed environment (manifest `env` + `PATH` only,
`manager.go:772-782`), but **subprocess steps inherit the full parent environment** —
`internal/runner/subprocess/subprocess.go` never sets `cmd.Env`, so a subprocess step
sees `ANTHROPIC_API_KEY`. Neither runner has an OS sandbox: no seccomp, namespaces,
chroot, or cgroups anywhere. A workflow author has arbitrary file and network access as
the daemon user. That is acceptable for a local single-user tool and unacceptable the
moment the GUI lets someone else author workflows.

---

## 11. Risk register

| ID | Risk | Severity | Mitigation |
|---|---|---|---|
| R1 | Live stream silently omits `waiting` | **Critical** | D1 / migration 0007 (§4) |
| R2 | `StepStats` returns zeros, not an error, on namespace mismatch | **Critical** | D2; make the wildcard explicit |
| R3 | Building the GUI against `CLI_CONTRACT.md` | High | Build against the binary; add golden tests |
| R4 | G1 freeze is comment-only, unenforced | High | Snapshot + reflective tests at G1 |
| R5 | ~~Stale `claude-sonnet-4-5` in the quality path~~ **CLOSED** — `modelQuality` is now `claude-sonnet-5`. The *live CI gate* (D4) remains OPEN, and is what would have caught this. | Medium (was High) | D4 still required |
| R6 | ~~B-31 scaffold ships an invalid `timeout_action`~~ **CLOSED** — scaffold, examples, OIP workflows and DSL doc corrected; five validator rules added | — | done |
| R18 | **Upgrade break (disclosed):** a workflow definition registered before 2026-08-30 containing an illegal enum value is now REJECTED at `Submit`. In-flight parked instances are unaffected. | Medium | Ship an upgrade note; `awis workflow validate` names the fix |
| R7 | Two engines on one DB | Medium | Document; add a PID/lock guard |
| R8 | Unbounded `ReadEvents` at scale | Medium | Add limit+cursor before real load |
| R9 | No authz | Medium now / Critical later | Loopback-bind V1; milestone before remote |
| R10 | `webhook` trigger type is a dead enum | Low | Hide in the UI or implement with the API layer |
| R11 | `pending` status is unreachable | Low | Remove, or document as reserved |
| R12 | SQLite lock-in (FTS5, pragmas, `AUTOINCREMENT`, string-matched constraint errors) | Low for V1 | Postgres is aspirational; no adapter exists |
| R13 | Subprocess steps inherit the full env incl. API keys | Low now / High multi-user | Scrub env as the plugin manager does |
| R14 | `TestSystemRehearsalInitStartSubmitTrace` flakes under full-suite contention (hardcoded 10s deadline) | Medium | Raise/derive the deadline. It sits in `cmd/awis` — the package the GUI's contract tests will live beside, so a flaky neighbour will erode trust in the new gates. |
| R15 | `cmd/awis` coverage is 47.1%, the lowest of any core package | Medium | This is exactly the CLI/JSON-contract surface the GUI must not regress. Raise it as part of G1. |
| R16 | Two weak tests assert nothing meaningful: `TestRecallSearchEventsDefaultLimit` (`recall_test.go:203`, contains `_ = results // empty is fine`) and `TestRemovePluginIsIdempotent` (`plugin_remove_test.go:90`, checks only `err == nil` twice) | Low | Strengthen or rename; as written their names overstate what they prove. |
| R17 | 6 stale unmerged `worktree-agent-*` branches | Low | Prune before release housekeeping. |

---

## 12. Founder decisions required

| ID | Decision | Recommendation |
|---|---|---|
| **D1** | Global cursor: `global_seq` on `execution_events`, or an additive `state_changes` table? | **`state_changes`.** Solves the cursor *and* the non-evented transitions, never touches the frozen append-only table, `VACUUM`-proof, in-repo precedent. |
| **D2** | Namespace: keep `"default"` as the wildcard sentinel, or retire the overload? | **Retire it.** `""`/`--all-namespaces` as wildcard, `"default"` an ordinary name. Take the breakage now, before a GUI encodes it. |
| **D3** | GUI backend in-module or a separate service? | **In-module** (`cmd/awis-server/`). Unblocks plugin/recall/audit surfaces immediately; defers public-API design until a second consumer justifies it. |
| **D4** | Live intelligence: gate it, or ship declared-unverified? | **Gate it** — a scheduled, secret-gated CI job outside `make verify`, asserting the response `model`. The stale Sonnet ID is proof the fake-server tests cannot catch this class. |

---

## 13. Milestone roadmap (deliverable H)

| ID | Milestone | Contents | Est. | Depends on |
|---|---|---|---|---|
| **G0** | Freeze close-out | B-31; stale binary; D4 decision | 1d | — |
| **G1** | Contract hardening | Golden JSON snapshots; `StoragePort` reflective pin; `GetWorkflow` sentinel; correct `CLI_CONTRACT.md` | 2–3d | G0 |
| **G2** | Cursor + eventing | Migration 0007; `state_changes` writes in both existing transactions; `ReadChangesSince`; SDK wrapper; parked-instance regression test | 2d | D1 |
| **G3** | Namespace resolution | Unify the three duplicated rules; `ReadEventRange` wildcard; `--all-namespaces`; migration notes | 2d | D2 |
| **G4** | API server | `cmd/awis-server`, `internal/api`, all `/api/v1` routes, SSE stream, `healthz`, loopback bind | 5–7d | G2, G3, D3 |
| **G5** | Read-only GUI | Run list, run detail, live timeline, event inspector, read-only canvas | 8–10d | G4 |
| **G6** | Serializer + editor | Definition→YAML + round-trip test; canvas editing; layout in `Metadata.ui`; live validation; expression linting | 8–12d | G5, B-c |
| **G7** | Palette + introspection | `List*` accessors for handlers/plugins/capabilities; node palette from live registry | 3–4d | G6 |
| **G8** | Observability | Incremental step stats off the change cursor; token/cost dashboard via `json_extract` + index | 4–5d | G4 |
| **G9** | AI generation | JSON Schema artifact; structured-issue repair loop; NL→workflow | 5–8d | G6, D4 |
| **G10** | Multi-user | Authn/authz, per-namespace ACLs, subprocess env scrubbing, remote bind | 10–15d | G5 |

**Critical path to a demonstrable live GUI: G0 → G2 → G4 → G5.** Roughly three weeks,
and G3 can run in parallel.

### 13.1 Paths

- **Cheapest** — skip G2; poll `ListPaged` every second. Delivers a working list-and-detail GUI in ~2 weeks with zero engine change. Ceiling: no live timeline, and load grows with instance count.
- **Fastest to something on screen** — G4 read-only against fixtures while G2 lands. The frontend is fully parallelizable once the JSON contract is pinned, which is what G1 is for.
- **Safest** — full G0→G1→G2→G3 before any UI code. Adds ~1 week, and closes the two silent-wrong-answer defects (R1, R2) before anything depends on them.
- **Recommended** — **G0 → G1 → G2 in series (≈5 days), then G3 and G4 in parallel, then G5.** It front-loads the two changes that are cheap now and expensive after a GUI exists (the cursor and the namespace semantics), pins the contract before anyone builds on it, and still puts a live GUI on screen inside a month. The cheapest path is a false economy: polling is not much less work than the SSE handler, and it must be thrown away.

---

## 14. Answering the brief directly

**1. Work remaining before engine freeze.** B-31, the D4 decision, the stale binary,
and the flaky system test. Everything else the program claimed is confirmed —
including, on a live-revert sample, that the regression tests are load-bearing.

**2. Work remaining before GUI can begin.** Nothing, for the frontend and history
views — those can start tomorrow. For the live view: migration 0007 (G2). For the
editor's save path: the serializer (G6).

**3. Can the GUI be developed independently afterwards?** Yes, with one constraint:
in-module (D3). After G2 and G3, the engine surface is stable enough that GUI work
does not require further engine changes until G7 (introspection accessors) and G10
(authz) — both additive.

**4. Hidden architectural blockers.** Three, none in the brief's original list: the
`internal/` module boundary (§6.2); the silent `StepStats` namespace failure (§6.4);
and the `CLI_CONTRACT.md` divergence combined with a comment-only freeze (§6.3).

**5. Missing engine APIs.** Plugin/trigger/health enumeration; handler and capability
introspection; cancel-with-compensate on the SDK; wait-record detail on `Status`;
definition list/get wrappers.

**6. Would any future GUI requirement force an engine redesign?** No. Every gap found
is additive — a new table, a new method behind a type assertion, a new accessor. The
G1-frozen surfaces (12 event types, 12 `StoragePort` methods, payload field names,
`StepError`) do not need to change, and the recommended D1 design is chosen
specifically to keep it that way. The one thing that *would* force a redesign is
sub-tick latency: the engine is pull-based with no notification path anywhere, so
"instant" UI feedback is not achievable without reworking the tick loop. Don't promise
it; 100ms is good enough.

**7. Is event sourcing sufficient?** For history, replay, debugging and the canvas:
yes. For realtime UI and observability: not as it stands — see §3. The honest framing
is that AWIS is event-sourced for its *history* and projection-based for its *live
state*, and the GUI must be built on that truth rather than on the aspiration.

---

*Prepared by supervised repo-wide review. Sub-investigations covered the event model,
storage and concurrency, the runtime, the DSL, the CLI/SDK surface, the intelligence
and plugin paths, and an adversarial audit of the reports against code. That audit
independently re-verified two fixes (B-4, B-30) by reverting them and observing the
named tests fail with the documented symptom.*

*One inventory pass returned fabricated symbol tables — it misstated the event-type
enum, the `StoragePort` method set, and the instance-status values. It was discarded;
only its schema, migration, CLI and dependency catalogs were retained, and only where
they matched direct inspection. This is recorded because it is the same failure mode
the evidence hierarchy exists to catch: a confident secondary source contradicting the
code. Runtime claims here were verified against a live daemon and a freshly built
binary, never against the stale one in the repo root.*
