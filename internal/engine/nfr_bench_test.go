package engine

// Engine-level NFR benchmarks (M18-C1): NFR-P-01 (execution-loop tick latency)
// and NFR-P-02 (native step dispatch latency, CONTRA-2 dual threshold).
//
// Run through `make bench` with -benchtime=1x. Sample counts are fixed by the
// NFR harness and are not derived from b.N, so each benchmark performs its full
// measurement exactly once and prints one nfrbench result line. Storage is the
// production on-disk WAL store (storage.Open), not an in-memory shortcut. Each
// fixture asserts its own invariants and fails the benchmark if they do not
// hold, so a result always describes the workload it names.

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/nfrbench"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

const (
	// nfrTickInstances is the NFR-P-01 fixture size: running instances, each
	// parked in retry backoff after one failed attempt, so no step is runnable.
	// Every Tick scans and evaluates all of them and dispatches nothing. The PRD
	// does not fix an instance count; 100 is this benchmark's stated bound.
	nfrTickInstances = 100
	// nfrTickSamples is the number of timed Tick calls for NFR-P-01.
	nfrTickSamples = 1000
	// nfrDispatchSamples is the number of single-native-step instances for
	// NFR-P-02, each timed through the one Tick that claims, dispatches and
	// settles its step.
	nfrDispatchSamples = 500
	// nfrDispatchStepID is the step id of the NFR-P-02 workflow.
	nfrDispatchStepID = "n"
)

// nfrOpenStore opens an on-disk store at dbPath with the production WAL and
// busy-timeout configuration, and closes it when tb finishes.
func nfrOpenStore(tb testing.TB, dbPath string) *storage.SQLiteStorage {
	tb.Helper()
	db, err := storage.Open(dbPath, time.Now)
	if err != nil {
		tb.Fatalf("storage.Open %q: %v", dbPath, err)
	}
	tb.Cleanup(func() { _ = db.Close() })
	return storage.NewSQLiteStorage(db, time.Now)
}

// nfrNewEngine builds the engine under test on s: the production Tick path, a
// native runner holding handlers, a deterministic instance-id source and the
// production clock (Config.Clock nil means time.Now).
func nfrNewEngine(s *storage.SQLiteStorage, handlers ...core.StepHandler) *Engine {
	nr := native.New()
	for _, h := range handlers {
		nr.Register(h)
	}
	var seq atomic.Int64
	return New(s, map[core.StepType]Runner{core.StepTypeNative: nr}, Config{
		MaxParallelSteps: 4,
		WorkerID:         "nfr-bench",
		NewID:            func() string { return fmt.Sprintf("nfr-%08d", seq.Add(1)) },
	}, discardLogger())
}

// nfrNativeDef is a one-step native workflow "id@1.0.0" whose only step is
// nfrDispatchStepID, bound to handler, with an optional retry policy.
func nfrNativeDef(id, handler string, retry *core.RetryPolicy) core.WorkflowDefinition {
	step := nativeStep(nfrDispatchStepID, handler)
	step.Retry = retry
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: id, Version: "1.0.0",
		Namespace: "bench", Name: id,
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{step},
		InitialStep: nfrDispatchStepID, FinalSteps: []string{nfrDispatchStepID},
		Metadata: map[string]any{},
	}
}

// nfrEventCount returns the total number of EventLog rows across ids.
func nfrEventCount(tb testing.TB, s *storage.SQLiteStorage, ids []core.InstanceID) int {
	tb.Helper()
	total := 0
	for _, id := range ids {
		evs, err := s.ReadEvents(context.Background(), id, 0)
		if err != nil {
			tb.Fatalf("ReadEvents %s: %v", id, err)
		}
		total += len(evs)
	}
	return total
}

// nfrStepSpan returns the EventLog span from the first StepStarted to the first
// StepCompleted for stepID, from the events' own EmittedAt timestamps. It is
// the PRD §31 "EventLog analysis" view of native step execution latency.
func nfrStepSpan(evs []core.ExecutionEvent, stepID string) (time.Duration, error) {
	var started, completed *core.ExecutionEvent
	for i := range evs {
		ev := &evs[i]
		if ev.StepID != stepID {
			continue
		}
		switch ev.EventType {
		case core.EventTypeStepStarted:
			if started == nil {
				started = ev
			}
		case core.EventTypeStepCompleted:
			if completed == nil {
				completed = ev
			}
		}
	}
	if started == nil || completed == nil {
		return 0, fmt.Errorf("step %q: EventLog lacks StepStarted or StepCompleted", stepID)
	}
	return completed.EmittedAt.Sub(started.EmittedAt), nil
}

// nfrFailFirstAttempt fails every instance's first attempt and succeeds on later
// attempts. It keys on the per-instance attempt ordinal (StepContext.Attempt),
// so each instance in the NFR-P-01 fixture fails exactly once. (flakyHandler
// keeps one counter across all instances and cannot express this.)
type nfrFailFirstAttempt struct {
	id    string
	calls int32
}

func (h *nfrFailFirstAttempt) ID() string { return h.id }

func (h *nfrFailFirstAttempt) Execute(sc core.StepContext) (core.StepResult, error) {
	atomic.AddInt32(&h.calls, 1)
	if sc.Attempt <= 1 {
		return core.StepResult{}, errors.New("nfr transient failure")
	}
	return core.StepResult{Outputs: map[string]any{}}, nil
}

// runTickSamples builds the NFR-P-01 fixture (k running instances, each parked
// in retry backoff after one failed attempt) and returns the wall-clock duration
// of n consecutive Tick calls. It fails tb if the fixture does not hold (k
// running instances, k first attempts, none dispatched again) or if sampling
// dispatched any step or wrote any EventLog row.
func runTickSamples(tb testing.TB, k, n int) []time.Duration {
	tb.Helper()
	ctx := context.Background()
	s := nfrOpenStore(tb, filepath.Join(tb.TempDir(), "tick.db"))
	def := nfrNativeDef("nfr.tick", "nfr-flaky", &core.RetryPolicy{
		Attempts: 2, Backoff: "linear", InitialDelay: "1h", MaxDelay: "1h",
	})
	if err := s.RegisterWorkflow(ctx, def); err != nil {
		tb.Fatalf("RegisterWorkflow: %v", err)
	}
	failer := &nfrFailFirstAttempt{id: "nfr-flaky"}
	e := nfrNewEngine(s, failer)

	ids := make([]core.InstanceID, 0, k)
	for i := 0; i < k; i++ {
		iid, err := e.Submit(ctx, "nfr.tick", "1.0.0", nil)
		if err != nil {
			tb.Fatalf("Submit %d: %v", i, err)
		}
		ids = append(ids, iid)
	}

	// Fixture tick: every first attempt fails and its retry is scheduled an hour
	// out, so each instance stays running with no runnable step.
	if err := e.Tick(ctx); err != nil {
		tb.Fatalf("fixture tick: %v", err)
	}
	if got := atomic.LoadInt32(&failer.calls); got != int32(k) {
		tb.Fatalf("fixture: want %d first attempts, handler ran %d times", k, got)
	}
	running, err := s.ListInstances(ctx, core.InstanceFilter{Status: core.InstanceStatusRunning})
	if err != nil {
		tb.Fatalf("ListInstances: %v", err)
	}
	if len(running) != k {
		tb.Fatalf("fixture: want %d running instances, got %d", k, len(running))
	}

	rowsBefore := nfrEventCount(tb, s, ids)
	samples := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		t0 := time.Now()
		tickErr := e.Tick(ctx)
		d := time.Since(t0)
		if tickErr != nil {
			tb.Fatalf("sample %d: Tick: %v", i, tickErr)
		}
		samples = append(samples, d)
	}
	if got := atomic.LoadInt32(&failer.calls); got != int32(k) {
		tb.Fatalf("sampling dispatched work: handler calls %d, want %d", got, k)
	}
	if rows := nfrEventCount(tb, s, ids); rows != rowsBefore {
		tb.Fatalf("sampling wrote EventLog rows: %d before, %d after", rowsBefore, rows)
	}
	return samples
}

// runDispatchSamples submits one single-native-step instance per sample (untimed)
// and times the one Tick that claims, dispatches and settles it. It returns the
// Tick durations and, as the PRD §31 cross-check, the EventLog
// StepStarted->StepCompleted span of each step. It fails tb if any sample does
// not complete within that one Tick.
func runDispatchSamples(tb testing.TB, n int) (ticks, spans []time.Duration) {
	tb.Helper()
	ctx := context.Background()
	s := nfrOpenStore(tb, filepath.Join(tb.TempDir(), "dispatch.db"))
	if err := s.RegisterWorkflow(ctx, nfrNativeDef("nfr.dispatch", "nfr-noop", nil)); err != nil {
		tb.Fatalf("RegisterWorkflow: %v", err)
	}
	e := nfrNewEngine(s, &stepHandler{id: "nfr-noop", outputs: map[string]any{}})

	for i := 0; i < n; i++ {
		iid, err := e.Submit(ctx, "nfr.dispatch", "1.0.0", nil)
		if err != nil {
			tb.Fatalf("sample %d: Submit: %v", i, err)
		}
		t0 := time.Now()
		tickErr := e.Tick(ctx)
		d := time.Since(t0)
		if tickErr != nil {
			tb.Fatalf("sample %d: Tick: %v", i, tickErr)
		}
		inst, err := s.GetInstance(ctx, iid)
		if err != nil {
			tb.Fatalf("sample %d: GetInstance: %v", i, err)
		}
		if inst.Status != core.InstanceStatusCompleted {
			tb.Fatalf("sample %d: instance %s is %q after one Tick; the timed unit must claim, dispatch and settle the step",
				i, iid, inst.Status)
		}
		evs, err := s.ReadEvents(ctx, iid, 0)
		if err != nil {
			tb.Fatalf("sample %d: ReadEvents: %v", i, err)
		}
		span, err := nfrStepSpan(evs, nfrDispatchStepID)
		if err != nil {
			tb.Fatalf("sample %d: %v", i, err)
		}
		ticks = append(ticks, d)
		spans = append(spans, span)
	}
	return ticks, spans
}

// BenchmarkNFRP01TickLatency measures NFR-P-01: execution-loop tick latency with
// no step execution (PRD §11: "< 5ms P99", "no I/O"). A tick over parked
// instances still reads the EventLog and projection, so "no I/O" is taken as no
// step I/O and no writes: the fixture asserts zero dispatches and zero EventLog
// rows during sampling. The verdict is on P99.
func BenchmarkNFRP01TickLatency(b *testing.B) {
	samples := runTickSamples(b, nfrTickInstances, nfrTickSamples)
	sum, err := nfrbench.Summarize(samples)
	if err != nil {
		b.Fatalf("NFR-P-01 summarize: %v", err)
	}
	res, err := nfrbench.NewResult("NFR-P-01", "execution-loop tick latency (no step I/O)",
		fmt.Sprintf("%d running instances parked in retry backoff (0 runnable steps, 0 dispatches and 0 EventLog writes during sampling); on-disk WAL store; %d Tick calls",
			nfrTickInstances, nfrTickSamples),
		sum,
		nfrbench.Bound{Label: "NFR-P-01 target", Limit: 5 * time.Millisecond, Strict: true,
			Basis: nfrbench.BasisP99, Binding: true},
	)
	if err != nil {
		b.Fatalf("NFR-P-01 verdict: %v", err)
	}
	nfrbench.Report(b, res)
}

// BenchmarkNFRP02NativeDispatch measures NFR-P-02: native step dispatch latency
// for a Go handler with no I/O of its own (PRD §32 "P95 <= 50ms"; NFR table
// "< 10ms P95"). CONTRA-2 is dual: the 50ms ship gate is binding and the 10ms
// engineering target is informational. Both are judged on P95 and both appear
// in the one result line.
func BenchmarkNFRP02NativeDispatch(b *testing.B) {
	ticks, spans := runDispatchSamples(b, nfrDispatchSamples)
	sum, err := nfrbench.Summarize(ticks)
	if err != nil {
		b.Fatalf("NFR-P-02 summarize: %v", err)
	}
	spanSum, err := nfrbench.Summarize(spans)
	if err != nil {
		b.Fatalf("NFR-P-02 span summarize: %v", err)
	}
	res, err := nfrbench.NewResult("NFR-P-02", "native step dispatch latency (Go handler, no I/O)",
		fmt.Sprintf("1 native step per instance, no-op Go handler; timed unit = one Tick that claims, dispatches and settles it; on-disk WAL store; %d instances; EventLog StepStarted->StepCompleted P95=%s (PRD §31 cross-check)",
			nfrDispatchSamples, spanSum.P95),
		sum,
		nfrbench.Bound{Label: "ship gate (PRD §32)", Limit: 50 * time.Millisecond,
			Basis: nfrbench.BasisP95, Binding: true},
		nfrbench.Bound{Label: "engineering target (CONTRA-2)", Limit: 10 * time.Millisecond, Strict: true,
			Basis: nfrbench.BasisP95, Binding: false},
	)
	if err != nil {
		b.Fatalf("NFR-P-02 verdict: %v", err)
	}
	nfrbench.Report(b, res)
}
