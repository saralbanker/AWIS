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

	runners := map[core.StepType]engine.Runner{
		core.StepTypeNative:       nr,
		core.StepTypeIntelligence: runintel.New(disp),
		core.StepTypeSubprocess:   subprocess.New(),
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
		defs:      make(map[string]core.SemVer),
		handlers:  make(map[string]bool),
	}, nil
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
func (r *Runtime) Start(ctx context.Context) error {
	return r.eng.Run(ctx)
}

// Tick runs one engine tick synchronously. Useful for tests and the example
// app where a continuous Run loop is not desired.
func (r *Runtime) Tick(ctx context.Context) error {
	return r.eng.Tick(ctx)
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
