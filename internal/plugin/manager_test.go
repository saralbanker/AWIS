package plugin

// manager_test.go — FSM lifecycle tests for Manager (M12-C3, T7, T10).
//
// Tests: happy path; handshake mismatch; crash mid-call; 4 consecutive crashes
// → FAILED; counter reset after success; idle kill → respawn; timeout controlled
// kill; shutdown grace; goroutine-leak check; NFR-S-02 env isolation.
//
// Fixture plugins: testdata/bin/*.sh (NDJSON JSON-RPC speakers).
// Test manifests:  testdata/manifests/*.yaml
//
// TRACEABILITY: T7 (FSM manager); T10 (checkpoint — see checkpoint_test.go).

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// openTestStorage opens a fresh SQLite storage for manager tests.
// The returned *storage.SQLiteStorage implements storage.PluginStore.
func openTestStorage(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := storage.Open(path, nil)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return storage.NewSQLiteStorage(db, nil)
}

// newTestManager creates a Manager with fast-seam config for tests.
// handshakeTimeout defaults to 3s; no idle interval by default.
func newTestManager(t *testing.T, s *storage.SQLiteStorage, idleInterval time.Duration) *Manager {
	t.Helper()
	cfg := ManagerConfig{
		HandshakeTimeout:  3 * time.Second,
		ShutdownGrace:     500 * time.Millisecond,
		IdleCheckInterval: idleInterval,
	}
	return NewManager(s, cfg)
}

// resolveManifestPath returns an absolute path to a test manifest.
// Tests run from the package directory so "testdata/manifests/..." is fine,
// but we make it absolute to be safe.
func resolveManifestPath(t *testing.T, name string) string {
	t.Helper()
	// runtime.Caller(0) gives us this source file's path.
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	return filepath.Join(dir, "testdata", "manifests", name)
}

// countGoroutines returns the approximate number of live goroutines.
func countGoroutines() int {
	return runtime.NumGoroutine()
}

// TestManager_HappyPath tests spawn→handshake→execute→outputs (TDS-05 §7 happy path).
func TestManager_HappyPath(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)

	ctx := context.Background()
	manifestPath := resolveManifestPath(t, "test-plugin.yaml")

	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	outputs, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "hello"}, "")
	if stepErr != nil {
		t.Fatalf("Call: %v", stepErr)
	}
	if outputs["result"] != "ok" {
		t.Errorf("outputs[result]: got %v, want %q", outputs["result"], "ok")
	}

	// Shutdown cleanly.
	mgr.Shutdown(ctx)
}

// TestManager_HappyPath_ByPluginName tests plugin-name resolution (TDS-05 §6).
func TestManager_HappyPath_ByPluginName(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)

	ctx := context.Background()
	manifestPath := resolveManifestPath(t, "test-plugin.yaml")

	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Use plugin name as handler; "test-plugin" has one capability with input key "value".
	outputs, stepErr := mgr.Call(ctx, "test-plugin", "step-1", map[string]any{"value": "hello"}, "")
	if stepErr != nil {
		t.Fatalf("Call by plugin name: %v", stepErr)
	}
	if outputs["result"] != "ok" {
		t.Errorf("outputs[result]: got %v, want %q", outputs["result"], "ok")
	}

	mgr.Shutdown(ctx)
}

// TestManager_HandshakeMismatch tests that a wrong handshake name returns
// plugin_handshake_error and increments the crash counter (TDS-05 §2, §8).
func TestManager_HandshakeMismatch(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)

	ctx := context.Background()
	manifestPath := resolveManifestPath(t, "bad-handshake-plugin.yaml")

	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "x"}, "")
	if stepErr == nil {
		t.Fatal("expected error for bad handshake, got nil")
	}
	if stepErr.Code != "plugin_handshake_error" {
		t.Errorf("error code: got %q, want %q", stepErr.Code, "plugin_handshake_error")
	}

	// Crash counter should be 1.
	mgr.mu.Lock()
	entry := mgr.plugins["bad-handshake-plugin"]
	mgr.mu.Unlock()
	entry.mu.Lock()
	cc := entry.crashCount
	entry.mu.Unlock()
	if cc != 1 {
		t.Errorf("crashCount: got %d, want 1", cc)
	}

	mgr.Shutdown(ctx)
}

// TestManager_CrashMidCall tests crash during execute → plugin_crash + respawn
// on next call (TDS-05 §7 Crash; counter≤3 → REGISTERED = auto-restart).
func TestManager_CrashMidCall(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)

	ctx := context.Background()
	manifestPath := resolveManifestPath(t, "crash-plugin.yaml")

	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// First call crashes.
	_, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "x"}, "")
	if stepErr == nil {
		t.Fatal("expected plugin_crash, got nil")
	}
	if stepErr.Code != "plugin_crash" {
		t.Errorf("error code: got %q, want %q", stepErr.Code, "plugin_crash")
	}

	// State should be REGISTERED (auto-restart ready), counter=1.
	mgr.mu.Lock()
	entry := mgr.plugins["crash-plugin"]
	mgr.mu.Unlock()
	entry.mu.Lock()
	state := entry.state
	cc := entry.crashCount
	entry.mu.Unlock()
	if state != stateRegistered {
		t.Errorf("state: got %v, want REGISTERED", state)
	}
	if cc != 1 {
		t.Errorf("crashCount: got %d, want 1", cc)
	}

	mgr.Shutdown(ctx)
}

// TestManager_FourConsecutiveCrashes tests that >3 consecutive crashes
// transitions the plugin to FAILED and subsequent calls fail with plugin_failed
// (TDS-05 §7 Crash, "Counter > 3 → FAILED").
func TestManager_FourConsecutiveCrashes(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)

	ctx := context.Background()
	manifestPath := resolveManifestPath(t, "crash-plugin.yaml")

	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Trigger 4 consecutive crashes.
	for i := 1; i <= 4; i++ {
		_, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "x"}, "")
		if stepErr == nil {
			t.Fatalf("call %d: expected error, got nil", i)
		}
		if i < 4 && stepErr.Code != "plugin_crash" {
			t.Errorf("call %d: expected plugin_crash, got %q", i, stepErr.Code)
		}
		if i == 4 && stepErr.Code != "plugin_failed" {
			t.Errorf("call 4 (first FAILED): expected plugin_failed, got %q", stepErr.Code)
		}
	}

	// All further calls must fail with plugin_failed.
	_, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "x"}, "")
	if stepErr == nil || stepErr.Code != "plugin_failed" {
		t.Errorf("after FAILED: expected plugin_failed, got %v", stepErr)
	}

	// Verify DB status was set to failed.
	// (Async store write; give it a moment.)
	time.Sleep(50 * time.Millisecond)
	row, err := s.GetPlugin(ctx, "crash-plugin")
	if err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}
	if row.Status != "failed" {
		t.Errorf("DB status: got %q, want %q", row.Status, "failed")
	}

	mgr.Shutdown(ctx)
}

// TestManager_CounterResetAfterSuccess tests that the crash counter resets on
// a successful execute (TDS-05 §7 "Counter RESETS on a successful execute").
func TestManager_CounterResetAfterSuccess(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	// Use a manifest that alternates between crash and happy behaviours.
	// We can't change a fixture's behaviour per-call, so we'll directly
	// manipulate the crash counter to simulate 2 prior crashes, then
	// issue a call to the happy-path plugin to verify reset.
	manifestPath := resolveManifestPath(t, "test-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Manually bump the crash counter to simulate 2 prior crashes.
	mgr.mu.Lock()
	entry := mgr.plugins["test-plugin"]
	mgr.mu.Unlock()
	entry.mu.Lock()
	entry.crashCount = 2
	entry.mu.Unlock()

	// Successful call must reset crash counter to 0.
	outputs, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "hello"}, "")
	if stepErr != nil {
		t.Fatalf("Call: %v", stepErr)
	}
	if outputs["result"] != "ok" {
		t.Errorf("outputs[result]: got %v", outputs["result"])
	}

	entry.mu.Lock()
	cc := entry.crashCount
	entry.mu.Unlock()
	if cc != 0 {
		t.Errorf("crashCount after success: got %d, want 0", cc)
	}

	mgr.Shutdown(ctx)
}

// TestManager_IdleKillRespawn tests that an idle kill transparently respawns
// the plugin on the next call (TDS-05 §7 IDLE). Uses the fast clock seam.
func TestManager_IdleKillRespawn(t *testing.T) {
	s := openTestStorage(t)
	// Use a very short idle interval (fast clock seam; SPEC "internal package — allowed").
	mgr := newTestManager(t, s, 50*time.Millisecond)
	ctx := context.Background()

	manifestPath := resolveManifestPath(t, "test-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// First call: spawn and serve.
	outputs, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "a"}, "")
	if stepErr != nil {
		t.Fatalf("first Call: %v", stepErr)
	}
	if outputs["result"] != "ok" {
		t.Errorf("first call result: got %v", outputs["result"])
	}

	// Capture the PID after the first call.
	pid1 := mgr.PluginPID("test-plugin")
	if pid1 == 0 {
		t.Fatal("expected non-zero PID after first call")
	}

	// Wait for idle kill (50ms interval, add buffer).
	// Per card constraint: use the seam; ≤2s test.
	time.Sleep(200 * time.Millisecond)

	// Verify state is IDLE.
	mgr.mu.Lock()
	entry := mgr.plugins["test-plugin"]
	mgr.mu.Unlock()
	entry.mu.Lock()
	state := entry.state
	entry.mu.Unlock()
	if state != stateIdle {
		t.Errorf("state after idle: got %v, want IDLE", state)
	}

	// Second call: transparent respawn.
	outputs2, stepErr2 := mgr.Call(ctx, "test.cap", "step-2", map[string]any{"value": "b"}, "")
	if stepErr2 != nil {
		t.Fatalf("second Call (after idle): %v", stepErr2)
	}
	if outputs2["result"] != "ok" {
		t.Errorf("second call result: got %v", outputs2["result"])
	}

	// The plugin should have a new PID (different process).
	pid2 := mgr.PluginPID("test-plugin")
	if pid2 == 0 {
		t.Fatal("expected non-zero PID after respawn")
	}
	if pid2 == pid1 {
		t.Errorf("expected different PID after respawn (got same: %d)", pid1)
	}

	mgr.Shutdown(ctx)
}

// TestManager_TimeoutControlledKill tests that a call timeout kills the process
// group but does NOT increment the crash counter (TDS-05 §5, §8).
func TestManager_TimeoutControlledKill(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	manifestPath := resolveManifestPath(t, "slow-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Issue call with a very short timeout (100ms).
	callCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, stepErr := mgr.Call(callCtx, "test.slow", "step-1", map[string]any{"value": "x"}, "")
	if stepErr == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if stepErr.Code != "timeout" {
		t.Errorf("error code: got %q, want %q", stepErr.Code, "timeout")
	}

	// Crash counter must NOT have been incremented (timeout = controlled kill).
	mgr.mu.Lock()
	entry := mgr.plugins["slow-plugin"]
	mgr.mu.Unlock()
	entry.mu.Lock()
	cc := entry.crashCount
	entry.mu.Unlock()
	if cc != 0 {
		t.Errorf("crashCount after timeout: got %d, want 0 (timeout must not count)", cc)
	}

	mgr.Shutdown(ctx)
}

// TestManager_AmbiguousResolution tests the plugin_capability_ambiguous error
// (TDS-05 §6: zero or multiple matches → typed error).
func TestManager_AmbiguousResolution(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	// git-context-plugin has two capabilities with different input key sets.
	manifestPath := resolveManifestPath(t, "git-context-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Use an input key set that matches zero capabilities.
	_, stepErr := mgr.Call(ctx, "git-context-plugin", "step-1",
		map[string]any{"unknown_key": "x"}, "")
	if stepErr == nil {
		t.Fatal("expected plugin_capability_ambiguous, got nil")
	}
	if stepErr.Code != "plugin_capability_ambiguous" {
		t.Errorf("error code: got %q, want %q", stepErr.Code, "plugin_capability_ambiguous")
	}
}

// TestManager_UnknownHandler tests plugin_not_found for unknown handler ref
// (TDS-05 §6, §8).
func TestManager_UnknownHandler(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	_, stepErr := mgr.Call(ctx, "no-such-plugin", "step-1", map[string]any{}, "")
	if stepErr == nil {
		t.Fatal("expected plugin_not_found, got nil")
	}
	if stepErr.Code != "plugin_not_found" {
		t.Errorf("error code: got %q, want %q", stepErr.Code, "plugin_not_found")
	}
}

// TestManager_EnvIsolation tests NFR-S-02: canary parent env var must NOT
// appear in the plugin child process environment.
func TestManager_EnvIsolation(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	// Set the canary variable in the parent process.
	const canaryKey = "TEST_CANARY_PARENT_VAR"
	const canaryVal = "canary_value_must_not_leak"
	t.Setenv(canaryKey, canaryVal)

	manifestPath := resolveManifestPath(t, "env-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	outputs, stepErr := mgr.Call(ctx, "test.env", "step-1", map[string]any{"check": "env"}, "")
	if stepErr != nil {
		t.Fatalf("Call: %v", stepErr)
	}

	// The plugin reports what it found for TEST_CANARY_PARENT_VAR.
	// It must be "absent" (the default in the shell script) — NOT the canary value.
	got, _ := outputs["env_var"].(string)
	if got == canaryVal {
		t.Errorf("NFR-S-02 VIOLATED: canary env var leaked into plugin child process (got %q)", got)
	}
	if got != "absent" {
		t.Logf("env_var=%q (expected 'absent'; any non-canary value is acceptable)", got)
	}

	mgr.Shutdown(ctx)
}

// TestManager_ShutdownGraceAndKill tests Manager.Shutdown: notification sent,
// grace period, then SIGKILL (TDS-05 §7 Shutdown).
func TestManager_ShutdownGraceAndKill(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	manifestPath := resolveManifestPath(t, "test-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Spawn by calling.
	if _, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "x"}, ""); stepErr != nil {
		t.Fatalf("Call: %v", stepErr)
	}

	start := time.Now()
	mgr.Shutdown(ctx)
	elapsed := time.Since(start)

	// Shutdown must complete within ShutdownGrace + small buffer (500ms + 200ms).
	if elapsed > 800*time.Millisecond {
		t.Errorf("Shutdown took too long: %s", elapsed)
	}

	// Idempotent: second Shutdown must not panic.
	mgr.Shutdown(ctx)
}

// TestManager_ShutdownIdempotent tests that Shutdown is safe to call multiple times
// (TDS-05 §7 "Idempotent").
func TestManager_ShutdownIdempotent(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	// Call Shutdown before any plugin is spawned.
	mgr.Shutdown(ctx)
	mgr.Shutdown(ctx)
	mgr.Shutdown(ctx)
	// No panic = pass.
}

// TestManager_GoroutineLeak checks that no goroutines leak after Shutdown
// (SPEC invariant 5: "monitor goroutines never leak past Shutdown").
func TestManager_GoroutineLeak(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	before := countGoroutines()

	manifestPath := resolveManifestPath(t, "test-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Spawn by calling.
	if _, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "x"}, ""); stepErr != nil {
		t.Fatalf("Call: %v", stepErr)
	}

	mgr.Shutdown(ctx)

	// Allow goroutines to settle.
	time.Sleep(100 * time.Millisecond)

	after := countGoroutines()
	// Allow a small tolerance for Go runtime goroutines.
	if after > before+2 {
		t.Errorf("goroutine leak: before=%d, after=%d (delta=%d)", before, after, after-before)
	}
}

// TestManager_ReRegister tests that re-registering the same plugin resets its state.
func TestManager_ReRegister(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	manifestPath := resolveManifestPath(t, "test-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	// Simulate prior crash count.
	mgr.mu.Lock()
	entry := mgr.plugins["test-plugin"]
	mgr.mu.Unlock()
	entry.mu.Lock()
	entry.crashCount = 4
	entry.state = stateFailed
	entry.mu.Unlock()

	// Re-register: must reset state and crash count.
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("second Register: %v", err)
	}

	entry.mu.Lock()
	state := entry.state
	cc := entry.crashCount
	entry.mu.Unlock()
	if state != stateRegistered {
		t.Errorf("state after re-register: got %v, want REGISTERED", state)
	}
	if cc != 0 {
		t.Errorf("crashCount after re-register: got %d, want 0", cc)
	}

	// Call must succeed again.
	outputs, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "x"}, "")
	if stepErr != nil {
		t.Fatalf("Call after re-register: %v", stepErr)
	}
	if outputs["result"] != "ok" {
		t.Errorf("result: got %v", outputs["result"])
	}

	mgr.Shutdown(ctx)
}

// TestManager_PluginNameResolutionUnique tests direct capability id resolution.
func TestManager_PluginNameResolutionUnique(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	ctx := context.Background()

	// git-context-plugin has two capabilities.
	// git.context.assemble takes {repo_path, ref}.
	// git.diff.fetch takes {repo_path, from_ref, to_ref}.
	// We don't actually spawn it (no real python), but we can test resolution.
	// Instead use the test-plugin manifest for a direct capability id call.
	manifestPath := resolveManifestPath(t, "test-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Direct capability id — must work without ambiguity.
	outputs, stepErr := mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "direct"}, "")
	if stepErr != nil {
		t.Fatalf("Call by capability id: %v", stepErr)
	}
	if outputs["result"] != "ok" {
		t.Errorf("result: got %v", outputs["result"])
	}

	mgr.Shutdown(ctx)
}

// TestManager_HandshakeInvariant2_NoDead tests invariant 2: crash during
// HANDSHAKING must not deadlock waiters (TDS-05 §7 invariant 2).
func TestManager_HandshakeInvariant2_NoDead(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	// Short handshake timeout to speed up the test.
	mgr.cfg.HandshakeTimeout = 200 * time.Millisecond
	ctx := context.Background()

	// bad-handshake-plugin returns wrong name → handshake failure → crash.
	manifestPath := resolveManifestPath(t, "bad-handshake-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// This must return (not deadlock) within the test timeout.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = mgr.Call(ctx, "test.cap", "step-1", map[string]any{"value": "x"}, "")
	}()
	select {
	case <-done:
		// OK: returned without deadlock.
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock: Call did not return after handshake crash")
	}

	mgr.Shutdown(ctx)
}

// TestPluginRunner_UnknownHandler tests the PluginRunner's plugin_not_found path.
func TestPluginRunner_UnknownHandler(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	runner := NewPluginRunner(mgr)

	ctx := context.Background()
	sc := core.StepContext{
		StepID: "s1",
		Inputs: map[string]any{},
	}
	step := core.Step{
		ID:      "s1",
		Type:    core.StepTypePlugin,
		Handler: "no-such-plugin",
	}

	_, stepErr := runner.Run(ctx, sc, step)
	if stepErr == nil {
		t.Fatal("expected error, got nil")
	}
	if stepErr.Code != "plugin_not_found" {
		t.Errorf("code: got %q, want plugin_not_found", stepErr.Code)
	}
}

// TestPluginRunner_HappyPath tests the full PluginRunner dispatch.
func TestPluginRunner_HappyPath(t *testing.T) {
	s := openTestStorage(t)
	mgr := newTestManager(t, s, 0)
	runner := NewPluginRunner(mgr)

	ctx := context.Background()
	manifestPath := resolveManifestPath(t, "test-plugin.yaml")
	if err := mgr.Register(ctx, manifestPath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	sc := core.StepContext{
		StepID: "s1",
		Inputs: map[string]any{"value": "runner-test"},
	}
	step := core.Step{
		ID:      "s1",
		Type:    core.StepTypePlugin,
		Handler: "test.cap",
	}

	result, stepErr := runner.Run(ctx, sc, step)
	if stepErr != nil {
		t.Fatalf("Run: %v", stepErr)
	}
	if result.Outputs["result"] != "ok" {
		t.Errorf("result: got %v", result.Outputs["result"])
	}

	mgr.Shutdown(ctx)
}

// TestBuildEnv tests NFR-S-02 env construction: only manifest env + PATH.
func TestBuildEnv(t *testing.T) {
	// Set a "parent" env var that must not appear in the built env.
	const leakKey = "SHOULD_NOT_LEAK"
	t.Setenv(leakKey, "1")

	manifestEnv := map[string]string{"FOO": "bar"}
	env := buildEnv(manifestEnv, "")

	for _, kv := range env {
		if strings.HasPrefix(kv, leakKey+"=") {
			t.Errorf("NFR-S-02: parent env var %q leaked into built env", leakKey)
		}
	}

	// Must contain FOO=bar.
	found := false
	for _, kv := range env {
		if kv == "FOO=bar" {
			found = true
		}
	}
	if !found {
		t.Error("manifest env FOO=bar not found in built env")
	}

	// Must contain PATH=...
	hasPath := false
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			hasPath = true
		}
	}
	if !hasPath {
		t.Error("PATH not found in built env")
	}
}

// TestComputeEffectiveTimeout tests min(ctx deadline, step.Timeout, cap timeout_ms).
func TestComputeEffectiveTimeout(t *testing.T) {
	manifest := &Manifest{
		Name:    "m",
		Version: "1.0.0",
		Capabilities: []Capability{
			{ID: "cap-a", TimeoutMS: 5000},
		},
	}
	mgr := &Manager{cfg: ManagerConfig{}}

	// Only capability timeout available.
	ctx := context.Background()
	got := mgr.computeEffectiveTimeout(ctx, manifest, "cap-a", "")
	if got != 5*time.Second {
		t.Errorf("cap-only: got %s, want 5s", got)
	}

	// Step timeout < cap timeout.
	got2 := mgr.computeEffectiveTimeout(ctx, manifest, "cap-a", core.Duration("2s"))
	if got2 != 2*time.Second {
		t.Errorf("step<cap: got %s, want 2s", got2)
	}

	// ctx deadline < step timeout < cap timeout.
	deadlineCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	got3 := mgr.computeEffectiveTimeout(deadlineCtx, manifest, "cap-a", core.Duration("2s"))
	// ctx remaining ≈ 1s; step=2s; cap=5s → min = ~1s
	if got3 > 1*time.Second+50*time.Millisecond || got3 < 800*time.Millisecond {
		t.Errorf("ctx<step<cap: got %s, want ~1s", got3)
	}
}

// TestInputKeySetMatches tests the capability resolution key-set matching.
func TestInputKeySetMatches(t *testing.T) {
	declared := map[string]any{"a": "string", "b": "string"}
	tests := []struct {
		name   string
		inputs map[string]bool
		want   bool
	}{
		{"exact match", map[string]bool{"a": true, "b": true}, true},
		{"extra key", map[string]bool{"a": true, "b": true, "c": true}, false},
		{"missing key", map[string]bool{"a": true}, false},
		{"empty", map[string]bool{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inputKeySetMatches(declared, tt.inputs)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestBuildEnvResolvesRelativeModulePaths covers B-21: the shipped manifest
// must be self-sufficient. A relative AWIS_PLUGIN_LIBPATH / PYTHONPATH used
// to be interpreted against the AWIS process's working directory, so the
// shipped git-context-plugin resolved its own package only when awis happened
// to be run from the plugin's directory.
func TestBuildEnvResolvesRelativeModulePaths(t *testing.T) {
	const dir = "/opt/awis/plugins/git-context"

	env := buildEnv(map[string]string{
		"AWIS_PLUGIN_LIBPATH": "../../python/awis-plugin",
		"PYTHONPATH":          "lib",
		"NOT_A_PATH":          "../../python/awis-plugin",
	}, dir)

	got := envMap(env)

	if want := "/opt/awis/python/awis-plugin"; got["AWIS_PLUGIN_LIBPATH"] != want {
		t.Errorf("AWIS_PLUGIN_LIBPATH = %q, want %q", got["AWIS_PLUGIN_LIBPATH"], want)
	}
	if want := dir + "/lib"; got["PYTHONPATH"] != want {
		t.Errorf("PYTHONPATH = %q, want %q", got["PYTHONPATH"], want)
	}
	// A key not on the allowlist must be passed through untouched: silently
	// rewriting a value the plugin author did not mean as a path would be a
	// hard-to-debug corruption of the plugin's environment.
	if want := "../../python/awis-plugin"; got["NOT_A_PATH"] != want {
		t.Errorf("NOT_A_PATH = %q, want %q (unchanged)", got["NOT_A_PATH"], want)
	}
}

// TestBuildEnvLeavesAbsolutePathsAndEmptyDirAlone pins the two no-op cases:
// an already-absolute entry must not be re-anchored, and an empty dir (a
// manifest parsed from bytes with no real path) must change nothing at all.
func TestBuildEnvLeavesAbsolutePathsAndEmptyDirAlone(t *testing.T) {
	abs := buildEnv(map[string]string{"PYTHONPATH": "/already/absolute"}, "/opt/plugin")
	if got := envMap(abs)["PYTHONPATH"]; got != "/already/absolute" {
		t.Errorf("absolute PYTHONPATH = %q, want it unchanged", got)
	}

	none := buildEnv(map[string]string{"PYTHONPATH": "relative/path"}, "")
	if got := envMap(none)["PYTHONPATH"]; got != "relative/path" {
		t.Errorf("PYTHONPATH with empty dir = %q, want it unchanged", got)
	}
}

// TestBuildEnvResolvesEveryEntryOfAPathList checks the separator handling: a
// path LIST must have each relative element anchored independently, with
// absolute elements left alone.
func TestBuildEnvResolvesEveryEntryOfAPathList(t *testing.T) {
	const dir = "/opt/plugin"
	sep := string(os.PathListSeparator)

	env := buildEnv(map[string]string{
		"PYTHONPATH": "lib" + sep + "/abs/lib" + sep + "vendor",
	}, dir)

	want := "/opt/plugin/lib" + sep + "/abs/lib" + sep + "/opt/plugin/vendor"
	if got := envMap(env)["PYTHONPATH"]; got != want {
		t.Errorf("PYTHONPATH = %q, want %q", got, want)
	}
}

// envMap turns a KEY=VALUE slice into a map for readable assertions.
func envMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, kv := range env {
		if i := strings.Index(kv, "="); i >= 0 {
			out[kv[:i]] = kv[i+1:]
		}
	}
	return out
}
