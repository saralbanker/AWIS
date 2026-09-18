# OIP Database Schema — oip.db

## Ownership and Location (B5 Option A)

OIP owns its own application SQLite database at `.decisions/.index/oip.db`.
This path is relative to the configured record root (default: the working directory).
The database is **git-ignored** (see root `.gitignore`).

```
.decisions/
├── entries/               <- The Record (plain .md files, in git)
│   └── D-2026-07-10-001.md
└── .index/
    └── oip.db             <- OIP's application SQLite (git-ignored)

.awis/
└── runtime.db             <- AWIS's execution SQLite (git-ignored, separate)
```

The `.md` files are the source of truth. `oip.db` is a **rebuildable index**: if it is
lost or corrupted, `oip rebuild-index` repopulates it from `.decisions/entries/*.md`.
No data is permanently stored only in `oip.db`.

---

## Schema

### Table: `entries`

Metadata index for appended record entries.

```sql
CREATE TABLE IF NOT EXISTS entries (
    id         TEXT PRIMARY KEY,
    path       TEXT NOT NULL,
    created_at TEXT NOT NULL,
    tags       TEXT NOT NULL DEFAULT ''
);
```

| Column | Type | Description |
|---|---|---|
| `id` | TEXT PK | Entry ID (e.g., `D-2026-07-10-001`) |
| `path` | TEXT | Relative path to the `.md` file under the record root |
| `created_at` | TEXT | ISO 8601 timestamp of append |
| `tags` | TEXT | Comma-separated tags (denormalized for FTS join simplicity) |

### Table: `entries_fts` (FTS5)

Full-text search index over entry content. Populated by `RecordAppendHandler` and
repopulated by `RebuildIndexHandler`.

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts
USING fts5(id UNINDEXED, content, tokenize='porter ascii');
```

| Column | Description |
|---|---|
| `id` | Entry ID (unindexed — used for join back to `entries`) |
| `content` | Full text of the `.md` file (frontmatter + body) |

FTS5 `porter ascii` tokenizer provides stemming for English-language decision text.

### Table: `entry_vectors`

Reserved for V2 semantic search. **EMPTY in V1** (CONTRA-3: no embeddings generated).
The table is created to avoid schema migration in V2 but contains no rows.

```sql
CREATE TABLE IF NOT EXISTS entry_vectors (
    id      TEXT PRIMARY KEY REFERENCES entries(id),
    vector  BLOB NOT NULL
);
```

| Column | Description |
|---|---|
| `id` | Entry ID (foreign key to `entries`) |
| `vector` | Float32Array BLOB (unused in V1) |

---

## Rebuild Path

`oip.db` is disposable. The rebuild procedure:

1. Drop (or delete) `oip.db`.
2. Run `oip rebuild-index` (or submit the `rebuild-index` AWIS workflow).
3. The handler scans `.decisions/entries/*.md`, parses each file, and inserts into
   `entries` and `entries_fts`.

This implements the principle: **Record irreplaceable; index disposable** (Art. 5).

---

## Cross-Database Isolation

OIP's `oip.db` and AWIS's `runtime.db` are independent SQLite files. There are no
cross-database joins. OIP queries `oip.db` directly through its own handlers.
AWIS's `StoragePort` interface is unchanged by M15.
