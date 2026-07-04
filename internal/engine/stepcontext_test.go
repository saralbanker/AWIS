package engine

// assembleContext template-resolution tests (CONTRA-10 / FR-WD-05).

import (
	"bytes"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// newLoggingEngine builds an Engine whose logger writes to buf, for Warn capture.
func newLoggingEngine(t *testing.T, buf *bytes.Buffer) *Engine {
	t.Helper()
	s := openStorage(t)
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return New(s, nil, Config{Clock: func() time.Time { return storageBase }}, logger)
}

func TestAssembleContext_NestedResolution(t *testing.T) {
	var buf bytes.Buffer
	e := newLoggingEngine(t, &buf)

	inst := core.WorkflowInstance{
		InstanceID: "inst-1",
		Variables: map[string]any{
			"inputs": map[string]any{"foo": "bar"},
			"stepA":  map[string]any{"out": "val"},
		},
		CurrentSteps: []string{},
	}
	step := core.Step{
		ID:   "target",
		Type: core.StepTypeNative,
		Inputs: map[string]any{
			"a":      "{{workflow.inputs.foo}}",
			"nested": map[string]any{"b": "{{steps.stepA.outputs.out}}"},
			"list":   []any{"{{workflow.inputs.foo}}", "literal"},
		},
	}

	sc, serr := e.assembleContext(inst, step, 1)
	if serr != nil {
		t.Fatalf("assembleContext returned error: %+v", serr)
	}
	want := map[string]any{
		"a":      "bar",
		"nested": map[string]any{"b": "val"},
		"list":   []any{"bar", "literal"},
	}
	if !reflect.DeepEqual(sc.Inputs, want) {
		t.Fatalf("resolved inputs\n got=%#v\nwant=%#v", sc.Inputs, want)
	}
	if sc.Attempt != 1 {
		t.Fatalf("Attempt = %d, want 1", sc.Attempt)
	}
}

func TestAssembleContext_WarningOnNonCompletedStep(t *testing.T) {
	var buf bytes.Buffer
	e := newLoggingEngine(t, &buf)

	// stepB is running (not completed); reading its outputs yields ""+Warning.
	inst := core.WorkflowInstance{
		InstanceID:   "inst-1",
		Variables:    map[string]any{"inputs": map[string]any{}},
		CurrentSteps: []string{"stepB"},
	}
	step := core.Step{
		ID:     "target",
		Type:   core.StepTypeNative,
		Inputs: map[string]any{"x": "{{steps.stepB.outputs.y}}"},
	}

	sc, serr := e.assembleContext(inst, step, 1)
	if serr != nil {
		t.Fatalf("assembleContext returned error: %+v", serr)
	}
	if got := sc.Inputs["x"]; got != "" {
		t.Fatalf("x should resolve to empty string, got %q", got)
	}
	logged := buf.String()
	if !strings.Contains(logged, "template resolution warning") {
		t.Fatalf("expected a Warn line, got: %s", logged)
	}
	if !strings.Contains(logged, "steps.stepB.outputs.y") {
		t.Fatalf("Warn line must carry the offending path; got: %s", logged)
	}
}

func TestAssembleContext_ParseTemplateFailure(t *testing.T) {
	var buf bytes.Buffer
	e := newLoggingEngine(t, &buf)

	inst := core.WorkflowInstance{
		InstanceID:   "inst-1",
		Variables:    map[string]any{"inputs": map[string]any{}},
		CurrentSteps: []string{},
	}
	// "{{workflow}}" is a bare scope ⇒ ParseTemplate error ⇒ template_error.
	step := core.Step{
		ID:     "target",
		Type:   core.StepTypeNative,
		Inputs: map[string]any{"bad": "{{workflow}}"},
	}

	_, serr := e.assembleContext(inst, step, 1)
	if serr == nil {
		t.Fatalf("expected a *core.StepError for a parse failure")
	}
	if serr.Code != "template_error" {
		t.Fatalf("code = %q, want template_error", serr.Code)
	}
}
