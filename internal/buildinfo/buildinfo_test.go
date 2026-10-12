package buildinfo

import "testing"

// TestVersionDefaultIsPackageVar pins the unlinked development default and the
// shape -X ldflags need. Release binaries override Version at link time
// (.goreleaser.yaml); a plain `go test` never does. Taking the address compiles
// only for a variable, so this test fails to build if Version is ever turned back
// into a constant.
func TestVersionDefaultIsPackageVar(t *testing.T) {
	const want = "0.1.0-dev"
	p := &Version
	if *p != want {
		t.Fatalf("Version = %q, want development default %q", *p, want)
	}
}
