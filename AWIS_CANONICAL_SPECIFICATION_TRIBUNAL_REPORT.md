# AWIS CANONICAL SPECIFICATION TRIBUNAL REPORT
## Adversarial Review of the Complete AWIS Specification

**Produced by:** Canonical AWIS Specification Tribunal v2.0
**Date:** 2026-07-02
**Documents under review:**
- `OIP_CONSTITUTION.md` (52 articles)
- `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` (OIP V1 standalone design)
- `AWIS_ARCHITECTURE_BLUEPRINT.md` (AWIS platform design)
- `AWIS_PRODUCT_REQUIREMENTS_INVESTIGATION.md` (product specification)

**Default posture:** Skepticism. Every subsystem must justify its existence.
**Status:** Pre-implementation. All decisions are reversible at acceptable cost.

---

## 1. EXECUTIVE VERDICT

The specification is **substantively correct** in its foundational decisions and **substantively incomplete** in its specification precision. The architecture is sound: pull-based execution, EventLog as source of truth, IntelligencePort seam, SQLite local storage, OIP-as-first-application validation gate, and CLI-first product experience are all well-reasoned and defensible under adversarial scrutiny.

However, the specification contains **seven must-fix issues** that, if carried forward into a PRD, will produce either an internally contradictory document or will force architectural decisions mid-implementation under time pressure — the worst possible moment. These issues are not large. They are specification gaps and cross-document inconsistencies, all resolvable in a focused correction pass. They are catalogued in full in §38.

**Seven critical findings:**
1. Document schism: two incompatible OIP architectures coexist (standalone vs. AWIS-application)
2. Expression language undefined: YAML DSL uses two different syntaxes with no formal grammar
3. Signal delivery atomicity: crash between signal_inbox persistence and StateStore update undefined
4. `awis cancel` semantics: unspecified — does compensation run? do in-flight steps complete?
5. OIP's FTS5 index ownership: which SQLite file? AWIS's runtime.db or OIP's own?
6. Compensation violates the append-only invariant in the canonical example
7. `parallel` StepType listed in the type enum but never defined

None of these require architectural redesign. All require specification precision. The specification earns:

**VERDICT: ADDITIONAL_ARCHITECTURAL_WORK_REQUIRED**

Estimated correction effort: one focused day. After correction: READY_FOR_CANONICAL_PRD.

---

## 2. OVERALL SPECIFICATION READINESS

| Dimension | Assessment | Score |
|---|---|---|
| Architectural soundness | Well-reasoned; pull-based, event-sourced, port-based design is proven | 9/10 |
| Cross-document consistency | Two incompatible OIP blueprints; critical contradictions | 4/10 |
| Specification completeness | Several gaps that would block implementation mid-stream | 6/10 |
| Solo-founder sustainability | Correctly sized for one engineer; some V1 over-reach | 7/10 |
| Product clarity | Strong; `awis trace` as hero command is excellent product thinking | 8/10 |
| Implementation feasibility | Core is buildable; plugin system V1 scope requires explicit justification | 7/10 |
| Gall's Law compliance | Borderline; V1 is complex enough to concern | 6/10 |
| YAGNI compliance | Several V2+ features pulled forward; plugin system is the primary offender | 6/10 |

**Overall readiness: 6.6/10 — needs focused correction before PRD generation.**

---

## 3. PRODUCT CONSISTENCY AUDIT

The Product Requirements Investigation is internally consistent and well-structured. Findings are minor.

**PR-PASS-01:** The three-tier product vision (V1/V2/V3) aligns with the architecture roadmap. Feature assignments are plausible and defensible.

**PR-PASS-02:** The `awis trace` hero command framing is excellent product strategy. The debugging experience as the product's trust-builder is a durable insight.

**PR-PASS-03:** The "intelligence as power level" mental model accurately describes the NullAdapter → AnthropicAdapter transition. This is the right way to communicate AI-optionality.

**PR-ISSUE-01 (SAFE_TO_DEFER):** The three personas (Platform Builder, Application Developer, Plugin Developer) are presented as distinct users. In the V1 solo-founder context, all three are the same person. Documentation organized by persona will confuse the primary V1 user who inhabits all three simultaneously. PRD should note: "In solo-founder V1, all personas are the same engineer." Documentation should be task-organized, not persona-organized, until V2 when distinct roles emerge.

**PR-ISSUE-02 (SAFE_TO_DEFER):** Section §23 (Onboarding) promises `awis init` creates `README_AWIS.md` — "3 pages max." This is a content commitment in a product spec, which is correct, but the PRD should note that this file's content requires authoring time in the implementation schedule. It is easy to underestimate.

**PR-ISSUE-03 (SHOULD_FIX):** The V1 implementation timeline is stated as "Weeks 1–8" in the Product Investigation, "6-week verdict" in the OIP Engineering Blueprint, and "6-week sequence" in the AWIS Architecture Blueprint. These cannot all be true. The six-week sequence in the AWIS Blueprint places OIP integration in weeks 5-6, with CLI and dogfood in week 6. The OIP Blueprint's six-week sequence is a separate plan for OIP alone. The correct interpretation: AWIS foundation (weeks 1-4) + OIP on AWIS (weeks 5-6) + dogfood (week 6 overlap) = 6 weeks for AWIS+OIP combined. But the AWIS Blueprint also includes "CLI" as a week-6 deliverable alongside dogfood — this is too much for one week. **The PRD must reconcile these into a single, realistic implementation timeline.**

**PR-ISSUE-04 (SAFE_TO_DEFER):** Section §28 references "awis.dev/plugins" as a community registry URL. This is a brand/marketing assumption embedded in a product specification. The domain may not exist; the registry may be built differently. This should be expressed as a placeholder, not a URL.

---

## 4. ARCHITECTURE CONSISTENCY AUDIT

The AWIS Architecture Blueprint is internally consistent on its own. The critical issues arise in cross-document comparison (§5, §10). Architecture-internal findings:

**ARCH-PASS-01:** The five-layer runtime architecture (Workflow Engine → Step Runtime → Intelligence Layer → Persistence Layer → Observability Engine) is clean and non-overlapping. Each layer has a single responsibility.

**ARCH-PASS-02:** The StoragePort interface is correctly designed. Four stores (EventLog, StateStore, WorkflowRegistry, StepResultCache) with clear access patterns and a single abstraction boundary.

**ARCH-PASS-03:** The IntelligencePort interface is minimal and correct. Five methods; four adapters; NullAdapter as permanent last-in-chain. This is the specification's strongest individual design decision.

**ARCH-PASS-04:** ADR-002 (pull-based over push) is correctly reasoned. The 100ms poll interval is appropriate for all V1 target use cases; none require sub-100ms step initiation.

**ARCH-PASS-05:** ADR-003 (append-only EventLog as source of truth) is proven engineering. The StateStore-as-projection with rebuild capability is the correct recovery strategy.

**ARCH-ISSUE-01 (MUST_FIX):** `parallel` is listed as a StepType in §6 (`StepType: native | subprocess | plugin | intelligence | signal | parallel`) but is never defined anywhere in the document. What is a parallel step? What fields does it have? How does the runtime know when a parallel step "completes" (when all children complete? when any child completes?)? How are child steps defined (sub-steps within the parallel step, or sibling steps in the transitions graph with dependencies on the same parent)? This is a specification gap that will create ambiguity during implementation. **Either define parallel step semantics or remove it from the V1 type enum.**

**ARCH-ISSUE-02 (MUST_FIX):** The YAML DSL uses two syntactically distinct expression mechanisms in the canonical example:
- Template interpolation: `"{{workflow.inputs.repo_path}}"` and `"{{steps.assemble-context.outputs.context}}"` (double-brace syntax)
- Condition expressions: `"steps['draft-entry'].status == 'fallback'"` and `"event.branch == 'main'"` (bare expression syntax with bracket notation)

These are different grammars with no formal specification for either. Concrete problems this creates:
- Implementers will make different choices for dot vs. bracket notation, string comparison, null handling, and type coercion
- Documentation cannot explain the syntax without a formal grammar
- Validation cannot be complete without knowing what expressions are legal
- The condition `steps['draft-entry'].status` uses bracket notation while the template uses dot notation (`steps.assemble-context.outputs.context`) — inconsistent within the same document

**This is a must-fix before PRD.** The fix is a one-page formal grammar specification, not an architectural change.

**ARCH-ISSUE-03 (MUST_FIX):** Signal delivery transaction semantics are not specified. The execution loop (§8) describes:
1. `SIGNAL_SCAN`: check signal inbox for pending signals
2. Deliver to waiting instances

But "deliver" requires two writes:
- Update `workflow_instances.status` from `waiting` to `running` (StateStore)
- Append `SignalReceived` event to `execution_events` (EventLog)
- Update `signal_inbox.delivered_at`

If the process crashes between any of these writes, the system is inconsistent. Does the signal get delivered again on next startup (double-delivery)? Is it lost? The spec must declare the atomicity contract. The pragmatic answer (SQLite WAL, single transaction, StateStore+EventLog+signal_inbox all in the same database file in local mode) resolves this, but it must be stated — because in cloud mode (Postgres), the same transaction might not work across tables.

**ARCH-ISSUE-04 (MUST_FIX):** Cancellation semantics for `awis cancel <instance-id>` are not defined. The CLI contract lists `cancel` with a `--reason` flag, and `InstanceStatus` includes `cancelled`. But:
- What happens to in-flight steps when cancellation is requested? (Kill immediately? Wait for completion?)
- Does compensation run on cancellation, or only on failure?
- What happens to WAIT steps — are their signal inbox entries removed?
- Is cancellation idempotent (cancelling a `completed` workflow — what happens)?
- What happens if a step completes between the cancel request and the runtime processing it?

These are implementation-critical semantics that the PRD must define.

**ARCH-ISSUE-05 (SHOULD_FIX):** The plugin idle state machine describes "IDLE — Plugin receives no requests for `idle_timeout`; paused" but "paused" is ambiguous between:
- (A) Process killed (simpler, cross-platform, but 2-5 second respawn latency for Python plugins)
- (B) Process suspended via SIGSTOP (instant resumption, but POSIX-only — not portable to Windows)

The spec must choose. Given local-first on Linux/macOS and the stated preference for simplicity, option (A) is likely correct — but the respawn latency implication (Python loading `gitpython` takes ~2 seconds) must be acknowledged. A step with a 30-second timeout will still complete, but the first call after an idle period has +2s latency. Developers should know this to set appropriate expectations.

**ARCH-ISSUE-06 (SHOULD_FIX):** The `event` trigger type is used in the canonical OIP YAML example:
```yaml
triggers:
  - type: event
    config:
      event: git.push.completed
      filter: "event.branch == 'main'"
```
But the specification never defines how AWIS receives external events in V1. Is `git.push.completed` produced by a git hook that calls an AWIS internal API? By a file watch? By polling `git log`? By an application that calls the EventBus programmatically? The event trigger type is referenced throughout the spec but its source mechanism for V1 is not defined. Either: (a) define the event source mechanism for the git.push.completed event, or (b) remove the event trigger type from V1 and use manual submit in the OIP capture workflow, which is simpler and more honest.

**ARCH-ISSUE-07 (SHOULD_FIX):** The CapabilityRouter as designed (§17) is V2+ infrastructure. It includes model hints (fast/quality/local), per-capability provider overrides, cost-aware routing (listed as V2 but the interface accommodates it), and a fallback chain configuration. V1 has exactly two adapters: Anthropic and Null. The V1 routing logic is: "if Anthropic is configured and capable, use it; otherwise use Null." The full CapabilityRouter machinery — configurable fallback chains, per-capability overrides, model hints — is premature. V1 routing can be a 15-line function. Designing the full router now creates documentation and maintenance overhead that V1 doesn't justify. **Simplify V1 routing; formalize the router in V2 when a third adapter creates genuine routing decisions.**

---

## 5. CROSS-DOCUMENT CONSISTENCY MATRIX

| Claim | Constitution | OIP Blueprint | AWIS Blueprint | PRD Investigation | Verdict |
|---|---|---|---|---|---|
| OIP is a standalone binary | — | YES (single `oip` binary, `.decisions/` dir) | NO (OIP is an AWIS application) | NO (references AWIS runtime) | **CONTRADICTION** |
| Intelligence interface name | — | `IntelligencePort` (OIP's own) | `IntelligencePort` (AWIS runtime's) | `IntelligencePort` (AWIS's) | Same name, different owner |
| OIP's index storage | — | `.decisions/.index/index.db` (OIP's own SQLite) | "OIP namespace" (in AWIS runtime.db?) | Not specified | **AMBIGUOUS** |
| 6-week implementation | — | YES (standalone OIP) | YES (AWIS + OIP) | "8 weeks" (V1 scope) | **INCONSISTENT** |
| Append-only Record | Article 7 | ENFORCED (AP-4, write-once files) | References compensation that may violate it | Not addressed | **TENSION** |
| RecordPort | — | YES (OIP's own interface) | Absorbed into AWIS StoragePort | Not mentioned | Resolved only in AWIS Blueprint |
| Index is a cache | — | YES (AP-2: `oip rebuild-index`) | StateStore is a projection (same principle) | EventLog is truth | Consistent in spirit, different mechanism |
| Exit is `git clone` | Article 49 | YES (files in git repo) | PARTIAL (EventLog in runtime.db, not git) | Not addressed | **PARTIAL INCONSISTENCY** |
| Null adapter as default | Article 32 | YES (E3 gate: all commands pass with Null) | YES (NullAdapter always registered) | YES (NullAdapter default) | Consistent |
| AI never mandatory | Article 32 | YES (degraded ADR practice without AI) | YES (NullAdapter fallback chain) | YES (power level model) | Consistent |
| Record is organization's property | Article 6 | YES (files in org's git repo) | OIP Record remains OIP-owned in `.decisions/` | Not addressed | Consistent only if OIP files stay outside AWIS runtime.db |

**Critical row: Exit is `git clone`.** The Constitution requires that exit is a `git clone`. The OIP Blueprint satisfies this because The Record lives as plain files in the git repository. The AWIS Blueprint places workflow execution history (EventLog) in `runtime.db` — which is git-ignored. The OIP *decision record* still lives in `.decisions/entries/` and can be cloned. But the AWIS *execution history* (who ran what workflows, when, with what outcomes) lives only in `runtime.db`. If `runtime.db` is git-ignored and not backed up, execution history is not portable via git clone. The Constitution's exit rights apply to The Record (OIP entries), not to AWIS operational history — so this is not a constitutional violation — but the specification should state this distinction explicitly so it is not silently assumed.

---

## 6. PRODUCT ↔ ARCHITECTURE ALIGNMENT

**ALIGNED — 12 points:**
1. CLI-first V1 aligns with single-binary architecture
2. `awis trace` as hero command aligns with EventLog as source of truth
3. "Intelligence as power level" aligns with NullAdapter-as-default architecture
4. 5-minute first-run target aligns with zero-dependency SQLite architecture
5. Plugin install as "one command" aligns with subprocess JSON-RPC architecture
6. OIP as V1 validation aligns with "first application validates platform" principle
7. V1 single namespace aligns with single-developer constraint
8. Three personas align with three SDK surfaces (WorkflowBuilder, StepHandler, Plugin protocol)
9. Recall command aligns with RecallAPI + IntelligencePort synthesis capability
10. V3 visual designer aligns with stable WorkflowDefinition data model commitment
11. Multi-namespace V2 aligns with row-level namespace isolation in StoragePort
12. "Trust is built in microseconds" aligns with precise structured error messages in architecture

**MISALIGNED — 3 points:**

**MIS-01 (SHOULD_FIX):** The Product Investigation (§8, V1 In Scope) lists "Basic recall (`awis recall`)" as P2 priority. The Architecture Blueprint includes `RecallAPI` with full intelligence synthesis as a V1 component. Synthesis-mode recall requires the IntelligencePort to synthesize over multiple entries — this is a non-trivial prompt engineering and response-parsing task. If this is P2 in the product scope, the PRD should reflect that `awis recall` in V1 may be keyword search (FTS over EventLog) without synthesis, with synthesis upgrading in V2. Leaving both documents with different scopes creates implementation ambiguity.

**MIS-02 (SHOULD_FIX):** The Product Investigation §12 describes `awis workflow validate` working "without a running runtime" for syntax validation, with "handler existence checks deferred to registration time." But the Architecture Blueprint §15 (YAML DSL, DSL Design Rules) says "validation checks: all handler refs resolvable." These are contradictory. If handler resolution requires a running runtime, `awis workflow validate <file>` (pre-registration) cannot fully validate. If handler resolution is deferred to registration, the DSL design rules statement is wrong. **The PRD must specify exactly what `awis workflow validate` checks vs. what runtime registration checks.**

**MIS-03 (MUST_FIX):** The Product Investigation §13 (Screen Inventory) lists `awis workflow show <id>` as "Show definition detail (steps, transitions)" but `awis workflow show` does not appear in the Architecture Blueprint's CLI contract (§28 deployment section, or CLI discussion). The command exists in the PRD but not in the architecture. Small gap but must be formally added to the CLI specification.

---

## 7. HIDDEN ASSUMPTIONS

The following assumptions are made throughout the specification but never explicitly stated. Each creates an implementation risk if the assumption is wrong.

**HA-01 (MUST_FIX):** *Assumption: OIP's implementation switches from the OIP Engineering Blueprint to the AWIS-application model.*
Neither document explicitly states this. A developer reading both documents will find two complete, different architectures for OIP. The PRD must explicitly deprecate the OIP Engineering Blueprint as the standalone design and declare it superseded by the AWIS-application model. Otherwise, an implementer might build OIP twice — once standalone (following the OIP Blueprint) and once on AWIS (following the AWIS Blueprint).

**HA-02 (SHOULD_FIX):** *Assumption: git context assembly is best implemented as a Python plugin, not a native Go step handler.*
The entire V1 plugin system exists to support `git-context-plugin` as a Python subprocess. If git context assembly were implemented as a native Go step handler (using `go-git` or `os/exec("git ...")`), the plugin system could be deferred to V2. This assumption is never examined. The specification should explicitly justify: "We implement git context as a Python plugin rather than a Go step handler because [reason]." If the reason is "demonstrate the plugin system," the PRD should say so.

**HA-03 (SHOULD_FIX):** *Assumption: the go-git or shell git integration is straightforward.*
The git-context-plugin must assemble commit messages, diffs, PR metadata, and thread context. This requires git API calls that vary significantly between different git hosting providers (GitHub, GitLab, Gitea, local bare repos). The specification doesn't address this complexity. A reference plugin that only works on GitHub-hosted repos has a narrower scope than the specification implies.

**HA-04 (SHOULD_FIX):** *Assumption: `awis start` and the application binary run in the same process.*
The Architecture Blueprint §25 shows Go applications calling `awis.NewRuntime(...)` and `runtime.Start(...)` — this implies the AWIS runtime is an in-process library, not a separate process. But the CLI says `awis start` starts the runtime. If AWIS is an in-process library embedded in the application, `awis start` as a separate CLI command doesn't make sense for the embedded use case. If AWIS is a separate process, the application communicates with it via the HTTP SDK (V2 server mode). **V1's deployment model — whether the runtime is in-process or out-of-process for the application — is critical and must be made explicit.**

**HA-05 (SAFE_TO_DEFER):** *Assumption: the developer knows what namespace to use.*
`awis init` creates a `config.yaml` with `namespace: awis` as default. The developer must change this to `oip` or `neurodashboard`. The spec assumes developers will notice and update this. In practice, developers often run with defaults. The onboarding experience should prompt for namespace at `init` time.

**HA-06 (SAFE_TO_DEFER):** *Assumption: AWIS can be meaningfully used without reading documentation.*
The specification repeatedly says things like "all commands have `--help`" and "error messages suggest the correct command." These are commitments that require significant documentation authoring effort. They are correct goals but hidden implementation cost items.

---

## 8. MISSING DECISIONS

Decisions the specification has not made but must make before implementation.

**MD-01 (MUST_FIX):** *What happens when `awis cancel` is called?*
Required decisions:
- Do in-flight steps complete before cancellation? (Recommendation: yes, for cleanup integrity)
- Does compensation run on cancellation? (Recommendation: configurable per workflow; default: no)
- Are signal inbox entries for this instance cleaned up? (Recommendation: yes)
- Is cancellation idempotent? (Recommendation: yes — cancelling a completed workflow is a no-op with a warning)

**MD-02 (MUST_FIX):** *What is the expression language grammar?*
Required decisions:
- One syntax for both templates and conditions, or two?
- Supported operators: equality, comparison, boolean, string interpolation
- Null/missing value handling (fail validation? return empty string? return null?)
- Scope: `workflow.inputs`, `steps.<id>.outputs`, `steps.<id>.status`, `env.<var>` — what is in scope?
- Error behavior: what happens if an expression references a step that hasn't completed?

**MD-03 (MUST_FIX):** *Who owns OIP's FTS5 index — AWIS runtime.db or OIP?*
Two options, each with different implications:
- Option A: OIP owns a separate `.decisions/.index/index.db` SQLite (as per OIP Engineering Blueprint). AWIS doesn't know about it. The FTS5 index is OIP's internal implementation detail.
- Option B: OIP uses AWIS's `runtime.db` with namespace-prefixed tables including FTS5 virtual tables. AWIS's StoragePort must expose a mechanism to create and query arbitrary tables.

Option A is simpler and maintains the OIP Engineering Blueprint's behavior. Option B requires extending StoragePort with arbitrary table creation. **The PRD must choose.**

**MD-04 (MUST_FIX):** *What is a `parallel` step?*
Required decisions:
- Are parallel steps a distinct step type, or just multiple transitions from one step to multiple others?
- If it's a distinct type: what fields does it have? How are sub-steps defined?
- What is the completion condition? All sub-steps complete? Any sub-step completes?
- How does failure in one parallel branch affect others?
- Are parallel steps in V1 or V2?

**Recommendation:** Remove `parallel` from the V1 StepType enum. Parallel execution can already be expressed as a workflow-level graph (multiple transitions from one step to multiple subsequent steps, with a join point). A distinct `parallel` StepType adds complexity without adding capability that the transition model doesn't already provide.

**MD-05 (SHOULD_FIX):** *What happens when a workflow definition is replaced?*
The spec says registration is by `(id, version)` pair. But `awis start` auto-discovers YAML files and registers them. If a developer changes `version: 1.0.0` to `version: 1.1.0` in their YAML, both versions exist in the registry after restart. What if the developer forgets to increment the version? Does re-registering `v1.0.0` with different content overwrite or error? The spec says "once registered, a definition version is immutable" — but the auto-discovery on `awis start` must enforce this.

**MD-06 (SHOULD_FIX):** *How does workflow deregistration work?*
A developer removes a workflow from their YAML directory. On next `awis start`, the workflow is no longer auto-discovered. But existing instances of that workflow (if any) still exist in the StateStore, pinned to their version. Does the registry retain the definition for those instances? Forever? With a TTL? The spec says "old instances run to completion on their version" — but if the definition is removed from the registry, can they be introspected via `awis workflow show`?

**MD-07 (SHOULD_FIX):** *How is the runtime started in the embedded SDK use case?*
The Go SDK shows applications calling `awis.NewRuntime(...)` and `runtime.Start(...)`. This is embedded (in-process). But the CLI shows `awis start` as a top-level command. For OIP's use case, does the developer:
- (A) Run `awis start` in one terminal, then run `oip` in another — two processes
- (B) Run `oip` which internally calls `awis.NewRuntime(...).Start(...)` — one process

If (A): the application communicates with AWIS via local socket or HTTP (server mode — which is V2)
If (B): `awis start` as a CLI command is for standalone mode when no application embeds the runtime

This is a critical deployment model decision. V1 most likely means (B) — OIP embeds the AWIS runtime. The standalone `awis start` command is for future use cases (standalone AWIS server). **The PRD must state this clearly.**

---

## 9. MISSING COMPONENTS

Things that the specification references but does not define.

**MC-01 (MUST_FIX):** *`parallel` StepType implementation.* Referenced but undefined (see MD-04 above).

**MC-02 (MUST_FIX):** *`awis workflow show <id>` CLI command.* Listed in the Product Investigation's screen inventory but not in the Architecture Blueprint's CLI specification or deployment model section. Needs to be added to the CLI contract.

**MC-03 (SHOULD_FIX):** *Event source for `type: event` triggers.* The trigger type exists; the mechanism for receiving events is not specified. For V1, the most likely mechanism is: the application emits a domain event to AWIS's EventBus via an SDK call. This call path is not in the SDK surface (§12). The RecallAPI and WorkflowRunner are defined; an EventBus submission method is not.

**MC-04 (SHOULD_FIX):** *`awis workflow register` command.* The spec describes auto-discovery (`awis start` discovers `./workflows/`) and SDK registration (`runtime.RegisterWorkflow()`). But there is no explicit `awis workflow register <file>` command for manually registering a single workflow without restarting the runtime. `awis workflow validate` validates; something must register. If auto-discovery handles this at startup, what about workflows added after startup?

**MC-05 (SHOULD_FIX):** *Plugin test harness.* §27 (Testing Strategy) says plugins can be tested by "simulating AWIS's JSON-RPC protocol" but provides no tooling for this. The PRD should specify a `awis plugin test` command or a reference test harness that simulates the plugin protocol, so plugin developers can test without a running runtime.

**MC-06 (SAFE_TO_DEFER):** *Workflow step graph visualization for `awis workflow show`.* The architecture mentions this command produces "step graph + transitions" — but what does a step graph look like in a terminal? A textual DAG representation (similar to `git log --graph`)? This is a UX detail that needs to be decided before implementation.

---

## 10. CONTRADICTIONS

Direct contradictions between documents or within a single document.

**CON-01 (MUST_FIX) — The OIP Architecture Schism:**

*Document 1 (ENGINEERING_ARCHITECTURE_BLUEPRINT.md):*
```
OIP V1 BOUNDARY: oip CLI binary
├── capture | recall | browse | init | rebuild-index | export
.decisions/
├── entries/       ← THE RECORD (files in git)
└── .index/
    └── index.db   (OIP's own SQLite)
```
"The V1 system is a single binary and a `.decisions/` directory. No server, no database server."

*Document 2 (AWIS_ARCHITECTURE_BLUEPRINT.md):*
```
OIP Application (namespace: "oip")
├── Workflow: capture-decision [AWIS workflow]
├── Step Handlers: RecordAppendHandler, IndexFTSHandler
└── Data Stores: .decisions/entries/*.md + .decisions/.index/
```

These describe fundamentally different systems. One is a standalone tool; the other is an application on AWIS's runtime. Both use the same project name. A developer reading both must choose, and the specification does not tell them to.

**Root cause:** The OIP Engineering Blueprint was produced first as a standalone design. The AWIS Architecture Blueprint redesigned OIP as an AWIS application. The OIP Blueprint was not explicitly superseded.

**Correction:** The PRD must explicitly state: "ENGINEERING_ARCHITECTURE_BLUEPRINT.md describes OIP as a standalone tool. This design is superseded by the AWIS-application model in AWIS_ARCHITECTURE_BLUEPRINT.md. For implementation, OIP is an AWIS application. The standalone OIP architecture is archived as a historical reference only."

**CON-02 (MUST_FIX) — Compensation Violates Article 7:**

*AWIS Blueprint §7 (YAML example):*
```yaml
compensation:
  steps:
    - step: append-to-record
      undo: oip.record.mark-draft
```

*OIP Constitution Article 7:*
"The Record is corrected by appending truth, never by silently rewriting the past."

*OIP Engineering Blueprint AP-4:*
"Entry files are written once and never modified. Corrections are new entries."

The compensation action `oip.record.mark-draft` is called to "undo" an appended Record entry. If this handler modifies the entry file, it violates Article 7 and AP-4. If it appends a new entry that marks the original as superseded (the constitutionally correct approach), then "undo" is a misleading name — and the compensation model is not truly undoing anything (the original entry still exists, permanently).

**Root cause:** The compensation example was written without checking it against the append-only invariant.

**Correction:** Either: (a) Remove the `append-to-record` step from the compensation plan (appending an entry is intentional and does not need undoing; the Record is the durable artifact); or (b) Rename the compensation handler to `oip.record.supersede-entry` and document that it appends a new entry with `status: voided` referencing the original, not modifying it.

**CON-03 (MUST_FIX) — Signal Delivery: Two Syntaxes for One Mechanism:**
*(See ARCH-ISSUE-02 above — duplicate confirmed as a contradiction, not just a gap.)*

The YAML DSL uses `{{...}}` for value interpolation and bare expression syntax for conditions. These are different syntactic forms for expressions in the same document. The WorkflowValidator must implement one parser or two. The spec doesn't acknowledge this as a choice.

**CON-04 (SHOULD_FIX) — "Never Deleted" vs. "Governed Pruning":**

*Architecture Blueprint P3:*
"State is event-sourced; history is never deleted."

*Architecture Blueprint §9:*
"Governed pruning via `awis prune-events --before=<date> --dry-run`"

*Constitution Article 4 (Non-negotiable #4):*
"History is durable: the Record is corrected by appending truth, never by silently rewriting the past. Erasure exists only as a deliberate, governed, itself-recorded act."

These are not truly contradictory — governed erasure is explicitly permitted by the Constitution. But P3's statement "history is never deleted" is stronger than the Constitution's actual position and contradicts the pruning mechanism. P3 should read: "State is event-sourced; history is durable by default and erased only by governed, recorded act."

**CON-05 (SHOULD_FIX) — Validation: Pre-runtime vs. At-registration:**

*Product Investigation §15:*
"`awis workflow validate` works without a running runtime (validates syntax and schema; defers handler existence checks to registration time)"

*Architecture Blueprint §7:*
"Validation checks: ...all handler refs resolvable, all schema refs valid"

If handler resolution requires checking the runtime's registered handler list, it cannot be done without a running runtime. If "all handler refs resolvable" means only syntax-level checking (the handler string is present and non-empty), this is a different meaning than "resolvable." These need to be reconciled: define two validation modes — `awis workflow validate --offline` (syntax only) and `awis workflow validate` (requires running runtime for handler resolution).

**CON-06 (SAFE_TO_DEFER) — Two Timeline Specifications:**
*(See PR-ISSUE-03 above — restated as confirmed contradiction.)*

Three different V1 timelines exist across three documents. Must be reconciled into one in the PRD.

---

## 11. ARCHITECTURAL INVARIANT VIOLATIONS

Invariants the specification declares and then violates.

**INV-01 (MUST_FIX):** *Invariant: "No abstraction without two real use cases" (P9, AP-5).*

The YAML DSL `parallel` StepType has zero defined use cases. The `parallel` type appears in the type enum with a note that "validation checks: no cycles (unless parallel type)" — implying the type exists and is handled — but no workflow example uses it, no semantics are defined, and no second use case is identified. By the specification's own principle, this abstraction must be removed from V1.

**INV-02 (MUST_FIX):** *Invariant: "Append-only correctness" (P3, AP-4, Article 7).*

The compensation example `oip.record.mark-draft` as an "undo" of `append-to-record` implies modification of a completed Record entry, which violates the append-only invariant. This invariant is the specification's most-cited design constraint. Its violation in a canonical example is a significant credibility issue for the spec.

**INV-03 (SHOULD_FIX):** *Invariant: "Intelligence is a declared capability, not a dependency" (P2).*

The `awis recall "<query>"` command, as specified, requires intelligence synthesis to produce meaningful answers. The empty state for `awis recall` when intelligence is null reads:
"This query requires: ...Intelligence configured for synthesis (current: null adapter)"

This makes `awis recall` **non-functional without intelligence** — which contradicts P2. Either: (a) `awis recall` in V1 uses FTS-only search (no synthesis, no intelligence required), presenting keyword-matched results instead of synthesized answers, or (b) `awis recall` is a V2 feature that is honest about requiring intelligence. Do not ship a command that is broken by default.

**INV-04 (SHOULD_FIX):** *Invariant: "Minimal core, open edge" (P9, Article 41-42).*

The CapabilityRouter (§17) with model hints, per-capability overrides, and fallback chain configuration is a V2 capability placed in the V1 core. V1's two adapters (Anthropic + Null) create a routing decision space of exactly one: "use Anthropic if available, else Null." The full router is an edge capability that has been pulled into the core without the "two real use cases" justification. A third adapter (the second real use case that justifies the router's full design) is V2.

---

## 12. RESPONSIBILITY OWNERSHIP ANALYSIS

| Subsystem | Owner | Clear? | Concerns |
|---|---|---|---|
| Workflow execution scheduling | AWIS Runtime | Clear | None |
| Step execution | AWIS Runtime | Clear | None |
| EventLog persistence | AWIS Runtime | Clear | None |
| StateStore projection | AWIS Runtime | Clear | None |
| IntelligencePort routing | AWIS Runtime | Clear | Router overcomplicated for V1 |
| Plugin lifecycle | AWIS Runtime | Clear | Idle state (killed vs suspended) unresolved |
| OIP's FTS5 index | **AMBIGUOUS** | **No** | Is it in AWIS runtime.db or OIP's own SQLite? |
| OIP's Record entries (`.decisions/entries/`) | OIP application | Clear | Correctly application-owned |
| Application business logic | Application | Clear | None |
| Workflow definition registration | **AMBIGUOUS** | **No** | Auto-discovery vs. programmatic vs. CLI command |
| Signal source (who produces domain events) | Application | Unclear | SDK event emission method not defined |
| YAML expression evaluation | AWIS Runtime | Clear | Expression language not defined |
| Context budget enforcement | AWIS Runtime | Clear | Token counting mechanism not specified |
| Plugin test tooling | **ABSENT** | **No** | Not in any document |

**Key ambiguity: OIP FTS5 index ownership** is the most impactful because it determines the StoragePort interface boundary. If OIP owns its own SQLite, StoragePort doesn't need to support FTS5 virtual tables. If OIP uses AWIS's SQLite, StoragePort needs a raw query interface that breaks the abstraction.

---

## 13. ABSTRACTION LEAKAGE ANALYSIS

**ABS-01 (MUST_FIX):** *Expression Language Leakage*
The YAML DSL abstracts workflow logic into a declarative format. But the expression language (`{{...}}` and bare conditions) is essentially a mini-scripting language that leaks implementation complexity into what should be a simple configuration format. If the expression language grows (conditional logic, arithmetic, string operations), the YAML DSL will become a programming language — a pattern that every config-as-code tool has fallen into (Helm templates, Ansible Jinja2, GitHub Actions expressions are cautionary tales). **The PRD should define the expression language boundary: what expressions are explicitly NOT supported, to prevent scope creep.**

**ABS-02 (SHOULD_FIX):** *Intelligence-specific fields in Step*
The `Step` struct has a top-level `intelligence: IntelReq?` field that is only meaningful when `type == intelligence`. This is a type-tagged union implemented as a flat struct with optional fields — a known anti-pattern that makes the Step struct valid in invalid states (e.g., `type: native, intelligence: {...}` — what does this mean?). The cleaner model is a discriminated union or a separate `IntelligenceStep` struct. The current approach leaks the intelligence-specific model into all step types. At minimum, the spec should state "if `intelligence` is set on a non-intelligence step, it is ignored" — or better, make it a compile-time error.

**ABS-03 (SAFE_TO_DEFER):** *NullAdapter as "default" leaks AI dependency assumption*
The specification correctly says NullAdapter is the default. But the product UX shows:
```
AWIS ● running  intelligence: null (no API key configured)
```
The phrase "no API key configured" implies that an API key is *expected* and its absence is a configuration gap. This subtly communicates AI dependence. The correct phrasing: "intelligence: none (add API key to enable AI features)" — which treats AI as an opt-in, not a missing configuration.

---

## 14. WORKFLOW RUNTIME REVIEW

The workflow runtime (execution engine, step model, state machine, signal protocol) is the specification's strongest section. The pull-based execution loop, EventLog-as-truth, and optimistic locking model are well-reasoned and precedented. The following findings are refinements, not fundamental challenges.

**RT-01 (MUST_FIX):** `parallel` StepType is undefined. Remove from V1 or define completely.

**RT-02 (MUST_FIX):** Signal delivery atomicity. In SQLite local mode, this is easily solved (one transaction across signal_inbox + execution_events + workflow_instances, all in the same database file). The spec must state this explicitly rather than leaving it implicit. In Postgres mode (V2), the same transaction semantics apply. The fix: "Signal delivery is atomic within a single database transaction covering signal_inbox, execution_events, and workflow_instances."

**RT-03 (SHOULD_FIX):** The execution loop SETTLE step says: "append result event to EventLog" then "Upsert WorkflowInstance in StateStore." If the EventLog append succeeds but StateStore upsert fails (crash, constraint violation), the EventLog has an event that the StateStore doesn't reflect. The recover path is `awis rebuild-state` — but this is not mentioned as the recovery from this specific failure. The spec should note: "Any StateStore inconsistency with the EventLog is recoverable via `awis rebuild-state`. The EventLog is always authoritative."

**RT-04 (SHOULD_FIX):** The `StepResultCache` has TTL-based expiry. But the spec doesn't define what "TTL" means in context: is it time since first cache (expiry at creation+TTL) or time since last access (LRU)? For idempotency purposes, TTL from creation is correct (the same step invocation key should always return the same cached result within the retry window, not be evicted by LRU). Clarify.

**RT-05 (SAFE_TO_DEFER):** The execution loop tick is 100ms by default. For long-running workflows with WAIT steps spanning days/weeks, a 100ms tick wastes 86,400 SQLite queries per day checking for signals on a dormant instance. V2 should add smart tick rate adaptation: full rate for active instances, degraded rate (1s or longer) for instances waiting on signals. Not a V1 issue, but worth noting for the V2 implementation.

---

## 15. INTELLIGENCE LAYER REVIEW

The IntelligencePort is the specification's cleanest design decision. The following findings are refinements.

**INT-01 (SHOULD_FIX):** Context budget enforcement as token count is model-specific. A `context_budget: 3000` in a step declaration means 3,000 tokens — but token count varies by model and tokenizer (Claude's tokenization differs from GPT-4's). The Intelligence Layer "enforces this before dispatching to the provider" — but to do so, it needs a model-specific tokenizer. Options:
- (A) Enforce in characters (simple, approximate, model-agnostic): `context_budget: 12000` (chars, ~3000 tokens in most models)
- (B) Leave enforcement to the adapter (adapter knows its tokenizer)
- (C) Implement a rough approximation (÷4 for English text) with known imprecision

**Recommendation: Option (B) for V1.** Each adapter enforces its own budget. The `context_budget` field is a hint to the adapter, not an enforced platform invariant. The adapter may truncate context if it exceeds the budget and should log when it does. This is simpler and more accurate than platform-level token counting.

**INT-02 (MUST_FIX):** `awis recall` requires intelligence synthesis to be useful. But P2 says "Intelligence is a declared capability, not a dependency." `awis recall` as a platform command violates this by being non-functional with NullAdapter (the empty state explicitly says it "requires intelligence configured for synthesis"). Either:
- (A) `awis recall` in V1 does FTS-only search (no synthesis) and returns matched entries with relevance ranking — fully functional without intelligence. Intelligence synthesis is a `--synthesize` flag, optional.
- (B) `awis recall` is V2 only.

**Recommendation: Option (A).** FTS-mode recall is genuinely useful for developers wanting to find past executions. Synthesis enhances recall; it doesn't create it.

**INT-03 (SAFE_TO_DEFER):** The `classify` capability in IntelligencePort appears in the interface definition but has no V1 use case. OIP's workflows don't need classification; NeuroDashboard and Shade Ledger don't exist in V1. By P9 ("no abstraction without two real use cases"), `classify` should be removed from the V1 interface and added when the second use case exists.

---

## 16. PLUGIN ARCHITECTURE REVIEW

The plugin system is the specification's most complex V1 component. The stdin/stdout JSON-RPC design is correct for the reasons stated (ADR-006). The question is whether the full system belongs in V1.

**PLG-01 (MUST_FIX):** Plugin idle state must be specified as killed or suspended. See ARCH-ISSUE-05.

**PLG-02 (SHOULD_FIX — potentially MUST_FIX):** *Is the V1 plugin system justified?*

The entire plugin system (manifest format, JSON-RPC 2.0 protocol, plugin registry, PluginRegistry SQLite tables, capability router integration, lifecycle manager, health monitoring, restart logic) is justified in V1 by one use case: `git-context-plugin` implemented in Python.

**Adversarial challenge:** Could `git-context-plugin` be implemented as a native Go step handler using `os/exec("git", ...)` or the `go-git` library? If yes, the plugin system has zero V1 use cases and should be deferred to V2.

**Evidence for Go alternative:**
- `go-git` is a pure Go git library that can read commit history, diffs, and remotes without shelling out
- `os/exec("git", "log", ...)` works on any machine with git installed (V1 targets developers who by definition have git)
- The output is parseable JSON or structured text

**Evidence against Go alternative:**
- If multiple plugins are anticipated (not just git), the plugin system design work now pays off in V2
- Python may be preferred for this specific plugin because it has richer git library ecosystem (GitPython handles edge cases in commit parsing)
- The plugin protocol is the V1 "test case" for V2's plugin ecosystem

**Assessment:** The plugin system in V1 is justifiable IF the spec explicitly states "we are building the plugin system in V1 to demonstrate the protocol and validate it with one reference plugin (git-context-plugin). If git-context were the only plugin we ever needed, we would implement it as a native Go handler." The PRD should make this justification explicit. Without it, the plugin system is V2+ complexity in V1 without documented rationale.

**PLG-03 (SHOULD_FIX):** The plugin capability declaration includes `timeout_ms` per capability. But the step also has a `timeout: Duration?` field. Which timeout wins? The step-level timeout is a guarantee to the workflow; the plugin-level timeout is a hint to the runtime. The spec needs to clarify the precedence: "step timeout governs; plugin capability timeout is a default that the step timeout overrides."

**PLG-04 (SHOULD_FIX):** Plugin dependency installation (`pip install gitpython` in the install example) happens as part of `awis plugin install`. If the system Python environment is in use by other projects and packages conflict, plugin installation will fail or corrupt the environment. The spec should note that plugins should be installed in isolated Python virtual environments (`venv` per plugin), not the system Python. The reference plugin should demonstrate this.

---

## 17. SDK REVIEW

The Go SDK design is clean and well-bounded. Findings are refinements.

**SDK-01 (SHOULD_FIX):** The `StepHandler` interface requires implementing `ID() string`. This is a convention for registration, but it couples the step handler to its registration key. A cleaner pattern: `runtime.RegisterHandler("handler-id", &MyHandler{})` — the handler doesn't need to know its own ID. The current design means renaming a handler requires changing its implementation file. Minor but worth fixing before the interface is published.

**SDK-02 (SHOULD_FIX):** The `WorkflowRunner` interface includes `List(ctx, filter InstanceFilter)` which returns a `[]WorkflowStatus`. But `InstanceFilter` is not defined in the SDK spec. What fields does it have? The PRD should include a minimal filter definition: `{namespace, definition_id, status, from_time, to_time, limit}`.

**SDK-03 (SHOULD_FIX):** The `WorkflowTestHarness` is described but not enough of its API is defined. Key gaps: 
- How does the harness advance clock time (for timeout testing)?
- How does `h.WaitForCompletion` work — polling? blocking? timeout?
- How are step handler errors injected for failure-path testing?
- Can multiple workflows run concurrently in the harness?

These are critical for testing WAIT steps with timeouts, which is the most complex V1 test scenario.

**SDK-04 (SAFE_TO_DEFER):** The `RecallAPI.ReplayInstance` method "replays a completed instance for debugging." What does replay produce? A new WorkflowInstance with the same inputs but re-executed? Or a read-only trace reconstruction? The distinction matters: one creates new execution artifacts; the other is a read-only debug view. Clarify.

---

## 18. DEVELOPER EXPERIENCE REVIEW

**DX-01 (SHOULD_FIX):** The `awis trace` output is the product's most important UX surface and receives the most attention in the specification. One gap: the trace format shows `outputs` for each step, but the spec doesn't define output truncation. A step that outputs a large JSON object (e.g., the full git context from `assemble-context`) will flood the trace view. The spec should define: `--full` flag for complete outputs; default truncates to 256 chars per field with an indicator of truncation.

**DX-02 (SHOULD_FIX):** The onboarding sequence creates three example workflows: `hello-world.yaml`, `with-signal.yaml`, `with-intelligence.yaml`. But these require different step handler registrations. If OIP is the first real application, the example handlers in the AWIS init directory will be in a different Go package than OIP's handlers. The onboarding flow must explain how to run the example handlers. The current spec implies `awis start` auto-discovers workflows but doesn't explain who provides the step handlers for the example workflows. This will confuse first-time users.

**DX-03 (SHOULD_FIX):** The V1 scope includes `awis recall` at P2 priority, but makes it non-functional without intelligence (INV-03). A developer who runs `awis recall "what happened"` and gets a "requires intelligence configured" message after a successful onboarding will feel confused — the CLI seems broken. This is a trust-breaking moment. Fix: make recall functional in FTS mode without intelligence.

**DX-04 (SAFE_TO_DEFER):** The `awis config edit` command "opens config in $EDITOR." This is a good pattern (used by `git config --edit`, `kubectl edit`). But it requires defining what validation happens after editing — does AWIS re-validate the config on save? Does it restart if intelligence configuration changed? The spec should note that config changes require `awis stop && awis start` to take effect (except for logging level, which might be dynamic).

---

## 19. CLI REVIEW

The CLI is well-designed. The command vocabulary is coherent, the organization makes sense, and the primary commands (`status`, `trace`, `submit`, `signal`) are one-word as specified. Findings are minor.

**CLI-01 (MUST_FIX):** `awis workflow show <id>` exists in the Product Investigation's screen inventory but not in the Architecture Blueprint's CLI contract. Add it to the architecture CLI specification.

**CLI-02 (SHOULD_FIX):** `awis submit <workflow-id>` takes `--input='<json>'` as a flag. For complex JSON inputs, shell escaping becomes painful. An alternative: `--input=@filename.json` (file reference) and `--input=-` (stdin). This pattern is well-established (curl, jq) and should be in the V1 CLI spec.

**CLI-03 (SHOULD_FIX):** `awis status` refreshes every 5 seconds in `--watch` mode. But the spec doesn't define what happens to the terminal on refresh — full clear-and-redraw? Append? Full clear-and-redraw is the right UX (like `watch -n 5 awis status` but integrated), but this is not stated.

**CLI-04 (SAFE_TO_DEFER):** `awis audit [--from=<date>]` is in the command list but the audit log schema and queryable fields are not in the CLI spec. What can you filter by? Who created which entries? This is a V1 governance requirement that needs a minimal query model.

**CLI-05 (SAFE_TO_DEFER):** The CLI includes both `awis logs` (structured log stream from the runtime) and `awis trace <id>` (execution timeline for one instance). These overlap in what they show. The distinction should be clearer: `awis logs` is the runtime's operational log (startup, shutdown, configuration); `awis trace` is the workflow execution record. Developers debugging step failures should always use `awis trace`, not `awis logs`. The spec should make this explicit.

---

## 20. EXTENSIBILITY REVIEW

**EXT-01 (SHOULD_FIX):** The spec defines three extension points (step handlers, plugins, intelligence adapters) but does not define how a new *trigger type* is added. V1 has `manual` and `event` triggers; V2 adds `webhook` and `schedule`. How does a developer add a custom trigger type? This is a fourth extension point that is missing from the Extension Strategy (§29) and should be added.

**EXT-02 (SAFE_TO_DEFER):** The spec mentions "new trigger types in V2 follow the same plugin model" — but trigger sources are runtime-level concerns, not step-level concerns. A plugin that provides trigger sources must integrate with the execution loop (SCAN_TRIGGERABLE), which is different from step execution. The trigger plugin model needs a separate specification when V2 begins.

---

## 21. GALL'S LAW COMPLIANCE

Gall's Law: "A complex system that works is invariably found to have evolved from a simple system that worked."

**Assessment:** The AWIS V1 specification is on the edge of Gall's Law compliance. The spec correctly derives its design from simpler precedents (pull-based like Celery, event log like Git, single binary like SQLite tools). However, the cumulative V1 surface is substantial:

V1 includes: EventLog + StateStore + WorkflowRegistry + StepResultCache (4 storage tables) + NativeRunner + SubprocessRunner + PluginRunner + IntelligenceRunner (4 runners) + Plugin manifest format + JSON-RPC protocol + Plugin lifecycle manager + Plugin health monitoring + IntelligencePort + CapabilityRouter + AnthropicAdapter + NullAdapter + YAML DSL + expression language + Go SDK (WorkflowBuilder + StepHandler + WorkflowRunner + RecallAPI) + WorkflowTestHarness + full CLI (15 commands) + structured logging + metrics + audit log + signal protocol + compensation plans + workflow versioning.

This is more than a "simple system." The question is whether each piece is genuinely load-bearing for V1 (OIP) or represents anticipatory design.

**Gall's Law verdict: Borderline.** The core (EventLog + StateStore + NativeRunner + IntelligencePort + NullAdapter + minimal CLI) is simple. The following V1 components deserve explicit Gall's Law justification:

- Plugin system (see PLG-02 — justified only by git-context, could be native Go)
- CapabilityRouter (see INT-04 — V1 has Null + Anthropic; no routing decisions)
- Workflow versioning (one developer, one app; version management adds overhead)
- Compensation plans (does OIP actually use compensation?)
- RecallAPI + `awis recall` (FTS-only version is simpler and good enough)
- `parallel` StepType (zero use cases)

**If these six components were cut or simplified, V1 would be a clearly "simple system" that satisfies Gall's Law. With them, V1 is on the edge.**

---

## 22. YAGNI COMPLIANCE

You Aren't Gonna Need It analysis.

| Component | V1 Use Case | YAGNI Assessment |
|---|---|---|
| Plugin system (full lifecycle) | git-context-plugin for OIP | CONDITIONAL — justified only if plugin system is strategic V1 goal; could defer |
| CapabilityRouter (full) | Null+Anthropic routing | VIOLATION — V1 needs a 15-line function, not a configurable router |
| `parallel` StepType | None defined | VIOLATION — remove from V1 |
| Workflow semver versioning | One app, one developer | CONDITIONAL — simpler model suffices in V1 |
| `awis recall` (synthesis) | OIP recall workflow handles this | CONDITIONAL — FTS-mode recall is fine; synthesis is V2 |
| `classify` capability in IntelligencePort | No V1 use case | VIOLATION — remove from V1 interface |
| AuditLog (charter activations) | Charters are V3 | CONDITIONAL — audit log is fine; charter activation events are premature |
| CompensationPlan (complex) | OIP capture doesn't need it | CONDITIONAL — simple retry is sufficient for V1 OIP; compensation is V2 |

**YAGNI violations requiring correction (5):**
1. `parallel` StepType — remove
2. `classify` from IntelligencePort — remove from V1 interface
3. Full CapabilityRouter — simplify to V1 routing function
4. Synthesis-mode `awis recall` as default — FTS-first
5. Charter activation events in AuditLog — remove (charters don't exist in V1)

---

## 23. KISS COMPLIANCE

Keep It Simple analysis. (Simplest thing that could possibly work.)

**KISS-01 (SHOULD_FIX):** The expression language. The simplest expression model: only support variable interpolation (`{{steps.foo.outputs.bar}}`), no conditions in the template syntax, and no arithmetic. Conditions in transitions use a minimal predicate syntax with defined operators: `==`, `!=`, `>`, `<`, `>=`, `<=`, `&&`, `||`, `!`, and references to step status and outputs. Define the complete set upfront and enforce it. Never allow arbitrary code in YAML.

**KISS-02 (SHOULD_FIX):** The CapabilityRouter. For V1 with two adapters, the simplest router is: "iterate the fallback chain; return first adapter where `IsAvailable() && HasCapability(req.capability)`." No model hints, no per-capability overrides. Add these in V2 when a third adapter creates actual routing decisions.

**KISS-03 (SAFE_TO_DEFER):** The WorkflowTestHarness. The simplest V1 harness: runs one workflow synchronously, steps through one tick at a time, supports signal injection via a direct method call. Everything else (concurrent workflows in harness, replay, clock injection) is V2.

---

## 24. OVER-ENGINEERING ANALYSIS

Components that are more complex than their V1 use cases justify.

| Component | Complexity | V1 Justification | Verdict |
|---|---|---|---|
| Full CapabilityRouter | High | Two adapters, one routing decision | Over-engineered for V1 |
| Plugin system (full) | Very High | One plugin (git-context) | Borderline; requires explicit strategic justification |
| Semver workflow versioning | Medium | One developer, one application | Mild over-engineering |
| `parallel` StepType | High | No use cases | Over-engineered; remove |
| `classify` in IntelligencePort | Low | No V1 use case | Remove from V1 |
| Synthesis-mode `awis recall` | High | OIP has its own recall workflow | Over-engineered for V1 platform |
| Compensation plans (complex) | Medium | OIP capture may not need it | Mild over-engineering |

**The most important simplification: cut `parallel` StepType and `classify` from IntelligencePort.** These are zero-use-case abstractions in V1.

---

## 25. UNDER-ENGINEERING ANALYSIS

Components where the specification is insufficiently precise.

| Component | Gap | Risk |
|---|---|---|
| Expression language | No grammar defined | Inconsistent implementation |
| `parallel` StepType | Undefined | Impossible to implement correctly |
| Signal delivery atomicity | Not specified | Data inconsistency on crash |
| Cancellation semantics | Not specified | Non-deterministic behavior |
| OIP FTS5 index storage | Not specified | Requires architectural decision mid-implementation |
| Plugin idle behavior | "Paused" is ambiguous | Wrong implementation on target platform |
| Event trigger source | Not specified | Trigger type is unusable in V1 |
| `awis workflow show` output format | Not specified | Inconsistent terminal output |
| `InstanceFilter` struct | Not defined | SDK cannot be used without guessing |
| `WorkflowTestHarness` API completeness | Incomplete | Testing WAIT steps is unclear |

---

## 26. COMPLEXITY BUDGET ASSESSMENT

A reasonable complexity budget for a solo-founder V1: components that can be built, documented, and maintained by one developer with confidence.

**Complexity accounting:**

| Component | Complexity Units | Justified? |
|---|---|---|
| EventLog + StateStore (SQLite) | 3 | Yes |
| Pull-based execution loop | 2 | Yes |
| NativeRunner | 1 | Yes |
| SubprocessRunner | 2 | Yes |
| IntelligencePort + 2 adapters | 2 | Yes |
| YAML DSL + expression language | 3 | Expressible; grammar must be bounded |
| Go SDK (WorkflowBuilder, StepHandler) | 2 | Yes |
| WorkflowTestHarness | 2 | Yes |
| Signal protocol | 2 | Yes |
| CLI (15 commands) | 3 | Yes |
| Plugin system (full lifecycle) | 5 | Conditional |
| CapabilityRouter (full) | 3 | Replace with 1 |
| Workflow versioning | 2 | Conditional |
| Compensation plans | 2 | Conditional |
| `awis recall` with synthesis | 2 | Defer synthesis |
| `parallel` StepType | 2 | Delete |
| Structured metrics + audit log | 2 | Yes |

**Total with full V1 spec: ~38 units**
**After recommended cuts/simplifications: ~27 units**

27 units is a reasonable solo-founder V1. 38 units is ambitious and risks incomplete implementation under schedule pressure.

**Recommended V1 budget target: cut or simplify to ≤30 units.**

---

## 27. SOLO FOUNDER SUSTAINABILITY

**SFS-01 (SHOULD_FIX):** The plugin system is the highest-maintenance V1 component. It requires: a protocol implementation on both sides (runtime + plugin server), documentation for plugin developers, a reference implementation, and ongoing protocol versioning as the interface evolves. For a solo founder, this is 20-30% of V1 maintenance burden for one use case (git context assembly). Either commit to the plugin system as a strategic priority (explicit in the PRD) or defer it.

**SFS-02 (SHOULD_FIX):** The YAML DSL requires a parser, a validator, a documentation site for the DSL grammar, and ongoing maintenance as new step types and trigger types are added. Each new feature that touches step types or triggers requires updating the parser and validator. This is manageable but must be acknowledged as an ongoing cost in the PRD.

**SFS-03 (SAFE_TO_DEFER):** Metrics collection (`awis metrics` computing aggregates from EventLog) grows slower as EventLog grows. The current spec uses "computes on demand" as the V1 metrics strategy. This is fine for V1 (<100K events), but 1M+ events will require indexed aggregation. The PRD should note the scale at which `awis metrics` performance degrades.

**SFS-04 (SAFE_TO_DEFER):** The specification correctly identifies the solo-founder bus factor risk (R10 in the risk register). The mitigations (minimal core, clean Go, ADRs for all decisions) are appropriate. One addition: the PRD should include a "AWIS Conceptual Overview" section that describes the platform in 5 pages — the kind of document that allows a new contributor to understand the platform's structure in one hour. This reduces the bus-factor risk more than any technical decision.

---

## 28. SCALABILITY ASSESSMENT

**SCALE-01 (SAFE_TO_DEFER):** The EventLog grows at approximately `events_per_workflow × workflows_per_day`. For OIP at 5 workflows/day with 5 events each = 25 events/day = 9,125 events/year. SQLite handles this trivially. The first scalability concern appears around 1M events (~100K workflows), which OIP won't reach in V1. No V1 scalability action required.

**SCALE-02 (SAFE_TO_DEFER):** The pull-based execution loop queries the StateStore every 100ms. For 100 active workflow instances, this is 100 SQLite queries/second — easily handled. For 10,000 active instances, this becomes 1,000 queries/second — still manageable for SQLite but approaches its write concurrency limit. This is a V2 concern (server mode, Postgres).

**SCALE-03 (SHOULD_FIX):** The specification notes "smart tick rate adaptation" as a V2 improvement (ticking slower for waiting instances). However, this should be noted as a V1 design consideration — the tick scan should exclude `completed`, `failed`, `cancelled`, and `compensated` instances. The current scan description says `status = 'running'` — which is correct. But the signal scan also runs every 100ms regardless of how many instances are waiting. A `waiting` instance check on every 100ms tick is unnecessary; a 1-second interval for signal delivery would be fine. This is a V1 implementation note, not an architecture change.

---

## 29. MAINTAINABILITY ASSESSMENT

**MAINT-01:** The EventLog format is declared the "platform's irreversible artifact" with migration tooling required for schema changes. The format schema must therefore have `schema_version: 1` from day one. The specification defines `schema_version` implicitly (the `format` of events) but does not explicitly state the format version in the event schema. Add `schema_version: int` to the `ExecutionEvent` struct as a required field.

**MAINT-02:** The WorkflowDefinition schema is also potentially irreversible — once workflows are registered and instances are running against version 1.0.0, the definition schema must be backward-compatible or migration tooling is required. The spec's `format_version` field in the EventLog needs a counterpart in the WorkflowDefinition (there is already a semver `version` field, but that's the workflow's version, not the schema format version). This is subtly different: workflow `version` is "v1.0.0 of the capture-decision workflow"; format_version would be "version 1 of the WorkflowDefinition schema." The OIP Constitution handles this correctly (format_version: 1 in every entry). AWIS should do the same.

**MAINT-03:** CLI command backward compatibility is a maintenance contract the spec should acknowledge. Once `awis submit`, `awis trace`, and `awis signal` are shipped, changing their flags or output format is a breaking change for any scripts or CI integrations. The PRD should note: "CLI commands have a stability contract from V1.0.0 onward. Breaking changes require a major version bump of the AWIS binary."

---

## 30. SECURITY ASSESSMENT

**SEC-01 (SHOULD_FIX):** V1 has no authentication or authorization. This is explicitly stated and correct for local-first single-user deployment. However, the spec does not address the scenario where AWIS's local HTTP API (used in the embedded SDK's socket communication, if any) is accessible on `0.0.0.0` rather than `127.0.0.1`. The PRD must specify: "In V1 local mode, AWIS binds exclusively to 127.0.0.1. No remote access is supported or permitted."

**SEC-02 (SHOULD_FIX):** Secret management. The spec says "API keys and credentials are never stored in workflow definitions or the EventLog." But `workflow.inputs` are stored in `workflow_instances.variables` in the StateStore, which is in `runtime.db`. If a developer accidentally passes a secret as a workflow input, it is persisted in plaintext in the SQLite database. The PRD should add: "Secrets must not be passed as workflow inputs. Use environment variables and access them in step handlers directly. AWIS does not encrypt the StateStore in V1."

**SEC-03 (SHOULD_FIX):** The plugin system runs subprocess processes. The spec says "Plugin processes inherit a minimal environment (no platform secrets)." But "minimal environment" is not defined. Go's `os/exec.Cmd.Env` defaults to inheriting the parent process's environment, which includes the `ANTHROPIC_API_KEY` and any other secrets. The plugin spec must explicitly define which environment variables are passed to plugin subprocesses (recommendation: an explicit allowlist from the plugin manifest's `env` field, and nothing else).

**SEC-04 (SAFE_TO_DEFER):** Plugin manifests are trusted by default in V1 (installed from a path/URL specified by the developer). In V2 with a plugin marketplace, manifest signing and verification will be required. The spec should note this as a V2 prerequisite for the marketplace.

---

## 31. PERFORMANCE RISKS

**PERF-01 (SHOULD_FIX):** Plugin respawn latency. A Python plugin with dependencies (gitpython, PyGithub) takes approximately 2-4 seconds to spawn and import dependencies. The spec's `idle_timeout_s: 300` means after 5 minutes of inactivity, the plugin is killed. The next call after idle will see 2-4 second latency on a step that normally takes 1 second. For OIP's capture workflow, this means the first step after any idle period has a 2-4 second overhead. This should be acknowledged in the PRD as a known performance characteristic, not a bug.

**PERF-02 (SHOULD_FIX):** In-process cosine similarity for semantic search. The AWIS Blueprint inherits OIP's "in-process cosine similarity over Float32Array BLOBs" model. The upgrade trigger was defined as "P75 > 500ms or >50K entries." But this threshold check is never specified as a monitoring metric. The `awis metrics` output should include embedding search latency so the developer knows when they're approaching the threshold.

**PERF-03 (SAFE_TO_DEFER):** The pull-based loop's 100ms tick rate means maximum step initiation latency is 100ms from trigger to first dispatch. For NeuroDashboard with real-time health monitoring, 100ms initiation latency on a health alert workflow may be acceptable (health data processing is not inherently real-time). But if a future application requires <10ms initiation, the pull model would need to be supplemented with a push channel. Document this ceiling explicitly.

---

## 32. ADOPTION RISKS

**ADO-01 (SHOULD_FIX):** *The "learn AWIS before building your application" barrier.* AWIS's value is in building applications on it. But developers must understand AWIS's primitives (Step, WorkflowDefinition, IntelligencePort) before they can use them. The learning curve is longer than for tools that embed these concepts in a familiar framework. The PRD should address this with: a concrete end-to-end example of OIP's simplest workflow built on AWIS (showing the actual code, not just the architecture). This is a documentation commitment, not an architecture change.

**ADO-02 (SHOULD_FIX):** *The "why not just write the code directly?" objection.* For a simple 3-step workflow, using AWIS requires more code than a direct implementation: define WorkflowDefinition, implement StepHandler, register at startup, submit via WorkflowRunner — versus just calling the three functions directly in sequence. The PRD needs to address this objection explicitly: "For a workflow that runs once and never fails, direct code is simpler. AWIS adds value when: the workflow may fail and retry, you need to audit what happened, you need to add AI integration, or the same pattern repeats across multiple applications."

**ADO-03 (SAFE_TO_DEFER):** *Single-language SDK limitation.* V1's Go SDK is the only officially supported SDK. Developers building applications in Python or TypeScript cannot use the Go SDK. They can define workflows in YAML and implement step handlers as subprocesses — but the full SDK experience (WorkflowBuilder, WorkflowTestHarness) is Go-only. The PRD should acknowledge this limitation and the V2 plan for Python and TypeScript SDKs.

---

## 33. BUSINESS RISKS

**BIZ-01 (SHOULD_FIX):** *Competitive positioning is unclear in the spec.* The Product Investigation's "not this" table correctly rejects Zapier/Make/n8n comparisons. But it doesn't address the most direct comparison: "Why not use Temporal?" Temporal is the closest architectural precedent (durable execution, event sourcing, SDK-first). The PRD needs a one-paragraph "AWIS vs. Temporal" comparison: Temporal requires a coordination cluster (3+ processes); AWIS runs as a single binary with SQLite. Temporal's complexity ceiling is higher; AWIS's operational complexity floor is lower. For a solo founder building a portfolio of applications, Temporal's infrastructure overhead is prohibitive. AWIS is the right choice in this specific context.

**BIZ-02 (SAFE_TO_DEFER):** *Monetization path is not defined.* The Constitution mentions "commercial strategy" and "retention through exit rights." The PRD for V1 intentionally does not address monetization. This is correct — V1 validates the product, not the business. But the PRD should at least note: "V1 is open-source or source-available. Monetization model is deferred to post-V1 evidence."

**BIZ-03 (SAFE_TO_DEFER):** *AWIS acronym is undefined.* None of the three foundational documents define what AWIS stands for. This creates marketing and communication friction. The PRD should either define the acronym or retire it in favor of a descriptive name. Suggestion: if it does stand for something, define it; if it's just a name, say so.

---

## 34. COMPETITIVE ANALYSIS

**Genuine differentiation (validated by adversarial review):**

1. **Zero infrastructure locally.** No existing workflow tool — Temporal, Prefect, Dagster, n8n, Kestra — runs as a single binary with zero external dependencies in full-featured mode. Temporal requires 3+ processes; Airflow requires Postgres + Redis + workers; n8n requires Node.js + database. AWIS's `awis start` + SQLite is a genuine differentiator.

2. **Complete execution history as first-class product.** `awis trace` with full step-by-step timeline including intelligence adapter usage is not available in any comparable tool. Temporal has replay but not the observability-first UX. Prefect has Flow Runs but they require the cloud. AWIS's local EventLog + trace is genuinely novel for the local-first use case.

3. **Intelligence as a declared step capability with structured fallback.** LangGraph couples AI to its execution model. n8n has AI nodes but no structured fallback. AWIS's `required: false` + fallback step is a clean architectural pattern that no competitor has formalized.

**Where competition is stronger:**

1. **Ecosystem.** n8n has 400+ integrations. Temporal has SDKs in 7 languages. AWIS has one plugin (to be built) and one SDK language. This gap is a 3-5 year problem, not a V1 problem.

2. **Visual tooling.** n8n, Camunda, and Temporal's UI are mature. AWIS's CLI-first V1 will lose any "visual" comparison. The PRD should not attempt to compare visual tooling.

3. **Managed service.** Temporal Cloud, Prefect Cloud, and n8n Cloud offer managed infrastructure. AWIS has no equivalent in V1-V3. The local-first positioning is a feature, not a gap — but some potential users want managed.

**Conclusion:** AWIS's differentiation is real and defensible. The product investigation correctly identifies the competitive moat as zero-infrastructure local execution + complete debuggable history + AI-optional architecture. No major correction needed.

---

## 35. TECHNICAL DEBT FORECAST

**TD-01:** The OIP Engineering Blueprint will become technical debt immediately if a developer reads it expecting it to describe the current OIP design. It must be explicitly deprecated (see CON-01).

**TD-02:** The expression language, if not formally specified before implementation, will accumulate technical debt as each developer's "reasonable interpretation" diverges from another's. The PRD's grammar specification is the debt-prevention investment.

**TD-03:** The CapabilityRouter, if built to its full V2+ spec in V1, will be complex machinery with no routing decisions to make until V2. The unused complexity will tempt future developers to add features "since it's already there" — the usual over-engineering trap.

**TD-04:** The `parallel` StepType, if left undefined and kept in the type enum, will either be partially implemented (creating inconsistent behavior) or never implemented (creating a misleading type constant that generates questions). Either outcome is technical debt.

---

## 36. SIMPLIFICATION OPPORTUNITIES

| Opportunity | Change | Risk | Benefit |
|---|---|---|---|
| Remove `parallel` StepType from V1 | Delete from type enum | None | Removes undefined behavior |
| Remove `classify` from IntelligencePort | Delete from interface | None | Simpler interface (4→3 methods) |
| Simplify CapabilityRouter to V1 function | 15-line function replaces full router | Low | Reduces complexity by ~3 units |
| Implement git-context as Go native handler | Replaces full plugin system | Medium (defers ecosystem) | Reduces V1 complexity by ~4 units |
| Simplify `awis recall` to FTS-only in V1 | Remove synthesis; add `--synthesize` flag | Low | Satisfies P2 invariant; simpler default |
| Defer workflow versioning overhead | Remove semver requirement; V1 uses monotonic counter or timestamp | Low | Simpler registry and UI |
| Define expression language boundary | Formal grammar + explicit unsupported list | None | Prevents scope creep |
| Merge OIP Blueprint into AWIS Blueprint | Single OIP architecture document | None | Eliminates document schism |

---

## 37. REQUIRED PRD CORRECTIONS

The following changes must be made before the canonical PRD is generated.

**CORRECTION-01:** Explicitly supersede `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` (OIP standalone design) with the AWIS-application model. Add to PRD: "OIP is implemented as an AWIS application. The standalone OIP engineering blueprint (ENGINEERING_ARCHITECTURE_BLUEPRINT.md) is archived as a historical reference. The AWIS_ARCHITECTURE_BLUEPRINT.md governs OIP's implementation."

**CORRECTION-02:** Define OIP's FTS5 index ownership. Choose Option A (OIP owns its own SQLite, separate from AWIS runtime.db) or Option B (OIP uses AWIS's runtime.db with namespace-prefixed virtual tables). **Recommendation: Option A** — cleaner boundary, consistent with OIP Engineering Blueprint, simpler StoragePort interface.

**CORRECTION-03:** Define the expression language grammar. One page: supported template syntax, supported condition operators, scope (what variables are available), null/missing handling, and unsupported features (explicit list of what will NOT be added).

**CORRECTION-04:** Define signal delivery atomicity. Add to the architecture spec: "Signal delivery is atomic within a single SQLite transaction covering signal_inbox, execution_events, and workflow_instances tables. In local mode, all three tables reside in a single runtime.db file. The EventLog is authoritative; any StateStore inconsistency is recoverable via `awis rebuild-state`."

**CORRECTION-05:** Define `awis cancel` semantics. Add to the CLI specification: in-flight step behavior, compensation policy, signal inbox cleanup, idempotency on completed/failed workflows.

**CORRECTION-06:** Remove `parallel` StepType from V1 or define it completely. **Recommendation: remove from V1 StepType enum.** Parallel execution via multiple transitions + join-point step achieves the same result.

**CORRECTION-07:** Fix the compensation example. Replace `undo: oip.record.mark-draft` with constitutionally-correct behavior: either remove the compensation plan from `append-to-record` (the most reasonable choice — a successfully appended Record entry should not be undone) or replace with `undo: oip.record.void-entry` and document that voiding appends a superseding entry, not modifying the original.

---

## 38. CRITICAL BLOCKERS

Issues that will cause implementation failure or PRD self-contradiction if not resolved.

**BLOCKER-01 — Document Schism (CON-01)**
*Finding:* Two incompatible OIP architectures coexist.
*Evidence:* ENGINEERING_ARCHITECTURE_BLUEPRINT.md says "single binary, no server"; AWIS_ARCHITECTURE_BLUEPRINT.md says "OIP is an AWIS application."
*Root cause:* AWIS Blueprint was produced without explicitly superseding the OIP Blueprint.
*Impact:* Implementation will be blocked at week 5 when the developer must choose between two incompatible designs.
*Severity:* MUST_FIX_BEFORE_PRD
*Correction:* Add supersession statement to PRD; deprecate OIP standalone blueprint.

**BLOCKER-02 — Expression Language Undefined (ARCH-ISSUE-02)**
*Finding:* YAML DSL uses two distinct expression syntaxes with no formal grammar.
*Evidence:* `{{...}}` for templates; `"steps['draft-entry'].status == 'fallback'"` for conditions; inconsistent bracket/dot notation.
*Root cause:* DSL designed by example without formal specification.
*Impact:* Inconsistent parser implementation; invalid YAML deemed valid or vice versa; documentation impossible.
*Severity:* MUST_FIX_BEFORE_PRD
*Correction:* Write a formal one-page grammar before PRD.

**BLOCKER-03 — Signal Atomicity Unspecified (ARCH-ISSUE-03)**
*Finding:* Crash between signal_inbox persistence and StateStore update has undefined behavior.
*Evidence:* Signal delivery requires writing to three tables (signal_inbox, execution_events, workflow_instances) with no stated atomicity guarantee.
*Root cause:* Specification assumes single-database SQLite atomicity without stating it.
*Impact:* Inconsistent signal delivery on crash; duplicate signals on recovery; silent data loss.
*Severity:* MUST_FIX_BEFORE_PRD
*Correction:* State "Signal delivery is a single SQLite transaction across all three tables."

**BLOCKER-04 — Cancellation Semantics Undefined (ARCH-ISSUE-04)**
*Finding:* `awis cancel` behavior is not defined.
*Evidence:* The command appears in the CLI contract; `cancelled` appears in InstanceStatus; no behavior description exists.
*Root cause:* Cancellation was included in the CLI scope without defining its semantics.
*Impact:* Inconsistent implementation; two developers implement differently; breaking behavior change post-PRD.
*Severity:* MUST_FIX_BEFORE_PRD
*Correction:* Define cancellation semantics: step completion on cancel, compensation policy, signal inbox cleanup.

**BLOCKER-05 — OIP FTS5 Ownership Ambiguous (CON-01 corollary)**
*Finding:* OIP's FTS5 index lives in either AWIS's runtime.db or OIP's own SQLite — unspecified.
*Evidence:* OIP Engineering Blueprint: `.decisions/.index/index.db`. AWIS Blueprint: "OIP namespace" in AWIS runtime.
*Root cause:* Document schism; OIP's index was redesigned without resolving the storage question.
*Impact:* StoragePort interface scope is wrong if OIP uses AWIS's SQLite; deployment story is wrong if OIP has its own SQLite.
*Severity:* MUST_FIX_BEFORE_PRD
*Correction:* Choose Option A (OIP owns its own index.db); document it explicitly.

**BLOCKER-06 — `parallel` StepType Undefined (ARCH-ISSUE-01)**
*Finding:* `parallel` is in the StepType enum with zero definition.
*Evidence:* AWIS Blueprint §6: `StepType: native | subprocess | plugin | intelligence | signal | parallel` with no parallel step description.
*Root cause:* Type was included in the enum without completing the specification.
*Impact:* The validator and executor must handle `parallel` steps; undefined behavior on encounter.
*Severity:* MUST_FIX_BEFORE_PRD
*Correction:* Remove from V1 enum.

**BLOCKER-07 — Compensation Violates Append-Only Invariant (CON-02)**
*Finding:* The canonical compensation example (`oip.record.mark-draft`) violates the constitutionally mandated append-only Record invariant.
*Evidence:* OIP Constitution Article 7: entries may never be modified. AWIS Blueprint compensation example shows modifying a Record entry.
*Root cause:* Compensation example was written without checking it against the Constitution.
*Impact:* If implemented as written, the OIP Record is mutable in violation of the platform's governing principle.
*Severity:* MUST_FIX_BEFORE_PRD
*Correction:* Remove `append-to-record` from the compensation plan. Successful record appends are not undoable; this is constitutionally correct.

---

## 39. NON-CRITICAL IMPROVEMENTS

Improvements that would strengthen the specification but do not block PRD generation.

| ID | Finding | Severity | Benefit |
|---|---|---|---|
| NCI-01 | Remove `classify` from V1 IntelligencePort | SHOULD_FIX | Simpler interface; YAGNI compliant |
| NCI-02 | Simplify CapabilityRouter to V1 function | SHOULD_FIX | Reduces complexity; YAGNI compliant |
| NCI-03 | Define plugin idle behavior (killed vs suspended) | SHOULD_FIX | Prevents wrong platform assumption |
| NCI-04 | Define event trigger source mechanism for V1 | SHOULD_FIX | Makes event triggers usable or removes them |
| NCI-05 | Reconcile V1 implementation timeline (6 vs 8 weeks) | SHOULD_FIX | Prevents false schedule commitment |
| NCI-06 | Justify plugin system in V1 or defer to V2 | SHOULD_FIX | Reduces complexity budget if deferred |
| NCI-07 | Change context_budget enforcement to adapter-level | SHOULD_FIX | Avoids model-specific tokenizer in platform |
| NCI-08 | Make `awis recall` FTS-first in V1 | SHOULD_FIX | Satisfies P2 invariant |
| NCI-09 | Add `awis workflow show` to Architecture CLI spec | SHOULD_FIX | Consistency between product and architecture docs |
| NCI-10 | Define `InstanceFilter` struct in SDK spec | SHOULD_FIX | SDK usable without guessing |
| NCI-11 | Add `--input=@file` support to `awis submit` | SHOULD_FIX | UX improvement for complex inputs |
| NCI-12 | Define validation modes (offline vs. with-runtime) | SHOULD_FIX | Resolves CON-05 |
| NCI-13 | Add V1 deployment model clarification (embedded vs. standalone) | SHOULD_FIX | Resolves HA-04 |
| NCI-14 | Add `schema_version` to ExecutionEvent struct | SHOULD_FIX | Enables future EventLog format migration |
| NCI-15 | Clarify secret injection exclusion from workflow variables | SHOULD_FIX | Security hygiene |
| NCI-16 | Define plugin environment variable allowlist | SHOULD_FIX | Prevents secret leakage to plugins |
| NCI-17 | Note P75 embedding search latency as a tracked metric | SAFE_TO_DEFER | Enables data-driven upgrade decision |
| NCI-18 | Define AWIS acronym or retire it | SAFE_TO_DEFER | Brand clarity |
| NCI-19 | Note plugin respawn latency as known performance characteristic | SAFE_TO_DEFER | Accurate developer expectations |
| NCI-20 | Add "AWIS vs. Temporal" comparison to PRD | SAFE_TO_DEFER | Competitive clarity |

---

## 40. CONFIDENCE ASSESSMENT

| Finding Category | Confidence | Notes |
|---|---|---|
| BLOCKER-01 (Document Schism) | Very High | Directly observed; two contradictory documents |
| BLOCKER-02 (Expression Language) | Very High | Two syntaxes visible in same document; no grammar specified |
| BLOCKER-03 (Signal Atomicity) | High | Absence of specification is observable; SQLite fix is well-known |
| BLOCKER-04 (Cancellation) | Very High | `awis cancel` exists in CLI with no semantics defined |
| BLOCKER-05 (FTS5 Ownership) | High | Two contradictory storage descriptions; correct option is clear |
| BLOCKER-06 (`parallel` StepType) | Very High | Listed in enum with zero definition |
| BLOCKER-07 (Compensation × Append-Only) | Very High | Constitutional conflict is direct and observable |
| Architecture overall soundness | High | Pull-based, event-sourced, port-based is proven pattern |
| Intelligence layer design | High | IntelligencePort is well-designed and minimal |
| Solo-founder sustainability | Medium | Complexity assessment is estimate; actual burden depends on implementation skill |
| Plugin system V1 justification | Medium | Strategic call; both paths (include/defer) are defensible |
| Product experience quality | High | `awis trace` as hero command is correct; CLI design is clean |

---

## 41. FINAL VERDICT

### Evidence Summary

**What the specification gets right (and gets right firmly):**
- The pull-based execution loop over an append-only EventLog is proven, well-justified, and correctly specified
- The IntelligencePort seam with NullAdapter as default is the specification's best single design decision
- SQLite for local-first, Postgres-compatible for cloud, behind a StoragePort abstraction is correct
- CLI-first V1 with `awis trace` as the hero command is the right product decision
- OIP as the first application validation gate is exactly the right development discipline
- Namespace isolation design is forward-compatible without over-engineering
- The two-tier DSL (YAML + Go SDK, same runtime) is correctly motivated

**What the specification gets wrong (and must fix):**
Seven issues, all specification gaps or consistency failures, none requiring architectural redesign:
1. Two incompatible OIP architecture documents coexist without supersession notice
2. YAML DSL uses two expression syntaxes with no formal grammar
3. Signal delivery atomicity is unspecified
4. `awis cancel` semantics are undefined
5. OIP's FTS5 index storage ownership is ambiguous
6. `parallel` StepType is in the enum with zero definition
7. The compensation example violates the constitutionally protected append-only invariant

### The Verdict

**ADDITIONAL_ARCHITECTURAL_WORK_REQUIRED**

Not because the architecture is wrong — it is substantially correct and well-reasoned. Because the seven blockers listed above will force implementation decisions under time pressure that should have been made during specification. A PRD generated from the current specification will either inherit these gaps (propagating them into implementation) or require the implementing engineer to make architectural decisions without authority or specification guidance.

The corrective work is limited in scope:
- One specification document explicitly superseding the OIP standalone blueprint
- One-page expression language grammar
- Three paragraphs defining atomicity, cancellation, and FTS5 storage
- Two enum/interface simplifications (`parallel` removal, compensation fix)

**Estimated correction effort:** One focused day for an engineer who has read all three documents.

**After correction:** READY_FOR_CANONICAL_PRD.

The architecture has earned implementation. The specification has not yet earned a PRD.

---

*Produced by Canonical AWIS Specification Tribunal v2.0 — 2026-07-02*
*Documents reviewed: 4 | Findings: 7 blockers, 20 non-critical improvements, 10 validations*
*Verdict: ADDITIONAL_ARCHITECTURAL_WORK_REQUIRED*
*Condition for reversal: Resolve all 7 MUST_FIX findings*
