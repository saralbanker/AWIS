# M05 — AI Execution Context

**Model allocation (IMP §28):** **Opus** — "Parser correctness against formal grammars; high
verification burden (fuzzing, corpus)." Both builder cards run on Opus (no downward substitution,
EDR-004). Verifier card: Sonnet (verification is checklist-driven).

**Builder cards:** M05-C1 (Opus: both grammars + Env + corpus/fuzz tests), M05-C2 (Opus:
WorkflowValidator + its tests).
**Verifier card:** M05-V1 (Sonnet, clean clone, read-only).

**Context per card:** this module's files + docs/EXPRESSION_GRAMMARS.md (TDS-03, frozen) +
`internal/expr/corpus/corpus.go` + (C2 only) `internal/core/{workflow,step,scalars}.go` +
PRD §18 checklist excerpt embedded in the card. Never the IMP.

**Frozen constraints binding every card:**
- TDS-03 grammars + prohibited lists + null rules are FROZEN (G1). The 42-row corpus is the
  oracle: parsers conform to it; the corpus is never edited.
- Identifiers contain hyphens as single tokens (`steps.draft-entry.status`) — the lexer must not
  treat `-` as an operator inside identifiers.
- No external expression library (B2 rejected Option D). No new dependencies.
- Toolchain: `export PATH=$HOME/toolchains/go/bin:$HOME/toolchains/bin:$PATH`; no sudo.
- Builders do NOT commit; CE reviews and commits.
- Frozen-text gap/conflict ⇒ STOP and report (CONTRA protocol); EDR-010 already covers the known
  unspecified evaluation semantics — do not invent others silently.
