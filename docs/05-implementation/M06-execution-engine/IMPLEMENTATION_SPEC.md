# M06 — Implementation Spec (compiled 2026-07-03 from IMP §27.M6 + Blueprint §5 L1/L2, §8, §10 + Finalization B4/B6/B7 + Constitution Art. 32 + F-2, per IKB §4 row)

## Objective (IMP §27.M6)
The pull loop (SCAN→CLAIM→DISPATCH→SETTLE per Blueprint §8), NativeRunner + IntelligenceRunner,
transition evaluation incl. fan-out within `max_parallel`, RetryPolicy with backoff (ADJ-6),
fallback activation, compensation (reverse order, own retries, `compensation_failed`),
cancellation FSM verbatim from Finalization B4, idempotency-key check against StepResultCache,
structured JSON logging, **[F-2]** SCAN_TRIGGERABLE + DomainEvent ingestion + `domain_events`
(TTL 7d). `Out:` `internal/engine`, `internal/runner/{native,intelligence}`, migration 0002.
**Merge bar:** CI + race + **E1** green. **Repo after:** native workflows execute end-to-end with
retry/fallback/compensation/cancellation.

## Adjudications in force (entry, founder-approved 2026-07-03)
ADJ-6 (RetryPolicy full §8 shape; defaults 1s/30s), ADJ-7 (IntelReq.Required), CONTRA-10
(inputs = template value-map; §7+PRD§18r1+FR-WD-05 outrank §6's schema tag). Applied to TDS-02 §7
+ core/step.go at materialization. Plus inherited: CONTRA-5 (WorkflowCancelled `{reason}` only,
cancelled_at derived — FR-WE-10 reads through it), CONTRA-6 (claims, never a status), EDR-005
(engine assigns sequence_num), EDR-006 (claim release), EDR-007 (projection rules = forward-path
spec), EDR-008/009/010 (intelligence + expression semantics).

## Tasks

### T1 — Engine core (`internal/engine`): tick pipeline + submit + event emission + forward projection
- `Config{TickInterval time.Duration /*default 100ms*/; MaxParallelSteps int /*default 4, §8*/; WorkerID string; Clock func() time.Time}`.
- `New(storage core.StoragePort, runners map[core.StepType]Runner, intel *intelligence.Dispatcher, cfg Config, logger *slog.Logger) *Engine`.
- `Runner` interface (engine-owned): `Run(ctx context.Context, sc core.StepContext, step core.Step) (core.StepResult, *core.StepError)` — all runners produce `StepResult{outputs} | StepError` (Blueprint §5 L2). Unregistered StepType at dispatch ⇒ StepError `{code:"runner_unavailable"}` (signal=M07, subprocess=M11, plugin=M12 register later).
- **Submit path** (`Submit(ctx, definitionID string, version core.SemVer, inputs map[string]any) (core.InstanceID, error)`): load def from registry (empty version ⇒ latest by semver ordering among ListWorkflows results? NO — registry Get requires version; latest-selection is M08's. Submit takes explicit version at M06); validate via `validate.Validate` (any Issue ⇒ reject, FR-WD-04); create instance UUID; emit `WorkflowStarted{inputs}` (seq 1); upsert instance per EDR-007 row (running, StartedAt=emitted_at, variables={"inputs":inputs}, current_steps=[]); activate InitialStep.
- **Event emission**: engine assigns `sequence_num` (EDR-005) — maintain per-instance next-seq in memory guarded by storage's monotonicity check (ErrSequenceViolation ⇒ reload MAX+1, retry once); envelope schema_version=1; emitted_at from Config.Clock.
- **Forward projection (`applyEvent`)**: MUST mirror EDR-007 §2 status-map table EXACTLY (that doc is the spec; G2 verifies engine-vs-rebuild equivalence on event-sequence fixtures). Every emit is: append event THEN upsert projection (optimistic version from the read copy). Terminal upserts release claims (EDR-006, storage does it in-tx).
- **Tick** (`Tick(ctx) error`, plus `Run(ctx)` loop at TickInterval): stages in §8 order — SCAN_RUNNABLE (instances status=running with activatable steps — see T3 activation model), SCAN_TRIGGERABLE (T5), CLAIM (storage.ClaimStep at-most-once; loser skips), DISPATCH (bounded by MaxParallelSteps, goroutines + errgroup-style wait — in-flight steps finish within the tick call for V1 single-process determinism), SETTLE (emit result event; evaluate transitions; upsert; T3/T4), SIGNAL_SCAN (stub hook, M07 — document).
- **StepContext assembly**: Inputs = step.Inputs template-resolved (CONTRA-10): walk the map; every string leaf → `expr.ParseTemplate` + `Resolve(env)`; env from instance Variables (EDR-007 shape) + StepStatus map (derived: completed steps = variables keys minus "inputs" — NO: track statuses in engine per instance from events... simplest correct: statuses map built during SETTLE bookkeeping held on the engine's in-memory instance view; rebuild-safe because it is derivable from EventLog. Keep an engine-side `stepStatus map[string]string` per active instance, seeded on scan from events? V1 single-process: derive from Variables (completed=has outputs) + CurrentSteps (running). Document precisely in code.) Resolve Warnings → `logger.Warn` per FR-WD-05 (M05 HANDOFF obligation).
- **Idempotency** (§5 L2): key = `IdempotencyKey(instance_id + "/" + step_id + "/" + attempt)`; before Run → GetCachedResult; hit ⇒ use cached result WITHOUT re-executing; miss ⇒ Run then CacheResult (TTL: 24h engine default, config-able).
- **Structured JSON logging**: `log/slog` JSONHandler; every emit/claim/dispatch/settle logs with instance_id/step_id/event_type keys. Package doc explains the tick pipeline (IMP DoD).

### T2 — Runners (`internal/runner/native`, `internal/runner/intelligence`)
- **NativeRunner**: registry `map[string]core.StepHandler` (`Register(h)`, keyed h.ID()); Run looks up step.Handler, calls Execute(sc) with per-step timeout (step.Timeout via context deadline); handler Go error ⇒ StepError{code:"handler_error", message}; missing handler ⇒ StepError{code:"handler_not_found"} (registration-time check is M08; runtime miss must still be typed).
- **IntelligenceRunner**: for step.Type=intelligence; builds the request from resolved Inputs
  (capability=step.Intelligence.Capability): `draft` → DraftRequest{Context: inputs["context"] stringified, Schema: step.Outputs}; `synthesize` → SynthesisRequest{Query: inputs["query"], Entries: inputs["entries"], MaxLen}; calls `dispatcher.Draft/Synthesize(ctx, *step.Intelligence, step.Intelligence.Required, req)` (ADJ-7 closes M04's parameter deferral).
  - `FallbackSignal` error ⇒ StepError{code:"capability_fallback"} — the ENGINE (not runner) sees this code and activates step.Fallback via the T4 fallback path (Blueprint §17: router/runner reports, engine activates).
  - `CapabilityUnavailableError` ⇒ StepError{code:"capability_unavailable"} (fails per FR-IL-07).
  - Success: outputs = DraftResponse.Output / {"text": SynthesisResponse.Text}; **Usage → StepCompleted payload fields `{adapter, model, tokens_used}` (FR-IL-09)** — Runner returns usage via StepResult outputs? NO — keep StepResult frozen: runner returns (StepResult, usage) via an engine-internal side-channel: the intelligence Runner implements an extended interface `usageReporter` the engine type-asserts. Document.
  - Runtime assembly rule (M04 HANDOFF obligation): the production Registration list places NullAdapter LAST in fallback_chain; add `intelligence.ValidateChain(regs, chain) error` construction-time check (null present ⇒ must be last) — lives in `internal/intelligence` (sanctioned +1 exported func, M04-V1 advisory).
- classify/embed: NO dispatch (FR-IL-10, CONTRA-3) — unknown capability ⇒ StepError{code:"capability_unknown"}.

### T3 — Transition evaluation + fan-out (Blueprint §8 SETTLE, B7) — EDR-011
On StepCompleted of step S: evaluate ALL transitions with from=S in definition order —
condition absent ⇒ fires; condition present ⇒ `expr.ParseCondition` (parsed at submit, cached) `.Eval(env)`; on_error transitions fire ONLY on StepFailed-final (B: on_error=true + step failed terminally). ALL firing transitions activate their targets (fan-out, B7/FR-WD-11 — never first-match).
**Join gate (§5 L193/§8 "all upstream deps completed"):** target T activates iff every DISTINCT `from` of T's inbound transitions has status completed AND ≥1 inbound transition fired. Steps whose upstream was skipped stall (author semantics per B7 note; EDR-011 documents). Activation: add to current_steps via StepStarted at claim time (EDR-007: StepStarted adds step_id). Concurrency: activated steps dispatch in the same tick up to MaxParallelSteps; excess wait for next tick (claims make this safe).
Final-step completion: when a completed step ∈ FinalSteps and no other current_steps remain ⇒ emit `WorkflowCompleted{outputs, duration_ms}` — outputs = map[final_step_id]→its outputs map (EDR-011); duration_ms = emitted_at − StartedAt.

### T4 — Retry, fallback, failure, compensation, cancellation (Phase B)
- **Retry (ADJ-6)**: on runner StepError: attempt < policy.Attempts AND (RetryableErrors empty OR code ∈ list) ⇒ emit StepFailed{retrying:true}, schedule re-dispatch after backoff(attempt): immediate=0; linear=initial*n capped max; exponential=initial*2^(n−1) capped max (defaults 1s/30s when fields absent; no policy ⇒ 1 attempt). V1 scheduling: next eligible tick ≥ now+delay (engine keeps per-step nextAttemptAt; claims already used — a retry re-dispatch must NOT re-claim (claim exists); track attempt in engine + StepStarted{attempt}).
- **Fallback**: attempts exhausted (or capability_fallback): step.Fallback≠"" ⇒ emit StepFailed{retrying:false} THEN StepFallbackActivated{step_id, fallback_step_id, reason} (EDR-007: removes step_id from current_steps) ⇒ activate fallback step (StepStarted at its claim). Fallback steps' own transitions carry the flow onward.
- **Workflow failure**: exhausted + no fallback ⇒ StepFailed{retrying:false} + `WorkflowFailed{step_id, error}` ⇒ status failed. If def.Compensation≠nil AND ≥1 step completed ⇒ `WorkflowCompensating{from_step}` then run plan steps REVERSE order, each = an undo HandlerRef run through NativeRunner with its own retry (plan-step retry policy: engine default 1 attempt V1 unless CompensationStep carries one — shape decision below), skipping plan steps whose step_id never completed; all ok ⇒ `WorkflowCompensated{}`; any undo exhausts ⇒ `WorkflowCompensationFailed{step_id, error}` (ADJ-5 status; "AuditLog always records it" ⇒ slog.Error at M06, audit table is M07/F-4 — document).
- **Core shape completions (M06-owned)**: `CompensationPlan{Steps []CompensationStep}`, `CompensationStep{StepID string; UndoHandler HandlerRef; Retry *RetryPolicy?}` — Blueprint §8 L537–539 shape + "own retry policies" sentence sanctions optional Retry; `CompensationRef{Handler HandlerRef}`; `Logger` interface `{Info/Warn/Error(msg string, args ...any)}` (slog-shaped); StepContext.Logger wired.
- **Cancellation (B4 VERBATIM — the FSM)**: `Cancel(ctx, instanceID, reason string, compensate bool) error`:
  terminal status ⇒ nil + slog.Warn "instance <id> is already in terminal state <status>; cancel is a no-op" (B4 verbatim);
  pending ⇒ immediately WorkflowCancelled{reason} → cancelled;
  else set cancellation_requested=1 (storage helper — see T6) ⇒ in-flight steps finish normally (their events append normally); after each settle the transition evaluator checks the flag: set ⇒ NO next step activates regardless of conditions; when current_steps drains: compensate=false ⇒ WorkflowCancelled{reason} → cancelled; compensate=true ⇒ WorkflowCompensating → plan → compensated (or compensation_failed). wait_records deletion = M07 (none exist; document). Event-sequence test MUST reproduce B4's fixture: WorkflowStarted, StepStarted, [cancel], StepCompleted, WorkflowCancelled.

### T5 — F-2: Triggers + domain_events (migration 0002)
- `migrations/0002_domain_events.sql`: `domain_events(event_id TEXT PK, namespace TEXT NOT NULL, event_type TEXT NOT NULL, source TEXT NOT NULL, payload TEXT NOT NULL, emitted_at TEXT NOT NULL, consumed_at TEXT)` + idx (namespace, event_type, emitted_at). Blueprint §10 DomainEvent shape verbatim. N−1 fixture test updated. (Numbering per F-2/F-4 renumber: 0002=domain_events/M06; signals+audit renumber to 0003/0004 at M07 — record in TRACEABILITY.)
- `core.DomainEvent` struct (§10 verbatim; M06-owned completion; NOT a G1 format — additive CONTRA-4-class table) + StoragePort? NO — StoragePort is frozen 12 methods. Trigger storage = engine-internal store interface `TriggerStore{InsertDomainEvent; ListUnconsumed(namespace); MarkConsumed; PruneExpired(ttl)}` implemented by SQLiteStorage as ADDITIVE methods (not part of the frozen 12; document: StoragePort untouched, SQLite adapter grows engine-facing methods — same pattern as ExposedDB test accessor).
- `Ingest(ctx, ev core.DomainEvent) error` on the engine (SDK TriggerAPI intake = M08/F-2).
- SCAN_TRIGGERABLE: unconsumed domain events × registered defs with event triggers: match `config["event"] == ev.event_type` (+ namespace); `config["filter"]` string ⇒ ParseCondition.Eval with Env{Event: payload} (event scope — M05); match ⇒ Submit (inputs = event payload? Blueprint silent ⇒ EDR-011: inputs = `{"event": payload}`… NO: keep inputs = payload verbatim? DECISION (EDR-011): submitted inputs = DomainEvent.payload map — smallest reading of "trigger fires workflow" FR-WE-13; document) + MarkConsumed same logical flow (consume even if filter false? NO: consume only on fire OR TTL expiry; an event matching no trigger stays until TTL prune. Multiple defs may match one event: all fire; consumed when ≥1 fired). PruneExpired(7d default) each tick (cheap indexed delete).

### T6 — E1 gate + fixtures + cancellation storage helper
- `SetCancellationRequested(instanceID) / cancellation flag read` — additive SQLiteStorage methods + `cancellation_requested` read into engine scans (column exists since 0001).
- **AWIS-E1 (Constitution Art. 32)**: `make e1` ⇒ `go test ./internal/engine/... -run TestE1 -count=1` — E2E tests that run complete workflows (fan-out, retry, fallback, compensation, cancellation, trigger ingestion) with intelligence ENTIRELY ABSENT (no dispatcher configured / nil) proving the platform functions as a system of record+coordination with zero AI. Intelligence steps in E1 workflows take the fallback path (required=false). CI e1 job already calls make e1 (M00). PERMANENT gate.
- **Event-sequence fixtures**: table-driven E2E tests asserting the EXACT ordered (event_type, step_id) sequence for: happy path linear; fan-out+join; retry-then-success; retry-exhaust→fallback; failure→compensation (reverse order); compensation failure; B4 cancellation fixture verbatim; pending-cancel. THEN: for each fixture, wipe + RebuildState ⇒ projection byte-equal to forward-path row (G2 evidence: engine mirrors EDR-007).

## Scope walls
- StoragePort's frozen 12 methods UNTOUCHED (additive adapter methods only). sdk/ UNTOUCHED (M08 owns surface; aliases for completed shapes only if a placeholder was already aliased). No YAML (M10), no CLI (M14), no signals/wait_records (M07 — signal steps ⇒ runner_unavailable), no subprocess/plugin runners (M11/M12), no audit table (M07/F-4).
- `internal/core` edits: ONLY the M06-owned completions (CompensationPlan/CompensationStep/CompensationRef/Logger/DomainEvent) + the ADJ-6/7 fields (already applied at materialization).
- EventLog payloads: EXACTLY the frozen TDS-01 §2 schemas as amended by **ADJ-8**
  (founder-approved 2026-07-03, CONTRA-11): StepCompleted carries optional
  `adapter/model/tokens_used` for intelligence steps only (NullAdapter path writes
  "null"/"null"/0 via the runner's usage side-channel); all other payloads verbatim.

## Acceptance criteria (IMP §27.M6)
- AC-1: AWIS-E1 gate LIVE in CI (make e1 runs real tests) and green.
- AC-2: every Blueprint §9 event type emitted correctly (12/12 exercised across fixtures; forward projection byte-equal to RebuildState per fixture).
- AC-3: `-race` clean over internal/engine + runners.
- AC-4: cancellation event sequence matches the Finalization B4 fixture exactly.
- AC-5: retry/backoff (ADJ-6 defaults), fallback, compensation (reverse order, own retries, compensation_failed), fan-out within MaxParallelSteps all fixture-tested.
- AC-6: F-2: DomainEvent ingest→trigger match→submit works; domain_events TTL prune; migration 0002 + N−1 fixture green.
- AC-7: engine package doc explains the tick pipeline (IMP DoD); `make build test lint race contract` green.
