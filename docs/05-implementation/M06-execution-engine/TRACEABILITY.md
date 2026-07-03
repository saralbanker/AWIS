# M06 — Traceability

| Task | Source (verbatim coordinate) | Requirement |
|---|---|---|
| T1 tick pipeline | Blueprint §8 L484–510 (six stages); §5 L189–207 (L1 loop); §5 L211–226 (L2 runners, idempotency) | FR-WE-01 (submit), NFR via §20 |
| T1 forward projection | EDR-007 §2 status map (= TDS-01 replay statements) | NFR-R-03 equivalence |
| T2 NativeRunner | §5 L2 table row 1; core.StepHandler (Blueprint §12) | — |
| T2 IntelligenceRunner | §5 L2 row 4; §17 tree via M04 Dispatcher; FR-IL-09 (ADJ-8) | FR-IL-05..09 |
| T3 transitions/fan-out | §8 SETTLE; B7 (fan-out is THE parallel mechanism); §5 L193 join phrase | FR-WD-11/12 |
| T4 retry/backoff | §8 L518–530 + ADJ-6 | — |
| T4 fallback | §8 L530; §17 bottom | FR-IL-06 |
| T4 compensation | §8 L532–546 (reverse order, own retries, compensation_failed, final-step note B6) | — |
| T4 cancellation | Finalization B4 verbatim (incl. fixture L333–340, warning L322) | FR-WE-07/08/09/10 (through CONTRA-5) |
| T5 triggers | F-2 (Verification L353–354); Blueprint §10 L704–724 (DomainEvent shape, TTL 7d) | FR-WE-13 |
| T6 E1 | Constitution Art. 32; IMP AWIS-E1 (live from M6, permanent) | QG-2 seed |

## Adjudications (founder-approved 2026-07-03, at M06 entry — gate log rows)
- **CONTRA-8 → ADJ-6**: RetryPolicy TDS-02 3-field vs §8 5-field ⇒ TDS-02 amended to full §8
  shape (optional initial_delay/max_delay, defaults 1s/30s; backoff enum).
- **CONTRA-9 → ADJ-7**: `required` absent from TDS-02 intelligence block vs §13 YAML +
  FR-IL-06/07 ⇒ optional `required: bool` added (default false). Closes M04's parameter deferral.
- **CONTRA-11 → ADJ-8**: FR-IL-09/§17 usage-in-StepCompleted vs frozen 4-field payload ⇒
  optional `adapter/model/tokens_used` added to TDS-01 StepCompleted (intelligence steps only,
  replay-neutral).
- **CONTRA-10 (disposition by authority order, no amendment)**: step `inputs` = template
  value-map (Blueprint §7 L362–364 + PRD §18 DSL rule 1 + FR-WD-05 vs §6 L284's "JSON Schema"
  tag). Type already `map<string,any>`; `outputs` remains schema-shaped. M10's round-trip oracle
  inherits this reading.

## Decisions (EDR-011 — engine semantics beyond frozen text; G2 docket)
Join gate concretion; WorkflowCompleted.outputs = map[final_step_id]→outputs; trigger-submit
inputs = DomainEvent.payload verbatim; domain-event consumption on fire-or-TTL; retry
re-dispatch does not re-claim (claim persists across attempts); signal steps ⇒
runner_unavailable until M07; compensation-step optional Retry shape; migration renumbering
0002=domain_events (F-2), signals/audit → 0003/0004 at M07 (F-4).

## G2 docket items carried/created
- Definition identity not evented — EDR-007 §4 option (b) interim (rebuild preserves via
  snapshot; cold rebuild loses identity, re-joinable from registry). G2 decides (a) vs (b).
- EDR-008 estimator + EDR-010 semantics + EDR-011 join/stall semantics — G2 review set.
- 'in_flight' reading (CONTRA-6) — revisit with the real engine now existing.

## Deviations
- Named-agent dispatch unavailable mid-session → general-purpose Agent with model pinned +
  identity embedded (standing AEO deviation).
