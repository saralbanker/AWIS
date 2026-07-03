package intelligence

import (
	"errors"
	"testing"

	"github.com/awis/awis/internal/core"
)

// TestDispatcherChainWalk: first adapter in the chain errors on Draft; second
// succeeds → the second's response is returned. The first adapter's draftCalls
// counter confirms it was attempted.
func TestDispatcherChainWalk(t *testing.T) {
	firstErr := errors.New("first adapter: transient error")
	wantResp := core.DraftResponse{
		Output: map[string]any{"result": "ok"},
		Usage:  core.Usage{Adapter: "second", Model: "fake", TokensUsed: 0},
	}

	first := &fakeAdapter{
		name:     "first",
		availFn:  func() bool { return true },
		caps:     []string{"draft"},
		draftErr: firstErr,
	}
	second := &fakeAdapter{
		name:      "second",
		availFn:   func() bool { return true },
		caps:      []string{"draft"},
		draftResp: wantResp,
	}

	router, err := NewRouter(
		[]Registration{
			{Adapter: first, Locality: LocalityCloud},
			{Adapter: second, Locality: LocalityCloud},
		},
		[]string{"first", "second"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	d := NewDispatcher(router)

	resp, err := d.Draft(bg(),
		core.IntelReq{Capability: "draft"},
		true,
		core.DraftRequest{Context: "ctx"},
	)
	if err != nil {
		t.Fatalf("Draft: unexpected error: %v", err)
	}
	if resp.Output["result"] != "ok" {
		t.Errorf("Draft.Output[result]: want %q, got %v", "ok", resp.Output["result"])
	}
	if first.draftCalls != 1 {
		t.Errorf("first.draftCalls: want 1, got %d", first.draftCalls)
	}
}

// TestDispatcherExhaustedRequired: all eligible adapters fail × required=true →
// errors.As finds CapabilityUnavailableError AND the last provider error is
// preserved via errors.Join (both inspectable from the returned error).
func TestDispatcherExhaustedRequired(t *testing.T) {
	lastCallErr := errors.New("provider: service unavailable")

	a := &fakeAdapter{
		name:     "a",
		availFn:  func() bool { return true },
		caps:     []string{"draft"},
		draftErr: lastCallErr,
	}
	router, err := NewRouter(
		[]Registration{{Adapter: a, Locality: LocalityCloud}},
		[]string{"a"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	d := NewDispatcher(router)

	_, err = d.Draft(bg(),
		core.IntelReq{Capability: "draft"},
		true,
		core.DraftRequest{Context: "ctx"},
	)
	if err == nil {
		t.Fatal("Draft: want error, got nil")
	}

	// CapabilityUnavailableError must be findable.
	var capErr CapabilityUnavailableError
	if !errors.As(err, &capErr) {
		t.Errorf("errors.As CapabilityUnavailableError: not found in %T: %v", err, err)
	}

	// Last provider error must be preserved (errors.Join wraps both).
	if !errors.Is(err, lastCallErr) {
		t.Errorf("errors.Is lastCallErr: not found in %T: %v", err, err)
	}
}

// TestDispatcherExhaustedFallback: all eligible adapters fail × required=false →
// errors.As finds FallbackSignal.
func TestDispatcherExhaustedFallback(t *testing.T) {
	a := &fakeAdapter{
		name:     "a",
		availFn:  func() bool { return true },
		caps:     []string{"draft"},
		draftErr: errors.New("adapter error"),
	}
	router, err := NewRouter(
		[]Registration{{Adapter: a, Locality: LocalityCloud}},
		[]string{"a"},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	d := NewDispatcher(router)

	_, err = d.Draft(bg(),
		core.IntelReq{Capability: "draft"},
		false,
		core.DraftRequest{Context: "ctx"},
	)
	if err == nil {
		t.Fatal("Draft: want error, got nil")
	}

	var sig FallbackSignal
	if !errors.As(err, &sig) {
		t.Errorf("errors.As FallbackSignal: not found in %T: %v", err, err)
	}
}
