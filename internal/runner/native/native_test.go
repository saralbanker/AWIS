package native

// NativeRunner outcome tests (Blueprint §5 L2 / §12). Every runner outcome is a
// StepResult or a typed *core.StepError. The timeout test uses real time (the
// runner's deadline is inherently time-driven); all other tests are synchronous.

import (
	"context"
	"errors"
	"reflect"
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
}

func (h *fakeHandler) ID() string { return h.id }

func (h *fakeHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	if h.sleep > 0 {
		time.Sleep(h.sleep)
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
