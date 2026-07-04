package engine

// Compensation E2E (Blueprint §8 L532–546, EDR-011 §6): reverse-order undo,
// never-completed plan entries skipped, own-retry undo, and compensation-failure
// (WorkflowCompensationFailed + compensation_failed status + slog.Error).

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
)

// compDef builds a linear a→b→c→d workflow (d fails) with a compensation plan
// over [a, b, c, x]; x never completes and must be skipped.
func compDef() core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.comp", Version: "1.0.0", Namespace: "t", Name: "comp",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"},
			{ID: "b", Name: "b", Type: core.StepTypeNative, Handler: "hb"},
			{ID: "c", Name: "c", Type: core.StepTypeNative, Handler: "hc"},
			{ID: "d", Name: "d", Type: core.StepTypeNative, Handler: "hd"},
		},
		Transitions: []core.Transition{
			{From: "a", To: "b"}, {From: "b", To: "c"}, {From: "c", To: "d"},
		},
		Compensation: &core.CompensationPlan{Steps: []core.CompensationStep{
			{StepID: "a", UndoHandler: "undo-a"},
			{StepID: "b", UndoHandler: "undo-b"},
			{StepID: "c", UndoHandler: "undo-c"},
			{StepID: "x", UndoHandler: "undo-x"}, // x never completes ⇒ skipped
		}},
		InitialStep: "a", FinalSteps: []string{"d"}, Metadata: map[string]any{},
	}
}

func TestCompensation_ReverseOrderSkipsNeverCompleted(t *testing.T) {
	def := compDef()
	var order []string
	var mu sync.Mutex
	handlers := []core.StepHandler{
		&stepHandler{id: "ha", outputs: map[string]any{"r": "a"}},
		&stepHandler{id: "hb", outputs: map[string]any{"r": "b"}},
		&stepHandler{id: "hc", outputs: map[string]any{"r": "c"}},
		&stepHandler{id: "hd", err: errBoom},
		&recordingHandler{id: "undo-a", order: &order, mu: &mu},
		&recordingHandler{id: "undo-b", order: &order, mu: &mu},
		&recordingHandler{id: "undo-c", order: &order, mu: &mu},
		&recordingHandler{id: "undo-x", order: &order, mu: &mu}, // must never run
	}

	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, handlers...)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.comp", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompensated {
		t.Fatalf("status = %q, want compensated", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepCompleted, "a"},
		{core.EventTypeStepStarted, "b"}, {core.EventTypeStepCompleted, "b"},
		{core.EventTypeStepStarted, "c"}, {core.EventTypeStepCompleted, "c"},
		{core.EventTypeStepStarted, "d"}, {core.EventTypeStepFailed, "d"},
		{core.EventTypeWorkflowFailed, ""},
		{core.EventTypeWorkflowCompensating, ""},
		{core.EventTypeWorkflowCompensated, ""},
	}
	assertPairs(t, pairs, want)

	// Undo call order is REVERSE of the plan, skipping the never-completed x.
	mu.Lock()
	got := append([]string{}, order...)
	mu.Unlock()
	wantOrder := []string{"undo-c", "undo-b", "undo-a"}
	if strings.Join(got, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("undo order = %v, want %v (reverse, x skipped)", got, wantOrder)
	}

	// WorkflowCompensating.from_step = the failed step (d).
	wc := findEvent(t, evs, core.EventTypeWorkflowCompensating, "")
	var wcp workflowCompensatingPayload
	decodePayload(t, wc, &wcp)
	if wcp.FromStep != "d" {
		t.Fatalf("WorkflowCompensating.from_step = %q, want d", wcp.FromStep)
	}

	assertProjectionEquivalence(t, s, iid)
}

// TestCompensation_UndoRetrySucceeds: an undo whose handler fails once then
// succeeds, under CompensationStep.Retry attempts=2, completes compensation.
func TestCompensation_UndoRetrySucceeds(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.compr", Version: "1.0.0", Namespace: "t", Name: "compr",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"},
			{ID: "d", Name: "d", Type: core.StepTypeNative, Handler: "hd"},
		},
		Transitions: []core.Transition{{From: "a", To: "d"}},
		Compensation: &core.CompensationPlan{Steps: []core.CompensationStep{
			{StepID: "a", UndoHandler: "undo-a",
				Retry: &core.RetryPolicy{Attempts: 2, Backoff: "immediate"}},
		}},
		InitialStep: "a", FinalSteps: []string{"d"}, Metadata: map[string]any{},
	}
	undo := &flakyHandler{id: "undo-a", outputs: map[string]any{}, err: errBoom, failUntil: 1}
	handlers := []core.StepHandler{
		&stepHandler{id: "ha", outputs: map[string]any{"r": "a"}},
		&stepHandler{id: "hd", err: errBoom},
		undo,
	}
	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now, handlers...)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.compr", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompensated {
		t.Fatalf("status = %q, want compensated", inst.Status)
	}
	if undo.calls != 2 {
		t.Fatalf("undo-a executed %d times, want 2 (fail then retry-succeed)", undo.calls)
	}
	assertProjectionEquivalence(t, s, iid)
}

// TestCompensation_UndoExhaustsFails: an undo that always fails (retry exhausts)
// yields WorkflowCompensationFailed + status compensation_failed + slog.Error.
func TestCompensation_UndoExhaustsFails(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.compf", Version: "1.0.0", Namespace: "t", Name: "compf",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"},
			{ID: "d", Name: "d", Type: core.StepTypeNative, Handler: "hd"},
		},
		Transitions: []core.Transition{{From: "a", To: "d"}},
		Compensation: &core.CompensationPlan{Steps: []core.CompensationStep{
			{StepID: "a", UndoHandler: "undo-a"}, // no retry ⇒ single attempt
		}},
		InitialStep: "a", FinalSteps: []string{"d"}, Metadata: map[string]any{},
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s := openStorage(t)
	if err := s.RegisterWorkflow(context.Background(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	nr := native.New()
	for _, h := range []core.StepHandler{
		&stepHandler{id: "ha", outputs: map[string]any{"r": "a"}},
		&stepHandler{id: "hd", err: errBoom},
		&stepHandler{id: "undo-a", err: errBoom},
	} {
		nr.Register(h)
	}
	clk := newManualClock(engineStart)
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: 1, Clock: clk.now}, logger)

	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.compf", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompensationFailed {
		t.Fatalf("status = %q, want compensation_failed", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	last := pairs[len(pairs)-1]
	if last.Type != core.EventTypeWorkflowCompensationFailed {
		t.Fatalf("terminal event = %v, want WorkflowCompensationFailed", last)
	}
	cf := findEvent(t, evs, core.EventTypeWorkflowCompensationFailed, "")
	var cfp workflowCompensationFailedPayload
	decodePayload(t, cf, &cfp)
	if cfp.StepID != "a" {
		t.Fatalf("WorkflowCompensationFailed.step_id = %q, want a", cfp.StepID)
	}
	if !strings.Contains(buf.String(), "compensation undo exhausted") {
		t.Fatalf("expected an slog.Error for the exhausted undo; log=%s", buf.String())
	}

	assertProjectionEquivalence(t, s, iid)
}
