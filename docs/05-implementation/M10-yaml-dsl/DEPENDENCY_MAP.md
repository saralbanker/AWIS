# M10 — Dependency Map
Canonical full edge list: `docs/07-indices/dependency-index.md`.

## Upstream (M10 requires)
| Milestone | Artifact consumed |
|---|---|
| M08 | `sdk.WorkflowBuilder` (the other half of the FR-WD-02 equivalence oracle); F-1 alias surface |
| M05 | `internal/validate` (WorkflowValidator — Issues data); `internal/expr` (grammars, via validate) |
| M09 | `awistesting` harness — event-stream equivalence oracle (M09 HANDOFF: "YAML↔builder equivalence oracle can assert identical harness event streams") |
| M01 | TDS-02 frozen WorkflowDefinition serialization (field names verbatim) |

**Branch discipline:** `m10-yaml-dsl` branches from `main` **after M09 squash-merges**.
Do not branch from `m09-test-infrastructure`.

**O-1 note (module README):** M10 may start after M01+M05 if float is needed; that option was
not exercised — M08 and M09 are merged predecessors.

## Downstream (blocks)
| Milestone | What it needs from M10 |
|---|---|
| M14 (hard block) | `dsl.Discover` + `dsl.ValidateFile` + PRD §18 rendering for `awis start` / `workflow validate` |
| M15 (hard block) | `apps/oip/workflows/*.yaml` fixtures registered unchanged; YAML tier proven equivalent |
| M17 | the three `examples/workflows/*.yaml` embedded and emitted by `awis init` (FR-RM-01) |

## Critical path position
M10 is OFF the critical path (float; Week 3 parallel — IMP §11). Critical path runs
M08 → M11 → M12 → M13 → M15 → M18. M10 must merge before M14/M15 begin and is a designated
absorber of schedule pressure (IMP §10 note).
