# AWIS Documentation Divergence Report

This report audits the document layer itself — not whether individual features work, but whether the repository's own account of itself is coherent, current, and honest about what it hasn't checked.

---

## 1. A single fact, rediscovered four times, never fixed

The uncommitted state of `internal/api/`, `internal/buildinfo/`, `cmd/awis-server/`, and `web/` (see `RELEASE_BLOCKERS.md` Blocker 1) is not a new finding. It has independently surfaced at least four times before this review:

- **2026-08-30** — an earlier repository-truth pass under `docs/09-gui-planning/` already names the uncommitted tree.
- **2026-09-01** — recorded again, same conclusion.
- **2026-09-03** — `docs/10-release-candidate-audit/RELEASE_CANDIDATE_AUDIT.md:5` explicitly frames its own evaluation target as *"`engine-hardening` @ `e75c1f1` + uncommitted API/GUI implementation"* — an unambiguous, dated acknowledgment that the problem was known.
- **2026-09-05 (twice)** — `FINAL_RELEASE_VERDICT.md` and `RELEASE_AUDIT_REPORT.md`, both untracked, both produced the same day, each writing as though this were a fresh discovery, with no reference back to the September 3rd document that already said the same thing.

Between the first mention (Aug 30) and this review (Sep 6), **zero commits added any of these four paths to git.** The fact was documented, not acted on, four separate times.

## 2. Direct contradictions between existing reports

| Topic | Document A says | Document B says | Resolution (this review) |
|---|---|---|---|
| FTS recall severity | An earlier verdict doc describes FTS rebuild cost as "off the hot path" | `RELEASE_AUDIT_REPORT.md` corrects this to "not off the hot path... user-facing performance defect" | **B is correct.** `awis recall` is a documented, user-facing CLI command; the rebuild cost is measured at 3–7s per call at 80k events (`ARCHITECTURAL_DEBT_REGISTER.md` AD-04, `SCALABILITY_ASSESSMENT.md` §2.4) and grows with total log size. See `RELEASE_BLOCKERS.md` Blocker 6. |
| B-31 status | Commit `e75c1f1` and its message claim the silent-defaults class is closed | `RELEASE_AUDIT_REPORT.md` claims it is "reopened across the entire API surface" | **Neither fully right.** The CLI-side fix is real and confirmed (`internal/validate/validate.go:114-522`). The "reopened on the API" claim is **incorrect**: `internal/api` never calls `validate.*` and is GET-only — there is no write-validation surface for this defect class to recur on. See `RELEASE_READINESS_REPORT.md` §6. |
| Overall verdict | `docs/10-release-candidate-audit/README.md` (tracked in git) still asserts **"REJECTED FOR BETA RELEASE"** | Untracked Sep-5 reports (`FINAL_RELEASE_VERDICT.md`, `RELEASE_AUDIT_REPORT.md`) reach **"PASS WITH RISKS"** / conditional-pass framings | These are now three revisions apart and were never reconciled — the tracked, in-repo verdict a new contributor would find by reading `docs/` directly contradicts the newer, untracked, more-informed verdicts sitting in repo root. |
| Defect ID schemes | Different documents use `SEC-*`, `D-*`, `RA-*`/`AD-*`, and `C-*` prefixes for what are sometimes the same underlying defects | — | No cross-reference table exists anywhere mapping these schemes to each other. Tracking "is this defect actually closed" requires manually correlating four incompatible numbering systems by reading commit messages. |

## 3. The tracked `docs/` tree vs. the untracked root-level reports

`docs/08-engine-hardening/`, `docs/09-gui-planning/`, and `docs/10-release-candidate-audit/` are the parts of this documentation actually committed to git. They are, on the whole, more careful than the untracked root-level pile — `docs/10-release-candidate-audit/` is the one place in the tracked corpus that explicitly named the uncommitted-tree problem with a specific commit hash. But it is now stale: its terminal verdict (`README.md`: "REJECTED FOR BETA RELEASE") has not been updated to reflect three subsequent rounds of (untracked) reassessment. A reader relying only on what's actually in git would see a rejected verdict; a reader who happens to open repo root would see a conditional pass. Both are present, simultaneously, uncommitted-to-committed, with no note connecting them.

## 4. The canonical corpus was never amended for the GUI/API at all

`AWIS_PRD.md` and the architecture blueprint documents were finalized in a single commit on 2026-07-02–03 and have not been substantively touched since. They therefore predate the GUI and HTTP API by roughly two months. This is not merely "the docs are stale" — the PRD **explicitly scopes** the HTTP API as V2-only and the web dashboard plus authentication as V3-only. Every one of the many audit documents produced since (evaluating GUI screens, API contracts, dashboard ordering, etc.) has been assessing a feature set the canonical specification does not authorize for the release under review. None of the 16+ audit documents in repo root raise this point — every one of them evaluates the GUI/API on its technical merits without ever checking whether it should exist in this release's scope at all. That check is the single most important thing missing from the entire document corpus, and this review is the first to raise it. See `RELEASE_BLOCKERS.md` Blocker 2.

## 5. A pattern worth naming: repeated re-litigation without resolution

Three audit bursts in the span of a week (Aug 30, Sep 1, Sep 3, and two more on Sep 5) each independently re-derived from "current repository state," each implicitly or explicitly disclaimed the authority of the prior round, and each converged on substantially the same core finding — without the underlying `git add` ever being run in between rounds. The volume itself (16+ overlapping root-level `*.md` verdict/audit/register documents, several produced same-day) is evidence of a process that keeps re-answering the same question rather than closing the gap the answer keeps pointing at.

This pattern was also visible in real time during this review, though a specific, cautious claim is warranted rather than a sweeping one: this review's own investigation was itself split across several parallel sub-investigations against this same repository state, and more than one of them independently wrote conclusions directly to disk before being reconciled into these five documents — including, at least once, a factual error (an incorrect "the two branches merge cleanly" claim, and separately an incorrect description of what actually differs between their CI configs) that only surfaced because it was re-checked rather than taken on faith, and a claim about "eight other independently-running agent sessions with names like [...]" that appeared in an earlier draft of this file and could not be substantiated on review — no mechanism available to any of these sub-investigations can actually enumerate other sessions' names, so that specific claim has been removed as unverified rather than repeated. What is verifiable is simpler and still makes the point: multiple uncoordinated passes at the identical question, even within one review, reproduce exactly the sprawl-without-resolution pattern this section describes — a reason to treat "one more audit" as lower-value than actually committing the fix, not a reason to trust the next audit's every claim by default either.

## 6. Recommendation

Before generating any further audit documents:

1. Fix Blocker 1 (commit the GUI/API tree) — this alone invalidates most of the caveats in the existing 16 documents.
2. Retire or clearly date-stamp the superseded root-level reports; they currently coexist with no indication of which is authoritative.
3. Update `docs/10-release-candidate-audit/README.md`'s verdict once Blockers 1–4 are addressed, so the tracked corpus and reality agree again.
4. Build one cross-reference table mapping `SEC-*`/`D-*`/`AD-*`/`C-*` IDs to each other, or retire all but one scheme going forward.
