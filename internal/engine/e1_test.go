package engine

// AWIS-E1 gate (Constitution Art. 32 — PERMANENT). These end-to-end tests drive
// COMPLETE workflows through the real engine and real SQLite storage with
// intelligence ENTIRELY ABSENT: every engine here is constructed with a
// native-only runner map, NO dispatcher, and NO intelligence runner. This file
// imports no intelligence/dispatcher package. The gate proves AWIS functions as
// a system of record + coordination with zero AI, and that intelligence-TYPE
// steps degrade to their fallback path when intelligence is absent (Art. 32's
// core claim). `make e1` runs exactly `go test ./internal/engine/... -run TestE1`.
//
// Each fixture closes with assertProjectionEquivalence (EDR-007 forward ≡
// RebuildState) — the gate's evidence is both the terminal status/sequence and
// the projection-equivalence property.

import (
	"context"
	"testing"

	"github.com/awis/awis/internal/core"
)

// ── (1) linear native workflow completes ──────────────────────────────────────

func TestE1_LinearNativeCompletes(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "e1.linear", Version: "1.0.0", Namespace: "e1", Name: "linear",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("a", "ha"), nativeStep("b", "hb")},
		Transitions: []core.Transition{{From: "a", To: "b"}},
		InitialStep: "a", FinalSteps: []string{"b"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", outputs: map[string]any{"ra": "a"}}
	hb := &stepHandler{id: "hb", outputs: map[string]any{"rb": "b"}}
	e, s, _ := buildEngine(t, def, 1, ha, hb)

	iid, err := e.Submit(context.Background(), "e1.linear", "1.0.0", map[string]any{"in": "v"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}
	assertProjectionEquivalence(t, s, iid)
}

// ── (2) fan-out + join completes ──────────────────────────────────────────────

func TestE1_FanOutJoinCompletes(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "e1.diamond", Version: "1.0.0", Namespace: "e1", Name: "diamond",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			nativeStep("a", "ha"), nativeStep("b", "hb"),
			nativeStep("c", "hc"), nativeStep("d", "hd"),
		},
		Transitions: []core.Transition{
			{From: "a", To: "b"}, {From: "a", To: "c"},
			{From: "b", To: "d"}, {From: "c", To: "d"},
		},
		InitialStep: "a", FinalSteps: []string{"d"}, Metadata: map[string]any{},
	}
	e, s, _ := buildEngine(t, def, 2,
		&stepHandler{id: "ha", outputs: map[string]any{"r": "a"}},
		&stepHandler{id: "hb", outputs: map[string]any{"r": "b"}},
		&stepHandler{id: "hc", outputs: map[string]any{"r": "c"}},
		&stepHandler{id: "hd", outputs: map[string]any{"r": "d"}},
	)
	iid, err := e.Submit(context.Background(), "e1.diamond", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}
	assertProjectionEquivalence(t, s, iid)
}

// ── (3) retry then success ────────────────────────────────────────────────────

func TestE1_RetryThenSuccess(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "e1.retry", Version: "1.0.0", Namespace: "e1", Name: "retry",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{{
			ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha",
			Retry: &core.RetryPolicy{Attempts: 2, Backoff: "immediate"},
		}},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
	ha := &flakyHandler{id: "ha", outputs: map[string]any{"r": "ok"}, err: errBoom, failUntil: 1}
	e, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now, ha)

	iid, err := e.Submit(context.Background(), "e1.retry", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", inst.Status)
	}
	assertProjectionEquivalence(t, s, iid)
}

// ── (4) retry exhaust → fallback (native step with a fallback) ────────────────

func TestE1_RetryExhaustToFallback(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "e1.fb", Version: "1.0.0", Namespace: "e1", Name: "fb",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha", Fallback: "fb",
				Retry: &core.RetryPolicy{Attempts: 2, Backoff: "immediate"}},
			nativeStep("fb", "hfb"),
		},
		InitialStep: "a", FinalSteps: []string{"fb"}, Metadata: map[string]any{},
	}
	ha := &stepHandler{id: "ha", err: errBoom}
	hfb := &stepHandler{id: "hfb", outputs: map[string]any{"r": "fell-back"}}
	e, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now, ha, hfb)

	iid, err := e.Submit(context.Background(), "e1.fb", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed (via fallback)", inst.Status)
	}
	pairs, _ := eventPairs(t, s, iid)
	idx := indexer(pairs)
	if idx(core.EventTypeStepFallbackActivated, "a") == -1 {
		t.Fatalf("expected StepFallbackActivated for a")
	}
	if idx(core.EventTypeStepCompleted, "fb") == -1 {
		t.Fatalf("expected fallback step fb to complete")
	}
	assertProjectionEquivalence(t, s, iid)
}

// ── (5) failure → compensation ────────────────────────────────────────────────

func TestE1_FailureCompensation(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "e1.comp", Version: "1.0.0", Namespace: "e1", Name: "comp",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			{ID: "a", Name: "a", Type: core.StepTypeNative, Handler: "ha"},
			{ID: "d", Name: "d", Type: core.StepTypeNative, Handler: "hd"},
		},
		Transitions: []core.Transition{{From: "a", To: "d"}},
		Compensation: &core.CompensationPlan{Steps: []core.CompensationStep{
			{StepID: "a", UndoHandler: "undo-a"},
		}},
		InitialStep: "a", FinalSteps: []string{"d"}, Metadata: map[string]any{},
	}
	e, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now,
		&stepHandler{id: "ha", outputs: map[string]any{"r": "a"}},
		&stepHandler{id: "hd", err: errBoom},
		&stepHandler{id: "undo-a", outputs: map[string]any{}},
	)
	iid, err := e.Submit(context.Background(), "e1.comp", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompensated {
		t.Fatalf("status = %q, want compensated", inst.Status)
	}
	assertProjectionEquivalence(t, s, iid)
}

// ── (6) cancellation mid-flight (Finalization B4) ─────────────────────────────

func TestE1_CancellationMidFlight(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "e1.cancel", Version: "1.0.0", Namespace: "e1", Name: "cancel",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("step1", "h1"), nativeStep("step2", "h2")},
		Transitions: []core.Transition{{From: "step1", To: "step2"}},
		InitialStep: "step1", FinalSteps: []string{"step2"}, Metadata: map[string]any{},
	}
	h1 := &cancelHandler{id: "h1", reason: "user requested", compensate: false, outputs: map[string]any{"r": "s1"}}
	h2 := &stepHandler{id: "h2", outputs: map[string]any{"r": "s2"}}
	e, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now, h1, h2)
	h1.eng = e

	iid, err := e.Submit(context.Background(), "e1.cancel", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCancelled {
		t.Fatalf("status = %q, want cancelled", inst.Status)
	}
	// step2 must never start under cancellation (B4 step 3).
	pairs, _ := eventPairs(t, s, iid)
	if indexer(pairs)(core.EventTypeStepStarted, "step2") != -1 {
		t.Fatalf("step2 must never start under cancellation")
	}
	assertProjectionEquivalence(t, s, iid)
}

// ── (7) trigger ingestion → fire → complete with inputs == payload ────────────

func TestE1_TriggerIngestionFires(t *testing.T) {
	def := eventTriggerDef("e1.trig", "e1", "thing.happened", "")
	ha := &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}}
	e, s := newTriggerEngine(t, []core.WorkflowDefinition{def}, ha)
	ctx := context.Background()

	payload := map[string]any{"who": "alice", "n": float64(1)}
	ev := core.DomainEvent{EventID: "e1-de", Namespace: "e1", EventType: "thing.happened", Source: "src", Payload: payload}
	if err := e.Ingest(ctx, ev); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	driveTicks(t, e, 5)

	insts := instancesIn(t, s, "e1")
	if len(insts) != 1 || insts[0].Status != core.InstanceStatusCompleted {
		t.Fatalf("trigger must fire and complete one instance; got %+v", insts)
	}
	_, evs := eventPairs(t, s, insts[0].InstanceID)
	ws := findEvent(t, evs, core.EventTypeWorkflowStarted, "")
	var wsp struct {
		Inputs map[string]any `json:"inputs"`
	}
	decodePayload(t, ws, &wsp)
	assertJSONEqual(t, wsp.Inputs, payload)
	assertProjectionEquivalence(t, s, insts[0].InstanceID)
}

// ── (8) intelligence step degrades to fallback when intelligence is absent ────

// TestE1_IntelligenceAbsentTakesFallback is Art. 32's core claim: an
// intelligence-TYPE step (required=false, fallback="manual") in a runner map
// with NO intelligence runner dispatches to runner_unavailable and takes its
// fallback path, so the workflow still completes with zero AI configured.
func TestE1_IntelligenceAbsentTakesFallback(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "e1.intel", Version: "1.0.0", Namespace: "e1", Name: "intel",
		Triggers: []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps: []core.Step{
			intelStep("gen", "draft", false, "manual"),
			nativeStep("manual", "hman"),
		},
		InitialStep: "gen", FinalSteps: []string{"manual"}, Metadata: map[string]any{},
	}
	hman := &stepHandler{id: "hman", outputs: map[string]any{"r": "manual-done"}}
	// native-only engine: the intelligence runner is DELIBERATELY not registered.
	e, s := newNativeEngine(t, def, 1, newManualClock(engineStart).now, hman)

	iid, err := e.Submit(context.Background(), "e1.intel", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	inst := runToTerminal(t, e, s, iid)
	if inst.Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed (intelligence-absent fallback)", inst.Status)
	}

	pairs, evs := eventPairs(t, s, iid)
	idx := indexer(pairs)
	if idx(core.EventTypeStepFallbackActivated, "gen") == -1 {
		t.Fatalf("intelligence step must activate its fallback when no runner is registered")
	}
	if idx(core.EventTypeStepCompleted, "manual") == -1 {
		t.Fatalf("fallback step manual must complete")
	}
	// The failure that routed to fallback is runner_unavailable (intelligence absent).
	sf := findEvent(t, evs, core.EventTypeStepFailed, "gen")
	var sfp struct {
		Error core.StepError `json:"error"`
	}
	decodePayload(t, sf, &sfp)
	if sfp.Error.Code != "runner_unavailable" {
		t.Fatalf("StepFailed.error.code = %q, want runner_unavailable", sfp.Error.Code)
	}
	assertProjectionEquivalence(t, s, iid)
}
