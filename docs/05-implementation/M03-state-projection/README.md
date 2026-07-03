# M03 — State Projection & Registry
**Status:** Partitioned — materialize at entry · **Effort:** 2d · **Window:** Week 1 (Track A)
**Objective:** StateStore (upsert/get/list/claim + optimistic version), WorkflowRegistry (immutable semver registration), StepResultCache (TTL), `rebuild-state` as a library function — the recovery tool deliberately predates the engine.
**Depends on:** M02 · **Blocks:** M06
**Primary sources:** IMP §27.M3; Blueprint §9 (two-layer state), §20 · **Compilation spec:** IKB §4/M03
**Key ACs:** replay 10K-event fixture → byte-identical projection (NFR-R-03); re-registering an existing id+version fails; all 12 StoragePort methods contract-tested.
