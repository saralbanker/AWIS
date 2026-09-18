package engine

// TestStall_DeadEndBranchFailsInsteadOfWedging (SEC-08 / RC-6): a workflow
// where the branch actually taken at runtime terminates on a step with no
// outgoing transition that is not itself a final_steps entry, while the
// sibling branch that WOULD reach the final step never fires (its guard
// condition is false). Before the tick.go fix, this left the instance wedged
// in status "running" with CurrentSteps=[] forever: activatableFor and
// completedFinalOutputs are both empty, and the old code just `return nil`ed
// every tick. validate.Validate() does not catch this shape (it only checks
// that final_steps entries are defined steps and that every defined step is
// reachable from initial_step — it never requires that a step with no
// outgoing transitions be listed in final_steps), and neither
// sdk.Runtime.RegisterWorkflow nor the `awis start` discovery path invoke the
// validator, so this is reachable in production via ordinary YAML authoring,
// not just a synthetic construction.

import (
	"context"
	"testing"

	"github.com/awis/awis/internal/core"
)

func TestStall_DeadEndBranchFailsInsteadOfWedging(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.stall", Version: "1.0.0", Namespace: "t", Name: "stall",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "ha"), // fans out
			nativeStep("b", "hb"), // dead end: no outgoing transition, not in final_steps
			nativeStep("c", "hc"), // the intended final step — only reached if a's flag is "yes"
		},
		Transitions: []core.Transition{
			{From: "a", To: "b"}, // unconditional — always fires
			{From: "a", To: "c", Condition: "steps.a.outputs.flag == 'yes'"}, // never fires here
		},
		InitialStep: "a", FinalSteps: []string{"c"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", outputs: map[string]any{"flag": "no"}}
	hb := &stepHandler{id: "hb", outputs: map[string]any{"done": true}}
	hc := &stepHandler{id: "hc", outputs: map[string]any{"done": true}}

	e, s, _ := buildEngine(t, def, 1, ha, hb, hc)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.stall", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	final := runToTerminal(t, e, s, iid)
	if final.Status != core.InstanceStatusFailed {
		t.Fatalf("status = %q, want failed (dead-end branch must not wedge the instance in running)", final.Status)
	}
	if len(final.CurrentSteps) != 0 {
		t.Fatalf("CurrentSteps = %v, want empty at terminal", final.CurrentSteps)
	}

	evs, err := s.ReadEvents(ctx, iid, 0)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	last := evs[len(evs)-1]
	if last.EventType != core.EventTypeWorkflowFailed {
		t.Fatalf("last event = %s, want WorkflowFailed", last.EventType)
	}
	var p struct {
		StepID string `json:"step_id"`
		Error  struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodePayload(t, last, &p)
	if p.Error.Code != "stalled" {
		t.Fatalf("WorkflowFailed.error.code = %q, want %q", p.Error.Code, "stalled")
	}

	assertProjectionEquivalence(t, s, iid)
}
