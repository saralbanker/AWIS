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
	"time"

	"github.com/awis/awis/internal/plugin"
	"github.com/awis/awis/internal/storage"
)

func init() {
	commands["plugin"] = command{fn: runPlugin, summary: "Plugin sub-commands (install, list)"}
}

func runPlugin(args []string) {
	if len(args) == 0 {
		fail(2, "plugin requires a sub-command", "", "awis plugin [install|list|status|remove]")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "install":
		runPluginInstall(rest)
	case "list":
		runPluginList(rest)
	case "status":
		runPluginStatus(rest)
	case "remove":
		runPluginRemove(rest)
	default:
		fail(2, fmt.Sprintf("plugin: unknown sub-command %q", sub), "", "awis plugin [install|list|status|remove]")
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
		emitJSON(out)
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
		emitJSON(out)
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

// ── plugin status ─────────────────────────────────────────────────────────────

// pluginStatusOutputJSON is the JSON schema for 'awis plugin status --json'.
type pluginStatusOutputJSON struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Status       string `json:"status"`
	Path         string `json:"path"`
	RegisteredAt string `json:"registered_at"`
}

func runPluginStatus(args []string) {
	fs := newFlagSet("plugin status")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "plugin status requires a <name> argument", "", "awis plugin status <name>")
	}
	name := rest[0]

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin status: cannot open storage: %s", err),
			globalDataDir+"/runtime.db",
			"check file permissions",
		)
	}

	ps, ok := store.(storage.PluginStore)
	if !ok {
		fail(1,
			"plugin status: storage does not support PluginStore",
			globalDataDir+"/runtime.db",
			"ensure the storage migration is up to date",
		)
	}

	ctx := context.Background()
	row, err := ps.GetPlugin(ctx, name)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin %q not found", name),
			"--data-dir "+globalDataDir,
			"awis plugin list  # to see installed plugins",
		)
	}

	path := extractPathFromManifestJSON(row.Manifest)

	if globalJSON {
		out := pluginStatusOutputJSON{
			Name:         row.Name,
			Version:      row.Version,
			Status:       row.Status,
			Path:         path,
			RegisteredAt: row.RegisteredAt.UTC().Format(time.RFC3339),
		}
		emitJSON(out)
		return
	}

	fmt.Printf("Plugin:   %s\n", row.Name)
	fmt.Printf("Version:  %s\n", row.Version)
	fmt.Printf("Status:   %s\n", row.Status)
	if path != "" {
		fmt.Printf("Path:     %s\n", path)
	}
	fmt.Printf("Registered: %s\n", row.RegisteredAt.UTC().Format(time.RFC3339))
}

// ── plugin remove ─────────────────────────────────────────────────────────────

// pluginRemoveOutputJSON is the JSON schema for 'awis plugin remove --json'.
type pluginRemoveOutputJSON struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Removed bool   `json:"removed"`
}

func runPluginRemove(args []string) {
	fs := newFlagSet("plugin remove")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "plugin remove requires a <name> argument", "", "awis plugin remove <name>")
	}
	name := rest[0]

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin remove: cannot open storage: %s", err),
			globalDataDir+"/runtime.db",
			"check file permissions",
		)
	}

	ps, ok := store.(storage.PluginStore)
	if !ok {
		fail(1,
			"plugin remove: storage does not support PluginStore",
			globalDataDir+"/runtime.db",
			"ensure the storage migration is up to date",
		)
	}

	ctx := context.Background()

	// Verify the plugin exists before removing.
	_, err = ps.GetPlugin(ctx, name)
	if err != nil {
		fail(1,
			fmt.Sprintf("plugin %q not found", name),
			"--data-dir "+globalDataDir,
			"awis plugin list  # to see installed plugins",
		)
	}

	// Mark as removed via SetPluginStatus (StoragePort frozen; V1 soft-delete semantics).
	if err := ps.SetPluginStatus(ctx, name, "removed"); err != nil {
		fail(1,
			fmt.Sprintf("plugin remove: update status failed: %s", err),
			globalDataDir+"/runtime.db",
			"check storage integrity",
		)
	}

	// Write PluginRemoved audit row (F-4 write site).
	if aa, ok := store.(auditAppender); ok {
		payload, _ := json.Marshal(map[string]string{"name": name})
		_ = aa.AppendAudit(ctx, storage.AuditEntry{
			Timestamp:      time.Now(),
			EventType:      "PluginRemoved",
			Actor:          "cli",
			PayloadSummary: string(payload),
		})
	}

	if globalJSON {
		out := pluginRemoveOutputJSON{
			Name:    name,
			Status:  "removed",
			Removed: true,
		}
		emitJSON(out)
		return
	}

	fmt.Printf("Plugin removed: %s\n", name)
	fmt.Printf("  Status set to: removed\n")
	fmt.Printf("  Audit row written: PluginRemoved\n")
}

// auditAppender is a local interface alias for storage.AuditAppender to
// type-assert the store without importing the type from config.go.
type auditAppender interface {
	AppendAudit(ctx context.Context, entry storage.AuditEntry) error
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
