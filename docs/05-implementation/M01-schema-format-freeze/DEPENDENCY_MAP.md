# M01 — Dependency Map
## Upstream (consumes)
| Input | From |
|---|---|
| Green workspace, docs/, sdk/ slot | M00 |
| Blueprint §6/§9, Finalization Blocker 2 texts | frozen corpus (docs/02-architecture pointers) |
| F-1 alias pattern | Verification report §Must-Fix |

## Downstream (produces for)
| Artifact | Consumed by | How |
|---|---|---|
| TDS-01 | M02 | migration 0001 event columns; M03 rebuild; M06 replay tests |
| TDS-02 | M02 (definitions table), M08 (SDK registration), M10 (YAML→schema equivalence oracle) |
| TDS-03 + corpus | M05 | parser conformance target; fuzz seed corpus |
| `internal/core` types | M02–M18 | every package imports core, not sdk (F-1 direction) |
| `sdk` alias surface | M08, M15 | public API grows behavior on this frozen shape |
| G1 approval | ALL downstream | the only hard full-stop gate: no code against unapproved schema |

## Critical-path position
Second node; G1 is the single hard full-stop in the plan (IMP §10). M04/M05 fan out from here in parallel with M02 — M01's completion unlocks THREE tracks simultaneously.
