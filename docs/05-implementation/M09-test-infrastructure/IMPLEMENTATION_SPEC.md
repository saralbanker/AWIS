# M09 — Implementation Spec
**Canonical sources:** IMP §27.M9; IMP §19 (integration row + test-data policy); IMP §20 (M9
checkpoint); Blueprint §27 (WorkflowTestHarness + Deterministic Execution Mode, verbatim
reference); PRD FR-SDK-06/07/08; PRD QG-5.

## Scope
`sdk/testing`: WorkflowTestHarness, DeterministicMode, MockIntelligence. The harness drives the
**real** engine with deterministic sources — never a re-implementation (IMP §27.M9 risk row).
Prerequisite seams (injectable ID source; sdk clock/id passthrough; intelligence-runner wiring)
are M09 work because M08 exited without them (M08 HANDOFF "Seam status at M08 exit").

## 1. Determinism seams (M09-C1)

### 1a. Engine ID source (additive)
`internal/engine/engine.go` `Config` gains one field:
```go
// NewID is the injectable instance-id source (IMP §3 determinism rule);
// nil ⇒ crypto/rand UUIDv4 (newUUIDv4).
NewID func() string
```
`engine.New` wires `cfg.NewID` into the existing unexported `newID` field (currently hardcoded
to `newUUIDv4` at `engine.New`). nil ⇒ `newUUIDv4`. Zero behavior change when nil.

### 1b. sdk passthrough (additive; surface-freeze compliant)
`sdk/runtime.go` `Config` gains:
```go
// Clock is the injectable time source; nil ⇒ time.Now. Primarily for sdk/testing.
Clock func() time.Time
// NewID is the injectable instance-id source; nil ⇒ engine default (UUIDv4).
NewID func() string
```
`NewRuntime` passes both into `engine.Config`. Additive fields on an existing struct are
freeze-compliant (IMP §13 freeze forbids changing/removing existing identifiers; FR-SDK-07
mandates this capability).

### 1c. Intelligence wiring in NewRuntime
`NewRuntime` currently assembles only the NativeRunner (per M08 spec). M09-C1 completes it:
- `cfg.Intelligence` nil ⇒ NullAdapter (`internal/intelligence/adapters/null`).
- Build a `intel.Dispatcher` over the chosen port and register the
  `internal/runner/intelligence` Runner under `core.StepTypeIntelligence` in the runners map.
- Wiring pattern reference: `internal/engine/intelligence_test.go` (`newIntelEngine`,
  `nullDispatcher`).

### 1d. `sdk.DeterministicMode()` (FR-SDK-07)
```go
// DeterministicMode returns a Config preset for deterministic test execution:
// fixed clock, seeded sequential IDs, manual tick (no poll loop). Caller sets
// Namespace and Storage; Intelligence nil ⇒ NullAdapter (FR-SDK-07).
func DeterministicMode() Config
```
- Clock: fixed instant advancing a constant step per call (stable, monotonic).
- NewID: deterministic sequence (`"det-000001"`, …).
- TickInterval: set so the pull loop is never relied on — tests advance via `Runtime.Tick`.

## 2. WorkflowTestHarness (M09-C2) — `sdk/testing`, package `awistesting`
Directory `sdk/testing/` (IMP §5 L81: `harness.go`, `mock.go`), package name `awistesting`
(Blueprint §27 usage). Imports `sdk` + `internal/*` freely (inside the module); applications
import `github.com/awis/awis/sdk/testing`.

```go
func NewHarness(t *testing.T, opts ...Option) *Harness

func WithMockIntelligence(m core.IntelligencePort) Option   // route intelligence to a mock/fixture port
func WithStepHandler(h core.StepHandler) Option              // register a native handler
func WithClock(fn func() time.Time) Option                   // override the deterministic default
func WithIDSource(fn func() string) Option                   // override the deterministic default

type RunResult struct { InstanceID core.InstanceID }

func (h *Harness) Run(def *core.WorkflowDefinition, inputs map[string]any) (RunResult, error)
func (h *Harness) Signal(id core.InstanceID, name string, payload map[string]any)
func (h *Harness) Tick()
func (h *Harness) WaitForCompletion(id core.InstanceID, timeout time.Duration)
func (h *Harness) GetOutput(id core.InstanceID, key string) any
```
Semantics (Blueprint §27 verbatim behavior):
- `NewHarness` builds a Runtime from `sdk.DeterministicMode()` over an in-memory SQLite
  storage (`sdk.SQLiteStorage(":memory:")`); registers options' handlers; no external services
  (FR-SDK-06).
- `Run` registers the definition (semver from def), submits, then ticks synchronously until the
  instance is terminal OR waiting on a signal; returns the InstanceID. Never spawns goroutines
  that outlive the test; never sleeps on the poll interval.
- `Signal` delivers via `Runtime.Signal` then ticks until terminal-or-waiting again.
- `WaitForCompletion` ticks (bounded by timeout) until terminal; `t.Fatal` on timeout.
- `GetOutput` reads `Runtime.Status(...).Outputs[key]`.
- All failures report through `h.t` with the failing instance's status for debuggability.

## 3. MockIntelligence (M09-C2) — FR-SDK-08
```go
func NewMockIntelligence() *MockIntelligence   // implements core.IntelligencePort

func (m *MockIntelligence) OnDraft(resp core.DraftResponse)
func (m *MockIntelligence) OnEmbed(vec []float32)
func (m *MockIntelligence) OnClassify(category string, confidence float64)
```
- Fixture-response mock: each `On*` sets the canned response for that capability.
- `Synthesize` (port completeness; not named by FR-SDK-08): returns a zero-value
  fixture unless a test sets it via an unexported default — keep the exported surface exactly
  the FR-SDK-08 trio + constructor.
- Un-fixtured capability invoked → error mirroring NullAdapter degradation semantics.

## 4. Fixtures + §19 integration suite + QG-5 (M09-C3)
- `internal/fixtures`: shared deterministic workflow definitions (IMP §19 test-data policy —
  "one shared fixture package … so a semantic change breaks loudly everywhere at once").
  Definitions for: linear, fan-out+join, retry-exhaustion→fallback, compensation,
  cancellation (± compensate), WAIT/signal, timeout-action workflows.
- `sdk/testing/integration_test.go` (or split files): the IMP §19 integration row executed on
  the harness — one test per listed shape. Engine-level precursors from M06/M07 remain
  untouched; this suite is the §19 integration layer going forward (CI-blocking from M09).
- **QG-5 test:** complete workflow including WAIT/signal delivery in a single Go test,
  asserted `< 1s` wall clock, zero external dependencies (in-memory SQLite only).
- **IMP §20.M9 checkpoint test:** an OIP-shaped workflow (multi-step capture-like flow with the
  plugin step stubbed as a native handler) runs in a unit test `< 1s`.
- Determinism proof: run the same fixture workflow twice in one test; assert the two event
  streams are identical in type sequence and step ids (stable IDs/clock).

## Non-scope (do not implement in M09)
- YAML DSL (M10); subprocess/plugin runners (M11/M12); CLI (M14).
- Real intelligence adapters (M16) — mock + Null only.
- Benchmarks beyond the QG-5/checkpoint timing assertions (M18).
- No new StoragePort methods; no migrations; no changes to frozen shapes.
