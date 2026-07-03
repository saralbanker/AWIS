package storage_test

import (
	"bufio"
	"context"
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

// TestHelperProcess is the subprocess entry point for the crash-durability test
// (standard Go helper-process idiom). It is invoked via os.Args[0] re-exec with
// the env var GO_TEST_HELPER_PROCESS=1 set.
//
// It opens the DB at path given by CRASH_DB_PATH, then appends events
// sequentially, printing each committed sequence number to stdout.
// It runs forever until the parent SIGKILLs it.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" {
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
	iid := core.InstanceID("crash-instance-1")
	ctx := context.Background()

	for seq := 1; ; seq++ {
		e := core.ExecutionEvent{
			EventID:       fmt.Sprintf("crash-evt-%d", seq),
			InstanceID:    iid,
			Namespace:     "crash-ns",
			EventType:     core.EventTypeWorkflowStarted,
			StepID:        "",
			Payload:       json.RawMessage(`{"inputs":{}}`),
			EmittedAt:     time.Now().UTC(),
			SequenceNum:   seq,
			SchemaVersion: 1,
		}
		if err := s.AppendEvent(ctx, e); err != nil {
			fmt.Fprintf(os.Stderr, "AppendEvent seq=%d: %v\n", seq, err)
			os.Exit(1)
		}
		// Print committed seq to stdout — parent reads these.
		fmt.Println(seq)
	}
}

// TestCrashDurability (NFR-R-02): spawns a child process that appends events
// and prints committed sequence numbers, then SIGKILLs it mid-stream and
// verifies every reported-committed event is present and intact in the DB.
func TestCrashDurability(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip crash durability test")
	}

	dbPath := filepath.Join(t.TempDir(), "crash.db")

	// Pre-create the database so the child can open it without running migrations
	// concurrently with the parent.
	initDB, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	_ = initDB.Close()

	// Launch child via os.Args[0] re-exec.
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "-test.v")
	cmd.Env = append(os.Environ(),
		"GO_TEST_HELPER_PROCESS=1",
		"CRASH_DB_PATH="+dbPath,
	)

	// Capture stdout (committed seq numbers) via pipe.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	// Collect committed seqs via a channel so the goroutine and main goroutine
	// never race on the slice.
	const minCommitted = 20
	seqCh := make(chan int, 256)

	scanner := bufio.NewScanner(stdout)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			n, err := strconv.Atoi(line)
			if err == nil {
				seqCh <- n
			}
		}
		close(seqCh)
	}()

	// Drain seqCh into committed slice until we have minCommitted or deadline.
	committed := make([]int, 0, 128)
	deadline := time.After(30 * time.Second)
collectLoop:
	for len(committed) < minCommitted {
		select {
		case n, ok := <-seqCh:
			if !ok {
				break collectLoop
			}
			committed = append(committed, n)
		case <-deadline:
			t.Fatalf("timeout: only collected %d committed seqs", len(committed))
		}
	}

	// SIGKILL the child mid-stream.
	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("SIGKILL: %v", err)
	}
	_ = cmd.Wait() // reap

	// Drain any remaining seqs that arrived before the pipe closed.
	for n := range seqCh {
		committed = append(committed, n)
	}
	<-done

	if len(committed) == 0 {
		t.Fatal("no committed seqs collected before SIGKILL")
	}
	t.Logf("collected %d committed seqs before SIGKILL; max seq = %d", len(committed), committed[len(committed)-1])

	// Reopen DB and verify every reported-committed event is present and intact.
	db, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer func() { _ = db.Close() }()

	s := storage.NewSQLiteStorage(db, nil)
	events, err := s.ReadEvents(context.Background(), "crash-instance-1", 1)
	if err != nil {
		t.Fatalf("ReadEvents after crash: %v", err)
	}

	// Build a set of sequence numbers present in the DB.
	present := make(map[int]bool, len(events))
	for _, e := range events {
		present[e.SequenceNum] = true
	}

	// Every reported-committed seq must be present.
	for _, seq := range committed {
		if !present[seq] {
			t.Errorf("committed seq %d not found in DB after crash", seq)
		}
	}

	// Also verify the ones present are intact (namespace + event_id correct).
	for _, e := range events {
		if e.Namespace != "crash-ns" {
			t.Errorf("seq %d: namespace corrupted: %q", e.SequenceNum, e.Namespace)
		}
		wantID := fmt.Sprintf("crash-evt-%d", e.SequenceNum)
		if e.EventID != wantID {
			t.Errorf("seq %d: event_id corrupted: want %q got %q", e.SequenceNum, wantID, e.EventID)
		}
	}
}

// TestAppend100K appends 100_000 events in sequence and records wall time
// (informational; binding NFR-P-05/06 targets are at M03/M14).
func TestAppend100K(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip 100K append test")
	}
	if raceEnabled {
		t.Skip("-race: 100K test skipped under race detector (too slow)")
	}

	dbPath := filepath.Join(t.TempDir(), "perf100k.db")
	db, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	s := storage.NewSQLiteStorage(db, nil)
	iid := core.InstanceID("perf-instance-1")
	ctx := context.Background()
	const n = 100_000

	start := time.Now()
	for i := 1; i <= n; i++ {
		e := core.ExecutionEvent{
			EventID:       fmt.Sprintf("perf-evt-%d", i),
			InstanceID:    iid,
			Namespace:     "perf-ns",
			EventType:     core.EventTypeWorkflowStarted,
			StepID:        "",
			Payload:       json.RawMessage(`{"inputs":{}}`),
			EmittedAt:     time.Now().UTC(),
			SequenceNum:   i,
			SchemaVersion: 1,
		}
		if err := s.AppendEvent(ctx, e); err != nil {
			t.Fatalf("AppendEvent seq=%d: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	t.Logf("TestAppend100K: appended %d events in %v (%.0f events/sec)", n, elapsed, float64(n)/elapsed.Seconds())
}
