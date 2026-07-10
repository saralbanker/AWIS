package subprocess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/awis/awis/internal/core"
)

const (
	// protocol is the wire protocol string (TDS-04 §2).
	protocol = "awis-subprocess/1"
	// stderrTailMax is the maximum number of bytes captured from stderr (TDS-04 §1).
	stderrTailMax = 4 * 1024
	// deadlineFormat is the RFC3339Nano format for the deadline field (TDS-04 §2).
	// Go's time.RFC3339Nano trims trailing zeros; the golden files keep all 9 digits,
	// so we use a fixed 9-digit nanosecond format matching the golden wire format.
	deadlineFormat = "2006-01-02T15:04:05.000000000Z07:00"
)

// SubprocessRunner implements engine.Runner for core.StepTypeSubprocess.
// Each Run call spawns exactly one child process (TDS-04 §1 one-shot model):
// writes a JSON request envelope to stdin, closes stdin, reads a JSON response
// envelope from stdout, and maps the outcome to StepResult or *core.StepError
// per the error mapping table in docs/SUBPROCESS_PROTOCOL.md §7.
type SubprocessRunner struct{}

// New returns a SubprocessRunner. It requires no configuration; the handler
// binary is determined at run-time from step.Handler.
func New() *SubprocessRunner { return &SubprocessRunner{} }

// requestEnvelope is the JSON object sent to the child's stdin (TDS-04 §2).
// Field names and ordering are frozen (CE-pinned).
type requestEnvelope struct {
	Protocol   string         `json:"protocol"`
	Handler    string         `json:"handler"`
	InstanceID string         `json:"instance_id"`
	StepID     string         `json:"step_id"`
	Attempt    int            `json:"attempt"`
	Inputs     map[string]any `json:"inputs"`
	Deadline   string         `json:"deadline,omitempty"`
}

// responseEnvelope is the JSON object read from the child's stdout (TDS-04 §3, §4).
type responseEnvelope struct {
	Protocol string              `json:"protocol"`
	Outputs  map[string]any      `json:"outputs,omitempty"`
	Error    *responseErrorBlock `json:"error,omitempty"`
}

// responseErrorBlock is the error sub-object inside a response envelope (TDS-04 §4).
type responseErrorBlock struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Run executes the subprocess step. It implements engine.Runner.
//
// Timeout handling mirrors internal/runner/native (coordinate: internal/runner/native):
// step.Timeout is parsed with time.ParseDuration; when positive, a deadline is
// computed and applied. If the context already carries an earlier deadline, the
// earlier of the two wins.
func (r *SubprocessRunner) Run(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.StepError) {
	// -- Timeout: mirror native runner (coordinate: internal/runner/native) --
	// Apply the step timeout as a context deadline when it parses to a positive
	// duration. Use the earlier of step.Timeout and any pre-existing ctx deadline.
	if d := parseTimeout(step.Timeout); d > 0 {
		deadline := time.Now().Add(d)
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
		// Propagate computed deadline into the StepContext for the request envelope.
		sc.Deadline = deadline
	}
	// If the context already carries an earlier deadline, use that for the envelope.
	if ctxDL, ok := ctx.Deadline(); ok {
		if sc.Deadline.IsZero() || ctxDL.Before(sc.Deadline) {
			sc.Deadline = ctxDL
		}
	}

	// -- Build request envelope (TDS-04 §2) --
	env := requestEnvelope{
		Protocol:   protocol,
		Handler:    string(step.Handler),
		InstanceID: sc.InstanceID,
		StepID:     sc.StepID,
		Attempt:    sc.Attempt,
		Inputs:     sc.Inputs,
	}
	if !sc.Deadline.IsZero() {
		env.Deadline = sc.Deadline.UTC().Format(deadlineFormat)
	}
	reqBytes, err := json.Marshal(env)
	if err != nil {
		return core.StepResult{}, &core.StepError{
			Code:    "spawn_error",
			Message: fmt.Sprintf("subprocess: marshal request envelope: %v", err),
		}
	}
	reqBytes = append(reqBytes, '\n')

	// -- Parse argv: whitespace-split, no shell (TDS-04 §5) --
	argv := strings.Fields(string(step.Handler))
	if len(argv) == 0 {
		return core.StepResult{}, &core.StepError{
			Code:    "spawn_error",
			Message: "subprocess: step.handler is empty; cannot build argv",
		}
	}

	// -- Spawn child in a new process group (TDS-04 §6 SIGKILL reaps descendants) --
	// We do NOT use exec.CommandContext here so that we can kill the entire
	// process group (not just the leader) on timeout (TDS-04 §6). The context
	// is monitored via a goroutine that kills the group and cancels our internal
	// done channel.
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return core.StepResult{}, &core.StepError{
			Code:    "spawn_error",
			Message: fmt.Sprintf("subprocess: StdinPipe: %v", err),
		}
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return core.StepResult{}, &core.StepError{
			Code:    "spawn_error",
			Message: fmt.Sprintf("subprocess: StdoutPipe: %v", err),
		}
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &tailWriter{buf: &stderrBuf, max: stderrTailMax}

	if err := cmd.Start(); err != nil {
		return core.StepResult{}, &core.StepError{
			Code:    "spawn_error",
			Message: fmt.Sprintf("subprocess: Start: %v", err),
		}
	}

	// Watch the context in a goroutine: when it is cancelled or deadline-exceeded,
	// kill the process group so the child and all descendants are reaped, which
	// also unblocks the io.ReadAll below (TDS-04 §6).
	ctxDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			killGroup(cmd)
		case <-ctxDone:
		}
	}()

	// -- Write request to stdin and close it (TDS-04 §1) --
	_, writeErr := stdinPipe.Write(reqBytes)
	// Close stdin to signal EOF to the child (TDS-04 §1). The close error is
	// suppressed: if Write failed, writeErr already captures the failure; if
	// Write succeeded, a close error on a fully-written pipe is not actionable.
	_ = stdinPipe.Close()
	if writeErr != nil {
		close(ctxDone)
		killGroup(cmd)
		_ = cmd.Wait()
		return core.StepResult{}, &core.StepError{
			Code:    "spawn_error",
			Message: fmt.Sprintf("subprocess: write to stdin: %v", writeErr),
		}
	}

	// -- Read response from stdout (TDS-04 §1) --
	stdoutBytes, _ := io.ReadAll(stdoutPipe)

	// Wait for the child to exit.
	waitErr := cmd.Wait()
	// Signal the context watcher that we are done (no kill needed).
	close(ctxDone)

	// Determine if the context timed out.
	if ctx.Err() == context.DeadlineExceeded {
		return core.StepResult{}, &core.StepError{
			Code:    "timeout",
			Message: fmt.Sprintf("step %q exceeded its timeout", step.ID),
		}
	}

	// Determine exit code.
	exitCode := 0
	if waitErr != nil {
		if ee, ok := waitErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		}
	}

	stderrTail := stderrBuf.String()

	// -- Parse stdout as a response envelope (TDS-04 §3, §4, §7) --
	var resp responseEnvelope
	parseErr := json.Unmarshal(bytes.TrimSpace(stdoutBytes), &resp)

	// Error-mapping order (TDS-04 §7):
	//
	// 1. Nonzero exit WITH a valid error envelope → envelope wins.
	// 2. Nonzero exit WITHOUT a valid envelope → subprocess_error.
	// 3. Zero exit with invalid JSON or wrong protocol value → protocol_error.
	// 4. Zero exit with valid success envelope → success.

	if exitCode != 0 {
		// Check if we have a valid error envelope (envelope-wins rule, TDS-04 §7).
		if parseErr == nil && resp.Protocol == protocol && resp.Error != nil {
			code := resp.Error.Code
			if code == "" {
				code = "handler_error"
			}
			return core.StepResult{}, &core.StepError{
				Code:    code,
				Message: resp.Error.Message,
				Details: resp.Error.Details,
			}
		}
		// Nonzero exit without a valid error envelope → subprocess_error.
		return core.StepResult{}, &core.StepError{
			Code:    "subprocess_error",
			Message: fmt.Sprintf("subprocess exited with code %d", exitCode),
			Details: map[string]any{
				"exit_code": exitCode,
				"stderr":    stderrTail,
			},
		}
	}

	// Zero exit: validate the envelope.
	if parseErr != nil || resp.Protocol != protocol {
		return core.StepResult{}, &core.StepError{
			Code:    "protocol_error",
			Message: buildProtocolErrorMessage(parseErr, resp.Protocol),
			Details: map[string]any{
				"stderr":    stderrTail,
				"exit_code": exitCode,
			},
		}
	}

	// Zero exit with a valid error envelope → also envelope wins (TDS-04 §7: the
	// envelope's code/message/details are used verbatim).
	if resp.Error != nil {
		code := resp.Error.Code
		if code == "" {
			code = "handler_error"
		}
		return core.StepResult{}, &core.StepError{
			Code:    code,
			Message: resp.Error.Message,
			Details: resp.Error.Details,
		}
	}

	// -- Success (TDS-04 §3) --
	outputs := resp.Outputs
	if outputs == nil {
		outputs = map[string]any{}
	}
	return core.StepResult{Outputs: outputs}, nil
}

// buildProtocolErrorMessage constructs a human-readable message for protocol_error.
func buildProtocolErrorMessage(parseErr error, gotProtocol string) string {
	if parseErr != nil {
		return fmt.Sprintf("subprocess: stdout is not valid JSON: %v", parseErr)
	}
	return fmt.Sprintf("subprocess: wrong protocol value %q (want %q)", gotProtocol, protocol)
}

// killGroup sends SIGKILL to the entire process group of cmd. It is a no-op if
// the process has already exited or has not been started. Used to ensure all
// descendants are reaped on timeout (TDS-04 §6). Unix-only; acceptable per QG-1.
func killGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	// Negative pid targets the process group (Unix-only; acceptable per QG-1).
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

// parseTimeout parses a core.Duration into a time.Duration. An empty or
// unparseable value yields 0 ("no deadline"). Mirrors internal/runner/native
// (coordinate: internal/runner/native).
func parseTimeout(d core.Duration) time.Duration {
	s := string(d)
	if s == "" {
		return 0
	}
	parsed, err := time.ParseDuration(s)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

// tailWriter is an io.Writer that keeps only the last max bytes written to buf.
type tailWriter struct {
	buf *bytes.Buffer
	max int
}

func (w *tailWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.buf.Write(p)
	if w.buf.Len() > w.max {
		// Trim to the last max bytes.
		b := w.buf.Bytes()
		trimmed := make([]byte, w.max)
		copy(trimmed, b[len(b)-w.max:])
		w.buf.Reset()
		w.buf.Write(trimmed)
	}
	return n, nil
}
