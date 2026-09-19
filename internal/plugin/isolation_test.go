package plugin

// isolation_test.go — ResolvePluginUser tests (D-11).

import (
	"os/user"
	"strconv"
	"testing"
)

// TestResolvePluginUser_ByUsername verifies that a bare username resolves to
// the uid/gid the OS itself reports for it (os/user.Lookup), using the
// current test-running user so the test needs no fixture account.
func TestResolvePluginUser_ByUsername(t *testing.T) {
	cur, err := user.Current()
	if err != nil {
		t.Skipf("user.Current unavailable in this environment: %v", err)
	}
	wantUID, err := strconv.ParseUint(cur.Uid, 10, 32)
	if err != nil {
		t.Fatalf("parse current uid %q: %v", cur.Uid, err)
	}
	wantGID, err := strconv.ParseUint(cur.Gid, 10, 32)
	if err != nil {
		t.Fatalf("parse current gid %q: %v", cur.Gid, err)
	}

	gotUID, gotGID, err := ResolvePluginUser(cur.Username)
	if err != nil {
		t.Fatalf("ResolvePluginUser(%q): %v", cur.Username, err)
	}
	if gotUID != uint32(wantUID) || gotGID != uint32(wantGID) {
		t.Fatalf("ResolvePluginUser(%q) = (%d, %d), want (%d, %d)", cur.Username, gotUID, gotGID, wantUID, wantGID)
	}
}

// TestResolvePluginUser_ByNumericUID verifies that a bare numeric uid
// resolves its primary gid via os/user.LookupId.
func TestResolvePluginUser_ByNumericUID(t *testing.T) {
	cur, err := user.Current()
	if err != nil {
		t.Skipf("user.Current unavailable in this environment: %v", err)
	}
	wantGID, err := strconv.ParseUint(cur.Gid, 10, 32)
	if err != nil {
		t.Fatalf("parse current gid %q: %v", cur.Gid, err)
	}

	gotUID, gotGID, err := ResolvePluginUser(cur.Uid)
	if err != nil {
		t.Fatalf("ResolvePluginUser(%q): %v", cur.Uid, err)
	}
	wantUID, _ := strconv.ParseUint(cur.Uid, 10, 32)
	if gotUID != uint32(wantUID) || gotGID != uint32(wantGID) {
		t.Fatalf("ResolvePluginUser(%q) = (%d, %d), want (%d, %d)", cur.Uid, gotUID, gotGID, wantUID, wantGID)
	}
}

// TestResolvePluginUser_UidGidForm verifies the literal "uid:gid" form is
// taken as-is, with no OS lookup on either side.
func TestResolvePluginUser_UidGidForm(t *testing.T) {
	gotUID, gotGID, err := ResolvePluginUser("1234:5678")
	if err != nil {
		t.Fatalf("ResolvePluginUser(\"1234:5678\"): %v", err)
	}
	if gotUID != 1234 || gotGID != 5678 {
		t.Fatalf("ResolvePluginUser(\"1234:5678\") = (%d, %d), want (1234, 5678)", gotUID, gotGID)
	}
}

// TestResolvePluginUser_UnknownUser verifies the error path for a username
// that does not resolve — callers (sdk.NewRuntime) must fail outright on
// this, never fall back to an unisolated spawn.
func TestResolvePluginUser_UnknownUser(t *testing.T) {
	_, _, err := ResolvePluginUser("this-user-definitely-does-not-exist-d11")
	if err == nil {
		t.Fatal("ResolvePluginUser(unknown user): want error, got nil")
	}
}

// TestResolvePluginUser_EmptySpec verifies the empty-spec error path.
func TestResolvePluginUser_EmptySpec(t *testing.T) {
	_, _, err := ResolvePluginUser("")
	if err == nil {
		t.Fatal("ResolvePluginUser(\"\"): want error, got nil")
	}
	_, _, err = ResolvePluginUser("   ")
	if err == nil {
		t.Fatal("ResolvePluginUser(\"   \"): want error, got nil")
	}
}

// TestResolvePluginUser_MalformedUidGid verifies non-numeric uid/gid halves
// of the "uid:gid" form are rejected rather than silently misparsed.
func TestResolvePluginUser_MalformedUidGid(t *testing.T) {
	cases := []string{"abc:123", "123:abc", "123:", ":123"}
	for _, spec := range cases {
		if _, _, err := ResolvePluginUser(spec); err == nil {
			t.Errorf("ResolvePluginUser(%q): want error, got nil", spec)
		}
	}
}

// TestResolvePluginUser_UnknownNumericUID verifies a numeric uid with no
// passwd entry is an error (its gid cannot be inferred).
func TestResolvePluginUser_UnknownNumericUID(t *testing.T) {
	// uid 4294967294 is astronomically unlikely to have a passwd entry.
	if _, _, err := ResolvePluginUser("4294967294"); err == nil {
		t.Fatal("ResolvePluginUser(unassigned uid): want error, got nil")
	}
}
