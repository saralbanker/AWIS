package main

// audit_test.go — tests for 'awis audit' command (M17-C1 TDS-07 §3).
//
// Tests:
//   - JSON schema has required fields
//   - Empty audit log: human empty-state message
//   - Empty audit log: JSON output with entries=[]
//   - RecallStore.ListAudit returns audit rows after AppendAudit
//   - Goldens: audit.json, audit.txt

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/storage"
)

// TestAuditOutputJSONShape verifies auditOutputJSON has required fields.
func TestAuditOutputJSONShape(t *testing.T) {
	out := auditOutputJSON{
		Entries: []auditEntryJSON{
			{
				ID:             1,
				Timestamp:      "2026-07-10T14:00:00Z",
				EventType:      "WorkflowRegistered",
				Actor:          "system",
				PayloadSummary: `{"name":"test"}`,
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
	if _, ok := parsed["entries"]; !ok {
		t.Error("auditOutputJSON: missing 'entries' field")
	}
	entries, ok := parsed["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatal("auditOutputJSON: 'entries' should be a non-empty array")
	}
	entry, ok := entries[0].(map[string]any)
	if !ok {
		t.Fatal("auditOutputJSON: entries[0] should be an object")
	}
	for _, field := range []string{"id", "timestamp", "event_type", "actor", "payload_summary"} {
		if _, ok := entry[field]; !ok {
			t.Errorf("auditEntryJSON: missing field %q", field)
		}
	}
}

// TestAuditEmptyStateHuman verifies human output when audit log is empty.
func TestAuditEmptyStateHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runAudit(nil)
	})
	s := string(got)
	if !strings.Contains(s, "No audit log entries") {
		t.Errorf("empty state: expected 'No audit log entries'; got:\n%s", s)
	}
	if !strings.Contains(s, "awis start") {
		t.Errorf("empty state: expected 'awis start' what-now hint; got:\n%s", s)
	}
}

// TestAuditEmptyStateJSON verifies JSON output when audit log is empty.
func TestAuditEmptyStateJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runAudit(nil)
	})
	var out auditOutputJSON
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("audit --json (empty): invalid JSON: %v\noutput: %s", err, got)
	}
	if out.Entries == nil {
		t.Error("audit --json (empty): entries should be [] not null")
	}
}

// TestAuditListAfterAppend verifies that ListAudit returns rows after AppendAudit.
func TestAuditListAfterAppend(t *testing.T) {
	dataDir := t.TempDir()
	store, err := OpenStorage(dataDir)
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}

	// AppendAudit via SQLiteStorage (type-assert to access AppendAudit).
	type auditAppender interface {
		AppendAudit(ctx context.Context, entry storage.AuditEntry) error
	}
	aa, ok := store.(auditAppender)
	if !ok {
		t.Fatal("storage does not implement AppendAudit")
	}
	ctx := context.Background()
	if err := aa.AppendAudit(ctx, storage.AuditEntry{
		Timestamp:      time.Now(),
		EventType:      "TestEvent",
		Actor:          "test",
		PayloadSummary: `{"key":"value"}`,
	}); err != nil {
		t.Fatalf("AppendAudit: %v", err)
	}

	// ListAudit via RecallStore.
	rs, ok := store.(storage.RecallStore)
	if !ok {
		t.Fatal("storage does not implement RecallStore")
	}
	rows, err := rs.ListAudit(ctx, 10)
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if len(rows) == 0 {
		t.Error("ListAudit: expected at least one row after AppendAudit")
	}
	found := false
	for _, r := range rows {
		if r.EventType == "TestEvent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListAudit: TestEvent row not found in results: %v", rows)
	}
}

// TestAuditGoldenJSON generates the golden for audit --json (empty log).
func TestAuditGoldenJSON(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, true)
	defer restore()

	got := captureOutput(func() {
		runAudit(nil)
	})
	checkGolden(t, "audit.json", got)
}

// TestAuditGoldenHuman generates the golden for audit (human, empty log).
func TestAuditGoldenHuman(t *testing.T) {
	dataDir := t.TempDir()
	restore := setGlobals(dataDir, false)
	defer restore()

	got := captureOutput(func() {
		runAudit(nil)
	})
	checkGolden(t, "audit.txt", got)
}
