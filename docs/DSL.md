# AWIS YAML Workflow DSL Guide

**Canonical coordinates:** Blueprint §7 (DSL tier, examples, design rules);
TDS-02 `docs/WORKFLOW_SCHEMA.md` (frozen field names and types);
TDS-03 `docs/EXPRESSION_GRAMMARS.md` (condition and template grammars);
PRD §18 (validation behavior and output formats).

This guide is a reader's companion, not a normative restatement.
All normative content lives at the coordinates above.

---

## Overview

The YAML DSL is Tier 1 of the two-tier workflow definition surface (Blueprint §7).
It is for declarative, linear, or conditionally-branching workflows.
Both tiers — YAML and Go SDK — produce identical `core.WorkflowDefinition` structs;
the runtime operates exclusively on that struct (Blueprint §6; TDS-02).

**Parse path:** `internal/dsl.ParseFile(path)` → `core.WorkflowDefinition`

**Validate path (runtime-free):** `internal/dsl.ValidateFile(path)` → `*dsl.Report`

**Auto-discovery:** `internal/dsl.Discover(dir)` returns `<dir>/workflows/*.yaml`
sorted and deterministic, used by `awis start` (PRD §18 "file location").

---

## Annotated Example: OIP capture-decision

The following is the Blueprint §7 canonical example (verbatim; see
`apps/oip/workflows/capture-decision.yaml`), annotated with TDS-02 field
coordinates.

```yaml
# No schema_version: absent is accepted; the only supported value is 1 (TDS-02 §1).
id: capture-decision          # TDS-02 §1 — namespaced string, unique within registry
version: 1.0.0                # TDS-02 §5 — semver; immutable once registered
namespace: oip                # TDS-02 §1 — owning namespace
name: Capture Decision        # TDS-02 §1 — human-readable label
description: Assemble context, draft entry, confirm, append to record  # optional

triggers:                     # TDS-02 §4 — zero or more trigger declarations
  - type: manual              # starts the workflow by explicit submission
  - type: event
    config:
      event: git.push.completed
      filter: "event.branch == 'main'"
      # filter is a boolean condition (TDS-03 §2); event scope allowed here

steps:                        # TDS-02 §2 — ordered list of step nodes
  - id: assemble-context      # TDS-02 §2 — unique within the workflow
    name: Assemble Git Context
    type: plugin              # TDS-02 §2 StepType: native | subprocess | plugin | intelligence | signal
    handler: git-context-plugin
    inputs:                   # TDS-02 §2 InputSchema — map; values may contain {{...}} templates (TDS-03 §1)
      repo_path: "{{workflow.inputs.repo_path}}"
      ref: "{{workflow.inputs.ref}}"
    outputs:                  # TDS-02 §2 OutputSchema — map of JSON Schema descriptors
      context: {type: object}
    retry:                    # TDS-02 §7 RetryPolicy — optional
      attempts: 3
      backoff: exponential    # immediate | linear | exponential

  - id: draft-entry
    name: Draft Decision Entry
    type: intelligence        # routes to IntelligencePort
    intelligence:             # TDS-02 §2 IntelReq — required when type=intelligence
      capability: draft       # capability name declared by the provider
      context_budget: 3000    # max tokens for assembled context; must be > 0
    inputs:
      context: "{{steps.assemble-context.outputs.context}}"
    outputs:
      draft: {type: string}
    fallback: manual-entry    # step id to run when capability unavailable (FR-IL-06)

  - id: manual-entry
    name: Manual Entry (AI unavailable)
    type: signal              # parks until a named signal is delivered
    wait_signal:              # TDS-02 §2 WaitConfig — required when type=signal
      name: manual_draft_provided   # signal_name (Blueprint §7 alias; TDS-02 uses signal_name)
      timeout: 24h
      timeout_action: fail    # fail | compensate | continue

  - id: confirm-entry
    name: Human Confirmation
    type: signal
    wait_signal:
      name: entry_confirmed
      timeout: 72h
      timeout_action: fail

  - id: append-to-record
    name: Append to Record
    type: native
    handler: oip.record.append
    inputs:
      entry: "{{steps.confirm-entry.outputs.confirmed_entry}}"
    outputs:
      entry_id: {type: string}

initial_step: assemble-context   # TDS-02 §1 — id of the first step to execute

transitions:                     # TDS-02 §3 — conditional edges between steps
  - from: assemble-context
    to: draft-entry
  - from: draft-entry
    to: confirm-entry
  - from: draft-entry
    to: manual-entry
    condition: "steps.draft-entry.status == 'fallback'"  # TDS-03 §2 boolean expression
  - from: manual-entry
    to: confirm-entry
  - from: confirm-entry
    to: append-to-record

final_steps: [append-to-record]  # TDS-02 §1 — ids of terminal steps

# No compensation plan: append-to-record is the final step.
# Compensation handlers are specified only on steps with downstream successors
# that could fail — never on final steps, whose handlers would never be invoked.
```

---

## Field Reference

All field names, types, and required/optional markers are defined normatively in
**TDS-02 `docs/WORKFLOW_SCHEMA.md`**.  The table below is an index only.

| YAML key | TDS-02 section | Notes |
|---|---|---|
| `schema_version` | §1 | Integer; 1 is the only supported value; absent defaults to 1 |
| `id` | §1 | Namespaced string, e.g. `oip.capture-decision` |
| `version` | §5 | Semver string; immutable once registered |
| `namespace` | §1 | Owning namespace, e.g. `oip` |
| `name` | §1 | Human-readable label |
| `description` | §1 | Optional |
| `triggers` | §4 | List of trigger declarations; `type` + optional `config` |
| `steps` | §2 | List of step nodes; see Step sub-fields below |
| `steps[].id` | §2 | Unique within the workflow |
| `steps[].type` | §2 | `native` / `subprocess` / `plugin` / `intelligence` / `signal` |
| `steps[].handler` | §2 | Required for `native`, `subprocess`, `plugin` types |
| `steps[].inputs` | §2 | InputSchema map; string values may contain `{{...}}` templates |
| `steps[].outputs` | §2 | OutputSchema map |
| `steps[].retry` | §7 | RetryPolicy: `attempts`, `backoff`, `initial_delay`, `max_delay` |
| `steps[].fallback` | §2 | Step id to activate on failure or capability unavailability |
| `steps[].wait_signal` | §2 | WaitConfig: `signal_name` (or `name`), `timeout`, `timeout_action` |
| `steps[].intelligence` | §2 | IntelReq: `capability`, `context_budget`, `model_hint`, `required` |
| `steps[].compensation` | §8 | CompensationRef: `handler` |
| `steps[].timeout` | §2 | Per-step duration string, e.g. `30s`, `5m` |
| `transitions` | §3 | List of edges: `from`, `to`, optional `condition`, optional `on_error` |
| `initial_step` | §1 | Id of the first step |
| `final_steps` | §1 | List of terminal step ids |
| `compensation` | §8 | CompensationPlan: `steps` list of `{step_id, undo_handler, retry}` |
| `timeout` | §1 | Maximum total workflow duration |
| `metadata` | §1 | Application-defined map; absent defaults to `{}` |

---

## Expression Rules

Expression syntax is defined normatively in **TDS-03 `docs/EXPRESSION_GRAMMARS.md`**.
Summary (not normative):

**Templates** (`{{path-ref}}` in string field values, TDS-03 §1):
- Resolve at execution time, not parse time (Blueprint §7 DSL Design Rules).
- `path-ref` is a dot-path: `workflow.inputs.<key>`, `steps.<step-id>.outputs.<key>`,
  or `steps.<step-id>.status`.
- Identifiers may contain hyphens; bracket notation is prohibited.
- Missing paths resolve to empty string with a logged warning.

**Conditions** (in `transition.condition` and `trigger.config.filter`, TDS-03 §2):
- Boolean expressions: `==`, `!=`, `>`, `<`, `>=`, `<=`, `&&`, `||`, `!`, `()`.
- Same dot-path notation as templates.
- String literals use single quotes; numeric and boolean literals supported.
- `null` literal for null checks only.
- `event` scope (`event.<field>`) is valid **only** in trigger filter fields,
  not in transition conditions (PRD §18 validation check).
- Arithmetic, function calls, and bracket notation are prohibited.
- Conditions are validated at workflow registration time; invalid syntax fails registration.

---

## Validation Behavior

Validation behavior and output formats are defined normatively in **PRD §18**.
Summary (not normative):

`dsl.ValidateFile(path)` performs these checks (PRD §18 "Validation checks"):
1. Duplicate step ids
2. `initial_step` is a defined step
3. Every `final_steps` entry is a defined step
4. Every `transition.from` / `transition.to` is a defined step
5. Every non-empty `step.fallback` is a defined step
6. No unreachable steps (orphans), over transition + fallback edges
7. No cycles over transition edges
8. Condition expressions parse; no `event` scope in conditions
9. Trigger filter expressions parse
10. `type=signal` steps have `wait_signal` config
11. `type=intelligence` steps have `intelligence` config
12. `native`, `subprocess`, `plugin` steps have a non-empty `handler`
13. `intelligence.context_budget` is a positive integer
14. `intelligence.model_hint` is `fast`, `quality`, `local`, or absent

Handler existence is **not** checked by `ValidateFile` — that check is deferred
to registration time (PRD §18 "defers handler existence checks"; FR-WD-15).

Valid output format (PRD §18 verbatim):

```
✓ <id> v<version> is valid
  steps: N  intelligence: N  plugins: N  triggers: N
```

Invalid output format (PRD §18 verbatim):

```
✗ Validation failed: <file>
  Line N: <message>
  Suggestion: <hint>
  Example: ...
```

---

## Additional Examples

- `examples/workflows/hello-world.yaml` — minimal two-step linear native workflow
- `examples/workflows/with-signal.yaml` — prepare → WAIT/signal → finalize
- `examples/workflows/with-intelligence.yaml` — AI draft with manual fallback
- `apps/oip/workflows/capture-decision.yaml` — full OIP example (Blueprint §7 verbatim)
- `apps/oip/workflows/recall-decision.yaml` — fts-search → semantic-rank → synthesize (Blueprint §29)

All five parse and validate with zero issues via `dsl.ValidateFile`.

---

## Design Rules

See Blueprint §7 "DSL Design Rules" for the canonical list.  Key points:

- Expression templates (`{{...}}`) resolve at execution time, not parse time.
- Both tiers (YAML and Go SDK) pass the same `WorkflowValidator` before registration.
- Parallel execution uses fan-out transitions: multiple `Transition` entries with
  the same `from` step activate all target steps concurrently.
- There is no `parallel` StepType; fan-out transitions are sufficient for V1.
- The YAML tier is a subset of SDK expressiveness; runtime representation is identical.
- Unknown YAML keys are a parse error (strict mode; no silent field loss).
