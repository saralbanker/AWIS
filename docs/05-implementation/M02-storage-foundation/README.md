# M02 — Storage Foundation
**Status:** Partitioned — materialize at entry (see ../README.md policy) · **Effort:** 2d · **Window:** Week 1 (Track A)
**Objective:** StoragePort interface + SQLite adapter + migration runner + EventLog append/read + migration 0001 (events/definitions/cache portions) + StoragePort contract-test suite (reused verbatim for V2 Postgres).
**Depends on:** M01 (TDS-01/02, G1 approved) · **Blocks:** M03
**Amendments:** none directly (F-2's `domain_events` table lands with M06's scope)
**Primary sources:** IMP §27.M2, §14; Blueprint §9, §20; TDS-01/02 · **Compilation spec:** IMPLEMENTATION_KNOWLEDGE_BASE.md §4/M02
**Key ACs:** contract suite green; WAL crash-durability demo (kill mid-write, committed events survive — NFR-R-02); namespace predicate on every query (NFR-S-04).
