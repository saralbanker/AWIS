# Expression Grammars

- **Doc ID:** TDS-03
- **Version:** 1.0.0
- **Date:** 2026-07-03
- **Source coordinate:** `AWIS_ARCHITECTURE_FINALIZATION.md` — Finalization Blocker 2 (Expression Language Specification)
- **Status:** **FROZEN — G1 APPROVED 2026-07-03** (founder sign-off)

AWIS uses two distinct, formally bounded expression grammars: **template expressions**
(resolve to values) and **condition expressions** (resolve to booleans). Both grammars,
their scope definitions, null-handling rules, and prohibited lists below are transcribed
verbatim from Finalization Blocker 2. The conformance fixture corpus (last section) is
grammar-derived and was written before any parser exists (IR-2 mitigation); its rows are
mirrored 1:1 in `internal/expr/corpus/corpus.go`. M05's parsers must pass every row.

---

## Grammar 1: Template Expressions `{{ ... }}`

Templates appear in YAML string field values and are resolved at step execution time to
produce concrete values passed as step inputs.

<!-- BEGIN verbatim from Finalization B2 -->
**Syntax:**
```
template-string ::= literal-text | "{{" path-ref "}}" | template-string template-string
path-ref        ::= scope "." identifier ("." identifier)*
scope           ::= "workflow" | "steps"
identifier      ::= [a-zA-Z_][a-zA-Z0-9_-]*
```

**Scope definitions:**
- `workflow.inputs.<key>` — the workflow's original inputs, provided at submission
- `steps.<step-id>.outputs.<key>` — a completed step's output field
- `steps.<step-id>.status` — a step's current status (string: pending | running | completed | failed | cancelled)

**Null handling:** If a referenced path does not exist or the referencing step has not yet completed, the template resolves to the empty string `""` and a warning is logged. Templates never fail at validation time; they fail gracefully at execution time.

**Prohibited:**
- Arithmetic operators (`+`, `-`, `*`, `/`)
- Function calls (`len(...)`, `toUpper(...)`, etc.)
- Conditionals within template expressions (`{{if ...}}`)
- Nested templates (`{{steps.{{x}}.outputs.y}}`)
- Bracket notation (`steps['step-id']` — use dot notation `steps.step-id` with hyphens allowed in identifiers)
<!-- END verbatim from Finalization B2 -->

---

## Grammar 2: Condition Expressions (Transition `condition` field)

Conditions appear in `Transition.condition` and `Trigger.config.filter` fields. They
evaluate to a boolean. The runtime evaluates them after a step completes to determine which
transition(s) to activate.

<!-- BEGIN verbatim from Finalization B2 -->
**Syntax:**
```
condition   ::= or-expr
or-expr     ::= and-expr | or-expr "||" and-expr
and-expr    ::= not-expr | and-expr "&&" not-expr
not-expr    ::= compare-expr | "!" not-expr | "(" condition ")"
compare-expr ::= path-ref compare-op value | path-ref "==" "null" | path-ref "!=" "null"
compare-op  ::= "==" | "!=" | ">" | "<" | ">=" | "<="
path-ref    ::= scope "." identifier ("." identifier)*
value       ::= string-lit | number-lit | bool-lit
string-lit  ::= "'" [^']* "'"
number-lit  ::= [0-9]+ ("." [0-9]+)?
bool-lit    ::= "true" | "false"
scope       ::= "workflow" | "steps" | "event"
```

**`event` scope:** Only valid in Trigger `filter` fields. Provides access to the domain event payload: `event.<key>` where `<key>` is a top-level field in the `DomainEvent.payload`.

**Null handling:** Comparing a path that resolves to null against a non-null literal with `>`, `<`, `>=`, `<=` evaluates to `false`. `null == null` evaluates to `true`.

**Prohibited in conditions:**
- Arithmetic operators
- Function calls
- String concatenation
- Bracket notation (`steps['step-id']` — use `steps.step-id`)
- Ternary operators
<!-- END verbatim from Finalization B2 -->

**Note on dot-path identifiers (verbatim from Finalization B2):** Identifiers may contain
hyphens (`step-id` is valid). This means `steps.draft-entry.status` is the canonical form.
Hyphens in identifiers are treated as single tokens, not subtraction operators.

---

## Conformance Fixture Corpus

Grammar-derived. `Verdict = valid` means the string is well-formed under the grammar above
(a parser must accept it); `Verdict = invalid` means it must be rejected. Verdicts reflect
**grammatical** validity only — template null-handling and condition null-comparison outcomes
are execution-time semantics, not parse verdicts. Every production rule has ≥1 valid fixture;
every prohibited construct has ≥1 invalid fixture.

### Table 1 — Template fixtures (`{{ ... }}`)

| # | Source string | Verdict | Grammar rule exercised |
|---|---|---|---|
| T1 | `` (empty) | valid | literal-text (empty) |
| T2 | `Hello, world` | valid | literal-text |
| T3 | `{{workflow.inputs.repo_path}}` | valid | path-ref scope=workflow |
| T4 | `{{steps.draft-entry.outputs.content}}` | valid | path-ref steps.<id>.outputs.<key> |
| T5 | `{{steps.draft-entry.status}}` | valid | path-ref steps.<id>.status |
| T6 | `{{workflow._meta.key}}` | valid | identifier (underscore start) |
| T7 | `prefix {{workflow.inputs.name}} suffix` | valid | template-string concatenation (literal+ref) |
| T8 | `{{workflow.inputs.a}}-{{steps.build.status}}` | valid | template-string concatenation (multiple refs) |
| T9 | `{{workflow.inputs.a + workflow.inputs.b}}` | invalid | PROHIBITED: arithmetic operators |
| T10 | `{{toUpper(workflow.inputs.name)}}` | invalid | PROHIBITED: function calls |
| T11 | `{{if workflow.inputs.flag}}` | invalid | PROHIBITED: conditionals in template |
| T12 | `{{steps.{{x}}.outputs.y}}` | invalid | PROHIBITED: nested templates |
| T13 | `{{steps['draft-entry'].status}}` | invalid | PROHIBITED: bracket notation |
| T14 | `{{event.branch}}` | invalid | scope must be workflow\|steps (event invalid) |
| T15 | `{{workflow}}` | invalid | path-ref requires scope + identifier |

### Table 2 — Condition fixtures

| # | Source string | Verdict | Grammar rule exercised |
|---|---|---|---|
| C1 | `event.branch == 'main'` | valid | compare-op ==, string-lit, scope=event |
| C2 | `event.branch != 'main'` | valid | compare-op != |
| C3 | `event.count > 5` | valid | compare-op >, number-lit (int) |
| C4 | `event.count < 5` | valid | compare-op < |
| C5 | `event.count >= 5` | valid | compare-op >= |
| C6 | `event.count <= 5` | valid | compare-op <= |
| C7 | `event.score == 3.14` | valid | number-lit (decimal) |
| C8 | `event.enabled == true` | valid | bool-lit true |
| C9 | `event.enabled == false` | valid | bool-lit false |
| C10 | `steps.draft-entry.status == null` | valid | compare-expr path-ref == null |
| C11 | `steps.draft-entry.status != null` | valid | compare-expr path-ref != null |
| C12 | `workflow.inputs.mode == 'auto'` | valid | scope=workflow |
| C13 | `steps.draft-entry.status == 'completed'` | valid | scope=steps, identifier hyphen |
| C14 | `steps.draft-entry.outputs.content == 'x'` | valid | path-ref multi-segment |
| C15 | `event.branch == 'main' \|\| event.branch == 'master'` | valid | or-expr \|\| |
| C16 | `steps.build.status == 'completed' && event.branch == 'main'` | valid | and-expr && |
| C17 | `!(event.branch == 'main')` | valid | not-expr ! |
| C18 | `(event.branch == 'main')` | valid | not-expr ( condition ) |
| C19 | `(steps.a.status == 'completed' \|\| event.branch == 'main') && steps.b.status == 'completed'` | valid | nested parentheses / grouping |
| C20 | `event.count + 1 == 5` | invalid | PROHIBITED: arithmetic operators |
| C21 | `len(event.items) > 0` | invalid | PROHIBITED: function calls |
| C22 | `event.branch == 'ma' + 'in'` | invalid | PROHIBITED: string concatenation |
| C23 | `steps['draft-entry'].status == 'fallback'` | invalid | PROHIBITED: bracket notation |
| C24 | `event.branch == 'main' ? true : false` | invalid | PROHIBITED: ternary operators |
| C25 | `event.branch == 'main` | invalid | string-lit (unterminated) |
| C26 | `steps.draft-entry.status > null` | invalid | null only valid with == / != |
| C27 | `steps.draft-entry.status` | invalid | bare path-ref is not a condition |

**Row counts:** Template = 15 (8 valid, 7 invalid); Condition = 27 (19 valid, 8 invalid);
total = 42. These equal the entry counts in `internal/expr/corpus/corpus.go`.
