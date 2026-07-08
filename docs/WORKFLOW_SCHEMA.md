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
| `retry` | RetryPolicy? | optional | attempts; backoff (`immediate \| linear \| exponential`); initial_delay? (Duration, default 1s); max_delay? (Duration, default 30s); retryable_errors? — **full Blueprint §8 shape adopted by G1-amendment ADJ-6, founder-approved 2026-07-03** (CONTRA-8: §6 L287 listed 3 fields, §8 L521–528 specifies 5; additive optional fields, serialization-compatible; see §7 below). |
| `timeout` | Duration? | optional | per-step timeout. |
| `fallback` | string? | optional | step id to run if this step fails/capability unavailable. |
| `compensation` | CompensationRef? | optional | undo action if workflow fails after this step completes. |
| `wait_signal` | WaitConfig? | optional | for `type=signal`: signal name + timeout + timeout_action. |
| `intelligence` | IntelReq? | optional | for `type=intelligence`: capability + model_hint + context_budget + required? (bool, default false) — **`required` added by G1-amendment ADJ-7, founder-approved 2026-07-03** (CONTRA-9: Blueprint §13 YAML and FR-IL-06/07 depend on it; §6 L292 omitted it; additive optional field; see §7 below). |

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

## 7. G1 amendments adopted at M06 entry (ADJ-6, ADJ-7 — APPROVED 2026-07-03)

Both are additive optional-field amendments (ADJ-5 pattern), founder-approved, recorded in the
04-planning gate log and the M06 module TRACEABILITY:

- **ADJ-6 (CONTRA-8) — RetryPolicy full §8 shape.** §6 L287 ("attempts, backoff,
  retryable_errors") conflicts with §8 L521–528's five-field policy. Canonical shape:
  `{attempts: int, backoff: "immediate"|"linear"|"exponential", initial_delay?: Duration,
  max_delay?: Duration, retryable_errors?: string[]}`. Absent delays default to 1s / 30s
  (mirroring §16 CloudRetryPolicy defaults). Definitions serialized under the 3-field shape
  remain valid.
- **ADJ-7 (CONTRA-9) — IntelReq `required`.** `required: bool` (optional, default `false`)
  added to the intelligence block, matching Blueprint §13's YAML verbatim. `false` = route to
  step fallback when no capable provider (FR-IL-06); `true` = fail the step with
  CapabilityUnavailableError (FR-IL-07).
- **CONTRA-10 (disposition by authority order, no format change).** Step `inputs` is a template
  value-map resolved at execution time (Blueprint §7 example L362–364, PRD §18 DSL rule 1,
  FR-WD-05) — §6 L284's "JSON Schema for expected inputs" tag is the single outlier
  coordinate. The serialized type (`map<string, any>`) is unchanged; `outputs` remains
  schema-shaped. Recorded here so the M10 round-trip oracle inherits one reading.

---

## Appendix — WorkflowInstance (§6, lines 315–329, reference)

Included for completeness; the runtime instance mirrors the StateStore `workflow_instances`
table (§9). Fields (VERBATIM §6): `instance_id: UUID`, `definition_id: string`,
`definition_version: SemVer`, `namespace: string`, `status: InstanceStatus`,
`current_steps: string[]`, `variables: map<string, any>`, `started_at: Timestamp`,
`updated_at: Timestamp`, `completed_at: Timestamp?`.
`InstanceStatus = pending | running | waiting | completed | failed | cancelled | compensating | compensated | compensation_failed`
(§6 line 328 lists eight values; **`compensation_failed` added by G1-amendment ADJ-5, founder-approved 2026-07-03** — CONTRA-7 disposition: Finalization B4 and Blueprint §8 name it as a terminal status and the Finalization outranks §6 in the authority order. Recorded in the M03 module TRACEABILITY and the 04-planning gate log.)
