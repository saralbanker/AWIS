# M10 — Implementation Spec
**Canonical sources:** IMP §27.M10; Blueprint §7 (WORKFLOW DSL, incl. the capture-decision YAML
example verbatim + DSL Design Rules); TDS-02 `docs/WORKFLOW_SCHEMA.md` (FROZEN field names);
TDS-03 `docs/EXPRESSION_GRAMMARS.md` (condition/template grammars — enforced via M05
`internal/expr` + `internal/validate`, never re-implemented); PRD §18 (output formats +
validation checks, verbatim); PRD FR-WD-01..15; IMP §5 (paths); IMP dependency policy L47
(`gopkg.in/yaml.v3` approved).

## Scope
`internal/dsl`: YAML parser → the SAME `core.WorkflowDefinition` the WorkflowBuilder produces
(F-1: `sdk` types alias `internal/core`); auto-discovery of `./workflows/*.yaml`; validation
error rendering with file/line/example (PRD §18 formats); validate-only path that needs no
running runtime (FR-WD-15 library half — the CLI command itself is M14). Keystone test:
round-trip equivalence, YAML vs Builder → deep-equal structs (FR-WD-02; IMP §19 L324).

## 1. Parser core (M10-C1) — `internal/dsl`
- New approved dependency: `gopkg.in/yaml.v3` (IMP L47). No other new dependencies.
- YAML keys are the TDS-02 serialized field names VERBATIM (snake_case: `schema_version`,
  `initial_step`, `final_steps`, `wait_signal`, `timeout_action`, …). The Blueprint §7
  capture-decision example is the parse oracle: it must parse into a valid definition.
- `dsl.ParseFile(path string) (*core.WorkflowDefinition, error)` and
  `dsl.Parse(r io.Reader, filename string) (*core.WorkflowDefinition, error)`:
  unmarshal → the frozen struct; syntax errors carry file + line (yaml.v3 node line info).
- `schema_version` handling per TDS-02 (int, `1` for this spec); `version` semver checked with
  the same rule the Builder uses at Build() (stdlib regexp — reuse, do not duplicate, the sdk
  builder's check if exported; else mirror its exact pattern with a coordinate comment).
- Parse produces DATA only; it does NOT validate graph semantics — it hands the definition to
  `internal/validate` (M05). dsl never re-implements a validator or a grammar (TDS-03 walls).
- Unknown YAML keys are a parse error (`yaml.v3 KnownFields(true)`) — equivalence with the
  frozen struct set is the milestone risk (IMP §27.M10 Risk row).

## 2. Validate-only path + rendering + discovery (M10-C2)
- `dsl.ValidateFile(path string) (*Report, error)`: Parse → `validate.Workflow(def)`; no
  runtime, no storage, no handler-registry access — handler EXISTENCE checks stay deferred to
  registration (PRD §18 "defers handler existence checks"; FR-WD-15).
- Rendering (PRD §18 formats, verbatim shape):
  - Valid: the `✓ <id> v<version> is valid` block with step/intelligence/plugin/trigger summary.
  - Invalid: the `✗ Validation failed: <file>` block — `Line N:` + message, `Suggestion:` line,
    `Example:` block. Line numbers come from the yaml.v3 node map captured at parse time;
    validate Issues are mapped to the node that raised them.
- `dsl.Discover(dir string) ([]string, error)`: returns `<dir>/workflows/*.yaml` (sorted,
  deterministic) — the auto-discovery half of `awis start` (IMP §27.M10; PRD §18 "file
  location"); starting a runtime is NOT M10 scope.
- Golden-output tests for both rendered formats.

## 3. Equivalence oracle + fixtures + docs (M10-C3)
- **FR-WD-02 keystone test:** define one non-trivial workflow twice — once in YAML, once via
  `sdk.WorkflowBuilder` — assert `reflect.DeepEqual` (or cmp with no options) on the two
  `WorkflowDefinition` structs (IMP §19 L324 "round-trip equivalence test").
- **Harness stream oracle (M09 HANDOFF):** run BOTH definitions on `awistesting` harnesses and
  assert identical event-type/step-id sequences — the M09-guaranteed equivalence oracle.
- **Fixtures (pre-written, parse + validate green):**
  - `examples/workflows/hello-world.yaml`, `with-signal.yaml`, `with-intelligence.yaml`
    (IMP §5 L104-105 — the three example workflows; embedded/emitted by `awis init` at M17).
  - `apps/oip/workflows/capture-decision.yaml` — Blueprint §7 example VERBATIM.
  - `apps/oip/workflows/recall-decision.yaml` — Blueprint §29 "How OIP Lives on AWIS" shape
    (fts-search [native: oip.index.fts] → semantic-rank [native: oip.index.semantic] →
    synthesize-answer [intelligence: synthesize]); M15 will register these files unchanged
    (IMP §27.M15 "YAML exactly per Blueprint §7/§10").
- **DoD docs:** `docs/DSL.md` — DSL guide containing the full annotated example (IMP §27.M10
  DoD row); normative statements cite Blueprint §7 / TDS-02 / TDS-03 coordinates.

## Non-scope (do not implement in M10)
- CLI commands (`awis workflow validate|list|show`, `awis start`) — M14 (FR-WD-16/17 too).
- Subprocess/plugin runners (M11/M12); OIP handlers (M15) — the OIP YAML fixtures only parse
  and validate here; they do not run.
- Template/condition grammar changes — TDS-03 is frozen; `internal/expr` is untouched.
- No new StoragePort methods; no migrations; no changes to frozen shapes; no sdk exported
  surface changes (internal/dsl is internal; sdk untouched).
