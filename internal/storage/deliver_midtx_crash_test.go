package storage_test

// Mid-transaction crash injection for DeliverSignal (Loop A, backlog M18-B5).
//
// TestDeliverSignal_CrashPreCommit (deliver_crash_test.go) kills the child
// BEFORE DeliverSignal is called, so no existing test crashes INSIDE the
// Finalization B3 transaction:
//
//	guard UPDATE signal_inbox → INSERT SignalReceived → optimistic UPDATE
//	workflow_instances → COMMIT
//
// This file closes that gap without any production hook:
//
//   - The child process (and only the child) registers the application-defined
//     SQL function crash_point(stage) with modernc.org/sqlite BEFORE opening the
//     database. When stage == CRASH_AT the function prints "AT <stage>", flushes,
//     and SIGKILLs its own process.
//   - The parent installs three SQLite AFTER triggers that call crash_point at
//     the intra-transaction points. Triggers fire inside the statement that
//     performs the write, so the kill lands while the DeliverSignal transaction
//     is open, after the writes that precede the trigger's statement.
//
// Canonical coordinate: Finalization B3 — single tx per Finalization B3
// (guard → append → optimistic lock; SignalReceived in EventLog ⇔ delivered).
// A crash before COMMIT must leave NO trace of the transaction: delivered_at IS
// NULL, no SignalReceived event, and the instance still waiting at its original
// version. The invariant checkers below encode exactly that, and they return an
// error instead of failing the test so the sensitivity meta-test can assert that
// a non-atomic variant is reported as a violation.

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"modernc.org/sqlite"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

const (
	// midTxHelperType gates the child helper (GO_TEST_HELPER_PROCESS=1 plus
	// CRASH_TEST_TYPE=signal_midtx). Distinct from the "signal" helper in
	// deliver_crash_test.go so the two children never run each other's code.
	midTxHelperType = "signal_midtx"

	// midTxTimeout is a hang guard only. Crash points are deterministic (the
	// child kills itself at a named trigger); nothing here is timing-based.
	midTxTimeout = 60 * time.Second

	midTxInstance   = "midtx-instance"
	midTxSignalID   = "midtx-signal-1"
	midTxSignalName = "midtx-go"
	midTxEventID    = "ev-midtx-1"

	// midTxWaitVersion is the instance version after UpsertInstance(waiting, 1)
	// (insert → 1, then the waiting update → 2); it is the ExpectedVersion of
	// the B3 resume.
	midTxWaitVersion = 2
	// midTxResumedVersion is the version after the B3 resume (+1).
	midTxResumedVersion = 3
)

var midTxNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// midTxStages are the three intra-transaction crash points, in B3 order.
var midTxStages = []string{"after_guard", "after_append", "before_commit"}

// midTxTriggerInstall are the fault-injection triggers. Stage names are passed
// verbatim to crash_point. after_resume fires after write 3 (the optimistic
// resume), i.e. the last point before COMMIT, and reports stage "before_commit".
var midTxTriggerInstall = []string{
	`CREATE TRIGGER midtx_after_guard AFTER UPDATE OF delivered_at ON signal_inbox
		WHEN NEW.delivered_at IS NOT NULL AND OLD.delivered_at IS NULL
		BEGIN SELECT crash_point('after_guard'); END`,
	`CREATE TRIGGER midtx_after_append AFTER INSERT ON execution_events
		WHEN NEW.event_type='SignalReceived'
		BEGIN SELECT crash_point('after_append'); END`,
	`CREATE TRIGGER midtx_after_resume AFTER UPDATE ON workflow_instances
		WHEN OLD.status='waiting' AND NEW.status='running'
		BEGIN SELECT crash_point('before_commit'); END`,
}

var midTxTriggerDrop = []string{
	`DROP TRIGGER IF EXISTS midtx_after_guard`,
	`DROP TRIGGER IF EXISTS midtx_after_append`,
	`DROP TRIGGER IF EXISTS midtx_after_resume`,
}

// midTxDeliverInput builds the single B3 delivery under test. The child and the
// parent's recovery call use the same function, so "same inputs" is literal.
func midTxDeliverInput() (storage.DeliverInput, error) {
	payload, err := json.Marshal(map[string]any{
		"signal_name": midTxSignalName,
		"payload":     map[string]any{"n": 1},
	})
	if err != nil {
		return storage.DeliverInput{}, fmt.Errorf("marshal signal payload: %w", err)
	}
	return storage.DeliverInput{
		InstanceID:      core.InstanceID(midTxInstance),
		SignalName:      midTxSignalName,
		EventPayload:    payload,
		ExpectedVersion: midTxWaitVersion,
		EventID:         midTxEventID,
		Now:             midTxNow,
	}, nil
}

// midTxOutcomeName names a DeliverOutcome for the child's stdout protocol.
func midTxOutcomeName(o storage.DeliverOutcome) string {
	switch o {
	case storage.DeliverNoOp:
		return "NoOp"
	case storage.DeliverLockConflict:
		return "LockConflict"
	case storage.Delivered:
		return "Delivered"
	default:
		return fmt.Sprintf("Unknown(%d)", int(o))
	}
}

// midTxSelfKill SIGKILLs the current process. SIGKILL cannot be caught, so the
// trailing select never returns; if the kill somehow fails, the child exits
// non-zero and the parent's SIGKILL assertion fails.
func midTxSelfKill() {
	if err := syscall.Kill(os.Getpid(), syscall.SIGKILL); err != nil {
		fmt.Fprintf(os.Stderr, "self-kill failed: %v\n", err)
		os.Exit(3)
	}
	select {}
}

// midTxCrashPoint is the body of the child-only crash_point(stage) SQL function.
// It returns NULL (transaction continues) unless stage equals CRASH_AT, in which
// case it emits the stage marker and SIGKILLs the process mid-statement.
func midTxCrashPoint(crashAt string) func(*sqlite.FunctionContext, []driver.Value) (driver.Value, error) {
	return func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("crash_point: want 1 argument, got %d", len(args))
		}
		stage, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("crash_point: stage must be TEXT, got %T", args[0])
		}
		if crashAt == "" || stage != crashAt {
			return nil, nil
		}
		if _, err := fmt.Fprintf(os.Stdout, "AT %s\n", stage); err != nil {
			// Without the marker the parent cannot prove the crash point was
			// reached; exit non-zero so the SIGKILL assertion fails loudly.
			fmt.Fprintf(os.Stderr, "crash_point: write marker: %v\n", err)
			os.Exit(3)
		}
		// os.Stdout is unbuffered; Sync is requested by the protocol but fsync
		// on a pipe returns EINVAL, which is expected and harmless here.
		if err := os.Stdout.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
			return nil, fmt.Errorf("crash_point: sync stdout: %w", err)
		}
		midTxSelfKill()
		return nil, nil // unreachable
	}
}

// midTxNonAtomicDeliver is the deliberately NON-atomic sensitivity variant. It
// is test-only and never reachable from production. It issues the SAME three
// writes as deliver.go, with the same SQL text, but each as its own autocommit
// statement on the *sql.DB (no transaction, no rollback on lock conflict). It
// exists only to prove that the crash harness and the invariant checker can
// observe a violation of B3.
func midTxNonAtomicDeliver(ctx context.Context, db *sql.DB, in storage.DeliverInput) (storage.DeliverOutcome, error) {
	nowStr := in.Now.UTC().Format(time.RFC3339Nano)

	// write 1: guard (autocommit)
	res1, err := db.ExecContext(ctx, `
		UPDATE signal_inbox SET delivered_at = ?
		WHERE instance_id = ? AND signal_name = ? AND delivered_at IS NULL`,
		nowStr, string(in.InstanceID), in.SignalName,
	)
	if err != nil {
		return storage.DeliverNoOp, fmt.Errorf("nonatomic guard: %w", err)
	}
	n1, err := res1.RowsAffected()
	if err != nil {
		return storage.DeliverNoOp, fmt.Errorf("nonatomic guard rows: %w", err)
	}
	if n1 == 0 {
		return storage.DeliverNoOp, nil
	}

	var namespace string
	if err := db.QueryRowContext(ctx,
		`SELECT namespace FROM workflow_instances WHERE instance_id = ?`,
		string(in.InstanceID),
	).Scan(&namespace); err != nil {
		return storage.DeliverNoOp, fmt.Errorf("nonatomic read namespace: %w", err)
	}
	var maxSeq sql.NullInt64
	if err := db.QueryRowContext(ctx,
		`SELECT MAX(sequence_num) FROM execution_events WHERE instance_id = ?`,
		string(in.InstanceID),
	).Scan(&maxSeq); err != nil {
		return storage.DeliverNoOp, fmt.Errorf("nonatomic read max seq: %w", err)
	}
	seqNum := 1
	if maxSeq.Valid {
		seqNum = int(maxSeq.Int64) + 1
	}
	payload := []byte(in.EventPayload)
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	// write 2: append SignalReceived (autocommit)
	var stepID *string
	if _, err := db.ExecContext(ctx, `
		INSERT INTO execution_events
			(event_id, instance_id, namespace, event_type, step_id, payload,
			 emitted_at, sequence_num, schema_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.EventID,
		string(in.InstanceID),
		namespace,
		string(core.EventTypeSignalReceived),
		stepID,
		string(payload),
		nowStr,
		seqNum,
		1,
	); err != nil {
		return storage.DeliverNoOp, fmt.Errorf("nonatomic append event: %w", err)
	}

	// write 3: optimistic-lock resume (autocommit)
	res3, err := db.ExecContext(ctx, `
		UPDATE workflow_instances
		SET status = 'running', updated_at = ?, version = version + 1
		WHERE instance_id = ? AND status = 'waiting' AND version = ?`,
		nowStr, string(in.InstanceID), in.ExpectedVersion,
	)
	if err != nil {
		return storage.DeliverNoOp, fmt.Errorf("nonatomic resume: %w", err)
	}
	n3, err := res3.RowsAffected()
	if err != nil {
		return storage.DeliverNoOp, fmt.Errorf("nonatomic resume rows: %w", err)
	}
	if n3 == 0 {
		// Non-atomic bug under test: the guard flip and the appended event are
		// already durable and are NOT rolled back here.
		return storage.DeliverLockConflict, nil
	}
	return storage.Delivered, nil
}

// TestHelperProcess_SignalMidTx is the child entry point for the mid-transaction
// crash tests. It runs only when GO_TEST_HELPER_PROCESS=1 and
// CRASH_TEST_TYPE=signal_midtx are set; the parent re-execs the test binary with
// -test.run anchored to this name.
//
// Protocol (stdout): "SETUP" once the waiting instance and undelivered signal
// exist; "AT <stage>" only if crash_point fires (the child dies immediately);
// "DELIVERED <outcome>" only if DeliverSignal returned.
//
// CRASH_AT selects the crash stage ("after_guard", "after_append",
// "before_commit"). CRASH_AT=none runs the post-commit control: the child prints
// DELIVERED and then SIGKILLs itself. CRASH_DELIVER_IMPL=nonatomic swaps the real
// DeliverSignal for midTxNonAtomicDeliver (sensitivity variant only).
func TestHelperProcess_SignalMidTx(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" ||
		os.Getenv("CRASH_TEST_TYPE") != midTxHelperType {
		return
	}

	helperExit := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
		os.Exit(1)
	}

	dbPath := os.Getenv("CRASH_DB_PATH")
	if dbPath == "" {
		helperExit("CRASH_DB_PATH not set")
	}
	crashAt := os.Getenv("CRASH_AT")

	// Registration must precede storage.Open: the driver makes a registered
	// function visible only to connections opened afterwards. Child-only: the
	// parent process never registers crash_point.
	if err := sqlite.RegisterScalarFunction("crash_point", 1, midTxCrashPoint(crashAt)); err != nil {
		helperExit("register crash_point: %v", err)
	}

	db, err := storage.Open(dbPath, nil)
	if err != nil {
		helperExit("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	s := storage.NewSQLiteStorage(db, nil)
	ctx := context.Background()
	iid := core.InstanceID(midTxInstance)

	// Same version pattern as the existing helper: insert running (version 1),
	// then waiting (version 2 == midTxWaitVersion).
	run := core.WorkflowInstance{
		InstanceID: iid, Namespace: "crash-ns",
		Status:       core.InstanceStatusRunning,
		CurrentSteps: []string{}, Variables: map[string]any{},
		StartedAt: midTxNow, UpdatedAt: midTxNow,
	}
	if err := s.UpsertInstance(ctx, run, 0); err != nil {
		helperExit("UpsertInstance running: %v", err)
	}
	wait := run
	wait.Status = core.InstanceStatusWaiting
	if err := s.UpsertInstance(ctx, wait, 1); err != nil {
		helperExit("UpsertInstance waiting: %v", err)
	}
	if err := s.InsertSignal(ctx, storage.Signal{
		SignalID:   midTxSignalID,
		InstanceID: iid,
		SignalName: midTxSignalName,
		Payload:    map[string]any{"n": 1},
		ReceivedAt: midTxNow,
	}); err != nil {
		helperExit("InsertSignal: %v", err)
	}

	in, err := midTxDeliverInput()
	if err != nil {
		helperExit("%v", err)
	}

	fmt.Printf("SETUP\n")

	var outcome storage.DeliverOutcome
	if os.Getenv("CRASH_DELIVER_IMPL") == "nonatomic" {
		outcome, err = midTxNonAtomicDeliver(ctx, db.ExposedDB(), in)
	} else {
		outcome, err = s.DeliverSignal(ctx, in)
	}
	if err != nil {
		helperExit("DeliverSignal: %v", err)
	}
	fmt.Printf("DELIVERED %s\n", midTxOutcomeName(outcome))

	if crashAt == "none" {
		// Post-commit control: the parent asserts the state after a kill that
		// lands strictly after DeliverSignal returned.
		midTxSelfKill()
	}
}

// ── Parent-side harness ───────────────────────────────────────────────────────

// midTxPrepareDB creates a fully migrated database file in a fresh per-test
// directory and closes the handle, so the child is the only opener.
func midTxPrepareDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "midtx.db")
	db, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close init db: %v", err)
	}
	return dbPath
}

// midTxRawExec runs DDL against dbPath through a raw handle, then closes it.
func midTxRawExec(t *testing.T, dbPath string, stmts []string) {
	t.Helper()
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open raw: %v", err)
	}
	defer func() {
		if cerr := raw.Close(); cerr != nil {
			t.Errorf("close raw db: %v", cerr)
		}
	}()
	for _, stmt := range stmts {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
}

// midTxOpen reopens the database with the production Open (migrations are
// already applied, so this is a pure recovery reopen).
func midTxOpen(t *testing.T, dbPath string) *storage.DB {
	t.Helper()
	db, err := storage.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// midTxChildEnvBlocklist are variables the parent must never leak into the
// child, so each run is governed only by the env it sets explicitly.
var midTxChildEnvBlocklist = map[string]bool{
	"GO_TEST_HELPER_PROCESS": true,
	"CRASH_TEST_TYPE":        true,
	"CRASH_DB_PATH":          true,
	"CRASH_AT":               true,
	"CRASH_DELIVER_IMPL":     true,
}

// midTxRunChild launches the mid-transaction helper against dbPath and returns
// its stdout and the Wait error. The parent passes env via cmd.Env only; it
// never mutates its own process environment. The context is a hang guard: a
// timeout is a test failure, not a crash point.
func midTxRunChild(t *testing.T, dbPath string, extraEnv ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), midTxTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperProcess_SignalMidTx$")
	env := make([]string, 0, len(os.Environ())+len(extraEnv)+3)
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if !midTxChildEnvBlocklist[key] {
			env = append(env, kv)
		}
	}
	env = append(env,
		"GO_TEST_HELPER_PROCESS=1",
		"CRASH_TEST_TYPE="+midTxHelperType,
		"CRASH_DB_PATH="+dbPath,
	)
	cmd.Env = append(env, extraEnv...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("child hung past %v (hang guard); stdout=%q stderr=%q",
			midTxTimeout, stdout.String(), stderr.String())
	}
	return stdout.String(), runErr
}

// midTxRequireSIGKILL asserts the child terminated by SIGKILL (not a clean exit
// and not a self-inflicted non-zero status).
func midTxRequireSIGKILL(t *testing.T, runErr error) {
	t.Helper()
	var ee *exec.ExitError
	if !errors.As(runErr, &ee) {
		t.Fatalf("child did not die by signal (Run err=%v); want SIGKILL", runErr)
	}
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() || ws.Signal() != syscall.SIGKILL {
		t.Fatalf("child exit = %v; want signal SIGKILL", ee)
	}
}

// midTxHasLine reports whether stdout contains a line exactly equal to want.
func midTxHasLine(stdout, want string) bool {
	for _, line := range strings.Split(stdout, "\n") {
		if strings.TrimRight(line, "\r") == want {
			return true
		}
	}
	return false
}

// midTxHasLinePrefix reports whether any stdout line starts with prefix.
func midTxHasLinePrefix(stdout, prefix string) bool {
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// midTxState is the observable B3 state for the single delivery under test.
type midTxState struct {
	deliveredAt    sql.NullString
	signalReceived int
	status         string
	version        int
}

// midTxReadState reads the B3 state. It returns errors rather than failing the
// test so the sensitivity meta-test can use it on a deliberately bad variant.
func midTxReadState(raw *sql.DB) (midTxState, error) {
	ctx := context.Background()
	var st midTxState
	if err := raw.QueryRowContext(ctx,
		`SELECT delivered_at FROM signal_inbox WHERE signal_id = ?`, midTxSignalID,
	).Scan(&st.deliveredAt); err != nil {
		return midTxState{}, fmt.Errorf("read signal_inbox: %w", err)
	}
	if err := raw.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM execution_events WHERE instance_id = ? AND event_type = 'SignalReceived'`,
		midTxInstance,
	).Scan(&st.signalReceived); err != nil {
		return midTxState{}, fmt.Errorf("count SignalReceived: %w", err)
	}
	if err := raw.QueryRowContext(ctx,
		`SELECT status, version FROM workflow_instances WHERE instance_id = ?`, midTxInstance,
	).Scan(&st.status, &st.version); err != nil {
		return midTxState{}, fmt.Errorf("read workflow_instances: %w", err)
	}
	return st, nil
}

// midTxMustReadCurrentSteps reads workflow_instances.current_steps (a JSON
// array) and decodes it, so forward and rebuilt projections compare as values.
func midTxMustReadCurrentSteps(t *testing.T, raw *sql.DB) []string {
	t.Helper()
	var rawJSON string
	if err := raw.QueryRowContext(context.Background(),
		`SELECT current_steps FROM workflow_instances WHERE instance_id = ?`, midTxInstance,
	).Scan(&rawJSON); err != nil {
		t.Fatalf("read current_steps: %v", err)
	}
	var steps []string
	if err := json.Unmarshal([]byte(rawJSON), &steps); err != nil {
		t.Fatalf("decode current_steps %q: %v", rawJSON, err)
	}
	return steps
}

// midTxMustReadState is midTxReadState for the real tests (fails on error).
func midTxMustReadState(t *testing.T, raw *sql.DB) midTxState {
	t.Helper()
	st, err := midTxReadState(raw)
	if err != nil {
		t.Fatalf("read B3 state: %v", err)
	}
	return st
}

// midTxCheckPreCommit is the invariant for a crash anywhere before COMMIT. It
// returns nil iff the database shows the fully rolled-back pre-commit state:
// delivered_at IS NULL, zero SignalReceived events, instance waiting at its
// original version. Canonical: Finalization B3 (single tx per B3).
func midTxCheckPreCommit(st midTxState) error {
	if st.deliveredAt.Valid {
		return fmt.Errorf("pre-commit violation: signal delivered_at=%q set, want NULL", st.deliveredAt.String)
	}
	if st.signalReceived != 0 {
		return fmt.Errorf("pre-commit violation: %d SignalReceived event(s), want 0", st.signalReceived)
	}
	if st.status != "waiting" {
		return fmt.Errorf("pre-commit violation: instance status=%q, want waiting", st.status)
	}
	if st.version != midTxWaitVersion {
		return fmt.Errorf("pre-commit violation: instance version=%d, want %d", st.version, midTxWaitVersion)
	}
	return nil
}

// midTxCheckPostCommit is the invariant after a committed B3 delivery:
// delivered_at IS NOT NULL, exactly one SignalReceived event, instance running
// at the resumed version. Canonical: Finalization B3.
func midTxCheckPostCommit(st midTxState) error {
	if !st.deliveredAt.Valid {
		return fmt.Errorf("post-commit violation: signal delivered_at IS NULL, want set")
	}
	if st.signalReceived != 1 {
		return fmt.Errorf("post-commit violation: %d SignalReceived event(s), want exactly 1", st.signalReceived)
	}
	if st.status != "running" {
		return fmt.Errorf("post-commit violation: instance status=%q, want running", st.status)
	}
	if st.version != midTxResumedVersion {
		return fmt.Errorf("post-commit violation: instance version=%d, want %d", st.version, midTxResumedVersion)
	}
	return nil
}

// ── Invariant-checker self-test (enumerated state space) ─────────────────────

// TestMidTxInvariantCheckers enumerates the observable-state space around the
// B3 transaction and pins which states each checker accepts. It proves the
// checkers themselves are not vacuous before they are used to judge crashes.
func TestMidTxInvariantCheckers(t *testing.T) {
	valid := sql.NullString{String: midTxNow.Format(time.RFC3339Nano), Valid: true}
	none := sql.NullString{}

	cases := []struct {
		name     string
		st       midTxState
		wantPre  bool // checkPreCommit must return nil
		wantPost bool // checkPostCommit must return nil
	}{
		{"pre-commit clean", midTxState{none, 0, "waiting", midTxWaitVersion}, true, false},
		{"post-commit clean", midTxState{valid, 1, "running", midTxResumedVersion}, false, true},
		{"delivered without SignalReceived (non-atomic)", midTxState{valid, 0, "waiting", midTxWaitVersion}, false, false},
		{"SignalReceived without delivered_at", midTxState{none, 1, "waiting", midTxWaitVersion}, false, false},
		{"pre-commit but instance already running", midTxState{none, 0, "running", midTxWaitVersion}, false, false},
		{"pre-commit with version drift", midTxState{none, 0, "waiting", midTxWaitVersion + 1}, false, false},
		{"post-commit with duplicate SignalReceived", midTxState{valid, 2, "running", midTxResumedVersion}, false, false},
		{"post-commit still waiting", midTxState{valid, 1, "waiting", midTxWaitVersion}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := midTxCheckPreCommit(tc.st) == nil; got != tc.wantPre {
				t.Errorf("checkPreCommit accepts=%v, want %v (err=%v)", got, tc.wantPre, midTxCheckPreCommit(tc.st))
			}
			if got := midTxCheckPostCommit(tc.st) == nil; got != tc.wantPost {
				t.Errorf("checkPostCommit accepts=%v, want %v (err=%v)", got, tc.wantPost, midTxCheckPostCommit(tc.st))
			}
		})
	}
}

// ── Crash matrix: kill INSIDE the open B3 transaction ────────────────────────

// TestDeliverSignal_MidTxCrash kills the child at each intra-transaction point
// of the real DeliverSignal. Acceptance (STATE.md M18-B5): after reopen,
// delivered_at IS NULL and no SignalReceived event; recovery then delivers
// exactly once, and RebuildState converges to the forward projection.
func TestDeliverSignal_MidTxCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip mid-transaction crash test")
	}
	for _, stage := range midTxStages {
		t.Run(stage, func(t *testing.T) {
			dbPath := midTxPrepareDB(t)
			midTxRawExec(t, dbPath, midTxTriggerInstall)

			stdout, runErr := midTxRunChild(t, dbPath, "CRASH_AT="+stage)
			midTxRequireSIGKILL(t, runErr)
			if !midTxHasLine(stdout, "AT "+stage) {
				t.Fatalf("crash point %q was not reached inside the B3 tx; child stdout=%q", stage, stdout)
			}
			if midTxHasLinePrefix(stdout, "DELIVERED") {
				t.Fatalf("DeliverSignal returned before the crash; child stdout=%q", stdout)
			}

			midTxRawExec(t, dbPath, midTxTriggerDrop)
			db := midTxOpen(t, dbPath)
			raw := db.ExposedDB()

			st := midTxMustReadState(t, raw)
			if err := midTxCheckPreCommit(st); err != nil {
				t.Fatalf("crash at %s: %v", stage, err)
			}

			// Recovery: the same delivery, on the reopened store, must deliver once.
			s := storage.NewSQLiteStorage(db, nil)
			ctx := context.Background()
			in, err := midTxDeliverInput()
			if err != nil {
				t.Fatalf("build input: %v", err)
			}
			outcome, err := s.DeliverSignal(ctx, in)
			if err != nil {
				t.Fatalf("recovery DeliverSignal: %v", err)
			}
			if outcome != storage.Delivered {
				t.Fatalf("recovery outcome=%s, want Delivered", midTxOutcomeName(outcome))
			}
			st = midTxMustReadState(t, raw)
			if err := midTxCheckPostCommit(st); err != nil {
				t.Fatalf("after recovery: %v", err)
			}

			// Idempotency: a second delivery of the same signal is a no-op.
			outcome, err = s.DeliverSignal(ctx, in)
			if err != nil {
				t.Fatalf("second DeliverSignal: %v", err)
			}
			if outcome != storage.DeliverNoOp {
				t.Fatalf("second outcome=%s, want NoOp", midTxOutcomeName(outcome))
			}
			st = midTxMustReadState(t, raw)
			if err := midTxCheckPostCommit(st); err != nil {
				t.Fatalf("after second delivery (duplicate?): %v", err)
			}
			if got := instanceStatus(t, raw, midTxInstance); got != "running" {
				t.Fatalf("instance status=%q after recovery, want running", got)
			}

			// Forward ≡ rebuild, convergence only (CE ruling, LA-B5): match the
			// existing post-commit test, which asserts status after RebuildState.
			// current_steps is compared as well. The version is deliberately NOT
			// asserted: this fixture seeds instances via UpsertInstance with NO
			// WorkflowStarted (or other) events, so RebuildState sees only the
			// SignalReceived event and cannot reproduce the forward version (the
			// seeding and resume writes that moved it to 3). That is a fixture
			// artifact, not a rebuild defect; both versions are logged below.
			forwardSteps := midTxMustReadCurrentSteps(t, raw)
			if err := s.RebuildState(ctx); err != nil {
				t.Fatalf("RebuildState: %v", err)
			}
			if got := instanceStatus(t, raw, midTxInstance); got != "running" {
				t.Fatalf("after RebuildState status=%q, want running (forward status running)", got)
			}
			rebuiltSteps := midTxMustReadCurrentSteps(t, raw)
			if !slices.Equal(forwardSteps, rebuiltSteps) {
				t.Fatalf("after RebuildState current_steps=%v, forward=%v", rebuiltSteps, forwardSteps)
			}
			t.Logf("RebuildState convergence: status=running, current_steps=%v; version forward=%d rebuilt=%d (version not asserted: fixture artifact)",
				rebuiltSteps, st.version, midTxMustReadState(t, raw).version)
		})
	}
}

// TestDeliverSignal_MidTxPostCommitControl is the control for the matrix above:
// the child is killed strictly AFTER DeliverSignal committed. It also proves that
// crash_point is inert when its stage never matches (CRASH_AT=none), i.e. the
// triggers do not perturb the normal path.
func TestDeliverSignal_MidTxPostCommitControl(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip mid-transaction post-commit control")
	}
	dbPath := midTxPrepareDB(t)
	midTxRawExec(t, dbPath, midTxTriggerInstall)

	stdout, runErr := midTxRunChild(t, dbPath, "CRASH_AT=none")
	midTxRequireSIGKILL(t, runErr)
	if midTxHasLinePrefix(stdout, "AT ") {
		t.Fatalf("control run hit a crash point; child stdout=%q", stdout)
	}
	if !midTxHasLine(stdout, "DELIVERED Delivered") {
		t.Fatalf("control run did not complete DeliverSignal; child stdout=%q", stdout)
	}

	midTxRawExec(t, dbPath, midTxTriggerDrop)
	db := midTxOpen(t, dbPath)
	raw := db.ExposedDB()
	if err := midTxCheckPostCommit(midTxMustReadState(t, raw)); err != nil {
		t.Fatalf("post-commit control: %v", err)
	}

	s := storage.NewSQLiteStorage(db, nil)
	in, err := midTxDeliverInput()
	if err != nil {
		t.Fatalf("build input: %v", err)
	}
	outcome, err := s.DeliverSignal(context.Background(), in)
	if err != nil {
		t.Fatalf("DeliverSignal after control kill: %v", err)
	}
	if outcome != storage.DeliverNoOp {
		t.Fatalf("second outcome=%s, want NoOp", midTxOutcomeName(outcome))
	}
	if err := midTxCheckPostCommit(midTxMustReadState(t, raw)); err != nil {
		t.Fatalf("post-commit control after second delivery: %v", err)
	}
}

// ── Sensitivity proof: the harness must FAIL against a non-atomic variant ────

// TestDeliverSignal_MidTxNonAtomicSensitivity runs the same harness against
// midTxNonAtomicDeliver (autocommit writes, no transaction). The invariant
// checker MUST report a violation here; otherwise the atomic PASS above proves
// nothing. Stages are those where the non-atomic variant has already made a
// write durable before the kill: after_append (guard durable, no
// SignalReceived) and before_commit (guard and append durable, resume not).
//
// Why after_guard is EXCLUDED from this proof: the after_guard trigger is an
// AFTER UPDATE on signal_inbox, so it fires inside write 1 (the guard
// statement), before that statement's autocommit completes. When the process
// dies there, the guard statement itself is rolled back, so under the
// non-atomic variant no write is durable yet and the checker correctly reports
// NO violation (verified: delivered_at NULL, 0 SignalReceived, waiting, v2).
// after_guard therefore cannot distinguish atomic from non-atomic writes. It
// stays in TestDeliverSignal_MidTxCrash as a genuine in-transaction kill point
// for the real, atomic DeliverSignal. The non-atomic variant's violation is
// observable only after write 1 has become durable: after_append fires during
// write 2 (guard already durable), and before_commit fires during write 3
// (guard and append already durable; the trigger runs after write 3's row
// update and before COMMIT in the atomic variant).
func TestDeliverSignal_MidTxNonAtomicSensitivity(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skip mid-transaction sensitivity test")
	}
	for _, stage := range []string{"after_append", "before_commit"} {
		t.Run(stage, func(t *testing.T) {
			dbPath := midTxPrepareDB(t)
			midTxRawExec(t, dbPath, midTxTriggerInstall)

			stdout, runErr := midTxRunChild(t, dbPath,
				"CRASH_AT="+stage, "CRASH_DELIVER_IMPL=nonatomic")
			midTxRequireSIGKILL(t, runErr)
			if !midTxHasLine(stdout, "AT "+stage) {
				t.Fatalf("non-atomic crash point %q not reached; child stdout=%q", stage, stdout)
			}

			midTxRawExec(t, dbPath, midTxTriggerDrop)
			db := midTxOpen(t, dbPath)
			st := midTxMustReadState(t, db.ExposedDB())
			err := midTxCheckPreCommit(st)
			if err == nil {
				t.Fatalf("sensitivity failure: non-atomic variant killed at %s but the checker reports no violation (state=%+v)", stage, st)
			}
			t.Logf("non-atomic variant at %s: checker reports violation as required: %v", stage, err)
		})
	}
}
