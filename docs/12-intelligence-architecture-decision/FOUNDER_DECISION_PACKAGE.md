# FOUNDER_DECISION_PACKAGE

**One question: which intelligence architecture does AWIS commit to for the next 3–5 years?**
Evidence: production code at `8a87f70`. Tier-2 investigation documents were validated, not assumed. Ten attacks were made on the recommendation; three forced corrections, which are incorporated.

---

## 1. Which architecture — and the one fact that decides it

# **Option B — Registry + Driver + Instance**

The decision looked like a complexity trade-off: A is simple, B adds a layer, C adds more. **Code shows that framing is wrong.**

`anthropic.Config` (`anthropic.go:70-87`) already exposes `APIKey`, `BaseURL`, `HTTPClient`, `MaxAttempts`, `ModelFast`, `ModelQuality`. Endpoint, credential, transport, retry, and model identity are **already externally parameterized**. A "driver" adds an auth-header shape and a wire-body shape. Option B adds a `driver:` selector in config and an instance id as the registry key.

> **Option B is not more abstract than Option A. It is less code — three packages instead of nine.**

The complexity objection inverts. B is the *simpler* option at the scale AWIS has committed to.

---

## 2. Why

| # | Reason | Evidence |
|---|---|---|
| **1** | Six of nine named providers already share one wire protocol; a seventh (Ollama) probably does | Present fact about the named list, not a forecast |
| **2** | B costs **nothing extra today** — the first new provider (OpenAI) is identical work under A and B | Divergence starts at the *third* provider |
| **3** | B degrades to A if protocols fragment | Worst case = A's package count + one config field. Bounded, non-scaling |
| **4** | A multiplies a **measured** defect | `Retry-After` never plumbed (`retry.go:42`, `anthropic.go:281`), documented as working, survived three audit passes — at N=1. A proposes N=9 |
| **5** | Instance identity preserves telemetry that consolidation would otherwise destroy | `Usage.Adapter` comes from `ProviderName()` (`anthropic.go:298`); without instance ids, six vendors all report `openai-chat` and per-vendor cost attribution dies |
| **6** | Option C's seam is **already built** | `Route()` is `Eligible()[0]` (`router.go:191-199`) over an ordered candidate list, covered by 11 tests. Deferring C costs ~nothing |

---

## 3. What was rejected

| Rejected | Verdict | Why |
|---|---|---|
| **Option A — Registry Only** | **Viable but expensive** — *not broken* | Hosts all nine named providers. Rejected for 9 packages vs 3, nine retry paths, and making every future compatible vendor a code change and a release. A is a decision to pay linearly, forever, for a problem B solves once. |
| **Option C — + Policy Engine** | **Correct end state, wrong sequence** | Requires cost data that structurally does not exist (`Usage` discards the input/output split at `anthropic.go:300`). AWIS's existing trivial policy layer, `model_hint`, is **provably inert**. Not rejected on merit — on timing. |
| **Universal OpenAI-compatible layer** | **Rejected on evidence** | Gemini is not OpenAI-compatible; neither is Anthropic. Would demote working tested code to a translation target. Adopted as *one driver*, never as *the seam*. |
| **Subsystem replacement** | **Re-confirmed rejected** | Discards 32 passing tests — 11 specifying exactly the target routing semantics — plus a provider-neutral DSL, EventLog, and dependency graph. |

---

## 4. What this choice eliminates

- Six near-duplicate OpenAI-compatible packages
- Eight redundant retry implementations, eight prompt renderers, eight usage parsers
- "New provider" as a release event, for any OpenAI-compatible vendor
- The B-6 silent-collision class
- A future migration from type-identity to instance-identity
- The open question of whether to standardise on OpenAI-compat — resolved: one driver, not the seam

## 5. What this choice enables

- Cost tracking (one invoker computes usage uniformly)
- Failover with stable semantics (shared prompt rendering makes providers comparable)
- Local + cloud tiering (`Locality` becomes configured, not hardcoded)
- Multi-account and multi-tenant operation
- The policy engine later, cheaply, at a seam that already exists
- Native structured output per protocol, rather than a lowest common denominator

**What it does not fix:** B-7 — keyless local providers (Ollama, LM Studio, vLLM) are blocked by `IsAvailable()` returning `cfg.APIKey != ""` (`anthropic.go:124`) under **all three options**. Option-independent; a prerequisite, not a differentiator.

---

## 6. Migration cost

**Low, and unusually so — because of a window that is open now and will close.**

| Item | Cost | Note |
|---|---|---|
| Existing workflows | **Zero** | Workflow YAML names capabilities only |
| DSL | **Zero** until model identity lands | `internal/dsl` untouched |
| Engine | **Zero** except one additive payload amendment | No vendor identifiers in `internal/engine` |
| GUI | **Zero** | Reads no provider data; and `awis-server` has never executed a real intelligence call |
| Existing tests | **Zero** — all 32 must pass unmodified | A test needing modification signals a design defect |
| Anthropic behaviour | **Must be proven preserved** | Prompt byte-equality gate on invoker extraction |
| `Usage` / EventLog reshape | **Cheap now, permanent later** | The usage triple has **no consumers today** (`trace.go:220-223` renders only `step_id` and `duration_ms`). Reshaping is currently free; once cost reporting ships, prior history becomes uninterpretable |

**The load-bearing timing fact: identity and usage shape are free to change today and permanently fixed once a reader exists.**

---

## 7. Implementation cost

Relative, not calendar. No roadmap is proposed — that is out of scope here.

| Work | Cost | Under A |
|---|---|---|
| Multi-provider arity + routing metadata from config | Small | Same |
| Config `providers:` block + credential refs + instance identity | Medium | Same (A needs identity too) |
| Invoker extraction (retry, prompts, redaction, usage) | Medium-high | **Skipped — and paid nine times instead** |
| First new driver (`openai-chat`) | Medium | **Identical** |
| Second new driver (`google-genai`) | Medium | Identical |
| Providers 3–9 | **Config entries** | **Six more packages** |

**A is cheaper only until the third provider. After that A pays per provider and B pays nothing.**

---

## 8. Long-term maintenance cost

The decisive dimension over a 3–5 year horizon.

| | **A** | **B** |
|---|---|---|
| Packages to maintain (9 providers) | 9 | **3** |
| Retry implementations | 9 | **1** |
| Prompt renderers | 9 | **1** |
| Structured-output negotiators | 9 | **3** |
| Cost of a 10th compatible vendor | Package + release | **Config line** |
| Cost of a novel protocol | Package | Driver — *equal* |
| Blast radius of a transport bug fix | 9 places | **1 place** |

The last row is the practical one. The `Retry-After` defect required a fix in one place and was missed anyway. Under A the same class of fix would require nine correct edits.

---

## 9. Required amendments

| # | Amendment | Gate |
|---|---|---|
| **AM-1** | `sdk.Config` freeze amendment (post-M08) for N providers + fallback chain | **Founder decision required. No workaround exists.** Precedent: ADJ-7, ADJ-8 |
| **AM-2** | `core.Usage` extended **once**, comprehensively — instance id, input/output split, cache tokens, latency, request id | Append-only log; a second amendment costs far more than unused fields |
| **AM-3** | `StepCompleted` payload amendment (additive) | Follows ADJ-8; replay-neutral (EDR-007 reads only `outputs`) |
| **AM-4** | `Registration` replaced — instance id, driver, endpoint, credential ref | Current compile-time struct cannot express an instance |
| **AM-5** | Config schema `providers:` block | `awis config validate` currently **errors** on any new provider key (`config.go:420-431`) |
| **AM-6** | `docs/PROVIDERS.md` drift corrections (DD-1…DD-5) | Documents a `model_hint` mapping and `Retry-After` handling that do not exist |

## 10. Implementation prerequisites

| # | Prerequisite |
|---|---|
| **P-1** | AM-1 approved — hard gate, nothing proceeds without it |
| **P-2** | Instance identity fixed **before** any registry is populated |
| **P-3** | `Locality`/`CostRank`/`QualityRank` sourced from config **in the same change** as multi-provider arity — otherwise the first multi-provider system ships with `fast`/`quality` hints that silently do nothing |
| **P-4** | `Usage` shape decided **before** the second driver |
| **P-5** | Driver conformance suite (extending `porttest`) **before** the second driver |
| **P-6** | Invoker extraction proven behaviour-preserving (prompt byte-equality) **before** the second driver |
| **P-7** | One gated live smoke per driver, **actually executed once** — `VERIFIED_DEFECT_REGISTER` D-05 records a shipped HTTP 404 a single live run would have caught |
| **P-8** | B-7 (keyless availability) solved independently — blocks local providers under every option |

---

## 11. The one risk this decision carries

**The `openai-chat` driver could become a conditional swamp.** OpenAI-compatibility is a marketing claim, not a specification: OpenRouter needs custom headers, vLLM omits parameters, Ollama's `/v1` is partial, Groq and Together diverge on tool calling.

This attack **survived adversarial review** and cannot be disproven in advance. It is accepted with a discipline and a tripwire:

- Quirks are carried as **data** (per-instance headers, declared parameter support), never as branches on instance identity.
- **Tripwire:** an instance requiring *behavioural* divergence gets its own driver — B degrading to A locally, by design.
- **Review trigger:** if behavioural divergence exceeds data divergence across the openai-chat set, revisit this ADR.

Option A does not avoid this problem; it pre-pays for it in all six cases whether or not divergence materialises.

---

## 12. What would change this recommendation

| If | Then |
|---|---|
| The nine-provider list is aspirational, real scope is 2–3 distinct protocols | **Option A** |
| `openai-chat` instances need behavioural, not data, divergence | **Option A** |
| Hard cost-routing requirement lands before the second driver | **Option C**, at higher cost |
| AM-1 is refused | No option reachable — escalation, not architecture |

**The single assumption this recommendation depends on: that the nine-provider list represents real intent.** If it does, B is correct and A is a standing tax. If it does not, A is correct and B is over-built. That question is the founder's to answer, and it is the only input the code cannot supply.
