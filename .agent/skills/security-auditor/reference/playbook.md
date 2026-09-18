# Security Auditor — Playbook

## Broken access control (OWASP A01)
```typescript
// Vulnerable — checks auth, not ownership
app.get('/api/orders/:id', authMiddleware, async (req, res) => {
  const order = await db.order.findUnique({ where: { id: req.params.id } });
  res.json(order); // any authenticated user can read any order
});

// Fixed — verifies ownership
app.get('/api/orders/:id', authMiddleware, async (req, res) => {
  const order = await db.order.findUnique({ where: { id: req.params.id } });
  if (order.userId !== req.user.id) return res.status(403).json({ error: 'Forbidden' });
  res.json(order);
});
```

## Injection (OWASP A03)
```typescript
// SQL — vulnerable vs. parameterized
db.query(`SELECT * FROM users WHERE id = '${req.params.id}'`);   // vulnerable
db.query('SELECT * FROM users WHERE id = $1', [req.params.id]);  // fixed

// XSS — vulnerable vs. safe
element.innerHTML = userInput;                        // vulnerable
element.textContent = userInput;                      // fixed
element.innerHTML = DOMPurify.sanitize(userInput);     // fixed, if HTML is required
```

## Authentication failures (OWASP A07)
```typescript
jwt.verify(token, secret);                              // vulnerable — accepts alg:none
jwt.verify(token, secret, { algorithms: ['HS256'] });    // fixed — explicit allow-list
```
Rate-limit login endpoints (e.g. 5 attempts/minute/IP); regenerate the
session ID on privilege escalation; invalidate sessions on logout.

## Secrets-detection patterns
```
(api[_-]?key|password|secret|token|credential|private[_-]?key)[\s]*[=:]\s*['"][^'"]{8,}['"]
(AKIA[0-9A-Z]{16})                            # AWS access key
(ghp_[A-Za-z0-9]{36})                         # GitHub PAT
```

## Scan commands
```bash
grep -rn -iE "(password|secret|api_key|token|private_key)\s*[=:]\s*['\"][^'\"]{8,}" src/
grep -rn "innerHTML\s*=" src/
grep -rn "exec(\|execSync(\|spawn(" src/
npm audit --json | jq '.vulnerabilities | to_entries[] | select(.value.severity=="high" or .value.severity=="critical")'
```

## CVSS severity bands
0.0-3.9 low (schedule) · 4.0-6.9 medium (this sprint) · 7.0-8.9 high (48h) ·
9.0-10.0 critical (immediately, consider rollback).

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| Auth middleware present, still exploitable | authz missing, not just authn | add an explicit ownership/role check per handler |
| `npm audit` clean but dependencies feel old | stale lockfile | `rm package-lock.json && npm install && npm audit` |
| Secret rotated in code but still exploitable | old value survives in git history | rotate immediately; purge history (BFG/filter-branch) — rotation matters more than history-scrubbing timing |
