# System Auditor — Playbook

## Five Whys, worked example
```
1. Why did the API return 500?        → DB query timed out
2. Why did the query time out?        → connection pool exhausted
3. Why was the pool exhausted?        → connections not released after use
4. Why weren't they released?         → missing `finally` in the query wrapper
5. Why wasn't this caught earlier?    → no connection-pool monitoring/alerting
```
Root cause: missing connection cleanup *and* no monitoring. Fix both, not
just the code path.

## Evidence collection
```bash
grep -i "error\|fatal\|exception" /var/log/app/*.log | sort | tail -50
git log --oneline -10 --since="2 days ago"       # recent deploys
diff <(env | sort) <(ssh prod 'env | sort')       # local vs. prod config
```

## Blast-radius check
```bash
grep -rn "import.*from.*<failing-module>" src/
grep -rn "DATABASE_URL\|REDIS_URL" */config/ */.env
```

## No logs available (blind investigation)
List every component in the request path, health-check each independently
(`SELECT 1` for a DB, `redis-cli ping` for cache, a `/health` endpoint for
an API, consumer lag for a queue), then binary-search: does the failure
occur before or after component X? Once isolated, turn on logging,
reproduce, and continue as if logs were available from the start.

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| No errors in logs, but the system is clearly broken | a failure is being silently swallowed | add logging at suspected failure points and reproduce |
| Issue disappears when investigated closely | timing/load-dependent (Heisenbug) | reproduce under load; add persistent metrics instead of interactive debugging |
| Several services fail at once | shared dependency is down (DB/cache/DNS/cert) | check shared infrastructure health first, before individual services |
| Logs too noisy to find the signal | no log levels / unstructured logging | filter to WARN/ERROR/FATAL only as a first pass |
