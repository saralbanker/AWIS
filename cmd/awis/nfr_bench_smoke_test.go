package main

// nfr_bench_smoke_test.go holds the unit tests for the M18-C1 harness and runs
// its CLI and runtime measurement paths with small fixtures and sample counts
// under `make test`. The real 100K-event measurements run only under
// `make bench`.

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// mustPanic fails t unless fn panics.
func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: want panic, got none", name)
		}
	}()
	fn()
}

func TestNFRFixtureEventsExactCount100K(t *testing.T) {
	const instances = 500
	evs := nfrFixtureEvents(instances, nfrFixtureTotal)
	if len(evs) != nfrFixtureTotal {
		t.Fatalf("events = %d, want %d", len(evs), nfrFixtureTotal)
	}
	perInstance := nfrFixtureTotal / instances
	// Events are generated instance by instance, so each instance is one run.
	for i := 0; i < instances; i++ {
		run := evs[i*perInstance : (i+1)*perInstance]
		if run[0].InstanceID != nfrInstanceID(i) {
			t.Fatalf("instance %d: run starts with %s", i, run[0].InstanceID)
		}
		for j, ev := range run {
			if ev.InstanceID != nfrInstanceID(i) || ev.SequenceNum != j+1 {
				t.Fatalf("instance %d event %d: id %s seq %d, want seq %d",
					i, j, ev.InstanceID, ev.SequenceNum, j+1)
			}
		}
		if run[0].EventType != core.EventTypeWorkflowStarted {
			t.Fatalf("instance %d: first event %s, want WorkflowStarted", i, run[0].EventType)
		}
		last := run[len(run)-1].EventType
		want := core.EventTypeStepStarted
		if i%2 == 0 {
			want = core.EventTypeWorkflowCompleted
		}
		if last != want {
			t.Fatalf("instance %d: last event %s, want %s", i, last, want)
		}
	}
}

func TestNFRFixtureEventsDeterministic(t *testing.T) {
	a := nfrFixtureEvents(7, 70)
	b := nfrFixtureEvents(7, 70)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same arguments produced different events")
	}
}

func TestNFRFixtureEventsShape(t *testing.T) {
	// k=8 per instance. Instance 0 (even) closes with WorkflowCompleted; instance 1
	// (odd) leaves its next step (pair 3, step-b) in flight.
	evs := nfrFixtureEvents(2, 16)
	type row struct {
		et   core.EventType
		step string
	}
	// Seq 1..7 are shared by both instances.
	shared := []row{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "step-a"},
		{core.EventTypeStepCompleted, "step-a"},
		{core.EventTypeStepStarted, "step-b"},
		{core.EventTypeStepCompleted, "step-b"},
		{core.EventTypeStepStarted, "step-a"},
		{core.EventTypeStepCompleted, "step-a"},
	}
	closing := []row{
		{core.EventTypeWorkflowCompleted, ""}, // instance 0 (even)
		{core.EventTypeStepStarted, "step-b"}, // instance 1 (odd): step left in flight
	}
	for i := 0; i < 2; i++ {
		run := evs[i*8 : (i+1)*8]
		for j, w := range shared {
			if run[j].EventType != w.et || run[j].StepID != w.step {
				t.Errorf("instance %d seq %d: got %s/%q, want %s/%q",
					i, j+1, run[j].EventType, run[j].StepID, w.et, w.step)
			}
		}
		if last := run[7]; last.EventType != closing[i].et || last.StepID != closing[i].step {
			t.Errorf("instance %d seq 8: got %s/%q, want %s/%q",
				i, last.EventType, last.StepID, closing[i].et, closing[i].step)
		}
	}
}

func TestNFRFixtureEventsRejectsBadShape(t *testing.T) {
	mustPanic(t, "zero instances", func() { nfrFixtureEvents(0, 10) })
	mustPanic(t, "total not a multiple", func() { nfrFixtureEvents(3, 10) })
	mustPanic(t, "odd events per instance", func() { nfrFixtureEvents(1, 7) })
	mustPanic(t, "fewer than 4 per instance", func() { nfrFixtureEvents(1, 2) })
}

func TestNFRCountTimeline(t *testing.T) {
	// Two event lines count. An indented line before the heading, a blank line,
	// and an unindented footer after the timeline do not.
	out := []byte("Trace: x / i\n  indented header\nTimeline:\n" +
		"  00:00  ● WorkflowStarted\n  00:01  ► StepStarted a\n\nfooter line\n")
	if got := nfrCountTimeline(out); got != 2 {
		t.Errorf("nfrCountTimeline = %d, want 2", got)
	}
	if got := nfrCountTimeline([]byte("no timeline here\n  indented\n")); got != 0 {
		t.Errorf("nfrCountTimeline without heading = %d, want 0", got)
	}
}

func TestNFRTraceAndRebuildSmoke(t *testing.T) {
	dir := t.TempDir()
	nfrBuildFixtureDB(t, dir, nfrFixtureEvents(10, 2000))
	nfrPinCLI(t, dir, false)

	// Instance 0 holds 200 events; nfrTraceSamples fails if awis trace prints
	// any other count.
	if got := nfrTraceSamples(t, nfrInstanceID(0), 200, 2); len(got) != 2 {
		t.Fatalf("trace samples = %d, want 2", len(got))
	}
	if got := nfrRebuildSamples(t, 10, 2); len(got) != 2 {
		t.Fatalf("rebuild samples = %d, want 2", len(got))
	}
}

func TestNFRErrorRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cli-stderr.log")
	log := `{"level":"INFO","msg":"emit"}
{"level":"WARN","msg":"signal timeout scan"}
{"level":"ERROR","msg":"process instance failed"}

{"level":"ERROR","msg":"tick failed"}
`
	if err := os.WriteFile(path, []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	recs, err := nfrErrorRecords(path)
	if err != nil {
		t.Fatalf("nfrErrorRecords: %v", err)
	}
	want := []string{
		`{"level":"ERROR","msg":"process instance failed"}`,
		`{"level":"ERROR","msg":"tick failed"}`,
	}
	if !reflect.DeepEqual(recs, want) {
		t.Errorf("error records = %q, want %q", recs, want)
	}
	if recs, err := nfrErrorRecords(filepath.Join(t.TempDir(), "absent.log")); err == nil || recs != nil {
		t.Errorf("missing log: got (%v, %v), want (nil, error)", recs, err)
	}
}

func TestNFRSubmitAndSignalOnLiveLoopSmoke(t *testing.T) {
	// Each latency must be positive and well inside the await timeout; a value
	// outside that range means the harness measured the wrong events.
	check := func(name string, samples []time.Duration) {
		t.Helper()
		if len(samples) != 2 {
			t.Fatalf("%s samples = %d, want 2", name, len(samples))
		}
		for i, d := range samples {
			if d <= 0 || d >= nfrAwaitTimeout {
				t.Errorf("%s sample %d = %v, want in (0, %v)", name, i, d, nfrAwaitTimeout)
			}
		}
	}
	check("NFR-P-07", runSubmitSamples(t, 2))
	check("NFR-P-04", runSignalSamples(t, 2))
}
