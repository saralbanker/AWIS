# M08 — Validation Checklist (binary; exit list)
Global DoD (IMP §24) + M08 rows (IMP §27.M8, FR-SDK-*). Verifier executes via cards/M08-V1.md.
**Verified: M08-V1 PASS (awis-verifier, 2026-07-08, HEAD 728cfed).** V1's table reported 21 rows
(the two build-time boundary rows — example compiles + internal unimportable — share one build
evidence line); all 22 boxes below are covered by that verdict. Count noted at D-CLOSE, no gap.

- [x] V-COMMON block all ✅ (`make build test lint race` + `make e1`; clean tree; HEAD recorded)
- [x] `NewRuntime(cfg)` returns non-nil `*Runtime`; `cfg.Namespace=""` returns error
- [x] `RegisterHandler` with nil handler returns error with PRD §18 format
- [x] `RegisterWorkflow` with nil def returns error with PRD §18 format
- [x] `RegisterWorkflow` with version `"1.0"` (not semver) returns error
- [x] `RegisterWorkflow` with duplicate (id+version) returns error with PRD §18 format
- [x] `WorkflowBuilder.Build()` rejects `"1.0"` and `"not-a-version"`; accepts `"1.0.0"` and `"2.1.3"`
- [x] `WorkflowRunner.Submit` starts a workflow instance and returns a non-empty `InstanceID`
- [x] `WorkflowRunner.Signal` delivers to a waiting instance (integration: WAIT step resumes on next tick)
- [x] `WorkflowRunner.Status` returns `WorkflowStatus` with populated `InstanceID` and `Status` fields
- [x] `WorkflowRunner.Cancel` cancels a running instance; `Status` subsequently shows Cancelled
- [x] `WorkflowRunner.List` returns non-empty slice after at least one Submit; filter by DefinitionID works
- [x] `RecallAPI.QueryHistory` returns `[]ExecutionRecord` with ≥1 record after completed workflow run
- [x] `RecallAPI.StepStats` returns `StepStatistics` with `TotalRuns ≥ 1` after completed run
- [x] `RecallAPI.ReplayInstance` returns without panic (stub OK)
- [x] TriggerAPI (F-2): `Runtime.Triggers().SubmitEvent(ctx, ev)` inserts a row in `domain_events`
- [x] `WorkflowRegistered` audit row written after `RegisterWorkflow` on storage with `AppendAudit`
      (SELECT from audit_log: exactly one `WorkflowRegistered` row per registration call)
- [x] `core.WorkflowStatus`, `core.HistoryQuery`, `core.ExecutionRecord`, `core.StepStatistics`
      shapes filled (no longer empty structs)
- [x] Example app compiles: `go build ./examples/...` green
- [x] FR-SDK-09 boundary: `grep -rn '"github.com/awis/awis/internal' examples/` returns empty
- [x] Every exported sdk type/function has a godoc comment (spot-check 5 identifiers)
- [x] `internal/` is unimportable from outside the module: example app import verified at build time
