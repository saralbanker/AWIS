# M11 → M12/M15 Handoff
**Status: STAGED — actuals filled at M11 completion.**

## Guaranteed outputs (contract — to be confirmed as actuals)
- `docs/SUBPROCESS_PROTOCOL.md` (TDS-04, finalized per DoD) + golden wire files at
  `internal/runner/subprocess/testdata/protocol/` (single wire truth, both sides tested).
- `internal/runner/subprocess`: engine.Runner for `type: subprocess` steps — spawn (no shell,
  whitespace argv), one-shot stdin/stdout JSON exchange, timeout → process-group kill,
  typed StepError mapping (spawn_error/timeout/protocol_error/subprocess_error/envelope).
- `sdk.NewRuntime` (and therefore the awistesting harness) routes subprocess steps.
- `python/awis-step`: pip-installable-from-path `awis_step` (@step, StepContext, StepResult,
  StepError, step.serve()); zero runtime deps; pytest in CI via `make pytest`.

## What M12 may assume (drafted; confirm at completion)
- TDS-04 authoring pattern (envelope tables + embedded goldens) is the template for TDS-05.
- Subprocess spawn/kill/stderr-capture patterns are reusable; M12's transport is JSON-RPC 2.0
  over long-lived stdin/stdout (a DIFFERENT protocol — do not reuse the one-shot envelope).
- Error-code classes (spawn_error/timeout/protocol_error) are the naming precedent.

## What M15 may assume (drafted; confirm at completion)
- Polyglot steps work end-to-end (FR-SE-02 proven in harness); OIP contingency paths may use
  subprocess-typed steps if a plugin slips (IMP §10 contingency note).

## Known limitations (drafted)
- One process per step execution (one-shot); no process reuse/pooling — that is M12's
  lifecycle domain.
- Python-only client library in V1; shell/TS speak the protocol directly (TDS-04 documents it).
- `awis-step` is repo-local (pip install from path); PyPI is a release-day nicety, not a gate.

## Actuals (filled at completion)
- C1 commit: · C2 commit: · C3 commit:
- V1 verification:
- Deviations:
