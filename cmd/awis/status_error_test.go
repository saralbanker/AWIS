package main

// status_error_test.go — regression tests for D-8: 'awis status' silently
// swallowed a ListInstances storage error ("if err != nil { continue }"),
// producing an incomplete report indistinguishable from a correct one, for
// both the human table and --json paths.
//
// The storage error is induced for real (not via a fault-injection hook in
// production code): the DB is initialized through the real CLI so it has the
// production schema, then the workflow_instances table is dropped with a raw
// *sql.DB connection — the same low-level-db-manipulation technique
// plugin_test.go's checkAuditRowExists uses — so ListInstances genuinely
// fails with "no such table" the next time status runs. This is the
// "test-only seam": a way to make the real production code observe a real
// storage error, without adding any test hook to the CLI itself.

import (
	"database/sql"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// corruptWorkflowInstancesTable drops the workflow_instances table from the
// SQLite db at dbPath, so any subsequent ListInstances call against it fails.
func corruptWorkflowInstancesTable(t *testing.T, dbPath string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("corruptWorkflowInstancesTable: open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`DROP TABLE workflow_instances`); err != nil {
		t.Fatalf("corruptWorkflowInstancesTable: drop table: %v", err)
	}
}

// TestStatusStorageErrorSurfacedHuman proves that a ListInstances failure is
// surfaced (operational error, exit 1) rather than silently skipped in the
// human 'awis status' path (D-8).
func TestStatusStorageErrorSurfacedHuman(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping system test under -short")
	}

	binDir := t.TempDir()
	bin := buildBinary(t, binDir)

	dataDir := t.TempDir()

	// Materialize the real schema by running status once against an empty,
	// healthy data dir (exit 0 expected).
	if out, err := exec.Command(bin, "--data-dir="+dataDir, "status").CombinedOutput(); err != nil {
		t.Fatalf("initial healthy 'status' failed: %v\noutput: %s", err, out)
	}

	corruptWorkflowInstancesTable(t, filepath.Join(dataDir, "runtime.db"))

	cmd := exec.Command(bin, "--data-dir="+dataDir, "status")
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()

	if runErr == nil {
		t.Fatalf("status with a broken storage bucket exited 0, want a nonzero (operational error) exit\nstdout: %s\nstderr: %s",
			stdout.String(), stderr.String())
	}
	if code := cmd.ProcessState.ExitCode(); code != 1 {
		t.Errorf("exit code = %d, want 1 (operational error per TDS-07 §2)\nstdout: %s\nstderr: %s",
			code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Errorf("stdout = %q, want empty — a storage failure must not render a partial/incomplete table", stdout.String())
	}
	if !strings.Contains(stderr.String(), "status:") || !strings.Contains(stderr.String(), "cannot list") {
		t.Errorf("stderr = %q, want it to identify the failing status command and reason", stderr.String())
	}
}

// TestStatusStorageErrorSurfacedJSON proves that a ListInstances failure is
// surfaced (operational error, exit 1, no output) rather than silently
// skipped in the 'awis status --json' path (D-8). Critically, no JSON
// document is emitted at all: a machine consumer must not be able to parse a
// well-formed-but-silently-incomplete document as if it were a complete one.
func TestStatusStorageErrorSurfacedJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping system test under -short")
	}

	binDir := t.TempDir()
	bin := buildBinary(t, binDir)

	dataDir := t.TempDir()

	if out, err := exec.Command(bin, "--data-dir="+dataDir, "--json", "status").CombinedOutput(); err != nil {
		t.Fatalf("initial healthy 'status --json' failed: %v\noutput: %s", err, out)
	}

	corruptWorkflowInstancesTable(t, filepath.Join(dataDir, "runtime.db"))

	cmd := exec.Command(bin, "--data-dir="+dataDir, "--json", "status")
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()

	if runErr == nil {
		t.Fatalf("status --json with a broken storage bucket exited 0, want a nonzero (operational error) exit\nstdout: %s\nstderr: %s",
			stdout.String(), stderr.String())
	}
	if code := cmd.ProcessState.ExitCode(); code != 1 {
		t.Errorf("exit code = %d, want 1 (operational error per TDS-07 §2)\nstdout: %s\nstderr: %s",
			code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Errorf("stdout = %q, want empty — a --json consumer must never receive a well-formed but "+
			"silently incomplete document (D-8); it gets no document at all plus a nonzero exit instead",
			stdout.String())
	}
	if !strings.Contains(stderr.String(), "status:") || !strings.Contains(stderr.String(), "cannot list") {
		t.Errorf("stderr = %q, want it to identify the failing status command and reason", stderr.String())
	}
}

// TestStatusHealthySuccessPathUnchanged proves that when storage is healthy,
// both 'status' and 'status --json' still succeed with exit 0 and produce
// output — i.e. the D-8 fix only activates on the previously-swallowed error
// branch and does not alter the success path.
func TestStatusHealthySuccessPathUnchanged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping system test under -short")
	}

	binDir := t.TempDir()
	bin := buildBinary(t, binDir)

	dataDir := t.TempDir()

	humanOut, err := exec.Command(bin, "--data-dir="+dataDir, "status").CombinedOutput()
	if err != nil {
		t.Fatalf("'status' on healthy storage failed: %v\noutput: %s", err, humanOut)
	}
	if !strings.Contains(string(humanOut), "AWIS status") || !strings.Contains(string(humanOut), "ACTIVE") {
		t.Errorf("healthy human status output missing expected sections: %s", humanOut)
	}

	jsonOut, err := exec.Command(bin, "--data-dir="+dataDir, "--json", "status").CombinedOutput()
	if err != nil {
		t.Fatalf("'status --json' on healthy storage failed: %v\noutput: %s", err, jsonOut)
	}
	if !strings.Contains(string(jsonOut), `"active"`) || !strings.Contains(string(jsonOut), `"recent"`) {
		t.Errorf("healthy JSON status output missing expected fields: %s", jsonOut)
	}
}
