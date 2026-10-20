# EDR-013 — Recovery of Non-Running Instances: Evidence, Retry Once, Stop (LA-D1)

**Status:** Proposed — the founder set the policy (2026-10-18); this design and its acceptance tests
await founder approval before implementation.
**Authored:** Loop A, CE (2026-10-18)
**Coordinates:** EDR-011 §5/§6 (compensation), §8 · Finalization B3/B4 · STATE.md "LOOP A" → LA-D1,
N4, LA-C5 STOP-1, L9a/L9b · EDR-012

---

## 1. Founder policy (verbatim intent)

Capture the crash location and as much diagnostic evidence as possible. Retry once. If the retry
fails, stop and preserve the evidence. Retries must account for operations that may already have
succeeded.

## 2. Cases in scope

`Tick` scans only `running` instances, so every case below sits outside all recovery today.

| ID | Durable state found | Status | Today | Crash-point label |
|---|---|---|---|---|
| W1 (L9a) | `WorkflowFailed` naming a step, plan exists, ≥1 step completed, no `WorkflowCompensating` | `failed` | compensation never runs | `compensation_not_started` |
| W2 (L9b) | `WorkflowCompensating`, every eligible undo ran (in-process knowledge or completion records), no closing event | `compensating` | stuck forever | `compensation_close_lost` |
| W3 | `WorkflowCompensating`, some undos ran, process died mid-bracket | `compensating` | stuck forever | `compensation_interrupted` |
| W4 (N4) | `WorkflowCompensating` durable, its projection write failed | `running` | a second `WorkflowCompensating` is appended | `compensating_projection_lost` |
| W5 (LA-C5 STOP-1) | `StepFailed(signal_timeout)` with action fail/compensate, no `WorkflowFailed` after it | `waiting`, no wait_record | stuck forever | `timeout_route_lost` |
| W7 (review 5, finding 1) | compensation-bracket write fault: `WorkflowCompensating` durable but ack-lost, or its projection write failed → row still `failed`, **zero undos ran**; or `WorkflowCompensated` durable but its projection was lost | `failed` / `compensating` | silently abandoned (the terminal fence leaves compensating-last logs alone, by LA-D1 design) | `compensation_bracket_write_lost` |
| W6 (review 4, LOW-1/HIGH-2) | committed route-(3) `StepFailed{retrying:false}` with no terminal event, **or** a durable `WorkflowFailed` whose projection write was lost, on an instance parked by a sibling wait; **no further signal or timeout arrives** | `waiting` | in-process: handled by the LA-C6/LA-C7 committed-route stage, **except** when a stop is requested in the same tick (the stage is skipped and the instance is never recorded; review 5 finding 2, fix pending). After a restart nothing touches the instance, so nothing triggers recovery; stuck forever | `parked_route_unfinished` |

Out of scope here: LA-R6 (a `running` instance whose sibling wait cannot be delivered) is a
projection-status issue, not a non-running recovery issue.

Partly covered today: LA-C6/LA-C7 (R3-M1) gate signal delivery and timeouts on a `waiting` instance
whose log holds a committed route-(3) decision, or a durable terminal event its projection missed.
They complete that route in a dedicated tick step. That covers every case where a signal or timeout
touches the instance, and the in-process error path. It does **not** cover W6 after a restart: no
signal or timeout means nothing triggers recovery. The startup sweep below closes W6. Once this EDR
is approved, the RECOVERY stage subsumes the narrow LA-C6/LA-C7 path.

## 3. Accounting for operations that may already have succeeded

| Operation | Can it be checked durably? | Retry rule |
|---|---|---|
| Event append (`WorkflowFailed`, `WorkflowCompensating`, `WorkflowCompensated`, …) | yes: the EventLog | append only if the log does not already hold that event for this bracket (the LA-C4a fence pattern). This alone fixes W4/N4. |
| Projection write | yes: re-derive from the log | re-project from the log (idempotent) |
| wait_record delete | yes | idempotent |
| Signal delivery | yes: B3 guard | idempotent by B3 |
| **Undo handler** | **no**: undos emit no events (EDR-011 §5, 12 frozen event types) | see below |

**Undo accounting without a format change.** Two durable records per undo go in the existing
`step_results_cache` table, which has a 24h TTL (`idempotencyTTL`). They use dedicated key
prefixes and touch no schema:

- `undo-start/<instance>/<step>/<bracketSeq>`: written before the undo handler runs;
- `undo-done/<instance>/<step>/<bracketSeq>`: written after it returns success.

`bracketSeq` is the `sequence_num` of the bracket's `WorkflowCompensating`, so a later bracket never
reuses an earlier record.

On retry, each eligible undo falls into one of four states:

| start | done | Meaning | Action |
|---|---|---|---|
| no | no | provably never ran | run it |
| yes | yes | ran successfully | skip it |
| yes | no | **may or may not have taken effect** | **do not re-run; stop with evidence** |
| expired (> 24h) | — | unknowable | stop with evidence |

Undo handlers are never re-run on ambiguity, because the frozen `CompensationStep` schema has no
"idempotent" flag.

## 4. Mechanism

1. **Detection (where cases are found).**
   - *In-process:* any error inside a compensation bracket, a timeout route, or a terminal-route
     completion adds `(instance, crash point, error chain)` to an in-memory recovery set.
   - *After restart:* a **startup sweep**, run once when `Run` starts, before the first tick,
     covers instances in `compensating` and `waiting`, plus `failed` instances whose definition has
     a compensation plan. The case is classified from the EventLog (§2 table).
2. **Evidence capture**, done once at detection and again at the stop:
   - crash-point label;
   - projection snapshot (status, version, current_steps);
   - the last 20 events (type, step, sequence_num);
   - the instance's wait_records and step_claims;
   - undo start/done records;
   - the original error chain, when known;
   - worker id, attempt number and timestamps.
   Evidence is written to:
   (a) the audit log, as `EventType: "RecoveryEvidence"` with a bounded JSON `PayloadSummary` (the
   audit event-type set is open, CONTRA-4);
   (b) an Error-level log line.
3. **Retry once.** A new tick stage, `RECOVERY`, runs after SCAN_RUNNABLE and before SIGNAL_SCAN.
   It completes the case's remaining tail exactly once, using only the §3 rules:
   - W1: open the bracket, then run the undos;
   - W2: append the closing event;
   - W3: resume the undos per the §3 table, then close;
   - W4: no second opener; re-project and continue the bracket;
   - W5: append the committed `WorkflowFailed` and compensate (timeout action from the definition).
4. **Retry budget.** The audit log records each attempt (`EventType: "RecoveryAttempt"`, instance +
   crash point), so "once" holds across restarts. **Founder confirmation needed:** whether a
   restart should grant a fresh attempt. Proposed: no.
5. **Stop.** If the single retry fails, or §3 says "stop":
   - compensation cases (W1–W4) append `WorkflowCompensationFailed{step_id, error{code:
     "recovery_stopped", message, details: <evidence>}}`, an existing terminal event whose
     optional `details` map holds the evidence. The instance ends `compensation_failed`, which is
     visible in `awis status` and `awis trace`;
   - W5 appends `WorkflowFailed{step_id, error{code: "recovery_stopped", details}}` without
     compensation;
   - if even the stop write fails (storage persistently down), the evidence stays in the log file
     and the in-memory set, and no further attempts are made in this process.

Changes needed:
- new engine code: the startup sweep, the RECOVERY stage, and the undo records;
- uses existing tables and events only;
- no schema, event-type or payload-shape change.

## 5. Acceptance tests

| # | Scenario | Assertion |
|---|---|---|
| D1-1 | W1: `WorkflowCompensating` append fails once (in-process, and after restart) | bracket opens once, undos run once each, `WorkflowCompensated`; evidence row present |
| D1-2 | W2: `WorkflowCompensated` append fails once | exactly one closing event; no undo re-runs (undo invocation count unchanged) |
| D1-3 | W3: crash after undo A done, before undo B starts | after restart B runs once, A not re-run; bracket closes |
| D1-4 | W3-ambiguous: crash between undo B's handler return and its done-record | no re-run of B; `WorkflowCompensationFailed{code: recovery_stopped}` with evidence naming B |
| D1-5 | W4/N4: `WorkflowCompensating` projection fails | exactly one `WorkflowCompensating` in the log |
| D1-6 | W5: `WorkflowFailed` lost after `StepFailed(signal_timeout)` (fail and compensate actions) | the committed route completes once; the instance leaves `waiting` |
| D1-7 | retry fails too (fault injected twice) | stop event appended; evidence contains crash point, error chain, last events; no third attempt |
| D1-8 | retry budget across restart | an instance with a recorded attempt is not retried after restart (per the §4.4 ruling) |
| D1-9 | stop write also fails | no panic; Error log carries the evidence; no further attempts in this process |
| D1-10 | ack-lost / projection loss on every bracket write (W7) | no duplicate events; **exactly one** run per eligible undo; exactly one closing event (`WorkflowCompensated` or `WorkflowCompensationFailed`); final status equals the no-fault run |
| D1-12 | W6: committed route-(3) (fail-before and ack-lost `WorkflowFailed`) on a parked instance, restart, no signal | the startup sweep completes the route once; outcome identical to the no-fault run |
| D1-11 | startup sweep cost | sweep over 10K terminal instances stays within an NFR-style bound (proposed < 2s), recorded |

## 6. Failure modes and risks

- **24h undo-record TTL:** a bracket interrupted for more than 24h becomes unknowable, so it stops
  with evidence by design.
- **Startup sweep cost** grows with failed instances whose definitions have plans (local mode;
  bounded by D1-11).
- **Audit log as budget store:** if the audit write fails, the budget is not durable. Rule: an
  audit-write failure counts as "attempt used".
- **Behavior change for operators:** `compensation_failed` now appears with code `recovery_stopped`.
  This is documented, not silent.
