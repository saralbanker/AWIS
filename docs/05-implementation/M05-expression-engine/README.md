# M05 — Expression Engine & Validator
**Status:** Partitioned — materialize at entry · **Effort:** 2d · **Window:** Weeks 1–2 (Track C)
**Objective:** Hand-written implementations of BOTH Finalization Blocker-2 grammars (template resolver; condition parser/evaluator: null semantics, hyphenated identifiers, single-quoted strings) + WorkflowValidator (orphans, cycles, handler/fallback/transition refs, grammar validation, trigger filters, intelligence field checks — FR-WD-03).
**Depends on:** M01 (TDS-03 + conformance corpus) · **Blocks:** M06, M10
**Primary sources:** IMP §27.M5; Finalization Blocker 2 (normative grammar text); TDS-03 · **Compilation spec:** IKB §4/M05
**Key ACs:** full corpus passes (every production + every prohibited construct rejected with precise position); fuzzing zero panics; missing template path → `""` + warning (FR-WD-05). NO external expression library (Blocker 2 rejected Option D).
