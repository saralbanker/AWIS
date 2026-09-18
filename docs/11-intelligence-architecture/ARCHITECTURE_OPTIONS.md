# ARCHITECTURE_OPTIONS

**Question answered:** what architecture should AWIS ultimately target, and what is the lowest-cost path that does not create future rewrite debt?

Five options, evaluated against repository truth. Cost estimates are relative effort, not calendar commitments.

---

## Evaluation criteria

| Criterion | Why it matters here |
|---|---|
| **Unblocks the six targets** | The stated objective. Weighted highest. |
| **Rewrite debt** | Work that must be *undone* later. The brief explicitly forbids creating it. |
| **Preserves the 32 tests** | The router's semantics are the asset (`INTELLIGENCE_TRUTH_AUDIT` §1.2). |
| **Backward compatibility** | Workflows, DSL, engine, GUI — four of which are free (`GAP_ANALYSIS` §5). |
| **Blast radius** | How much of the frozen post-M08 `sdk` surface must change. |
| **Cost** | Relative implementation effort. |

---

## Option 1 — Anthropic-only (status quo)

Keep one provider. Delete or ignore the dormant routing layer.

| | |
|---|---|
| **Unblocks targets** | ❌ None |
| **Rewrite debt** | Low now, unbounded later |
| **Cost** | Zero |

**Pros.** No work. No risk. The current system is verified and shipping.

**Cons.** Fails the objective outright. Also worth stating plainly: this option is *more expensive than it looks*, because the dormant machinery is not free — 32 tests, four files, and a documented architecture are being maintained for behaviour that never executes. Choosing Option 1 honestly means **deleting** `CapabilityRouter`, `Dispatcher`'s chain walk, `Registration`, and `ValidateChain`, not merely leaving them.

**Verdict: REJECT.** Fails the objective. Its only honest form (deletion) destroys the most valuable asset in the subsystem.

---

## Option 2 — Provider Registry (config-driven instances, one adapter per provider)

Add a `providers:` config block. Each entry names an adapter type and its settings. `sdk.Config` takes N registrations. Keep the existing "one Go package per provider" adapter model.

| | |
|---|---|
| **Unblocks targets** | ✅ All six, but at N adapters for N providers |
| **Rewrite debt** | **Medium** |
| **Preserves tests** | ✅ Fully — this is the router's intended input |
| **Blast radius** | `sdk.Config` (additive), config schema, `start.go` |
| **Cost** | Low-medium |

**Pros.** Directly clears B-1, B-2, B-3, B-4, B-8 — five of eight blockers — and *avoids* B-6 for distinct-provider topologies (though not for two instances of the same provider, e.g. local vLLM plus remote vLLM, which still collide). Requires no interface change. The router's 11 tests become live specifications rather than dormant ones. Fully additive to `sdk.Config`.

**Cons.** Leaves the duplication problem untouched: OpenAI, OpenRouter, vLLM, LM Studio, and Ollama-via-`/v1` are **the same wire protocol** (`GAP_ANALYSIS` §3.1). Building five near-identical adapters, each with its own retry (repeating the `retry.go:42` defect five times) and its own prompt builders, is the rewrite debt. You would consolidate them within two providers of shipping.

**Verdict: NECESSARY BUT INSUFFICIENT.** Every component of Option 2 is required. It is a *layer* of the answer, not the answer.

---

## Option 3 — Driver Architecture (driver = wire protocol; instance = configured endpoint)

Separate what is *compiled* (a driver implementing one wire protocol) from what is *configured* (a provider instance: id, driver, base_url, credential, models, traits).

| | |
|---|---|
| **Unblocks targets** | ✅ All six, with **two** drivers |
| **Rewrite debt** | **Low** |
| **Preserves tests** | ✅ Fully |
| **Blast radius** | Adds a layer; `IntelligencePort` can survive Phase 2 unchanged |
| **Cost** | Medium |

**Pros.** This is the option the repository's own evidence points at. Five of six targets collapse onto one `openai-chat` driver; only Gemini needs bespoke work. It is the *only* option that makes B-6 (name collision) solvable in principle, because instance identity becomes a first-class configured value rather than a compile-time constant. Adding a seventh provider that speaks OpenAI-compatible becomes a **config edit, not a release** — which is the literal stated goal ("without requiring engine changes").

**Cons.** Introduces two concepts where there was one. Requires instance IDs to be threaded into `Usage` and the EventLog. Extraction of retry/prompt/redaction out of the Anthropic adapter is behaviour-preserving work that must be *proven* behaviour-preserving, not assumed.

**Verdict: THE CORE OF THE ANSWER.** Combined with Option 2's registry, it clears all eight blockers and four of six rewrite-debt items.

---

## Option 4 — OpenAI-Compatible Layer (one adapter; everything speaks OpenAI)

Normalise on the OpenAI chat-completions shape. Providers that differ get a translation shim.

| | |
|---|---|
| **Unblocks targets** | ⚠️ Five cleanly, one badly |
| **Rewrite debt** | **High** |
| **Cost** | Low initially, high on the second exception |

**Pros.** Genuinely cheapest for five of six targets. Tempting, and it is the pattern most of the ecosystem converged on.

**Cons — and these are decisive.**
1. **Gemini is not OpenAI-compatible.** Model in the URL path, `contents`/`parts` body, different auth. A shim is a lossy hop, not a config entry.
2. **Anthropic is not OpenAI-compatible either.** The existing, working, tested adapter would be demoted to a translation target — actively destroying shipped value.
3. **Lossy on exactly the features that matter.** Native structured output (`response_format` vs `responseSchema` vs tool-use), cache-control, reasoning tokens, and tool-calling do not survive a lowest-common-denominator hop. Since G-4 (unenforced structured output) is already rewrite debt, standardising on a shape that *cannot* express native schema modes locks the defect in.
4. **OpenRouter's compatibility is itself imperfect** and varies by upstream model, so even the "compatible" set needs per-instance escapes.

The brief's forbidden-assumptions list names this explicitly: *"Do not assume OpenAI-compatible APIs are the best abstraction."* The evidence agrees. OpenAI-compatibility is the right shape for **one driver**, not for **the seam**.

**Verdict: REJECT as an architecture. ADOPT as a driver.** This is the single most important distinction in this document: `openai-chat` is enormously valuable as one driver among several (it serves five targets), and actively harmful as the universal interface.

---

## Option 5 — Policy Engine + Driver Layer (full target)

Option 3 plus a model catalog with descriptors, named routing policies as configuration, constraint-based selection (`requires: {context_tokens, locality, …}`), failover policy, cost-aware routing, and health/circuit breaking.

| | |
|---|---|
| **Unblocks targets** | ✅ All six, plus future ones |
| **Rewrite debt** | **None** |
| **Blast radius** | Large: `IntelligencePort` replacement, `IntelReq` extension, DSL change, EventLog amendment |
| **Cost** | High |

**Pros.** The correct end state. Answers cost tracking, capability discovery, and local-model health properly. Removes `model_hint` — an abstraction the evidence shows is already inert and already wrong (`TRUTH_AUDIT` §2.4).

**Cons.** Buying all of it now is speculative against real demand. There is no operator today asking to route by cost — there is no operator today running *two* providers. Building a policy engine before a second provider exists means designing against imagined requirements, which the brief forbids ("no future-state fantasies").

**Verdict: CORRECT TARGET, WRONG PURCHASE ORDER.** Adopt its *interface shape* now; defer its *machinery*.

---

## Comparison

| | Opt 1 Anthropic-only | Opt 2 Registry | Opt 3 Driver | Opt 4 OpenAI-compat | Opt 5 Policy+Driver |
|---|---|---|---|---|---|
| Unblocks six targets | ❌ | ✅ (6 adapters) | ✅ (2 drivers) | ⚠️ (5 well, 1 badly) | ✅ |
| Rewrite debt created | Unbounded | Medium | **Low** | **High** | None |
| Preserves 32 tests | ❌ (deletes) | ✅ | ✅ | ⚠️ | ✅ |
| Workflows/DSL/GUI compat | ✅ | ✅ | ✅ | ✅ | ⚠️ (DSL change) |
| Clears B-6 (collision) | n/a | ⚠️ partially | ✅ | ✅ | ✅ |
| Clears B-7 (keyless local) | n/a | ❌ | ⚠️ enables | ⚠️ enables | ✅ |
| Solves cost tracking | ❌ | ❌ | ⚠️ enables | ❌ | ✅ |
| Config-only new provider | ❌ | ❌ | ✅ | ✅ | ✅ |
| Cost | 0 | Low-med | **Medium** | Low→High | High |

---

## Recommendation

### **Option 2 + Option 3, built in that order, shaped by Option 5, with Option 4 adopted strictly as a driver.**

Concretely:

1. **Adopt Option 2's registry** — config-driven provider instances, N registrations, instance identity separate from provider type. Clears five of eight blockers directly, and closes B-6 fully once instance identity replaces `ProviderName()` as the key.
2. **Adopt Option 3's driver split** — driver (compiled wire protocol) vs instance (configured endpoint). Makes the seventh provider a config edit.
3. **Implement Option 4 as the `openai-chat` driver** — one deliverable that serves five of six targets. Never as the seam.
4. **Reserve Option 5's seams without building them** — this is the anti-rewrite-debt clause, and it costs little:
   - Shape `Usage` **once**, comprehensively, including fields not yet computed (instance id, input/output split, cache tokens, latency). The EventLog is append-only; a second amendment is far more expensive than unused fields.
   - Model selection enters as a **requirement** (`requires: {context_tokens, locality}`), never as a model ID in workflow YAML — so the policy engine, when built, has somewhere to attach.
   - Keep `IntelligencePort` in Phase 2, but do not add capability-specific methods to it. Any new capability arrives through a dispatch-shaped call, so the eventual `Invoke` replacement is a narrowing, not a rewrite.

### Why this is the lowest-cost option that avoids rewrite debt

Option 2 alone incurs debt (five duplicate adapters). Option 4 alone incurs worse debt (a lossy universal shape that breaks Gemini and demotes working Anthropic code). Option 5 alone is correct but pays now for demand that does not exist. **Option 2+3 is the only combination where every unit of work built in the first phase survives into the end state**, because a config-driven instance registry and a driver split are prerequisites of Option 5 rather than alternatives to it.

### The one non-negotiable

**B-6 (`ProviderName()` collision) must be fixed before any registry is populated, not after.** Instance identity propagates into `Usage`, into the EventLog, and into every routing decision. Retrofitting it once history exists means rewriting append-only rows, which is not possible. This is the single decision that, if deferred, converts a low-debt path into a high-debt one.
