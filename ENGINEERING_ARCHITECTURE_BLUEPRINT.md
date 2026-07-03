# ENGINEERING ARCHITECTURE BLUEPRINT
## Organizational Intelligence Platform — V1

**Produced by:** System Architecture Deep Research Tribunal
**Date:** 2026-07-02
**Authority:** `OIP_CONSTITUTION.md` (Tier 0) + `PRD_RESEARCH_REPORT.md`
**Status:** ~~Final gate before implementation planning~~
**Governing constraint:** One experienced engineer, six weeks, production-ready foundation

---

> **⚠ SUPERSEDED — DO NOT USE FOR IMPLEMENTATION**
>
> This document describes a historical standalone OIP architecture produced before the AWIS platform was specified. It is superseded by **`AWIS_ARCHITECTURE_BLUEPRINT.md`**, which governs all implementation.
>
> **Correct architecture:** OIP is an AWIS application. Its capture and recall loops are AWIS workflows. Its step handlers implement the AWIS `StepHandler` interface. Its intelligence layer is AWIS's `IntelligencePort`. See `AWIS_ARCHITECTURE_BLUEPRINT.md` §10 ("How OIP Lives on AWIS") and `AWIS_ARCHITECTURE_FINALIZATION.md` §Blocker 1 for the full resolution.
>
> This document is retained as a historical reference showing the standalone design that preceded the platform decision. The OIP Constitution, the FTS index design (EDR-002), the entry format (EDR-001), the append-only enforcement (EDR-006), and the Git-as-sync model (EDR-003) remain correct and are adopted into the AWIS-application model. The module structure, IntelligencePort seam, and six-week implementation intuition all transferred cleanly.

---

## TRIBUNAL PREAMBLE

Five parallel investigations were conducted — Pattern Research, Platform Intelligence, AI Architecture, Dependency Analysis, and Solo-Founder Sustainability — then adversarially merged. Every recommendation below survived rejection attempts. Every abstraction required two genuine use cases before admission. Speculative architecture was ejected on discovery.

The Constitution is the upstream law. Where any EDR below could be read as contradicting a constitutional article, the article governs and the EDR must be amended. Article 51 (properties, never mechanisms) is the standing self-repair clause invoked throughout.

---

## 1. EXECUTIVE ARCHITECTURE SUMMARY

V1 is a **CLI-native, local-first decision-record instrument** for software teams. It implements exactly six modules, one of which (the Record) is the only durable artifact the project will ever produce. Everything else is either a replaceable adapter or a disposable surface.

The architecture's center of gravity is a 2-second description: **plain-text files in the team's repository, indexed by SQLite, queried by a short CLI.** Git provides versioning, history, permissions, sync, and exit rights at zero cost. An AI provider sits behind a single internal interface seam; swapping providers is a config change. With that seam removed entirely, the tool degrades to ADR practice — which is the zero-AI floor required by Article 32.

The one irreversible decision — the entry format — is the only place where slowness earns its price. Every other module is built to be thrown away without tears.

**Six modules:**

| Module | Nature | Replaceability |
|---|---|---|
| Capture Surface(s) | CLI command + optional git hook | Fully disposable |
| Context Assembler | Thin glue over git/VCS APIs | Easily replaced |
| Drafting Intelligence | Provider-abstracted AI call | One config key |
| **The Record** | Plain-text files, append-only | **NEVER replaced; format evolves only by versioned amendment** |
| Index | SQLite FTS5 + vectors, rebuildable cache | Fully disposable |
| Recall Engine | Query → cited answer | Replaceable |

**Six-week sequence in one sentence:** *Finalize the format → build append/read → build FTS index → build keyword recall → build context assembly + AI drafting → build confirm loop → build semantic recall → dogfood.*

---

## 2. CONSTITUTIONAL MAPPING

Every article is addressed. Articles dormant in V1 are noted with their activation stage.

### Title II — The Record (Articles 5–13): FULLY ACTIVE in V1
| Article | Engineering manifestation |
|---|---|
| 5 — Record is the only irreplaceable component | All modules are designed as replaceable consumers of the Record. The format is the architectural unit of highest design investment. |
| 6 — Organization owns the Record absolutely | Files live in the organization's own repository. `git clone` = complete export. |
| 7 — Append-only: correction by appending, never rewriting | Entry files are write-once. Corrections are new entries with `corrects:` reference. |
| 8 — Erasure is explicit, authorized, and itself recorded | V1: erasure handled manually as a recorded act. Tooling deferred to V2 (first real request is the trigger). |
| 9 — Every entry bears full provenance | Entry schema: `author`, `confirmer`, `machine_drafted`, `confidence`, `timestamp`, `evidence_links`. |
| 10 — Honesty registers (observation/description/intention) | V1 minimal form: two fields — `evidence_links` (observation) vs body text (description). `machine_drafted: true` marks AI contributions (Article 29). |
| 11 — Rationale is first-class, not optional | Entry body requires `rationale` and `alternatives_rejected`. These fields cannot be blank at confirmation. |
| 12 — Intelligible to a future reader without this platform | YAML frontmatter + Markdown body. E5 (decade-reader test) validates this in week 5. |
| 13 — Marks the limits of its own knowledge | Recall engine returns explicit "The Record is silent on this" when no matching entries found. |

### Title III — Capture and Memory (Articles 14–18): FULLY ACTIVE in V1
| Article | Engineering manifestation |
|---|---|
| 14 — Capture must not tax work | Draft-and-confirm loop: AI drafts, human confirms in ≤60s. Experiment E1. |
| 15 — The observed must receive value | Recall is the payoff for capture. Weekly recall events tracked from week 2. |
| 16 — No surveillance of persons | V1 capture is invoked or confirmed by the person whose work it is. No passive/silent observation. |
| 17 — Reference honestly what cannot be captured | `tacit_knowledge_holder` optional field: names the person who knows what can't be written. |
| 18 — Prevent accidental forgetting | Append-only + Git history + rebuildable index = zero accidental loss paths. |

### Title IV — Actors, Authority, and Trust (Articles 19–26): ATTRIBUTION ACTIVE; CHARTERS DORMANT
| Article | Engineering manifestation |
|---|---|
| 19 — Every act is attributable | Author + confirmer on every entry. Git commit attribution reinforces this. |
| 20–26 — Charters, trust ladder | No machine actor acts in V1. Full charter machinery activates at V3 (first action taken). V1 builds the attribution groundwork only. |

### Title V — Intelligence (Articles 27–33): PROVIDER SEAM ACTIVE; GOVERNANCE DORMANT
| Article | Engineering manifestation |
|---|---|
| 27 — Reasoning capacity is replaceable | `IntelligencePort` interface. Provider = one config key. |
| 28 — No intelligence owns or gates the Record | AI reads/writes through the same `RecordPort` as every other consumer. No private API paths. |
| 29 — Machine contributions marked permanently | `machine_drafted: true` + `machine_sections: [rationale, alternatives]` in every AI-produced entry. |
| 30 — Surface uncertainty | Recall response includes `confidence` field and explicit "the Record is silent" response type. |
| 31 — Machine memory lives in the Record | No AI provider state outside the Record. Context windows are ephemeral. |
| 32 — Functions without AI | `NullIntelligenceAdapter` produces blank templates; recall returns raw FTS results. E3 tests this. |
| 33 — Platform does not compete on model brilliance | Differentiator is context assembly quality and format longevity. |

### Title VI — Understanding and Action (Articles 34–40): DORMANT in V1
All six articles activate at V3 (chartered action). V1 has no execution. The groundwork (provenance, attribution, Record) is the prerequisite.

### Title VII — Form and Extension (Articles 41–46): CORE ACTIVE; EXTENSION DEFERRED
| Article | Engineering manifestation |
|---|---|
| 41–42 — Minimal core | V1 ships six modules. Nothing added without two genuine use cases. |
| 43 — No private doors | The `RecordPort` and `IntelligencePort` interfaces are the platform's own interfaces. External consumers use them identically. |
| 44–46 — Extensions inherit the constitution | Deferred: extension points defined after Ring 1 (first genuine external consumer). |

### Title VIII — Independence and Perpetuity (Articles 47–52): FULLY ACTIVE
| Article | Engineering manifestation |
|---|---|
| 47 — No load-bearing single-vendor dependency | Every dependency replaceable: AI provider (seam), SQLite (index is cache), Git (sync only), file format (versioned, open). |
| 48 — No projection is the platform | CLI is not the platform; index is not the Record; recall views are not the truth. |
| 49 — Exit engineered as real | `git clone` is complete exit. `oip export` produces JSON. Format spec is the exit artifact. |
| 50 — Plans for own succession | Record format is documented independently of the tool (RECORD_FORMAT.md). |
| 51 — Properties, never mechanisms | Every EDR below states the property being served, not only the mechanism chosen. |
| 52 — Amended by evidence only | This document is amendment-eligible. Section 16 (EDRs) carries the amendment record going forward. |

---

## 3. SYSTEM CONTEXT

```
┌─────────────────────────────────────────────────────────────────┐
│                    EXTERNAL ACTORS                               │
│                                                                  │
│  Developer / Tech Lead          AI Provider APIs                │
│  (capture, recall, browse)      (Anthropic, OpenAI, Ollama)    │
│                                                                  │
│  Git / VCS                      Editor ($EDITOR)                │
│  (identity, sync, history)      (confirmation UX)               │
│                                                                  │
│  GitHub / GitLab APIs           Future: Chat surfaces           │
│  (PR metadata, issues)          (CLI commands in Slack/chat)    │
└────────────────────────┬────────────────────────────────────────┘
                         │
         ┌───────────────▼──────────────────┐
         │         OIP V1 BOUNDARY          │
         │                                  │
         │  oip CLI binary                  │
         │  ├── capture                     │
         │  ├── recall                      │
         │  ├── browse                      │
         │  ├── init                        │
         │  ├── rebuild-index               │
         │  └── export                      │
         │                                  │
         │  .decisions/                     │
         │  ├── entries/       ← THE RECORD │
         │  ├── RECORD_FORMAT.md            │
         │  └── .index/        ← cache      │
         │      └── index.db               │
         └──────────────────────────────────┘
         │
         └── lives inside the team's git repo
```

**The V1 system is a single binary and a `.decisions/` directory.** No server, no database server, no cloud account, no network requirement beyond AI provider calls.

---

## 4. ARCHITECTURE PRINCIPLES

Derived from the Constitution and tempered by solo-founder constraints. Each principle states the constitutional source and the engineering consequence.

**AP-1 — The Record is the only artifact designed for permanence.**
*Source: Articles 5, 7, 12, FP-7.*
*Consequence:* The entry format receives disproportionate design time. All other modules are designed to be replaceable. The format is versioned from commit one (`format_version: 1` in every entry). Amendment procedure is documented before the format is used.

**AP-2 — The index is a cache, never a source of truth.**
*Source: Article 6, 47; PRD §10.*
*Consequence:* `oip rebuild-index` reconstructs the complete index from files alone. This command is tested in CI. Any query that cannot be answered from files + an empty index is a design defect.

**AP-3 — One provider seam; swapping is a config change.**
*Source: Articles 27, 47; FP-11.*
*Consequence:* `IntelligencePort` is defined as an interface before any AI call is written. The adapter (Anthropic, OpenAI, Ollama, Null) is selected at startup from config. No provider-specific type, SDK, or constant leaks past the seam.

**AP-4 — Append-only correctness: the past is amended, never rewritten.**
*Source: Article 7.*
*Consequence:* Entry files are written once and never modified. Corrections are new entries (`corrects: D-YYYY-MM-DD-NNN`). The index re-resolves correction chains at query time. This principle is enforced by making the `entries/` directory non-writable after the index build, enforced in tests.

**AP-5 — No abstractions without two genuine use cases.**
*Source: Article 42; FORBIDDEN_ASSUMPTIONS.*
*Consequence:* Every interface and abstraction in the codebase must map to at least two distinct concrete implementations or callers before it is kept. Abstract prematurely = delete on next review.

**AP-6 — Surfaces are disposable by declaration.**
*Source: Articles 41, 48.*
*Consequence:* The CLI commands, git hooks, and any future chat commands are structured as thin shells over the use cases. They contain no domain logic. Replacing the surface = write a new shell, keep the use cases.

**AP-7 — Context budget is governed.**
*Source: Articles 29, 30; PRD §12 Risk #2.*
*Consequence:* The context assembler enforces a hard token limit before sending to the intelligence adapter. Truncation is visible in the draft (a `[context truncated]` marker). Quality of context beats volume of context.

**AP-8 — Zero-AI viability is a mandatory test, not a principle left in comments.**
*Source: Article 32; PRD §14 Experiment E3.*
*Consequence:* The `NullIntelligenceAdapter` is the default during testing. CI runs the full capture/recall/export loop with intelligence disabled. Any test that requires a real AI call is integration-tagged and excluded from the default suite.

---

## 5. MAJOR PLATFORM MODULES

### Module 1: Capture Surface(s)

**Nature:** Disposable tooling. Contains zero domain logic.
**V1 implementation:** A single `oip capture` CLI subcommand plus an optional `post-merge` git hook that emits a reminder prompt.
**Responsibility:** Accept a trigger signal, invoke the Context Assembler use case, present the draft to the user for confirmation, and write the confirmed entry to the Record.
**Confirmation UX:** Opens the AI-drafted entry in `$EDITOR` (identical UX to `git commit` — familiar, keyboard-native, no chrome required). User saves and closes = confirmed. User deletes content = aborted.
**Manual path:** `oip capture --manual` opens a blank template. Zero AI. Same file format. Article 32 satisfied.

### Module 2: Context Assembler

**Nature:** Thin, product-specific glue layer. Easily replaced.
**Responsibility:** Given a trigger event (post-merge, explicit invocation), gather all available artifacts — git diff summary, commit messages, PR title/body/labels, linked issue titles, names of files changed — and assemble a structured `DecisionContext` object within the governed token budget.
**Sources:** `git` CLI (always available), GitHub/GitLab API (optional, degrades gracefully to git-only), local Record entries (related decisions via FTS query).
**Token budget:** Hard ceiling of 3,000 tokens assembled context. Diff is summarized (file list + line count) if it would exceed budget. Rationale: the draft quality depends more on what the context *identifies* than on full diff text.

### Module 3: Drafting Intelligence

**Nature:** Provider-abstracted AI adapter behind `IntelligencePort`.
**Responsibility:** Given a `DecisionContext`, produce a `DraftEntry` that fills the structured entry format: `decision`, `rationale`, `alternatives_rejected`, `evidence_links`. Mark every AI-produced field with `machine_drafted: true`.
**No side effects on the Record.** The adapter reads context; the use case writes the confirmed entry.
**Fallback:** `NullIntelligenceAdapter` returns an empty template with `machine_drafted: false`. Manual flow.

### Module 4: The Record

**Nature:** The platform's only irreplaceable module. The entry format is the product's one permanent artifact.
**Responsibility:** Append new entries as plain-text files. Read entries by ID or list all. Support correction entries. Provide the stream of entries for index rebuild.
**Physical form:** `.decisions/entries/D-YYYY-MM-DD-NNN.md` — one file per entry.
**Format:** YAML frontmatter + Markdown body (detailed in §12 — Data Architecture).
**Git provides:** history, diff, blame, permissions, sync, backup, export. No additional versioning system needed or built.

### Module 5: Index

**Nature:** Disposable cache. Rebuilds from files in seconds at any scale V1 will reach.
**Responsibility:** Provide keyword (FTS) and semantic (vector) search over entry content. Maintain entry metadata for fast listing and filtering.
**Physical form:** `.decisions/.index/index.db` — single SQLite file, git-ignored.
**Contents:** FTS5 virtual table over entry text + `embeddings` table (entry ID → 1536-dim float vector stored as BLOB or via sqlite-vec extension).
**Rebuild:** `oip rebuild-index` reads all files in `entries/` and repopulates. Tested in CI.

### Module 6: Recall Engine

**Nature:** Query layer with honesty built in.
**Responsibility:** Accept a natural-language query, search the index (FTS + semantic), retrieve matching entries, synthesize a cited answer via the intelligence adapter (or return raw excerpts without synthesis if AI unavailable), and explicitly state "The Record is silent on this" when no relevant entries exist.
**Citation format:** Every answer includes `[D-YYYY-MM-DD-NNN]` references that resolve to files the user can inspect. No disembodied claims.
**Honesty gate:** If the top result's similarity score is below threshold, prepend "Weak match — low confidence:" to the answer.

---

## 6. CAPABILITY GRAPH

```
CORE CAPABILITIES
│
├── RECORD MANAGEMENT
│   ├── append_entry(entry) → EntryId          [Record module]
│   ├── read_entry(id) → Entry                  [Record module]
│   ├── list_entries(filter?) → Entry[]         [Record module]
│   ├── correct_entry(id, text) → EntryId       [Record module]
│   └── export_all() → JSON|CSV                 [Record module]
│
├── INDEXING
│   ├── index_entry(entry) → void               [Index module]
│   ├── rebuild_index(entries[]) → void         [Index module]
│   ├── fts_search(query, limit) → Result[]     [Index module]
│   └── semantic_search(vector, limit) → Result[] [Index module]
│
├── CAPTURE
│   ├── assemble_context(trigger) → DecisionContext [Assembler]
│   ├── draft_entry(context) → DraftEntry       [Intelligence]
│   ├── confirm_entry(draft) → Entry            [Capture Surface]
│   └── embed_entry(text) → vector              [Intelligence]
│
├── RECALL
│   ├── query(text) → RecallResponse            [Recall Engine]
│   ├── synthesize(query, entries[]) → Answer   [Intelligence]
│   └── state_silence() → SilenceResponse       [Recall Engine]
│
└── MAINTENANCE
    ├── init_record(path) → void                [Record module]
    ├── rebuild_index() → void                  [Index module]
    └── export(format) → file                   [Record module]
```

---

## 7. DEPENDENCY GRAPH (DAG)

```
RECORD_FORMAT (schema spec, no code deps)
        │
        ├──────────────────────────────────┐
        ▼                                  ▼
  RecordPort                        IntelligencePort
  (read/write                       (draft/embed/
   interface)                        synthesize)
        │                                  │
        ├──────────┐               ┌───────┤
        ▼          ▼               ▼       ▼
    IndexPort  ContextAssembler  DraftingAdapter
        │       (git CLI, PR API)      │
        ▼              │               │
   SQLiteAdapter        └──────────────┤
        │                             ▼
        │                      ConfirmLoop (Capture Surface)
        │                             │
        └──────────────────────┐      │
                               ▼      ▼
                            RecordStore
                            (files on disk)
                                  │
                                  ▼
                            IndexBuild
                                  │
                     ┌────────────┘
                     ▼
               RecallEngine
               (FTS + semantic + synthesis)
```

**Critical path:** `RECORD_FORMAT → RecordPort → RecordStore → IndexBuild → RecallEngine`

**Parallel work after week 1:**
- Track A: `RecordStore` → `IndexBuild` → `RecallEngine` (recall loop)
- Track B: `ContextAssembler` → `DraftingAdapter` → `ConfirmLoop` (capture loop)
- Tracks merge at week 4 when both loops are wired together.

**Coupling risks:**
- Any coupling between the Intelligence adapter and the Record store is a constitutional violation (Article 28). The intelligence adapter reads context passed to it; it never calls RecordPort directly.
- The Index must not be treated as the source of truth. Any code path that would fail if the index were deleted is a defect.

---

## 8. PUBLIC CONTRACTS

These are the interfaces that external code — including future extensions, external tools, and the test suite — will depend on. They must be stable from commit one.

### The Record Entry Format

```yaml
---
# REQUIRED FIELDS — none may be empty at confirmation
id: D-2026-07-02-001            # Assigned at append time. Format: D-YYYY-MM-DD-NNN (NNN = daily sequence)
format_version: 1               # Schema version. Increment only with amendment procedure.
date: 2026-07-02T14:32:00Z      # ISO 8601. Set at append time. Immutable.
status: accepted                # accepted | superseded | deprecated (superseded/deprecated → corrects: field required)
decision: "Short statement of what was decided"   # max 120 chars; the entry's headline
machine_drafted: true           # true if AI produced any part of this entry
confidence: high                # high | medium | low — the author's confidence in completeness

# ATTRIBUTION — required
author:
  id: "git-email-or-platform-id"    # The person who made the decision
  name: "Human-readable name"
confirmer:
  id: "git-email-or-platform-id"    # The person who confirmed this record (may equal author)
  name: "Human-readable name"

# PROVENANCE — at least one evidence_link required for accepted entries
evidence_links:
  - type: pr | commit | issue | thread | doc | url
    ref: "e.g. github.com/org/repo/pull/42 or commit SHA"
    description: "Optional: why this artifact is relevant"

# HONESTY REGISTERS (Article 10 — minimal V1 form)
# machine_sections names which body sections were AI-drafted (empty list = fully manual)
machine_sections: [rationale, alternatives_rejected]

# OPTIONAL FIELDS
corrects: D-YYYY-MM-DD-NNN      # If this entry supersedes or corrects another
tacit_knowledge_holder:         # Person who holds knowledge not captured here (Article 17)
  id: "git-email-or-platform-id"
  name: "Human-readable name"
  what_they_know: "Brief description of uncaptured knowledge"
tags: [infrastructure, security, api-design]  # Free-form; used for browsing, not for recall logic
---

## Decision

[What was decided — expanded from the `decision:` headline if needed]

## Rationale

[Why this decision was made. Written from the perspective of the deciders at the time.
 If machine_drafted, this section was AI-produced and reviewed/confirmed by the confirmer.]

## Alternatives Rejected

[What was considered and not chosen, and why.
 Each alternative should answer: what it was, why it lost.]

## Context

[Optional: any situational context the rationale doesn't capture — constraints,
 deadlines, team composition, adjacent decisions that influenced this one.]
```

### CLI Contract

```
oip init                          Initialize .decisions/ in current git repo
oip capture [--manual] [--context=<type>]
                                  Invoke capture loop
                                    --manual: blank template, no AI
                                    --context=merge|pr|explicit
oip recall "<natural language query>"
                                  Query the Record; returns cited answer or silence
oip browse [--tag=<tag>] [--limit=N]
                                  List entries in pager (most recent first)
oip show <entry-id>               Display a specific entry
oip rebuild-index                 Rebuild index from files (idempotent, safe to run anytime)
oip export [--format=json|csv] [--output=<path>]
                                  Export all entries (exit right)
oip config set <key> <value>      Set configuration
oip config get <key>              Read configuration
```

---

## 9. INTERNAL CONTRACTS

These interfaces enforce the architecture's replaceability boundaries. They are not public APIs; they are internal seams the compiler enforces.

### IntelligencePort

```
interface IntelligencePort {
  // Given assembled context, produce a draft entry for human confirmation.
  // Must fill: decision, rationale, alternatives_rejected.
  // Must mark all AI-produced sections.
  draft(ctx: DecisionContext): Promise<DraftEntry>

  // Produce an embedding vector for an entry or query string.
  // Dimension is provider-specific; IndexPort adapts.
  embed(text: string): Promise<number[]>

  // Given a query and a set of retrieved entries, synthesize a cited answer.
  // Must include entry IDs as citations. Must return SilenceResponse if entries empty.
  synthesize(query: string, entries: Entry[]): Promise<RecallResponse>

  // Check if this adapter is operational (used for health/degradation reporting)
  isAvailable(): boolean
}

// NullIntelligenceAdapter implements IntelligencePort:
//   draft()       → blank template, machine_drafted: false
//   embed()       → all-zeros vector (FTS-only recall still works)
//   synthesize()  → returns raw entry excerpts, no synthesis
//   isAvailable() → false
```

### IndexPort

```
interface IndexPort {
  // Add or update an entry in the index.
  index(entry: Entry, embedding: number[]): Promise<void>

  // Full-text search. Returns entry IDs ranked by relevance.
  searchFTS(query: string, limit: number): Promise<SearchResult[]>

  // Vector similarity search. Returns entry IDs ranked by cosine similarity.
  searchSemantic(vector: number[], limit: number): Promise<SearchResult[]>

  // Drop and rebuild from scratch. Must be callable with zero side effects.
  rebuild(entries: Entry[]): Promise<void>
}
```

### RecordPort

```
interface RecordPort {
  // Write a confirmed entry to disk. Returns the assigned EntryId.
  // Throws if an entry with this ID already exists (append-only guarantee).
  append(entry: ConfirmedEntry): Promise<EntryId>

  // Read a single entry by ID.
  read(id: EntryId): Promise<Entry>

  // List all entries, optionally filtered.
  list(filter?: EntryFilter): Promise<Entry[]>

  // Resolve the correction chain for an entry (returns the latest valid entry).
  resolve(id: EntryId): Promise<Entry>
}
```

---

## 10. BUILD vs INTEGRATE MATRIX

| Subsystem | Decision | Reasoning | Constitutional anchor |
|---|---|---|---|
| **Record entry format + semantics** | **BUILD** — the platform's only constitutional IP. Spend all design budget here. | No existing format carries provenance, honesty registers, and the correction model. ADR formats lack all three. | Article 5 |
| **Draft-and-confirm capture loop** | **BUILD** — the core T1 hypothesis; nothing prior does confirmation-cost capture | No prior art. Experiment E1 is the test. | Article 14 + 11 |
| **Context assembly** | **BUILD (thin)** — product-specific glue over existing APIs | Git CLI + optional GitHub API. < 200 lines. Entirely replaceable. | Article 14 |
| **File storage** | **ADOPT: plain text on disk** — zero cost, zero ops, Git handles everything | Obsidian precedent. Decade-reader test satisfied by construction. | Articles 6, 12, FP-7 |
| **Index / keyword search** | **ADOPT: SQLite FTS5** — built-in to SQLite, zero additional dependency | Library-of-Congress longevity endorsement. Rebuilds from files in seconds. | Article 47 |
| **Index / semantic search** | **ADOPT: sqlite-vec extension OR in-process cosine similarity** | For V1 scale (< 10K entries), compute cosine similarity in the process itself on loaded BLOB vectors. sqlite-vec adds a dependency; defer until V2 when volume justifies it. | Article 47 |
| **Versioning, sync, history, backup** | **ADOPT: Git** — the team already runs it | Permissions, sync, attribution, history, and export inherited free. Build no transport. | Articles 6, 47 |
| **Drafting intelligence** | **INTEGRATE: rented model APIs behind IntelligencePort seam** | Provider swap = config change. Never build or fine-tune models in V1. | Articles 27, 47 |
| **Embedding model** | **INTEGRATE: provider-supplied** — same seam as drafting | text-embedding-3-small (OpenAI) or nomic-embed-text (Ollama) are adequate. No local fine-tuning. | Article 47 |
| **Identity / authentication** | **REUSE: VCS/git identity** — `git config user.email` | 5–50 person teams have git identity. Building auth is pure enterprise optimization. | Articles 19, 41 |
| **CLI framework** | **ADOPT: language-native** — commander/clap/click/typer per language choice | See EDR-009. | Article 41 |
| **Execution engine** | **NONE in V1; ADOPT (Temporal-class) in V3 when execution is earned** | R9 (Constitution). Rebuilding a durable-execution engine would be constitutional vandalism. | Article 34 |
| **Web application / UI** | **DEFER** — CLI + files cover V1 completely | Linear evidence: the niche forgives no web app; rewards speed. | Article 41 |
| **Sync beyond Git** | **DEFER** — E7 determines whether this is needed | If E7 shows Git is insufficient, adopt a sync transport; never build one. | Article 47 |

---

## 11. AI ARCHITECTURE

### Foundational Ruling

AI is infrastructure, not product. The platform's contribution is the context, format, and provenance model that no rented mind can bring. The AI provider is a temporary worker hired per request, paid per token, replaceable on any working day.

### Provider Seam

The `IntelligencePort` interface (§9) is the only place any provider-specific concept is permitted. The rule is absolute: **if a provider SDK type, a provider API constant, a provider-specific error, or a provider-specific retry behavior appears outside an adapter file, it is a bug, not a style issue.**

**Supported adapters (V1):**

| Adapter | Use case | Notes |
|---|---|---|
| `AnthropicAdapter` | Production default for drafting and recall synthesis | Claude Haiku for drafting (fast, cheap, adequate); Claude Sonnet for recall synthesis (higher quality for complex queries). |
| `OpenAIAdapter` | Alternative provider for drafting and synthesis | GPT-4o-mini for drafting; text-embedding-3-small for embeddings. |
| `OllamaAdapter` | Local development, zero-cost | llama3.3 or similar for drafting; nomic-embed-text for embeddings. No API cost; latency is higher. Recommended for the solo founder's own development machine. |
| `NullAdapter` | Zero-AI viability, testing default | Returns blank templates for drafting; returns FTS-only results for recall. Article 32 is verified against this adapter. |

**Provider swap mechanism:**

```yaml
# .decisions/config.yaml (git-ignored)
intelligence:
  provider: anthropic          # anthropic | openai | ollama | null
  draft_model: claude-haiku-4-5-20251001
  recall_model: claude-sonnet-4-6
  embed_model: text-embedding-3-small
  base_url: ~                  # Override for Ollama: http://localhost:11434
  api_key: ~                   # Read from env: ANTHROPIC_API_KEY | OPENAI_API_KEY
```

### Context Management

The capture context is governed to prevent prompt bloat and runaway cost.

```
CAPTURE CONTEXT BUDGET: 3,000 tokens max

Sources (assembled in priority order until budget reached):
1. PR title + description           (~200 tokens; always included)
2. Commit messages in the branch    (~200 tokens; always included)
3. Changed file list                (~100 tokens; always included)
4. Related prior decisions (FTS)    (~500 tokens; top 2 results)
5. PR diff summary (file + ±lines)  (~300 tokens; never raw diff)
6. Issue titles linked in PR        (~200 tokens; if API available)
7. Thread excerpts                  (~500 tokens; if available)

Hard stop: never exceed budget. Truncate source 7 → 6 → 5 before cutting 1–4.
Visible in draft: [context truncated: diff summary only] when truncation occurred.
```

**Recall context budget:** 8,000 tokens (query + top N entries). N capped at 8 entries. Synthesis call uses Sonnet-class model only when entry count > 2 (Haiku-class for simple retrieval of 1–2 entries).

### AI Failure Handling

| Failure mode | Behavior |
|---|---|
| Provider unavailable (network, rate limit) | Degrade to manual capture template. Log the failure. Do not crash. |
| Context too large (truncation required) | Apply budget rules, mark `[context truncated]` in draft. |
| Draft quality below threshold (E1 monitoring) | No automatic quality gate in V1. Metrics collected; intervention is human-led. |
| Embedding provider unavailable | Fall back to FTS-only search. Recall still works. Log degradation. |
| API key missing | Surface clear error at startup: "Set ANTHROPIC_API_KEY or configure `intelligence.provider: null` for manual-only mode." |

### AI Boundaries — What AI Must Never Do

- **Never write to RecordPort directly.** The intelligence adapter is called by the use case; the use case writes. The adapter cannot call RecordPort.
- **Never modify existing entries.** Corrections are new entries appended by the use case.
- **Never decide what to capture.** AI drafts; humans decide. The capture surface always requires explicit human confirmation.
- **Never gate access to the Record.** Query recall with AI unavailable returns FTS results — not an error.
- **Never retain state between calls.** No session management. Each call is stateless from the adapter's perspective.

---

## 12. DATA ARCHITECTURE

### Record Storage Layout

```
.decisions/                          ← tracked by Git; this IS the Record
  RECORD_FORMAT.md                   ← format specification (E5's test artifact)
  entries/
    D-2026-07-02-001.md              ← one file per confirmed entry
    D-2026-07-02-002.md              ← daily sequence number (three digits, padded)
    D-2026-07-04-001.md
  .index/                            ← git-ignored; disposable cache
    index.db                         ← SQLite: FTS5 + embeddings
    index.version                    ← schema version of last rebuild
  config.yaml                        ← git-ignored; local config (provider, keys)
  .gitignore                         ← generated by `oip init`; ignores .index/ and config.yaml
```

### Index Schema (SQLite)

```sql
-- Metadata table (fast list/filter)
CREATE TABLE entries (
  id TEXT PRIMARY KEY,               -- D-YYYY-MM-DD-NNN
  date TEXT NOT NULL,                -- ISO 8601
  decision TEXT NOT NULL,            -- headline (120 chars)
  author_id TEXT NOT NULL,
  author_name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'accepted',
  confidence TEXT NOT NULL DEFAULT 'high',
  machine_drafted INTEGER NOT NULL DEFAULT 0,  -- boolean
  corrects TEXT,                     -- nullable EntryId
  tags TEXT,                         -- JSON array stored as text
  format_version INTEGER NOT NULL DEFAULT 1
);

-- Full-text search over complete entry content
CREATE VIRTUAL TABLE entries_fts USING fts5(
  id UNINDEXED,
  decision,
  rationale_text,                    -- extracted markdown body: rationale section
  alternatives_text,                 -- extracted markdown body: alternatives section
  context_text,                      -- extracted markdown body: context section
  content='entries',
  tokenize='porter unicode61'
);

-- Embedding vectors (V1: BLOBs; V2: migrate to sqlite-vec if justified)
CREATE TABLE embeddings (
  entry_id TEXT PRIMARY KEY REFERENCES entries(id),
  vector BLOB NOT NULL,              -- Float32Array, little-endian, 1536 dims
  model TEXT NOT NULL,               -- e.g. "text-embedding-3-small"
  model_version TEXT NOT NULL        -- provider version at embed time
);

-- Evidence links (denormalized for fast filter)
CREATE TABLE evidence_links (
  entry_id TEXT NOT NULL REFERENCES entries(id),
  type TEXT NOT NULL,                -- pr | commit | issue | thread | doc | url
  ref TEXT NOT NULL,
  description TEXT
);
```

### Entry ID Scheme

`D-YYYY-MM-DD-NNN` where NNN is a zero-padded daily sequence number assigned at append time by scanning existing files for the current date. No central counter; no race condition (single-user tool). Lexicographic sort = chronological sort.

### Correction Chain Resolution

When entry `D-2026-07-02-001` is corrected by `D-2026-07-05-001`:

```
D-2026-07-02-001 (status: superseded, no corrects field)
D-2026-07-05-001 (status: accepted, corrects: D-2026-07-02-001)
```

`RecordPort.resolve(D-2026-07-02-001)` follows the correction chain and returns `D-2026-07-05-001`. Recall presents the resolved entry and notes "This supersedes D-2026-07-02-001."

### Export Contract (Article 6 — Exit Right)

`oip export --format=json` produces:

```json
{
  "format_version": 1,
  "exported_at": "2026-07-02T14:32:00Z",
  "tool_version": "0.1.0",
  "entries": [
    {
      "id": "D-2026-07-02-001",
      "date": "2026-07-02T14:32:00Z",
      "decision": "...",
      "rationale": "...",
      "alternatives_rejected": "...",
      "machine_drafted": true,
      "author": { "id": "...", "name": "..." },
      "confirmer": { "id": "...", "name": "..." },
      "evidence_links": [...],
      "status": "accepted",
      "corrects": null
    }
  ]
}
```

This is the exit artifact. A competent engineer can reconstruct the full organizational decision history from this JSON without the tool, satisfying Article 6 and the decade-reader test.

---

## 13. EVOLUTION STRATEGY

Each stage is activated by evidence from the previous stage, never by ambition or schedule.

### V1 — Memory (the organism)

*What:* Capture → Confirm → Record → Recall. One team. Files + index. CLI only. Two surfaces max (CLI + optional git hook).

*Validates:* Articles 5–19, 27–33, 41–50. Experiments E1–E7.

*Gate to V1.5:* E1 passes (≥5 confirmed captures/week/team past week 4, ≤60s median confirm). E2 passes (≥3 recall events/week/team).

### V1.5 — Memory that speaks up

*What:* Proactive surfacing at capture and code-review time. "This PR touches decision D-041: we rejected this approach because…" Still zero machine autonomy — the suggest rung (Article 22 first rung above observe). No action, only suggestions.

*Validates:* Suggest-rung trust mechanics. The compounding claim in constitutional miniature.

*Gate to V2:* Suggestion uptake rate > 40% (suggestions acted upon). Silent-record pattern resolved.

### V2 — Memory across boundaries

*What:* Commitments join decisions as Record entry kinds (if Q1 evidence says so). Cross-team recall. `RECORD_FORMAT.md` published as an open spec (first external-consumer enablement). Consent-shaped observation beyond git exhaust.

*Validates:* Article 10 honesty registers at scale. Constitutional Q2 (social license). Cold-start economics at org level.

*Gate to V3:* One genuine external consumer of the format (the signal Git received). E7 verdict on sync.

### V3 — Chartered action

*What:* First capability that does something — a small, reversible, chartered act grounded in the Record. Execution substrate **adopted, not built** (Temporal or similar). Full trust-ladder and charter machinery implemented, because now something exists to govern.

*Validates:* Articles 20–26 (charters, trust ladder). Articles 34–40 (understanding before automation, recoverability, declarative reconciliation).

### V4+ — The platform

*What:* Multiple intelligences over one Record. Extension ecosystem (Articles 41–46 externalized). Exit-rights commercial strategy faces its real test at renewal scale (Q3, Q7).

*Constitutional articles reached:* All 52 are live. The full shape of the Constitution is instantiated.

### Architectural Evolution Rules

1. **The format evolves by amendment.** `format_version: 2` entries can coexist with `format_version: 1` entries in the same `entries/` directory. The reader handles all versions. Migration is never required — additive evolution only.
2. **The intelligence seam never narrows.** New providers can be added; no provider-specific behavior may leak past the seam into new code.
3. **New entry kinds are additive.** When commitments or cases join the Record, they are new files with a new `entry_kind:` field. Old tooling that doesn't recognize the kind treats it as a forward-compat unknown and skips it gracefully.
4. **The index schema versions independently.** `index.version` file tracks the last-built index schema. `oip rebuild-index` rebuilds if version mismatch detected.

---

## 14. ARCHITECTURE RISK MATRIX

Ranked by irreversibility × likelihood × long-term damage.

| # | Risk | Class | Damage | Probability | Irreversibility | Mitigation |
|---|---|---|---|---|---|---|
| **R1** | **Record format too software-specific or too rigid** — blocks every future vertical; the one true irreversible artifact | Architectural | **Critical** | Medium | **Very high** | Niche-neutrality rule (T3): niche shapes surfaces and vocabulary, never format semantics. E5 (decade-reader test) in week 5 before format freeze. Format version from day one. |
| **R2** | **Draft quality below confirmation threshold** — humans rewrite instead of confirm → authoring cost returns → ADR death spiral | Adoption/Technical | **Critical** | Medium | Medium | Rich context assembly beats clever prompting. Measure edit-distance from draft to confirmed entry. If > 40% of text is rewritten, the context assembly is the problem. Manual path as non-zero-friction floor. |
| **R3** | **Capture moment mis-located** — hook fires when no decision concluded; false positives → users kill the hook | Product | High | Medium-High | Low (fix the trigger) | Start with manual invocation only (`oip capture`). Add hook as opt-in in week 4. Dogfood tunes the trigger signal. |
| **R4** | **AI dependency creep into core** — Article 32 silently violated | Architectural | High | Medium | Medium | `NullAdapter` is the testing default. CI suite runs fully with intelligence disabled. If any test fails with NullAdapter that isn't tagged `[integration]`, it's a blocking bug. |
| **R5** | **Cold-start gap** — recall has nothing meaningful to say for weeks | Adoption | High | Medium-High | Low | ADR-folder import (`oip import-adrs`) fast-tracks the initial Record. E6 determines whether import must be Must-level before Ring 1. "Related decisions at capture time" creates value from entry #10. |
| **R6** | **Provider seam leak** — provider-specific error types, retry logic, or model names appear outside adapter files | Architectural | Medium-High | Medium | Low (refactor is cheap early) | Code review rule: every PR is checked for SDK types outside `adapters/`. Pre-commit lint rule checking for provider SDK imports in non-adapter paths. |
| **R7** | **IntelligencePort interface too narrow** — must be broken to add capabilities | Architectural | Medium | Low | Medium once used by two adapters | Design the interface with three methods (draft, embed, synthesize) and a fourth escape hatch (raw_call) for unforeseen needs. Review interface design in week 1 before any adapter is written. |
| **R8** | **Git as sync insufficient** — non-developer team members or Windows users encounter friction | Product | Medium | Medium | Medium (seam is there) | E7 monitors. Transport abstraction is in place (git is one transport implementation). First alternative transport is not built until E7 fails. |
| **R9** | **Solo-maintainer burnout via surface sprawl** | Maintenance | Medium | High (classic failure mode) | Low (delete surfaces) | Hard cap: two capture surfaces until V1 metrics pass Ring 1. Every surface added must have a deletion plan. |
| **R10** | **Semantic search quality inadequate** — users get wrong answers with confidence | Trust | Medium | Low-Medium | Low (tune threshold) | Honesty gate: low-similarity results are marked "Weak match — low confidence." Silence response is the fallback. Threshold is tunable config. |

---

## 15. ENGINEERING EXPERIMENTS

These are the implementation-stage experiments that produce the evidence the Constitution requires. Each has a named hypothesis, a measurement plan, and a failure action.

**E1 — Confirmation-cost capture (the load-bearing experiment)**
- Hypothesis: AI-drafted, human-confirmed capture sustains ≥5 entries/week/team past week 4, ≤60s median confirm time.
- Measurement: Capture telemetry (timestamp at draft presented → timestamp at confirm); edit-distance between draft and confirmed entry.
- Pass condition: ≥5 entries/week AND ≤60s median AND edit-distance < 40% of draft text.
- Failure action: If edit-distance > 40%: improve context assembly (richer context beats better prompts). If entries < 5: capture moment is wrong or friction is elsewhere — dogfood interviews. If both fail past week 8: amend Article 11 (rationale is best-effort, not first-class mandatory) and reconsider the product center.

**E2 — Recall value**
- Hypothesis: Teams ask the Record ≥3 why-questions/week and act on the answers.
- Measurement: Query logs + weekly qualitative check ("did this save re-litigation?").
- Pass condition: ≥3 recall events/week/team AND ≥1 saved-archaeology incident reported/week.
- Failure action: Silent-record pattern → proactive surfacing (V1.5 moved up). If proactive fails → demand assumption (§3.1 PRD) is falsified → Constitution §2 needs examination against reality.

**E3 — Zero-AI degradation**
- Hypothesis: With NullAdapter active, capture/recall/export all function, and manual capture ≈ ADR-practice effort.
- Measurement: Scripted test run with `intelligence.provider: null`. All core commands must succeed or degrade gracefully (no crashes).
- Pass condition: All commands succeed or print a graceful degradation message. Manual capture produces a valid entry. Export produces complete JSON.
- Run schedule: Weekly in CI during V1 development; quarterly after launch.
- Failure action: Any core function that requires a model call is a blocking architectural bug.

**E4 — Registers in the small**
- Hypothesis: Users understand the `evidence_links` vs body-text distinction without instruction and don't rebel against `machine_drafted:` marks.
- Measurement: Confirm-time deltas with vs without evidence fields; field omission rate; partner interviews at week 8.
- Pass condition: < 20% of entries submitted with empty evidence_links; no expressed friction with machine_drafted marks.
- Failure action: If evidence_links routinely left empty, consider making them optional after Ring 1. If machine_drafted marks cause confusion, adjust UX (perhaps just a subtle footer, not frontmatter). Article 10 is demoted to best-effort notation.

**E5 — Decade-reader round-trip**
- Hypothesis: A competent engineer with no access to the tool reconstructs a team's decision history from raw files in < 1 hour.
- Measurement: Staged trial with an outsider given only the `entries/` directory and `RECORD_FORMAT.md`.
- Pass condition: Outsider can answer "what did the team decide about X?" from files alone, without the tool, in < 1 hour.
- Run timing: Week 5, before format freeze.
- Failure action: Format requires the app to be intelligible → redesign the format. Do not freeze format until this passes.

**E6 — Cold-start floor**
- Hypothesis: Recall becomes subjectively useful before 50 entries when seeded with an ADR import.
- Measurement: Partner interviews at entry #25 and #50 asking "did the recall answer help?"
- Pass condition: ≥70% of queries return subjectively useful results by entry #50.
- Failure action: Useful at #25 → onboarding is fine. Useful only at #200 → ADR import is mandatory for Ring 1 teams.

**E7 — Git as team transport**
- Hypothesis: Repo-based sharing covers 5–50-person team needs without any built sync.
- Measurement: Partner friction reports about merge conflicts, discovery failures, or non-developer access.
- Pass condition: No P1/P2 friction complaints about sync in first 12 weeks of Ring 1.
- Failure action: Sync friction → design a transport abstraction; evaluate hosted option. Git stays as one implementation.

---

## 16. ENGINEERING DECISION RECORDS (EDRs)

Each EDR is this project's own use of the Record format it is building.

---

**EDR-001: Plain-Text YAML+Markdown as the Record's physical format**

*Decision:* Entry files are YAML frontmatter + Markdown body, stored as one `.md` file per entry in the team's git repository.

*Context:* The format is the platform's only irreversible artifact. It must satisfy: human readability without tooling (decade-reader test, Article 12); portability (Article 6); extension without migration (additive evolution); familiarity to software teams (adoption, FP-6).

*Alternatives Considered:*
- Pure YAML: More machine-readable; less human-readable. Markdown narrative sections become awkward. Fails the decade-reader test for multi-paragraph rationale.
- JSON: Machine-canonical; hostile to human editors. Fails Article 12.
- TOML frontmatter: Technically equivalent; YAML has higher ecosystem support (Jekyll, Hugo, Obsidian). No advantage.
- SQLite as the record itself (not just the index): Violates Article 12 (binary format); violates exit rights (requires tooling to read); violates decade-reader test.
- Custom DSL: Invents proprietary notation where a standard exists. Violates Article 46.

*Trade-offs:*
- YAML has footguns (indentation sensitivity, multiline strings). Mitigated by: the tool generates YAML; humans primarily edit the Markdown body; YAML errors are caught at index build.
- Markdown is ambiguous without a specification. Mitigated by: adopting CommonMark as the spec; no processing of Markdown structure beyond section headings.

*Evidence:* Jekyll (2008), Hugo (2013), Obsidian (2020), countless static-site generators use YAML+Markdown for exactly this reason. The format is readable by any plain-text editor made after 1980.

*Confidence:* High. *Future Impact:* Format versioning from day one makes this decision amendable. The highest-regret version of this decision is not YAML vs. TOML — it's semantics too software-specific. Niche-neutrality (T3) guards against that.

---

**EDR-002: SQLite for the index (FTS5 + BLOB vectors)**

*Decision:* The index is a single SQLite file using FTS5 for keyword search and BLOB-stored Float32Array for vector similarity (computed in-process). sqlite-vec extension deferred until V2 scale justifies the dependency.

*Context:* The index must be: zero-ops (solo founder), disposable and rebuildable, capable of FTS and vector similarity, survivable across OS migrations.

*Alternatives Considered:*
- PostgreSQL: Requires a running server. Zero-ops violated. Over-engineered for local, single-user tool.
- Elasticsearch / OpenSearch: Same problems; worse.
- Chroma / Qdrant / Weaviate (vector DBs): Additional running server; vector-DB-only; no built-in FTS. Two deps instead of one.
- DuckDB: Excellent for analytics; no built-in FTS5; newer (2019) than SQLite's 35-year track record.
- sqlite-vec: Right choice for V2. In V1, cosine similarity over < 10K entries in-process is < 100ms; the dependency is not yet justified.

*Trade-offs:*
- In-process cosine similarity: O(N) per query but fast enough at V1 scale. Profile at 5K entries and decide.
- BLOB embedding storage: Not queryable by SQL. Acceptable because vectors are loaded in-process for similarity computation.

*Evidence:* SQLite is used as an application file format by Obsidian, WhatsApp, Firefox, and the U.S. Library of Congress. "The Library of Congress recommends SQLite as an archival format." 35-year track record. Zero ops. Single file.

*Confidence:* Very high. *Future Impact:* The index is a cache; this decision has zero regret upside regardless of correctness.

---

**EDR-003: Git as versioning, sync, history, backup, and exit mechanism**

*Decision:* The `.decisions/entries/` directory lives in the team's existing git repository. Git provides all versioning, history, collaboration, permissions, and backup. No sync infrastructure is built.

*Context:* The team already runs git. Sync is solved. History is solved. Attribution is augmented by git commit authorship. Exit is `git clone`.

*Alternatives Considered:*
- Custom sync protocol: Builds something mature infrastructure already provides. Violates Article 47 (load-bearing dependency on custom infrastructure).
- Hosted database (Postgres, Firebase): Requires a running server, an account, and a network. Introduces vendor dependency for the Record's survival (Article 47 violation).
- rsync / Dropbox: No history, no attribution, no branching model for proposed corrections.

*Trade-offs:*
- Git couples early UX to developer tooling (E7 monitors this). Non-developer team members in future verticals may not use git. The transport is behind a seam; alternatives can be added without changing the Record format.
- Merge conflicts in `.decisions/entries/` are unlikely (one file per entry, append-only) but possible (concurrent editing of `config.yaml`). Mitigated by the per-entry file structure.

*Evidence:* Git's data model has outlived every Git client. The repository format from 2005 is still valid. The platform's goal in 2031 ("the Record outlives every engine that ever ran against it") is exactly what Git achieves for source code.

*Confidence:* Very high for the beachhead. Monitored by E7.

---

**EDR-004: IntelligencePort — single provider seam, zero leakage**

*Decision:* All AI provider interactions go through a single `IntelligencePort` interface defined in one file. No provider SDK type, error, model name, or retry strategy appears outside the adapter implementation file. Provider selection is a config key.

*Context:* Article 47 (no load-bearing single-vendor dependency) is violated the moment any provider SDK leaks past the seam. FP-11 (reasoning trends toward commodity) makes provider independence not just principled but practically correct within the planning horizon.

*Alternatives Considered:*
- LangChain / LlamaIndex: Framework-level abstraction. Adds framework dependency (another load-bearing vendor). Every framework update is a forced upgrade. For the minimal operations of V1 (draft, embed, synthesize), a framework is over-engineered by a factor of 20.
- Direct SDK per provider: Fast to start; violates Article 47 the moment a second provider is tested.
- OpenAI-compatible shim: Many providers implement the OpenAI API. Could simplify the seam. Risk: relies on OpenAI's API design remaining stable; couples the seam to a specific provider's protocol.

*Trade-offs:*
- The seam requires discipline in code review. A pre-commit lint rule enforcing no SDK imports outside adapter files makes this machine-enforceable.
- Three methods (draft, embed, synthesize) may not be sufficient for all future intelligence capabilities. The escape hatch method `raw_call(request)` provides extension without seam breakage.

*Evidence:* The database-agnostic repository pattern has survived decades of ORM evolution. The same logic applies here.

*Confidence:* High. *Future Impact:* This is the second-highest-leverage architectural decision after the format. Provider fragmentation accelerates; the seam's value compounds yearly.

---

**EDR-005: CLI-first; no web app in V1**

*Decision:* V1 ships a CLI binary. No web application, no Electron app, no browser extension. The read view (`oip browse`, `oip show`) uses the terminal pager.

*Context:* The beachhead user (founding engineer / tech lead) lives in the terminal. Linear won its market with CLI-native, keyboard-first interaction while Jira had web polish. The niche rewards speed over chrome. Surfaces are declared disposable by Article 48.

*Alternatives Considered:*
- Local web app (served on localhost): More browsable but adds a build pipeline (webpack/vite), a frontend framework, a local HTTP server, and CORS considerations. Zero constitutional pull.
- TUI (terminal UI with tview/bubbletea/blessed): More visual than plain pager; adds a rendering library dependency. Potentially warranted for `oip browse`; deferred to V1.5 based on dogfood friction.
- Obsidian plugin: Would place the Record in Obsidian's vault. Dependent on Obsidian's plugin API (not an open standard). Better as a future projection/extension than as the primary interface.

*Trade-offs:*
- Non-developer stakeholders (who might want to read decisions) cannot easily access the Record without git. Mitigated: (a) the JSON export is always available; (b) a read-only web view is V1.5 if dogfood reveals this friction.

*Confidence:* High. Reversible: surfaces are declared disposable from the start.

---

**EDR-006: Append-only correctness enforced structurally**

*Decision:* The `entries/` directory is populated by a write-once function. No function in the codebase modifies an existing entry file. Corrections are always new files. A CI test verifies that running `oip capture` twice does not modify any file written by the first invocation.

*Context:* Article 7 (append-only, correction by appending) is architecturally non-negotiable. The history of audit-trail systems shows that "we'll never modify it" as a convention fails; it must be enforced.

*Alternatives Considered:*
- Mutable files with git history as the audit trail: Git history is the last line of defense, not the first. If the tool modifies files, git diff of entries/ is opaque. The audit is the append structure, not git blame.
- Cryptographic signatures on files: Provides tamper evidence; adds key management overhead. Deferred: the decade-reader test (E5) is the V1 tamper-evidence mechanism. Cryptographic signing is a V3 addition when the trust stakes justify it.

*Confidence:* Very high. *Future Impact:* The correction chain (`corrects:` field) is the mechanism that makes append-only human-workable. Invest in making the `oip correct <id>` command smooth.

---

**EDR-007: VCS identity as the attribution mechanism**

*Decision:* Author and confirmer identity on every entry is sourced from `git config user.email` and `git config user.name`. No separate identity system is built. The tool does not authenticate users.

*Context:* 5–50 person software teams have git identity established. Building an auth system is pure enterprise optimization banned by FORBIDDEN_ASSUMPTIONS. Article 19 (every act attributable) is satisfied because git identity is the team's existing attributable identity.

*Alternatives Considered:*
- Email/password auth: Adds an identity database, password hashing, session management, and email verification. Violates Article 41 (minimal core) and FORBIDDEN_ASSUMPTIONS.
- OAuth (GitHub/Google): Adds OAuth flow and token storage. Heavier than git identity for a local tool. V3 consideration when collaborative web access exists.
- Anonymous entries: Violates Article 19 directly.

*Trade-offs:*
- Git identity is not verified by the tool (users could lie). Accepted: the tool is not a security system; it's a memory system. The team's social contract governs identity honesty. This is no different from git commit authorship.

*Confidence:* High. *Activation of proper identity:* When V3 introduces chartered machine action, verified identity (OAuth or similar) is required. Not before.

---

**EDR-008: Semantic search via in-process cosine similarity; defer sqlite-vec**

*Decision:* Embeddings are stored as BLOB in SQLite. Cosine similarity is computed in-process by loading embedding BLOBs into a Float32Array and computing dot products. sqlite-vec extension is deferred until V2 when entry volume warrants it.

*Context:* For V1 scale (< 10,000 entries across all teams), in-process cosine similarity over the full embedding set takes < 100ms on commodity hardware. The sqlite-vec dependency adds a compiled extension that must be distributed per platform.

*Alternatives Considered:*
- sqlite-vec: Right choice for V2+ when query latency matters or entry count exceeds 50K. Deferred.
- Chroma / Qdrant: Running server required. Zero-ops violated.
- Pinecone / hosted vector DB: Vendor dependency for a local-first tool. Constitutional violation.
- No semantic search: FTS5 alone. Acceptable as V1 floor (E3). Semantic search is the enhancement.

*Performance projection:* At 10K entries × 1536 dims × 4 bytes = 58MB of vectors in memory. On a 2024 laptop with AVX2: cosine similarity across 10K entries ≈ 20ms. Acceptable until V2.

*Confidence:* High for V1. *Upgrade trigger:* P75 recall query latency > 500ms, or entry count > 50K, or distribution complexity outweighs in-process simplicity. Upgrade path: swap IndexAdapter to use sqlite-vec; no Record format change required.

---

**EDR-009: Language selection criteria**

*Decision:* The Constitution forbids mandating a language. This EDR documents the evaluation criteria and the top two options, leaving the final choice to the builder's demonstrated fluency.

*Required capabilities:*
1. Single-binary distribution (no runtime install on user machine)
2. SQLite with FTS5 and BLOB support
3. Async HTTP client for AI provider APIs
4. Markdown/YAML parsing
5. File I/O with atomic writes
6. Cross-platform (macOS + Linux primary; Windows secondary)
7. Fast iteration (< 30 second build cycle)

*Option A — TypeScript + Bun:*
- `bun compile` produces a single binary with zero Node.js install required
- `better-sqlite3` (via Bun's native bindings) for SQLite
- `@anthropic-ai/sdk`, `openai` for providers (behind the seam)
- YAML: `js-yaml`; Markdown: `marked` or `remark`
- Iteration speed: hot reload, fast test runner
- Distribution: single binary per platform via `bun build --compile`

*Option B — Python + uv:*
- `PyApp` or `uv run` distributions for single-binary-ish shipping
- `sqlite3` in stdlib; `apsw` for advanced FTS5 usage
- `anthropic`, `openai` for providers
- YAML: `pyyaml`; Markdown: `mistune` or `markdown-it-py`
- Iteration speed: fast; slightly slower cold start
- Distribution: `uv` bundle or PyInstaller; more complex than Bun

*Option C — Go:*
- `go build` produces a single static binary trivially
- `modernc.org/sqlite` (pure Go, no CGo) for zero-dependency SQLite
- HTTP client in stdlib; provider SDKs via community libraries
- YAML: `go-yaml`; Markdown: `goldmark`
- Iteration speed: fast compile; excellent tooling
- Distribution: dead simple (`GOOS=darwin GOARCH=arm64 go build`)

*Recommendation order:* Go > TypeScript/Bun > Python, based on: Go's trivial single-binary cross-compilation, no runtime requirement, and well-suited CLI ecosystem. TypeScript/Bun wins if the builder has deep TypeScript fluency. Python wins only if the builder's fluency is Python-exclusive. The architecture is language-agnostic; the recommendation is velocity-pragmatic.

*Confidence:* High on criteria. Language choice is human-reviewed.

---

## 17. IMPLEMENTATION PLANNING FOUNDATION

### Epics

| Epic | Description | Modules | Weeks |
|---|---|---|---|
| **E-RECORD** | Define and implement the Record: format spec, append, read, list, correct, export | Record (Module 4) | 1–2 |
| **E-INDEX** | Build the SQLite index: FTS5, metadata table, embeddings, rebuild | Index (Module 5) | 2 |
| **E-RECALL** | Build recall: keyword + semantic query, synthesis call, honesty gate | Recall (Module 6) + Intelligence (seam) | 3 |
| **E-CAPTURE** | Build capture loop: context assembly, AI drafting, editor confirmation | Context Assembler (M2) + Intelligence (M3) + Surface (M1) | 3–4 |
| **E-DOGFOOD** | Use the tool on itself from day one. Track captures, recalls, friction. | All modules | 1–6 |
| **E-E3** | Run E3 (zero-AI) weekly in CI | All modules | ongoing |

### Modules × Capabilities × Dependency Order

```
Week 1 (parallel tracks possible after day 3):

[Day 1–3] RECORD FORMAT DESIGN — single-threaded, no code yet
  → Write RECORD_FORMAT.md
  → E5 informal trial: can you read what you wrote without the tool?
  → Lock format version 1

[Day 4–5] RECORD IMPLEMENTATION (Track A)    |  [Day 4–5] INTELLIGENCE SEAM (Track B)
  → RecordPort interface defined               |    → IntelligencePort interface defined
  → File I/O: append, read, list              |    → NullAdapter implemented
  → EntryId generator (D-YYYY-MM-DD-NNN)     |    → AnthropicAdapter skeleton
  → Append-only enforcement + test            |    → Provider selection from config

Week 2:

[Track A] INDEX                              |  [Track B] CONTEXT ASSEMBLER
  → SQLite schema (entries, entries_fts,     |    → git CLI wrapper (diff, log, files)
    embeddings, evidence_links)              |    → Context budget enforcement
  → FTS5 indexing from file scan            |    → DecisionContext struct/type
  → `oip rebuild-index` command             |    → Optional: GitHub API for PR metadata
  → CI test: E3 (zero-AI, index rebuild)

Week 3:

RECALL ENGINE (depends on Track A weeks 1–2)
  → FTS search implementation
  → In-process cosine similarity (embeddings)
  → Merge + rank FTS + semantic results
  → Honesty gate (low-similarity warning)
  → Silence response ("Record is silent on this")
  → `oip recall "<query>"` command

DRAFTING INTELLIGENCE (depends on Track B weeks 1–2)
  → AnthropicAdapter: draft() implementation
  → AnthropicAdapter: embed() implementation
  → Synthesis call for recall answers
  → Context-to-prompt construction
  → Token budget enforced before API call
  → E3 test: NullAdapter draft → blank template

Week 4:

CAPTURE LOOP (depends on weeks 1–3, both tracks)
  → Assemble context on `oip capture` invocation
  → Call IntelligencePort.draft()
  → Write draft to temp file
  → Open $EDITOR (like `git commit -e`)
  → On save: validate YAML frontmatter, assign EntryId, call RecordPort.append()
  → On editor abort (empty file): discard, log message
  → `oip capture --manual`: skip AI, open blank template
  → Optional: post-merge git hook (opt-in, emits reminder)

Week 5:

SEMANTIC RECALL + POLISH
  → AnthropicAdapter: embed() wired into index build
  → Recall: semantic search replacing zero-vector fallback
  → `oip recall` now uses FTS + semantic, merged
  → Synthesis: cited answers with [D-YYYY-MM-DD-NNN] references
  → E5 (decade-reader test) — outsider reads entries/ without tool
  → `oip export --format=json` — exit right verified
  → `oip browse` — basic pager output

Week 6:

DOGFOOD EVALUATION + METRICS
  → All V1 design decisions are in the tool's own Record (self-hosting)
  → E1 measurement: confirm times, edit distances
  → E2 measurement: recall events, saved re-litigation
  → E3 scripted run in CI confirmed passing
  → Fix critical UX friction (one sprint reserved entirely for dogfood fixes)
  → RECORD_FORMAT.md finalized and published (E5 artifact)
  → Ring 1 onboarding materials written from Record entries
```

### Blockers

| Blocker | Blocks | Resolution |
|---|---|---|
| Record format not finalized | All modules (nothing can read/write until format is stable) | Day 1–3 dedicated to format. No code until format is signed off. |
| IntelligencePort interface not defined | Drafting, Embedding, Synthesis | Define interface before any adapter implementation. Seam first, adapter second. |
| AI provider API key for Anthropic/OpenAI | Drafting, Embedding | Use OllamaAdapter for first 2 weeks of development (zero cost, local). Swap to production provider for E1 testing. |
| E5 failure (format not decade-readable) | Format freeze | E5 blocks format freeze. Run in week 5; fix format issues before proceeding to Ring 1. |

### Human Review Gates

| Gate | Timing | What is reviewed | Pass criterion |
|---|---|---|---|
| **Format Gate** | End of week 1 | RECORD_FORMAT.md — is it niche-neutral? Amendable? Decade-readable? | Niche-neutrality check (no software-specific fields in core schema); format_version: 1 present; E5 informal pass |
| **Seam Gate** | End of week 1 (Track B) | IntelligencePort interface — is it narrow enough? Does the escape hatch exist? Are any provider types present? | Three methods + raw_call; zero provider-specific types; NullAdapter fully implements the interface |
| **Capture UX Gate** | End of week 4 | Is `oip capture` actually invokable in < 30 seconds? Does the editor open correctly? Is manual path usable? | Dogfood: make 5 captures; time them; median < 90s (buffer above the 60s target) |
| **Decade-Reader Gate (E5)** | Week 5 | Outsider reads `entries/` without the tool in < 1 hour | Outsider completes test successfully; format is not modified after this gate |
| **E3 Gate** | Ongoing from week 2 | CI run with NullAdapter: all core commands succeed or degrade gracefully | Zero crashes or blocking failures with intelligence disabled |
| **Ring 0 Exit Gate** | End of week 6 | E1 metrics passing? E2 metrics emerging? E3 scripted CI pass? Format frozen? | E1: ≥5 captures from dogfood past week 4; E3: CI green; E5: passed; at least 10 Record entries in the tool's own Record |

### AI Execution Candidates

Tasks in implementation where AI tooling directly accelerates work:

| Task | AI role | Tool |
|---|---|---|
| Drafting the `IntelligencePort` interface and adapters | Generate adapter boilerplate from interface definition | Claude Code (this session) |
| Drafting `RECORD_FORMAT.md` v1 | Draft the format spec document for human review | Claude Code |
| Writing the SQLite schema and FTS5 setup | Generate schema + migration script | Claude Code |
| Context assembly prompt design | Draft and iterate on the system prompt for the drafting adapter | Claude Code + empirical testing |
| Writing E3 CI test scripts | Generate scripted test against NullAdapter | Claude Code |
| Initial dogfood entries (weeks 1–6) | The tool drafts its own design decisions | The tool itself (self-hosting) |

---

## FINAL VERDICT

> **If one experienced engineer had six weeks to build the first production-ready foundation, which architecture maximizes long-term leverage while minimizing irreversible mistakes, and why?**

**The architecture is:** A single CLI binary, six modules, one durable artifact, and one intelligence seam — all organized so that the only irreversible decision (the entry format) absorbs all the design time, and every other decision is either trivially reversible or made obsolete by mature infrastructure.

**The architecture that maximizes long-term leverage** is the one that answers "yes" to every substrate-replacement test (Article 47) on day one: Is the AI provider replaceable? Yes — one config key. Is the index replaceable? Yes — it's a git-ignored SQLite cache that rebuilds from files in seconds. Is Git as sync replaceable? Yes — the transport is behind a seam, monitored by E7. Is the format replaceable? No — and it shouldn't be. The format is the point.

**The irreversible mistake to minimize** is a Record format that leaks the first team's shape into the schema, or that requires the tool to be intelligible. Both are prevented by: (a) the niche-neutrality rule (T3), which bars any software-team-specific concept from the core schema, and (b) the decade-reader gate (E5), which fires before the format is frozen, with a human outsider as the test.

**The six-week sequence that minimizes regret:**

1. **Weeks 1 (days 1–3):** Design the format. No code. Run E5 informally. Sign off. This is the most expensive three days in the project's life; they compound for years.

2. **Week 1 (days 4–5) through week 2:** Build the Record and Index in parallel with the IntelligencePort seam and ContextAssembler. Both tracks can run simultaneously; they merge at week 3.

3. **Week 3:** Wire the Recall engine (FTS + placeholder cosine) and the Drafting adapter together. Run E3 in CI for the first time. If E3 fails (anything requires a real AI call), fix it immediately — the architecture is compromised.

4. **Week 4:** Close the capture loop. Editor opens, human confirms, entry appends, index updates. Make 5 captures of your own decisions to validate the UX.

5. **Week 5:** Add semantic recall and run E5 (the real outsider test). The format does not freeze until E5 passes.

6. **Week 6:** Dogfood metrics. Fix what the metrics reveal. The tool's own design decisions fill its own Record. This is Ring 0. If E1 and E2 metrics are passing, Ring 1 is ready.

**Why this architecture:**

Every long-lived system studied — UNIX, SQL, Git, PostgreSQL — centers a minimal, stable representation of truth and treats all engines as replaceable. The format is the UNIX inode. The Index is the filesystem cache. The IntelligencePort seam is the device driver. The surfaces are shells.

The platforms that failed — BPM suites, RPA products, knowledge management systems — failed because they built on the wrong invariant: the process, the automation, the document, the model. The OIP builds on the only invariant that has survived every technology revolution the Constitution studied: the organization's need to remember why it decided.

A format that a stranger can read without the tool, stored in files the organization owns, indexed by a database that rebuilds in seconds, searched by AI that is never load-bearing — this architecture has a thirty-year argument for its own existence. Build it. Start with the format. Let everything else earn its place one confirmed decision at a time.

---

*End of Engineering Architecture Blueprint. Amendment procedure: every EDR carries a confidence level and future-impact assessment. Amendments append; they do not rewrite. This document keeps its own record.*
