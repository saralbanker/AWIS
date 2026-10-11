package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/awis/awis/internal/core"
)

// cacheLen returns the number of statements held by c.
func cacheLen(c *stmtCache) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.stmts)
}

// cacheCopy returns a copy of c's statements keyed by SQL text.
func cacheCopy(c *stmtCache) map[string]*sql.Stmt {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]*sql.Stmt, len(c.stmts))
	for q, st := range c.stmts {
		out[q] = st
	}
	return out
}

// TestStmtCache_ReusesStatementForFixedSQL: each fixed tick-path read prepares
// its statement once, and repeated calls reuse it.
func TestStmtCache_ReusesStatementForFixedSQL(t *testing.T) {
	ctx := context.Background()
	s, iid := seedInstance(t, core.InstanceStatusRunning)
	before := cacheLen(s.db.stmts)
	for i := 0; i < 3; i++ {
		if _, err := s.GetInstance(ctx, iid); err != nil {
			t.Fatalf("GetInstance %d: %v", i, err)
		}
	}
	if got := cacheLen(s.db.stmts); got != before+1 {
		t.Fatalf("after 3 GetInstance calls: %d cached statements, want %d", got, before+1)
	}
	for i := 0; i < 3; i++ {
		if _, err := s.CancellationRequested(ctx, iid); err != nil {
			t.Fatalf("CancellationRequested %d: %v", i, err)
		}
	}
	if got := cacheLen(s.db.stmts); got != before+2 {
		t.Fatalf("after CancellationRequested calls: %d cached statements, want %d", got, before+2)
	}
}

// TestStmtCache_CachedReadSeesLaterWrites: a reused statement reads the row as
// it is now, not as it was when the statement was prepared.
func TestStmtCache_CachedReadSeesLaterWrites(t *testing.T) {
	ctx := context.Background()
	s, iid := seedInstance(t, core.InstanceStatusRunning)
	if got, err := s.GetInstance(ctx, iid); err != nil || got.Status != core.InstanceStatusRunning {
		t.Fatalf("first GetInstance: status %q, err %v; want running", got.Status, err)
	}
	version, err := s.InstanceVersion(ctx, iid)
	if err != nil {
		t.Fatalf("InstanceVersion: %v", err)
	}
	next := core.WorkflowInstance{
		InstanceID:   iid,
		DefinitionID: "d",
		Namespace:    "t",
		Status:       core.InstanceStatusWaiting,
		CurrentSteps: []string{},
		Variables:    map[string]any{},
	}
	if err := s.UpsertInstance(ctx, next, version); err != nil {
		t.Fatalf("UpsertInstance: %v", err)
	}
	got, err := s.GetInstance(ctx, iid)
	if err != nil {
		t.Fatalf("GetInstance after write: %v", err)
	}
	if got.Status != core.InstanceStatusWaiting {
		t.Fatalf("GetInstance after write: status %q, want waiting", got.Status)
	}
}

// TestStmtCache_NotFoundErrorUnchanged: a missing row still reports
// ErrInstanceNotFound naming the instance, on the first call and on a reused
// statement.
func TestStmtCache_NotFoundErrorUnchanged(t *testing.T) {
	ctx := context.Background()
	s, _ := seedInstance(t, core.InstanceStatusRunning)
	missing := core.InstanceID("no-such-instance")
	for i := 0; i < 2; i++ {
		if _, err := s.GetInstance(ctx, missing); !errors.Is(err, ErrInstanceNotFound) || !strings.Contains(err.Error(), string(missing)) {
			t.Fatalf("GetInstance missing (call %d): err = %v, want ErrInstanceNotFound naming %s", i, err, missing)
		}
		if _, err := s.CancellationRequested(ctx, missing); !errors.Is(err, ErrInstanceNotFound) || !strings.Contains(err.Error(), string(missing)) {
			t.Fatalf("CancellationRequested missing (call %d): err = %v, want ErrInstanceNotFound naming %s", i, err, missing)
		}
	}
}

// TestStmtCache_ClosedDBErrorUnchanged: after DB.Close the tick-path reads fail
// with the database's own closed error, not a statement error and not
// ErrInstanceNotFound.
func TestStmtCache_ClosedDBErrorUnchanged(t *testing.T) {
	ctx := context.Background()
	s, iid := seedInstance(t, core.InstanceStatusRunning)
	if _, err := s.GetInstance(ctx, iid); err != nil {
		t.Fatalf("GetInstance before Close: %v", err)
	}
	if err := s.db.Close(); err != nil {
		t.Fatalf("DB.Close: %v", err)
	}
	if _, err := s.GetInstance(ctx, iid); err == nil || errors.Is(err, ErrInstanceNotFound) || !strings.Contains(err.Error(), "sql: database is closed") {
		t.Fatalf("GetInstance after Close: err = %v, want the closed-database error", err)
	}
	if _, err := s.CancellationRequested(ctx, iid); err == nil || errors.Is(err, ErrInstanceNotFound) || !strings.Contains(err.Error(), "sql: database is closed") {
		t.Fatalf("CancellationRequested after Close: err = %v, want the closed-database error", err)
	}
}

// TestStmtCache_CloseClosesStatements: DB.Close closes every cached statement
// (not merely forgets it), leaves the cache empty, and a second Close is a no-op.
func TestStmtCache_CloseClosesStatements(t *testing.T) {
	ctx := context.Background()
	s, iid := seedInstance(t, core.InstanceStatusRunning)
	if _, err := s.GetInstance(ctx, iid); err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if _, err := s.CancellationRequested(ctx, iid); err != nil {
		t.Fatalf("CancellationRequested: %v", err)
	}
	held := cacheCopy(s.db.stmts)
	if len(held) != 2 {
		t.Fatalf("held %d cached statements, want 2", len(held))
	}
	if err := s.db.Close(); err != nil {
		t.Fatalf("DB.Close: %v", err)
	}
	for q, st := range held {
		if err := st.QueryRowContext(ctx, string(iid)).Err(); err == nil || !strings.Contains(err.Error(), "statement is closed") {
			t.Fatalf("statement %q after Close: err = %v, want a closed-statement error", q, err)
		}
	}
	if n := cacheLen(s.db.stmts); n != 0 {
		t.Fatalf("after Close: %d cached statements remain, want 0", n)
	}
	if err := s.db.Close(); err != nil {
		t.Fatalf("second DB.Close: %v", err)
	}
}

// TestStmtCache_GetFallsBackOnPrepareError: text that does not prepare yields no
// cached statement (the caller then runs it ad hoc and reports its error); valid
// text is cached afterwards; a closed cache hands out nothing.
func TestStmtCache_GetFallsBackOnPrepareError(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	c := newStmtCache(db.db)
	t.Cleanup(func() { _ = c.close() })

	if st := c.get(ctx, "SELEC nonsense"); st != nil {
		t.Fatalf("get(invalid SQL) returned a statement; want nil so the caller falls back")
	}
	if n := cacheLen(c); n != 0 {
		t.Fatalf("after failed prepare: %d cached statements, want 0", n)
	}
	first := c.get(ctx, "SELECT 1")
	if first == nil {
		t.Fatalf("get(valid SQL) returned nil")
	}
	if again := c.get(ctx, "SELECT 1"); again != first {
		t.Fatalf("second get returned a different statement; want the cached one")
	}
	if n := cacheLen(c); n != 1 {
		t.Fatalf("after valid prepare: %d cached statements, want 1", n)
	}
	if err := c.close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if st := c.get(ctx, "SELECT 1"); st != nil {
		t.Fatalf("get after close returned a statement; want nil")
	}
}

// TestStmtCache_ConcurrentReads: concurrent tick-path reads each return the
// stored row and share one statement per query. make race runs this under -race.
func TestStmtCache_ConcurrentReads(t *testing.T) {
	ctx := context.Background()
	s, iid := seedInstance(t, core.InstanceStatusRunning)
	const workers, reads = 8, 25
	errc := make(chan error, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < reads; i++ {
				inst, err := s.GetInstance(ctx, iid)
				if err != nil {
					errc <- err
					return
				}
				if inst.Status != core.InstanceStatusRunning {
					errc <- fmt.Errorf("GetInstance status %q, want running", inst.Status)
					return
				}
				if _, err := s.CancellationRequested(ctx, iid); err != nil {
					errc <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		t.Error(err)
	}
	if n := cacheLen(s.db.stmts); n != 2 {
		t.Fatalf("after concurrent reads: %d cached statements, want 2", n)
	}
}
