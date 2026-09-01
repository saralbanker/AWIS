// Package validate implements the AWIS WorkflowValidator (M05 task T4): the
// static, structural checks a WorkflowDefinition must pass before it can be
// registered or run. It is the data half of FR-WD-03 / FR-WD-04 and covers the
// full "Validation checks (required)" checklist in PRD §18. The condition and
// trigger-filter grammars it enforces are the two hand-written languages in
// internal/expr, whose normative text lives in docs/EXPRESSION_GRAMMARS.md
// (TDS-03, Finalization Blocker 2).
//
// Scope walls (M05, per docs/05-implementation/M05-expression-engine):
//   - Validate produces DATA (structured Issues). Rendering an Issue with a
//     file/line and a fix suggestion in the PRD §18 output format is M10 (YAML
//     layer) / M14 (CLI) work; this package ships no rendering.
//   - Handler references are checked for PRESENCE only. Runtime resolvability of
//     a handler ("resolvable at runtime", PRD §18; FR-WD-15) is deferred to
//     registration time (M08); this package cannot see the handler registry.
//   - The `event` scope is grammatically valid everywhere at the parser level
//     (internal/expr accepts corpus rows C1–C9 with no field context). The
//     TDS-03 restriction "event scope: only valid in Trigger filter fields"
//     (TDS-03-derived, table TDS-03) is a FIELD-CONTEXT rule enforced here: an
//     event-scoped path in a transition condition is an Issue; in a trigger
//     filter it is allowed.
//
// Graph-semantics decisions (EDR-010, flagged for G2 review): reachability for
// the orphan check follows BOTH transition edges (from→to) AND fallback edges
// (step→its fallback), so a step reachable only as another step's fallback is
// not orphaned. Cycle detection runs over TRANSITION edges ONLY — a fallback
// join cannot re-enter the forward path, so fallback edges count for
// reachability but never for cycles (EDR-010 note; M05 TRACEABILITY).
package validate

import (
	"errors"
	"fmt"
	"sort"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/expr"
)

// Issue is one structural defect found in a WorkflowDefinition. It is pure data:
// Code identifies the rule, StepID/Field locate the offending element (either
// may be empty when not meaningful), Message is a human-readable explanation,
// and Position is a byte offset into the offending expression source — but ONLY
// when Code is a grammar code (condition-syntax, filter-syntax); for every other
// code Position is 0. Presentation (file/line/suggestion, PRD §18 format) is
// M10/M14's responsibility.
type Issue struct {
	// Code is a stable, machine-readable rule identifier (one of the Code*
	// constants in this package).
	Code string
	// StepID is the id of the step the Issue concerns, or the source step of a
	// transition; empty when the Issue is not step-scoped (e.g. a trigger filter).
	StepID string
	// Field is a definition-relative locator, e.g. "initial_step",
	// "transitions[2].condition", "intelligence.context_budget"; may be empty for
	// graph findings that concern the step as a whole (orphaned-step, cycle).
	Field string
	// Message is a human-readable explanation of the defect.
	Message string
	// Position is a byte offset into the offending expression source; nonzero only
	// for grammar codes (condition-syntax, filter-syntax), else 0.
	Position int
}

// Issue codes. Each is a stable string contract consumed by M10/M14 renderers.
const (
	// CodeDuplicateStepID: two or more steps share the same id.
	CodeDuplicateStepID = "duplicate-step-id"
	// CodeInitialStepUndefined: initial_step names no defined step.
	CodeInitialStepUndefined = "initial-step-undefined"
	// CodeFinalStepUndefined: a final_steps entry names no defined step.
	CodeFinalStepUndefined = "final-step-undefined"
	// CodeTransitionFromUndefined: a transition.from names no defined step.
	CodeTransitionFromUndefined = "transition-from-undefined"
	// CodeTransitionToUndefined: a transition.to names no defined step.
	CodeTransitionToUndefined = "transition-to-undefined"
	// CodeFallbackUndefined: a step.fallback names no defined step.
	CodeFallbackUndefined = "fallback-undefined"
	// CodeOrphanedStep: a defined step is unreachable from initial_step over
	// transition + fallback edges.
	CodeOrphanedStep = "orphaned-step"
	// CodeCycle: a defined step participates in a transition-edge cycle.
	CodeCycle = "cycle"
	// CodeConditionSyntax: a transition.condition does not parse (grammar code;
	// Position set).
	CodeConditionSyntax = "condition-syntax"
	// CodeConditionEventScope: a transition.condition uses the event scope, which
	// is valid only in trigger filters.
	CodeConditionEventScope = "condition-event-scope"
	// CodeFilterSyntax: a trigger filter does not parse, or is not a string
	// (grammar code; Position set on a parse failure, 0 for a non-string).
	CodeFilterSyntax = "filter-syntax"
	// CodeIntelligenceContextBudget: an intelligence step's context_budget is not
	// a positive integer.
	CodeIntelligenceContextBudget = "intelligence-context-budget"
	// CodeIntelligenceModelHint: an intelligence step's model_hint is not one of
	// fast|quality|local|"".
	CodeIntelligenceModelHint = "intelligence-model-hint"
	// CodeHandlerMissing: a native/subprocess/plugin step has an empty handler.
	CodeHandlerMissing = "handler-missing"
	// CodeSignalConfigMissing: a type=signal step has no wait_signal config.
	CodeSignalConfigMissing = "signal-config-missing"
	// CodeIntelligenceConfigMissing: a type=intelligence step has no intelligence
	// config.
	CodeIntelligenceConfigMissing = "intelligence-config-missing"
	// CodeStepTypeUnknown: a step.type is not one of the frozen StepType values
	// (native|subprocess|plugin|intelligence|signal).
	CodeStepTypeUnknown = "step-type-unknown"
	// CodeTriggerTypeUnknown: a trigger.type is not one of the frozen TriggerType
	// values (manual|schedule|event|webhook).
	CodeTriggerTypeUnknown = "trigger-type-unknown"
	// CodeSignalTimeoutAction: a wait_signal.timeout_action is not one of
	// fail|compensate|continue. Unlike model_hint/backoff, empty is REJECTED: the
	// DDL column is NOT NULL and there is no documented default.
	CodeSignalTimeoutAction = "signal-timeout-action"
	// CodeRetryBackoff: a retry.backoff is not one of immediate|linear|exponential
	// or empty (empty means "unspecified", handled by the engine's default path).
	CodeRetryBackoff = "retry-backoff"
	// CodeIntelligenceCapability: an intelligence step's capability is not one of
	// draft|synthesize|classify|embed. classify/embed are legal-but-not-dispatchable
	// (FR-IL-10, CONTRA-3) — that is a runtime concern, not a schema error, so they
	// are accepted here.
	CodeIntelligenceCapability = "intelligence-capability"
)

// Validate runs every PRD §18 required check over def and returns all Issues it
// finds. It NEVER panics and NEVER stops at the first defect — it accumulates
// every Issue so a caller sees the complete picture in one pass. Output order is
// deterministic: checks run in the fixed order below; within a check, steps and
// transitions are visited in definition order, and graph findings (orphaned-step,
// cycle) are emitted sorted by step id. Validate returns nil for a valid
// definition.
func Validate(def core.WorkflowDefinition) []Issue {
	var issues []Issue

	// Set of defined step ids (an id present at least once is "defined").
	defined := make(map[string]bool, len(def.Steps))
	for _, s := range def.Steps {
		defined[s.ID] = true
	}

	// 1. Duplicate step ids — prerequisite for graph sanity. Emit exactly one
	// Issue per duplicated id, at the point of its second occurrence, so ordering
	// follows definition order.
	seen := make(map[string]int, len(def.Steps))
	for _, s := range def.Steps {
		seen[s.ID]++
		if seen[s.ID] == 2 {
			issues = append(issues, Issue{
				Code:    CodeDuplicateStepID,
				StepID:  s.ID,
				Field:   "id",
				Message: fmt.Sprintf("step id %q is defined more than once", s.ID),
			})
		}
	}

	// 2. initial_step defined; every final_steps entry defined.
	if !defined[def.InitialStep] {
		issues = append(issues, Issue{
			Code:    CodeInitialStepUndefined,
			StepID:  def.InitialStep,
			Field:   "initial_step",
			Message: fmt.Sprintf("initial_step %q is not a defined step", def.InitialStep),
		})
	}
	for i, fs := range def.FinalSteps {
		if !defined[fs] {
			issues = append(issues, Issue{
				Code:    CodeFinalStepUndefined,
				StepID:  fs,
				Field:   fmt.Sprintf("final_steps[%d]", i),
				Message: fmt.Sprintf("final_steps entry %q is not a defined step", fs),
			})
		}
	}

	// 3. Every transition.from / transition.to defined. Track whether the
	// transition graph is clean so the cycle check can guard against cascade noise.
	edgesClean := true
	for i, t := range def.Transitions {
		if !defined[t.From] {
			edgesClean = false
			issues = append(issues, Issue{
				Code:    CodeTransitionFromUndefined,
				StepID:  t.From,
				Field:   fmt.Sprintf("transitions[%d].from", i),
				Message: fmt.Sprintf("transition.from %q is not a defined step", t.From),
			})
		}
		if !defined[t.To] {
			edgesClean = false
			issues = append(issues, Issue{
				Code:    CodeTransitionToUndefined,
				StepID:  t.To,
				Field:   fmt.Sprintf("transitions[%d].to", i),
				Message: fmt.Sprintf("transition.to %q is not a defined step", t.To),
			})
		}
	}

	// 4. Every non-empty step.fallback defined in the same workflow.
	for _, s := range def.Steps {
		if s.Fallback != "" && !defined[s.Fallback] {
			issues = append(issues, Issue{
				Code:    CodeFallbackUndefined,
				StepID:  s.ID,
				Field:   "fallback",
				Message: fmt.Sprintf("fallback %q is not a defined step", s.Fallback),
			})
		}
	}

	// Sorted unique step ids for deterministic graph-finding output.
	sortedIDs := make([]string, 0, len(defined))
	for id := range defined {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Strings(sortedIDs)

	// 5. Orphans: reachability from initial_step over transition edges (from→to)
	// PLUS fallback edges (step→its fallback). Skipped when initial_step is
	// undefined, to avoid a cascade of orphan noise on an already-reported defect.
	if defined[def.InitialStep] {
		adj := make(map[string][]string)
		for _, t := range def.Transitions {
			adj[t.From] = append(adj[t.From], t.To)
		}
		for _, s := range def.Steps {
			if s.Fallback != "" {
				adj[s.ID] = append(adj[s.ID], s.Fallback)
			}
		}
		reachable := make(map[string]bool)
		stack := []string{def.InitialStep}
		reachable[def.InitialStep] = true
		for len(stack) > 0 {
			u := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, v := range adj[u] {
				if !reachable[v] {
					reachable[v] = true
					stack = append(stack, v)
				}
			}
		}
		for _, id := range sortedIDs {
			if !reachable[id] {
				issues = append(issues, Issue{
					Code:    CodeOrphanedStep,
					StepID:  id,
					Message: fmt.Sprintf("step %q is not reachable from initial_step", id),
				})
			}
		}
	}

	// 6. Cycles over transition edges ONLY (fallback edges excluded, EDR-010).
	// Skipped when any transition.from/.to was undefined (cascade guard). Each
	// step id participating in a detected cycle is reported once, sorted.
	if edgesClean {
		for _, id := range detectCycleNodes(def.Transitions, sortedIDs) {
			issues = append(issues, Issue{
				Code:    CodeCycle,
				StepID:  id,
				Message: fmt.Sprintf("step %q participates in a transition cycle", id),
			})
		}
	}

	// 7. Transition conditions parse; on success, reject event scope.
	for i, t := range def.Transitions {
		cond := string(t.Condition)
		if cond == "" {
			continue
		}
		field := fmt.Sprintf("transitions[%d].condition", i)
		if _, err := expr.ParseCondition(cond); err != nil {
			pos, msg := 0, err.Error()
			var pe *expr.ParseError
			if errors.As(err, &pe) {
				pos, msg = pe.Position, pe.Msg
			}
			issues = append(issues, Issue{
				Code:     CodeConditionSyntax,
				StepID:   t.From,
				Field:    field,
				Message:  msg,
				Position: pos,
			})
			continue
		}
		if containsEventScope(cond) {
			issues = append(issues, Issue{
				Code:    CodeConditionEventScope,
				StepID:  t.From,
				Field:   field,
				Message: "event scope is only valid in trigger filter fields, not in a transition condition",
			})
		}
	}

	// 8. Trigger filters parse (event scope allowed here). A non-string filter is
	// itself an Issue.
	for j, tr := range def.Triggers {
		raw, ok := tr.Config["filter"]
		if !ok {
			continue
		}
		field := fmt.Sprintf("triggers[%d].config.filter", j)
		fs, ok := raw.(string)
		if !ok {
			issues = append(issues, Issue{
				Code:    CodeFilterSyntax,
				Field:   field,
				Message: "filter must be a string",
			})
			continue
		}
		if _, err := expr.ParseCondition(fs); err != nil {
			pos, msg := 0, err.Error()
			var pe *expr.ParseError
			if errors.As(err, &pe) {
				pos, msg = pe.Position, pe.Msg
			}
			issues = append(issues, Issue{
				Code:     CodeFilterSyntax,
				Field:    field,
				Message:  msg,
				Position: pos,
			})
		}
	}

	// 9. Per-step type config presence.
	for _, s := range def.Steps {
		switch s.Type {
		case core.StepTypeSignal:
			if s.WaitSignal == nil {
				issues = append(issues, Issue{
					Code:    CodeSignalConfigMissing,
					StepID:  s.ID,
					Field:   "wait_signal",
					Message: "type=signal step requires wait_signal config",
				})
			}
		case core.StepTypeIntelligence:
			if s.Intelligence == nil {
				issues = append(issues, Issue{
					Code:    CodeIntelligenceConfigMissing,
					StepID:  s.ID,
					Field:   "intelligence",
					Message: "type=intelligence step requires intelligence config",
				})
			}
		case core.StepTypeNative, core.StepTypeSubprocess, core.StepTypePlugin:
			if s.Handler == "" {
				issues = append(issues, Issue{
					Code:    CodeHandlerMissing,
					StepID:  s.ID,
					Field:   "handler",
					Message: fmt.Sprintf("type=%s step requires a non-empty handler", s.Type),
				})
			}
		default:
			issues = append(issues, Issue{
				Code:    CodeStepTypeUnknown,
				StepID:  s.ID,
				Field:   "type",
				Message: fmt.Sprintf("type must be one of native|subprocess|plugin|intelligence|signal, got %q", s.Type),
			})
		}
	}

	// 10. Intelligence field checks (when configured).
	for _, s := range def.Steps {
		if s.Intelligence == nil {
			continue
		}
		if s.Intelligence.ContextBudget <= 0 {
			issues = append(issues, Issue{
				Code:    CodeIntelligenceContextBudget,
				StepID:  s.ID,
				Field:   "intelligence.context_budget",
				Message: fmt.Sprintf("context_budget must be a positive integer, got %d", s.Intelligence.ContextBudget),
			})
		}
		if !validModelHint(s.Intelligence.ModelHint) {
			issues = append(issues, Issue{
				Code:    CodeIntelligenceModelHint,
				StepID:  s.ID,
				Field:   "intelligence.model_hint",
				Message: fmt.Sprintf("model_hint must be one of fast|quality|local or empty, got %q", s.Intelligence.ModelHint),
			})
		}
		if !validIntelligenceCapability(s.Intelligence.Capability) {
			issues = append(issues, Issue{
				Code:    CodeIntelligenceCapability,
				StepID:  s.ID,
				Field:   "intelligence.capability",
				Message: fmt.Sprintf("capability must be one of draft|synthesize|classify|embed, got %q", s.Intelligence.Capability),
			})
		}
	}

	// 11. Trigger types are one of the frozen TriggerType values.
	for j, tr := range def.Triggers {
		if !validTriggerType(tr.Type) {
			issues = append(issues, Issue{
				Code:    CodeTriggerTypeUnknown,
				Field:   fmt.Sprintf("triggers[%d].type", j),
				Message: fmt.Sprintf("trigger type must be one of manual|schedule|event|webhook, got %q", tr.Type),
			})
		}
	}

	// 12. wait_signal.timeout_action, when a signal wait is configured, is one of
	// fail|compensate|continue. Empty is REJECTED (NOT NULL DDL column, no
	// documented default).
	for _, s := range def.Steps {
		if s.WaitSignal == nil {
			continue
		}
		if !validTimeoutAction(s.WaitSignal.TimeoutAction) {
			issues = append(issues, Issue{
				Code:    CodeSignalTimeoutAction,
				StepID:  s.ID,
				Field:   "wait_signal.timeout_action",
				Message: fmt.Sprintf("timeout_action must be one of fail|compensate|continue, got %q", s.WaitSignal.TimeoutAction),
			})
		}
	}

	// 13. retry.backoff, when a retry policy is configured, is one of
	// immediate|linear|exponential or empty. Applies to both a step's own Retry
	// and a CompensationStep's Retry (same field, different location).
	for _, s := range def.Steps {
		if s.Retry == nil {
			continue
		}
		if !validBackoff(s.Retry.Backoff) {
			issues = append(issues, Issue{
				Code:    CodeRetryBackoff,
				StepID:  s.ID,
				Field:   "retry.backoff",
				Message: fmt.Sprintf("backoff must be one of immediate|linear|exponential or empty, got %q", s.Retry.Backoff),
			})
		}
	}
	if def.Compensation != nil {
		for i, cs := range def.Compensation.Steps {
			if cs.Retry == nil {
				continue
			}
			if !validBackoff(cs.Retry.Backoff) {
				issues = append(issues, Issue{
					Code:    CodeRetryBackoff,
					StepID:  cs.StepID,
					Field:   fmt.Sprintf("compensation.steps[%d].retry.backoff", i),
					Message: fmt.Sprintf("backoff must be one of immediate|linear|exponential or empty, got %q", cs.Retry.Backoff),
				})
			}
		}
	}

	if len(issues) == 0 {
		return nil
	}
	return issues
}

// validModelHint reports whether h is an accepted model hint (fast|quality|local
// or the empty string, meaning "unspecified").
func validModelHint(h string) bool {
	switch h {
	case "", "fast", "quality", "local":
		return true
	default:
		return false
	}
}

// validIntelligenceCapability reports whether c is one of the frozen
// intelligence capabilities (draft|synthesize|classify|embed). classify/embed
// are legal-but-not-dispatchable (FR-IL-10, CONTRA-3); rejecting dispatch is a
// runtime concern (internal/runner/intelligence), not a schema error.
func validIntelligenceCapability(c string) bool {
	switch c {
	case "draft", "synthesize", "classify", "embed":
		return true
	default:
		return false
	}
}

// validTriggerType reports whether t is one of the frozen TriggerType values
// (manual|schedule|event|webhook).
func validTriggerType(t core.TriggerType) bool {
	switch t {
	case core.TriggerTypeManual, core.TriggerTypeSchedule, core.TriggerTypeEvent, core.TriggerTypeWebhook:
		return true
	default:
		return false
	}
}

// validTimeoutAction reports whether a is one of the frozen timeout actions
// (fail|compensate|continue). Unlike validModelHint/validBackoff, the empty
// string is REJECTED: WaitConfig.TimeoutAction maps to a NOT NULL DDL column
// (Blueprint §9 L1411) and there is no documented default.
func validTimeoutAction(a string) bool {
	switch a {
	case "fail", "compensate", "continue":
		return true
	default:
		return false
	}
}

// validBackoff reports whether b is an accepted retry backoff strategy
// (immediate|linear|exponential or the empty string, meaning "unspecified" —
// RetryPolicy with no backoff is legal; the engine's parseDurationOr/default
// path handles it).
func validBackoff(b string) bool {
	switch b {
	case "", "immediate", "linear", "exponential":
		return true
	default:
		return false
	}
}

// detectCycleNodes returns, sorted, the ids of every step that participates in a
// cycle over the transition edges (from→to). It uses a three-color DFS: on a
// back edge to a gray (on-stack) node it marks every node between that node and
// the top of the current DFS path as in-cycle, which captures self-loops
// (A→A) and simple cycles (A→B→C→A) alike. Callers must pass edges known to be
// clean (all endpoints defined); ids is the sorted set of defined step ids and
// doubles as the deterministic DFS root order.
func detectCycleNodes(transitions []core.Transition, ids []string) []string {
	adj := make(map[string][]string)
	for _, t := range transitions {
		adj[t.From] = append(adj[t.From], t.To)
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	inCycle := make(map[string]bool)
	var path []string

	var visit func(u string)
	visit = func(u string) {
		color[u] = gray
		path = append(path, u)
		for _, v := range adj[u] {
			switch color[v] {
			case gray:
				// Back edge: everything from v up to the top of the path is a cycle.
				for k := len(path) - 1; k >= 0; k-- {
					inCycle[path[k]] = true
					if path[k] == v {
						break
					}
				}
			case white:
				visit(v)
			}
		}
		path = path[:len(path)-1]
		color[u] = black
	}

	for _, id := range ids {
		if color[id] == white {
			visit(id)
		}
	}

	out := make([]string, 0, len(inCycle))
	for _, id := range ids {
		if inCycle[id] {
			out = append(out, id)
		}
	}
	return out
}

// containsEventScope reports whether src (a GRAMMATICALLY VALID condition source
// — call only after expr.ParseCondition succeeds) uses the `event` scope, i.e.
// contains a path-ref that begins with the scope token `event`. It scans the raw
// source outside single-quoted string literals for the token "event" immediately
// followed by "." at a path-ref START boundary.
//
// A path-ref START boundary means the byte before "event" is neither an
// identifier byte ([a-zA-Z0-9_-]) NOR a dot: a leading '.' means "event" is a
// continuation segment of a longer path (e.g. steps.event.status), not a scope.
// This resolves the internal tension in the M05-C2 card, whose literal boundary
// wording ("previous byte not [a-zA-Z0-9_-]") would misclassify a mid-path
// `event`; the card's own required negative (steps.event.status ⇒ no issue)
// dictates excluding '.' as well. Because strings are single-quoted and may hold
// any bytes, an `event.x` inside 'event.x' is correctly ignored.
func containsEventScope(src string) bool {
	const tok = "event"
	inStr := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inStr {
			if c == '\'' {
				inStr = false
			}
			continue
		}
		if c == '\'' {
			inStr = true
			continue
		}
		if c != 'e' || i+len(tok) >= len(src) || src[i:i+len(tok)] != tok {
			continue
		}
		if src[i+len(tok)] != '.' {
			continue
		}
		if i > 0 {
			p := src[i-1]
			if isPathByte(p) {
				continue
			}
		}
		return true
	}
	return false
}

// isPathByte reports whether b may appear inside a path-ref: an identifier byte
// ([a-zA-Z0-9_-]) or the segment separator '.'.
func isPathByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '_' || b == '-' || b == '.':
		return true
	default:
		return false
	}
}
