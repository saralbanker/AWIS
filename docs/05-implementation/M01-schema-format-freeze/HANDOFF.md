# M01 → M02/M04/M05 Handoff
**Status: MERGED (`aef89bf`) + G1 APPROVED** — *Guaranteed outputs* are the contract; *Actuals* below (merge commit filled at merge).

## Guaranteed outputs (contract)
- TDS-01/02/03 frozen under G1 approval — downstream code treats them as Tier-0-derived normative text
- `internal/core` canonical types + `sdk` alias surface, compiling, logic-free
- Grammar conformance corpus as compiling Go test data

## What downstream may assume
- **M02:** EventLog envelope + workflow serialization are final; migration 0001 columns derive from TDS-01/02 mechanically.
- **M04:** IntelligencePort type shape is frozen in `internal/core`; adapter work is implementation-only.
- **M05:** the parser target is TDS-03's corpus; grammar questions are CLOSED — a parser that disagrees with the corpus is wrong by definition.
- **M08:** the public surface already exists as aliases; M08 adds behavior (registration, options), never reshapes types.

## Known limitations (by design)
- No storage, no parsing, no engine. Types have no methods. `apps/oip` still empty.

## Actuals (drafted at execution 2026-07-03; merge commit filled at merge)
- Merge commit: `aef89bf` (squash; verified state `d66df49`, M01-V1 PASS 21/21; merge directed by founder 2026-07-03 via M02 execution order)
- G1 sign-off: **APPROVED by founder 2026-07-03** with adjudications ADJ-1..4 adopted (dispositions verbatim in TRACEABILITY.md and the 04-planning gate log); sign-off text recorded in the milestone PR/merge description
- Event types enumerated: **12** (TDS-01 §2; §8 cross-check found no unlisted type)
- CONTRA entries raised during transcription: **CONTRA-5** — WorkflowCancelled payload conflict (Blueprint §9 L671 `{reason}` vs §8 L603 `{reason, cancelled_at}`); G1 disposition: §9 canonical, `cancelled_at` derived from envelope `emitted_at`. Plus three ADJ completions (schema_version ×2, StepError shape) — additive CONTRA-4-class, all G1-approved
- StepError frozen as persisted type `{code, message, details?}` (TDS-01 §2.1) — downstream M02/M06 treat it as format, not API
- 13 placeholder shells (M04: DraftResponse/SynthesisResponse/Capability/Example · M06: Logger/CompensationPlan/CompensationRef · M08: WorkflowStatus/InstanceFilter/HistoryQuery/ExecutionRecord/ReplayTrace/StepStatistics) — shapes owed by those milestones, NOT G1-frozen
- Deviations: (1) M01-C3 first run STOPPED on the 11 undefined type shapes → resolved via G1 ADJ-4, re-run completed; a mid-run session-limit interruption left `scalars.go` orphaned, absorbed by continuation card C3r2. (2) Opus drafting cards ran as general-purpose agents pinned to Opus (named agents register next session). (3) Type-only representation choices flagged by implementer (Payload=json.RawMessage, Duration=string placeholder, RetryPolicy 3 fields per §6 L288) — all pre-M08-mutable, none persisted-format-bearing except Payload raw passthrough which preserves format fidelity
