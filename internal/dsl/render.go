package dsl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/validate"
)

// Render produces the PRD §18 human-readable validation report (T5).
//
// Valid definition:
//
//	✓ <id> v<version> is valid
//	  Steps: N (type1: N1, ...)
//	  Intelligence: <capability> capability (required: <bool>, fallback: <id|none>)
//	  Plugins: <handler>
//	  Triggers: <type>, ...
//
// Invalid definition or parse failure:
//
//	✗ Validation failed: <file>
//	  Line N: <message>
//	  Suggestion: ...
//	  Example:
//	    <yaml>
func (r *Report) Render() string {
	if r.Valid() {
		return r.renderValid()
	}
	return r.renderInvalid()
}

func (r *Report) renderValid() string {
	var sb strings.Builder
	def := r.Def

	fmt.Fprintf(&sb, "✓ %s v%s is valid\n", def.ID, def.Version)

	// Steps: N (type1: N1, type2: N2, ...) — types sorted for determinism.
	typeCounts := make(map[string]int)
	for _, s := range def.Steps {
		typeCounts[string(s.Type)]++
	}
	typeNames := make([]string, 0, len(typeCounts))
	for t := range typeCounts {
		typeNames = append(typeNames, t)
	}
	sort.Strings(typeNames)
	parts := make([]string, 0, len(typeNames))
	for _, t := range typeNames {
		parts = append(parts, fmt.Sprintf("%s: %d", t, typeCounts[t]))
	}
	fmt.Fprintf(&sb, "  Steps: %d (%s)\n", len(def.Steps), strings.Join(parts, ", "))

	// One Intelligence line per intelligence step.
	for _, s := range def.Steps {
		if s.Type == core.StepTypeIntelligence && s.Intelligence != nil {
			fallback := "none"
			if s.Fallback != "" {
				fallback = s.Fallback
			}
			fmt.Fprintf(&sb, "  Intelligence: %s capability (required: %v, fallback: %s)\n",
				s.Intelligence.Capability, s.Intelligence.Required, fallback)
		}
	}

	// Plugins: handlers of all plugin steps (in definition order).
	var plugins []string
	for _, s := range def.Steps {
		if s.Type == core.StepTypePlugin {
			plugins = append(plugins, string(s.Handler))
		}
	}
	if len(plugins) > 0 {
		fmt.Fprintf(&sb, "  Plugins: %s\n", strings.Join(plugins, ", "))
	}

	// Triggers: <type> or event (<event-name>) for event triggers.
	trigParts := make([]string, 0, len(def.Triggers))
	for _, t := range def.Triggers {
		if t.Type == core.TriggerTypeEvent {
			if ev, ok := t.Config["event"].(string); ok && ev != "" {
				trigParts = append(trigParts, fmt.Sprintf("event (%s)", ev))
				continue
			}
		}
		trigParts = append(trigParts, string(t.Type))
	}
	if len(trigParts) > 0 {
		fmt.Fprintf(&sb, "  Triggers: %s\n", strings.Join(trigParts, ", "))
	}

	return sb.String()
}

func (r *Report) renderInvalid() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "✗ Validation failed: %s\n", r.File)
	for _, ri := range r.Issues {
		fmt.Fprintf(&sb, "\n")
		if ri.Line > 0 {
			fmt.Fprintf(&sb, "  Line %d: %s\n", ri.Line, ri.Message)
		} else {
			fmt.Fprintf(&sb, "  %s\n", ri.Message)
		}
		sug, ex := r.hintFor(ri)
		if sug != "" {
			fmt.Fprintf(&sb, "\n  Suggestion: %s\n", sug)
		}
		if ex != "" {
			fmt.Fprintf(&sb, "  Example:\n")
			for _, line := range strings.Split(ex, "\n") {
				fmt.Fprintf(&sb, "    %s\n", line)
			}
		}
	}
	return sb.String()
}

// hintFor returns the Suggestion and Example strings for a ReportIssue
// following the PRD §18 what/where/what-now shape.
func (r *Report) hintFor(ri ReportIssue) (suggestion, example string) {
	switch ri.Code {
	case "parse-error":
		return "Fix the YAML syntax error and retry.", ""

	case validate.CodeFallbackUndefined:
		target := r.fallbackOf(ri.StepID)
		return fmt.Sprintf("Add a step with id %q, or remove the fallback reference.", target),
			fmt.Sprintf("- id: %s\n  name: <name>\n  type: signal\n  wait_signal:\n    signal_name: <signal_name>\n    timeout: 24h\n    timeout_action: fail", target)

	case validate.CodeOrphanedStep:
		return fmt.Sprintf("Remove step %q or add a transition leading to it from another step.", ri.StepID),
			fmt.Sprintf("- from: <some-step>\n  to: %s", ri.StepID)

	case validate.CodeCycle:
		return fmt.Sprintf("Remove the cycle involving step %q. Workflows must be acyclic; use a new instance for repeating work.", ri.StepID),
			""

	case validate.CodeDuplicateStepID:
		return fmt.Sprintf("Rename one of the steps with id %q to a unique id.", ri.StepID),
			fmt.Sprintf("- id: %s-2\n  name: ...\n  type: ...", ri.StepID)

	case validate.CodeInitialStepUndefined:
		return "Set initial_step to a step id that is defined in this workflow.",
			"initial_step: <defined-step-id>"

	case validate.CodeFinalStepUndefined:
		return "Update final_steps to reference step ids defined in this workflow.",
			"final_steps: [<defined-step-id>]"

	case validate.CodeTransitionFromUndefined:
		return "Update the transition's from field to reference a defined step id.",
			"- from: <defined-step-id>\n  to: <defined-step-id>"

	case validate.CodeTransitionToUndefined:
		return "Update the transition's to field to reference a defined step id.",
			"- from: <defined-step-id>\n  to: <defined-step-id>"

	case validate.CodeConditionSyntax:
		return "Fix the condition expression. See docs/EXPRESSION_GRAMMARS.md for the syntax.",
			"condition: \"steps.<step-id>.outputs.<field> == '<value>'\""

	case validate.CodeConditionEventScope:
		return "Use the event scope only in trigger filter fields, not in transition conditions.",
			"condition: \"steps.<step-id>.outputs.<field> == '<value>'\""

	case validate.CodeFilterSyntax:
		return "Fix the trigger filter expression. See docs/EXPRESSION_GRAMMARS.md for the syntax.",
			"config:\n  filter: \"event.branch == 'main'\""

	case validate.CodeHandlerMissing:
		return fmt.Sprintf("Add a non-empty handler field to step %q.", ri.StepID),
			"handler: <handler-name>"

	case validate.CodeSignalConfigMissing:
		return fmt.Sprintf("Add a wait_signal block to step %q.", ri.StepID),
			"wait_signal:\n  signal_name: <signal_name>\n  timeout: 24h\n  timeout_action: fail"

	case validate.CodeIntelligenceConfigMissing:
		return fmt.Sprintf("Add an intelligence block to step %q.", ri.StepID),
			"intelligence:\n  capability: <capability>\n  context_budget: 4096"

	case validate.CodeIntelligenceContextBudget:
		return fmt.Sprintf("Set context_budget to a positive integer in step %q.", ri.StepID),
			"intelligence:\n  context_budget: 4096"

	case validate.CodeIntelligenceModelHint:
		return fmt.Sprintf("Set model_hint to one of: fast, quality, local (or omit it) in step %q.", ri.StepID),
			"intelligence:\n  model_hint: fast"
	}
	return "", ""
}
