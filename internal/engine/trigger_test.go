package engine

// F-2 trigger fixtures (EDR-011 §4). DomainEvent ingestion + SCAN_TRIGGERABLE
// matching: event_type / namespace / filter matching, inputs=payload VERBATIM,
// consume-on-fire, multi-definition fan-out, TTL prune, and the bad-filter /
// validation guards. Storage-level TriggerStore tests live in
// internal/storage/triggers_test.go.

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// eventTriggerDef builds a single-native-step workflow started by an event
// trigger on eventType (+ optional filter) in namespace ns.
func eventTriggerDef(id, ns, eventType, filter string) core.WorkflowDefinition {
	cfg := map[string]any{"event": eventType}
	if filter != "" {
		cfg["filter"] = filter
	}
	return core.WorkflowDefinition{
		SchemaVersion: 1, ID: id, Version: "1.0.0", Namespace: ns, Name: id,
		Triggers:    []core.Trigger{{Type: core.TriggerTypeEvent, Config: cfg}},
		Steps:       []core.Step{nativeStep("a", "ha")},
		InitialStep: "a", FinalSteps: []string{"a"}, Metadata: map[string]any{},
	}
}

// newTriggerEngine registers every def, wires a native runner with handlers, and
// builds an engine on a fixed manual clock (durations are not asserted here).
func newTriggerEngine(t *testing.T, defs []core.WorkflowDefinition, handlers ...core.StepHandler) (*Engine, *storage.SQLiteStorage) {
	t.Helper()
	s := openStorage(t)
	for _, d := range defs {
		if err := s.RegisterWorkflow(context.Background(), d); err != nil {
			t.Fatalf("RegisterWorkflow %s: %v", d.ID, err)
		}
	}
	nr := native.New()
	for _, h := range handlers {
		nr.Register(h)
	}
	e := New(s, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: 4, Clock: newManualClock(engineStart).now}, discardLogger())
	return e, s
}

// driveTicks runs n ticks (no ticker, no sleeps).
func driveTicks(t *testing.T, e *Engine, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := e.Tick(context.Background()); err != nil {
			t.Fatalf("Tick: %v", err)
		}
	}
}

// instancesIn lists all instances in ns.
func instancesIn(t *testing.T, s *storage.SQLiteStorage, ns string) []core.WorkflowInstance {
	t.Helper()
	insts, err := s.ListInstances(context.Background(), core.InstanceFilter{Namespace: ns})
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	return insts
}

// unconsumedIn returns the unconsumed domain events in ns.
func unconsumedIn(t *testing.T, s *storage.SQLiteStorage, ns string) []core.DomainEvent {
	t.Helper()
	evs, err := s.ListUnconsumedDomainEvents(context.Background(), ns)
	if err != nil {
		t.Fatalf("ListUnconsumedDomainEvents: %v", err)
	}
	return evs
}

// ── filter-less match: fires, inputs = payload verbatim, event consumed ────────

func TestTrigger_FiresInputsVerbatimAndConsumes(t *testing.T) {
	def := eventTriggerDef("t.trig", "t", "thing.happened", "")
	ha := &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}}
	e, s := newTriggerEngine(t, []core.WorkflowDefinition{def}, ha)
	ctx := context.Background()

	payload := map[string]any{"a": "x", "n": float64(7)}
	ev := core.DomainEvent{EventID: "de-1", Namespace: "t", EventType: "thing.happened", Source: "src", Payload: payload}
	if err := e.Ingest(ctx, ev); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	driveTicks(t, e, 5)
	insts := instancesIn(t, s, "t")
	if len(insts) != 1 {
		t.Fatalf("want 1 fired instance, got %d", len(insts))
	}
	if insts[0].Status != core.InstanceStatusCompleted {
		t.Fatalf("status = %q, want completed", insts[0].Status)
	}

	// inputs == payload VERBATIM (assert against the WorkflowStarted payload).
	_, evs := eventPairs(t, s, insts[0].InstanceID)
	ws := findEvent(t, evs, core.EventTypeWorkflowStarted, "")
	var wsp struct {
		Inputs map[string]any `json:"inputs"`
	}
	decodePayload(t, ws, &wsp)
	if !reflect.DeepEqual(wsp.Inputs, payload) {
		t.Fatalf("WorkflowStarted.inputs = %#v, want payload verbatim %#v", wsp.Inputs, payload)
	}

	// Event consumed exactly once (no longer unconsumed).
	if u := unconsumedIn(t, s, "t"); len(u) != 0 {
		t.Fatalf("event must be consumed after firing; unconsumed=%d", len(u))
	}
	assertProjectionEquivalence(t, s, insts[0].InstanceID)
}

// ── wrong event_type: unmatched, survives unconsumed ──────────────────────────

func TestTrigger_WrongEventTypeSurvives(t *testing.T) {
	def := eventTriggerDef("t.trig", "t", "thing.happened", "")
	ha := &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}}
	e, s := newTriggerEngine(t, []core.WorkflowDefinition{def}, ha)
	ctx := context.Background()

	ev := core.DomainEvent{EventID: "de-x", Namespace: "t", EventType: "other.type", Source: "src", Payload: map[string]any{}}
	if err := e.Ingest(ctx, ev); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	driveTicks(t, e, 3)

	if n := len(instancesIn(t, s, "t")); n != 0 {
		t.Fatalf("wrong event_type must not fire; instances=%d", n)
	}
	if u := unconsumedIn(t, s, "t"); len(u) != 1 {
		t.Fatalf("unmatched event must survive unconsumed; unconsumed=%d", len(u))
	}
}

// ── namespace mismatch: no fire ───────────────────────────────────────────────

func TestTrigger_NamespaceMismatchNoFire(t *testing.T) {
	def := eventTriggerDef("t.trig", "t", "thing.happened", "")
	ha := &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}}
	e, s := newTriggerEngine(t, []core.WorkflowDefinition{def}, ha)
	ctx := context.Background()

	// Event in a DIFFERENT namespace than the definition.
	ev := core.DomainEvent{EventID: "de-ns", Namespace: "other", EventType: "thing.happened", Source: "src", Payload: map[string]any{}}
	if err := e.Ingest(ctx, ev); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	driveTicks(t, e, 3)

	if n := len(instancesIn(t, s, "t")); n != 0 {
		t.Fatalf("namespace mismatch must not fire in t; instances=%d", n)
	}
	if n := len(instancesIn(t, s, "other")); n != 0 {
		t.Fatalf("no definition in namespace other; instances=%d", n)
	}
	if u := unconsumedIn(t, s, "other"); len(u) != 1 {
		t.Fatalf("namespace-mismatched event must survive unconsumed; unconsumed=%d", len(u))
	}
}

// ── filter false: no fire, survives; filter true: fires ───────────────────────

func TestTrigger_FilterFalseNoFire(t *testing.T) {
	def := eventTriggerDef("t.trig", "t", "thing.happened", `event.priority == 'high'`)
	ha := &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}}
	e, s := newTriggerEngine(t, []core.WorkflowDefinition{def}, ha)
	ctx := context.Background()

	ev := core.DomainEvent{EventID: "de-lo", Namespace: "t", EventType: "thing.happened", Source: "src",
		Payload: map[string]any{"priority": "low"}}
	if err := e.Ingest(ctx, ev); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	driveTicks(t, e, 3)

	if n := len(instancesIn(t, s, "t")); n != 0 {
		t.Fatalf("filter false must not fire; instances=%d", n)
	}
	if u := unconsumedIn(t, s, "t"); len(u) != 1 {
		t.Fatalf("filter-false event must survive unconsumed; unconsumed=%d", len(u))
	}
}

func TestTrigger_FilterTrueFires(t *testing.T) {
	def := eventTriggerDef("t.trig", "t", "thing.happened", `event.priority == 'high'`)
	ha := &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}}
	e, s := newTriggerEngine(t, []core.WorkflowDefinition{def}, ha)
	ctx := context.Background()

	payload := map[string]any{"priority": "high", "id": "42"}
	ev := core.DomainEvent{EventID: "de-hi", Namespace: "t", EventType: "thing.happened", Source: "src", Payload: payload}
	if err := e.Ingest(ctx, ev); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	driveTicks(t, e, 5)

	insts := instancesIn(t, s, "t")
	if len(insts) != 1 || insts[0].Status != core.InstanceStatusCompleted {
		t.Fatalf("filter true must fire and complete; instances=%+v", insts)
	}
	_, evs := eventPairs(t, s, insts[0].InstanceID)
	ws := findEvent(t, evs, core.EventTypeWorkflowStarted, "")
	var wsp struct {
		Inputs map[string]any `json:"inputs"`
	}
	decodePayload(t, ws, &wsp)
	if !reflect.DeepEqual(wsp.Inputs, payload) {
		t.Fatalf("inputs = %#v, want payload verbatim %#v", wsp.Inputs, payload)
	}
	if u := unconsumedIn(t, s, "t"); len(u) != 0 {
		t.Fatalf("fired event must be consumed; unconsumed=%d", len(u))
	}
}

// ── multi-definition match: ALL fire, event consumed once ─────────────────────

func TestTrigger_MultiDefAllFire(t *testing.T) {
	d1 := eventTriggerDef("t.d1", "t", "thing.happened", "")
	d2 := eventTriggerDef("t.d2", "t", "thing.happened", "")
	ha := &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}}
	e, s := newTriggerEngine(t, []core.WorkflowDefinition{d1, d2}, ha)
	ctx := context.Background()

	ev := core.DomainEvent{EventID: "de-multi", Namespace: "t", EventType: "thing.happened", Source: "src", Payload: map[string]any{"k": "v"}}
	if err := e.Ingest(ctx, ev); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	driveTicks(t, e, 5)

	insts := instancesIn(t, s, "t")
	if len(insts) != 2 {
		t.Fatalf("both matching definitions must fire; instances=%d", len(insts))
	}
	fired := map[string]bool{}
	for _, in := range insts {
		if in.Status != core.InstanceStatusCompleted {
			t.Fatalf("instance %s status = %q, want completed", in.InstanceID, in.Status)
		}
		fired[in.DefinitionID] = true
	}
	if !fired["t.d1"] || !fired["t.d2"] {
		t.Fatalf("want one instance per definition; got %v", fired)
	}
	// Consumed exactly once despite two fires.
	if u := unconsumedIn(t, s, "t"); len(u) != 0 {
		t.Fatalf("multi-fire event must be consumed once; unconsumed=%d", len(u))
	}
}

// ── TTL prune: old event deleted, fresh kept ──────────────────────────────────

func TestTrigger_TTLPruneDeletesOldKeepsFresh(t *testing.T) {
	def := eventTriggerDef("t.trig", "t", "never.matches", "") // never fires: isolates prune
	ha := &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}}
	e, s := newTriggerEngine(t, []core.WorkflowDefinition{def}, ha)
	ctx := context.Background()

	// Old event (older than the 7-day TTL relative to the fixed clock).
	old := core.DomainEvent{EventID: "de-old", Namespace: "t", EventType: "irrelevant", Source: "src",
		Payload: map[string]any{}, EmittedAt: engineStart.Add(-domainEventTTL - time.Hour)}
	if err := e.Ingest(ctx, old); err != nil {
		t.Fatalf("Ingest old: %v", err)
	}
	// Fresh event (emitted at the clock — well within TTL).
	fresh := core.DomainEvent{EventID: "de-fresh", Namespace: "t", EventType: "irrelevant", Source: "src",
		Payload: map[string]any{}}
	if err := e.Ingest(ctx, fresh); err != nil {
		t.Fatalf("Ingest fresh: %v", err)
	}

	driveTicks(t, e, 1) // one tick prunes at scanTriggerable.

	u := unconsumedIn(t, s, "t")
	if len(u) != 1 || u[0].EventID != "de-fresh" {
		t.Fatalf("TTL prune must delete de-old and keep de-fresh; unconsumed=%+v", u)
	}
}

// ── Ingest validation ─────────────────────────────────────────────────────────

func TestIngest_ValidationErrors(t *testing.T) {
	e, _ := newTriggerEngine(t, nil)
	ctx := context.Background()
	cases := []core.DomainEvent{
		{EventID: "", Namespace: "t", EventType: "x"}, // missing event_id
		{EventID: "e", Namespace: "t", EventType: ""}, // missing event_type
		{EventID: "e", Namespace: "", EventType: "x"}, // missing namespace
	}
	for i, ev := range cases {
		if err := e.Ingest(ctx, ev); err == nil {
			t.Fatalf("case %d: Ingest must reject %+v", i, ev)
		}
	}
	// A fully-specified event is accepted.
	if err := e.Ingest(ctx, core.DomainEvent{EventID: "ok", Namespace: "t", EventType: "x", Payload: map[string]any{}}); err != nil {
		t.Fatalf("valid Ingest rejected: %v", err)
	}
}

// ── bad filter: non-match + warn logged, scan continues ───────────────────────

func TestTrigger_BadFilterWarnsAndContinues(t *testing.T) {
	// A definition whose filter fails to parse (arithmetic is rejected by the
	// condition grammar, C20) must NOT wedge the scan: it is a non-match, a warn
	// is logged, and other events continue to be scanned.
	def := eventTriggerDef("t.bad", "t", "thing.happened", `1 + 1`)
	var buf bytes.Buffer
	ha := &stepHandler{id: "ha", outputs: map[string]any{"r": "ok"}}
	e, s := newBufferEngine(t, def, &buf, ha)
	ctx := context.Background()

	ev := core.DomainEvent{EventID: "de-bad", Namespace: "t", EventType: "thing.happened", Source: "src", Payload: map[string]any{}}
	if err := e.Ingest(ctx, ev); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	driveTicks(t, e, 3)

	if n := len(instancesIn(t, s, "t")); n != 0 {
		t.Fatalf("bad filter must be a non-match; instances=%d", n)
	}
	if u := unconsumedIn(t, s, "t"); len(u) != 1 {
		t.Fatalf("non-matched event must survive; unconsumed=%d", len(u))
	}
	if !bytes.Contains(buf.Bytes(), []byte("trigger filter parse failed")) {
		t.Fatalf("expected a warn for the bad filter; log=%s", buf.String())
	}
}
