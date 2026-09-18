# Annotated example: sql-query-optimizer

> A worked comparison of a good skill in this kit's lean format vs. the old
> heavy format this kit was rebuilt away from. Read this when quality is
> uncertain.

## The good version

```yaml
---
name: sql-query-optimizer
description: >
  Use when a specific SQL query is slow and needs evidence-based
  optimization — not for general schema design or ORM selection.
last_verified: 2026-08-30
kill_condition: n/a — EXPLAIN-plan-driven optimization is DB-engine-general, not version-pinned.
---
```
Good: `description` states when to use it *and* implies when not to
("general schema design" is a different question). `kill_condition` is
honest that this doesn't really go stale — that's a fine answer when true,
better than inventing a fake expiry condition.

```markdown
# SQL Query Optimizer

## When to use
A specific query is measurably slow and the fix isn't obvious from reading it.

## Procedure
1. Get the execution plan (`EXPLAIN ANALYZE <query>`) before changing anything.
2. Find the most expensive node in the plan — usually a `Seq Scan` on a
   large table, or a nested loop over unindexed data.
3. Match the fix to the finding: missing index → add one on the filtered/
   joined column; wrong join order → check planner statistics are current
   (`ANALYZE <table>`); `SELECT *` pulling unneeded columns → select only
   what's used.
4. Re-run `EXPLAIN ANALYZE` and compare — confirm the expensive node is
   gone, not just that the query "feels faster."
5. Check the fix didn't just move the cost to write time (an index speeds
   reads but costs every insert/update) — worth it only if reads dominate.

## Avoid
Adding an index without checking the planner actually uses it; guessing at
a rewrite without an EXPLAIN before/after to prove it helped.

## Checklist
- [ ] Before/after EXPLAIN plans both captured
- [ ] The specific expensive node is gone or measurably cheaper
- [ ] New indexes checked against write-cost tradeoff

See `reference/playbook.md` for engine-specific EXPLAIN syntax.
```
Good: ~25 lines, every step is a concrete action, nothing here needs
updating when a framework ships a new version — it's about how query
planners work, which doesn't change year to year.

## The anti-pattern this kit removed

Earlier versions of this kit generated skills like this for *every* domain,
including this one:

- A **9-field frontmatter** (`version`, `token_budget: 4000`,
  `tools_required: ["bash", "read_file", "write_file"]`,
  `output_contract`, `works_with`) — none of these fields are read by any
  harness. They looked authoritative and did nothing.
- A **mandatory Mermaid "Reasoning Graph"** restating the same branching
  logic already implied by the numbered steps — a second copy of the same
  information, paid for in tokens every time the skill loaded.
- **A 200+ line body** with a "Core Concepts" essay, worked code examples,
  a failure-mode table, and a 12-item validation gate, all loaded on every
  activation regardless of which part was actually needed.
- Real version pins buried in prose ("as of 2025...") that were already
  stale a year later, with no `last_verified`/`kill_condition` anywhere to
  flag it.

None of that made the guidance better — it made it more expensive to load
and harder to keep current. If you're forging a skill and it's creeping
past ~100 lines in the body, that's the signal to move the excess into
`reference/`, not to keep going.
