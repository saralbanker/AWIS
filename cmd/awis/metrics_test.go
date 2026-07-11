package main

// metrics_test.go — tests for 'awis metrics' command (M17-C1 TDS-07 §3).
//
// Tests:
//   - JSON schema has required fields
//   - Empty DB produces valid output
//   - Human output contains expected sections
//   - Goldens: metrics.json, metrics.txt

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMetricsOutputJSONShape verifies metricsOutputJSON has required fields.
func TestMetricsOutputJSONShape(t *testing.T) {
	out := metricsOutputJSON{
		Namespace:         "default",
		ByStatus:          map[string]int{"completed": 5, "failed": 1},
		TotalInstances:    6,
		WorkflowBreakdown: []workflowMetricEntryJSON{},
		GeneratedAt:       "2026-07-10T14:00:00Z",
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"namespace", "by_status", "total_instances", "workflow_breakdown", "generated_at"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("metricsOutputJSON: missing field %q", field)
		}
	}
}

// TestMetricsEmptyDB verifies human output on empty database.
func TestMetricsEmptyDB(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runMetrics(nil)
	})
	s := string(got)
	if !strings.Contains(s, "METRICS") {
		t.Errorf("metrics: expected 'METRICS' header; got:\n%s", s)
	}
	if !strings.Contains(s, "Total instances: 0") {
		t.Errorf("metrics: expected 'Total instances: 0'; got:\n%s", s)
	}
}

// TestMetricsEmptyDBJSON verifies JSON output on empty database.
func TestMetricsEmptyDBJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runMetrics(nil)
	})
	var out metricsOutputJSON
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("metrics --json (empty): invalid JSON: %v\noutput: %s", err, got)
	}
	if out.TotalInstances != 0 {
		t.Errorf("expected 0 total instances, got %d", out.TotalInstances)
	}
	if out.ByStatus == nil {
		t.Error("by_status should be non-nil map (even if empty)")
	}
	if out.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}
}

// TestMetricsGoldenJSON generates the golden for metrics --json (empty db).
func TestMetricsGoldenJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runMetrics(nil)
	})
	// Normalize generated_at to a stable placeholder.
	var raw map[string]any
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("TestMetricsGoldenJSON: invalid JSON: %v", err)
	}
	raw["generated_at"] = "<generated-at>"
	stable, _ := json.Marshal(raw)
	stable = append(stable, '\n')
	checkGolden(t, "metrics.json", stable)
}

// TestMetricsGoldenHuman generates the golden for metrics (human, empty db).
func TestMetricsGoldenHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runMetrics(nil)
	})
	// Normalize timestamp in first line.
	lines := strings.SplitN(string(got), "\n", 2)
	if len(lines) >= 1 {
		// Replace the timestamp portion after "METRICS  ".
		idx := strings.Index(lines[0], "METRICS  ")
		if idx >= 0 {
			lines[0] = lines[0][:idx+9] + "<timestamp>"
		}
	}
	stable := strings.Join(lines, "\n")
	checkGolden(t, "metrics.txt", []byte(stable))
}
