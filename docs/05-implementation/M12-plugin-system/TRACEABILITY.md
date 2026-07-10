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
