package engine

// Cancellation FSM E2E (Finalization B4, EDR-011 §8). The four cancellation
// modes the frozen text distinguishes — in-flight (post-settle gate), terminal
// no-op, pending-immediate, --compensate — plus the cancel-during-retry-wait
// interleaving, each asserted as an EXACT event sequence and closed with the
// EDR-007 forward ≡ RebuildState property.

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// cancelHandler is a StepHandler that requests cancellation of its OWN instance
// from inside Execute (the B4 in-flight case: the engine's post-settle gate
// picks it up on the same tick). It reaches the engine through a back-reference
// set by the test after construction and reads its instance id from StepContext.
type cancelHandler struct {
	id         string
	eng        *Engine
	reason     string
	compensate bool
	outputs    map[string]any
	calls      int32
}

func (h *cancelHandler) ID() string { return h.id }

func (h *cancelHandler) Execute(sc core.StepContext) (core.StepResult, error) {
	atomic.AddInt32(&h.calls, 1)
	// In-flight by definition: this runs during DISPATCH, before SETTLE.
	if err := h.eng.Cancel(context.Background(), core.InstanceID(sc.InstanceID), h.reason, h.compensate); err != nil {
		return core.StepResult{}, err
	}
	return core.StepResult{Outputs: h.outputs}, nil
}

// newBufferEngine builds a native-only engine whose logger writes to buf (for
// the exact-warning capture, stepcontext_test.go's idiom) with a manual clock.
func newBufferEngine(t *testing.T, def core.WorkflowDefinition, buf *bytes.Buffer, handlers ...core.StepHandler) (*Engine, *storage.SQLiteStorage) {
	t.Helper()
	s := openStorage(t)
	if err := s.RegisterWorkflow(context.Background(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	nr := native.New()
	for _, h := range handlers {
		nr.Register(h)
	}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: 1, Clock: newManualClock(engineStart).now}, logger)
	return e, s
}

// ── B4 in-flight: cancel from inside the running step ─────────────────────────

// TestCancel_B4InFlightFromHandler is the Finalization B4 fixture: step1's
// handler cancels the instance while it is in flight; step2 NEVER starts and the
// instance finalizes cancelled. The [cancel] marker is positional (it appends no
// event of its own) — it falls between StepStarted{step1} and StepCompleted{step1}
// in wall-time but the ONLY terminal event is WorkflowCancelled.
func TestCancel_B4InFlightFromHandler(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.b4", Version: "1.0.0", Namespace: "t", Name: "b4",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("step1", "h1"),
			nativeStep("step2", "h2"),
		},
		Transitions: []core.Transition{{From: "step1", To: "step2"}},
		InitialStep: "step1", FinalSteps: []string{"step2"}, Metadata: map[string]any{},
	}
	h1 := &cancelHandler{id: "h1", reason: "user requested", compensate: false, outputs: map[string]any{"r": "s1"}}
	h2 := &stepHandler{id: "h2", outputs: map[string]any{"r": "s2"}}

	e, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now, h1, h2)
	h1.eng = e
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.b4", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCancelled {
		t.Fatalf("status = %q, want cancelled", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "step1"},
		{core.EventTypeStepCompleted, "step1"},
		{core.EventTypeWorkflowCancelled, ""},
	}
	assertPairs(t, pairs, want)

	// step2 NEVER starts (B4 step 3: no next step activates once cancellation is set).
	if idx := indexer(pairs)(core.EventTypeStepStarted, "step2"); idx != -1 {
		t.Fatalf("step2 must never start under cancellation; found StepStarted at %d", idx)
	}

	wc := findEvent(t, evs, core.EventTypeWorkflowCancelled, "")
	var wcp workflowCancelledPayload
	decodePayload(t, wc, &wcp)
	if wcp.Reason != "user requested" {
		t.Fatalf("WorkflowCancelled.reason = %q, want %q", wcp.Reason, "user requested")
	}

	assertProjectionEquivalence(t, s, iid)
}

// ── Terminal no-op: cancel an already-completed instance ──────────────────────

func TestCancel_TerminalNoOpWarning(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.done", Version: "1.0.0", Namespace: "t", Name: "done",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("a", "ha")},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	var buf bytes.Buffer
	e, s := newBufferEngine(t, def, &buf, &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}})
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.done", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}

	if err := e.Cancel(ctx, iid, "too late", false); err != nil {
		t.Fatalf("Cancel of a completed instance must return nil, got %v", err)
	}
	// EXACT B4 idempotency warning.
	want := "instance " + string(iid) + " is already in terminal state completed; cancel is a no-op"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("missing exact B4 no-op warning\n want substring: %s\n log: %s", want, buf.String())
	}

	assertProjectionEquivalence(t, s, iid)
}

// ── Pending-immediate: cancel a pending instance (no events, no steps) ────────

func TestCancel_PendingImmediate(t *testing.T) {
	s := openStorage(t)
	e := New(s, nil, Config{MaxParallelSteps: 1, Clock: newManualClock(engineStart).now}, discardLogger())
	ctx := context.Background()

	iid := core.InstanceID("inst-pending")
	// Seed a pending-status row directly (no events). StartedAt is left zero so
	// the WorkflowCancelled-only event stream rebuilds to an equivalent row.
	seed := core.WorkflowInstance{
		InstanceID:        iid,
		DefinitionID:      "t.pending",
		DefinitionVersion: "1.0.0",
		Namespace:         "t",
		Status:            core.InstanceStatusPending,
		CurrentSteps:      []string{},
		Variables:         map[string]any{},
	}
	if err := s.UpsertInstance(ctx, seed, 0); err != nil {
		t.Fatalf("UpsertInstance seed: %v", err)
	}

	if err := e.Cancel(ctx, iid, "abort", false); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	inst, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if inst.Status != core.InstanceStatusCancelled {
		t.Fatalf("status = %q, want cancelled", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{{core.EventTypeWorkflowCancelled, ""}}
	assertPairs(t, pairs, want) // no StepStarted anywhere: no steps executed.

	wc := findEvent(t, evs, core.EventTypeWorkflowCancelled, "")
	var wcp workflowCancelledPayload
	decodePayload(t, wc, &wcp)
	if wcp.Reason != "abort" {
		t.Fatalf("WorkflowCancelled.reason = %q, want abort", wcp.Reason)
	}

	assertProjectionEquivalence(t, s, iid)
}

// ── --compensate cancel: cancel with compensation from inside the 2nd step ────

func TestCancel_CompensateFromHandler(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.ccomp", Version: "1.0.0", Namespace: "t", Name: "ccomp",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("step1", "h1"),
			nativeStep("step2", "h2"),
		},
		Transitions: []core.Transition{{From: "step1", To: "step2"}},
		Compensation: &core.CompensationPlan{Steps: []core.CompensationStep{
			{StepID: "step1", UndoHandler: "undo-step1"},
			{StepID: "step2", UndoHandler: "undo-step2"},
		}},
		InitialStep: "step1", FinalSteps: []string{"step2"}, Metadata: map[string]any{},
	}
	var order []string
	var mu sync.Mutex
	h1 := &stepHandler{id: "h1", outputs: map[string]any{"r": "s1"}}
	h2 := &cancelHandler{id: "h2", reason: "rollback", compensate: true, outputs: map[string]any{"r": "s2"}}
	undo1 := &recordingHandler{id: "undo-step1", order: &order, mu: &mu}
	undo2 := &recordingHandler{id: "undo-step2", order: &order, mu: &mu}

	e, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now, h1, h2, undo1, undo2)
	h2.eng = e
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.ccomp", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompensated {
		t.Fatalf("status = %q, want compensated", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "step1"}, {core.EventTypeStepCompleted, "step1"},
		{core.EventTypeStepStarted, "step2"}, {core.EventTypeStepCompleted, "step2"},
		{core.EventTypeWorkflowCompensating, ""},
		{core.EventTypeWorkflowCompensated, ""},
	}
	assertPairs(t, pairs, want)

	// NO WorkflowFailed and NO WorkflowCancelled on a --compensate cancel (B4 step 4).
	idx := indexer(pairs)
	if i := idx(core.EventTypeWorkflowFailed, ""); i != -1 {
		t.Fatalf("--compensate cancel must NOT emit WorkflowFailed; found at %d", i)
	}
	if i := idx(core.EventTypeWorkflowCancelled, ""); i != -1 {
		t.Fatalf("--compensate cancel must NOT emit WorkflowCancelled; found at %d", i)
	}

	// Undos run in REVERSE plan order over the completed steps.
	mu.Lock()
	got := append([]string{}, order...)
	mu.Unlock()
	if strings.Join(got, ",") != "undo-step2,undo-step1" {
		t.Fatalf("undo order = %v, want [undo-step2 undo-step1]", got)
	}

	// WorkflowCompensating.from_step is empty (no prior failure — cancellation).
	wcg := findEvent(t, evs, core.EventTypeWorkflowCompensating, "")
	var wcgp workflowCompensatingPayload
	decodePayload(t, wcg, &wcgp)
	if wcgp.FromStep != "" {
		t.Fatalf("WorkflowCompensating.from_step = %q, want empty (cancellation)", wcgp.FromStep)
	}

	assertProjectionEquivalence(t, s, iid)
}

// ── Cancel during signal wait (M07-C3r B4.6) ─────────────────────────────────

// TestCancel_DuringWait asserts Finalization B4 step 6: when Cancel is called on
// a waiting instance (signal step parked), the engine deletes its wait_records,
// emits StepFailed{code:cancelled} for the parked step, and finalizes with
// WorkflowCancelled. The event sequence matches the M06 cancellation sequence;
// assertProjectionEquivalence passes because the transient `waiting` status
// resolves to `cancelled` identically under both forward projection and
// RebuildState (EDR-007 §9 gap closes before the terminal event).
func TestCancel_DuringWait(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.cw", Version: "1.0.0", Namespace: "t", Name: "cw",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{signalStep("w", "go")},
		InitialStep: "w", FinalSteps: []string{"w"}, Metadata: map[string]any{},
	}
	clk := newManualClock(engineStart)
	e, s := newNativeEngine(t, def, 1, clk.now)
	ctx := context.Background()

	iid, err := e.Submit(ctx, "t.cw", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick 1: signal step activates and parks the instance in `waiting`.
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 1: %v", err)
	}
	inst, _ := s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusWaiting {
		t.Fatalf("after WAIT entry status = %q, want waiting", inst.Status)
	}

	// Wait_record must exist before cancellation.
	_, wrFound, err := s.GetWaitRecord(ctx, iid, "w")
	if err != nil {
		t.Fatalf("GetWaitRecord before cancel: %v", err)
	}
	if !wrFound {
		t.Fatalf("wait_record must exist before cancellation")
	}

	// Cancel while the instance is waiting — B4.6 path.
	if err := e.Cancel(ctx, iid, "stop waiting", false); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	// Cancel is synchronous for the waiting state: instance is cancelled immediately.
	inst, _ = s.GetInstance(ctx, iid)
	if inst.Status != core.InstanceStatusCancelled {
		t.Fatalf("after Cancel status = %q, want cancelled", inst.Status)
	}

	// semantics-bearing: B4 step 6 — wait_record deleted on cancellation.
	_, wrFound, err = s.GetWaitRecord(ctx, iid, "w")
	if err != nil {
		t.Fatalf("GetWaitRecord after cancel: %v", err)
	}
	if wrFound {
		t.Fatalf("wait_record must be deleted on cancellation (B4.6)")
	}

	// Exact event sequence: WorkflowStarted, StepStarted("w"), StepFailed("w"),
	// WorkflowCancelled — matching the M06 cancellation sequence with a
	// signal-step StepFailed instead of a native StepFailed.
	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "w"},
		{core.EventTypeStepFailed, "w"},  // code="cancelled", retrying=false
		{core.EventTypeWorkflowCancelled, ""},
	}
	assertPairs(t, pairs, want)

	// StepFailed.error.code = "cancelled" (not "signal_timeout").
	sf := findEvent(t, evs, core.EventTypeStepFailed, "w")
	var sfp struct {
		Retrying bool           `json:"retrying"`
		Error    core.StepError `json:"error"`
	}
	decodePayload(t, sf, &sfp)
	if sfp.Error.Code != "cancelled" {
		t.Fatalf("StepFailed.error.code = %q, want cancelled", sfp.Error.Code)
	}
	if sfp.Retrying {
		t.Fatalf("StepFailed.retrying = true, want false")
	}

	// WorkflowCancelled.reason is verbatim.
	wc := findEvent(t, evs, core.EventTypeWorkflowCancelled, "")
	var wcp workflowCancelledPayload
	decodePayload(t, wc, &wcp)
	if wcp.Reason != "stop waiting" {
		t.Fatalf("WorkflowCancelled.reason = %q, want %q", wcp.Reason, "stop waiting")
	}

	// forward ≡ rebuild: the transient `waiting` status resolves to `cancelled`
	// identically under both paths (EDR-007 §9 gap closes before the terminal event).
	assertProjectionEquivalence(t, s, iid)
}

// ── Cancel during retry wait: the scheduled retry never re-dispatches ─────────

func TestCancel_DuringRetryWait(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.crw", Version: "1.0.0", Namespace: "t", Name: "crw",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{{
			ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha",
			Retry: &core.RetryPolicy{Attempts: 3, Backoff: "immediate"},
		}},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", err: errBoom}

	e, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now, ha)
	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.crw", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick 1: attempt 1 fails and schedules a retry (StepFailed{retrying:true}).
	if err := e.Tick(ctx); err != nil {
		t.Fatalf("Tick 1: %v", err)
	}
	pairs, _ := eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepFailed, "a"); n != 1 {
		t.Fatalf("after tick 1: StepFailed count = %d, want 1", n)
	}

	// Cancel while the retry is scheduled (not yet re-dispatched).
	if err := e.Cancel(ctx, iid, "stop", false); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCancelled {
		t.Fatalf("status = %q, want cancelled", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, // attempt 1
		{core.EventTypeStepFailed, "a"},  // retrying:true
		{core.EventTypeStepFailed, "a"},  // retrying:false, code cancelled
		{core.EventTypeWorkflowCancelled, ""},
	}
	assertPairs(t, pairs, want)

	// The retry never re-dispatched: the handler was called exactly once.
	if got := atomic.LoadInt32(&ha.calls); got != 1 {
		t.Fatalf("handler calls = %d, want 1 (retry must not re-dispatch under cancellation)", got)
	}

	// The terminal StepFailed carries code "cancelled" with retrying:false.
	fails := allEvents(evs, core.EventTypeStepFailed, "a")
	var last struct {
		Retrying bool           `json:"retrying"`
		Error    core.StepError `json:"error"`
	}
	decodePayload(t, fails[len(fails)-1], &last)
	if last.Retrying {
		t.Fatalf("terminal StepFailed.retrying = true, want false")
	}
	if last.Error.Code != "cancelled" {
		t.Fatalf("terminal StepFailed.error.code = %q, want cancelled", last.Error.Code)
	}

	assertProjectionEquivalence(t, s, iid)
}
