# M02 — Traceability
| Task / artifact | Canonical source | Coordinate |
|---|---|---|
| SQLite adapter + WAL, pure-Go driver | Blueprint §20; PRD deps; edr-003 | storage stack; NFR portability |
| Migration runner (embedded, sequential, forward-only, auto on startup) | Blueprint §20; PRD | Migration Strategy; FR-ST-06 |
| Migration 0001 M02 portions (events/definitions/cache + schema_version table) | IMP §14; Blueprint §20 SQL | migration table row 0001 |
| `execution_events` 9-column DDL | TDS-01 §1 (G1 ADJ-1) | envelope frozen at G1 |
| Append-only, no update/delete | PRD | FR-ST-01 |
| Sequence monotonicity ENFORCED in append tx; assignment stays with engine | TDS-01 §3; **EDR-005** (additive, reversible) | monotonic-per-instance invariant |
| Namespace predicate on namespace-carrying queries | PRD | NFR-S-04 |
| Registry immutability ((id,version) re-register fails) | TDS-02 §5; IMP §27.M3 AC (pulled to registry site) | semver immutability |
| Cache TTL via injectable clock | Blueprint §20 DDL; IMP §3 | determinism rule |
| StateStore stubs until M03 | IMP §27.M2 scope ("events/definitions/cache portions") | M03 boundary |
| Contract suite as V2-Postgres reuse artifact | IMP §13.1, §27.M2 DoD | contract tests |
| Crash durability | PRD; IMP §20 | NFR-R-02; M2 checkpoint |
| 100K append informational (binding targets NFR-P-05/06 land M03/M14) | PRD §NFR-P | performance table |

**Contradictions touched:**
- **CONTRA-5 addendum (recorded here at M02 materialization):** PRD FR-WE-10 (L401) also states `WorkflowCancelled {reason, cancelled_at}` — a third coordinate for the same conflict adjudicated at G1 (ADJ-2). The G1 disposition covers it identically: canonical payload `{reason}`; `cancelled_at` derived from envelope `emitted_at`. M06 (which implements FR-WE-10) inherits this disposition; no new decision required.
- **EDR-005 (new, additive):** frozen texts do not state where `sequence_num` is assigned. Disposition: storage ENFORCES strictly-increasing-per-instance inside the append transaction; assignment is the engine's (M06). Reversible pre-tag; recorded in `docs/edr/edr-005-sequence-assignment.md`.
