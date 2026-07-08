# M09 — Traceability
Every task → canonical coordinate. Cards: docs/05-implementation/M09-test-infrastructure/cards/.

| Row | Task (card) | Canonical coordinate |
|---|---|---|
| T1 | `engine.Config.NewID` injectable ID source (C1) | IMP §3 L50 (determinism design constraint); IMP §27.M9 risk row |
| T2 | `sdk.Config` Clock/NewID passthrough (C1) | PRD FR-SDK-07; M08 HANDOFF seam-status note; IMP §13 (additive only) |
| T3 | `cfg.Intelligence` → intelligence Runner wiring in `NewRuntime` (C1) | Blueprint §12 (Config.Intelligence); M08 HANDOFF known-gap note; Blueprint §27 (mock routing) |
| T4 | `sdk.DeterministicMode()` (C1) | PRD FR-SDK-07 verbatim; Blueprint §27 "Deterministic Execution Mode" |
| T5 | `sdk/testing` WorkflowTestHarness (NewHarness/Run/Signal/Tick/WaitForCompletion/GetOutput) (C2) | PRD FR-SDK-06; Blueprint §27 harness example (verbatim usage); IMP §27.M9 Obj; IMP §5 L81 |
| T6 | `NewMockIntelligence` OnDraft/OnEmbed/OnClassify (C2) | PRD FR-SDK-08 verbatim; Blueprint §27 "Intelligence Testing" |
| T7 | `internal/fixtures` shared fixture package (C3) | IMP §19 test-data policy (verbatim) |
| T8 | §19 integration suite on the harness (7 workflow shapes) (C3) | IMP §19 Integration row (verbatim list); IMP §27.M9 DoD |
| T9 | QG-5 acceptance test (< 1s, zero external deps) (C3) | PRD QG-5 L2152 verbatim; IMP §27.M9 Val |
| T10 | IMP §20.M9 checkpoint (OIP-shaped workflow, stubbed plugin step, < 1s) (C3) | IMP §20 M9 row verbatim |

## Notes / dispositions
- **Real-engine mandate:** IMP §27.M9 risk row — "harness drives the *real* engine with
  deterministic sources, never a re-implementation." Enforced as escalation delta 4 and
  checklist row.
- **Surface freeze interaction (IMP §13):** M09's sdk additions (DeterministicMode, two Config
  fields, sdk/testing package) are additive and FR-mandated; recorded here as the written
  justification the freeze requires.
- **FR-IL-10:** `IntelligencePort.Classify` is a declared-but-not-runtime-invoked placeholder
  (core/ports.go note). `OnClassify` fixtures the mock anyway per FR-SDK-08; the harness test
  exercises Draft (runtime-invocable) and asserts Classify fixture via direct port call.
- **§19 migration reading:** "integration suites of §19 migrated onto the harness" (IMP §27.M9
  DoD) = the §19 integration-layer list is implemented ON the harness from M09 onward.
  M06/M07 engine-level tests remain as-is (they are the §19 "engine-level precursors").

## Execution record (appended during B-BUILD/C-VERIFY)
- M09-C1 DONE 2026-07-09 b64723d — T1/T2/T3/T4 all satisfied; make build test lint race e1 green;
  6 test behaviors in sdk/deterministic_test.go; no existing tests modified; no exported identifiers
  changed; only additive sdk exports: DeterministicMode, Config.Clock, Config.NewID.
