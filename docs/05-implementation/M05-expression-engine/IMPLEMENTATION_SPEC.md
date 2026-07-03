# M05 — Implementation Spec (compiled 2026-07-03 from IMP §27.M5 + TDS-03 + Finalization B2 + IR-2 per IKB §4 row)

## Objective (IMP §27.M5)
Hand-written implementations of BOTH Finalization B2 grammars + WorkflowValidator.
`Out:` `internal/expr`, `internal/validate`. `Dep:` M01 (TDS-03 + 42-fixture corpus = the oracle).
**NO external expression library** — B2 explicitly rejected Option D (expr-lang/CEL/JSONata).
**Risk IR-2:** grammar drift from Finalization text — mitigations: corpus written before parsers
(done at M01), fuzzing here.

## Tasks

### T1 — Template grammar (`internal/expr`): parser + resolver
Grammar (TDS-03 §1, frozen): `template-string ::= literal-text | "{{" path-ref "}}" | concat`;
`path-ref ::= scope "." identifier ("." identifier)*`; `scope ::= workflow|steps`;
`identifier ::= [a-zA-Z_][a-zA-Z0-9_-]*` (hyphens are single tokens, never subtraction).
- `ParseTemplate(src string) (*Template, error)` — parse-time verdict per corpus Table 1; errors
  carry precise byte **Position** + message (AC: "every prohibited construct rejected with precise
  position"). Prohibited (must reject): arithmetic, function calls, `{{if …}}`, nested templates,
  bracket notation, scope other than workflow|steps, bare scope without identifier.
- `(*Template).Resolve(env Env) (string, []Warning)` — execution-time semantics (FR-WD-05, frozen):
  missing path OR not-yet-completed step ⇒ that ref resolves to `""` AND a Warning is appended.
  **Templates never fail at resolution** — no error return. Warnings are data (engine logs them at
  M06 — Logger shape is M06-owned; document this transport decision).
- Non-string values at a resolved path stringify deterministically (document: strconv for
  bool/number per JSON canonical forms, fmt %v otherwise — EDR-010 note).

### T2 — Condition grammar (`internal/expr`): parser + evaluator
Grammar (TDS-03 §2, frozen): precedence `||` < `&&` < `!`; parentheses; compare-expr =
`path-ref op value` | `path-ref ==|!= null`; value = single-quoted string | number | bool;
scope workflow|steps|event. `event` scope is grammatically valid everywhere at parse level —
restriction "only in Trigger filter fields" is enforced by the VALIDATOR (T3), not the parser
(corpus C1 etc. are valid rows with event scope and no field context).
- `ParseCondition(src string) (*ConditionExpr, error)` — corpus Table 2 verdicts; precise Position
  on every rejection (incl. C25 unterminated string, C26 null with ordering op, C27 bare path-ref).
- `(*ConditionExpr).Eval(env Env) (bool, error)` — frozen null rules: ordering op vs null ⇒ false;
  `null == null` ⇒ true (both sides: path==null literal compares resolved-null; two paths can't be
  compared — grammar only allows path OP value/null). Unspecified semantics filled by **EDR-010**
  (strict, no coercion): numbers compare numerically (int/decimal unified float64); string==string
  lexical equality only for ==/!=; ordering ops defined for numbers ONLY — non-numeric or
  type-mismatched operands ⇒ ordering evaluates false, equality ⇒ false, inequality ⇒ true
  (pattern-consistent with the frozen null rule). Eval error only for environment access failure,
  never for type mismatch.
- Shared internals (lexer, path-ref parsing) may be common to T1/T2 inside `internal/expr`.

### T3 — Env: shared resolution environment (`internal/expr`)
`Env` = the M03/EDR-007 variables shape: `{"inputs": map, "<step_id>": outputs-map}` + step
statuses + optional event payload. Concrete struct:
`Env{Inputs map[string]any; StepOutputs map[string]map[string]any; StepStatus map[string]string; Event map[string]any}`.
Path semantics: `workflow.inputs.<key…>` walks Inputs; `steps.<id>.outputs.<key…>` walks
StepOutputs[id]; `steps.<id>.status` reads StepStatus[id] (pending|running|completed|failed|cancelled);
`event.<key>` reads Event top-level (conditions only). Nested map walking for multi-segment keys;
absent anywhere ⇒ null/missing.

### T4 — WorkflowValidator (`internal/validate`, FR-WD-03 + PRD §18 checklist)
`Validate(def core.WorkflowDefinition) []Issue` — structured issues (never panics; collects ALL
issues, not first-fail). `Issue{Code, StepID, Field, Message, Position}` — rendering with
file/line/suggestion is M10/M14's job (PRD §18 formats); M05 delivers the data.
Checks (PRD §18 "Validation checks (required)" + FR-WD-03):
1. `initial_step` references a defined step; `final_steps` all defined.
2. No orphaned steps (unreachable from initial_step via transitions + fallback edges).
3. No cycles (FR-WD-13) — transitions graph (fallback edges included in reachability, and in cycle
   detection as edges? DECISION: cycle check over transition edges only; fallback edges join
   reachability but a fallback cannot re-enter the path — document; EDR-010 note).
4. All `transition.from`/`to` reference defined step ids.
5. All `fallback` references point to steps defined in the same workflow.
6. All `transition.condition` parse per grammar (position in Issue); `event` scope in a
   transition condition is an Issue (event only valid in trigger filters, TDS-03).
7. All `trigger.config.filter` (when present, string) parse per grammar; event scope allowed here.
8. Intelligence steps: `context_budget` positive int; `model_hint` ∈ {fast, quality, local, ""}.
9. Handler refs: presence-only at M05 (non-empty Handler for native/subprocess/plugin steps);
   runtime resolvability is registration-time (M08) per PRD §18 "resolvable at runtime".
10. type=signal steps have WaitSignal; type=intelligence steps have Intelligence (schema-implied).

### T5 — Corpus conformance + fuzzing (IR-2)
- Corpus test: iterate `corpus.Template` and `corpus.Condition`; every row's parse verdict must
  equal `Valid`; for every invalid row assert Position is present and within the source bounds.
  42/42 required — the corpus is the oracle; parsers adapt to it, NEVER the reverse (corpus edits
  are G1-frozen-adjacent and forbidden here).
- Fuzz: `FuzzParseTemplate` + `FuzzParseCondition` (`testing.F`), corpus rows as seed inputs;
  invariant: no panics, and on nil error the parse tree re-serializes/re-parses consistently where
  cheap. CI runs fuzz in short mode (`-fuzztime` not in CI; the fuzz functions run their seed
  corpus as unit tests by default — that satisfies "fuzzing zero panics" for the gate; a longer
  local run is executed once during this milestone and its duration recorded in HANDOFF).
- Evaluation tests: frozen null rules verbatim; EDR-010 matrix (each cell); template resolution
  incl. FR-WD-05 ""+warning, concatenation, multi-ref, non-string stringification.

## Scope walls
- NO YAML parsing (M10). NO engine integration (M06). NO CLI rendering (M14). NO logger.
- `internal/core` is UNTOUCHED this milestone. `sdk/` untouched. corpus.go untouched (oracle).
- No new go.mod dependencies (hand-written per B2 decision).

## Acceptance criteria (IMP §27.M5)
- AC-1: full corpus passes — 42/42 verdicts, invalid rows with precise positions.
- AC-2: fuzzing zero panics (seed corpus in CI; longer local run recorded).
- AC-3: missing template path → `""` + Warning (FR-WD-05).
- AC-4: validator implements every PRD §18 required check listed in T4 (structured Issues).
- AC-5: `make build test lint` green; zero new dependencies; grammar docs cross-linked (DoD:
  package docs reference docs/EXPRESSION_GRAMMARS.md and EDR-010).
