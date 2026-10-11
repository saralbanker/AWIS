package main

// trace_golden_test.go — output characterization for `awis trace` (M18-C4;
// NFR-P-05). The goldens pin the bytes of the human timeline (default and
// --full) and of --json for a fixture that reaches the human-format branches of
// formatTraceEvent. They were captured from the unbuffered renderer; any change
// to how the renderer writes must reproduce them exactly.

import (
	"testing"

	"github.com/awis/awis/internal/core"
)

// traceGoldenEvents returns one instance (nfr-i-000000) whose timeline reaches
// the human-format branches: StepStarted and StepCompleted with and without a
// step_id in the payload, StepFailed, StepFallbackActivated, SignalReceived and
// WorkflowCompleted. Timestamps are fixed by nfrEvent.
func traceGoldenEvents() []core.ExecutionEvent {
	return []core.ExecutionEvent{
		nfrEvent(0, 1, core.EventTypeWorkflowStarted, "",
			map[string]any{"inputs": map[string]any{}}),
		nfrEvent(0, 2, core.EventTypeStepStarted, "step-a",
			map[string]any{"step_id": "step-a", "attempt": 1, "inputs": map[string]any{}}),
		nfrEvent(0, 3, core.EventTypeStepCompleted, "step-a",
			map[string]any{"step_id": "step-a", "attempt": 1, "outputs": map[string]any{"r": 3}, "duration_ms": 1500}),
		nfrEvent(0, 4, core.EventTypeStepStarted, "step-b",
			map[string]any{"attempt": 1, "inputs": map[string]any{}}),
		nfrEvent(0, 5, core.EventTypeStepFailed, "step-b",
			map[string]any{"error": "boom"}),
		nfrEvent(0, 6, core.EventTypeStepFallbackActivated, "step-b",
			map[string]any{"fallback_step_id": "step-c"}),
		nfrEvent(0, 7, core.EventTypeStepStarted, "step-c",
			map[string]any{"step_id": "step-c", "attempt": 2, "inputs": map[string]any{}}),
		nfrEvent(0, 8, core.EventTypeStepCompleted, "step-c",
			map[string]any{"duration_ms": 250}),
		nfrEvent(0, 9, core.EventTypeSignalReceived, "wait",
			map[string]any{"signal_name": "approve"}),
		nfrEvent(0, 10, core.EventTypeWorkflowCompleted, "",
			map[string]any{"outputs": map[string]any{}, "duration_ms": 9000}),
	}
}

// TestTraceGoldenOutput pins `awis trace` human output (default and --full) and
// --json output for traceGoldenEvents against testdata/golden.
func TestTraceGoldenOutput(t *testing.T) {
	dir := t.TempDir()
	nfrBuildFixtureDB(t, dir, traceGoldenEvents())
	id := string(nfrInstanceID(0))

	nfrPinCLI(t, dir, false)
	checkGolden(t, "trace_golden_human.txt", nfrCapture(t, func() { runTrace([]string{id}) }))
	checkGolden(t, "trace_golden_full.txt", nfrCapture(t, func() { runTrace([]string{"--full", id}) }))

	nfrPinCLI(t, dir, true)
	checkGolden(t, "trace_golden_json.json", nfrCapture(t, func() { runTrace([]string{id}) }))
}
