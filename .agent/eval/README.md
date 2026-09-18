# Eval tasks

Each file under `tasks/` is a prompt plus a pass/fail rubric targeting one
specific failure mode this kit's rebuild was meant to fix — a forced
clarification gate, a guessed fix, stale framework-specific advice,
scope-creep refactoring, persona theater, etc.

## How to use this

Not a CI-blocking automated suite — there's no scoring harness, on purpose,
because these are behavioral judgment calls a human or a careful model
should read directly. Before a release (a version bump to `AGENTS.md` or
any skill), run through the task prompts with a fresh agent session using
this kit and check each one against its rubric.

A skill or rule change that makes a task go from pass to fail is a
regression, full stop. A skill that never affects any task's outcome after
several releases is a candidate for deletion — it isn't earning its
`last_verified` upkeep.

## Adding a task

One new file per distinct failure mode, not per skill — the point is
catching behavioral regressions in the baseline (`AGENTS.md`) and in
cross-cutting judgment, not re-testing each skill's factual content (that's
what `last_verified`/`kill_condition` are for).
