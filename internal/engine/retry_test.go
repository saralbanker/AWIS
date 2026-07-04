package engine

// Retry mechanics (ADJ-6, EDR-011 §3): backoff math (immediate/linear/
// exponential, caps, ADJ-6 defaults), retryable_errors filtering both ways, and
// the retry-then-success E2E through the real engine (attempt payload fields +
// exact event sequence + forward ≡ rebuild).

import (
	"context"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

func TestBackoffDelay_Immediate(t *testing.T) {
	p := core.RetryPolicy{Backoff: "immediate"}
	for attempt := 1; attempt <= 5; attempt++ {
		if got := backoffDelay(p, attempt); got != 0 {
			t.Fatalf("immediate attempt %d = %v, want 0", attempt, got)
		}
	}
}

func TestBackoffDelay_Linear(t *testing.T) {
	p := core.RetryPolicy{Backoff: "linear", InitialDelay: "2s", MaxDelay: "1m"}
	want := map[int]time.Duration{1: 2 * time.Second, 2: 4 * time.Second, 3: 6 * time.Second}
	for attempt, w := range want {
		if got := backoffDelay(p, attempt); got != w {
			t.Fatalf("linear attempt %d = %v, want %v", attempt, got, w)
		}
	}
}

func TestBackoffDelay_Exponential(t *testing.T) {
	p := core.RetryPolicy{Backoff: "exponential", InitialDelay: "1s", MaxDelay: "1m"}
	want := map[int]time.Duration{1: 1 * time.Second, 2: 2 * time.Second, 3: 4 * time.Second, 4: 8 * time.Second}
	for attempt, w := range want {
		if got := backoffDelay(p, attempt); got != w {
			t.Fatalf("exponential attempt %d = %v, want %v", attempt, got, w)
		}
	}
}

func TestBackoffDelay_CapAtMax(t *testing.T) {
	p := core.RetryPolicy{Backoff: "exponential", InitialDelay: "10s", MaxDelay: "30s"}
	// 10s, 20s, 40s→cap 30s, 80s→cap 30s.
	if got := backoffDelay(p, 3); got != 30*time.Second {
		t.Fatalf("exponential attempt 3 = %v, want capped 30s", got)
	}
	if got := backoffDelay(p, 4); got != 30*time.Second {
		t.Fatalf("exponential attempt 4 = %v, want capped 30s", got)
	}
}

func TestBackoffDelay_ADJ6Defaults(t *testing.T) {
	// Absent InitialDelay/MaxDelay ⇒ 1s / 30s (ADJ-6).
	p := core.RetryPolicy{Backoff: "linear"}
	if got := backoffDelay(p, 1); got != 1*time.Second {
		t.Fatalf("default-initial linear attempt 1 = %v, want 1s", got)
	}
	// linear default: 1s×40 = 40s → capped at default max 30s.
	if got := backoffDelay(p, 40); got != 30*time.Second {
		t.Fatalf("default-max cap = %v, want 30s", got)
	}
}

func TestRetryEligible_RetryableErrorsFilter(t *testing.T) {
	// Filtered-in: code in the list ⇒ eligible.
	p := &core.RetryPolicy{Attempts: 3, Backoff: "immediate", RetryableErrors: []string{"handler_error"}}
	if !retryEligible(p, 1, "handler_error") {
		t.Fatalf("handler_error must be retryable when listed")
	}
	// Filtered-out: code NOT in the list ⇒ not eligible.
	if retryEligible(p, 1, "timeout") {
		t.Fatalf("timeout must NOT be retryable when not listed")
	}
	// Empty list ⇒ all codes retryable.
	pAll := &core.RetryPolicy{Attempts: 3, Backoff: "immediate"}
	if !retryEligible(pAll, 1, "anything") {
		t.Fatalf("empty retryable_errors must retry all codes")
	}
	// Attempts exhausted ⇒ not eligible even for a listed code.
	if retryEligible(p, 3, "handler_error") {
		t.Fatalf("attempt at/over Attempts must not be eligible")
	}
	// No policy ⇒ single attempt (never eligible).
	if retryEligible(nil, 1, "handler_error") {
		t.Fatalf("nil policy must be single-attempt")
	}
}

// TestRetryThenSuccess drives a one-step workflow whose handler fails attempt 1
// and succeeds attempt 2, asserting the EXACT event sequence with attempt fields
// and that the retry is not due until the backoff has elapsed.
func TestRetryThenSuccess(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.retry", Version: "1.0.0", Namespace: "t", Name: "retry",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{{
			ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha",
			Retry: &core.RetryPolicy{Attempts: 2, Backoff: "linear", InitialDelay: "1s", MaxDelay: "30s"},
		}},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	ha := &flakyHandler{id: "ha", outputs: map[string]any{"r": "ok"}, err: errBoom, failUntil: 1}

	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, ha)
	ctx := context.Background()

	iid, err := e.Submit(ctx, "t.retry", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick 1: activate + dispatch a (fails attempt 1) → StepFailed{retrying:true}.
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 1: %v", err)
	}
	// Tick 2 WITHOUT advancing: the retry is not yet due (backoff 1s) ⇒ no progress.
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 2: %v", err)
	}
	pairs, _ := eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("retry must not re-dispatch before backoff elapses; got %d StepStarted, want 1", n)
	}

	// Advance past the 1s backoff; tick again ⇒ attempt 2 runs and succeeds.
	clk.advance(1 * time.Second)
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"},
		{core.EventTypeStepFailed, "a"},
		{core.EventTypeStepStarted, "a"},
		{core.EventTypeStepCompleted, "a"},
		{core.EventTypeWorkflowCompleted, ""},
	}
	assertPairs(t, pairs, want)

	// attempt fields: first StepStarted attempt 1; StepFailed{retrying:true, attempt 1};
	// second StepStarted attempt 2; StepCompleted attempt 2.
	starts := allEvents(evs, core.EventTypeStepStarted, "a")
	if a := attemptOf(t, starts[0]); a != 1 {
		t.Fatalf("first StepStarted attempt = %d, want 1", a)
	}
	if a := attemptOf(t, starts[1]); a != 2 {
		t.Fatalf("second StepStarted attempt = %d, want 2", a)
	}
	sf := findEvent(t, evs, core.EventTypeStepFailed, "a")
	var sfp struct {
		Attempt  int  `json:"attempt"`
		Retrying bool `json:"retrying"`
	}
	decodePayload(t, sf, &sfp)
	if !sfp.Retrying || sfp.Attempt != 1 {
		t.Fatalf("StepFailed = {attempt:%d, retrying:%v}, want {1, true}", sfp.Attempt, sfp.Retrying)
	}
	sc := findEvent(t, evs, core.EventTypeStepCompleted, "a")
	if a := attemptOf(t, sc); a != 2 {
		t.Fatalf("StepCompleted attempt = %d, want 2", a)
	}

	assertProjectionEquivalence(t, s, iid)
}

// TestRetryExhaustNoFallback: a step with attempts=2 whose handler always fails
// exhausts retries and, with no fallback/on_error, fails the workflow.
func TestRetryExhaustNoFallback(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.rexh", Version: "1.0.0", Namespace: "t", Name: "rexh",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{{
			ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha",
			Retry: &core.RetryPolicy{Attempts: 2, Backoff: "immediate"},
		}},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", err: errBoom}

	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, ha)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.rexh", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusFailed {
		t.Fatalf("status = %q, want failed", inst.Status)
	}
	pairs, _ := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, // attempt 1
		{core.EventTypeStepFailed, "a"},  // retrying:true
		{core.EventTypeStepStarted, "a"}, // attempt 2 (immediate backoff, due same/next tick)
		{core.EventTypeStepFailed, "a"},  // retrying:false (exhausted)
		{core.EventTypeWorkflowFailed, ""},
	}
	assertPairs(t, pairs, want)
	assertProjectionEquivalence(t, s, iid)
}

// countPairs counts (type, step) occurrences in a pair stream.
func countPairs(pairs []pair, et core.EventType, step string) int {
	n := 0
	for _, p := range pairs {
		if p.Type == et && p.StepID == step {
			n++
		}
	}
	return n
}

// allEvents returns every event matching (type, step) in order.
func allEvents(evs []core.ExecutionEvent, et core.EventType, step string) []core.ExecutionEvent {
	var out []core.ExecutionEvent
	for _, ev := range evs {
		if ev.EventType == et && ev.StepID == step {
			out = append(out, ev)
		}
	}
	return out
}

// attemptOf decodes the attempt field of a step event payload.
func attemptOf(t *testing.T, ev core.ExecutionEvent) int {
	t.Helper()
	var p struct {
		Attempt int `json:"attempt"`
	}
	decodePayload(t, ev, &p)
	return p.Attempt
}
