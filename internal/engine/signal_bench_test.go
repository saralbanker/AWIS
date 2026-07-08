package engine

// BenchmarkSignalReceiptToResumedStep measures the end-to-end latency from
// signal intake (Signal()) through delivery + step completion (one Tick()),
// expressed in ms/op alongside Go's ns/op.
//
// Binding NFR-P-04 target is ≤200ms; enforcement is M18. This benchmark is
// informational (M07-C3r card output: "Bench number reported next to 200ms").

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

func BenchmarkSignalReceiptToResumedStep(b *testing.B) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "bench.sig", Version: "1.0.0",
		Namespace: "bench", Name: "sig-bench",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{signalStep("w", "bench-signal")},
		InitialStep: "w", FinalSteps: []string{"w"}, Metadata: map[string]any{},
	}

	// Inline storage setup (helpers_test.go uses *testing.T, not testing.TB).
	dbPath := filepath.Join(b.TempDir(), "bench.db")
	db, err := storage.Open(dbPath, func() time.Time { return time.Now() })
	if err != nil {
		b.Fatalf("storage.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	s := storage.NewSQLiteStorage(db, func() time.Time { return time.Now() })

	ctx := context.Background()
	if err := s.RegisterWorkflow(ctx, def); err != nil {
		b.Fatalf("RegisterWorkflow: %v", err)
	}

	e := New(s, map[core.StepType]Runner{core.StepTypeNative: native.New()},
		Config{MaxParallelSteps: 1, WorkerID: "bench-worker"},
		discardLogger())

	// Pre-allocate instances to avoid allocation during the timed section.
	// Each iteration needs a unique instance — submit all upfront, park each
	// with one Tick, then measure Signal+Tick in the loop body.
	iids := make([]core.InstanceID, b.N)
	for i := 0; i < b.N; i++ {
		iid, err := e.Submit(ctx, "bench.sig", "1.0.0", nil)
		if err != nil {
			b.Fatalf("Submit[%d]: %v", i, err)
		}
		// Park the signal step in waiting state.
		if err := e.Tick(ctx); err != nil {
			b.Fatalf("Tick(park)[%d]: %v", i, err)
		}
		iids[i] = iid
	}

	// Measured section: Signal() + delivery Tick() per instance.
	var totalNs int64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if err := e.Signal(ctx, iids[i], "bench-signal", map[string]any{"i": i}); err != nil {
			b.Fatalf("Signal[%d]: %v", i, err)
		}
		if err := e.Tick(ctx); err != nil {
			b.Fatalf("Tick(deliver)[%d]: %v", i, err)
		}
		totalNs += time.Since(start).Nanoseconds()
	}
	b.StopTimer()

	msPerOp := float64(totalNs) / float64(b.N) / float64(time.Millisecond)
	b.ReportMetric(msPerOp, "ms/op")
	b.Logf("signal receipt → step completion: %.2f ms/op  (target: ≤200ms; binding: M18)", msPerOp)
}
