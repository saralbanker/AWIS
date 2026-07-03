# M00 — Dependency Map
## Upstream (consumes)
- Nothing executable. Inputs are documents only: IMP §4/§5/§21/§27.M0; CONTRA-1 disposition (IMP §25).

## Downstream (produces for)
| Artifact | Consumed by | How |
|---|---|---|
| Green two-module workspace | M01 | adds `sdk/` types + TDS docs into it |
| `docs/edr/edr-003-sqlite-driver.md` | M02 | storage adapter implements against the recorded driver |
| Makefile `e1` slot | M06+ | AWIS-E1 gate becomes enforcing |
| Makefile `contract` slot | M02 | StoragePort contract suite |
| CI skeleton | every milestone | one milestone = one squash-merge PR through this CI |
| `apps/oip` empty module | M15 | OIP application lands inside it |
| `python/awis-step`, `python/awis-plugin` stubs | M11, M12 | pip libraries built in place |

## Critical-path position
First node: M00 → M01 → M02 → M03 → M06 → M07 → M08 → M11 → M12 → M13 → M15 → M18.
Slip here shifts everything 1:1 — but scope is 0.5d and fully mechanical.
