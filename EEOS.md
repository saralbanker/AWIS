# EEOS — Engineering Execution Operating System (Bootstrap)
**CANONICAL · v2.0 · frozen for Baseline V1.** Normative spec + rationale: `AWIS_EEOS.md`.
Roles & inter-role protocol: `AWIS_ENGINEERING_ORGANIZATION.md` (AEO). This file is the single
entry point for every execution session. It contains ZERO milestone-specific content.

## S — Bootstrap (every session, in this order, nothing else)
```
S1  Read docs/05-implementation/STATE.md      (the execution ledger — all live state)
S2  Read docs/00-foundation/README.md         (non-negotiables + contradiction protocol)
S3  Act per the phase table below. Load NOTHING else until the matching row says so.
```

## Phase machine (one milestone = one pass; STATE carries the current phase)
```
A-INIT   [FABLE]   branch · materialize module (IKB §3/§4) · write ALL cards READY · STATE→B
B-BUILD  [dispatch] per STATE NEXT: dispatch card → report → STATE write → repeat
C-VERIFY [dispatch] dispatch MXX-V1 to awis-verifier on clean tree · all-✅ → STATE→D
D-CLOSE  [FABLE]   semantic review of full diff · HANDOFF actuals · PR + evidence ·
                   (non-gated boundary: also run A-INIT of the next milestone, cards READY)
E-MERGE  [HUMAN]   founder reviews PR, gate verdict where due, squash-merges · STATE→next
```

## Phase-resume table (STATE row → additionally load → action)
| STATE says | Load | Action |
|---|---|---|
| `IDLE` / milestone not entered | its `README.md` + IKB §3 + §4 row | Phase A-INIT (Fable-grade only; otherwise say so and stop) |
| `A-INIT` incomplete | partial module files | resume at first missing IKB §3 step; cut remaining cards |
| `B-BUILD` | the one card `NEXT:` names | dispatch it (prompt P1) · on report: write STATE, next card |
| `B-BUILD`, card `DISPATCHED` | — | prior run died: re-dispatch same card with P1+P2 (once; 2nd death → STOPPED, CE audit) |
| `B-BUILD`, `BLOCKERS: E1…` | STOP report + cited coordinates | Fable wake: adjudicate per AEO §10; E2 → CONTRA → human |
| `C-VERIFY` | `cards/MXX-V1*.md` | dispatch verifier · ❌ → revision cards → B |
| `D-CLOSE` | full milestone diff + module | Fable wake: review, evidence, PR, gate brief; fuse next A-INIT if boundary non-gated |
| `E-MERGE` | — | blocked on human; report what awaits the founder; do nothing |
| ledger contradicts repo | — | STOP — Fable wake; never "fix" STATE to match a guess |

## P1 — Dispatch prompt (frozen; identical for every card)
```
Execute the Execution Card at <card path>.
Read that file first. Then read ONLY the files it names, within its stated
context budget. Honor every constraint in your agent identity. Work only on
the branch the card names. Return your report per your identity's contract.
Commit exactly once, when every card output is complete and its tests are
green; commit message = the card id + one line.
```

## P2 — Salvage preamble (frozen; append to P1 only when STATE already shows the card DISPATCHED)
```
A prior run of this card was interrupted. Uncommitted work may exist on the
branch (git status / git diff). Before writing anything: audit what you find
against this card's OUTPUTS and ACCEPTANCE. Keep files that pass, complete or
rewrite the rest. List salvaged-vs-rewritten in your report's DEVIATIONS.
```

## Rules (each one sentence; authority in parentheses)
1. **Ledger law:** no phase or card transition is real until written to STATE.md (EEOS §3.3).
2. **Card-primary context:** a worker loads exactly one card + the files it names — never the
   milestone module, never the corpus (EEOS §4.3; AEO §19).
3. **Cards are immutable once READY**; defects get a new revision file `MXX-Cnr.md`, never an
   edit; interruptions get P2, never an authored continuation (EEOS §4.2, §15 D2).
4. **Status lives only in STATE**; durable outcomes go to the module `TRACEABILITY.md`;
   engineering state is git — branch, one commit per card (DONE ⇔ sha), working tree (§15 D2).
5. **Verification is independent:** MXX-V1 cites `docs/05-implementation/V-COMMON.md` + milestone
   checkpoints; the verifier never sees implementer reasoning (AEO §12).
6. **Merges are human-only**; PR carries diff, verification report, checkpoint transcript,
   DEVIATIONS, STATE diff, and the `cards/` files (AEO §13; EEOS §10).
7. **Escalation is the AEO §10 ladder** — E1 STOP to CE; frozen-text conflict is CONTRA (E2),
   never resolved inline; suspected frozen-format defect is E3.
8. **Model discipline:** the card's DISPATCH row binds the agent (IMP §28 via AEO §6/§20);
   substitution is upward-only, by CE.
9. **Cache stability:** this file, the four identities, V-COMMON.md, and READY cards are
   byte-frozen during a milestone (EEOS §15 D5).
10. **Fable wakes only for:** A-INIT · unresolvable E1/contradiction/ledger-repo conflict ·
    deadlock (STOP storm) · D-CLOSE · gate briefs/merge recommendation · CONTRA (EEOS §8).
    Any other Fable turn is a defect — log one line in STATE `LAST:`.

## Lookup discipline
A question about the frozen design → `docs/07-indices/cross-reference-index.md` → one coordinate
→ load only that section. Browsing canonical documents "for context" is a protocol violation.
