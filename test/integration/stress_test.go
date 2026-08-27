//go:build integration

package integration

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestStress_ConcurrentSubmissionsAllReachTerminal drives many instances
// through the shipped binary at once.
//
// This is the concurrency probe for the engine's claim/dispatch/settle path.
// Every instance must reach a TERMINAL status — the failure mode being
// guarded against is not a crash but a HANG: B-4's re-activation loop, a lost
// step claim, or an optimistic-concurrency conflict all leave an instance
// permanently 'running' with nothing running, which no single-instance test
// reliably surfaces.
func TestStress_ConcurrentSubmissionsAllReachTerminal(t *testing.T) {
	const (
		instances   = 30
		concurrency = 8
	)

	f := newProject(t)
	f.writeWorkflow("linear.yaml", linearNative)
	f.start()

	ids := make([]string, instances)
	errs := make([]error, instances)
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var sub submitOutput
			// runJSONErr, not runJSON: t.Fatalf from a spawned goroutine
			// would kill the goroutine without failing the test.
			if err := f.runJSONErr(&sub, "submit", "linear-native", "--input", fmt.Sprintf("name=w%d", i)); err != nil {
				errs[i] = err
				return
			}
			ids[i] = sub.InstanceID
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent submit %d failed: %v", i, err)
		}
	}

	// Generous but bounded: a hang is the defect being hunted, so the deadline
	// has to be long enough that slowness alone does not fail the test.
	deadline := 90 * time.Second
	for i, id := range ids {
		if id == "" {
			t.Fatalf("instance %d has no id — submit did not return one", i)
		}
		f.waitForStatus(id, "completed", deadline)
	}

	// Distinct ids: a collision would mean two submissions shared an instance.
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate instance id %s across concurrent submissions", id)
		}
		seen[id] = true
	}
}

// TestStress_EventSequencesStayContiguousUnderLoad checks the durability
// invariant that concurrency is most likely to break: per-instance
// sequence_num must be a contiguous 1..N with no gaps and no repeats, even
// when many instances are being claimed and settled at once.
//
// A gap means an event was appended and its projection lost; a repeat means
// two writers agreed on the same sequence. Either is corruption of the
// append-only source of truth, and neither is visible from status alone.
func TestStress_EventSequencesStayContiguousUnderLoad(t *testing.T) {
	const instances = 15

	f := newProject(t)
	f.writeWorkflow("linear.yaml", linearNative)
	f.start()

	ids := make([]string, instances)
	errs := make([]error, instances)
	var wg sync.WaitGroup
	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var sub submitOutput
			if err := f.runJSONErr(&sub, "submit", "linear-native", "--input", fmt.Sprintf("name=s%d", i)); err != nil {
				errs[i] = err
				return
			}
			ids[i] = sub.InstanceID
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent submit %d failed: %v", i, err)
		}
	}

	for _, id := range ids {
		f.waitForStatus(id, "completed", 90*time.Second)
	}

	for _, id := range ids {
		var trace traceOutputJSON
		f.runJSON(&trace, "trace", "--full", id)
		if len(trace.Events) == 0 {
			t.Errorf("%s: no events", id)
			continue
		}
		for i, ev := range trace.Events {
			if want := i + 1; ev.Seq != want {
				t.Errorf("%s: events[%d].seq = %d, want %d (full: %v)",
					id, i, ev.Seq, want, seqList(trace.Events))
				break
			}
		}
	}
}

// TestStress_MixedOutcomesUnderLoad runs succeeding, failing, retrying and
// recovering workflows simultaneously. The failure paths share the engine's
// pending/failed bookkeeping with the success path, so exercising them
// together is what catches one instance's terminal failure leaking into
// another's activatable set.
func TestStress_MixedOutcomesUnderLoad(t *testing.T) {
	f := newProject(t)
	f.writeWorkflow("linear.yaml", linearNative)
	f.writeWorkflow("terminal-fail.yaml", terminalFail)
	f.writeWorkflow("retry-flaky.yaml", retryFlaky)
	f.writeWorkflow("with-on-error.yaml", withOnError)
	f.start()

	kinds := []struct {
		workflow string
		want     string
	}{
		{"linear-native", "completed"},
		{"terminal-fail", "failed"},
		{"retry-flaky", "completed"},
		{"with-on-error", "completed"},
	}

	type submitted struct{ id, want string }
	var (
		mu   sync.Mutex
		subs []submitted
		errs []error
		wg   sync.WaitGroup
	)

	for round := 0; round < 4; round++ {
		for _, k := range kinds {
			wg.Add(1)
			go func(workflow, want string) {
				defer wg.Done()
				var sub submitOutput
				if err := f.runJSONErr(&sub, "submit", workflow); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					return
				}
				mu.Lock()
				subs = append(subs, submitted{sub.InstanceID, want})
				mu.Unlock()
			}(k.workflow, k.want)
		}
	}
	wg.Wait()

	for _, err := range errs {
		t.Fatalf("concurrent submit failed: %v", err)
	}
	if len(subs) != 4*len(kinds) {
		t.Fatalf("only %d of %d submissions succeeded", len(subs), 4*len(kinds))
	}

	for _, s := range subs {
		f.waitForStatus(s.id, s.want, 90*time.Second)
	}
}
