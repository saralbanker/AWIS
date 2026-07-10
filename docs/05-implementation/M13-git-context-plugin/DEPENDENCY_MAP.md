# M13 — Dependency Map
## Upstream (M13 requires)
| Milestone | Artifact consumed |
|---|---|
| M12 | awis_plugin lib + testing.mock_request; TDS-05; Manager.Register; resolution rule |
| M09 | harness for the e2e workflow |

**Branch discipline:** `m13-git-context-plugin` stacked on `m12-plugin-system`.

## Downstream (blocks)
| Milestone | What it needs from M13 |
|---|---|
| M15 (hard block) | `git-context-plugin` registered → OIP capture-decision `assemble-context` step runs |

## Critical path position
ON the critical path: M12 → **M13** → M15 → M18 (IMP §10).
