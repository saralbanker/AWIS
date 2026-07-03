# M03 — Validation Checklist (all binary; all must pass)
Verified by awis-verifier card M03-V1 on clean clone at HEAD `980d8b6`: 17/18 ✅ + item 4 fixed in `2eb9604` (GetInstanceNotFound subtest added, passes; checklist field-count corrected). Rebuild: 10,849 events/60 instances byte-identical ×2; 6 anchors; crash+rebuild green; 100K rebuild 447ms (informational, NFR-P-06 <30s).
- [x] Clean clone: `make build test lint contract` exit 0; `go test -race ./internal/storage/...` exit 0
- [x] 0001 fold-forward: `workflow_instances` (12 cols incl. `version` + `cancellation_requested INTEGER NOT NULL DEFAULT 0`) + `idx_instances_ns_status` + `step_claims` table exist; N−1 fixture test updated
- [x] UpsertInstance optimistic lock: stale expectedVersion → typed ErrVersionConflict (test)
- [x] GetInstance not-found typed error (contract subtest); 10-field faithful roundtrip (`version`/`cancellation_requested` are DB-internal, not struct fields)
- [x] ListInstances: namespace predicate + status predicate via InstanceFilter{Namespace,Status} (tests; NFR-S-04)
- [x] ClaimStep at-most-once: second claim (false,nil); claim bumps instance version; instance-absent errors (tests)
- [x] Terminal-status upsert deletes the instance's step_claims (EDR-006 release site; test)
- [x] All 12 StoragePort methods contract-tested (no ErrNotImplemented remains)
- [x] Rebuild: 10K-event deterministic fixture, all 12 event types → wipe → RebuildState → row-snapshot BYTE-IDENTICAL (NFR-R-03; test evidence)
- [x] WorkflowCompensationFailed projects to `compensation_failed` (ADJ-5; test)
- [x] Crash mid-append → reopen → rebuild → consistent projection (test)
- [x] 100K rebuild wall time recorded (informational vs NFR-P-06 <30s)
- [x] InstanceFilter completed {Namespace, Status} — only core change beyond CE's ADJ-5 enum edit (`git diff main -- internal/core` shows exactly those two)
- [x] sdk/ untouched; M02 registry/cache/EventLog behavior unaltered (M02 contract tests green unchanged)
- [x] edr-006 + edr-007 committed; CONTRA-6/7 + ADJ-5 recorded in module TRACEABILITY + gate log
- [x] CE rebuild-fidelity review recorded in merge description
- [x] Global DoD (IMP §24) items 1–7 (macOS CI leg pending remote)
