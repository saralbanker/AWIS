# M04 → M06/M16 Handoff
**Status: EXECUTED + VERIFIED (M04-V1 12/12 at `b8989a9`)** — merge commit filled at merge.

## Guaranteed outputs (contract)
- `internal/intelligence`: CapabilityRouter (§17 tree, all branches tested), Dispatcher (budget → route → call),
  typed errors `ErrCapabilityUnavailable` / `ErrContextBudgetExceeded`, FallbackSignal outcome.
- `internal/intelligence/adapters/null`: NullAdapter per Blueprint §14 verbatim (FR-IL-01).
- `internal/intelligence/porttest`: IntelligencePort contract suite (M16 reuse artifact).
- Completed core shapes: DraftResponse/SynthesisResponse/Capability/Example + Usage; sdk aliases.

## What M06 may assume
- Router never activates fallbacks itself — it reports FallbackSignal; engine owns activation.
- Budget rejection happens before any provider call; a dispatched request was within budget (estimator, EDR-008).
- Usage is present on every successful response; NullAdapter reports zeros.

## What M16 may assume
- porttest is the acceptance bar; Registration traits classify the new adapter (cloud, ranks).

## Obligations owed BY M06/M16 (not delivered here)
- **Null-last-in-chain** (FR-IL-01 "always registered as the last fallback") is a runtime-assembly
  rule: whoever assembles the production Registration list + fallback_chain (M06 runtime init /
  M16 config wiring) must place null last and should add a construction-time chain validation
  (M04-V1 advisory). M04 delivers the complementary half: null is never *selected* (IsAvailable=false, tested).
- **`required` field on IntelReq**: dispatcher takes `required bool` explicitly; the field lands on
  IntelReq at M06/M08 when step-config wiring arrives (TRACEABILITY decision note).
- EDR-008 estimator is flagged for G2 review with engine dispatch semantics.

## Actuals (execution 2026-07-03)
- Merge commit: _ (pending; branch tip `b8989a9`, M04-V1 12/12 PASS clean-clone)
- Cards: M04-C1 (Sonnet: shapes+NullAdapter+porttest, 19 tests), M04-C2/C2r (Sonnet: router+budget+dispatcher, 17 branch tests; C2 session-limit cutoff after router.go+budget.go — both kept, C2r finished)
- CE review: C1/C2 diffs reviewed in full; no semantic findings; no fix cycle needed (first milestone with zero verification failures)
- Deviations: standing named-agent dispatch deviation; C2 session-limit continuation
