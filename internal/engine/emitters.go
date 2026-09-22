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

// emitStepCompletedUsage emits the ADJ-8 StepCompleted variant carrying the
// {adapter, model, tokens_used} usage triple (intelligence steps only). Callers
// pass a non-nil usage only for intelligence-type steps with a live provider
// side-channel; native steps use emitStepCompleted (no usage keys).
func (e *Engine) emitStepCompletedUsage(ctx context.Context, inst core.WorkflowInstance, stepID string, attempt int, outputs map[string]any, startedAt time.Time, usage core.Usage) error {
	at := e.now()
	payload, err := marshalPayload(stepCompletedUsagePayload{
		StepID:     stepID,
		Attempt:    attempt,
		Outputs:    outputs,
		DurationMs: at.Sub(startedAt).Milliseconds(),
		Adapter:    usage.Adapter,
		Model:      usage.Model,
		TokensUsed: usage.TokensUsed,
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

// emitStepFallbackActivated emits StepFallbackActivated (EDR-011 §8 case 1;
// TDS-01 §2 payload {step_id, fallback_step_id, reason}). The projection removes
// step_id from current_steps (EDR-007). reason carries the terminal StepError's
// code (a stable, machine-matchable value; the frozen format leaves it a free
// string — C2r judgment call).
func (e *Engine) emitStepFallbackActivated(ctx context.Context, inst core.WorkflowInstance, stepID, fallbackStepID, reason string) error {
	payload, err := marshalPayload(stepFallbackActivatedPayload{
		StepID:         stepID,
		FallbackStepID: fallbackStepID,
		Reason:         reason,
	})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeStepFallbackActivated,
		StepID:     stepID,
		Payload:    payload,
		EmittedAt:  e.now(),
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion)
}

// emitWorkflowCancelled emits WorkflowCancelled with {reason} ONLY (CONTRA-5).
func (e *Engine) emitWorkflowCancelled(ctx context.Context, inst core.WorkflowInstance, reason string) error {
	payload, err := marshalPayload(workflowCancelledPayload{Reason: reason})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeWorkflowCancelled,
		Payload:    payload,
		EmittedAt:  e.now(),
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion)
}

// emitWorkflowCompensating emits WorkflowCompensating ({from_step}).
func (e *Engine) emitWorkflowCompensating(ctx context.Context, inst core.WorkflowInstance, fromStep string) error {
	payload, err := marshalPayload(workflowCompensatingPayload{FromStep: fromStep})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeWorkflowCompensating,
		Payload:    payload,
		EmittedAt:  e.now(),
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion)
}

// emitWorkflowCompensated emits WorkflowCompensated ({} empty payload).
func (e *Engine) emitWorkflowCompensated(ctx context.Context, inst core.WorkflowInstance) error {
	payload, err := marshalPayload(workflowCompensatedPayload{})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeWorkflowCompensated,
		Payload:    payload,
		EmittedAt:  e.now(),
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion)
}

// emitWorkflowCompensationFailed emits WorkflowCompensationFailed ({step_id,
// error}). EDR-011 §5: the audit_log write is M07/F-4; at M06 the caller also
// slog.Error's the failure.
func (e *Engine) emitWorkflowCompensationFailed(ctx context.Context, inst core.WorkflowInstance, stepID string, stepErr core.StepError) error {
	payload, err := marshalPayload(workflowCompensationFailedPayload{StepID: stepID, Error: stepErr})
	if err != nil {
		return err
	}
	ev := core.ExecutionEvent{
		InstanceID: inst.InstanceID,
		Namespace:  inst.Namespace,
		EventType:  core.EventTypeWorkflowCompensationFailed,
		StepID:     stepID,
		Payload:    payload,
		EmittedAt:  e.now(),
	}
	e.logEmit(ev)
	return e.emit(ctx, &ev, inst.DefinitionID, inst.DefinitionVersion)
}

// logEmit records an emission at Info with the structured keys required by the
// IMP DoD (instance_id / step_id / event_type).
func (e *Engine) logEmit(ev core.ExecutionEvent) {
	e.log().Info("emit",
		"instance_id", string(ev.InstanceID),
		"step_id", ev.StepID,
		"event_type", string(ev.EventType),
	)
}
