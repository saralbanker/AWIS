package main

// prune_test.go — tests for 'awis prune-events' command (M17-C1 TDS-07 §3).
//
// Tests:
//   - JSON schema has required fields
//   - dry_run must always be true in JSON output
//   - Empty DB: 0 eligible, 0 scanned
//   - Human output contains expected note about V1 dry-run only
//   - Goldens: prune_events.json, prune_events.txt

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestPruneReportJSONShape verifies pruneReportJSON has required fields.
func TestPruneReportJSONShape(t *testing.T) {
	out := pruneReportJSON{
		DryRun:        true,
		Before:        "2026-07-03T00:00:00Z",
		EligibleCount: 0,
		TotalScanned:  0,
		Note:          "Destructive prune is post-V1. This is a dry-run report only.",
		ScannedAt:     "2026-07-10T14:00:00Z",
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"dry_run", "before", "eligible_count", "total_scanned", "note", "scanned_at"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("pruneReportJSON: missing field %q", field)
		}
	}
	// dry_run must always be true (V1 constraint).
	if dr, ok := parsed["dry_run"].(bool); !ok || !dr {
		t.Error("pruneReportJSON: dry_run must be true in V1")
	}
}

// TestPruneEmptyDBJSON verifies JSON output on empty database with dry-run.
func TestPruneEmptyDBJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runPruneEvents([]string{"--dry-run"})
	})
	var out pruneReportJSON
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("prune-events --json --dry-run (empty): invalid JSON: %v\noutput: %s", err, got)
	}
	if !out.DryRun {
		t.Error("dry_run should be true")
	}
	if out.EligibleCount != 0 {
		t.Errorf("eligible_count: expected 0, got %d", out.EligibleCount)
	}
	if out.TotalScanned != 0 {
		t.Errorf("total_scanned: expected 0, got %d", out.TotalScanned)
	}
	if out.Before == "" {
		t.Error("before should not be empty")
	}
}

// TestPruneEmptyDBHuman verifies human output on empty database.
func TestPruneEmptyDBHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runPruneEvents([]string{"--dry-run"})
	})
	s := string(got)
	if !strings.Contains(s, "PRUNE-EVENTS") {
		t.Errorf("prune-events: expected 'PRUNE-EVENTS' header; got:\n%s", s)
	}
	if !strings.Contains(s, "Destructive prune is post-V1") {
		t.Errorf("prune-events: expected V1 note; got:\n%s", s)
	}
}

// TestPruneGoldenJSON generates the golden for prune-events --json --dry-run.
func TestPruneGoldenJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runPruneEvents([]string{"--dry-run"})
	})
	// Normalize dynamic fields.
	var raw map[string]any
	if err := json.Unmarshal(got, &raw); err != nil {
		t.Fatalf("TestPruneGoldenJSON: invalid JSON: %v", err)
	}
	raw["before"] = "<before>"
	raw["scanned_at"] = "<scanned-at>"
	stable, _ := json.Marshal(raw)
	stable = append(stable, '\n')
	checkGolden(t, "prune_events.json", stable)
}

// TestPruneGoldenHuman generates the golden for prune-events (human, dry-run).
func TestPruneGoldenHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runPruneEvents([]string{"--dry-run"})
	})
	// Normalize cutoff line.
	lines := strings.Split(string(got), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Cutoff:") {
			idx := strings.Index(line, "Cutoff:")
			lines[i] = line[:idx+7] + "         <cutoff>"
		}
	}
	checkGolden(t, "prune_events.txt", []byte(strings.Join(lines, "\n")))
}
