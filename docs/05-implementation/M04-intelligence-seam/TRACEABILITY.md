# M04 — Traceability

| Task | Source (verbatim coordinate) | Requirement |
|---|---|---|
| T1 shapes | Blueprint §13 (requests frozen at M01; responses undefined → owning-milestone definition per M01 ADJ-4b practice) | FR-IL-09 payload fields via Usage |
| T2 NullAdapter | Blueprint §14 "NullAdapter (test/degraded mode)" block, verbatim behaviors | FR-IL-01, FR-IL-10 |
| T3 router | Blueprint §17 decision tree (verbatim); §13 "Capability Router Logic"; §14 fallback_chain | FR-IL-05/06/07 |
| T4 budget | Blueprint §13 "Context Budget Enforcement" — "rejected, not silently truncated" | FR-IL-08 |
| T5 contract suite | IMP §27.M4 DoD "IntelligencePort contract suite exists"; consumed by M16 | — |

## Decisions (EDR)
- **EDR-008** — context-budget measurement: no tokenizer in V1 and no frozen text prescribes one;
  conservative `ceil(bytes/4)` estimator, enforcement-only. Not a CONTRA: fills an unspecified
  mechanism without touching any frozen format; flagged for G2 review.
- **EDR-009** — adapter traits (locality/cost/quality ranks) live in Go-level Registration metadata,
  not in any frozen format; honors §17's public routing spec while staying internally simple
  (Finalization open item, IMP risk note).

- **required-field deferral** — Blueprint §13 YAML shows `required:` inside the step's
  intelligence block, but M01's IntelReq froze `{capability, model_hint, context_budget}`
  (transcribed from Blueprint §6 L292, which omits required). IntelReq is NOT G1-frozen
  (sdk mutable until M08), so this is an owning-milestone completion, not a conflict:
  Router/Dispatcher take `required bool` as an explicit parameter now; the field lands on
  IntelReq when its consumer (M06 engine wiring TDS-02 step config → dispatch) arrives.

## Deviations
- Named-agent dispatch unavailable mid-session → general-purpose Agent with model pinned +
  identity embedded (standing AEO deviation, recorded each milestone).
- M04-C2 session-limit interruption after router.go+budget.go (both complete, kept);
  continuation card M04-C2r finished dispatcher + tests.
