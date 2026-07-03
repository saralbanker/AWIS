# EDR-007 — Projection Rules for RebuildState

**Status:** Reversible — pre-G2 (implementation decisions; superseded only by an explicit G2 amendment)
**Authored:** M03-C2 (2026-07-03)
**Coordinates:** TDS-01 §2/§3 · Blueprint §6/§8/§9/§20 · ADJ-5 · M03 IMPLEMENTATION_SPEC.md Deliverable 3
**Downstream obligation:** M06 must mirror these rules in its submit path; any divergence is a projection-consistency defect.

---

## 1. Purpose

`RebuildState` (`internal/storage/rebuild.go`) replays the append-only
`execution_events` table and reconstructs the `workflow_instances` projection
from scratch. This document transcribes the complete, authoritative rule set
so that M06 and later milestones can implement the same logic in the forward
(submit-time) path without re-deriving it.

---

## 2. Status Map Table

The following table maps each of the 12 event types to the resulting
`workflow_instances.status` and the additional field mutations. Fields not
listed are unchanged by that event type. `UpdatedAt` is always set to the
event's `emitted_at` and is therefore omitted from individual rows.

| Event type | → status | StartedAt | CompletedAt | current_steps | variables |
|---|---|---|---|---|---|
| `WorkflowStarted` | `running` | = emitted_at | — | `[]` (reset) | `{"inputs": payload.inputs}` |
| `StepStarted` | — | — | — | += envelope `step_id` | — |
| `StepCompleted` | — | — | — | -= step_id | `[step_id]` = payload.outputs |
| `StepFailed` (retrying=true) | — | — | — | (unchanged) | — |
| `StepFailed` (retrying=false) | — | — | — | -= step_id | — |
| `StepFallbackActivated` | — | — | — | -= payload.step_id | — |
| `SignalReceived` | `running` | — | — | — | — |
| `WorkflowCompleted` | `completed` | — | = emitted_at | `[]` (cleared) | — |
| `WorkflowFailed` | `failed` | — | = emitted_at | — | — |
| `WorkflowCancelled` | `cancelled` | — | = emitted_at | — | — |
| `WorkflowCompensating` | `compensating` | — | — | — | — |
| `WorkflowCompensated` | `compensated` | — | = emitted_at | — | — |
| `WorkflowCompensationFailed` | `compensation_failed` (ADJ-5) | — | = emitted_at | — | — |

**"—" means: field is not mutated by this event.**

All `emitted_at` values are sourced from the event envelope column, not from
any payload field (CONTRA-5 canonical disposition for `WorkflowCancelled`).

---

## 3. Variable Scoping

Variables are accumulated per instance across the lifetime of the event stream
(TDS-03 dot-path reference model; Blueprint §9):

```
variables = {
    "inputs":   <WorkflowStarted payload.inputs map>,
    "<step_id>": <StepCompleted payload.outputs map for that step>,
    ...
}
```

- `WorkflowStarted` seeds the map with `{"inputs": payload.inputs}`. Any
  prior variable state is discarded (WorkflowStarted resets the projection).
- Each `StepCompleted` adds or overwrites `variables[step_id]` with
  `payload.outputs`. Key is the step's id; value is the entire outputs map.
- No other event type mutates variables.
- Key space is collision-free: "inputs" is never a valid step_id (reserved).

---

## 4. Definition-Identity Gap (EDR-007 Gap)

**Gap:** `workflow_instances.definition_id` and `definition_version` are not
carried in any event envelope or payload (Blueprint §9 DDL; TDS-01 §1 — the
`WorkflowStarted` payload carries only `inputs`). These fields are set at
submit time by M06 and are **not evented**.

**Mechanism (smallest viable):** `RebuildState` snapshots the
`(instance_id → definition_id, definition_version)` pairs from the current
`workflow_instances` rows *before* wiping. After projection replay, each
rebuilt row is populated from that snapshot. Instances absent from the
snapshot (never previously submitted, or DB fully wiped) receive empty strings
for both fields.

**G2 docket item:** M06 must either (a) append a `WorkflowStarted`-extension
event carrying `definition_id`/`version` (requires a payload amendment — G2
decision) or (b) accept that a cold rebuild (from a fully wiped StateStore)
loses definition identity. Option (b) is acceptable because definition
identity can be re-joined from the registry at read time. This decision is
deferred to G2.

---

## 5. `cancellation_requested` — Always Rebuilt as 0

`cancellation_requested` is a non-evented runtime request flag: it is set by
the cancel API at request time and is not written as an `ExecutionEvent`.
Therefore `RebuildState` always writes `cancellation_requested = 0`.

The cancel API caller must re-issue the cancellation request against the
rebuilt instance if needed after a rebuild. This is an accepted limitation
(documented here to make it explicit for M06 and the cancel-path author).

---

## 6. Rebuild Version = 1

Rebuilt rows are inserted with `version = 1`. Rationale: the version column
is an optimistic-lock counter used by the runtime (UpsertInstance / ClaimStep)
to detect concurrent mutations. After rebuild, the StateStore is a fresh
projection; any in-flight engine workers that held a stale version reference
before the rebuild must lose their conflict check and re-read. Setting
`version = 1` achieves this: a pre-rebuild worker that held `version = N > 1`
will see a conflict on its next UpsertInstance and must retry.

---

## 7. step_claims — Wiped, Not Rebuilt

`step_claims` is runtime state (Blueprint §8 at-most-once claim gate), not
history. The EventLog contains no `ClaimStep` events; claims cannot be
replayed. `RebuildState` wipes `step_claims` and does not repopulate it.

Consequence: workers that held claims before a rebuild lose them. Workers must
re-claim after rebuild. This is the correct behavior: a rebuild implies a
recovery scenario in which stale claim state must not block forward progress.

---

## 8. Defensive Rule — No `WorkflowStarted` Event

If an instance's event stream begins without a `WorkflowStarted` event
(e.g., due to a partial write or future event-type extension), the projection
starts from zero-value instance state with `namespace` taken from the first
event's envelope `namespace` column. `status` defaults to `running`.

This rule is pre-authorized and does not imply a gap in normal operation;
normal operation always begins with `WorkflowStarted`.

---

## 9. Waiting-Status Entry (M07 Docket)

The `waiting` status (Blueprint §6, §8 signal gate) is not directly produced
by any event in the current 12-type set. Entry into `waiting` is an engine
decision made at tick time; it is not recorded as a standalone event.
`SignalReceived` exits `waiting` → `running` (rule above). The waiting-entry
event (or lack thereof) is a known gap deferred to M07.

---

## 10. Scope and Forward Obligations

- **M06** must mirror these projection rules in its forward path (the live
  UpsertInstance writes that the engine issues as events arrive). Any
  divergence between the live path and the replay path is a
  projection-consistency defect.
- **ADJ-5** governs the `compensation_failed` status literal (9th
  `InstanceStatus` enum value); this EDR records its projection rule.
- **TDS-01 §2/§3** is authoritative for payload field names; this EDR records
  which fields are consumed during replay.
- **Blueprint §8** ("SETTLE: release claim") is satisfied by the wipe of
  `step_claims` in rebuild; EDR-006 records the forward-path release site.
