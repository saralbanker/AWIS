# EDR-009 — Adapter Traits as Registration Metadata

**Status:** Reversible — pre-M08 (internal configuration; not a frozen surface)
**Authored:** M04 (2026-07-03)
**Coordinates:** Blueprint §17 decision tree · §14 fallback_chain · IMP §27.M4 risk note ("router may be internally simple while honoring the public routing spec") · `internal/intelligence/router.go`
**Downstream obligation:** M16 registers AnthropicAdapter with `{Locality: cloud, CostRank, QualityRank}`; M14/M16 own mapping config.yaml routing keys onto Registrations.

---

## Decision

The Blueprint §17 hint branches ("local" → prefer local; "fast" → cheapest
cloud; "quality" → highest-quality cloud) require the router to classify
adapters, but no frozen text defines where that classification lives. It lives
in Go-level registration metadata:

```go
type Registration struct {
    Adapter     core.IntelligencePort
    Locality    Locality // local | cloud
    CostRank    int      // lower = cheaper   (hint "fast")
    QualityRank int      // lower = better    (hint "quality")
}
```

- **Chain is the universe:** only adapters named in `fallback_chain` are ever
  selected, in chain order (Blueprint §14 registers adapters and declares the
  chain together). Registered-but-unchained adapters are never routed to.
- Ties in rank resolve to chain order (stable).
- Unknown hint values behave as no hint (strict chain order).
- `IsAvailable()` is evaluated at decision time, per request.
- NullAdapter always reports unavailable (FR-IL-01) so the router never
  *selects* it; it is the explicit degraded-mode adapter callers use directly.

This honors the public routing spec while keeping the V1 router internally
simple, exactly as the Finalization open item permits. Traits are not part of
any frozen format and may be reshaped freely until the M08 surface freeze.
