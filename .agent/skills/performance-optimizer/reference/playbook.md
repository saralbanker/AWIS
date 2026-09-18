# Performance Optimizer — Playbook

## Big O, at a glance (10K items)

| Operation | Complexity | Ops |
|---|---|---|
| Hash map lookup | O(1) | 1 |
| Binary search | O(log n) | 14 |
| Array scan | O(n) | 10,000 |
| Nested loops | O(n²) | 100,000,000 |

```typescript
// O(n²) — a .find() inside a loop
users.forEach(user => { const order = orders.find(o => o.userId === user.id); });

// O(n) — build the lookup once
const orderMap = new Map(orders.map(o => [o.userId, o]));
users.forEach(user => { const order = orderMap.get(user.id); });
```

## Baseline + profile by bound

**CPU-bound:**
```bash
node --prof app.js
node --prof-process isolate-*.log > profile.txt   # find hottest functions
python -m cProfile script.py | sort -k cumtime
perf record -g ./program && perf report
```

**Memory (leak suspected):** take a heap snapshot, repeat the suspect
operation N times, take another snapshot, diff. Growing objects usually
mean: event listeners never removed, closures holding large objects, an
unbounded cache, or a global collection that's never cleared.

**I/O-bound (DB/API/filesystem):**
```bash
EXPLAIN ANALYZE <query>          # look for Seq Scan vs Index Scan
```
Check for N+1: a loop issuing one query per item instead of a single
batched `WHERE id IN (...)`.

**Network-bound:** measure which external call is slowest; parallelize
independent calls (`Promise.all`); cache results with a sane TTL.

## Regression test pattern

```typescript
it('should_complete_bulk_import_under_500ms', async () => {
  const start = performance.now();
  await bulkImport(testData10k);
  expect(performance.now() - start).toBeLessThan(500);
});
```

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| Optimization made no measurable difference | wrong bottleneck targeted | re-profile; don't trust the first guess twice |
| Leak invisible in heap snapshots | leak is in native code / external lib | isolate: does it leak without the library? |
| Query still slow after adding an index | planner isn't using it | `EXPLAIN ANALYZE` — check for `Seq Scan` vs `Index Scan` |
| Benchmark shows high variance | other load on the system, GC pauses | run 3x, take the median, warm up before measuring |
