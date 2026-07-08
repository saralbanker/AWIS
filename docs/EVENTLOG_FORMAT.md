# EventLog Format — ExecutionEvent Specification

- **Spec ID:** TDS-01
- **Version:** 1.0.0
- **Date:** 2026-07-03
- **Source coordinate:** AWIS Architecture Blueprint §9 (STATE MANAGEMENT), lines 616–695
- **Status:** **FROZEN — G1 APPROVED 2026-07-03** (founder sign-off; adjudications ADJ-1/2/4a adopted, recorded inline and in the M01 module TRACEABILITY)

Scope: this document specifies **ExecutionEvents** only. Blueprint §10 DomainEvents
(TTL 7d, ingested at M06 per Verification F-2) are out of scope and not specified here.

Every normative block below cites its source coordinate inline. Where the frozen source
is silent on content the deliverable demands, a `GAP-Gn` subsection records the silence
verbatim — no value is invented (see Constitution / contradiction protocol).

---

## 1. ExecutionEvent Envelope (§9)

The EventLog is the `execution_events` table (§9, lines 624–637). Each row is one
ExecutionEvent. Field names, types, and nullability are transcribed VERBATIM from the
`CREATE TABLE execution_events` DDL:

| Field | Type | Nullability | Semantics (§9) |
|-------|------|-------------|----------------|
| `event_id` | TEXT | NOT NULL (PRIMARY KEY) | UUID; unique identity of the event. |
| `instance_id` | TEXT | NOT NULL | Workflow instance this event belongs to. |
| `namespace` | TEXT | NOT NULL | Owning namespace (e.g. `oip`). |
| `event_type` | TEXT | NOT NULL | One of the 12 enumerated types in §2. |
| `step_id` | TEXT | NULLABLE | Step the event concerns; null for workflow-level events. |
| `payload` | TEXT | NOT NULL | JSON; per-type payload schema per §2. |
| `emitted_at` | TEXT | NOT NULL | ISO8601 timestamp of emission. |
| `sequence_num` | INTEGER | NOT NULL | Monotonically increasing per instance (§9). |
| `schema_version` | INTEGER | NOT NULL DEFAULT 1 | Version of the payload-schema pack that wrote this event (G1 ADJ-1, additive to §9). |

Indexes (§9): `idx_events_instance(instance_id, sequence_num)`,
`idx_events_namespace(namespace, emitted_at)`.

### 1.1 `schema_version` semantics (G1 ADJ-1 — APPROVED 2026-07-03)

Blueprint §9's DDL (lines 624–634) does not enumerate `schema_version`; the Gate G1
checklist (IMP §23 item 1) demands it. Resolved at G1 as an **additive, backward-compatible
completion** (CONTRA-4 class), founder-approved:

- Column: `schema_version INTEGER NOT NULL DEFAULT 1` on `execution_events`.
- Semantics: the version of the **payload-schema pack** (this document's §2) in force when
  the event was written. Readers MUST interpret `payload` per that version.
- The value changes only via a future amendment to this document; the EventLog remains
  append-only — rows written under version N are never rewritten (P8).
- Distinct from the database migration version (the `schema_version` *table*, IMP §14) and
  from any workflow-level version.

---

## 2. Event-Type Payload Schemas (§9)

§9 ("Event Types", lines 662–675) enumerates exactly **12** event types. Each payload
schema below is transcribed VERBATIM; each carries a REPLAY-SUFFICIENCY statement (IR-1).

1. **WorkflowStarted** — `{inputs: map}` (§9)
   - REPLAY: reconstructs the instance's initial variable state because it carries the
     full `inputs` map that seeds workflow variables.

2. **StepStarted** — `{step_id, attempt: int, inputs: map}` (§9)
   - REPLAY: reconstructs a step activation because it carries `step_id`, the `attempt`
     ordinal, and the resolved `inputs` handed to the runner.

3. **StepCompleted** — `{step_id, attempt: int, outputs: map, duration_ms: int, adapter?: string, model?: string, tokens_used?: int}` (§9; optional usage triple added by **G1-amendment ADJ-8, founder-approved 2026-07-03** — CONTRA-11: FR-IL-09 Must Have + Blueprint §17 "recorded in the StepCompleted event payload" vs the §9-derived four-field schema; additive, present only for intelligence steps, replay-neutral — the EDR-007 projection reads only `outputs`)
   - REPLAY: reconstructs post-step variable state because it carries the produced
     `outputs` keyed to `step_id`/`attempt`, plus `duration_ms` for audit. The optional
     usage triple is observability data; it never mutates projection state.

4. **StepFailed** — `{step_id, attempt: int, error: StepError, retrying: bool}` (§9)
   - REPLAY: reconstructs the failure and retry decision because it carries the `error`,
     the `attempt` that failed, and the `retrying` flag determining next transition.

5. **StepFallbackActivated** — `{step_id, fallback_step_id, reason: string}` (§9)
   - REPLAY: reconstructs the fallback edge because it carries the failed `step_id`, the
     chosen `fallback_step_id`, and the `reason`.

6. **SignalReceived** — `{signal_name, payload: map}` (§9)
   - REPLAY: reconstructs the waiting→running transition because it carries the
     `signal_name` and delivered `payload` (authoritative per §8 Signal Atomicity).

7. **WorkflowCompleted** — `{outputs: map, duration_ms: int}` (§9)
   - REPLAY: reconstructs terminal success state because it carries the final `outputs`
     and total `duration_ms`.

8. **WorkflowFailed** — `{step_id, error: StepError}` (§9)
   - REPLAY: reconstructs terminal failure because it carries the failing `step_id` and
     the `error`.

9. **WorkflowCancelled** — `{reason: string}` (§9; **canonical per G1 ADJ-2 / CONTRA-5**)
   - REPLAY: reconstructs terminal cancellation because it carries the `reason`.
   - **CONTRA-5 disposition (G1-approved 2026-07-03):** §8 line 603 restates this event as
     `{reason: string, cancelled_at: Timestamp}`, conflicting with §9 line 671. Canonical
     payload is §9's `{reason}` only; `cancelled_at` is fully derived from the envelope's
     `emitted_at` on the same row and MUST NOT be duplicated in the payload. Zero
     information loss.

10. **WorkflowCompensating** — `{from_step: string}` (§9)
    - REPLAY: reconstructs entry into compensation because it carries `from_step`, the
      point from which reverse-order rollback begins.

11. **WorkflowCompensated** — `{}` (§9)
    - REPLAY: reconstructs successful compensation completion; no fields are required
      because the envelope's `instance_id`/`sequence_num` fully locate the terminal state.

12. **WorkflowCompensationFailed** — `{step_id, error: StepError}` (§9)
    - REPLAY: reconstructs the critical compensation-failure state because it carries the
      `step_id` whose undo failed and the `error`.

### 2.1 StepError — persisted payload type (G1 ADJ-4a — FROZEN 2026-07-03)

`StepError` appears inside three persisted payloads (events 4, 8, 12) and is therefore
part of the irreversible format. §9 names it without defining it; frozen at G1 as the
smallest shape satisfying its frozen usages (RetryPolicy `retryable_errors` matches by
`code`, §6; error-message discipline, PRD §26):

```
StepError {
  code:     string              // stable, machine-matchable error class
  message:  string              // human-readable description
  details?: map<string, any>    // optional handler-supplied context
}
```

- REPLAY: any event carrying a `StepError` reconstructs the failure cause because `code`
  is stable across releases and `message`/`details` are self-contained.
- Non-persisted API types (DraftResponse, Logger, HistoryQuery, …) are explicitly NOT
  part of this freeze (G1 ADJ-4b): they are minimal shells until their owning milestones
  (IMP §13 — sdk surface mutable until M08).

---

## 3. Sequence Rules (§9)

- **Monotonic per instance:** `sequence_num` is "monotonically increasing per instance"
  (§9, line 633; index `idx_events_instance`).
- **Append-only:** the EventLog is "Append-only. Every state change is an event. Never
  modified." (§9, line 622).
- **Gap semantics:** the log "Persists forever unless explicitly pruned by governed action"
  (§9, line 622). §9 does not otherwise define permissible gaps in `sequence_num`;
  monotonicity is the only stated ordering invariant.

---

## 4. Authority Statement (§9)

The EventLog is the **source of truth** (§9, line 620: "Layer 1 — EventLog (source of
truth)"). The StateStore (`workflow_instances`) is a **rebuildable projection** (§9,
line 639: "Layer 2 — StateStore (materialized projection)"; line 641: "Rebuilt from
EventLog if corrupted"). `awis rebuild-state` replays the EventLog to rebuild the
StateStore and "is safe to run at any time" (§9, lines 677–679).
