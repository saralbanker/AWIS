# M11 — SubprocessRunner + awis-step
**Status:** Partitioned — materialize at entry · **Effort:** 2d · **Window:** Week 4
**Objective:** TDS-04 (write first); SubprocessRunner (spawn, stdin/stdout JSON exchange, timeout, error mapping); Python `awis-step` library (@step decorator + step.serve()); pytest vs golden protocol files — golden files are the single wire truth for BOTH sides.
**Depends on:** M06 (M08 for e2e test) · **Blocks:** M12
**Primary sources:** IMP §27.M11; Blueprint §25 · **Compilation spec:** IKB §4/M11
**Key ACs:** Python step executes inside a harness workflow; outputs flow to next step (FR-SE-02, FR-SDK-10).
