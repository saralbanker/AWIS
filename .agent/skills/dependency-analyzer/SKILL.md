---
name: dependency-analyzer
description: >
  Use when auditing project dependencies for security vulnerabilities,
  license compliance, or outdated/transitive risk — before a release, after
  adding packages, or on a periodic health check. Not for application-level
  security logic (security-auditor); this is about the supply chain.
last_verified: 2026-08-30
kill_condition: the CVSS severity bands or SemVer conventions referenced here change industry-wide.
---

# Dependency Analyzer

## When to use
Before a release, after installing new packages, when a vulnerability scan
flags something, or as a periodic health check.

## Procedure
1. **Run the security scan** for the ecosystem in use (`npm audit`,
   `pip-audit`, `cargo audit`) and pull out Critical/High findings first.
2. **Trace transitive exposure**: a CVE in a dependency-of-a-dependency
   still affects you. `npm ls <package>` shows the chain; the fix is
   usually upgrading the direct dependency that pulls the bad transitive
   version.
3. **Check licenses** for anything copyleft (GPL/AGPL) in a commercial
   closed-source project — that's a legal question, not a technical one;
   flag it for review rather than deciding unilaterally.
4. **Check currency**: packages more than one major version behind are
   missing security fixes and accumulating breaking-change debt; packages
   with an available patch should almost always take it.
5. **Produce a prioritized action list**: fix critical CVEs now, high
   within the sprint, review license conflicts, schedule major upgrades.

## Avoid
Treating "0 CVEs reported" as "secure" (audit tools only know published
CVEs); upgrading a major version without reading its changelog/migration
guide; ignoring a brand-new low-download-count package's provenance when
it's about to become a direct dependency (typosquatting is real).

## Checklist
- [ ] 0 unaddressed critical/high CVEs in production dependencies
- [ ] No GPL/AGPL in a commercial closed-source project without legal review
- [ ] Outdated majors are scheduled, not just noted
- [ ] Action list has a specific command or PR per item, not just a severity label

See `reference/playbook.md` for scan commands, the license-risk table, and
CVSS severity bands.
