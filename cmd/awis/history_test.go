package main

// history_test.go — tests for 'awis history' command (M17-C1 TDS-07 §3).
//
// Tests:
//   - JSON schema has required fields
//   - Human output contains expected headers
//   - Empty state message
//   - Duration formatting
//   - Golden: history.json, history.txt

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestHistoryOutputJSONShape verifies historyOutputJSON has required fields.
func TestHistoryOutputJSONShape(t *testing.T) {
	out := historyOutputJSON{
		Instances: []historyEntryJSON{
			{
				InstanceID:  "i-abc123",
				WorkflowID:  "capture-decision",
				Namespace:   "default",
				Status:      "completed",
				DurationMs:  89000,
				CompletedAt: "2026-07-10T14:21:32Z",
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
	if _, ok := parsed["instances"]; !ok {
		t.Error("historyOutputJSON: missing 'instances' field")
	}
	instances, ok := parsed["instances"].([]any)
	if !ok || len(instances) == 0 {
		t.Fatal("historyOutputJSON: 'instances' should be a non-empty array")
	}
	entry, ok := instances[0].(map[string]any)
	if !ok {
		t.Fatal("historyOutputJSON: instances[0] should be an object")
	}
	for _, field := range []string{"instance_id", "workflow_id", "namespace", "status", "duration_ms", "completed_at"} {
		if _, ok := entry[field]; !ok {
			t.Errorf("historyEntryJSON: missing field %q", field)
		}
	}
}

// TestHistoryEmptyStateHuman verifies that an empty-state human output contains
// the required empty-state text (PP-5: empty states teach, not apologize).
func TestHistoryEmptyStateHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runHistory(nil)
	})
	s := string(got)
	if !strings.Contains(s, "No completed") {
		t.Errorf("empty state: expected 'No completed' in output; got:\n%s", s)
	}
	if !strings.Contains(s, "awis submit") {
		t.Errorf("empty state: expected 'awis submit' what-now hint; got:\n%s", s)
	}
}

// TestHistoryEmptyStateJSON verifies that JSON output on empty DB is valid with instances=[].
func TestHistoryEmptyStateJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runHistory(nil)
	})
	var out historyOutputJSON
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("history --json (empty): invalid JSON: %v\noutput: %s", err, got)
	}
	// instances must be non-nil (empty array, not null).
	if out.Instances == nil {
		t.Error("history --json (empty): instances should be [] not null")
	}
}

// TestHistorySymbol verifies historySymbol returns correct symbols.
func TestHistorySymbol(t *testing.T) {
	cases := []struct {
		status string
		want   string
	}{
		{"completed", "✓"},
		{"compensated", "✓"},
		{"failed", "✗"},
		{"cancelled", "✗"},
		{"compensation_failed", "✗"},
		{"unknown", "·"},
	}
	for _, c := range cases {
		if got := historySymbol(c.status); got != c.want {
			t.Errorf("historySymbol(%q) = %q, want %q", c.status, got, c.want)
		}
	}
}

// TestFormatDurationMs verifies formatDurationMs produces expected output.
func TestFormatDurationMs(t *testing.T) {
	cases := []struct {
		ms   int
		want string
	}{
		{0, "0ms"},
		{500, "500ms"},
		{1000, "1.0s"},
		{89000, "1m29s"},
		{3600000, "60m0s"},
	}
	for _, c := range cases {
		if got := formatDurationMs(c.ms); got != c.want {
			t.Errorf("formatDurationMs(%d) = %q, want %q", c.ms, got, c.want)
		}
	}
}

// TestHistoryGoldenJSON generates the golden file for history --json on empty db.
func TestHistoryGoldenJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runHistory(nil)
	})
	checkGolden(t, "history.json", got)
}

// TestHistoryGoldenHuman generates the golden file for history (human) on empty db.
func TestHistoryGoldenHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runHistory(nil)
	})
	checkGolden(t, "history.txt", got)
}
