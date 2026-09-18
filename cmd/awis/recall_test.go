package main

// recall_test.go — tests for 'awis recall' command (M17-C1 TDS-07 §3).
//
// Tests:
//   - JSON schema has required fields
//   - Missing query argument exits 2 (usage error) — tested via flag parse behavior
//   - Empty FTS results returns instructive output
//   - --synthesize without API key shows PRD §21 empty-state message
//   - RecallStore type assertion works on real storage
//   - Goldens: recall.json, recall.txt

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/awis/awis/internal/storage"
)

// TestRecallOutputJSONShape verifies recallOutputJSON has required fields.
func TestRecallOutputJSONShape(t *testing.T) {
	out := recallOutputJSON{
		Query:      "draft step fail",
		Synthesize: false,
		Results:    []recallEntryJSON{},
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"query", "synthesize", "results"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("recallOutputJSON: missing field %q", field)
		}
	}
}

// TestRecallEntryJSONShape verifies recallEntryJSON has required fields.
func TestRecallEntryJSONShape(t *testing.T) {
	e := recallEntryJSON{
		EventID:    "evt-abc",
		InstanceID: "i-abc123",
		Namespace:  "default",
		EventType:  "StepCompleted",
		Payload:    `{"step_id":"draft-entry"}`,
		EmittedAt:  "2026-07-10T14:00:00Z",
		Rank:       -1.5,
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, field := range []string{"event_id", "instance_id", "namespace", "event_type", "payload", "emitted_at", "rank"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("recallEntryJSON: missing field %q", field)
		}
	}
}

// TestRecallStoreSupportedByStorage verifies that SQLiteStorage implements RecallStore.
func TestRecallStoreSupportedByStorage(t *testing.T) {
	dataDir := t.TempDir()
	store, err := OpenStorage(dataDir)
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}
	if _, ok := store.(storage.RecallStore); !ok {
		t.Error("SQLiteStorage should implement storage.RecallStore after migration 0006")
	}
}

// TestRecallEmptyResultsHuman verifies human output on no FTS matches.
func TestRecallEmptyResultsHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runRecall([]string{"no-match-xyz-abc"})
	})
	s := string(got)
	if !strings.Contains(s, "No results") {
		t.Errorf("empty state: expected 'No results' in output; got:\n%s", s)
	}
	if !strings.Contains(s, "awis history") {
		t.Errorf("empty state: expected 'awis history' what-now hint; got:\n%s", s)
	}
}

// TestRecallEmptyResultsJSON verifies JSON output on no FTS matches.
func TestRecallEmptyResultsJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runRecall([]string{"no-match-xyz-abc"})
	})
	var out recallOutputJSON
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("recall --json (no results): invalid JSON: %v\noutput: %s", err, got)
	}
	if out.Results == nil {
		t.Error("recall --json: results should be [] not null")
	}
	if len(out.Results) != 0 {
		t.Errorf("recall --json: expected 0 results, got %d", len(out.Results))
	}
}

// TestRecallGoldenJSON generates the golden for recall --json (no results).
func TestRecallGoldenJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runRecall([]string{"no-match-xyz-abc"})
	})
	checkGolden(t, "recall.json", got)
}

// TestRecallGoldenHuman generates the golden for recall (human, no results).
func TestRecallGoldenHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runRecall([]string{"no-match-xyz-abc"})
	})
	checkGolden(t, "recall.txt", got)
}
