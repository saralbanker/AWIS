# M09 — Test Infrastructure
**Status:** Partitioned — materialize at entry · **Effort:** 2d · **Window:** Week 3 (parallel)
**Objective:** `sdk/testing`: WorkflowTestHarness (synchronous run, h.Signal, h.Tick, h.WaitForCompletion, h.GetOutput), DeterministicMode (fixed clock, seeded IDs, NullAdapter, manual tick), NewMockIntelligence (OnDraft/OnEmbed/OnClassify). Harness drives the REAL engine with deterministic sources — never a re-implementation.
**Depends on:** M08 · **Blocks:** M15
**Primary sources:** IMP §27.M9; Blueprint §27 · **Compilation spec:** IKB §4/M09
**Key ACs:** QG-5 — complete workflow incl. WAIT/signal delivery in a Go unit test < 1s, zero external deps; §19 integration suites migrated onto the harness.
