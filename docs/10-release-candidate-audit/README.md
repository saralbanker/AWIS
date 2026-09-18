# Release Candidate Challenge Audit

The Release Candidate Challenge Phase for AWIS Beta (`engine-hardening` @ `e75c1f1` + uncommitted API and GUI implementation).

Its objective is **not** to prove readiness, but to **disprove readiness**: an adversarial, evidence-backed challenge of engine correctness, crash recovery, state transitions, API contracts, security posture, GUI usability, scaling bounds, and Phase 2 / visual builder blockers.

---

## The Documents

| File | Description |
|---|---|
| [RELEASE_CANDIDATE_AUDIT.md](RELEASE_CANDIDATE_AUDIT.md) | **Master Audit Report** — Executive verdict, 12 severity-ranked findings (SEC-01 through SEC-12), code and schema citations, reproduction steps, recommended fixes, and blocking matrix. |
| [ENGINE_READINESS_SCORECARD.md](ENGINE_READINESS_SCORECARD.md) | **Readiness Scorecard** — Detailed 8-dimension scorecard (Correctness, Reliability, Maintainability, Extensibility, GUI Readiness, Builder Readiness, Security, Observability) with evidence citations. |
| [PHASE2_BLOCKERS.md](PHASE2_BLOCKERS.md) | **Blocker Register** — Ranked analysis of critical, high, and medium architectural blockers for workflow creation, editing, visual builder, and n8n-style evolution. |
| [DOCUMENT_DRIFT_REPORT.md](DOCUMENT_DRIFT_REPORT.md) | **Document Drift Analysis** — Comparison of `docs/09-gui-planning/` planning documents against actual code, migrations, and runtime behavior. |
| [FINAL_VERDICT.md](FINAL_VERDICT.md) | **Final Verdict & Action Plan** — Direct answers to release readiness, engineering focus, and the high-leverage `BETA-STABILIZATION-GATE` next phase. |

---

## Summary Verdict

**Status: REJECTED FOR BETA RELEASE.**

While unit tests are green, the release candidate contains:
1. **Critical:** In-flight step recovery deadlock on crash/restart ([`SEC-01`](RELEASE_CANDIDATE_AUDIT.md#finding-sec-01-in-flight-step-crash-recovery-deadlock-zombie-instances)).
2. **Critical:** Volatile in-memory cancellation intent silently bypassing compensation ([`SEC-02`](RELEASE_CANDIDATE_AUDIT.md#finding-sec-02-volatile-cancellation-intent-silent-compensation-bypass-on-restart)).
3. **Critical:** Inverted pagination ordering (`ORDER BY started_at ASC`) displaying the oldest 50 workflows on the landing screen ([`SEC-03`](RELEASE_CANDIDATE_AUDIT.md#finding-sec-03-dashboard-landing-inversion-oldest-50-workflows-shown-on-page-1)).
4. **High:** Subprocess environment variable secret leakage ([`SEC-04`](RELEASE_CANDIDATE_AUDIT.md#finding-sec-04-secret-leakage-subprocess-steps-inherit-all-host-environment-variables)).
5. **High:** Fictional Anthropic model identifiers failing live execution ([`SEC-05`](RELEASE_CANDIDATE_AUDIT.md#finding-sec-05-anthropic-live-integration-broken-fictional-model-ids--untested-in-ci)).

Remediation requires a focused 1.5–2 day `BETA-STABILIZATION-GATE` before transitioning to Phase 2 write-side enablement.
