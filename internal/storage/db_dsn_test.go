package storage

import "testing"

// TestBuildDSNCarriesImmediateLocking is a concurrency guard.
//
// Without _txlock=immediate, a read-then-write transaction (AppendEvent's
// SELECT MAX(sequence_num) followed by an INSERT) begins as a reader under
// DEFERRED locking and fails its write upgrade with SQLITE_BUSY the moment
// another process has committed — immediately, WITHOUT honouring
// busy_timeout, because a stale read snapshot cannot be resolved by waiting.
//
// The symptom was `awis submit` erroring with "database is locked" whenever
// a few submissions ran concurrently.
func TestBuildDSNCarriesImmediateLocking(t *testing.T) {
	got := buildDSN("/tmp/x/runtime.db")
	for _, want := range []string{"_txlock=immediate", "busy_timeout"} {
		if !contains(got, want) {
			t.Errorf("buildDSN(...) = %q, missing %q", got, want)
		}
	}
}

// TestBuildDSNPreservesAnExistingQuery: a caller-supplied DSN keeps its own
// options, appended with & rather than a second ?.
func TestBuildDSNPreservesAnExistingQuery(t *testing.T) {
	got := buildDSN("file:x.db?mode=ro")
	if !contains(got, "mode=ro") {
		t.Errorf("buildDSN dropped the caller's options: %q", got)
	}
	if countRune(got, '?') != 1 {
		t.Errorf("buildDSN produced %d '?' separators in %q, want exactly 1", countRune(got, '?'), got)
	}
	if !contains(got, "&_txlock=immediate") {
		t.Errorf("buildDSN did not append with '&': %q", got)
	}
}

// TestBuildDSNPlainPathGetsOneSeparator covers the ordinary case.
func TestBuildDSNPlainPathGetsOneSeparator(t *testing.T) {
	got := buildDSN("runtime.db")
	if countRune(got, '?') != 1 {
		t.Errorf("buildDSN(%q) = %q, want exactly one '?'", "runtime.db", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func countRune(s string, r rune) int {
	n := 0
	for _, c := range s {
		if c == r {
			n++
		}
	}
	return n
}
