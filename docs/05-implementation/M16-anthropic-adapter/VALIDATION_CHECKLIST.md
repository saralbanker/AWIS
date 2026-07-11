# M16 — Validation Checklist (binary; exit list)
- [x] V-COMMON all ✅ + pytest regression; NO network in CI tests (fixtures/fake server only)
- [x] Adapter implements the frozen IntelligencePort; porttest contract suite PASS against
      fake-server-backed adapter; interface diff EMPTY (R4)
- [x] Draft/Synthesize/Classify wired; model_hint fast/quality mapping per SPEC; Embed absent
      with typed unavailable error (CONTRA-3)
- [x] CloudRetryPolicy: 429 w/ Retry-After honored, 5xx backoff ≤3 attempts, 4xx non-retry,
      ctx cancel — test-proven against fake server
- [x] FR-IL-09: harness test shows StepCompleted usage carries adapter/model/tokens
- [x] NFR-S-01: key never in logs/errors/fixtures (leak-scan test + fixture grep)
- [x] TestLiveSmoke exists, skips without ANTHROPIC_LIVE=1
- [x] cmd/awis start: with key env → header "anthropic (cloud)"; without → NullAdapter path
      byte-identical to M15 behavior (zero-AI unchanged; Article 32)
- [x] docs/PROVIDERS.md exists (env config, models, retry, zero-AI note)
- [x] go.mod diff EMPTY; engine/storage/sdk/core/dsl/plugin untouched (cmd/awis/start.go
      wiring block + new adapter pkg + docs only); no existing test modified
