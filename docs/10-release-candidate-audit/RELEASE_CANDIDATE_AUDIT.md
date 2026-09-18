# Release Candidate Challenge Audit: AWIS Beta

**Date:** 2026-09-03  
**Auditor:** Adversarial Release Verification Team  
**Evaluation Target:** AWIS Beta Release Candidate (`engine-hardening` @ `e75c1f1` + uncommitted API/GUI implementation)  
**Authority Level:** Tier 1 Evidence (Source Code, Database Schema, Direct Execution, Runtime Invariants)

---

## 1. Executive Verdict

**VERDICT: REJECTED FOR RELEASE (NOT PRODUCTION/BETA READY)**

AWIS Beta is **not release-ready**. While the test suite passes (`go test ./...` is 100% green) and the TypeScript frontend builds cleanly, an adversarial challenge against Tier-1 runtime invariants reveals **critical engine correctness defects, crash recovery flaws, silent data loss during cancellation, serious security vulnerabilities, and a fundamental API pagination defect that breaks the primary dashboard user experience**.

Passing tests in this codebase have created a false sense of security:
1. **Crash Recovery is Unimplemented for In-Flight Steps:** If the process crashes or restarts while a step is executing, the workflow instance is **permanently wedged in `running` status forever**. `step_claims` retains an unexpiring lock, and the engine's hydration logic ignores in-flight steps.
2. **Silent Loss of Compensation and Cancel Reasons:** Cancellation intent (`reason`, `compensate`) is stored solely in volatile process memory (`e.cancels`). A restart during cancellation silently erases the reason and **completely aborts compensation**, leaking external resources.
3. **Dashboard Landing Screen Fails Its Primary Purpose:** The SQLite query in `ListInstancesPaged` orders by `started_at, instance_id ASC` (oldest first). On databases with >50 instances, the landing screen displays the 50 oldest workflows from system creation, client-side sorting them. **Newly started, running, or recently failed workflows never appear on Page 1**.
4. **Severe Secret Leakage via Subprocess Execution:** `SubprocessRunner` executes child processes without sanitizing environment variables (`cmd.Env == nil`), leaking `ANTHROPIC_API_KEY`, database credentials, and host tokens to every executed subprocess.
5. **Anthropic Integration Uses Fictional Model Names:** Hardcoded model strings `claude-haiku-4-5` and `claude-sonnet-5` do not exist in Anthropic's API and fail immediately with HTTP 404 in production. Live tests were skipped in CI.

---

## 2. Severity-Ranked Findings Matrix

| ID | Severity | Finding | Blocks Beta | Blocks Phase 2 | Blocks Builder |
|---|---|---|:---:|:---:|:---:|
| **SEC-01** | **Critical** | In-Flight Step Crash Recovery Deadlock (Zombie Instances) | **YES** | **YES** | **YES** |
| **SEC-02** | **Critical** | Volatile Cancellation Intent: Silent Compensation Bypass on Restart | **YES** | **YES** | No |
| **SEC-03** | **Critical** | Dashboard Landing Inversion: Oldest 50 Workflows Shown on Page 1 | **YES** | **YES** | No |
| **SEC-04** | **High** | Secret Leakage: Subprocess Steps Inherit All Host Environment Variables | **YES** | **YES** | **YES** |
| **SEC-05** | **High** | Anthropic Live Integration Broken: Fictional Model IDs & Untested CI | **YES** | No | No |
| **SEC-06** | **High** | Full-Table FTS Index Destruction & Rebuild on Every Search Query | **YES** | **YES** | No |
| **SEC-07** | **High** | Single SQLite Connection (`MaxOpenConns(1)`) Lock Contention | **YES** | **YES** | No |
| **SEC-08** | **Medium** | Stalled Non-Final Workflows Silently Hang in `running` Indefinitely | **YES** | **YES** | **YES** |
| **SEC-09** | **Medium** | Global Namespace Collision in `workflow_definitions` Primary Key | No | **YES** | **YES** |
| **SEC-10** | **Medium** | Absence of Workflow Mutation API, Draft Storage, and YAML Serializer | No | **YES** | **YES** |
| **SEC-11** | **Medium** | Strict Acyclic DAG Enforcement Precludes n8n-Style Iteration/Loops | No | No | **YES** |
| **SEC-12** | **Low** | Fake Healthcheck: `/api/v1/healthz` Returns 200 with Dead DB/Engine | No | No | No |

---

## 3. Deep-Dive Findings, Evidence & Reproductions

### Finding SEC-01: In-Flight Step Crash Recovery Deadlock (Zombie Instances)
- **Severity:** Critical
- **Impact:** Permanent deadlock of running workflow instances after any unexpected crash, daemon kill, or node reboot.
- **Affected Components:**
  - `internal/engine/tick.go:198-202, 222-233, 262-266`
  - `internal/engine/failure.go:133`
  - `internal/engine/hydrate.go:63-83`
  - `internal/storage/sqlite.go:894-953`
  - `internal/storage/rebuild.go:355-366`

#### Evidence
When a step begins execution:
1. `e.claim(ctx, inst.InstanceID, stepID)` calls `s.storage.ClaimStep(...)`, inserting a row into `step_claims (instance_id, step_id, worker_id, claimed_at)` with `PRIMARY KEY (instance_id, step_id)`.
2. `e.emitStepStarted(...)` appends `StepStarted` to `execution_events` and updates `workflow_instances.current_steps` to include `stepID`.
3. The step runs asynchronously in a worker goroutine (`dispatchOne`).

If the process crashes or is killed (SIGKILL, panic, power failure) before `StepCompleted` or `StepFailed` is emitted:
- The `step_claims` table retains `(instance_id, step_id, worker_id, claimed_at)` indefinitely because claims have **no TTL, no heartbeat, and no lease expiration timestamp**.
- On engine restart, `e.hydrate(ctx, dv, inst)` parses `execution_events`. It sees `EventTypeStepStarted` for `stepID`. In `hydrate.go:105-124`, only events of type `EventTypeStepFailed` are examined for scheduling retries. `relevant[stepID]` is `EventTypeStepStarted`, so `retries` is NOT populated.
- On the next tick, `e.gatherDispatch(ctx, dv, inst)` calls:
  - `e.activatableFor(dv, inst)`: Skips `stepID` because `running[stepID]` is true (`inst.CurrentSteps` contains `stepID`).
  - `e.dueRetries(...)`: Checks `e.retries`, which is empty. Returns nil.
- `items` is empty, so no steps dispatch.
- The completion check (`tick.go:198`):
  ```go
  if !found || len(inst2.CurrentSteps) != 0 {
      return nil
  }
  ```
  Aborts because `inst2.CurrentSteps` still contains `stepID`.
- Even running `RebuildState` does not resolve this: `rebuild.go:365` projects `p.currentSteps = appendUniq(p.currentSteps, pid)` upon reading `StepStarted`.
- Result: **The instance is permanently wedged in `running` status, cannot complete, cannot fail, cannot retry, and can never be claimed again.**

#### Reproduction Steps
1. Create a workflow with a step that sleeps for 30 seconds (`nativeStep("slow", ...)`).
2. Submit and tick the engine once. Confirm `StepStarted` is written and `step_claims` has a row for `slow`.
3. Terminate the process (`kill -9`).
4. Reopen the database with a new `Engine` instance and execute `Tick()`.
5. Observe: `Tick()` executes with 0 dispatches. The instance remains in status `running` with `current_steps: ["slow"]`. Repeat 100 ticks; the instance never moves.

#### Recommended Fix
1. Add a `lease_expires_at` column to `step_claims`.
2. In `hydrate()`, inspect `inst.CurrentSteps`. Any step in `CurrentSteps` whose latest event is `StepStarted` and has no active worker lease must be recovered: emit a `StepFailed` event with code `worker_crash` (or re-schedule attempt 1).

---

### Finding SEC-02: Volatile Cancellation Intent: Silent Compensation Bypass on Restart
- **Severity:** Critical
- **Impact:** External side effects left uncompensated; audit trail corruption.
- **Affected Components:**
  - `internal/engine/cancel.go:47-48, 129-132, 204-215`
  - `internal/engine/engine.go:79, 101-105`
  - `internal/storage/migrations/0001_core_execution.sql:40`

#### Evidence
`Engine.Cancel(ctx, instanceID, reason, compensate)` sets `cancellation_requested = 1` in `workflow_instances`.
However, `reason` and `compensate` are recorded **only in volatile memory**:
```go
e.cancels[instanceID] = cancelIntent{reason: reason, compensate: compensate}
```
If the process restarts before in-flight steps settle:
- `e.cancels` starts as an empty map.
- When in-flight steps settle, `handleCancellation` calls `finalizeCancellation(ctx, dv, cur)`:
```go
intent := e.cancels[inst.InstanceID] // Returns zero-value: reason="", compensate=false!
delete(e.cancels, inst.InstanceID)

if intent.compensate && dv.def.Compensation != nil && len(completedSet(inst)) > 0 {
    return e.runCompensation(ctx, dv, inst, "")
}
return e.emitWorkflowCancelled(ctx, inst, intent.reason)
```
- Because `intent.compensate` is `false`, `runCompensation` is **completely bypassed**.
- The workflow is finalized as `WorkflowCancelled` with an empty string reason.

#### Reproduction Steps
1. Define a workflow with compensation handlers on completed steps.
2. Run step 1 to completion.
3. While step 2 is running, call `Cancel(ctx, iid, "critical security stop", true)`.
4. Kill the process immediately.
5. Restart the process and run `Tick()`.
6. Observe: Event log contains `WorkflowCancelled{reason: ""}`. Compensation steps are never dispatched.

#### Recommended Fix
Store cancellation intent durably: either add `cancellation_reason TEXT` and `cancellation_compensate INTEGER` to `workflow_instances`, or append a `WorkflowCancellationRequested` event directly to `execution_events`.

---

### Finding SEC-03: Dashboard Landing Inversion: Oldest 50 Workflows Shown on Page 1
- **Severity:** Critical
- **Impact:** Total failure of operational awareness; operators cannot monitor active workflows.
- **Affected Components:**
  - `internal/storage/sqlite.go:807-814`
  - `web/src/screens/instanceList.ts:96-125`

#### Evidence
In `internal/storage/sqlite.go:812`:
```sql
SELECT instance_id, definition_id, definition_version, namespace, status,
       current_steps, variables, started_at, updated_at, completed_at
FROM workflow_instances
WHERE 1=1
ORDER BY started_at, instance_id
LIMIT ? OFFSET ?
```
Notice `ORDER BY started_at, instance_id` is **ASCENDING** (oldest first).
In `web/src/screens/instanceList.ts:96-101`:
```typescript
// The API's native order is (started_at, instance_id) ascending
// (internal/storage/sqlite.go's ListInstancesPaged) — NOT
// updated_at-descending as originally assumed in
// GUI_PHASE1_SCREEN_SPEC.md's "recent activity" landing description.
// Sorted client-side here rather than changing the backend...
```
The frontend fetches Page 1 (`limit: 50, offset: 0`), receiving the **50 oldest workflows ever recorded**. It then client-side sorts those 50 rows. If there are 500 workflows, instances 51 through 500—including every workflow launched today—**never reach the frontend on the landing screen**.

#### Reproduction Steps
1. Seed the SQLite database with 60 completed workflow instances.
2. Launch a new workflow instance `inst-live-now`.
3. Open the GUI dashboard at `http://127.0.0.1:8090/#/instances`.
4. Observe: `inst-live-now` is not in the list. The list displays instances 1 through 50 from the past.

#### Recommended Fix
Modify `ListInstancesPaged` in `internal/storage/sqlite.go` to support `ORDER BY started_at DESC, instance_id DESC`. Expose this in `core.InstanceFilter` and `internal/api`.

---

### Finding SEC-04: Secret Leakage: Subprocess Steps Inherit All Host Environment Variables
- **Severity:** High
- **Impact:** Arbitrary subprocess steps can exfiltrate all host secrets (`ANTHROPIC_API_KEY`, AWS tokens, DB passwords).
- **Affected Components:**
  - `internal/runner/subprocess/subprocess.go:125-127`

#### Evidence
In `internal/runner/subprocess/subprocess.go`:
```go
cmd := exec.Command(argv[0], argv[1:]...)
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
```
`cmd.Env` is never set. In the Go standard library, when `cmd.Env == nil`, `os.Environ()` is inherited directly from the parent process. Furthermore, `strings.Fields(string(step.Handler))` splits on whitespace with no path restrictions, allowing arbitrary executable invocation.

#### Reproduction Steps
1. Start `awis` with `ANTHROPIC_API_KEY=sk-ant-test-secret-12345`.
2. Define a workflow with a subprocess step: `handler: python3 -c "import os; print(os.environ.get('ANTHROPIC_API_KEY'))"`.
3. Execute the workflow.
4. Verify stdout in step output contains the plaintext key.

#### Recommended Fix
Explicitly assign `cmd.Env` in `SubprocessRunner.Run` to a sterile baseline:
```go
cmd.Env = []string{
    "PATH=/usr/bin:/bin",
    "LANG=en_US.UTF-8",
    fmt.Sprintf("AWIS_INSTANCE_ID=%s", sc.InstanceID),
    fmt.Sprintf("AWIS_STEP_ID=%s", sc.StepID),
}
```

---

### Finding SEC-05: Anthropic Live Integration Broken: Fictional Model IDs & Untested CI
- **Severity:** High
- **Impact:** Any workflow utilizing Claude intelligence steps fails immediately upon execution against Anthropic.
- **Affected Components:**
  - `internal/intelligence/adapters/anthropic/anthropic.go:34-36`
  - `internal/intelligence/adapters/anthropic/live_test.go:28-34`

#### Evidence
In `internal/intelligence/adapters/anthropic/anthropic.go`:
```go
const (
	modelFast = "claude-haiku-4-5"
	modelQuality = "claude-sonnet-5"
)
```
Anthropic has never released models under these identifiers. Live API calls return HTTP 404 (`not_found_error`). The live integration test in `live_test.go` has `if os.Getenv("ANTHROPIC_LIVE") != "1" { t.Skip(...) }`, meaning it has never been executed in CI. Unit tests mock HTTP using local fixtures in `testdata/`.

#### Reproduction Steps
1. Export `ANTHROPIC_API_KEY=<valid_key>` and `ANTHROPIC_LIVE=1`.
2. Run `go test -v ./internal/intelligence/adapters/anthropic/ -run TestLiveSmoke`.
3. Observe HTTP 404 failure from Anthropic API.

#### Recommended Fix
Change model constants to valid model identifiers (`claude-3-5-haiku-20241022`, `claude-3-5-sonnet-20241022`), and make them configurable via `sdk.Config`.

---

### Finding SEC-06: Full-Table FTS Index Destruction & Rebuild on Every Search Query
- **Severity:** High
- **Impact:** Massive write lock contention, disk thrashing, and engine stalls on event search.
- **Affected Components:**
  - `internal/storage/recall.go:77-79, 128-152`
  - `internal/storage/migrations/0006_recall_fts.sql:7-9`

#### Evidence
In `internal/storage/recall.go`:
```go
func (s *SQLiteStorage) rebuildFTSIndex(ctx context.Context) error {
    tx, err := s.db.db.BeginTx(ctx, nil)
    ...
    tx.ExecContext(ctx, `DELETE FROM execution_events_fts`)
    tx.ExecContext(ctx, `
        INSERT INTO execution_events_fts(event_id, instance_id, namespace, event_type, step_id, payload)
        SELECT event_id, instance_id, namespace, event_type, COALESCE(step_id, ''), payload
        FROM execution_events`)
    return tx.Commit()
}
```
`rebuildFTSIndex` is invoked on **every invocation of `SearchEvents()`**. A search query opens an exclusive write transaction in SQLite, wipes the entire virtual table, and scans/re-inserts all events in `execution_events`.

#### Recommended Fix
Implement SQLite `AFTER INSERT` triggers in a new migration, or track `last_indexed_rowid` to perform incremental indexing.

---

### Finding SEC-07: Single SQLite Connection (`MaxOpenConns(1)`) Lock Contention
- **Severity:** High
- **Impact:** HTTP queries directly stall the engine's 100ms execution loop.
- **Affected Components:**
  - `internal/storage/db.go:42`
  - `cmd/awis-server/main.go:72-125`

#### Evidence
`db.go:42` sets `sqlDB.SetMaxOpenConns(1)`. In `cmd/awis-server`, the engine runtime loop (`rt.Start`) and the HTTP handler pool share this single connection. When an HTTP handler executes `ListInstancesPaged` (which does `CountInstances` followed by `ListInstancesPaged`), the engine's tick loop blocks waiting for connection acquisition.

#### Recommended Fix
Configure connection pooling: use WAL mode with separate read pools (`sql.DB` with `MaxOpenConns(10)`) for API read paths, and a dedicated single connection for write transactions.

---

### Finding SEC-08: Stalled Non-Final Workflows Silently Hang in `running` Indefinitely
- **Severity:** Medium
- **Impact:** Incomplete workflows consume CPU ticks forever without failing or alerting.
- **Affected Components:**
  - `internal/engine/tick.go:204-207`
  - `internal/validate/validate.go:167-176`

#### Evidence
In `tick.go:204`:
```go
finals := completedFinalOutputs(dv, inst2)
if len(finals) == 0 {
    return nil // stalled (join stall / no final reached) — author semantics, EDR-011 §1.
}
return e.emitWorkflowCompleted(ctx, inst2, finals)
```
If a workflow terminates on a fallback step or an error branch that is not explicitly enumerated in `final_steps`:
- `inst2.CurrentSteps` is empty.
- `activatableFor` returns empty.
- `completedFinalOutputs` returns empty.
- The engine does nothing and returns `nil`. The instance remains in `status: "running"` indefinitely, and is queried by `ListInstances(Status: running)` on every single engine tick forever.

#### Recommended Fix
If `len(CurrentSteps) == 0` and `len(activatableFor) == 0` and `len(finals) == 0`, transition the workflow to a terminal `stalled` or `failed` state with reason `no_final_step_reached`.

---

### Finding SEC-09: Global Namespace Collision in `workflow_definitions` Primary Key
- **Severity:** Medium
- **Impact:** Prevents multi-tenant isolation; workflows in different namespaces collide on registration.
- **Affected Components:**
  - `internal/storage/migrations/0001_core_execution.sql:18`
  - `internal/storage/sqlite.go:327`
  - `sdk/runtime_runner.go:61-63`

#### Evidence
In `migrations/0001_core_execution.sql`:
```sql
CREATE TABLE workflow_definitions (
  id                TEXT NOT NULL,
  version           TEXT NOT NULL,
  namespace         TEXT NOT NULL,
  definition        TEXT NOT NULL,
  registered_at     TEXT NOT NULL,
  PRIMARY KEY (id, version)
);
```
`namespace` is excluded from the primary key. If namespace `team-a` registers `etl@1.0.0`, namespace `team-b` receives `ErrAlreadyRegistered` when attempting to register `etl@1.0.0`.

#### Recommended Fix
Alter the table primary key in a new migration to `PRIMARY KEY (namespace, id, version)`.

---

### Finding SEC-10: Absence of Workflow Mutation API, Draft Storage, and YAML Serializer
- **Severity:** Medium (Phase 2 & Visual Builder Blocker)
- **Impact:** Blocks GUI workflow editing and visual builder creation.
- **Affected Components:**
  - `internal/api/router.go`
  - `internal/storage/sqlite.go:309-332`
  - `internal/dsl/`

#### Evidence
1. `internal/api/` contains zero `POST`, `PUT`, `PATCH`, or `DELETE` endpoints for workflows.
2. `internal/storage/sqlite.go` only has `RegisterWorkflow` which rejects updates with `ErrAlreadyRegistered`.
3. `yaml.Marshal` is called 0 times in the Go codebase.

#### Recommended Fix
Develop a YAML AST emitter (`internal/dsl/serialize.go`), implement a draft storage table, and expose workflow creation/update endpoints.

---

### Finding SEC-11: Strict Acyclic DAG Enforcement Precludes n8n-Style Iteration/Loops
- **Severity:** Medium (Future Builder Blocker)
- **Impact:** AWIS cannot support iterative loops (for-each, repeat-until) common in n8n-class tools.
- **Affected Components:**
  - `internal/validate/validate.go:83, 260-266`
  - `internal/storage/migrations/0001_core_execution.sql:48`
  - `internal/dsl/render.go:140`

#### Evidence
The validator explicitly disallows any transition cycle (`CodeCycle = "cycle"`). Furthermore, `step_claims` has `PRIMARY KEY (instance_id, step_id)`, making it impossible for a step to run more than once per instance even if the validator allowed it.

#### Recommended Fix
To support n8n-class loops, step identity must be decoupled from static step IDs by introducing iteration ordinals `(instance_id, step_id, iteration)`.

---

### Finding SEC-12: Fake Healthcheck: `/api/v1/healthz` Returns 200 with Dead DB/Engine
- **Severity:** Low
- **Impact:** Orchestrators (Kubernetes/systemd) will route traffic to dead instances.
- **Affected Components:**
  - `internal/api/healthz.go:16-20`

#### Evidence
`handleHealthz` returns `{"status":"ok"}` statically without executing a database ping or verifying the engine pull loop.

#### Recommended Fix
Execute `store.PingContext(ctx)` within `handleHealthz`.

---

## 4. Release Blocker Verdict

| Release Milestone | Status | Key Blockers |
|---|---|---|
| **AWIS Beta Release** | **BLOCKED** | SEC-01 (Crash recovery), SEC-02 (Silent compensation bypass), SEC-03 (Landing page inverted), SEC-04 (Subprocess secret leak), SEC-05 (Anthropic model IDs) |
| **GUI Phase 2** | **BLOCKED** | SEC-03 (Instance pagination), SEC-07 (Connection starvation), SEC-10 (Missing mutation endpoints & YAML serializer) |
| **Visual Workflow Builder** | **BLOCKED** | SEC-10 (No serializer/draft model), SEC-09 (Namespace collision), SEC-11 (DAG cycle constraints) |
