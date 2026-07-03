package sdk

import "github.com/awis/awis/internal/core"

// WorkflowDefinition is the platform's stable data structure: both the YAML DSL
// and the Go SDK produce instances of it, and the runtime operates exclusively
// on it. Field names and required/optional markers are frozen at Gate G1;
// a registered (id, version) pair is immutable and any change needs a new
// version. SchemaVersion is the serialization-format version (value 1);
// Version is the workflow's own immutable SemVer.
type WorkflowDefinition = core.WorkflowDefinition

// Transition is a conditional edge between two steps. An absent Condition means
// the transition is unconditional; OnError fires it on step failure.
type Transition = core.Transition

// WorkflowInstance is a running instance of a WorkflowDefinition; it mirrors the
// StateStore projection of the EventLog.
type WorkflowInstance = core.WorkflowInstance

// InstanceStatus enumerates the lifecycle states of a workflow instance.
type InstanceStatus = core.InstanceStatus

// The frozen workflow-instance lifecycle states.
const (
	// InstanceStatusPending is created but not yet running.
	InstanceStatusPending = core.InstanceStatusPending
	// InstanceStatusRunning is actively executing.
	InstanceStatusRunning = core.InstanceStatusRunning
	// InstanceStatusWaiting is parked awaiting a signal.
	InstanceStatusWaiting = core.InstanceStatusWaiting
	// InstanceStatusCompleted finished successfully.
	InstanceStatusCompleted = core.InstanceStatusCompleted
	// InstanceStatusFailed terminated in failure.
	InstanceStatusFailed = core.InstanceStatusFailed
	// InstanceStatusCancelled was cancelled.
	InstanceStatusCancelled = core.InstanceStatusCancelled
	// InstanceStatusCompensating is running rollback compensation.
	InstanceStatusCompensating = core.InstanceStatusCompensating
	// InstanceStatusCompensated finished compensation.
	InstanceStatusCompensated = core.InstanceStatusCompensated
)

// CompensationPlan is the ordered rollback plan run when a workflow fails after
// one or more steps have completed. Its shape is completed at M06; it is not
// part of the G1 format freeze (sdk surface mutable until M08).
type CompensationPlan = core.CompensationPlan

// CompensationRef is a step's undo action, invoked if the workflow fails after
// the step completes. Its shape is completed at M06; it is not part of the G1
// format freeze (sdk surface mutable until M08).
type CompensationRef = core.CompensationRef
