# M16 — AI Execution Context
**Model:** Sonnet. C1 to `awis-builder`; V1 to `awis-verifier`. Single-card milestone (C1 → V1).
Branch: `m16-anthropic-adapter` (stacked on m15-oip-on-awis).

**Gate:** none (IMPLEMENTATION_SPEC.md, README.md).

**Key constraints (IMPLEMENTATION_SPEC.md "CE pins"):**
1. `internal/intelligence/adapters/anthropic` implements the frozen IntelligencePort. No new
   Go dependency — stdlib `net/http` against the Anthropic Messages API. Interface diff EMPTY
   (VALIDATION_CHECKLIST.md row R4).
2. Capabilities: Draft + Synthesize + Classify only. Embed is explicitly NOT implemented
   (CONTRA-3) — requesting it returns a typed unavailable error; do not stub it.
3. CI = recorded fixtures only (`testdata/*.json` request/response pairs via fake
   `http.Server`); zero network in CI tests. `TestLiveSmoke` guarded by `ANTHROPIC_LIVE=1`,
   skipped in CI.
4. NFR-S-01: API key never appears in logs/errors/fixtures — masking helper + leak-scan test.
5. No engine/storage/sdk/core changes at all (IMPLEMENTATION_SPEC.md "Non-scope"). sdk wiring
   needed: NONE — `Config.Intelligence` already accepts any IntelligencePort; the only CLI
   touch is one small `cmd/awis/start.go` wiring block (CLI is not a frozen surface).

**Milestone escalation deltas:** none recorded beyond the standard EEOS triggers — this
milestone ran as a single card with no deviations (IMPLEMENTATION_SPEC.md "Execution record").
