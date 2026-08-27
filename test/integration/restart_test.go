//go:build integration

package integration

import (
	"testing"
	"time"
)

// TestB0_B15_RestartResumesWaitingInstance: submit signalOnly; signal "go";
// assert it advances to waiting on wait-b; restart the runtime; signal
// "done"; assert the instance reaches 'completed'.
func TestB0_B15_RestartResumesWaitingInstance(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("waiter.yaml", signalOnly)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "waiter")
	// Both waits are 'waiting', so every assertion here is on the STEP, not
	// just the status — otherwise a poll for "waiting" returns instantly on
	// the pre-signal state and proves nothing about the signal.
	f.waitForStep(sub.InstanceID, "waiting", "wait-a", 10*time.Second)

	res := f.run("signal", sub.InstanceID, "go")
	if res.exitCode != 0 {
		t.Fatalf("signal go exited %d: stdout=%s stderr=%s", res.exitCode, res.stdout, res.stderr)
	}
	f.waitForStep(sub.InstanceID, "waiting", "wait-b", 10*time.Second)

	f.restart()

	res = f.run("signal", sub.InstanceID, "done")
	if res.exitCode != 0 {
		t.Fatalf("signal done exited %d: stdout=%s stderr=%s", res.exitCode, res.stdout, res.stderr)
	}
	f.waitForStatus(sub.InstanceID, "completed", 10*time.Second)
}

// TestB1_RebuildStatePreservesWaitingStatus: submit signalOnly; assert
// 'waiting'; stop runtime; 'awis rebuild-state --all'; assert the instance
// status is STILL 'waiting' (not 'running'); start runtime; signal "go";
// assert it advances.
func TestB1_RebuildStatePreservesWaitingStatus(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("waiter.yaml", signalOnly)
	f.start()

	var sub submitOutput
	f.runJSON(&sub, "submit", "waiter")
	f.waitForStep(sub.InstanceID, "waiting", "wait-a", 10*time.Second)

	f.stop()

	res := f.run("rebuild-state", "--all")
	if res.exitCode != 0 {
		t.Fatalf("rebuild-state --all exited %d: stdout=%s stderr=%s", res.exitCode, res.stdout, res.stderr)
	}

	var st statusDetailJSON
	f.runJSON(&st, "status", sub.InstanceID)
	if st.Status != "waiting" {
		t.Fatalf("after rebuild-state (runtime stopped): status = %q, want waiting (not running)", st.Status)
	}

	f.start()

	res = f.run("signal", sub.InstanceID, "go")
	if res.exitCode != 0 {
		t.Fatalf("signal go exited %d: stdout=%s stderr=%s", res.exitCode, res.stdout, res.stderr)
	}
	f.waitForStep(sub.InstanceID, "waiting", "wait-b", 10*time.Second)
}
