package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// seedInstance inserts a fresh workflow_instances row (version 1,
// cancellation_requested 0) and returns the SQLiteStorage wrapping db.
func seedInstance(t *testing.T, status core.InstanceStatus) (*SQLiteStorage, core.InstanceID) {
	t.Helper()
	db := openTestDB(t)
	s := NewSQLiteStorage(db, func() time.Time { return time.Unix(0, 0).UTC() })
	iid := core.InstanceID("inst-cancel-1")
	inst := core.WorkflowInstance{
		InstanceID:   iid,
		DefinitionID: "d",
		Namespace:    "t",
		Status:       status,
		CurrentSteps: []string{},
		Variables:    map[string]any{},
	}
	if err := s.UpsertInstance(context.Background(), inst, 0); err != nil {
		t.Fatalf("UpsertInstance seed: %v", err)
	}
	return s, iid
}

func TestCancellationRequested_DefaultsFalse(t *testing.T) {
	s, iid := seedInstance(t, core.InstanceStatusRunning)
	got, err := s.CancellationRequested(context.Background(), iid)
	if err != nil {
		t.Fatalf("CancellationRequested: %v", err)
	}
	if got {
		t.Fatalf("cancellation_requested = true, want false at seed")
	}
}

func TestSetCancellationRequested_SetsFlag(t *testing.T) {
	s, iid := seedInstance(t, core.InstanceStatusRunning)
	if err := s.SetCancellationRequested(context.Background(), iid); err != nil {
		t.Fatalf("SetCancellationRequested: %v", err)
	}
	got, err := s.CancellationRequested(context.Background(), iid)
	if err != nil {
		t.Fatalf("CancellationRequested: %v", err)
	}
	if !got {
		t.Fatalf("cancellation_requested = false, want true after set")
	}
}

// The flag must not alter status or the projected user fields (CONTRA-6).
func TestSetCancellationRequested_LeavesStatusUntouched(t *testing.T) {
	s, iid := seedInstance(t, core.InstanceStatusRunning)
	if err := s.SetCancellationRequested(context.Background(), iid); err != nil {
		t.Fatalf("SetCancellationRequested: %v", err)
	}
	inst, err := s.GetInstance(context.Background(), iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if inst.Status != core.InstanceStatusRunning {
		t.Fatalf("status = %q, want running (flag must not change status)", inst.Status)
	}
}

func TestSetCancellationRequested_NotFound(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	err := s.SetCancellationRequested(context.Background(), "no-such-instance")
	if !errors.Is(err, ErrInstanceNotFound) {
		t.Fatalf("err = %v, want ErrInstanceNotFound", err)
	}
}

func TestCancellationRequested_NotFound(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	_, err := s.CancellationRequested(context.Background(), "no-such-instance")
	if !errors.Is(err, ErrInstanceNotFound) {
		t.Fatalf("err = %v, want ErrInstanceNotFound", err)
	}
}

func TestSetCancellationIntent_Durability(t *testing.T) {
	s, iid := seedInstance(t, core.InstanceStatusRunning)
	reason := "user requested cancel with compensation"
	if err := s.SetCancellationIntent(context.Background(), iid, reason, true); err != nil {
		t.Fatalf("SetCancellationIntent: %v", err)
	}
	r, comp, req, err := s.CancellationIntent(context.Background(), iid)
	if err != nil {
		t.Fatalf("CancellationIntent: %v", err)
	}
	if !req {
		t.Fatalf("requested = false, want true")
	}
	if r != reason {
		t.Fatalf("reason = %q, want %q", r, reason)
	}
	if !comp {
		t.Fatalf("compensate = false, want true")
	}
}
