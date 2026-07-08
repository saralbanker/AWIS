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

## 8. Terminal-failure routing order + failure-driven activation (added at C2 dispatch)

Frozen text: §8 L530 explicitly orders fallback ("If the step has a `fallback`, the fallback
step is activated. If no fallback: workflow transitions to `failed` and compensation begins")
but never orders §6 L301's `on_error` transitions relative to it — on_error would be dead code
if failure always went fallback-or-workflow-failure.

**Rule:** on terminal step failure (attempts exhausted, or capability_fallback signal):
(1) `step.fallback` ≠ "" ⇒ fallback path (L530 is explicit — fallback wins);
(2) else if ≥1 `on_error` transition from the failed step fires (condition absent = fires;
present = Eval) ⇒ StepFailed{retrying:false} then those targets activate (same all-that-fire
fan-out rule; no WorkflowFailed — the definition declared an error route);
(3) else StepFailed{retrying:false} + WorkflowFailed + compensation per §8.

**Activation bookkeeping:** failure-driven activations (fallback targets, on_error targets)
bypass the join gate — they are DIRECT activations recorded in an engine in-memory pending set
consumed by the next scan (V1 single-process). On restart the set is reconstructible from the
event log (StepFallbackActivated / terminal StepFailed events whose routed targets have no
later StepStarted); V1 documents this recovery derivation without exercising multi-process
restart. Rebuild resets `cancellation_requested` to 0 (EDR-007) — a crash mid-cancellation
loses the request; the caller re-issues Cancel (documented limitation).

**SignalReceived at M06:** the engine has no emission site (signals are M07). The 12/12
event-type coverage item is satisfied for SignalReceived at projection level (direct append +
forward/rebuild equivalence fixture); the forward emission site arrives with M07.

## Why not CONTRAs

Every rule fills a mechanism the frozen text names but does not specify; none touches a frozen
format beyond the founder-approved ADJ-6/7/8. All are reversible pre-G2 and are exercised by
the M06 event-sequence fixtures G2 will review.
