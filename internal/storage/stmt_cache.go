package storage

// Prepared-statement cache for the single-row reads on the engine tick path
// (M18-C4; NFR-P-01).
//
// database/sql hands every ad-hoc query to the SQLite parser again. The tick
// reads one row per running instance (GetInstance, CancellationRequested), so
// a tick over parked instances spent about a third of its time re-parsing SQL
// text that never changes. A statement prepared once against the *sql.DB and
// reused runs the same SQL with the same arguments, so results and errors do
// not change; only the parse is skipped.
//
// Only fixed SQL text belongs here. Entries are keyed by the query string and
// are never evicted, so dynamic SQL (such as ListInstances' filter predicate)
// stays on the ad-hoc path. DB.Close closes every cached statement before it
// closes the *sql.DB.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

// stmtCache maps fixed SQL text to a *sql.Stmt prepared on db. It is safe for
// concurrent use.
type stmtCache struct {
	db     *sql.DB
	mu     sync.Mutex
	stmts  map[string]*sql.Stmt
	closed bool
}

func newStmtCache(db *sql.DB) *stmtCache {
	return &stmtCache{db: db, stmts: make(map[string]*sql.Stmt)}
}

// get returns the cached statement for query, preparing it on first use. It
// returns nil when the cache is closed or when preparation fails. A nil result
// is not reported as an error here: the caller then runs query ad hoc, so any
// failure surfaces from that query with the same text it had before the cache
// existed.
//
// Preparing happens under mu. The pool holds one connection (DB.Open sets
// MaxOpenConns to 1), so a prepare that must wait for it waits for the same
// connection every other caller waits for, and no new lock order is introduced.
func (c *stmtCache) get(ctx context.Context, query string) *sql.Stmt {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	if st, ok := c.stmts[query]; ok {
		return st
	}
	st, err := c.db.PrepareContext(ctx, query)
	if err != nil {
		return nil
	}
	c.stmts[query] = st
	return st
}

// close closes every cached statement and disables the cache, so later get
// calls return nil. It is safe to call more than once.
func (c *stmtCache) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	var errs []error
	for query, st := range c.stmts {
		if err := st.Close(); err != nil {
			errs = append(errs, fmt.Errorf("storage: close prepared statement: %w", err))
		}
		delete(c.stmts, query)
	}
	return errors.Join(errs...)
}

// queryRow runs a single-row read through the statement cache when a statement
// is available, and otherwise on the database with the same SQL and arguments.
// The returned row scans exactly as the ad-hoc QueryRowContext row would.
func (d *DB) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if st := d.stmts.get(ctx, query); st != nil {
		return st.QueryRowContext(ctx, args...)
	}
	return d.db.QueryRowContext(ctx, query, args...)
}
