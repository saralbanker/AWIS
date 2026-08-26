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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/dsl"
	"github.com/awis/awis/internal/examples"
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

	// Apply config.yaml (B-10). `awis init` writes a config.yaml declaring
	// namespace / tick / anthropic_api_key, and until now the runtime read none
	// of it — the file was decorative. Precedence is the conventional one:
	// an explicitly-supplied flag wins over the config file, which wins over the
	// built-in default. fs.Visit reports only flags the user actually set, which
	// is how "explicitly supplied" is distinguished from "left at its default".
	cfg := loadConfigKeys()
	setFlags := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })
	if !setFlags["tick"] {
		if v, ok := cfg["tick"]; ok && v != "" {
			tickStr = v
		}
	}
	if !setFlags["namespace"] {
		if v, ok := cfg["namespace"]; ok && v != "" {
			namespace = v
		}
	}

	tick, err := time.ParseDuration(tickStr)
	if err != nil {
		fail(1, fmt.Sprintf("start: invalid --tick value %q: %s", tickStr, err),
			configPath(), "use a valid Go duration, e.g. 100ms")
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
	// The environment wins over config.yaml so a key never has to be written to
	// disk; a config value is the fallback for projects that prefer it. The key
	// itself is NEVER logged or echoed — only the resolved provider name is.
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = cfg["anthropic_api_key"]
		// A scaffolded config may carry the literal placeholder "$ANTHROPIC_API_KEY";
		// treat any unexpanded $VAR as absent rather than sending it as a credential.
		if strings.HasPrefix(apiKey, "$") {
			apiKey = ""
		}
	}
	if apiKey != "" {
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

	// 4b. Register the built-in native handlers (B-9).
	// The CLI daemon is a pre-compiled binary, so the only native handlers it can
	// dispatch are the ones compiled into it. Without this the scaffolded example
	// workflows fail on their first step with handler_not_found and the `native`
	// step type is unreachable through the shipped binary entirely. Users' own
	// native handlers require embedded SDK mode — see internal/examples for the
	// full rationale, and the unresolved-handler diagnostic below, which makes
	// that limitation visible at startup rather than at dispatch time.
	for _, h := range examples.Handlers() {
		if rerr := rt.RegisterHandler(h); rerr != nil {
			fail(1, fmt.Sprintf("start: cannot register built-in handler %q: %s", h.ID(), rerr), "", "this is a build error; reinstall awis")
		}
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

	// 5b. Warn about native handlers a discovered workflow references but that
	// this binary cannot dispatch (B-9 diagnostic). Previously such a workflow
	// started normally and then failed mid-run with handler_not_found, which
	// gives the operator no hint that the cause is structural rather than
	// transient.
	if missing := unresolvedHandlers(defs, examples.IDs()); len(missing) > 0 {
		fmt.Fprintf(os.Stderr,
			"awis: warning: %d native handler(s) referenced by your workflows are not compiled into this binary: %s\n",
			len(missing), strings.Join(missing, ", "))
		fmt.Fprintf(os.Stderr,
			"awis:   steps using them will fail with handler_not_found. Native handlers you write must be\n"+
				"awis:   registered in-process via the SDK (rt.RegisterHandler) — see examples/hello_workflow/main.go.\n")
	}

	// 6. Optional plugin discovery — if PluginStore supported.
	// Registration goes through rt.RegisterPlugin, NOT the raw PluginStore: the
	// store write only records a row, while the runtime's plugin.Manager is what
	// actually dispatches type=plugin steps. Writing the row alone left every
	// plugin step failing with plugin_not_found (B-19).
	var pluginNames []string
	if _, ok := store.(storage.PluginStore); ok {
		pluginNames = discoverPlugins(cwd, rt)
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
		emitJSON(out)
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

	// 8. Open file-sink log (sanctioned M17-C1 addition; SPEC §CE pins).
	// Creates <data-dir>/awis.log on startup (append); 'awis logs' reads this file.
	// Best-effort: errors do not abort start.
	logPath := filepath.Join(dataDir, "awis.log")
	openFileSink(logPath, pid, version, intelligenceLevel, workflowIDs)

	// 9. Write PID file.
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

	// 10. Start cron scanner goroutine (F-2).
	// Scans definitions with type:schedule triggers on a per-minute cadence.
	// Engine and core are NOT touched (SPEC non-scope); scanner lives here only.
	cronDefs := defsToCronDefs(defs)
	cronEntries := buildCronEntries(cronDefs, func(id, expr string, err error) {
		fmt.Fprintf(os.Stderr, "awis cron: workflow %q schedule %q invalid: %s\n", id, expr, err)
	})
	if len(cronEntries) > 0 {
		go runCronScanner(ctx, rt, cronEntries, time.Now)
	}

	// 11. Block until engine returns.
	_ = rt.Start(ctx)

	// Clean up PID file on graceful exit.
	_ = os.Remove(pidPath)
}

// defsToCronDefs converts parsed *core.WorkflowDefinition slices to the minimal
// cronWorkflowDef representation used by the cron scanner.
func defsToCronDefs(defs []*core.WorkflowDefinition) []cronWorkflowDef {
	result := make([]cronWorkflowDef, 0, len(defs))
	for _, d := range defs {
		triggers := make([]cronWorkflowTrigger, 0, len(d.Triggers))
		for _, t := range d.Triggers {
			triggers = append(triggers, cronWorkflowTrigger{
				triggerType: string(t.Type),
				config:      t.Config,
			})
		}
		result = append(result, cronWorkflowDef{
			id:        d.ID,
			namespace: d.Namespace,
			triggers:  triggers,
		})
	}
	return result
}

// cronSubmitter is the interface the cron scanner uses to enqueue workflow instances.
// Matches *sdk.Runtime so tests can inject a fake.
type cronSubmitter interface {
	Submit(ctx context.Context, definitionID string, inputs map[string]any) (core.InstanceID, error)
}

// runCronScanner is the cron trigger goroutine (F-2; SPEC: stdlib only, engine untouched).
// It fires when ctx is cancelled and sleeps until the next minute boundary on each cycle.
// now is injectable for deterministic fake-clock tests.
// afterFn is injectable for testing; pass nil to use time.After.
func runCronScanner(ctx context.Context, rt cronSubmitter, entries []cronEntry, now func() time.Time) {
	runCronScannerWith(ctx, rt, entries, now, nil)
}

// runCronScannerWith is the testable variant that accepts an injectable afterFn.
// afterFn(d) returns a channel that fires after d — same contract as time.After.
// If afterFn is nil, time.After is used (production path).
func runCronScannerWith(ctx context.Context, rt cronSubmitter, entries []cronEntry, now func() time.Time, afterFn func(time.Duration) <-chan time.Time) {
	if afterFn == nil {
		afterFn = time.After
	}

	// Align to the next minute boundary before starting the main loop.
	// This ensures the scanner fires at :00 of each minute.
	sleepUntilNextMinuteWith(ctx, now, afterFn)

	for {
		if ctx.Err() != nil {
			return
		}
		t := now().Truncate(time.Minute)
		for _, e := range entries {
			if e.schedule.Matches(t) {
				// Enqueue the instance through the same intake path as 'awis submit'.
				// Fire-and-forget: cron enqueue errors are best-effort (start must not fail).
				_, _ = rt.Submit(ctx, e.workflowID, nil)
			}
		}
		sleepUntilNextMinuteWith(ctx, now, afterFn)
	}
}

// sleepUntilNextMinuteWith is the injectable variant for tests.
func sleepUntilNextMinuteWith(ctx context.Context, now func() time.Time, afterFn func(time.Duration) <-chan time.Time) {
	t := now()
	next := t.Truncate(time.Minute).Add(time.Minute)
	d := next.Sub(t)
	if d <= 0 {
		d = time.Minute
	}
	select {
	case <-ctx.Done():
	case <-afterFn(d):
	}
}

// discoverPlugins finds plugins/*/awis-plugin.yaml files and registers them.
// Returns a list of plugin names that were successfully discovered/registered.
// Errors during individual plugin registration are skipped silently (F-5 V1 local-path semantics).
func discoverPlugins(cwd string, rt *sdk.Runtime) []string {
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
		name, _, _, _, perr := parsePluginManifestBytes(data)
		if perr != nil || name == "" {
			continue
		}
		// rt.RegisterPlugin parses the manifest properly (internal/plugin.ParseManifest),
		// persists it via the PluginStore AND adds it to the in-memory manager so the
		// engine can dispatch to it. A failure here is reported rather than skipped:
		// silently dropping a plugin the operator installed is how B-19 stayed hidden.
		if err := rt.RegisterPlugin(ctx, manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "awis: warning: plugin %q not registered: %s\n", name, err)
			continue
		}
		names = append(names, name)
	}
	return names
}

// unresolvedHandlers returns, sorted and deduplicated, the native step handlers
// referenced by defs that are not present in registered. It is the input to the
// startup diagnostic that makes the CLI-daemon handler limitation explicit.
func unresolvedHandlers(defs []*core.WorkflowDefinition, registered []string) []string {
	have := make(map[string]bool, len(registered))
	for _, id := range registered {
		have[id] = true
	}
	seen := make(map[string]bool)
	var missing []string
	for _, def := range defs {
		for _, step := range def.Steps {
			if step.Type != core.StepTypeNative {
				continue
			}
			ref := string(step.Handler)
			if ref == "" || have[ref] || seen[ref] {
				continue
			}
			seen[ref] = true
			missing = append(missing, ref)
		}
	}
	sort.Strings(missing)
	return missing
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

// openFileSink creates/appends to <data-dir>/awis.log and writes a structured
// startup JSON line. Best-effort: any error is silently ignored (start must not
// fail due to a log write). Called by runStart after the startup header is printed.
func openFileSink(logPath string, pid int, ver, intelligence string, workflows []string) {
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = lf.Close() }()

	line, _ := json.Marshal(map[string]any{
		"time":         time.Now().UTC().Format(time.RFC3339),
		"level":        "info",
		"msg":          "runtime started",
		"pid":          pid,
		"version":      ver,
		"intelligence": intelligence,
		"workflows":    workflows,
	})
	_, _ = fmt.Fprintf(lf, "%s\n", line)
}
