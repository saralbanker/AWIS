package storage_test

// la_busy_experiment_test.go — LA-P3-BUSY bounded reproduction experiment for
// backlog item M18-B2: one unreproduced SQLITE_BUSY on `PRAGMA journal_mode=WAL`
// during the first `awis submit` issued right after `awis start`.
//
// EXPERIMENT ONLY. No production code is changed and no retry logic exists.
// Every experiment is gated by LA_BUSY_EXPERIMENT=1 and skips otherwise, so a
// normal `go test` never runs it. Run with:
//
//	LA_BUSY_EXPERIMENT=1 go test ./internal/storage/ -run 'TestLABusy' -count=1 -v -timeout 30m
//
// Subprocess roles are dispatched by TestLABusyHelper (GO_TEST_HELPER_PROCESS=1,
// role in LA_BUSY_ROLE; standard re-exec idiom as in crash_test.go). Children
// report on stdout with a "LABUSY " prefix as key=value fields; a trailing
// " err=" carries the verbatim error text (rest of line). The parent
// coordinates contenders with file barriers: each child prints READY and polls
// for its go-file; the parent creates the go-file once every contender is ready.
//
// laDSN and laPragmas are verbatim copies of storage.buildDSN and
// storage.applyPragmas (internal/storage/db.go). They are unexported, so this
// external test package cannot call them; they must be kept in step with db.go.

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/storage"
)

const (
	laGateEnv      = "LA_BUSY_EXPERIMENT"
	laHelperEnv    = "GO_TEST_HELPER_PROCESS"
	laRoleEnv      = "LA_BUSY_ROLE"
	laPathEnv      = "LA_BUSY_PATH"
	laGoEnv        = "LA_BUSY_GO"
	laStopEnv      = "LA_BUSY_STOP"
	laHoldEnv      = "LA_BUSY_HOLD_MS"
	laChildTimeout = 90 * time.Second
	laModuleRoot   = "../.."
)

// laPragmas mirrors storage.applyPragmas (db.go) verbatim, same order.
var laPragmas = []string{
	`PRAGMA journal_mode=WAL`,
	`PRAGMA busy_timeout=5000`,
	`PRAGMA foreign_keys=ON`,
	`PRAGMA synchronous=NORMAL`,
}

// laDSN mirrors storage.buildDSN (db.go) verbatim.
func laDSN(path string) string {
	const opts = "_txlock=immediate&_pragma=busy_timeout(5000)"
	if strings.Contains(path, "?") {
		return path + "&" + opts
	}
	return path + "?" + opts
}

// laGate skips the calling experiment unless LA_BUSY_EXPERIMENT=1.
func laGate(t *testing.T) {
	t.Helper()
	if os.Getenv(laGateEnv) != "1" {
		t.Skip("LA-P3-BUSY experiment: set LA_BUSY_EXPERIMENT=1 to run")
	}
}

// ---------------------------------------------------------------------------
// Child side
// ---------------------------------------------------------------------------

// TestLABusyHelper is the subprocess entry point. It does nothing unless
// GO_TEST_HELPER_PROCESS=1 (so it is inert in normal test runs).
func TestLABusyHelper(t *testing.T) {
	if os.Getenv(laHelperEnv) != "1" {
		return
	}
	var err error
	switch role := os.Getenv(laRoleEnv); role {
	case "open":
		err = laRoleOpen()
	case "hold-write":
		err = laRoleHoldWrite()
	case "ckpt-loop":
		err = laRoleCkptLoop()
	case "open-loop":
		err = laRoleOpenLoop()
	case "raw-pragma":
		err = laRoleRawPragma()
	case "pragma-timed":
		err = laRolePragmaTimed()
	default:
		err = fmt.Errorf("unknown role %q", role)
	}
	if err != nil {
		laSay("FATAL err=%s", laErrText(err))
		os.Exit(2)
	}
	os.Exit(0)
}

func laSay(format string, args ...any) {
	fmt.Printf("LABUSY "+format+"\n", args...)
}

func laErrText(err error) string {
	if err == nil {
		return ""
	}
	return laOneLine(err.Error())
}

func laOneLine(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}

func laHold() (time.Duration, error) {
	ms, err := strconv.Atoi(os.Getenv(laHoldEnv))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", laHoldEnv, err)
	}
	return time.Duration(ms) * time.Millisecond, nil
}

// laWaitGo blocks until the parent creates the barrier file.
func laWaitGo(goFile string) error {
	deadline := time.Now().Add(laChildTimeout)
	for {
		_, err := os.Stat(goFile)
		if err == nil {
			return nil
		}
		if !os.IsNotExist(err) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("barrier %s never released", goFile)
		}
		time.Sleep(time.Millisecond)
	}
}

// laRoleOpen is the plain contender: one storage.Open call, timed.
func laRoleOpen() error {
	laSay("READY")
	if err := laWaitGo(os.Getenv(laGoEnv)); err != nil {
		return err
	}
	start := time.Now()
	db, err := storage.Open(os.Getenv(laPathEnv), nil)
	end := time.Now()
	res, errText := "ok", ""
	switch {
	case err != nil:
		res, errText = "err", laErrText(err)
	default:
		if cerr := db.Close(); cerr != nil {
			res, errText = "err", "close: "+laErrText(cerr)
		}
	}
	laSay("RESULT res=%s start_ns=%d end_ns=%d dur_ns=%d err=%s",
		res, start.UnixNano(), end.UnixNano(), end.Sub(start).Nanoseconds(), errText)
	return nil
}

// laRoleHoldWrite takes the write lock with BEGIN IMMEDIATE, writes one row to
// the scratch table, holds for LA_BUSY_HOLD_MS, then COMMITs.
func laRoleHoldWrite() error {
	hold, err := laHold()
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := sql.Open("sqlite", laDSN(os.Getenv(laPathEnv)))
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer func() { _ = db.Close() }() // process exits right after; close error is moot
	db.SetMaxOpenConns(1)
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db.Conn: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		laSay("HELD ok=0 ns=%d err=%s", time.Now().UnixNano(), laErrText(err))
		return nil
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO la_scratch (v) VALUES ('holder')`); err != nil {
		laSay("HELD ok=0 ns=%d err=%s", time.Now().UnixNano(), laErrText(err))
		_, _ = conn.ExecContext(ctx, "ROLLBACK") // best effort; process exits next
		return nil
	}
	laSay("HELD ok=1 ns=%d err=", time.Now().UnixNano())
	time.Sleep(hold)
	_, cerr := conn.ExecContext(ctx, "COMMIT")
	laSay("COMMITTED ns=%d err=%s", time.Now().UnixNano(), laErrText(cerr))
	return nil
}

// laRoleCkptLoop (E3 holder) seeds recent committed WAL frames, then loops
// PRAGMA wal_checkpoint(TRUNCATE) for LA_BUSY_HOLD_MS.
func laRoleCkptLoop() error {
	hold, err := laHold()
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := sql.Open("sqlite", laDSN(os.Getenv(laPathEnv)))
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed begin: %w", err)
	}
	for i := 0; i < 500; i++ {
		if _, err := tx.ExecContext(ctx, `INSERT INTO la_scratch (v) VALUES ('ckpt')`); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("seed insert: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("seed commit: %w", err)
	}

	laSay("READY")
	if err := laWaitGo(os.Getenv(laGoEnv)); err != nil {
		return err
	}
	deadline := time.Now().Add(hold)
	var iters, errs, busy int
	first := ""
	for time.Now().Before(deadline) {
		var blocked, logFrames, ckpt int
		err := db.QueryRowContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`).Scan(&blocked, &logFrames, &ckpt)
		iters++
		if err != nil {
			errs++
			if first == "" {
				first = err.Error()
			}
			continue
		}
		if blocked != 0 {
			busy++
		}
	}
	laSay("CKPT iters=%d errs=%d busy=%d err=%s", iters, errs, busy, laOneLine(first))
	return nil
}

// laRoleOpenLoop (E3 contender) calls storage.Open repeatedly until the parent
// creates LA_BUSY_STOP. Each failed attempt is reported as an ATT line.
func laRoleOpenLoop() error {
	laSay("READY")
	if err := laWaitGo(os.Getenv(laGoEnv)); err != nil {
		return err
	}
	stopFile := os.Getenv(laStopEnv)
	var attempts, oks, errs int
	for {
		_, serr := os.Stat(stopFile)
		if serr == nil {
			break
		}
		if !os.IsNotExist(serr) {
			return serr
		}
		attempts++
		db, err := storage.Open(os.Getenv(laPathEnv), nil)
		if err != nil {
			errs++
			laSay("ATT n=%d err=%s", attempts, laErrText(err))
			continue
		}
		oks++
		if cerr := db.Close(); cerr != nil {
			errs++
			laSay("ATT n=%d err=%s", attempts, "close: "+laErrText(cerr))
		}
	}
	laSay("LOOP attempts=%d ok=%d errs=%d", attempts, oks, errs)
	return nil
}

// laRoleRawPragma (E4 process A) emulates storage.Open as two explicit steps:
// sql.Open returns (A_OPENED), then, after the barrier, PRAGMA journal_mode=WAL.
func laRoleRawPragma() error {
	db, err := sql.Open("sqlite", laDSN(os.Getenv(laPathEnv)))
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	laSay("A_OPENED ns=%d", time.Now().UnixNano())
	if err := laWaitGo(os.Getenv(laGoEnv)); err != nil {
		return err
	}
	_, perr := db.Exec(laPragmas[0])
	errText := ""
	if perr != nil {
		errText = fmt.Sprintf("storage: exec %q: %v", laPragmas[0], perr)
	}
	laSay("A_PRAGMA ns=%d err=%s", time.Now().UnixNano(), laOneLine(errText))
	return nil
}

// laRolePragmaTimed (E2b contender) runs the applyPragmas statements one by one
// on a single connection (as storage.Open does), timing each, and stops at the
// first error (as storage.Open does).
func laRolePragmaTimed() error {
	db, err := sql.Open("sqlite", laDSN(os.Getenv(laPathEnv)))
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	laSay("READY")
	if err := laWaitGo(os.Getenv(laGoEnv)); err != nil {
		return err
	}
	for i, p := range laPragmas {
		t0 := time.Now()
		_, perr := db.Exec(p)
		d := time.Since(t0)
		errText := ""
		if perr != nil {
			errText = fmt.Sprintf("storage: exec %q: %v", p, perr)
		}
		laSay("STMT i=%d dur_ns=%d err=%s", i, d.Nanoseconds(), laOneLine(errText))
		if perr != nil {
			break
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Parent side
// ---------------------------------------------------------------------------

// laProc is one spawned child. Its LABUSY lines are read by a goroutine into
// lines; lines that do not match the awaited tag are kept in others.
type laProc struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	lines  chan string
	stderr *bytes.Buffer
	others []string
}

func laSpawn(role string, env ...string) (*laProc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), laChildTimeout)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLABusyHelper$")
	cmd.Env = append(os.Environ(), laHelperEnv+"=1", laRoleEnv+"="+role)
	cmd.Env = append(cmd.Env, env...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("laSpawn %s: StdoutPipe: %w", role, err)
	}
	p := &laProc{cmd: cmd, cancel: cancel, lines: make(chan string, 4096), stderr: &bytes.Buffer{}}
	cmd.Stderr = p.stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("laSpawn %s: Start: %w", role, err)
	}
	go func() {
		defer close(p.lines)
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			if rest, ok := strings.CutPrefix(sc.Text(), "LABUSY "); ok {
				p.lines <- rest
			}
		}
	}()
	return p, nil
}

// expect returns the next line with the given tag, keeping skipped lines in
// p.others. A FATAL line from the child is returned as an error.
func (p *laProc) expect(tag string) (string, error) {
	deadline := time.NewTimer(laChildTimeout)
	defer deadline.Stop()
	for {
		select {
		case line, ok := <-p.lines:
			if !ok {
				return "", fmt.Errorf("child closed stdout before %s", tag)
			}
			if strings.HasPrefix(line, "FATAL") {
				return "", fmt.Errorf("child %s", line)
			}
			if strings.HasPrefix(line, tag) {
				return line, nil
			}
			p.others = append(p.others, line)
		case <-deadline.C:
			return "", fmt.Errorf("timeout waiting for %s", tag)
		}
	}
}

// finish drains remaining output, reaps the child and returns its exit error
// together with its one-line stderr.
func (p *laProc) finish() (string, error) {
	for line := range p.lines {
		p.others = append(p.others, line)
	}
	exitErr := p.cmd.Wait()
	p.cancel()
	return laOneLine(p.stderr.String()), exitErr
}

// laKV splits "TAG k=v k=v ... err=<rest of line>" into fields and err text.
func laKV(line string) (map[string]string, string) {
	head, errText := line, ""
	if i := strings.Index(line, " err="); i >= 0 {
		head, errText = line[:i], line[i+len(" err="):]
	}
	kv := make(map[string]string)
	for _, f := range strings.Fields(head) {
		if k, v, ok := strings.Cut(f, "="); ok {
			kv[k] = v
		}
	}
	return kv, errText
}

func laInt(kv map[string]string, key string) (int64, error) {
	v, ok := kv[key]
	if !ok {
		return 0, fmt.Errorf("missing field %s", key)
	}
	return strconv.ParseInt(v, 10, 64)
}

// laRelease creates a barrier file, releasing every child polling for it.
func laRelease(path string) error {
	return os.WriteFile(path, []byte("go\n"), 0o600)
}

// laClassify names the statement a storage error came from.
func laClassify(errText string) string {
	const execPrefix = `storage: exec "`
	switch {
	case errText == "":
		return "none"
	case strings.Contains(errText, execPrefix):
		rest := errText[strings.Index(errText, execPrefix)+len(execPrefix):]
		if i := strings.Index(rest, `": `); i >= 0 {
			return "pragma " + rest[:i]
		}
		return "pragma (unparsed)"
	case strings.Contains(errText, "storage: migrate"):
		return "migration"
	case strings.Contains(errText, "storage: chmod"):
		return "chmod"
	case strings.Contains(errText, "storage: open sqlite"):
		return "sql.Open"
	default:
		return "other"
	}
}

// laInitDB creates an initialised DB (storage.Open + Close) and the scratch
// table used by the holder roles.
func laInitDB(path string) error {
	db, err := storage.Open(path, nil)
	if err != nil {
		return fmt.Errorf("init storage.Open: %w", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("init Close: %w", err)
	}
	raw, err := sql.Open("sqlite", laDSN(path))
	if err != nil {
		return fmt.Errorf("init sql.Open: %w", err)
	}
	_, execErr := raw.Exec(`CREATE TABLE IF NOT EXISTS la_scratch (id INTEGER PRIMARY KEY, v TEXT NOT NULL)`)
	closeErr := raw.Close()
	if err := errors.Join(execErr, closeErr); err != nil {
		return fmt.Errorf("init scratch table: %w", err)
	}
	return nil
}

// laTally aggregates one experiment's outcomes for the report.
type laTally struct {
	name       string
	conditions string
	trials     int // attempted trials
	harness    int // trials that could not complete (child/harness error)
	failTrials int // completed trials with at least one failing run
	runs       int // child runs / statements / Open attempts observed
	failures   int
	busy       int // failures whose text has SQLITE_BUSY or "database is locked"
	classes    map[string]int
	texts      map[string]int
	sides      map[string]int // failures attributed to a named process role
}

func newLATally(name, conditions string) *laTally {
	return &laTally{name: name, conditions: conditions, classes: map[string]int{}, texts: map[string]int{}, sides: map[string]int{}}
}

// record notes one run; errText == "" means success. Returns true on failure.
func (tl *laTally) record(errText string) bool {
	tl.runs++
	if errText == "" {
		return false
	}
	tl.failures++
	if strings.Contains(errText, "SQLITE_BUSY") || strings.Contains(errText, "database is locked") {
		tl.busy++
	}
	tl.classes[laClassify(errText)]++
	tl.texts[errText]++
	return true
}

// recordSide is record plus per-role attribution of the failure (side != "").
func (tl *laTally) recordSide(side, errText string) bool {
	failed := tl.record(errText)
	if failed && side != "" {
		tl.sides[side]++
	}
	return failed
}

func (tl *laTally) endTrial(t *testing.T, i int, failed bool, err error) {
	t.Helper()
	tl.trials++
	if err != nil {
		tl.harness++
		t.Logf("LA-BUSY-HARNESS name=%s trial=%d err=%q", tl.name, i, err.Error())
		return
	}
	if failed {
		tl.failTrials++
	}
}

func (tl *laTally) report(t *testing.T) {
	t.Helper()
	t.Logf("LA-BUSY-SUMMARY name=%s trials=%d completed=%d harness_errors=%d runs=%d failures=%d busy=%d failing_trials=%d conditions=%q",
		tl.name, tl.trials, tl.trials-tl.harness, tl.harness, tl.runs, tl.failures, tl.busy, tl.failTrials, tl.conditions)
	for _, k := range laSortedKeys(tl.classes) {
		t.Logf("LA-BUSY-CLASS name=%s class=%q count=%d", tl.name, k, tl.classes[k])
	}
	for _, k := range laSortedKeys(tl.texts) {
		t.Logf("LA-BUSY-TEXT name=%s count=%d text=%q", tl.name, tl.texts[k], k)
	}
	for _, k := range laSortedKeys(tl.sides) {
		t.Logf("LA-BUSY-SIDE name=%s side=%q failures=%d", tl.name, k, tl.sides[k])
	}
}

func laSortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// laCollectOpenResults reads each open-role child's RESULT line and reaps it.
// A child that exits non-zero is a harness error, not a data point. side (if
// non-empty) attributes failures to that role in the tally.
func laCollectOpenResults(procs []*laProc, tl *laTally, side string) (bool, error) {
	failed := false
	for _, p := range procs {
		line, err := p.expect("RESULT")
		if err != nil {
			return failed, err
		}
		kv, errText := laKV(line)
		if kv["res"] != "ok" && errText == "" {
			errText = "(res=" + kv["res"] + " with empty error text)"
		}
		if tl.recordSide(side, errText) {
			failed = true
		}
		if stderr, exitErr := p.finish(); exitErr != nil {
			return failed, fmt.Errorf("child exit: %w (stderr: %s)", exitErr, stderr)
		}
	}
	return failed, nil
}

// ---------------------------------------------------------------------------
// E1 — fresh-file race
// ---------------------------------------------------------------------------

func laE1Trial(dir string, tl *laTally) (bool, error) {
	path := filepath.Join(dir, "fresh.db")
	goFile := filepath.Join(dir, "go")
	var procs []*laProc
	defer func() {
		for _, p := range procs {
			p.cancel()
		}
	}()
	for range 2 {
		p, err := laSpawn("open", laPathEnv+"="+path, laGoEnv+"="+goFile)
		if err != nil {
			return false, err
		}
		procs = append(procs, p)
	}
	for _, p := range procs {
		if _, err := p.expect("READY"); err != nil {
			return false, err
		}
	}
	if err := laRelease(goFile); err != nil {
		return false, err
	}
	return laCollectOpenResults(procs, tl, "")
}

func TestLABusyE1FreshFileRace(t *testing.T) {
	laGate(t)
	const trials = 200
	tl := newLATally("E1", "DB file absent; 2 concurrent storage.Open subprocesses; file barrier; 200 trials")
	for i := range trials {
		dir := t.TempDir()
		failed, err := laE1Trial(dir, tl)
		tl.endTrial(t, i, failed, err)
		if failed {
			t.Logf("LA-BUSY-FAIL name=E1 trial=%d", i)
		}
	}
	tl.report(t)
}

// ---------------------------------------------------------------------------
// E2 — holder write-transaction; contender is storage.Open
// ---------------------------------------------------------------------------

// laHolderWindow spawns the holder, waits until it holds the write lock,
// releases the contender, and waits for the holder's COMMITTED line. It
// returns the HELD and COMMITTED field maps.
func laHolderWindow(path, goFile string, contender *laProc) (map[string]string, map[string]string, error) {
	h, err := laSpawn("hold-write", laPathEnv+"="+path, laHoldEnv+"=1500")
	if err != nil {
		return nil, nil, err
	}
	heldLine, err := h.expect("HELD")
	if err != nil {
		h.cancel()
		return nil, nil, err
	}
	if err := laRelease(goFile); err != nil {
		h.cancel()
		return nil, nil, err
	}
	held, heldErr := laKV(heldLine)
	if held["ok"] != "1" {
		h.cancel()
		return held, nil, fmt.Errorf("holder did not take write lock: %s", heldErr)
	}
	commitLine, err := h.expect("COMMITTED")
	if err != nil {
		h.cancel()
		return held, nil, err
	}
	committed, commitErr := laKV(commitLine)
	if commitErr != "" {
		return held, committed, fmt.Errorf("holder COMMIT failed: %s", commitErr)
	}
	if stderr, exitErr := h.finish(); exitErr != nil {
		return held, committed, fmt.Errorf("holder exit: %w (stderr: %s)", exitErr, stderr)
	}
	return held, committed, nil
}

func laE2Trial(t *testing.T, dir string, tl *laTally, idx int) (bool, error) {
	path := filepath.Join(dir, "e2.db")
	goFile := filepath.Join(dir, "go")
	if err := laInitDB(path); err != nil {
		return false, err
	}
	c, err := laSpawn("open", laPathEnv+"="+path, laGoEnv+"="+goFile)
	if err != nil {
		return false, err
	}
	defer c.cancel()
	if _, err := c.expect("READY"); err != nil {
		return false, err
	}
	_, committed, err := laHolderWindow(path, goFile, c)
	if err != nil {
		return false, err
	}
	line, err := c.expect("RESULT")
	if err != nil {
		return false, err
	}
	res, errText := laKV(line)
	if res["res"] != "ok" && errText == "" {
		errText = "(res=" + res["res"] + " with empty error text)"
	}
	dur, err := laInt(res, "dur_ns")
	if err != nil {
		return false, err
	}
	endNs, err := laInt(res, "end_ns")
	if err != nil {
		return false, err
	}
	commitNs, err := laInt(committed, "ns")
	if err != nil {
		return false, err
	}
	failed := tl.record(errText)
	if stderr, exitErr := c.finish(); exitErr != nil {
		return failed, fmt.Errorf("contender exit: %w (stderr: %s)", exitErr, stderr)
	}
	t.Logf("LA-BUSY-E2 trial=%d res=%s dur_ms=%.1f returned_before_commit=%t err=%q",
		idx, res["res"], float64(dur)/1e6, endNs < commitNs, errText)
	return failed, nil
}

func TestLABusyE2HolderWriteTx(t *testing.T) {
	laGate(t)
	const trials = 50
	tl := newLATally("E2", "holder: BEGIN IMMEDIATE + INSERT, hold 1500ms, COMMIT; contender: storage.Open during hold; 50 trials")
	for i := range trials {
		dir := t.TempDir()
		failed, err := laE2Trial(t, dir, tl, i)
		tl.endTrial(t, i, failed, err)
	}
	tl.report(t)
}

// laE2bTrial is the per-statement variant: the contender runs the four
// applyPragmas statements one at a time, each timed, during the same hold.
func laE2bTrial(t *testing.T, dir string, tl *laTally, stmtDur map[int][]time.Duration, idx int) (bool, error) {
	path := filepath.Join(dir, "e2b.db")
	goFile := filepath.Join(dir, "go")
	if err := laInitDB(path); err != nil {
		return false, err
	}
	c, err := laSpawn("pragma-timed", laPathEnv+"="+path, laGoEnv+"="+goFile)
	if err != nil {
		return false, err
	}
	defer c.cancel()
	if _, err := c.expect("READY"); err != nil {
		return false, err
	}
	if _, _, err := laHolderWindow(path, goFile, c); err != nil {
		return false, err
	}
	failed := false
	for range laPragmas {
		line, err := c.expect("STMT")
		if err != nil {
			return failed, err
		}
		kv, errText := laKV(line)
		i, err := strconv.Atoi(kv["i"])
		if err != nil {
			return failed, fmt.Errorf("STMT index: %w", err)
		}
		dur, err := laInt(kv, "dur_ns")
		if err != nil {
			return failed, err
		}
		stmtDur[i] = append(stmtDur[i], time.Duration(dur))
		if tl.record(errText) {
			failed = true
		}
		t.Logf("LA-BUSY-E2B trial=%d stmt=%d dur_ms=%.3f err=%q", idx, i, float64(dur)/1e6, errText)
		if errText != "" {
			break
		}
	}
	if stderr, exitErr := c.finish(); exitErr != nil {
		return failed, fmt.Errorf("pragma-timed exit: %w (stderr: %s)", exitErr, stderr)
	}
	return failed, nil
}

func TestLABusyE2bPragmaStatementTiming(t *testing.T) {
	laGate(t)
	const trials = 50
	tl := newLATally("E2b", "same holder window as E2; contender times each applyPragmas statement; 50 trials")
	stmtDur := map[int][]time.Duration{}
	for i := range trials {
		dir := t.TempDir()
		failed, err := laE2bTrial(t, dir, tl, stmtDur, i)
		tl.endTrial(t, i, failed, err)
	}
	tl.report(t)
	for i, stmt := range laPragmas {
		ds := stmtDur[i]
		sort.Slice(ds, func(a, b int) bool { return ds[a] < ds[b] })
		if len(ds) == 0 {
			t.Logf("LA-BUSY-STMT i=%d stmt=%q n=0", i, stmt)
			continue
		}
		t.Logf("LA-BUSY-STMT i=%d stmt=%q n=%d min_ms=%.3f median_ms=%.3f max_ms=%.3f",
			i, stmt, len(ds), float64(ds[0])/1e6, float64(ds[len(ds)/2])/1e6, float64(ds[len(ds)-1])/1e6)
	}
}

// ---------------------------------------------------------------------------
// E3 — checkpoint contention
// ---------------------------------------------------------------------------

func laE3Trial(t *testing.T, dir string, tl *laTally, idx int) (bool, error) {
	path := filepath.Join(dir, "e3.db")
	goFile := filepath.Join(dir, "go")
	stopFile := filepath.Join(dir, "stop")
	if err := laInitDB(path); err != nil {
		return false, err
	}
	h, err := laSpawn("ckpt-loop", laPathEnv+"="+path, laGoEnv+"="+goFile, laHoldEnv+"=2000")
	if err != nil {
		return false, err
	}
	defer h.cancel()
	if _, err := h.expect("READY"); err != nil {
		return false, err
	}
	c, err := laSpawn("open-loop", laPathEnv+"="+path, laGoEnv+"="+goFile, laStopEnv+"="+stopFile)
	if err != nil {
		return false, err
	}
	defer c.cancel()
	if _, err := c.expect("READY"); err != nil {
		return false, err
	}
	if err := laRelease(goFile); err != nil {
		return false, err
	}
	ckLine, err := h.expect("CKPT")
	if err != nil {
		return false, err
	}
	if err := laRelease(stopFile); err != nil {
		return false, err
	}
	loopLine, err := c.expect("LOOP")
	if err != nil {
		return false, err
	}
	ck, ckErr := laKV(ckLine)
	lp, _ := laKV(loopLine)
	failed := false
	for _, o := range c.others {
		if !strings.HasPrefix(o, "ATT") {
			continue
		}
		_, attErr := laKV(o)
		if tl.record(attErr) {
			failed = true
		}
	}
	okRuns, err := laInt(lp, "ok")
	if err != nil {
		return failed, err
	}
	tl.runs += int(okRuns)
	if stderr, exitErr := h.finish(); exitErr != nil {
		return failed, fmt.Errorf("ckpt-loop exit: %w (stderr: %s)", exitErr, stderr)
	}
	if stderr, exitErr := c.finish(); exitErr != nil {
		return failed, fmt.Errorf("open-loop exit: %w (stderr: %s)", exitErr, stderr)
	}
	t.Logf("LA-BUSY-E3 trial=%d checkpoints=%s ckpt_errs=%s ckpt_busy=%s ckpt_err=%q | contender attempts=%s ok=%s errs=%s",
		idx, ck["iters"], ck["errs"], ck["busy"], ckErr, lp["attempts"], lp["ok"], lp["errs"])
	return failed, nil
}

func TestLABusyE3CheckpointContention(t *testing.T) {
	laGate(t)
	const trials = 20
	tl := newLATally("E3", "holder: committed-row seed then PRAGMA wal_checkpoint(TRUNCATE) loop for 2000ms; contender: storage.Open loop for the same window; 20 trials")
	for i := range trials {
		dir := t.TempDir()
		failed, err := laE3Trial(t, dir, tl, i)
		tl.endTrial(t, i, failed, err)
	}
	tl.report(t)
}

// ---------------------------------------------------------------------------
// E4 — fresh DB, opener vs WAL switch
// ---------------------------------------------------------------------------

func laE4Trial(dir string, tl *laTally) (bool, error) {
	path := filepath.Join(dir, "e4.db")
	goA := filepath.Join(dir, "goA")
	goB := filepath.Join(dir, "goB")
	a, err := laSpawn("raw-pragma", laPathEnv+"="+path, laGoEnv+"="+goA)
	if err != nil {
		return false, err
	}
	defer a.cancel()
	b, err := laSpawn("open", laPathEnv+"="+path, laGoEnv+"="+goB)
	if err != nil {
		return false, err
	}
	defer b.cancel()
	if _, err := a.expect("A_OPENED"); err != nil {
		return false, err
	}
	if _, err := b.expect("READY"); err != nil {
		return false, err
	}
	// B is released first, then A's journal_mode=WAL step.
	if err := laRelease(goB); err != nil {
		return false, err
	}
	if err := laRelease(goA); err != nil {
		return false, err
	}
	aLine, err := a.expect("A_PRAGMA")
	if err != nil {
		return false, err
	}
	_, aErr := laKV(aLine)
	failed := tl.recordSide("A-raw-pragma", aErr)
	if stderr, exitErr := a.finish(); exitErr != nil {
		return failed, fmt.Errorf("raw-pragma exit: %w (stderr: %s)", exitErr, stderr)
	}
	bFailed, err := laCollectOpenResults([]*laProc{b}, tl, "B-storage.Open")
	if err != nil {
		return failed, err
	}
	return failed || bFailed, nil
}

func TestLABusyE4FreshWALSwitch(t *testing.T) {
	laGate(t)
	const trials = 200
	tl := newLATally("E4", "DB file absent; A = sql.Open then barrier then PRAGMA journal_mode=WAL; B = storage.Open released at A_OPENED; 200 trials")
	for i := range trials {
		dir := t.TempDir()
		failed, err := laE4Trial(dir, tl)
		tl.endTrial(t, i, failed, err)
		if failed {
			t.Logf("LA-BUSY-FAIL name=E4 trial=%d", i)
		}
	}
	tl.report(t)
}

// ---------------------------------------------------------------------------
// E5 — CLI level: init, start (background), immediate submit, stop
// ---------------------------------------------------------------------------

type laCLIResult struct {
	code   int
	stdout string
	stderr string
}

func laRunCLI(ctx context.Context, bin, dir string, args ...string) (laCLIResult, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	res := laCLIResult{stdout: so.String(), stderr: se.String()}
	var ee *exec.ExitError
	switch {
	case err == nil:
		res.code = 0
	case errors.As(err, &ee):
		res.code = ee.ExitCode()
	default:
		return res, err
	}
	return res, nil
}

func laBuildAwis(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "awis")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/awis")
	cmd.Dir = laModuleRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./cmd/awis: %v\n%s", err, out)
	}
	return bin
}

func laE5Trial(t *testing.T, bin, root string, tl *laTally, idx int) (bool, error) {
	proj := filepath.Join(root, "proj")
	data := filepath.Join(root, "data")
	pidPath := filepath.Join(data, "awis.pid")
	ctx, cancel := context.WithTimeout(context.Background(), laChildTimeout)
	defer cancel()

	ini, err := laRunCLI(ctx, bin, root, "init", proj)
	if err != nil {
		return false, err
	}
	if ini.code != 0 {
		return false, fmt.Errorf("awis init exit=%d stderr=%s", ini.code, laOneLine(ini.stderr))
	}

	start := exec.CommandContext(ctx, bin, "--data-dir", data, "start")
	start.Dir = proj
	stdout, err := start.StdoutPipe()
	if err != nil {
		return false, fmt.Errorf("start StdoutPipe: %w", err)
	}
	var startErr bytes.Buffer
	start.Stderr = &startErr
	if err := start.Start(); err != nil {
		return false, fmt.Errorf("start: %w", err)
	}
	// Read stdout up to the "● running" line, then drain the rest in background.
	reader := bufio.NewReader(stdout)
	reached := false
	for {
		line, rerr := reader.ReadString('\n')
		if strings.Contains(line, "● running") {
			reached = true
			break
		}
		if rerr != nil {
			break
		}
	}
	// start is reaped as soon as it exits (EOF on its stdout). `awis stop`
	// treats an unreaped zombie as still running and waits its full 10s before
	// failing, so start must not stay unreaped until this function returns.
	exited := make(chan error, 1)
	go func() {
		// Discarding the runtime's post-banner stdout is intentional; the
		// experiment reads stderr separately.
		_, _ = io.Copy(io.Discard, reader)
		exited <- start.Wait()
	}()
	if !reached {
		cancel()
		werr := <-exited
		return false, fmt.Errorf("start never printed running line: wait=%v stderr=%s", werr, laOneLine(startErr.String()))
	}

	// Immediately submit the scaffold's workflow.
	sub, err := laRunCLI(ctx, bin, proj, "--data-dir", data, "submit", "hello-world")
	if err != nil {
		return false, fmt.Errorf("submit run: %w", err)
	}
	subErr := ""
	if sub.code != 0 {
		subErr = laOneLine(sub.stderr)
		if subErr == "" {
			subErr = fmt.Sprintf("(exit=%d empty stderr)", sub.code)
		}
	}
	failed := tl.record(subErr)

	// Wait for the PID file (written by start after the banner), then stop.
	for ctx.Err() == nil {
		if _, serr := os.Stat(pidPath); serr == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	stop, err := laRunCLI(ctx, bin, proj, "--data-dir", data, "stop")
	if err != nil {
		return failed, fmt.Errorf("stop run: %w", err)
	}
	var werr error
	select {
	case werr = <-exited:
	case <-time.After(15 * time.Second):
		cancel()
		werr = <-exited
	}

	t.Logf("LA-BUSY-E5 trial=%d submit_exit=%d submit_err=%q stop_exit=%d stop_err=%q start_exit=%v",
		idx, sub.code, subErr, stop.code, laOneLine(stop.stderr), werr)
	return failed, nil
}

func TestLABusyE5CLI(t *testing.T) {
	laGate(t)
	const trials = 100
	bin := laBuildAwis(t)
	tl := newLATally("E5", "awis built from HEAD; per trial: awis init <proj>; awis --data-dir <trial>/data start (cwd=proj) read to '● running'; immediately awis --data-dir <trial>/data submit hello-world; awis stop; 100 trials")
	for i := range trials {
		root := t.TempDir()
		failed, err := laE5Trial(t, bin, root, tl, i)
		tl.endTrial(t, i, failed, err)
	}
	tl.report(t)
}
