# Debugging Master — Hard Cases

## Heisenbugs (disappear when observed)
A bug that vanishes when you add a `console.log` or attach a debugger is
almost always a race condition or async-ordering issue — the observation
itself changed the timing. Don't rely on interactive breakpoints for these;
add **deterministic** sequence IDs/timestamps to logs instead, reproduce the
exact ordering, then reason about it.

```bash
grep -rn "setTimeout\|setInterval\|Promise.all\|race" src/
```

## Flaky / intermittent failures
Assume async/timing first, not "flaky test, just retry." Look for unhandled
promise rejections, shared mutable state between tests, or genuine race
conditions. Reproduce the failure 3 times with the same input before
declaring a fix — a fix that "seems to work" after one clean run is not
verified.

## Prod-only repros ("works locally, breaks in prod")
The bug is almost always an environment difference, not different code.
Diff configuration before diffing logic:
```bash
diff <(env | sort) <(ssh prod 'env | sort')
node -v   # compare local vs. prod runtime version
```

## Multi-service / distributed bugs
Trace the request pipeline forward from the entry point (load balancer →
gateway → auth → handler → DB → response) and find the first point where
observed state diverges from expected state — that divergence point, not
the service that surfaced the error, is where to focus. When multiple
services fail simultaneously, check shared infrastructure (DB, cache, DNS,
certs) before investigating each service individually.

## Common failure-mode → recovery map

| Symptom | Likely cause | What to do |
|---|---|---|
| `Cannot read properties of undefined` | object null/undefined at call site | trace where it should have been set; add a guard at the source, not the call site |
| `ECONNREFUSED` | dependent service not running / wrong port | check the service is actually up (`docker ps`, health endpoint) |
| Unhandled promise rejection | missing `.catch()`/`try-catch` on an `await` | wrap the await, don't swallow — log and rethrow or handle explicitly |
| Passes locally, fails in CI | env var or runtime version mismatch | diff `process.env` keys and runtime version between environments |
| Fix works, but the test still fails | test asserts the wrong thing | re-read the test: is it checking the bug's symptom or the actual correct behavior? |
