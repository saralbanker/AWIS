// Integration tests for RecallAPI methods on *Runtime (M08-C2 output 5;
// AWIS DoD §24.3).
package sdk

import (
	"context"
	"testing"

	"github.com/awis/awis/internal/core"
)

// TestQueryHistory_ReturnsRecordAfterCompletion verifies that QueryHistory
// returns at least one ExecutionRecord after a workflow completes.
func TestQueryHistory_ReturnsRecordAfterCompletion(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "recall-ns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def, _ := NewWorkflowBuilder("recallflow", "1.0.0").
		SetNamespace("recall-ns").
		AddStep(core.Step{
			ID:      "greet",
			Name:    "Greet",
			Type:    core.StepTypeNative,
			Handler: core.HandlerRef("greet"),
		}).
		SetInitialStep("greet").
		AddFinalStep("greet").
		Build()
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	_, err = rt.Submit(ctx, "recallflow", map[string]any{"name": "history"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Run one tick to complete the workflow.
	if err := rt.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	records, err := rt.QueryHistory(ctx, core.HistoryQuery{Namespace: "recall-ns"})
	if err != nil {
		t.Fatalf("QueryHistory: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("QueryHistory returned 0 records, want ≥1")
	}

	rec := records[0]
	// F3: InstanceID must be non-empty.
	if rec.InstanceID == "" {
		t.Error("record.InstanceID is empty, want non-empty")
	}
	if rec.DefinitionID != "recallflow" {
		t.Errorf("record.DefinitionID = %q, want %q", rec.DefinitionID, "recallflow")
	}
}

// TestStepStats_AfterCompletion verifies that StepStats returns TotalRuns >= 1
// after a workflow completes (F4).
func TestStepStats_AfterCompletion(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "stats-ns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def, _ := NewWorkflowBuilder("statsflow", "1.0.0").
		SetNamespace("stats-ns").
		AddStep(core.Step{
			ID:      "greet",
			Name:    "Greet",
			Type:    core.StepTypeNative,
			Handler: core.HandlerRef("greet"),
		}).
		SetInitialStep("greet").
		AddFinalStep("greet").
		Build()
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	if _, err := rt.Submit(ctx, "statsflow", nil); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick to run and complete the workflow.
	if err := rt.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	stats, err := rt.StepStats(ctx, "statsflow", "greet")
	if err != nil {
		t.Fatalf("StepStats: %v", err)
	}
	if stats.TotalRuns < 1 {
		t.Errorf("StepStats.TotalRuns = %d, want >= 1", stats.TotalRuns)
	}
}

// TestQueryHistory_FilterByDefinitionID verifies that DefinitionID filtering
// excludes unrelated workflows.
func TestQueryHistory_FilterByDefinitionID(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "filter-ns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}

	for _, id := range []string{"flow-a", "flow-b"} {
		def, _ := NewWorkflowBuilder(id, "1.0.0").
			SetNamespace("filter-ns").
			AddStep(core.Step{
				ID:      "greet",
				Name:    "Greet",
				Type:    core.StepTypeNative,
				Handler: core.HandlerRef("greet"),
			}).
			SetInitialStep("greet").
			AddFinalStep("greet").
			Build()
		if err := rt.RegisterWorkflow(def); err != nil {
			t.Fatalf("RegisterWorkflow %s: %v", id, err)
		}
		if _, err := rt.Submit(ctx, id, nil); err != nil {
			t.Fatalf("Submit %s: %v", id, err)
		}
	}

	records, err := rt.QueryHistory(ctx, core.HistoryQuery{
		Namespace:    "filter-ns",
		DefinitionID: "flow-a",
	})
	if err != nil {
		t.Fatalf("QueryHistory: %v", err)
	}
	for _, rec := range records {
		if rec.DefinitionID != "flow-a" {
			t.Errorf("expected only flow-a records, got DefinitionID=%q", rec.DefinitionID)
		}
	}
}

// TestReplayInstance_Stub verifies that ReplayInstance returns an empty trace
// and no error (M17 stub).
func TestReplayInstance_Stub(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "ns", Storage: s})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	trace, err := rt.ReplayInstance(ctx, core.InstanceID("does-not-matter"))
	if err != nil {
		t.Fatalf("ReplayInstance returned unexpected error: %v", err)
	}
	_ = trace // must be zero value; no fields to check on ReplayTrace{}
}
