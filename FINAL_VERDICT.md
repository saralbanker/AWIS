# Final Verdict: AWIS Beta

**Date:** 2026-09-05
**Supersedes:** the 2026-09-03 verdict of the same name (REJECTED FOR RELEASE), which was correct for the state it evaluated.

---

## VERDICT: **PASS WITH RISKS**

---

## Basis for this verdict

The 2026-09-03 audit (`RELEASE_CANDIDATE_AUDIT.md`) correctly rejected the Beta candidate for five Critical/High defects (SEC-01 through SEC-05). All five have since been fixed and are independently re-verified against today's working tree in `VERIFIED_GEMINI_FINDINGS.md`:

1. In-flight step crash recovery deadlock — **fixed, tested.**
2. Cancellation intent lost on restart, compensation silently skipped — **fixed, tested.**
3. Dashboard landing page showing oldest instances instead of newest — **fixed, tested.**
4. Subprocess steps leaking host secrets via inherited environment — **fixed, tested.**
5. Anthropic model identifiers — **fixed, but only after this verification caught that the first fix attempt had itself introduced a regression** (it replaced a correct model ID with a deprecated one). Corrected this session.

This verification additionally found and fixed two defects the prior remediation pass did not address:

6. A workflow whose taken execution path dead-ends on a non-final leaf step permanently wedges the instance in `status: running` forever — **fixed, tested, with a new reproduction test.**
7. The healthcheck endpoint never checked database connectivity — **fixed, tested.**

`go build`, `go vet`, the full unit suite (28 packages), the integration suite, and the frontend build are all green. See `REGRESSION_REPORT.md`.

## Why "PASS WITH RISKS" and not a clean PASS

Three genuine, verified findings remain unaddressed by design, not by oversight:

- Full FTS-index rebuild on every search query (D-08) — real, but a documented V1 tradeoff off the hot path.
- Single shared SQLite connection between engine ticks and HTTP reads (D-09) — real contention risk under concurrent load, but undemonstrated at V1's current single-tenant scale.
- No live-API round-trip validates the corrected Anthropic model IDs (residual risk on D-05) — this verifier's knowledge is the only source of truth for those IDs being current, since `ANTHROPIC_LIVE` is unset in CI.

None of these block a single-tenant Beta release. All three should be resolved or explicitly re-accepted before the next milestone that changes those assumptions (higher search volume, concurrent multi-user load, or shipping a Claude-backed workflow to production without a live smoke test).

Two further findings from the original audit (workflow_definitions namespace isolation, absence of a workflow mutation API) are real but correctly out of scope — they gate Phase 2 (multi-tenancy, visual builder), not Beta. One finding (acyclic DAG precluding loops) is not a defect at all — it is a deliberate, frozen architectural decision.

## Why not BLOCKED or NEEDS WORK

No known correctness, data-loss, or security defect remains open against the Beta engine surface. "NEEDS WORK" would overstate the state of things — the deferred items are backlog-appropriate technical debt with recorded justification, not unfinished remediation of the audit's findings.

## Supporting evidence

- `VERIFIED_GEMINI_FINDINGS.md` — per-finding disposition and verification method for all 12 audit findings.
- `VERIFIED_DEFECT_REGISTER.md` — 13 entries (D-01 through D-13) with severity, impact, root cause, and recommended action.
- `IMPLEMENTATION_REPORT.md` — every change this session made, with reasoning and evidence.
- `REGRESSION_REPORT.md` — full build/vet/test output, including the isolated-vs-parallel investigation of the one flaky test observed.
- `REPOSITORY_HEALTH_REPORT.md` — technical debt, risk register, and recommended next phase.
