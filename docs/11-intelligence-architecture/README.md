# 11 — Intelligence Architecture Investigation

Provider-agnostic redesign investigation for the AWIS intelligence subsystem.
Evidence base: production code at `8a87f70` (branch `engine-hardening`), independently adversarially verified.

**Planning and investigation only. No code was written or modified.**

## Reading order

| # | Document | Answers |
|---|---|---|
| 1 | [INTELLIGENCE_TRUTH_AUDIT.md](INTELLIGENCE_TRUTH_AUDIT.md) | What exists today, built vs reachable; reality vs assumptions; documentation drift. **Question 1.** |
| 2 | [PROVIDER_AGNOSTIC_GAP_ANALYSIS.md](PROVIDER_AGNOSTIC_GAP_ANALYSIS.md) | What blocks OpenAI/Gemini/Ollama/OpenRouter/LM Studio/vLLM, with exact files. **Question 2.** |
| 3 | [ARCHITECTURE_OPTIONS.md](ARCHITECTURE_OPTIONS.md) | Five architectures compared; lowest-cost option without rewrite debt. **Question 4.** |
| 4 | [IMPLEMENTATION_PROGRAM.md](IMPLEMENTATION_PROGRAM.md) | Phase 0–3 roadmap with deliverables, dependencies, exit criteria. **Questions 3 and 5.** |
| 5 | [ADVERSARIAL_REVIEW.md](ADVERSARIAL_REVIEW.md) | Disproof attempts against every major conclusion. Three did not survive. |
| 6 | [FINAL_VERDICT.md](FINAL_VERDICT.md) | Founder decision package and single recommendation. |

Background: [00-ARCHITECTURE_REVIEW.md](00-ARCHITECTURE_REVIEW.md) — the initial architecture review that opened this investigation.

## Verdict

**PROCEED WITH CHANGES.** Authorise Phase 0 (activation); authorise Phases 1–3 contingent on its exit criteria.

Requires a founder decision first: `sdk.Config` is frozen post-M08 and deliverable 0.1 needs a documented additive amendment.
