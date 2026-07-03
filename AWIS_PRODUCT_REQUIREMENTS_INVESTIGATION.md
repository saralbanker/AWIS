# AWIS PRODUCT REQUIREMENTS INVESTIGATION
## Transforming the Architecture into a Product

**Produced by:** AWIS Product Strategy Tribunal v2.0
**Date:** 2026-07-02
**Authority:** AWIS_ARCHITECTURE_BLUEPRINT.md (binding constraint)
**Mode:** Product investigation — architecture is fixed; product experience is the subject
**Status:** Pre-PRD investigation — provides complete context for PRD generation

---

## TRIBUNAL PREAMBLE

Nine specialist roles operated simultaneously: Product Manager, Platform Product Architect, Developer Experience Architect, UX Architect, Workflow Platform Designer, Systems Thinker, Technical Writer, Product Researcher, Human-Computer Interaction Specialist. Research drew on product patterns from n8n, Temporal, GitHub Actions, Raycast, Linear, Supabase, and Figma — extracting principles, not features. The architecture was treated as fixed law; product decisions worked within it, never against it.

The central discipline of this investigation: **AWIS is infrastructure. Its users are developers. Its product is developer experience.** Every recommendation is filtered through that reality.

---

## 1. EXECUTIVE SUMMARY

AWIS is a **workflow intelligence platform for developers building software products**. It is not a user-facing product — it is the shared foundation beneath a portfolio of user-facing products. Its users are developers who would otherwise rebuild workflow execution, AI integration, and execution history separately in every application they ship.

The product promise is specific: **Define a workflow. Run it locally. Trust it completely.** Trust is the operative word — trust that the workflow will retry on failure, trust that every execution is recorded, trust that intelligence steps degrade gracefully when AI is unavailable, trust that `awis trace` will tell you exactly what happened when something went wrong.

V1 is CLI-first, local-first, single-namespace, and deliberately minimal. The developer experience is modeled on Git: a small vocabulary of composable commands, predictable behavior, complete local control. The runtime runs invisibly in the background; developers interact with it through familiar terminal workflows.

The platform serves three distinct developer personas — Platform Builder, Application Developer, and Plugin Developer — with different interaction patterns that share a single mental model. End users of applications built on AWIS are not AWIS users; they interact with the application layer and may never know the platform exists.

**The product's competitive moat is not features.** It is the combination of: zero infrastructure to operate locally, complete execution history that enables genuine debugging, and an intelligence abstraction that makes AI optional rather than load-bearing. No existing workflow tool offers all three.

---

## 2. PRODUCT DEFINITION

### What AWIS Is

AWIS is a locally-hosted workflow execution platform with an optional intelligence layer. Developers use it as a shared runtime foundation for multiple software products, eliminating the need to rebuild step execution, failure recovery, state persistence, AI integration, and execution history in every application.

A developer building OIP, NeuroDashboard, Shade Ledger, and Job Application Automation on AWIS implements workflow infrastructure once and ships four applications on it. Without AWIS, they implement four private, incompatible versions of the same infrastructure — and maintain all four indefinitely.

### What AWIS Is Not

| Not This | Because |
|---|---|
| A no-code automation tool (Zapier/Make) | AWIS requires developers; business users are not its audience |
| A visual workflow builder | V1 is CLI-first; visual is a V3 layer on a stable data model |
| A managed cloud service | V1 runs on a single laptop with zero external dependencies |
| An AI-first product | Intelligence is optional infrastructure; NullAdapter is the default |
| An application framework | AWIS owns execution; applications own business logic |
| A monitoring dashboard | Observability is a terminal-native experience (V1); web dashboard is V3 |
| A BPM/BPMN system | BPMN is enterprise-workflow theater; AWIS is developer workflow reality |
| A data pipeline tool | Airflow/Dagster are data-centric; AWIS is application-workflow-centric |
| A process automation platform | UiPath/Automation Anywhere target business operations; AWIS targets software applications |

### The Product in One Sentence

AWIS is the workflow engine developers never want to rebuild, made local, AI-optional, and fully observable.

---

## 3. PRODUCT PHILOSOPHY

### Three Inviolable Product Truths

**Truth 1: The runtime is invisible; the experience is everything.**
Developers don't care that AWIS uses a pull-based execution loop, SQLite, or IntelligencePort. They care that their workflow ran, that when it failed they could see why, and that adding AI required only an API key. The runtime's sophistication must manifest as experience simplicity, not experience complexity.

**Truth 2: Trust is built in microseconds and lost in minutes.**
A developer who runs `awis submit` and gets a cryptic error will close the terminal and not return. A developer who runs `awis submit` and gets a clear status line, then `awis trace` and sees a precise timeline with adapter names and token counts, will build their next application on AWIS. The debugging experience is the product.

**Truth 3: Intelligence is a power level, not a mode switch.**
When intelligence is absent (NullAdapter, no API key), workflows run and fall back silently. When intelligence is added (one environment variable), the same workflows become smarter — drafts appear, classifications surface, syntheses answer questions. The developer's workflow definition does not change. This is what "AI-optional" means in product terms: the same interface at every power level.

### The Compounding Effect

AWIS's business case is compounding knowledge transfer. When a developer learns `awis trace` debugging on OIP, they use the same tool to debug NeuroDashboard. When they configure the AnthropicAdapter for OIP, they configure the same interface for Shade Ledger. When they write a Go StepHandler for one application, the pattern is identical for the next. Every hour spent learning AWIS pays dividends across every application in the portfolio.

This compounding effect is the product's deepest value proposition and the reason it is worth building as a platform rather than embedding workflow logic in each application.

---

## 4. PRODUCT VISION

### The V1 Vision (Weeks 1–8)

"A developer can define a multi-step workflow with a human confirmation gate, run it locally, watch it execute step by step, debug failures in a single command, and add AI drafting with one environment variable — all in under 30 minutes from zero."

This vision is deliberately narrow. It excludes: cloud deployment, multiple namespaces, visual design, team collaboration, cost tracking, and any feature that requires more than one developer. The goal is mastery before scale.

### The V2 Vision (Months 3–6)

"A small team can run AWIS as a shared server, build multiple applications in isolated namespaces, use different intelligence providers per application, install plugins from a registry, and deploy to a cloud environment without changing their workflow definitions."

### The V3 Vision (Months 7–18)

"A growing organization can onboard new applications in minutes, visualize workflow graphs in a browser, define charter-bounded machine actors, and let AWIS's adaptive routing optimize intelligence calls based on observed performance — while retaining complete execution history and exit rights."

### The Decade Vision

By 2031, AWIS is the substrate layer for a portfolio of software products, the way PostgreSQL is the substrate for web applications. It is not the product customers see; it is the platform that makes every product the developer builds more reliable, more observable, and more intelligent than it would be without it.

---

## 5. PRODUCT BOUNDARIES

### AWIS Owns

- Step execution lifecycle (scheduling, running, retry, compensation)
- Execution history (event log, trace, replay)
- Intelligence routing (capability matching, fallback chains, provider abstraction)
- Plugin lifecycle (spawn, monitor, restart, deprecate)
- Signal delivery (WAIT step unblocking)
- State management (workflow instance status, variables, version pinning)
- Observability (structured logs, metrics, audit log)
- CLI (the developer's primary interface to all of the above)

### AWIS Does Not Own

- Business entities (OIP's Record entries, NeuroDashboard's health metrics, Shade Ledger's transactions)
- Domain logic (the rules that govern those entities)
- Application UX (what the end user of OIP sees)
- Intelligence provider credentials (these live in the environment, not in AWIS)
- Data portability of application data (each application manages its own export/import)
- Authentication and authorization (V1; inherited from deployment environment)

### The Bright Line

The cleanest test: if you could run the same workflow definition against a different application's step handlers and get sensible results, the workflow definition belongs to AWIS's model. If the workflow definition makes no sense without specific application knowledge, it belongs to the application.

Practical example: a workflow that runs `assemble-context → draft → confirm → append` is AWIS territory (the shape is application-agnostic). The step handler for `oip.record.append` belongs to OIP.

---

## 6. USER PERSONAS

### Persona 1: The Platform Builder

**Who:** A solo founder or lead developer who sets up and maintains the AWIS runtime for their product portfolio. In a solo-founder context, this is the same person as Persona 2, wearing a different hat.

**Goals:**
- Initialize AWIS once; have it serve all applications without repeated setup
- Configure intelligence providers and plugins once at the platform level
- Monitor overall runtime health without wading through application-level details
- Recover from failures (storage corruption, crashes) without data loss
- Export or migrate execution history as the platform grows

**Pain points without AWIS:**
- Re-implementing retry logic, state persistence, and failure recovery in each new product
- No unified view of what's running across all applications
- AI integration reimplemented differently in every codebase
- No consistent debugging workflow across products

**Primary interactions:**
- `awis init` — project initialization
- `awis start / stop` — runtime management
- `awis config` — intelligence and storage configuration
- `awis plugin install` — extending platform capabilities
- `awis rebuild-state` — disaster recovery
- `awis metrics` — platform-wide health

**Mental model:** "AWIS is my platform's operating environment. I set it up; my applications run on it."

---

### Persona 2: The Application Developer

**Who:** A developer building a specific application on AWIS (OIP, NeuroDashboard, etc.). Defines workflows, writes step handlers, and ships features. Interacts with AWIS primarily through the SDK and CLI debugging tools.

**Goals:**
- Define workflows without reimplementing scheduling, retry, or state management
- Debug failed workflow instances quickly and precisely
- Test workflow behavior deterministically without a running runtime
- Add intelligence steps without deep AI integration knowledge
- Ship features that compose existing AWIS capabilities

**Pain points without AWIS:**
- Every workflow is an ad-hoc state machine with inconsistent error handling
- Debugging requires reading logs scattered across multiple services
- Adding AI requires maintaining API clients, retry logic, and fallback paths
- Testing workflow behavior requires mocking elaborate async systems

**Primary interactions:**
- Go SDK (WorkflowBuilder, StepHandler, WorkflowTestHarness) — daily development
- YAML workflow files — workflow definition for simple workflows
- `awis workflow validate` — catch errors before deployment
- `awis submit` — manual workflow triggering during development
- `awis trace <id>` — the primary debugging tool
- `awis signal <id> <name>` — unblocking WAIT steps during testing

**Mental model:** "I define what my workflow does; AWIS handles how it runs."

---

### Persona 3: The Plugin Developer

**Who:** A developer (often not Go-proficient) who extends AWIS with a new external capability — a git integration, a data API, a hardware interface, a specialized computation. May be the same person as Persona 2 for one-off integrations, or a separate developer contributing to a shared plugin.

**Goals:**
- Expose a capability to AWIS without learning the runtime internals
- Write in their preferred language (Python, shell, TypeScript)
- Test their plugin independently of a running AWIS runtime
- Declare capabilities clearly so applications can depend on them

**Pain points without the plugin system:**
- Contributing to a Go runtime requires Go proficiency
- No standard protocol for external capabilities
- Capabilities are tightly coupled to the runtime binary

**Primary interactions:**
- `awis-plugin.yaml` — capability manifest
- stdin/stdout JSON-RPC protocol — the only runtime interface
- `awis plugin install / status / remove` — lifecycle management
- Test harness (simulate JSON-RPC protocol without runtime)

**Mental model:** "I announce what I can do; AWIS calls me when needed."

---

### Persona 4: The End User (Indirect)

**Who:** The person who uses an application built on AWIS — someone who captures decisions with OIP's CLI, views analytics in NeuroDashboard, or manages applications with the Job Automation tool.

**Relationship to AWIS:** Indirect. The end user interacts with the application; the application happens to run on AWIS. The end user may never know AWIS exists. AWIS should be invisible to this persona unless something goes wrong.

**AWIS product implication:** When AWIS fails visibly (runtime down, workflow stuck), the end user sees an application error. The application developer (Persona 2) must be able to diagnose and fix it quickly with AWIS's tools. The end user experience is therefore a product quality constraint on the developer experience.

---

## 7. CORE USER STORIES

### Platform Builder Stories

**PB-1:** As a Platform Builder, I want to run `awis init` in my project directory and have a working runtime environment in under 2 minutes, so that I can start building applications immediately.

**PB-2:** As a Platform Builder, I want to configure my intelligence provider by setting an environment variable, so that all applications using the runtime get AI capabilities without per-application configuration.

**PB-3:** As a Platform Builder, I want to run `awis status` and see all active and recent workflows across all namespaces, so that I can assess platform health at a glance.

**PB-4:** As a Platform Builder, I want to run `awis rebuild-state` and have the runtime reconstruct its state from the event log, so that I can recover from storage corruption without losing execution history.

**PB-5:** As a Platform Builder, I want to install a plugin with a single command and have it immediately available to all applications, so that I extend capabilities without touching application code.

### Application Developer Stories

**AD-1:** As an Application Developer, I want to define a workflow in YAML and have it validated before registration, so that syntax errors surface before execution time.

**AD-2:** As an Application Developer, I want to implement a `StepHandler` interface in Go and register it at startup, so that my business logic runs inside AWIS workflows with retry and observability for free.

**AD-3:** As an Application Developer, I want to run `awis submit my-workflow` and see a workflow instance ID immediately, so that I can reference and track the execution.

**AD-4:** As an Application Developer, I want to run `awis trace <instance-id>` and see a precise timeline of every step — what ran, how long it took, which intelligence adapter was used, what it produced — so that I can debug failures in one command.

**AD-5:** As an Application Developer, I want to deliver a signal to a waiting workflow with `awis signal <id> signal-name`, so that I can unblock human confirmation steps during development and testing.

**AD-6:** As an Application Developer, I want to write workflow tests using `WorkflowTestHarness` that run synchronously without a live runtime, so that my CI pipeline validates workflow behavior without infrastructure.

**AD-7:** As an Application Developer, I want to see in `awis status` which step a running workflow is currently executing, so that I can verify the workflow is progressing as expected.

**AD-8:** As an Application Developer, I want intelligence steps to route to fallback steps when no provider is configured, so that my workflow runs in all environments including those without API keys.

**AD-9:** As an Application Developer, I want to query execution history with `awis recall` and get cited, synthesis-style answers, so that I can understand patterns in my application's workflow behavior without reading raw logs.

### Plugin Developer Stories

**PD-1:** As a Plugin Developer, I want to declare my plugin's capabilities in a YAML manifest, so that AWIS knows what my plugin can do before spawning it.

**PD-2:** As a Plugin Developer, I want to implement my plugin in Python with a minimal JSON-RPC wrapper, so that I can write capability logic in my preferred language.

**PD-3:** As a Plugin Developer, I want to test my plugin's JSON-RPC protocol with a mock runtime client, so that I can develop and validate the plugin without running AWIS.

**PD-4:** As a Plugin Developer, I want to see plugin health and call statistics with `awis plugin status`, so that I can diagnose performance issues without parsing logs.

---

## 8. PRODUCT SCOPE

### V1 In Scope

| Capability | Priority |
|---|---|
| AWIS project initialization (`awis init`) | P0 |
| Runtime start/stop (`awis start/stop`) | P0 |
| YAML workflow definition | P0 |
| Go SDK (WorkflowBuilder + StepHandler) | P0 |
| Native step execution (Go handlers) | P0 |
| Subprocess step execution (Python/shell) | P0 |
| Signal delivery (WAIT steps) | P0 |
| EventLog (SQLite) | P0 |
| StateStore (SQLite) | P0 |
| Execution trace (`awis trace`) | P0 |
| Status view (`awis status`) | P0 |
| Workflow validation (`awis workflow validate`) | P0 |
| WorkflowTestHarness (in-memory testing) | P0 |
| Null intelligence adapter (default) | P0 |
| Anthropic intelligence adapter | P1 |
| Plugin system (subprocess JSON-RPC) | P1 |
| Plugin manifest + registration | P1 |
| git-context-plugin (first reference plugin) | P1 |
| Execution history (`awis history`) | P1 |
| Structured logging (`awis logs`) | P1 |
| Aggregate metrics (`awis metrics`) | P1 |
| Workflow definition registry | P1 |
| Compensation (rollback on failure) | P1 |
| Basic recall (`awis recall`) | P2 |
| State rebuild (`awis rebuild-state`) | P2 |
| Export (`awis export`) | P2 |

### V1 Explicitly Out of Scope

| Excluded Capability | Why |
|---|---|
| Web dashboard / visual designer | Requires stable data model first (that's V1's job) |
| Multi-namespace isolation | Single developer; single app in V1 |
| Postgres storage adapter | Cloud mode is V2 |
| OpenAI / Ollama adapters | Anthropic is sufficient for V1 validation |
| Server mode (HTTP API) | V2 when second application is added |
| Multi-worker execution | V2 cloud mode requirement |
| Plugin marketplace / registry | V2; V1 supports manual install |
| Cost tracking | V2 when multiple adapters create cost complexity |
| Webhook triggers | V2 |
| WASM sandboxing | V3 |
| Charter model / trust ladder | V3 |
| Multi-tenant workspaces | V3 |
| Authentication / authorization | V3 |
| Visual workflow designer | V3 |

---

## 9. FEATURE HIERARCHY

### Tier 0 — Platform Core (must exist for AWIS to function)

- EventLog (append-only SQLite write path)
- StateStore (workflow instance state projection)
- Execution loop (pull-based scheduler)
- NativeRunner (Go step execution)
- StepHandler interface (the SDK's core contract)
- WorkflowDefinition struct (the universal workflow representation)
- Signal delivery (WAIT step unblocking)
- NullAdapter (zero-dependency intelligence default)

### Tier 1 — Developer Experience (must exist for AWIS to be usable)

- `awis init` — project setup
- `awis start / stop` — runtime lifecycle
- `awis submit` — workflow triggering
- `awis status` — the primary monitoring view
- `awis trace` — the primary debugging view
- `awis signal` — signal delivery CLI
- `awis workflow validate` — pre-registration error detection
- WorkflowTestHarness — in-process test double
- YAML DSL — non-Go workflow definition
- Structured error messages with context and remediation hints

### Tier 2 — Application Features (must exist for OIP to run on AWIS)

- SubprocessRunner (Python/shell step execution)
- Plugin system (subprocess JSON-RPC + manifest)
- Anthropic intelligence adapter
- Capability-based intelligence routing
- Fallback chain execution
- Compensation plans (rollback)
- Execution history (`awis history`)
- Metrics aggregation (`awis metrics`)
- Structured logging (`awis logs`)

### Tier 3 — Platform Growth (V2)

- Multi-namespace isolation
- Postgres storage adapter
- OpenAI + Ollama adapters
- Server mode (HTTP API)
- Plugin registry
- Cost tracking
- Webhook triggers
- Namespace quotas

### Tier 4 — Platform Maturity (V3)

- Web dashboard
- Visual workflow designer
- Multi-tenant workspaces
- Charter model
- WASM sandboxing
- Adaptive intelligence routing
- Plugin marketplace
- Authentication and authorization

---

## 10. INFORMATION ARCHITECTURE

### Primary Objects

AWIS's information architecture has two primary object types and three supporting types.

**Primary:**

```
WorkflowDefinition
  ├── id (namespaced: "oip.capture-decision")
  ├── version (semver)
  ├── steps[]
  ├── transitions[]
  ├── triggers[]
  └── compensation?

WorkflowInstance
  ├── instance_id (UUID)
  ├── definition_id + version
  ├── status (pending|running|waiting|completed|failed|cancelled|compensating)
  ├── current_steps[]
  ├── variables (accumulated step outputs)
  └── execution timeline (derived from EventLog)
```

**Supporting:**

```
ExecutionEvent     — append-only record of every state change within an instance
Plugin             — registered external capability provider
IntelligenceConfig — provider configuration and routing policy
Signal             — inbox message that unblocks a WAIT step
```

### Conceptual Hierarchy

```
AWIS Runtime
├── Namespace (default: single namespace in V1)
│   ├── WorkflowDefinitions (registered by applications)
│   │   └── Versions (immutable once registered)
│   ├── WorkflowInstances (created by trigger or manual submit)
│   │   ├── Steps (within each instance)
│   │   ├── Signals (inbox per instance)
│   │   └── Events (the execution timeline)
│   └── ExecutionHistory (queryable view of completed instances)
├── Plugins (global, shared across namespaces in V1)
├── Intelligence (configured globally, used per-step)
└── Observability (logs, metrics, audit log)
```

### Object Relationships

- One WorkflowDefinition → many WorkflowInstances
- One WorkflowInstance → many Steps → many ExecutionEvents
- One WorkflowInstance → zero or more Signals (in inbox)
- One Step → zero or one intelligence capability call
- One Step → zero or one plugin capability call
- One Plugin → many capability declarations

---

## 11. NAVIGATION STRUCTURE

### CLI Command Hierarchy

```
awis
│
├── RUNTIME MANAGEMENT
│   ├── init [name]                   Initialize project structure and config
│   ├── start [--config=<path>]       Start the runtime (foreground by default)
│   ├── stop                          Gracefully stop the runtime
│   └── version                       Show version and build info
│
├── WORKFLOW EXECUTION
│   ├── submit <workflow-id>           Submit a workflow instance
│   │   [--input='<json>']
│   │   [--async]
│   ├── signal <instance-id> <name>   Deliver signal to waiting instance
│   │   [--payload='<json>']
│   └── cancel <instance-id>          Cancel a running or waiting instance
│       [--reason=<string>]
│
├── OBSERVABILITY
│   ├── status [--namespace=<ns>]     Live status of active and recent instances
│   │   [--all]
│   ├── trace <instance-id>           Full execution trace for one instance
│   │   [--json]
│   ├── history [--n=20]              Recent completed instances
│   │   [--namespace=<ns>]
│   │   [--workflow=<id>]
│   │   [--status=failed|completed]
│   ├── logs [--instance=<id>]        Structured log stream
│   │   [--level=error|info|debug]
│   │   [--tail]
│   ├── metrics [--namespace=<ns>]    Aggregate execution statistics
│   └── recall "<natural query>"      Query execution history (intelligence-enhanced)
│       [--namespace=<ns>]
│
├── WORKFLOW MANAGEMENT
│   ├── workflow list [--namespace=x] List registered workflow definitions
│   ├── workflow show <id>            Show definition detail (steps, transitions)
│   └── workflow validate <file>      Validate a YAML workflow definition
│
├── PLUGIN MANAGEMENT
│   ├── plugin list                   List installed plugins and status
│   ├── plugin install <path|url>     Install a plugin
│   ├── plugin remove <name>          Remove a plugin
│   └── plugin status <name>          Plugin health and call statistics
│
├── CONFIGURATION
│   ├── config show                   View current configuration
│   ├── config set <key> <value>      Set a configuration value
│   ├── config validate               Validate configuration file
│   └── config edit                   Open config in $EDITOR
│
└── MAINTENANCE
    ├── rebuild-state [--namespace=x] Rebuild state projection from event log
    ├── export [--format=json|csv]    Export execution history
    │   [--namespace=<ns>]
    │   [--from=<date>] [--to=<date>]
    └── audit [--from=<date>]         View audit log entries
```

### Command Design Principles

1. **Primary commands are one word.** `awis status`, `awis trace`, `awis submit` — no sub-commands for the 80% case.
2. **Sub-commands for management domains.** `awis workflow`, `awis plugin`, `awis config` — organized namespaces for less frequent operations.
3. **All commands have a `--json` flag.** Machine-parseable output for scripting and CI integration.
4. **All commands have `--help`.** Short, scannable, with examples.
5. **No command requires reading documentation to use.** Error messages suggest the correct command when the wrong one is used.

---

## 12. USER JOURNEY

### Journey 1: First-Time Platform Setup (Platform Builder, Day 1)

```
Developer opens terminal in project directory.

Step 1: Install
  $ go install github.com/awis/awis@latest
  → "awis v1.0.0 installed"

Step 2: Initialize
  $ awis init
  → Creates: .awis/ directory
              .awis/config.yaml (with sensible defaults)
              .awis/.gitignore (ignores runtime.db, config.yaml)
              workflows/hello-world.yaml (starter example)
  → Prints: "Project initialized. Run 'awis start' to start the runtime."
             "Example workflow: workflows/hello-world.yaml"
             "Next: awis submit hello-world"

Step 3: Start
  $ awis start
  → "AWIS v1.0.0 starting..."
  → "Storage: .awis/runtime.db (SQLite)"
  → "Intelligence: null (no API key configured)"
  → "Workflows: hello-world v1.0.0"
  → "Runtime ready. Press Ctrl+C to stop."

Step 4: Submit the example workflow
  $ awis submit hello-world
  → "Submitted: hello-world / instance i-abc123"
  → "Status: awis status | Trace: awis trace i-abc123"

Step 5: See it work
  $ awis status
  → (shows instance completed in 0.1s)

Step 6: Debug it
  $ awis trace i-abc123
  → (shows full timeline: 2 steps, outputs, duration)
```

Time to first successful workflow: **under 5 minutes**.

---

### Journey 2: Building the First Real Application (Application Developer, Week 1)

```
Developer is building OIP on AWIS.

Day 1: Define the workflow
  → Create workflows/capture-decision.yaml
  → awis workflow validate workflows/capture-decision.yaml
  → Fix validation errors (clear messages with line numbers)
  → awis start (auto-registers validated workflows)
  → awis workflow list (see capture-decision v1.0.0)

Day 2: Write the step handler
  → Implement oip.RecordAppendHandler (Go, StepHandler interface)
  → Register in main.go: runtime.RegisterHandler(&RecordAppendHandler{})
  → awis submit capture-decision --input='{"repo_path": "."}'
  → awis trace <id> → see step ran, see error in append step

Day 3: Add intelligence
  → Add intelligence step to capture-decision.yaml
  → awis workflow validate (now has intelligence step)
  → awis submit → trace shows "intelligence: null, fallback: manual-entry activated"
  → export ANTHROPIC_API_KEY=sk-ant-...
  → awis start → "Intelligence: anthropic (claude-haiku)"
  → awis submit → trace shows "intelligence: anthropic, tokens: 847, draft: [...]"

Day 4: Test it properly
  → Write WorkflowTestHarness test
  → Run workflow synchronously with mock intelligence
  → Signal the WAIT step in the test: h.Signal(id, "entry_confirmed", ...)
  → Assert outputs
  → All passing in CI (no API keys required)

Day 5: Handle failures
  → awis trace <failed-instance-id>
  → See: draft-entry failed (attempt 2/3, next retry in 4s)
  → See: step failed after 3 attempts, fallback activated
  → Fix handler; redeploy; workflow self-heals on next trigger
```

---

### Journey 3: Adding a Plugin (Platform Builder + Plugin Developer, Week 2)

```
Platform Builder installs the git-context-plugin.

  $ awis plugin install github.com/awis-plugins/git-context-plugin
  → Downloads v1.0.0
  → Installs Python dependencies
  → Registers capabilities: git.context.assemble, git.diff.fetch
  → "Plugin installed: git-context-plugin v1.0.0"

  $ awis plugin status git-context-plugin
  → "Status: healthy | Capabilities: 2 | Calls: 0"

Application Developer updates workflow to use the plugin.

  # In capture-decision.yaml:
  - id: assemble-context
    type: plugin
    handler: git-context-plugin
    ...

  $ awis workflow validate workflows/capture-decision.yaml
  → "Valid: capture-decision v1.0.0 (1 plugin dependency: git-context-plugin)"

  $ awis submit capture-decision --input='{"repo_path": "."}'
  $ awis trace <id>
  → "assemble-context: plugin=git-context-plugin, duration=1.2s, outputs={context: {...}}"
```

---

### Journey 4: Debugging a Failure (Application Developer, Ongoing)

```
awis status shows a red ✗ in recent completions.

  $ awis status
  ✗ failed  capture-decision  i-m4n5o6  23s  1h ago  draft-entry: intelligence timeout

  $ awis trace i-m4n5o6
  00:00  ● WorkflowStarted
  00:00  ► StepStarted       assemble-context (attempt 1)
  00:01  ✓ StepCompleted     assemble-context  1.1s
  00:01  ► StepStarted       draft-entry (attempt 1)
  00:11  ✗ StepFailed        draft-entry  10.0s  error: request timeout (10s)
  00:11  ► StepStarted       draft-entry (attempt 2)
  00:21  ✗ StepFailed        draft-entry  10.0s  error: request timeout (10s)
  00:21  ► StepStarted       draft-entry (attempt 3)
  00:23  ✗ StepFailed        draft-entry  2.1s   error: connection refused
  00:23  ● WorkflowFailed    reason: draft-entry exhausted retries (3/3)
                              Hint: awis config set intelligence.timeout 30s

Developer sees the problem immediately, fixes the timeout config, retries.
```

This journey — **from symptom to diagnosis to fix in under 2 minutes** — is the product's most important use case. `awis trace` is the product's most important command.

---

## 13. SCREEN INVENTORY

AWIS V1 has no graphical screens. The "screen inventory" is the set of terminal output formats developers read.

| Command | Output Format | Primary User | Frequency |
|---|---|---|---|
| `awis init` | One-time setup summary | Platform Builder | Once per project |
| `awis start` | Runtime startup log | Platform Builder | Per session |
| `awis status` | Multi-section live status | Platform Builder / App Dev | Continuous / on-demand |
| `awis trace <id>` | Sequential timeline | App Dev | On failure or curiosity |
| `awis history` | Tabular list of instances | App Dev | Periodic review |
| `awis submit` | Confirmation with instance ID | App Dev | Development iteration |
| `awis signal` | Delivery confirmation | App Dev | WAIT step unblocking |
| `awis workflow list` | Tabular workflow registry | App Dev | Setup verification |
| `awis workflow show <id>` | Step graph + transitions | App Dev | Workflow introspection |
| `awis workflow validate` | Validation result with errors | App Dev | Authoring iteration |
| `awis plugin list` | Plugin status table | Platform Builder | Platform management |
| `awis plugin status <n>` | Plugin health + call stats | Platform Builder | Debugging |
| `awis metrics` | Aggregate statistics | Platform Builder | Health review |
| `awis logs` | Streaming structured logs | App Dev | Active debugging |
| `awis config show` | Current configuration | All | Setup verification |
| `awis recall "<query>"` | Cited synthesis answer | App Dev | History query |

---

## 14. DASHBOARD EXPERIENCE

### V1: The Terminal Dashboard

AWIS V1's "dashboard" is `awis status`. It must provide a complete platform health picture in a single command, scannable in under 5 seconds.

```
AWIS v1.0.0  ●  running
Namespace: oip  |  Storage: .awis/runtime.db  |  Intelligence: anthropic (claude-haiku-4-5)
Uptime: 3h 42m  |  Plugins: git-context-plugin (healthy)

────────────────────── ACTIVE ──────────────────────
 ●  running   capture-decision  i-a1b2c3  step: draft-entry     12s
 ○  waiting   capture-decision  i-d4e5f6  signal: confirm-entry  71h 48m remaining

────────────────────── RECENT ──────────────────────
 ✓  completed  recall-decision   i-g7h8i9   2.3s    2m ago
 ✓  completed  capture-decision  i-j1k2l3  89s     15m ago
 ✗  failed     capture-decision  i-m4n5o6  23s     1h ago    draft-entry (3 attempts)
 ✓  completed  recall-decision   i-p7q8r9   1.8s    2h ago

Run 'awis trace <id>' to inspect any instance.
```

**Design decisions:**
- Header: runtime health at a glance (status, intelligence, plugin health)
- Active section: only instances that are currently running or waiting — these need attention
- Recent section: last N completions with outcome, duration, and age
- Failure information: shown inline (which step failed) to enable instant `awis trace`
- Footer: the most useful next command

### V2: Web-Accessible Status Page

V2 adds a read-only HTTP endpoint (`/status`) that renders the same information as `awis status` in a browser-viewable format. Not a full dashboard — just the status view as a webpage.

### V3: Full Web Dashboard

V3 adds a complete web dashboard with:
- Live workflow status (WebSocket-updated)
- Execution history with filtering and search
- Workflow definition viewer (graph visualization)
- Metrics charts (step latency, completion rates, failure rates)
- Plugin management UI
- Intelligence configuration UI

---

## 15. WORKFLOW AUTHORING EXPERIENCE

### YAML Authoring (Tier 1 DSL)

The YAML authoring experience is modeled on GitHub Actions: familiar format, immediate validation feedback, clear error messages.

**Authoring flow:**

```
1. Create workflows/my-workflow.yaml in project root
2. $ awis workflow validate workflows/my-workflow.yaml
   (output on valid)
   ✓ my-workflow v1.0.0 is valid
     Steps: 4 (native: 2, intelligence: 1, signal: 1)
     Intelligence: draft capability required (using: anthropic or fallback: manual-entry)
     Plugins: none
   
   (output on error)
   ✗ Validation failed: workflows/my-workflow.yaml
     Line 23: step 'confirm-entry' references unknown signal type
       Hint: type must be 'signal'; wait_signal.name declares the expected signal name
       Example:
         - id: confirm-entry
           type: signal
           wait_signal:
             name: entry_confirmed
             timeout: 72h
   
3. Errors are fixed; re-validate until clean
4. $ awis start
   → Auto-discovers and registers all workflows in ./workflows/
   → "Registered: my-workflow v1.0.0"
```

**Authoring principles:**
- Error messages include the line number, a description of the problem, and a corrected example
- Validation catches: missing required fields, unknown step types, unreachable steps, undefined handler references, invalid schema expressions
- Validation does NOT require intelligence providers or plugins to be available (handler references are validated against registered handlers, which requires the runtime to be running)
- `awis workflow validate` works without a running runtime (validates syntax and schema; defers handler existence checks to registration time)

### Go SDK Authoring (Tier 2)

The Go SDK authoring experience is modeled on a fluent builder API with immediate compile-time feedback where possible and clear runtime errors at registration where not.

```go
// Simple, readable, IDE-autocomplete-friendly
workflow := awis.NewWorkflowBuilder("oip.capture-decision", "1.0.0").
    WithNamespace("oip").
    WithTrigger(awis.ManualTrigger()).
    WithTrigger(awis.EventTrigger("git.push.completed").
        Where("event.branch == 'main'")).
    AddStep(awis.Step{
        ID:      "assemble-context",
        Type:    awis.StepTypePlugin,
        Handler: "git-context-plugin.git.context.assemble",
        Inputs:  awis.Inputs("repo_path", awis.Ref("workflow.inputs.repo_path")),
        Outputs: awis.Outputs("context", awis.ObjectType()),
        Retry:   awis.ExponentialRetry(3),
    }).
    AddStep(awis.Step{
        ID:   "draft-entry",
        Type: awis.StepTypeIntelligence,
        Intelligence: &awis.IntelReq{
            Capability:    awis.CapabilityDraft,
            ContextBudget: 3000,
            Required:      false,
        },
        Fallback: "manual-entry",
    }).
    // ...
    Build() // Returns (WorkflowDefinition, error)
```

**Registration produces clear errors:**

```
awis: registration failed for "oip.capture-decision"
  step "assemble-context": handler "git-context-plugin.git.context.assemble" not found
  Hint: install the plugin first: awis plugin install git-context-plugin
  Available plugins: none installed
```

### Workflow Versioning in Authoring

When a developer changes a workflow definition:
- Increment the version field (`"1.0.0"` → `"1.1.0"`)
- Re-register: `awis start` (or runtime detects file change if watch mode is active)
- Old instances continue on v1.0.0; new instances use v1.1.0
- `awis workflow list` shows both versions with instance counts

---

## 16. WORKFLOW EXECUTION EXPERIENCE

### Manual Execution

```
$ awis submit capture-decision --input='{"repo_path": "/path/to/repo", "ref": "abc123"}'
Submitted: capture-decision v1.0.0
Instance:  i-a1b2c3
Status:    pending → running

Monitor: awis status
Debug:   awis trace i-a1b2c3
```

Simple. Immediate. Actionable.

### Triggered Execution

When a trigger fires automatically (git push event, schedule, webhook in V2), the execution appears in `awis status` without developer action. The developer can always see what triggered an instance with `awis trace`:

```
Trace: capture-decision / i-d4e5f6
Status: completed ✓  Duration: 89s  Trigger: git.push.completed (ref: abc123)

Timeline:
  00:00  ● WorkflowStarted   trigger: git.push.completed
  ...
```

### WAIT Step Experience

When a workflow reaches a WAIT step:

```
$ awis status
 ○  waiting   capture-decision  i-d4e5f6  signal: confirm-entry  71h 48m remaining
```

The developer delivers the signal:

```
$ awis signal i-d4e5f6 entry_confirmed --payload='{"confirmed_entry": {...}}'
Signal delivered: entry_confirmed → i-d4e5f6
Instance resumed; next step: append-to-record
```

### Fallback Activation (Intelligence Unavailable)

When an intelligence step routes to its fallback, the execution continues without interruption and the fallback is visible in the trace:

```
00:01  ► StepStarted       draft-entry (attempt 1)
00:01  → IntelligenceFallback  draft-entry: capability 'draft' unavailable (null adapter)
00:01  ► StepStarted       manual-entry (fallback activated)
00:01  ○ WaitingForSignal  manual-entry: signal 'manual_draft_provided'
```

The developer can see exactly what happened without parsing logs. The fallback is not an error — it is an explicit, visible execution path.

---

## 17. WORKFLOW MONITORING EXPERIENCE

### The Status View (Primary)

`awis status` is the first command developers run when something seems wrong. It must answer three questions in one glance:
1. Is the runtime healthy?
2. Is anything stuck or failing?
3. What completed recently and how did it go?

Design: color-coded (but never color-only — all information is also in text symbols), sortable by age, filterable by status. In V1, auto-refreshes every 5 seconds with `awis status --watch`.

### The Trace View (Debugging)

`awis trace <id>` is the surgical debugging tool. It must answer: "What happened, in order, with timing, adapter usage, and outputs?"

Design principles:
- **Sequential order** — not alphabetical, not by step name, but chronological
- **Timing on every step** — duration in human-readable units (1.2s, not 1234ms)
- **Adapter metadata inline** — `adapter: anthropic, model: claude-haiku, tokens: 847`
- **Outputs visible** — show output values (truncated to reasonable length; `--full` flag for complete)
- **Failures highlighted** — error message, attempt count, next action
- **Signals visible** — when signals arrived, what payload they carried
- **Fallbacks visible** — when a fallback was activated, why

```
$ awis trace i-a1b2c3 --json | jq '.spans[] | select(.status == "failed")'
```

The `--json` flag enables programmatic trace analysis, essential for CI integration.

### The History View

`awis history` provides a list view for pattern spotting across multiple instances:

```
$ awis history --workflow=capture-decision --n=20

  ID         STATUS     DURATION  TRIGGER              AGE
  i-a1b2c3   completed  89s       manual               2h ago
  i-d4e5f6   completed  124s      git.push.completed   4h ago
  i-m4n5o6   failed     23s       git.push.completed   1h ago    draft-entry (timeout)
  i-p7q8r9   completed  95s       manual               1d ago
  ...

20 instances shown. Total: 47 (44 completed, 2 failed, 1 cancelled)
```

### The Metrics View

`awis metrics` provides aggregate health statistics, useful for identifying systemic patterns:

```
$ awis metrics --workflow=capture-decision

Workflow: capture-decision  |  Period: last 7 days  |  Instances: 47

Completion Rate    94%     (44/47 completed)
Failure Rate        4%     ( 2/47 failed)
Cancellation Rate   2%     ( 1/47 cancelled)

Step Latency (p50 / p95)
  assemble-context      1.1s  /  2.3s    0 failures
  draft-entry           2.4s  /  8.1s    2 failures (timeout)
  confirm-entry        34m   / 2.1h     0 failures (human signal)
  append-to-record      0.1s  /  0.3s    0 failures

Intelligence Usage
  draft-entry: anthropic (44 calls, avg 823 tokens, avg 2.4s)
               fallback activated: 3 times (null adapter or timeout)
```

---

## 18. PLUGIN EXPERIENCE

### Installation

Plugin installation is modeled on `brew install` — one command, visible progress, immediate availability.

```
$ awis plugin install git-context-plugin
Fetching plugin manifest...
Downloading git-context-plugin v1.0.0...
Checking dependencies: python3 (3.11.0 ✓), gitpython (3.1.0 → installing)
Installing dependencies...
✓ git-context-plugin v1.0.0 installed

  Capabilities:
    git.context.assemble  → git.ContextAssembleHandler
    git.diff.fetch        → git.DiffFetchHandler

  Runtime: python3 -m git_context_plugin
  Idle timeout: 5 minutes

To use in a workflow:
  type: plugin
  handler: git-context-plugin
```

### Health Monitoring

```
$ awis plugin status git-context-plugin

git-context-plugin v1.0.0  ●  healthy
Process: PID 12345, running for 2h 3m
Memory: 48 MB

Capabilities         Calls  Avg Latency  P95 Latency  Errors
git.context.assemble    34     1.2s         2.1s         0
git.diff.fetch          12     0.4s         0.8s         0

Last error: none
Config: /home/user/.awis/plugins/git-context-plugin/config.yaml
```

### Plugin Failure Behavior

When a plugin crashes, the developer sees it in the step trace:

```
00:01  ► StepStarted       assemble-context (attempt 1)
00:02  ✗ StepFailed        assemble-context  1.0s
                            error: plugin git-context-plugin crashed (exit code 1)
                            Plugin auto-restarting (1/3)...

00:02  ► StepStarted       assemble-context (attempt 2)
00:03  ✓ StepCompleted     assemble-context  0.9s  (plugin restarted successfully)
```

The plugin restart is transparent to the workflow; the step retries. If the plugin fails all restart attempts, `awis plugin status git-context-plugin` shows the crash log.

### The Plugin Developer's Perspective

A plugin developer needs three things:
1. A simple manifest format to declare capabilities
2. A simple protocol to implement (JSON-RPC over stdin/stdout)
3. A way to test the plugin without running the full runtime

```python
# Minimal Python plugin (git_context_plugin/__main__.py)
from awis_plugin import PluginServer, capability

server = PluginServer()

@capability("git.context.assemble")
def assemble_context(inputs):
    repo_path = inputs["repo_path"]
    ref = inputs["ref"]
    # ... git logic ...
    return {"context": {...}}

if __name__ == "__main__":
    server.serve()

# Test without runtime:
from awis_plugin.testing import mock_request
result = mock_request("git.context.assemble", {"repo_path": ".", "ref": "HEAD"})
assert "context" in result
```

---

## 19. INTELLIGENCE CONFIGURATION

### The Power Level Model

Intelligence in AWIS has four power levels, activated by configuration, not by code changes:

**Level 0 — No intelligence (default):**
No API key configured. NullAdapter is active. Intelligence steps route to their declared fallbacks. Every workflow runs.
```
AWIS ● running  intelligence: null (no API key configured)
```

**Level 1 — Basic cloud intelligence:**
One environment variable. The most used model is automatically selected.
```
$ export ANTHROPIC_API_KEY=sk-ant-...
$ awis start
AWIS ● running  intelligence: anthropic (claude-haiku-4-5, claude-sonnet-4-6)
```

**Level 2 — Configured intelligence:**
Model selection, context budgets, and model tiers in `config.yaml`.
```yaml
intelligence:
  primary:
    provider: anthropic
    draft_model: claude-haiku-4-5-20251001
    quality_model: claude-sonnet-4-6
```

**Level 3 — Multi-provider routing:**
Multiple adapters with routing policy. Used when cost optimization, local inference, or provider redundancy is needed.
```yaml
intelligence:
  routing:
    fallback_chain: [anthropic, ollama, null]
    capability_overrides:
      embed: [openai, ollama]
      classify: [ollama, anthropic]
```

### Configuration UX

```
$ awis config show

intelligence:
  primary.provider:     anthropic          (from: ANTHROPIC_API_KEY env var)
  primary.draft_model:  claude-haiku-4-5   (default)
  primary.quality_model: claude-sonnet-4-6 (default)
  embed.provider:       [not configured]   → embedding unavailable; semantic search disabled

storage:
  type:          sqlite
  path:          .awis/runtime.db

runtime:
  poll_interval: 100ms
  max_parallel:  4

To configure:  awis config set <key> <value>
To edit:       awis config edit
```

### Intelligence Visibility in Traces

The developer always sees which intelligence provider was used for each step:

```
✓ StepCompleted  draft-entry  2.4s
  adapter:    anthropic
  model:      claude-haiku-4-5-20251001
  tokens:     847 (prompt: 643, completion: 204)
  latency:    2.4s
```

This makes intelligence costs observable without separate cost dashboards.

---

## 20. PROJECT & APPLICATION MANAGEMENT

### V1: Single Project, Single Namespace

V1's project model is simple: one `awis init` per project, one namespace, one `runtime.db`. The project is the directory where `awis init` was run.

```
my-project/
├── .awis/
│   ├── config.yaml       # runtime configuration (git-ignored)
│   ├── runtime.db        # SQLite event log + state (git-ignored)
│   └── .gitignore
├── workflows/
│   ├── capture-decision.yaml
│   └── recall-decision.yaml
└── (application code)
```

The namespace is declared in `config.yaml`:
```yaml
namespace: oip
```

### V1: Application Registration

Applications register workflows at startup:

```go
func main() {
    runtime, err := awis.NewRuntime(awis.Config{
        Namespace: "oip",
        // Storage and intelligence from .awis/config.yaml
    })
    runtime.RegisterHandler(&oip.RecordAppendHandler{})
    runtime.RegisterWorkflow(oip.CaptureDecisionWorkflow())
    runtime.Start(context.Background())
}
```

### V2: Multiple Applications, Multiple Namespaces

V2 introduces multi-namespace support for running multiple applications on one runtime:

```
$ awis namespace create neurodashboard
$ awis namespace list

NAME             STATUS   WORKFLOWS  ACTIVE_INSTANCES
oip              active   2          1
neurodashboard   active   3          0

$ awis status --namespace=oip
$ awis status --namespace=neurodashboard
$ awis status --all
```

---

## 21. ERROR HANDLING UX

### Error Design Principles

Every error AWIS shows must answer three questions:
1. **What happened?** (specific, not generic)
2. **Where?** (instance ID, step ID, plugin name, config key)
3. **What now?** (a concrete next command or action)

### Error Taxonomy

**User errors (configuration, definition):**
```
awis: workflow validation failed: capture-decision.yaml
  Line 45: step 'confirm-entry' declares fallback 'manual-entry'
           but 'manual-entry' is not defined in this workflow

  Suggestion: Add a step with id 'manual-entry', or remove the fallback declaration.
```

**Execution errors (step failures):**
```
awis: step 'draft-entry' failed after 3 attempts
  Instance:  i-m4n5o6
  Workflow:  capture-decision v1.0.0
  Error:     connection refused (anthropic API)
  Status:    workflow moved to 'failed' state; compensation running

  Diagnose:  awis trace i-m4n5o6
  Check API: awis config show | grep intelligence
```

**Infrastructure errors (storage, plugin):**
```
awis: plugin 'git-context-plugin' failed to spawn (3/3 attempts)
  Error:    python3: ModuleNotFoundError: No module named 'gitpython'
  Impact:   Steps requiring 'git.context.assemble' will fail

  Fix:      pip install gitpython
            awis plugin status git-context-plugin
```

**Transient errors (handled silently):**
Intelligence timeouts triggering retry, step retry within policy limits, plugin restart within limits — these are NOT shown as errors. They appear only in `awis trace` as part of the execution timeline. Silent recovery is a feature, not a hidden failure.

### Signal Timeout UX

```
$ awis status
 ✗ failed   capture-decision  i-d4e5f6  signal timeout: confirm-entry  3d ago

$ awis trace i-d4e5f6
...
  72:00  ✗ WaitTimeout  confirm-entry  72h elapsed; no signal received
                         timeout_action: fail
  72:00  ● WorkflowFailed  reason: confirm-entry timed out (72h)
                            To retry: awis submit capture-decision
```

---

## 22. EMPTY STATES

Every empty state communicates: what this space holds, why it's empty, and what the next action is.

### `awis status` — No Active Instances

```
AWIS ● running  namespace: oip

No active workflows.

Submit a workflow:    awis submit <workflow-id>
List workflows:       awis workflow list
View history:         awis history
```

### `awis workflow list` — No Workflows Registered

```
No workflows registered.

Register a workflow:
  Option 1 (YAML):    Create workflows/my-workflow.yaml, then restart AWIS
  Option 2 (Go SDK):  runtime.RegisterWorkflow(myWorkflow) in your application
  
Documentation:        awis workflow validate --example
```

### `awis plugin list` — No Plugins Installed

```
No plugins installed.

Install a plugin:
  awis plugin install <path-or-url>

Available plugins:    See awis.dev/plugins (community registry, V2)
Build a plugin:       awis --help plugins
```

### `awis history` — No Completed Workflows

```
No completed workflow instances yet.

Run your first workflow:    awis submit <workflow-id>
List registered workflows:  awis workflow list
```

### `awis recall "..."` — No Matching History

```
No matching workflow history found for: "how were database migrations handled?"

This query requires:
  1. Workflow instances that completed in the oip namespace
  2. Intelligence configured for synthesis (current: null adapter)

Configure intelligence:  awis config set intelligence.primary.provider anthropic
View raw history:        awis history
```

---

## 23. ONBOARDING EXPERIENCE

### The Three-Phase Onboarding

**Phase 1: Zero-to-Running (5 minutes)**

Goal: developer submits their first workflow and sees the trace before they've read any documentation.

```
$ awis init
...creates example workflow...

$ awis start
...auto-registers example workflow...

$ awis submit hello-world
Submitted: hello-world / i-abc123

$ awis trace i-abc123
00:00  ● WorkflowStarted
00:00  ► StepStarted     step-one
00:00  ✓ StepCompleted   step-one   0.1s   outputs: {message: "hello"}
00:00  ► StepStarted     step-two
00:00  ✓ StepCompleted   step-two   0.0s   outputs: {message: "world"}
00:00  ● WorkflowCompleted   0.1s
```

**Phase 2: First Real Workflow (30 minutes)**

Goal: developer defines their own workflow, registers a step handler, and handles a failure.

The starter example created by `awis init` includes:
- `workflows/hello-world.yaml` — the minimal "it works" workflow
- `workflows/example-with-signal.yaml` — demonstrates WAIT steps
- `handlers/example_handler.go` — a commented Go StepHandler example
- `README_AWIS.md` — 3-page quick reference (not a full manual)

**Phase 3: Intelligence Introduction (15 minutes)**

Goal: developer adds an intelligence step, observes NullAdapter fallback, then adds an API key and observes real intelligence output.

The example workflow deliberately includes an intelligence step with a visible fallback so the developer experiences the fallback path before adding an API key. The "aha moment" is comparing the trace before and after the API key: the same workflow, the same YAML, a completely different draft-entry output.

### What `awis init` Creates

```
.awis/
  config.yaml          # Defaults: namespace=awis, intelligence=null
  .gitignore           # Ignores runtime.db, *.db-wal, config.yaml

workflows/
  hello-world.yaml     # Two native steps, no external dependencies
  with-signal.yaml     # Demonstrates WAIT + signal delivery
  with-intelligence.yaml  # Demonstrates intelligence step + fallback

handlers/
  example_handler.go   # Minimal StepHandler implementation with comments

README_AWIS.md         # What to do next (3 pages max)
```

### First-Run Printing

On first `awis start` after `awis init`:

```
AWIS v1.0.0 — first run
══════════════════════════════════════════════════

Welcome to AWIS. Here's what was set up:

  Namespace:       awis (change: config.yaml → namespace)
  Storage:         .awis/runtime.db (SQLite)
  Intelligence:    null (add API key to enable)
  Workflows (3):   hello-world, with-signal, with-intelligence

Try these commands:

  awis submit hello-world            # Run the example workflow
  awis trace <id>                    # Inspect what happened
  awis submit with-signal            # See a workflow that waits
  awis signal <id> my-signal         # Unblock the waiting workflow

Add intelligence:

  export ANTHROPIC_API_KEY=<key>
  awis start                         # Intelligence activates automatically

Documentation:
  README_AWIS.md    Quick reference
  awis --help       Command reference

══════════════════════════════════════════════════
Runtime ready. Press Ctrl+C to stop.
```

---

## 24. SETTINGS PHILOSOPHY

### Minimal Configuration Mandate

AWIS's configuration surface must be as small as possible. The principle: **every configuration option that exists must have been requested by a real user facing a real problem.** Speculative configuration is deleted.

V1 configuration file (`config.yaml`):

```yaml
# AWIS Configuration
# All values are optional — defaults are production-ready

namespace: awis            # Workflow namespace (required; change to your app name)

intelligence:
  primary:
    provider: null         # Change to: anthropic | openai | ollama
    # api_key read from: ANTHROPIC_API_KEY | OPENAI_API_KEY

runtime:
  poll_interval: 100ms     # Execution loop cadence (increase for lower CPU)
  max_parallel: 4          # Max concurrent step executions

logging:
  level: error             # error | info | debug
```

Eight options. That is the entire V1 configuration surface. Users who need more get it in V2 when their use case proves the need.

### Configuration Hierarchy

Environment variables override `config.yaml`; command flags override both.

```
config.yaml (base)
  ↓ overridden by
environment variables (ANTHROPIC_API_KEY, AWIS_NAMESPACE, etc.)
  ↓ overridden by
command-line flags (--namespace, --config, etc.)
```

### Settings That Don't Exist in V1

- Per-step timeout overrides (use workflow-level timeout)
- Intelligence cost limits (V2)
- Plugin resource limits (V2)
- Namespace quotas (V2)
- Authentication configuration (V3)
- Multi-adapter routing policy (V2)
- Metrics export endpoints (V2)

---

## 25. NOTIFICATION MODEL

### V1: Terminal-Native Notifications

AWIS V1 produces no push notifications. All information is available on demand via CLI commands. The developer's workflow is: run `awis status` to see what's happening; run `awis trace` when something needs inspection.

**`--watch` mode** for continuous monitoring:

```
$ awis status --watch
(auto-refreshes every 5 seconds; Ctrl+C to exit)
```

**Long-running workflow alerts:**

When a developer submits a workflow with `--wait`, the command blocks until completion:

```
$ awis submit capture-decision --wait --timeout=10m
Waiting for i-a1b2c3...
  ●  assemble-context: running (2s)
  ✓  assemble-context: completed (1.2s)
  ●  draft-entry: running...
  ✓  draft-entry: completed (2.4s)
  ○  confirm-entry: waiting for signal 'entry_confirmed' (72h timeout)

(Blocking. Deliver signal with: awis signal i-a1b2c3 entry_confirmed)
```

### V2: Webhook Notifications

V2 adds configurable webhooks for workflow completion/failure events:

```yaml
notifications:
  webhooks:
    - url: https://myapp.internal/awis-webhook
      events: [workflow.completed, workflow.failed]
      namespace: oip
```

### V3: Rich Notification System

V3 adds: email notifications for long-wait timeouts, Slack integration, in-browser alerts in the web dashboard.

---

## 26. ACCESSIBILITY STRATEGY

### CLI Accessibility (V1)

The CLI is the entire product in V1. Accessibility means:

1. **No color-only information.** Every piece of information conveyed by color is also conveyed by symbol (✓, ✗, ●, ○) and by text. Developers using terminals without color support (or with screen readers) see complete information.

2. **Machine-parseable output.** Every command supports `--json` output. Assistive tools can parse structured JSON; they cannot parse formatted terminal tables.

3. **Predictable output structure.** Command output follows consistent patterns. Screen-reading workflows are learnable because the same information always appears in the same position.

4. **No time-based UI.** No spinners that require real-time terminal updates. Progress is shown as text updates, not animation. `--watch` mode refreshes the full output, not in-place updates.

5. **Keyboard-only completable.** Every AWIS operation can be accomplished via keyboard. No mouse interaction required.

### Web Accessibility (V3)

V3's web dashboard targets WCAG 2.1 Level AA:
- Full keyboard navigation with visible focus indicators
- Screen reader announcements for workflow status changes
- No information conveyed by color alone
- Minimum contrast ratios on all text
- No time-limited interactions

---

## 27. VERSION ROADMAP (V1 → V3)

### V1 — The Foundation (Weeks 1–8)

**Theme:** "Build your first workflow. Trust it completely."

**What ships:**
- CLI: init, start/stop, submit, signal, status, trace, history, logs, metrics, workflow validate/list/show, plugin install/list/status, config show/set
- Go SDK: WorkflowBuilder, StepHandler, WorkflowTestHarness, WorkflowRunner
- YAML DSL: full parser and validator
- Storage: SQLite EventLog + StateStore + WorkflowRegistry
- Runners: NativeRunner (Go), SubprocessRunner (Python/shell)
- Intelligence: Null adapter (default) + Anthropic adapter
- Plugins: subprocess JSON-RPC protocol + manifest system
- Reference plugin: git-context-plugin
- OIP built on AWIS (validates the platform boundary)

**What does NOT ship:**
- Web dashboard
- Multi-namespace
- OpenAI / Ollama adapters
- Server mode
- Postgres adapter
- Plugin registry / marketplace
- Multi-worker execution
- Cost tracking

**V1 Quality Gates:**
- Time from `awis init` to first completed workflow trace: ≤ 5 minutes
- `awis trace` output parseable without reading documentation
- All workflows run with NullAdapter (zero-AI test)
- OIP's capture-decision and recall-decision workflows run without platform surgery

---

### V2 — Multi-Application (Months 3–6)

**Theme:** "Run all your applications on one platform."

**What ships:**
- Multi-namespace isolation with quotas
- Server mode (HTTP API, localhost)
- Postgres storage adapter
- OpenAI adapter + Ollama adapter
- Multi-provider routing with fallback chains
- Plugin registry (local, then community-contributed)
- Cost tracking (per-namespace, per-adapter)
- Metrics export (Prometheus-compatible)
- `awis recall` with full intelligence synthesis
- Second + third applications validated (NeuroDashboard or Shade Ledger)
- Webhook triggers
- `awis namespace` commands
- `awis status --all` for cross-namespace view

**V2 Quality Gates:**
- Two applications in separate namespaces, zero data bleed
- Intelligence cost per workflow instance visible in trace
- Plugin from community registry installs in ≤ 2 minutes

---

### V3 — Platform (Months 7–18)

**Theme:** "Onboard new applications in minutes. Govern machine authority explicitly."

**What ships:**
- Web dashboard (status, trace, history, workflow graph visualization)
- Multi-tenant workspaces with authentication
- Visual workflow designer (read workflow definitions; edit in V3.1)
- Charter model (bounded machine authority delegation)
- WASM sandboxing for untrusted plugins
- Adaptive intelligence routing (performance-based)
- Cross-namespace governed data grants
- Plugin marketplace
- V3 charter-aware `awis signal` (signals require authorization level check)
- `awis audit` with searchable audit log

**V3 Quality Gates:**
- New application onboarded without platform code change
- Charter-bounded machine action taken and auditable
- Cross-namespace data access requires and shows explicit grant

---

## 28. SUCCESS METRICS

### Developer Experience Metrics (Quantitative)

| Metric | V1 Target | Measurement |
|---|---|---|
| Time to first workflow (init → trace) | ≤ 5 minutes | Measured on clean machine |
| Time to first application on AWIS | ≤ 4 hours | OIP integration time |
| Time to diagnose a failure via `awis trace` | ≤ 2 minutes | User study timing |
| Workflow definition validation error clarity | ≥ 90% clear | User comprehension test |
| Zero-AI test pass rate | 100% | CI gate |
| WorkflowTestHarness adoption | ≥ 80% of workflows have tests | Code coverage metric |

### Platform Health Metrics (Quantitative)

| Metric | V1 Target | Measurement |
|---|---|---|
| Workflow completion rate (local) | ≥ 98% | EventLog |
| Step execution latency P95 (native, no AI) | ≤ 50ms | EventLog |
| Step execution latency P95 (intelligence) | ≤ 5s | EventLog |
| Intelligence fallback rate (when configured) | ≤ 5% | EventLog |
| Plugin crash rate | ≤ 1% of calls | EventLog |
| EventLog rebuild time (100K events) | ≤ 30s | Benchmark |

### Product Validation (Qualitative)

| Hypothesis | Validation Method |
|---|---|
| Developers trust AWIS for production workflows | OIP E1 experiment passes (≥5 entries/week) |
| Platform boundary is correct | OIP built without platform surgery |
| Debugging is genuinely faster than alternatives | User comparison: AWIS trace vs. custom logging |
| Intelligence optional architecture holds | Zero-AI test in CI never fails |
| Plugin model is learnable | Plugin developer creates plugin in < 4 hours |

---

## 29. NON-GOALS

### Explicit Non-Goals for V1

1. **Visual workflow designer.** Building a visual designer before the data model is stable creates tech debt, not value. V3 is the right time.

2. **Real-time collaboration.** Multiple developers editing the same workflow simultaneously is a V3 concern. V1 serves a solo founder; Git provides collaboration.

3. **AI marketplace or model selection UI.** Model selection is a configuration concern, not a product concern. The developer sets `draft_model: claude-haiku-4-5-20251001`; AWIS uses it.

4. **Built-in authentication.** V1 runs locally with no network exposure. Authentication is a V3 concern when multi-tenant workspaces require it.

5. **Native mobile app.** AWIS is a developer infrastructure tool. The terminal is the right interface for V1.

6. **Prebuilt workflow templates marketplace.** Templates are a discovery feature for a community that doesn't yet exist. Ship after the platform has users.

7. **Automatic workflow optimization.** The adaptive routing in V3 depends on accumulated performance history. V1 collects the data; V3 acts on it.

8. **Enterprise governance features.** Compliance reporting, role-based access control, and SOC 2 controls are V3+ concerns. Building them in V1 consumes engineering time that should go to the core experience.

### Permanent Non-Goals (regardless of version)

1. **Owning business logic.** AWIS owns execution; applications own logic. This boundary never moves.

2. **Replacing application databases.** AWIS's EventLog records execution history, not application data. OIP's `.decisions/` folder is OIP's, not AWIS's.

3. **Locking in intelligence providers.** IntelligencePort is a permanent abstraction. No provider-specific code ever enters the runtime.

4. **Requiring cloud connectivity.** Local-first is a permanent commitment. An internet-unreachable laptop running AWIS is a fully supported deployment.

---

## 30. PRODUCT RISKS

| ID | Risk | Probability | Impact | Mitigation |
|---|---|---|---|---|
| PR-1 | Platform boundary confusion — developers don't know what AWIS owns vs. their application | High | High | Clear "What AWIS Owns" documentation; `awis init` example shows boundary explicitly |
| PR-2 | `awis trace` output too raw — developers can't read it without documentation | Medium | High | User-test the trace output format early; clear column headers; progressive disclosure (summary first, `--full` for details) |
| PR-3 | NullAdapter fallback is invisible — developers don't know intelligence isn't working | Medium | Medium | `awis start` header always shows current intelligence level; trace shows "null adapter" explicitly |
| PR-4 | First-run failure — `awis init` or `awis start` fails on developer's machine | Medium | High | Test on clean machines (macOS, Linux); clear dependency requirements; single binary with no runtime deps |
| PR-5 | Plugin installation complexity — Python dependencies break on install | Medium | Medium | Isolated plugin environments (venv per plugin); dependency pinning in manifest |
| PR-6 | YAML DSL too limiting — developers hit its ceiling quickly | Medium | Low | Go SDK is always available; YAML ceiling is a known design; document the ceiling honestly |
| PR-7 | OIP integration requires platform surgery — SDK boundary is wrong | Low | High | OIP integration is the V1 gate; if surgery required, fix platform before second application |
| PR-8 | Debugging too slow — time from failure to diagnosis exceeds 10 minutes | Low | High | `awis trace` must be the first tool developers reach for; test with real failures |
| PR-9 | EventLog grows unbounded — developers run out of disk space | Low | Medium | Default 90-day retention; `awis prune-events --dry-run` before any deletion; V2 configurable retention |
| PR-10 | Solo-founder abandonment — platform unmaintained if founder exits | Medium | High | Minimal core; no clever abstractions; clean Go; ADRs for all decisions; clear architecture enables community pickup |

---

## 31. FUTURE EXPANSION STRATEGY

### Expansion by Composition, Not Complexity

AWIS's expansion strategy follows FP-8 (minimal core, open edge): new capabilities land at the edge (new plugins, new adapters, new SDK surfaces) without enlarging the core. The expansion test: "does this new capability require changes to the runtime, the EventLog schema, or the StoragePort interface?" If yes, it requires architectural-level justification. If no, it is an edge extension and can ship without risk.

### The Plugin Ecosystem as Growth Vector

The plugin system is AWIS's primary external growth mechanism. Every new external capability (GitHub integration, Jira integration, database connectors, hardware interfaces, domain-specific APIs) can be a plugin written by anyone in any language. The platform grows without the platform growing.

**Plugin ecosystem strategy:**
- V1: Ship one reference plugin (git-context-plugin) that demonstrates the protocol
- V2: Ship a community plugin registry; accept PRs for curated plugins
- V3: Plugin marketplace with install counts, ratings, verified publishers

### The Intelligence Adapter Registry

As new AI providers emerge, new adapters can be contributed as separate packages without touching the core:

```go
import "github.com/awis-adapters/mistral"
runtime := awis.NewRuntime(awis.Config{
    Intelligence: awis.IntelligenceConfig{
        Adapters: []IntelligencePort{
            mistral.NewAdapter(os.Getenv("MISTRAL_API_KEY")),
            awis.NewNullAdapter(),
        },
    },
})
```

No core changes. The adapter ecosystem evolves independently.

### Applications as the Platform's Proof

Every application built on AWIS is a product proof-of-concept that extends the platform's argument: "You could build this without AWIS, but why would you?" The more applications in the portfolio, the more the compounding advantage materializes.

The expansion roadmap for applications:
- V1: OIP (decision memory)
- V2: NeuroDashboard (health analytics) + Shade Ledger (financial ledger)
- V3: Job Application Automation + Industrial SaaS
- V4+: External developers building on the platform

### The Path to External Platform

When the platform serves external developers (beyond the solo-founder's own portfolio), the product shifts:

1. The plugin marketplace becomes a discovery and monetization surface
2. The SDK becomes a public API with versioning guarantees and deprecation cycles
3. The web dashboard becomes the primary onboarding surface for non-CLI developers
4. The charter model becomes the trust and governance foundation for shared workspaces

This is a genuine platform shift — from "infrastructure for my applications" to "infrastructure for everyone's applications." The current architecture supports this shift; the current product does not need to prematurely target it.

---

## CANONICAL PRODUCT DEFINITION

### What AWIS Is

AWIS is a **locally-hosted workflow execution platform** for developers building software products. It provides a shared runtime — step execution, durable state, failure recovery, AI integration, and execution history — that multiple applications can build on without reimplementing infrastructure.

### Who AWIS Serves

AWIS serves developers, not end users. Specifically:
- **Platform Builders** who set up and maintain the runtime for a portfolio of applications
- **Application Developers** who define workflows, write step handlers, and debug execution failures
- **Plugin Developers** who extend the platform with external capabilities in their preferred language

End users interact with applications built on AWIS; they do not interact with AWIS directly.

### The Problems AWIS Solves

**Problem 1: Repeated infrastructure.** Every non-trivial software product needs step execution, retry logic, state persistence, AI integration, and failure history. Without a platform, developers rebuild this in every product — slightly different each time, all maintained in parallel. AWIS builds it once.

**Problem 2: Opaque failures.** Debugging async workflows in custom implementations means reading scattered logs, reconstructing state mentally, and guessing at causation. AWIS's EventLog and `awis trace` replace this with a complete, chronological, step-by-step timeline of every execution.

**Problem 3: Fragile AI integration.** AI providers change APIs, hit rate limits, and have outages. Applications with hardcoded AI integration break when their provider does. AWIS's IntelligencePort puts every provider behind a single swappable interface; changing providers is a config key.

**Problem 4: Untestable workflows.** Async workflow logic is hard to test deterministically. Unit tests mock one step at a time; integration tests require live infrastructure. AWIS's WorkflowTestHarness runs complete workflows synchronously in tests with zero external dependencies.

### What Makes AWIS Fundamentally Different

Three things distinguish AWIS from existing workflow tools:

1. **Zero infrastructure locally.** No message broker, no coordinator, no Docker, no cloud account. `awis start` is one binary on one laptop. This is not a "lite mode" — it is the full platform. Cloud is a storage upgrade, not a prerequisite.

2. **Complete execution history as first-class product.** The EventLog is not a log — it is the source of truth. Every execution is always inspectable, always reproducible, always citeable. `awis trace` answers questions that custom logging cannot. This turns execution history into organizational operational memory.

3. **Intelligence that degrades gracefully at every level.** Every workflow runs with NullAdapter (no AI), with Ollama (local AI), or with cloud providers (Anthropic/OpenAI), using identical workflow definitions. Intelligence is a "power level" that the developer turns up by adding a config key — not an architectural commitment they make at design time.

### Why the Architecture Naturally Supports the Product Vision

The AWIS architecture was designed with one product truth in its foundation: **the runtime is invisible; the experience is everything.** This manifests in three architectural choices that directly produce the product experience:

1. **The EventLog as source of truth** directly produces the `awis trace` experience. Because every state change is an event, every execution is fully introspectable. The product's best debugging feature is a free consequence of the right storage architecture.

2. **IntelligencePort with NullAdapter as default** directly produces the "intelligence as power level" experience. Because the NullAdapter is always registered, workflows always run. The developer's first-run experience never fails due to missing API keys. Adding intelligence is a one-line config change with zero workflow definition changes.

3. **Pull-based execution with a single binary** directly produces the "zero infrastructure" experience. Because the runtime is a pull-based loop in a single process, there is no coordinator service, no message broker, no additional process. `awis start` is the entire infrastructure.

The architecture was built to support exactly this product. The product is the architecture's face.

---

*Produced by AWIS Product Strategy Tribunal v2.0 — 2026-07-02*
*Authority: AWIS_ARCHITECTURE_BLUEPRINT.md | Status: Pre-PRD investigation complete*
*Next step: Generate AWIS_PRD.md from this investigation*
