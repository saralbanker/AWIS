# M16 — AnthropicAdapter
**Status:** Partitioned — materialize at entry · **Effort:** 1.5d · **Window:** Week-5 float / Week 6 (F-6: start early)
**Objective:** AnthropicAdapter: Draft (haiku/sonnet per model_hint), Synthesize (sonnet), Classify (haiku, placeholder wiring); CloudRetryPolicy (429/5xx, Retry-After); adapter/model/tokens in StepCompleted (FR-IL-09); env-var key wiring; secret masking in config show/logs/export (SR-01). Embed unavailable in V1 (CONTRA-3).
**Depends on:** M04 (+M14 for config UX) · **Blocks:** M18
**Primary sources:** IMP §27.M16; Blueprint §14/§16; PRD §24 · **Compilation spec:** IKB §4/M16
**Key ACs:** IntelligencePort contract suite vs recorded fixtures (CI, zero network); live smoke manual-only; rollback guarantee: platform runs identically without it (Art. 32/P2).
