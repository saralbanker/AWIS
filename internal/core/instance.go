package core

import "time"

// WorkflowInstance is a running instance of a WorkflowDefinition; it mirrors the
// StateStore workflow_instances projection (Blueprint §6; TDS-02 appendix). The
// StateStore is a rebuildable projection of the EventLog.
type WorkflowInstance struct {
	// InstanceID is the unique identity of the instance (UUID form).
	InstanceID InstanceID `json:"instance_id"`
	// DefinitionID is the id of the workflow definition.
	DefinitionID string `json:"definition_id"`
	// DefinitionVersion is the SemVer of the workflow definition.
	DefinitionVersion SemVer `json:"definition_version"`
	// Namespace is the owning namespace.
	Namespace string `json:"namespace"`
	// Status is the current lifecycle status.
	Status InstanceStatus `json:"status"`
	// CurrentSteps are the active (running/waiting) step ids.
	CurrentSteps []string `json:"current_steps"`
	// Variables are accumulated step outputs plus workflow inputs.
	Variables map[string]any `json:"variables"`
	// StartedAt is when the instance started.
	StartedAt time.Time `json:"started_at"`
	// UpdatedAt is when the instance was last updated.
	UpdatedAt time.Time `json:"updated_at"`
	// CompletedAt is when the instance completed; nil while still active.
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// InstanceStatus enumerates the lifecycle states of a workflow instance
// (Blueprint §6, line 328; TDS-02 appendix).
type InstanceStatus string

// InstanceStatus frozen literal values (G1).
const (
	// InstanceStatusPending is created but not yet running.
	InstanceStatusPending InstanceStatus = "pending"
	// InstanceStatusRunning is actively executing.
	InstanceStatusRunning InstanceStatus = "running"
	// InstanceStatusWaiting is parked awaiting a signal.
	InstanceStatusWaiting InstanceStatus = "waiting"
	// InstanceStatusCompleted finished successfully.
	InstanceStatusCompleted InstanceStatus = "completed"
	// InstanceStatusFailed terminated in failure.
	InstanceStatusFailed InstanceStatus = "failed"
	// InstanceStatusCancelled was cancelled.
	InstanceStatusCancelled InstanceStatus = "cancelled"
	// InstanceStatusCompensating is running rollback compensation.
	InstanceStatusCompensating InstanceStatus = "compensating"
	// InstanceStatusCompensated finished compensation.
	InstanceStatusCompensated InstanceStatus = "compensated"
	// InstanceStatusCompensationFailed is the terminal state when compensation
	// itself failed (rare but critical; always audited). Added by G1-amendment
	// ADJ-5 per Finalization B4 / Blueprint §8 (CONTRA-7 disposition).
	InstanceStatusCompensationFailed InstanceStatus = "compensation_failed"
)
