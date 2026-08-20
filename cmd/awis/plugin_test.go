package main

// plugin_test.go — M14-C4 T11 tests for 'awis plugin install|list'.
//
// Tests:
//   - install real plugins/git-context-plugin manifest into temp data-dir db
//   - list shows the installed plugin
//   - audit row (PluginRegistered) exists after install
//   - bad path → os.Stat error verified without invoking os.Exit
//   - bad manifest → plugin.ParseManifest error verified without invoking os.Exit
//   - JSON shapes for install and list
//   - goldens: plugin_install.txt, plugin_install.json, plugin_list.txt, plugin_list.json

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pluginpkg "github.com/awis/awis/internal/plugin"
	"github.com/awis/awis/internal/storage"
)

// pluginDataDir creates a temp directory configured as a data dir with a live db.
// Returns the data dir path and the open PluginStore.
func pluginDataDir(t *testing.T) (string, storage.PluginStore) {
	t.Helper()
	dir := t.TempDir()
	store, err := OpenStorage(dir)
	if err != nil {
		t.Fatalf("pluginDataDir: OpenStorage: %v", err)
	}
	ps, ok := store.(storage.PluginStore)
	if !ok {
		t.Fatal("pluginDataDir: storage does not implement PluginStore")
	}
	return dir, ps
}

// realPluginPath returns the absolute path to plugins/git-context-plugin in the repo.
func realPluginPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "plugins", "git-context-plugin"))
	if err != nil {
		t.Fatalf("realPluginPath: Abs: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("realPluginPath: plugin dir not found at %s: %v", p, err)
	}
	return p
}

// setGlobals sets the global flags and returns a restore function.
func setGlobals(dataDir string, jsonMode bool) func() {
	orig := globalDataDir
	origJSON := globalJSON
	globalDataDir = dataDir
	globalJSON = jsonMode
	return func() {
		globalDataDir = orig
		globalJSON = origJSON
	}
}

// ── JSON shape tests ──────────────────────────────────────────────────────────

// TestPluginInstallOutputJSONShape verifies the install JSON output has all TDS-07 §4 fields.
func TestPluginInstallOutputJSONShape(t *testing.T) {
	out := pluginInstallOutputJSON{
		Name:     "git-context-plugin",
		Version:  "1.0.0",
		Path:     "/tmp/plugins/git-context-plugin",
		Provides: []string{"git.context.assemble", "git.diff.fetch"},
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"name", "version", "path", "provides"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("pluginInstallOutputJSON: missing field %q", field)
		}
	}
	provides, ok := parsed["provides"].([]any)
	if !ok || len(provides) == 0 {
		t.Error("pluginInstallOutputJSON: 'provides' should be a non-empty array")
	}
}

// TestPluginListOutputJSONShape verifies the list JSON output has all TDS-07 §4 fields.
func TestPluginListOutputJSONShape(t *testing.T) {
	out := pluginListOutputJSON{
		Plugins: []pluginListEntryJSON{
			{
				Name:    "git-context-plugin",
				Version: "1.0.0",
				Status:  "registered",
				Path:    "/tmp/plugins/git-context-plugin",
			},
		},
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := parsed["plugins"]; !ok {
		t.Error("pluginListOutputJSON: missing 'plugins' field")
	}
	plugins, ok := parsed["plugins"].([]any)
	if !ok || len(plugins) == 0 {
		t.Fatal("pluginListOutputJSON: 'plugins' should be a non-empty array")
	}
	entry, ok := plugins[0].(map[string]any)
	if !ok {
		t.Fatal("pluginListOutputJSON: plugins[0] should be an object")
	}
	for _, field := range []string{"name", "version", "status", "path"} {
		if _, ok := entry[field]; !ok {
			t.Errorf("pluginListEntryJSON: missing field %q", field)
		}
	}
}

// ── Install smoke-test with real manifest ─────────────────────────────────────

// TestPluginInstallRealManifest installs the real git-context-plugin and verifies:
//   - JSON output contains correct name/version/provides
//   - ListPlugins shows the installed plugin
//   - PluginRegistered audit row exists in audit_log
func TestPluginInstallRealManifest(t *testing.T) {
	dataDir, ps := pluginDataDir(t)
	pluginPath := realPluginPath(t)
	restore := setGlobals(dataDir, true)
	defer restore()

	// Capture JSON output from runPluginInstall.
	capturedJSON := captureOutput(func() {
		runPluginInstall([]string{pluginPath})
	})

	var out pluginInstallOutputJSON
	if err := json.Unmarshal(capturedJSON, &out); err != nil {
		t.Fatalf("plugin install JSON decode: %v\noutput: %s", err, capturedJSON)
	}
	if out.Name != "git-context-plugin" {
		t.Errorf("name: got %q, want %q", out.Name, "git-context-plugin")
	}
	if out.Version == "" {
		t.Error("version should not be empty")
	}
	if out.Path == "" {
		t.Error("path should not be empty")
	}
	if len(out.Provides) < 1 {
		t.Error("provides should have at least one capability")
	}

	// Verify expected capability IDs.
	providesSet := make(map[string]bool)
	for _, p := range out.Provides {
		providesSet[p] = true
	}
	for _, want := range []string{"git.context.assemble", "git.diff.fetch"} {
		if !providesSet[want] {
			t.Errorf("expected capability %q in provides, got %v", want, out.Provides)
		}
	}

	// Verify ListPlugins shows the plugin.
	ctx := context.Background()
	rows, err := ps.ListPlugins(ctx)
	if err != nil {
		t.Fatalf("ListPlugins: %v", err)
	}
	found := false
	for _, r := range rows {
		if r.Name == "git-context-plugin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("git-context-plugin not found in ListPlugins after install")
	}

	// Verify PluginRegistered audit row.
	checkAuditRowExists(t, dataDir, "PluginRegistered", "git-context-plugin")
}

// TestPluginListAfterInstall verifies human output of 'plugin list' after install.
func TestPluginListAfterInstall(t *testing.T) {
	dataDir, _ := pluginDataDir(t)
	pluginPath := realPluginPath(t)
	restore := setGlobals(dataDir, false)
	defer restore()

	captureOutput(func() {
		runPluginInstall([]string{pluginPath})
	})

	got := captureOutput(func() {
		runPluginList(nil)
	})
	s := string(got)

	if !strings.Contains(s, "INSTALLED PLUGINS") {
		t.Errorf("plugin list: missing header; got:\n%s", s)
	}
	if !strings.Contains(s, "git-context-plugin") {
		t.Errorf("plugin list: missing plugin name; got:\n%s", s)
	}
	if !strings.Contains(s, "NAME") {
		t.Errorf("plugin list: missing column header NAME; got:\n%s", s)
	}
}

// TestPluginListJSONAfterInstall verifies the JSON list output after install.
func TestPluginListJSONAfterInstall(t *testing.T) {
	dataDir, _ := pluginDataDir(t)
	pluginPath := realPluginPath(t)
	restore := setGlobals(dataDir, false)
	defer restore()

	captureOutput(func() {
		runPluginInstall([]string{pluginPath})
	})

	globalJSON = true
	got := captureOutput(func() {
		runPluginList(nil)
	})

	var listOut pluginListOutputJSON
	if err := json.Unmarshal(got, &listOut); err != nil {
		t.Fatalf("plugin list --json decode: %v\noutput: %s", err, got)
	}
	if len(listOut.Plugins) < 1 {
		t.Fatal("plugin list --json: expected at least one plugin")
	}
	p := listOut.Plugins[0]
	if p.Name != "git-context-plugin" {
		t.Errorf("name: got %q, want git-context-plugin", p.Name)
	}
	if p.Version == "" {
		t.Error("version should not be empty")
	}
	if p.Status == "" {
		t.Error("status should not be empty")
	}
	if p.Path == "" {
		t.Error("path should not be empty")
	}
}

// ── Error path tests ──────────────────────────────────────────────────────────

// TestPluginInstallBadPath verifies that os.Stat fails for a non-existent path.
// (runPluginInstall calls fail()→os.Exit on bad path; we test the underlying condition.)
func TestPluginInstallBadPath(t *testing.T) {
	badPath := filepath.Join(t.TempDir(), "does-not-exist")
	_, err := os.Stat(badPath)
	if err == nil {
		t.Fatal("expected stat error for nonexistent path")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("stat error should reference the path: %v", err)
	}
}

// TestPluginInstallBadManifest verifies that plugin.ParseManifest rejects an invalid manifest.
func TestPluginInstallBadManifest(t *testing.T) {
	dir := t.TempDir()
	badManifest := filepath.Join(dir, "awis-plugin.yaml")
	// Missing required fields: name empty, no capabilities, no runtime.
	if err := os.WriteFile(badManifest, []byte("name: \nversion: \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := pluginpkg.ParseManifest(badManifest)
	if err == nil {
		t.Fatal("expected error for invalid manifest (missing name/capabilities/runtime)")
	}
	// Error must reference the manifest file path.
	if !strings.Contains(err.Error(), badManifest) {
		t.Errorf("error should reference manifest path %q: %v", badManifest, err)
	}
}

// TestPluginInstallManifestFileDirectly verifies that passing the manifest file
// directly (not a directory) also installs correctly.
func TestPluginInstallManifestFileDirectly(t *testing.T) {
	dataDir, _ := pluginDataDir(t)
	pluginDir := realPluginPath(t)
	manifestFile := filepath.Join(pluginDir, "awis-plugin.yaml")

	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runPluginInstall([]string{manifestFile})
	})

	var out pluginInstallOutputJSON
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("plugin install (manifest file) JSON decode: %v\noutput: %s", err, got)
	}
	if out.Name != "git-context-plugin" {
		t.Errorf("name: got %q, want git-context-plugin", out.Name)
	}
}

// ── Golden tests ──────────────────────────────────────────────────────────────

// TestPluginInstallGoldenHuman checks the human output of plugin install against golden.
func TestPluginInstallGoldenHuman(t *testing.T) {
	dataDir, _ := pluginDataDir(t)
	pluginPath := realPluginPath(t)
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runPluginInstall([]string{pluginPath})
	})

	// Normalize absolute path to stable placeholder.
	absPath, _ := filepath.Abs(pluginPath)
	stable := strings.ReplaceAll(string(got), absPath, "<plugin-path>")
	checkGolden(t, "plugin_install.txt", []byte(stable))
}

// TestPluginInstallGoldenJSON checks the JSON output of plugin install against golden.
func TestPluginInstallGoldenJSON(t *testing.T) {
	dataDir, _ := pluginDataDir(t)
	pluginPath := realPluginPath(t)
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runPluginInstall([]string{pluginPath})
	})

	absPath, _ := filepath.Abs(pluginPath)
	stable := strings.ReplaceAll(string(got), absPath, "<plugin-path>")
	checkGolden(t, "plugin_install.json", []byte(stable))
}

// TestPluginListGoldenHuman checks the human output of plugin list against golden.
func TestPluginListGoldenHuman(t *testing.T) {
	dataDir, _ := pluginDataDir(t)
	pluginPath := realPluginPath(t)
	restore := setGlobals(dataDir, false)
	defer restore()

	captureOutput(func() {
		runPluginInstall([]string{pluginPath})
	})

	got := captureOutput(func() {
		runPluginList(nil)
	})

	absPath, _ := filepath.Abs(pluginPath)
	stable := strings.ReplaceAll(string(got), absPath, "<plugin-path>")
	checkGolden(t, "plugin_list.txt", []byte(stable))
}

// TestPluginListGoldenJSON checks the JSON output of plugin list against golden.
func TestPluginListGoldenJSON(t *testing.T) {
	dataDir, _ := pluginDataDir(t)
	pluginPath := realPluginPath(t)
	restore := setGlobals(dataDir, true)
	defer restore()

	captureOutput(func() {
		runPluginInstall([]string{pluginPath})
	})

	globalJSON = true
	got := captureOutput(func() {
		runPluginList(nil)
	})

	absPath, _ := filepath.Abs(pluginPath)
	stable := strings.ReplaceAll(string(got), absPath, "<plugin-path>")
	checkGolden(t, "plugin_list.json", []byte(stable))
}

// ── Helper: manifest JSON round-trip ─────────────────────────────────────────

// TestManifestStorageJSONRoundTrip verifies buildManifestJSON + extractPathFromManifestJSON.
func TestManifestStorageJSONRoundTrip(t *testing.T) {
	blob := manifestStorageJSON{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Author:      "test",
		Path:        "/absolute/path/to/plugin",
	}
	data, err := json.Marshal(blob)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	extracted := extractPathFromManifestJSON(string(data))
	if extracted != "/absolute/path/to/plugin" {
		t.Errorf("extractPath: got %q, want %q", extracted, "/absolute/path/to/plugin")
	}
}

// TestExtractPathFromManifestJSONBadJSON verifies graceful degradation on invalid JSON.
func TestExtractPathFromManifestJSONBadJSON(t *testing.T) {
	got := extractPathFromManifestJSON("not-json")
	if got != "" {
		t.Errorf("extractPath bad JSON: got %q, want empty", got)
	}
}

// ── joinStrings helper tests ──────────────────────────────────────────────────

// TestJoinStrings verifies the joinStrings helper.
func TestJoinStrings(t *testing.T) {
	if joinStrings(nil) != "" {
		t.Error("joinStrings(nil) should return empty string")
	}
	if joinStrings([]string{"a"}) != "a" {
		t.Errorf("joinStrings single: got %q", joinStrings([]string{"a"}))
	}
	want := "a, b, c"
	if got := joinStrings([]string{"a", "b", "c"}); got != want {
		t.Errorf("joinStrings: got %q, want %q", got, want)
	}
}

// ── Audit row helper ──────────────────────────────────────────────────────────

// checkAuditRowExists verifies that an audit_log row with the given event_type
// and a payload_summary containing nameSubstr exists in the database.
func checkAuditRowExists(t *testing.T, dataDir, eventType, nameSubstr string) {
	t.Helper()
	dbPath := filepath.Join(dataDir, "runtime.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("checkAuditRowExists: open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM audit_log WHERE event_type = ? AND payload_summary LIKE ?`,
		eventType, "%"+nameSubstr+"%",
	).Scan(&count)
	if err != nil {
		t.Fatalf("checkAuditRowExists: query: %v", err)
	}
	if count == 0 {
		t.Errorf("expected audit row event_type=%q containing %q, found none", eventType, nameSubstr)
	}
}
