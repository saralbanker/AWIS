# M16 → M17/M18 Handoff
> **Reconstructed post-hoc on 2026-09-18 at commit f3a897b.** This file was not authored
> during M16 execution; the module shipped without it. Every entry below is transcribed
> from `docs/05-implementation/STATE.md`, this module's `IMPLEMENTATION_SPEC.md`
> execution record, and its `VALIDATION_CHECKLIST.md`. No verification was re-run to
> produce this document and no claim here is independent evidence.

**Status: PHASE E-MERGE (blocked on founder) per STATE.md; D-CLOSE (Fable, 2026-07-11)
APPROVED FOR SQUASH MERGE.** The `m16-anthropic-adapter` branch was merged into `main` at
`f3a897b` on 2026-09-18 as part of a repository consolidation, without a recorded founder
merge action beyond the D-CLOSE recommendation in the repo.

## Guaranteed outputs (contract — per IMPLEMENTATION_SPEC.md / README.md)
- `internal/intelligence/adapters/anthropic`: Draft (haiku/sonnet per model_hint), Synthesize
  (sonnet), Classify (haiku, placeholder wiring); Embed NOT implemented (CONTRA-3).
- `CloudRetryPolicy` (429/5xx, Retry-After, exponential backoff, max 3 attempts, ctx-aware).
- FR-IL-09: adapter/model/tokens carried through StepCompleted via the M08 UsageRunner seam.
- NFR-S-01: key-masking helper; key never appears in logs/errors/fixtures.
- `docs/PROVIDERS.md` (env config, model mapping, retry behavior, zero-AI fallback note).
- `cmd/awis start` wiring: constructs the adapter when `ANTHROPIC_API_KEY` is set; header
  line shows "anthropic (cloud)".

## What M17 may assume (per README.md soft-dependency line)
- `recall --synthesize` works against NullAdapter's empty-state without M16 present; the M16
  adapter is used automatically when an API key is configured. — not independently confirmed
  in this reconstruction; see IMPLEMENTATION_SPEC.md "Card split" (C1 only) for scope.

## What M18 may assume (per README.md hard-dependency line)
- AnthropicAdapter is available as one intelligence path for the 1-week OIP dogfood window.
  — not independently confirmed in this reconstruction.

## Known limitations (IMPLEMENTATION_SPEC.md "Non-scope")
- Embed/OpenAI deferred to V2. Config-file + `config show` masking UX deferred to M17.
  Streaming and caching out of scope.

## Actuals (per STATE.md CARDS block and IMPLEMENTATION_SPEC.md "Execution record")
- C1: 3b12689 (2026-07-11) — full milestone, no deviations recorded.
- V1: PASS all rows, at 3b12689 (per STATE.md; VALIDATION_CHECKLIST.md shows all 10 rows
  ticked `[x]`).
- D-CLOSE: Fable, 2026-07-11 — "scope exact (new adapter pkg + start wiring + docs); frozen
  port untouched; zero-AI path proven unchanged (E1 8/8). APPROVED FOR SQUASH MERGE."
  (IMPLEMENTATION_SPEC.md, verbatim.)
- Deviations: none recorded.
