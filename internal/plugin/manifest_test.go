package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testdataPath returns an absolute path under testdata/.
func testdataPath(t *testing.T, rel string) string {
	t.Helper()
	// Resolve relative to the package directory at test time.
	abs, err := filepath.Abs(filepath.Join("testdata", rel))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

// ── golden conformance ────────────────────────────────────────────────────────

// TestGoldenFilesExist verifies all 7 golden protocol files are present and
// that the 6 valid ones parse as JSON.
func TestGoldenFilesExist(t *testing.T) {
	validGoldens := []string{
		"handshake-request.json",
		"handshake-response.json",
		"execute-request.json",
		"execute-response.json",
		"execute-error.json",
		"shutdown-notification.json",
	}
	invalidGolden := "bad-jsonrpc.json"

	for _, name := range validGoldens {
		t.Run(name, func(t *testing.T) {
			path := testdataPath(t, filepath.Join("protocol", name))
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("missing golden file %s: %v", name, err)
			}
			var v any
			if err := json.Unmarshal(data, &v); err != nil {
				t.Fatalf("golden %s is not valid JSON: %v", name, err)
			}
		})
	}

	// bad-jsonrpc.json must exist but is intentionally missing the jsonrpc field.
	t.Run(invalidGolden, func(t *testing.T) {
		path := testdataPath(t, filepath.Join("protocol", invalidGolden))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("missing golden file %s: %v", invalidGolden, err)
		}
		// Must parse as JSON (it is syntactically valid — just missing the jsonrpc field).
		var v map[string]any
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatalf("bad-jsonrpc.json must be syntactically valid JSON: %v", err)
		}
		if _, ok := v["jsonrpc"]; ok {
			t.Fatalf("bad-jsonrpc.json must not contain the jsonrpc field")
		}
	})
}

// TestExecuteRequestHasCapabilityField verifies the execute-request golden
// carries an explicit "capability" field (SPEC pin; TRACEABILITY disposition).
func TestExecuteRequestHasCapabilityField(t *testing.T) {
	path := testdataPath(t, filepath.Join("protocol", "execute-request.json"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var envelope struct {
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cap, ok := envelope.Params["capability"]
	if !ok {
		t.Fatal("execute-request.json params must contain 'capability'")
	}
	if cap != "git.context.assemble" {
		t.Fatalf("expected capability 'git.context.assemble', got %q", cap)
	}
}

// ── manifest happy path ───────────────────────────────────────────────────────

// TestParseManifest_BlueprintOracle parses the Blueprint §11 manifest verbatim
// and spot-asserts ≥6 fields including capabilities[1].timeout_ms, runtime.env,
// and idle_timeout_s.
func TestParseManifest_BlueprintOracle(t *testing.T) {
	path := testdataPath(t, filepath.Join("manifests", "git-context-plugin.yaml"))
	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}

	// Field 1: name
	if m.Name != "git-context-plugin" {
		t.Errorf("name: got %q, want %q", m.Name, "git-context-plugin")
	}
	// Field 2: version
	if m.Version != "1.0.0" {
		t.Errorf("version: got %q, want %q", m.Version, "1.0.0")
	}
	// Field 3: description
	if m.Description != "Assembles git context for capture workflows" {
		t.Errorf("description: got %q", m.Description)
	}
	// Field 4: author
	if m.Author != "awis" {
		t.Errorf("author: got %q", m.Author)
	}
	// Field 5: capabilities count
	if len(m.Capabilities) != 2 {
		t.Fatalf("capabilities: got %d, want 2", len(m.Capabilities))
	}
	// Field 6: capabilities[1].timeout_ms (git.diff.fetch = 10000)
	if m.Capabilities[1].ID != "git.diff.fetch" {
		t.Errorf("capabilities[1].id: got %q, want %q", m.Capabilities[1].ID, "git.diff.fetch")
	}
	if m.Capabilities[1].TimeoutMS != 10000 {
		t.Errorf("capabilities[1].timeout_ms: got %d, want 10000", m.Capabilities[1].TimeoutMS)
	}
	// Field 7: runtime.env
	if m.Runtime.Env["GIT_TERMINAL_PROMPT"] != "0" {
		t.Errorf("runtime.env[GIT_TERMINAL_PROMPT]: got %q, want %q",
			m.Runtime.Env["GIT_TERMINAL_PROMPT"], "0")
	}
	// Field 8: idle_timeout_s
	if m.Runtime.IdleTimeoutS != 300 {
		t.Errorf("runtime.idle_timeout_s: got %d, want 300", m.Runtime.IdleTimeoutS)
	}
	// Field 9: runtime.command
	if m.Runtime.Command != "python" {
		t.Errorf("runtime.command: got %q, want %q", m.Runtime.Command, "python")
	}
	// Field 10: runtime.args
	if len(m.Runtime.Args) != 2 || m.Runtime.Args[0] != "-m" || m.Runtime.Args[1] != "git_context_plugin" {
		t.Errorf("runtime.args: got %v", m.Runtime.Args)
	}
}

// TestParseManifestBytes_BlueprintOracle confirms ParseManifestBytes works
// with the same fixture content.
func TestParseManifestBytes_BlueprintOracle(t *testing.T) {
	path := testdataPath(t, filepath.Join("manifests", "git-context-plugin.yaml"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	m, err := ParseManifestBytes(data, path)
	if err != nil {
		t.Fatalf("ParseManifestBytes: %v", err)
	}
	if m.Name != "git-context-plugin" {
		t.Errorf("name: got %q", m.Name)
	}
}

// ── unknown key rejection ─────────────────────────────────────────────────────

// TestParseManifest_UnknownKeyRejected verifies that an unknown YAML field is
// rejected and the error message carries a line number (KnownFields(true) /
// dsl pattern).
func TestParseManifest_UnknownKeyRejected(t *testing.T) {
	yaml := `name: test-plugin
version: 1.0.0
unknown_field: bad
capabilities:
  - id: cap.one
    timeout_ms: 1000
runtime:
  command: python
`
	_, err := ParseManifestBytes([]byte(yaml), "test.yaml")
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "line") {
		t.Errorf("error should contain 'line', got: %v", err)
	}
}

// ── validation failing cases ──────────────────────────────────────────────────

// TestValidation_EmptyName rejects a manifest with no name.
func TestValidation_EmptyName(t *testing.T) {
	yaml := `name: ""
version: 1.0.0
capabilities:
  - id: cap.one
    timeout_ms: 1000
runtime:
  command: python
`
	_, err := ParseManifestBytes([]byte(yaml), "test.yaml")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention 'name', got: %v", err)
	}
}

// TestValidation_BadVersion rejects a non-semver version.
func TestValidation_BadVersion(t *testing.T) {
	yaml := `name: my-plugin
version: "not-semver"
capabilities:
  - id: cap.one
    timeout_ms: 1000
runtime:
  command: python
`
	_, err := ParseManifestBytes([]byte(yaml), "test.yaml")
	if err == nil {
		t.Fatal("expected error for bad version")
	}
	if !strings.Contains(err.Error(), "semver") {
		t.Errorf("error should mention 'semver', got: %v", err)
	}
}

// TestValidation_EmptyVersion rejects a manifest with no version.
func TestValidation_EmptyVersion(t *testing.T) {
	yaml := `name: my-plugin
version: ""
capabilities:
  - id: cap.one
    timeout_ms: 1000
runtime:
  command: python
`
	_, err := ParseManifestBytes([]byte(yaml), "test.yaml")
	if err == nil {
		t.Fatal("expected error for empty version")
	}
}

// TestValidation_NoCapabilities rejects a manifest with zero capabilities.
func TestValidation_NoCapabilities(t *testing.T) {
	yaml := `name: my-plugin
version: 1.0.0
capabilities: []
runtime:
  command: python
`
	_, err := ParseManifestBytes([]byte(yaml), "test.yaml")
	if err == nil {
		t.Fatal("expected error for zero capabilities")
	}
	if !strings.Contains(err.Error(), "capability") {
		t.Errorf("error should mention 'capability', got: %v", err)
	}
}

// TestValidation_DuplicateCapabilityID rejects duplicate capability ids.
func TestValidation_DuplicateCapabilityID(t *testing.T) {
	yaml := `name: my-plugin
version: 1.0.0
capabilities:
  - id: cap.one
    timeout_ms: 1000
  - id: cap.one
    timeout_ms: 2000
runtime:
  command: python
`
	_, err := ParseManifestBytes([]byte(yaml), "test.yaml")
	if err == nil {
		t.Fatal("expected error for duplicate capability id")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention 'duplicate', got: %v", err)
	}
}

// TestValidation_EmptyCapabilityID rejects a capability with an empty id.
func TestValidation_EmptyCapabilityID(t *testing.T) {
	yaml := `name: my-plugin
version: 1.0.0
capabilities:
  - id: ""
    timeout_ms: 1000
runtime:
  command: python
`
	_, err := ParseManifestBytes([]byte(yaml), "test.yaml")
	if err == nil {
		t.Fatal("expected error for empty capability id")
	}
}

// TestValidation_EmptyRuntimeCommand rejects a manifest with no runtime command.
func TestValidation_EmptyRuntimeCommand(t *testing.T) {
	yaml := `name: my-plugin
version: 1.0.0
capabilities:
  - id: cap.one
    timeout_ms: 1000
runtime:
  command: ""
`
	_, err := ParseManifestBytes([]byte(yaml), "test.yaml")
	if err == nil {
		t.Fatal("expected error for empty runtime.command")
	}
	if !strings.Contains(err.Error(), "command") {
		t.Errorf("error should mention 'command', got: %v", err)
	}
}

// TestParseManifest_FileNotFound returns a non-nil error for a missing file.
func TestParseManifest_FileNotFound(t *testing.T) {
	_, err := ParseManifest("/nonexistent/path/awis-plugin.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
