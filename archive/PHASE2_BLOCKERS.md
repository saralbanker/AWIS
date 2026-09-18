# Phase 2 & Visual Builder Blocker Report

**Date:** 2026-09-03  
**Auditor:** Adversarial Release Verification Team  
**Scope:** Strict challenge of capabilities required for:
1. Workflow Creation UI
2. Workflow Editing UI
3. Visual Workflow Builder
4. n8n-Style GUI Evolution

---

## 1. Executive Summary

Transitioning AWIS from an engine-centric CLI tool to an interactive GUI workflow builder (GUI Phase 2 / n8n-class platform) is currently **blocked by foundational backend and architectural constraints**. 

The current read-only Beta GUI proves that static execution state can be rendered, but **almost none of the write-side infrastructure exists in the codebase today**. The engine and storage architectures were explicitly built around immutable, static YAML files discovered at boot time.

---

## 2. Ranked Blocker Register

### 2.1 Critical Blockers (Must Be Resolved Before Any Workflow Creation/Editing UI Can Function)

#### BLOCKER-01: Zero Workflow Mutation HTTP API
- **Target Area:** Workflow Creation / Editing
- **Severity:** Critical
- **Why It Blocks:**
  The server binary (`cmd/awis-server`) and router (`internal/api/router.go`) expose **strictly read-only routes**:
  - `GET /api/v1/workflows`
  - `GET /api/v1/workflows/{id}/{version}`
  - `GET /api/v1/instances`
  - `GET /api/v1/instances/{id}`
  - `GET /api/v1/instances/{id}/events`
  There are zero mutation routes (`POST /api/v1/workflows`, `PUT /api/v1/workflows/{id}/{version}`, `POST /api/v1/workflows/validate`). A frontend workflow editor has no HTTP endpoint to submit new workflows, update existing workflows, or trigger validation.
- **Evidence:** `internal/api/router.go:33-42`
- **Resolution Pathway:**
  1. Add `POST /api/v1/workflows` to register a workflow definition.
  2. Add `POST /api/v1/workflows/validate` to run `validate.Validate()` on a candidate workflow payload without persisting it.
  3. Wire execution to `StoragePort` and `internal/dsl`.

---

#### BLOCKER-02: Total Absence of YAML Serializer / Emitter Repo-Wide
- **Target Area:** Workflow Creation / Editing / Visual Builder
- **Severity:** Critical
- **Why It Blocks:**
  AWIS stores workflow definitions on disk as YAML files (`internal/dsl/discover.go`). However, **`yaml.Marshal` is called zero times in the Go codebase**. The system only has a parser (`internal/dsl/dsl.go:Parse`), never an emitter.
  If an operator designs a workflow on a visual canvas and clicks "Save":
  - The backend has no code to serialize `core.WorkflowDefinition` into valid AWIS DSL YAML.
  - Using default `gopkg.in/yaml.v3` `Marshal` on Go structs with `map[string]any` strips comments, randomizes dictionary key order, and generates massive Git diff churn.
- **Evidence:**
  - `docs/09-gui-planning/FINAL_VERDICT.md:58` ("`yaml.Marshal` is called nowhere in the repo today")
  - `internal/dsl/` contains `discover.go`, `dsl.go`, `render.go`, `validate.go` — no serializer exists.
- **Resolution Pathway:**
  Develop an AST-aware YAML emitter (e.g. `internal/dsl/serialize.go`) using `yaml.Node` to preserve stable key order and clean formatting.

---

#### BLOCKER-03: Storage Immutability & Complete Lack of Draft/Staging State
- **Target Area:** Workflow Editing / Builder State
- **Severity:** Critical
- **Why It Blocks:**
  The SQLite storage engine enforces strict immutability on workflow definitions:
  ```sql
  CREATE TABLE workflow_definitions (
    id TEXT NOT NULL,
    version TEXT NOT NULL,
    namespace TEXT NOT NULL,
    definition TEXT NOT NULL,
    registered_at TEXT NOT NULL,
    PRIMARY KEY (id, version)
  );
  ```
  In `internal/storage/sqlite.go:327`, re-registering an `(id, version)` pair returns `ErrAlreadyRegistered`.
  - There is no `UpdateWorkflow` method in `StoragePort`.
  - There is no `DeleteWorkflow` method in `StoragePort`.
  - There is no draft storage table (`workflow_drafts`).
  In any visual builder, a user creates intermediate, invalid, or draft versions over minutes or hours. In the current engine, any save must either increment SemVer on every click or fail with a constraint violation.
- **Evidence:** `internal/storage/sqlite.go:309-332`
- **Resolution Pathway:**
  Create migration `0007_workflow_drafts.sql` with mutable draft semantics, or provide an explicit `UpdateWorkflow` route with revision tracking.

---

### 2.2 High Blockers (Must Be Resolved for n8n-Style Evolution and Multi-Tenant Builders)

#### BLOCKER-04: Strict Acyclic Graph Enforcement & PK Constraint Blocking n8n-Style Loops
- **Target Area:** Visual Builder / n8n-Style GUI Evolution
- **Severity:** High
- **Why It Blocks:**
  A core strength of n8n and modern automation platforms is loop nodes ("Loop Over Items", "Repeat While", "Batch Processing").
  In AWIS, loops are **architecturally impossible on two independent layers**:
  1. **Validation Rejection:** `internal/validate/validate.go:83, 260-266` actively traverses transitions via three-color DFS and fails with `CodeCycle = "cycle"` on any loop.
  2. **Step Claim Collision:** Even if validation were disabled, `step_claims` in SQLite (`migrations/0001_core_execution.sql:48`) has `PRIMARY KEY (instance_id, step_id)`. If step `A` loops back to itself or an earlier step, the engine's second execution of step `A` fails in `ClaimStep` with a UNIQUE constraint error, resulting in a lost claim and permanent workflow hang.
  3. **Event Projection Collision:** `workflow_instances.variables` maps `step_id -> outputs`. Multiple runs of the same step overwrite previous iteration data.
- **Evidence:**
  - `internal/validate/validate.go:83, 260-266`
  - `internal/dsl/render.go:140` ("Workflows must be acyclic; use a new instance for repeating work.")
  - `internal/storage/migrations/0001_core_execution.sql:48`
- **Resolution Pathway:**
  To support n8n-class loops, the engine must evolve step execution keys from `(instance_id, step_id)` to `(instance_id, step_id, iteration_num)`.

---

#### BLOCKER-05: Global Primary Key on `workflow_definitions` Without Namespace Isolation
- **Target Area:** Workflow Creation / Multi-Tenant Builder
- **Severity:** High
- **Why It Blocks:**
  `workflow_definitions` defines `PRIMARY KEY (id, version)`, omitting `namespace`.
  If user A in namespace `marketing` registers `sync-leads@1.0.0`, user B in namespace `sales` **cannot register `sync-leads@1.0.0`**. The second registration fails with `ErrAlreadyRegistered`.
  This violates the core mental model of namespaces in visual builders and creates cross-tenant name-squatting risks.
- **Evidence:**
  - `internal/storage/migrations/0001_core_execution.sql:18`
  - `sdk/runtime_runner.go:61-63`
- **Resolution Pathway:**
  Add a database migration to redefine `workflow_definitions` primary key to `PRIMARY KEY (namespace, id, version)`.

---

#### BLOCKER-06: Complete Absence of Real-Time State Streaming (SSE / WebSockets)
- **Target Area:** Visual Builder Live Debugging / Phase 2 Live View
- **Severity:** High
- **Why It Blocks:**
  The current GUI relies on 15-second polling (`setInterval(..., 15000)`).
  In an n8n-style visual builder:
  - Users run test workflows and expect instant visual feedback as nodes pulse, transition from running to completed, and render live output payloads.
  - Polling at 15s is far too slow for interactive debugging.
  - Decreasing poll interval to 500ms will immediately exhaust the single SQLite connection (`MaxOpenConns(1)`), causing database locks.
  - SSE requires `state_changes` table (migration 0007) and change notification infrastructure that was omitted from Beta.
- **Evidence:** `docs/09-gui-planning/FINAL_VERDICT.md:46-53`
- **Resolution Pathway:**
  Implement migration 0007 (`state_changes`), add an in-memory pub/sub broker to the engine runtime, and expose `GET /api/v1/instances/{id}/live` via Server-Sent Events (SSE).

---

### 2.3 Medium Blockers (Degrades Builder Usability & Architecture)

#### BLOCKER-07: Untyped UI Canvas Metadata
- **Target Area:** Visual Workflow Builder
- **Severity:** Medium
- **Why It Blocks:**
  Node coordinates (x, y), node dimensions, pan/zoom offsets, and edge connection anchors are currently stuffed into `core.WorkflowDefinition.Metadata` under `metadata.ui.positions`.
  - There is no JSON schema validation for UI positions.
  - If a user moves nodes around the canvas, the backend treats this as arbitrary unvalidated map mutation.
  - Map key non-determinism can corrupt visual layout on reload.
- **Evidence:** `internal/core/workflow.go:21`
- **Resolution Pathway:**
  Formalize a typed `CanvasLayout` struct inside `Metadata` with schema validation in `internal/validate`.

---

#### BLOCKER-08: Absence of Single-Step Isolated Execution ("Test Step")
- **Target Area:** Visual Workflow Builder (n8n-style Test Node)
- **Severity:** Medium
- **Why It Blocks:**
  In n8n, a user building a workflow can click "Test step" to execute a single HTTP request or Python script with mock inputs without running the entire workflow.
  In AWIS, the only execution entrypoint is `Runtime.Submit(ctx, workflowID, version, inputs)`, which launches an entire workflow instance with full EventLog and OCC tracking. There is no API or engine runner seam to execute an isolated step context (`sc core.StepContext`).
- **Evidence:** `internal/engine/engine.go:112` & `sdk/runtime_runner.go:22`
- **Resolution Pathway:**
  Expose a `POST /api/v1/steps/test` endpoint backed by direct runner invocation (`e.runners[step.Type].Run(...)`) without persisting state to the event log.

---

#### BLOCKER-09: Subprocess Environment Variable Leakage in User-Authored Workflows
- **Target Area:** Workflow Creation / Builder Security
- **Severity:** Medium
- **Why It Blocks:**
  When a user can author arbitrary workflows in a GUI builder, allowing them to create a `subprocess` step allows them to execute `env` or `cat /etc/passwd` and read the host's `ANTHROPIC_API_KEY` and configuration secrets (`internal/runner/subprocess/subprocess.go:125`).
- **Evidence:** `internal/runner/subprocess/subprocess.go:125-127`
- **Resolution Pathway:**
  Sanitize subprocess environments before enabling workflow authoring in the GUI.

---

### 2.4 Low Blockers (Papercuts & UX Degradations)

#### BLOCKER-10: Hash Router Parameter Truncation
- **Target Area:** Visual Builder Deep Linking
- **Severity:** Low
- **Why It Blocks:**
  `web/src/router.ts:25, 50` drops query strings (e.g. `#/builder?workflow_id=foo&version=1.0.0`), redirecting to `#/instances` because the route regex fails to match URLs with query strings.
- **Evidence:** `web/src/router.ts:21-25, 38-51`
- **Resolution Pathway:**
  Update router to strip and parse `?query` from `location.hash` before regex pattern matching.
