# PROVIDER_AGNOSTIC_GAP_ANALYSIS

**Question answered:** what specifically prevents OpenAI, Gemini, Ollama, OpenRouter, LM Studio, and vLLM from being plugged in *today*?

All citations are production code at `8a87f70`. Every blocker below was reached by tracing an actual attempt to add a provider, not by inspecting the design.

---

## 1. The result of actually trying

Walk `docs/PROVIDERS.md`'s own "Adding a New Provider" procedure for, say, Ollama, and record where it stops.

| Step (per the docs) | Outcome |
|---|---|
| 1. Create `internal/intelligence/adapters/ollama/` | ✅ Works |
| 2. Implement `core.IntelligencePort` | ⚠️ Works, but forces two dead methods (`Embed`, `Classify` have no dispatch path) and `IsAvailable()` has no correct implementation for a keyless local server (§3, B-7) |
| 3. Run `porttest.Run` | ✅ Works — the contract suite is genuinely adapter-agnostic |
| 4. "Wire construction in `cmd/awis/start.go`" | ❌ **STOPS HERE.** You can wire *one* provider. Passing Ollama means **removing Anthropic**, because `sdk.Config.Intelligence` is a single field (`sdk/runtime.go:33`) |
| 5. Configure it | ❌ **STOPS HERE.** No config key can name it, no key can hold its `base_url`, and `awis config validate` **rejects** any new key |

**So the honest answer to "what prevents a provider being plugged in today" is: you can plug exactly one provider in, by editing and recompiling the CLI, and you cannot configure it.** The adapter layer is not the obstacle. The composition and configuration layers are.

---

## 2. Universal blockers — apply identically to all six providers

Ordered by how early they stop you.

### B-1 · `sdk.Config` admits exactly one provider — **BLOCKING, all six**

```
sdk/runtime.go:33   Intelligence core.IntelligencePort      // singular
sdk/runtime.go:86-89
    router, err := intel.NewRouter(
        []intel.Registration{{Adapter: port, Locality: intel.LocalityLocal}},
        []string{port.ProviderName()},
    )
```

One registration, one-element chain. There is no `AddProvider`, no `RegisterIntelligence`, no post-construction injection point — independently confirmed by adversarial review against the full `*Runtime` method set.

**Consequence:** multi-provider is not "unconfigured", it is **inexpressible through the public SDK**. Every downstream blocker is moot until this is fixed.

### B-2 · Composition root hardcodes one vendor — **BLOCKING, all six**

```
cmd/awis/start.go:36        import ".../adapters/anthropic"
cmd/awis/start.go:141-152   apiKey := os.Getenv("ANTHROPIC_API_KEY")
                            if apiKey == "" { apiKey = cfg["anthropic_api_key"] }
                            if apiKey != "" { intelligencePort = anthropic.New(...) }
```

The only credential the runtime will look for is Anthropic's. There is no loop over configured providers, and no indirection between "a credential exists" and "construct *this* adapter."

### B-3 · Config schema rejects any new key — **BLOCKING, all six**

```
cmd/awis/config.go:420-431   recognized := map[string]bool{
                               "namespace","tick","anthropic_api_key","api_key",
                               "data_dir","log_level","intelligence","plugins_dir" }
```

`validateConfigYAML` emits `unknown config key: "openai_api_key"` for anything outside that allowlist. So `awis config validate` **fails a config file that mentions any other provider.** This is a hard blocker, not a cosmetic one: it is a shipped CLI command returning an error on the configuration the feature requires.

Note the design tension to preserve: `configVisibleKeys` (`config.go:47-56`) is a deliberate **allowlist of the harmless**, chosen because a denylist printed an `authorization` key verbatim into an immutable audit row. Any provider-config expansion must extend that model, not weaken it — a `providers:` block introduces exactly the credential-shaped key names the current design was hardened against.

### B-4 · No endpoint/`base_url` anywhere in configuration — **BLOCKING for Ollama, LM Studio, vLLM, OpenRouter, local endpoints**

`anthropic.Config.BaseURL` exists (`anthropic.go:76-78`) but is documented "useful for fake servers in tests" and is **unreachable from configuration** — `start.go:151` constructs `anthropic.Config{APIKey: apiKey}` and sets nothing else.

Four of the six targets are defined *primarily* by their endpoint. An Ollama or vLLM adapter with no configurable `base_url` is not a provider; it is a constant.

### B-5 · No model identity in the request path — **BLOCKING for Ollama, vLLM, OpenRouter; SEVERE for OpenAI, Gemini**

`core.IntelReq` (`internal/core/step.go:87-99`) carries `Capability`, `ModelHint`, `ContextBudget`, `Required`. `ModelHint` is validated to `{"", fast, quality, local}` (`internal/validate/validate.go:476`). It is consumed by the router for **ordering only** and never forwarded — `DraftRequest`/`SynthesisRequest` (`ports.go:66,78`) have no model field.

Why this is fatal for three targets specifically:
- **Ollama / LM Studio / vLLM** require an explicit model tag (`llama3.1:8b`, a served model name). There is no "fast" or "quality" — there is the model you pulled.
- **OpenRouter** model IDs are namespaced (`anthropic/claude-sonnet-4`, `meta-llama/llama-3.1-70b`) and number in the hundreds. A three-value enum cannot address them.

### B-6 · One registry key namespace, silent collision — **BLOCKING for any two OpenAI-compatible endpoints**

```
internal/intelligence/router.go:85-87
    byName := make(map[string]Registration, len(regs))
    for _, r := range regs { byName[r.Adapter.ProviderName()] = r }
```

Plain map assignment, last-write-wins, no duplicate detection (adversarially confirmed). `ProviderName()` is a compile-time constant per adapter type (`anthropic.go:139` returns `providerName`).

**Consequence:** the natural target topology — one `openai-chat` driver serving a local vLLM *and* OpenRouter *and* OpenAI — is unrepresentable. Registering the second silently discards the first. "Prefer local vLLM, fail over to OpenRouter" cannot be expressed at all.

This is the strongest single argument that **provider *instance* identity must be separated from provider *type*** before any registry work begins.

### B-7 · `IsAvailable()` is a credential check, which excludes keyless providers — **BLOCKING for Ollama, LM Studio, default vLLM**

```
anthropic.go:124   func (a *Adapter) IsAvailable() bool { return a.cfg.APIKey != "" }
```

The router calls this per request (`router.go:134`) and skips any adapter returning false.

Ollama, LM Studio, and a default vLLM server **have no API key**. An adapter following the established repository pattern reports permanently unavailable and is **never routed to** — the provider would be configured, healthy, and unreachable.

The interface offers no alternative: `IsAvailable()` is documented as making no network round-trip, so there is no defined place to put the liveness probe these providers actually need. This is an *interface-level* gap, not an adapter bug.

### B-8 · Routing metadata is compile-time, not configurable — **DEGRADING, all six**

`Registration{Locality, CostRank, QualityRank}` (`router.go:24-33`) is populated at the call site. `sdk/runtime.go:87` stamps `LocalityLocal` on everything — **including the Anthropic cloud adapter** — and leaves both ranks at Go zero.

Adversarial review flagged this as a latent defect that surfaces the instant B-1 is fixed: the `fast` and `quality` hints filter to `Locality == LocalityCloud` (`router.go:157,171`), find zero members, and silently fall through to chain order. `stableSortByCost`/`stableSortByQuality` (`router.go:202-224`) sort by fields that are 0 for every adapter, degenerating to chain order regardless.

**So fixing B-1 alone would produce a multi-provider system whose routing hints silently do nothing.** B-8 must ship with B-1, not after it.

---

## 3. Per-provider specifics — what each needs beyond the universal blockers

The universal blockers (B-1…B-8) must be cleared for all six. Beyond those, wire-protocol work:

| Provider | Wire protocol | Beyond B-1…B-8 |
|---|---|---|
| **OpenAI** | `POST {base}/v1/chat/completions` (or Responses API) | New driver. `Authorization: Bearer` — the adapter hardcodes `x-api-key` + `anthropic-version` (`anthropic.go:264-266`). Native `response_format` JSON-schema mode is strictly better than the current prompt-append (fixes the D-08 class for this driver). |
| **OpenRouter** | OpenAI-compatible | **Same driver as OpenAI.** Needs `base_url` (B-4), Bearer auth, optional `HTTP-Referer`/`X-Title` headers → requires per-instance custom headers in config. Namespaced model IDs make B-5 unavoidable. |
| **vLLM** | OpenAI-compatible | **Same driver as OpenAI.** `base_url` mandatory (B-4). Auth usually absent → B-7 is the blocker. Serves one model; B-5 needed to name it. |
| **LM Studio** | OpenAI-compatible | **Same driver as OpenAI.** Identical to vLLM: B-4 + B-7 + B-5. |
| **Ollama** | `POST {base}/api/chat` native, **and** an OpenAI-compatible surface at `/v1` | **No new driver strictly required** — its OpenAI-compatible surface works with the OpenAI driver. B-4 + B-7 (keyless, and frequently not running, so a real liveness probe is mandatory, not optional) + B-5 (model tag). |
| **Gemini** | `POST {base}/v1beta/models/{model}:generateContent` | New driver. Genuinely different shape: **model is in the URL path** (so B-5 is structural, not a convenience), `contents`/`parts` body rather than `messages`, auth via `x-goog-api-key` or query param, `responseSchema` for structured output. |

### 3.1 The load-bearing observation

**Five of the six targets are served by two drivers.**

- `openai-chat` → OpenAI, OpenRouter, vLLM, LM Studio, Ollama (via its `/v1` surface) — **five of six**
- `google-genai` → Gemini — **one**

Only Gemini needs bespoke wire work. This is the single most consequential fact in the gap analysis, because it means **the driver layer is cheap and the registry layer is where the value is**. Effort spent generalising the adapter interface pays for one provider; effort spent on config-driven instance registration pays for five.

It also inverts the naive plan. "Add OpenAI, then add Ollama, then add vLLM" is three milestones under the current architecture and **one deliverable plus three config entries** under an instance registry.

---

## 4. Gaps that are not blockers today but become rewrite debt

These do not stop a provider being added. They guarantee rework if deferred past the point where multiple drivers exist.

| Gap | Location | Why deferring costs more later |
|---|---|---|
| `core.Usage` cannot express cost | `ports.go:100-107` | The input/output split is discarded at the adapter boundary (`anthropic.go:300`), so every call made before the fix is permanently uncosted in the append-only log. Cost of fixing scales with history, not with code. |
| Retry lives inside the adapter | `adapters/anthropic/retry.go` | Each new driver re-implements it. The Retry-After defect (`retry.go:42` uncalled) already demonstrates the failure mode **at N=1**; at N=3 there are three chances to repeat it. |
| Prompt rendering lives inside the adapter | `anthropic.go:320-372` | Once two drivers have their own prompts, failover silently changes semantics and the two outcomes are not comparable. Extracting later requires re-validating every prompt against recorded outputs. |
| Structured output is prompt-and-hope | `anthropic.go:374-392` — returns `{"text": raw}` on parse failure | Every provider has a *native* structured-output mode. Building three drivers on the prompt-append pattern means discarding three implementations later. |
| Nested retry compounding | `engine/failure.go:239` + `retry.go:104` | A step with `retry: {attempts: 3}` already issues up to **9 billable provider calls** (3 engine × 3 adapter). Multiplying this across providers before centralising retry multiplies real spend. |
| No health/circuit breaking | — | Required by B-7 for local providers. Retrofitting after the router is live means changing routing behaviour under running workflows. |

---

## 5. What is *not* a gap — constraints that turn out to be free

The brief required preserving workflows, DSL, engine behaviour, GUI behaviour, and tests. Code says four of those five are near-zero-cost constraints:

| Stated constraint | Actual cost | Evidence |
|---|---|---|
| Preserve existing workflows | **Free** | Workflow YAML names capabilities only (`apps/oip/workflows/*.yaml`, `examples/workflows/with-intelligence.yaml`). No provider, no model, nothing to migrate. |
| Preserve the DSL | **Free through Phase 2** | `internal/dsl/dsl.go:88-90` maps three neutral fields. Nothing needs to change until model identity lands (B-5). |
| Preserve engine behaviour | **Free** | `internal/engine` has no provider identifier. The one intelligence-aware branch — `capability_fallback` skips retry (`engine/failure.go:238-239`) — is correct and orthogonal. |
| Preserve GUI behaviour | **Free — and vacuously so** | `web/src` reads only `capability` and `model_hint` from workflow definitions (`types.ts:31-34`, `workflowDetail.ts:185-186`) and renders **no** adapter/model/token data anywhere (adversarially confirmed across all of `web/src`). Moreover `cmd/awis-server/main.go:78-82` never sets `Intelligence`, so **the GUI backend has never executed a real intelligence call.** There is no behaviour to regress. |
| Preserve existing tests | **Real, and valuable** | 32 production-behaviour tests over the seam (`00-ARCHITECTURE_REVIEW` §1.2 / `INTELLIGENCE_TRUTH_AUDIT` §1.2). The 11 router tests specify precisely the multi-provider semantics being built toward. **This is the only one of the five constraints with teeth, and it is an asset rather than a tax.** |

The practical upshot: the compatibility burden that would normally dominate a subsystem migration is, here, concentrated entirely in a test suite that already describes the target behaviour.

---

## 6. Gap summary

| # | Gap | Blocks | Severity | Fix locus |
|---|---|---|---|---|
| B-1 | Single-provider SDK arity | All six | **Blocking** | `sdk/runtime.go:33,86-89` |
| B-2 | Vendor-hardcoded composition root | All six | **Blocking** | `cmd/awis/start.go:36,141-152` |
| B-3 | Config allowlist rejects new keys | All six | **Blocking** | `cmd/awis/config.go:420-431` |
| B-4 | No `base_url` in config | Ollama, LM Studio, vLLM, OpenRouter | **Blocking** | config schema + `start.go` |
| B-5 | No model identity | Ollama, vLLM, OpenRouter (Gemini structurally) | **Blocking** | `core/step.go:87`, `ports.go:66,78` |
| B-6 | `ProviderName()` collides silently | Any 2 OpenAI-compatible instances | **Blocking** | `router.go:85-87` |
| B-7 | `IsAvailable()` excludes keyless providers | Ollama, LM Studio, vLLM | **Blocking** | `ports.go:57` (interface-level) |
| B-8 | Routing metadata not configurable | All six (silent degradation) | **Ships-with-B-1** | `router.go:24`, `sdk/runtime.go:87` |
| G-1 | Usage cannot express cost | — | Rewrite debt | `ports.go:100` |
| G-2 | Retry inside adapter | — | Rewrite debt | `adapters/anthropic/retry.go` |
| G-3 | Prompts inside adapter | — | Rewrite debt | `anthropic.go:320-372` |
| G-4 | Structured output unenforced | — | Rewrite debt | `anthropic.go:374-392` |
| G-5 | Nested retry compounding | — | Cost defect | `engine/failure.go:239` |
| G-6 | No health / circuit breaking | Ollama, LM Studio, vLLM | Follows B-7 | new |

**Eight blockers. Five of them (B-1, B-2, B-3, B-4, B-8) live in configuration and composition — not in the intelligence subsystem at all.** Only B-5, B-6, and B-7 require touching `internal/intelligence` or `internal/core`.

> **ID scheme note.** `B-*`/`G-*` are this document's blocker and debt identifiers, used throughout `IMPLEMENTATION_PROGRAM.md` and `FINAL_VERDICT.md`. The `D-*` identifiers used in `00-ARCHITECTURE_REVIEW.md` are a separate, earlier scheme; where both refer to the same defect the mapping is: D-08 = G-4 (structured output), D-11 = the live half of G-2 (Retry-After), D-15 = Phase 3.5 (budget placement), D-16 = Phase 2.3 (redaction).
