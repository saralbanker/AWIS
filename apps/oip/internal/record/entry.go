package record

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // register the SQLite driver
)

// Entry holds the TDS-06 record fields for a single decision entry.
type Entry struct {
	Title                string
	Date                 string // YYYY-MM-DD
	Status               string // "decided" or "superseded"
	Tags                 []string
	Corrects             string // ID of prior entry or ""
	Provenance           Provenance
	Distinguishes        Distinguishes
	Decision             string
	Rationale            string
	RejectedAlternatives string
	Unknowns             string
}

// Provenance holds the attribution block (Art. 9).
type Provenance struct {
	Origin     string
	Authority  string
	Sources    []string
	Confidence string
}

// Distinguishes holds the three honesty registers (Art. 10).
type Distinguishes struct {
	Observation string
	Description string
	Intention   string
}

// Writer manages TDS-06 entries in a record root directory.
type Writer struct {
	root string // absolute path to the record root (e.g. working dir)
}

// NewWriter creates a Writer rooted at root. The root must exist.
func NewWriter(root string) *Writer {
	return &Writer{root: root}
}

// entriesDir returns the path to .decisions/entries/.
func (w *Writer) entriesDir() string {
	return filepath.Join(w.root, ".decisions", "entries")
}

// indexDir returns the path to .decisions/.index/.
func (w *Writer) indexDir() string {
	return filepath.Join(w.root, ".decisions", ".index")
}

// dbPath returns the path to oip.db.
func (w *Writer) dbPath() string {
	return filepath.Join(w.indexDir(), "oip.db")
}

// nextID computes the next per-day NNN counter by scanning existing files.
// It never overwrites: if D-date-NNN.md exists, it advances NNN.
func (w *Writer) nextID(date string) (string, error) {
	dir := w.entriesDir()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("record: scan entries dir: %w", err)
	}

	prefix := "D-" + date + "-"
	max := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".md") {
			continue
		}
		nnnStr := strings.TrimPrefix(strings.TrimSuffix(name, ".md"), prefix)
		n, err := strconv.Atoi(nnnStr)
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}

	nnn := max + 1
	return fmt.Sprintf("D-%s-%03d", date, nnn), nil
}

// Append writes a new TDS-06 entry to disk and indexes it in oip.db.
// It returns the generated entry ID. The operation is:
//  1. Compute next NNN for today's date.
//  2. Render and write the .md file (NEVER overwrites).
//  3. Open/create oip.db and insert into entries + entries_fts.
//
// If the db insert fails after the file write, the error is returned (the
// rebuild-index command repairs the database in this case).
func (w *Writer) Append(ctx context.Context, e Entry) (string, error) {
	date := e.Date
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}

	// Ensure directories exist.
	if err := os.MkdirAll(w.entriesDir(), 0o755); err != nil {
		return "", fmt.Errorf("record: mkdir entries: %w", err)
	}
	if err := os.MkdirAll(w.indexDir(), 0o755); err != nil {
		return "", fmt.Errorf("record: mkdir index: %w", err)
	}

	id, err := w.nextID(date)
	if err != nil {
		return "", err
	}

	// Render the .md file.
	content := renderEntry(id, date, e)
	mdPath := filepath.Join(w.entriesDir(), id+".md")

	// Append-only: fail if file exists (advance NNN logic handles races).
	f, err := os.OpenFile(mdPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			// Collision: advance NNN by re-scanning.
			return w.Append(ctx, e)
		}
		return "", fmt.Errorf("record: create entry file: %w", err)
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("record: write entry file: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("record: close entry file: %w", err)
	}

	// Index into oip.db.
	relPath, err := filepath.Rel(w.root, mdPath)
	if err != nil {
		relPath = mdPath
	}
	if err := w.insertDB(ctx, id, relPath, date, e.Tags, content); err != nil {
		return id, fmt.Errorf("record: db insert (file written; run rebuild-index to repair): %w", err)
	}

	return id, nil
}

// insertDB opens oip.db and inserts a row into entries and entries_fts.
func (w *Writer) insertDB(ctx context.Context, id, path, date string, tags []string, content string) error {
	db, err := openDB(w.dbPath())
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if err := ensureSchema(db); err != nil {
		return err
	}

	tagsStr := strings.Join(tags, ",")
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err = db.ExecContext(ctx,
		`INSERT INTO entries(id, path, created_at, tags) VALUES(?,?,?,?)`,
		id, path, createdAt, tagsStr)
	if err != nil {
		return fmt.Errorf("record: insert entries: %w", err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO entries_fts(id, content) VALUES(?,?)`,
		id, content)
	if err != nil {
		return fmt.Errorf("record: insert entries_fts: %w", err)
	}
	return nil
}

// RebuildIndex drops and recreates the entries and entries_fts tables from
// all .md files in .decisions/entries/. This implements Art. 5: Record
// irreplaceable; index disposable.
func (w *Writer) RebuildIndex(ctx context.Context) error {
	db, err := openDB(w.dbPath())
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Drop and recreate.
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS entries_fts`); err != nil {
		return fmt.Errorf("record: rebuild: drop fts: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS entries`); err != nil {
		return fmt.Errorf("record: rebuild: drop entries: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS entry_vectors`); err != nil {
		return fmt.Errorf("record: rebuild: drop vectors: %w", err)
	}
	if err := ensureSchema(db); err != nil {
		return err
	}

	dir := w.entriesDir()
	mdFiles, err := filepath.Glob(filepath.Join(dir, "D-*.md"))
	if err != nil {
		return fmt.Errorf("record: rebuild: glob: %w", err)
	}
	sort.Strings(mdFiles)

	for _, mdFile := range mdFiles {
		base := filepath.Base(mdFile)
		id := strings.TrimSuffix(base, ".md")

		content, err := os.ReadFile(mdFile)
		if err != nil {
			return fmt.Errorf("record: rebuild: read %s: %w", base, err)
		}

		// Parse date and tags from frontmatter for entries table.
		date, tags := parseFrontmatterBasic(string(content))
		if date == "" {
			// Derive date from ID: D-YYYY-MM-DD-NNN
			parts := strings.Split(id, "-")
			if len(parts) >= 4 {
				date = strings.Join(parts[1:4], "-")
			}
		}

		relPath, err := filepath.Rel(w.root, mdFile)
		if err != nil {
			relPath = mdFile
		}
		tagsStr := strings.Join(tags, ",")
		createdAt := date + "T00:00:00Z"

		_, err = db.ExecContext(ctx,
			`INSERT INTO entries(id, path, created_at, tags) VALUES(?,?,?,?)`,
			id, relPath, createdAt, tagsStr)
		if err != nil {
			return fmt.Errorf("record: rebuild: insert entries %s: %w", id, err)
		}
		_, err = db.ExecContext(ctx,
			`INSERT INTO entries_fts(id, content) VALUES(?,?)`,
			id, string(content))
		if err != nil {
			return fmt.Errorf("record: rebuild: insert fts %s: %w", id, err)
		}
	}
	return nil
}

// ReadEntryContent reads the full content of an entry by ID.
func (w *Writer) ReadEntryContent(id string) (string, error) {
	mdPath := filepath.Join(w.entriesDir(), id+".md")
	b, err := os.ReadFile(mdPath)
	if err != nil {
		return "", fmt.Errorf("record: read entry %s: %w", id, err)
	}
	return string(b), nil
}

// parseFrontmatterBasic extracts date and tags from a YAML frontmatter block.
// It uses simple line scanning to avoid a YAML dependency in this package.
func parseFrontmatterBasic(content string) (date string, tags []string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	inFrontmatter := false
	inTags := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break // end of frontmatter
		}
		if !inFrontmatter {
			continue
		}
		if strings.HasPrefix(line, "date:") {
			date = strings.TrimSpace(strings.TrimPrefix(line, "date:"))
			date = strings.Trim(date, `"`)
			inTags = false
		} else if strings.HasPrefix(line, "tags:") {
			inTags = true
		} else if inTags && strings.HasPrefix(line, "  - ") {
			tag := strings.TrimPrefix(line, "  - ")
			tags = append(tags, strings.TrimSpace(tag))
		} else if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inTags = false
		}
	}
	return date, tags
}

// openDB opens (or creates) the SQLite database at path using the modernc driver.
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("record: open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// ensureSchema creates the tables if they don't exist.
func ensureSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS entries (
			id         TEXT PRIMARY KEY,
			path       TEXT NOT NULL,
			created_at TEXT NOT NULL,
			tags       TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts
		USING fts5(id UNINDEXED, content, tokenize='porter ascii')`,
		`CREATE TABLE IF NOT EXISTS entry_vectors (
			id      TEXT PRIMARY KEY REFERENCES entries(id),
			vector  BLOB NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("record: schema: %w", err)
		}
	}
	return nil
}

// renderEntry produces the TDS-06 .md file content for an entry.
func renderEntry(id, date string, e Entry) string {
	var b strings.Builder

	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", id)
	fmt.Fprintf(&b, "date: \"%s\"\n", date)
	fmt.Fprintf(&b, "title: %q\n", e.Title)
	status := e.Status
	if status == "" {
		status = "decided"
	}
	fmt.Fprintf(&b, "status: %s\n", status)

	if len(e.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, t := range e.Tags {
			fmt.Fprintf(&b, "  - %s\n", t)
		}
	} else {
		b.WriteString("tags: []\n")
	}

	corrects := e.Corrects
	if corrects == "" {
		b.WriteString("corrects: null\n")
	} else {
		fmt.Fprintf(&b, "corrects: %s\n", corrects)
	}

	b.WriteString("provenance:\n")
	fmt.Fprintf(&b, "  origin: %s\n", e.Provenance.Origin)
	fmt.Fprintf(&b, "  authority: %s\n", e.Provenance.Authority)
	if len(e.Provenance.Sources) > 0 {
		b.WriteString("  sources:\n")
		for _, s := range e.Provenance.Sources {
			fmt.Fprintf(&b, "    - %q\n", s)
		}
	} else {
		b.WriteString("  sources: []\n")
	}
	fmt.Fprintf(&b, "  confidence: %s\n", e.Provenance.Confidence)

	b.WriteString("distinguishes:\n")
	obs := e.Distinguishes.Observation
	if obs == "" {
		b.WriteString("  observation: null\n")
	} else {
		fmt.Fprintf(&b, "  observation: %q\n", obs)
	}
	desc := e.Distinguishes.Description
	if desc == "" {
		b.WriteString("  description: null\n")
	} else {
		fmt.Fprintf(&b, "  description: %q\n", desc)
	}
	intent := e.Distinguishes.Intention
	if intent == "" {
		b.WriteString("  intention: null\n")
	} else {
		fmt.Fprintf(&b, "  intention: %q\n", intent)
	}
	b.WriteString("---\n\n")

	b.WriteString("## Decision\n\n")
	b.WriteString(e.Decision)
	b.WriteString("\n\n")
	b.WriteString("## Rationale\n\n")
	b.WriteString(e.Rationale)
	b.WriteString("\n\n")
	b.WriteString("## Rejected Alternatives\n\n")
	b.WriteString(e.RejectedAlternatives)
	b.WriteString("\n\n")
	b.WriteString("## Unknowns\n\n")
	b.WriteString(e.Unknowns)
	b.WriteString("\n")

	return b.String()
}

// ParseEntryID checks whether a string matches the D-YYYY-MM-DD-NNN pattern.
func ParseEntryID(s string) bool {
	parts := strings.Split(s, "-")
	// D-YYYY-MM-DD-NNN → 5 parts
	if len(parts) != 5 {
		return false
	}
	if parts[0] != "D" {
		return false
	}
	if len(parts[1]) != 4 || len(parts[2]) != 2 || len(parts[3]) != 2 || len(parts[4]) != 3 {
		return false
	}
	_, errY := strconv.Atoi(parts[1])
	_, errM := strconv.Atoi(parts[2])
	_, errD := strconv.Atoi(parts[3])
	_, errN := strconv.Atoi(parts[4])
	return errY == nil && errM == nil && errD == nil && errN == nil
}
