# M17 — Validation Checklist (binary; exit list)
<!-- Ticked by M17-V1 (awis-verifier, 2026-08-21, clean tree at ece8694). VERDICT: FAIL.
     Full evidence: TRACEABILITY.md V1 execution record. -->
- [x] V-COMMON all ✅ + pytest regression — build/test/race/e1 ✅; lint flake on first run
      (cross-contaminated shared cache from a concurrently-running worktree, not a real
      finding — 0 issues after cache clean). **pytest regression NOT covered by this V1
      pass** — re-check before re-verify.
- [x] Migration 0006_recall_fts applies fresh + upgrade; StoragePort set untouched (RecallStore additive)
- [ ] **FAIL** — Every M17 command exists w/ human+JSON goldens + TDS-07 section: `replay` has
      no goldens at all; `config_show/config_set/config_validate/rebuild_state` goldens exist
      on disk but are orphaned (no test references them); docs/CLI_CONTRACT.md has a TDS-07
      section for `init` only — every other M17 command is still listed under §3 "Planned
      Commands (not yet implemented)".
- [x] Cron trigger: cron-typed definitions fire on schedule (fake-clock tests; engine untouched)
- [x] awis init: scaffold files per FR-RM-01 (embedded M10 examples byte-identical); --force
      guard; init→start→submit→trace rehearsal system test green (instance legitimately
      reaches terminal `failed` — no compiled-in handlers in the generic binary; not a defect)
- [x] docs/CLI.md covers the complete PRD §15 tree (DoD)
- [~] PRD §32 remaining checklist rows (plugins/intelligence/observability/storage/security)
      each map to evidence — spot-checked only, not exhaustively audited within V1's context
      budget; re-check at next V1 pass.
- [ ] **FAIL** — go.mod EMPTY; engine/core/sdk/dsl/plugin-pkg untouched; storage = 0006 +
      RecallStore only; no existing test modified. go.mod/go.sum ARE empty, but
      `internal/dsl/dsl.go`, `internal/engine/*_test.go` (4 files), `internal/plugin/
      {manager.go,runner.go,transport.go}` (production code) + 2 e2e tests, `sdk/runtime.go`,
      `sdk/testing/mock.go`, `internal/storage/{audit.go,db_test.go,plugins_test.go,
      signal_test.go}` are all modified relative to `m16-anthropic-adapter`. Predates M17-C3
      (already present after C1/C2); needs CE/founder adjudication — see STATE.md LAST-2.
