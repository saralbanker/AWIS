package main

// logs_test.go — tests for 'awis logs' command (M17-C1 TDS-07 §3).
//
// Tests:
//   - JSON schema (logsOutputJSON) has required fields
//   - Empty state when log file absent (human + JSON)
//   - Log level filter helper
//   - Real log content is returned when file exists
//   - Goldens: logs.json, logs.txt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLogsOutputJSONShape verifies logsOutputJSON has required fields.
func TestLogsOutputJSONShape(t *testing.T) {
	out := logsOutputJSON{
		LogFile: "/tmp/.awis/awis.log",
		Lines:   []interface{}{},
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"log_file", "lines"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("logsOutputJSON: missing field %q", field)
		}
	}
}

// TestLogsEmptyStateHuman verifies human output when log file doesn't exist.
func TestLogsEmptyStateHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runLogs(nil)
	})
	s := string(got)
	if !strings.Contains(s, "No log file") {
		t.Errorf("empty state: expected 'No log file' in output; got:\n%s", s)
	}
	if !strings.Contains(s, "awis start") {
		t.Errorf("empty state: expected 'awis start' what-now hint; got:\n%s", s)
	}
}

// TestLogsEmptyStateJSON verifies JSON output when log file doesn't exist.
func TestLogsEmptyStateJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runLogs(nil)
	})
	var out logsOutputJSON
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("logs --json (no file): invalid JSON: %v\noutput: %s", err, got)
	}
	if out.Lines == nil {
		t.Error("logs --json (no file): lines should be [] not null")
	}
}

// TestLogsWithContent verifies that existing log lines are returned.
func TestLogsWithContent(t *testing.T) {
	dataDir := t.TempDir()
	logPath := filepath.Join(dataDir, "awis.log")
	content := `{"time":"2026-07-10T14:00:00Z","level":"info","msg":"runtime started","pid":12345}
{"time":"2026-07-10T14:00:01Z","level":"info","msg":"step started","step_id":"draft-entry"}
{"time":"2026-07-10T14:00:02Z","level":"error","msg":"step failed","step_id":"draft-entry"}
`
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runLogs([]string{"--tail=10"})
	})
	s := string(got)
	if !strings.Contains(s, "runtime started") {
		t.Errorf("logs: expected 'runtime started' in output; got:\n%s", s)
	}
	if !strings.Contains(s, "step started") {
		t.Errorf("logs: expected 'step started' in output; got:\n%s", s)
	}
}

// TestLogsLevelFilter verifies that level filtering works.
func TestLogsLevelFilter(t *testing.T) {
	dataDir := t.TempDir()
	logPath := filepath.Join(dataDir, "awis.log")
	content := `{"time":"2026-07-10T14:00:00Z","level":"info","msg":"runtime started"}
{"time":"2026-07-10T14:00:02Z","level":"error","msg":"step failed"}
`
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runLogs([]string{"--level=error"})
	})
	s := string(got)
	if !strings.Contains(s, "step failed") {
		t.Errorf("level filter: expected error line; got:\n%s", s)
	}
	if strings.Contains(s, "runtime started") {
		t.Errorf("level filter: info line should be filtered out; got:\n%s", s)
	}
}

// TestLogLineMatchesLevel verifies level-matching helper.
func TestLogLineMatchesLevel(t *testing.T) {
	if !logLineMatchesLevel(`{"level":"info","msg":"ok"}`, "info") {
		t.Error("should match info level")
	}
	if !logLineMatchesLevel(`{"level":"error","msg":"fail"}`, "error") {
		t.Error("should match error level")
	}
	if logLineMatchesLevel(`{"level":"info","msg":"ok"}`, "error") {
		t.Error("should not match error for info line")
	}
	if !logLineMatchesLevel("level=info msg=ok", "info") {
		t.Error("should match logfmt info")
	}
}

// TestLogsGoldenJSON generates the golden for logs --json (no file).
func TestLogsGoldenJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runLogs(nil)
	})
	// Normalize the log_file path to a stable placeholder.
	logPath := filepath.Join(dataDir, "awis.log")
	stable := strings.ReplaceAll(string(got), logPath, "<log-path>")
	checkGolden(t, "logs.json", []byte(stable))
}

// TestLogsGoldenHuman generates the golden for logs (human, no file).
func TestLogsGoldenHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runLogs(nil)
	})
	logPath := filepath.Join(dataDir, "awis.log")
	stable := strings.ReplaceAll(string(got), logPath, "<log-path>")
	checkGolden(t, "logs.txt", []byte(stable))
}
