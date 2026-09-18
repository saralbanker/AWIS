package sdk

import (
	"context"
	"errors"
	"testing"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// submitN registers the hello workflow and submits n instances, returning
// their ids in submission order.
func submitN(t *testing.T, rt *Runtime, ns string, n int) []core.InstanceID {
	t.Helper()
	ctx := context.Background()

	if err := rt.RegisterHandler(&greetHandler{}); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}
	if err := rt.RegisterWorkflow(buildHelloWorkflow(ns)); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	ids := make([]core.InstanceID, 0, n)
	for i := 0; i < n; i++ {
		iid, err := rt.Submit(ctx, "hello", map[string]any{"name": "n"})
		if err != nil {
			t.Fatalf("Submit %d: %v", i, err)
		}
		ids = append(ids, iid)
	}
	return ids
}

// TestListPagedCoversEveryInstanceExactlyOnce is the core pagination
// guarantee: walking the offsets must yield every row once, with no gaps and
// no repeats.
//
// This is the assertion that catches an unstable sort. started_at is not
// unique — several instances submitted in the same tick share a timestamp —
// so without the instance_id tiebreak, SQLite is free to order ties
// differently between the two queries that produce two adjacent pages, and a
// row silently vanishes or appears twice.
func TestListPagedCoversEveryInstanceExactlyOnce(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	const total = 25
	want := submitN(t, rt, "test", total)

	seen := make(map[core.InstanceID]int, total)
	var pages int
	for offset := 0; ; offset += 4 {
		page, err := rt.ListPaged(ctx, core.InstanceFilter{}, 4, offset)
		if err != nil {
			t.Fatalf("ListPaged(offset=%d): %v", offset, err)
		}
		if page.Total != total {
			t.Errorf("page.Total = %d, want %d", page.Total, total)
		}
		if len(page.Instances) == 0 {
			break
		}
		for _, inst := range page.Instances {
			seen[inst.InstanceID]++
		}
		pages++
		if pages > total {
			t.Fatal("pagination did not terminate")
		}
	}

	if len(seen) != total {
		t.Errorf("saw %d distinct instances across all pages, want %d", len(seen), total)
	}
	for _, id := range want {
		switch seen[id] {
		case 1: // exactly once, as required
		case 0:
			t.Errorf("instance %s never appeared on any page", id)
		default:
			t.Errorf("instance %s appeared %d times across pages", id, seen[id])
		}
	}
}

// TestListPagedReportsAppliedLimit: storage normalises a non-positive limit
// and clamps an oversized one, so a caller must be told what was actually
// applied rather than assuming its request was honoured verbatim.
func TestListPagedReportsAppliedLimit(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	submitN(t, rt, "test", 3)

	cases := []struct {
		name      string
		requested int
		want      int
	}{
		{"zero takes the default", 0, storageDefaultPageSize},
		{"negative takes the default", -5, storageDefaultPageSize},
		{"in range is honoured", 2, 2},
		{"oversized is clamped", storageMaxPageSize * 10, storageMaxPageSize},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, err := rt.ListPaged(ctx, core.InstanceFilter{}, tc.requested, 0)
			if err != nil {
				t.Fatalf("ListPaged: %v", err)
			}
			if page.Limit != tc.want {
				t.Errorf("page.Limit = %d, want %d", page.Limit, tc.want)
			}
		})
	}
}

// TestSDKPageBoundsMatchStorage pins the duplicated bounds. sdk must not
// import the concrete storage adapter, so it restates the limits; if storage
// changed them and this did not, ListPaged would report a limit it never
// applied.
func TestSDKPageBoundsMatchStorage(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	// storageDefaultPageSize+1 instances proves the default really is the
	// boundary storage applies, rather than a number this package invented.
	submitN(t, rt, "test", storageDefaultPageSize+1)

	page, err := rt.ListPaged(ctx, core.InstanceFilter{}, 0, 0)
	if err != nil {
		t.Fatalf("ListPaged: %v", err)
	}
	if len(page.Instances) != storageDefaultPageSize {
		t.Errorf("default page returned %d rows, want %d — sdk's copy of the "+
			"default has drifted from storage's", len(page.Instances), storageDefaultPageSize)
	}
	if page.Total != storageDefaultPageSize+1 {
		t.Errorf("page.Total = %d, want %d", page.Total, storageDefaultPageSize+1)
	}
}

// TestListPagedHasMore covers the pagination-control helper at both
// boundaries, including the exactly-full final page.
func TestListPagedHasMore(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	submitN(t, rt, "test", 6)

	first, err := rt.ListPaged(ctx, core.InstanceFilter{}, 4, 0)
	if err != nil {
		t.Fatalf("ListPaged: %v", err)
	}
	if !first.HasMore() {
		t.Error("first page of 4/6: HasMore() = false, want true")
	}

	last, err := rt.ListPaged(ctx, core.InstanceFilter{}, 4, 4)
	if err != nil {
		t.Fatalf("ListPaged: %v", err)
	}
	if last.HasMore() {
		t.Error("last page (offset 4 of 6): HasMore() = true, want false")
	}

	// Exactly-full final page: offset 3 + limit 3 == total 6, so nothing follows.
	exact, err := rt.ListPaged(ctx, core.InstanceFilter{}, 3, 3)
	if err != nil {
		t.Fatalf("ListPaged: %v", err)
	}
	if exact.HasMore() {
		t.Error("exactly-full final page: HasMore() = true, want false")
	}
}

// TestListPagedFilterMatchesUnpagedList: the paged and unpaged reads must
// apply identical predicates, or a GUI and the engine would disagree about
// what exists.
func TestListPagedFilterMatchesUnpagedList(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	submitN(t, rt, "test", 5)

	filters := []core.InstanceFilter{
		{},
		{Namespace: "test"},
		{Namespace: "nonexistent"},
		{Status: core.InstanceStatusRunning},
	}
	for _, f := range filters {
		unpaged, err := rt.List(ctx, f)
		if err != nil {
			t.Fatalf("List(%+v): %v", f, err)
		}
		page, err := rt.ListPaged(ctx, f, storageMaxPageSize, 0)
		if err != nil {
			t.Fatalf("ListPaged(%+v): %v", f, err)
		}
		if page.Total != len(unpaged) {
			t.Errorf("filter %+v: ListPaged Total = %d, List returned %d", f, page.Total, len(unpaged))
		}
		if len(page.Instances) != len(unpaged) {
			t.Errorf("filter %+v: ListPaged returned %d rows, List returned %d",
				f, len(page.Instances), len(unpaged))
		}
	}
}

// TestListPagedOffsetPastEndIsEmptyNotAnError mirrors SQL LIMIT/OFFSET
// semantics — a GUI walking off the end must get an empty page, not a failure.
func TestListPagedOffsetPastEndIsEmptyNotAnError(t *testing.T) {
	ctx := context.Background()
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	submitN(t, rt, "test", 2)

	page, err := rt.ListPaged(ctx, core.InstanceFilter{}, 10, 500)
	if err != nil {
		t.Fatalf("ListPaged past end: %v", err)
	}
	if len(page.Instances) != 0 {
		t.Errorf("got %d rows past the end, want 0", len(page.Instances))
	}
	if page.Total != 2 {
		t.Errorf("page.Total = %d, want 2 (the total is independent of the offset)", page.Total)
	}
	if page.HasMore() {
		t.Error("HasMore() = true past the end of the result set")
	}
}

// nonPagedStore is a StoragePort that does NOT implement pagedInstanceStore.
type nonPagedStore struct{ core.StoragePort }

// TestListPagedRefusesRatherThanSilentlyFallingBack: a storage without the
// paginated read model must produce a typed error. Falling back to an
// unbounded read + slice would read the whole table to return one page —
// silently reintroducing exactly the cost the caller asked to avoid.
func TestListPagedRefusesRatherThanSilentlyFallingBack(t *testing.T) {
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: nonPagedStore{s}, WorkerID: "w1"})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	_, err = rt.ListPaged(context.Background(), core.InstanceFilter{}, 10, 0)
	if !errors.Is(err, ErrPaginationUnsupported) {
		t.Fatalf("err = %v, want ErrPaginationUnsupported", err)
	}
}

// Compile-time guard: the real adapter must satisfy the capability, or
// ListPaged silently degrades to the error path for every caller.
var _ pagedInstanceStore = (*storage.SQLiteStorage)(nil)
