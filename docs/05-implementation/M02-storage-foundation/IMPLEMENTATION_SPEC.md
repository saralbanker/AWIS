# M02 — Implementation Specification
**Authority:** IMP §27.M2, §14; Blueprint §20 (L1323–1465), §9 (L616–695); TDS-01/02 (G1-frozen). Materialized at entry per IKB §3/§4-M02. This spec transcribes; it does not design.

## Objective
StoragePort's SQLite adapter for the M02 portions — migration runner, migration `0001` (events/definitions/cache), EventLog append/read, WorkflowRegistry, StepResultCache — plus the adapter-agnostic StoragePort contract-test suite (reused verbatim for the V2 Postgres adapter). StateStore methods stub with a defined error until M03.

## Deliverables (ordered per IMP §14 implementation order)
1. **Dependency pin:** `modernc.org/sqlite` (pure Go, CGO_DISABLED) — per edr-003; version pinned at this milestone. The ONLY new require. (testify approved by IMP §3 for tests but stdlib `testing` preferred; add only if materially needed.)
2. **DB open + config:** `internal/storage` — open `runtime.db`, WAL journal mode (edr-003), busy timeout, single connection semantics per Blueprint §9 "no concurrent writers" (local mode).
3. **Migration runner:** embedded SQL via `go:embed`, sequential, forward-only; auto-runs on open in local mode (FR-ST-06); `schema_version` migration-tracking table; every migration tested from a fixture DB at version N−1 (IMP §14; N−1 for 0001 = empty DB).
4. **Migration `0001_core_execution.sql` — M02 portions only** (M03 folds in `workflow_instances`, pre-tag fold-forward per IMP §14/§26):
   - `schema_version` tracking table (runner-owned).
   - `execution_events` — TDS-01 §1 envelope VERBATIM: 9 columns incl. `schema_version INTEGER NOT NULL DEFAULT 1` (G1 ADJ-1); indexes `idx_events_instance_seq(instance_id, sequence_num)`, `idx_events_ns_time(namespace, emitted_at)` (Blueprint §20).
   - `workflow_definitions` — Blueprint §20 DDL verbatim: `(id, version, namespace, definition JSON, registered_at)`, PK `(id, version)`.
   - `step_results_cache` — Blueprint §20 DDL verbatim: `(idempotency_key PK, result JSON, cached_at, expires_at)`.
5. **EventLog paths** (deep-review scope per IMP §28 "Opus review on append path" → CE deep review per EDR-004):
   - `AppendEvent`: append-only insert; enforces per-instance strictly-increasing `sequence_num` inside the append transaction (rejects `<= last`); never updates or deletes (FR-ST-01). Sequence ASSIGNMENT stays with the caller (engine, M06) — enforcement-only here; recorded as **EDR-005** (additive, reversible pre-tag).
   - `ReadEvents(instanceID, fromSeq)` ordered by `sequence_num`; `ReadEventRange(namespace, from, to)` namespace-scoped (NFR-S-04).
   - Column↔struct mapping preserves all 9 TDS-01 fields byte-faithfully; `payload` stored as raw JSON.
6. **WorkflowRegistry:** `RegisterWorkflow` — serialized TDS-02 JSON into `definition`; re-registering an existing `(id, version)` FAILS (immutability, TDS-02 §5); `GetWorkflow`, `ListWorkflows(namespace)` (NFR-S-04 predicate).
7. **StepResultCache:** `CacheResult(key, result, ttl)` / `GetCachedResult` honoring `expires_at`; time via an injectable clock (IMP §3 determinism rule — all clocks injectable from M2 onward).
8. **StateStore stubs:** `UpsertInstance/GetInstance/ListInstances/ClaimStep` return a package-level `ErrNotImplemented` annotated "M03"; SQLiteStorage must still satisfy `core.StoragePort` at compile time.
9. **Contract suite:** exported adapter-agnostic package (`internal/storage/storagetest`) driven by a `func() core.StoragePort` factory — the V2-Postgres reuse artifact (DoD). Coverage: append/read roundtrip with all 9 fields + StepError-bearing payloads; monotonicity rejection; range query namespace+time filtering; registry immutability; TDS-02 JSON fidelity (incl. `schema_version`); cache TTL expiry via injected clock.
10. **Crash-durability test (NFR-R-02, §20-checkpoint):** helper process appends events; SIGKILL mid-stream; reopen DB; every committed event present, none corrupted. Plus **100K-event append**: completes cleanly; wall time recorded (informational — the binding 100K targets are NFR-P-05/06 at M03/M14).
11. **Makefile:** `contract` target replaces its NOT-YET stub (runs the contract suite).

## Scope walls (FORBIDDEN)
- No StateStore behavior (M03), no `rebuild-state` (M03), no signals/audit/domain_events/plugin tables (M06/M07/M12), no engine, no CLI.
- No changes to `internal/core` or `sdk` (a type gap → STOP, never patch).
- No second dependency beyond the SQLite driver (+ optionally testify per IMP §3).
- Namespace isolation: every query whose frozen signature carries `namespace` filters by it (NFR-S-04).

## Acceptance criteria (IMP §27.M2)
- Contract suite green. Crash mid-write: committed events survive (NFR-R-02).
- Append 100K events: clean completion, timing recorded.
- Re-registering existing (id, version) fails.
- CI green incl. `-race` on the storage package.

## Merge / RB / Repo-after
**Merge:** CI + contract green; CE deep review of append path recorded. **RB:** revert PR. **Repo after:** durable event recording exists.
