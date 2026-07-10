# M10 → M14/M15/M17 Handoff
**Status: STAGED — activates at M09 merge; actuals filled at M10 completion.**

## Guaranteed outputs (contract — to be confirmed as actuals)
- `internal/dsl`: `ParseFile`/`Parse` (YAML → frozen `core.WorkflowDefinition`, TDS-02 field
  names), `ValidateFile` (+ rendered PRD §18 report), `Discover` (./workflows/*.yaml).
- PRD §18 valid/invalid rendering (file/line/suggestion/example) as reusable output, ready for
  M14's `awis workflow validate` and registration error paths.
- FR-WD-02 proven: YAML tier and Builder tier produce deep-equal structs; harness event-stream
  oracle green.
- Fixtures on disk, parsing and validating: `examples/workflows/{hello-world,with-signal,
  with-intelligence}.yaml`; `apps/oip/workflows/{capture-decision,recall-decision}.yaml`.
- `docs/DSL.md` with the full annotated example.
- Only new dependency: `gopkg.in/yaml.v3`.

## What M14 may assume (drafted; confirm at completion)
- `awis start` auto-discovery = `dsl.Discover` + `dsl.ParseFile` + registration; `awis workflow
  validate <file>` = `dsl.ValidateFile` + the rendered report (FR-WD-15 wiring only).

## What M15 may assume (drafted; confirm at completion)
- The two OIP workflow YAMLs are registration-ready as written; M15 registers the files
  unchanged and supplies the handlers/plugin they name.

## What M17 may assume (drafted; confirm at completion)
- The three examples/workflows YAMLs are `awis init` scaffolding content (FR-RM-01), embeddable
  via go:embed as-is.

## Known limitations (drafted)
- Handler-existence checks are NOT in ValidateFile (deferred to registration per PRD §18).
- Rendering is a library (strings); CLI exit codes, --json, and command surface are M14.
- OIP YAML fixtures parse/validate only; they cannot run until M13 (plugin) + M15 (handlers).

## Actuals (filled at completion)
- C1 commit: a575113 · C2 commit: b958c79 · C3 commit: —
- V1 verification: —
- Deviations: —
