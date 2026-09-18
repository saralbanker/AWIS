# M16 — Dependency Map
## Upstream (M16 requires)
| Milestone | Artifact consumed |
|---|---|
| M04 | IntelligencePort contract (frozen; `internal/intelligence/porttest` contract suite) |
| M14 | CLI config UX (soft dependency, per README.md — `+M14 for config UX`) |

**Branch discipline:** `m16-anthropic-adapter` stacked on `m15-oip-on-awis`.

## Downstream (blocks)
| Milestone | What it needs from M16 |
|---|---|
| M17 (README.md) | soft dependency: `recall --synthesize` works against NullAdapter empty-state without M16; M16 adapter used when an API key is present |
| M18 (README.md, hard block) | AnthropicAdapter as an available (non-required) intelligence path for the dogfood window |

## Critical path position
Off the critical path (README.md: "Window: Week-5 float / Week 6"); a hard block only on M18
per README.md's dependency line, not on M17 (soft dependency only).

## Status note
M16 sat at `PHASE: E-MERGE (blocked on founder)` per `docs/05-implementation/STATE.md`. The
`m16-anthropic-adapter` branch was merged into `main` at `f3a897b` on 2026-09-18 as part of a
repository consolidation. Unlike M15, M16 carries no gate (`GATE: none`), so no founder
verdict was outstanding for this module specifically — see M15
`docs/05-implementation/M15-oip-on-awis/DEPENDENCY_MAP.md` for the G3/TDS-06 gap that applies
to the branch M16 is stacked on.
