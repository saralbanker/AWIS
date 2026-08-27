package plugin

// manager.go — CE-pinned lifecycle FSM manager (M12-C3, T7).
//
// Protocol authority: TDS-05 docs/PLUGIN_PROTOCOL.md §7, §5, §8.
// IMPLEMENTATION_SPEC.md "CE-pinned lifecycle FSM" section.
//
// States: REGISTERED, SPAWNING, HANDSHAKING, ACTIVE, IDLE, TERMINATED, FAILED.
// DB status stays coarse: registered | active | failed.
// In-memory IDLE keeps DB status active (SPEC disposition).
//
// TRACEABILITY: T7 (lifecycle FSM manager), T10 (§20 checkpoint via crash→restart).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// pluginState is the in-memory FSM state for one registered plugin.
type pluginState int

const (
	stateRegistered  pluginState = iota // REGISTERED: manifest parsed, not yet spawned
	stateSpawning                       // SPAWNING: exec in progress
	stateHandshaking                    // HANDSHAKING: handshake request in flight
	stateActive                         // ACTIVE: ready to serve calls
	stateIdle                           // IDLE: process killed due to idle timeout; DB stays active
	stateTerminated                     // TERMINATED: cleanly shut down
	stateFailed                         // FAILED: crash budget exhausted; permanent until re-register
)

// crashLimit is the maximum consecutive crashes before a plugin enters FAILED.
// TDS-05 §7: "Counter > 3 → FAILED". So >3 means 4+ crashes.
const crashLimit = 3

// defaultHandshakeTimeout is the default 5s handshake deadline (TDS-05 §2).
const defaultHandshakeTimeout = 5 * time.Second

// defaultShutdownGrace is the default ≤2s wait after shutdown notification (TDS-05 §4).
const defaultShutdownGrace = 2 * time.Second

// ManagerConfig configures the Manager. Zero fields receive defaults.
type ManagerConfig struct {
	// Clock is the injectable time source (IMP §3 determinism rule); nil ⇒ time.Now.
	Clock func() time.Time
	// IdleCheckInterval is the resolution of the idle watchdog timer (seam for fast tests).
	// If zero, the plugin's idle_timeout_s is used directly.
	IdleCheckInterval time.Duration
	// HandshakeTimeout is the deadline for the handshake exchange; zero ⇒ 5s (TDS-05 §2).
	HandshakeTimeout time.Duration
	// ShutdownGrace is the wait after the shutdown notification; zero ⇒ 2s (TDS-05 §4).
	ShutdownGrace time.Duration
}

// pluginEntry holds the full runtime state for one registered plugin.
// Two locks are used:
//   - callMu: serializes calls per plugin (V1 pin; held for the duration of a
//     doExecute I/O exchange to prevent concurrent calls while allowing state
//     queries like PluginPID to proceed without blocking on I/O).
//   - mu: protects all fields below (state, handle, crashCount, etc.).
//     Held only for state transitions; NOT held during blocking I/O.
type pluginEntry struct {
	callMu       sync.Mutex // serializes calls (V1 pin: one call at a time per plugin)
	mu           sync.Mutex // protects all fields below
	manifest     *Manifest
	state        pluginState
	handle       *processHandle
	crashCount   int           // consecutive crash counter
	monitorDone  chan struct{} // closed when the monitor goroutine exits
	intentional  bool          // set before a controlled kill so monitor skips crash counting
	idleTimer    *time.Timer
	shutdownOnce sync.Once
}

// Manager manages the lifecycle of all registered plugins (TDS-05 §7).
// NewManager is the constructor; Register / Call / Shutdown are the API.
type Manager struct {
	store storage.PluginStore
	cfg   ManagerConfig
	clock func() time.Time

	mu      sync.Mutex
	plugins map[string]*pluginEntry // keyed by manifest.Name
	closed  bool

	// statusMu/statusQueue/statusWake/statusDone/statusWG implement the B-18
	// ordered status writer: a single background goroutine applies
	// plugins.status writes to the store strictly in the order they were
	// queued, so a slow write can never land after — and silently overwrite —
	// a write that was queued later (see setPluginStatusAsync).
	statusMu    sync.Mutex
	statusQueue []pluginStatusChange
	statusWake  chan struct{}
	statusDone  chan struct{}
	statusWG    sync.WaitGroup
}

// pluginStatusChange is one pending write to the plugins.status column,
// queued by setPluginStatusAsync and applied in order by statusWriterLoop
// (B-18).
type pluginStatusChange struct {
	name   string
	status string
}

// NewManager constructs a Manager backed by the given PluginStore.
// cfg fields with zero values receive the documented defaults.
func NewManager(store storage.PluginStore, cfg ManagerConfig) *Manager {
	clk := cfg.Clock
	if clk == nil {
		clk = time.Now
	}
	if cfg.HandshakeTimeout == 0 {
		cfg.HandshakeTimeout = defaultHandshakeTimeout
	}
	if cfg.ShutdownGrace == 0 {
		cfg.ShutdownGrace = defaultShutdownGrace
	}
	m := &Manager{
		store:      store,
		cfg:        cfg,
		clock:      clk,
		plugins:    make(map[string]*pluginEntry),
		statusWake: make(chan struct{}, 1),
		statusDone: make(chan struct{}),
	}
	m.statusWG.Add(1)
	go m.statusWriterLoop()
	return m
}

// setPluginStatusAsync queues a plugins.status write for name without
// blocking the caller — spawnAndHandshakeLocked, monitorProcess and
// handleCrashLocked all call this while holding entry.mu, and storage I/O
// must not run under that lock. The queue append itself is synchronous
// (never a fire-and-forget goroutine), so the order writes are queued in
// exactly matches the order the FSM decided them in, per plugin name (each of
// those call sites already serializes on entry.mu). statusWriterLoop is the
// single consumer that applies queued writes to the store in that same order
// and logs — never discards — a failed write (B-18: independent
// "go func(){ store.SetPluginStatus(...) }()" goroutines raced on I/O
// duration, so a write queued earlier could still land in the DB after one
// queued later, leaving a stale status such as "active" after a subsequent
// "failed").
func (m *Manager) setPluginStatusAsync(name, status string) {
	m.statusMu.Lock()
	m.statusQueue = append(m.statusQueue, pluginStatusChange{name: name, status: status})
	m.statusMu.Unlock()
	select {
	case m.statusWake <- struct{}{}:
	default:
	}
}

// statusWriterLoop is the single ordered consumer for queued plugins.status
// writes (B-18). It runs until Shutdown closes statusDone, and performs one
// final drain afterward so nothing queued just before close is lost.
func (m *Manager) statusWriterLoop() {
	defer m.statusWG.Done()
	for {
		m.drainStatusQueue()
		select {
		case <-m.statusDone:
			m.drainStatusQueue()
			return
		case <-m.statusWake:
		}
	}
}

// drainStatusQueue applies every currently queued status change to the store,
// in FIFO order, one at a time. Failures are logged, never swallowed (B-18).
func (m *Manager) drainStatusQueue() {
	for {
		m.statusMu.Lock()
		if len(m.statusQueue) == 0 {
			m.statusMu.Unlock()
			return
		}
		change := m.statusQueue[0]
		m.statusQueue = m.statusQueue[1:]
		m.statusMu.Unlock()

		if err := m.store.SetPluginStatus(context.Background(), change.name, change.status); err != nil {
			log.Printf("plugin: SetPluginStatus(%q, %q) failed: %v", change.name, change.status, err)
		}
	}
}

// Register parses the manifest at manifestPath, persists it to the store, and
// sets the plugin's in-memory state to REGISTERED (TDS-05 §7 Registration).
// Re-registering the same name is an upsert (SPEC disposition).
func (m *Manager) Register(ctx context.Context, manifestPath string) error {
	manifest, err := ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("plugin.Manager.Register: %w", err)
	}

	// Serialize manifest to JSON for storage.
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("plugin.Manager.Register: marshal manifest: %w", err)
	}

	capIDs := make([]string, len(manifest.Capabilities))
	for i, c := range manifest.Capabilities {
		capIDs[i] = c.ID
	}

	// Persist to storage + audit (F-4 PluginRegistered in same tx, C2).
	if err := m.store.RegisterPlugin(ctx, manifest.Name, manifest.Version, string(manifestBytes), capIDs); err != nil {
		return fmt.Errorf("plugin.Manager.Register: store: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.plugins[manifest.Name]; ok {
		// Re-registration: stop any idle timer, kill existing process if alive.
		existing.mu.Lock()
		m.killEntryLocked(existing)
		existing.manifest = manifest
		existing.state = stateRegistered
		existing.crashCount = 0
		existing.mu.Unlock()
	} else {
		m.plugins[manifest.Name] = &pluginEntry{
			manifest: manifest,
			state:    stateRegistered,
		}
	}
	return nil
}

// Call resolves handlerRef to a capability id, ensures the plugin is ACTIVE,
// and dispatches an execute call (TDS-05 §3, §5, §7).
//
// handlerRef is either a capability id (exact match) or a plugin name — in the
// latter case resolution uses the unique input-key-set rule (TDS-05 §6).
//
// stepTimeout is the step-level timeout (may be 0 if not set). The effective
// timeout is min(ctx deadline, stepTimeout, capability timeout_ms) per TDS-05 §5.
//
// Locking model:
//   - entry.callMu: held for the full duration of a call to serialize calls
//     per plugin (V1 pin), released during I/O so state queries proceed.
//   - entry.mu: held only for state transitions (not during blocking I/O).
func (m *Manager) Call(
	ctx context.Context,
	handlerRef string,
	stepID string,
	inputs map[string]any,
	stepTimeout core.Duration,
) (map[string]any, *core.StepError) {
	// Resolve to (pluginName, capabilityID).
	pluginName, capID, err := m.resolveHandler(ctx, handlerRef, inputs)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	entry, ok := m.plugins[pluginName]
	m.mu.Unlock()
	if !ok {
		return nil, &core.StepError{Code: "plugin_not_found", Message: fmt.Sprintf("plugin %q not registered", pluginName)}
	}

	// Serialize calls per plugin (V1 pin: one call at a time per plugin).
	entry.callMu.Lock()
	defer entry.callMu.Unlock()

	// --- Phase 1: Ensure ACTIVE (under entry.mu) ---
	entry.mu.Lock()
	spawnErr := m.ensureActiveLocked(ctx, entry)
	entry.mu.Unlock()
	if spawnErr != nil {
		return nil, spawnErr
	}

	// --- Phase 2: Prepare call parameters (under entry.mu) ---
	entry.mu.Lock()
	effectiveTimeout := m.computeEffectiveTimeout(ctx, entry.manifest, capID, stepTimeout)
	m.stopIdleTimerLocked(entry) // Invariant 3: idle killer must not fire while call is in flight
	handle := entry.handle
	reqID := handle.nextReqID()
	entry.mu.Unlock()

	params := executeParams{
		Capability: capID,
		StepID:     stepID,
		Inputs:     inputs,
		TimeoutMS:  int64(effectiveTimeout.Milliseconds()),
	}

	// Arm a per-call context with the effective deadline.
	callCtx := ctx
	var callCancel context.CancelFunc
	if effectiveTimeout > 0 {
		callCtx, callCancel = context.WithTimeout(ctx, effectiveTimeout)
		defer callCancel()
	}

	// --- Phase 3: I/O (WITHOUT entry.mu — allows PluginPID and Shutdown to proceed) ---
	resp, execErr := handle.doExecute(callCtx, reqID, params)

	// --- Phase 4: Post-I/O state update (under entry.mu) ---
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if execErr != nil {
		// Context deadline exceeded = controlled kill (TDS-05 §5, §8: timeout,
		// does NOT count as crash).
		if errors.Is(execErr, context.DeadlineExceeded) || errors.Is(execErr, context.Canceled) {
			// Set intentional flag before killing so the monitor skips crash counting.
			entry.intentional = true
			handle.killGroup()
			// Wait for the monitor goroutine to finish (it will see intentional=true).
			if entry.monitorDone != nil {
				done := entry.monitorDone
				entry.mu.Unlock()
				select {
				case <-done:
				case <-time.After(500 * time.Millisecond):
				}
				entry.mu.Lock()
			}
			entry.intentional = false
			// Only reset state if this handle is still current (monitor might have
			// replaced it already).
			if entry.handle == handle {
				entry.handle = nil
				entry.state = stateRegistered
			}
			m.restartIdleTimerLocked(entry)
			return nil, &core.StepError{Code: "timeout", Message: fmt.Sprintf("plugin %q step %q exceeded timeout", pluginName, stepID)}
		}
		// Read error = unexpected crash. The monitor goroutine may have already
		// bumped crashCount and updated state (entry.handle == nil if so).
		if entry.handle == nil || entry.handle != handle {
			// Monitor already handled this crash.
			if entry.state == stateFailed {
				return nil, &core.StepError{
					Code:    "plugin_failed",
					Message: fmt.Sprintf("plugin %q permanently failed after consecutive crashes", pluginName),
				}
			}
			return nil, &core.StepError{Code: "plugin_crash", Message: fmt.Sprintf("plugin %q crashed", pluginName)}
		}
		return nil, m.handleCrashLocked(entry, pluginName, "execute read")
	}

	// Validate response id and jsonrpc fields (TDS-05 §8: protocol_error = crash-equivalent).
	if resp.JSONRPC != "2.0" || resp.ID != reqID {
		handle.killGroup()
		monDone := entry.monitorDone
		entry.mu.Unlock()
		if monDone != nil {
			select {
			case <-monDone:
			case <-time.After(500 * time.Millisecond):
			}
		}
		entry.mu.Lock()
		if entry.handle == nil || entry.handle != handle {
			if entry.state == stateFailed {
				return nil, &core.StepError{Code: "plugin_failed", Message: fmt.Sprintf("plugin %q permanently failed", pluginName)}
			}
			return nil, &core.StepError{Code: "protocol_error", Message: "plugin: id or jsonrpc mismatch"}
		}
		return nil, m.handleCrashLocked(entry, pluginName, "protocol: id or jsonrpc mismatch")
	}

	// JSON-RPC error response (TDS-05 §3, §8): process still alive.
	if resp.Error != nil {
		m.restartIdleTimerLocked(entry)
		errCode := "plugin_error"
		if resp.Error.Data != nil {
			var d rpcErrorData
			if jsonErr := json.Unmarshal(resp.Error.Data, &d); jsonErr == nil && d.Code != "" {
				errCode = d.Code
			}
		}
		return nil, &core.StepError{
			Code:    errCode,
			Message: resp.Error.Message,
		}
	}

	// Parse success result.
	if resp.Result == nil {
		m.restartIdleTimerLocked(entry)
		return nil, &core.StepError{Code: "protocol_error", Message: "plugin returned null result"}
	}
	var result executeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		handle.killGroup()
		return nil, m.handleCrashLocked(entry, pluginName, fmt.Sprintf("protocol: unmarshal execute result: %v", err))
	}

	// SUCCESS: reset crash counter (TDS-05 §7).
	entry.crashCount = 0
	outputs := result.Outputs
	if outputs == nil {
		outputs = map[string]any{}
	}

	m.restartIdleTimerLocked(entry)
	return outputs, nil
}

// Shutdown sends shutdown notification to each active plugin, waits up to
// ShutdownGrace, then SIGKILLs. Idempotent (TDS-05 §7 Shutdown).
// Invariant 5: monitor goroutines never leak past Shutdown.
func (m *Manager) Shutdown(ctx context.Context) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	// Snapshot entries.
	entries := make([]*pluginEntry, 0, len(m.plugins))
	for _, e := range m.plugins {
		entries = append(entries, e)
	}
	m.mu.Unlock()

	for _, entry := range entries {
		m.shutdownEntry(ctx, entry)
	}

	// Stop the B-18 ordered status writer and wait for it to drain, so any
	// status write queued up to this point (e.g. a "failed" from a crash
	// that raced with shutdown) is applied before Shutdown returns rather
	// than lost when the process exits.
	close(m.statusDone)
	m.statusWG.Wait()
}

// shutdownEntry cleanly terminates one plugin entry.
func (m *Manager) shutdownEntry(ctx context.Context, entry *pluginEntry) {
	entry.shutdownOnce.Do(func() {
		entry.mu.Lock()
		defer entry.mu.Unlock()

		m.stopIdleTimerLocked(entry)

		if entry.state != stateActive && entry.state != stateHandshaking && entry.state != stateSpawning {
			entry.state = stateTerminated
			return
		}

		handle := entry.handle
		if handle != nil {
			// Send shutdown notification (TDS-05 §4).
			_ = handle.sendNotification("shutdown")
		}

		// Set intentional so monitor goroutine skips crash counting.
		entry.intentional = true

		// Wait up to ShutdownGrace for the process to exit.
		if entry.monitorDone != nil {
			done := entry.monitorDone
			entry.mu.Unlock()
			select {
			case <-done:
			case <-time.After(m.cfg.ShutdownGrace):
			case <-ctx.Done():
			}
			entry.mu.Lock()
		}

		// SIGKILL the process group to ensure all descendants are reaped (TDS-05 §4).
		if handle != nil {
			handle.killGroup()
		}

		// Wait for monitor goroutine to finish (invariant 5: no leaks).
		if entry.monitorDone != nil {
			done := entry.monitorDone
			entry.mu.Unlock()
			<-done
			entry.mu.Lock()
		}

		entry.state = stateTerminated
		entry.handle = nil
	})
}

// ensureActiveLocked guarantees the plugin is in ACTIVE state, spawning and
// handshaking if necessary (TDS-05 §7 lazy spawn).
// Must be called with entry.mu held.
func (m *Manager) ensureActiveLocked(ctx context.Context, entry *pluginEntry) *core.StepError {
	switch entry.state {
	case stateActive:
		return nil
	case stateFailed:
		return &core.StepError{Code: "plugin_failed", Message: fmt.Sprintf("plugin %q is in FAILED state (crash budget exhausted)", entry.manifest.Name)}
	case stateTerminated:
		return &core.StepError{Code: "plugin_failed", Message: fmt.Sprintf("plugin %q is TERMINATED", entry.manifest.Name)}
	case stateRegistered, stateIdle:
		return m.spawnAndHandshakeLocked(ctx, entry)
	default:
		return &core.StepError{Code: "plugin_error", Message: fmt.Sprintf("plugin %q in unexpected state", entry.manifest.Name)}
	}
}

// spawnAndHandshakeLocked spawns the plugin process and performs the handshake.
// Transitions: REGISTERED/IDLE → SPAWNING → HANDSHAKING → ACTIVE.
// Must be called with entry.mu held.
func (m *Manager) spawnAndHandshakeLocked(ctx context.Context, entry *pluginEntry) *core.StepError {
	manifest := entry.manifest

	// Build env: manifest env + PATH ONLY (NFR-S-02). Relative PYTHONPATH /
	// AWIS_PLUGIN_LIBPATH entries are resolved against manifest.Dir so the
	// shipped manifest is self-sufficient no matter what directory awis is
	// run from (B-21).
	env := buildEnv(manifest.Runtime.Env, manifest.Dir)

	entry.state = stateSpawning
	// manifest.Dir becomes cmd.Dir so a relative runtime.command/args (e.g.
	// "python3 -m git_context_plugin") resolves against the plugin's own
	// directory rather than the awis process's working directory (B-21).
	handle, err := spawnPlugin(manifest.Runtime.Command, manifest.Runtime.Args, env, manifest.Dir)
	if err != nil {
		entry.state = stateRegistered
		return &core.StepError{Code: "spawn_error", Message: fmt.Sprintf("plugin %q: %v", manifest.Name, err)}
	}
	entry.handle = handle

	entry.state = stateHandshaking

	// Start the monitor goroutine BEFORE handshake so unexpected exits are caught.
	monitorDone := make(chan struct{})
	entry.monitorDone = monitorDone
	go m.monitorProcess(entry, handle, monitorDone)

	// Perform handshake (5s deadline, TDS-05 §2).
	// We hold entry.mu; handshake I/O is inside the lock (safe: monitor only signals via monitorDone).
	if err := handle.doHandshake(ctx, m.cfg.HandshakeTimeout, manifest); err != nil {
		// Handshake failure = crash-equivalent (TDS-05 §2, §8: plugin_handshake_error counts as crash).
		// The monitor must NOT additionally count this as a crash (we handle it here).
		entry.intentional = true
		handle.killGroup()
		entry.mu.Unlock()
		select {
		case <-monitorDone:
		case <-time.After(500 * time.Millisecond):
		}
		entry.mu.Lock()
		entry.intentional = false
		entry.handle = nil
		// Count as crash but return plugin_handshake_error (not plugin_crash).
		m.stopIdleTimerLocked(entry)
		entry.crashCount++
		if entry.crashCount > crashLimit {
			entry.state = stateFailed
			m.setPluginStatusAsync(manifest.Name, "failed")
			return &core.StepError{
				Code:    "plugin_failed",
				Message: fmt.Sprintf("plugin %q permanently failed after %d consecutive handshake errors", manifest.Name, entry.crashCount),
			}
		}
		entry.state = stateRegistered
		return &core.StepError{
			Code:    "plugin_handshake_error",
			Message: fmt.Sprintf("plugin %q handshake failed: %v", manifest.Name, err),
		}
	}

	entry.state = stateActive
	// Update DB status to active (TDS-05 §7: "DB status → active on first successful handshake").
	// Queued via setPluginStatusAsync (B-18): non-blocking, but ordered
	// relative to any other status write for this plugin.
	m.setPluginStatusAsync(manifest.Name, "active")

	// Restart idle timer (TDS-05 §7 IDLE: no execute for idle_timeout_s → kill).
	m.restartIdleTimerLocked(entry)
	return nil
}

// monitorProcess watches for unexpected process exit (TDS-05 §7 Crash).
// Runs in its own goroutine; closes done when the process exits.
// When entry.intentional is set, the kill was controlled (timeout/idle/shutdown)
// and the crash counter is NOT incremented (TDS-05 §5, §8).
func (m *Manager) monitorProcess(entry *pluginEntry, handle *processHandle, done chan struct{}) {
	defer close(done)
	// Wait blocks until the process exits (any exit).
	_ = handle.cmd.Wait()
	// Process has exited.
	entry.mu.Lock()
	defer entry.mu.Unlock()

	// If this handle is no longer the current one, the manager already handled it.
	if entry.handle != handle {
		return
	}
	// If the kill was intentional (timeout/idle/shutdown), do not count as crash.
	if entry.intentional {
		entry.intentional = false
		return
	}
	if entry.state == stateActive || entry.state == stateHandshaking {
		// Unexpected exit: mark as crash.
		entry.handle = nil
		m.stopIdleTimerLocked(entry)
		entry.crashCount++
		if entry.crashCount > crashLimit {
			entry.state = stateFailed
			m.setPluginStatusAsync(entry.manifest.Name, "failed")
		} else {
			entry.state = stateRegistered
		}
	}
}

// handleCrashLocked increments the crash counter and transitions the state.
// Returns the appropriate StepError.
// Must be called with entry.mu held.
// This is called when the caller (not the monitor) detects a crash (e.g. read
// error, protocol error). The monitor may also fire — both paths are idempotent
// because we check entry.handle.
func (m *Manager) handleCrashLocked(entry *pluginEntry, pluginName string, reason string) *core.StepError {
	m.stopIdleTimerLocked(entry)
	entry.crashCount++
	entry.handle = nil
	if entry.crashCount > crashLimit {
		entry.state = stateFailed
		go func() {
			_ = m.store.SetPluginStatus(context.Background(), pluginName, "failed")
		}()
		return &core.StepError{
			Code:    "plugin_failed",
			Message: fmt.Sprintf("plugin %q permanently failed after %d consecutive crashes (last: %s)", pluginName, entry.crashCount, reason),
		}
	}
	entry.state = stateRegistered
	return &core.StepError{
		Code:    "plugin_crash",
		Message: fmt.Sprintf("plugin %q crashed: %s", pluginName, reason),
	}
}

// resolveHandler resolves handlerRef to (pluginName, capabilityID).
// TDS-05 §6: capability id direct match → plugin name match by input key set.
func (m *Manager) resolveHandler(ctx context.Context, handlerRef string, inputs map[string]any) (string, string, *core.StepError) {
	// Rule 1: exact capability id match via storage.
	pluginName, err := m.store.LookupCapability(ctx, handlerRef)
	if err == nil {
		return pluginName, handlerRef, nil
	}
	if !errors.Is(err, storage.ErrCapabilityNotFound) {
		return "", "", &core.StepError{Code: "plugin_error", Message: fmt.Sprintf("capability lookup: %v", err)}
	}

	// Rule 2: handler == plugin name → resolve by unique input-key-set match.
	m.mu.Lock()
	entry, ok := m.plugins[handlerRef]
	m.mu.Unlock()
	if !ok {
		return "", "", &core.StepError{Code: "plugin_not_found", Message: fmt.Sprintf("no plugin or capability named %q", handlerRef)}
	}

	// Build the input key set from the provided inputs.
	inputKeys := make(map[string]bool, len(inputs))
	for k := range inputs {
		inputKeys[k] = true
	}

	// Find the unique capability whose declared input keys match.
	entry.mu.Lock()
	manifest := entry.manifest
	entry.mu.Unlock()

	var matchedID string
	matchCount := 0
	for _, cap := range manifest.Capabilities {
		if inputKeySetMatches(cap.Inputs, inputKeys) {
			matchedID = cap.ID
			matchCount++
		}
	}

	if matchCount == 0 || matchCount > 1 {
		return "", "", &core.StepError{
			Code:    "plugin_capability_ambiguous",
			Message: fmt.Sprintf("plugin %q: %d capabilities match input key set; use explicit capability id as handler", handlerRef, matchCount),
		}
	}
	return handlerRef, matchedID, nil
}

// inputKeySetMatches returns true when the declared input keys of a capability
// equal the provided input key set (TDS-05 §6 capability resolution).
func inputKeySetMatches(declaredInputs map[string]any, inputKeys map[string]bool) bool {
	if len(declaredInputs) != len(inputKeys) {
		return false
	}
	for k := range declaredInputs {
		if !inputKeys[k] {
			return false
		}
	}
	return true
}

// computeEffectiveTimeout computes min(ctx deadline, step.Timeout, capability timeout_ms)
// per TDS-05 §5.
func (m *Manager) computeEffectiveTimeout(ctx context.Context, manifest *Manifest, capID string, stepTimeout core.Duration) time.Duration {
	var candidates []time.Duration

	// ctx deadline.
	if dl, ok := ctx.Deadline(); ok {
		remaining := time.Until(dl)
		if remaining > 0 {
			candidates = append(candidates, remaining)
		}
	}

	// step.Timeout.
	if st := parsePluginTimeout(stepTimeout); st > 0 {
		candidates = append(candidates, st)
	}

	// capability timeout_ms.
	for _, c := range manifest.Capabilities {
		if c.ID == capID && c.TimeoutMS > 0 {
			candidates = append(candidates, time.Duration(c.TimeoutMS)*time.Millisecond)
			break
		}
	}

	if len(candidates) == 0 {
		return 0
	}
	min := candidates[0]
	for _, d := range candidates[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

// parsePluginTimeout parses a core.Duration string into time.Duration.
// Returns 0 for empty or unparseable values.
func parsePluginTimeout(d core.Duration) time.Duration {
	s := string(d)
	if s == "" {
		return 0
	}
	parsed, err := time.ParseDuration(s)
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

// buildEnv constructs the plugin process environment: manifest env + PATH ONLY
// (NFR-S-02 minimal env).
func buildEnv(manifestEnv map[string]string, dir string) []string {
	env := make([]string, 0, len(manifestEnv)+1)
	for k, v := range manifestEnv {
		if dir != "" && pathValuedEnvKeys[strings.ToUpper(k)] {
			v = resolveEnvPaths(v, dir)
		}
		env = append(env, k+"="+v)
	}
	// Always include PATH (needed for command resolution).
	env = append(env, "PATH="+pathEnv())
	return env
}

// pathValuedEnvKeys are the manifest env keys whose values are filesystem
// path lists, and which are therefore resolved against the manifest's own
// directory when relative (B-21).
//
// The list is deliberately an explicit allowlist rather than a heuristic over
// value shapes: rewriting an env value the plugin author did not mean as a
// path would be a silent, hard-to-debug corruption of the plugin's
// environment. Only interpreter module-search paths appear here, because
// those are the ones a plugin must express relative to itself in order for
// the SHIPPED manifest to work from any working directory.
var pathValuedEnvKeys = map[string]bool{
	"PYTHONPATH":          true,
	"AWIS_PLUGIN_LIBPATH": true,
	"NODE_PATH":           true,
}

// resolveEnvPaths rewrites each relative entry of an OS-path-list-separated
// value to be absolute against dir, leaving absolute entries and empty
// segments untouched.
//
// This is what lets the shipped git-context-plugin manifest carry a relative
// AWIS_PLUGIN_LIBPATH and still work: before B-21, a relative value was
// interpreted against the awis process's working directory, so the plugin
// resolved its own package only when awis happened to be run from the plugin
// directory. The one e2e test that covered it passed solely because the test
// wrote its own manifest with an absolute path injected — it proved the wire
// protocol worked and proved nothing about the plugin that actually ships.
func resolveEnvPaths(value, dir string) string {
	sep := string(os.PathListSeparator)
	parts := strings.Split(value, sep)
	for i, p := range parts {
		if p == "" || filepath.IsAbs(p) {
			continue
		}
		parts[i] = filepath.Join(dir, p)
	}
	return strings.Join(parts, sep)
}

// pathEnv returns the current PATH value or a sensible default.
func pathEnv() string {
	// Use os.Getenv to get the parent PATH; this is the ONLY parent env var
	// inherited (NFR-S-02).
	path := os.Getenv("PATH")
	if path == "" {
		path = "/usr/local/bin:/usr/bin:/bin"
	}
	return path
}

// restartIdleTimerLocked resets (or starts) the idle watchdog timer for entry.
// Must be called with entry.mu held.
// The timer fires after idle_timeout_s seconds; when it fires it kills the
// process group and transitions to IDLE (DB stays active; TDS-05 §7 IDLE).
func (m *Manager) restartIdleTimerLocked(entry *pluginEntry) {
	m.stopIdleTimerLocked(entry)
	idleDur := m.idleDuration(entry)
	if idleDur <= 0 {
		return
	}
	entry.idleTimer = time.AfterFunc(idleDur, func() {
		m.onIdleTimeout(entry)
	})
}

// stopIdleTimerLocked stops and drains the idle timer. entry.mu must be held.
func (m *Manager) stopIdleTimerLocked(entry *pluginEntry) {
	if entry.idleTimer != nil {
		entry.idleTimer.Stop()
		entry.idleTimer = nil
	}
}

// idleDuration returns the idle timeout for entry, using IdleCheckInterval if
// set (fast-clock seam for tests), else the manifest idle_timeout_s.
func (m *Manager) idleDuration(entry *pluginEntry) time.Duration {
	if m.cfg.IdleCheckInterval > 0 {
		return m.cfg.IdleCheckInterval
	}
	s := entry.manifest.Runtime.IdleTimeoutS
	if s <= 0 {
		return 0
	}
	return time.Duration(s) * time.Second
}

// onIdleTimeout is called when the idle watchdog fires. It kills the process
// group (if still active) and transitions to IDLE (TDS-05 §7 IDLE).
// Invariant 3: the idle killer must never kill a process with a call in flight
// — by design the timer is stopped inside the lock before any call is dispatched.
func (m *Manager) onIdleTimeout(entry *pluginEntry) {
	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Only kill if still ACTIVE (a call might have arrived and reset the timer).
	if entry.state != stateActive {
		return
	}
	// Set intentional before killing so the monitor skips crash counting.
	entry.intentional = true
	handle := entry.handle
	if handle != nil {
		handle.killGroup()
	}
	// Wait for the monitor goroutine to finish.
	if entry.monitorDone != nil {
		done := entry.monitorDone
		entry.mu.Unlock()
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
		}
		entry.mu.Lock()
	}
	entry.intentional = false
	entry.handle = nil
	entry.state = stateIdle
	// DB status stays active (SPEC disposition: in-memory IDLE keeps DB active).
}

// killEntryLocked kills the process (if any) and resets the entry.
// Must be called with entry.mu held.
func (m *Manager) killEntryLocked(entry *pluginEntry) {
	m.stopIdleTimerLocked(entry)
	if entry.handle != nil {
		entry.intentional = true
		entry.handle.killGroup()
		entry.handle = nil
	}
	entry.state = stateTerminated
}

// PluginPID returns the OS PID of a named plugin's running process.
// Returns 0 if the plugin is not active. This is a test hook for the §20
// checkpoint test (the card names it "test hook").
func (m *Manager) PluginPID(name string) int {
	m.mu.Lock()
	entry, ok := m.plugins[name]
	m.mu.Unlock()
	if !ok {
		return 0
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.handle == nil {
		return 0
	}
	return entry.handle.pid()
}
