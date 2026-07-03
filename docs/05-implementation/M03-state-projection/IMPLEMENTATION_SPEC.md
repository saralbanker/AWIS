# M03 — Implementation Specification
**Authority:** IMP §27.M3, §14, §26; Blueprint §9 (two-layer state), §8 (loop/claim semantics), §20; TDS-01/02 (G1-frozen, incl. ADJ-5). Materialized at entry per IKB §3/§4-M03.
**Adjudications in force (founder, 2026-07-03):** CONTRA-6 (claim = additive `step_claims` table + optimistic version; InstanceStatus enum untouched; §8 'in_flight' read as informal wording) · CONTRA-7/ADJ-5 (`compensation_failed` added as 9th InstanceStatus per Finalization B4 authority; TDS-02 + core enum amended).

## Objective
Complete the StoragePort: StateStore (upsert/get/list/claim with optimistic versioning) + `rebuild-state` as a library function — the recovery tool that deliberately predates the engine (IMP §26). Registry + cache already live (pulled into M02; recorded there).

## Deliverables
1. **Migration 0001 fold-forward** (pre-tag, sanctioned IMP §14/§26): add
   - `workflow_instances` — Blueprint §20 DDL verbatim + `cancellation_requested INTEGER NOT NULL DEFAULT 0` (IMP §14 greenfield fold-in per Finalization B4) — 12 columns total incl. `version INTEGER NOT NULL DEFAULT 0` optimistic-lock counter; index `idx_instances_ns_status(namespace, status)`.
   - `step_claims (instance_id TEXT NOT NULL, step_id TEXT NOT NULL, worker_id TEXT NOT NULL, claimed_at TEXT NOT NULL, PRIMARY KEY (instance_id, step_id))` — CONTRA-6 additive mechanism.
2. **StateStore methods** (replacing the four M02 stubs):
   - `UpsertInstance(instance, expectedVersion)`: INSERT when instance absent (expectedVersion 0 semantics: create); UPDATE guarded `WHERE version = expectedVersion`, storing `version = expectedVersion + 1`; mismatch → typed `ErrVersionConflict`. On upsert to a TERMINAL status, delete the instance's `step_claims` rows (claim release site, §8 SETTLE "release claim" — no frozen release method exists; recorded EDR-006).
   - `GetInstance` faithful 11-field mapping (+ version handled internally); missing → typed `ErrInstanceNotFound`.
   - `ListInstances(filter)`: `InstanceFilter` shape COMPLETED here (first consumer; pre-M08-mutable per ADJ-4b): `{Namespace string; Status InstanceStatus}` — empty field = no predicate, but NFR-S-04 namespace predicate applies whenever Namespace set; uses `idx_instances_ns_status`.
   - `ClaimStep(instanceID, stepID, workerID)`: single tx — INSERT into `step_claims` (PK conflict → `(false, nil)`); success also bumps instance `version` (optimistic-lock coupling per §8 step 3). Instance absent → error. At-most-once contract-tested.
3. **`rebuild-state` library** (`internal/storage/rebuild.go`, deep-review scope — "Opus review: rebuild fidelity" → CE deep review per EDR-004): `RebuildState(ctx, storage-or-db)`: wipe projection (instances + claims), replay ALL events per instance in `sequence_num` order, apply projection rules (EDR-007), write projected instances. Projection rules (from TDS-01 replay statements + §6/§8, recorded EDR-007):
   - Variable scoping: `variables = {"inputs": <WorkflowStarted.inputs>, "<step_id>": <StepCompleted.outputs>, ...}` (matches TDS-03 dot-path references; collision-free).
   - WorkflowStarted → create: status `running`, StartedAt=emitted_at, variables.inputs; StepStarted → add to current_steps; StepCompleted → remove + variables[step_id]=outputs; StepFailed retrying:true → keep, retrying:false → remove; StepFallbackActivated → remove failed step_id; SignalReceived → status `running` (waiting→running per TDS-01); WorkflowCompleted → `completed`, CompletedAt, current_steps=[]; WorkflowFailed → `failed`, CompletedAt; WorkflowCancelled → `cancelled`, CompletedAt; WorkflowCompensating → `compensating`; WorkflowCompensated → `compensated`, CompletedAt; WorkflowCompensationFailed → `compensation_failed` (ADJ-5), CompletedAt.
   - UpdatedAt = last event's emitted_at. `cancellation_requested` rebuilds as 0 (non-evented request flag — see Limitations).
   - Namespace/definition fields from WorkflowStarted's envelope + first-event context; definition id/version: from envelope? **Not in the envelope** — WorkflowStarted payload carries only inputs. Definition identity comes from `workflow_instances` at submit time (M06). Rebuild sources definition_id/version from... nothing evented. GAP: record in EDR-007 — smallest mechanism: extend WorkflowStarted payload? NO (frozen). Rebuild preserves definition_id/version by reading the pre-wipe row when present; a wiped/lost row leaves them empty pending M06 (which must event them or accept the limitation → G2 docket item).
4. **Contract suite extension** (`storagetest`): all 12 methods now covered — upsert create/update/version-conflict; get not-found; list namespace+status predicates; claim at-most-once (second claim false), claim bumps version, terminal upsert releases claims; rebuild: byte-identical replay (see below).
5. **Tests / checkpoint (IMP §20 M2/M3):** 10K-event fixture replay → projection **byte-identical** (NFR-R-03): generate deterministic event fixture (multiple instances, all 12 event types), project via live Upsert path, snapshot `workflow_instances` rows, wipe, `RebuildState`, snapshot again, assert identical. Crash-variant: SIGKILL mid-append (M02 helper pattern), reopen, rebuild → consistent. 100K rebuild wall time recorded (informational vs NFR-P-06 <30s, binding at M18).
6. **EDR notes:** `docs/edr/edr-006-claim-mechanism.md` (CONTRA-6 disposition + release site) · `docs/edr/edr-007-projection-rules.md` (variable scoping, status mapping, definition-identity gap, waiting-status entry owed to M07).

## Scope walls (FORBIDDEN)
- No engine, no tick loop, no signals/wait_records, no CLI (`awis rebuild-state` command is M14/M17; library only).
- No changes to internal/core beyond: InstanceFilter shape completion + nothing else (enum already amended by CE under ADJ-5).
- No new dependencies. No touching sdk/, CI.
- Registry/cache: already contract-locked in M02 — do not modify.

## Acceptance criteria (IMP §27.M3)
- All 12 StoragePort methods contract-tested; re-register existing (id,version) fails (M02 test still green).
- 10K replay byte-identical (NFR-R-03). CI green incl. `-race`.

## Merge / RB / Repo-after
**Merge:** CI + contract green; CE rebuild-fidelity review recorded. **RB:** revert PR (fold-forward 0001 reverts with it — no external installs). **Repo after:** full persistence layer complete; recovery tool predates everything that could need it.
