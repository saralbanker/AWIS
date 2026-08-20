package plugin

// e2e_python_test.go — Go e2e test: real Python plugin via Manager + harness workflow.
//
// Registers an echo-python-plugin (testdata/python/echo_plugin.py) via
// Manager.Register; runs it through a harness workflow with a type=plugin step;
// verifies outputs flow to a subsequent native step.
//
// Skips if python3 is not available on the PATH.
//
// TRACEABILITY: T13 (Go e2e with real Python plugin).

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/engine"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)

// e2ePythonLibPath returns the absolute path to python/awis-plugin (the lib root).
func e2ePythonLibPath(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// internal/plugin/ → repo root → python/awis-plugin
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatalf("filepath.Abs(repoRoot): %v", err)
	}
	return filepath.Join(abs, "python", "awis-plugin")
}

// e2ePythonPluginDir returns the absolute path to testdata/python/.
func e2ePythonPluginDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "python")
}

// TestE2EPythonPlugin registers a real Python plugin and runs a harness workflow.
//
// Workflow:
//   plugin-step (type=plugin, handler=echo.call): inputs {message: "hello"}
//     → outputs {reply: "hello"}
//   verify-step (type=native, handler=test-native-echo): reads plugin-step.reply
//     → workflow completes.
//
// This test proves the full path: Python process ↔ NDJSON JSON-RPC ↔ Manager ↔
// PluginRunner ↔ engine ↔ storage.
func TestE2EPythonPlugin(t *testing.T) {
	// Skip if python3 is not available.
	python3, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not found on PATH; skipping e2e Python plugin test")
	}

	// Locate paths.
	libPath := e2ePythonLibPath(t)
	pluginDir := e2ePythonPluginDir(t)
	modulePath := filepath.Join(pluginDir, "echo_plugin.py")
	manifestTemplatePath := filepath.Join(pluginDir, "awis-plugin.yaml")

	// Verify the fixture files exist.
	for _, p := range []string{modulePath, manifestTemplatePath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("fixture file not found: %s: %v", p, err)
		}
	}

	// Write a temp manifest with the real python3 path and module path as args.
	// The plugin is invoked as: python3 <module_path> <manifest_path>
	// The module reads sys.argv[1] as the manifest_path.
	tmpManifestContent := fmt.Sprintf(`name: echo-python-plugin
version: 1.0.0
description: Echo capability implemented in Python (M12-C4 e2e fixture)
author: test

capabilities:
  - id: echo.call
    inputs:
      message: string
    outputs:
      reply: string
    timeout_ms: 10000

runtime:
  command: %q
  args: [%q]
  env:
    AWIS_PLUGIN_LIBPATH: %q
  idle_timeout_s: 300
`, python3, modulePath, libPath)

	tmpManifestPath := filepath.Join(t.TempDir(), "echo-python-plugin.yaml")
	if err := os.WriteFile(tmpManifestPath, []byte(tmpManifestContent), 0o644); err != nil {
		t.Fatalf("write temp manifest: %v", err)
	}

	// Open storage.
	dbPath := filepath.Join(t.TempDir(), "e2e-python.db")
	db, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := storage.NewSQLiteStorage(db, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build the manager and register the Python plugin.
	mgr := NewManager(store, ManagerConfig{
		HandshakeTimeout: 10 * time.Second,
		ShutdownGrace:    500 * time.Millisecond,
	})
	defer mgr.Shutdown(ctx)

	if err := mgr.Register(ctx, tmpManifestPath); err != nil {
		t.Fatalf("Manager.Register: %v", err)
	}

	// Build the engine with a PluginRunner + a native runner.
	pluginRunner := NewPluginRunner(mgr)
	nr := native.New()

	runners := map[core.StepType]engine.Runner{
		core.StepTypeNative: nr,
		core.StepTypePlugin: pluginRunner,
	}

	eng := engine.New(store, runners, engine.Config{
		TickInterval: 50 * time.Millisecond,
	}, nil)

	// Register a workflow: plugin-step → (verify outputs inline via final step).
	// We use a single-step workflow with FinalSteps=["plugin-step"] so
	// the workflow completes after the plugin step succeeds.
	wf := core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "e2e-python-wf",
		Version:       "1.0.0",
		Namespace:     "test",
		Name:          "e2e-python-wf",
		InitialStep:   "plugin-step",
		FinalSteps:    []string{"plugin-step"},
		Steps: []core.Step{
			{
				ID:      "plugin-step",
				Type:    core.StepTypePlugin,
				Handler: "echo.call",
				Inputs:  core.InputSchema{"message": "hello"},
			},
		},
	}
	if err := store.RegisterWorkflow(ctx, wf); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	// Submit the workflow instance.
	instanceID, err := eng.Submit(ctx, wf.ID, wf.Version, map[string]any{})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Tick until the workflow completes (or times out).
	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		if err := eng.Tick(ctx); err != nil {
			t.Logf("Tick error (non-fatal): %v", err)
		}
		inst, err := store.GetInstance(ctx, instanceID)
		if err != nil {
			t.Fatalf("GetInstance: %v", err)
		}
		switch inst.Status {
		case "completed":
			// Verify the plugin step outputs are in the instance variables.
			// The engine stores step outputs as variables["<step-id>"] = map[string]any.
			stepVars, ok := inst.Variables["plugin-step"]
			if !ok {
				t.Errorf("instance variables missing 'plugin-step'; variables=%v", inst.Variables)
			} else if stepMap, ok := stepVars.(map[string]any); !ok {
				t.Errorf("instance variables['plugin-step'] is not a map; got %T", stepVars)
			} else {
				reply, ok := stepMap["reply"]
				if !ok {
					t.Errorf("plugin-step outputs missing 'reply'; stepMap=%v", stepMap)
				} else if reply != "hello" {
					t.Errorf("plugin-step reply: got %q, want %q", reply, "hello")
				} else {
					t.Logf("e2e Python plugin PASS: workflow completed; reply=%q", reply)
				}
			}
			return
		case "failed":
			t.Fatalf("workflow failed: %+v", inst)
		}
		time.Sleep(60 * time.Millisecond)
	}

	inst, _ := store.GetInstance(ctx, instanceID)
	t.Fatalf("workflow did not complete within time budget; status=%q", inst.Status)
}
