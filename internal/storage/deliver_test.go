package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// deliverBase is a fixed delivery timestamp for the B3 transaction tests.
var deliverBase = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// mkWaiting creates a workflow_instances row in status=waiting with a known
// optimistic-lock version and inserts one undelivered inbox signal. It returns
// the instance id, the current version, and the signal name.
func mkWaiting(t *testing.T, s *SQLiteStorage) (core.InstanceID, int, string) {
	t.Helper()
	ctx := context.Background()
	iid := core.InstanceID("inst-1")
	inst := core.WorkflowInstance{
		InstanceID: iid, Namespace: "t", Status: core.InstanceStatusRunning,
		StartedAt: deliverBase, UpdatedAt: deliverBase,
	}
	if err := s.UpsertInstance(ctx, inst, 0); err != nil { // insert → version 1
		t.Fatalf("UpsertInstance insert: %v", err)
	}
	inst.Status = core.InstanceStatusWaiting
	if err := s.UpsertInstance(ctx, inst, 1); err != nil { // → version 2, waiting
		t.Fatalf("UpsertInstance waiting: %v", err)
	}
	if err := s.InsertSignal(ctx, Signal{
		SignalID: "sig-1", InstanceID: iid, SignalName: "go",
		Payload: map[string]any{"by": "alice"}, ReceivedAt: deliverBase,
	}); err != nil {
		t.Fatalf("InsertSignal: %v", err)
	}
	return iid, 2, "go"
}

func mkInput(iid core.InstanceID, name string, expected int) DeliverInput {
	payload, _ := json.Marshal(map[string]any{"signal_name": name, "payload": map[string]any{"by": "alice"}})
	return DeliverInput{
		InstanceID: iid, SignalName: name, EventPayload: payload,
		ExpectedVersion: expected, EventID: "ev-" + string(iid), Now: deliverBase,
	}
}

// TestDeliverSignal_HappyPath proves the three B3 writes commit atomically: the
// inbox flips delivered, one SignalReceived event lands, and the instance
// resumes running.
func TestDeliverSignal_HappyPath(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	iid, ver, name := mkWaiting(t, s)

	out, err := s.DeliverSignal(ctx, mkInput(iid, name, ver))
	if err != nil {
		t.Fatalf("DeliverSignal: %v", err)
	}
	if out != Delivered {
		t.Fatalf("outcome = %v, want Delivered", out)
	}

	undel, _ := s.ListUndeliveredSignals(ctx, iid)
	if len(undel) != 0 {
		t.Fatalf("signal must be delivered; %d still undelivered", len(undel))
	}
	evs, _ := s.ReadEvents(ctx, iid, 0)
	if len(evs) != 1 || evs[0].EventType != core.EventTypeSignalReceived {
		t.Fatalf("want exactly one SignalReceived event, got %+v", evs)
	}
	if evs[0].EmittedAt.UTC() != deliverBase {
		t.Fatalf("emitted_at = %v, want %v", evs[0].EmittedAt.UTC(), deliverBase)
	}
	inst, _ := s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusRunning {
		t.Fatalf("status = %q, want running", inst.Status)
	}
	if !inst.UpdatedAt.Equal(deliverBase) {
		t.Fatalf("updated_at = %v, want %v (same tx now as emitted_at)", inst.UpdatedAt, deliverBase)
	}
}

// TestDeliverSignal_DoubleDeliveryIsNoOp proves re-running delivery after a
// committed delivery changes nothing — the idempotency guard (B3 step 1) matches
// no undelivered row (NFR-R-04).
func TestDeliverSignal_DoubleDeliveryIsNoOp(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	iid, ver, name := mkWaiting(t, s)

	if out, err := s.DeliverSignal(ctx, mkInput(iid, name, ver)); err != nil || out != Delivered {
		t.Fatalf("first delivery: out=%v err=%v", out, err)
	}
	// Second attempt: status is now running (version bumped); the guard alone must
	// short-circuit to a no-op regardless of the passed version.
	out, err := s.DeliverSignal(ctx, mkInput(iid, name, ver+1))
	if err != nil {
		t.Fatalf("second DeliverSignal: %v", err)
	}
	if out != DeliverNoOp {
		t.Fatalf("second outcome = %v, want DeliverNoOp", out)
	}
	evs, _ := s.ReadEvents(ctx, iid, 0)
	if len(evs) != 1 {
		t.Fatalf("want exactly one SignalReceived event after double delivery, got %d", len(evs))
	}
}

// TestDeliverSignal_LockConflictRollsBack proves a forced version skew fails the
// step-3 optimistic lock, ROLLS BACK the whole tx (guard flip AND appended event
// discarded), and leaves the entry undelivered so the next attempt succeeds (B3).
func TestDeliverSignal_LockConflictRollsBack(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	iid, ver, name := mkWaiting(t, s)

	out, err := s.DeliverSignal(ctx, mkInput(iid, name, ver+99)) // forced skew
	if err != nil {
		t.Fatalf("DeliverSignal (skew): %v", err)
	}
	if out != DeliverLockConflict {
		t.Fatalf("outcome = %v, want DeliverLockConflict", out)
	}
	// Rollback: inbox entry still undelivered, no event, still waiting.
	undel, _ := s.ListUndeliveredSignals(ctx, iid)
	if len(undel) != 1 {
		t.Fatalf("lock conflict must leave signal undelivered, got %d undelivered", len(undel))
	}
	evs, _ := s.ReadEvents(ctx, iid, 0)
	if len(evs) != 0 {
		t.Fatalf("lock conflict must append no event, got %d", len(evs))
	}
	inst, _ := s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusWaiting {
		t.Fatalf("status = %q, want waiting (rolled back)", inst.Status)
	}
	// Re-attempt with the correct version succeeds — redelivered on the next tick.
	out2, err := s.DeliverSignal(ctx, mkInput(iid, name, ver))
	if err != nil || out2 != Delivered {
		t.Fatalf("re-attempt: out=%v err=%v, want Delivered", out2, err)
	}
}

// TestDeliverSignal_GuardNoOpWhenNoInbox proves that with no matching undelivered
// inbox row the tx is a no-op (no event, instance untouched).
func TestDeliverSignal_GuardNoOpWhenNoInbox(t *testing.T) {
	s := newSignalStore(t)
	ctx := context.Background()
	iid, ver, _ := mkWaiting(t, s)

	out, err := s.DeliverSignal(ctx, mkInput(iid, "not-awaited", ver))
	if err != nil {
		t.Fatalf("DeliverSignal: %v", err)
	}
	if out != DeliverNoOp {
		t.Fatalf("outcome = %v, want DeliverNoOp", out)
	}
	evs, _ := s.ReadEvents(ctx, iid, 0)
	if len(evs) != 0 {
		t.Fatalf("no-op must append no event, got %d", len(evs))
	}
	inst, _ := s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusWaiting {
		t.Fatalf("status = %q, want waiting (untouched)", inst.Status)
	}
}
