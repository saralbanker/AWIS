package subprocess

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/awis/awis/internal/core"
)

// echoStepScript returns the absolute path to the Python echo_step.py fixture.
func echoStepScript(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", "py", "echo_step.py")
}

// TestE2E_PythonStepOutputsFlow is the M11-C3 e2e keystone: a subprocess step
// implemented with awis_step (Python) produces outputs that flow to a second
// Run call (simulating the native step), demonstrating FR-SE-02 and FR-SDK-10.
//
// The test is skipped when python3 is absent from PATH.
func TestE2E_PythonStepOutputsFlow(t *testing.T) {
	// Skip when python3 is not available.
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found on PATH — skipping e2e Python keystone")
	}

	scriptPath := echoStepScript(t)

	// handler string: "python3 <abs_path>" — whitespace-split by runner into argv
	// (TDS-04 §5; runner calls exec.Command("python3", scriptPath)).
	// echo_step.py registers @step(id="python3 <abs_path>") to match this value.
	handler := "python3 " + scriptPath

	r := New()

	// -- Step 1: subprocess Python step --
	sc1 := core.StepContext{
		InstanceID: "e2e-inst-001",
		StepID:     "py-echo",
		Attempt:    1,
		Inputs:     map[string]any{"value": "hello-from-e2e", "num": float64(7)},
	}
	step1 := core.Step{
		ID:      "py-echo",
		Name:    "Python echo step",
		Type:    core.StepTypeSubprocess,
		Handler: core.HandlerRef(handler),
	}

	res1, se := r.Run(context.Background(), sc1, step1)
	if se != nil {
		t.Fatalf("Python subprocess step returned error: code=%q message=%q", se.Code, se.Message)
	}

	// Verify echo_step.py returned the inputs plus echo=true.
	if res1.Outputs["echo"] != true {
		t.Errorf("expected outputs.echo=true, got %v", res1.Outputs["echo"])
	}
	if res1.Outputs["value"] != "hello-from-e2e" {
		t.Errorf("expected outputs.value=hello-from-e2e, got %v", res1.Outputs["value"])
	}
	if res1.Outputs["num"] != float64(7) {
		t.Errorf("expected outputs.num=7, got %v", res1.Outputs["num"])
	}

	// -- Step 2: native-style step consuming the Python step's outputs --
	// Run a second subprocess step with the Python outputs as inputs, proving
	// values flow onward (FR-SE-02, FR-SDK-10).
	sc2 := core.StepContext{
		InstanceID: "e2e-inst-001",
		StepID:     "py-echo-2",
		Attempt:    1,
		Inputs:     res1.Outputs, // outputs from step 1 flow as inputs to step 2
	}
	step2 := core.Step{
		ID:      "py-echo-2",
		Name:    "Python echo step 2",
		Type:    core.StepTypeSubprocess,
		Handler: core.HandlerRef(handler),
	}

	res2, se := r.Run(context.Background(), sc2, step2)
	if se != nil {
		t.Fatalf("Second Python subprocess step returned error: code=%q message=%q", se.Code, se.Message)
	}

	// The output from step 1 (including echo=true) was passed as input to step 2
	// and echoed back, confirming that values flow between steps (FR-SE-02).
	if res2.Outputs["value"] != "hello-from-e2e" {
		t.Errorf("step 2: expected outputs.value=hello-from-e2e (propagated from step 1), got %v", res2.Outputs["value"])
	}
	if res2.Outputs["echo"] != true {
		t.Errorf("step 2: expected outputs.echo=true, got %v", res2.Outputs["echo"])
	}
}
