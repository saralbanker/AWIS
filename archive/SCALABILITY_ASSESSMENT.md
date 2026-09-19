# AWIS — Scalability Assessment

**Date:** 2026-09-05 · **HEAD:** `8a87f70`
**Basis:** Measured on this repository's binaries against seeded databases of 2,000–40,000 instances and 80,000 events. All figures below are observed, not modelled, unless explicitly labelled *extrapolated*.

**Headline:** AWIS is comfortable to roughly **5,000 instances** and becomes operationally painful **past ~10,000**. The binding limit is not SQLite, the engine, or the API — all three are fine. It is an **O(n²) sort in the CLI's status path**, and it is cheap to fix.

---

## 1. Method

Seeded via the real `storage` API (`UpsertInstance` + `AppendEvent`) — not raw SQL — so rows are byte-identical to production. Instances carry a realistic `variables` map; each has 4 events. Measurements from the real `awis` and `awis-server` binaries.

Reference dataset: **20,000 instances / 80,000 events → 34 MB** on disk.

---

## 2. Current limits

### 2.1 CLI — the binding constraint

`awis status`, default invocation (which displays **10 rows**):

| instances | wall time |
|---|---|
| 2,000 | 0.08 s |
| 5,000 | 0.58 s |
| 10,000 | 1.33 s |
| 20,000 | **4.93 s** |

`awis history` at 20,000: **5.17 s**.

**Attribution.** Isolating `sortByUpdatedAt` (`cmd/awis/status.go:408`) from all I/O and decoding:

| instances | sort alone |
|---|---|
| 10,000 | 1.21 s |
| 20,000 | **4.73 s** |
| 40,000 | **19.08 s** |

**4.73 s of the 4.93 s total is the sort.** The curve is exactly quadratic — 4× per doubling. It is an insertion sort, written that way (per the call-site comment) to avoid importing `sort`.

Compounding it, `ListInstances` is unbounded by contract (AD-02), so `status` runs **9 full-table queries** — one per active and terminal status — and JSON-decodes `variables` for every row, to print 10.

**The control case proves it is not inherent.** `awis metrics` answers a harder question over the same 20,000 instances in **0.37 s**, because it aggregates in SQL. The right pattern already exists in this codebase.

### 2.2 HTTP API — properly bounded

Correctly paginated (`ListInstancesPaged`, `ReadEventsPaged`), limits clamped (default 100, max 1000). On the same 20,000-instance database:

```
GET /api/v1/instances              → 0.19 s
GET /api/v1/instances?limit=1000   → 0.24 s
```

**The API scales where the CLI does not.** The B-17 pagination work was real and effective.

### 2.3 API concurrency — serialized

`GET /api/v1/instances`, 200 requests:

| concurrency | p50 | p95 | p99 | throughput |
|---|---|---|---|---|
| 1 | 0.41 ms | 0.67 ms | 2.05 ms | 2,155 rps |
| 10 | 2.47 ms | 6.47 ms | 9.25 ms | 3,441 rps |
| 50 | 11.43 ms | 28.70 ms | 38.66 ms | 3,473 rps |
| 100 | 23.22 ms | 52.58 ms | 58.56 ms | 3,100 rps |

Throughput plateaus at ~3.4k rps while p50 grows **linearly with concurrency** — a textbook single-server queue, confirming `SetMaxOpenConns(1)` (AD-03) serializes all API access.

**Stated honestly:** measured on a 1-row dataset, so this is the *ceiling*. Real queries cost more and the ceiling falls proportionally. ~3.4k rps is nonetheless far beyond single-operator Beta needs.

### 2.4 Search — degrades with total log size

`awis recall` on 80,000 events: **3.08 s** first run, **6.69 s** second. The index is rebuilt unconditionally per query (AD-04), so cost tracks *total* log size, not new data — and repeated searches get slower, not faster.

### 2.5 Storage growth

34 MB / 20k instances / 80k events ≈ **1.7 KB per instance**, ≈ 425 B per event. Linear and modest. Extrapolated: 100k instances ≈ 170 MB; 1M ≈ 1.7 GB. Disk is not a limit.

But `prune-events` is **dry-run only in V1**, so there is no supported way to reclaim it. The EventLog grows monotonically forever, and every unbounded read (§2.1) and every FTS rebuild (§2.4) gets slower with it.

---

## 3. Failure modes

Ordered by how soon they bite.

**F1 — `awis status` becomes unusable (~20k instances).**
5s at 20k; **extrapolated** ~2 minutes at 100k. Operators lose their primary monitoring command *and* their incident-response tool, precisely when the system is busiest. Highest-priority scalability risk.

**F2 — Memory growth on unbounded reads.**
`status` materializes every matching instance with decoded `variables` into a slice, 9 times. At 20k this is tolerable; at 1M instances the CLI would attempt to load the entire table into memory. **Extrapolated — I did not reproduce OOM.**

**F3 — Writer stalls block all HTTP reads.**
One connection shared between the engine loop and the API (AD-03). A long write — `rebuild-state` over a large EventLog is the obvious case — stalls **every** API request for its duration. The GUI freezes with no indication of why. Not reproduced, but it follows directly from `SetMaxOpenConns(1)`.

**F4 — Search cost grows without bound.**
Full FTS rebuild per query against a log that can never be pruned. 3–7s at 80k events; grows indefinitely.

**F5 — Cross-process write contention.**
Every `awis` invocation is a separate process on the same file. `_txlock=immediate` + `busy_timeout(5000)` handle this correctly — and the code comment explaining why is accurate and unusually good. But the timeout is a hard 5s: enough concurrent submitters and writes fail with `SQLITE_BUSY` rather than queueing. Not reached in testing; the mitigation is sound for single-operator use.

**F6 — Connection exhaustion via idle sockets.**
No HTTP timeouts (RA-04): I held a connection open 40s with an incomplete request. Unlimited concurrent connections plus no read timeout means a trivial client exhausts file descriptors. Reproduced.

**F7 — Unbounded event history on one instance.**
`ReadEventsPaged` bounds the API. `ReadEvents` — the frozen, unbounded port method — is still used elsewhere; a single long-running instance with a very large event count will materialize all of it on those paths.

---

## 4. Growth risks by dimension

| Dimension | Comfortable | Degraded | Breaking | Bound by |
|---|---|---|---|---|
| Total instances | ≤ 5,000 | 10k–20k | > 50k | **CLI sort (F1)** |
| Events per instance | ≤ 10,000 | — | unbounded reads | Frozen port (AD-02) |
| Total events | ≤ 50,000 | 80k+ | no pruning | **FTS rebuild (F4)** |
| Concurrent API clients | 1–10 | 10–50 | > 100 | Single connection (F3) |
| Concurrent writer processes | 1–5 | — | busy timeout | 5s `busy_timeout` (F5) |
| Disk | — | — | — | Not a limit |

**Multi-tenancy** is out of scope for Beta and correctly deferred, but note the compounding: namespace filtering happens in the `WHERE` clause of the same unbounded queries, so a second tenant's data slows down the first tenant's `status`. Multi-tenancy must not ship before §5.1 and §5.2.

---

## 5. Recommendations, in order

**5.1 Replace the insertion sort with `sort.Slice`.** One line. Removes F1 outright and takes `status` at 20k from 4.93s to roughly 0.2s. *Highest value-to-effort ratio in this report.*

**5.2 Route `status` and `history` through `ListInstancesPaged`.** Already exists. Removes the 9 unbounded queries and F2.

**5.3 Set HTTP server timeouts.** Two lines. Removes F6.

**5.4 Make the FTS sync incremental** on a `sequence_num` watermark. Removes F4.

**5.5 Split read and write connection pools.** WAL supports concurrent readers with one writer; this removes F3 without changing the concurrency model.

**5.6 Ship real event pruning** before the EventLog is the binding constraint.

Items 5.1–5.3 are roughly an afternoon's work and address the three failure modes actually reproduced in this audit.
