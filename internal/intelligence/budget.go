package intelligence

import "fmt"

// ContextBudgetExceededError is returned when the estimated token count for the
// assembled context exceeds the declared budget (FR-IL-08). The context is
// rejected; it is never truncated.
type ContextBudgetExceededError struct {
	// Budget is the maximum allowed token count (from IntelReq.ContextBudget).
	Budget int
	// Estimated is the conservative token estimate for the context string.
	Estimated int
}

func (e ContextBudgetExceededError) Error() string {
	return fmt.Sprintf("context budget exceeded: estimated %d tokens, budget %d", e.Estimated, e.Budget)
}

// estimateTokens returns a conservative token estimate for s using the formula
// ceil(len(s)/4) (EDR-008). This is an enforcement-only estimator; no tokenizer
// exists in V1. Provider-actual token counts arrive with FR-IL-09 at M16 and
// are never used for billing here.
func estimateTokens(s string) int {
	return (len(s) + 3) / 4
}

// enforceBudget rejects contextStr if its estimated token count exceeds budget.
// A budget of 0 or less means no budget is declared and enforcement is skipped.
//
// FR-IL-08: over-budget contexts are rejected, never truncated.
func enforceBudget(contextStr string, budget int) error {
	if budget <= 0 {
		return nil
	}
	est := estimateTokens(contextStr)
	if est > budget {
		return ContextBudgetExceededError{Budget: budget, Estimated: est}
	}
	return nil
}
