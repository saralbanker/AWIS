//go:build integration

// Package integration is AWIS's binary-level integration tier: every test
// drives the SHIPPED awis binary as a subprocess (never the internal Go
// APIs) and asserts real end-user behaviour. Build-tagged 'integration' so
// 'make verify' / 'go test ./...' never compiles or runs this package; run
// it with 'make integration'.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// awisBinPath is the path to the freshly built awis binary, set once by
// TestMain before any test in this package runs.
var awisBinPath string

// ── fixture ──────────────────────────────────────────────────────────────────

// fixture is a self-contained AWIS project directory plus an optional
// background 'awis start' process, used to drive the shipped binary exactly
// as a real user would from a shell in that directory.
type fixture struct {
	t       *testing.T
	dir     string // project directory; cwd for every invocation
	dataDir string // <dir>/.awis — matches the CLI's default --data-dir

	proc *exec.Cmd
	done chan struct{} // closed once proc.Wait() returns
	werr error         // proc.Wait() result, valid after <-done
}

// newProject returns a fixture rooted at a fresh, empty temp directory.
// Cleanup (killing any surviving background process) is registered
// automatically via t.Cleanup, so it runs even if the test fails.
func newProject(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	f := &fixture{t: t, dir: dir, dataDir: filepath.Join(dir, ".awis")}
	t.Cleanup(f.cleanup)
	return f
}

// writeWorkflow writes yaml to <project>/workflows/<name>, creating the
// workflows/ directory if needed, and returns the absolute path written.
func (f *fixture) writeWorkflow(name, yaml string) string {
	f.t.Helper()
	wfDir := filepath.Join(f.dir, "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		f.t.Fatalf("writeWorkflow: mkdir %s: %v", wfDir, err)
	}
	path := filepath.Join(wfDir, name)
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		f.t.Fatalf("writeWorkflow: write %s: %v", path, err)
	}
	return path
}

// result is the captured outcome of one 'awis' subprocess invocation.
type result struct {
	stdout   string
	stderr   string
	exitCode int
}

// commandTimeout bounds every single 'awis' invocation so a hung binary
// fails the offending test instead of hanging the whole suite.
const commandTimeout = 30 * time.Second

// run executes the awis binary with cwd = the project directory and returns
// its captured stdout/stderr/exit code. It never fails the test on a
// non-zero exit code — callers decide whether that is expected.
func (f *fixture) run(args ...string) result {
	f.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, awisBinPath, args...)
	cmd.Dir = f.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr):
			code = exitErr.ExitCode()
		case ctx.Err() == context.DeadlineExceeded:
			f.t.Fatalf("run(%v): timed out after %s (stdout=%s stderr=%s)", args, commandTimeout, stdout.String(), stderr.String())
		default:
			f.t.Fatalf("run(%v): %v", args, err)
		}
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), exitCode: code}
}

// runJSON runs the binary with a leading global --json flag (which MUST
// precede the subcommand — the stdlib flag package stops scanning flags at
// the first non-flag argument) and strictly unmarshals stdout into v. It
// FAILS the test if stdout is empty or is not valid JSON: this is the
// JSON-contract gate every --json command must pass.
func (f *fixture) runJSON(v any, args ...string) result {
	f.t.Helper()
	res := f.run(append([]string{"--json"}, args...)...)
	if strings.TrimSpace(res.stdout) == "" {
		f.t.Fatalf("runJSON(%v): empty stdout (exit=%d stderr=%s)", args, res.exitCode, res.stderr)
	}
	if err := json.Unmarshal([]byte(res.stdout), v); err != nil {
		f.t.Fatalf("runJSON(%v): invalid JSON: %v\nstdout=%s", args, err, res.stdout)
	}
	return res
}

// runJSONErr is the goroutine-safe form of runJSON: it RETURNS an error
// instead of calling t.Fatalf.
//
// testing.T.Fatalf must only be called from the goroutine running the test.
// From a spawned goroutine it calls runtime.Goexit(), which kills that
// goroutine without failing the test — so a concurrent step could fail and
// the test would still report PASS. Every concurrent helper in this suite
// must therefore use this form and surface the error from the main goroutine.
func (f *fixture) runJSONErr(v any, args ...string) error {
	res := f.run(append([]string{"--json"}, args...)...)
	if strings.TrimSpace(res.stdout) == "" {
		return fmt.Errorf("awis %v: empty stdout (exit=%d stderr=%s)", args, res.exitCode, res.stderr)
	}
	if err := json.Unmarshal([]byte(res.stdout), v); err != nil {
		return fmt.Errorf("awis %v: invalid JSON: %w (stdout=%s)", args, err, res.stdout)
	}
	return nil
}

// ── background runtime lifecycle ────────────────────────────────────────────

const readyTimeout = 10 * time.Second

// start launches 'awis start' in the background, in its own process group
// (so kill() can sweep any stray children), and blocks until
// <project>/.awis/awis.pid AND runtime.db both exist — never a bare sleep —
// or until the process exits early, or readyTimeout elapses.
func (f *fixture) start() {
	f.t.Helper()
	if f.proc != nil {
		f.t.Fatalf("start: runtime already running")
	}

	cmd := exec.Command(awisBinPath, "start")
	cmd.Dir = f.dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		f.t.Fatalf("start: launch 'awis start': %v", err)
	}

	f.proc = cmd
	f.done = make(chan struct{})
	go func() {
		f.werr = cmd.Wait()
		close(f.done)
	}()

	pidPath := filepath.Join(f.dataDir, "awis.pid")
	dbPath := filepath.Join(f.dataDir, "runtime.db")
	deadline := time.Now().Add(readyTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-f.done:
			f.t.Fatalf("start: 'awis start' exited early (err=%v)\nstdout=%s\nstderr=%s", f.werr, stdout.String(), stderr.String())
		default:
		}
		if fileExists(pidPath) && fileExists(dbPath) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	f.t.Fatalf("start: not ready within %s (pid=%s db=%s)\nstdout=%s\nstderr=%s", readyTimeout, pidPath, dbPath, stdout.String(), stderr.String())
}

// stop sends SIGTERM and waits (bounded) for the background process to exit
// gracefully.
func (f *fixture) stop() {
	f.t.Helper()
	f.signalAndWait(syscall.SIGTERM, 15*time.Second, "stop")
}

// kill sends SIGKILL to the whole process group and waits (bounded) for
// exit — the hard-stop counterpart to stop().
func (f *fixture) kill() {
	f.t.Helper()
	if f.proc == nil {
		return
	}
	_ = syscall.Kill(-f.proc.Process.Pid, syscall.SIGKILL)
	f.waitExit(5*time.Second, "kill")
}

// restart stops then starts the runtime again against the same data dir.
func (f *fixture) restart() {
	f.t.Helper()
	f.stop()
	f.start()
}

func (f *fixture) signalAndWait(sig syscall.Signal, timeout time.Duration, who string) {
	if f.proc == nil {
		f.t.Fatalf("%s: runtime not running", who)
	}
	if err := f.proc.Process.Signal(sig); err != nil {
		f.t.Fatalf("%s: signal %v: %v", who, sig, err)
	}
	f.waitExit(timeout, who)
}

func (f *fixture) waitExit(timeout time.Duration, who string) {
	select {
	case <-f.done:
		f.proc = nil
	case <-time.After(timeout):
		f.t.Fatalf("%s: process did not exit within %s", who, timeout)
	}
}

// cleanup kills any surviving background process. Registered via
// t.Cleanup so it runs even when the test fails or panics.
func (f *fixture) cleanup() {
	if f.proc == nil {
		return
	}
	_ = syscall.Kill(-f.proc.Process.Pid, syscall.SIGKILL)
	select {
	case <-f.done:
	case <-time.After(5 * time.Second):
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ── polling ──────────────────────────────────────────────────────────────────

// statusDetailJSON mirrors the fields of 'awis --json status <id>' this
// harness needs (TDS-07 §4 status JSON schema, per-instance shape).
type statusDetailJSON struct {
	InstanceID  string  `json:"instance_id"`
	Status      string  `json:"status"`
	CurrentStep *string `json:"current_step"`
}

// String renders the status readably. Without it, %v on this struct prints
// CurrentStep as a POINTER ADDRESS, which is worse than useless in a failure
// message: "current_step = 0x2f1a42010de0" tells you nothing about whether
// the value was wrong or merely absent.
func (s statusDetailJSON) String() string {
	return fmt.Sprintf("{instance=%s status=%s current_step=%s}",
		s.InstanceID, s.Status, stepOrNone(s.CurrentStep))
}

// stepOrNone dereferences an optional step id for display, distinguishing a
// nil pointer (no current step) from a pointer to the empty string.
func stepOrNone(p *string) string {
	if p == nil {
		return "<none>"
	}
	return strconv.Quote(*p)
}

// waitFor polls 'awis --json status <instanceID>' until pred is satisfied or
// timeout elapses, failing the test with desc and the last observed state.
//
// Polling on a PREDICATE rather than only on Status matters: several
// interesting transitions do not change the status at all. An instance
// parked on signal "go" and the same instance parked on the next signal
// "done" are both 'waiting', so a test that polls for "waiting" after
// delivering a signal returns instantly on the PRE-signal state and asserts
// against it — passing or failing for reasons unrelated to the signal.
func (f *fixture) waitFor(instanceID, desc string, timeout time.Duration, pred func(statusDetailJSON) bool) statusDetailJSON {
	f.t.Helper()
	deadline := time.Now().Add(timeout)
	var last statusDetailJSON
	for time.Now().Before(deadline) {
		res := f.run("--json", "status", instanceID)
		if res.exitCode == 0 && strings.TrimSpace(res.stdout) != "" {
			var out statusDetailJSON
			if err := json.Unmarshal([]byte(res.stdout), &out); err == nil {
				last = out
				if pred(out) {
					return out
				}
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	f.t.Fatalf("waitFor(%s): timed out after %s waiting for %s; last observed %s",
		instanceID, timeout, desc, last)
	return last
}

// waitForStatus polls until Status == want.
func (f *fixture) waitForStatus(instanceID, want string, timeout time.Duration) statusDetailJSON {
	f.t.Helper()
	return f.waitFor(instanceID, "status "+strconv.Quote(want), timeout, func(s statusDetailJSON) bool {
		return s.Status == want
	})
}

// waitForStep polls until Status == wantStatus AND CurrentStep == wantStep.
// Use this instead of waitForStatus whenever the transition under test moves
// an instance BETWEEN two states that share a status — most importantly one
// signal wait to the next, where both are 'waiting'.
func (f *fixture) waitForStep(instanceID, wantStatus, wantStep string, timeout time.Duration) statusDetailJSON {
	f.t.Helper()
	desc := "status " + strconv.Quote(wantStatus) + " on step " + strconv.Quote(wantStep)
	return f.waitFor(instanceID, desc, timeout, func(s statusDetailJSON) bool {
		return s.Status == wantStatus && s.CurrentStep != nil && *s.CurrentStep == wantStep
	})
}
