# OPTION_COMPARISON_MATRIX

Side-by-side comparison of Options A, B, and C against repository truth at `8a87f70`.

---

## 0. First, a definitional correction

The three options as posed are not three points on one axis. Code inspection shows they resolve into **two independent binary decisions plus one deferred feature**:

| Fork | Question | Consequence |
|---|---|---|
| **Fork 1** | Is the registry key an adapter **type** or a configured **instance**? | Determines whether two endpoints of the same vendor (prod + dev vLLM, two OpenAI accounts) can coexist. |
| **Fork 2** | May one adapter type serve **multiple vendors**? | Determines package count: 3 vs 9. |
| **Fork 3** | Is selection `candidates[0]` or a **scoring function**? | This is Option C, and it is a *component*, not an architecture. |

**Option A** = Fork 2 "no". **Option B** = Fork 1 "instance" + Fork 2 "yes". **Option C** = B + Fork 3.

This matters because it exposes something the option labels hide: **Fork 1 is unavoidable in all three options.** Option A must also answer it, and if A answers "instance", A has already built the substrate B needs.

---

## 1. What "driver" actually costs — measured, not assumed

The strongest objection to Option B is that a driver layer is premature abstraction. Code contradicts this.

`internal/intelligence/adapters/anthropic/anthropic.go:70-87` — the adapter's existing `Config`:

```go
type Config struct {
    APIKey       string        // credential
    BaseURL      string        // endpoint override
    HTTPClient   *http.Client  // transport
    MaxAttempts  int           // retry policy
    ModelFast    string        // model identity
    ModelQuality string        // model identity
}
```

**The adapter is already a parameterized driver in everything but name.** Endpoint, credential, transport, retry, and model identity are already externally configurable. What a "driver" adds over this is precisely two things: an auth-header shape and a wire-body shape.

Therefore:

> **Option B does not add an architectural layer. It adds a `driver:` selector in config and changes the registry key from a compile-time constant to a configured id. Option B is *less code* than Option A — three packages instead of nine.**

The premature-abstraction objection assumes B introduces indirection A avoids. It does not. The comparison below is scored on that basis.

---

## 2. Comparison matrix

| Dimension | **A — Registry Only** | **B — Registry + Driver + Instance** | **C — B + Policy Engine** |
|---|---|---|---|
| **Go packages for the 9 named providers** | 9 | **3** (`openai-chat`, `anthropic-messages`, `google-genai`) | 3 |
| **Providers per package (max)** | 1 | **6–7** | 6–7 |
| **New OpenAI-compatible vendor (Groq, Together, Cerebras…)** | New package + release | **Config entry** | Config entry |
| **Two endpoints of one vendor** (prod+dev vLLM, two OpenAI keys) | ❌ collides at `router.go:85-87` unless Fork 1 = instance | ✅ | ✅ |
| **Retry implementations to maintain** | **9** | **1** (shared invoker) | 1 |
| **Prompt renderers to maintain** | 9 | 1 | 1 |
| **Structured-output negotiators** | 9 | 3 (one per wire shape) | 3 |
| **Novel future protocol** | New package | New driver (**degrades to A**) | New driver |
| **Worst case if OpenAI-compat fragments** | = A | **= A** (B ⊇ A) | = A |
| **Requires cost data that does not exist** | No | No | ⚠️ **Yes** — `Usage` cannot express cost |
| **Insertion seam already built** | n/a | Router core + `Eligible()` | ✅ `Route()` = `Eligible()[0]`, `router.go:191-199` |
| **Blockers cleared** (of B-1…B-8) | 5 directly; B-6 only if Fork 1 = instance | **8** | 8 |
| **Preserves 32 seam tests** | ✅ | ✅ | ✅ |
| **Workflow / DSL / GUI compatibility** | ✅ free | ✅ free | ⚠️ DSL extension for `requires{}` |
| **Implementation cost** | Low per provider, **9×** | **Medium once** | High |
| **Maintenance cost** | **9× surface** | 3× surface | 3× + policy surface |
| **Rewrite debt created** | **High** (see §3) | **None** | None |
| **Speculative content** | None | None | ⚠️ **Routing policy with no operator demand** |

---

## 3. The rewrite debt in Option A — stated precisely

Option A works. It is not incorrect. Its cost is concentrated in three places, all evidenced:

**3.1 · Nine independent retry implementations.**
The repository has already demonstrated it cannot maintain **one** correct retry implementation. `parseRetryAfter` (`retry.go:42`) has no non-test caller; `httpError` (`anthropic.go:281,308`) never receives `resp.Header`; `retryAfter` is therefore always zero and backoff is always exponential — while `docs/PROVIDERS.md` documents the opposite. **That is a silent, documented-as-working defect at N=1.** Option A proposes N=9. This is the single most concrete argument in the matrix, because it is a measured failure rate rather than a predicted one.

**3.2 · Identity recorded in an append-only log.**
`Usage.Adapter` is set from the adapter's compile-time `ProviderName()` (`anthropic.go:298`, `null.go:52`) and written into the `StepCompleted` payload (`engine/emitters.go:99`, `engine/emit.go:48`). If A ships with `adapter: "openrouter"` and identity later becomes an instance id, historical rows and new rows describe different things in a log that cannot be rewritten.

*Honest qualification, found during this review:* **nothing currently reads that value.** `formatTraceEvent`'s `StepCompleted` case renders only `step_id` and `duration_ms` (`cmd/awis/trace.go:220-223`); no API endpoint, CLI command, or GUI path reads `adapter`, `model`, or `tokens_used`. The triple is write-only telemetry today, so the discontinuity is presently **inert**. It becomes real the moment a reader exists — which is exactly what cost reporting requires. The argument is therefore weaker than "irreversible today" and stronger than "cosmetic": *it is free to fix now and permanently unfixable for pre-existing history once a reader ships.*

**3.3 · Divergent prompt semantics.**
Prompt construction lives inside the adapter (`anthropic.go:320-372`). Under A, nine adapters mean nine prompt renderers, so a failover between two providers silently sends a different prompt and the two outcomes are not comparable.

---

## 4. Why Option C is not an architecture

`internal/intelligence/router.go:191-199`:

```go
func (r *CapabilityRouter) Route(capability, hint string, required bool) (Decision, error) {
    list := r.Eligible(capability, hint)   // already an ORDERED CANDIDATE LIST
    ...
    return list[0], nil                    // ← the policy seam, one line
}
```

`Eligible()` already returns `[]Decision` in preference order. A policy engine replaces `list[0]` with a scoring function over that list. **The insertion point exists, is implemented, and is covered by 11 tests.**

So Option C differs from Option B by one component attached at an existing seam — not by an architecture. Two consequences:

- **Choosing B does not reject C.** It schedules it.
- **Choosing C now means building cost-based routing before cost data exists.** `core.Usage` (`ports.go:100-107`) has no input/output split; the split is discarded at `anthropic.go:300` and never reaches the emitter. Cost routing has nothing to route on.

Additional evidence that C is premature: AWIS's existing crude policy mechanism — `model_hint` — is **provably inert**. Every adapter is registered `LocalityLocal` (`sdk/runtime.go:87`), `CostRank`/`QualityRank` have no assignment site in production, and `fast`/`quality` filter on `LocalityCloud` and therefore match nothing. AWIS has never successfully operated the simplest possible policy layer. Building a sophisticated one is not the next step.

---

## 5. Dominance analysis

The decisive structural relationship:

> **B ⊇ A.** If every future provider turns out to need its own wire protocol, Option B collapses to one driver per provider — which *is* Option A. B's worst case equals A's expected case.

Therefore B cannot be worse than A on protocol-fragmentation risk, which is the only risk A is better positioned against. Against the forbidden assumption *"future providers will behave like current providers"*: **B does not assume protocol convergence. It merely benefits from it where it occurs and degrades to A where it does not.**

The converse does not hold. A has no mechanism to *gain* from convergence: six OpenAI-compatible vendors remain six packages forever.

| | A better | Equal | B better |
|---|:--:|:--:|:--:|
| Protocol fragmentation (all novel) | | ✅ | |
| Protocol convergence (6–7 share one) | | | ✅ |
| Two endpoints, one vendor | | | ✅ |
| Package count | | | ✅ |
| Retry correctness surface | | | ✅ |
| Conceptual simplicity | ⚠️ marginal | | |
| Cost to reach first second provider | ⚠️ marginal | | |

A wins two cells, both marginal and both short-horizon. B wins five, all structural and all compounding.

---

## 6. Scoring against the founder's stated horizon (3–5 years)

| Criterion | A | B | C |
|---|:--:|:--:|:--:|
| Survives the 9 named providers without rewrite | ⚠️ functionally yes, at 9× maintenance | ✅ | ✅ |
| Survives a 10th OpenAI-compatible vendor | ❌ code change + release | ✅ config | ✅ config |
| Survives a novel protocol | ✅ | ✅ | ✅ |
| Survives two endpoints of one vendor | ❌ (unless Fork 1 = instance) | ✅ | ✅ |
| Builds nothing speculative | ✅ | ✅ | ❌ |
| Defers what has no demand | ✅ | ✅ | ❌ |
| **Verdict** | **Viable, expensive** | **Recommended** | **Correct later, premature now** |
