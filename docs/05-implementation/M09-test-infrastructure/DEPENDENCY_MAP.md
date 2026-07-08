# M09 — Dependency Map
Canonical full edge list: `docs/07-indices/dependency-index.md`.

## Upstream (M09 requires)
| Milestone | Artifact consumed |
|---|---|
| M08 | `sdk.Runtime`/`sdk.NewRuntime`/`sdk.Config`; `sdk.WorkflowBuilder`; `sdk.SQLiteStorage`; `Runtime.Tick` (manual advance); WorkflowRunner surface (Submit/Signal/Status/Cancel/List) |
| M07 | signal subsystem (WAIT steps, `engine.Signal`) — exercised by harness WAIT/signal tests |
| M06 | `internal/engine.Engine` + `engine.Config.Clock` (existing seam); retry/fallback/compensation/cancellation semantics under test |
| M04 | `core.IntelligencePort` (mock implements it); NullAdapter; Dispatcher/CapabilityRouter |

**Branch discipline:** `m09-test-infrastructure` branches from `main` **after M08
squash-merges**. Do not branch from `m08-sdk-public-surface`.

## Downstream (blocks)
| Milestone | What it needs from M09 |
|---|---|
| M15 (hard block) | WorkflowTestHarness — OIP workflows are developed test-first on it (IMP §27.M9 Blocks row) |
| M10 | harness for YAML↔builder equivalence-oracle tests |
| M11/M12 | harness integration for subprocess/plugin step tests |
| M18 | QG-5 is re-executed literally at G4 |

## Critical path position
M09 is OFF the critical path (float; Week 3 parallel — IMP §11). Critical path runs
M08 → M11 → M12 → M13 → M15 → M18. M09 must merge before M15 begins (Blocks row) and is the
designated absorber of schedule pressure (IMP §10 note).
