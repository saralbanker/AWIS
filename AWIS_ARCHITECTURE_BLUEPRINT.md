# AWIS ARCHITECTURE BLUEPRINT
## Workflow Intelligence Platform — Canonical Architecture

**Produced by:** AWIS Architecture Investigation Tribunal v2.0
**Date:** 2026-07-02
**Authority:** OIP_CONSTITUTION.md (Tier 0) + AWIS Investigation Directive v2.0
**Status:** Final canonical architecture — implementation may begin
**Governing constraint:** Solo-founder maintainable; supports a decade-long software portfolio

---

## TRIBUNAL PREAMBLE

Nine specialist roles were operated simultaneously: Distinguished Software Architect, Workflow Runtime Architect, Distributed Systems Engineer, Platform Engineer, SDK Architect, Plugin Framework Designer, Workflow Engine Architect, AI Infrastructure Architect, Systems Researcher. Five subagent investigations were run in parallel: Workflow Pattern Research (Temporal/Camunda/Prefect/Dagster/Airflow/Kestra/Windmill/LangGraph/n8n), Platform Intelligence (competitive execution model analysis), AI Architecture (provider abstraction patterns), Dependency Analysis (build vs. integrate for each subsystem), and Solo-Founder Sustainability (long-term maintainability under single-engineer constraint).

Every abstraction below survived a deletion attempt. Every pattern required either two real use cases from the target application portfolio or strong precedent from systems with demonstrated longevity. Fashionable architecture was ejected on contact. The result is one architecture, not a menu.

---

## 1. EXECUTIVE SUMMARY

AWIS is a **locally-hosted, AI-optional workflow execution runtime** designed as the shared engineering foundation for a portfolio of independent software products. It provides three things and nothing more: a durable execution substrate (run steps, persist state, recover from failure), an intelligence abstraction layer (optional enhancement behind a provider seam), and an application SDK (the minimum surface area applications need to build on the platform without reimplementing infrastructure).

The platform's center of gravity is the **Step** — an atomic, idempotent unit of work with typed inputs, typed outputs, and a handler that can be native code, a script, a plugin, or an intelligence call. Workflows are directed graphs of Steps. State is an event-sourced append-only log. Storage is SQLite locally, Postgres-compatible in the cloud. All intelligence providers sit behind a single interface; swapping providers is a config key change; removing all providers leaves a deterministic, AI-free system that still executes fully.

AWIS exists because building workflow infrastructure from scratch in every product is the dominant hidden cost in a software portfolio. The platform absorbs that cost once. Applications own their business logic. The runtime owns execution.

**Implementation model for OIP:** OIP is implemented as an AWIS application. The `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` document describes the historical standalone OIP design produced before AWIS was specified. It is superseded by this blueprint. Implementation follows the AWIS-application model described in §10. The standalone document is retained as a historical reference only and must not be used as an implementation guide.

**Applications the architecture must support (V1–V3 horizon):**
- OIP (organizational decision memory) — the first application; validates the platform
- NeuroDashboard (health analytics and visualization)
- Shade Ledger (domain-specific financial ledger)
- Job Application Automation (multi-step human-in-the-loop workflows)
- Industrial SaaS products (complex process workflows, scheduling, monitoring)
- Internal AI systems (intelligence-enhanced operational workflows)

**The six-week verdict:** Build the Step executor and SQLite event log, build the Go SDK, build OIP on top of them. If OIP's capture-and-recall loop can be expressed cleanly as AWIS workflows with no platform surgery, the platform boundary is correct.

---

## 2. PLATFORM PHILOSOPHY

AWIS exists to answer one recurring engineering question in a software portfolio: *"Why are we rebuilding this again?"*

Every non-trivial software product needs: step execution with retry, durable state persistence, failure recovery, AI integration behind a seam, observability. Without a platform, each product builds a private version — slightly different, incompatible, all maintained in parallel. The cost accumulates invisibly because each instance seems small, but the aggregate is a tax on everything.

**What AWIS is:**
- A runtime that executes workflow definitions expressed as Step graphs
- An event log that records every state change (execution history = organizational operational memory)
- An intelligence layer that makes AI a declarable capability in any step, not a special path
- An SDK that gives applications a clean surface to define workflows, register step handlers, and query execution history

**What AWIS is not:**
- A business-logic owner — applications own entities, rules, and domain concepts
- An application framework — AWIS has no opinion on application structure beyond the SDK interface
- A cloud-first product — it runs in a single process on a laptop; cloud is an upgrade path
- An AI-first product — intelligence is optional infrastructure; every workflow runs without it
- A visual programming platform — that is a UI layer built on top, not the platform itself

**The constitutional inheritance:**
AWIS inherits six principles directly from the OIP Constitution because they are not OIP-specific — they are correct for any long-lived platform:
- FP-7 (data outlives code): the event log is designed for decades; the runtime is designed for replacement
- FP-8 (minimal core, open edge): AWIS's core is the Step executor and event log; everything else is at the edge
- FP-9 (Gall's law): start with a simple Step executor that works, grow from there
- FP-11 (reasoning commoditizes; context does not): AWIS holds execution context; intelligence providers are interchangeable
- Article 32 (zero-AI viability): every workflow must execute with NullAdapter; AI is enhancement, not requirement
- Article 41–46 (minimal core): capability at the edge by default; platform core grows only under evidence

---

## 3. ARCHITECTURAL PRINCIPLES

These ten principles are non-negotiable. No recommendation in this document may violate them.

**P1 — Applications own business logic; the runtime owns execution.**
Applications define step handlers, workflow graphs, and trigger conditions. The runtime schedules, executes, persists state, and recovers from failure. Neither may cross this boundary.

**P2 — Intelligence is a declared capability, not a dependency.**
Any step may declare an intelligence capability (draft, embed, synthesize, classify). The runtime satisfies it from the configured provider. If no provider is configured or available, the step routes to its declared fallback. Zero intelligence = degraded but standing system.

**P3 — State is event-sourced; history is never deleted.**
Every state change emits an event to an append-only log. Current state is a projection. Corrections append new events; they never modify prior ones. Erasure is a governed, recorded act — never a side effect.

**P4 — Local-first; cloud is a storage and concurrency upgrade, not a prerequisite.**
The same binary runs locally with SQLite and in the cloud with Postgres. Mode is a deployment decision, not an architectural one. No workflow definition assumes network connectivity.

**P5 — New applications require zero platform modification.**
The SDK exposes a stable surface. Applications register step handlers and workflow definitions against that surface. A new application is new user code, not a platform change.

**P6 — All providers are swappable at configuration time.**
Intelligence providers, storage backends, and trigger sources are adapters behind port interfaces. Swapping any one of them changes one config key.

**P7 — Workflows are deterministically testable without external services.**
Every adapter has a test double. A workflow run against mock adapters produces the same execution graph as a production run. Tests do not require network, AI API keys, or a running database.

**P8 — The event log is the platform's irreversible artifact.**
Format changes to the event log require versioned migration paths and backward-compatible readers. No other component has this constraint.

**P9 — No abstraction without two real use cases.**
Every interface, port, and adapter pattern in this blueprint is justified by at least two concrete uses from the application portfolio. Speculative generalization is explicitly rejected.

**P10 — Human authority is never eroded by default.**
Machine actors execute steps, but humans configure, authorize, and review. WAIT steps for human confirmation are a first-class primitive. No workflow escalates machine authority without explicit, bounded human delegation.

---

## 4. SYSTEM CONTEXT

```
┌────────────────────────────────────────────────────────────────┐
│                    APPLICATION LAYER                           │
│                                                                │
│  ┌─────────┐  ┌──────────────┐  ┌─────────────┐  ┌────────┐  │
│  │   OIP   │  │NeuroDashboard│  │ Shade Ledger │  │  Job   │  │
│  │decision │  │  analytics   │  │  ledger      │  │ Auto   │  │
│  │ records │  │  workflows   │  │  workflows   │  │       │  │
│  └────┬────┘  └──────┬───────┘  └──────┬───────┘  └───┬────┘  │
│       │              │                  │              │       │
│  ─────┴──────────────┴──────────────────┴──────────────┴────   │
│                    APPLICATION SDK                             │
│        WorkflowBuilder │ StepRegistry │ TriggerAPI │ RecallAPI │
└───────────────────────────────┬────────────────────────────────┘
                                │
┌───────────────────────────────▼────────────────────────────────┐
│                      AWIS RUNTIME                              │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                  WORKFLOW ENGINE                         │  │
│  │  Scheduler │ Executor │ State Machine │ Signal Handler   │  │
│  └───────────────────────┬──────────────────────────────────┘  │
│                          │                                     │
│  ┌───────────────────────▼──────────────────────────────────┐  │
│  │                   STEP RUNTIME                           │  │
│  │  NativeRunner (Go) │ SubprocessRunner │ PluginRunner     │  │
│  └───────────────────────┬──────────────────────────────────┘  │
│                          │                                     │
│  ┌───────────────────────▼──────────────────────────────────┐  │
│  │                INTELLIGENCE LAYER                        │  │
│  │  IntelligencePort │ CapabilityRouter │ FallbackChain    │  │
│  └───────────────────────┬──────────────────────────────────┘  │
│                          │                                     │
│  ┌───────────────────────▼──────────────────────────────────┐  │
│  │               PERSISTENCE LAYER                          │  │
│  │  StoragePort │ EventLog │ StateStore │ WorkflowRegistry  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │               OBSERVABILITY ENGINE                       │  │
│  │       ExecutionTracer │ MetricsCollector │ AuditLog      │  │
│  └──────────────────────────────────────────────────────────┘  │
└───────────────────────────────┬────────────────────────────────┘
                                │
          ┌─────────────────────┼──────────────────────┐
          │                     │                      │
   ┌──────▼──────┐   ┌──────────▼─────────┐   ┌───────▼──────┐
   │   STORAGE   │   │  INTELLIGENCE      │   │   PLUGINS    │
   │             │   │  PROVIDERS         │   │              │
   │  SQLite     │   │  Anthropic         │   │  git-context │
   │  (local)    │   │  OpenAI            │   │  custom-step │
   │  Postgres   │   │  Ollama (local)    │   │  shell-hook  │
   │  (cloud)    │   │  Null (test)       │   │  ...         │
   └─────────────┘   └────────────────────┘   └──────────────┘
```

**Bounded contexts:**
- AWIS owns: step execution, event log, workflow state, intelligence routing, plugin lifecycle
- Applications own: business entities, domain logic, step handler implementations, workflow definitions
- Plugins own: external capability integrations (git, APIs, hardware)
- Intelligence providers own: model selection, API transport, token management

**What crosses the boundary (explicit contracts):**
- Application → AWIS: WorkflowDefinition, StepHandler registrations, TriggerConditions
- AWIS → Application: StepContext (inputs), ExecutionEvent notifications, RecallQuery results
- AWIS → Plugin: JSON-RPC capability requests
- AWIS → Intelligence: IntelligencePort method calls with context budgets
- AWIS → Storage: StoragePort operations (event append, state upsert, definition fetch)

---

## 5. RUNTIME ARCHITECTURE

The runtime has five layers. Each layer has one responsibility and communicates with adjacent layers through defined interfaces.

### Layer 1: Workflow Engine

**Responsibility:** Determine what runs next and when.

The engine runs a pull-based execution loop. Default cadence: 100ms locally, configurable. Each tick:

```
1. Load runnable steps
   → instances with status=running where next step's upstream deps are all completed
   → instances with status=pending where trigger condition is met

2. For each runnable item:
   a. Claim it (optimistic lock prevents double-execution)
   b. Emit StepStarted event to EventLog
   c. Dispatch to Step Runtime
   d. On result: emit StepCompleted or StepFailed event
   e. Evaluate transitions: compute next step(s)
   f. If final step: emit WorkflowCompleted event

3. Load signal-triggered unblocks
   → instances with status=waiting where a signal has arrived
   → Unblock them into running state
```

This is a pull model, not push. Pull is simpler to reason about, debuggable (you can inspect the queue), and works identically in single-process local mode and in multi-worker cloud mode (workers compete via optimistic locking on the state store).

### Layer 2: Step Runtime

**Responsibility:** Execute one step in the right handler environment.

Four runner types:

| Runner | Handler type | Isolation | When used |
|---|---|---|---|
| NativeRunner | Go StepHandler interface | In-process | Go step handlers (fastest path) |
| SubprocessRunner | Any executable | Subprocess (stdout/stdin JSON) | Python, shell, Node scripts |
| PluginRunner | Registered AWIS plugin | Subprocess (JSON-RPC) | External capability plugins |
| IntelligenceRunner | IntelligencePort call | In-process + external API | Steps typed as `intelligence` |

All runners produce the same output: `StepResult{outputs: map, error?: StepError}`.

Idempotency: every step execution gets a deterministic key (`instance_id + step_id + attempt_number`). Before executing, the runtime checks the StepResultCache. If a result already exists for this key, it returns it without re-executing. This makes retries safe.

### Layer 3: Intelligence Layer

**Responsibility:** Route intelligence requests to the appropriate provider.

Described fully in §13–17. The layer sits between the Step Runtime (which calls it) and the intelligence providers (which satisfy requests). It never appears in the critical path of non-intelligence steps.

### Layer 4: Persistence Layer

**Responsibility:** Durably store all platform state.

Four stores, all behind StoragePort:

| Store | Contents | Access pattern |
|---|---|---|
| EventLog | All execution events (append-only) | Append-heavy, sequential read for replay |
| StateStore | Current workflow instance states | Read-heavy, point-in-time upsert on transitions |
| WorkflowRegistry | Workflow definitions by ID + version | Read-heavy, rare write |
| StepResultCache | Idempotency keys + results | Point lookup, TTL-based expiry |

### Layer 5: Observability Engine

**Responsibility:** Record what happened, why, and how long it took.

Three components: ExecutionTracer (structured spans per step/workflow), MetricsCollector (counters and histograms), AuditLog (governance-level record of configuration changes, signal deliveries, charter activations). Details in §26.

---

## 6. WORKFLOW REPRESENTATION

A WorkflowDefinition is the platform's stable data structure. Both the YAML DSL and the Go SDK produce instances of this struct. The runtime operates exclusively on WorkflowDefinition; it has no knowledge of how the definition was produced.

```
WorkflowDefinition {
  id:           string          // namespaced: "oip.capture-decision"
  version:      SemVer          // "1.0.0"; immutable once registered
  namespace:    string          // "oip" | "neurodashboard" | ...
  name:         string          // human-readable
  description:  string?

  triggers:     Trigger[]       // what starts this workflow
  steps:        Step[]          // all steps (nodes)
  transitions:  Transition[]    // conditional edges between steps
  initial_step: string          // which step runs first
  final_steps:  string[]        // which steps end the workflow

  compensation: CompensationPlan?  // ordered rollback steps on failure
  timeout:      Duration?          // max total workflow duration
  metadata:     map<string, any>   // application-defined
}
```

```
Step {
  id:           string          // unique within workflow
  name:         string
  type:         StepType        // native | subprocess | plugin | intelligence | signal
  handler:      HandlerRef      // "handler-name" for native; "script.py" for subprocess; etc.
  inputs:       InputSchema     // JSON Schema for expected inputs
  outputs:      OutputSchema    // JSON Schema for produced outputs
  retry:        RetryPolicy?    // attempts, backoff, retryable_errors
  timeout:      Duration?       // per-step timeout
  fallback:     string?         // step id to run if this step fails/capability unavailable
  compensation: CompensationRef? // undo action if workflow fails after this step completes
  wait_signal:  WaitConfig?     // for type=signal: signal name + timeout + timeout_action
  intelligence: IntelReq?       // for type=intelligence: capability + model_hint + context_budget
}
```

```
Transition {
  from:         string          // step id
  to:           string          // step id
  condition:    Condition?      // expression over step outputs; absent = unconditional
  on_error:     bool?           // this transition fires on step failure (not success)
}
```

```
Trigger {
  type:         TriggerType     // manual | schedule | event | webhook
  config:       map<string, any> // type-specific configuration
}
```

**WorkflowInstance** — a running instance of a definition:

```
WorkflowInstance {
  instance_id:         UUID
  definition_id:       string
  definition_version:  SemVer
  namespace:           string
  status:              InstanceStatus
  current_steps:       string[]      // active (running/waiting) step ids
  variables:           map<string, any>  // accumulated step outputs + workflow inputs
  started_at:          Timestamp
  updated_at:          Timestamp
  completed_at:        Timestamp?
}

InstanceStatus = pending | running | waiting | completed | failed | cancelled | compensating | compensated
```

---

## 7. WORKFLOW DSL

The platform supports two tiers for workflow definition. Both produce identical WorkflowDefinition structs.

### Tier 1: YAML DSL

For simple, declarative workflows. Preferred when logic is linear or conditionally branching without dynamic step generation.

```yaml
# Example: OIP capture-decision workflow
id: capture-decision
version: 1.0.0
namespace: oip
name: Capture Decision
description: Assemble context, draft entry, confirm, append to record

triggers:
  - type: manual
  - type: event
    config:
      event: git.push.completed
      filter: "event.branch == 'main'"
      # event.* references DomainEvent.payload fields; valid only in trigger filters

steps:
  - id: assemble-context
    name: Assemble Git Context
    type: plugin
    handler: git-context-plugin
    inputs:
      repo_path: "{{workflow.inputs.repo_path}}"
      ref: "{{workflow.inputs.ref}}"
    outputs:
      context: {type: object}
    retry:
      attempts: 3
      backoff: exponential

  - id: draft-entry
    name: Draft Decision Entry
    type: intelligence
    intelligence:
      capability: draft
      context_budget: 3000
    inputs:
      context: "{{steps.assemble-context.outputs.context}}"
    outputs:
      draft: {type: string}
    fallback: manual-entry

  - id: manual-entry
    name: Manual Entry (AI unavailable)
    type: signal
    wait_signal:
      name: manual_draft_provided
      timeout: 24h
      timeout_action: fail

  - id: confirm-entry
    name: Human Confirmation
    type: signal
    wait_signal:
      name: entry_confirmed
      timeout: 72h
      timeout_action: cancel

  - id: append-to-record
    name: Append to Record
    type: native
    handler: oip.record.append
    inputs:
      entry: "{{steps.confirm-entry.outputs.confirmed_entry}}"
    outputs:
      entry_id: {type: string}

initial_step: assemble-context

transitions:
  - from: assemble-context
    to: draft-entry
  - from: draft-entry
    to: confirm-entry
  - from: draft-entry
    to: manual-entry
    condition: "steps.draft-entry.status == 'fallback'"
  - from: manual-entry
    to: confirm-entry
  - from: confirm-entry
    to: append-to-record

final_steps: [append-to-record]

# No compensation plan: append-to-record is the final step.
# Compensation handlers are specified only on steps with downstream successors
# that could fail — never on final steps, whose handlers would never be invoked.
```

### Tier 2: Go SDK

For workflows with dynamic step generation, recursion, complex conditions, or programmatic composition.

```go
// Example: NeuroDashboard analysis pipeline
func AnalysisPipeline(cfg AnalysisConfig) awis.WorkflowDefinition {
    b := awis.NewWorkflowBuilder("neurodashboard.analysis-pipeline", "1.0.0")

    b.AddStep(awis.Step{
        ID:      "ingest-data",
        Type:    awis.StepTypeNative,
        Handler: "neurodashboard.ingest",
        Inputs:  awis.InputSchema{"source": awis.StringSchema()},
        Outputs: awis.OutputSchema{"records": awis.ArraySchema()},
        Retry:   awis.ExponentialRetry(3),
    })

    // Dynamic parallel steps based on configuration
    for _, metric := range cfg.Metrics {
        m := metric // capture loop var
        b.AddStep(awis.Step{
            ID:      fmt.Sprintf("compute-%s", m.Name),
            Type:    awis.StepTypeNative,
            Handler: "neurodashboard.compute",
            Inputs:  awis.InputSchema{"records": awis.Ref("steps.ingest-data.outputs.records")},
            Outputs: awis.OutputSchema{"result": awis.ObjectSchema()},
        })
        b.AddTransition("ingest-data", fmt.Sprintf("compute-%s", m.Name))
    }

    b.SetInitialStep("ingest-data")
    b.SetFinalSteps(cfg.metricStepIDs())
    return b.Build()
}
```

### DSL Design Rules

- YAML resolves all expression templates (`{{...}}`) at execution time, not parse time
- Both tiers compile via the same WorkflowValidator before registration
- Validation checks: no orphaned steps, no cycles (cyclic workflows must use new instance submission from a step handler), all handler refs resolvable, all schema refs valid
- The YAML tier is a subset of what the SDK can express; they are not equivalent in power, only in runtime representation
- **Parallel execution** is achieved via fan-out transitions: multiple `Transition` entries with the same `from` step activate all target steps concurrently within the execution loop. There is no `parallel` StepType; this model is sufficient for all V1 use cases.
- **Expression language (formal specification):**
  - *Templates* (`{{path-ref}}` in string field values): `path-ref` is a dot-path resolving to `workflow.inputs.<key>`, `steps.<step-id>.outputs.<key>`, or `steps.<step-id>.status`. Identifiers may contain hyphens. No operators, no function calls, no arithmetic. Missing paths resolve to empty string with a logged warning. Bracket notation (`steps['id']`) is prohibited — use dot notation (`steps.step-id`).
  - *Conditions* (in `Transition.condition` and `Trigger.config.filter`): boolean expressions using `==`, `!=`, `>`, `<`, `>=`, `<=`, `&&`, `||`, `!`, and parentheses. Path references use the same dot-path notation as templates. String literals use single quotes. Numeric and boolean literals supported. `null` literal supported for null checks only. Arithmetic, function calls, and bracket notation are prohibited. Conditions are validated at workflow registration time; invalid syntax causes registration failure.

---

## 8. EXECUTION ENGINE

### The Execution Loop

```
TICK (every 100ms locally, configurable):

1. SCAN_RUNNABLE
   Query StateStore: instances WHERE status = 'running'
                     AND current_steps includes steps where all upstream deps completed

2. SCAN_TRIGGERABLE
   Query StateStore: trigger conditions against current system state

3. CLAIM (optimistic lock)
   For each candidate: attempt to set status = 'in_flight' with expected_version = N
   If version mismatch: another worker got it; skip (cloud multi-worker path)

4. DISPATCH
   For each claimed step: send to appropriate runner (NativeRunner etc.)
   Execution is concurrent up to configured parallelism limit

5. SETTLE
   On runner completion: append result event to EventLog
   Compute next steps via transition evaluation
   Upsert WorkflowInstance in StateStore
   Release claim

6. SIGNAL_SCAN
   Check signal inbox for pending signals; deliver to waiting instances
```

### Concurrency Model

Local mode: single goroutine dispatcher; all native step handlers run as goroutines within the same process. Subprocess and plugin runners spawn processes. Parallelism is bounded by `max_parallel_steps` config (default: 4 locally).

Cloud mode: multiple worker processes sharing a Postgres StateStore. Optimistic locking on instance version prevents double-execution. Parallelism scales with worker count.

### Retry Policy

```
RetryPolicy {
  attempts:         int          // max total attempts (including first)
  backoff:          BackoffType  // immediate | linear | exponential
  initial_delay:    Duration
  max_delay:        Duration
  retryable_errors: string[]?    // if null: retry all transient errors
}
```

After all attempts exhausted: step transitions to `failed` state. If the step has a `fallback`, the fallback step is activated. If no fallback: workflow transitions to `failed` and compensation begins.

### Compensation

When a workflow fails after one or more steps have completed, compensation runs in reverse order:

```
CompensationPlan {
  steps: [{step_id: string, undo_handler: HandlerRef}]
}
```

Compensation steps have their own retry policies. If compensation itself fails, the workflow enters `compensation_failed` state and alerts are emitted (application-defined alert handler). This is a rare but critical case; the AuditLog always records it.

**When to specify compensation:** Compensation handlers are specified only on steps that have downstream steps following them in the happy path. A final step's compensation handler is never invoked — the workflow cannot fail after the last step successfully completes. Specifying a compensation handler on a final step is harmless but unnecessary and should be avoided to prevent confusion.

**Compensation and append-only invariants:** Applications whose data model is append-only (such as OIP, governed by OIP Constitution Article 7) must implement compensation handlers as new appends, not as modifications to existing records. An undo handler named `oip.record.void-entry` that appends a voiding entry is constitutional; a handler that modifies the original entry is not. The platform does not enforce this invariant — it is the application's responsibility.

### Long-Running Workflows

WAIT steps park the workflow instance indefinitely:
- Instance status → `waiting`
- A `WaitRecord` is stored: `{instance_id, signal_name, created_at, timeout_at, timeout_action}`
- The signal scanner checks WaitRecords each tick
- Signal delivery: `awis signal <instance_id> <signal_name> [--payload=<json>]`
- On timeout: `timeout_action` determines outcome (fail | compensate | continue with default value)

Long-running workflows can span days or weeks. The StateStore must support this; SQLite handles it trivially.

### Signal Delivery Atomicity

Signal delivery is a single database transaction across `signal_inbox`, `execution_events`, and `workflow_instances`:

```sql
BEGIN TRANSACTION;
  -- 1. Mark signal delivered (idempotency guard prevents double-delivery)
  UPDATE signal_inbox SET delivered_at = <now>
    WHERE instance_id = <id> AND signal_name = <name> AND delivered_at IS NULL;

  -- 2. Append event to EventLog
  INSERT INTO execution_events (...) VALUES (..., 'SignalReceived', ...);

  -- 3. Transition instance from waiting → running (optimistic lock)
  UPDATE workflow_instances
    SET status = 'running', updated_at = <now>, version = version + 1
    WHERE instance_id = <id> AND status = 'waiting' AND version = <expected_version>;
COMMIT;
```

If the `delivered_at IS NULL` guard finds no row (already delivered): transaction is a no-op. Delivery is idempotent.

If the optimistic lock in step 3 fails (cloud multi-worker race): transaction rolls back; signal_inbox entry remains undelivered; next scan tick re-attempts.

**Invariant:** The EventLog is authoritative. If `SignalReceived` exists in the EventLog for instance X and signal Y, the signal was delivered. Any StateStore inconsistency is recoverable via `awis rebuild-state`.

### Cancellation Semantics

```go
// SDK
runner.Cancel(ctx, instanceID, "reason") error

// CLI
awis cancel <instance-id> [--reason="..."] [--compensate]
```

**Behavior on cancellation request:**

1. Runtime sets `workflow_instances.cancellation_requested = 1` for the instance.
2. In-flight steps (already dispatched) run to completion. Their `StepCompleted` or `StepFailed` events are appended normally. The runtime does not kill step goroutines or processes.
3. After each in-flight step settles, the transition evaluator checks `cancellation_requested`. If set, no next step is activated regardless of transition conditions.
4. Without `--compensate` (default): instance transitions to `cancelled` after in-flight steps settle.
5. With `--compensate`: instance transitions to `compensating`; compensation plan runs; final state is `compensated`.
6. All pending `wait_records` for the instance are deleted. A cancelled instance cannot receive signals.
7. `WorkflowCancelled {reason: string, cancelled_at: Timestamp}` is appended to the EventLog.

**Idempotency:** Calling Cancel on a terminal instance (`completed`, `failed`, `cancelled`, `compensated`, `compensation_failed`) returns no error and logs a warning. Calling Cancel on a `pending` instance transitions it to `cancelled` immediately with no steps executed.

**StateStore schema addition:**
```sql
ALTER TABLE workflow_instances ADD COLUMN cancellation_requested INTEGER NOT NULL DEFAULT 0;
```

**When to specify compensation on cancel:** Only when the application needs to undo completed steps. Default is "stop without undoing" — consistent with industry-standard workflow engine behavior (Temporal, Conductor).

---

## 9. STATE MANAGEMENT

### Two-Layer State Architecture

**Layer 1 — EventLog (source of truth):**

Append-only. Every state change is an event. Never modified. Persists forever unless explicitly pruned by governed action.

```sql
CREATE TABLE execution_events (
  event_id      TEXT PRIMARY KEY,      -- UUID
  instance_id   TEXT NOT NULL,
  namespace     TEXT NOT NULL,
  event_type    TEXT NOT NULL,
  step_id       TEXT,
  payload       TEXT NOT NULL,         -- JSON
  emitted_at    TEXT NOT NULL,         -- ISO8601
  sequence_num  INTEGER NOT NULL       -- monotonically increasing per instance
);
CREATE INDEX idx_events_instance ON execution_events(instance_id, sequence_num);
CREATE INDEX idx_events_namespace ON execution_events(namespace, emitted_at);
```

**Layer 2 — StateStore (materialized projection):**

Read-optimized. Rebuilt from EventLog if corrupted. The runtime uses this for the execution loop; the EventLog is only read during replay and audit.

```sql
CREATE TABLE workflow_instances (
  instance_id          TEXT PRIMARY KEY,
  definition_id        TEXT NOT NULL,
  definition_version   TEXT NOT NULL,
  namespace            TEXT NOT NULL,
  status               TEXT NOT NULL,
  current_steps        TEXT NOT NULL,   -- JSON array of step IDs
  variables            TEXT NOT NULL,   -- JSON map
  started_at           TEXT NOT NULL,
  updated_at           TEXT NOT NULL,
  completed_at         TEXT,
  version              INTEGER NOT NULL  -- optimistic lock counter
);
CREATE INDEX idx_instances_namespace_status ON workflow_instances(namespace, status);
```

### Event Types

```
WorkflowStarted      {inputs: map}
StepStarted          {step_id, attempt: int, inputs: map}
StepCompleted        {step_id, attempt: int, outputs: map, duration_ms: int}
StepFailed           {step_id, attempt: int, error: StepError, retrying: bool}
StepFallbackActivated {step_id, fallback_step_id, reason: string}
SignalReceived       {signal_name, payload: map}
WorkflowCompleted    {outputs: map, duration_ms: int}
WorkflowFailed       {step_id, error: StepError}
WorkflowCancelled    {reason: string}
WorkflowCompensating {from_step: string}
WorkflowCompensated  {}
WorkflowCompensationFailed {step_id, error: StepError}
```

### State Rebuild

`awis rebuild-state [--namespace=<ns>]` replays the EventLog and rebuilds the StateStore. This is the recovery path after storage corruption, migration, or upgrade. Because the EventLog is the source of truth and the StateStore is a projection, this operation is safe to run at any time.

### Consistency Guarantees

Local mode (SQLite): serializable transactions; no concurrent writers. Signal delivery is a single transaction across all three affected tables (see §8 Signal Delivery Atomicity).
Cloud mode (Postgres): read-committed + optimistic locking; at-most-once step execution via claim/version mechanism. Signal delivery uses the same single-transaction model natively.

AWIS does NOT provide distributed transactions between the EventLog and external systems. Step handlers that write to external systems must be idempotent (or use a two-phase pattern via compensation).

### Application Database Boundary

The StoragePort interface and `runtime.db` cover AWIS execution state only: EventLog, StateStore, WorkflowRegistry, StepResultCache, signal_inbox, wait_records, plugin registry.

Applications that require application-specific databases (FTS indexes, domain entity storage, vector stores) manage their own SQLite files. AWIS does not provide a generic document store or raw SQL passthrough. This separation ensures StoragePort remains a clean interface and application-specific schemas do not couple to the platform. See §10 for OIP's specific storage boundary.

---

## 10. EVENT SYSTEM

### Internal vs. Domain Events

**Internal events** (above in §9) are execution lifecycle records stored in the EventLog. They are produced by the runtime and consumed by the Observability Engine and State Machine. Applications do not produce internal events directly.

**Domain events** are application-defined events that trigger workflow instances. They are produced by applications (or external systems) and consumed by the Trigger subsystem.

```
DomainEvent {
  event_id:   UUID
  namespace:  string
  event_type: string          // "git.push.completed", "user.signup", etc.
  source:     string          // what produced it
  payload:    map<string, any>
  emitted_at: Timestamp
}
```

### Event Bus (Internal)

An in-process publish/subscribe bus. The runtime publishes internal events; the Observability Engine, Signal Handler, and application-registered event handlers subscribe. Not distributed in V1.

V2 introduces an optional persistent event queue (SQLite-backed locally, Postgres queue in cloud) for cross-process event delivery without network coordination.

### Event Retention

Internal events (EventLog): retained permanently by default; governed pruning via `awis prune-events --before=<date> --dry-run`.
Domain events: retained for trigger matching (configurable TTL, default 7 days).

### Event Correlation

Workflows can be correlated to domain events via the `correlation_id` field on WorkflowInstance. `awis recall --correlation-id=<id>` returns all instances associated with a business object.

---

## 11. PLUGIN ARCHITECTURE

### Design Rationale

Plugins extend the platform with external capabilities (git integration, API calls, hardware interfaces, custom step types) without being part of the platform binary. They are:
- **Process-isolated**: plugin crashes cannot crash the runtime
- **Language-agnostic**: written in any language
- **Capability-declared**: plugins announce what they can do at registration time
- **Lifecycle-managed**: the runtime starts, monitors, and restarts plugins

### Communication Protocol

Plugins communicate with the AWIS runtime via **JSON-RPC 2.0 over stdin/stdout**. This requires no networking, no service discovery, and works identically in every environment.

```
Runtime → Plugin (stdin):
{
  "jsonrpc": "2.0",
  "id": "req-123",
  "method": "execute",
  "params": {
    "step_id": "assemble-context",
    "inputs": {"repo_path": "/path/to/repo", "ref": "abc123"},
    "timeout_ms": 30000
  }
}

Plugin → Runtime (stdout):
{
  "jsonrpc": "2.0",
  "id": "req-123",
  "result": {
    "outputs": {"context": {...}},
    "duration_ms": 450
  }
}
```

### Plugin Lifecycle

```
REGISTER     Plugin declares capabilities in awis-plugin.yaml
SPAWN        Runtime spawns plugin process with environment
HANDSHAKE    Plugin sends Manifest; runtime validates capabilities
ACTIVE       Runtime routes steps to plugin via JSON-RPC
IDLE         Plugin receives no requests for idle_timeout; paused
TERMINATE    Runtime sends shutdown; plugin exits cleanly
RESTART      On crash: runtime detects exit code; auto-restarts (max 3x)
```

### Plugin Manifest

```yaml
# awis-plugin.yaml
name: git-context-plugin
version: 1.0.0
description: Assembles git context for capture workflows
author: awis

capabilities:
  - id: git.context.assemble
    inputs:
      repo_path: string
      ref: string
    outputs:
      context: object
    timeout_ms: 30000

  - id: git.diff.fetch
    inputs:
      repo_path: string
      from_ref: string
      to_ref: string
    outputs:
      diff: string
    timeout_ms: 10000

runtime:
  command: "python"
  args: ["-m", "git_context_plugin"]
  env:
    GIT_TERMINAL_PROMPT: "0"
  idle_timeout_s: 300
```

### Plugin Registry

```sql
CREATE TABLE plugins (
  plugin_id    TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  version      TEXT NOT NULL,
  manifest     TEXT NOT NULL,    -- JSON
  status       TEXT NOT NULL,    -- registered | active | suspended | failed
  registered_at TEXT NOT NULL
);
CREATE TABLE plugin_capabilities (
  plugin_id    TEXT NOT NULL REFERENCES plugins(plugin_id),
  capability_id TEXT NOT NULL,
  PRIMARY KEY (plugin_id, capability_id)
);
```

### What Plugins Cannot Do

- Access the EventLog or StateStore directly (they only receive step inputs and return step outputs)
- Register new workflow definitions (only applications can do this via the SDK)
- Hold state between calls (plugin state must be external to the plugin process)
- Access another namespace's data

---

## 12. SDK ARCHITECTURE

### Purpose

The Application SDK is the surface applications use to interact with AWIS. It hides the runtime internals. Applications never import internal runtime packages.

### SDK Layers

```
awis/sdk (public)
├── workflow.go          WorkflowBuilder, WorkflowDefinition
├── step.go              Step, StepHandler interface, StepContext, StepResult
├── trigger.go           TriggerCondition, TriggerBuilder
├── runner.go            WorkflowRunner (submit, cancel, signal, status)
├── recall.go            RecallAPI (query execution history)
└── testing/
    ├── mock_runner.go   Deterministic workflow runner for tests
    └── mock_intel.go    NullAdapter + fixture-based mock
```

### StepHandler Interface

```go
// Applications implement this interface for native steps
type StepHandler interface {
    // ID is the handler identifier used in WorkflowDefinition.step.handler
    ID() string

    // Execute runs the step; returns outputs or an error
    Execute(ctx StepContext) (StepResult, error)
}

type StepContext struct {
    InstanceID  string
    StepID      string
    Attempt     int
    Inputs      map[string]any
    Intelligence IntelligencePort  // nil if no provider configured
    Logger      Logger
    Deadline    time.Time
}

type StepResult struct {
    Outputs map[string]any
    // No error field; return Go error from Execute() for failures
}
```

### WorkflowRunner Interface

```go
type WorkflowRunner interface {
    // Submit starts a new workflow instance
    Submit(ctx context.Context, definitionID string, inputs map[string]any) (InstanceID, error)

    // Signal delivers a signal to a waiting workflow instance
    Signal(ctx context.Context, instanceID InstanceID, signalName string, payload map[string]any) error

    // Status returns the current state of a workflow instance
    Status(ctx context.Context, instanceID InstanceID) (WorkflowStatus, error)

    // Cancel requests cancellation of a running instance
    Cancel(ctx context.Context, instanceID InstanceID, reason string) error

    // List returns instances matching the filter
    List(ctx context.Context, filter InstanceFilter) ([]WorkflowStatus, error)
}
```

### RecallAPI

```go
type RecallAPI interface {
    // QueryHistory returns execution history matching the query
    QueryHistory(ctx context.Context, query HistoryQuery) ([]ExecutionRecord, error)

    // ReplayInstance replays a completed instance for debugging
    ReplayInstance(ctx context.Context, instanceID InstanceID) (ReplayTrace, error)

    // StepStats returns aggregate statistics for a step across all instances
    StepStats(ctx context.Context, definitionID, stepID string) (StepStatistics, error)
}
```

### Registration

Applications register their step handlers and workflow definitions at startup:

```go
func main() {
    runtime := awis.NewRuntime(awis.Config{
        Namespace: "oip",
        Storage:   awis.SQLiteStorage("./oip.db"),
        Intelligence: awis.AnthropicAdapter(os.Getenv("ANTHROPIC_API_KEY")),
    })

    // Register step handlers
    runtime.RegisterHandler(&oip.RecordAppendHandler{})
    runtime.RegisterHandler(&oip.IndexSearchHandler{})

    // Register workflow definitions
    runtime.RegisterWorkflow(oip.CaptureDecisionWorkflow())
    runtime.RegisterWorkflow(oip.RecallDecisionWorkflow())

    // Start the runtime
    runtime.Start(context.Background())
}
```

### SDK Boundaries (what the SDK hides)

Applications have no access to:
- The EventLog (they receive ExecutionRecord summaries, not raw events)
- Other namespaces
- Plugin internals
- Intelligence provider credentials
- The execution loop internals

---

## 13. INTELLIGENCE LAYER

### Design Philosophy

Intelligence is a **declared capability in a step**, not a platform dependency. The Intelligence Layer is a translation layer between step declarations and provider implementations. If the layer is removed entirely, the runtime routes intelligence steps to their fallbacks and continues executing.

### IntelligencePort Interface

```go
type IntelligencePort interface {
    // Core capabilities
    Draft(ctx context.Context, req DraftRequest) (DraftResponse, error)
    Embed(ctx context.Context, text string) ([]float32, error)
    Synthesize(ctx context.Context, req SynthesisRequest) (SynthesisResponse, error)
    Classify(ctx context.Context, text string, categories []string) (Classification, error)

    // Provider introspection
    IsAvailable() bool
    Capabilities() []Capability
    ProviderName() string
}

type DraftRequest struct {
    Context     string          // assembled context (bounded by ContextBudget)
    Schema      map[string]any  // expected output structure
    Persona     string?         // optional: role to assume
    Examples    []Example?      // optional: few-shot
}

type SynthesisRequest struct {
    Query   string
    Entries []any
    MaxLen  int
}

type Classification struct {
    Category   string
    Confidence float32
    Reasoning  string?
}
```

### Capability Declaration in Steps

```yaml
- id: draft-decision
  type: intelligence
  intelligence:
    capability: draft
    context_budget: 3000     # max tokens for assembled context
    model_hint: fast         # fast | quality | local — routing hint, not mandate
    required: false          # if true: step fails if no provider; if false: routes to fallback
  fallback: manual-draft
```

### Capability Router Logic

```
REQUEST for capability X arrives:

1. Find all adapters that declare capability X and IsAvailable() = true
2. Apply routing policy:
   a. If model_hint = "local": prefer Ollama; fall back to cloud
   b. If model_hint = "fast": prefer cheapest cloud; fall back to local
   c. If model_hint = "quality": prefer highest-quality cloud
   d. Default: first available adapter with capability
3. If no adapter available AND step.intelligence.required = false:
   → Activate step.fallback
4. If no adapter available AND step.intelligence.required = true:
   → Fail step with CapabilityUnavailableError
```

### Context Budget Enforcement

Intelligence steps receive a bounded context window. The Context Assembler (application-defined) must produce context within the declared budget. The Intelligence Layer enforces this before dispatching to the provider — a request that exceeds the budget is rejected, not silently truncated.

---

## 14. AI PROVIDER ABSTRACTION

### Four Adapters

All four implement `IntelligencePort` identically. The runtime uses the adapter declared in config without conditional logic.

```
AnthropicAdapter
  Draft:     Claude Haiku (fast) or Sonnet (quality), per model_hint
  Embed:     Not natively supported; delegates to OpenAI embed if configured
  Synthesize: Claude Sonnet
  Classify:  Claude Haiku

OpenAIAdapter
  Draft:     GPT-4o-mini (fast) or GPT-4o (quality)
  Embed:     text-embedding-3-small
  Synthesize: GPT-4o
  Classify:  GPT-4o-mini

OllamaAdapter (local)
  Draft:     Configurable model (e.g., llama3.2, mistral)
  Embed:     nomic-embed-text or mxbai-embed-large
  Synthesize: Configurable model
  Classify:  Configurable model

NullAdapter (test/degraded mode)
  Draft:     Returns a deterministic fixture or error per configuration
  Embed:     Returns a zero vector
  Synthesize: Returns "The record is silent."
  Classify:  Returns first category with confidence 0.0
  IsAvailable(): false always
```

### Configuration

```yaml
# config.yaml (application-level, git-ignored)
intelligence:
  primary:
    provider: anthropic          # anthropic | openai | ollama | null
    api_key: ~                   # from env: ANTHROPIC_API_KEY
    draft_model: claude-haiku-4-5-20251001
    quality_model: claude-sonnet-4-6
    base_url: ~                  # override for proxy

  embed:
    provider: openai             # separate provider for embeddings if primary doesn't support
    api_key: ~                   # from env: OPENAI_API_KEY
    model: text-embedding-3-small

  local:
    provider: ollama
    base_url: http://localhost:11434
    draft_model: llama3.2
    embed_model: nomic-embed-text

  routing:
    prefer_local: false          # true: route to Ollama first if capable
    cost_aware: false            # V2 feature
    fallback_chain: [primary, local, null]
```

### Adapter Registration

Adapters are registered at runtime initialization. Multiple adapters can be registered; the CapabilityRouter selects among them per the routing policy.

```go
runtime := awis.NewRuntime(awis.Config{
    Intelligence: awis.IntelligenceConfig{
        Adapters: []IntelligencePort{
            awis.NewAnthropicAdapter(cfg.Anthropic),
            awis.NewOllamaAdapter(cfg.Ollama),
            awis.NewNullAdapter(),  // always registered as final fallback
        },
        FallbackChain: []string{"anthropic", "ollama", "null"},
    },
})
```

---

## 15. LOCAL INTELLIGENCE STRATEGY

### When Local Intelligence Is Used

| Scenario | Recommendation |
|---|---|
| No internet access | Ollama only |
| Cost sensitivity | Ollama for draft; cloud for quality recall |
| Data privacy (PII/proprietary) | Ollama for all capabilities involving sensitive content |
| Development / testing | NullAdapter (fastest) or Ollama (realistic) |
| Low-latency requirement (<200ms) | Ollama on local GPU; or NullAdapter |

### Ollama Integration

The OllamaAdapter calls Ollama's local REST API (HTTP, not stdin/stdout). Ollama must be running as a separate process (`ollama serve`). The adapter checks availability via a health endpoint before each request; if Ollama is not running, it returns `IsAvailable() = false` and the CapabilityRouter moves to the next in the fallback chain.

### Model Recommendations for Local

| Capability | Recommended Model | Min VRAM |
|---|---|---|
| Draft (fast) | llama3.2:3b | 4 GB |
| Draft (quality) | llama3.1:8b | 8 GB |
| Embed | nomic-embed-text | 2 GB (CPU) |
| Classify | llama3.2:3b | 4 GB |
| Synthesize | llama3.1:8b | 8 GB |

For hardware below these thresholds: use NullAdapter locally and cloud for production. Do not route to an under-resourced local model — the latency and quality degradation will undermine user trust in AI capabilities.

### Local Intelligence Limitation

Local models as of 2026 produce lower-quality drafts than cloud frontier models. This is known and acceptable:
- For testing: NullAdapter is sufficient
- For development: Ollama provides realistic (if lower quality) behavior
- For production: cloud models produce better drafts (E1 metric is sensitive to draft quality)

The architecture does not hide this tradeoff. The quality difference is a routing input, not an implementation detail.

---

## 16. CLOUD INTELLIGENCE STRATEGY

### When Cloud Intelligence Is Used

Production deployments where draft quality affects the core value hypothesis (E1: ≥5 confirmed entries/week/team past week 4). Cloud models currently produce significantly better zero-shot drafts, which directly impacts the median confirm time (E1 secondary metric: ≤60s).

### Cost Management

Cloud API costs are real. Management strategies:

1. **Context budget enforcement** (§13): prevents runaway token consumption per step
2. **Model tiering per step**: fast/cheap models for classification and drafting; quality models only for recall synthesis
3. **Caching** (V2): embed-cache avoids re-embedding unchanged entries; draft-cache avoids re-drafting identical contexts
4. **Batch embedding**: embed multiple entries in one API call

V1 has no cost metering. V2 introduces per-namespace cost tracking and configurable limits.

### Retry and Rate Limiting

```go
type CloudRetryPolicy struct {
    MaxAttempts    int           // default: 3
    RetryOn:       []int         // HTTP status codes: [429, 500, 502, 503]
    InitialBackoff time.Duration // default: 1s
    MaxBackoff     time.Duration // default: 30s
    Timeout        time.Duration // per-request: default: 30s
}
```

Rate limit headers (429 responses) are respected; backoff is extended per the Retry-After header when present.

---

## 17. AI ROUTING

### Routing Decision Tree

```
STEP requests capability C with hint H:

Is a provider available with capability C?
  └── NO → Is step.intelligence.required?
              └── YES → StepFailed(CapabilityUnavailable)
              └── NO  → Activate step.fallback

  └── YES → Apply hint H:
              ├── H = "local"   → Route to OllamaAdapter if capable; else cloud
              ├── H = "fast"    → Route to cheapest capable cloud adapter
              ├── H = "quality" → Route to highest-quality capable cloud adapter
              └── H = nil       → Route per fallback_chain order

Provider call fails (network/API error):
  └── Try next in fallback_chain
  └── If all fail and step.intelligence.required = true → StepFailed
  └── If all fail and step.intelligence.required = false → Activate fallback step
```

### Routing Configuration Example

```yaml
intelligence:
  routing:
    fallback_chain: [anthropic, ollama, null]
    capability_overrides:
      embed: [openai, ollama]       # embeddings: OpenAI preferred, then Ollama
      draft: [anthropic, ollama]    # drafting: Anthropic preferred, then Ollama
      classify: [ollama, anthropic] # classify: local first (cheaper)
```

### V2 Cost-Aware Routing

V2 adds cost-per-token tracking per adapter. The router can prefer the cheapest adapter that meets a quality threshold. Quality thresholds are declared per workflow, not per step (workflow-level policy).

### Routing Transparency

The adapter selected for each step execution is recorded in the StepCompleted event payload: `{"adapter": "anthropic", "model": "claude-haiku-4-5-20251001", "tokens_used": 847}`. This allows cost attribution and routing audit.

---

## 18. ORGANIZATIONAL MEMORY

### The Memory Stack

AWIS creates two complementary memory layers:

**Layer 1 — Operational Memory (AWIS EventLog):**
What the platform actually did: every workflow started, every step completed, every signal delivered, every failure recovered. This is the raw operational record — complete, append-only, machine-produced.

**Layer 2 — Decision Memory (OIP on top of AWIS):**
Why decisions were made: human-confirmed, rationale-bearing, cited entries. OIP is an AWIS application that produces curated entries from the raw operational record. The Record (`.decisions/entries/*.md`) is OIP's output, not AWIS's.

**The relationship:**

```
Organizational Memory
├── Operational Layer (AWIS)
│   └── EventLog: "what happened mechanically, and when"
│       - WorkflowStarted, StepCompleted, SignalReceived, etc.
│       - Raw; always complete; no curation
│       - Read by: Observability, RecallAPI, ReplayEngine
│
└── Decision Layer (OIP, built on AWIS)
    └── The Record: "what was decided, why, by whom, with what evidence"
        - Curated; human-confirmed; YAML+Markdown files
        - Read by: OIP recall CLI, future projections
        - Written by: OIP capture-decision workflow (itself running on AWIS)
```

Neither layer substitutes for the other. Operational records without decision context tell you *that* something happened; decision records without operational context tell you *why* something was decided but not whether it was implemented.

### Memory Access Patterns

Applications access organizational memory through the RecallAPI (execution history) or their own domain-specific indexes (OIP's SQLite FTS5 index, NeuroDashboard's time-series index). Cross-application memory access is not a V1 feature; it is a governed capability in V3 (requires explicit cross-namespace grant).

### Memory Durability Guarantees

- EventLog: survives runtime restart (WAL mode in SQLite)
- StateStore: rebuildable from EventLog
- OIP Record: in git; survives any single storage failure
- Embeddings cache: rebuildable; not source of truth

---

## 19. WORKFLOW LEARNING MODEL

V1 collects; V2 analyzes; V3 acts on patterns.

### V1 — Collection Foundation

The EventLog accumulates execution data sufficient for future learning. Every step records:
- Duration (actual vs. expected)
- Attempt count (retry signal)
- Intelligence adapter used
- Tokens consumed
- Fallback activated (yes/no)

This data is available immediately for manual analysis; no automated learning in V1.

### V2 — Pattern Extraction

V2 adds a background analytics engine that reads the EventLog and computes:

| Pattern | Derived from | Use |
|---|---|---|
| Step failure rate | StepFailed events / StepStarted events | Alert when rate exceeds threshold |
| Median step duration | StepCompleted.duration_ms | SLA monitoring |
| Fallback activation rate | StepFallbackActivated / StepStarted | Intelligence provider reliability |
| Workflow completion rate | WorkflowCompleted / WorkflowStarted | End-to-end health |
| Most common failure step | StepFailed grouped by step_id | Targeted reliability investment |

### V3 — Adaptive Routing

V3 uses historical performance to improve routing:
- Route intelligence requests to the adapter with the best success rate for that capability
- Adjust retry policies based on observed failure patterns
- Predict step duration for workflow scheduling optimization

V3 is explicitly out of scope for the initial build. No V3 architectural decisions are made here that would constrain V1 or V2.

---

## 20. STORAGE ARCHITECTURE

### StoragePort Interface

```go
type StoragePort interface {
    // EventLog
    AppendEvent(ctx context.Context, event ExecutionEvent) error
    ReadEvents(ctx context.Context, instanceID InstanceID, fromSeq int) ([]ExecutionEvent, error)
    ReadEventRange(ctx context.Context, namespace string, from, to time.Time) ([]ExecutionEvent, error)

    // StateStore
    UpsertInstance(ctx context.Context, instance WorkflowInstance, expectedVersion int) error
    GetInstance(ctx context.Context, instanceID InstanceID) (WorkflowInstance, error)
    ListInstances(ctx context.Context, filter InstanceFilter) ([]WorkflowInstance, error)
    ClaimStep(ctx context.Context, instanceID InstanceID, stepID string, workerID string) (bool, error)

    // WorkflowRegistry
    RegisterWorkflow(ctx context.Context, def WorkflowDefinition) error
    GetWorkflow(ctx context.Context, id string, version SemVer) (WorkflowDefinition, error)
    ListWorkflows(ctx context.Context, namespace string) ([]WorkflowDefinition, error)

    // StepResultCache
    CacheResult(ctx context.Context, key IdempotencyKey, result StepResult, ttl time.Duration) error
    GetCachedResult(ctx context.Context, key IdempotencyKey) (StepResult, bool, error)
}
```

### SQLite Schema (Full)

```sql
-- Core tables
CREATE TABLE workflow_definitions (
  id                TEXT NOT NULL,
  version           TEXT NOT NULL,
  namespace         TEXT NOT NULL,
  definition        TEXT NOT NULL,  -- JSON serialized WorkflowDefinition
  registered_at     TEXT NOT NULL,
  PRIMARY KEY (id, version)
);

CREATE TABLE workflow_instances (
  instance_id       TEXT PRIMARY KEY,
  definition_id     TEXT NOT NULL,
  definition_version TEXT NOT NULL,
  namespace         TEXT NOT NULL,
  status            TEXT NOT NULL,
  current_steps     TEXT NOT NULL,  -- JSON
  variables         TEXT NOT NULL,  -- JSON
  started_at        TEXT NOT NULL,
  updated_at        TEXT NOT NULL,
  completed_at      TEXT,
  version           INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE execution_events (
  event_id          TEXT PRIMARY KEY,
  instance_id       TEXT NOT NULL,
  namespace         TEXT NOT NULL,
  event_type        TEXT NOT NULL,
  step_id           TEXT,
  payload           TEXT NOT NULL,  -- JSON
  emitted_at        TEXT NOT NULL,
  sequence_num      INTEGER NOT NULL
);

CREATE TABLE step_results_cache (
  idempotency_key   TEXT PRIMARY KEY,  -- hash(instance_id + step_id + attempt)
  result            TEXT NOT NULL,     -- JSON
  cached_at         TEXT NOT NULL,
  expires_at        TEXT NOT NULL
);

CREATE TABLE signal_inbox (
  signal_id         TEXT PRIMARY KEY,
  instance_id       TEXT NOT NULL,
  signal_name       TEXT NOT NULL,
  payload           TEXT NOT NULL,   -- JSON
  delivered_at      TEXT,
  received_at       TEXT NOT NULL
);

CREATE TABLE wait_records (
  instance_id       TEXT NOT NULL,
  step_id           TEXT NOT NULL,
  signal_name       TEXT NOT NULL,
  created_at        TEXT NOT NULL,
  timeout_at        TEXT,
  timeout_action    TEXT NOT NULL,  -- fail | compensate | continue
  PRIMARY KEY (instance_id, step_id)
);

-- Indexes
CREATE INDEX idx_instances_ns_status ON workflow_instances(namespace, status);
CREATE INDEX idx_events_instance_seq ON execution_events(instance_id, sequence_num);
CREATE INDEX idx_events_ns_time ON execution_events(namespace, emitted_at);
CREATE INDEX idx_wait_timeout ON wait_records(timeout_at) WHERE timeout_at IS NOT NULL;
CREATE INDEX idx_signals_instance ON signal_inbox(instance_id, signal_name);
```

### Storage Layout on Disk

```
~/.awis/                          # global config
  config.yaml
  plugins/
    git-context-plugin/
      awis-plugin.yaml
      ...plugin files...

.awis/                            # per-project (in project root)
  runtime.db                      # SQLite: all runtime state
  runtime.db-wal                  # SQLite WAL
  config.yaml                     # project-level config overrides (git-ignored)
  .gitignore                      # ignores runtime.db, *.db-wal, config.yaml
```

### Migration Strategy

Schema migrations use a sequential versioned migration approach (similar to Flyway/golang-migrate):

```
.awis/migrations/
  0001_initial_schema.sql
  0002_add_signal_inbox.sql
  0003_add_plugin_registry.sql
```

On startup, the runtime checks the current schema version against applied migrations and runs forward migrations automatically. No migration runs automatically in cloud multi-worker mode (operator must run `awis migrate` explicitly before deploying new workers).

### Postgres Upgrade Path

SQLiteStorageAdapter → PostgresStorageAdapter is the upgrade path to cloud/team mode. The migration is:

1. Export all data: `awis export --format=migration-bundle`
2. Point config at Postgres connection string
3. Import: `awis import --from=migration-bundle`
4. Verify: `awis verify-storage`

The two adapters implement the identical StoragePort interface; no application code changes.

---

## 21. SECURITY MODEL

### V1 Threat Model

V1 is local-first, single-user. The threat model is:
- Plugin code that crashes or hangs (handled by process isolation)
- Step handlers that read/write outside their declared scope (addressed by application discipline, not enforcement in V1)
- Secrets in workflow variables (addressed by config, not encryption in V1)

V1 does NOT implement: authentication, authorization, network security, encrypted storage, or access control. These are V2 requirements when AWIS supports multiple concurrent users.

### Plugin Sandboxing

Process isolation is the primary containment mechanism. Plugins:
- Run as a separate process (not as goroutines in the runtime)
- Receive only the step inputs declared in their manifest
- Have no access to the StoragePort, EventLog, or StateStore
- Communicate exclusively via stdin/stdout (no filesystem access by convention; WASM enforcement in V3)

### Secret Management

API keys and credentials are never stored in workflow definitions or the EventLog. They are:
- Read from environment variables at runtime startup
- Injected into adapter constructors at initialization
- Never passed to step handlers as inputs

Plugin processes inherit a minimal environment (no platform secrets); if a plugin needs credentials, they are declared in `plugin.env` and read from the environment at plugin spawn time.

### Audit Log

A separate, append-only audit log records governance-level events:
```
WorkflowRegistered    {namespace, definition_id, version, registered_by}
WorkflowDeregistered  {namespace, definition_id, version, reason}
PluginRegistered      {plugin_id, capabilities}
ConfigChanged         {key, old_value_hash, new_value_hash}
SignalDelivered       {instance_id, signal_name, delivered_by}
```

The AuditLog is append-only and is never pruned without explicit operator action (separate from the EventLog).

---

## 22. MULTI-APPLICATION ISOLATION

### Namespace Model

Every workflow definition, workflow instance, event, and plugin capability is tagged with a namespace. The namespace is a string identifier that an application declares at initialization (`awis.Config{Namespace: "oip"}`).

**Isolation guarantees in V1 (single-process):**
- StoragePort prepends namespace to all query predicates
- The SDK prevents applications from constructing queries outside their namespace
- No cross-namespace reads are possible via the SDK

**Isolation guarantees in V2 (multi-application on one runtime):**
- Namespace-level resource quotas (max concurrent instances, max event log size)
- Explicit cross-namespace grants for shared data access
- Operator-level namespace management (`awis namespace create/list/delete`)

### Data Separation in SQLite

All tables include a `namespace TEXT NOT NULL` column. The StoragePort implementation enforces namespace in every query as a non-optional predicate. SQL injection is prevented by parameterized queries throughout.

```sql
-- Example: namespace-scoped query (internal, not exposed to applications)
SELECT * FROM workflow_instances
WHERE namespace = ? AND status = ?
ORDER BY started_at DESC;
-- The namespace parameter is always provided by the StoragePort, never by user input
```

### Application Registration

Applications identify themselves at initialization. The namespace must be a valid identifier (`[a-z][a-z0-9-]*`, max 63 chars). Namespace collision on the same runtime is an initialization error in V1.

---

## 23. MULTI-TENANT STRATEGY

### V1: Single Tenant, Local Deployment

One AWIS runtime per user/workstation. No authentication. No tenant isolation. All applications on the runtime trust each other (same operator).

### V2: Workspace Isolation

Multiple teams, one server. Workspaces map to namespaces. Authentication via API tokens (issued per workspace). No cross-workspace data access.

```
Workspace A: oip (team 1 decision records)
Workspace B: neurodashboard (team 2 analytics)
→ Isolated storage, isolated EventLog, isolated intelligence quotas
→ Shared AWIS runtime binary and plugin registry
```

### V3: Full Multi-Tenant SaaS

Full tenant isolation with:
- Per-tenant databases or schema isolation
- Per-tenant AI provider configuration (BYOK: bring your own API key)
- Per-tenant plugin allowlists
- Cross-tenant data sharing only via governed export/import

The multi-tenant architecture is NOT designed in V1. V3 requires a dedicated architectural phase. The V1/V2 architecture leaves the door open by keeping namespace isolation clean, but does not make V3-enabling decisions prematurely.

---

## 24. VISUAL DESIGNER ARCHITECTURE

### V1: No Visual Designer

V1 has no visual workflow designer. This is a deliberate absence: visual tooling built before the underlying model is stable locks in premature assumptions.

### What Must Be Stable Before a Visual Layer Can Be Built

The visual designer is a projection of the WorkflowDefinition struct. Before a visual layer makes sense, these must be stable:
1. WorkflowDefinition schema (stabilized in V1)
2. Step type set (extensible, not finalized — visual layer must handle unknown types)
3. Transition condition syntax (stabilized in V1 with a simple expression language)
4. State machine semantics (pending/running/waiting/completed/failed — stable in V1)

### V3 Visual Designer Contract

The visual designer will:
- Read WorkflowDefinition from the WorkflowRegistry (read-only access)
- Write WorkflowDefinition back to the registry (via the same registration API)
- Represent steps as draggable nodes, transitions as edges
- Render step status overlaid on the graph during live execution (via EventLog streaming)

The visual layer is a client of the SDK, not a platform component. It has no special platform access.

---

## 25. CODE SDK ARCHITECTURE

### Language Strategy

| Language | Role | Mechanism |
|---|---|---|
| Go | Primary: runtime, core SDK, native step handlers | In-process |
| Python | Secondary: subprocess step handlers, plugins | stdin/stdout JSON |
| TypeScript/Bun | Tertiary: subprocess step handlers | stdin/stdout JSON |
| Shell | Utility: one-liner steps, glue | subprocess |

### Go SDK Design Decisions

The Go SDK is structured as a single importable module (`github.com/awis/sdk`). Internal packages (`awis/internal/*`) are not importable by applications. The SDK surface is minimal and stable.

```
github.com/awis/sdk
├── awis.go          Config, NewRuntime, Runtime interface
├── workflow.go      WorkflowBuilder, WorkflowDefinition
├── step.go          Step, StepHandler, StepContext, StepResult, StepType
├── trigger.go       TriggerCondition, TriggerType
├── runner.go        WorkflowRunner
├── intelligence.go  IntelligencePort, adapters package
├── recall.go        RecallAPI
└── testing/
    ├── mock.go      Deterministic test doubles for all ports
    └── harness.go   WorkflowTestHarness (run a workflow synchronously in tests)
```

### Python Subprocess Protocol

For Python step handlers, a thin Python library (`pip install awis-step`) handles the JSON-RPC protocol. Application code implements one function:

```python
# my_step.py
from awis_step import step, StepContext, StepResult

@step(id="neurodashboard.compute-hrv")
def compute_hrv(ctx: StepContext) -> StepResult:
    records = ctx.inputs["records"]
    hrv = calculate_hrv(records)
    return StepResult(outputs={"hrv": hrv})

if __name__ == "__main__":
    step.serve()  # starts stdin/stdout JSON-RPC server
```

The AWIS runtime spawns this as `python my_step.py` and communicates via the SubprocessRunner.

### SDK Versioning

The SDK follows semver. Breaking changes to the public SDK surface require a major version bump. The runtime maintains backward compatibility for one major version behind (v1.x runtime runs v1.x and v0.x workflows).

---

## 26. OBSERVABILITY

### Execution Traces

Every workflow execution produces a trace composed of spans:

```
WorkflowTrace {
  trace_id:      UUID
  instance_id:   UUID
  definition_id: string
  spans: [
    Span {
      span_id:   UUID
      parent_id: UUID?
      step_id:   string
      status:    started | completed | failed | fallback
      started_at: Timestamp
      ended_at:   Timestamp
      duration_ms: int
      metadata:   map  // adapter used, tokens consumed, attempt number
    }
  ]
}
```

In V1, traces are stored in the EventLog (derived from execution events). V2 exports traces to OpenTelemetry-compatible collectors for distributed visualization.

### Metrics

```
awis_workflow_started_total{namespace, definition_id}
awis_workflow_completed_total{namespace, definition_id}
awis_workflow_failed_total{namespace, definition_id}
awis_step_duration_ms{namespace, definition_id, step_id, p50, p90, p99}
awis_step_failure_rate{namespace, definition_id, step_id}
awis_intelligence_requests_total{namespace, adapter, capability}
awis_intelligence_latency_ms{namespace, adapter, capability, p50, p99}
awis_signal_wait_duration_ms{namespace, definition_id, step_id}
```

V1: metrics available via `awis metrics` CLI (reads EventLog, computes on demand).
V2: metrics exposed via Prometheus-compatible /metrics endpoint.

### Structured Logging

All runtime log lines are structured JSON:
```json
{
  "ts": "2026-07-02T14:32:00.123Z",
  "level": "info",
  "component": "executor",
  "instance_id": "a1b2c3",
  "step_id": "draft-entry",
  "attempt": 1,
  "msg": "step started"
}
```

Log level is configurable: `error` (production default), `info` (debugging), `debug` (development).

### CLI Observability Commands

```
awis status [<instance-id>]          Current status of instance(s)
awis trace <instance-id>             Full execution trace
awis metrics [--namespace=<ns>]      Aggregate metrics
awis log [--instance=<id>] [--tail]  Structured log stream
awis replay <instance-id>            Re-run a completed instance in dry-run mode
awis audit [--from=<date>]           Audit log entries
```

---

## 27. TESTING STRATEGY

### Testing Layers

| Layer | What it tests | Tools |
|---|---|---|
| Unit | Individual step handlers; intelligence adapter logic | Go testing, mock StepContext |
| Integration | Workflow execution end-to-end; step interactions | WorkflowTestHarness (in-memory runtime) |
| Contract | StoragePort adapter behavior | Shared test suite run against both SQLite and Postgres adapters |
| System | Full platform startup, workflow registration, execution, signal delivery | Test binary with real SQLite |
| Experiment | Platform-level validations (E1 equivalent for AWIS) | Metrics from dogfood execution |

### WorkflowTestHarness

The harness runs a workflow synchronously in a test, stepping through execution until completion or failure:

```go
func TestCaptureDecisionWorkflow(t *testing.T) {
    h := awistesting.NewHarness(t,
        awistesting.WithMockIntelligence(fixtures.DraftResponse("drafted entry")),
        awistesting.WithStepHandler(&oip.RecordAppendHandler{Record: testRecord}),
    )

    result, err := h.Run(oip.CaptureDecisionWorkflow(), map[string]any{
        "repo_path": "/tmp/test-repo",
        "ref":       "abc123",
    })
    require.NoError(t, err)

    // Advance past the WAIT step (human confirmation)
    h.Signal(result.InstanceID, "entry_confirmed", map[string]any{
        "confirmed_entry": fixtures.ConfirmedEntry(),
    })
    h.WaitForCompletion(result.InstanceID, 5*time.Second)

    // Assert outputs
    assert.Equal(t, "D-2026-07-02-001", h.GetOutput(result.InstanceID, "entry_id"))
}
```

### Deterministic Execution Mode

The runtime supports a `--deterministic` flag (or `awis.DeterministicMode()` in tests) that:
- Fixes all timestamp sources to a provided clock
- Seeds all random ID generation with a fixed seed
- Routes all intelligence requests to the NullAdapter
- Disables the execution loop poll interval (manual advance via `h.Tick()`)

This makes test output stable and debuggable.

### Intelligence Testing

```go
// Mock that returns fixture responses per capability
mock := awistesting.NewMockIntelligence()
mock.OnDraft(fixtures.DraftResponse("test draft"))
mock.OnEmbed(fixtures.ZeroVector(1536))
mock.OnClassify("positive", 0.9)
```

### E3 Equivalent for AWIS

The E3 gate from OIP (all core commands succeed with NullAdapter) has its platform equivalent:

**AWIS-E1**: Every workflow in the test suite completes successfully with `NullAdapter` configured as the sole intelligence provider. This is a required CI gate. Any step that fails when intelligence is absent has an architectural error: it either needs a proper fallback or should not declare `required: true`.

---

## 28. DEPLOYMENT MODELS

### Mode 1: Local CLI (V1 target)

Single binary, single process, SQLite storage. The developer runs AWIS on their machine alongside their application.

```
$ awis start --namespace oip --config .awis/config.yaml
AWIS runtime started (local mode, SQLite: .awis/runtime.db)
Registered workflows: capture-decision v1.0.0, recall-decision v1.0.0
Listening for triggers...
```

The application communicates with the runtime via the embedded Go SDK (same process) or via local socket (if the application is a separate process).

### Mode 2: Local Server (V2)

The runtime exposes an HTTP API, allowing multiple local applications to connect to one runtime.

```
$ awis server --port 8080 --config ~/.awis/config.yaml
```

Applications use the HTTP SDK instead of the embedded SDK:
```go
runtime := awis.NewHTTPRuntime("http://localhost:8080", awis.Config{Namespace: "oip"})
```

### Mode 3: Cloud Deployment (V2/V3)

Single runtime binary, Postgres storage, multiple worker instances. Workers compete via optimistic locking (no central coordinator). Deployment is a standard containerized service.

```
Environment variables:
  AWIS_STORAGE_DSN=postgres://...
  AWIS_WORKERS=4
  AWIS_NAMESPACE=production
  ANTHROPIC_API_KEY=...
```

No Kubernetes required. A single VM running Docker Compose is sufficient for V2 cloud deployment.

### Binary Distribution

AWIS ships as a single static binary (Go). No runtime dependencies. The binary includes:
- The runtime and execution engine
- The SQLite driver
- The CLI
- The Null intelligence adapter

Intelligence adapters with external API dependencies are included but inert without configuration.

---

## 29. EXTENSION STRATEGY

### Three Extension Points

| Extension Point | What it adds | Mechanism |
|---|---|---|
| Step handlers | New native step types for an application | Implement StepHandler; register at startup |
| Plugins | New external capability integrations | awis-plugin.yaml + subprocess server |
| Intelligence adapters | New AI providers | Implement IntelligencePort; register in Config |

### What Requires No Extension

- New workflow definitions (register via WorkflowRegistry — no code change to platform)
- New namespace (declare at init — no platform change)
- New trigger types in V2 (trigger adapters follow the same plugin model)

### Adding a New Intelligence Provider

```go
// 1. Implement the interface
type MyProviderAdapter struct { /* ... */ }
func (a *MyProviderAdapter) Draft(...) {...}
func (a *MyProviderAdapter) IsAvailable() bool {...}
// ... all interface methods

// 2. Register at runtime initialization
runtime := awis.NewRuntime(awis.Config{
    Intelligence: awis.IntelligenceConfig{
        Adapters: []IntelligencePort{
            awis.NewAnthropicAdapter(cfg),
            &MyProviderAdapter{APIKey: os.Getenv("MYPROVIDER_KEY")},
            awis.NewNullAdapter(),
        },
    },
})
```

No platform code changes. New providers are user-space code.

### Extension Anti-patterns (Explicitly Rejected)

- **Forking the runtime**: Never. Platform patches go upstream; applications extend via the SDK.
- **Direct database access by applications**: Never. StoragePort is the only entry point.
- **Intelligence provider code inside the platform binary**: Adapters are in the `intelligence/adapters/` package, not the core runtime. New adapters ship as separate packages.

---

## 30. ARCHITECTURE DECISION RECORDS

### ADR-001: Step as the Fundamental Workflow Primitive

**Problem:** What is the smallest reusable unit of work in AWIS?

**Evidence:** Temporal uses Activities; Airflow uses Operators; n8n uses Nodes; GitHub Actions uses Steps. All converge on an atomic, named, typed unit.

**Candidates:** Task (Celery), Activity (Temporal), Operator (Airflow), Node (LangGraph), Step (GitHub Actions / Kestra)

**Trade-offs:**
- Activity (Temporal): excellent semantics but requires full Temporal infrastructure
- Operator (Airflow): Python-specific; DAG-centric; scheduling focus
- Node (LangGraph): AI-workflow-specific; couples to graph execution model
- Step: language-agnostic; simple; understood; composable

**Rejected alternatives:** Coroutine (Temporal's durable execution model is powerful but requires infrastructure a solo founder cannot maintain); Function (too granular; no lifecycle); Saga (too high-level; composition pattern, not primitive)

**Chosen solution:** Step — an atomic, idempotent unit with typed I/O, a named handler, retry policy, and optional compensation.

**Long-term consequences:** The Step model is simple enough to visualize (each step = node in a graph), simple enough to test (mock the handler, verify outputs), and simple enough to reason about (no coroutine state to introspect). Temporal's durable execution model would be powerful but requires a coordination cluster; the pull-based step model achieves similar durability at vastly lower operational cost.

**Confidence:** High.

---

### ADR-002: Pull-Based Execution Loop over Push-Based Event-Driven

**Problem:** How does the runtime decide what to execute next?

**Evidence:** Celery, Prefect, and Dagster all use worker-pool polling models. Temporal uses push via an internal queue but still requires a coordination service. Airflow's push scheduler is a documented single-point-of-failure.

**Candidates:** Push (event bus triggers execution), Pull (workers poll), Coroutine (code-as-workflow)

**Trade-offs:**
- Push: lower latency; harder to reason about; distributed coordination complexity
- Pull: slightly higher latency (poll interval); dead simple; debuggable queue; scales by adding workers
- Coroutine: eliminates the "where was I?" problem but requires replay infrastructure

**Rejected alternatives:** Coroutine/continuation (Temporal-style) — requires a coordination cluster that a solo founder cannot reasonably operate; eliminates the option of local-only deployment.

**Chosen solution:** Pull-based loop with configurable poll interval (100ms default). Cloud mode adds optimistic locking for multi-worker competition. No central coordinator process.

**Long-term consequences:** Maximum latency is one poll interval (100ms locally; configurable lower for cloud). This is acceptable for all target applications; none require sub-100ms step initiation.

**Confidence:** High.

---

### ADR-003: Append-Only EventLog as State Source of Truth

**Problem:** How is workflow execution state persisted such that it survives crashes, supports replay, and provides audit?

**Evidence:** Event sourcing is proven in financial systems (every bank's core ledger), CQRS systems, and is the same principle as Git's object store. The OIP Record applies the same principle to decision history. Temporal uses an event journal under its durable execution model.

**Candidates:** Snapshot-only (store only current state), Event log (store all events, project current state), Hybrid (events + periodic snapshots)

**Trade-offs:**
- Snapshot-only: simplest read path; loses history; no replay; cannot audit
- Event log: complete audit; replay; debuggable; read path requires projection
- Hybrid: best of both; slightly more complexity

**Rejected alternatives:** Snapshot-only fails the "organizations cannot see themselves" requirement (FP-3) and eliminates the organizational memory value proposition.

**Chosen solution:** Append-only EventLog as source of truth + materialized StateStore as read projection. StateStore is rebuilt from EventLog if needed.

**Long-term consequences:** EventLog grows indefinitely; governed pruning is required. The projection rebuild (`awis rebuild-state`) is an operational necessity, not a recovery-only tool — operators should know it and trust it.

**Confidence:** High.

---

### ADR-004: SQLite for Local Storage; Postgres-Compatible Interface for Cloud

**Problem:** What storage backend does AWIS use?

**Evidence:** SQLite handles terabytes of data in production. It is the world's most deployed database. It has zero configuration, zero administration, and zero additional process overhead. The Library of Congress uses it. Bun, n8n, and many local-first tools have chosen SQLite for identical reasons. The upgrade path to Postgres for team deployments is well-understood (same SQL dialect plus minor extensions).

**Candidates:** SQLite, Postgres (from start), LevelDB, BoltDB, File-based

**Trade-offs:**
- Postgres from start: correct for cloud; over-engineered for local; requires a running server
- SQLite: perfect for local; no concurrent writers across processes; upgrade path to Postgres is needed

**Rejected alternatives:** File-based (no transactions; no query; no indexing); LevelDB/BoltDB (no SQL; no ecosystem; adds a novel dependency); Postgres from V1 (violates local-first principle).

**Chosen solution:** SQLite with WAL mode for local; PostgresStorageAdapter behind the same StoragePort for cloud. Migration tooling ships in V2 when cloud mode is needed.

**Confidence:** High.

---

### ADR-005: IntelligencePort with Capability-Based Routing

**Problem:** How are AI intelligence providers integrated without making any one of them a dependency?

**Evidence:** The OIP Engineering Architecture Blueprint established this pattern and it survives scrutiny at platform scale. Langchain's model abstraction is the widely-deployed precedent; its weakness (over-abstraction) is avoided here by keeping the interface narrow (five methods) and capability-declared at the step level.

**Candidates:** Direct SDK use (Anthropic/OpenAI), LangChain abstraction, Custom port interface, Provider-per-step configuration

**Rejected alternatives:**
- Direct SDK use: provider leakage throughout the codebase; migration = rewrite
- LangChain: over-abstracted; 600+ dependencies; adds a moving abstraction on top of the underlying APIs
- Provider-per-step configuration: more granular but more config burden; CapabilityRouter handles this transparently

**Chosen solution:** IntelligencePort interface with four adapters (Anthropic, OpenAI, Ollama, Null) and capability-based routing. Provider = one config key.

**Confidence:** High.

---

### ADR-006: Plugin Communication via stdin/stdout JSON-RPC

**Problem:** How do plugins communicate with the AWIS runtime?

**Evidence:** Git hooks use this model. SQLite loadable extensions use shared libraries (the alternative). LSP (Language Server Protocol) uses stdin/stdout JSON-RPC for IDE plugins with tremendous success (cross-language, stable, debuggable).

**Candidates:** Shared library (dlopen), HTTP local server, Unix domain sockets, stdin/stdout JSON-RPC, gRPC, WASM

**Trade-offs:**
- Shared library: fastest; coupling; language-specific; crash isolation impossible
- HTTP local server: good isolation; networking overhead; port management complexity
- Unix domain sockets: fast; OS-dependent; requires lifecycle management
- stdin/stdout JSON-RPC: slightly slower; works everywhere; trivially testable; crash-isolated
- WASM: excellent isolation; complex to develop; deferred to V3

**Rejected alternatives:** Shared library (violates crash isolation); gRPC (binary protocol complexity without proportionate benefit at this scale); WASM (valuable but not justified until plugin ecosystem proves need).

**Chosen solution:** stdin/stdout JSON-RPC with JSON-RPC 2.0 protocol. Plugin processes are spawned and managed by the runtime.

**Confidence:** High.

---

### ADR-007: Namespace-Per-Application Isolation

**Problem:** How do multiple applications coexist on one AWIS runtime without data bleed?

**Evidence:** Multi-tenant SaaS systems use schema-per-tenant or row-level namespace isolation. Row-level is simpler to manage and sufficient for a trusted-application model (all apps on one runtime are operated by the same person in V1).

**Candidates:** Process-per-application, Schema-per-application, Row-level namespace

**Trade-offs:**
- Process-per-application: strongest isolation; coordination complexity; resource overhead per app
- Schema-per-application: strong isolation; SQLite doesn't support multiple schemas elegantly
- Row-level namespace: simple; sufficient for V1 (trusted applications); scales to V2 with enforcement

**Chosen solution:** Row-level namespace enforced by StoragePort. V2 adds namespace-level resource quotas and explicit grants for cross-namespace access.

**Confidence:** High.

---

### ADR-008: YAML DSL as Tier 1 with Go SDK as Tier 2

**Problem:** How do applications define workflows without requiring Go expertise?

**Evidence:** Kestra, GitHub Actions, and Docker Compose all use YAML successfully for workflow/pipeline definition. YAML is universally readable. Temporal's code-only approach is powerful but requires Go/Java/Python SDK expertise. The dual-tier model (YAML for simple; code for complex) is proven in AWS Step Functions (state machine JSON + Lambda code).

**Candidates:** YAML only, Code-first only, JSON Schema, Visual-first

**Rejected alternatives:** Visual-first (requires stable model first; visual is a V3 layer); JSON Schema (verbose; no comments; poor human ergonomics); Code-only (excludes non-Go developers from defining simple workflows).

**Chosen solution:** Two tiers producing identical WorkflowDefinition structs. YAML for declarative, sequential, configuration-driven workflows. Go SDK for dynamic, conditional, code-driven workflows.

**Confidence:** High.

---

### ADR-009: Optimistic Locking for Multi-Worker Cloud Mode

**Problem:** How do multiple workers in cloud mode avoid double-executing the same step?

**Evidence:** Postgres advisory locks, SELECT FOR UPDATE SKIP LOCKED, and optimistic locking via version counters are all proven patterns. SKIP LOCKED is Postgres's native queue pattern. Optimistic locking with a version counter is simpler and requires no database-specific features.

**Chosen solution:** Version counter on `workflow_instances.version`. Workers claim steps with `UPDATE ... WHERE instance_id=? AND version=?`; only one writer succeeds. The loser retries next tick.

**Confidence:** High (for V2; V1 is single-worker and this path is never hit).

---

### ADR-010: OIP is the First AWIS Application

**Problem:** What is V1 of AWIS, and how does OIP relate to it?

**Evidence:** The best way to validate a platform is to build a real application on it. OIP was already designed (ENGINEERING_ARCHITECTURE_BLUEPRINT.md) and has clear workflow structure: capture-decision and recall-decision are the two primary workflows. Building OIP on AWIS proves the SDK boundary and tests the platform's expressiveness.

**Chosen solution:** OIP V1 is redesigned as an AWIS application. OIP's six modules (Capture Surface, Context Assembler, Drafting Intelligence, The Record, Index, Recall) map to:
- Capture Surface → AWIS trigger (git hook event trigger)
- Context Assembler → AWIS plugin (git-context-plugin)
- Drafting Intelligence → AWIS intelligence step (draft capability)
- The Record → OIP-owned file store (OIP's RecordPort, not AWIS storage)
- Index → OIP-owned SQLite FTS5 index (in OIP's namespace)
- Recall → AWIS workflow (recall-decision: search → semantic-rank → synthesize → respond)

OIP's Record remains OIP's irreversible artifact. AWIS provides the execution infrastructure.

**Confidence:** High.

---

### ADR-011: No Central Coordinator Process

**Problem:** Does AWIS need a central scheduler process separate from the workers?

**Evidence:** Airflow's central scheduler is its #1 scaling and reliability problem. Temporal requires a coordination cluster. Prefect's agents are self-scheduling. Celery workers are self-directing.

**Chosen solution:** No central coordinator. Every worker runs its own execution loop. In local mode: one loop. In cloud mode: multiple loops compete via optimistic locking. No separate scheduler process.

**Confidence:** High.

---

### ADR-012: Workflow Versioning via Immutable Semver

**Problem:** What happens when a workflow definition changes while instances are running?

**Evidence:** Temporal's versioning API is complex (change functions). Apache Camel uses route versioning. The simplest correct approach is version pinning at creation time.

**Chosen solution:** Each WorkflowInstance stores the definition version at creation. Running instances execute against their creation-time version until completion. New definitions register as new versions. No forced migration. Migration tooling (`awis migrate-instances`) is a V2 feature when migrations first become necessary.

**Confidence:** High.

---

### ADR-013: Two-Process Step Execution Model (Native + Subprocess)

**Problem:** How do non-Go step handlers execute?

**Evidence:** Windmill's script-as-step model runs each step as an isolated process. GitHub Actions runners execute each step as a subprocess. This is proven and language-agnostic.

**Chosen solution:** Native (Go StepHandler interface, in-process) for performance-critical paths. Subprocess (stdin/stdout JSON exchange) for Python, TypeScript, and shell steps. WASM deferred to V3 for untrusted third-party steps.

**Confidence:** High.

---

### ADR-014: Local-First with No Network Assumptions

**Problem:** Can AWIS be used without internet access?

**Evidence:** OIP's constitutional requirement (zero-AI viability, Article 32) extends to the platform: any workflow must execute with only local resources. NullAdapter + SQLite + local handlers = fully offline operation.

**Chosen solution:** All network-dependent components are adapters (IntelligencePort adapters for cloud AI, StoragePort adapter for Postgres). In offline mode: NullAdapter + SQLiteStorageAdapter + SubprocessRunner with local scripts = full functionality.

**Confidence:** High.

---

### ADR-015: Go as Primary Implementation Language

**Problem:** Which language implements the AWIS runtime and core SDK?

**Evidence:** Go is a compiled, statically typed, cross-platform language with excellent concurrency primitives (goroutines, channels), minimal runtime overhead, and single-binary distribution. It is the language of infrastructure tools: Docker, Kubernetes, Terraform, Consul, Vault, InfluxDB, CockroachDB. For a runtime that must be reliable, low-resource, and easily deployed, Go has no peer among the practical options.

**Trade-offs vs. Rust:** Rust provides better memory safety guarantees; Go provides faster development cycles. For a solo-founder tool, development velocity beats marginal performance.
**Trade-offs vs. TypeScript:** TypeScript has a larger ecosystem but introduces a Node.js runtime dependency, slower startup, and more complex single-binary distribution.
**Trade-offs vs. Python:** Python is excellent for scripting; poor for a runtime (GIL, slow startup, dependency management complexity).

**Chosen solution:** Go for the runtime, core SDK, and CLI. Python and TypeScript are first-class citizens for step handlers via the subprocess protocol.

**Confidence:** High.

---

## 31. TRADE-OFF ANALYSIS

### T1 — Simplicity vs. Capability: Pull-Based Loop vs. Push-Based Event Bus

**The tension:** A push-based event bus (Kafka, NATS) enables sub-millisecond step initiation. A pull-based loop has a minimum latency equal to the poll interval.

**Resolution:** 100ms poll interval is the right trade. All target applications (decision capture, health analytics, job automation, ledger workflows) have no sub-second latency requirement. The pull model is debuggable (you can inspect the queue), portable (no message broker to operate), and identical in local and cloud modes. Push can be added in V3 when evidence shows poll latency as a real constraint.

### T2 — Solo-Founder Sustainability vs. Correctness: Single Binary vs. Microservices

**The tension:** Microservices (separate scheduler, separate worker, separate API, separate event bus) are theoretically more scalable. A monolith binary is actually maintainable by one person.

**Resolution:** Monolith binary. Temporal's scheduler, history service, frontend, and matching service require a team to operate. AWIS's single binary with a configurable worker count achieves 80% of Temporal's capability at 5% of the operational complexity. The separation point (when to split) is evidence-driven, not pre-designed.

### T3 — Plugin Isolation vs. Plugin Performance: Subprocess vs. In-Process

**The tension:** Subprocess plugins pay a process spawn overhead (~10ms) and stdin/stdout serialization overhead per call. In-process plugins (shared library) have zero overhead.

**Resolution:** Subprocess isolation wins. A crashing shared library takes down the runtime. For long-running plugins (spawned once, many calls), the overhead is per-initialization, not per-call. For high-frequency steps that need performance, the pattern is to implement them as native Go step handlers, not plugins.

### T4 — Complete History vs. Storage Growth: Append-Only EventLog vs. Snapshots

**The tension:** The EventLog grows indefinitely. At high volume, storage cost becomes real.

**Resolution:** Append-only is correct; managed pruning is the solution. Governed pruning (`awis prune-events --before=<date>`) with pre-pruning export (`awis export --before=<date>`) preserves the data-outlives-code principle while managing storage. V2 adds configurable retention policies per namespace.

### T5 — Intelligence Optionality vs. Product Value: AI-Optional vs. AI-Dependent

**The tension:** Making AI optional (NullAdapter) means the system degrades gracefully but may degrade below usefulness. T4 from the OIP PRD (without AI, OIP's advantage over a folder of markdown approaches zero) applies here too.

**Resolution:** AI-optional is architecturally correct; the business bet is that AI will be available and improve. The platform's job is not to mandate AI but to make AI replacement cheap. If a particular application's value proposition requires AI (OIP's E1 hypothesis), that is an application-level constraint, not a platform constraint.

### T6 — Namespace Isolation vs. Cross-Application Intelligence: Data Walls vs. Platform Value

**The tension:** Strict namespace isolation prevents applications from sharing execution history. Cross-application recall is potentially the most valuable platform capability (OIP's decision history informing NeuroDashboard's protocol decisions).

**Resolution:** V1 strict isolation; V2 explicit governed grants. The value is real; the risk (unauthorized cross-namespace access, data bleed, compliance violation) is also real. Trust must be earned per-domain, per grant. Cross-namespace access is a governed act, not a default.

---

## 32. RISK REGISTER

| ID | Risk | Probability | Impact | Irreversibility | Mitigation |
|---|---|---|---|---|---|
| R1 | EventLog format locked in by early implementation; migration becomes painful | High | High | Very High | Design format with `schema_version` field from day 1; migration tooling before first production deployment |
| R2 | OIP's RecordPort (file store) doesn't fit cleanly on AWIS SDK; platform-application boundary is wrong | Medium | High | Medium | Build OIP on AWIS first; if boundary requires surgery, redesign before adding second application |
| R3 | Plugin subprocess overhead too high for performance-sensitive steps | Medium | Medium | Low | Profile early; promote hot plugins to native Go handlers; WASM upgrade path in V3 |
| R4 | Intelligence adapter interface too narrow; new provider capability doesn't fit | Medium | Medium | Medium | Keep interface minimal; new capabilities added as optional interface extensions; don't try to model all possible AI capabilities upfront |
| R5 | SQLite optimistic locking insufficient for high-contention cloud workloads | Low | Medium | Medium | V1 is local (no contention); V2 adds SKIP LOCKED for Postgres before contention is an issue |
| R6 | YAML DSL becomes limiting; complex applications forced into unreadable YAML | Medium | Low | Low | Go SDK is always available; YAML is strictly optional. Applications that need complexity use the SDK |
| R7 | Namespace model too coarse; two applications in same namespace need different isolation | Low | Medium | Low | Namespaces are cheap; if needed, one application uses two namespaces (e.g., "oip-prod", "oip-staging") |
| R8 | Signal delivery reliability; signals arrive while runtime is down; lost | Medium | High | Low | Signal inbox is persisted in SQLite before delivery; redelivery on startup; signal idempotency via signal_id |
| R9 | Workflow learning model (V2+) produces bad routing decisions; degrades instead of improves | Low | Medium | Low | Learning is additive (opt-in routing hints); always falls back to static config; kill switch: `routing.learning: false` |
| R10 | Solo-founder bus factor: platform becomes unmaintainable over time | Medium | High | Medium | Minimal core (P9); no unnecessary abstraction; well-structured Go with clear module boundaries; ADRs for all decisions |

---

## 33. EVOLUTION ROADMAP

### V1 — The Foundation (Weeks 1–8)
**Scope:** AWIS runtime + OIP as first application

**Deliverables:**
- Step executor (NativeRunner + SubprocessRunner)
- EventLog + StateStore (SQLite)
- WorkflowEngine (pull-based execution loop)
- Signal protocol (WAIT steps + CLI delivery)
- IntelligencePort (Anthropic + Null adapters)
- Go SDK (WorkflowBuilder, StepHandler, WorkflowRunner)
- YAML DSL (parser + validator)
- OIP built on AWIS (capture-decision + recall-decision workflows)
- CLI: `awis start`, `awis submit`, `awis signal`, `awis status`, `awis trace`
- Observability: structured logging + `awis metrics`

**Validation gate:** OIP's E1 experiment passes (≥5 confirmed entries/week/team past week 4) using AWIS as its execution substrate.

### V2 — Multi-Application (Months 3–6)
**Scope:** Second + third applications; namespace enforcement; cloud mode

**Deliverables:**
- Plugin system (subprocess JSON-RPC + PluginRegistry)
- Multi-namespace enforcement + namespace quotas
- PostgresStorageAdapter + migration tooling
- Server mode (HTTP API + multi-process workers)
- OpenAI + Ollama intelligence adapters
- Workflow versioning enforcement
- Cost-aware intelligence routing
- Event bus persistence (SQLite-backed)
- NeuroDashboard or Shade Ledger as second application (validates namespace isolation)

**Validation gate:** Two applications run on one AWIS instance; EventLog entries from one are not visible to the other via the SDK.

### V3 — Cloud and Autonomy (Months 7–18)
**Scope:** Multi-tenant; visual layer; WASM sandboxing; charter model

**Deliverables:**
- Multi-tenant workspace isolation
- WASM step sandbox (for untrusted third-party steps)
- Visual workflow designer (read-only initially; edit in V3.1)
- Trust ladder implementation (charter model from OIP Constitution)
- Cross-namespace governed grants
- Pattern extraction from EventLog (analytics engine)
- Workflow learning model (adaptive routing)
- Prometheus metrics export
- OpenTelemetry trace export

### V4+ — The Platform
**Scope:** Predictive optimization; marketplace; protocol

**This phase is deliberately not designed now.** V4 decisions depend on V3 evidence. Designing V4 now would be premature (FP-9: complex systems that work evolve from simple ones that worked).

---

## FINAL CANONICAL ARCHITECTURE

### The Answer in One Statement

AWIS is a **pull-based durable step execution runtime** over an **append-only event log**, with an **intelligence capability layer** behind a provider seam and an **application SDK** that lets any application define workflows, register step handlers, and query execution history — without touching platform internals.

### The Six Core Structures

```
Step              — atomic, idempotent work unit; typed I/O; named handler; retry policy
WorkflowDefinition — directed graph of Steps with triggers, transitions, and compensation plan
WorkflowInstance   — a running workflow pinned to a definition version; status machine state
ExecutionEvent     — append-only record of every state change; source of truth
IntelligencePort   — interface over all AI providers; four adapters; capability-declared at step level
StoragePort        — interface over all storage backends; SQLite local; Postgres cloud
```

### The Five Layer Responsibilities

```
Workflow Engine    — determines what runs next (pull-based, 100ms tick)
Step Runtime       — executes one step in the right runner (native/subprocess/plugin/intelligence)
Intelligence Layer — routes capability requests to available providers; enforces fallback chains
Persistence Layer  — appends events, upserts state, caches results
Observability      — traces every step, collects metrics, maintains audit log
```

### The Three Extension Points

```
StepHandler interface  — Go: implement and register; non-Go: subprocess JSON protocol
Plugin manifest        — any language; subprocess JSON-RPC; declared capabilities
IntelligencePort impl  — implement the interface; register in Config.Intelligence.Adapters
```

### How OIP Lives on AWIS

```
OIP Application (namespace: "oip")
├── Workflow: capture-decision
│   ├── Step: assemble-context   [plugin: git-context-plugin]
│   ├── Step: draft-entry        [intelligence: draft capability]
│   ├── Step: confirm-entry      [signal: entry_confirmed, timeout: 72h]
│   └── Step: append-to-record   [native: oip.record.append]
│
├── Workflow: recall-decision
│   ├── Step: fts-search         [native: oip.index.fts]
│   ├── Step: semantic-rank      [native: oip.index.semantic (in-process cosine)]
│   └── Step: synthesize-answer  [intelligence: synthesize capability]
│
├── Step Handlers: RecordAppendHandler, IndexFTSHandler, SemanticRankHandler
├── Plugin: git-context-plugin
└── Data Stores:
    ├── .decisions/entries/*.md  (OIP Record — OIP-owned plain files, in git)
    ├── oip.db                   (OIP application SQLite — OIP-owned, git-ignored)
    │   ├── entries              (entry metadata for search)
    │   ├── entries_fts          (FTS5 virtual table for full-text search)
    │   └── entry_vectors        (Float32 BLOBs for cosine similarity)
    └── .awis/runtime.db         (AWIS execution state — separate file, git-ignored)

**Storage boundary:** OIP's `oip.db` and AWIS's `runtime.db` are independent SQLite databases.
AWIS's StoragePort interface covers `runtime.db` only (EventLog, StateStore, WorkflowRegistry,
StepResultCache, signal_inbox). OIP's native step handlers open `oip.db` directly; there is no
cross-database query. OIP's index is rebuildable from `.decisions/entries/*.md` via a re-index
workflow submission. AWIS's StateStore is rebuildable from its EventLog via `awis rebuild-state`.
```

### The Six-Week Implementation Sequence

**Days 1–3: Core structures only (no execution yet)**
Define ExecutionEvent schema, WorkflowDefinition schema, Step schema. Review for schema_version fields. Write the EventLog format spec (E5-equivalent for AWIS). Sign off.

**Days 4–7: EventLog + StateStore (Track A) || IntelligencePort + NullAdapter (Track B)**
Track A: SQLite schema, StoragePort implementation, EventLog append/read, StateStore upsert/query.
Track B: IntelligencePort interface, NullAdapter, AnthropicAdapter skeleton, CapabilityRouter (capability match only, no routing logic yet).

**Week 2: Execution Engine**
Pull-based execution loop. NativeRunner (Go StepHandler). Signal handler + WaitRecord persistence. WorkflowEngine ties it together. AWIS-E1 gate: all steps complete with NullAdapter in CI.

**Week 3: SDK + YAML DSL**
WorkflowBuilder Go SDK. YAML parser → WorkflowDefinition. WorkflowTestHarness (deterministic in-memory runner). WorkflowRegistry (SQLite-backed).

**Week 4: SubprocessRunner + Plugin system skeleton**
Subprocess step handler protocol. git-context-plugin (Python). SubprocessRunner in the execution engine. Plugin manifest loading and registration.

**Week 5: OIP on AWIS**
Build OIP's two workflows on AWIS. Register OIP's step handlers against the SDK. Replace OIP's bespoke execution with AWIS's runner. If the SDK surface requires surgery, redesign before proceeding.

**Week 6: CLI + dogfood**
`awis start`, `awis submit`, `awis signal`, `awis status`, `awis trace`, `awis metrics`. Dogfood OIP for one week. Measure E1 equivalent metrics. Confirm the platform boundary is correct.

### The Canonical Verdict

> **The Step is UNIX's process. The EventLog is Git's object store. The IntelligencePort is HTTP's TCP — a seam that makes the layer above it irrelevant to the layer below.** A solo founder with six weeks builds the Step executor, the EventLog, and the SDK. Then builds OIP on top. If OIP's capture-and-recall loop can be expressed as two clean workflows without platform surgery, the boundary is correct. If surgery is required, the boundary moves — not the application. Everything built in weeks one through four is reused across every subsequent application. That is the entire economic argument for a platform.

---

*Produced by AWIS Architecture Investigation Tribunal v2.0 — 2026-07-02*
*Authority: OIP_CONSTITUTION.md (Tier 0) | Status: Final canonical architecture | Next step: Implementation planning*
