# M17 — Validation Checklist (binary; exit list)
- [ ] V-COMMON all ✅ + pytest regression
- [ ] Migration 0006_recall_fts applies fresh + upgrade; StoragePort set untouched (RecallStore additive)
- [ ] Every M17 command exists w/ human+JSON goldens + TDS-07 section: history, logs --tail,
      metrics, recall (+--synthesize Null empty-state per PRD §21), replay (dry-run, no writes),
      audit, plugin status, plugin remove (+audit row), config show(masked)|set(+ConfigChanged
      audit)|validate|edit(CI-skip), rebuild-state, export, prune-events --dry-run
- [ ] Cron trigger: cron-typed definitions fire on schedule (fake-clock tests; engine untouched)
- [ ] awis init: scaffold files per FR-RM-01 (embedded M10 examples byte-identical); --force
      guard; init→start→submit→trace rehearsal system test green
- [ ] docs/CLI.md covers the complete PRD §15 tree (DoD)
- [ ] PRD §32 remaining checklist rows (plugins/intelligence/observability/storage/security)
      each map to evidence
- [ ] go.mod EMPTY; engine/core/sdk/dsl/plugin-pkg untouched; storage = 0006 + RecallStore only;
      no existing test modified
