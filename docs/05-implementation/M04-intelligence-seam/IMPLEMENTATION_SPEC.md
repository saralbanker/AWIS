# M04 — Implementation Spec (compiled 2026-07-03 from IMP §27.M4 + Blueprint §13–§17 + CONTRA-3 per IKB §4 row)

## Objective (IMP §27.M4)
IntelligencePort seam live without any cloud code: NullAdapter + CapabilityRouter + context-budget
enforcement. `Out:` `internal/intelligence` + adapter subpackage. `Dep:` M01 only.
**Risk:** over-building the router — "V1 router may be internally simple while honoring the public
routing spec (Finalization open item)."

## Tasks

### T1 — Complete the M04-owned core shapes (sanctioned `internal/core` edit)
The four placeholder shells marked "Shape completed at M04": `DraftResponse`, `SynthesisResponse`,
`Capability`, `Example` (ports.go). Blueprint §13 defines the request types (already frozen at M01)
but NOT the response shapes — the owning milestone defines them, minimal:
- `Example{Input map[string]any; Output map[string]any}` (few-shot pair)
- `Capability{Name string}` (a provider declares capabilities by name: draft/embed/synthesize/classify)
- `DraftResponse{Output map[string]any; Usage Usage}` — Output matches DraftRequest.Schema
- `SynthesisResponse{Text string; Usage Usage}` — NullAdapter must return "The record is silent."
- `Usage{Adapter string; Model string; TokensUsed int}` — new type; feeds the FR-IL-09 StepCompleted
  payload `{adapter, model, tokens_used}` at M06. sdk aliases updated (pure aliases only).

### T2 — NullAdapter (`internal/intelligence/adapters/null`)
Blueprint §14 verbatim behaviors (FR-IL-01):
- `Draft`: deterministic fixture or error **per configuration** (constructor takes fixture map
  and/or error toggle; zero-config default = empty Output, deterministic)
- `Embed`: zero vector (CONTRA-3: Embed unavailable in V1; the zero vector is the null response)
- `Synthesize`: `"The record is silent."`
- `Classify`: first category, confidence 0.0 (FR-IL-10: implements the placeholder)
- `IsAvailable()`: **false, always** · `ProviderName()`: `"null"` · `Capabilities()`: all four.
Usage on all responses: `{Adapter: "null", Model: "null", TokensUsed: 0}`.

### T3 — CapabilityRouter (`internal/intelligence`)
Blueprint §17 decision tree, every branch (IMP Val: "router unit tests for every branch"):
1. capable+available set empty → `required=true` ⇒ typed `ErrCapabilityUnavailable`;
   `required=false` ⇒ typed `FallbackSignal` outcome (activation itself is M06 engine behavior —
   the router only *reports* the decision; do not reach into step execution).
2. hint `local` → prefer local trait, else cloud; `fast` → cheapest cloud; `quality` →
   highest-quality cloud; nil → fallback_chain order.
3. Provider call fails → next in fallback_chain → exhaustion maps per `required` as branch 1.
Adapter traits (locality, cost rank, quality rank) are **registration metadata**
(`Registration{Adapter, Locality, CostRank, QualityRank}`) — internal config, not frozen surface
(EDR-009). NullAdapter is always the final chain entry (FR-IL-01) but is never *selected* as
available (IsAvailable=false) — it is the explicit degraded-mode adapter callers may use directly.
Router selection honors `IsAvailable()` at decision time.

### T4 — Context-budget enforcement (FR-IL-08)
Enforced in the dispatch path **before** any adapter call: over-budget ⇒ typed
`ErrContextBudgetExceeded{Budget, Estimated}` — never truncate. Measurement: no tokenizer exists in
V1 and no frozen text prescribes one ⇒ conservative estimator `estimateTokens = ceil(len(bytes)/4)`
recorded as **EDR-008** (provider-actual counts arrive with FR-IL-09 at M16; estimator is
enforcement-only, never billed). Budget source: `IntelReq.ContextBudget`; 0 = no budget declared ⇒
no enforcement.

### T5 — Dispatch façade + IntelligencePort contract suite (IMP DoD)
`Dispatcher.Draft/Synthesize/...(ctx, IntelReq, req)` = budget check → route → call → on call
error walk chain (§17 tree bottom). Contract suite `internal/intelligence/porttest` (pattern:
`storagetest`): adapter-agnostic assertions any IntelligencePort implementation must satisfy
(deterministic re-call, capability list consistency with method behavior, null-safety); run against
NullAdapter now, AnthropicAdapter at M16 (IMP §27.M16 "M4 contract suite").

## Scope walls
- NO cloud/HTTP code anywhere (Repo state: "no cloud code anywhere"). NO Ollama/OpenAI (FR-IL-03/04 = V2).
- NO engine wiring (StepContext.Intelligence stays as-is; M06 consumes the seam).
- NO config.yaml parsing (M14/M16 own config UX) — router config is Go-level Registration values.
- `internal/core` edits: exactly the T1 shapes. `sdk/`: alias lines for new/completed types only.
- classify: implemented on adapters, routable in principle, but **no caller** — FR-IL-10 placeholder.

## Acceptance criteria (IMP §27.M4 + PRD)
- AC-1: every branch of the §17 decision tree unit-tested (table-driven, fake adapters).
- AC-2: FR-IL-08 rejection-not-truncation enforced + tested (exact boundary: at budget passes, one over rejects).
- AC-3: FR-IL-01 NullAdapter behaviors exact (incl. "The record is silent." verbatim, IsAvailable=false).
- AC-4: IntelligencePort contract suite exists and NullAdapter passes it.
- AC-5: `make build test lint` green; no new module deps (stdlib only).
