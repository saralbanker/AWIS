# M08 → M09/M10/M14/M15 Handoff
**Status: ACTUALS CONFIRMED at M08 D-CLOSE (2026-07-08); merge commit filled at E-MERGE.**

## Guaranteed outputs (confirmed as actuals, D-CLOSE 2026-07-08)
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

## What M09 may assume (confirmed)
- `sdk.NewRuntime(cfg)` is stable and fully wired.
- `sdk.WorkflowBuilder` can produce valid definitions.
- `sdk.WorkflowRunner` can Submit + tick (`Runtime.Tick`) + read Status deterministically.
- The TestHarness (M09) drives the **real** engine via `NewRuntime` with injectable clock/id
  (IMP §28 M9 row — "harness drives the *real* engine with deterministic sources").
  **Seam status at M08 exit:** `engine.Config.Clock` is injectable; the engine ID source is NOT
  yet injectable (`newUUIDv4` hardcoded at engine.New) and `sdk.Config` does not yet pass
  clock/id through — both seams are M09-C1 work (additive fields; nil ⇒ current behavior).
- `sdk.Config.Intelligence` is accepted but not yet wired to a runner: `NewRuntime` assembles
  the NativeRunner only, exactly per the M08 spec. Wiring `cfg.Intelligence` → the
  `internal/runner/intelligence` Runner (NullAdapter when nil) is M09-C1 work.

## What M10/M14/M15 may assume (confirmed)
- M10 (YAML DSL): `core.WorkflowDefinition` shape is identical to `sdk.WorkflowDefinition` (F-1).
- M14 (CLI): `sdk.Runtime.Start(ctx)` drives the pull loop.
- M15 (OIP): full `sdk` surface available; imports `sdk` only.

## Known limitations (confirmed)
- `RecallAPI.ReplayInstance` returns an empty stub until M17.
- Storage must be `*storage.SQLiteStorage` for all current functionality (no V2 adapter yet).
- `Runtime.Submit` resolves "latest registered version" as last-visited during map iteration —
  deterministic only while a single version per definition id is registered. Acceptable for V1
  (registration sets are small and single-version in practice); proper semver-max resolution is
  a candidate for the M09/M10 window.
- `WorkflowStatus.Inputs` and `.Outputs` both expose the instance `Variables` snapshot (same
  map); the shape freeze is unaffected. Split when the engine separates input/output channels.
- `StepStats` scans a fixed 10-year event window via `ReadEventRange` (bounded-query
  workaround); adequate for V1 data volumes.

## Actuals (filled at completion)
- Merge commit: 87c1632 (founder squash-merge to main; recorded at M09 D-CLOSE, 2026-07-09)
- Verification: **M08-V1 PASS 21/21** (awis-verifier, 2026-07-08, HEAD 728cfed) — first V1 run
  FAILED with 5 test gaps (6dd2bb5); revision card M08-C2r closed them at e8aca86; re-verified.
- Deviations:
  - `sdk.SQLiteStorage(path)` and `Runtime.Tick(ctx)` added beyond the spec's explicit list —
    both canonical (Blueprint §12 L934 shows `awis.SQLiteStorage`; Tick is required for the
    spec §8 example's "runs one tick" without importing internal/). Not drift.
  - C2r revision card cut after V1 FAIL (EEOS §4.2 — new file, no card edits).
  - No model substitutions; C1/C2/C2r on awis-builder (Sonnet), V1 on awis-verifier per DISPATCH.
