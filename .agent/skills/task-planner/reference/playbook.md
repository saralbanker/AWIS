# Task Planner — Playbook

## Worked dependency graph

```
Task 1 (DB schema)      → Task 3 (API routes) → Task 5 (Frontend integration)
Task 2 (Auth middleware) → Task 4 (Protected routes)

Critical path: 1 → 3 → 5 (longest chain, 3 tasks)
Parallel: Task 2 can proceed alongside Task 1
```

## Acceptance criteria format

```
Task: Add JWT verification middleware
Given: a request hits any /api/* route
When: the Authorization header is missing or contains an invalid JWT
Then: the server returns 401 with body { error: "Unauthorized" }
```

## Sizing without false precision
Use T-shirt sizes (S = under 2h, M = 2-8h, L = over 8h) rather than hour
estimates for anything with real uncertainty — a precise-looking number for
an unknown task is misleading, not more useful.

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| Plan never gets executed | tasks too large, no clear entry point | keep splitting until the first task is under ~4 hours |
| Circular dependency found | shared prerequisite wasn't identified | extract it into its own task, have both depend on it |
| Scope creep mid-task | acceptance criteria were vague or missing | rewrite the AC before continuing, not after |
| Estimate wildly off | unknown complexity wasn't flagged | mark it explicitly as high-risk/unknown, size with S/M/L not hours |
