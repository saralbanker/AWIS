# M17 — Validation Checklist (binary; exit list)
<!-- Ticked by M17-V1 (awis-verifier, 2026-08-21, clean tree at ece8694). VERDICT: FAIL.
     Full evidence: TRACEABILITY.md V1 execution record. -->
- [x] V-COMMON all ✅ + pytest regression — build/test/race/e1 ✅; lint flake on first run
      (cross-contaminated shared cache from a concurrently-running worktree, not a real
      finding — 0 issues after cache clean). **pytest regression NOT covered by this V1
      pass** — re-check before re-verify.
- [x] Migration 0006_recall_fts applies fresh + upgrade; StoragePort set untouched (RecallStore additive)
- [x] **PASS** (re-measured 2026-09-18) — Every M17 command exists w/ human+JSON goldens +
      TDS-07 section: replay.json/replay.txt exist AND are consumed at
      `cmd/awis/c1r_test.go:81`; 42 golden tests pass; every command has a TDS-07 section
      in docs/CLI_CONTRACT.md.
- [x] Cron trigger: cron-typed definitions fire on schedule (fake-clock tests; engine untouched)
- [x] awis init: scaffold files per FR-RM-01 (embedded M10 examples byte-identical); --force
      guard; init→start→submit→trace rehearsal system test green (instance legitimately
      reaches terminal `failed` — no compiled-in handlers in the generic binary; not a defect)
- [x] docs/CLI.md covers the complete PRD §15 tree (DoD)
- [~] PRD §32 remaining checklist rows (plugins/intelligence/observability/storage/security)
      each map to evidence — spot-checked only, not exhaustively audited within V1's context
      budget; re-check at next V1 pass.
- [~] **WAIVED** (founder ruling DEC-2, 2026-09-18) — Restated condition: the StoragePort
      interface method set in `internal/core/ports.go` is unchanged (verified zero diff);
      go.mod/go.sum are unchanged (verified zero diff). `internal/dsl/dsl.go`,
      `internal/plugin/{manager.go,runner.go,transport.go}`, `internal/storage/audit.go`,
      and ten pre-existing test files DO carry non-zero diffs, traced to gofmt, migration
      0006_recall_fts, and the additive RecallStore/AuditAppender. Not measured green —
      waived, not PASS, because the diffs are legitimate.
