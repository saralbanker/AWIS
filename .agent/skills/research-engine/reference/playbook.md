# Research Engine — Playbook

## Source hierarchy

| Tier | Source type | Trust |
|---|---|---|
| 1 | Official docs, RFCs, specs | High |
| 2 | Recognized experts' own writing, official engineering blogs | High |
| 3 | Established tech publications | Medium |
| 4 | High-vote Q&A (accepted, well-upvoted answers) | Medium |
| 5 | Individual blogs / tutorials | Low — corroborate before citing |
| 6 | AI-generated content found online | Very low — verify against tiers 1-3 |

## Recency calibration
LLM/AI tooling: prefer sources from the last 6 months. Frontend frameworks:
last 12 months. Languages generally: last 2 years is usually fine. CS
fundamentals: evergreen, age doesn't matter. Always note the publication
date on anything time-sensitive.

## Worked synthesis example

```
Conclusion: For this use case (serverless, TypeScript, simple CRUD),
Drizzle is the better fit.
Evidence: lower abstraction overhead than a full ORM (source 1), faster
cold starts in serverless (source 2), type-safe queries without a codegen
step (source 3).
Caveat: the alternative has a larger ecosystem and more mature migration
tooling — matters more for complex schemas (source 4).
Confidence: MEDIUM — sources broadly agree, but the ecosystem gap could
matter depending on how the schema evolves.
```

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| Sources contradict each other | different versions/contexts | note the version/context dependency; give a conditional answer |
| No authoritative source exists | very new or niche topic | say confidence is low; reason from first principles; label it experimental |
| User seems overwhelmed by options | too many alternatives, no clear pick | lead with one recommendation and why; alternatives are secondary |
