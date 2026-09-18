# AWIS GUI Dependency Map

Status: proposed. This document exists to answer one question fast: **if X is late or
changes, what does it take down with it?** Everything here is derived from
`GUI_MASTER_PLAN.md`; see that document for the evidence behind each node.

---

## 1. Dependency graph

```mermaid
graph TD
    D1["D1: state_changes vs global_seq\n(recommend: state_changes)"]
    D2["D2: retire the \"default\"\nnamespace overload"]
    D3["D3: GUI backend in-module\n(cmd/awis-server)"]
    D4["D4: gate live-intelligence\nor ship unverified"]

    G0["G0: Freeze close-out\nB-31, stale binary, flaky test"]
    G1["G1: Contract hardening\ngolden snapshots, sentinel errors"]
    G2["G2: Cursor + eventing\nmigration 0007"]
    G3["G3: Namespace resolution"]
    G4["G4: API server + SSE"]
    G5["G5: Read-only GUI"]
    G6["G6: Serializer + editor"]
    G7["G7: Palette + introspection"]
    G8["G8: Observability"]
    G9["G9: AI generation"]
    G10["G10: Multi-user / authz"]

    Ba["B-a: non-evented transitions\n(CRITICAL)"]
    Bb["B-b: no global cursor\n(HIGH)"]
    Bc["B-c: no YAML serializer\n(HIGH)"]
    Bd["B-d: namespace silently\nwrong (HIGH)"]
    Be["B-e: unbounded reads\n(MEDIUM)"]
    Bf["B-f: no authz\n(MEDIUM/CRITICAL later)"]

    P1["Phase 1: Read-only dashboard"]
    P2["Phase 2: Workflow monitoring"]
    P3["Phase 3: Workflow management"]
    P4["Phase 4: Visual editor"]
    P5["Phase 5: Full n8n-class"]

    D1 --> G2
    D2 --> G3
    D3 --> G4
    D4 --> G0
    D4 --> G9

    G0 --> G1
    G1 --> G2
    G2 --> G4
    G3 --> G4
    G4 --> G5
    G5 --> G6
    G5 --> G10
    G6 --> G7
    G6 --> G9
    G4 --> G8

    G2 -.resolves.-> Ba
    G2 -.resolves.-> Bb
    G3 -.resolves.-> Bd
    G6 -.resolves.-> Bc
    G4 -.partially resolves, needs G8.-> Be
    G10 -.resolves.-> Bf

    G4 --> P1
    G5 --> P1
    G2 --> P2
    G5 --> P3
    G6 --> P4
    G7 --> P4
    G8 --> P5
    G9 --> P5
    G10 --> P5
```

---

## 2. Founder decisions and their blast radius

| Decision | If deferred | If decided wrong |
|---|---|---|
| **D1** (state_changes vs global_seq) | G2 cannot start; nothing downstream of G2 (G4, G5, and everything after) can start | Choosing `global_seq` alone ships a live view that still can't show `waiting` (B-a stays open) and requires an append-only-table rebuild to fix later |
| **D2** (retire namespace overload) | G3 cannot start; G4 is blocked (depends on G3); `StepStats`/metrics dashboards stay silently wrong (B-d) | Deferring past G5 means the GUI has already encoded the ambiguous sentinel into saved filters/URLs — becomes a breaking change for users, not just code |
| **D3** (in-module server) | No architectural blocker, but building outside the module first means a later forced migration into `cmd/awis-server` plus promoting 4 row types under time pressure | Choosing "separate service" now means designing and freezing a public API contract before any real GUI consumer exists to inform its shape |
| **D4** (gate live intelligence) | G0 (freeze close-out) cannot close; G9 (AI generation) ships without ever having exercised the real Anthropic API in CI | Shipping "declared-unverified" AI features risks a repeat of the stale-model-ID class of bug — the fake-server tests structurally cannot catch it |

---

## 3. Milestone dependency table

| Milestone | Hard prerequisites | Soft/parallel-safe with | Blocks |
|---|---|---|---|
| G0 | D4 | — | G1, and therefore everything after |
| G1 | G0 | — | G2 |
| G2 | G1, D1 | — | G4, Phase 2 |
| G3 | D2 | can run parallel with G2/G4 | G4 |
| G4 | G2, G3, D3 | — | G5, G8, Phase 1 |
| G5 | G4 | — | G6, G10, Phase 3 |
| G6 | G5, B-c (serializer) | — | G7, G9, Phase 4 |
| G7 | G6 | can run parallel with G8, G9, G10 | Phase 4 (palette) |
| G8 | G4 | can run parallel with G5, G6, G7, G9, G10 | Phase 5 (observability) |
| G9 | G6, D4 | can run parallel with G7, G8, G10 | Phase 5 (AI generation) |
| G10 | G5 | can run parallel with G6, G7, G8, G9 | Phase 5 (multi-user) |

**Reading this table for scheduling:** everything above the G4 row is strictly serial
(each blocks the next) except G3, which can overlap G2. Everything at or below G5
fans out — G8 and G10 in particular do not need to wait for the editor (G6/G7) at
all, and should be staffed in parallel with it once G4/G5 land, not queued behind it.

---

## 4. Blocker resolution map

The founder's original five-item blocker brief turned out to be six items with
different real severities than assumed (`GUI_MASTER_PLAN.md` §2). Mapped to what
resolves each:

| Blocker | Severity | Resolved by | Residual risk if unresolved |
|---|---|---|---|
| B-a: non-evented state transitions | **CRITICAL** | G2 (migration 0007) | Live run list shows parked instances as permanently "running" |
| B-b: no global event cursor | HIGH | G2 (same migration) | No way to build an efficient tailing stream at all |
| B-c: no definition→YAML serializer | HIGH | G6 | Editor can display but never save a workflow |
| B-d: namespace overloading | HIGH | G3 | `StepStats` and similar reads silently return wrong-but-plausible zeros |
| B-e: unbounded read paths | MEDIUM | G4 (partial) + G8 (full) | Fine at V1 volume; a single large instance can return an unbounded result set to render one timeline |
| B-f: no authn/authz | MEDIUM now / CRITICAL later | G10 | Acceptable while loopback-bound and single-user; not shippable to a second user without this |

Note: the founder's brief listed "live-provider validation" as a candidate blocker.
Per the review, it's real but does not block *building* the GUI — it blocks *shipping
AI features with confidence*, which is D4/G9's concern, not an architectural
dependency of anything else in this table.

---

## 5. Reading order for a new contributor

1. `GUI_MASTER_PLAN.md` — why each decision and finding exists, with evidence.
2. This document — what depends on what.
3. `GUI_ARCHITECTURE.md` — what to build.
4. `GUI_ROADMAP.md` — in what order, and what can run in parallel.
5. `GUI_PRD.md` — what each phase looks like to a user.

---

*Companion to `GUI_MASTER_PLAN.md`, `GUI_ARCHITECTURE.md`, `GUI_ROADMAP.md`, and
`GUI_PRD.md`.*
