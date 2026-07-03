# EDR-008 — Context-Budget Token Estimator

**Status:** Reversible — pre-G2 (implementation decision; flagged for G2 review)
**Authored:** M04 (2026-07-03)
**Coordinates:** Blueprint §13 "Context Budget Enforcement" · PRD FR-IL-08 · `internal/intelligence/budget.go`
**Downstream obligation:** M16 records provider-actual token counts (FR-IL-09); the estimator remains enforcement-only and is never used for billing or reporting.

---

## Decision

`IntelReq.ContextBudget` is declared in tokens (Blueprint §13), but V1 ships no
tokenizer and no frozen text prescribes a measurement method. Enforcement uses
the conservative estimator:

```
estimateTokens(s) = ceil(len(bytes(s)) / 4)
```

- Applied **before** any routing or provider call; an over-budget context
  returns `ContextBudgetExceededError{Budget, Estimated}` — rejected, never
  truncated (FR-IL-08 verbatim: "rejected, not silently truncated").
- `ContextBudget <= 0` means no budget declared → enforcement skipped.
- Draft budgets `DraftRequest.Context` (the assembled context, Blueprint §13);
  Synthesize budgets `SynthesisRequest.Query`. Entries are structured data,
  not assembled context, and are not counted in V1.

## Why not a CONTRA

The gap is a missing *mechanism*, not a conflict in frozen text: no format,
schema, or public interface is touched, and the ~4-bytes/token heuristic is the
industry-standard conservative bound for English text. The choice is reversible
(swap the estimator, behavior only becomes more/less strict) and is flagged for
G2 review alongside the engine's dispatch semantics.
