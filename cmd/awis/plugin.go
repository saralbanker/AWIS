package main

// plugin.go — 'awis plugin' sub-commands (TDS-07 §4; M14-C4 T11).
//
// plugin install <path> — parse awis-plugin.yaml; register via PluginStore
//                         (local-path semantics: no copy); writes PluginRegistered audit row.
// plugin list           — list all installed plugins from PluginStore.
//
// V1 semantics (F-5): registers the local path in-place; no file copy, no venv automation.
// plugin status/remove are M17 — do NOT implement here.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/awis/awis/internal/plugin"
	"github.com/awis/awis/internal/storage"
)

func init() {
	commands["plugin"] = command{fn: runPlugin, summary: "Plugin sub-commands (install, list)"}
}

func runPlugin(args []string) {
	if len(args) == 0 {
		fail(2, "plugin requires a sub-command", "", "awis plugin [install|list]")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "install":
		runPluginInstall(rest)
	case "list":
		runPluginList(rest)
	default:
		fail(2, fmt.Sprintf("plugin: unknown sub-command %q", sub), "", "awis plugin [install|list]")
	}
}

// ── plugin install ────────────────────────────────────────────────────────────

// pluginInstallOutputJSON is the TDS-07 §4 JSON schema for plugin install.
type pluginInstallOutputJSON struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Path     string   `json:"path"`
	Provides []string `json:"provides"`
}

func runPluginInstall(args []string) {
	fs := newFlagSet("plugin install")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "plugin install requires a <path> argument", "", "awis plugin install <path>")
	}
	pluginPath := rest[0]

	// Resolve the manifest path: if pluginPath is a directory, append the manifest filename.
	manifestPath := pluginPath
	info, err := os.Stat(pluginPath)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin install: path not found: %s", err),
			pluginPath,
			"check the plugin directory path",
		)
	}
	if info.IsDir() {
		manifestPath = filepath.Join(pluginPath, "awis-plugin.yaml")
	}

	// Parse and validate the manifest.
	m, err := plugin.ParseManifest(manifestPath)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin install: manifest missing or invalid: %s", err),
			manifestPath,
			"ensure the directory contains a valid awis-plugin.yaml",
		)
	}

	// Build the canonical absolute path for storage (V1 local-path semantics).
	absPath, err := filepath.Abs(pluginPath)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin install: cannot resolve absolute path: %s", err),
			pluginPath,
			"check the plugin directory path",
		)
	}

	// Build capability IDs list.
	capIDs := make([]string, 0, len(m.Capabilities))
	for _, c := range m.Capabilities {
		capIDs = append(capIDs, c.ID)
	}

	// Build manifest JSON for storage (includes path as V1 local-path annotation).
	manifestJSON, err := buildManifestJSON(m, absPath)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin install: cannot serialize manifest: %s", err),
			manifestPath,
			"",
		)
	}

	// Open storage and register.
	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin install: cannot open storage: %s", err),
			globalDataDir+"/runtime.db",
			"check file permissions",
		)
	}

	// Type-assert to PluginStore.
	ps, ok := store.(storage.PluginStore)
	if !ok {
		fail(1,
			"plugin install: storage does not support PluginStore",
			globalDataDir+"/runtime.db",
			"ensure the storage migration is up to date",
		)
	}

	ctx := context.Background()
	if err := ps.RegisterPlugin(ctx, m.Name, m.Version, manifestJSON, capIDs); err != nil {
		fail(1,
			fmt.Sprintf("plugin install: registration failed: %s", err),
			globalDataDir+"/runtime.db",
			"check storage integrity",
		)
	}

	if globalJSON {
		out := pluginInstallOutputJSON{
			Name:     m.Name,
			Version:  m.Version,
			Path:     absPath,
			Provides: capIDs,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	// TDS-07 §4 human output:
	// Plugin installed: git-context-plugin  v0.1.0
	//   Path:     plugins/git-context-plugin
	//   Provides: git.context.assemble, git.diff.fetch
	fmt.Printf("Plugin installed: %s  v%s\n", m.Name, m.Version)
	fmt.Printf("  Path:     %s\n", absPath)
	fmt.Printf("  Provides: %s\n", joinStrings(capIDs))
}

// ── plugin list ───────────────────────────────────────────────────────────────

// pluginListOutputJSON is the TDS-07 §4 JSON schema for plugin list.
type pluginListOutputJSON struct {
	Plugins []pluginListEntryJSON `json:"plugins"`
}

// pluginListEntryJSON is a single plugin entry in the plugin list JSON output.
type pluginListEntryJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Path    string `json:"path"`
}

func runPluginList(args []string) {
	fs := newFlagSet("plugin list")
	mustParse(fs, args)

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin list: cannot open storage: %s", err),
			globalDataDir+"/runtime.db",
			"check file permissions",
		)
	}

	ps, ok := store.(storage.PluginStore)
	if !ok {
		fail(1,
			"plugin list: storage does not support PluginStore",
			globalDataDir+"/runtime.db",
			"ensure the storage migration is up to date",
		)
	}

	ctx := context.Background()
	rows, err := ps.ListPlugins(ctx)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin list: storage query failed: %s", err),
			globalDataDir+"/runtime.db",
			"check storage integrity",
		)
	}

	// Extract path from stored manifest JSON (V1 local-path semantics).
	entries := make([]pluginListEntryJSON, 0, len(rows))
	for _, r := range rows {
		path := extractPathFromManifestJSON(r.Manifest)
		entries = append(entries, pluginListEntryJSON{
			Name:    r.Name,
			Version: r.Version,
			Status:  r.Status,
			Path:    path,
		})
	}

	if globalJSON {
		out := pluginListOutputJSON{Plugins: entries}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	// TDS-07 §4 human output:
	// INSTALLED PLUGINS
	//
	//   NAME                  VERSION   STATUS   PATH
	//   git-context-plugin    0.1.0     active   plugins/git-context-plugin
	fmt.Println("INSTALLED PLUGINS")
	fmt.Println()
	fmt.Printf("  %-24s %-10s %-10s %s\n", "NAME", "VERSION", "STATUS", "PATH")
	for _, e := range entries {
		fmt.Printf("  %-24s %-10s %-10s %s\n", e.Name, e.Version, e.Status, e.Path)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// manifestStorageJSON is the JSON structure stored in the plugins.manifest column.
// It carries the plugin manifest fields plus the V1 local path annotation.
type manifestStorageJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Path        string `json:"path"` // V1 local-path annotation (no copy semantics)
}

// buildManifestJSON constructs the JSON blob stored in the plugins.manifest column.
// The path field records the absolute local path (V1 semantics).
func buildManifestJSON(m *plugin.Manifest, absPath string) (string, error) {
	blob := manifestStorageJSON{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Author:      m.Author,
		Path:        absPath,
	}
	b, err := json.Marshal(blob)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// extractPathFromManifestJSON extracts the path field from the stored manifest JSON.
// Returns empty string on any parse failure (graceful degradation).
func extractPathFromManifestJSON(manifestJSON string) string {
	var blob manifestStorageJSON
	if err := json.Unmarshal([]byte(manifestJSON), &blob); err != nil {
		return ""
	}
	return blob.Path
}

// joinStrings joins a slice of strings with ", ".
func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += ", " + s
	}
	return result
}
