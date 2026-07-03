package core

import "time"

// Step is a single node in a workflow (Blueprint §6; TDS-02 §2).
type Step struct {
	// ID is unique within the workflow.
	ID string `json:"id"`
	// Name is a human-readable label.
	Name string `json:"name"`
	// Type is the step kind: native, subprocess, plugin, intelligence, or signal.
	Type StepType `json:"type"`
	// Handler identifies the step's handler ("handler-name" for native,
	// "script.py" for subprocess, etc.).
	Handler HandlerRef `json:"handler"`
	// Inputs is the JSON Schema for the step's expected inputs.
	Inputs InputSchema `json:"inputs"`
	// Outputs is the JSON Schema for the step's produced outputs.
	Outputs OutputSchema `json:"outputs"`
	// Retry is the optional retry policy.
	Retry *RetryPolicy `json:"retry,omitempty"`
	// Timeout is the optional per-step timeout.
	Timeout Duration `json:"timeout,omitempty"`
	// Fallback is the optional step id to run if this step fails or its
	// capability is unavailable.
	Fallback string `json:"fallback,omitempty"`
	// Compensation is the optional undo action if the workflow fails after this
	// step completes.
	Compensation *CompensationRef `json:"compensation,omitempty"`
	// WaitSignal configures a type=signal step (signal name, timeout,
	// timeout_action).
	WaitSignal *WaitConfig `json:"wait_signal,omitempty"`
	// Intelligence configures a type=intelligence step (capability, model_hint,
	// context_budget).
	Intelligence *IntelReq `json:"intelligence,omitempty"`
}

// StepType enumerates the kinds of step (Blueprint §6; TDS-02 §2).
type StepType string

// StepType frozen literal values (G1).
const (
	// StepTypeNative is a handler implemented in-process.
	StepTypeNative StepType = "native"
	// StepTypeSubprocess is a handler run as an external subprocess.
	StepTypeSubprocess StepType = "subprocess"
	// StepTypePlugin is a handler provided by a plugin.
	StepTypePlugin StepType = "plugin"
	// StepTypeIntelligence is an intelligence-capability step.
	StepTypeIntelligence StepType = "intelligence"
	// StepTypeSignal is a step that waits for an external signal.
	StepTypeSignal StepType = "signal"
)

// RetryPolicy configures retry behaviour for a step (Blueprint §8, lines
// 521–528; full shape adopted by G1-amendment ADJ-6, founder-approved
// 2026-07-03 — CONTRA-8 disposition, TDS-02 §7).
type RetryPolicy struct {
	// Attempts is the maximum total number of attempts (including the first).
	Attempts int `json:"attempts"`
	// Backoff is the backoff strategy: immediate, linear, or exponential.
	Backoff string `json:"backoff"`
	// InitialDelay is the delay before the first retry; absent defaults to 1s
	// (ADJ-6; §16 CloudRetryPolicy default pattern).
	InitialDelay Duration `json:"initial_delay,omitempty"`
	// MaxDelay caps the backoff delay; absent defaults to 30s (ADJ-6).
	MaxDelay Duration `json:"max_delay,omitempty"`
	// RetryableErrors lists error codes eligible for retry; empty means retry all
	// transient errors.
	RetryableErrors []string `json:"retryable_errors,omitempty"`
}

// WaitConfig configures a type=signal step (Blueprint §6, line 291:
// "signal name + timeout + timeout_action").
type WaitConfig struct {
	// SignalName is the name of the awaited signal.
	SignalName string `json:"signal_name"`
	// Timeout is the optional maximum wait duration.
	Timeout Duration `json:"timeout,omitempty"`
	// TimeoutAction is the action on timeout (e.g. fail, compensate, continue).
	TimeoutAction string `json:"timeout_action"`
}

// IntelReq configures a type=intelligence step (Blueprint §6, line 292 +
// §13 capability declaration; `required` added by G1-amendment ADJ-7,
// founder-approved 2026-07-03 — CONTRA-9 disposition, TDS-02 §7).
type IntelReq struct {
	// Capability is the requested intelligence capability, e.g. "draft".
	Capability string `json:"capability"`
	// ModelHint is a routing hint: fast, quality, or local.
	ModelHint string `json:"model_hint,omitempty"`
	// ContextBudget is the maximum tokens for the assembled context.
	ContextBudget int `json:"context_budget,omitempty"`
	// Required controls the no-capable-provider outcome: true fails the step
	// with CapabilityUnavailableError (FR-IL-07); false (default) routes to the
	// step's fallback (FR-IL-06).
	Required bool `json:"required,omitempty"`
}

// CompensationRef is a step's undo action, invoked if the workflow fails after
// the step completes (Blueprint §6).
//
// Shape completed at M06 (owning milestone); not part of the G1 format freeze
// (IMP §13 — sdk surface mutable until M08).
type CompensationRef struct{}

// StepHandler is implemented by applications for native steps (Blueprint §12).
type StepHandler interface {
	// ID is the handler identifier used in WorkflowDefinition step.handler.
	ID() string
	// Execute runs the step; it returns outputs or an error.
	Execute(ctx StepContext) (StepResult, error)
}

// StepContext is the execution context passed to a StepHandler (Blueprint §12).
type StepContext struct {
	// InstanceID is the id of the owning workflow instance.
	InstanceID string
	// StepID is the id of the executing step.
	StepID string
	// Attempt is the 1-based attempt ordinal.
	Attempt int
	// Inputs are the resolved inputs handed to the step.
	Inputs map[string]any
	// Intelligence is the configured intelligence port; nil if none configured.
	Intelligence IntelligencePort
	// Logger is the step-scoped logger.
	Logger Logger
	// Deadline is the step's execution deadline.
	Deadline time.Time
}

// StepResult is the successful output of a StepHandler (Blueprint §12). Failures
// are signalled by returning a Go error from Execute, not via a result field.
type StepResult struct {
	// Outputs are the values the step produced.
	Outputs map[string]any
}

// Logger is the step-scoped logging interface handed to handlers.
//
// Shape completed at M06 (owning milestone); not part of the G1 format freeze
// (IMP §13 — sdk surface mutable until M08).
type Logger interface{}
