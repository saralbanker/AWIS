package native

// NativeRunner outcome tests (Blueprint §5 L2 / §12). Every runner outcome is a
// StepResult or a typed *core.StepError. The timeout test uses real time (the
// runner's deadline is inherently time-driven); all other tests are synchronous.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// fakeHandler is a configurable core.StepHandler for the runner tests.
type fakeHandler struct {
	id      string
	outputs map[string]any
	err     error
	sleep   time.Duration
	// block, if non-nil, is read from (and so blocks Execute) until the test
	// closes it. Used to model a handler that never returns on its own
	// (Defect B-2 tests).
	block chan struct{}
	// panicVal, if non-nil, is panicked with instead of returning (Defect B-3
	// tests). It may be any value, not just an error.
	panicVal any
}

func (h *fakeHandler) ID() string { return h.id }

func (h *fakeHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	if h.sleep > 0 {
		time.Sleep(h.sleep)
	}
	if h.block != nil {
		<-h.block
	}
	if h.panicVal != nil {
		panic(h.panicVal)
	}
	if h.err != nil {
		return core.StepResult{}, h.err
	}
	return core.StepResult{Outputs: h.outputs}, nil
}

func TestNativeRun_HandlerNotFound(t *testing.T) {
	r := New()
	step := core.Step{ID: "s", Type: core.StepTypeNative, Handler: "missing"}
	_, serr := r.Run(context.Background(), core.StepContext{}, step)
	if serr == nil || serr.Code != "handler_not_found" {
		t.Fatalf("want handler_not_found, got %+v", serr)
	}
}

func TestNativeRun_HandlerError(t *testing.T) {
	r := New()
	r.Register(&fakeHandler{id: "h", err: errors.New("kaboom")})
	step := core.Step{ID: "s", Type: core.StepTypeNative, Handler: "h"}
	_, serr := r.Run(context.Background(), core.StepContext{}, step)
	if serr == nil || serr.Code != "handler_error" {
		t.Fatalf("want handler_error, got %+v", serr)
	}
	if serr.Message == "" || serr.Message != "kaboom" {
		t.Fatalf("handler_error must carry the handler message; got %q", serr.Message)
	}
}

func TestNativeRun_Timeout(t *testing.T) {
	r := New()
	// Handler sleeps well past the 50ms step timeout.
	r.Register(&fakeHandler{id: "slow", sleep: 500 * time.Millisecond})
	step := core.Step{ID: "s", Type: core.StepTypeNative, Handler: "slow", Timeout: "50ms"}
	_, serr := r.Run(context.Background(), core.StepContext{}, step)
	if serr == nil || serr.Code != "timeout" {
		t.Fatalf("want timeout, got %+v", serr)
	}
}

func TestNativeRun_Success(t *testing.T) {
	r := New()
	out := map[string]any{"k": "v"}
	r.Register(&fakeHandler{id: "h", outputs: out})
	step := core.Step{ID: "s", Type: core.StepTypeNative, Handler: "h"}
	res, serr := r.Run(context.Background(), core.StepContext{}, step)
	if serr != nil {
		t.Fatalf("unexpected error: %+v", serr)
	}
	if !reflect.DeepEqual(res.Outputs, out) {
		t.Fatalf("outputs = %#v, want %#v", res.Outputs, out)
	}
}

func TestNativeRun_NoDeadlineWhenTimeoutEmptyOrUnparseable(t *testing.T) {
	for _, tc := range []struct {
		name    string
		timeout core.Duration
	}{
		{"empty", ""},
		{"unparseable", "not-a-duration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := New()
			// Handler sleeps briefly; with no deadline it must run to completion.
			r.Register(&fakeHandler{id: "h", outputs: map[string]any{"ok": true}, sleep: 20 * time.Millisecond})
			step := core.Step{ID: "s", Type: core.StepTypeNative, Handler: "h", Timeout: tc.timeout}
			res, serr := r.Run(context.Background(), core.StepContext{}, step)
			if serr != nil {
				t.Fatalf("unexpected error with %s timeout: %+v", tc.name, serr)
			}
			if res.Outputs["ok"] != true {
				t.Fatalf("handler should have completed; outputs=%#v", res.Outputs)
			}
		})
	}
}

// -- Defect B-2: a step with no (or no usable) timeout must not block the
// runner forever; it must fall back to the runner's configured default. --

// TestNativeRun_NoTimeoutFieldFallsBackToDefaultAndTimesOut is card test 1: a
// handler that blocks until the test releases it, on a step with NO
// `timeout:` field at all, must still be bounded by the runner's default.
// The default is injected short (50ms) so the assertion runs fast instead of
// waiting on the real 30s production default.
func TestNativeRun_NoTimeoutFieldFallsBackToDefaultAndTimesOut(t *testing.T) {
	block := make(chan struct{})
	defer close(block) // release the orphaned handler goroutine so it can finish

	const shortDefault = 50 * time.Millisecond
	r := NewWithConfig(Config{DefaultTimeout: shortDefault})
	r.Register(&fakeHandler{id: "h", block: block})
	step := core.Step{ID: "s", Type: core.StepTypeNative, Handler: "h"} // no Timeout set

	start := time.Now()
	_, serr := r.Run(context.Background(), core.StepContext{}, step)
	elapsed := time.Since(start)

	if serr == nil || serr.Code != "timeout" {
		t.Fatalf("want timeout, got %+v", serr)
	}
	if elapsed > time.Second {
		t.Fatalf("Run should have returned within the injected %v default, took %v", shortDefault, elapsed)
	}
}

// TestNativeRun_ExplicitTimeoutShorterThanDefaultWins is card test 4: an
// explicit positive step.Timeout must win over the runner's default even
// when the default is much longer.
func TestNativeRun_ExplicitTimeoutShorterThanDefaultWins(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	r := NewWithConfig(Config{DefaultTimeout: 5 * time.Second}) // long default
	r.Register(&fakeHandler{id: "h", block: block})
	step := core.Step{ID: "s", Type: core.StepTypeNative, Handler: "h", Timeout: "50ms"} // short explicit

	start := time.Now()
	_, serr := r.Run(context.Background(), core.StepContext{}, step)
	elapsed := time.Since(start)

	if serr == nil || serr.Code != "timeout" {
		t.Fatalf("want timeout, got %+v", serr)
	}
	if elapsed >= time.Second {
		t.Fatalf("explicit 50ms step.Timeout should have won over the 5s default; took %v", elapsed)
	}
}

// TestNativeRun_UnparseableExplicitTimeoutFallsBackToDefault is card test 5:
// an explicit but unparseable step.Timeout must fall back to the runner's
// default (bounded), not mean "no deadline".
func TestNativeRun_UnparseableExplicitTimeoutFallsBackToDefault(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	const shortDefault = 50 * time.Millisecond
	r := NewWithConfig(Config{DefaultTimeout: shortDefault})
	r.Register(&fakeHandler{id: "h", block: block})
	step := core.Step{ID: "s", Type: core.StepTypeNative, Handler: "h", Timeout: "not-a-duration"}

	start := time.Now()
	_, serr := r.Run(context.Background(), core.StepContext{}, step)
	elapsed := time.Since(start)

	if serr == nil || serr.Code != "timeout" {
		t.Fatalf("unparseable step.Timeout must fall back to the runner's default and still time out; got %+v", serr)
	}
	if elapsed > time.Second {
		t.Fatalf("Run should have returned within the injected %v default, took %v", shortDefault, elapsed)
	}
}

// -- Defect B-3: a handler panic must be recovered and turned into a typed
// *core.StepError, never crash the process. --

// TestNativeRun_HandlerPanic is card tests 2 and 3: a handler panic (with an
// error value, a plain string, or any other non-error value) is recovered
// and reported as code:"handler_panic" — and, since this test function
// itself returns normally, it demonstrates the panic did not crash the test
// process.
func TestNativeRun_HandlerPanic(t *testing.T) {
	for _, tc := range []struct {
		name     string
		panicVal any
	}{
		{"error value", errors.New("kaboom")},
		{"string value", "boom"},
		{"int value", 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := New()
			r.Register(&fakeHandler{id: "h", panicVal: tc.panicVal})
			step := core.Step{ID: "s", Type: core.StepTypeNative, Handler: "h"}

			res, serr := r.Run(context.Background(), core.StepContext{}, step)
			if serr == nil || serr.Code != "handler_panic" {
				t.Fatalf("want handler_panic, got res=%+v serr=%+v", res, serr)
			}
			wantSubstr := fmt.Sprintf("%v", tc.panicVal)
			if !strings.Contains(serr.Message, wantSubstr) {
				t.Fatalf("handler_panic message %q must include recovered value %q", serr.Message, wantSubstr)
			}
			if serr.Details == nil || serr.Details["stack"] == nil {
				t.Fatalf("handler_panic must carry a stack trace in Details[\"stack\"]; got %+v", serr.Details)
			}
			if stack, ok := serr.Details["stack"].(string); !ok || stack == "" {
				t.Fatalf("handler_panic Details[\"stack\"] must be a non-empty string; got %#v", serr.Details["stack"])
			}
		})
	}
}
