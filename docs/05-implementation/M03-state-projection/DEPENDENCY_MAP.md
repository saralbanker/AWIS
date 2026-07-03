# M03 — Dependency Map
## Upstream (consumes)
| Artifact | From | How |
|---|---|---|
| EventLog append/read (contract-locked) | M02 | rebuild replays via ReadEvents, never raw SQL |
| Migration runner + 0001 | M02 | fold-forward adds instances/claims |
| storagetest suite pattern | M02 | extended to all 12 methods |
| TDS-01 replay statements | M01 | projection ground truth |
| ADJ-5 enum amendment | M03 entry (founder) | compensation_failed projection |

## Downstream (produces for)
| Artifact | Consumed by | How |
|---|---|---|
| Full StoragePort (12/12 live) | M06 engine (scan/claim/settle), M07 (signal tx), M08 (runner API) | the persistence layer, complete |
| step_claims at-most-once | M06 CLAIM stage | §8 step 3 mechanism (CONTRA-6) |
| RebuildState library | M14/M17 `awis rebuild-state`; every post-migration re-verify (IMP §14) | recovery path |
| Projection rules (EDR-007) | M06 — engine transitions MUST mirror them | de-facto projection spec until G2 |
| G2 docket items (definition-identity gap; in_flight reading) | G2 review after M07 | recorded, not resolved |

## Critical-path position
M00 → M01 → M02 → **M03** → M06 → … M04/M05 (Tracks B/C) remain independent and unstarted.
