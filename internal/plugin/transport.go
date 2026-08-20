package plugin

// transport.go — long-lived NDJSON JSON-RPC 2.0 process handle (M12-C3, T8).
//
// Protocol: TDS-05 docs/PLUGIN_PROTOCOL.md
// Pattern: process-group spawn/kill and tailWriter adapted from
//   internal/runner/subprocess/subprocess.go (coordinate: subprocess.killGroup,
//   subprocess.tailWriter) — different protocol (long-lived vs one-shot).
//
// TRACEABILITY: T8 (JSON-RPC 2.0 NDJSON transport, long-lived, id-correlated).

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"
)

const (
	// stderrTailMax is the maximum stderr bytes captured (TDS-05 §1).
	stderrTailMax = 4 * 1024
)

// rpcRequest is a JSON-RPC 2.0 request object sent to a plugin.
// id is omitted for notifications (shutdown).
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// rpcResponse is a JSON-RPC 2.0 response object received from a plugin.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError is the error sub-object in a JSON-RPC 2.0 error response (TDS-05 §3).
type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// rpcErrorData is the optional data block within rpcError.
type rpcErrorData struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// handshakeParams is the params block for the handshake method (TDS-05 §2).
type handshakeParams struct {
	ProtocolVersion string `json:"protocol_version"`
}

// handshakeResult is the result block of a successful handshake response (TDS-05 §2).
type handshakeResult struct {
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Capabilities []handshakeCapEntry `json:"capabilities"`
}

// handshakeCapEntry is one capability entry in the handshake result.
type handshakeCapEntry struct {
	ID string `json:"id"`
}

// executeParams is the params block for the execute method (TDS-05 §3).
type executeParams struct {
	Capability string         `json:"capability"`
	StepID     string         `json:"step_id"`
	Inputs     map[string]any `json:"inputs"`
	TimeoutMS  int64          `json:"timeout_ms"`
}

// executeResult is the result block of a successful execute response (TDS-05 §3).
type executeResult struct {
	Outputs    map[string]any `json:"outputs"`
	DurationMS int64          `json:"duration_ms"`
}

// processHandle is a long-lived plugin process (TDS-05 §1 long-lived model).
// One instance is created per spawn; destroyed on crash or shutdown.
type processHandle struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	reader     *bufio.Reader
	stderrBuf  *bytes.Buffer
	reqCounter int // monotonic per process; protected by manager mutex
}

// spawnPlugin starts the plugin process from the manifest runtime config.
// env is the full environment to set (manifest env + PATH ONLY; NFR-S-02).
// Process group is set (Setpgid) so SIGKILL reaps all descendants.
//
// Pattern: subprocess.go cmd.SysProcAttr + StdinPipe/StdoutPipe (coordinate:
//
//	internal/runner/subprocess/subprocess.go lines 126–143).
func spawnPlugin(command string, args []string, env []string) (*processHandle, error) {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = env // NFR-S-02: only manifest env + PATH

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("plugin: StdinPipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("plugin: StdoutPipe: %w", err)
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &tailWriter{buf: &stderrBuf, max: stderrTailMax}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("plugin: spawn %q: %w", command, err)
	}

	return &processHandle{
		cmd:       cmd,
		stdin:     stdinPipe,
		reader:    bufio.NewReader(stdoutPipe),
		stderrBuf: &stderrBuf,
	}, nil
}

// nextReqID returns the next monotonic request id string ("req-<n>") and
// increments the counter. Must be called under the manager mutex.
func (h *processHandle) nextReqID() string {
	h.reqCounter++
	return fmt.Sprintf("req-%d", h.reqCounter)
}

// sendRequest serialises req as a single NDJSON line to the plugin's stdin.
func (h *processHandle) sendRequest(req rpcRequest) error {
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("plugin: marshal request: %w", err)
	}
	b = append(b, '\n')
	_, err = h.stdin.Write(b)
	return err
}

// sendNotification sends a JSON-RPC notification (no id) (TDS-05 §4).
func (h *processHandle) sendNotification(method string) error {
	req := rpcRequest{JSONRPC: "2.0", Method: method}
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("plugin: marshal notification: %w", err)
	}
	b = append(b, '\n')
	_, err = h.stdin.Write(b)
	return err
}

// readResponse reads one NDJSON line from stdout and unmarshals it.
// Returns the raw response. Caller is responsible for id / jsonrpc validation.
func (h *processHandle) readResponse() (rpcResponse, error) {
	line, err := h.reader.ReadBytes('\n')
	if err != nil {
		return rpcResponse{}, fmt.Errorf("plugin: read stdout: %w", err)
	}
	var resp rpcResponse
	if err := json.Unmarshal(bytes.TrimRight(line, "\n"), &resp); err != nil {
		return rpcResponse{}, fmt.Errorf("plugin: unmarshal response: %w", err)
	}
	return resp, nil
}

// doHandshake sends the handshake request and validates the response against
// the registered manifest (TDS-05 §2). Must be called under the manager mutex
// immediately after spawn. Returns an error (counts as crash) on any failure.
func (h *processHandle) doHandshake(ctx context.Context, timeout time.Duration, manifest *Manifest) error {
	id := h.nextReqID()
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "handshake",
		Params:  handshakeParams{ProtocolVersion: "awis-plugin/1"},
	}
	if err := h.sendRequest(req); err != nil {
		return fmt.Errorf("handshake send: %w", err)
	}

	// Read response with deadline.
	type readResult struct {
		resp rpcResponse
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		resp, err := h.readResponse()
		ch <- readResult{resp, err}
	}()

	deadline := time.After(timeout)
	select {
	case <-deadline:
		return fmt.Errorf("handshake timeout after %s", timeout)
	case <-ctx.Done():
		return fmt.Errorf("handshake context cancelled: %w", ctx.Err())
	case r := <-ch:
		if r.err != nil {
			return fmt.Errorf("handshake read: %w", r.err)
		}
		return validateHandshakeResponse(r.resp, id, manifest)
	}
}

// validateHandshakeResponse checks the response fields per TDS-05 §2.
func validateHandshakeResponse(resp rpcResponse, expectedID string, manifest *Manifest) error {
	if resp.JSONRPC != "2.0" {
		return fmt.Errorf("plugin_handshake_error: jsonrpc field missing or wrong: %q", resp.JSONRPC)
	}
	if resp.ID != expectedID {
		return fmt.Errorf("plugin_handshake_error: id mismatch: got %q, want %q", resp.ID, expectedID)
	}
	if resp.Error != nil {
		return fmt.Errorf("plugin_handshake_error: plugin returned error: %s", resp.Error.Message)
	}
	if resp.Result == nil {
		return fmt.Errorf("plugin_handshake_error: missing result")
	}
	var result handshakeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("plugin_handshake_error: unmarshal result: %w", err)
	}
	if result.Name != manifest.Name {
		return fmt.Errorf("plugin_handshake_error: name mismatch: got %q, want %q", result.Name, manifest.Name)
	}
	if result.Version != manifest.Version {
		return fmt.Errorf("plugin_handshake_error: version mismatch: got %q, want %q", result.Version, manifest.Version)
	}
	// Capability-id set must be a superset of the registered ids (TDS-05 §2).
	registered := make(map[string]bool, len(manifest.Capabilities))
	for _, c := range manifest.Capabilities {
		registered[c.ID] = true
	}
	for _, c := range result.Capabilities {
		delete(registered, c.ID)
	}
	if len(registered) > 0 {
		// Build a list of missing ids for the error message.
		missing := make([]string, 0, len(registered))
		for id := range registered {
			missing = append(missing, id)
		}
		return fmt.Errorf("plugin_handshake_error: capability ids not in plugin response: %v", missing)
	}
	return nil
}

// doExecute sends an execute request and reads the response.
// Returns the raw rpcResponse. The deadline channel is provided by the caller.
// This MUST be called under the manager mutex (serialized per plugin, V1 pin).
func (h *processHandle) doExecute(ctx context.Context, id string, params executeParams) (rpcResponse, error) {
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "execute",
		Params:  params,
	}
	if err := h.sendRequest(req); err != nil {
		return rpcResponse{}, fmt.Errorf("execute send: %w", err)
	}

	type readResult struct {
		resp rpcResponse
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		resp, err := h.readResponse()
		ch <- readResult{resp, err}
	}()

	select {
	case <-ctx.Done():
		// Context cancelled/deadline exceeded; return so caller can kill.
		return rpcResponse{}, ctx.Err()
	case r := <-ch:
		return r.resp, r.err
	}
}

// killGroup sends SIGKILL to the entire process group of the plugin.
// Pattern: subprocess.killGroup (coordinate:
//
//	internal/runner/subprocess/subprocess.go line 286–292).
func (h *processHandle) killGroup() {
	if h.cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-h.cmd.Process.Pid, syscall.SIGKILL)
}

// pid returns the OS process id, or 0 if not started.
func (h *processHandle) pid() int {
	if h.cmd.Process == nil {
		return 0
	}
	return h.cmd.Process.Pid
}

// tailWriter is an io.Writer that keeps only the last max bytes in buf.
// Pattern: subprocess.tailWriter (coordinate:
//
//	internal/runner/subprocess/subprocess.go lines 309–327).
type tailWriter struct {
	buf *bytes.Buffer
	max int
}

func (w *tailWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.buf.Write(p)
	if w.buf.Len() > w.max {
		b := w.buf.Bytes()
		trimmed := make([]byte, w.max)
		copy(trimmed, b[len(b)-w.max:])
		w.buf.Reset()
		w.buf.Write(trimmed)
	}
	return n, nil
}
