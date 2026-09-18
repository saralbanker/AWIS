package validate

import (
	"reflect"
	"testing"

	"github.com/awis/awis/internal/core"
)

// validDef builds a fully valid, multi-step workflow used as the passing baseline
// across tests: 4+ steps, fan-out transitions, a conditional transition using the
// steps scope, a trigger with an event-scoped filter, an intelligence step with a
// fallback, and a signal step. Validate(validDef()) must return nil.
func validDef() core.WorkflowDefinition {
	return core.WorkflowDefinition{
		ID: "oip.example",
		Triggers: []core.Trigger{
			{Type: core.TriggerTypeEvent, Config: map[string]any{
				"filter": "event.branch == 'main'",
			}},
			{Type: core.TriggerTypeManual, Config: nil},
		},
		Steps: []core.Step{
			{ID: "start", Type: core.StepTypeNative, Handler: "start-handler"},
			{ID: "draft", Type: core.StepTypeIntelligence, Fallback: "review",
				Intelligence: &core.IntelReq{Capability: "draft", ModelHint: "quality", ContextBudget: 4000}},
			{ID: "review", Type: core.StepTypeSignal,
				WaitSignal: &core.WaitConfig{SignalName: "approved", TimeoutAction: "fail"}},
			{ID: "finish", Type: core.StepTypeSubprocess, Handler: "finish.py"},
		},
		Transitions: []core.Transition{
			{From: "start", To: "draft"},
			{From: "start", To: "finish", Condition: "steps.start.status == 'completed'"},
			{From: "draft", To: "review"},
			{From: "review", To: "finish"},
		},
		InitialStep: "start",
		FinalSteps:  []string{"finish"},
	}
}

// codes extracts the multiset of Issue codes for set-membership assertions.
func codes(issues []Issue) map[string]int {
	m := make(map[string]int)
	for _, is := range issues {
		m[is.Code]++
	}
	return m
}

// findIssue returns the first Issue with the given code, or false.
func findIssue(issues []Issue, code string) (Issue, bool) {
	for _, is := range issues {
		if is.Code == code {
			return is, true
		}
	}
	return Issue{}, false
}

func hasCode(issues []Issue, code string) bool {
	_, ok := findIssue(issues, code)
	return ok
}

func TestValidateValidDefIsClean(t *testing.T) {
	if got := Validate(validDef()); got != nil {
		t.Fatalf("expected nil for a valid definition, got %d issues: %+v", len(got), got)
	}
}

func TestDuplicateStepID(t *testing.T) {
	def := validDef()
	def.Steps = append(def.Steps, core.Step{ID: "start", Type: core.StepTypeNative, Handler: "h"})
	issues := Validate(def)
	is, ok := findIssue(issues, CodeDuplicateStepID)
	if !ok {
		t.Fatalf("expected %s, got %+v", CodeDuplicateStepID, issues)
	}
	if is.StepID != "start" {
		t.Fatalf("expected StepID=start, got %q", is.StepID)
	}
	// Exactly one duplicate Issue for the one duplicated id.
	if n := codes(issues)[CodeDuplicateStepID]; n != 1 {
		t.Fatalf("expected exactly 1 duplicate issue, got %d", n)
	}
	// Passing case: no duplicates.
	if hasCode(Validate(validDef()), CodeDuplicateStepID) {
		t.Fatal("valid def must not report a duplicate")
	}
}

func TestInitialStepUndefined(t *testing.T) {
	def := validDef()
	def.InitialStep = "nope"
	is, ok := findIssue(Validate(def), CodeInitialStepUndefined)
	if !ok {
		t.Fatal("expected initial-step-undefined")
	}
	if is.StepID != "nope" || is.Field != "initial_step" {
		t.Fatalf("unexpected locator: %+v", is)
	}
	if hasCode(Validate(validDef()), CodeInitialStepUndefined) {
		t.Fatal("valid def must not report initial-step-undefined")
	}
}

func TestFinalStepUndefined(t *testing.T) {
	def := validDef()
	def.FinalSteps = []string{"finish", "ghost"}
	is, ok := findIssue(Validate(def), CodeFinalStepUndefined)
	if !ok {
		t.Fatal("expected final-step-undefined")
	}
	if is.StepID != "ghost" || is.Field != "final_steps[1]" {
		t.Fatalf("unexpected locator: %+v", is)
	}
	if hasCode(Validate(validDef()), CodeFinalStepUndefined) {
		t.Fatal("valid def must not report final-step-undefined")
	}
}

func TestTransitionFromToUndefined(t *testing.T) {
	def := validDef()
	def.Transitions = append(def.Transitions,
		core.Transition{From: "ghostFrom", To: "finish"},
		core.Transition{From: "start", To: "ghostTo"},
	)
	issues := Validate(def)
	fromIs, ok := findIssue(issues, CodeTransitionFromUndefined)
	if !ok || fromIs.StepID != "ghostFrom" {
		t.Fatalf("expected transition-from-undefined for ghostFrom, got %+v", issues)
	}
	toIs, ok := findIssue(issues, CodeTransitionToUndefined)
	if !ok || toIs.StepID != "ghostTo" {
		t.Fatalf("expected transition-to-undefined for ghostTo, got %+v", issues)
	}
	clean := Validate(validDef())
	if hasCode(clean, CodeTransitionFromUndefined) || hasCode(clean, CodeTransitionToUndefined) {
		t.Fatal("valid def must not report transition endpoint issues")
	}
}

func TestFallbackUndefined(t *testing.T) {
	def := validDef()
	def.Steps[1].Fallback = "missing"
	is, ok := findIssue(Validate(def), CodeFallbackUndefined)
	if !ok {
		t.Fatal("expected fallback-undefined")
	}
	if is.StepID != "draft" || is.Field != "fallback" {
		t.Fatalf("unexpected locator: %+v", is)
	}
	if hasCode(Validate(validDef()), CodeFallbackUndefined) {
		t.Fatal("valid def must not report fallback-undefined")
	}
}

func TestOrphanedStep(t *testing.T) {
	def := validDef()
	// Add a defined step with no inbound path.
	def.Steps = append(def.Steps, core.Step{ID: "island", Type: core.StepTypeNative, Handler: "h"})
	is, ok := findIssue(Validate(def), CodeOrphanedStep)
	if !ok {
		t.Fatal("expected orphaned-step")
	}
	if is.StepID != "island" {
		t.Fatalf("expected StepID=island, got %q", is.StepID)
	}
	if hasCode(Validate(validDef()), CodeOrphanedStep) {
		t.Fatal("valid def must not report orphaned-step")
	}
}

// TestOrphanReachableViaFallbackOnly: a step reachable ONLY through a fallback
// edge is not orphaned.
func TestOrphanReachableViaFallbackOnly(t *testing.T) {
	def := core.WorkflowDefinition{
		Steps: []core.Step{
			{ID: "a", Type: core.StepTypeNative, Handler: "h", Fallback: "b"},
			{ID: "b", Type: core.StepTypeNative, Handler: "h"},
		},
		Transitions: nil, // b is reachable only via a's fallback edge.
		InitialStep: "a",
		FinalSteps:  []string{"b"},
	}
	if hasCode(Validate(def), CodeOrphanedStep) {
		t.Fatalf("step reachable via fallback must not be orphaned: %+v", Validate(def))
	}
}

// TestOrphanSkippedWhenInitialUndefined: with an undefined initial_step, the
// orphan check is skipped (only the initial-step issue surfaces, no cascade).
func TestOrphanSkippedWhenInitialUndefined(t *testing.T) {
	def := validDef()
	def.InitialStep = "ghost"
	issues := Validate(def)
	if !hasCode(issues, CodeInitialStepUndefined) {
		t.Fatal("expected initial-step-undefined")
	}
	if hasCode(issues, CodeOrphanedStep) {
		t.Fatalf("orphan check must be skipped when initial_step undefined, got %+v", issues)
	}
}

func TestCycleThreeNode(t *testing.T) {
	def := core.WorkflowDefinition{
		Steps: []core.Step{
			{ID: "a", Type: core.StepTypeNative, Handler: "h"},
			{ID: "b", Type: core.StepTypeNative, Handler: "h"},
			{ID: "c", Type: core.StepTypeNative, Handler: "h"},
		},
		Transitions: []core.Transition{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "a"},
		},
		InitialStep: "a",
	}
	issues := Validate(def)
	want := map[string]bool{"a": true, "b": true, "c": true}
	got := map[string]bool{}
	for _, is := range issues {
		if is.Code == CodeCycle {
			got[is.StepID] = true
		}
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected cycle nodes a,b,c; got %v (issues %+v)", got, issues)
	}
}

func TestCycleSelfLoop(t *testing.T) {
	def := core.WorkflowDefinition{
		Steps:       []core.Step{{ID: "a", Type: core.StepTypeNative, Handler: "h"}},
		Transitions: []core.Transition{{From: "a", To: "a"}},
		InitialStep: "a",
	}
	is, ok := findIssue(Validate(def), CodeCycle)
	if !ok || is.StepID != "a" {
		t.Fatalf("expected self-loop cycle on a, got %+v", Validate(def))
	}
}

func TestNoCycleInValidDef(t *testing.T) {
	if hasCode(Validate(validDef()), CodeCycle) {
		t.Fatal("valid def is acyclic")
	}
}

// TestCycleSkippedOnUndefinedEndpoint: cascade guard — cycle check skipped when a
// transition endpoint is undefined.
func TestCycleSkippedOnUndefinedEndpoint(t *testing.T) {
	def := core.WorkflowDefinition{
		Steps: []core.Step{
			{ID: "a", Type: core.StepTypeNative, Handler: "h"},
			{ID: "b", Type: core.StepTypeNative, Handler: "h"},
		},
		Transitions: []core.Transition{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
			{From: "a", To: "ghost"}, // undefined endpoint dirties the graph
		},
		InitialStep: "a",
	}
	issues := Validate(def)
	if !hasCode(issues, CodeTransitionToUndefined) {
		t.Fatal("expected transition-to-undefined")
	}
	if hasCode(issues, CodeCycle) {
		t.Fatalf("cycle check must be skipped on a dirty graph, got %+v", issues)
	}
}

func TestConditionSyntax(t *testing.T) {
	def := validDef()
	def.Transitions[1].Condition = "steps.start.status ==" // missing RHS value
	is, ok := findIssue(Validate(def), CodeConditionSyntax)
	if !ok {
		t.Fatal("expected condition-syntax")
	}
	if is.Field != "transitions[1].condition" {
		t.Fatalf("unexpected field: %q", is.Field)
	}
	if is.Position < 0 || is.Position > len(string(def.Transitions[1].Condition)) {
		t.Fatalf("position %d out of bounds", is.Position)
	}
	if is.Message == "" {
		t.Fatal("expected a non-empty parser message")
	}
	if hasCode(Validate(validDef()), CodeConditionSyntax) {
		t.Fatal("valid def has parseable conditions")
	}
}

func TestConditionEventScope(t *testing.T) {
	def := validDef()
	def.Transitions[1].Condition = "event.branch == 'main'"
	is, ok := findIssue(Validate(def), CodeConditionEventScope)
	if !ok {
		t.Fatal("expected condition-event-scope")
	}
	if is.Field != "transitions[1].condition" {
		t.Fatalf("unexpected field: %q", is.Field)
	}
	if is.Position != 0 {
		t.Fatalf("event-scope is not a grammar code; Position must be 0, got %d", is.Position)
	}
}

// TestEventScopeInFilterAllowed: the same event-scoped expression is fine in a
// trigger filter.
func TestEventScopeInFilterAllowed(t *testing.T) {
	def := validDef() // trigger[0] filter is "event.branch == 'main'"
	issues := Validate(def)
	if hasCode(issues, CodeConditionEventScope) || hasCode(issues, CodeFilterSyntax) {
		t.Fatalf("event scope must be allowed in a trigger filter, got %+v", issues)
	}
}

// TestEventScopeTrickyNegatives: event mid-path and event inside a string literal
// are NOT event-scope violations.
func TestEventScopeTrickyNegatives(t *testing.T) {
	cases := []string{
		"steps.event.status == 'x'",         // event is a mid-path segment
		"workflow.inputs.mode == 'event.x'", // event.x is inside a string literal
	}
	for _, c := range cases {
		def := validDef()
		def.Transitions[1].Condition = core.Condition(c)
		issues := Validate(def)
		if hasCode(issues, CodeConditionEventScope) {
			t.Fatalf("%q must not be flagged as event scope, got %+v", c, issues)
		}
		if hasCode(issues, CodeConditionSyntax) {
			t.Fatalf("%q should parse cleanly, got %+v", c, issues)
		}
	}
}

func TestFilterSyntax(t *testing.T) {
	def := validDef()
	def.Triggers[0].Config["filter"] = "event.branch ==" // malformed
	is, ok := findIssue(Validate(def), CodeFilterSyntax)
	if !ok {
		t.Fatal("expected filter-syntax")
	}
	if is.Field != "triggers[0].config.filter" {
		t.Fatalf("unexpected field: %q", is.Field)
	}
	if is.Position < 0 {
		t.Fatalf("bad position %d", is.Position)
	}
}

func TestFilterNotAString(t *testing.T) {
	def := validDef()
	def.Triggers[0].Config["filter"] = 42
	is, ok := findIssue(Validate(def), CodeFilterSyntax)
	if !ok {
		t.Fatal("expected filter-syntax for non-string filter")
	}
	if is.Message != "filter must be a string" || is.Position != 0 {
		t.Fatalf("unexpected issue: %+v", is)
	}
}

func TestFilterAbsentIsFine(t *testing.T) {
	def := validDef()
	// Manual trigger has nil Config; event trigger has a valid filter. No issues.
	if hasCode(Validate(def), CodeFilterSyntax) {
		t.Fatal("no filter-syntax expected on the baseline")
	}
}

func TestSignalConfigMissing(t *testing.T) {
	def := validDef()
	def.Steps[2].WaitSignal = nil // "review" is the signal step
	is, ok := findIssue(Validate(def), CodeSignalConfigMissing)
	if !ok || is.StepID != "review" {
		t.Fatalf("expected signal-config-missing on review, got %+v", Validate(def))
	}
	if hasCode(Validate(validDef()), CodeSignalConfigMissing) {
		t.Fatal("valid signal step must not be flagged")
	}
}

func TestIntelligenceConfigMissing(t *testing.T) {
	def := validDef()
	def.Steps[1].Intelligence = nil // "draft" is the intelligence step
	is, ok := findIssue(Validate(def), CodeIntelligenceConfigMissing)
	if !ok || is.StepID != "draft" {
		t.Fatalf("expected intelligence-config-missing on draft, got %+v", Validate(def))
	}
	if hasCode(Validate(validDef()), CodeIntelligenceConfigMissing) {
		t.Fatal("valid intelligence step must not be flagged")
	}
}

func TestHandlerMissing(t *testing.T) {
	def := validDef()
	def.Steps[0].Handler = "" // "start" is native
	is, ok := findIssue(Validate(def), CodeHandlerMissing)
	if !ok || is.StepID != "start" || is.Field != "handler" {
		t.Fatalf("expected handler-missing on start, got %+v", Validate(def))
	}
	if hasCode(Validate(validDef()), CodeHandlerMissing) {
		t.Fatal("valid handlers must not be flagged")
	}
}

func TestIntelligenceContextBudget(t *testing.T) {
	def := validDef()
	def.Steps[1].Intelligence.ContextBudget = 0
	is, ok := findIssue(Validate(def), CodeIntelligenceContextBudget)
	if !ok || is.StepID != "draft" {
		t.Fatalf("expected intelligence-context-budget on draft, got %+v", Validate(def))
	}
	// Negative is also invalid.
	def.Steps[1].Intelligence.ContextBudget = -5
	if !hasCode(Validate(def), CodeIntelligenceContextBudget) {
		t.Fatal("negative budget must be flagged")
	}
	if hasCode(Validate(validDef()), CodeIntelligenceContextBudget) {
		t.Fatal("positive budget must not be flagged")
	}
}

func TestIntelligenceModelHint(t *testing.T) {
	def := validDef()
	def.Steps[1].Intelligence.ModelHint = "turbo" // invalid
	is, ok := findIssue(Validate(def), CodeIntelligenceModelHint)
	if !ok || is.StepID != "draft" {
		t.Fatalf("expected intelligence-model-hint on draft, got %+v", Validate(def))
	}
	// Each valid hint (incl. empty) passes.
	for _, h := range []string{"", "fast", "quality", "local"} {
		d := validDef()
		d.Steps[1].Intelligence.ModelHint = h
		if hasCode(Validate(d), CodeIntelligenceModelHint) {
			t.Fatalf("model_hint %q must be accepted", h)
		}
	}
}

func TestStepTypeUnknown(t *testing.T) {
	def := validDef()
	def.Steps[0].Type = core.StepType("bogus")
	is, ok := findIssue(Validate(def), CodeStepTypeUnknown)
	if !ok || is.StepID != "start" || is.Field != "type" {
		t.Fatalf("expected step-type-unknown on start, got %+v", Validate(def))
	}
	// Each legal type is accepted. The step is actually retyped per iteration —
	// validating an unmodified fixture here would assert nothing about ty.
	// (Retyping may raise OTHER codes, e.g. signal-config-missing; we assert
	// only that step-type-unknown is absent.)
	for _, ty := range []core.StepType{core.StepTypeNative, core.StepTypeSubprocess, core.StepTypePlugin, core.StepTypeIntelligence, core.StepTypeSignal} {
		d := validDef()
		d.Steps[0].Type = ty
		if hasCode(Validate(d), CodeStepTypeUnknown) {
			t.Fatalf("legal type %q must be accepted", ty)
		}
	}
}

func TestTriggerTypeUnknown(t *testing.T) {
	def := validDef()
	def.Triggers[1].Type = core.TriggerType("bogus")
	is, ok := findIssue(Validate(def), CodeTriggerTypeUnknown)
	if !ok || is.Field != "triggers[1].type" {
		t.Fatalf("expected trigger-type-unknown on triggers[1], got %+v", Validate(def))
	}
	for _, ty := range []core.TriggerType{core.TriggerTypeManual, core.TriggerTypeSchedule, core.TriggerTypeEvent, core.TriggerTypeWebhook} {
		d := validDef()
		d.Triggers[1].Type = ty
		if hasCode(Validate(d), CodeTriggerTypeUnknown) {
			t.Fatalf("trigger type %q must be accepted", ty)
		}
	}
}

// TestSignalTimeoutAction is the B-31 regression: wait_signal.timeout_action is
// documented as a frozen set (fail|compensate|continue) but had no validator
// rule, so a scaffolded `cancel` value passed validation and the engine's
// unknown-action backstop silently routed it as a failure.
func TestSignalTimeoutAction(t *testing.T) {
	def := validDef()
	def.Steps[2].WaitSignal.TimeoutAction = "cancel" // "review" is the signal step
	is, ok := findIssue(Validate(def), CodeSignalTimeoutAction)
	if !ok || is.StepID != "review" || is.Field != "wait_signal.timeout_action" {
		t.Fatalf("expected signal-timeout-action on review, got %+v", Validate(def))
	}
	// Empty is REJECTED (NOT NULL DDL column, no documented default) — unlike
	// model_hint/backoff.
	def = validDef()
	def.Steps[2].WaitSignal.TimeoutAction = ""
	if !hasCode(Validate(def), CodeSignalTimeoutAction) {
		t.Fatal("empty timeout_action must be rejected")
	}
	// Each legal value is accepted.
	for _, a := range []string{"fail", "compensate", "continue"} {
		d := validDef()
		d.Steps[2].WaitSignal.TimeoutAction = a
		if hasCode(Validate(d), CodeSignalTimeoutAction) {
			t.Fatalf("timeout_action %q must be accepted", a)
		}
	}
}

// TestRetryBackoff is the B-31 regression's sibling: retry.backoff is
// documented as a frozen set (immediate|linear|exponential) with no validator
// rule; internal/engine/retry.go's default branch used to silently claim the
// validator already rejected unknown values.
func TestRetryBackoff(t *testing.T) {
	def := validDef()
	def.Steps[0].Retry = &core.RetryPolicy{Attempts: 2, Backoff: "eventually"}
	is, ok := findIssue(Validate(def), CodeRetryBackoff)
	if !ok || is.StepID != "start" || is.Field != "retry.backoff" {
		t.Fatalf("expected retry-backoff on start, got %+v", Validate(def))
	}
	// Empty is ALLOWED (RetryPolicy with no backoff is legal).
	def = validDef()
	def.Steps[0].Retry = &core.RetryPolicy{Attempts: 2, Backoff: ""}
	if hasCode(Validate(def), CodeRetryBackoff) {
		t.Fatal("empty backoff must be accepted")
	}
	// Each legal value is accepted.
	for _, b := range []string{"immediate", "linear", "exponential"} {
		d := validDef()
		d.Steps[0].Retry = &core.RetryPolicy{Attempts: 2, Backoff: b}
		if hasCode(Validate(d), CodeRetryBackoff) {
			t.Fatalf("backoff %q must be accepted", b)
		}
	}
}

// TestCompensationRetryBackoff proves the same rule applies to
// CompensationStep.Retry.Backoff (same field, different location).
func TestCompensationRetryBackoff(t *testing.T) {
	def := validDef()
	def.Compensation = &core.CompensationPlan{Steps: []core.CompensationStep{
		{StepID: "draft", UndoHandler: "undo-draft", Retry: &core.RetryPolicy{Attempts: 2, Backoff: "bogus"}},
	}}
	is, ok := findIssue(Validate(def), CodeRetryBackoff)
	if !ok || is.StepID != "draft" || is.Field != "compensation.steps[0].retry.backoff" {
		t.Fatalf("expected retry-backoff on compensation step draft, got %+v", Validate(def))
	}
	// A legal value is accepted.
	def = validDef()
	def.Compensation = &core.CompensationPlan{Steps: []core.CompensationStep{
		{StepID: "draft", UndoHandler: "undo-draft", Retry: &core.RetryPolicy{Attempts: 2, Backoff: "immediate"}},
	}}
	if hasCode(Validate(def), CodeRetryBackoff) {
		t.Fatal("legal compensation backoff must be accepted")
	}
}

// TestIntelligenceCapability: unknown capabilities are rejected, but
// classify/embed — legal-but-not-dispatchable (FR-IL-10, CONTRA-3) — are
// schema-legal; rejecting dispatch is the runner's job, not the validator's.
func TestIntelligenceCapability(t *testing.T) {
	def := validDef()
	def.Steps[1].Intelligence.Capability = "summarize" // invalid
	is, ok := findIssue(Validate(def), CodeIntelligenceCapability)
	if !ok || is.StepID != "draft" || is.Field != "intelligence.capability" {
		t.Fatalf("expected intelligence-capability on draft, got %+v", Validate(def))
	}
	for _, c := range []string{"draft", "synthesize", "classify", "embed"} {
		d := validDef()
		d.Steps[1].Intelligence.Capability = c
		if hasCode(Validate(d), CodeIntelligenceCapability) {
			t.Fatalf("capability %q must be accepted (legal, even if not dispatchable)", c)
		}
	}
}

// TestMultiDefect: a definition packed with defects surfaces ALL expected Issue
// codes simultaneously (no first-fail short-circuit).
func TestMultiDefect(t *testing.T) {
	def := core.WorkflowDefinition{
		Triggers: []core.Trigger{
			{Type: core.TriggerTypeEvent, Config: map[string]any{"filter": "event.x =="}}, // filter-syntax
		},
		Steps: []core.Step{
			{ID: "dup", Type: core.StepTypeNative, Handler: ""},  // handler-missing
			{ID: "dup", Type: core.StepTypeNative, Handler: "h"}, // duplicate-step-id
			{ID: "sig", Type: core.StepTypeSignal},               // signal-config-missing
			{ID: "intel", Type: core.StepTypeIntelligence, // intelligence-config-missing
				Fallback: "ghost"}, // fallback-undefined
			{ID: "island", Type: core.StepTypeNative, Handler: "h"}, // orphaned-step
		},
		Transitions: []core.Transition{
			{From: "dup", To: "sig", Condition: "event.z == 'q'"}, // condition-event-scope
			{From: "sig", To: "intel", Condition: "steps.sig =="}, // condition-syntax
		},
		InitialStep: "dup",
		FinalSteps:  []string{"nope"}, // final-step-undefined
	}
	issues := Validate(def)
	want := []string{
		CodeDuplicateStepID,
		CodeFinalStepUndefined,
		CodeFallbackUndefined,
		CodeOrphanedStep,
		CodeConditionSyntax,
		CodeConditionEventScope,
		CodeFilterSyntax,
		CodeSignalConfigMissing,
		CodeIntelligenceConfigMissing,
		CodeHandlerMissing,
	}
	c := codes(issues)
	for _, w := range want {
		if c[w] == 0 {
			t.Errorf("expected code %s present, issues: %+v", w, issues)
		}
	}
}

// TestDeterminism: validating the same definition twice yields identical slices.
func TestDeterminism(t *testing.T) {
	def := validDef()
	// Introduce a couple of graph defects so there is graph output to order.
	def.Steps = append(def.Steps,
		core.Step{ID: "z-island", Type: core.StepTypeNative, Handler: "h"},
		core.Step{ID: "a-island", Type: core.StepTypeNative, Handler: "h"},
	)
	first := Validate(def)
	second := Validate(def)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("non-deterministic output:\nfirst=%+v\nsecond=%+v", first, second)
	}
	// Orphan findings must be sorted by step id (a-island before z-island).
	var order []string
	for _, is := range first {
		if is.Code == CodeOrphanedStep {
			order = append(order, is.StepID)
		}
	}
	if !reflect.DeepEqual(order, []string{"a-island", "z-island"}) {
		t.Fatalf("orphan issues must be sorted by step id, got %v", order)
	}
}
