package engine

// LA-C7 LOW-2 and LOW-3 (independent review round 4 of LA-C6 R3-M1). TEST-ONLY file.
//
// LOW-2 pins failure.go routeTerminalFailure's route (3): when maybeCompensate fails after the
// route's WorkflowFailed is durable, the function returns (terminal=true, err). The settle loop stops
// on terminal (LA-C5 N2), so no sibling is settled after a committed route (3) decision. The
// `return true, err` after maybeCompensate is the line pinned here. Changing it to false must make
// TestLA_C7_LOW2_RouteThreeMaybeCompensateErrorIsTerminal fail (the mutation check of the card).
//
// LOW-3 pins staticRouteThree (hydrate.go), the single static route-(3) predicate.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// TestLA_C7_LOW2_RouteThreeMaybeCompensateErrorIsTerminal calls routeTerminalFailure directly for
// b1 (no fallback, no on_error: route (3)), with the compensation's first append failing before its
// write. WorkflowFailed is durable when maybeCompensate fails, so the call must report terminal.
func TestLA_C7_LOW2_RouteThreeMaybeCompensateErrorIsTerminal(t *testing.T) {
	w := signalStep("w", "go")
	def := c6SiblingDef(w)
	s, fs := laSetup(t, def)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	rootH := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1H := &stepHandler{id: "la_b1", err: errLAHandler}
	var u c5Undo
	handlers := []core.StepHandler{rootH, b1H, u.handler("undo-root"), u.handler("undo-w"), u.handler("undo-b1")}
	e := laEngine(fs, clk, ids, 2, handlers...)
	iid := laSubmit(t, e, def.ID)
	ctx := context.Background()
	if err := e.Tick(ctx); err != nil { // root completes: the plan has a completed step to undo
		t.Fatalf("tick 1: %v", err)
	}
	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeWorkflowCompensating})
	inst := laInstance(t, s, iid)
	dv, err := e.defViewFor(ctx, inst)
	if err != nil {
		t.Fatalf("defViewFor: %v", err)
	}
	stepErr := core.StepError{Code: "handler_error", Message: "la: handler failure"}
	terminal, rerr := e.routeTerminalFailure(ctx, dv, inst, dv.steps["b1"], 1, stepErr)
	if !errors.Is(rerr, errLAInjected) {
		t.Fatalf("route (3): the injected compensation failure did not surface: %v", rerr)
	}
	if !terminal {
		t.Errorf("route (3): WorkflowFailed is durable and its compensation failed: terminal=false; want true, so the settle loop stops")
	}
	if n := c5Count(laEvents(t, s, iid), core.EventTypeWorkflowFailed); n != 1 {
		t.Errorf("WorkflowFailed count=%d; want 1 (the route's own decision)", n)
	}
}

// TestLA_C7_LOW3_StaticRouteThree pins the truth table of staticRouteThree: route (3) iff the step
// has no Fallback and no on_error transition, conditional or not. The static rule never evaluates a
// condition, so the conditional on_error edge is route (2) here.
func TestLA_C7_LOW3_StaticRouteThree(t *testing.T) {
	fb := nativeStep("fb", "h")
	fb.Fallback = "t1"
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.c7.static", Version: "1.0.0", Namespace: "la", Name: "c7static",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("plain", "h"),  // no Fallback, no on_error: route (3)
			fb,                        // Fallback: route (1)
			nativeStep("uncond", "h"), // unconditional on_error: route (2)
			nativeStep("cond", "h"),   // conditional on_error: route (2) (static; never evaluated)
			nativeStep("t1", "h"),
			nativeStep("t2", "h"),
		},
		Transitions: []core.Transition{
			{From: "plain", To: "t1"},
			{From: "uncond", To: "t1", OnError: true},
			{From: "cond", To: "t2", OnError: true, Condition: "steps.plain.outputs.flag == 'yes'"},
		},
		InitialStep: "plain", FinalSteps: []string{"t1", "t2"}, Metadata: map[string]any{},
	}
	dv, err := buildDefView(def)
	if err != nil {
		t.Fatalf("buildDefView: %v", err)
	}
	want := map[string]bool{"plain": true, "fb": false, "uncond": false, "cond": false}
	for id, wantRouteThree := range want {
		if got := staticRouteThree(dv, dv.steps[id]); got != wantRouteThree {
			t.Errorf("staticRouteThree(%s)=%v; want %v", id, got, wantRouteThree)
		}
	}
}
