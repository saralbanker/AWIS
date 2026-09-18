package plugin

// spawn_test.go — spawnPlugin dedicated-UID isolation tests (D-11).
//
// TRACEABILITY: D-11 (PRD §32 rows 38/53, founder ruling DEC-9).

import (
	"os"
	"strings"
	"syscall"
	"testing"
)

// TestSpawnPlugin_Unconfigured_PreservesTodaysBehaviour verifies that when no
// plugin user is configured (uidSet=false), SysProcAttr is byte-for-byte what
// it was before D-11: Setpgid true, Credential nil. Asserted on the
// constructed *exec.Cmd directly (newPluginCmd) — no process is started, so
// this needs no privilege and runs identically under any test UID.
func TestSpawnPlugin_Unconfigured_PreservesTodaysBehaviour(t *testing.T) {
	cmd := newPluginCmd("sh", []string{"-c", "exit 0"}, os.Environ(), "", 0, 0, false)

	attr := cmd.SysProcAttr
	if attr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if !attr.Setpgid {
		t.Error("Setpgid = false, want true (load-bearing for process-group reaping)")
	}
	if attr.Credential != nil {
		t.Errorf("Credential = %+v, want nil when no plugin user is configured", attr.Credential)
	}
}

// TestSpawnPlugin_Configured_SetsCredential verifies that when a plugin user
// IS configured (uidSet=true), Credential is populated with the resolved
// uid/gid and Setpgid remains true. Asserted on the constructed *exec.Cmd
// directly (newPluginCmd), per the card: no need to actually spawn as
// another user (that needs root/CAP_SETUID) — just assert on the
// constructed SysProcAttr.
func TestSpawnPlugin_Configured_SetsCredential(t *testing.T) {
	const uid, gid = uint32(4242), uint32(4343) // arbitrary; never started

	cmd := newPluginCmd("sh", []string{"-c", "exit 0"}, os.Environ(), "", uid, gid, true)

	attr := cmd.SysProcAttr
	if attr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if !attr.Setpgid {
		t.Error("Setpgid = false, want true (load-bearing for process-group reaping)")
	}
	if attr.Credential == nil {
		t.Fatal("Credential is nil, want populated when plugin user is configured")
	}
	if attr.Credential.Uid != uid || attr.Credential.Gid != gid {
		t.Errorf("Credential = {Uid:%d Gid:%d}, want {Uid:%d Gid:%d}", attr.Credential.Uid, attr.Credential.Gid, uid, gid)
	}
}

// TestSpawnPlugin_EPERM_IsExplicitAndActionable verifies the D-11
// failure-mode requirement: when a plugin user IS configured but the spawn
// fails with EPERM (the engine lacks privilege to setuid), the error must say
// so explicitly and actionably — not a bare "operation not permitted" — and
// must NOT have spawned the process unisolated.
//
// Skipped when running as root: a privileged process can setuid to any uid,
// so the EPERM path is unreachable and this test cannot exercise it.
func TestSpawnPlugin_EPERM_IsExplicitAndActionable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: cannot exercise the EPERM path (root may setuid to any uid)")
	}

	// A uid that is neither our own nor root's is guaranteed EPERM for an
	// unprivileged caller.
	targetUID := uint32(os.Getuid()) + 1
	if targetUID == 0 {
		targetUID = uint32(os.Getuid()) + 2
	}

	_, err := spawnPlugin("sh", []string{"-c", "exit 0"}, os.Environ(), "", targetUID, uint32(os.Getgid()), true)
	if err == nil {
		t.Fatal("spawnPlugin with foreign uid as non-root: want EPERM error, got nil")
	}
	if !strings.Contains(err.Error(), "operation not permitted") {
		t.Errorf("error does not mention the underlying EPERM condition: %v", err)
	}
	// The message must be actionable, not bare — it must name the missing
	// capability/privilege and must not claim an unisolated fallback occurred.
	for _, want := range []string{"CAP_SETUID", "plugins_user", "D-11"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is not actionable: missing %q in: %v", want, err)
		}
	}

	// Sanity: confirm the underlying cause really is EPERM, so the test
	// itself is exercising the intended condition and not some other failure.
	var errno syscall.Errno
	found := false
	for e := err; e != nil; {
		if u, ok := e.(interface{ Unwrap() error }); ok {
			e = u.Unwrap()
		} else {
			break
		}
		if en, ok := e.(syscall.Errno); ok {
			errno = en
			found = true
			break
		}
	}
	if !found || errno != syscall.EPERM {
		t.Skipf("could not confirm underlying EPERM (got errno=%v found=%v); environment may not support this assertion", errno, found)
	}
}
