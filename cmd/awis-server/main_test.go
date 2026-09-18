package main

// main_test.go — E-G4-1 acceptance test: build the real binary, start it
// against a temp SQLite DB and an ephemeral loopback port, confirm it comes
// up, send SIGTERM, and confirm the process exits promptly and cleanly. A
// hang here would indicate the engine goroutine or the HTTP listener failed
// to observe the shutdown signal (goroutine leak); this mirrors the
// subprocess pattern cmd/awis/system_test.go uses for the same class of
// assertion against 'awis start'.

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// buildServerBinary compiles the awis-server binary into dir and returns the
// path to the binary.
func buildServerBinary(t *testing.T, dir string) string {
	t.Helper()
	bin := filepath.Join(dir, "awis-server")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/awis/awis/cmd/awis-server")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build binary: %v", err)
	}
	return bin
}

// freeLoopbackAddr asks the OS for an unused loopback port and returns its
// address string ("127.0.0.1:PORT"). The listener is closed immediately
// before returning, so there is a small unavoidable race with any other
// process grabbing the same port between the Close and the subprocess's
// bind; this is the standard Go testing idiom for "give me a free port".
func freeLoopbackAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("close probe listener: %v", err)
	}
	return addr
}

// waitForListening polls addr with plain TCP dials until one succeeds or the
// timeout expires, returning whether the server came up in time.
func waitForListening(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// TestServerStartSignalExit builds the binary, starts it, confirms the HTTP
// listener comes up, sends SIGTERM, and confirms the process exits cleanly
// within a bounded window (E-G4-1 acceptance criteria).
func TestServerStartSignalExit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test under -short")
	}

	binDir := t.TempDir()
	bin := buildServerBinary(t, binDir)

	dbPath := filepath.Join(t.TempDir(), "test.db")
	addr := freeLoopbackAddr(t)

	cmd := exec.Command(bin, "--db", dbPath, "--addr", addr)
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		t.Fatalf("start awis-server: %v", err)
	}
	_ = pw.Close()
	defer func() { _ = pr.Close() }()

	if !waitForListening(addr, 10*time.Second) {
		_ = cmd.Process.Kill()
		t.Fatalf("awis-server did not start listening on %s within 10s", addr)
	}

	// Sanity: "/" now serves the GUI Phase 1 frontend's embedded index.html
	// (static.go) rather than 404ing — the router was wired to the real
	// internal/api handler plus the static asset handler in the same
	// session that added the frontend. See TestServerServesEmbeddedFrontend
	// for a more specific assertion on the embedded content itself.
	resp, err := http.Get(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("GET /: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / status = %d, want %d (embedded frontend)", resp.StatusCode, http.StatusOK)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("send SIGTERM: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("awis-server exited with error after SIGTERM (want clean exit): %v", err)
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("awis-server did not exit within 10s of SIGTERM (shutdown hang / possible goroutine leak)")
	}
}
