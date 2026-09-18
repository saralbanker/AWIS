# AWIS — RECONSTRUCTION ANSWER SHEET (CAVEMAN NOTES)

Answers to the reconstruction question list. Terse on purpose.
Every line tagged **VERIFIED** (repo doc/code evidence, src cited) / **INFERRED** / **UNKNOWN**.
Compiled 2026-09-06 against `engine-hardening` @ `8a87f70`.

Rule used: **code beats doc.** Where docs claim something code does not do, said so.

---

## SECTION 0 — REPO STATE & DOCUMENT META

### Q: Current repo state vs the reconstruction doc? Which commit/branch authoritative?
- Working branch = `engine-hardening`, HEAD `8a87f70` (2026-09-05). VERIFIED [git log]
- **`engine-hardening` is the de-facto authoritative line, NOT `main`.** 62 commits ahead of `main`, 6 behind. VERIFIED [git rev-list --left-right main...engine-hardening = 6 62]
- `main` HEAD `98350f6` (2026-08-20, "ci: add macOS to verify matrix"). `main` contains M00–M14 only. VERIFIED
- Branches M10..M17 were built STACKED, never squash-merged to main (founder-only E-MERGE gate); `engine-hardening` contains all of them as ancestors. VERIFIED [git merge-base --is-ancestor m10/m14/m15/m17 engine-hardening = true]
- Merge-base(main, engine-hardening) = `827a084` (M14 D-CLOSE). VERIFIED
- Milestone tags exist: M02..M09 + M14. VERIFIED [git tag]
- Ledger truth: `docs/05-implementation/STATE.md` — M00–M09 DONE-MILESTONES; M10,M11,M12,M13,M14,M15,M16 all `PHASE: E-MERGE (blocked on founder)`; **M17 = `C-VERIFY`, fresh M17-V1 re-run required**; M18-hardening-release not started. VERIFIED [STATE.md:6-130,tail]
- **Tree is DIRTY: 339 changed paths** (89 added, 220 deleted, 6 modified, 24 untracked). VERIFIED [git status --porcelain]
- **CRITICAL: the entire Beta GUI/API deliverable is UNTRACKED.** `internal/api/`, `cmd/awis-server/`, `internal/buildinfo/`, `web/` have zero tracked files → a clean clone cannot build `awis-server`; CI has never built or tested that code. VERIFIED [git status; FINAL_RELEASE_VERDICT.md:14-20; ARCHITECTURAL_DEBT_REGISTER.md AD-01]
- 15 root-level audit reports are also untracked (RELEASE_*, FINAL_*, SCALABILITY_ASSESSMENT, debt registers…). VERIFIED
- Practical answer: **no commit is fully authoritative right now.** Authoritative *code* = `engine-hardening`@`8a87f70` **plus the uncommitted working tree**. That gap is the #1 open defect (AD-01, timing "Now").

### Q: Is the reconstruction doc a complete PDR or only opening section?
- **UNKNOWN.** String "PDR" appears NOWHERE in the repo (grep -ri, 0 hits). The reconstruction document is external to this repo; repo cannot answer its scope.
- Repo's own equivalent = `AWIS_PRD.md` (2574 lines, FROZEN tier-0, self-describes as complete canonical PRD v1.0). If the reconstruction is measured against anything, measure against that. VERIFIED

### Q: Which statements are verified by repo evidence vs inferred?
- Per-answer tags below. Broad shape: engine/storage/CLI/plugin answers are **code-VERIFIED**; vision/personas/roadmap/licensing answers are **doc-VERIFIED only** (no code can confirm intent); monetization, licensing, compliance, DR are **UNKNOWN/absent**.
- Known doc-vs-code drift is tracked in-repo: `DOCUMENT_DRIFT_REPORT.md`, `DOCUMENTATION_DIVERGENCE_REPORT.md`. VERIFIED

### Q: What reports/audits exist besides those referenced?
- Root (untracked audit layer): ENGINE_READINESS_SCORECARD, FINAL_RELEASE_VERDICT, FINAL_VERDICT, FINAL_EXECUTIVE_SUMMARY, RELEASE_AUDIT_REPORT, RELEASE_BLOCKERS, RELEASE_CANDIDATE_AUDIT, RELEASE_CANDIDATE_REMEDIATION_REPORT, RELEASE_READINESS_REPORT, OPERATIONAL_READINESS_REVIEW, SCALABILITY_ASSESSMENT, ARCHITECTURAL_DEBT_REGISTER, DEFERRED_TECHNICAL_DEBT, VERIFIED_DEFECT_REGISTER, VERIFIED_GEMINI_FINDINGS, PHASE2_BLOCKERS, REGRESSION_REPORT, REPOSITORY_HEALTH_REPORT, DOCUMENT_DRIFT_REPORT, DOCUMENTATION_DIVERGENCE_REPORT, IMPLEMENTATION_REPORT, AWIS_CANONICAL_SPECIFICATION_TRIBUNAL_REPORT, PRD_RESEARCH_REPORT, IMPLEMENTATION_MASTER_PLAN_VERIFICATION_REPORT. VERIFIED [ls]
- `docs/08-engine-hardening/` — 01-PRE-REVIEW-REPORT, 02-POST-REVIEW-REPORT, ENGINE_HARDENING_PLAN, ENGINE_FREEZE_REPORT. VERIFIED
- `docs/09-gui-planning/` — 39 GUI docs (GUI_PRD, GUI_BETA_PRD, GUI_ARCHITECTURE, GUI_ROADMAP, GUI_PHASE1_*, REPOSITORY_TRUTH_AUDIT, …). UNTRACKED. VERIFIED
- `docs/10-release-candidate-audit/`, `docs/11-intelligence-architecture/` (8 docs), `docs/12-intelligence-architecture-decision/` (7 docs incl. ARCHITECTURE_DECISION_RECORD). VERIFIED

### Q: Formal system architecture diagram / doc that supersedes a reconstruction?
- YES. Canonical authority chain, conflicts resolve UPWARD:
  `OIP_CONSTITUTION.md` → `AWIS_ARCHITECTURE_FINALIZATION.md` → `AWIS_ARCHITECTURE_BLUEPRINT.md` → `AWIS_PRD.md` → `IMPLEMENTATION_MASTER_PLAN.md` (+ VERIFICATION_REPORT amendments F-1..F-5) → `docs/` KB (views only).
  VERIFIED [docs/07-indices/canonical-reference-map.md; docs/README.md]
- Tier-0 docs are FROZEN. `archive/ENGINEERING_ARCHITECTURE_BLUEPRINT.md` = SUPERSEDED, "never cite". VERIFIED
- Any external reconstruction is **subordinate** to all of the above.

### Q: Engine maturity right now (post-hardening)?
- `docs/08-engine-hardening/ENGINE_FREEZE_REPORT.md`: 11 CRITICAL + 12 IMPORTANT defects → **0 open**; 0 → **21 binary-level integration tests**; `make verify` + `make integration` both PASS. Verdict "engine ready to freeze". VERIFIED
- `FINAL_RELEASE_VERDICT.md` (2026-09-05, supersedes FINAL_VERDICT.md): **CONDITIONAL PASS** — "engine is Beta-quality, its release envelope is not". Blocking: uncommitted deliverable + HTTP API has no auth/CORS/rate-limit/body-limit/timeouts. VERIFIED
- `ENGINE_READINESS_SCORECARD.md` (2026-09-03, **PRE-hardening — largely superseded**): overall 3.9/10 NOT READY. Treat as historical; most listed defects were closed by `engine-hardening`. VERIFIED
- Architectural debt open: AD-01 no VCS history, AD-06 no migration downgrade guard, AD-07 no HTTP middleware seam (all "Now"); AD-02 frozen StoragePort can't express bounded reads, AD-04 FTS rebuilt per query, AD-05 durability undocumented (synchronous=NORMAL), AD-09 frontend types unenforced ("Next"); AD-03 single shared SQLite conn, AD-08 no mutation API ("Watch"). VERIFIED [ARCHITECTURAL_DEBT_REGISTER.md]

---

## SECTION A — PRODUCT: VISION, SCOPE, PERSONAS, TERMINOLOGY

### Q1: What does AWIS stand for?
- No expansion found. explicitly flagged as undefined issue. [src: AWIS_CANONICAL_SPECIFICATION_TRIBUNAL_REPORT.md:764]
- "NCI-18 | Define AWIS acronym or retire it | SAFE_TO_DEFER". [src: AWIS_CANONICAL_SPECIFICATION_TRIBUNAL_REPORT.md:922]
- glossary defines OIP = "Organizational Intelligence Platform" but NOT AWIS itself. [src: AWIS_PRD.md:2429-2503 §36]
- VERIFIED (explicitly undefined/flagged, not merely absent).

### Q2: Boundary workflow engine / orchestration runtime / automation platform / plugin ecosystem
- AWIS = "locally-hosted workflow execution platform with an optional intelligence layer" = runtime foundation, one sentence: "workflow engine developers never want to rebuild, made local, AI-optional, fully observable". [src: AWIS_PRD.md:99-105]
- explicitly NOT a no-code automation tool (Zapier/Make), NOT visual builder, NOT BPM/BPMN, NOT data pipeline tool, NOT process automation platform (UiPath). [src: AWIS_PRD.md:107-119 table]
- plugins = separate process-isolated capability providers via JSON-RPC over stdin/stdout, declared in awis-plugin.yaml; NOT the core engine, an add-on layer. [src: AWIS_PRD.md:319, 2429-2503 glossary "Plugin"]
- "orchestration runtime" term not used verbatim in PRD; product calls itself "workflow runtime" / "execution platform". [src: README.md:3, AWIS_PRD.md:99]
- VERIFIED.

### Q3: Primary user personas today
- Persona 1: The Platform Builder (sets up/maintains runtime). [src: AWIS_PRD.md:204]
- Persona 2: The Application Developer (defines workflows, writes handlers). [src: AWIS_PRD.md:221]
- Persona 3: The Plugin Developer (extends via JSON-RPC plugin). [src: AWIS_PRD.md:241]
- Persona 4: The End User (Indirect) — never touches AWIS directly. [src: AWIS_PRD.md:261]
- VERIFIED.

### Q4: Self-hosted / cloud / hybrid?
- V1: self-hosted only, single laptop, zero external services, no cloud. [src: AWIS_PRD.md:99-105, 196]
- V2: adds server mode (local HTTP API) + Postgres cloud storage adapter = hybrid path opens. [src: AWIS_PRD.md:2374-2401]
- V3+: multi-tenant workspaces w/ auth; V4+: SaaS deployment model w/ per-tenant BYOK. [src: AWIS_PRD.md:2402-2428]
- Permanent non-goal: requiring cloud connectivity — local-first always supported. [src: AWIS_PRD.md:196]
- VERIFIED: currently self-hosted-only; roadmap intends all three eventually.

### Q5: Maturity stage
- Docs call current state "AWIS Beta" / Release Candidate under audit; verdict says NOT release-ready as of that audit. [src: docs/10-release-candidate-audit/FINAL_VERDICT.md:1,5,12]
- Plan to tag `v0.1.0-beta` after fixing BETA-STABILIZATION-GATE issues. [src: docs/10-release-candidate-audit/FINAL_VERDICT.md:86]
- Later engine-hardening commits (RC-1..RC-5 fixes) verified post-audit (per git log 8a87f70) — hardening continuing past that audit.
- No "production" or "GA" label found anywhere.
- VERIFIED: Beta / release-candidate stage, not production.

### Q6: Explicit non-goals
V1 Non-Goals: [src: AWIS_PRD.md:164-186]
- NG-1 Visual workflow designer
- NG-2 Real-time team collaboration
- NG-3 AI model selection UI
- NG-4 Built-in authentication
- NG-5 Native mobile app
- NG-6 Prebuilt workflow template marketplace
- NG-7 Automatic workflow optimization
- NG-8 Enterprise governance (compliance/RBAC/SOC2)
- NG-9 Multi-worker execution
- NG-10 Postgres storage
Permanent Non-Goals (all versions): [src: AWIS_PRD.md:188-198]
- PNG-1 Owning business logic
- PNG-2 Replacing application databases
- PNG-3 Locking in intelligence providers
- PNG-4 Requiring cloud connectivity
- VERIFIED.

### Q7: Original problem statement
Four user problems drive AWIS: [src: AWIS_PRD.md:271-304]
- Problem 1: Repeated Infrastructure Tax — every product rebuilds step exec/retry/state/AI/history; costs 2-4wk per app, 10-20wk for 5-app portfolio.
- Problem 2: Opaque Workflow Failures — debugging async workflows via scattered logs; 5min diagnosis becomes 2+ hrs.
- Problem 3: Fragile AI Integration — hardcoded provider integration breaks on API/outage changes.
- Problem 4: Untestable Async Workflows — can't test state transitions deterministically without live infra.
- VERIFIED.

### Q8: Competitors/benchmarks considered
- Research drew on n8n, Temporal, GitHub Actions, Raycast, Linear, Supabase, Figma — "extracting principles, not features". [src: AWIS_PRODUCT_REQUIREMENTS_INVESTIGATION.md:14]
- Explicit contrast table: Zapier/Make (no-code, ruled out), Airflow/Dagster (data pipeline, ruled out), UiPath/Automation Anywhere (business process automation, ruled out), BPMN systems ("enterprise-workflow theater"). [src: AWIS_PRD.md:110-119]
- Temporal referenced re: execution engine maturity: "NONE in V1; ADOPT in V3 if execution is earned... Temporal-class engines are mature — rebuilding one would be constitutional vandalism." [src: PRD_RESEARCH_REPORT.md:260]
- Windmill, Node-RED, Prefect, LangGraph: not mentioned anywhere in repo (grep clean).
- VERIFIED for n8n/Temporal/Airflow/Dagster/UiPath/Zapier/Make/BPMN; UNKNOWN/not-considered for Windmill/Node-RED/Prefect/LangGraph.

### Q9: Is multi-tenancy required?
- Not in V1/V2 — V1 is single namespace, V2 is multi-namespace (not multi-tenant) on one runtime/one operator. [src: AWIS_PRD.md:1615-1673]
- Multi-tenant workspaces (with auth, isolated storage, quotas) explicitly deferred to V3 ("Out of scope for V1 and V2"). [src: AWIS_PRD.md:1674-1678]
- V3 quality gate + V4 SaaS w/ per-tenant BYOK envisioned later. [src: AWIS_PRD.md:2402-2428]
- VERIFIED: not required now; planned for V3+, not permanent requirement of V1/V2.

### Q10: Licensing model planned
- No LICENSE file in repo root (find returned no matches).
- No mention of MIT/Apache/GPL/license terms anywhere in PRD/architecture docs (grep clean).
- UNKNOWN.

### Q11: Monetization model envisioned
- V1-V3: none described (developer infra tool, no pricing/revenue language found).
- V4+: "Public plugin marketplace as discovery and monetization surface" is the only monetization mention. [src: AWIS_PRD.md:2419-2428]
- V4+: "SaaS deployment model with per-tenant BYOK" implies eventual paid hosting, but no pricing model stated.
- VERIFIED (partial): only V4+ plugin marketplace named as monetization surface; no concrete pricing/monetization plan otherwise = UNKNOWN beyond that.

### Q12: Roadmap milestones beyond "n8n-class Automation Platform"
Note: phrase "n8n-class Automation Platform" not found verbatim in repo (grep clean) — treating as roadmap-beyond-V1 question.
- V2 "Multi-Application" (Months 3-6): multi-namespace, server mode/HTTP API, Postgres adapter, OpenAI+Ollama adapters, multi-provider routing, cost tracking, Prometheus metrics, OTel tracing, plugin registry, webhook triggers, web status page. [src: AWIS_PRD.md:2374-2401]
- V3 "Platform" (Months 7-18): full web dashboard w/ graph viz, multi-tenant workspaces+auth, visual workflow designer, Charter model (bounded machine authority), WASM sandboxing for plugins, adaptive intelligence routing, cross-namespace governed data grants, plugin marketplace, full audit log. [src: AWIS_PRD.md:2402-2418]
- V4+ "External Platform": public plugin marketplace, SDK as public API w/ versioning, web dashboard as primary onboarding, SaaS w/ BYOK. [src: AWIS_PRD.md:2419-2427]
- Decade Vision (by 2031): "substrate layer for a portfolio of software products, the way PostgreSQL is the substrate for web applications." [src: AWIS_PRD.md:91-95]
- VERIFIED.

### Q13: Scalability targets. Availability targets.
PRD NFR-Performance targets (§11): [src: AWIS_PRD.md:541-551]
- Execution loop tick latency <5ms P99
- Native step dispatch <10ms P95
- Intelligence step (Anthropic Haiku draft) <5s P95
- Signal delivery latency <200ms
- awis trace query (100K events) <500ms
- awis rebuild-state (100K events) <30s
- Workflow instance startup latency <300ms
- Plugin spawn (cold) <3s
- Plugin capability call latency (excl. handler) <50ms
PRD NFR-Reliability (§11, availability-adjacent, no formal uptime % given): [src: AWIS_PRD.md:555-565]
- Workflow completion rate (local, no AI/external) ≥99.5%
- EventLog durability across crash: no data loss for committed events (WAL mode)
- State reconstruction after rebuild-state: 100% identical to pre-crash
- Signal delivery: exactly-once
- Plugin crash recovery: auto-restart ≤3 attempts
- No formal "availability %" (e.g. 99.9% uptime) SLA found anywhere — single-laptop local-first model has no uptime SLA concept.
Measured (not target) scalability from SCALABILITY_ASSESSMENT.md (2026-09-05, HEAD 8a87f70): [src: SCALABILITY_ASSESSMENT.md:1-40]
- "comfortable to roughly 5,000 instances", "operationally painful past ~10,000"
- binding limit = O(n²) insertion sort in CLI status path (cmd/awis/status.go:408), not SQLite/engine/API
- awis status: 2K=0.08s, 5K=0.58s, 10K=1.33s, 20K=4.93s; sort-alone 40K=19.08s
- HTTP API properly paginated: 20K instances, GET /api/v1/instances = 0.19s
- VERIFIED for PRD NFR targets; VERIFIED (measured, not "target") for scalability ceiling; no formal availability % SLA = UNKNOWN/not applicable.

### Q14: Compliance requirements. Disaster-recovery expectations.
- Compliance: explicitly out of scope — "NG-8 Enterprise governance. Compliance reporting, RBAC, SOC 2 are V3+ concerns." [src: AWIS_PRD.md:182]
- No GDPR/HIPAA mention anywhere (grep clean).
- DR: no formal "disaster recovery"/RTO/RPO terms found in repo (grep clean).
- Closest DR-equivalent: EventLog durability (WAL mode, no data loss for committed events) + rebuild-state reconstructs StateStore 100% identical from EventLog; runtime.db is a "rebuildable cache". [src: AWIS_PRD.md:560-562, 947-966]
- Audit log requirement (not compliance-driven): AuditLog records WorkflowRegistered/PluginRegistered/ConfigChanged/SignalDelivered, never pruned without operator action. [src: AWIS_PRD.md:592-593 NFR-S-05]
- VERIFIED: no compliance requirements in V1/V2 (explicitly deferred to V3+); no formal DR/RTO/RPO plan, only crash-recovery/rebuild guarantees exist.

### Q15: Terminology definitions (glossary §36 + IA §14)
- workflow: not defined as standalone term; only "WorkflowDefinition" defined = "the stable data structure representing a workflow graph: steps, transitions, triggers, compensation." [src: AWIS_PRD.md §36]
- execution: NOT defined as standalone glossary term (only "ExecutionEvent" defined). [src: AWIS_PRD.md §36] — NOT DEFINED
- run: NOT defined (no glossary entry; "awis submit" creates a "WorkflowInstance", not a "run"). — NOT DEFINED
- event: "ExecutionEvent" defined = "an individual record in the EventLog representing one state change (WorkflowStarted, StepCompleted, SignalReceived, etc.)." Also "DomainEvent" defined = application-defined trigger event, distinct from ExecutionEvent. [src: AWIS_PRD.md §36]
- state: NOT defined as standalone term (only "StateStore" defined = "the SQLite table (workflow_instances) that holds current state... materialized projection of the EventLog"). [src: AWIS_PRD.md §36] — "state" itself NOT DEFINED
- snapshot: NOT DEFINED anywhere in glossary or IA §14 (grep of PRD glossary confirms absent).
- job: NOT DEFINED (no glossary entry; concept expressed via "WorkflowInstance"/"Step").
- task: NOT DEFINED as a formal term (glossary uses "Step" instead).
- node: NOT DEFINED as standalone glossary term; used informally in IA §14 as "steps[] (all steps / nodes)" i.e. synonym for Step, not separately defined. [src: AWIS_PRD.md:872]
- plugin: defined = "a process-isolated external capability provider communicating via JSON-RPC 2.0 over stdin/stdout. Declared in awis-plugin.yaml." [src: AWIS_PRD.md §36]
- runner: NOT defined as generic term; only concrete "PluginRunner" and "SubprocessRunner" defined. [src: AWIS_PRD.md §36] — generic "runner" NOT DEFINED
- provider: NOT defined as standalone glossary term (used loosely for intelligence adapters/providers; "Adapter" is the defined term: "a concrete implementation of IntelligencePort or StoragePort for a specific provider"). — "provider" itself NOT DEFINED, only "Adapter"
- project: NOT defined in glossary; described operationally in §23 ("V1: Single Project, Single Namespace", project = directory created by `awis init`). [src: AWIS_PRD.md:1613-1633] — no formal glossary def
- tenant: NOT DEFINED anywhere in glossary; "multi-tenant" used only descriptively re: V3 workspaces. — NOT DEFINED
- workspace: NOT defined in glossary §36; described in §23 as V3 concept = "multi-tenant boundaries: one workspace per team, with authentication, isolated storage, and separate intelligence quotas." [src: AWIS_PRD.md:1674-1678] — defined in prose (§23), not in glossary
- Namespace: (bonus, is defined) = "a string identifier that scopes all workflow definitions, instances, and events to a specific application." [src: AWIS_PRD.md §36]
- Terms explicitly NOT defined anywhere (glossary or IA): execution, run, state, snapshot, job, task, node, runner (generic), provider (generic), tenant, project (no formal def), workspace (no formal glossary def, only prose in §23).
- VERIFIED (glossary content checked directly; absence confirmed by full read of §36 lines 2429-2503).

### Q16: Canonical architecture doc set + authority order
- Canonical reference map file exists: docs/07-indices/canonical-reference-map.md — declares itself the authority index. [src: docs/07-indices/canonical-reference-map.md]
- Declared authority order (Tier 0, all FROZEN, resolves upward):
  1. OIP_CONSTITUTION.md — values/invariants/articles
  2. AWIS_ARCHITECTURE_BLUEPRINT.md — system architecture: layers, six core structures, schemas, plugin protocol
  3. AWIS_ARCHITECTURE_FINALIZATION.md — blocker resolutions 1-7 (grammars, signal atomicity, cancellation, FTS ownership)
  4. AWIS_PRD.md — requirements (FR-*/NFR-*), quality gates QG-1..5, CLI contract, 6-week envelope
- Tier 1 (ARCHIVAL): IMPLEMENTATION_MASTER_PLAN.md, IMPLEMENTATION_MASTER_PLAN_VERIFICATION_REPORT.md
- Tier 2 (LIVING): docs/ (this KB) — partitioned operational views, becomes normative once G-gated
- archive/ENGINEERING_ARCHITECTURE_BLUEPRINT.md explicitly SUPERSEDED, "never cite". [src: docs/07-indices/canonical-reference-map.md]
- Reading order for authority disputes: "Constitution → Blueprint → Finalization → PRD → IMP+Verification → KB." [src: docs/07-indices/canonical-reference-map.md]
- This doc set DOES supersede any external reconstruction — it is explicitly declared canonical/frozen at tier 0.
- VERIFIED.

---

## SECTION B — ENGINE: EVENTS, STATE, STORAGE, WORKFLOWS, EXPRESSIONS

### Q1: persisted event types (enumerate)
- 12 types, frozen consts. [src: internal/core/event.go:39-62]
- WorkflowStarted, StepStarted, StepCompleted, StepFailed, StepFallbackActivated, SignalReceived, WorkflowCompleted, WorkflowFailed, WorkflowCancelled, WorkflowCompensating, WorkflowCompensated, WorkflowCompensationFailed. VERIFIED (code)
- comment cites "12 enumerated event types (§2)" / "TDS-01 §2; Blueprint §9". VERIFIED (doc ref in code comment)

### Q2: event sourcing hard requirement or impl choice
- Blueprint states as design fact, not "may": "State is an event-sourced append-only log." [src: AWIS_ARCHITECTURE_BLUEPRINT.md:24]
- rebuild.go comment: "append-only source of truth is worse than a failed read" — treated as load-bearing invariant, not optional. [src: internal/storage/rebuild.go:73]
- No ADR/EDR proposes alternative to event sourcing; EDR-005/007/011 all assume it as given. VERIFIED (doc+code) — treated as hard architectural requirement, not a swappable impl choice.

### Q3: delivery guarantees
- Signal delivery: PRD states "Exactly-once delivery guaranteed" (NFR-R-04). [src: AWIS_PRD.md:562]
- Blueprint: signal delivery is single DB transaction across signal_inbox/execution_events/workflow_instances, idempotency guard `delivered_at IS NULL`, no-op if already delivered → idempotent (effectively exactly-once via dedup, i.e. at-least-once attempt + idempotent apply = exactly-once effect). [src: AWIS_ARCHITECTURE_BLUEPRINT.md:559-579]
- Step execution: idempotency key `instance_id+step_id+attempt_number`, StepResultCache prevents re-exec on retry — this is at-most-once-effect via cache, not a raw delivery guarantee. [src: AWIS_ARCHITECTURE_BLUEPRINT.md:226; AWIS_PRD.md:439]
- Cloud/Postgres mode (NOT IMPLEMENTED, see Q10): "at-most-once step execution via claim/version mechanism." [src: AWIS_ARCHITECTURE_BLUEPRINT.md:684]
- Local mode (SQLite, what actually runs): "serializable transactions; no concurrent writers." [src: AWIS_ARCHITECTURE_BLUEPRINT.md:683]
- No single global word "exactly-once/at-least-once/at-most-once/configurable" applies platform-wide — guarantee is per-subsystem (signals: exactly-once-effect; steps: idempotent-retry / at-most-once-effect via cache+claim). VERIFIED (doc), code ClaimStep confirms atomic claim exists (internal/core/ports.go:26).

### Q4: determinism level on state reconstruction
- RebuildState replays ALL events in sequence_num order per instance inside one transaction; deterministic, pure function of event log content (+ two documented non-evented exceptions, see Q5/Q16 below). [src: internal/storage/rebuild.go:24-60]
- Explicit non-evented gaps documented: `waiting` status (tick-time decision, no event) and `cancellation_requested` flag are NOT derivable from events alone; rebuild recovers them from wait_records table / pre-wipe snapshot respectively, not from the log. VERIFIED (code comment): "a pure event replay can never reconstruct it" [src: internal/storage/rebuild.go:38-44]
- So: deterministic given (event log + wait_records + pre-wipe snapshot), NOT deterministic from event log alone. VERIFIED (code)

### Q5: nondeterministic ops during replay
- No explicit "nondeterminism during replay" handling code found (no sandboxing of side effects, no replay-mode flag on step handlers). UNKNOWN/gap.
- The two known non-deterministic-from-events gaps (waiting status, cancellation_requested) are patched by reading auxiliary tables, not by re-executing anything nondeterministic. [src: internal/storage/rebuild.go:24-60] VERIFIED (code)
- Step retry backoff/delay is "tick-quantized, not sleep-exact" (EDR-011 §3) — timing is not replayed, only recorded outcomes are. [src: docs/edr/edr-011-engine-semantics.md §3]
- No evidence of re-running step handlers (e.g. intelligence calls) during replay — replay only reconstructs projection state from already-recorded event payloads, so handler nondeterminism is a non-issue by construction (outputs already frozen in StepCompleted payload). INFERRED (code: rebuild.go never calls StepHandler).

### Q6: replay = first-class runtime capability or debug/recovery only
- RebuildState doc comment: "a library function that predates the engine (IMP §26) and is safe to call at any time" — described as recovery/projection tool, not a runtime execution mode. [src: internal/storage/rebuild.go:3-6]
- EDR-011 §8: "V1 documents this recovery derivation without exercising multi-process restart" — replay/recovery path exists but is crash-recovery framing, not a first-class "replay a workflow" feature exposed to users. VERIFIED (doc)
- No CLI/API found in ports.go or code named "Replay" as a user-facing workflow op (only WorkflowRunner/RecallAPI). CONCLUSION: recovery/debug tool, not first-class runtime capability. VERIFIED (code, absence)

### Q7: canonical source of truth
- Event log (execution_events, append-only) is canonical; workflow_instances is a derived projection. rebuild.go comment: "append-only source of truth" = the EventLog. [src: internal/storage/rebuild.go:73]
- Two named exceptions are NOT sourced from the event log: `waiting` status and `cancellation_requested` are non-evented runtime facts (EDR-007 gap), sourced from wait_records / raw instance row instead. [src: internal/storage/rebuild.go:38-44] VERIFIED (code)
- WorkflowDefinition (workflow registry) is a separate source of truth for workflow *shape*, immutable once registered (see Q16), not overwritten by replay. VERIFIED (schema doc: docs/WORKFLOW_SCHEMA.md §1 `version` "immutable once registered")

### Q8: storage abstraction (interface + file)
- `StoragePort` interface, comment: "the persistence boundary for the runtime: EventLog, StateStore, WorkflowRegistry, and StepResultCache ... adapters (SQLite, Postgres) implement it identically." [src: internal/core/ports.go:8-38]
- Also narrower interfaces: `AuditAppender` (internal/storage/audit.go:25), `RecallStore` (internal/storage/recall.go:42), `PluginStore` (internal/storage/plugins.go:46). VERIFIED (code)

### Q9: storage backends that WORK today
- SQLite only. Driver: `modernc.org/sqlite` (pure-Go, registered as "sqlite" driver). [src: internal/storage/db.go:12; go.mod:9 `modernc.org/sqlite v1.53.0`]
- Concrete impl type `SQLiteStorage` implements StoragePort (RebuildState method on it, AppendEvent etc. in sqlite.go). [src: internal/storage/rebuild.go:68; internal/storage/sqlite.go]
- No other backend files exist (`ls internal/storage/*.go` shows only sqlite-related files, no postgres*.go). VERIFIED (code, absence)

### Q10: storage backends planned but NOT implemented
- Postgres. StoragePort doc explicitly names it as a planned adapter ("adapters (SQLite, Postgres) implement it identically") but zero Postgres code exists in internal/storage. [src: internal/core/ports.go:9; grep for postgres/mysql/badger/bolt in internal/storage/*.go = no hits]
- Blueprint: "Storage is SQLite locally, Postgres-compatible in the cloud." — aspirational/architectural, not implemented. [src: AWIS_ARCHITECTURE_BLUEPRINT.md:24]
- Blueprint also describes "V2 introduces an optional persistent event queue (SQLite-backed locally, Postgres queue in cloud)" — explicitly V2/future. [src: AWIS_ARCHITECTURE_BLUEPRINT.md:719]
- EDR-005 rationale explicitly frames current design choices as preserving "the adapter-agnostic contract ... stable for the Postgres V2 reuse path" — confirms Postgres is deferred to V2, not built. [src: docs/edr/edr-005-sequence-assignment.md]
- VERIFIED: doc says Postgres planned, code has none. Doc/code agree (doc marks it future, code has zero impl — no disagreement).

### Q11: persistence guarantees across restart/crash
- WAL mode: `PRAGMA journal_mode=WAL` applied on open. [src: internal/storage/db.go:109]
- `PRAGMA synchronous=NORMAL` (not FULL) — WAL+NORMAL is a durability tradeoff (small crash window loses last commits vs FULL, per SQLite semantics); code does not use FULL. [src: internal/storage/db.go:112] VERIFIED (code) — no explicit fsync call beyond what PRAGMA synchronous=NORMAL+WAL provides.
- `PRAGMA busy_timeout=5000`, `PRAGMA foreign_keys=ON` also set. [src: internal/storage/db.go:110-111]
- Claim/lease recovery: `ClaimStep` atomic claim in StoragePort (internal/core/ports.go:26); step_claims table wiped and NOT rebuilt from events on RebuildState ("step_claims remain wiped — they are runtime state, not history") — meaning claims do NOT survive a rebuild; they are re-derived by engine restart logic separately (in-memory pending set reconstructed from event log per EDR-011 §8). [src: internal/storage/rebuild.go step 3 comment; docs/edr/edr-011-engine-semantics.md §8]
- Wait recovery: `signal.go`'s `recoverWait` recovers live waits from durable `wait_records` table after restart. [src: internal/storage/rebuild.go:41-44 referencing signal.go recoverWait]
- Cancellation: crash mid-cancellation loses the request (rebuild resets cancellation_requested to 0); documented as a limitation requiring caller to re-issue Cancel. [src: docs/edr/edr-011-engine-semantics.md §8] VERIFIED (code+doc)

### Q12: workflow definition format(s)
- YAML DSL (internal/dsl, ymlWorkflowDef struct) and Go SDK, both producing the canonical in-memory `WorkflowDefinition` struct. [src: internal/dsl/dsl.go:35; docs/WORKFLOW_SCHEMA.md: "both the YAML DSL and the Go SDK produce instances of it, and the runtime operates exclusively on it"] VERIFIED (doc+code)

### Q13: formal schema for workflows
- Yes: docs/WORKFLOW_SCHEMA.md (TDS-02, v1.0.0, FROZEN — G1 APPROVED). Transcribed verbatim from Blueprint §6. [src: docs/WORKFLOW_SCHEMA.md:1-13] VERIFIED (doc)

### Q14: are workflows versioned, how
- Yes. `version` field, type SemVer, "immutable once registered." Also separate `schema_version` (int, serialization-format version, distinct from workflow semver). [src: docs/WORKFLOW_SCHEMA.md §1 table]
- `GetWorkflow(ctx, id string, version SemVer)` — registry keyed by (id, version). [src: internal/core/ports.go:30] VERIFIED (code+doc)

### Q15: workflow migration handling
- No workflow-definition migration mechanism found (no "migrate workflow version" code/doc). Only DB schema migrations exist (internal/storage/migrations/, numbered 0001-0006 per EDR-011 §7: domain_events, signals, audit_log, plugins, recall FTS). [src: docs/edr/edr-011-engine-semantics.md §7; internal/storage/migrations/]
- These are SQLite schema migrations, NOT workflow-definition version migrations. UNKNOWN/gap: no in-flight-instance migration-to-new-workflow-version logic found. UNKNOWN (absence of evidence)

### Q16: are workflows immutable once execution starts
- Workflow *definitions* are immutable once registered (version field, "immutable once registered" — WORKFLOW_SCHEMA.md §1), independent of whether execution started — immutability is at registration time, not execution-start time. VERIFIED (doc)
- Running instances reference `definition_id + definition_version` (preserved across rebuild per rebuild.go step 1 snapshot) — an instance is pinned to the exact version it started with. [src: internal/storage/rebuild.go:26-27] INFERRED (code)

### Q17: execution primitives that exist today
- Step (native/subprocess/plugin/intelligence/signal types), Transition (conditional edges), Trigger (manual/schedule/event/webhook), CompensationPlan/CompensationStep, RetryPolicy, WaitConfig (signal steps). [src: docs/WORKFLOW_SCHEMA.md §1-§2; internal/core/step.go; internal/core/workflow.go]
- Engine tick/pull loop (`Run` drives pull loop at Config.TickInterval). [src: internal/engine/tick.go:15]
- ClaimStep atomic dispatch, StepResultCache idempotency. [src: internal/core/ports.go:26,52-55] VERIFIED (code)

### Q18: node/task/step types (enumerate from code)
- 5 StepType consts: native, subprocess, plugin, intelligence, signal. [src: internal/core/step.go:44-52] VERIFIED (code)

### Q19: control-flow constructs present/absent
- Branching: YES — Transition.condition (condition-expr grammar), on_error transitions. [src: docs/WORKFLOW_SCHEMA.md §3; docs/edr/edr-011-engine-semantics.md §8]
- Fan-out: YES — "fan-out transitions are THE parallel mechanism" (Blueprint B7); multiple transitions from one step all fire. [src: docs/edr/edr-011-engine-semantics.md §1]
- Fan-in/join: YES — join gate rule: target activates iff every distinct upstream `from` step completed AND ≥1 inbound transition fired; step_claims dedup at-most-once. [src: docs/edr/edr-011-engine-semantics.md §1]
- Retries: YES — RetryPolicy{attempts, backoff: immediate|linear|exponential, initial_delay, max_delay, retryable_errors}; backoff math implemented in internal/engine/retry.go. [src: docs/WORKFLOW_SCHEMA.md §2; internal/engine/retry.go:34]
- Waits: YES — type=signal step + WaitConfig{signal name, timeout, timeout_action}; durable wait_records table; recoverWait on restart. [src: docs/WORKFLOW_SCHEMA.md §2; internal/storage/rebuild.go:41-44]
- Schedules: **IMPLEMENTED** (corrected — not a stub). `TriggerTypeSchedule` const [src: internal/core/workflow.go:69] + a stdlib 5-field cron parser (exact/`*`/`*/N`/lists/ranges) [src: cmd/awis/cron.go:1-30] + a cron scanner goroutine started by `awis start` [src: cmd/awis/start.go:274-295]. Landed as M17-C2 (F-2). Scheduler lives in the CLI process, NOT in internal/engine — that is why an internal/-only grep misses it. VERIFIED (code)
- Loops (for-each/while): NOT FOUND — no "loop" construct in DSL/WORKFLOW_SCHEMA.md; no loop-related fields in Step/Transition schema. VERIFIED (absence in schema+code) — loops do not exist as a workflow primitive.

### Q20: data model flowing between steps
- `Variables map[string]any` on WorkflowInstance — "accumulated step outputs plus workflow inputs." Keyed by step id → that step's outputs map; `inputs` key holds workflow inputs. [src: internal/core/instance.go:21-22; internal/engine/emit.go:331,353-356]
- Steps declare typed `inputs`/`outputs` as JSON Schema (InputSchema/OutputSchema). [src: docs/WORKFLOW_SCHEMA.md §2] VERIFIED (code+doc)

### Q21: typed workflow system? yes/no + how
- YES, JSON-Schema-typed at step boundary: Step.inputs and Step.outputs are JSON Schema definitions validated presumably at registration/execution (validator in internal/validate). [src: docs/WORKFLOW_SCHEMA.md §2 "InputSchema ... JSON Schema for expected inputs"]
- Not a general static type system for the whole DSL — typing is per-step I/O contract via JSON Schema, plus template/condition expression grammars are themselves untyped-but-bounded (string/number/bool/null only, see Q23). VERIFIED (doc)

### Q22: how variables represented and persisted
- Represented in-memory as `map[string]any` on WorkflowInstance.Variables during execution. [src: internal/core/instance.go:22]
- Persisted: NOT itself an event-log concept as a first-class field — derived/rebuilt from StepCompleted payloads (outputs) and WorkflowStarted payload (inputs) during projection (rebuild.go marshals "vars" into projection row as JSON). [src: internal/storage/rebuild.go:241 "marshal vars"] VERIFIED (code) — Variables is a projected/derived value, not separately durable outside the event payloads + projection row JSON column.

### Q23: expression language (name, grammar file, sample)
- No single named language; TDS-03 defines TWO grammars: Template Expressions `{{ ... }}` and Condition Expressions. [src: docs/EXPRESSION_GRAMMARS.md]
- Grammar file: docs/EXPRESSION_GRAMMARS.md (TDS-03, FROZEN — G1 APPROVED). Code: internal/expr/ (parser), internal/expr/corpus/corpus.go (conformance fixtures mirroring doc rows 1:1). [src: docs/EXPRESSION_GRAMMARS.md:14]
- Template sample: `{{ workflow.inputs.<key> }}`, `{{ steps.<step-id>.outputs.<key> }}`, `{{ steps.<step-id>.status }}`. [src: docs/EXPRESSION_GRAMMARS.md Grammar 1]
- Condition sample grammar: `condition ::= or-expr`, ops `== != > < >= <=`, `&&`, `||`, `!`, scopes `workflow|steps|event`. [src: docs/EXPRESSION_GRAMMARS.md Grammar 2] VERIFIED (doc+code)

### Q24: are expressions deterministic? enforced how
- YES, deterministic by construction: grammars prohibit arithmetic, function calls, nested templates, string concatenation — i.e. no side-effecting or environment-dependent ops beyond path lookups into workflow/steps/event data already frozen in the projection/event payloads. [src: docs/EXPRESSION_GRAMMARS.md "Prohibited" lists, both grammars]
- `Eval` "never errors on types" — comparisons on mismatched types resolve to fixed false/true per table, no randomness/coercion ambiguity. [src: docs/edr/edr-010-expression-evaluation-semantics.md §1]
- Template `Resolve` returns `(string, []Warning)`, deterministic string output, never error. [src: docs/edr/edr-010-expression-evaluation-semantics.md §2]
- Enforcement mechanism: grammar-level prohibition (no arithmetic/functions/nesting) rather than a runtime sandbox — determinism is a property of what the grammar can even express, not a checked runtime constraint. VERIFIED (doc)

---

## SECTION C — INTELLIGENCE, PLUGINS, CLI, API, GUI, OPS, SECURITY

### Q1: Intelligence-provider abstraction — what supported
- interface `core.IntelligencePort` [src: internal/core/ports.go:44]
- methods: Draft, Embed, Synthesize, Classify, IsAvailable, Capabilities, ProviderName. [src: internal/core/ports.go:44-63]
- LLM: yes (Draft/Synthesize). Embeddings: method exists but unsupported by real adapter (see Q2/Q4). Tools/agents: NO concept in interface. Multimodal: NO concept in interface.
- tag VERIFIED (code)

### Q2: Providers implemented
- NullAdapter (zero-AI/default fallback) [src: internal/intelligence/adapters/null/null.go]
- Anthropic adapter (Messages API, stdlib net/http only) [src: internal/intelligence/adapters/anthropic/anthropic.go]
- no other adapters (no OpenAI/Ollama/etc dirs) [src: internal/intelligence/adapters/ dir listing]
- tag VERIFIED

### Q3: Contract/interface (terse signatures)
- `Draft(ctx, DraftRequest) (DraftResponse, error)`
- `Embed(ctx, text string) ([]float32, error)`
- `Synthesize(ctx, SynthesisRequest) (SynthesisResponse, error)`
- `Classify(ctx, text string, categories []string) (Classification, error)` — declared non-callable placeholder V1 (FR-IL-10) [src: internal/core/ports.go:51-55]
- `IsAvailable() bool`, `Capabilities() []Capability`, `ProviderName() string`
- [src: internal/core/ports.go:44-63]
- tag VERIFIED

### Q4: Provider failure handling
- Router = `CapabilityRouter`: eligibility restricted to adapters named in a declared `fallbackChain`; ordered chain, first eligible wins [src: internal/intelligence/router.go:70-78]
- `Dispatcher.Draft`: enforces token budget BEFORE routing (ceil(len/4) estimator, EDR-008); over-budget = immediate `ContextBudgetExceededError`, no provider touched [src: internal/intelligence/dispatcher.go:44-59]
- if no eligible adapter + required=true → `CapabilityUnavailableError`; required=false → `FallbackSignal` (engine activates step fallback) [src: internal/intelligence/router.go:44-65]
- provider call fails → tries next in chain; if all fail, error + last provider error joined via `errors.Join` [src: internal/intelligence/dispatcher.go:56-59]
- Anthropic-specific retry (CloudRetryPolicy): retry on HTTP 429/5xx, honors `Retry-After` header else exponential backoff (500ms,1s,2s…), max 3 attempts (configurable), non-retryable 4xx fails immediately, all sleeps ctx-aware [src: docs/PROVIDERS.md "Retry Behaviour"]
- no "power levels" concept found in code/docs — not a real term here
- tag VERIFIED (code + doc, consistent)

### Q5: What are "subprocess/plugin runners"
- Two distinct one-shot vs long-lived mechanisms:
- `internal/runner/subprocess`: one-shot subprocess step runner per TDS-04 (each step execution spawns fresh process, single request/response over stdio, no persistence between calls) [src: internal/runner/subprocess/subprocess.go, doc.go]
- `internal/runner/native`: in-process Go step execution [src: internal/runner/native/native.go]
- `internal/runner/intelligence`: routes intelligence-typed steps to the IntelligencePort dispatcher [src: internal/runner/intelligence/intelligence.go]
- `internal/plugin`: long-lived plugin manager — spawns plugin process ONCE, routes many step executions to it over persistent stdin/stdout JSON-RPC 2.0 (contrast with one-shot subprocess) [src: docs/PLUGIN_PROTOCOL.md lines 15-25, internal/plugin/manager.go, runner.go]
- tag VERIFIED

### Q6: What are plugins today
- Local child processes spawned by the Go runtime via os/exec, communicating over stdin/stdout NDJSON JSON-RPC 2.0. NOT containers, NOT remote services, NOT MCP servers.
- Language-neutral protocol; Python reference lib `python/awis-plugin` exists; example `plugins/git-context-plugin` [src: docs/PLUGIN_PROTOCOL.md:1-25, plugins/git-context-plugin]
- No container/remote/MCP support anywhere in code or docs.
- tag VERIFIED

### Q7: Isolation & security model for plugins
- Process isolation (separate OS process, not goroutine) [src: AWIS_ARCHITECTURE_BLUEPRINT.md §21 "Plugin Sandboxing"]
- Minimal env inheritance: manifest `env` + `PATH` only, no other parent env vars (NFR-S-02) [src: docs/PLUGIN_PROTOCOL.md §10]
- No extra file descriptor inheritance (os/exec default) [src: docs/PLUGIN_PROTOCOL.md §10]
- Process group isolation via `Setpgid`; SIGKILL to whole group on kill (reaps descendants) [src: docs/PLUGIN_PROTOCOL.md §10, internal/plugin/manager.go]
- No filesystem access "by convention" — NOT enforced; WASM enforcement deferred to V3 [src: AWIS_ARCHITECTURE_BLUEPRINT.md §21]
- What plugins cannot do (by protocol contract, not sandboxed at OS level beyond above): access EventLog/StateStore directly, register workflow definitions, hold state between calls, access another namespace's data [src: docs/PLUGIN_PROTOCOL.md §10 "What Plugins Cannot Do"]
- V1 threat model explicitly excludes authN/authZ/network security/encryption/access control — deferred to V2 [src: AWIS_ARCHITECTURE_BLUEPRINT.md §21 "V1 Threat Model"]
- tag VERIFIED (doc, consistent with code: manifest.go, manager.go)

### Q8: How permissions enforced
- No permission system exists. Plugin capability restriction is by PROTOCOL CONTRACT ONLY (plugins only receive declared step inputs/outputs; no code-level ACL, no sandbox enforcing "cannot access EventLog" beyond simply not passing it a handle).
- No RBAC/ABAC anywhere; no auth middleware seam even exists in HTTP API (see Q17/Q18).
- tag VERIFIED (absence confirmed via grep across internal/, docs)

### Q9: Plugin lifecycle
- States (Blueprint verbatim): REGISTER → SPAWN → HANDSHAKE → ACTIVE → IDLE → TERMINATE; RESTART on crash (max 3x) [src: docs/PLUGIN_PROTOCOL.md §7]
- In-memory FSM states: REGISTERED, SPAWNING, HANDSHAKING, ACTIVE, IDLE, TERMINATED, FAILED [src: docs/PLUGIN_PROTOCOL.md §7]
- DB `plugins.status` coarse: registered|active|failed ("suspended" unused in V1)
- Registration: `Manager.Register(manifestPath)` parses+validates manifest, writes DB rows, emits `PluginRegistered` audit event
- Lazy spawn on first call (`ensureActive()`); handshake 5s deadline
- Calls serialized per-plugin (mutex); cross-plugin concurrency unaffected
- Crash: in-flight call fails `plugin_crash`; counter>3 → FAILED permanently until re-registration; counter resets on successful execute
- Idle: no execute for `idle_timeout_s` → kill process group, respawns transparently next call
- Shutdown: `shutdown` notification, wait ≤2s, then SIGKILL group; idempotent
- [src: docs/PLUGIN_PROTOCOL.md §7, internal/plugin/manager.go]
- tag VERIFIED

### Q10: Plugin registry concept
- Exists in DB schema terms: `plugins` + `plugin_capabilities` tables (part of `runtime.db` StoragePort scope) [src: AWIS_ARCHITECTURE_BLUEPRINT.md line 690 "...WorkflowRegistry, StepResultCache, signal_inbox, wait_records, plugin registry"; migration `0003_add_plugin_registry.sql` line 1448]
- No marketplace/discovery registry. Install is MANUAL in V1 (place manifest+module on host, call `Manager.Register`); `awis plugin install` CLI planned M14; full marketplace explicitly deferred to V2 (FR-PS-11/12) [src: docs/PLUGIN_GUIDE.md §7]
- CLI already has `awis plugin` subcommand (install, list) implemented [src: cmd/awis/plugin.go:25]
- tag VERIFIED — registry = local SQLite table registry, not a marketplace

### Q11: Is CLI the primary control plane today
- Yes. CLI (`awis`) does ALL mutations (submit/signal/cancel/config/plugin install) via direct SQLite access under WAL+busy-timeout — no socket/RPC layer [src: docs/CLI_CONTRACT.md §1 "Interaction Model"]
- HTTP API (`internal/api`, served by `cmd/awis-server`) is READ-ONLY (GET only), no mutation routes exist [src: internal/api/router.go:29-40; ARCHITECTURAL_DEBT_REGISTER.md AD-08 "Read-only API with no mutation seam"]
- tag VERIFIED

### Q12: CLI framework used (EDR-002)
- EDR-002 DECIDED `spf13/cobra` (rationale: broad multi-level command tree) [src: docs/edr/edr-002-cli-framework.md]
- ACTUAL CODE: stdlib `flag` package with a hand-rolled `commands map[string]command` dispatch mux — cobra is NOT imported/used anywhere. main.go's own doc comment: "no third-party CLI framework is used (dependency policy...)" [src: cmd/awis/main.go:1-31]
- **DOC-VS-CODE DRIFT**: EDR-002 (decision doc) says cobra; shipped code uses stdlib flag mux only. Flag this explicitly.
- tag VERIFIED (code is ground truth: stdlib flag, not cobra) — DRIFT confirmed

### Q13: CLI commands existing today (from cmd/awis init() registrations)
- version, start, stop, submit, signal, cancel, status, history, trace, export, replay, rebuild-state, recall, prune-events, audit, logs, metrics, config (show/set/validate/edit), workflow (validate/list/show), plugin (install/list), init
- [src: grep of `commands["..."]  = command{...}` across cmd/awis/*.go, e.g. cmd/awis/start.go:42, submit.go:22, plugin.go:25, workflow.go:20, config.go:24, init.go:33]
- tag VERIFIED

### Q14: CLI commands planned but not implemented
- Full plugin marketplace commands (V2, FR-PS-11/12) — `awis plugin install` from a registry/marketplace vs current manual manifest registration [src: docs/PLUGIN_GUIDE.md §7]
- `awis migrate` explicit command mentioned for cloud multi-worker mode (V2) — not found in cmd/awis command list [src: AWIS_ARCHITECTURE_BLUEPRINT.md line 1451]
- No GUI-mutation-triggering CLI extensions planned to change (CLI remains sole mutation path per GUI Beta scope) [src: docs/09-gui-planning/GUI_BETA_GAP_ANALYSIS.md:47]
- tag INFERRED from doc statements + absence in code

### Q15: API architecture
- REST-like (JSON over HTTP, versioned `/api/v1`), stdlib `net/http.ServeMux` with Go 1.22+ method+path patterns; NO third-party router; NOT GraphQL/RPC [src: internal/api/router.go:1-40 header comment + code]
- HTTP API DOES exist: `internal/api` package + `cmd/awis-server` binary mounts it [src: internal/api/router.go, cmd/awis-server/main.go]
- Current routes (all GET, read-only): `/api/v1/healthz`, `/api/v1/info`, `/api/v1/workflows`, `/api/v1/workflows/{id}/{version}`, `/api/v1/instances`, `/api/v1/instances/{id}`, `/api/v1/instances/{id}/events` [src: internal/api/router.go:29-40]
- planned addition: `GET /stream` (SSE, `state_changes` tail) — NOT yet in router.go, documented only [src: docs/09-gui-planning/GUI_ARCHITECTURE.md:238]
- `web/` frontend served by `cmd/awis-server/static.go` (embeds built static assets) [src: cmd/awis-server/static.go]
- tag VERIFIED (code); planned items INFERRED from doc

### Q16: Is API contract formally documented
- Partially. `docs/09-gui-planning/GUI_ARCHITECTURE.md` §6 documents the route table + ADRs (§5.1 ADR-4 SSE, §4.2 ADR-1 state_changes, §4.3 ADR-2 namespace, §7 ADR-3 module boundary) as a design/decision doc, not a frozen contract like CLI_CONTRACT.md.
- No equivalent "TDS-0X API Contract" doc exists (unlike TDS-07 for CLI, TDS-05 for plugin protocol). No OpenAPI/Swagger spec found in repo.
- tag VERIFIED (absence) — API contract is less formally frozen than CLI/plugin protocols

### Q17: AuthN mechanisms that exist
- NONE. V1 threat model explicitly states: "V1 does NOT implement: authentication, authorization, network security, encrypted storage, or access control." [src: AWIS_ARCHITECTURE_BLUEPRINT.md §21]
- Confirmed in code: no auth/token/session/jwt logic in internal/api/*.go [src: grep internal/api/*.go — zero hits for auth/token/session]
- ARCHITECTURAL_DEBT_REGISTER AD-07: "no middleware seam at all" — not just missing auth but nowhere TO add it without touching all 7 routes [src: ARCHITECTURAL_DEBT_REGISTER.md AD-07]
- REPOSITORY_TRUTH_AUDIT.md: "zero user/role/session/tenant concept anywhere (VERIFIED FACT)" [src: docs/09-gui-planning/REPOSITORY_TRUTH_AUDIT.md:54]
- tag VERIFIED

### Q18: AuthZ model
- NONE exists. Same evidence as Q17. Deferred explicitly to V2 ("when AWIS supports multiple concurrent users") [src: AWIS_ARCHITECTURE_BLUEPRINT.md §21]
- Only "isolation" analog today is namespace row-level scoping (ADR-007) which is data partitioning, not authorization/access-control (any caller with DB access can pass any namespace string) [src: AWIS_ARCHITECTURE_BLUEPRINT.md ADR-007 line 2030]
- tag VERIFIED

### Q19: Multi-tenancy in code today
- Row-level namespace field (`Namespace string`) on events/instances/workflow definitions, filterable in queries — this is the ADR-007 "trusted-application" isolation model, NOT true multi-tenant security (no enforcement preventing cross-namespace reads; namespace is just a filter param) [src: internal/storage/sqlite.go: Namespace fields lines 155,267,328,534,578,644,698,736,850,974-976]
- `"default"` namespace has known overload/ambiguity bug flagged as ADR-2/D2 decision (must retire wildcard behavior) — unresolved [src: docs/09-gui-planning/GUI_ARCHITECTURE.md:145, ENGINE_GUI_DECISION_RECORD.md D2]
- No tenant/user/role concept exists (see Q17). V1 = single-tenant local deployment by design [src: AWIS_ARCHITECTURE_BLUEPRINT.md line 1545 "V1: Single Tenant, Local Deployment"]
- tag VERIFIED

### Q20: "read-only operational GUI" meaning + capabilities + stack
- Precise meaning: GUI Phase 1 exposes only GET/read routes — view workflows, instances, event timelines; ALL mutations (submit/signal/cancel/create/edit workflows) remain CLI-only. Explicit scope line: "Workflow creation/editing/mutation from the GUI — explicitly out of scope" [src: docs/09-gui-planning/GUI_BETA_GAP_ANALYSIS.md:47; web/package.json description: "read-only operational dashboard"]
- GUI capabilities existing TODAY (code, not just planned): instance list, instance detail, event timeline, workflow list, workflow detail screens; app shell + router; polling-based updates (`startVisibilityAwarePoll`) [src: web/src/main.ts, web/src/screens/*.ts]
- Frontend framework: `lit-html` (^3.2.0) — NOT React/Vue/Svelte — built with esbuild+TypeScript [src: web/package.json]
- Backend serving GUI: `cmd/awis-server` binary; embeds static build via `static.go`; mounts `internal/api` router [src: cmd/awis-server/main.go, static.go]
- Realtime mechanism: currently POLLING only in shipped code (`startVisibilityAwarePoll`); SSE is PLANNED (ADR-4, `/stream` route on `state_changes` table) but not yet implemented in router.go [src: web/src/ui.ts, docs/09-gui-planning/GUI_ARCHITECTURE.md §5.1 ADR-4]
- tag VERIFIED (code) for current state; SSE plan = INFERRED/planned only

### Q21: GUI Beta vs Workflow Creation/Edit APIs — what separates them
- GUI Beta (current/near-term scope) = read-only dashboard: view definitions, instances, events, live-ish status. No write path.
- Workflow Creation/Edit APIs = future "G6" visual editor milestone requiring: a YAML SERIALIZER (`yaml.Marshal` emitter — "called nowhere in the repo today"), map-key-order round-trip safety (N4 risk, unaddressed), save-from-canvas API, round-trip regression suite — none of this exists yet [src: docs/09-gui-planning/ENGINE_GUI_DECISION_RECORD.md:39 "yaml.Marshal is called nowhere in the repo today"; REPOSITORY_TRUTH_AUDIT.md N4; ENGINE_GUI_WORK_BREAKDOWN.md G-G6-2..12]
- Separator = presence of a definition→YAML serializer + mutation API; GUI Beta explicitly has neither (D-11 defect: "No workflow mutation API or YAML serializer") [src: VERIFIED_DEFECT_REGISTER.md D-11]
- tag VERIFIED

### Q22: Visual builder as primary authoring? Text-first? Round-trip requirement?
- Blueprint ADR-008 (frozen architecture decision): "YAML DSL as Tier 1 with Go SDK as Tier 2" — YAML/text is explicitly the PRIMARY tier, code is Tier 2 for complex cases. Visual builder is not mentioned in this ADR at all — it's a GUI-planning-phase addition layered on top. [src: AWIS_ARCHITECTURE_BLUEPRINT.md ADR-008 line 2049]
- Visual builder is explicitly a SAVE-PATH ONTO the same YAML/WorkflowDefinition shape, not a replacement: "canvas → WorkflowDefinition → YAML, round-tripped" (GUI_PRD FR-4.3) — confirms definitions stay text-first (YAML is the source of truth on disk) and the visual builder is an alternate editing surface, not primary authoring.
- Round-trip GUI<->declarative editing IS an explicit requirement: FR-4.3 + acceptance criterion "author in canvas → serialize → parse → compare Transition/Compensation/Fallback sets" identical to hand-authored — E2E round-trip acceptance test (G-G6-12) is a planned gate. [src: docs/09-gui-planning/GUI_PRD.md:159,181; ENGINE_GUI_WORK_BREAKDOWN.md G-G6-12]
- tag VERIFIED (doc) — text-first confirmed by frozen ADR-008; visual builder is secondary, round-trip is a hard requirement, not yet built (serializer doesn't exist)

### Q23: ADRs/EDRs that exist
EDRs (docs/edr/):
- EDR-001 Module path (CONTRA-1 disposition)
- EDR-002 CLI framework — decided cobra (DRIFT: code uses stdlib flag, see Q12)
- EDR-003 SQLite driver — modernc.org/sqlite (pure Go, CGO_DISABLED... though CI notes CGO_ENABLED=1 for race)
- EDR-004 AEO model mapping
- EDR-005 sequence_num assignment vs enforcement
- EDR-006 step_claims: claim mechanism + release site
- EDR-007 Projection Rules for RebuildState
- EDR-008 Context-Budget Token Estimator
- EDR-009 Adapter Traits as Registration Metadata
- EDR-010 Expression Evaluation & Validation Semantics
- EDR-011 Engine Semantics Beyond the Frozen Text
[src: docs/edr/edr-001..011-*.md]

ADRs — AWIS_ARCHITECTURE_BLUEPRINT.md §30 (canonical, frozen):
- ADR-001 Step as the Fundamental Workflow Primitive
- ADR-002 Pull-Based Execution Loop over Push-Based Event-Driven
- ADR-003 Append-Only EventLog as State Source of Truth
- ADR-004 SQLite for Local Storage; Postgres-Compatible Interface for Cloud
- ADR-005 IntelligencePort with Capability-Based Routing
- ADR-006 Plugin Communication via stdin/stdout JSON-RPC
- ADR-007 Namespace-Per-Application Isolation
- ADR-008 YAML DSL as Tier 1 with Go SDK as Tier 2
- ADR-009 Optimistic Locking for Multi-Worker Cloud Mode
- ADR-010 OIP is the First AWIS Application
- ADR-011 No Central Coordinator Process
- ADR-012 Workflow Versioning via Immutable Semver
- ADR-013 Two-Process Step Execution Model (Native + Subprocess)
- ADR-014 Local-First with No Network Assumptions
- ADR-015 Go as Primary Implementation Language
[src: AWIS_ARCHITECTURE_BLUEPRINT.md lines 1898-2145]

GUI-planning ADRs (docs/09-gui-planning/GUI_ARCHITECTURE.md, NOT yet ratified — labeled "Recommended", tied to open founder decisions D1-D4):
- ADR-1 `state_changes` additive table (global change cursor)
- ADR-2 Retire `"default"` namespace wildcard overload
- ADR-3 GUI backend in-module (`cmd/awis-server`)
- ADR-4 SSE over WebSocket
[src: docs/09-gui-planning/GUI_ARCHITECTURE.md:364-367; ENGINE_GUI_DECISION_RECORD.md — "Independently re-confirmed 2026-09-01: none of D1-D4 has been decided"]
- tag VERIFIED

### Q24: Major architectural decisions still unresolved
- D1-D4 (GUI founder decisions): state_changes table vs global_seq column; retire "default" namespace wildcard; GUI backend in-module vs separate service; gate vs ship-unverified live intelligence — ALL FOUR confirmed still open as of 2026-09-01 [src: docs/09-gui-planning/ENGINE_GUI_DECISION_RECORD.md §1]
- "What is the engine" process decision: `main` branch 60 commits behind `engine-hardening`, STATE.md shows stale milestone status — must be resolved before GUI work proceeds [src: docs/09-gui-planning/ENGINE_GUI_DECISION_RECORD.md §2]
- CONTRA-5: local-mode "no concurrent writers" (Blueprint §9) vs CLI-as-second-writer sanctioned by §28 — resolved by TDS-07 refinement (single-engine-instance interpretation), but recorded as a standing tension, not silently closed [src: docs/CLI_CONTRACT.md §1]
- tag VERIFIED (doc)

### Q25: Known technical debts (top items)
ARCHITECTURAL_DEBT_REGISTER.md:
- AD-01 no version-control history for the deliverable (Now)
- AD-02 frozen 12-method StoragePort cannot express bounded reads (Next)
- AD-03 SetMaxOpenConns(1) shared between engine loop and HTTP reads (Watch)
- AD-04 FTS index rebuilt per query, not maintained (Next)
- AD-05 EventLog durability = synchronous=NORMAL (Next)
- AD-06 no downgrade guard in migration runner (Now/cheap)
- AD-07 HTTP surface has no security/middleware seam at all (Now)
- AD-08 read-only API, no mutation seam (Watch — correctly deferred)
- AD-09 frontend types unenforced (Next)
[src: ARCHITECTURAL_DEBT_REGISTER.md]

VERIFIED_DEFECT_REGISTER.md (top defects, D-01..D-13):
- D-01 in-flight step crash recovery deadlock
- D-02 cancellation intent lost on restart
- D-03 dashboard shows oldest not newest instances
- D-04 subprocess steps inherit full host environment
- D-05 Anthropic model ID bug (first fix also wrong)
- D-06 non-final dead-end branch wedges instance in `running` forever
- D-07 healthcheck never checks the database
- D-08 FTS index fully rebuilt every search
- D-09 single SQLite connection shared by engine tick + all HTTP reads
- D-10 workflow_definitions PK omits namespace
- D-11 no workflow mutation API or YAML serializer
- D-12 strict acyclic DAG precludes loop constructs
- D-13 flaky system-rehearsal test under parallel load
[src: VERIFIED_DEFECT_REGISTER.md]

DEFERRED_TECHNICAL_DEBT.md: single SQLite connection (safe to defer), 6 stale worktree-agent branches, GUI doesn't render Step in/out JSON Schema, .gitignore missing awis-server binary, awis-core-engineer.md still Opus-configured, one flaky system test.
- tag VERIFIED

### Q26: Observability stack
- Logging: `log/slog`, JSON handler to stderr by default; structured file sink to `<data-dir>/awis.log` (`awis start`); `awis logs` CLI reads/filters it (--tail/--level/--instance) [src: internal/engine/engine.go:65,112,130; cmd/awis/logs.go]
- Metrics: NONE — "zero Prometheus/OpenTelemetry/expvar anywhere (VERIFIED FACT)" [src: docs/09-gui-planning/REPOSITORY_TRUTH_AUDIT.md:54,133]
- Tracing: NONE — same citation; "no metrics or tracing library of any kind anywhere in the repository"
- `awis metrics` CLI command exists but is an aggregate-stats-from-DB query, not a metrics/instrumentation system [src: cmd/awis/metrics.go]
- Audit log exists (DB `audit_log` table, `awis audit` CLI) — distinct from metrics/tracing [src: cmd/awis/audit.go]
- tag VERIFIED — observability infra is materially absent per audit doc, confirmed by code search

### Q27: Testing strategy
- Layers per Blueprint §27: Unit (Go testing + mock StepContext), Integration (`WorkflowTestHarness` in-memory), Contract (shared StoragePort suite run against SQLite [and planned Postgres]), System (real SQLite test binary), Experiment (dogfood metrics) [src: AWIS_ARCHITECTURE_BLUEPRINT.md §27]
- Deterministic execution mode: fixed clock, seeded IDs, routes intelligence to NullAdapter, manual tick advance — for stable test output
- "AWIS-E1" CI gate: every workflow completes with NullAdapter as sole provider (zero-AI). Actually wired: `make e1` target [src: Makefile "e1:"; AWIS_ARCHITECTURE_BLUEPRINT.md §27]
- Golden-file testing used extensively for CLI (`cmd/awis/testdata/golden/*.json|.txt`)
- Actual CI/make targets: gofmt-check, vet, lint (golangci-lint), oip-isolation (GOWORK=off build/vet of apps/oip as standalone module), build, test, race (-race), e1; `contract`, `bench`(NOT-YET), `release-dry`(NOT-YET), `integration` (tag-gated) also present [src: Makefile]
- tag VERIFIED

### Q28: Deployment targets supported
- V1 (current): single-binary local-first, SQLite, no network assumptions, single-user (ADR-014, ADR-004) [src: AWIS_ARCHITECTURE_BLUEPRINT.md ADR-004, ADR-014]
- V2/V3 planned: "Mode 3: Cloud Deployment" — same binary + Postgres storage, multiple worker instances via optimistic locking, no central coordinator; "no Kubernetes required — a single VM running Docker Compose is sufficient" [src: AWIS_ARCHITECTURE_BLUEPRINT.md §28 line 1823-1835]
- NO Dockerfile exists in repo today (grep found none) — containerization is planned prose only, not implemented
- tag VERIFIED (V1 code); V2/V3 = documented plan only, unimplemented

### Q29: CI/CD pipeline
- `.github/workflows/ci.yml`: single job "Verify (Linux)", ubuntu-latest, 30 min timeout, on push/PR to any branch
- Steps: checkout (fetch-depth:0, required for git-diff test), setup-go 1.26.x, install golangci-lint v2.12.2, then named gates: gofmt, go vet, golangci-lint, apps/oip standalone module check (GOWORK=off), build, test, race, "Zero-AI gate (E1)" (`make e1`)
- Deliberately NOT wired: macOS matrix, docs-lint (fails due to incomplete EEOS artifacts), pytest, benchmarks, release-dry — explicitly documented as known gaps in the workflow file's own comments
- CGO_ENABLED left at default 1 (required for `-race`)
- Makefile `verify` target = single source of truth; CI runs same gates as local `make verify`
- tag VERIFIED

### Q30: Security requirements
- Threat model (V1, explicit, Blueprint §21): scope = plugin code crash/hang (mitigated via process isolation), step handlers reading/writing outside declared scope (NOT enforced — "application discipline" only), secrets in workflow variables (config-based, no encryption)
- Explicitly OUT of V1 scope: authentication, authorization, network security, encrypted storage, access control — deferred to V2 multi-user requirements [src: AWIS_ARCHITECTURE_BLUEPRINT.md §21 "V1 Threat Model"]
- Secrets handling: API keys/credentials never stored in workflow definitions or EventLog; read from env vars at runtime startup; injected into adapter constructors; never passed to step handlers as inputs [src: AWIS_ARCHITECTURE_BLUEPRINT.md §21 "Secret Management"]
- Anthropic adapter never logs raw API key — `mask(key)` → "sk-ant-…<last4>" in all logs/errors/fixtures (NFR-S-01), tested [src: docs/PROVIDERS.md "Key Masking"]
- `awis config show` masks by **ALLOWLIST of the harmless, not a denylist of the dangerous** — any key not on the allowlist is masked, so unknown/new keys fail closed [src: cmd/awis/config.go:28-60, isSecretConfigKey:59]. (Substring/credential-shape matching was the OLD behaviour; replaced by commit `0ddf1ce` "mask config values by allowlist" + `c143fed` "make config secret masking fail closed".) VERIFIED (code)
- Plugin processes: minimal env (manifest env + PATH only), no platform secret leakage; plugin credentials must be explicitly declared in `plugin.env` [src: AWIS_ARCHITECTURE_BLUEPRINT.md §21 "Secret Management" last line]
- HTTP API AD-07: no security middleware seam exists at all yet (see Q17)
- tag VERIFIED

---

*End. Sources are repo-relative paths at `engine-hardening@8a87f70` + uncommitted working tree.*
