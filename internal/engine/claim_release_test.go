package engine

// D-6 fault-injection test: proves gatherDispatch releases an orphaned step
// claim when the StepStarted emission that must follow a successful ClaimStep
// fails, and that the instance still reaches a terminal status afterward.
//
// This is the deterministic reproduction of the captured defect: ClaimStep
// commits the step_claims INSERT (and the instance version bump) in its own
// transaction (sqlite.go); the subsequent StepStarted emission happens
// separately in gatherDispatch (tick.go). Before the D-6 fix, a failure at
// that emission left the claim orphaned forever — activatableFor never
// consults step_claims, so the step stayed "activatable" while the claim
// could never be won again (PK conflict), wedging the instance at
// status=running, current_steps=[], permanently.
//
// faultInjectingStorage is a TEST-ONLY seam (this file), never compiled into
// production code: it embeds the real *storage.SQLiteStorage (so every
// StoragePort method AND every additive interface method — ReleaseStepClaim,
// InstanceVersion, etc. — are forwarded untouched) and overrides only
// AppendEvent, selectively failing it for StepStarted events while the fault
// is armed.

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

// faultInjectingStorage wraps *storage.SQLiteStorage to force a transient
// AppendEvent failure for StepStarted events on demand (test-only; internal/
// engine's production code has no fault hook of any kind).
type faultInjectingStorage struct {
	*storage.SQLiteStorage

	mu      sync.Mutex
	armed   bool
	tripped int // count of injected failures actually returned
}

func (f *faultInjectingStorage) arm() {
	f.mu.Lock()
	f.armed = true
	f.mu.Unlock()
}

func (f *faultInjectingStorage) disarm() {
	f.mu.Lock()
	f.armed = false
	f.mu.Unlock()
}

func (f *faultInjectingStorage) trippedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tripped
}

var errInjectedStepStartedFailure = errors.New("injected: transient storage error during StepStarted emission")

func (f *faultInjectingStorage) AppendEvent(ctx context.Context, event core.ExecutionEvent) error {
	f.mu.Lock()
	fail := f.armed && event.EventType == core.EventTypeStepStarted
	if fail {
		f.tripped++
	}
	f.mu.Unlock()
	if fail {
		return errInjectedStepStartedFailure
	}
	return f.SQLiteStorage.AppendEvent(ctx, event)
}

// TestD6_ClaimReleasedWhenStepStartedEmissionFails is the fault-injection
// proof (D-6 Test 1). It MUST fail against the pre-fix gatherDispatch (which
// returns the emission error without releasing the already-committed claim)
// and MUST pass against the fix.
func TestD6_ClaimReleasedWhenStepStartedEmissionFails(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.d6orphan", Version: "1.0.0", Namespace: "t", Name: "d6orphan",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("greet", "hgreet")},
		InitialStep: "greet", FinalSteps: []string{"greet"}, Metadata: map[string]any{},
	}
	hgreet := &stepHandler{id: "hgreet", outputs: map[string]any{"msg": "hi"}}

	real := openStorage(t)
	if err := real.RegisterWorkflow(context.Background(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	fis := &faultInjectingStorage{SQLiteStorage: real}

	nr := native.New()
	nr.Register(hgreet)
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	e := New(fis, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: 1, Clock: clk.now}, discardLogger())

	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.d6orphan", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Arm the fault: the very next StepStarted emission fails, AFTER the
	// claim underneath it has already committed.
	fis.arm()
	if err := e.Tick(ctx); err == nil {
		t.Fatal("Tick: want the injected StepStarted emission error to propagate, got nil")
	} else if !errors.Is(err, errInjectedStepStartedFailure) {
		t.Fatalf("Tick error = %v, want it to wrap errInjectedStepStartedFailure", err)
	}
	if got := fis.trippedCount(); got != 1 {
		t.Fatalf("injected fault tripped %d times on this tick, want exactly 1", got)
	}

	// (i) The claim ClaimStep committed before the emission failed must have
	// been released — proven by successfully claiming it again as a
	// different worker, using the real, unwrapped storage method (production
	// API surface only, no test hook).
	won, err := real.ClaimStep(ctx, iid, "greet", "probe-after-fault")
	if err != nil {
		t.Fatalf("probe ClaimStep after fault: %v", err)
	}
	if !won {
		t.Fatal("D-6 regression: step claim was NOT released after the StepStarted emission failed; " +
			"the step can never be won again and the instance is permanently wedged")
	}
	// Release the probe claim so it does not itself block the real engine's
	// next tick (this call is test bookkeeping, not part of the assertion).
	if err := real.ReleaseStepClaim(ctx, iid, "greet"); err != nil {
		t.Fatalf("release probe claim: %v", err)
	}

	// Disarm: the fault was transient (as documented in the defect report);
	// subsequent ticks proceed normally.
	fis.disarm()

	// (ii) The instance must still reach a terminal status — proving the
	// released claim can be won again and the workflow completes rather than
	// wedging at status=running, current_steps=[] forever.
	inst := runToTerminal(t, e, real, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}
}
