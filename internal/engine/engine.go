package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/expr"
)

// idempotencyTTL is the cache TTL for step results (Blueprint §5 L2). It is an
// engine-internal constant at C1; C2 does not change it.
const idempotencyTTL = 24 * time.Hour

// defaultTickInterval, defaultMaxParallel are the Config zero-value defaults
// (Blueprint §8: 100ms tick, max_parallel 4).
const (
	defaultTickInterval = 100 * time.Millisecond
	defaultMaxParallel  = 4
)

// Config configures an Engine. Zero-value fields receive defaults in New.
type Config struct {
	// TickInterval is the pull-loop period; zero ⇒ 100ms (Blueprint §8).
	TickInterval time.Duration
	// MaxParallelSteps bounds concurrent step dispatch per tick; zero ⇒ 4 (§8).
	MaxParallelSteps int
	// WorkerID identifies this worker in step claims; empty ⇒ hostname-pid.
	WorkerID string
	// Clock is the injectable time source (IMP §3 determinism rule); nil ⇒ time.Now.
	Clock func() time.Time
	// NewID is the injectable instance-id source (IMP §3 determinism rule);
	// nil ⇒ crypto/rand UUIDv4 (newUUIDv4).
	NewID func() string
}

// Runner executes one step. All runner kinds (native, intelligence, …) produce
// either a StepResult or a *core.StepError (Blueprint §5 L2). The runner map on
// the Engine is the extension point: C2 registers the intelligence Runner, M07
// the signal runner, M11/M12 subprocess/plugin — all as additional map entries.
type Runner interface {
	Run(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.StepError)
}

// UsageRunner is the optional side-channel a Runner implements to report a
// per-dispatch core.Usage alongside its result (ADJ-8). core.StepResult stays
// frozen (no usage field), so the engine type-asserts UsageRunner before falling
// back to Runner.Run; the intelligence runner implements both. Usage is a live
// provider-call metric — it is NOT cached, so a cache-hit dispatch reports no
// usage (correct: no provider call happened) and its StepCompleted omits the
// {adapter, model, tokens_used} triple.
type UsageRunner interface {
	RunWithUsage(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.Usage, *core.StepError)
}

// Engine is the pull-based execution engine. Construct via New.
type Engine struct {
	storage core.StoragePort
	runners map[core.StepType]Runner
	cfg     Config
	logger  *slog.Logger
	now     func() time.Time
	newID   func() string

	mu   sync.Mutex
	seq  map[core.InstanceID]int // per-instance next-assigned sequence_num
	ver  map[core.InstanceID]int // per-instance optimistic-lock version (V1 single writer)
	defs map[defKey]*defView     // parsed definition views (conditions cached)

	// C2 (T4) in-memory bookkeeping — all V1 single-process, all reconstructible
	// from the EventLog on restart (EDR-011 §8 documents the derivations; a crash
	// loses these and the recovery derivation is not exercised at V1).
	retries map[retryKey]retrySched               // scheduled retries by (instance, step) (ADJ-6/EDR-011 §3)
	pending map[core.InstanceID]map[string]bool   // failure-driven direct activations (EDR-011 §8)
	cancels map[core.InstanceID]cancelIntent      // remembered Cancel reason+mode (Finalization B4)
	waits   map[core.InstanceID]map[string]string // live signal waits: instance → signalName → stepID (M07; EDR-011 §8)

	// engine-hardening Step 2a/2b: per-instance restart hydration and the B-4
	// terminally-failed-step guard.
	hydrated map[core.InstanceID]bool            // set once hydrate(iid) has run this process (hydrate.go)
	failed   map[core.InstanceID]map[string]bool // terminally-failed steps, never re-activated (B-4, failure.go)
}

// retryKey identifies a per-instance per-step retry schedule.
type retryKey struct {
	iid  core.InstanceID
	step string
}

// retrySched is one pending retry: the attempt ordinal to re-dispatch and the
// earliest tick time it may run (V1 tick-quantized, EDR-011 §3).
type retrySched struct {
	nextAttempt   int
	nextAttemptAt time.Time
}

// cancelIntent records a running-instance Cancel request's parameters until the
// tick drains in-flight work and finalizes cancellation (Finalization B4 step
// 4/5). V1 single-process; a crash loses it and rebuild resets
// cancellation_requested to 0 (EDR-011 §8) so the caller re-issues Cancel.
type cancelIntent struct {
	reason     string
	compensate bool
}

// New builds an Engine. runners is the (owned) dispatch map; a nil logger is
// replaced by a slog JSON handler on stderr.
func New(storage core.StoragePort, runners map[core.StepType]Runner, cfg Config, logger *slog.Logger) *Engine {
	if cfg.TickInterval == 0 {
		cfg.TickInterval = defaultTickInterval
	}
	if cfg.MaxParallelSteps == 0 {
		cfg.MaxParallelSteps = defaultMaxParallel
	}
	if cfg.WorkerID == "" {
		host, err := os.Hostname()
		if err != nil {
			host = "unknown-host"
		}
		cfg.WorkerID = fmt.Sprintf("%s-%d", host, os.Getpid())
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	if runners == nil {
		runners = map[core.StepType]Runner{}
	}
	newIDFn := cfg.NewID
	if newIDFn == nil {
		newIDFn = newUUIDv4
	}
	return &Engine{
		storage:  storage,
		runners:  runners,
		cfg:      cfg,
		logger:   logger,
		now:      cfg.Clock,
		newID:    newIDFn,
		seq:      make(map[core.InstanceID]int),
		ver:      make(map[core.InstanceID]int),
		defs:     make(map[defKey]*defView),
		retries:  make(map[retryKey]retrySched),
		pending:  make(map[core.InstanceID]map[string]bool),
		cancels:  make(map[core.InstanceID]cancelIntent),
		waits:    make(map[core.InstanceID]map[string]string),
		hydrated: make(map[core.InstanceID]bool),
		failed:   make(map[core.InstanceID]map[string]bool),
	}
}

// defKey identifies a cached definition view by (id, version).
type defKey struct {
	id      string
	version core.SemVer
}

// defView is a parsed, indexed WorkflowDefinition. Conditions are parsed once
// (at Submit or first scan) and cached here (T3: "parsed at submit, cached").
type defView struct {
	def       core.WorkflowDefinition
	inbound   map[string][]transEval // to-step id → inbound transitions (parsed)
	finalSet  map[string]bool
	stepTypes map[string]core.StepType
	steps     map[string]core.Step
}

// transEval is a transition with its condition pre-parsed (nil when absent).
type transEval struct {
	t    core.Transition
	cond *expr.ConditionExpr
}

// buildDefView parses and indexes def. A condition that fails to parse is an
// error (validate.Validate has already rejected such definitions at Submit; this
// guard keeps recovery paths typed).
func buildDefView(def core.WorkflowDefinition) (*defView, error) {
	dv := &defView{
		def:       def,
		inbound:   make(map[string][]transEval),
		finalSet:  make(map[string]bool, len(def.FinalSteps)),
		stepTypes: make(map[string]core.StepType, len(def.Steps)),
		steps:     make(map[string]core.Step, len(def.Steps)),
	}
	for _, s := range def.Steps {
		dv.stepTypes[s.ID] = s.Type
		dv.steps[s.ID] = s
	}
	for _, fs := range def.FinalSteps {
		dv.finalSet[fs] = true
	}
	for _, t := range def.Transitions {
		te := transEval{t: t}
		if cond := string(t.Condition); cond != "" {
			parsed, err := expr.ParseCondition(cond)
			if err != nil {
				return nil, fmt.Errorf("engine: parse condition %q (from=%s to=%s): %w", cond, t.From, t.To, err)
			}
			te.cond = parsed
		}
		dv.inbound[t.To] = append(dv.inbound[t.To], te)
	}
	return dv, nil
}

// defViewFor returns the cached def view for an instance, loading and caching it
// from the registry on a miss (recovery path: an instance submitted in a prior
// process run has no cached view).
func (e *Engine) defViewFor(ctx context.Context, inst core.WorkflowInstance) (*defView, error) {
	key := defKey{id: inst.DefinitionID, version: inst.DefinitionVersion}
	e.mu.Lock()
	dv, ok := e.defs[key]
	e.mu.Unlock()
	if ok {
		return dv, nil
	}
	def, err := e.storage.GetWorkflow(ctx, inst.DefinitionID, inst.DefinitionVersion)
	if err != nil {
		return nil, fmt.Errorf("engine: load definition %s@%s: %w", inst.DefinitionID, inst.DefinitionVersion, err)
	}
	dv, err = buildDefView(def)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	e.defs[key] = dv
	e.mu.Unlock()
	return dv, nil
}
