package core

import (
	"context"
	"time"
)

// StoragePort is the persistence boundary for the runtime: EventLog, StateStore,
// WorkflowRegistry, and StepResultCache (Blueprint §20). The method set is frozen
// verbatim; adapters (SQLite, Postgres) implement it identically.
type StoragePort interface {
	// AppendEvent appends one event to the EventLog.
	AppendEvent(ctx context.Context, event ExecutionEvent) error
	// ReadEvents reads events for an instance from a sequence number onward.
	ReadEvents(ctx context.Context, instanceID InstanceID, fromSeq int) ([]ExecutionEvent, error)
	// ReadEventRange reads events for a namespace within a time range.
	ReadEventRange(ctx context.Context, namespace string, from, to time.Time) ([]ExecutionEvent, error)

	// UpsertInstance writes an instance using optimistic concurrency control.
	UpsertInstance(ctx context.Context, instance WorkflowInstance, expectedVersion int) error
	// GetInstance fetches an instance by id.
	GetInstance(ctx context.Context, instanceID InstanceID) (WorkflowInstance, error)
	// ListInstances lists instances matching a filter.
	ListInstances(ctx context.Context, filter InstanceFilter) ([]WorkflowInstance, error)
	// ClaimStep atomically claims a step for a worker; reports whether it won.
	ClaimStep(ctx context.Context, instanceID InstanceID, stepID string, workerID string) (bool, error)

	// RegisterWorkflow registers a workflow definition.
	RegisterWorkflow(ctx context.Context, def WorkflowDefinition) error
	// GetWorkflow fetches a definition by id and version.
	GetWorkflow(ctx context.Context, id string, version SemVer) (WorkflowDefinition, error)
	// ListWorkflows lists the definitions in a namespace.
	ListWorkflows(ctx context.Context, namespace string) ([]WorkflowDefinition, error)

	// CacheResult caches a step result under an idempotency key with a TTL.
	CacheResult(ctx context.Context, key IdempotencyKey, result StepResult, ttl time.Duration) error
	// GetCachedResult fetches a cached step result; reports whether it was found.
	GetCachedResult(ctx context.Context, key IdempotencyKey) (StepResult, bool, error)
}

// IntelligencePort is the translation layer between step declarations and
// provider implementations (Blueprint §13). If the layer is absent the runtime
// routes intelligence steps to their fallbacks and continues executing.
type IntelligencePort interface {
	// Draft produces structured output from an assembled context.
	Draft(ctx context.Context, req DraftRequest) (DraftResponse, error)
	// Embed returns an embedding vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
	// Synthesize composes a synthesized answer over supplied entries.
	Synthesize(ctx context.Context, req SynthesisRequest) (SynthesisResponse, error)
	// Classify assigns text to one of the supplied categories.
	//
	// FR-IL-10: this is a non-callable placeholder in the current surface; it is
	// declared for interface completeness and is not invoked by the runtime yet.
	Classify(ctx context.Context, text string, categories []string) (Classification, error)

	// IsAvailable reports whether the provider is currently reachable.
	IsAvailable() bool
	// Capabilities lists the capabilities this provider declares.
	Capabilities() []Capability
	// ProviderName returns the provider's stable name.
	ProviderName() string
}

// DraftRequest is the input to IntelligencePort.Draft (Blueprint §13).
type DraftRequest struct {
	// Context is the assembled context, bounded by the step's context budget.
	Context string
	// Schema is the expected output structure.
	Schema map[string]any
	// Persona is an optional role for the model to assume.
	Persona string
	// Examples are optional few-shot examples.
	Examples []Example
}

// SynthesisRequest is the input to IntelligencePort.Synthesize (Blueprint §13).
type SynthesisRequest struct {
	// Query is the synthesis question.
	Query string
	// Entries are the source entries to synthesize over.
	Entries []any
	// MaxLen is the maximum output length.
	MaxLen int
}

// Classification is the result of IntelligencePort.Classify (Blueprint §13).
type Classification struct {
	// Category is the chosen category.
	Category string
	// Confidence is the model's confidence in [0,1].
	Confidence float32
	// Reasoning is an optional explanation.
	Reasoning string
}

// Usage records provider-reported consumption for a single intelligence call.
// It feeds the FR-IL-09 StepCompleted payload {adapter, model, tokens_used}
// assembled at M06. Shape completed at M04.
type Usage struct {
	// Adapter is the ProviderName() of the adapter that served the request.
	Adapter string
	// Model is the model identifier reported by the adapter ("null" for NullAdapter).
	Model string
	// TokensUsed is the number of tokens consumed as reported by the provider;
	// 0 for adapters that do not bill tokens (e.g. NullAdapter).
	TokensUsed int
}

// DraftResponse is the result of IntelligencePort.Draft (Blueprint §13).
// Output conforms to DraftRequest.Schema. Shape completed at M04.
type DraftResponse struct {
	// Output is the structured result conforming to the request Schema.
	Output map[string]any
	// Usage records adapter and token consumption for this call.
	Usage Usage
}

// SynthesisResponse is the result of IntelligencePort.Synthesize (Blueprint §13).
// Shape completed at M04.
type SynthesisResponse struct {
	// Text is the composed synthesized answer.
	Text string
	// Usage records adapter and token consumption for this call.
	Usage Usage
}

// Capability describes an intelligence capability a provider declares
// (Blueprint §13). Providers announce capabilities by name: "draft", "embed",
// "synthesize", "classify". Shape completed at M04.
type Capability struct {
	// Name is the capability identifier (e.g. "draft", "embed", "synthesize", "classify").
	Name string
}

// Example is a few-shot example supplied to a draft request (Blueprint §13).
// Shape completed at M04.
type Example struct {
	// Input is the example input, keyed by field name.
	Input map[string]any
	// Output is the example output, keyed by field name.
	Output map[string]any
}

// WorkflowRunner is the application-facing runtime control surface for workflow
// instances (Blueprint §12).
type WorkflowRunner interface {
	// Submit starts a new workflow instance and returns its id.
	Submit(ctx context.Context, definitionID string, inputs map[string]any) (InstanceID, error)
	// Signal delivers a signal to a waiting instance.
	Signal(ctx context.Context, instanceID InstanceID, signalName string, payload map[string]any) error
	// Status returns the current state of an instance.
	Status(ctx context.Context, instanceID InstanceID) (WorkflowStatus, error)
	// Cancel requests cancellation of a running instance.
	Cancel(ctx context.Context, instanceID InstanceID, reason string) error
	// List returns instances matching a filter.
	List(ctx context.Context, filter InstanceFilter) ([]WorkflowStatus, error)
}

// RecallAPI is the application-facing read surface over execution history
// (Blueprint §12).
type RecallAPI interface {
	// QueryHistory returns execution records matching a query.
	QueryHistory(ctx context.Context, query HistoryQuery) ([]ExecutionRecord, error)
	// ReplayInstance replays a completed instance for debugging.
	ReplayInstance(ctx context.Context, instanceID InstanceID) (ReplayTrace, error)
	// StepStats returns aggregate statistics for a step across all instances.
	StepStats(ctx context.Context, definitionID, stepID string) (StepStatistics, error)
}

// WorkflowStatus is the summarized current state of an instance returned by the
// runner (Blueprint §12).
//
// Shape completed at M08 (owning milestone); not part of the G1 format freeze
// (IMP §13 — sdk surface mutable until M08).
type WorkflowStatus struct {
	// InstanceID is the unique identity of the instance.
	InstanceID InstanceID
	// DefinitionID is the id of the workflow definition.
	DefinitionID string
	// Version is the SemVer of the workflow definition.
	Version SemVer
	// Status is the current lifecycle status.
	Status InstanceStatus
	// CurrentSteps are the active (running/waiting) step ids.
	CurrentSteps []string
	// Inputs are the workflow inputs (Variables snapshot).
	Inputs map[string]any
	// Outputs are the workflow outputs (Variables snapshot).
	Outputs map[string]any
	// CreatedAt is when the instance started.
	CreatedAt time.Time
	// UpdatedAt is when the instance was last updated.
	UpdatedAt time.Time
}

// InstanceFilter selects instances for List/ListInstances (Blueprint §12/§20).
//
// First-consumer shape completed at M03 (StateStore/ListInstances). Fields are
// additive and default to no-predicate when zero; the full shape is mutable
// until M08 when the sdk surface freezes (ADJ-4b).
type InstanceFilter struct {
	// Namespace restricts results to the given namespace. Empty = no predicate.
	Namespace string
	// Status restricts results to the given lifecycle status. Empty = no predicate.
	Status InstanceStatus
}

// HistoryQuery selects execution history for RecallAPI.QueryHistory
// (Blueprint §12).
//
// Shape completed at M08 (owning milestone); not part of the G1 format freeze
// (IMP §13 — sdk surface mutable until M08).
type HistoryQuery struct {
	// Namespace restricts results to the given namespace.
	Namespace string
	// DefinitionID restricts results to the given definition; empty = all.
	DefinitionID string
	// Status restricts results to the given lifecycle status; zero = all.
	Status InstanceStatus
}

// ExecutionRecord is a summarized history record returned by the RecallAPI
// (Blueprint §12).
//
// Shape completed at M08 (owning milestone); not part of the G1 format freeze
// (IMP §13 — sdk surface mutable until M08).
type ExecutionRecord struct {
	// InstanceID is the unique identity of the instance.
	InstanceID InstanceID
	// DefinitionID is the id of the workflow definition.
	DefinitionID string
	// Version is the SemVer of the workflow definition.
	Version SemVer
	// Status is the lifecycle status.
	Status InstanceStatus
	// StartedAt is when the instance started.
	StartedAt time.Time
	// CompletedAt is when the instance completed; nil if still running.
	CompletedAt *time.Time
	// Inputs are the workflow inputs.
	Inputs map[string]any
	// Outputs are the workflow outputs.
	Outputs map[string]any
}

// ReplayTrace is the trace produced by RecallAPI.ReplayInstance (Blueprint §12).
//
// Shape completed at M08 (owning milestone); not part of the G1 format freeze
// (IMP §13 — sdk surface mutable until M08).
type ReplayTrace struct {
	// InstanceID is the unique identity of the replayed instance.
	InstanceID InstanceID
	// DefinitionID is the id of the workflow definition.
	DefinitionID string
	// Version is the SemVer of the workflow definition.
	Version SemVer
	// Namespace is the owning namespace.
	Namespace string
	// Status is the instance's lifecycle status as recorded in storage.
	Status InstanceStatus
	// StartedAt is when the instance started.
	StartedAt time.Time
	// CompletedAt is when the instance completed; nil if still active.
	CompletedAt *time.Time
	// Events is the full ordered EventLog for the instance, from sequence 0.
	Events []ExecutionEvent
}

// StepStatistics are aggregate statistics for a step (Blueprint §12).
//
// Shape completed at M08 (owning milestone); not part of the G1 format freeze
// (IMP §13 — sdk surface mutable until M08).
type StepStatistics struct {
	// DefinitionID is the id of the workflow definition.
	DefinitionID string
	// StepID is the id of the step.
	StepID string
	// TotalRuns is the total number of times this step has been dispatched.
	TotalRuns int
	// SuccessCount is the number of successful runs.
	SuccessCount int
	// FailureCount is the number of failed runs.
	FailureCount int
	// AvgDurationMs is the average execution duration in milliseconds.
	AvgDurationMs float64
}
