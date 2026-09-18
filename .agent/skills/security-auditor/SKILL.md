---
name: security-auditor
description: >
  Use when reviewing code for security vulnerabilities — injection, auth
  flaws, secrets exposure, dependency CVEs. Not for general code quality
  (code-synthesizer) or performance (performance-optimizer).
last_verified: 2026-08-30
kill_condition: revisit if the OWASP Top 10 categories are superseded by a new edition, since category names/numbers here follow the current list.
---

# Security Auditor

## When to use
A security review request, code that handles user input/auth/sensitive
data, a pre-deploy gate, or after adding new dependencies.

## Procedure
1. **Scan for hardcoded secrets** and obvious injection vectors (string-
   concatenated SQL, `innerHTML` assigned from user input, `exec`/`eval`
   with unsanitized input) before anything else — these are cheap to find
   and severe when present.
2. **Check authorization, not just authentication.** A route being behind
   login middleware only proves identity; each handler must also verify
   the caller owns/can access the specific resource requested.
3. **Check dependency CVEs** (`npm audit` or ecosystem equivalent) — high
   and critical findings block release.
4. **Review auth mechanics**: JWTs verified with an explicit algorithm
   allow-list (reject `alg: none`), sessions invalidated on logout, login
   endpoints rate-limited.
5. **Check error handling**: responses to the client should never leak
   stack traces or internal details — log internally, return a generic
   message externally.
6. **Report by severity** (critical → high → medium), each finding with
   file:line evidence and a concrete before/after fix — no finding without
   proof, and no codebase is ever declared simply "secure."

## Avoid
Treating "no findings from an automated scan" as proof of security;
reporting a vulnerability class without a specific file:line instance;
skipping authorization checks because authentication passed.

## Checklist
- [ ] No hardcoded secrets found (real hits, not placeholders)
- [ ] All SQL is parameterized, no string concatenation
- [ ] No unsanitized `innerHTML` assignment from user input
- [ ] 0 unaddressed high/critical dependency CVEs
- [ ] Every sensitive route checks both auth and resource ownership
- [ ] JWT verification specifies an algorithm allow-list
- [ ] Error responses don't leak stack traces

See `reference/playbook.md` for OWASP-category code examples, secret-
detection regexes, and scan commands.
