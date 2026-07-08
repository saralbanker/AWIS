# M08 → M09/M10/M14/M15 Handoff
**Status: STAGED — activates at M07 merge; actuals filled at M08 completion.**

## Guaranteed outputs (contract — to be confirmed as actuals)
- `sdk.Runtime` + `sdk.NewRuntime(cfg)`: fully wired constructor wrapping the engine.
- `sdk.RegisterHandler` / `sdk.RegisterWorkflow` with PRD §18 error format.
- `sdk.WorkflowBuilder` + `Build()` with semver validation.
- `sdk.WorkflowRunner` (Submit/Signal/Status/Cancel/List) — complete functional implementation.
- `sdk.RecallAPI` (QueryHistory/StepStats) — `ReplayInstance` is a stub (M17).
- `sdk.TriggerAPI.SubmitEvent` — F-2 sdk intake surface for domain events.
- `WorkflowRegistered` audit call site in `RegisterWorkflow` (F-4).
- `examples/hello_workflow/` — compiles against `sdk` only; boundary proven.
- `core.WorkflowStatus`, `core.HistoryQuery`, `core.ExecutionRecord`, `core.StepStatistics`
  shapes FROZEN at M08 merge (IMP §13).

## What M09 may assume (drafted; confirm at completion)
- `sdk.NewRuntime(cfg)` is stable and fully wired.
- `sdk.WorkflowBuilder` can produce valid definitions.
- `sdk.WorkflowRunner` can Submit + tick + read Status deterministically.
- The TestHarness (M09) drives the **real** engine via `NewRuntime` with injectable clock/id
  (IMP §28 M9 row — "harness drives the *real* engine with deterministic sources").

## What M10/M14/M15 may assume (drafted; confirm at completion)
- M10 (YAML DSL): `core.WorkflowDefinition` shape is identical to `sdk.WorkflowDefinition` (F-1).
- M14 (CLI): `sdk.Runtime.Start(ctx)` drives the pull loop.
- M15 (OIP): full `sdk` surface available; imports `sdk` only.

## Known limitations (drafted)
- `RecallAPI.ReplayInstance` returns an empty stub until M17.
- Storage must be `*storage.SQLiteStorage` for all current functionality (no V2 adapter yet).

## Actuals (filled at completion)
- Merge commit: _ · Verification: _ · Deviations: _
