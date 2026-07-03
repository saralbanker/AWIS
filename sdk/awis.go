// Package sdk is the public surface applications use to interact with AWIS. It
// is the ONLY package applications import; the runtime internals stay hidden
// (Blueprint §12). Under Amendment F-1 every type here is a type alias for the
// canonical definition in internal/core, so the serialized field names frozen at
// Gate G1 (TDS-01, TDS-02) live in exactly one place. This file re-exports the
// EventLog envelope, event types, the persisted StepError, the storage port, and
// the shared scalar types.
package sdk

import "github.com/awis/awis/internal/core"

// SemVer is a semantic version string, e.g. "1.0.0". A workflow's version is
// immutable once registered.
type SemVer = core.SemVer

// HandlerRef identifies a step's handler: a native "handler-name" or a
// subprocess "script.py", etc.
type HandlerRef = core.HandlerRef

// Condition is an expression over step outputs; an absent condition means the
// transition is unconditional. This layer carries the source string only.
type Condition = core.Condition

// Duration is a workflow- or step-level duration.
type Duration = core.Duration

// InputSchema is the JSON Schema describing a step's expected inputs.
type InputSchema = core.InputSchema

// OutputSchema is the JSON Schema describing a step's produced outputs.
type OutputSchema = core.OutputSchema

// InstanceID is the unique identity of a workflow instance (UUID form).
type InstanceID = core.InstanceID

// IdempotencyKey keys the step-result cache: hash(instance_id + step_id + attempt).
type IdempotencyKey = core.IdempotencyKey

// ExecutionEvent is one row of the append-only EventLog, the platform's source
// of truth. Field names and types are frozen at Gate G1; the log is append-only
// and rows are never rewritten.
type ExecutionEvent = core.ExecutionEvent

// EventType enumerates the 12 execution event types.
type EventType = core.EventType

// The 12 frozen execution event types.
const (
	// EventTypeWorkflowStarted marks workflow start; payload {inputs}.
	EventTypeWorkflowStarted = core.EventTypeWorkflowStarted
	// EventTypeStepStarted marks a step activation; payload {step_id, attempt, inputs}.
	EventTypeStepStarted = core.EventTypeStepStarted
	// EventTypeStepCompleted marks a step success; payload {step_id, attempt, outputs, duration_ms}.
	EventTypeStepCompleted = core.EventTypeStepCompleted
	// EventTypeStepFailed marks a step failure; payload {step_id, attempt, error, retrying}.
	EventTypeStepFailed = core.EventTypeStepFailed
	// EventTypeStepFallbackActivated marks a fallback edge; payload {step_id, fallback_step_id, reason}.
	EventTypeStepFallbackActivated = core.EventTypeStepFallbackActivated
	// EventTypeSignalReceived marks a waiting→running transition; payload {signal_name, payload}.
	EventTypeSignalReceived = core.EventTypeSignalReceived
	// EventTypeWorkflowCompleted marks terminal success; payload {outputs, duration_ms}.
	EventTypeWorkflowCompleted = core.EventTypeWorkflowCompleted
	// EventTypeWorkflowFailed marks terminal failure; payload {step_id, error}.
	EventTypeWorkflowFailed = core.EventTypeWorkflowFailed
	// EventTypeWorkflowCancelled marks terminal cancellation; payload {reason}.
	EventTypeWorkflowCancelled = core.EventTypeWorkflowCancelled
	// EventTypeWorkflowCompensating marks entry into compensation; payload {from_step}.
	EventTypeWorkflowCompensating = core.EventTypeWorkflowCompensating
	// EventTypeWorkflowCompensated marks successful compensation; payload {}.
	EventTypeWorkflowCompensated = core.EventTypeWorkflowCompensated
	// EventTypeWorkflowCompensationFailed marks compensation failure; payload {step_id, error}.
	EventTypeWorkflowCompensationFailed = core.EventTypeWorkflowCompensationFailed
)

// StepError is the persisted error shape carried inside StepFailed,
// WorkflowFailed, and WorkflowCompensationFailed payloads. It is part of the
// irreversible format and frozen at Gate G1: Code is a stable machine-matchable
// class, Message is human-readable, Details is optional handler context.
type StepError = core.StepError

// StoragePort is the persistence boundary for the runtime: EventLog, StateStore,
// WorkflowRegistry, and StepResultCache. The method set is frozen; adapters
// (SQLite, Postgres) implement it identically.
type StoragePort = core.StoragePort
