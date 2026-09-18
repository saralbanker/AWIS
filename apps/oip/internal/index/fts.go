package index

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // register the SQLite driver
)

// Result is one FTS match result.
type Result struct {
	ID      string
	Path    string
	Snippet string
	Rank    float64
}

// Querier performs FTS queries against oip.db.
type Querier struct {
	dbPath string
}

// NewQuerier creates a Querier using the oip.db at root/.decisions/.index/oip.db.
func NewQuerier(root string) *Querier {
	return &Querier{
		dbPath: filepath.Join(root, ".decisions", ".index", "oip.db"),
	}
}

// FTSQuery performs a ranked FTS5 MATCH query and returns ordered results.
// Each result includes the entry ID, path, a snippet, and the FTS rank.
func (q *Querier) FTSQuery(ctx context.Context, query string) ([]Result, error) {
	db, err := sql.Open("sqlite", q.dbPath)
	if err != nil {
		return nil, fmt.Errorf("index: open db: %w", err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)

	// FTS5 query: join entries_fts with entries for path, order by rank (lower = better).
	rows, err := db.QueryContext(ctx, `
		SELECT
			f.id,
			e.path,
			snippet(entries_fts, 1, '<b>', '</b>', '…', 20),
			f.rank
		FROM entries_fts f
		JOIN entries e ON e.id = f.id
		WHERE entries_fts MATCH ?
		ORDER BY f.rank
		LIMIT 20
	`, sanitizeFTSQuery(query))
	if err != nil {
		if isFTSNoTable(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("index: fts query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []Result
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.ID, &r.Path, &r.Snippet, &r.Rank); err != nil {
			return nil, fmt.Errorf("index: fts scan: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: fts rows: %w", err)
	}
	return results, nil
}

// SemanticPassthrough returns the FTS results unchanged. In V1 there are no
// vectors (CONTRA-3); semantic ranking is a pass-through.
func SemanticPassthrough(results []Result) []Result {
	return results
}

// sanitizeFTSQuery wraps the query string for safe FTS5 MATCH use.
// FTS5 interprets bare tokens as column names; wrap the whole query in
// double quotes to treat it as a phrase search, escaping any internal
// double-quote characters.
func sanitizeFTSQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	// Escape internal double-quotes and wrap in double-quotes for phrase search.
	q = strings.ReplaceAll(q, `"`, `""`)
	return `"` + q + `"`
}

// isFTSNoTable returns true if the error indicates the FTS table doesn't exist.
func isFTSNoTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such table") || strings.Contains(msg, "no table")
}
