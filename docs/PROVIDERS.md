# AWIS Intelligence Providers

This document describes how intelligence providers are configured, selected, and
wired into the AWIS runtime (Blueprint §14; IMP §27.M16).

---

## Zero-AI Mode (default)

When no provider is configured the runtime runs in **zero-AI mode**: the
NullAdapter is used, `IsAvailable()` returns `false`, and every intelligence
step is routed to its declared fallback (FR-IL-01 by construction). No external
network calls are made. The startup header shows:

```
Intelligence:         none (zero-AI mode)
```

This is Article 32 behaviour: the platform operates identically to a
provider-enabled deployment — only the intelligence step outcomes differ.

---

## Anthropic Provider (M16)

### Overview

The Anthropic adapter (`internal/intelligence/adapters/anthropic`) calls the
[Anthropic Messages API](https://api.anthropic.com/v1/messages) using stdlib
`net/http` only (no third-party SDK).

### Configuration

Set the environment variable before starting the runtime:

```sh
export ANTHROPIC_API_KEY=sk-ant-api03-...
awis start
```

Config-file wiring is M17. The env var is the V1 path.

When the key is present the startup header shows:

```
Intelligence:         anthropic (cloud)
```

Constructing the adapter directly (e.g. in sdk consumers):

```go
import "github.com/awis/awis/internal/intelligence/adapters/anthropic"

port := anthropic.New(anthropic.Config{
    APIKey:      os.Getenv("ANTHROPIC_API_KEY"),
    MaxAttempts: 3, // default
})
cfg := sdk.Config{
    Intelligence: port,
    // ... other fields
}
rt, err := sdk.NewRuntime(cfg)
```

`sdk.Config.Intelligence` accepts any `core.IntelligencePort`; the adapter
construction is the only provider-specific step.

### Model Mapping

| `model_hint` | Anthropic model (V1 pin) |
|---|---|
| `fast` | `claude-haiku-4-5` |
| `quality` | `claude-sonnet-4-5` |

Model IDs are pinned in a single const block in `anthropic.go` with a comment
marking them as config-overridable post-V1.

Classify uses the fast model (placeholder wiring per IMP §27.M16).

### Capabilities

| Capability | Status |
|---|---|
| `draft` | Available |
| `synthesize` | Available |
| `classify` | Available (fast model placeholder) |
| `embed` | **Not available** (CONTRA-3 — Blueprint §14 V1) |

Requesting `embed` returns `anthropic.ErrEmbedUnavailable`; the router falls
through to the step fallback.

### Retry Behaviour (CloudRetryPolicy)

- **Retryable**: HTTP 429 (rate limit) and 5xx (server errors).
- **Retry-After**: if the `Retry-After` header is present its value (seconds)
  is honoured; otherwise exponential backoff is used (500 ms, 1 s, 2 s …).
- **Max attempts**: 3 (configurable via `Config.MaxAttempts`).
- **Non-retryable 4xx**: returned immediately as `*anthropic.ProviderError`;
  the router treats this as a provider failure and falls through to the fallback
  chain.
- **Context-aware**: all sleeps honour `ctx.Done()`.

### Key Masking (NFR-S-01)

The API key **never** appears in logs, error messages, or test fixtures. Any
code path that would surface the key calls `mask(key)` which returns
`"sk-ant-…<last4>"`. Tests assert the raw key is absent from all error strings.

### Usage Recording (FR-IL-09)

Every successful `Draft` and `Synthesize` call populates `core.Usage`:

```go
core.Usage{
    Adapter:    "anthropic",
    Model:      "<model-id reported by API>",
    TokensUsed: inputTokens + outputTokens,
}
```

This feeds the `UsageRunner` side-channel (ADJ-8) so `StepCompleted` events
record `{adapter, model, tokens_used}`.

### Live Smoke Test

The `TestLiveSmoke` test in `live_test.go` is **skipped in CI** (requires a
real key and network). Run it manually:

```sh
ANTHROPIC_LIVE=1 ANTHROPIC_API_KEY=sk-ant-... \
  go test ./internal/intelligence/adapters/anthropic/ -run TestLiveSmoke -v
```

---

## Adding a New Provider (post-V1)

1. Create `internal/intelligence/adapters/<name>/`.
2. Implement `core.IntelligencePort` (compile-time assertion required).
3. Run `porttest.Run` in the adapter's test package.
4. Wire construction in `cmd/awis/start.go` (check env var, construct, pass to
   `sdk.Config.Intelligence`).
5. Document here.

OpenAI and additional providers are V2 scope.
