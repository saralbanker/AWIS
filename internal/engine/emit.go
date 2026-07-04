package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// ── Payload structs (TDS-01 §2, VERBATIM field names) ───────────────────────────
//
// Field names/types are transcribed verbatim from EVENTLOG_FORMAT.md §2. The
// optional ADJ-8 usage triple on StepCompleted (adapter/model/tokens_used) is a
// C2 (intelligence) concern; C1 emits the plain four-field StepCompleted.

type workflowStartedPayload struct {
	Inputs map[string]any `json:"inputs"`
}

type stepStartedPayload struct {
	StepID  string         `json:"step_id"`
	Attempt int            `json:"attempt"`
	Inputs  map[string]any `json:"inputs"`
}

type stepCompletedPayload struct {
	StepID     string         `json:"step_id"`
	Attempt    int            `json:"attempt"`
	Outputs    map[string]any `json:"outputs"`
	DurationMs int64          `json:"duration_ms"`
}

// stepCompletedUsagePayload is the ADJ-8 (CONTRA-11) StepCompleted variant that
// carries the optional {adapter, model, tokens_used} usage triple for
// INTELLIGENCE steps only. The triple is written WITHOUT omitempty so a null-path
// tokens_used of 0 is present (the null adapter yields "null"/"null"/0). Native
// steps use the plain four-field stepCompletedPayload — the usage keys are absent
// entirely. Projection reads only `outputs`, so the two payloads are
// replay-equivalent (EDR-007; ADJ-8 "replay-neutral").
type stepCompletedUsagePayload struct {
	StepID     string         `json:"step_id"`
	Attempt    int            `json:"attempt"`
	Outputs    map[string]any `json:"outputs"`
	DurationMs int64          `json:"duration_ms"`
	Adapter    string         `json:"adapter"`
	Model      string         `json:"model"`
	TokensUsed int            `json:"tokens_used"`
}

// stepFallbackActivatedPayload is StepFallbackActivated's payload (TDS-01 §2:
// {step_id, fallback_step_id, reason}).
type stepFallbackActivatedPayload struct {
	StepID         string `json:"step_id"`
	FallbackStepID string `json:"fallback_step_id"`
	Reason         string `json:"reason"`
}

// workflowCancelledPayload is WorkflowCancelled's payload — {reason} ONLY
// (CONTRA-5 / G1 ADJ-2; cancelled_at is derived from the envelope's emitted_at,
// never stored in the payload).
type workflowCancelledPayload struct {
	Reason string `json:"reason"`
}

// workflowCompensatingPayload is WorkflowCompensating's payload ({from_step}).
type workflowCompensatingPayload struct {
	FromStep string `json:"from_step"`
}

// workflowCompensatedPayload is WorkflowCompensated's payload — {} (empty object).
type workflowCompensatedPayload struct{}

// workflowCompensationFailedPayload is WorkflowCompensationFailed's payload
// ({step_id, error}).
type workflowCompensationFailedPayload struct {
	StepID string         `json:"step_id"`
	Error  core.StepError `json:"error"`
}

type stepFailedPayload struct {
	StepID   string         `json:"step_id"`
	Attempt  int            `json:"attempt"`
	Error    core.StepError `json:"error"`
	Retrying bool           `json:"retrying"`
}

type workflowCompletedPayload struct {
	Outputs    map[string]any `json:"outputs"`
	DurationMs int64          `json:"duration_ms"`
}

type workflowFailedPayload struct {
	StepID string         `json:"step_id"`
	Error  core.StepError `json:"error"`
}

// ── Sequence assignment + append (EDR-005) ──────────────────────────────────────

// appendEvent assigns the next per-instance sequence_num and appends ev. On
// storage.ErrSequenceViolation the in-memory counter is reloaded from MAX+1 (the
// recovery path when the engine's view is stale) and the append is retried ONCE
// (EDR-005). The caller must have set EventID? No — appendEvent fills EventID,
// SchemaVersion, and SequenceNum; the caller sets the rest (payload, emitted_at).
func (e *Engine) appendEvent(ctx context.Context, ev *core.ExecutionEvent) error {
	ev.EventID = e.newID()
	ev.SchemaVersion = 1

	e.mu.Lock()
	e.seq[ev.InstanceID]++
	ev.SequenceNum = e.seq[ev.InstanceID]
	e.mu.Unlock()

	err := e.storage.AppendEvent(ctx, *ev)
	if err == nil {
		return nil
	}
	if !errors.Is(err, storage.ErrSequenceViolation) {
		return fmt.Errorf("engine: append %s seq=%d: %w", ev.EventType, ev.SequenceNum, err)
	}

	// Reload MAX+1 and retry once (single-writer reconciliation).
	max, rerr := e.maxSeq(ctx, ev.InstanceID)
	if rerr != nil {
		return rerr
	}
	e.mu.Lock()
	e.seq[ev.InstanceID] = max + 1
	ev.SequenceNum = e.seq[ev.InstanceID]
	e.mu.Unlock()

	if err := e.storage.AppendEvent(ctx, *ev); err != nil {
		return fmt.Errorf("engine: append retry %s seq=%d: %w", ev.EventType, ev.SequenceNum, err)
	}
	return nil
}

// maxSeq returns the highest sequence_num persisted for instanceID, or 0.
func (e *Engine) maxSeq(ctx context.Context, instanceID core.InstanceID) (int, error) {
	evs, err := e.storage.ReadEvents(ctx, instanceID, 0)
	if err != nil {
		return 0, fmt.Errorf("engine: reload max seq for %s: %w", instanceID, err)
	}
	max := 0
	for _, ev := range evs {
		if ev.SequenceNum > max {
			max = ev.SequenceNum
		}
	}
	return max, nil
}

// ── Forward projection (EDR-007 §2 — mirrors RebuildState) ──────────────────────

// emit is the single event-emission chokepoint: it appends the (already fully
// built) event and then upserts the forward projection. defID/defVer seed a
// freshly-created row (WorkflowStarted only; every later event finds the row and
// its identity preserved). All emit calls are made from the serial tick/submit
// path — never from dispatch goroutines.
func (e *Engine) emit(ctx context.Context, ev *core.ExecutionEvent, defID string, defVer core.SemVer) error {
	if err := e.appendEvent(ctx, ev); err != nil {
		return err
	}
	return e.project(ctx, *ev, defID, defVer)
}

// project applies ev to the workflow_instances projection under optimistic
// concurrency control, mirroring EDR-007 §2 exactly. On ErrVersionConflict it
// re-reads and re-applies ONCE (the card's version-conflict retry).
func (e *Engine) project(ctx context.Context, ev core.ExecutionEvent, defID string, defVer core.SemVer) error {
	for attempt := 0; attempt < 2; attempt++ {
		e.mu.Lock()
		expected := e.ver[ev.InstanceID]
		e.mu.Unlock()

		inst, found, err := e.getInstance(ctx, ev.InstanceID)
		if err != nil {
			return err
		}
		if !found {
			inst = core.WorkflowInstance{
				InstanceID:        ev.InstanceID,
				DefinitionID:      defID,
				DefinitionVersion: defVer,
			}
		}
		if err := applyEventToInstance(&inst, ev); err != nil {
			return err
		}

		err = e.storage.UpsertInstance(ctx, inst, expected)
		if err == nil {
			e.mu.Lock()
			e.ver[ev.InstanceID] = expected + 1
			e.mu.Unlock()
			return nil
		}
		if errors.Is(err, storage.ErrVersionConflict) && attempt == 0 {
			// A version we did not account for (e.g. a claim bump); advance the
			// tracked version by one and re-read/re-apply once.
			e.mu.Lock()
			e.ver[ev.InstanceID] = expected + 1
			e.mu.Unlock()
			continue
		}
		return fmt.Errorf("engine: upsert projection %s (instance=%s): %w", ev.EventType, ev.InstanceID, err)
	}
	return nil
}

// getInstance wraps GetInstance, mapping ErrInstanceNotFound to found=false.
func (e *Engine) getInstance(ctx context.Context, instanceID core.InstanceID) (core.WorkflowInstance, bool, error) {
	inst, err := e.storage.GetInstance(ctx, instanceID)
	if err != nil {
		if errors.Is(err, storage.ErrInstanceNotFound) {
			return core.WorkflowInstance{}, false, nil
		}
		return core.WorkflowInstance{}, false, fmt.Errorf("engine: get instance %s: %w", instanceID, err)
	}
	return inst, true, nil
}

// bumpVersion records a version increment caused by a successful ClaimStep (which
// bumps workflow_instances.version in the same transaction, EDR-006). Keeping the
// engine's tracked version in lock-step with claim bumps is what makes the
// single-writer OCC exact (no conflict in normal operation).
func (e *Engine) bumpVersion(instanceID core.InstanceID) {
	e.mu.Lock()
	e.ver[instanceID]++
	e.mu.Unlock()
}

// applyEventToInstance mutates inst per EDR-007 §2's status map. It MUST stay
// byte-for-byte equivalent to internal/storage.projectInstance: the M06
// event-sequence fixtures assert forward-path ≡ RebuildState. It reads values
// from ev.Payload the same way projectInstance does (payload JSON, not the
// engine's in-memory typed payload) so both paths consume identical bytes.
//
// UpdatedAt is always set to the event's emitted_at (EDR-007 §2). All 12 event
// types are handled now (per the card) so C2/C3 only add emission sites.
//
// semantics-bearing: single projection rule set, forward ≡ rebuild (EDR-007 §10).
func applyEventToInstance(inst *core.WorkflowInstance, ev core.ExecutionEvent) error {
	var payload map[string]json.RawMessage
	if len(ev.Payload) > 0 {
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return fmt.Errorf("engine: unmarshal %s payload: %w", ev.EventType, err)
		}
	}
	emittedAt := ev.EmittedAt.UTC()

	switch ev.EventType {
	case core.EventTypeWorkflowStarted:
		inst.Namespace = ev.Namespace
		inst.Status = core.InstanceStatusRunning
		inst.StartedAt = emittedAt
		inst.CurrentSteps = []string{}
		var inputs any
		if raw, ok := payload["inputs"]; ok {
			if err := json.Unmarshal(raw, &inputs); err != nil {
				return fmt.Errorf("engine: WorkflowStarted.inputs: %w", err)
			}
		}
		inst.Variables = map[string]any{"inputs": inputs}

	case core.EventTypeStepStarted:
		pid, err := stepIDOf(ev, payload)
		if err != nil {
			return err
		}
		if pid != "" {
			inst.CurrentSteps = appendUniq(inst.CurrentSteps, pid)
		}

	case core.EventTypeStepCompleted:
		pid, err := stepIDOf(ev, payload)
		if err != nil {
			return err
		}
		if pid != "" {
			inst.CurrentSteps = removeStep(inst.CurrentSteps, pid)
			var outputs any
			if err := unmarshalField(payload, "outputs", &outputs); err != nil {
				return err
			}
			if inst.Variables == nil {
				inst.Variables = map[string]any{}
			}
			inst.Variables[pid] = outputs
		}

	case core.EventTypeStepFailed:
		var retrying bool
		if err := unmarshalField(payload, "retrying", &retrying); err != nil {
			return err
		}
		if !retrying {
			pid, err := stepIDOf(ev, payload)
			if err != nil {
				return err
			}
			if pid != "" {
				inst.CurrentSteps = removeStep(inst.CurrentSteps, pid)
			}
		}

	case core.EventTypeStepFallbackActivated:
		var pid string
		if err := unmarshalField(payload, "step_id", &pid); err != nil {
			return err
		}
		if pid != "" {
			inst.CurrentSteps = removeStep(inst.CurrentSteps, pid)
		}

	case core.EventTypeSignalReceived:
		inst.Status = core.InstanceStatusRunning

	case core.EventTypeWorkflowCompleted:
		inst.Status = core.InstanceStatusCompleted
		inst.CurrentSteps = []string{}
		inst.CompletedAt = &emittedAt

	case core.EventTypeWorkflowFailed:
		inst.Status = core.InstanceStatusFailed
		inst.CompletedAt = &emittedAt

	case core.EventTypeWorkflowCancelled:
		inst.Status = core.InstanceStatusCancelled
		inst.CompletedAt = &emittedAt

	case core.EventTypeWorkflowCompensating:
		inst.Status = core.InstanceStatusCompensating

	case core.EventTypeWorkflowCompensated:
		inst.Status = core.InstanceStatusCompensated
		inst.CompletedAt = &emittedAt

	case core.EventTypeWorkflowCompensationFailed:
		inst.Status = core.InstanceStatusCompensationFailed
		inst.CompletedAt = &emittedAt
	}

	inst.UpdatedAt = emittedAt
	return nil
}

// stepIDOf reads the event's step id from the envelope, falling back to the
// payload (mirrors projectInstance).
func stepIDOf(ev core.ExecutionEvent, payload map[string]json.RawMessage) (string, error) {
	if ev.StepID != "" {
		return ev.StepID, nil
	}
	var pid string
	if err := unmarshalField(payload, "step_id", &pid); err != nil {
		return "", err
	}
	return pid, nil
}

// unmarshalField decodes payload[key] into dst when present; absent leaves dst
// untouched (mirrors internal/storage.unmarshalField).
func unmarshalField(payload map[string]json.RawMessage, key string, dst any) error {
	raw, ok := payload[key]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("engine: field %q: %w", key, err)
	}
	return nil
}

// appendUniq appends s only if absent (mirrors internal/storage.appendUniq).
func appendUniq(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// removeStep removes the first occurrence of s (mirrors internal/storage.removeStep).
func removeStep(slice []string, s string) []string {
	for i, v := range slice {
		if v == s {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
