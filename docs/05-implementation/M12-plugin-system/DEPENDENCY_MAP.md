# M12 — Dependency Map
Canonical full edge list: `docs/07-indices/dependency-index.md`.

## Upstream (M12 requires)
| Milestone | Artifact consumed |
|---|---|
| M11 | subprocess spawn/kill/stderr patterns (cited, not modified); TDS-04 as TDS-05 authoring precedent; error-code naming |
| M07 | audit API (`internal/storage/audit.go`) — F-4 `PluginRegistered` write site |
| M06 | `engine.Runner` interface; runners-map extension point |
| M08/M09 | sdk runtime wiring point; awistesting harness for checkpoint + e2e |
| M02 | migration runner + storage patterns (0005 follows 0003/0004 conventions) |
| M10 | yaml.v3 dependency (already in go.mod) for manifest parsing |

**Branch discipline:** `m12-plugin-system` stacked on `m11-subprocess-runner` (founder
directive 2026-07-10).

## Downstream (blocks)
| Milestone | What it needs from M12 |
|---|---|
| M13 (hard block) | TDS-05 + `awis-plugin` lib + mock_request harness — the reference plugin implements against them |
| M14 | `Manager.Register` / PluginStore for `awis plugin install|list` (F-5 minimal surface) |
| M15 (hard block via M13) | OIP `assemble-context` is plugin-typed |

## Critical path position
M12 is ON the critical path: M08 → M11 → **M12** → M13 → M15 → M18 (IMP §10). Slip
contingency (IMP §10 note): M15 may start with stubbed plugin capability; fold real plugin in
before G3 verdict.
