# M11 — Dependency Map
Canonical full edge list: `docs/07-indices/dependency-index.md`.

## Upstream (M11 requires)
| Milestone | Artifact consumed |
|---|---|
| M06 | `engine.Runner` interface (engine.go:45) — the contract SubprocessRunner implements; runners-map extension point |
| M08 | `sdk.NewRuntime` wiring point (runtime.go runners map); RegisterWorkflow/Submit for e2e |
| M09 | `awistesting` harness — drives the e2e keystone (Python step inside a workflow) |
| M01 | frozen `core.StepError` shape (TDS-01 §2.1) — protocol error envelope maps onto it verbatim |

**Branch discipline:** `m11-subprocess-runner` is stacked on `m10-yaml-dsl` (founder directive
2026-07-10: milestones proceed while E-MERGEs queue; M11 does not consume M10 artifacts).

## Downstream (blocks)
| Milestone | What it needs from M11 |
|---|---|
| M12 (hard block) | subprocess spawn/exchange patterns + TDS-04 as the protocol-authoring precedent for TDS-05; error-mapping code classes |
| M15 | polyglot capability proven (OIP contingency paths) |

## Critical path position
M11 is ON the critical path: M08 → **M11** → M12 → M13 → M15 → M18 (IMP §10). The plugin
chain is on the critical path solely because OIP's `assemble-context` step is plugin-typed
(IMP §10 note).
