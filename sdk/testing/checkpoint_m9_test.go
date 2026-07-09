// IMP §20.M9 checkpoint test: an OIP-shaped workflow (multi-step capture-like
// flow with the plugin step stubbed as a native handler) runs in a unit test
// in < 1s, zero external dependencies (in-memory SQLite only).
package awistesting

import (
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// oipCaptureHandlers are the three native handlers for the OIP checkpoint workflow.
// The "analyze" step represents what would be a plugin step in production;
// it is stubbed here as a native handler (IMP §20.M9 "plugin step stubbed").
type captureH struct{}
type analyzeH struct{}
type storeH struct{}

func (h *captureH) ID() string { return "oip.checkpoint.capture" }
func (h *captureH) Execute(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{"captured": true, "payload": "data"}}, nil
}

func (h *analyzeH) ID() string { return "oip.checkpoint.analyze" }
func (h *analyzeH) Execute(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{"analyzed": true, "tag": "classified"}}, nil
}

func (h *storeH) ID() string { return "oip.checkpoint.store" }
func (h *storeH) Execute(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{"stored": true}}, nil
}

// oipCaptureWorkflow defines the OIP-shaped three-step capture workflow.
func oipCaptureWorkflow() *core.WorkflowDefinition {
	return &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "oip.checkpoint.capture-flow",
		Version:       "1.0.0",
		Namespace:     "oip",
		Name:          "OIP Capture (M9 checkpoint)",
		Triggers:      []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{ID: "capture", Name: "capture", Type: core.StepTypeNative, Handler: "oip.checkpoint.capture"},
			{ID: "analyze", Name: "analyze", Type: core.StepTypeNative, Handler: "oip.checkpoint.analyze"},
			{ID: "store", Name: "store", Type: core.StepTypeNative, Handler: "oip.checkpoint.store"},
		},
		Transitions: []core.Transition{
			{From: "capture", To: "analyze"},
			{From: "analyze", To: "store"},
		},
		InitialStep: "capture",
		FinalSteps:  []string{"store"},
		Metadata:    map[string]any{"checkpoint": "M9"},
	}
}

func TestCheckpointM9_OIPCaptureWorkflow(t *testing.T) {
	start := time.Now()

	h := NewHarness(t,
		WithStepHandler(&captureH{}),
		WithStepHandler(&analyzeH{}),
		WithStepHandler(&storeH{}),
	)

	res, err := h.Run(oipCaptureWorkflow(), map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("M9 checkpoint Run: %v", err)
	}

	if st, _ := h.rt.Status(h.ctx, res.InstanceID); st.Status != core.InstanceStatusCompleted {
		t.Fatalf("M9 checkpoint: status = %q, want completed", st.Status)
	}

	stored := h.GetOutput(res.InstanceID, "store")
	if stored == nil {
		t.Fatal("M9 checkpoint: expected output key 'store' from the final step")
	}

	elapsed := time.Since(start)
	if elapsed >= time.Second {
		t.Fatalf("M9 checkpoint wall clock %v ≥ 1s (must complete in < 1s)", elapsed)
	}
}
