# AWIS Intelligence Subsystem — Architecture Review

**Scope:** provider-agnostic redesign of the intelligence layer.
**Authority:** Tier 1 = repository code as of `8a87f70` (branch `engine-hardening`). Tier 2 = AWIS architectural documents. Tier 3 = industry patterns, advisory only.
**Status:** architecture only. No code was written or modified. No implementation plan is proposed beyond migration sequencing.

> Governance note: `CLAUDE.md` freezes Baseline V1 implementation to Sonnet/Haiku and permits higher-tier models only for human-requested architecture reviews. This document is exactly that and produces no implementation.

---

## 0. Executive Summary

The seam is in the **right place** and the **wrong shape**.

Three things are already correct and must be preserved: workflow definitions reference AI by *capability only* (never provider, never model); `internal/core` contains no Anthropic type, constant, or import; and the EventLog records a provider-neutral `{adapter, model, tokens_used}` triple.

One thing is structurally broken: **the multi-provider machinery exists but is unreachable.** `CapabilityRouter`, `Dispatcher`, `Registration`, `ValidateChain` and the fallback-chain concept are all built and tested — and the only production constructor, `sdk.NewRuntime`, accepts exactly one provider and hard-codes a one-entry chain (`sdk/runtime.go:82-93`). Failover, cost-ranked routing, locality routing, and chain validation are all dead paths in every shipped binary. AWIS does not currently have a single-provider architecture that needs generalising; it has a multi-provider architecture with a single-provider front door nailed shut.

Beyond that, the port's *shape* cannot carry the target providers: a fixed method per capability, a `Capability` struct that is a bare name with no parameters, no model identity anywhere in the request path, a 3-value `model_hint` enum, compile-time-only registration metadata, `ProviderName()` used as the registry primary key (which collides the moment two OpenAI-compatible endpoints are configured), and a `Usage` triple from which cost cannot be computed.

**Verdict: EVOLVE the architecture; REPLACE `core.IntelligencePort`'s method set and `sdk.Config.Intelligence`'s arity.** Full justification in §6.

---

## 1. Current Intelligence Architecture

### 1.1 Component map

| Component | File | Role |
|---|---|---|
| `core.IntelligencePort` | `internal/core/ports.go:44` | The provider interface. 7 methods: `Draft`, `Embed`, `Synthesize`, `Classify`, `IsAvailable`, `Capabilities`, `ProviderName`. |
| `core.IntelReq` | `internal/core/step.go:87` | The step-level AI declaration: `{Capability, ModelHint, ContextBudget, Required}`. |
| `sdk.IntelligencePort` et al. | `sdk/intelligence.go` | Pure type aliases re-exporting `core`. Zero logic. |
| `intelligence.CapabilityRouter` | `internal/intelligence/router.go:76` | Selects an ordered candidate list from registrations ∩ fallback chain, filtered by capability + `IsAvailable()`, ordered by `model_hint`. |
| `intelligence.Registration` | `internal/intelligence/router.go:24` | Compile-time routing metadata: `{Adapter, Locality, CostRank, QualityRank}`. |
| `intelligence.Dispatcher` | `internal/intelligence/dispatcher.go:32` | Budget enforcement → routing → sequential try-next-on-failure. Only `Draft` and `Synthesize` have dispatch methods. |
| `intelligence.enforceBudget` | `internal/intelligence/budget.go:31` | `ceil(len(s)/4)` token estimate; reject, never truncate. |
| `intelligence.ValidateChain` | `internal/intelligence/validate_chain.go:22` | Construction-time chain check. **Never called from production code.** |
| `null.Adapter` | `internal/intelligence/adapters/null/null.go` | Degraded-mode adapter. `IsAvailable()` is permanently `false`. |
| `anthropic.Adapter` | `internal/intelligence/adapters/anthropic/anthropic.go` | The only real provider. stdlib `net/http`, Messages API, own retry policy, own prompt builders, own response parsers. |
| `runintel.Runner` | `internal/runner/intelligence/intelligence.go` | Engine `Runner` for `type=intelligence` steps. Maps step inputs → capability request, dispatch errors → typed `StepError`. |
| `porttest.Run` | `internal/intelligence/porttest/suite.go` | Adapter-agnostic contract suite. Correctly factored. |

### 1.2 Data flow (production, `awis start`)

```
config.yaml / ANTHROPIC_API_KEY
        │  cmd/awis/start.go:141-152 — env wins, then cfg["anthropic_api_key"];
        │  "$VAR" placeholder treated as absent
        ▼
  anthropic.New(Config{APIKey}) ──► core.IntelligencePort   (or nil ⇒ zero-AI)
        │
        ▼  sdk.NewRuntime — sdk/runtime.go:82-93
  port := cfg.Intelligence; if nil → null.New()
  NewRouter([]Registration{{Adapter: port, Locality: LocalityLocal}}, []string{port.ProviderName()})
        │                                    ▲                    ▲
        │                          ALWAYS "local"        ALWAYS length 1
        ▼
  Dispatcher ──► runners[StepTypeIntelligence] = runintel.New(disp)
        │
        ▼  engine tick ──► Runner.RunWithUsage (intelligence.go:60)
  switch req.Capability
    "draft"      → DraftRequest{Context: inputs["context"], Schema: step.Outputs}
    "synthesize" → SynthesisRequest{Query, Entries, MaxLen}
    default      → StepError{capability_unknown}, no dispatch
        │
        ▼  Dispatcher.Draft — dispatcher.go:56
  enforceBudget(dr.Context, req.ContextBudget)   ← pre-routing, provider-blind
  router.Eligible(capability, model_hint)        ← 1 candidate, always
  for each candidate: Adapter.Draft(...)         ← try-next never fires (n=1)
        │
        ▼  anthropic.Draft — anthropic.go:143
  buildDraftPrompt(req)  ← prompt semantics live INSIDE the adapter
  complete(ctx, cfg.ModelQuality, prompt, maxTokensDefault=1024)
  retry.Do → doRequest → POST /v1/messages, x-api-key, anthropic-version: 2023-06-01
  parseJSONOutput(text) — best effort; on failure returns {"text": <raw>}
        │
        ▼
  core.DraftResponse{Output, Usage{Adapter, Model, TokensUsed: in+out}}
        │
        ▼  UsageRunner side-channel (ADJ-8) — engine/tick.go:315
  StepCompleted payload += {adapter, model, tokens_used}   — engine/emit.go:43
        │
        ▼
  SQLite EventLog → awis trace / /api/v1 events
```

### 1.3 Dependency direction

Clean, and worth stating plainly: `core` → nothing. `intelligence` → `core`. `adapters/*` → `core`. `runner/intelligence` → `core`, `intelligence`. `sdk` → all of the above. `cmd/awis` → `sdk`, `adapters/anthropic`. The **only** production import of an Anthropic package outside its own directory is `cmd/awis/start.go:36`. That is the correct and intended coupling point for a composition root.

### 1.4 Coupling points — inventory

| # | Coupling | Location | Nature |
|---|---|---|---|
| C1 | Composition root imports the Anthropic package directly | `cmd/awis/start.go:36,151` | Intended (composition root), but hard-coded: adding a provider requires editing this file. |
| C2 | Credential key name is Anthropic-specific | `start.go:141-143`; `config.go:423` | Config schema hard-codes `anthropic_api_key` / `ANTHROPIC_API_KEY`. |
| C3 | Single-port arity | `sdk/runtime.go:33,82-93` | The public SDK cannot express more than one provider. **Primary defect.** |
| C4 | `LocalityLocal` stamped unconditionally | `sdk/runtime.go:87` | A cloud provider is registered as local. |
| C5 | `CostRank`/`QualityRank` left at zero | `sdk/runtime.go:87` | Cost/quality routing inert. |
| C6 | Capability names hard-coded in three packages | `validate.go:489`; `router.go:98-104`; `intelligence.go:63,75,90` | Adding a capability is a cross-package edit. |
| C7 | `model_hint` closed enum | `validate.go:476`; `router.go:141-172` | `fast|quality|local` only. |
| C8 | Prompt construction inside the adapter | `anthropic.go:320-372` | Prompt semantics are provider-private → failover changes the prompt silently. |
| C9 | Retry policy inside the adapter | `adapters/anthropic/retry.go` | Every future provider re-implements transport resilience. |
| C10 | Secret masking inside the adapter, `sk-ant-…` prefix | `anthropic.go:406` | Provider-shaped redaction at the wrong layer. |

### 1.5 Anthropic leakage assessment (objective #3)

| Layer | Leak? | Evidence |
|---|---|---|
| Runtime / engine | **No** | `internal/engine` and `internal/core` contain no Anthropic identifier. `Usage.Adapter` is an opaque string. |
| Workflow definitions | **No** | `apps/oip/workflows/*.yaml`, `examples/workflows/with-intelligence.yaml` declare `capability:` + `context_budget:` only. No provider, no model. |
| DSL | **No** | `internal/dsl/dsl.go:88-90` carries `capability / model_hint / context_budget` — all neutral. |
| Storage | **No** | `{adapter, model, tokens_used}` are neutral strings/ints (`engine/emit.go:43-51`; `docs/EVENTLOG_FORMAT.md:68`). |
| API responses | **No** | `internal/api` has no provider-specific field. `web/src/types.ts:33` surfaces `model_hint` only. |
| **Configuration** | **Yes** | `anthropic_api_key` is a first-class recognised key (`config.go:423`) and the sole credential path (`start.go:143`). |
| **Composition root** | **Yes** | `start.go:36,151` — by design, but not extensible. |
| **Docs** | **Yes** | `docs/PROVIDERS.md` step 4 of "Adding a New Provider" instructs editing `start.go`, i.e. the extension model is *recompile the CLI*. |

**Conclusion:** vendor coupling is confined to configuration and the composition root. The durable artefacts — workflow definitions and the EventLog — are already provider-neutral. This is the single most valuable property in the system and it is not at risk.

---

## 2. Architecture Defect Report

Severity: **S1** blocks the multi-provider goal outright · **S2** forces a breaking change later if not addressed now · **S3** correctness/operability defect within current scope · **S4** hygiene.

### S1 — Blocking

**D-01 · The SDK admits exactly one provider.**
`sdk/runtime.go:82-93` constructs `NewRouter` with a one-element registration slice and a one-element chain derived from `port.ProviderName()`. `sdk.Config` (`runtime.go:33`) has no plural form. Consequence: the fallback chain has length 1 in every production process, so `Dispatcher`'s try-next-on-failure loop (`dispatcher.go:65-72`) can never iterate, `ValidateChain` has nothing to validate, and every routing behaviour tested in `router_test.go` is unreachable outside tests. **Impact:** failover, provider mixing, local+cloud tiering, and cost routing are all vaporware today. **Scaling:** this is the one defect that must be fixed before any other work has value.

**D-02 · `ProviderName()` is the registry primary key, and collides silently.**
`router.go:85-87` builds `byName[r.Adapter.ProviderName()] = r` with no duplicate check; a second registration of the same provider *type* overwrites the first without error. The target list contains at least six OpenAI-compatible endpoints (OpenRouter, Ollama, LM Studio, vLLM, local endpoints, OpenAI itself) that would naturally share a driver and therefore a name. **Impact:** "route to local vLLM, fall back to OpenRouter" is inexpressible. **Scaling:** provider *instance identity* must be separated from provider *type* before any registry is built.

**D-03 · No model identity anywhere in the request path.**
`IntelReq` (`step.go:87`) carries `ModelHint ∈ {fast, quality, local}`. Nothing else names a model. The Anthropic adapter resolves the model internally and — despite `docs/PROVIDERS.md`'s mapping table — ignores the hint entirely: `Draft` and `Synthesize` both hard-code `cfg.ModelQuality` (`anthropic.go:145,167`); `cfg.ModelFast` is used only by `Classify` (`anthropic.go:181`), which is not dispatchable. **Impact:** a 3-value enum cannot address OpenRouter's hundreds of models, arbitrary Ollama tags, or a vLLM server's single served model. **Scaling:** `model_hint` is the wrong abstraction and must be replaced, not extended.

**D-04 · Capability is a fixed method per capability.**
`IntelligencePort` (`ports.go:44`) declares `Draft`/`Embed`/`Synthesize`/`Classify` as distinct methods with distinct request types. Adding rerank, vision, tool-use, or long-context summarisation is an **interface change that breaks every adapter simultaneously**. `Capability` (`ports.go:131`) is `struct{ Name string }` — it carries no context window, no max output, no modality, no parameter support, no pricing. Capability *discovery* is therefore name-only and cannot answer "can this provider handle a 180k-token context?". **Impact:** the extension model is closed. **Scaling:** this is the interface-shape decision the whole target architecture turns on.

**D-05 · Registration metadata is compile-time Go, not configuration.**
`Registration{Locality, CostRank, QualityRank}` (`router.go:24`) is a Go struct populated at call sites. There is no config path to it; `sdk/runtime.go:87` therefore stamps `LocalityLocal` on everything and leaves both ranks at zero. **Impact:** the `local` hint is meaningless (everything is local), and `fast`/`quality` degrade to chain order because their branches require `LocalityCloud` members (`router.go:150-172`). All three hints are currently inert. Changing routing requires recompiling.

### S2 — Forces a later breaking change

**D-06 · `Usage` cannot express cost.**
`Usage{Adapter, Model, TokensUsed}` (`ports.go:100`) sums input and output tokens (`anthropic.go:300`). Input and output are priced differently by every provider, so **cost is unrecoverable from the recorded data** — including retrospectively, because the sum is what lands in the immutable EventLog. Missing: input/output split, cache-read/cache-write tokens, reasoning tokens, request latency, provider request-id, and the resolved provider *instance*. Objective #7's "Cost Tracking" is not merely unimplemented; the recorded schema forecloses it. Fixing it later means an EventLog payload amendment.

**D-07 · No request parameters.**
`DraftRequest` (`ports.go:66`) is `{Context, Schema, Persona, Examples}`. There is no temperature, max output tokens, system prompt, stop sequence, seed, response-format/JSON mode, tool definition, or timeout. `Draft` hard-codes `maxTokensDefault = 1024` (`anthropic.go:48,145`). `Synthesize` overloads `MaxLen` — documented as "maximum output length" — directly as the provider's `max_tokens` (`anthropic.go:164-167`), conflating characters and tokens. **Impact:** determinism controls and output-length control are unavailable to workflow authors; providers with mandatory parameters cannot be driven.

**D-08 · Structured output is prompt-and-hope.**
`buildDraftPrompt` appends the schema as prose (`anthropic.go:320-343`); `parseJSONOutput` strips fences, attempts `json.Unmarshal`, and on failure **returns `{"text": <raw>}`** (`anthropic.go:374-392`). The step's declared `outputs` schema is never enforced. A step declaring `{draft: string, confidence: number}` can silently receive `{text: "…"}`, and downstream expression evaluation then fails far from the cause. Every provider has a different native structured-output mechanism (Anthropic tool-use, OpenAI `response_format`/strict schemas, Gemini `responseSchema`, Ollama `format: json`), so per-adapter prompt hacking will diverge without bound.

**D-09 · Prompt construction is provider-private.**
`buildDraftPrompt` / `buildSynthesisPrompt` / `buildClassifyPrompt` live inside the Anthropic adapter (`anthropic.go:320-372`). A failover from Anthropic to a local model would silently send a *different prompt*, making the two outcomes non-comparable and the step's behaviour a function of which provider happened to be up. There is also no record of what was actually sent: the EventLog stores outputs and usage, never the rendered request. **Impact:** intelligence steps are not auditable, and failover is not semantically transparent.

**D-10 · `IsAvailable()` is a credential check, not a health check.**
`anthropic.go:124` returns `cfg.APIKey != ""` and explicitly performs no round-trip. The router calls it per request (`router.go:134`). So availability never reflects reachability: a provider that is down, rate-limited, or out of quota still reports available, and unavailability is discovered only by a failed call. There is no circuit breaker, no health cache, no cooldown, no rate limiter, and no concurrency cap. For local endpoints (Ollama, LM Studio) — which are frequently simply *not running* — a per-request probe is required, and the current interface offers nowhere to put one that isn't on the hot path.

### S3 — Correctness / operability

**D-11 · `Retry-After` is never honoured.** `parseRetryAfter` (`retry.go:42`) has no non-test caller. `doRequest` builds the error with `httpError(resp.StatusCode, raw)` (`anthropic.go:281`), which never sees `resp.Header` and never sets `retryAfter` (`anthropic.go:308-316`). The guard at `retry.go:132` is therefore always false and backoff is always exponential. `docs/PROVIDERS.md` documents the opposite. This matters generically: 429 handling is a property every provider needs and it is currently implemented at the wrong layer *and* broken.

**D-12 · The `intelligence:` config key is recognised but ignored.** `config.go:427` accepts it, `config.go:53` marks it printable, `config.go:453` advertises it in the scaffold — and `start.go` never reads it (`cfg[…]` is read only for `tick`, `namespace`, `anthropic_api_key`). Setting `intelligence: null` on a machine with `ANTHROPIC_API_KEY` exported silently does nothing. This is precisely the silent-defaults class closed as B-31 elsewhere in the repo, recurring in the intelligence config surface.

**D-13 · `ValidateChain` is dead in production.** Its null-adapter-must-be-last invariant (`validate_chain.go:35-42`) is never enforced at runtime because no production path builds a chain longer than one.

**D-14 · Two of four interface methods have no dispatch path.** `Embed` (CONTRA-3) and `Classify` (FR-IL-10) are implemented by every adapter, accepted by the validator (`validate.go:489`), and rejected at execution with `capability_unknown` (`intelligence.go:90-95`). A workflow declaring `capability: embed` validates clean and fails at run time. Documented as intentional; the cost is that every future adapter must implement two dead methods.

**D-15 · The token estimator is byte-based and layer-misplaced.** `estimateTokens` is `(len(s)+3)/4` over **bytes** (`budget.go:23`), so CJK and emoji-heavy contexts are overestimated by ~3×. More structurally: the budget is enforced *before routing* (`dispatcher.go:57`), so it cannot know the selected model's context window — the one number that actually matters. A per-step `context_budget` is a workflow-author guess about a model the workflow is not allowed to name.

### S4 — Hygiene

**D-16** · `mask()` (`anthropic.go:406`) has no non-test caller — a dead safety net whose `sk-ant-…` prefix is provider-specific. Redaction belongs at the config/seam layer where it can cover every credential shape.
**D-17** · `modelFast` / `modelQuality` (`anthropic.go:38-39`) are unused const aliases of the defaults.
**D-18** · `docs/PROVIDERS.md` misstates behaviour in three places: the `model_hint`→model mapping (D-03), `Retry-After` honouring (D-11), and "requesting `embed` … the router falls through to the step fallback" (the runner rejects `embed` before any router involvement, D-14).

---

## 3. Provider-Agnostic Target Architecture

### 3.1 The organising insight

The nine named targets are not nine providers. They are **four wire protocols and N configured endpoints**:

| Wire protocol (driver) | Instances it serves |
|---|---|
| `anthropic-messages` | Anthropic |
| `openai-chat` (+`openai-responses`) | OpenAI, OpenRouter, vLLM, LM Studio, Together/Groq/Fireworks, any local OpenAI-compatible endpoint |
| `google-genai` | Gemini |
| `ollama-native` | Ollama (its OpenAI-compatible surface also works via `openai-chat`) |

Six of the nine targets collapse into one driver differing only by `base_url`, auth header, and model catalogue. Therefore the extension unit that must be **configurable at runtime** is the *provider instance*, and the extension unit that may remain **compiled in** is the *driver*. Today AWIS conflates the two into "adapter", which is why adding a provider means editing `start.go` and recompiling.

### 3.2 Layered component model

```
┌────────────────────────────────────────────────────────────────────────┐
│ WORKFLOW DEFINITION (durable, versioned, provider-neutral)             │
│   intelligence: { capability, requires{…}, policy?, pin? }             │
└───────────────────────────────┬────────────────────────────────────────┘
                                │ CapabilityRequest (neutral)
┌───────────────────────────────▼────────────────────────────────────────┐
│ ① INTELLIGENCE SERVICE  (the seam — one object, replaces the port      │
│    as sdk.Config's field type)                                         │
│    Invoke(ctx, CapabilityRequest) (CapabilityResponse, error)          │
└───────────────────────────────┬────────────────────────────────────────┘
                                │
┌───────────────────────────────▼────────────────────────────────────────┐
│ ② POLICY ENGINE — resolves request+policy → ordered candidate list      │
│    named policies are CONFIG DATA, not Go structs                      │
│    inputs: capability, requires{ctx_window,max_out,modality,locality,  │
│            determinism,residency}, policy name, budget state           │
│    output: []Candidate{instance_id, model_id, rank_reason}             │
└───────────────────────────────┬────────────────────────────────────────┘
                                │
┌───────────────────────────────▼────────────────────────────────────────┐
│ ③ MODEL CATALOG — the fact base the policy engine queries              │
│    ModelDescriptor{instance_id, model_id, capabilities[],              │
│      context_window, max_output, modalities[], params_supported[],     │
│      pricing{in,out,cache_read,cache_write}, locality, tier}           │
│    sources: config (authoritative) ← optional /models discovery         │
└───────────────────────────────┬────────────────────────────────────────┘
                                │
┌───────────────────────────────▼────────────────────────────────────────┐
│ ④ INVOKER — everything provider-INDEPENDENT, executed once, centrally  │
│    prompt rendering · schema/response-format negotiation · timeouts    │
│    retry+Retry-After · rate limit · circuit breaker · failover walk    │
│    redaction · usage normalisation + COST computation · request audit  │
└───────────────────────────────┬────────────────────────────────────────┘
                                │ DriverRequest (still neutral)
┌───────────────────────────────▼────────────────────────────────────────┐
│ ⑤ DRIVERS — wire translation ONLY, no policy, no retry, no prompts     │
│    anthropic-messages │ openai-chat │ google-genai │ ollama-native     │
│    Translate(DriverRequest) → HTTP → Translate(response) → Usage       │
└───────────────────────────────┬────────────────────────────────────────┘
                                │
┌───────────────────────────────▼────────────────────────────────────────┐
│ ⑥ PROVIDER INSTANCE REGISTRY — CONFIG-DRIVEN, no recompilation         │
│    ProviderInstance{id, driver, base_url, credential_ref, headers,     │
│                     models[], locality, limits{rps,concurrency}}       │
└────────────────────────────────────────────────────────────────────────┘
```

### 3.3 Responsibility boundaries (the rule that keeps it neutral)

> **A driver may know its wire format and nothing else. Everything a second provider would also need belongs above the driver.**

Applying that rule moves five things out of today's adapter: retry (D-11), prompt rendering (D-09), structured-output negotiation (D-08), redaction (D-16), and cost/usage normalisation (D-06). What remains in a driver is roughly 150 lines of request/response marshalling — which is why adding Gemini should not be a milestone-sized task.

### 3.4 The interface change

Replace the four capability methods with one dispatch method plus one descriptor method:

```
Invoke(ctx, CapabilityRequest) (CapabilityResponse, error)
Describe() ProviderDescriptor      // instance id, driver, models[], limits
```

`CapabilityRequest` carries `{kind, messages/context, schema, params, model_id}`; `CapabilityResponse` carries `{content, structured, usage, provider_meta}`. Adding a capability kind then becomes **data** (a new kind + schema, drivers that don't support it decline via `Describe()`), not an interface break. This is the direct answer to D-04.

`Capability{Name}` is replaced by `ModelDescriptor` — the same word, but carrying the facts a router actually needs.

### 3.5 Answers to objective #7

| Concern | Target design |
|---|---|
| **Provider registration** | Config-driven `ProviderInstance` entries keyed by an operator-chosen `id`, distinct from `driver`. Resolves D-02. |
| **Provider discovery** | Two-tier: config is authoritative; optional `/models` probe (OpenAI `/v1/models`, Ollama `/api/tags`, OpenRouter's catalogue) enriches descriptors at startup and is cached. Never required — an air-gapped vLLM with a static config must work. |
| **Provider configuration** | Structured `providers:` block, per-instance `base_url` / `credential_ref` / `headers` / `models` / `limits`. Credentials are **references** (`env:OPENAI_API_KEY`, `file:…`), never literals in config. Replaces the flat `anthropic_api_key` key (C2, D-12). |
| **Capability discovery** | `ModelDescriptor.capabilities[]` + declared limits, queryable by the policy engine. Fixes name-only discovery (D-04). |
| **Provider selection** | Policy engine over the catalog. Ordered candidates, not a single winner. |
| **Model selection** | Policy engine selects `(instance, model)` **as one unit**. Workflows never name a model (§3.6). |
| **Failover** | Invoker walks the candidate list; classifies errors as retry-same / failover-next / terminal; records the walk in the EventLog. Failover across *providers* re-renders the prompt through the shared renderer, so semantics are preserved (fixes D-09). |
| **Cost tracking** | `Usage` extended to `{instance_id, driver, model, tokens_in, tokens_out, cache_read, cache_write, reasoning, latency_ms, request_id}`; cost computed in the invoker from catalog pricing and recorded alongside. Additive EventLog amendment (fixes D-06). |
| **Local models** | `locality: local` on the instance; health probing mandatory (local endpoints are routinely down); zero-cost pricing; policies may prefer or require local for residency. Fixes D-05/D-10. |
| **OpenAI-compatible endpoints** | Not a special case — the ordinary configuration of the `openai-chat` driver with a `base_url`. This is the payoff of separating driver from instance. |

### 3.6 How workflows reference AI (objective #8)

**Recommendation: capability + declarative requirements + optional named policy. Never a provider. Never a model.**

```yaml
intelligence:
  capability: generate_structured      # what kind of work
  requires:                            # what the work needs (facts, not vendors)
    context_tokens: 8000
    max_output_tokens: 1200
    locality: any                      # any | local | cloud
  policy: default                      # optional; named in config, not defined here
  required: true
```

Rationale: a workflow definition is a **durable, versioned, content-addressed artefact**. A model ID inside it is a time bomb — models are deprecated on the provider's schedule, not the workflow's, and this repository has already been burned once by hard-coded model identifiers (`VERIFIED_DEFECT_REGISTER.md` D-05). Requirements are stable facts about the work; provider and model are volatile facts about the deployment, and belong in config where an operator can change them without minting a new workflow version.

`requires.context_tokens` also relocates the budget correctly: today `context_budget` is enforced pre-routing against a number the author guessed (D-15). As a *requirement*, it becomes a routing input — the policy engine filters to models whose `context_window` can hold it, and enforcement uses the selected model's real limit.

An explicit `pin: {instance, model}` escape hatch should exist for reproducibility-critical steps, be validated as a warning, and be recorded in the event payload — so pinning is possible, visible, and rare.

### 3.7 Is a router layer required? (objective #9)

**Yes — and one already exists.** `CapabilityRouter` is not the problem. What is missing is a **policy layer above it** (named, config-defined routing rules instead of a 3-value enum) and a **provider instance registry below it** (config-driven, instance-keyed). Given a real registry and real descriptors, today's `Eligible`/`Route` shape — ordered candidate list, chain as the selection universe, stable tie-breaking — is a sound core and should be kept. The router is the one component of the current design that survives largely intact.

---

## 4. Migration Strategy

### Stage 0 — Unblock arity *(small, additive, no interface change)*

Add `Config.Providers []ProviderRegistration` and `Config.FallbackChain []string` to `sdk.Config`; keep `Config.Intelligence` as a deprecated single-provider shim that desugars to a one-element registration. Call `ValidateChain` from `NewRuntime` (fixes D-13). Set `Locality` from the registration rather than hard-coding `LocalityLocal` (D-04-adjacent, C4). Reject duplicate `ProviderName()` in `NewRouter` with an error instead of overwriting (partial D-02).

*Why first:* every routing behaviour already built and tested becomes reachable. This is the highest value-per-line change in the whole programme, and it is purely additive.
*Risk:* low. `sdk.Config` is post-M08 frozen surface — this is additive, but the freeze needs a documented amendment (the ADJ-7/ADJ-8 precedent applies).

### Stage 1 — Configuration & instance identity

Introduce the structured `providers:` config block with `id`/`driver`/`base_url`/`credential_ref`. Add instance-ID identity distinct from `ProviderName()`, resolving D-02. Move credential resolution and redaction to a shared config layer (D-16), and make `intelligence:` either functional or removed (D-12). Retire the `anthropic_api_key` special case behind a compatibility mapping.

*Risk:* medium — touches the config schema, `config show` masking, and the audit-row path. The existing allowlist-of-the-harmless masking design (`config.go:47-56`) is correct and must be preserved through the change; it is the reason a `base_url`-and-`authorization` config surface is safe to add at all.

### Stage 2 — Invoker & second driver

Extract retry, timeout, Retry-After (D-11), rate limiting, circuit breaking, prompt rendering (D-09), structured-output negotiation (D-08), and usage/cost normalisation (D-06) into a provider-independent invoker. Then implement the `openai-chat` driver. Extend `Usage` and amend the `StepCompleted` payload additively.

*Why this order:* the second driver is the forcing function that proves the boundary. Writing it before the extraction guarantees a second copy of the retry bug.
*Risk:* medium-high — this is where the Anthropic adapter's behaviour changes shape. `porttest.Run` must be extended into a driver conformance suite first, and both drivers must pass it against recorded fixtures (the existing no-network-in-CI rule, IMP P7, holds).

### Stage 3 — Interface replacement

Replace the four capability methods with `Invoke`/`Describe` (D-04, D-07). Retain a shim that presents the old four-method interface over the new one so external SDK consumers are not broken at the same moment. Extend `IntelReq` with `requires{}` and `policy`, keeping `model_hint` as a deprecated alias that maps onto a built-in policy (D-03).

*Risk:* high — this is the breaking change. It must be a single deliberate version boundary with the shim, not a drift.

### Stage 4 — Policy engine, catalog, cost

Model catalog with optional discovery; named policies as config; failover walk recorded in the EventLog; cost computation and reporting; health probing for local endpoints (D-10). Retire `model_hint`.

### Cross-cutting risks

| Risk | Mitigation |
|---|---|
| EventLog payload changes are immutable once written | Extend `Usage` **once**, in Stage 2, with every field §3.5 names — including ones not yet computed. A second amendment is far more expensive than an over-provisioned first one. |
| `sdk` surface is frozen post-M08 | Stages 0/1 are additive; Stage 3 is the one intentional break. Bundle it with a documented version boundary. |
| Fixture-only CI cannot catch wire-format drift across four providers | Per-driver recorded fixtures plus one gated live smoke per driver, following the existing `ANTHROPIC_LIVE` pattern — and *actually run at least once*, which `VERIFIED_DEFECT_REGISTER.md` D-05 records as the residual risk that produced a shipped 404. |
| Prompt centralisation changes existing outputs | Snapshot current Anthropic prompts as the initial renderer templates; assert byte-equality in Stage 2 so the extraction is provably behaviour-preserving before Stage 4 changes anything. |
| Scope creep into agents/tool-use/streaming | Explicitly out of scope until Stage 4 completes. `CapabilityRequest` should be *shaped* to admit them (message list, tool defs, streaming flag) without implementing them. |

---

## 5. Decision Record

**DR-1 · Provider abstraction unit — driver vs adapter.**
*Alternatives:* (a) keep one adapter per provider; (b) one OpenAI-compatible adapter plus per-vendor adapters; (c) **driver (wire protocol) + configured instance**.
*(a)* Pros: simple, matches today. Cons: six near-identical adapters for the OpenAI-compatible targets; adding an endpoint requires a code change.
*(b)* Pros: less duplication. Cons: hides the instance-identity problem (D-02) rather than solving it.
*(c)* Pros: collapses six of nine targets into one driver; makes endpoint addition a config edit; separates volatile deployment facts from compiled code. Cons: two concepts where there was one; requires instance IDs everywhere including the EventLog.
**→ (c).** The stated goal is "support new providers without engine changes"; only (c) actually delivers that for the named list.

**DR-2 · Interface shape — methods vs single dispatch.**
*Alternatives:* (a) keep four methods; (b) add methods as capabilities grow; (c) **`Invoke(CapabilityRequest)` + `Describe()`**.
*(a)* Pros: type-safe, readable, already tested. Cons: closed to extension; two of four are already dead (D-14).
*(b)* Pros: keeps type safety. Cons: every addition breaks every adapter — a cost that grows linearly in providers and is paid by third parties.
*(c)* Pros: capability kinds become data; drivers decline unsupported kinds via `Describe()`; one code path for retry/usage/audit. Cons: loses compile-time typing at the boundary (mitigated by typed constructors + the conformance suite); a genuine breaking change.
**→ (c),** at a deliberate version boundary with a shim. This is the only defect on the list that cannot be fixed additively, which is exactly why it should be decided now rather than after three more drivers exist.

**DR-3 · How workflows reference AI.**
*Alternatives:* (a) `provider + model`; (b) **capability + requirements + optional named policy**; (c) capability only (today); (d) routing policy expression inline.
*(a)* Pros: explicit, reproducible. Cons: puts a volatile vendor fact in a durable versioned artefact; model deprecations then break stored workflows — a failure mode already recorded in this repo.
*(c)* Pros: maximally neutral. Cons: cannot express "needs 180k context" or "must stay local"; forces all nuance into global config.
*(d)* Pros: maximum flexibility. Cons: turns workflow YAML into a routing DSL; policy churn forces workflow re-versioning.
*(b)* Pros: workflow declares stable facts about the work, config owns volatile facts about the deployment; policies change without re-versioning workflows; requirements double as routing inputs (fixing D-15). Cons: needs an accurate catalog; adds a `requires` block to the DSL.
**→ (b),** with a discouraged, recorded `pin:` escape hatch.

**DR-4 · Where retry / rate-limit / circuit-breaking live.**
*Alternatives:* (a) in each driver (today); (b) **in a shared invoker**; (c) in the engine's generic step retry.
*(a)* Cons: N implementations, N chances to repeat D-11 — which has already happened once at N=1.
*(c)* Cons: engine retry is step-level and cannot distinguish retry-same-provider from failover-next; it also cannot see HTTP semantics.
*(b)* Pros: one correct implementation; drivers only classify their errors into a neutral taxonomy.
**→ (b),** with drivers supplying an error classifier.

**DR-5 · Prompt construction location.**
*Alternatives:* (a) in the driver (today); (b) **in a shared renderer above the driver**; (c) in the workflow definition.
*(a)* Cons: failover silently changes semantics (D-09); prompts are untestable across providers.
*(c)* Cons: leaks model-craft into durable artefacts and makes every prompt improvement a workflow version bump.
*(b)* Pros: one prompt per capability across all providers; failover is semantically transparent; the rendered request becomes auditable. Cons: loses provider-specific prompt tuning — recoverable via optional per-driver template overrides, which should be the rare exception.
**→ (b).**

**DR-6 · Usage & cost schema.**
*Alternatives:* (a) keep `{adapter, model, tokens_used}`; (b) add cost as a separate later event; (c) **extend `Usage` once, comprehensively, in Stage 2**.
*(a)* Cons: cost is permanently unrecoverable from history (D-06).
*(b)* Cons: two sources of truth over an append-only log; reconciliation forever.
*(c)* Cons: one EventLog payload amendment. Pros: it is additive, replay-neutral (the EDR-007 projection reads only `outputs`), and follows the ADJ-8 precedent exactly.
**→ (c).** Over-provision the field set; a second amendment costs far more than an unused column.

**DR-7 · Health and availability.**
*Alternatives:* (a) `IsAvailable()` as a credential check (today); (b) synchronous probe per request; (c) **background health cache + circuit breaker, with `IsAvailable()` reading the cache**.
*(b)* Cons: doubles latency and hammers local endpoints.
*(c)* Pros: local endpoints (routinely down) are correctly excluded; failed providers are skipped without paying a timeout; hot path stays synchronous and cheap. Cons: introduces background state into a runtime that is otherwise pull-based and stateless — must be per-process and must not be persisted.
**→ (c).**

**DR-8 · Keep or discard `CapabilityRouter`.**
*Alternatives:* (a) discard and rewrite; (b) **keep the selection core, add policy above and registry below**.
**→ (b).** Its contracts — chain-as-selection-universe, ordered candidates, stable tie-breaks, typed `CapabilityUnavailableError`/`FallbackSignal` — are sound and well tested (`router_test.go`, 419 lines). The defects are in what feeds it, not in what it does.

---

## 6. Final Verdict

# **EVOLVE CURRENT ARCHITECTURE**

— with one bounded, deliberate replacement: `core.IntelligencePort`'s method set (DR-2) and `sdk.Config.Intelligence`'s arity (Stage 0).

### Why not REPLACE

Replacement would discard three properties that are correct, hard to get right, and already paid for:

1. **Workflow definitions are already provider-neutral.** `capability: draft` + `context_budget` and nothing else (`apps/oip/workflows/*.yaml`, `internal/dsl/dsl.go:88-90`). This is the property that determines whether stored workflows survive a provider change, and it is the single hardest thing to retrofit — a replacement design would have to re-derive it and would risk getting it wrong.
2. **The dependency direction is clean.** `internal/core` and `internal/engine` contain no Anthropic identifier; the sole production import outside the adapter directory is the composition root (`cmd/awis/start.go:36`). The seam is *in the right place*; only its shape is wrong.
3. **The EventLog is neutral and the router core is sound.** `{adapter, model, tokens_used}` needs extension, not replacement. `CapabilityRouter` needs better inputs, not a rewrite.

### Why not KEEP

Five S1 defects each independently block the stated goal, and they are not incremental gaps — they are structural:

- The SDK cannot express more than one provider (D-01), so **nothing** in the existing routing layer executes in production.
- `ProviderName()` as the registry key collides across OpenAI-compatible endpoints (D-02) — the exact configuration six of the nine named targets require.
- No model identity exists in the request path; `model_hint` is a 3-value enum (D-03).
- Capability is a compiled method, so extension breaks every adapter (D-04).
- Routing metadata is compile-time Go, so all three hints are inert today (D-05).

"Keep" would mean shipping a provider abstraction whose provider-selection behaviour has never once executed outside a test.

### The honest summary

AWIS did the difficult architectural work correctly and then wired it to a single hard-coded provider. The gap between the design in `internal/intelligence/` and the runtime in `sdk/runtime.go:82-93` is the whole finding: **eleven lines of composition are the reason a multi-provider system behaves as a single-vendor one.** Stage 0 closes that gap and is small. The remaining stages are about giving that machinery inputs worth routing on — a config-driven instance registry, model descriptors with real facts, an invoker that owns everything a second provider would otherwise duplicate, and a `Usage` record from which cost can actually be computed.

Sequence the interface replacement (Stage 3) as a single deliberate boundary. Do it before three more drivers exist, not after.
