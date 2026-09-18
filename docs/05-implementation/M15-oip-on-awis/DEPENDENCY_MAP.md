# M15 — Dependency Map
## Upstream (M15 requires)
| Milestone | Artifact consumed |
|---|---|
| M09 | `sdk/testing` WorkflowTestHarness + DeterministicMode (QG-3 tests run against the real engine, deterministic sources) |
| M10 | dsl.ParseFile / YAML discovery (capture-decision + recall-decision workflow definitions register through this path) |
| M13 | git-context-plugin (reference plugin pattern; not directly consumed by OIP's own handlers, but the plugin-system proof M15 relies on) |
| M14 | CLI dev loop (start/submit/status/trace/signal/plugin install) — G3 runs OIP "via CLI" |

**Branch discipline:** `m15-oip-on-awis` stacked on `m14-core-cli`.

## Downstream (blocks)
| Milestone | What it needs from M15 |
|---|---|
| M16 | stacked branch continuation (`m16-anthropic-adapter` stacked on `m15-oip-on-awis`) |
| M18 (hard block, README) | QG-3/QG-4 evidence; OIP as the dogfood application for the 1-week dogfood window |

## Critical path position
On the critical path to G3 (founder-only gate) and, transitively, to M18 — every downstream
milestone (M16, M17, M18) is stacked on `m15-oip-on-awis` and cannot merge to `main` ahead of it.

## Status note
M15 sat at `PHASE: E-MERGE (blocked on founder: G3 VERDICT + TDS-06 SIGN-OFF)` per
`docs/05-implementation/STATE.md`. The `m15-oip-on-awis` branch (and the stacked `m16-`,
`m17-` branches above it) were merged into `main` at `f3a897b` on 2026-09-18 as part of a
repository consolidation, **without** a recorded founder G3 verdict or TDS-06 sign-off in the
repo. This gap is recorded here for visibility; resolving it is out of scope for this
document.
