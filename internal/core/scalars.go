// Package core holds the canonical AWIS type layer (Amendment F-1). It is a
// leaf package: it imports only the Go standard library and contains ZERO
// executable logic — no method bodies, no constructors, no validation. The
// public `sdk` package re-exports every identifier here via type aliases, so
// the serialized field names frozen at Gate G1 (TDS-01, TDS-02) live in exactly
// one place. Applications depend on `sdk`; they never import this package.
package core

// SemVer is a semantic version string, e.g. "1.0.0" (Blueprint §6, line 262).
// A workflow's version is immutable once registered.
type SemVer string

// HandlerRef identifies a step's handler: a native "handler-name" or a
// subprocess "script.py", etc. (Blueprint §6, line 284).
type HandlerRef string

// Condition is an expression over step outputs; an absent condition means the
// transition is unconditional (Blueprint §6, line 300). The grammar is defined
// by TDS-03 (EXPRESSION_GRAMMARS); this layer carries the source string only.
type Condition string

// Duration is a workflow- or step-level duration.
//
// Shape completed at M10 (owning milestone); not part of the G1 format freeze
// (IMP §13 — sdk surface mutable until M08). The frozen corpus does not show a
// serialization form; the DSL (M10) and SDK (M08) decide the parse form.
type Duration string

// InputSchema is the JSON Schema describing a step's expected inputs
// (Blueprint §6, line 285).
type InputSchema = map[string]any

// OutputSchema is the JSON Schema describing a step's produced outputs
// (Blueprint §6, line 286).
type OutputSchema = map[string]any

// InstanceID is the unique identity of a workflow instance (UUID form;
// Blueprint §6 line 316, §12 WorkflowRunner.Submit return).
type InstanceID string

// IdempotencyKey keys the StepResultCache: hash(instance_id + step_id + attempt)
// (Blueprint §20, step_results_cache DDL).
type IdempotencyKey string
