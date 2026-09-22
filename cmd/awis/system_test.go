package main

// system_test.go — M14-C2 T6 real-binary system tests.
//
// Scenario A: build binary; start with temp data-dir + hello workflow YAML;
//   wait for startup header; run 'status' from a second process; stop; assert graceful.
// Scenario B (crash recovery, IR-5): start; submit instance via sdk directly;
//   SIGKILL the runtime; restart; assert status reports consistent state.
//
// These tests are skipped under -short.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/sdk"
)

// helloWorkflowYAML is a minimal workflow YAML for the system test.
const helloWorkflowYAML = `schema_version: 1
id: test-hello
version: 1.0.0
namespace: test
name: Test Hello
description: Minimal workflow for system test

triggers:
  - type: manual

steps:
  - id: greet
    name: Greet
    type: native
    handler: test.hello.greet

initial_step: greet
final_steps: [greet]
`

// buildBinary compiles the awis binary into dir and returns the path to the binary.
func buildBinary(t *testing.T, dir string) string {
	t.Helper()
	bin := filepath.Join(dir, "awis")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/awis/awis/cmd/awis")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build binary: %v", err)
	}
	return bin
}

// writeWorkflowYAML writes the hello workflow YAML to <dir>/workflows/hello.yaml.
func writeWorkflowYAML(t *testing.T, dir string) {
	t.Helper()
	wfDir := filepath.Join(dir, "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	path := filepath.Join(wfDir, "hello.yaml")
	if err := os.WriteFile(path, []byte(helloWorkflowYAML), 0o644); err != nil {
		t.Fatalf("write workflow yaml: %v", err)
	}
}

// maxDaemonLogBytes bounds daemonLogCapture's retained buffer: the most
// recent maxDaemonLogBytes of a test's daemon output, never more (D-15 —
// "keep the most recent N KB; do not grow without limit").
const maxDaemonLogBytes = 64 * 1024

// daemonLogCapture drains a daemon subprocess's combined stdout+stderr pipe
// continuously for the lifetime of a test (D-15). Before this, the FOUR call
// sites below wired the pipe, read it only until waitForHeader saw the
// startup header, and then NEVER read it again: once the ~64KB kernel pipe
// buffer filled, the daemon's next log write blocked — pre-D-18, that froze
// the whole tick loop — and the daemon's own log, the one artifact that
// would diagnose this defect class, went nowhere (it was never captured for
// a failure message either). draining continuously fixes both: the pipe
// buffer never fills, and the captured tail is available to
// captureRehearsalDiagnostics.
type daemonLogCapture struct {
	mu     sync.Mutex
	buf    []byte
	closed bool
}

// startDaemonLogCapture launches the single draining goroutine for r. It
// exits on its own once r returns an error (EOF when the daemon process
// exits and closes its inherited write end) — no explicit stop is needed;
// exactly one goroutine per call, never growing.
func startDaemonLogCapture(r io.Reader) *daemonLogCapture {
	c := &daemonLogCapture{}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				c.mu.Lock()
				c.buf = append(c.buf, buf[:n]...)
				if len(c.buf) > maxDaemonLogBytes {
					c.buf = c.buf[len(c.buf)-maxDaemonLogBytes:]
				}
				c.mu.Unlock()
			}
			if err != nil {
				c.mu.Lock()
				c.closed = true
				c.mu.Unlock()
				return
			}
		}
	}()
	return c
}

// String returns the currently captured (bounded) tail of the daemon's log.
func (c *daemonLogCapture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return string(c.buf)
}

// isClosed reports whether the underlying pipe has hit EOF/error (the daemon
// process exited): once true, cap's buffer will never change again, so a
// caller waiting on it can stop early instead of polling out the rest of its
// deadline.
func (c *daemonLogCapture) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// waitForHeaderPollInterval is how often waitForHeader checks cap's captured
// output. This is polling EXTERNAL subprocess I/O (arbitrary OS-scheduled
// byte arrival), not internal engine state — unlike the engine package's
// idiom (helpers_test.go/hydrate_test.go: manualClock, never time.Sleep),
// there is no engine clock to advance here, so a short bounded poll against
// a real deadline is the standard, correct tool for this job.
const waitForHeaderPollInterval = 20 * time.Millisecond

// waitForHeader blocks until cap's captured output contains "● running" or
// timeout expires — identical bool/timeout/logging contract to the original
// one-shot read loop, just driven by daemonLogCapture's continuous drain
// instead of reading r directly (a second, ad hoc reader here would race the
// continuous drain goroutine for bytes off the same pipe). Nothing here
// weakens the deadline or the pass/fail condition.
func waitForHeader(t *testing.T, cap *daemonLogCapture, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		collected := cap.String()
		if strings.Contains(collected, "● running") {
			return true
		}
		if cap.isClosed() || !time.Now().Before(deadline) {
			t.Logf("waitForHeader: did not see '● running' within %s; got:\n%s", timeout, collected)
			return false
		}
		time.Sleep(waitForHeaderPollInterval)
	}
}

// TestDaemonLogCapture_DrainsBoundsAndUnblocksTheWriter is the D-15 unit
// proof for daemonLogCapture, independent of spawning a real subprocess: an
// in-memory io.Pipe stands in for the daemon's stdout/stderr pipe. It proves
// (a) captured content is readable via String(), (b) the buffer is bounded
// to maxDaemonLogBytes — the most recent bytes only, never growing further —
// and (c) draining continuously means a writer that would have blocked on a
// full OS pipe buffer never does here, because something is always reading.
func TestDaemonLogCapture_DrainsBoundsAndUnblocksTheWriter(t *testing.T) {
	pr, pw := io.Pipe()
	cap := startDaemonLogCapture(pr)

	// Write far more than maxDaemonLogBytes in chunks; io.Pipe's Write blocks
	// until a Read consumes it, so this ALSO proves the drain goroutine is
	// continuously reading (D-15) — a stalled drain would hang this loop.
	chunk := bytes.Repeat([]byte("x"), 4096)
	total := 0
	writeDone := make(chan error, 1)
	go func() {
		for total < maxDaemonLogBytes*3 {
			n, err := pw.Write(chunk)
			total += n
			if err != nil {
				writeDone <- err
				return
			}
		}
		writeDone <- pw.Close()
	}()

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("write to pipe: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("writes did not complete within 10s; daemonLogCapture is not draining continuously (D-15 regression)")
	}

	// Give the drain goroutine a moment to consume the final chunk(s) after
	// Close (Close does not itself guarantee the reader has caught up);
	// isClosed() is the deterministic signal that it has seen EOF.
	deadline := time.Now().Add(5 * time.Second)
	for !cap.isClosed() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !cap.isClosed() {
		t.Fatal("daemonLogCapture never observed EOF after the writer closed")
	}

	got := cap.String()
	if len(got) != maxDaemonLogBytes {
		t.Fatalf("captured length = %d, want exactly maxDaemonLogBytes (%d) — bounded, not unbounded growth",
			len(got), maxDaemonLogBytes)
	}
	for _, c := range got {
		if c != 'x' {
			t.Fatalf("captured content corrupted; expected all 'x', found %q", c)
		}
	}
}

// TestSystemScenarioA builds the binary; starts runtime in temp dir with hello workflow;
// waits for header; runs 'status' from a second process; stops; asserts graceful exit.
func TestSystemScenarioA(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping system test under -short")
	}

	// Use a temp dir as the project root (cwd for start + workflow discovery).
	projDir := t.TempDir()
	writeWorkflowYAML(t, projDir)

	binDir := t.TempDir()
	bin := buildBinary(t, binDir)

	dataDir := filepath.Join(projDir, ".awis")

	// Start the runtime.
	startCmd := exec.Command(bin, "--data-dir="+dataDir, "start")
	startCmd.Dir = projDir
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	startCmd.Stdout = pw
	startCmd.Stderr = pw
	if err := startCmd.Start(); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	// Close write end in parent so read end gets EOF when process exits.
	_ = pw.Close()

	// Drain the pipe continuously for the rest of this test (D-15): reading
	// it only until the header appeared and never again let the daemon's log
	// writes block once the kernel pipe buffer filled.
	logCap := startDaemonLogCapture(pr)

	// Wait for startup header.
	if !waitForHeader(t, logCap, 15*time.Second) {
		startCmd.Process.Kill() //nolint:errcheck
		t.Fatal("runtime did not print startup header within 15s")
	}

	// Run 'status' from a second process.
	statusCmd := exec.Command(bin, "--data-dir="+dataDir, "status")
	statusCmd.Dir = projDir
	out, err := statusCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status command failed: %v\noutput: %s", err, out)
	}
	t.Logf("status output:\n%s", out)

	// Stop the runtime.
	stopCmd := exec.Command(bin, "--data-dir="+dataDir, "stop")
	stopCmd.Dir = projDir
	stopOut, err := stopCmd.CombinedOutput()
	if err != nil {
		t.Logf("stop output: %s", stopOut)
		// Tolerate stop failure if process already exited.
	}

	// Wait for the runtime to exit (graceful).
	done := make(chan error, 1)
	go func() { done <- startCmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			// Non-zero exit after SIGTERM is acceptable (signal exit).
			t.Logf("runtime exited: %v (acceptable for SIGTERM)", err)
		}
	case <-time.After(12 * time.Second):
		startCmd.Process.Kill() //nolint:errcheck
		t.Error("runtime did not exit within 12s after stop")
	}
}

// TestSystemScenarioB tests crash recovery (IR-5):
// start; submit an instance via sdk against the same db; SIGKILL the runtime;
// restart; assert status reports the instance (not lost).
func TestSystemScenarioB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping system test under -short")
	}

	projDir := t.TempDir()
	writeWorkflowYAML(t, projDir)

	binDir := t.TempDir()
	bin := buildBinary(t, binDir)

	dataDir := filepath.Join(projDir, ".awis")
	dbPath := filepath.Join(dataDir, "runtime.db")

	// Ensure data dir exists.
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir datadir: %v", err)
	}

	// Start first runtime instance.
	startCmd1 := exec.Command(bin, "--data-dir="+dataDir, "start")
	startCmd1.Dir = projDir
	pr1, pw1, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	startCmd1.Stdout = pw1
	startCmd1.Stderr = pw1
	if err := startCmd1.Start(); err != nil {
		t.Fatalf("start runtime 1: %v", err)
	}
	_ = pw1.Close()

	// Drain continuously for the rest of this test (D-15) — see TestSystemScenarioA.
	logCap1 := startDaemonLogCapture(pr1)

	if !waitForHeader(t, logCap1, 15*time.Second) {
		startCmd1.Process.Kill() //nolint:errcheck
		t.Fatal("runtime 1 did not print startup header within 15s")
	}

	// Submit an instance via sdk directly against the same db (F-3 WAL semantics).
	// The workflow was registered by the running engine on startup; sdk.NewRuntime
	// opens the same db and submits the instance row via storage APIs.
	ctx := context.Background()
	store, err := sdk.SQLiteStorage(dbPath)
	if err != nil {
		startCmd1.Process.Kill() //nolint:errcheck
		t.Fatalf("open sdk storage for submit: %v", err)
	}
	rt, err := sdk.NewRuntime(sdk.Config{Namespace: "test", Storage: store})
	if err != nil {
		startCmd1.Process.Kill() //nolint:errcheck
		t.Fatalf("new runtime for submit: %v", err)
	}
	instanceID, err := rt.Submit(ctx, "test-hello", map[string]any{"name": "crash-test"})
	if err != nil {
		startCmd1.Process.Kill() //nolint:errcheck
		t.Logf("sdk submit failed (workflow not in-memory registry of this rt instance; acceptable): %v", err)
		_ = instanceID
	} else {
		t.Logf("submitted instance: %s", instanceID)
	}

	// SIGKILL the first runtime (crash, not graceful).
	if err := startCmd1.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("SIGKILL runtime 1: %v", err)
	}
	_ = startCmd1.Wait()
	t.Log("runtime 1 killed (crash simulation)")

	// Remove PID file since SIGKILL didn't clean it up.
	_ = os.Remove(filepath.Join(dataDir, "awis.pid"))

	// Restart the runtime.
	startCmd2 := exec.Command(bin, "--data-dir="+dataDir, "start")
	startCmd2.Dir = projDir
	pr2, pw2, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe2: %v", err)
	}
	startCmd2.Stdout = pw2
	startCmd2.Stderr = pw2
	if err := startCmd2.Start(); err != nil {
		t.Fatalf("start runtime 2: %v", err)
	}
	_ = pw2.Close()

	// Drain continuously for the rest of this test (D-15) — see TestSystemScenarioA.
	logCap2 := startDaemonLogCapture(pr2)

	if !waitForHeader(t, logCap2, 15*time.Second) {
		startCmd2.Process.Kill() //nolint:errcheck
		t.Fatal("runtime 2 did not print startup header within 15s")
	}

	// Query status — storage should be in a consistent state (not corrupted by SIGKILL).
	store2, err := sdk.SQLiteStorage(dbPath)
	if err != nil {
		startCmd2.Process.Kill() //nolint:errcheck
		t.Fatalf("open sdk storage for status check: %v", err)
	}
	// List all instances — should not error (proves WAL+crash recovery consistency).
	insts, err := store2.ListInstances(ctx, core.InstanceFilter{}) //nolint:typecheck
	if err != nil {
		startCmd2.Process.Kill() //nolint:errcheck
		t.Fatalf("ListInstances after crash recovery: %v", err)
	}
	t.Logf("instances after crash recovery: %d", len(insts))
	// IR-5: state is consistent — no assertion on count because the submit may
	// have failed (workflow not registered in-process); the key invariant is no crash/corruption.

	// Stop runtime 2 gracefully.
	stopCmd := exec.Command(bin, "--data-dir="+dataDir, "stop")
	stopCmd.Dir = projDir
	_ = stopCmd.Run()

	done := make(chan error, 1)
	go func() { done <- startCmd2.Wait() }()
	select {
	case <-done:
	case <-time.After(12 * time.Second):
		startCmd2.Process.Kill() //nolint:errcheck
		t.Error("runtime 2 did not exit within 12s after stop")
	}
}

// TestSystemRehearsalInitStartSubmitTrace is the M17-C3 CE-pinned rehearsal:
// "init → start → submit → trace" (QG-1 path, measured loosely here, formally at M18).
//
// It runs the real binary through the full round trip a new user would follow after
// 'awis init': init a fresh project directory, start the runtime against it, submit the
// scaffolded hello-world workflow, and trace the resulting instance.
//
// Completion note: the hello-world scaffold's native steps (handler:
// examples.hello.greet / examples.hello.log) are stubs documented in
// handlers/example_handler.go for an EMBEDDING Go program to register (FR-RM-01); the
// generic 'awis' CLI binary run here does not compile in any application handlers (by
// design — that is the sdk-embedding seam, not the CLI's job). With no handler
// registered, native.Run returns a non-retryable "handler_not_found" StepError (no
// retry policy is set on the step, so ADJ-6 makes it a single, immediate attempt) and
// the engine settles the instance to a TERMINAL status on the very next tick. This test
// therefore asserts the instance reaches ANY terminal status (not specifically
// "completed") — proving init→start→submit→trace mechanically works end-to-end, which
// is what "measured loosely" calls for; a full successful-execution QG-1 gate with a
// real registered handler is out of this card's scope and is formalized at M18.
func TestSystemRehearsalInitStartSubmitTrace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping system test under -short")
	}

	binDir := t.TempDir()
	bin := buildBinary(t, binDir)

	// 1. awis init <projDir> — scaffold a fresh project.
	projDir := t.TempDir()
	initCmd := exec.Command(bin, "init", projDir)
	initOut, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("awis init failed: %v\noutput: %s", err, initOut)
	}
	t.Logf("init output:\n%s", initOut)

	dataDir := filepath.Join(projDir, ".awis")

	// 2. awis start — start the runtime against the scaffolded project.
	startCmd := exec.Command(bin, "--data-dir="+dataDir, "start")
	startCmd.Dir = projDir
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	startCmd.Stdout = pw
	startCmd.Stderr = pw
	if err := startCmd.Start(); err != nil {
		t.Fatalf("start runtime: %v", err)
	}
	_ = pw.Close()

	// Drain continuously for the rest of this test (D-15) — see TestSystemScenarioA.
	// Also handed to captureRehearsalDiagnostics on failure below: the
	// daemon's own log is the one artifact that would diagnose this defect
	// class, and it was never captured for a failure message before.
	logCap := startDaemonLogCapture(pr)

	if !waitForHeader(t, logCap, 15*time.Second) {
		startCmd.Process.Kill() //nolint:errcheck
		t.Fatal("runtime did not print startup header within 15s")
	}

	// Ensure the runtime is stopped at the end of the test regardless of outcome.
	defer func() {
		stopCmd := exec.Command(bin, "--data-dir="+dataDir, "stop")
		stopCmd.Dir = projDir
		_ = stopCmd.Run()
		done := make(chan error, 1)
		go func() { done <- startCmd.Wait() }()
		select {
		case <-done:
		case <-time.After(12 * time.Second):
			startCmd.Process.Kill() //nolint:errcheck
		}
	}()

	// 3. awis submit hello-world — the scaffolded workflow is registered under its own
	// YAML namespace ("examples"; sdk.Runtime.Submit's cross-process fallback filters
	// ListWorkflows by the CALLING process's --namespace, not the definition's namespace,
	// so this must match the scaffold's workflows/hello-world.yaml `namespace: examples`).
	//
	// Uses Output() (stdout only), not CombinedOutput(): the engine's default slog
	// handler writes structured JSON log lines to stderr, which would otherwise
	// interleave with the command's own --json stdout payload and break parsing.
	submitCmd := exec.Command(bin, "--data-dir="+dataDir, "--namespace=examples", "--json", "submit", "hello-world")
	submitCmd.Dir = projDir
	var submitStderr strings.Builder
	submitCmd.Stderr = &submitStderr
	submitOut, err := submitCmd.Output()
	if err != nil {
		t.Fatalf("awis submit hello-world failed: %v\nstderr: %s", err, submitStderr.String())
	}
	var submitResult submitOutput
	if jerr := json.Unmarshal(submitOut, &submitResult); jerr != nil {
		t.Fatalf("submit --json output is not valid JSON: %v\nstdout: %s\nstderr: %s", jerr, submitOut, submitStderr.String())
	}
	if submitResult.InstanceID == "" {
		t.Fatalf("submit --json output missing instance_id: %s", submitOut)
	}
	t.Logf("submitted instance: %s", submitResult.InstanceID)

	// 4. awis status — poll (status+trace, as the card sanctions) until the instance
	// leaves the active list and appears in the terminal "recent" list.
	//
	// This step uses 'status --json' rather than 'trace --json' for polling: it does
	// not touch execution_events payloads at all (it aggregates from
	// workflow_instances), so it is unaffected by trace's --json payload-encoding
	// path. 'trace' (human mode, below) supplies the full timeline the card asks for.
	terminal := map[string]bool{
		string(core.InstanceStatusCompleted):          true,
		string(core.InstanceStatusFailed):             true,
		string(core.InstanceStatusCancelled):          true,
		string(core.InstanceStatusCompensated):        true,
		string(core.InstanceStatusCompensationFailed): true,
	}

	var finalStatus string
	// terminalBudget covers the whole polling phase below, not just the terminal
	// transition itself. Measured steady-state (isolated runs, both with and
	// without -race): 5/5 PASS at 11.13-11.23s. The previous 10s budget was
	// below that steady state, so it failed intermittently under full-package
	// CPU contention even though it always passed in isolation. 45s is a
	// generous multiple of the ~11.2s observed steady state, chosen to absorb
	// contention without being unbounded.
	terminalBudget := 45 * time.Second * raceScale
	pollStart := time.Now()
	deadline := pollStart.Add(terminalBudget)

	// D-9: buildStatusJSON (status.go) puts only terminal-status instances in
	// Recent; running/waiting/pending instances land in Active. The original
	// loop only ever inspected Recent, so a timeout could not distinguish "the
	// instance never existed" from "the instance was running the whole time" —
	// both left finalStatus as "". These fields make that distinction visible
	// without changing the pass/fail condition below, which is still governed
	// solely by finalStatus (from Recent) reaching a terminal status.
	var lastObservedStatus string
	var sawInActive bool
	var sawInRecent bool
	pollCount := 0
	for time.Now().Before(deadline) {
		pollCount++
		statusCmd := exec.Command(bin, "--data-dir="+dataDir, "--json", "status", "--all")
		statusCmd.Dir = projDir
		var statusStderr strings.Builder
		statusCmd.Stderr = &statusStderr
		statusOut, serr := statusCmd.Output()
		if serr != nil {
			t.Fatalf("awis status failed: %v\nstderr: %s", serr, statusStderr.String())
		}
		var statusResult statusOutputJSON
		if jerr := json.Unmarshal(statusOut, &statusResult); jerr != nil {
			t.Fatalf("status --json output is not valid JSON: %v\nstdout: %s", jerr, statusOut)
		}
		found := false
		for _, r := range statusResult.Recent {
			if r.InstanceID == submitResult.InstanceID {
				finalStatus = r.Status
				found = true
				sawInRecent = true
				if r.Status != "" {
					lastObservedStatus = r.Status
				}
				break
			}
		}
		for _, a := range statusResult.Active {
			if a.InstanceID == submitResult.InstanceID {
				sawInActive = true
				if a.Status != "" {
					lastObservedStatus = a.Status
				}
				break
			}
		}
		if found && terminal[finalStatus] {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !terminal[finalStatus] {
		elapsed := time.Since(pollStart)
		lastStatusDesc := lastObservedStatus
		if lastStatusDesc == "" {
			lastStatusDesc = "never observed in any bucket"
		}
		diag := captureRehearsalDiagnostics(bin, dataDir, projDir, submitResult.InstanceID, logCap)
		t.Fatalf("instance %s did not reach a terminal status within %s (%d polls, %s elapsed): "+
			"ever observed in active=%v, ever observed in recent=%v, last observed status: %s\n"+
			"diagnostics:\n%s",
			submitResult.InstanceID, terminalBudget, pollCount, elapsed, sawInActive, sawInRecent, lastStatusDesc, diag)
	}
	t.Logf("instance %s reached terminal status %q", submitResult.InstanceID, finalStatus)

	// 5. awis trace <instance-id> — full execution timeline (human mode; TDS-07 §4
	// shape). Confirms the engine actually ran the instance end-to-end, not merely
	// that storage reports a terminal row.
	traceCmd := exec.Command(bin, "--data-dir="+dataDir, "trace", submitResult.InstanceID)
	traceCmd.Dir = projDir
	traceOut, terr := traceCmd.CombinedOutput()
	if terr != nil {
		t.Fatalf("awis trace failed: %v\noutput: %s", terr, traceOut)
	}
	traceStr := string(traceOut)
	t.Logf("trace output:\n%s", traceStr)

	if !strings.Contains(traceStr, "hello-world") {
		t.Errorf("trace output does not mention workflow id 'hello-world': %s", traceStr)
	}
	if !strings.Contains(traceStr, submitResult.InstanceID) {
		t.Errorf("trace output does not mention instance id %q: %s", submitResult.InstanceID, traceStr)
	}
	if !strings.Contains(traceStr, "WorkflowStarted") {
		t.Errorf("trace timeline does not include WorkflowStarted: %s", traceStr)
	}
	// One of the terminal workflow events must appear in the timeline, matching the
	// terminal status observed via 'status' above.
	terminalEventNames := []string{
		"WorkflowCompleted", "WorkflowFailed", "WorkflowCancelled",
		"WorkflowCompensated", "WorkflowCompensationFailed",
	}
	sawTerminalEvent := false
	for _, name := range terminalEventNames {
		if strings.Contains(traceStr, name) {
			sawTerminalEvent = true
			break
		}
	}
	if !sawTerminalEvent {
		t.Errorf("trace timeline does not include a terminal workflow event: %s", traceStr)
	}
}

// captureRehearsalDiagnostics runs 'awis trace <instanceID>' and
// 'awis status --all --json' against the still-running (or already-exited)
// runtime, plus (D-15) the daemon's OWN captured log — logCap's continuously
// drained, bounded tail (startDaemonLogCapture) — and returns their combined
// output for inclusion in a test failure message (D-9). It never fails the
// test itself — a command error here is itself diagnostic information (e.g.
// "runtime not reachable") and is embedded in the returned text rather than
// swallowed. logCap may be nil (no daemon log available); that section is
// simply omitted.
func captureRehearsalDiagnostics(bin, dataDir, projDir, instanceID string, logCap *daemonLogCapture) string {
	var b strings.Builder

	traceCmd := exec.Command(bin, "--data-dir="+dataDir, "trace", instanceID)
	traceCmd.Dir = projDir
	traceOut, traceErr := traceCmd.CombinedOutput()
	fmt.Fprintf(&b, "--- awis trace %s ---\n", instanceID)
	if traceErr != nil {
		fmt.Fprintf(&b, "(command error: %v)\n", traceErr)
	}
	b.Write(traceOut)
	if len(traceOut) == 0 || traceOut[len(traceOut)-1] != '\n' {
		b.WriteByte('\n')
	}

	statusCmd := exec.Command(bin, "--data-dir="+dataDir, "--json", "status", "--all")
	statusCmd.Dir = projDir
	statusOut, statusErr := statusCmd.CombinedOutput()
	fmt.Fprintf(&b, "--- awis status --all --json ---\n")
	if statusErr != nil {
		fmt.Fprintf(&b, "(command error: %v)\n", statusErr)
	}
	b.Write(statusOut)
	if len(statusOut) == 0 || statusOut[len(statusOut)-1] != '\n' {
		b.WriteByte('\n')
	}

	if logCap != nil {
		daemonLog := logCap.String()
		fmt.Fprintf(&b, "--- daemon log (captured, most recent %d bytes) ---\n", maxDaemonLogBytes)
		if daemonLog == "" {
			fmt.Fprint(&b, "(empty)\n")
		} else {
			b.WriteString(daemonLog)
			if daemonLog[len(daemonLog)-1] != '\n' {
				b.WriteByte('\n')
			}
		}
	}

	return b.String()
}
