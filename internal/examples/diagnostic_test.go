package examples

import (
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// handlerByID looks up a built-in handler, failing the test if it is absent.
// The diagnostic ids are referenced by name from the binary-level integration
// fixtures (test/integration/fixtures.go), so a rename here must break a test
// here rather than only showing up as a handler_not_found at run time.
func handlerByID(t *testing.T, id string) core.StepHandler {
	t.Helper()
	for _, h := range Handlers() {
		if h.ID() == id {
			return h
		}
	}
	t.Fatalf("no built-in handler with id %q", id)
	return nil
}

// TestDiagFailAlwaysFails: the routing diagnostics are worthless if the
// handler they point at can ever succeed.
func TestDiagFailAlwaysFails(t *testing.T) {
	h := handlerByID(t, "examples.diagnostic.fail")
	for attempt := 1; attempt <= 5; attempt++ {
		_, err := h.Execute(core.StepContext{StepID: "s", Attempt: attempt})
		if err == nil {
			t.Fatalf("attempt %d: want an error, got nil", attempt)
		}
	}
}

// TestDiagFailUsesCustomMessage: the `message` input has to reach the
// StepFailed payload, otherwise a test cannot tell two failing steps apart.
func TestDiagFailUsesCustomMessage(t *testing.T) {
	h := handlerByID(t, "examples.diagnostic.fail")
	_, err := h.Execute(core.StepContext{
		StepID:  "primary",
		Attempt: 2,
		Inputs:  map[string]any{"message": "boom-marker"},
	})
	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !strings.Contains(err.Error(), "boom-marker") {
		t.Errorf("error %q does not carry the supplied message", err)
	}
	if !strings.Contains(err.Error(), "primary") {
		t.Errorf("error %q does not name the step", err)
	}
}

// TestDiagFlakyFailsThenSucceeds pins the exact attempt boundary: everything
// before succeed_on fails, everything from it onward succeeds. A retry test
// that asserts "eventually completed" would still pass if this boundary were
// off by one, so it is checked directly.
func TestDiagFlakyFailsThenSucceeds(t *testing.T) {
	h := handlerByID(t, "examples.diagnostic.flaky")
	const succeedOn = 3

	for attempt := 1; attempt < succeedOn; attempt++ {
		if _, err := h.Execute(core.StepContext{
			StepID: "flaky", Attempt: attempt,
			Inputs: map[string]any{"succeed_on": succeedOn},
		}); err == nil {
			t.Errorf("attempt %d: want failure (succeed_on=%d), got success", attempt, succeedOn)
		}
	}
	for attempt := succeedOn; attempt <= succeedOn+2; attempt++ {
		out, err := h.Execute(core.StepContext{
			StepID: "flaky", Attempt: attempt,
			Inputs: map[string]any{"succeed_on": succeedOn},
		})
		if err != nil {
			t.Errorf("attempt %d: want success (succeed_on=%d), got %v", attempt, succeedOn, err)
			continue
		}
		if out.Outputs["attempt"] != attempt {
			t.Errorf("attempt %d: outputs[attempt] = %v", attempt, out.Outputs["attempt"])
		}
	}
}

// TestDiagFlakyDefaultsToSucceedOnTwo: a fixture that omits succeed_on must
// still fail exactly once, which is what makes "no inputs" a usable retry
// fixture.
func TestDiagFlakyDefaultsToSucceedOnTwo(t *testing.T) {
	h := handlerByID(t, "examples.diagnostic.flaky")
	if _, err := h.Execute(core.StepContext{StepID: "f", Attempt: 1}); err == nil {
		t.Error("attempt 1 with no succeed_on: want failure, got success")
	}
	if _, err := h.Execute(core.StepContext{StepID: "f", Attempt: 2}); err != nil {
		t.Errorf("attempt 2 with no succeed_on: want success, got %v", err)
	}
}

// TestDiagFlakyIsStateless is the property the restart tests depend on:
// the same attempt number must always produce the same outcome. A flaky
// handler backed by a process-local counter would pass a naive retry test
// and then silently invalidate every restart test, because after a restart
// its counter would be back at zero.
func TestDiagFlakyIsStateless(t *testing.T) {
	h := handlerByID(t, "examples.diagnostic.flaky")
	for round := 0; round < 3; round++ {
		if _, err := h.Execute(core.StepContext{StepID: "f", Attempt: 1}); err == nil {
			t.Fatalf("round %d: attempt 1 succeeded — handler is carrying state across calls", round)
		}
		if _, err := h.Execute(core.StepContext{StepID: "f", Attempt: 2}); err != nil {
			t.Fatalf("round %d: attempt 2 failed — handler is carrying state across calls", round)
		}
	}
}

// TestDiagPanicPanics: the handler must actually panic, or the panic
// containment path it exists to exercise is never reached and the
// containment test silently proves nothing.
func TestDiagPanicPanics(t *testing.T) {
	h := handlerByID(t, "examples.diagnostic.panic")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("examples.diagnostic.panic did not panic")
		}
	}()
	_, _ = h.Execute(core.StepContext{StepID: "boom", Attempt: 1})
}

// TestDiagSlowSleepsAndIsBounded checks both halves of the contract: it
// really does block for the requested duration (so it can hold a step
// `running`), and it refuses an unbounded one (so a misconfigured workflow
// cannot park a goroutine the runner can no longer interrupt).
func TestDiagSlowSleepsAndIsBounded(t *testing.T) {
	h := handlerByID(t, "examples.diagnostic.slow")

	start := time.Now()
	out, err := h.Execute(core.StepContext{
		StepID: "slow", Attempt: 1,
		Inputs: map[string]any{"duration_ms": 120},
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if elapsed < 100*time.Millisecond {
		t.Errorf("returned after %s, want at least ~120ms — it is not actually sleeping", elapsed)
	}
	if got := out.Outputs["slept_ms"]; got != int64(120) {
		t.Errorf("slept_ms = %v (%T), want int64(120)", got, got)
	}

}

// TestClampSleepBounds asserts the cap directly. It deliberately does NOT go
// through Execute: diagSlow sleeps for whatever clampSleep returns, so
// observing the cap through Execute would mean the test itself sleeping for
// the full cap.
func TestClampSleepBounds(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want time.Duration
	}{
		{"negative clamps to zero", -5 * time.Second, 0},
		{"zero stays zero", 0, 0},
		{"in range passes through", 250 * time.Millisecond, 250 * time.Millisecond},
		{"at the cap passes through", maxDiagnosticSleep, maxDiagnosticSleep},
		{"above the cap clamps down", 999999 * time.Second, maxDiagnosticSleep},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clampSleep(tc.in); got != tc.want {
				t.Errorf("clampSleep(%s) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

// TestAsIntAcceptsEveryInputEncoding covers the coercion that makes a
// diagnostic behave identically no matter how its input reached it: a YAML
// literal arrives as int, anything that round-tripped through JSON (a
// --input flag, an EventLog payload) arrives as float64, and a template that
// interpolated into a string arrives as a string.
func TestAsIntAcceptsEveryInputEncoding(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want int
	}{
		{"int", 3, 3},
		{"int64", int64(4), 4},
		{"float64 from JSON", float64(5), 5},
		{"string from template", "6", 6},
		{"nil falls back", nil, 99},
		{"unparseable string falls back", "not-a-number", 99},
		{"wrong type falls back", []string{"7"}, 99},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := asInt(tc.in, 99); got != tc.want {
				t.Errorf("asInt(%#v, 99) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestDiagSlowCapIsNotAbsurd guards the constant itself: the cap has to stay
// far above any timeout an integration test would plausibly set, or a
// timeout test would start passing for the wrong reason (the handler
// returning early rather than the deadline firing).
func TestDiagSlowCapIsNotAbsurd(t *testing.T) {
	if maxDiagnosticSleep < 30*time.Second {
		t.Errorf("maxDiagnosticSleep = %s, too low to outlast a timeout fixture", maxDiagnosticSleep)
	}
}
