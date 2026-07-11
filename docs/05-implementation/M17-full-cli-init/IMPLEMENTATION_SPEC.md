# M17 — Implementation Spec
**Sources:** IMP §27.M17; §16 L303; TDS-07 (extend per command — same contract discipline);
PRD §15/§21/§22, §32 remaining checklists; F-2 (cron trigger); CONTRA-4 (FTS additive);
FR-RM-01 (init scaffolding); IMP §14 (recall FTS migration — renumbered **0006** per the F-4
sequence, disposition recorded in M12 TRACEABILITY); M14 patterns (mux/error renderer/goldens).

## CE pins
- Migration `0006_recall_fts.sql`: FTS5 virtual table over execution_events payloads
  (platform-owned, inside runtime.db; CONTRA-4 additive; StoragePort untouched — additive
  RecallStore interface, audit.go pattern).
- Commands (each: TDS-07 section added + human/JSON goldens + what/where/what-now errors):
  - `history [workflow-id] [--limit]` — completed/failed instance list w/ durations.
  - `logs [--tail N]` — runtime structured-log reader (from <data-dir>/awis.log; start gains
    a file sink if absent — small start.go addition, sanctioned).
  - `metrics` — counts/durations aggregated from events (instances by status, steps/sec
    style basics per PRD §20).
  - `recall <query> [--synthesize]` — FTS over 0006; --synthesize routes through configured
    intelligence (Null ⇒ PRD §21 empty-state message; M16 adapter when key present).
  - `replay <instance-id>` — DRY-RUN: re-walk recorded events, print what WOULD run (PRD
    Should-Have; no state writes).
  - `audit [--limit]` — audit_log reader.
  - `plugin status <name>` (health/process info from registry + manager state where
    available; degraded gracefully) · `plugin remove <name>` (registry delete + audit).
  - `config show|set|validate|edit` — config.yaml in --data-dir (keys: namespace, tick,
    anthropic env passthrough note); `show` masks secrets (NFR-S-01, mask helper);
    `set` writes + `ConfigChanged` audit row (completes the F-4 write-site set);
    `validate` schema-checks; `edit` opens $EDITOR (skip test in CI).
  - `rebuild-state <instance-id>|--all` — storage rebuild projection (M03 path) + report.
  - `export [--out dir]` — runtime.db events+definitions to JSON files (PRD §25 portability).
  - `prune-events --dry-run` — TTL scan report (domain_events 7d TTL per F-2); ONLY dry-run
    in V1 (destructive prune post-V1; PRD Should-Have).
- **Cron trigger (F-2):** `start` gains a cron scanner: definitions with `type: cron`
  triggers (config.schedule, standard 5-field) enqueue trigger intakes on schedule; minimal
  parser — stdlib only, support `*/N`, lists, ranges, exact fields; tests with fake clock.
- **`awis init`** (FR-RM-01): go:embed scaffold — config.yaml, .gitignore, workflows/
  {hello-world,with-signal,with-intelligence}.yaml (the M10 examples EMBEDDED AS-IS),
  handlers/example_handler.go, README_AWIS.md; refuses non-empty target unless --force;
  `init → start → submit → trace` rehearsal system test (QG-1 path, measured loosely here,
  formally at M18).
- docs/CLI.md completed for the full tree (DoD).

## Card split
C1: migration 0006 + RecallStore + read-path commands (history/logs/metrics/recall/replay/
audit/export/prune-events --dry-run) + goldens.
C2: config cmds + ConfigChanged audit + plugin status/remove + rebuild-state + cron trigger.
C3: awis init + embed + rehearsal system test + docs/CLI.md complete + TDS-07 sections.
V1: verify.

## Non-scope
Destructive prune; remote plugin install; TUI; `--synthesize` quality tuning. Frozen
surfaces: StoragePort method set, engine semantics (cron scanner lives in cmd/awis start
loop, NOT the engine), core types — all untouched; sdk untouched.
