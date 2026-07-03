// Package storagetest provides an adapter-agnostic contract test suite for
// core.StoragePort implementations. It is designed to be reused verbatim for
// any adapter (SQLite M02, Postgres V2, etc.) — the only variation is the
// factory function supplied by the caller.
//
// Clock injection: for cache TTL tests the factory must return a
// *FakeClockStorage (wrapping the real adapter). Use WrapWithFakeClock to
// construct it with a shared *time.Time pointer so SetNow propagates into the
// adapter's clock function.
//
// Usage:
//
//	func TestStorageContract(t *testing.T) {
//	    storagetest.Run(t, func(t *testing.T) core.StoragePort {
//	        db := // open adapter ...
//	        clockPtr := new(time.Time)
//	        *clockPtr = time.Now()
//	        inner := storage.NewSQLiteStorage(db, func() time.Time { return *clockPtr })
//	        return storagetest.WrapWithFakeClock(inner, clockPtr)
//	    })
//	}
package storagetest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// FakeClockStorage wraps a core.StoragePort and exposes SetNow so the suite
// can advance time to test TTL expiry. The inner adapter must be constructed
// with a clock that reads from the same *time.Time pointer as clockPtr.
type FakeClockStorage struct {
	core.StoragePort
	clockPtr *time.Time
}

// WrapWithFakeClock wraps inner with a fake clock. clockPtr must be the same
// pointer the inner adapter's clock function reads from.
func WrapWithFakeClock(inner core.StoragePort, clockPtr *time.Time) *FakeClockStorage {
	return &FakeClockStorage{StoragePort: inner, clockPtr: clockPtr}
}

// SetNow updates the shared clock pointer, which the inner adapter's clock
// function reads on subsequent calls.
func (f *FakeClockStorage) SetNow(t time.Time) {
	*f.clockPtr = t
}

// Run executes the full StoragePort contract suite against the adapter returned
// by factory. Each subtest receives its own adapter instance.
func Run(t *testing.T, factory func(t *testing.T) core.StoragePort) {
	t.Helper()

	t.Run("AppendReadRoundtrip", func(t *testing.T) {
		testAppendReadRoundtrip(t, factory(t))
	})
	t.Run("SequenceMonotonicityRejection", func(t *testing.T) {
		testSequenceMonotonicityRejection(t, factory(t))
	})
	t.Run("ReadEventsFromSeqFilterAndOrder", func(t *testing.T) {
		testReadEventsFromSeqFilterAndOrder(t, factory(t))
	})
	t.Run("ReadEventRangeNamespaceIsolation", func(t *testing.T) {
		testReadEventRangeNamespaceIsolation(t, factory(t))
	})
	t.Run("ReadEventRangeTimeBounds", func(t *testing.T) {
		testReadEventRangeTimeBounds(t, factory(t))
	})
	t.Run("RegistryImmutability", func(t *testing.T) {
		testRegistryImmutability(t, factory(t))
	})
	t.Run("RegistryJSONRoundtrip", func(t *testing.T) {
		testRegistryJSONRoundtrip(t, factory(t))
	})
	t.Run("ListWorkflowsNamespacePredicate", func(t *testing.T) {
		testListWorkflowsNamespacePredicate(t, factory(t))
	})
	t.Run("CacheTTLExpiry", func(t *testing.T) {
		testCacheTTLExpiry(t, factory(t))
	})
	t.Run("CacheAbsentKeyMiss", func(t *testing.T) {
		testCacheAbsentKeyMiss(t, factory(t))
	})
	t.Run("SchemaVersionZeroNormalized", func(t *testing.T) {
		testSchemaVersionZeroNormalized(t, factory(t))
	})
	// StateStore subtests (M03).
	t.Run("UpsertCreateThenGet", func(t *testing.T) {
		testUpsertCreateThenGet(t, factory(t))
	})
	t.Run("UpsertOptimisticConflict", func(t *testing.T) {
		testUpsertOptimisticConflict(t, factory(t))
	})
	t.Run("UpsertUpdateAdvancesVersion", func(t *testing.T) {
		testUpsertUpdateAdvancesVersion(t, factory(t))
	})
	t.Run("ListInstancesNamespacePredicate", func(t *testing.T) {
		testListInstancesNamespacePredicate(t, factory(t))
	})
	t.Run("ListInstancesStatusPredicate", func(t *testing.T) {
		testListInstancesStatusPredicate(t, factory(t))
	})
	t.Run("ClaimAtMostOnce", func(t *testing.T) {
		testClaimAtMostOnce(t, factory(t))
	})
	t.Run("ClaimBumpsVersion", func(t *testing.T) {
		testClaimBumpsVersion(t, factory(t))
	})
	t.Run("ClaimInstanceAbsentErrors", func(t *testing.T) {
		testClaimInstanceAbsentErrors(t, factory(t))
	})
	t.Run("TerminalUpsertReleasesClaims", func(t *testing.T) {
		testTerminalUpsertReleasesClaims(t, factory(t))
	})
	// Note: StubsReturnErrNotImplemented has been removed — all four StateStore
	// methods are implemented in M03 and no stubs remain.
}

// ── helpers ───────────────────────────────────────────────────────────────────

func bg() context.Context { return context.Background() }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := 20
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

func makeEvent(instanceID core.InstanceID, namespace string, seq int, emittedAt time.Time) core.ExecutionEvent {
	return core.ExecutionEvent{
		EventID:       "evt-" + string(instanceID) + "-" + itoa(seq),
		InstanceID:    instanceID,
		Namespace:     namespace,
		EventType:     core.EventTypeWorkflowStarted,
		StepID:        "",
		Payload:       json.RawMessage(`{"inputs":{}}`),
		EmittedAt:     emittedAt,
		SequenceNum:   seq,
		SchemaVersion: 1,
	}
}

// testSchemaVersionZeroNormalized asserts a zero-valued SchemaVersion is
// persisted as 1, mirroring the DDL default (TDS-01 §1.1: defaults to 1) —
// no adapter may write schema_version 0 rows.
func testSchemaVersionZeroNormalized(t *testing.T, s core.StoragePort) {
	t.Helper()
	ev := makeEvent("inst-sv0", "ns-sv0", 1, time.Unix(1000, 0).UTC())
	ev.SchemaVersion = 0
	if err := s.AppendEvent(bg(), ev); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	got, err := s.ReadEvents(bg(), "inst-sv0", 0)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 event, got %d", len(got))
	}
	if got[0].SchemaVersion != 1 {
		t.Errorf("SchemaVersion: want 1 (normalized from 0), got %d", got[0].SchemaVersion)
	}
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("mustMarshal: " + err.Error())
	}
	return b
}

func minimalWorkflow(id, version, namespace string) core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            id,
		Version:       core.SemVer(version),
		Namespace:     namespace,
		Name:          id,
		InitialStep:   "start",
		FinalSteps:    []string{"start"},
		Steps: []core.Step{
			{ID: "start", Name: "Start", Type: core.StepTypeNative, Handler: "noop"},
		},
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Metadata: map[string]any{},
	}
}

// ── subtests ──────────────────────────────────────────────────────────────────

// testAppendReadRoundtrip verifies all 9 fields survive a write/read cycle,
// including a StepFailed event with a StepError payload (TDS-01 §2.4).
func testAppendReadRoundtrip(t *testing.T, s core.StoragePort) {
	t.Helper()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	iid := core.InstanceID("inst-rt-1")

	e1 := core.ExecutionEvent{
		EventID:       "evt-rt-1",
		InstanceID:    iid,
		Namespace:     "test",
		EventType:     core.EventTypeStepStarted,
		StepID:        "step-alpha",
		Payload:       mustMarshal(map[string]any{"step_id": "step-alpha", "attempt": 1, "inputs": map[string]any{}}),
		EmittedAt:     base,
		SequenceNum:   1,
		SchemaVersion: 1,
	}

	stepErr := core.StepError{
		Code:    "ERR_TIMEOUT",
		Message: "step timed out",
		Details: map[string]any{"attempt": 1},
	}
	e2 := core.ExecutionEvent{
		EventID:       "evt-rt-2",
		InstanceID:    iid,
		Namespace:     "test",
		EventType:     core.EventTypeStepFailed,
		StepID:        "step-alpha",
		Payload:       mustMarshal(map[string]any{"step_id": "step-alpha", "attempt": 1, "error": stepErr, "retrying": false}),
		EmittedAt:     base.Add(time.Second),
		SequenceNum:   2,
		SchemaVersion: 1,
	}

	if err := s.AppendEvent(bg(), e1); err != nil {
		t.Fatalf("AppendEvent e1: %v", err)
	}
	if err := s.AppendEvent(bg(), e2); err != nil {
		t.Fatalf("AppendEvent e2: %v", err)
	}

	events, err := s.ReadEvents(bg(), iid, 1)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}

	got := events[0]
	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"EventID", got.EventID, e1.EventID},
		{"InstanceID", string(got.InstanceID), string(iid)},
		{"Namespace", got.Namespace, "test"},
		{"EventType", string(got.EventType), string(core.EventTypeStepStarted)},
		{"StepID", got.StepID, "step-alpha"},
		{"SequenceNum", got.SequenceNum, 1},
		{"SchemaVersion", got.SchemaVersion, 1},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: want %v got %v", c.field, c.want, c.got)
		}
	}
	if !got.EmittedAt.Equal(base) {
		t.Errorf("EmittedAt: want %v got %v", base, got.EmittedAt)
	}

	// StepError roundtrip inside e2 payload.
	var payload map[string]any
	if err := json.Unmarshal(events[1].Payload, &payload); err != nil {
		t.Fatalf("unmarshal e2 payload: %v", err)
	}
	errMap, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("payload.error not a map: %T", payload["error"])
	}
	if errMap["code"] != stepErr.Code {
		t.Errorf("StepError.Code: want %q got %v", stepErr.Code, errMap["code"])
	}
	if errMap["message"] != stepErr.Message {
		t.Errorf("StepError.Message: want %q got %v", stepErr.Message, errMap["message"])
	}
	details, _ := errMap["details"].(map[string]any)
	if details["attempt"] != float64(1) {
		t.Errorf("StepError.Details.attempt: want 1 got %v", details["attempt"])
	}
}

// testSequenceMonotonicityRejection verifies equal and lower seq_num both
// return ErrSequenceViolation (typed error check).
func testSequenceMonotonicityRejection(t *testing.T, s core.StoragePort) {
	t.Helper()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	iid := core.InstanceID("inst-mono-1")

	if err := s.AppendEvent(bg(), makeEvent(iid, "test", 5, base)); err != nil {
		t.Fatalf("AppendEvent seq=5: %v", err)
	}

	// Equal seq → rejection.
	eEq := makeEvent(iid, "test", 5, base.Add(time.Second))
	eEq.EventID = "evt-mono-eq"
	if err := s.AppendEvent(bg(), eEq); !errors.Is(err, storage.ErrSequenceViolation) {
		t.Errorf("equal seq: want ErrSequenceViolation, got %v", err)
	}

	// Lower seq → rejection.
	eLow := makeEvent(iid, "test", 3, base.Add(2*time.Second))
	eLow.EventID = "evt-mono-low"
	if err := s.AppendEvent(bg(), eLow); !errors.Is(err, storage.ErrSequenceViolation) {
		t.Errorf("lower seq: want ErrSequenceViolation, got %v", err)
	}

	// Higher seq → success.
	eHi := makeEvent(iid, "test", 6, base.Add(3*time.Second))
	eHi.EventID = "evt-mono-hi"
	if err := s.AppendEvent(bg(), eHi); err != nil {
		t.Errorf("higher seq: unexpected error: %v", err)
	}
}

// testReadEventsFromSeqFilterAndOrder verifies fromSeq filtering and ascending order.
func testReadEventsFromSeqFilterAndOrder(t *testing.T, s core.StoragePort) {
	t.Helper()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	iid := core.InstanceID("inst-flt-1")

	for i := 1; i <= 5; i++ {
		if err := s.AppendEvent(bg(), makeEvent(iid, "test", i, base.Add(time.Duration(i)*time.Second))); err != nil {
			t.Fatalf("AppendEvent seq=%d: %v", i, err)
		}
	}

	events, err := s.ReadEvents(bg(), iid, 3)
	if err != nil {
		t.Fatalf("ReadEvents fromSeq=3: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("want 3 events (seq 3,4,5), got %d", len(events))
	}
	for i, e := range events {
		if want := 3 + i; e.SequenceNum != want {
			t.Errorf("events[%d].SequenceNum: want %d got %d", i, want, e.SequenceNum)
		}
	}
}

// testReadEventRangeNamespaceIsolation verifies that events from ns-a do not
// appear when querying ns-b and vice versa.
func testReadEventRangeNamespaceIsolation(t *testing.T, s core.StoragePort) {
	t.Helper()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	eA := makeEvent("inst-nsa-1", "ns-iso-a", 1, base)
	eA.EventID = "evt-iso-a"
	eB := makeEvent("inst-nsb-1", "ns-iso-b", 1, base)
	eB.EventID = "evt-iso-b"

	if err := s.AppendEvent(bg(), eA); err != nil {
		t.Fatalf("AppendEvent ns-iso-a: %v", err)
	}
	if err := s.AppendEvent(bg(), eB); err != nil {
		t.Fatalf("AppendEvent ns-iso-b: %v", err)
	}

	window := func(ns string) []core.ExecutionEvent {
		got, err := s.ReadEventRange(bg(), ns, base.Add(-time.Second), base.Add(time.Second))
		if err != nil {
			t.Fatalf("ReadEventRange %s: %v", ns, err)
		}
		return got
	}

	if a := window("ns-iso-a"); len(a) != 1 || a[0].Namespace != "ns-iso-a" {
		t.Errorf("ns-iso-a: want 1 event, got %d", len(a))
	}
	if b := window("ns-iso-b"); len(b) != 1 || b[0].Namespace != "ns-iso-b" {
		t.Errorf("ns-iso-b: want 1 event, got %d", len(b))
	}
}

// testReadEventRangeTimeBounds verifies events outside [from,to] are excluded.
func testReadEventRangeTimeBounds(t *testing.T, s core.StoragePort) {
	t.Helper()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	iid := core.InstanceID("inst-tbound-1")

	before := makeEvent(iid, "ns-tbound", 1, base.Add(-10*time.Minute))
	before.EventID = "evt-before"
	inside := makeEvent(iid, "ns-tbound", 2, base)
	inside.EventID = "evt-inside"
	after := makeEvent(iid, "ns-tbound", 3, base.Add(10*time.Minute))
	after.EventID = "evt-after"

	for _, e := range []core.ExecutionEvent{before, inside, after} {
		if err := s.AppendEvent(bg(), e); err != nil {
			t.Fatalf("AppendEvent %s: %v", e.EventID, err)
		}
	}

	got, err := s.ReadEventRange(bg(), "ns-tbound", base.Add(-time.Minute), base.Add(time.Minute))
	if err != nil {
		t.Fatalf("ReadEventRange: %v", err)
	}
	if len(got) != 1 || got[0].EventID != "evt-inside" {
		t.Errorf("want only evt-inside, got %d events", len(got))
	}
}

// testRegistryImmutability verifies re-registering (id, version) → ErrAlreadyRegistered.
func testRegistryImmutability(t *testing.T, s core.StoragePort) {
	t.Helper()
	def := minimalWorkflow("wf-imm-1", "1.0.0", "reg-ns")
	if err := s.RegisterWorkflow(bg(), def); err != nil {
		t.Fatalf("RegisterWorkflow first: %v", err)
	}
	if err := s.RegisterWorkflow(bg(), def); !errors.Is(err, storage.ErrAlreadyRegistered) {
		t.Errorf("duplicate: want ErrAlreadyRegistered, got %v", err)
	}
}

// testRegistryJSONRoundtrip verifies schema_version=1 and all fields survive
// through the registry (TDS-02 JSON fidelity).
func testRegistryJSONRoundtrip(t *testing.T, s core.StoragePort) {
	t.Helper()
	def := minimalWorkflow("wf-jrt-1", "2.0.0", "rt-ns")
	def.SchemaVersion = 1
	def.Name = "Roundtrip Workflow"
	def.Description = "tests JSON fidelity"

	if err := s.RegisterWorkflow(bg(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	got, err := s.GetWorkflow(bg(), def.ID, def.Version)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}

	fields := []struct {
		name string
		got  any
		want any
	}{
		{"SchemaVersion", got.SchemaVersion, 1},
		{"ID", got.ID, def.ID},
		{"Version", string(got.Version), string(def.Version)},
		{"Namespace", got.Namespace, def.Namespace},
		{"Name", got.Name, def.Name},
		{"Description", got.Description, def.Description},
		{"InitialStep", got.InitialStep, def.InitialStep},
	}
	for _, f := range fields {
		if f.got != f.want {
			t.Errorf("%s: want %v got %v", f.name, f.want, f.got)
		}
	}
}

// testListWorkflowsNamespacePredicate verifies namespace filtering on ListWorkflows.
func testListWorkflowsNamespacePredicate(t *testing.T, s core.StoragePort) {
	t.Helper()
	defA := minimalWorkflow("wf-list-a", "1.0.0", "list-ns-a")
	defB := minimalWorkflow("wf-list-b", "1.0.0", "list-ns-b")

	if err := s.RegisterWorkflow(bg(), defA); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := s.RegisterWorkflow(bg(), defB); err != nil {
		t.Fatalf("register b: %v", err)
	}

	listA, err := s.ListWorkflows(bg(), "list-ns-a")
	if err != nil {
		t.Fatalf("ListWorkflows a: %v", err)
	}
	if len(listA) != 1 || listA[0].ID != "wf-list-a" {
		t.Errorf("ns-a: want [wf-list-a], got %d items", len(listA))
	}

	listB, err := s.ListWorkflows(bg(), "list-ns-b")
	if err != nil {
		t.Fatalf("ListWorkflows b: %v", err)
	}
	if len(listB) != 1 || listB[0].ID != "wf-list-b" {
		t.Errorf("ns-b: want [wf-list-b], got %d items", len(listB))
	}
}

// testCacheTTLExpiry requires a *FakeClockStorage factory to control time.
// It verifies present-before-expiry and absent-after-expiry.
func testCacheTTLExpiry(t *testing.T, s core.StoragePort) {
	t.Helper()
	fcs, ok := s.(*FakeClockStorage)
	if !ok {
		t.Skip("TTL expiry test requires factory to return *FakeClockStorage")
	}

	key := core.IdempotencyKey("ikey-ttl-1")
	result := core.StepResult{Outputs: map[string]any{"v": 42}}
	ttl := 10 * time.Minute

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fcs.SetNow(t0)

	if err := fcs.CacheResult(bg(), key, result, ttl); err != nil {
		t.Fatalf("CacheResult: %v", err)
	}

	// Before expiry.
	got, found, err := fcs.GetCachedResult(bg(), key)
	if err != nil || !found {
		t.Fatalf("before expiry: want found=true; got found=%v err=%v", found, err)
	}
	if got.Outputs["v"] != float64(42) {
		t.Errorf("Outputs[v]: want 42 got %v", got.Outputs["v"])
	}

	// Advance past expiry.
	fcs.SetNow(t0.Add(11 * time.Minute))
	_, found, err = fcs.GetCachedResult(bg(), key)
	if err != nil {
		t.Fatalf("after expiry: unexpected error: %v", err)
	}
	if found {
		t.Errorf("after expiry: want found=false, got found=true")
	}
}

// testCacheAbsentKeyMiss verifies (zero, false, nil) for a missing key.
func testCacheAbsentKeyMiss(t *testing.T, s core.StoragePort) {
	t.Helper()
	_, found, err := s.GetCachedResult(bg(), core.IdempotencyKey("no-such-key-"+itoa(int(time.Now().UnixNano()))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Errorf("absent key: want found=false")
	}
}

// ── StateStore subtests (M03) ─────────────────────────────────────────────────

// makeInstance builds a minimal WorkflowInstance for test use.
func makeInstance(id, defID, ns string, status core.InstanceStatus) core.WorkflowInstance {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return core.WorkflowInstance{
		InstanceID:        core.InstanceID(id),
		DefinitionID:      defID,
		DefinitionVersion: core.SemVer("1.0.0"),
		Namespace:         ns,
		Status:            status,
		CurrentSteps:      []string{},
		Variables:         map[string]any{},
		StartedAt:         base,
		UpdatedAt:         base,
		CompletedAt:       nil,
	}
}

// testUpsertCreateThenGet verifies a full 10-field roundtrip (all WorkflowInstance
// fields, incl. nil CompletedAt) via UpsertInstance(expectedVersion=0) then GetInstance.
func testUpsertCreateThenGet(t *testing.T, s core.StoragePort) {
	t.Helper()
	inst := makeInstance("inst-cg-1", "wf-cg", "ns-cg", core.InstanceStatusRunning)
	inst.CurrentSteps = []string{"step-a", "step-b"}
	inst.Variables = map[string]any{"inputs": map[string]any{"x": float64(1)}}

	if err := s.UpsertInstance(bg(), inst, 0); err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	got, err := s.GetInstance(bg(), inst.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}

	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"InstanceID", string(got.InstanceID), string(inst.InstanceID)},
		{"DefinitionID", got.DefinitionID, inst.DefinitionID},
		{"DefinitionVersion", string(got.DefinitionVersion), string(inst.DefinitionVersion)},
		{"Namespace", got.Namespace, inst.Namespace},
		{"Status", string(got.Status), string(inst.Status)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: want %v got %v", c.field, c.want, c.got)
		}
	}

	if !got.StartedAt.Equal(inst.StartedAt) {
		t.Errorf("StartedAt: want %v got %v", inst.StartedAt, got.StartedAt)
	}
	if !got.UpdatedAt.Equal(inst.UpdatedAt) {
		t.Errorf("UpdatedAt: want %v got %v", inst.UpdatedAt, got.UpdatedAt)
	}
	if got.CompletedAt != nil {
		t.Errorf("CompletedAt: want nil, got %v", got.CompletedAt)
	}

	if len(got.CurrentSteps) != 2 || got.CurrentSteps[0] != "step-a" || got.CurrentSteps[1] != "step-b" {
		t.Errorf("CurrentSteps: want [step-a step-b], got %v", got.CurrentSteps)
	}

	inputs, _ := got.Variables["inputs"].(map[string]any)
	if inputs["x"] != float64(1) {
		t.Errorf("Variables.inputs.x: want 1 got %v", inputs["x"])
	}
}

// testUpsertOptimisticConflict verifies that supplying a stale expectedVersion
// returns a typed ErrVersionConflict (errors.Is).
func testUpsertOptimisticConflict(t *testing.T, s core.StoragePort) {
	t.Helper()
	inst := makeInstance("inst-oc-1", "wf-oc", "ns-oc", core.InstanceStatusRunning)

	// Create (version becomes 1 in DB).
	if err := s.UpsertInstance(bg(), inst, 0); err != nil {
		t.Fatalf("UpsertInstance create: %v", err)
	}

	// Attempt update with stale version 0.
	err := s.UpsertInstance(bg(), inst, 0)
	if !errors.Is(err, storage.ErrVersionConflict) {
		t.Errorf("stale version: want ErrVersionConflict, got %v", err)
	}
}

// testUpsertUpdateAdvancesVersion verifies create then update succeeds when the
// caller uses the correct version sequence (0 → created at 1; 1 → updated to 2).
func testUpsertUpdateAdvancesVersion(t *testing.T, s core.StoragePort) {
	t.Helper()
	inst := makeInstance("inst-uav-1", "wf-uav", "ns-uav", core.InstanceStatusRunning)

	if err := s.UpsertInstance(bg(), inst, 0); err != nil {
		t.Fatalf("UpsertInstance create: %v", err)
	}

	inst.Status = core.InstanceStatusWaiting
	inst.UpdatedAt = inst.UpdatedAt.Add(time.Second)

	if err := s.UpsertInstance(bg(), inst, 1); err != nil {
		t.Fatalf("UpsertInstance update (expectedVersion=1): %v", err)
	}

	got, err := s.GetInstance(bg(), inst.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if got.Status != core.InstanceStatusWaiting {
		t.Errorf("Status after update: want waiting, got %s", got.Status)
	}
}

// testListInstancesNamespacePredicate verifies that ListInstances with a non-empty
// Namespace returns only instances in that namespace.
func testListInstancesNamespacePredicate(t *testing.T, s core.StoragePort) {
	t.Helper()
	a := makeInstance("inst-lns-a", "wf-lns", "list-ns-ia", core.InstanceStatusRunning)
	b := makeInstance("inst-lns-b", "wf-lns", "list-ns-ib", core.InstanceStatusRunning)

	if err := s.UpsertInstance(bg(), a, 0); err != nil {
		t.Fatalf("UpsertInstance a: %v", err)
	}
	if err := s.UpsertInstance(bg(), b, 0); err != nil {
		t.Fatalf("UpsertInstance b: %v", err)
	}

	listA, err := s.ListInstances(bg(), core.InstanceFilter{Namespace: "list-ns-ia"})
	if err != nil {
		t.Fatalf("ListInstances ns-a: %v", err)
	}
	if len(listA) != 1 || string(listA[0].InstanceID) != "inst-lns-a" {
		t.Errorf("ns-a: want [inst-lns-a], got %v", listA)
	}

	listB, err := s.ListInstances(bg(), core.InstanceFilter{Namespace: "list-ns-ib"})
	if err != nil {
		t.Fatalf("ListInstances ns-b: %v", err)
	}
	if len(listB) != 1 || string(listB[0].InstanceID) != "inst-lns-b" {
		t.Errorf("ns-b: want [inst-lns-b], got %v", listB)
	}
}

// testListInstancesStatusPredicate verifies that ListInstances with a non-empty
// Status returns only instances with that status.
func testListInstancesStatusPredicate(t *testing.T, s core.StoragePort) {
	t.Helper()
	run := makeInstance("inst-lsp-r", "wf-lsp", "ns-lsp", core.InstanceStatusRunning)
	wait := makeInstance("inst-lsp-w", "wf-lsp", "ns-lsp", core.InstanceStatusWaiting)

	if err := s.UpsertInstance(bg(), run, 0); err != nil {
		t.Fatalf("UpsertInstance running: %v", err)
	}
	if err := s.UpsertInstance(bg(), wait, 0); err != nil {
		t.Fatalf("UpsertInstance waiting: %v", err)
	}

	running, err := s.ListInstances(bg(), core.InstanceFilter{Namespace: "ns-lsp", Status: core.InstanceStatusRunning})
	if err != nil {
		t.Fatalf("ListInstances running: %v", err)
	}
	if len(running) != 1 || running[0].Status != core.InstanceStatusRunning {
		t.Errorf("running: want 1 running instance, got %d", len(running))
	}

	waiting, err := s.ListInstances(bg(), core.InstanceFilter{Namespace: "ns-lsp", Status: core.InstanceStatusWaiting})
	if err != nil {
		t.Fatalf("ListInstances waiting: %v", err)
	}
	if len(waiting) != 1 || waiting[0].Status != core.InstanceStatusWaiting {
		t.Errorf("waiting: want 1 waiting instance, got %d", len(waiting))
	}
}

// testClaimAtMostOnce verifies the at-most-once claim contract: a second claim
// by any worker on the same (instance, step) returns (false, nil).
func testClaimAtMostOnce(t *testing.T, s core.StoragePort) {
	t.Helper()
	inst := makeInstance("inst-amo-1", "wf-amo", "ns-amo", core.InstanceStatusRunning)
	if err := s.UpsertInstance(bg(), inst, 0); err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}

	// First claim → must succeed.
	claimed, err := s.ClaimStep(bg(), inst.InstanceID, "step-x", "worker-1")
	if err != nil {
		t.Fatalf("ClaimStep first: %v", err)
	}
	if !claimed {
		t.Errorf("first claim: want true, got false")
	}

	// Second claim (different worker) → must return (false, nil).
	claimed2, err := s.ClaimStep(bg(), inst.InstanceID, "step-x", "worker-2")
	if err != nil {
		t.Fatalf("ClaimStep second: %v", err)
	}
	if claimed2 {
		t.Errorf("second claim: want false, got true")
	}
}

// testClaimBumpsVersion verifies that a successful ClaimStep increments the
// instance version, so a subsequent UpsertInstance with the pre-claim
// expectedVersion returns ErrVersionConflict.
func testClaimBumpsVersion(t *testing.T, s core.StoragePort) {
	t.Helper()
	inst := makeInstance("inst-cbv-1", "wf-cbv", "ns-cbv", core.InstanceStatusRunning)

	// Create: DB version becomes 1.
	if err := s.UpsertInstance(bg(), inst, 0); err != nil {
		t.Fatalf("UpsertInstance create: %v", err)
	}

	// Claim: DB version becomes 2.
	claimed, err := s.ClaimStep(bg(), inst.InstanceID, "step-y", "worker-1")
	if err != nil || !claimed {
		t.Fatalf("ClaimStep: claimed=%v err=%v", claimed, err)
	}

	// Upsert with pre-claim expectedVersion=1 → conflict (actual is 2).
	err = s.UpsertInstance(bg(), inst, 1)
	if !errors.Is(err, storage.ErrVersionConflict) {
		t.Errorf("want ErrVersionConflict after claim bump, got %v", err)
	}
}

// testClaimInstanceAbsentErrors verifies that ClaimStep returns a non-nil error
// when the instance_id does not exist (not (false, nil) — distinct from conflict).
func testClaimInstanceAbsentErrors(t *testing.T, s core.StoragePort) {
	t.Helper()
	_, err := s.ClaimStep(bg(), "no-such-instance-cia", "step-z", "worker-1")
	if err == nil {
		t.Errorf("absent instance: want error, got nil")
	}
}

// testTerminalUpsertReleasesClaims verifies the EDR-006 release site:
// upserting a terminal status deletes existing step_claims, so a subsequent
// ClaimStep on the same (instance, step) succeeds again (true, nil).
func testTerminalUpsertReleasesClaims(t *testing.T, s core.StoragePort) {
	t.Helper()
	inst := makeInstance("inst-trc-1", "wf-trc", "ns-trc", core.InstanceStatusRunning)

	// Create instance.
	if err := s.UpsertInstance(bg(), inst, 0); err != nil {
		t.Fatalf("UpsertInstance create: %v", err)
	}

	// Claim a step — DB version becomes 2.
	claimed, err := s.ClaimStep(bg(), inst.InstanceID, "step-fin", "worker-1")
	if err != nil || !claimed {
		t.Fatalf("ClaimStep: claimed=%v err=%v", claimed, err)
	}

	// Upsert terminal status with expectedVersion=2 → releases step_claims.
	now := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	inst.Status = core.InstanceStatusCompleted
	inst.CompletedAt = &now
	inst.UpdatedAt = now
	if err := s.UpsertInstance(bg(), inst, 2); err != nil {
		t.Fatalf("UpsertInstance terminal: %v", err)
	}

	// Claim again on the same step — must succeed because claims were released.
	claimed2, err := s.ClaimStep(bg(), inst.InstanceID, "step-fin", "worker-2")
	if err != nil {
		t.Fatalf("ClaimStep after terminal: %v", err)
	}
	if !claimed2 {
		t.Errorf("after terminal upsert: want claimed=true (claims released), got false")
	}
}
