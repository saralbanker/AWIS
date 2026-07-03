# M05 → M06/M08/M10 Handoff
**Status: IN EXECUTION** — Actuals filled at completion; merge commit at merge.

## Guaranteed outputs (contract)
- `internal/expr`: ParseTemplate/Template.Resolve, ParseCondition/ConditionExpr.Eval, shared Env;
  corpus 42/42; fuzz targets with zero panics; precise-position errors.
- `internal/validate`: Validate(WorkflowDefinition) []Issue covering the full PRD §18 required list
  (M05 scope: structural + grammar + field checks; handler presence only).

## What M06 may assume
- Resolve never errors: missing/incomplete refs yield "" + Warning — engine MUST log warnings (FR-WD-05).
- Eval implements the frozen null rules + EDR-010 strict semantics; engine never needs type guards.
- Env is EDR-007-compatible: inputs map + per-step outputs + step statuses.

## What M10 may assume
- Issues carry Code/StepID/Field/Message/Position — enough to render PRD §18 error format with
  the YAML file/line added by the DSL layer.
- event-scope-in-transition-condition is already an Issue; the DSL layer need not re-check grammar.

## Known limitations (by design)
- Evaluation semantics beyond the frozen null rules are EDR-010 (G2 docket).
- Handler resolvability is presence-only until M08 registration.

## Actuals (filled at completion)
- Merge commit: _
