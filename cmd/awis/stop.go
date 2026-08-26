package main

// stop.go — 'awis stop' command (TDS-07 §4; M14-C2 T4).
//
// Reads <data-dir>/awis.pid, sends SIGTERM, polls process liveness ≤10s.
// On stale PID (process gone): cleans up file, exits 0 with note.
// On timeout: reports error per TDS-07, exits 1.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func init() {
	commands["stop"] = command{fn: runStop, summary: "Gracefully stop the runtime"}
}

// stopOutput is the JSON schema for 'awis stop --json' (TDS-07 §4).
type stopOutput struct {
	PID       int  `json:"pid"`
	Stopped   bool `json:"stopped"`
	ElapsedMs int  `json:"elapsed_ms"`
}

func runStop(args []string) {
	fs := newFlagSet("stop")
	mustParse(fs, args)

	dataDir := globalDataDir
	pidPath := filepath.Join(dataDir, "awis.pid")

	// Read PID file.
	raw, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			fail(1, "runtime not started (no PID file found)", pidPath, "awis start")
		}
		fail(1, fmt.Sprintf("stop: cannot read PID file: %s", err), pidPath, "check file permissions")
	}

	pidStr := strings.TrimSpace(string(raw))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fail(1, fmt.Sprintf("stop: PID file contains invalid value %q", pidStr), pidPath, "remove the file manually and restart the runtime")
	}

	// Check if process exists.
	proc, err := os.FindProcess(pid)
	if err != nil {
		// On Linux, FindProcess always succeeds for any PID; error means bad PID.
		_ = os.Remove(pidPath)
		fail(1, fmt.Sprintf("stop: process %d not found (stale PID file)", pid), pidPath, "file has been removed; run 'awis start' to start a new instance")
	}

	// Send SIGTERM.
	if !globalJSON {
		fmt.Printf("Stopping AWIS (PID %d)...\n", pid)
	}
	start := time.Now()
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Process is already gone — clean up stale PID file.
		_ = os.Remove(pidPath)
		elapsedMs := int(time.Since(start).Milliseconds())
		if globalJSON {
			out := stopOutput{PID: pid, Stopped: true, ElapsedMs: elapsedMs}
			emitJSON(out)
			return
		}
		fmt.Printf("stopped. (process was already gone; PID file cleaned up)\n")
		return
	}

	// Poll process liveness ≤10s.
	const maxWait = 10 * time.Second
	const pollInterval = 100 * time.Millisecond
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)
		// Kill -0 (signal 0) tests whether the process is alive without sending a signal.
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			// Process is gone.
			_ = os.Remove(pidPath)
			elapsedMs := int(time.Since(start).Milliseconds())
			if globalJSON {
				out := stopOutput{PID: pid, Stopped: true, ElapsedMs: elapsedMs}
				emitJSON(out)
				return
			}
			fmt.Printf("stopped.\n")
			return
		}
	}

	// Timeout.
	elapsedMs := int(time.Since(start).Milliseconds())
	if globalJSON {
		out := stopOutput{PID: pid, Stopped: false, ElapsedMs: elapsedMs}
		emitJSON(out)
		os.Exit(1)
	}
	// TDS-07 §4 timeout format (verbatim):
	// awis: runtime did not stop within 10s (PID 12345)
	//   Where:    <data-dir>/awis.pid
	//   What now: kill -9 12345
	fmt.Fprintf(os.Stderr, "awis: runtime did not stop within 10s (PID %d)\n", pid)
	fmt.Fprintf(os.Stderr, "  Where:    %s\n", pidPath)
	fmt.Fprintf(os.Stderr, "  What now: kill -9 %d\n", pid)
	os.Exit(1)
}
