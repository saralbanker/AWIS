# Agent Baseline

Default working principles for any AI coding agent in this repository,
regardless of which tool is reading this file. Skills under `.agent/skills/`
extend this for specific tasks; they do not override it.

## Operating stance

- Act on reasonable defaults. Ask only when a choice is genuinely
  irreversible, materially ambiguous, or the cost of guessing wrong is high.
  Don't ask permission for things inferable from the code, the request, or
  convention.
- State assumptions instead of asking about them, so they're visible and
  correctable.
- Prefer the smallest change that correctly solves the stated problem. Don't
  refactor, add abstractions, or "improve" adjacent code that wasn't asked
  about.
- Match response tone and length to the question's complexity. No fixed
  word/line budgets — a one-line question deserves a one-line answer, a
  multi-file change deserves a real explanation.

## Working with code

- Match the surrounding code's existing conventions before writing new code —
  don't impose a personal style.
- Verify claims about the codebase (a function exists, a test passes, a file
  is unused) by reading or running it — not from memory or a filename guess.
- When a task involves a real tradeoff (performance vs. readability,
  abstraction vs. duplication), state which one you chose and why, in one
  line. Don't silently pick one and move on.

## Boundaries

- Confirm before anything hard to reverse or visible outside this workspace:
  force-pushes, deleting things you didn't create this session, sending
  messages, publishing packages, modifying shared infrastructure.
- Everything else — running tests, editing files, local commits when asked —
  proceed without asking.

## Skills

- `.agent/skills/*/SKILL.md` describes when to engage a capability and the
  procedure to follow. Read a skill's `reference/` files only when its
  SKILL.md tells you to — don't preload them.
- If a recurring, real need has no matching skill, use `skill-forge` to
  create a narrow, dated one rather than improvising a permanent habit that
  nobody reviews later.

## Staleness

- Skills carry `last_verified` and `kill_condition` metadata. If you notice a
  `kill_condition` has become true — the fact it depended on has changed —
  say so instead of silently continuing to follow stale guidance.
