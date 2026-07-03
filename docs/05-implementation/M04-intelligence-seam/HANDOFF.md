# M04 → M06/M16 Handoff
**Status: IN EXECUTION** — Actuals filled at completion; merge commit at merge.

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

## Actuals (filled at completion)
- Merge commit: _
