# Execution Sequence

The exact implementation order. Two parallel tracks, staffed from day one — this is the
load-bearing assumption behind every date below. If only one track is staffed, engine
work still finishes first at roughly the same pace as the prior comprehensive plan
(`ENGINE_GUI_TRANSITION_PROGRAM.md`); the acceleration below comes specifically from
running both tracks concurrently, not from any single track getting faster.

Card IDs reference `ENGINE_GUI_WORK_BREAKDOWN.md`. This document does not introduce new
milestones — it reorders and re-scopes the existing ones.

---

## Stage 0 — Day 1, both tracks start simultaneously

**Engine track:**
- `E-G0-1` — commit the B-31 remediation (0.5h)
- `E-G1-3` — `GetWorkflow` typed sentinel (2h)
- Begin `E-G4-1` / `E-G4-3` — server + API skeletons (8h combined, no dependency on the above two, can start in parallel with them)

**GUI track:**
- Author the fixture (see `GUI_START_LINE.md` — built from `ExecutionEvent`, `StoragePort`, and `GUI_ARCHITECTURE.md` §6, never from `CLI_CONTRACT.md`)
- `G-G5-1` — SPA shell & API client, pointed at the fixture (1d)
- `G-G6-1` — YAML key-order ADR (1d, zero dependency on anything, can run any time — starting now removes it from the critical path entirely by the time G6 needs it)

## Stage 1 — Days 2-3

**Engine track:**
- `E-G4-4`, `E-G4-5`, `E-G4-6` — read-only workflow/instance/event routes (10h)
- `E-G4-7` — `/healthz` (0.5h)
- **→ Minimal engine gate complete (`GUI_START_LINE.md` start line 2).**

**GUI track:**
- `G-G5-2` through `G-G5-5` — definitions, instance list/detail, static timeline, against the fixture (4.5d)
- `G-G5-8` — read-only canvas, against a `WorkflowDefinition` fixture (2d, overlaps into Stage 2)

## Stage 2 — Day ~4-5 — Dashboard Live milestone

**Cutover:** point the GUI track's API client at the real `cmd/awis-server` (from Stage 1)
instead of the fixture. No rebuild — the fixture was shaped from the same code the real
API serves.

- `G-G5-9` — contract-drift CI check, now that `E-G1-3`'s sentinel and a real API exist (1d)
- **→ Phase 1 (read-only dashboard) is live on real data.** This is `GUI_START_LINE.md`'s start line 2, reached in calendar time, not just card-availability.

**GUI track continues immediately, still with zero further engine dependency:**
- `G-G6-6` — editable canvas, built on `G5-8`, client-side in-memory graph model, save button stubbed (3d)
- `G-G6-7`, `G-G6-9`, `G-G6-11` — 6-relationship edge editing, expression squiggles, inputs-as-template-values UI, in parallel (max 2d)
- **→ "Clickable editor" milestone reached ≈7 engineering-days after `G5-8` starts, entirely decoupled from the engine track.**

## Stage 3 — Ongoing, engine track only, gated on decisions

These proceed whenever D1/D2 are decided — not before, and not blocking anything above.
Request D1 and D2 from the founder now (decision latency is free); do not wait for
engineering capacity to ask.

- **On D1 landing:** `E-G2-1` through `E-G2-7` (migration 0007, both write-path wires, `ReadChangesSince`, regressions) — ≈2.4d
- **Immediately after G2:** `E-G4-11` — SSE `/stream`, the first real G2 consumer (1d)
- **Then, GUI track:** `G-G5-6`/`G-G5-7` — SSE wiring and honest live-timeline UI (3.5d)
- **→ Phase 2 (truthful live monitoring) complete. Combined with Stage 2: Phase 1+2 beta scope reached.**

- **On D2 landing, fully parallel with the above, does not block it:** `E-G3-1` through `E-G3-6` — namespace unification (≈1.6d). Triggers only when a per-namespace filter or `/stats/steps` is actually needed (Phase 3+/5) — not before.

## Stage 4 — Beta cut point

**Phase 1 + Phase 2, view-only.** No mutation UI, no G3, no G4 mutation/introspection
cards. Operators keep using the CLI to submit/signal/cancel during beta.

- Total from Stage 0: ≈13-15 engineering-days across two parallel tracks (≈2.5-3
  calendar weeks), assuming D1 is decided within the first week (it gates only Stage 3,
  not Stages 0-2, so decision latency here is absorbed rather than blocking).
- **This is the beta.** Ship it labeled as an observability release — do not layer Phase
  3's mutation UI on top before calling it a beta; that was the prior plan's ~4-week
  scope and is no longer the minimum.

## Stage 5 — Editor savable (post-beta, or in parallel if a third track is staffed)

- `E-G4-10` — register+validate route (3h, engine track, whenever picked up)
- `G-G6-2` through `G-G6-5` — YAML emitter, map-field serialization, round-trip suite, save-from-canvas API (≈6.5d backend track, can start any time after `G6-1` — Stage 0 — and run fully parallel with Stages 2-4)
- `G-G6-10` — layout persistence (0.5d, needs both the clickable canvas from Stage 2 and the save API above)
- `G-G6-12` — E2E round-trip acceptance (1d)
- **→ Phase 4 (visual editor, full save path) complete.** Convergence adds ≈1.5 days on top of whichever of the clickable-editor track (Stage 2) or the serializer track (this stage) finishes last — they were never on the same critical path.

## Stage 6 — Remaining milestones, no fixed order beyond their own listed dependencies

Staffed as capacity allows, in parallel with each other and with Stage 5:

| Milestone | Trigger | Notes |
|---|---|---|
| G7 (palette) | after G6 | 3.5d, independent accessors |
| G8 (observability) | after G2 (Stage 3) | ≈9d, re-scoped — see `ENGINE_GUI_WORK_BREAKDOWN.md` |
| G9 (AI generation) | after G6, D4 decided | ≈8d; request D4 whenever convenient, it gates nothing before this |
| G10 (multi-user) | after G5 (Stage 2) only — does not need G6-G9 | ≈12.5d, except `G-G10-3` (subprocess env scrubbing) — pull this forward, see below |

**Pull forward, independent of everything:** `G-G10-3` (1d, subprocess environment
scrubbing, R13 — a live secret-exposure fix with zero dependency on any other card).
Schedule it in any idle slot on the engine track; it does not need to wait for G10's
multi-user scope to start.

**Anytime, no urgency:** `E-G0-2` (D4 CI gate, defer to G9's actual start), `E-G0-3/4/5`
(binary cleanup, flake fix, housekeeping), `E-G0-6` (STATE.md/main-merge — process only,
schedule whenever the founder has an hour; it blocks nothing engineering-side).

---

## Summary timeline (two tracks staffed from day one)

| Milestone reached | Calendar point |
|---|---|
| Fixture-based GUI engineering begins | Day 1 |
| Minimal engine gate lands | Day ~3 |
| **Phase 1 (dashboard) live on real data** | **Day ~4-5** |
| Clickable editor (no save) | Day ~11-12 |
| **Phase 1+2 beta (view-only)** | **~2.5-3 weeks** (pending D1 decided in week 1) |
| Editor savable (Phase 4 complete) | Beta + ~1-2 weeks, if G6's backend track was staffed in parallel starting Stage 0 |
| Full n8n-class (G7-G10 complete) | Beta + several weeks, capacity-dependent, no engine-track gate beyond G2 |

---

*Companion documents: `GUI_START_LINE.md`, `MINIMUM_ENGINE_PROGRAM.md` (the classification
this sequence is built from), `DEFERRED_WORK_REGISTER.md` (everything parked here with
its trigger condition), `FINAL_RECOMMENDATION.md`.*
