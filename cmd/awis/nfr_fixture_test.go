package main

// nfr_fixture_test.go — deterministic EventLog fixture for the M18-C1 trace and
// rebuild-state benchmarks (NFR-P-05, NFR-P-06; CURRENT_TRUTH_LEDGER.md §7 names
// no reusable 100K fixture, and the existing TestRebuild100K generator lives in
// an external test package, so this generator is new). The fixture is built at
// benchmark time and never committed. Generation is pure: no clock, randomness
// or I/O, so the same arguments always yield the same events.

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// nfrFixtureBase is the fixed base timestamp for fixture events.
var nfrFixtureBase = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// nfrInstanceID returns the id of the i-th (0-based) fixture instance.
func nfrInstanceID(i int) core.InstanceID {
	return core.InstanceID(fmt.Sprintf("nfr-i-%06d", i))
}

// nfrFixtureEvents returns exactly total events spread evenly over instances
// instances. total must be a multiple of instances, and each instance must get
// an even count of at least 4 events. Any other shape panics: a fixture shape
// error is a programming error in the harness, not an input to handle.
//
// Instance i has sequence numbers 1..k, where k = total/instances:
//
//	seq 1        WorkflowStarted
//	seq 2..k-1   alternating StepStarted/StepCompleted pairs over step-a, step-b
//	seq k        WorkflowCompleted for even i; for odd i a StepStarted left in
//	             flight, so that instance stays running
func nfrFixtureEvents(instances, total int) []core.ExecutionEvent {
	if instances < 1 || total%instances != 0 {
		panic(fmt.Sprintf("nfrFixtureEvents: total %d is not a multiple of instances %d", total, instances))
	}
	k := total / instances
	if k < 4 || k%2 != 0 {
		panic(fmt.Sprintf("nfrFixtureEvents: %d events per instance; want an even count >= 4", k))
	}
	evs := make([]core.ExecutionEvent, 0, total)
	for i := 0; i < instances; i++ {
		evs = appendNFRInstance(evs, i, k)
	}
	return evs
}

// appendNFRInstance appends the k events of instance i to evs.
func appendNFRInstance(evs []core.ExecutionEvent, i, k int) []core.ExecutionEvent {
	evs = append(evs, nfrEvent(i, 1, core.EventTypeWorkflowStarted, "",
		map[string]any{"inputs": map[string]any{}}))
	for seq := 2; seq <= k-1; seq++ {
		pair := (seq - 2) / 2
		step := nfrStepName(pair)
		if (seq-2)%2 == 0 {
			evs = append(evs, nfrEvent(i, seq, core.EventTypeStepStarted, step,
				map[string]any{"step_id": step, "attempt": 1, "inputs": map[string]any{}}))
		} else {
			evs = append(evs, nfrEvent(i, seq, core.EventTypeStepCompleted, step,
				map[string]any{"step_id": step, "attempt": 1, "outputs": map[string]any{"r": seq}, "duration_ms": 10}))
		}
	}
	if i%2 == 0 {
		evs = append(evs, nfrEvent(i, k, core.EventTypeWorkflowCompleted, "",
			map[string]any{"outputs": map[string]any{}, "duration_ms": 1}))
	} else {
		step := nfrStepName((k - 2) / 2)
		evs = append(evs, nfrEvent(i, k, core.EventTypeStepStarted, step,
			map[string]any{"step_id": step, "attempt": 1, "inputs": map[string]any{}}))
	}
	return evs
}

// nfrStepName alternates between two step ids by pair index.
func nfrStepName(pair int) string {
	if pair%2 == 0 {
		return "step-a"
	}
	return "step-b"
}

// nfrEvent builds one fixture event for instance i at sequence seq.
func nfrEvent(i, seq int, et core.EventType, stepID string, payload map[string]any) core.ExecutionEvent {
	return core.ExecutionEvent{
		EventID:       fmt.Sprintf("nfr-%06d-%06d", i, seq),
		InstanceID:    nfrInstanceID(i),
		Namespace:     "default",
		EventType:     et,
		StepID:        stepID,
		Payload:       nfrJSON(payload),
		EmittedAt:     nfrFixtureBase.Add(time.Duration(seq) * time.Second),
		SequenceNum:   seq,
		SchemaVersion: 1,
	}
}

// nfrJSON marshals a fixture payload. The payloads are plain maps of basic
// types, so marshalling cannot fail; a failure is a harness bug and panics.
func nfrJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("nfrJSON: " + err.Error())
	}
	return b
}

// nfrBuildFixtureDB writes events into dir/runtime.db through the production
// append path, then runs the production RebuildState so every instance has a
// projection row (awis trace reads the projection before the EventLog). This is
// untimed setup.
func nfrBuildFixtureDB(tb testing.TB, dir string, events []core.ExecutionEvent) {
	tb.Helper()
	db, err := storage.Open(filepath.Join(dir, "runtime.db"), time.Now)
	if err != nil {
		tb.Fatalf("storage.Open fixture db: %v", err)
	}
	s := storage.NewSQLiteStorage(db, time.Now)
	ctx := context.Background()
	for _, ev := range events {
		if err := s.AppendEvent(ctx, ev); err != nil {
			_ = db.Close()
			tb.Fatalf("fixture AppendEvent %s seq %d: %v", ev.InstanceID, ev.SequenceNum, err)
		}
	}
	if err := s.RebuildState(ctx); err != nil {
		_ = db.Close()
		tb.Fatalf("fixture RebuildState: %v", err)
	}
	if err := db.Close(); err != nil {
		tb.Fatalf("close fixture db: %v", err)
	}
}
