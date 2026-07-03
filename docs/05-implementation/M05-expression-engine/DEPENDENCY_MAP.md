# M05 — Dependency Map

**Upstream (consumes):**
- M01: TDS-03 (frozen grammars) + `internal/expr/corpus` (42-row oracle) + core types
  (WorkflowDefinition/Step/Transition/Trigger/Condition) — read-only.
- M03 (shape only): EDR-007 variables scoping `{"inputs":…, "<step_id>": outputs}` mirrored by Env.

**Downstream (blocks):**
- M06 (engine): evaluates transition conditions via ConditionExpr.Eval; resolves step-input
  templates via Template.Resolve; logs returned Warnings (FR-WD-05); builds Env from projection
  variables + step statuses.
- M10 (YAML DSL): calls validate.Validate on parsed definitions; renders Issues with
  file/line/suggestion per PRD §18 formats (M05 ships data, not rendering).
- M08 (registration): registration-time validation path (FR-WD-04) + handler resolvability check.

**Handoff contract:** see HANDOFF.md.
