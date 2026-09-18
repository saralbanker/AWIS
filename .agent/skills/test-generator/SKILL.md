---
name: test-generator
description: >
  Use when writing tests for a function, module, or system — new code,
  missing coverage, or a bug that needs a capturing test. Not for debugging
  itself (debugging-master), though write the bug-capturing test if asked
  during a debug session.
last_verified: 2026-08-30
kill_condition: n/a — test pyramid ratios, AAA structure, and boundary-value analysis are general testing method.
---

# Test Generator

## When to use
Writing tests for new or existing code, improving coverage, or capturing a
bug before fixing it.

## Procedure
1. **Respect the test pyramid**: mostly fast, isolated unit tests (no I/O,
   mocked dependencies); fewer integration tests (real components,
   contract-level); a small number of E2E tests covering only critical
   user flows. Inverting this ratio produces a slow, flaky suite people
   stop running.
2. **Structure every test as Arrange-Act-Assert**, one behavior and
   preferably one assertion per test — multiple assertions means multiple
   failure reasons tangled into one signal.
3. **Cover boundaries, not just the happy path**: the typical case, the
   min/max valid boundary, just outside each boundary, and null/undefined/
   empty inputs.
4. **Name tests by behavior**: `should_<expected>_when_<condition>`. The
   name is the documentation a failing CI run shows first.
5. **Run the suite** before calling it done — a test that hasn't been run
   isn't verified, it's a guess.

## Avoid
Testing implementation details instead of input→output behavior (makes
tests brittle on refactors); giving tests names like `test1` or `it works`;
mocking so much that a test no longer proves the real code works.

## Checklist
- [ ] Happy path covered
- [ ] Boundary values covered (min, max, just outside each)
- [ ] Null/undefined/empty inputs covered
- [ ] Every documented error/throw path has a test
- [ ] All tests named `should_<behavior>_when_<condition>` or equivalent
- [ ] Full suite passes, tests are independent of execution order

See `reference/playbook.md` for the AAA example and boundary-value
worked case.
