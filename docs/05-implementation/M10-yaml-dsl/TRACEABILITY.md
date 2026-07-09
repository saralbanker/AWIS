# M10 — Traceability
Every task → canonical coordinate. Cards: docs/05-implementation/M10-yaml-dsl/cards/.

| Row | Task (card) | Canonical coordinate |
|---|---|---|
| T1 | `internal/dsl` YAML parser → frozen WorkflowDefinition (C1) | IMP §27.M10 Obj/Out; TDS-02 (field names verbatim); Blueprint §7 example (parse oracle); FR-WD-01 |
| T2 | `gopkg.in/yaml.v3` dependency | IMP dependency policy L47 (approved list, verbatim) |
| T3 | Parse-error precision (file/line) (C1) | FR-WD-04; PRD §18 invalid-output format |
| T4 | Validate-only path, no runtime (C2) | FR-WD-15; PRD §18 "works without a running runtime"; handler existence deferred per PRD §18 |
| T5 | PRD §18 valid/invalid rendering with file/line/example (C2) | IMP §27.M10 Obj "(PRD §18 formats)"; PRD §18 required-format blocks verbatim |
| T6 | Auto-discovery `./workflows/*.yaml` (C2) | IMP §27.M10 Obj; PRD §18 "File location … auto-discovered by `awis start`" |
| T7 | FR-WD-02 round-trip equivalence keystone test (C3) | FR-WD-02; IMP §19 L324 verbatim; IMP §27.M10 Val + Risk rows |
| T8 | Harness event-stream equivalence oracle (C3) | M09 HANDOFF "What M10 may assume"; IKB §4/M10 Src "(equivalence oracle)" |
| T9 | Three example workflows + OIP capture/recall YAML fixtures (C3) | IMP §27.M10 Val "(pre-written as fixtures)"; IMP §5 L104-105 paths; Blueprint §7 (capture verbatim); Blueprint §29 (recall shape); IMP §27.M15 "YAML exactly per Blueprint §7/§10" |
| T10 | `docs/DSL.md` with full annotated example (C3) | IMP §27.M10 DoD row |

## Notes / dispositions
- **Grammar/validator reuse:** FR-WD-03/05..08 are satisfied by M05's `internal/expr` +
  `internal/validate`; M10 only surfaces them through the file path and renders their Issues.
  Re-implementation is the milestone risk row and escalation delta 2.
- **FR-WD-09..14** are Builder/engine semantics already in force (M06/M08); M10's fixtures and
  oracle exercise them through the YAML tier but add no new enforcement code.
- **FR-WD-16/17** (workflow list/show) are CLI scope → M14 (per PRD §18 / IMP §27.M14); the
  FR-WD AC for M10 is 01..15 (IMP §27.M10 AC row).
- **`dsl` → sdk-types dependency direction** (IMP §6): dsl produces the F-1-aliased
  `core.WorkflowDefinition`; consumed via registration. dsl imports nothing from cmd/.

## Execution record (appended during B-BUILD/C-VERIFY)
- C1 (a575113, 2026-07-09): internal/dsl package created. ParseFile/Parse (yaml.v3 KnownFields),
  lineMap, rawToCore. go.mod: yaml.v3 v3.0.1 (only new dep). 6/6 tests pass incl. oracle parse
  + validate.Validate zero-issues check. make build/test/lint/race/e1 all green. No deviations.
