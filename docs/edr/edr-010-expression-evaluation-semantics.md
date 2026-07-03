# EDR-010 — Expression Evaluation & Validation Semantics Beyond the Frozen Text

**Status:** Reversible — pre-G2 (implementation decisions; flagged for G2 review with engine semantics)
**Authored:** M05 (2026-07-03)
**Coordinates:** TDS-03 (frozen grammars + null rules) · Finalization B2 (names "type coercion" as an open question its decision section does not answer) · PRD FR-WD-05/06/07 · PRD §18 · `internal/expr` · `internal/validate`
**Downstream obligation:** M06 builds Env from projection variables (EDR-007 shape) + step statuses and LOGS the Warnings Resolve returns (FR-WD-05 "a warning is logged"); M10 renders validate.Issue data into the PRD §18 output format.

---

## 1. Condition evaluation — strict, no coercion

The frozen text specifies only the null rules (ordering vs null ⇒ false; null == null ⇒ true).
Everything else below follows the same "comparisons that don't make sense are false" pattern:

| Left value ↓ / literal → | number | string | bool | null |
|---|---|---|---|---|
| number | numeric, all 6 ops (float64-unified) | == false / != true / ordering false | == false / != true / ordering false | == false / != true (frozen: null≠non-null) |
| string | mismatch | ==/!= lexical; ordering **false** | mismatch | same |
| bool | mismatch | mismatch | ==/!= ; ordering **false** | same |
| null/missing | ordering **false** (frozen); == false / != true | same | same | == **true** / != false (frozen) |

mismatch = `==` false, `!=` true, ordering false. `Eval` never errors on types.
An explicitly-nil value present in the environment is null for conditions.

## 2. Template resolution (FR-WD-05 concretions)

- Resolve returns `(string, []Warning)` — never an error. Warnings are DATA; this package owns no
  logger (Logger shape is M06's); the engine logs them.
- "Not yet completed": a `steps.<id>.outputs.…` ref yields ""+Warning when StepStatus[id] exists
  and ≠ "completed" — even if outputs are present. A `steps.<id>.status` read is never gated.
- Explicitly-nil present value stringifies to "" with NO warning (the path exists).
- Non-string values stringify: bool → true/false; numbers → strconv shortest round-trip;
  composites → `fmt %v`.
- Whitespace: ASCII spaces tolerated inside `{{ }}` around the path-ref (YAML-author convention;
  contradicts no corpus row). A `}}` with no opener is literal text; an unclosed `{{` is a
  ParseError at the opener.

## 3. Validator graph semantics

- **Orphan reachability** follows transition edges AND fallback edges (a fallback-only step is not
  orphaned — it is the §17/Blueprint fallback path).
- **Cycle detection** runs over transition edges ONLY: a fallback join cannot re-enter the forward
  path (FR-WD-13's intent is forward-progress termination).
- **event scope** is enforced as a field-context rule in the validator, not the parser: the frozen
  corpus (C1–C9) contains context-free valid event-scoped rows, so parsers accept them; TDS-03's
  "only valid in Trigger filter fields" produces `condition-event-scope` Issues for transition
  conditions.
- Handler refs: presence-only at M05; resolvability is registration-time (M08) per PRD §18
  "resolvable at runtime" + FR-WD-15.

## Why not a CONTRA

No frozen format, grammar production, prohibited-list entry, or corpus verdict is touched. B2
explicitly left type coercion unanswered; these are the conservative fills, pattern-consistent
with the frozen null rules, all reversible before G2.
