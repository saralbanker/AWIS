# ADR-001 — AWIS Intelligence Architecture

| | |
|---|---|
| **Status** | Proposed — awaiting founder decision |
| **Date** | 2026-09-06 |
| **Scope** | The architecture AWIS commits to for intelligence provider support, 3–5 year horizon |
| **Evidence base** | Production code at `8a87f70`; Tier-2 investigation documents validated, not assumed |
| **Decision** | **Option B — Registry + Driver + Instance** |

---

## Context

AWIS supports one real intelligence provider (Anthropic). A selection layer — `CapabilityRouter`, `Dispatcher`, fallback chains, capability filtering — exists, is covered by 32 tests, and has **never executed in a shipped binary**: `sdk.NewRuntime` accepts one provider and builds a one-element chain (`sdk/runtime.go:86-89`).

The subsystem is provider-neutral where it matters. Workflow YAML names capabilities only; `internal/core` and `internal/engine` contain zero vendor identifiers; the GUI reads no provider data. Vendor coupling is confined to configuration (`anthropic_api_key`) and one composition root (`cmd/awis/start.go:36`).

Nine providers are in scope: Anthropic, OpenAI, Gemini, Ollama, OpenRouter, LM Studio, vLLM, Together, Groq.

---

## Decision

**Adopt Option B: a configuration-driven provider registry whose registry key is a provider *instance*, served by compiled *drivers* that each implement one wire protocol.**

- **Driver** — compiled Go package implementing one wire protocol. Three: `openai-chat`, `anthropic-messages`, `google-genai`.
- **Instance** — a configuration entry: `{id, driver, base_url, credential_ref, headers, models, traits}`. Nine or more.
- **Registry key** — the instance `id`, not `ProviderName()`.

Explicitly **not** adopted now: the policy engine (Option C). Its insertion seam is retained and unbuilt.

---

## Rationale

### R1 · "Driver" is not a new layer — it is the adapter that already exists

The premature-abstraction objection assumes B adds indirection A avoids. Code refutes this. `anthropic.Config` (`anthropic.go:70-87`) already exposes `APIKey`, `BaseURL`, `HTTPClient`, `MaxAttempts`, `ModelFast`, `ModelQuality`. Endpoint, credential, transport, retry, and model identity are **already externally parameterized**.

What a driver adds beyond this is an auth-header shape and a wire-body shape. What Option B adds beyond Option A is a `driver:` selector in config and an instance id as the registry key.

> **Option B is not more abstract than Option A. It is less code: three packages instead of nine.**

This inverts the complexity question, and it is the single most important finding in this decision.

### R2 · B strictly dominates A

If every future provider requires its own wire protocol, B collapses to one driver per provider — **matching A's package count and A's per-provider cost**, while retaining a small constant config indirection (one `driver:` field, one map key). B's worst case is A's expected case plus bounded, non-scaling overhead. B therefore cannot lose materially on protocol-fragmentation risk, the only axis where A is better positioned.

*(Phrasing corrected under adversarial review, Attack 2: the earlier claim that B "is" A at 1:1 overstated — the indirection persists, but it is configuration, not code paths, and does not grow with provider count.)*

The converse fails: A has no mechanism to benefit from convergence. Six OpenAI-compatible vendors remain six packages permanently.

Critically, this argument does **not** rely on the forbidden assumption that future providers resemble current ones. B's benefit is opportunistic and its downside is bounded by A.

### R3 · The maintenance argument is measured, not predicted

AWIS has already failed to maintain **one** correct retry implementation. `parseRetryAfter` (`retry.go:42`) has no non-test caller; `httpError` (`anthropic.go:281,308`) never receives `resp.Header`; `retryAfter` is therefore permanently zero while `docs/PROVIDERS.md` documents the opposite.

**A silent, documented-as-working defect at N=1.** Option A proposes nine independent retry implementations, nine prompt renderers, nine usage parsers, nine error mappers. This is not a predicted maintenance burden; it is an observed failure rate multiplied by nine.

### R4 · Instance identity is required to preserve telemetry granularity under consolidation

`Usage.Adapter` is populated from the adapter's compile-time `ProviderName()` (`anthropic.go:298`) and written to the EventLog (`emitters.go:99`, `emit.go:48`). Under a type-keyed registry, Option B's own consolidation would make **six OpenAI-compatible vendors all report `adapter: "openai-chat"`**, destroying the per-vendor cost attribution that FR-IL-09's usage triple exists to provide. Instance identity is what makes consolidation safe.

Secondarily — and this justification is *plausible but unevidenced in the repository*, per adversarial review Attack 4 — `router.go:85-87` builds `byName[r.Adapter.ProviderName()] = r` with plain map assignment, last-write-wins, no duplicate detection, so two vLLM servers or two OpenAI accounts would collide silently.

Option A can only avoid the collision by adopting instance identity — that is, by adopting half of Option B. **Fork 1 is unavoidable in every option**, which means the marginal decision is Fork 2 alone, and Fork 2 is where B saves six packages.

### R5 · Six of nine named providers already share one protocol

OpenAI, OpenRouter, Together, Groq, LM Studio, and vLLM are one wire protocol; Ollama is a seventh via its `/v1` surface. This is a **present fact about the named list**, not a forecast. Groq and Together were not in the original brief and arrived later — each a config line under B, each a package and a release under A.

### R6 · Option C is a component at an existing seam, not an architecture

`Route()` is `Eligible()[0]` (`router.go:191-199`), and `Eligible()` already returns an ordered `[]Decision` covered by 11 tests. A policy engine replaces `[0]` with a scoring function. **The seam is one line and it is already built and tested.**

Deferring C therefore costs approximately nothing, while adopting C now means building cost-based routing before cost data exists: `core.Usage` (`ports.go:100-107`) has no input/output split, and the split is discarded at `anthropic.go:300` before reaching the emitter.

Corroborating evidence that C is premature: AWIS's existing crude policy mechanism, `model_hint`, is **provably inert** — every adapter is registered `LocalityLocal` (`sdk/runtime.go:87`), `CostRank`/`QualityRank` have no production assignment site, and `fast`/`quality` filter on `LocalityCloud` and match nothing. AWIS has never operated the simplest possible policy layer.

---

## Consequences

### What this eliminates

| Eliminated | Because |
|---|---|
| Six near-duplicate OpenAI-compatible packages | One `openai-chat` driver serves all |
| Eight redundant retry implementations | Retry moves to a shared invoker |
| Eight redundant prompt renderers | Rendering moves above the driver |
| "New provider" as a release event (for compatible vendors) | It becomes a config edit |
| The B-6 collision class | Instance id replaces `ProviderName()` as key |
| A future migration from type-identity to instance-identity | Decided once, before history accumulates |
| Ambiguity about whether to standardise on OpenAI-compat | Resolved: it is one driver, never the seam |

### What this enables

| Enabled | Mechanism |
|---|---|
| Cost tracking | A single invoker computes usage/cost uniformly |
| Failover with stable semantics | Shared prompt rendering makes providers comparable |
| Local + cloud tiering | `Locality` becomes configured rather than hardcoded |
| Multi-account / multi-tenant | Instance identity |
| The policy engine, later, cheaply | Seam preserved at `Route()` |
| Native structured output per protocol | Three drivers, three native modes — not a lowest common denominator |

### What this does **not** fix

**B-7 is option-independent.** Keyless local providers (Ollama, LM Studio, vLLM) are blocked by `IsAvailable()` returning `cfg.APIKey != ""` (`anthropic.go:124`) with the router skipping false (`router.go:134`). This is an interface-level gap requiring a liveness probe the interface currently forbids. **All three options must solve it; it is not evidence for any of them.**

### Costs accepted

| Cost | Assessment |
|---|---|
| Two concepts (driver, instance) where there was one | Small; Fork 1 forces the instance concept regardless |
| Per-instance quirk handling (OpenRouter headers, vLLM param gaps) | Mitigated by keeping quirks as **data** (headers map, param declarations), never conditionals. **Threshold: an instance needing *behavioural* divergence gets its own driver.** This is B degrading to A locally, by design. |
| Instance ids must reach `Usage` and the EventLog | Additive payload amendment; the triple is currently write-only (no consumers), so the window to do this cheaply is open now |
| Invoker extraction changes shipped Anthropic behaviour | Must be gated on proven behaviour preservation, not assumed |

---

## Alternatives rejected

**Option A — Registry Only. Rejected: viable but expensive.**
Hosts all nine named providers; it is not broken. Rejected because it requires nine packages against B's three, multiplies a *measured* retry defect ninefold, cannot express two endpoints of one vendor without adopting instance identity anyway, and makes every future compatible vendor a code change and a release. **A is a decision to pay linearly, forever, for a problem B solves once.**

**Option C — Registry + Driver + Policy Engine. Rejected for now: correct end state, wrong sequence.**
Requires cost data that structurally does not exist. Its insertion seam is already implemented and tested, so deferral is nearly free and adoption is reversible-in. AWIS has never successfully operated its existing trivial policy layer. **Not rejected on merit — rejected on timing.**

**Universal OpenAI-compatible layer. Rejected on evidence.**
Gemini is not OpenAI-compatible (model in URL path, `contents`/`parts`, `responseSchema`); neither is Anthropic. Standardising on the OpenAI shape demotes working, tested code to a translation target and forecloses native structured-output modes. Adopted as **one driver**, rejected as **the seam**.

**Replacement of the intelligence subsystem. Rejected (re-confirmed).**
Discards 32 passing tests, 11 of which specify precisely the routing semantics being built toward, plus a provider-neutral DSL, EventLog, and dependency graph. No evidence identifies a wrong abstraction in the router — only wrong inputs and a missing layer above it.

---

## Required amendments

| # | Amendment | Why it is required |
|---|---|---|
| **AM-1** | **`sdk.Config` freeze amendment** (frozen post-M08) to admit N providers and a fallback chain | No workaround exists; B-1 cannot be fixed without touching this surface. Precedent: ADJ-7, ADJ-8 |
| **AM-2** | **`core.Usage` shape amendment** — extend once, comprehensively, including fields not yet computed (instance id, input/output split, cache tokens, latency, request id) | The EventLog is append-only. A second amendment costs far more than unused fields. The triple has **no consumers today**, so this window is unusually cheap |
| **AM-3** | **`StepCompleted` payload amendment** (additive) carrying the extended usage | Follows ADJ-8 precedent; replay-neutral, since the EDR-007 projection reads only `outputs` |
| **AM-4** | **`Registration` type replacement** — instance id, driver, endpoint, credential reference | The current compile-time struct cannot express an instance |
| **AM-5** | **Config schema amendment** — `providers:` block; `validateConfigYAML`'s allowlist (`config.go:420-431`) currently **errors** on any new provider key | `awis config validate` fails on the configuration the feature requires |
| **AM-6** | **`docs/PROVIDERS.md` drift corrections** (DD-1…DD-5) | It documents a `model_hint`→model mapping and `Retry-After` handling that do not exist |

---

## Implementation prerequisites

Ordered. Each must hold before the next is safe.

| # | Prerequisite | Rationale |
|---|---|---|
| **P-1** | AM-1 approved by the founder | Hard gate. Nothing proceeds without it |
| **P-2** | Instance identity fixed **before** any registry is populated | Identity propagates into `Usage` and an append-only log. Free now; permanently unfixable for prior history once a reader ships |
| **P-3** | `Locality`/`CostRank`/`QualityRank` sourced from configuration **in the same change** as multi-provider arity | Otherwise the first multi-provider system ships with `fast`/`quality` hints that silently do nothing — every adapter is `LocalityLocal` and both ranks are zero |
| **P-4** | `Usage` shape (AM-2) decided **before** the second driver | Avoids a second append-only amendment |
| **P-5** | Driver conformance suite (extending `porttest`) landed **before** the second driver | `porttest` is already adapter-agnostic and correctly factored; extend it rather than write a new one |
| **P-6** | Invoker extraction proven behaviour-preserving (prompt byte-equality against current Anthropic output) **before** the second driver | The riskiest step; must be proven, not assumed |
| **P-7** | One gated live smoke per driver, **actually executed at least once** | `VERIFIED_DEFECT_REGISTER` D-05 records a shipped HTTP 404 that a single live run would have caught |
| **P-8** | B-7 (keyless availability) solved independently | Option-independent; blocks Ollama, LM Studio, vLLM under every option |

---

## Decision review triggers

This ADR should be revisited if any of the following becomes true:

- The provider list narrows to **two providers with distinct protocols** — then Option A is correct and this ADR is over-engineered (R2 and R5 both weaken).
- OpenAI-compatible endpoints fragment enough that per-instance **behavioural** divergence exceeds per-instance data divergence — then B degrades to A, as designed.
- A hard cost-control requirement arrives before the second driver ships — then pull Option C forward and accept the higher cost.
- AM-1 is refused — then the entire program is blocked and requires a different escalation, not a different architecture.
