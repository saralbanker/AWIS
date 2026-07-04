package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// marshalPayload marshals a typed payload struct to json.RawMessage.
func marshalPayload(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("engine: marshal payload %T: %w", v, err)
	}
	return b, nil
}

// emitWorkflowStarted emits WorkflowStarted (seq 1) and creates the projection
// row. defID/defVer/ns seed the row's non-evented identity fields (EDR-007 gap).
func (e *Engine) emitWorkflowStarted(ctx context.Context, iid core.InstanceID, ns, defID string, defVer core.SemVer, inputs map[string]any) error {
	payload, err := marshalPayload(workflowStartedPayload{Inputs: inputs})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: iid,
		Namespace:  ns,
		EventType:  core.EventTypeWorkflowStarted,
		Payload:    payload,
		EmittedAt:  e.now(),
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, defID, defVer)
}

// emitStepStarted emits StepStarted and returns the emission time so the caller
// can compute the step's duration at completion.
func (e *Engine) emitStepStarted(ctx context.Context, inst core.WorkflowInstance, stepID string, attempt int, inputs map[string]any) (time.Time, error) {
	payload, err := marshalPayload(stepStartedPayload{StepID: stepID, Attempt: attempt, Inputs: inputs})
	if err != nil {
		return time.Time{}, err
	}
	at := e.now()
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeStepStarted,
		StepID:     stepID,
		Payload:    payload,
		EmittedAt:  at,
	}
	e.logEmit(ev)
	if err := e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion); err != nil {
		return time.Time{}, err
	}
	return at, nil
}

// emitStepCompleted emits StepCompleted with the plain four-field payload
// (ADJ-8 usage triple is C2's intelligence concern). duration_ms is
// emitted_at − startedAt; a single clock read backs both so they stay consistent.
func (e *Engine) emitStepCompleted(ctx context.Context, inst core.WorkflowInstance, stepID string, attempt int, outputs map[string]any, startedAt time.Time) error {
	at := e.now()
	payload, err := marshalPayload(stepCompletedPayload{
		StepID:     stepID,
		Attempt:    attempt,
		Outputs:    outputs,
		DurationMs: at.Sub(startedAt).Milliseconds(),
	})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeStepCompleted,
		StepID:     stepID,
		Payload:    payload,
		EmittedAt:  at,
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion)
}

// emitStepFailed emits StepFailed. C1 only ever emits retrying=false (the
// degenerate no-retry path); C2 adds the retrying=true site.
func (e *Engine) emitStepFailed(ctx context.Context, inst core.WorkflowInstance, stepID string, attempt int, stepErr core.StepError, retrying bool) error {
	payload, err := marshalPayload(stepFailedPayload{
		StepID:   stepID,
		Attempt:  attempt,
		Error:    stepErr,
		Retrying: retrying,
	})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeStepFailed,
		StepID:     stepID,
		Payload:    payload,
		EmittedAt:  e.now(),
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion)
}

// emitWorkflowCompleted emits WorkflowCompleted (EDR-011 §2 payload). duration_ms
// is emitted_at − StartedAt; a single clock read backs both.
func (e *Engine) emitWorkflowCompleted(ctx context.Context, inst core.WorkflowInstance, outputs map[string]any) error {
	at := e.now()
	payload, err := marshalPayload(workflowCompletedPayload{
		Outputs:    outputs,
		DurationMs: at.Sub(inst.StartedAt).Milliseconds(),
	})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeWorkflowCompleted,
		Payload:    payload,
		EmittedAt:  at,
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion)
}

// emitWorkflowFailed emits WorkflowFailed (the C1 degenerate terminal-failure
// path: no retry, no fallback — the correct frozen behaviour when no policy or
// fallback exists; C2 generalizes the path, it does not replace it).
func (e *Engine) emitWorkflowFailed(ctx context.Context, inst core.WorkflowInstance, stepID string, stepErr core.StepError) error {
	payload, err := marshalPayload(workflowFailedPayload{StepID: stepID, Error: stepErr})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeWorkflowFailed,
		Payload:    payload,
		EmittedAt:  e.now(),
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion)
}

// logEmit records an emission at Info with the structured keys required by the
// IMP DoD (instance_id / step_id / event_type).
func (e *Engine) logEmit(ev core.ExecutionEvent) {
	e.logger.Info("emit",
		"instance_id", string(ev.InstanceID),
		"step_id", ev.StepID,
		"event_type", string(ev.EventType),
	)
}
