---
name: performance-optimizer
description: >
  Use when measuring and improving the speed or resource usage of code or a
  system — slow responses, high latency, memory growth. Not for functional
  bugs (debugging-master) or structural problems (architecture-analyst).
last_verified: 2026-08-30
kill_condition: n/a — measure-first workflow and complexity analysis are timeless; only the specific profiler commands in the playbook are tool-version-bound.
---

# Performance Optimizer

## When to use
Slow responses, high latency, memory that grows over time, or a system not
meeting its performance target.

## Procedure
1. **Measure a baseline before touching anything.** Without a "before"
   number you can't prove an optimization worked — the baseline is the
   test.
2. **Profile, don't guess.** Intuition about where time goes is wrong often
   enough that it's not worth trusting; use an actual profiler (see
   reference for per-language commands) and trust its output over your
   hunch.
3. **Apply Amdahl's Law**: optimizing something that's 5% of total time
   can't help by more than 5%, however hard you optimize it. Fix the
   largest contributor first.
4. **Look for the classic wins**: O(n²) loops collapsible to O(n) with a
   map/set, N+1 queries collapsible to one batched query, missing indexes,
   sequential calls that could run in parallel.
5. **Measure again with the same method as step 1.** If the improvement is
   marginal, you optimized the wrong thing — re-profile.
6. **Add a regression test** with a threshold so a future change that
   reintroduces the slowdown gets caught.

## Avoid
Optimizing before measuring; declaring victory without a comparable
after-number; trading away readability for a speedup that turned out to be
under 20%.

## Checklist
- [ ] Baseline measured before any change
- [ ] Bottleneck identified from profiler output, not intuition
- [ ] Bottleneck confirmed to be a meaningful share of total time (Amdahl)
- [ ] After-measurement shows real improvement by the same method
- [ ] No functional regression; a performance regression test was added

See `reference/playbook.md` for profiler commands by bound (CPU/memory/I/O/
network) and the Big-O quick reference.
