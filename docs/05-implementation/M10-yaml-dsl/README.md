# M10 — YAML DSL
**Status:** Partitioned — materialize at entry · **Effort:** 2d · **Window:** Week 3 (parallel)
**Objective:** YAML parser → WorkflowDefinition; auto-discovery of ./workflows/*.yaml; validation error rendering with file/line/example (PRD §18 formats).
**Depends on:** M05, M08 (O-1: may start after M01+M05 if float is needed) · **Blocks:** M14, M15
**Primary sources:** IMP §27.M10; Blueprint §7; TDS-02/03 · **Compilation spec:** IKB §4/M10
**Key ACs:** round-trip equivalence (YAML vs Builder → deep-equal structs, FR-WD-02); three example workflows + OIP capture/recall YAML fixtures parse and validate; validate-only path works without a runtime (FR-WD-15).
