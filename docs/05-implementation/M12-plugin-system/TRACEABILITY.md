# M12 — Traceability
Every task below has a card; every AC row maps to a checklist row. Primary: IMP §27.M12.

| T | Task | Source coordinates | Card |
|---|---|---|---|
| T1 | TDS-05 `docs/PLUGIN_PROTOCOL.md` (JSON-RPC 2.0, lifecycle, manifest, resolution rule) | IMP §12 TDS-05 row; Blueprint §11; SPEC pins | C1 |
| T2 | Golden JSON-RPC wire files | IMP §27.M12 Risk row precedent (M11); SPEC §1 | C1 |
| T3 | Manifest parsing (`awis-plugin.yaml` → Manifest; Blueprint §11 YAML verbatim oracle) | FR-PS-02 | C1 |
| T4 | Migration `0005_plugins.sql` (Blueprint §11 SQL verbatim) | IMP §14 (F-4 renumber); Blueprint §11 | C2 |
| T5 | PluginStore additive storage interface + SQLite impl | audit.go precedent; NFR-S-02 boundary | C2 |
| T6 | `PluginRegistered` audit write site | F-4; PRD NFR-S-05 | C2 |
| T7 | Lifecycle FSM manager (CE-pinned states/transitions/counters) | FR-PS-03/04/05; Blueprint §11 lifecycle; Finalization idle-kill | C3 |
| T8 | JSON-RPC 2.0 NDJSON transport (long-lived, id-correlated) | FR-PS-01; Blueprint §11 protocol | C3 |
| T9 | PluginRunner + capability resolution rule + sdk wiring | FR-PS-06; frozen §7 fixture (`handler: git-context-plugin`) | C3 |
| T10 | §20 checkpoint: kill mid-call → restart → retry OK; idle-kill → respawn | IMP §20.M12 | C3 |
| T11 | `awis-plugin` Python lib (@plugin.capability, serve) | FR-PS-14; Blueprint §11 | C4 |
| T12 | `awis_plugin.testing.mock_request` offline harness | FR-PS-15 | C4 |
| T13 | pytest vs goldens + Go e2e with real Python plugin | IMP §27.M12 Merge (CI green) | C4 |
| T14 | `docs/PLUGIN_GUIDE.md` developer guide | IMP §27.M12 DoD | C4 |

## Notes / dispositions (CE, A-INIT 2026-07-10)
- **`execute` params carry explicit `capability`** — Blueprint §11's example omits it
  (illustrative); TDS-05 is the designated normative home (IMP §12) and requires it. Not a
  CONTRA.
- **Plugin-name handler resolution** — frozen §7 fixture uses `handler: git-context-plugin`
  with two declared capabilities; runtime resolves by unique input-key-set match, else typed
  `plugin_capability_ambiguous`. Capability-id handlers resolve directly.
- **Migration renumber**: IMP §14 said `0003_plugins.sql`; F-4 landed signals/audit at
  0003/0004 (M07). Plugins registry = `0005_plugins.sql`; recall FTS (M17) becomes 0006.
  Forward-only numbering by arrival order — cross-reference-index already records the F-4
  renumber.
- **`plugin_id` = manifest `name`** (V1: one installed version per name).
- **DB status `suspended`** (Blueprint SQL comment) unused in V1; in-memory IDLE keeps DB
  status `active`.
- **Per-plugin call serialization** (mutex) is a V1 pin; cross-plugin concurrency unaffected.
- **Timeout kill does not increment the crash counter** (controlled kill); handshake failures
  do.

## Execution record (appended during B-BUILD/C-VERIFY)
- C1 (5e6eebb, 2026-07-10): TDS-05 authored (all CE pins verbatim); 7 goldens; manifest.go
  (KnownFields, 5 validation rules, file+line errors); 14/14 tests incl. Blueprint §11
  manifest verbatim oracle. make build/test/lint/race/e1 green; go.mod empty. No deviations.
- C2 (58902ff, 2026-07-10): 0005_plugins.sql (Blueprint §11 SQL verbatim); PluginStore additive
  interface + SQLite impl (register upsert + capability replace + PluginRegistered audit in one
  tx, F-4); 11 tests incl. v4→v5 upgrade path. Deviation (CE-ENDORSED): schema-head version
  constants in db_test.go/signal_test.go advanced 4→5 — mechanically required by any new
  migration, same as prior 2→4 advances; no test logic weakened. All gates green.
- C3 (72496b8, 2026-07-10): transport.go (NDJSON JSON-RPC, id correlation, pgroup kill, 4KiB
  stderr tail); manager.go (pinned FSM verbatim; two-lock model callMu/mu, mu never held over
  blocking I/O; intentional-kill flag separates controlled kills from crashes); runner.go
  (resolution rule + typed errors); sdk wiring (nil-safe noPluginStoreRunner + Stop shutdown).
  38 tests in 1.4s incl. §20.M12 checkpoint (SIGKILL mid-call → restart → retry OK) with real
  engine harness. Race clean. No deviations.
- C4 (c9566c6, 2026-07-10): awis_plugin lib (@plugin.capability/serve NDJSON loop/testing.
  mock_request sharing serve's _dispatch); 16 pytest vs shared goldens; Go e2e real Python
  plugin through harness workflow; docs/PLUGIN_GUIDE.md; Makefile pytest runs both suites.
  All gates + pytest green. Minor deviations (manifest lookup via entry-module dir; two pytest
  invocations for rootdir isolation) — implementation details, no protocol deviation.
- V1 (awis-verifier, 2026-07-10, at 147920c): PASS 27/27, zero failures. §20.M12 checkpoint
  3× consecutive clean; NFR-S-02 env-isolation proven; SQL/YAML verbatim rows byte-checked.
- D-CLOSE (Fable, 2026-07-10): adversarial FSM review PASS — Call four-phase lock discipline
  (callMu whole-call; mu never over execute I/O; handshake I/O under mu is deadlock-free since
  child EOF unblocks doHandshake's own 5s deadline); monitor crash-count races resolved by
  handle-identity checks; controlled kills fenced by intentional flag; shutdownOnce + double
  monitorDone wait satisfies invariant 5. Benign edge recorded in HANDOFF. Frozen surfaces:
  StoragePort set untouched (PluginStore additive), engine untouched, sdk wiring-only.
  Gates re-run at HEAD: build/test/lint(0)/race/e1/pytest(8+16) green.
  Merge recommendation: APPROVED FOR SQUASH MERGE.
