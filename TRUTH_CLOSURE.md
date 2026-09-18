# AWIS — Truth Closure

**Status:** CANONICAL. Supersedes the audit corpus listed in §7.
**Established:** 2026-09-09, against `engine-hardening` @ `8a87f70`, toolchains go1.26.4/.5/.6.
**Method:** every major claim in the 24-document corpus re-tested against running code.

Evidence hierarchy used throughout, highest first:
**T1** runtime execution · **T2** repository source · **T3** automated tests · **T4** git history · **T5** CI config · **T6** documentation.
Reports are hypotheses. Documentation is intent. Runtime is truth.

---

## 1. What AWIS actually is today

AWIS is a working, committed, genuinely tested **CLI workflow engine** in Go — Step / EventLog /
IntelligencePort / StoragePort over SQLite, pull-based execution. A fresh clone builds clean; the full
gate suite passes; 22 Go test packages, an 8-file integration suite and 41 Python tests all pass. The
documented CLI contract matches the shipped commands one-for-one, and a spot-check of the PRD's V1
functional requirements found no material undelivered promise.

Alongside it sits a **read-only HTTP API and web GUI that have never existed in version control** — zero
commits, on any branch, in the project's entire history. They are complete, they build, they have their
own tests, and they are invisible to CI, to a clone, and to everyone but one machine.

The gap between those two paragraphs is the whole story, and it is not primarily technical.

---

## 2. The finding that changes the picture — `engine-hardening` could not pass CI [T1]

Every confidence claim in the corpus rests on "the gates are green." That green was machine-local.

`engine-hardening` pinned its version goldens to the exact Go patch release of whoever last regenerated
them (`awis version 0.1.0-dev  go go1.26.5`). CI pins `go-version: "1.26.x"`, which resolves to the
newest 1.26 patch. `main` had already fixed this class in `d5ac004` — a commit whose own message records
it was "found by the first CI run in the project's history" — and the fix never reached `engine-hardening`.

```
fresh clone of engine-hardening @ 8a87f70
  go1.26.5 (this machine)   go test ./cmd/awis/ -run TestVersion   ok
  go1.26.6 (what CI gets)   FAIL TestVersionHuman / FAIL TestVersionJSON
main @ 98350f6, same go1.26.6                                      ok
```

So the branches had split into a state nobody had written down: **the branch with all 62 hardening commits
could not go green, and the branch that could go green had none of the hardening work.**

### Resolved in this pass (uncommitted)

Two repairs, both "intent wins" — the code was right, the tests were wrong about the world.

1. Ported `main`'s `d5ac004` normalisation onto `engine-hardening`; both goldens are now byte-identical
   to `main`'s. Verified across go1.26.4 / .5 / .6.
2. `cmd/awis/system_test.go` — raised `terminalBudget` from 10s to 45s. The budget was provably smaller
   than the test's own measured steady state (11.13–11.23s).

```
make verify (go1.26.5)                      ALL GATES PASSED
go test ./... under go1.26.6 (CI's)         NO FAILURES
cmd/awis full package x3 (the flaky mode)   ok · ok · ok
4 files touched, uncommitted
```

Left uncommitted deliberately: the release branch is an open founder decision (§6 D2), and this working
tree cannot absorb a careless `git add -A` (§5).

---

## 3. Claims that did NOT survive verification

| Claim | Verdict | What is actually true |
|---|---|---|
| **UF-51** — the multi-provider routing layer is "built, tested and inert; it has never executed" | **FALSE** [T2] | `sdk/runtime.go:86-93` builds router *and* dispatcher on every runtime; `:112` wires it to the engine; `internal/runner/intelligence/intelligence.go:68` calls it on **every intelligence step**. It is **degenerate, not dead** — always built with exactly one registration, `Locality` hardcoded to `LocalityLocal`. It needs configuration, not resurrection. The corpus's own adversarial review re-framed this claim and kept the false half. |
| **RC-5 / D-05** — the Anthropic model IDs are "fictional non-existent strings" | **FALSE** [T2] | `claude-sonnet-5` and `claude-haiku-4-5-20251001` are both valid, current model IDs. The proposed fix (`claude-3-5-sonnet-20241022`) would have downgraded to deprecated models, and was in fact applied once before being reverted. The surviving defect is the guard, not the constants — see §4 UF-N1. |
| **BLOCKER 3** — the localhost condition "is not documented or enforced anywhere in the deployment path" | **OVERSTATED** [T2] | Loopback *is* the enforced default: `cmd/awis-server/main.go:40`, `defaultAddr = "127.0.0.1:8090"`, with an explanatory comment. All 7 routes are `GET`; there is no write surface. Residual risk: an operator can override with `-addr 0.0.0.0` and there is no auth behind that. Guard-rail gap, not an open server. |
| **RA-07** — defect class B-31 "reopened on the entire API surface" | **FALSE** [T2] | `internal/api` is GET-only and never calls `validate.*`; there is no write-validation surface for the class to recur on. The underlying bug (malformed query params silently accepted) is real but distinct and always-open. |

Corollary: "no body limits / no rate limiting" is moot on a GET-only surface. Missing CORS policy,
absent server timeouts, and verbatim internal error strings remain real.

---

## 4. Findings that are real — several worse than filed

### The namespace defect is silent data loss, not a visible error [T1]

Filed everywhere as "registration fails with `ErrAlreadyRegistered`" — an error an operator would see.
Run end-to-end it does something worse:

```
two workflows, same id+version, namespaces "marketing" and "sales"
  $ awis --json start
  {"workflows":["sync-leads 1.0.0","sync-leads 1.0.0"]}   exit 0, nothing on stderr
  sqlite> SELECT id,version,namespace FROM workflow_definitions;
  sync-leads|1.0.0|marketing                              <- "sales" silently discarded
```

Three parts combine: the primary key `(id, version)` excludes namespace
(`0001_core_execution.sql:18`); `cmd/awis/start.go:187-195` treats an already-registered error as a
successful idempotent restart; and `start.go:478-483` detects that case with
`strings.Contains(err, "already registered")` — a substring match that **structurally cannot distinguish
an idempotent restart from a different namespace being dropped**. The legitimate restart feature is what
converts a visible failure into silent loss.

### Register

| ID | Finding | Verdict | Evidence | Owner |
|---|---|---|---|---|
| UF-13 | GUI + HTTP API have zero commits in all history | TRUE [T2] | Real commit surface is **42 files, not 553** — the rest is correctly-ignored `node_modules`. The `go:embed` assets under `cmd/awis-server/static/` are committable, so the 42-file set does produce a buildable server. One deliberate `git add`. | Impl |
| UF-10 | Cross-namespace registration | **WORSE** [T1] | See above | Arch |
| UF-N2 | `RebuildState` silently drops cancellation intent | OPEN [T2] | Migration 0007 adds `cancellation_reason` + `cancellation_compensate`; the rebuild INSERT (`rebuild.go:259-263`) lists neither. After `rebuild-state`, cancellation finalises as plain `cancelled`, **skipping compensation** — RC-2's exact failure mode via a second path, inside RC-2's own fix. B-1 fixed the sibling column and stopped one short. | Impl |
| UF-N1 | The model-ID guard is tautological | OPEN [T3] | `anthropic_test.go:85-92` asserts the constant equals a literal copy of its own definition. Detects an accidental edit; cannot detect incorrectness — the failure that has already happened twice. Its doc comment claims otherwise. | Impl |
| UF-54 | `Retry-After` is never honoured | OPEN [T2] | Retry runs (`anthropic.go:233`), but `parseRetryAfter` has zero call sites, `retryAfter` is read at `retry.go:132` and never assigned, and `anthropic.go:315` builds the error without it and never sees the headers. **Orphan** — absent from every Beta-scoped register. `docs/PROVIDERS.md` documents the opposite. | Impl |
| UF-62 | The `intelligence:` config key is inert | OPEN [T2] | Provider selection keys solely off `anthropic_api_key` (`start.go:142-153`). Setting `intelligence: anthropic`, exactly as the generated scaffold suggests, validates fine and does nothing. | Impl |
| B-17 | Closed, but not fixed as stated | REOPEN [T2] | Original wording: "`ListInstances` has no LIMIT and is called every 100ms tick." `tick.go:39` still calls it unbounded on every tick. What shipped was an *additive* paged read model for the SDK — a different thing. | Impl |
| B-26 | Fake guard | OPEN [T3] | Reverting the `awis export` namespace fix leaves both `go test ./cmd/awis/` and the full integration suite **green**. The defect can silently return. | Impl |
| UF-08 | FTS recall rebuilds the whole index every search | TRUE [T1] | Re-measured: 27ms @1k · 302ms @10k · 1.96s @50k · **3.1s @80k** (range 2.35–3.64s). Lower than the corpus's 3.08–6.69s, but linear in *total* log size. `prune-events` confirmed dry-run-only — no relief valve. | Impl |
| UF-18 | `integration` + `pytest` never run in CI | OPEN [T1] | CI's own "deliberately not wired" list omits the Go `integration` target entirely. Both suites currently **pass** (8.31s; 41 tests) — risk is latent, not realised. Matters because B-30's only behavioural guard is integration-only. | Impl |
| BLOCKER 7 | The merge conflict is four files, not one | OPEN [T2] | `ci.yml`, `Makefile`, `internal/plugin/transport.go`, `subprocess_test.go`. Resolving `transport.go` toward `main` **does not compile** (merged body uses a `dir` param `main`'s signature lacks) — `-X ours` is actively unsafe. The report corrected this blocker once and still only re-checked `ci.yml`. | Impl |
| UF-31 | The "flaky" system test | **FIXED** [T1] | The corpus measured it in isolation, where it passes 8/8. CI runs package mode, where it failed ~2 of 3. Budget was 10s against 11.2s steady state. | — |

### Fixes confirmed genuine by revert-testing [T3]

Not accepted on the register's word — each fix was reverted in a scratch copy and the named guard
confirmed to fail: **RC-1** (crash recovery), **RC-2** (cancellation durability), **RC-3** (instance
ordering), **RC-4** (subprocess env leak), **D-06** (dead-end stall), **B-1**, **B-4**, **B-15**,
**B-18**, **B-22**, **B-28**, and config masking. **B-30** reproduced at T1 — removing
`_txlock=immediate` reproduces `database is locked (5) (SQLITE_BUSY)` verbatim.

### Singly-sourced — reproduce before actioning

**N-1** (HIGH, T1) `hydrate` classifies a signal step sitting between the delivery-tx commit and the
scanner's out-of-tx `CompleteStep` as a crashed worker; the signal is recorded delivered then discarded,
orphaning the wait record. **N-3** removed plugins remain dispatchable by name (`manager.go:669`).
**N-4** `hydrate.go:197` marks an instance hydrated before a loop that can error, restoring the original
D-01 wedge on a transient storage error. **N-5**, **N-6**, **N-7** as filed.

---

## 5. One root cause, three instances

Filed as unrelated defects across three registers, by two lineages that never cross-referenced each
other. **"Silently accept, silently ignore"** — AWIS accepts input that looks valid, does nothing with
it, and reports success:

- **B-31** — partially-numeric diagnostic inputs (`asInt("12abc") == 12`). *Closed.*
- **UF-62** — the `intelligence:` config key: validated, scaffolded, never read. *Open.*
- **UF-10** — cross-namespace registration: dropped, reported as success. *Open.*

Track as one class with one owner. The pattern is what recurs, not the individual site.

### Working-tree hazard

90 staged files (including an `.agent/` submodule→directory conversion), 228 unstaged (219 of them
deletions of a separate `.agents/` tree), 120 untracked (42 GUI/API, ~59 under `docs/`, ~19 root-level
audit reports). A `git add -A` here would commit all of it as one indistinguishable blob. Any commit
must be a deliberate, separate change-set.

---

## 6. Decisions only the founder can make

Every remaining blocker reduces to one of these. None can be closed by engineering; none has a durable,
dated record of having been decided.

- **D1 — Is the HTTP API and GUI in V1 scope at all?** The canonical PRD (one commit, 2026-07-02, never
  amended) scopes the HTTP API as **V2** and the web dashboard plus auth as **V3**, in four independent
  places (§17, §21, §34, §35). No milestone M00–M18 authorises either. The GUI's own planning corpus is
  untracked and self-labelled "not yet ratified," and explicitly records D1–D4 as undecided.
  *Until this is answered, "commit the 42 files" is not an engineering task — it is executing a scope
  decision nobody has made.*
- **D2 — Which branch cuts the release?** `engine-hardening` +62 / `main` +6; `main` uniquely holds the
  golden fix and the macOS matrix. Tags stop at `milestone/M14`. No document names a target.
- **D3 — Single-tenant, or enforce it?** Shipping with the namespace PK as-is is defensible *only* under
  an assumption nothing in the code enforces, and the failure mode is silent data loss.
- **D4 — The auth stance, stated explicitly.** Read-only, loopback-by-default, no auth, overridable.
  A coherent posture for a single-operator Beta — but currently implicit.
- **D5 — Restore `docs-lint`, or drop it?** The premise for dropping it is **true** (`scripts/docs-lint.sh`
  exits 1), but the gap is exactly **7 missing documents and 2 misplaced extras** across M15/M16/M17.
  The choice is not "red CI or no gate."
- **D6 — Reconcile the model-allocation policy.** `CLAUDE.md` freezes Opus as forbidden and states it
  "MUST NOT be selected automatically"; `.claude/agents/awis-core-engineer.md` carries `model: opus` with
  an auto-selectable description. Filed as cosmetic; it is a direct contradiction between two governing
  documents.

---

## 7. Corpus disposition

This document supersedes the following for all questions of defect status and release readiness.
Recommend moving them to `archive/` rather than deleting, and citing only this file going forward:

`FINAL_EXECUTIVE_SUMMARY.md`, `FINAL_RELEASE_VERDICT.md`, `FINAL_VERDICT.md` (both copies — root and
`docs/10-release-candidate-audit/`, which hold **opposite verdicts** under the same filename),
`RELEASE_BLOCKERS.md`, `RELEASE_READINESS_REPORT.md`, `RELEASE_AUDIT_REPORT.md`,
`RELEASE_CANDIDATE_AUDIT.md`, `RELEASE_CANDIDATE_REMEDIATION_REPORT.md`, `VERIFIED_DEFECT_REGISTER.md`,
`VERIFIED_GEMINI_FINDINGS.md`, `ENGINE_READINESS_SCORECARD.md`, `PHASE2_BLOCKERS.md`,
`ARCHITECTURAL_DEBT_REGISTER.md`, `DEFERRED_TECHNICAL_DEBT.md`, `SCALABILITY_ASSESSMENT.md`,
`OPERATIONAL_READINESS_REVIEW.md`, `REGRESSION_REPORT.md`, `REPOSITORY_HEALTH_REPORT.md`,
`IMPLEMENTATION_REPORT.md`, `DOCUMENT_DRIFT_REPORT.md`, `DOCUMENTATION_DIVERGENCE_REPORT.md`,
`AWIS_28_ANSWERS.md`, `AWIS_OPEN_DECISIONS.md`, `AWIS_RECONSTRUCTION_ANSWERS.md`.

Retained as canonical and NOT superseded: `AWIS_PRD.md`, `IMPLEMENTATION_MASTER_PLAN.md`,
`AWIS_EEOS.md`, `AWIS_ENGINEERING_ORGANIZATION.md`, the architecture blueprints, and `docs/edr/`.

### Why the corpus kept failing

At least **11 distinct ID schemes**, with three unrelated uses of the bare prefix `D-`, two of `B-`, two
of `BLOCKER`, two unrelated things called `ADR-001`, and two independent `Attack 1–10` sequences. Exactly
one collision was ever self-flagged. Two investigation lineages — the Beta/release audits and the
intelligence-architecture program — ran over the same codebase and **never once cited each other**; that
is how UF-54, a reproducible unfixed production defect, sat outside every defect register.

And the evidence chain is not durable: **the audit corpus is itself untracked.** `RELEASE_BLOCKERS.md`
cites `SCALABILITY_ASSESSMENT.md §2.4` for its FTS numbers, and neither file is in version control. None
of it reproduces from a clone. The corrective is not another document — it is that the tracked repository
becomes the only thing anyone is allowed to cite.

---

## 8. Remaining work, by owner

| Owner | Item | Gate |
|---|---|---|
| Founder | D1–D6 | blocks everything downstream |
| Impl | Commit the 42-file GUI/API set as one deliberate change | after D1 |
| Impl | Resolve the 4-file merge; do **not** use `-X ours` | after D2 |
| Impl | **UF-N2** — add the two cancellation columns to the rebuild INSERT | now · correctness |
| Impl | **UF-N1** — replace the tautological model-ID guard, or delete its false comment | now · cheap |
| Impl | **B-26** — make the export guard actually fail when reverted | now · cheap |
| Impl | **UF-54** — parse and honour `Retry-After`, or correct `PROVIDERS.md` | now |
| Impl | **UF-62** — read the `intelligence:` key, or stop scaffolding it | now |
| Impl | **B-17** — reopen under original wording, or re-scope in writing | now · bookkeeping |
| Impl | Gate `integration` + `pytest` in CI; both already pass | now · mechanical |
| Impl | Reproduce and fix N-1, N-3, N-4 | reproduce first |
| Scribe | 7 missing EEOS module documents | after D5 |
| Scribe | Correct stale comments in `storage/cancellation.go:13-15`, `engine/cancel.go:47-48` | now |
| Product | Fix the FTS rebuild, or document that `recall` degrades with log size | V1 claim |

---

## 9. Stated limits of this closure

- Findings N-1, N-3, N-4, N-5, N-6, N-7 are singly-sourced. Reproduce before actioning.
- The V1 functional-requirement sweep was a spot-check across the areas most likely to be stubbed, not
  an exhaustive proof of completeness across all ~120 FR IDs.
- FTS timings are from one machine; the mechanism is confirmed, the absolute numbers are hardware-bound.
- Two code fixes were applied to the working tree and left **uncommitted**.
- Everything above concerns the committed engine and the untracked GUI/API as they stood at `8a87f70`.
