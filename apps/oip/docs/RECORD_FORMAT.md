**SIGNED — FOUNDER SIGN-OFF 2026-09-19 (Gate G3 Question 2 / TDS-06); the Record format is OIP's irreversible artifact.**
**This sign-off covers the three specification defects corrected 2026-09-18: superseded
made a derived read-time state, `distinguishes` keys mandatory with nullable values, and
the ID allocation concurrency rule. Satisfies Gate G3 Question 2 / TDS-06.**

# OIP Record Format — TDS-06

## Overview

An OIP Record entry is a durable, provenance-bearing, append-only document that captures
an organizational decision: what was decided, why, what was rejected, and what confidence
was held (Articles 5, 7, 9, 10, 11, 13 of the OIP Constitution).

---

## Entry Layout

### Frontmatter (YAML between `---` delimiters)

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | string | yes | Entry identifier — ID scheme D-YYYY-MM-DD-NNN (see below) |
| `date` | string (ISO 8601) | yes | Date the entry was created (YYYY-MM-DD) |
| `title` | string | yes | Short human-readable title of the decision |
| `status` | enum: `decided` | yes | Lifecycle state of this entry as stored on disk. `superseded` is never a stored value — see "Derived state" below. |
| `tags` | array of strings | no | Organizational classification tags |
| `corrects` | string or null | no | ID of a prior entry this entry corrects (Art. 7/8) |
| `provenance` | object | yes | Attribution block — see sub-fields below |
| `provenance.origin` | string | yes | How the entry was captured (e.g., `manual`, `git-context-plugin`) |
| `provenance.authority` | string | yes | Actor ID who authorized the append |
| `provenance.sources` | array of strings | no | Source references (commit SHAs, URLs, step IDs) |
| `provenance.confidence` | string | yes | Epistemic confidence level (e.g., `high`, `medium`, `low`) |
| `distinguishes` | object | yes | Honesty register (Art. 10) — see sub-fields below |
| `distinguishes.observation` | string or null | yes | What was actually observed (evidence of what happened) |
| `distinguishes.description` | string or null | yes | What someone said or described |
| `distinguishes.intention` | string or null | yes | What someone intended or wanted to happen |

### Body Sections (Markdown)

All four body sections are required. An entry without any one of them is malformed.

```
## Decision

[The decision itself — what was chosen, in plain language.]

## Rationale

[Why this option was chosen over alternatives; constraints that applied;
 who was consulted. Preserves the "why" that evaporates fastest (Art. 11).]

## Rejected Alternatives

[What was considered and not chosen, and why each was set aside.]

## Unknowns

[What the decision does not resolve; open questions that remain (Art. 13).]
```

---

## ID Scheme

```
D-YYYY-MM-DD-NNN
```

- `D` — literal prefix for Decision entry
- `YYYY-MM-DD` — ISO 8601 date of creation (UTC)
- `NNN` — 3-digit per-day counter, zero-padded, starting at `001`
- Example: `D-2026-07-10-001`, `D-2026-07-10-002`

Files are stored at `.decisions/entries/<id>.md`.

### ID allocation

- The allocator scans `.decisions/entries/` for existing entries bearing the same
  `YYYY-MM-DD` and selects the next unused counter (max existing `NNN` + 1, or `001`
  if none exist for that date).
- Entry files MUST be created atomically and exclusively (`O_CREAT|O_EXCL` semantics,
  i.e. creation fails if the path already exists). On a collision — another appender
  claimed the same `NNN` first — the allocator retries with the next counter.
- Consequently, two concurrent appenders can never overwrite one another; an existing
  entry file is never replaced.
- Counter order reflects allocation order only and carries no semantic meaning (it is
  not a priority, sequence-of-truth, or ranking signal).

---

## Append-Only and Corrects Semantics

### Append-Only (Article 7)

Files under `.decisions/entries/` are NEVER rewritten once created. The Record is
corrected by appending truth, not by silently editing the past.

### Corrects Link (Articles 7 and 8)

When a new entry supersedes or corrects a prior entry:
1. The new entry sets `corrects: D-YYYY-MM-DD-NNN` pointing to the prior entry's ID.
2. The prior entry is NOT modified; its `status` field remains `decided`.
3. Readers following the `corrects` chain can reconstruct the full decision history.

Example:
```yaml
id: D-2026-07-10-002
corrects: D-2026-07-10-001
status: decided
```

This signals: "D-2026-07-10-002 is the current truth; D-2026-07-10-001 has been corrected."

### Derived state

An entry is considered SUPERSEDED when another entry's `corrects` field points at its
`id`. This is computed by the reader/index at read time — it is NEVER written into the
entry file, and no stored entry is ever rewritten to change its `status`. `status` on
disk therefore only ever holds `decided`; "superseded" is a label a reader applies after
walking the `corrects` chain, not a value that appears in any frontmatter block.

---

## Provenance (Article 9)

Every entry must answer: who/what created this, on whose authority, from what sources,
with what confidence. An unattributed assertion has no standing in the Record.

```yaml
provenance:
  origin: git-context-plugin      # capture mechanism
  authority: markus@example.com   # human who authorized the append (illustrative placeholder)
  sources:
    - "commit:abc123"
    - "pr:42"
  confidence: high
```

---

## Observation / Description / Intention Block (Article 10)

The `distinguishes` block prevents the Record from laundering one epistemic register
into another (observation ≠ description ≠ intention). The three KEYS
(`observation`, `description`, `intention`) MUST be present in every entry's
`distinguishes` block; their VALUES MAY be null when genuinely absent. An entry that
omits any of the three keys is malformed, even if the value it would have held is null.

```yaml
distinguishes:
  observation: "CI passed on all 3 runs; no flakes observed."
  description: "Team reports the new module is ready."
  intention: "Ship to production by end of sprint."
```

---

## Full Example Entry

```markdown
---
id: D-2026-07-10-001
date: "2026-07-10"
title: "Adopt modernc.org/sqlite as OIP's embedded database driver"
status: decided
tags:
  - architecture
  - storage
  - dependencies
corrects: null
provenance:
  origin: manual
  authority: markus@example.com   # illustrative placeholder
  sources:
    - "docs/05-implementation/M15-oip-on-awis/IMPLEMENTATION_SPEC.md"
  confidence: high
distinguishes:
  observation: "Platform go.mod already excludes CGO drivers; pure-Go driver confirmed."
  description: "modernc.org/sqlite is a CGO-free SQLite driver for Go."
  intention: "OIP should remain deployable without a C toolchain."
---

## Decision

Adopt `modernc.org/sqlite` as the embedded SQLite driver for OIP's `oip.db`. The driver
is pure Go (no CGO), consistent with the platform's existing dependency policy.

## Rationale

CGO-based drivers (mattn/go-sqlite3) require a C compiler at build time. OIP targets
environments where `CGO_ENABLED=0` must be possible. `modernc.org/sqlite` provides
identical SQLite semantics without the CGO requirement. The platform driver (used by
AWIS itself) uses the same selection for the same reason.

## Rejected Alternatives

- `mattn/go-sqlite3`: CGO dependency introduces build complexity and cross-compilation
  friction. Rejected.
- PostgreSQL: Adds an external service dependency, inconsistent with local-first
  deployment goals. Rejected for V1.

## Unknowns

- Performance characteristics at high-volume FTS workloads (expected to be acceptable
  for organizational decision volumes; not yet benchmarked).
- Whether V2 semantic search will require a different storage backend.
```

---

## Article-by-Article Compliance Table

| Article | Title | How this format satisfies it |
|---|---|---|
| Art. 5 | Record irreplaceable | `.md` files are the source of truth; `oip.db` is a rebuildable index. Rebuild path documented in OIP_DB.md. |
| Art. 7 | Durable history, append-only | Files are NEVER rewritten. Corrections append a new entry with `corrects:` link. |
| Art. 8 | Erasure is deliberate | No automated deletion path exists. Erasure requires explicit governed action outside this format. |
| Art. 9 | Provenance required | `provenance` block is required; `origin`, `authority`, `confidence` are required sub-fields. An entry missing provenance is malformed. |
| Art. 10 | Observation/description/intention | `distinguishes` block is required; the three registers are explicitly named and kept separate. |
| Art. 11 | Rationale first-class | `## Rationale` and `## Rejected Alternatives` are required body sections, not optional annotations. |
| Art. 12 | Intelligible to a future reader | Plain Markdown + YAML frontmatter; standard formats readable without this platform. |
| Art. 13 | Mark knowledge boundaries | `## Unknowns` is a required body section; `distinguishes.observation` may be null if evidence is absent (honest null > false certainty). |
