# AWIS — Architectural Debt Register

**Date:** 2026-09-05 · **HEAD:** `8a87f70` · **Basis:** Tier 1 (source, execution, measurement)

Debt that is **architectural** — inherent to a structural decision, not a coding slip. Coding defects are in `RELEASE_AUDIT_REPORT.md`. Each entry states what was decided, what it costs, whether the cost is currently acceptable, and the *trigger* that makes it unacceptable.

Timing: **Now** = before Beta · **Next** = before the next milestone · **Watch** = accept, but instrument the trigger.

---

## AD-01 · The deliverable has no version-control history — **Now**

**Decision (implicit, never recorded):** the API, server, and GUI were developed directly in the working tree and never committed.

**Cost:** No history means no bisect, no blame, no diff review, no revert, and no CI. This is not merely process debt — it removes the substrate every other engineering control depends on. Every "PASS" issued on the GUI was issued on code no tool ever gated.

**Acceptable now?** No.

**Fix:** Commit the four paths; correct the false `.gitignore` comment. Then re-run the full verification pass, because it will be the first one that has ever actually covered this code.

---

## AD-02 · The frozen 12-method `StoragePort` cannot express bounded reads — **Next**

**Decision:** `core.StoragePort` is frozen verbatim (`ports.go:7-9`) so adapters implement it identically. `ListInstances` and `ReadEvents` take no limit and are therefore **unbounded by contract**.

**Consequence:** Every bounded read must be smuggled in as an *additive* method reached by type assertion — `ListInstancesPaged`, `ReadEventsPaged`, `RecallStore`, `pinger`, `cancellationStore`. There are now at least five of these. The pattern is applied consistently and is well documented, but the architecture has effectively grown a shadow interface that the frozen one is supposed to define.

**Cost, measured:**
- Callers that use the frozen method inherit unbounded reads. `awis status` issues 9 unbounded full-table queries (RA-05).
- Correctness degrades silently: `handleListEvents` returns **501 Not Implemented** if the backend lacks `eventsPagedStore`. A conforming `StoragePort` implementation is a legal AWIS backend on which the GUI's timeline simply does not work.
- The freeze's stated benefit — adapters implement it identically — is already void, since a Postgres adapter must also implement five undeclared interfaces to be usable.

**Recommendation:** Do not unfreeze mid-Beta. **Do** record the additive set as a documented second-tier contract with its own conformance tests, so a future adapter author knows the real requirement. Revisit the freeze at the Postgres milestone.

---

## AD-03 · `SetMaxOpenConns(1)` shared between the engine loop and HTTP reads — **Watch (trigger is close)**

**Decision:** `internal/storage/db.go:41` — one connection per process, for single-writer semantics.

**Sound for the CLI**, where the process *is* the engine. **Consequential for `awis-server`**, where the same single connection serves the engine tick loop *and* every HTTP request.

**Measured** (`GET /api/v1/instances`, 200 requests):

| concurrency | p50 | p99 | throughput |
|---|---|---|---|
| 1 | 0.41 ms | 2.0 ms | 2,155 rps |
| 10 | 2.47 ms | 9.3 ms | 3,441 rps |
| 50 | 11.43 ms | 38.7 ms | 3,473 rps |
| 100 | 23.22 ms | 58.6 ms | 3,100 rps |

Throughput plateaus at ~3.4k rps while p50 rises **linearly with concurrency** — the signature of a single-server queue. Confirmed serialization.

**Caveat, stated honestly:** measured on a trivial 1-row dataset, so this is the *best* case. Per-query cost rises with data volume and the ceiling falls proportionally.

`FINAL_VERDICT.md` deferred this (D-09) as "undemonstrated at V1's current single-tenant scale." That was fair; it is now demonstrated. It remains acceptable for single-operator Beta.

**Trigger:** any second concurrent dashboard user, or any long-running write transaction (a `rebuild-state` on a large log will stall every HTTP read for its duration).

**Recommendation:** Separate read and write pools — SQLite in WAL mode supports concurrent readers with a single writer, so this is achievable without changing the concurrency model.

---

## AD-04 · The FTS index is rebuilt per query rather than maintained — **Next**

**Decision:** `recall.go:73` — lazy full rebuild before each search, explicitly to avoid per-insert trigger overhead.

**The tradeoff was chosen on a false dichotomy.** The alternatives are not "trigger on every insert" vs "rebuild on every query"; the standard answer is an incremental sync watermark — reindex only events newer than the last indexed `sequence_num`.

**Measured** at 80k events: 3.08s first query, **6.69s second** — cost is unconditional and grows with total log size, not with new data.

**Recommendation:** Incremental sync keyed on `sequence_num`. Contained change, large win, no schema break.

---

## AD-05 · Event-log durability is `synchronous=NORMAL` — **Next (document, then decide)**

**Decision:** `db.go:112` sets `PRAGMA synchronous=NORMAL` under WAL.

**What this actually means:** the database stays *consistent* across an OS crash or power loss, but **recently committed transactions can be lost** — WAL+NORMAL does not fsync on every commit. For an ordinary application this is the right, standard tradeoff.

AWIS is not an ordinary application: it is **event-sourced**, and its recovery story rests on the EventLog being the authoritative record. A lost tail of committed events means a rebuilt projection legitimately disagrees with what the system already told the user had happened.

**I am not calling this a defect** — it is a defensible choice, and `FULL` would materially slow every step transition. It is undocumented debt: no report, ADR, or doc records that AWIS's durability guarantee stops short of crash-proof commit.

**Recommendation:** Record the guarantee explicitly, and consider exposing `synchronous` as configuration so an operator who wants `FULL` can pay for it.

---

## AD-06 · No downgrade guard in the migration runner — **Now (cheap)**

`migrate.go:82` applies migrations above `current` and never checks whether `current` exceeds the newest migration the binary knows. An older binary silently operates against a newer schema.

Structural because it is a property of the versioning *scheme*, not a missing `if`. Fix is still one `if`: refuse to open when `current > maxKnownVersion`. Do it before anyone has two versions in the field.

---

## AD-07 · The HTTP surface has no security layer to extend — **Now**

Not "auth is missing" (RA-03) but the structural fact that there is **no middleware seam at all**. `NewRouter` maps patterns directly to `wrap(handler)`; `wrap` handles only error mapping. There is no place to add auth, CORS, rate limiting, request logging, or tracing without touching all seven routes.

**Recommendation:** Introduce a middleware chain now, while there are seven routes and one consumer. Ship it with request logging and the timeouts from RA-04 even if auth lands later.

---

## AD-08 · Read-only API with no mutation seam — **Watch (correctly deferred)**

The API exposes GETs only; every mutation goes through the CLI. Prior reports flag this as Phase 2 scope and **that is correct** — I re-derived it and agree. Recorded here only so it is not mistaken for an oversight.

Note the coupling to AD-07: mutation endpoints without an auth seam would be a genuine P0, so AD-07 must land first.

---

## AD-09 · Frontend types are unenforced — **Next**

`web/build.mjs` runs esbuild, which strips types without checking them; `typescript` is a devDependency nothing invokes. Architectural rather than trivial because the GUI's correctness argument *rests* on TypeScript — chosen for safety, then never enforced. Add `tsc --noEmit` to the build and to CI.

---

## Summary

| ID | Debt | Timing |
|---|---|---|
| AD-01 | Deliverable has no VCS history | **Now** |
| AD-06 | No migration downgrade guard | **Now** |
| AD-07 | No middleware seam on HTTP surface | **Now** |
| AD-02 | Frozen port can't express bounded reads | Next |
| AD-04 | FTS rebuilt per query | Next |
| AD-05 | Durability guarantee undocumented | Next |
| AD-09 | Frontend types unenforced | Next |
| AD-03 | Single shared SQLite connection | Watch |
| AD-08 | No mutation API | Watch |
