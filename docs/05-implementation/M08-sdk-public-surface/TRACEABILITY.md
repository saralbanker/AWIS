# M08 — Traceability
Every task → canonical coordinate. Cards: docs/05-implementation/M08-sdk-public-surface/cards/.

| Row | Task (card) | Canonical coordinate |
|---|---|---|
| T1 | `NewRuntime` / `Config` constructor | Blueprint §12 L934–950; IMP §17 L322; IMP §27.M8 Obj |
| T2 | `RegisterHandler` / `RegisterWorkflow` + PRD §18 errors | PRD §18 L1200–1206; IMP §27.M8 Obj; Blueprint §12 L1341 |
| T3 | `WorkflowBuilder.Build()` semver validation | IMP §27.M8 AC; PRD §18 L1214; FR-SDK-01/11 |
| T4 | `WorkflowRegistered` audit call site | **F-4** (first M08 call site: WorkflowRegistered); NFR-S-05 |
| T5 | `WorkflowRunner` (Submit/Signal/Status/Cancel/List) | Blueprint §12 L892–910; IMP §27.M8 Obj; IMP §13 |
| T6 | `core.WorkflowStatus` shape freeze | Blueprint §12 L171–176; IMP §13 (sdk surface mutable until M08) |
| T7 | `RecallAPI` (QueryHistory/StepStats) | Blueprint §12 L913–925; IMP §27.M8 Obj |
| T8 | `core.HistoryQuery`, `ExecutionRecord`, `StepStatistics` shape freeze | Blueprint §12; IMP §13 |
| T9 | TriggerAPI sdk intake surface (F-2) | **F-2** (M08 TriggerAPI row); cross-reference-index trigger row |
| T10 | Example app + FR-SDK-09 boundary proof | IMP §27.M8 AC (FR-SDK-03/04/05/09); IMP §5 L110 |

## Notes / dispositions
- **F-1 alias surface:** type aliases in `sdk/` point at `internal/core` definitions frozen at G1.
  M08 adds behavior (constructors, method implementations) without modifying the alias targets
  except for the four shape-completion types (WorkflowStatus, HistoryQuery, ExecutionRecord,
  StepStatistics) owned by M08 per IMP §13.
- **F-2 TriggerAPI:** `TriggerAPI.SubmitEvent` → `engine.Ingest` → domain_events table
  (SCAN_TRIGGERABLE path from M06). No new storage methods.
- **CONTRA-1 note (module path):** sdk package is `github.com/awis/awis/sdk` per CONTRA-1 disposition.
  Example app uses this path; V2 vanity-import redirect is out of scope.

## Execution record (appended during B-BUILD/C-VERIFY; closed at D-CLOSE)
| Card | Agent | Commit | Outcome |
|---|---|---|---|
| M08-C1 | awis-builder | 7d23931 | DONE — Runtime constructor, registration, WorkflowBuilder, core shapes (T1–T4, T6, T8) |
| M08-C2 | awis-builder | 92828e7 | DONE — WorkflowRunner, RecallAPI, TriggerAPI, hello_workflow (T5, T7, T9, T10) |
| M08-V1 (1st) | awis-verifier | at 8bd7bf5 | **FAIL** — 5 test gaps (signal-resume, cancel-status, InstanceID, StepStats, real audit_log) |
| M08-C2r | awis-builder | e8aca86 | DONE — closed all 5 gaps (revision card per EEOS §4.2) |
| M08-V1 (2nd) | awis-verifier | at 728cfed | **PASS 21/21** — full checklist ✅ |

### D-CLOSE semantic review (Fable, 2026-07-08)
- **No architectural drift.** `internal/core/ports.go` delta is exactly the four M08-owned shape
  completions (WorkflowStatus, HistoryQuery, ExecutionRecord, StepStatistics), field-for-field
  per IMPLEMENTATION_SPEC; no frozen surface (EventLog format, port interfaces, F-1 alias
  targets) touched. Compile-time assertions pin `*Runtime` to `core.WorkflowRunner`/`core.RecallAPI`.
- **F-2/F-4 honored:** TriggerAPI → `engine.Ingest` only (no new storage paths); WorkflowRegistered
  audit via type-asserted `AppendAudit` with graceful degradation.
- **FR-SDK-09 boundary proven:** examples/ import grep empty (V1 + re-checked at D-CLOSE).
- **`sdk.SQLiteStorage` citation verified** against Blueprint §12 (L934 region, verbatim
  `awis.SQLiteStorage("./oip.db")`) — canonical, recorded as spec-completion not drift.
- Residual quirks recorded in HANDOFF known limitations (Submit latest-version map iteration;
  Inputs/Outputs aliasing; StepStats 10y window). None blocks merge.
