package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

// TestPluginStore_RegisterGetRoundTrip registers a plugin and verifies Get/List.
func TestPluginStore_RegisterGetRoundTrip(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	ctx := context.Background()

	manifest := `{"name":"test-plugin","version":"1.0.0"}`
	if err := s.RegisterPlugin(ctx, "test-plugin", "1.0.0", manifest, []string{"cap.a", "cap.b"}); err != nil {
		t.Fatalf("RegisterPlugin: %v", err)
	}

	p, err := s.GetPlugin(ctx, "test-plugin")
	if err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}
	if p.Name != "test-plugin" {
		t.Errorf("Name: got %q, want %q", p.Name, "test-plugin")
	}
	if p.Version != "1.0.0" {
		t.Errorf("Version: got %q, want %q", p.Version, "1.0.0")
	}
	if p.Manifest != manifest {
		t.Errorf("Manifest mismatch: got %q", p.Manifest)
	}
	if p.Status != "registered" {
		t.Errorf("Status: got %q, want %q", p.Status, "registered")
	}
	if p.PluginID != "test-plugin" {
		t.Errorf("PluginID: got %q, want %q", p.PluginID, "test-plugin")
	}
	if p.RegisteredAt.IsZero() {
		t.Error("RegisteredAt is zero")
	}
}

// TestPluginStore_ListPlugins verifies listing returns all registered plugins.
func TestPluginStore_ListPlugins(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	ctx := context.Background()

	if err := s.RegisterPlugin(ctx, "alpha-plugin", "1.0.0", `{}`, []string{"cap.alpha"}); err != nil {
		t.Fatalf("register alpha-plugin: %v", err)
	}
	if err := s.RegisterPlugin(ctx, "beta-plugin", "2.0.0", `{}`, []string{"cap.beta"}); err != nil {
		t.Fatalf("register beta-plugin: %v", err)
	}

	list, err := s.ListPlugins(ctx)
	if err != nil {
		t.Fatalf("ListPlugins: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListPlugins: want 2 plugins, got %d", len(list))
	}
	// Ordered by name: alpha comes before beta.
	if list[0].Name != "alpha-plugin" {
		t.Errorf("list[0].Name: got %q, want %q", list[0].Name, "alpha-plugin")
	}
	if list[1].Name != "beta-plugin" {
		t.Errorf("list[1].Name: got %q, want %q", list[1].Name, "beta-plugin")
	}
}

// TestPluginStore_ReregisterUpsertsAndReplacesCapabilities verifies that
// re-registering the same plugin name updates rows and replaces capabilities.
func TestPluginStore_ReregisterUpsertsAndReplacesCapabilities(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	ctx := context.Background()

	// First registration: two capabilities.
	if err := s.RegisterPlugin(ctx, "my-plugin", "1.0.0", `{"v":1}`, []string{"old.cap.1", "old.cap.2"}); err != nil {
		t.Fatalf("first RegisterPlugin: %v", err)
	}

	// Re-register with new version and different capabilities.
	if err := s.RegisterPlugin(ctx, "my-plugin", "1.1.0", `{"v":2}`, []string{"new.cap.x"}); err != nil {
		t.Fatalf("second RegisterPlugin: %v", err)
	}

	p, err := s.GetPlugin(ctx, "my-plugin")
	if err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}
	if p.Version != "1.1.0" {
		t.Errorf("Version after re-register: got %q, want %q", p.Version, "1.1.0")
	}
	if p.Status != "registered" {
		t.Errorf("Status after re-register: got %q, want %q", p.Status, "registered")
	}

	// Old capabilities must be gone; new one must exist.
	_, err = s.LookupCapability(ctx, "old.cap.1")
	if !errors.Is(err, ErrCapabilityNotFound) {
		t.Errorf("old.cap.1 should be gone after re-register, got: %v", err)
	}
	owner, err := s.LookupCapability(ctx, "new.cap.x")
	if err != nil {
		t.Fatalf("LookupCapability new.cap.x: %v", err)
	}
	if owner != "my-plugin" {
		t.Errorf("LookupCapability owner: got %q, want %q", owner, "my-plugin")
	}

	// Only one plugin row should exist.
	list, err := s.ListPlugins(ctx)
	if err != nil {
		t.Fatalf("ListPlugins: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 plugin after re-register, got %d", len(list))
	}
}

// TestPluginStore_SetPluginStatus verifies status transitions.
func TestPluginStore_SetPluginStatus(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	ctx := context.Background()

	if err := s.RegisterPlugin(ctx, "status-plugin", "1.0.0", `{}`, []string{"cap.s"}); err != nil {
		t.Fatalf("RegisterPlugin: %v", err)
	}

	for _, status := range []string{"active", "failed", "registered"} {
		if err := s.SetPluginStatus(ctx, "status-plugin", status); err != nil {
			t.Fatalf("SetPluginStatus(%q): %v", status, err)
		}
		p, err := s.GetPlugin(ctx, "status-plugin")
		if err != nil {
			t.Fatalf("GetPlugin: %v", err)
		}
		if p.Status != status {
			t.Errorf("after SetPluginStatus(%q): got %q", status, p.Status)
		}
	}
}

// TestPluginStore_SetPluginStatus_NotFound verifies ErrPluginNotFound on
// updating a plugin that does not exist.
func TestPluginStore_SetPluginStatus_NotFound(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	ctx := context.Background()

	err := s.SetPluginStatus(ctx, "nonexistent", "active")
	if !errors.Is(err, ErrPluginNotFound) {
		t.Errorf("want ErrPluginNotFound, got: %v", err)
	}
}

// TestPluginStore_LookupCapabilityHitMiss verifies capability lookup success
// and the miss path (ErrCapabilityNotFound).
func TestPluginStore_LookupCapabilityHitMiss(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	ctx := context.Background()

	if err := s.RegisterPlugin(ctx, "cap-plugin", "1.0.0", `{}`, []string{"git.context.assemble", "git.diff.fetch"}); err != nil {
		t.Fatalf("RegisterPlugin: %v", err)
	}

	// Hit.
	owner, err := s.LookupCapability(ctx, "git.context.assemble")
	if err != nil {
		t.Fatalf("LookupCapability hit: %v", err)
	}
	if owner != "cap-plugin" {
		t.Errorf("LookupCapability owner: got %q, want %q", owner, "cap-plugin")
	}

	// Second capability hit.
	owner2, err := s.LookupCapability(ctx, "git.diff.fetch")
	if err != nil {
		t.Fatalf("LookupCapability second hit: %v", err)
	}
	if owner2 != "cap-plugin" {
		t.Errorf("LookupCapability owner2: got %q, want %q", owner2, "cap-plugin")
	}

	// Miss.
	_, err = s.LookupCapability(ctx, "does.not.exist")
	if !errors.Is(err, ErrCapabilityNotFound) {
		t.Errorf("miss: want ErrCapabilityNotFound, got: %v", err)
	}
}

// TestPluginStore_GetPlugin_NotFound verifies ErrPluginNotFound on missing plugin.
func TestPluginStore_GetPlugin_NotFound(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	ctx := context.Background()

	_, err := s.GetPlugin(ctx, "ghost-plugin")
	if !errors.Is(err, ErrPluginNotFound) {
		t.Errorf("want ErrPluginNotFound, got: %v", err)
	}
}

// TestPluginStore_AuditRowPresentAfterRegister verifies that RegisterPlugin
// writes a PluginRegistered audit row (F-4 write site; TRACEABILITY T6).
// The audit row is queried directly via the audit_log table.
func TestPluginStore_AuditRowPresentAfterRegister(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	ctx := context.Background()

	if err := s.RegisterPlugin(ctx, "audit-plugin", "3.0.0", `{"name":"audit-plugin"}`, []string{"cap.audit"}); err != nil {
		t.Fatalf("RegisterPlugin: %v", err)
	}

	// Query audit_log directly.
	var eventType, payloadSummary string
	err := db.db.QueryRowContext(ctx,
		`SELECT event_type, payload_summary FROM audit_log WHERE event_type = 'PluginRegistered' AND actor = 'system'`,
	).Scan(&eventType, &payloadSummary)
	if err != nil {
		t.Fatalf("audit_log query: %v", err)
	}
	if eventType != "PluginRegistered" {
		t.Errorf("audit event_type: got %q, want %q", eventType, "PluginRegistered")
	}
	// payload_summary is JSON containing name and version.
	if payloadSummary == "" {
		t.Error("audit payload_summary is empty")
	}
}

// TestPluginStore_AuditRowOnReregister verifies that re-registering a plugin
// writes a second PluginRegistered audit row (re-register also audits; F-4).
func TestPluginStore_AuditRowOnReregister(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	ctx := context.Background()

	if err := s.RegisterPlugin(ctx, "rereg-plugin", "1.0.0", `{}`, []string{"c1"}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := s.RegisterPlugin(ctx, "rereg-plugin", "1.1.0", `{}`, []string{"c2"}); err != nil {
		t.Fatalf("second register: %v", err)
	}

	var count int
	if err := db.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE event_type = 'PluginRegistered'`,
	).Scan(&count); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 2 {
		t.Errorf("want 2 PluginRegistered audit rows, got %d", count)
	}
}

// TestPluginStore_Migration0005UpgradePath verifies that migration 0005 can
// be applied to a DB that was created at version 4 (pre-0005 state), and that
// the plugins + plugin_capabilities tables exist after upgrade.
func TestPluginStore_Migration0005UpgradePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "upgrade.db")

	// Build a version-4 fixture: open a raw sqlite DB, apply migrations 0001–0004
	// by hand, record them in schema_version, then close.
	rawDB, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	closeRaw := func() {
		if cerr := rawDB.Close(); cerr != nil {
			t.Errorf("rawDB.Close: %v", cerr)
		}
	}
	if err := applyPragmas(rawDB); err != nil {
		closeRaw()
		t.Fatalf("pragmas: %v", err)
	}
	if err := ensureSchemaVersionTable(rawDB); err != nil {
		closeRaw()
		t.Fatalf("ensureSchemaVersionTable: %v", err)
	}

	// Apply migrations 0001–0004.
	for _, name := range []string{
		"0001_core_execution.sql",
		"0002_domain_events.sql",
		"0003_signals.sql",
		"0004_audit.sql",
	} {
		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			closeRaw()
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := rawDB.Exec(string(data)); err != nil {
			closeRaw()
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	// Record version 4 in schema_version.
	for v := 1; v <= 4; v++ {
		if _, err := rawDB.Exec(
			`INSERT INTO schema_version (version, applied_at) VALUES (?, '2026-01-01T00:00:00Z')`, v,
		); err != nil {
			closeRaw()
			t.Fatalf("record version %d: %v", v, err)
		}
	}
	// Verify fixture is at version 4.
	var fixtureVer int
	if err := rawDB.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&fixtureVer); err != nil {
		closeRaw()
		t.Fatalf("currentVersion fixture: %v", err)
	}
	if fixtureVer != 4 {
		closeRaw()
		t.Fatalf("fixture: want version 4, got %d", fixtureVer)
	}
	// plugins table must NOT yet exist in the fixture.
	var tbl string
	if err := rawDB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='plugins'`).Scan(&tbl); err == nil {
		closeRaw()
		t.Fatal("plugins table must not exist before 0005")
	}
	closeRaw()

	// Now open via Open() — must apply migration 0005.
	db, err := Open(path, nil)
	if err != nil {
		t.Fatalf("Open after v4 fixture: %v", err)
	}
	defer func() { _ = db.Close() }()

	v, err := currentVersion(db.db)
	if err != nil {
		t.Fatalf("currentVersion after upgrade: %v", err)
	}
	if v != 8 {
		t.Fatalf("want schema_version 8 after upgrade, got %d", v)
	}

	// Both tables must exist.
	for _, tblName := range []string{"plugins", "plugin_capabilities"} {
		var name string
		if err := db.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tblName,
		).Scan(&name); err != nil {
			t.Errorf("table %q not found after 0005: %v", tblName, err)
		}
	}
}

// TestPluginStore_TypeAssertion verifies that *SQLiteStorage satisfies PluginStore.
func TestPluginStore_TypeAssertion(t *testing.T) {
	db := openTestDB(t)
	s := NewSQLiteStorage(db, nil)
	var _ PluginStore = s // compile-time + runtime check
	_ = s
}
