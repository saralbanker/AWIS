# IMPLEMENTATION_PROGRAM

**Planning only. No code changes, no commits.**
Target architecture: `ARCHITECTURE_OPTIONS` recommendation — config-driven provider **registry** (Opt 2) + **driver/instance split** (Opt 3), with `openai-chat` adopted as a driver (Opt 4) and the policy engine's seams reserved but unbuilt (Opt 5).

Answers Question 3 (smallest path) and Question 5 (sequence).

---

## Answer to Question 3 — smallest implementation path

**(A) Enable the existing dormant architecture — then a bounded (B) refactor. Not (C).**

| Option | Verdict | Evidence |
|---|---|---|
| **A — Enable dormant architecture** | ✅ **Necessary and nearly sufficient for *operation*** | The selection layer is ~85% built and 0% reachable. The deficit is concentrated in ~11 lines at `sdk/runtime.go:86-89`. Five of eight blockers (B-1, B-2, B-3, B-4, B-8) are composition/config, not subsystem work; only B-5, B-6, B-7 touch `internal/intelligence` or `internal/core`. |
| **B — Bounded refactor** | ✅ **Required for the six named providers** | Activation alone cannot clear B-5 (model identity), B-6 (name collision), or B-7 (keyless availability). These three require touching `internal/core` and `internal/intelligence`. |
| **C — Replace subsystem** | ❌ **Rejected** | Would discard: 32 passing tests including 11 that specify exactly the target routing semantics; a provider-neutral DSL and EventLog; a clean dependency graph where `core`/`engine` contain zero vendor identifiers; and a correctly factored adapter-agnostic contract suite. Nothing in the audit identifies a *wrong* abstraction in the router — only wrong inputs. |

**The smallest path is A→B, sequenced so that A ships standalone value and B is entered only with A's tests green.**

One correction to naive-A: activation alone would produce a multi-provider system whose routing hints **silently do nothing** (B-8 — `Locality` hardcoded, ranks zero). B-8 must ship *inside* Phase 0, not after it.

---

## Program constraints (binding on every phase)

| Constraint | How it is enforced |
|---|---|
| Preserve existing workflows | No workflow YAML changes before Phase 3; Phase 3 additions are optional fields only |
| Preserve the DSL | `internal/dsl` untouched through Phase 2 |
| Preserve engine behaviour | `internal/engine` untouched except the single additive `Usage` payload amendment in Phase 2 |
| Preserve GUI behaviour | `web/` untouched in all phases (zero coupling proven — `GAP_ANALYSIS` §5) |
| Preserve current tests | All 32 seam tests must pass unmodified at every phase gate. **A test requiring modification is a design defect, not a test defect** — escalate rather than edit |
| No commits | Planning artefact only |

---

## Phase 0 — Activation

**Objective:** make the built-and-tested routing layer reachable, with no new provider, no new interface, and no config schema change.

### Deliverables

| # | Deliverable | Locus | Type |
|---|---|---|---|
| 0.1 | `sdk.Config.Providers []intel.Registration` + `sdk.Config.FallbackChain []string` | `sdk/runtime.go:26-43` | Additive |
| 0.2 | `Config.Intelligence` retained as a deprecated shim desugaring to a one-element registration | `sdk/runtime.go:82-93` | Compat |
| 0.3 | `Locality`/`CostRank`/`QualityRank` taken from the caller instead of hardcoded (**closes B-8**) | `sdk/runtime.go:87` | Fix |
| 0.4 | `NewRouter` returns an error on duplicate `ProviderName()` instead of silently overwriting (**partial B-6**) | `router.go:85-87` | Fix |
| 0.5 | `ValidateChain` called from `NewRuntime` (**makes D-13 live**) | `sdk/runtime.go` | Fix |
| 0.6 | Plumb `resp.Header` into `httpError` so `parseRetryAfter` is reached (**closes G-2's live defect**) | `anthropic.go:281,308` | Bug fix |
| 0.7 | Make `intelligence:` config key functional **or** remove it — no third option | `cmd/awis/config.go:427,453` | Bug fix |
| 0.8 | Decide and document zero-AI status of `awis-server` and `oip` | `cmd/awis-server/main.go:78`, `apps/oip/cmd/oip/main.go:57` | Decision |
| 0.9 | Correct `docs/PROVIDERS.md` drift DD-1…DD-5 | `docs/PROVIDERS.md` | Docs |

### Dependencies
None. Phase 0 is entered from current `main`.

### Exit criteria
- All 32 existing seam tests pass **unmodified**.
- New test: a 2-provider chain where provider 1 fails and provider 2 serves — proving `dispatcher.go:65-72` iterates for the first time in the project's history.
- New test: `fast`/`quality` hints produce cost/quality ordering with a real cloud+local mix (proves 0.3).
- New test: duplicate `ProviderName()` registration returns an error (proves 0.4).
- `awis start` behaviour byte-identical for a single-Anthropic configuration.

### Risk
**Low.** Entirely additive to `sdk.Config`. The one governance item: `sdk` surface is frozen post-M08, so 0.1 requires a documented amendment following the ADJ-7/ADJ-8 precedent.

### Why this phase is sequenced first
It converts 85%-built/0%-reachable routing into 85%/85%, closes two live bugs (0.6, 0.7) and one latent one (B-8), and creates the first end-to-end failover test in the project's history. It adds **no** provider and therefore carries no wire-protocol risk.

**Do not oversell it.** Adversarial review partially rejected the claim that Phase 0 "ships standalone value" (`ADVERSARIAL_REVIEW` C-2): only two of its fixes are user-visible. Phase 0's real value is **optionality and de-risking** — it is where B-8 is closed *before* it can silently corrupt routing decisions. Justify it on those terms.

---

## Phase 1 — Configuration & instance identity

**Objective:** providers become configuration rather than code. Instance identity separates from provider type.

### Deliverables

| # | Deliverable | Locus |
|---|---|---|
| 1.1 | Structured `providers:` config block: per-instance `id`, `driver`, `base_url`, `credential_ref`, `headers`, `models`, `locality`, `cost_rank`, `quality_rank`, `limits` (**closes B-3, B-4**) | `cmd/awis/config.go` |
| 1.2 | Credential **references** (`env:NAME`, `file:PATH`) — never literals in config | new config layer |
| 1.3 | Instance ID as the registry key, distinct from `ProviderName()` (**fully closes B-6**) | `router.go`, `Registration` |
| 1.4 | `start.go` builds registrations by **iterating configured instances** (**closes B-2**) | `cmd/awis/start.go:133-152` |
| 1.5 | `anthropic_api_key` retained via a compatibility mapping to a synthesised default instance | `cmd/awis/config.go` |
| 1.6 | Extend the allowlist-of-the-harmless masking model to per-instance keys; `base_url` visible, credentials masked | `cmd/awis/config.go:47-56` |
| 1.7 | `awis config validate` accepts the new block and still rejects unknown keys | `cmd/awis/config.go:420-431` |

### Dependencies
Phase 0 complete (needs 0.1 to have somewhere to register N providers, 0.4 for collision safety).

### Exit criteria
- Two Anthropic instances with different credentials register and route independently — the direct proof that B-6 is closed.
- `awis config show` masks every credential-shaped per-instance key; audit rows contain no secret. **Regression test required**: the existing masking design exists because a denylist once printed an `authorization` value into an immutable audit row.
- A pre-existing `anthropic_api_key`-only config file starts unchanged.

### Risk
**Medium.** Touches the config schema, `config show` masking, and the audit path. The masking inversion at `config.go:47-56` is a security control — it must be extended, never relaxed.

---

## Phase 2 — Invoker extraction + `openai-chat` driver

**Objective:** move everything provider-independent above the driver, then add the driver that serves five of six targets.

### Deliverables

| # | Deliverable | Locus |
|---|---|---|
| 2.1 | Extract retry/backoff/Retry-After into a shared invoker (**closes G-2**) | new; from `adapters/anthropic/retry.go` |
| 2.2 | Extract prompt rendering into a shared renderer (**closes G-3**) | new; from `anthropic.go:320-372` |
| 2.3 | Extract credential redaction to the seam; retire the `sk-ant-…` masker (**closes D-16**) | new; from `anthropic.go:406` |
| 2.4 | Centralise usage normalisation; **extend `core.Usage` once, comprehensively** (**closes G-1**) | `internal/core/ports.go:100-107` |
| 2.5 | Additive `StepCompleted` payload amendment for the extended usage | `internal/engine/emit.go:43-51` |
| 2.6 | Resolve nested-retry compounding: single shared attempt budget (**closes G-5**) | `engine/failure.go:239` + invoker |
| 2.7 | Extend `porttest` into a **driver conformance suite** | `internal/intelligence/porttest/` |
| 2.8 | `openai-chat` driver — Bearer auth, configurable base_url, native `response_format` | new `adapters/openaichat/` |
| 2.9 | Structured-output negotiation: native schema mode where supported, prompt-append fallback (**closes G-4**) | invoker + drivers |

### Dependencies
Phase 1 complete (2.8 is unusable without configurable `base_url` and credentials).

### Exit criteria
- **Prompt byte-equality gate (2.2):** rendered Anthropic prompts are byte-identical to current output before and after extraction. This makes the riskiest deliverable provably behaviour-preserving.
- Both drivers pass the conformance suite against recorded fixtures. **No network in CI** — the existing IMP P7 rule holds.
- One gated live smoke per driver, following the `ANTHROPIC_LIVE` pattern — and **actually executed at least once**. `VERIFIED_DEFECT_REGISTER` D-05 records a shipped HTTP 404 that a single live run would have caught.
- `Usage` extension is additive; EventLog replay over pre-amendment history is unchanged (EDR-007 projection reads only `outputs`).

### Risk
**Medium-high.** This is where existing Anthropic behaviour changes shape. Mitigations: conformance suite lands *before* the second driver (2.7 before 2.8); prompt byte-equality gate on 2.2; `Usage` extended once with over-provisioned fields, because the log is append-only.

### Unlocked on completion
OpenAI, OpenRouter, vLLM, LM Studio, and Ollama-via-`/v1` become **configuration entries**, not code. Five of six targets, one deliverable.

---

## Phase 3 — Model identity, health, and Gemini

**Objective:** close the remaining blockers and the last target.

### Deliverables

| # | Deliverable | Locus |
|---|---|---|
| 3.1 | Model identity in the request path (**closes B-5**) | `core/step.go:87`, `ports.go:66,78` |
| 3.2 | `requires: {context_tokens, max_output_tokens, locality}` in `IntelReq`; `model_hint` becomes a deprecated alias mapping to a built-in policy | `core/step.go`, `internal/dsl/dsl.go:88-90`, `internal/validate/validate.go:476` |
| 3.3 | Model descriptors: context window, max output, modalities, pricing — config-authoritative, optional `/models` enrichment | new |
| 3.4 | Health probe + circuit breaker; `IsAvailable()` reads a cached probe (**closes B-7, G-6**) | `ports.go:57`, new |
| 3.5 | Budget enforcement moves post-routing, against the selected model's real context window (**closes D-15**) | `dispatcher.go:57` |
| 3.6 | `google-genai` driver (**closes the sixth target**) | new `adapters/googlegenai/` |
| 3.7 | Cost computation from descriptors, recorded alongside usage | invoker |

### Dependencies
Phase 2 complete (3.7 requires 2.4's usage shape; 3.6 requires the driver split).

### Exit criteria
- Ollama and vLLM route successfully with **no API key** — the direct proof B-7 is closed.
- An intelligence step declaring `requires: {context_tokens: 150000}` selects only models whose descriptor can hold it.
- Existing workflows using bare `capability:` + `context_budget:` run unchanged. **`model_hint` continues to validate and behave.**
- Cost appears in `awis trace` for a real call.

### Risk
**Medium.** First phase touching the DSL. Strictly additive fields; `model_hint` retained as an alias, never removed in this phase.

---

## Explicitly out of scope

Named to prevent scope drift, per the brief's prohibition on speculative redesign.

| Not building | Why |
|---|---|
| **Policy engine** (named routing policies, cost-aware selection) | Correct end state, wrong purchase order. No operator runs two providers today. Its seams are reserved in Phase 2/3; its machinery waits for demand. |
| **Universal OpenAI-compatible translation layer** | Rejected in `ARCHITECTURE_OPTIONS` Opt 4: lossy for Gemini, demotes working Anthropic code, forecloses native structured output. |
| **Model IDs in workflow YAML** | Puts a volatile vendor fact in a durable versioned artefact. This repo already shipped an HTTP 404 from hard-coded model IDs (`VERIFIED_DEFECT_REGISTER` D-05). |
| **`Embed`/`Classify` dispatch** | CONTRA-3 / FR-IL-10 deliberately defer these. Implementing them to fill interface holes is unrequested scope. |
| **`IntelligencePort` → `Invoke` replacement** | The one genuinely breaking change. Deferred past Phase 3 and bundled with a documented version boundary. Phases 0–3 are additive precisely so this stays optional. |
| **Any GUI work** | Zero coupling proven. `web/` is untouched in all four phases. |
| **Streaming, tool-use, multi-turn** | Real future needs. `CapabilityRequest` should be *shaped* to admit them; none are implemented. |

---

## Dependency graph

```
Phase 0  Activation ─────────────► standalone value, ships alone
   │       closes B-1, B-8, partial B-6; fixes Retry-After, inert config key
   ▼
Phase 1  Config & identity ──────► providers become configuration
   │       closes B-2, B-3, B-4, B-6 (full)
   ▼
Phase 2  Invoker + openai-chat ──► 5 of 6 targets unlocked
   │       closes G-1..G-5
   ▼
Phase 3  Model identity + health ► 6 of 6 targets; DSL extended additively
           closes B-5, B-7, G-6, D-15
                                     ╷
                                     └─► [deferred] Policy engine · Invoke replacement
```

Each phase is independently shippable and independently revertible. No phase requires the next to be correct.

---

## Blocker coverage

| Blocker | Phase 0 | Phase 1 | Phase 2 | Phase 3 |
|---|:--:|:--:|:--:|:--:|
| B-1 SDK arity | ✅ | | | |
| B-2 Hardcoded root | | ✅ | | |
| B-3 Config allowlist | | ✅ | | |
| B-4 No base_url | | ✅ | | |
| B-5 Model identity | | | | ✅ |
| B-6 Name collision | ⚠️ partial | ✅ | | |
| B-7 Keyless availability | | | | ✅ |
| B-8 Routing metadata | ✅ | | | |
| G-1 Usage/cost | | | ✅ | |
| G-2 Retry location | ⚠️ bug fixed | | ✅ | |
| G-3 Prompt location | | | ✅ | |
| G-4 Structured output | | | ✅ | |
| G-5 Retry compounding | | | ✅ | |
| G-6 Health/breaker | | | | ✅ |

All eight blockers and all six debt items are covered. **Nothing is deferred into an unplanned future** except the two items explicitly named out of scope.
