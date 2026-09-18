# Engine → GUI Decision Record

Three sequencing paths for the same work, plus the standing founder decisions (D1-D4)
and the one process decision (main-merge / `STATE.md`) that gate them. Evidence for every
claim below is in `GUI_MASTER_PLAN.md` §12-13 and `GUI_DEPENDENCY_MAP.md` §2; this
document adds the tradeoff framing and reconciles it against the card-level breakdown in
`ENGINE_GUI_WORK_BREAKDOWN.md`.

---

## 1. Founder decisions this record assumes get made, and how

These four are unchanged asks from `GUI_MASTER_PLAN.md` §12 — re-listed here because
every path below is a function of when they get decided, not just what gets decided.

| ID | Decision | Recommendation | Blast radius if deferred |
|---|---|---|---|
| **D1** | `state_changes` table vs. `global_seq` column | **`state_changes`.** Solves the cursor *and* the three non-evented transitions; never touches the frozen append-only table; `VACUUM`-proof; in-repo precedent (`audit_log`). | G2 cannot start; nothing downstream of G2 (G4, G5, everything after) can start. |
| **D2** | Retire the `"default"` namespace overload | **Retire it.** `""`/`--all-namespaces` as wildcard, `"default"` an ordinary name. Take the breakage now. | G3 cannot start; G4 is blocked; `StepStats`/metrics dashboards stay silently wrong. Deferring past G5 means the ambiguity gets encoded into saved filters/URLs — a breaking change for users later instead of code now. |
| **D3** | GUI backend in-module vs. separate service | **In-module** (`cmd/awis-server`). Unblocks plugin/recall/audit surfaces immediately (they are `internal/`-typed and unreachable from outside the module); defers public-API design until a second consumer exists. | No hard blocker if deferred, but building outside the module first means a forced later migration plus promoting four row types under time pressure. |
| **D4** | Gate live intelligence, or ship declared-unverified | **Gate it** — a scheduled, secret-gated CI job outside `make verify`, asserting the response `model` field. The stale `claude-sonnet-4-5` ID (found and fixed) is proof the fake-server test suite structurally cannot catch this class. | G0 cannot close; G9 ships AI features that have never executed against the real API in any automated gate. |

**Independently re-confirmed 2026-09-01: none of D1-D4 has been decided since the corpus
was written.** All four decisions remain open and are the actual rate-limiter on every
path below — not engineering capacity.

## 2. The process decision: what is "the engine"?

Before any of D1-D4 matters, one more decision determines what all of the above is
*against*. `main` is 60 commits behind `engine-hardening` and does not contain the
30-commit defect-closure program (M15, M16, M17, and the hardening branch itself),
including data-loss-class fixes (RC-A: durable state kept in-memory only, lost on
restart). `STATE.md`, even on `main`, still shows M10-M14 as unmerged though they landed
at `f6aa755`, and shows M17 stuck in `C-VERIFY` with an unresolved row
(`GUI_MASTER_PLAN.md` §1.5). This is `REPOSITORY_TRUTH_AUDIT.md`'s N7, ranked above G0
itself in its recommended-actions table (action #0).

**Recommendation: resolve this first, as Phase A, before G0 engineering starts** — not
because it blocks any single engineering task (it doesn't; G0-G4 cards work identically
regardless), but because every path below assumes "the engine" names one commit, and
right now it doesn't. Cost: ~1 day of founder process time (reconcile `STATE.md`, execute
the merge sequence or declare `engine-hardening` the new base), zero engineering. This is
the cheapest, highest-leverage action available in the entire program.

---

## 3. Recommended path

**Phase A (process, ~1d) → G0 → G1 → G2 in series (≈5-6d) → G3 ∥ G4 (parallel) → G5.**
Then G6/G7 (editor), with G8/G9/G10 staffed in parallel once G4/G5 land, not queued
behind the editor.

```
Phase A ──▶ G0 ──▶ G1 ──▶ G2 ──┬──▶ G4 ──▶ G5 ──┬──▶ G6 ──▶ G7 ──┐
                    │          │                 │                ├──▶ (n8n-class)
                    └──▶ G3 ───┘                 └──▶ G10          │
                                                       │            │
                                          G8 (after G2) ────────────┤
                                          G9 (after G6, D4) ────────┘
```

**Why this wins.** It front-loads the two changes that are cheap now and expensive after
a GUI exists — the cursor (D1/G2) and the namespace semantics (D2/G3) — before any UI
code depends on either shape. It pins the contract (G1) before anyone builds on it. It
still puts a demonstrable, *truthful* live GUI on screen inside three to four weeks. Per
the card-level breakdown, seven of G4's thirteen cards have zero dependency on G2/G3 and
can be built the moment D3 is provisionally accepted, so the "G3 ∥ G4" step is not idle
waiting — API skeleton and read-only routes proceed while G2/G3 land.

**Cost:** ~3-4 weeks to a live GUI, ~6-7 weeks to a working visual editor (G6 alone is
10-12d, the largest single milestone in the program, because the write side has zero
existing code — `yaml.Marshal` is called nowhere in the repo today).

**Risk retired by this path:** R1 (Critical — live stream silently omits `waiting`) and
R2 (Critical — `StepStats` returns zeros, not an error, on namespace mismatch) are both
closed before any GUI code depends on the shapes they'd otherwise corrupt.

---

## 4. Alternative A — Cheapest / fastest-to-screen

**Skip G2 initially. Poll `Runtime.ListPaged` every second or two instead of building
SSE.** Ship a read-only-plus-quasi-live GUI (Phase 1, with a polling approximation of
Phase 2) in ~1-2 weeks with zero engine change beyond G0/G1. Add G2 and swap to SSE later
as a follow-up increment.

**What you get fast:** something on screen for stakeholder buy-in inside two weeks,
without waiting on D1 or any engine-internals work.

**What it costs later.** `GUI_MASTER_PLAN.md` §13.1 already priced this exactly: *"the
cheapest path is a false economy: polling is not much less work than the SSE handler,
and it must be thrown away."* Concretely: the polling frontend code (list-patching,
diffing, refresh-interval tuning) is not reusable when SSE lands — it gets deleted, not
extended. And polling on `ListPaged` alone inherits B-a: a parked instance's `waiting`
transition is still invisible in the underlying data (`ListPaged` reflects the same
projection state SSE would tail, but without G2's `state_changes` writes there is nothing
new to distinguish "still running" from "parked" between polls unless the poll happens to
land after the projection write — which it does, since `UpsertInstance` writes status
synchronously, so this path *does* correctly show `waiting` once G0/G1 land, just later
and coarser-grained than SSE). The honest framing: this path is not blocked on B-a the
way G2-first assumed, but it ships a strictly worse product (higher latency, no
reconnect/resume semantics, wasted throwaway frontend work) for a ~1-2 week head start
that then gets partially clawed back rewriting the live layer.

**When to choose this instead of the recommended path:** only if there is a hard external
deadline for *any* visible progress before D1 can be decided, and the team explicitly
accepts throwaway frontend work as the price.

---

## 5. Alternative B — Safest / decision-gated

**Get D1-D4 and Phase A all resolved, and G0-G3 fully landed and merged, before writing
any GUI/API code — not even G4's zero-dependency read-only cards run ahead of the tail of
G1/G2/G3.** This differs from the recommended path only in refusing to front-load G4's
seven zero-dependency cards in parallel with G1's tail; everything else is identical.

**What you get:** zero risk of rework from a contract shifting under in-flight API code;
the cleanest possible audit trail, which matters given the model-allocation-policy
tension `GUI_MASTER_PLAN.md` §1.4 already flagged (the hardening program that produced
this codebase ran on Opus in violation of the repository's own frozen Sonnet/Haiku-only
policy) — a founder who wants extra assurance that G2/G3's correctness-critical changes
get isolated, single-purpose review cycles gets that most cleanly if nothing else is
in flight around them.

**What it costs:** adds roughly the ~2 days of G4 work that the recommended path would
otherwise have overlapped with G1/G2/G3's tail — call it ~1 week slower to a live GUI
than the recommended path, for a program of this size not a large premium.

**When to choose this instead of the recommended path:** if the founder specifically
wants G2 and G3 reviewed and merged in isolation (given they are the correctness-critical
class B-31 just closed five instances of) before any other code changes in the same
window, or if engineering capacity is constrained enough that strict serialization is
easier to manage than deliberate parallelization.

---

## 6. A scope alternative, orthogonal to sequencing: where to stop for a beta

Independent of which sequencing path is chosen, there is a separate question of *how far*
to build before calling something a beta. `ENGINE_GUI_WORK_BREAKDOWN.md`'s GUI-track
analysis is direct on this: a defensible beta is **Phases 1-3 (through G5, i.e., full
read + live + submit/signal/cancel/register via API) — not Phase 4.** Shipping the visual
editor (G6) in a beta means shipping N4's YAML key-order round-trip risk under time
pressure, in the same milestone the corpus already calls "the largest, least-de-risked
milestone in the roadmap." Card count for a Phase-1-3 beta: roughly 25-28 cards, ~4 weeks
total (the ~3-week critical path to G5 plus ~4-5 days of Phase 3's mutation UI layered on
top). See `FINAL_VERDICT.md` for this stated as a direct answer.

---

## 7. Tradeoff summary

| | Recommended | Alt A: Cheapest | Alt B: Safest |
|---|---|---|---|
| Time to first live GUI | ~3-4 weeks | ~1-2 weeks (polling), then a rewrite | ~4-5 weeks |
| Time to *truthful* live GUI (B-a closed) | ~3-4 weeks | ~3-4 weeks (after the SSE rewrite) — no faster in the end | ~4-5 weeks |
| Throwaway work | None | Polling frontend code, discarded at SSE cutover | None |
| Contract-shift rework risk | Low (G1 pinned before G4's dependent cards) | Low | None (fully eliminated) |
| Review isolation for correctness-critical cards (G2/G3) | Partial — G4's independent cards run concurrently | Partial | Full |
| Best fit when... | Default choice; no unusual constraint | A demo deadline predates D1 being decided | Founder wants maximal review isolation, or capacity is tight |

**Recommendation stands as stated in §3.** Neither alternative is wrong, but both are
answers to a constraint (an early demo deadline, or a need for maximal isolation) rather
than a General improvement on the recommended path — choose one of them only if that
specific constraint is actually in force.

---

*Companion to `ENGINE_GUI_TRANSITION_PROGRAM.md` (milestone structure and acceptance
gates) and `ENGINE_GUI_WORK_BREAKDOWN.md` (the cards these paths are built from). Direct
answers to the founder's brief are in `FINAL_VERDICT.md`.*
