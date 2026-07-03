# M12 — Plugin System + awis-plugin
**Status:** Partitioned — materialize at entry · **Effort:** 3d · **Window:** Week 4
**Objective:** TDS-05 (write first); manifest parsing (awis-plugin.yaml), lifecycle FSM (REGISTER→SPAWN→HANDSHAKE→ACTIVE→IDLE→TERMINATE), JSON-RPC 2.0 over stdin/stdout, crash auto-restart ≤3, idle process-kill + respawn (<2s), migration 0003 (registry), PluginRunner, audit call site PluginRegistered [F-4]; Python `awis-plugin` + testing.mock_request (FR-PS-14/15).
**Depends on:** M11 · **Blocks:** M13
**Primary sources:** IMP §27.M12; Blueprint §11; Finalization (idle-kill recommendation) · **Compilation spec:** IKB §4/M12
**Key ACs:** external kill mid-call → restart → step retry succeeds; minimal env + no fd inheritance (NFR-S-02); plugins never touch runtime.db.
