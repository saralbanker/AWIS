# INTELLIGENCE_TRUTH_AUDIT

**Repository truth as of `8a87f70` (branch `engine-hardening`).**
Authority: production Go code only. `_test.go` and `./archive` excluded from claims about behaviour. Documentation treated as advisory and, where it conflicts with code, **overruled and recorded as drift**.

---

## 0. The one-sentence truth

AWIS has a **competently designed, well-tested multi-provider intelligence subsystem** whose selection machinery **never executes** in any shipped binary, because the only public constructor accepts one provider and builds a one-element fallback chain — and because two of the three production composition roots don't configure a provider at all.

---

## 1. What actually exists

### 1.1 Inventory — built vs reachable

"Built" = the code exists and is tested. "Reachable" = it can execute in a shipped binary given some legal configuration.

| # | Component | File | Built | Reachable | Note |
|---|---|---|---|---|---|
| 1 | `core.IntelligencePort` (7 methods) | `internal/core/ports.go:44` | ✅ | ✅ | Shape is closed (§2.3) |
| 2 | Type aliases into `sdk` | `sdk/intelligence.go` | ✅ | ✅ | Zero logic; pure re-export |
| 3 | `CapabilityRouter` eligibility + hint ordering | `internal/intelligence/router.go:121` | ✅ | ⚠️ trivially | Executes, but over a 1-element set |
| 4 | Chain-as-selection-universe rule | `router.go:69-79` | ✅ | ⚠️ trivially | Universe is always size 1 |
| 5 | `Registration{Locality,CostRank,QualityRank}` | `router.go:24` | ✅ | ❌ | No config path; hardcoded at `sdk/runtime.go:87` |
| 6 | `Dispatcher` budget→route→try-next | `dispatcher.go:56,87` | ✅ | ⚠️ partial | Loop body runs once, always |
| 7 | Try-next-on-failure failover | `dispatcher.go:65-72` | ✅ | ❌ | Cannot iterate (§1.3) |
| 8 | `ValidateChain` null-last invariant | `validate_chain.go:22` | ✅ | ❌ | **No production caller** |
| 9 | Budget enforcement (reject, never truncate) | `budget.go:31` | ✅ | ✅ | Byte-based estimator (§2.6) |
| 10 | `NullAdapter` degraded mode | `adapters/null/null.go` | ✅ | ✅ | `IsAvailable()` permanently false |
| 11 | `AnthropicAdapter` | `adapters/anthropic/anthropic.go` | ✅ | ✅ | Only real provider |
| 12 | Adapter-local retry (429/5xx) | `adapters/anthropic/retry.go` | ✅ | ⚠️ partial | Retry-After dead (§2.5) |
| 13 | `runintel.Runner` + typed error mapping | `runner/intelligence/intelligence.go` | ✅ | ✅ | Dispatches 2 of 4 capabilities |
| 14 | `capability_fallback` skips engine retry | `engine/failure.go:238-239` | ✅ | ✅ | Correct, verified |
| 15 | ADJ-8 usage triple in `StepCompleted` | `engine/emit.go:43-51` | ✅ | ✅ | Cost-incapable (§2.7) |
| 16 | `porttest` adapter-agnostic contract suite | `intelligence/porttest/suite.go` | ✅ | ✅ | Correctly factored, reusable |
| 17 | Config-driven provider registry | — | ❌ | ❌ | Does not exist |
| 18 | Model catalog / descriptors | — | ❌ | ❌ | Does not exist |
| 19 | Cost model | — | ❌ | ❌ | Does not exist |
| 20 | Health / circuit breaking | — | ❌ | ❌ | Does not exist |
| 21 | Shared prompt rendering | — | ❌ | ❌ | Lives inside the adapter |
| 22 | Second driver of any kind | — | ❌ | ❌ | Does not exist |

### 1.2 Test coverage over the seam — 32 production-behaviour tests

| Suite | Count | What it locks down |
|---|---|---|
| `router_test.go` | 11 | Hint ordering (local/fast/quality/empty), chain-as-universe, availability at call time, null never selected, unknown chain entry |
| `dispatcher_test.go` | 3 | Chain walk, exhausted-required, exhausted-fallback |
| `budget_test.go` | 3 | Boundary, budget-before-call, synthesize-budgets-query |
| `validate_chain_test.go` | 5 | Unregistered entry, null-not-last, null-last-OK, no-null-OK, empty-OK |
| `runner/intelligence` | 5 | Fallback, unavailable, capability-unknown-no-dispatch, nil config, success+usage |
| `engine/intelligence_test.go` | 3 | Fallback→manual routing, required-unavailable fails workflow, ADJ-8 payload |
| `cmd/awis/start_intel_test.go` | 2 | Startup intelligence level with/without key |
| `porttest` contract | 8 subtests | Adapter-agnostic port conformance |

**This is the single strongest argument against replacement.** The router's *semantics* — the hard part — are specified by 11 tests that describe exactly the behaviour a multi-provider system needs. Those tests are the asset. The wiring is the liability.

### 1.3 Dead / unreachable production code — proven

| Item | Location | Proof of unreachability |
|---|---|---|
| Failover loop iteration ≥2 | `dispatcher.go:65-72` | Chain is `[]string{port.ProviderName()}`, length 1, at the only constructor (`sdk/runtime.go:88-91`) |
| `ValidateChain` | `validate_chain.go:22` | No non-test caller in repo |
| `Locality`/`CostRank`/`QualityRank` semantics | `router.go:141-172` | `sdk/runtime.go:87` stamps `LocalityLocal`, leaves both ranks 0; `fast`/`quality` branches require `LocalityCloud` members and therefore always fall through to chain order |
| `cfg.ModelFast` | `anthropic.go:84,109-111` | Used only by `Classify` (`anthropic.go:181`), which `runner/intelligence/intelligence.go:90` refuses to dispatch |
| `parseRetryAfter` | `retry.go:42` | No non-test caller; `httpError` never receives `resp.Header` |
| `mask()` | `anthropic.go:406` | No non-test caller |
| `modelFast` / `modelQuality` consts | `anthropic.go:38-39` | Unused aliases of the defaults |
| `cfg["intelligence"]` | `cmd/awis/config.go:427,453` | Accepted by validator, advertised in scaffold, **never read** — `start.go` reads only `tick`, `namespace`, `anthropic_api_key` |
| `Embed` dispatch | `ports.go:48` | No `Dispatcher` method exists |
| `Classify` dispatch | `ports.go:55` | No `Dispatcher` method exists; runner rejects |

### 1.4 The composition-root truth — three roots, one wired

`sdk.Config.Intelligence` is assigned in **exactly one** production location.

| Composition root | Sets `Intelligence`? | Consequence |
|---|---|---|
| `cmd/awis/start.go:160` | ✅ yes | The only path that can reach a real provider |
| `cmd/awis-server/main.go:78-82` | ❌ **no** | **The GUI backend is permanently zero-AI.** Every intelligence step served by the GUI runtime routes to its fallback, always. |
| `apps/oip/cmd/oip/main.go:57-60` | ❌ **no** | **OIP is permanently zero-AI.** `capture-decision`'s `draft-entry` step always falls through to the `manual-entry` signal; `recall-decision`'s synthesis always returns "The record is silent." |
| `examples/hello_workflow/main.go:49` | ❌ no | Example only |
| `cmd/awis/{submit,signal,cancel}.go` | ❌ no | Control-plane only; never executes steps |
| `sdk/testing/harness.go:95` | ❌ no | Deterministic harness (correct — determinism requires it) |

This materially changes the shape of the problem. The prior review said "single-provider entry point". The truth is narrower and more useful: **there is one AI-capable entry point out of three, and the two flagship consumers — the GUI and OIP, the first AWIS application — have never executed a real intelligence call.**

Two consequences, both favourable:
- **GUI compatibility is a non-constraint.** You cannot regress behaviour that has never occurred.
- **OIP's fallback paths are the only intelligence behaviour ever exercised in an application**, which means the fallback machinery is the battle-tested part and the provider path is the unproven part — the inverse of the usual assumption.

---

## 2. Reality vs assumptions

Each row states a belief that is plausible from the documentation, then the code verdict.

### 2.1 "AWIS is Anthropic-bound." — **FALSE**

`internal/core` and `internal/engine` contain **zero** Anthropic identifiers. Workflow YAML declares `capability:` + `context_budget:` and never a provider or model (`apps/oip/workflows/*.yaml`, `examples/workflows/with-intelligence.yaml`). The EventLog stores `{adapter, model, tokens_used}` as opaque strings/ints (`engine/emit.go:43-51`). The GUI reads `capability` and `model_hint` only (`web/src/types.ts:31-34`) and never renders provider or usage data.

**The only production import of an Anthropic package outside its own directory is `cmd/awis/start.go:36`** — a composition root, which is architecturally the correct place for it.

Vendor coupling is confined to two places: the **configuration schema** (`anthropic_api_key` as a first-class recognised key, `config.go:423`) and the **composition root**.

### 2.2 "Multi-provider support is partially implemented." — **TRUE BUT MISLEADING**

Partially implemented implies a spectrum. The reality is a clean break: the *selection layer* is ~85% complete and 0% reachable; the *registration/configuration layer* is ~10% complete. It is not a half-built machine. It is a **finished engine with no fuel line**.

### 2.3 "The IntelligencePort is a reasonable multi-provider abstraction." — **FALSE**

Four capability-specific methods (`Draft`, `Embed`, `Synthesize`, `Classify`) mean any new capability is an interface change that breaks every adapter at once. `Capability` is `struct{ Name string }` (`ports.go:131`) — no context window, no max output, no modality, no pricing, no supported parameters. Capability *discovery* can answer "do you claim to draft?" and nothing else. Two of the four methods (`Embed`, `Classify`) already have no dispatch path, so every adapter implements dead weight today.

### 2.4 "`model_hint` selects a model." — **FALSE (documentation drift)**

`docs/PROVIDERS.md` publishes a `model_hint → model` mapping table. In code, `Draft` (`anthropic.go:145`) and `Synthesize` (`anthropic.go:167`) both pass `a.cfg.ModelQuality` unconditionally. `ModelHint` reaches `router.Eligible` for *ordering* and is never forwarded to the adapter — `DraftRequest` and `SynthesisRequest` have no field for it (`ports.go:66,78`). The documented mapping does not exist.

### 2.5 "Retry-After is honoured." — **FALSE (documentation drift)**

`doRequest` builds the error with `httpError(resp.StatusCode, raw)` (`anthropic.go:281`), which never sees `resp.Header` and constructs `&retryableHTTPError{statusCode, message}` with `retryAfter` unset (`anthropic.go:308-316`). The guard at `retry.go:132` is therefore always false; backoff is always exponential. `parseRetryAfter` (`retry.go:42`) has no non-test caller.

### 2.6 "Context budget protects the provider call." — **PARTIALLY TRUE, STRUCTURALLY MISPLACED**

Enforcement works and correctly rejects rather than truncates (`budget.go:31-40`, FR-IL-08). But `estimateTokens` is `(len(s)+3)/4` over **bytes** (`budget.go:23`), overestimating CJK/emoji content by roughly 3×; and enforcement runs **before routing** (`dispatcher.go:57`), so it cannot consult the selected model's real context window. The budget is a workflow author's guess about a model the workflow is forbidden to name.

### 2.7 "Usage is recorded, so cost tracking is a reporting exercise." — **FALSE**

`Usage.TokensUsed` is `InputTokens + OutputTokens` (`anthropic.go:300`). Input and output are priced differently by every provider, so cost is **unrecoverable**. Adversarial review sharpened this: `core.Usage` (`internal/core/ports.go:100-107`) has no `InputTokens`/`OutputTokens` fields at all, so the split is discarded **at the adapter boundary** — it never exists past `anthropic.go:300` and therefore never reaches the emitter (`engine/emitters.go:92`) to be dropped. The loss is by construction, not by an emitter omission, which means it cannot be recovered retrospectively from any persisted artefact. Also absent: cache-read/cache-write tokens, reasoning tokens, latency, provider request-id, and the provider *instance* identity. Cost tracking is not unimplemented; it is **foreclosed by the recorded schema**.

### 2.8 "Structured output conforms to the step's declared schema." — **FALSE**

The schema is appended to the prompt as prose (`anthropic.go:320-343`). `parseJSONOutput` strips code fences, attempts `json.Unmarshal`, and **on failure returns `{"text": <raw string>}`** (`anthropic.go:374-392`). A step declaring `outputs: {draft: string, confidence: number}` can silently receive `{text: "..."}`. Nothing validates the response against `step.Outputs`. The failure surfaces later, as an expression-evaluation error far from its cause.

### 2.9 "Provider failure is retried sensibly." — **PARTIALLY TRUE, WITH A COMPOUNDING BUG**

Two independent retry layers stack. The adapter retries up to 3 times internally (`retry.go:104`), and a provider failure surfaces as `StepError{intelligence_error}` which is then **also** eligible for the engine's generic step retry (`engine/failure.go:239` — only `capability_fallback` is excluded). A step declaring `retry: {attempts: 3}` therefore issues up to **9 provider calls**, each billable, with no shared budget and no circuit breaker. The `capability_fallback` exclusion is correct and verified; the compounding is not designed.

### 2.10 "`IsAvailable()` reports whether the provider is usable." — **FALSE**

`anthropic.go:124` returns `a.cfg.APIKey != ""` and explicitly performs no round-trip. Availability never reflects reachability: a provider that is down, rate-limited, or out of quota still reports available. Unavailability is discovered only by a failed call — which, given §1.3, has nowhere to fail over to.

The sharper consequence is forward-looking: **this predicate structurally excludes keyless local providers.** Ollama, LM Studio, and a default vLLM server have no API key. Any driver reusing this availability pattern reports permanently unavailable and is never routed to.

---

## 3. Documentation drift register

Claims in `docs/PROVIDERS.md` contradicted by code:

| # | Doc claim | Code truth |
|---|---|---|
| DD-1 | `model_hint` maps `fast`→haiku, `quality`→sonnet | Both Draft and Synthesize hardcode `ModelQuality` (§2.4) |
| DD-2 | "`Retry-After` … its value is honoured" | Never plumbed (§2.5) |
| DD-3 | "Requesting `embed` returns `ErrEmbedUnavailable`; **the router falls through** to the step fallback" | The runner rejects `embed` with `capability_unknown` **before any router involvement** (`runner/intelligence/intelligence.go:90-95`); `ErrEmbedUnavailable` is unreachable in production |
| DD-4 | "Adding a New Provider: … 4. Wire construction in `cmd/awis/start.go`" | Correct — and it documents the defect: **the extension model is "recompile the CLI"** |
| DD-5 | Config-file wiring "is M17" | `anthropic_api_key` config fallback exists (`start.go:143`), but the `intelligence:` key is inert (§1.3) |

---

## 4. Answer to Question 1 — how much already exists

Percentages are judgment-calibrated against the target architecture, expressed as two independent numbers: **Built** (exists and is tested) and **Reachable** (can execute in a shipped binary).

| Dimension | Built | Reachable | Evidence |
|---|---:|---:|---|
| Provider abstraction | 70% | 70% | Interface + 2 adapters + contract suite exist; shape closed (§2.3) |
| Routing | 85% | 5% | 11 tests specify correct semantics; executes over a 1-element set |
| Capability dispatch | 50% | 50% | 2 of 4 capabilities dispatchable |
| Failover | 80% | 0% | Loop written and tested; cannot iterate (§1.3) |
| Registration | 30% | 10% | Struct exists; no config path; name collisions silent |
| Configuration | 10% | 10% | Flat YAML, one hardcoded vendor key, `intelligence:` inert |
| Model selection | 5% | 0% | 3-value hint, never forwarded to the adapter (§2.4) |
| Prompt handling | 15% | 15% | Works for one provider; zero reuse, no record of what was sent |
| Structured outputs | 20% | 20% | Prompt-append + best-effort parse + silent degradation (§2.8) |
| Retries | 45% | 45% | 429/5xx correct; Retry-After dead; compounds with engine retry (§2.9) |
| Observability | 40% | 40% | Usage triple persisted; no cost, no instance, no in/out split |
| **Unweighted mean** | **41%** | **24%** | |

**Headline: roughly 40% of a true multi-provider architecture is built; roughly 25% of it can actually run.** The 16-point gap between those numbers is the activation deficit, and closing it is cheap — it is concentrated in about a dozen lines of composition at `sdk/runtime.go:82-93`.

---

## 5. What this audit changes about the prior conclusion

The earlier review (`00-ARCHITECTURE_REVIEW.md`) concluded "a multi-provider architecture with a single-provider entry point." That survives, with two corrections this audit adds:

1. **It is not one entry point — it is one of three, and the other two are the flagship consumers.** The GUI backend and OIP have never executed a real intelligence call (§1.4). This makes "preserve GUI behaviour" and "preserve existing workflows" far weaker constraints than assumed.
2. **The activation deficit is smaller than the design deficit.** Reachability (24%) is the cheap problem. The expensive problems are configuration (10%), model selection (0%), and structured outputs (20%) — and none of them are fixed by activation alone.
