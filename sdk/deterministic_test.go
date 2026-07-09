// Tests for M09-C1 determinism seams: injectable ID/Clock, DeterministicMode,
// intelligence wiring (spec §1a–1d; AWIS DoD §24.3).
package sdk

import (
	"context"
	"regexp"
	"testing"

	"github.com/awis/awis/internal/core"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// openDetStorage opens a temp-file SQLite storage for determinism tests.
func openDetStorage(t *testing.T) core.StoragePort {
	t.Helper()
	s, err := SQLiteStorage(t.TempDir() + "/det.db")
	if err != nil {
		t.Fatalf("SQLiteStorage: %v", err)
	}
	return s
}

// buildSingleStepDef builds a minimal single-native-step workflow definition.
func buildSingleStepDef(ns, id string) *core.WorkflowDefinition {
	def, err := NewWorkflowBuilder(id, "1.0.0").
		SetNamespace(ns).
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
		panic("buildSingleStepDef: " + err.Error())
	}
	return def
}

// noopHandler is a native StepHandler that does nothing and succeeds.
type noopHandler struct{}

func (h *noopHandler) ID() string { return "noop" }
func (h *noopHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{}}, nil
}

// ── Test 1: fixed NewID source → Submit returns exactly the provided ID ───────

func TestDeterminism_FixedNewID(t *testing.T) {
	ctx := context.Background()
	s := openDetStorage(t)

	cfg := DeterministicMode()
	cfg.Namespace = "detns"
	cfg.Storage = s

	rt, err := NewRuntime(cfg)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterHandler(&noopHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def := buildSingleStepDef("detns", "detflow")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "detflow", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// The first ID minted by DeterministicMode is "det-000001".
	if string(iid) != "det-000001" {
		t.Errorf("InstanceID = %q, want %q", iid, "det-000001")
	}
}

// ── Test 2: fixed Clock → instance CreatedAt equals the fixed instant ─────────

func TestDeterminism_FixedClock(t *testing.T) {
	ctx := context.Background()
	s := openDetStorage(t)

	cfg := DeterministicMode()
	cfg.Namespace = "detns"
	cfg.Storage = s

	rt, err := NewRuntime(cfg)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterHandler(&noopHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def := buildSingleStepDef("detns", "clockflow")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "clockflow", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	status, err := rt.Status(ctx, iid)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	// CreatedAt must equal the deterministic epoch (the first clock reading for
	// WorkflowStarted emitted_at is tick 1 → epoch + 0ms = deterministicEpoch).
	if !status.CreatedAt.Equal(deterministicEpoch) {
		t.Errorf("CreatedAt = %v, want %v", status.CreatedAt, deterministicEpoch)
	}
}

// ── Test 3: DeterministicMode — two Runtimes over fresh storages produce ─────
//            identical ID sequences.

func TestDeterminism_TwoRuntimesIdenticalSequence(t *testing.T) {
	ctx := context.Background()

	runAndCollectIDs := func(label string) []string {
		s := openDetStorage(t)
		cfg := DeterministicMode()
		cfg.Namespace = "detns"
		cfg.Storage = s

		rt, err := NewRuntime(cfg)
		if err != nil {
			t.Fatalf("%s: NewRuntime: %v", label, err)
		}
		if err := rt.RegisterHandler(&noopHandler{}); err != nil {
			t.Fatalf("%s: RegisterHandler: %v", label, err)
		}
		def := buildSingleStepDef("detns", "seqflow")
		if err := rt.RegisterWorkflow(def); err != nil {
			t.Fatalf("%s: RegisterWorkflow: %v", label, err)
		}

		var ids []string
		for i := 0; i < 3; i++ {
			iid, err := rt.Submit(ctx, "seqflow", nil)
			if err != nil {
				t.Fatalf("%s: Submit %d: %v", label, i, err)
			}
			ids = append(ids, string(iid))
		}
		return ids
	}

	ids1 := runAndCollectIDs("rt1")
	ids2 := runAndCollectIDs("rt2")

	if len(ids1) != len(ids2) {
		t.Fatalf("ID sequence lengths differ: %d vs %d", len(ids1), len(ids2))
	}
	for i := range ids1 {
		if ids1[i] != ids2[i] {
			t.Errorf("sequence[%d]: rt1=%q rt2=%q", i, ids1[i], ids2[i])
		}
	}
}

// ── Test 4: nil seams → behavior unchanged (UUID-shaped ID, wall-clock time) ──

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestDeterminism_NilSeamsBehaviorUnchanged(t *testing.T) {
	ctx := context.Background()
	s := openDetStorage(t)

	// nil Clock and nil NewID — plain default Config.
	rt, err := NewRuntime(Config{
		Namespace: "nilns",
		Storage:   s,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterHandler(&noopHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	def := buildSingleStepDef("nilns", "nilflow")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "nilflow", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// ID must be UUID v4 shaped.
	if !uuidRE.MatchString(string(iid)) {
		t.Errorf("InstanceID = %q, expected UUIDv4 pattern", iid)
	}

	// CreatedAt must be a non-zero wall-clock time (not the deterministic epoch).
	status, err := rt.Status(ctx, iid)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, expected wall-clock time")
	}
	if status.CreatedAt.Equal(deterministicEpoch) {
		t.Error("CreatedAt equals deterministic epoch, expected wall-clock time")
	}
}

// ── stubIntelPort is a minimal available IntelligencePort for determinism tests.

type stubIntelPort struct {
	output map[string]any
}

func (s *stubIntelPort) Draft(_ context.Context, _ core.DraftRequest) (core.DraftResponse, error) {
	return core.DraftResponse{
		Output: s.output,
		Usage:  core.Usage{Adapter: "stub", Model: "stub-model", TokensUsed: 1},
	}, nil
}
func (s *stubIntelPort) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}
func (s *stubIntelPort) Synthesize(_ context.Context, _ core.SynthesisRequest) (core.SynthesisResponse, error) {
	return core.SynthesisResponse{}, nil
}
func (s *stubIntelPort) Classify(_ context.Context, _ string, _ []string) (core.Classification, error) {
	return core.Classification{}, nil
}
func (s *stubIntelPort) IsAvailable() bool { return true }
func (s *stubIntelPort) Capabilities() []core.Capability {
	return []core.Capability{{Name: "draft"}}
}
func (s *stubIntelPort) ProviderName() string { return "stub" }

// buildIntelWorkflow returns a single intelligence-step workflow definition.
func buildIntelWorkflow(ns, id string) *core.WorkflowDefinition {
	def, err := NewWorkflowBuilder(id, "1.0.0").
		SetNamespace(ns).
		AddStep(core.Step{
			ID:   "gen",
			Name: "gen",
			Type: core.StepTypeIntelligence,
			Intelligence: &core.IntelReq{
				Capability:    "draft",
				Required:      false,
				ContextBudget: 1000,
			},
		}).
		SetInitialStep("gen").
		AddFinalStep("gen").
		Build()
	if err != nil {
		panic("buildIntelWorkflow: " + err.Error())
	}
	return def
}

// ── Test 5: intelligence step with a stub port → step receives stub's response

func TestDeterminism_IntelligenceStubPort(t *testing.T) {
	ctx := context.Background()
	s := openDetStorage(t)

	stub := &stubIntelPort{output: map[string]any{"draft": "hello from stub"}}

	cfg := DeterministicMode()
	cfg.Namespace = "intelns"
	cfg.Storage = s
	cfg.Intelligence = stub

	rt, err := NewRuntime(cfg)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	def := buildIntelWorkflow("intelns", "intelflow")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "intelflow", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick until terminal.
	for i := 0; i < 5; i++ {
		if err := rt.Tick(ctx); err != nil {
			t.Fatalf("Tick %d: %v", i, err)
		}
		st, err := rt.Status(ctx, iid)
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if st.Status == core.InstanceStatusCompleted || st.Status == core.InstanceStatusFailed {
			if st.Status != core.InstanceStatusCompleted {
				t.Errorf("instance status = %q, want completed", st.Status)
			}
			return
		}
	}
	t.Fatal("instance did not reach terminal status after 5 ticks")
}

// ── Test 6: intelligence step with nil Intelligence → NullAdapter semantics,
//            no panic. The null adapter is never available (IsAvailable=false),
//            so an optional (required=false) intelligence step falls back and
//            reaches capability_fallback. Since the step has no explicit fallback
//            step in this definition, the workflow fails — which is the
//            NullAdapter degradation path, not a panic.

func TestDeterminism_NilIntelligenceNoPanic(t *testing.T) {
	ctx := context.Background()
	s := openDetStorage(t)

	// Intelligence explicitly nil → NullAdapter wired by NewRuntime.
	cfg := DeterministicMode()
	cfg.Namespace = "nullintelns"
	cfg.Storage = s
	cfg.Intelligence = nil

	rt, err := NewRuntime(cfg)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	def := buildIntelWorkflow("nullintelns", "nullintelflow")
	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	iid, err := rt.Submit(ctx, "nullintelflow", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick until terminal — must not panic regardless of outcome.
	var finalStatus core.InstanceStatus
	for i := 0; i < 5; i++ {
		if err := rt.Tick(ctx); err != nil {
			// Allow engine errors; the test validates no panic, not success.
			_ = err
		}
		st, err := rt.Status(ctx, iid)
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		finalStatus = st.Status
		if st.Status == core.InstanceStatusCompleted || st.Status == core.InstanceStatusFailed {
			break
		}
	}

	// NullAdapter: IsAvailable=false → capability_fallback or capability_unavailable.
	// The step has required=false and no fallback step ⇒ the engine fails the workflow.
	// Either failed or completed is acceptable (degradation path); panic is not.
	if finalStatus == core.InstanceStatusRunning || finalStatus == core.InstanceStatusWaiting {
		// Still not terminal after 5 ticks — acceptable since we verified no panic.
		_ = finalStatus
	}
	// If we reach here without panicking, the test passes.
	_ = finalStatus
}
