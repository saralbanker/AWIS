# Dependency Analyzer — Playbook

## Scan commands

```bash
# Security — Node
npm audit
# Security — Python / Rust
pip-audit --format=json
cargo audit

# License summary (Node)
npx license-checker --production --summary
npx license-checker --production --failOn "GPL-2.0;GPL-3.0;AGPL-3.0"

# Currency
npm outdated   # Current | Wanted (respects semver range) | Latest

# Trace why a vulnerable package is installed
npm ls <vulnerable-package>
```

## SemVer, quick reference

`MAJOR.MINOR.PATCH` — major = breaking (read the migration guide), minor =
new features, backward compatible, patch = bug/security fix, always take it.
`^1.2.3` allows minor+patch updates (npm default, usually correct); `~1.2.3`
allows patch only; a bare `1.2.3` is pinned and needs manual updates.

## License risk, at a glance

| License | Commercial use | Forces your code open? |
|---|---|---|
| MIT / Apache-2.0 / BSD / ISC | yes | no |
| LGPL | yes | only the library itself, dynamic linking usually fine |
| GPL-2.0/3.0 | conditional | yes, if distributed |
| AGPL-3.0 | conditional | yes, even for SaaS (network copyleft) |
| Unlicensed | no | can't legally use it |

GPL/AGPL inside a commercial closed-source project is a legal-review item,
not something to resolve by picking an interpretation.

## CVSS severity bands

| Score | Severity | Typical response time |
|---|---|---|
| 0.0-3.9 | Low | next maintenance cycle |
| 4.0-6.9 | Medium | this sprint |
| 7.0-8.9 | High | 48 hours |
| 9.0-10.0 | Critical | immediately, consider rollback if already deployed |

## Supply-chain red flags before adding a new dependency
Typosquatted name (`lodahs` vs `lodash`), very low weekly downloads for
something about to become load-bearing, first published very recently,
unclear/unknown maintainer identity.

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| `npm audit` clean but deps feel stale | stale lockfile | `rm package-lock.json && npm install && npm audit` |
| Upgrade breaks the app | major version, breaking API change | read the changelog/migration guide before upgrading, not after |
| `license-checker` lists 1000+ entries | including dev dependencies | add `--production` |
| Peer dependency conflict blocks upgrade | A wants B@1.x, C wants B@2.x | check if A has a release supporting B@2.x before reaching for `--legacy-peer-deps` |
