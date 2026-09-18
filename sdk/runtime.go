// Package sdk is the public surface applications use to interact with AWIS.
// This file implements the Runtime constructor and lifecycle methods
// (Blueprint §12; IMP §17; IMP §27.M8).
package sdk

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/engine"
	intel "github.com/awis/awis/internal/intelligence"
	"github.com/awis/awis/internal/intelligence/adapters/null"
	"github.com/awis/awis/internal/plugin"
	runintel "github.com/awis/awis/internal/runner/intelligence"
	"github.com/awis/awis/internal/runner/native"
	"github.com/awis/awis/internal/runner/subprocess"
	"github.com/awis/awis/internal/storage"
)

// Config is the configuration for a Runtime. Namespace and Storage are
// required; all other fields receive sensible defaults when zero.
type Config struct {
	// Namespace is the owning namespace for workflows run by this runtime.
	// Required: must be non-empty.
	Namespace string
	// Storage is the persistence boundary. Required: must be non-nil.
	Storage core.StoragePort
	// Intelligence is the optional intelligence provider. nil = NullAdapter behaviour.
	Intelligence core.IntelligencePort
	// WorkerID identifies this worker in step claims and audit records.
	// Defaults to hostname when empty.
	WorkerID string
	// TickInterval is the pull-loop period. Defaults to engine default (100ms).
	TickInterval time.Duration
	// Clock is the injectable time source; nil ⇒ time.Now. Primarily for sdk/testing.
	Clock func() time.Time
	// NewID is the injectable instance-id source; nil ⇒ engine default (UUIDv4).
	NewID func() string
}

// Runtime wraps the internal engine and exposes the application-facing sdk
// surface (Blueprint §12). Construct via NewRuntime.
type Runtime struct {
	eng       *engine.Engine
	nr        *native.NativeRunner
	storage   core.StoragePort
	namespace string
	workerID  string
	pluginMgr *plugin.Manager // nil when storage lacks PluginStore

	mu       sync.Mutex
	defs     map[string]core.SemVer // id → latest registered version
	handlers map[string]bool        // handler id → registered
}

// NewRuntime validates cfg and constructs a Runtime backed by the internal
// engine. Returns an error if cfg.Namespace is empty or cfg.Storage is nil.
func NewRuntime(cfg Config) (*Runtime, error) {
	if cfg.Namespace == "" {
		return nil, fmt.Errorf("awis: NewRuntime: Config.Namespace must be non-empty")
	}
	if cfg.Storage == nil {
		return nil, fmt.Errorf("awis: NewRuntime: Config.Storage must be non-nil")
	}

	workerID := cfg.WorkerID
	if workerID == "" {
		host, err := os.Hostname()
		if err != nil {
			host = "unknown-host"
		}
		workerID = host
	}

	nr := native.New()

	// Wire intelligence: nil cfg.Intelligence ⇒ NullAdapter (spec §1c).
	port := cfg.Intelligence
	if port == nil {
		port = null.New()
	}
	router, err := intel.NewRouter(
		[]intel.Registration{{Adapter: port, Locality: intel.LocalityLocal}},
		[]string{port.ProviderName()},
	)
	if err != nil {
		return nil, fmt.Errorf("awis: NewRuntime: intelligence router: %w", err)
	}
	disp := intel.NewDispatcher(router)

	// Wire plugin manager: type-assert PluginStore from storage (additive
	// interface — SPEC §3; StoragePort 12-method set is NOT changed).
	// If storage does not implement PluginStore, the plugin runner always
	// returns plugin_error "storage does not support plugins" (SPEC §3).
	var pluginMgr *plugin.Manager
	var pluginRunner engine.Runner
	if ps, ok := cfg.Storage.(storage.PluginStore); ok {
		pluginMgr = plugin.NewManager(ps, plugin.ManagerConfig{
			Clock: cfg.Clock,
		})
		pluginRunner = plugin.NewPluginRunner(pluginMgr)
	} else {
		pluginRunner = &noPluginStoreRunner{}
	}

	runners := map[core.StepType]engine.Runner{
		core.StepTypeNative:       nr,
		core.StepTypeIntelligence: runintel.New(disp),
		core.StepTypeSubprocess:   subprocess.New(),
		core.StepTypePlugin:       pluginRunner,
	}

	eng := engine.New(cfg.Storage, runners, engine.Config{
		TickInterval: cfg.TickInterval,
		WorkerID:     workerID,
		Clock:        cfg.Clock,
		NewID:        cfg.NewID,
	}, nil)

	return &Runtime{
		eng:       eng,
		nr:        nr,
		storage:   cfg.Storage,
		namespace: cfg.Namespace,
		workerID:  workerID,
		pluginMgr: pluginMgr,
		defs:      make(map[string]core.SemVer),
		handlers:  make(map[string]bool),
	}, nil
}

// noPluginStoreRunner is the plugin runner used when the storage does not
// implement PluginStore (SPEC §3 nil-safe wiring).
type noPluginStoreRunner struct{}

func (r *noPluginStoreRunner) Run(_ context.Context, _ core.StepContext, _ core.Step) (core.StepResult, *core.StepError) {
	return core.StepResult{}, &core.StepError{
		Code:    "plugin_error",
		Message: "storage does not support plugins",
	}
}

// SQLiteStorage opens (or creates) a SQLite-backed StoragePort at path.
// Blueprint §12 L934 authorizes this as a public sdk helper so the example
// app and application code can open storage without importing internal/storage.
// The returned StoragePort satisfies all additive interfaces (cancellation,
// signals, triggers, audit) that the runtime expects via type assertion.
func SQLiteStorage(path string) (core.StoragePort, error) {
	db, err := storage.Open(path, time.Now)
	if err != nil {
		return nil, fmt.Errorf("sdk: SQLiteStorage: %w", err)
	}
	return storage.NewSQLiteStorage(db, time.Now), nil
}

// Start starts the engine pull loop. It blocks until ctx is cancelled.
// On return (Stop path) it shuts down the plugin manager if one was wired.
func (r *Runtime) Start(ctx context.Context) error {
	err := r.eng.Run(ctx)
	// Stop path: shut down plugin manager to reap all plugin goroutines
	// (TDS-05 §7 Shutdown; SPEC §3 wiring; invariant 5 no goroutine leaks).
	if r.pluginMgr != nil {
		// Use a background context: the original ctx is already cancelled.
		r.pluginMgr.Shutdown(context.Background())
	}
	return err
}

// Tick runs one engine tick synchronously. Useful for tests and the example
// app where a continuous Run loop is not desired.
func (r *Runtime) Tick(ctx context.Context) error {
	return r.eng.Tick(ctx)
}

// RegisterPlugin registers a plugin from its manifest file with the runtime's
// plugin manager (M15-P1 seam; IMP §17 plugin tier in SDK rollout). It parses
// the manifest at manifestPath, persists it to the PluginStore, and adds it to
// the in-memory manager so the engine can dispatch type=plugin steps.
//
// Returns ErrNoPluginManager when the storage backend does not implement
// PluginStore (i.e. the runtime was created without SQLiteStorage or equivalent).
func (r *Runtime) RegisterPlugin(ctx context.Context, manifestPath string) error {
	if r.pluginMgr == nil {
		return fmt.Errorf("sdk: RegisterPlugin: no plugin manager (storage does not implement PluginStore)")
	}
	if err := r.pluginMgr.Register(ctx, manifestPath); err != nil {
		return fmt.Errorf("sdk: RegisterPlugin %q: %w", manifestPath, err)
	}
	return nil
}

// Runner returns this Runtime as a core.WorkflowRunner. The Runtime
// self-implements the WorkflowRunner interface.
func (r *Runtime) Runner() core.WorkflowRunner {
	return r
}

// Recall returns this Runtime as a core.RecallAPI. The Runtime
// self-implements the RecallAPI interface.
func (r *Runtime) Recall() core.RecallAPI {
	return r
}
