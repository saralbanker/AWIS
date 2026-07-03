package sdk

import "github.com/awis/awis/internal/core"

// Step is a single node in a workflow. Its serialized field names are frozen at
// Gate G1; Type selects the step kind and the type-specific config fields
// (WaitSignal for signal steps, Intelligence for intelligence steps).
type Step = core.Step

// StepType enumerates the kinds of step.
type StepType = core.StepType

// The frozen step-type values.
const (
	// StepTypeNative is a handler implemented in-process.
	StepTypeNative = core.StepTypeNative
	// StepTypeSubprocess is a handler run as an external subprocess.
	StepTypeSubprocess = core.StepTypeSubprocess
	// StepTypePlugin is a handler provided by a plugin.
	StepTypePlugin = core.StepTypePlugin
	// StepTypeIntelligence is an intelligence-capability step.
	StepTypeIntelligence = core.StepTypeIntelligence
	// StepTypeSignal is a step that waits for an external signal.
	StepTypeSignal = core.StepTypeSignal
)

// RetryPolicy configures retry behaviour for a step: Attempts (total, including
// the first), Backoff strategy, and the optional RetryableErrors codes.
type RetryPolicy = core.RetryPolicy

// WaitConfig configures a signal step: the awaited SignalName, an optional
// Timeout, and the TimeoutAction on expiry.
type WaitConfig = core.WaitConfig

// IntelReq configures an intelligence step: the requested Capability, an
// optional ModelHint, and the ContextBudget in tokens.
type IntelReq = core.IntelReq

// StepHandler is implemented by applications for native steps: ID returns the
// handler identifier and Execute runs the step, returning outputs or an error.
type StepHandler = core.StepHandler

// StepContext is the execution context passed to a StepHandler: instance/step
// identity, attempt ordinal, resolved inputs, the intelligence port (nil if
// none), a logger, and the execution deadline.
type StepContext = core.StepContext

// StepResult is the successful output of a StepHandler. Failures are signalled by
// returning a Go error from Execute, not via a result field.
type StepResult = core.StepResult

// Logger is the step-scoped logging interface handed to handlers. Its shape is
// completed at M06; it is not part of the G1 format freeze (sdk surface mutable
// until M08).
type Logger = core.Logger
