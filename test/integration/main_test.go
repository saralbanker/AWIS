//go:build integration

package integration

// main_test.go — the package's TestMain.
//
// TestMain MUST live in a _test.go file: Go only recognises it as the test
// entry point there. It previously sat in harness.go (a non-test file), where
// it compiled fine but was never invoked, so awisBinPath stayed empty and every
// test failed with "exec: no command".

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain builds ./cmd/awis exactly once (go build -o <tmp>/awis) and fails
// fast if the build does not succeed, so every test below exercises the same
// binary a real user would run — never the internal Go APIs directly.
func TestMain(m *testing.M) {
	root, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration: locate repo root:", err)
		os.Exit(1)
	}
	binDir, err := os.MkdirTemp("", "awis-integration-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration: mkdir temp bin dir:", err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(binDir) }()

	awisBinPath = filepath.Join(binDir, "awis")
	build := exec.Command("go", "build", "-o", awisBinPath, "./cmd/awis")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "integration: go build ./cmd/awis failed: %v\n%s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// repoRoot walks up from this source file until it finds go.mod.
func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", filepath.Dir(file))
		}
		dir = parent
	}
}
