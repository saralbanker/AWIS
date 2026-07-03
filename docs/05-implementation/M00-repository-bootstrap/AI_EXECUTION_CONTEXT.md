# M00 — AI Execution Context
**Model allocation (IMP §28):** Sonnet — mechanical scaffolding, low architectural sensitivity. Human merges.

## Session loading order (≈6k tokens total)
1. `docs/00-foundation/README.md` (authority order + contradiction protocol)
2. This module's `IMPLEMENTATION_SPEC.md` + `VALIDATION_CHECKLIST.md`
3. IMP §4–§5 verbatim ONLY if the spec seems ambiguous (it should not — escalate instead)

Do NOT load: Blueprint, PRD, Finalization, Constitution. M00 touches no frozen interface.

## Hard constraints
- The IMP §5 tree is exhaustive: create only what it names (plus the three EDR files).
- `apps/oip/go.mod` declares `module github.com/awis/oip` — never merge the modules "for simplicity".
- CI must fail loudly on lint findings from commit one; do not soften lint config to get green.
- The `e1` Makefile target may be a stub, but it must exist and be wired into CI as a named (allowed-to-skip-until-M6) job so the AWIS-E1 gate slot is visible from day one.
- No secrets anywhere; CI uses no API keys (Constitution/PRD secret rules bind from the first commit).

## Escalate to human when
- Any tool/framework choice not covered by edr-001..003 arises.
- The two-module workspace fights the chosen CI setup.
