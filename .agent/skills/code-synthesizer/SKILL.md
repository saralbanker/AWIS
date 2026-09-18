---
name: code-synthesizer
description: >
  Use when writing new production code with a testable behavioral contract —
  a function, module, class, or feature whose correctness can be asserted.
  Not for fixing existing code (debugging-master), behavior-preserving
  cleanup (refactoring-specialist), or a purely visual/creative deliverable
  (animation, styling, generative art) where the bar is aesthetic judgment,
  not a pass/fail contract — those don't need a test-first gate.
last_verified: 2026-08-30
kill_condition: n/a — TDD/SOLID/complexity limits are language-agnostic design principles.
---

# Code Synthesizer

## When to use
Writing a new function, module, class, or feature *whose correctness is
testable* — a clear input/output contract, not a visual or creative
deliverable judged by look and feel.

## Anti-trigger
A request to build a visual/creative artifact (an animation, a generative
design, a one-off aesthetic demo) is not this skill, even if it involves
writing code — forcing a test-first gate onto something whose actual
quality bar is aesthetic judgment wastes effort and can crowd out the
polish that's the actual point. Write the code directly; add tests only for
any genuinely logical sub-parts (e.g. a timing/easing calculation), not for
the visual output itself.

## Procedure
1. **Define the contract first**: typed inputs, typed output, the errors it
   can throw. If you can't write the signature, the requirement isn't clear
   enough yet — ask or state the assumption you're making.
2. **Write the test before the implementation** (red → green → refactor):
   happy path, one edge case, one error case, minimum.
3. **Implement the simplest code that passes.** Don't add branches or options
   nothing asked for.
4. **Check complexity**: if a function has more than ~10 decision branches
   (`if`/`else`/`&&`/`||`/loops/ternaries), split it by responsibility —
   each branch is a test case you now owe.
5. **Don't abstract on first or second use.** Wait for a third real
   occurrence before extracting a shared abstraction (rule of three) —
   premature abstractions are harder to remove than the duplication they
   replace.
6. **Run the full suite**, not just the new test, before calling it done.

## Avoid
Silent `undefined`/`null` returns on unexpected input (throw a typed error
instead); one function doing two unrelated things ("and" in its description
is the split point); suppressing type errors instead of fixing the mismatch
at the boundary.

## Checklist
- [ ] Test written before implementation, and it actually failed first
- [ ] Happy path + edge case + error case covered
- [ ] Complexity reasonable (no 15-branch function)
- [ ] Parameters and return values are typed, no unjustified `any`
- [ ] Full test suite passes, not just the new test

See `reference/playbook.md` for the SOLID breakdown, a worked complexity
example, and common failure modes.
