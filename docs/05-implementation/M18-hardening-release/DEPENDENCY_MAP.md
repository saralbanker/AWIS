# M18 — Dependency Map
## Upstream (M18 requires)
| Milestone | Artifact consumed |
|---|---|
| M15 | OIP-on-AWIS boundary (G3 complete, TDS-06 signed) — QG-4 re-confirmation target |
| M16 | AnthropicAdapter — NFR-P-03 would exercise this if a credential existed (UNMEASURABLE; see IMPLEMENTATION_SPEC.md Non-scope) |
| M17 | Full CLI tree, `awis init` scaffold, embedded example handler — QG-1's formal walkthrough target; `docs/CLI_CONTRACT.md` for the acceptance re-measurement in C4 |
| M09 (via M17 chain) | `WorkflowTestHarness` — QG-5 re-confirmation target |
| M06 (via CI) | AWIS-E1 zero-AI gate — QG-3 re-confirmation target |

All five are closed and in `main` (STATE.md DONE-MILESTONES). M18's upstream dependency is
satisfied by repository state, not by a pending card.

**Branch discipline:** `m18-hardening-release`, branched directly from `main` at `4977de7`
(M15/M16/M17 already reached `main` by branch stacking per STATE.md LAST-9 reconciliation —
there is no milestone branch to stack on top of; M18 stacks on `main` itself).

## Downstream (blocks)
| Consumer | What it needs from M18 |
|---|---|
| `v1.0.0` tag | G4 verdict + goreleaser artifacts (this module's C3/C4/V1) |
| V2 planning | Blueprint §33: V2 planning is gated on OIP dogfood evidence, which this module's advisory dogfood week starts collecting (non-gating for M18 itself, but the evidence M18 produces is what unblocks V2 planning later) |
| Any MCP/OpenRouter/local-LLM work | IMPLEMENTATION_SEQUENCE.md (this repo's own direction report, 2026-10-01): all three explicitly wait on M18 shipping v1.0.0 first |

## Critical path position
M18 is the terminal node of the IMP §10 critical path
(M00→M01→M02→M03→M06→M07→M08→M11→M12→M13→M15→M18, ≈28d planned). Nothing in this repo's own
planning documents is scheduled to start after M18 without M18 having shipped — it is the
release gate, not an intermediate step.

## Status note
As of A-INIT (this document, 2026-10-10): M18 has zero committed cards, zero benchmark
numbers, zero release artifacts. The only fact this module can rely on is that `main` at
`4977de7` is CI-green on real GitHub Actions (both `ubuntu-latest` and `macos-latest`),
verified this session — not a local simulation (CURRENT_TRUTH_LEDGER.md §7 Limitation 1,
now resolved for this specific commit).
