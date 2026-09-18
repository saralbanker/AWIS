# Document Drift and Planning Corpus Audit

**Date:** 2026-09-03  
**Auditor:** Adversarial Release Verification Team  
**Evaluation Scope:** All documents under `docs/09-gui-planning/` and root architectural specifications vs. Current Codebase Reality (`engine-hardening` @ `e75c1f1` + uncommitted API/GUI implementation).  
**Authority Rule:** If any document conflicts with code, tests, or runtime behavior: **CODE AND RUNTIME WIN**.

---

## 1. Executive Summary

The `docs/09-gui-planning/` directory contains 38 markdown files totaling over 440KB of planning, architecture, decision records, and roadmaps produced across multiple iterations (2026-08-30 through 2026-09-01).

While these documents were thorough at the time of drafting, **rapid implementation of GUI Phase 1 MVP and GUI Beta has introduced substantial document drift**. Furthermore, several foundational planning assumptions have been falsified by actual code and runtime execution:
1. Documents claim prerequisite gates (such as merging to `main`, deciding founder decision `D1`, and applying `migration 0007_state_changes.sql`) were mandatory before GUI work could proceed. In reality, both Phase 1 and Beta were implemented without them.
2. Status tracking documents (`DASHBOARD_GATE_STATUS.md`, `GUI_BETA_WORK_BREAKDOWN.md`) describe implementation cards as "Not started" or "In backlog" when they are already fully implemented in the working tree.
3. PRD and Screen Specs specified descending updated-time order (`updated_at DESC`) for the landing screen, but the backend implemented ascending start-time order (`started_at ASC`), creating a severe UI failure that was masked rather than corrected.

---

## 2. Stale Plans & Completed Backlogs Still Marked "Pending"

### 2.1 `DASHBOARD_GATE_STATUS.md` Is Completely Obsolete
- **Document Text:**
  ```markdown
  | Card | Status |
  |---|---|
  | `E-G4-4/5` (read-only workflow + instance routes) | ⬜ Not started — unblocked |
  | `E-G4-6` (event-history route) | ⬜ Not started — unblocked, and not required for the gate itself |
  ```
- **Code Reality:**
  Both cards are **fully implemented, tested, and running**:
  - `E-G4-4/5`: Implemented in `internal/api/workflows.go` (`handleListWorkflows`, `handleGetWorkflow`) and `internal/api/instances.go` (`handleListInstances`, `handleGetInstance`).
  - `E-G4-6`: Implemented in `internal/api/events.go` (`handleListEvents`) and `internal/storage/sqlite.go:202-225` (`ReadEventsPaged`).
  - Unit tests in `internal/api/router_test.go`, `internal/api/workflows_test.go`, `internal/api/instances_test.go`, and `internal/api/events_test.go` all pass.

---

### 2.2 `GUI_BETA_WORK_BREAKDOWN.md` Backlog Is 100% Implemented
- **Document Text:**
  Presents 13 cards (`BE-1`, `BE-2a`, `BE-2b`, `FE-1` through `FE-8`) as an unstarted backlog requiring ~35-39 engineering hours.
- **Code Reality:**
  Every single card is fully implemented in the working tree:
  - `BE-1` (`definition_id` filter): Landed in `internal/core/ports.go`, `internal/storage/sqlite.go:963`, and `internal/api/instances.go:79`.
  - `BE-2a` (shared version package): Landed in `internal/buildinfo/buildinfo.go` and `cmd/awis/main.go`.
  - `BE-2b` (`GET /api/v1/info`): Landed in `cmd/awis-server/main.go:69`, `internal/api/info.go`, and `internal/api/router.go:34`.
  - `FE-1` (Workflow Detail screen): Landed in `web/src/screens/workflowDetail.ts`.
  - `FE-2` (`definition_id` filter UI): Landed in `web/src/screens/instanceList.ts`.
  - `FE-3` (Jump to instance ID): Landed in `web/src/screens/instanceList.ts`.
  - `FE-4a/b` (Freshness indicator): Landed in `web/src/ui.ts` and `web/src/screens/instanceList.ts`.
  - `FE-5` (Onboarding empty state): Landed in `web/src/onboarding.ts`.
  - `FE-6a/b` (Connection health / version indicator): Landed in `web/src/main.ts:73-104`.
  - `FE-7` (Preserve row order on refresh): Landed in `web/src/screens/instanceList.ts:103-122`.
  - `FE-8` (Event timeline pagination & payload collapse): Landed in `web/src/screens/eventTimeline.ts:21, 99-106`.

---

### 2.3 `FINAL_VERDICT.md` Git Base & Commit Pin Drift
- **Document Text:**
  `docs/09-gui-planning/FINAL_VERDICT.md:5-6`:
  > "Basis: `engine-hardening` @ `f004f4f` + uncommitted B-31 remediation, independently re-confirmed current as of 2026-09-01."
- **Code Reality:**
  The working repository is on commit `e75c1f1` ("Close B-31 silent-defaults validation class"), which closed the B-31 remediation. Additionally, the working tree has 7 modified files and 5 untracked directories representing the completed API server and GUI implementation.

---

## 3. Invalid Assumptions

### 3.1 Assumption: "Passing Test Suite Implies Production / Beta Readiness"
- **Assumption in Docs:**
  All transition documents assume that because the 12-method `StoragePort` contract suite and engine integration tests pass, the engine is rock-solid.
- **Falsification by Code:**
  - `internal/engine/restart_test.go:31` only tests restarting when **zero steps are in-flight** (step `a` has completed, step `b` has not started).
  - In reality, restarting with an in-flight step deadlocks the instance forever because `hydrate()` does not recover in-flight claims (Finding SEC-01).
  - `internal/intelligence/adapters/anthropic/live_test.go:28` skips live tests (`t.Skip`), masking that the hardcoded model names `claude-haiku-4-5` and `claude-sonnet-5` fail on live Anthropic APIs (Finding SEC-05).

---

### 3.2 Assumption: "Landing Page Displays Recent Activity"
- **Assumption in Docs:**
  `GUI_PHASE1_SCREEN_SPEC.md §1`:
  > "Landing screen displays recent instance activity, ordered by updated_at descending."
- **Falsification by Code:**
  The backend implementation `internal/storage/sqlite.go:812` uses:
  ```sql
  ORDER BY started_at, instance_id LIMIT ? OFFSET ?
  ```
  This is **ascending order (oldest first)**. The frontend in `instanceList.ts:96-101` hacks a client-side sort on the returned page of 50 items. In a database with >50 instances, recent instances never appear on Page 1.

---

### 3.3 Assumption: "Namespaces Provide Tenant Isolation"
- **Assumption in Docs:**
  `GUI_PRD.md §4` and `GUI_ARCHITECTURE.md §3` treat namespaces as administrative tenant boundaries.
- **Falsification by Code:**
  In `migrations/0001_core_execution.sql:18`, `workflow_definitions` has `PRIMARY KEY (id, version)`. Namespace is not part of the primary key. Two different namespaces cannot register the same workflow ID and version. Namespace is an unenforced filter label, not an isolation boundary.

---

## 4. Direct Contradictions Between Documents and Implementation

| Topic | Planning Document Assertion | Actual Code / Runtime Reality |
|---|---|---|
| **Prerequisite Gates** | `FINAL_VERDICT.md`: G0-G2 (`state_changes` migration 0007), founder decision D1, and merging `engine-hardening` to `main` are mandatory before GUI work. | GUI Phase 1 and Beta were implemented directly on `engine-hardening` without D1, without migration 0007, and without merging to `main`. |
| **Live State Updates** | `GUI_ARCHITECTURE.md §5`: Live tracking requires Server-Sent Events (SSE) and `state_changes` streaming; polling is rejected. | The implemented GUI relies entirely on 15-second client polling (`setInterval(..., 15000)` in `instanceList.ts` and `instanceDetail.ts`). |
| **Instance Sorting** | `GUI_PHASE1_SCREEN_SPEC.md §1`: Orders instances by `updated_at DESC`. | `internal/storage/sqlite.go:812`: Orders by `started_at, instance_id ASC`. |
| **Workflow Detail Design**| `GUI_PHASE1_SCREEN_SPEC.md §5`: Designed a table with a generic `Handler` column for all step types. | `web/src/screens/workflowDetail.ts:181-204`: Corrected to type-specific details because `intelligence` and `signal` steps carry no `handler` field. |
| **Health Check Validity** | `GUI_ARCHITECTURE.md §6`: `/healthz` verifies system operational readiness. | `internal/api/healthz.go:16-20`: Hardcoded static return `{"status":"ok"}` without checking DB connection or engine loop health. |

---

## 5. Superseded Decisions

1. **"GUI work must wait for engine freeze and master branch merge" (Superseded):**  
   Proven false by the working implementation in `web/` and `cmd/awis-server/`.
2. **"Disable clickable workflow rows in Phase 1 to prevent dead links" (Superseded):**  
   Card `FE-1` implemented `web/src/screens/workflowDetail.ts` and re-enabled row clicks.
3. **"Subprocess environment scrubbing can be deferred to G10 (Multi-user)" (Invalidated):**  
   Recorded as deferred in `DEFERRED_WORK_REGISTER.md`. In reality, this is an active security vulnerability in single-user Beta that leaks host `ANTHROPIC_API_KEY` to any subprocess.
4. **"Lazy FTS rebuild is safe for developer use" (Invalidated):**  
   Rebuilding the entire FTS table on every query via `DELETE + INSERT SELECT` (`internal/storage/recall.go:128-152`) acquires exclusive SQLite write locks, starving concurrent engine execution.

---

## 6. Required Document Reconciliation Actions

1. Archive or mark as superseded:
   - `docs/09-gui-planning/DASHBOARD_GATE_STATUS.md`
   - `docs/09-gui-planning/STARTLINE_EXECUTION_REPORT.md`
   - `docs/09-gui-planning/STARTLINE_INTEGRATION_REPORT.md`
2. Update `GUI_BETA_WORK_BREAKDOWN.md` and `GUI_BETA_PRD.md` to reflect that all 13 Beta cards are implemented.
3. Correct `GUI_PHASE1_SCREEN_SPEC.md` and `GUI_PRD.md` to specify `ORDER BY started_at DESC` as the authoritative query requirement.
