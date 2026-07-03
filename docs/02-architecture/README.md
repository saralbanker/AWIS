# 02 — Architecture (pointers, FROZEN)

Canonical, immutable, at repository root:
- `../../AWIS_ARCHITECTURE_BLUEPRINT.md` — 33 sections, 15 ADRs. The architecture.
- `../../AWIS_ARCHITECTURE_FINALIZATION.md` — 7 blocker resolutions (expression grammars,
  signal atomicity SQL, cancellation semantics, FTS ownership, compensation rules, StepType enum).
  Overrides the Blueprint where they touch.
- `../../ENGINEERING_ARCHITECTURE_BLUEPRINT.md` — SUPERSEDED (see ../../archive/). Never implement from it.

Any implementation question about structures (§6), execution (§8), state (§9), plugins (§11),
SDK (§12), intelligence (§13–17), or storage (§20) resolves at the cited Blueprint section.
