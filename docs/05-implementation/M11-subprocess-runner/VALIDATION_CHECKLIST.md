# M11 — Validation Checklist (binary; exit list)
Global DoD (IMP §24) + M11 rows (IMP §27.M11; FR-SE-02, FR-SDK-10). Verifier executes via
cards/M11-V1.md.

- [x] V-COMMON block all ✅ (`make build test lint race` + `make e1`; clean tree)
- [x] `make pytest` green (awis-step suite; CI includes it per IMP §27.M11 Merge row)
- [x] `docs/SUBPROCESS_PROTOCOL.md` (TDS-04) exists; envelope fields match SPEC §1 verbatim;
      every golden file is embedded/cited in it
- [x] Golden files exist under `internal/runner/subprocess/testdata/protocol/` and BOTH the Go
      tests and the pytest suite read those exact files (no copies)
- [x] SubprocessRunner implements engine.Runner; no engine interface change
- [x] Error mapping proven by tests: spawn_error, timeout (process killed), protocol_error,
      subprocess_error (nonzero exit), envelope error verbatim (code/message/details)
- [x] Timeout kill reaps the process group; test completes fast (≤ ~2s budget)
- [x] `sdk/runtime.go` diff = the one runners-map entry (no sdk exported-surface change)
- [x] `awis_step` public API matches Blueprint §25 example verbatim (@step, StepContext,
      StepResult, step.serve())
- [x] pytest: request parsing, response emission JSON-equal to goldens, handler_not_found,
      handler_error with traceback details, typed StepError raise, serve() over real pipes
- [x] e2e keystone: Python step executes inside an awistesting harness workflow; its outputs
      flow to the next (native) step (FR-SE-02, FR-SDK-10)
- [x] No new Go dependency; no Python runtime dependency (go.mod diff empty; pyproject deps empty)
- [x] No existing test file modified; M05–M10 suites untouched and green
- [x] internal/core, engine, storage, expr, validate, dsl untouched; no migrations
- [x] Every exported Go identifier in internal/runner/subprocess has a godoc comment (spot-check 5)
