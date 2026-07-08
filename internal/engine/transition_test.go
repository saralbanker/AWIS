package engine

// Join-gate + fan-out + InitialStep activation tests (EDR-011 §1). These drive
// the stateless activation derivation directly (buildDefView + activatableSteps)
// so the state space is enumerated in the suite, not in prose (IMP §28 rule 8).

import (
	"testing"

	"github.com/awis/awis/internal/core"
)

// joinDef: two distinct froms (p, q) converge on target t; s is the initial
// step that fans out to p and q.
func joinDef(pCond, qCond core.Condition) core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "t.join",
		Version:       "1.0.0",
		Namespace:     "t",
		Name:          "join",
		Steps: []core.Step{
			{ID: "s", Type: core.StepTypeNative, Handler: "h"},
			{ID: "p", Type: core.StepTypeNative, Handler: "h"},
			{ID: "q", Type: core.StepTypeNative, Handler: "h"},
			{ID: "t", Type: core.StepTypeNative, Handler: "h"},
		},
		Transitions: []core.Transition{
			{From: "s", To: "p"},
			{From: "s", To: "q"},
			{From: "p", To: "t", Condition: pCond},
			{From: "q", To: "t", Condition: qCond},
		},
		InitialStep: "s",
		FinalSteps:  []string{"t"},
		Metadata:    map[string]any{},
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// instWith builds a projection with the named completed steps (each carrying an
// outputs map) and running steps.
func instWith(completed map[string]map[string]any, running ...string) core.WorkflowInstance {
	vars := map[string]any{"inputs": map[string]any{}}
	for id, out := range completed {
		vars[id] = out
	}
	return core.WorkflowInstance{
		InstanceID:   "inst-1",
		Status:       core.InstanceStatusRunning,
		Variables:    vars,
		CurrentSteps: append([]string{}, running...),
	}
}

func TestActivatable_JoinGate_BothCompletedActivatesOnce(t *testing.T) {
	dv, err := buildDefView(joinDef("", ""))
	if err != nil {
		t.Fatalf("buildDefView: %v", err)
	}
	// Both p and q completed; both inbound edges unconditional ⇒ target fires.
	inst := instWith(map[string]map[string]any{
		"p": {"ok": true},
		"q": {"ok": true},
	})
	acts := activatableSteps(dv, inst)
	if !contains(acts, "t") {
		t.Fatalf("target t must activate when both froms completed; got %v", acts)
	}
	// Two firing edges into one target ⇒ it appears exactly once.
	n := 0
	for _, a := range acts {
		if a == "t" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("target t must be listed once (at-most-once claim dedups), got %d", n)
	}
}

func TestActivatable_JoinGate_OneFromIncompleteStalls(t *testing.T) {
	dv, err := buildDefView(joinDef("", ""))
	if err != nil {
		t.Fatalf("buildDefView: %v", err)
	}
	// Only p completed; q not ⇒ join gate (a) fails.
	inst := instWith(map[string]map[string]any{"p": {"ok": true}})
	acts := activatableSteps(dv, inst)
	if contains(acts, "t") {
		t.Fatalf("target t must NOT activate while a distinct from is incomplete; got %v", acts)
	}
}

func TestActivatable_JoinGate_ConditionalSkipStall(t *testing.T) {
	// Both froms completed but every inbound condition is false ⇒ gate (b) fails.
	dv, err := buildDefView(joinDef(
		"steps.p.outputs.flag == 'yes'",
		"steps.q.outputs.flag == 'yes'",
	))
	if err != nil {
		t.Fatalf("buildDefView: %v", err)
	}
	inst := instWith(map[string]map[string]any{
		"p": {"flag": "no"},
		"q": {"flag": "no"},
	})
	acts := activatableSteps(dv, inst)
	if contains(acts, "t") {
		t.Fatalf("target t must stall when all inbound conditions are false; got %v", acts)
	}
}

func TestActivatable_JoinGate_OneConditionTrueActivates(t *testing.T) {
	dv, err := buildDefView(joinDef(
		"steps.p.outputs.flag == 'yes'",
		"steps.q.outputs.flag == 'yes'",
	))
	if err != nil {
		t.Fatalf("buildDefView: %v", err)
	}
	// Both froms completed; p's condition true, q's false ⇒ ≥1 edge fires.
	inst := instWith(map[string]map[string]any{
		"p": {"flag": "yes"},
		"q": {"flag": "no"},
	})
	acts := activatableSteps(dv, inst)
	if !contains(acts, "t") {
		t.Fatalf("target t must activate when ≥1 inbound condition is true; got %v", acts)
	}
}

func TestActivatable_FanOut_BothUnconditionalTargets(t *testing.T) {
	// One completed step (s) with two unconditional outbound edges ⇒ both
	// targets activatable in the same derivation.
	dv, err := buildDefView(joinDef("", ""))
	if err != nil {
		t.Fatalf("buildDefView: %v", err)
	}
	inst := instWith(map[string]map[string]any{"s": {"ok": true}})
	acts := activatableSteps(dv, inst)
	if !contains(acts, "p") || !contains(acts, "q") {
		t.Fatalf("both fan-out targets p and q must activate after s; got %v", acts)
	}
}

func TestActivatable_InitialStep_SelfActivatesOnlyAtStart(t *testing.T) {
	dv, err := buildDefView(joinDef("", ""))
	if err != nil {
		t.Fatalf("buildDefView: %v", err)
	}

	// At the very start (nothing completed/running) the InitialStep self-activates.
	start := core.WorkflowInstance{
		InstanceID:   "inst-1",
		Status:       core.InstanceStatusRunning,
		Variables:    map[string]any{"inputs": map[string]any{}},
		CurrentSteps: []string{},
	}
	if acts := activatableSteps(dv, start); !contains(acts, "s") {
		t.Fatalf("InitialStep s must self-activate at the very start; got %v", acts)
	}

	// Once something has completed, the InitialStep never re-activates.
	afterCompleted := instWith(map[string]map[string]any{"s": {"ok": true}})
	if acts := activatableSteps(dv, afterCompleted); contains(acts, "s") {
		t.Fatalf("InitialStep s must NOT re-activate after completion; got %v", acts)
	}

	// While a step is running, the InitialStep is not re-derived either.
	afterRunning := core.WorkflowInstance{
		InstanceID:   "inst-1",
		Status:       core.InstanceStatusRunning,
		Variables:    map[string]any{"inputs": map[string]any{}},
		CurrentSteps: []string{"p"},
	}
	if acts := activatableSteps(dv, afterRunning); contains(acts, "s") {
		t.Fatalf("InitialStep s must NOT activate while a step is running; got %v", acts)
	}
}
