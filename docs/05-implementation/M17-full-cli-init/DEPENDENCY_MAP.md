# M17 — Dependency Map
## Upstream (M17 requires)
| Milestone | Artifact consumed |
|---|---|
| M14 | mux/error renderer/golden-test harness pattern (this milestone extends it mechanically) |
| M12 | plugin registry/manager (`plugin status\|remove` commands read plugin-system state) |
| M16 (soft, README.md) | `recall --synthesize` routes through the M16 adapter when an API key is present; falls back to NullAdapter empty-state without it |

**Branch discipline:** `m17-full-cli-init` stacked on `m16-anthropic-adapter`.

## Downstream (blocks)
| Milestone | What it needs from M17 |
|---|---|
| M18 (README.md, hard block) | the full CLI command tree, `awis init` scaffold, QG-1 path plumbing (README.md: "measured loosely here, formally at M18" per TRACEABILITY.md C3 entry) |

## Critical path position
On the critical path to M18: M18's README.md lists M17 as a hard dependency. **As of this
document, M17 is NOT verified-complete** — PHASE is `C-VERIFY`, two independent M17-V1 runs
have returned FAIL, and row 8 (frozen-surface scope) is unresolved pending CE/founder
adjudication (STATE.md, current NEXT line). M18 should not assume M17's outputs are final
until a clean, independent M17-V1 PASS is recorded and D-CLOSE runs.

## Status note
STATE.md records M17's `m17-full-cli-init` branch as stacked on `m16-anthropic-adapter`,
which itself sat at `PHASE: E-MERGE (blocked on founder)`. Both branches were merged into
`main` at `f3a897b` on 2026-09-18 as part of a repository consolidation, while M17 itself was
still in `PHASE: C-VERIFY` with an open, unresolved V1 blocker (row 8) — i.e. this module's
own outputs were not signed off by an independent M17-V1 PASS before landing on `main`.
