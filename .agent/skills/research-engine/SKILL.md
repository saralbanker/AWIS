---
name: research-engine
description: >
  Use when investigating an unfamiliar technology, evaluating competing
  options, or synthesizing multiple sources into one cited answer. Not for
  established patterns already known cold — activate when the answer
  requires investigation, not recall.
last_verified: 2026-08-30
kill_condition: n/a — source-hierarchy and synthesis-over-summary are durable research practices.
---

# Research Engine

## When to use
"Which is better, X or Y", evaluating a library/approach, or any question
whose answer needs checking against current external sources rather than
recall.

## Procedure
1. **Rank sources by authority** before trusting them: official docs and
   specs first, established engineering blogs and books second, high-vote
   Q&A third, individual blog posts last — never cite a lone low-tier
   source as decisive.
2. **Check recency**, calibrated to how fast the domain moves: fast-moving
   ecosystems (AI/LLM tooling, frontend frameworks) need recent sources;
   language/CS fundamentals are largely evergreen. Date-stamp anything that
   might be time-sensitive ("as of March 2026, ...").
3. **Triangulate**: check what multiple independent sources say about the
   same claim before trusting it. Agreement across 3 sources is high
   confidence; a single source making a claim is low confidence.
4. **Synthesize, don't summarize.** Answer the question in the first
   sentence, then give the evidence — don't just list "source A says X,
   source B says Y."
5. **State a confidence level and at least one caveat** — the condition
   under which the recommendation would change.

## Avoid
Presenting a book-report of sources instead of an answer; treating a single
blog post as settled; leading with a wall of alternatives instead of one
clear recommendation.

## Checklist
- [ ] Conclusion stated first, in one or two sentences
- [ ] Every factual claim has a source
- [ ] Sources checked for recency where it matters
- [ ] Confidence level stated (high/medium/low) with a caveat
- [ ] Output is a synthesis, not a list of what each source said

See `reference/playbook.md` for the source-tier table and a worked
comparison example.
