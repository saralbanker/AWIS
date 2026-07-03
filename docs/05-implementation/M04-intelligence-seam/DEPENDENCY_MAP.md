# M04 — Dependency Map

**Upstream (consumes):**
- M01: IntelligencePort + request/response types in `internal/core` (F-1 layering), IntelReq in step.go.
- Nothing from M02/M03 — Track B is storage-independent by design.

**Downstream (blocks):**
- M06 (engine): IntelligenceRunner dispatches through the Dispatcher; FallbackSignal/ErrCapabilityUnavailable
  drive fallback activation; Usage feeds StepCompleted `{adapter, model, tokens_used}` (FR-IL-09).
- M16 (AnthropicAdapter): implements IntelligencePort behind the frozen interface; must pass porttest.

**Handoff contract:** see HANDOFF.md.
