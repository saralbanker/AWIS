package api

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/sdk"
)

// noopHandler is a StepHandler that succeeds immediately with no outputs —
// the same minimal shape sdk/testing's own noopH uses for linear-workflow
// fixtures.
type noopHandler struct{}

func (noopHandler) ID() string { return "noop" }
func (noopHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{}}, nil
}

// newTestRuntime builds a real *sdk.Runtime backed by a temp-file SQLite DB
// (not :memory: — matches this repo's existing test convention, e.g.
// cmd/awis-server/main_test.go) with the noop handler registered, ready to
// pass to NewRouter.
func newTestRuntime(t *testing.T) (*sdk.Runtime, core.StoragePort) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := sdk.SQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	rt, err := sdk.NewRuntime(sdk.Config{
		Namespace:    "default",
		Storage:      store,
		TickInterval: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	if err := rt.RegisterHandler(noopHandler{}); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	return rt, store
}

// linearWorkflow returns a single-native-step workflow that completes
// immediately, mirroring sdk/testing/harness_test.go's linearDef.
func linearWorkflow(id string) *core.WorkflowDefinition {
	def, err := sdk.NewWorkflowBuilder(id, "1.0.0").
		SetNamespace("default").
		AddStep(core.Step{
			ID:      "s1",
			Name:    "s1",
			Type:    core.StepTypeNative,
			Handler: core.HandlerRef("noop"),
		}).
		SetInitialStep("s1").
		AddFinalStep("s1").
		Build()
	if err != nil {
		panic("linearWorkflow: " + err.Error())
	}
	return def
}

// waitWorkflow returns a workflow that parks on signal "go" with a 5-minute
// timeout, mirroring sdk/testing/harness_test.go's waitDef — used to drive
// an instance into InstanceStatusWaiting for the wait-record wrapper tests.
func waitWorkflow(id string) *core.WorkflowDefinition {
	def, err := sdk.NewWorkflowBuilder(id, "1.0.0").
		SetNamespace("default").
		AddStep(core.Step{
			ID:         "w",
			Name:       "wait",
			Type:       core.StepTypeSignal,
			WaitSignal: &core.WaitConfig{SignalName: "go", Timeout: core.Duration("5m"), TimeoutAction: "fail"},
		}).
		SetInitialStep("w").
		AddFinalStep("w").
		Build()
	if err != nil {
		panic("waitWorkflow: " + err.Error())
	}
	return def
}

// tickUntil ticks rt until instance id reaches status want or timeout
// elapses, failing the test on timeout.
func tickUntil(t *testing.T, rt *sdk.Runtime, id core.InstanceID, want core.InstanceStatus, timeout time.Duration) core.WorkflowStatus {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := rt.Tick(ctx); err != nil {
			t.Fatalf("tick: %v", err)
		}
		status, err := rt.Status(ctx, id)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		if status.Status == want {
			return status
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("instance %s did not reach status %s within %s", id, want, timeout)
	return core.WorkflowStatus{}
}
