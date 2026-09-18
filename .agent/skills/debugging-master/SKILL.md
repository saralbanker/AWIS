---
name: debugging-master
description: >
  Use when a bug, test failure, or unexpected behavior needs root-cause
  diagnosis rather than a guessed fix — especially after a first fix attempt
  didn't stick, or the stack trace doesn't point at the real cause.
last_verified: 2026-08-30
kill_condition: n/a — general reasoning procedure, not tied to a tool/framework fact.
---

# Debugging Master

## When to use
Before patching symptoms: intermittent failures, "it worked yesterday,"
errors whose trace doesn't match the real cause, or a bug that survived a
first guess-and-fix.

## Procedure
1. **Reproduce first.** Get a minimal, reliable repro before touching code.
   Can't reproduce it? Say so and gather more signal (exact input, logs,
   environment) — don't guess.
2. **Localize by evidence, not suspicion.** Bisect: targeted logging, or
   disabling half of the suspect path, to narrow which component is
   actually wrong.
3. **One hypothesis at a time.** State it, predict what you'd observe if
   it's true, then check. Discard and move on if disproven — don't stack
   unverified assumptions.
4. **Fix the cause, not the symptom.** A null-check/retry/try-catch with no
   explanation of *why* the bad state occurred isn't a fix yet.
5. **Verify with the original repro**, then check the codebase for the same
   pattern elsewhere — root causes are often duplicated.

## Avoid
Shotgun-debugging (changing several things, re-running until green); fixing
the first plausible-looking thing without confirming it actually triggered
on this run; silently swallowing the error instead of understanding it.

## Checklist
- [ ] Root cause identified — not the symptom ("undefined" is a symptom;
      "missing null check on a DB query that can return no rows" is a cause)
- [ ] A hypothesis was stated and checked before code was changed
- [ ] A test captures the bug and fails without the fix, passes with it
- [ ] Full suite still passes; same pattern checked elsewhere in the codebase

See `reference/hard-cases.md` for flaky/concurrency bugs, prod-only repros,
and multi-service tracing.
