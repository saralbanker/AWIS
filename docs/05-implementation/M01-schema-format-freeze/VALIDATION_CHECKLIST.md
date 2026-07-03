# M01 — Validation Checklist (all binary; all must pass)
## Specs
- [ ] `docs/EVENTLOG_FORMAT.md` (TDS-01) exists, ≤3 pages, versioned, `schema_version` semantics defined
- [ ] Every Blueprint §9 event type has a payload schema (count recorded in PR; 12+ expected)
- [ ] Replay-sufficiency statement per event type (IR-1 / G1 checklist item)
- [ ] `docs/WORKFLOW_SCHEMA.md` (TDS-02): full JSON serialization + semver/immutability rules
- [ ] `docs/EXPRESSION_GRAMMARS.md` (TDS-03): both grammars + prohibited lists VERBATIM from Finalization Blocker 2 (diff against source recorded)
- [ ] Conformance fixture table present: valid AND invalid corpus for both grammars
## Types (F-1)
- [ ] Canonical types in `internal/core`; `sdk` contains only type aliases + doc comments
- [ ] `sdk` files match FR-SDK-02 layout: awis.go workflow.go step.go trigger.go runner.go intelligence.go recall.go
- [ ] Zero executable logic in either package (reviewed, stated in PR)
- [ ] `go build ./...` green; fixtures compile as Go test data (DoD)
## Gate G1
- [ ] `schema_version` present on every table/format
- [ ] Decade-reader read-through performed and noted
- [ ] Human sign-off recorded in the PR (merge blocker)
- [ ] Global DoD (IMP §24) items 1, 2, 5, 6, 7 verified
