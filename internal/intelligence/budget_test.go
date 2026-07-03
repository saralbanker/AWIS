package intelligence

import (
	"errors"
	"strings"
	"testing"

	"github.com/awis/awis/internal/core"
)

// TestBudgetBoundary verifies the ceil(len(s)/4) estimator and enforcement
// boundaries (EDR-008, FR-IL-08):
//   - A 40-byte context has estimate 10, which equals budget 10 → passes.
//   - A 41-byte context has estimate 11, which exceeds budget 10 → rejected with
//     ContextBudgetExceededError{Budget:10, Estimated:11}.
//   - A budget of 0 means no limit; any size passes.
func TestBudgetBoundary(t *testing.T) {
	cases := []struct {
		name       string
		ctx        string
		budget     int
		wantErr    bool
		wantBudget int
		wantEst    int
	}{
		{
			name:    "40-byte-at-limit-passes",
			ctx:     strings.Repeat("x", 40), // ceil(40/4)=10 == budget 10
			budget:  10,
			wantErr: false,
		},
		{
			name:       "41-byte-over-limit-rejected",
			ctx:        strings.Repeat("x", 41), // ceil(41/4)=11 > budget 10
			budget:     10,
			wantErr:    true,
			wantBudget: 10,
			wantEst:    11,
		},
		{
			name:    "budget-zero-no-enforcement",
			ctx:     strings.Repeat("x", 1000),
			budget:  0,
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := enforceBudget(tc.ctx, tc.budget)
			if tc.wantErr {
				if err == nil {
					t.Fatal("enforceBudget: want error, got nil")
				}
				var budgetErr ContextBudgetExceededError
				if !errors.As(err, &budgetErr) {
					t.Fatalf("enforceBudget: want ContextBudgetExceededError, got %T: %v", err, err)
				}
				if budgetErr.Budget != tc.wantBudget {
					t.Errorf("ContextBudgetExceededError.Budget: want %d, got %d",
						tc.wantBudget, budgetErr.Budget)
				}
				if budgetErr.Estimated != tc.wantEst {
					t.Errorf("ContextBudgetExceededError.Estimated: want %d, got %d",
						tc.wantEst, budgetErr.Estimated)
				}
			} else {
				if err != nil {
					t.Errorf("enforceBudget: want nil, got %v", err)
				}
			}
		})
	}
}

// TestDispatcherBudgetBeforeCall asserts that an over-budget Draft context is
// rejected before any routing or adapter call. The fakeAdapter's draftCalls
// counter must remain 0.
func TestDispatcherBudgetBeforeCall(t *testing.T) {
	a := &fakeAdapter{
		name:      "a",
		availFn:   func() bool { return true },
		caps:      []string{"draft"},
		draftResp: core.DraftResponse{Output: map[string]any{}},
	}
	router, err := NewRouter(
		[]Registration{{Adapter: a, Locality: LocalityCloud}},
		[]string{"a"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	d := NewDispatcher(router)

	// 41 bytes → estimate 11 > budget 10 → must be rejected before any adapter call.
	overBudgetCtx := strings.Repeat("x", 41)
	req := core.IntelReq{Capability: "draft", ContextBudget: 10}
	dr := core.DraftRequest{Context: overBudgetCtx}

	_, err = d.Draft(bg(), req, true, dr)
	if err == nil {
		t.Fatal("Draft: want error for over-budget context, got nil")
	}
	var budgetErr ContextBudgetExceededError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("Draft: want ContextBudgetExceededError, got %T: %v", err, err)
	}
	if a.draftCalls != 0 {
		t.Errorf("draftCalls: want 0 (budget enforced before adapter), got %d", a.draftCalls)
	}
}

// TestDispatcherSynthesizeBudgetOnQuery asserts that budget enforcement for
// Synthesize applies to sr.Query only. Large Entries do not trigger rejection
// (Blueprint §13: Entries are structured data, not assembled context).
func TestDispatcherSynthesizeBudgetOnQuery(t *testing.T) {
	a := &fakeAdapter{
		name:      "a",
		availFn:   func() bool { return true },
		caps:      []string{"synthesize"},
		synthResp: core.SynthesisResponse{Text: "ok"},
	}
	router, err := NewRouter(
		[]Registration{{Adapter: a, Locality: LocalityCloud}},
		[]string{"a"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	d := NewDispatcher(router)

	req := core.IntelReq{Capability: "synthesize", ContextBudget: 10}

	// Over-budget Query (estimate 11 > budget 10); large Entries must not matter.
	sr := core.SynthesisRequest{
		Query:   strings.Repeat("y", 41),
		Entries: []any{strings.Repeat("z", 10_000)}, // not budget-counted in V1
	}
	_, err = d.Synthesize(bg(), req, true, sr)
	if err == nil {
		t.Fatal("Synthesize: want error for over-budget query, got nil")
	}
	var budgetErr ContextBudgetExceededError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("Synthesize: want ContextBudgetExceededError, got %T: %v", err, err)
	}

	// Within-budget Query with the same large Entries must succeed.
	sr.Query = strings.Repeat("y", 40) // estimate 10 == budget 10 → passes
	resp, err := d.Synthesize(bg(), req, true, sr)
	if err != nil {
		t.Fatalf("Synthesize: want success for at-limit query, got %v", err)
	}
	if resp.Text != "ok" {
		t.Errorf("Synthesize.Text: want %q, got %q", "ok", resp.Text)
	}
}
