// Package intelligence implements the IntelligenceRunner: the engine Runner for
// type=intelligence steps (IMPLEMENTATION_SPEC T2; Blueprint §5 L2, §13, §17). It
// builds a capability request from the step's resolved inputs, dispatches through
// the intelligence seam's Dispatcher, and maps the outcome to a StepResult or a
// typed *core.StepError. Provider usage is reported to the engine via the
// UsageRunner side-channel (ADJ-8) so core.StepResult stays frozen.
package intelligence

import (
	"context"
	"errors"
	"fmt"

	"github.com/awis/awis/internal/core"
	intel "github.com/awis/awis/internal/intelligence"
)

// Runner is the engine Runner for intelligence steps. It holds the Dispatcher
// that owns the fallback chain (Blueprint §17).
type Runner struct {
	dispatcher *intel.Dispatcher
}

// New returns a Runner backed by dispatcher.
func New(dispatcher *intel.Dispatcher) *Runner {
	return &Runner{dispatcher: dispatcher}
}

// Run satisfies engine.Runner. It discards the usage side-channel (a caller that
// wants usage uses RunWithUsage).
func (r *Runner) Run(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.StepError) {
	out, _, serr := r.RunWithUsage(ctx, sc, step)
	return out, serr
}

// RunWithUsage satisfies engine.UsageRunner. It returns the provider usage (ADJ-8)
// alongside the result on success; usage is nil on any error path.
//
// Capability handling:
//   - "draft"      → DraftRequest{Context: stringified inputs["context"],
//     Schema: step.Outputs}; outputs = DraftResponse.Output.
//   - "synthesize" → SynthesisRequest{Query: inputs["query"], Entries:
//     inputs["entries"], MaxLen: inputs["max_len"] if present}; outputs =
//     {"text": SynthesisResponse.Text}.
//   - anything else (classify/embed/unknown) → StepError{capability_unknown} with
//     NO dispatch (FR-IL-10, CONTRA-3).
//
// Error mapping (errors.As against the seam's typed errors):
//   - FallbackSignal            → StepError{capability_fallback} (engine skips
//     retries and routes to the step fallback, FR-IL-06).
//   - CapabilityUnavailableError → StepError{capability_unavailable} (FR-IL-07).
//   - any other error            → StepError{intelligence_error}.
func (r *Runner) RunWithUsage(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.Usage, *core.StepError) {
	if step.Intelligence == nil {
		return core.StepResult{}, nil, &core.StepError{
			Code:    "intelligence_error",
			Message: fmt.Sprintf("intelligence step %q has no intelligence config", step.ID),
		}
	}
	req := *step.Intelligence

	switch req.Capability {
	case "draft":
		dr := core.DraftRequest{
			Context: stringify(sc.Inputs["context"]),
			Schema:  step.Outputs,
		}
		resp, err := r.dispatcher.Draft(ctx, req, req.Required, dr)
		if err != nil {
			return core.StepResult{}, nil, mapDispatchError(err)
		}
		usage := resp.Usage
		return core.StepResult{Outputs: resp.Output}, &usage, nil

	case "synthesize":
		sr := core.SynthesisRequest{
			Query:   stringify(sc.Inputs["query"]),
			Entries: entriesOf(sc.Inputs["entries"]),
		}
		if ml, ok := toInt(sc.Inputs["max_len"]); ok {
			sr.MaxLen = ml
		}
		resp, err := r.dispatcher.Synthesize(ctx, req, req.Required, sr)
		if err != nil {
			return core.StepResult{}, nil, mapDispatchError(err)
		}
		usage := resp.Usage
		return core.StepResult{Outputs: map[string]any{"text": resp.Text}}, &usage, nil

	default:
		// classify / embed / unknown ⇒ no dispatch (FR-IL-10, CONTRA-3).
		return core.StepResult{}, nil, &core.StepError{
			Code:    "capability_unknown",
			Message: fmt.Sprintf("capability %q is not dispatchable", req.Capability),
		}
	}
}

// mapDispatchError maps a Dispatcher error to a typed engine StepError. The
// Dispatcher joins its typed no-eligible error with the last provider error via
// errors.Join, so errors.As still finds the typed sentinel.
func mapDispatchError(err error) *core.StepError {
	var fb intel.FallbackSignal
	if errors.As(err, &fb) {
		return &core.StepError{Code: "capability_fallback", Message: err.Error()}
	}
	var un intel.CapabilityUnavailableError
	if errors.As(err, &un) {
		return &core.StepError{Code: "capability_unavailable", Message: err.Error()}
	}
	return &core.StepError{Code: "intelligence_error", Message: err.Error()}
}

// stringify renders an input value as a string: a string passes through; any
// other value is formatted with %v (the assembled context is a rendered template
// value, Blueprint §13). Nil yields "".
func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	default:
		return fmt.Sprintf("%v", x)
	}
}

// entriesOf coerces the "entries" input to []any (the synthesis source list,
// Blueprint §13). A non-slice value yields an empty slice.
func entriesOf(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

// toInt coerces a numeric input (JSON numbers decode to float64) to an int.
func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	default:
		return 0, false
	}
}
