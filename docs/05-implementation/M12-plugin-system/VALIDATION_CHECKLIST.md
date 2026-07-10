# M12 — Validation Checklist (binary; exit list)
Global DoD (IMP §24) + M12 rows (IMP §27.M12; FR-PS-01..06,14,15; NFR-S-02; F-4; IMP §20.M12).
Verifier executes via cards/M12-V1.md.

- [ ] V-COMMON block all ✅ (`make build test lint race` + `make e1`; clean tree)
- [ ] `make pytest` green (BOTH python packages: awis-step regression + awis-plugin)
- [ ] `docs/PLUGIN_PROTOCOL.md` (TDS-05) exists; methods/fields match SPEC pins verbatim;
      goldens embedded; lifecycle states are the Blueprint §11 seven verbatim
- [ ] Goldens single location `internal/plugin/testdata/protocol/`; Go tests AND pytest read
      those exact files (no copies)
- [ ] Manifest parser: Blueprint §11 awis-plugin.yaml verbatim parses as testdata oracle;
      KnownFields rejection; semver + uniqueness validation; file/line errors
- [ ] Migration `0005_plugins.sql` = Blueprint §11 registry SQL verbatim (tables `plugins`,
      `plugin_capabilities`); applies cleanly on fresh AND existing DB
- [ ] PluginStore is additive (type-assert; StoragePort 12-method set untouched — diff proof)
- [ ] Registration writes `PluginRegistered` audit row (F-4) — test evidence
- [ ] FSM: crash mid-call → `plugin_crash`; next call auto-respawns; 4th consecutive crash →
      FAILED + `plugin_failed`; counter resets on success — all test-proven
- [ ] Handshake validates name/version/capability-superset; 5s deadline; failure counts as
      crash — test-proven
- [ ] Idle: no calls for idle_timeout → process killed; next call transparent respawn —
      test-proven, suite budget ≤2s per test
- [ ] Timeout: controlled kill, StepError `timeout`, crash counter NOT incremented
- [ ] §20.M12 checkpoint: external SIGKILL mid-call → auto-restart → step retry succeeds in a
      harness workflow (retry attempts ≥2)
- [ ] Capability resolution: capability-id direct; plugin-name unique-input-key-set match;
      ambiguity → `plugin_capability_ambiguous` — test-proven
- [ ] NFR-S-02: spawned env = manifest env + PATH only (test asserts absence of a canary
      parent env var); no fd inheritance beyond std streams
- [ ] Plugins never touch runtime.db: no storage handle crosses the process boundary
      (code-inspection row: transport/manager pass only inputs/outputs)
- [ ] `awis_plugin` API: @plugin.capability + plugin.serve + testing.mock_request
      (FR-PS-14/15); mock_request exercises the same dispatch path as serve
- [ ] e2e: real Python plugin registered and called from a harness workflow; outputs flow
- [ ] `docs/PLUGIN_GUIDE.md` exists, cites TDS-05 (DoD)
- [ ] sdk diff = runtime wiring only (manager construction + runners entry + Stop shutdown);
      no exported sdk surface change
- [ ] No new Go dependency; no Python runtime deps; no existing test modified;
      core/engine/expr/validate/dsl/runner-native/runner-subprocess untouched
- [ ] Every exported identifier in internal/plugin has a godoc comment (spot-check 5)
