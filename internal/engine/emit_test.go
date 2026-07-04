package engine

// Sequence assignment + ErrSequenceViolation recovery tests (EDR-005).

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// rawEvent builds a minimal appendable event (EventID/SchemaVersion/SequenceNum
// are filled by appendEvent for the engine path).
func rawEvent(iid core.InstanceID) *core.ExecutionEvent {
	payload, _ := json.Marshal(map[string]any{})
	return &core.ExecutionEvent{
		InstanceID: iid,
		Namespace:  "t",
		EventType:  core.EventTypeSignalReceived,
		Payload:    payload,
		EmittedAt:  engineStart,
	}
}

func TestAppendEvent_ConsecutiveSequenceNumbers(t *testing.T) {
	s := openStorage(t)
	e := New(s, nil, Config{Clock: func() time.Time { return engineStart }}, discardLogger())
	ctx := context.Background()
	const iid core.InstanceID = "seq-inst"

	for want := 1; want <= 3; want++ {
		ev := rawEvent(iid)
		if err := e.appendEvent(ctx, ev); err != nil {
			t.Fatalf("appendEvent #%d: %v", want, err)
		}
		if ev.SequenceNum != want {
			t.Fatalf("append #%d assigned sequence_num %d, want %d", want, ev.SequenceNum, want)
		}
	}

	evs, err := s.ReadEvents(ctx, iid, 0)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(evs) != 3 {
		t.Fatalf("stored %d events, want 3", len(evs))
	}
	for i, ev := range evs {
		if ev.SequenceNum != i+1 {
			t.Fatalf("stored event %d has sequence_num %d, want %d", i, ev.SequenceNum, i+1)
		}
	}
}

// TestAppendEvent_StaleCounterReloadRecovery seeds the log with an out-of-band
// event at seq 5 (behind the engine's back), leaving the engine's in-memory
// counter stale at 0. The next append attempts seq 1, is rejected with
// ErrSequenceViolation, then reloads MAX+1 (=6) and retries once successfully
// (EDR-005 recovery path).
func TestAppendEvent_StaleCounterReloadRecovery(t *testing.T) {
	s := openStorage(t)
	e := New(s, nil, Config{Clock: func() time.Time { return engineStart }}, discardLogger())
	ctx := context.Background()
	const iid core.InstanceID = "stale-inst"

	// Out-of-band append directly through storage, bypassing the engine counter.
	oob := rawEvent(iid)
	oob.EventID = "oob-5"
	oob.SchemaVersion = 1
	oob.SequenceNum = 5
	if err := s.AppendEvent(ctx, *oob); err != nil {
		t.Fatalf("out-of-band AppendEvent: %v", err)
	}

	// Engine's e.seq[iid] is still 0 ⇒ it will try seq 1, violate, reload, retry.
	ev := rawEvent(iid)
	if err := e.appendEvent(ctx, ev); err != nil {
		t.Fatalf("appendEvent recovery: %v", err)
	}
	if ev.SequenceNum != 6 {
		t.Fatalf("recovered sequence_num = %d, want 6 (MAX+1)", ev.SequenceNum)
	}

	// A subsequent append continues monotonically from the reloaded counter.
	ev2 := rawEvent(iid)
	if err := e.appendEvent(ctx, ev2); err != nil {
		t.Fatalf("appendEvent after recovery: %v", err)
	}
	if ev2.SequenceNum != 7 {
		t.Fatalf("post-recovery sequence_num = %d, want 7", ev2.SequenceNum)
	}
}
