// Package expr implements AWIS's two hand-written expression languages: the
// template grammar ("{{ ... }}", resolves to a value) and the condition grammar
// (boolean transition/trigger predicates). Both grammars, their prohibited
// construct lists, and the null-handling rules are transcribed verbatim from the
// frozen normative text in docs/EXPRESSION_GRAMMARS.md (TDS-03, Finalization
// Blocker 2). No external expression library is used: Finalization B2 explicitly
// rejected expr-lang / CEL / JSONata (Option D), so these parsers are stdlib-only
// and hand-written, and the grammar-conformance corpus in
// internal/expr/corpus/corpus.go is the oracle both parsers must satisfy.
//
// Execution-time semantics (template resolution and condition evaluation) follow
// EDR-010 (strict, no coercion) and the frozen FR-WD-05 template null rule:
//   - Template resolution never fails: a missing path or a not-yet-completed step
//     contributes the empty string and appends a Warning (see Template.Resolve).
//   - Condition evaluation applies EDR-010 strict comparison: numbers compare
//     numerically as float64, strings/bools compare only with ==/!=, ordering ops
//     on non-numeric or type-mismatched operands evaluate to false, and a path
//     resolving to null/missing follows the frozen null rules (see
//     ConditionExpr.Eval).
//
// Scope walls (M05): this package performs NO YAML parsing (M10), NO engine
// integration (M06), and owns NO logger; Warnings are returned as data for the
// engine to log at M06.
package expr
