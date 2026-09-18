# M17 — AI Execution Context
**Model:** Sonnet. Cards to `awis-builder` (C1, C2, C3); the goldens/TDS-07 revision card
(C1r) went to `awis-scribe` (mechanical CLI/doc/golden-fixture batch, IKB fit); V1 to
`awis-verifier`. Dispatch order per IMPLEMENTATION_SPEC.md "Card split": C1 → C2 → C3 → V1,
with a revision card (C1r) and a second V1 re-run inserted after the first V1 FAIL.
Branch: `m17-full-cli-init` (stacked on m16-anthropic-adapter).

**Gate:** — (no human gate; README.md).

**STATUS AS OF THIS DOCUMENT (2026-09-18, per STATE.md): M17 is NOT complete.** PHASE is
`C-VERIFY`. Two independent M17-V1 runs have returned FAIL (2026-08-21 at `ece8694`, and a
fresh isolated-worktree re-run at `ef626e7`). Row 8 (frozen-surface scope diff) remains
unresolved on a literal-wording basis pending an actual CE/founder call — no self-trace has
been accepted as adjudication by an independent verifier. Rows 1 and 3 are believed closed
per M17-C2r's own report but were NOT independently confirmed as of the last STATE.md entry.
Do not treat any of this milestone's outputs as verified-complete.

**Key constraints (IMPLEMENTATION_SPEC.md "CE pins"):**
1. Migration `0006_recall_fts.sql` is additive (CONTRA-4); StoragePort method set untouched —
   `RecallStore` is a new interface, following the `audit.go` pattern.
2. Cron trigger scanner (F-2) lives in `cmd/awis` start loop, not the engine — engine
   semantics stay frozen.
3. Each command ships: TDS-07 section + human/JSON goldens + what/where/what-now errors
   (M14 pattern, extended, not redesigned).
4. `awis init`'s embedded workflow examples must be byte-identical to the M10
   `examples/workflows/*.yaml` sources — proven by test, not "similar."
5. No engine/core/sdk changes beyond the additive migration/RecallStore (IMPLEMENTATION_SPEC.md
   "Non-scope"); frozen-surface diffs are exactly what V1's row 8 exists to catch.

**Milestone escalation deltas (from the actual record, STATE.md/TRACEABILITY.md):**
1. C3 died mid-dispatch with no commit on a prior run → handled as a P2 salvage re-dispatch
   against the preserved (non-green, not-merged) WIP branch `m17-c3-wip`, per EEOS §15 D2 —
   not a silent re-try.
2. V1 FAIL (first run) surfaced a frozen-surface diff finding → the same session closed it
   without CE/founder adjudication via commit-by-commit tracing (LAST-3). A second,
   independent V1 re-run explicitly declined to accept that self-trace as adjudication and
   re-flagged the same row FAIL (LAST-6) — this is the standing, unresolved item.
3. `trace --json` JSON-encode-error defect found outside a card's scope → flagged, not fixed
   in-card; left for M17-V1/M18 to route (per TRACEABILITY.md C3 entry).
