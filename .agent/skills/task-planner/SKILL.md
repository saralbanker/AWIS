---
name: task-planner
description: >
  Use when breaking down a complex, multi-step goal into ordered subtasks
  with dependencies and acceptance criteria. Not for a single, already-atomic
  task ("add a button that does X") — that doesn't need decomposition.
last_verified: 2026-08-30
kill_condition: n/a — DAG decomposition and critical-path reasoning are general planning method, not tool-specific.
---

# Task Planner

## When to use
A feature, project, or goal that clearly requires multiple steps, or a
request to "break this down" / "where do I start."

## Procedure
1. **Check atomicity.** A task is atomic when it has exactly one
   unambiguous definition of done. If the DoD needs an "and," split it.
2. **Map dependencies as a DAG.** For each task, ask what must exist before
   it can start. If you find a cycle (A needs B, B needs A), extract the
   shared prerequisite into its own task.
3. **Find the critical path** — the longest dependency chain. These tasks
   set the floor for delivery time and should never wait on non-critical
   work.
4. **Order by risk, not convenience.** Tackle the highest-uncertainty task
   first when dependencies allow — discovering a blocker on day one is
   recoverable; discovering it on day seven is a reset.
5. **Write acceptance criteria per task** as Given/When/Then. No AC means
   no way to verify "done," which is how scope creeps.

## Avoid
Decomposing a task that's already atomic; leaving a circular dependency
unresolved; vague acceptance criteria like "it should work."

## Checklist
- [ ] Every task has exactly one definition of done
- [ ] No circular dependencies
- [ ] Critical path identified
- [ ] At least one high-risk/unknown task is flagged for early execution
- [ ] Every task has Given/When/Then acceptance criteria
- [ ] The first action is obvious enough to start immediately

See `reference/playbook.md` for a worked dependency-graph example and the
acceptance-criteria format.
