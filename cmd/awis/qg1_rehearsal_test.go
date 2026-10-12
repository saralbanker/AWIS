package main

// qg1_rehearsal_test.go — M18-C2r formal QG-1 rehearsal (PRD §31 QG-1):
//
//	awis init → awis start → awis submit hello-world → awis trace <id>
//
// succeeds on a COMPLETING instance, timed against the 5-minute budget.
//
// The test runs the shipped awis binary, built from ./cmd/awis with no extra
// handlers compiled in and no source changes. The binary registers the scaffold's
// examples.hello.greet and examples.hello.log handlers when `awis start` runs (B-9,
// cmd/awis/start.go), so the hello-world instance is expected to reach `completed`.
// M17's TestSystemRehearsalInitStartSubmitTrace accepts any terminal status and ends
// `failed` by design. This test does not: it asserts `completed`.
//
// The timed path starts at `awis init` and ends when `awis trace` returns. The binary
// build is a precondition and is not timed.
//
// The test is an ordinary test: no build tag, no environment gate, no skip. Every awis
// process runs under qg1ChildEnv, with HOME and the XDG base directories inside the
// test's own temp tree, so nothing comes from the developer's ~/.awis or other global
// state. The runtime daemon is stopped on every exit path, and every temp directory is
// removed by t.TempDir.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// qg1Budget is the PRD §31 QG-1 time budget: at most 5 minutes.
const qg1Budget = 5 * time.Minute

// qg1Steps are the scaffolded hello-world steps (cmd/awis/scaffold/workflows/hello-world.yaml).
// A completed trace must show each one as StepCompleted.
var qg1Steps = []string{"greet", "log"}

// qg1Terminal is the set of instance statuses that end polling. It is the same set
// the M17 rehearsal polls for.
var qg1Terminal = map[string]bool{
	string(core.InstanceStatusCompleted):          true,
	string(core.InstanceStatusFailed):             true,
	string(core.InstanceStatusCancelled):          true,
	string(core.InstanceStatusCompensated):        true,
	string(core.InstanceStatusCompensationFailed): true,
}

// qg1ChildEnv is the environment for every awis process the rehearsal starts: PATH,
// plus HOME and the XDG base directories under home (the test's own temp dir). Every
// other inherited variable is dropped, including ANTHROPIC_API_KEY, so a developer's
// profile or API key cannot change the run.
func qg1ChildEnv(home string) []string {
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME=" + filepath.Join(home, ".local", "state"),
		"XDG_CACHE_HOME=" + filepath.Join(home, ".cache"),
	}
}

// qg1HasStepCompleted reports whether some trace line names both the StepCompleted
// event and step as whitespace-separated tokens.
func qg1HasStepCompleted(trace, step string) bool {
	for _, line := range strings.Split(trace, "\n") {
		hasEvent, hasStep := false, false
		for _, field := range strings.Fields(line) {
			if field == "StepCompleted" {
				hasEvent = true
			}
			if field == step {
				hasStep = true
			}
		}
		if hasEvent && hasStep {
			return true
		}
	}
	return false
}

// qg1TraceProblems returns one message per failed QG-1 assertion on the human-mode
// `awis trace` output for instanceID. An empty result means the trace reports a
// completed hello-world run: the header says completed, WorkflowStarted and
// WorkflowCompleted both appear, and each step in steps has a StepCompleted line.
func qg1TraceProblems(trace, instanceID string, steps []string) []string {
	var problems []string
	if !strings.Contains(trace, "Status: completed") {
		problems = append(problems, "trace header does not report \"Status: completed\"")
	}
	if !strings.Contains(trace, "hello-world") {
		problems = append(problems, "trace does not name workflow hello-world")
	}
	if !strings.Contains(trace, instanceID) {
		problems = append(problems, "trace does not name instance "+instanceID)
	}
	for _, event := range []string{"WorkflowStarted", "WorkflowCompleted"} {
		if !strings.Contains(trace, event) {
			problems = append(problems, "trace timeline has no "+event)
		}
	}
	for _, step := range steps {
		if !qg1HasStepCompleted(trace, step) {
			problems = append(problems, "trace timeline has no StepCompleted line for step "+step)
		}
	}
	return problems
}

// qg1StopRuntime registers, as t cleanup, the stop of the runtime daemon that
// `awis start` launched: `awis stop`, then up to 12s for the process to exit, then a
// kill. Cleanup runs on pass, fail, and Fatal alike, so no runtime outlives the test.
//
// exited receives the daemon's Wait result. The caller starts the goroutine that
// produces it right after Start, so the child is reaped as soon as it exits. An
// unreaped child is a zombie that still answers kill(pid, 0), which makes `awis stop`
// poll until its own 10s timeout before it reports the runtime stopped.
func qg1StopRuntime(t *testing.T, bin, dataDir, projDir string, env []string, daemon *exec.Cmd, exited <-chan error) {
	t.Helper()
	t.Cleanup(func() {
		stop := exec.Command(bin, "--data-dir="+dataDir, "stop")
		stop.Dir = projDir
		stop.Env = env
		if out, err := stop.CombinedOutput(); err != nil {
			t.Logf("awis stop: %v\n%s", err, out)
		}
		select {
		case err := <-exited:
			if err != nil {
				t.Logf("awis start exited: %v", err)
			}
		case <-time.After(12 * time.Second):
			if err := daemon.Process.Kill(); err != nil {
				t.Logf("kill awis start: %v", err)
			}
			select {
			case err := <-exited:
				t.Logf("awis start killed after stop timeout: %v", err)
			case <-time.After(5 * time.Second):
				t.Logf("awis start did not exit after kill")
			}
		}
	})
}

// qg1AwaitTerminal polls `awis status --all --json` until instanceID appears in the
// terminal "recent" list, and returns its status. The poll contract is M17's: it polls
// status, not trace, so polling does not touch the trace payload path. It fails t if
// the QG-1 deadline passes first, with the same diagnostics the M17 rehearsal captures.
func qg1AwaitTerminal(t *testing.T, bin, dataDir, projDir string, env []string, instanceID string, deadline time.Time, logCap *daemonLogCapture) string {
	t.Helper()
	for polls := 1; ; polls++ {
		statusCmd := exec.Command(bin, "--data-dir="+dataDir, "--json", "status", "--all")
		statusCmd.Dir = projDir
		statusCmd.Env = env
		var statusStderr strings.Builder
		statusCmd.Stderr = &statusStderr
		statusOut, err := statusCmd.Output()
		if err != nil {
			t.Fatalf("awis status failed: %v\nstderr: %s", err, statusStderr.String())
		}
		var status statusOutputJSON
		if err := json.Unmarshal(statusOut, &status); err != nil {
			t.Fatalf("status --json output is not valid JSON: %v\nstdout: %s", err, statusOut)
		}
		for _, r := range status.Recent {
			if r.InstanceID == instanceID && qg1Terminal[r.Status] {
				return r.Status
			}
		}
		if !time.Now().Before(deadline) {
			t.Fatalf("instance %s not terminal within the QG-1 budget %s after %d polls\ndiagnostics:\n%s",
				instanceID, qg1Budget, polls, captureRehearsalDiagnostics(bin, dataDir, projDir, instanceID, logCap))
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// TestQG1RehearsalCompletesHelloWorld runs the QG-1 path on the shipped binary and
// asserts that the hello-world instance completes within the 5-minute budget.
func TestQG1RehearsalCompletesHelloWorld(t *testing.T) {
	// The build is a precondition, not part of the timed QG-1 path.
	bin := buildBinary(t, t.TempDir())

	home := t.TempDir()
	env := qg1ChildEnv(home)
	projDir := t.TempDir()
	dataDir := filepath.Join(projDir, ".awis")

	start := time.Now()
	deadline := start.Add(qg1Budget)

	// 1. awis init <projDir>: scaffold a fresh project.
	initCmd := exec.Command(bin, "init", projDir)
	initCmd.Env = env
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("awis init failed: %v\noutput: %s", err, out)
	}

	// 2. awis start: the runtime is a daemon. Its pipe is drained for the life of the
	// test (D-15), and it is stopped on every exit path once registered.
	startCmd := exec.Command(bin, "--data-dir="+dataDir, "start")
	startCmd.Dir = projDir
	startCmd.Env = env
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	startCmd.Stdout = pw
	startCmd.Stderr = pw
	if err := startCmd.Start(); err != nil {
		if cerr := pr.Close(); cerr != nil {
			t.Logf("close daemon pipe reader: %v", cerr)
		}
		if cerr := pw.Close(); cerr != nil {
			t.Logf("close daemon pipe writer: %v", cerr)
		}
		t.Fatalf("start runtime: %v", err)
	}
	if err := pw.Close(); err != nil {
		t.Logf("close daemon pipe writer: %v", err)
	}
	logCap := startDaemonLogCapture(pr)
	exited := make(chan error, 1)
	go func() { exited <- startCmd.Wait() }()
	qg1StopRuntime(t, bin, dataDir, projDir, env, startCmd, exited)
	if !waitForHeader(t, logCap, 15*time.Second) {
		t.Fatalf("runtime did not print its startup header within 15s\ndaemon log:\n%s", logCap.String())
	}

	// 3. awis submit hello-world. --json is only for parsing the instance id. The
	// command is otherwise the card's literal form, with no namespace flag.
	submitCmd := exec.Command(bin, "--data-dir="+dataDir, "--json", "submit", "hello-world")
	submitCmd.Dir = projDir
	submitCmd.Env = env
	var submitStderr strings.Builder
	submitCmd.Stderr = &submitStderr
	submitOut, err := submitCmd.Output()
	if err != nil {
		t.Fatalf("awis submit hello-world failed: %v\nstderr: %s", err, submitStderr.String())
	}
	var submitted submitOutput
	if err := json.Unmarshal(submitOut, &submitted); err != nil {
		t.Fatalf("submit --json output is not valid JSON: %v\nstdout: %s\nstderr: %s", err, submitOut, submitStderr.String())
	}
	if submitted.InstanceID == "" {
		t.Fatalf("submit --json output has no instance_id: %s", submitOut)
	}
	t.Logf("submitted instance %s (workflow %s)", submitted.InstanceID, submitted.WorkflowID)

	// 4. Poll until the instance is terminal, and require it to be completed.
	finalStatus := qg1AwaitTerminal(t, bin, dataDir, projDir, env, submitted.InstanceID, deadline, logCap)
	if finalStatus != string(core.InstanceStatusCompleted) {
		t.Fatalf("instance %s finished with status %q; QG-1 requires %q\ndiagnostics:\n%s",
			submitted.InstanceID, finalStatus, core.InstanceStatusCompleted,
			captureRehearsalDiagnostics(bin, dataDir, projDir, submitted.InstanceID, logCap))
	}

	// 5. awis trace <id> (human mode, stdout). The timed path ends here.
	traceCmd := exec.Command(bin, "--data-dir="+dataDir, "trace", submitted.InstanceID)
	traceCmd.Dir = projDir
	traceCmd.Env = env
	var traceStderr strings.Builder
	traceCmd.Stderr = &traceStderr
	traceOut, err := traceCmd.Output()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("awis trace %s failed: %v\nstderr: %s", submitted.InstanceID, err, traceStderr.String())
	}
	trace := string(traceOut)
	t.Logf("trace output:\n%s", trace)
	for _, problem := range qg1TraceProblems(trace, submitted.InstanceID, qg1Steps) {
		t.Error(problem)
	}
	if elapsed > qg1Budget {
		t.Errorf("QG-1 path took %s, over the %s budget", elapsed, qg1Budget)
	}
	t.Logf("QG-1 wall-clock: %s (budget %s; awis init -> awis start -> awis submit hello-world -> awis trace %s; status %s)",
		elapsed.Round(time.Millisecond), qg1Budget, submitted.InstanceID, finalStatus)
}

// TestQG1TraceProblemsAcceptsCompletedAndRejectsGaps pins the trace assertions: a
// complete hello-world trace has no problems, and a failed header, a missing
// StepCompleted, or a missing terminal event each produce one.
func TestQG1TraceProblemsAcceptsCompletedAndRejectsGaps(t *testing.T) {
	const id = "33556155-53ee-4a73-b7b9-f3272bc91339"
	completeLines := []string{
		"Trace: hello-world / " + id,
		"Status: completed ✓  Duration: 0s",
		"",
		"Timeline:",
		"  00:00  ● WorkflowStarted",
		"  00:00  ► StepStarted          greet (attempt 1)",
		"  00:00  ✓ StepCompleted        greet                 0.0s",
		"  00:00  ► StepStarted          log (attempt 1)",
		"  00:00  ✓ StepCompleted        log                   0.0s",
		"  00:00  ✓ WorkflowCompleted     total: 0s",
	}
	complete := strings.Join(completeLines, "\n") + "\n"
	if problems := qg1TraceProblems(complete, id, qg1Steps); len(problems) != 0 {
		t.Fatalf("complete trace: unexpected problems %v", problems)
	}

	failedHeader := strings.Replace(complete, "Status: completed", "Status: failed", 1)
	if problems := qg1TraceProblems(failedHeader, id, qg1Steps); len(problems) == 0 {
		t.Error("failed header: want a problem, got none")
	}

	noLogCompleted := strings.Replace(complete, "✓ StepCompleted        log", "► StepStarted          log", 1)
	if problems := qg1TraceProblems(noLogCompleted, id, qg1Steps); len(problems) != 1 {
		t.Errorf("log step not completed: want exactly one problem, got %v", problems)
	}

	noTerminal := strings.Replace(complete, "WorkflowCompleted", "WorkflowRunning", 1)
	if problems := qg1TraceProblems(noTerminal, id, qg1Steps); len(problems) != 1 {
		t.Errorf("no WorkflowCompleted: want exactly one problem, got %v", problems)
	}
}

// TestQG1ChildEnvIsolatesFromProfileAndAPIKey pins the isolation rule: the child
// environment points HOME at the test's own directory and drops the developer's HOME
// and ANTHROPIC_API_KEY.
func TestQG1ChildEnvIsolatesFromProfileAndAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "must-not-leak")
	t.Setenv("HOME", "/developer/home")
	home := t.TempDir()
	env := qg1ChildEnv(home)

	wantHome := "HOME=" + home
	sawHome := false
	for _, kv := range env {
		if strings.HasPrefix(kv, "ANTHROPIC_API_KEY=") {
			t.Errorf("child env leaks ANTHROPIC_API_KEY: %q", kv)
		}
		if strings.Contains(kv, "/developer/home") {
			t.Errorf("child env leaks the developer HOME: %q", kv)
		}
		if kv == wantHome {
			sawHome = true
		}
	}
	if !sawHome {
		t.Errorf("child env has no %q entry; env = %v", wantHome, env)
	}
}
