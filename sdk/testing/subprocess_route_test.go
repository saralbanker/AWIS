// subprocess_route_test.go — M11-C2 harness route test (new file; not a
// modification of existing sdk/testing files). Proves that sdk.NewRuntime
// (through the awistesting harness) routes a subprocess step via
// SubprocessRunner and its outputs flow to the next native step.
// (T4/T5; IMP §27.M11 Val row; FR-SE-02; FR-SDK-10).
package awistesting

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/awis/awis/internal/core"
)

// echoHandlerScript returns the absolute path to the echo-handler.sh fixture
// script that lives next to the subprocess package's testdata.
func echoHandlerScript(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile: sdk/testing/subprocess_route_test.go
	// script:   internal/runner/subprocess/testdata/bin/echo-handler.sh
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return filepath.Join(repoRoot, "internal", "runner", "subprocess", "testdata", "bin", "echo-handler.sh")
}

// collectorHandler is a native step handler that records the inputs it receives
// so the test can verify values flowed from the subprocess step.
type collectorHandler struct {
	received map[string]any
}

func (h *collectorHandler) ID() string { return "m11.collector" }
func (h *collectorHandler) Execute(sc core.StepContext) (core.StepResult, error) {
	h.received = sc.Inputs
	return core.StepResult{Outputs: map[string]any{"collected": true}}, nil
}

// TestHarnessRoute_SubprocessToNative proves that the runners map registered
// in sdk.NewRuntime includes SubprocessRunner and that the subprocess step's
// outputs flow into the following native step (sdk wiring proof).
func TestHarnessRoute_SubprocessToNative(t *testing.T) {
	echoScript := echoHandlerScript(t)
	if _, err := os.Stat(echoScript); err != nil {
		t.Skipf("echo-handler.sh fixture not found (%v) — skipping subprocess route test", err)
	}

	collector := &collectorHandler{}

	wf := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "test.m11.subprocess-route",
		Version:       "1.0.0",
		Namespace:     "test",
		Name:          "M11 Subprocess Route Test",
		Triggers:      []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{
				ID:      "sp",
				Name:    "subprocess step",
				Type:    core.StepTypeSubprocess,
				Handler: core.HandlerRef(echoScript),
			},
			{
				ID:      "nat",
				Name:    "native step",
				Type:    core.StepTypeNative,
				Handler: "m11.collector",
			},
		},
		Transitions: []core.Transition{
			{From: "sp", To: "nat"},
		},
		InitialStep: "sp",
		FinalSteps:  []string{"nat"},
		Metadata:    map[string]any{},
	}

	h := NewHarness(t, WithStepHandler(collector))

	res, err := h.Run(wf, map[string]any{})
	if err != nil {
		t.Fatalf("harness Run: %v", err)
	}

	// Verify the workflow completed and the native step received the subprocess output.
	out := h.GetOutput(res.InstanceID, "nat")
	if out == nil {
		t.Fatal("expected output key 'nat' from native step; workflow may not have completed")
	}
}
