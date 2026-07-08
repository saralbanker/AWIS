# M06 — Validation Checklist (all binary; all must pass)
Verified by awis-verifier card M06-V1r on clean clone at HEAD `01d2f5c`: **15/15 PASS**, no fix
cycle required (third consecutive zero-fix-cycle milestone). 183 total PASS across internal/...;
E1 gate 8/8; 31 forward≡rebuild equivalence call sites; zero TODO/FIXME in engine source.

- [x] Clean clone: `make build test lint race contract` exit 0; `go test -race ./internal/engine/... ./internal/runner/... -count=1` exit 0
- [x] `make e1` runs REAL engine E2E tests (no placeholder echo) with intelligence entirely absent and passes; CI e1 job unchanged and wired (Constitution Art. 32; PERMANENT) — 8 TestE1_* incl. intelligence-TYPE-step-degrades-to-fallback
- [x] All 12 Blueprint §9 event types emitted across fixtures; every payload matches TDS-01 §2 as amended (ADJ-8 usage triple on intelligence StepCompleted only; native StepCompleted lacks the keys — asserted both ways). SignalReceived covered at projection level per EDR-011 §8 (forward site is M07's)
- [x] For every event-sequence fixture: forward-path projection row == RebuildState row (reflect.DeepEqual; EDR-007 mirror proven; 31 call sites)
- [x] Cancellation: B4 event-sequence fixture reproduced EXACTLY (WorkflowStarted, StepStarted, [cancel], StepCompleted, WorkflowCancelled{reason}); terminal-state cancel = warning + no-op (B4 verbatim string asserted); pending cancel → immediate cancelled; --compensate path → compensating → compensated; cancel-during-retry-wait → StepFailed{retrying:false, code=cancelled}
- [x] Retry: backoff immediate/linear/exponential each tested (ADJ-6 defaults 1s/30s when absent); retryable_errors filtering; attempt counts in StepStarted/StepFailed payloads
- [x] Fallback: exhausted-retries → StepFailed{retrying:false} + StepFallbackActivated + fallback executes; intelligence FallbackSignal (required=false) same path; required=true → capability_unavailable → workflow failure path
- [x] Compensation: reverse order proven; undo-handler own retry; skips never-completed steps; failure → WorkflowCompensationFailed + compensation_failed status + slog.Error
- [x] Fan-out: N parallel targets activate concurrently bounded by MaxParallelSteps; join gate (all distinct upstream froms completed + ≥1 fired) tested incl. the stall case documented in EDR-011
- [x] Idempotency: cached result short-circuits re-execution (fake handler counts calls); key = instance/step/attempt
- [x] F-2: migration 0002_domain_events (TTL 7d) + N−1 fixture green; Ingest→match(event_type+namespace+filter)→Submit(inputs=payload VERBATIM, asserted deep-equal); filter uses event scope via M05; unmatched events survive to TTL prune; multi-def match fires all, consumes once
- [x] StoragePort interface diff vs main: ZERO (additive SQLiteStorage methods only — cancellation.go, triggers.go); sdk/ zero-diff; core diff = step.go/workflow.go/event.go only (ADJ-6/7 fields + CompensationPlan/Step/Ref, Logger, DomainEvent)
- [x] Signal-type step dispatch → StepError{runner_unavailable} (M07 owns); no wait_records/signal code (grep empty)
- [x] Engine package doc explains the tick pipeline (IMP DoD); structured slog JSON on emit/claim/dispatch/settle
- [x] edr-011 committed (§1–8; §8 added at C2 dispatch: terminal-failure routing order); ADJ-6/7/8 + CONTRA-10 recorded in TDS-01/02 + gate log + module TRACEABILITY; global DoD (IMP §24) items 1–7 (macOS CI leg pending remote)
