# FINAL_VERDICT — Intelligence Architecture Commitment

**Question:** which intelligence architecture should AWIS commit to for the next 3–5 years?
**Evidence:** production code at `8a87f70`. Ten attacks made on the recommendation; three forced corrections, all incorporated. No attack defeated it.

---

# **OPTION B — Registry + Driver + Instance**

---

## The decision in one fact

The choice appeared to be a complexity trade-off: A simple, B adds a layer, C adds more. **Code shows that framing is inverted.**

`anthropic.Config` (`internal/intelligence/adapters/anthropic/anthropic.go:70-87`) already exposes `APIKey`, `BaseURL`, `HTTPClient`, `MaxAttempts`, `ModelFast`, `ModelQuality`. Endpoint, credential, transport, retry policy, and model identity are **already externally parameterized**. What a "driver" adds beyond this is an auth-header shape and a wire-body shape. What Option B adds beyond Option A is a `driver:` selector in configuration and an instance id as the registry key.

> **Option B does not add an architectural layer. It is three packages where Option A is nine.**
>
> **At the scale AWIS has committed to, B is the simpler architecture, not the more complex one.**

---

## Evidence

**1 · Six of nine named providers already share one wire protocol.** OpenAI, OpenRouter, Together, Groq, LM Studio, and vLLM are all OpenAI-compatible; Ollama is a seventh via its `/v1` surface. Anthropic and Gemini are genuinely distinct. This is a present fact about the named list, not a forecast — and Groq and Together, both late additions to the list, are direct evidence that the long tail is real.

**2 · Option B carries no upfront premium.** Under A, OpenAI is a new package. Under B, OpenAI is a new driver. **Identical work.** Divergence begins at the third provider. Every argument for deferring the decision assumes B costs more today; it does not.

**3 · Option B degrades to Option A.** If protocols fragment, B collapses to one driver per provider — matching A's package count and per-provider cost, retaining one config field and one map key. Bounded, non-scaling. B's downside is capped by A's baseline, so the forbidden assumption *"future providers will behave like current providers"* is **not load-bearing**.

**4 · Option A multiplies a measured defect.** `parseRetryAfter` has no non-test caller (`retry.go:42`); `httpError` never receives `resp.Header` (`anthropic.go:281,308`); `Retry-After` is therefore never honoured while `docs/PROVIDERS.md` documents the opposite. This survived three adversarial audit passes — **at N=1**. Option A proposes nine independent retry implementations.

**5 · Instance identity preserves telemetry that consolidation would destroy.** `Usage.Adapter` is populated from `ProviderName()` (`anthropic.go:298`) and written to the EventLog. Without instance ids, B's own consolidation would make six vendors all report `adapter: "openai-chat"`, killing the per-vendor cost attribution the FR-IL-09 triple exists to provide.

**6 · Option C's insertion seam is already built.** `Route()` is `Eligible()[0]` (`router.go:191-199`) over an ordered `[]Decision`, covered by 11 tests. A policy engine replaces one line. Deferring C costs approximately nothing; adopting it now means building cost routing before cost data exists — `Usage` discards the input/output split at `anthropic.go:300`.

---

## Rejected

**Option A — Registry Only. Viable but expensive. Not broken.**
It hosts all nine named providers, and any claim otherwise overstates. Rejected because it requires nine packages against three, nine retry paths against one, cannot express two endpoints of one vendor without adopting instance identity anyway, and makes every future OpenAI-compatible vendor a code change and a release. **A is a decision to pay linearly, forever, for a problem B solves once.**

**Option C — Registry + Driver + Policy Engine. Correct end state, wrong sequence.**
Requires cost data that structurally does not exist. AWIS's existing trivial policy layer — `model_hint` — is **provably inert**: every adapter is registered `LocalityLocal` (`sdk/runtime.go:87`), `CostRank`/`QualityRank` have no production assignment site, and `fast`/`quality` filter on `LocalityCloud` and match nothing. Building sophisticated routing before the trivial layer has ever worked is not defensible. **Rejected on timing, not merit** — and choosing B schedules C rather than precluding it.

---

## Corrections forced by adversarial review

Three attacks partially succeeded and are reflected in the package:

1. **"B ⊇ A" overstated.** B at 1:1 is A's package count *plus* a small constant config indirection — not literally A. The dominance conclusion holds; the phrasing was amended.
2. **Instance identity's original justification was unevidenced.** Prod/dev vLLM and multi-account deployments are plausible but unsupported by anything in the repository. The rationale was replaced with an evidenced one: telemetry granularity under consolidation.
3. **Rejecting A was framed too strongly.** A is expensive, not incapable. Corrected throughout.

One risk **survived** and is carried explicitly: **the `openai-chat` driver could become a conditional swamp.** OpenAI-compatibility is a marketing claim, not a specification. Mitigated by keeping quirks as data rather than conditionals, with a tripwire — an instance needing *behavioural* divergence gets its own driver, which is B degrading to A locally, by design. Option A does not avoid this problem; it pre-pays for it in all six cases regardless.

---

## Required amendments

| # | Amendment |
|---|---|
| **AM-1** | `sdk.Config` freeze amendment (post-M08) for N providers + fallback chain — **founder decision required; no workaround exists** |
| **AM-2** | `core.Usage` extended **once**, comprehensively (instance id, input/output split, cache tokens, latency, request id) |
| **AM-3** | `StepCompleted` payload amendment (additive, replay-neutral) |
| **AM-4** | `Registration` replaced with an instance-shaped type |
| **AM-5** | Config schema `providers:` block — `awis config validate` currently **errors** on any new provider key |
| **AM-6** | `docs/PROVIDERS.md` drift corrections (DD-1…DD-5) |

## Implementation prerequisites

| # | Prerequisite |
|---|---|
| **P-1** | AM-1 approved — hard gate |
| **P-2** | Instance identity fixed **before** any registry is populated |
| **P-3** | `Locality`/`CostRank`/`QualityRank` from config **in the same change** as multi-provider arity |
| **P-4** | `Usage` shape decided **before** the second driver |
| **P-5** | Driver conformance suite (extending `porttest`) **before** the second driver |
| **P-6** | Invoker extraction proven behaviour-preserving (prompt byte-equality) |
| **P-7** | One gated live smoke per driver, **actually executed once** |
| **P-8** | B-7 (keyless availability) solved independently — option-independent |

---

## The timing argument

Two things are **free to decide today and permanently fixed later**:

- **Identity.** `Usage.Adapter` lands in an append-only log. It currently has **no consumers** — `formatTraceEvent` renders only `step_id` and `duration_ms` (`trace.go:220-223`), and no API, CLI, or GUI path reads it. Reshaping is free now; once cost reporting ships, prior history becomes uninterpretable.
- **Usage shape.** Same log, same window.

This is the strongest reason to decide now rather than defer. The deferral case ("choose A, migrate when a third compatible provider arrives") assumes B costs more today — it does not — and schedules the migration for exactly the moment delivery pressure is highest.

---

## Verdict

# **OPTION B**

Adopt a configuration-driven provider registry keyed on provider **instances**, served by compiled **drivers** that each implement one wire protocol. Three drivers — `openai-chat`, `anthropic-messages`, `google-genai` — cover all nine named providers.

Do **not** build the policy engine. Its seam exists, is tested, and stays unbuilt until cost data exists and demand is real.

**One assumption this rests on, and it is the founder's to confirm: that the nine-provider list represents real intent.** If it does, B is correct and A is a standing tax. If the true scope is two or three providers with distinct protocols, A is correct and B is over-built. **That is the only input the code cannot supply** — everything else in this decision is settled by evidence.
