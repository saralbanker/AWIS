package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// update is set via 'go test -run TestVersion -update' to regenerate golden files.
var update = flag.Bool("update", false, "update golden files")

// captureOutput runs fn with os.Stdout redirected to a buffer and returns the
// captured bytes. It restores os.Stdout after fn returns.
func captureOutput(fn func()) []byte {
	// Redirect stdout via a pipe.
	r, w, err := os.Pipe()
	if err != nil {
		panic(fmt.Sprintf("captureOutput: Pipe: %v", err))
	}
	orig := os.Stdout
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		panic(fmt.Sprintf("captureOutput: Close: %v", err))
	}
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.Bytes()
}

// goldenPath returns the absolute path to a golden file in testdata/golden/.
func goldenPath(name string) string {
	return filepath.Join("testdata", "golden", name)
}

// checkGolden compares got against the contents of the golden file at name.
// If -update is set, it writes got to the file instead.
func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := goldenPath(name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("checkGolden: mkdir: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("checkGolden: write %s: %v", path, err)
		}
		t.Logf("golden updated: %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("checkGolden: read %s: %v (run with -update to create)", path, err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("golden mismatch for %s:\nwant:\n%s\ngot:\n%s", name, want, got)
	}
}

// stableGoVersion replaces the compiling toolchain's version string with a
// fixed placeholder, so a golden asserts the output SHAPE rather than the Go
// build that happened to produce it.
//
// runVersion reports runtime.Version(), which changes with every Go patch
// release and therefore differs between machines. Without this, version.txt
// and version.json only pass on whichever toolchain last regenerated them —
// they broke on the first CI run (local go1.26.5 vs runner go1.26.6) and would
// equally break on the next local `go` upgrade.
//
// Same normalisation idiom as plugin_test.go's <plugin-path>.
func stableGoVersion(got []byte) []byte {
	return []byte(strings.ReplaceAll(string(got), runtime.Version(), "<go-version>"))
}

// TestVersionHuman verifies that 'awis version' (human mode) matches the golden.
func TestVersionHuman(t *testing.T) {
	// Reset global flags to defaults before each test.
	globalJSON = false
	globalDataDir = "./.awis/"

	got := captureOutput(func() {
		runVersion(nil)
	})
	checkGolden(t, "version.txt", stableGoVersion(got))
}

// TestVersionJSON verifies that 'awis --json version' matches the golden.
func TestVersionJSON(t *testing.T) {
	globalJSON = true
	globalDataDir = "./.awis/"

	got := captureOutput(func() {
		runVersion(nil)
	})
	globalJSON = false // reset

	checkGolden(t, "version.json", stableGoVersion(got))
}

// TestVersionJSONShape verifies the JSON output is valid and has the required fields.
func TestVersionJSONShape(t *testing.T) {
	globalJSON = true
	globalDataDir = "./.awis/"
	got := captureOutput(func() {
		runVersion(nil)
	})
	globalJSON = false

	var out versionOutput
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("version --json: invalid JSON: %v\noutput: %s", err, got)
	}
	if out.Version == "" {
		t.Error("version --json: version field is empty")
	}
	if out.GoVersion == "" {
		t.Error("version --json: go_version field is empty")
	}
	if out.Version != version {
		t.Errorf("version --json: version=%q, want %q", out.Version, version)
	}
	if out.GoVersion != runtime.Version() {
		t.Errorf("version --json: go_version=%q, want %q", out.GoVersion, runtime.Version())
	}
}

// TestUnknownCommandUsage verifies that an unknown command exits 2 (usage error).
// We can't call main() directly without forking, so we test the dispatch logic.
func TestUnknownCommandUsage(t *testing.T) {
	_, ok := commands["nonexistent-command-xyz"]
	if ok {
		t.Fatal("commands map should not contain 'nonexistent-command-xyz'")
	}
}

// TestVersionHumanFormat verifies the human format matches the expected shape
// without comparing the exact go version (avoids coupling to runtime).
func TestVersionHumanFormat(t *testing.T) {
	globalJSON = false
	got := captureOutput(func() {
		runVersion(nil)
	})
	s := strings.TrimRight(string(got), "\n")
	if !strings.HasPrefix(s, "awis version ") {
		t.Errorf("human output prefix: got %q", s)
	}
	if !strings.Contains(s, "  go ") {
		t.Errorf("human output missing go version separator: got %q", s)
	}
}
