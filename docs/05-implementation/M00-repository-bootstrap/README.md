# M00 — Repository Bootstrap
**Status:** FULLY MATERIALIZED (next executable milestone) · **Effort:** 0.5d · **Window:** Week 0 / Day 1
**Objective:** Empty-but-green repository per IMP §4 — repo, two-module Go workspace, Makefile, CI skeleton, lint config. NO product code.
**Depends on:** nothing · **Blocks:** M01 (and transitively everything)
**Amendments:** none (F-1..F-5 first bind at M01)
**Primary sources:** IMP §4 (bootstrap plan), §5 (repository structure), §21 (CI), §27.M0
**Compilation spec:** IMPLEMENTATION_KNOWLEDGE_BASE.md §4/M00 (already executed — this module is the output)
**Key ACs:** `make build test lint` passes on a clean clone; CI green on Linux + macOS; EDR notes committed.

## Module files
| File | Purpose |
|---|---|
| IMPLEMENTATION_SPEC.md | Exact build steps, scope walls, acceptance criteria |
| AI_EXECUTION_CONTEXT.md | What an AI session loads, model allocation, constraints |
| VALIDATION_CHECKLIST.md | Binary pass/fail exit checklist |
| HANDOFF.md | Guaranteed outputs for M01 (finalized at completion) |
| TRACEABILITY.md | Every task → canonical source coordinate |
| DEPENDENCY_MAP.md | Upstream/downstream artifact flow |
