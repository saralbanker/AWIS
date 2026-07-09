# M10 — Validation Checklist (binary; exit list)
Global DoD (IMP §24) + M10 rows (IMP §27.M10, FR-WD-01..15). Verifier executes via
cards/M10-V1.md.

- [ ] V-COMMON block all ✅ (`make build test lint race` + `make e1`; clean tree)
- [ ] Only new dependency is `gopkg.in/yaml.v3` (go.mod diff; IMP L47 policy)
- [ ] `dsl.ParseFile` parses the Blueprint §7 capture-decision example into a valid definition
- [ ] YAML keys are TDS-02 serialized field names verbatim; unknown keys are a parse error
- [ ] Parse errors carry file + line (FR-WD-04 precision)
- [ ] `dsl.ValidateFile` runs with no runtime, no storage, no handler registry (FR-WD-15);
      handler-existence checks are absent (deferred to registration per PRD §18)
- [ ] Invalid rendering matches the PRD §18 required format: `✗ Validation failed:` + `Line N:`
      + `Suggestion:` + `Example:` (golden-output test)
- [ ] Valid rendering matches the PRD §18 `✓ … is valid` summary format (golden-output test)
- [ ] All PRD §18 validation checks surface through ValidateFile (orphans, cycles, fallback
      refs, transition refs, condition/filter grammar, initial_step, final_steps,
      intelligence context_budget/model_hint) — via internal/validate, not re-implemented
- [ ] `dsl.Discover` returns `./workflows/*.yaml` deterministically (sorted)
- [ ] FR-WD-02 keystone: YAML-defined vs Builder-defined identical workflow → deep-equal
      `WorkflowDefinition` structs
- [ ] Harness oracle: both definitions produce identical event-type/step-id sequences on
      `awistesting` harness runs
- [ ] Fixtures parse AND validate: `examples/workflows/hello-world.yaml`, `with-signal.yaml`,
      `with-intelligence.yaml`, `apps/oip/workflows/capture-decision.yaml` (Blueprint §7
      verbatim), `apps/oip/workflows/recall-decision.yaml` (Blueprint §29 shape)
- [ ] `docs/DSL.md` exists with the full annotated example (IMP §27.M10 DoD)
- [ ] No existing test file modified (additive-only; M05–M09 suites untouched and green)
- [ ] No sdk exported-surface change; internal/expr and internal/validate untouched;
      internal/core untouched; no migrations
- [ ] Every exported internal/dsl identifier has a godoc comment (spot-check 5)
