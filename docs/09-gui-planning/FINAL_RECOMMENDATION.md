# Final Recommendation

Direct answers only. Evidence and full reasoning: `GUI_START_LINE.md`,
`MINIMUM_ENGINE_PROGRAM.md`, `EXECUTION_SEQUENCE.md`, `DEFERRED_WORK_REGISTER.md`.

---

**What should be worked on next?**

Two things, starting today, in parallel:

1. **Engine track (≈2.5-3 engineering-days):** `E-G0-1` (commit B-31) →
   `E-G1-3` (`GetWorkflow` sentinel) → `E-G4-1`, `E-G4-3`, `E-G4-4`, `E-G4-5`,
   `E-G4-6`, `E-G4-7` (server skeleton + read-only routes + `/healthz`). This is the
   entire mandatory engine gate — not G0 or G1 in full, eight specific cards.
2. **GUI track (starts immediately, no engine dependency):** SPA shell + a fixture built
   from `ExecutionEvent`/`StoragePort`/`GUI_ARCHITECTURE.md` §6 (never from
   `docs/CLI_CONTRACT.md`), definitions/instance views, the read-only canvas, and the
   YAML key-order design decision (`G-G6-1`) — the one G6 item worth front-loading
   because it has zero dependency on anything and removes a hard predecessor from the
   editor's critical path later.

Also: **request D1 and D2 from the founder this week.** They cost nothing to decide now
and gate Stage 3 of the execution sequence (Phase 2 and Phase 3 respectively) — deciding
them early means they're never on the critical path; deciding them late means they might
be.

---

**What should NOT be worked on next?**

- **All of G2 and G3.** Real work, correctly scoped, but neither gates the GUI start
  line — G2 gates Phase 2 specifically, G3 gates a per-namespace filter and
  `/stats/steps` that are both post-beta. Starting them now instead of the eight-card
  minimal gate is the single most common way to make GUI-dominant work start a week
  later than it needs to.
- **G4's mutation and introspection cards** (`E-G4-8/9/10/12/13` — submit, signal,
  cancel, register, stats, plugins). Phase 3+ scope. Building them now serves no near-term
  milestone.
- **G0's housekeeping** (stale binary, flake fix, branch pruning) and the
  **STATE.md/main-merge process action.** Real, but zero engineering card depends on any
  of them. Do them in idle slots, not as prioritized work.
- **Chasing D4 or the live-intelligence CI gate.** Two edges in the whole dependency
  graph reference D4, both in G9. It is not on any path to a usable GUI.
- **The full G6 backend track (`G6-2..5`, the YAML serializer) as a prerequisite to
  starting the editor UI.** The editor is clickable a full week before it needs to be
  savable — don't block frontend editor work on the serializer landing first.

---

**What is the fastest path to a usable GUI?**

**≈4-5 engineering days**, with two tracks staffed from day one: the eight-card engine
gate lands in ≈3 days; the GUI track's fixture-built views are already running by then
and cut over to the real API with no rebuild. This is Phase 1 — the read-only dashboard —
live on real data. It requires zero founder decisions (D1/D2/D4 all unresolved is fine)
and only a soft assumption on D3 (in-module server), which already has a safe default.

---

**What is the fastest path to a beta?**

**≈13-15 engineering-days (≈2.5-3 calendar weeks)**, same two-track staffing, to a
**Phase 1+2, view-only beta** — dashboard plus a truthful live view (the instance-parked/
`waiting` state is correctly shown, closing the program's one Critical-severity finding,
B-a). No mutation UI, no G3, no per-namespace filter beyond an "all namespaces" toggle.
Operators keep using the CLI to submit/signal/cancel during the beta window. This
requires D1 to be decided within roughly the first week (it gates only Stage 3 of the
sequence, not the dashboard milestone that precedes it) — request it immediately, don't
wait to need it.

This is a smaller, faster beta than the prior plan's Phase 1-3 recommendation
(previously ≈4 weeks, ≈25-28 cards): it drops the entire mutation-UI layer and all of G3
by shipping observability-only and deferring "operators drive workflows from the GUI" to
the phase after beta. That is a real product-scope reduction, not free — communicate the
beta explicitly as an observability release, not general availability.

---

*Full card-level detail and evidence trail: `GUI_START_LINE.md`,
`MINIMUM_ENGINE_PROGRAM.md`, `EXECUTION_SEQUENCE.md`, `DEFERRED_WORK_REGISTER.md`, and
the underlying `docs/09-gui-planning/` evidence corpus.*
