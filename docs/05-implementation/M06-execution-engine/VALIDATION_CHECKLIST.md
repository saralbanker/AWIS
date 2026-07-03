# M06 — Validation Checklist (all binary; all must pass)

- [ ] Clean clone: `make build test lint race contract` exit 0; `go test -race ./internal/engine/... ./internal/runner/... -count=1` exit 0
- [ ] `make e1` runs REAL engine E2E tests (no placeholder echo) with intelligence entirely absent and passes; CI e1 job unchanged and wired (Constitution Art. 32; PERMANENT)
- [ ] All 12 Blueprint §9 event types emitted across fixtures; every payload matches TDS-01 §2 as amended (ADJ-8 usage triple on intelligence StepCompleted only)
- [ ] For every event-sequence fixture: forward-path projection row == RebuildState row (byte-equal comparison; EDR-007 mirror proven)
- [ ] Cancellation: B4 event-sequence fixture reproduced EXACTLY (WorkflowStarted, StepStarted, [cancel], StepCompleted, WorkflowCancelled{reason}); terminal-state cancel = warning + no-op (B4 verbatim string); pending cancel → immediate cancelled; --compensate path → compensating → compensated
- [ ] Retry: backoff immediate/linear/exponential each tested (ADJ-6 defaults 1s/30s when absent); retryable_errors filtering; attempt counts in StepStarted/StepFailed payloads
- [ ] Fallback: exhausted-retries → StepFailed{retrying:false} + StepFallbackActivated + fallback executes; intelligence FallbackSignal (required=false) same path; required=true → capability_unavailable → workflow failure path
- [ ] Compensation: reverse order proven; undo-handler own retry; skips never-completed steps; failure → WorkflowCompensationFailed + compensation_failed status + slog.Error
- [ ] Fan-out: N parallel targets activate concurrently bounded by MaxParallelSteps; join gate (all distinct upstream froms completed + ≥1 fired) tested incl. the stall case documented in EDR-011
- [ ] Idempotency: cached result short-circuits re-execution (fake handler counts calls); key = instance/step/attempt
- [ ] F-2: migration 0002_domain_events (TTL 7d) + N−1 fixture green; Ingest→match(event_type+namespace+filter)→Submit(inputs=payload); filter uses event scope via M05; unmatched events survive to TTL prune; multi-def match fires all
- [ ] StoragePort interface diff vs main: ZERO (additive SQLiteStorage methods only); sdk/ zero-func; core diff = ADJ-6/7 fields + M06-owned shapes (CompensationPlan/Step/Ref, Logger, DomainEvent) only
- [ ] Signal-type step dispatch → StepError{runner_unavailable} (M07 owns); no wait_records/signal code
- [ ] Engine package doc explains the tick pipeline (IMP DoD); structured slog JSON on emit/claim/dispatch/settle
- [ ] edr-011 committed; ADJ-6/7/8 + CONTRA-10 recorded in TDS-01/02 + gate log + module TRACEABILITY; global DoD (IMP §24) items 1–7 (macOS CI leg pending remote)
