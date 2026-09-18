# M16 — Traceability
> **Reconstructed post-hoc on 2026-09-18 at commit f3a897b.** This file was not authored
> during M16 execution; the module shipped without it. Every entry below is transcribed
> from `docs/05-implementation/STATE.md`, this module's `IMPLEMENTATION_SPEC.md`
> execution record, and its `VALIDATION_CHECKLIST.md`. No verification was re-run to
> produce this document and no claim here is independent evidence.

| T | Task | Source coordinates | Card |
|---|---|---|---|
| T1 | `internal/intelligence/adapters/anthropic`: Draft/Synthesize/Classify, model_hint mapping | IMP §27.M16; Blueprint §14/§16; PRD §24; FR-IL-02/05..09 | C1 |
| T2 | Embed NOT implemented (typed unavailable error) | CONTRA-3 | C1 |
| T3 | `CloudRetryPolicy` (429/5xx, Retry-After, backoff, max 3 attempts, ctx-aware) | IMPLEMENTATION_SPEC.md CE pins | C1 |
| T4 | FR-IL-09 usage side-channel (adapter/model/tokens via StepCompleted, M08 UsageRunner seam) | FR-IL-09 | C1 |
| T5 | NFR-S-01 key masking + leak-scan test | NFR-S-01; SR-01 | C1 |
| T6 | CI fixture-only contract suite (`porttest` vs fake http.Server) + `TestLiveSmoke` (`ANTHROPIC_LIVE=1`-gated) | IMPLEMENTATION_SPEC.md CE pins | C1 |
| T7 | `docs/PROVIDERS.md` | IMPLEMENTATION_SPEC.md DoD | C1 |
| T8 | `cmd/awis start` wiring block (construct adapter if key present; "anthropic (cloud)" header) | IMPLEMENTATION_SPEC.md CE pins | C1 |

## Notes / dispositions (per IMPLEMENTATION_SPEC.md)
- **Card split:** C1 (everything above) → V1. Single-card milestone.
- **Non-scope:** Embed/OpenAI (V2); config file + `config show` masking UX (M17); streaming;
  caching. No engine/storage/sdk/core changes at all.

## Execution record (transcribed verbatim from IMPLEMENTATION_SPEC.md "Execution record")
- C1 (3b12689, 2026-07-11): full milestone, no deviations. V1 (at 3b12689): PASS all rows.
- D-CLOSE (Fable, 2026-07-11): scope exact (new adapter pkg + start wiring + docs); frozen
  port untouched; zero-AI path proven unchanged (E1 8/8). APPROVED FOR SQUASH MERGE.

## VALIDATION_CHECKLIST.md status (transcribed, all rows ticked `[x]` in the repo)
- V-COMMON all + pytest regression; no network in CI tests.
- Adapter implements frozen IntelligencePort; porttest contract suite PASS; interface diff EMPTY.
- Draft/Synthesize/Classify wired per SPEC; Embed absent with typed unavailable error.
- CloudRetryPolicy test-proven against fake server.
- FR-IL-09 harness test shows StepCompleted usage carries adapter/model/tokens.
- NFR-S-01 leak-scan test + fixture grep.
- TestLiveSmoke exists, skips without ANTHROPIC_LIVE=1.
- cmd/awis start header behavior confirmed for both key-present and key-absent paths.
- docs/PROVIDERS.md exists.
- go.mod diff EMPTY; frozen surfaces untouched; no existing test modified.

STATE.md records no LAST-N narrative entries for M16 beyond the CARDS block (`M16-C1 DONE
3b12689`, `M16-V1 DONE (PASS all rows at 3b12689)`) and the MERGE-RECOMMENDATION line — this
module's STATE.md entry is shorter than M15's/M17's and carries no BLOCKERS line.
