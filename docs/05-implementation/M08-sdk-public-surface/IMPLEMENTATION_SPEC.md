# M08 — Implementation Spec
**Canonical sources:** IMP §27.M8; IMP §17 (SDK Rollout); Blueprint §12; PRD FR-SDK-*; F-1; F-2; F-4.

## Scope
Implement the `sdk` package behavioral layer. The type aliases (F-1 surface) already exist from M01.
M08 adds the constructors, registration, builder, runtime methods, and the example app.

## 1. Runtime Constructor (`sdk/runtime.go`)

```go
type Config struct {
    Namespace    string              // required; non-empty
    Storage      core.StoragePort    // required; additive signal/audit methods accessed by type assertion
    Intelligence core.IntelligencePort // optional; nil = NullAdapter behavior
    WorkerID     string              // optional; defaults to hostname
    TickInterval time.Duration       // optional; defaults to engine.DefaultTickInterval
}

type Runtime struct { /* wraps *engine.Engine; unexported fields */ }

func NewRuntime(cfg Config) (*Runtime, error)
```

`NewRuntime` validates `cfg.Namespace` non-empty and `cfg.Storage` non-nil; assembles the
internal `engine.Engine` with `NativeRunner`; returns `(*Runtime, error)`.

## 2. Registration (`sdk/registration.go`)

```go
func (r *Runtime) RegisterHandler(h core.StepHandler) error
func (r *Runtime) RegisterWorkflow(def *core.WorkflowDefinition) error
```

**Error format (PRD §18):** registration failures return a `*RegistrationError` implementing
`error` with a human-readable `Message` field:
```
awis: registration failed for "<id>"
  <reason>
  Hint: <hint>
```
Error cases:
- `RegisterHandler`: nil handler → error; duplicate handler ID → error (include existing ID in msg)
- `RegisterWorkflow`: nil def → error; invalid semver (def.Version) → error; duplicate (id+version)
  → error

**WorkflowRegistered audit (F-4):** `RegisterWorkflow` on success calls `AppendAudit` (type-asserted
from `cfg.Storage`) with `EventType: "WorkflowRegistered"`, `Actor: cfg.WorkerID`,
`PayloadSummary: "<def.ID> v<def.Version>"`. A storage that does not implement `AppendAudit`
silently skips the audit write (graceful degradation — audit is not core to registration).

## 3. WorkflowBuilder (`sdk/builder.go`)

```go
type WorkflowBuilder struct { /* unexported */ }

func NewWorkflowBuilder(id, version string) *WorkflowBuilder
// builder methods: AddStep, AddTransition, SetDescription, etc. (as needed for example app)

func (b *WorkflowBuilder) Build() (*core.WorkflowDefinition, error)
```

`Build()` validates:
- `id` non-empty
- `version` is valid semver: use `regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)`)` —
  no heavy deps; pure regexp is sufficient for V1 (IMP §27.M8 AC: "semver check at Build()").

## 4. WorkflowRunner implementation (`sdk/runtime_runner.go`)

`*Runtime` implements `core.WorkflowRunner`. Each method is a thin wrapper:

| Method | Engine call |
|---|---|
| `Submit(ctx, definitionID, inputs)` | look up latest registered version → `engine.Submit(ctx, definitionID, version, inputs)` |
| `Signal(ctx, instanceID, name, payload)` | `engine.Signal(ctx, instanceID, name, payload)` |
| `Status(ctx, instanceID)` | `storage.GetInstance(ctx, instanceID)` → populate `WorkflowStatus` |
| `Cancel(ctx, instanceID, reason)` | `engine.Cancel(ctx, instanceID, reason, false)` |
| `List(ctx, filter)` | `storage.ListInstances(ctx, filter)` → map to `[]WorkflowStatus` |

**WorkflowStatus shape (freeze at M08):**
```go
type WorkflowStatus struct {
    InstanceID   core.InstanceID
    DefinitionID string
    Version      core.SemVer
    Status       core.InstanceStatus
    CurrentSteps []string
    Inputs       map[string]any
    Outputs      map[string]any
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

## 5. RecallAPI implementation (`sdk/runtime_recall.go`)

`*Runtime` implements `core.RecallAPI`:

- `QueryHistory(ctx, query)`: read `workflow_instances` (filtered by HistoryQuery fields) + join
  with `execution_events` for start/end times; return `[]ExecutionRecord`.
- `StepStats(ctx, definitionID, stepID)`: aggregate `StepCompleted` events for the step across
  all instances; return `StepStatistics`.
- `ReplayInstance`: stub returning `core.ReplayTrace{}` with a TODO comment ("M17").

**Shape freeze at M08:**
```go
type HistoryQuery struct {
    Namespace    string
    DefinitionID string           // optional; empty = all definitions
    Status       core.InstanceStatus // optional; zero = all statuses
}
type ExecutionRecord struct {
    InstanceID   core.InstanceID
    DefinitionID string
    Version      core.SemVer
    Status       core.InstanceStatus
    StartedAt    time.Time
    CompletedAt  *time.Time       // nil if still running
    Inputs       map[string]any
    Outputs      map[string]any
}
type StepStatistics struct {
    DefinitionID string
    StepID       string
    TotalRuns    int
    SuccessCount int
    FailureCount int
    AvgDurationMs float64
}
```

## 6. TriggerAPI — F-2 SDK intake (`sdk/trigger_api.go`)

F-2 mandates: M08 owns the TriggerAPI sdk surface. The engine's `Ingest(ctx, DomainEvent)` is
the underlying path.

```go
type TriggerAPI struct { engine *engine.Engine }

func (r *Runtime) Triggers() *TriggerAPI

func (t *TriggerAPI) SubmitEvent(ctx context.Context, ev core.DomainEvent) error
// delegates to t.engine.Ingest(ctx, ev)
```

## 7. Runtime lifecycle

```go
func (r *Runtime) Start(ctx context.Context) error  // starts the engine pull loop (engine.Run)
func (r *Runtime) Runner() core.WorkflowRunner       // returns r (self-implements WorkflowRunner)
func (r *Runtime) Recall() core.RecallAPI            // returns r (self-implements RecallAPI)
```

## 8. Example app (`examples/hello_workflow/main.go`)

A minimal runnable Go `main` package that:
1. Opens an in-memory SQLite storage (or temp file)
2. Creates a `Runtime` via `NewRuntime`
3. Registers a handler and a `WorkflowBuilder`-built workflow
4. Submits an instance
5. Runs one tick and prints the status
6. Imports ONLY `github.com/awis/awis/sdk` (never `internal/`)

The example must compile clean: `go build ./examples/...`.
The boundary check `grep -rn '"github.com/awis/awis/internal' examples/` must return empty.

## 9. Core type completions (M08 owns)

`core.WorkflowStatus`, `core.HistoryQuery`, `core.ExecutionRecord`, `core.StepStatistics` are
currently empty structs. M08 fills them per the shapes above. These ARE core types (canonical
coordinate: `internal/core/ports.go`) — update them in core; sdk/ aliases pick up the shapes
automatically (F-1).

## Non-scope (do not implement in M08)
- `sdk/testing/` (WorkflowTestHarness) → M09
- YAML DSL → M10
- `RecallAPI.ReplayInstance` (non-stub) → M17
- Plugin/subprocess runners → M11/M12
- CLI → M14
