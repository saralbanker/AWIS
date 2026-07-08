package storage_test

// Crash-injection tests for DeliverSignal (M07-C3r, IMP §20.M7).
//
// The B3 transaction (deliver.go) performs three writes inside ONE SQLite
// transaction: (1) UPDATE signal_inbox.delivered_at, (2) INSERT SignalReceived
// into execution_events, (3) UPDATE workflow_instances (status→running, version+1).
// Because they are atomic, a crash at any point before COMMIT leaves the DB in
// one of two consistent states:
//
//   pre-commit:  delivered_at IS NULL  ∧  NO SignalReceived event
//   post-commit: delivered_at IS NOT NULL ∧  SignalReceived event present
//
// These tests prove the invariant empirically using the subprocess+SIGKILL
// harness from crash_test.go. The subprocess prints synchronisation markers
// ("SETUP N" before calling DeliverSignal, "DELIVERED N" after commit) so the
// parent can kill at precisely the desired crash point.
//
// RebuildState convergence after a post-commit crash is also asserted:
// SignalReceived in the EventLog causes RebuildState to reconstruct the
// instance row with status=running, matching the committed projection.

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// TestHelperProcess_Signal is the subprocess entry point for the DeliverSignal
// crash test. It is activated only when GO_TEST_HELPER_PROCESS=1 AND
// CRASH_TEST_TYPE=signal are set (the parent sets both).
//
// For each iteration n=1,2,...:
//   - Inserts a waiting workflow_instances row "sig-crash-<n>"
//   - Inserts an undelivered signal_inbox entry
//   - Prints "SETUP <n>" — crash at this marker → pre-commit state
//   - Calls DeliverSignal (B3 tx: all three writes atomic)
//   - If committed: prints "DELIVERED <n>" — crash at this marker → post-commit
func TestHelperProcess_Signal(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" ||
		os.Getenv("CRASH_TEST_TYPE") != "signal" {
		return
	}

	dbPath := os.Getenv("CRASH_DB_PATH")
	if dbPath == "" {
		fmt.Fprintln(os.Stderr, "CRASH_DB_PATH not set")
		os.Exit(1)
	}

	db, err := storage.Open(dbPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	s := storage.NewSQLiteStorage(db, nil)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()

	for n := 1; ; n++ {
		iid := core.InstanceID(fmt.Sprintf("sig-crash-%d", n))
		sigID := fmt.Sprintf("sig-crash-signal-%d", n)

		// Insert instance: running (version 0→1), then waiting (version 1→2).
		// version=2 is the ExpectedVersion passed to DeliverSignal below.
		run := core.WorkflowInstance{
			InstanceID: iid, Namespace: "crash-ns",
			Status:       core.InstanceStatusRunning,
			CurrentSteps: []string{}, Variables: map[string]any{},
			StartedAt: now, UpdatedAt: now,
		}
		if err := s.UpsertInstance(ctx, run, 0); err != nil {
			fmt.Fprintf(os.Stderr, "UpsertInstance run n=%d: %v\n", n, err)
			os.Exit(1)
		}
		wait := run
		wait.Status = core.InstanceStatusWaiting
		if err := s.UpsertInstance(ctx, wait, 1); err != nil {
			fmt.Fprintf(os.Stderr, "UpsertInstance wait n=%d: %v\n", n, err)
			os.Exit(1)
		}

		// Insert undelivered inbox signal.
		if err := s.InsertSignal(ctx, storage.Signal{
			SignalID:   sigID,
			InstanceID: iid,
			SignalName: "crash-go",
			Payload:    map[string]any{"n": n},
			ReceivedAt: now,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "InsertSignal n=%d: %v\n", n, err)
			os.Exit(1)
		}

		// Crash-point 1: setup complete; DeliverSignal not yet called.
		// A kill here leaves delivered_at=NULL and no SignalReceived event (pre-commit).
		fmt.Printf("SETUP %d\n", n)

		payload, _ := json.Marshal(map[string]any{
			"signal_name": "crash-go",
			"payload":     map[string]any{"n": n},
		})
		outcome, err := s.DeliverSignal(ctx, storage.DeliverInput{
			InstanceID:      iid,
			SignalName:      "crash-go",
			EventPayload:    payload,
			ExpectedVersion: 2, // version after UpsertInstance(wait, 1)
			EventID:         fmt.Sprintf("ev-crash-%d", n),
			Now:             now,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "DeliverSignal n=%d: %v\n", n, err)
			os.Exit(1)
		}

		// Crash-point 2: all three B3 writes committed (or none, if outcome≠Delivered).
		// A kill here leaves delivered_at≠NULL and SignalReceived event present (post-commit).
		if outcome == storage.Delivered {
			fmt.Printf("DELIVERED %d\n", n)
		}
	}
}

// launchSignalCrashChild spawns the TestHelperProcess_Signal subprocess against
// dbPath and returns the command and a line scanner over its stdout.
func launchSignalCrashChild(t *testing.T, dbPath string) (*exec.Cmd, *bufio.Scanner) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess_Signal", "-test.v")
	cmd.Env = append(os.Environ(),
		"GO_TEST_HELPER_PROCESS=1",
		"CRASH_TEST_TYPE=signal",
		"CRASH_DB_PATH="+dbPath,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}
	return cmd, bufio.NewScanner(stdout)
}

// ── Invariant check helpers ──────────────────────────────────────────────────

// undeliveredCount returns the number of signal_inbox rows with delivered_at IS NULL.
func undeliveredCount(t *testing.T, rawDB *sql.DB) int {
	t.Helper()
	var n int
	row := rawDB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM signal_inbox WHERE delivered_at IS NULL`)
	if err := row.Scan(&n); err != nil {
		t.Fatalf("undeliveredCount query: %v", err)
	}
	return n
}

// signalReceivedCount returns the number of SignalReceived events in execution_events.
func signalReceivedCount(t *testing.T, rawDB *sql.DB) int {
	t.Helper()
	var n int
	row := rawDB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM execution_events WHERE event_type = 'SignalReceived'`)
	if err := row.Scan(&n); err != nil {
		t.Fatalf("signalReceivedCount query: %v", err)
	}
	return n
}

// deliveredCount returns the number of signal_inbox rows with delivered_at IS NOT NULL.
func deliveredCount(t *testing.T, rawDB *sql.DB) int {
	t.Helper()
	var n int
	row := rawDB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM signal_inbox WHERE delivered_at IS NOT NULL`)
	if err := row.Scan(&n); err != nil {
		t.Fatalf("deliveredCount query: %v", err)
	}
	return n
}

// instanceStatus returns the status of the workflow_instances row for iid, or
// "NOT_FOUND" when absent.
func instanceStatus(t *testing.T, rawDB *sql.DB, iid string) string {
	t.Helper()
	var status string
	err := rawDB.QueryRowContext(context.Background(),
		`SELECT status FROM workflow_instances WHERE instance_id = ?`, iid).Scan(&status)
	if err == sql.ErrNoRows {
		return "NOT_FOUND"
	}
	if err != nil {
		t.Fatalf("instanceStatus query: %v", err)
	}
	return status
}

// ── TestDeliverSignal_CrashPreCommit ────────────────────────────────────────

// TestDeliverSignal_CrashPreCommit kills the subprocess after "SETUP 1" —
// the crash occurs before or during the B3 transaction (before the commit
// can persist all three writes). After reopening the DB, every signal must
// have delivered_at=NULL and no SignalReceived event may exist.
//
// This simulates crash-injection between each of the three B3 write pairs:
// because the writes are atomic, all outcomes are equivalent (rollback).
func TestDeliverSignal_CrashPreCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip crash pre-commit test")
	}

	dbPath := filepath.Join(t.TempDir(), "crash_pre.db")
	initDB, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	_ = initDB.Close()

	cmd, scanner := launchSignalCrashChild(t, dbPath)
	scanCh := make(chan string, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			scanCh <- scanner.Text()
		}
		close(scanCh)
	}()

	// Wait for at least one SETUP line so the child is alive and past setup.
	gotSetup := false
	deadline := time.After(30 * time.Second)
waitSetup:
	for {
		select {
		case line, ok := <-scanCh:
			if !ok {
				break waitSetup
			}
			if strings.HasPrefix(line, "SETUP ") {
				gotSetup = true
				break waitSetup
			}
		case <-deadline:
			t.Fatalf("timeout waiting for SETUP marker")
		}
	}
	if !gotSetup {
		t.Fatal("child exited before printing SETUP marker")
	}

	// Kill before any DELIVERED is printed — crash in pre-commit territory.
	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("SIGKILL: %v", err)
	}
	_ = cmd.Wait()
	<-done

	// Reopen and verify invariant: zero signals delivered ⟺ zero SignalReceived events.
	db2, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer func() { _ = db2.Close() }()
	rawDB := db2.ExposedDB()

	undel := undeliveredCount(t, rawDB)
	srCount := signalReceivedCount(t, rawDB)
	del := deliveredCount(t, rawDB)

	if del != 0 {
		t.Errorf("pre-commit crash: %d signal(s) marked delivered, want 0", del)
	}
	if srCount != 0 {
		t.Errorf("pre-commit crash: %d SignalReceived event(s), want 0", srCount)
	}
	// Invariant: delivered_at IS NULL ⟺ no SignalReceived (both sides false here).
	t.Logf("pre-commit crash: %d undelivered signals, %d SignalReceived events (invariant holds)", undel, srCount)
}

// ── TestDeliverSignal_CrashPostCommit ───────────────────────────────────────

// TestDeliverSignal_CrashPostCommit kills the subprocess after it has printed
// at least minDelivered "DELIVERED" markers, proving that the crash occurs
// AFTER the B3 transaction commits. For every confirmed-delivered instance:
//
//   - signal_inbox.delivered_at IS NOT NULL
//   - exactly one SignalReceived event in execution_events
//   - after RebuildState, workflow_instances.status = running
//
// Invariant: delivered_at IS NOT NULL ⟺ SignalReceived ∈ EventLog (all true).
func TestDeliverSignal_CrashPostCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip crash post-commit test")
	}

	const minDelivered = 5

	dbPath := filepath.Join(t.TempDir(), "crash_post.db")
	initDB, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	_ = initDB.Close()

	cmd, scanner := launchSignalCrashChild(t, dbPath)
	scanCh := make(chan string, 64)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			scanCh <- scanner.Text()
		}
		close(scanCh)
	}()

	// Collect confirmed DELIVERED iterations.
	delivered := make([]int, 0, minDelivered)
	deadline := time.After(30 * time.Second)
collectLoop:
	for len(delivered) < minDelivered {
		select {
		case line, ok := <-scanCh:
			if !ok {
				break collectLoop
			}
			if strings.HasPrefix(line, "DELIVERED ") {
				n, err := strconv.Atoi(strings.TrimPrefix(line, "DELIVERED "))
				if err == nil {
					delivered = append(delivered, n)
				}
			}
		case <-deadline:
			t.Fatalf("timeout: only collected %d/%d DELIVERED markers", len(delivered), minDelivered)
		}
	}
	if len(delivered) < minDelivered {
		t.Fatalf("child exited with only %d DELIVERED markers", len(delivered))
	}

	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("SIGKILL: %v", err)
	}
	_ = cmd.Wait()
	<-done

	t.Logf("post-commit crash: %d confirmed deliveries before kill", len(delivered))

	// Reopen and verify invariant for each confirmed-delivered iteration.
	db2, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer func() { _ = db2.Close() }()

	s2 := storage.NewSQLiteStorage(db2, nil)
	rawDB := db2.ExposedDB()
	ctx := context.Background()

	for _, n := range delivered {
		iid := core.InstanceID(fmt.Sprintf("sig-crash-%d", n))

		// signal_inbox must have delivered_at ≠ NULL.
		undelivered, err := s2.ListUndeliveredSignals(ctx, iid)
		if err != nil {
			t.Errorf("n=%d ListUndeliveredSignals: %v", n, err)
			continue
		}
		if len(undelivered) != 0 {
			t.Errorf("n=%d: signal must be delivered (delivered_at≠NULL), got %d undelivered", n, len(undelivered))
		}

		// execution_events must have exactly one SignalReceived event.
		evs, err := s2.ReadEvents(ctx, iid, 0)
		if err != nil {
			t.Errorf("n=%d ReadEvents: %v", n, err)
			continue
		}
		srEvents := 0
		for _, ev := range evs {
			if ev.EventType == core.EventTypeSignalReceived {
				srEvents++
			}
		}
		if srEvents != 1 {
			t.Errorf("n=%d: want exactly 1 SignalReceived event, got %d", n, srEvents)
		}
		t.Logf("n=%d: delivered_at≠NULL ✓, SignalReceived event ✓ (invariant holds)", n)
	}

	// RebuildState convergence: for each delivered instance, the rebuilt row
	// must have status=running (SignalReceived → running in projectInstance).
	if err := s2.RebuildState(ctx); err != nil {
		t.Fatalf("RebuildState: %v", err)
	}
	for _, n := range delivered {
		iid := fmt.Sprintf("sig-crash-%d", n)
		got := instanceStatus(t, rawDB, iid)
		if got != "running" {
			t.Errorf("n=%d: after RebuildState status=%q, want running", n, got)
		}
	}
	t.Logf("post-commit crash: RebuildState converges to running for all %d delivered instances ✓", len(delivered))

	// Global invariant: deliveredCount == signalReceivedCount.
	del := deliveredCount(t, rawDB)
	sr := signalReceivedCount(t, rawDB)
	if del != sr {
		t.Errorf("global invariant violated: %d delivered_at≠NULL vs %d SignalReceived events", del, sr)
	} else {
		t.Logf("global invariant: %d delivered_at≠NULL = %d SignalReceived events ✓", del, sr)
	}
}
