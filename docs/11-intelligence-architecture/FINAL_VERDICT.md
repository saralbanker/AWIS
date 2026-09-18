# FINAL_VERDICT — Intelligence Architecture Decision Package

**Founder decision package.** Scope: converting the AWIS intelligence layer from single-provider operation to a provider-agnostic platform.
**Evidence base:** production code at `8a87f70`, independently adversarially verified. Ten conclusions challenged; three did not survive in their original form and are stated here in corrected form.

---

## 1. What exists today

A **competently designed, well-tested provider selection layer that has never executed.**

- `CapabilityRouter` — eligibility filtering, chain-as-enabled-set, hint ordering, stable tie-breaks, typed errors. **11 tests specify exactly the multi-provider semantics you want.**
- `Dispatcher` — budget enforcement, routing, try-next-on-failure.
- Two adapters (Anthropic, Null), an adapter-agnostic contract suite, and 32 production-behaviour tests over the seam.
- Provider-neutral durable artefacts: workflow YAML names **capabilities only**; the EventLog stores opaque `{adapter, model, tokens_used}`; `internal/core` and `internal/engine` contain **zero** vendor identifiers.

And the fact that reframes everything:

> **`sdk.NewRuntime` accepts exactly one provider and builds a one-element fallback chain (`sdk/runtime.go:86-89`). Of three production composition roots, only `cmd/awis start` configures a provider at all — the GUI backend and OIP have never executed a real intelligence call.**

Approximately 40% of a multi-provider architecture is built; roughly 25% of it can run. *(Read as a ranking, not a measurement — see `ADVERSARIAL_REVIEW` C-3.)*

---

## 2. What is missing

**Eight blockers. Five of them (B-1, B-2, B-3, B-4, B-8) live in configuration and composition — not in the intelligence subsystem at all.** Only B-5, B-6, and B-7 require touching `internal/intelligence` or `internal/core`.

| | Blocker | Blocks |
|---|---|---|
| B-1 | `sdk.Config` admits one provider | All six targets |
| B-2 | Composition root hardcodes Anthropic | All six |
| B-3 | `awis config validate` **errors** on any new provider key | All six |
| B-4 | No `base_url` in configuration | Ollama, LM Studio, vLLM, OpenRouter |
| B-5 | No model identity in the request path | Ollama, vLLM, OpenRouter |
| B-6 | `ProviderName()` collides silently on duplicate registration | Any two OpenAI-compatible endpoints |
| B-7 | `IsAvailable()` is a credential check → excludes keyless providers | Ollama, LM Studio, vLLM |
| B-8 | Routing metadata hardcoded → **hints silently do nothing** | All six (latent) |

Plus six rewrite-debt items, of which one is urgent: **`core.Usage` discards the input/output token split at the adapter boundary**, so cost is unrecoverable from the append-only log — permanently, for every call made before it is fixed.

---

## 3. What should be built

**A config-driven provider *registry* + a *driver/instance* split.**

The load-bearing fact: **five of the six targets share one wire protocol.** OpenAI, OpenRouter, vLLM, LM Studio, and Ollama (via its `/v1` surface) are all `openai-chat` with a different `base_url`. Only Gemini needs bespoke work.

This inverts the naive plan. "Add OpenAI, then Ollama, then vLLM" is three milestones today and **one deliverable plus three config entries** under an instance registry.

Separate what is *compiled* (a driver = one wire protocol) from what is *configured* (an instance = id, driver, endpoint, credential, models, traits). The seventh provider then becomes a config edit rather than a release — which is the literal objective.

---

## 4. What should NOT be built

| Not building | Why |
|---|---|
| **Policy engine** (cost-aware routing, named policies) | Correct end state, wrong purchase order. No operator runs two providers today. Reserve its seams; defer its machinery. |
| **Universal OpenAI-compatible layer** | Lossy for Gemini, demotes working Anthropic code, forecloses the native structured-output modes that fix G-4. Adopt OpenAI-compat as **one driver**, never as the seam. |
| **Model IDs in workflow YAML** | Puts a volatile vendor fact in a durable versioned artefact. This repo already shipped an HTTP 404 from hard-coded model IDs (`VERIFIED_DEFECT_REGISTER` D-05). |
| **Generalizing the Anthropic adapter** | Backwards. **Narrow** it — lift retry, prompts, redaction, and usage-summing upward, leaving ~150 lines of wire translation. |
| **`Embed`/`Classify` dispatch** | Deliberately deferred by CONTRA-3 / FR-IL-10. Filling interface holes is unrequested scope. |
| **Any GUI work** | Zero coupling proven across all of `web/src`. Untouched in all four phases. |
| **`IntelligencePort` → `Invoke` replacement** | The one genuinely breaking change. Phases 0–3 are additive precisely so this stays optional and can be bundled with a deliberate version boundary. |

---

## 5–7. Cheapest / safest / fastest

| | Path | Assessment |
|---|---|---|
| **Cheapest** | Phase 0 only — activation | Real but limited. Closes B-1 and B-8, fixes two live bugs, creates the first failover test in the project's history. **Adds no provider.** Its value is optionality, not function. |
| **Safest** | Phase 0 → 1 → 2 → 3, gated | Entirely additive to `sdk.Config`; every phase independently shippable and revertible; all 32 existing tests must pass **unmodified** at every gate. The riskiest step (prompt extraction) is protected by a byte-equality gate. |
| **Fastest to a second provider** | Phase 0 → 1 → 2.8 (`openai-chat` driver), deferring 2.1–2.7 | Reaches five of six targets soonest — and **incurs the debt this program exists to avoid**: retry, prompts, and redaction get duplicated into the second driver, and `Usage` calls made before the fix are permanently uncosted. |

The fastest and safest paths differ by exactly one decision: whether the invoker extraction (2.1–2.7) precedes or follows the second driver. **It must precede it.** The `Retry-After` defect already demonstrates the failure mode at N=1 — the retry policy inside the adapter was documented as working and was not. Duplicating that pattern before centralising it is how one silent defect becomes three.

---

## 8. Recommended path

**Phase 0 → Phase 1 → Phase 2 → Phase 3**, per `IMPLEMENTATION_PROGRAM.md`.

| Phase | Outcome | Risk |
|---|---|---|
| **0 · Activation** | Routing becomes reachable. Closes B-1, B-8, partial B-6. Fixes Retry-After and the inert `intelligence:` key. No new provider. | Low |
| **1 · Config & identity** | Providers become configuration. Closes B-2, B-3, B-4, and B-6 fully. | Medium |
| **2 · Invoker + `openai-chat`** | **Five of six targets unlocked.** Closes G-1…G-5. | Medium-high |
| **3 · Model identity + health + Gemini** | Sixth target. Closes B-5, B-7, G-6. DSL extended additively. | Medium |

**Three sequencing decisions that are not negotiable:**

1. **B-8 ships inside Phase 0, not after it.** Independent verification confirmed that fixing arity alone yields a multi-provider system whose `fast`/`quality` hints silently fall through — every adapter is stamped `LocalityLocal` and both rank fields are zero.
2. **B-6 is fixed before any registry is populated.** Instance identity propagates into `Usage` and the append-only EventLog. Retrofitting it once history exists is not possible.
3. **`core.Usage` is extended once, comprehensively, in Phase 2** — including fields not yet computed. The log is append-only; a second amendment costs far more than unused columns.

---

## Verdict

# **PROCEED WITH CHANGES**

**Proceed**, because the evidence is unusually favourable. The hard architectural work — a provider-neutral DSL, a clean dependency graph, a tested router with correct semantics, a neutral EventLog — is done and correct. Four of the five stated compatibility constraints (workflows, DSL, engine, GUI) cost **nothing**, and the fifth (tests) is an asset rather than a tax: the 11 router tests already specify the target behaviour. The subsystem does not need replacing; it needs a fuel line.

**With changes**, because three conclusions did not survive adversarial review and the program is built on their corrected forms:

1. **"Multi-provider already exists" is REJECTED.** A selection layer exists. Registration and configuration do not. Planning that treats this as a switch-flip will ship a system whose routing hints silently do nothing.
2. **"Driver architecture is needed" is PARTIALLY REJECTED.** It is economically justified, not technically required — and that justification is **contingent on the six-provider target list**. If the list narrows to two providers with distinct protocols, revisit and choose the simpler registry-only option.
3. **"The Anthropic adapter should be generalized" is REJECTED.** Narrow it instead. Generalizing it is the OpenAI-compatible-layer trap, which breaks Gemini and demotes working code.

Also: **the quantification is a ranking, not a measurement.** Do not carry "40%/25%" into commitments.

**One item warrants attention independent of this program.** A step declaring `retry: {attempts: 3}` currently issues up to **nine billable provider calls** — three engine retries times three adapter retries, with no shared budget and no circuit breaker (`engine/failure.go:239` + `retry.go:104`). This is a live cost defect today, at one provider. It is scheduled in Phase 2 (G-5), but it is worth knowing about now.

**Recommended immediate action:** authorise Phase 0. It is low-risk, entirely additive, closes two live bugs and one latent one, and is the prerequisite for every subsequent option. Authorise Phases 1–3 as a program contingent on Phase 0's exit criteria — chiefly the first end-to-end failover test this project has ever been able to write.

The one governance item requiring a founder decision before Phase 0 can start: **`sdk.Config` is frozen post-M08.** Deliverable 0.1 is additive but requires a documented freeze amendment, following the ADJ-7/ADJ-8 precedent. There is no workaround — B-1 cannot be fixed without touching that surface.
