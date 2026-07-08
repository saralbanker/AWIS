# M09 — Validation Checklist (binary; exit list)
Global DoD (IMP §24) + M09 rows (IMP §27.M9, FR-SDK-06/07/08, QG-5). Verifier executes via
cards/M09-V1.md.

- [ ] V-COMMON block all ✅ (`make build test lint race` + `make e1`; clean tree; HEAD recorded)
- [ ] `engine.Config.NewID` exists; nil ⇒ UUIDv4; a fixed source yields the exact provided IDs
- [ ] `sdk.Config` has `Clock` and `NewID`; both flow through `NewRuntime` to the engine
- [ ] `sdk.DeterministicMode()` returns a Config whose clock and ID sequence are stable across runs
- [ ] `NewRuntime` with non-nil `Intelligence` dispatches an intelligence step to that port
- [ ] `NewRuntime` with nil `Intelligence` runs intelligence steps against NullAdapter (no panic)
- [ ] `awistesting.NewHarness(t)` + `h.Run(linear fixture)` completes synchronously (no poll-loop sleep)
- [ ] `h.Signal` resumes a WAIT-step instance; `h.WaitForCompletion` observes terminal state
- [ ] `h.Tick` advances exactly one engine tick; `h.GetOutput` returns a step-written output value
- [ ] `WithMockIntelligence`: OnDraft fixture surfaces in a workflow's step outputs
- [ ] `NewMockIntelligence` exposes exactly OnDraft/OnEmbed/OnClassify (+ constructor); implements core.IntelligencePort
- [ ] FR-SDK-06: harness tests open no network sockets and exec no processes (in-memory SQLite only)
- [ ] QG-5: one Go test runs a complete workflow incl. WAIT/signal delivery, asserted < 1s wall clock
- [ ] IMP §20.M9 checkpoint: OIP-shaped workflow (plugin step stubbed native) passes in < 1s
- [ ] `internal/fixtures` package exists; harness integration suite consumes it
- [ ] §19 integration shapes each have a harness test: linear · fan-out+join · retry-exhaustion→fallback · compensation · cancellation (± compensate) · WAIT/signal · timeout actions
- [ ] Determinism proof: two runs of the same fixture produce identical event-type/step-id sequences
- [ ] No existing test file modified (additive-only; M06/M07 suites untouched and green)
- [ ] No new sdk exported identifier beyond DeterministicMode, Config.Clock, Config.NewID, and the sdk/testing surface named in the spec
- [ ] Every exported sdk/testing identifier has a godoc comment (spot-check 5)
