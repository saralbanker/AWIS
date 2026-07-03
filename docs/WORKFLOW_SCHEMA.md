# Workflow Schema — WorkflowDefinition Serialization

- **Spec ID:** TDS-02
- **Version:** 1.0.0
- **Date:** 2026-07-03
- **Source coordinate:** AWIS Architecture Blueprint §6 (WORKFLOW REPRESENTATION), lines 255–332
- **Status:** **FROZEN — G1 APPROVED 2026-07-03** (founder sign-off; adjudication ADJ-3 adopted, recorded inline and in the M01 module TRACEABILITY)

The WorkflowDefinition is the platform's stable data structure; both the YAML DSL and the
Go SDK produce instances of it, and the runtime operates exclusively on it (§6, line 257).
Field names, types, and required/optional markers below are transcribed VERBATIM from §6.
Where the frozen source is silent on demanded content, a `GAP-Gn` subsection records the
silence — no value is invented.

Type convention (§6): trailing `?` = optional; `T[]` = array of `T`; `map<K,V>` = map.

---

## 1. WorkflowDefinition (§6, lines 260–277)

| Field | Type | Required | Semantics (§6) |
|-------|------|----------|----------------|
| `schema_version` | int | required | Serialization-format version; `1` for this spec (G1 ADJ-3, additive to §6). Distinct from `version` below (the workflow's own semver). |
| `id` | string | required | namespaced, e.g. `"oip.capture-decision"`. |
| `version` | SemVer | required | `"1.0.0"`; immutable once registered. |
| `namespace` | string | required | `"oip"` \| `"neurodashboard"` \| … |
| `name` | string | required | human-readable. |
| `description` | string? | optional | — |
| `triggers` | Trigger[] | required | what starts this workflow. |
| `steps` | Step[] | required | all steps (nodes). |
| `transitions` | Transition[] | required | conditional edges between steps. |
| `initial_step` | string | required | which step runs first. |
| `final_steps` | string[] | required | which steps end the workflow. |
| `compensation` | CompensationPlan? | optional | ordered rollback steps on failure. |
| `timeout` | Duration? | optional | max total workflow duration. |
| `metadata` | map<string, any> | required | application-defined. |

## 2. Step (§6, lines 280–293)

| Field | Type | Required | Semantics (§6) |
|-------|------|----------|----------------|
| `id` | string | required | unique within workflow. |
| `name` | string | required | — |
| `type` | StepType | required | `native \| subprocess \| plugin \| intelligence \| signal`. |
| `handler` | HandlerRef | required | `"handler-name"` (native); `"script.py"` (subprocess); etc. |
| `inputs` | InputSchema | required | JSON Schema for expected inputs. |
| `outputs` | OutputSchema | required | JSON Schema for produced outputs. |
| `retry` | RetryPolicy? | optional | attempts, backoff, retryable_errors. |
| `timeout` | Duration? | optional | per-step timeout. |
| `fallback` | string? | optional | step id to run if this step fails/capability unavailable. |
| `compensation` | CompensationRef? | optional | undo action if workflow fails after this step completes. |
| `wait_signal` | WaitConfig? | optional | for `type=signal`: signal name + timeout + timeout_action. |
| `intelligence` | IntelReq? | optional | for `type=intelligence`: capability + model_hint + context_budget. |

## 3. Transition (§6, lines 297–303)

| Field | Type | Required | Semantics (§6) |
|-------|------|----------|----------------|
| `from` | string | required | step id. |
| `to` | string | required | step id. |
| `condition` | Condition? | optional | expression over step outputs; absent = unconditional. |
| `on_error` | bool? | optional | this transition fires on step failure (not success). |

## 4. Trigger (§6, lines 306–310)

| Field | Type | Required | Semantics (§6) |
|-------|------|----------|----------------|
| `type` | TriggerType | required | `manual \| schedule \| event \| webhook`. |
| `config` | map<string, any> | required | type-specific configuration. |

---

## 5. Semver & Immutability Rules (§6)

- `version` is a **SemVer** and is **immutable once registered** (§6, line 262:
  `version: SemVer  // "1.0.0"; immutable once registered`).
- A registered `(name, version)` pair is immutable; any change to a WorkflowDefinition
  requires a **new version**. (Derived from §6, line 262; IMP §27.M1 Deliverable 2.)
- The runtime operates only on WorkflowDefinition and has no knowledge of how the
  definition was produced (§6, line 257), so version identity is the sole compatibility
  handle.

---

## 6. `schema_version` on the serialized format (G1 ADJ-3 — APPROVED 2026-07-03)

Blueprint §6 (lines 260–277) does not enumerate a format-level `schema_version`; the Gate
G1 checklist (IMP §23 item 1) demands one. Resolved at G1 as an **additive completion**
(CONTRA-4 class), founder-approved:

- Top-level field `schema_version: int`, required, value `1` under this spec (§1 table).
- Semantics: the version of THIS serialization format. **Distinct from** `version: SemVer`
  (line 262), which is the workflow's own semantic version and remains immutable once
  registered.
- The value changes only via a future amendment to this document; registered definitions
  serialized under version N are never rewritten.

---

## Appendix — WorkflowInstance (§6, lines 315–329, reference)

Included for completeness; the runtime instance mirrors the StateStore `workflow_instances`
table (§9). Fields (VERBATIM §6): `instance_id: UUID`, `definition_id: string`,
`definition_version: SemVer`, `namespace: string`, `status: InstanceStatus`,
`current_steps: string[]`, `variables: map<string, any>`, `started_at: Timestamp`,
`updated_at: Timestamp`, `completed_at: Timestamp?`.
`InstanceStatus = pending | running | waiting | completed | failed | cancelled | compensating | compensated` (§6, line 328).
