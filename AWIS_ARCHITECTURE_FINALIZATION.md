# AWIS ARCHITECTURE FINALIZATION
## Resolution of 7 Critical Blockers — v1.0 Freeze

**Authority:** AWIS_CANONICAL_SPECIFICATION_TRIBUNAL_REPORT.md
**Produces:** AWIS Architecture v1.0 (frozen)
**Date:** 2026-07-02
**Scope:** Minimum decisions required to freeze the architecture. No new features. No redesign.

Each blocker is resolved with: root cause → alternatives → trade-off analysis → decision → justification → affected text.

---

## BLOCKER 1 — OIP Architecture Schism

### Root Cause

Two complete, incompatible OIP architectures exist simultaneously:

- `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` (OIP standalone): OIP is a single `oip` binary. It owns its own IntelligencePort, RecordPort, IndexPort. No AWIS dependency. Storage is `.decisions/.index/index.db` (OIP's SQLite). Six weeks to implement OIP alone.
- `AWIS_ARCHITECTURE_BLUEPRINT.md` §10 (OIP on AWIS): OIP is an AWIS application. It implements AWIS StepHandlers and WorkflowDefinitions. Its capture and recall loops are AWIS workflows. The platform provides execution infrastructure.

The AWIS Blueprint was produced after the OIP Engineering Blueprint and redesigned OIP's implementation model without explicitly retiring the prior document.

### Alternatives

**A — Explicit supersession.** The AWIS Blueprint governs OIP's implementation. The Engineering Blueprint is archived as a historical record. OIP is built as an AWIS application from day one.

**B — Sequential delivery.** Build OIP standalone first (OIP Engineering Blueprint), validate the product, then migrate OIP to AWIS later. Two implementation phases; two different architectures to maintain simultaneously during migration.

**C — Keep both, let implementer choose.** No guidance provided. Acceptable in a large organization with separate teams; not acceptable for a solo founder who must make one clear choice.

### Trade-off Analysis

| Criterion | Option A (Supersede) | Option B (Sequential) | Option C (Choose) |
|---|---|---|---|
| Implementation clarity | One path, one architecture | Two paths, sequential | Ambiguous |
| Validation of AWIS | OIP validates AWIS immediately | OIP validation deferred | Variable |
| Implementation risk | AWIS must exist before OIP can ship | OIP ships faster (no AWIS) | Variable |
| Long-term maintenance | Single codebase; OIP gets platform features | Two migrations of OIP code | Variable |
| Solo-founder cognitive load | One system to understand | Two systems, then migration | Maximum |

Option B sounds safe but creates the worst-case solo-founder situation: two full implementation projects, followed by a migration. The OIP Engineering Blueprint's six-week standalone estimate does not shrink if AWIS is built afterward — it shifts right while AWIS is built, then adds migration cost. Total: (AWIS 6 weeks) + (OIP standalone 6 weeks) + (migration N weeks) >> Option A's (AWIS+OIP 6 weeks combined).

### Decision

**Option A: Explicit Supersession.**

`ENGINEERING_ARCHITECTURE_BLUEPRINT.md` is the historical standalone OIP design. It is superseded by `AWIS_ARCHITECTURE_BLUEPRINT.md`. For implementation, OIP is an AWIS application. The standalone architecture is archived; it must not be used as an implementation guide.

### Justification

The AWIS Architecture Blueprint was designed specifically to validate the platform against OIP as the first application. Building OIP standalone would skip that validation entirely, which is its own architectural risk (ADR-010: OIP as first application). The six-week AWIS implementation sequence already accounts for OIP in weeks 5–6. Building OIP inside AWIS costs zero additional time versus standalone; it produces a validated platform as a side effect.

Additionally, OIP already inherited the correct abstractions in the Engineering Blueprint: IntelligencePort, RecordPort, IndexPort. AWIS generalizes and promotes these seams to platform level. The implementation work is nearly identical; the difference is who owns the interfaces.

### Affected Documents

- `ENGINEERING_ARCHITECTURE_BLUEPRINT.md`: receives a supersession header (see §Architecture Updates below)
- `AWIS_ARCHITECTURE_BLUEPRINT.md` §10: receives an explicit statement of OIP's implementation model
- No workflow, step, or interface definitions change

---

## BLOCKER 2 — Expression Language Specification

### Root Cause

The YAML DSL in `AWIS_ARCHITECTURE_BLUEPRINT.md` §7 uses two syntactically distinct mechanisms without defining either:

1. **Template interpolation**: `"{{workflow.inputs.repo_path}}"` — references resolved at execution time
2. **Condition expressions**: `"steps['draft-entry'].status == 'fallback'"` and `"event.branch == 'main'"` — boolean predicates

These use inconsistent notation (dot-path in templates, bracket-path in conditions) and have no formal grammar. An implementer must make up rules for: operator precedence, null handling, quoted strings, scope, type coercion, and error behavior.

### Alternatives

**A — Unified syntax.** One expression language for both interpolation and conditions. Templates are `{{expr}}` where `expr` is also the condition language. Complex; blurs the semantic difference between a value reference and a boolean test.

**B — Two distinct formal grammars.** Templates and conditions are different syntactic forms with different grammars, each formally specified. Templates resolve to values; conditions resolve to booleans.

**C — No conditions in YAML.** Transitions with conditions must use the Go SDK. YAML becomes strictly declarative (linear flows only). Simple to specify; removes power from the declarative tier unnecessarily.

**D — Embed a known expression language.** Use `expr-lang` (Go library), CEL (Common Expression Language), or JSONata. Proven grammar; documentation exists; test suite exists. Dependency added.

### Trade-off Analysis

| Criterion | A (Unified) | B (Two grammars) | C (No conditions) | D (Library) |
|---|---|---|---|---|
| Specification effort | High (complex grammar) | Medium (two simple grammars) | Low | Low (grammar is external) |
| Implementation effort | High | Medium | Low | Low (import library) |
| Documentation effort | High | Medium | Low | Low (link to external docs) |
| Dependency burden | None | None | None | One external package |
| Expressiveness | Highest | High | Low | Highest |
| Surface to support | Large | Medium | Small | Medium (subset of library) |
| Ability to say "no" to feature requests | Difficult | Moderate | Easy | Difficult (library adds features) |

Option C loses too much — OIP's capture workflow needs the `steps['draft-entry'].status == 'fallback'` condition to distinguish the AI-draft path from the manual-draft fallback path. Without conditions in YAML, this workflow requires the Go SDK.

Option D is attractive but trades specification dependency for a runtime dependency. Any `expr-lang` library update is now a platform update. The expression surface for AWIS is small enough to specify formally; a general expression library is over-capability for this use case.

### Decision

**Option B: Two distinct formal grammars, explicitly bounded.**

Templates and conditions are semantically different and should remain syntactically distinct. Both grammars are minimal and formally specified below.

### Expression Language Formal Specification

#### Grammar 1: Template Expressions `{{ ... }}`

Templates appear in YAML string field values and are resolved at step execution time to produce concrete values passed as step inputs.

**Syntax:**
```
template-string ::= literal-text | "{{" path-ref "}}" | template-string template-string
path-ref        ::= scope "." identifier ("." identifier)*
scope           ::= "workflow" | "steps"
identifier      ::= [a-zA-Z_][a-zA-Z0-9_-]*
```

**Scope definitions:**
- `workflow.inputs.<key>` — the workflow's original inputs, provided at submission
- `steps.<step-id>.outputs.<key>` — a completed step's output field
- `steps.<step-id>.status` — a step's current status (string: pending | running | completed | failed | cancelled)

**Null handling:** If a referenced path does not exist or the referencing step has not yet completed, the template resolves to the empty string `""` and a warning is logged. Templates never fail at validation time; they fail gracefully at execution time.

**Prohibited:**
- Arithmetic operators (`+`, `-`, `*`, `/`)
- Function calls (`len(...)`, `toUpper(...)`, etc.)
- Conditionals within template expressions (`{{if ...}}`)
- Nested templates (`{{steps.{{x}}.outputs.y}}`)
- Bracket notation (`steps['step-id']` — use dot notation `steps.step-id` with hyphens allowed in identifiers)

#### Grammar 2: Condition Expressions (Transition `condition` field)

Conditions appear in `Transition.condition` and `Trigger.config.filter` fields. They evaluate to a boolean. The runtime evaluates them after a step completes to determine which transition(s) to activate.

**Syntax:**
```
condition   ::= or-expr
or-expr     ::= and-expr | or-expr "||" and-expr
and-expr    ::= not-expr | and-expr "&&" not-expr
not-expr    ::= compare-expr | "!" not-expr | "(" condition ")"
compare-expr ::= path-ref compare-op value | path-ref "==" "null" | path-ref "!=" "null"
compare-op  ::= "==" | "!=" | ">" | "<" | ">=" | "<="
path-ref    ::= scope "." identifier ("." identifier)*
value       ::= string-lit | number-lit | bool-lit
string-lit  ::= "'" [^']* "'"
number-lit  ::= [0-9]+ ("." [0-9]+)?
bool-lit    ::= "true" | "false"
scope       ::= "workflow" | "steps" | "event"
```

**`event` scope:** Only valid in Trigger `filter` fields. Provides access to the domain event payload: `event.<key>` where `<key>` is a top-level field in the `DomainEvent.payload`.

**Null handling:** Comparing a path that resolves to null against a non-null literal with `>`, `<`, `>=`, `<=` evaluates to `false`. `null == null` evaluates to `true`.

**Prohibited in conditions:**
- Arithmetic operators
- Function calls
- String concatenation
- Bracket notation (`steps['step-id']` — use `steps.step-id`)
- Ternary operators

**Validation at registration time:** The WorkflowValidator parses all condition expressions before registering a workflow definition. Invalid syntax causes registration to fail with a precise error identifying the field and syntax error.

**Note on dot-path identifiers:** Identifiers may contain hyphens (`step-id` is valid). This means `steps.draft-entry.status` is the canonical form. Hyphens in identifiers are treated as single tokens, not subtraction operators.

### Justification

The bounded grammar prevents the YAML DSL from becoming a programming language over time. Every feature request for the expression language can be evaluated against the grammar boundary: if it requires new syntax, it belongs in the Go SDK tier. The "prohibited" lists are as important as the grammar itself — they are explicit commitments that these features will not be added.

A solo founder maintaining a documented, bounded expression language for 5+ years is sustainable. A solo founder maintaining a general-purpose expression language is not.

### Affected Documents

- `AWIS_ARCHITECTURE_BLUEPRINT.md` §7: replaces implicit expression references with the formal grammars above
- No runtime, SDK, or interface changes

---

## BLOCKER 3 — Signal Atomicity

### Root Cause

Signal delivery modifies three pieces of state:
1. `signal_inbox`: mark the signal as delivered (`delivered_at = now`)
2. `execution_events`: append a `SignalReceived` event
3. `workflow_instances`: update `status` from `waiting` to `running`, increment `version`, update `updated_at`

No atomicity guarantee is stated. A crash between any of these writes leaves the system in an inconsistent state.

### Alternatives

**A — Single database transaction across all three tables.** All three writes are wrapped in a `BEGIN`/`COMMIT`. Commit succeeds or fails atomically. In SQLite (local mode), all three tables are in a single database file; a transaction spanning them is native and trivially correct. In Postgres (cloud mode), the same transaction semantics apply natively.

**B — Two-phase delivery with status column.** Add `status: pending | delivering | delivered` to `signal_inbox`. Phase 1: set `status = 'delivering'` (idempotency marker). Phase 2: write EventLog event and StateStore update. Phase 3: set `status = 'delivered'`. On crash during phase 2: next scan finds `status = 'delivering'` and re-attempts delivery from phase 2. At-least-once delivery; idempotency key prevents double-processing.

**C — EventLog-first delivery.** Write `SignalReceived` to EventLog first. Then update StateStore. Crash after EventLog write but before StateStore update: next `awis rebuild-state` restores consistency. This is the "EventLog is authoritative" principle applied to signal delivery.

### Trade-off Analysis

| Criterion | A (Single transaction) | B (Two-phase) | C (EventLog-first) |
|---|---|---|---|
| Local mode (SQLite) correctness | Perfect | Correct | Correct |
| Cloud mode (Postgres) correctness | Perfect (native transactions) | Correct (but more complex) | Correct after rebuild (not automatic) |
| Implementation complexity | Minimal | Medium | Minimal |
| Recovery path | Automatic | Automatic | Manual (`awis rebuild-state`) |
| Crash safety | Complete | Complete | Complete, manual recovery |
| Schema change required | None | Add status column to signal_inbox | None |

Option A is correct for both modes with zero additional complexity. Option B's two-phase approach is appropriate for distributed signal delivery (V3 distributed queues) but is over-engineering for a single-database local system. Option C is correct in principle but requires manual recovery via `awis rebuild-state` for a scenario where Option A requires no recovery at all.

### Decision

**Option A: Single database transaction across all three tables, with EventLog as the tiebreaker invariant.**

### Signal Delivery Atomicity Specification

Signal delivery is a single database transaction:

```
BEGIN TRANSACTION
  1. UPDATE signal_inbox
        SET delivered_at = <now>
      WHERE instance_id = <id>
        AND signal_name = <name>
        AND delivered_at IS NULL;          -- idempotency guard

  2. INSERT INTO execution_events
        (event_id, instance_id, namespace, event_type, payload, emitted_at, sequence_num)
      VALUES (..., 'SignalReceived', <payload>, ...);

  3. UPDATE workflow_instances
        SET status = 'running',
            updated_at = <now>,
            version = version + 1
      WHERE instance_id = <id>
        AND status = 'waiting'
        AND version = <expected_version>;  -- optimistic lock

COMMIT
```

If step 1's `delivered_at IS NULL` guard finds no row (signal already delivered): transaction is a no-op. This makes delivery idempotent — re-running the delivery scan never double-delivers.

If step 3's `version = <expected_version>` guard fails (cloud multi-worker race): transaction rolls back. The signal_inbox entry remains undelivered. The next scan tick re-attempts. This prevents double-delivery in cloud mode.

**Invariant:** The EventLog is authoritative. If `SignalReceived` exists in the EventLog for instance `X` and signal `Y`, the signal was delivered. If it does not exist, the signal was not delivered. `awis rebuild-state` can reconstruct the StateStore from EventLog at any time and will correctly represent delivered/undelivered signals.

**V2 note:** In multi-process cloud mode (Postgres), all three writes happen within a single Postgres transaction with the same semantics. The optimistic version lock on step 3 prevents concurrent workers from double-delivering the same signal.

### Justification

A single transaction is the correct tool for maintaining consistency across multiple tables in a single relational database. SQLite's WAL mode supports this natively with full ACID semantics. Adding complexity (two-phase protocol, saga pattern, event-sourced recovery) for a case that SQLite handles correctly in three lines of SQL violates KISS and adds maintenance burden without benefit.

The EventLog-as-authoritative-record principle adds the second layer of safety: even if the SQLite transaction mechanism itself somehow failed (corruption, driver bug), `awis rebuild-state` would restore correct state from the EventLog. Two independent correctness guarantees rather than one.

### Affected Documents

- `AWIS_ARCHITECTURE_BLUEPRINT.md` §8 (Execution Engine) and §9 (State Management): add the atomicity specification above
- `AWIS_ARCHITECTURE_BLUEPRINT.md` §9 Consistency Guarantees section: clarify signal delivery transaction semantics

---

## BLOCKER 4 — Cancellation Semantics

### Root Cause

`WorkflowRunner.Cancel()` and `awis cancel <instance-id>` appear in the SDK and CLI with `InstanceStatus.cancelled` in the status machine, but no behavior specification exists for:
- What happens to in-flight steps when cancellation is requested
- Whether compensation runs on cancellation
- What happens to pending wait_records (signal inbox entries)
- Whether cancellation is idempotent on terminal workflows
- What event is emitted

### Alternatives

**A — Immediate interrupt.** The runtime kills in-flight step goroutines and processes immediately. Status transitions to `cancelled`. Fast; may leave external writes in partial states.

**B — Graceful completion.** The runtime sets a cancellation flag. The current in-flight step runs to completion. After the step settles, the engine checks the flag and terminates the instance rather than activating the next step. Status transitions to `cancelled` after the current step finishes.

**C — Signal-based cooperative cancellation.** Applications define a "cancellation step" and the cancel command is implemented as a special signal delivery. Maximally flexible; requires application cooperation; cannot be used as a platform primitive.

For the specific sub-questions:

| Decision point | Options | Choice |
|---|---|---|
| In-flight step behavior | Interrupt vs. graceful completion | Graceful: step completes, next step does not start |
| Compensation on cancel | Always / Never / Optional | Never by default; `--compensate` flag enables it |
| Signal inbox cleanup | Clean up / Leave | Clean up: mark pending wait_records inactive |
| Idempotency on terminal state | Error / Warning + no-op | Warning + no-op |
| Event emitted | None / WorkflowCancelled | WorkflowCancelled {reason, cancelled_at} |

### Decision

**Option B: Graceful completion semantics with explicit compensation opt-in.**

### Cancellation Specification

**Triggering cancellation:**
```go
// SDK
runner.Cancel(ctx, instanceID, "user requested") error

// CLI
awis cancel <instance-id> [--reason="user requested"] [--compensate]
```

**Execution behavior on cancellation request:**

1. The runtime sets `workflow_instances.cancellation_requested = true` for the instance (add column to StateStore schema, see below).
2. Currently in-flight steps (those already dispatched to a runner) run to completion. The runtime does not kill their goroutines or processes. Their `StepCompleted` or `StepFailed` events are appended to the EventLog as normal.
3. After each step settles, the transition evaluator checks `cancellation_requested`. If true, no next step is activated regardless of transition conditions.
4. If `--compensate` is passed: instead of transitioning directly to `cancelled`, the instance transitions to `compensating` and the compensation plan runs. After compensation, status becomes `compensated`.
5. If `--compensate` is not passed (default): after in-flight steps settle, status transitions to `cancelled`.
6. All pending `wait_records` for the instance are deleted from the signal_inbox. A cancelled instance cannot receive signals.
7. `WorkflowCancelled {reason: string, cancelled_at: Timestamp}` is appended to the EventLog.

**Idempotency:**
- Cancel on a `completed`, `failed`, `cancelled`, `compensated`, or `compensation_failed` instance: returns no error; logs a warning: `"instance <id> is already in terminal state <status>; cancel is a no-op"`.
- Cancel on a `pending` instance (not yet started): transitions to `cancelled` immediately with no steps executed.

**StateStore schema addition:**
```sql
ALTER TABLE workflow_instances ADD COLUMN cancellation_requested INTEGER DEFAULT 0;
-- 0 = not requested; 1 = requested
```

This column is set to 1 when Cancel is called and cleared (or irrelevant) once the instance reaches a terminal status. It is the only schema change required by this blocker.

**Event sequence for a cancellation with one in-flight step:**
```
WorkflowStarted
StepStarted {step: "draft-entry"}
[Cancel called — cancellation_requested = 1]
StepCompleted {step: "draft-entry"}   ← in-flight step completes normally
WorkflowCancelled {reason: "user requested", cancelled_at: "..."}
```

### Justification

Graceful completion rather than immediate interrupt is correct for external-consistency reasons. A step that is writing to a file, calling an API, or appending to The Record cannot be safely interrupted mid-execution without risking partial writes or inconsistent external state. The workflow framework guarantees that every started step either completes or fails; it never leaves a step in an indeterminate state.

Not running compensation by default on cancellation is standard practice (Temporal, Cadence, Conductor all share this default). Cancellation means "stop going forward" — it is not necessarily a rollback request. When the application needs rollback on cancel, it passes `--compensate`. Making compensation opt-in prevents surprising rollback behavior when a developer simply wants to stop a workflow.

Deleting wait_records on cancellation prevents ghost signals from being delivered to a cancelled instance on a future scan, which would create confusing EventLog entries (a `SignalReceived` after a `WorkflowCancelled`).

### Affected Documents

- `AWIS_ARCHITECTURE_BLUEPRINT.md` §6 (WorkflowInstance schema): add `cancellation_requested` column
- `AWIS_ARCHITECTURE_BLUEPRINT.md` §8 (Execution Engine) and §9 (State Management): add cancellation behavior
- `AWIS_ARCHITECTURE_BLUEPRINT.md` §9 (Event Types): `WorkflowCancelled` payload is now `{reason: string, cancelled_at: Timestamp}`
- `AWIS_PRODUCT_REQUIREMENTS_INVESTIGATION.md` §CLI: `awis cancel` now has defined semantics and a `--compensate` flag

---

## BLOCKER 5 — FTS Ownership Boundary

### Root Cause

Two documents make contradictory claims about where OIP's full-text search index lives:

- `ENGINEERING_ARCHITECTURE_BLUEPRINT.md`: OIP owns `.decisions/.index/index.db` — a separate SQLite file
- `AWIS_ARCHITECTURE_BLUEPRINT.md` §10: references OIP storing its FTS5 index in "OIP namespace" — implying AWIS's runtime.db

If OIP's FTS index lives in AWIS's `runtime.db`, the StoragePort interface must expose either (a) FTS5 virtual table creation and query, or (b) a raw SQL passthrough — either of which breaks StoragePort's clean abstraction. If OIP owns its own SQLite, StoragePort remains clean but OIP must manage a second database file.

### Alternatives

**A — OIP owns its own application SQLite (`oip.db`).**

OIP's native step handlers (`RecordAppendHandler`, `IndexFTSHandler`) open and manage `oip.db` directly. AWIS's `runtime.db` holds only execution state. Two SQLite files exist: `runtime.db` (AWIS) and `oip.db` (OIP application).

**B — OIP's FTS index lives in AWIS's runtime.db via an extended StoragePort.**

StoragePort grows a `RawSQL(query string, params ...any) (rows, error)` or `FTS5Register(table, schema)` method. OIP calls these to create and query its FTS tables within the AWIS database.

**C — OIP's FTS index lives in AWIS's runtime.db as a fixed-schema AWIS feature.**

AWIS adds a "document store" feature with FTS5 to StoragePort as a first-class storage type. Applications register documents; AWIS handles the FTS indexing. No raw SQL; a defined API.

### Trade-off Analysis

| Criterion | A (OIP owns oip.db) | B (RawSQL extension) | C (AWIS doc store) |
|---|---|---|---|
| StoragePort cleanliness | Perfect | Breaks abstraction | Acceptable |
| OIP autonomy | Full | Constrained by AWIS schema | Constrained by AWIS API |
| P1 compliance (apps own logic) | Perfect | Acceptable | Acceptable |
| P9 compliance (two use cases) | N/A | Only OIP uses it (V1) | Only OIP uses it (V1) |
| AWIS complexity | Unchanged | Increases (RawSQL is dangerous) | Increases (new feature) |
| Backup story | Two files | One file | One file |
| OIP's upgrade path to vector search | OIP manages sqlite-vec | OIP can't without AWIS changes | Requires AWIS changes |

Option B (RawSQL) gives applications direct SQL access to the AWIS database — a serious abstraction violation that would allow applications to corrupt the EventLog or read another namespace's data. It's not acceptable under any framing.

Option C requires AWIS to build a document store feature with zero V1 use cases outside OIP. This violates P9. It also means OIP's schema is defined by the AWIS API, not by OIP — exactly the coupling P1 prohibits.

### Decision

**Option A: OIP owns its own application SQLite (`oip.db`). AWIS's StoragePort interface is unchanged.**

### FTS Boundary Specification

**AWIS's runtime.db holds:**
- `execution_events` (EventLog)
- `workflow_instances` (StateStore)
- `workflow_definitions` (WorkflowRegistry)
- `step_results_cache` (StepResultCache)
- `signal_inbox`
- `wait_records`
- `plugins`, `plugin_capabilities`

**OIP's oip.db holds:**
- `entries` — metadata about appended record entries (id, path, created_at, tags)
- `entries_fts` — FTS5 virtual table for full-text search over entry content
- `entry_vectors` — Float32Array BLOBs for cosine similarity search (semantic recall)

**Population mechanism:**
OIP's `RecordAppendHandler.Execute()` performs two writes in one operation:
1. Writes the `.md` file to `.decisions/entries/`
2. Opens `oip.db` and inserts the entry's metadata and content into `entries` and `entries_fts`

OIP's `IndexFTSHandler.Execute()` queries `oip.db` directly.

**Deployment:**
```
.decisions/
├── entries/          ← The Record (plain .md files, in git)
│   └── D-2026-07-02-001.md
└── .index/
    └── oip.db        ← OIP's application SQLite (git-ignored)

.awis/
└── runtime.db        ← AWIS's execution SQLite (git-ignored)
```

**Rebuild path:**
`oip rebuild-index` (or an `awis submit` of a re-indexing workflow) repopulates `oip.db` from `.decisions/entries/*.md`. This is the same principle as `awis rebuild-state` — the on-disk files (`.md`) are the source of truth; the index is a rebuildable cache.

**Cross-database queries:**
OIP's recall workflow (`fts-search` step → `semantic-rank` step) queries `oip.db`, not `runtime.db`. There is no cross-database join. The two databases are independent.

### Justification

P1 is unambiguous: "Applications own business logic; the runtime owns execution." The FTS index is OIP's business logic — it is a search index over OIP's domain data (decision entries). AWIS is the execution substrate; it has no more reason to own OIP's FTS index than it has reason to own OIP's `.md` files. Both are application data.

The "two files" concern (backup complexity) is addressed by the separation of concerns: `runtime.db` is ephemeral — it holds execution history that can be rebuilt from the EventLog. `oip.db` is also rebuildable from `.decisions/entries/`. The truly irreplaceable artifact (The Record) is the `.md` files, which are in git. A developer who backs up their git repo has everything necessary to reconstruct both SQLite databases.

### Affected Documents

- `AWIS_ARCHITECTURE_BLUEPRINT.md` §10 (OIP on AWIS): add explicit storage boundary statement
- `AWIS_ARCHITECTURE_BLUEPRINT.md` §9 (State Management): clarify that runtime.db holds execution state only; application databases are separate
- `ENGINEERING_ARCHITECTURE_BLUEPRINT.md`: OIP's `.decisions/.index/index.db` model is correct; path becomes `oip.db` per the deployment layout above (consistent with either document)

---

## BLOCKER 6 — Compensation vs. Append-Only Event Model

### Root Cause

The canonical OIP capture workflow YAML includes:
```yaml
compensation:
  steps:
    - step: append-to-record
      undo: oip.record.mark-draft
```

OIP Constitution Article 7 mandates: "The Record is corrected by appending truth, never by silently rewriting the past." Entries are write-once; corrections are new entries with `corrects:` reference. `oip.record.mark-draft` as written implies modifying an existing record entry — a direct violation.

### Alternatives

**A — Remove the compensation plan from `append-to-record` entirely.**

`append-to-record` is the final step in the capture workflow. No subsequent step can fail after it. The compensation plan on a final step is logically vacuous — it can never be triggered. Removing it eliminates the contradiction without changing any platform behavior.

**B — Replace `oip.record.mark-draft` with `oip.record.void-entry`: append a new entry marking the original as superseded.**

Constitutionally correct — the "undo" is a new append, not a modification. The original entry remains; a new entry is appended with `status: voided, corrects: <original-id>`. Operationally confusing — from a user's perspective, the compensation "undid" the record append; from a constitutional perspective, the record now contains both the original and a voiding entry.

**C — Redefine the AWIS compensation contract: compensation handlers must always be constitutional (append-only) operations, enforced at the platform level.**

Platform makes no attempt to enforce — it cannot know what OIP's handlers do internally. This is a documentation constraint, not an enforcement mechanism.

### Decision

**Option A: Remove the compensation plan from `append-to-record`.**

### Justification

The question to resolve is: "In the OIP capture workflow, when would compensation of the `append-to-record` step be triggered?"

Answer: Never. Compensation triggers when a workflow **fails after one or more steps have completed**. `append-to-record` is the **final step**. After it completes successfully, the workflow is `completed` — not `failed`. There is no subsequent step that can fail after it.

The compensation plan `{step: append-to-record, undo: oip.record.mark-draft}` requires:
1. `append-to-record` completes successfully (entry is in The Record)
2. A subsequent step fails (triggering compensation)
3. Compensation runs `oip.record.mark-draft` on the already-appended entry

Step 2 is impossible in the current workflow definition. `append-to-record` is a final step. There is no workflow that reaches this compensation handler. The compensation plan is dead code.

**Broader principle established:** The compensation plan on a step should be specified only if another step follows it in the happy path. A final step's compensation handler is never invoked; it should not be written.

**For the platform documentation:** The compensation feature is correctly specified. The YAML example simply misuses it by placing a compensation handler on a final step. The example is corrected; the feature is unchanged.

**Note on Option B:** If OIP's business logic ever requires undoing an appended Record entry (for example, if a future version adds a post-processing step that fails after the append), the correct implementation is `oip.record.void-entry` — a new append that constitutionally marks the original as voided. That handler would be added to OIP's step handler registry when the use case exists. The platform does not need to change.

### Affected Documents

- `AWIS_ARCHITECTURE_BLUEPRINT.md` §7 (YAML DSL example): remove the `compensation` block from the OIP workflow example
- No platform code changes; no interface changes; compensation feature is unaffected

---

## BLOCKER 7 — Parallel Step Definition (or Removal)

### Root Cause

`parallel` appears in the `StepType` enum in `AWIS_ARCHITECTURE_BLUEPRINT.md` §6:
```
type: StepType  // native | subprocess | plugin | intelligence | signal | parallel
```

It also appears in DSL Design Rules: "no cycles (unless parallel type)" — implying the type is handled. But `parallel` is never defined. No fields are specified. No semantics are documented. No example uses it. No V1 use case justifies it.

### Alternatives

**A — Remove `parallel` from the V1 StepType enum.** Parallel execution is achieved via the existing transition graph (fan-out transitions from one step to multiple steps, with a join-point step). The type enum becomes `native | subprocess | plugin | intelligence | signal`.

**B — Define `parallel` as a container step** with child step definitions embedded within it. The parallel step completes when all children complete (or any child fails). Requires: new fields in `Step`, new executor logic in the runtime, new validator logic for nested step definitions, documentation for a new concept.

**C — Define `parallel` as a routing decorator** that marks multiple independent transitions from the same source step as "all must complete before proceeding," distinct from the current "first matching transition fires" model. Changes transition evaluation semantics.

### Trade-off Analysis

Parallel execution in AWIS's V1 workflow graph is already achievable without a `parallel` type. The NeuroDashboard Go SDK example in §7 demonstrates this:

```go
for _, metric := range cfg.Metrics {
    b.AddStep(awis.Step{ID: fmt.Sprintf("compute-%s", m.Name), ...})
    b.AddTransition("ingest-data", fmt.Sprintf("compute-%s", m.Name))
}
```

Multiple transitions from `ingest-data` to `compute-metric-A`, `compute-metric-B`, etc. are all activated concurrently. The execution loop dispatches all of them. A join step reads all their outputs. This is parallel execution without a `parallel` StepType.

The P9 principle applies directly: the `parallel` StepType has zero V1 use cases that cannot be expressed with the existing transition model.

| Criterion | A (Remove) | B (Container step) | C (Routing decorator) |
|---|---|---|---|
| V1 use cases | Transition graph handles all V1 needs | None | None |
| Implementation effort | Zero | High | Medium |
| Specification effort | Zero | High | Medium |
| Correctness risk | Zero | High (nested step parsing) | Medium (transition evaluation changes) |
| Reversibility | Can add in V2 | Can add in V2 | Can add in V2 |

### Decision

**Option A: Remove `parallel` from the V1 StepType enum. Update the DSL Design Rule accordingly.**

### Justification

**P9 (no abstraction without two real use cases):** The `parallel` StepType has zero real use cases in V1. All V1 parallel execution needs are satisfied by fan-out transitions, which are already specified.

**Gall's Law:** Adding a complex container-step type to a V1 system that already achieves parallel execution through simpler means is an AWIS-specific illustration of Gall's Law's warning: don't design for complexity you haven't encountered yet.

**Reversibility:** This decision is easily reversed in V2. Adding `parallel` to the enum when a use case exists (and the transition-graph model proves insufficient) is a minor addition. Removing a `parallel` type that was shipped in V1 but is unused is harder — it creates migration questions for users who may have written workflows against it.

**Affected DSL rule:** "no cycles (unless parallel type)" → "no cycles (period)." Cyclic workflows are not supported in V1. A workflow that needs to loop should use a new instance submission from a step handler.

### Affected Documents

- `AWIS_ARCHITECTURE_BLUEPRINT.md` §6 (Step struct): StepType becomes `native | subprocess | plugin | intelligence | signal`
- `AWIS_ARCHITECTURE_BLUEPRINT.md` §7 (DSL Design Rules): "no cycles (unless parallel type)" → "no cycles; cyclic workflows must use new instance submission"
- `AWIS_ARCHITECTURE_BLUEPRINT.md` §7 (DSL Design Rules): add note: "Parallel execution is achieved via fan-out transitions: multiple transitions from one step activate all target steps concurrently. A downstream step that requires all fan-out results should check input availability in its condition."

---

## Architecture Updates Applied

The following changes are made to `AWIS_ARCHITECTURE_BLUEPRINT.md`. Each change is minimal — only what the blocker requires.

### Update 1 — Supersession Cross-Reference (Blocker 1)

Added to `AWIS_ARCHITECTURE_BLUEPRINT.md` §1 (Executive Summary), after the opening paragraph:

> **Implementation model for OIP:** OIP is implemented as an AWIS application. The `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` document describes the historical standalone OIP design produced before AWIS was specified. It is superseded by this blueprint. Implementation follows the AWIS-application model described in §10. The standalone document is retained as a historical record only.

### Update 2 — Expression Language Formal Grammar (Blocker 2)

Replaces the implicit expression usage in `AWIS_ARCHITECTURE_BLUEPRINT.md` §7 with the formal grammar defined in this document. The YAML example's condition `"steps['draft-entry'].status == 'fallback'"` is corrected to use dot-path notation: `"steps.draft-entry.status == 'fallback'"`. DSL Design Rules updated to reference the formal grammar.

### Update 3 — Signal Atomicity (Blocker 3)

Added to `AWIS_ARCHITECTURE_BLUEPRINT.md` §8 (Execution Engine) after the SIGNAL_SCAN step, and to §9 (Consistency Guarantees):

> **Signal delivery atomicity:** Signal delivery is a single database transaction across `signal_inbox`, `execution_events`, and `workflow_instances`. The transaction includes: (1) mark signal as delivered, guarded by `delivered_at IS NULL` idempotency check; (2) append `SignalReceived` to EventLog; (3) update instance status from `waiting` to `running`, increment version. All three writes commit atomically or not at all. The EventLog is authoritative: if `SignalReceived` exists in the EventLog for an instance and signal, the signal was delivered. Any StateStore inconsistency is recoverable via `awis rebuild-state`.

### Update 4 — Cancellation Semantics (Blocker 4)

New subsection added to `AWIS_ARCHITECTURE_BLUEPRINT.md` §8 (Execution Engine): **Cancellation Semantics.** Contains the full specification from Blocker 4 resolution above. `workflow_instances` schema updated with `cancellation_requested INTEGER DEFAULT 0`.

### Update 5 — FTS Storage Boundary (Blocker 5)

Added to `AWIS_ARCHITECTURE_BLUEPRINT.md` §10 (OIP on AWIS):

> **Storage boundary:** OIP maintains its own application SQLite database (`oip.db`) for full-text and semantic search over decision entries. AWIS's `runtime.db` holds execution state only (EventLog, StateStore, WorkflowRegistry, StepResultCache, signal_inbox). OIP's native step handlers (`RecordAppendHandler`, `IndexFTSHandler`) open and write to `oip.db` directly. There is no cross-database query between `runtime.db` and `oip.db`. OIP's index is rebuildable from `.decisions/entries/*.md` via the `rebuild-index` workflow.

Added to `AWIS_ARCHITECTURE_BLUEPRINT.md` §9 (State Management):

> **Application databases:** The StoragePort interface covers AWIS execution state only. Applications that need application-specific databases (FTS indexes, domain entity storage, vector stores) manage their own SQLite files. These are application concerns, not platform concerns. AWIS does not provide a generic document store or raw SQL passthrough.

### Update 6 — Compensation Example Corrected (Blocker 6)

The `compensation` block is removed from the OIP capture workflow YAML example in §7. The `append-to-record` step is correctly identified as a final step with no compensation handler needed.

A new explanatory note is added to the compensation specification in §8:

> **When to specify compensation:** Compensation handlers are specified only on steps that have downstream steps following them. A final step's compensation handler is never invoked — the workflow cannot fail after the last step completes. Specifying compensation on a final step is harmless but unnecessary.

### Update 7 — StepType Corrected (Blocker 7)

`AWIS_ARCHITECTURE_BLUEPRINT.md` §6 StepType updated: `native | subprocess | plugin | intelligence | signal` (no `parallel`).

DSL Design Rule updated: "no cycles (period); cyclic workflows must use new instance submission from a step handler."

Note added: "Parallel execution is achieved via fan-out transitions. Multiple transitions from one step activate all target steps concurrently within the execution loop."

---

## Impacted Documents

| Document | Changes | Status |
|---|---|---|
| `AWIS_ARCHITECTURE_BLUEPRINT.md` | 7 targeted additions/corrections per above | **Updated** |
| `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` | Supersession header added | **Updated** |
| `AWIS_PRODUCT_REQUIREMENTS_INVESTIGATION.md` | `awis cancel` semantics + `--compensate` flag | **Note added** |
| `OIP_CONSTITUTION.md` | No change — compensation resolution is consistent with Article 7 | Unchanged |
| `AWIS_CANONICAL_SPECIFICATION_TRIBUNAL_REPORT.md` | Reference only — this document resolves its findings | Unchanged |

---

## Remaining Risks

All 7 critical blockers are resolved. The following non-critical items from the Tribunal Report remain open — they are **not blockers** for the PRD but are noted for completeness:

| Risk | Status | Notes |
|---|---|---|
| `classify` in IntelligencePort (no V1 use case) | Open | Stays in V1 as a non-callable placeholder; costs nothing to implement with a single-line NullAdapter method; removed from interface in V2 if still no use case |
| CapabilityRouter full complexity | Open | The V1 implementation may be simplified internally while maintaining the public routing spec; implementation detail, not specification blocker |
| Plugin system V1 justification | Open | PRD must state the strategic rationale for including the plugin system in V1 |
| `awis recall` synthesis vs FTS-first | Open | PRD to specify that V1 `awis recall` returns FTS results; synthesis is enabled via `--synthesize` flag when intelligence is available |
| V1 timeline inconsistency (6 vs 8 weeks) | Open | PRD defines the authoritative timeline; all prior timeline references in blueprints are estimates, not commitments |
| Plugin idle state (killed vs suspended) | Open | Implementation decision; recommendation is process kill (simpler, cross-platform); PRD should note expected respawn latency |

None of these prevent PRD generation.

---

## Final Answer

**Is AWIS Architecture v1.0 now sufficiently complete, internally consistent, and frozen to proceed with generating the canonical PRD?**

Yes.

All seven critical blockers are resolved with precise specifications:

1. **OIP Architecture Schism** → Resolved: `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` is explicitly superseded; OIP is an AWIS application.
2. **Expression Language** → Resolved: Two formal grammars specified; prohibited features listed; dot-path notation canonicalized.
3. **Signal Atomicity** → Resolved: Single SQLite transaction across all three tables; EventLog as tiebreaker invariant.
4. **Cancellation Semantics** → Resolved: Graceful in-flight completion; no compensation by default; signal inbox cleanup; idempotent on terminal states; `cancellation_requested` column added.
5. **FTS Ownership** → Resolved: OIP owns `oip.db`; AWIS owns `runtime.db`; StoragePort interface unchanged.
6. **Compensation vs. Append-Only** → Resolved: Compensation block removed from final step; Article 7 is now unchallenged; compensation feature is unchanged.
7. **Parallel StepType** → Resolved: Removed from V1 enum; parallel execution via fan-out transitions; V1 DSL rule updated.

No new features were introduced. No architectural redesign occurred. Every decision is the smallest correct solution.

---

**Architecture Frozen — Ready for Canonical PRD Generation.**
