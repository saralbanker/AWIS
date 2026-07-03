# M02 — Dependency Map
## Upstream (consumes)
| Artifact | From | How |
|---|---|---|
| TDS-01 envelope (9 cols, ADJ-1) | M01 | `execution_events` DDL + struct mapping derive mechanically |
| TDS-02 serialization + immutability | M01 | `workflow_definitions.definition` JSON; re-register rule |
| `core.StoragePort` (12 methods) + core types | M01 (F-1) | adapter implements; 8 methods live, 4 stubbed |
| edr-003 driver decision | M00 | `modernc.org/sqlite`, WAL default |
| Makefile `contract` slot | M00 | stub replaced by real suite |

## Downstream (produces for)
| Artifact | Consumed by | How |
|---|---|---|
| Durable EventLog append/read | M03 (rebuild-state), M06 (engine), M07 (signal tx) | the platform's source of truth |
| Migration runner + 0001 (partial) | M03 folds `workflow_instances` in; M06+ add tables | sequential pre-tag fold-forward |
| WorkflowRegistry | M08 (registration API), M10 (DSL registration) | immutable (id,version) store |
| StepResultCache | M06 (idempotency check) | claim-dedup mechanism |
| Contract suite (storagetest) | V2 Postgres adapter; M03 completes coverage | adapter-agnostic factory pattern |

## Critical-path position
M00 → M01 → **M02** → M03 → M06 → … Slip shifts the path 1:1; Track B (M04) and Track C (M05) run parallel and are unaffected.
