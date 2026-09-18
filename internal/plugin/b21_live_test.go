package plugin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestB21_ShippedManifestRunsFromAnyWorkingDirectory is the B-21 acceptance
// test, and it is deliberately harsh about its own setup.
//
// The pre-existing e2e coverage for the reference plugin passed only because
// it WROTE ITS OWN manifest with an absolute PYTHONPATH injected. That proved
// the plugin wire protocol worked and proved nothing about the plugin that
// actually ships. This test registers plugins/git-context-plugin's real
// awis-plugin.yaml, unmodified, from a temp working directory that is not the
// plugin's own, with no injected environment.
func TestB21_ShippedManifestRunsFromAnyWorkingDirectory(t *testing.T) {
	if _, err := os.Stat("/usr/bin/python3"); err != nil {
		if _, err := exeInPath("python3"); err != nil {
			t.Skip("python3 not available")
		}
	}

	root := repoRootForTest(t)
	manifestPath := filepath.Join(root, "plugins", "git-context-plugin", "awis-plugin.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("shipped manifest missing at %s: %v", manifestPath, err)
	}

	// Run from a directory that is NOT the plugin's, NOT the repo root, and
	// NOT the package directory — the condition B-21 was invisible under.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	s := openTestStorage(t)
	mgr := NewManager(s, ManagerConfig{
		HandshakeTimeout: 15 * time.Second,
		ShutdownGrace:    2 * time.Second,
	})
	ctx := context.Background()
	t.Cleanup(func() { mgr.Shutdown(context.Background()) })

	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register shipped manifest: %v", err)
	}

	// git.context.assemble against the AWIS repo itself. Reaching a real
	// result means spawn, cmd.Dir resolution, AWIS_PLUGIN_LIBPATH resolution,
	// handshake and dispatch all worked with the manifest exactly as shipped.
	outputs, stepErr := mgr.Call(ctx, "git.context.assemble", "step-1",
		map[string]any{"repo_path": root, "ref": "HEAD"}, "")
	if stepErr != nil {
		t.Fatalf("Call git.context.assemble with the SHIPPED manifest from %s: %+v", tmp, stepErr)
	}
	if outputs["context"] == nil {
		t.Errorf("outputs[context] is nil; got %v", outputs)
	}
}

// repoRootForTest walks up from this source file to the module root.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func exeInPath(name string) (string, error) {
	return exec.LookPath(name)
}
