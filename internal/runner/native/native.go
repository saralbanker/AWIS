// Package native implements the NativeRunner: the engine Runner for
// type=native steps (Blueprint §5 L2, §12). It resolves a step's handler from an
// in-process registry, applies the step timeout as a context deadline, and runs
// the handler, mapping every outcome to a StepResult or a typed *core.StepError.
package native

import (
	"context"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// NativeRunner dispatches native steps to registered StepHandlers.
type NativeRunner struct {
	handlers map[string]core.StepHandler
}

// New returns an empty NativeRunner. Register handlers before use.
func New() *NativeRunner {
	return &NativeRunner{handlers: make(map[string]core.StepHandler)}
}

// Register adds h to the registry keyed by h.ID(). A later registration under
// the same id replaces the earlier one.
func (r *NativeRunner) Register(h core.StepHandler) {
	r.handlers[h.ID()] = h
}

// Run executes step's handler. Outcomes (all typed):
//
//   - handler not found            → StepError{code:"handler_not_found"}
//   - handler returns a Go error   → StepError{code:"handler_error", message}
//   - step.Timeout exceeded        → StepError{code:"timeout"}
//   - success                      → StepResult
//
// step.Timeout is applied as a context deadline when it parses to a positive
// duration. NOTE (C1 judgment call for CE review): core.Duration's serialized
// form is deferred to M10; C1 parses it with time.ParseDuration (the natural
// stdlib reading). An empty or unparseable Timeout means "no deadline".
func (r *NativeRunner) Run(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.StepError) {
	h, ok := r.handlers[string(step.Handler)]
	if !ok {
		return core.StepResult{}, &core.StepError{
			Code:    "handler_not_found",
			Message: fmt.Sprintf("no native handler registered for %q", step.Handler),
		}
	}

	if d := parseTimeout(step.Timeout); d > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
		if dl, ok := ctx.Deadline(); ok {
			sc.Deadline = dl
		}
	}

	// Run the handler in a goroutine so a deadline can pre-empt it. The buffered
	// channel ensures the goroutine never blocks on send even if we return early.
	type handlerResult struct {
		out core.StepResult
		err error
	}
	done := make(chan handlerResult, 1)
	go func() {
		out, err := h.Execute(sc)
		done <- handlerResult{out: out, err: err}
	}()

	select {
	case <-ctx.Done():
		return core.StepResult{}, &core.StepError{
			Code:    "timeout",
			Message: fmt.Sprintf("step %q exceeded its timeout: %v", step.ID, ctx.Err()),
		}
	case hr := <-done:
		if hr.err != nil {
			return core.StepResult{}, &core.StepError{
				Code:    "handler_error",
				Message: hr.err.Error(),
			}
		}
		return hr.out, nil
	}
}

// parseTimeout parses a core.Duration into a time.Duration. An empty or
// unparseable value yields 0 ("no deadline"); the M10 DSL owns the final form.
func parseTimeout(d core.Duration) time.Duration {
	s := string(d)
	if s == "" {
		return 0
	}
	parsed, err := time.ParseDuration(s)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}
