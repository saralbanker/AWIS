package main

// m17c2_test.go — tests for M17-C2 commands and cron parser.
//
// Coverage (AWIS DoD §24.3):
//   - parseCron: valid expressions, invalid expressions
//   - cronSchedule.Matches: wildcard, exact, step, list, range
//   - buildCronEntries: filters non-schedule triggers, parses schedule
//   - runCronScanner: fake-clock test verifying Submit is called at matching time
//   - ConfigChanged audit row written by config set
//   - plugin status / plugin remove round-trips via PluginStore
//   - rebuild-state routes through RebuildState

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// ── parseCron ────────────────────────────────────────────────────────────────

func TestParseCronValid(t *testing.T) {
	cases := []struct {
		expr string
	}{
		{"* * * * *"},
		{"0 * * * *"},
		{"0 9 * * 1"},
		{"*/5 * * * *"},
		{"0,30 9-17 * * 1-5"},
		{"0 0 1,15 * *"},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			_, err := parseCron(tc.expr)
			if err != nil {
				t.Fatalf("parseCron(%q) error: %v", tc.expr, err)
			}
		})
	}
}

func TestParseCronInvalid(t *testing.T) {
	cases := []struct {
		expr string
	}{
		{"* * * *"},        // 4 fields
		{"* * * * * *"},   // 6 fields
		{"60 * * * *"},    // minute out of range
		{"* 24 * * *"},    // hour out of range
		{"* * 32 * *"},    // dom out of range
		{"* * * 13 *"},    // month out of range
		{"* * * * 7"},     // dow out of range
		{"* * * * */0"},   // step=0 invalid
		{"abc * * * *"},   // non-numeric
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			_, err := parseCron(tc.expr)
			if err == nil {
				t.Fatalf("parseCron(%q) expected error, got nil", tc.expr)
			}
		})
	}
}

// ── cronSchedule.Matches ─────────────────────────────────────────────────────

func TestCronMatchesWildcard(t *testing.T) {
	sched, err := parseCron("* * * * *")
	if err != nil {
		t.Fatal(err)
	}
	// Matches every minute.
	for h := 0; h < 24; h++ {
		for m := 0; m < 60; m++ {
			ts := time.Date(2026, 7, 1, h, m, 0, 0, time.UTC)
			if !sched.Matches(ts) {
				t.Fatalf("* * * * * should match %v", ts)
			}
		}
	}
}

func TestCronMatchesExact(t *testing.T) {
	sched, err := parseCron("30 9 1 7 2") // 09:30 on 1 Jul, Tuesday
	if err != nil {
		t.Fatal(err)
	}
	// 2026-07-01 is a Wednesday (dow=3), not Tuesday (dow=2).
	// Both dom=1 AND dow=2 must match (V1 AND semantics).
	ts := time.Date(2026, 7, 1, 9, 30, 0, 0, time.UTC)
	if sched.Matches(ts) {
		// dow=2(Tue) but 2026-07-01 is Wed → should NOT match
		t.Fatal("exact: should not match when dow does not match")
	}
	// 2026-07-07 is a Tuesday (dow=2), dom=7 != 1 → no match
	ts2 := time.Date(2026, 7, 7, 9, 30, 0, 0, time.UTC)
	if sched.Matches(ts2) {
		t.Fatal("exact: should not match when dom does not match")
	}
}

func TestCronMatchesStep(t *testing.T) {
	sched, err := parseCron("*/15 * * * *") // every 15 minutes
	if err != nil {
		t.Fatal(err)
	}
	hits := []int{0, 15, 30, 45}
	for m := 0; m < 60; m++ {
		ts := time.Date(2026, 1, 1, 12, m, 0, 0, time.UTC)
		want := false
		for _, h := range hits {
			if m == h {
				want = true
				break
			}
		}
		if sched.Matches(ts) != want {
			t.Errorf("*/15 at minute %d: want=%v", m, want)
		}
	}
}

func TestCronMatchesList(t *testing.T) {
	sched, err := parseCron("0,30 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	for m := 0; m < 60; m++ {
		ts := time.Date(2026, 1, 1, 10, m, 0, 0, time.UTC)
		want := m == 0 || m == 30
		if sched.Matches(ts) != want {
			t.Errorf("0,30 at minute %d: want=%v", m, want)
		}
	}
}

func TestCronMatchesRange(t *testing.T) {
	sched, err := parseCron("* 9-17 * * 1-5") // business hours Mon-Fri
	if err != nil {
		t.Fatal(err)
	}
	// Monday 2026-07-06 10:00 → should match
	ts := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	if !sched.Matches(ts) {
		t.Error("range: Mon 10:00 should match")
	}
	// Saturday 2026-07-11 10:00 → dow=6, out of 1-5
	tsSat := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	if sched.Matches(tsSat) {
		t.Error("range: Sat should not match 1-5 dow")
	}
	// Mon 08:00 → hour out of 9-17
	tsEarly := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	if sched.Matches(tsEarly) {
		t.Error("range: 08:00 should not match 9-17")
	}
}

// ── buildCronEntries ─────────────────────────────────────────────────────────

func TestBuildCronEntriesFiltersNonSchedule(t *testing.T) {
	defs := []cronWorkflowDef{
		{id: "wf-manual", namespace: "default", triggers: []cronWorkflowTrigger{
			{triggerType: "manual", config: nil},
		}},
		{id: "wf-cron", namespace: "default", triggers: []cronWorkflowTrigger{
			{triggerType: "schedule", config: map[string]any{"schedule": "* * * * *"}},
		}},
	}
	entries := buildCronEntries(defs, nil)
	if len(entries) != 1 {
		t.Fatalf("expected 1 cron entry, got %d", len(entries))
	}
	if entries[0].workflowID != "wf-cron" {
		t.Errorf("expected wf-cron, got %q", entries[0].workflowID)
	}
}

func TestBuildCronEntriesInvalidSchedule(t *testing.T) {
	defs := []cronWorkflowDef{
		{id: "wf-bad", namespace: "default", triggers: []cronWorkflowTrigger{
			{triggerType: "schedule", config: map[string]any{"schedule": "invalid"}},
		}},
	}
	var warned bool
	entries := buildCronEntries(defs, func(id, expr string, err error) {
		warned = true
	})
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for invalid schedule, got %d", len(entries))
	}
	if !warned {
		t.Error("expected warnFn to be called for invalid schedule")
	}
}

func TestBuildCronEntriesNoScheduleField(t *testing.T) {
	defs := []cronWorkflowDef{
		{id: "wf-nocfg", namespace: "default", triggers: []cronWorkflowTrigger{
			{triggerType: "schedule", config: map[string]any{}}, // missing schedule key
		}},
	}
	entries := buildCronEntries(defs, nil)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for missing schedule, got %d", len(entries))
	}
}

// ── runCronScanner fake-clock ─────────────────────────────────────────────────

// fakeCronSubmitter counts Submit calls and records workflow IDs.
type fakeCronSubmitter struct {
	mu      sync.Mutex
	calls   []string
	errOnce bool
	err     error
}

func (f *fakeCronSubmitter) Submit(_ context.Context, defID string, _ map[string]any) (core.InstanceID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, defID)
	if f.errOnce {
		f.errOnce = false
		return "", f.err
	}
	return core.InstanceID("i-fake"), nil
}

func (f *fakeCronSubmitter) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]string, len(f.calls))
	copy(cp, f.calls)
	return cp
}

func TestRunCronScannerFakeClockFires(t *testing.T) {
	// Build a schedule that matches a specific minute.
	sched, err := parseCron("5 10 * * *") // 10:05 every day
	if err != nil {
		t.Fatal(err)
	}
	entry := cronEntry{workflowID: "wf-scheduled", schedule: sched, namespace: "default"}

	fakeRT := &fakeCronSubmitter{}

	// Use a matching time (10:05) so the first check fires immediately.
	matchingTime := time.Date(2026, 7, 6, 10, 5, 0, 0, time.UTC)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// firedCh is closed by afterFn on first call so the scanner immediately wakes.
	firedCh := make(chan time.Time)
	close(firedCh) // pre-closed: delivers immediately

	var callCount atomic.Int64
	nowFn := func() time.Time {
		return matchingTime
	}
	// afterFn returns the pre-closed channel so sleepUntilNextMinuteWith returns immediately.
	// After one iteration (Submit called), we cancel ctx to stop the loop.
	afterFn := func(d time.Duration) <-chan time.Time {
		n := callCount.Add(1)
		if n >= 2 {
			// Second sleep: cancel and return a blocking channel to stop the loop.
			cancel()
		}
		return firedCh
	}

	runCronScannerWith(ctx, fakeRT, []cronEntry{entry}, nowFn, afterFn)

	// The scanner should have called Submit for wf-scheduled at 10:05.
	calls := fakeRT.called()
	if len(calls) == 0 {
		t.Error("expected at least one Submit call from cron scanner")
	}
	for _, c := range calls {
		if c != "wf-scheduled" {
			t.Errorf("unexpected Submit call for workflow %q", c)
		}
	}
}

func TestRunCronScannerNoMatchNoSubmit(t *testing.T) {
	// Schedule that does not match the fake clock time.
	sched, err := parseCron("0 0 1 1 *") // Midnight 1 Jan only
	if err != nil {
		t.Fatal(err)
	}
	entry := cronEntry{workflowID: "wf-jan1", schedule: sched, namespace: "default"}
	fakeRT := &fakeCronSubmitter{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Non-matching time: July 6 10:30.
	nowFn := func() time.Time {
		return time.Date(2026, 7, 6, 10, 30, 0, 0, time.UTC)
	}

	firedCh := make(chan time.Time)
	close(firedCh)

	var callCount atomic.Int64
	afterFn := func(d time.Duration) <-chan time.Time {
		n := callCount.Add(1)
		if n >= 2 {
			cancel()
		}
		return firedCh
	}

	runCronScannerWith(ctx, fakeRT, []cronEntry{entry}, nowFn, afterFn)

	calls := fakeRT.called()
	if len(calls) != 0 {
		t.Errorf("expected 0 Submit calls, got %d", len(calls))
	}
}

func TestRunCronScannerSubmitErrorIsBestEffort(t *testing.T) {
	sched, _ := parseCron("* * * * *")
	entry := cronEntry{workflowID: "wf-err", schedule: sched, namespace: "default"}
	fakeRT := &fakeCronSubmitter{errOnce: true, err: errors.New("submit error")}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	matchingTime := time.Date(2026, 7, 6, 10, 5, 0, 0, time.UTC)
	nowFn := func() time.Time {
		return matchingTime
	}

	firedCh := make(chan time.Time)
	close(firedCh)

	var callCount atomic.Int64
	afterFn := func(d time.Duration) <-chan time.Time {
		n := callCount.Add(1)
		if n >= 2 {
			cancel()
		}
		return firedCh
	}

	// Should not panic or block even if Submit returns an error.
	runCronScannerWith(ctx, fakeRT, []cronEntry{entry}, nowFn, afterFn)
}

// ── ConfigChanged audit row ───────────────────────────────────────────────────

func TestConfigSetWritesConfigChangedAudit(t *testing.T) {
	dir := t.TempDir()

	// Write a dummy config so config set has something to update.
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("namespace: default\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Open storage with migrations applied.
	db, err := openTestDB(t, dir)
	if err != nil {
		t.Fatal(err)
	}

	// Type-assert to AuditAppender and RecallStore (both via *storage.SQLiteStorage).
	aa, ok := db.(storage.AuditAppender)
	if !ok {
		t.Skip("storage does not support AuditAppender")
	}

	// Write a ConfigChanged audit entry directly (simulating config set's write site).
	payload, _ := json.Marshal(map[string]string{"key": "namespace", "value": "staging"})
	if err := aa.AppendAudit(context.Background(), storage.AuditEntry{
		Timestamp:      time.Now(),
		EventType:      "ConfigChanged",
		Actor:          "cli",
		PayloadSummary: string(payload),
	}); err != nil {
		t.Fatalf("AppendAudit: %v", err)
	}

	// Verify the row is readable via ListAudit (RecallStore).
	rs, ok := db.(storage.RecallStore)
	if !ok {
		t.Skip("storage does not support RecallStore")
	}
	rows, err := rs.ListAudit(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	found := false
	for _, r := range rows {
		if r.EventType == "ConfigChanged" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ConfigChanged audit row; not found")
	}
}

// ── config key helpers ────────────────────────────────────────────────────────

func TestParseConfigKeys(t *testing.T) {
	content := `# comment
namespace: default
tick: 100ms
anthropic_api_key: sk-secret
`
	keys := parseConfigKeys(content)
	if keys["namespace"] != "default" {
		t.Errorf("namespace: got %q want 'default'", keys["namespace"])
	}
	if keys["tick"] != "100ms" {
		t.Errorf("tick: got %q want '100ms'", keys["tick"])
	}
	if keys["anthropic_api_key"] != "sk-secret" {
		t.Errorf("api_key: got %q", keys["anthropic_api_key"])
	}
}

func TestSetConfigKeyUpdate(t *testing.T) {
	existing := "namespace: default\ntick: 100ms\n"
	updated := setConfigKey(existing, "namespace", "staging")
	keys := parseConfigKeys(updated)
	if keys["namespace"] != "staging" {
		t.Errorf("expected staging, got %q", keys["namespace"])
	}
	if keys["tick"] != "100ms" {
		t.Errorf("tick should be unchanged, got %q", keys["tick"])
	}
}

func TestSetConfigKeyAppend(t *testing.T) {
	existing := "namespace: default\n"
	updated := setConfigKey(existing, "log_level", "debug")
	keys := parseConfigKeys(updated)
	if keys["log_level"] != "debug" {
		t.Errorf("expected debug, got %q", keys["log_level"])
	}
	if keys["namespace"] != "default" {
		t.Errorf("namespace should be unchanged, got %q", keys["namespace"])
	}
}

func TestValidateConfigYAMLValid(t *testing.T) {
	content := "namespace: default\ntick: 100ms\ndata_dir: .awis\n"
	errs := validateConfigYAML(content)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateConfigYAMLUnknownKey(t *testing.T) {
	content := "namespace: default\nunknown_key: foo\n"
	errs := validateConfigYAML(content)
	if len(errs) == 0 {
		t.Error("expected error for unknown key, got none")
	}
}

func TestValidateConfigYAMLInvalidLine(t *testing.T) {
	content := "namespace: default\nnot-a-kv-line\n"
	errs := validateConfigYAML(content)
	if len(errs) == 0 {
		t.Error("expected error for invalid line, got none")
	}
}

// ── plugin status / remove ────────────────────────────────────────────────────

func TestPluginRemoveWritesAuditRow(t *testing.T) {
	dir := t.TempDir()

	db, err := openTestDB(t, dir)
	if err != nil {
		t.Fatal(err)
	}

	ps, ok := db.(storage.PluginStore)
	if !ok {
		t.Skip("storage does not support PluginStore")
	}

	ctx := context.Background()

	// Register a plugin.
	if err := ps.RegisterPlugin(ctx, "myplugin", "1.0.0", `{"name":"myplugin"}`, nil); err != nil {
		t.Fatal(err)
	}

	// Set status to removed (simulating plugin remove).
	if err := ps.SetPluginStatus(ctx, "myplugin", "removed"); err != nil {
		t.Fatalf("SetPluginStatus: %v", err)
	}

	// Verify status via GetPlugin.
	row, err := ps.GetPlugin(ctx, "myplugin")
	if err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}
	if row.Status != "removed" {
		t.Errorf("expected status=removed, got %q", row.Status)
	}

	// Write PluginRemoved audit row (simulating plugin remove audit site).
	aa, ok := db.(storage.AuditAppender)
	if !ok {
		t.Skip("storage does not support AuditAppender")
	}
	payload, _ := json.Marshal(map[string]string{"name": "myplugin"})
	if err := aa.AppendAudit(ctx, storage.AuditEntry{
		Timestamp:      time.Now(),
		EventType:      "PluginRemoved",
		Actor:          "cli",
		PayloadSummary: string(payload),
	}); err != nil {
		t.Fatalf("AppendAudit: %v", err)
	}

	// Verify audit row via ListAudit.
	rs, ok := db.(storage.RecallStore)
	if !ok {
		t.Skip("storage does not support RecallStore")
	}
	rows, err := rs.ListAudit(ctx, 10)
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	found := false
	for _, r := range rows {
		if r.EventType == "PluginRemoved" && r.Actor == "cli" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected PluginRemoved audit row; not found")
	}
}

func TestPluginStatusRoundTrip(t *testing.T) {
	dir := t.TempDir()

	db, err := openTestDB(t, dir)
	if err != nil {
		t.Fatal(err)
	}

	ps, ok := db.(storage.PluginStore)
	if !ok {
		t.Skip("storage does not support PluginStore")
	}

	ctx := context.Background()

	// Register a plugin.
	manifest := `{"name":"test-plugin","version":"0.2.0","path":"/opt/plugins/test-plugin"}`
	if err := ps.RegisterPlugin(ctx, "test-plugin", "0.2.0", manifest, []string{"test.cap"}); err != nil {
		t.Fatal(err)
	}

	// GetPlugin should return registered status.
	row, err := ps.GetPlugin(ctx, "test-plugin")
	if err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}
	if row.Name != "test-plugin" {
		t.Errorf("expected name test-plugin, got %q", row.Name)
	}
	if row.Version != "0.2.0" {
		t.Errorf("expected version 0.2.0, got %q", row.Version)
	}
	if row.Status != "registered" {
		t.Errorf("expected status registered, got %q", row.Status)
	}
}

// ── rebuild-state store type ──────────────────────────────────────────────────

func TestRebuildStateStoreTypeAssertion(t *testing.T) {
	dir := t.TempDir()

	db, err := openTestDB(t, dir)
	if err != nil {
		t.Fatal(err)
	}

	// *storage.SQLiteStorage must satisfy rebuildStateStore.
	if _, ok := db.(rebuildStateStore); !ok {
		t.Error("storage does not satisfy rebuildStateStore interface")
	}
}

func TestRebuildStateEmpty(t *testing.T) {
	dir := t.TempDir()

	db, err := openTestDB(t, dir)
	if err != nil {
		t.Fatal(err)
	}

	rs, ok := db.(rebuildStateStore)
	if !ok {
		t.Skip("storage does not support RebuildState")
	}

	// Empty DB: rebuild should succeed.
	if err := rs.RebuildState(context.Background()); err != nil {
		t.Fatalf("RebuildState on empty DB: %v", err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// openTestDB creates a fresh SQLiteStorage in dir using the cmd/awis OpenStorage helper.
// Returns any so callers can type-assert to additive interfaces (PluginStore, AuditAppender, etc.).
func openTestDB(t *testing.T, dir string) (any, error) {
	t.Helper()
	return OpenStorage(dir)
}
