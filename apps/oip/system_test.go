// Package oip_test contains the M15-C3 real-binary e2e test.
//
// TestOIPSystemE2E builds the oip and awis binaries, starts an oip runtime in a
// temp dir with copies of the OIP YAML fixtures, installs the git-context-plugin
// (or a stub when python3 is absent), submits capture-decision via the awis CLI,
// signals the workflow through to append, then submits recall-decision and verifies
// the entry id appears in the output.
//
// Skipped under -short.
package oip_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// repoRoot returns the absolute path to the AWIS repository root
// (two levels above apps/oip/).
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	abs, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

// buildOIPBinary compiles the oip binary into dir and returns the path.
func buildOIPBinary(t *testing.T, dir string) string {
	t.Helper()
	bin := filepath.Join(dir, "oip")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/awis/oip/cmd/oip")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build oip binary: %v", err)
	}
	return bin
}

// buildAwisBinary compiles the awis binary into dir and returns the path.
func buildAwisBinary(t *testing.T, dir string) string {
	t.Helper()
	bin := filepath.Join(dir, "awis")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/awis/awis/cmd/awis")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build awis binary: %v", err)
	}
	return bin
}

// copyWorkflowYAMLs copies the OIP workflow fixtures to <dir>/workflows/.
// Fixtures are copied byte-unchanged (M15 constraint).
func copyWorkflowYAMLs(t *testing.T, srcDir, destDir string) {
	t.Helper()
	wfDest := filepath.Join(destDir, "workflows")
	if err := os.MkdirAll(wfDest, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	for _, name := range []string{
		"capture-decision.yaml",
		"recall-decision.yaml",
		"rebuild-index.yaml",
	} {
		src := filepath.Join(srcDir, "workflows", name)
		dst := filepath.Join(wfDest, name)
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read workflow %s: %v", name, err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatalf("write workflow %s: %v", name, err)
		}
	}
}

// writePluginManifest writes a git-context-plugin or stub manifest into
// <pluginsDir>/git-context-plugin/awis-plugin.yaml.
// When python3Path is empty, writes a stub manifest that uses a shell echo
// to return a minimal valid JSON response (skip when no python3).
func writePluginManifest(t *testing.T, pluginsDir, python3Path, awisRoot string) string {
	t.Helper()
	pluginDir := filepath.Join(pluginsDir, "git-context-plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	manifestPath := filepath.Join(pluginDir, "awis-plugin.yaml")

	var content string
	if python3Path != "" {
		awisPluginLib := filepath.Join(awisRoot, "python", "awis-plugin")
		gitCtxPluginDir := filepath.Join(awisRoot, "plugins", "git-context-plugin")
		pythonPath := awisPluginLib + string(os.PathListSeparator) + gitCtxPluginDir
		content = fmt.Sprintf(`name: git-context-plugin
version: 1.0.0
description: Assembles git context for capture workflows
author: awis

capabilities:
  - id: git.context.assemble
    inputs:
      repo_path: string
      ref: string
    outputs:
      context: object
    timeout_ms: 30000

  - id: git.diff.fetch
    inputs:
      repo_path: string
      from_ref: string
      to_ref: string
    outputs:
      diff: string
    timeout_ms: 10000

runtime:
  command: %s
  args: ["-m", "git_context_plugin"]
  env:
    GIT_TERMINAL_PROMPT: "0"
    PYTHONPATH: %s
    AWIS_PLUGIN_LIBPATH: %s
  idle_timeout_s: 300
`, python3Path, pythonPath, awisPluginLib)
	} else {
		// Stub: write a minimal Python script that outputs valid context JSON.
		stubScript := filepath.Join(pluginDir, "stub_plugin.py")
		stubContent := `import json, sys
req = json.loads(sys.stdin.read())
print(json.dumps({"protocol":"awis-plugin/1","outputs":{"context":{"sha":"stub","message":"stub context","stats":{}}}}))
`
		if err := os.WriteFile(stubScript, []byte(stubContent), 0o755); err != nil {
			t.Fatalf("write stub plugin script: %v", err)
		}
		content = fmt.Sprintf(`name: git-context-plugin
version: 1.0.0
description: Stub git context plugin
author: awis

capabilities:
  - id: git.context.assemble
    inputs:
      repo_path: string
      ref: string
    outputs:
      context: object
    timeout_ms: 10000

runtime:
  command: python3
  args: [%q]
  idle_timeout_s: 30
`, stubScript)
	}

	if err := os.WriteFile(manifestPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}
	return manifestPath
}

// waitForOIPStartup reads from r until it sees "engine starting" in the output
// (the oip binary's startup log line) or times out.
func waitForOIPStartup(t *testing.T, buf *syncBuffer, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), "engine starting") {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Logf("waitForOIPStartup: timeout; collected output:\n%s", buf.String())
	return false
}

// awisRun runs the awis binary with the given data-dir, namespace, and subcommand+args.
// Returns stdout output and any error. Stderr is discarded (engine slog output).
func awisRun(t *testing.T, awisBin, dataDir, namespace string, args ...string) (string, error) {
	t.Helper()
	fullArgs := append([]string{"--data-dir=" + dataDir, "--json", "--namespace=" + namespace}, args...)
	cmd := exec.Command(awisBin, fullArgs...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	out := stdoutBuf.String()
	t.Logf("awis %v => stdout=%s stderr=%s", args, out, stderrBuf.String())
	return out, err
}

// pollInstanceStatus polls the awis status <instance-id> command until the
// instance reaches wantStatus or timeout. Returns the last status seen.
// Uses the detail JSON output from 'awis status <id> --json'.
func pollInstanceStatus(
	t *testing.T,
	ctx context.Context,
	awisBin, dataDir, namespace, instanceID string,
	wantStatus string,
	timeout time.Duration,
) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastStatus string
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return lastStatus
		default:
		}
		out, err := awisRun(t, awisBin, dataDir, namespace, "status", instanceID)
		if err == nil {
			// 'awis status <id> --json' returns a statusActiveJSON object.
			var detail struct {
				InstanceID  string  `json:"instance_id"`
				Status      string  `json:"status"`
				CurrentStep *string `json:"current_step"`
			}
			if jerr := json.Unmarshal([]byte(out), &detail); jerr == nil && detail.Status != "" {
				lastStatus = detail.Status
				if detail.Status == wantStatus {
					return lastStatus
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return lastStatus
}

// pollInstanceStep polls the awis status <instance-id> command until the
// instance reaches wantStep (current_step field) with wantStatus, or timeout.
// Returns the last current_step seen.
func pollInstanceStep(
	t *testing.T,
	ctx context.Context,
	awisBin, dataDir, namespace, instanceID string,
	wantStatus, wantStep string,
	timeout time.Duration,
) (string, string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastStatus, lastStep string
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return lastStatus, lastStep
		default:
		}
		out, err := awisRun(t, awisBin, dataDir, namespace, "status", instanceID)
		if err == nil {
			var detail struct {
				Status      string  `json:"status"`
				CurrentStep *string `json:"current_step"`
			}
			if jerr := json.Unmarshal([]byte(out), &detail); jerr == nil && detail.Status != "" {
				lastStatus = detail.Status
				if detail.CurrentStep != nil {
					lastStep = *detail.CurrentStep
				}
				if detail.Status == wantStatus && lastStep == wantStep {
					return lastStatus, lastStep
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return lastStatus, lastStep
}

// pollEntryFile polls until at least one .md file appears under
// <recordRoot>/.decisions/entries/ or timeout.
func pollEntryFile(recordRoot string, timeout time.Duration) (string, bool) {
	entriesDir := filepath.Join(recordRoot, ".decisions", "entries")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(entriesDir)
		if err == nil && len(entries) > 0 {
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".md") {
					return strings.TrimSuffix(e.Name(), ".md"), true
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", false
}

// syncBuffer is a goroutine-safe bytes.Buffer for collecting subprocess output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestOIPSystemE2E is the M15-C3r real-binary e2e test.
//
// Flow: build oip + awis binaries; start oip in a temp dir with copies of the
// OIP YAML fixtures and the git-context-plugin (or stub); submit
// capture-decision via awis CLI; signal manual_draft_provided (NullAdapter
// triggers fallback → manual-entry WAIT per restored fixture); then signal
// entry_confirmed with confirmed_entry JSON payload; poll until entry .md
// appended; submit recall-decision with a word from the entry; verify the
// entry id appears in the instance variables.
//
// Skipped under -short.
func TestOIPSystemE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e system test under -short")
	}

	root := repoRoot(t)

	// Build binaries.
	binDir := t.TempDir()
	oipBin := buildOIPBinary(t, binDir)
	awisBin := buildAwisBinary(t, binDir)

	// Set up temp directories.
	projDir := t.TempDir()
	dataDir := filepath.Join(projDir, ".awis")
	recordRoot := filepath.Join(projDir, "records")
	pluginsDir := filepath.Join(projDir, "plugins")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir dataDir: %v", err)
	}
	if err := os.MkdirAll(recordRoot, 0o755); err != nil {
		t.Fatalf("mkdir recordRoot: %v", err)
	}
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatalf("mkdir pluginsDir: %v", err)
	}

	// Source OIP app directory for workflow YAML fixtures.
	_, thisFile, _, _ := runtime.Caller(0)
	oipSrcDir := filepath.Dir(thisFile)

	// Copy OIP YAML fixtures byte-unchanged.
	copyWorkflowYAMLs(t, oipSrcDir, projDir)

	// Install git-context-plugin (real or stub if python3 absent).
	python3Path, _ := exec.LookPath("python3")
	writePluginManifest(t, pluginsDir, python3Path, root)

	// Start the oip binary using its natural namespace "oip" (the YAML fixture default).
	// The awis CLI targets this namespace via --namespace=oip (M15-C3r fix: the
	// --namespace global flag allows the CLI to reach any namespace without
	// embedding application names in platform code).
	startCmd := exec.Command(oipBin,
		"--data-dir="+dataDir,
		"--record-root="+recordRoot,
		"--namespace=oip",
		"--plugins-dir="+pluginsDir,
	)
	startCmd.Dir = projDir // workflow files discovered relative to cwd

	var oipOut syncBuffer
	startCmd.Stdout = &oipOut
	startCmd.Stderr = &oipOut

	if err := startCmd.Start(); err != nil {
		t.Fatalf("start oip: %v", err)
	}
	t.Cleanup(func() {
		_ = startCmd.Process.Signal(os.Interrupt)
		_ = startCmd.Wait()
	})

	// Wait for oip engine to start.
	if !waitForOIPStartup(t, &oipOut, 30*time.Second) {
		t.Fatal("oip did not start within 30s")
	}
	t.Log("oip started")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// All awis CLI calls target the "oip" namespace (--namespace=oip) so that the
	// CLI can find workflows registered by the oip binary in the "oip" namespace.
	const oipNS = "oip"

	// ── Submit capture-decision ───────────────────────────────────────────────
	// NOTE: --input flags must appear BEFORE the positional workflow-id argument
	// because flag.FlagSet.Parse stops at the first non-flag positional arg.
	submitOut, err := awisRun(t, awisBin, dataDir, oipNS, "submit",
		"--input", "repo_path="+root,
		"--input", "ref=HEAD",
		"capture-decision",
	)
	if err != nil {
		t.Fatalf("awis submit capture-decision: %v\noutput: %s", err, submitOut)
	}

	// Parse instance id from submit output.
	var submitJSON struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.Unmarshal([]byte(submitOut), &submitJSON); err != nil {
		t.Fatalf("parse submit output: %v\noutput: %s", err, submitOut)
	}
	instanceID := submitJSON.InstanceID
	t.Logf("submitted capture-decision; instance_id=%s", instanceID)

	// ── Wait for manual_draft_provided (manual-entry WAIT) ───────────────────
	// The NullAdapter triggers capability_fallback on draft-entry.
	// The restored fixture routes fallback → manual-entry (signal wait), which
	// waits for manual_draft_provided before proceeding to confirm-entry.
	status := pollInstanceStatus(t, ctx, awisBin, dataDir, oipNS, instanceID, "waiting", 45*time.Second)
	if status != "waiting" {
		t.Fatalf("instance did not reach waiting state for manual_draft_provided; last status=%q\noip output:\n%s", status, oipOut.String())
	}
	t.Log("instance waiting for manual_draft_provided")

	// Send manual_draft_provided with the draft content fields.
	// The card specifies: payload = the draft fields.
	manualPayload := map[string]any{
		"title":                 "E2E Test Decision",
		"decision":              "Adopt real binary e2e testing for OIP.",
		"rationale":             "Binary-level tests catch integration failures.",
		"rejected_alternatives": "Unit tests alone are insufficient.",
		"unknowns":              "Long-term maintenance cost.",
		"provenance_origin":     "manual",
		"provenance_authority":  "e2e-test",
		"provenance_confidence": "high",
	}
	manualPayloadBytes, _ := json.Marshal(manualPayload)
	manualPayloadFile := filepath.Join(t.TempDir(), "manual_payload.json")
	if err := os.WriteFile(manualPayloadFile, manualPayloadBytes, 0o644); err != nil {
		t.Fatalf("write manual payload: %v", err)
	}

	manualOut, err := awisRun(t, awisBin, dataDir, oipNS, "signal",
		"--payload", "@"+manualPayloadFile,
		instanceID, "manual_draft_provided",
	)
	if err != nil {
		t.Fatalf("signal manual_draft_provided: %v\noutput: %s", err, manualOut)
	}
	t.Logf("signaled manual_draft_provided: %s", manualOut)

	// ── Wait for entry_confirmed (confirm-entry WAIT) ─────────────────────────
	// After manual_draft_provided, the engine advances from manual-entry to
	// confirm-entry. Poll until current_step == "confirm-entry" with status "waiting".
	confirmStatus, confirmStep := pollInstanceStep(t, ctx, awisBin, dataDir, oipNS, instanceID, "waiting", "confirm-entry", 30*time.Second)
	if confirmStatus != "waiting" || confirmStep != "confirm-entry" {
		t.Fatalf("instance did not reach confirm-entry waiting state; last status=%q step=%q\noip output:\n%s", confirmStatus, confirmStep, oipOut.String())
	}
	t.Log("instance waiting for entry_confirmed")

	// Build the confirmed_entry JSON string (pre-encoded so the template engine
	// passes it through as a string; RecordAppendHandler JSON-decodes it).
	confirmedEntry := map[string]any{
		"title":                 "E2E Test Decision",
		"decision":              "Adopt real binary e2e testing for OIP.",
		"rationale":             "Binary-level tests catch integration failures.",
		"rejected_alternatives": "Unit tests alone are insufficient.",
		"unknowns":              "Long-term maintenance cost.",
		"provenance_origin":     "manual",
		"provenance_authority":  "e2e-test",
		"provenance_confidence": "high",
	}
	confirmedEntryBytes, err := json.Marshal(confirmedEntry)
	if err != nil {
		t.Fatalf("marshal confirmed_entry: %v", err)
	}

	// Send confirmed_entry as a JSON-encoded string in the payload.
	// The template {{steps.confirm-entry.outputs.confirmed_entry}} will stringify
	// the value; since it's already a string, it passes through unchanged.
	// RecordAppendHandler JSON-decodes the entry string.
	entryPayload := map[string]any{
		"confirmed_entry": string(confirmedEntryBytes),
	}
	entryPayloadBytes, _ := json.Marshal(entryPayload)
	entryPayloadFile := filepath.Join(t.TempDir(), "entry_payload.json")
	if err := os.WriteFile(entryPayloadFile, entryPayloadBytes, 0o644); err != nil {
		t.Fatalf("write entry payload: %v", err)
	}

	// NOTE: --payload flag must precede positional args (flag.FlagSet.Parse stops at
	// first non-flag positional argument).
	signalOut, err := awisRun(t, awisBin, dataDir, oipNS, "signal",
		"--payload", "@"+entryPayloadFile,
		instanceID, "entry_confirmed",
	)
	if err != nil {
		t.Fatalf("signal entry_confirmed: %v\noutput: %s", err, signalOut)
	}
	t.Logf("signaled entry_confirmed: %s", signalOut)

	// ── Poll until entry .md appended ────────────────────────────────────────
	entryID, found := pollEntryFile(recordRoot, 30*time.Second)
	if !found {
		// Dump final state for debugging.
		traceOut, _ := awisRun(t, awisBin, dataDir, oipNS, "trace", "--full", instanceID)
		t.Logf("final trace:\n%s", traceOut)
		t.Fatalf("entry .md not created within 30s\noip output:\n%s", oipOut.String())
	}
	t.Logf("entry .md created: %s", entryID)

	// ── Submit recall-decision ────────────────────────────────────────────────
	// Use a word from the entry title as the query.
	// NOTE: --input before positional id (flag.FlagSet stops at first non-flag).
	recallSubmitOut, err := awisRun(t, awisBin, dataDir, oipNS, "submit",
		"--input", "query=E2E Test Decision",
		"recall-decision",
	)
	if err != nil {
		t.Fatalf("awis submit recall-decision: %v\noutput: %s", err, recallSubmitOut)
	}

	var recallSubmitJSON struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.Unmarshal([]byte(recallSubmitOut), &recallSubmitJSON); err != nil {
		t.Fatalf("parse recall submit output: %v\noutput: %s", err, recallSubmitOut)
	}
	recallInstanceID := recallSubmitJSON.InstanceID
	t.Logf("submitted recall-decision; instance_id=%s", recallInstanceID)

	// Poll recall instance to completion (intelligence step uses NullAdapter → degrades gracefully).
	recallStatus := pollInstanceStatus(t, ctx, awisBin, dataDir, oipNS, recallInstanceID, "completed", 30*time.Second)
	// Recall uses an intelligence step (synthesize) that degrades with NullAdapter;
	// the instance may complete or fail at the intelligence step.
	// Core assertion: FTS step must have found the entry — check via trace.
	t.Logf("recall instance status: %s", recallStatus)

	// Use --full to get complete trace payloads (avoid 120-char truncation).
	traceOut, _ := awisRun(t, awisBin, dataDir, oipNS, "trace", "--full", recallInstanceID)
	t.Logf("recall trace (full):\n%s", traceOut)

	// The FTS step (oip.index.fts) runs and its StepCompleted payload contains
	// the results array with the entry id. Verify the entry id is in the trace.
	if !strings.Contains(traceOut, entryID) {
		t.Errorf("recall trace does not contain entry id %q\ntrace:\n%s", entryID, traceOut)
	}
}
