# AWIS — Final Decision Review

**Date:** 2026-08-20 · **Inputs:** Repository Truth Audit (A) · Independent Audit (B) ·
Architecture Arbitration (C) · **Tree of record:** `main` @ `98350f6`
**Standing:** final review. All three prior documents are evidence. C is not assumed correct;
three of its conclusions are amended below, one materially.

This is a decision record, not a fourth audit. Four evidence gaps that A, B and C all left open
were closed first, because each one changes a decision. Everything else is judgment on evidence
already in hand.

**Evidence hierarchy applied: EXEC > QUOTE > TRACE > INTERP.** Where a prior document asserted at
TRACE or second-hand QUOTE and the claim gates a decision, it was re-established at the highest
level actually obtainable rather than re-graded.

---

## What this review closed before deciding

| # | Gap | Result | Level |
|---|---|---|---|
| N-1 | C flagged EDR-007 as transcribed-but-unverified and said "verify before acting" | Read first-hand. §2 row is `\| StepFallbackActivated \| — \| — \| — \| -= payload.step_id \| — \|`; §3 states "No other event type mutates variables." **Confirmed exactly** | QUOTE (first-hand) |
| N-2 | Whether the hang class is confined to `fallback:` | **It is not.** An `on_error` route on a non-initial step hangs identically — and hangs on `m15` **too**. P2 is an incomplete fix | **EXEC** |
| N-3 | The M17 branch relationship, asserted by A and never re-checked by B or C | `m17-c3-wip` and `m17-full-cli-init` are **siblings**, diverging at `a29076b` (1 commit vs 2). A's "exactly one commit ahead" is refuted | **EXEC** |
| N-4 | What actually gates M17-V1, and whether Fable is authorized to run D-CLOSE | The V1 card cites neither V-COMMON nor `docs-lint`; and a standing founder directive already authorizes work "through M18" | QUOTE |

### N-2 in full, because it is the finding that matters most

EDR-007 §2 gives `StepFailed (retrying=false)` the **same** `variables: —` treatment as
`StepFallbackActivated`. If the hang is caused by the missing `variables` entry, the `on_error`
route should exhibit it too — and P2 patches only `StepFallbackActivated`.

Probe: `a → b`, `b → h (on_error)`, `b` fails terminally, `h` is final.

| Tree | Result |
|---|---|
| `main` (pre-P2) | **no terminal status in 200 ticks** |
| `m15` (post-P2) | **no terminal status in 200 ticks** |

**P2 fixes one of two paths into the same defect.** The engine has a general rule — a step removed
from `current_steps` without entering `variables` remains permanently activatable and permanently
blocks the completion check — and `StepFallbackActivated` is only the instance that OIP happened to
hit. `StepFailed(retrying=false)` + `on_error` is the other.

Current exposure is **latent, not active**: neither shipped workflow uses `on_error` (both use
`fallback:`). But `on_error` is live code — M17-C2 completed the failure-handling generalization —
so any workflow author using the documented feature hits it.

**Consequence for the founder packet:** the EDR-007 amendment must cover **both** event rows. Sending
it scoped to `StepFallbackActivated` alone guarantees a second amendment round on a frozen document.
That is now the single most preventable delay on the board.

### N-3 in full, because it removes work from the plan

```
                a29076b
                ├── be13cf9 (S2 gofmt · S3 lint · S5 oip module · S6 verify+CI)
                │   └── 49ce489  = m17-full-cli-init
                └── 153772c      = m17-c3-wip   ("does NOT build green")
```

Verified on `m17-full-cli-init`: `anthropic.go:259` is already `defer func() { _ = resp.Body.Close() }()`
(S3 applied); `apps/oip/go.mod` already carries `github.com/awis/awis v0.0.0` + `replace ⇒ ../..`
(S5 applied); the `Makefile` already has a `verify` target. None of these are on `m17-c3-wip`.

So the branch that the **immutable M17-V1 card already names** — `BRANCH: m17-full-cli-init
(clean tree)` — is also the branch that already contains three of the six fixes B enumerated. B's
fix inventory was correct *for the tree B chose*; it chose the wrong tree. Working on
`m17-full-cli-init` is simultaneously the EEOS-compliant option, the no-new-branch option, and the
least-work option. That is rare enough to be worth acting on immediately.

---

## Part 1 — Decision Matrix

| Topic | Current verdict | Evidence | Confidence | Actionable now? |
|---|---|---|---|---|
| **Fallback hang on `main`** | **PROVEN** — non-initial `fallback:` never terminates | EXEC ×2 | Certain | **Yes** |
| **Hang generalises to `on_error`** | **PROVEN** — hangs on `main` *and* `m15` | EXEC | Certain | **Yes — this is new** |
| **P2 fixes the fallback path** | **PROVEN** | EXEC | Certain | Yes |
| **P2 is a complete fix** | **REFUTED** | EXEC | Certain | **Yes — amend scope** |
| **EDR-007 contradiction** | **PROVEN** — and it spans two rows, not one | QUOTE (first-hand) | Certain | Yes, as a founder item |
| **Status semantics (`null`→`completed`)** | **HIGH CONFIDENCE** | TRACE | High | Yes — decide with EDR-007, do not probe first |
| **`WorkflowCompleted.outputs` change** | **PROVEN** | EXEC | Certain | Yes |
| **M15 verification status** | **HIGH CONFIDENCE** — tick unexecuted, substantively correct (0 lint issues at tip) | EXEC | High | Yes — annotate, do not withdraw |
| **M16 verification status** | **PROVEN FALSE** — 3 real lint issues under the ticked row | EXEC ×2 | Certain | Yes — annotate as false |
| **M17 readiness** | **HIGH CONFIDENCE** — 2 line fixes + `-update` goldens + `d5ac004`, on `m17-full-cli-init` | EXEC (on the wrong tree) | High | **Yes — but re-run on the correct branch** |
| **`docs-lint` failures** | **PROVEN**, with a proven mechanical remedy | EXEC | Certain | Yes — but see timing below |
| **`docs-lint` gates M17-V1** | **REFUTED** — the V1 card cites neither V-COMMON nor `docs-lint` | QUOTE | High | Yes — deprioritise doc work |
| **`STATE.md` contradiction** | **PROVEN**, and it is a *recurrence* | EXEC + QUOTE | Certain | Yes — but defer |
| **CI evidence gap on milestone branches** | **PROVEN**; also proven **closed** — CI fires on any branch today | EXEC | Certain | Yes |
| **M18 A-INIT timing** | **HIGH CONFIDENCE** — opens at M17 D-CLOSE, no merge, no new founder act | QUOTE ×3 | High | **Yes — this is the plan's spine** |
| **Merge ordering (stack order)** | **HIGH CONFIDENCE** for M15→M16; **not required** for M17 | EXEC + INTERP | Moderate-High | Partially — see Part 4 |
| **Intermediate merge states green** | **UNPROVEN** — never tested | — | — | **No** — must be gated when reached |
| **M16 adapter behaviour** | **UNPROVEN** — never executed by any review | — | — | **No** — real gap, defer to M16 merge prep |
| **Dead `status == 'fallback'` transitions** | **PLAUSIBLE** | TRACE | Moderate | No — cheap to settle, but changes nothing today |
| **"Namespace extraction"** | **REFUTED** — does not exist | EXEC | Certain | No — closed permanently |

Two things moved since the Arbitration: P2's completeness (**PROVEN → REFUTED**) and M17's correct
working branch. One thing I decline to raise: the status-semantics question stays at TRACE and
that is *sufficient* — probing it would not change the decision, which is the founder's either way.

---

## Part 2 — Founder Decisions: D1–D5 re-adjudicated

| | Decision | Ruling | Change from C |
|---|---|---|---|
| **D1** | Amend EDR-007 + fallback status semantics | **FOUNDER REQUIRED — scope expanded** | Must now cover `StepFailed(retrying=false)` as well |
| **D2** | QG-4 / G3 platform boundary verdict | **FOUNDER REQUIRED** | Unchanged |
| **D3** | TDS-06 Record format sign-off | **FOUNDER REQUIRED — merge into D2** | Was separate; it is one brief |
| **D4** | The three squash merges | **FOUNDER REQUIRED** | Now QUOTE-confirmed twice |
| **D5** | Authorize a standalone P2 hotfix | **REJECTED — not a live escalation** | Overturned |

**D1 — founder required, and the scope was wrong.** EDR-007 states it is superseded only by an
explicit G2 amendment; engineering amending its own constraint is the failure the frozen-spec regime
exists to prevent. What changed: N-2 proves the amendment must cover two event rows. Sending D1
scoped to one row is not a smaller decision, it is a wrong one — the founder would ratify a partial
rule and be asked again. **Do not send D1 until it names both rows.**

**D2 — founder required.** IMP: `**G3 — Platform Boundary Verdict** | M15 | "Did OIP require platform
surgery?"`. The party that wrote the platform changes cannot certify they were minimal. Non-delegable
by construction.

**D3 — founder required, but stop treating it as separate latency.** `G3_BRIEF.md` asks QG-4 and
TDS-06 in the same document with one verdict line. C listed them as two decisions with two latencies;
they are one brief, and splitting them in the packet invites two round-trips where one is needed.
**Merge D3 into D2.**

**D4 — founder required, rule-bound.** AEO §13.4 ("The human performs every merge. No agent,
including CE, merges to main") and the standing directive in `STATE.md` independently: *"milestones
stack branches, E-MERGE stays human."* Two sources, no discretion. Note the strict limit: this gates
**merges only**, and merges are not on the path to M18 A-INIT.

**D5 — rejected. I am overturning the Arbitration here, which is my own prior escalation.**
Three reasons, any one sufficient: (1) N-2 proves the fix is incomplete, so a hotfix would ship a
partial remedy and still require D1; (2) the standing founder directive already fixes the process
("milestones stack branches, E-MERGE stays human"), so asking again is asking a settled question;
(3) by the time P2 is complete enough to hotfix, D1 will have been answered, which moots it. Putting
D5 in the packet spends founder attention on a hypothetical and dilutes the three decisions that are
real. **Cut it.**

**Confirmed delegable — and one correction.** Verdict annotation (no rule governs it); the 12 contract
files; all code fixes; `BOUNDARY_EVIDENCE`/`G3_BRIEF` placement; branch mechanics. `STATE.md`
reconciliation goes to **Fable**, not the founder — and N-4 confirms Fable needs no fresh invocation:
*"founder directive: continue through M18 with Sonnet/Haiku subagents only (no Opus; Opus-designated
work is done inline by Fable, upward substitution per EEOS rule 8)"*. C inferred this; it is written
down.

---

## Part 3 — Minimum Viable Execution Plan

Constraints honoured: EEOS-compliant, **no new branches**, no rebaselining, nothing done to satisfy
an audit's preference. Work happens on `m17-full-cli-init` — the branch the immutable M17-V1 card
names, which already carries S3, S5, gofmt and the `verify` target (N-3).

| # | Step | Owner | Prerequisite | Risk | Duration |
|---|---|---|---|---|---|
| 1 | Tag `preserve/m17-c3-wip` → `153772c` | Sonnet | — | None | 5 min |
| 2 | Pin `golangci-lint@v2.12.2` in the Makefile | Sonnet | — | None | 15 min |
| 3 | **Send founder packet: D1 (both EDR-007 rows) + D2/D3 as one G3 brief** | Human | N-2 understood | Low — wrong scope costs a second round | 1 h |
| 4 | On `m17-full-cli-init`: apply C3 content from `153772c`; cherry-pick `d5ac004` | Sonnet | 1 | Low — `d5ac004` is targeted; avoids the known `ci.yml`/`Makefile` conflict a full main-merge would raise | 45 min |
| 5 | Apply `//go:embed all:scaffold` + test path depth; generate goldens via `-update` | Sonnet | 4 | Low — proven mechanical | 30 min |
| 6 | Push `m17-full-cli-init` → first real CI on an M17 branch | Verifier | 5 | Low | 15 min wall |
| 7 | Dispatch **M17-V1** per its existing card, clean tree | awis-verifier | 6 | Low — card is READY and unmodified | 2 h |
| 8 | **M17 D-CLOSE (Fable) → M18 A-INIT opens, cards READY** | Fable | 7 | Low — pre-authorized (N-4) | 2 h |
| 9 | Extend the sentinel to `StepFailed(retrying=false)`; add both probe cases as tests via `assertProjectionEquivalence` | Fable-grade | D1 answered | Medium — frozen surface; must not land before D1 | 3 h |
| 10 | Compile 12 contract files; 2 `git mv`s; M15 `DEPENDENCY_MAP.md` | Sonnet | — (parallel) | Low | 3–4 h |
| 11 | `STATE.md` reconciliation as a Fable-wake ledger/repo contradiction | Fable | — (parallel) | Medium | 1 h |
| 12 | Founder squash-merges M15 → M16 → M17 | Human | D1, D2/D3, 9, 10, 11 | Low | latency |

**Steps 1–8 are the whole path to M18 A-INIT: ~6 hours, zero founder dependency.** Step 3 runs in
parallel and starts human latency immediately. Steps 9–12 are merge preparation and belong after
A-INIT, not before it.

**Removed from C's plan and why:** the `awis-integration` branch (N-3 makes it unnecessary — the
card's own branch is better); the full-stack merge as a working tree (it was a measurement, not a
plan); and step ordering that put doc compilation ahead of verification (N-4 proves it does not gate).

---

## Part 4 — Critical Path

**1. What is the earliest point M18 A-INIT can legally open?**
At **M17's D-CLOSE** — approximately 6 hours of machine work from now, with no merge and no new
founder act. EEOS's phase machine:

> `D-CLOSE  [FABLE]   semantic review of full diff · HANDOFF actuals · PR + evidence ·`
> `                   (non-gated boundary: also run A-INIT of the next milestone, cards READY)`

**2. What exact condition unlocks it?**
M17 reaching D-CLOSE, which requires C-VERIFY to return all-✅. The gate is the M17-V1 card, verbatim:

> `DISPATCH: awis-verifier · BRANCH: m17-full-cli-init (clean tree)`
> `OBJECTIVE: Verify every VALIDATION_CHECKLIST.md row; binary evidence; read-only.`

Two things follow that C only inferred. The card **names its branch**, and cards are immutable once
READY (EEOS rule 3) — so verification belongs on `m17-full-cli-init`, not on a new branch. And the
card cites **neither V-COMMON nor `docs-lint`** — so the 12 contract files do not gate A-INIT. They
gate the merge.

The boundary is genuinely non-gated: IMP places `**G3 — Platform Boundary Verdict** | M15` and
`**G4 — Release** | M18 | "Do QG-1..5 pass…"`. G4 is M18's **exit**. Nothing gates M18's entry.

**3. Which milestone currently gates that condition?**
**M17 alone**, and only its C3 card. M15 and M16 sit at `E-MERGE (blocked on founder)` and do not
block M17's phase progression — the repository already demonstrates this, since M17 reached B-BUILD
while both predecessors were parked at E-MERGE.

**4. Is founder latency on the critical path?**
**To M18 A-INIT: no.** Not for the merges (E-MERGE is a later phase), not for Fable (pre-authorized
through M18 by standing directive), not for G3 (M15's gate, not M17's).

**To M15/M16/M17 merges: yes, and it is the only thing on it** — D1, D2/D3, D4.

**To M18 completion and G4: yes, decisively.** M18 requires "one week OIP dogfood", OIP is M15, and
M15 is founder-gated. The dogfood week cannot start until D2/D3 clear. **That — not A-INIT — is what
founder latency actually costs, and it is why step 3 runs today.**

---

## Part 5 — Kill List

**Do not do these. They were proposed, they are not justified.**

| Killed | Source | Why |
|---|---|---|
| Rebaseline M16 onto `main` | A | Refuted as an EEOS requirement; stack merge is conflict-free; extraction costs a hand-edited ledger and a fresh V2 |
| Create `m16-rb-on-main`, `m17-rb-on-main`, `m15-rb-on-main` | A | Three branches to escape a constraint that dissolves in stack order |
| Create an `awis-integration` branch | **C (mine)** | N-3: the card's own branch is better and already carries three of the fixes |
| Fold `BOUNDARY_EVIDENCE.md` into `DEPENDENCY_MAP.md` | A | Violates the IKB genre contract; two `git mv`s achieve `docs-lint: OK` |
| Weigh an EEOS §13 amendment for M15's extra files | A | Constitutional capital spent on bookkeeping with a proven mechanical fix |
| Withdraw M15-V1 | A | Refuted — M15's tip lints clean; the tick was unexecuted, not false |
| Re-verify M15/M16 on their branch tips | A | Verifies trees that will never merge as-is; the merge candidate is stronger evidence |
| Escalate D5 to the founder | **C (mine)** | Moot once N-2 lands; the standing directive already settles the process |
| Remap Fable → Opus 5 | A | `CLAUDE.md` forbids Opus; the standing directive assigns the work to Fable explicitly |
| **A fourth review of any kind** | — | Three documents have converged the factual surface. The remaining unknowns are two probes and one founder brief, not more analysis |

**Safely deferred until after M18 A-INIT:**

- All 12 contract files and the two `git mv`s — gate the merge, not A-INIT (QUOTE-proven).
- `STATE.md` reconciliation — real, recurring, and blocking nothing.
- The `status == 'fallback'` dead-transition probe — changes no decision now.
- M16 adapter behavioural testing — a genuine evidence gap, but it belongs in M16 merge prep.
- Verdict annotations for M15-V1 / M16-V1 — 30 minutes, do it when touching those modules.
- Gating the intermediate merge states — cannot be done until those merges are actually attempted.

---

## Part 6 — Final Recommendation

**1. If I owned AWIS, what would I do tomorrow morning?**

Two things before anything else, in this order. Write the founder brief — **scoped to both EDR-007
rows**, with QG-4 and TDS-06 as one G3 verdict, and D5 removed. Then put an engineer on
`m17-full-cli-init` with the C3 content and `d5ac004`, and take M17 to D-CLOSE by end of day. That
opens M18 without asking anyone's permission. I would not touch a contract file, a ledger, or a merge
until both are done.

**2. Single action with the largest risk reduction?**

**Extending the sentinel to `StepFailed(retrying=false)` and landing both probes as regression tests.**
Not P2 itself — P2 is half a fix, and shipping half a fix on a frozen surface is worse than shipping
none, because it makes the defect look closed. The tests matter as much as the code: a hang class
survived twelve milestones and three reviews because no test drove a failure through a non-initial
step. Two test cases in a harness that already exists close that permanently.

**3. Single action with the largest schedule acceleration?**

**Sending the G3 brief today.** Everything else on the board is hours of machine work that I control.
D2/D3 is the only item with unbounded latency, and it gates the OIP dogfood week — the longest single
activity in M18. Every day that brief sits unwritten is a day added to the release, and no amount of
engineering throughput recovers it.

**4. Most likely way this project gets delayed from here?**

**A second EDR-007 amendment round.** The packet goes out scoped to `StepFallbackActivated`, the
founder ratifies it, engineering then discovers `on_error` has the same defect, and a frozen document
must be reopened — after a G2 amendment has already been spent on it, and possibly after `v1.0.0`
makes the change far more expensive. It is entirely preventable today and it is invisible unless
someone reads the second row of the EDR-007 table.

The runner-up is quieter and more likely to actually happen: **mistaking merge preparation for
progress.** Twelve contract files, a ledger reconciliation and three squash merges are visible,
tractable, satisfying work — and none of it opens M18 or fixes the engine. The pattern is already
established. Two of these three reviews produced double-digit-step programmes for what measures out
to roughly twenty lines of code, one test file, and three decisions.

---

*Final review. Four evidence gaps closed by execution and first-hand reading before any decision was
taken. Three Arbitration conclusions amended, two of them mine. No further review is recommended.*
