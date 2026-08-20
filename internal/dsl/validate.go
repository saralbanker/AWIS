package dsl

import (
	"os"
	"strconv"
	"strings"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/validate"
)

// Report holds the result of ValidateFile: the parsed definition (nil on parse
// failure) plus any Issues, each annotated with the YAML source line.
type Report struct {
	// File is the path passed to ValidateFile.
	File string
	// Def is the parsed definition. Nil when parsing failed.
	Def *core.WorkflowDefinition
	// Issues is the ordered list of all problems found.
	// A parse failure produces a single "parse-error" Issue.
	Issues []ReportIssue
}

// Valid reports whether the Report contains no Issues.
func (r *Report) Valid() bool { return len(r.Issues) == 0 }

// ReportIssue is one problem annotated with its YAML source line (0 when
// unknown, e.g. for parse errors or when no lineMap entry exists).
type ReportIssue struct {
	Line    int    // YAML source line; 0 if unknown
	Code    string // validate.Code* constant, or "parse-error"
	StepID  string // mirrors validate.Issue.StepID
	Field   string // mirrors validate.Issue.Field
	Message string // human-readable description
}

// ValidateFile opens path, parses it as a workflow YAML, and validates the
// resulting definition. Parse failures yield a Report with a single
// "parse-error" Issue that is still renderable. I/O errors (e.g. file not
// found) are returned directly as Go errors.
// No runtime, storage, or handler-registry access is performed (FR-WD-15; T4).
func ValidateFile(path string) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	def, lm, parseErr := parseInternal(f, path)
	if parseErr != nil {
		return &Report{
			File: path,
			Issues: []ReportIssue{{
				Code:    "parse-error",
				Message: parseErr.Error(),
			}},
		}, nil
	}

	return &Report{
		File:   path,
		Def:    def,
		Issues: resolveLines(validate.Validate(*def), lm),
	}, nil
}

// resolveLines converts validate.Issues to ReportIssues with source-line
// numbers resolved via the lineMap captured during parsing (T3).
func resolveLines(issues []validate.Issue, lm *lineMap) []ReportIssue {
	out := make([]ReportIssue, len(issues))
	for i, iss := range issues {
		out[i] = ReportIssue{
			Line:    lineFor(iss, lm),
			Code:    iss.Code,
			StepID:  iss.StepID,
			Field:   iss.Field,
			Message: iss.Message,
		}
	}
	return out
}

// lineFor resolves the YAML source line for a validate.Issue by consulting
// the lineMap captured during parsing (PRD §18 file/line format; T3/T5).
func lineFor(iss validate.Issue, lm *lineMap) int {
	if lm == nil {
		return 0
	}
	switch iss.Code {
	case validate.CodeOrphanedStep, validate.CodeCycle,
		validate.CodeDuplicateStepID, validate.CodeFallbackUndefined,
		validate.CodeHandlerMissing, validate.CodeSignalConfigMissing,
		validate.CodeIntelligenceConfigMissing,
		validate.CodeIntelligenceContextBudget,
		validate.CodeIntelligenceModelHint:
		return lm.Steps[iss.StepID]
	case validate.CodeInitialStepUndefined:
		return lm.TopLevel["initial_step"]
	case validate.CodeFinalStepUndefined:
		return lm.TopLevel["final_steps"]
	case validate.CodeTransitionFromUndefined, validate.CodeTransitionToUndefined,
		validate.CodeConditionSyntax, validate.CodeConditionEventScope:
		if idx := indexFromField(iss.Field, "transitions"); idx >= 0 && idx < len(lm.Transitions) {
			return lm.Transitions[idx]
		}
	case validate.CodeFilterSyntax:
		if idx := indexFromField(iss.Field, "triggers"); idx >= 0 && idx < len(lm.Triggers) {
			return lm.Triggers[idx]
		}
	}
	return 0
}

// indexFromField parses the bracket index from a field locator such as
// "transitions[2].condition" (prefix="transitions") or
// "triggers[0].config.filter" (prefix="triggers").
func indexFromField(field, prefix string) int {
	expect := prefix + "["
	if !strings.HasPrefix(field, expect) {
		return -1
	}
	rest := field[len(expect):]
	end := strings.IndexByte(rest, ']')
	if end < 0 {
		return -1
	}
	idx, err := strconv.Atoi(rest[:end])
	if err != nil {
		return -1
	}
	return idx
}

// fallbackOf returns the fallback step id for the step with the given id.
func (r *Report) fallbackOf(stepID string) string {
	if r.Def == nil {
		return ""
	}
	for _, s := range r.Def.Steps {
		if s.ID == stepID {
			return s.Fallback
		}
	}
	return ""
}
