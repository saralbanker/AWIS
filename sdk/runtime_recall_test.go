// Integration tests for RecallAPI methods on *Runtime (M08-C2 output 5;
// AWIS DoD §24.3).
package sdk

import (
	"context"
	"fmt"
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

// TestReplayInstance_UnknownInstance_ReturnsError verifies that ReplayInstance
// returns a real error (not a zero-value ReplayTrace with nil error) for an
// instance id that does not exist (B-11a: the old stub always returned
// success while doing nothing).
func TestReplayInstance_UnknownInstance_ReturnsError(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "ns", Storage: s})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	_, err = rt.ReplayInstance(ctx, core.InstanceID("does-not-matter"))
	if err == nil {
		t.Fatal("ReplayInstance: expected error for unknown instance id, got nil")
	}
}

// TestReplayInstance_ReturnsFullEventListForCompletedInstance verifies that
// ReplayInstance returns the full ordered EventLog and matching identity/
// lifecycle fields for a completed instance (B-11a).
func TestReplayInstance_ReturnsFullEventListForCompletedInstance(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "replay-ns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def := buildHelloWorkflow("replay-ns")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "hello", map[string]any{"name": "replay"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := rt.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	wantEvents, err := s.ReadEvents(ctx, iid, 0)
	if err != nil {
		t.Fatalf("ReadEvents (reference): %v", err)
	}
	if len(wantEvents) == 0 {
		t.Fatal("reference ReadEvents returned 0 events; test setup is broken")
	}

	trace, err := rt.ReplayInstance(ctx, iid)
	if err != nil {
		t.Fatalf("ReplayInstance: %v", err)
	}

	if trace.InstanceID != iid {
		t.Errorf("trace.InstanceID = %q, want %q", trace.InstanceID, iid)
	}
	if trace.DefinitionID != "hello" {
		t.Errorf("trace.DefinitionID = %q, want %q", trace.DefinitionID, "hello")
	}
	if trace.Namespace != "replay-ns" {
		t.Errorf("trace.Namespace = %q, want %q", trace.Namespace, "replay-ns")
	}
	if len(trace.Events) != len(wantEvents) {
		t.Fatalf("trace.Events has %d events, want %d", len(trace.Events), len(wantEvents))
	}
	for i := range wantEvents {
		if trace.Events[i].EventID != wantEvents[i].EventID {
			t.Errorf("trace.Events[%d].EventID = %q, want %q (order must match EventLog)",
				i, trace.Events[i].EventID, wantEvents[i].EventID)
		}
	}
}

// buildHelloWorkflowWithInputs is buildHelloWorkflow plus an explicit step
// input mapping, so the workflow input actually reaches the handler. The shared
// buildHelloWorkflow declares no `Inputs`, and AWIS step inputs are explicit
// templates (internal/engine.assembleContext resolves step.Inputs against the
// instance env) — so with no mapping the handler receives an empty map and
// greetHandler falls back to its default name. The Inputs/Outputs split tests
// need a real value flowing end to end, hence this variant.
func buildHelloWorkflowWithInputs(ns string) *core.WorkflowDefinition {
	def, _ := NewWorkflowBuilder("hello", "1.0.0").
		SetNamespace(ns).
		AddStep(core.Step{
			ID:      "greet",
			Name:    "Greet",
			Type:    core.StepTypeNative,
			Handler: core.HandlerRef("greet"),
			Inputs:  core.InputSchema{"name": "{{workflow.inputs.name}}"},
		}).
		SetInitialStep("greet").
		AddFinalStep("greet").
		Build()
	return def
}

// TestStatus_InputsOutputsSplit verifies that Status.Inputs contains only the
// workflow inputs and Status.Outputs contains only step outputs — never the
// same map (B-11b).
func TestStatus_InputsOutputsSplit(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "io-ns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def := buildHelloWorkflowWithInputs("io-ns")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "hello", map[string]any{"name": "Ada"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := rt.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	status, err := rt.Status(ctx, iid)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if len(status.Inputs) != 1 || status.Inputs["name"] != "Ada" {
		t.Errorf("Status.Inputs = %#v, want {name: Ada}", status.Inputs)
	}
	if _, ok := status.Inputs["greet"]; ok {
		t.Error("Status.Inputs must not contain step output key \"greet\"")
	}
	if _, ok := status.Outputs["inputs"]; ok {
		t.Error("Status.Outputs must not contain the nested \"inputs\" key")
	}
	greetOut, ok := status.Outputs["greet"].(map[string]any)
	if !ok {
		t.Fatalf("Status.Outputs[\"greet\"] missing or wrong type: %#v", status.Outputs)
	}
	if greetOut["greeting"] != "hello Ada" {
		t.Errorf("Status.Outputs[\"greet\"][\"greeting\"] = %v, want %q", greetOut["greeting"], "hello Ada")
	}
}

// TestQueryHistory_InputsOutputsSplit verifies the same split (B-11b) through
// the QueryHistory path (ExecutionRecord), which must not drift from Status.
func TestQueryHistory_InputsOutputsSplit(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "io-hist-ns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def := buildHelloWorkflowWithInputs("io-hist-ns")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "hello", map[string]any{"name": "Grace"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := rt.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	records, err := rt.QueryHistory(ctx, core.HistoryQuery{Namespace: "io-hist-ns"})
	if err != nil {
		t.Fatalf("QueryHistory: %v", err)
	}
	var rec *core.ExecutionRecord
	for i := range records {
		if records[i].InstanceID == iid {
			rec = &records[i]
		}
	}
	if rec == nil {
		t.Fatalf("QueryHistory did not return the submitted instance %q", iid)
	}

	if len(rec.Inputs) != 1 || rec.Inputs["name"] != "Grace" {
		t.Errorf("rec.Inputs = %#v, want {name: Grace}", rec.Inputs)
	}
	if _, ok := rec.Inputs["greet"]; ok {
		t.Error("rec.Inputs must not contain step output key \"greet\"")
	}
	if _, ok := rec.Outputs["inputs"]; ok {
		t.Error("rec.Outputs must not contain the nested \"inputs\" key")
	}
	greetOut, ok := rec.Outputs["greet"].(map[string]any)
	if !ok {
		t.Fatalf("rec.Outputs[\"greet\"] missing or wrong type: %#v", rec.Outputs)
	}
	if greetOut["greeting"] != "hello Grace" {
		t.Errorf("rec.Outputs[\"greet\"][\"greeting\"] = %v, want %q", greetOut["greeting"], "hello Grace")
	}
}

// TestStepStats_SameStepIDAcrossDefinitions_ReportsSeparately verifies that
// StepStats scopes events to definitionID, so two workflows that both have a
// step called "work" get independent statistics instead of being silently
// merged (B-11c).
func TestStepStats_SameStepIDAcrossDefinitions_ReportsSeparately(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "multi-ns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}

	// Two distinct workflow definitions, each with a step id "work" (via the
	// shared "greet" handler so both dispatch and complete deterministically).
	for _, defID := range []string{"defA", "defB"} {
		def, buildErr := NewWorkflowBuilder(defID, "1.0.0").
			SetNamespace("multi-ns").
			AddStep(core.Step{
				ID:      "work",
				Name:    "Work",
				Type:    core.StepTypeNative,
				Handler: core.HandlerRef("greet"),
			}).
			SetInitialStep("work").
			AddFinalStep("work").
			Build()
		if buildErr != nil {
			t.Fatalf("Build %s: %v", defID, buildErr)
		}
		if err := rt.RegisterWorkflow(def); err != nil {
			t.Fatalf("RegisterWorkflow %s: %v", defID, err)
		}
	}

	// defA: submit + complete once. defB: submit + complete twice.
	if _, err := rt.Submit(ctx, "defA", nil); err != nil {
		t.Fatalf("Submit defA: %v", err)
	}
	if _, err := rt.Submit(ctx, "defB", nil); err != nil {
		t.Fatalf("Submit defB #1: %v", err)
	}
	if _, err := rt.Submit(ctx, "defB", nil); err != nil {
		t.Fatalf("Submit defB #2: %v", err)
	}
	if err := rt.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	statsA, err := rt.StepStats(ctx, "defA", "work")
	if err != nil {
		t.Fatalf("StepStats defA: %v", err)
	}
	statsB, err := rt.StepStats(ctx, "defB", "work")
	if err != nil {
		t.Fatalf("StepStats defB: %v", err)
	}

	if statsA.TotalRuns != 1 {
		t.Errorf("statsA.TotalRuns = %d, want 1 (must not include defB's runs)", statsA.TotalRuns)
	}
	if statsB.TotalRuns != 2 {
		t.Errorf("statsB.TotalRuns = %d, want 2 (must not include defA's runs)", statsB.TotalRuns)
	}
}

// flakyOnceHandler fails its first call then succeeds on every call after,
// used to exercise the documented TotalRuns/SuccessCount/FailureCount
// semantics for a step that retries once (B-11c).
type flakyOnceHandler struct {
	calls int
}

func (h *flakyOnceHandler) ID() string { return "flaky" }
func (h *flakyOnceHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	h.calls++
	if h.calls == 1 {
		return core.StepResult{}, fmt.Errorf("flaky: forced failure on first attempt")
	}
	return core.StepResult{Outputs: map[string]any{"ok": true}}, nil
}

// TestStepStats_RetriedStep_ReportsDocumentedCounts verifies that a step
// which fails once and then succeeds on retry reports TotalRuns=2 (both
// dispatch attempts), SuccessCount=1 (the one terminal success), and
// FailureCount=0 (the first failure had retrying:true, so it is not a
// terminal failure) — matching the StepStats doc comment exactly.
func TestStepStats_RetriedStep_ReportsDocumentedCounts(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)

	rt, err := NewRuntime(Config{Namespace: "retry-ns", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	h := &flakyOnceHandler{}
	if err := rt.RegisterHandler(h); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}

	def, buildErr := NewWorkflowBuilder("retryflow", "1.0.0").
		SetNamespace("retry-ns").
		AddStep(core.Step{
			ID:      "flaky",
			Name:    "Flaky",
			Type:    core.StepTypeNative,
			Handler: core.HandlerRef("flaky"),
			Retry:   &core.RetryPolicy{Attempts: 2, Backoff: "immediate"},
		}).
		SetInitialStep("flaky").
		AddFinalStep("flaky").
		Build()
	if buildErr != nil {
		t.Fatalf("Build: %v", buildErr)
	}
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "retryflow", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick until terminal; immediate backoff means the retry is due on the
	// very next tick (no manual clock advance needed).
	var finalStatus core.InstanceStatus
	for i := 0; i < 10; i++ {
		if err := rt.Tick(ctx); err != nil {
			t.Fatalf("Tick %d: %v", i, err)
		}
		st, statusErr := rt.Status(ctx, iid)
		if statusErr != nil {
			t.Fatalf("Status: %v", statusErr)
		}
		finalStatus = st.Status
		if finalStatus == core.InstanceStatusCompleted || finalStatus == core.InstanceStatusFailed {
			break
		}
	}
	if finalStatus != core.InstanceStatusCompleted {
		t.Fatalf("instance did not complete; final status = %q", finalStatus)
	}

	stats, err := rt.StepStats(ctx, "retryflow", "flaky")
	if err != nil {
		t.Fatalf("StepStats: %v", err)
	}
	if stats.TotalRuns != 2 {
		t.Errorf("stats.TotalRuns = %d, want 2 (one per attempt)", stats.TotalRuns)
	}
	if stats.SuccessCount != 1 {
		t.Errorf("stats.SuccessCount = %d, want 1", stats.SuccessCount)
	}
	if stats.FailureCount != 0 {
		t.Errorf("stats.FailureCount = %d, want 0 (the retrying:true failure is not terminal)", stats.FailureCount)
	}
}
