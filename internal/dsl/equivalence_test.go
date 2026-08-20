package dsl_test

// equivalence_test.go — FR-WD-02 keystone: YAML tier ≡ Builder tier (M10-C3; T7/T8).
//
// Two sub-tests:
//   1. KEYSTONE (deep-equal):  ParseFile(keystone.yaml) and sdk.WorkflowBuilder must produce
//      reflect.DeepEqual *core.WorkflowDefinition values (IMP §19 L324).
//   2. HARNESS ORACLE (event stream):  both definitions run on separate awistesting.Harness
//      instances and emit identical event-type/step-id sequences (T8; M09 HANDOFF).

import (
	"reflect"
	"testing"

	awistesting "github.com/awis/awis/sdk/testing"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/dsl"
	sdk "github.com/awis/awis/sdk"
)

// keystoneBuilderDef constructs the same workflow as testdata/keystone.yaml via
// sdk.WorkflowBuilder.  Every field value must match what the YAML parser produces
// so that reflect.DeepEqual passes (FR-WD-02 keystone).
func keystoneBuilderDef(t *testing.T) *core.WorkflowDefinition {
	t.Helper()
	b := sdk.NewWorkflowBuilder("keystone.round-trip", "1.0.0").
		SetName("Keystone Round-Trip").
		SetDescription("Non-trivial workflow for FR-WD-02 equivalence test (M10-C3; T7)").
		SetNamespace("keystone")

	b.AddTrigger(core.Trigger{
		Type:   core.TriggerTypeManual,
		Config: map[string]any{},
	})

	b.AddStep(core.Step{
		ID:      "fetch",
		Name:    "Fetch Data",
		Type:    core.StepTypeNative,
		Handler: "keystone.fetch",
		Inputs:  map[string]any{"source": "{{workflow.inputs.source}}"},
		Outputs: map[string]any{"data": map[string]any{"type": "object"}},
		Retry:   &core.RetryPolicy{Attempts: 3, Backoff: "exponential"},
	})
	b.AddStep(core.Step{
		ID:      "enrich",
		Name:    "Enrich",
		Type:    core.StepTypeNative,
		Handler: "keystone.enrich",
		Outputs: map[string]any{"enriched": map[string]any{"type": "object"}},
	})
	b.AddStep(core.Step{
		ID:      "cache",
		Name:    "Cache",
		Type:    core.StepTypeNative,
		Handler: "keystone.cache",
		Outputs: map[string]any{"cached": map[string]any{"type": "boolean"}},
	})
	b.AddStep(core.Step{
		ID:   "await-approval",
		Name: "Await Approval",
		Type: core.StepTypeSignal,
		WaitSignal: &core.WaitConfig{
			SignalName:    "approved",
			Timeout:       core.Duration("1h"),
			TimeoutAction: "fail",
		},
	})
	b.AddStep(core.Step{
		ID:      "publish",
		Name:    "Publish",
		Type:    core.StepTypeNative,
		Handler: "keystone.publish",
		Outputs: map[string]any{"published": map[string]any{"type": "boolean"}},
	})

	b.SetInitialStep("fetch")
	b.AddTransition(core.Transition{From: "fetch", To: "enrich"})
	b.AddTransition(core.Transition{From: "fetch", To: "cache"})
	b.AddTransition(core.Transition{From: "enrich", To: "await-approval"})
	b.AddTransition(core.Transition{From: "cache", To: "await-approval"})
	b.AddTransition(core.Transition{
		From:      "await-approval",
		To:        "publish",
		Condition: core.Condition("steps.await-approval.status != 'failed'"),
	})
	b.AddFinalStep("publish")

	def, err := b.Build()
	if err != nil {
		t.Fatalf("keystoneBuilderDef: Build: %v", err)
	}
	return def
}

// TestKeystoneDeepEqual is the FR-WD-02 keystone (IMP §19 L324; T7):
// the YAML-parsed and Builder-constructed WorkflowDefinitions must be
// reflect.DeepEqual.
func TestKeystoneDeepEqual(t *testing.T) {
	yamlDef, err := dsl.ParseFile("testdata/keystone.yaml")
	if err != nil {
		t.Fatalf("ParseFile(keystone.yaml): %v", err)
	}

	builderDef := keystoneBuilderDef(t)

	if !reflect.DeepEqual(yamlDef, builderDef) {
		t.Errorf("YAML-parsed and Builder-constructed WorkflowDefinitions are not deep-equal")
		t.Logf("YAML:    %+v", yamlDef)
		t.Logf("Builder: %+v", builderDef)
		// Field-by-field diff for diagnostics.
		if yamlDef.SchemaVersion != builderDef.SchemaVersion {
			t.Logf("  SchemaVersion: YAML=%d Builder=%d", yamlDef.SchemaVersion, builderDef.SchemaVersion)
		}
		if yamlDef.ID != builderDef.ID {
			t.Logf("  ID: YAML=%q Builder=%q", yamlDef.ID, builderDef.ID)
		}
		if yamlDef.Version != builderDef.Version {
			t.Logf("  Version: YAML=%q Builder=%q", yamlDef.Version, builderDef.Version)
		}
		if yamlDef.Namespace != builderDef.Namespace {
			t.Logf("  Namespace: YAML=%q Builder=%q", yamlDef.Namespace, builderDef.Namespace)
		}
		if yamlDef.Name != builderDef.Name {
			t.Logf("  Name: YAML=%q Builder=%q", yamlDef.Name, builderDef.Name)
		}
		if yamlDef.Description != builderDef.Description {
			t.Logf("  Description: YAML=%q Builder=%q", yamlDef.Description, builderDef.Description)
		}
		if !reflect.DeepEqual(yamlDef.Triggers, builderDef.Triggers) {
			t.Logf("  Triggers: YAML=%+v Builder=%+v", yamlDef.Triggers, builderDef.Triggers)
		}
		for i, s := range yamlDef.Steps {
			if i < len(builderDef.Steps) && !reflect.DeepEqual(s, builderDef.Steps[i]) {
				t.Logf("  Steps[%d] YAML=%+v Builder=%+v", i, s, builderDef.Steps[i])
			}
		}
		if !reflect.DeepEqual(yamlDef.Transitions, builderDef.Transitions) {
			t.Logf("  Transitions: YAML=%+v Builder=%+v", yamlDef.Transitions, builderDef.Transitions)
		}
		if yamlDef.InitialStep != builderDef.InitialStep {
			t.Logf("  InitialStep: YAML=%q Builder=%q", yamlDef.InitialStep, builderDef.InitialStep)
		}
		if !reflect.DeepEqual(yamlDef.FinalSteps, builderDef.FinalSteps) {
			t.Logf("  FinalSteps: YAML=%v Builder=%v", yamlDef.FinalSteps, builderDef.FinalSteps)
		}
		if !reflect.DeepEqual(yamlDef.Metadata, builderDef.Metadata) {
			t.Logf("  Metadata: YAML=%v Builder=%v", yamlDef.Metadata, builderDef.Metadata)
		}
	}
}

// keystoneHandler is a deterministic stub handler for the keystone harness oracle.
type keystoneHandler struct{ id string }

func (h *keystoneHandler) ID() string { return h.id }
func (h *keystoneHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{"ok": true}}, nil
}

// keystoneHandlers returns the four native-step handlers the keystone workflow
// needs. All succeed immediately and return a minimal output.
func keystoneHandlers() []core.StepHandler {
	return []core.StepHandler{
		&keystoneHandler{"keystone.fetch"},
		&keystoneHandler{"keystone.enrich"},
		&keystoneHandler{"keystone.cache"},
		&keystoneHandler{"keystone.publish"},
	}
}

// eventKey is the (event_type, step_id) pair used for sequence comparison.
type eventKey struct {
	EventType string
	StepID    string
}

// runKeystoneHarness submits def to a fresh harness, parks at the signal wait,
// delivers the "approved" signal, ticks to completion, and returns the event
// stream as []eventKey (type + step_id, ordered by sequence_num).
func runKeystoneHarness(t *testing.T, def *core.WorkflowDefinition) []eventKey {
	t.Helper()

	opts := make([]awistesting.Option, 0, len(keystoneHandlers()))
	for _, h := range keystoneHandlers() {
		opts = append(opts, awistesting.WithStepHandler(h))
	}
	h := awistesting.NewHarness(t, opts...)

	// Run until the workflow parks at the signal WAIT step.
	res, err := h.Run(def, map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("harness.Run: %v", err)
	}

	// Deliver the "approved" signal to resume and complete the workflow.
	h.Signal(res.InstanceID, "approved", map[string]any{})

	events, err := h.ReadEvents(res.InstanceID)
	if err != nil {
		t.Fatalf("harness.ReadEvents: %v", err)
	}

	keys := make([]eventKey, len(events))
	for i, e := range events {
		keys[i] = eventKey{EventType: string(e.EventType), StepID: e.StepID}
	}
	return keys
}

// TestHarnessOracle is the T8 harness event-stream equivalence oracle (M09
// HANDOFF "What M10 may assume"): both the YAML-parsed and Builder-constructed
// definitions produce identical event-type/step-id sequences when run on
// separate awistesting.Harness instances in deterministic mode.
func TestHarnessOracle(t *testing.T) {
	yamlDef, err := dsl.ParseFile("testdata/keystone.yaml")
	if err != nil {
		t.Fatalf("ParseFile(keystone.yaml): %v", err)
	}
	builderDef := keystoneBuilderDef(t)

	yamlEvents := runKeystoneHarness(t, yamlDef)
	builderEvents := runKeystoneHarness(t, builderDef)

	if len(yamlEvents) != len(builderEvents) {
		t.Fatalf("event stream length mismatch: YAML=%d Builder=%d", len(yamlEvents), len(builderEvents))
	}
	for i := range yamlEvents {
		if yamlEvents[i] != builderEvents[i] {
			t.Errorf("event[%d] mismatch: YAML={%s %q} Builder={%s %q}",
				i, yamlEvents[i].EventType, yamlEvents[i].StepID,
				builderEvents[i].EventType, builderEvents[i].StepID)
		}
	}
}
