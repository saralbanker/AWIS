// Package corpus holds the grammar-conformance fixture corpus for the AWIS
// expression languages.
//
// Grammar-conformance corpus, generated from docs/EXPRESSION_GRAMMARS.md
// (Finalization Blocker 2) before any parser exists (IR-2). M05's parsers must
// pass every row.
//
// This file is DATA ONLY: no functions, no methods, no parser logic.
package corpus

// Fixture is one grammar-conformance row. Src is the expression source string,
// Valid is its grammatical verdict (true = a parser must accept it), and Rule
// names the grammar production or prohibited construct it exercises.
type Fixture struct {
	Src   string
	Valid bool
	Rule  string
}

// Template mirrors docs/EXPRESSION_GRAMMARS.md Table 1 (rows T1..T15):
// 8 valid, 7 invalid, 15 total.
var Template = []Fixture{
	{Src: "", Valid: true, Rule: "literal-text (empty)"},
	{Src: "Hello, world", Valid: true, Rule: "literal-text"},
	{Src: "{{workflow.inputs.repo_path}}", Valid: true, Rule: "path-ref scope=workflow"},
	{Src: "{{steps.draft-entry.outputs.content}}", Valid: true, Rule: "path-ref steps.<id>.outputs.<key>"},
	{Src: "{{steps.draft-entry.status}}", Valid: true, Rule: "path-ref steps.<id>.status"},
	{Src: "{{workflow._meta.key}}", Valid: true, Rule: "identifier (underscore start)"},
	{Src: "prefix {{workflow.inputs.name}} suffix", Valid: true, Rule: "template-string concatenation (literal+ref)"},
	{Src: "{{workflow.inputs.a}}-{{steps.build.status}}", Valid: true, Rule: "template-string concatenation (multiple refs)"},
	{Src: "{{workflow.inputs.a + workflow.inputs.b}}", Valid: false, Rule: "PROHIBITED: arithmetic operators"},
	{Src: "{{toUpper(workflow.inputs.name)}}", Valid: false, Rule: "PROHIBITED: function calls"},
	{Src: "{{if workflow.inputs.flag}}", Valid: false, Rule: "PROHIBITED: conditionals in template"},
	{Src: "{{steps.{{x}}.outputs.y}}", Valid: false, Rule: "PROHIBITED: nested templates"},
	{Src: "{{steps['draft-entry'].status}}", Valid: false, Rule: "PROHIBITED: bracket notation"},
	{Src: "{{event.branch}}", Valid: false, Rule: "scope must be workflow|steps (event invalid)"},
	{Src: "{{workflow}}", Valid: false, Rule: "path-ref requires scope + identifier"},
}

// Condition mirrors docs/EXPRESSION_GRAMMARS.md Table 2 (rows C1..C27):
// 19 valid, 8 invalid, 27 total.
var Condition = []Fixture{
	{Src: "event.branch == 'main'", Valid: true, Rule: "compare-op ==, string-lit, scope=event"},
	{Src: "event.branch != 'main'", Valid: true, Rule: "compare-op !="},
	{Src: "event.count > 5", Valid: true, Rule: "compare-op >, number-lit (int)"},
	{Src: "event.count < 5", Valid: true, Rule: "compare-op <"},
	{Src: "event.count >= 5", Valid: true, Rule: "compare-op >="},
	{Src: "event.count <= 5", Valid: true, Rule: "compare-op <="},
	{Src: "event.score == 3.14", Valid: true, Rule: "number-lit (decimal)"},
	{Src: "event.enabled == true", Valid: true, Rule: "bool-lit true"},
	{Src: "event.enabled == false", Valid: true, Rule: "bool-lit false"},
	{Src: "steps.draft-entry.status == null", Valid: true, Rule: "compare-expr path-ref == null"},
	{Src: "steps.draft-entry.status != null", Valid: true, Rule: "compare-expr path-ref != null"},
	{Src: "workflow.inputs.mode == 'auto'", Valid: true, Rule: "scope=workflow"},
	{Src: "steps.draft-entry.status == 'completed'", Valid: true, Rule: "scope=steps, identifier hyphen"},
	{Src: "steps.draft-entry.outputs.content == 'x'", Valid: true, Rule: "path-ref multi-segment"},
	{Src: "event.branch == 'main' || event.branch == 'master'", Valid: true, Rule: "or-expr ||"},
	{Src: "steps.build.status == 'completed' && event.branch == 'main'", Valid: true, Rule: "and-expr &&"},
	{Src: "!(event.branch == 'main')", Valid: true, Rule: "not-expr !"},
	{Src: "(event.branch == 'main')", Valid: true, Rule: "not-expr ( condition )"},
	{Src: "(steps.a.status == 'completed' || event.branch == 'main') && steps.b.status == 'completed'", Valid: true, Rule: "nested parentheses / grouping"},
	{Src: "event.count + 1 == 5", Valid: false, Rule: "PROHIBITED: arithmetic operators"},
	{Src: "len(event.items) > 0", Valid: false, Rule: "PROHIBITED: function calls"},
	{Src: "event.branch == 'ma' + 'in'", Valid: false, Rule: "PROHIBITED: string concatenation"},
	{Src: "steps['draft-entry'].status == 'fallback'", Valid: false, Rule: "PROHIBITED: bracket notation"},
	{Src: "event.branch == 'main' ? true : false", Valid: false, Rule: "PROHIBITED: ternary operators"},
	{Src: "event.branch == 'main", Valid: false, Rule: "string-lit (unterminated)"},
	{Src: "steps.draft-entry.status > null", Valid: false, Rule: "null only valid with == / !="},
	{Src: "steps.draft-entry.status", Valid: false, Rule: "bare path-ref is not a condition"},
}
