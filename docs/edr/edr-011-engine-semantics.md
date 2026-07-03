# EDR-011 — Engine Semantics Beyond the Frozen Text

**Status:** Reversible — pre-G2 (implementation decisions; G2 reviews with the event-sequence fixtures)
**Authored:** M06 materialization (2026-07-03, CE)
**Coordinates:** Blueprint §5 L189–226, §8, §10 · Finalization B4/B6/B7 · EDR-005/006/007 · ADJ-6/7/8 · CONTRA-10
**Downstream obligation:** M07 replaces the SIGNAL_SCAN stub and owns waiting-status semantics; M08 wraps Submit/Cancel/Ingest; M09 drives the engine through Config.Clock; G2 reviews every rule below against the running engine.

---

## 1. Join gate (fan-out convergence)

Frozen text: §5 L193 / §8 SCAN_RUNNABLE say a step is runnable when "all upstream deps
completed"; B7 says fan-out transitions are THE parallel mechanism and advises join steps to
"check input availability in its condition."

**Rule:** target step T activates iff (a) every DISTINCT `from` step among T's inbound
transitions has completed, AND (b) at least one inbound transition fired (condition true or
absent). Consequences: a join whose upstream branch was conditionally skipped **stalls** (author
semantics — B7's advisory note is the mitigation); multiple firing edges activate T once
(step_claims at-most-once dedups, CONTRA-6/EDR-006). Steps with zero inbound transitions
activate only as InitialStep or fallback targets.

## 2. WorkflowCompleted payload

`outputs` = `map[final_step_id] → that step's outputs map` for every FinalStep that completed
(frozen text says only "final outputs"). `duration_ms` = emitted_at − StartedAt. Projection
ignores both (EDR-007).

## 3. Retry mechanics

Claims persist across attempts: the FIRST activation claims (ClaimStep); retries re-dispatch
under the existing claim with `attempt` incremented in StepStarted/StepFailed payloads.
Backoff: immediate=0; linear=initial×n; exponential=initial×2^(n−1); capped at max; ADJ-6
defaults 1s/30s when fields absent; no policy ⇒ single attempt. Delay is enforced as
"not before now+delay" against the next eligible tick (V1 tick-quantized, not sleep-exact).

## 4. Triggers (F-2)

Trigger-submitted instances receive `inputs = DomainEvent.payload` verbatim (smallest reading
of FR-WE-13). Matching: trigger.type=event AND config["event"]==ev.event_type AND
namespace match AND (no filter OR filter Eval true with Env{Event: payload}). A domain event is
marked consumed when ≥1 workflow fired from it; unmatched events survive until the 7-day TTL
prune (§10 L724). Multiple matching definitions all fire from one event.

## 5. Deferred-subsystem behavior

Signal-type steps dispatch to StepError{code:"runner_unavailable"} until M07 registers the real
behavior; same code for subprocess (M11) and plugin (M12) types. WorkflowCompensationFailed is
slog.Error'd at M06; the audit_log write site arrives with M07's 0004 (F-4).

## 6. Compensation-step retries

`CompensationStep{step_id, undo_handler, retry?}` — §8 L537–539 shape plus an optional
per-step RetryPolicy sanctioned by §8's "Compensation steps have their own retry policies."
Absent ⇒ single attempt.

## 7. Migration renumbering (F-2/F-4)

0002 = domain_events (M06, F-2). 0003 = signals, 0004 = audit_log (M07, F-4). 0005 = plugins
(M12). 0006 = recall FTS (M17). IMP §14's pre-amendment table is overridden by the amendments
("amendments override the IMP text they amend", IKB §3).

## Why not CONTRAs

Every rule fills a mechanism the frozen text names but does not specify; none touches a frozen
format beyond the founder-approved ADJ-6/7/8. All are reversible pre-G2 and are exercised by
the M06 event-sequence fixtures G2 will review.
