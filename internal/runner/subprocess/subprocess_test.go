package subprocess

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// goldenDir is the path to the golden protocol files shared by the Go
// conformance tests and (in C3) the Python pytest suite (TDS-04 §9).
func goldenDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", "protocol")
}

// binDir returns the path to the fixture shell scripts.
func binDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", "bin")
}

// readGolden reads a golden file and returns its bytes (fails test on error).
func readGolden(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(goldenDir(t), name))
	if err != nil {
		t.Fatalf("readGolden(%q): %v", name, err)
	}
	return b
}

// unmarshalAny unmarshals JSON bytes into map[string]any for comparison.
func unmarshalAny(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshalAny: %v", err)
	}
	return m
}

// jsonEqual returns true iff a and b represent the same JSON value.
func jsonEqual(a, b []byte) bool {
	var va, vb any
	if json.Unmarshal(a, &va) != nil || json.Unmarshal(b, &vb) != nil {
		return false
	}
	ra, _ := json.Marshal(va)
	rb, _ := json.Marshal(vb)
	return string(ra) == string(rb)
}

// baseStep returns a minimal subprocess step (no timeout).
func baseStep() core.Step {
	return core.Step{
		ID:      "compute-hrv",
		Name:    "compute-hrv",
		Type:    core.StepTypeSubprocess,
		Handler: "python my_step.py",
	}
}

// baseSC returns the StepContext matching the golden basic/deadline fixtures.
func baseSC() core.StepContext {
	return core.StepContext{
		InstanceID: "inst-001",
		StepID:     "compute-hrv",
		Attempt:    1,
		Inputs:     map[string]any{"records": []any{float64(1), float64(2), float64(3)}},
	}
}

// ── golden conformance: request emission ─────────────────────────────────────

// TestGoldenConformance_RequestBasic verifies that the runner emits a request
// JSON-equal to request-basic.json for a step with no timeout.
func TestGoldenConformance_RequestBasic(t *testing.T) {
	want := unmarshalAny(t, readGolden(t, "request-basic.json"))

	env := requestEnvelope{
		Protocol:   protocol,
		Handler:    "python my_step.py",
		InstanceID: "inst-001",
		StepID:     "compute-hrv",
		Attempt:    1,
		Inputs:     map[string]any{"records": []any{float64(1), float64(2), float64(3)}},
	}
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !jsonEqual(got, readGolden(t, "request-basic.json")) {
		gotM := unmarshalAny(t, got)
		t.Errorf("request envelope mismatch\ngot:  %v\nwant: %v", gotM, want)
	}
}

// TestGoldenConformance_RequestDeadline verifies that the runner emits a
// request JSON-equal to request-deadline.json when a deadline is set.
func TestGoldenConformance_RequestDeadline(t *testing.T) {
	deadline, err := time.Parse(time.RFC3339Nano, "2099-01-01T00:00:00.000000000Z")
	if err != nil {
		t.Fatalf("parse deadline: %v", err)
	}

	env := requestEnvelope{
		Protocol:   protocol,
		Handler:    "python my_step.py",
		InstanceID: "inst-001",
		StepID:     "compute-hrv",
		Attempt:    1,
		Inputs:     map[string]any{"records": []any{float64(1), float64(2), float64(3)}},
		Deadline:   deadline.UTC().Format(deadlineFormat),
	}
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !jsonEqual(got, readGolden(t, "request-deadline.json")) {
		t.Errorf("request-deadline mismatch\ngot:  %s\nwant: %s", got, readGolden(t, "request-deadline.json"))
	}
}

// TestGoldenConformance_ResponseOK verifies that response-ok.json is parsed to
// the expected StepResult.
func TestGoldenConformance_ResponseOK(t *testing.T) {
	b := readGolden(t, "response-ok.json")
	var resp responseEnvelope
	if err := json.Unmarshal(b, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Protocol != protocol {
		t.Errorf("protocol: got %q, want %q", resp.Protocol, protocol)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error block: %+v", resp.Error)
	}
	hrv, ok := resp.Outputs["hrv"]
	if !ok {
		t.Fatalf("outputs missing 'hrv' key")
	}
	if hrv != float64(42.5) {
		t.Errorf("hrv: got %v, want 42.5", hrv)
	}
}

// TestGoldenConformance_ResponseError verifies that response-error.json is
// parsed to the expected StepError.
func TestGoldenConformance_ResponseError(t *testing.T) {
	b := readGolden(t, "response-error.json")
	var resp responseEnvelope
	if err := json.Unmarshal(b, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Protocol != protocol {
		t.Errorf("protocol: got %q, want %q", resp.Protocol, protocol)
	}
	if resp.Error == nil {
		t.Fatal("expected error block")
	}
	if resp.Error.Code != "validation_error" {
		t.Errorf("code: got %q, want %q", resp.Error.Code, "validation_error")
	}
	if resp.Error.Message != "records field must not be empty" {
		t.Errorf("message: got %q", resp.Error.Message)
	}
	if resp.Error.Details["field"] != "records" {
		t.Errorf("details.field: got %v", resp.Error.Details["field"])
	}
}

// TestGoldenConformance_BadProtocol verifies bad-protocol.json is rejected as protocol_error.
func TestGoldenConformance_BadProtocol(t *testing.T) {
	b := readGolden(t, "bad-protocol.json")
	var resp responseEnvelope
	if err := json.Unmarshal(b, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Protocol == protocol {
		t.Errorf("expected wrong protocol, got %q", resp.Protocol)
	}
}

// TestGoldenConformance_BadTruncated verifies bad-truncated.json fails JSON parse.
func TestGoldenConformance_BadTruncated(t *testing.T) {
	b := readGolden(t, "bad-truncated.json")
	var resp responseEnvelope
	if err := json.Unmarshal(b, &resp); err == nil {
		t.Error("expected JSON parse error for bad-truncated.json, got nil")
	}
}

// ── behavior tests ────────────────────────────────────────────────────────────

// TestBehavior_OKPath tests the successful round-trip via ok.sh.
func TestBehavior_OKPath(t *testing.T) {
	r := New()
	sc := baseSC()
	step := baseStep()
	step.Handler = core.HandlerRef(filepath.Join(binDir(t), "ok.sh"))

	res, se := r.Run(context.Background(), sc, step)
	if se != nil {
		t.Fatalf("Run returned error: %+v", se)
	}
	if res.Outputs["result"] != "ok" {
		t.Errorf("expected outputs.result=ok, got %v", res.Outputs)
	}
}

// TestBehavior_ErrorEnvelopePath tests that nonzero exit + valid error envelope
// → envelope wins (TDS-04 §7).
func TestBehavior_ErrorEnvelopePath(t *testing.T) {
	r := New()
	sc := baseSC()
	step := baseStep()
	step.Handler = core.HandlerRef(filepath.Join(binDir(t), "error-envelope.sh"))

	_, se := r.Run(context.Background(), sc, step)
	if se == nil {
		t.Fatal("expected StepError, got nil")
	}
	if se.Code != "validation_error" {
		t.Errorf("code: got %q, want %q", se.Code, "validation_error")
	}
	if se.Message != "records field must not be empty" {
		t.Errorf("message: got %q", se.Message)
	}
}

// TestBehavior_NonzeroExitGarbageStdout tests that nonzero exit + garbage
// stdout → subprocess_error (TDS-04 §7).
func TestBehavior_NonzeroExitGarbageStdout(t *testing.T) {
	r := New()
	sc := baseSC()
	step := baseStep()
	step.Handler = core.HandlerRef(filepath.Join(binDir(t), "garbage-stdout.sh"))

	_, se := r.Run(context.Background(), sc, step)
	if se == nil {
		t.Fatal("expected StepError, got nil")
	}
	if se.Code != "subprocess_error" {
		t.Errorf("code: got %q, want %q", se.Code, "subprocess_error")
	}
	if _, ok := se.Details["exit_code"]; !ok {
		t.Errorf("details missing exit_code: %v", se.Details)
	}
}

// TestBehavior_BadProtocol tests that zero exit + wrong protocol → protocol_error.
func TestBehavior_BadProtocol(t *testing.T) {
	r := New()
	sc := baseSC()
	step := baseStep()
	step.Handler = core.HandlerRef(filepath.Join(binDir(t), "bad-protocol.sh"))

	_, se := r.Run(context.Background(), sc, step)
	if se == nil {
		t.Fatal("expected StepError, got nil")
	}
	if se.Code != "protocol_error" {
		t.Errorf("code: got %q, want %q", se.Code, "protocol_error")
	}
}

// TestBehavior_Timeout tests that a sleeping child is killed within ≤2s and
// returns a timeout error (TDS-04 §6).
func TestBehavior_Timeout(t *testing.T) {
	r := New()
	sc := baseSC()
	step := baseStep()
	step.Handler = core.HandlerRef(filepath.Join(binDir(t), "sleep.sh"))
	step.Timeout = core.Duration("200ms") // fast timeout, test budget ≤2s

	start := time.Now()
	_, se := r.Run(context.Background(), sc, step)
	elapsed := time.Since(start)

	if se == nil {
		t.Fatal("expected StepError, got nil")
	}
	if se.Code != "timeout" {
		t.Errorf("code: got %q, want %q", se.Code, "timeout")
	}
	if elapsed > 2*time.Second {
		t.Errorf("timeout test took %v (want ≤2s)", elapsed)
	}
}

// TestBehavior_SpawnFailure tests that a nonexistent binary → spawn_error.
func TestBehavior_SpawnFailure(t *testing.T) {
	r := New()
	sc := baseSC()
	step := baseStep()
	step.Handler = "/nonexistent/binary/that/does/not/exist"

	_, se := r.Run(context.Background(), sc, step)
	if se == nil {
		t.Fatal("expected StepError, got nil")
	}
	if se.Code != "spawn_error" {
		t.Errorf("code: got %q, want %q", se.Code, "spawn_error")
	}
}

// TestBehavior_StderrCapture tests that stderr content appears in error details.
func TestBehavior_StderrCapture(t *testing.T) {
	r := New()
	sc := baseSC()
	step := baseStep()
	step.Handler = core.HandlerRef(filepath.Join(binDir(t), "garbage-stdout.sh"))

	_, se := r.Run(context.Background(), sc, step)
	if se == nil {
		t.Fatal("expected StepError, got nil")
	}
	// stderr tail must be present in details (TDS-04 §8).
	if _, ok := se.Details["stderr"]; !ok {
		t.Errorf("details missing stderr key: %v", se.Details)
	}
}

// TestSubprocess_SanitizedEnvironment verifies that host secrets are not leaked
// to subprocesses, while PATH and AWIS context variables are passed (RC-4).
func TestSubprocess_SanitizedEnvironment(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "secret-key-12345")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "check_env.sh")
	content := `#!/bin/sh
cat > /dev/null
printf '{"protocol":"awis-subprocess/1","outputs":{"has_api_key":"%s","has_db_url":"%s","has_path":"%s","inst_id":"%s"}}\n' "$ANTHROPIC_API_KEY" "$DATABASE_URL" "$PATH" "$AWIS_INSTANCE_ID"
`
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	r := New()
	sc := baseSC()
	sc.InstanceID = "test-inst-123"
	step := baseStep()
	step.Handler = core.HandlerRef(scriptPath)

	res, se := r.Run(context.Background(), sc, step)
	if se != nil {
		t.Fatalf("Run: %v", se)
	}
	if res.Outputs["has_api_key"] != "" {
		t.Errorf("host secret ANTHROPIC_API_KEY was leaked to subprocess: %v", res.Outputs["has_api_key"])
	}
	if res.Outputs["has_db_url"] != "" {
		t.Errorf("host secret DATABASE_URL was leaked to subprocess: %v", res.Outputs["has_db_url"])
	}
	if res.Outputs["has_path"] == "" {
		t.Errorf("PATH was not preserved in subprocess")
	}
	if res.Outputs["inst_id"] != "test-inst-123" {
		t.Errorf("AWIS_INSTANCE_ID not passed: got %v", res.Outputs["inst_id"])
	}
}
