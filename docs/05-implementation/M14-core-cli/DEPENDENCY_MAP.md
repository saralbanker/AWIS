# M14 — Dependency Map
## Upstream (M14 requires)
| Milestone | Artifact consumed |
|---|---|
| M06/M07 | engine run loop, cancellation, signal path (CLI writes ride the 100ms tick) |
| M08 | sdk.NewRuntime/SQLiteStorage/RegisterWorkflow/Submit + Runtime surface |
| M10 | dsl.Discover/ParseFile/ValidateFile + PRD §18 rendering |
| M12 | PluginStore + manifest parse (F-5 minimal install/list) |
| M07 (F-4) | audit API for WorkflowRegistered write site |

**Branch discipline:** `m14-core-cli` stacked on `m13-git-context-plugin`.

## Downstream (blocks)
| Milestone | What it needs from M14 |
|---|---|
| M15 (hard block) | the dev loop: start/submit/status/trace/signal + plugin install/list (F-5) — G3 runs OIP "via CLI" |
| M17 | command mux, error renderer, TDS-07 contract, golden-test harness (mechanical batch extends them) |
| M18 | QG-1 (init→trace ≤5min) and QG-2 (trace readable) measure M14 formats |

## Critical path position
OFF the critical path but a HARD M15 dependency (IMP §11 Week-4/5 parallel track; §10 note).
