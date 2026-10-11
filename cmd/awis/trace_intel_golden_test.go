package main

// trace_intel_golden_test.go — M18-C4r output characterization for the ADJ-8
// usage suffix on intelligence StepCompleted events (TDS-07 §4 timeline sample:
// `✓ StepCompleted ... 13.1s   adapter: anthropic, tokens: 847`). Native
// StepCompleted lines are pinned by trace_golden_test.go and must not change.

import (
	"fmt"
	"testing"

	"github.com/awis/awis/internal/core"
)

// traceIntelGoldenEvents returns one instance (nfr-i-000000) whose timeline holds
// an intelligence StepCompleted with the NullAdapter usage shape (adapter "null",
// tokens_used 0), an intelligence StepCompleted with adapter "anthropic" and
// tokens_used 847, and a native StepCompleted between them that must stay bare.
func traceIntelGoldenEvents() []core.ExecutionEvent {
	return []core.ExecutionEvent{
		nfrEvent(0, 1, core.EventTypeWorkflowStarted, "",
			map[string]any{"inputs": map[string]any{}}),
		nfrEvent(0, 2, core.EventTypeStepStarted, "step-a",
			map[string]any{"step_id": "step-a", "attempt": 1, "inputs": map[string]any{}}),
		nfrEvent(0, 3, core.EventTypeStepCompleted, "step-a",
			map[string]any{"step_id": "step-a", "attempt": 1, "outputs": map[string]any{"r": 3},
				"duration_ms": 1500, "adapter": "null", "model": "null", "tokens_used": 0}),
		nfrEvent(0, 4, core.EventTypeStepStarted, "step-b",
			map[string]any{"step_id": "step-b", "attempt": 1, "inputs": map[string]any{}}),
		nfrEvent(0, 5, core.EventTypeStepCompleted, "step-b",
			map[string]any{"step_id": "step-b", "attempt": 1, "outputs": map[string]any{"r": 5},
				"duration_ms": 13100, "adapter": "anthropic", "model": "test-model", "tokens_used": 847}),
		nfrEvent(0, 6, core.EventTypeStepStarted, "step-c",
			map[string]any{"step_id": "step-c", "attempt": 1, "inputs": map[string]any{}}),
		nfrEvent(0, 7, core.EventTypeStepCompleted, "step-c",
			map[string]any{"step_id": "step-c", "attempt": 1, "outputs": map[string]any{},
				"duration_ms": 250}),
		nfrEvent(0, 8, core.EventTypeWorkflowCompleted, "",
			map[string]any{"outputs": map[string]any{}, "duration_ms": 7000}),
	}
}

// TestTraceIntelGoldenOutput pins the human timeline for traceIntelGoldenEvents
// against testdata/golden/trace_golden_intel_human.txt.
func TestTraceIntelGoldenOutput(t *testing.T) {
	dir := t.TempDir()
	nfrBuildFixtureDB(t, dir, traceIntelGoldenEvents())
	id := string(nfrInstanceID(0))

	nfrPinCLI(t, dir, false)
	checkGolden(t, "trace_golden_intel_human.txt", nfrCapture(t, func() { runTrace([]string{id}) }))
}

// TestFormatTraceEventStepCompletedUsageSuffix checks the StepCompleted line
// directly: the suffix is appended only when the payload carries an adapter key,
// and a native payload produces the bare line.
func TestFormatTraceEventStepCompletedUsageSuffix(t *testing.T) {
	base := fmt.Sprintf("%-20s %-20s  %.1fs", "StepCompleted", "draft-entry", 13.1)
	cases := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "anthropic usage triple",
			payload: `{"step_id":"draft-entry","attempt":1,"outputs":{},"duration_ms":13100,"adapter":"anthropic","model":"m","tokens_used":847}`,
			want:    base + "   adapter: anthropic, tokens: 847",
		},
		{
			name:    "null adapter usage triple",
			payload: `{"step_id":"draft-entry","attempt":1,"outputs":{},"duration_ms":13100,"adapter":"null","model":"null","tokens_used":0}`,
			want:    base + "   adapter: null, tokens: 0",
		},
		{
			name:    "native payload has no suffix",
			payload: `{"step_id":"draft-entry","attempt":1,"outputs":{},"duration_ms":13100}`,
			want:    base,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev := core.ExecutionEvent{
				EventType: core.EventTypeStepCompleted,
				StepID:    "draft-entry",
				Payload:   []byte(tc.payload),
			}
			if got := formatTraceEvent(ev, false); got != tc.want {
				t.Errorf("formatTraceEvent:\nwant %q\ngot  %q", tc.want, got)
			}
		})
	}
}
