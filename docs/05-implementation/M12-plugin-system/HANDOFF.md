# M12 → M13/M14/M15 Handoff
**Status: COMPLETE — D-CLOSE (incl. Fable adversarial FSM review) passed 2026-07-10.**

## Guaranteed outputs (contract — to be confirmed as actuals)
- `docs/PLUGIN_PROTOCOL.md` (TDS-05) + goldens at `internal/plugin/testdata/protocol/`.
- `internal/plugin`: manifest parsing, registry-backed lifecycle Manager (CE-pinned FSM:
  lazy spawn, handshake validation, ≤3 consecutive-crash restarts, idle kill/respawn,
  serialized calls), NDJSON JSON-RPC transport, PluginRunner wired for `type: plugin`.
- Migration `0005_plugins.sql` (Blueprint §11 SQL verbatim); additive `PluginStore`;
  `PluginRegistered` audit rows (F-4).
- `python/awis-plugin`: `@plugin.capability` + `plugin.serve()` + `awis_plugin.testing.
  mock_request` (offline, no runtime needed — FR-PS-15); pytest in CI.
- `docs/PLUGIN_GUIDE.md` developer guide.

## What M13 may assume (drafted; confirm at completion)
- Implementing a plugin = awis-plugin.yaml + `@plugin.capability` functions + `plugin.serve()`;
  offline tests via mock_request; no AWIS runtime needed for the plugin's own suite.
- Registration for integration tests: `plugin.Manager.Register(manifestPath)`.

## What M14 may assume (drafted; confirm at completion)
- `awis plugin install <path>` ≈ Manager.Register + audit; `awis plugin list` ≈
  PluginStore.ListPlugins (F-5 minimal surface: wiring only).

## What M15 may assume (drafted; confirm at completion)
- `handler: git-context-plugin` (frozen §7 fixture form) resolves via the input-key-set rule;
  `assemble-context` runs once M13's plugin is registered.

## Known limitations (drafted)
- Calls serialized per plugin (V1); no concurrent multiplexing to one process.
- One installed version per plugin name (plugin_id = name).
- CLI surface arrives M14 (F-5 minimal) / M17 (full); no venv management until M13 (PR-5).

## Actuals (filled at completion)
- C1 commit: 5e6eebb · C2 commit: 58902ff · C3 commit: 72496b8 · C4 commit: c9566c6
- V1 verification: PASS 27/27 (awis-verifier, 2026-07-10, at 147920c; §20.M12 checkpoint 3×
  flake-free; idle 0.39s / timeout 0.23s vs 2s budgets).
- Deviations (all CE-endorsed, module TRACEABILITY): schema-head test constants 4→5 (C2,
  mechanically required); manifest lookup via entry-module dir + dual pytest invocation (C4).
- Fable adversarial FSM review (D-CLOSE): invariants 1–5 verified in source — handshake-death,
  shutdown-mid-call, monitor-vs-caller races resolve via handle-identity + intentional-kill
  discipline; -race clean. Benign edge: Shutdown during an in-flight call surfaces as
  plugin_crash to that call (engine is stopping; acceptable V1).
