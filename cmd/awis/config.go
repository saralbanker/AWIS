package main

// config.go — 'awis config' sub-commands (TDS-07 §3 M17; M17-C2).
//
// config show     — view current config.yaml (masks secrets)
// config set      — set a key; writes ConfigChanged audit row
// config validate — schema-check config.yaml
// config edit     — open $EDITOR (skipped in CI)

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/awis/awis/internal/storage"
)

func init() {
	commands["config"] = command{fn: runConfig, summary: "Configuration sub-commands (show, set, validate, edit)"}
}

// configVisibleKeys are the ONLY config keys whose values are safe to print.
// Everything else is masked.
//
// This is an allowlist of the harmless, not a denylist of the dangerous, and
// the inversion is the whole point. The previous version matched credential-
// shaped SUBSTRINGS (key/token/secret/password/credential), which is a
// denylist wearing a disguise: it printed the value of any key whose name the
// list failed to anticipate. A key named `authorization` — the exact shape a
// non-Anthropic provider's Bearer credential takes — matched none of those
// fragments and was printed verbatim by `config show` AND written verbatim
// into the append-only audit table, where it cannot be redacted afterwards.
//
// Enumerating every credential-shaped name is not a winnable game. Enumerating
// the handful of settings an operator needs to read back is trivial, and the
// two costs are wildly asymmetric: a masked non-secret costs one
// `cat config.yaml`, while a printed secret is in terminal scrollback, CI logs
// and an immutable audit row forever.
//
// These are the recognised V1 keys that hold no credential (see
// validateConfigYAML), plus model/timeout, which are commonly present and
// never sensitive.
var configVisibleKeys = map[string]bool{
	"namespace":    true,
	"tick":         true,
	"data_dir":     true,
	"log_level":    true,
	"intelligence": true,
	"plugins_dir":  true,
	"plugins_user": true,
	"model":        true,
	"timeout":      true,
}

// isSecretConfigKey reports whether key's VALUE must be masked wherever it is
// displayed or recorded. Unknown keys are secret by default.
func isSecretConfigKey(key string) bool {
	return !configVisibleKeys[strings.ToLower(strings.TrimSpace(key))]
}

func runConfig(args []string) {
	if len(args) == 0 {
		fail(2, "config requires a sub-command", "", "awis config [show|set|validate|edit]")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "show":
		runConfigShow(rest)
	case "set":
		runConfigSet(rest)
	case "validate":
		runConfigValidate(rest)
	case "edit":
		runConfigEdit(rest)
	default:
		fail(2, fmt.Sprintf("config: unknown sub-command %q", sub), "", "awis config [show|set|validate|edit]")
	}
}

// ── config show ───────────────────────────────────────────────────────────────

// configShowOutputJSON is the JSON schema for 'awis config show --json'.
type configShowOutputJSON struct {
	ConfigFile string            `json:"config_file"`
	Keys       map[string]string `json:"keys"`
	Raw        string            `json:"raw,omitempty"`
}

func runConfigShow(args []string) {
	fs := newFlagSet("config show")
	mustParse(fs, args)

	cfgPath := configPath()

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			if globalJSON {
				out := configShowOutputJSON{
					ConfigFile: cfgPath,
					Keys:       map[string]string{},
				}
				emitJSON(out)
				return
			}
			fmt.Printf("Config file not found: %s\n\n", cfgPath)
			fmt.Println("  What now: awis init  (creates a default config.yaml)")
			return
		}
		fail(1, fmt.Sprintf("config show: cannot read config: %s", err), cfgPath, "check file permissions")
	}

	// Parse key: value lines, masking secrets (NFR-S-01).
	keys := parseConfigKeys(string(raw))
	masked := make(map[string]string, len(keys))
	for k, v := range keys {
		if isSecretConfigKey(k) {
			masked[k] = "***"
		} else {
			masked[k] = v
		}
	}

	if globalJSON {
		out := configShowOutputJSON{
			ConfigFile: cfgPath,
			Keys:       masked,
		}
		emitJSON(out)
		return
	}

	// Human output: show config file path + each key: value (secrets masked).
	fmt.Printf("Config: %s\n\n", cfgPath)
	if len(masked) == 0 {
		fmt.Println("  (empty)")
	}
	for k, v := range masked {
		fmt.Printf("  %-30s %s\n", k+":", v)
	}
}

// ── config set ────────────────────────────────────────────────────────────────

// configSetOutputJSON is the JSON schema for 'awis config set --json'.
type configSetOutputJSON struct {
	ConfigFile string `json:"config_file"`
	Key        string `json:"key"`
	Value      string `json:"value"`
	Masked     bool   `json:"masked"`
}

func runConfigSet(args []string) {
	fs := newFlagSet("config set")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 2 {
		fail(2, "config set requires <key> <value> arguments", "", "awis config set <key> <value>")
	}
	key := rest[0]
	value := strings.Join(rest[1:], " ")

	cfgPath := configPath()

	// Read existing config or start with empty.
	var existing string
	if data, err := os.ReadFile(cfgPath); err == nil {
		existing = string(data)
	}

	// Update or append the key.
	updated := setConfigKey(existing, key, value)

	// Ensure data directory exists.
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		fail(1, fmt.Sprintf("config set: cannot create data directory: %s", err), filepath.Dir(cfgPath), "check permissions")
	}
	if err := os.WriteFile(cfgPath, []byte(updated), 0o644); err != nil {
		fail(1, fmt.Sprintf("config set: cannot write config: %s", err), cfgPath, "check file permissions")
	}

	// Write ConfigChanged audit row (F-4 write-site).
	writeConfigChangedAudit(key, value)

	masked := isSecretConfigKey(key)
	displayValue := value
	if masked {
		displayValue = "***"
	}

	if globalJSON {
		out := configSetOutputJSON{
			ConfigFile: cfgPath,
			Key:        key,
			Value:      displayValue,
			Masked:     masked,
		}
		emitJSON(out)
		return
	}

	fmt.Printf("Set: %s = %s\n", key, displayValue)
	fmt.Printf("Written to: %s\n", cfgPath)
}

// writeConfigChangedAudit writes a ConfigChanged audit row (F-4).
func writeConfigChangedAudit(key, value string) {
	store, err := OpenStorage(globalDataDir)
	if err != nil {
		return // best-effort; don't fail config set on audit write error
	}
	type auditAppender interface {
		AppendAudit(ctx context.Context, entry storage.AuditEntry) error
	}
	aa, ok := store.(auditAppender)
	if !ok {
		return
	}
	// Mask secrets in audit log too.
	auditValue := value
	if isSecretConfigKey(key) {
		auditValue = "***"
	}
	payload, _ := json.Marshal(map[string]string{"key": key, "value": auditValue})
	_ = aa.AppendAudit(context.Background(), storage.AuditEntry{
		Timestamp:      time.Now(),
		EventType:      "ConfigChanged",
		Actor:          "cli",
		PayloadSummary: string(payload),
	})
}

// ── config validate ───────────────────────────────────────────────────────────

// configValidateOutputJSON is the JSON schema for 'awis config validate --json'.
type configValidateOutputJSON struct {
	ConfigFile string   `json:"config_file"`
	Valid      bool     `json:"valid"`
	Errors     []string `json:"errors"`
}

func runConfigValidate(args []string) {
	fs := newFlagSet("config validate")
	mustParse(fs, args)

	cfgPath := configPath()

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			if globalJSON {
				out := configValidateOutputJSON{ConfigFile: cfgPath, Valid: false, Errors: []string{"config file not found"}}
				emitJSON(out)
				return
			}
			fail(1, fmt.Sprintf("config validate: file not found: %s", cfgPath), cfgPath, "awis init  to create a default config")
		}
		fail(1, fmt.Sprintf("config validate: cannot read config: %s", err), cfgPath, "check file permissions")
	}

	errs := validateConfigYAML(string(raw))

	if globalJSON {
		out := configValidateOutputJSON{
			ConfigFile: cfgPath,
			Valid:      len(errs) == 0,
			Errors:     errs,
		}
		if out.Errors == nil {
			out.Errors = []string{}
		}
		emitJSON(out)
		return
	}

	if len(errs) == 0 {
		fmt.Printf("%s  valid\n", cfgPath)
		return
	}
	fmt.Printf("awis: config validation failed: %s\n", cfgPath)
	for _, e := range errs {
		fmt.Printf("  %s\n", e)
	}
	os.Exit(1)
}

// ── config edit ───────────────────────────────────────────────────────────────

func runConfigEdit(args []string) {
	fs := newFlagSet("config edit")
	mustParse(fs, args)

	// Skip in CI (no $EDITOR; SPEC pin: "skip test in CI").
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		if globalJSON {
			out := map[string]string{"error": "no $EDITOR set", "config_file": configPath()}
			emitJSON(out)
			return
		}
		fail(1, "config edit: no $EDITOR set",
			"",
			"export EDITOR=nano  or set $VISUAL to your preferred editor")
	}

	cfgPath := configPath()
	// Ensure the config file and directory exist.
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		fail(1, fmt.Sprintf("config edit: cannot create data directory: %s", err), filepath.Dir(cfgPath), "check permissions")
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err2 := os.WriteFile(cfgPath, []byte(defaultConfigYAML()), 0o644); err2 != nil {
			fail(1, fmt.Sprintf("config edit: cannot create default config: %s", err2), cfgPath, "check file permissions")
		}
	}

	cmd := exec.Command(editor, cfgPath) //nolint:gosec // user-supplied $EDITOR is intentional
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fail(1, fmt.Sprintf("config edit: editor exited with error: %s", err), editor, "check that $EDITOR is a valid executable")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// configPath returns the config.yaml path in globalDataDir.
// configPath resolves the project's config.yaml.
//
// `awis init` writes config.yaml at the PROJECT ROOT, but every config
// sub-command used to look only in <data-dir>/config.yaml — so `awis config
// show` immediately after `awis init` reported "Config file not found" while
// the file sat right there in the working directory (B-10). Resolution now
// prefers the location `awis init` actually writes, and still finds a
// config.yaml under --data-dir for projects that put one there.
//
// When neither exists the project-root path is returned, so `awis config set`
// creates the file where `awis init` would have put it.
func configPath() string {
	root := "config.yaml"
	if _, err := os.Stat(root); err == nil {
		return root
	}
	dataDir := filepath.Join(globalDataDir, "config.yaml")
	if _, err := os.Stat(dataDir); err == nil {
		return dataDir
	}
	return root
}

// loadConfigKeys reads and parses the resolved config.yaml, returning an empty
// map when the file is absent or unreadable. Config is advisory: a missing or
// malformed config must never prevent the runtime from starting.
func loadConfigKeys() map[string]string {
	raw, err := os.ReadFile(configPath())
	if err != nil {
		return map[string]string{}
	}
	return parseConfigKeys(string(raw))
}

// parseConfigKeys parses a YAML-ish config file and returns key: value pairs.
// V1 config is simple key: value lines (no nested objects; PP-7 minimal surface).
func parseConfigKeys(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		if k != "" {
			result[k] = v
		}
	}
	return result
}

// setConfigKey updates or appends a key: value pair in a config YAML string.
func setConfigKey(content, key, value string) string {
	lines := strings.Split(content, "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+":") || trimmed == key+":" {
			lines[i] = key + ": " + value
			found = true
			break
		}
	}
	if !found {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + key + ": " + value + "\n"
	}
	return strings.Join(lines, "\n")
}

// validateConfigYAML returns a list of validation errors for a config YAML string.
// V1 schema is minimal: recognized keys only, no unknown keys above threshold.
func validateConfigYAML(content string) []string {
	var errs []string
	// V1 recognized keys (PRD §7 PP-7: eight configuration options).
	recognized := map[string]bool{
		"namespace":         true,
		"tick":              true,
		"anthropic_api_key": true,
		"api_key":           true,
		"data_dir":          true,
		"log_level":         true,
		"intelligence":      true,
		"plugins_dir":       true,
		"plugins_user":      true,
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			errs = append(errs, fmt.Sprintf("invalid line (missing colon): %q", line))
			continue
		}
		k := strings.TrimSpace(line[:idx])
		if k != "" && !recognized[k] {
			errs = append(errs, fmt.Sprintf("unknown config key: %q", k))
		}
	}
	return errs
}

// defaultConfigYAML returns a minimal config.yaml content for new projects.
func defaultConfigYAML() string {
	return `# AWIS configuration
namespace: default
tick: 100ms
# intelligence: anthropic
# anthropic_api_key: (set via ANTHROPIC_API_KEY env var)
`
}
