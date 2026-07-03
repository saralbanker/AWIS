# AWIS Implementation Knowledge Base (IKB) — Root Router

**Status:** Operational implementation interface. The canonical documents at the repository
root are IMMUTABLE; this tree partitions and cross-references them — it never restates them.

## Authority order (conflicts resolve upward)
OIP_CONSTITUTION.md → AWIS_ARCHITECTURE_FINALIZATION.md → AWIS_ARCHITECTURE_BLUEPRINT.md
→ AWIS_PRD.md → IMPLEMENTATION_MASTER_PLAN.md (+ IMPLEMENTATION_MASTER_PLAN_VERIFICATION_REPORT.md amendments F-1..F-5)

## Tree
- `00-foundation/` — how to load this KB (AI session protocol)
- `01-vision/` — pointer: constitution & vision
- `02-architecture/` — pointers: blueprint + finalization (frozen)
- `03-product/` — pointers: canonical PRD + investigations
- `04-planning/` — pointers: IMP (archival) + verification report (binding amendments)
- `05-implementation/` — one module per milestone M00–M18 (the operational surface)
- `06-reference/` — normative TDS specs (created BY milestones; IMP §12 `docs/X.md` paths resolve here)
- `07-indices/` — traceability matrix, dependency index, cross-reference index, canonical map
- `../archive/` — superseded documents (historical only)

## AI session loading protocol (deterministic)
1. Load `05-implementation/<current-milestone>/AI_EXECUTION_CONTEXT.md`.
2. Load `HANDOFF.md` of each direct predecessor (per the milestone's `DEPENDENCY_MAP.md`).
3. Load only the TDS files in `06-reference/` that the execution context names.
4. Escalate to a canonical document ONLY at the exact section cited — never load a canonical document whole.
5. Never load `IMPLEMENTATION_MASTER_PLAN.md` during implementation; its content is partitioned here.
