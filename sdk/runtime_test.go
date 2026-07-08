// Unit tests for the sdk Runtime constructor, registration, and WorkflowBuilder
// (M08-C1; AWIS DoD §24.3).
package sdk

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
	_ "modernc.org/sqlite" // register "sqlite" driver for direct audit_log queries
)

// openTestStorage opens a temp-file SQLite storage suitable for sdk tests.
func openTestStorage(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	path := filepath.Join(t.TempDir(), "awis.db")
	db, err := storage.Open(path, time.Now)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return storage.NewSQLiteStorage(db, time.Now)
}

// ── NewRuntime validation ─────────────────────────────────────────────────────

// TestNewRuntime_NilStorage verifies that Config.Storage nil returns an error.
func TestNewRuntime_NilStorage(t *testing.T) {
	_, err := NewRuntime(Config{
		Namespace: "test",
		Storage:   nil,
	})
	if err == nil {
		t.Fatal("expected error for nil Storage, got nil")
	}
}

// TestNewRuntime_EmptyNamespace verifies that Config.Namespace empty returns an error.
func TestNewRuntime_EmptyNamespace(t *testing.T) {
	s := openTestStorage(t)
	_, err := NewRuntime(Config{
		Namespace: "",
		Storage:   s,
	})
	if err == nil {
		t.Fatal("expected error for empty Namespace, got nil")
	}
}

// ── RegisterHandler ───────────────────────────────────────────────────────────

// TestRegisterHandler_Nil verifies that a nil handler returns a *RegistrationError
// whose Error() string matches the PRD §18 format.
func TestRegisterHandler_Nil(t *testing.T) {
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	err = rt.RegisterHandler(nil)
	if err == nil {
		t.Fatal("expected error for nil handler, got nil")
	}

	// Verify concrete type.
	regErr, ok := err.(*RegistrationError)
	if !ok {
		t.Fatalf("expected *RegistrationError, got %T", err)
	}

	// PRD §18 format: header line + detail + Hint.
	errStr := regErr.Error()
	if !strings.HasPrefix(errStr, "awis: registration failed for ") {
		t.Errorf("error string does not start with expected prefix: %q", errStr)
	}
	if !strings.Contains(errStr, "Hint:") {
		t.Errorf("error string missing Hint: %q", errStr)
	}
}

// ── RegisterWorkflow ──────────────────────────────────────────────────────────

// TestRegisterWorkflow_Nil verifies that a nil definition returns an error.
func TestRegisterWorkflow_Nil(t *testing.T) {
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if err := rt.RegisterWorkflow(nil); err == nil {
		t.Fatal("expected error for nil WorkflowDefinition, got nil")
	}
}

// TestRegisterWorkflow_InvalidSemver verifies that "1.0" is rejected and "1.0.0"
// is accepted by semver validation in RegisterWorkflow (card ACCEPTANCE).
func TestRegisterWorkflow_InvalidSemver(t *testing.T) {
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	// "1.0" — invalid semver: must be rejected.
	badDef := &core.WorkflowDefinition{
		ID:          "myflow",
		Version:     core.SemVer("1.0"),
		Namespace:   "test",
		Transitions: []core.Transition{},
		Steps:       []core.Step{},
		FinalSteps:  []string{},
		Triggers:    []core.Trigger{},
		Metadata:    map[string]any{},
	}
	if err := rt.RegisterWorkflow(badDef); err == nil {
		t.Error("expected error for version \"1.0\", got nil")
	}

	// "1.0.0" — valid semver: must succeed.
	goodDef := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "myflow",
		Version:       core.SemVer("1.0.0"),
		Namespace:     "test",
		Transitions:   []core.Transition{},
		Steps:         []core.Step{},
		FinalSteps:    []string{},
		Triggers:      []core.Trigger{},
		Metadata:      map[string]any{},
	}
	if err := rt.RegisterWorkflow(goodDef); err != nil {
		t.Errorf("expected no error for version \"1.0.0\", got: %v", err)
	}
}

// TestRegisterWorkflow_Duplicate verifies that registering the same (id, version)
// pair twice returns an error on the second call.
func TestRegisterWorkflow_Duplicate(t *testing.T) {
	s := openTestStorage(t)
	rt, err := NewRuntime(Config{Namespace: "test", Storage: s})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "dupflow",
		Version:       core.SemVer("1.0.0"),
		Namespace:     "test",
		Transitions:   []core.Transition{},
		Steps:         []core.Step{},
		FinalSteps:    []string{},
		Triggers:      []core.Trigger{},
		Metadata:      map[string]any{},
	}

	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("first RegisterWorkflow: %v", err)
	}
	if err := rt.RegisterWorkflow(def); err == nil {
		t.Error("expected error on duplicate registration, got nil")
	}
}

// TestRegisterWorkflow_Audit verifies that a successful RegisterWorkflow call
// writes a WorkflowRegistered audit entry to the SQLite audit_log table (F5).
// Uses real SQLiteStorage backed by a temp file and queries the table directly.
func TestRegisterWorkflow_Audit(t *testing.T) {
	ctx := context.Background()
	// Open a named temp file so a second SQL connection can read it.
	dbPath := filepath.Join(t.TempDir(), "audit_test.db")

	db, err := storage.Open(dbPath, time.Now)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s := storage.NewSQLiteStorage(db, time.Now)

	rt, err := NewRuntime(Config{
		Namespace: "test",
		Storage:   s,
		WorkerID:  "test-worker",
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "auditflow",
		Version:       core.SemVer("1.0.0"),
		Namespace:     "test",
		Transitions:   []core.Transition{},
		Steps:         []core.Step{},
		FinalSteps:    []string{},
		Triggers:      []core.Trigger{},
		Metadata:      map[string]any{},
	}

	if err := rt.RegisterWorkflow(def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}

	// Open a second connection to the same SQLite file and query audit_log directly.
	verifyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open verify: %v", err)
	}
	t.Cleanup(func() { _ = verifyDB.Close() })

	var count int
	if err := verifyDB.QueryRowContext(ctx,
		`SELECT count(*) FROM audit_log WHERE event_type='WorkflowRegistered'`,
	).Scan(&count); err != nil {
		t.Fatalf("query audit_log: %v", err)
	}
	if count != 1 {
		t.Errorf("audit_log WorkflowRegistered count = %d, want 1", count)
	}
}

// ── WorkflowBuilder ───────────────────────────────────────────────────────────

// TestWorkflowBuilder_Build verifies that:
//   - invalid semver versions ("1.0", "not-a-version", "") return errors from Build()
//   - valid semver versions ("1.0.0", "2.1.3-alpha") return a non-nil WorkflowDefinition
func TestWorkflowBuilder_Build(t *testing.T) {
	invalidVersions := []string{"1.0", "not-a-version", ""}
	for _, v := range invalidVersions {
		b := NewWorkflowBuilder("myflow", v)
		def, err := b.Build()
		if err == nil {
			t.Errorf("expected error for version %q, got nil (def=%v)", v, def)
		}
	}

	validVersions := []string{"1.0.0", "2.1.3-alpha"}
	for _, v := range validVersions {
		b := NewWorkflowBuilder("myflow", v)
		def, err := b.Build()
		if err != nil {
			t.Errorf("expected no error for version %q, got: %v", v, err)
		}
		if def == nil {
			t.Errorf("expected non-nil WorkflowDefinition for version %q", v)
		}
	}
}
