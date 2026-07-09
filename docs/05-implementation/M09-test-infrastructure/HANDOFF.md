# M09 → M10/M11/M12/M15/M18 Handoff
**Status: COMPLETE — actuals confirmed at D-CLOSE (Fable, 2026-07-09); merge sha fills at merge.**

## Guaranteed outputs (confirmed as actuals at D-CLOSE)
- `sdk/testing` (package `awistesting`): `NewHarness`, `Run`, `Signal`, `Tick`,
  `WaitForCompletion`, `GetOutput` + options (`WithMockIntelligence`, `WithStepHandler`,
  `WithClock`, `WithIDSource`).
- `sdk.DeterministicMode()` — fixed clock, seeded IDs, manual tick (FR-SDK-07).
- `NewMockIntelligence()` with `OnDraft`/`OnEmbed`/`OnClassify` (FR-SDK-08).
- `engine.Config.NewID` + `sdk.Config.{Clock,NewID}` determinism seams (nil-safe, additive).
- `NewRuntime` wires `cfg.Intelligence` (nil ⇒ NullAdapter) — the M08 gap is closed.
- `internal/fixtures` shared deterministic workflow definitions (IMP §19 test-data policy).
- §19 integration suite running on the harness; CI-blocking from M09.
- QG-5 satisfied and executable as a single test (re-run literally at G4).

## What M15 may assume (confirmed at D-CLOSE)
- OIP workflows can be developed test-first on the harness with zero external services.
- The §20.M9 checkpoint test is the template for OIP-shaped harness tests.

## What M10/M11/M12 may assume (confirmed at D-CLOSE)
- M10: harness executes builder-produced definitions; YAML↔builder equivalence oracle can
  assert identical harness event streams.
- M11/M12: new runner kinds slot into the harness via the engine runners map; harness itself
  needs no changes for new step types.

## Known limitations (confirmed at D-CLOSE)
- Harness targets in-memory SQLite only (matches M08 storage limitation).
- MockIntelligence returns fixed fixtures (no scripted sequences/latency injection — V2 if needed).
- `Classify` is fixture-complete but not runtime-invoked (FR-IL-10 placeholder).

## Actuals (filled at completion)
- C1 commit: b64723d · C2 commit: 3764a3a · C3 commit: c8fc10b
- V1 verification: PASS 20/20 (awis-verifier, 2026-07-09); V1 fix commit sha: 7f700a8
- Deviations (V1 fix): `sdk/events.go` deleted (was an unspec'd Runtime.ReadEvents export);
  replaced with `Harness.storage core.StoragePort` field in harness.go so `Harness.ReadEvents`
  reads storage directly — no new sdk package exports. All 20 checklist rows ✅.
- D-CLOSE (Fable, 2026-07-09): semantic review of full diff (16 files, +1929/−34) found NO
  architectural drift; internal/core diff empty; FR-SDK-09 examples/ import grep empty;
  `make build test lint race` + `make e1` re-run green at HEAD 9da8dd0. Merge sha: fill at merge.
