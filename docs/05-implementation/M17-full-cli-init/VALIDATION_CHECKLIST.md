# M17 — Validation Checklist (binary; exit list)
<!-- Ticked by M17-V1 (awis-verifier, 2026-08-21, clean tree at ece8694). VERDICT: FAIL.
     Full evidence: TRACEABILITY.md V1 execution record. -->
- [~] **CLOSED WITH RISK ACCEPTED** (founder ruling DEC-5, 2026-09-18) — The rehearsal
      test failed once under `-race -count=3` and has not reproduced in 48 subsequent
      `-race` repetitions (17-rep root-cause sweep, 3-rep build check, then a 25-rep soak:
      20 targeted + 5 whole-package, 393s total). Full-scope `make race` across both
      modules passes. Residual risk tracked as defect D-6 in
      docs/13-beta-baseline/CURRENT_TRUTH_LEDGER.md; D-6 remains OPEN, no root cause. The
      test was instrumented (not fixed) so a recurrence is diagnosable: commit 3f2eeaf on
      branch beta-baseline/d9-test-diagnosability. Not measured green — accepted risk, not
      PASS.
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
- [~] **MEASURED WITH EXCEPTIONS** (founder ruling DEC-4 restated as a measurable
      condition, measured 2026-09-18) — Condition: every acceptance row in AWIS_PRD.md §32
      (lines 2156-2247, 59 rows) is measured and its verdict recorded. Result: 49
      SATISFIED, 6 NOT SATISFIED (rows 8, 15, 37, 38, 40, 53), 1 PARTIAL (row 44), 3
      UNMEASURABLE (rows 5, 28, 32). The 6 failures are tracked as defects D-11 (rows
      38/53, security — plugin cannot be prevented from reading runtime.db), D-12 (rows 8,
      15 — CLI contract divergence), D-13 (rows 37, 44 — missing observability
      schema/metrics), D-14 (row 40 — `--watch` interval), all in
      docs/13-beta-baseline/CURRENT_TRUTH_LEDGER.md. Rows 28 and 32 are UNMEASURABLE
      because no Anthropic credential exists in any verification environment; the adapter
      is implemented and mock-verified, live verification unavailable. Not PASS.
- [~] **WAIVED** (founder ruling DEC-2, 2026-09-18) — Restated condition: the StoragePort
      interface method set in `internal/core/ports.go` is unchanged (verified zero diff);
      go.mod/go.sum are unchanged (verified zero diff). `internal/dsl/dsl.go`,
      `internal/plugin/{manager.go,runner.go,transport.go}`, `internal/storage/audit.go`,
      and ten pre-existing test files DO carry non-zero diffs, traced to gofmt, migration
      0006_recall_fts, and the additive RecallStore/AuditAppender. Not measured green —
      waived, not PASS, because the diffs are legitimate.
