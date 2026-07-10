# M15 — Implementation Spec
**Sources:** IMP §27.M15; Blueprint §10/§18/§29; Constitution Title II (Art. 5–13) + Title III;
Finalization B5 (oip.db ownership, Option A verbatim), B6; CONTRA-3 (semantic-rank
pass-through); IR-4; TDS-06 `apps/oip/docs/RECORD_FORMAT.md` (authored here; **founder
sign-off required — see disposition**); QG-3/QG-4; frozen fixtures
`apps/oip/workflows/*.yaml` (M10; register UNCHANGED).

## CE disposition (Fable, 2026-07-10)
TDS-06 sign-off and the G3 verdict are FOUNDER-ONLY. Under the founder's 2026-07-10 directive
("complete all milestones"), M15 builds to the gate: TDS-06 is authored and marked
**DRAFT — PENDING FOUNDER SIGN-OFF**; handlers implement it at-risk; nothing is irreversible
pre-merge (no production Record exists; format changes before merge are free). The E-MERGE
brief carries: TDS-06 for sign-off, the boundary evidence artifact, and the G3 verdict
request. G3's verdict itself is NOT rendered by any model.

## CE pins
- **Module boundary (QG-4, the milestone's reason):** `apps/oip` is module `github.com/awis/oip`
  (go.work member). It imports `github.com/awis/awis/sdk` (+ sdk/testing in tests) and its OWN
  deps — NOTHING under `github.com/awis/awis/internal`. Platform diff for the whole milestone
  = EMPTY outside apps/oip + docs/ledger (boundary artifact proves it).
- **oip.db (Finalization B5 Option A verbatim):** OIP owns `.decisions/.index/oip.db`;
  tables `entries` (id, path, created_at, tags), `entries_fts` (FTS5 over content),
  `entry_vectors` (BLOB; EMPTY in V1 — CONTRA-3). Driver: modernc.org/sqlite in apps/oip's
  own go.mod (application dependency, not platform).
- **TDS-06 Record format:** entry = `.decisions/entries/D-YYYY-MM-DD-NNN.md`; YAML frontmatter:
  `id, date, title, status(decided|superseded), tags[], corrects: <id|null>, provenance:
  {origin, authority, sources[], confidence}, distinguishes: {observation, description,
  intention}` + Markdown body sections `## Decision / ## Rationale / ## Rejected Alternatives
  / ## Unknowns` (Articles 7/9/10/11/13). Append-only: `corrects:` links a new entry to the
  corrected one; files are never rewritten (Art. 7). NNN = per-day counter, 3 digits.
- **Handlers (native, registered via sdk only):**
  - `RecordAppendHandler` (`oip.record.append`): inputs {title, body sections, tags,
    corrects?, provenance…} (from workflow variables/confirmed draft) → ONE operation: write
    .md THEN insert entries+entries_fts (file is source of truth; db is index — Art. 5;
    partial-failure rule: db insert failure after file write → error surfaced, rebuild-index
    repairs).
  - `IndexFTSHandler` (`oip.index.fts`): input {query} → FTS5 MATCH → ordered [{id, path,
    snippet, rank}].
  - `SemanticRankHandler` (`oip.index.semantic`): input {results} → V1 PASS-THROUGH unchanged
    (CONTRA-3; no vectors exist).
- **`rebuild-index` workflow** (YAML, apps/oip/workflows/): native step `oip.index.rebuild`
  handler — drops+rebuilds oip.db from the .md files (Art. 5: Record irreplaceable, index
  disposable).
- **Entrypoint** `apps/oip/cmd/oip/main.go` (or doc'd equivalent): opens runtime storage via
  sdk.SQLiteStorage, NewRuntime, registers the three handlers + rebuild handler + the two
  YAML workflow files unchanged (parse via… **sdk surface only** — if the sdk lacks YAML
  registration, register by reading the file through `sdk`-exposed dsl access; if none
  exists → the entrypoint shells `awis start` convention instead: document which path the
  sdk offers; a needed platform change is a STOP, not a fix).
- **QG-3 tests (harness, NullAdapter):** capture-decision: submit w/ inputs → plugin step
  (mock the git-context capability via a registered test plugin OR the real one when
  python3 present) → draft-entry intelligence step falls back (Null) → manual-entry path →
  WAIT `entry_confirmed` signal → record-append → entry file exists + FTS row; recall-decision:
  fts-search → semantic-rank pass-through → synthesize (Null-degraded) → results returned.
  Plus rebuild-index equivalence (delete oip.db → rebuild → recall still finds the entry).
- **CLI validation + boundary artifact (IMP Val):** real binary: start (discovers OIP YAMLs),
  plugin install git-context-plugin, submit capture-decision, signal entry_confirmed, recall
  finds entry; `git log --stat <milestone commits>` boundary proof saved to
  `docs/05-implementation/M15-oip-on-awis/BOUNDARY_EVIDENCE.md` (QG-4/G3 evidence, IMP §20.M15).

## Card split
C1: TDS-06 (DRAFT banner) + oip.db schema + apps/oip module scaffolding (go.mod deps, doc.go).
C2: handlers + rebuild workflow + entrypoint + QG-3 harness tests (the bulk).
C3: CLI e2e + BOUNDARY_EVIDENCE.md + G3 brief draft.
V1: verify incl. QG-4 empty-platform-diff proof.

## Non-scope
Real embeddings/semantic rank (V2); AnthropicAdapter (M16); `awis init` (M17); any platform
change whatsoever (that is a FAILED milestone per RB row, not a fix).
