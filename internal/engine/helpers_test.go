package engine

// Shared test helpers for the M06-C1 engine test slice (card M06-C1r). These
// live in `package engine` (internal test) so the join-gate / sequence /
// dispatch unit tests can touch unexported fields — the existing repo idiom
// (storage/*_test.go) permits internal tests.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// errBoom is the canned handler failure used by the failure fixture.
var errBoom = errors.New("boom")

// assertPairs compares an observed (event_type, step_id) stream against want.
func assertPairs(t *testing.T, got, want []pair) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("event count = %d, want %d\n got=%v\nwant=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %v, want %v\n full got=%v", i, got[i], want[i], got)
		}
	}
}

// assertJSONEqual deep-compares two JSON-decoded values.
func assertJSONEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("value mismatch\n got=%#v\nwant=%#v", got, want)
	}
}

// indexer returns a lookup that yields the position of the first (type, step)
// pair in the stream, or -1 if absent.
func indexer(pairs []pair) func(core.EventType, string) int {
	return func(et core.EventType, step string) int {
		for i, p := range pairs {
			if p.Type == et && p.StepID == step {
				return i
			}
		}
		return -1
	}
}

// storageBase is the fixed clock the SQLite adapter reads (claimed_at, cache
// TTL). Determinism of duration_ms lives in the engine clock (fakeClock), not
// here; a fixed storage clock keeps cache entries non-expired within a test.
var storageBase = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

// engineStart is the fake-clock origin for the engine (Config.Clock). Each
// engine now() advances by 10ms so emitted_at / duration_ms are assertable.
var engineStart = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// openStorage opens a fresh temp-file SQLite adapter (existing db_test.go
// idiom) with a fixed clock; the concrete type is returned so RebuildState (a
// *SQLiteStorage method, not a StoragePort method) is reachable.
func openStorage(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	path := filepath.Join(t.TempDir(), "awis.db")
	db, err := storage.Open(path, func() time.Time { return storageBase })
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return storage.NewSQLiteStorage(db, func() time.Time { return storageBase })
}

// discardLogger is a no-op slog logger for tests that do not inspect log output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeClock is a deterministic stepping clock: each now() returns the current
// value, records it as last, then advances by step. It is mutex-guarded so a
// concurrent DISPATCH goroutine reading the clock is race-clean (dispatchOne
// does not call the engine clock, but the guard keeps the helper safe).
type fakeClock struct {
	mu   sync.Mutex
	cur  time.Time
	step time.Duration
	last time.Time
}

func newFakeClock(start time.Time, step time.Duration) *fakeClock {
	return &fakeClock{cur: start, step: step, last: start}
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.cur
	c.last = t
	c.cur = c.cur.Add(c.step)
	return t
}

// stepHandler is the tiny fake StepHandler helper (deliverable 4): configurable
// outputs/error and an atomic call counter (atomic so concurrent DISPATCH in
// the fan-out fixture is race-clean).
type stepHandler struct {
	id      string
	outputs map[string]any
	err     error
	calls   int32
}

func (h *stepHandler) ID() string { return h.id }

func (h *stepHandler) Execute(_ core.StepContext) (core.StepResult, error) {
	atomic.AddInt32(&h.calls, 1)
	if h.err != nil {
		return core.StepResult{}, h.err
	}
	return core.StepResult{Outputs: h.outputs}, nil
}

// isTerminal reports whether a status is terminal (bounded-loop exit condition).
func isTerminal(s core.InstanceStatus) bool {
	switch s {
	case core.InstanceStatusCompleted, core.InstanceStatusFailed,
		core.InstanceStatusCancelled, core.InstanceStatusCompensated,
		core.InstanceStatusCompensationFailed:
		return true
	}
	return false
}

// runToTerminal drives Tick manually (no ticker, no sleeps) until the instance
// reaches a terminal status or the bounded iteration guard trips.
func runToTerminal(t *testing.T, e *Engine, s *storage.SQLiteStorage, iid core.InstanceID) core.WorkflowInstance {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < 200; i++ {
		if err := e.Tick(ctx); err != nil {
			t.Fatalf("Tick: %v", err)
		}
		inst, err := s.GetInstance(ctx, iid)
		if err != nil {
			t.Fatalf("GetInstance: %v", err)
		}
		if isTerminal(inst.Status) {
			return inst
		}
	}
	t.Fatalf("instance %s did not reach a terminal status within 200 ticks", iid)
	return core.WorkflowInstance{}
}

// pair is one (event_type, step_id) tuple for exact-sequence assertions.
type pair struct {
	Type   core.EventType
	StepID string
}

// eventPairs reads the full event stream and projects it to ordered pairs.
func eventPairs(t *testing.T, s *storage.SQLiteStorage, iid core.InstanceID) ([]pair, []core.ExecutionEvent) {
	t.Helper()
	evs, err := s.ReadEvents(context.Background(), iid, 0)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	pairs := make([]pair, 0, len(evs))
	for _, ev := range evs {
		pairs = append(pairs, pair{Type: ev.EventType, StepID: ev.StepID})
	}
	return pairs, evs
}

// findEvent returns the first event matching (type, step); step "" matches any.
func findEvent(t *testing.T, evs []core.ExecutionEvent, et core.EventType, step string) core.ExecutionEvent {
	t.Helper()
	for _, ev := range evs {
		if ev.EventType == et && (step == "" || ev.StepID == step) {
			return ev
		}
	}
	t.Fatalf("event %s/%q not found", et, step)
	return core.ExecutionEvent{}
}

// assertProjectionEquivalence proves the EDR-007 forward-path ≡ RebuildState
// property (VALIDATION_CHECKLIST item 4): the forward-projected row equals the
// row RebuildState reconstructs purely from the event log.
//
// semantics-bearing: forward ≡ rebuild (EDR-007 §10).
func assertProjectionEquivalence(t *testing.T, s *storage.SQLiteStorage, iid core.InstanceID) {
	t.Helper()
	ctx := context.Background()
	fwd, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance (forward): %v", err)
	}
	if err := s.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState: %v", err)
	}
	rb, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance (rebuilt): %v", err)
	}
	if !reflect.DeepEqual(fwd, rb) {
		t.Fatalf("forward projection != RebuildState\n forward=%#v\n rebuild=%#v", fwd, rb)
	}
}

// decodePayload unmarshals an event payload into dst.
func decodePayload(t *testing.T, ev core.ExecutionEvent, dst any) {
	t.Helper()
	if err := json.Unmarshal(ev.Payload, dst); err != nil {
		t.Fatalf("decode %s payload: %v", ev.EventType, err)
	}
}
