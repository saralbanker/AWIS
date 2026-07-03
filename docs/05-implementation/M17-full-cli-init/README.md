# M17 — Full CLI + Init
**Status:** Partitioned — materialize at entry · **Effort:** 2.5d (−F-5 scope) · **Window:** Week-5 float / Week 6 · **Phases:** A, B
**Phase A:** init (FR-RM-01 scaffolding via go:embed: config.yaml, .gitignore, 3 example workflows, handlers/example_handler.go, README_AWIS.md), config show|set|validate|edit (source annotations, [F-4] ConfigChanged audit site).
**Phase B:** history, logs, metrics, recall (+--synthesize, empty-state per PRD §21), replay (dry-run), audit (read path), plugin status|remove ([F-5]: install/list already in M14), rebuild-state, export, prune-events --dry-run; migration 0005 (recall FTS over execution_events — CONTRA-4); **[F-2] schedule/cron trigger (FR-WE-14, Should Have — explicitly deferrable)**.
**Depends on:** M14, M12; soft on M16 (synthesize works against NullAdapter empty-state) · **Blocks:** M18
**Primary sources:** IMP §27.M17, §15; PRD §15/§21/§22; Verification F-2/F-5 · **Compilation spec:** IKB §4/M17
