package engine

// LA-P1-REPRO: deterministic FAILING reproducers for the "stranded StepStarted"
// failure class (Loop A Priority 1/2). TEST-ONLY: this file adds no production
// code, no fault hook inside internal/engine, and no dependency.
//
// Failure model (every lead): ONE storage call site fails ONCE (transient). The
// nth matching call returns errLAInjected WITHOUT reaching the real store, so
// the failed call commits nothing; the fault then disarms itself and every later
// call is forwarded untouched.
//
// Each lead has two tests asserting the CORRECT behavior named by the card:
//   - TestLA_<Lead>_...: after the fault, the SAME engine reaches the expected
//     state within laK further Tick calls (fake clock, no sleeps).
//   - TestLA_<Lead>_..._AfterRestart: after the fault, a NEW engine on the SAME
//     storage (process-restart simulation) ticks up to laK times.
//
// On current code a test FAILS when its lead is real. That red baseline is the
// deliverable: no production code is changed to make these pass. Every test logs
// the wedge/terminal state (status, current_steps, ordered events, live
// wait_records, undelivered signals) via t.Logf.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

const (
	// laK is the card bound: further Tick calls, in the same (or restarted)
	// engine, within which the instance must reach its expected state.
	laK = 50
	// laSetupTicks bounds the ticks that only REACH the fault site. They are not
	// part of K.
	laSetupTicks = 10
)

var (
	errLAInjected = errors.New("injected: transient storage fault (LA-P1-REPRO)")
	errLAHandler  = errors.New("la: handler failure (deterministic StepError source)")
)

// laOp names the storage method a laFault targets.
type laOp string

const (
	laOpAppendEvent      laOp = "AppendEvent"
	laOpClaimStep        laOp = "ClaimStep"
	laOpCreateWaitRecord laOp = "CreateWaitRecord"
	laOpUpsertInstance   laOp = "UpsertInstance"
	laOpDeleteWaitRecord laOp = "DeleteWaitRecord"
)

// laFault is ONE transient injected failure. The nth call to op whose arguments
// satisfy the filter fails once; every other call is forwarded. Zero-valued
// filter fields are wildcards. The fault disarms itself after its single failure.
type laFault struct {
	op        laOp
	nth       int                 // 1-based index among matching calls
	eventType core.EventType      // AppendEvent filter ("" = any)
	stepID    string              // step-id filter ("" = any)
	status    core.InstanceStatus // UpsertInstance filter ("" = any)
}

// laFaultStorage wraps *storage.SQLiteStorage (test-only; no production hook).
// It embeds the real store, so every StoragePort method and every additive
// interface the engine reaches by type assertion (claimReleaseStore, signal.Store,
// signalWaitStore, ...) is forwarded untouched. Only the five methods below are
// overridden.
type laFaultStorage struct {
	*storage.SQLiteStorage

	mu      sync.Mutex
	armed   bool
	fault   laFault
	matched int // matching calls observed since the last arm
	tripped int // injected failures actually returned, cumulative
}

func newLAFaultStorage(s *storage.SQLiteStorage) *laFaultStorage {
	return &laFaultStorage{SQLiteStorage: s}
}

// arm installs ft as the pending fault, replacing any previous one.
func (f *laFaultStorage) arm(ft laFault) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed = true
	f.fault = ft
	f.matched = 0
}

func (f *laFaultStorage) trippedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tripped
}

// fire decides whether the current call fails. It is the only place where the
// nth-match and the transient disarm are decided.
func (f *laFaultStorage) fire(op laOp, matches func(laFault) bool) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.armed || f.fault.op != op || !matches(f.fault) {
		return false
	}
	f.matched++
	if f.matched != f.fault.nth {
		return false
	}
	f.armed = false // transient: exactly one failure, then forward again
	f.tripped++
	return true
}

func laInjected(op laOp) error {
	return fmt.Errorf("%w (op=%s)", errLAInjected, op)
}

func (f *laFaultStorage) AppendEvent(ctx context.Context, ev core.ExecutionEvent) error {
	if f.fire(laOpAppendEvent, func(ft laFault) bool {
		return (ft.eventType == "" || ft.eventType == ev.EventType) &&
			(ft.stepID == "" || ft.stepID == ev.StepID)
	}) {
		return laInjected(laOpAppendEvent)
	}
	return f.SQLiteStorage.AppendEvent(ctx, ev)
}

func (f *laFaultStorage) ClaimStep(ctx context.Context, instanceID core.InstanceID, stepID string, workerID string) (bool, error) {
	if f.fire(laOpClaimStep, func(ft laFault) bool {
		return ft.stepID == "" || ft.stepID == stepID
	}) {
		return false, laInjected(laOpClaimStep)
	}
	return f.SQLiteStorage.ClaimStep(ctx, instanceID, stepID, workerID)
}

func (f *laFaultStorage) CreateWaitRecord(ctx context.Context, wr storage.WaitRecord) error {
	if f.fire(laOpCreateWaitRecord, func(ft laFault) bool {
		return ft.stepID == "" || ft.stepID == wr.StepID
	}) {
		return laInjected(laOpCreateWaitRecord)
	}
	return f.SQLiteStorage.CreateWaitRecord(ctx, wr)
}

// UpsertInstance: the running→waiting write is the only UpsertInstance whose
// projected status is `waiting` (enterWait); a status filter isolates it.
func (f *laFaultStorage) UpsertInstance(ctx context.Context, inst core.WorkflowInstance, expectedVersion int) error {
	if f.fire(laOpUpsertInstance, func(ft laFault) bool {
		return ft.status == "" || ft.status == inst.Status
	}) {
		return laInjected(laOpUpsertInstance)
	}
	return f.SQLiteStorage.UpsertInstance(ctx, inst, expectedVersion)
}

func (f *laFaultStorage) DeleteWaitRecord(ctx context.Context, instanceID core.InstanceID, stepID string) error {
	if f.fire(laOpDeleteWaitRecord, func(ft laFault) bool {
		return ft.stepID == "" || ft.stepID == stepID
	}) {
		return laInjected(laOpDeleteWaitRecord)
	}
	return f.SQLiteStorage.DeleteWaitRecord(ctx, instanceID, stepID)
}

// laIDs returns a deterministic, concurrency-safe id source (Config.NewID). The
// SAME source is shared by both engines of a restart test, so event and signal
// ids stay unique across the simulated process boundary.
func laIDs() func() string {
	var mu sync.Mutex
	n := 0
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		n++
		return fmt.Sprintf("la-id-%06d", n)
	}
}

// laEngine builds an Engine over st with one native runner holding handlers.
// Calling it again on the same st/clk/ids simulates a process restart.
func laEngine(st core.StoragePort, clk *fakeClock, ids func() string, maxParallel int, handlers ...core.StepHandler) *Engine {
	nr := native.New()
	for _, h := range handlers {
		nr.Register(h)
	}
	return New(st, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: maxParallel, Clock: clk.now, NewID: ids}, discardLogger())
}

// laSetup opens a fresh SQLite store (own TempDir via openStorage), registers def,
// and returns the store with the fault layer on top (initially unarmed).
func laSetup(t *testing.T, def core.WorkflowDefinition) (*storage.SQLiteStorage, *laFaultStorage) {
	t.Helper()
	s := openStorage(t)
	if err := s.RegisterWorkflow(context.Background(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	return s, newLAFaultStorage(s)
}

func laSubmit(t *testing.T, e *Engine, defID string) core.InstanceID {
	t.Helper()
	iid, err := e.Submit(context.Background(), defID, "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return iid
}

// laDefSingle: one native step "a" that is both initial and final.
func laDefSingle() core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.single", Version: "1.0.0", Namespace: "la", Name: "single",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("a", "la_a")},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
}

// laDefFanOut: root → {b1, b2}, both b1 and b2 final (no join).
func laDefFanOut() core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.fanout", Version: "1.0.0", Namespace: "la", Name: "fanout",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("root", "la_root"), nativeStep("b1", "la_b1"), nativeStep("b2", "la_b2"),
		},
		Transitions: []core.Transition{{From: "root", To: "b1"}, {From: "root", To: "b2"}},
		InitialStep: "root", FinalSteps: []string{"b1", "b2"}, Metadata: map[string]any{},
	}
}

// laDefWait: native a → signal w (wait_signal "go") → native z (final).
func laDefWait() core.WorkflowDefinition {
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: "la.wait", Version: "1.0.0", Namespace: "la", Name: "wait",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "la_a"), signalStep("w", "go"), nativeStep("z", "la_z"),
		},
		Transitions: []core.Transition{{From: "a", To: "w"}, {From: "w", To: "z"}},
		InitialStep: "a", FinalSteps: []string{"z"}, Metadata: map[string]any{},
	}
}

// laTickUntilTripped ticks e until fs has injected its fault (at most
// laSetupTicks). Tick errors are returned, not fatal: the injected failure is
// expected to surface through them. A fault that never trips means the lead's
// fault site is not reached on this path, which is itself a construction failure.
func laTickUntilTripped(t *testing.T, e *Engine, fs *laFaultStorage) []error {
	t.Helper()
	var errs []error
	for n := 1; n <= laSetupTicks; n++ {
		if err := e.Tick(context.Background()); err != nil {
			errs = append(errs, err)
		}
		if fs.trippedCount() > 0 {
			return errs
		}
	}
	t.Fatalf("injected fault never tripped within %d setup ticks: lead's fault site not reached", laSetupTicks)
	return nil
}

// laWantInjected asserts the injected failure propagated out of Tick (the engine
// must surface a failed claim/emission/wait write, never swallow it).
func laWantInjected(t *testing.T, errs []error) {
	t.Helper()
	if !errors.Is(errors.Join(errs...), errLAInjected) {
		t.Fatalf("injected fault did not surface through Tick: errs=%v", errs)
	}
}

func laInstance(t *testing.T, s *storage.SQLiteStorage, iid core.InstanceID) core.WorkflowInstance {
	t.Helper()
	inst, err := s.GetInstance(context.Background(), iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	return inst
}

func laIsTerminal(inst core.WorkflowInstance) bool { return isTerminal(inst.Status) }

// laTickUntil ticks e at most bound times, stopping as soon as pred holds. Tick
// errors are collected, not fatal (see laTickUntilTripped).
func laTickUntil(t *testing.T, e *Engine, s *storage.SQLiteStorage, iid core.InstanceID, bound int, pred func(core.WorkflowInstance) bool) (int, []error) {
	t.Helper()
	var errs []error
	for n := 1; n <= bound; n++ {
		if err := e.Tick(context.Background()); err != nil {
			errs = append(errs, err)
		}
		if pred(laInstance(t, s, iid)) {
			return n, errs
		}
	}
	return bound, errs
}

func laEvents(t *testing.T, s *storage.SQLiteStorage, iid core.InstanceID) []core.ExecutionEvent {
	t.Helper()
	evs, err := s.ReadEvents(context.Background(), iid, 0)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	return evs
}

// laStepOutcome reports how stepID ended: the LATEST of {StepStarted,
// StepCompleted, StepFailed} for that step, with StepFailed's code and retrying
// flag decoded from its payload. "StepStarted (never settled)" = stranded.
func laStepOutcome(t *testing.T, evs []core.ExecutionEvent, stepID string) string {
	t.Helper()
	out := "none"
	for _, ev := range evs {
		if ev.StepID != stepID {
			continue
		}
		switch ev.EventType {
		case core.EventTypeStepStarted:
			out = "StepStarted (never settled)"
		case core.EventTypeStepCompleted:
			out = "StepCompleted"
		case core.EventTypeStepFailed:
			var p stepFailedPayload
			decodePayload(t, ev, &p)
			out = fmt.Sprintf("StepFailed(code=%s,retrying=%t)", p.Error.Code, p.Retrying)
		}
	}
	return out
}

// laReport logs the wedge/terminal state: status, current_steps, the ordered
// event stream (seq:type[step]), live wait_records and undelivered inbox signals.
func laReport(t *testing.T, s *storage.SQLiteStorage, iid core.InstanceID, label string) {
	t.Helper()
	ctx := context.Background()
	inst := laInstance(t, s, iid)
	names := make([]string, 0)
	for _, ev := range laEvents(t, s, iid) {
		n := fmt.Sprintf("%d:%s", ev.SequenceNum, ev.EventType)
		if ev.StepID != "" {
			n += "[" + ev.StepID + "]"
		}
		names = append(names, n)
	}
	waits, werr := s.ListWaitRecordsByInstance(ctx, iid)
	sigs, serr := s.ListUndeliveredSignals(ctx, iid)
	t.Logf("%s: status=%s current_steps=%v", label, inst.Status, inst.CurrentSteps)
	t.Logf("%s: events=[%s]", label, strings.Join(names, " "))
	t.Logf("%s: live_wait_records=%d (err=%v) undelivered_signals=%d (err=%v)",
		label, len(waits), werr, len(sigs), serr)
}

// laExpect is one expected step outcome; the actual outcome must start with it.
type laExpect struct {
	step    string
	outcome string
}

// laCheck logs the state and asserts the terminal status and step outcomes.
func laCheck(t *testing.T, s *storage.SQLiteStorage, iid core.InstanceID, label string, want core.InstanceStatus, steps ...laExpect) {
	t.Helper()
	laReport(t, s, iid, label)
	inst := laInstance(t, s, iid)
	evs := laEvents(t, s, iid)
	if inst.Status != want {
		t.Errorf("%s: status=%s current_steps=%v; want status=%s",
			label, inst.Status, inst.CurrentSteps, want)
	}
	for _, x := range steps {
		got := laStepOutcome(t, evs, x.step)
		t.Logf("%s: step %q outcome=%s", label, x.step, got)
		if !strings.HasPrefix(got, x.outcome) {
			t.Errorf("%s: step %q outcome=%s; want prefix %q", label, x.step, got, x.outcome)
		}
	}
}

func laLabel(lead string, restart bool) string {
	if restart {
		return lead + " after-restart"
	}
	return lead + " same-process"
}

func laCalls(h *stepHandler) int32 { return atomic.LoadInt32(&h.calls) }

// laHasWorkerCrash reports whether any persisted StepFailed carries code worker_crash.
func laHasWorkerCrash(t *testing.T, evs []core.ExecutionEvent) bool {
	t.Helper()
	for _, ev := range evs {
		if ev.EventType != core.EventTypeStepFailed {
			continue
		}
		var p stepFailedPayload
		decodePayload(t, ev, &p)
		if p.Error.Code == "worker_crash" {
			return true
		}
	}
	return false
}

// laAssertNoWorkerCrash fails if any StepFailed{worker_crash} was persisted.
// CE ruling LA-C1: for L2, L4a, L4b and L5 the RC-1 worker_crash path must not be
// taken at all; the wait/sibling/delivery must be completed, not failed.
func laAssertNoWorkerCrash(t *testing.T, label string, evs []core.ExecutionEvent) {
	t.Helper()
	if laHasWorkerCrash(t, evs) {
		t.Errorf("%s: CE ruling LA-C1: a worker_crash StepFailed was persisted; want none", label)
	}
}

// laHasEvent reports whether evs contains an event of type et.
func laHasEvent(evs []core.ExecutionEvent, et core.EventType) bool {
	for _, ev := range evs {
		if ev.EventType == et {
			return true
		}
	}
	return false
}

// ── L1 settleSuccess: StepCompleted(a) append fails once ──────────────────────
// CE ruling LA-C1: the expectation is the RC-1 restart contract, not StepCompleted.
// The handler ran and its result is cached, but StepCompleted(a) was never
// persisted, so RC-1 settles "a" as worker_crash, the instance fails, and the
// handler executes exactly once in total.
// open question LA-Q1: a successful handler result whose StepCompleted append failed is recovered as worker_crash per RC-1; recovering it from the idempotency cache would be new semantics — founder decision, not implemented.

func laScenarioL1(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("L1", restart)
	def := laDefSingle()
	s, fs := laSetup(t, def)
	h := &stepHandler{id: "la_a", outputs: map[string]any{"msg": "done"}}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 1, h)
	iid := laSubmit(t, e, def.ID)

	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepCompleted, stepID: "a"})
	laWantInjected(t, laTickUntilTripped(t, e, fs))
	if restart {
		e = laEngine(fs, clk, ids, 1, h) // process restart: first engine dropped
	}
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count=%d", label, n, len(errs), laCalls(h))
	// CE ruling LA-C1: instance failed; "a" settled as worker_crash (RC-1); handler
	// executed exactly once. Open question LA-Q1 applies (see the L1 header).
	laCheck(t, s, iid, label, core.InstanceStatusFailed, laExpect{"a", "StepFailed(code=worker_crash"})
	if got := laCalls(h); got != 1 {
		t.Errorf("%s: CE ruling LA-C1: handler executions = %d, want 1", label, got)
	}
}

func TestLA_L1_SettleSuccess_StepCompletedAppendFails(t *testing.T) {
	laScenarioL1(t, false)
}

// CE ruling LA-C1: regression guard, not a defect detector; pins the already-holding RC-1 restart contract and passes pre-fix.
func TestLA_L1_SettleSuccess_StepCompletedAppendFails_AfterRestart(t *testing.T) {
	laScenarioL1(t, true)
}

// ── L2 gatherDispatch: sibling ClaimStep fails after first sibling started ────
// Expected (card): both b1 and b2 complete; b1 is not marked failed although it
// never ran.

func laScenarioL2(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("L2", restart)
	def := laDefFanOut()
	s, fs := laSetup(t, def)
	root := &stepHandler{id: "la_root", outputs: map[string]any{}}
	b1 := &stepHandler{id: "la_b1", outputs: map[string]any{"who": "b1"}}
	b2 := &stepHandler{id: "la_b2", outputs: map[string]any{"who": "b2"}}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 2, root, b1, b2)
	iid := laSubmit(t, e, def.ID)

	// Tick 1 settles root; tick 2 claims+starts b1, then the claim for b2 fails.
	fs.arm(laFault{op: laOpClaimStep, nth: 1, stepID: "b2"})
	laWantInjected(t, laTickUntilTripped(t, e, fs))
	if restart {
		e = laEngine(fs, clk, ids, 2, root, b1, b2)
	}
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count root=%d b1=%d b2=%d",
		label, n, len(errs), laCalls(root), laCalls(b1), laCalls(b2))
	laCheck(t, s, iid, label, core.InstanceStatusCompleted,
		laExpect{"b1", "StepCompleted"}, laExpect{"b2", "StepCompleted"})
	// CE ruling LA-C1: b1 and b2 each StepCompleted, each handler executed exactly
	// once, and no worker_crash anywhere (b1 really ran; it must not be failed).
	if laCalls(root) != 1 || laCalls(b1) != 1 || laCalls(b2) != 1 {
		t.Errorf("%s: CE ruling LA-C1: handler executions root=%d b1=%d b2=%d; want 1 each",
			label, laCalls(root), laCalls(b1), laCalls(b2))
	}
	laAssertNoWorkerCrash(t, label, laEvents(t, s, iid))
}

func TestLA_L2_GatherDispatch_SiblingClaimFails(t *testing.T) {
	laScenarioL2(t, false)
}

func TestLA_L2_GatherDispatch_SiblingClaimFails_AfterRestart(t *testing.T) {
	laScenarioL2(t, true)
}

// ── L3 settleFailure: StepFailed(a) append fails once ─────────────────────────
// Expected (card): terminal failed, with "a" ending StepFailed (not stuck running).

func laScenarioL3(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("L3", restart)
	def := laDefSingle()
	s, fs := laSetup(t, def)
	h := &stepHandler{id: "la_a", err: errLAHandler}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 1, h)
	iid := laSubmit(t, e, def.ID)

	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepFailed, stepID: "a"})
	laWantInjected(t, laTickUntilTripped(t, e, fs))
	if restart {
		e = laEngine(fs, clk, ids, 1, h)
	}
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count=%d", label, n, len(errs), laCalls(h))
	// CE ruling LA-C1: the StepFailed(a) append for the handler error failed, so RC-1
	// settles "a" as worker_crash (no retry policy in this fixture) and the instance
	// fails; the handler executes exactly once in total.
	laCheck(t, s, iid, label, core.InstanceStatusFailed, laExpect{"a", "StepFailed(code=worker_crash"})
	if got := laCalls(h); got != 1 {
		t.Errorf("%s: CE ruling LA-C1: handler executions = %d, want 1", label, got)
	}
}

func TestLA_L3_SettleFailure_StepFailedAppendFails(t *testing.T) {
	laScenarioL3(t, false)
}

// CE ruling LA-C1: regression guard, not a defect detector; pins the already-holding RC-1 restart contract and passes pre-fix.
func TestLA_L3_SettleFailure_StepFailedAppendFails_AfterRestart(t *testing.T) {
	laScenarioL3(t, true)
}

// ── L4a / L4b enterWait: the wait-entry write fails once ──────────────────────
// L4a: CreateWaitRecord(w) fails. L4b: the running→waiting UpsertInstance fails
// (its wait_record IS already durable). Both then receive signal "go".
// Expected (card): instance waits, receives the signal, completes; "w" is NOT
// failed with worker_crash.

func laScenarioWait(t *testing.T, restart bool, lead string, ft laFault) {
	t.Helper()
	label := laLabel(lead, restart)
	def := laDefWait()
	s, fs := laSetup(t, def)
	ha := &stepHandler{id: "la_a", outputs: map[string]any{}}
	hz := &stepHandler{id: "la_z", outputs: map[string]any{}}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 1, ha, hz)
	iid := laSubmit(t, e, def.ID)

	fs.arm(ft)
	laWantInjected(t, laTickUntilTripped(t, e, fs))
	if restart {
		e = laEngine(fs, clk, ids, 1, ha, hz)
	}
	// CE ruling LA-C1: the instance must pass through `waiting` (the wait is re-entered
	// by RC-1, F3b, or its duplicate wait_record completes the running→waiting write,
	// F3/L4b) BEFORE the signal is sent; it must then receive the signal and complete.
	laTickUntil(t, e, s, iid, laK, func(i core.WorkflowInstance) bool {
		return i.Status == core.InstanceStatusWaiting
	})
	if got := laInstance(t, s, iid).Status; got != core.InstanceStatusWaiting {
		t.Fatalf("%s: CE ruling LA-C1: instance never reached waiting within %d ticks: status=%s", label, laK, got)
	}
	if err := e.Signal(context.Background(), iid, "go", map[string]any{"by": "alice"}); err != nil {
		t.Fatalf("Signal: %v", err)
	}
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count a=%d z=%d",
		label, n, len(errs), laCalls(ha), laCalls(hz))
	// CE ruling LA-C1: w and z StepCompleted, z executed exactly once, the signal was
	// delivered (SignalReceived persisted), and no worker_crash anywhere.
	laCheck(t, s, iid, label, core.InstanceStatusCompleted,
		laExpect{"w", "StepCompleted"}, laExpect{"z", "StepCompleted"})
	evs := laEvents(t, s, iid)
	if !laHasEvent(evs, core.EventTypeSignalReceived) {
		t.Errorf("%s: CE ruling LA-C1: no SignalReceived persisted; the signal was not delivered", label)
	}
	if got := laCalls(hz); got != 1 {
		t.Errorf("%s: CE ruling LA-C1: z handler executions = %d, want 1", label, got)
	}
	laAssertNoWorkerCrash(t, label, evs)
}

func TestLA_L4a_EnterWait_CreateWaitRecordFails(t *testing.T) {
	laScenarioWait(t, false, "L4a", laFault{op: laOpCreateWaitRecord, nth: 1, stepID: "w"})
}

func TestLA_L4a_EnterWait_CreateWaitRecordFails_AfterRestart(t *testing.T) {
	laScenarioWait(t, true, "L4a", laFault{op: laOpCreateWaitRecord, nth: 1, stepID: "w"})
}

func TestLA_L4b_EnterWait_RunningToWaitingUpsertFails(t *testing.T) {
	laScenarioWait(t, false, "L4b", laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusWaiting})
}

func TestLA_L4b_EnterWait_RunningToWaitingUpsertFails_AfterRestart(t *testing.T) {
	laScenarioWait(t, true, "L4b", laFault{op: laOpUpsertInstance, nth: 1, status: core.InstanceStatusWaiting})
}

// ── L5 post-delivery CompleteStep: StepCompleted(w) append fails once ─────────
// Fault armed only after the instance is parked in `waiting`. It fires inside
// SIGNAL_SCAN after the B3 commit (Tick itself returns nil; the scan logs it).
// Expected (card): instance completes; "w" StepCompleted with the signal payload.
// CE ruling LA-C1: additionally no worker_crash anywhere; z executes exactly once.

func TestLA_L5_PostDelivery_StepCompletedAppendFails(t *testing.T) {
	laScenarioL5(t, false)
}

func TestLA_L5_PostDelivery_StepCompletedAppendFails_AfterRestart(t *testing.T) {
	laScenarioL5(t, true)
}

func laScenarioL5(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("L5", restart)
	def := laDefWait()
	s, fs := laSetup(t, def)
	ha := &stepHandler{id: "la_a", outputs: map[string]any{}}
	hz := &stepHandler{id: "la_z", outputs: map[string]any{}}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 1, ha, hz)
	iid := laSubmit(t, e, def.ID)

	// Park in `waiting` with NO fault armed.
	n, errs := laTickUntil(t, e, s, iid, laSetupTicks, func(i core.WorkflowInstance) bool {
		return i.Status == core.InstanceStatusWaiting
	})
	if got := laInstance(t, s, iid).Status; got != core.InstanceStatusWaiting {
		t.Fatalf("%s: setup: status=%s after %d ticks, want waiting", label, got, n)
	}
	if len(errs) != 0 {
		t.Fatalf("%s: setup: unexpected Tick error before the fault: %v", label, errs)
	}

	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepCompleted, stepID: "w"})
	if err := e.Signal(context.Background(), iid, "go", map[string]any{"by": "alice"}); err != nil {
		t.Fatalf("Signal: %v", err)
	}
	// The fault fires inside SIGNAL_SCAN, after the B3 commit.
	deliverErrs := laTickUntilTripped(t, e, fs)
	t.Logf("%s: fault tripped inside SIGNAL_SCAN; Tick errors on the trip tick=%d", label, len(deliverErrs))
	if restart {
		e = laEngine(fs, clk, ids, 1, ha, hz)
	}
	n, errs = laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count a=%d z=%d",
		label, n, len(errs), laCalls(ha), laCalls(hz))
	laCheck(t, s, iid, label, core.InstanceStatusCompleted,
		laExpect{"w", "StepCompleted"}, laExpect{"z", "StepCompleted"})

	// On completion, "w" must carry the signal payload as its outputs.
	evs := laEvents(t, s, iid)
	for _, ev := range evs {
		if ev.EventType != core.EventTypeStepCompleted || ev.StepID != "w" {
			continue
		}
		var p stepCompletedPayload
		decodePayload(t, ev, &p)
		assertJSONEqual(t, p.Outputs, map[string]any{"by": "alice"})
	}
	// CE ruling LA-C1: z executes exactly once and no worker_crash is persisted; the
	// delivered signal's step "w" completes (it must not be failed by RC-1).
	if got := laCalls(hz); got != 1 {
		t.Errorf("%s: CE ruling LA-C1: z handler executions = %d, want 1", label, got)
	}
	laAssertNoWorkerCrash(t, label, evs)
}

// ── L6 hydrate RC-1: the RC-1 StepFailed(worker_crash) append fails once ──────
// Phase 1 (L1's fault, FIRST engine) leaves StepStarted(a) stranded; that engine
// is then dropped. Phase 2: a fresh engine's RC-1 StepFailed(a) append fails once.
// Expected (card): the fresh engine still recovers (reaches a terminal status)
// within K ticks in the same process. The AfterRestart variant ticks a third
// engine; the card gives no outcome beyond "recovers", so it asserts terminal.
// CE ruling LA-C1: both variants assert the RC-1 contract outcome (see the tail).

func laScenarioL6(t *testing.T, restart bool) {
	t.Helper()
	label := laLabel("L6", restart)
	def := laDefSingle()
	s, fs := laSetup(t, def)
	h := &stepHandler{id: "la_a", outputs: map[string]any{"msg": "done"}}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()

	// Phase 1: stranded StepStarted(a); the first engine is dropped.
	e1 := laEngine(fs, clk, ids, 1, h)
	iid := laSubmit(t, e1, def.ID)
	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepCompleted, stepID: "a"})
	laWantInjected(t, laTickUntilTripped(t, e1, fs))

	// Phase 2: fresh engine; its RC-1 StepFailed(a, worker_crash) append fails once.
	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepFailed, stepID: "a"})
	e2 := laEngine(fs, clk, ids, 1, h)
	laWantInjected(t, laTickUntilTripped(t, e2, fs))
	e := e2
	if restart {
		e = laEngine(fs, clk, ids, 1, h)
	}
	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("%s: ticks=%d tick_errors=%d handler_exec_count=%d", label, n, len(errs), laCalls(h))
	// CE ruling LA-C1: terminal within K, and the RC-1 contract outcome: instance
	// failed, "a" settled as worker_crash (no retry policy here), and the handler
	// executed exactly once in total (RC-1 never re-runs the handler).
	laCheck(t, s, iid, label, core.InstanceStatusFailed, laExpect{"a", "StepFailed(code=worker_crash"})
	if got := laCalls(h); got != 1 {
		t.Errorf("%s: CE ruling LA-C1: handler executions = %d, want 1", label, got)
	}
}

func TestLA_L6_HydrateRC1_StepFailedAppendFails(t *testing.T) {
	laScenarioL6(t, false)
}

// CE ruling LA-C1: regression guard, not a defect detector; pins the already-holding RC-1 restart contract and passes pre-fix.
func TestLA_L6_HydrateRC1_StepFailedAppendFails_AfterRestart(t *testing.T) {
	laScenarioL6(t, true)
}

// ── F1 (LA-C1): in-process recovery ≡ restart recovery ──────────────────────────
// TestLA_F1_RecoveryResetsBookkeeping proves that resetInstanceRecovery discards
// every per-instance entry hydrate reconstructs, so a STALE in-memory retry cannot
// survive the reset and resurrect a step. Setup is the L1 fault: StepStarted(a) is
// stranded (the handler ran once; StepCompleted(a) was never persisted). Stale
// entries are then planted for iid — a due retry for "a" at attempt 2 (the
// resurrection vector) plus pending/failed/claim-lost markers — and an unrelated
// retry for another instance id. After the reset, iid's entries are gone and the
// unrelated one survives; the next tick settles "a" by RC-1 (worker_crash, instance
// failed) and the handler is NOT executed again.
func TestLA_F1_RecoveryResetsBookkeeping(t *testing.T) {
	def := laDefSingle()
	s, fs := laSetup(t, def)
	h := &stepHandler{id: "la_a", outputs: map[string]any{"msg": "done"}}
	clk := newFakeClock(engineStart, 10*time.Millisecond)
	ids := laIDs()
	e := laEngine(fs, clk, ids, 1, h)
	iid := laSubmit(t, e, def.ID)

	fs.arm(laFault{op: laOpAppendEvent, nth: 1, eventType: core.EventTypeStepCompleted, stepID: "a"})
	laWantInjected(t, laTickUntilTripped(t, e, fs))

	const unrelated = core.InstanceID("la-f1-unrelated")
	stepKey := retryKey{iid: iid, step: "a"}
	e.mu.Lock()
	e.hydrated[iid] = true
	e.retries[stepKey] = retrySched{nextAttempt: 2, nextAttemptAt: clk.now().Add(-time.Second)}
	e.pending[iid] = map[string]bool{"a": true}
	e.failed[iid] = map[string]bool{"a": true}
	e.claimLostStreak[stepKey] = 3
	e.retries[retryKey{iid: unrelated, step: "a"}] = retrySched{nextAttempt: 2, nextAttemptAt: clk.now()}
	e.mu.Unlock()

	e.resetInstanceRecovery(iid)

	e.mu.Lock()
	_, hydrated := e.hydrated[iid]
	_, hasRetry := e.retries[stepKey]
	_, hasPending := e.pending[iid]
	_, hasFailed := e.failed[iid]
	_, hasStreak := e.claimLostStreak[stepKey]
	_, unrelatedRetry := e.retries[retryKey{iid: unrelated, step: "a"}]
	e.mu.Unlock()
	if hydrated || hasRetry || hasPending || hasFailed || hasStreak {
		t.Errorf("F1 (LA-C1): reset left iid bookkeeping behind: hydrated=%t retry=%t pending=%t failed=%t claim_lost_streak=%t",
			hydrated, hasRetry, hasPending, hasFailed, hasStreak)
	}
	if !unrelatedRetry {
		t.Errorf("F1 (LA-C1): reset of %s also cleared an unrelated instance's retry entry", iid)
	}

	n, errs := laTickUntil(t, e, s, iid, laK, laIsTerminal)
	t.Logf("F1: ticks=%d tick_errors=%d handler_exec_count=%d", n, len(errs), laCalls(h))
	laCheck(t, s, iid, "F1", core.InstanceStatusFailed, laExpect{"a", "StepFailed(code=worker_crash"})
	if got := laCalls(h); got != 1 {
		t.Errorf("F1 (LA-C1): stale retry resurrected a step: handler executions = %d, want 1", got)
	}
}
