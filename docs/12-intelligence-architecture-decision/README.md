# 12 — Intelligence Architecture Decision

Architecture commitment exercise: which intelligence architecture should AWIS adopt for the next 3–5 years?
Evidence base: production code at `8a87f70`. **No implementation, no code changes, no commits, no roadmap.**

Supersedes nothing. Builds on the investigation in [`../11-intelligence-architecture/`](../11-intelligence-architecture/), whose findings were validated rather than assumed.

## Verdict

# **OPTION B — Registry + Driver + Instance**

Three drivers (`openai-chat`, `anthropic-messages`, `google-genai`) cover all nine named providers. The policy engine (Option C) is deferred; its seam already exists and stays unbuilt.

**Gate:** AM-1, a `sdk.Config` freeze amendment, requires a founder decision. No workaround exists.

## Documents

| # | Document | Purpose |
|---|---|---|
| 1 | [ARCHITECTURE_DECISION_RECORD.md](ARCHITECTURE_DECISION_RECORD.md) | Formal ADR: decision, rationale, consequences, amendments, prerequisites |
| 2 | [OPTION_COMPARISON_MATRIX.md](OPTION_COMPARISON_MATRIX.md) | A vs B vs C, side by side |
| 3 | [FUTURE_COMPATIBILITY_ANALYSIS.md](FUTURE_COMPATIBILITY_ANALYSIS.md) | Nine providers against each architecture |
| 4 | [ADVERSARIAL_DECISION_REVIEW.md](ADVERSARIAL_DECISION_REVIEW.md) | Ten attacks on Option B; three forced corrections |
| 5 | [FOUNDER_DECISION_PACKAGE.md](FOUNDER_DECISION_PACKAGE.md) | Executive summary |
| 6 | [FINAL_VERDICT.md](FINAL_VERDICT.md) | Single recommendation |

## The decisive finding

`anthropic.Config` already exposes `BaseURL`, `HTTPClient`, `MaxAttempts`, `ModelFast`, `ModelQuality` — the adapter is a parameterized driver in all but name. **Option B adds no architectural layer; it is three packages where Option A is nine.** At AWIS's committed scale, B is the simpler option.
