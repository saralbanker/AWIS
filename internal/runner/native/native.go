// Package native implements the NativeRunner: the engine Runner for
// type=native steps (Blueprint §5 L2, §12). It resolves a step's handler from an
// in-process registry, applies the step timeout (or a configured default) as a
// context deadline, recovers handler panics, and runs the handler, mapping
// every outcome to a StepResult or a typed *core.StepError.
package native

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/awis/awis/internal/core"
)

// defaultProductionTimeout is the deadline New() applies to a step when
// step.Timeout is absent, empty, or unparseable. 30s is chosen as a
// conservative outer bound: generous enough not to abort legitimate native
// handlers (DB/HTTP calls, in-process computation) under normal load, but
// small enough that a stuck handler can never block a tick indefinitely.
// This matters because engine.tick's processInstance does wg.Wait() over all
// dispatched steps in a tick; without SOME deadline here, one hung handler
// blocks that wg.Wait() forever, which stalls the engine's Run loop and
// freezes every other in-flight instance too (Defect B-2). NewWithConfig lets
// callers (notably tests) inject a different value.
const defaultProductionTimeout = 30 * time.Second

// maxPanicStackBytes bounds the stack trace recorded in StepError.Details.
// StepError is carried inside the StepFailed payload, which is appended to the
// APPEND-ONLY EventLog and can never be rewritten (TDS-01 §1). An unbounded
// goroutine stack — easily tens of KB under deep call chains — would be
// persisted forever on every panic. The head of the trace is where the panic
// site lives, so truncating the tail preserves the diagnostic value.
const maxPanicStackBytes = 4096

// Config configures a NativeRunner.
type Config struct {
	// DefaultTimeout is the deadline applied to a step when step.Timeout does
	// not itself resolve to a positive duration (absent, empty, or
	// unparseable — see parseTimeout). Zero means no default is applied in
	// that case, i.e. the pre-fix "no deadline" behaviour; production code
	// should not do this (see New), but it lets tests exercise the
	// no-deadline path explicitly if ever needed.
	DefaultTimeout time.Duration
}

// NativeRunner dispatches native steps to registered StepHandlers.
type NativeRunner struct {
	handlers       map[string]core.StepHandler
	defaultTimeout time.Duration
}

// New returns an empty NativeRunner configured with the production default
// timeout (defaultProductionTimeout, 30s). Register handlers before use.
func New() *NativeRunner {
	return NewWithConfig(Config{DefaultTimeout: defaultProductionTimeout})
}

// NewWithConfig returns an empty NativeRunner using cfg (see Config). Tests
// use this to inject a short default timeout so timeout-path tests don't
// wait on the production 30s default. Register handlers before use.
func NewWithConfig(cfg Config) *NativeRunner {
	return &NativeRunner{
		handlers:       make(map[string]core.StepHandler),
		defaultTimeout: cfg.DefaultTimeout,
	}
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
//   - handler panics               → StepError{code:"handler_panic", message, details.stack}
//   - step.Timeout exceeded        → StepError{code:"timeout"}
//   - success                      → StepResult
//
// step.Timeout is applied as a context deadline when it parses to a positive
// duration (NOTE, C1 judgment call for CE review: core.Duration's serialized
// form is deferred to M10; C1 parses it with time.ParseDuration, the natural
// stdlib reading). An empty or unparseable Timeout no longer means "no
// deadline" (Defect B-2): it falls back to r.defaultTimeout, so every native
// step dispatched through this runner always has SOME bound unless the
// runner was explicitly constructed with Config.DefaultTimeout == 0.
func (r *NativeRunner) Run(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.StepError) {
	h, ok := r.handlers[string(step.Handler)]
	if !ok {
		return core.StepResult{}, &core.StepError{
			Code:    "handler_not_found",
			Message: fmt.Sprintf("no native handler registered for %q", step.Handler),
		}
	}

	d := parseTimeout(step.Timeout)
	if d <= 0 {
		// step.Timeout was absent, empty, unparseable, or non-positive: an
		// explicit positive step.Timeout always wins over the default: this
		// branch is only reached when it didn't resolve to one.
		d = r.defaultTimeout
	}
	if d > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
		if dl, ok := ctx.Deadline(); ok {
			sc.Deadline = dl
		}
	}

	// Run the handler in a goroutine so a deadline can pre-empt it. The buffered
	// channel ensures the goroutine never blocks on send even if we return early
	// on ctx.Done(). The deferred recover converts a handler panic (Defect B-3)
	// into a result sent on the same channel instead of crashing the process.
	//
	// NOTE: on timeout, the handler goroutine itself is NOT and cannot be
	// force-killed by Go — it keeps running (and may eventually send into the
	// buffered channel, where the send succeeds but the result is discarded,
	// since nothing is left reading it) after Run has already returned. The
	// guarantee added here is that the ENGINE's tick is never blocked past d;
	// it is not that the orphaned goroutine is reclaimed.
	type handlerResult struct {
		out      core.StepResult
		err      error
		panicked bool
		panicVal any
		stack    []byte
	}
	done := make(chan handlerResult, 1)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				done <- handlerResult{panicked: true, panicVal: rec, stack: truncateStack(debug.Stack())}
			}
		}()
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
		if hr.panicked {
			return core.StepResult{}, &core.StepError{
				Code:    "handler_panic",
				Message: fmt.Sprintf("step %q handler panicked: %v", step.ID, hr.panicVal),
				Details: map[string]any{"stack": string(hr.stack)},
			}
		}
		if hr.err != nil {
			return core.StepResult{}, &core.StepError{
				Code:    "handler_error",
				Message: hr.err.Error(),
			}
		}
		return hr.out, nil
	}
}

// truncateStack bounds a captured stack trace to maxPanicStackBytes, keeping
// the head (the panic site) and marking the elision.
func truncateStack(b []byte) []byte {
	if len(b) <= maxPanicStackBytes {
		return b
	}
	return append(b[:maxPanicStackBytes:maxPanicStackBytes], []byte("\n... stack truncated ...")...)
}

// parseTimeout parses a core.Duration into a time.Duration. An empty,
// unparseable, or non-positive value yields 0; Run interprets 0 as "fall back
// to the runner's configured default timeout" rather than "no deadline" (see
// Run's doc comment). The M10 DSL owns the final serialized form.
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
