package engine

// D-17 fault-injection proof: a single instance's persistent error must not
// abort the whole Tick — every other instance must still be processed, and
// SIGNAL_SCAN / SIGNAL_TIMEOUT_SCAN must still run.
//
// hydrateFaultingStorage is a TEST-ONLY seam (this file), never compiled into
// production code: it embeds the real *storage.SQLiteStorage (so every
// StoragePort method AND every additive interface method is forwarded
// untouched) and overrides only ReadEvents, failing it PERSISTENTLY (every
// call, not one-shot) for one targeted instance ID. hydrate() (hydrate.go)
// calls ReadEvents(ctx, iid, 0) on the first tick that touches an instance and
// only marks it hydrated on success, so targeting ReadEvents reproduces a
// realistic, indefinitely-repeating per-instance processInstance failure
// without touching engine internals or any frozen interface.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// d17ClockBase deliberately starts on a NON-zero, non-trailing-zero
// millisecond boundary (.123, .133, ... at a fixed 10ms step). started_at is
// persisted as RFC3339Nano text (internal/storage/sqlite.go) and
// ListInstances sorts with a plain `ORDER BY started_at` (TEXT) — RFC3339Nano
// trims a wholly-zero fractional part, so a clock landing exactly on a whole
// second (e.g. the shared engineStart used elsewhere in this package) would
// format WITHOUT a fractional suffix while every later, sub-second-offset
// instance formats WITH one, and '.' (0x2E) sorts before 'Z' (0x5A) — text
// order would then put the LATER instance first, silently defeating the
// oldest-first ordering the D-17 fixtures rely on. Not a defect this card
// covers (see the report's "noticed but not changed" note); d17ClockBase
// simply avoids the boundary so this fixture measures D-17, not that.
var d17ClockBase = time.Date(2026, 6, 1, 12, 0, 0, 123_000_000, time.UTC)

// hydrateFaultingStorage wraps *storage.SQLiteStorage to force ReadEvents to
// fail, every time, for one targeted instance ID (set via setTarget). Other
// instances' ReadEvents calls are forwarded to the real storage unchanged.
type hydrateFaultingStorage struct {
	*storage.SQLiteStorage

	mu     sync.Mutex
	target core.InstanceID // empty ⇒ fault disabled
	trips  int
}

func (f *hydrateFaultingStorage) setTarget(iid core.InstanceID) {
	f.mu.Lock()
	f.target = iid
	f.mu.Unlock()
}

func (f *hydrateFaultingStorage) tripCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.trips
}

var errInjectedHydrateFailure = errors.New("injected: persistent storage error reading events for hydrate")

func (f *hydrateFaultingStorage) ReadEvents(ctx context.Context, instanceID core.InstanceID, fromSeq int) ([]core.ExecutionEvent, error) {
	f.mu.Lock()
	fail := f.target != "" && instanceID == f.target
	if fail {
		f.trips++
	}
	f.mu.Unlock()
	if fail {
		return nil, errInjectedHydrateFailure
	}
	return f.SQLiteStorage.ReadEvents(ctx, instanceID, fromSeq)
}

// TestD17_LaterInstanceProgressesDespiteEarlierInstanceError is the D-17 proof
// (Test 1): two instances of the same single-native-step workflow, the FIRST
// (by started_at — a monotonically-advancing fakeClock fixes submission order
// to started_at order) faulted so it fails persistently in processInstance
// (hydrate → ReadEvents). It asserts the SECOND instance still reaches
// InstanceStatusCompleted, i.e. its own error does not starve every instance
// ordered after it.
//
// This test MUST fail against the pre-fix tick.go (a single `return err` in
// the SCAN_RUNNABLE loop that aborts the entire Tick on the first instance's
// error) — verified by stashing the tick.go fix and re-running this test; see
// the D-17 commit report for both results.
func TestD17_LaterInstanceProgressesDespiteEarlierInstanceError(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.d17iso", Version: "1.0.0", Namespace: "t", Name: "d17iso",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("greet", "hgreet")},
		InitialStep: "greet", FinalSteps: []string{"greet"}, Metadata: map[string]any{},
	}
	hgreet := &stepHandler{id: "hgreet", outputs: map[string]any{"msg": "hi"}}

	real := openStorage(t)
	if err := real.RegisterWorkflow(context.Background(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	fs := &hydrateFaultingStorage{SQLiteStorage: real}

	nr := native.New()
	nr.Register(hgreet)
	clk := newFakeClock(d17ClockBase, 10*time.Millisecond)
	e := New(fs, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: 1, Clock: clk.now}, discardLogger())

	ctx := context.Background()

	// Submit the instance that will be faulted FIRST, so its started_at is
	// strictly earlier (fakeClock advances 10ms on every now() call, and
	// Submit's WorkflowStarted emission is the source of started_at).
	iidFailing, err := e.Submit(ctx, "t.d17iso", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit (failing instance): %v", err)
	}
	iidOK, err := e.Submit(ctx, "t.d17iso", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit (ok instance): %v", err)
	}

	// Confirm the ordering assumption the fixture depends on (ListInstances is
	// ORDER BY started_at ascending, so the failing instance is processed
	// first this tick, exactly the D-17 scenario).
	failingInst, err := real.GetInstance(ctx, iidFailing)
	if err != nil {
		t.Fatalf("GetInstance (failing): %v", err)
	}
	okInst, err := real.GetInstance(ctx, iidOK)
	if err != nil {
		t.Fatalf("GetInstance (ok): %v", err)
	}
	if !failingInst.StartedAt.Before(okInst.StartedAt) {
		t.Fatalf("fixture invariant violated: failing instance started_at %v is not before ok instance started_at %v",
			failingInst.StartedAt, okInst.StartedAt)
	}
	listed, err := real.ListInstances(ctx, core.InstanceFilter{Status: core.InstanceStatusRunning})
	if err != nil {
		t.Fatalf("ListInstances (order check): %v", err)
	}
	if len(listed) != 2 || listed[0].InstanceID != iidFailing || listed[1].InstanceID != iidOK {
		t.Fatalf("fixture invariant violated: ListInstances order = %v, want [failing, ok]", listed)
	}

	// Arm the fault: every ReadEvents call for iidFailing errors, forever, so
	// hydrate() (called from processInstance) fails EVERY tick for that
	// instance and it never advances past status=running, current_steps=[].
	fs.setTarget(iidFailing)

	// Drive ticks manually (not runToTerminal — that helper t.Fatal()s on any
	// Tick error, and the failing instance guarantees one every tick). Bounded
	// iteration guard mirrors runToTerminal's.
	var lastTickErr error
	okDone := false
	for i := 0; i < 200; i++ {
		lastTickErr = e.Tick(ctx)
		inst, err := real.GetInstance(ctx, iidOK)
		if err != nil {
			t.Fatalf("GetInstance (ok, tick %d): %v", i, err)
		}
		if isTerminal(inst.Status) {
			okDone = true
			okInst = inst
			break
		}
	}
	if !okDone {
		t.Fatalf("ok instance %s did not reach a terminal status within 200 ticks while the earlier instance %s failed persistently",
			iidOK, iidFailing)
	}
	if okInst.Status != core.InstanceStatusCompleted {
		t.Fatalf("ok instance status = %q, want completed", okInst.Status)
	}

	// The failing instance must still be exactly where it started: never
	// hydrated past defViewFor, no StepStarted ever emitted for it.
	failingInst, err = real.GetInstance(ctx, iidFailing)
	if err != nil {
		t.Fatalf("GetInstance (failing, final): %v", err)
	}
	if failingInst.Status != core.InstanceStatusRunning {
		t.Fatalf("failing instance status = %q, want running (never progressed)", failingInst.Status)
	}
	if len(failingInst.CurrentSteps) != 0 {
		t.Fatalf("failing instance current_steps = %v, want [] (never dispatched)", failingInst.CurrentSteps)
	}
	if fs.tripCount() == 0 {
		t.Fatal("fault never tripped; fixture did not exercise the failure path")
	}

	// The last tick must still report the failure (D-17 point 3: errors are
	// collected and returned, never silently dropped), wrapping the injected
	// sentinel so a caller can identify the cause.
	if lastTickErr == nil {
		t.Fatal("Tick: want the injected persistent hydrate failure to be reported, got nil")
	}
	if !errors.Is(lastTickErr, errInjectedHydrateFailure) {
		t.Fatalf("Tick error = %v, want it to wrap errInjectedHydrateFailure", lastTickErr)
	}
}

// TestD17_SignalScansRunDespiteInstanceError proves SIGNAL_SCAN and
// SIGNAL_TIMEOUT_SCAN both still execute on a tick where a `running` instance
// fails persistently in processInstance. Both scan stages operate on
// instances OUTSIDE the `running` filter SCAN_RUNNABLE lists (a `waiting`
// instance with a due signal delivery / a due timeout), so this isolates
// "did the stage run at all" from the SCAN_RUNNABLE loop's own per-instance
// behavior (covered by the sibling test above).
func TestD17_SignalScansRunDespiteInstanceError(t *testing.T) {
	failingDef := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.d17fail", Version: "1.0.0", Namespace: "t", Name: "d17fail",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("greet", "hgreet")},
		InitialStep: "greet", FinalSteps: []string{"greet"}, Metadata: map[string]any{},
	}
	deliverDef := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.d17deliver", Version: "1.0.0", Namespace: "t", Name: "d17deliver",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{signalStep("w", "go")},
		InitialStep: "w", FinalSteps: []string{"w"}, Metadata: map[string]any{},
	}
	timeoutDef := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.d17timeout", Version: "1.0.0", Namespace: "t", Name: "d17timeout",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{signalStepTimeout("w", "go", "5s", "fail")},
		InitialStep: "w", FinalSteps: []string{"w"}, Metadata: map[string]any{},
	}
	hgreet := &stepHandler{id: "hgreet", outputs: map[string]any{"msg": "hi"}}

	real := openStorage(t)
	for _, def := range []core.WorkflowDefinition{failingDef, deliverDef, timeoutDef} {
		if err := real.RegisterWorkflow(context.Background(), def); err != nil {
			t.Fatalf("RegisterWorkflow %s: %v", def.ID, err)
		}
	}
	fs := &hydrateFaultingStorage{SQLiteStorage: real}

	nr := native.New()
	nr.Register(hgreet)
	clk := newManualClock(d17ClockBase)
	e := New(fs, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: 1, Clock: clk.now}, discardLogger())

	ctx := context.Background()

	// Submit the two signal-wait instances and drive one tick each so both
	// park in `waiting` with a live wait_record (before the fault is armed —
	// getting them to `waiting` must succeed).
	iidDeliver, err := e.Submit(ctx, "t.d17deliver", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit (deliver): %v", err)
	}
	iidTimeout, err := e.Submit(ctx, "t.d17timeout", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit (timeout): %v", err)
	}
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick (WAIT entry for both signal instances): %v", err)
	}
	for _, iid := range []core.InstanceID{iidDeliver, iidTimeout} {
		inst, err := real.GetInstance(ctx, iid)
		if err != nil {
			t.Fatalf("GetInstance %s: %v", iid, err)
		}
		if inst.Status != core.InstanceStatusWaiting {
			t.Fatalf("instance %s status = %q, want waiting before the fault is armed", iid, inst.Status)
		}
	}

	// Now submit the instance that will fail persistently, and arm the fault
	// targeting it. It is `running`, so SCAN_RUNNABLE will attempt it every
	// tick and fail; the two `waiting` instances above are NOT in that filter.
	iidFailing, err := e.Submit(ctx, "t.d17fail", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit (failing): %v", err)
	}
	fs.setTarget(iidFailing)

	// Queue the signal for the deliver instance, and advance the clock past
	// the timeout instance's 5s deadline, so this next tick has both a due
	// delivery and a due timeout to act on.
	if err := e.Signal(ctx, iidDeliver, "go", map[string]any{"by": "alice"}); err != nil {
		t.Fatalf("Signal: %v", err)
	}
	clk.advance(6 * time.Second)

	// This tick: SCAN_RUNNABLE processes iidFailing and fails (logged,
	// non-aborting); SIGNAL_SCAN must still deliver "go" to iidDeliver;
	// SIGNAL_TIMEOUT_SCAN must still expire iidTimeout's wait_record.
	err = e.Tick(ctx)
	if err == nil {
		t.Fatal("Tick: want the failing instance's error to be reported, got nil")
	}
	if !errors.Is(err, errInjectedHydrateFailure) {
		t.Fatalf("Tick error = %v, want it to wrap errInjectedHydrateFailure", err)
	}

	deliverInst, gerr := real.GetInstance(ctx, iidDeliver)
	if gerr != nil {
		t.Fatalf("GetInstance (deliver): %v", gerr)
	}
	if deliverInst.Status != core.InstanceStatusRunning {
		t.Fatalf("SIGNAL_SCAN did not run despite the other instance's error: deliver instance status = %q, want running (resumed)",
			deliverInst.Status)
	}
	if len(deliverInst.CurrentSteps) != 0 {
		t.Fatalf("deliver instance current_steps = %v, want [] (signal step completed)", deliverInst.CurrentSteps)
	}

	timeoutInst, gerr := real.GetInstance(ctx, iidTimeout)
	if gerr != nil {
		t.Fatalf("GetInstance (timeout): %v", gerr)
	}
	if timeoutInst.Status != core.InstanceStatusFailed {
		t.Fatalf("SIGNAL_TIMEOUT_SCAN did not run despite the other instance's error: timeout instance status = %q, want failed",
			timeoutInst.Status)
	}
}
