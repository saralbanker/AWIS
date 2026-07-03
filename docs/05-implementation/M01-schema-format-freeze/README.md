# M01 — Canonical Schema & Format Freeze → **GATE G1**
**Status:** FULLY MATERIALIZED (blocked only by M00) · **Effort:** 2.5d · **Window:** Days 1–3 (Track A, hard full-stop until G1)
**Objective:** Freeze the irreversible artifacts before any code: TDS-01 (EventLog format), TDS-02 (WorkflowDefinition serialization), TDS-03 (grammar conformance corpus); `sdk` package with public types + doc comments only. Explicitly NO executable logic.
**Depends on:** M00 · **Blocks:** M02, M04, M05 (and transitively all)
**Amendments:** **F-1 BINDS HERE** — canonical types live in `internal/core`; `sdk` re-exports via type aliases (kills the sdk⇄internal/engine import cycle; public surface unchanged).
**Primary sources:** IMP §12 (TDS-01/02/03 rows), §27.M1, §23 (G1), §25 IR-1; Blueprint §6, §9; Finalization Blocker 2; Verification F-1
**Compilation spec:** IMPLEMENTATION_KNOWLEDGE_BASE.md §4/M01 (already executed — this module is the output)
**Key ACs:** three TDS docs complete; G1 human sign-off recorded in PR; fixture corpus compiles as Go test data.

## Gate G1 (from IMP §23)
Human answers: *"Would I still accept this EventLog/WorkflowDefinition format in five years?"* (P8, R1, decade-reader test, Constitution Art. 12). Evidence: TDS-01/02/03 + fixture corpus. Failure: iterate M01; nothing downstream starts.

## Module files
README.md · IMPLEMENTATION_SPEC.md · AI_EXECUTION_CONTEXT.md · VALIDATION_CHECKLIST.md · HANDOFF.md · TRACEABILITY.md · DEPENDENCY_MAP.md
