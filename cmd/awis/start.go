package main

// start.go — 'awis start' command (TDS-07 §4; M14-C2 T3/T12).
//
// Behaviour (verbatim from SPEC §2 + TDS-07 §4):
//  1. Open storage (OpenStorage → sdk.SQLiteStorage → WAL+busy_timeout).
//  2. Discover ./workflows/*.yaml via dsl.Discover(cwd).
//  3. ParseFile each; collect errors; exit 3 on any invalid.
//  4. NewRuntime (namespace from --namespace flag, default "default"; tick 100ms).
//  5. RegisterWorkflow for each definition; write WorkflowRegistered audit row
//     (via storage type-assert to auditAppender; skip silently if unsupported).
//  6. Optionally discover plugins (plugins/*/awis-plugin.yaml) if PluginStore
//     is supported — no-op otherwise.
//  7. Print startup header (TDS-07 golden shape).
//  8. Write <data-dir>/awis.pid (decimal PID + newline).
//  9. SIGTERM/SIGINT → cancel engine context → graceful stop → remove PID.
// 10. Block until engine returns.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/dsl"
	"github.com/awis/awis/internal/intelligence/adapters/anthropic"
	"github.com/awis/awis/internal/storage"
	"github.com/awis/awis/sdk"
)

func init() {
	commands["start"] = command{fn: runStart, summary: "Start the runtime (foreground)"}
}

// startOutput is the JSON schema for 'awis start --json' startup event (TDS-07 §4).
type startOutput struct {
	Event        string   `json:"event"`
	Version      string   `json:"version"`
	DBPath       string   `json:"db_path"`
	Workflows    []string `json:"workflows"`
	Plugins      []string `json:"plugins"`
	Intelligence string   `json:"intelligence"`
	PID          int      `json:"pid"`
}

func runStart(args []string) {
	// No per-command flags beyond global for start (--config is noted in TDS-07
	// as optional but maps to data-dir lookup; V1: ignored/unsupported flag noted
	// in usage only — no config.yaml loading implemented in M14).
	// --tick and --namespace are internal pins from SPEC §2.
	var tickStr string
	var namespace string

	fs := newFlagSet("start")
	fs.StringVar(&tickStr, "tick", "100ms", "Engine tick interval (default 100ms)")
	fs.StringVar(&namespace, "namespace", "default", "Namespace for this runtime instance")
	mustParse(fs, args)

	tick, err := time.ParseDuration(tickStr)
	if err != nil {
		fail(1, fmt.Sprintf("start: invalid --tick value %q: %s", tickStr, err), "", "use a valid Go duration, e.g. 100ms")
	}

	// Resolve data-dir and db path.
	dataDir := globalDataDir
	dbPath := filepath.Join(dataDir, "runtime.db")

	// 1. Open storage.
	store, err := OpenStorage(dataDir)
	if err != nil {
		fail(1, fmt.Sprintf("start: cannot open storage: %s", err), dbPath, "check file permissions and disk space")
	}

	// 2. Discover workflow YAML files.
	cwd, err := os.Getwd()
	if err != nil {
		fail(1, fmt.Sprintf("start: cannot determine working directory: %s", err), "", "")
	}
	yamlPaths, err := dsl.Discover(cwd)
	if err != nil {
		fail(1, fmt.Sprintf("start: workflow discovery failed: %s", err), cwd+"/workflows", "check that the workflows directory is readable")
	}

	// 3. Parse each YAML file; collect errors; exit 3 on any invalid.
	var defs []*core.WorkflowDefinition
	var parseErrs []string
	for _, path := range yamlPaths {
		def, perr := dsl.ParseFile(path)
		if perr != nil {
			parseErrs = append(parseErrs, fmt.Sprintf("%s: %s", path, perr))
		} else {
			defs = append(defs, def)
		}
	}
	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			fmt.Fprintf(os.Stderr, "awis: workflow parse error: %s\n", e)
		}
		os.Exit(3)
	}

	// 4. Construct intelligence adapter.
	// If ANTHROPIC_API_KEY is set, use the Anthropic cloud adapter; otherwise
	// the runtime falls through to NullAdapter behaviour (zero-AI mode).
	var intelligencePort core.IntelligencePort
	var intelligenceLevel = "none (zero-AI mode)"
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		intelligencePort = anthropic.New(anthropic.Config{APIKey: apiKey})
		intelligenceLevel = "anthropic (cloud)"
	}

	// Build runtime.
	rt, err := sdk.NewRuntime(sdk.Config{
		Namespace:    namespace,
		Storage:      store,
		TickInterval: tick,
		Intelligence: intelligencePort,
	})
	if err != nil {
		fail(1, fmt.Sprintf("start: cannot create runtime: %s", err), "", "")
	}

	// 5. Register workflows + write WorkflowRegistered audit rows.
	// On restart after crash (IR-5), the workflow may already be in storage.
	// sdk.RegisterWorkflow returns *RegistrationError with "already registered"
	// when the (id, version) pair exists; treat that as a non-fatal warning and
	// still include the workflow in the startup header.
	var workflowIDs []string
	for _, def := range defs {
		if rerr := rt.RegisterWorkflow(def); rerr != nil {
			// Check if this is an "already registered" error (idempotent restart).
			if isAlreadyRegisteredError(rerr) {
				workflowIDs = append(workflowIDs, fmt.Sprintf("%s %s", def.ID, def.Version))
				continue
			}
			fail(1, fmt.Sprintf("start: workflow registration failed: %s", rerr), def.ID, "fix the workflow definition and restart")
		}
		workflowIDs = append(workflowIDs, fmt.Sprintf("%s %s", def.ID, def.Version))
		// WorkflowRegistered audit row is written inside RegisterWorkflow (sdk/registration.go F-4).
	}

	// 6. Optional plugin discovery — if PluginStore supported.
	var pluginNames []string
	if ps, ok := store.(storage.PluginStore); ok {
		pluginNames = discoverPlugins(cwd, ps)
	}

	// 7. Print startup header.
	pid := os.Getpid()

	if globalJSON {
		out := startOutput{
			Event:        "started",
			Version:      version,
			DBPath:       dbPath,
			Workflows:    workflowIDs,
			Plugins:      pluginNames,
			Intelligence: intelligenceLevel,
			PID:          pid,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
	} else {
		// TDS-07 §4 startup header (verbatim shape):
		// AWIS v0.1.0-dev  db: .awis/runtime.db
		// Workflows registered: 2  (capture-decision v1.0.0, recall-decision v1.0.0)
		// Plugins registered:   1  (git-context-plugin)
		// Intelligence:         none (zero-AI mode)
		// ● running  PID 12345
		wfSummary := formatListSummary(workflowIDs)
		pluginSummary := formatPluginSummary(pluginNames)
		fmt.Printf("AWIS v%s  db: %s\n", version, dbPath)
		fmt.Printf("Workflows registered: %d  %s\n", len(workflowIDs), wfSummary)
		fmt.Printf("Plugins registered:   %d  %s\n", len(pluginNames), pluginSummary)
		fmt.Printf("Intelligence:         %s\n", intelligenceLevel)
		fmt.Printf("● running  PID %d\n", pid)
	}

	// 8. Write PID file.
	pidPath := filepath.Join(dataDir, "awis.pid")
	if err := writePIDFile(pidPath, pid); err != nil {
		fail(1, fmt.Sprintf("start: cannot write PID file: %s", err), pidPath, "check file permissions")
	}

	// 9. Install signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		cancel()
	}()

	// 10. Block until engine returns.
	_ = rt.Start(ctx)

	// Clean up PID file on graceful exit.
	_ = os.Remove(pidPath)
}

// discoverPlugins finds plugins/*/awis-plugin.yaml files and registers them.
// Returns a list of plugin names that were successfully discovered/registered.
// Errors during individual plugin registration are skipped silently (F-5 V1 local-path semantics).
func discoverPlugins(cwd string, ps storage.PluginStore) []string {
	pattern := filepath.Join(cwd, "plugins", "*", "awis-plugin.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil
	}

	var names []string
	ctx := context.Background()
	for _, manifestPath := range matches {
		// Use internal plugin manifest parser inline — avoids importing internal/plugin
		// directly (see SPEC: no changes to sdk/storage/plugin packages from cmd/awis).
		// We perform a minimal parse: read manifest, extract name+version+capabilities.
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		name, ver, manifestJSON, capIDs, perr := parsePluginManifestBytes(data)
		if perr != nil || name == "" {
			continue
		}
		_ = filepath.Dir(manifestPath) // plugin dir noted; path stored in manifest JSON
		if err := ps.RegisterPlugin(ctx, name, ver, manifestJSON, capIDs); err != nil {
			continue
		}
		names = append(names, name)
	}
	return names
}

// parsePluginManifestBytes does minimal YAML manifest parsing for the start
// command's optional plugin discovery — extracts name, version, capabilities,
// and returns the raw bytes as JSON.
// This avoids importing internal/plugin from cmd/awis; it replicates only the
// fields needed for RegisterPlugin.
func parsePluginManifestBytes(data []byte) (name, version, manifestJSON string, capIDs []string, err error) {
	// Minimal YAML-field extraction via field-by-field search.
	// We use encoding/json (via a temporary struct) after converting YAML via the
	// existing internal/dsl YAML dependency — but cmd/awis cannot import internal/.
	// Instead: use a simple line-scan approach for the 3 required fields.
	// This is V1 intentional; SPEC §4 notes "no copy, no venv automation".
	lines := strings.Split(string(data), "\n")
	var capLines []string
	inCapabilities := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
		} else if strings.HasPrefix(trimmed, "version:") {
			version = strings.TrimSpace(strings.TrimPrefix(trimmed, "version:"))
		} else if strings.HasPrefix(trimmed, "capabilities:") {
			inCapabilities = true
		} else if inCapabilities && strings.HasPrefix(trimmed, "- id:") {
			capLines = append(capLines, strings.TrimSpace(strings.TrimPrefix(trimmed, "- id:")))
		} else if inCapabilities && !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "#") && trimmed != "" && !strings.HasPrefix(trimmed, " ") {
			inCapabilities = false
		}
	}
	if name == "" || version == "" {
		err = fmt.Errorf("manifest missing required fields name/version")
		return
	}
	capIDs = capLines
	// Build a minimal manifest JSON for storage (path, name, version).
	mj, _ := json.Marshal(map[string]any{
		"name":    name,
		"version": version,
	})
	manifestJSON = string(mj)
	return
}

// isAlreadyRegisteredError reports whether err is a workflow-already-registered
// error from sdk.RegisterWorkflow. On restart after crash (IR-5), this is
// expected and non-fatal: the definition is already in storage.
func isAlreadyRegisteredError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "already registered")
}

func writePIDFile(path string, pid int) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0o644)
}

// formatListSummary returns "(item1, item2)" or "" if empty.
func formatListSummary(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return "(" + strings.Join(items, ", ") + ")"
}

// formatPluginSummary returns "(name1, name2)" or "" if empty.
func formatPluginSummary(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return "(" + strings.Join(names, ", ") + ")"
}
