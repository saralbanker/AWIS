//go:build integration

package integration

import (
	"encoding/json"
	"testing"
	"time"
)

// TestB4_OnErrorRoutingReachesCompleted is the B-4 oracle.
//
// The bug: `completedSet` is derived from the instance's Variables keys. A
// step that fails TERMINALLY is removed from current_steps but never enters
// Variables, so the join gate saw its upstream as complete and re-nominated
// it on every tick, forever — the claim was already held, the step was
// skipped, and the completion check bailed out because the activatable set
// was non-empty. The instance stayed 'running' with nothing running.
//
// So the assertion that matters is TERMINALITY, not merely that the recovery
// step ran.
func TestB4_OnErrorRoutingReachesCompleted(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("with-on-error.yaml", withOnError)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "with-on-error")
	f.waitForStatus(sub.InstanceID, "completed", 20*time.Second)

	var trace traceOutputJSON
	f.runJSON(&trace, "trace", "--full", sub.InstanceID)

	// The failing step must be attempted exactly once. Under B-4 it was
	// re-nominated every tick, so this count is the direct regression guard
	// — a status assertion alone would not catch a re-activation that still
	// happened to terminate.
	if got := countStepStarted(trace.Events, "primary"); got != 1 {
		t.Errorf("step 'primary' started %d times, want exactly 1 (B-4 re-activation)", got)
	}
	if got := countStepStarted(trace.Events, "recover"); got != 1 {
		t.Errorf("step 'recover' started %d times, want exactly 1", got)
	}
	if findEvent(trace.Events, "StepFailed") == nil {
		t.Error("no StepFailed event — the on_error route was never actually exercised")
	}
}

// TestB4_FallbackRoutingReachesCompleted is the same guarantee on the
// fallback branch, which routeTerminalFailure handles before on_error
// (EDR-011 §8: fallback wins).
func TestB4_FallbackRoutingReachesCompleted(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("with-fallback.yaml", withFallback)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "with-fallback")
	f.waitForStatus(sub.InstanceID, "completed", 20*time.Second)

	var trace traceOutputJSON
	f.runJSON(&trace, "trace", "--full", sub.InstanceID)
	if got := countStepStarted(trace.Events, "primary"); got != 1 {
		t.Errorf("step 'primary' started %d times, want exactly 1 (B-4 re-activation)", got)
	}
	if findEvent(trace.Events, "StepFallbackActivated") == nil {
		t.Error("no StepFallbackActivated event — the fallback route was never exercised")
	}
}

// TestB4_TerminalFailureReachesFailed covers the third routeTerminalFailure
// branch: no fallback, no on_error, so the workflow itself must fail. This is
// the branch where a re-activation loop is most visible, because there is no
// recovery step whose completion could mask it.
func TestB4_TerminalFailureReachesFailed(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("terminal-fail.yaml", terminalFail)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "terminal-fail")
	f.waitForStatus(sub.InstanceID, "failed", 20*time.Second)

	var trace traceOutputJSON
	f.runJSON(&trace, "trace", "--full", sub.InstanceID)
	if got := countStepStarted(trace.Events, "doomed"); got != 1 {
		t.Errorf("step 'doomed' started %d times, want exactly 1 (B-4 re-activation)", got)
	}
	if findEvent(trace.Events, "WorkflowFailed") == nil {
		t.Errorf("no WorkflowFailed event: %v", eventTypes(trace.Events))
	}
}

// TestB3_HandlerPanicIsContainedAndRuntimeSurvives is B-3 through the
// SHIPPED BINARY. Before the native runner grew a recover(), a panicking
// handler killed the whole process and every other in-flight instance with
// it.
//
// Two things are asserted, and the second is the real one: the panic becomes
// a StepError with code handler_panic, and the runtime is still alive and
// still able to run a NEW instance to completion afterwards.
func TestB3_HandlerPanicIsContainedAndRuntimeSurvives(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("panic-step.yaml", panicStep)
	f.writeWorkflow("linear.yaml", linearNative)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "panic-step")
	f.waitForStatus(sub.InstanceID, "failed", 20*time.Second)

	var trace traceOutputJSON
	f.runJSON(&trace, "trace", "--full", sub.InstanceID)
	failed := findEvent(trace.Events, "StepFailed")
	if failed == nil {
		t.Fatalf("no StepFailed event: %v", eventTypes(trace.Events))
	}
	if code := stepFailedCode(t, failed); code != "handler_panic" {
		t.Errorf("StepFailed code = %q, want handler_panic", code)
	}

	// The containment guarantee: the process survived, and a subsequent
	// instance still runs to completion.
	var after submitOutput
	f.runJSON(&after, "submit", "linear-native", "--input", "name=Survivor")
	f.waitForStatus(after.InstanceID, "completed", 20*time.Second)
}

// TestB2_SlowHandlerTimesOutAndDoesNotWedgeRuntime is B-2 through the
// shipped binary. A handler with no bound used to block wg.Wait() in
// processInstance, freezing the tick loop for EVERY instance.
//
// The fixture sleeps 30s behind a 1s timeout, so a runtime that honours the
// deadline fails the step quickly, while one that blocks on the handler
// cannot complete the second instance within the window this test allows.
func TestB2_SlowHandlerTimesOutAndDoesNotWedgeRuntime(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("slow-step.yaml", slowStep)
	f.writeWorkflow("linear.yaml", linearNative)
	f.start()

	var slow submitOutput
	f.runJSON(&slow, "submit", "slow-step")

	// An unrelated instance submitted while the slow one is in flight must
	// still complete — this is the guarantee B-2 protects.
	var other submitOutput
	f.runJSON(&other, "submit", "linear-native", "--input", "name=Concurrent")
	f.waitForStatus(other.InstanceID, "completed", 20*time.Second)

	// And the slow step itself must be terminated by its own deadline rather
	// than running to its full 30s sleep.
	f.waitForStatus(slow.InstanceID, "failed", 20*time.Second)
}

// TestRetry_FlakyStepShowsRetryingThenLaterSuccess: a step that fails once
// under a retry: policy must emit StepFailed{retrying:true}, then be retried,
// then complete — and the instance must reach 'completed'.
func TestRetry_FlakyStepShowsRetryingThenLaterSuccess(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("retry-flaky.yaml", retryFlaky)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "retry-flaky")
	f.waitForStatus(sub.InstanceID, "completed", 30*time.Second)

	var trace traceOutputJSON
	f.runJSON(&trace, "trace", "--full", sub.InstanceID)
	if !hasRetryingStepFailed(trace.Events) {
		t.Fatalf("no StepFailed{retrying:true} in a workflow that must retry once: %v", eventTypes(trace.Events))
	}
	// succeed_on is 2, so exactly two attempts: one failed, one succeeded.
	if got := countStepStarted(trace.Events, "flaky"); got != 2 {
		t.Errorf("step 'flaky' started %d times, want exactly 2 (fail once, then succeed)", got)
	}
	if findEvent(trace.Events, "StepCompleted") == nil {
		t.Error("no StepCompleted event — the retry never succeeded")
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

// countStepStarted counts StepStarted events for one step id. This is the
// primary B-4 instrument: a re-activating step accumulates StepStarted
// events without bound.
func countStepStarted(events []traceEventJSON, stepID string) int {
	n := 0
	for _, ev := range events {
		if ev.EventType != "StepStarted" {
			continue
		}
		var p struct {
			StepID string `json:"step_id"`
		}
		if json.Unmarshal(ev.Payload, &p) == nil && p.StepID == stepID {
			n++
		}
	}
	return n
}

// stepFailedCode extracts error.code from a StepFailed payload (TDS-01 §2).
func stepFailedCode(t *testing.T, ev *traceEventJSON) string {
	t.Helper()
	var p struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		t.Fatalf("decode StepFailed payload: %v (payload=%s)", err, ev.Payload)
	}
	return p.Error.Code
}

// eventTypes lists the event types in order, for readable failure messages.
func eventTypes(events []traceEventJSON) []string {
	out := make([]string, len(events))
	for i, ev := range events {
		out[i] = ev.EventType
	}
	return out
}
