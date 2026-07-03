package core

// WorkflowDefinition is the platform's stable data structure: both the YAML DSL
// and the Go SDK produce instances of it, and the runtime operates exclusively
// on it (Blueprint §6; TDS-02). Field names, types, and required/optional
// markers are frozen at Gate G1 (WORKFLOW_SCHEMA.md). A registered
// (id, version) pair is immutable; any change requires a new version.
type WorkflowDefinition struct {
	// SchemaVersion is the version of THIS serialization format; value 1 under
	// TDS-02 (G1 ADJ-3). Distinct from Version, the workflow's own SemVer.
	SchemaVersion int `json:"schema_version"`
	// ID is the namespaced identifier, e.g. "oip.capture-decision".
	ID string `json:"id"`
	// Version is the workflow's own semantic version; immutable once registered.
	Version SemVer `json:"version"`
	// Namespace is the owning namespace, e.g. "oip".
	Namespace string `json:"namespace"`
	// Name is a human-readable label.
	Name string `json:"name"`
	// Description is an optional human-readable description.
	Description string `json:"description,omitempty"`
	// Triggers enumerate what starts this workflow.
	Triggers []Trigger `json:"triggers"`
	// Steps are all nodes of the workflow.
	Steps []Step `json:"steps"`
	// Transitions are the conditional edges between steps.
	Transitions []Transition `json:"transitions"`
	// InitialStep is the id of the step that runs first.
	InitialStep string `json:"initial_step"`
	// FinalSteps are the ids of steps that end the workflow.
	FinalSteps []string `json:"final_steps"`
	// Compensation is the optional ordered rollback plan run on failure.
	Compensation *CompensationPlan `json:"compensation,omitempty"`
	// Timeout is the optional maximum total workflow duration.
	Timeout Duration `json:"timeout,omitempty"`
	// Metadata is application-defined data.
	Metadata map[string]any `json:"metadata"`
}

// Transition is a conditional edge between two steps (Blueprint §6; TDS-02 §3).
type Transition struct {
	// From is the source step id.
	From string `json:"from"`
	// To is the destination step id.
	To string `json:"to"`
	// Condition is an optional expression over step outputs; absent means the
	// transition is unconditional.
	Condition Condition `json:"condition,omitempty"`
	// OnError, when true, fires this transition on step failure (not success).
	OnError bool `json:"on_error,omitempty"`
}

// Trigger declares one way a workflow may be started (Blueprint §6; TDS-02 §4).
type Trigger struct {
	// Type is the trigger kind: manual, schedule, event, or webhook.
	Type TriggerType `json:"type"`
	// Config is type-specific configuration.
	Config map[string]any `json:"config"`
}

// TriggerType enumerates the kinds of workflow trigger (Blueprint §6; TDS-02 §4).
type TriggerType string

// TriggerType frozen literal values (G1).
const (
	// TriggerTypeManual starts the workflow by explicit submission.
	TriggerTypeManual TriggerType = "manual"
	// TriggerTypeSchedule starts the workflow on a schedule.
	TriggerTypeSchedule TriggerType = "schedule"
	// TriggerTypeEvent starts the workflow on a domain event.
	TriggerTypeEvent TriggerType = "event"
	// TriggerTypeWebhook starts the workflow on an inbound webhook.
	TriggerTypeWebhook TriggerType = "webhook"
)

// CompensationPlan is the ordered rollback plan run when a workflow fails after
// one or more steps have completed (Blueprint §6).
//
// Shape completed at M06 (owning milestone); not part of the G1 format freeze
// (IMP §13 — sdk surface mutable until M08).
type CompensationPlan struct{}
