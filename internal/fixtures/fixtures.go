// Package fixtures provides shared, deterministic workflow definitions for use
// across engine-level and harness-level tests (IMP §19 test-data policy: one
// shared fixture package so a semantic change breaks loudly everywhere at once).
//
// Each exported function returns a *core.WorkflowDefinition and the native
// handlers it needs. Fixtures never reference wall-clock time or random sources;
// all non-determinism is injected through the harness seams.
package fixtures

import (
	"errors"

	"github.com/awis/awis/internal/core"
)

// nativeHandler is a deterministic StepHandler used inside fixtures.
type nativeHandler struct {
	id      string
	outputs map[string]any
	err     error
}

func (h *nativeHandler) ID() string { return h.id }
func (h *nativeHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	if h.err != nil {
		return core.StepResult{}, h.err
	}
	return core.StepResult{Outputs: h.outputs}, nil
}

func manualTrigger() []core.Trigger {
	return []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}}
}

// Linear returns a three-step sequential workflow (a → b → c) and its handlers.
// All steps succeed. Terminal status: Completed.
func Linear() (*core.WorkflowDefinition, []core.StepHandler) {
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "fixtures.linear",
		Version:       "1.0.0",
		Namespace:     "fixtures",
		Name:          "Linear",
		Triggers:      manualTrigger(),
		Steps: []core.Step{
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "fixtures.linear.a"},
			{ID: "b", Name: "b", Type: core.StepTypeNative, Handler: "fixtures.linear.b"},
			{ID: "c", Name: "c", Type: core.StepTypeNative, Handler: "fixtures.linear.c"},
		},
		Transitions: []core.Transition{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
		InitialStep: "a",
		FinalSteps:  []string{"c"},
		Metadata:    map[string]any{},
	}
	return def, []core.StepHandler{
		&nativeHandler{id: "fixtures.linear.a", outputs: map[string]any{"result": "a-done"}},
		&nativeHandler{id: "fixtures.linear.b", outputs: map[string]any{"result": "b-done"}},
		&nativeHandler{id: "fixtures.linear.c", outputs: map[string]any{"result": "c-done"}},
	}
}

// FanOutJoin returns a diamond workflow (start → branch_a + branch_b → join) and
// its handlers. Both branches run concurrently (MaxParallelSteps ≥ 2, which is
// the engine default). Terminal status: Completed.
func FanOutJoin() (*core.WorkflowDefinition, []core.StepHandler) {
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "fixtures.fanout",
		Version:       "1.0.0",
		Namespace:     "fixtures",
		Name:          "FanOutJoin",
		Triggers:      manualTrigger(),
		Steps: []core.Step{
			{ID: "start", Name: "start", Type: core.StepTypeNative, Handler: "fixtures.fo.start"},
			{ID: "branch_a", Name: "branch_a", Type: core.StepTypeNative, Handler: "fixtures.fo.branch_a"},
			{ID: "branch_b", Name: "branch_b", Type: core.StepTypeNative, Handler: "fixtures.fo.branch_b"},
			{ID: "join", Name: "join", Type: core.StepTypeNative, Handler: "fixtures.fo.join"},
		},
		Transitions: []core.Transition{
			{From: "start", To: "branch_a"},
			{From: "start", To: "branch_b"},
			{From: "branch_a", To: "join"},
			{From: "branch_b", To: "join"},
		},
		InitialStep: "start",
		FinalSteps:  []string{"join"},
		Metadata:    map[string]any{},
	}
	return def, []core.StepHandler{
		&nativeHandler{id: "fixtures.fo.start", outputs: map[string]any{"fanned": true}},
		&nativeHandler{id: "fixtures.fo.branch_a", outputs: map[string]any{"branch": "a"}},
		&nativeHandler{id: "fixtures.fo.branch_b", outputs: map[string]any{"branch": "b"}},
		&nativeHandler{id: "fixtures.fo.join", outputs: map[string]any{"joined": true}},
	}
}

// RetryExhaustionFallback returns a workflow where the primary step always fails
// after all retry attempts (Attempts: 2) and the Fallback field routes to a
// recovery step. Terminal status: Completed (via the recovery step).
func RetryExhaustionFallback() (*core.WorkflowDefinition, []core.StepHandler) {
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "fixtures.retry-fallback",
		Version:       "1.0.0",
		Namespace:     "fixtures",
		Name:          "RetryExhaustionFallback",
		Triggers:      manualTrigger(),
		Steps: []core.Step{
			{
				ID:      "flaky",
				Name:    "flaky",
				Type:    core.StepTypeNative,
				Handler: "fixtures.rf.flaky",
				Retry:   &core.RetryPolicy{Attempts: 2, Backoff: "immediate"},
				Fallback: "recovery",
			},
			{ID: "recovery", Name: "recovery", Type: core.StepTypeNative, Handler: "fixtures.rf.recovery"},
		},
		InitialStep: "flaky",
		FinalSteps:  []string{"recovery"},
		Metadata:    map[string]any{},
	}
	return def, []core.StepHandler{
		&nativeHandler{id: "fixtures.rf.flaky", err: errors.New("transient failure")},
		&nativeHandler{id: "fixtures.rf.recovery", outputs: map[string]any{"recovered": true}},
	}
}

// Compensation returns a workflow where step2 always fails; the engine runs the
// CompensationPlan (undoing step1) before reporting terminal status.
// Terminal status: Compensated.
func Compensation() (*core.WorkflowDefinition, []core.StepHandler) {
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "fixtures.compensation",
		Version:       "1.0.0",
		Namespace:     "fixtures",
		Name:          "Compensation",
		Triggers:      manualTrigger(),
		Steps: []core.Step{
			{ID: "step1", Name: "step1", Type: core.StepTypeNative, Handler: "fixtures.comp.step1"},
			{ID: "step2", Name: "step2", Type: core.StepTypeNative, Handler: "fixtures.comp.step2"},
		},
		Transitions: []core.Transition{{From: "step1", To: "step2"}},
		Compensation: &core.CompensationPlan{
			Steps: []core.CompensationStep{
				{StepID: "step1", UndoHandler: "fixtures.comp.undo1"},
			},
		},
		InitialStep: "step1",
		FinalSteps:  []string{"step2"},
		Metadata:    map[string]any{},
	}
	return def, []core.StepHandler{
		&nativeHandler{id: "fixtures.comp.step1", outputs: map[string]any{"committed": true}},
		&nativeHandler{id: "fixtures.comp.step2", err: errors.New("step2 failed — triggers compensation")},
		&nativeHandler{id: "fixtures.comp.undo1", outputs: map[string]any{"undone": true}},
	}
}

// Cancellation returns a workflow that parks at a signal WAIT step with no
// timeout, suitable for explicit-cancel tests. Deliver cancellation via
// Runtime.Cancel; expected terminal status: Cancelled.
func Cancellation() (*core.WorkflowDefinition, []core.StepHandler) {
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "fixtures.cancellation",
		Version:       "1.0.0",
		Namespace:     "fixtures",
		Name:          "Cancellation",
		Triggers:      manualTrigger(),
		Steps: []core.Step{
			{
				ID:         "wait",
				Name:       "wait",
				Type:       core.StepTypeSignal,
				WaitSignal: &core.WaitConfig{SignalName: "proceed", TimeoutAction: "fail"},
			},
		},
		InitialStep: "wait",
		FinalSteps:  []string{"wait"},
		Metadata:    map[string]any{},
	}
	return def, nil
}

// CancellationWithCompensate returns a workflow that completes step1 and then
// parks at a signal WAIT step. A CompensationPlan is registered (but not
// triggered by a plain Cancel call). Expected terminal status after
// Runtime.Cancel: Cancelled.
func CancellationWithCompensate() (*core.WorkflowDefinition, []core.StepHandler) {
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "fixtures.cancellation-comp",
		Version:       "1.0.0",
		Namespace:     "fixtures",
		Name:          "CancellationWithCompensate",
		Triggers:      manualTrigger(),
		Steps: []core.Step{
			{ID: "step1", Name: "step1", Type: core.StepTypeNative, Handler: "fixtures.cc.step1"},
			{
				ID:         "wait",
				Name:       "wait",
				Type:       core.StepTypeSignal,
				WaitSignal: &core.WaitConfig{SignalName: "proceed", TimeoutAction: "fail"},
			},
		},
		Transitions: []core.Transition{{From: "step1", To: "wait"}},
		Compensation: &core.CompensationPlan{
			Steps: []core.CompensationStep{
				{StepID: "step1", UndoHandler: "fixtures.cc.undo1"},
			},
		},
		InitialStep: "step1",
		FinalSteps:  []string{"wait"},
		Metadata:    map[string]any{},
	}
	return def, []core.StepHandler{
		&nativeHandler{id: "fixtures.cc.step1", outputs: map[string]any{"ready": true}},
		&nativeHandler{id: "fixtures.cc.undo1", outputs: map[string]any{"undone": true}},
	}
}

// WaitSignal returns a workflow that executes a native step, parks at a signal
// WAIT, then resumes with a second native step after signal delivery.
// Terminal status: Completed.
func WaitSignal() (*core.WorkflowDefinition, []core.StepHandler) {
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "fixtures.waitsignal",
		Version:       "1.0.0",
		Namespace:     "fixtures",
		Name:          "WaitSignal",
		Triggers:      manualTrigger(),
		Steps: []core.Step{
			{ID: "before", Name: "before", Type: core.StepTypeNative, Handler: "fixtures.ws.before"},
			{
				ID:         "wait",
				Name:       "wait",
				Type:       core.StepTypeSignal,
				WaitSignal: &core.WaitConfig{SignalName: "approve", TimeoutAction: "fail"},
			},
			{ID: "after", Name: "after", Type: core.StepTypeNative, Handler: "fixtures.ws.after"},
		},
		Transitions: []core.Transition{
			{From: "before", To: "wait"},
			{From: "wait", To: "after"},
		},
		InitialStep: "before",
		FinalSteps:  []string{"after"},
		Metadata:    map[string]any{},
	}
	return def, []core.StepHandler{
		&nativeHandler{id: "fixtures.ws.before", outputs: map[string]any{"pre": "done"}},
		&nativeHandler{id: "fixtures.ws.after", outputs: map[string]any{"post": "done"}},
	}
}

// TimeoutAction returns a workflow with a signal WAIT step that carries a 5ms
// timeout and timeout_action=fail. With the deterministic clock advancing 1ms
// per call, the timeout fires within a handful of ticks. Terminal status: Failed.
func TimeoutAction() (*core.WorkflowDefinition, []core.StepHandler) {
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "fixtures.timeout",
		Version:       "1.0.0",
		Namespace:     "fixtures",
		Name:          "TimeoutAction",
		Triggers:      manualTrigger(),
		Steps: []core.Step{
			{
				ID:   "wait",
				Name: "wait",
				Type: core.StepTypeSignal,
				WaitSignal: &core.WaitConfig{
					SignalName:    "never-arrives",
					Timeout:       core.Duration("5ms"),
					TimeoutAction: "fail",
				},
			},
		},
		InitialStep: "wait",
		FinalSteps:  []string{"wait"},
		Metadata:    map[string]any{},
	}
	return def, nil
}
