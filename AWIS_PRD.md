# AWIS CANONICAL PRODUCT REQUIREMENTS DOCUMENT
## Workflow Intelligence Platform — v1.0

**Document Status:** Canonical — Implementation-Ready
**PRD Version:** 1.0
**Date:** 2026-07-02
**Authority:** AWIS_ARCHITECTURE_BLUEPRINT.md + AWIS_ARCHITECTURE_FINALIZATION.md (all 7 blockers resolved)
**Produced by:** AWIS Product Requirements Process — Post-Architecture Freeze
**Cross-reference:** AWIS_ARCHITECTURE_BLUEPRINT.md (33 sections, 15 ADRs), AWIS_ARCHITECTURE_FINALIZATION.md (7 blocker resolutions)

> **"This document is the Canonical Product Requirements Document for AWIS v1.0 and shall serve as the primary product specification for implementation."**

---

## TABLE OF CONTENTS

1. Executive Summary
2. Product Vision
3. Product Definition
4. Goals
5. Non-Goals
6. Target Users & Personas
7. User Problems
8. Value Proposition
9. Product Principles
10. Functional Requirements
11. Non-Functional Requirements
12. User Stories
13. User Flows
14. Information Architecture
15. Navigation Structure
16. Screen Inventory
17. Dashboard Requirements
18. Workflow Builder Requirements
19. Workflow Execution Experience
20. Workflow Monitoring Experience
21. Intelligence Layer UX
22. Plugin Management
23. Project & Workspace Management
24. AI Provider Configuration
25. Local-First Behavior
26. Error Handling
27. Permissions & Security
28. Accessibility
29. Notifications
30. Settings
31. Success Metrics
32. Acceptance Criteria
33. Risks & Constraints
34. V1 Scope
35. Future Roadmap (V2/V3)
36. Glossary
37. Document Metadata

---

## 1. EXECUTIVE SUMMARY

AWIS is a **locally-hosted, AI-optional workflow execution platform** for developers building software products. It is not an end-user product — it is the shared engineering foundation that eliminates repeated infrastructure work across a portfolio of independent software applications.

The platform delivers three things and nothing more:

1. **Durable execution substrate** — run steps, persist state, recover from failure, retry with policy, compensate on failure.
2. **Intelligence abstraction layer** — optional AI enhancement behind a provider seam; removing it leaves a fully functional system.
3. **Application SDK** — the minimum surface applications need to define workflows and query execution history without reimplementing infrastructure.

**V1 in one sentence:** A CLI-native, local-first workflow runtime where developers define step graphs in YAML or Go, run them on SQLite with optional Anthropic intelligence, and debug them with `awis trace` — all from a single binary with no external services.

**The product's competitive moat** is not features — it is the combination of zero infrastructure to operate locally, complete execution history enabling genuine debugging, and an intelligence abstraction that makes AI optional rather than load-bearing. No existing workflow tool offers all three simultaneously.

**V1 is validated** when OIP (Organizational Intelligence Platform) — the first application built on AWIS — can run its two core workflows (`capture-decision`, `recall-decision`) without requiring any modification to the AWIS platform itself.

---

## 2. PRODUCT VISION

### V1 Vision (Weeks 1–8)

> "A developer can define a multi-step workflow with a human confirmation gate, run it locally, watch it execute step by step, debug failures in a single command, and add AI drafting with one environment variable — all in under 30 minutes from zero."

This vision excludes: cloud deployment, multiple namespaces, visual design, team collaboration, cost tracking, and any feature requiring more than one developer. Mastery before scale.

### V2 Vision (Months 3–6)

> "A small team can run AWIS as a shared server, build multiple applications in isolated namespaces, use different intelligence providers per application, install plugins from a registry, and deploy to a cloud environment without changing their workflow definitions."

### V3 Vision (Months 7–18)

> "A growing organization can onboard new applications in minutes, visualize workflow graphs in a browser, define charter-bounded machine actors, and let AWIS's adaptive routing optimize intelligence calls based on observed performance — while retaining complete execution history and exit rights."

### The Decade Vision

By 2031, AWIS is the substrate layer for a portfolio of software products, the way PostgreSQL is the substrate for web applications. It is not the product end-users see; it is the platform that makes every product the developer builds more reliable, more observable, and more intelligent than it would be without it.

---

## 3. PRODUCT DEFINITION

### What AWIS Is

AWIS is a locally-hosted workflow execution platform with an optional intelligence layer. Developers use it as a shared runtime foundation for multiple software products, eliminating the need to rebuild step execution, failure recovery, state persistence, AI integration, and execution history in every application.

A developer building OIP, NeuroDashboard, Shade Ledger, and Job Application Automation on AWIS implements workflow infrastructure once and ships four applications on it. Without AWIS, they implement four private, incompatible versions of the same infrastructure — and maintain all four indefinitely.

**The product in one sentence:** AWIS is the workflow engine developers never want to rebuild, made local, AI-optional, and fully observable.

### What AWIS Is Not

| Not This | Because |
|---|---|
| A no-code automation tool (Zapier/Make) | AWIS requires developers; business users are not its audience |
| A visual workflow builder | V1 is CLI-first; visual is a V3 layer on a stable data model |
| A managed cloud service | V1 runs on a single laptop with zero external dependencies |
| An AI-first product | Intelligence is optional infrastructure; NullAdapter is the default |
| An application framework | AWIS owns execution; applications own business logic |
| A monitoring dashboard | Observability is terminal-native in V1; web dashboard is V3 |
| A BPM/BPMN system | BPMN is enterprise-workflow theater; AWIS is developer workflow reality |
| A data pipeline tool | Airflow/Dagster are data-centric; AWIS is application-workflow-centric |
| A process automation platform | UiPath targets business operations; AWIS targets software applications |

### Platform Applications (V1–V3 Horizon)

The AWIS architecture must support the following applications without modification between them:

- **OIP** (organizational decision memory) — the first application; validates the platform
- **NeuroDashboard** (health analytics and visualization)
- **Shade Ledger** (domain-specific financial ledger)
- **Job Application Automation** (multi-step human-in-the-loop workflows)
- **Industrial SaaS products** (complex process workflows, scheduling, monitoring)
- **Internal AI systems** (intelligence-enhanced operational workflows)

---

## 4. GOALS

### Product Goals

**G-1 (Platform Foundation):** Provide a reusable workflow runtime that eliminates repeated infrastructure implementation across the application portfolio.

**G-2 (Developer Trust):** Make every workflow execution fully inspectable — any failure diagnosable in under 2 minutes using `awis trace`.

**G-3 (AI Optionality):** Ensure every workflow runs identically whether intelligence is configured or not. NullAdapter is always the last fallback.

**G-4 (Local-First):** Deliver the complete platform experience from a single binary on a single laptop with no external services required.

**G-5 (Platform Validation):** Prove the platform boundary is correct by building OIP as the first AWIS application without requiring platform surgery.

**G-6 (Developer Velocity):** Reduce time from `awis init` to first successful workflow trace to ≤ 5 minutes.

**G-7 (Test Confidence):** Enable deterministic workflow testing without a live runtime, external API keys, or a running database.

### V1 Success Criteria

| Goal | Metric | Target |
|---|---|---|
| G-2 | Time from failure to diagnosis via `awis trace` | ≤ 2 minutes |
| G-4 | First workflow with zero external services | ✓ on clean machine |
| G-5 | OIP workflows run without platform surgery | ✓ confirmed |
| G-6 | `awis init` → first trace | ≤ 5 minutes |
| G-7 | All workflow tests pass without API keys | ✓ CI gate |

---

## 5. NON-GOALS

### V1 Non-Goals

**NG-1 — Visual workflow designer.** Building before the data model is stable creates tech debt. V3 is the right time.

**NG-2 — Real-time team collaboration.** Multiple developers editing the same workflow simultaneously is V3. Git provides collaboration in V1.

**NG-3 — AI model selection UI.** Model selection is configuration (`draft_model: claude-haiku-4-5-20251001`), not a product feature.

**NG-4 — Built-in authentication.** V1 runs locally with no network exposure. Authentication is V3.

**NG-5 — Native mobile app.** AWIS is a developer infrastructure tool. Terminal is the correct interface for V1.

**NG-6 — Prebuilt workflow template marketplace.** Templates are a discovery feature for a community that doesn't yet exist.

**NG-7 — Automatic workflow optimization.** Adaptive routing (V3) depends on accumulated history. V1 collects the data; V3 acts on it.

**NG-8 — Enterprise governance.** Compliance reporting, RBAC, SOC 2 are V3+ concerns.

**NG-9 — Multi-worker execution.** Cloud multi-worker mode is V2.

**NG-10 — Postgres storage.** Cloud storage upgrade is V2.

### Permanent Non-Goals (All Versions)

**PNG-1 — Owning business logic.** AWIS owns execution; applications own logic. This boundary never moves.

**PNG-2 — Replacing application databases.** AWIS's EventLog records execution history, not application data. OIP's `.decisions/` folder is OIP's, not AWIS's.

**PNG-3 — Locking in intelligence providers.** IntelligencePort is a permanent abstraction. No provider-specific code ever enters the runtime.

**PNG-4 — Requiring cloud connectivity.** Local-first is a permanent commitment. An internet-unreachable laptop running AWIS is a fully supported deployment.

---

## 6. TARGET USERS & PERSONAS

AWIS serves developers, not end users. Three distinct developer personas interact with the platform.

### Persona 1: The Platform Builder

**Who:** A solo founder or lead developer who sets up and maintains the AWIS runtime for a product portfolio. In a solo-founder context, this is the same person as Persona 2, wearing a different hat.

**Goals:**
- Initialize AWIS once; have it serve all applications without repeated setup
- Configure intelligence providers and plugins once at the platform level
- Monitor overall runtime health without wading through application-level detail
- Recover from failures without data loss

**Primary CLI interactions:**
`awis init`, `awis start/stop`, `awis config`, `awis plugin install`, `awis rebuild-state`, `awis metrics`

**Mental model:** "AWIS is my platform's operating environment. I set it up; my applications run on it."

---

### Persona 2: The Application Developer

**Who:** A developer building a specific application on AWIS (OIP, NeuroDashboard, etc.). Defines workflows, writes step handlers, and ships features. Interacts with AWIS primarily through the SDK and CLI debugging tools.

**Goals:**
- Define workflows without reimplementing scheduling, retry, or state management
- Debug failed workflow instances quickly and precisely
- Test workflow behavior deterministically without a live runtime
- Add intelligence steps without deep AI integration knowledge

**Primary interactions:**
- Go SDK (WorkflowBuilder, StepHandler, WorkflowTestHarness) — daily development
- YAML workflow files — workflow definition for simple workflows
- `awis workflow validate` — catch errors before deployment
- `awis submit`, `awis trace`, `awis signal` — the core development loop

**Mental model:** "I define what my workflow does; AWIS handles how it runs."

---

### Persona 3: The Plugin Developer

**Who:** A developer (often not Go-proficient) who extends AWIS with a new external capability. May be the same person as Persona 2 for one-off integrations.

**Goals:**
- Expose a capability to AWIS without learning runtime internals
- Write in their preferred language (Python, shell, TypeScript)
- Test their plugin independently of a running AWIS runtime
- Declare capabilities clearly so applications can depend on them

**Primary interactions:**
- `awis-plugin.yaml` — capability manifest
- stdin/stdout JSON-RPC protocol — the only runtime interface
- `awis plugin install/status/remove` — lifecycle management
- Test harness — simulate JSON-RPC protocol without runtime

**Mental model:** "I announce what I can do; AWIS calls me when needed."

---

### Persona 4: The End User (Indirect)

**Who:** The person who uses an application built on AWIS — someone who captures decisions with OIP's CLI, views analytics in NeuroDashboard, etc.

**Relationship to AWIS:** Indirect. The end user interacts with the application; the application runs on AWIS. The end user may never know AWIS exists.

**AWIS product implication:** When AWIS fails visibly, the end user sees an application error. The application developer (Persona 2) must diagnose and fix it quickly using AWIS's tools. The end-user experience is therefore a product quality constraint on the developer experience.

---

## 7. USER PROBLEMS

### Problem 1: Repeated Infrastructure Tax

Every non-trivial software product needs step execution, retry logic, state persistence, AI integration, and failure history. Without a platform, developers rebuild this in every product — slightly different each time, incompatible with each other, all maintained in parallel.

**Cost:** Each private implementation consumes 2–4 weeks of initial development and ongoing maintenance indefinitely. For a five-application portfolio, this is 10–20 weeks of pure infrastructure work that produces zero user value.

---

### Problem 2: Opaque Workflow Failures

Debugging async workflows in custom implementations means reading scattered logs, reconstructing state mentally, and guessing at causation. There is no standard "what happened" view; developers must instrument each workflow differently.

**Cost:** A failure that should take 5 minutes to diagnose can consume 2+ hours. In production, time-to-diagnosis is a direct business cost.

---

### Problem 3: Fragile AI Integration

AI providers change APIs, hit rate limits, and have outages. Applications with hardcoded AI integration break when their provider does. Provider migration requires touching every integration point in the application.

**Cost:** Provider outages cause workflow failures. Provider migrations require cross-application refactoring.

---

### Problem 4: Untestable Async Workflows

Async workflow logic is hard to test deterministically. Unit tests mock one step at a time and can't test workflow-level state transitions. Integration tests require live infrastructure — API keys, running databases, external services.

**Cost:** Low test confidence leads to production surprises. Long feedback loops slow development.

---

## 8. VALUE PROPOSITION

### For Platform Builders

AWIS is the infrastructure tax you pay once. Instead of building step execution, retry, state management, and AI integration in every application, you build them zero times and deploy them everywhere.

**The compound return:** Every hour spent configuring AWIS produces value in every application that runs on it. The fifth application costs less than the first, not more.

### For Application Developers

AWIS replaces three things you would have built anyway — a state machine, a log, and an AI integration layer — with one debuggable, testable, AI-optional runtime.

**The debugging advantage:** `awis trace` answers questions about what happened that custom logging cannot. The EventLog is not a log — it is the chronological, complete, machine-readable record of every state change.

### For Plugin Developers

AWIS's plugin protocol is a single JSON-RPC interface over stdin/stdout. You write in any language, declare your capabilities in a YAML manifest, and AWIS handles the rest. Your plugin is process-isolated, auto-restarted on crash, and callable from any workflow without modification.

### The Differentiated Position

Three things distinguish AWIS from existing workflow tools:

1. **Zero infrastructure locally.** No message broker, no coordinator, no Docker, no cloud account. `awis start` is one binary on one laptop. This is not a "lite mode" — it is the full platform.

2. **Complete execution history as first-class product.** The EventLog is the source of truth. Every execution is always inspectable, always reproducible. `awis trace` turns execution history into organizational operational memory.

3. **Intelligence that degrades gracefully at every level.** Every workflow runs with NullAdapter (no AI), with Ollama (local AI), or with cloud providers, using identical workflow definitions. Intelligence is a "power level," not an architectural commitment.

---

## 9. PRODUCT PRINCIPLES

These ten principles govern all product decisions. Conflicts between features are resolved by these principles, not by feature priority alone.

**PP-1 — The runtime is invisible; the experience is everything.**
Developers don't care about pull-based execution loops or IntelligencePort. They care that their workflow ran, that failures are debuggable, and that adding AI requires only an API key. The runtime's sophistication must manifest as experience simplicity.

**PP-2 — Trust is built in microseconds and lost in minutes.**
A developer who gets a cryptic error on first run will not return. A developer who gets a clear status line, then a precise timeline with adapter names, will build their next application on AWIS. The debugging experience is the product.

**PP-3 — Intelligence is a power level, not a mode switch.**
When intelligence is absent, workflows run and fall back silently. When intelligence is added, the same workflows become smarter. The developer's workflow definition does not change. This is what "AI-optional" means.

**PP-4 — Error messages answer three questions.**
What happened? Where? What now? Any error that does not answer all three is incomplete.

**PP-5 — Empty states teach, not apologize.**
Every empty state communicates what this space holds, why it is empty, and what the next action is.

**PP-6 — Every command output has a `--json` flag.**
Machine-parseable output is not optional. Scripting and CI integration are first-class use cases.

**PP-7 — Configuration surface is minimal by discipline.**
Every configuration option must have been requested by a real user facing a real problem. Speculative configuration is deleted. V1 has eight configuration options.

**PP-8 — Silent recovery is a feature, not a hidden failure.**
Intelligence timeouts triggering retry, step retry within policy limits, plugin restart within limits — these are NOT shown as errors. They appear only in `awis trace`. Recovery should be silent; it is not a failure.

**PP-9 — The compounding effect is the product's business case.**
Every hour spent learning AWIS pays dividends across every application in the portfolio. Design for this compounding — consistent patterns, consistent commands, consistent mental models across all applications.

**PP-10 — Human authority is never eroded by default.**
Machine actors execute steps, but humans configure, authorize, and review. WAIT steps for human confirmation are a first-class primitive. No workflow escalates machine authority without explicit, bounded human delegation.

---

## 10. FUNCTIONAL REQUIREMENTS

Requirements are classified as **Must Have** (V1 required), **Should Have** (V1 target, deferrable), or **Future** (V2/V3).

### 10.1 Runtime Management

| ID | Requirement | Priority |
|---|---|---|
| FR-RM-01 | `awis init` creates `.awis/` directory, default `config.yaml`, `.gitignore`, and three example workflows | Must Have |
| FR-RM-02 | `awis start` starts the runtime in foreground with startup log showing namespace, storage path, intelligence status, and registered workflows | Must Have |
| FR-RM-03 | `awis start` auto-discovers and registers all YAML workflow files in `./workflows/` | Must Have |
| FR-RM-04 | `awis stop` gracefully stops the runtime (in-flight steps complete; no new steps dispatched) | Must Have |
| FR-RM-05 | `awis version` shows version, build timestamp, Go version, and platform | Must Have |
| FR-RM-06 | Runtime exits cleanly on SIGTERM; logs reason; status reflects shutdown | Must Have |
| FR-RM-07 | `awis start --config=<path>` overrides default config location | Should Have |
| FR-RM-08 | `awis start --detach` (daemon mode) | Future (V2) |

### 10.2 Workflow Execution

| ID | Requirement | Priority |
|---|---|---|
| FR-WE-01 | `awis submit <workflow-id>` creates a workflow instance and returns an instance ID immediately | Must Have |
| FR-WE-02 | `awis submit` accepts `--input='<json>'` for workflow inputs | Must Have |
| FR-WE-03 | `awis submit` accepts `--async` to return without waiting; default is async (returns ID immediately) | Must Have |
| FR-WE-04 | `awis submit --wait [--timeout=<duration>]` blocks until completion, streaming step progress | Should Have |
| FR-WE-05 | `awis signal <instance-id> <signal-name> [--payload='<json>']` delivers a signal to a waiting instance | Must Have |
| FR-WE-06 | Signal delivery is atomic across `signal_inbox`, `execution_events`, and `workflow_instances` in a single transaction | Must Have |
| FR-WE-07 | `awis cancel <instance-id> [--reason=<string>] [--compensate]` cancels a running or waiting instance | Must Have |
| FR-WE-08 | Cancellation is graceful: in-flight steps complete; no next step is activated after cancellation_requested is set | Must Have |
| FR-WE-09 | Cancellation is idempotent on terminal instances (logs warning; returns no error) | Must Have |
| FR-WE-10 | `WorkflowCancelled {reason, cancelled_at}` is appended to the EventLog on cancellation | Must Have |
| FR-WE-11 | Pending wait_records are deleted when a workflow is cancelled | Must Have |
| FR-WE-12 | Manual trigger fires immediately on `awis submit` | Must Have |
| FR-WE-13 | Event trigger fires when a matching DomainEvent is received | Must Have |
| FR-WE-14 | Schedule trigger fires on cron schedule | Should Have |
| FR-WE-15 | Webhook trigger fires on incoming HTTP request | Future (V2) |

### 10.3 Workflow Definition

| ID | Requirement | Priority |
|---|---|---|
| FR-WD-01 | YAML DSL parser accepts workflow definitions conforming to the WorkflowDefinition schema | Must Have |
| FR-WD-02 | Go SDK WorkflowBuilder produces identical WorkflowDefinition structs as the YAML DSL | Must Have |
| FR-WD-03 | WorkflowValidator validates: no orphaned steps, no cycles, all handler refs present, all schema refs valid, all conditions parseable per formal grammar | Must Have |
| FR-WD-04 | Validation at registration time: invalid syntax causes registration failure with a precise error | Must Have |
| FR-WD-05 | Template expressions `{{path-ref}}` are resolved at execution time; missing paths resolve to `""` with a logged warning | Must Have |
| FR-WD-06 | Condition expressions (Transition.condition) support: `==`, `!=`, `>`, `<`, `>=`, `<=`, `&&`, `\|\|`, `!`, parentheses, dot-path refs, single-quoted strings, numeric literals, boolean literals, null literal | Must Have |
| FR-WD-07 | Condition expressions: arithmetic, function calls, string concatenation, bracket notation, and ternary operators are prohibited and cause validation failure | Must Have |
| FR-WD-08 | Template expressions: arithmetic, function calls, conditionals, nested templates, and bracket notation are prohibited | Must Have |
| FR-WD-09 | Workflow definitions are immutable once registered; a new version must be registered as a new semver | Must Have |
| FR-WD-10 | Old instances continue on their registered version; new instances use the latest registered version | Must Have |
| FR-WD-11 | Parallel execution via fan-out transitions: multiple transitions with the same `from` step activate all target steps concurrently | Must Have |
| FR-WD-12 | There is no `parallel` StepType in V1; fan-out transitions are sufficient | Must Have |
| FR-WD-13 | Cyclic workflows are not supported in V1; loops must use new instance submission from a step handler | Must Have |
| FR-WD-14 | Compensation handlers are specified only on steps with downstream successors; final step compensation is unnecessary and should not be authored | Must Have |
| FR-WD-15 | `awis workflow validate <file>` validates a YAML file without a running runtime (syntax and schema only; handler existence deferred to registration) | Must Have |
| FR-WD-16 | `awis workflow list [--namespace=<ns>]` lists all registered workflow definitions | Must Have |
| FR-WD-17 | `awis workflow show <id>` shows definition detail: steps, step types, transitions, triggers | Must Have |

### 10.4 Step Execution

| ID | Requirement | Priority |
|---|---|---|
| FR-SE-01 | NativeRunner executes Go StepHandler implementations in-process | Must Have |
| FR-SE-02 | SubprocessRunner executes Python, shell, and TypeScript step handlers via stdin/stdout JSON | Must Have |
| FR-SE-03 | PluginRunner routes steps to registered plugins via JSON-RPC 2.0 over stdin/stdout | Must Have |
| FR-SE-04 | IntelligenceRunner routes intelligence steps to the configured IntelligencePort adapter | Must Have |
| FR-SE-05 | All runners produce the same output: `StepResult{outputs: map, error?: StepError}` | Must Have |
| FR-SE-06 | Every step execution gets a deterministic idempotency key (`instance_id + step_id + attempt_number`) | Must Have |
| FR-SE-07 | Before executing, the runtime checks StepResultCache; if a result exists for the key, it is returned without re-executing | Must Have |
| FR-SE-08 | RetryPolicy specifies: attempts, backoff (immediate/linear/exponential), initial_delay, max_delay, retryable_errors | Must Have |
| FR-SE-09 | After all retry attempts exhausted: if step has a `fallback`, fallback step is activated; if no fallback, workflow fails | Must Have |
| FR-SE-10 | Fallback activation is recorded as `StepFallbackActivated` in the EventLog | Must Have |
| FR-SE-11 | StepType enum: `native \| subprocess \| plugin \| intelligence \| signal` | Must Have |
| FR-SE-12 | Signal (WAIT) steps park the workflow instance in `waiting` state until a named signal is delivered or a timeout expires | Must Have |
| FR-SE-13 | WAIT step timeout action: `fail \| compensate \| continue` | Must Have |
| FR-SE-14 | Execution loop cadence: 100ms locally (configurable via `runtime.poll_interval`) | Must Have |
| FR-SE-15 | Max concurrent step executions: configurable via `runtime.max_parallel` (default: 4) | Must Have |
| FR-SE-16 | `awis rebuild-state [--namespace=<ns>]` replays the EventLog and rebuilds the StateStore | Must Have |

### 10.5 Intelligence Layer

| ID | Requirement | Priority |
|---|---|---|
| FR-IL-01 | NullAdapter: always registered as the last fallback; `IsAvailable()` returns false; provides deterministic fixture responses | Must Have |
| FR-IL-02 | AnthropicAdapter: supports Draft (Haiku/Sonnet), Synthesize (Sonnet), Classify (Haiku) | Must Have |
| FR-IL-03 | OpenAIAdapter: supports Draft, Embed, Synthesize, Classify | Future (V2) |
| FR-IL-04 | OllamaAdapter: supports Draft, Embed, Classify, Synthesize via local Ollama REST API | Future (V2) |
| FR-IL-05 | CapabilityRouter selects adapter based on model_hint (fast/quality/local) and fallback_chain | Must Have |
| FR-IL-06 | If no capable adapter is available and `required: false`: activate step fallback | Must Have |
| FR-IL-07 | If no capable adapter is available and `required: true`: fail step with CapabilityUnavailableError | Must Have |
| FR-IL-08 | Context budget is enforced before dispatching to provider; requests exceeding budget are rejected, not truncated | Must Have |
| FR-IL-09 | Provider selection for each step execution is recorded in the StepCompleted event: `{adapter, model, tokens_used}` | Must Have |
| FR-IL-10 | `classify` is present in IntelligencePort V1 as a non-callable placeholder; NullAdapter implements it; revisit in V2 | Should Have |
| FR-IL-11 | `awis recall "<query>"` returns FTS-first results in V1; `--synthesize` flag enables AI synthesis when intelligence is available | Must Have |

### 10.6 Plugin System

| ID | Requirement | Priority |
|---|---|---|
| FR-PS-01 | Plugins communicate with AWIS runtime via JSON-RPC 2.0 over stdin/stdout | Must Have |
| FR-PS-02 | Plugin manifest format: `awis-plugin.yaml` declares name, version, capabilities (id, inputs, outputs, timeout_ms), and runtime (command, args, env, idle_timeout_s) | Must Have |
| FR-PS-03 | Plugin lifecycle: REGISTER → SPAWN → HANDSHAKE → ACTIVE → IDLE → TERMINATE; auto-restart on crash (max 3 attempts) | Must Have |
| FR-PS-04 | On crash: the runtime auto-restarts the plugin process; step retries against restarted plugin | Must Have |
| FR-PS-05 | Plugin idle state: process is killed after `idle_timeout_s`; respawned on next capability request (expected respawn latency: <2s for Python/shell) | Must Have |
| FR-PS-06 | Plugins have no access to the EventLog, StateStore, or other namespaces; they receive only step inputs and return step outputs | Must Have |
| FR-PS-07 | `awis plugin install <path>` installs a plugin from local path; downloads and installs dependencies per manifest | Must Have |
| FR-PS-08 | `awis plugin list` shows all installed plugins with status (healthy/degraded/failed) | Must Have |
| FR-PS-09 | `awis plugin status <name>` shows plugin health, process info, capability call counts, latency stats, last error | Must Have |
| FR-PS-10 | `awis plugin remove <name>` removes a plugin | Must Have |
| FR-PS-11 | Plugin install from remote URL | Future (V2) |
| FR-PS-12 | Plugin registry / marketplace | Future (V2) |
| FR-PS-13 | Reference plugin: `git-context-plugin` (Python) implementing `git.context.assemble` and `git.diff.fetch` | Must Have |
| FR-PS-14 | Plugin developer Python library (`awis-plugin`) for implementing the JSON-RPC protocol | Must Have |
| FR-PS-15 | Plugin developer test harness: simulate JSON-RPC calls without a running AWIS runtime | Must Have |

### 10.7 Observability

| ID | Requirement | Priority |
|---|---|---|
| FR-OB-01 | `awis status [--namespace=<ns>] [--all] [--watch]` shows runtime health, active instances, and recent completions | Must Have |
| FR-OB-02 | `awis status --watch` auto-refreshes every 5 seconds | Must Have |
| FR-OB-03 | `awis trace <instance-id> [--json] [--full]` shows the complete execution timeline in chronological order | Must Have |
| FR-OB-04 | Trace output includes: step name, status, duration, attempt count, intelligence adapter used, token count, outputs (truncated; `--full` for complete), error message, fallback activations, signal arrivals | Must Have |
| FR-OB-05 | `awis history [--workflow=<id>] [--n=20] [--status=failed\|completed] [--namespace=<ns>]` shows recent instances tabularly | Must Have |
| FR-OB-06 | `awis logs [--instance=<id>] [--level=error\|info\|debug] [--tail]` streams structured JSON logs | Must Have |
| FR-OB-07 | `awis metrics [--namespace=<ns>] [--workflow=<id>]` shows aggregate statistics: completion rate, failure rate, step latency percentiles, intelligence usage | Must Have |
| FR-OB-08 | `awis audit [--from=<date>]` shows governance-level events from the AuditLog | Must Have |
| FR-OB-09 | `awis replay <instance-id>` re-runs a completed instance in dry-run mode | Should Have |
| FR-OB-10 | All commands support `--json` for machine-parseable output | Must Have |
| FR-OB-11 | Structured log lines are JSON with fields: ts, level, component, instance_id, step_id, attempt, msg | Must Have |
| FR-OB-12 | Log level is configurable: `error` (production default), `info`, `debug` | Must Have |
| FR-OB-13 | V2: metrics exported via Prometheus-compatible `/metrics` endpoint | Future (V2) |
| FR-OB-14 | V2: traces exported via OpenTelemetry-compatible collectors | Future (V2) |

### 10.8 Storage

| ID | Requirement | Priority |
|---|---|---|
| FR-ST-01 | EventLog is append-only; no event is ever modified or deleted except via governed `awis prune-events` | Must Have |
| FR-ST-02 | StateStore is a materialized projection of the EventLog; always rebuildable via `awis rebuild-state` | Must Have |
| FR-ST-03 | All SQLite operations use WAL mode for durability across crashes | Must Have |
| FR-ST-04 | AWIS uses `runtime.db` for execution state only (EventLog, StateStore, WorkflowRegistry, StepResultCache, signal_inbox, wait_records, plugin registry) | Must Have |
| FR-ST-05 | Applications manage their own SQLite databases (e.g., OIP's `oip.db`); AWIS's StoragePort is unchanged | Must Have |
| FR-ST-06 | Schema migrations use sequential versioned SQL files; applied automatically on startup (local mode) | Must Have |
| FR-ST-07 | `awis export [--format=json] [--namespace=<ns>] [--from=<date>] [--to=<date>]` exports execution history | Should Have |
| FR-ST-08 | Postgres adapter for cloud mode | Future (V2) |
| FR-ST-09 | `awis export --format=migration-bundle` + `awis import` for SQLite → Postgres migration | Future (V2) |
| FR-ST-10 | `awis prune-events --before=<date> --dry-run` for governed EventLog pruning (dry-run required) | Should Have |

### 10.9 SDK

| ID | Requirement | Priority |
|---|---|---|
| FR-SDK-01 | Go SDK public package: `github.com/awis/sdk` with stable, semver-versioned public API | Must Have |
| FR-SDK-02 | SDK surface: `awis.go`, `workflow.go`, `step.go`, `trigger.go`, `runner.go`, `intelligence.go`, `recall.go`, `testing/` | Must Have |
| FR-SDK-03 | StepHandler interface: `ID() string`, `Execute(ctx StepContext) (StepResult, error)` | Must Have |
| FR-SDK-04 | WorkflowRunner interface: Submit, Signal, Status, Cancel, List | Must Have |
| FR-SDK-05 | RecallAPI interface: QueryHistory, ReplayInstance, StepStats | Must Have |
| FR-SDK-06 | WorkflowTestHarness: runs workflows synchronously in tests; supports Signal delivery; no external services required | Must Have |
| FR-SDK-07 | Deterministic execution mode (`awis.DeterministicMode()`): fixed clock, fixed random seed, NullAdapter, manual tick | Must Have |
| FR-SDK-08 | Mock intelligence: `NewMockIntelligence()` with `OnDraft`, `OnEmbed`, `OnClassify` fixture responses | Must Have |
| FR-SDK-09 | Internal runtime packages are not importable by applications; only `github.com/awis/sdk` is public | Must Have |
| FR-SDK-10 | Python subprocess library `awis-step` for implementing step handlers in Python | Must Have |
| FR-SDK-11 | Runtime maintains backward compatibility for one major SDK version behind | Should Have |

---

## 11. NON-FUNCTIONAL REQUIREMENTS

### Performance

| ID | Requirement | Target | Priority |
|---|---|---|---|
| NFR-P-01 | Execution loop tick latency (no I/O) | < 5ms P99 | Must Have |
| NFR-P-02 | Native step dispatch latency (Go handler, no AI) | < 10ms P95 | Must Have |
| NFR-P-03 | Intelligence step latency (Anthropic Haiku, draft) | < 5s P95 | Must Have |
| NFR-P-04 | Signal delivery latency (time from `awis signal` to step resuming) | < 200ms | Must Have |
| NFR-P-05 | `awis trace` query time (100K events) | < 500ms | Must Have |
| NFR-P-06 | `awis rebuild-state` (100K events) | < 30s | Must Have |
| NFR-P-07 | Workflow instance startup latency (`awis submit` to first step started) | < 300ms | Must Have |
| NFR-P-08 | Plugin spawn time (Python, cold start) | < 3s | Must Have |
| NFR-P-09 | Plugin capability call latency (excluding handler logic) | < 50ms | Must Have |

### Reliability

| ID | Requirement | Target | Priority |
|---|---|---|---|
| NFR-R-01 | Workflow completion rate (local mode, no AI, no external services) | ≥ 99.5% | Must Have |
| NFR-R-02 | EventLog durability across process crash (WAL mode) | No data loss for committed events | Must Have |
| NFR-R-03 | State reconstruction accuracy after `rebuild-state` | 100% identical to pre-crash state | Must Have |
| NFR-R-04 | Signal delivery idempotency | Exactly-once delivery guaranteed | Must Have |
| NFR-R-05 | Plugin crash recovery | Auto-restart ≤ 3 attempts; workflow step retries | Must Have |
| NFR-R-06 | Intelligence timeout handling | Step retries per RetryPolicy; fallback activates if all exhausted | Must Have |

### Usability

| ID | Requirement | Target | Priority |
|---|---|---|---|
| NFR-U-01 | Time from `awis init` to first completed trace | ≤ 5 minutes on clean machine | Must Have |
| NFR-U-02 | Time to diagnose a failure via `awis trace` | ≤ 2 minutes | Must Have |
| NFR-U-03 | Validation error messages: include file path, line number, description, corrected example | Must Have | Must Have |
| NFR-U-04 | `awis trace` output readable without prior documentation | Must Have | Must Have |
| NFR-U-05 | No AWIS command requires reading documentation to use; errors suggest correct command | Must Have | Must Have |

### Maintainability

| ID | Requirement | Priority |
|---|---|---|
| NFR-M-01 | Solo-founder maintainable for 5+ years; no "clever" abstractions without justified use cases (P9 from architecture) | Must Have |
| NFR-M-02 | All ADRs documented in the Architecture Blueprint; no undocumented architectural decisions | Must Have |
| NFR-M-03 | Platform boundary test: OIP built without runtime modification | Must Have |

### Security

| ID | Requirement | Priority |
|---|---|---|
| NFR-S-01 | API keys and credentials read from environment variables only; never stored in workflow definitions or EventLog | Must Have |
| NFR-S-02 | Plugins run in separate processes; no access to EventLog, StateStore, or other namespaces | Must Have |
| NFR-S-03 | All SQL queries use parameterized statements; no string concatenation in queries | Must Have |
| NFR-S-04 | Namespace isolation: StoragePort prepends namespace to all queries; applications cannot query outside their namespace | Must Have |
| NFR-S-05 | AuditLog records: WorkflowRegistered, PluginRegistered, ConfigChanged, SignalDelivered; never pruned without explicit operator action | Must Have |

---

## 12. USER STORIES

### Platform Builder Stories

**PB-1:** As a Platform Builder, I want to run `awis init` in my project directory and have a working runtime environment in under 2 minutes, so that I can start building applications immediately.

**Acceptance:** `awis init` creates `.awis/config.yaml`, `.awis/.gitignore`, and three example workflows. `awis start` starts without error. Time: ≤ 2 minutes on a clean machine.

---

**PB-2:** As a Platform Builder, I want to configure my intelligence provider by setting an environment variable, so that all applications using the runtime get AI capabilities without per-application configuration.

**Acceptance:** Setting `ANTHROPIC_API_KEY` and running `awis start` shows `intelligence: anthropic (claude-haiku-4-5)` in the startup header. All intelligence steps in all workflows use the configured provider.

---

**PB-3:** As a Platform Builder, I want to run `awis status` and see all active and recent workflows, so that I can assess platform health at a glance.

**Acceptance:** `awis status` shows: runtime status, namespace, storage path, intelligence status, plugin health, active instances (with current step), and last N completed instances (with outcome, duration, age, failure reason). Total render time: ≤ 1 second.

---

**PB-4:** As a Platform Builder, I want to run `awis rebuild-state` and have the runtime reconstruct its state from the event log, so that I can recover from storage corruption without losing execution history.

**Acceptance:** `awis rebuild-state` replays the EventLog, rebuilds the StateStore, and produces output identical to pre-corruption state. Completes in ≤ 30s for 100K events.

---

**PB-5:** As a Platform Builder, I want to install a plugin with a single command and have it immediately available to all applications, so that I extend capabilities without touching application code.

**Acceptance:** `awis plugin install git-context-plugin` installs the plugin, installs dependencies, and registers capabilities. `awis plugin status git-context-plugin` shows `healthy`. Workflow steps using the plugin capability execute immediately.

---

### Application Developer Stories

**AD-1:** As an Application Developer, I want to define a workflow in YAML and have it validated before registration, so that syntax errors surface before execution time.

**Acceptance:** `awis workflow validate my-workflow.yaml` reports: valid (with step summary) or invalid (with file path, line number, error description, and corrected example). Works without a running runtime.

---

**AD-2:** As an Application Developer, I want to implement a `StepHandler` interface in Go and register it at startup, so that my business logic runs inside AWIS workflows with retry and observability for free.

**Acceptance:** An implemented StepHandler (two methods: `ID()`, `Execute()`) registered via `runtime.RegisterHandler()` executes as part of a workflow. Retry, EventLog recording, and trace visibility require zero additional developer code.

---

**AD-3:** As an Application Developer, I want to run `awis submit my-workflow` and see a workflow instance ID immediately, so that I can reference and track the execution.

**Acceptance:** `awis submit` prints instance ID within 300ms. Output includes the monitoring and tracing commands as next steps.

---

**AD-4:** As an Application Developer, I want to run `awis trace <instance-id>` and see a precise timeline of every step, so that I can debug failures in one command.

**Acceptance:** `awis trace` output includes: chronological step timeline, per-step duration, intelligence adapter + model + token count, step outputs (truncated), error messages with attempt counts, fallback activations, signal arrivals. Readable without prior documentation.

---

**AD-5:** As an Application Developer, I want to deliver a signal to a waiting workflow with `awis signal <id> signal-name`, so that I can unblock human confirmation steps during development and testing.

**Acceptance:** `awis signal` delivers the signal, transitions the workflow from `waiting` to `running`, and prints the next step. Delivery is atomic and idempotent.

---

**AD-6:** As an Application Developer, I want to write workflow tests using `WorkflowTestHarness` that run synchronously without a live runtime, so that my CI pipeline validates workflow behavior without infrastructure.

**Acceptance:** WorkflowTestHarness runs a complete workflow end-to-end in a Go test. Supports Signal delivery via `h.Signal()`. Requires no API keys, no external services, no running runtime. Completes in < 1 second for typical workflows.

---

**AD-7:** As an Application Developer, I want to see in `awis status` which step a running workflow is currently executing, so that I can verify the workflow is progressing as expected.

**Acceptance:** `awis status` active section shows: workflow ID, instance ID, current step name, and elapsed time.

---

**AD-8:** As an Application Developer, I want intelligence steps to route to fallback steps when no provider is configured, so that my workflow runs in all environments including those without API keys.

**Acceptance:** With NullAdapter active, a step declaring `required: false` and a `fallback:` activates the fallback step and records `StepFallbackActivated` in the EventLog. The workflow completes without error.

---

**AD-9:** As an Application Developer, I want to query execution history with `awis recall` and get FTS results, so that I can find relevant past executions.

**Acceptance:** `awis recall "query text"` returns matching instances from the EventLog via FTS. With `--synthesize` flag and intelligence configured, returns a synthesized answer. Without intelligence, returns raw FTS results.

---

### Plugin Developer Stories

**PD-1:** As a Plugin Developer, I want to declare my plugin's capabilities in a YAML manifest, so that AWIS knows what my plugin can do before spawning it.

**Acceptance:** `awis-plugin.yaml` with declared capabilities is sufficient for registration. `awis plugin list` shows declared capabilities after install.

---

**PD-2:** As a Plugin Developer, I want to implement my plugin in Python with a minimal JSON-RPC wrapper, so that I can write capability logic in my preferred language.

**Acceptance:** A Python plugin using `from awis_plugin import PluginServer, capability` and implementing `server.serve()` communicates with the AWIS runtime over stdin/stdout. No Go knowledge required.

---

**PD-3:** As a Plugin Developer, I want to test my plugin's JSON-RPC protocol with a mock runtime client, so that I can develop and validate the plugin without running AWIS.

**Acceptance:** `from awis_plugin.testing import mock_request` allows calling plugin capabilities in tests without a running AWIS runtime.

---

**PD-4:** As a Plugin Developer, I want to see plugin health and call statistics with `awis plugin status`, so that I can diagnose performance issues without parsing logs.

**Acceptance:** `awis plugin status <name>` shows: status, PID, uptime, memory, per-capability call count, average latency, P95 latency, error count, last error.

---

## 13. USER FLOWS

### Flow 1: First-Time Platform Setup (5 minutes)

```
1. Install
   $ go install github.com/awis/awis@latest
   → awis v1.0.0 installed

2. Initialize
   $ awis init
   → Creates: .awis/config.yaml, .awis/.gitignore
   → Creates: workflows/hello-world.yaml, workflows/with-signal.yaml,
              workflows/with-intelligence.yaml
   → Creates: handlers/example_handler.go, README_AWIS.md
   → Prints next steps

3. Start
   $ awis start
   → AWIS v1.0.0 first-run banner
   → Namespace, storage path, intelligence level, registered workflows
   → "Runtime ready. Press Ctrl+C to stop."

4. Submit example
   $ awis submit hello-world
   → "Submitted: hello-world / instance i-abc123"
   → "Monitor: awis status | Trace: awis trace i-abc123"

5. See trace
   $ awis trace i-abc123
   → Two-step timeline, both completed, durations, outputs
```

**Exit condition:** Developer has seen a complete execution trace. Time: ≤ 5 minutes.

---

### Flow 2: First Real Application Workflow (30 minutes)

```
Day 1 — Define the workflow
   Create workflows/capture-decision.yaml
   $ awis workflow validate workflows/capture-decision.yaml
   → Fix validation errors (clear messages with line numbers)
   $ awis start → "Registered: capture-decision v1.0.0"

Day 2 — Write the step handler
   Implement oip.RecordAppendHandler (Go, StepHandler interface)
   Register: runtime.RegisterHandler(&RecordAppendHandler{})
   $ awis submit capture-decision --input='{"repo_path": "."}'
   $ awis trace <id> → see step ran; diagnose error in append step

Day 3 — Add intelligence
   Add intelligence step to capture-decision.yaml
   $ awis submit → trace shows "intelligence: null, fallback: manual-entry activated"
   $ export ANTHROPIC_API_KEY=sk-ant-...
   $ awis start → "Intelligence: anthropic (claude-haiku-4-5-20251001)"
   $ awis submit → trace shows "intelligence: anthropic, tokens: 847"

Day 4 — Test properly
   Write WorkflowTestHarness test
   h.Signal(id, "entry_confirmed", ...) to unblock WAIT step
   All passing in CI (no API keys required)

Day 5 — Handle failures
   $ awis trace <failed-id>
   → See retry timeline; fix handler; redeploy
```

---

### Flow 3: Debugging a Failure (Under 2 minutes)

```
1. Notice failure
   $ awis status
   ✗ failed  capture-decision  i-m4n5o6  23s  1h ago  draft-entry: intelligence timeout

2. Inspect trace
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

3. Fix
   $ awis config set intelligence.timeout 30s
   $ awis submit capture-decision (retry)
```

**Exit condition:** Failure diagnosed and fixed. Time: ≤ 2 minutes.

---

### Flow 4: Plugin Installation and Use

```
1. Install plugin
   $ awis plugin install git-context-plugin
   → Downloads, installs Python dependencies, registers capabilities
   → "✓ git-context-plugin v1.0.0 installed"

2. Verify
   $ awis plugin status git-context-plugin
   → Status: healthy | Capabilities: 2 | Calls: 0

3. Use in workflow (update YAML)
   - id: assemble-context
     type: plugin
     handler: git-context-plugin
     ...

4. Validate
   $ awis workflow validate workflows/capture-decision.yaml
   → "Valid: capture-decision v1.0.0 (1 plugin dependency: git-context-plugin)"

5. Execute and trace
   $ awis submit capture-decision --input='{"repo_path": "."}'
   $ awis trace <id>
   → "assemble-context: plugin=git-context-plugin, duration=1.2s"
```

---

### Flow 5: Delivering a Signal to a Waiting Workflow

```
1. Submit workflow with WAIT step
   $ awis submit capture-decision --input='{"repo_path": "."}'
   → Submitted: capture-decision / i-d4e5f6

2. Monitor while waiting
   $ awis status
   ○ waiting  capture-decision  i-d4e5f6  signal: confirm-entry  71h 48m remaining

3. Deliver signal
   $ awis signal i-d4e5f6 entry_confirmed \
       --payload='{"confirmed_entry": {"title": "Use SQLite for V1"}}'
   → Signal delivered: entry_confirmed → i-d4e5f6
   → Instance resumed; next step: append-to-record

4. Confirm completion
   $ awis status
   ✓ completed  capture-decision  i-d4e5f6  89s  just now
```

---

## 14. INFORMATION ARCHITECTURE

### Primary Objects

**WorkflowDefinition** — the stable data structure representing a workflow graph.
```
WorkflowDefinition
  ├── id               (namespaced: "oip.capture-decision")
  ├── version          (semver: "1.0.0"; immutable once registered)
  ├── namespace        ("oip")
  ├── triggers[]       (what starts this workflow)
  ├── steps[]          (all steps / nodes)
  ├── transitions[]    (conditional edges between steps)
  ├── initial_step     (first step to execute)
  ├── final_steps[]    (steps that end the workflow)
  └── compensation?    (rollback plan on failure)
```

**WorkflowInstance** — a running instance of a definition.
```
WorkflowInstance
  ├── instance_id          (UUID)
  ├── definition_id        (reference to WorkflowDefinition)
  ├── definition_version   (pinned at submission time)
  ├── namespace
  ├── status               (pending|running|waiting|completed|failed|cancelled|compensating|compensated)
  ├── current_steps[]      (active step IDs)
  ├── variables            (accumulated step outputs + workflow inputs)
  ├── cancellation_requested (0|1)
  ├── started_at, updated_at, completed_at
  └── version              (optimistic lock counter)
```

**ExecutionEvent** — an append-only record of every state change.
```
ExecutionEvent
  ├── event_id, instance_id, namespace
  ├── event_type           (WorkflowStarted, StepStarted, StepCompleted, etc.)
  ├── step_id?
  ├── payload              (JSON — event-type-specific data)
  ├── emitted_at
  └── sequence_num         (monotonically increasing per instance)
```

**Plugin** — a registered external capability provider.
```
Plugin
  ├── plugin_id, name, version
  ├── manifest             (JSON-serialized awis-plugin.yaml)
  ├── status               (registered|active|suspended|failed)
  ├── capabilities[]       (declared capability IDs)
  └── registered_at
```

**IntelligenceConfig** — provider configuration and routing policy (not a stored object; part of config.yaml).

**Signal** — a message in the signal inbox that unblocks a WAIT step.
```
Signal
  ├── signal_id, instance_id, signal_name
  ├── payload              (JSON)
  ├── received_at
  └── delivered_at?        (null = pending; set = delivered)
```

### Conceptual Hierarchy

```
AWIS Runtime
├── Namespace (single in V1)
│   ├── WorkflowDefinitions (registered by applications, versioned)
│   └── WorkflowInstances (created by trigger or manual submit)
│       ├── Steps (within each instance, typed by StepType)
│       ├── Signals (inbox per instance, consumed on WAIT step delivery)
│       └── ExecutionEvents (the append-only timeline)
├── Plugins (global, shared across namespaces in V1)
├── Intelligence (configured globally, used per-step)
└── Observability (structured logs, metrics, AuditLog)
```

### Storage Separation

```
.awis/runtime.db       ← AWIS runtime state ONLY
  execution_events     (EventLog — append-only)
  workflow_instances   (StateStore — rebuildable projection)
  workflow_definitions (WorkflowRegistry)
  step_results_cache   (idempotency keys, TTL)
  signal_inbox         (pending and delivered signals)
  wait_records         (WAIT step timeout tracking)
  plugins              (registered plugins)
  plugin_capabilities  (capability declarations)

.decisions/.index/oip.db  ← OIP application database ONLY
  entries              (decision entry metadata)
  entries_fts          (FTS5 virtual table for full-text search)
  entry_vectors        (Float32 arrays for semantic search — V2)
```

These two databases are independent. No cross-database queries. `runtime.db` is a rebuildable cache of execution state. `oip.db` is rebuildable from `.decisions/entries/*.md`. The `.md` files in git are the irreplaceable artifact.

---

## 15. NAVIGATION STRUCTURE

### CLI Command Hierarchy

```
awis
│
├── RUNTIME MANAGEMENT
│   ├── init [name]                   Initialize project structure and config
│   ├── start [--config=<path>]       Start the runtime (foreground)
│   ├── stop                          Gracefully stop the runtime
│   └── version                       Show version and build info
│
├── WORKFLOW EXECUTION
│   ├── submit <workflow-id>           Submit a workflow instance
│   │   [--input='<json>']
│   │   [--wait] [--timeout=<dur>]
│   ├── signal <instance-id> <name>   Deliver signal to waiting instance
│   │   [--payload='<json>']
│   └── cancel <instance-id>          Cancel a running or waiting instance
│       [--reason=<string>] [--compensate]
│
├── OBSERVABILITY
│   ├── status [--namespace=<ns>]     Live status: active + recent instances
│   │   [--all] [--watch]
│   ├── trace <instance-id>           Full execution trace for one instance
│   │   [--json] [--full]
│   ├── history [--workflow=<id>]     Recent completed instances
│   │   [--n=20] [--namespace=<ns>]
│   │   [--status=failed|completed]
│   ├── logs [--instance=<id>]        Structured log stream
│   │   [--level=error|info|debug] [--tail]
│   ├── metrics [--namespace=<ns>]    Aggregate execution statistics
│   │   [--workflow=<id>]
│   ├── recall "<natural query>"      Query execution history
│   │   [--namespace=<ns>] [--synthesize]
│   ├── replay <instance-id>          Re-run completed instance (dry-run)
│   └── audit [--from=<date>]         View audit log entries
│
├── WORKFLOW MANAGEMENT
│   ├── workflow list [--namespace=x] List registered workflow definitions
│   ├── workflow show <id>            Show definition: steps, transitions
│   └── workflow validate <file>      Validate a YAML definition file
│
├── PLUGIN MANAGEMENT
│   ├── plugin list                   List installed plugins and status
│   ├── plugin install <path>         Install a plugin from local path
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
    ├── rebuild-state [--namespace=x] Rebuild StateStore from EventLog
    ├── export [--format=json]        Export execution history
    │   [--namespace=<ns>]
    │   [--from=<date>] [--to=<date>]
    ├── prune-events --before=<date>  Prune EventLog (--dry-run required first)
    │   [--dry-run]
    └── audit [--from=<date>]         View audit log entries
```

### Command Design Principles

1. Primary commands are one word: `awis status`, `awis trace`, `awis submit`.
2. Sub-commands for management domains: `awis workflow`, `awis plugin`, `awis config`.
3. All commands have `--json` for machine-parseable output.
4. All commands have `--help` with examples.
5. No command requires reading documentation to use; errors suggest the correct command.
6. All flags use `--long-form`; no single-letter flags except `-h` (alias for `--help`).

---

## 16. SCREEN INVENTORY

AWIS V1 has no graphical screens. The "screen inventory" is the set of terminal output formats that constitute the product interface.

| Command | Output Description | Primary Persona | Frequency |
|---|---|---|---|
| `awis init` | One-time setup summary with created files and next steps | Platform Builder | Once per project |
| `awis start` | Runtime startup log: version, namespace, storage, intelligence, plugins, registered workflows | Platform Builder | Per session |
| `awis status` | Multi-section: header (runtime health), active instances, recent completions | Both | Continuous |
| `awis trace <id>` | Chronological timeline with per-step metadata | App Developer | On failure |
| `awis history` | Tabular list: ID, status, duration, trigger, age, failure step | App Developer | Periodic |
| `awis submit` | Confirmation: workflow ID, instance ID, next commands | App Developer | Development iteration |
| `awis signal` | Delivery confirmation: signal name, instance ID, next step | App Developer | WAIT unblocking |
| `awis cancel` | Cancellation confirmation: instance ID, reason, final status | Both | Rare |
| `awis workflow list` | Tabular: ID, version, namespace, step count, instance count | App Developer | Setup verification |
| `awis workflow show <id>` | Step list with types and transitions; trigger summary | App Developer | Introspection |
| `awis workflow validate` | Valid: step summary. Invalid: line numbers, errors, examples | App Developer | Authoring iteration |
| `awis plugin list` | Tabular: name, version, status, capabilities, uptime | Platform Builder | Management |
| `awis plugin status <n>` | Detail: health, process, per-capability call stats | Platform Builder | Debugging |
| `awis plugin install` | Progress: download, dependency install, capability registration | Platform Builder | On demand |
| `awis metrics` | Aggregate: completion/failure/cancellation rates, step latency P50/P95, intelligence usage | Platform Builder | Health review |
| `awis logs` | Streaming: structured JSON log lines | App Developer | Active debugging |
| `awis config show` | YAML: current configuration with source annotation (env/file/default) | Both | Setup verification |
| `awis recall "<q>"` | FTS results list; or (with --synthesize + intelligence) synthesized answer with citations | App Developer | History query |
| `awis audit` | Tabular: timestamp, event type, actor, payload summary | Platform Builder | Governance |
| Empty states | Instructive empty states for all commands (see §26) | All | On first use |

---

## 17. DASHBOARD REQUIREMENTS

### V1: The Terminal Dashboard (`awis status`)

`awis status` is the primary dashboard in V1. It must answer three questions in one glance: Is the runtime healthy? Is anything stuck or failing? What completed recently?

**Required output sections:**

**Header (always shown):**
```
AWIS v1.0.0  ●  running
Namespace: oip  |  Storage: .awis/runtime.db  |  Intelligence: anthropic (claude-haiku-4-5-20251001)
Uptime: 3h 42m  |  Plugins: git-context-plugin (healthy)
```

**Active section (instances with status running or waiting):**
```
────────────────────── ACTIVE ──────────────────────
 ●  running   capture-decision  i-a1b2c3  step: draft-entry     12s
 ○  waiting   capture-decision  i-d4e5f6  signal: confirm-entry  71h 48m remaining
```

**Recent section (last N completed instances, default N=10):**
```
────────────────────── RECENT ──────────────────────
 ✓  completed  recall-decision   i-g7h8i9   2.3s    2m ago
 ✓  completed  capture-decision  i-j1k2l3  89s     15m ago
 ✗  failed     capture-decision  i-m4n5o6  23s     1h ago    draft-entry (3 attempts)
 ✓  completed  recall-decision   i-p7q8r9   1.8s    2h ago

Run 'awis trace <id>' to inspect any instance.
```

**Design rules:**
- Color-coded but never color-only: symbols (●, ○, ✓, ✗) carry the same information as color
- Active section only shows instances needing attention; completed instances in Recent
- Failure reason shown inline to enable immediate `awis trace`
- Footer: the most useful next command
- `--watch` auto-refreshes every 5 seconds (full-page refresh, not in-place update)
- `--json` outputs the complete status object as JSON

### V2: Web-Accessible Status Page

Read-only HTTP endpoint (`GET /status`) rendering the same information as `awis status` in a browser-accessible format. Not an interactive dashboard — static HTML with auto-refresh.

### V3: Full Web Dashboard

Full-featured web dashboard (see §35 Future Roadmap).

---

## 18. WORKFLOW BUILDER REQUIREMENTS

### YAML DSL Authoring

**File location:** `workflows/<workflow-id>.yaml` in project root. Auto-discovered by `awis start`.

**Validation command:** `awis workflow validate <file>` — works without a running runtime (syntax + schema); defers handler existence checks to registration time.

**Valid output:**
```
✓ capture-decision v1.0.0 is valid
  Steps: 4 (plugin: 1, intelligence: 1, signal: 2)
  Intelligence: draft capability (required: false, fallback: manual-entry)
  Plugins: git-context-plugin (git.context.assemble)
  Triggers: manual, event (git.push.completed)
```

**Invalid output (required format):**
```
✗ Validation failed: workflows/capture-decision.yaml
  Line 23: step 'confirm-entry' references unknown fallback 'manual-entry'
           but 'manual-entry' is not defined in this workflow

  Suggestion: Add a step with id 'manual-entry', or remove the fallback reference.
  Example:
    - id: manual-entry
      name: Manual Entry
      type: signal
      wait_signal:
        name: manual_draft_provided
        timeout: 24h
        timeout_action: fail
```

**Validation checks (required):**
- No orphaned steps (defined but not reachable from initial_step)
- No cycles (all paths terminate)
- All `handler` refs in native/subprocess/plugin steps must be resolvable at runtime
- All `fallback` references point to steps defined in the same workflow
- All `transition.from` and `transition.to` reference defined step IDs
- All `transition.condition` expressions parse per the formal grammar
- All `trigger.config.filter` expressions parse per the formal grammar
- `initial_step` references a defined step
- All `final_steps` references defined steps
- Intelligence steps: `context_budget` is a positive integer; `model_hint` is `fast|quality|local|nil`

**DSL Design Rules:**
1. YAML resolves all template expressions (`{{...}}`) at execution time, not parse time
2. Both YAML DSL and Go SDK compile to identical WorkflowDefinition structs
3. Parallel execution via fan-out: multiple `Transition` entries with same `from` activate all `to` steps concurrently
4. No `parallel` StepType exists; fan-out transitions are the mechanism
5. No cycles; cyclic workflows must use new instance submission from a step handler

### Go SDK Authoring

The SDK's WorkflowBuilder provides a fluent API for defining workflows programmatically. It is used when workflows require dynamic step generation, complex conditions, or programmatic composition beyond what the YAML DSL can express.

**Registration flow:**
```go
func main() {
    runtime, err := awis.NewRuntime(awis.Config{
        Namespace: "oip",
    })
    if err != nil { log.Fatal(err) }

    runtime.RegisterHandler(&oip.RecordAppendHandler{})
    runtime.RegisterHandler(&oip.IndexSearchHandler{})
    runtime.RegisterWorkflow(oip.CaptureDecisionWorkflow())
    runtime.RegisterWorkflow(oip.RecallDecisionWorkflow())
    runtime.Start(context.Background())
}
```

**Registration error format:**
```
awis: registration failed for "oip.capture-decision"
  step "assemble-context": handler "git-context-plugin.git.context.assemble" not found
  Hint: install the plugin first: awis plugin install git-context-plugin
  Available plugins: none installed
```

### Workflow Versioning in Authoring

- Workflow definitions are immutable once registered. Changing a workflow requires incrementing the semver.
- Old instances continue executing on their pinned version.
- New instances use the latest registered version of the workflow ID.
- `awis workflow list` shows all versions with active instance counts.
- The Go SDK validates the version field is a valid semver at Build() time.


---

## 19. WORKFLOW EXECUTION EXPERIENCE

### Manual Execution

```
$ awis submit capture-decision --input='{"repo_path": "/path/to/repo", "ref": "abc123"}'
Submitted: capture-decision v1.0.0
Instance:  i-a1b2c3
Status:    pending → running

Monitor:  awis status
Debug:    awis trace i-a1b2c3
```

Simple, immediate, actionable. The developer never needs to check a dashboard — the commands are printed.

### Triggered Execution

When a trigger fires automatically (git push event, schedule), the execution appears in `awis status` without developer action. The trigger is visible in `awis trace`:

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

When an intelligence step routes to its fallback, execution continues without interruption and the fallback is visible in the trace:

```
00:01  ► StepStarted         draft-entry (attempt 1)
00:01  → IntelligenceFallback  draft-entry: capability 'draft' unavailable (null adapter)
00:01  ► StepStarted         manual-entry (fallback activated)
00:01  ○ WaitingForSignal    manual-entry: signal 'manual_draft_provided'
```

The fallback is not an error — it is an explicit, visible execution path.

### Retry Execution

When a step fails and retries, the developer sees the retry timeline in `awis trace`:

```
00:01  ► StepStarted   draft-entry (attempt 1)
00:11  ✗ StepFailed    draft-entry  10.0s  error: request timeout  [retrying in 2s]
00:13  ► StepStarted   draft-entry (attempt 2)
00:15  ✓ StepCompleted draft-entry  2.3s   adapter: anthropic, tokens: 847
```

Silent recovery: transient failures handled within retry policy are NOT shown as errors in `awis status`. They appear only in the trace.

### Compensation on Failure

When a workflow fails after completing steps with compensation handlers:

```
$ awis trace i-failed
...
  00:25  ✗ WorkflowFailed     step: create-reservation, error: payment service timeout
  00:25  ● WorkflowCompensating
  00:25  ► CompensationStep   void-reservation (undo for: create-reservation)
  00:26  ✓ CompensationStep   void-reservation  0.8s
  00:26  ● WorkflowCompensated
```

### Cancellation Flow

```
$ awis cancel i-a1b2c3 --reason="user requested stop"
Cancellation requested: i-a1b2c3
In-flight steps will complete; no new steps will start.

$ awis status
 ✗ cancelled  capture-decision  i-a1b2c3  reason: user requested stop

$ awis trace i-a1b2c3
...
  00:12  ✓ StepCompleted     draft-entry  2.3s   [cancellation pending]
  00:12  ● WorkflowCancelled reason: user requested stop
```

With `--compensate`:

```
$ awis cancel i-a1b2c3 --compensate --reason="rollback required"
→ Instance transitions to 'compensating'; compensation plan runs
→ Final status: compensated
```

---

## 20. WORKFLOW MONITORING EXPERIENCE

### The Status View (Primary Monitoring)

`awis status` must answer three questions in one glance: Is the runtime healthy? Is anything stuck or failing? What completed recently?

**Behavioral requirements:**
- Shows ALL actively running and waiting instances (no pagination on active section)
- Shows last N completed instances in Recent (default N=10; `--n=<int>` to override)
- Failure instances show the failed step name inline
- Waiting instances show signal name and remaining timeout
- Running instances show current step name and elapsed time
- `--watch` refreshes the complete output every 5 seconds
- `--json` outputs a structured JSON object with identical information

### The Trace View (Debugging)

`awis trace <instance-id>` is the surgical debugging tool.

**Required fields per step in trace:**
- Event type (WorkflowStarted, StepStarted, StepCompleted, StepFailed, StepFallbackActivated, SignalReceived, WaitTimeout, WorkflowCompleted, WorkflowFailed, WorkflowCancelled, WorkflowCompensating, WorkflowCompensated)
- Timestamp relative to WorkflowStarted (e.g., `00:01`, `01:23`, `72:00`)
- Step ID
- Attempt number (for steps with retries)
- Duration (for completed/failed steps)
- Error message (for failed steps)
- Intelligence adapter, model, tokens used (for intelligence steps)
- Fallback step activated (for fallback events)
- Signal name and payload summary (for signal events)
- Step outputs (truncated to 120 chars; `--full` for complete)

**Output rules:**
- Chronological order always
- Human-readable durations (1.2s, not 1234ms; 72h, not 259200s)
- Color (green/red/yellow/grey) that always has a text/symbol equivalent
- `--json` outputs the complete WorkflowTrace object

### The History View (Pattern Spotting)

`awis history` provides a list view for spotting patterns across multiple instances.

**Required columns:** ID, STATUS, DURATION, TRIGGER, AGE, FAILURE_STEP (if failed)
**Required filters:** `--workflow=<id>`, `--n=<int>` (default 20), `--status=failed|completed|cancelled`, `--namespace=<ns>`
**Footer:** Total counts by status for the filtered set

### The Metrics View (Aggregate Health)

`awis metrics` provides aggregate health statistics per workflow and per step.

**Required sections:**
1. **Workflow summary:** instance count, completion rate, failure rate, cancellation rate for the period
2. **Step latency:** P50 and P95 per step, with failure count
3. **Intelligence usage:** calls per adapter, average tokens, average latency, fallback activation count

**Required filters:** `--workflow=<id>`, `--namespace=<ns>`, `--from=<date>`, `--to=<date>` (default: last 7 days)

---

## 21. INTELLIGENCE LAYER UX

### The Power Level Model

Intelligence in AWIS operates at four power levels, activated by configuration without any workflow definition changes:

**Level 0 — No intelligence (default):**
No API key configured. NullAdapter active. All intelligence steps route to their declared fallbacks. Every workflow runs.
```
AWIS ● running  intelligence: null (no API key configured)
```

**Level 1 — Basic cloud intelligence (one env var):**
```
$ export ANTHROPIC_API_KEY=sk-ant-...
$ awis start
AWIS ● running  intelligence: anthropic (claude-haiku-4-5-20251001)
```

**Level 2 — Configured intelligence (config.yaml):**
Model selection, context budgets, and model tier configuration in `config.yaml`.

**Level 3 — Multi-provider routing (V2):**
Multiple adapters with capability-based routing policy.

### Intelligence Visibility

The developer always sees which intelligence provider was used for each step:

```
✓ StepCompleted  draft-entry  2.4s
  adapter:    anthropic
  model:      claude-haiku-4-5-20251001
  tokens:     847 (prompt: 643, completion: 204)
  latency:    2.4s
```

This appears in `awis trace`. No separate cost dashboard required in V1.

### Fallback Visibility

When intelligence is unavailable or a step is `required: false`:

```
→ IntelligenceFallback  draft-entry: capability 'draft' unavailable (null adapter)
                         fallback activated: manual-entry
```

This is not an error — it is a visible execution path. The developer can distinguish between "intelligence worked" and "intelligence fell back" at a glance.

### Intelligence Configuration UX

```
$ awis config show

intelligence:
  primary.provider:     anthropic          (from: ANTHROPIC_API_KEY env var)
  primary.draft_model:  claude-haiku-4-5-20251001   (default)
  primary.quality_model: claude-sonnet-4-6           (default)
  embed.provider:       [not configured]   → embedding unavailable; semantic search disabled

storage:
  type:          sqlite
  path:          .awis/runtime.db

runtime:
  poll_interval: 100ms
  max_parallel:  4

logging:
  level: error
```

Source annotation (env var / config file / default) is shown for every value. Missing configuration that degrades functionality is flagged inline.

### `awis recall` Behavior (V1)

`awis recall "<query>"` searches execution history.

**V1 behavior:**
- Default (no flag): FTS full-text search over EventLog payloads and step outputs; returns matching instances sorted by relevance
- `--synthesize` flag (requires intelligence configured): routes query to IntelligencePort.Synthesize() with FTS results as context; returns synthesized answer with instance citations

**Empty state when intelligence not configured:**
```
$ awis recall "why did the draft step fail last week" --synthesize
Synthesis requires intelligence configured.
Current: intelligence: null (no API key configured)

Configure: export ANTHROPIC_API_KEY=<key> && awis start
Raw FTS results below:

  [FTS results for "draft step fail last week"]
```

---

## 22. PLUGIN MANAGEMENT

### Installation Experience

Plugin installation is modeled on `brew install`: one command, visible progress, immediate availability.

```
$ awis plugin install ./plugins/git-context-plugin
Fetching plugin manifest...
Reading awis-plugin.yaml from: ./plugins/git-context-plugin
Checking runtime: python3 (3.11.0 ✓)
Checking dependencies: gitpython → installing...
Installing gitpython==3.1.41...
✓ git-context-plugin v1.0.0 installed

  Capabilities:
    git.context.assemble  (timeout: 30s)
    git.diff.fetch        (timeout: 10s)

  Runtime: python3 -m git_context_plugin
  Idle timeout: 5 minutes (process killed; respawn on next request: ~1s)

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

Capability               Calls  Avg Latency  P95 Latency  Errors
git.context.assemble        34     1.2s         2.1s         0
git.diff.fetch              12     0.4s         0.8s         0

Last error: none
Installed at: /home/user/.awis/plugins/git-context-plugin/
```

### Plugin Failure Behavior

When a plugin crashes, the failure and recovery are visible in `awis trace`:

```
00:01  ► StepStarted   assemble-context (attempt 1)
00:02  ✗ StepFailed    assemble-context  1.0s
                        error: plugin git-context-plugin crashed (exit code 1)
                        Plugin auto-restarting (attempt 1/3)...

00:02  ► StepStarted   assemble-context (attempt 2)
00:03  ✓ StepCompleted assemble-context  0.9s  (plugin restarted successfully)
```

If the plugin fails all 3 restart attempts, the step fails and the workflow's retry policy applies.

### Plugin Idle Behavior

When a plugin has been idle for `idle_timeout_s` seconds, the runtime kills the plugin process. On the next capability request, the runtime respawns the process. Expected respawn latency for Python/shell plugins: < 2 seconds. Idle kill is transparent to workflows; the step sees no error.

### The Plugin Developer Experience

**Minimum viable Python plugin:**
```python
# git_context_plugin/__main__.py
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
```

**Testing without runtime:**
```python
from awis_plugin.testing import mock_request

result = mock_request("git.context.assemble", {"repo_path": ".", "ref": "HEAD"})
assert "context" in result
```

**Plugin manifest (`awis-plugin.yaml`):**
```yaml
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
  command: "python3"
  args: ["-m", "git_context_plugin"]
  env:
    GIT_TERMINAL_PROMPT: "0"
  idle_timeout_s: 300
```

---

## 23. PROJECT & WORKSPACE MANAGEMENT

### V1: Single Project, Single Namespace

V1's project model: one `awis init` per project, one namespace, one `runtime.db`.

**Project structure:**
```
my-project/
├── .awis/
│   ├── config.yaml       (git-ignored: API keys, paths)
│   ├── runtime.db        (git-ignored: SQLite EventLog + StateStore)
│   ├── runtime.db-wal    (git-ignored: WAL file)
│   └── .gitignore        (auto-created by awis init)
├── workflows/
│   ├── hello-world.yaml
│   ├── with-signal.yaml
│   └── with-intelligence.yaml
├── handlers/
│   └── example_handler.go
└── README_AWIS.md
```

**Namespace declaration in config.yaml:**
```yaml
namespace: oip
```

Namespace must match pattern `[a-z][a-z0-9-]*` (max 63 chars). Namespace collision on the same runtime is an initialization error.

### V1: Application Registration

Applications register handlers and workflow definitions at startup:

```go
func main() {
    runtime, err := awis.NewRuntime(awis.Config{
        Namespace: "oip",
    })
    if err != nil { log.Fatal(err) }
    runtime.RegisterHandler(&oip.RecordAppendHandler{})
    runtime.RegisterWorkflow(oip.CaptureDecisionWorkflow())
    runtime.Start(context.Background())
}
```

### V2: Multiple Namespaces

V2 introduces multi-namespace support for running multiple applications on one runtime:

```
$ awis namespace create neurodashboard
$ awis namespace list
NAME             STATUS   WORKFLOWS  ACTIVE_INSTANCES
oip              active   2          1
neurodashboard   active   3          0

$ awis status --namespace=oip
$ awis status --all
```

### Workspace Concept (V3)

V3 introduces "workspaces" as multi-tenant boundaries: one workspace per team, with authentication, isolated storage, and separate intelligence quotas. Out of scope for V1 and V2.

---

## 24. AI PROVIDER CONFIGURATION

### Configuration Structure

```yaml
# .awis/config.yaml (application-level; git-ignored)

intelligence:
  primary:
    provider: anthropic          # anthropic | openai | ollama | null
    # API key read from ANTHROPIC_API_KEY env var; never stored in config
    draft_model: claude-haiku-4-5-20251001
    quality_model: claude-sonnet-4-6
    base_url: ~                  # optional: proxy override

  # V2: separate embedding provider
  embed:
    provider: openai
    # API key from OPENAI_API_KEY
    model: text-embedding-3-small

  # V2: local inference
  local:
    provider: ollama
    base_url: http://localhost:11434
    draft_model: llama3.2
    embed_model: nomic-embed-text

  # V2: routing policy
  routing:
    prefer_local: false
    fallback_chain: [anthropic, null]
    capability_overrides: {}
```

### V1 Supported Providers

| Provider | V1 Support | Capabilities | Configuration |
|---|---|---|---|
| `null` | ✓ Always present | Stub returns; IsAvailable()=false | Automatic; no config |
| `anthropic` | ✓ V1 | Draft, Synthesize, Classify | ANTHROPIC_API_KEY env var |
| `openai` | Future (V2) | Draft, Embed, Synthesize, Classify | OPENAI_API_KEY env var |
| `ollama` | Future (V2) | Draft, Embed, Classify, Synthesize | Ollama running locally |

### Provider Selection Logic (V1)

With only Anthropic and Null available in V1:

1. If `ANTHROPIC_API_KEY` is set and valid: AnthropicAdapter is primary; NullAdapter is fallback
2. If no API key: NullAdapter only
3. `model_hint: fast` → `draft_model` (claude-haiku-4-5-20251001)
4. `model_hint: quality` → `quality_model` (claude-sonnet-4-6)
5. `model_hint: local` → falls back to cloud (no Ollama in V1); logs warning

### Secret Handling Requirements

- API keys are NEVER stored in workflow definitions, step inputs, EventLog payloads, or StateStore
- API keys are read from environment variables at runtime startup
- Plugin processes inherit a minimal environment; plugin-specific credentials declared in `plugin.env` are read from the environment at spawn time, not from AWIS config

---

## 25. LOCAL-FIRST BEHAVIOR

### Local-First as a Product Commitment

AWIS is designed to run completely on a developer's laptop with zero external services. This is not a "lite mode" or a "development mode" — it is the complete, production-quality platform. Cloud is a storage and concurrency upgrade, not a prerequisite.

### What "Local-First" Requires

**FR-LF-01:** `awis start` must succeed on a machine with no internet connection, no cloud accounts, no Docker, and no external services. Required: Go binary, SQLite (bundled). Optional: Anthropic API key (for intelligence).

**FR-LF-02:** All workflow functionality — step execution, retry, compensation, signal delivery, WAIT steps, state persistence, recovery — must work identically offline and online.

**FR-LF-03:** The NullAdapter must be the default intelligence provider. No workflow may fail due to the absence of an intelligence provider.

**FR-LF-04:** A workflow defined with intelligence steps but without a configured provider must execute fully, routing intelligence steps to their declared fallbacks.

**FR-LF-05:** `awis rebuild-state` must work offline. The EventLog is the source of truth; rebuilding from it requires no network.

**FR-LF-06:** `awis export` must work offline; it reads from `runtime.db` only.

### Storage Location

```
~/.awis/              (global config; created by first awis init)
  config.yaml
  plugins/
    <plugin-name>/

.awis/                (per-project; created by awis init in project root)
  runtime.db          (SQLite WAL mode)
  runtime.db-wal
  runtime.db-shm
  config.yaml         (git-ignored project overrides)
  .gitignore
```

### Cloud Upgrade Path (V2)

The upgrade from local (SQLite) to cloud (Postgres) requires:
1. `awis export --format=migration-bundle`
2. Update `config.yaml` with Postgres connection string
3. `awis import --from=migration-bundle`
4. `awis verify-storage`

No workflow definitions, step handlers, or application code changes. The StoragePort interface is identical; only the adapter changes.

---

## 26. ERROR HANDLING

### Error Design Principle

Every error AWIS shows must answer three questions:
1. **What happened?** (specific, not generic)
2. **Where?** (instance ID, step ID, plugin name, config key, file line number)
3. **What now?** (a concrete next command or action)

### Error Taxonomy

**Category 1 — User errors (configuration, definition):**
```
awis: workflow validation failed: workflows/capture-decision.yaml
  Line 45: step 'confirm-entry' declares fallback 'manual-entry'
           but 'manual-entry' is not defined in this workflow

  Suggestion: Add a step with id 'manual-entry', or remove the fallback declaration.
  Example:
    - id: manual-entry
      name: Manual Entry
      type: signal
      wait_signal:
        name: manual_draft_provided
        timeout: 24h
        timeout_action: fail
```

**Category 2 — Execution errors (step failures; visible in `awis trace`):**
```
awis: step 'draft-entry' failed after 3 attempts
  Instance:  i-m4n5o6
  Workflow:  capture-decision v1.0.0
  Error:     connection refused (anthropic API)
  Status:    workflow moved to 'failed' state; compensation running

  Diagnose:  awis trace i-m4n5o6
  Check API: awis config show | grep intelligence
```

**Category 3 — Infrastructure errors (storage, plugin):**
```
awis: plugin 'git-context-plugin' failed to spawn (3/3 attempts)
  Error:    python3: ModuleNotFoundError: No module named 'gitpython'
  Impact:   Steps requiring 'git.context.assemble' will fail

  Fix:      pip install gitpython
            awis plugin status git-context-plugin
```

**Category 4 — Transient errors (silently recovered):**
- Intelligence timeouts triggering retry
- Step retry within policy limits
- Plugin restart within limits (1/3, 2/3)

These do NOT appear in `awis status` or as error output. They appear ONLY in `awis trace` as part of the execution timeline. Silent recovery is a feature, not a hidden failure.

### Signal Timeout Error Display

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

### Empty State Error Handling

When a command that queries data finds nothing, it provides instructive output (not an error code):

```
$ awis trace i-notfound
awis: instance 'i-notfound' not found in namespace 'oip'

  List instances:   awis history
  Check namespace:  awis config show | grep namespace
```

---

## 27. PERMISSIONS & SECURITY

### V1 Threat Model

V1 is local-first, single-user. The primary threats are:
- Plugin code that crashes or hangs — handled by process isolation and auto-restart
- Secrets exposed in workflow definitions — addressed by strict secret handling rules
- SQL injection — addressed by parameterized queries throughout

V1 does NOT implement: authentication, authorization, network security, encrypted storage, or access control. These are V2 requirements when AWIS supports multiple concurrent users.

### Plugin Sandboxing

Plugins are process-isolated:
- Run as separate processes (not goroutines in the runtime)
- Receive only the step inputs declared in their manifest
- Have no access to StoragePort, EventLog, StateStore, or other namespaces
- Communicate exclusively via stdin/stdout
- Cannot register new workflow definitions
- Cannot hold state between calls (plugin state must be external to the process)

V3 will add WASM sandboxing for untrusted third-party plugins.

### Secret Management Requirements

**SR-01:** API keys and credentials must never appear in:
- Workflow definition YAML or Go SDK code
- EventLog event payloads
- StateStore variables
- Log output (structured or unstructured)
- `awis config show` output (masked with `***`)
- `awis export` output

**SR-02:** API keys are read from environment variables at runtime startup and injected into adapter constructors. They are not passed to step handlers as inputs.

**SR-03:** Plugin processes inherit a minimal environment. If a plugin needs credentials, they are declared in `plugin.env` in the manifest and read from the environment at plugin spawn time.

**SR-04:** `config.yaml` (which may contain base_url overrides) is git-ignored by the `.awis/.gitignore` created by `awis init`.

### Namespace Isolation

**NI-01:** All tables include a `namespace TEXT NOT NULL` column.

**NI-02:** The StoragePort enforces namespace in every query as a non-optional predicate. No application code can construct a query without a namespace parameter.

**NI-03:** Applications cannot read or write to another namespace's data via the SDK.

**NI-04:** Namespace collision on the same runtime (V1 single-process) is an initialization error.

### Audit Log

The AuditLog records governance-level events and is never pruned without explicit operator action:

```
WorkflowRegistered    {namespace, definition_id, version}
WorkflowDeregistered  {namespace, definition_id, version, reason}
PluginRegistered      {plugin_id, capabilities}
ConfigChanged         {key, old_value_hash, new_value_hash}
SignalDelivered       {instance_id, signal_name, delivered_at}
```

`awis audit [--from=<date>]` queries the AuditLog. The AuditLog is separate from the EventLog and uses a different storage table, making it independently auditable.

### V2 Security Requirements

V2 (multi-user server mode) will add:
- API token authentication per namespace
- HTTPS for all HTTP endpoints
- Per-namespace resource quotas
- Encrypted storage for API keys at rest

---

## 28. ACCESSIBILITY

### CLI Accessibility (V1)

The CLI is the entire product in V1. Accessibility means:

**A11Y-01 — No color-only information.** Every piece of information conveyed by color is also conveyed by symbol and text:
- ✓ = completed (green)
- ✗ = failed (red)
- ● = running/active (yellow)
- ○ = waiting (grey)
- ► = started (blue)
- → = routing/transition (grey)

Developers using terminals without color, with `NO_COLOR` set, or with screen readers see complete information.

**A11Y-02 — Machine-parseable output.** Every command supports `--json`. Assistive tools can parse structured JSON where formatted terminal tables are inaccessible.

**A11Y-03 — Predictable output structure.** Command output follows consistent patterns. Screen-reading workflows are learnable because the same information always appears in the same position.

**A11Y-04 — No time-based UI.** No spinners requiring real-time terminal updates. Progress is shown as discrete text updates. `--watch` refreshes the full output; it does not use in-place cursor manipulation.

**A11Y-05 — Keyboard-only completable.** Every AWIS operation can be accomplished via keyboard. No mouse interaction required.

**A11Y-06 — `NO_COLOR` support.** When `NO_COLOR` env var is set, all output is plain text with no ANSI color codes. All symbols remain.

**A11Y-07 — Configurable output width.** Output respects `COLUMNS` environment variable. Truncation with `--full` flag for wide content.

### Web Accessibility (V3)

V3's web dashboard targets WCAG 2.1 Level AA:
- Full keyboard navigation with visible focus indicators
- Screen reader announcements for workflow status changes
- No information conveyed by color alone
- Minimum contrast ratios on all text (4.5:1 for normal text, 3:1 for large text)
- No time-limited interactions
- Alt text on all workflow graph visualizations

---

## 29. NOTIFICATIONS

### V1: Terminal-Native, Pull-Based

AWIS V1 produces no push notifications. All information is available on demand via CLI commands. The developer's workflow is pull-based: run `awis status` to see what's happening; run `awis trace` when something needs inspection.

**`--watch` mode:**
```
$ awis status --watch
(auto-refreshes every 5 seconds; Ctrl+C to exit)
```

**`--wait` for blocking submission:**
```
$ awis submit capture-decision --wait --timeout=10m
Waiting for i-a1b2c3...
  ►  assemble-context: running (2s)
  ✓  assemble-context: completed (1.2s)
  ●  draft-entry: running...
  ✓  draft-entry: completed (2.4s)
  ○  confirm-entry: waiting for signal 'entry_confirmed' (72h timeout)
     Deliver: awis signal i-a1b2c3 entry_confirmed [--payload='<json>']
```

If the workflow reaches a WAIT step while `--wait` is active, the command continues blocking and prints the signal delivery instruction. It does not exit.

### V2: Event-Driven Notifications

V2 adds configurable webhooks for workflow lifecycle events:

```yaml
notifications:
  webhooks:
    - url: https://myapp.internal/awis-events
      events: [workflow.completed, workflow.failed, workflow.cancelled]
      namespace: oip
      secret: ~  # from AWIS_WEBHOOK_SECRET env var for HMAC verification
```

### V3: Rich Notification System

V3 adds:
- Email notifications for WAIT step timeouts (configurable per workflow)
- Slack/Discord integration for workflow failure alerts
- In-browser real-time alerts in the web dashboard (WebSocket)
- PagerDuty integration for critical workflow failures

---

## 30. SETTINGS

### Minimal Configuration Mandate

V1's configuration surface is intentionally minimal. Every configuration option must have been requested by a real user facing a real problem. Speculative configuration is not added.

### V1 Configuration File (`config.yaml`)

```yaml
# AWIS Configuration — .awis/config.yaml
# All values are optional; defaults are production-ready
# This file is git-ignored (NEVER commit API keys)

namespace: awis             # Required: change to your application name

intelligence:
  primary:
    provider: null           # Change to: anthropic | openai | ollama
    # api_key read from ANTHROPIC_API_KEY | OPENAI_API_KEY env var

runtime:
  poll_interval: 100ms       # Execution loop cadence (increase for lower CPU)
  max_parallel: 4            # Max concurrent step executions

logging:
  level: error               # error | info | debug
```

Eight options. That is the entire V1 configuration surface.

### Configuration Hierarchy

```
config.yaml (base)
  ↓ overridden by
Environment variables (ANTHROPIC_API_KEY, AWIS_NAMESPACE, AWIS_LOG_LEVEL, etc.)
  ↓ overridden by
Command-line flags (--namespace, --config, --log-level, etc.)
```

### `awis config show` Display

Each configuration value is displayed with its source annotation:
```
namespace:              oip          (from: config.yaml)
intelligence.provider:  anthropic    (from: ANTHROPIC_API_KEY env var)
intelligence.draft_model: claude-haiku-4-5-20251001  (default)
runtime.poll_interval:  100ms        (default)
runtime.max_parallel:   4            (default)
logging.level:          error        (default)
```

### Settings That Do Not Exist in V1

The following are NOT in V1 configuration by design:
- Per-step timeout overrides (use workflow-level `timeout:` in the definition)
- Intelligence cost limits (V2)
- Plugin resource limits (V2)
- Namespace quotas (V2)
- Authentication configuration (V3)
- Multi-adapter routing policy (V2)
- Metrics export endpoints (V2)
- Webhook configuration (V2)
- Retention policy (V2 — default is indefinite; governed pruning only)

---

## 31. SUCCESS METRICS

### Developer Experience Metrics (Quantitative)

| Metric | V1 Target | Measurement Method |
|---|---|---|
| Time from `awis init` to first completed trace | ≤ 5 minutes | Measured on clean macOS + Linux |
| Time from `awis init` to first application workflow | ≤ 4 hours | OIP integration timing |
| Time to diagnose a failure via `awis trace` | ≤ 2 minutes | User study with real failures |
| Validation error comprehension rate | ≥ 90% understood without docs | User comprehension test |
| Zero-AI test pass rate | 100% | CI gate (automated) |
| WorkflowTestHarness adoption | ≥ 80% of workflows have tests | Code coverage metric |
| Plugin developer: create first working plugin | ≤ 4 hours | User study |

### Platform Health Metrics (Quantitative)

| Metric | V1 Target | Measurement Method |
|---|---|---|
| Workflow completion rate (local, no AI) | ≥ 99.5% | EventLog analysis |
| Native step execution latency P95 | ≤ 50ms | EventLog analysis |
| Intelligence step latency P95 (Anthropic Haiku) | ≤ 5s | EventLog analysis |
| Intelligence fallback rate (when configured) | ≤ 5% | EventLog analysis |
| Plugin crash rate | ≤ 1% of calls | EventLog analysis |
| EventLog rebuild time (100K events) | ≤ 30s | Benchmark |
| Signal delivery latency | ≤ 200ms | EventLog analysis |

### Product Validation (Qualitative)

| Hypothesis | Validation Method | Pass Condition |
|---|---|---|
| Developers trust AWIS for production workflows | OIP E1 experiment | ≥ 5 confirmed entries/week past week 4 |
| Platform boundary is correct | OIP built on AWIS | OIP's two workflows run without platform surgery |
| Debugging is faster than alternatives | User study: AWIS trace vs. custom logging | ≥ 60% time reduction |
| Intelligence-optional architecture holds | Zero-AI CI gate | Never fails across all workflows |
| Plugin model is learnable | Plugin developer study | Working plugin in ≤ 4 hours |

### V1 Quality Gates (Shipped/Not Shipped)

These gates determine whether V1 is complete:

**QG-1:** `awis init` → `awis start` → `awis submit hello-world` → `awis trace <id>` succeeds on a clean macOS and Linux machine with no prior setup. Time: ≤ 5 minutes.

**QG-2:** `awis trace` output is parsed correctly by a developer who has not read any AWIS documentation.

**QG-3:** All workflows (including OIP's) complete successfully with `NullAdapter` as the sole intelligence provider. (AWIS-E1 gate.)

**QG-4:** OIP's `capture-decision` and `recall-decision` workflows run on AWIS without any modification to the AWIS platform code.

**QG-5:** The `WorkflowTestHarness` runs a complete workflow (including WAIT step delivery) in a Go unit test in < 1 second with zero external dependencies.

---

## 32. ACCEPTANCE CRITERIA

Acceptance criteria are organized per functional area. All Must Have criteria must pass before V1 ships.

### Runtime Management

- [ ] `awis init` completes in < 30 seconds on a clean machine
- [ ] `awis init` creates all required files and directories
- [ ] `awis start` shows intelligence level in startup header
- [ ] `awis start` registers all workflows in `./workflows/` automatically
- [ ] `awis stop` completes all in-flight steps before exiting
- [ ] `awis version` outputs version and build info

### Workflow Execution

- [ ] `awis submit` returns an instance ID within 300ms
- [ ] `awis submit` with `--input='<json>'` passes inputs to the workflow
- [ ] `awis signal` delivers signal within 200ms; instance transitions to running
- [ ] Signal delivery is idempotent: delivering the same signal twice has no effect after first delivery
- [ ] `awis cancel` with no `--compensate` leaves in-flight steps to complete; no new steps start
- [ ] `awis cancel --compensate` runs compensation plan after in-flight steps complete
- [ ] Cancelling a terminal instance returns no error

### Workflow Validation

- [ ] `awis workflow validate` on valid YAML: shows step summary and exits 0
- [ ] `awis workflow validate` on invalid YAML: shows file path, line number, error, and example; exits 1
- [ ] Validation works without a running runtime
- [ ] Fan-out transitions (multiple transitions with same `from`) are validated as valid
- [ ] Condition expressions with invalid syntax cause validation failure with precise location
- [ ] Cyclic workflows cause validation failure

### Step Execution

- [ ] Native (Go) steps execute and their output appears in the next step's inputs
- [ ] Subprocess (Python) steps execute via stdin/stdout JSON; outputs available to next step
- [ ] Plugin steps route to the correct plugin capability via JSON-RPC
- [ ] Intelligence steps route to configured adapter; or to fallback if unavailable
- [ ] Step idempotency key prevents re-execution of already-cached results
- [ ] Retry policy is respected: attempts, backoff, delay
- [ ] After all retries exhausted: fallback activates (if declared); else workflow fails

### Intelligence

- [ ] NullAdapter active with no API key: intelligence steps route to fallback; no error
- [ ] AnthropicAdapter active with API key: draft capability executes; tokens recorded in trace
- [ ] `StepFallbackActivated` event appears in EventLog when fallback is used
- [ ] Context budget enforcement: requests exceeding budget are rejected before dispatch
- [ ] `awis recall "query"` returns FTS results in V1
- [ ] `awis recall "query" --synthesize` returns synthesized answer with intelligence configured

### Plugin System

- [ ] Plugin install via `awis plugin install <path>` completes; capabilities registered
- [ ] Plugin capability call via workflow step: JSON-RPC request sent; outputs returned
- [ ] Plugin crash: auto-restart within 3 attempts; step retries against restarted plugin
- [ ] Plugin idle kill after `idle_timeout_s`: process killed; respawned on next request
- [ ] `awis plugin status` shows health, PID, call counts, latency
- [ ] Plugin has no access to EventLog or StateStore

### Observability

- [ ] `awis status` renders in < 1 second; shows all active and last 10 completed instances
- [ ] `awis status --watch` auto-refreshes every 5 seconds
- [ ] `awis trace <id>` shows chronological timeline with timing, adapter, tokens, outputs
- [ ] `awis trace --json` outputs valid JSON
- [ ] `awis history` shows tabular list with required columns and filters
- [ ] `awis metrics` shows completion rate, failure rate, step latency percentiles
- [ ] All commands support `--json` flag
- [ ] All color information is duplicated in symbols

### Storage

- [ ] EventLog: events are append-only; no event is modified after writing
- [ ] WAL mode: no data loss on process crash after SQLite WAL commit
- [ ] `awis rebuild-state`: reconstructs StateStore; result identical to pre-crash state
- [ ] OIP's `oip.db` is independent of AWIS's `runtime.db`; no cross-database query
- [ ] Schema migration runs automatically on startup (local mode)

### Security

- [ ] API keys never appear in EventLog, StateStore, or log output
- [ ] Plugin process cannot read runtime.db
- [ ] All SQL queries are parameterized
- [ ] Namespace prepended to all storage queries; cross-namespace access impossible via SDK

### Performance

- [ ] Native step dispatch latency P95 ≤ 50ms (Go handler, no I/O)
- [ ] Signal delivery latency ≤ 200ms
- [ ] `awis trace` query time (100K events) ≤ 500ms
- [ ] `awis rebuild-state` (100K events) ≤ 30s

---

## 33. RISKS & CONSTRAINTS

### Product Risks

| ID | Risk | Probability | Impact | Mitigation |
|---|---|---|---|---|
| PR-1 | Platform boundary confusion: developers don't know what AWIS owns vs. their application | High | High | "What AWIS Owns" documentation; `awis init` example shows boundary explicitly; clear SDK surface |
| PR-2 | `awis trace` output too dense: developers can't read it without documentation | Medium | High | User-test trace format early; clear symbols; progressive disclosure (`--full` for complete outputs) |
| PR-3 | NullAdapter fallback invisible: developers don't know intelligence isn't working | Medium | Medium | `awis start` header always shows intelligence level; trace shows "null adapter" explicitly |
| PR-4 | First-run failure: `awis init` or `awis start` fails on developer's machine | Medium | High | Test on clean macOS 13+ and Linux (Ubuntu 22.04); single binary, bundled SQLite; no runtime deps |
| PR-5 | Plugin installation complexity: Python dependencies break | Medium | Medium | Isolated plugin environments (venv per plugin); dependency pinning in manifest; clear error messages |
| PR-6 | YAML DSL ceiling: developers hit its limits quickly | Medium | Low | Go SDK is always available; YAML ceiling is documented honestly; SDK migration is documented |
| PR-7 | OIP integration requires platform surgery: SDK boundary is wrong | Low | High | OIP integration is the V1 gate; fix platform before second application if surgery required |
| PR-8 | EventLog grows unbounded: disk space exhaustion | Low | Medium | Log rotation guidance; `awis prune-events --dry-run` before any deletion; V2 configurable retention |
| PR-9 | Solo-founder abandonment: platform unmaintained | Medium | High | Minimal core; clean Go; full ADR documentation; architecture enables community pickup |
| PR-10 | Intelligence latency degrades UX: AI calls too slow | Medium | Medium | Context budget limits; model_hint routing; fallback to NullAdapter on timeout |

### Technical Constraints

**TC-1 — Solo-founder maintainable.** The platform must be maintainable by one developer for 5+ years. No "clever" abstractions. No over-engineered patterns. Every abstraction requires two real use cases.

**TC-2 — Go primary language.** The runtime, core SDK, and native step handlers are Go. Python is for subprocess handlers and plugins. TypeScript/Bun and shell are supported for subprocess steps.

**TC-3 — SQLite for V1.** No Postgres, no Redis, no message broker in V1. The entire platform runs on one SQLite file per project.

**TC-4 — No network required in V1.** The runtime must function completely offline. Intelligence is optional; local-first is mandatory.

**TC-5 — Architecture is frozen.** The AWIS Architecture Blueprint (33 sections, 15 ADRs) plus the 7 blocker resolutions in AWIS_ARCHITECTURE_FINALIZATION.md constitute the canonical architecture. This PRD must not contradict them.

**TC-6 — Expression language is bounded.** The formal grammars for template expressions and condition expressions are fixed. No new operators or constructs may be added to the YAML DSL without a documented architectural decision.

### Assumptions

**A-1:** The developer has Go 1.22+ installed.

**A-2:** Plugin developers have Python 3.10+ for Python plugins.

**A-3:** For intelligence features, the developer has a valid Anthropic API key (V1 only requires Anthropic; other providers are V2).

**A-4:** The Anthropic API is available and responsive for intelligence steps. Timeouts trigger retry; sustained unavailability routes to NullAdapter fallback.

**A-5:** OIP is the first and only application built on AWIS in V1. Multi-namespace isolation is not required until V2.

**A-6:** All workflows run on a single machine in V1. Distributed execution is not required until V2.

---

## 34. V1 SCOPE

### V1 Ships (Must Have)

| Capability | Priority |
|---|---|
| `awis init` — project initialization | P0 |
| `awis start/stop` — runtime lifecycle | P0 |
| YAML workflow DSL with formal expression grammar | P0 |
| Go SDK (WorkflowBuilder, StepHandler, WorkflowRunner) | P0 |
| Native runner (Go step handlers in-process) | P0 |
| Subprocess runner (Python/shell via stdin/stdout JSON) | P0 |
| Signal delivery (WAIT steps) — atomic single-transaction | P0 |
| EventLog (SQLite, append-only, WAL mode) | P0 |
| StateStore (SQLite, rebuildable projection) | P0 |
| WorkflowRegistry (SQLite) | P0 |
| NullAdapter (default intelligence, always present) | P0 |
| `awis trace` — full execution timeline | P0 |
| `awis status` — runtime dashboard | P0 |
| `awis workflow validate` — pre-registration validation | P0 |
| WorkflowTestHarness (deterministic in-process testing) | P0 |
| Cancellation semantics (graceful, idempotent, `--compensate` opt-in) | P0 |
| Compensation plans (reverse-order rollback on failure) | P1 |
| AnthropicAdapter (draft, synthesize, classify) | P1 |
| Plugin system (subprocess JSON-RPC + manifest) | P1 |
| `git-context-plugin` (reference Python plugin) | P1 |
| `awis-plugin` Python library | P1 |
| `awis history` — recent instances | P1 |
| `awis logs` — structured log stream | P1 |
| `awis metrics` — aggregate statistics | P1 |
| `awis recall "<query>"` (FTS-first; `--synthesize` flag) | P2 |
| `awis rebuild-state` — StateStore recovery | P2 |
| `awis export` — history export | P2 |
| `awis prune-events` — governed EventLog pruning | P2 |

### V1 Does NOT Ship

| Excluded Capability | Reason |
|---|---|
| Web dashboard / visual designer | Data model must stabilize first (V3) |
| Multi-namespace isolation | Single developer; single app in V1 |
| Postgres storage adapter | Cloud mode is V2 |
| OpenAI adapter | Anthropic sufficient for V1 validation |
| Ollama adapter | Local intelligence is V2 |
| Server mode (HTTP API) | V2 when second application is added |
| Multi-worker execution | V2 cloud mode requirement |
| Plugin registry / marketplace | V2; V1 supports manual install |
| Cost tracking per namespace | V2 when multi-adapter creates complexity |
| Webhook triggers | V2 |
| WASM plugin sandboxing | V3 |
| Charter model / machine authority delegation | V3 |
| Multi-tenant workspaces | V3 |
| Authentication / authorization | V3 |
| Metrics Prometheus export | V2 |
| OpenTelemetry trace export | V2 |

### V1 Implementation Sequence

The authoritative implementation timeline for V1:

| Period | Work |
|---|---|
| Days 1–3 | Schema design only: EventLog and WorkflowDefinition formats. `schema_version` field mandatory on all tables. No code. |
| Week 1 | EventLog + StateStore (SQLite) in parallel with IntelligencePort + NullAdapter |
| Week 2 | Execution engine: pull-based loop, NativeRunner, Signal handler |
| Week 3 | Go SDK (WorkflowBuilder, WorkflowTestHarness) + YAML DSL parser |
| Week 4 | SubprocessRunner + git-context-plugin (Python) |
| Week 5 | OIP built on AWIS: `capture-decision` and `recall-decision` workflows. Platform boundary validation. |
| Week 6 | CLI (all commands), dogfood metrics, AnthropicAdapter, V1 quality gates |

**Note:** This timeline supersedes all prior timeline estimates in the Architecture Blueprint and Investigation documents, which were estimates, not commitments.

---

## 35. FUTURE ROADMAP (V2/V3)

### V2 — Multi-Application (Months 3–6)

**Theme:** "Run all your applications on one platform."

**V2 Capabilities:**
- Multi-namespace isolation with per-namespace resource quotas
- Server mode: HTTP API exposed locally; multiple applications connect to one runtime
- Postgres storage adapter (cloud deployment path)
- OpenAI adapter + Ollama adapter
- Multi-provider routing with fallback chains and capability overrides
- Cost tracking: per-namespace, per-adapter, per-step
- Metrics export: Prometheus-compatible `/metrics` endpoint
- OpenTelemetry trace export
- `awis recall` with full intelligence synthesis (not just FTS)
- Plugin registry: curated community plugins
- Webhook triggers (HTTP incoming)
- `awis namespace` commands (create, list, delete)
- `awis status --all` for cross-namespace view
- Web-accessible status page (`GET /status`) — read-only HTML
- Schema migration: operator-controlled in multi-worker mode
- Embed capability: embedding-based semantic search over execution history

**V2 Quality Gates:**
- Two applications in separate namespaces: zero data bleed between them
- Intelligence cost per workflow instance visible in trace
- Plugin from community registry installs in ≤ 2 minutes
- Postgres migration from SQLite completes without data loss

### V3 — Platform (Months 7–18)

**Theme:** "Onboard new applications in minutes. Govern machine authority explicitly."

**V3 Capabilities:**
- Full web dashboard: status, trace, history, workflow graph visualization (read-only first; edit in V3.1)
- Multi-tenant workspaces with authentication (API tokens per workspace)
- Visual workflow designer: reads WorkflowDefinition from registry; writes back via SDK
- Charter model: bounded machine authority delegation; charted actors; trust ladder
- WASM sandboxing for untrusted third-party plugins
- Adaptive intelligence routing: performance-based adapter selection from historical data
- Cross-namespace governed data grants (explicit, auditable)
- Plugin marketplace with install counts, ratings, verified publishers
- `awis audit` with full-text searchable audit log
- Charter-aware `awis signal`: signals to chartered actors require authorization level check
- V3 quality gates: new application onboarded without platform code change; charter-bounded action taken and auditable

### V4+ — External Platform

When the platform serves external developers beyond the solo-founder's portfolio:
- Public plugin marketplace as discovery and monetization surface
- SDK becomes a public API with formal versioning guarantees and deprecation cycles
- Web dashboard becomes the primary onboarding surface for non-CLI developers
- SaaS deployment model with per-tenant BYOK (bring your own API key)

---

## 36. GLOSSARY

**Adapter** — A concrete implementation of IntelligencePort or StoragePort for a specific provider (AnthropicAdapter, SQLiteAdapter, etc.).

**AuditLog** — An append-only, separately stored log of governance-level events (workflow registration, plugin registration, config changes). Never pruned without operator action.

**Capability** — A declared function a plugin or intelligence provider can perform (e.g., `git.context.assemble`, `draft`, `embed`).

**CapabilityRouter** — The runtime component that selects an intelligence adapter for a given capability request based on model_hint and fallback_chain.

**Compensation** — Rollback logic that runs in reverse step order when a workflow fails. Specified per step as a `CompensationRef`; invoked only on steps with downstream successors.

**Condition Expression** — A boolean expression in `Transition.condition` or `Trigger.config.filter`. Uses the formal grammar specified in AWIS_ARCHITECTURE_FINALIZATION.md §Blocker 2.

**DomainEvent** — An application-defined event that can trigger a workflow instance. Distinct from ExecutionEvent (which is produced by the runtime).

**EventLog** — The append-only SQLite table (`execution_events`) that records every state change in every workflow instance. The source of truth for all execution state.

**ExecutionEvent** — An individual record in the EventLog representing one state change (WorkflowStarted, StepCompleted, SignalReceived, etc.).

**Fan-out Transition** — Multiple Transition records with the same `from` step. All target steps are activated concurrently by the execution loop. The mechanism for parallel step execution in V1.

**Fallback** — A step declared in `Step.fallback` that is activated when the step fails after all retry attempts, or when an intelligence capability is unavailable.

**Handler** — A concrete implementation of StepHandler (Go) or a capability function (Python plugin) that performs the actual work of a step.

**IdempotencyKey** — A deterministic key (`instance_id + step_id + attempt_number`) used to prevent re-execution of a step whose result is already cached in StepResultCache.

**IntelligencePort** — The Go interface (`Draft`, `Embed`, `Synthesize`, `Classify`, `IsAvailable`) that all AI adapters implement. The seam between the runtime and intelligence providers.

**Namespace** — A string identifier that scopes all workflow definitions, instances, and events to a specific application. Enforced in all StoragePort queries.

**NullAdapter** — The always-registered intelligence adapter that returns deterministic stub responses and `IsAvailable() = false`. The default when no provider is configured.

**OIP (Organizational Intelligence Platform)** — The first application built on AWIS. Its two workflows (`capture-decision`, `recall-decision`) are the platform's V1 validation target.

**Plugin** — A process-isolated external capability provider communicating via JSON-RPC 2.0 over stdin/stdout. Declared in `awis-plugin.yaml`.

**PluginRunner** — The runtime component that routes `plugin`-type steps to registered plugins via JSON-RPC.

**Pull-based Execution** — The runtime's execution model: the engine scans for runnable steps each tick (100ms) rather than being pushed work by a message broker.

**Signal** — A named message delivered to a workflow instance to unblock a WAIT step. Delivered via `awis signal <instance-id> <signal-name>`.

**StateStore** — The SQLite table (`workflow_instances`) that holds the current state of all workflow instances. A materialized projection of the EventLog; always rebuildable.

**Step** — The atomic unit of work in AWIS. Has typed inputs, typed outputs, a handler reference, a retry policy, and an optional fallback step.

**StepHandler** — The Go interface (`ID() string`, `Execute(ctx StepContext) (StepResult, error)`) that application developers implement for native steps.

**StepResultCache** — A TTL-keyed cache of step results indexed by IdempotencyKey. Prevents re-execution of already-completed steps on retry.

**StepType** — The execution environment for a step: `native | subprocess | plugin | intelligence | signal`.

**StoragePort** — The Go interface that all storage adapters implement. In V1: SQLiteStorageAdapter. In V2+: PostgresStorageAdapter.

**SubprocessRunner** — The runtime component that executes `subprocess`-type steps by spawning a process and communicating via stdin/stdout JSON.

**Template Expression** — A `{{path-ref}}` embedded in a YAML string field value. Resolved at execution time to a concrete value from `workflow.inputs` or `steps.<id>.outputs`. Uses the formal grammar specified in AWIS_ARCHITECTURE_FINALIZATION.md §Blocker 2.

**Transition** — A directed edge in the workflow graph from one step to another, optionally conditioned on a boolean expression.

**Trigger** — A condition that starts a new workflow instance: `manual | schedule | event | webhook`.

**WAIT Step** — A step of type `signal` that parks the workflow instance in `waiting` status until a named signal is delivered or a timeout expires.

**WorkflowDefinition** — The stable data structure representing a workflow graph: steps, transitions, triggers, compensation. Immutable once registered; versioned by semver.

**WorkflowInstance** — A running (or completed) execution of a WorkflowDefinition, pinned to a specific version. Has its own accumulated variables and execution timeline.

**WorkflowTestHarness** — The SDK testing utility that runs a workflow synchronously in a Go test with no external dependencies, supporting deterministic clock, mock intelligence, and signal delivery.

---

## 37. DOCUMENT METADATA

### Document Information

| Field | Value |
|---|---|
| Document Title | AWIS Canonical Product Requirements Document |
| PRD Version | 1.0 |
| Document Status | **Canonical — Implementation-Ready** |
| Date | 2026-07-02 |
| Author | AWIS Product Requirements Process |
| Authority Basis | AWIS_ARCHITECTURE_BLUEPRINT.md + AWIS_ARCHITECTURE_FINALIZATION.md |

### Change Log

| Version | Date | Change | Author |
|---|---|---|---|
| 1.0 | 2026-07-02 | Initial canonical PRD — post-architecture freeze | PRD Process |

### Cross-References

| Document | Relationship | Status |
|---|---|---|
| `AWIS_ARCHITECTURE_BLUEPRINT.md` | Canonical architecture — this PRD must not contradict it | **Frozen** |
| `AWIS_ARCHITECTURE_FINALIZATION.md` | 7 blocker resolutions — all decisions incorporated | **Frozen** |
| `AWIS_CANONICAL_SPECIFICATION_TRIBUNAL_REPORT.md` | Tribunal findings — all blockers resolved | Reference |
| `AWIS_PRODUCT_REQUIREMENTS_INVESTIGATION.md` | Pre-PRD investigation — fully incorporated | Reference |
| `OIP_CONSTITUTION.md` | Governing principles for OIP application | Active |
| `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` | Historical standalone OIP design — **superseded** | Archived |
| `PRD_RESEARCH_REPORT.md` | V1 product definition research | Reference |

### Assumptions and Dependencies

**Architecture Dependencies:**
- The AWIS Architecture Blueprint is frozen. Any change to the architecture requires a new Architecture Decision Record and a PRD amendment.
- The 7 blocker resolutions in AWIS_ARCHITECTURE_FINALIZATION.md are binding. Specifically: expression language grammars, signal atomicity model, cancellation semantics, FTS ownership boundary, compensation rules, and StepType enum are settled.

**Implementation Dependencies:**
- Go 1.22+ toolchain
- SQLite (bundled in the Go binary via `modernc.org/sqlite` or equivalent; no C dependency)
- Python 3.10+ (for plugin development and subprocess steps)
- Anthropic API access (for intelligence features; optional)

**Open Items (Non-blocking for PRD):**
These items are settled enough to not block implementation but are noted for implementer awareness:

| Item | Status | Resolution Path |
|---|---|---|
| `classify` in IntelligencePort (no V1 use case) | Non-callable placeholder in V1 | Remove from interface if still unused in V2 |
| CapabilityRouter V1 simplification | Internal implementation detail | May simplify internally while maintaining public routing spec |
| Plugin idle state (process kill vs. suspend) | Process kill recommended | Document expected respawn latency (< 2s for Python) |
| V2 timeline | Not committed | Define after V1 ships and OIP integration is validated |

---

## DECLARATION

> **"This document is the Canonical Product Requirements Document for AWIS v1.0 and shall serve as the primary product specification for implementation."**

This PRD:
- Faithfully translates the frozen AWIS Architecture Blueprint into a product specification
- Does not contradict the Architecture Blueprint or the 7 Blocker Resolutions
- Incorporates all resolved decisions from AWIS_ARCHITECTURE_FINALIZATION.md
- Supersedes all prior product definition documents as the source of truth for what AWIS V1 is, what it does, and how users experience it
- Is complete enough for engineers, designers, QA, and future AI implementation agents to proceed with implementation without requiring additional specification work

Architecture is frozen. Product requirements are defined. Implementation may begin.

---

*AWIS Canonical PRD v1.0 — 2026-07-02*
*Cross-reference: AWIS_ARCHITECTURE_BLUEPRINT.md (frozen) + AWIS_ARCHITECTURE_FINALIZATION.md (all 7 blockers resolved)*
