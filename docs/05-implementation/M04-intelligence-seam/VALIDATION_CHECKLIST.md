# M04 — Validation Checklist (all binary; all must pass)
Verified by awis-verifier card M04-V1 on clean clone at HEAD `b8989a9`: **12/12 PASS**, no fix
cycle required. Advisory (non-blocking): "null always last in chain" is runtime-assembly, owed by
M06/M16 (pinned in HANDOFF).

- [x] Clean clone: `make build test lint` exit 0; `go test -race ./internal/intelligence/...` exit 0
- [x] §17 decision tree: every branch has a passing unit test (no-adapter×required, no-adapter×fallback, hint local/fast/quality/nil, call-failure chain walk, exhaustion×required, exhaustion×fallback)
- [x] FR-IL-08: at-budget request dispatches; one-over-budget returns ErrContextBudgetExceeded; no truncation code path exists
- [x] FR-IL-01: NullAdapter IsAvailable()==false; Synthesize=="The record is silent." (verbatim); Embed==zero vector; Classify==first category conf 0.0; Draft deterministic per configuration
- [x] FR-IL-10: Classify implemented but has no runtime caller (grep: no engine/runner references)
- [x] IntelligencePort contract suite exists (porttest) and NullAdapter passes it
- [x] internal/core diff = exactly the four M04-owned shapes + Usage; sdk diff = pure aliases only (zero func)
- [x] No new go.mod dependencies; no HTTP/cloud imports under internal/intelligence
- [x] edr-008 (token estimator) + edr-009 (adapter traits) committed
- [x] Module TRACEABILITY rows complete; global DoD (IMP §24) items 1–7 (macOS CI leg pending remote)
