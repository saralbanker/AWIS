---
name: refactoring-specialist
description: >
  Use when improving existing code quality without changing behavior — code
  smells, technical debt, hard-to-read code. Not for adding features
  (code-synthesizer) or fixing bugs that change behavior (debugging-master).
last_verified: 2026-08-30
kill_condition: n/a — Fowler's smell taxonomy and behavior-preservation discipline are language-agnostic.
---

# Refactoring Specialist

## When to use
Cleaning up, reducing technical debt, or improving readability without
changing what the code does.

## Procedure
1. **Confirm tests exist and pass before touching anything.** If they don't
   exist, write characterization tests first (tests that document current
   behavior, even if that behavior is imperfect) — refactoring without a
   safety net is a rewrite wearing a disguise.
2. **Identify one smell** (see reference for the taxonomy) with concrete
   evidence — a line count, a duplicate count, a parameter count — not a
   feeling.
3. **Apply exactly one refactoring**, run the tests, and only then move on.
   If a test fails, the behavior changed — revert, don't debug forward.
4. **Name extracted functions/classes by intent, not mechanism**
   (`calculateDiscount`, not `processStep2`).
5. **Commit each refactoring separately** so a regression is trivial to
   isolate and revert.
6. **Repeat** until the code meets a reasonable bar or the time box ends —
   don't try to fix everything in one pass.

## Avoid
Refactoring code with no test coverage and no plan to add it; bundling
several transformations into one change; treating "smaller diff" as the
goal instead of "same behavior, clearer code" — duplication is cheaper than
the wrong abstraction.

## Checklist
- [ ] Tests existed or were written before refactoring started
- [ ] Tests pass after every individual step, not just at the end
- [ ] Each transformation is its own commit
- [ ] Complexity/duplication measurably improved
- [ ] No behavior change — tests are the proof, not an assertion

See `reference/playbook.md` for the smell-detection table and thresholds.
