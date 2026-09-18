# ADVERSARIAL_REVIEW

**Method.** Every load-bearing conclusion in this package was subjected to a disproof attempt. Two independent passes:

1. **Independent code verification** — a separate verifier agent, given no access to this analysis's reasoning, was asked to *disprove* nine factual claims against production code and to report unprompted findings. Its remit was explicitly adversarial: a claim survived only if disproof was attempted and failed.
2. **Self-challenge** — the five conclusions the brief nominated, plus four of this package's own conclusions, challenged on the assumption they are wrong.

A conclusion is marked **VERIFIED** only if a disproof was genuinely attempted and failed. Conclusions that did not survive are marked **REJECTED** and the corrected claim is stated.

---

## Part A — Independent factual verification

Nine claims, all against production code (`_test.go` and `./archive` excluded). **All nine VERIFIED.** One returned a correction that *strengthened* the finding.

| # | Claim | Verdict | Disproof attempted |
|---|---|:--:|---|
| 1 | `sdk.NewRuntime` makes multi-provider impossible via the public SDK | **VERIFIED** | Searched every `sdk.Config{}` site and the full `*Runtime` method set for post-construction provider injection. None exists. |
| 2 | Dispatcher's failover loop can never iterate >1 in a shipped binary | **VERIFIED** | Checked whether `Eligible()` could expand beyond the registered set. It cannot — it returns only chain ∩ regs, and chain is length 1. |
| 3 | Only `cmd/awis/start.go` sets `Intelligence`; awis-server and OIP are permanently zero-AI | **VERIFIED** | Enumerated all 8 non-test `sdk.Config{}` sites. Only `start.go:160` sets the field. |
| 4 | `NewRouter` silently overwrites on duplicate `ProviderName()` | **VERIFIED** | Looked for duplicate detection. `router.go:85-87` is plain map assignment, last-write-wins. |
| 5 | `ModelHint` never reaches the adapter; `ModelFast` unreachable in production | **VERIFIED** | Traced `ModelHint` end-to-end. `IntelReq` is consumed only for *selection*; `DraftRequest`/`SynthesisRequest` have no model field. `Classify` (sole `ModelFast` user) has zero non-test callers. |
| 6 | `Retry-After` never honoured; `parseRetryAfter` has no non-test caller | **VERIFIED** | Searched for any caller and any path where `resp.Header` reaches `httpError`. Zero call sites; `httpError(statusCode int, body []byte)` cannot see headers. |
| 7 | Config key `intelligence:` is validated but never read | **VERIFIED** | Grepped every read of the key. It appears only in validation/known-key maps. |
| 8 | `Usage.TokensUsed` is a sum; per-direction cost unrecoverable | **VERIFIED (+ correction)** | Checked for any preserved split in emit/storage. **Correction: the claim understated it.** `core.Usage` has no input/output fields at all, so the split is discarded at `anthropic.go:300` — it never exists past the adapter boundary. Loss is by construction, not by emitter omission. |
| 9 | GUI has zero coupling to provider or usage data | **VERIFIED** | Searched all of `web/src` for capability/model/adapter/tokens/usage/provider. Only `capability` and `model_hint` are read, both from workflow definitions. |

### Unprompted findings from independent verification

Both were already identified in this analysis; independent confirmation raises confidence and one adds a sharper consequence.

- **`sdk/runtime.go:87` hardcodes `Locality: LocalityLocal`** for whatever adapter is passed — including the Anthropic *cloud* adapter. The verifier added the consequence this analysis had understated: because `fast`/`quality` filter to `Locality == LocalityCloud` (`router.go:157,171`), they would find **zero eligible cloud adapters** and silently fall through to chain order. *"Currently masked by chain length 1, but a real latent defect that surfaces the moment claim 1's premise changes."* This directly drove the decision to place B-8 **inside** Phase 0 rather than after it.
- **`CostRank`/`QualityRank` have no assignment site in production** — zero-valued for every adapter, so `stableSortByCost`/`stableSortByQuality` (`router.go:202-224`) degenerate to chain order even if multi-provider chains were wired.

### One citation error found and corrected

Independent verification returned line numbers differing from this analysis in four places. Re-checked against source; **the verifier was correct in all four**. Corrections applied to `00-ARCHITECTURE_REVIEW.md` and `INTELLIGENCE_TRUTH_AUDIT.md`: `httpError` call site is `anthropic.go:281` (not 301); usage sum is `anthropic.go:300` (not 296); `parseRetryAfter` is `retry.go:42` (not 39); `byName` construction is `router.go:85-87` (not 86-88). No conclusion changed.

---

## Part B — Challenges to the five nominated conclusions

### B-1 · "Multi-provider already exists" — **REJECTED**

**Disproof attempt.** If true, some legal configuration should produce two-provider operation. There is none: `sdk.Config.Intelligence` is one field, no `AddProvider` exists, and the chain is constructed as `[]string{port.ProviderName()}` (`sdk/runtime.go:88`).

**Verdict: REJECTED as stated.** What exists is a *selection layer* (~85% built, 0% reachable). What does not exist is registration (30%/10%) and configuration (10%/10%) — precisely the parts that make a system multi-provider rather than merely multi-provider-*capable*.

**This is the most dangerous sentence in the program.** It invites "just flip a switch" planning, and a switch-flip alone yields a system whose routing hints silently do nothing (B-8). 

**Corrected claim:** *AWIS has a built and well-tested provider **selection** layer that has never executed, and does not have provider registration or configuration at all.*

### B-2 · "Driver architecture is needed" — **PARTIALLY REJECTED**

**Disproof attempt.** Is a driver/instance split *technically required* to add the six providers? **No.** Option 2 — a registry plus one adapter per provider — unblocks all six without any driver concept. The claim as stated is false.

**What survives.** Five of six targets share one wire protocol (`GAP_ANALYSIS` §3.1). Without a driver split you build five near-identical adapters, each re-implementing retry (with five chances to repeat the `retry.go:42` defect that already occurred at N=1) and prompts.

**Verdict: PARTIALLY REJECTED.** Driver architecture is **economically justified, not technically required**.

**The dependency worth stating:** this justification rests on *this specific target list*. If the targets were OpenAI and Gemini only — two distinct protocols — the driver split would buy nothing and Option 2 would be correct. The recommendation is contingent on the six-provider list, and should be revisited if that list changes.

### B-3 · "The Anthropic adapter should be generalized" — **REJECTED**

**Disproof attempt.** Generalising the working adapter to handle multiple providers *is* Option 4 (OpenAI-compatible layer), rejected on evidence: Gemini is not OpenAI-compatible, Anthropic is not either, and a lowest-common-denominator shape forecloses the native structured-output modes that fix G-4.

**Verdict: REJECTED — and the correct direction is the opposite.** The Anthropic adapter should be **narrowed**, not generalized. Phase 2 removes retry (`retry.go`), prompts (`anthropic.go:320-372`), redaction (`anthropic.go:406`), and usage-summing (`anthropic.go:300`) *upward* into a shared invoker, leaving ~150 lines of wire translation.

**Corrected claim:** *Generalize the layer above the adapter. Shrink the adapter itself.*

### B-4 · "Configuration must change" — **VERIFIED**

**Disproof attempt 1 — env vars only?** Partially viable for OpenAI and Gemini (key + model). Fails for OpenRouter, vLLM, LM Studio, and Ollama, which need `base_url` and per-instance settings. Four of six targets cannot be configured this way.

**Disproof attempt 2 — is B-3 really a *blocker*, or cosmetic?** It is a blocker. `validateConfigYAML` (`config.go:420-431`) is an allowlist; `awis config validate` **returns an error** on `openai_api_key`. A shipped CLI command fails on the configuration the feature requires.

**Disproof attempt 3 — smuggle it in via an existing key?** `api_key` is recognised, but `start.go` never reads it — only `anthropic_api_key` (`start.go:143`). No path exists.

**Verdict: VERIFIED.** The strongest claim in the package. Even an env-only or JSON-blob approach is still a configuration change; the only question is its medium.

### B-5 · "Existing router infrastructure is reusable" — **VERIFIED WITH CORRECTION**

**Disproof attempt.** Is the router's *algorithm* wrong for a registry world? The unusual choice is chain-as-selection-universe: registered-but-not-in-chain adapters are never routed to (`router.go:69-79`). Challenged as a possible design error — it is not. It is exactly the "enabled set" semantics a config-driven registry needs, and it is deliberate and documented.

**Verdict: VERIFIED — with a correction the original claim obscures.**

The router's **algorithm** is reusable (11 tests specify precisely the target semantics). Its **input types are not**: `Registration` (`router.go:24`) is a compile-time struct with no instance identity, no endpoint, no credential reference, and traits that no production code assigns. 

**Corrected claim:** *The router's selection logic and its test suite are reusable as-is. `Registration` must be replaced.* Reusability is real but does not extend to the type the router consumes — which is why Phase 1 exists as a separate phase.

---

## Part C — Self-challenge on this package's own conclusions

### C-1 · "GUI compatibility is free" — **VERIFIED**

**Challenge.** If `awis-server` later wires a provider, intelligence steps would begin executing where they never had — surely a behaviour change?

**Response.** That is *new* behaviour, not *regressed* behaviour. Nothing currently observable through the GUI changes, because nothing intelligence-related currently executes there (`cmd/awis-server/main.go:78-82` never sets `Intelligence`). **VERIFIED**, with the note that Phase 0 deliverable 0.8 makes this an explicit decision rather than an accident.

### C-2 · "Phase 0 ships standalone value" — **PARTIALLY REJECTED**

**Challenge.** Phase 0 adds no provider. What does a user get?

**Response.** Honestly: little that is directly user-visible. Two user-visible bug fixes (Retry-After actually honoured; the inert `intelligence:` key no longer silently lying) and one correctness fix nobody can observe until Phase 1 (B-8).

**Verdict: PARTIALLY REJECTED.** Phase 0's value is **optionality and de-risking**, not user-visible function. It should be justified to stakeholders on those terms, not oversold. It remains correctly sequenced first — it is where the latent B-8 defect gets closed before it can silently corrupt routing — but calling it "standalone value" overstates the case.

### C-3 · "41% built / 24% reachable" — **REJECTED as precision, VERIFIED as ordering**

**Challenge.** These are unweighted means over eleven judgment-assigned dimension scores. Nothing measures them. Two decimal-free percentages imply a rigour that does not exist.

**Verdict: REJECTED as precision.** The numbers should not be quoted as measurements or used in commitments. What survives disproof is the **ordering and the gap**: routing is the most-built and least-reachable dimension; configuration and model selection are near-zero on both axes; and reachability trails build across the board. That ordering is directly supported by code and is what the phasing depends on. Read the table as a ranking, not a metric.

### C-4 · "The smallest path is A→B, not C" — **VERIFIED**

**Challenge.** Steelman replacement: if the router's abstractions were wrong, reuse would be a trap.

**Response.** Nothing in the audit identifies a wrong abstraction in the router — only wrong *inputs* (B-6, B-8) and a missing *layer above* it. Against that, replacement discards 32 passing tests, a provider-neutral DSL and EventLog, a clean dependency graph with zero vendor identifiers in `core`/`engine`, and a correctly factored contract suite. No evidence supports paying that price. **VERIFIED.**

---

## Part D — What would falsify the recommendation

Stated so the recommendation is testable rather than merely argued.

| If this turned out to be true | The recommendation would change to |
|---|---|
| The target list narrows to two providers with distinct protocols (e.g. OpenAI + Gemini only) | Option 2 (registry, no driver split) — see B-2 |
| A hard requirement for cost-based routing arrives before Phase 2 | Pull Option 5's policy engine forward; accept higher cost |
| Phase 2's prompt byte-equality gate fails | Halt extraction; prompts are load-bearing in ways not yet understood |
| `sdk` surface freeze cannot be amended | Phase 0 blocks entirely; escalate to founder — there is no workaround for B-1 |
| A second consumer besides `awis start` needs providers urgently | Reorder: pull 0.8 forward as a Phase 0 blocking deliverable |

---

## Summary of verdicts

| Conclusion | Verdict |
|---|---|
| 9 independent factual claims | **VERIFIED** (all nine; one strengthened) |
| "Multi-provider already exists" | **REJECTED** → selection layer exists; registration/config do not |
| "Driver architecture is needed" | **PARTIALLY REJECTED** → economically justified, not technically required; contingent on the six-provider list |
| "Anthropic adapter should be generalized" | **REJECTED** → it should be *narrowed*; generalize the layer above it |
| "Configuration must change" | **VERIFIED** — strongest claim in the package |
| "Router infrastructure is reusable" | **VERIFIED with correction** → algorithm and tests yes; `Registration` no |
| "GUI compatibility is free" | **VERIFIED** |
| "Phase 0 ships standalone value" | **PARTIALLY REJECTED** → value is optionality, not user-visible function |
| "41% built / 24% reachable" | **REJECTED as precision**, VERIFIED as ordering |
| "Smallest path is A→B, not C" | **VERIFIED** |

Three of ten conclusions did not survive in their original form. The corrected claims are carried into `FINAL_VERDICT.md`.
