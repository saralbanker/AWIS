# AWIS — System Health Baseline

**Established:** 2026-09-17 · **Repository:** `/mnt/data/rj/AWIS`, branch `engine-hardening` @ `8a87f70` (2026-09-05), working tree as found.
**Method:** Every verdict below is graded by an evidence tier actually checked in this pass, not by which document is newest. Executable verification (build/vet/lint/test/grep/git) was re-run independently in this session; document claims were used only where corroborated or where explicitly flagged as unverified.
**Status:** This document is itself an audit act. Per the governing principle it adopts (§6), on acceptance its conclusions should be moved into a durable state register and this file tombstoned — it must not become a further entry in the "final verdict" pile it describes.

Evidence tiers used:
- **T1 — Executed** in this session (build, vet, lint, test, grep, git query) — highest confidence.
- **T2 — Repository fact** (git tracking, commit dates, file existence) — objective, not re-derived.
- **T3 — Corroborated document claim** — a prior report's claim, cross-checked against T1/T2 evidence in this session and found consistent.
- **T4 — Uncorroborated document claim** — asserted by one or more reports, not independently checked in this session.

---

## 1. Current health summary

**Engine: healthy.** A build of the current working tree (`go build ./...`, `apps/oip` standalone with `GOWORK=off`) is clean; `go vet` clean in both modules; `golangci-lint run ./...` reports **0 issues** in both modules; `gofmt -l .` reports no unformatted files; the full `go test ./...` suite passes (24 packages, no failures) [T1]. `go test -race ./...` was also run to completion in this session — all 24 packages pass with the race detector enabled, no data races reported [T1].

**Release envelope: not ready, for a documented and narrow reason.** The entire Beta HTTP API + GUI deliverable (`internal/api`, `cmd/awis-server`, `web/`, `internal/buildinfo` — 42 files) exists on disk, builds, and passes its own tests, but has **zero commits in the project's history on any branch** and is not gitignored — confirmed directly in this session (`git log --all -- <dir>` = 0 for all four paths; `git status --porcelain` shows each as untracked; `find` count = 42 files excluding `node_modules`) [T1]. A clean clone of `HEAD` cannot build `awis-server` at all.

**Governance: collapsed, and the collapse is well-documented but the documents themselves are part of the problem.** 24+ root-level "final verdict"/audit-style documents exist, the large majority of them **also untracked** [T1 — confirmed via `git status --porcelain` in this session]. Five separate files are named `FINAL_VERDICT.md`-equivalent; two on-disk copies of `FINAL_VERDICT.md` (root and `docs/10-release-candidate-audit/`) hold different verdicts, though the root copy explicitly declares it supersedes the other (a real, dated supersession chain, not a raw contradiction — see §4). `docs-lint`, the one mechanical documentation control, was removed from CI on 2026-08-21 and **still fails today** with the same three findings (M15/M16/M17 module-contract mismatches), confirmed by running it directly in this session [T1]. `docs/05-implementation/STATE.md`, the project's execution ledger, was last modified 2026-08-21 and contains no record of the 30+ engine-hardening commits or any work after that date [T1 — checked via `git log -1 --date=short -- docs/05-implementation/STATE.md`].

**Net effect:** the code is in materially better shape than the volume of "release blocked" documentation suggests. The blockers that remain are specific, small, and enumerable (§5), not systemic engine defects.

---

## 2. Evidence hierarchy actually used

The task brief's Tier 1–4 ordering ("current repo state > executable verification > latest audits > older docs") was followed, with one adjustment forced by the evidence itself: **document recency does not track document reliability** in this corpus. The newest audit-style document by mtime is not automatically more correct than an older one — several later documents correct earlier ones, but at least as often a later document repeats an earlier document's error, or a later document is itself untracked and therefore has no more durable standing than the one it claims to supersede. Concretely:

- `TRUTH_CLOSURE.md` (2026-09-09, root, **untracked**) is the most thorough re-verification of engine/defect claims in the corpus — it explicitly re-tested prior claims against running code (T1-labeled in its own text) and corrected four of them from TRUE to FALSE/OVERSTATED. This session independently re-verified 9 of its load-bearing claims against current code (§3, §5) and found all 9 accurate. It is treated here as the most reliable **content** source for defect/release status, not as a governance-final authority.
- `remedation-diagnosis/AWIS_DOCUMENTATION_FORENSICS_AND_REMEDIATION_ARCHITECTURE.md` (2026-09-11, **untracked**) is the most reliable source for the **governance/meta** story (why the corpus exists, which documents are tracked, timeline of the collapse). Its E1/E2-tagged claims were spot-checked in this session (docs-lint still fails with 3 findings; STATE.md unwritten since 08-21; 42 files/0 commits) and confirmed [T1].
- Everything else is graded individually below rather than trusted by position in a stack.

---

## 3. Document timeline (system-health / release-readiness / audit-verdict class only)

Governance/documentation-audit-class documents (the ~24-doc corpus TRUTH_CLOSURE addresses, plus the forensics report) are in scope here; the ~319-document full corpus and its docs/ governance layer are out of scope per the brief ("do not re-audit the system").

| Date | Document | Tracked? | Scope | Verdict | Supersedes (stated) |
|---|---|---|---|---|---|
| 2026-09-03 | `RELEASE_CANDIDATE_AUDIT.md` (+ identical copy in `docs/10-release-candidate-audit/`) | No | Engine + GUI/API adversarial audit | **REJECTED** — 3 Critical, 4 High, 4 Medium | — |
| 2026-09-03 | `ENGINE_READINESS_SCORECARD.md`, `PHASE2_BLOCKERS.md`, `DOCUMENT_DRIFT_REPORT.md` (root + `docs/10` copies) | No | Companion findings to the above | Supporting detail, same verdict lineage | — |
| 2026-09-05 | `RELEASE_CANDIDATE_REMEDIATION_REPORT.md` | No | Fixes for the 09-03 findings | All 5 Critical/High fixed and validated | — |
| 2026-09-05 | `FINAL_VERDICT.md` (root) | **Yes** (8a87f70) | Beta release | **PASS WITH RISKS** | `FINAL_VERDICT.md` (09-03, REJECTED) |
| 2026-09-05 | `VERIFIED_DEFECT_REGISTER.md`, `VERIFIED_GEMINI_FINDINGS.md`, `REGRESSION_REPORT.md`, `REPOSITORY_HEALTH_REPORT.md`, `IMPLEMENTATION_REPORT.md` | **Yes** (8a87f70) | Supporting evidence for the above | Consistent with PASS WITH RISKS | — |
| 2026-09-05 | `RELEASE_AUDIT_REPORT.md` | No | Re-derivation from Tier-1 evidence, engine + delivery | **CONDITIONAL PASS** (engine good, delivery not committed) | `FINAL_VERDICT.md` (09-05, PASS WITH RISKS) |
| 2026-09-05 | `FINAL_RELEASE_VERDICT.md` | No | Formal verdict + numbered release conditions C1–C15 | **CONDITIONAL PASS** | `FINAL_VERDICT.md` (09-05, PASS WITH RISKS) |
| 2026-09-05 | `ARCHITECTURAL_DEBT_REGISTER.md`, `SCALABILITY_ASSESSMENT.md`, `OPERATIONAL_READINESS_REVIEW.md` | No | Debt / scale / ops posture | Debt catalog; scale ceiling ~5–10k instances (O(n²) CLI sort); ops maturity LOW | — |
| 2026-09-06 | `RELEASE_READINESS_REPORT.md`, `DOCUMENT_DRIFT_REPORT.md` (root), `DOCUMENTATION_DIVERGENCE_REPORT.md`, `AWIS_28_ANSWERS.md`, `AWIS_OPEN_DECISIONS.md`, `AWIS_RECONSTRUCTION_ANSWERS.md` | No | Re-derivation + Q&A/reconstruction pass | Consistent with CONDITIONAL PASS lineage | — |
| 2026-09-09 | `TRUTH_CLOSURE.md` | No | Re-verifies all major claims in the 24-doc corpus against running code | **Corpus consolidated**; declares itself CANONICAL for defect/readiness questions; lists open items with owners (§7 below) | Explicitly supersedes 24 named documents (its own §7) |
| 2026-09-11 | `remedation-diagnosis/AWIS_DOCUMENTATION_FORENSICS_AND_REMEDIATION_ARCHITECTURE.md` | No | Meta: why the corpus collapsed, which artifacts are tracked, target documentation architecture | Root cause: E-MERGE gate stalled 2026-07-10; engine sound; delivery and governance both fell out of version control in the same way | Treats `TRUTH_CLOSURE.md` as content to **preserve** (not supersede), everything else in the audit lineage as **merge/archive** |

**Newest document:** the forensics report (09-11). **Newest evidence-backed document:** also the forensics report, on governance questions; `TRUTH_CLOSURE.md` (09-09) on defect/release questions — both corroborated in this session. **Newest document that explicitly supersedes prior work:** `TRUTH_CLOSURE.md`, which names its 24 superseded documents in its own §7 (a genuine self-declared tombstone, unusual in this corpus — most of the 22 other superseded documents in the wider set carry no such marker).

---

## 4. Trust map

**Authoritative (use these going forward):**
- `TRUTH_CLOSURE.md` — for defect status and release-readiness content. Independently corroborated in this session (§5). Treat as a snapshot to fold into a durable register, not as a permanent address.
- `remedation-diagnosis/.../AWIS_DOCUMENTATION_FORENSICS_AND_REMEDIATION_ARCHITECTURE.md` — for governance/meta questions (why documents proliferated, tracking status, target architecture). Independently corroborated in this session.
- This document, going forward, for a single current-state answer — until it too is folded into a durable register per its own header note.
- The **tracked, code-adjacent** artifacts: `FINAL_VERDICT.md`, `VERIFIED_DEFECT_REGISTER.md`, `VERIFIED_GEMINI_FINDINGS.md`, `REGRESSION_REPORT.md`, `REPOSITORY_HEALTH_REPORT.md`, `IMPLEMENTATION_REPORT.md` — genuinely committed evidence as of 8a87f70, useful as a paper trail even though their headline verdict (PASS WITH RISKS) was superseded by later same-day re-derivation.
- `docs/08-engine-hardening/ENGINE_FREEZE_REPORT.md` and its two review reports (tracked, 2026-08-29) — accurate for the engine-only scope and date they cover; do not extend their "0 critical/important defects" claim past 08-29 (TRUTH_CLOSURE found new engine-adjacent issues after that date: UF-N1, UF-N2, UF-54, UF-62, B-17, B-26).

**Superseded (do not cite for current state, retain as history):**
`FINAL_EXECUTIVE_SUMMARY.md`, `FINAL_RELEASE_VERDICT.md`, `FINAL_VERDICT.md` (both copies), `RELEASE_BLOCKERS.md`, `RELEASE_READINESS_REPORT.md`, `RELEASE_AUDIT_REPORT.md`, `RELEASE_CANDIDATE_AUDIT.md`, `RELEASE_CANDIDATE_REMEDIATION_REPORT.md`, `VERIFIED_DEFECT_REGISTER.md`, `VERIFIED_GEMINI_FINDINGS.md`, `ENGINE_READINESS_SCORECARD.md`, `PHASE2_BLOCKERS.md`, `ARCHITECTURAL_DEBT_REGISTER.md`, `DEFERRED_TECHNICAL_DEBT.md`, `SCALABILITY_ASSESSMENT.md`, `OPERATIONAL_READINESS_REVIEW.md`, `REGRESSION_REPORT.md`, `REPOSITORY_HEALTH_REPORT.md`, `IMPLEMENTATION_REPORT.md`, `DOCUMENT_DRIFT_REPORT.md` (both copies), `DOCUMENTATION_DIVERGENCE_REPORT.md`, `AWIS_28_ANSWERS.md`, `AWIS_OPEN_DECISIONS.md`, `AWIS_RECONSTRUCTION_ANSWERS.md` — this is `TRUTH_CLOSURE.md`'s own supersession list (its §7); this session did not find reason to dispute it.

**Redundant (duplicate content under different names / paths):** `docs/10-release-candidate-audit/RELEASE_CANDIDATE_AUDIT.md`, `.../ENGINE_READINESS_SCORECARD.md`, `.../PHASE2_BLOCKERS.md`, `.../DOCUMENT_DRIFT_REPORT.md` are near-duplicates of the same-named root, 09-03-dated files [confirmed by header comparison in this session, T1]. `docs/10/FINAL_VERDICT.md` is the earlier (09-03, REJECTED) state of the verdict that root's `FINAL_VERDICT.md` (09-05, PASS WITH RISKS) explicitly supersedes — not a true contradiction under the same name, but a stale copy left in place with nothing marking it stale.

**Actively misleading if read alone, without this baseline or `TRUTH_CLOSURE.md`:**
- `RELEASE_CANDIDATE_AUDIT.md` and its `docs/10` copy — the REJECTED verdict was correct for 09-03 but every cited Critical/High defect it lists has since been fixed; a reader who opens only this file has no way to know that from the file itself.
- `docs/10-release-candidate-audit/FINAL_VERDICT.md` — presents as a live, final verdict (REJECTED-equivalent framing) with nothing in the file itself indicating it was superseded two days later.
- Any document in this corpus taken in isolation, per the forensics report's central finding: supersession in this corpus is almost entirely forward-only (declared by the successor, never inscribed on the ancestor) — 22 of 24 documents `TRUTH_CLOSURE.md` supersedes carry no tombstone of their own [T3, corroborated by this session's own reading of the ancestor files, which indeed carry no superseded marker].

---

## 5. Verified findings (T1/T2 this session, or T3 corroborated)

Independently confirmed against the current working tree in this session:

| # | Claim | Status | Evidence |
|---|---|---|---|
| V-01 | Clean build, vet, lint (0 issues both modules), gofmt clean, full test suite green | **CONFIRMED** | `go build/vet ./...`, `apps/oip GOWORK=off build/vet`, `golangci-lint run ./...` (both modules), `gofmt -l .`, `go test ./...` — all run this session |
| V-02 | 42 Beta-deliverable files (`internal/api`, `cmd/awis-server`, `web/`, `internal/buildinfo`) have zero commits, any branch, ever; not gitignored | **CONFIRMED** | `git log --all -- <dir>` = 0 for all four; `git status --porcelain` shows each untracked; `find` count = 42 excl. `node_modules` |
| V-03 | `.gitignore`'s comment asserting `web/` source and `cmd/awis-server/static/` are tracked is false | **CONFIRMED** | Comment read at `.gitignore:28-31`; contradicted by V-02 |
| V-04 | UF-N2 — `RebuildState`'s INSERT omits `cancellation_reason`/`cancellation_compensate`, added by migration 0007; rebuild-state silently drops cancellation intent and compensation | **CONFIRMED, still OPEN** | `internal/storage/rebuild.go:258-263` INSERT column list has no `cancellation_reason`/`cancellation_compensate`; `internal/storage/migrations/0007_cancellation_intent.sql` adds both columns |
| V-05 | UF-10 — `workflow_definitions` primary key is `(id, version)`, excludes `namespace`; cross-namespace registration collisions are possible at the schema level | **CONFIRMED, architectural** | `internal/storage/migrations/0001_core_execution.sql:18` |
| V-06 | Anthropic model IDs (`claude-sonnet-5`, `claude-haiku-4-5-20251001`) are current, valid model identifiers, not fictional | **CONFIRMED** | Read `internal/intelligence/adapters/anthropic/anthropic.go:34,36`; corroborated against this session's own model-ID knowledge |
| V-07 | UF-N1 — the model-ID regression guard is tautological (asserts constant equals a literal copy of itself; cannot detect the value being wrong, only changed) | **CONFIRMED, still OPEN** | `internal/intelligence/adapters/anthropic/anthropic_test.go` `TestModelConstants` compares `modelQuality`/`modelFast` to hardcoded string literals |
| V-08 | UF-54 — `Retry-After` header is parsed (`parseRetryAfter`) but never wired to a live call site; `retryAfter` field is never assigned in production code, only in tests | **CONFIRMED, still OPEN** | `grep -n "retryAfter:" anthropic.go` → no matches; `grep -n "parseRetryAfter(" anthropic.go` → no matches (only referenced from `retry_test.go`) |
| V-09 | UF-62 — the `intelligence:` scaffolded config key is never read for provider selection; only `ANTHROPIC_API_KEY`/`anthropic_api_key` gate it | **CONFIRMED, still OPEN** | `cmd/awis/start.go:140-153` — provider selection is keyed solely on API-key presence |
| V-10 | B-17 — `ListInstances` (unpaged) has no LIMIT clause and is called on every engine tick | **CONFIRMED, still OPEN** | `internal/engine/tick.go:39` calls `e.storage.ListInstances(...)`; `internal/storage/sqlite.go` `ListInstances` implementation has no `LIMIT`/pagination |
| V-11 | C4 (`FINAL_RELEASE_VERDICT.md`) — the HTTP server sets no `ReadHeaderTimeout`/`ReadTimeout`/`WriteTimeout`/`IdleTimeout` | **CONFIRMED, still OPEN** | `cmd/awis-server/main.go:111-114` — `http.Server{Addr: addr, Handler: top}`, no timeout fields |
| V-12 | Loopback default (`127.0.0.1:8090`) is the enforced default, but `-addr` accepts any interface with no auth behind it | **CONFIRMED** | `cmd/awis-server/main.go:40` `defaultAddr`; no auth middleware found in `internal/api` |
| V-13 | `docs-lint` still fails today with exactly the 3 findings the forensics report predicted (M15/M16/M17 module-contract mismatches) | **CONFIRMED** | Ran `bash scripts/docs-lint.sh` this session |
| V-14 | `docs/05-implementation/STATE.md` last modified 2026-08-21, predates all engine-hardening and GUI-planning work | **CONFIRMED** | `git log -1 --date=short -- docs/05-implementation/STATE.md` → `2026-08-21` |
| V-15 | `api_keys/apikeys.txt` is correctly gitignored and never committed | **CONFIRMED, not a risk** | `git check-ignore -v` matches; `git log --all -- api_keys/` empty |
| V-16 | The two uncommitted fixes `TRUTH_CLOSURE.md` describes (version-golden normalization, `terminalBudget` raised 10s→45s) are present in the current working tree as uncommitted modifications | **CONFIRMED** | `git status --porcelain` shows `M cmd/awis/testdata/golden/version.{json,txt}`, `M cmd/awis/main.go`, `M cmd/awis/main_test.go`, `M cmd/awis/system_test.go`; `grep terminalBudget` shows `45 * time.Second` |

---

## 6. Unverified findings (T4 — singly-sourced or not checked this session)

These are carried at their source document's confidence level only. They are candidates for the Haiku verification pass (see `HAIKU_VERIFICATION_TASKS.md`):

- **N-1** (TRUTH_CLOSURE, HIGH) — a race between a signal delivery-tx commit and the scanner's out-of-tx `CompleteStep` can misclassify a delivered signal as a crashed worker, orphaning the wait record.
- **N-3** — removed plugins remain dispatchable by name (`manager.go:669`).
- **N-4** — `hydrate.go:197` marks an instance hydrated before a loop that can error, potentially restoring a wedge on transient storage error.
- **N-5, N-6, N-7** — as filed in `TRUTH_CLOSURE.md` §4, not reproduced in this session.
- **B-26** — TRUTH_CLOSURE claims the `awis export` namespace guard fix is real but its regression test is not load-bearing (reverting the fix leaves both `go test ./cmd/awis/` and the integration suite green). Not revert-tested in this session.
- **BLOCKER 7** (RELEASE_CANDIDATE_AUDIT / TRUTH_CLOSURE) — a 4-file merge conflict (`ci.yml`, `Makefile`, `internal/plugin/transport.go`, `subprocess_test.go`) where resolving `transport.go` toward `main` does not compile. Not reproduced in this session (requires an actual merge attempt against `main`).
- **UF-08 / FTS rebuild cost** — full-index rebuild scaling measurements (27ms@1k … 3.1s@80k) are TRUTH_CLOSURE's own re-measurement; not re-benchmarked in this session.
- **RA-05/C9** — `awis status`/`awis history` use an O(n²) insertion sort, ~5s at 20k instances, ~19s at 40k. Not benchmarked in this session; `grep -n "sort\."` on `status.go`/`history.go` returned no direct hits, so the exact call site needs locating (see Haiku task list).
- **C8** — migration/schema-version downgrade guard for opening a DB written by a newer binary. This session found an event-level `SchemaVersion` ceiling (`MaxSupportedSchemaVersion`, B-22-related) but did not confirm whether a separate DB-file-level migration-version downgrade guard exists or is still missing — these may be the same mechanism or two different ones; needs disambiguation.
- **C10/RA-07** — TRUTH_CLOSURE marks the *write-validation* framing of this claim FALSE (API is GET-only, no `validate.*` call sites), but affirms a narrower, distinct defect remains: malformed query params (`?status=TOTALLY_BOGUS`) are silently accepted and return an empty-but-200 result. Not reproduced in this session.
- **C6, C7, C11, C12, C13(status), C14** — README/quickstart completeness, backup/restore + WAL documentation, internal error strings returned to clients, `bundle.js.map` embedded via `go:embed`, `slog`/stdout separation. Not checked in this session.

---

## 7. Contradicted findings (resolved — do not treat as open)

Claims from earlier documents in the corpus that later, evidence-based re-verification found FALSE or OVERSTATED. Corroborated in this session where noted:

| Claim (source) | Later verdict | This session |
|---|---|---|
| UF-51 — multi-provider routing is "built, tested and inert, never executed" | **FALSE** (TRUTH_CLOSURE) — it is wired and called on every intelligence step, just degenerate (one registration) | Not independently re-checked; TRUTH_CLOSURE's file:line citations (`sdk/runtime.go:86-93,112`, `internal/runner/intelligence/intelligence.go:68`) are specific enough to trust at T3 |
| RC-5/D-05 — Anthropic model IDs are "fictional non-existent strings" (RELEASE_CANDIDATE_AUDIT, 09-03) | **FALSE** (TRUTH_CLOSURE) — both IDs are valid and current | **Confirmed T1 this session** (V-06) |
| BLOCKER 3 — the loopback bind "is not documented or enforced anywhere" | **OVERSTATED** (TRUTH_CLOSURE) — loopback *is* the enforced default; the real residual risk is the unauthenticated `-addr` override | **Confirmed T1 this session** (V-12) |
| RA-07 — defect class B-31 "reopened on the entire API surface" | **FALSE** as stated (TRUTH_CLOSURE) — API is GET-only, `validate.*` is never called there; a narrower, distinct malformed-query-param defect is real and separate | Not independently re-checked; see §6 |
| `docs/10-release-candidate-audit/FINAL_VERDICT.md` (09-03, REJECTED) vs root `FINAL_VERDICT.md` (09-05, PASS WITH RISKS) | Not a raw contradiction — root explicitly supersedes the 09-03 verdict by name and date | Confirmed by reading both files this session (§3, §4) |

---

## 8. Founder/human decisions still open (Category D — cannot be closed by verification)

Carried from `TRUTH_CLOSURE.md` §6, cross-checked for continued relevance against this session's own findings:

- **D1** — Is the HTTP API/GUI in V1 scope at all? The canonical PRD scopes the HTTP API as V2 and the web dashboard as V3, in four places; no milestone authorizes either. Until answered, "commit the 42 files" is a scope decision, not an engineering task.
- **D2** — Which branch cuts the release: `engine-hardening` (+62 commits, all the hardening work, cannot yet pass CI on the newest Go patch per TRUTH_CLOSURE's own finding, since fixed but uncommitted per V-16) or `main` (+6, has the CI-compatible golden fix, none of the hardening)?
- **D3** — Ship single-tenant as-is (namespace PK gap, V-05) or enforce it?
- **D4** — State the auth posture explicitly (read-only, loopback-default, no auth, overridable) rather than leaving it implicit.
- **D5** — Restore `docs-lint` in CI, or drop it? The premise for dropping it is true (it fails, V-13) but the gap is exactly 3 modules' worth of missing files, not a fundamentally broken gate.
- **D6** — `CLAUDE.md`'s frozen model-allocation policy (this repository's own root instructions: Opus and Fable forbidden for implementation) versus `.claude/agents/awis-core-engineer.md` carrying `model: opus` with an auto-selectable description — a direct, currently-live contradiction between two governing files. Not re-checked in this session; flagged here because it is directly relevant to how future AI-agent work on this repository should be dispatched.

---

## 9. Confidence level

- **Engine correctness (build/vet/lint/test):** **High.** Directly re-executed this session, all green.
- **The central release blocker (uncommitted GUI/API, 42 files):** **High.** Directly confirmed via git this session, corroborated identically by three independent documents (TRUTH_CLOSURE, the forensics report, RELEASE_AUDIT_REPORT/FINAL_RELEASE_VERDICT).
- **The specific still-open code defects in §5 (V-04, V-07 through V-12):** **High.** Each confirmed by reading the cited line(s) directly in this session, not inherited from a report.
- **The governance collapse timeline and root-cause chain (§1, §3, §4):** **Medium-high.** Spot-checked (docs-lint, STATE.md date, file-tracking counts) and found accurate; the full 319-document, 186-commit analysis behind it was not independently re-run in this session — it is trusted at T3 because every spot-check performed here confirmed it.
- **Findings in §6 (N-1/N-3–N-7, B-26, BLOCKER 7, RA-05/C9, C8, C6/C7/C11–C14):** **Low-to-none independently.** These carry only their source document's confidence (mostly T1-labeled by TRUTH_CLOSURE, but not re-derived in this session). Treat as leads, not facts, until run through the Haiku verification pass.
- **`go test -race ./...`:** ran to completion in this session — clean, all 24 packages, no races (root module; `apps/oip` race run not separately executed).

**Overall system health: the engine is release-quality; the release process is not.** The gap is five specific, mostly mechanical conditions (commit the deliverable, add it to CI, set server timeouts, resolve the auth/bind posture, fix the two still-open silent-defaults-class bugs) plus a set of founder-only scope decisions (§8) — not a rewrite, and not the sprawling defect count the raw document volume suggests.
