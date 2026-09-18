package main

// export_test.go — tests for 'awis export' command (M17-C1 TDS-07 §3).
//
// Tests:
//   - JSON schema has required fields
//   - Export on empty DB creates both JSON files
//   - instances.json and definitions.json are valid JSON
//   - Human output contains expected paths
//   - Golden: export.json, export.txt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExportOutputJSONShape verifies exportOutputJSON has required fields.
func TestExportOutputJSONShape(t *testing.T) {
	out := exportOutputJSON{
		OutDir:          "/tmp/export",
		InstancesFile:   "/tmp/export/instances.json",
		DefinitionsFile: "/tmp/export/definitions.json",
		InstanceCount:   0,
		DefinitionCount: 0,
		ExportedAt:      "2026-07-10T14:00:00Z",
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"out_dir", "instances_file", "definitions_file", "instance_count", "definition_count", "exported_at"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("exportOutputJSON: missing field %q", field)
		}
	}
}

// TestExportCreatesFiles verifies that export writes instances.json and definitions.json.
func TestExportCreatesFiles(t *testing.T) {
	dataDir := t.TempDir()
	outDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	captureOutput(func() {
		runExport([]string{"--out", outDir})
	})

	instancesFile := filepath.Join(outDir, "instances.json")
	defsFile := filepath.Join(outDir, "definitions.json")

	if _, err := os.Stat(instancesFile); err != nil {
		t.Errorf("instances.json not created: %v", err)
	}
	if _, err := os.Stat(defsFile); err != nil {
		t.Errorf("definitions.json not created: %v", err)
	}

	// Verify instances.json is valid JSON array.
	data, _ := os.ReadFile(instancesFile)
	var instances []any
	if err := json.Unmarshal(data, &instances); err != nil {
		t.Errorf("instances.json is not valid JSON array: %v\ncontent: %s", err, data)
	}

	// Verify definitions.json is valid JSON array.
	data, _ = os.ReadFile(defsFile)
	var defs []any
	if err := json.Unmarshal(data, &defs); err != nil {
		t.Errorf("definitions.json is not valid JSON array: %v\ncontent: %s", err, data)
	}
}

// TestExportJSONOutput verifies the JSON summary output.
func TestExportJSONOutput(t *testing.T) {
	dataDir := t.TempDir()
	outDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runExport([]string{"--out", outDir})
	})
	var out exportOutputJSON
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("export --json: invalid JSON: %v\noutput: %s", err, got)
	}
	if out.ExportedAt == "" {
		t.Error("exported_at should not be empty")
	}
	if out.InstancesFile == "" {
		t.Error("instances_file should not be empty")
	}
	if out.DefinitionsFile == "" {
		t.Error("definitions_file should not be empty")
	}
}

// TestExportHumanOutput verifies the human output contains expected info.
func TestExportHumanOutput(t *testing.T) {
	dataDir := t.TempDir()
	outDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runExport([]string{"--out", outDir})
	})
	s := string(got)
	if !strings.Contains(s, "Export complete") {
		t.Errorf("export human: expected 'Export complete'; got:\n%s", s)
	}
	if !strings.Contains(s, "instances.json") {
		t.Errorf("export human: expected 'instances.json'; got:\n%s", s)
	}
	if !strings.Contains(s, "definitions.json") {
		t.Errorf("export human: expected 'definitions.json'; got:\n%s", s)
	}
}

// TestExportGoldenJSON generates the golden for export --json.
func TestExportGoldenJSON(t *testing.T) {
	dataDir := t.TempDir()
	outDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runExport([]string{"--out", outDir})
	})
	// Normalize dynamic paths and timestamps.
	var raw map[string]any
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("TestExportGoldenJSON: invalid JSON: %v", err)
	}
	raw["out_dir"] = "<out-dir>"
	raw["instances_file"] = "<out-dir>/instances.json"
	raw["definitions_file"] = "<out-dir>/definitions.json"
	raw["exported_at"] = "<exported-at>"
	stable, _ := json.Marshal(raw)
	stable = append(stable, '\n')
	checkGolden(t, "export.json", stable)
}

// TestExportGoldenHuman generates the golden for export (human).
func TestExportGoldenHuman(t *testing.T) {
	dataDir := t.TempDir()
	outDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runExport([]string{"--out", outDir})
	})
	// Normalize dynamic parts.
	s := string(got)
	// Replace the timestamp portion.
	if idx := strings.Index(s, "Export complete  "); idx >= 0 {
		end := strings.Index(s[idx:], "\n")
		if end >= 0 {
			s = s[:idx] + "Export complete  <timestamp>" + s[idx+end:]
		}
	}
	// Replace the out dir path.
	s = strings.ReplaceAll(s, outDir, "<out-dir>")
	checkGolden(t, "export.txt", []byte(s))
}
