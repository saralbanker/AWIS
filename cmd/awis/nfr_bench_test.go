package main

// nfr_bench_test.go — M18-C1 CLI and runtime NFR benchmarks: NFR-P-04 (signal
// delivery), NFR-P-05 (awis trace over a 100K-event store), NFR-P-06 (awis
// rebuild-state over a 100K-event store) and NFR-P-07 (awis submit to first
// step started).
//
// Each benchmark drives the real command entry point (runSignal, runTrace,
// runRebuildState, runSubmit) against an on-disk runtime.db. NFR-P-04 and
// NFR-P-07 also run the real runtime pull loop (sdk.Runtime.Start, the awis
// start path at its 100ms default tick), so scheduler wait is inside the
// measured interval. Their latencies are read from EventLog EmittedAt
// timestamps, so polling intervals never enter the measurement. NFR-P-05 and
// NFR-P-06 time the command call directly.
//
// Run through `make bench` with -benchtime=1x. Sample counts are fixed by the
// harness and are not derived from b.N. Each benchmark prints one nfrbench
// result line and records its percentiles as benchmark metrics.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/nfrbench"
	"github.com/awis/awis/sdk"
)

const (
	// nfrLoopTick is the awis start --tick default, and the tick the runtime
	// loop runs at in NFR-P-04 and NFR-P-07.
	nfrLoopTick = 100 * time.Millisecond
	// nfrAwaitTimeout bounds every wait on the runtime loop in the harness. A
	// wait that exceeds it fails the benchmark rather than hanging.
	nfrAwaitTimeout = 10 * time.Second
	// nfrCLIStderrName is the file, inside the store directory, that captures
	// the CLI's and the runtime's stderr.
	nfrCLIStderrName = "cli-stderr.log"

	// nfrSignalSamples and nfrSubmitSamples are the NFR-P-04 and NFR-P-07 sample
	// counts. Each sample is one CLI command plus one runtime-loop pickup.
	nfrSignalSamples = 100
	nfrSubmitSamples = 100

	// nfrFixtureTotal is the NFR-P-05 and NFR-P-06 store size: 100,000 events.
	nfrFixtureTotal = 100000
	// nfrFixtureInstances is the multi-instance shape: 500 instances x 200 events.
	nfrFixtureInstances = 500
	// nfrTraceSamplesMulti and nfrTraceSamplesSingle are the NFR-P-05 sample
	// counts for the two shapes. The single-instance trace prints 100,000 lines.
	nfrTraceSamplesMulti  = 20
	nfrTraceSamplesSingle = 10
	// nfrRebuildSampleCount is the NFR-P-06 sample count. Each sample is a full
	// projection rebuild over the 100K-event store.
	nfrRebuildSampleCount = 5

	// nfrSubmitStepID identifies the NFR-P-07 workflow's single native step.
	nfrSubmitStepID = "n"
	// nfrSignalStepID and nfrSignalName identify the NFR-P-04 workflow's single
	// signal wait step.
	nfrSignalStepID = "w"
	nfrSignalName   = "nfr-sig"
)

// nfrPinCLI points the CLI at dataDir/runtime.db for the life of tb and sets the
// JSON flag. It also sets the default namespace that main() would have set from
// the -namespace flag (the zero value is not valid to NewRuntime).
//
// The CLI commands and the runtime's engine log at INFO as JSON to os.Stderr,
// and sdk gives the engine no logger, so os.Stderr is redirected to
// dataDir/cli-stderr.log for the life of tb. nfrCheckCLIErrors reads that log
// for ERROR records at the end of each measured phase, so the redirect cannot
// hide an engine error inside a measured run.
func nfrPinCLI(tb testing.TB, dataDir string, jsonOut bool) {
	tb.Helper()
	prevDir, prevJSON, prevNS := globalDataDir, globalJSON, globalNamespace
	globalDataDir, globalJSON, globalNamespace = dataDir, jsonOut, "default"

	logPath := filepath.Join(dataDir, nfrCLIStderrName)
	logFile, err := os.Create(logPath)
	if err != nil {
		tb.Fatalf("create %s: %v", logPath, err)
	}
	prevStderr := os.Stderr
	os.Stderr = logFile
	tb.Cleanup(func() {
		globalDataDir, globalJSON, globalNamespace = prevDir, prevJSON, prevNS
		os.Stderr = prevStderr
		if err := logFile.Close(); err != nil {
			tb.Errorf("close %s: %v", logPath, err)
		}
	})
}

// nfrCheckCLIErrors fails tb if the CLI/runtime log holds an ERROR record. It
// must run at the end of a measured phase, before teardown. Teardown cancels the
// runtime loop, which can interrupt an in-flight tick and log a context-canceled
// ERROR; that is shutdown noise, not measured behaviour.
func nfrCheckCLIErrors(tb testing.TB) {
	tb.Helper()
	path := filepath.Join(globalDataDir, nfrCLIStderrName)
	recs, err := nfrErrorRecords(path)
	if err != nil {
		tb.Fatalf("read %s: %v", path, err)
	}
	for _, rec := range recs {
		tb.Errorf("runtime logged an ERROR record during the measured run: %s", rec)
	}
}

// nfrToStdout runs fn with os.Stdout redirected to w and restores it afterwards.
// The CLI commands print through os.Stdout at call time, so this captures or
// discards their output without changing the commands.
func nfrToStdout(w *os.File, fn func()) {
	prev := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = prev }()
	fn()
}

// nfrDevNull returns an open /dev/null for discarding command output during
// timed runs, so the timed region measures the command rather than disk I/O
// for its output.
func nfrDevNull(tb testing.TB) *os.File {
	tb.Helper()
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		tb.Fatalf("open %s: %v", os.DevNull, err)
	}
	tb.Cleanup(func() { _ = f.Close() })
	return f
}

// nfrScratchFile returns a read-write file in dir for capturing command output.
func nfrScratchFile(tb testing.TB, dir, name string) *os.File {
	tb.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		tb.Fatalf("create scratch %s: %v", name, err)
	}
	tb.Cleanup(func() { _ = f.Close() })
	return f
}

// nfrCapture runs fn and returns everything it printed to stdout.
func nfrCapture(tb testing.TB, fn func()) []byte {
	tb.Helper()
	f := nfrScratchFile(tb, tb.TempDir(), "capture.txt")
	nfrToStdout(f, fn)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		tb.Fatalf("seek capture: %v", err)
	}
	out, err := io.ReadAll(f)
	if err != nil {
		tb.Fatalf("read capture: %v", err)
	}
	return out
}

// nfrCountTimeline counts the event lines of `awis trace` human output: the
// lines after the "Timeline:" heading that begin with two spaces.
func nfrCountTimeline(out []byte) int {
	n, inTimeline := 0, false
	for _, line := range strings.Split(string(out), "\n") {
		if line == "Timeline:" {
			inTimeline = true
			continue
		}
		if inTimeline && strings.HasPrefix(line, "  ") {
			n++
		}
	}
	return n
}

// nfrNoopHandler is a native Go handler that does no work and no I/O.
type nfrNoopHandler struct{ id string }

func (h nfrNoopHandler) ID() string { return h.id }

func (h nfrNoopHandler) Execute(core.StepContext) (core.StepResult, error) {
	return core.StepResult{Outputs: map[string]any{}}, nil
}

// nfrSubmitDef is the NFR-P-07 workflow: one native step bound to handler.
func nfrSubmitDef(id, handler string) core.WorkflowDefinition {
	return nfrDef(id, core.Step{ID: nfrSubmitStepID, Name: nfrSubmitStepID,
		Type: core.StepTypeNative, Handler: core.HandlerRef(handler)})
}

// nfrSignalDef is the NFR-P-04 workflow: one signal wait step. Its timeout action
// is "fail", but the harness always delivers the signal in time, so it never fires.
func nfrSignalDef(id string) core.WorkflowDefinition {
	return nfrDef(id, core.Step{ID: nfrSignalStepID, Name: nfrSignalStepID,
		Type:       core.StepTypeSignal,
		WaitSignal: &core.WaitConfig{SignalName: nfrSignalName, TimeoutAction: "fail"}})
}

// nfrDef wraps one step as a single-step manual workflow "id@1.0.0" in the
// default namespace, the namespace `awis submit` and `awis start` both use.
func nfrDef(id string, step core.Step) core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: id, Version: "1.0.0",
		Namespace: "default", Name: id,
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{step},
		InitialStep: step.ID, FinalSteps: []string{step.ID},
		Metadata: map[string]any{},
	}
}

// nfrStartRuntime starts the real runtime pull loop over dir/runtime.db,
// registering handlers and workflows the way `awis start` does, and stops the
// loop when tb finishes. It fails tb if the loop returns an unexpected error.
// nfrPinCLI must already be in effect, so the loop's log is captured.
func nfrStartRuntime(tb testing.TB, dir string, defs []core.WorkflowDefinition, handlers ...core.StepHandler) {
	tb.Helper()
	store, err := OpenStorage(dir)
	if err != nil {
		tb.Fatalf("OpenStorage: %v", err)
	}
	rt, err := sdk.NewRuntime(sdk.Config{
		Namespace:    "default",
		Storage:      store,
		TickInterval: nfrLoopTick,
		WorkerID:     "nfr-bench",
	})
	if err != nil {
		tb.Fatalf("sdk.NewRuntime: %v", err)
	}
	for _, h := range handlers {
		if err := rt.RegisterHandler(h); err != nil {
			tb.Fatalf("RegisterHandler %s: %v", h.ID(), err)
		}
	}
	for i := range defs {
		if err := rt.RegisterWorkflow(&defs[i]); err != nil {
			tb.Fatalf("RegisterWorkflow %s: %v", defs[i].ID, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- rt.Start(ctx) }()
	tb.Cleanup(func() {
		cancel()
		if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
			tb.Errorf("runtime loop returned: %v", err)
		}
	})
}

// nfrErrorRecords returns the lines of the JSON log at path that carry level
// ERROR. The engine's JSON handler writes the level as `"level":"ERROR"`.
func nfrErrorRecords(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var recs []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, `"level":"ERROR"`) {
			recs = append(recs, line)
		}
	}
	return recs, nil
}

// nfrOpenPoll opens a read handle on dir/runtime.db for harness polling, so
// polling does not share a connection with the runtime loop.
func nfrOpenPoll(tb testing.TB, dir string) core.StoragePort {
	tb.Helper()
	s, err := OpenStorage(dir)
	if err != nil {
		tb.Fatalf("OpenStorage (poll): %v", err)
	}
	return s
}

// nfrSubmitCLI runs `awis submit <workflowID>` in JSON mode and returns the new
// instance id. The caller times the call; out is a scratch file reused across
// samples, reset here before each call.
func nfrSubmitCLI(tb testing.TB, out *os.File, workflowID string) (core.InstanceID, time.Time) {
	tb.Helper()
	if err := out.Truncate(0); err != nil {
		tb.Fatalf("truncate submit output: %v", err)
	}
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		tb.Fatalf("seek submit output: %v", err)
	}
	t0 := time.Now()
	nfrToStdout(out, func() { runSubmit([]string{workflowID}) })
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		tb.Fatalf("seek submit output: %v", err)
	}
	var got struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.NewDecoder(out).Decode(&got); err != nil {
		tb.Fatalf("decode awis submit --json output: %v", err)
	}
	if got.InstanceID == "" {
		tb.Fatalf("awis submit --json: empty instance_id")
	}
	return core.InstanceID(got.InstanceID), t0
}

// nfrAwaitEvent polls the EventLog of iid until an event of type et for stepID
// appears, then returns it. Only the event's own EmittedAt is used for latency,
// so the poll interval does not enter any measurement.
func nfrAwaitEvent(tb testing.TB, s core.StoragePort, iid core.InstanceID, et core.EventType, stepID string) core.ExecutionEvent {
	tb.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(nfrAwaitTimeout)
	for {
		evs, err := s.ReadEvents(ctx, iid, 0)
		if err != nil {
			tb.Fatalf("ReadEvents %s: %v", iid, err)
		}
		for _, ev := range evs {
			if ev.EventType == et && ev.StepID == stepID {
				return ev
			}
		}
		if time.Now().After(deadline) {
			tb.Fatalf("instance %s: no %s for step %q within %s", iid, et, stepID, nfrAwaitTimeout)
		}
		time.Sleep(time.Millisecond)
	}
}

// nfrAwaitStatus polls iid until its status is want. Untimed setup only.
func nfrAwaitStatus(tb testing.TB, s core.StoragePort, iid core.InstanceID, want core.InstanceStatus) {
	tb.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(nfrAwaitTimeout)
	for {
		inst, err := s.GetInstance(ctx, iid)
		if err != nil {
			tb.Fatalf("GetInstance %s: %v", iid, err)
		}
		if inst.Status == want {
			return
		}
		if time.Now().After(deadline) {
			tb.Fatalf("instance %s: status %q, want %q within %s", iid, inst.Status, want, nfrAwaitTimeout)
		}
		time.Sleep(time.Millisecond)
	}
}

// runSubmitSamples measures NFR-P-07 on a live runtime loop: from the start of
// `awis submit` to the StepStarted timestamp of the workflow's first step. The
// loop picks the instance up on its next tick, so that wait is inside the
// measurement, as it is for a user.
func runSubmitSamples(tb testing.TB, n int) []time.Duration {
	tb.Helper()
	dir := tb.TempDir()
	nfrPinCLI(tb, dir, true)
	nfrStartRuntime(tb, dir, []core.WorkflowDefinition{nfrSubmitDef("nfr-submit", "nfr-noop")},
		nfrNoopHandler{id: "nfr-noop"})
	poll := nfrOpenPoll(tb, dir)
	out := nfrScratchFile(tb, dir, "submit-out.json")

	samples := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		iid, t0 := nfrSubmitCLI(tb, out, "nfr-submit")
		ev := nfrAwaitEvent(tb, poll, iid, core.EventTypeStepStarted, nfrSubmitStepID)
		samples = append(samples, ev.EmittedAt.Sub(t0))
	}
	nfrCheckCLIErrors(tb)
	return samples
}

// runSignalSamples measures NFR-P-04 on a live runtime loop: from the start of
// `awis signal` to the StepCompleted timestamp of the signal step, which is the
// moment the waiting step resumes. Each sample first parks a fresh instance
// (untimed), then delivers the signal (timed).
func runSignalSamples(tb testing.TB, n int) []time.Duration {
	tb.Helper()
	dir := tb.TempDir()
	nfrPinCLI(tb, dir, true)
	nfrStartRuntime(tb, dir, []core.WorkflowDefinition{nfrSignalDef("nfr-signal")})
	poll := nfrOpenPoll(tb, dir)
	out := nfrScratchFile(tb, dir, "signal-submit-out.json")
	null := nfrDevNull(tb)

	samples := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		iid, _ := nfrSubmitCLI(tb, out, "nfr-signal")
		nfrAwaitStatus(tb, poll, iid, core.InstanceStatusWaiting)

		t0 := time.Now()
		nfrToStdout(null, func() { runSignal([]string{string(iid), nfrSignalName}) })
		ev := nfrAwaitEvent(tb, poll, iid, core.EventTypeStepCompleted, nfrSignalStepID)
		samples = append(samples, ev.EmittedAt.Sub(t0))
	}
	nfrCheckCLIErrors(tb)
	return samples
}

// nfrTraceSamples measures `awis trace <id>` over a store whose traced instance
// holds wantEvents events. It first verifies, untimed, that the measured command
// prints the whole timeline; the timed runs then discard their output.
func nfrTraceSamples(tb testing.TB, id core.InstanceID, wantEvents, n int) []time.Duration {
	tb.Helper()
	captured := nfrCapture(tb, func() { runTrace([]string{string(id)}) })
	if got := nfrCountTimeline(captured); got != wantEvents {
		tb.Fatalf("fixture: awis trace %s printed %d timeline events, want %d", id, got, wantEvents)
	}
	null := nfrDevNull(tb)
	samples := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		t0 := time.Now()
		nfrToStdout(null, func() { runTrace([]string{string(id)}) })
		samples = append(samples, time.Since(t0))
	}
	nfrCheckCLIErrors(tb)
	return samples
}

// nfrRebuildSamples measures `awis rebuild-state --all` over the store in the
// current globalDataDir. After the timed runs it verifies, untimed, that the
// rebuilt projection holds one row per fixture instance.
func nfrRebuildSamples(tb testing.TB, wantInstances, n int) []time.Duration {
	tb.Helper()
	null := nfrDevNull(tb)
	samples := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		t0 := time.Now()
		nfrToStdout(null, func() { runRebuildState([]string{"--all"}) })
		samples = append(samples, time.Since(t0))
	}
	nfrCheckCLIErrors(tb)
	store := nfrOpenPoll(tb, globalDataDir)
	insts, err := store.ListInstances(context.Background(), core.InstanceFilter{})
	if err != nil {
		tb.Fatalf("ListInstances after rebuild: %v", err)
	}
	if len(insts) != wantInstances {
		tb.Fatalf("fixture: rebuilt projection has %d instances, want %d", len(insts), wantInstances)
	}
	return samples
}

// benchTrace100K builds one NFR-P-05 shape and reports it. Shape "multi-instance"
// holds 100,000 events over 500 instances; shape "single-instance" holds all
// 100,000 events in one instance, the heaviest trace the store can ask for.
func benchTrace100K(b *testing.B, shape string, instances, n int) {
	dir := b.TempDir()
	nfrBuildFixtureDB(b, dir, nfrFixtureEvents(instances, nfrFixtureTotal))
	nfrPinCLI(b, dir, false)
	traced := nfrInstanceID(0)
	perInstance := nfrFixtureTotal / instances

	samples := nfrTraceSamples(b, traced, perInstance, n)
	sum, err := nfrbench.Summarize(samples)
	if err != nil {
		b.Fatalf("NFR-P-05 summarize: %v", err)
	}
	res, err := nfrbench.NewResult("NFR-P-05", "awis trace query time",
		fmt.Sprintf("%d events across %d instance(s) (shape %s); traced instance %s holds %d events; human output to /dev/null; on-disk WAL store",
			nfrFixtureTotal, instances, shape, traced, perInstance),
		sum,
		nfrbench.Bound{Label: "NFR-P-05 target", Limit: 500 * time.Millisecond, Strict: true,
			Basis: nfrbench.BasisMax, Binding: true},
	)
	if err != nil {
		b.Fatalf("NFR-P-05 verdict: %v", err)
	}
	nfrbench.Report(b, res)
}

// BenchmarkNFRP04SignalDelivery measures NFR-P-04: signal delivery latency
// (PRD §11: "< 200ms"), from `awis signal` to the signal step resuming, on a
// live runtime loop. Verdict on the maximum sample, the strictest reading of a
// bound stated without a percentile.
func BenchmarkNFRP04SignalDelivery(b *testing.B) {
	samples := runSignalSamples(b, nfrSignalSamples)
	sum, err := nfrbench.Summarize(samples)
	if err != nil {
		b.Fatalf("NFR-P-04 summarize: %v", err)
	}
	res, err := nfrbench.NewResult("NFR-P-04", "signal delivery latency (awis signal to step resuming)",
		fmt.Sprintf("1 signal wait step per instance; runtime loop at %s tick (awis start default); latency = awis signal start to StepCompleted of the signal step (EventLog); %d signals; on-disk WAL store",
			nfrLoopTick, nfrSignalSamples),
		sum,
		nfrbench.Bound{Label: "NFR-P-04 target", Limit: 200 * time.Millisecond, Strict: true,
			Basis: nfrbench.BasisMax, Binding: true},
	)
	if err != nil {
		b.Fatalf("NFR-P-04 verdict: %v", err)
	}
	nfrbench.Report(b, res)
}

// BenchmarkNFRP05TraceQuery100K measures NFR-P-05: `awis trace` query time over a
// 100,000-event store (PRD §11: "< 500ms"). Two shapes run, and both must pass.
func BenchmarkNFRP05TraceQuery100K(b *testing.B) {
	b.Run("multi-instance", func(b *testing.B) {
		benchTrace100K(b, "multi-instance", nfrFixtureInstances, nfrTraceSamplesMulti)
	})
	b.Run("single-instance", func(b *testing.B) {
		benchTrace100K(b, "single-instance", 1, nfrTraceSamplesSingle)
	})
}

// BenchmarkNFRP06RebuildState100K measures NFR-P-06: `awis rebuild-state` over a
// 100,000-event store (PRD §11: "< 30s").
func BenchmarkNFRP06RebuildState100K(b *testing.B) {
	dir := b.TempDir()
	nfrBuildFixtureDB(b, dir, nfrFixtureEvents(nfrFixtureInstances, nfrFixtureTotal))
	nfrPinCLI(b, dir, false)

	samples := nfrRebuildSamples(b, nfrFixtureInstances, nfrRebuildSampleCount)
	sum, err := nfrbench.Summarize(samples)
	if err != nil {
		b.Fatalf("NFR-P-06 summarize: %v", err)
	}
	res, err := nfrbench.NewResult("NFR-P-06", "awis rebuild-state --all (full projection rebuild)",
		fmt.Sprintf("%d events across %d instances; each sample rebuilds the whole projection; %d samples; on-disk WAL store",
			nfrFixtureTotal, nfrFixtureInstances, nfrRebuildSampleCount),
		sum,
		nfrbench.Bound{Label: "NFR-P-06 target", Limit: 30 * time.Second, Strict: true,
			Basis: nfrbench.BasisMax, Binding: true},
	)
	if err != nil {
		b.Fatalf("NFR-P-06 verdict: %v", err)
	}
	nfrbench.Report(b, res)
}

// BenchmarkNFRP07SubmitToFirstStep measures NFR-P-07: workflow instance startup
// latency (PRD §11: "< 300ms"), from `awis submit` to the first step's
// StepStarted, on a live runtime loop.
func BenchmarkNFRP07SubmitToFirstStep(b *testing.B) {
	samples := runSubmitSamples(b, nfrSubmitSamples)
	sum, err := nfrbench.Summarize(samples)
	if err != nil {
		b.Fatalf("NFR-P-07 summarize: %v", err)
	}
	res, err := nfrbench.NewResult("NFR-P-07", "workflow instance startup (awis submit to first step started)",
		fmt.Sprintf("1 native step (no-op Go handler) per instance; runtime loop at %s tick (awis start default); latency = awis submit start to StepStarted of the first step (EventLog); %d submits; on-disk WAL store",
			nfrLoopTick, nfrSubmitSamples),
		sum,
		nfrbench.Bound{Label: "NFR-P-07 target", Limit: 300 * time.Millisecond, Strict: true,
			Basis: nfrbench.BasisMax, Binding: true},
	)
	if err != nil {
		b.Fatalf("NFR-P-07 verdict: %v", err)
	}
	nfrbench.Report(b, res)
}
