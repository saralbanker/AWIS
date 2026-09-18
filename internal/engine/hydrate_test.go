package engine

// hydrate_test.go — engine-hardening Step 2a regression: process-restart
// hydration of e.retries / e.pending / e.failed from the durable EventLog
// (hydrate.go). The restart-simulation pattern is
// TestRestart_DurableVersionSurvivesProcessRestart's (restart_test.go):
// build Engine 1 over a storage, tick it into the state under test, then
// construct a BRAND-NEW *Engine over the SAME storage — a fresh process with
// empty e.retries/e.pending/e.failed/e.hydrated maps — and assert the new
// engine resumes correctly. NEVER time.Sleep as synchronisation; the
// manualClock idiom (helpers_test.go) is advanced explicitly instead.

import (
	"context"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// buildEngine2 constructs a second Engine over an EXISTING storage — the
// "process restart" half of the pattern: a fresh Engine whose e.seq/e.ver/
// e.retries/e.pending/e.failed/e.hydrated maps all start empty, wired to the
// SAME durable storage a first engine already drove. It mirrors
// restart_test.go's inline engine-2 construction as a reusable helper for
// this file's several restart fixtures.
func buildEngine2(s *storage.SQLiteStorage, maxParallel int, clock func() time.Time, handlers ...core.StepHandler) *Engine {
	nr := native.New()
	for _, h := range handlers {
		nr.Register(h)
	}
	return New(s, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: maxParallel, Clock: clock}, discardLogger())
}

// ── retries survive restart ─────────────────────────────────────────────────

// TestHydrate_RetryRescheduledAtCorrectAttemptAfterRestart: engine 1
// dispatches step a's attempt 1 (fails, StepFailed{retrying:true}, backoff
// scheduled) and is then discarded (simulating a crash) BEFORE the backoff
// elapses and BEFORE attempt 2 ever dispatches. A brand-new engine 2 over the
// SAME storage and the SAME manualClock (so elapsed-backoff math is
// meaningful across the restart) must: (a) NOT dispatch attempt 2 before the
// backoff elapses, even across repeated ticks (proving hydrate reconstructed
// nextAttemptAt from the StepFailed's emitted_at, not "no schedule found ⇒
// immediately due"); (b) dispatch attempt 2 at the correct attempt number once
// the clock passes the backoff; (c) complete the workflow.
func TestHydrate_RetryRescheduledAtCorrectAttemptAfterRestart(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.hret", Version: "1.0.0", Namespace: "t", Name: "hret",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{{
			ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha",
			Retry: &core.RetryPolicy{Attempts: 2, Backoff: "linear", InitialDelay: "1s", MaxDelay: "30s"},
		}},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	ha1 := &flakyHandler{id: "ha", outputs: map[string]any{"r": "ok"}, err: errBoom, failUntil: 1}

	clk := newManualClock(engineStart)
	ctx := context.Background()

	e1, s := newNativeEngine(t, def, 1, clk.now, ha1)
	iid, err := e1.Submit(ctx, "t.hret", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := e1.Tick(ctx); err != nil { // attempt 1 fails, retry scheduled
		t.Fatalf("Tick 1 (engine1): %v", err)
	}
	pairs, _ := eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("pre-restart StepStarted(a) = %d, want 1", n)
	}

	// "Restart": a brand-new engine over the SAME storage and the SAME clock.
	// ha2 fails unconditionally — attempt 2 succeeding is the ONLY expected
	// outcome under this handler set, so if a broken hydrate mis-scheduled a
	// phantom attempt, the outcome would surface as either an extra
	// StepStarted or a stuck/failed workflow.
	ha2 := &flakyHandler{id: "ha", outputs: map[string]any{"r": "ok"}, err: errBoom, failUntil: 0}
	e2 := buildEngine2(s, 1, clk.now, ha2)

	// Tick engine 2 BEFORE the backoff elapses: must NOT re-dispatch yet.
	if err := e2.Tick(ctx); err != nil {
		t.Fatalf("Tick (engine2, pre-backoff): %v", err)
	}
	pairs, _ = eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("StepStarted(a) after a pre-backoff engine2 tick = %d, want 1 (retry must not fire before backoff elapses)", n)
	}
	// A second pre-backoff tick must also be a no-op (idempotent hydrate +
	// correctly-reconstructed nextAttemptAt, not "advances every tick").
	if err := e2.Tick(ctx); err != nil {
		t.Fatalf("Tick (engine2, pre-backoff, 2nd): %v", err)
	}
	pairs, _ = eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("StepStarted(a) after a second pre-backoff engine2 tick = %d, want 1", n)
	}

	// Advance past the 1s backoff; attempt 2 must now fire and succeed.
	clk.advance(1 * time.Second)
	inst := runToTerminal(t, e2, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, // attempt 1 (engine1)
		{core.EventTypeStepFailed, "a"},  // retrying:true (engine1)
		{core.EventTypeStepStarted, "a"}, // attempt 2 (engine2, post-restart)
		{core.EventTypeStepCompleted, "a"},
		{core.EventTypeWorkflowCompleted, ""},
	}
	assertPairs(t, pairs, want)
	if a := attemptOf(t, findEvent(t, evs, core.EventTypeStepCompleted, "a")); a != 2 {
		t.Fatalf("StepCompleted attempt = %d, want 2 (correct attempt number reconstructed across restart)", a)
	}
	assertProjectionEquivalence(t, s, iid)
}

// ── pending (failure-driven direct activations) survive restart ────────────

// TestHydrate_FallbackPendingTargetRunsAfterRestart: engine 1 runs p to
// completion and then fails a terminally (fallback: fb) — StepFallbackActivated
// is emitted and fb is recorded pending — but engine 1 is discarded BEFORE fb
// ever dispatches (the pending activation was recorded this tick; gatherDispatch
// for the NEXT tick is what would consume it). Engine 2, over the same
// storage, must still run fb to completion, and must NOT re-dispatch the
// already-terminally-failed a (this is B-4 surviving restart, via e.failed).
func TestHydrate_FallbackPendingTargetRunsAfterRestart(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.hfb", Version: "1.0.0", Namespace: "t", Name: "hfb",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("p", "hp"),
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha", Fallback: "fb"},
			nativeStep("fb", "hfb"),
		},
		Transitions: []core.Transition{{From: "p", To: "a"}},
		InitialStep: "p", FinalSteps: []string{"fb"}, Metadata: map[string]any{},
	}
	hp := &stepHandler{id: "hp", outputs: map[string]any{"rp": "p-done"}}
	ha := &stepHandler{id: "ha", err: errBoom}

	clk := newManualClock(engineStart)
	ctx := context.Background()
	e1, s := newNativeEngine(t, def, 1, clk.now, hp, ha)
	iid, err := e1.Submit(ctx, "t.hfb", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := e1.Tick(ctx); err != nil { // p activates + completes
		t.Fatalf("Tick 1 (engine1): %v", err)
	}
	if err := e1.Tick(ctx); err != nil { // a activates, fails terminally, fallback recorded pending
		t.Fatalf("Tick 2 (engine1): %v", err)
	}
	pairs, _ := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "p"}, {core.EventTypeStepCompleted, "p"},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepFailed, "a"},
		{core.EventTypeStepFallbackActivated, "a"},
	}
	assertPairs(t, pairs, want) // fb has NOT dispatched yet on engine1

	// "Restart": engine 2 over the SAME storage. ha is deliberately NOT
	// registered on engine2 — if a were ever wrongly re-dispatched (B-4
	// regressing), dispatch would hit a different failure shape (handler not
	// found), which the StepStarted(a) count assertion below still catches.
	hfb := &stepHandler{id: "hfb", outputs: map[string]any{"r": "recovered"}}
	e2 := buildEngine2(s, 1, clk.now, hfb)

	inst := runToTerminal(t, e2, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed (pending fallback target must run after restart)", inst.Status)
	}

	pairs, _ = eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("terminally-failed step 'a' must NOT be re-dispatched after restart: got %d StepStarted, want 1 (B-4 across restart)", n)
	}
	if n := countPairs(pairs, core.EventTypeStepStarted, "fb"); n != 1 {
		t.Fatalf("pending fallback target 'fb' StepStarted count = %d, want 1 (dispatched by engine2)", n)
	}
	assertProjectionEquivalence(t, s, iid)
}

// TestHydrate_OnErrorPendingTargetRunsAfterRestart: same shape as the fallback
// fixture above, but through the on_error branch — this exercises hydrate.go's
// SEPARATE on_error reconstruction path (it recomputes firingOnError against
// dv/buildEnv at hydrate time, unlike the fallback path which just replays
// StepFallbackActivated targets verbatim).
func TestHydrate_OnErrorPendingTargetRunsAfterRestart(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.honerr", Version: "1.0.0", Namespace: "t", Name: "honerr",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("p", "hp"),
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"},
			nativeStep("b", "hb"),
		},
		Transitions: []core.Transition{
			{From: "p", To: "a"},
			{From: "a", To: "b", OnError: true},
		},
		InitialStep: "p", FinalSteps: []string{"b"}, Metadata: map[string]any{},
	}
	hp := &stepHandler{id: "hp", outputs: map[string]any{"rp": "p-done"}}
	ha := &stepHandler{id: "ha", err: errBoom}

	clk := newManualClock(engineStart)
	ctx := context.Background()
	e1, s := newNativeEngine(t, def, 1, clk.now, hp, ha)
	iid, err := e1.Submit(ctx, "t.honerr", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := e1.Tick(ctx); err != nil { // p activates + completes
		t.Fatalf("Tick 1 (engine1): %v", err)
	}
	if err := e1.Tick(ctx); err != nil { // a activates, fails terminally, b recorded pending
		t.Fatalf("Tick 2 (engine1): %v", err)
	}
	pairs, _ := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "p"}, {core.EventTypeStepCompleted, "p"},
		{core.EventTypeStepStarted, "a"}, {core.EventTypeStepFailed, "a"},
	}
	assertPairs(t, pairs, want) // b has NOT dispatched yet on engine1

	hb := &stepHandler{id: "hb", outputs: map[string]any{"r": "handled"}}
	e2 := buildEngine2(s, 1, clk.now, hb)

	inst := runToTerminal(t, e2, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed (pending on_error target must run after restart)", inst.Status)
	}

	pairs, _ = eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 1 {
		t.Fatalf("terminally-failed step 'a' must NOT be re-dispatched after restart: got %d StepStarted, want 1 (B-4 across restart)", n)
	}
	if n := countPairs(pairs, core.EventTypeStepStarted, "b"); n != 1 {
		t.Fatalf("pending on_error target 'b' StepStarted count = %d, want 1 (dispatched by engine2)", n)
	}
	assertProjectionEquivalence(t, s, iid)
}

// ── hydrate is idempotent (per-instance, per-process, once) ────────────────

// TestHydrate_IdempotentGuardPreventsReReconstruction calls hydrate() directly
// twice for the same instance on the same engine and proves the SECOND call
// takes the e.hydrated fast path rather than re-running reconstruction: after
// the first hydrate populates e.retries for the mid-backoff step, the test
// overwrites that entry with a sentinel value (standing in for "real
// scheduling work performed by the live engine since hydration") and asserts
// a second hydrate() call leaves the sentinel untouched.
func TestHydrate_IdempotentGuardPreventsReReconstruction(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.hidem", Version: "1.0.0", Namespace: "t", Name: "hidem",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{{
			ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha",
			Retry: &core.RetryPolicy{Attempts: 3, Backoff: "linear", InitialDelay: "1s", MaxDelay: "30s"},
		}},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	ha := &flakyHandler{id: "ha", outputs: map[string]any{"r": "ok"}, err: errBoom, failUntil: 1}

	clk := newManualClock(engineStart)
	ctx := context.Background()
	e1, s := newNativeEngine(t, def, 1, clk.now, ha)
	iid, err := e1.Submit(ctx, "t.hidem", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := e1.Tick(ctx); err != nil { // attempt 1 fails, retry scheduled
		t.Fatalf("Tick 1 (engine1): %v", err)
	}

	e2 := buildEngine2(s, 1, clk.now, ha)
	inst, found, err := e2.getInstance(ctx, iid)
	if err != nil || !found {
		t.Fatalf("getInstance: found=%v err=%v", found, err)
	}
	dv, err := e2.defViewFor(ctx, inst)
	if err != nil {
		t.Fatalf("defViewFor: %v", err)
	}

	if err := e2.hydrate(ctx, dv, inst); err != nil {
		t.Fatalf("hydrate (1st): %v", err)
	}
	key := retryKey{iid: iid, step: "a"}
	e2.mu.Lock()
	orig, ok := e2.retries[key]
	e2.mu.Unlock()
	if !ok {
		t.Fatalf("expected hydrate to reconstruct a retry schedule for 'a'")
	}

	// Overwrite with a sentinel: if the second hydrate() call re-reconstructs
	// instead of taking the e.hydrated fast path, it will clobber this back to
	// (something close to) orig.
	sentinel := retrySched{nextAttempt: 999, nextAttemptAt: orig.nextAttemptAt.Add(time.Hour)}
	e2.mu.Lock()
	e2.retries[key] = sentinel
	e2.mu.Unlock()

	if err := e2.hydrate(ctx, dv, inst); err != nil {
		t.Fatalf("hydrate (2nd): %v", err)
	}
	e2.mu.Lock()
	got := e2.retries[key]
	e2.mu.Unlock()
	if got != sentinel {
		t.Fatalf("a second hydrate() call re-reconstructed instead of taking the already-hydrated fast path: got %+v, want sentinel %+v (idempotency guard broken)", got, sentinel)
	}
}

// ── the subtlest part of hydrate.go: "latest of StepStarted/StepCompleted/
// StepFailed wins" ──────────────────────────────────────────────────────────

// TestHydrate_RetryThenSuccessNotRescheduledAfterRestart: step a fails once
// (StepFailed{retrying:true}), retries, and SUCCEEDS — all on engine 1, before
// b (downstream of a) ever dispatches. hydrate must recognise that a's LATEST
// lifecycle event is StepCompleted (not the earlier StepFailed) and must
// reconstruct NEITHER a retry schedule NOR a terminal-failure entry for a. A
// hydrate that naively keyed off "does this step have a StepFailed event"
// (ignoring event order) would incorrectly populate e.retries["a"] here.
func TestHydrate_RetryThenSuccessNotRescheduledAfterRestart(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.hlatest", Version: "1.0.0", Namespace: "t", Name: "hlatest",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha",
				Retry: &core.RetryPolicy{Attempts: 2, Backoff: "linear", InitialDelay: "1s", MaxDelay: "30s"}},
			nativeStep("b", "hb"),
		},
		Transitions: []core.Transition{{From: "a", To: "b"}},
		InitialStep: "a", FinalSteps: []string{"b"}, Metadata: map[string]any{},
	}
	ha := &flakyHandler{id: "ha", outputs: map[string]any{"r": "ok"}, err: errBoom, failUntil: 1}
	// b is downstream of a and must SUCCEED once reached: this test's subject
	// is whether hydrate re-schedules a, so b failing for an unrelated reason
	// (handler_not_found) would mask the actual assertion behind a failed
	// workflow.
	hb := &stepHandler{id: "hb", outputs: map[string]any{"r": "handled"}}

	clk := newManualClock(engineStart)
	ctx := context.Background()
	e1, s := newNativeEngine(t, def, 1, clk.now, ha, hb)
	iid, err := e1.Submit(ctx, "t.hlatest", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := e1.Tick(ctx); err != nil { // attempt 1 fails, retry scheduled
		t.Fatalf("Tick 1 (engine1): %v", err)
	}
	clk.advance(1 * time.Second)
	if err := e1.Tick(ctx); err != nil { // attempt 2 dispatches and succeeds
		t.Fatalf("Tick 2 (engine1): %v", err)
	}
	pairs, _ := eventPairs(t, s, iid)
	want := []pair{
		{core.EventTypeWorkflowStarted, ""},
		{core.EventTypeStepStarted, "a"}, // attempt 1
		{core.EventTypeStepFailed, "a"},  // retrying:true
		{core.EventTypeStepStarted, "a"}, // attempt 2
		{core.EventTypeStepCompleted, "a"},
	}
	assertPairs(t, pairs, want) // b has NOT dispatched yet on engine1

	// "Restart": inspect hydrate's direct reconstruction on a fresh engine —
	// the internal-state check is the precise probe the card asks for (the
	// PENDING/dispatch-count assertions further down are the black-box
	// confirmation that nothing observably regresses either).
	e2 := buildEngine2(s, 1, clk.now, ha, hb)
	inst, found, err := e2.getInstance(ctx, iid)
	if err != nil || !found {
		t.Fatalf("getInstance: found=%v err=%v", found, err)
	}
	dv, err := e2.defViewFor(ctx, inst)
	if err != nil {
		t.Fatalf("defViewFor: %v", err)
	}
	if err := e2.hydrate(ctx, dv, inst); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	key := retryKey{iid: iid, step: "a"}
	e2.mu.Lock()
	_, stillScheduled := e2.retries[key]
	failedA := e2.failed[iid]["a"]
	e2.mu.Unlock()
	if stillScheduled {
		t.Fatalf("hydrate must NOT reconstruct a retry schedule for 'a': it already succeeded (StepCompleted is the LATEST event)")
	}
	if failedA {
		t.Fatalf("hydrate must NOT mark 'a' terminally failed: it already succeeded (StepCompleted is the LATEST event)")
	}

	// Behavioral confirmation: b (downstream of a) dispatches normally and the
	// workflow completes, with NO phantom third attempt at a.
	inst2 := runToTerminal(t, e2, s, iid)
	if inst2.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst2.Status)
	}
	pairs, _ = eventPairs(t, s, iid)
	if n := countPairs(pairs, core.EventTypeStepStarted, "a"); n != 2 {
		t.Fatalf("StepStarted(a) = %d, want exactly 2 (attempt 1 + attempt 2, no phantom re-schedule after restart)", n)
	}
	if n := countPairs(pairs, core.EventTypeStepStarted, "b"); n != 1 {
		t.Fatalf("StepStarted(b) = %d, want 1", n)
	}
	assertProjectionEquivalence(t, s, iid)
}
