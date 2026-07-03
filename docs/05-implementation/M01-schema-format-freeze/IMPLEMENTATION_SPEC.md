# M01 — Implementation Specification
**Authority:** IMP §12, §27.M1, §23; Blueprint §6/§9; Finalization Blocker 2; Verification amendment F-1.
Specs are **transcriptions and completions of the frozen documents into normative, versioned spec files** — no TDS introduces new design. Each ≤ 3 pages.

## Deliverable 1 — TDS-01 `docs/EVENTLOG_FORMAT.md`
Source: Blueprint §9. Contents:
- ExecutionEvent envelope (all fields, types, nullability) with `schema_version` semantics.
- A payload schema for **every** Blueprint §9 event type (12+; enumerate exhaustively — IR-1 mitigation demands replay sufficiency per type).
- Sequence rules: monotonic per instance, append-only, no gaps semantics.
- Explicit statement: EventLog is the source of truth; StateStore is a rebuildable projection.

## Deliverable 2 — TDS-02 `docs/WORKFLOW_SCHEMA.md`
Source: Blueprint §6. Contents:
- WorkflowDefinition / Step / Transition / Trigger JSON serialization (canonical field names, required/optional).
- Semver + immutability rules: a registered (name, version) is immutable; changes require a new version.
- `schema_version` on the serialized format itself.

## Deliverable 3 — TDS-03 `docs/EXPRESSION_GRAMMARS.md`
Source: Finalization Blocker 2, **verbatim**. Contents:
- Template grammar `{{dot.path.ref}}`: interpolation only, no operators, missing path → empty string.
- Condition grammar: `==  !=  >  <  >=  <=  &&  ||  !`, parens, single-quoted strings, dot-path refs (hyphens allowed), bracket notation PROHIBITED.
- The prohibited lists, verbatim.
- **Conformance fixture table:** valid/invalid corpus, generated from the grammar text BEFORE any parser exists (IR-2 mitigation). Fixtures land as Go test data files that compile.

## Deliverable 4 — `sdk` public types (F-1 shape)
- `internal/core/` holds the canonical struct definitions: Step, WorkflowDefinition, WorkflowInstance, ExecutionEvent, IntelligencePort, StoragePort (+ Transition, Trigger, enums).
- `sdk/` re-exports via type aliases (`type Step = core.Step`) with full doc comments — the public surface per FR-SDK-02 file layout (awis.go, workflow.go, step.go, trigger.go, runner.go, intelligence.go, recall.go).
- Doc comments only; **zero executable logic** (no methods with bodies beyond trivial accessors mandated by the frozen interface shapes; if in doubt, omit — logic belongs to M02+).

## Scope walls (FORBIDDEN)
- No parser, no storage, no engine code. No migration files.
- No new event types, fields, or grammar features beyond the frozen texts. Gaps → contradiction protocol (document, don't invent).
- No renaming of frozen identifiers "for Go style".

## Gate G1 checklist (from IMP §23 + IR-1)
1. `schema_version` present on every table/format.
2. Every Blueprint §9 event type has a payload schema sufficient for replay.
3. Decade-reader read-through performed (Constitution Art. 12).
4. Human sign-off recorded in the PR.

## AC / DoD / Merge / RB (IMP §27.M1)
**AC:** human sign-off recorded in PR. **DoD:** fixtures compile as Go test data. **Merge:** G1 approved. **RB:** iterate in place — nothing depends on it yet. **Repo after:** specs + types; still no behavior.
