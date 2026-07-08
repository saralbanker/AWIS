# M08 — Dependency Map
Canonical full edge list: `docs/07-indices/dependency-index.md`.

## Upstream (M08 requires)
| Milestone | Artifact consumed |
|---|---|
| M06 | `internal/engine.Engine` (Submit, Cancel, Run, Tick, Ingest); `internal/storage.SQLiteStorage` (StoragePort + additive methods); `internal/core.*` (all types) |
| M07 | `engine.Signal(ctx, instanceID, name, payload)` (signal intake); `storage.AppendAudit` (audit API); `storage.ListInstances` (via StoragePort method 11 — added M03) |

**Branch discipline:** `m08-sdk-public-surface` branches from `main` **after M07 squash-merges**.
Do not branch from `m07-signal-subsystem`; the branch must start from a clean main.

## Downstream (blocks)
| Milestone | What it needs from M08 |
|---|---|
| M09 | `sdk.Runtime`, `sdk.NewRuntime`, `sdk.WorkflowBuilder` — TestHarness wraps the real Runtime |
| M10 | `sdk.WorkflowBuilder` type (YAML DSL produces `*core.WorkflowDefinition`; same shape) |
| M14 | `sdk.Runtime`, `sdk.Config` — CLI wraps Runtime |
| M15 | Full `sdk` surface — OIP imports `sdk` only |

## Critical path position
M00 → M01 → M02 → M03 → M06 → M07 → **M08** → M11 → M12 → M13 → M15 → M18 (≈28d)

M08 is on the critical path. M09/M10 float work can proceed on a separate branch once M08 API
is visible (they don't need M08 to merge — they can branch from M08's branch for integration).
