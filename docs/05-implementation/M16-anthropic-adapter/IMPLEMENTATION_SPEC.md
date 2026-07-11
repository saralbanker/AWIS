# M16 — Implementation Spec
**Sources:** IMP §27.M16; Blueprint §14 (adapter design; CONTRA-3: NO Embed in V1) + §16;
PRD FR-IL-02/05..09; NFR-S-01; M4 contract suite (`internal/intelligence/porttest`);
null adapter as shape reference.

## CE pins
- `internal/intelligence/adapters/anthropic`: implements the frozen IntelligencePort. NO new
  Go dependency — stdlib net/http against the Messages API (https://api.anthropic.com/v1/
  messages, anthropic-version header). API key from `ANTHROPIC_API_KEY` (config file wiring
  is M17; env var is the V1 path — document).
- Capabilities: Draft + Synthesize + Classify. model_hint mapping: fast→claude-haiku-class,
  quality→claude-sonnet-class (pin current public model ids in ONE const block with a comment
  that ids are config-overridable post-V1); Classify uses the fast model (placeholder wiring
  per IMP). Embed: NOT implemented (CONTRA-3) — IsAvailable/capability set excludes it;
  requesting it → typed unavailable error (router falls through).
- CloudRetryPolicy: 429/5xx retry, honor Retry-After, exponential backoff, max 3 attempts,
  ctx-aware. Non-retryable 4xx → typed provider error.
- FR-IL-09: return usage (adapter name, model, input/output tokens) through the SAME usage
  side-channel the Null/dispatcher path uses (UsageRunner seam, M08) so StepCompleted records
  adapter/model/tokens. Verify via harness test with a FAKE local HTTP server.
- NFR-S-01: the API key never appears in logs, errors, or fixtures — masking helper
  (`mask(key) → "sk-ant-…last4"`); error paths scrubbed; test asserts no key leakage in a
  captured error string.
- CI = recorded fixtures ONLY (testdata/*.json request/response pairs; fake http.Server
  replays them). Contract suite: run `porttest` against the adapter backed by the fake
  server. Live smoke: `TestLiveSmoke` guarded by `ANTHROPIC_LIVE=1` + key (skipped in CI;
  IMP Val row "manual").
- DoD: `docs/PROVIDERS.md` — provider config (env var, model mapping, retry behavior,
  zero-AI fallback note: without a key the platform runs identically on NullAdapter,
  Article 32 by construction).
- sdk wiring: NONE required (Config.Intelligence already accepts any IntelligencePort;
  document construction `anthropic.New(anthropic.Config{...})` in PROVIDERS.md). cmd/awis
  start: if ANTHROPIC_API_KEY set, construct the adapter and pass it in Config.Intelligence;
  header line shows "anthropic (cloud)" — ONE small cmd/awis/start.go wiring block (CLI is
  not frozen; allowed).

## Card split
C1 (everything above) → V1.

## Non-scope
Embed/OpenAI (V2); config file + `config show` masking UX (M17); streaming; caching.
No engine/storage/sdk/core changes at all.
