# M08 — Validation Checklist (binary; exit list)
Global DoD (IMP §24) + M08 rows (IMP §27.M8, FR-SDK-*). Verifier executes via cards/M08-V1.md.

- [ ] V-COMMON block all ✅ (`make build test lint race` + `make e1`; clean tree; HEAD recorded)
- [ ] `NewRuntime(cfg)` returns non-nil `*Runtime`; `cfg.Namespace=""` returns error
- [ ] `RegisterHandler` with nil handler returns error with PRD §18 format
- [ ] `RegisterWorkflow` with nil def returns error with PRD §18 format
- [ ] `RegisterWorkflow` with version `"1.0"` (not semver) returns error
- [ ] `RegisterWorkflow` with duplicate (id+version) returns error with PRD §18 format
- [ ] `WorkflowBuilder.Build()` rejects `"1.0"` and `"not-a-version"`; accepts `"1.0.0"` and `"2.1.3"`
- [ ] `WorkflowRunner.Submit` starts a workflow instance and returns a non-empty `InstanceID`
- [ ] `WorkflowRunner.Signal` delivers to a waiting instance (integration: WAIT step resumes on next tick)
- [ ] `WorkflowRunner.Status` returns `WorkflowStatus` with populated `InstanceID` and `Status` fields
- [ ] `WorkflowRunner.Cancel` cancels a running instance; `Status` subsequently shows Cancelled
- [ ] `WorkflowRunner.List` returns non-empty slice after at least one Submit; filter by DefinitionID works
- [ ] `RecallAPI.QueryHistory` returns `[]ExecutionRecord` with ≥1 record after completed workflow run
- [ ] `RecallAPI.StepStats` returns `StepStatistics` with `TotalRuns ≥ 1` after completed run
- [ ] `RecallAPI.ReplayInstance` returns without panic (stub OK)
- [ ] TriggerAPI (F-2): `Runtime.Triggers().SubmitEvent(ctx, ev)` inserts a row in `domain_events`
- [ ] `WorkflowRegistered` audit row written after `RegisterWorkflow` on storage with `AppendAudit`
      (SELECT from audit_log: exactly one `WorkflowRegistered` row per registration call)
- [ ] `core.WorkflowStatus`, `core.HistoryQuery`, `core.ExecutionRecord`, `core.StepStatistics`
      shapes filled (no longer empty structs)
- [ ] Example app compiles: `go build ./examples/...` green
- [ ] FR-SDK-09 boundary: `grep -rn '"github.com/awis/awis/internal' examples/` returns empty
- [ ] Every exported sdk type/function has a godoc comment (spot-check 5 identifiers)
- [ ] `internal/` is unimportable from outside the module: example app import verified at build time
