# Framework-specific question: "how do I cache this data fetch in Next.js?"

Targets: stale, version-pinned advice presented as current.

**Pass**
- Checks (or asks for) the installed Next.js version before recommending a
  specific caching mechanism, since the caching API has changed across
  major versions.
- States the underlying tradeoff (revalidation strategy, cache scope)
  rather than only naming an API that may not exist in the project's version.

**Fail**: recommends a specific caching API/flag with no version check, as
if it's been stable and unchanged.
