package main

// c1r_test.go — M17-C1r golden tests for remaining M17 commands.
//
// Wires 6 orphaned goldens into real tests and adds replay golden:
//   - replay.json (strategy 1: fixed struct)
//   - config_show.{json,txt} (strategy 2: seeded temp db)
//   - config_set.{json,txt} (strategy 2: seeded temp db)
//   - config_validate.{json,txt} (strategy 2: seeded temp db)
//   - rebuild_state.{json,txt} (strategy 2: seeded temp db)
//   - plugin_remove.{json,txt} (strategy 1: fixed struct)
//   - plugin_status.{json,txt} (strategy 1: fixed struct)

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/sdk"
)

// ── replay.json ───────────────────────────────────────────────────────────────

// TestReplayGoldenJSON verifies the replay --json output golden.
// Uses strategy 1: build a fixed replayOutputJSON struct with deterministic values.
func TestReplayGoldenJSON(t *testing.T) {
	epoch := time.Date(2026, 7, 10, 14, 0, 0, 0, time.UTC)
	out := replayOutputJSON{
		InstanceID: "det-000001",
		WorkflowID: "capture-decision",
		Namespace:  "oip",
		Status:     "completed",
		DryRun:     true,
		Steps: []replayStepJSON{
			{
				Order:      1,
				EventType:  "WorkflowStarted",
				StepID:     "",
				EmittedAt:  epoch.Format(time.RFC3339),
				RelativeMs: 0,
				Action:     "initialize workflow state",
			},
			{
				Order:      2,
				EventType:  "StepStarted",
				StepID:     "draft-entry",
				EmittedAt:  epoch.Add(13 * time.Second).Format(time.RFC3339),
				RelativeMs: 13000,
				Action:     "dispatch step to worker",
			},
			{
				Order:      3,
				EventType:  "WorkflowCompleted",
				StepID:     "",
				EmittedAt:  epoch.Add(89 * time.Second).Format(time.RFC3339),
				RelativeMs: 89000,
				Action:     "mark instance completed",
			},
		},
	}
	got := encodeNoEscape(t, out)
	checkGolden(t, "replay.json", got)
}

// ── config_show.{json,txt} ────────────────────────────────────────────────────

// TestConfigShowGoldenJSON verifies the config show --json output golden.
// Strategy 2: seed empty config, call runConfigShow in JSON mode.
func TestConfigShowGoldenJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := setGlobals(dir, true)
	defer restore()

	got := captureOutput(func() {
		runConfigShow(nil)
	})
	stable := strings.ReplaceAll(string(got), cfgPath, ".awis/config.yaml")
	checkGolden(t, "config_show.json", []byte(stable))
}

// TestConfigShowGoldenText verifies the config show human output golden.
func TestConfigShowGoldenText(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := setGlobals(dir, false)
	defer restore()

	got := captureOutput(func() {
		runConfigShow(nil)
	})
	stable := strings.ReplaceAll(string(got), cfgPath, ".awis/config.yaml")
	checkGolden(t, "config_show.txt", []byte(stable))
}

// ── config_set.{json,txt} ─────────────────────────────────────────────────────

// TestConfigSetGoldenJSON verifies the config set --json output golden.
// Strategy 2: seed config file, call runConfigSet with namespace key.
func TestConfigSetGoldenJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("namespace: default\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := setGlobals(dir, true)
	defer restore()

	got := captureOutput(func() {
		runConfigSet([]string{"namespace", "staging"})
	})
	stable := strings.ReplaceAll(string(got), cfgPath, ".awis/config.yaml")
	checkGolden(t, "config_set.json", []byte(stable))
}

// TestConfigSetGoldenText verifies the config set human output golden.
func TestConfigSetGoldenText(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("namespace: default\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := setGlobals(dir, false)
	defer restore()

	got := captureOutput(func() {
		runConfigSet([]string{"namespace", "staging"})
	})
	stable := strings.ReplaceAll(string(got), cfgPath, ".awis/config.yaml")
	checkGolden(t, "config_set.txt", []byte(stable))
}

// ── config_validate.{json,txt} ────────────────────────────────────────────────

// TestConfigValidateGoldenJSON verifies the config validate --json output golden.
// Strategy 2: seed valid config file, call runConfigValidate.
func TestConfigValidateGoldenJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("namespace: default\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := setGlobals(dir, true)
	defer restore()

	got := captureOutput(func() {
		runConfigValidate(nil)
	})
	stable := strings.ReplaceAll(string(got), cfgPath, ".awis/config.yaml")
	checkGolden(t, "config_validate.json", []byte(stable))
}

// TestConfigValidateGoldenText verifies the config validate human output golden.
func TestConfigValidateGoldenText(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("namespace: default\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := setGlobals(dir, false)
	defer restore()

	got := captureOutput(func() {
		runConfigValidate(nil)
	})
	stable := strings.ReplaceAll(string(got), cfgPath, ".awis/config.yaml")
	checkGolden(t, "config_validate.txt", []byte(stable))
}

// ── rebuild_state.{json,txt} ──────────────────────────────────────────────────

// TestRebuildStateGoldenJSON verifies the rebuild-state --json output golden.
// Strategy 2: seed empty db, call runRebuildState.
func TestRebuildStateGoldenJSON(t *testing.T) {
	dir := t.TempDir()
	_, err := sdk.SQLiteStorage(filepath.Join(dir, "runtime.db"))
	if err != nil {
		t.Fatalf("rebuild_state golden: %v", err)
	}

	restore := setGlobals(dir, true)
	defer restore()

	got := captureOutput(func() {
		runRebuildState([]string{"--all"})
	})
	stable := strings.ReplaceAll(string(got), dir, ".awis")
	checkGolden(t, "rebuild_state.json", []byte(stable))
}

// TestRebuildStateGoldenText verifies the rebuild-state human output golden.
func TestRebuildStateGoldenText(t *testing.T) {
	dir := t.TempDir()
	_, err := sdk.SQLiteStorage(filepath.Join(dir, "runtime.db"))
	if err != nil {
		t.Fatalf("rebuild_state golden: %v", err)
	}

	restore := setGlobals(dir, false)
	defer restore()

	got := captureOutput(func() {
		runRebuildState([]string{"--all"})
	})
	stable := strings.ReplaceAll(string(got), dir, ".awis")
	checkGolden(t, "rebuild_state.txt", []byte(stable))
}

// ── plugin_remove.{json,txt} ──────────────────────────────────────────────────

// TestPluginRemoveGoldenJSON verifies the plugin remove --json output golden.
// Strategy 1: build fixed pluginRemoveOutputJSON struct.
func TestPluginRemoveGoldenJSON(t *testing.T) {
	out := pluginRemoveOutputJSON{
		Name:    "example-plugin",
		Status:  "removed",
		Removed: true,
	}
	got := encodeNoEscape(t, out)
	checkGolden(t, "plugin_remove.json", got)
}

// TestPluginRemoveGoldenText verifies the plugin remove human output golden.
// Strategy 1: fixed output structure.
func TestPluginRemoveGoldenText(t *testing.T) {
	expected := "Plugin removed: example-plugin\n" +
		"  Status set to: removed\n" +
		"  Audit row written: PluginRemoved\n"
	checkGolden(t, "plugin_remove.txt", []byte(expected))
}

// ── plugin_status.{json,txt} ──────────────────────────────────────────────────

// TestPluginStatusGoldenJSON verifies the plugin status --json output golden.
// Strategy 1: build fixed pluginStatusOutputJSON struct.
func TestPluginStatusGoldenJSON(t *testing.T) {
	out := pluginStatusOutputJSON{
		Name:         "example-plugin",
		Version:      "0.1.0",
		Status:       "registered",
		Path:         "/plugins/example-plugin",
		RegisteredAt: "2026-01-01T00:00:00Z",
	}
	got := encodeNoEscape(t, out)
	checkGolden(t, "plugin_status.json", got)
}

// TestPluginStatusGoldenText verifies the plugin status human output golden.
// Strategy 1: fixed output structure.
func TestPluginStatusGoldenText(t *testing.T) {
	expected := "Plugin:   example-plugin\n" +
		"Version:  0.1.0\n" +
		"Status:   registered\n" +
		"Path:     /plugins/example-plugin\n" +
		"Registered: 2026-01-01T00:00:00Z\n"
	checkGolden(t, "plugin_status.txt", []byte(expected))
}
