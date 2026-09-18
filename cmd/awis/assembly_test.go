package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/dsl"
	"github.com/awis/awis/internal/examples"
	"github.com/awis/awis/internal/validate"
)

// TestUnresolvedHandlers covers the B-9 startup diagnostic: `awis start` must
// tell the operator up front which native handlers its workflows reference but
// this binary cannot dispatch, instead of letting each step fail at run time
// with an unexplained handler_not_found.
func TestUnresolvedHandlers(t *testing.T) {
	nativeStep := func(id, handler string) core.Step {
		return core.Step{ID: id, Type: core.StepTypeNative, Handler: core.HandlerRef(handler)}
	}
	def := func(steps ...core.Step) *core.WorkflowDefinition {
		return &core.WorkflowDefinition{Steps: steps}
	}

	tests := []struct {
		name       string
		defs       []*core.WorkflowDefinition
		registered []string
		want       []string
	}{
		{
			name:       "all handlers registered",
			defs:       []*core.WorkflowDefinition{def(nativeStep("a", "h1"), nativeStep("b", "h2"))},
			registered: []string{"h1", "h2"},
			want:       nil,
		},
		{
			name:       "one missing",
			defs:       []*core.WorkflowDefinition{def(nativeStep("a", "h1"), nativeStep("b", "nope"))},
			registered: []string{"h1"},
			want:       []string{"nope"},
		},
		{
			name: "duplicates collapse and result is sorted",
			defs: []*core.WorkflowDefinition{
				def(nativeStep("a", "zeta"), nativeStep("b", "alpha")),
				def(nativeStep("c", "zeta")),
			},
			registered: nil,
			want:       []string{"alpha", "zeta"},
		},
		{
			name: "non-native steps are ignored",
			defs: []*core.WorkflowDefinition{def(
				core.Step{ID: "s", Type: core.StepTypeSignal},
				core.Step{ID: "p", Type: core.StepTypePlugin, Handler: "some.capability"},
				core.Step{ID: "i", Type: core.StepTypeIntelligence},
			)},
			registered: nil,
			want:       nil,
		},
		{
			name:       "empty handler ref is ignored",
			defs:       []*core.WorkflowDefinition{def(nativeStep("a", ""))},
			registered: nil,
			want:       nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := unresolvedHandlers(tc.defs, tc.registered)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("unresolvedHandlers() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestScaffoldWorkflowsValidateCleanly is the B-31 regression: the scaffold
// shipped `wait_signal.timeout_action: cancel`, an illegal value with no
// validator rule to catch it, so `awis workflow validate` and the runtime
// submit path both accepted it silently and the engine's degrade-on-unknown
// backstop routed every timeout as a failure. Every embedded scaffold
// workflow must produce zero Issues from validate.Validate — the same check
// `awis workflow validate` and engine/submit.go run.
func TestScaffoldWorkflowsValidateCleanly(t *testing.T) {
	for i, def := range parseScaffoldWorkflows(t) {
		if issues := validate.Validate(*def); len(issues) > 0 {
			t.Errorf("scaffold workflow %d (%s) has validation issues: %+v", i, def.ID, issues)
		}
	}
}

// TestScaffoldWorkflowsResolveAgainstBuiltins is the end-to-end form of the
// same guarantee: the workflows `awis init` scaffolds must reference zero
// unresolved handlers, or the documented quickstart fails on its first step.
func TestScaffoldWorkflowsResolveAgainstBuiltins(t *testing.T) {
	defs := parseScaffoldWorkflows(t)
	if missing := unresolvedHandlers(defs, examples.IDs()); len(missing) > 0 {
		t.Errorf("scaffolded workflows reference handlers not built into the binary: %v", missing)
	}
}

// parseScaffoldWorkflows writes the embedded scaffold workflows to a temp dir
// and parses them, so the test exercises the same bytes `awis init` emits.
func parseScaffoldWorkflows(t *testing.T) []*core.WorkflowDefinition {
	t.Helper()
	dir := t.TempDir()
	var defs []*core.WorkflowDefinition
	for _, embPath := range scaffoldFiles {
		if filepath.Ext(embPath) != ".yaml" || filepath.Dir(embPath) != "scaffold/workflows" {
			continue
		}
		data, err := scaffoldFS.ReadFile(embPath)
		if err != nil {
			t.Fatalf("read embedded %q: %v", embPath, err)
		}
		dest := filepath.Join(dir, filepath.Base(embPath))
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			t.Fatalf("write %q: %v", dest, err)
		}
		def, err := dsl.ParseFile(dest)
		if err != nil {
			t.Fatalf("parse scaffolded workflow %q: %v", embPath, err)
		}
		defs = append(defs, def)
	}
	if len(defs) == 0 {
		t.Fatal("no scaffolded workflows parsed — the check is not actually running")
	}
	return defs
}

// TestConfigPathPrefersProjectRoot covers B-10: `awis init` writes config.yaml
// at the project root, but every config sub-command used to look only under
// --data-dir, so `awis config show` right after `awis init` reported
// "Config file not found" while the file sat in the working directory.
func TestConfigPathPrefersProjectRoot(t *testing.T) {
	origDataDir := globalDataDir
	t.Cleanup(func() { globalDataDir = origDataDir })

	run := func(t *testing.T, setup func(dir string), want string) {
		t.Helper()
		dir := t.TempDir()
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd: %v", err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("Chdir: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(wd) })

		globalDataDir = ".awis"
		if err := os.MkdirAll(".awis", 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		setup(dir)
		if got := configPath(); got != want {
			t.Errorf("configPath() = %q, want %q", got, want)
		}
	}

	t.Run("project root wins when present", func(t *testing.T) {
		run(t, func(string) {
			if err := os.WriteFile("config.yaml", []byte("namespace: x\n"), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			if err := os.WriteFile(filepath.Join(".awis", "config.yaml"), []byte("namespace: y\n"), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
		}, "config.yaml")
	})

	t.Run("falls back to data-dir", func(t *testing.T) {
		run(t, func(string) {
			if err := os.WriteFile(filepath.Join(".awis", "config.yaml"), []byte("namespace: y\n"), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
		}, filepath.Join(".awis", "config.yaml"))
	})

	t.Run("defaults to project root when neither exists", func(t *testing.T) {
		run(t, func(string) {}, "config.yaml")
	})
}

// TestLoadConfigKeys checks that config loading is advisory: a missing or
// malformed file must yield an empty map, never block startup.
func TestLoadConfigKeys(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	origDataDir := globalDataDir
	globalDataDir = ".awis"
	t.Cleanup(func() { globalDataDir = origDataDir })

	if got := loadConfigKeys(); len(got) != 0 {
		t.Errorf("loadConfigKeys() with no file = %v, want empty", got)
	}

	if err := os.WriteFile("config.yaml", []byte("# comment\nnamespace: examples\ntick: 50ms\nbroken line\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := loadConfigKeys()
	if got["namespace"] != "examples" {
		t.Errorf("namespace = %q, want %q", got["namespace"], "examples")
	}
	if got["tick"] != "50ms" {
		t.Errorf("tick = %q, want %q", got["tick"], "50ms")
	}
}
