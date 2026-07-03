# M01 — Validation Checklist (all binary; all must pass)
Verified by awis-verifier card M01-V1 on clean clone at HEAD `d66df49` (2026-07-03): **PASS 21/21**.
## Specs
- [x] `docs/EVENTLOG_FORMAT.md` (TDS-01) exists, ≤3 pages (1083 words), versioned, `schema_version` semantics defined (§1.1, G1 ADJ-1)
- [x] Every Blueprint §9 event type has a payload schema (count: **12**; §8 cross-check clean)
- [x] Replay-sufficiency statement per event type (IR-1 / G1 checklist item)
- [x] `docs/WORKFLOW_SCHEMA.md` (TDS-02): full JSON serialization + semver/immutability rules (+ `schema_version` per G1 ADJ-3)
- [x] `docs/EXPRESSION_GRAMMARS.md` (TDS-03): both grammars + prohibited lists VERBATIM from Finalization Blocker 2 (BEGIN/END verbatim markers; diffable)
- [x] Conformance fixture table present: valid AND invalid corpus for both grammars (15 template + 27 condition = 42; doc rows == corpus entries)
## Types (F-1)
- [x] Canonical types in `internal/core` (leaf — stdlib imports only); `sdk` contains only type aliases + doc comments
- [x] `sdk` files match FR-SDK-02 layout: awis.go workflow.go step.go trigger.go runner.go intelligence.go recall.go (exactly 7)
- [x] Zero executable logic in either package (verified: zero `func` keyword in internal/core and sdk)
- [x] `go build ./...` green (both modules); fixtures compile as Go test data (DoD)
## Gate G1
- [x] `schema_version` present on every table/format (envelope column per ADJ-1; definition field per ADJ-3)
- [x] Decade-reader read-through performed and noted (founder review at G1, 2026-07-03)
- [x] Human sign-off recorded — **G1 APPROVED 2026-07-03** with ADJ-1..4 dispositions (04-planning gate log; TRACEABILITY; recorded in milestone PR/merge description at merge)
- [x] Global DoD (IMP §24) items 1, 2, 5, 6, 7 verified (item 7 merge approval = G1 verdict rendered; squash-merge pending founder; item 2 macOS leg pending first push, Linux verified)
