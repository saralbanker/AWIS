//go:build integration

package integration

import (
	"testing"
	"time"
)

// TestCancel_TerminalAndContiguousEventSequence: cancelling a waiting
// instance must reach a terminal 'cancelled' status with a contiguous event
// sequence (1..N, no gaps or repeats — no duplicate/phantom events).
func TestCancel_TerminalAndContiguousEventSequence(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("waiter.yaml", signalOnly)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "waiter")
	f.waitForStatus(sub.InstanceID, "waiting", 5*time.Second)

	res := f.run("cancel", sub.InstanceID)
	if res.exitCode != 0 {
		t.Fatalf("cancel exited %d: stdout=%s stderr=%s", res.exitCode, res.stdout, res.stderr)
	}
	f.waitForStatus(sub.InstanceID, "cancelled", 5*time.Second)

	var trace traceOutputJSON
	f.runJSON(&trace, "trace", "--full", sub.InstanceID)
	for i, ev := range trace.Events {
		want := i + 1
		if ev.Seq != want {
			t.Fatalf("event sequence not contiguous: events[%d].seq = %d, want %d (full sequence: %v)", i, ev.Seq, want, seqList(trace.Events))
		}
	}
}

func seqList(events []traceEventJSON) []int {
	out := make([]int, len(events))
	for i, ev := range events {
		out[i] = ev.Seq
	}
	return out
}

// TestCancel_RunningInstanceReachesTerminal cancels an instance while a step
// is genuinely RUNNING, not merely parked on a signal.
//
// The waiting-instance case above is the easy one: no step is in flight, so
// cancellation is a projection update. This is the harder one — the engine
// has to record the cancellation INTENT durably, terminally fail the
// in-flight step, and reach a terminal status. The intent must survive: it is
// observed on a later tick, not the one that received it.
func TestCancel_RunningInstanceReachesTerminal(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("slow-then-done.yaml", slowThenDone)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "slow-then-done")
	f.waitForStatus(sub.InstanceID, "running", 10*time.Second)

	res := f.run("cancel", sub.InstanceID, "--reason", "integration-test")
	if res.exitCode != 0 {
		t.Fatalf("cancel exited %d: stdout=%s stderr=%s", res.exitCode, res.stdout, res.stderr)
	}
	f.waitForStatus(sub.InstanceID, "cancelled", 30*time.Second)

	var trace traceOutputJSON
	f.runJSON(&trace, "trace", "--full", sub.InstanceID)
	for i, ev := range trace.Events {
		if want := i + 1; ev.Seq != want {
			t.Fatalf("event sequence not contiguous after cancel: events[%d].seq = %d, want %d (full: %v)",
				i, ev.Seq, want, seqList(trace.Events))
		}
	}
	// The step that followed the cancelled one must never have started.
	if got := countStepStarted(trace.Events, "done"); got != 0 {
		t.Errorf("step 'done' started %d times after cancellation, want 0", got)
	}
}

// TestCancel_SurvivesRestart is the B-5 remainder: a cancellation requested
// while the runtime is running must still be honoured after a restart. The
// intent lives in the database, not in the engine's process memory.
func TestCancel_SurvivesRestart(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("waiter.yaml", signalOnly)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "waiter")
	f.waitForStatus(sub.InstanceID, "waiting", 10*time.Second)

	res := f.run("cancel", sub.InstanceID, "--reason", "cancel-then-restart")
	if res.exitCode != 0 {
		t.Fatalf("cancel exited %d: stdout=%s stderr=%s", res.exitCode, res.stdout, res.stderr)
	}

	f.restart()

	// After the restart the instance must be (or become) terminal and must
	// NOT have resumed running.
	got := f.waitForStatus(sub.InstanceID, "cancelled", 20*time.Second)
	if got.Status != "cancelled" {
		t.Fatalf("status after restart = %s, want cancelled", got)
	}
}
