# M18 — Hardening & v1.0.0 Release  → GATE G4
**Status:** Partitioned — materialize at entry · **Effort:** 3d · **Window:** Week 6
**Objective:** Execute QG-1..5 LITERALLY on clean macOS + Linux; NFR benchmark suite (100K-event fixtures; NFR-P-01..09; CONTRA-2: 10ms target / 50ms gate for native dispatch); one week OIP dogfood overlapping (E1-equivalent metrics recorded, NOT gating — IR-6); fix-only changes; docs/V1_BENCHMARKS.md; tag v1.0.0; goreleaser artifacts (4 platforms).
**Depends on:** M15, M16, M17 · **Blocks:** — (release) · **Gate:** G4
**Primary sources:** IMP §27.M18, §20–21; PRD §31–32 · **Compilation spec:** IKB §4/M18
**Post-tag rule:** migrations become append-only FOREVER (P8); IMP remains archival; V2 planning only on OIP dogfood evidence (Blueprint §33).
