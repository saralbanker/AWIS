# AWIS — Final Executive Summary

**The question:** If the founder froze all development today, could AWIS be responsibly released as a Beta product?

**The answer: Not today, as-is — but it is close, and the gap is procedural, not technical.**

---

## Why not today

A "release" requires, at minimum, a commit someone else can check out and run. Right now, **the entire HTTP API and web GUI — `internal/api/`, `internal/buildinfo/`, `cmd/awis-server/`, and `web/` — exist only in this working directory. They have never been committed to git, on any branch, ever.** This was independently confirmed by direct `git log --all` and a full history scan across every commit on every branch.

This is compounded by two facts that make it worse than a simple "forgot to commit":

- A tracked file, `cmd/awis/main.go`, already has uncommitted edits that depend on the uncommitted `internal/buildinfo` package. Committing those edits alone, without the new package, would break the build for anyone who pulls.
- This exact problem was already discovered and written down three times before this review (2026-08-30, 09-01, 09-03) and never fixed. Two more root-level reports on 2026-09-05 rediscovered it again, apparently unaware of the earlier findings.

Separately, and just as importantly: the canonical, founder-approved PRD (frozen since a single commit on 2026-07-02, never amended) explicitly scopes the HTTP API as "V2" work and the web dashboard plus authentication as "V3" work. **Nobody with the authority to expand that scope has actually decided to.** The GUI and API grew past the approved V1 boundary by accumulation, not by decision — which is also why there's no authentication on an API that's about to be called a "Beta."

## What can ship today

The **CLI engine** — `cmd/awis`, `internal/core`, `internal/engine`, `internal/storage`, `internal/plugin`, `internal/runner`, `internal/signal`, `internal/intelligence` — is genuinely in good shape:

- It's committed, it's tested (21 of 22 test packages passed clean on a from-scratch run; the one flaky failure reproduced as PASS in isolation and was a timing-margin issue, not a functional bug), and CI actually exercises it.
- All eight previously-claimed "closed defects" (B-31, cancellation durability, crash recovery, dead-end stalls, subprocess env leakage, health endpoint, dashboard ordering, Anthropic adapter) were independently re-verified against current code with file:line evidence and hold up. One prior claim — that B-31 "reopened" on the API surface — was checked and found incorrect; the API has no write path where that defect class could recur.
- The core architecture (Step / EventLog / IntelligencePort / StoragePort, pull-based execution) matches its documentation and is not stale relative to what's actually built.

If the founder froze development today and the release were scoped to *just this* — the CLI and engine, which is what's actually in git — that would be a defensible Beta, with two caveats worth escalating rather than quietly deferring (see below).

The **GUI and API**, evaluated purely on code quality with the commit problem set aside, are also solid: no contract mismatches between frontend and backend, both build and typecheck clean, all five screens are real implementations, not stubs. This is not a "the GUI is broken" verdict — it's a "the GUI isn't real yet, in the only sense that matters for shipping something" verdict.

## What must happen before a Beta that includes the GUI/API

In rough order:

1. **Commit `internal/api/`, `internal/buildinfo/`, `cmd/awis-server/`, and `web/`** together with the pending `cmd/awis/main.go`/`start.go` changes, as one atomic commit. Hours of work, not a rewrite.
2. **Get an explicit founder decision** on pulling the HTTP API/GUI into V1 scope (amending the PRD), since the canonical spec currently reserves this for V2/V3.
3. **Decide and document the auth stance** — localhost-only for now, or build real auth, but stop shipping it as an implicit assumption.
4. **Add CI coverage** for the API and GUI — currently zero, because the code was never committed for CI to see.
5. **Resolve the pending `ci.yml` merge conflict** between `main` and `engine-hardening` before merging either into the other.

None of these are large engineering lifts. This is a release that is one focused work session away from being real, not a program that needs re-architecting.

## Two items that should not be quietly filed as "deferred, non-blocking"

Two of the three items commonly described that way turned out, under direct verification, to already be measured, real problems rather than theoretical ones:

- **Namespace isolation**: registering the same workflow name in two different namespaces fails today — a documented, reproducible bug with zero test coverage of the failure mode, not a hypothetical edge case.
- **FTS recall cost**: the `awis recall` command does a full, unconditional index rebuild on every call, measured at 3–7 seconds at 80,000 events, and the cost only grows since the event log is append-only and pruning is dry-run-only.

Both are fine to leave unfixed for a genuinely single-tenant, low-volume V1 — but they should be tracked as known, scoped commitments, not filed alongside genuinely low-risk items like the single-SQLite-connection design (which *is* safe to defer, with a clearly named trigger for revisiting it).

## On the state of the audit process itself

This repository currently holds 16+ self-generated audit and verdict documents produced across three bursts in the last week, several reaching different conclusions using four incompatible defect-numbering schemes, none of which had previously identified the PRD-scope gap above. This review itself was produced by several parallel investigation threads dispatched against this same repository state, and more than one of them wrote conclusions directly to disk before being reconciled into these five files — a correction was needed at least twice in that reconciliation (an incorrect "merge cleanly" claim and an incorrect description of what actually differs between the two branches' CI configs, both fixed in `RELEASE_READINESS_REPORT.md`/`RELEASE_BLOCKERS.md` before this summary was finalized). That is itself worth naming: even a single review, run across parallel threads, reproduced the same pattern the rest of this document criticizes — output that looked authoritative until it was independently re-checked. More audits are not what closes this gap — a single commit, a founder decision, and a CI update are.
