# M05 — Traceability

| Task | Source (verbatim coordinate) | Requirement |
|---|---|---|
| T1 template parser/resolver | TDS-03 §1 (= Finalization B2 Grammar 1, frozen) | FR-WD-05, FR-WD-08 |
| T2 condition parser/evaluator | TDS-03 §2 (= B2 Grammar 2, frozen); null rules verbatim | FR-WD-06, FR-WD-07 |
| T3 Env | M03 HANDOFF variables shape (EDR-007); TDS-03 scope definitions | — |
| T4 validator | PRD §18 "Validation checks (required)"; B2 "Validation at registration time" | FR-WD-03, FR-WD-04 (data half), FR-WD-13 |
| T5 corpus+fuzz | TDS-03 fixture tables (oracle); IMP §25 IR-2 row | AC "full corpus + fuzzing, zero panics" |

## Decisions (EDR)
- **EDR-010** — evaluation semantics not specified by frozen text (B2 names "type coercion" as a
  question but answers only null cases): strict no-coercion model; ordering numeric-only;
  mismatches follow the frozen null-rule pattern (ordering⇒false, ==⇒false, !=⇒true); warning
  transport as returned data (Logger is M06-owned); template stringification of non-string values;
  cycle detection over transition edges with fallback edges counted for reachability only.
  Flagged for G2 review with engine semantics.
- `event` scope enforcement (only in trigger filters) sits in the VALIDATOR, not the parser —
  the corpus contains context-free valid rows using event scope (C1–C9), so the parser must
  accept them; TDS-03's restriction is a field-context rule (T4 check 6/7).
- Handler-ref check at M05 is presence-only; runtime resolvability is registration-time (M08)
  per PRD §18 "resolvable at runtime" + FR-WD-15 "handler existence deferred to registration".

## Deviations
- Named-agent dispatch unavailable mid-session → general-purpose Agent with model pinned +
  identity embedded (standing AEO deviation, recorded each milestone).
