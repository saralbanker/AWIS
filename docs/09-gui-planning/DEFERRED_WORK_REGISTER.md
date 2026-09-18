# Deferred Work Register

Everything that can safely wait until after GUI development begins — with the trigger
condition that actually pulls each item back into scope, not just "later." Nothing here
is cancelled; deferred means parallel-track or delayed-start, not abandoned. Source
classification: `MINIMUM_ENGINE_PROGRAM.md`.

---

## Founder decisions — free to request now, no engineering cost to delay

| Item | Why deferrable | Trigger to actually resolve it |
|---|---|---|
| **D1** (`state_changes` vs `global_seq`) | Gates only G2, which itself gates only Phase 2 | Before Stage 3 of `EXECUTION_SEQUENCE.md` — request in week 1, don't block Stages 0-2 waiting |
| **D2** (retire namespace overload) | Gates only G3, which gates only the per-namespace filter and `/stats/steps`, both post-beta | When Phase 3 (mutation UI) or Phase 5 observability is actually scheduled |
| **D4** (gate live intelligence) | Only edges out of D4 are `→G0` and `→G9`; no G1-G5 card cites it | When G9 (AI generation) actually starts |
| **Phase A / `E-G0-6`** (STATE.md/main-merge) | No engineering card depends on it; blocks a third party branching from `main`, not work on `engine-hardening` | Whenever the founder has ~1 hour; zero urgency, zero engineering cost |

---

## Engine cards — G2 (entire milestone)

| Cards | Why deferrable | Trigger |
|---|---|---|
| `E-G2-1..7` | Correctly gates Phase 2 (live view) only; Phase 1 cites zero G2 cards | D1 decided, and Phase 2 is next on the roadmap (Stage 3 of the execution sequence) |

## Engine cards — G3 (entire milestone)

| Cards | Why deferrable | Trigger |
|---|---|---|
| `E-G3-1..4, 6` | Only `E-G4-12` (`/stats/steps`, itself Phase-5/deferrable) hard-depends on G3; `engine.Submit` stamps namespace from the workflow's own YAML, so submit/signal correctness doesn't need it; a UI "all namespaces" toggle covers Phase 1-2 with zero G3 engineering | A per-namespace filter or `/stats/steps` is actually scheduled |
| `E-G3-5` (migration notes) | Doc-only | Whenever G3 itself lands |

## Engine cards — G4 mutation/introspection subset

| Cards | Why deferrable | Trigger |
|---|---|---|
| `E-G4-8` (submit+signal) | Phase 3 scope | Phase 3 (mutation UI) scheduled |
| `E-G4-9` (cancel+compensate) | Phase 3 scope; also needs an SDK fix (`compensate` currently hardcoded false) | Phase 3 scheduled |
| `E-G4-10` (register+validate) | Feeds G6/Phase 4's save path | G6's backend track (Stage 5) starts |
| `E-G4-11` (SSE `/stream`) | Gates Phase 2 specifically | D1/G2 land (Stage 3) |
| `E-G4-12` (`/stats/steps`) | Phase-5 concern, G3-gated | G3 lands and Phase 5 observability is scheduled |
| `E-G4-13` (`/plugins`) | Feeds G7/Phase 4 | G7 starts |

## Engine cards — G0/G1 non-gating items

| Cards | Why deferrable | Trigger |
|---|---|---|
| `E-G0-2` (D4 CI gate) | See D4 above | G9 starts |
| `E-G0-3` (stale binary) | Hygiene, no dependents | Any idle slot |
| `E-G0-4` (flake fix) | Test hygiene, not GUI-blocking | Any idle slot; do before it erodes trust in newer test suites (R15 neighbor) |
| `E-G0-5` (branch/commit-count housekeeping) | No dependents | Any idle slot |
| `E-G1-1` (golden JSON snapshots) | Real risk reduction, but no G2/G4 card requires it first | Before `G-G5-9` (contract-drift CI) is scheduled |
| `E-G1-2` (`StoragePort` reflective pin) | Protects a surface G2 never touches | Same as above, low urgency |
| `E-G1-4` (fix `start.go` string-match) | No dependents | Any idle slot |
| `E-G1-5` (correct `CLI_CONTRACT.md`) | Doc-only; GUI is never built against this file | Any idle slot |
| `E-G1-6` (CLI JSON snapshots + coverage) | No dependents | Any idle slot |

---

## GUI-track milestones — G7 through G10

| Milestone | Why deferrable | Trigger |
|---|---|---|
| **G6-2..5** (serializer backend, "savable" editor) | The "clickable" editor (G6-6/7/9/11) ships a full week earlier without it; can run in parallel on its own track rather than gating the clickable milestone | Stage 5 of `EXECUTION_SEQUENCE.md`, or immediately if a third track is staffed |
| **G7** (palette/introspection) | Depends on G6; hardcode the five compile-time `StepType`s as a stand-in until then | G6 lands |
| **G8** (observability) | Depends only on G2/G4, but gates no earlier phase; re-scoped to ≈9d (N1 — zero metrics infra exists) | Any time after G2, staffed independently of the editor track |
| **G9** (AI generation) | Depends on G6 and D4 | G6 lands and D4 is decided |
| **G10-1/2/4/5/6/7** (identity, authz, UI, remote bind, tests, security review) | Depends only on G5, fully independent of G6-G9, but adds no value until multi-user is actually a requirement | Multi-user is scheduled — genuinely deferrable indefinitely for a single-operator deployment |

**Exception — do not park this one under G10's timeline:** `G-G10-3` (subprocess
environment scrubbing, R13) is a live, real security exposure — subprocess steps
currently inherit the full parent environment, including `ANTHROPIC_API_KEY` — with zero
dependency on any other card in the program. **Trigger: schedule within the next
available idle slot on any track, independent of when G10 itself starts.** This is the
one item on this register that should not wait for its parent milestone.

---

*Companion documents: `GUI_START_LINE.md`, `MINIMUM_ENGINE_PROGRAM.md`,
`EXECUTION_SEQUENCE.md`, `FINAL_RECOMMENDATION.md`.*
