package core

import (
	"encoding/json"
	"time"
)

// ExecutionEvent is one row of the append-only EventLog, the platform's source
// of truth (TDS-01 §1; Blueprint §9). Field names and types are transcribed
// verbatim from the execution_events DDL and frozen at Gate G1; the log is
// append-only and rows are never rewritten.
type ExecutionEvent struct {
	// EventID is the UUID primary key uniquely identifying the event.
	EventID string `db:"event_id" json:"event_id"`
	// InstanceID is the workflow instance this event belongs to.
	InstanceID InstanceID `db:"instance_id" json:"instance_id"`
	// Namespace is the owning namespace, e.g. "oip".
	Namespace string `db:"namespace" json:"namespace"`
	// EventType is one of the 12 enumerated event types (§2).
	EventType EventType `db:"event_type" json:"event_type"`
	// StepID is the step the event concerns; empty for workflow-level events.
	StepID string `db:"step_id" json:"step_id"`
	// Payload is the per-type JSON payload (schema per §2).
	Payload json.RawMessage `db:"payload" json:"payload"`
	// EmittedAt is the ISO8601 timestamp of emission.
	EmittedAt time.Time `db:"emitted_at" json:"emitted_at"`
	// SequenceNum is monotonically increasing per instance.
	SequenceNum int `db:"sequence_num" json:"sequence_num"`
	// SchemaVersion is the version of the payload-schema pack that wrote this
	// event; defaults to 1 (G1 ADJ-1).
	SchemaVersion int `db:"schema_version" json:"schema_version"`
}

// EventType enumerates the 12 execution event types (TDS-01 §2; Blueprint §9).
type EventType string

// EventType frozen literal values — the 12 event types (TDS-01 §2).
const (
	// EventTypeWorkflowStarted marks workflow start; payload {inputs}.
	EventTypeWorkflowStarted EventType = "WorkflowStarted"
	// EventTypeStepStarted marks a step activation; payload {step_id, attempt, inputs}.
	EventTypeStepStarted EventType = "StepStarted"
	// EventTypeStepCompleted marks a step success; payload {step_id, attempt, outputs, duration_ms}.
	EventTypeStepCompleted EventType = "StepCompleted"
	// EventTypeStepFailed marks a step failure; payload {step_id, attempt, error, retrying}.
	EventTypeStepFailed EventType = "StepFailed"
	// EventTypeStepFallbackActivated marks a fallback edge; payload {step_id, fallback_step_id, reason}.
	EventTypeStepFallbackActivated EventType = "StepFallbackActivated"
	// EventTypeSignalReceived marks a waiting→running transition; payload {signal_name, payload}.
	EventTypeSignalReceived EventType = "SignalReceived"
	// EventTypeWorkflowCompleted marks terminal success; payload {outputs, duration_ms}.
	EventTypeWorkflowCompleted EventType = "WorkflowCompleted"
	// EventTypeWorkflowFailed marks terminal failure; payload {step_id, error}.
	EventTypeWorkflowFailed EventType = "WorkflowFailed"
	// EventTypeWorkflowCancelled marks terminal cancellation; payload {reason} (G1 ADJ-2).
	EventTypeWorkflowCancelled EventType = "WorkflowCancelled"
	// EventTypeWorkflowCompensating marks entry into compensation; payload {from_step}.
	EventTypeWorkflowCompensating EventType = "WorkflowCompensating"
	// EventTypeWorkflowCompensated marks successful compensation; payload {}.
	EventTypeWorkflowCompensated EventType = "WorkflowCompensated"
	// EventTypeWorkflowCompensationFailed marks compensation failure; payload {step_id, error}.
	EventTypeWorkflowCompensationFailed EventType = "WorkflowCompensationFailed"
)

// DomainEvent is one row of the domain_events table (Blueprint §10; TTL 7d;
// ingested at M06 per Verification F-2). It is distinct from ExecutionEvent
// (§9): DomainEvents are external facts that may trigger workflows, not the
// append-only execution history.
//
// Shape completed at M06 (owning milestone; additive CONTRA-4-class table); not
// part of the G1 ExecutionEvent format freeze. The struct is defined here at
// C1; ingestion, matching, and the migration are wired at C3 (T5).
type DomainEvent struct {
	// EventID is the UUID primary key uniquely identifying the domain event.
	EventID string `json:"event_id"`
	// Namespace is the owning namespace, e.g. "oip".
	Namespace string `json:"namespace"`
	// EventType is the domain event's type, matched against trigger config["event"].
	EventType string `json:"event_type"`
	// Source identifies the emitter of the domain event.
	Source string `json:"source"`
	// Payload is the event's data; it becomes a trigger-submitted instance's inputs.
	Payload map[string]any `json:"payload"`
	// EmittedAt is when the domain event was emitted.
	EmittedAt time.Time `json:"emitted_at"`
	// ConsumedAt is when a workflow first fired from this event; nil until consumed.
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`
}

// StepError is the persisted error shape carried inside StepFailed,
// WorkflowFailed, and WorkflowCompensationFailed payloads (TDS-01 §2.1). It is
// part of the irreversible format and frozen at Gate G1.
type StepError struct {
	// Code is the stable, machine-matchable error class.
	Code string `json:"code"`
	// Message is a human-readable description.
	Message string `json:"message"`
	// Details is optional handler-supplied context.
	Details map[string]any `json:"details,omitempty"`
}
