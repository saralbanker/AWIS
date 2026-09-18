# ADVERSARIAL_DECISION_REVIEW

An attempt to disprove **Option B**. Each attack is stated in its strongest form, then tested against code. A defence counts only if the attack was genuinely engaged.

Outcomes: **B survived seven attacks, was weakened by two, and forced one correction to the ADR's reasoning.** No attack defeated it. Two attacks identified real residual risk that the decision must carry explicitly.

---

## Attack 1 — "Driver architecture is premature abstraction"

**The attack.** AWIS has *one* provider. Committing to a three-driver taxonomy for a system that has never run two is textbook speculative architecture. The brief forbids assuming more abstraction is better.

**Test.** Does B add an abstraction layer A avoids?

`anthropic.Config` (`anthropic.go:70-87`) already exposes `APIKey`, `BaseURL`, `HTTPClient`, `MaxAttempts`, `ModelFast`, `ModelQuality`. Endpoint, credential, transport, retry policy, and model identity are **already externally parameterized**. A driver adds an auth-header shape and a wire-body shape. B adds a `driver:` config selector and an instance id as the map key.

**Verdict: ATTACK FAILS.** B introduces no new layer. It produces *fewer* packages (3 vs 9) and *less* duplicated logic. The premature-abstraction framing assumes indirection that the code shows already exists.

**Residual concession.** The taxonomy is derived from the nine named providers. **If that list is aspirational rather than committed intent, this rationale weakens** — recorded as an ADR review trigger.

---

## Attack 2 — "B ⊇ A is a mathematical trick, not an engineering truth"

**The attack.** The dominance claim says B collapses to A when every provider needs its own protocol. False: B-at-1:1 still carries driver/instance indirection, instance ids in config, and instance ids in the EventLog. That is **A plus overhead**, not A.

**Test.** What survives at 1:1? A `driver:` field naming a package that serves one vendor, and an instance id that equals the vendor name. Both are configuration, not code paths.

**Verdict: ATTACK PARTIALLY SUCCEEDS — ADR corrected.**

The attack is right that "which *is* Option A" overstates. The accurate claim: **B degrades to A's package count and A's per-provider cost, retaining a small config indirection (one field, one map key).** The dominance conclusion holds — the overhead is bounded, constant, and does not scale with providers — but the ADR's phrasing was too strong and has been amended.

---

## Attack 3 — "The `openai-chat` driver will become a conditional swamp"

**The attack.** OpenAI-compatibility is a marketing claim, not a specification. OpenRouter needs custom headers; vLLM omits parameters; Ollama's `/v1` is partial; Groq and Together diverge on tool calling and streaming. One driver serving six vendors becomes a mass of `if instance == "openrouter"` conditionals — worse than six honest packages.

**Test.** This cannot be disproven in advance. It is a real, empirically-grounded failure mode of shared-driver designs.

**Verdict: ATTACK SURVIVES AS RESIDUAL RISK. This is the single place Option B can genuinely fail.**

The mitigation is a discipline, not a proof:
- Quirks are carried as **data** — a per-instance `headers` map, declared parameter support, declared capabilities — never as branches on instance identity.
- **Tripwire:** an instance requiring *behavioural* divergence (not data divergence) gets its own driver. That is B degrading to A locally, by design.
- **Review trigger:** if behavioural divergence exceeds data divergence across the openai-chat instance set, revisit this ADR.

The decision accepts this risk explicitly rather than claiming it away. Notably, A does not avoid the underlying problem — it pre-pays for it in all six cases whether or not divergence materialises.

---

## Attack 4 — "Instance identity is a solution to an imagined problem"

**The attack.** ADR §R4 justifies instance identity with prod/dev vLLM and multi-account deployments. **No evidence in the repository shows anyone needs this.** It is a plausible-sounding requirement, invented to justify the decision.

**Test.** Searched for evidence of multi-instance demand. **Found none.** The attack is correct that R4's stated justification is unevidenced.

**Verdict: ATTACK PARTIALLY SUCCEEDS — but the conclusion survives on different, evidenced grounds.**

`Usage.Adapter` is populated from the adapter's compile-time `ProviderName()` (`anthropic.go:298`) and written to the EventLog (`emitters.go:99`, `emit.go:48`). Under a type-keyed registry, **six OpenAI-compatible vendors all report `adapter: "openai-chat"`.** Per-vendor cost attribution — the explicit purpose of the FR-IL-09 usage triple — becomes impossible. That is not a hypothetical deployment shape; it is a direct consequence of Option B's own package consolidation.

**So instance identity is not required by imagined multi-tenancy. It is required because B's consolidation would otherwise destroy telemetry granularity that already exists under A.** The ADR rationale has been strengthened accordingly; the speculative justification is demoted to a secondary benefit.

---

## Attack 5 — "The nine-retry maintenance argument is weak induction"

**The attack.** ADR §R3 generalises from one bug (`Retry-After` never plumbed) to "nine implementations will be worse." One defect is not a failure rate. Deductively invalid.

**Test.** The attack is formally correct — this is inductive, not deductive.

But the signal is not merely that a bug exists. The `Retry-After` defect was **documented as working** in `docs/PROVIDERS.md`, and survived `RELEASE_CANDIDATE_AUDIT`, `FINAL_VERDICT`, and `REPOSITORY_HEALTH_REPORT` — multiple adversarial review passes over this exact adapter. The evidence is about *detection*, not incidence: the review process did not catch a silent transport defect at N=1.

**Verdict: ATTACK SURVIVES, ARGUMENT WEAKENED BUT INTACT.** R3 should be read as a calibrated risk indicator, not proof. It is corroborative; the decision does not rest on it.

---

## Attack 6 — "Choose A now, migrate to B when the third compatible provider arrives"

**The strongest attack in this review.** Nothing forces the decision today. Build A, add OpenAI, add Gemini, and adopt drivers only when a third OpenAI-compatible vendor makes duplication concrete. Decide with evidence instead of prediction.

**Test.** What does deferral actually cost?

1. **Wasted packages:** two OpenAI-compatible packages later merged into one driver. Modest.
2. **Identity discontinuity:** if A ships type-identity and history accumulates, migration splits the log. *Qualification found in this review:* the usage triple has **no consumers today** — `formatTraceEvent`'s `StepCompleted` case renders only `step_id` and `duration_ms` (`trace.go:220-223`), and no API, CLI, or GUI path reads `adapter`/`model`/`tokens_used`. The discontinuity is currently **inert**. It becomes permanent the moment a reader ships — which cost reporting requires.
3. **Schedule risk:** the migration point is the arrival of a third provider, i.e. exactly when delivery pressure is highest.

**And the decisive counter-test: what does B cost *extra*, today?**

Under A, OpenAI is a new package. Under B, OpenAI is a new driver. **Identical work.** The divergence begins at the *third* provider. **B carries no upfront premium over A for the first new provider.**

**Verdict: ATTACK FAILS.** The deferral argument assumes B costs more now. It does not. Choosing B costs nothing extra today, avoids a migration at the worst moment, and closes the identity question while it is still free. Deferral buys optionality that is already free and sells a fix that becomes permanent.

---

## Attack 7 — "Deferring Option C is a false economy"

**The attack.** Retrofitting a policy engine into a live router changes selection under running workflows. Build it now, while nothing depends on selection behaviour.

**Test.** `Route()` is `list[0]` over `Eligible()`'s ordered `[]Decision` (`router.go:191-199`), covered by 11 tests. Adding a scoring function changes behaviour **only when a policy is configured**; absent configuration, `[0]` is preserved and existing tests pass unmodified.

**Verdict: ATTACK FAILS.** The retrofit is opt-in at an existing, tested seam.

Corroboration that C is premature: `model_hint` — AWIS's existing trivial policy layer — is **provably inert** (`LocalityLocal` hardcoded at `sdk/runtime.go:87`; `CostRank`/`QualityRank` never assigned; `fast`/`quality` filter on `LocalityCloud` and match nothing). Building sophisticated routing before the trivial layer has ever worked is not a defensible sequence.

---

## Attack 8 — "Option C is required because cost control is inevitable"

**The attack.** Multi-provider without cost routing is incomplete. Cost pressure is certain, so C is not optional.

**Test.** Cost routing requires cost data. `core.Usage` (`ports.go:100-107`) has no input/output split; it is summed at `anthropic.go:300` and **never exists past the adapter boundary**. C cannot route on cost until AM-2 lands — which is Option B's work, not C's.

**Verdict: ATTACK FAILS, AND INVERTS.** Cost control being inevitable is an argument *for* completing B first: B is C's prerequisite. Building C now would produce a policy engine with nothing to evaluate.

---

## Attack 9 — "Rejecting Option A understates it"

**The attack.** The package overstates A's weakness. A hosts all nine named providers. Calling it inadequate is rhetorical.

**Test.** Correct — and the analysis agrees. `FUTURE_COMPATIBILITY_ANALYSIS` §4.1 states plainly that **A does not fail on any single named provider** and that claiming otherwise would overstate.

A fails on two things that are not single providers: the tenth compatible vendor (a release under A, a config line under B — and Groq and Together, both late additions, are evidence the long tail is real), and telemetry granularity under consolidation (Attack 4).

**Verdict: ATTACK FAILS ON SUBSTANCE, SUCCEEDS ON FRAMING.** A is *viable and expensive*, not broken. The ADR states this. Any summary describing A as unable to support these providers is wrong and should be corrected.

---

## Attack 10 — "B-7 invalidates the comparison"

**The attack.** Keyless local providers are blocked by `IsAvailable()` under every option. If Ollama, LM Studio, and vLLM cannot run under B either, B's advantage on local models is illusory.

**Test.** Correct. `anthropic.go:124` returns `cfg.APIKey != ""`; `router.go:134` skips adapters returning false. This is an interface-level gap affecting A, B, and C identically.

**Verdict: ATTACK SUCCEEDS AS A CORRECTION, NOT AS A REFUTATION.** B-7 is option-independent and must not be scored for or against any option. It is listed as prerequisite **P-8** precisely because it is orthogonal. No comparison claim in this package rests on it.

---

## Summary

| # | Attack | Verdict |
|---|---|---|
| 1 | Premature abstraction | **Fails** — B adds no layer; it removes six packages |
| 2 | "B ⊇ A" is a trick | **Partially succeeds** — ADR phrasing corrected; conclusion holds |
| 3 | `openai-chat` becomes a conditional swamp | **Survives — the one real residual risk**; tripwire recorded |
| 4 | Instance identity solves an imagined problem | **Partially succeeds** — rationale replaced with an evidenced one (telemetry granularity) |
| 5 | Nine-retry argument is weak induction | **Survives** — R3 demoted to corroborative |
| 6 | Choose A now, migrate later | **Fails** — B has no upfront premium; first provider costs the same |
| 7 | Deferring C is false economy | **Fails** — opt-in retrofit at a tested seam |
| 8 | C is required, cost control inevitable | **Fails and inverts** — B is C's prerequisite |
| 9 | Rejecting A understates it | **Fails on substance, succeeds on framing** — A is expensive, not broken |
| 10 | B-7 invalidates the comparison | **Succeeds as correction** — option-independent, scored for none |

**Three corrections were forced into the decision package** (Attacks 2, 4, 9), and **one residual risk is carried explicitly** (Attack 3). The recommendation stands.

## What would falsify Option B

| If this were true | Recommendation becomes |
|---|---|
| The nine-provider list is aspirational, and real scope is 2–3 distinct protocols | **Option A** |
| `openai-chat` instances need behavioural, not data, divergence | **Option A** (B degrades there by design) |
| A hard cost-routing requirement lands before the second driver | **Option C**, at higher cost |
| AM-1 (`sdk.Config` freeze amendment) is refused | No option is reachable — escalation, not architecture |
