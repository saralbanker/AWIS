package main

import (
	"context"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// TestBuildActiveJSONPopulatesWaitFields is the B-28 regression.
//
// signal_name and timeout_remaining_s are part of the published status JSON
// schema (TDS-07 §4) but were declared and never assigned, so they were
// permanently null. An instance reported as `waiting` gave no indication of
// WHICH signal it awaited — an operator could not tell what to pass to
// `awis signal` without opening the workflow YAML.
func TestBuildActiveJSONPopulatesWaitFields(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	timeoutAt := now.Add(90 * time.Second)

	inst := core.WorkflowInstance{
		InstanceID:   "i1",
		DefinitionID: "with-signal",
		Namespace:    "examples",
		Status:       core.InstanceStatusWaiting,
		CurrentSteps: []string{"await-approval"},
		StartedAt:    now.Add(-10 * time.Second),
	}
	wait := &storage.WaitRecord{
		InstanceID: "i1", StepID: "await-approval",
		SignalName: "approved", TimeoutAt: &timeoutAt, TimeoutAction: "cancel",
	}

	got := buildActiveJSON(now, inst, wait)

	if got.SignalName == nil || *got.SignalName != "approved" {
		t.Errorf("signal_name = %v, want %q", got.SignalName, "approved")
	}
	if got.TimeoutRemainingS == nil || *got.TimeoutRemainingS != 90 {
		t.Errorf("timeout_remaining_s = %v, want 90", got.TimeoutRemainingS)
	}
}

// TestBuildActiveJSONWaitFieldsNilWhenNotWaiting: a running instance has no
// wait to describe, and the fields must stay null rather than carry over.
func TestBuildActiveJSONWaitFieldsNilWhenNotWaiting(t *testing.T) {
	now := time.Now()
	inst := core.WorkflowInstance{
		InstanceID: "i2", Status: core.InstanceStatusRunning,
		CurrentSteps: []string{"work"}, StartedAt: now,
	}
	got := buildActiveJSON(now, inst, nil)
	if got.SignalName != nil {
		t.Errorf("signal_name = %v on a running instance, want null", *got.SignalName)
	}
	if got.TimeoutRemainingS != nil {
		t.Errorf("timeout_remaining_s = %v on a running instance, want null", *got.TimeoutRemainingS)
	}
}

// TestBuildActiveJSONUnboundedWaitHasNoTimeout: a wait with no timeout_at
// must report a signal name but a null countdown, not a zero one — zero would
// read as "about to time out".
func TestBuildActiveJSONUnboundedWaitHasNoTimeout(t *testing.T) {
	now := time.Now()
	inst := core.WorkflowInstance{
		InstanceID: "i3", Status: core.InstanceStatusWaiting,
		CurrentSteps: []string{"w"}, StartedAt: now,
	}
	got := buildActiveJSON(now, inst, &storage.WaitRecord{
		InstanceID: "i3", StepID: "w", SignalName: "go", TimeoutAt: nil,
	})
	if got.SignalName == nil || *got.SignalName != "go" {
		t.Errorf("signal_name = %v, want %q", got.SignalName, "go")
	}
	if got.TimeoutRemainingS != nil {
		t.Errorf("timeout_remaining_s = %v for an unbounded wait, want null", *got.TimeoutRemainingS)
	}
}

// TestBuildActiveJSONOverdueWaitFloorsAtZero: a wait already past its
// deadline (the timeout scan has not run yet) must not report a NEGATIVE
// countdown, which reads as a bug to whoever sees it.
func TestBuildActiveJSONOverdueWaitFloorsAtZero(t *testing.T) {
	now := time.Now()
	past := now.Add(-5 * time.Minute)
	inst := core.WorkflowInstance{
		InstanceID: "i4", Status: core.InstanceStatusWaiting,
		CurrentSteps: []string{"w"}, StartedAt: now,
	}
	got := buildActiveJSON(now, inst, &storage.WaitRecord{
		InstanceID: "i4", StepID: "w", SignalName: "go", TimeoutAt: &past,
	})
	if got.TimeoutRemainingS == nil || *got.TimeoutRemainingS != 0 {
		t.Errorf("timeout_remaining_s = %v for an overdue wait, want 0", got.TimeoutRemainingS)
	}
}

// TestLookupWaitPrefersTheCurrentStep: an instance joining several signal
// steps holds several wait records. signal_name must describe the SAME step
// that current_step reports, or the two fields contradict each other.
func TestLookupWaitPrefersTheCurrentStep(t *testing.T) {
	store := &fakeWaitStore{records: []storage.WaitRecord{
		{InstanceID: "i5", StepID: "wait-a", SignalName: "alpha"},
		{InstanceID: "i5", StepID: "wait-b", SignalName: "beta"},
	}}
	inst := core.WorkflowInstance{
		InstanceID: "i5", Status: core.InstanceStatusWaiting,
		CurrentSteps: []string{"wait-b"},
	}
	got := lookupWait(context.Background(), store, inst)
	if got == nil || got.SignalName != "beta" {
		t.Fatalf("lookupWait picked %v, want the record for current_step wait-b (beta)", got)
	}
}

// TestLookupWaitSkipsNonWaitingInstances: the query must not run at all for
// an instance that is not parked, so the cost stays bounded by the number of
// genuinely waiting instances.
func TestLookupWaitSkipsNonWaitingInstances(t *testing.T) {
	store := &fakeWaitStore{records: []storage.WaitRecord{
		{InstanceID: "i6", StepID: "w", SignalName: "go"},
	}}
	inst := core.WorkflowInstance{InstanceID: "i6", Status: core.InstanceStatusRunning}
	if got := lookupWait(context.Background(), store, inst); got != nil {
		t.Errorf("lookupWait returned %v for a running instance, want nil", got)
	}
	if store.calls != 0 {
		t.Errorf("storage was queried %d times for a non-waiting instance, want 0", store.calls)
	}
}

// TestLookupWaitDegradesOnStoreWithoutCapability: a store lacking the
// additive accessor must yield nil, not panic or fail a read-only command.
func TestLookupWaitDegradesOnStoreWithoutCapability(t *testing.T) {
	inst := core.WorkflowInstance{InstanceID: "i7", Status: core.InstanceStatusWaiting}
	if got := lookupWait(context.Background(), struct{}{}, inst); got != nil {
		t.Errorf("lookupWait = %v on a store without the capability, want nil", got)
	}
}

// fakeWaitStore implements waitRecordStore and counts calls.
type fakeWaitStore struct {
	records []storage.WaitRecord
	calls   int
}

func (f *fakeWaitStore) ListWaitRecordsByInstance(_ context.Context, iid core.InstanceID) ([]storage.WaitRecord, error) {
	f.calls++
	var out []storage.WaitRecord
	for _, r := range f.records {
		if r.InstanceID == iid {
			out = append(out, r)
		}
	}
	return out, nil
}
