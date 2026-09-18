# Minimum Engine Program

**Question:** what is the minimum engine work required before GUI development becomes
the primary engineering effort?

**Answer: ≈2.5-3 engineering-days of code (8 cards) plus zero mandatory decisions** (D3
has a safe default it can proceed against). Everything else the prior planning pass
scoped into G0-G4 (≈13 engineering-days at milestone-level rollup) is real work — it is
not required to *start* GUI-dominant work, only to complete later phases. This document
classifies every G0-G4 card, and lightly touches G7-G10's backend-only cards, against
that bar.

---

## The true minimal blocking set

**Must be code before frontend/product effort can become the dominant activity:**

| Card | Effort |
|---|---|
| `E-G0-1` — commit B-31 remediation | 0.5h |
| `E-G1-3` — `GetWorkflow` typed sentinel | 2h |
| `E-G4-1` — `cmd/awis-server` skeleton | 4h |
| `E-G4-3` — `internal/api` skeleton | 4h |
| `E-G4-4` — read-only workflow routes | 3h |
| `E-G4-5` — read-only instance routes | 4h |
| `E-G4-6` — event-history route | 3h |
| `E-G4-7` — `/healthz` | 0.5h |

**Must be decided, zero engineering cost:** none, strictly. D3 (in-module server) is
soft — `E-G4-1` can and should start against its already-recommended default
(in-module) without waiting for formal ratification.

**Explicitly not required:** D1, D2, D4, Phase A, all of G2, all of G3, and every G4 card
outside the eight above.

---

## Full G0-G4 classification

Legend: **MUST** = genuinely blocks GUI-dominant work. **SHOULD** = real value, not
blocking. **MAY-DEFER** = no forcing function, park indefinitely. **AFTER-GUI-START** =
genuinely parallel-track work — proceeds while GUI work is already the dominant activity,
gates a later phase, not the start line.

### G0 — Freeze close-out

| Card | Verdict | Why |
|---|---|---|
| `E-G0-1` commit B-31 | **MUST** | Blocks `E-G1-3`; trivial (0.5h) |
| `E-G0-2` D4 CI gate | **AFTER-GUI-START** | Only edges out of D4 are `D4→G0` and `D4→G9` (`GUI_DEPENDENCY_MAP.md`); no G1-G5 card cites it. Defer verbatim to G9's start. |
| `E-G0-3` remove stale binary | **SHOULD** | Cheap, no dependents |
| `E-G0-4` fix system-test flake | **SHOULD** | Test hygiene, not GUI-blocking |
| `E-G0-5` housekeeping | **MAY-DEFER** | — |
| `E-G0-6` STATE.md/main-merge | **AFTER-GUI-START** | Process-only; no engineering card depends on it; blocks a third party branching from `main`, not work on `engine-hardening` itself |

### G1 — Contract hardening

| Card | Verdict | Why |
|---|---|---|
| `E-G1-1` golden JSON snapshots | **SHOULD** | Valuable risk reduction; no G2/G4 card requires it to exist first |
| `E-G1-2` `StoragePort` reflective pin | **SHOULD** | Protects a surface G2 never touches |
| `E-G1-3` `GetWorkflow` sentinel | **MUST** | Direct `E-G4-4` dependency |
| `E-G1-4` fix `start.go` string-match | **MAY-DEFER** | No dependents |
| `E-G1-5` correct `CLI_CONTRACT.md` | **MAY-DEFER** | Doc-only; the GUI is not built against this file |
| `E-G1-6` CLI JSON snapshots + coverage | **MAY-DEFER** | No dependents |

**Finding:** the milestone-level arrow `G1 → G2` overstates the real dependency. Every
`E-G2-x` card's listed dependency is D1 or another G2 card — none cites any `E-G1-x`
card. G2 adds a wholly new table and a storage method behind a locally-declared
interface (the same pattern `cancellationStore`/`signalWaitStore`/`PluginStore` already
establish); it never touches `StoragePort`'s method count or the `ExecutionEvent`
shapes G1 pins. **G1 and G2 can run fully in parallel once D1 lands.**

### G2 — Cursor + eventing (gated on D1)

| Cards | Verdict | Why |
|---|---|---|
| `E-G2-1` through `E-G2-7` | **AFTER-GUI-START** | Real work that correctly gates *Phase 2* (the live view) via `E-G4-11`/`G-G5-6/7` — not required to *start* GUI-dominant work, since Phase 1 is read-only and cites zero G2 cards |

### G3 — Namespace resolution (gated on D2)

| Card | Verdict | Why |
|---|---|---|
| `E-G3-1..4` predicate unify, retire overload, `--all-namespaces`, wildcard branch | **AFTER-GUI-START** | Only two G4 cards cite G3: `E-G4-8` (soft note — but `engine.Submit` stamps namespace from the workflow's own YAML per `GUI_MASTER_PLAN.md` §6.4, so submit/signal correctness doesn't actually depend on G3) and `E-G4-12` (`/stats/steps`, a hard gate, but itself Phase-5-scoped and MAY-DEFER). `G-G5-3` already designs for an "all namespaces" toggle instead of a per-namespace filter — ships with zero G3 engineering. |
| `E-G3-5` migration notes | **MAY-DEFER** | Doc-only |
| `E-G3-6` regression test | **AFTER-GUI-START** | Same gate as G3 generally |

### G4 — API server (gated on D3, soft)

| Card | Verdict | Why |
|---|---|---|
| `E-G4-1` server skeleton | **MUST** | — |
| `E-G4-2` PID/lock guard | **SHOULD** | Real safety issue, cheap, but single-process dev use doesn't hit it yet |
| `E-G4-3` API skeleton | **MUST** | — |
| `E-G4-4/5/6` read-only routes | **MUST** | Phase-1 views |
| `E-G4-7` `/healthz` | **MUST** | Direct `G-G5-1` acceptance dependency |
| `E-G4-8` submit+signal | **AFTER-GUI-START** | Phase 3 |
| `E-G4-9` cancel+compensate | **AFTER-GUI-START** | Phase 3; also needs an SDK fix (`sdk/runtime_runner.go:127` hardcodes `compensate=false`) |
| `E-G4-10` register+validate | **AFTER-GUI-START** | Feeds G6/Phase 4 |
| `E-G4-11` SSE `/stream` | **AFTER-GUI-START** | Gates Phase 2 specifically, not Phase-1 start; first real consumer of G2 |
| `E-G4-12` `/stats/steps` | **MAY-DEFER** | Phase-5 concern, gated on G3 |
| `E-G4-13` `/plugins` | **AFTER-GUI-START** | Feeds G7/Phase 4 |

---

## G7-G10 backend cards — light touch

All of G7's accessors, all of G8, G9's backend cards (schema artifact, Issue plumbing, NL
endpoint, D4 CI gate), and G10-1/2/4 (identity, authz, remote bind) are
**AFTER-GUI-START** or **MAY-DEFER** — each is scoped to Phase 4 or 5, strictly
downstream of Phase 1-2, and none is a dependency of any G0-G4 card.

**One exception, flagged separately from its milestone: `G-G10-3` (subprocess
environment scrubbing, R13) is SHOULD-SOON, independently of G10's multi-user scope.**
Subprocess steps currently inherit the full parent environment, including
`ANTHROPIC_API_KEY` — a live, real security exposure with zero dependency on anything
else in the program. It shouldn't wait for G10 just because it's filed under that
milestone; schedule it as its own 1-day ticket whenever convenient, independent of the
GUI program's sequencing.

---

## Collapses evaluated

**G0 + G1 → one pass: partially yes.** `E-G0-1` (0.5h) rolls directly into a G1 PR by the
same engineer, since `E-G1-3` needs it landed first and nothing else needs to happen
between them. G0's other four cards (D4 gate, binary cleanup, flake fix, housekeeping)
are independent one-off tickets with no shared context — file them separately.

**G2 + G3 → one pass: no.** File-disjoint (G2 touches migrations plus `AppendEvent`/
`UpsertInstance`; G3 touches `export.go`/`history.go`/`metrics.go`/`ReadEventRange`) —
no merge-conflict reason to combine, and combining them would dilute review isolation
for the riskier change. G2 is the correctness-critical class — it edits the two
write-transactions of the append-only EventLog and the projection table, the same
surface class the B-31 remediation just closed five silent-defaults instances of, and
the class `GUI_MASTER_PLAN.md` §1.4 flagged as warranting extra scrutiny given this
repository's Sonnet/Haiku-only model-allocation policy. G3 is comparatively mechanical.
One engineer may work both back-to-back if capacity allows — but as two separate PRs
with separate review passes, not merged into one.

---

*Companion documents: `GUI_START_LINE.md` (the start-line answer this classification
produces), `EXECUTION_SEQUENCE.md` (ordered implementation path), `DEFERRED_WORK_REGISTER.md`
(the full list of what's parked and its trigger condition), `FINAL_RECOMMENDATION.md`.*
