# M01 → M02/M04/M05 Handoff
**Status: PENDING EXECUTION** — *Guaranteed outputs* are the contract; *Actuals* filled at merge.

## Guaranteed outputs (contract)
- TDS-01/02/03 frozen under G1 approval — downstream code treats them as Tier-0-derived normative text
- `internal/core` canonical types + `sdk` alias surface, compiling, logic-free
- Grammar conformance corpus as compiling Go test data

## What downstream may assume
- **M02:** EventLog envelope + workflow serialization are final; migration 0001 columns derive from TDS-01/02 mechanically.
- **M04:** IntelligencePort type shape is frozen in `internal/core`; adapter work is implementation-only.
- **M05:** the parser target is TDS-03's corpus; grammar questions are CLOSED — a parser that disagrees with the corpus is wrong by definition.
- **M08:** the public surface already exists as aliases; M08 adds behavior (registration, options), never reshapes types.

## Known limitations (by design)
- No storage, no parsing, no engine. Types have no methods. `apps/oip` still empty.

## Actuals (fill at merge)
- G1 sign-off: (PR link / date) _
- Event types enumerated: _ (count)
- CONTRA entries raised during transcription: _
- Deviations: _
