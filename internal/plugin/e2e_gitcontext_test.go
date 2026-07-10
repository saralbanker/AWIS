package plugin

// e2e_gitcontext_test.go — Go e2e tests: real git-context-plugin via Manager + harness
// workflow, running against THIS repository's real git history.
//
// TestE2EGitContextAssemble_PluginNameHandler:
//   Registers the git-context-plugin using a temp manifest (absolute PYTHONPATH),
//   runs a harness workflow with handler: git-context-plugin (plugin-name form),
//   exercises the input-key-set resolution rule (two capabilities registered),
//   asserts resolution chose git.context.assemble (context.sha non-empty,
//   context.message non-empty, stats present).
//
// TestE2EGitDiffFetch_CapabilityIDHandler:
//   Uses capability-id handler git.diff.fetch with refs HEAD~1..HEAD from this
//   repo; skips if history is too shallow; asserts non-empty diff string.
//
// Both tests skip when python3 or git is absent from PATH.
//
// TRACEABILITY: T5 (Go e2e: plugin-name handler + capability-id handler vs this
// repo's real history; frozen §7 fixture form; IMP §27.M13 Val).

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/engine"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/storage"
)


// e2eRepoRoot returns the absolute path to the repository root (two levels above
// internal/plugin/).
func e2eRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	abs, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

// e2eGitContextManifest writes a temp manifest for git-context-plugin with
// absolute PYTHONPATH covering python/awis-plugin and plugins/git-context-plugin.
func e2eGitContextManifest(t *testing.T, python3Path, repoRoot string) string {
	t.Helper()

	// PYTHONPATH: both library roots so `python3 -m git_context_plugin` resolves
	// git_context_plugin and awis_plugin from the working tree.
	awisPluginLib := filepath.Join(repoRoot, "python", "awis-plugin")
	gitContextPluginDir := filepath.Join(repoRoot, "plugins", "git-context-plugin")
	pythonPath := awisPluginLib + string(os.PathListSeparator) + gitContextPluginDir

	content := fmt.Sprintf(`name: git-context-plugin
version: 1.0.0
description: Assembles git context for capture workflows
author: awis

capabilities:
  - id: git.context.assemble
    inputs:
      repo_path: string
      ref: string
    outputs:
      context: object
    timeout_ms: 30000

  - id: git.diff.fetch
    inputs:
      repo_path: string
      from_ref: string
      to_ref: string
    outputs:
      diff: string
    timeout_ms: 10000

runtime:
  command: %s
  args: ["-m", "git_context_plugin"]
  env:
    GIT_TERMINAL_PROMPT: "0"
    PYTHONPATH: %s
    AWIS_PLUGIN_LIBPATH: %s
  idle_timeout_s: 300
`, python3Path, pythonPath, awisPluginLib)

	path := filepath.Join(t.TempDir(), "git-context-plugin.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp manifest: %v", err)
	}
	return path
}

// e2eGitContextSetup creates storage + Manager + engine wired with a PluginRunner
// and a native runner, registers the git-context-plugin, and returns them.
func e2eGitContextSetup(t *testing.T, manifestPath string) (
	*Manager, *engine.Engine, *storage.SQLiteStorage,
) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "e2e-gitcontext.db")
	db, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := storage.NewSQLiteStorage(db, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mgr := NewManager(store, ManagerConfig{
		HandshakeTimeout: 15 * time.Second,
		ShutdownGrace:    500 * time.Millisecond,
	})
	t.Cleanup(func() {
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		mgr.Shutdown(shutCtx)
	})

	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Manager.Register: %v", err)
	}

	pluginRunner := NewPluginRunner(mgr)
	nr := native.New()

	runners := map[core.StepType]engine.Runner{
		core.StepTypeNative: nr,
		core.StepTypePlugin: pluginRunner,
	}

	eng := engine.New(store, runners, engine.Config{
		TickInterval: 50 * time.Millisecond,
	}, nil)

	return mgr, eng, store
}

// runWorkflowToCompletion submits wf and ticks the engine until the instance
// completes, then returns the instance.
func runWorkflowToCompletion(
	t *testing.T,
	ctx context.Context,
	eng *engine.Engine,
	store *storage.SQLiteStorage,
	wf core.WorkflowDefinition,
	initialData map[string]any,
	timeout time.Duration,
) core.WorkflowInstance {
	t.Helper()

	regCtx, regCancel := context.WithTimeout(ctx, 5*time.Second)
	defer regCancel()
	if err := store.RegisterWorkflow(regCtx, wf); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	submitCtx, submitCancel := context.WithTimeout(ctx, 5*time.Second)
	defer submitCancel()
	instanceID, err := eng.Submit(submitCtx, wf.ID, wf.Version, initialData)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := eng.Tick(ctx); err != nil {
			t.Logf("Tick (non-fatal): %v", err)
		}
		inst, err := store.GetInstance(ctx, instanceID)
		if err != nil {
			t.Fatalf("GetInstance: %v", err)
		}
		switch inst.Status {
		case "completed":
			return inst
		case "failed":
			t.Fatalf("workflow failed: %+v", inst)
		}
		time.Sleep(60 * time.Millisecond)
	}

	inst, _ := store.GetInstance(ctx, instanceID)
	t.Fatalf("workflow did not complete within %s; status=%q", timeout, inst.Status)
	return core.WorkflowInstance{}
}

// TestE2EGitContextAssemble_PluginNameHandler registers the git-context-plugin and
// runs a harness workflow using handler: git-context-plugin (the frozen §7 plugin-name
// form).  Because the plugin has TWO capabilities, the input-key-set resolution rule
// must select git.context.assemble (inputs: repo_path + ref).
//
// Assertions: context.sha non-empty, context.message non-empty, stats present.
func TestE2EGitContextAssemble_PluginNameHandler(t *testing.T) {
	python3, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not found on PATH; skipping e2e git-context-plugin test")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH; skipping e2e git-context-plugin test")
	}

	repoRoot := e2eRepoRoot(t)
	manifestPath := e2eGitContextManifest(t, python3, repoRoot)

	_, eng, store := e2eGitContextSetup(t, manifestPath)

	// Workflow: plugin step (handler = plugin name form) → native consume step.
	wf := core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "e2e-gitcontext-assemble-wf",
		Version:       "1.0.0",
		Namespace:     "test",
		Name:          "e2e-gitcontext-assemble-wf",
		InitialStep:   "git-step",
		FinalSteps:    []string{"git-step"},
		Steps: []core.Step{
			{
				ID:      "git-step",
				Type:    core.StepTypePlugin,
				Handler: "git-context-plugin", // plugin-name form (frozen §7 fixture)
				Inputs: core.InputSchema{
					"repo_path": repoRoot,
					"ref":       "HEAD",
				},
			},
		},
	}

	ctx := context.Background()
	inst := runWorkflowToCompletion(t, ctx, eng, store, wf, map[string]any{}, 45*time.Second)

	// Verify outputs stored in instance variables.
	stepVars, ok := inst.Variables["git-step"]
	if !ok {
		t.Fatalf("instance variables missing 'git-step'; variables=%v", inst.Variables)
	}
	stepMap, ok := stepVars.(map[string]any)
	if !ok {
		t.Fatalf("instance variables['git-step'] is not a map; got %T", stepVars)
	}

	// The capability returns {context: { ... }}
	contextRaw, ok := stepMap["context"]
	if !ok {
		t.Fatalf("git-step outputs missing 'context'; stepMap=%v", stepMap)
	}
	contextMap, ok := contextRaw.(map[string]any)
	if !ok {
		t.Fatalf("git-step outputs['context'] is not a map; got %T", contextRaw)
	}

	// Assert resolution chose git.context.assemble: sha non-empty.
	sha, _ := contextMap["sha"].(string)
	if sha == "" {
		t.Errorf("context.sha is empty; contextMap=%v", contextMap)
	}

	// Assert message non-empty.
	message, _ := contextMap["message"].(string)
	if message == "" {
		t.Errorf("context.message is empty; contextMap=%v", contextMap)
	}

	// Assert stats present.
	statsRaw, ok := contextMap["stats"]
	if !ok {
		t.Errorf("context.stats missing; contextMap=%v", contextMap)
	}
	if _, ok := statsRaw.(map[string]any); !ok {
		t.Errorf("context.stats is not a map; got %T", statsRaw)
	}

	t.Logf("PASS: resolution chose git.context.assemble; sha=%s message=%q",
		sha, truncate(message, 60))
}

// TestE2EGitDiffFetch_CapabilityIDHandler calls the git.diff.fetch capability
// directly by capability-id handler using HEAD~1..HEAD from this repository.
// Skips if the repo history is too shallow (HEAD~1 not resolvable).
//
// Assertions: diff output is non-empty.
func TestE2EGitDiffFetch_CapabilityIDHandler(t *testing.T) {
	python3, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not found on PATH; skipping e2e git.diff.fetch test")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH; skipping e2e git.diff.fetch test")
	}

	repoRoot := e2eRepoRoot(t)

	// Pre-check: resolve HEAD~1 — skip if shallow.
	headMinus1Out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD~1").Output()
	if err != nil {
		t.Skip("repo history too shallow to resolve HEAD~1; skipping git.diff.fetch e2e test")
	}
	fromRef := strings.TrimSpace(string(headMinus1Out))
	if fromRef == "" {
		t.Skip("git rev-parse HEAD~1 returned empty; skipping git.diff.fetch e2e test")
	}

	manifestPath := e2eGitContextManifest(t, python3, repoRoot)
	_, eng, store := e2eGitContextSetup(t, manifestPath)

	// Workflow: plugin step using capability-id handler directly.
	wf := core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "e2e-gitdiff-fetch-wf",
		Version:       "1.0.0",
		Namespace:     "test",
		Name:          "e2e-gitdiff-fetch-wf",
		InitialStep:   "diff-step",
		FinalSteps:    []string{"diff-step"},
		Steps: []core.Step{
			{
				ID:      "diff-step",
				Type:    core.StepTypePlugin,
				Handler: "git.diff.fetch", // capability-id form
				Inputs: core.InputSchema{
					"repo_path": repoRoot,
					"from_ref":  fromRef,
					"to_ref":    "HEAD",
				},
			},
		},
	}

	ctx := context.Background()
	inst := runWorkflowToCompletion(t, ctx, eng, store, wf, map[string]any{}, 45*time.Second)

	// Verify diff output.
	stepVars, ok := inst.Variables["diff-step"]
	if !ok {
		t.Fatalf("instance variables missing 'diff-step'; variables=%v", inst.Variables)
	}
	stepMap, ok := stepVars.(map[string]any)
	if !ok {
		t.Fatalf("instance variables['diff-step'] is not a map; got %T", stepVars)
	}

	diff, _ := stepMap["diff"].(string)
	if diff == "" {
		t.Errorf("git.diff.fetch returned empty diff; stepMap=%v", stepMap)
	}

	t.Logf("PASS: git.diff.fetch; diff length=%d bytes", len(diff))
}

// truncate returns s truncated to n runes, with "…" appended if truncated.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
