# M04 — Validation Checklist (all binary; all must pass)

- [ ] Clean clone: `make build test lint` exit 0; `go test -race ./internal/intelligence/...` exit 0
- [ ] §17 decision tree: every branch has a passing unit test (no-adapter×required, no-adapter×fallback, hint local/fast/quality/nil, call-failure chain walk, exhaustion×required, exhaustion×fallback)
- [ ] FR-IL-08: at-budget request dispatches; one-over-budget returns ErrContextBudgetExceeded; no truncation code path exists
- [ ] FR-IL-01: NullAdapter IsAvailable()==false; Synthesize=="The record is silent." (verbatim); Embed==zero vector; Classify==first category conf 0.0; Draft deterministic per configuration
- [ ] FR-IL-10: Classify implemented but has no runtime caller (grep: no engine/runner references)
- [ ] IntelligencePort contract suite exists (porttest) and NullAdapter passes it
- [ ] internal/core diff = exactly the four M04-owned shapes + Usage; sdk diff = pure aliases only (zero func)
- [ ] No new go.mod dependencies; no HTTP/cloud imports under internal/intelligence
- [ ] edr-008 (token estimator) + edr-009 (adapter traits) committed
- [ ] Module TRACEABILITY rows complete; global DoD (IMP §24) items 1–7 (macOS CI leg pending remote)
