# M01 — AI Execution Context
**Model allocation (IMP §28):** **Opus + Human sign-off** — maximum architectural sensitivity; the irreversible artifact; deep cross-document verification. This is NOT a Sonnet milestone.

## Session loading order (≈15–25k tokens; heaviest legitimate load in the plan)
1. `docs/00-foundation/README.md` (authority order + contradiction protocol)
2. This module's `IMPLEMENTATION_SPEC.md` + `VALIDATION_CHECKLIST.md`
3. **Blueprint §6 and §9 verbatim** (the texts being transcribed)
4. **Finalization Blocker 2 verbatim** (grammar source)
5. Verification report F-1 section (the alias pattern)
Load PRD/Constitution sections only on demand via `docs/07-indices/cross-reference-index.md`.

## Hard constraints
- Transcription discipline: when the frozen text names a field, that exact name ships. Style preferences lose to the corpus, always.
- Any ambiguity or gap in the frozen texts → STOP; record a CONTRA-style entry; ask the human. Silent resolution at M01 poisons every downstream milestone (IR-1).
- F-1 is mandatory: canonical types in `internal/core`, aliases in `sdk`. Do not "simplify" to direct definitions in `sdk` — that recreates the import cycle at M06.
- Fixture corpus BEFORE any parser thinking; the corpus is grammar-derived, not implementation-derived (IR-2).
- Nothing merges without recorded human G1 approval. There is no autonomous path through this milestone.

## Escalate to human when
- Blueprint §9's event enumeration and any other frozen text disagree on an event type or field.
- A frozen interface shape cannot be expressed as a Go type alias without alteration.
- Always: for the final G1 verdict (mandatory, not conditional).
