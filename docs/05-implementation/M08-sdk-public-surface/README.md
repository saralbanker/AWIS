# M08 — SDK Public Surface
**Status:** Partitioned — materialize at entry · **Effort:** 2d · **Window:** Week 3
**Objective:** `NewRuntime`/`Config`, RegisterHandler/RegisterWorkflow (PRD §18 error format), WorkflowBuilder (+semver check at Build), WorkflowRunner (Submit/Signal/Status/Cancel/List), RecallAPI (QueryHistory/StepStats), audit call site WorkflowRegistered [F-4]. **[F-1] sdk imports internal/engine freely; canonical types stay in internal/core with sdk type aliases — surface per FR-SDK-02 byte-for-byte. [F-2] SDK event-intake surface (TriggerAPI / trigger.go) for DomainEvents.**
**Depends on:** M06, M07 · **Blocks:** M09, M10, M14
**Primary sources:** IMP §27.M8; Blueprint §12, §25; Verification F-1/F-2 · **Compilation spec:** IKB §4/M08
**Key ACs:** example app builds against sdk only; internal/* unimportable from it (compile-checked); post-merge surface change control in force (IMP §13).
